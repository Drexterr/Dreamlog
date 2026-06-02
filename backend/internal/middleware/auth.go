package middleware

import (
	"context"
	"net/http"
	"strings"

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
	ctxUserKey    contextKey = "user"
	ctxUserIDKey  contextKey = "user_id"
)

// supabaseClaims maps Supabase JWT fields.
type supabaseClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	// Supabase stores raw_user_meta_data and user_metadata here.
	UserMetadata map[string]interface{} `json:"user_metadata"`
}

// AuthMiddleware validates Supabase JWTs and auto-provisions users.
func AuthMiddleware(jwtSecret string, userSvc userProvisioner, log *zap.Logger) gin.HandlerFunc {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apierr.Unauthorized("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	}

	return func(c *gin.Context) {
		rawToken := extractBearer(c.GetHeader("Authorization"))
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierr.Unauthorized("missing authorization header"))
			return
		}

		claims := &supabaseClaims{}
		token, err := jwt.ParseWithClaims(rawToken, claims, keyFunc,
			jwt.WithValidMethods([]string{"HS256"}),
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

		// Upsert user — idempotent, cheap (single indexed query).
		user, err := userSvc.GetOrCreate(c.Request.Context(), supabaseID, email, name)
		if err != nil {
			log.Error("auth: upsert user failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, apierr.Internal("authentication error"))
			return
		}

		// Store user in context for downstream handlers.
		ctx := context.WithValue(c.Request.Context(), ctxUserKey, user)
		ctx = context.WithValue(ctx, ctxUserIDKey, user.ID)
		c.Request = c.Request.WithContext(ctx)

		// Also store in gin context for convenience.
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
