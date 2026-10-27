package handlers

import (
	"context"
	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type waitlistTestStore struct{ calls int }

func (s *waitlistTestStore) Join(context.Context, string) error { s.calls++; return nil }

func TestWaitlistHTTP(t *testing.T) {
	store := &waitlistTestStore{}
	router := gin.New()
	router.Use(middleware.ErrorHandler(zap.NewNop()))
	router.POST("/waitlist", NewWaitlistHandler(services.NewWaitlistService(store)).Join)
	for _, tc := range []struct {
		body string
		code int
	}{
		{`{"email":"you@example.com"}`, 200},
		{`{"email":"you@example.com"}`, 200},
		{`{"email":"invalid"}`, 400},
		{`{`, 400},
		{`{"email":"you@example.com","website":"bot"}`, 200},
		{`{"email":"` + strings.Repeat("a", 3000) + `"}`, 400},
	} {
		req := httptest.NewRequest(http.MethodPost, "/waitlist", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("want %d got %d: %s", tc.code, w.Code, w.Body)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("must not cache")
		}
		if strings.Contains(w.Body.String(), "you@example.com") {
			t.Fatal("email exposed")
		}
	}
	if store.calls != 2 {
		t.Fatalf("unexpected writes: %d", store.calls)
	}
}

func TestWaitlistPublicRateLimit(t *testing.T) {
	router := NewRouter(Deps{WaitlistSvc: services.NewWaitlistService(&waitlistTestStore{}), Log: zap.NewNop()})
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("POST", "/waitlist", strings.NewReader(`{"email":"you@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		want := 200
		if i == 3 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
}
