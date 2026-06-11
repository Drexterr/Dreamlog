package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeTherapyServicer struct {
	session     *models.TherapySession
	sendResp    *models.SendTherapyMessageResponse
	endResp     *models.EndSessionResponse
	listResp    *models.ListTherapySessionsResponse
	startErr    error
	sendErr     error
	endErr      error
	getErr      error
	gotPlan     models.Plan
	gotPersona  models.TherapyPersona
}

func (f *fakeTherapyServicer) StartSession(_ context.Context, userID uuid.UUID, plan models.Plan, persona models.TherapyPersona, _ string) (*models.TherapySession, error) {
	f.gotPlan = plan
	f.gotPersona = persona
	if f.startErr != nil {
		return nil, f.startErr
	}
	if f.session == nil {
		f.session = &models.TherapySession{
			ID: uuid.New(), UserID: userID, Status: models.TherapyStatusActive,
			Persona: persona, StartedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
		}
	}
	return f.session, nil
}

func (f *fakeTherapyServicer) SendMessage(_ context.Context, _, _ uuid.UUID, _ models.SendTherapyMessageRequest) (*models.SendTherapyMessageResponse, error) {
	return f.sendResp, f.sendErr
}

func (f *fakeTherapyServicer) EndSession(_ context.Context, _, _ uuid.UUID) (*models.EndSessionResponse, error) {
	return f.endResp, f.endErr
}

func (f *fakeTherapyServicer) GetSession(_ context.Context, _, _ uuid.UUID) (*models.TherapySession, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.session, nil
}

func (f *fakeTherapyServicer) ListSessions(_ context.Context, _ uuid.UUID) (*models.ListTherapySessionsResponse, error) {
	if f.listResp == nil {
		return &models.ListTherapySessionsResponse{Sessions: []models.TherapySessionSummary{}}, nil
	}
	return f.listResp, nil
}

type fakeTherapyPresigner struct{ err error }

func (f *fakeTherapyPresigner) PresignPut(_ context.Context, key, _ string, _ time.Duration) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return "https://storage.test/upload/" + key, "therapy/" + key, nil
}

type fakeTherapyUserGetter struct{ user *models.User }

func (f *fakeTherapyUserGetter) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if f.user != nil {
		return f.user, nil
	}
	return &models.User{ID: id, Plan: models.PlanFree}, nil
}

// ── router setup ──────────────────────────────────────────────────────────────

const therapyTestSecret = "therapy-test-jwt-secret-32bytes!"

func newTherapyTestRouter(t *testing.T, svc therapyServicer, presigner therapyPresigner, userGetter therapyUserGetter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(therapyTestSecret, "", &fakeProvisioner{user: &models.User{ID: uuid.New(), Email: "t@test.com"}}, log))

	h := NewTherapyHandler(svc, presigner, userGetter)
	r.POST("/therapy/sessions", h.StartSession)
	r.POST("/therapy/sessions/:id/presign", h.PresignAudio)
	r.POST("/therapy/sessions/:id/messages", h.SendMessage)
	r.POST("/therapy/sessions/:id/end", h.EndSession)
	r.GET("/therapy/sessions/:id", h.GetSession)
	r.GET("/therapy/sessions", h.ListSessions)
	return r
}

func therapyTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": "test-sub-therapy-001", "email": "t@test.com",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(therapyTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func therapyDo(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+therapyTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// ── StartSession ──────────────────────────────────────────────────────────────

func TestTherapyHandler_StartSession_DefaultPersona_Comforting(t *testing.T) {
	svc := &fakeTherapyServicer{}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})

	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions", map[string]string{})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if svc.gotPersona != models.PersonaComforting {
		t.Errorf("default persona must be comforting, got %s", svc.gotPersona)
	}
}

func TestTherapyHandler_StartSession_AllPersonasAccepted(t *testing.T) {
	for _, p := range []string{"comforting", "rational", "cbt", "mindful"} {
		svc := &fakeTherapyServicer{}
		r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
		w := therapyDo(t, r, http.MethodPost, "/therapy/sessions", map[string]string{"persona": p})
		if w.Code != http.StatusCreated {
			t.Errorf("persona %s: expected 201, got %d", p, w.Code)
		}
	}
}

func TestTherapyHandler_StartSession_InvalidPersona_Returns400(t *testing.T) {
	r := newTherapyTestRouter(t, &fakeTherapyServicer{}, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions", map[string]string{"persona": "freudian"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid persona, got %d", w.Code)
	}
}

func TestTherapyHandler_StartSession_NoCredits_Returns402(t *testing.T) {
	// The handler matches this by message (the service sentinel is unexported).
	svc := &fakeTherapyServicer{startErr: errors.New("therapy session requires payment")}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions", nil)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTherapyHandler_StartSession_UsesEffectivePlan_ExpiredProIsFree(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	svc := &fakeTherapyServicer{}
	getter := &fakeTherapyUserGetter{user: &models.User{ID: uuid.New(), Plan: models.PlanPro, PlanExpiresAt: &expired}}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, getter)

	if w := therapyDo(t, r, http.MethodPost, "/therapy/sessions", nil); w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if svc.gotPlan != models.PlanFree {
		t.Errorf("expired pro plan must be passed as free, got %s", svc.gotPlan)
	}
}

