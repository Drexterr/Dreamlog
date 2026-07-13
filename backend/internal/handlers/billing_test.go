package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
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

// ── Verified (production) mode: purchase must be proven server-side ──────────

type fakePaymentRecorder struct {
	seen        map[string]bool
	lastStore   string
	lastProduct string
}

func (f *fakePaymentRecorder) Record(_ context.Context, _ uuid.UUID, transactionID string, _ models.Plan, store, productID string) (bool, error) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	f.lastStore = store
	f.lastProduct = productID
	if f.seen[transactionID] {
		return false, nil
	}
	f.seen[transactionID] = true
	return true, nil
}

// fakeIAPVerifier stands in for IAPService: configured, and returns a fixed
// transaction ID or error for any purchase token.
type fakeIAPVerifier struct {
	txID         string
	err          error
	lastPlatform string
	lastProduct  string
	lastToken    string
}

func (f *fakeIAPVerifier) Configured() bool { return true }

func (f *fakeIAPVerifier) VerifyPurchase(_ context.Context, platform, productID, purchaseToken string) (string, error) {
	f.lastPlatform, f.lastProduct, f.lastToken = platform, productID, purchaseToken
	if f.err != nil {
		return "", f.err
	}
	if f.txID != "" {
		return f.txID, nil
	}
	return "tx_" + purchaseToken, nil
}

