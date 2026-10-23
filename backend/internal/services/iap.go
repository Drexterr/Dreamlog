package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// IAP product catalogue. Product IDs must match the SKUs configured in App
// Store Connect and the Play Console exactly, and the same IDs live in
// mobile/src/services/iap.ts - keep all three in sync.
//
// Passes are sold as consumable products (a 30-day or 365-day pass), not
// auto-renewing subscriptions, matching the existing one-time-pass model.
const (
	IAPProductPlusMonthly = "com.talktoode.app.plus.monthly"
	IAPProductPlusAnnual  = "com.talktoode.app.plus.annual"
	IAPProductProMonthly  = "com.talktoode.app.pro.monthly"
	IAPProductProAnnual   = "com.talktoode.app.pro.annual"
)

// IAPProduct describes what a store product ID grants.
type IAPProduct struct {
	Plan   models.Plan
	Period string // "monthly" | "annual"
}

// IAPCatalog maps store product IDs to the plan and pass length they grant.
// The backend trusts the store for pricing; it only needs to know what the
// verified product unlocks.
var IAPCatalog = map[string]IAPProduct{
	IAPProductPlusMonthly: {Plan: models.PlanPlus, Period: "monthly"},
	IAPProductPlusAnnual:  {Plan: models.PlanPlus, Period: "annual"},
	IAPProductProMonthly:  {Plan: models.PlanPro, Period: "monthly"},
	IAPProductProAnnual:   {Plan: models.PlanPro, Period: "annual"},
}

// Sentinel errors let the handler distinguish a bad purchase (client's fault,
// 402/400) from the verification service being unreachable (500).
var (
	// ErrPurchaseInvalid: the store looked at the receipt/token and said no -
	// not purchased, refunded, canceled, or for a different product.
	ErrPurchaseInvalid = errors.New("iap: purchase is not valid")
	// ErrStoreNotConfigured: credentials for the requested store are missing.
	ErrStoreNotConfigured = errors.New("iap: store credentials not configured")
)

const (
	appleProductionVerifyURL = "https://buy.itunes.apple.com"
	appleSandboxVerifyURL    = "https://sandbox.itunes.apple.com"
	googlePublisherBaseURL   = "https://androidpublisher.googleapis.com"

	// Apple verifyReceipt status codes we branch on.
	appleStatusOK          = 0
	appleStatusSandboxUsed = 21007 // production endpoint received a sandbox receipt

	// defaultAppleBundleID is the iOS bundle identifier receipts must carry.
	defaultAppleBundleID = "com.talktoode.app"
)

// IAPService verifies In-App Purchases server-side before any plan is granted.
// Never trust purchase state reported by the client - the same rule as the
// old Stripe verification path.
type IAPService struct {
	cfg        *appconfig.IAPConfig
	httpClient *http.Client

	// Test overrides; empty/nil = real store endpoints and OAuth exchange.
	appleBaseURL        string
	appleSandboxBaseURL string
	googleBaseURL       string
	googleTokenOverride func() (string, error)

	tokenOnce   sync.Once
	tokenSource oauth2.TokenSource
	tokenErr    error
}