// ── SendMessage ───────────────────────────────────────────────────────────────

func TestTherapyHandler_SendMessage_TextWithoutContent_Returns400(t *testing.T) {
	r := newTherapyTestRouter(t, &fakeTherapyServicer{}, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/messages",
		map[string]string{"input_mode": "text"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTherapyHandler_SendMessage_VoiceWithoutAudioKey_Returns400(t *testing.T) {
	r := newTherapyTestRouter(t, &fakeTherapyServicer{}, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/messages",
		map[string]string{"input_mode": "voice"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTherapyHandler_SendMessage_InvalidSessionID_Returns400(t *testing.T) {
	r := newTherapyTestRouter(t, &fakeTherapyServicer{}, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/not-a-uuid/messages",
		map[string]string{"input_mode": "text", "content": "hello"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad uuid, got %d", w.Code)
	}
}

func TestTherapyHandler_SendMessage_NotFound_Returns404(t *testing.T) {
	svc := &fakeTherapyServicer{sendErr: services.ErrTherapyNotFound}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/messages",
		map[string]string{"input_mode": "text", "content": "hello"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTherapyHandler_SendMessage_NotActive_Returns409(t *testing.T) {
	svc := &fakeTherapyServicer{sendErr: services.ErrTherapyNotActive}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/messages",
		map[string]string{"input_mode": "text", "content": "hello"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestTherapyHandler_SendMessage_Expired_Returns410(t *testing.T) {
	svc := &fakeTherapyServicer{sendErr: services.ErrTherapyExpired}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/messages",
		map[string]string{"input_mode": "text", "content": "hello"})
	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 for expired session, got %d", w.Code)
	}
}

func TestTherapyHandler_SendMessage_Success_Returns201(t *testing.T) {
	svc := &fakeTherapyServicer{sendResp: &models.SendTherapyMessageResponse{
		UserMessage:      models.TherapySessionMessage{Role: "user", Content: "hello"},
		AssistantMessage: models.TherapySessionMessage{Role: "assistant", Content: "hi"},
		SessionState:     models.TherapySessionState{Status: models.TherapyStatusActive, TurnCount: 1},
	}}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/messages",
		map[string]string{"input_mode": "text", "content": "hello"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// ── EndSession ────────────────────────────────────────────────────────────────

func TestTherapyHandler_EndSession_Success_Returns200(t *testing.T) {
	svc := &fakeTherapyServicer{endResp: &models.EndSessionResponse{
		SessionID: uuid.New(), Status: "completed", DurationSec: 100, TurnCount: 4,
		PostSessionSummary: "A good session.",
	}}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/end", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTherapyHandler_EndSession_AlreadyEnded_Returns409(t *testing.T) {
	svc := &fakeTherapyServicer{endErr: services.ErrTherapyAlreadyEnded}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/end", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for double-end, got %d", w.Code)
	}
}

func TestTherapyHandler_EndSession_NotFound_Returns404(t *testing.T) {
	svc := &fakeTherapyServicer{endErr: services.ErrTherapyNotFound}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+uuid.NewString()+"/end", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ── GetSession / ListSessions / Presign ───────────────────────────────────────

func TestTherapyHandler_GetSession_NotFound_Returns404(t *testing.T) {
	svc := &fakeTherapyServicer{getErr: services.ErrTherapyNotFound}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodGet, "/therapy/sessions/"+uuid.NewString(), nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTherapyHandler_GetSession_ReturnsSession(t *testing.T) {
	session := &models.TherapySession{
		ID: uuid.New(), Status: models.TherapyStatusActive,
		StartedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	svc := &fakeTherapyServicer{session: session}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodGet, "/therapy/sessions/"+session.ID.String(), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTherapyHandler_ListSessions_Returns200(t *testing.T) {
	r := newTherapyTestRouter(t, &fakeTherapyServicer{}, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodGet, "/therapy/sessions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTherapyHandler_Presign_ActiveSession_ReturnsURL(t *testing.T) {
	session := &models.TherapySession{ID: uuid.New(), Status: models.TherapyStatusActive}
	svc := &fakeTherapyServicer{session: session}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+session.ID.String()+"/presign",
		map[string]string{"filename": "turn.aac", "content_type": "audio/aac"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["upload_url"] == "" || resp["audio_key"] == "" {
		t.Error("presign must return upload_url and audio_key")
	}
}

func TestTherapyHandler_Presign_InactiveSession_Returns409(t *testing.T) {
	session := &models.TherapySession{ID: uuid.New(), Status: models.TherapyStatusCompleted}
	svc := &fakeTherapyServicer{session: session}
	r := newTherapyTestRouter(t, svc, &fakeTherapyPresigner{}, &fakeTherapyUserGetter{})
	w := therapyDo(t, r, http.MethodPost, "/therapy/sessions/"+session.ID.String()+"/presign",
		map[string]string{"filename": "turn.aac", "content_type": "audio/aac"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for inactive session, got %d", w.Code)
	}
}
