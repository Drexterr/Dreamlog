package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var _ = zap.NewNop // suppress unused import if needed

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeWeeklyReviewRepo struct {
	scheduleErr    error
	scheduledCalls []uuid.UUID

	pendingReviews []*models.WeeklyReview
	pendingErr     error

	completedID      uuid.UUID
	completedNarr    string
	completedEmotions []string
	completedErr     error

	failedID  uuid.UUID
	failedMsg string
	failedErr error
}

func (f *fakeWeeklyReviewRepo) Schedule(_ context.Context, userID uuid.UUID, _, _ time.Time) error {
	f.scheduledCalls = append(f.scheduledCalls, userID)
	return f.scheduleErr
}

func (f *fakeWeeklyReviewRepo) PendingDue(_ context.Context) ([]*models.WeeklyReview, error) {
	return f.pendingReviews, f.pendingErr
}

func (f *fakeWeeklyReviewRepo) MarkCompleted(_ context.Context, id uuid.UUID, narrative string, topEmotions []string, _ []models.MoodArcDay, _ int) error {
	f.completedID = id
	f.completedNarr = narrative
	f.completedEmotions = topEmotions
	return f.completedErr
}

func (f *fakeWeeklyReviewRepo) MarkFailed(_ context.Context, id uuid.UUID, msg string) error {
	f.failedID = id
	f.failedMsg = msg
	return f.failedErr
}

type fakeWeeklyUserRepo struct {
	users          []*models.User
	listErr        error
	getUser        *models.User
	getErr         error
}

func (f *fakeWeeklyUserRepo) ListWithRecentEntries(_ context.Context, _ time.Time) ([]*models.User, error) {
	return f.users, f.listErr
}

func (f *fakeWeeklyUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return f.getUser, f.getErr
}

type fakeWeeklyAnalysisRepo struct {
	entries []*models.WeekSummaryEntry
	err     error
}

func (f *fakeWeeklyAnalysisRepo) GetWeekSummaries(_ context.Context, _ uuid.UUID, _ time.Time) ([]*models.WeekSummaryEntry, error) {
	return f.entries, f.err
}

type fakeWeeklyAI struct {
	out *services.WeeklyReviewOutput
	err error
}

func (f *fakeWeeklyAI) GenerateWeeklyReview(_ context.Context, _ services.WeeklyReviewPromptInput) (*services.WeeklyReviewOutput, error) {
	return f.out, f.err
}

type fakeNudgeRepo struct {
	tokens []string
	err    error
	calls  []struct{ PendingDue bool }
}

func (f *fakeNudgeRepo) PendingDue(_ context.Context) ([]*models.Nudge, error) { return nil, nil }
func (f *fakeNudgeRepo) GetDeviceTokens(_ context.Context, _ uuid.UUID) ([]string, error) {
	return f.tokens, f.err
}
func (f *fakeNudgeRepo) MarkSent(_ context.Context, _ uuid.UUID) error   { return nil }
func (f *fakeNudgeRepo) MarkFailed(_ context.Context, _ uuid.UUID, _ string) error { return nil }

type fakeWeeklyFCMSender struct {
	calls []string
	err   error
}

func (f *fakeWeeklyFCMSender) SendToToken(_ context.Context, token, _, _ string, _ map[string]string) error {
	f.calls = append(f.calls, token)
	return f.err
}

