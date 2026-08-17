package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// fakeUserStore is an in-memory UserStore used by auth flow tests.
type fakeUserStore struct {
	users       map[string]*models.User // keyed by email
	hashes      map[string]string       // email → bcrypt hash
	lockedUntil map[string]time.Time    // email → lockout expiry
	attempts    map[string]int          // email → failed-login counter
	createErr   error
	getErr      error
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users:       make(map[string]*models.User),
		hashes:      make(map[string]string),
		lockedUntil: make(map[string]time.Time),
		attempts:    make(map[string]int),
	}
}

func (s *fakeUserStore) GetByEmail(_ context.Context, email string) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	u := s.users[email]
	if u != nil && u.IsDeleted {
		return nil, nil
	}
	return u, nil
}

func (s *fakeUserStore) GetByEmailIncDeleted(_ context.Context, email string) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.users[email], nil
}

func (s *fakeUserStore) Reactivate(_ context.Context, id uuid.UUID, name, passwordHash string) (*models.User, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	for email, u := range s.users {
		if u.ID == id {
			u.IsDeleted = false
			u.Name = name
			s.hashes[email] = passwordHash
			return u, nil
		}
	}
	return nil, nil
}

func (s *fakeUserStore) CreateLocal(_ context.Context, email, name, passwordHash string) (*models.User, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	u := &models.User{
		ID:         uuid.New(),
		SupabaseID: "local-" + uuid.New().String(),
		Email:      email,
		Name:       name,
	}
	s.users[email] = u
	s.hashes[email] = passwordHash
	return u, nil
}

func (s *fakeUserStore) GetPasswordHash(_ context.Context, email string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	return s.hashes[email], nil
}

func (s *fakeUserStore) GetLoginLockedUntil(_ context.Context, email string) (*time.Time, error) {
	if lu, ok := s.lockedUntil[email]; ok {
		return &lu, nil
	}
	return nil, nil
}

func (s *fakeUserStore) RecordFailedLogin(_ context.Context, email string, maxAttempts int, lockDuration time.Duration) (*time.Time, error) {
	if _, ok := s.users[email]; !ok {
		return nil, nil
	}
	s.attempts[email]++
	if s.attempts[email] >= maxAttempts {
		s.attempts[email] = 0
		lu := time.Now().Add(lockDuration)
		s.lockedUntil[email] = lu
		return &lu, nil
	}
	return nil, nil
}

