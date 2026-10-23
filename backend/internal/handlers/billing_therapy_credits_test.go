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
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// fakeCreditStore mirrors PaymentRepository's therapy-credit methods. Grants
// share ownership/granted state with the fakePaymentRecorder, like the real
// repository shares the payments table.
type fakeCreditStore struct {
	payments   *fakePaymentRecorder
	balance    int
	grantCalls int
	devGrants  int
	readErr    error
	grantErr   error
}

func (f *fakeCreditStore) TherapyCredits(context.Context, uuid.UUID) (int, error) {
	return f.balance, f.readErr
}

func (f *fakeCreditStore) GrantTherapyCredits(_ context.Context, transactionID string, userID uuid.UUID, credits int) (int, bool, error) {
	if f.grantErr != nil {
		return 0, false, f.grantErr
	}
	if f.payments.granted == nil {
		f.payments.granted = map[string]bool{}
	}
	if owner, ok := f.payments.seen[transactionID]; !ok || owner != userID {
		return 0, false, repositories.ErrPaymentNotOwned
	}
	if f.payments.granted[transactionID] {
		return f.balance, false, nil
	}
	f.grantCalls++
	f.balance += credits
	f.payments.granted[transactionID] = true
	return f.balance, true, nil
}

func (f *fakeCreditStore) DevGrantTherapyCredits(_ context.Context, _ uuid.UUID, credits int) (int, error) {
	f.devGrants++
	f.balance += credits
	return f.balance, nil
}

func newCreditsRouter(t *testing.T, payments *fakePaymentRecorder, iap purchaseVerifier, credits therapyCreditStore, user *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(billingTestSecret, "", &fakeProvisioner{user: user}, log))
	h := NewBillingHandler(&fakePlanManager{}, payments, nil, iap, credits)
	r.GET("/billing/plan", h.GetPlan)
	r.POST("/billing/therapy-credits", h.BuyTherapyCredits)
	return r
}

func creditBody(overrides map[string]string) map[string]string {
	body := map[string]string{
		"platform":       "android",
		"product_id":     services.IAPProductTherapySingle,
		"purchase_token": "credit-token-1",
	}
	for k, v := range overrides {
		body[k] = v
	}
	return body
}

func postCredits(t *testing.T, r *gin.Engine, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/therapy-credits", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func creditResp(t *testing.T, w *httptest.ResponseRecorder) (balance, granted float64) {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal %s: %v", w.Body.String(), err)
	}
	return resp["therapy_credits"].(float64), resp["credits_granted"].(float64)
}

func TestBillingTherapyCredits_VerifiedPurchase_GrantsOneCredit(t *testing.T) {
	payments := &fakePaymentRecorder{}
	iap := &fakeIAPVerifier{}
	credits := &fakeCreditStore{payments: payments}
	r := newCreditsRouter(t, payments, iap, credits, billingTestUser(models.PlanFree))

	w := postCredits(t, r, creditBody(nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if bal, granted := creditResp(t, w); bal != 1 || granted != 1 {
		t.Fatalf("balance=%v granted=%v, want 1/1", bal, granted)
	}
	if iap.lastProduct != services.IAPProductTherapySingle || iap.lastPlatform != "android" {
		t.Fatalf("store verification not called for the product: %+v", iap)
	}
	if payments.lastStore != "google" || payments.lastProduct != services.IAPProductTherapySingle {
		t.Fatalf("payment not recorded: store=%s product=%s", payments.lastStore, payments.lastProduct)
	}
}

func TestBillingTherapyCredits_SameUserReplay_IsIdempotent(t *testing.T) {
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{txID: "GPA.credit-1"}, credits, billingTestUser(models.PlanFree))

	for i := 0; i < 3; i++ {
		if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusOK {
			t.Fatalf("replay %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	if credits.grantCalls != 1 || credits.balance != 1 {
		t.Fatalf("one purchase must credit once: grants=%d balance=%d", credits.grantCalls, credits.balance)
	}
	w := postCredits(t, r, creditBody(nil))
	if _, granted := creditResp(t, w); granted != 0 {
		t.Fatalf("replay must report credits_granted=0, got %v", granted)
	}
}

func TestBillingTherapyCredits_RecordedButUngranted_GrantsOnRetry(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	payments := &fakePaymentRecorder{seen: map[string]uuid.UUID{"GPA.retry": user.ID}}
	credits := &fakeCreditStore{payments: payments}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{txID: "GPA.retry"}, credits, user)

	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if credits.grantCalls != 1 {
		t.Fatal("a recorded but ungranted purchase must be granted on retry")
	}
}

func TestBillingTherapyCredits_OtherUsersTransaction_Returns409(t *testing.T) {
	payments := &fakePaymentRecorder{seen: map[string]uuid.UUID{"GPA.taken": uuid.New()}}
	credits := &fakeCreditStore{payments: payments}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{txID: "GPA.taken"}, credits, billingTestUser(models.PlanFree))

	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
	if credits.grantCalls != 0 {
		t.Fatal("another user's purchase must never be credited")
	}
}

func TestBillingTherapyCredits_AccountTokenMismatch_Returns409(t *testing.T) {
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments}
	iap := &fakeIAPVerifier{accountToken: services.PurchaseAccountToken(uuid.New())}
	r := newCreditsRouter(t, payments, iap, credits, billingTestUser(models.PlanFree))

	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
	if len(payments.seen) != 0 || credits.grantCalls != 0 {
		t.Fatal("a purchase tagged for another account must not be recorded or granted")
	}
}

