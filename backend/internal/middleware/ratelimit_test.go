package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_AllowsBurstThenBlocks(t *testing.T) {
	rl := NewRateLimiter(0, 3) // no refill, burst of 3
	now := time.Now()

	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4", now) {
			t.Fatalf("request %d within burst should be allowed", i+1)
		}
	}
	if rl.allow("1.2.3.4", now) {
		t.Error("4th request beyond burst should be blocked")
	}
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(1, 1) // 1 token/sec, burst 1
	now := time.Now()

	if !rl.allow("5.6.7.8", now) {
		t.Fatal("first request should be allowed")
	}
	if rl.allow("5.6.7.8", now) {
		t.Fatal("immediate second request should be blocked")
	}
	// One second later, a token has refilled.
	if !rl.allow("5.6.7.8", now.Add(time.Second)) {
		t.Error("request after 1s refill should be allowed")
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	rl := NewRateLimiter(0, 1)
	now := time.Now()

	if !rl.allow("10.0.0.1", now) {
		t.Fatal("first IP should be allowed")
	}
	if !rl.allow("10.0.0.2", now) {
		t.Error("a different IP should have its own bucket")
	}
}

func TestRateLimiter_Middleware_Returns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(0, 1)
	r := gin.New()
	r.GET("/x", rl.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request: want 200, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	r.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("second request: want 429, got %d", second.Code)
	}
}
