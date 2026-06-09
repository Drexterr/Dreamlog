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

	s, err := scanSession(r.db.QueryRow(ctx, q, userID, persona, snapshotJSON, billingPaise))
	if err != nil {
		return nil, fmt.Errorf("therapy.Create: %w", err)
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
		WHERE id = $1`
	_, err = r.db.Exec(ctx, q, id, a.SessionNarrative, a.MoodScore, toneJSON, a.Topics, a.KeyInsights)
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