func TestBillingTherapyCredits_AccountTokenMatch_Granted(t *testing.T) {
	user := billingTestUser(models.PlanFree)
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments}
	iap := &fakeIAPVerifier{accountToken: services.PurchaseAccountToken(user.ID)}
	r := newCreditsRouter(t, payments, iap, credits, user)
	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestBillingTherapyCredits_Sandbox(t *testing.T) {
	cases := []struct {
		name     string
		verified bool
		allowed  bool
		want     int
	}{
		{"not allowlisted", true, false, http.StatusPaymentRequired},
		{"allowlisted but unverified email", false, true, http.StatusPaymentRequired},
		{"allowlisted verified email", true, true, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := billingTestUser(models.PlanFree)
			user.EmailVerified = tc.verified
			payments := &fakePaymentRecorder{}
			credits := &fakeCreditStore{payments: payments}
			r := newCreditsRouter(t, payments, &fakeIAPVerifier{sandbox: true, allowSandbox: tc.allowed}, credits, user)
			w := postCredits(t, r, creditBody(nil))
			if w.Code != tc.want {
				t.Fatalf("want %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
			if tc.want != http.StatusOK && credits.grantCalls != 0 {
				t.Fatal("refused sandbox purchase was credited")
			}
			if tc.want == http.StatusOK && !payments.lastSandbox {
				t.Fatal("sandbox purchase must be recorded as sandbox")
			}
		})
	}
}

func TestBillingTherapyCredits_StoreRejects_Returns402(t *testing.T) {
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{err: services.ErrPurchaseInvalid}, credits, billingTestUser(models.PlanFree))
	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusPaymentRequired {
		t.Fatalf("want 402, got %d", w.Code)
	}
	if credits.grantCalls != 0 {
		t.Fatal("rejected purchase credited")
	}
}

func TestBillingTherapyCredits_StoreUnavailable_Returns500(t *testing.T) {
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{err: errors.New("store down")}, credits, billingTestUser(models.PlanFree))
	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestBillingTherapyCredits_GrantFailure_Returns500(t *testing.T) {
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments, grantErr: errors.New("db down")}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, credits, billingTestUser(models.PlanFree))
	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestBillingTherapyCredits_WrongKindPayment_Returns409(t *testing.T) {
	payments := &fakePaymentRecorder{}
	credits := &fakeCreditStore{payments: payments, grantErr: repositories.ErrPaymentWrongKind}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, credits, billingTestUser(models.PlanFree))
	if w := postCredits(t, r, creditBody(nil)); w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestBillingTherapyCredits_Validation(t *testing.T) {
	cases := []struct {
		name string
		body map[string]string
	}{
		{"unknown product", creditBody(map[string]string{"product_id": "com.someone.else"})},
		{"plan product is not a credit", creditBody(map[string]string{"product_id": services.IAPProductPlusMonthly})},
		{"missing product", creditBody(map[string]string{"product_id": ""})},
		{"bad platform", creditBody(map[string]string{"platform": "web"})},
		{"missing token", creditBody(map[string]string{"purchase_token": ""})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payments := &fakePaymentRecorder{}
			credits := &fakeCreditStore{payments: payments}
			r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, credits, billingTestUser(models.PlanFree))
			if w := postCredits(t, r, tc.body); w.Code != http.StatusBadRequest {
				t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
			}
			if credits.grantCalls != 0 {
				t.Fatal("invalid request credited")
			}
		})
	}
	t.Run("malformed json", func(t *testing.T) {
		payments := &fakePaymentRecorder{}
		r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, &fakeCreditStore{payments: payments}, billingTestUser(models.PlanFree))
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/billing/therapy-credits", bytes.NewReader([]byte("{")))
		req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})
}

