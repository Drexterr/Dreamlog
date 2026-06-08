package handlers

import (
	"context"
	"io"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
)

// entryServicer is the minimal interface EntryHandler needs from EntryService.
// Satisfied by *services.EntryService in production.
type entryServicer interface {
	PresignUpload(ctx context.Context, userID uuid.UUID) (*models.PresignResponse, error)
	Create(ctx context.Context, userID uuid.UUID, input *models.CreateEntryInput, userCountry string) (*models.Entry, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*models.Entry, error)
	List(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.ListEntriesResponse, error)
}

// storageUploader is the minimal interface EntryHandler needs for the upload proxy.
// Satisfied by *services.StorageService in production.
type storageUploader interface {
	Upload(ctx context.Context, key string, r io.Reader) error
}

// entryQuerier is the minimal interface AnalysisHandler needs from EntryRepository.
// Satisfied by *repositories.EntryRepository in production.
type entryQuerier interface {
	GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Entry, error)
	List(ctx context.Context, opts repositories.ListEntriesOpts) ([]*models.Entry, int, error)
	SearchEntries(ctx context.Context, userID uuid.UUID, q string, limit int) ([]*models.Entry, error)
}

// analysisQuerier is the minimal interface AnalysisHandler and MoodHandler need from AnalysisRepository.
// Satisfied by *repositories.AnalysisRepository in production.
type analysisQuerier interface {
	GetByEntryID(ctx context.Context, entryID uuid.UUID) (*models.EntryAnalysis, error)
	MoodLast7Days(ctx context.Context, userID uuid.UUID) ([]*models.DailyMood, error)
	StreakInfo(ctx context.Context, userID uuid.UUID) (*models.StreakInfo, error)
	MoodHistory(ctx context.Context, userID uuid.UUID, days int) (*models.MoodHistoryResponse, error)
	EmotionPatterns(ctx context.Context, userID uuid.UUID, days int) (*models.PatternRadarResponse, error)
}

// deviceRegistrar is the minimal interface MoodHandler needs from NudgeRepository.
// Satisfied by *repositories.NudgeRepository in production.
type deviceRegistrar interface {
	UpsertDevice(ctx context.Context, userID uuid.UUID, token, platform string) error
}

// userProfiler is the minimal interface UserHandler needs from UserService.
// Satisfied by *services.UserService in production.
type userProfiler interface {
	UpdateProfile(ctx context.Context, userID uuid.UUID, input models.UpdateUserInput) (*models.User, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

// weeklyReviewListRepo is the minimal interface WeeklyReviewHandler needs from WeeklyReviewRepository.
// Satisfied by *repositories.WeeklyReviewRepository in production.
type weeklyReviewListRepo interface {
	GetLatestCompleted(ctx context.Context, userID uuid.UUID) (*models.WeeklyReview, error)
	ListCompleted(ctx context.Context, userID uuid.UUID, limit int) ([]*models.WeeklyReview, error)
}

// annualReviewListRepo is the minimal interface AnnualReviewHandler needs.
// Satisfied by *repositories.AnnualReviewRepository in production.
type annualReviewListRepo interface {
	GetLatestCompleted(ctx context.Context, userID uuid.UUID) (*models.AnnualReview, error)
	ListCompleted(ctx context.Context, userID uuid.UUID) ([]*models.AnnualReview, error)
}

// streakFreezer is the minimal interface MoodHandler needs for streak freeze operations.
// Satisfied by *repositories.UserRepository in production.
type streakFreezer interface {
	UseStreakFreeze(ctx context.Context, userID uuid.UUID, frozenDate time.Time) error
	StreakFreezeCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// entryQuotaChecker checks whether a user is allowed to create another entry this month.
// Satisfied by *services.SubscriptionService in production.
type entryQuotaChecker interface {
	CheckEntryQuota(ctx context.Context, userID uuid.UUID, plan models.Plan) error
}

// shareQuotaChecker checks whether a user is allowed to create another share link.
// Satisfied by *services.SubscriptionService in production.
type shareQuotaChecker interface {
	CheckShareQuota(ctx context.Context, userID uuid.UUID, plan models.Plan) error
}

// relationshipMapRepo is the minimal interface RelationshipHandler needs.
// Satisfied by *repositories.RelationshipRepository in production.
type relationshipMapRepo interface {
	GetMap(ctx context.Context, userID uuid.UUID) ([]*models.Person, error)
	GetDetail(ctx context.Context, personID, userID uuid.UUID) (*models.PersonDetail, error)
}

// lifeChapterRepo is the minimal interface LifeChapterHandler needs.
// Satisfied by *repositories.LifeChapterRepository in production.
type lifeChapterRepo interface {
	Create(ctx context.Context, userID uuid.UUID, input models.CreateChapterInput) (*models.LifeChapter, error)
	List(ctx context.Context, userID uuid.UUID) ([]*models.LifeChapter, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*models.LifeChapter, error)
	Update(ctx context.Context, id, userID uuid.UUID, input models.UpdateChapterInput) (*models.LifeChapter, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetDetail(ctx context.Context, id, userID uuid.UUID) (*models.ChapterDetail, error)
	GetEntriesInRange(ctx context.Context, userID uuid.UUID, startDate string, endDate *string) ([]*models.WeekSummaryEntry, error)
	StoreSummary(ctx context.Context, id, userID uuid.UUID, summary string) error
}
