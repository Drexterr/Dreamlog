package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
)

func newAppleTestService(t *testing.T, prodURL, sandboxURL string) *IAPService {
	t.Helper()
	s := NewIAPService(&appconfig.IAPConfig{AppleSharedSecret: "shared-secret"})
	s.appleBaseURL = prodURL
	s.appleSandboxBaseURL = sandboxURL
	return s
}

func appleStub(t *testing.T, resp map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verifyReceipt" {
			t.Errorf("unexpected apple path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestIAP_Configured(t *testing.T) {
	if NewIAPService(&appconfig.IAPConfig{}).Configured() {
		t.Error("no credentials must report not configured")
	}
	if !NewIAPService(&appconfig.IAPConfig{AppleSharedSecret: "x"}).Configured() {
		t.Error("apple secret alone must report configured")
	}
	if !NewIAPService(&appconfig.IAPConfig{GooglePlayCredentialsJSON: "{}"}).Configured() {
		t.Error("google credentials alone must report configured")
	}
}

func TestIAP_VerifyApple_Success_ReturnsLatestTransaction(t *testing.T) {
	apple := appleStub(t, map[string]any{
		"status": 0,
		"receipt": map[string]any{"bundle_id": "com.talktoode.app"},
		"latest_receipt_info": []map[string]any{
			{"product_id": IAPProductPlusMonthly, "transaction_id": "1000001", "purchase_date_ms": "1700000000000"},
			{"product_id": IAPProductPlusMonthly, "transaction_id": "1000002", "purchase_date_ms": "1800000000000"},
			{"product_id": IAPProductProMonthly, "transaction_id": "1000003", "purchase_date_ms": "1900000000000"},
		},
	})
	defer apple.Close()

	s := newAppleTestService(t, apple.URL, apple.URL)
	vp, err := s.VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "base64receipt", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vp.TransactionID != "1000002" {
		t.Fatalf("want newest matching transaction 1000002, got %s", vp.TransactionID)
	}
}

func TestIAP_VerifyApple_SandboxRetry(t *testing.T) {
	sandbox := appleStub(t, map[string]any{
		"status": 0,
		"receipt": map[string]any{
			"bundle_id": "com.talktoode.app",
			"in_app": []map[string]any{
				{"product_id": IAPProductProAnnual, "transaction_id": "2000001", "purchase_date_ms": "1700000000000"},
			},
		},
	})
	defer sandbox.Close()

	// Production endpoint answers 21007: "this is a sandbox receipt".
	prod := appleStub(t, map[string]any{"status": 21007})
	defer prod.Close()

	s := newAppleTestService(t, prod.URL, sandbox.URL)
	vp, err := s.VerifyPurchase(context.Background(), "ios", IAPProductProAnnual, "sandboxreceipt", "")
	if err != nil {
		t.Fatalf("sandbox receipt must verify via retry, got: %v", err)
	}
	if vp.TransactionID != "2000001" {
		t.Fatalf("want 2000001, got %s", vp.TransactionID)
	}
}

func TestIAP_VerifyApple_BadStatus_IsPurchaseInvalid(t *testing.T) {
	apple := appleStub(t, map[string]any{"status": 21003}) // receipt could not be authenticated
	defer apple.Close()

	s := newAppleTestService(t, apple.URL, apple.URL)
	_, err := s.VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "junk", "")
	if !errors.Is(err, ErrPurchaseInvalid) {
		t.Fatalf("want ErrPurchaseInvalid, got %v", err)
	}
}

func TestIAP_VerifyApple_ProductNotInReceipt_IsPurchaseInvalid(t *testing.T) {
	apple := appleStub(t, map[string]any{
		"status": 0,
		"latest_receipt_info": []map[string]any{
			{"product_id": IAPProductPlusMonthly, "transaction_id": "1", "purchase_date_ms": "1"},
		},
	})
	defer apple.Close()

	s := newAppleTestService(t, apple.URL, apple.URL)
	_, err := s.VerifyPurchase(context.Background(), "ios", IAPProductProAnnual, "receipt", "")
	if !errors.Is(err, ErrPurchaseInvalid) {
		t.Fatalf("receipt without the product must be invalid, got %v", err)
	}
}

