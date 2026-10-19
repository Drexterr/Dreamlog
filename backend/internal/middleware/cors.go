package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware restricts cross-origin requests to an explicit allowlist
// (config: CORS_ALLOWED_ORIGINS, comma-separated; defaults to localhost dev
// origins). Config is passed in, never read from the environment here (ADR-010).
func CORSMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	for _, o := range origins {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			allowed[trimmed] = true
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
