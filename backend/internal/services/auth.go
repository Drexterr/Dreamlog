package services

import (
	"context"
	"errors"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrEmailTaken is returned when an email address is already registered.
var ErrEmailTaken = errors.New("email already registered")

type AuthService struct {
	users     UserStore
	jwtSecret string
}

func NewAuthService(users UserStore, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

// Register creates a new local user and returns the user + signed JWT.
func (s *AuthService) Register(ctx context.Context, email, name, password string) (*models.User, string, error) {
	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if existing != nil {
		return nil, "", ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
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
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}

	hash, err := s.users.GetPasswordHash(ctx, email)
	if err != nil {
		return nil, "", err
	}

	if user == nil || hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, "", errors.New("invalid email or password")
	}

	token, err := s.mintJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) mintJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.SupabaseID,
		"email": user.Email,
		"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
