package repositories

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JourneyRepository struct {
	db *pgxpool.Pool
}

func NewJourneyRepository(db *pgxpool.Pool) *JourneyRepository {
	return &JourneyRepository{db: db}
}

// CreateSession inserts a new journey_sessions row and its steps.
func (r *JourneyRepository) CreateSession(
	ctx context.Context,
	userID uuid.UUID,
	journeyID string,
	prompts []string,
) (*models.JourneySession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("journey.CreateSession begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var s models.JourneySession
	err = tx.QueryRow(ctx, `
		INSERT INTO journey_sessions (user_id, journey_id, total_steps)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, journey_id, current_step, total_steps, status, created_at, updated_at`,
		userID, journeyID, len(prompts),
	).Scan(&s.ID, &s.UserID, &s.JourneyID, &s.CurrentStep, &s.TotalSteps, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("journey.CreateSession insert: %w", err)
	}

	for i, prompt := range prompts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO journey_session_steps (session_id, step_index, prompt)
			VALUES ($1, $2, $3)`, s.ID, i, prompt); err != nil {
			return nil, fmt.Errorf("journey.CreateSession step %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("journey.CreateSession commit: %w", err)
	}
	return &s, nil
}

// GetSession returns a session with its steps, ownership-checked by userID.
func (r *JourneyRepository) GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.JourneySession, error) {
	var s models.JourneySession
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, journey_id, current_step, total_steps, status, created_at, updated_at
		FROM journey_sessions
		WHERE id = $1 AND user_id = $2`, sessionID, userID,
	).Scan(&s.ID, &s.UserID, &s.JourneyID, &s.CurrentStep, &s.TotalSteps, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("journey.GetSession: %w", err)
	}

	steps, err := r.getSteps(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	s.Steps = steps
	return &s, nil
}

// ListSessions returns in-progress and recently completed sessions, newest first.
func (r *JourneyRepository) ListSessions(ctx context.Context, userID uuid.UUID) ([]*models.JourneySession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, journey_id, current_step, total_steps, status, created_at, updated_at
		FROM journey_sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT 20`, userID)
	if err != nil {
		return nil, fmt.Errorf("journey.ListSessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.JourneySession
	for rows.Next() {
		s := &models.JourneySession{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.JourneyID, &s.CurrentStep, &s.TotalSteps, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("journey.ListSessions scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// AdvanceStep records the entry_id for the given step and bumps current_step.
// If all steps are filled, it marks the session completed.
func (r *JourneyRepository) AdvanceStep(
	ctx context.Context,
	sessionID uuid.UUID,
	stepIndex int,
	entryID uuid.UUID,
	nextStep int,
	done bool,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("journey.AdvanceStep begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE journey_session_steps
		SET entry_id = $1
		WHERE session_id = $2 AND step_index = $3`,
		entryID, sessionID, stepIndex); err != nil {
		return fmt.Errorf("journey.AdvanceStep update step: %w", err)
	}

	status := models.JourneyStatusInProgress
	if done {
		status = models.JourneyStatusCompleted
	}

	if _, err := tx.Exec(ctx, `
		UPDATE journey_sessions
		SET current_step = $1, status = $2, updated_at = NOW()
		WHERE id = $3`,
		nextStep, status, sessionID); err != nil {
		return fmt.Errorf("journey.AdvanceStep update session: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *JourneyRepository) getSteps(ctx context.Context, sessionID uuid.UUID) ([]models.JourneyStep, error) {
	rows, err := r.db.Query(ctx, `
		SELECT step_index, prompt, entry_id
		FROM journey_session_steps
		WHERE session_id = $1
		ORDER BY step_index ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("journey.getSteps: %w", err)
	}
	defer rows.Close()

	var steps []models.JourneyStep
	for rows.Next() {
		var js models.JourneyStep
		if err := rows.Scan(&js.StepIndex, &js.Prompt, &js.EntryID); err != nil {
			return nil, fmt.Errorf("journey.getSteps scan: %w", err)
		}
		js.Completed = js.EntryID != nil
		steps = append(steps, js)
	}
	return steps, rows.Err()
}
