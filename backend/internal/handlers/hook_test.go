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
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeEntryGetter struct {
	entry *models.Entry
}

func (f *fakeEntryGetter) GetByID(_ context.Context, id, _ uuid.UUID) (*models.Entry, error) {
	if f.entry != nil && f.entry.ID == id {
		return f.entry, nil
	}
	return nil, nil
}

type fakeAnalysisGetter struct {
	analysis *models.EntryAnalysis
}

func (f *fakeAnalysisGetter) GetByEntryID(_ context.Context, _ uuid.UUID) (*models.EntryAnalysis, error) {
	return f.analysis, nil
}

type fakeFlashbackQuerier struct {
	fb  *models.Flashback
	err error
}

func (f *fakeFlashbackQuerier) Flashback(_ context.Context, _ uuid.UUID) (*models.Flashback, error) {
	return f.fb, f.err
}

type fakeCheckinNudger struct {
	scheduledAt time.Time
	err         error
	gotMessage  string
	calls       int
}

func (f *fakeCheckinNudger) ScheduleCheckinNudge(_ context.Context, _, _ uuid.UUID, message string) (time.Time, error) {
	f.calls++
	f.gotMessage = message
	if f.err != nil {
		return time.Time{}, f.err
	}
	return f.scheduledAt, nil
}

// ── test router ───────────────────────────────────────────────────────────────

const hookTestSecret = "hook-test-jwt-secret-32-bytes!!!"

func newHookTestRouter(t *testing.T, h *HookHandler, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(hookTestSecret, "", &fakeProvisioner{user: testUser}, log))
	r.GET("/entries/flashback", h.GetFlashback)
	r.POST("/entries/:id/checkin", h.CreateCheckin)
	return r
}

func hookTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "hook-test-sub",
		"email": "hook@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(hookTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func hookTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "hook@test.com", Name: "Hook Tester"}
}

func doHookRequest(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+hookTestJWT(t))
	r.ServeHTTP(w, req)
	return w
}

// ── GET /entries/flashback ────────────────────────────────────────────────────

func TestHook_Flashback_Returns200(t *testing.T) {
	fb := &models.Flashback{
		EntryID:   uuid.New(),
		Label:     "one_year_ago",
		Date:      time.Now().AddDate(-1, 0, 0),
		Summary:   "You were nervous about the move to Bangalore.",
		MoodScore: 58,
		Topics:    []string{"relocation"},
	}
	h := NewHookHandler(&fakeEntryGetter{}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{fb: fb}, &fakeCheckinNudger{})
	r := newHookTestRouter(t, h, hookTestUser())

	w := doHookRequest(t, r, http.MethodGet, "/entries/flashback")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.Flashback
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Label != "one_year_ago" || resp.Summary != fb.Summary {
		t.Errorf("unexpected flashback payload: %+v", resp)
	}
}

func TestHook_Flashback_None_Returns404(t *testing.T) {
	h := NewHookHandler(&fakeEntryGetter{}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{fb: nil}, &fakeCheckinNudger{})
	r := newHookTestRouter(t, h, hookTestUser())

	w := doHookRequest(t, r, http.MethodGet, "/entries/flashback")

	if w.Code != http.StatusNotFound {
		t.Errorf("no flashback: want 404, got %d", w.Code)
	}
}

// ── POST /entries/:id/checkin ─────────────────────────────────────────────────

func completedEntry(userID uuid.UUID) *models.Entry {
	return &models.Entry{ID: uuid.New(), UserID: userID, Status: models.EntryStatusCompleted}
}

