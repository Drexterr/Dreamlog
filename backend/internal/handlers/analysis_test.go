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
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake entryQuerier ─────────────────────────────────────────────────────────

type fakeEntryQuerier struct {
	getByIDResp    *models.Entry
	getByIDErr     error
	listResp       []*models.Entry
	listTotal      int
	listErr        error
	searchResp     []*models.Entry
	searchErr      error
}

func (f *fakeEntryQuerier) GetByID(_ context.Context, _, _ uuid.UUID) (*models.Entry, error) {
	return f.getByIDResp, f.getByIDErr
}

func (f *fakeEntryQuerier) List(_ context.Context, _ repositories.ListEntriesOpts) ([]*models.Entry, int, error) {
	return f.listResp, f.listTotal, f.listErr
}

func (f *fakeEntryQuerier) SearchEntries(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.Entry, error) {
	return f.searchResp, f.searchErr
}

// ── fake analysisQuerier ──────────────────────────────────────────────────────

type fakeAnalysisQuerier struct {
	getByEntryIDResp  *models.EntryAnalysis
	getByEntryIDErr   error
	moodResp          []*models.DailyMood
	moodErr           error
	streakResp        *models.StreakInfo
	streakErr         error
	moodHistoryResp   *models.MoodHistoryResponse
	moodHistoryErr    error
	patternResp       *models.PatternRadarResponse
	patternErr        error
}

func (f *fakeAnalysisQuerier) GetByEntryID(_ context.Context, _ uuid.UUID) (*models.EntryAnalysis, error) {
	return f.getByEntryIDResp, f.getByEntryIDErr
}

func (f *fakeAnalysisQuerier) MoodLast7Days(_ context.Context, _ uuid.UUID) ([]*models.DailyMood, error) {
	return f.moodResp, f.moodErr
}

func (f *fakeAnalysisQuerier) StreakInfo(_ context.Context, _ uuid.UUID) (*models.StreakInfo, error) {
	return f.streakResp, f.streakErr
}

func (f *fakeAnalysisQuerier) MoodHistory(_ context.Context, _ uuid.UUID, _ int) (*models.MoodHistoryResponse, error) {
	if f.moodHistoryErr != nil {
		return nil, f.moodHistoryErr
	}
	if f.moodHistoryResp != nil {
		return f.moodHistoryResp, nil
	}
	return &models.MoodHistoryResponse{Days: []*models.DailyMood{}}, nil
}

func (f *fakeAnalysisQuerier) EmotionPatterns(_ context.Context, _ uuid.UUID, _ int) (*models.PatternRadarResponse, error) {
	if f.patternErr != nil {
		return nil, f.patternErr
	}
	if f.patternResp != nil {
		return f.patternResp, nil
	}
	return &models.PatternRadarResponse{Emotions: []models.EmotionPattern{}}, nil
}

// ── fake deviceRegistrar ──────────────────────────────────────────────────────

type fakeDeviceRegistrar struct{ upsertErr error }

func (f *fakeDeviceRegistrar) UpsertDevice(_ context.Context, _ uuid.UUID, _, _ string) error {
	return f.upsertErr
}

// ── router builders ───────────────────────────────────────────────────────────

const analysisTestSecret = "analysis-test-jwt-secret-32!!!!!"

