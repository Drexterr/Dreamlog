package middleware

import (
	"net/http"
	"strings"

	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RequireAdmin guards operator endpoints. It must run after AuthMiddleware.
// A caller qualifies only when the authenticated account's email is on
// adminEmails (ADMIN_EMAILS) AND that email is verified by the identity
// provider - anyone can register an arbitrary address through local signup
// (email_verified = false), so an unverified match is worthless (N6). Nothing
// else (headers, claims, query params) is trusted.
//
// Everyone else - including every caller when the list is empty - gets 404,
// so the routes' existence is not revealed.
func RequireAdmin(adminEmails []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(adminEmails))
	for _, e := range adminEmails {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			allowed[e] = true
		}
	}
	return func(c *gin.Context) {
		user := UserFromCtx(c.Request.Context())
		if len(allowed) == 0 || user == nil || !user.EmailVerified ||
			!allowed[strings.ToLower(strings.TrimSpace(user.Email))] {
			c.AbortWithStatusJSON(http.StatusNotFound, apierr.NotFound("route"))
			return
		}
		c.Next()
	}
}