func TestHook_Checkin_Returns201WithScheduledAt(t *testing.T) {
	user := hookTestUser()
	entry := completedEntry(user.ID)
	analysis := &models.EntryAnalysis{EntryID: entry.ID, MorningNudge: "Notice how the interview felt."}
	nudger := &fakeCheckinNudger{scheduledAt: time.Now().Add(20 * time.Hour)}
	h := NewHookHandler(&fakeEntryGetter{entry: entry}, &fakeAnalysisGetter{analysis: analysis}, &fakeFlashbackQuerier{}, nudger)
	r := newHookTestRouter(t, h, user)

	w := doHookRequest(t, r, http.MethodPost, "/entries/"+entry.ID.String()+"/checkin")

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	if nudger.gotMessage != analysis.MorningNudge {
		t.Errorf("check-in message: want morning nudge reused, got %q", nudger.gotMessage)
	}
	var resp struct {
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ScheduledAt.IsZero() {
		t.Error("scheduled_at must be set in the response")
	}
}

func TestHook_Checkin_EntryNotFound_Returns404(t *testing.T) {
	h := NewHookHandler(&fakeEntryGetter{}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{}, &fakeCheckinNudger{})
	r := newHookTestRouter(t, h, hookTestUser())

	w := doHookRequest(t, r, http.MethodPost, "/entries/"+uuid.NewString()+"/checkin")

	if w.Code != http.StatusNotFound {
		t.Errorf("unknown entry: want 404, got %d", w.Code)
	}
}

func TestHook_Checkin_InvalidUUID_Returns400(t *testing.T) {
	h := NewHookHandler(&fakeEntryGetter{}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{}, &fakeCheckinNudger{})
	r := newHookTestRouter(t, h, hookTestUser())

	w := doHookRequest(t, r, http.MethodPost, "/entries/not-a-uuid/checkin")

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid uuid: want 400, got %d", w.Code)
	}
}

func TestHook_Checkin_EntryNotCompleted_Returns409(t *testing.T) {
	user := hookTestUser()
	entry := &models.Entry{ID: uuid.New(), UserID: user.ID, Status: models.EntryStatusPending}
	nudger := &fakeCheckinNudger{}
	h := NewHookHandler(&fakeEntryGetter{entry: entry}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{}, nudger)
	r := newHookTestRouter(t, h, user)

	w := doHookRequest(t, r, http.MethodPost, "/entries/"+entry.ID.String()+"/checkin")

	if w.Code != http.StatusConflict {
		t.Errorf("pending entry: want 409, got %d", w.Code)
	}
	if nudger.calls != 0 {
		t.Error("nudge service must not be called for a non-completed entry")
	}
}

func TestHook_Checkin_Duplicate_Returns409(t *testing.T) {
	user := hookTestUser()
	entry := completedEntry(user.ID)
	nudger := &fakeCheckinNudger{err: services.ErrCheckinExists}
	h := NewHookHandler(&fakeEntryGetter{entry: entry}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{}, nudger)
	r := newHookTestRouter(t, h, user)

	w := doHookRequest(t, r, http.MethodPost, "/entries/"+entry.ID.String()+"/checkin")

	if w.Code != http.StatusConflict {
		t.Errorf("duplicate check-in: want 409, got %d", w.Code)
	}
}

func TestHook_Checkin_NudgesDisabled_Returns409(t *testing.T) {
	user := hookTestUser()
	entry := completedEntry(user.ID)
	nudger := &fakeCheckinNudger{err: services.ErrNudgesDisabled}
	h := NewHookHandler(&fakeEntryGetter{entry: entry}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{}, nudger)
	r := newHookTestRouter(t, h, user)

	w := doHookRequest(t, r, http.MethodPost, "/entries/"+entry.ID.String()+"/checkin")

	if w.Code != http.StatusConflict {
		t.Errorf("nudges disabled: want 409, got %d", w.Code)
	}
}

func TestHook_Checkin_ServiceError_Returns500(t *testing.T) {
	user := hookTestUser()
	entry := completedEntry(user.ID)
	nudger := &fakeCheckinNudger{err: errors.New("db down")}
	h := NewHookHandler(&fakeEntryGetter{entry: entry}, &fakeAnalysisGetter{}, &fakeFlashbackQuerier{}, nudger)
	r := newHookTestRouter(t, h, user)

	w := doHookRequest(t, r, http.MethodPost, "/entries/"+entry.ID.String()+"/checkin")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("service error: want 500, got %d", w.Code)
	}
}
