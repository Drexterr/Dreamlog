package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake plan manager ─────────────────────────────────────────────────────────

type fakePlanManager struct {
	upgradedPlan   models.Plan
	upgradedExpiry *time.Time
	returnUser     *models.User
}

func (f *fakePlanManager) GetPlanDetails(plan models.Plan) *models.PlanLimits {
	return models.GetPlanLimits(plan)
}

func (f *fakePlanManager) UpgradePlan(_ context.Context, _ uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error) {
	f.upgradedPlan = plan
	f.upgradedExpiry = expiresAt
	u := f.returnUser
	if u == nil {
		u = &models.User{ID: uuid.New(), Plan: plan, PlanExpiresAt: expiresAt}
	} else {
		u.Plan = plan
		u.PlanExpiresAt = expiresAt
	}
	return u, nil
}

// ── test router ───────────────────────────────────────────────────────────────

const billingTestSecret = "billing-test-jwt-secret-32-bytes"

func newBillingTestRouter(t *testing.T, svc planManager, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(billingTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := &BillingHandler{svc: svc}
	r.GET("/billing/plan", h.GetPlan)
	r.POST("/billing/upgrade", h.Upgrade)
	return r
}

func billingTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-billing-001",
		"email": "billing@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(billingTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func billingTestUser(plan models.Plan) *models.User {
	return &models.User{ID: uuid.New(), Email: "billing@test.com", Name: "Billing User", Plan: plan}
}

// ── GET /billing/plan ─────────────────────────────────────────────────────────

func TestBillingHandler_GetPlan_FreeUser_Returns200WithLimits(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["plan"] != "free" {
		t.Fatalf("expected plan=free, got %v", resp["plan"])
	}
	if resp["limits"] == nil {
		t.Fatal("expected limits in response")
	}
	if resp["all_plans"] == nil {
		t.Fatal("expected all_plans in response")
	}
}

func TestBillingHandler_GetPlan_PlusUser_Returns200WithPlusPlan(t *testing.T) {
	user := billingTestUser(models.PlanPlus)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["plan"] != "plus" {
		t.Fatalf("expected plan=plus, got %v", resp["plan"])
	}
}

