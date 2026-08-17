package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets baseline hardening headers on every response.
//
// This is a JSON API with no server-rendered HTML and no cookie-based
// sessions of its own (bearer JWT only - see AuthMiddleware), so most of the
// browser-facing headers below are cheap, low-risk defaults rather than
// load-bearing controls. HSTS is the one that matters most for a public API:
// it tells any browser/client that ever sees an HTTPS response from this host
// to never downgrade to plain HTTP again, closing off SSL-stripping style
// downgrade attacks against clients that might hit the API from a browser
// context (e.g. the therapist portal, share links).
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Force HTTPS for a year, including subdomains, and allow browser
		// preload lists to hardcode this. Harmless in local dev over HTTP -
		// browsers only honor HSTS on responses actually served over HTTPS.
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Stop browsers from MIME-sniffing responses into an executable type.
		c.Header("X-Content-Type-Options", "nosniff")

		// This API is never meant to be framed - blocks clickjacking of any
		// HTML this host might ever serve (e.g. error pages).
		c.Header("X-Frame-Options", "DENY")

		// No HTML is served, so a strict CSP that allows nothing is always
		// correct here and costs nothing.
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Don't leak this API's URLs (which can carry share tokens/passcodes
		// in the query string) to third parties via the Referer header.
		c.Header("Referrer-Policy", "no-referrer")

		// No reason for this API to be reachable from other origins' feature
		// policies (camera, mic, geolocation, etc.).
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
