package services

import (
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ── mintJWT ───────────────────────────────────────────────────────────────────

func TestMintJWT_ReturnsNonEmptyToken(t *testing.T) {
	svc := &AuthService{jwtSecret: "test-secret-32-bytes-minimum!!!"}
	user := &models.User{SupabaseID: "sub-abc", Email: "user@test.com"}

	token, err := svc.mintJWT(user)
	if err != nil {
		t.Fatalf("mintJWT error: %v", err)
	}
	if token == "" {
		t.Error("token must not be empty")
	}
}

func TestMintJWT_ClaimsAreCorrect(t *testing.T) {
	secret := "test-secret-32-bytes-minimum!!!"
	svc := &AuthService{jwtSecret: secret}
	user := &models.User{
		SupabaseID: "supabase-sub-001",
		Email:      "alice@example.com",
	}

	tokenStr, err := svc.mintJWT(user)
	if err != nil {
		t.Fatal(err)
	}

	// Parse the token back and verify claims
	parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if !parsed.Valid {
		t.Error("minted token must be valid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims must be MapClaims")
	}
	if claims["sub"] != user.SupabaseID {
		t.Errorf("sub: want %q, got %v", user.SupabaseID, claims["sub"])
	}
	if claims["email"] != user.Email {
		t.Errorf("email: want %q, got %v", user.Email, claims["email"])
	}
}

func TestMintJWT_TokenExpiresIn30Days(t *testing.T) {
	secret := "test-secret-32-bytes-minimum!!!"
	svc := &AuthService{jwtSecret: secret}
	user := &models.User{SupabaseID: "sub", Email: "e@e.com"}

	before := time.Now()
	tokenStr, err := svc.mintJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()

	parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	claims := parsed.Claims.(jwt.MapClaims)
	expUnix := int64(claims["exp"].(float64))
	expTime := time.Unix(expUnix, 0)

	// JWT exp is truncated to seconds, so allow 1-second slack before the window.
	minExp := before.Add(30 * 24 * time.Hour).Add(-time.Second)
	maxExp := after.Add(30 * 24 * time.Hour).Add(5 * time.Second)

	if expTime.Before(minExp) || expTime.After(maxExp) {
		t.Errorf("token expiry %v out of expected 30-day window [%v, %v]", expTime, minExp, maxExp)
	}
}

func TestMintJWT_WrongSecretFailsValidation(t *testing.T) {
	svc := &AuthService{jwtSecret: "correct-secret-for-signing!!!!!"}
	user := &models.User{SupabaseID: "sub", Email: "e@e.com"}

	tokenStr, err := svc.mintJWT(user)
	if err != nil {
		t.Fatal(err)
	}

	_, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret-for-validation!!!!"), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err == nil {
		t.Error("JWT signed with one secret must fail validation with a different secret")
	}
}

func TestMintJWT_UsesHS256(t *testing.T) {
	svc := &AuthService{jwtSecret: "test-secret!!"}
	user := &models.User{SupabaseID: "sub", Email: "e@e.com"}

	tokenStr, err := svc.mintJWT(user)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the algorithm claim by parsing back with HS256 whitelist.
	_, err = jwt.Parse(tokenStr, func(tok *jwt.Token) (interface{}, error) {
		return []byte("test-secret!!"), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Errorf("token must be valid HS256: %v", err)
	}
}

// ── bcrypt helpers ────────────────────────────────────────────────────────────

func TestBcrypt_HashIsNotPlaintext(t *testing.T) {
	password := "super-secret-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) == password {
		t.Error("bcrypt hash must not equal plaintext password")
	}
}

func TestBcrypt_DifferentHashEachTime(t *testing.T) {
	password := "same-password"
	hash1, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	hash2, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if string(hash1) == string(hash2) {
		t.Error("bcrypt must produce a different hash each call (random salt)")
	}
}

func TestBcrypt_CorrectPasswordVerifies(t *testing.T) {
	password := "my-secure-password"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		t.Errorf("correct password must verify successfully: %v", err)
	}
}

func TestBcrypt_WrongPasswordFails(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)

	err := bcrypt.CompareHashAndPassword(hash, []byte("wrong-password"))
	if err == nil {
		t.Error("wrong password must fail bcrypt comparison")
	}
}

func TestBcrypt_EmptyPasswordFails(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("actual-password"), bcrypt.DefaultCost)

	err := bcrypt.CompareHashAndPassword(hash, []byte(""))
	if err == nil {
		t.Error("empty password must fail bcrypt comparison")
	}
}
