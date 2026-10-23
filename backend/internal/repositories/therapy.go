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

type TherapyRepository struct {
	db *pgxpool.Pool
}

func NewTherapyRepository(db *pgxpool.Pool) *TherapyRepository {
	return &TherapyRepository{db: db}
}

const therapySessionColumns = `
	id, user_id, status, persona, started_at, expires_at, ended_at, duration_sec,
	turn_count, crisis_warnings, context_snapshot, post_session_summary,
	session_mood_score, session_emotional_tone, session_topics, session_key_insights,
	billing_amount_paise, created_at`

func scanSession(row pgx.Row) (*models.TherapySession, error) {
	s := &models.TherapySession{}
	var snapshotJSON []byte
	var toneJSON []byte
	err := row.Scan(
		&s.ID, &s.UserID, &s.Status, &s.Persona, &s.StartedAt, &s.ExpiresAt,
		&s.EndedAt, &s.DurationSec, &s.TurnCount, &s.CrisisWarnings,
		&snapshotJSON, &s.PostSessionSummary,
		&s.SessionMoodScore, &toneJSON, &s.SessionTopics, &s.SessionKeyInsights,
		&s.BillingAmountPaise, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(snapshotJSON, &s.ContextSnapshot); err != nil {
		return nil, fmt.Errorf("therapy: unmarshal context_snapshot: %w", err)
	}
	if toneJSON != nil {
		_ = json.Unmarshal(toneJSON, &s.SessionEmotionalTone)
	}
	s.TimeRemainingSec = computeTimeRemaining(s)
	return s, nil
}

func computeTimeRemaining(s *models.TherapySession) int {
	if s.Status != models.TherapyStatusActive {
		return 0
	}
	rem := time.Until(s.ExpiresAt)
	if rem < 0 {
		return 0
	}
	return int(rem.Seconds())
}

// Create inserts a new therapy session and returns it.
func (r *TherapyRepository) Create(ctx context.Context, userID uuid.UUID, persona models.TherapyPersona, snapshot models.TherapyContextSnapshot, billingPaise int) (*models.TherapySession, error) {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("therapy.Create: marshal snapshot: %w", err)
	}

	const q = `
		INSERT INTO therapy_sessions (user_id, persona, expires_at, context_snapshot, billing_amount_paise)
		VALUES ($1, $2, NOW() + INTERVAL '1 hour', $3, $4)
		RETURNING ` + therapySessionColumns

	s, err := scanSession(r.db.QueryRow(ctx, q, userID, persona, string(snapshotJSON), billingPaise))
	if err != nil {
		return nil, fmt.Errorf("therapy.Create: %w", err)
	}
	return s, nil
}

