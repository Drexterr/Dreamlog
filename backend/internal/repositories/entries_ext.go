package repositories

// Phase 2 extensions to EntryRepository (separate file to avoid touching Phase 1 code).

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// ListCompletedBefore returns up to limit completed entries created before the given time,
// newest first. Used by context builder to get historical summaries.
func (r *EntryRepository) ListCompletedBefore(ctx context.Context, userID uuid.UUID, before time.Time, limit int) ([]*models.Entry, error) {
	const q = `
		SELECT id, user_id, audio_key, audio_size_bytes, duration_sec,
		       status, mode, transcript, language, error_msg, retry_count, created_at, updated_at
		FROM entries
		WHERE user_id = $1
		  AND status = 'completed'
		  AND created_at < $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, q, userID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("entries.ListCompletedBefore: %w", err)
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
			return nil, fmt.Errorf("entries.ListCompletedBefore scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// SearchEntries performs full-text search on transcripts for a given user.
func (r *EntryRepository) SearchEntries(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*models.Entry, error) {
	const q = `
		SELECT id, user_id, audio_key, audio_size_bytes, duration_sec,
		       status, mode, transcript, language, error_msg, retry_count, created_at, updated_at
		FROM entries
		WHERE user_id = $1
		  AND search_vector @@ plainto_tsquery('english', $2)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $2)) DESC,
		         created_at DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, q, userID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("entries.Search: %w", err)
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
			return nil, fmt.Errorf("entries.Search scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
