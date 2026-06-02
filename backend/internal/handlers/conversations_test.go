package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Fake stores ───────────────────────────────────────────────────────────────

type hFakeConvRepo struct {
	conv   *models.Conversation
	msgs   []models.ConversationMessage
	getErr error
	addErr error
}

func (r *hFakeConvRepo) GetOrCreate(_ context.Context, entryID, userID uuid.UUID) (*models.Conversation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.conv == nil {
		r.conv = &models.Conversation{
			ID:        uuid.New(),
			EntryID:   entryID,
			UserID:    userID,
			TurnCount: 0,
			IsClosed:  false,
		}
	}
	return r.conv, nil
}

func (r *hFakeConvRepo) GetByID(_ context.Context, id, userID uuid.UUID) (*models.Conversation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.conv == nil || r.conv.ID != id || r.conv.UserID != userID {
		return nil, nil
	}
	return r.conv, nil
}

func (r *hFakeConvRepo) AddMessage(_ context.Context, convID uuid.UUID, role, content string) (*models.Conversation, *models.ConversationMessage, error) {
	if r.addErr != nil {
		return nil, nil, r.addErr
	}
	msg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           role,
		Content:        content,
	}
	r.msgs = append(r.msgs, *msg)
	if role == "user" && r.conv != nil {
		r.conv.TurnCount++
		if r.conv.TurnCount >= models.MaxConversationTurns {
			r.conv.IsClosed = true
		}
	}
	return r.conv, msg, nil
}

func (r *hFakeConvRepo) ListMessages(_ context.Context, _ uuid.UUID) ([]models.ConversationMessage, error) {
	return r.msgs, nil
}

type hFakeEntryReader struct {
	entry *models.Entry
}

func (r *hFakeEntryReader) GetByIDInternal(_ context.Context, id uuid.UUID) (*models.Entry, error) {
	if r.entry != nil && r.entry.ID == id {
		return r.entry, nil
	}
	return nil, nil
}

type hFakeAnalysisReader struct {
	analysis *models.EntryAnalysis
}

func (r *hFakeAnalysisReader) GetByEntryID(_ context.Context, _ uuid.UUID) (*models.EntryAnalysis, error) {
	return r.analysis, nil
}

// fakeProvisioner satisfies middleware.userProvisioner (unexported interface, structural match).
type fakeProvisioner struct{ user *models.User }

func (p *fakeProvisioner) GetOrCreate(_ context.Context, _, _, _ string) (*models.User, error) {
	return p.user, nil
}

// ── Test router builder ───────────────────────────────────────────────────────

const convTestSecret = "test-jwt-secret-32-bytes-minimum"

func newConvTestRouter(t *testing.T, cr services.ConvRepository, er services.EntryStoreReader, ar services.AnalysisStoreReader, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	claude := services.NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "test"})
	svc := services.NewConversationService(cr, er, ar, claude)

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(convTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewConversationHandler(svc)
	r.POST("/entries/:id/conversation", h.GetOrCreate)
	r.POST("/conversations/:id/messages", h.SendMessage)
	return r
}

func convTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-001",
		"email": "conv@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(convTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func authHeader(t *testing.T) string {
	return "Bearer " + convTestJWT(t)
}

func convTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "conv@test.com", Name: "Test User"}
}

func convTestEntry(entryID uuid.UUID) *models.Entry {
	tr := "I had a great day."
	return &models.Entry{
		ID:         entryID,
		Status:     models.EntryStatusCompleted,
		Transcript: &tr,
	}
}

func convTestAnalysis(entryID uuid.UUID) *models.EntryAnalysis {
	return &models.EntryAnalysis{
		EntryID:    entryID,
		Reflection: "Great day. What made it special?",
	}
}

// ── GetOrCreate tests ─────────────────────────────────────────────────────────

func TestConvHandler_GetOrCreate_Returns200(t *testing.T) {
	entryID := uuid.New()
	user := convTestUser()
	cr := &hFakeConvRepo{}
	er := &hFakeEntryReader{entry: convTestEntry(entryID)}
	ar := &hFakeAnalysisReader{analysis: convTestAnalysis(entryID)}

	r := newConvTestRouter(t, cr, er, ar, user)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/"+entryID.String()+"/conversation", nil)
	req.Header.Set("Authorization", authHeader(t))

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.Conversation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.EntryID != entryID {
		t.Errorf("entry_id: want %v, got %v", entryID, resp.EntryID)
	}
}

