package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TherapistNotesRepository owns external_clients, client_sessions and
// therapist_keys. It only ever sees ciphertext for sensitive fields -
// encryption/decryption happens in the service layer.
type TherapistNotesRepository struct {
	db *pgxpool.Pool
}

func NewTherapistNotesRepository(db *pgxpool.Pool) *TherapistNotesRepository {
	return &TherapistNotesRepository{db: db}
}

// ── Wrapped data keys ────────────────────────────────────────────────────────

// GetWrappedDEK returns the therapist's wrapped data key, or nil if none exists.
func (r *TherapistNotesRepository) GetWrappedDEK(ctx context.Context, therapistID uuid.UUID) ([]byte, error) {
	var wrapped []byte
	err := r.db.QueryRow(ctx,
		`SELECT wrapped_dek FROM therapist_keys WHERE therapist_id = $1`, therapistID,
	).Scan(&wrapped)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("notes.GetWrappedDEK: %w", err)
	}
	return wrapped, nil
}

// InsertWrappedDEK stores a wrapped DEK. On a concurrent insert race the
// existing row wins and is returned, so both goroutines end up with one key.
func (r *TherapistNotesRepository) InsertWrappedDEK(ctx context.Context, therapistID uuid.UUID, wrapped []byte) ([]byte, error) {
	var stored []byte
	err := r.db.QueryRow(ctx, `
		INSERT INTO therapist_keys (therapist_id, wrapped_dek)
		VALUES ($1, $2)
		ON CONFLICT (therapist_id) DO UPDATE SET wrapped_dek = therapist_keys.wrapped_dek
		RETURNING wrapped_dek`,
		therapistID, wrapped,
	).Scan(&stored)
	if err != nil {
		return nil, fmt.Errorf("notes.InsertWrappedDEK: %w", err)
	}
	return stored, nil
}

// ── External clients ─────────────────────────────────────────────────────────