func newAnalysisTestRouter(
	t *testing.T,
	eq entryQuerier,
	aq analysisQuerier,
	testUser *models.User,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(analysisTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewAnalysisHandler(eq, aq, nil)
	r.GET("/entries/:id/analysis", h.GetAnalysis)
	r.GET("/timeline", h.GetTimeline)
	r.GET("/entries/search", h.Search)
	return r
}

type fakeStreakFreezer struct{}

func (f *fakeStreakFreezer) UseStreakFreeze(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (f *fakeStreakFreezer) StreakFreezeCount(_ context.Context, _ uuid.UUID) (int, error)    { return 1, nil }

func newMoodTestRouter(t *testing.T, aq analysisQuerier, dr deviceRegistrar, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(analysisTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewMoodHandler(aq, dr, &fakeStreakFreezer{})
	r.GET("/mood/weekly", h.WeeklyMood)
	r.GET("/mood/streak", h.Streak)
	r.GET("/mood/history", h.MoodHistory)
	r.GET("/mood/patterns", h.PatternRadar)
	r.POST("/devices", h.RegisterDevice)
	return r
}

func analysisTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-analysis-001",
		"email": "analysis@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(analysisTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func analysisTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "analysis@test.com", Name: "Analysis User", Plan: models.PlanPlus}
}

// ── GetAnalysis tests ─────────────────────────────────────────────────────────

func TestAnalysisHandler_GetAnalysis_CompletedEntry_Returns200(t *testing.T) {
	entryID := uuid.New()
	eq := &fakeEntryQuerier{getByIDResp: &models.Entry{
		ID:     entryID,
		Status: models.EntryStatusCompleted,
	}}
	aq := &fakeAnalysisQuerier{getByEntryIDResp: &models.EntryAnalysis{
		ID:         uuid.New(),
		EntryID:    entryID,
		MoodScore:  72,
		Reflection: "A thoughtful reflection. What matters most to you today?",
		IsCrisis:   false,
	}}

	r := newAnalysisTestRouter(t, eq, aq, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+entryID.String()+"/analysis", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("completed entry analysis: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.EntryAnalysis
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.MoodScore != 72 {
		t.Errorf("mood_score: want 72, got %d", resp.MoodScore)
	}
	if resp.Reflection == "" {
		t.Error("reflection must not be empty")
	}
}

func TestAnalysisHandler_GetAnalysis_EntryNotFound_Returns404(t *testing.T) {
	eq := &fakeEntryQuerier{getByIDResp: nil}
	aq := &fakeAnalysisQuerier{}

	r := newAnalysisTestRouter(t, eq, aq, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+uuid.New().String()+"/analysis", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("entry not found: want 404, got %d", w.Code)
	}
}

func TestAnalysisHandler_GetAnalysis_AnalysisNotFound_Returns404(t *testing.T) {
	entryID := uuid.New()
	eq := &fakeEntryQuerier{getByIDResp: &models.Entry{
		ID:     entryID,
		Status: models.EntryStatusCompleted,
	}}
	// Analysis does not exist yet (entry is still processing or failed).
	aq := &fakeAnalysisQuerier{getByEntryIDResp: nil}

	r := newAnalysisTestRouter(t, eq, aq, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+entryID.String()+"/analysis", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("analysis not found: want 404, got %d", w.Code)
	}
}

func TestAnalysisHandler_GetAnalysis_InvalidEntryID_Returns400(t *testing.T) {
	r := newAnalysisTestRouter(t, &fakeEntryQuerier{}, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/not-a-uuid/analysis", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid id: want 400, got %d", w.Code)
	}
}

func TestAnalysisHandler_GetAnalysis_MissingAuth_Returns401(t *testing.T) {
	r := newAnalysisTestRouter(t, &fakeEntryQuerier{}, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+uuid.New().String()+"/analysis", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── Search tests ──────────────────────────────────────────────────────────────

func TestAnalysisHandler_Search_ValidQuery_Returns200(t *testing.T) {
	tr := "I felt anxious at work today."
	eq := &fakeEntryQuerier{
		searchResp: []*models.Entry{
			{ID: uuid.New(), Status: models.EntryStatusCompleted, Transcript: &tr},
		},
	}
	r := newAnalysisTestRouter(t, eq, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/search?q=anxiety", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("search: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatal("response must have entries array")
	}
	if len(entries) != 1 {
		t.Errorf("want 1 search result, got %d", len(entries))
	}
}

func TestAnalysisHandler_Search_EmptyQuery_Returns400(t *testing.T) {
	r := newAnalysisTestRouter(t, &fakeEntryQuerier{}, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/search?q=", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("empty query: want 400, got %d", w.Code)
	}
}

func TestAnalysisHandler_Search_NoResults_ReturnsEmptyArray(t *testing.T) {
	eq := &fakeEntryQuerier{searchResp: nil}
	r := newAnalysisTestRouter(t, eq, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/search?q=xyzunknown", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("no results: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatal("response must have entries field")
	}
	if len(entries) != 0 {
		t.Errorf("no results: want empty array, got %d items", len(entries))
	}
}

// ── WeeklyMood tests ──────────────────────────────────────────────────────────

func TestMoodHandler_WeeklyMood_Returns200(t *testing.T) {
	aq := &fakeAnalysisQuerier{
		moodResp: []*models.DailyMood{
			{Day: "2026-05-21", AvgMood: 65, EntryCount: 2},
			{Day: "2026-05-22", AvgMood: 70, EntryCount: 1},
		},
	}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("weekly mood: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	days, ok := resp["days"].([]any)
	if !ok {
		t.Fatal("response must have 'days' array")
	}
	if len(days) != 2 {
		t.Errorf("want 2 days, got %d", len(days))
	}
}

func TestMoodHandler_WeeklyMood_EmptyData_ReturnsEmptyArray(t *testing.T) {
	aq := &fakeAnalysisQuerier{moodResp: nil}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("empty mood: want 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	days, ok := resp["days"].([]any)
	if !ok {
		t.Fatal("response must have 'days' field")
	}
	if len(days) != 0 {
		t.Errorf("empty data: want empty array, got %d items", len(days))
	}
}

func TestMoodHandler_WeeklyMood_MissingAuth_Returns401(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/weekly", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── Streak tests ──────────────────────────────────────────────────────────────

func TestMoodHandler_Streak_Returns200(t *testing.T) {
	aq := &fakeAnalysisQuerier{
		streakResp: &models.StreakInfo{
			CurrentStreak: 5,
			LongestStreak: 21,
			TotalDays:     34,
		},
	}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/streak", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("streak: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.StreakInfo
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CurrentStreak != 5 {
		t.Errorf("current_streak: want 5, got %d", resp.CurrentStreak)
	}
	if resp.LongestStreak != 21 {
		t.Errorf("longest_streak: want 21, got %d", resp.LongestStreak)
	}
}

// ── MoodHistory tests ──────────────────────────────────────────────────────────

func TestMoodHandler_MoodHistory_DefaultRange_Returns200(t *testing.T) {
	avg := 70
	prev := 62
	delta := 8
	aq := &fakeAnalysisQuerier{
		moodHistoryResp: &models.MoodHistoryResponse{
			Days:        []*models.DailyMood{{Day: "2026-04-28", AvgMood: 70, EntryCount: 2}},
			Range:       "30d",
			AvgMood:     &avg,
			PrevAvgMood: &prev,
			MoodDelta:   &delta,
			TopEmotions: []string{"hopeful", "anxious"},
			EntryCount:  14,
		},
	}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/history", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("default range: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.MoodHistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Range != "30d" {
		t.Errorf("range: want 30d, got %s", resp.Range)
	}
	if resp.AvgMood == nil || *resp.AvgMood != 70 {
		t.Errorf("avg_mood: want 70, got %v", resp.AvgMood)
	}
	if resp.MoodDelta == nil || *resp.MoodDelta != 8 {
		t.Errorf("mood_delta: want 8, got %v", resp.MoodDelta)
	}
	if len(resp.TopEmotions) != 2 {
		t.Errorf("top_emotions: want 2, got %d", len(resp.TopEmotions))
	}
}

func TestMoodHandler_MoodHistory_ExplicitRanges_Returns200(t *testing.T) {
	cases := []struct{ range_ string }{
		{"30d"},
		{"90d"},
		{"365d"},
	}
	for _, tc := range cases {
		t.Run(tc.range_, func(t *testing.T) {
			aq := &fakeAnalysisQuerier{}
			r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/mood/history?range="+tc.range_, nil)
			req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("range %s: want 200, got %d: %s", tc.range_, w.Code, w.Body.String())
			}
			var resp models.MoodHistoryResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Range != tc.range_ {
				t.Errorf("range: want %s, got %s", tc.range_, resp.Range)
			}
		})
	}
}

func TestMoodHandler_MoodHistory_InvalidRange_Returns400(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/history?range=7d", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid range: want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMoodHandler_MoodHistory_MissingAuth_Returns401(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── PatternRadar tests ────────────────────────────────────────────────────────

func TestMoodHandler_PatternRadar_DefaultRange_Returns200WithEmotions(t *testing.T) {
	aq := &fakeAnalysisQuerier{
		patternResp: &models.PatternRadarResponse{
			Range: "30d",
			Emotions: []models.EmotionPattern{
				{Emotion: "hopeful", Frequency: 8, AvgIntensity: 0.72, Score: 1.0},
				{Emotion: "anxious", Frequency: 5, AvgIntensity: 0.65, Score: 0.58},
				{Emotion: "calm", Frequency: 3, AvgIntensity: 0.55, Score: 0.30},
			},
			TotalEntries: 12,
			MoodDistribution: models.MoodDistribution{High: 6, Neutral: 5, Low: 1},
		},
	}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/patterns", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PatternRadarResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Range != "30d" {
		t.Errorf("range: want 30d, got %s", resp.Range)
	}
	if len(resp.Emotions) != 3 {
		t.Errorf("emotions: want 3, got %d", len(resp.Emotions))
	}
	if resp.Emotions[0].Emotion != "hopeful" {
		t.Errorf("first emotion: want hopeful, got %s", resp.Emotions[0].Emotion)
	}
	if resp.Emotions[0].Score != 1.0 {
		t.Errorf("top score: want 1.0, got %f", resp.Emotions[0].Score)
	}
	if resp.TotalEntries != 12 {
		t.Errorf("total_entries: want 12, got %d", resp.TotalEntries)
	}
	if resp.MoodDistribution.High != 6 {
		t.Errorf("mood_distribution.high: want 6, got %d", resp.MoodDistribution.High)
	}
}

func TestMoodHandler_PatternRadar_ExplicitRanges_Returns200(t *testing.T) {
	for _, rng := range []string{"30d", "90d", "365d"} {
		t.Run(rng, func(t *testing.T) {
			r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/mood/patterns?range="+rng, nil)
			req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("range %s: want 200, got %d: %s", rng, w.Code, w.Body.String())
			}
			var resp models.PatternRadarResponse
			json.NewDecoder(w.Body).Decode(&resp)
			if resp.Range != rng {
				t.Errorf("range field: want %s, got %s", rng, resp.Range)
			}
		})
	}
}

func TestMoodHandler_PatternRadar_InvalidRange_Returns400(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/patterns?range=7d", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid range: want 400, got %d", w.Code)
	}
}

func TestMoodHandler_PatternRadar_NoData_ReturnsEmptyEmotions(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/patterns", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("no data: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PatternRadarResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Emotions == nil {
		t.Error("emotions: must be empty array, not null")
	}
	if len(resp.Emotions) != 0 {
		t.Errorf("emotions: want 0, got %d", len(resp.Emotions))
	}
}

func TestMoodHandler_PatternRadar_RepoError_Returns500(t *testing.T) {
	aq := &fakeAnalysisQuerier{patternErr: errors.New("db error")}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/patterns", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("repo error: want 500, got %d", w.Code)
	}
}

func TestMoodHandler_PatternRadar_MissingAuth_Returns401(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/patterns", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestMoodHandler_PatternRadar_FreePlanAccess_Returns200(t *testing.T) {
	// Pattern radar is available to all plans.
	user := &models.User{ID: uuid.New(), Email: "free@test.com", Plan: models.PlanFree}
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, user)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/patterns", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("free plan should access patterns, got %d", w.Code)
	}
}

// ── GetTimeline tests ─────────────────────────────────────────────────────────

func TestAnalysisHandler_GetTimeline_Returns200WithEntries(t *testing.T) {
	tr := "Busy week."
	eq := &fakeEntryQuerier{
		listResp: []*models.Entry{
			{ID: uuid.New(), Status: models.EntryStatusCompleted, Transcript: &tr},
		},
		listTotal: 1,
	}
	aq := &fakeAnalysisQuerier{
		getByEntryIDResp: &models.EntryAnalysis{MoodScore: 72, Reflection: "Great."},
	}
	r := newAnalysisTestRouter(t, eq, aq, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/timeline", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("timeline: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["entries"] == nil {
		t.Error("response must have entries field")
	}
	if resp["total"] == nil {
		t.Error("response must have total field")
	}
	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatal("entries must be an array")
	}
	if len(entries) != 1 {
		t.Errorf("want 1 entry in timeline, got %d", len(entries))
	}
}

func TestAnalysisHandler_GetTimeline_EmptyReturnsEmptyEntries(t *testing.T) {
	eq := &fakeEntryQuerier{listResp: nil, listTotal: 0}
	r := newAnalysisTestRouter(t, eq, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/timeline", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("empty timeline: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatal("entries must be an array")
	}
	if len(entries) != 0 {
		t.Errorf("empty timeline: want 0 entries, got %d", len(entries))
	}
}

func TestAnalysisHandler_GetTimeline_MissingAuth_Returns401(t *testing.T) {
	r := newAnalysisTestRouter(t, &fakeEntryQuerier{}, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/timeline", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestAnalysisHandler_GetTimeline_RepoError_Returns500(t *testing.T) {
	eq := &fakeEntryQuerier{listErr: errors.New("db error")}
	r := newAnalysisTestRouter(t, eq, &fakeAnalysisQuerier{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/timeline", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("repo error: want 500, got %d", w.Code)
	}
}

// ── Additional mood/analysis edge cases ───────────────────────────────────────

func TestMoodHandler_MoodHistory_FreePlanReturns403(t *testing.T) {
	freeUser := &models.User{ID: uuid.New(), Email: "free@test.com", Plan: models.PlanFree}
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, freeUser)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/history", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("free plan mood history: want 403, got %d", w.Code)
	}
}

func TestMoodHandler_WeeklyMood_RepoError_Returns500(t *testing.T) {
	aq := &fakeAnalysisQuerier{moodErr: errors.New("db error")}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/weekly", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("weekly mood repo error: want 500, got %d", w.Code)
	}
}

