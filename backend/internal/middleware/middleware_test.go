package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── ErrorHandler ──────────────────────────────────────────────────────────────

func newErrorTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler(zap.NewNop()))
	return r
}

func TestErrorHandler_APIError_UsesCodeAndMessage(t *testing.T) {
	r := newErrorTestRouter()
	r.GET("/x", func(c *gin.Context) {
		_ = c.Error(apierr.NotFound("widget"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "widget not found") {
		t.Errorf("expected api error message in body, got %s", w.Body.String())
	}
}

func TestErrorHandler_UnknownError_Returns500WithoutLeaking(t *testing.T) {
	r := newErrorTestRouter()
	r.GET("/x", func(c *gin.Context) {
		_ = c.Error(http.ErrBodyNotAllowed) // arbitrary non-API error
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), http.ErrBodyNotAllowed.Error()) {
		t.Error("internal error details must not leak to the client")
	}
	if !strings.Contains(w.Body.String(), "unexpected error") {
		t.Errorf("expected generic message, got %s", w.Body.String())
	}
}

func TestErrorHandler_NoError_PassesThrough(t *testing.T) {
	r := newErrorTestRouter()
	r.GET("/x", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestErrorHandler_LastErrorWins(t *testing.T) {
	r := newErrorTestRouter()
	r.GET("/x", func(c *gin.Context) {
		_ = c.Error(apierr.BadRequest("first"))
		_ = c.Error(apierr.Conflict("second"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected last error's 409, got %d", w.Code)
	}
}

// ── RecoveryHandler ───────────────────────────────────────────────────────────

func TestRecoveryHandler_PanicReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RecoveryHandler(zap.NewNop()))
	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "boom") {
		t.Error("panic value must not leak to the client")
	}
}

// ── RequestLogger ─────────────────────────────────────────────────────────────

func TestRequestLogger_SetsRequestIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger(zap.NewNop()))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

// ── CORSMiddleware ────────────────────────────────────────────────────────────

func newCORSRouter(t *testing.T, allowedOrigins string) *gin.Engine {
	t.Helper()
	old := os.Getenv("CORS_ALLOWED_ORIGINS")
	os.Setenv("CORS_ALLOWED_ORIGINS", allowedOrigins)
	t.Cleanup(func() { os.Setenv("CORS_ALLOWED_ORIGINS", old) })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/x", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func TestCORS_AllowedOrigin_GetsACAOHeader(t *testing.T) {
	r := newCORSRouter(t, "https://app.dreamlog.app")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.dreamlog.app")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.dreamlog.app" {
		t.Errorf("expected ACAO for allowed origin, got %q", got)
	}
}

func TestCORS_DisallowedOrigin_NoACAOHeader(t *testing.T) {
	r := newCORSRouter(t, "https://app.dreamlog.app")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin must not receive ACAO, got %q", got)
	}
}

func TestCORS_PreflightOPTIONS_Returns204(t *testing.T) {
	r := newCORSRouter(t, "https://app.dreamlog.app")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://app.dreamlog.app")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Allow-Methods header on preflight")
	}
}

func TestCORS_DefaultsToLocalhostDevOrigins(t *testing.T) {
	r := newCORSRouter(t, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected localhost dev origin allowed by default, got %q", got)
	}
}
