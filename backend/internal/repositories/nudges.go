package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NudgeRepository struct {
	db *pgxpool.Pool
}

func NewNudgeRepository(db *pgxpool.Pool) *NudgeRepository {
	return &NudgeRepository{db: db}
}

// Create inserts a new nudge record.
func (r *NudgeRepository) Create(ctx context.Context, userID uuid.UUID, entryID *uuid.UUID, message string, scheduledAt time.Time, timezone string) (*models.Nudge, error) {
	return r.CreateWithType(ctx, userID, entryID, message, scheduledAt, timezone, "morning")
}

// CreateWithType inserts a nudge with an explicit type ("morning" | "reengagement").
func (r *NudgeRepository) CreateWithType(ctx context.Context, userID uuid.UUID, entryID *uuid.UUID, message string, scheduledAt time.Time, timezone, nudgeType string) (*models.Nudge, error) {
	const q = `
		INSERT INTO nudges (user_id, entry_id, message, scheduled_at, timezone, nudge_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, entry_id, message, scheduled_at, timezone, status, sent_at, created_at`

	n := &models.Nudge{}
	err := r.db.QueryRow(ctx, q, userID, entryID, message, scheduledAt, timezone, nudgeType).Scan(
		&n.ID, &n.UserID, &n.EntryID, &n.Message, &n.ScheduledAt, &n.Timezone, &n.Status, &n.SentAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("nudges.CreateWithType: %w", err)
	}
	return n, nil
}

// PendingDue returns all nudges that are past their scheduled_at time and still pending.
// Used by the cron scheduler.
func (r *NudgeRepository) PendingDue(ctx context.Context) ([]*models.Nudge, error) {
	const q = `
		SELECT id, user_id, entry_id, message, scheduled_at, timezone, status, sent_at, created_at
		FROM nudges
		WHERE status = 'pending' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT 500` // hard cap per batch

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("nudges.PendingDue: %w", err)
	}
	defer rows.Close()

	var nudges []*models.Nudge
	for rows.Next() {
		n := &models.Nudge{}
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.EntryID, &n.Message,
			&n.ScheduledAt, &n.Timezone, &n.Status, &n.SentAt, &n.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("nudges.PendingDue scan: %w", err)
		}
		nudges = append(nudges, n)
	}
	return nudges, rows.Err()
}

// MarkSent updates a nudge to sent status.
func (r *NudgeRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE nudges SET status = 'sent', sent_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// MarkFailed stores the error and marks the nudge failed.
func (r *NudgeRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `UPDATE nudges SET status = 'failed', error_msg = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, errMsg)
	return err
}

// UpsertDevice registers or updates a device FCM token for a user.
func (r *NudgeRepository) UpsertDevice(ctx context.Context, userID uuid.UUID, token, platform string) error {
	const q = `
		INSERT INTO user_devices (user_id, fcm_token, platform)
		VALUES ($1, $2, $3)
		ON CONFLICT (fcm_token) DO UPDATE
		    SET user_id = EXCLUDED.user_id,
		        platform = EXCLUDED.platform,
		        updated_at = NOW()`
	_, err := r.db.Exec(ctx, q, userID, token, platform)
	if err != nil {
		return fmt.Errorf("nudges.UpsertDevice: %w", err)
	}
	return nil
}

// LapsedUserAtNudgeHour returns user IDs + timezones for users who:
//   - have nudge_enabled = true
//   - have at least one FCM device token
//   - have at least one completed entry ever (active users)
//   - have NOT had a completed entry in the last lapseHours hours
//   - have NOT received a reengagement nudge in the last 23 hours
//   - whose current local hour matches their fcm_nudge_hour
//
// Called by the reengagement scheduler every minute.
type LapsedUser struct {
	UserID   uuid.UUID
	Timezone string
}

func (r *NudgeRepository) LapsedUsersAtNudgeHour(ctx context.Context, lapseHours int) ([]LapsedUser, error) {
	const q = `
		SELECT DISTINCT u.id, COALESCE(NULLIF(u.timezone, ''), 'UTC')
		FROM users u
		JOIN user_devices ud ON ud.user_id = u.id
		WHERE u.nudge_enabled = true
		  AND u.deleted_at IS NULL
		  -- at least one completed entry ever
		  AND EXISTS (
		      SELECT 1 FROM entries e
		      WHERE e.user_id = u.id AND e.status = 'completed'
		  )
		  -- no completed entry within the lapse window
		  AND NOT EXISTS (
		      SELECT 1 FROM entries e
		      WHERE e.user_id = u.id
		        AND e.status = 'completed'
		        AND e.created_at > NOW() - ($1 || ' hours')::INTERVAL
		  )
		  -- no reengagement nudge sent in last 23 hours (dedup)
		  AND NOT EXISTS (
		      SELECT 1 FROM nudges n
		      WHERE n.user_id = u.id
		        AND n.nudge_type = 'reengagement'
		        AND n.created_at > NOW() - INTERVAL '23 hours'
		  )
		  -- current local hour matches their configured nudge hour
		  AND EXTRACT(HOUR FROM NOW() AT TIME ZONE COALESCE(NULLIF(u.timezone, ''), 'UTC'))::int = u.fcm_nudge_hour`

	rows, err := r.db.Query(ctx, q, lapseHours)
	if err != nil {
		return nil, fmt.Errorf("nudges.LapsedUsersAtNudgeHour: %w", err)
	}
	defer rows.Close()

	var users []LapsedUser
	for rows.Next() {
		var u LapsedUser
		if err := rows.Scan(&u.UserID, &u.Timezone); err != nil {
			return nil, fmt.Errorf("nudges.LapsedUsersAtNudgeHour scan: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetDeviceTokens returns all FCM tokens for a user.
func (r *NudgeRepository) GetDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	const q = `SELECT fcm_token FROM user_devices WHERE user_id = $1`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("nudges.GetDeviceTokens: %w", err)
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
