package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// planManager is what BillingHandler needs from SubscriptionService.
type planManager interface {
	GetPlanDetails(plan models.Plan) *models.PlanLimits
	UpgradePlan(ctx context.Context, userID uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error)
}

// paymentRecorder is what BillingHandler needs from PaymentRepository.
type paymentRecorder interface {
	Record(ctx context.Context, userID uuid.UUID, transactionID string, plan models.Plan, store, productID string) (bool, error)
}

// purchaseVerifier is what BillingHandler needs from IAPService: server-side
// verification of App Store / Play Store purchases.
type purchaseVerifier interface {
	Configured() bool
	VerifyPurchase(ctx context.Context, platform, productID, purchaseToken string) (string, error)
}

// Purchases are one-time store products (a 30-day or 365-day pass), not
// auto-renewing subscriptions - the mobile copy must describe them as such.
const (
	periodMonthly = "monthly"
	periodAnnual  = "annual"

	monthlyPlanDuration = 30 * 24 * time.Hour
	annualPlanDuration  = 365 * 24 * time.Hour
)

// normalizePeriod validates the billing period, defaulting to monthly for
// requests that don't carry one.
func normalizePeriod(p string) (string, bool) {
	switch p {
	case "", periodMonthly:
		return periodMonthly, true
	case periodAnnual:
		return periodAnnual, true
	default:
		return "", false
	}
}

func planDuration(period string) time.Duration {
	if period == periodAnnual {
		return annualPlanDuration
	}
	return monthlyPlanDuration
}

type BillingHandler struct {
	svc       planManager
	payments  paymentRecorder
	analytics analyticsTracker
	iap       purchaseVerifier
}

// analyticsTracker is the subset of AnalyticsService used by BillingHandler.
type analyticsTracker interface {
	TrackUser(ctx context.Context, userID uuid.UUID, event string, props map[string]any)
}

func NewBillingHandler(svc planManager, payments paymentRecorder, analytics analyticsTracker, iap purchaseVerifier) *BillingHandler {
	return &BillingHandler{
		svc:       svc,
		payments:  payments,
		analytics: analytics,
		iap:       iap,
	}
}

// GET /billing/plan - returns the authenticated user's current plan and its limits.
func (h *BillingHandler) GetPlan(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		_ = c.Error(apierr.Unauthorized("user not found"))
		return
	}
	// Report the effective plan: an expired paid plan is shown (and gated) as free.
	plan := user.EffectivePlan()
	limits := h.svc.GetPlanDetails(plan)
	c.JSON(http.StatusOK, gin.H{
		"plan":            plan,
		"plan_expires_at": user.PlanExpiresAt,
		"limits":          limits,
		"all_plans":       allPlanDetails(),
	})
}

// POST /billing/upgrade - sets the user's plan after server-side IAP verification.
//
// Security model:
//   - free: always allowed (self-downgrade), clears expiry.
//   - plus/pro with IAP configured: platform, product_id, and purchase_token
//     are REQUIRED. The purchase is verified with the store (Apple
//     verifyReceipt / Play purchases.products.get); the product must map to
//     the requested plan+period; each store transaction grants a plan exactly
//     once (payments table, unique on transaction_id). Expiry is set
//     server-side - never taken from the client. Pricing is store-managed, so
//     there is no amount check: a verified purchase of the product is proof
//     the store collected its price.
//   - b2b: rejected in production; b2b is provisioned out-of-band.
//   - Dev (no store credentials): grants without verification so the local
//     stack needs no external APIs.
func (h *BillingHandler) Upgrade(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var req struct {
		Plan          models.Plan `json:"plan" binding:"required"`
		Period        string      `json:"period"`
		Platform      string      `json:"platform"`       // "ios" | "android"
		ProductID     string      `json:"product_id"`     // store SKU, e.g. com.ode.app.plus.monthly
		PurchaseToken string      `json:"purchase_token"` // iOS: base64 receipt; Android: Play purchase token
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	switch req.Plan {
	case models.PlanFree, models.PlanPlus, models.PlanPro, models.PlanB2B:
		// valid
	default:
		_ = c.Error(apierr.BadRequest("plan must be one of: free, plus, pro, b2b"))
		return
	}

	period, ok := normalizePeriod(req.Period)
	if !ok {
		_ = c.Error(apierr.BadRequest("period must be monthly or annual"))
		return
	}

	var expiresAt *time.Time

	switch {
	case req.Plan == models.PlanFree:
		// Self-downgrade: no purchase, no expiry.

	case h.iap == nil || !h.iap.Configured():
		// Dev stub: grant paid plans with a server-set expiry for the period.
		if req.Plan != models.PlanB2B {
			t := time.Now().Add(planDuration(period))
			expiresAt = &t
		}

	case req.Plan == models.PlanB2B:
		_ = c.Error(apierr.BadRequest("b2b plans are provisioned by sales, not self-serve"))
		return

	default:
		// Production: verify the purchase with the store before granting anything.
		if req.Platform != "ios" && req.Platform != "android" {
			_ = c.Error(apierr.BadRequest("platform must be ios or android"))
			return
		}
		if req.ProductID == "" || req.PurchaseToken == "" {
			_ = c.Error(apierr.BadRequest("product_id and purchase_token are required"))
			return
		}
		product, known := services.IAPCatalog[req.ProductID]
		if !known {
			_ = c.Error(apierr.BadRequest("unknown product_id"))
			return
		}
		if product.Plan != req.Plan || product.Period != period {
			_ = c.Error(apierr.BadRequest("purchase was made for a different plan or billing period"))
			return
		}

		transactionID, err := h.iap.VerifyPurchase(c.Request.Context(), req.Platform, req.ProductID, req.PurchaseToken)
		if err != nil {
			if errors.Is(err, services.ErrPurchaseInvalid) {
				_ = c.Error(apierr.New(http.StatusPaymentRequired, "purchase could not be verified"))
				return
			}
			_ = c.Error(apierr.Internal("purchase verification unavailable"))
			return
		}

		store := "apple"
		if req.Platform == "android" {
			store = "google"
		}
		inserted, err := h.payments.Record(c.Request.Context(), userID, transactionID, req.Plan, store, req.ProductID)
		if err != nil {
			_ = c.Error(apierr.Internal("failed to record purchase"))
			return
		}
		if !inserted {
			_ = c.Error(apierr.Conflict("this purchase has already been used"))
			return
		}

		t := time.Now().Add(planDuration(period))
		expiresAt = &t
	}

	user, err := h.svc.UpgradePlan(c.Request.Context(), userID, req.Plan, expiresAt)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to update plan"))
		return
	}
	if user == nil {
		_ = c.Error(apierr.NotFound("user"))
		return
	}

	if h.analytics != nil {
		h.analytics.TrackUser(c.Request.Context(), userID, "plan_changed", map[string]any{
			"plan": string(req.Plan),
		})
	}

	limits := h.svc.GetPlanDetails(user.Plan)
	c.JSON(http.StatusOK, gin.H{
		"plan":            user.Plan,
		"plan_expires_at": user.PlanExpiresAt,
		"limits":          limits,
	})
}

// allPlanDetails returns the limits for every plan - used on pricing pages.
func allPlanDetails() map[models.Plan]*models.PlanLimits {
	plans := []models.Plan{models.PlanFree, models.PlanPlus, models.PlanPro, models.PlanB2B}
	out := make(map[models.Plan]*models.PlanLimits, len(plans))
	for _, p := range plans {
		out[p] = models.GetPlanLimits(p)
	}
	return out
}
