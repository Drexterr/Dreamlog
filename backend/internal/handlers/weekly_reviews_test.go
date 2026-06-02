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

type fakeWeeklyReviewListRepo struct {
	latest    *models.WeeklyReview
	latestErr error
	list      []*models.WeeklyReview
	listErr   error
}

func (f *fakeWeeklyReviewListRepo) GetLatestCompleted(_ context.Context, _ uuid.UUID) (*models.WeeklyReview, error) {
	return f.latest, f.latestErr
}

func (f *fakeWeeklyReviewListRepo) ListCompleted(_ context.Context, _ uuid.UUID, _ int) ([]*models.WeeklyReview, error) {
	return f.list, f.listErr
}

// ── Test router & JWT ─────────────────────────────────────────────────────────

const weeklyReviewTestSecret = "weekly-review-jwt-secret-32-bytes"

func weeklyReviewTestRouter(repo weeklyReviewListRepo) (*gin.Engine, *models.User) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	testUser := &models.User{ID: uuid.New(), Email: "test@dreamlog.dev", Name: "Tester", Plan: models.PlanPlus}

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(weeklyReviewTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewWeeklyReviewHandler(repo)
	r.GET("/reviews/weekly", h.List)
	r.GET("/reviews/weekly/latest", h.GetLatest)
	return r, testUser
}

func weeklyReviewJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-weekly-001",
		"email": "test@dreamlog.dev",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(weeklyReviewTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestGetLatestWeeklyReview_Returns200(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeWeeklyReviewListRepo{
		latest: &models.WeeklyReview{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			WeekStart: now,
			Narrative: "A meaningful week.",
			Status:    "completed",
		},
	}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly/latest", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var rv models.WeeklyReview
	if err := json.Unmarshal(w.Body.Bytes(), &rv); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if rv.Narrative != "A meaningful week." {
		t.Errorf("unexpected narrative: %q", rv.Narrative)
	}
}

func TestGetLatestWeeklyReview_NoneReturns404(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{latest: nil}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly/latest", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetLatestWeeklyReview_RepoErrorReturns500(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{latestErr: errors.New("db error")}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly/latest", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetLatestWeeklyReview_MissingAuthReturns401(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly/latest", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListWeeklyReviews_Returns200WithReviews(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeWeeklyReviewListRepo{
		list: []*models.WeeklyReview{
			{ID: uuid.New(), WeekStart: now, Narrative: "Week one.", Status: "completed"},
			{ID: uuid.New(), WeekStart: now.AddDate(0, 0, -7), Narrative: "Week two.", Status: "completed"},
		},
	}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Reviews []*models.WeeklyReview `json:"reviews"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(body.Reviews))
	}
}

func TestListWeeklyReviews_EmptyReturnsEmptyArray(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{list: nil}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Reviews []*models.WeeklyReview `json:"reviews"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Reviews == nil {
		t.Error("expected empty array, got nil")
	}
	if len(body.Reviews) != 0 {
		t.Errorf("expected 0 reviews, got %d", len(body.Reviews))
	}
}

func TestListWeeklyReviews_RepoErrorReturns500(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{listErr: errors.New("db error")}

	r, _ := weeklyReviewTestRouter(repo)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListWeeklyReviews_MissingAuthReturns401(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{}
	r, _ := weeklyReviewTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Plan gating ────────────────────────────────────────────────────────────────

func weeklyReviewFreeUserRouter(repo weeklyReviewListRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	freeUser := &models.User{ID: uuid.New(), Email: "free@dreamlog.dev", Name: "Free", Plan: models.PlanFree}
	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(weeklyReviewTestSecret, &fakeProvisioner{user: freeUser}, log))
	h := NewWeeklyReviewHandler(repo)
	r.GET("/reviews/weekly", h.List)
	r.GET("/reviews/weekly/latest", h.GetLatest)
	return r
}

func TestGetLatestWeeklyReview_FreePlanReturns403(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{latest: &models.WeeklyReview{Narrative: "A week."}}
	r := weeklyReviewFreeUserRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly/latest", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("free plan: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListWeeklyReviews_FreePlanReturns403(t *testing.T) {
	repo := &fakeWeeklyReviewListRepo{list: []*models.WeeklyReview{{Narrative: "A week."}}}
	r := weeklyReviewFreeUserRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reviews/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+weeklyReviewJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("free plan: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
