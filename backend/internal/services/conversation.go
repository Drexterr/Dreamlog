package services

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

type ConversationService struct {
	convRepo     ConvRepository
	entryRepo    EntryStoreReader
	analysisRepo AnalysisStoreReader
	claude       *ClaudeService
}

func NewConversationService(
	convRepo ConvRepository,
	entryRepo EntryStoreReader,
	analysisRepo AnalysisStoreReader,
	claude *ClaudeService,
) *ConversationService {
	return &ConversationService{
		convRepo:     convRepo,
		entryRepo:    entryRepo,
		analysisRepo: analysisRepo,
		claude:       claude,
	}
}

// GetOrCreate returns the conversation for an entry, creating it if needed.
func (s *ConversationService) GetOrCreate(ctx context.Context, entryID, userID uuid.UUID) (*models.Conversation, error) {
	conv, err := s.convRepo.GetOrCreate(ctx, entryID, userID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.GetOrCreate: %w", err)
	}

	msgs, err := s.convRepo.ListMessages(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.GetOrCreate messages: %w", err)
	}
	conv.Messages = msgs
	return conv, nil
}

// SendMessage handles one user turn in the follow-up conversation.
// Returns the updated conversation (with assistant reply appended).
func (s *ConversationService) SendMessage(ctx context.Context, convID, userID uuid.UUID, content string) (*models.Conversation, error) {
	// Load conversation (ownership check).
	conv, err := s.convRepo.GetByID(ctx, convID, userID)
	if err != nil || conv == nil {
		return nil, fmt.Errorf("convSvc.SendMessage: conversation not found")
	}
	if conv.IsClosed {
		return nil, fmt.Errorf("convSvc.SendMessage: conversation is closed")
	}
	if conv.TurnCount >= models.MaxConversationTurns {
		return nil, fmt.Errorf("convSvc.SendMessage: max turns reached")
	}

	// Load supporting data for Claude.
	entry, err := s.entryRepo.GetByIDInternal(ctx, conv.EntryID)
	if err != nil || entry == nil {
		return nil, fmt.Errorf("convSvc.SendMessage: entry not found")
	}
	analysis, err := s.analysisRepo.GetByEntryID(ctx, conv.EntryID)
	if err != nil || analysis == nil {
		return nil, fmt.Errorf("convSvc.SendMessage: analysis not found")
	}

	// Load prior messages to build history for Claude.
	prior, err := s.convRepo.ListMessages(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: list messages: %w", err)
	}

	// Determine the opening question (first assistant message, or the reflection).
	openingQuestion := analysis.Reflection
	history := make([]anthropicMessage, 0, len(prior))
	for i, m := range prior {
		if i == 0 && m.Role == "assistant" {
			// Skip — this is the opening question, already passed to Claude separately.
			continue
		}
		history = append(history, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	// Save the user message first.
	conv, _, err = s.convRepo.AddMessage(ctx, convID, "user", content)
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: save user msg: %w", err)
	}

	// Generate Claude reply.
	transcript := ""
	if entry.Transcript != nil {
		transcript = *entry.Transcript
	}
	reply, err := s.claude.GenerateFollowUp(ctx, FollowUpInput{
		OriginalTranscript: transcript,
		OriginalReflection: analysis.Reflection,
		OpeningQuestion:    openingQuestion,
		History:            history,
		UserMessage:        content,
	})
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: claude: %w", err)
	}

	// Save assistant reply.
	conv, _, err = s.convRepo.AddMessage(ctx, convID, "assistant", reply)
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: save assistant msg: %w", err)
	}

	// Reload messages for the response.
	msgs, err := s.convRepo.ListMessages(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: reload messages: %w", err)
	}
	conv.Messages = msgs
	return conv, nil
}
