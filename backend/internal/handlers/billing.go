package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// planManager is what BillingHandler needs from SubscriptionService.
type planManager interface {
	GetPlanDetails(plan models.Plan) *models.PlanLimits
	UpgradePlan(ctx context.Context, userID uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error)
}

type BillingHandler struct {
	svc                  planManager
	stripeSecretKey      string
	stripePublishableKey string
}

func NewBillingHandler(svc planManager, stripeSecretKey, stripePublishableKey string) *BillingHandler {
	return &BillingHandler{
		svc:                  svc,
		stripeSecretKey:      stripeSecretKey,
		stripePublishableKey: stripePublishableKey,
	}
}

// GET /billing/plan — returns the authenticated user's current plan and its limits.
func (h *BillingHandler) GetPlan(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		_ = c.Error(apierr.Unauthorized("user not found"))
		return
	}
	limits := h.svc.GetPlanDetails(user.Plan)
	c.JSON(http.StatusOK, gin.H{
		"plan":            user.Plan,
		"plan_expires_at": user.PlanExpiresAt,
		"limits":          limits,
		"all_plans":       allPlanDetails(),
	})
}

// POST /billing/upgrade — stub upgrade endpoint (no payment in dev; sets plan directly).
// In production this is called after Stripe confirms payment via POST /billing/create-payment-intent.
func (h *BillingHandler) Upgrade(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var req struct {
		Plan      models.Plan `json:"plan" binding:"required"`
		ExpiresAt *time.Time  `json:"expires_at"`
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

	user, err := h.svc.UpgradePlan(c.Request.Context(), userID, req.Plan, req.ExpiresAt)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to update plan"))
		return
	}
	if user == nil {
		_ = c.Error(apierr.NotFound("user"))
		return
	}

	limits := h.svc.GetPlanDetails(user.Plan)
	c.JSON(http.StatusOK, gin.H{
		"plan":            user.Plan,
		"plan_expires_at": user.PlanExpiresAt,
		"limits":          limits,
	})
}

// POST /billing/create-payment-intent — creates a Stripe PaymentIntent for a plan upgrade.
// Returns client_secret for the mobile Stripe SDK to present the payment sheet.
// When STRIPE_SECRET_KEY is not set (dev), returns a stub client_secret.
func (h *BillingHandler) CreatePaymentIntent(c *gin.Context) {
	var req struct {
		Plan     models.Plan `json:"plan"     binding:"required"`
		Currency string      `json:"currency" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	if req.Currency != "inr" && req.Currency != "usd" && req.Currency != "eur" {
		_ = c.Error(apierr.BadRequest("currency must be inr, eur, or usd"))
		return
	}

	switch req.Plan {
	case models.PlanPlus, models.PlanPro:
		// only paid plans can be purchased
	default:
		_ = c.Error(apierr.BadRequest("plan must be plus or pro"))
		return
	}

	amount := planAmount(req.Plan, req.Currency)

	// Dev mode: Stripe keys not configured — return a test stub.
	if h.stripeSecretKey == "" {
		c.JSON(http.StatusOK, gin.H{
			"client_secret":   "pi_stub_dev_only_secret",
			"amount":          amount,
			"currency":        req.Currency,
			"publishable_key": h.stripePublishableKey,
		})
		return
	}

	clientSecret, err := createStripePaymentIntent(
		c.Request.Context(),
		h.stripeSecretKey,
		amount,
		req.Currency,
		string(req.Plan),
	)
	if err != nil {
		_ = c.Error(apierr.Internal("payment service unavailable"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"client_secret":   clientSecret,
		"amount":          amount,
		"currency":        req.Currency,
		"publishable_key": h.stripePublishableKey,
	})
}

// planAmount returns the payment amount in the smallest currency unit (paise / euro cents / cents).
func planAmount(plan models.Plan, currency string) int64 {
	switch currency {
	case "inr":
		switch plan {
		case models.PlanPlus:
			return 19900 // ₹199
		case models.PlanPro:
			return 49900 // ₹499
		}
	case "eur":
		switch plan {
		case models.PlanPlus:
			return 699 // €6.99
		case models.PlanPro:
			return 1299 // €12.99
		}
	default: // usd
		switch plan {
		case models.PlanPlus:
			return 799 // $7.99
		case models.PlanPro:
			return 1499 // $14.99
		}
	}
	return 0
}

// createStripePaymentIntent calls the Stripe API directly using net/http (no SDK dependency).
func createStripePaymentIntent(ctx context.Context, secretKey string, amount int64, currency, planMeta string) (string, error) {
	body := url.Values{}
	body.Set("amount", strconv.FormatInt(amount, 10))
	body.Set("currency", currency)
	body.Set("automatic_payment_methods[enabled]", "true")
	body.Set("metadata[plan]", planMeta)
	body.Set("metadata[product]", "dreamlog_subscription")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.stripe.com/v1/payment_intents",
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		ClientSecret string `json:"client_secret"`
		Error        *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("stripe: %s", result.Error.Message)
	}
	return result.ClientSecret, nil
}

// allPlanDetails returns the limits for every plan — used on pricing pages.
func allPlanDetails() map[models.Plan]*models.PlanLimits {
	plans := []models.Plan{models.PlanFree, models.PlanPlus, models.PlanPro, models.PlanB2B}
	out := make(map[models.Plan]*models.PlanLimits, len(plans))
	for _, p := range plans {
		out[p] = models.GetPlanLimits(p)
	}
	return out
}
