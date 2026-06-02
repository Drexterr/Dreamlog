package handlers

import (
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

// ── Fake repo ────────────────────────────────────────────────────────────────

type fakeAnnualReviewListRepo struct {
	latest    *models.AnnualReview
	latestErr error
	list      []*models.AnnualReview
	listErr   error
}

func (f *fakeAnnualReviewListRepo) GetLatestCompleted(_ context.Context, _ uuid.UUID) (*models.AnnualReview, error) {
	return f.latest, f.latestErr
}

func (f *fakeAnnualReviewListRepo) ListCompleted(_ context.Context, _ uuid.UUID) ([]*models.AnnualReview, error) {
	return f.list, f.listErr
}

// ── Test router & JWT ─────────────────────────────────────────────────────────

const annualReviewTestSecret = "annual-review-jwt-secret-32-bytes!"

func annualReviewTestRouter(repo annualReviewListRepo, plan models.Plan) (*gin.Engine, *models.User) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	testUser := &models.User{ID: uuid.New(), Email: "test@dreamlog.dev", Name: "Tester", Plan: plan}

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(annualReviewTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewAnnualReviewHandler(repo)
	r.GET("/reviews/annual", h.List)
	r.GET("/reviews/annual/latest", h.GetLatest)
	return r, testUser
}

func annualReviewJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-annual-001",
		"email": "test@dreamlog.dev",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(annualReviewTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func sampleAnnualReview(year int) *models.AnnualReview {
	avg := 72
	return &models.AnnualReview{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Year:        year,
		Narrative:   "A year of quiet growth.",
		TopEmotions: []string{"hopeful", "calm", "uncertain", "warm", "reflective"},
		TopTopics:   []string{"work", "family", "rest", "creativity", "health"},
		MoodArc: []models.MonthlyMoodArcDay{
			{Month: "2025-01", AvgMood: 65, EntryCount: 4},
			{Month: "2025-06", AvgMood: 78, EntryCount: 6},
		},
		EntryCount:  42,
		AvgMood:     &avg,
		Status:      models.AnnualReviewStatusCompleted,
		ScheduledAt: time.Now().Add(-24 * time.Hour),
	}
}

// ── Tests: GetLatest ──────────────────────────────────────────────────────────

func TestGetLatestAnnualReview_Returns200(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{latest: sampleAnnualReview(2025)}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual/latest", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var rv models.AnnualReview
	if err := json.Unmarshal(w.Body.Bytes(), &rv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rv.Year != 2025 {
		t.Errorf("expected year=2025, got %d", rv.Year)
	}
	if rv.Narrative != "A year of quiet growth." {
		t.Errorf("unexpected narrative: %q", rv.Narrative)
	}
	if len(rv.TopEmotions) != 5 {
		t.Errorf("expected 5 top_emotions, got %d", len(rv.TopEmotions))
	}
	if len(rv.TopTopics) != 5 {
		t.Errorf("expected 5 top_topics, got %d", len(rv.TopTopics))
	}
	if len(rv.MoodArc) != 2 {
		t.Errorf("expected 2 mood_arc entries, got %d", len(rv.MoodArc))
	}
}

func TestGetLatestAnnualReview_NoneReturns404(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{latest: nil}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual/latest", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetLatestAnnualReview_RepoErrorReturns500(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{latestErr: errors.New("db error")}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual/latest", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetLatestAnnualReview_FreePlanReturns403(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{latest: sampleAnnualReview(2025)}
	r, _ := annualReviewTestRouter(repo, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual/latest", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestGetLatestAnnualReview_MissingAuthReturns401(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual/latest", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: List ───────────────────────────────────────────────────────────────

func TestListAnnualReviews_Returns200WithReviews(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{
		list: []*models.AnnualReview{
			sampleAnnualReview(2025),
			sampleAnnualReview(2024),
		},
	}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Reviews []*models.AnnualReview `json:"reviews"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(body.Reviews))
	}
	if body.Reviews[0].Year != 2025 {
		t.Errorf("expected newest first (2025), got %d", body.Reviews[0].Year)
	}
}

func TestListAnnualReviews_EmptyReturnsEmptyArray(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{list: nil}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Reviews []*models.AnnualReview `json:"reviews"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Reviews == nil {
		t.Error("expected empty array, got nil")
	}
	if len(body.Reviews) != 0 {
		t.Errorf("expected 0 reviews, got %d", len(body.Reviews))
	}
}

func TestListAnnualReviews_RepoErrorReturns500(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{listErr: errors.New("db error")}
	r, _ := annualReviewTestRouter(repo, models.PlanPlus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListAnnualReviews_FreePlanReturns403(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{}
	r, _ := annualReviewTestRouter(repo, models.PlanFree)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestListAnnualReviews_ProPlanReturns200(t *testing.T) {
	repo := &fakeAnnualReviewListRepo{list: []*models.AnnualReview{sampleAnnualReview(2025)}}
	r, _ := annualReviewTestRouter(repo, models.PlanPro)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/annual", nil)
	req.Header.Set("Authorization", "Bearer "+annualReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for Pro plan, got %d", w.Code)
	}
}
