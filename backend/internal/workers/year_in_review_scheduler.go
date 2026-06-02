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

// YearInReviewScheduler creates and processes annual review rows.
// It ticks every 6 hours: first scheduling pending reviews for active users,
// then generating any past-due pending reviews via Claude.
type YearInReviewScheduler struct {
	reviewRepo   annualReviewRepo
	userRepo     weeklyReviewUserRepo // same interface — reuse
	analysisRepo annualReviewAnalysisRepo
	claude       annualReviewAI
	nudgeRepo    nudgeDispatcher
	fcm          fcmSender
	log          *zap.Logger
}

type YearInReviewSchedulerDeps struct {
	ReviewRepo   annualReviewRepo
	UserRepo     weeklyReviewUserRepo
	AnalysisRepo annualReviewAnalysisRepo
	Claude       annualReviewAI
	NudgeRepo    nudgeDispatcher
	FCM          fcmSender
	Log          *zap.Logger
}

func NewYearInReviewScheduler(deps YearInReviewSchedulerDeps) *YearInReviewScheduler {
	return &YearInReviewScheduler{
		reviewRepo:   deps.ReviewRepo,
		userRepo:     deps.UserRepo,
		analysisRepo: deps.AnalysisRepo,
		claude:       deps.Claude,
		nudgeRepo:    deps.NudgeRepo,
		fcm:          deps.FCM,
		log:          deps.Log,
	}
}

// Run blocks until ctx is cancelled, ticking every 6 hours.
func (s *YearInReviewScheduler) Run(ctx context.Context) {
	s.log.Info("year-in-review scheduler starting")
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("year-in-review scheduler stopping")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *YearInReviewScheduler) tick(ctx context.Context) {
	s.scheduleForActiveUsers(ctx)
	s.processPending(ctx)
}

// scheduleForActiveUsers finds users with entries in the previous calendar year
// and inserts a pending annual_review row for that year. Idempotent.
func (s *YearInReviewScheduler) scheduleForActiveUsers(ctx context.Context) {
	prevYearStart := time.Date(time.Now().UTC().Year()-1, 1, 1, 0, 0, 0, 0, time.UTC)
	users, err := s.userRepo.ListWithRecentEntries(ctx, prevYearStart)
	if err != nil {
		s.log.Error("year-in-review scheduler: list active users", zap.Error(err))
		return
	}
	if len(users) == 0 {
		return
	}

	for _, u := range users {
		scheduledAt := lastJan1_10AM(u.Timezone)
		if scheduledAt.IsZero() {
			continue
		}
		reviewYear := scheduledAt.UTC().Year() - 1
		if err := s.reviewRepo.Schedule(ctx, u.ID, reviewYear, scheduledAt); err != nil {
			s.log.Warn("year-in-review scheduler: schedule row",
				zap.String("user_id", u.ID.String()), zap.Error(err))
		}
	}
}

// processPending fetches past-due pending rows and generates via Claude.
func (s *YearInReviewScheduler) processPending(ctx context.Context) {
	reviews, err := s.reviewRepo.PendingDue(ctx)
	if err != nil {
		s.log.Error("year-in-review scheduler: fetch pending", zap.Error(err))
		return
	}
	if len(reviews) == 0 {
		return
	}

	s.log.Info("year-in-review scheduler: processing", zap.Int("count", len(reviews)))

	for _, rv := range reviews {
		if err := s.processOne(ctx, rv); err != nil {
			s.log.Warn("year-in-review scheduler: process failed",
				zap.String("review_id", rv.ID.String()), zap.Error(err))
			_ = s.reviewRepo.MarkFailed(ctx, rv.ID, err.Error())
		}
	}
}