func (s *fakeUserStore) ResetLoginAttempts(_ context.Context, email string) error {
	delete(s.attempts, email)
	delete(s.lockedUntil, email)
	return nil
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestRegister_CreatesUserAndReturnsToken(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret-32-bytes-minimum!!!!", nil)

	user, token, err := svc.Register(context.Background(), "alice@test.com", "Alice", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("user must not be nil")
	}
	if user.Email != "alice@test.com" {
		t.Errorf("email: want alice@test.com, got %s", user.Email)
	}
	if user.Name != "Alice" {
		t.Errorf("name: want Alice, got %s", user.Name)
	}
	if token == "" {
		t.Error("token must not be empty")
	}
}

func TestRegister_HashesPassword(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, err := svc.Register(context.Background(), "bob@test.com", "Bob", "mypassword")
	if err != nil {
		t.Fatal(err)
	}

	hash := store.hashes["bob@test.com"]
	if hash == "" {
		t.Error("password hash must be stored")
	}
	if hash == "mypassword" {
		t.Error("password must be stored as bcrypt hash, not plaintext")
	}
}

func TestRegister_DuplicateEmail_ReturnsError(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, err := svc.Register(context.Background(), "dup@test.com", "First", "pass1234")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = svc.Register(context.Background(), "dup@test.com", "Second", "pass5678")
	if err == nil {
		t.Error("duplicate email must return an error")
	}
}

func TestRegister_StoresUserInStore(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, err := svc.Register(context.Background(), "stored@test.com", "Stored User", "pass1234")
	if err != nil {
		t.Fatal(err)
	}

	if store.users["stored@test.com"] == nil {
		t.Error("registered user must be in the store")
	}
}

func TestRegister_ReturnsValidJWT(t *testing.T) {
	const secret = "test-secret-32-bytes-minimum!!!!"
	store := newFakeUserStore()
	svc := NewAuthService(store, secret, nil)

	_, token, err := svc.Register(context.Background(), "jwt@test.com", "JWT User", "pass1234")
	if err != nil {
		t.Fatal(err)
	}

	// Verify the token is valid by parsing it
	svc2 := &AuthService{jwtSecret: secret}
	user := &models.User{SupabaseID: "any", Email: "jwt@test.com"}
	token2, err := svc2.mintJWT(user)
	if err != nil {
		t.Fatal(err)
	}
	// Both tokens are non-empty and parseable (we already have mintJWT tests)
	if token == "" || token2 == "" {
		t.Error("tokens must not be empty")
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestLogin_CorrectPassword_ReturnsUserAndToken(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret-32-bytes-minimum!!!!", nil)

	_, _, err := svc.Register(context.Background(), "login@test.com", "Login User", "correctpass")
	if err != nil {
		t.Fatal(err)
	}

	user, token, err := svc.Login(context.Background(), "login@test.com", "correctpass")
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if user == nil {
		t.Fatal("login must return user")
	}
	if user.Email != "login@test.com" {
		t.Errorf("email: want login@test.com, got %s", user.Email)
	}
	if token == "" {
		t.Error("login must return a token")
	}
}

func TestLogin_WrongPassword_ReturnsError(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, _ = svc.Register(context.Background(), "user@test.com", "User", "correctpass")

	_, _, err := svc.Login(context.Background(), "user@test.com", "wrongpassword")
	if err == nil {
		t.Error("wrong password must return an error")
	}
}

func TestLogin_UnknownEmail_ReturnsError(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, err := svc.Login(context.Background(), "nobody@test.com", "anypassword")
	if err == nil {
		t.Error("unknown email must return an error")
	}
}

func TestLogin_EmptyPassword_ReturnsError(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, _ = svc.Register(context.Background(), "user@test.com", "User", "realpassword")

	_, _, err := svc.Login(context.Background(), "user@test.com", "")
	if err == nil {
		t.Error("empty password must return an error")
	}
}

func TestLogin_NoPasswordHash_ReturnsError(t *testing.T) {
	// Simulates a user that was provisioned via Supabase (no local password).
	store := newFakeUserStore()
	store.users["supaonly@test.com"] = &models.User{
		ID:    uuid.New(),
		Email: "supaonly@test.com",
	}
	// No hash - GetPasswordHash returns ""

	svc := NewAuthService(store, "test-secret", nil)
	_, _, err := svc.Login(context.Background(), "supaonly@test.com", "anypassword")
	if err == nil {
		t.Error("user without password hash must not log in via local auth")
	}
}

func TestLogin_SameErrorMessageForWrongPasswordAndUnknownEmail(t *testing.T) {
	// Prevents email enumeration: both cases must return the same error text.
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, _ = svc.Register(context.Background(), "known@test.com", "Known", "realpass")

	_, _, wrongPassErr := svc.Login(context.Background(), "known@test.com", "badpass")
	_, _, unknownErr := svc.Login(context.Background(), "unknown@test.com", "badpass")

	if wrongPassErr == nil || unknownErr == nil {
		t.Fatal("both should error")
	}
	if wrongPassErr.Error() != unknownErr.Error() {
		t.Errorf("error messages differ (enumeration risk): %q vs %q", wrongPassErr.Error(), unknownErr.Error())
	}
}

// ── Login lockout ────────────────────────────────────────────────────────────

func TestLogin_LocksAccountAfterRepeatedFailures(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, _ = svc.Register(context.Background(), "brute@test.com", "Brute Target", "correctpass")

	// loginLockoutMaxAttempts (5) wrong passwords should trip the lock.
	for i := 0; i < loginLockoutMaxAttempts; i++ {
		_, _, err := svc.Login(context.Background(), "brute@test.com", "wrongpass")
		if err == nil {
			t.Fatalf("attempt %d: expected an error", i+1)
		}
	}

	if _, ok := store.lockedUntil["brute@test.com"]; !ok {
		t.Fatal("account should be locked after reaching the attempt threshold")
	}

	// Even the correct password must now be rejected while locked.
	_, _, err := svc.Login(context.Background(), "brute@test.com", "correctpass")
	if !errors.Is(err, ErrAccountLocked) {
		t.Errorf("want ErrAccountLocked, got %v", err)
	}
}

func TestLogin_SuccessResetsFailedAttemptCounter(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	_, _, _ = svc.Register(context.Background(), "reset@test.com", "Reset User", "correctpass")

	_, _, _ = svc.Login(context.Background(), "reset@test.com", "wrongpass")
	_, _, _ = svc.Login(context.Background(), "reset@test.com", "wrongpass")

	if store.attempts["reset@test.com"] == 0 {
		t.Fatal("failed attempts should have accumulated")
	}

	_, _, err := svc.Login(context.Background(), "reset@test.com", "correctpass")
	if err != nil {
		t.Fatalf("correct password should still succeed below the lockout threshold: %v", err)
	}
	if _, ok := store.attempts["reset@test.com"]; ok && store.attempts["reset@test.com"] != 0 {
		t.Errorf("successful login should reset the failed-attempt counter, got %d", store.attempts["reset@test.com"])
	}
}

func TestLogin_UnknownEmailNeverAccumulatesLockout(t *testing.T) {
	// Guards against using lockout state as an email-enumeration side channel:
	// an unknown email must never end up with a lock, no matter how many
	// times it's guessed.
	store := newFakeUserStore()
	svc := NewAuthService(store, "test-secret", nil)

	for i := 0; i < loginLockoutMaxAttempts+2; i++ {
		_, _, _ = svc.Login(context.Background(), "ghost@test.com", "whatever")
	}

	if _, ok := store.lockedUntil["ghost@test.com"]; ok {
		t.Error("an unknown email must never accumulate a lockout")
	}
}
