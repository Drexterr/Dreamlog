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
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake therapistRepo ────────────────────────────────────────────────────────

type fakeTherapistRepo struct {
	getByUserIDResp  *models.Therapist
	getByUserIDErr   error
	registerResp     *models.Therapist
	registerErr      error
	linkClientErr    error
	unlinkClientErr  error
	listClientsResp  []*models.ClientSummary
	listClientsErr   error
	getClientLinkResp *models.ClientTherapistLink
	getClientLinkErr  error
	recentEntriesResp []*models.ExportEntrySummary
	moodStatsAvg7d    *int
	moodStatsEmotions []string
	moodStatsTrend    string
	entryCount        int
	displayName       string
	summariesText     string
}

func (f *fakeTherapistRepo) GetByUserID(_ context.Context, _ uuid.UUID) (*models.Therapist, error) {
	return f.getByUserIDResp, f.getByUserIDErr
}

func (f *fakeTherapistRepo) Register(_ context.Context, _ uuid.UUID, name, email, credentials string) (*models.Therapist, error) {
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	if f.registerResp != nil {
		return f.registerResp, nil
	}
	return &models.Therapist{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Name:        name,
		Email:       email,
		Credentials: credentials,
		Plan:        "trial",
		CreatedAt:   time.Now(),
	}, nil
}

func (f *fakeTherapistRepo) LinkClient(_ context.Context, _, _ uuid.UUID) error {
	return f.linkClientErr
}

func (f *fakeTherapistRepo) UnlinkClient(_ context.Context, _, _ uuid.UUID) error {
	return f.unlinkClientErr
}

func (f *fakeTherapistRepo) ListClients(_ context.Context, _ uuid.UUID) ([]*models.ClientSummary, error) {
	return f.listClientsResp, f.listClientsErr
}

func (f *fakeTherapistRepo) GetClientLink(_ context.Context, _, _ uuid.UUID) (*models.ClientTherapistLink, error) {
	return f.getClientLinkResp, f.getClientLinkErr
}

func (f *fakeTherapistRepo) ClientRecentEntries(_ context.Context, _ uuid.UUID) ([]*models.ExportEntrySummary, error) {
	return f.recentEntriesResp, nil
}

func (f *fakeTherapistRepo) ClientMoodStats(_ context.Context, _ uuid.UUID) (avg7d *int, topEmotions []string, trend string, err error) {
	return f.moodStatsAvg7d, f.moodStatsEmotions, f.moodStatsTrend, nil
}

func (f *fakeTherapistRepo) ClientEntryCount(_ context.Context, _ uuid.UUID) (int, error) {
	return f.entryCount, nil
}

func (f *fakeTherapistRepo) ClientDisplayName(_ context.Context, _ uuid.UUID) (string, error) {
	return f.displayName, nil
}

func (f *fakeTherapistRepo) ClientRecentSummariesText(_ context.Context, _ uuid.UUID, _ time.Time) (string, error) {
	return f.summariesText, nil
}

// ── fake therapistAnalysisRepo ────────────────────────────────────────────────

type fakeTherapistAnalysisRepo struct {
	exportDataResp *models.ExportData
	exportDataErr  error
}

func (f *fakeTherapistAnalysisRepo) ExportData(_ context.Context, _ uuid.UUID, _, _ time.Time) (*models.ExportData, error) {
	return f.exportDataResp, f.exportDataErr
}

// ── fake briefGenerator ───────────────────────────────────────────────────────

type fakeBriefGenerator struct {
	brief string
	err   error
}

func (f *fakeBriefGenerator) GenerateBrief(_ context.Context, _, _, _ string, _ *int) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.brief != "" {
		return f.brief, nil
	}
	return "The client shows moderate mood this week. A stable trend is worth acknowledging. Consider asking: what felt most different for you this week?", nil
}

// ── router builder ────────────────────────────────────────────────────────────

const therapistTestSecret = "therapist-test-jwt-secret-32!!!"

func newTherapistTestRouter(
	t *testing.T,
	repo *fakeTherapistRepo,
	aRepo *fakeTherapistAnalysisRepo,
	gen *fakeBriefGenerator,
	testUser *models.User,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(therapistTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := NewTherapistHandler(repo, aRepo, gen)
	r.POST("/therapists/register", h.Register)
	r.POST("/therapists/clients/link", h.LinkClient)
	r.DELETE("/therapists/clients/:clientID", h.UnlinkClient)
	r.GET("/therapists/clients", h.ListClients)
	r.GET("/therapists/clients/:clientID/brief", h.ClientBrief)
	return r
}

func therapistTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-therapist-001",
		"email": "therapist@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(therapistTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func therapistTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "therapist@test.com", Name: "Dr Test"}
}