func (s *YearInReviewScheduler) processOne(ctx context.Context, rv *models.AnnualReview) error {
	user, err := s.userRepo.GetByID(ctx, rv.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("fetch user: %w", err)
	}

	yearStart := time.Date(rv.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(rv.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	entries, err := s.analysisRepo.GetYearSummaries(ctx, rv.UserID, yearStart, yearEnd)
	if err != nil {
		return fmt.Errorf("fetch year summaries: %w", err)
	}
	if len(entries) == 0 {
		_ = s.reviewRepo.MarkFailed(ctx, rv.ID, "no entries found for the year")
		return nil
	}

	name := user.Name
	if user.PreferredName != nil && *user.PreferredName != "" {
		name = *user.PreferredName
	}

	moodArc, avgMood, monthlyLines, topEmotions, topTopics, summaries := aggregateYearData(entries, rv.Year)

	input := services.YearInReviewPromptInput{
		Name:        name,
		Year:        rv.Year,
		EntryCount:  len(entries),
		AvgMood:     avgMood,
		MonthlyArc:  monthlyLines,
		TopEmotions: topEmotions,
		TopTopics:   topTopics,
		Summaries:   summaries,
	}

	out, err := s.claude.GenerateYearInReview(ctx, input)
	if err != nil {
		return fmt.Errorf("claude: %w", err)
	}

	if err := s.reviewRepo.MarkCompleted(ctx, rv.ID, out.Narrative, out.TopEmotions, out.TopTopics, moodArc, len(entries), avgMood); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	s.sendPush(ctx, rv.UserID, rv.ID, out.Narrative, rv.Year)
	return nil
}

func (s *YearInReviewScheduler) sendPush(ctx context.Context, userID, reviewID uuid.UUID, narrative string, year int) {
	tokens, err := s.nudgeRepo.GetDeviceTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}
	preview := truncate(narrative, 120)
	for _, token := range tokens {
		_ = s.fcm.SendToToken(ctx, token,
			fmt.Sprintf("Your %d in review", year),
			preview,
			map[string]string{
				"type":      "annual_review",
				"review_id": reviewID.String(),
			},
		)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// lastJan1_10AM returns Jan 1 of the most recent past year at 10:00 AM in the
// given IANA timezone. If today is Jan 1 before 10 AM, it returns the previous
// year's Jan 1. Returns zero value on unknown timezone.
func lastJan1_10AM(timezone string) time.Time {
	if timezone == "" {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	result := time.Date(now.Year(), 1, 1, 10, 0, 0, 0, loc)
	if result.After(now) {
		result = time.Date(now.Year()-1, 1, 1, 10, 0, 0, 0, loc)
	}
	return result.UTC()
}

// aggregateYearData derives mood arc, avg mood, monthly description lines,
// top emotions, top topics, and representative summaries from raw year data.
func aggregateYearData(entries []*models.YearSummaryEntry, year int) (
	moodArc []models.MonthlyMoodArcDay,
	avgMood int,
	monthlyLines []string,
	topEmotions []string,
	topTopics []string,
	summaries []string,
) {
	type monthBucket struct {
		moods    []int
		emotions []string
		topics   []string
		summary  string // one representative summary per month (first seen)
	}

	buckets := make(map[int]*monthBucket) // month 1–12
	emotionCounts := map[string]int{}
	topicCounts := map[string]int{}
	allMoods := []int{}

	for _, e := range entries {
		m := int(e.Date.UTC().Month())
		if _, ok := buckets[m]; !ok {
			buckets[m] = &monthBucket{}
		}
		b := buckets[m]
		b.moods = append(b.moods, e.MoodScore)
		b.emotions = append(b.emotions, e.Emotions...)
		b.topics = append(b.topics, e.Topics...)
		if b.summary == "" && e.Summary != "" {
			b.summary = e.Summary
		}
		allMoods = append(allMoods, e.MoodScore)
		for _, em := range e.Emotions {
			emotionCounts[em]++
		}
		for _, tp := range e.Topics {
			topicCounts[tp]++
		}
	}

	avgMood = avgInt(allMoods)

	// Build monthly arc for each month of the year.
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for m := 1; m <= 12; m++ {
		b, ok := buckets[m]
		if !ok {
			continue
		}
		monthStr := fmt.Sprintf("%d-%02d", year, m)
		avg := avgInt(b.moods)
		moodArc = append(moodArc, models.MonthlyMoodArcDay{
			Month:      monthStr,
			AvgMood:    avg,
			EntryCount: len(b.moods),
		})
		monthlyLines = append(monthlyLines, fmt.Sprintf("%s %d: mood %d (%d entries)", monthNames[m-1], year, avg, len(b.moods)))
		if b.summary != "" {
			summaries = append(summaries, b.summary)
		}
	}

	topEmotions = topN(emotionCounts, 5)
	topTopics = topN(topicCounts, 5)
	return
}