func TestIAP_VerifyApple_NotConfigured(t *testing.T) {
	s := NewIAPService(&appconfig.IAPConfig{})
	_, err := s.VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "receipt", "")
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("want ErrStoreNotConfigured, got %v", err)
	}
}

func TestIAP_VerifyApple_StoreUnreachable_IsNotPurchaseInvalid(t *testing.T) {
	s := newAppleTestService(t, "http://127.0.0.1:1", "http://127.0.0.1:1")
	_, err := s.VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "receipt", "")
	if err == nil {
		t.Fatal("expected error when apple is unreachable")
	}
	if errors.Is(err, ErrPurchaseInvalid) {
		t.Fatal("an outage must not be classified as an invalid purchase")
	}
}

// ── Google ────────────────────────────────────────────────────────────────────

func newGoogleTestService(t *testing.T, baseURL string) *IAPService {
	t.Helper()
	s := NewIAPService(&appconfig.IAPConfig{
		GooglePlayCredentialsJSON: `{"type":"service_account"}`,
		GooglePlayPackageName:     "com.talktoode.app",
	})
	s.googleBaseURL = baseURL
	s.googleTokenOverride = func() (string, error) { return "test-access-token", nil }
	return s
}

func googleStub(t *testing.T, status int, resp map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
			t.Errorf("missing bearer token, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestIAP_VerifyGoogle_Success_ReturnsOrderID(t *testing.T) {
	google := googleStub(t, http.StatusOK, map[string]any{"purchaseState": 0, "orderId": "GPA.1234-5678"})
	defer google.Close()

	s := newGoogleTestService(t, google.URL)
	vp, err := s.VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "play-token", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vp.TransactionID != "GPA.1234-5678" {
		t.Fatalf("want orderId, got %s", vp.TransactionID)
	}
}

func TestIAP_VerifyGoogle_NoOrderID_FallsBackToToken(t *testing.T) {
	google := googleStub(t, http.StatusOK, map[string]any{"purchaseState": 0})
	defer google.Close()

	s := newGoogleTestService(t, google.URL)
	vp, err := s.VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "promo-token", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vp.TransactionID != "promo-token" {
		t.Fatalf("want fallback to purchase token, got %s", vp.TransactionID)
	}
}

func TestIAP_VerifyGoogle_Pending_IsPurchaseInvalid(t *testing.T) {
	google := googleStub(t, http.StatusOK, map[string]any{"purchaseState": 2, "orderId": "GPA.9"})
	defer google.Close()

	s := newGoogleTestService(t, google.URL)
	_, err := s.VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "pending-token", "")
	if !errors.Is(err, ErrPurchaseInvalid) {
		t.Fatalf("pending purchase must be invalid, got %v", err)
	}
}

func TestIAP_VerifyGoogle_UnknownToken_IsPurchaseInvalid(t *testing.T) {
	google := googleStub(t, http.StatusNotFound, map[string]any{})
	defer google.Close()

	s := newGoogleTestService(t, google.URL)
	_, err := s.VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "forged-token", "")
	if !errors.Is(err, ErrPurchaseInvalid) {
		t.Fatalf("unknown token must be invalid, got %v", err)
	}
}

func TestIAP_VerifyGoogle_ServerError_IsNotPurchaseInvalid(t *testing.T) {
	google := googleStub(t, http.StatusInternalServerError, map[string]any{})
	defer google.Close()

	s := newGoogleTestService(t, google.URL)
	_, err := s.VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "token", "")
	if err == nil {
		t.Fatal("expected error on play outage")
	}
	if errors.Is(err, ErrPurchaseInvalid) {
		t.Fatal("a play outage must not be classified as an invalid purchase")
	}
}

func TestIAP_VerifyGoogle_NotConfigured(t *testing.T) {
	s := NewIAPService(&appconfig.IAPConfig{AppleSharedSecret: "x"})
	_, err := s.VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "token", "")
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("want ErrStoreNotConfigured, got %v", err)
	}
}

