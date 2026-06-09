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

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeAnnualReviewRepo struct {
	scheduleErr    error
	scheduledCalls []uuid.UUID

	pendingReviews []*models.AnnualReview
	pendingErr     error

	completedID      uuid.UUID
	completedNarr    string
	completedEmotions []string
	completedTopics   []string
	completedErr     error

	failedID  uuid.UUID
	failedMsg string
	failedErr error
}

func (f *fakeAnnualReviewRepo) Schedule(_ context.Context, userID uuid.UUID, _ int, _ time.Time) error {
	f.scheduledCalls = append(f.scheduledCalls, userID)
	return f.scheduleErr
}

func (f *fakeAnnualReviewRepo) PendingDue(_ context.Context) ([]*models.AnnualReview, error) {
	return f.pendingReviews, f.pendingErr
}

func (f *fakeAnnualReviewRepo) MarkCompleted(_ context.Context, id uuid.UUID, narrative string, topEmotions, topTopics []string, _ []models.MonthlyMoodArcDay, _, _ int) error {
	f.completedID = id
	f.completedNarr = narrative
	f.completedEmotions = topEmotions
	f.completedTopics = topTopics
	return f.completedErr
}

func (f *fakeAnnualReviewRepo) MarkFailed(_ context.Context, id uuid.UUID, msg string) error {
	f.failedID = id
	f.failedMsg = msg
	return f.failedErr
}

type fakeAnnualAnalysisRepo struct {
	entries []*models.YearSummaryEntry
	err     error
}

func (f *fakeAnnualAnalysisRepo) GetYearSummaries(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.YearSummaryEntry, error) {
	return f.entries, f.err
}

type fakeAnnualAI struct {
	out *services.YearInReviewOutput
	err error
}

func (f *fakeAnnualAI) GenerateYearInReview(_ context.Context, _ services.YearInReviewPromptInput) (*services.YearInReviewOutput, error) {
	return f.out, f.err
}

// ── Constructor helper ────────────────────────────────────────────────────────

func newTestYearScheduler(
	reviewRepo annualReviewRepo,
	userRepo weeklyReviewUserRepo,
	analysisRepo annualReviewAnalysisRepo,
	ai annualReviewAI,
	nudge nudgeDispatcher,
	fcm fcmSender,
) *YearInReviewScheduler {
	return NewYearInReviewScheduler(YearInReviewSchedulerDeps{
		ReviewRepo:   reviewRepo,
		UserRepo:     userRepo,
		AnalysisRepo: analysisRepo,
		Claude:       ai,
		NudgeRepo:    nudge,
		FCM:          fcm,
		Log:          zap.NewNop(),
	})
}

// ── Tests: scheduleForActiveUsers ────────────────────────────────────────────

