package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RateLimiter is a lightweight per-IP token-bucket limiter for abuse-sensitive
// endpoints (login, registration, public share lookup). It is in-memory and
// therefore per-process: with several API replicas the effective limit is
// (limit × replicas). That still raises the bar on brute-force enormously; the
// share-link lockout in the database (failed_attempts / locked_until) is the
// cross-instance backstop for the passcode path.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens added per second
	burst    float64 // max tokens (also the initial allowance)
	lastSeen map[string]time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter builds a limiter allowing `burst` requests immediately and
// refilling at `rate` requests per second per client IP. It starts a background
// sweeper that evicts idle IPs so the map cannot grow unbounded.
func NewRateLimiter(rate, burst float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		burst:    burst,
		lastSeen: make(map[string]time.Time),
	}
	go rl.sweep()
	return rl
}

// allow reports whether a request from ip may proceed, consuming one token.
func (rl *RateLimiter) allow(ip string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.lastSeen[ip] = now
	b, ok := rl.buckets[ip]
	if !ok {
		rl.buckets[ip] = &bucket{tokens: rl.burst - 1, last: now}
		return true
	}

	// Refill based on elapsed time, capped at burst.
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep periodically removes IP entries not seen in the last 10 minutes.
func (rl *RateLimiter) sweep() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		rl.mu.Lock()
		for ip, seen := range rl.lastSeen {
			if seen.Before(cutoff) {
				delete(rl.buckets, ip)
				delete(rl.lastSeen, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns a gin handler that enforces the limit per client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP(), time.Now()) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				apierr.New(http.StatusTooManyRequests, "too many requests, please slow down"))
			return
		}
		c.Next()
	}
}
