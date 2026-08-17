package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodyBytes bounds every JSON request this API accepts. Journal
// audio and therapist note photos never reach this API in the first place -
// per ADR-008, the mobile app and portal upload those directly to S3-compatible
// storage via pre-signed URLs - so every route behind this middleware only
// ever needs to parse small JSON control-plane payloads. 2MB is generous
// headroom over the largest legitimate body (e.g. a batch of session note
// bullets) while still bounding how much memory a single request can force
// the server to allocate before any field-level validation runs.
const MaxRequestBodyBytes = 2 << 20 // 2 MiB

// devUploadProxyPath is the one exception to ADR-008: a dev-only route
// (only registered when STORAGE_PROXY_BASE_URL is set - see router.go) that
// streams raw audio bytes from the mobile client straight through this API
// to MinIO, for local dev setups where the device can't reach MinIO
// directly. A 30-minute AAC recording is well over MaxRequestBodyBytes, so
// this path must be exempt from the global body cap.
const devUploadProxyPath = "/upload"

// MaxBodySize rejects request bodies larger than limitBytes. It wraps the
// request body in an http.MaxBytesReader, which fails the read (and any
// downstream c.ShouldBindJSON) once the limit is exceeded, rather than
// buffering an arbitrarily large body into memory first.
func MaxBodySize(limitBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == devUploadProxyPath {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limitBytes)
		c.Next()
	}
}