func NewIAPService(cfg *appconfig.IAPConfig) *IAPService {
	return &IAPService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Configured reports whether at least one store can be verified. When false
// (local dev), the billing handler grants plans without verification, same as
// the old dev-stub behaviour.
func (s *IAPService) Configured() bool {
	return s.cfg.AppleSharedSecret != "" || s.cfg.GooglePlayCredentialsJSON != ""
}

// VerifiedPurchase is the result of a successful store verification.
type VerifiedPurchase struct {
	// TransactionID is the store's unique transaction ID, used for replay
	// protection (payments.transaction_id UNIQUE).
	TransactionID string
	// Sandbox is true for App Store sandbox / TestFlight receipts and Play
	// license-tester purchases - no money changed hands. The handler only
	// honours these for allowlisted accounts (SandboxAllowed).
	Sandbox bool
	// AccountToken is the account the purchase was tagged with at purchase
	// time (Play obfuscatedExternalAccountId / Apple app_account_token), or ""
	// when the purchase carries none (older app versions).
	AccountToken string
}

// PurchaseAccountToken is the opaque per-user id the app attaches to every
// store purchase (Play obfuscatedAccountId, Apple appAccountToken). It is
// UUID-shaped because Apple requires a UUID, carries no PII, and lets the
// backend refuse a purchase redeemed by a different account (N17).
func PurchaseAccountToken(userID uuid.UUID) string {
	sum := sha256.Sum256([]byte("ode-purchase-account:" + userID.String()))
	var id uuid.UUID
	copy(id[:], sum[:16])
	id[6] = (id[6] & 0x0f) | 0x50 // version 5 (name-based, SHA)
	id[8] = (id[8] & 0x3f) | 0x80 // RFC 4122 variant
	return id.String()
}

// AccountMatches reports whether a verified purchase may be redeemed by the
// user: untagged purchases (older clients) are accepted; tagged ones only by
// the account they were made for.
func (v *VerifiedPurchase) AccountMatches(userID uuid.UUID) bool {
	return v.AccountToken == "" || strings.EqualFold(v.AccountToken, PurchaseAccountToken(userID))
}

// VerifyPurchase validates a store purchase.
//
//	platform "ios":     purchaseToken is the base64 app receipt (transactionReceipt);
//	                    transactionID (optional) pins the exact transaction to redeem
//	platform "android": purchaseToken is the Play Billing purchase token
func (s *IAPService) VerifyPurchase(ctx context.Context, platform, productID, purchaseToken, transactionID string) (*VerifiedPurchase, error) {
	switch platform {
	case "ios":
		return s.verifyApple(ctx, productID, purchaseToken, transactionID)
	case "android":
		return s.verifyGoogle(ctx, productID, purchaseToken)
	default:
		return nil, fmt.Errorf("iap: unsupported platform %q", platform)
	}
}

// SandboxAllowed reports whether a sandbox/test purchase may grant a plan to
// this account. Production only accepts them for allowlisted accounts (the App
// Review demo account, internal testers) - otherwise any free TestFlight
// purchase would unlock a real plan.
func (s *IAPService) SandboxAllowed(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	for _, allowed := range s.cfg.SandboxAllowedEmails {
		if strings.ToLower(strings.TrimSpace(allowed)) == email {
			return true
		}
	}
	return false
}

// ── Apple ─────────────────────────────────────────────────────────────────────

type appleVerifyResponse struct {
	Status            int              `json:"status"`
	Environment       string           `json:"environment"` // "Production" | "Sandbox"
	LatestReceiptInfo []appleInAppItem `json:"latest_receipt_info"`
	Receipt           struct {
		BundleID string           `json:"bundle_id"`
		InApp    []appleInAppItem `json:"in_app"`
	} `json:"receipt"`
}

type appleInAppItem struct {
	ProductID       string `json:"product_id"`
	TransactionID   string `json:"transaction_id"`
	PurchaseDate    string `json:"purchase_date_ms"`
	AppAccountToken string `json:"app_account_token"`
}

// verifyApple validates the receipt with Apple's verifyReceipt endpoint and
// returns the transaction ID of the newest purchase of productID. A sandbox
// receipt sent to production (status 21007) is retried against the sandbox
// endpoint - Apple's documented flow, and required for App Review.
func (s *IAPService) verifyApple(ctx context.Context, productID, receipt, transactionID string) (*VerifiedPurchase, error) {
	if s.cfg.AppleSharedSecret == "" {
		return nil, ErrStoreNotConfigured
	}

	base := s.appleBaseURL
	if base == "" {
		base = appleProductionVerifyURL
	}
	resp, err := s.appleVerifyCall(ctx, base, receipt)
	if err != nil {
		return nil, err
	}

	sandbox := false
	if resp.Status == appleStatusSandboxUsed {
		sandbox = true
		sandboxURL := s.appleSandboxBaseURL
		if sandboxURL == "" {
			sandboxURL = appleSandboxVerifyURL
		}
		if resp, err = s.appleVerifyCall(ctx, sandboxURL, receipt); err != nil {
			return nil, err
		}
	}
	if resp.Status != appleStatusOK {
		return nil, fmt.Errorf("%w: apple status %d", ErrPurchaseInvalid, resp.Status)
	}
	if strings.EqualFold(resp.Environment, "Sandbox") {
		sandbox = true
	}

	// Product IDs are only unique per app: a receipt from someone else's app
	// with a same-named SKU must never grant a plan here.
	wantBundle := s.cfg.AppleBundleID
	if wantBundle == "" {
		wantBundle = defaultAppleBundleID
	}
	if resp.Receipt.BundleID != wantBundle {
		return nil, fmt.Errorf("%w: receipt bundle_id %q", ErrPurchaseInvalid, resp.Receipt.BundleID)
	}

	// Prefer latest_receipt_info; fall back to receipt.in_app. When the client
	// names the transaction it is redeeming, match exactly that one - never
	// fall back to an older (already used) transaction of the same product.
	// Otherwise pick the most recent transaction for the product.
	items := resp.LatestReceiptInfo
	if len(items) == 0 {
		items = resp.Receipt.InApp
	}
	var best *appleInAppItem
	for i := range items {
		if items[i].ProductID != productID {
			continue
		}
		if transactionID != "" {
			if items[i].TransactionID == transactionID {
				best = &items[i]
			}
			continue
		}
		if best == nil || items[i].PurchaseDate > best.PurchaseDate {
			best = &items[i]
		}
	}
	if best == nil || best.TransactionID == "" {
		return nil, fmt.Errorf("%w: receipt contains no matching purchase of %s", ErrPurchaseInvalid, productID)
	}
	return &VerifiedPurchase{TransactionID: best.TransactionID, Sandbox: sandbox, AccountToken: best.AppAccountToken}, nil
}

func (s *IAPService) appleVerifyCall(ctx context.Context, baseURL, receipt string) (*appleVerifyResponse, error) {
	body, err := json.Marshal(map[string]any{
		"receipt-data":             receipt,
		"password":                 s.cfg.AppleSharedSecret,
		"exclude-old-transactions": false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/verifyReceipt", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iap: apple verify: %w", err)
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("iap: apple verify read: %w", err)
	}
	var resp appleVerifyResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("iap: apple verify decode: %w", err)
	}
	return &resp, nil
}

