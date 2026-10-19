package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrConversationClosed is returned when a message is sent to a conversation
// that is closed or has used all MaxConversationTurns user turns (ADR-006).
var ErrConversationClosed = errors.New("conversation is closed")

// crisisScreener is the subset of CrisisDetector used by ConversationService.
type crisisScreener interface {
	Screen(ctx context.Context, transcript, country string) (*CrisisResult, error)
}

type ConversationService struct {
	convRepo     ConvRepository
	entryRepo    EntryStoreReader
	analysisRepo AnalysisStoreReader
	claude       *ClaudeService
	crisis       crisisScreener
	log          *zap.Logger
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
		log:          zap.NewNop(),
	}
}

// WithCrisisScreener enables crisis screening of every user message (ADR-002 /
// ADR-013: first-person journaling surfaces are always screened). Production
// wiring must set it.
func (s *ConversationService) WithCrisisScreener(c crisisScreener, log *zap.Logger) *ConversationService {
	s.crisis = c
	if log != nil {
		s.log = log
	}
	return s
}

// GetOrCreate returns the conversation for an entry, creating it if needed.
// Only completed, non-crisis entries can be followed up on: a crisis entry's
// "reflection" is a resource message, not something to chat about.
func (s *ConversationService) GetOrCreate(ctx context.Context, entryID, userID uuid.UUID) (*models.Conversation, error) {
	entry, err := s.entryRepo.GetByIDInternal(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.GetOrCreate entry: %w", err)
	}
	if entry == nil || entry.UserID != userID || entry.Status != models.EntryStatusCompleted {
		return nil, apierr.NotFound("entry")
	}
	analysis, err := s.analysisRepo.GetByEntryID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.GetOrCreate analysis: %w", err)
	}
	if analysis == nil || analysis.IsCrisis {
		return nil, apierr.NotFound("entry")
	}

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
// country (ISO 3166-1 alpha-2, may be empty) localises crisis resources.
func (s *ConversationService) SendMessage(ctx context.Context, convID, userID uuid.UUID, content, country string) (*models.Conversation, error) {
	// Load conversation (ownership check).
	conv, err := s.convRepo.GetByID(ctx, convID, userID)
	if err != nil || conv == nil {
		return nil, fmt.Errorf("convSvc.SendMessage: conversation not found")
	}
	if conv.IsClosed || conv.TurnCount >= models.MaxConversationTurns {
		return nil, ErrConversationClosed
	}

	// Crisis screening runs on every user message, before anything reaches
	// the reflection model. Screener errors fail safe (ADR-002).
	if s.crisis != nil {
		result, serr := s.crisis.Screen(ctx, content, country)
		if serr != nil {
			s.log.Warn("convSvc: crisis screener error, failing safe to crisis (ADR-002)", zap.Error(serr))
			result = &CrisisResult{Detected: true, Response: CrisisResponse(country)}
		}
		if result.Detected {
			return s.respondToCrisis(ctx, convID, content, result.Response)
		}
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
			// Skip - this is the opening question, already passed to Claude separately.
			continue
		}
		history = append(history, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	// Save the user message first. The repository claims the turn atomically
	// (turn_count < max AND NOT is_closed), so concurrent requests can never
	// exceed the cap - a lost race returns ErrConversationClosed here, before
	// any Claude call is made.
	conv, _, err = s.convRepo.AddMessage(ctx, convID, "user", content)
	if err != nil {
		if errors.Is(err, models.ErrConversationTurnUnavailable) {
			return nil, ErrConversationClosed
		}
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

// respondToCrisis stores the user's message and the crisis-resource reply,
// closes the conversation and flags the response so the client renders
// hotline cards. The reflection model is never called.
func (s *ConversationService) respondToCrisis(ctx context.Context, convID uuid.UUID, content, response string) (*models.Conversation, error) {
	if _, _, err := s.convRepo.AddMessage(ctx, convID, "user", content); err != nil && !errors.Is(err, models.ErrConversationTurnUnavailable) {
		return nil, fmt.Errorf("convSvc.SendMessage: save crisis user msg: %w", err)
	}
	if _, _, err := s.convRepo.AddMessage(ctx, convID, "assistant", response); err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: save crisis reply: %w", err)
	}
	conv, err := s.convRepo.Close(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: close after crisis: %w", err)
	}
	msgs, err := s.convRepo.ListMessages(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("convSvc.SendMessage: reload messages: %w", err)
	}
	conv.Messages = msgs
	conv.IsCrisis = true
	return conv, nil
}
