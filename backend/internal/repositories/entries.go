package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EntryRepository struct {
	db *pgxpool.Pool
}

func NewEntryRepository(db *pgxpool.Pool) *EntryRepository {
	return &EntryRepository{db: db}
}

// Create inserts a new entry with status=pending.
func (r *EntryRepository) Create(ctx context.Context, userID uuid.UUID, audioKey string, sizeBytes int64, durationSec float64, mode models.EntryMode) (*models.Entry, error) {
	if !mode.Valid() {
		mode = models.EntryModeProcessing
	}
	const q = `
		INSERT INTO entries (user_id, audio_key, audio_size_bytes, duration_sec, status, mode)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, user_id, audio_key, audio_size_bytes, duration_sec,
		          status, mode, transcript, language, error_msg, retry_count, created_at, updated_at`

	e := &models.Entry{}
	err := r.db.QueryRow(ctx, q, userID, audioKey, sizeBytes, durationSec, string(mode)).Scan(
		&e.ID, &e.UserID, &e.AudioKey, &e.AudioSizeBytes, &e.DurationSec,
		&e.Status, &e.Mode, &e.Transcript, &e.Language, &e.ErrorMsg, &e.RetryCount,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("entries.Create: %w", err)
	}
	return e, nil
}

// GetByID returns an entry only if it belongs to the given user (ownership check).
func (r *EntryRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Entry, error) {
	const q = `
		SELECT id, user_id, audio_key, audio_size_bytes, duration_sec,
		       status, mode, transcript, language, error_msg, retry_count, created_at, updated_at
		FROM entries
		WHERE id = $1 AND user_id = $2`

	e := &models.Entry{}
	err := r.db.QueryRow(ctx, q, id, userID).Scan(
		&e.ID, &e.UserID, &e.AudioKey, &e.AudioSizeBytes, &e.DurationSec,
		&e.Status, &e.Mode, &e.Transcript, &e.Language, &e.ErrorMsg, &e.RetryCount,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("entries.GetByID: %w", err)
	}
	return e, nil
}

// GetByIDInternal bypasses user ownership - used by background worker.
func (r *EntryRepository) GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Entry, error) {
	const q = `
		SELECT id, user_id, audio_key, audio_size_bytes, duration_sec,
		       status, mode, transcript, language, error_msg, retry_count, created_at, updated_at
		FROM entries WHERE id = $1`

	e := &models.Entry{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&e.ID, &e.UserID, &e.AudioKey, &e.AudioSizeBytes, &e.DurationSec,
		&e.Status, &e.Mode, &e.Transcript, &e.Language, &e.ErrorMsg, &e.RetryCount,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("entries.GetByIDInternal: %w", err)
	}
	return e, nil
}

type ListEntriesOpts struct {
	UserID   uuid.UUID
	Page     int // 1-indexed
	PageSize int
}

// List returns paginated entries for a user, newest first.
func (r *EntryRepository) List(ctx context.Context, opts ListEntriesOpts) ([]*models.Entry, int, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 || opts.PageSize > 100 {
		opts.PageSize = 20
	}
	offset := (opts.Page - 1) * opts.PageSize

	const countQ = `SELECT COUNT(*) FROM entries WHERE user_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, opts.UserID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("entries.List count: %w", err)
	}

	const listQ = `
		SELECT id, user_id, audio_key, audio_size_bytes, duration_sec,
		       status, mode, transcript, language, error_msg, retry_count, created_at, updated_at
		FROM entries
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, listQ, opts.UserID, opts.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("entries.List query: %w", err)
	}
	defer rows.Close()

	var entries []*models.Entry
	for rows.Next() {
		e := &models.Entry{}
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.AudioKey, &e.AudioSizeBytes, &e.DurationSec,
			&e.Status, &e.Mode, &e.Transcript, &e.Language, &e.ErrorMsg, &e.RetryCount,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("entries.List scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("entries.List rows: %w", err)
	}
	return entries, total, nil
}

// SetProcessing transitions an entry to processing (idempotent check via CAS).
// Returns false if the entry is not in a state that allows processing.
func (r *EntryRepository) SetProcessing(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `
		UPDATE entries
		SET status = 'processing'
		WHERE id = $1 AND status IN ('pending', 'failed')
		RETURNING id`

	var dummy uuid.UUID
	err := r.db.QueryRow(ctx, q, id).Scan(&dummy)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("entries.SetProcessing: %w", err)
	}
	return true, nil
}

// SetCompleted stores the transcript and marks entry as completed.
func (r *EntryRepository) SetCompleted(ctx context.Context, id uuid.UUID, transcript, language string) error {
	const q = `
		UPDATE entries
		SET status = 'completed',
		    transcript = $2,
		    language = $3,
		    error_msg = NULL
		WHERE id = $1`

	_, err := r.db.Exec(ctx, q, id, transcript, language)
	if err != nil {
		return fmt.Errorf("entries.SetCompleted: %w", err)
	}
	return nil
}

// SetFailed increments retry_count and stores the error message.
func (r *EntryRepository) SetFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `
		UPDATE entries
		SET status = 'failed',
		    error_msg = $2,
		    retry_count = retry_count + 1
		WHERE id = $1`

	_, err := r.db.Exec(ctx, q, id, errMsg)
	if err != nil {
		return fmt.Errorf("entries.SetFailed: %w", err)
	}
	return nil
}
