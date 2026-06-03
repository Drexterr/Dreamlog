package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeInsightRepo struct {
	cardData    *models.InsightCardData
	cardErr     error
	shareResult *models.InsightShare
	shareErr    error
	count       int
	countErr    error
}

func (f *fakeInsightRepo) GetCardData(_ context.Context, _ uuid.UUID, _ time.Time) (*models.InsightCardData, error) {
	return f.cardData, f.cardErr
}

func (f *fakeInsightRepo) RecordShare(_ context.Context, _ uuid.UUID, _ string) (*models.InsightShare, error) {
	return f.shareResult, f.shareErr
}

func (f *fakeInsightRepo) CountByUser(_ context.Context, _ uuid.UUID) (int, error) {
	return f.count, f.countErr
}

type fakeInsightStreak struct {
	info *models.StreakInfo
	err  error
}

func (f *fakeInsightStreak) StreakInfo(_ context.Context, _ uuid.UUID) (*models.StreakInfo, error) {
	return f.info, f.err
}

// ── test router ───────────────────────────────────────────────────────────────

const insightTestSecret = "insight-test-jwt-secret-32-bytes!"

func newInsightTestRouter(t *testing.T, repo insightRepo, streak insightStreakQuerier, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(insightTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := NewInsightHandler(repo, streak)
	r.GET("/insights/card", h.GetCard)
	r.POST("/insights/share", h.RecordShare)
	return r
}

func insightTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-insight-001",
		"email": "insight@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(insightTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func insightTestUser() *models.User {
	return &models.User{
		ID:    uuid.New(),
		Email: "insight@test.com",
		Name:  "Insight User",
		Plan:  models.PlanFree,
	}
}

func sampleCardData() *models.InsightCardData {
	return &models.InsightCardData{
		WeekLabel:   "May 26 – Jun 1",
		WeekStart:   "2026-05-26",
		MoodArc:     []models.MoodArcDay{{Date: "2026-05-26", AvgMood: 72}},
		TopEmotions: []string{"hopeful", "calm"},
		Streak:      0, // set by handler from streakRepo
		EntryCount:  3,
		ShareCount:  1,
	}
}

// ── GET /insights/card ────────────────────────────────────────────────────────

func TestInsightHandler_GetCard_Returns200WithCardData(t *testing.T) {
	card := sampleCardData()
	streak := &fakeInsightStreak{info: &models.StreakInfo{CurrentStreak: 5}}
	repo := &fakeInsightRepo{cardData: card}
	r := newInsightTestRouter(t, repo, streak, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/insights/card", nil)
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.InsightCardData
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.WeekLabel != card.WeekLabel {
		t.Errorf("expected week_label=%q, got %q", card.WeekLabel, resp.WeekLabel)
	}
	if resp.EntryCount != card.EntryCount {
		t.Errorf("expected entry_count=%d, got %d", card.EntryCount, resp.EntryCount)
	}
	if resp.Streak != 5 {
		t.Errorf("expected streak=5 (from streakRepo), got %d", resp.Streak)
	}
	if len(resp.TopEmotions) != 2 {
		t.Errorf("expected 2 top emotions, got %d", len(resp.TopEmotions))
	}
}

func TestInsightHandler_GetCard_StreakError_StillReturns200(t *testing.T) {
	// If streak service fails, card should still be returned (streak=0).
	streak := &fakeInsightStreak{err: errors.New("streak unavailable")}
	repo := &fakeInsightRepo{cardData: sampleCardData()}
	r := newInsightTestRouter(t, repo, streak, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/insights/card", nil)
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even with streak error, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.InsightCardData
	json.Unmarshal(w.Body.Bytes(), &resp)
	// streak should remain 0 (default) since streakRepo failed
	if resp.Streak != 0 {
		t.Errorf("expected streak=0 on streak error, got %d", resp.Streak)
	}
}

func TestInsightHandler_GetCard_RepoError_Returns500(t *testing.T) {
	streak := &fakeInsightStreak{info: &models.StreakInfo{CurrentStreak: 3}}
	repo := &fakeInsightRepo{cardErr: errors.New("db down")}
	r := newInsightTestRouter(t, repo, streak, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/insights/card", nil)
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on repo error, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInsightHandler_GetCard_MissingAuth_Returns401(t *testing.T) {
	r := newInsightTestRouter(t, &fakeInsightRepo{cardData: sampleCardData()}, &fakeInsightStreak{}, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/insights/card", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestInsightHandler_GetCard_AvailableToFreeUsers(t *testing.T) {
	// No plan gate — free users can access the card.
	user := insightTestUser()
	user.Plan = models.PlanFree
	repo := &fakeInsightRepo{cardData: sampleCardData()}
	streak := &fakeInsightStreak{info: &models.StreakInfo{CurrentStreak: 1}}
	r := newInsightTestRouter(t, repo, streak, user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/insights/card", nil)
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("free user should access insight card, got %d", w.Code)
	}
}

// ── POST /insights/share ──────────────────────────────────────────────────────

func TestInsightHandler_RecordShare_Returns201WithTotalShares(t *testing.T) {
	shareRecord := &models.InsightShare{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		WeekStart: "2026-05-26",
		CreatedAt: time.Now(),
	}
	repo := &fakeInsightRepo{shareResult: shareRecord, count: 3}
	r := newInsightTestRouter(t, repo, &fakeInsightStreak{}, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/insights/share", strings.NewReader(`{"week_start":"2026-05-26"}`))
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.InsightShareResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TotalShares != 3 {
		t.Errorf("expected total_shares=3, got %d", resp.TotalShares)
	}
	if resp.WeekStart != "2026-05-26" {
		t.Errorf("expected week_start=2026-05-26, got %s", resp.WeekStart)
	}
}

func TestInsightHandler_RecordShare_NoBody_UsesCurrentWeek(t *testing.T) {
	shareRecord := &models.InsightShare{ID: uuid.New(), WeekStart: "2026-05-25"}
	repo := &fakeInsightRepo{shareResult: shareRecord, count: 1}
	r := newInsightTestRouter(t, repo, &fakeInsightStreak{}, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/insights/share", nil)
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 with no body, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.InsightShareResult
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.WeekStart == "" {
		t.Error("expected week_start in response")
	}
}

func TestInsightHandler_RecordShare_RepoError_Returns500(t *testing.T) {
	repo := &fakeInsightRepo{shareErr: errors.New("db write failed")}
	r := newInsightTestRouter(t, repo, &fakeInsightStreak{}, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/insights/share", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on repo error, got %d", w.Code)
	}
}

func TestInsightHandler_RecordShare_CountError_StillReturns201WithZero(t *testing.T) {
	// CountByUser failure is non-critical — return 201 with total=0.
	shareRecord := &models.InsightShare{ID: uuid.New(), WeekStart: "2026-05-26"}
	repo := &fakeInsightRepo{shareResult: shareRecord, countErr: errors.New("count failed")}
	r := newInsightTestRouter(t, repo, &fakeInsightStreak{}, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/insights/share", strings.NewReader(`{"week_start":"2026-05-26"}`))
	req.Header.Set("Authorization", "Bearer "+insightTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 even with count error, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.InsightShareResult
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.TotalShares != 0 {
		t.Errorf("expected total_shares=0 on count error, got %d", resp.TotalShares)
	}
}

func TestInsightHandler_RecordShare_MissingAuth_Returns401(t *testing.T) {
	r := newInsightTestRouter(t, &fakeInsightRepo{shareResult: &models.InsightShare{}}, &fakeInsightStreak{}, insightTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/insights/share", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
