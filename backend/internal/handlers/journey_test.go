package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ── fake journey manager ──────────────────────────────────────────────────────

type fakeJourneyManager struct {
	templates    []models.JourneyTemplate
	startSession *models.JourneySession
	startErr     error
	getSession   *models.JourneySession
	getErr       error
	listSessions []*models.JourneySession
	listErr      error
	advSession   *models.JourneySession
	advErr       error
}

func (f *fakeJourneyManager) ListTemplates() []models.JourneyTemplate { return f.templates }

func (f *fakeJourneyManager) GetTemplate(journeyID string) (*models.JourneyTemplate, bool) {
	for i := range f.templates {
		if f.templates[i].ID == journeyID {
			return &f.templates[i], true
		}
	}
	return nil, false
}

func (f *fakeJourneyManager) StartSession(_ context.Context, _ uuid.UUID, _ string) (*models.JourneySession, error) {
	return f.startSession, f.startErr
}

func (f *fakeJourneyManager) GetSession(_ context.Context, _, _ uuid.UUID) (*models.JourneySession, error) {
	return f.getSession, f.getErr
}

func (f *fakeJourneyManager) ListSessions(_ context.Context, _ uuid.UUID) ([]*models.JourneySession, error) {
	return f.listSessions, f.listErr
}

func (f *fakeJourneyManager) AdvanceSession(_ context.Context, _, _ uuid.UUID, _ uuid.UUID) (*models.JourneySession, error) {
	return f.advSession, f.advErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

const journeyTestSecret = "journey-test-jwt-secret-32-bytes!"

func newJourneyTestRouter(t *testing.T, svc journeyManager, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(journeyTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := NewJourneyHandler(svc)
	r.GET("/journeys", h.ListTemplates)
	r.POST("/journeys/:journeyID/start", h.StartSession)
	r.GET("/journeys/sessions", h.ListSessions)
	r.GET("/journeys/sessions/:sessionID", h.GetSession)
	r.POST("/journeys/sessions/:sessionID/advance", h.AdvanceSession)
	return r
}

func journeyTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-journey-001",
		"email": "journey@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(journeyTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func journeyTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "journey@test.com", Name: "Journey User", Plan: models.PlanFree}
}

func sampleTemplate() models.JourneyTemplate {
	return models.JourneyTemplate{
		ID:               "stress_relief",
		Title:            "Stress Relief",
		Description:      "Explore what's weighing on you.",
		StepCount:        3,
		EstimatedMinutes: 8,
		Tags:             []string{"stress"},
		Prompts:          []string{"Prompt 1", "Prompt 2", "Prompt 3"},
	}
}

func sampleSession() *models.JourneySession {
	return &models.JourneySession{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		JourneyID:    "stress_relief",
		JourneyTitle: "Stress Relief",
		CurrentStep:  0,
		TotalSteps:   3,
		Status:       models.JourneyStatusInProgress,
		Steps: []models.JourneyStep{
			{StepIndex: 0, Prompt: "Prompt 1", Completed: false},
			{StepIndex: 1, Prompt: "Prompt 2", Completed: false},
			{StepIndex: 2, Prompt: "Prompt 3", Completed: false},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ── GET /journeys ─────────────────────────────────────────────────────────────

func TestJourneyHandler_ListTemplates_Returns200WithJourneys(t *testing.T) {
	svc := &fakeJourneyManager{templates: []models.JourneyTemplate{sampleTemplate()}}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/journeys", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	journeys, ok := resp["journeys"].([]any)
	if !ok || len(journeys) != 1 {
		t.Fatalf("expected 1 journey, got %v", resp["journeys"])
	}
}

func TestJourneyHandler_ListTemplates_MissingAuth_Returns401(t *testing.T) {
	svc := &fakeJourneyManager{templates: []models.JourneyTemplate{sampleTemplate()}}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/journeys", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

// ── POST /journeys/:journeyID/start ──────────────────────────────────────────

func TestJourneyHandler_StartSession_KnownJourney_Returns201(t *testing.T) {
	session := sampleSession()
	svc := &fakeJourneyManager{
		templates:    []models.JourneyTemplate{sampleTemplate()},
		startSession: session,
	}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/journeys/stress_relief/start", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.JourneySession
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.JourneyID != "stress_relief" {
		t.Errorf("want journey_id=stress_relief, got %s", resp.JourneyID)
	}
	if len(resp.Steps) != 3 {
		t.Errorf("want 3 steps, got %d", len(resp.Steps))
	}
}

func TestJourneyHandler_StartSession_UnknownJourney_Returns404(t *testing.T) {
	svc := &fakeJourneyManager{templates: []models.JourneyTemplate{}} // no templates
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/journeys/nonexistent/start", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJourneyHandler_StartSession_ServiceError_Returns500(t *testing.T) {
	svc := &fakeJourneyManager{
		templates: []models.JourneyTemplate{sampleTemplate()},
		startErr:  errors.New("db error"),
	}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/journeys/stress_relief/start", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

// ── GET /journeys/sessions ────────────────────────────────────────────────────

func TestJourneyHandler_ListSessions_Returns200(t *testing.T) {
	session := sampleSession()
	svc := &fakeJourneyManager{listSessions: []*models.JourneySession{session}}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/journeys/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessions, ok := resp["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %v", resp["sessions"])
	}
}

func TestJourneyHandler_ListSessions_NilReturnsEmptyArray(t *testing.T) {
	svc := &fakeJourneyManager{listSessions: nil}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/journeys/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessions, ok := resp["sessions"].([]any)
	if !ok {
		t.Fatal("sessions field must be an array, not null")
	}
	if len(sessions) != 0 {
		t.Errorf("want empty array, got %d items", len(sessions))
	}
}

// ── GET /journeys/sessions/:sessionID ────────────────────────────────────────

func TestJourneyHandler_GetSession_Returns200WithSteps(t *testing.T) {
	session := sampleSession()
	svc := &fakeJourneyManager{getSession: session}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/journeys/sessions/%s", session.ID), nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.JourneySession
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Steps) != 3 {
		t.Errorf("want 3 steps, got %d", len(resp.Steps))
	}
}

func TestJourneyHandler_GetSession_InvalidID_Returns400(t *testing.T) {
	svc := &fakeJourneyManager{}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/journeys/sessions/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestJourneyHandler_GetSession_NotFound_Returns404(t *testing.T) {
	svc := &fakeJourneyManager{getErr: errors.New("not found")}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/journeys/sessions/%s", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /journeys/sessions/:sessionID/advance ────────────────────────────────

func TestJourneyHandler_AdvanceSession_Returns200WithUpdatedSession(t *testing.T) {
	updated := sampleSession()
	updated.CurrentStep = 1
	svc := &fakeJourneyManager{advSession: updated}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	entryID := uuid.New()
	body, _ := json.Marshal(map[string]string{"entry_id": entryID.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/journeys/sessions/%s/advance", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.JourneySession
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.CurrentStep != 1 {
		t.Errorf("want current_step=1, got %d", resp.CurrentStep)
	}
}

func TestJourneyHandler_AdvanceSession_MissingEntryID_Returns400(t *testing.T) {
	svc := &fakeJourneyManager{}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/journeys/sessions/%s/advance", uuid.New()), bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestJourneyHandler_AdvanceSession_AlreadyCompleted_Returns409(t *testing.T) {
	svc := &fakeJourneyManager{advErr: errors.New("session already completed")}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	entryID := uuid.New()
	body, _ := json.Marshal(map[string]string{"entry_id": entryID.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/journeys/sessions/%s/advance", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJourneyHandler_AdvanceSession_InvalidSessionID_Returns400(t *testing.T) {
	svc := &fakeJourneyManager{}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	body, _ := json.Marshal(map[string]string{"entry_id": uuid.New().String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/journeys/sessions/bad-id/advance", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+journeyTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestJourneyHandler_AdvanceSession_MissingAuth_Returns401(t *testing.T) {
	svc := &fakeJourneyManager{}
	r := newJourneyTestRouter(t, svc, journeyTestUser())

	body, _ := json.Marshal(map[string]string{"entry_id": uuid.New().String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/journeys/sessions/%s/advance", uuid.New()), bytes.NewReader(body))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}
