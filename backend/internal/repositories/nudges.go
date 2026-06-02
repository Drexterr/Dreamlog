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
	const q = `
		INSERT INTO nudges (user_id, entry_id, message, scheduled_at, timezone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, entry_id, message, scheduled_at, timezone, status, sent_at, created_at`

	n := &models.Nudge{}
	err := r.db.QueryRow(ctx, q, userID, entryID, message, scheduledAt, timezone).Scan(
		&n.ID, &n.UserID, &n.EntryID, &n.Message, &n.ScheduledAt, &n.Timezone, &n.Status, &n.SentAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("nudges.Create: %w", err)
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
