package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WeeklyReviewScheduler creates and processes weekly review rows.
// It ticks hourly: first scheduling pending reviews for active users,
// then generating any past-due pending reviews via Claude.
type WeeklyReviewScheduler struct {
	reviewRepo   weeklyReviewRepo
	userRepo     weeklyReviewUserRepo
	analysisRepo weeklyReviewAnalysisRepo
	claude       weeklyReviewAI
	nudgeRepo    nudgeDispatcher
	fcm          fcmSender
	freezeGranter weeklyFreezeGranter
	log          *zap.Logger
}

type WeeklyReviewSchedulerDeps struct {
	ReviewRepo    weeklyReviewRepo
	UserRepo      weeklyReviewUserRepo
	AnalysisRepo  weeklyReviewAnalysisRepo
	Claude        weeklyReviewAI
	NudgeRepo     nudgeDispatcher
	FCM           fcmSender
	FreezeGranter weeklyFreezeGranter
	Log           *zap.Logger
}

func NewWeeklyReviewScheduler(deps WeeklyReviewSchedulerDeps) *WeeklyReviewScheduler {
	return &WeeklyReviewScheduler{
		reviewRepo:    deps.ReviewRepo,
		userRepo:      deps.UserRepo,
		analysisRepo:  deps.AnalysisRepo,
		claude:        deps.Claude,
		nudgeRepo:     deps.NudgeRepo,
		fcm:           deps.FCM,
		freezeGranter: deps.FreezeGranter,
		log:           deps.Log,
	}
}

// Run blocks until ctx is cancelled, ticking every hour.
func (s *WeeklyReviewScheduler) Run(ctx context.Context) {
	s.log.Info("weekly review scheduler starting")
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("weekly review scheduler stopping")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *WeeklyReviewScheduler) tick(ctx context.Context) {
	s.grantWeeklyFreezes(ctx)
	s.scheduleForActiveUsers(ctx)
	s.processPending(ctx)
}

func (s *WeeklyReviewScheduler) grantWeeklyFreezes(ctx context.Context) {
	if s.freezeGranter == nil {
		return
	}
	if err := s.freezeGranter.GrantWeeklyFreezes(ctx); err != nil {
		s.log.Warn("weekly review scheduler: grant weekly freezes", zap.Error(err))
	}
}

// scheduleForActiveUsers finds users with entries in the last 7 days and
// inserts a pending weekly_review row for the most recent past Sunday 10 AM
// (idempotent — ON CONFLICT DO NOTHING prevents duplicates).
func (s *WeeklyReviewScheduler) scheduleForActiveUsers(ctx context.Context) {
	since := time.Now().Add(-7 * 24 * time.Hour)
	users, err := s.userRepo.ListWithRecentEntries(ctx, since)
	if err != nil {
		s.log.Error("weekly review scheduler: list active users", zap.Error(err))
		return
	}
	if len(users) == 0 {
		return
	}

	for _, u := range users {
		scheduledAt := lastSunday10AM(u.Timezone)
		if scheduledAt.IsZero() {
			continue
		}
		weekStart := sundayDate(scheduledAt)
		if err := s.reviewRepo.Schedule(ctx, u.ID, weekStart, scheduledAt); err != nil {
			s.log.Warn("weekly review scheduler: schedule row",
				zap.String("user_id", u.ID.String()), zap.Error(err))
		}
	}
}

// processPending fetches past-due pending rows and generates their review via Claude.
func (s *WeeklyReviewScheduler) processPending(ctx context.Context) {
	reviews, err := s.reviewRepo.PendingDue(ctx)
	if err != nil {
		s.log.Error("weekly review scheduler: fetch pending", zap.Error(err))
		return
	}
	if len(reviews) == 0 {
		return
	}

	s.log.Info("weekly review scheduler: processing", zap.Int("count", len(reviews)))

	for _, rv := range reviews {
		if err := s.processOne(ctx, rv); err != nil {
			s.log.Warn("weekly review scheduler: process failed",
				zap.String("review_id", rv.ID.String()), zap.Error(err))
			_ = s.reviewRepo.MarkFailed(ctx, rv.ID, err.Error())
		}
	}
}

