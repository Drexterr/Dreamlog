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

type AnnualReviewRepository struct {
	db *pgxpool.Pool
}

func NewAnnualReviewRepository(db *pgxpool.Pool) *AnnualReviewRepository {
	return &AnnualReviewRepository{db: db}
}

// Schedule inserts a pending review row for the given user + year. Idempotent.
func (r *AnnualReviewRepository) Schedule(ctx context.Context, userID uuid.UUID, year int, scheduledAt time.Time) error {
	const q = `
		INSERT INTO annual_reviews (user_id, year, scheduled_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, year) DO NOTHING`

	_, err := r.db.Exec(ctx, q, userID, year, scheduledAt)
	if err != nil {
		return fmt.Errorf("annualReviews.Schedule: %w", err)
	}
	return nil
}

// PendingDue returns up to 50 pending reviews whose scheduled_at has passed.
func (r *AnnualReviewRepository) PendingDue(ctx context.Context) ([]*models.AnnualReview, error) {
	const q = `
		SELECT id, user_id, year, narrative, top_emotions, top_topics, mood_arc,
		       entry_count, avg_mood, status, scheduled_at, generated_at, created_at
		FROM annual_reviews
		WHERE status = 'pending' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT 50`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("annualReviews.PendingDue: %w", err)
	}
	defer rows.Close()

	var reviews []*models.AnnualReview
	for rows.Next() {
		rv, err := scanAnnualReview(rows)
		if err != nil {
			return nil, fmt.Errorf("annualReviews.PendingDue scan: %w", err)
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

// MarkCompleted stores the generated content and marks the review completed.
func (r *AnnualReviewRepository) MarkCompleted(
	ctx context.Context,
	id uuid.UUID,
	narrative string,
	topEmotions, topTopics []string,
	moodArc []models.MonthlyMoodArcDay,
	entryCount, avgMood int,
) error {
	arcJSON, err := json.Marshal(moodArc)
	if err != nil {
		return fmt.Errorf("annualReviews.MarkCompleted marshal arc: %w", err)
	}

	const q = `
		UPDATE annual_reviews
		SET status       = 'completed',
		    narrative    = $2,
		    top_emotions = $3,
		    top_topics   = $4,
		    mood_arc     = $5,
		    entry_count  = $6,
		    avg_mood     = $7,
		    generated_at = NOW()
		WHERE id = $1`

	_, err = r.db.Exec(ctx, q, id, narrative, topEmotions, topTopics, arcJSON, entryCount, avgMood)
	if err != nil {
		return fmt.Errorf("annualReviews.MarkCompleted: %w", err)
	}
	return nil
}

// MarkFailed stores the error and marks the review failed.
func (r *AnnualReviewRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `UPDATE annual_reviews SET status = 'failed', error_msg = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, errMsg)
	if err != nil {
		return fmt.Errorf("annualReviews.MarkFailed: %w", err)
	}
	return nil
}

// GetLatestCompleted returns the most recent completed annual review for a user, or nil.
func (r *AnnualReviewRepository) GetLatestCompleted(ctx context.Context, userID uuid.UUID) (*models.AnnualReview, error) {
	const q = `
		SELECT id, user_id, year, narrative, top_emotions, top_topics, mood_arc,
		       entry_count, avg_mood, status, scheduled_at, generated_at, created_at
		FROM annual_reviews
		WHERE user_id = $1 AND status = 'completed'
		ORDER BY year DESC
		LIMIT 1`

	rv, err := scanAnnualReview(r.db.QueryRow(ctx, q, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("annualReviews.GetLatestCompleted: %w", err)
	}
	return rv, nil
}

// ListCompleted returns all completed annual reviews for a user, newest first.
func (r *AnnualReviewRepository) ListCompleted(ctx context.Context, userID uuid.UUID) ([]*models.AnnualReview, error) {
	const q = `
		SELECT id, user_id, year, narrative, top_emotions, top_topics, mood_arc,
		       entry_count, avg_mood, status, scheduled_at, generated_at, created_at
		FROM annual_reviews
		WHERE user_id = $1 AND status = 'completed'
		ORDER BY year DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("annualReviews.ListCompleted: %w", err)
	}
	defer rows.Close()

	var reviews []*models.AnnualReview
	for rows.Next() {
		rv, err := scanAnnualReview(rows)
		if err != nil {
			return nil, fmt.Errorf("annualReviews.ListCompleted scan: %w", err)
		}
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

type annualReviewScanner interface {
	Scan(dest ...any) error
}

func scanAnnualReview(row annualReviewScanner) (*models.AnnualReview, error) {
	rv := &models.AnnualReview{}
	var arcJSON []byte

	err := row.Scan(
		&rv.ID, &rv.UserID, &rv.Year, &rv.Narrative,
		&rv.TopEmotions, &rv.TopTopics, &arcJSON,
		&rv.EntryCount, &rv.AvgMood, &rv.Status,
		&rv.ScheduledAt, &rv.GeneratedAt, &rv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(arcJSON) > 0 {
		if err := json.Unmarshal(arcJSON, &rv.MoodArc); err != nil {
			return nil, fmt.Errorf("scanAnnualReview unmarshal arc: %w", err)
		}
	}

	return rv, nil
}