func (r *TherapistNotesRepository) CreateExternalClient(ctx context.Context, therapistID uuid.UUID, nameEnc []byte, role string) (*models.ExternalClientRow, error) {
	row := &models.ExternalClientRow{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO external_clients (therapist_id, name_enc, role)
		VALUES ($1, $2, $3)
		RETURNING id, therapist_id, name_enc, role, archived, created_at, updated_at`,
		therapistID, nameEnc, role,
	).Scan(&row.ID, &row.TherapistID, &row.NameEnc, &row.Role, &row.Archived, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("notes.CreateExternalClient: %w", err)
	}
	return row, nil
}

// ListExternalClients returns non-archived clients (or all when includeArchived)
// with per-client session stats, newest activity first.
func (r *TherapistNotesRepository) ListExternalClients(ctx context.Context, therapistID uuid.UUID, includeArchived bool) ([]*models.ExternalClientRow, error) {
	const q = `
		SELECT ec.id, ec.therapist_id, ec.name_enc, ec.role, ec.archived,
		       COUNT(cs.id)::INT AS session_count,
		       MAX(cs.created_at) AS last_session_at,
		       ec.created_at, ec.updated_at
		FROM external_clients ec
		LEFT JOIN client_sessions cs ON cs.external_client_id = ec.id
		WHERE ec.therapist_id = $1 AND (ec.archived = FALSE OR $2)
		GROUP BY ec.id
		ORDER BY MAX(cs.created_at) DESC NULLS LAST, ec.created_at DESC`

	rows, err := r.db.Query(ctx, q, therapistID, includeArchived)
	if err != nil {
		return nil, fmt.Errorf("notes.ListExternalClients: %w", err)
	}
	defer rows.Close()

	var out []*models.ExternalClientRow
	for rows.Next() {
		row := &models.ExternalClientRow{}
		if err := rows.Scan(&row.ID, &row.TherapistID, &row.NameEnc, &row.Role, &row.Archived,
			&row.SessionCount, &row.LastSessionAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("notes.ListExternalClients scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetExternalClient returns a single client scoped to the therapist, or nil.
func (r *TherapistNotesRepository) GetExternalClient(ctx context.Context, therapistID, clientID uuid.UUID) (*models.ExternalClientRow, error) {
	row := &models.ExternalClientRow{}
	err := r.db.QueryRow(ctx, `
		SELECT id, therapist_id, name_enc, role, archived, created_at, updated_at
		FROM external_clients
		WHERE id = $1 AND therapist_id = $2`,
		clientID, therapistID,
	).Scan(&row.ID, &row.TherapistID, &row.NameEnc, &row.Role, &row.Archived, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("notes.GetExternalClient: %w", err)
	}
	return row, nil
}

// UpdateExternalClient patches the provided fields. Nil pointers are left unchanged.
func (r *TherapistNotesRepository) UpdateExternalClient(ctx context.Context, therapistID, clientID uuid.UUID, nameEnc []byte, role *string, archived *bool) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE external_clients SET
			name_enc = COALESCE($3, name_enc),
			role     = COALESCE($4, role),
			archived = COALESCE($5, archived),
			updated_at = NOW()
		WHERE id = $1 AND therapist_id = $2`,
		clientID, therapistID, nameEnc, role, archived,
	)
	if err != nil {
		return false, fmt.Errorf("notes.UpdateExternalClient: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteExternalClient removes the client and (via FK cascade) all their sessions.
func (r *TherapistNotesRepository) DeleteExternalClient(ctx context.Context, therapistID, clientID uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM external_clients WHERE id = $1 AND therapist_id = $2`,
		clientID, therapistID,
	)
	if err != nil {
		return false, fmt.Errorf("notes.DeleteExternalClient: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ── Client sessions ──────────────────────────────────────────────────────────

// CreateSession inserts a session row. Exactly one of externalClientID /
// linkedUserID must be non-nil (enforced by a DB CHECK as the backstop).
func (r *TherapistNotesRepository) CreateSession(ctx context.Context, s *models.ClientSessionRow) (*models.ClientSessionRow, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO client_sessions
			(therapist_id, external_client_id, linked_user_id, session_date, status, image_key, raw_text_enc, bullets_enc)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		s.TherapistID, s.ExternalClientID, s.LinkedUserID, s.SessionDate, s.Status, s.ImageKey, s.RawTextEnc, s.BulletsEnc,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("notes.CreateSession: %w", err)
	}
	return s, nil
}

const sessionCols = `id, therapist_id, external_client_id, linked_user_id, session_date,
	status, image_key, raw_text_enc, bullets_enc, summary_enc, error_msg, retry_count, created_at, updated_at`

func scanClientSession(row pgx.Row) (*models.ClientSessionRow, error) {
	s := &models.ClientSessionRow{}
	err := row.Scan(&s.ID, &s.TherapistID, &s.ExternalClientID, &s.LinkedUserID, &s.SessionDate,
		&s.Status, &s.ImageKey, &s.RawTextEnc, &s.BulletsEnc, &s.SummaryEnc, &s.ErrorMsg, &s.RetryCount,
		&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetSession returns a session scoped to the therapist, or nil.
func (r *TherapistNotesRepository) GetSession(ctx context.Context, therapistID, sessionID uuid.UUID) (*models.ClientSessionRow, error) {
	q := fmt.Sprintf(`SELECT %s FROM client_sessions WHERE id = $1 AND therapist_id = $2`, sessionCols)
	s, err := scanClientSession(r.db.QueryRow(ctx, q, sessionID, therapistID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("notes.GetSession: %w", err)
	}
	return s, nil
}

// GetSessionByID returns a session without therapist scoping - worker use only
// (the worker authenticates jobs by queue provenance, not user identity).
func (r *TherapistNotesRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.ClientSessionRow, error) {
	q := fmt.Sprintf(`SELECT %s FROM client_sessions WHERE id = $1`, sessionCols)
	s, err := scanClientSession(r.db.QueryRow(ctx, q, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("notes.GetSessionByID: %w", err)
	}
	return s, nil
}

// ListSessionsForExternalClient returns sessions newest-first.
func (r *TherapistNotesRepository) ListSessionsForExternalClient(ctx context.Context, therapistID, clientID uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM client_sessions
		WHERE therapist_id = $1 AND external_client_id = $2
		ORDER BY session_date DESC, created_at DESC
		LIMIT $3`, sessionCols)
	return r.listSessions(ctx, q, therapistID, clientID, limit)
}

// ListSessionsForLinkedClient returns sessions about a linked app user, newest-first.
func (r *TherapistNotesRepository) ListSessionsForLinkedClient(ctx context.Context, therapistID, linkedUserID uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM client_sessions
		WHERE therapist_id = $1 AND linked_user_id = $2
		ORDER BY session_date DESC, created_at DESC
		LIMIT $3`, sessionCols)
	return r.listSessions(ctx, q, therapistID, linkedUserID, limit)
}

// ListRecentSessions returns the therapist's most recent sessions across all clients.
func (r *TherapistNotesRepository) ListRecentSessions(ctx context.Context, therapistID uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM client_sessions
		WHERE therapist_id = $1
		ORDER BY session_date DESC, created_at DESC
		LIMIT $2`, sessionCols)

	rows, err := r.db.Query(ctx, q, therapistID, limit)
	if err != nil {
		return nil, fmt.Errorf("notes.ListRecentSessions: %w", err)
	}
	defer rows.Close()
	return collectSessions(rows)
}

func (r *TherapistNotesRepository) listSessions(ctx context.Context, q string, therapistID, clientID uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	rows, err := r.db.Query(ctx, q, therapistID, clientID, limit)
	if err != nil {
		return nil, fmt.Errorf("notes.listSessions: %w", err)
	}
	defer rows.Close()
	return collectSessions(rows)
}

func collectSessions(rows pgx.Rows) ([]*models.ClientSessionRow, error) {
	var out []*models.ClientSessionRow
	for rows.Next() {
		s, err := scanClientSession(rows)
		if err != nil {
			return nil, fmt.Errorf("notes.collectSessions scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SetSessionProcessing transitions pending → processing (idempotency gate for
// the OCR worker, mirroring the entry pipeline).
func (r *TherapistNotesRepository) SetSessionProcessing(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE client_sessions SET status = 'processing', updated_at = NOW()
		WHERE id = $1 AND status = 'pending'`, sessionID)
	if err != nil {
		return false, fmt.Errorf("notes.SetSessionProcessing: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// CompleteSessionOCR stores encrypted OCR output, clears the image key, and
// marks the session completed.
func (r *TherapistNotesRepository) CompleteSessionOCR(ctx context.Context, sessionID uuid.UUID, rawTextEnc, bulletsEnc []byte) error {
	_, err := r.db.Exec(ctx, `
		UPDATE client_sessions SET
			status = 'completed', raw_text_enc = $2, bullets_enc = $3,
			image_key = NULL, error_msg = NULL, updated_at = NOW()
		WHERE id = $1`, sessionID, rawTextEnc, bulletsEnc)
	if err != nil {
		return fmt.Errorf("notes.CompleteSessionOCR: %w", err)
	}
	return nil
}

// SetSessionFailed records an error. Status returns to 'pending' while retries
// remain (so a retry can re-enter processing) and 'failed' once maxed out.
func (r *TherapistNotesRepository) SetSessionFailed(ctx context.Context, sessionID uuid.UUID, errMsg string, final bool) error {
	status := "pending"
	if final {
		status = "failed"
	}
	_, err := r.db.Exec(ctx, `
		UPDATE client_sessions SET
			status = $2, error_msg = $3, retry_count = retry_count + 1, updated_at = NOW()
		WHERE id = $1`, sessionID, status, errMsg)
	if err != nil {
		return fmt.Errorf("notes.SetSessionFailed: %w", err)
	}
	return nil
}

// UpdateSessionBullets replaces the encrypted bullets on a completed session.
func (r *TherapistNotesRepository) UpdateSessionBullets(ctx context.Context, therapistID, sessionID uuid.UUID, bulletsEnc []byte) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE client_sessions SET bullets_enc = $3, updated_at = NOW()
		WHERE id = $1 AND therapist_id = $2`, sessionID, therapistID, bulletsEnc)
	if err != nil {
		return false, fmt.Errorf("notes.UpdateSessionBullets: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateSessionSummary stores the encrypted AI summary.
func (r *TherapistNotesRepository) UpdateSessionSummary(ctx context.Context, therapistID, sessionID uuid.UUID, summaryEnc []byte) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE client_sessions SET summary_enc = $3, updated_at = NOW()
		WHERE id = $1 AND therapist_id = $2`, sessionID, therapistID, summaryEnc)
	if err != nil {
		return false, fmt.Errorf("notes.UpdateSessionSummary: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteSession removes a session scoped to the therapist. Returns the image
// key (if any) so the caller can delete the orphaned storage object.
func (r *TherapistNotesRepository) DeleteSession(ctx context.Context, therapistID, sessionID uuid.UUID) (imageKey *string, found bool, err error) {
	err = r.db.QueryRow(ctx, `
		DELETE FROM client_sessions WHERE id = $1 AND therapist_id = $2
		RETURNING image_key`, sessionID, therapistID,
	).Scan(&imageKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("notes.DeleteSession: %w", err)
	}
	return imageKey, true, nil
}

// ── Overview metrics ─────────────────────────────────────────────────────────

// Overview aggregates caseload metrics for the therapist dashboard.
func (r *TherapistNotesRepository) Overview(ctx context.Context, therapistID uuid.UUID) (*models.TherapistOverview, error) {
	o := &models.TherapistOverview{}

	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM external_clients WHERE therapist_id = $1 AND archived = FALSE),
			(SELECT COUNT(*) FROM client_therapist_links WHERE therapist_id = $1 AND status = 'active'),
			COUNT(*) FILTER (WHERE cs.created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE cs.created_at >= NOW() - INTERVAL '30 days'),
			COUNT(*),
			MAX(cs.created_at)
		FROM client_sessions cs
		WHERE cs.therapist_id = $1`,
		therapistID,
	).Scan(&o.ExternalClients, &o.LinkedClients, &o.SessionsThisWeek, &o.SessionsThisMonth, &o.TotalSessions, &o.LastSessionAt)
	if err != nil {
		return nil, fmt.Errorf("notes.Overview: %w", err)
	}
	return o, nil
}

// ── ToS acceptance (users + therapists) ──────────────────────────────────────

// AcceptUserTerms records the user's ToS acceptance.
func (r *TherapistNotesRepository) AcceptUserTerms(ctx context.Context, userID uuid.UUID, version string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET tos_accepted_at = NOW(), tos_version = $2, updated_at = NOW()
		WHERE id = $1`, userID, version)
	if err != nil {
		return fmt.Errorf("notes.AcceptUserTerms: %w", err)
	}
	return nil
}

// GetUserTerms returns the user's current acceptance state.
func (r *TherapistNotesRepository) GetUserTerms(ctx context.Context, userID uuid.UUID) (acceptedAt *time.Time, version *string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT tos_accepted_at, tos_version FROM users WHERE id = $1`, userID,
	).Scan(&acceptedAt, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("notes.GetUserTerms: %w", err)
	}
	return acceptedAt, version, nil
}

// AcceptTherapistClientConsent records the therapist's client-data
// responsibility acceptance.
func (r *TherapistNotesRepository) AcceptTherapistClientConsent(ctx context.Context, therapistID uuid.UUID, version string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE therapists SET client_consent_accepted_at = NOW(), client_consent_version = $2, updated_at = NOW()
		WHERE id = $1`, therapistID, version)
	if err != nil {
		return fmt.Errorf("notes.AcceptTherapistClientConsent: %w", err)
	}
	return nil
}

// TherapistClientConsentAccepted reports whether the therapist has accepted
// the client-data responsibility terms.
func (r *TherapistNotesRepository) TherapistClientConsentAccepted(ctx context.Context, therapistID uuid.UUID) (bool, error) {
	var acceptedAt *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT client_consent_accepted_at FROM therapists WHERE id = $1`, therapistID,
	).Scan(&acceptedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("notes.TherapistClientConsentAccepted: %w", err)
	}
	return acceptedAt != nil, nil
}