func TestConvHandler_GetOrCreate_InvalidEntryID_Returns400(t *testing.T) {
	user := convTestUser()
	cr := &hFakeConvRepo{}
	r := newConvTestRouter(t, cr, &hFakeEntryReader{}, &hFakeAnalysisReader{}, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/not-a-uuid/conversation", nil)
	req.Header.Set("Authorization", authHeader(t))

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid entry id: want 400, got %d", w.Code)
	}
}

func TestConvHandler_GetOrCreate_MissingAuth_Returns401(t *testing.T) {
	cr := &hFakeConvRepo{}
	r := newConvTestRouter(t, cr, &hFakeEntryReader{}, &hFakeAnalysisReader{}, convTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/"+uuid.New().String()+"/conversation", nil)
	// No Authorization header

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── SendMessage tests ─────────────────────────────────────────────────────────

func TestConvHandler_SendMessage_FirstTurn_Returns200(t *testing.T) {
	entryID := uuid.New()
	user := convTestUser()
	convID := uuid.New()

	cr := &hFakeConvRepo{conv: &models.Conversation{
		ID:        convID,
		EntryID:   entryID,
		UserID:    user.ID,
		TurnCount: 0,
		IsClosed:  false,
	}}
	er := &hFakeEntryReader{entry: convTestEntry(entryID)}
	ar := &hFakeAnalysisReader{analysis: convTestAnalysis(entryID)}

	r := newConvTestRouter(t, cr, er, ar, user)

	body := toJSON(t, map[string]string{"content": "Hello, how are you?"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader(t))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("first turn: want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConvHandler_SendMessage_ThirdTurn_ClosesConv(t *testing.T) {
	entryID := uuid.New()
	user := convTestUser()
	convID := uuid.New()

	cr := &hFakeConvRepo{conv: &models.Conversation{
		ID:        convID,
		EntryID:   entryID,
		UserID:    user.ID,
		TurnCount: 2, // one away from max
		IsClosed:  false,
	}}
	er := &hFakeEntryReader{entry: convTestEntry(entryID)}
	ar := &hFakeAnalysisReader{analysis: convTestAnalysis(entryID)}

	r := newConvTestRouter(t, cr, er, ar, user)

	body := toJSON(t, map[string]string{"content": "This is my third message."})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader(t))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("third turn: want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.Conversation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.IsClosed {
		t.Error("conversation must be closed after the 3rd user turn")
	}
}

func TestConvHandler_SendMessage_FourthTurn_Returns409(t *testing.T) {
	entryID := uuid.New()
	user := convTestUser()
	convID := uuid.New()

	// Conversation is already closed (3 turns done).
	cr := &hFakeConvRepo{conv: &models.Conversation{
		ID:        convID,
		EntryID:   entryID,
		UserID:    user.ID,
		TurnCount: 3,
		IsClosed:  true,
	}}

	r := newConvTestRouter(t, cr, &hFakeEntryReader{}, &hFakeAnalysisReader{}, user)

	body := toJSON(t, map[string]string{"content": "One more!"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader(t))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("4th turn: want 409 Conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConvHandler_SendMessage_EmptyContent_Returns400(t *testing.T) {
	user := convTestUser()
	convID := uuid.New()
	cr := &hFakeConvRepo{conv: &models.Conversation{
		ID:      convID,
		UserID:  user.ID,
		EntryID: uuid.New(),
	}}

	r := newConvTestRouter(t, cr, &hFakeEntryReader{}, &hFakeAnalysisReader{}, user)

	body := toJSON(t, map[string]string{"content": ""})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader(t))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("empty content: want 400, got %d", w.Code)
	}
}

func TestConvHandler_SendMessage_InvalidConvID_Returns400(t *testing.T) {
	user := convTestUser()
	r := newConvTestRouter(t, &hFakeConvRepo{}, &hFakeEntryReader{}, &hFakeAnalysisReader{}, user)

	body := toJSON(t, map[string]string{"content": "Hello"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/conversations/not-a-uuid/messages", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader(t))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid conv id: want 400, got %d", w.Code)
	}
}

func TestConvHandler_SendMessage_MissingAuth_Returns401(t *testing.T) {
	r := newConvTestRouter(t, &hFakeConvRepo{}, &hFakeEntryReader{}, &hFakeAnalysisReader{}, convTestUser())

	body := toJSON(t, map[string]string{"content": "Hello"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/conversations/"+uuid.New().String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}
