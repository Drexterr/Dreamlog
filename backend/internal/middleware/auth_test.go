package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── helper: mint a test JWT ───────────────────────────────────────────────────

func mintTestJWT(t *testing.T, secret, sub, email string, expiry time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   sub,
		"email": email,
		"exp":   time.Now().Add(expiry).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("mintTestJWT: %v", err)
	}
	return signed
}

// fakeUserProvisioner satisfies the userProvisioner interface without a DB.
type fakeUserProvisioner struct {
	user *models.User
	err  error
}

func (f *fakeUserProvisioner) GetOrCreate(_ context.Context, supabaseID, email, name string) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user != nil {
		return f.user, nil
	}
	return &models.User{
		ID:         uuid.New(),
		SupabaseID: supabaseID,
		Email:      email,
		Name:       name,
	}, nil
}

func newTestRouter(secret string, prov userProvisioner) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	log := zap.NewNop()
	r.Use(AuthMiddleware(secret, prov, log))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

// ── extractBearer ─────────────────────────────────────────────────────────────

func TestExtractBearer_ValidHeader(t *testing.T) {
	got := extractBearer("Bearer my-token-value")
	want := "my-token-value"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractBearer_MissingPrefix_ReturnsEmpty(t *testing.T) {
	if extractBearer("Token my-token-value") != "" {
		t.Error("non-Bearer prefix must return empty string")
	}
	if extractBearer("my-token-value") != "" {
		t.Error("bare token must return empty string")
	}
}

func TestExtractBearer_EmptyString_ReturnsEmpty(t *testing.T) {
	if extractBearer("") != "" {
		t.Error("empty string must return empty string")
	}
}

// ── extractName ───────────────────────────────────────────────────────────────

func TestExtractName_FromFullName(t *testing.T) {
	meta := map[string]interface{}{"full_name": "Alice Sharma"}
	if got := extractName(meta); got != "Alice Sharma" {
		t.Errorf("want %q, got %q", "Alice Sharma", got)
	}
}

func TestExtractName_FallsBackToName(t *testing.T) {
	meta := map[string]interface{}{"name": "Bob"}
	if got := extractName(meta); got != "Bob" {
		t.Errorf("want %q, got %q", "Bob", got)
	}
}

func TestExtractName_PreferFullNameOverName(t *testing.T) {
	meta := map[string]interface{}{
		"full_name": "Alice Full",
		"name":      "alice_short",
	}
	if got := extractName(meta); got != "Alice Full" {
		t.Errorf("full_name must take priority; want %q, got %q", "Alice Full", got)
	}
}

func TestExtractName_NilMap_ReturnsEmpty(t *testing.T) {
	if got := extractName(nil); got != "" {
		t.Errorf("nil map must return empty string, got %q", got)
	}
}

func TestExtractName_MissingKeys_ReturnsEmpty(t *testing.T) {
	meta := map[string]interface{}{"avatar_url": "https://example.com/pic.png"}
	if got := extractName(meta); got != "" {
		t.Errorf("missing name keys must return empty string, got %q", got)
	}
}

// ── AuthMiddleware ─────────────────────────────────────────────────────────────

func TestAuthMiddleware_MissingToken_Returns401(t *testing.T) {
	r := newTestRouter("secret", &fakeUserProvisioner{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing token: want 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	r := newTestRouter("secret", &fakeUserProvisioner{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: want 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSecret_Returns401(t *testing.T) {
	tokenStr := mintTestJWT(t, "signing-secret", "sub-001", "a@b.com", time.Hour)

	r := newTestRouter("different-secret", &fakeUserProvisioner{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong secret: want 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	tokenStr := mintTestJWT(t, "secret", "sub-001", "a@b.com", -1*time.Hour) // expired 1h ago

	r := newTestRouter("secret", &fakeUserProvisioner{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expired token: want 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_Returns200(t *testing.T) {
	const secret = "my-jwt-secret-32-bytes-long-!!!!"
	tokenStr := mintTestJWT(t, secret, "sub-001", "user@test.com", time.Hour)

	r := newTestRouter(secret, &fakeUserProvisioner{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("valid token: want 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingSub_Returns401(t *testing.T) {
	const secret = "my-jwt-secret"
	// Token without a "sub" claim
	claims := jwt.MapClaims{
		"email": "user@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	r := newTestRouter(secret, &fakeUserProvisioner{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing sub: want 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_UserProvisionerError_Returns500(t *testing.T) {
	const secret = "my-jwt-secret"
	tokenStr := mintTestJWT(t, secret, "sub-001", "user@test.com", time.Hour)

	prov := &fakeUserProvisioner{err: &provisionError{}}
	r := newTestRouter(secret, prov)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("provisioner error: want 500, got %d", w.Code)
	}
}

type provisionError struct{}

func (e *provisionError) Error() string { return "db connection failed" }

// ── UserFromCtx / UserIDFromCtx ───────────────────────────────────────────────

func TestUserFromCtx_ReturnsNilWhenAbsent(t *testing.T) {
	ctx := context.Background()
	if u := UserFromCtx(ctx); u != nil {
		t.Errorf("want nil, got %+v", u)
	}
}

func TestUserIDFromCtx_ReturnsZeroWhenAbsent(t *testing.T) {
	ctx := context.Background()
	id := UserIDFromCtx(ctx)
	if id != (uuid.UUID{}) {
		t.Errorf("want zero UUID, got %v", id)
	}
}
