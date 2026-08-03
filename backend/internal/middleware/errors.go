package middleware

import (
	"net/http"

	"github.com/dreamlog/backend/pkg/apierr"
	sentrygin "github.com/getsentry/sentry-go/gin"
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
			// 5xx means the failure is ours, not the client's - log and
			// report it even though it's a "known" apierr type. Without
			// this, every apierr.Internal(...) call site was invisible in
			// logs and Sentry: only truly unrecognized errors reached the
			// logging below. Client-facing message stays generic; the real
			// cause (when the call site provided one via InternalErr) goes
			// to the log/Sentry side only.
			if apiErr.Code >= http.StatusInternalServerError {
				logErr := error(apiErr)
				if apiErr.Cause != nil {
					logErr = apiErr.Cause
				}
				if hub := sentrygin.GetHubFromContext(c); hub != nil {
					hub.CaptureException(logErr)
				}
				log.Error("internal error",
					zap.Error(logErr),
					zap.String("message", apiErr.Message),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
			}
			c.JSON(apiErr.Code, apiErr)
			return
		}

		// Unknown error - report to Sentry (no-op when uninitialized), do not
		// leak internals to the client.
		if hub := sentrygin.GetHubFromContext(c); hub != nil {
			hub.CaptureException(err)
		}
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
