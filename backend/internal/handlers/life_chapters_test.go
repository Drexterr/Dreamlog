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

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeLifeChapterRepo struct {
	chapters    []*models.LifeChapter
	chapter     *models.LifeChapter
	detail      *models.ChapterDetail
	entries     []*models.WeekSummaryEntry
	createErr   error
	listErr     error
	getErr      error
	updateErr   error
	deleteErr   error
	detailErr   error
	entriesErr  error
	storeErr    error
	storedID    uuid.UUID
	storedSumm  string
}

func (f *fakeLifeChapterRepo) Create(_ context.Context, _ uuid.UUID, _ models.CreateChapterInput) (*models.LifeChapter, error) {
	return f.chapter, f.createErr
}
func (f *fakeLifeChapterRepo) List(_ context.Context, _ uuid.UUID) ([]*models.LifeChapter, error) {
	return f.chapters, f.listErr
}
func (f *fakeLifeChapterRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*models.LifeChapter, error) {
	return f.chapter, f.getErr
}
func (f *fakeLifeChapterRepo) Update(_ context.Context, _, _ uuid.UUID, _ models.UpdateChapterInput) (*models.LifeChapter, error) {
	return f.chapter, f.updateErr
}
func (f *fakeLifeChapterRepo) Delete(_ context.Context, _, _ uuid.UUID) error {
	return f.deleteErr
}
func (f *fakeLifeChapterRepo) GetDetail(_ context.Context, _, _ uuid.UUID) (*models.ChapterDetail, error) {
	return f.detail, f.detailErr
}
func (f *fakeLifeChapterRepo) GetEntriesInRange(_ context.Context, _ uuid.UUID, _ string, _ *string) ([]*models.WeekSummaryEntry, error) {
	return f.entries, f.entriesErr
}
func (f *fakeLifeChapterRepo) StoreSummary(_ context.Context, id, _ uuid.UUID, summary string) error {
	f.storedID = id
	f.storedSumm = summary
	return f.storeErr
}

type fakeChapterSummarizer struct {
	out *services.ChapterSummaryOutput
	err error
}

func (f *fakeChapterSummarizer) GenerateChapterSummary(_ context.Context, _ services.ChapterSummaryPromptInput) (*services.ChapterSummaryOutput, error) {
	return f.out, f.err
}

// ── Router & JWT helpers ──────────────────────────────────────────────────────

const chapterTestSecret = "chapter-test-jwt-secret-32-bytes!"

func chapterTestRouter(repo lifeChapterRepo, claude chapterSummarizer, plan models.Plan) (*gin.Engine, *models.User) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	testUser := &models.User{ID: uuid.New(), Email: "test@dreamlog.dev", Name: "Tester", Plan: plan}

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(chapterTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := NewLifeChapterHandler(repo, claude)
	r.GET("/chapters", h.List)
	r.POST("/chapters", h.Create)
	r.GET("/chapters/:id", h.GetByID)
	r.PUT("/chapters/:id", h.Update)
	r.DELETE("/chapters/:id", h.Delete)
	r.GET("/chapters/:id/detail", h.GetDetail)
	r.POST("/chapters/:id/summarize", h.Summarize)

	return r, testUser
}

func chapterJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-chapter-001",
		"email": "test@dreamlog.dev",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(chapterTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func sampleChapter() *models.LifeChapter {
	end := "2025-12-31"
	return &models.LifeChapter{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Title:       "A New Beginning",
		Description: "Starting fresh after the move",
		StartDate:   "2025-01-01",
		EndDate:     &end,
		Emoji:       "🌱",
		Color:       "#7C3AED",
		Summary:     "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func sampleChapterDetail() *models.ChapterDetail {
	ch := sampleChapter()
	avg := 72
	return &models.ChapterDetail{
		LifeChapter: *ch,
		EntryCount:  14,
		AvgMood:     &avg,
		TopEmotions: []string{"hopeful", "calm", "reflective"},
		MoodArc:     []models.MoodArcDay{{Date: "2025-03-01", AvgMood: 70}},
	}
}

// ── Tests: List ───────────────────────────────────────────────────────────────

func TestListChapters_Returns200WithChapters(t *testing.T) {
	repo := &fakeLifeChapterRepo{chapters: []*models.LifeChapter{sampleChapter(), sampleChapter()}}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Chapters []*models.LifeChapter `json:"chapters"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(body.Chapters))
	}
}

func TestListChapters_EmptyReturnsEmptyArray(t *testing.T) {
	repo := &fakeLifeChapterRepo{chapters: nil}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Chapters []*models.LifeChapter `json:"chapters"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Chapters == nil {
		t.Error("expected empty array, got nil")
	}
}

func TestListChapters_RepoErrorReturns500(t *testing.T) {
	repo := &fakeLifeChapterRepo{listErr: errors.New("db error")}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListChapters_MissingAuthReturns401(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: Create ─────────────────────────────────────────────────────────────

func TestCreateChapter_Returns201(t *testing.T) {
	repo := &fakeLifeChapterRepo{chapter: sampleChapter()}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	body := `{"title":"A New Beginning","start_date":"2025-01-01","color":"#7C3AED"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var ch models.LifeChapter
	if err := json.Unmarshal(w.Body.Bytes(), &ch); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ch.Title != "A New Beginning" {
		t.Errorf("unexpected title: %q", ch.Title)
	}
}

func TestCreateChapter_MissingTitleReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	body := `{"start_date":"2025-01-01"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateChapter_MissingStartDateReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	body := `{"title":"No date chapter"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateChapter_RepoErrorReturns500(t *testing.T) {
	repo := &fakeLifeChapterRepo{createErr: errors.New("db error")}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	body := `{"title":"Test","start_date":"2025-01-01"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Tests: GetByID ────────────────────────────────────────────────────────────

func TestGetChapterByID_Returns200(t *testing.T) {
	ch := sampleChapter()
	repo := &fakeLifeChapterRepo{chapter: ch}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/"+ch.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetChapterByID_NotFoundReturns404(t *testing.T) {
	repo := &fakeLifeChapterRepo{chapter: nil}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetChapterByID_InvalidIDReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetChapterByID_RepoErrorReturns500(t *testing.T) {
	repo := &fakeLifeChapterRepo{getErr: errors.New("db error")}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Tests: Update ─────────────────────────────────────────────────────────────

func TestUpdateChapter_Returns200(t *testing.T) {
	updated := sampleChapter()
	updated.Title = "Updated Title"
	repo := &fakeLifeChapterRepo{chapter: updated}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	newTitle := "Updated Title"
	bodyBytes, _ := json.Marshal(models.UpdateChapterInput{Title: &newTitle})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/chapters/"+updated.ID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ch models.LifeChapter
	if err := json.Unmarshal(w.Body.Bytes(), &ch); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ch.Title != "Updated Title" {
		t.Errorf("expected updated title, got %q", ch.Title)
	}
}

func TestUpdateChapter_NotFoundReturns404(t *testing.T) {
	repo := &fakeLifeChapterRepo{chapter: nil}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	newTitle := "Title"
	bodyBytes, _ := json.Marshal(models.UpdateChapterInput{Title: &newTitle})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/chapters/"+uuid.New().String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateChapter_InvalidIDReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/chapters/not-a-uuid", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── Tests: Delete ─────────────────────────────────────────────────────────────

func TestDeleteChapter_Returns204(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/chapters/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestDeleteChapter_InvalidIDReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/chapters/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteChapter_RepoErrorReturns500(t *testing.T) {
	repo := &fakeLifeChapterRepo{deleteErr: errors.New("db error")}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/chapters/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Tests: GetDetail ──────────────────────────────────────────────────────────

func TestGetChapterDetail_Returns200(t *testing.T) {
	detail := sampleChapterDetail()
	repo := &fakeLifeChapterRepo{detail: detail}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/"+detail.ID.String()+"/detail", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body models.ChapterDetail
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.EntryCount != 14 {
		t.Errorf("expected entry_count=14, got %d", body.EntryCount)
	}
	if len(body.TopEmotions) != 3 {
		t.Errorf("expected 3 top emotions, got %d", len(body.TopEmotions))
	}
	if len(body.MoodArc) != 1 {
		t.Errorf("expected 1 mood arc entry, got %d", len(body.MoodArc))
	}
}

func TestGetChapterDetail_NotFoundReturns404(t *testing.T) {
	repo := &fakeLifeChapterRepo{detail: nil}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/"+uuid.New().String()+"/detail", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetChapterDetail_InvalidIDReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/not-a-uuid/detail", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── Tests: Summarize ──────────────────────────────────────────────────────────

func TestSummarizeChapter_Returns200WithSummary(t *testing.T) {
	detail := sampleChapterDetail()
	repo := &fakeLifeChapterRepo{
		detail:  detail,
		entries: []*models.WeekSummaryEntry{{Summary: "A reflective day", MoodScore: 70}},
	}
	claude := &fakeChapterSummarizer{out: &services.ChapterSummaryOutput{Summary: "This was a chapter of growth."}}
	r, _ := chapterTestRouter(repo, claude, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/"+detail.ID.String()+"/summarize", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Summary != "This was a chapter of growth." {
		t.Errorf("unexpected summary: %q", body.Summary)
	}
}

func TestSummarizeChapter_StoresSummaryInRepo(t *testing.T) {
	detail := sampleChapterDetail()
	repo := &fakeLifeChapterRepo{
		detail:  detail,
		entries: []*models.WeekSummaryEntry{},
	}
	claude := &fakeChapterSummarizer{out: &services.ChapterSummaryOutput{Summary: "Stored narrative."}}
	r, _ := chapterTestRouter(repo, claude, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/"+detail.ID.String()+"/summarize", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if repo.storedSumm != "Stored narrative." {
		t.Errorf("expected StoreSummary called with 'Stored narrative.', got %q", repo.storedSumm)
	}
}

func TestSummarizeChapter_ChapterNotFoundReturns404(t *testing.T) {
	repo := &fakeLifeChapterRepo{detail: nil}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/"+uuid.New().String()+"/summarize", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSummarizeChapter_ClaudeErrorReturns500(t *testing.T) {
	detail := sampleChapterDetail()
	repo := &fakeLifeChapterRepo{
		detail:  detail,
		entries: []*models.WeekSummaryEntry{},
	}
	claude := &fakeChapterSummarizer{err: errors.New("claude timeout")}
	r, _ := chapterTestRouter(repo, claude, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/"+detail.ID.String()+"/summarize", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSummarizeChapter_InvalidIDReturns400(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/not-a-uuid/summarize", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateChapter_RepoError_Returns500(t *testing.T) {
	repo := &fakeLifeChapterRepo{updateErr: errors.New("db error")}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	newTitle := "Updated"
	bodyBytes, _ := json.Marshal(models.UpdateChapterInput{Title: &newTitle})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/chapters/"+uuid.New().String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetChapterDetail_RepoError_Returns500(t *testing.T) {
	repo := &fakeLifeChapterRepo{detailErr: errors.New("db error")}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/chapters/"+uuid.New().String()+"/detail", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSummarizeChapter_StoreSummaryError_Returns500(t *testing.T) {
	detail := sampleChapterDetail()
	repo := &fakeLifeChapterRepo{
		detail:   detail,
		entries:  []*models.WeekSummaryEntry{},
		storeErr: errors.New("db write failed"),
	}
	claude := &fakeChapterSummarizer{out: &services.ChapterSummaryOutput{Summary: "Good chapter."}}
	r, _ := chapterTestRouter(repo, claude, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/"+detail.ID.String()+"/summarize", nil)
	req.Header.Set("Authorization", "Bearer "+chapterJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d", w.Code)
	}
}

func TestSummarizeChapter_MissingAuthReturns401(t *testing.T) {
	repo := &fakeLifeChapterRepo{}
	r, _ := chapterTestRouter(repo, &fakeChapterSummarizer{}, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/chapters/"+uuid.New().String()+"/summarize", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