func registeredTherapist() *models.Therapist {
	return &models.Therapist{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Name:        "Dr Test",
		Email:       "therapist@test.com",
		Credentials: "PhD",
		Plan:        "trial",
		CreatedAt:   time.Now(),
	}
}

// ── Register tests ────────────────────────────────────────────────────────────

func TestTherapistHandler_Register_ValidInput_Returns201(t *testing.T) {
	repo := &fakeTherapistRepo{}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{
		"name":        "Dr Alice",
		"email":       "alice@clinic.com",
		"credentials": "PhD, Clinical Psychology",
	})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/register", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.Therapist
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "Dr Alice" {
		t.Errorf("name: want Dr Alice, got %s", resp.Name)
	}
	if resp.Email != "alice@clinic.com" {
		t.Errorf("email: want alice@clinic.com, got %s", resp.Email)
	}
	if resp.Plan != "trial" {
		t.Errorf("plan: want trial, got %s", resp.Plan)
	}
}

func TestTherapistHandler_Register_MissingName_Returns400(t *testing.T) {
	r := newTherapistTestRouter(t, &fakeTherapistRepo{}, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"email": "doc@clinic.com"})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/register", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing name: want 400, got %d", w.Code)
	}
}

func TestTherapistHandler_Register_MissingEmail_Returns400(t *testing.T) {
	r := newTherapistTestRouter(t, &fakeTherapistRepo{}, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"name": "Dr Alice"})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/register", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing email: want 400, got %d", w.Code)
	}
}

func TestTherapistHandler_Register_MissingAuth_Returns401(t *testing.T) {
	r := newTherapistTestRouter(t, &fakeTherapistRepo{}, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"name": "Dr Alice", "email": "alice@clinic.com"})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/register", bytes.NewReader(body))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── LinkClient tests ──────────────────────────────────────────────────────────

func TestTherapistHandler_LinkClient_ValidInput_Returns200(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: registeredTherapist()}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"client_id": uuid.New().String()})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/clients/link", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("link client: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body2 map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body2["status"] != "active" {
		t.Errorf("status: want active, got %v", body2["status"])
	}
}

func TestTherapistHandler_LinkClient_NotTherapist_Returns403(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: nil} // not registered as therapist
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"client_id": uuid.New().String()})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/clients/link", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("not therapist: want 403, got %d", w.Code)
	}
}

func TestTherapistHandler_LinkClient_InvalidClientID_Returns400(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: registeredTherapist()}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"client_id": "not-a-uuid"})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/clients/link", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid client_id: want 400, got %d", w.Code)
	}
}

func TestTherapistHandler_LinkClient_MissingClientID_Returns400(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: registeredTherapist()}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest(http.MethodPost, "/therapists/clients/link", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing client_id: want 400, got %d", w.Code)
	}
}

// ── UnlinkClient tests ────────────────────────────────────────────────────────

func TestTherapistHandler_UnlinkClient_ValidClient_Returns204(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: registeredTherapist()}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/therapists/clients/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("unlink: want 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTherapistHandler_UnlinkClient_NotTherapist_Returns403(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: nil}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/therapists/clients/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("not therapist: want 403, got %d", w.Code)
	}
}

func TestTherapistHandler_UnlinkClient_InvalidClientID_Returns400(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: registeredTherapist()}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/therapists/clients/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid id: want 400, got %d", w.Code)
	}
}

// ── ListClients tests ─────────────────────────────────────────────────────────

func TestTherapistHandler_ListClients_WithClients_Returns200(t *testing.T) {
	avg := 68
	repo := &fakeTherapistRepo{
		getByUserIDResp: registeredTherapist(),
		listClientsResp: []*models.ClientSummary{
			{ClientID: uuid.New(), Name: "Client One", EntryCount: 10, AvgMood30d: &avg},
			{ClientID: uuid.New(), Name: "Client Two", EntryCount: 5},
		},
	}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list clients: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	clients, ok := body["clients"].([]any)
	if !ok {
		t.Fatal("response must have clients array")
	}
	if len(clients) != 2 {
		t.Errorf("want 2 clients, got %d", len(clients))
	}
}

