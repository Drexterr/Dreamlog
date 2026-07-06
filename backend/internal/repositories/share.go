package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShareRepository struct {
	db *pgxpool.Pool
}

func NewShareRepository(db *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{db: db}
}

// Create inserts a new share link row.
func (r *ShareRepository) Create(ctx context.Context, in models.CreateShareLinkInput) (*models.ShareLink, error) {
	const q = `
		INSERT INTO share_links (user_id, token, passcode_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, token, passcode_hash, expires_at, revoked, created_at`

	var sl models.ShareLink
	err := r.db.QueryRow(ctx, q, in.UserID, in.Token, in.PasscodeHash, in.ExpiresAt).
		Scan(&sl.ID, &sl.UserID, &sl.Token, &sl.PasscodeHash, &sl.ExpiresAt, &sl.Revoked, &sl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// GetByToken fetches an active (not revoked, not expired) share link.
func (r *ShareRepository) GetByToken(ctx context.Context, token string) (*models.ShareLink, error) {
	const q = `
		SELECT id, user_id, token, passcode_hash, expires_at, revoked, failed_attempts, locked_until, created_at
		FROM share_links
		WHERE token = $1 AND revoked = FALSE AND expires_at > NOW()`

	var sl models.ShareLink
	err := r.db.QueryRow(ctx, q, token).
		Scan(&sl.ID, &sl.UserID, &sl.Token, &sl.PasscodeHash, &sl.ExpiresAt, &sl.Revoked, &sl.FailedAttempts, &sl.LockedUntil, &sl.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// RecordFailedPasscode increments the failed-attempt counter for a link and,
// once maxAttempts is reached, sets locked_until to now + lockDuration. It
// returns the resulting locked_until (nil when not yet locked) so the caller
// can surface the lockout. The attempt counter resets to 0 when a lock is set,
// so each lock window requires a fresh maxAttempts failures.
func (r *ShareRepository) RecordFailedPasscode(ctx context.Context, id uuid.UUID, maxAttempts int, lockDuration time.Duration) (*time.Time, error) {
	// In Postgres every SET expression reads the OLD row, so "failed_attempts + 1"
	// consistently refers to the pre-update value in both branches. On reaching
	// the threshold we lock and reset the counter to 0 (fresh window per lock).
	// make_interval(secs => $3) builds the lock window from a plain seconds value
	// (Go's Duration.String(), e.g. "15m0s", is NOT valid Postgres interval input).
	const q = `
		UPDATE share_links
		SET failed_attempts = CASE
		        WHEN failed_attempts + 1 >= $2 THEN 0
		        ELSE failed_attempts + 1
		    END,
		    locked_until = CASE
		        WHEN failed_attempts + 1 >= $2 THEN NOW() + make_interval(secs => $3)
		        ELSE locked_until
		    END
		WHERE id = $1
		RETURNING locked_until`

	var lockedUntil *time.Time
	err := r.db.QueryRow(ctx, q, id, maxAttempts, lockDuration.Seconds()).Scan(&lockedUntil)
	if err != nil {
		return nil, err
	}
	return lockedUntil, nil
}

// ResetPasscodeAttempts clears the failed-attempt counter and any lock after a
// successful passcode validation.
func (r *ShareRepository) ResetPasscodeAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE share_links SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, id)
	return err
}

// ListByUser returns all active share links for a user (newest first).
func (r *ShareRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.ShareLink, error) {
	const q = `
		SELECT id, user_id, token, passcode_hash, expires_at, revoked, created_at
		FROM share_links
		WHERE user_id = $1 AND revoked = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.ShareLink
	for rows.Next() {
		var sl models.ShareLink
		if err := rows.Scan(&sl.ID, &sl.UserID, &sl.Token, &sl.PasscodeHash, &sl.ExpiresAt, &sl.Revoked, &sl.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, &sl)
	}
	return links, rows.Err()
}

// CountMonthlyByUser returns the number of share links created by the user this calendar month.
func (r *ShareRepository) CountMonthlyByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM share_links
		WHERE user_id = $1
		  AND DATE_TRUNC('month', created_at AT TIME ZONE 'UTC') = DATE_TRUNC('month', NOW() AT TIME ZONE 'UTC')`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Revoke marks a share link as revoked.
func (r *ShareRepository) Revoke(ctx context.Context, id, userID uuid.UUID) error {
	const q = `UPDATE share_links SET revoked = TRUE WHERE id = $1 AND user_id = $2`
	tag, err := r.db.Exec(ctx, q, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("share link not found")
	}
	return nil
}

// ShareView fetches the data exposed via a share link: user name, 30d mood arc,
// per-entry AI summaries (no raw transcripts or reflections).
func (r *ShareRepository) ShareView(ctx context.Context, userID uuid.UUID) (*models.ShareLinkView, error) {
	// User display name
	var userName string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(preferred_name, name, email) FROM users WHERE id = $1`, userID).
		Scan(&userName)
	if err != nil {
		return nil, err
	}

	// 30-day mood arc (one point per day, excluding crisis)
	const arcQ = `
		SELECT TO_CHAR(DATE(e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
		       ROUND(AVG(ea.mood_score))::int AS avg_mood
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND e.created_at >= NOW() - INTERVAL '30 days'
		GROUP BY day
		ORDER BY day ASC`

	arcRows, err := r.db.Query(ctx, arcQ, userID)
	if err != nil {
		return nil, err
	}
	defer arcRows.Close()

	var arc []models.MoodArcDay
	var moodSum, moodCount int
	for arcRows.Next() {
		var d models.MoodArcDay
		if err := arcRows.Scan(&d.Date, &d.AvgMood); err != nil {
			return nil, err
		}
		arc = append(arc, d)
		moodSum += d.AvgMood
		moodCount++
	}
	if err := arcRows.Err(); err != nil {
		return nil, err
	}

	var avgMood *int
	if moodCount > 0 {
		v := moodSum / moodCount
		avgMood = &v
	}

	// Top emotions across the 30-day window
	const emoQ = `
		SELECT emotion_text, COUNT(*) AS cnt
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id,
		     LATERAL jsonb_array_elements(ea.emotional_tone::jsonb) AS et,
		     LATERAL (SELECT et->>'emotion' AS emotion_text) AS em
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND e.created_at >= NOW() - INTERVAL '30 days'
		  AND (et->>'intensity')::float >= 0.4
		GROUP BY emotion_text
		ORDER BY cnt DESC
		LIMIT 3`

	emoRows, err := r.db.Query(ctx, emoQ, userID)
	if err != nil {
		return nil, err
	}
	defer emoRows.Close()

	var topEmotions []string
	for emoRows.Next() {
		var em string
		var cnt int
		if err := emoRows.Scan(&em, &cnt); err != nil {
			return nil, err
		}
		topEmotions = append(topEmotions, em)
	}

	// Per-entry summaries - summary only, no transcript or reflection
	const sumQ = `
		SELECT TO_CHAR(DATE(e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
		       ea.summary,
		       ea.mood_score,
		       ea.topics
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND e.created_at >= NOW() - INTERVAL '30 days'
		ORDER BY e.created_at DESC
		LIMIT 30`

	sumRows, err := r.db.Query(ctx, sumQ, userID)
	if err != nil {
		return nil, err
	}
	defer sumRows.Close()

	var summaries []models.EntrySummary
	for sumRows.Next() {
		var s models.EntrySummary
		if err := sumRows.Scan(&s.Date, &s.Summary, &s.MoodScore, &s.Topics); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	if err := sumRows.Err(); err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(72 * time.Hour)
	return &models.ShareLinkView{
		UserName:    userName,
		Period:      "30d",
		MoodArc:     arc,
		AvgMood:     avgMood,
		TopEmotions: topEmotions,
		Summaries:   summaries,
		ExpiresAt:   expiresAt,
	}, nil
}
