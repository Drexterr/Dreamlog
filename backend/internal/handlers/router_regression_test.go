package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRouterPublicContractsAndAuthBoundaries(t *testing.T) {
	handler := NewRouter(Deps{JWTSecret: "local-router-unit-only", Log: zap.NewNop(), MinimumAppVersion: "1.2.3", AndroidStoreURL: "https://example.invalid/android"})
	engine := handler.(*gin.Engine)
	for _, path := range []string{"/health", "/version", "/journeys"} {
		t.Run("public"+path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if path == "/health" && body["status"] != "ok" {
				t.Fatalf("health: %v", body)
			}
			if path == "/version" && (body["minimum_version"] != "1.2.3" || body["android_store_url"] != "https://example.invalid/android") {
				t.Fatalf("version: %v", body)
			}
			if path == "/journeys" {
				catalog, ok := body["journeys"].([]any)
				if !ok || len(catalog) == 0 {
					t.Fatalf("catalogue: %v", body)
				}
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("security headers absent: %v", response.Header())
			}
		})
	}
	protected := 0
	for _, route := range engine.Routes() {
		if route.Path == "/health" || route.Path == "/version" || route.Path == "/auth/login" || route.Path == "/auth/register" || route.Method == http.MethodGet && (route.Path == "/journeys" || route.Path == "/share/:token") {
			continue
		}
		protected++
		path := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(route.Path, ":id", "00000000-0000-0000-0000-000000000001"), ":clientID", "00000000-0000-0000-0000-000000000001"), ":therapistID", "00000000-0000-0000-0000-000000000001"), ":sessionID", "00000000-0000-0000-0000-000000000001"), ":journeyID", "stress_relief")
		path = strings.ReplaceAll(path, ":slug", "qa-company")
		t.Run(route.Method+route.Path, func(t *testing.T) {
			for _, authorization := range []string{"", "Bearer malformed"} {
				request := httptest.NewRequest(route.Method, path, strings.NewReader(`{}`))
				request.Header.Set("Content-Type", "application/json")
				if authorization != "" {
					request.Header.Set("Authorization", authorization)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusUnauthorized {
					t.Fatalf("auth %q returned %d: %s", authorization, response.Code, response.Body.String())
				}
				if strings.Contains(response.Body.String(), "panic") {
					t.Fatal("auth did not short-circuit missing dependencies")
				}
			}
		})
	}
	if protected < 50 {
		t.Fatalf("unexpected protected route count: %d", protected)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/upload", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("upload proxy should be disabled: %d", response.Code)
	}
}

func TestRouterCredentialRateLimitAndProxyRegistration(t *testing.T) {
	handler := NewRouter(Deps{JWTSecret: "local-router-unit-only", Log: zap.NewNop(), StorageProxyBaseURL: "http://127.0.0.1:1"})
	found := false
	for _, route := range handler.(*gin.Engine).Routes() {
		if route.Method == http.MethodPut && route.Path == "/upload" {
			found = true
		}
	}
	if !found {
		t.Fatal("local upload proxy was not registered")
	}
	for i := 0; i < 6; i++ {
		request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "127.0.0.1:4567"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		expected := http.StatusBadRequest
		if i == 5 {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("attempt %d: %d %s", i, response.Code, response.Body.String())
		}
		if i == 5 && response.Header().Get("Retry-After") == "" {
			t.Fatal("rate limit missing Retry-After")
		}
	}
}
