package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID             `json:"id"`
	EntryID   uuid.UUID             `json:"entry_id"`
	UserID    uuid.UUID             `json:"user_id"`
	TurnCount int                   `json:"turn_count"`
	IsClosed  bool                  `json:"is_closed"`
	// IsCrisis is true once a user message in this conversation triggered
	// crisis screening: the reply holds crisis resources and the conversation
	// is closed. Persisted (migration 000046), so reopening the conversation
	// still renders hotline cards (N12).
	IsCrisis  bool                  `json:"is_crisis,omitempty"`
	Messages  []ConversationMessage `json:"messages"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

type ConversationMessage struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Role           string    `json:"role"` // "user" | "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

const MaxConversationTurns = 3

// ErrConversationTurnUnavailable is returned by the repository when a user
// turn cannot be claimed because the conversation is closed or at the cap.
var ErrConversationTurnUnavailable = errors.New("conversation turn unavailable")

type SendMessageInput struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}