func TestIAP_VerifyPurchase_UnsupportedPlatform(t *testing.T) {
	s := NewIAPService(&appconfig.IAPConfig{AppleSharedSecret: "x"})
	if _, err := s.VerifyPurchase(context.Background(), "web", IAPProductPlusMonthly, "token", ""); err == nil {
		t.Fatal("expected error for unsupported platform")
	}
}

func appleOK(items []map[string]any, bundle string) map[string]any {
	return map[string]any{
		"status":              0,
		"receipt":             map[string]any{"bundle_id": bundle},
		"latest_receipt_info": items,
	}
}

// H12: a receipt from another app with a same-named SKU must be rejected.
func TestIAP_VerifyApple_WrongBundle_IsPurchaseInvalid(t *testing.T) {
	apple := appleStub(t, appleOK([]map[string]any{
		{"product_id": IAPProductProAnnual, "transaction_id": "9", "purchase_date_ms": "1"},
	}, "com.attacker.app"))
	defer apple.Close()

	s := newAppleTestService(t, apple.URL, apple.URL)
	if _, err := s.VerifyPurchase(context.Background(), "ios", IAPProductProAnnual, "receipt", ""); !errors.Is(err, ErrPurchaseInvalid) {
		t.Fatalf("foreign bundle must be ErrPurchaseInvalid, got %v", err)
	}
}

// H12: sandbox receipts are flagged so the handler can refuse them.
func TestIAP_VerifyApple_SandboxIsFlagged(t *testing.T) {
	sandbox := appleStub(t, appleOK([]map[string]any{
		{"product_id": IAPProductPlusMonthly, "transaction_id": "5", "purchase_date_ms": "1"},
	}, "com.talktoode.app"))
	defer sandbox.Close()
	prod := appleStub(t, map[string]any{"status": 21007})
	defer prod.Close()

	vp, err := newAppleTestService(t, prod.URL, sandbox.URL).VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "r", "")
	if err != nil || !vp.Sandbox {
		t.Fatalf("sandbox receipt must verify with Sandbox=true: %+v %v", vp, err)
	}
}

// H11: a named transaction is matched exactly - never the newest older one.
func TestIAP_VerifyApple_TransactionHint(t *testing.T) {
	apple := appleStub(t, appleOK([]map[string]any{
		{"product_id": IAPProductPlusMonthly, "transaction_id": "old", "purchase_date_ms": "2"},
		{"product_id": IAPProductPlusMonthly, "transaction_id": "new", "purchase_date_ms": "1"},
	}, "com.talktoode.app"))
	defer apple.Close()
	s := newAppleTestService(t, apple.URL, apple.URL)

	vp, err := s.VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "r", "new")
	if err != nil || vp.TransactionID != "new" {
		t.Fatalf("want hinted transaction 'new', got %+v %v", vp, err)
	}
	if _, err := s.VerifyPurchase(context.Background(), "ios", IAPProductPlusMonthly, "r", "missing"); !errors.Is(err, ErrPurchaseInvalid) {
		t.Fatalf("a hinted transaction absent from the receipt must be invalid, got %v", err)
	}
}

// H12: Play license-tester purchases are flagged as sandbox.
func TestIAP_VerifyGoogle_TestPurchaseIsFlagged(t *testing.T) {
	google := googleStub(t, http.StatusOK, map[string]any{"purchaseState": 0, "orderId": "GPA.1", "purchaseType": 0})
	defer google.Close()
	vp, err := newGoogleTestService(t, google.URL).VerifyPurchase(context.Background(), "android", IAPProductPlusMonthly, "t", "")
	if err != nil || !vp.Sandbox {
		t.Fatalf("test purchase must be flagged sandbox: %+v %v", vp, err)
	}
}

func TestIAP_SandboxAllowed(t *testing.T) {
	s := NewIAPService(&appconfig.IAPConfig{SandboxAllowedEmails: []string{"Reviewer@Example.com"}})
	if !s.SandboxAllowed("reviewer@example.com") {
		t.Error("allowlisted email (case-insensitive) must be allowed")
	}
	if s.SandboxAllowed("someone@example.com") || s.SandboxAllowed("") {
		t.Error("other or empty emails must not be allowed")
	}
}