// ── Google ────────────────────────────────────────────────────────────────────

type googleProductPurchase struct {
	PurchaseState int    `json:"purchaseState"` // 0 purchased, 1 canceled, 2 pending
	OrderID       string `json:"orderId"`
	// PurchaseType is only present for non-real purchases: 0 = test
	// (license tester), 1 = promo, 2 = rewarded.
	PurchaseType *int `json:"purchaseType"`
	// ObfuscatedExternalAccountId is the account id the app set on the purchase.
	ObfuscatedExternalAccountID string `json:"obfuscatedExternalAccountId"`
}

// verifyGoogle validates a Play Billing purchase token via the Android
// Publisher API (purchases.products.get) and returns the order ID.
func (s *IAPService) verifyGoogle(ctx context.Context, productID, purchaseToken string) (*VerifiedPurchase, error) {
	if s.cfg.GooglePlayCredentialsJSON == "" {
		return nil, ErrStoreNotConfigured
	}

	accessToken, err := s.googleAccessToken()
	if err != nil {
		return nil, err
	}

	base := s.googleBaseURL
	if base == "" {
		base = googlePublisherBaseURL
	}
	endpoint := fmt.Sprintf("%s/androidpublisher/v3/applications/%s/purchases/products/%s/tokens/%s",
		base,
		url.PathEscape(s.cfg.GooglePlayPackageName),
		url.PathEscape(productID),
		url.PathEscape(purchaseToken),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	httpResp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iap: google verify: %w", err)
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("iap: google verify read: %w", err)
	}

	// Play returns 400/404 for tokens it has never issued - the client's
	// fault, not an outage.
	if httpResp.StatusCode == http.StatusBadRequest || httpResp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: play returned %d", ErrPurchaseInvalid, httpResp.StatusCode)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iap: google verify returned %d: %s", httpResp.StatusCode, string(raw))
	}

	var purchase googleProductPurchase
	if err := json.Unmarshal(raw, &purchase); err != nil {
		return nil, fmt.Errorf("iap: google verify decode: %w", err)
	}
	if purchase.PurchaseState != 0 {
		return nil, fmt.Errorf("%w: purchaseState %d", ErrPurchaseInvalid, purchase.PurchaseState)
	}

	// orderId is unique per purchase; fall back to the token for test/promo
	// purchases, which can lack one.
	testPurchase := purchase.PurchaseType != nil && *purchase.PurchaseType == 0
	txID := purchase.OrderID
	if txID == "" {
		txID = purchaseToken
	}
	return &VerifiedPurchase{TransactionID: txID, Sandbox: testPurchase, AccountToken: purchase.ObfuscatedExternalAccountID}, nil
}

// googleAccessToken exchanges the service account for a short-lived OAuth2
// token, cached and auto-refreshed - same pattern as FCMService.
func (s *IAPService) googleAccessToken() (string, error) {
	if s.googleTokenOverride != nil {
		return s.googleTokenOverride()
	}
	s.tokenOnce.Do(func() {
		conf, err := google.JWTConfigFromJSON(
			[]byte(s.cfg.GooglePlayCredentialsJSON),
			"https://www.googleapis.com/auth/androidpublisher",
		)
		if err != nil {
			s.tokenErr = fmt.Errorf("iap: parse google credentials: %w", err)
			return
		}
		s.tokenSource = conf.TokenSource(context.Background())
	})
	if s.tokenErr != nil {
		return "", s.tokenErr
	}
	token, err := s.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("iap: fetch google access token: %w", err)
	}
	return token.AccessToken, nil
}
