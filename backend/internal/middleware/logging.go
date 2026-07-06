package middleware

import (
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// sensitiveQueryParams are redacted before a request's query string is logged.
// The share passcode ("p") in particular grants access to clinical data and
// must never be written to logs (which may be shipped to third-party sinks).
var sensitiveQueryParams = map[string]bool{
	"p":        true, // share-link passcode (GET /share/:token?p=1234)
	"passcode": true,
	"token":    true,
	"password": true,
}

// RequestLogger logs structured request/response data using zap.
func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()

		log.Info("request",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", redactQuery(c.Request.URL.Query())),
			zap.Int("status", c.Writer.Status()),
			zap.Int("bytes", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

// redactQuery rebuilds a query string with sensitive parameter values replaced
// by "[REDACTED]", preserving the key so logs still show which params were sent.
func redactQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	out := make(url.Values, len(values))
	for key, vals := range values {
		if sensitiveQueryParams[key] {
			out.Set(key, "[REDACTED]")
			continue
		}
		out[key] = vals
	}
	return out.Encode()
}
