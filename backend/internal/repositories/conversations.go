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

type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// GetOrCreate returns an existing conversation for an entry, or creates one.
func (r *ConversationRepository) GetOrCreate(ctx context.Context, entryID, userID uuid.UUID) (*models.Conversation, error) {
	const upsertQ = `
		INSERT INTO conversations (entry_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (entry_id) DO UPDATE SET updated_at = NOW()
		RETURNING id, entry_id, user_id, turn_count, is_closed, created_at, updated_at`

	c := &models.Conversation{}
	err := r.db.QueryRow(ctx, upsertQ, entryID, userID).Scan(
		&c.ID, &c.EntryID, &c.UserID, &c.TurnCount, &c.IsClosed, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("conv.GetOrCreate: %w", err)
	}
	return c, nil
}

// GetByID returns a conversation by its own ID, verifying user ownership.
func (r *ConversationRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Conversation, error) {
	const q = `
		SELECT id, entry_id, user_id, turn_count, is_closed, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND user_id = $2`

	c := &models.Conversation{}
	err := r.db.QueryRow(ctx, q, id, userID).Scan(
		&c.ID, &c.EntryID, &c.UserID, &c.TurnCount, &c.IsClosed, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("conv.GetByID: %w", err)
	}
	return c, nil
}

// ListMessages returns all messages for a conversation ordered oldest first.
func (r *ConversationRepository) ListMessages(ctx context.Context, convID uuid.UUID) ([]models.ConversationMessage, error) {
	const q = `
		SELECT id, conversation_id, role, content, created_at
		FROM conversation_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, convID)
	if err != nil {
		return nil, fmt.Errorf("conv.ListMessages: %w", err)
	}
	defer rows.Close()

	var msgs []models.ConversationMessage
	for rows.Next() {
		m := models.ConversationMessage{}
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("conv.ListMessages scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// AddMessage inserts a message and increments the turn counter when role=user.
// Returns the updated conversation and new message, atomically.
func (r *ConversationRepository) AddMessage(ctx context.Context, convID uuid.UUID, role, content string) (*models.Conversation, *models.ConversationMessage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("conv.AddMessage: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert message.
	msg := &models.ConversationMessage{}
	const insertQ = `
		INSERT INTO conversation_messages (conversation_id, role, content)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, role, content, created_at`

	if err := tx.QueryRow(ctx, insertQ, convID, role, content).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt,
	); err != nil {
		return nil, nil, fmt.Errorf("conv.AddMessage insert msg: %w", err)
	}

	// Increment turn_count when the user speaks, and auto-close at MaxConversationTurns.
	var updateQ string
	if role == "user" {
		updateQ = `
			UPDATE conversations
			SET turn_count = turn_count + 1,
			    is_closed  = (turn_count + 1 >= $2),
			    updated_at = NOW()
			WHERE id = $1
			RETURNING id, entry_id, user_id, turn_count, is_closed, created_at, updated_at`
	} else {
		updateQ = `
			UPDATE conversations
			SET updated_at = NOW()
			WHERE id = $1
			RETURNING id, entry_id, user_id, turn_count, is_closed, created_at, updated_at`
	}

	conv := &models.Conversation{}
	var row pgx.Row
	if role == "user" {
		row = tx.QueryRow(ctx, updateQ, convID, models.MaxConversationTurns)
	} else {
		row = tx.QueryRow(ctx, updateQ, convID)
	}
	if err := row.Scan(
		&conv.ID, &conv.EntryID, &conv.UserID, &conv.TurnCount, &conv.IsClosed,
		&conv.CreatedAt, &conv.UpdatedAt,
	); err != nil {
		return nil, nil, fmt.Errorf("conv.AddMessage update conv: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("conv.AddMessage commit: %w", err)
	}
	return conv, msg, nil
}
