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

// paymentRecorder is what BillingHandler needs from PaymentRepository.
type paymentRecorder interface {
	Record(ctx context.Context, userID uuid.UUID, paymentIntentID string, plan models.Plan, amount int64, currency string) (bool, error)
}

// paidPlanDuration is how long one verified payment grants a paid plan.
// Payments are one-time PaymentIntents (a 30-day pass), not recurring
// subscriptions - the mobile copy must describe them as such.
const paidPlanDuration = 30 * 24 * time.Hour

type BillingHandler struct {
	svc                  planManager
	payments             paymentRecorder
	analytics            analyticsTracker
	stripeSecretKey      string
	stripePublishableKey string
	stripeBaseURL        string // overridable in tests; defaults to the real Stripe API
}

// analyticsTracker is the subset of AnalyticsService used by BillingHandler.
type analyticsTracker interface {
	TrackUser(ctx context.Context, userID uuid.UUID, event string, props map[string]any)
}

func NewBillingHandler(svc planManager, payments paymentRecorder, analytics analyticsTracker, stripeSecretKey, stripePublishableKey string) *BillingHandler {
	return &BillingHandler{
		svc:                  svc,
		payments:             payments,
		analytics:            analytics,
		stripeSecretKey:      stripeSecretKey,
		stripePublishableKey: stripePublishableKey,
		stripeBaseURL:        stripeAPIBaseURL,
	}
}

const stripeAPIBaseURL = "https://api.stripe.com"

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

// POST /billing/upgrade - sets the user's plan after server-side payment verification.
//
// Security model:
//   - free: always allowed (self-downgrade), clears expiry.
//   - plus/pro with Stripe configured: payment_intent_id is REQUIRED. The
//     intent is fetched from Stripe and must be succeeded, for the requested
//     plan, and for the exact amount. Each intent grants a plan exactly once
//     (payments table, unique on payment_intent_id). Expiry is set
//     server-side to now + 30 days - never taken from the client.
//   - b2b with Stripe configured: rejected; b2b is provisioned out-of-band.
//   - Dev (no STRIPE_SECRET_KEY): grants without verification so the local
//     stack needs no external APIs.
func (h *BillingHandler) Upgrade(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var req struct {
		Plan            models.Plan `json:"plan" binding:"required"`
		PaymentIntentID string      `json:"payment_intent_id"`
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

	var expiresAt *time.Time

	switch {
	case req.Plan == models.PlanFree:
		// Self-downgrade: no payment, no expiry.

	case h.stripeSecretKey == "":
		// Dev stub: grant paid plans with a server-set 30-day expiry.
		if req.Plan != models.PlanB2B {
			t := time.Now().Add(paidPlanDuration)
			expiresAt = &t
		}

	case req.Plan == models.PlanB2B:
		_ = c.Error(apierr.BadRequest("b2b plans are provisioned by sales, not self-serve"))
		return

	default:
		// Production: verify the payment with Stripe before granting anything.
		if req.PaymentIntentID == "" {
			_ = c.Error(apierr.BadRequest("payment_intent_id is required"))
			return
		}
		intent, err := retrieveStripePaymentIntent(c.Request.Context(), h.stripeBaseURL, h.stripeSecretKey, req.PaymentIntentID)
		if err != nil {
			_ = c.Error(apierr.Internal("payment verification unavailable"))
			return
		}
		if intent.Status != "succeeded" {
			_ = c.Error(apierr.New(http.StatusPaymentRequired, "payment has not succeeded"))
			return
		}
		if intent.Metadata.Plan != string(req.Plan) {
			_ = c.Error(apierr.BadRequest("payment was made for a different plan"))
			return
		}
		if intent.Amount < planAmount(req.Plan, intent.Currency) {
			_ = c.Error(apierr.BadRequest("payment amount does not match plan price"))
			return
		}

		inserted, err := h.payments.Record(c.Request.Context(), userID, intent.ID, req.Plan, intent.Amount, intent.Currency)
		if err != nil {
			_ = c.Error(apierr.Internal("failed to record payment"))
			return
		}
		if !inserted {
			_ = c.Error(apierr.Conflict("this payment has already been used"))
			return
		}

		t := time.Now().Add(paidPlanDuration)
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

// POST /billing/create-payment-intent - creates a Stripe PaymentIntent for a plan upgrade.
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

	// Dev mode: Stripe keys not configured - return a test stub.
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
			return 24900 // ₹249
		case models.PlanPro:
			return 49900 // ₹499
		}
	case "eur":
		switch plan {
		case models.PlanPlus:
			return 599 // €5.99
		case models.PlanPro:
			return 999 // €9.99
		}
	default: // usd
		switch plan {
		case models.PlanPlus:
			return 599 // $5.99
		case models.PlanPro:
			return 999 // $9.99
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

// stripePaymentIntent is the subset of the Stripe PaymentIntent object we verify.
type stripePaymentIntent struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Metadata struct {
		Plan string `json:"plan"`
	} `json:"metadata"`
}

// retrieveStripePaymentIntent fetches a PaymentIntent from Stripe for server-side
// verification. Never trust payment state reported by the client.
func retrieveStripePaymentIntent(ctx context.Context, baseURL, secretKey, intentID string) (*stripePaymentIntent, error) {
	if baseURL == "" {
		baseURL = stripeAPIBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		baseURL+"/v1/payment_intents/"+url.PathEscape(intentID), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		stripePaymentIntent
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("stripe: %s", result.Error.Message)
	}
	return &result.stripePaymentIntent, nil
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
