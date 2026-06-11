package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newES256Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// jwksServer serves the public half of key as a JWKS endpoint.
func jwksServer(t *testing.T, key *ecdsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()
	pad := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	byteLen := (key.Curve.Params().BitSize + 7) / 8
	x := key.PublicKey.X.FillBytes(make([]byte, byteLen))
	y := key.PublicKey.Y.FillBytes(make([]byte, byteLen))

	doc := map[string]interface{}{
		"keys": []map[string]string{
			{"kty": "EC", "crv": "P-256", "alg": "ES256", "kid": kid, "x": pad(x), "y": pad(y)},
		},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	}))
}

func mintES256JWT(t *testing.T, key *ecdsa.PrivateKey, kid, sub string, expiry time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   sub,
		"email": "es256@test.com",
		"exp":   time.Now().Add(expiry).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	if kid != "" {
		token.Header["kid"] = kid
	}
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func newJWKSTestRouter(secret, jwksURL string, prov userProvisioner) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware(secret, jwksURL, prov, zap.NewNop()))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func doPing(r *gin.Engine, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

// ── ES256 / JWKS path ─────────────────────────────────────────────────────────

func TestAuthES256_ValidToken_Accepted(t *testing.T) {
	key := newES256Key(t)
	srv := jwksServer(t, key, "kid-1")
	defer srv.Close()

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	w := doPing(r, mintES256JWT(t, key, "kid-1", "es256-user", time.Hour))
	if w.Code != http.StatusOK {
		t.Fatalf("valid ES256 token must be accepted, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthES256_NoKidHeader_FallsBackToFirstKey(t *testing.T) {
	key := newES256Key(t)
	srv := jwksServer(t, key, "kid-1")
	defer srv.Close()

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	w := doPing(r, mintES256JWT(t, key, "", "es256-user", time.Hour))
	if w.Code != http.StatusOK {
		t.Fatalf("ES256 token without kid must fall back to first JWKS key, got %d", w.Code)
	}
}

func TestAuthES256_WrongKey_Rejected(t *testing.T) {
	served := newES256Key(t)
	other := newES256Key(t)
	srv := jwksServer(t, served, "kid-1")
	defer srv.Close()

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	w := doPing(r, mintES256JWT(t, other, "kid-1", "es256-user", time.Hour))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token signed by a different key must be rejected, got %d", w.Code)
	}
}

func TestAuthES256_ExpiredToken_Rejected(t *testing.T) {
	key := newES256Key(t)
	srv := jwksServer(t, key, "kid-1")
	defer srv.Close()

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	w := doPing(r, mintES256JWT(t, key, "kid-1", "es256-user", -time.Hour))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired ES256 token must be rejected, got %d", w.Code)
	}
}

func TestAuthES256_NoJWKSConfigured_Rejected(t *testing.T) {
	key := newES256Key(t)
	r := newJWKSTestRouter("hs-secret", "", &fakeUserProvisioner{})
	w := doPing(r, mintES256JWT(t, key, "kid-1", "es256-user", time.Hour))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("ES256 token must be rejected when no JWKS URL is configured, got %d", w.Code)
	}
}

func TestAuthES256_JWKSEndpointDown_Rejected(t *testing.T) {
	key := newES256Key(t)
	srv := jwksServer(t, key, "kid-1")
	srv.Close() // immediately unavailable

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	w := doPing(r, mintES256JWT(t, key, "kid-1", "es256-user", time.Hour))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token must be rejected when JWKS endpoint is unreachable, got %d", w.Code)
	}
}

func TestAuthES256_CacheServesSecondRequestWithoutRefetch(t *testing.T) {
	key := newES256Key(t)
	fetches := 0
	pad := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	byteLen := (key.Curve.Params().BitSize + 7) / 8
	x := key.PublicKey.X.FillBytes(make([]byte, byteLen))
	y := key.PublicKey.Y.FillBytes(make([]byte, byteLen))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]string{
				{"kty": "EC", "crv": "P-256", "alg": "ES256", "kid": "kid-1", "x": pad(x), "y": pad(y)},
			},
		})
	}))
	defer srv.Close()

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	tok := mintES256JWT(t, key, "kid-1", "es256-user", time.Hour)
	for i := 0; i < 3; i++ {
		if w := doPing(r, tok); w.Code != http.StatusOK {
			t.Fatalf("request %d failed: %d", i, w.Code)
		}
	}
	if fetches != 1 {
		t.Errorf("JWKS must be fetched once and cached, got %d fetches", fetches)
	}
}

// HS256 must keep working when a JWKS URL is also configured.
func TestAuthHS256_StillWorksWhenJWKSConfigured(t *testing.T) {
	key := newES256Key(t)
	srv := jwksServer(t, key, "kid-1")
	defer srv.Close()

	r := newJWKSTestRouter("hs-secret", srv.URL, &fakeUserProvisioner{})
	w := doPing(r, mintTestJWT(t, "hs-secret", "hs-user", "hs@test.com", time.Hour))
	if w.Code != http.StatusOK {
		t.Fatalf("HS256 token must still be accepted, got %d", w.Code)
	}
}
