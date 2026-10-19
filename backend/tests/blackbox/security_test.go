//go:build integration

package blackbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func TestBlackboxSecurity(t *testing.T) {
	h := newAPI(t)
	h.request(t, "public.health", "GET", "/health", "", "", 200)
	h.request(t, "public.version", "GET", "/version", "", "", 200)
	h.request(t, "public.catalogue", "GET", "/journeys", "", "", 200)
	h.request(t, "public.unknown-route", "GET", "/no-such-route", "", "", 404)
	h.request(t, "public.upload-proxy-disabled", "PUT", "/upload?key=audio/test", "qa", "", 404)
	me := h.raw(t, "GET", "/me", "", h.owner, 200)
	email := me["email"].(string)
	login := h.request(t, "auth.login-valid-local", "POST", "/auth/login", fmt.Sprintf(`{"email":%q,"password":"QA-local-12345"}`, email), "", 200)
	if login["token"] == "" {
		t.Fatal("missing token")
	}
	h.request(t, "auth.login-wrong-password", "POST", "/auth/login", fmt.Sprintf(`{"email":%q,"password":"wrong"}`, email), "", 401)
	h.request(t, "auth.login-missing-fields", "POST", "/auth/login", `{}`, "", 400)
	h = newAPI(t)
	email = h.raw(t, "GET", "/me", "", h.owner, 200)["email"].(string)
	h.request(t, "auth.register-duplicate-email", "POST", "/auth/register", fmt.Sprintf(`{"email":%q,"name":"QA","password":"QA-local-12345"}`, email), "", 409)
	h.request(t, "auth.register-invalid-email", "POST", "/auth/register", `{"email":"bad","name":"QA","password":"QA-local-12345"}`, "", 400)
	h.request(t, "auth.register-short-password", "POST", "/auth/register", `{"email":"new@example.test","name":"QA","password":"short"}`, "", 400)
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "qa", "exp": time.Now().Add(-time.Hour).Unix()}).SignedString([]byte("qa-blackbox-local-only"))
	t.Run("security.protected-route-authentication", func(t *testing.T) {
		// Check every protected route as one boundary scenario, rather than counting repeated JWT checks as feature coverage.
		routes := handlers.NewRouter(handlers.Deps{JWTSecret: "qa-blackbox-local-only", Log: zap.NewNop()}).(*gin.Engine).Routes()
		checked := 0
		for _, route := range routes {
			if route.Path == "/health" || route.Path == "/version" || route.Path == "/journeys" || strings.HasPrefix(route.Path, "/auth/") || route.Method == "GET" && route.Path == "/share/:token" {
				continue
			}
			path := route.Path
			for _, param := range []string{":id", ":clientID", ":therapistID", ":sessionID", ":journeyID"} {
				path = strings.ReplaceAll(path, param, "00000000-0000-0000-0000-000000000001")
			}
			path = strings.ReplaceAll(path, ":slug", "qa")
			for _, token := range []string{"", "malformed", expired} {
				h.raw(t, route.Method, path, `{}`, token, 401)
			}
			checked++
		}
		if checked < 70 {
			t.Fatalf("route inventory unexpectedly small: %d", checked)
		}
		t.Logf("Protected routes checked: %d (missing, malformed and expired bearer)", checked)
	})
	t.Run("security.response-headers", func(t *testing.T) {
		res, err := http.Get(h.server.URL + "/health")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("X-Frame-Options") == "" {
			t.Fatal("security headers absent")
		}
		var body map[string]any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil || body["status"] != "ok" {
			t.Fatal("health schema")
		}
	})
	t.Run("security.cors-preflight", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", h.server.URL+"/me", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		req.Header.Set("Access-Control-Request-Method", "GET")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		io.Copy(io.Discard, res.Body)
		if res.StatusCode != 204 {
			t.Fatalf("preflight %d", res.StatusCode)
		}
	})
}
