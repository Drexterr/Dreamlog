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
	GetOrCreate(ctx context.Context, supabaseID, email, name string) (*models.User, error)
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
type jwksCache struct {
	mu        sync.RWMutex
	byKid     map[string]*ecdsa.PublicKey
	all       []*ecdsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
	endpoint  string
}

func newJWKSCache(endpoint string) *jwksCache {
	return &jwksCache{
		endpoint: endpoint,
		ttl:      time.Hour,
		byKid:    make(map[string]*ecdsa.PublicKey),
	}
}

func (c *jwksCache) get(kid string) (*ecdsa.PublicKey, error) {
	c.mu.RLock()
	fresh := time.Since(c.fetchedAt) < c.ttl && len(c.all) > 0
	if fresh {
		if kid != "" {
			if k, ok := c.byKid[kid]; ok {
				c.mu.RUnlock()
				return k, nil
			}
		} else if len(c.all) > 0 {
			k := c.all[0]
			c.mu.RUnlock()
			return k, nil
		}
	}
	c.mu.RUnlock()

	// Refresh.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if time.Since(c.fetchedAt) < c.ttl && len(c.all) > 0 {
		if kid != "" {
			if k, ok := c.byKid[kid]; ok {
				return k, nil
			}
		} else if len(c.all) > 0 {
			return c.all[0], nil
		}
	}

	byKid, all, err := fetchJWKS(c.endpoint)
	if err != nil {
		// Return stale keys if we have them rather than failing hard.
		if len(c.all) > 0 {
			if kid != "" {
				if k, ok := c.byKid[kid]; ok {
					return k, nil
				}
				return c.all[0], nil
			}
			return c.all[0], nil
		}
		return nil, err
	}

	c.byKid = byKid
	c.all = all
	c.fetchedAt = time.Now()

	if kid != "" {
		if k, ok := c.byKid[kid]; ok {
			return k, nil
		}
	}
	if len(c.all) > 0 {
		return c.all[0], nil
	}
	return nil, fmt.Errorf("no matching key found in JWKS")
}

func fetchJWKS(endpoint string) (map[string]*ecdsa.PublicKey, []*ecdsa.PublicKey, error) {
	resp, err := http.Get(endpoint) //nolint:noctx
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
	if jwksURL != "" {
		cache = newJWKSCache(jwksURL)
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
		)
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

		user, err := userSvc.GetOrCreate(c.Request.Context(), supabaseID, email, name)
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
