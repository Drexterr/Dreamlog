package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Fake user store for handler tests ────────────────────────────────────────

type handlerFakeUserStore struct {
	users     map[string]*models.User
	hashes    map[string]string
	createErr error
}

func newHandlerFakeStore() *handlerFakeUserStore {
	return &handlerFakeUserStore{
		users:  make(map[string]*models.User),
		hashes: make(map[string]string),
	}
}

func (s *handlerFakeUserStore) GetByEmail(_ context.Context, email string) (*models.User, error) {
	u := s.users[email]
	if u != nil && u.IsDeleted {
		return nil, nil
	}
	return u, nil
}

func (s *handlerFakeUserStore) GetByEmailIncDeleted(_ context.Context, email string) (*models.User, error) {
	return s.users[email], nil
}

func (s *handlerFakeUserStore) CreateLocal(_ context.Context, email, name, hash string) (*models.User, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	u := &models.User{ID: uuid.New(), SupabaseID: "local-" + uuid.New().String(), Email: email, Name: name}
	s.users[email] = u
	s.hashes[email] = hash
	return u, nil
}

func (s *handlerFakeUserStore) GetPasswordHash(_ context.Context, email string) (string, error) {
	return s.hashes[email], nil
}

func (s *handlerFakeUserStore) Reactivate(_ context.Context, id uuid.UUID, name, hash string) (*models.User, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	for email, u := range s.users {
		if u.ID == id {
			u.IsDeleted = false
			u.Name = name
			s.hashes[email] = hash
			return u, nil
		}
	}
	return nil, nil
}

// ── Router builder ────────────────────────────────────────────────────────────

func newAuthTestRouter(store services.UserStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	svc := services.NewAuthService(store, "test-secret-32-bytes-minimum!!!!")
	h := NewAuthHandler(svc)

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	return r
}

func toJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// ── Register tests ────────────────────────────────────────────────────────────

func TestAuthHandler_Register_ValidInput_Returns201(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{
		"email": "alice@test.com", "name": "Alice", "password": "secure123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["token"] == "" || resp["token"] == nil {
		t.Error("register response must include a token")
	}
	if resp["user"] == nil {
		t.Error("register response must include user object")
	}
}

func TestAuthHandler_Register_MissingEmail_Returns400(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{"name": "Alice", "password": "secure123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing email: want 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_InvalidEmail_Returns400(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{
		"email": "not-an-email", "name": "Alice", "password": "secure123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid email: want 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ShortPassword_Returns400(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{
		"email": "alice@test.com", "name": "Alice", "password": "short",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("short password: want 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_DuplicateEmail_Returns409(t *testing.T) {
	store := newHandlerFakeStore()
	r := newAuthTestRouter(store)

	body := toJSON(t, map[string]string{
		"email": "dup@test.com", "name": "First", "password": "password1",
	})
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Second registration with same email.
	body2 := toJSON(t, map[string]string{
		"email": "dup@test.com", "name": "Second", "password": "password2",
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("duplicate email: want 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAuthHandler_Register_EmptyBody_Returns400(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", nil)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("empty body: want 400, got %d", w.Code)
	}
}

// ── Login tests ───────────────────────────────────────────────────────────────

func TestAuthHandler_Login_ValidCredentials_Returns200(t *testing.T) {
	store := newHandlerFakeStore()
	r := newAuthTestRouter(store)

	// Register first
	regBody := toJSON(t, map[string]string{
		"email": "loginuser@test.com", "name": "Login User", "password": "password123",
	})
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Now login
	loginBody := toJSON(t, map[string]string{
		"email": "loginuser@test.com", "password": "password123",
	})
	w := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Errorf("valid login: want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("login response must include a token")
	}
}

func TestAuthHandler_Login_WrongPassword_Returns401(t *testing.T) {
	store := newHandlerFakeStore()
	r := newAuthTestRouter(store)

	regBody := toJSON(t, map[string]string{
		"email": "pw@test.com", "name": "PW User", "password": "correctpass",
	})
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	loginBody := toJSON(t, map[string]string{
		"email": "pw@test.com", "password": "wrongpassword",
	})
	w := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: want 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_UnknownEmail_Returns401(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{
		"email": "ghost@test.com", "password": "anypassword",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("unknown email: want 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingPassword_Returns400(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{"email": "user@test.com"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing password: want 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingEmail_Returns400(t *testing.T) {
	r := newAuthTestRouter(newHandlerFakeStore())
	body := toJSON(t, map[string]string{"password": "anypassword"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing email: want 400, got %d", w.Code)
	}
}
