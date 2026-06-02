package services

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// therapyRepo is the minimal repository interface TherapyService needs.
type therapyRepo interface {
	Create(ctx context.Context, userID uuid.UUID, persona models.TherapyPersona, snapshot models.TherapyContextSnapshot, billingPaise int) (*models.TherapySession, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*models.TherapySession, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.TherapySession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.TherapySessionStatus, endedAt *time.Time, durationSec *int) error
	IncrementTurn(ctx context.Context, id uuid.UUID) (int, error)
	IncrementCrisisWarning(ctx context.Context, id uuid.UUID) (int, error)
	SetSessionAnalysis(ctx context.Context, id uuid.UUID, a *models.TherapySessionAnalysis) error
	SetPostSessionSummary(ctx context.Context, id uuid.UUID, summary string) error
	CountAll(ctx context.Context, userID uuid.UUID) (int, error)
	CountThisMonth(ctx context.Context, userID uuid.UUID) (int, error)
	PastCompletedSummaries(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
	AddMessage(ctx context.Context, sessionID uuid.UUID, role, content, inputMode string) (*models.TherapySessionMessage, error)
	ListMessages(ctx context.Context, sessionID uuid.UUID) ([]models.TherapySessionMessage, error)
}

// therapyAnalysisRepo is used to load the journal context snapshot.
type therapyAnalysisRepo interface {
	MoodAvg30Days(ctx context.Context, userID uuid.UUID) (*float64, error)
	RecentSummaries(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
	TopEmotions(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
	TopTopics(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
}

// therapyStorageClient downloads audio for Whisper transcription and deletes it after.
type therapyStorageClient interface {
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// TherapyService orchestrates therapy session lifecycle.
type TherapyService struct {
	repo         therapyRepo
	analysisRepo therapyAnalysisRepo
	claude       *ClaudeService
	transcription *TranscriptionService
	storage      therapyStorageClient
	crisis       *CrisisDetector
	stubBilling  bool // when true (dev), billing checks always pass
}

func NewTherapyService(
	repo therapyRepo,
	analysisRepo therapyAnalysisRepo,
	claude *ClaudeService,
	transcription *TranscriptionService,
	storage therapyStorageClient,
	crisis *CrisisDetector,
	stubBilling bool,
) *TherapyService {
	return &TherapyService{
		repo:          repo,
		analysisRepo:  analysisRepo,
		claude:        claude,
		transcription: transcription,
		storage:       storage,
		crisis:        crisis,
		stubBilling:   stubBilling,
	}
}

// StartSession creates a new therapy session for the user.
// Billing logic:
//   - First session ever → free (acquisition hook)
//   - Pro plan → up to 2 sessions/month free
//   - Otherwise → ₹499 (stub in dev: allow with billing_amount_paise set)
func (s *TherapyService) StartSession(ctx context.Context, userID uuid.UUID, userPlan models.Plan, persona models.TherapyPersona) (*models.TherapySession, error) {
	billingPaise, err := s.computeBilling(ctx, userID, userPlan)
	if err != nil {
		return nil, err
	}

	snapshot, _, err := s.loadContextSnapshot(ctx, userID)
	if err != nil {
		// Non-fatal: proceed without context rather than failing the session.
		snapshot = models.TherapyContextSnapshot{}
	}

	session, err := s.repo.Create(ctx, userID, persona, snapshot, billingPaise)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.StartSession: %w", err)
	}
	return session, nil
}

// SendMessage processes one user turn: crisis check → optional Whisper → Claude → store.
// Audio is deleted from storage immediately after transcription (ADR-005).
func (s *TherapyService) SendMessage(ctx context.Context, sessionID, userID uuid.UUID, req models.SendTherapyMessageRequest) (*models.SendTherapyMessageResponse, error) {
	session, err := s.repo.GetByID(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.SendMessage: %w", err)
	}
	if session == nil {
		return nil, ErrTherapyNotFound
	}
	if session.Status != models.TherapyStatusActive {
		return nil, ErrTherapyNotActive
	}
	if time.Now().After(session.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, session.ID, models.TherapyStatusExpired, nil, nil)
		return nil, ErrTherapyExpired
	}

	// Resolve user content: transcribe if voice input.
	userContent := req.Content
	if req.InputMode == "voice" {
		if req.AudioKey == "" {
			return nil, errTherapyMissingAudio
		}
		transcribed, transcribeErr := s.transcribeAudio(ctx, req.AudioKey)
		// Delete audio from storage regardless of transcription outcome (ADR-005).
		_ = s.storage.Delete(ctx, req.AudioKey)
		if transcribeErr != nil {
			return nil, fmt.Errorf("therapySvc.SendMessage transcribe: %w", transcribeErr)
		}
		userContent = transcribed
	}

	if userContent == "" {
		return nil, errTherapyEmptyContent
	}

	// Crisis detection runs on every message (ADR-013).
	crisisResult, err := s.crisis.Screen(ctx, userContent)
	if err != nil {
		// Screen absorbs errors internally and fails safe; this path is never reached.
		return nil, fmt.Errorf("therapySvc.SendMessage crisis: %w", err)
	}
	if crisisResult.Detected {
		return s.handleCrisis(ctx, session, userContent, req.InputMode, crisisResult.Response)
	}

	// Fetch message history for Claude context.
	history, err := s.repo.ListMessages(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.SendMessage history: %w", err)
	}

	// Build per-turn system prompt: persona + time-aware wind-down.
	timeRem := int(math.Max(0, time.Until(session.ExpiresAt).Seconds()))
	windDown := buildWindDownInstruction(timeRem)
	systemPrompt := buildTherapyModeSystemPrompt(
		snapshotToPromptContext(session.ContextSnapshot, ""),
		string(session.Persona),
		windDown,
	)

	// Build Claude message history.
	claudeHistory := make([]chatMessage, 0, len(history))
	for _, m := range history {
		claudeHistory = append(claudeHistory, chatMessage{Role: m.Role, Content: m.Content})
	}

	aiReply, err := s.claude.TherapyTurn(ctx, TherapyTurnInput{
		SystemPrompt: systemPrompt,
		History:      claudeHistory,
		UserMessage:  userContent,
	})
	if err != nil {
		return nil, fmt.Errorf("therapySvc.SendMessage claude: %w", err)
	}

	// Persist both messages.
	userMsg, err := s.repo.AddMessage(ctx, session.ID, "user", userContent, req.InputMode)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.SendMessage store user msg: %w", err)
	}
	assistantMsg, err := s.repo.AddMessage(ctx, session.ID, "assistant", aiReply, "text")
	if err != nil {
		return nil, fmt.Errorf("therapySvc.SendMessage store assistant msg: %w", err)
	}

	// Increment turn counter.
	newTurnCount, _ := s.repo.IncrementTurn(ctx, session.ID)

	return &models.SendTherapyMessageResponse{
		UserMessage:      *userMsg,
		AssistantMessage: *assistantMsg,
		SessionState: models.TherapySessionState{
			Status:           models.TherapyStatusActive,
			TurnCount:        newTurnCount,
			TimeRemainingSec: timeRem,
			IsCrisis:         false,
			CrisisWarnings:   session.CrisisWarnings,
		},
	}, nil
}

// handleCrisis implements the two-stage crisis response (ADR-014).
// First detection: de-escalate, keep session open, increment crisis_warnings.
// Second detection (or fail-safe path): hard stop with crisis resources.
func (s *TherapyService) handleCrisis(ctx context.Context, session *models.TherapySession, userContent, inputMode, crisisResponse string) (*models.SendTherapyMessageResponse, error) {
	if session.CrisisWarnings == 0 {
		// Stage 1: de-escalate. Build a grounding response instead of the crisis resource dump.
		deEscalationReply, err := s.claude.TherapyTurn(ctx, TherapyTurnInput{
			SystemPrompt: buildDeEscalationPrompt(),
			History:      nil,
			UserMessage:  userContent,
		})
		if err != nil {
			// If Claude is unreachable on de-escalation → fail safe to hard stop (ADR-014).
			return s.hardStopCrisis(ctx, session, userContent, inputMode, crisisResponse)
		}

		userMsg, _ := s.repo.AddMessage(ctx, session.ID, "user", userContent, inputMode)
		assistantMsg, _ := s.repo.AddMessage(ctx, session.ID, "assistant", deEscalationReply, "text")
		newWarnings, _ := s.repo.IncrementCrisisWarning(ctx, session.ID)

		if userMsg == nil {
			userMsg = &models.TherapySessionMessage{Role: "user", Content: userContent, InputMode: inputMode}
		}
		if assistantMsg == nil {
			assistantMsg = &models.TherapySessionMessage{Role: "assistant", Content: deEscalationReply, InputMode: "text"}
		}

		timeRem := int(math.Max(0, time.Until(session.ExpiresAt).Seconds()))
		return &models.SendTherapyMessageResponse{
			UserMessage:      *userMsg,
			AssistantMessage: *assistantMsg,
			SessionState: models.TherapySessionState{
				Status:           models.TherapyStatusActive,
				TurnCount:        session.TurnCount + 1,
				TimeRemainingSec: timeRem,
				IsCrisis:         true,
				CrisisWarnings:   newWarnings,
			},
		}, nil
	}

	// Stage 2: hard stop.
	return s.hardStopCrisis(ctx, session, userContent, inputMode, crisisResponse)
}

func (s *TherapyService) hardStopCrisis(ctx context.Context, session *models.TherapySession, userContent, inputMode, crisisResponse string) (*models.SendTherapyMessageResponse, error) {
	userMsg, _ := s.repo.AddMessage(ctx, session.ID, "user", userContent, inputMode)
	assistantMsg, _ := s.repo.AddMessage(ctx, session.ID, "assistant", crisisResponse, "system")

	now := time.Now()
	elapsed := int(now.Sub(session.StartedAt).Seconds())
	_ = s.repo.UpdateStatus(ctx, session.ID, models.TherapyStatusCrisisDetected, &now, &elapsed)

	if userMsg == nil {
		userMsg = &models.TherapySessionMessage{Role: "user", Content: userContent, InputMode: inputMode}
	}
	if assistantMsg == nil {
		assistantMsg = &models.TherapySessionMessage{Role: "assistant", Content: crisisResponse, InputMode: "system"}
	}

	return &models.SendTherapyMessageResponse{
		UserMessage:      *userMsg,
		AssistantMessage: *assistantMsg,
		SessionState: models.TherapySessionState{
			Status:           models.TherapyStatusCrisisDetected,
			TurnCount:        session.TurnCount,
			TimeRemainingSec: 0,
			IsCrisis:         true,
			CrisisWarnings:   session.CrisisWarnings + 1,
		},
	}, nil
}

// EndSession closes the session and generates a post-session summary.
func (s *TherapyService) EndSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.EndSessionResponse, error) {
	session, err := s.repo.GetByID(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.EndSession: %w", err)
	}
	if session == nil {
		return nil, ErrTherapyNotFound
	}
	if session.Status == models.TherapyStatusCompleted || session.Status == models.TherapyStatusExpired {
		return nil, ErrTherapyAlreadyEnded
	}

	now := time.Now()
	elapsed := int(now.Sub(session.StartedAt).Seconds())

	// Fetch messages for analysis generation.
	messages, _ := s.repo.ListMessages(ctx, session.ID)
	summaryLines := make([]string, 0, len(messages))
	for _, m := range messages {
		summaryLines = append(summaryLines, fmt.Sprintf("[%s]: %s", m.Role, m.Content))
	}

	analysis, analysisErr := s.claude.TherapySummary(ctx, TherapySummaryInput{Messages: summaryLines})

	if err := s.repo.UpdateStatus(ctx, session.ID, models.TherapyStatusCompleted, &now, &elapsed); err != nil {
		return nil, fmt.Errorf("therapySvc.EndSession update status: %w", err)
	}

	narrative := ""
	if analysisErr == nil && analysis != nil {
		_ = s.repo.SetSessionAnalysis(ctx, session.ID, analysis)
		narrative = analysis.SessionNarrative
	}

	return &models.EndSessionResponse{
		SessionID:          session.ID,
		Status:             string(models.TherapyStatusCompleted),
		DurationSec:        elapsed,
		TurnCount:          session.TurnCount,
		PostSessionSummary: narrative,
	}, nil
}

// GetSession returns a session with its full message history.
func (s *TherapyService) GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.TherapySession, error) {
	session, err := s.repo.GetByID(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.GetSession: %w", err)
	}
	if session == nil {
		return nil, ErrTherapyNotFound
	}

	// Auto-expire if active and past expiry.
	if session.Status == models.TherapyStatusActive && time.Now().After(session.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, session.ID, models.TherapyStatusExpired, nil, nil)
		session.Status = models.TherapyStatusExpired
		session.TimeRemainingSec = 0
	}

	messages, err := s.repo.ListMessages(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.GetSession messages: %w", err)
	}
	session.Messages = messages
	return session, nil
}

