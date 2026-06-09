package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WeeklyReviewRepository struct {
	db *pgxpool.Pool
}

func NewWeeklyReviewRepository(db *pgxpool.Pool) *WeeklyReviewRepository {
	return &WeeklyReviewRepository{db: db}
}

// Schedule inserts a pending review row for the given user + week_start.
// Idempotent - ON CONFLICT DO NOTHING means calling it twice is safe.
func (r *WeeklyReviewRepository) Schedule(ctx context.Context, userID uuid.UUID, weekStart time.Time, scheduledAt time.Time) error {
	const q = `
		INSERT INTO weekly_reviews (user_id, week_start, scheduled_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, week_start) DO NOTHING`

	_, err := r.db.Exec(ctx, q, userID, weekStart.Format("2006-01-02"), scheduledAt)
	if err != nil {
		return fmt.Errorf("weeklyReviews.Schedule: %w", err)
	}
	return nil
}

// PendingDue returns up to 100 pending reviews whose scheduled_at has passed.
func (r *WeeklyReviewRepository) PendingDue(ctx context.Context) ([]*models.WeeklyReview, error) {
	const q = `
		SELECT id, user_id, week_start, narrative, top_emotions, mood_arc,
		       entry_count, status, scheduled_at, generated_at, created_at
		FROM weekly_reviews
		WHERE status = 'pending' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT 100`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("weeklyReviews.PendingDue: %w", err)
	}
	defer rows.Close()

	var reviews []*models.WeeklyReview
	for rows.Next() {
		rv, err := scanWeeklyReview(rows)
		if err != nil {
			return nil, fmt.Errorf("weeklyReviews.PendingDue scan: %w", err)
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

// MarkCompleted stores the generated content and marks the review completed.
func (r *WeeklyReviewRepository) MarkCompleted(
	ctx context.Context,
	id uuid.UUID,
	narrative string,
	topEmotions []string,
	moodArc []models.MoodArcDay,
	entryCount int,
) error {
	arcJSON, err := json.Marshal(moodArc)
	if err != nil {
		return fmt.Errorf("weeklyReviews.MarkCompleted marshal arc: %w", err)
	}

	const q = `
		UPDATE weekly_reviews
		SET status       = 'completed',
		    narrative    = $2,
		    top_emotions = $3,
		    mood_arc     = $4,
		    entry_count  = $5,
		    generated_at = NOW()
		WHERE id = $1`

	_, err = r.db.Exec(ctx, q, id, narrative, topEmotions, arcJSON, entryCount)
	if err != nil {
		return fmt.Errorf("weeklyReviews.MarkCompleted: %w", err)
	}
	return nil
}

// MarkFailed stores the error and marks the review failed.
func (r *WeeklyReviewRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `UPDATE weekly_reviews SET status = 'failed', error_msg = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, errMsg)
	if err != nil {
		return fmt.Errorf("weeklyReviews.MarkFailed: %w", err)
	}
	return nil
}

// GetLatestCompleted returns the most recent completed weekly review for a user, or nil.
func (r *WeeklyReviewRepository) GetLatestCompleted(ctx context.Context, userID uuid.UUID) (*models.WeeklyReview, error) {
	const q = `
		SELECT id, user_id, week_start, narrative, top_emotions, mood_arc,
		       entry_count, status, scheduled_at, generated_at, created_at
		FROM weekly_reviews
		WHERE user_id = $1 AND status = 'completed'
		ORDER BY week_start DESC
		LIMIT 1`

	rv, err := scanWeeklyReview(r.db.QueryRow(ctx, q, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("weeklyReviews.GetLatestCompleted: %w", err)
	}
	return rv, nil
}

// ListCompleted returns the most recent completed weekly reviews for a user.
func (r *WeeklyReviewRepository) ListCompleted(ctx context.Context, userID uuid.UUID, limit int) ([]*models.WeeklyReview, error) {
	const q = `
		SELECT id, user_id, week_start, narrative, top_emotions, mood_arc,
		       entry_count, status, scheduled_at, generated_at, created_at
		FROM weekly_reviews
		WHERE user_id = $1 AND status = 'completed'
		ORDER BY week_start DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("weeklyReviews.ListCompleted: %w", err)
	}
	defer rows.Close()

	var reviews []*models.WeeklyReview
	for rows.Next() {
		rv, err := scanWeeklyReview(rows)
		if err != nil {
			return nil, fmt.Errorf("weeklyReviews.ListCompleted scan: %w", err)
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

type weeklyReviewScanner interface {
	Scan(dest ...any) error
}

func scanWeeklyReview(row weeklyReviewScanner) (*models.WeeklyReview, error) {
	rv := &models.WeeklyReview{}
	var arcJSON []byte

	err := row.Scan(
		&rv.ID, &rv.UserID, &rv.WeekStart, &rv.Narrative,
		&rv.TopEmotions, &arcJSON,
		&rv.EntryCount, &rv.Status, &rv.ScheduledAt, &rv.GeneratedAt, &rv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(arcJSON) > 0 {
		if err := json.Unmarshal(arcJSON, &rv.MoodArc); err != nil {
			return nil, fmt.Errorf("scanWeeklyReview unmarshal arc: %w", err)
		}
	}

	return rv, nil
}
