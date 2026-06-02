package middleware

import (
	"net/http"

	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler is a final gin middleware that translates errors set via c.Error()
// into consistent JSON responses.
func ErrorHandler(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Use the last error set.
		err := c.Errors.Last().Err

		if apiErr, ok := apierr.As(err); ok {
			c.JSON(apiErr.Code, apiErr)
			return
		}

		// Unknown error — do not leak internals.
		log.Error("unhandled error", zap.Error(err),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
		c.JSON(http.StatusInternalServerError, apierr.Internal("an unexpected error occurred"))
	}
}

// RecoveryHandler catches panics and returns a 500 response.
func RecoveryHandler(log *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Error("panic recovered",
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierr.Internal("internal server error"))
	})
}
