package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID             `json:"id"`
	EntryID   uuid.UUID             `json:"entry_id"`
	UserID    uuid.UUID             `json:"user_id"`
	TurnCount int                   `json:"turn_count"`
	IsClosed  bool                  `json:"is_closed"`
	Messages  []ConversationMessage `json:"messages,omitempty"`
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

type SendMessageInput struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}