// CreateWithCredit starts a session paid with one prepaid therapy credit. The
// credit is consumed, the session created and the ledger written in ONE
// transaction: a failure anywhere rolls the credit back, and the conditional
// decrement (row-locked, re-checked by concurrent callers) means concurrent
// starts can never spend more credits than sessions they create. Returns
// models.ErrNoTherapyCredits when the balance is 0.
func (r *TherapyRepository) CreateWithCredit(ctx context.Context, userID uuid.UUID, persona models.TherapyPersona, snapshot models.TherapyContextSnapshot, billingPaise int) (*models.TherapySession, error) {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("therapy.CreateWithCredit: marshal snapshot: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("therapy.CreateWithCredit begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var remaining int
	err = tx.QueryRow(ctx,
		`UPDATE users SET therapy_credits = therapy_credits - 1, updated_at = NOW()
		 WHERE id = $1 AND therapy_credits > 0
		 RETURNING therapy_credits`, userID,
	).Scan(&remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, models.ErrNoTherapyCredits
	}
	if err != nil {
		return nil, fmt.Errorf("therapy.CreateWithCredit consume credit: %w", err)
	}

	const q = `
		INSERT INTO therapy_sessions (user_id, persona, expires_at, context_snapshot, billing_amount_paise)
		VALUES ($1, $2, NOW() + INTERVAL '1 hour', $3, $4)
		RETURNING ` + therapySessionColumns
	s, err := scanSession(tx.QueryRow(ctx, q, userID, persona, string(snapshotJSON), billingPaise))
	if err != nil {
		return nil, fmt.Errorf("therapy.CreateWithCredit create session: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO therapy_credit_ledger (user_id, delta, reason, session_id) VALUES ($1, -1, 'session', $2)`,
		userID, s.ID); err != nil {
		return nil, fmt.Errorf("therapy.CreateWithCredit ledger: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("therapy.CreateWithCredit commit: %w", err)
	}
	return s, nil
}

// GetByID returns a session, verifying user ownership. Returns nil if not found.
func (r *TherapyRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.TherapySession, error) {
	const q = `SELECT ` + therapySessionColumns + ` FROM therapy_sessions WHERE id = $1 AND user_id = $2`
	s, err := scanSession(r.db.QueryRow(ctx, q, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("therapy.GetByID: %w", err)
	}
	return s, nil
}

// ListByUser returns up to 20 sessions for a user, newest first.
func (r *TherapyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.TherapySession, error) {
	const q = `
		SELECT ` + therapySessionColumns + `
		FROM therapy_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 20`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("therapy.ListByUser: %w", err)
	}
	defer rows.Close()

	var sessions []*models.TherapySession
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("therapy.ListByUser scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// UpdateStatus sets the session status and optionally records ended_at + duration.
func (r *TherapyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TherapySessionStatus, endedAt *time.Time, durationSec *int) error {
	const q = `
		UPDATE therapy_sessions
		SET status = $2, ended_at = $3, duration_sec = $4
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, status, endedAt, durationSec)
	if err != nil {
		return fmt.Errorf("therapy.UpdateStatus: %w", err)
	}
	return nil
}

// IncrementCrisisWarning increments crisis_warnings and returns the updated count.
func (r *TherapyRepository) IncrementCrisisWarning(ctx context.Context, id uuid.UUID) (int, error) {
	const q = `
		UPDATE therapy_sessions SET crisis_warnings = crisis_warnings + 1 WHERE id = $1
		RETURNING crisis_warnings`
	var count int
	if err := r.db.QueryRow(ctx, q, id).Scan(&count); err != nil {
		return 0, fmt.Errorf("therapy.IncrementCrisisWarning: %w", err)
	}
	return count, nil
}

// PastCompletedSummaries returns up to limit post_session_summary values from the user's
// most recent completed sessions (oldest first among the N returned).
func (r *TherapyRepository) PastCompletedSummaries(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	const q = `
		SELECT post_session_summary
		FROM (
			SELECT post_session_summary, created_at
			FROM therapy_sessions
			WHERE user_id = $1
			  AND status = 'completed'
			  AND post_session_summary IS NOT NULL
			  AND post_session_summary <> ''
			ORDER BY created_at DESC
			LIMIT $2
		) sub
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("therapy.PastCompletedSummaries: %w", err)
	}
	defer rows.Close()

	var summaries []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("therapy.PastCompletedSummaries scan: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// IncrementTurn increments the turn count and returns the updated count.
func (r *TherapyRepository) IncrementTurn(ctx context.Context, id uuid.UUID) (int, error) {
	const q = `
		UPDATE therapy_sessions SET turn_count = turn_count + 1 WHERE id = $1
		RETURNING turn_count`
	var count int
	if err := r.db.QueryRow(ctx, q, id).Scan(&count); err != nil {
		return 0, fmt.Errorf("therapy.IncrementTurn: %w", err)
	}
	return count, nil
}

// SetSessionAnalysis stores the full structured post-session analysis from Claude.
// This replaces SetPostSessionSummary - both the prose narrative and structured fields are stored atomically.
// Only a COMPLETED session takes an analysis: if a crisis hard stop landed while
// the summary was generated, the write is a no-op (N3, ADR-016).
func (r *TherapyRepository) SetSessionAnalysis(ctx context.Context, id uuid.UUID, a *models.TherapySessionAnalysis) error {
	toneJSON, err := json.Marshal(a.EmotionalTone)
	if err != nil {
		return fmt.Errorf("therapy.SetSessionAnalysis: marshal tone: %w", err)
	}
	const q = `
		UPDATE therapy_sessions
		SET post_session_summary    = $2,
		    session_mood_score      = $3,
		    session_emotional_tone  = $4,
		    session_topics          = $5,
		    session_key_insights    = $6
		WHERE id = $1 AND status = 'completed'`
	_, err = r.db.Exec(ctx, q, id, a.SessionNarrative, a.MoodScore, string(toneJSON), a.Topics, a.KeyInsights)
	if err != nil {
		return fmt.Errorf("therapy.SetSessionAnalysis: %w", err)
	}
	return nil
}

// SetPostSessionSummary is retained for use in crisis-detected sessions where only
// a plain text message (not a full Claude analysis) is available.
func (r *TherapyRepository) SetPostSessionSummary(ctx context.Context, id uuid.UUID, summary string) error {
	const q = `UPDATE therapy_sessions SET post_session_summary = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, summary)
	if err != nil {
		return fmt.Errorf("therapy.SetPostSessionSummary: %w", err)
	}
	return nil
}

// CountAll returns the total number of therapy sessions for a user (for first-session-free check).
func (r *TherapyRepository) CountAll(ctx context.Context, userID uuid.UUID) (int, error) {
	const q = `SELECT COUNT(*) FROM therapy_sessions WHERE user_id = $1`
	var count int
	if err := r.db.QueryRow(ctx, q, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("therapy.CountAll: %w", err)
	}
	return count, nil
}

// CountThisMonth returns sessions started in the current calendar month (for Pro billing).
func (r *TherapyRepository) CountThisMonth(ctx context.Context, userID uuid.UUID) (int, error) {
	const q = `
		SELECT COUNT(*) FROM therapy_sessions
		WHERE user_id = $1
		  AND started_at >= date_trunc('month', NOW())`
	var count int
	if err := r.db.QueryRow(ctx, q, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("therapy.CountThisMonth: %w", err)
	}
	return count, nil
}

// ── Messages ─────────────────────────────────────────────────────────────────

// AddMessage inserts a single message into the session.
func (r *TherapyRepository) AddMessage(ctx context.Context, sessionID uuid.UUID, role, content, inputMode string) (*models.TherapySessionMessage, error) {
	const q = `
		INSERT INTO therapy_session_messages (session_id, role, content, input_mode)
		VALUES ($1, $2, $3, $4)
		RETURNING id, session_id, role, content, input_mode, created_at`

	m := &models.TherapySessionMessage{}
	err := r.db.QueryRow(ctx, q, sessionID, role, content, inputMode).Scan(
		&m.ID, &m.SessionID, &m.Role, &m.Content, &m.InputMode, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("therapy.AddMessage: %w", err)
	}
	return m, nil
}

// ListMessages returns all messages for a session ordered oldest first.
func (r *TherapyRepository) ListMessages(ctx context.Context, sessionID uuid.UUID) ([]models.TherapySessionMessage, error) {
	const q = `
		SELECT id, session_id, role, content, input_mode, created_at
		FROM therapy_session_messages
		WHERE session_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("therapy.ListMessages: %w", err)
	}
	defer rows.Close()

	var msgs []models.TherapySessionMessage
	for rows.Next() {
		m := models.TherapySessionMessage{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.InputMode, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("therapy.ListMessages scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// ── Turn idempotency + conditional status transitions (N3/N4) ────────────────

const therapyMessageColumns = `id, session_id, role, content, input_mode, created_at`

func scanTherapyMessage(row pgx.Row) (*models.TherapySessionMessage, error) {
	m := &models.TherapySessionMessage{}
	if err := row.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.InputMode, &m.CreatedAt); err != nil {
		return nil, err
	}
	return m, nil
}

// FindTurn returns the user message stored for requestID and its reply, if
// any. user is nil when no request with that id was ever claimed.
func (r *TherapyRepository) FindTurn(ctx context.Context, sessionID, requestID uuid.UUID) (user, reply *models.TherapySessionMessage, err error) {
	user, err = scanTherapyMessage(r.db.QueryRow(ctx,
		`SELECT `+therapyMessageColumns+` FROM therapy_session_messages WHERE session_id = $1 AND request_id = $2`,
		sessionID, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("therapy.FindTurn: %w", err)
	}
	reply, err = scanTherapyMessage(r.db.QueryRow(ctx,
		`SELECT `+therapyMessageColumns+` FROM therapy_session_messages WHERE reply_to = $1`, user.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return user, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("therapy.FindTurn reply: %w", err)
	}
	return user, reply, nil
}

// ClaimTurn stores the user's message for requestID under the session row
// lock. Unless allowEnded is set (crisis turns, which must always be recorded)
// it only does so while the session is active, otherwise returning
// models.ErrTherapySessionNotActive. created=false means another request
// already claimed this request id; the stored message is returned.
func (r *TherapyRepository) ClaimTurn(ctx context.Context, sessionID, requestID uuid.UUID, content, inputMode string, allowEnded bool) (msg *models.TherapySessionMessage, created bool, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("therapy.ClaimTurn begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var status models.TherapySessionStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM therapy_sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&status); err != nil {
		return nil, false, fmt.Errorf("therapy.ClaimTurn lock: %w", err)
	}
	existing, err := scanTherapyMessage(tx.QueryRow(ctx,
		`SELECT `+therapyMessageColumns+` FROM therapy_session_messages WHERE session_id = $1 AND request_id = $2`,
		sessionID, requestID))
	if err == nil {
		return existing, false, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("therapy.ClaimTurn lookup: %w", err)
	}
	if !allowEnded && status != models.TherapyStatusActive {
		return nil, false, models.ErrTherapySessionNotActive
	}
	msg, err = scanTherapyMessage(tx.QueryRow(ctx, `
		INSERT INTO therapy_session_messages (session_id, role, content, input_mode, request_id)
		VALUES ($1, 'user', $2, $3, $4)
		RETURNING `+therapyMessageColumns,
		sessionID, content, inputMode, requestID))
	if err != nil {
		return nil, false, fmt.Errorf("therapy.ClaimTurn insert: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("therapy.ClaimTurn commit: %w", err)
	}
	return msg, true, nil
}

// ReleaseTurn removes a claimed user message that never got a reply (the turn
// failed before replying), so a retry with the same request id runs again.
func (r *TherapyRepository) ReleaseTurn(ctx context.Context, userMessageID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `
		DELETE FROM therapy_session_messages m
		WHERE m.id = $1
		  AND NOT EXISTS (SELECT 1 FROM therapy_session_messages r WHERE r.reply_to = m.id)`,
		userMessageID); err != nil {
		return fmt.Errorf("therapy.ReleaseTurn: %w", err)
	}
	return nil
}

// AddReply stores the assistant reply to a claimed user message. With
// requireActive it re-checks, under the session row lock, that the session is
// still active: a reply generated while a concurrent request hard-stopped or
// ended the session is discarded (models.ErrTherapySessionNotActive).
// countTurn increments turn_count in the same transaction. A turn is answered
// at most once - a second reply returns the first.
func (r *TherapyRepository) AddReply(ctx context.Context, sessionID, replyTo uuid.UUID, content, inputMode string, requireActive, countTurn bool) (msg *models.TherapySessionMessage, turnCount int, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("therapy.AddReply begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var status models.TherapySessionStatus
	if err := tx.QueryRow(ctx, `SELECT status, turn_count FROM therapy_sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&status, &turnCount); err != nil {
		return nil, 0, fmt.Errorf("therapy.AddReply lock: %w", err)
	}
	existing, err := scanTherapyMessage(tx.QueryRow(ctx,
		`SELECT `+therapyMessageColumns+` FROM therapy_session_messages WHERE reply_to = $1`, replyTo))
	if err == nil {
		return existing, turnCount, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, fmt.Errorf("therapy.AddReply lookup: %w", err)
	}
	if requireActive && status != models.TherapyStatusActive {
		return nil, 0, models.ErrTherapySessionNotActive
	}
	msg, err = scanTherapyMessage(tx.QueryRow(ctx, `
		INSERT INTO therapy_session_messages (session_id, role, content, input_mode, reply_to)
		VALUES ($1, 'assistant', $2, $3, $4)
		RETURNING `+therapyMessageColumns,
		sessionID, content, inputMode, replyTo))
	if err != nil {
		return nil, 0, fmt.Errorf("therapy.AddReply insert: %w", err)
	}
	if countTurn {
		if err := tx.QueryRow(ctx,
			`UPDATE therapy_sessions SET turn_count = turn_count + 1 WHERE id = $1 RETURNING turn_count`,
			sessionID).Scan(&turnCount); err != nil {
			return nil, 0, fmt.Errorf("therapy.AddReply turn: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("therapy.AddReply commit: %w", err)
	}
	return msg, turnCount, nil
}

// EndActive moves an ACTIVE session to status (completed / expired). It
// returns false when the session was no longer active - e.g. a concurrent
// crisis hard stop already ended it - so a stale caller never overwrites it.
func (r *TherapyRepository) EndActive(ctx context.Context, id uuid.UUID, status models.TherapySessionStatus, endedAt *time.Time, durationSec *int) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE therapy_sessions SET status = $2, ended_at = $3, duration_sec = $4
		WHERE id = $1 AND status = 'active'`,
		id, status, endedAt, durationSec)
	if err != nil {
		return false, fmt.Errorf("therapy.EndActive: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// MarkCrisis hard-stops a session (ADR-014). Crisis always wins: it applies
// even when a concurrent End or expiry got there first, and clears any
// post-session summary so a crisis transcript never enters session memory
// (ADR-016).
func (r *TherapyRepository) MarkCrisis(ctx context.Context, id uuid.UUID, endedAt time.Time, durationSec int) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE therapy_sessions
		SET status = 'crisis_detected', ended_at = $2, duration_sec = $3, post_session_summary = NULL
		WHERE id = $1 AND status <> 'crisis_detected'`,
		id, endedAt, durationSec); err != nil {
		return fmt.Errorf("therapy.MarkCrisis: %w", err)
	}
	return nil
}
