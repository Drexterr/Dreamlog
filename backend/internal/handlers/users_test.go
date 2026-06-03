package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake userProfiler ─────────────────────────────────────────────────────────

type fakeUserProfiler struct {
	updateResp *models.User
	updateErr  error
}

func (f *fakeUserProfiler) UpdateProfile(_ context.Context, _ uuid.UUID, _ models.UpdateUserInput) (*models.User, error) {
	return f.updateResp, f.updateErr
}

func (f *fakeUserProfiler) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

// ── test helpers ──────────────────────────────────────────────────────────────

const userTestSecret = "user-test-jwt-secret-32-bytes!!!"

func newUserTestRouter(t *testing.T, svc userProfiler, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(userTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := &UserHandler{svc: svc}
	r.GET("/me", h.GetMe)
	r.PUT("/me", h.UpdateMe)
	return r
}

func userTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-user-001",
		"email": "user@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(userTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func userTestUser() *models.User {
	goal := "stress"
	return &models.User{
		ID:    uuid.New(),
		Email: "user@test.com",
		Name:  "Test User",
		Goal:  &goal,
	}
}

// ── GET /me tests ─────────────────────────────────────────────────────────────

func TestUserHandler_GetMe_Returns200WithUser(t *testing.T) {
	testUser := userTestUser()
	r := newUserTestRouter(t, &fakeUserProfiler{}, testUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /me: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Email != testUser.Email {
		t.Errorf("email: want %q, got %q", testUser.Email, resp.Email)
	}
}

func TestUserHandler_GetMe_MissingAuth_Returns401(t *testing.T) {
	r := newUserTestRouter(t, &fakeUserProfiler{}, userTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── PUT /me tests ─────────────────────────────────────────────────────────────

func TestUserHandler_UpdateMe_NameOnly_Returns200(t *testing.T) {
	updated := userTestUser()
	updated.Name = "New Name"
	svc := &fakeUserProfiler{updateResp: updated}
	r := newUserTestRouter(t, svc, userTestUser())

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("name update: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "New Name" {
		t.Errorf("name: want %q, got %q", "New Name", resp.Name)
	}
}

func TestUserHandler_UpdateMe_GoalOnly_Returns200(t *testing.T) {
	goal := "anxiety"
	updated := userTestUser()
	updated.Goal = &goal
	svc := &fakeUserProfiler{updateResp: updated}
	r := newUserTestRouter(t, svc, userTestUser())

	body, _ := json.Marshal(map[string]string{"goal": "anxiety"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("goal update: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Goal == nil || *resp.Goal != "anxiety" {
		t.Errorf("goal: want %q, got %v", "anxiety", resp.Goal)
	}
}

func TestUserHandler_UpdateMe_PreferredName_Returns200(t *testing.T) {
	pn := "Alex"
	updated := userTestUser()
	updated.PreferredName = &pn
	svc := &fakeUserProfiler{updateResp: updated}
	r := newUserTestRouter(t, svc, userTestUser())

	body, _ := json.Marshal(map[string]string{"preferred_name": "Alex"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("preferred_name update: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PreferredName == nil || *resp.PreferredName != "Alex" {
		t.Errorf("preferred_name: want %q, got %v", "Alex", resp.PreferredName)
	}
}

func TestUserHandler_UpdateMe_AllFields_Returns200(t *testing.T) {
	pn := "B"
	goal := "curious"
	nudgeHour := 7
	tz := "Asia/Kolkata"
	updated := userTestUser()
	updated.Name = "Bharat"
	updated.PreferredName = &pn
	updated.Goal = &goal
	updated.FCMNudgeHour = nudgeHour
	updated.Timezone = tz
	svc := &fakeUserProfiler{updateResp: updated}
	r := newUserTestRouter(t, svc, userTestUser())

	body, _ := json.Marshal(map[string]any{
		"name":           "Bharat",
		"preferred_name": "B",
		"goal":           "curious",
		"fcm_nudge_hour": 7,
		"timezone":       "Asia/Kolkata",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("all-fields update: want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UpdateMe_NoFields_Returns400(t *testing.T) {
	r := newUserTestRouter(t, &fakeUserProfiler{}, userTestUser())

	body, _ := json.Marshal(map[string]any{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("no fields: want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UpdateMe_InvalidGoal_Returns400(t *testing.T) {
	r := newUserTestRouter(t, &fakeUserProfiler{}, userTestUser())

	body, _ := json.Marshal(map[string]string{"goal": "world_domination"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid goal: want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UpdateMe_FCMNudgeHourOutOfRange_Returns400(t *testing.T) {
	r := newUserTestRouter(t, &fakeUserProfiler{}, userTestUser())

	body, _ := json.Marshal(map[string]any{"fcm_nudge_hour": 25})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("nudge hour 25: want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UpdateMe_MissingAuth_Returns401(t *testing.T) {
	r := newUserTestRouter(t, &fakeUserProfiler{}, userTestUser())

	body, _ := json.Marshal(map[string]string{"goal": "stress"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestUserHandler_UpdateMe_EmptyBody_Returns400(t *testing.T) {
	r := newUserTestRouter(t, &fakeUserProfiler{}, userTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("empty body: want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UpdateMe_ServiceError_Returns500(t *testing.T) {
	svc := &fakeUserProfiler{updateErr: &fakeServiceError{msg: "db connection lost"}}
	r := newUserTestRouter(t, svc, userTestUser())

	body, _ := json.Marshal(map[string]string{"goal": "stress"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code < 500 {
		t.Errorf("service error: want 5xx, got %d", w.Code)
	}
}

// ── test error type ───────────────────────────────────────────────────────────

type fakeServiceError struct{ msg string }

func (e *fakeServiceError) Error() string { return e.msg }