func TestMoodHandler_Streak_MissingAuth_Returns401(t *testing.T) {
	r := newMoodTestRouter(t, &fakeAnalysisQuerier{}, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/streak", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestMoodHandler_Streak_RepoError_Returns500(t *testing.T) {
	aq := &fakeAnalysisQuerier{streakErr: errors.New("db error")}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/streak", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("streak repo error: want 500, got %d", w.Code)
	}
}

func TestMoodHandler_MoodHistory_NoData_ReturnsNullAvg(t *testing.T) {
	aq := &fakeAnalysisQuerier{
		moodHistoryResp: &models.MoodHistoryResponse{
			Days:        []*models.DailyMood{},
			Range:       "90d",
			AvgMood:     nil,
			PrevAvgMood: nil,
			MoodDelta:   nil,
			TopEmotions: []string{},
			EntryCount:  0,
		},
	}
	r := newMoodTestRouter(t, aq, &fakeDeviceRegistrar{}, analysisTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/history?range=90d", nil)
	req.Header.Set("Authorization", "Bearer "+analysisTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("no data: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["avg_mood"] != nil {
		t.Errorf("avg_mood: want null when no data, got %v", body["avg_mood"])
	}
	if body["mood_delta"] != nil {
		t.Errorf("mood_delta: want null when no data, got %v", body["mood_delta"])
	}
}