func TestYearScheduleForActiveUsers_SchedulesPerUser(t *testing.T) {
	uid1, uid2 := uuid.New(), uuid.New()
	userRepo := &fakeWeeklyUserRepo{
		users: []*models.User{
			{ID: uid1, Timezone: "UTC"},
			{ID: uid2, Timezone: "UTC"},
		},
	}
	reviewRepo := &fakeAnnualReviewRepo{}

	s := newTestYearScheduler(reviewRepo, userRepo, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.scheduleForActiveUsers(context.Background())

	if len(reviewRepo.scheduledCalls) != 2 {
		t.Fatalf("expected 2 schedule calls, got %d", len(reviewRepo.scheduledCalls))
	}
}

func TestYearScheduleForActiveUsers_NoUsersIsNoop(t *testing.T) {
	userRepo := &fakeWeeklyUserRepo{users: nil}
	reviewRepo := &fakeAnnualReviewRepo{}

	s := newTestYearScheduler(reviewRepo, userRepo, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.scheduleForActiveUsers(context.Background())

	if len(reviewRepo.scheduledCalls) != 0 {
		t.Fatalf("expected 0 schedule calls, got %d", len(reviewRepo.scheduledCalls))
	}
}

func TestYearScheduleForActiveUsers_ListErrorIsNoop(t *testing.T) {
	userRepo := &fakeWeeklyUserRepo{listErr: errors.New("db error")}
	reviewRepo := &fakeAnnualReviewRepo{}

	s := newTestYearScheduler(reviewRepo, userRepo, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.scheduleForActiveUsers(context.Background())

	if len(reviewRepo.scheduledCalls) != 0 {
		t.Fatalf("expected 0 schedule calls on list error, got %d", len(reviewRepo.scheduledCalls))
	}
}

// ── Tests: processPending ─────────────────────────────────────────────────────

func TestYearProcessPending_NoPendingIsNoop(t *testing.T) {
	reviewRepo := &fakeAnnualReviewRepo{pendingReviews: nil}

	s := newTestYearScheduler(reviewRepo, &fakeWeeklyUserRepo{}, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.processPending(context.Background())
	// no panic = pass
}

func TestYearProcessPending_PendingErrorIsNoop(t *testing.T) {
	reviewRepo := &fakeAnnualReviewRepo{pendingErr: errors.New("db error")}

	s := newTestYearScheduler(reviewRepo, &fakeWeeklyUserRepo{}, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	s.processPending(context.Background())
	// no panic = pass
}

// ── Tests: processOne ─────────────────────────────────────────────────────────

func TestYearProcessOne_HappyPath(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()
	year := 2025

	reviewRepo := &fakeAnnualReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Alice"}}
	analysisRepo := &fakeAnnualAnalysisRepo{
		entries: []*models.YearSummaryEntry{
			{Date: time.Date(year, time.March, 15, 0, 0, 0, 0, time.UTC), Summary: "Good day", MoodScore: 70, Emotions: []string{"hopeful"}, Topics: []string{"work"}},
			{Date: time.Date(year, time.July, 10, 0, 0, 0, 0, time.UTC), Summary: "Hard day", MoodScore: 40, Emotions: []string{"anxious"}, Topics: []string{"family"}},
		},
	}
	ai := &fakeAnnualAI{
		out: &services.YearInReviewOutput{
			Narrative:   "Alice's year was one of growth and challenge.",
			TopEmotions: []string{"hopeful", "anxious"},
			TopTopics:   []string{"work", "family"},
		},
	}

	s := newTestYearScheduler(reviewRepo, userRepo, analysisRepo, ai, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.AnnualReview{ID: reviewID, UserID: uid, Year: year}
	err := s.processOne(context.Background(), rv)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if reviewRepo.completedID != reviewID {
		t.Errorf("expected MarkCompleted called with review ID %s, got %s", reviewID, reviewRepo.completedID)
	}
	if reviewRepo.completedNarr != "Alice's year was one of growth and challenge." {
		t.Errorf("unexpected narrative: %q", reviewRepo.completedNarr)
	}
	if len(reviewRepo.completedEmotions) != 2 {
		t.Errorf("expected 2 top emotions, got %d", len(reviewRepo.completedEmotions))
	}
}

func TestYearProcessOne_NoEntriesMarksFailedSoftly(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()

	reviewRepo := &fakeAnnualReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Bob"}}
	analysisRepo := &fakeAnnualAnalysisRepo{entries: nil}

	s := newTestYearScheduler(reviewRepo, userRepo, analysisRepo, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.AnnualReview{ID: reviewID, UserID: uid, Year: 2025}
	err := s.processOne(context.Background(), rv)
	if err != nil {
		t.Fatalf("expected nil error (soft fail), got: %v", err)
	}
	if reviewRepo.failedID != reviewID {
		t.Errorf("expected MarkFailed called with review ID %s, got %s", reviewID, reviewRepo.failedID)
	}
}

func TestYearProcessOne_UserNotFoundReturnsError(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()

	reviewRepo := &fakeAnnualReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: nil, getErr: errors.New("db error")}

	s := newTestYearScheduler(reviewRepo, userRepo, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.AnnualReview{ID: reviewID, UserID: uid, Year: 2025}
	err := s.processOne(context.Background(), rv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestYearProcessOne_AnalysisRepoErrorReturnsError(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()

	reviewRepo := &fakeAnnualReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Carol"}}
	analysisRepo := &fakeAnnualAnalysisRepo{err: errors.New("db error")}

	s := newTestYearScheduler(reviewRepo, userRepo, analysisRepo, &fakeAnnualAI{}, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.AnnualReview{ID: reviewID, UserID: uid, Year: 2025}
	err := s.processOne(context.Background(), rv)
	if err == nil {
		t.Fatal("expected error from analysis repo, got nil")
	}
}

func TestYearProcessOne_ClaudeErrorReturnsError(t *testing.T) {
	uid := uuid.New()
	reviewID := uuid.New()
	year := 2025

	reviewRepo := &fakeAnnualReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Dave"}}
	analysisRepo := &fakeAnnualAnalysisRepo{
		entries: []*models.YearSummaryEntry{
			{Date: time.Date(year, time.June, 1, 0, 0, 0, 0, time.UTC), Summary: "A day", MoodScore: 60, Emotions: []string{"calm"}},
		},
	}
	ai := &fakeAnnualAI{err: errors.New("claude timeout")}

	s := newTestYearScheduler(reviewRepo, userRepo, analysisRepo, ai, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})

	rv := &models.AnnualReview{ID: reviewID, UserID: uid, Year: year}
	err := s.processOne(context.Background(), rv)
	if err == nil {
		t.Fatal("expected error from claude failure, got nil")
	}
}

func TestYearProcessOne_UsesPreferredName(t *testing.T) {
	uid := uuid.New()
	preferred := "Allie"
	reviewID := uuid.New()
	year := 2025

	reviewRepo := &fakeAnnualReviewRepo{}
	userRepo := &fakeWeeklyUserRepo{getUser: &models.User{ID: uid, Name: "Alice", PreferredName: &preferred}}
	analysisRepo := &fakeAnnualAnalysisRepo{
		entries: []*models.YearSummaryEntry{
			{Date: time.Date(year, time.May, 1, 0, 0, 0, 0, time.UTC), Summary: "A day", MoodScore: 65, Emotions: []string{"hopeful"}},
		},
	}

	var capturedInput services.YearInReviewPromptInput
	ai := &captureAnnualAI{
		out:           &services.YearInReviewOutput{Narrative: "Good year.", TopEmotions: []string{}, TopTopics: []string{}},
		capturedInput: &capturedInput,
	}

	s := newTestYearScheduler(reviewRepo, userRepo, analysisRepo, ai, &fakeNudgeRepo{}, &fakeWeeklyFCMSender{})
	rv := &models.AnnualReview{ID: reviewID, UserID: uid, Year: year}
	_ = s.processOne(context.Background(), rv)

	if capturedInput.Name != "Allie" {
		t.Errorf("expected preferred name 'Allie', got %q", capturedInput.Name)
	}
}

type captureAnnualAI struct {
	out           *services.YearInReviewOutput
	capturedInput *services.YearInReviewPromptInput
}

func (c *captureAnnualAI) GenerateYearInReview(_ context.Context, input services.YearInReviewPromptInput) (*services.YearInReviewOutput, error) {
	*c.capturedInput = input
	return c.out, nil
}

// ── Tests: sendPush ───────────────────────────────────────────────────────────

func TestYearSendPush_SendsToAllTokens(t *testing.T) {
	uid := uuid.New()
	nudgeRepo := &fakeNudgeRepo{tokens: []string{"token-a", "token-b"}}
	fcm := &fakeWeeklyFCMSender{}

	s := newTestYearScheduler(&fakeAnnualReviewRepo{}, &fakeWeeklyUserRepo{}, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, nudgeRepo, fcm)
	s.sendPush(context.Background(), uid, uuid.New(), "Great year narrative.", 2025)

	if len(fcm.calls) != 2 {
		t.Errorf("expected 2 FCM calls, got %d", len(fcm.calls))
	}
}

func TestYearSendPush_NoTokensIsNoop(t *testing.T) {
	nudgeRepo := &fakeNudgeRepo{tokens: nil}
	fcm := &fakeWeeklyFCMSender{}

	s := newTestYearScheduler(&fakeAnnualReviewRepo{}, &fakeWeeklyUserRepo{}, &fakeAnnualAnalysisRepo{}, &fakeAnnualAI{}, nudgeRepo, fcm)
	s.sendPush(context.Background(), uuid.New(), uuid.New(), "narrative", 2025)

	if len(fcm.calls) != 0 {
		t.Errorf("expected 0 FCM calls, got %d", len(fcm.calls))
	}
}

// ── Tests: lastJan1_10AM helper ───────────────────────────────────────────────

func TestLastJan1_10AM_IsInPast(t *testing.T) {
	result := lastJan1_10AM("UTC")
	if result.IsZero() {
		t.Fatal("expected non-zero time")
	}
	if result.After(time.Now()) {
		t.Errorf("lastJan1_10AM returned a future time: %v", result)
	}
}

func TestLastJan1_10AM_IsJanuary1(t *testing.T) {
	result := lastJan1_10AM("UTC")
	if result.Month() != time.January || result.Day() != 1 {
		t.Errorf("expected Jan 1, got %v", result)
	}
}

func TestLastJan1_10AM_UnknownTimezoneDefaultsToUTC(t *testing.T) {
	result := lastJan1_10AM("Not/A/Timezone")
	if result.IsZero() {
		t.Fatal("expected non-zero time for unknown timezone")
	}
}

func TestLastJan1_10AM_EmptyTimezoneDefaultsToUTC(t *testing.T) {
	result := lastJan1_10AM("")
	if result.IsZero() {
		t.Fatal("expected non-zero time for empty timezone")
	}
}

// ── Tests: aggregateYearData helper ──────────────────────────────────────────

func TestAggregateYearData_SingleEntry(t *testing.T) {
	year := 2025
	entries := []*models.YearSummaryEntry{
		{
			Date:      time.Date(year, time.March, 10, 0, 0, 0, 0, time.UTC),
			Summary:   "A good day",
			MoodScore: 70,
			Emotions:  []string{"hopeful", "calm"},
			Topics:    []string{"work"},
		},
	}

	moodArc, avgMood, monthlyLines, topEmotions, topTopics, summaries := aggregateYearData(entries, year)

	if avgMood != 70 {
		t.Errorf("expected avgMood=70, got %d", avgMood)
	}
	if len(moodArc) != 1 {
		t.Fatalf("expected 1 month in arc, got %d", len(moodArc))
	}
	if moodArc[0].Month != "2025-03" {
		t.Errorf("expected month 2025-03, got %q", moodArc[0].Month)
	}
	if moodArc[0].AvgMood != 70 {
		t.Errorf("expected arc avg_mood=70, got %d", moodArc[0].AvgMood)
	}
	if moodArc[0].EntryCount != 1 {
		t.Errorf("expected entry_count=1, got %d", moodArc[0].EntryCount)
	}
	if len(monthlyLines) != 1 {
		t.Errorf("expected 1 monthly line, got %d", len(monthlyLines))
	}
	if len(topEmotions) == 0 {
		t.Error("expected at least 1 top emotion")
	}
	if len(topTopics) == 0 {
		t.Error("expected at least 1 top topic")
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(summaries))
	}
}

func TestAggregateYearData_MultipleMonths(t *testing.T) {
	year := 2025
	entries := []*models.YearSummaryEntry{
		{Date: time.Date(year, time.January, 5, 0, 0, 0, 0, time.UTC), Summary: "Jan day", MoodScore: 60, Emotions: []string{"anxious"}},
		{Date: time.Date(year, time.June, 15, 0, 0, 0, 0, time.UTC), Summary: "Jun day", MoodScore: 80, Emotions: []string{"hopeful"}},
		{Date: time.Date(year, time.December, 20, 0, 0, 0, 0, time.UTC), Summary: "Dec day", MoodScore: 70, Emotions: []string{"calm"}},
	}

	moodArc, avgMood, monthlyLines, _, _, summaries := aggregateYearData(entries, year)

	if avgMood != 70 { // (60+80+70)/3
		t.Errorf("expected avgMood=70, got %d", avgMood)
	}
	if len(moodArc) != 3 {
		t.Errorf("expected 3 months in arc, got %d", len(moodArc))
	}
	if len(monthlyLines) != 3 {
		t.Errorf("expected 3 monthly lines, got %d", len(monthlyLines))
	}
	if len(summaries) != 3 {
		t.Errorf("expected 3 summaries (one per month), got %d", len(summaries))
	}
}

func TestAggregateYearData_TopEmotionsSortedByFrequency(t *testing.T) {
	year := 2025
	entries := []*models.YearSummaryEntry{
		{Date: time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC), Emotions: []string{"anxious", "anxious", "hopeful"}, Topics: []string{"work"}},
		{Date: time.Date(year, time.February, 1, 0, 0, 0, 0, time.UTC), Emotions: []string{"anxious", "calm"}, Topics: []string{"family"}},
		{Date: time.Date(year, time.March, 1, 0, 0, 0, 0, time.UTC), Emotions: []string{"hopeful", "calm"}, Topics: []string{"rest"}},
	}

	_, _, _, topEmotions, topTopics, _ := aggregateYearData(entries, year)

	if len(topEmotions) == 0 {
		t.Fatal("expected top emotions, got none")
	}
	if topEmotions[0] != "anxious" {
		t.Errorf("expected 'anxious' as top emotion (frequency 3), got %q", topEmotions[0])
	}
	if len(topTopics) != 3 {
		t.Errorf("expected 3 top topics, got %d", len(topTopics))
	}
}

func TestAggregateYearData_OneSummaryPerMonth(t *testing.T) {
	year := 2025
	// Two entries in the same month - only one summary should appear for that month
	entries := []*models.YearSummaryEntry{
		{Date: time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC), Summary: "First April", MoodScore: 65, Emotions: []string{"calm"}},
		{Date: time.Date(year, time.April, 15, 0, 0, 0, 0, time.UTC), Summary: "Second April", MoodScore: 70, Emotions: []string{"hopeful"}},
	}

	_, _, _, _, _, summaries := aggregateYearData(entries, year)

	if len(summaries) != 1 {
		t.Errorf("expected 1 summary for two entries in same month, got %d", len(summaries))
	}
	if summaries[0] != "First April" {
		t.Errorf("expected first entry summary, got %q", summaries[0])
	}
}

func TestAggregateYearData_MoodArcOrderedByMonth(t *testing.T) {
	year := 2025
	// Insert entries out of order
	entries := []*models.YearSummaryEntry{
		{Date: time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC), MoodScore: 50, Emotions: []string{"tired"}},
		{Date: time.Date(year, time.February, 1, 0, 0, 0, 0, time.UTC), MoodScore: 75, Emotions: []string{"hopeful"}},
		{Date: time.Date(year, time.May, 1, 0, 0, 0, 0, time.UTC), MoodScore: 65, Emotions: []string{"calm"}},
	}

	moodArc, _, _, _, _, _ := aggregateYearData(entries, year)

	if len(moodArc) != 3 {
		t.Fatalf("expected 3 months, got %d", len(moodArc))
	}
	if moodArc[0].Month != "2025-02" {
		t.Errorf("expected first arc entry to be Feb (2025-02), got %q", moodArc[0].Month)
	}
	if moodArc[1].Month != "2025-05" {
		t.Errorf("expected second arc entry to be May (2025-05), got %q", moodArc[1].Month)
	}
	if moodArc[2].Month != "2025-10" {
		t.Errorf("expected third arc entry to be Oct (2025-10), got %q", moodArc[2].Month)
	}
}