func TestBillingTherapyCredits_PlanSKUOnUpgradeIsNotACredit(t *testing.T) {
	// The upgrade endpoint must never accept a credit SKU as a plan.
	payments := &fakePaymentRecorder{}
	r := newVerifiedBillingRouter(t, &fakePlanManager{}, payments, &fakeIAPVerifier{}, billingTestUser(models.PlanFree))
	if w := postUpgrade(t, r, iapBody(map[string]string{"product_id": services.IAPProductTherapySingle})); w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestBillingTherapyCredits_MemberPrice(t *testing.T) {
	cases := []struct {
		plan models.Plan
		want int
	}{
		{models.PlanFree, http.StatusForbidden},
		{models.PlanPlus, http.StatusForbidden},
		{models.PlanPro, http.StatusOK},
		{models.PlanB2B, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(string(tc.plan), func(t *testing.T) {
			payments := &fakePaymentRecorder{}
			credits := &fakeCreditStore{payments: payments}
			r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, credits, billingTestUser(tc.plan))
			w := postCredits(t, r, creditBody(map[string]string{"product_id": services.IAPProductTherapyMember}))
			if w.Code != tc.want {
				t.Fatalf("want %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
			if tc.want == http.StatusForbidden {
				if credits.grantCalls != 0 {
					t.Fatal("member price credited to a non-Pro account")
				}
				if len(payments.seen) != 1 {
					t.Fatal("the charged purchase must still be recorded so it can be redeemed later")
				}
			}
		})
	}
}

func TestBillingTherapyCredits_MemberPrice_ExpiredProIsFree(t *testing.T) {
	user := billingTestUser(models.PlanPro)
	past := time.Now().Add(-time.Hour)
	user.PlanExpiresAt = &past
	payments := &fakePaymentRecorder{}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, &fakeCreditStore{payments: payments}, user)
	if w := postCredits(t, r, creditBody(map[string]string{"product_id": services.IAPProductTherapyMember})); w.Code != http.StatusForbidden {
		t.Fatalf("an expired Pro pass must not get the member price, got %d", w.Code)
	}
}

func TestBillingTherapyCredits_DevStub(t *testing.T) {
	t.Run("grants without verification", func(t *testing.T) {
		credits := &fakeCreditStore{payments: &fakePaymentRecorder{}}
		r := newCreditsRouter(t, &fakePaymentRecorder{}, nil, credits, billingTestUser(models.PlanFree))
		w := postCredits(t, r, map[string]string{"product_id": services.IAPProductTherapySingle})
		if w.Code != http.StatusOK || credits.devGrants != 1 || credits.balance != 1 {
			t.Fatalf("dev grant: %d %s devGrants=%d", w.Code, w.Body.String(), credits.devGrants)
		}
	})
	t.Run("still enforces the member price rule", func(t *testing.T) {
		credits := &fakeCreditStore{payments: &fakePaymentRecorder{}}
		r := newCreditsRouter(t, &fakePaymentRecorder{}, nil, credits, billingTestUser(models.PlanFree))
		w := postCredits(t, r, map[string]string{"product_id": services.IAPProductTherapyMember})
		if w.Code != http.StatusForbidden || credits.devGrants != 0 {
			t.Fatalf("want 403, got %d", w.Code)
		}
	})
	t.Run("still requires a known product", func(t *testing.T) {
		credits := &fakeCreditStore{payments: &fakePaymentRecorder{}}
		r := newCreditsRouter(t, &fakePaymentRecorder{}, nil, credits, billingTestUser(models.PlanFree))
		if w := postCredits(t, r, map[string]string{}); w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
	})
}

func TestBillingTherapyCredits_MissingAuth_Returns401(t *testing.T) {
	payments := &fakePaymentRecorder{}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, &fakeCreditStore{payments: payments}, billingTestUser(models.PlanFree))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billing/therapy-credits", bytes.NewReader([]byte(`{}`)))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestBillingHandler_GetPlan_IncludesTherapyCredits(t *testing.T) {
	payments := &fakePaymentRecorder{}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, &fakeCreditStore{payments: payments, balance: 3}, billingTestUser(models.PlanFree))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["therapy_credits"] != float64(3) {
		t.Fatalf("therapy_credits = %v, want 3", resp["therapy_credits"])
	}
}

func TestBillingHandler_GetPlan_TherapyCreditsReadError_Returns500(t *testing.T) {
	payments := &fakePaymentRecorder{}
	r := newCreditsRouter(t, payments, &fakeIAPVerifier{}, &fakeCreditStore{payments: payments, readErr: errors.New("db")}, billingTestUser(models.PlanFree))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billing/plan", nil)
	req.Header.Set("Authorization", "Bearer "+billingTestJWT(t))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
