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

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeStreakFreezerFull struct {
	useErr      error
	countResult int
	countErr    error
	usedDate    time.Time
}

func (f *fakeStreakFreezerFull) UseStreakFreeze(_ context.Context, _ uuid.UUID, d time.Time) error {
	f.usedDate = d
	return f.useErr
}

func (f *fakeStreakFreezerFull) StreakFreezeCount(_ context.Context, _ uuid.UUID) (int, error) {
	return f.countResult, f.countErr
}

const streakTestSecret = "streak-test-jwt-secret-32-bytes!"

func streakTestRouter(t *testing.T, freezer streakFreezer, aq analysisQuerier, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(streakTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewMoodHandler(aq, &fakeDeviceRegistrar{}, freezer)
	r.GET("/mood/streak", h.Streak)
	r.POST("/streak/freeze", h.UseFreeze)
	return r
}

func streakTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-streak-001",
		"email": "streak@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(streakTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func streakTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "streak@test.com", Name: "Tester", StreakFreezeCount: 1}
}

// ── UseFreeze tests ───────────────────────────────────────────────────────────

func TestUseFreeze_Returns200OnSuccess(t *testing.T) {
	freezer := &fakeStreakFreezerFull{countResult: 0}
	r := streakTestRouter(t, freezer, &fakeAnalysisQuerier{}, streakTestUser())

	body, _ := json.Marshal(map[string]string{"freeze_date": "2026-05-27"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/streak/freeze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+streakTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["freeze_date"] != "2026-05-27" {
		t.Errorf("expected freeze_date in response, got %v", resp)
	}
}

func TestUseFreeze_ParsesDateCorrectly(t *testing.T) {
	freezer := &fakeStreakFreezerFull{countResult: 1}
	r := streakTestRouter(t, freezer, &fakeAnalysisQuerier{}, streakTestUser())

	body, _ := json.Marshal(map[string]string{"freeze_date": "2026-05-20"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/streak/freeze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+streakTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	expected := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	if !freezer.usedDate.Equal(expected) {
		t.Errorf("expected frozen date %v, got %v", expected, freezer.usedDate)
	}
}

func TestUseFreeze_MissingBodyReturns400(t *testing.T) {
	freezer := &fakeStreakFreezerFull{}
	r := streakTestRouter(t, freezer, &fakeAnalysisQuerier{}, streakTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/streak/freeze", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+streakTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUseFreeze_InvalidDateFormatReturns400(t *testing.T) {
	freezer := &fakeStreakFreezerFull{}
	r := streakTestRouter(t, freezer, &fakeAnalysisQuerier{}, streakTestUser())

	body, _ := json.Marshal(map[string]string{"freeze_date": "27/05/2026"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/streak/freeze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+streakTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUseFreeze_NoFreezesRemainingReturns409(t *testing.T) {
	freezer := &fakeStreakFreezerFull{useErr: errors.New("no streak freezes remaining")}
	r := streakTestRouter(t, freezer, &fakeAnalysisQuerier{}, streakTestUser())

	body, _ := json.Marshal(map[string]string{"freeze_date": "2026-05-27"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/streak/freeze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+streakTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestUseFreeze_MissingAuthReturns401(t *testing.T) {
	freezer := &fakeStreakFreezerFull{}
	r := streakTestRouter(t, freezer, &fakeAnalysisQuerier{}, streakTestUser())

	body, _ := json.Marshal(map[string]string{"freeze_date": "2026-05-27"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/streak/freeze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Streak endpoint tests (freeze_count included) ─────────────────────────────

func TestGetStreak_IncludesFreezeCount(t *testing.T) {
	freezer := &fakeStreakFreezerFull{countResult: 2}
	aq := &fakeAnalysisQuerier{
		streakResp: &models.StreakInfo{CurrentStreak: 5, LongestStreak: 10, TotalDays: 20},
	}
	r := streakTestRouter(t, freezer, aq, streakTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mood/streak", nil)
	req.Header.Set("Authorization", "Bearer "+streakTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var info models.StreakInfo
	json.Unmarshal(w.Body.Bytes(), &info)
	if info.FreezeCount != 2 {
		t.Errorf("expected freeze_count 2, got %d", info.FreezeCount)
	}
	if info.CurrentStreak != 5 {
		t.Errorf("expected current_streak 5, got %d", info.CurrentStreak)
	}
}

// ── Milestone model tests ─────────────────────────────────────────────────────

func TestNextStreakMilestone(t *testing.T) {
	cases := []struct {
		current int
		want    int
	}{
		{0, 7},
		{6, 7},
		{7, 21},
		{20, 21},
		{21, 50},
		{49, 50},
		{50, 100},
		{99, 100},
		{100, 0},
		{200, 0},
	}
	for _, tc := range cases {
		got := models.NextStreakMilestone(tc.current)
		if got != tc.want {
			t.Errorf("NextStreakMilestone(%d) = %d, want %d", tc.current, got, tc.want)
		}
	}
}

func TestIsStreakMilestone(t *testing.T) {
	milestones := []int{7, 21, 50, 100}
	for _, m := range milestones {
		if !models.IsStreakMilestone(m) {
			t.Errorf("expected %d to be a milestone", m)
		}
	}
	for _, n := range []int{0, 1, 6, 8, 20, 22, 49, 51, 99, 101} {
		if models.IsStreakMilestone(n) {
			t.Errorf("expected %d to NOT be a milestone", n)
		}
	}
}