func (s *WeeklyReviewScheduler) processOne(ctx context.Context, rv *models.WeeklyReview) error {
	// Fetch the user for name / preferred name.
	user, err := s.userRepo.GetByID(ctx, rv.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("fetch user: %w", err)
	}

	// Fetch entry data for the 7 days ending on this week_start.
	weekStart := rv.WeekStart.UTC().Truncate(24 * time.Hour)
	since := weekStart.Add(-6 * 24 * time.Hour) // Mon–Sun window
	entries, err := s.analysisRepo.GetWeekSummaries(ctx, rv.UserID, since)
	if err != nil {
		return fmt.Errorf("fetch week summaries: %w", err)
	}

	if len(entries) == 0 {
		// No entries this week — mark failed with a soft reason.
		_ = s.reviewRepo.MarkFailed(ctx, rv.ID, "no entries found for the week")
		return nil
	}

	// Build prompt input.
	name := user.Name
	if user.PreferredName != nil && *user.PreferredName != "" {
		name = *user.PreferredName
	}

	moodArc, dailyMoods, topEmotions, summaries := aggregateWeekData(entries, weekStart)

	input := services.WeeklyReviewPromptInput{
		Name:        name,
		WeekLabel:   weekLabel(weekStart),
		EntryCount:  len(entries),
		DailyMoods:  dailyMoods,
		Summaries:   summaries,
		TopEmotions: topEmotions,
	}

	out, err := s.claude.GenerateWeeklyReview(ctx, input)
	if err != nil {
		return fmt.Errorf("claude: %w", err)
	}

	if err := s.reviewRepo.MarkCompleted(ctx, rv.ID, out.Narrative, out.TopEmotions, moodArc, len(entries)); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	// Send FCM push (non-fatal if it fails).
	s.sendPush(ctx, rv.UserID, rv.ID, out.Narrative)
	return nil
}

func (s *WeeklyReviewScheduler) sendPush(ctx context.Context, userID, reviewID uuid.UUID, narrative string) {
	tokens, err := s.nudgeRepo.GetDeviceTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}
	preview := truncate(narrative, 120)
	for _, token := range tokens {
		_ = s.fcm.SendToToken(ctx, token, "Your week in review", preview, map[string]string{
			"type":      "weekly_review",
			"review_id": reviewID.String(),
		})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// lastSunday10AM returns the most recent past Sunday at 10:00 AM in the given
// IANA timezone. Returns zero value on unknown timezone.
func lastSunday10AM(timezone string) time.Time {
	if timezone == "" {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	// Days since last Sunday (weekday 0=Sunday).
	daysSince := int(now.Weekday())
	lastSun := now.AddDate(0, 0, -daysSince)
	result := time.Date(lastSun.Year(), lastSun.Month(), lastSun.Day(), 10, 0, 0, 0, loc)

	// If "last Sunday 10 AM" is in the future (i.e., today is Sunday and it's before 10 AM), go back 7 days.
	if result.After(now) {
		result = result.AddDate(0, 0, -7)
	}
	return result.UTC()
}

// sundayDate returns midnight UTC of the Sunday in scheduledAt.
func sundayDate(scheduledAt time.Time) time.Time {
	t := scheduledAt.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// weekLabel formats a week range like "May 26 – Jun 1, 2026".
func weekLabel(weekStart time.Time) string {
	end := weekStart.AddDate(0, 0, 6)
	if weekStart.Month() == end.Month() {
		return fmt.Sprintf("%s %d – %d, %d", weekStart.Format("Jan"), weekStart.Day(), end.Day(), weekStart.Year())
	}
	return fmt.Sprintf("%s %d – %s %d, %d", weekStart.Format("Jan"), weekStart.Day(), end.Format("Jan"), end.Day(), weekStart.Year())
}

// aggregateWeekData derives the mood arc, daily mood strings, top emotions,
// and summaries from the raw entry data.
func aggregateWeekData(entries []*models.WeekSummaryEntry, weekStart time.Time) (
	moodArc []models.MoodArcDay,
	dailyMoods []string,
	topEmotions []string,
	summaries []string,
) {
	// Group by date.
	type dayBucket struct {
		moods    []int
		emotions []string
	}
	buckets := map[string]*dayBucket{}
	emotionCounts := map[string]int{}

	for _, e := range entries {
		day := e.Date.UTC().Format("2006-01-02")
		if _, ok := buckets[day]; !ok {
			buckets[day] = &dayBucket{}
		}
		buckets[day].moods = append(buckets[day].moods, e.MoodScore)
		buckets[day].emotions = append(buckets[day].emotions, e.Emotions...)
		for _, em := range e.Emotions {
			emotionCounts[em]++
		}
		if e.Summary != "" {
			summaries = append(summaries, e.Summary)
		}
	}

	// Build mood arc for each day of the week (Mon-Sun).
	for i := -6; i <= 0; i++ {
		d := weekStart.AddDate(0, 0, i).UTC().Format("2006-01-02")
		if b, ok := buckets[d]; ok {
			avg := avgInt(b.moods)
			moodArc = append(moodArc, models.MoodArcDay{Date: d, AvgMood: avg})
			dailyMoods = append(dailyMoods, fmt.Sprintf("%s: mood %d", formatShortDate(d), avg))
		}
	}

	// Top emotions by frequency.
	topEmotions = topN(emotionCounts, 3)
	return
}

func avgInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return sum / len(vals)
}

func formatShortDate(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return d
	}
	return t.Format("Mon Jan 2")
}

func topN(counts map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	sorted := make([]kv, 0, len(counts))
	for k, v := range counts {
		sorted = append(sorted, kv{k, v})
	}
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].v > sorted[j-1].v; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	result := make([]string, len(sorted))
	for i, kv := range sorted {
		result[i] = kv.k
	}
	return result
}

// truncate is re-declared locally since the workers package can't import services.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

