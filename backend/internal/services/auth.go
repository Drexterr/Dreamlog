package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ErrEmailTaken is returned when an email address is already registered.
var ErrEmailTaken = errors.New("email already registered")

// ErrAccountLocked is returned when a local-auth account is temporarily
// locked out after too many consecutive failed login attempts.
var ErrAccountLocked = errors.New("account temporarily locked due to too many failed login attempts")

// Login lockout policy: after this many consecutive wrong passwords, a local
// auth account is temporarily locked - mirrors the share-link passcode
// lockout in ShareHandler (sharePasscodeMaxAttempts / sharePasscodeLockWindow).
// This bounds brute-forcing a password for the dev-only local auth path;
// production auth (Supabase) has its own protections outside this codebase.
const (
	loginLockoutMaxAttempts = 5
	loginLockoutWindow      = 15 * time.Minute
)

type AuthService struct {
	users     UserStore
	jwtSecret string
	log       *zap.Logger
}

func NewAuthService(users UserStore, jwtSecret string, log *zap.Logger) *AuthService {
	if log == nil {
		log = zap.NewNop()
	}
	return &AuthService{users: users, jwtSecret: jwtSecret, log: log}
}

// Register creates a new local user and returns the user + signed JWT.
//
// A soft-deleted account (legacy rows from before hard delete) is NEVER
// reactivated: registering only proves knowledge of an email address, not
// ownership, so reviving the old row would hand its journal history to whoever
// registers first. The leftover row and all its data are purged and a fresh
// account is created.
func (s *AuthService) Register(ctx context.Context, email, name, password string) (*models.User, string, error) {
	existing, err := s.users.GetByEmailIncDeleted(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if existing != nil && !existing.IsDeleted {
		return nil, "", ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	if existing != nil && existing.IsDeleted {
		if err := s.users.PurgeDeleted(ctx, existing.ID); err != nil {
			return nil, "", err
		}
	}
	user, err := s.users.CreateLocal(ctx, email, name, string(hash))
	if err != nil {
		return nil, "", err
	}

	token, err := s.mintJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login verifies credentials and returns the user + signed JWT.
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	// Checked before touching credentials at all: unknown emails never have a
	// lock (RecordFailedLogin only ever writes to a matching row), so this
	// branch alone cannot be used to enumerate accounts.
	lockedUntil, err := s.users.GetLoginLockedUntil(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if lockedUntil != nil && time.Now().Before(*lockedUntil) {
		s.log.Warn("security: login rejected - account locked", zap.String("email", email), zap.Time("locked_until", *lockedUntil))
		return nil, "", ErrAccountLocked
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}

	hash, err := s.users.GetPasswordHash(ctx, email)
	if err != nil {
		return nil, "", err
	}

	if user == nil || hash == "" {
		// Spend the same bcrypt time as a real check so response latency
		// doesn't reveal whether the email has an account.
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(password))
	}
	if user == nil || hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		// Only a real account accumulates a lockout counter - an unknown
		// email is a no-op write (WHERE matches no row), so this can't be
		// used to enumerate whether an email is registered.
		if user != nil {
			s.log.Warn("security: failed login attempt", zap.String("email", email))
			if newLockedUntil, recErr := s.users.RecordFailedLogin(ctx, email, loginLockoutMaxAttempts, loginLockoutWindow); recErr == nil && newLockedUntil != nil && time.Now().Before(*newLockedUntil) {
				s.log.Warn("security: account locked after repeated failed logins", zap.String("email", email), zap.Time("locked_until", *newLockedUntil))
				return nil, "", ErrAccountLocked
			}
		}
		return nil, "", errors.New("invalid email or password")
	}

	// Correct password - clear any accumulated failed attempts.
	_ = s.users.ResetLoginAttempts(ctx, email)

	token, err := s.mintJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) mintJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.SupabaseID,
		"aud":   "authenticated", // same audience as Supabase tokens; required by the auth middleware
		"email": user.Email,
		// Short-lived: the therapist portal keeps this in a JS-readable cookie
		// (static export, no HttpOnly), so an XSS-stolen token must expire fast.
		"exp":   time.Now().Add(localTokenTTL).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// dummyPasswordHash is a bcrypt hash (DefaultCost) of a random value, used to
// equalize login timing for unknown emails.
func dummyPasswordHash() []byte {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	})
	return dummyHash
}

// localTokenTTL bounds tokens minted by the local-auth path (therapist portal).
const localTokenTTL = 24 * time.Hour