func TestTherapistHandler_ListClients_EmptyList_Returns200WithEmptyArray(t *testing.T) {
	repo := &fakeTherapistRepo{
		getByUserIDResp: registeredTherapist(),
		listClientsResp: nil,
	}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("empty list: want 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	clients, ok := body["clients"].([]any)
	if !ok {
		t.Fatal("response must have clients field")
	}
	if len(clients) != 0 {
		t.Errorf("want empty clients array, got %d items", len(clients))
	}
}

func TestTherapistHandler_ListClients_NotTherapist_Returns403(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: nil}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("not therapist: want 403, got %d", w.Code)
	}
}

// ── ClientBrief tests ─────────────────────────────────────────────────────────

func TestTherapistHandler_ClientBrief_LinkedClient_Returns200WithBrief(t *testing.T) {
	clientID := uuid.New()
	avg := 72
	repo := &fakeTherapistRepo{
		getByUserIDResp: registeredTherapist(),
		getClientLinkResp: &models.ClientTherapistLink{
			ID:       uuid.New(),
			ClientID: clientID,
			Status:   "active",
		},
		displayName:       "Alice",
		moodStatsAvg7d:    &avg,
		moodStatsEmotions: []string{"hopeful", "calm"},
		moodStatsTrend:    "improving",
		entryCount:        8,
		recentEntriesResp: []*models.ExportEntrySummary{
			{Date: time.Now(), Summary: "Had a productive week.", MoodScore: 72},
		},
		summariesText: "2026-05-21 | mood 72 | Had a productive week.",
	}
	gen := &fakeBriefGenerator{
		brief: "Alice shows improving mood this week with themes of productivity and calm. The upward trend in journaling frequency is worth acknowledging. Consider asking: what has felt most energising for you this week?",
	}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, gen, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients/"+clientID.String()+"/brief", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("client brief: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.ClientBrief
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Brief == "" {
		t.Error("brief must not be empty")
	}
	if resp.ClientName != "Alice" {
		t.Errorf("client_name: want Alice, got %s", resp.ClientName)
	}
	if resp.MoodTrend != "improving" {
		t.Errorf("mood_trend: want improving, got %s", resp.MoodTrend)
	}
	if resp.AvgMood7d == nil || *resp.AvgMood7d != 72 {
		t.Errorf("avg_mood_7d: want 72, got %v", resp.AvgMood7d)
	}
	if len(resp.TopEmotions) != 2 {
		t.Errorf("top_emotions: want 2, got %d", len(resp.TopEmotions))
	}
}

func TestTherapistHandler_ClientBrief_NotTherapist_Returns403(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: nil}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients/"+uuid.New().String()+"/brief", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("not therapist: want 403, got %d", w.Code)
	}
}

func TestTherapistHandler_ClientBrief_ClientNotLinked_Returns404(t *testing.T) {
	repo := &fakeTherapistRepo{
		getByUserIDResp:   registeredTherapist(),
		getClientLinkResp: nil, // link does not exist
	}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients/"+uuid.New().String()+"/brief", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("client not linked: want 404, got %d", w.Code)
	}
}

func TestTherapistHandler_ClientBrief_InvalidClientID_Returns400(t *testing.T) {
	repo := &fakeTherapistRepo{getByUserIDResp: registeredTherapist()}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, &fakeBriefGenerator{}, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients/not-a-uuid/brief", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid client id: want 400, got %d", w.Code)
	}
}

func TestTherapistHandler_ClientBrief_GenerateBriefError_Returns500(t *testing.T) {
	clientID := uuid.New()
	repo := &fakeTherapistRepo{
		getByUserIDResp: registeredTherapist(),
		getClientLinkResp: &models.ClientTherapistLink{
			ID: uuid.New(), ClientID: clientID, Status: "active",
		},
		displayName: "Alice",
	}
	gen := &fakeBriefGenerator{err: errBriefFailed}
	r := newTherapistTestRouter(t, repo, &fakeTherapistAnalysisRepo{}, gen, therapistTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/therapists/clients/"+clientID.String()+"/brief", nil)
	req.Header.Set("Authorization", "Bearer "+therapistTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("brief error: want 500, got %d", w.Code)
	}
}

var errBriefFailed = errors.New("AI unavailable")