func TestBillingHandler_GetPlan_MissingAuth_Returns401(t *testing.T) {
	r := newBillingTestRouter(t, &fakePlanManager{}, billingTestUser(models.PlanFree))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── POST /billing/upgrade ─────────────────────────────────────────────────────

func TestBillingHandler_Upgrade_ToPlus_Returns200(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	svc := &fakePlanManager{}
	r := newBillingTestRouter(t, svc, user)

	body, _ := json.Marshal(map[string]string{"plan": "plus"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanPlus {
		t.Fatalf("expected plan upgraded to plus, got %s", svc.upgradedPlan)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["plan"] != "plus" {
		t.Fatalf("expected plan=plus in response, got %v", resp["plan"])
	}
}

func TestBillingHandler_Upgrade_ToPro_Returns200(t *testing.T) {
	user := billingTestUser(models.PlanPlus)
	svc := &fakePlanManager{}
	r := newBillingTestRouter(t, svc, user)

	body, _ := json.Marshal(map[string]string{"plan": "pro"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanPro {
		t.Fatalf("expected plan upgraded to pro, got %s", svc.upgradedPlan)
	}
}

func TestBillingHandler_Upgrade_DowngradeToFree_Returns200(t *testing.T) {
	user := billingTestUser(models.PlanPro)
	svc := &fakePlanManager{}
	r := newBillingTestRouter(t, svc, user)

	body, _ := json.Marshal(map[string]string{"plan": "free"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for downgrade, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanFree {
		t.Fatalf("expected plan=free, got %s", svc.upgradedPlan)
	}
}

func TestBillingHandler_Upgrade_InvalidPlan_Returns400(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	body, _ := json.Marshal(map[string]string{"plan": "ultra_mega_premium"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid plan, got %d", w.Code)
	}
}

func TestBillingHandler_Upgrade_MissingPlan_Returns400(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing plan, got %d", w.Code)
	}
}

func TestBillingHandler_Upgrade_ClientExpiryIgnored_ServerSets30Days(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	svc := &fakePlanManager{}
	r := newBillingTestRouter(t, svc, user)

	// Client attempts to grant itself a plan until 2099 - must be ignored.
	body, _ := json.Marshal(map[string]string{"plan": "pro", "expires_at": "2099-01-01T00:00:00Z"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedExpiry == nil {
		t.Fatal("expected server-set expiry")
	}
	maxExpiry := time.Now().Add(31 * 24 * time.Hour)
	if svc.upgradedExpiry.After(maxExpiry) {
		t.Errorf("client-supplied expiry must be ignored; got %v (more than 31 days out)", svc.upgradedExpiry)
	}
}

func TestBillingHandler_Upgrade_MissingAuth_Returns401(t *testing.T) {
	r := newBillingTestRouter(t, &fakePlanManager{}, billingTestUser(models.PlanFree))

	body, _ := json.Marshal(map[string]string{"plan": "plus"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBillingHandler_GetPlan_ProUser_Returns200WithProPlan(t *testing.T) {
	user := billingTestUser(models.PlanPro)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["plan"] != "pro" {
		t.Fatalf("expected plan=pro, got %v", resp["plan"])
	}
}

func TestBillingHandler_Upgrade_ToB2B_Returns200(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	svc := &fakePlanManager{}
	r := newBillingTestRouter(t, svc, user)

	body, _ := json.Marshal(map[string]string{"plan": "b2b"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanB2B {
		t.Fatalf("expected plan upgraded to b2b, got %s", svc.upgradedPlan)
	}
}

func TestBillingHandler_GetPlan_ProLimitsHavePDFExport(t *testing.T) {
	user := billingTestUser(models.PlanPro)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Extract limits.has_pdf_export
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	limits, ok := resp["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("expected limits object")
	}
	if limits["has_pdf_export"] != true {
		t.Errorf("pro plan must have has_pdf_export=true, got %v", limits["has_pdf_export"])
	}
	if limits["has_weekly_review"] != true {
		t.Errorf("pro plan must have has_weekly_review=true, got %v", limits["has_weekly_review"])
	}
}

func TestBillingHandler_GetPlan_FreeLimitsHaveMonthlyEntries(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	limits, ok := resp["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("expected limits object")
	}
	if limits["has_pdf_export"] != false {
		t.Errorf("free plan must have has_pdf_export=false, got %v", limits["has_pdf_export"])
	}
	if limits["has_weekly_review"] != false {
		t.Errorf("free plan must have has_weekly_review=false, got %v", limits["has_weekly_review"])
	}
}

// ── Verified (production) mode: payment must be proven server-side ───────────

type fakePaymentRecorder struct {
	seen map[string]bool
}

func (f *fakePaymentRecorder) Record(_ context.Context, _ uuid.UUID, intentID string, _ models.Plan, _ int64, _ string) (bool, error) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	if f.seen[intentID] {
		return false, nil
	}
	f.seen[intentID] = true
	return true, nil
}

// stripeStub serves GET /v1/payment_intents/:id with a canned intent.
func stripeStub(t *testing.T, intent map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(intent)
	}))
}

func newVerifiedBillingRouter(t *testing.T, svc planManager, payments paymentRecorder, stripeURL string, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(billingTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := &BillingHandler{svc: svc, payments: payments, stripeSecretKey: "sk_test_123", stripeBaseURL: stripeURL}
	r.GET("/billing/plan", h.GetPlan)
	r.POST("/billing/upgrade", h.Upgrade)
	return r
}

func postUpgrade(t *testing.T, r *gin.Engine, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestBillingHandler_UpgradeVerified_MissingPaymentIntent_Returns400(t *testing.T) {
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, "http://localhost:0", billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "plus"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without payment_intent_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_PaymentNotSucceeded_Returns402(t *testing.T) {
	stripe := stripeStub(t, map[string]interface{}{
		"id": "pi_1", "status": "requires_payment_method", "amount": 19900, "currency": "inr",
		"metadata": map[string]string{"plan": "plus"},
	})
	defer stripe.Close()

	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, stripe.URL, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "plus", "payment_intent_id": "pi_1"})
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 for unpaid intent, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_WrongPlanMetadata_Returns400(t *testing.T) {
	stripe := stripeStub(t, map[string]interface{}{
		"id": "pi_2", "status": "succeeded", "amount": 19900, "currency": "inr",
		"metadata": map[string]string{"plan": "plus"},
	})
	defer stripe.Close()

	// Paid for plus, asking for pro.
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, stripe.URL, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "pro", "payment_intent_id": "pi_2"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for plan mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_AmountTooLow_Returns400(t *testing.T) {
	stripe := stripeStub(t, map[string]interface{}{
		"id": "pi_3", "status": "succeeded", "amount": 100, "currency": "inr",
		"metadata": map[string]string{"plan": "pro"},
	})
	defer stripe.Close()

	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, stripe.URL, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "pro", "payment_intent_id": "pi_3"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for amount mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_Success_GrantsServerExpiry(t *testing.T) {
	stripe := stripeStub(t, map[string]interface{}{
		"id": "pi_4", "status": "succeeded", "amount": 49900, "currency": "inr",
		"metadata": map[string]string{"plan": "pro"},
	})
	defer stripe.Close()

	svc := &fakePlanManager{}
	r := newVerifiedBillingRouter(t, svc, &fakePaymentRecorder{}, stripe.URL, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "pro", "payment_intent_id": "pi_4"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanPro {
		t.Fatalf("expected pro granted, got %s", svc.upgradedPlan)
	}
	if svc.upgradedExpiry == nil {
		t.Fatal("expected server-set expiry on verified upgrade")
	}
}

func TestBillingHandler_UpgradeVerified_ReplayedIntent_Returns409(t *testing.T) {
	stripe := stripeStub(t, map[string]interface{}{
		"id": "pi_5", "status": "succeeded", "amount": 19900, "currency": "inr",
		"metadata": map[string]string{"plan": "plus"},
	})
	defer stripe.Close()

	payments := &fakePaymentRecorder{}
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, payments, stripe.URL, billingTestUser(models.PlanFree))

	if w := postUpgrade(t, r, map[string]string{"plan": "plus", "payment_intent_id": "pi_5"}); w.Code != http.StatusOK {
		t.Fatalf("first use must succeed, got %d: %s", w.Code, w.Body.String())
	}
	if w := postUpgrade(t, r, map[string]string{"plan": "plus", "payment_intent_id": "pi_5"}); w.Code != http.StatusConflict {
		t.Fatalf("replayed intent must return 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_B2BSelfServe_Returns400(t *testing.T) {
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, "http://localhost:0", billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "b2b"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("b2b must not be self-serve in production, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_DowngradeToFree_NoPaymentNeeded(t *testing.T) {
	svc := &fakePlanManager{}
	r := newVerifiedBillingRouter(t, svc, &fakePaymentRecorder{}, "http://localhost:0", billingTestUser(models.PlanPro))
	w := postUpgrade(t, r, map[string]string{"plan": "free"})
	if w.Code != http.StatusOK {
		t.Fatalf("downgrade to free must not require payment, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanFree {
		t.Fatalf("expected free, got %s", svc.upgradedPlan)
	}
	if svc.upgradedExpiry != nil {
		t.Error("free plan must have nil expiry")
	}
}

// ── POST /billing/create-payment-intent (dev stub mode) ─────────────────────

func newPaymentIntentRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(billingTestSecret, "", &fakeProvisioner{user: billingTestUser(models.PlanFree)}, log))
	h := &BillingHandler{svc: &fakePlanManager{}, stripePublishableKey: "pk_test_x"}
	r.POST("/billing/create-payment-intent", h.CreatePaymentIntent)
	return r
}

func postPaymentIntent(t *testing.T, r *gin.Engine, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/create-payment-intent", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestCreatePaymentIntent_DevStub_ReturnsStubSecret(t *testing.T) {
	r := newPaymentIntentRouter(t)
	w := postPaymentIntent(t, r, map[string]string{"plan": "plus", "currency": "inr"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["client_secret"] == "" || resp["client_secret"] == nil {
		t.Error("expected stub client_secret in dev mode")
	}
	if resp["amount"] != float64(19900) {
		t.Errorf("plus INR amount: want 19900, got %v", resp["amount"])
	}
}

func TestCreatePaymentIntent_InvalidCurrency_Returns400(t *testing.T) {
	r := newPaymentIntentRouter(t)
	if w := postPaymentIntent(t, r, map[string]string{"plan": "plus", "currency": "gbp"}); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported currency, got %d", w.Code)
	}
}

func TestCreatePaymentIntent_FreePlan_Returns400(t *testing.T) {
	r := newPaymentIntentRouter(t)
	if w := postPaymentIntent(t, r, map[string]string{"plan": "free", "currency": "usd"}); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for free plan, got %d", w.Code)
	}
}

func TestPlanAmount_AllCurrencies(t *testing.T) {
	cases := []struct {
		plan     models.Plan
		currency string
		want     int64
	}{
		{models.PlanPlus, "inr", 19900},
		{models.PlanPro, "inr", 49900},
		{models.PlanPlus, "usd", 799},
		{models.PlanPro, "usd", 1499},
		{models.PlanPlus, "eur", 699},
		{models.PlanPro, "eur", 1299},
		{models.PlanFree, "inr", 0},
	}
	for _, tc := range cases {
		if got := planAmount(tc.plan, tc.currency); got != tc.want {
			t.Errorf("planAmount(%s, %s): want %d, got %d", tc.plan, tc.currency, tc.want, got)
		}
	}
}

// ── plan_expires_at enforcement ──────────────────────────────────────────────

func TestBillingHandler_GetPlan_ExpiredPlus_ReportsFree(t *testing.T) {
	expired := time.Now().Add(-24 * time.Hour)
	user := billingTestUser(models.PlanPlus)
	user.PlanExpiresAt = &expired

	r := newBillingTestRouter(t, &fakePlanManager{}, user)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["plan"] != "free" {
		t.Fatalf("expired plus plan must report as free, got %v", resp["plan"])
	}
}

func TestUser_EffectivePlan(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	cases := []struct {
		name   string
		plan   models.Plan
		expiry *time.Time
		want   models.Plan
	}{
		{"free never expires", models.PlanFree, nil, models.PlanFree},
		{"plus no expiry", models.PlanPlus, nil, models.PlanPlus},
		{"plus future expiry", models.PlanPlus, &future, models.PlanPlus},
		{"plus expired", models.PlanPlus, &past, models.PlanFree},
		{"pro expired", models.PlanPro, &past, models.PlanFree},
		{"b2b expired", models.PlanB2B, &past, models.PlanFree},
	}
	for _, tc := range cases {
		u := &models.User{Plan: tc.plan, PlanExpiresAt: tc.expiry}
		if got := u.EffectivePlan(); got != tc.want {
			t.Errorf("%s: want %s, got %s", tc.name, tc.want, got)
		}
	}
}

func TestBillingHandler_GetPlan_AllPlansPresent(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	r := newBillingTestRouter(t, &fakePlanManager{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	allPlans, ok := resp["all_plans"].(map[string]interface{})
	if !ok {
		t.Fatal("expected all_plans object")
	}
	for _, p := range []string{"free", "plus", "pro", "b2b"} {
		if allPlans[p] == nil {
			t.Errorf("all_plans must contain %q plan", p)
		}
	}
}