// ListSessions returns a summary list of sessions for a user.
func (s *TherapyService) ListSessions(ctx context.Context, userID uuid.UUID) (*models.ListTherapySessionsResponse, error) {
	sessions, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("therapySvc.ListSessions: %w", err)
	}

	summaries := make([]models.TherapySessionSummary, 0, len(sessions))
	for _, sess := range sessions {
		summaries = append(summaries, models.TherapySessionSummary{
			ID:                 sess.ID,
			Status:             sess.Status,
			StartedAt:          sess.StartedAt,
			EndedAt:            sess.EndedAt,
			DurationSec:        sess.DurationSec,
			TurnCount:          sess.TurnCount,
			PostSessionSummary: sess.PostSessionSummary,
		})
	}
	return &models.ListTherapySessionsResponse{Sessions: summaries}, nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func (s *TherapyService) computeBilling(ctx context.Context, userID uuid.UUID, plan models.Plan) (int, error) {
	// First session ever is always free.
	total, err := s.repo.CountAll(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("therapySvc.billing: %w", err)
	}
	if total == 0 {
		return 0, nil
	}

	// Pro plan includes 2 sessions/month.
	if plan.AtLeast(models.PlanPro) {
		monthCount, err := s.repo.CountThisMonth(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("therapySvc.billing: %w", err)
		}
		if monthCount < models.TherapyProMonthlyAllowance {
			return 0, nil
		}
	}

	// In dev stub mode: allow with billing amount set (no real payment).
	if s.stubBilling {
		return models.TherapySessionPricePaise, nil
	}

	// In prod: real payment would be processed here. For now, return 402.
	return 0, errTherapyPaymentRequired
}