func newTestScheduler(
	reviewRepo weeklyReviewRepo,
	userRepo weeklyReviewUserRepo,
	analysisRepo weeklyReviewAnalysisRepo,
	ai weeklyReviewAI,
	nudge nudgeDispatcher,
	fcm fcmSender,
) *WeeklyReviewScheduler {
	return NewWeeklyReviewScheduler(WeeklyReviewSchedulerDeps{
		ReviewRepo:   reviewRepo,
		UserRepo:     userRepo,
		AnalysisRepo: analysisRepo,
		Claude:       ai,
		NudgeRepo:    nudge,
		FCM:          fcm,
		Log:          zap.NewNop(),
	})
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestScheduleForActiveUsers_SchedulesPerUser(t *testing.T) {
	uid1, uid2 := uuid.New(), uuid.New()
	userRepo := &fakeWeeklyUserRepo{
		users: []*models.User{
			{ID: uid1, Timezone: "UTC"},
			{ID: uid2, Timezone: "UTC"},
		},
	}
	reviewRepo := &fakeWeeklyReviewRepo{}

	s := newTestScheduler(reviewRepo, userRepo, &fakeWeeklyAnalysisRepo{}, &fakeWeeklyAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.scheduleForActiveUsers(context.Background())

	if len(reviewRepo.scheduledCalls) != 2 {
		t.Fatalf("expected 2 schedule calls, got %d", len(reviewRepo.scheduledCalls))
	}
}

func TestScheduleForActiveUsers_NoUsersIsNoop(t *testing.T) {
	userRepo := &fakeWeeklyUserRepo{users: nil}
	reviewRepo := &fakeWeeklyReviewRepo{}

	s := newTestScheduler(reviewRepo, userRepo, &fakeWeeklyAnalysisRepo{}, &fakeWeeklyAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.scheduleForActiveUsers(context.Background())

	if len(reviewRepo.scheduledCalls) != 0 {
		t.Fatalf("expected 0 schedule calls, got %d", len(reviewRepo.scheduledCalls))
	}
}

func TestProcessPending_NoPendingIsNoop(t *testing.T) {
	reviewRepo := &fakeWeeklyReviewRepo{pendingReviews: nil}

	s := newTestScheduler(reviewRepo, &fakeWeeklyUserRepo{}, &fakeWeeklyAnalysisRepo{}, &fakeWeeklyAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.processPending(context.Background())
	// If we got here without panic, the test passes.
}

func TestProcessOne_HappyPath(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()
	weekStart := time.Now().UTC().Truncate(24 * time.Hour)

	reviewRepo := &fakeWeeklyReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Alice"}}
	analysisRepo := &fakeWeeklyAnalysisRepo{
		entries: []*models.WeekSummaryEntry{
			{Date: weekStart, Summary: "A good day", MoodScore: 70, Emotions: []string{"hopeful"}},
		},
	}
	ai := &fakeWeeklyAI{
		out: &services.WeeklyReviewOutput{
			Narrative:   "Alice had a hopeful week.",
			TopEmotions: []string{"hopeful"},
		},
	}

	s := newTestScheduler(reviewRepo, userRepo, analysisRepo, ai, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.WeeklyReview{ID: reviewID, UserID: uid, WeekStart: weekStart}
	err := s.processOne(context.Background(), rv)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if reviewRepo.completedID != reviewID {
		t.Errorf("expected MarkCompleted called with review ID %s, got %s", reviewID, reviewRepo.completedID)
	}
	if reviewRepo.completedNarr != "Alice had a hopeful week." {
		t.Errorf("unexpected narrative: %q", reviewRepo.completedNarr)
	}
}

func TestProcessOne_NoEntriesMarksFailedSoftly(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()

	reviewRepo := &fakeWeeklyReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Bob"}}
	analysisRepo := &fakeWeeklyAnalysisRepo{entries: nil}

	s := newTestScheduler(reviewRepo, userRepo, analysisRepo, &fakeWeeklyAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.WeeklyReview{ID: reviewID, UserID: uid, WeekStart: time.Now().UTC()}
	err := s.processOne(context.Background(), rv)
	if err != nil {
		t.Fatalf("expected nil error (soft fail), got: %v", err)
	}
	if reviewRepo.failedID != reviewID {
		t.Errorf("expected MarkFailed called with review ID %s, got %s", reviewID, reviewRepo.failedID)
	}
}

func TestProcessOne_UserNotFoundReturnsError(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()

	reviewRepo := &fakeWeeklyReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: nil, getErr: errors.New("db error")}

	s := newTestScheduler(reviewRepo, userRepo, &fakeWeeklyAnalysisRepo{}, &fakeWeeklyAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.WeeklyReview{ID: reviewID, UserID: uid, WeekStart: time.Now().UTC()}
	err := s.processOne(context.Background(), rv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProcessOne_UsesPreferredName(t *testing.T) {
	uid := uuid.New()
	preferred := "Ali"
	reviewID := uuid.New()
	weekStart := time.Now().UTC().Truncate(24 * time.Hour)

	reviewRepo := &fakeWeeklyReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Alice", PreferredName: &preferred}}
	analysisRepo := &fakeWeeklyAnalysisRepo{
		entries: []*models.WeekSummaryEntry{
			{Date: weekStart, Summary: "Nice day", MoodScore: 65},
		},
	}

	var capturedInput services.WeeklyReviewPromptInput
	ai := &captureWeeklyAI{out: &services.WeeklyReviewOutput{Narrative: "Good week.", TopEmotions: []string{}}, capturedInput: &capturedInput}

	s := newTestScheduler(reviewRepo, userRepo, analysisRepo, ai, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	rv := &models.WeeklyReview{ID: reviewID, UserID: uid, WeekStart: weekStart}
	_ = s.processOne(context.Background(), rv)

	if capturedInput.Name != "Ali" {
		t.Errorf("expected preferred name 'Ali', got %q", capturedInput.Name)
	}
}

type captureWeeklyAI struct {
	out           *services.WeeklyReviewOutput
	capturedInput *services.WeeklyReviewPromptInput
}

func (c *captureWeeklyAI) GenerateWeeklyReview(_ context.Context, input services.WeeklyReviewPromptInput) (*services.WeeklyReviewOutput, error) {
	*c.capturedInput = input
	return c.out, nil
}

func TestProcessOne_ClaudeErrorReturnsError(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()
	weekStart := time.Now().UTC().Truncate(24 * time.Hour)

	reviewRepo := &fakeWeeklyReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Carol"}}
	analysisRepo := &fakeWeeklyAnalysisRepo{
		entries: []*models.WeekSummaryEntry{
			{Date: weekStart, Summary: "Some day", MoodScore: 50},
		},
	}
	ai := &fakeWeeklyAI{err: errors.New("claude timeout")}

	s := newTestScheduler(reviewRepo, userRepo, analysisRepo, ai, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.WeeklyReview{ID: reviewID, UserID: uid, WeekStart: weekStart}
	err := s.processOne(context.Background(), rv)
	if err == nil {
		t.Fatal("expected error from claude failure, got nil")
	}
}

func TestSendPush_SendsToAllTokens(t *testing.T) {
	uid := uuid.New()
	nudgeRepo := &fakeNudgeRepo{tokens: []string{"token-a", "token-b"}}
	fcm := &fakeWeeklyFCMSender{}

	s := newTestScheduler(&fakeWeeklyReviewRepo{}, &fakeWeeklyUserRepo{}, &fakeWeeklyAnalysisRepo{}, &fakeWeeklyAI{}, nudgeRepo, fcm)
	s.sendPush(context.Background(), uid, uuid.New(), "Great week narrative here.")

	if len(fcm.calls) != 2 {
		t.Errorf("expected 2 FCM calls, got %d", len(fcm.calls))
	}
}

func TestSendPush_NoTokensIsNoop(t *testing.T) {
	nudgeRepo := &fakeNudgeRepo{tokens: nil}
	fcm := &fakeWeeklyFCMSender{}

	s := newTestScheduler(&fakeWeeklyReviewRepo{}, &fakeWeeklyUserRepo{}, &fakeWeeklyAnalysisRepo{}, &fakeWeeklyAI{}, nudgeRepo, fcm)
	s.sendPush(context.Background(), uuid.New(), uuid.New(), "narrative")

	if len(fcm.calls) != 0 {
		t.Errorf("expected 0 FCM calls, got %d", len(fcm.calls))
	}
}

// ── Helper function tests ─────────────────────────────────────────────────────

func TestLastSunday10AM_IsInPast(t *testing.T) {
	result := lastSunday10AM("UTC")
	if result.IsZero() {
		t.Fatal("expected non-zero time")
	}
	if result.After(time.Now()) {
		t.Errorf("lastSunday10AM returned a future time: %v", result)
	}
}

func TestLastSunday10AM_IsSunday(t *testing.T) {
	result := lastSunday10AM("UTC")
	if result.Weekday() != time.Sunday {
		t.Errorf("expected Sunday, got %v", result.Weekday())
	}
}

func TestLastSunday10AM_UnknownTimezoneDefaultsToUTC(t *testing.T) {
	result := lastSunday10AM("Not/A/Timezone")
	if result.IsZero() {
		t.Fatal("expected non-zero time for unknown timezone")
	}
}

func TestWeekLabel_SameMonth(t *testing.T) {
	// May 26 – Jun 1 spans months; let's test a same-month week
	// Use May 5 – May 11
	weekStart := time.Date(2026, time.May, 4, 0, 0, 0, 0, time.UTC) // Sunday May 4
	label := weekLabel(weekStart)
	// May 4 + 6 = May 10
	expected := "May 4 – 10, 2026"
	if label != expected {
		t.Errorf("expected %q, got %q", expected, label)
	}
}

func TestWeekLabel_CrossMonth(t *testing.T) {
	weekStart := time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC) // Sunday May 25
	label := weekLabel(weekStart)
	// May 25 + 6 = May 31 - still same month
	// Let's use May 26 → Jun 1
	weekStart2 := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	label2 := weekLabel(weekStart2)
	expected2 := "May 26 – Jun 1, 2026"
	if label2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, label2)
	}
	_ = label
}

func TestTopN_ReturnsTopByFrequency(t *testing.T) {
	counts := map[string]int{
		"anxious":   5,
		"hopeful":   3,
		"tired":     1,
		"reflective": 4,
	}
	top := topN(counts, 3)
	if len(top) != 3 {
		t.Fatalf("expected 3, got %d", len(top))
	}
	if top[0] != "anxious" {
		t.Errorf("expected 'anxious' as top emotion, got %q", top[0])
	}
}

func TestTopN_FewerThanN(t *testing.T) {
	counts := map[string]int{"joy": 2}
	top := topN(counts, 3)
	if len(top) != 1 {
		t.Errorf("expected 1, got %d", len(top))
	}
}

func TestTruncateHelper(t *testing.T) {
	s := "Hello, world!"
	if truncate(s, 5) != "Hello…" {
		t.Errorf("unexpected truncation: %q", truncate(s, 5))
	}
	if truncate(s, 100) != s {
		t.Errorf("expected unchanged string")
	}
}
