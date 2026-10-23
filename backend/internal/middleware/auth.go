package middleware

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// userProvisioner is satisfied by *services.UserService. Defined here so
// AuthMiddleware can be unit-tested without a real database.
type userProvisioner interface {
	GetOrCreate(ctx context.Context, supabaseID, email, name string, emailVerified bool) (*models.User, error)
}

type contextKey string

const (
	ctxUserKey   contextKey = "user"
	ctxUserIDKey contextKey = "user_id"
)

// supabaseClaims maps Supabase JWT fields.
type supabaseClaims struct {
	jwt.RegisteredClaims
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
}

// jwksKey is a single key from a JWKS endpoint.
type jwksKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Crv string `json:"crv"`
	Alg string `json:"alg"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// jwksCache fetches and caches EC public keys from a JWKS endpoint.
//
// Network fetches never happen under the lock, use a bounded HTTP timeout, and
// a refresh for an unknown kid is rate-limited (jwksMinRefresh), so a flood of
// tokens with random kids can neither stall every authenticated request nor
// make the API hammer Supabase.
type jwksCache struct {
	mu          sync.RWMutex
	byKid       map[string]*ecdsa.PublicKey
	all         []*ecdsa.PublicKey
	fetchedAt   time.Time
	lastAttempt time.Time
	ttl         time.Duration
	endpoint    string
	refreshMu   sync.Mutex // serializes fetches (one in flight at a time)
}

const jwksMinRefresh = time.Minute

var jwksHTTPClient = &http.Client{Timeout: 5 * time.Second}

func newJWKSCache(endpoint string) *jwksCache {
	return &jwksCache{
		endpoint: endpoint,
		ttl:      time.Hour,
		byKid:    make(map[string]*ecdsa.PublicKey),
	}
}

// lookup returns the key for kid from the current cache (kid "" = first key).
func (c *jwksCache) lookup(kid string) (*ecdsa.PublicKey, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fresh := time.Since(c.fetchedAt) < c.ttl && len(c.all) > 0
	if kid == "" {
		if len(c.all) > 0 {
			return c.all[0], true, fresh
		}
		return nil, false, fresh
	}
	k, ok := c.byKid[kid]
	return k, ok, fresh
}

func (c *jwksCache) get(kid string) (*ecdsa.PublicKey, error) {
	if k, ok, fresh := c.lookup(kid); ok && fresh {
		return k, nil
	}

	// Miss or stale: refresh, at most one fetch in flight and at most one per
	// jwksMinRefresh. Concurrent callers wait for the in-flight fetch.
	c.refreshMu.Lock()
	if k, ok, fresh := c.lookup(kid); ok && fresh {
		c.refreshMu.Unlock()
		return k, nil
	}
	c.mu.RLock()
	throttled := time.Since(c.lastAttempt) < jwksMinRefresh
	c.mu.RUnlock()
	if !throttled {
		byKid, all, err := fetchJWKS(c.endpoint)
		c.mu.Lock()
		c.lastAttempt = time.Now()
		if err == nil {
			c.byKid, c.all, c.fetchedAt = byKid, all, time.Now()
		}
		c.mu.Unlock()
	}
	c.refreshMu.Unlock()

	// Stale keys are still served when a refresh fails, but an unknown kid is
	// never mapped to some other key.
	if k, ok, _ := c.lookup(kid); ok {
		return k, nil
	}
	return nil, fmt.Errorf("no JWKS key matches kid %q", kid)
}

func fetchJWKS(endpoint string) (map[string]*ecdsa.PublicKey, []*ecdsa.PublicKey, error) {
	resp, err := jwksHTTPClient.Get(endpoint) //nolint:noctx
	if err != nil {
		return nil, nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, nil, fmt.Errorf("decode jwks: %w", err)
	}

	byKid := make(map[string]*ecdsa.PublicKey)
	var all []*ecdsa.PublicKey

	for _, k := range jwks.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			continue
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			continue
		}
		pub := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}
		all = append(all, pub)
		if k.Kid != "" {
			byKid[k.Kid] = pub
		}
	}

	if len(all) == 0 {
		return nil, nil, fmt.Errorf("no EC P-256 keys found at JWKS endpoint")
	}
	return byKid, all, nil
}

// AuthMiddleware validates JWTs and auto-provisions users.
// Supports HS256 (local /auth/register+login path) and ES256 (Supabase JWKS).
// jwksURL may be empty - in that case only HS256 is accepted.
func AuthMiddleware(jwtSecret, jwksURL string, userSvc userProvisioner, log *zap.Logger) gin.HandlerFunc {
	hsKey := []byte(jwtSecret)

	var cache *jwksCache
	// Supabase issues ES256 tokens with iss = <SUPABASE_URL>/auth/v1, which is
	// the JWKS URL minus its well-known suffix.
	esIssuer := ""
	if jwksURL != "" {
		cache = newJWKSCache(jwksURL)
		if strings.HasSuffix(jwksURL, "/.well-known/jwks.json") {
			esIssuer = strings.TrimSuffix(jwksURL, "/.well-known/jwks.json")
		}
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		switch token.Method.Alg() {
		case "HS256":
			return hsKey, nil
		case "ES256":
			if cache == nil {
				return nil, apierr.Unauthorized("ES256 tokens not supported (SUPABASE_URL not configured)")
			}
			kid, _ := token.Header["kid"].(string)
			return cache.get(kid)
		default:
			return nil, apierr.Unauthorized("unexpected signing method: " + token.Method.Alg())
		}
	}

	return func(c *gin.Context) {
		rawToken := extractBearer(c.GetHeader("Authorization"))
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierr.Unauthorized("missing authorization header"))
			return
		}

		claims := &supabaseClaims{}
		token, err := jwt.ParseWithClaims(rawToken, claims, keyFunc,
			jwt.WithValidMethods([]string{"HS256", "ES256"}),
			// Tokens without exp would otherwise be valid forever, and any token
			// signed with the shared secret for another audience would pass.
			jwt.WithExpirationRequired(),
			jwt.WithAudience("authenticated"),
		)
		if err == nil && token.Method.Alg() == "ES256" && esIssuer != "" && claims.Issuer != esIssuer {
			err = fmt.Errorf("unexpected issuer %q", claims.Issuer)
		}
		if err != nil || !token.Valid {
			log.Warn("invalid jwt", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierr.Unauthorized("invalid or expired token"))
			return
		}

		supabaseID := claims.Subject
		if supabaseID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierr.Unauthorized("token missing sub claim"))
			return
		}

		email := claims.Email
		name := extractName(claims.UserMetadata)

		user, err := userSvc.GetOrCreate(c.Request.Context(), supabaseID, email, name, emailVerifiedClaim(supabaseID, claims.UserMetadata))
		if err != nil {
			log.Error("auth: upsert user failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, apierr.Internal("authentication error"))
			return
		}

		ctx := context.WithValue(c.Request.Context(), ctxUserKey, user)
		ctx = context.WithValue(ctx, ctxUserIDKey, user.ID)
		c.Request = c.Request.WithContext(ctx)

		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// emailVerifiedClaim reports whether the token's email was verified (N6).
// Tokens minted by the local /auth/login path (sub "local-...") never are:
// /auth/register accepts any address. Supabase only issues sessions for
// confirmed emails or OAuth identities, so its tokens are verified unless
// user_metadata explicitly says email_verified=false.
func emailVerifiedClaim(supabaseID string, meta map[string]interface{}) bool {
	if strings.HasPrefix(supabaseID, "local-") {
		return false
	}
	if v, ok := meta["email_verified"].(bool); ok {
		return v
	}
	return true
}

// UserFromCtx retrieves the authenticated User from a request context.
func UserFromCtx(ctx context.Context) *models.User {
	u, _ := ctx.Value(ctxUserKey).(*models.User)
	return u
}

// UserIDFromCtx retrieves the authenticated user's internal UUID.
func UserIDFromCtx(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxUserIDKey).(uuid.UUID)
	return id
}

func extractBearer(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func extractName(meta map[string]interface{}) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta["full_name"].(string); ok {
		return v
	}
	if v, ok := meta["name"].(string); ok {
		return v
	}
	return ""
}
