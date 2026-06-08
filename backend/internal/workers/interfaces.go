package workers

import (
	"context"
	"io"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
)

// These narrow interfaces let the worker be unit-tested with fakes.
// Every concrete type used in production satisfies the relevant interface.

type jobQueue interface {
	Dequeue(ctx context.Context) ([]byte, error)
	Enqueue(ctx context.Context, v any) error
	EnqueueDLQ(ctx context.Context, payload []byte, errMsg string) error
}

type entryStore interface {
	SetProcessing(ctx context.Context, id uuid.UUID) (bool, error)
	GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Entry, error)
	SetCompleted(ctx context.Context, id uuid.UUID, transcript, language string) error
	SetFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

type analysisStore interface {
	Upsert(ctx context.Context, entryID uuid.UUID, a *models.EntryAnalysis) (*models.EntryAnalysis, error)
}

type audioStorage interface {
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type audioTranscriber interface {
	Transcribe(ctx context.Context, r io.Reader, filename string) (*services.WhisperResponse, error)
}

type crisisScreener interface {
	Screen(ctx context.Context, transcript, country string) (*services.CrisisResult, error)
}

type contextAssembler interface {
	Build(ctx context.Context, entryID, userID uuid.UUID) (*services.AnalyzeEntryInput, error)
}

type aiAnalyzer interface {
	AnalyzeEntry(ctx context.Context, input services.AnalyzeEntryInput) (*models.ClaudeAnalysisOutput, error)
}

type nudgeScheduler interface {
	ScheduleMorningNudge(ctx context.Context, userID, entryID uuid.UUID, message string) error
}

// nudgeDispatcher is the subset of NudgeRepository used by NudgeScheduler.
type nudgeDispatcher interface {
	PendingDue(ctx context.Context) ([]*models.Nudge, error)
	GetDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error)
	MarkSent(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

// fcmSender is the subset of FCMService used by NudgeScheduler.
type fcmSender interface {
	SendToToken(ctx context.Context, token, title, body string, data map[string]string) error
}

// weeklyReviewRepo is the subset of WeeklyReviewRepository used by WeeklyReviewScheduler.
type weeklyReviewRepo interface {
	Schedule(ctx context.Context, userID uuid.UUID, weekStart time.Time, scheduledAt time.Time) error
	PendingDue(ctx context.Context) ([]*models.WeeklyReview, error)
	MarkCompleted(ctx context.Context, id uuid.UUID, narrative string, topEmotions []string, moodArc []models.MoodArcDay, entryCount int) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

// weeklyReviewUserRepo is the subset of UserRepository used by WeeklyReviewScheduler.
type weeklyReviewUserRepo interface {
	ListWithRecentEntries(ctx context.Context, since time.Time) ([]*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// weeklyReviewAnalysisRepo is the subset of AnalysisRepository used by WeeklyReviewScheduler.
type weeklyReviewAnalysisRepo interface {
	GetWeekSummaries(ctx context.Context, userID uuid.UUID, since time.Time) ([]*models.WeekSummaryEntry, error)
}

// weeklyReviewAI is the subset of ClaudeService used by WeeklyReviewScheduler.
type weeklyReviewAI interface {
	GenerateWeeklyReview(ctx context.Context, input services.WeeklyReviewPromptInput) (*services.WeeklyReviewOutput, error)
}

// weeklyFreezeGranter is satisfied by *repositories.UserRepository.
type weeklyFreezeGranter interface {
	GrantWeeklyFreezes(ctx context.Context) error
}

// annualReviewRepo is the subset of AnnualReviewRepository used by YearInReviewScheduler.
type annualReviewRepo interface {
	Schedule(ctx context.Context, userID uuid.UUID, year int, scheduledAt time.Time) error
	PendingDue(ctx context.Context) ([]*models.AnnualReview, error)
	MarkCompleted(ctx context.Context, id uuid.UUID, narrative string, topEmotions, topTopics []string, moodArc []models.MonthlyMoodArcDay, entryCount, avgMood int) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

// annualReviewAnalysisRepo is the subset of AnalysisRepository used by YearInReviewScheduler.
type annualReviewAnalysisRepo interface {
	GetYearSummaries(ctx context.Context, userID uuid.UUID, yearStart, yearEnd time.Time) ([]*models.YearSummaryEntry, error)
}

// annualReviewAI is the subset of ClaudeService used by YearInReviewScheduler.
type annualReviewAI interface {
	GenerateYearInReview(ctx context.Context, input services.YearInReviewPromptInput) (*services.YearInReviewOutput, error)
}

// personExtractor is the subset of ClaudeService used by TranscriptionWorker for relationship extraction.
type personExtractor interface {
	ExtractPeople(ctx context.Context, transcript string) (*models.PersonExtractionOutput, error)
}

// personMentionStore is the subset of RelationshipRepository used by TranscriptionWorker.
type personMentionStore interface {
	UpsertPersonMentions(ctx context.Context, userID, entryID uuid.UUID, people []models.ExtractedPerson) error
}
