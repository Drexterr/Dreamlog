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
	r.Use(middleware.AuthMiddleware(billingTestSecret, &fakeProvisioner{user: testUser}, log))

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

func TestBillingHandler_Upgrade_WithExpiry_Returns200(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	svc := &fakePlanManager{}
	r := newBillingTestRouter(t, svc, user)

	expiry := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body, _ := json.Marshal(map[string]string{"plan": "pro", "expires_at": expiry})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/upgrade", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with expiry, got %d: %s", w.Code, w.Body.String())
	}
	if svc.upgradedExpiry == nil {
		t.Fatal("expected expiry to be passed through to service")
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