func (s *TherapyService) loadContextSnapshot(ctx context.Context, userID uuid.UUID) (models.TherapyContextSnapshot, bool, error) {
	snapshot := models.TherapyContextSnapshot{}

	avg, err := s.analysisRepo.MoodAvg30Days(ctx, userID)
	if err == nil {
		snapshot.MoodAvg30d = avg
	}

	emotions, err := s.analysisRepo.TopEmotions(ctx, userID, 5)
	if err == nil {
		snapshot.TopEmotions = emotions
	}

	topics, err := s.analysisRepo.TopTopics(ctx, userID, 5)
	if err == nil {
		snapshot.TopTopics = topics
	}

	summaries, err := s.analysisRepo.RecentSummaries(ctx, userID, 5)
	if err == nil {
		snapshot.RecentSummaries = summaries
	}

	// Inject last 3 completed session summaries for continuity (ADR-016).
	pastSummaries, err := s.repo.PastCompletedSummaries(ctx, userID, 3)
	if err == nil {
		snapshot.PastSessionSummaries = pastSummaries
	}

	contextLoaded := snapshot.MoodAvg30d != nil || len(snapshot.RecentSummaries) > 0
	return snapshot, contextLoaded, nil
}

func (s *TherapyService) transcribeAudio(ctx context.Context, audioKey string) (string, error) {
	rc, err := s.storage.GetObject(ctx, audioKey)
	if err != nil {
		return "", fmt.Errorf("fetch audio: %w", err)
	}
	defer rc.Close()

	whisperCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := s.transcription.Transcribe(whisperCtx, rc, audioKey)
	if err != nil {
		return "", fmt.Errorf("whisper: %w", err)
	}
	return resp.Text, nil
}

// snapshotToPromptContext converts a stored snapshot to the prompt input struct.
func snapshotToPromptContext(snap models.TherapyContextSnapshot, name string) TherapyPromptContext {
	return TherapyPromptContext{
		Name:                 name,
		MoodAvg30d:           snap.MoodAvg30d,
		TopEmotions:          snap.TopEmotions,
		TopTopics:            snap.TopTopics,
		RecentSummaries:      snap.RecentSummaries,
		PastSessionSummaries: snap.PastSessionSummaries,
	}
}

// ── Sentinel errors ──────────────────────────────────────────────────────────

var (
	ErrTherapyNotFound       = fmt.Errorf("therapy session not found")
	ErrTherapyNotActive      = fmt.Errorf("therapy session is not active")
	ErrTherapyExpired        = fmt.Errorf("therapy session has expired")
	ErrTherapyAlreadyEnded   = fmt.Errorf("therapy session already ended")
	errTherapyMissingAudio   = fmt.Errorf("audio_key is required for voice input")
	errTherapyEmptyContent   = fmt.Errorf("message content is empty")
	errTherapyPaymentRequired = fmt.Errorf("therapy session requires payment")
)