func newVerifiedBillingRouter(t *testing.T, svc planManager, payments paymentRecorder, iap purchaseVerifier, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(billingTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := &BillingHandler{svc: svc, payments: payments, iap: iap}
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

// iapBody builds a valid verified-upgrade request body, with overrides.
func iapBody(overrides map[string]string) map[string]string {
	body := map[string]string{
		"plan":           "plus",
		"period":         "monthly",
		"platform":       "android",
		"product_id":     services.IAPProductPlusMonthly,
		"purchase_token": "token-1",
	}
	for k, v := range overrides {
		body[k] = v
	}
	return body
}

func TestBillingHandler_UpgradeVerified_MissingPurchaseToken_Returns400(t *testing.T) {
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "plus", "platform": "android", "product_id": services.IAPProductPlusMonthly})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without purchase_token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_MissingPlatform_Returns400(t *testing.T) {
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(map[string]string{"platform": ""}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without platform, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_UnknownProduct_Returns400(t *testing.T) {
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(map[string]string{"product_id": "com.dreamlog.app.mystery.box"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown product, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_PurchaseInvalid_Returns402(t *testing.T) {
	iap := &fakeIAPVerifier{err: services.ErrPurchaseInvalid}
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, iap, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(nil))
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 for invalid purchase, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_StoreUnreachable_Returns500(t *testing.T) {
	iap := &fakeIAPVerifier{err: errors.New("network down")}
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, iap, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when store verification is unreachable, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_WrongPlanForProduct_Returns400(t *testing.T) {
	// Bought the plus product, asking for pro.
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(map[string]string{"plan": "pro"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for plan mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_PeriodMismatch_Returns400(t *testing.T) {
	// Bought the monthly product, asking for an annual pass.
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(map[string]string{"period": "annual"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for period mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_Success_GrantsServerExpiry(t *testing.T) {
	svc := &fakePlanManager{}
	payments := &fakePaymentRecorder{}
	r := newVerifiedBillingRouter(t, svc, payments, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(map[string]string{
		"plan": "pro", "product_id": services.IAPProductProMonthly, "platform": "ios",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanPro {
		t.Fatalf("expected pro granted, got %s", svc.upgradedPlan)
	}
	if svc.upgradedExpiry == nil {
		t.Fatal("expected server-set expiry on verified upgrade")
	}
	if payments.lastStore != "apple" {
		t.Errorf("ios purchase must be recorded with store=apple, got %q", payments.lastStore)
	}
	if payments.lastProduct != services.IAPProductProMonthly {
		t.Errorf("product_id must be recorded, got %q", payments.lastProduct)
	}
}

func TestBillingHandler_UpgradeVerified_Android_RecordsGoogleStore(t *testing.T) {
	payments := &fakePaymentRecorder{}
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, payments, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	if w := postUpgrade(t, r, iapBody(nil)); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if payments.lastStore != "google" {
		t.Errorf("android purchase must be recorded with store=google, got %q", payments.lastStore)
	}
}

func TestBillingHandler_UpgradeVerified_Annual_GrantsYearExpiry(t *testing.T) {
	svc := &fakePlanManager{}
	r := newVerifiedBillingRouter(t, svc, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, iapBody(map[string]string{
		"plan": "pro", "period": "annual", "product_id": services.IAPProductProAnnual,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedExpiry == nil {
		t.Fatal("expected server-set expiry on verified annual upgrade")
	}
	// Expiry must be ~365 days out, not the 30-day monthly pass.
	if got := time.Until(*svc.upgradedExpiry); got < 360*24*time.Hour || got > 366*24*time.Hour {
		t.Fatalf("annual expiry must be ~365 days out, got %v", got)
	}
}

func TestBillingHandler_UpgradeVerified_ReplayedTransaction_Returns409(t *testing.T) {
	payments := &fakePaymentRecorder{}
	// Fixed transaction ID: both requests resolve to the same store transaction.
	iap := &fakeIAPVerifier{txID: "GPA.1234-5678"}
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, payments, iap, billingTestUser(models.PlanFree))

	if w := postUpgrade(t, r, iapBody(nil)); w.Code != http.StatusOK {
		t.Fatalf("first use must succeed, got %d: %s", w.Code, w.Body.String())
	}
	if w := postUpgrade(t, r, iapBody(nil)); w.Code != http.StatusConflict {
		t.Fatalf("replayed transaction must return 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_B2BSelfServe_Returns400(t *testing.T) {
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	w := postUpgrade(t, r, map[string]string{"plan": "b2b"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("b2b must not be self-serve in production, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_UpgradeVerified_DowngradeToFree_NoPurchaseNeeded(t *testing.T) {
	svc := &fakePlanManager{}
	r := newVerifiedBillingRouter(t, svc, &fakePaymentRecorder{}, &fakeIAPVerifier{}, billingTestUser(models.PlanPro))
	w := postUpgrade(t, r, map[string]string{"plan": "free"})
	if w.Code != http.StatusOK {
		t.Fatalf("downgrade to free must not require a purchase, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedPlan != models.PlanFree {
		t.Fatalf("expected free, got %s", svc.upgradedPlan)
	}
	if svc.upgradedExpiry != nil {
		t.Error("free plan must have nil expiry")
	}
}

func TestIAPCatalog_CoversAllPaidPlanPeriods(t *testing.T) {
	want := map[string]services.IAPProduct{
		services.IAPProductPlusMonthly: {Plan: models.PlanPlus, Period: "monthly"},
		services.IAPProductPlusAnnual:  {Plan: models.PlanPlus, Period: "annual"},
		services.IAPProductProMonthly:  {Plan: models.PlanPro, Period: "monthly"},
		services.IAPProductProAnnual:   {Plan: models.PlanPro, Period: "annual"},
	}
	if len(services.IAPCatalog) != len(want) {
		t.Fatalf("catalog size: want %d, got %d", len(want), len(services.IAPCatalog))
	}
	for id, p := range want {
		got, ok := services.IAPCatalog[id]
		if !ok || got != p {
			t.Errorf("catalog[%s]: want %+v, got %+v (ok=%v)", id, p, got, ok)
		}
	}
}

func TestNormalizePeriod(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"", "monthly", true},        // legacy requests / intents default to monthly
		{"monthly", "monthly", true},
		{"annual", "annual", true},
		{"yearly", "", false},
		{"weekly", "", false},
	}
	for _, tc := range cases {
		got, ok := normalizePeriod(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("normalizePeriod(%q): want (%q, %v), got (%q, %v)", tc.in, tc.want, tc.wantOK, got, ok)
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
