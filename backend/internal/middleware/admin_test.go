package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// adminRouter injects user (as AuthMiddleware would) and guards /x.
func adminRouter(admins []string, user *models.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			ctx := context.WithValue(c.Request.Context(), ctxUserKey, user)
			ctx = context.WithValue(ctx, ctxUserIDKey, user.ID)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	})
	r.GET("/x", RequireAdmin(admins), func(c *gin.Context) { c.String(http.StatusOK, "admin") })
	return r
}

func TestRequireAdmin(t *testing.T) {
	verified := func(email string) *models.User {
		return &models.User{ID: uuid.New(), Email: email, EmailVerified: true}
	}
	unverified := verified("ops@example.test")
	unverified.EmailVerified = false

	cases := []struct {
		name   string
		admins []string
		user   *models.User
		want   int
	}{
		{"empty list disables admin routes for everyone", nil, verified("ops@example.test"), http.StatusNotFound},
		{"blank entries do not count", []string{" ", ""}, verified(""), http.StatusNotFound},
		{"no authenticated user", []string{"ops@example.test"}, nil, http.StatusNotFound},
		{"email not on the list", []string{"ops@example.test"}, verified("someone@example.test"), http.StatusNotFound},
		{"unverified email on the list (local signup squat)", []string{"ops@example.test"}, unverified, http.StatusNotFound},
		{"verified listed email", []string{"ops@example.test"}, verified("ops@example.test"), http.StatusOK},
		{"case and whitespace insensitive", []string{"  OPS@Example.TEST "}, verified("ops@example.test"), http.StatusOK},
		{"one of several", []string{"a@example.test", "ops@example.test"}, verified("ops@example.test"), http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			// Headers must never grant access.
			req.Header.Set("X-Admin", "true")
			adminRouter(tc.admins, tc.user).ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("want %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
			if tc.want != http.StatusOK && w.Body.String() == "admin" {
				t.Fatal("handler ran for a non-admin")
			}
		})
	}
}
