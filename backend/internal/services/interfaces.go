package services

import (
	"context"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// UserStore is the minimal user-repository interface required by AuthService.
// Satisfied by *repositories.UserRepository in production.
type UserStore interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByEmailIncDeleted(ctx context.Context, email string) (*models.User, error)
	CreateLocal(ctx context.Context, email, name, passwordHash string) (*models.User, error)
	GetPasswordHash(ctx context.Context, email string) (string, error)
	Reactivate(ctx context.Context, id uuid.UUID, name, passwordHash string) (*models.User, error)
}

// ConvRepository is the minimal conversation-repository interface required by ConversationService.
// Satisfied by *repositories.ConversationRepository in production.
type ConvRepository interface {
	GetOrCreate(ctx context.Context, entryID, userID uuid.UUID) (*models.Conversation, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Conversation, error)
	AddMessage(ctx context.Context, convID uuid.UUID, role, content string) (*models.Conversation, *models.ConversationMessage, error)
	ListMessages(ctx context.Context, convID uuid.UUID) ([]models.ConversationMessage, error)
}

// EntryStoreReader is the minimal entry-repository interface required by ConversationService.
// Satisfied by *repositories.EntryRepository in production.
type EntryStoreReader interface {
	GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Entry, error)
}

// AnalysisStoreReader is the minimal analysis-repository interface required by ConversationService.
// Satisfied by *repositories.AnalysisRepository in production.
type AnalysisStoreReader interface {
	GetByEntryID(ctx context.Context, entryID uuid.UUID) (*models.EntryAnalysis, error)
}
