package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// ── Minimal fakes ─────────────────────────────────────────────────────────────

type fakeTherapyRepo struct {
	sessions        map[uuid.UUID]*models.TherapySession
	messages        map[uuid.UUID][]models.TherapySessionMessage
	countAll        int
	countMonth      int
	pastSummaries   []string // injected per-test

	turns       map[fakeTurnKey]uuid.UUID // (session, request_id) → user message
	replies     map[uuid.UUID]uuid.UUID   // user message → assistant reply
	released    int                       // ReleaseTurn calls that removed a turn
	beforeReply func()                    // runs inside AddReply before the status re-check
}

func newFakeTherapyRepo() *fakeTherapyRepo {
	return &fakeTherapyRepo{
		sessions: make(map[uuid.UUID]*models.TherapySession),
		messages: make(map[uuid.UUID][]models.TherapySessionMessage),
	}
}

func (r *fakeTherapyRepo) Create(_ context.Context, userID uuid.UUID, persona models.TherapyPersona, snapshot models.TherapyContextSnapshot, billingPaise int) (*models.TherapySession, error) {
	id := uuid.New()
	now := time.Now()
	s := &models.TherapySession{
		ID:                 id,
		UserID:             userID,
		Status:             models.TherapyStatusActive,
		Persona:            persona,
		StartedAt:          now,
		ExpiresAt:          now.Add(models.TherapySessionDuration),
		TurnCount:          0,
		CrisisWarnings:     0,
		ContextSnapshot:    snapshot,
		BillingAmountPaise: billingPaise,
		CreatedAt:          now,
	}
	r.sessions[id] = s
	r.countAll++
	r.countMonth++
	return s, nil
}

func (r *fakeTherapyRepo) GetByID(_ context.Context, id, userID uuid.UUID) (*models.TherapySession, error) {
	s, ok := r.sessions[id]
	if !ok || s.UserID != userID {
		return nil, nil
	}
	return s, nil
}

func (r *fakeTherapyRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*models.TherapySession, error) {
	var out []*models.TherapySession
	for _, s := range r.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeTherapyRepo) UpdateStatus(_ context.Context, id uuid.UUID, status models.TherapySessionStatus, endedAt *time.Time, durationSec *int) error {
	s, ok := r.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	s.Status = status
	s.EndedAt = endedAt
	s.DurationSec = durationSec
	return nil
}

func (r *fakeTherapyRepo) IncrementTurn(_ context.Context, id uuid.UUID) (int, error) {
	s, ok := r.sessions[id]
	if !ok {
		return 0, errors.New("session not found")
	}
	s.TurnCount++
	return s.TurnCount, nil
}

func (r *fakeTherapyRepo) IncrementCrisisWarning(_ context.Context, id uuid.UUID) (int, error) {
	s, ok := r.sessions[id]
	if !ok {
		return 0, errors.New("session not found")
	}
	s.CrisisWarnings++
	return s.CrisisWarnings, nil
}

func (r *fakeTherapyRepo) SetSessionAnalysis(_ context.Context, id uuid.UUID, a *models.TherapySessionAnalysis) error {
	s, ok := r.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	if s.Status != models.TherapyStatusCompleted {
		return nil // mirrors the repository: only a completed session takes an analysis
	}
	s.PostSessionSummary = &a.SessionNarrative
	s.SessionMoodScore = &a.MoodScore
	s.SessionEmotionalTone = a.EmotionalTone
	s.SessionTopics = a.Topics
	s.SessionKeyInsights = a.KeyInsights
	return nil
}

func (r *fakeTherapyRepo) SetPostSessionSummary(_ context.Context, id uuid.UUID, summary string) error {
	s, ok := r.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	s.PostSessionSummary = &summary
	return nil
}

func (r *fakeTherapyRepo) CountAll(_ context.Context, _ uuid.UUID) (int, error) {
	return r.countAll, nil
}

func (r *fakeTherapyRepo) CountThisMonth(_ context.Context, _ uuid.UUID) (int, error) {
	return r.countMonth, nil
}

func (r *fakeTherapyRepo) PastCompletedSummaries(_ context.Context, _ uuid.UUID, _ int) ([]string, error) {
	return r.pastSummaries, nil
}

func (r *fakeTherapyRepo) AddMessage(_ context.Context, sessionID uuid.UUID, role, content, inputMode string) (*models.TherapySessionMessage, error) {
	msg := models.TherapySessionMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		InputMode: inputMode,
		CreatedAt: time.Now(),
	}
	r.messages[sessionID] = append(r.messages[sessionID], msg)
	return &msg, nil
}

func (r *fakeTherapyRepo) ListMessages(_ context.Context, sessionID uuid.UUID) ([]models.TherapySessionMessage, error) {
	return r.messages[sessionID], nil
}

type fakeTherapyAnalysisRepo struct{}

func (r *fakeTherapyAnalysisRepo) MoodAvg30Days(_ context.Context, _ uuid.UUID) (*float64, error) {
	v := 65.0
	return &v, nil
}
func (r *fakeTherapyAnalysisRepo) RecentSummaries(_ context.Context, _ uuid.UUID, _ int) ([]string, error) {
	return []string{"Had a hard week at work.", "Feeling better after the weekend."}, nil
}
func (r *fakeTherapyAnalysisRepo) TopEmotions(_ context.Context, _ uuid.UUID, _ int) ([]string, error) {
	return []string{"anxious", "hopeful"}, nil
}
func (r *fakeTherapyAnalysisRepo) TopTopics(_ context.Context, _ uuid.UUID, _ int) ([]string, error) {
	return []string{"work", "relationships"}, nil
}

type fakeTherapyStorage struct {
	deleteCalledWith []string
}

func (s *fakeTherapyStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("fake audio bytes")), nil
}

func (s *fakeTherapyStorage) Delete(_ context.Context, key string) error {
	s.deleteCalledWith = append(s.deleteCalledWith, key)
	return nil
}

func newStubTherapyService(repo *fakeTherapyRepo, crisisDetector *CrisisDetector) *TherapyService {
	return newStubTherapyServiceWithTTS(repo, crisisDetector, nil)
}

func newStubTherapyServiceWithTTS(repo *fakeTherapyRepo, crisisDetector *CrisisDetector, tts *TTSService) *TherapyService {
	claude := NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"})
	transcription := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"})
	if crisisDetector == nil {
		crisisDetector = NewCrisisDetector(nil, nil)
	}
	return NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, claude, transcription, &fakeTherapyStorage{}, crisisDetector, tts, true, nil)
}

func claudeBlockingTherapyServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
}

func startSession(t *testing.T, svc *TherapyService, userID uuid.UUID) *models.TherapySession {
	t.Helper()
	s, err := svc.StartSession(context.Background(), userID, models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	return s
}

// ── Session lifecycle ─────────────────────────────────────────────────────────

func TestTherapy_StartSession_FirstSessionFree(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll = 0
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.BillingAmountPaise != 0 {
		t.Errorf("first session should be free; got billing_amount_paise=%d", session.BillingAmountPaise)
	}
	if session.Status != models.TherapyStatusActive {
		t.Errorf("expected status=active, got %s", session.Status)
	}
}

func TestTherapy_StartSession_ProPlanIncluded(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll = 1
	repo.countMonth = 0
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanPro, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.BillingAmountPaise != 0 {
		t.Errorf("Pro plan within allowance should be free; got %d", session.BillingAmountPaise)
	}
}

func TestTherapy_StartSession_PayPerUseInStubMode(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll = 3
	repo.countMonth = 5
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error in stub mode: %v", err)
	}
	if session.BillingAmountPaise != models.TherapySessionPricePaise {
		t.Errorf("expected billing_amount_paise=%d, got %d", models.TherapySessionPricePaise, session.BillingAmountPaise)
	}
}

func TestTherapy_StartSession_ProBeyondAllowance_MemberPrice(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll = 3
	repo.countMonth = models.TherapyProMonthlyAllowance // included session already used
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanPro, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error in stub mode: %v", err)
	}
	if session.BillingAmountPaise != models.TherapyMemberSessionPricePaise {
		t.Errorf("Pro member beyond allowance should pay member price %d, got %d",
			models.TherapyMemberSessionPricePaise, session.BillingAmountPaise)
	}
}

func TestTherapy_StartSession_LoadsJournalContext(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll = 0
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(session.ContextSnapshot.TopEmotions) == 0 {
		t.Error("context snapshot must include top emotions")
	}
	if len(session.ContextSnapshot.RecentSummaries) == 0 {
		t.Error("context snapshot must include recent summaries")
	}
}

func TestTherapy_GetSession_ReturnsFullHistory(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)
	_, _ = repo.AddMessage(context.Background(), session.ID, "user", "hello", "text")
	_, _ = repo.AddMessage(context.Background(), session.ID, "assistant", "hi", "text")

	got, err := svc.GetSession(context.Background(), session.ID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(got.Messages))
	}
}

func TestTherapy_EndSession_GeneratesSummary(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)
	_, _ = repo.AddMessage(context.Background(), session.ID, "user", "I have been stressed.", "text")

	resp, err := svc.EndSession(context.Background(), session.ID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PostSessionSummary == "" {
		t.Error("expected non-empty post-session summary")
	}
	if resp.Status != string(models.TherapyStatusCompleted) {
		t.Errorf("expected status=completed, got %s", resp.Status)
	}
}

func TestTherapy_EndSession_AlreadyEnded_Returns409(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)
	_, _ = svc.EndSession(context.Background(), session.ID, userID)

	_, err := svc.EndSession(context.Background(), session.ID, userID)
	if !errors.Is(err, ErrTherapyAlreadyEnded) {
		t.Errorf("expected ErrTherapyAlreadyEnded, got %v", err)
	}
}

// ── Expiry enforcement (ADR-012) ──────────────────────────────────────────────

func TestTherapy_ExpiredSession_RejectsMessages(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)
	session.ExpiresAt = time.Now().Add(-1 * time.Second)

	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "hello", InputMode: "text",
	})
	if !errors.Is(err, ErrTherapyExpired) {
		t.Errorf("expected ErrTherapyExpired, got %v", err)
	}
}

func TestTherapy_ExpiryEnforcedServerSide(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)
	session.ExpiresAt = time.Now().Add(-5 * time.Minute)

	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "still trying", InputMode: "text",
	})
	if !errors.Is(err, ErrTherapyExpired) {
		t.Errorf("server-side expiry check failed; expected ErrTherapyExpired, got %v", err)
	}
}

// ── Original crisis detection (ADR-013) ──────────────────────────────────────

func TestTherapy_CrisisFailSafe_ClaudeUnreachable(t *testing.T) {
	errSrv := claudeErrorServer(t)
	defer errSrv.Close()

	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(newClaudeWithServer(errSrv), nil))
	userID := uuid.New()

	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content:   "I feel like it's not worth living anymore",
		InputMode: "text",
	})
	if err != nil {
		t.Fatalf("SendMessage must not error when Claude is unreachable during crisis check: %v", err)
	}
	if !resp.SessionState.IsCrisis {
		t.Error("Claude unreachable in Stage 2 must default to crisis (fail-safe, ADR-002)")
	}
}

func TestTherapy_CrisisFailSafe_ContextCancelled(t *testing.T) {
	blockSrv := claudeBlockingTherapyServer(t)
	defer blockSrv.Close()

	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(newClaudeWithServer(blockSrv), nil))
	userID := uuid.New()

	session := startSession(t, svc, userID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := svc.SendMessage(ctx, session.ID, userID, models.SendTherapyMessageRequest{
		Content:   "I feel like nobody cares if I live or die",
		InputMode: "text",
	})
	if err != nil {
		t.Fatalf("SendMessage must absorb context errors during crisis check: %v", err)
	}
	if !resp.SessionState.IsCrisis {
		t.Error("cancelled context during crisis Stage 2 must default to crisis (fail-safe)")
	}
}

// ── Text message round-trip ───────────────────────────────────────────────────

func TestTherapy_TextInput_ReturnsAssistantReply(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content:   "I have been feeling stressed at work.",
		InputMode: "text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserMessage.Content != "I have been feeling stressed at work." {
		t.Errorf("user message content mismatch: %s", resp.UserMessage.Content)
	}
	if resp.AssistantMessage.Content == "" {
		t.Error("assistant reply must not be empty")
	}
	if resp.SessionState.TurnCount != 1 {
		t.Errorf("expected turn_count=1, got %d", resp.SessionState.TurnCount)
	}
}

// ── Audio deletion (ADR-005) ──────────────────────────────────────────────────

func TestTherapy_AudioDeletedAfterTranscription(t *testing.T) {
	repo := newFakeTherapyRepo()
	storage := &fakeTherapyStorage{}
	claude := NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"})
	transcription := NewTranscriptionService(&appconfig.OpenAIConfig{
		BaseURL: "http://localhost:0", APIKey: "test",
	})
	svc := NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, claude, transcription, storage, NewCrisisDetector(nil, nil), nil, true, nil)

	userID := uuid.New()
	session := startSession(t, svc, userID)

	key := TherapyAudioPrefix(userID, session.ID) + "voice.aac"
	_, _ = svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		AudioKey:  key,
		InputMode: "voice",
	})

	if len(storage.deleteCalledWith) == 0 {
		t.Fatal("audio must be deleted from storage even when transcription fails")
	}
	if storage.deleteCalledWith[0] != key {
		t.Errorf("wrong key deleted: %s", storage.deleteCalledWith[0])
	}
}

// ── TTS (non-fatal, ADR-005-adjacent: never blocks the reply) ────────────────

func TestTherapy_SendMessage_TTSFailure_FallsBackToTextOnly(t *testing.T) {
	azure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid key"))
	}))
	defer azure.Close()

	tts := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key: "bad-key", BaseURL: azure.URL,
	}, &fakeTTSStorage{})

	repo := newFakeTherapyRepo()
	svc := newStubTherapyServiceWithTTS(repo, nil, tts)
	userID := uuid.New()
	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "hello", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("SendMessage must not fail when TTS synthesis fails: %v", err)
	}
	if resp.AssistantMessage.TTSUrl != nil {
		t.Errorf("TTSUrl must be nil when synthesis failed, got %q", *resp.AssistantMessage.TTSUrl)
	}
}

func TestTherapy_SendMessage_TTSSuccess_SetsURL(t *testing.T) {
	azure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-mp3-bytes"))
	}))
	defer azure.Close()

	tts := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key: "good-key", BaseURL: azure.URL,
	}, &fakeTTSStorage{})

	repo := newFakeTherapyRepo()
	svc := newStubTherapyServiceWithTTS(repo, nil, tts)
	userID := uuid.New()
	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "hello", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if resp.AssistantMessage.TTSUrl == nil || *resp.AssistantMessage.TTSUrl == "" {
		t.Error("TTSUrl must be set when synthesis succeeds")
	}
}

// ── Payment required (prod mode) ─────────────────────────────────────────────

func TestTherapy_PaymentRequired_NonStubbedMode(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll = 5
	repo.countMonth = 5

	claude := NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"})
	transcription := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"})
	svc := NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, claude, transcription, &fakeTherapyStorage{}, NewCrisisDetector(nil, nil), nil, false, nil)

	_, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err == nil {
		t.Error("expected payment required error in non-stub mode; got nil")
	}
}

// ── Phase 8: Layered Crisis (ADR-014) - blocking, same bar as Priority 1 ─────

func TestTherapy_LayeredCrisis_FirstDetection_DeEscalates(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	userID := uuid.New()

	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content:   "I want to kill myself right now",
		InputMode: "text",
	})
	if err != nil {
		t.Fatalf("SendMessage must not error on first crisis detection: %v", err)
	}
	// Session must stay open after first detection.
	if resp.SessionState.Status == models.TherapyStatusCrisisDetected {
		t.Error("first crisis detection must NOT hard-stop the session; status should remain active")
	}
	if resp.SessionState.Status != models.TherapyStatusActive {
		t.Errorf("expected status=active after first detection, got %s", resp.SessionState.Status)
	}
	if resp.SessionState.CrisisWarnings != 1 {
		t.Errorf("expected crisis_warnings=1 after first detection, got %d", resp.SessionState.CrisisWarnings)
	}
	if !resp.SessionState.IsCrisis {
		t.Error("is_crisis must be true even during de-escalation")
	}
}

func TestTherapy_LayeredCrisis_SessionOpenAfterFirstWarning(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	userID := uuid.New()

	session := startSession(t, svc, userID)

	// First crisis - de-escalate.
	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to kill myself", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("first crisis SendMessage: %v", err)
	}

	// Session must still accept a next message.
	_, err = svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I am feeling a bit calmer now", InputMode: "text",
	})
	if errors.Is(err, ErrTherapyNotActive) {
		t.Error("session must remain open after first crisis warning")
	}
}

func TestTherapy_LayeredCrisis_SecondDetection_HardStop(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	userID := uuid.New()

	session := startSession(t, svc, userID)

	// First detection - de-escalate.
	_, _ = svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to kill myself", InputMode: "text",
	})

	// Second detection - hard stop.
	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to end my life", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("second crisis SendMessage must not error: %v", err)
	}
	if resp.SessionState.Status != models.TherapyStatusCrisisDetected {
		t.Errorf("second crisis must hard-stop; expected crisis_detected, got %s", resp.SessionState.Status)
	}
}

func TestTherapy_LayeredCrisis_SessionClosedAfterSecondWarning(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	userID := uuid.New()

	session := startSession(t, svc, userID)
	_, _ = svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to kill myself", InputMode: "text",
	})
	_, _ = svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to end my life", InputMode: "text",
	})

	// Third message must be rejected.
	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "just testing", InputMode: "text",
	})
	if !errors.Is(err, ErrTherapyNotActive) {
		t.Errorf("post hard-stop session must reject messages; expected ErrTherapyNotActive, got %v", err)
	}
}

func TestTherapy_LayeredCrisis_FailSafe_ClaudeUnreachableOnFirst(t *testing.T) {
	// If Claude is unreachable during de-escalation on the first detection,
	// must fall through to hard stop immediately (ADR-014 + ADR-002).
	errSrv := claudeErrorServer(t)
	defer errSrv.Close()

	repo := newFakeTherapyRepo()
	// Use a real crisis detector that will detect Stage 1 keyword.
	// Claude is the errSrv which will fail the de-escalation call too.
	crisis := NewCrisisDetector(newClaudeWithServer(errSrv), nil)
	claude := newClaudeWithServer(errSrv) // de-escalation call will also fail

	transcription := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"})
	svc := NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, claude, transcription, &fakeTherapyStorage{}, crisis, nil, true, nil)

	userID := uuid.New()
	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to kill myself right now", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("must not error even when Claude is unreachable during de-escalation: %v", err)
	}
	// Must hard-stop immediately - not attempt de-escalation then leave open.
	if resp.SessionState.Status != models.TherapyStatusCrisisDetected {
		t.Errorf("Claude unreachable on first crisis + de-escalation must hard-stop; got %s", resp.SessionState.Status)
	}
}

func TestTherapy_LayeredCrisis_FailSafe_TimeoutOnFirst(t *testing.T) {
	blockSrv := claudeBlockingTherapyServer(t)
	defer blockSrv.Close()

	repo := newFakeTherapyRepo()
	crisis := NewCrisisDetector(newClaudeWithServer(blockSrv), nil)
	claude := newClaudeWithServer(blockSrv)
	transcription := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"})
	svc := NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, claude, transcription, &fakeTherapyStorage{}, crisis, nil, true, nil)

	userID := uuid.New()
	session := startSession(t, svc, userID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately - forces timeout path

	resp, err := svc.SendMessage(ctx, session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I feel like nobody cares if I live or die", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("must not error on timeout: %v", err)
	}
	if !resp.SessionState.IsCrisis {
		t.Error("timeout during crisis must still set IsCrisis=true")
	}
}

// ── Phase 8: Personas ─────────────────────────────────────────────────────────

func TestTherapy_StartSession_DefaultPersona(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Persona != models.PersonaComforting {
		t.Errorf("expected persona=comforting, got %s", session.Persona)
	}
}

func TestTherapy_StartSession_AllPersonasAccepted(t *testing.T) {
	personas := []models.TherapyPersona{
		models.PersonaComforting,
		models.PersonaRational,
		models.PersonaCBT,
		models.PersonaMindful,
	}
	for _, p := range personas {
		repo := newFakeTherapyRepo()
		svc := newStubTherapyService(repo, nil)
		session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, p, "", "auto", "")
		if err != nil {
			t.Errorf("persona %s: unexpected error: %v", p, err)
		}
		if session.Persona != p {
			t.Errorf("persona %s: stored persona mismatch, got %s", p, session.Persona)
		}
	}
}

func TestTherapy_PersonaStoredInSession(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session, _ := svc.StartSession(context.Background(), userID, models.PlanFree, models.PersonaRational, "", "auto", "")

	got, err := svc.GetSession(context.Background(), session.ID, userID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Persona != models.PersonaRational {
		t.Errorf("persona not persisted; expected rational, got %s", got.Persona)
	}
}

func TestTherapy_PersonaPrompt_Comforting_ToneWords(t *testing.T) {
	prompt := buildPersonaBlock("comforting")
	for _, word := range []string{"warmth", "validation", "heard", "warm"} {
		if !strings.Contains(strings.ToLower(prompt), word) {
			t.Errorf("comforting persona prompt should contain %q", word)
		}
	}
}

func TestTherapy_PersonaPrompt_Rational_ToneWords(t *testing.T) {
	prompt := buildPersonaBlock("rational")
	for _, word := range []string{"logic", "structured", "control"} {
		if !strings.Contains(strings.ToLower(prompt), word) {
			t.Errorf("rational persona prompt should contain %q", word)
		}
	}
}

func TestTherapy_PersonaPrompt_CBT_ToneWords(t *testing.T) {
	prompt := buildPersonaBlock("cbt")
	for _, word := range []string{"pattern", "thought", "reframe"} {
		if !strings.Contains(strings.ToLower(prompt), word) {
			t.Errorf("cbt persona prompt should contain %q", word)
		}
	}
}

func TestTherapy_PersonaPrompt_Mindful_ToneWords(t *testing.T) {
	prompt := buildPersonaBlock("mindful")
	for _, word := range []string{"present", "breath", "grounding", "ground"} {
		found := false
		for _, w := range []string{"present", "breath", "grounding", "ground"} {
			if strings.Contains(strings.ToLower(prompt), w) {
				found = true
				break
			}
		}
		_ = word
		if !found {
			t.Error("mindful persona prompt should contain grounding/present-moment language")
			break
		}
	}
}

func TestTherapy_PersonaSystemPromptRouting(t *testing.T) {
	cases := []struct {
		persona string
		marker  string
	}{
		{"comforting", "COMFORTING"},
		{"rational", "RATIONAL"},
		{"cbt", "CBT"},
		{"mindful", "MINDFUL"},
	}
	ctx := TherapyPromptContext{Name: "Test"}
	for _, tc := range cases {
		prompt := buildTherapyModeSystemPrompt(ctx, tc.persona, "Time remaining: 3600 seconds")
		if !strings.Contains(prompt, tc.marker) {
			t.Errorf("persona %s: expected prompt to contain %q marker", tc.persona, tc.marker)
		}
	}
}

// ── Phase 8: Session Continuity (ADR-016) ─────────────────────────────────────

func TestTherapy_StartSession_InjectsPastSummaries(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.pastSummaries = []string{
		"Explored work-related anxiety.",
		"Discussed family dynamics.",
		"Reflected on progress in relationships.",
	}
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(session.ContextSnapshot.PastSessionSummaries) != 3 {
		t.Errorf("expected 3 past session summaries in snapshot, got %d", len(session.ContextSnapshot.PastSessionSummaries))
	}
}

func TestTherapy_StartSession_NoPastSessions(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.pastSummaries = nil
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("first session must not error: %v", err)
	}
	if len(session.ContextSnapshot.PastSessionSummaries) != 0 {
		t.Error("no past summaries should be injected on first session")
	}
}

func TestTherapy_StartSession_FewerThanThreeSessions(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.pastSummaries = []string{"One past session."}
	svc := newStubTherapyService(repo, nil)

	session, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(session.ContextSnapshot.PastSessionSummaries) != 1 {
		t.Errorf("expected 1 past session summary, got %d", len(session.ContextSnapshot.PastSessionSummaries))
	}
}

func TestTherapy_PastSummaryInSystemPrompt(t *testing.T) {
	ctx := TherapyPromptContext{
		Name:                 "Test",
		PastSessionSummaries: []string{"Talked about work stress.", "Explored grief around a loss."},
	}
	prompt := buildTherapyModeSystemPrompt(ctx, "comforting", "Time remaining: 3600 seconds")
	if !strings.Contains(prompt, "MEMORY FROM PAST SESSIONS") {
		t.Error("system prompt must include past session memory section when summaries are present")
	}
	if !strings.Contains(prompt, "Talked about work stress.") {
		t.Error("system prompt must include actual past summary content")
	}
}

func TestTherapy_NarratedOpening_WithPastSessions(t *testing.T) {
	ctx := TherapyPromptContext{
		Name:                 "Test",
		PastSessionSummaries: []string{"Talked about work stress."},
	}
	prompt := buildTherapyModeSystemPrompt(ctx, "comforting", "Time remaining: 3600 seconds")
	if !strings.Contains(prompt, "SESSION OPENING") {
		t.Error("system prompt must instruct a narrated opening when past sessions exist")
	}
}

func TestTherapy_NarratedOpening_AbsentOnFirstSession(t *testing.T) {
	ctx := TherapyPromptContext{Name: "Test"}
	prompt := buildTherapyModeSystemPrompt(ctx, "comforting", "Time remaining: 3600 seconds")
	if strings.Contains(prompt, "SESSION OPENING") {
		t.Error("first-ever session has no past thread to pick up - opening instruction must be absent")
	}
}

func TestTherapy_PastSummariesFromSnapshotNotLive(t *testing.T) {
	// Verify context is read from the stored snapshot, not re-queried on each turn.
	repo := newFakeTherapyRepo()
	repo.pastSummaries = []string{"Old summary at session start."}
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)

	// Change repo data after session started - should NOT affect this session.
	repo.pastSummaries = []string{"New summary added after start."}

	got, err := svc.GetSession(context.Background(), session.ID, userID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if len(got.ContextSnapshot.PastSessionSummaries) != 1 {
		t.Fatalf("expected 1 past summary in snapshot, got %d", len(got.ContextSnapshot.PastSessionSummaries))
	}
	if got.ContextSnapshot.PastSessionSummaries[0] != "Old summary at session start." {
		t.Errorf("snapshot must be frozen at session start; got %q", got.ContextSnapshot.PastSessionSummaries[0])
	}
}

// ── Phase 8: Wind-Down ────────────────────────────────────────────────────────

func TestTherapy_WindDown_PromptIncludesTimeRemaining(t *testing.T) {
	instruction := buildWindDownInstruction(2400)
	if !strings.Contains(instruction, "2400") {
		t.Error("wind-down instruction must include time_remaining_sec value")
	}
}

func TestTherapy_WindDown_SubTenMinuteInstructions(t *testing.T) {
	instruction := buildWindDownInstruction(500) // < 600
	if !strings.Contains(strings.ToLower(instruction), "close") &&
		!strings.Contains(strings.ToLower(instruction), "natural") {
		t.Error("sub-10-minute wind-down instruction must mention closing the conversation")
	}
}

func TestTherapy_WindDown_SubTwoMinuteInstructions(t *testing.T) {
	instruction := buildWindDownInstruction(90) // < 120
	if !strings.Contains(strings.ToLower(instruction), "wrap") &&
		!strings.Contains(strings.ToLower(instruction), "this") {
		t.Error("sub-2-minute wind-down instruction must instruct wrap-up in this turn")
	}
}

func TestTherapy_WindDown_TimeRemainingAccurate(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	session := startSession(t, svc, userID)

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "hi", InputMode: "text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be close to 3600 (1 hour), definitely > 0.
	if resp.SessionState.TimeRemainingSec <= 0 {
		t.Errorf("time_remaining_sec must be > 0 for a fresh session; got %d", resp.SessionState.TimeRemainingSec)
	}
	if resp.SessionState.TimeRemainingSec > 3600 {
		t.Errorf("time_remaining_sec must not exceed session duration; got %d", resp.SessionState.TimeRemainingSec)
	}
}

// staleReadTherapyRepo models a second in-flight request: GetByID returns the
// session as it was before a concurrent request claimed the de-escalation.
type staleReadTherapyRepo struct{ *fakeTherapyRepo }

func (r *staleReadTherapyRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.TherapySession, error) {
	s, err := r.fakeTherapyRepo.GetByID(ctx, id, userID)
	if s == nil || err != nil {
		return s, err
	}
	cp := *s
	cp.CrisisWarnings = 0 // stale
	return &cp, nil
}

// H10: two concurrent crisis messages must yield one de-escalation and one hard stop.
func TestTherapy_LayeredCrisis_ConcurrentSecondDetection_HardStops(t *testing.T) {
	base := newFakeTherapyRepo()
	userID := uuid.New()
	session := startSession(t, newStubTherapyService(base, nil), userID)
	svc := NewTherapyService(&staleReadTherapyRepo{base}, &fakeTherapyAnalysisRepo{}, nil,
		NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"}),
		NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"}),
		&fakeTherapyStorage{}, NewCrisisDetector(nil, nil), nil, true, nil)

	// The "other" request already took the de-escalation slot.
	base.sessions[session.ID].CrisisWarnings = 1

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to kill myself", InputMode: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.SessionState.Status != models.TherapyStatusCrisisDetected {
		t.Errorf("second concurrent crisis must hard-stop, got status %s", resp.SessionState.Status)
	}
}

// H10: ending a crisis-stopped session must not flip it back to completed.
func TestTherapy_EndSession_CrisisDetected_Rejected(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	userID := uuid.New()
	session := startSession(t, svc, userID)
	repo.sessions[session.ID].Status = models.TherapyStatusCrisisDetected

	if _, err := svc.EndSession(context.Background(), session.ID, userID); !errors.Is(err, ErrTherapyAlreadyEnded) {
		t.Errorf("want ErrTherapyAlreadyEnded, got %v", err)
	}
	if repo.sessions[session.ID].Status != models.TherapyStatusCrisisDetected {
		t.Error("crisis_detected status must be preserved")
	}
}

// L2: the therapy prompt addresses the user by name.
func TestTherapy_StartSession_SnapshotsUserName(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	s, err := svc.StartSession(context.Background(), uuid.New(), models.PlanFree, models.PersonaComforting, "", "auto", "Asha")
	if err != nil {
		t.Fatal(err)
	}
	if repo.sessions[s.ID].ContextSnapshot.UserName != "Asha" {
		t.Errorf("snapshot user name: want Asha, got %q", repo.sessions[s.ID].ContextSnapshot.UserName)
	}
}

// M13: another user's (or another session's) therapy key is refused and never
// transcribed or deleted.
func TestTherapy_ForeignAudioKey_Rejected(t *testing.T) {
	repo := newFakeTherapyRepo()
	storage := &fakeTherapyStorage{}
	svc := NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil,
		NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"}),
		NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:0", APIKey: "test"}),
		storage, NewCrisisDetector(nil, nil), nil, true, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)

	foreign := TherapyAudioPrefix(uuid.New(), session.ID) + "voice.aac"
	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		AudioKey: foreign, InputMode: "voice",
	})
	if !errors.Is(err, errTherapyInvalidAudioKey) {
		t.Fatalf("want errTherapyInvalidAudioKey, got %v", err)
	}
	if len(storage.deleteCalledWith) != 0 {
		t.Error("a foreign key must never be deleted")
	}
}

// ── Turn idempotency / conditional transitions on the fake (N3/N4) ──────────

type fakeTurnKey struct{ session, request uuid.UUID }

func (r *fakeTherapyRepo) turnMaps() {
	if r.turns == nil {
		r.turns = make(map[fakeTurnKey]uuid.UUID)
		r.replies = make(map[uuid.UUID]uuid.UUID)
	}
}

func (r *fakeTherapyRepo) message(sessionID, id uuid.UUID) *models.TherapySessionMessage {
	for i := range r.messages[sessionID] {
		if r.messages[sessionID][i].ID == id {
			m := r.messages[sessionID][i]
			return &m
		}
	}
	return nil
}

func (r *fakeTherapyRepo) FindTurn(_ context.Context, sessionID, requestID uuid.UUID) (*models.TherapySessionMessage, *models.TherapySessionMessage, error) {
	r.turnMaps()
	uid, ok := r.turns[fakeTurnKey{sessionID, requestID}]
	if !ok {
		return nil, nil, nil
	}
	user := r.message(sessionID, uid)
	if rid, ok := r.replies[uid]; ok {
		return user, r.message(sessionID, rid), nil
	}
	return user, nil, nil
}

func (r *fakeTherapyRepo) ClaimTurn(ctx context.Context, sessionID, requestID uuid.UUID, content, inputMode string, allowEnded bool) (*models.TherapySessionMessage, bool, error) {
	r.turnMaps()
	if uid, ok := r.turns[fakeTurnKey{sessionID, requestID}]; ok {
		return r.message(sessionID, uid), false, nil
	}
	if s := r.sessions[sessionID]; !allowEnded && (s == nil || s.Status != models.TherapyStatusActive) {
		return nil, false, models.ErrTherapySessionNotActive
	}
	m, _ := r.AddMessage(ctx, sessionID, "user", content, inputMode)
	r.turns[fakeTurnKey{sessionID, requestID}] = m.ID
	return m, true, nil
}

func (r *fakeTherapyRepo) ReleaseTurn(_ context.Context, userMessageID uuid.UUID) error {
	r.turnMaps()
	if _, answered := r.replies[userMessageID]; answered {
		return nil
	}
	r.released++
	for k, v := range r.turns {
		if v == userMessageID {
			delete(r.turns, k)
		}
	}
	for sid, msgs := range r.messages {
		for i, m := range msgs {
			if m.ID == userMessageID {
				r.messages[sid] = append(msgs[:i:i], msgs[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (r *fakeTherapyRepo) AddReply(ctx context.Context, sessionID, replyTo uuid.UUID, content, inputMode string, requireActive, countTurn bool) (*models.TherapySessionMessage, int, error) {
	r.turnMaps()
	s := r.sessions[sessionID]
	if rid, ok := r.replies[replyTo]; ok {
		return r.message(sessionID, rid), s.TurnCount, nil
	}
	if r.beforeReply != nil {
		r.beforeReply()
	}
	if requireActive && s.Status != models.TherapyStatusActive {
		return nil, 0, models.ErrTherapySessionNotActive
	}
	m, _ := r.AddMessage(ctx, sessionID, "assistant", content, inputMode)
	r.replies[replyTo] = m.ID
	if countTurn {
		s.TurnCount++
	}
	return m, s.TurnCount, nil
}

func (r *fakeTherapyRepo) EndActive(_ context.Context, id uuid.UUID, status models.TherapySessionStatus, endedAt *time.Time, durationSec *int) (bool, error) {
	s, ok := r.sessions[id]
	if !ok {
		return false, errors.New("session not found")
	}
	if s.Status != models.TherapyStatusActive {
		return false, nil
	}
	s.Status, s.EndedAt, s.DurationSec = status, endedAt, durationSec
	return true, nil
}

func (r *fakeTherapyRepo) MarkCrisis(_ context.Context, id uuid.UUID, endedAt time.Time, durationSec int) error {
	s, ok := r.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	if s.Status != models.TherapyStatusCrisisDetected {
		s.Status, s.EndedAt, s.DurationSec, s.PostSessionSummary = models.TherapyStatusCrisisDetected, &endedAt, &durationSec, nil
	}
	return nil
}

// ── N3 / N4 regressions ──────────────────────────────────────────────────────

func countRole(msgs []models.TherapySessionMessage, role string) int {
	n := 0
	for _, m := range msgs {
		if m.Role == role {
			n++
		}
	}
	return n
}

// N4: a retried turn with the same request_id returns the original pair and
// stores / counts nothing new.
func TestTherapy_SendMessage_SameRequestID_ReturnsOriginalPair(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)
	req := models.SendTherapyMessageRequest{Content: "Work has been heavy.", InputMode: "text", RequestID: uuid.NewString()}

	first, err := svc.SendMessage(context.Background(), session.ID, userID, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.SendMessage(context.Background(), session.ID, userID, req)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if second.UserMessage.ID != first.UserMessage.ID || second.AssistantMessage.ID != first.AssistantMessage.ID {
		t.Errorf("retry must return the original pair")
	}
	if n := len(repo.messages[session.ID]); n != 2 {
		t.Errorf("retry must not store messages again: want 2, got %d", n)
	}
	if tc := repo.sessions[session.ID].TurnCount; tc != 1 {
		t.Errorf("retry must not count a second turn: want 1, got %d", tc)
	}
}

// N4: a retried crisis message is the SAME detection - it must not escalate a
// de-escalated session to a hard stop (ADR-014).
func TestTherapy_SendMessage_CrisisRetry_NotASecondDetection(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, NewCrisisDetector(nil, nil))
	userID := uuid.New()
	session := startSession(t, svc, userID)
	req := models.SendTherapyMessageRequest{Content: "I want to kill myself", InputMode: "text", RequestID: uuid.NewString()}

	first, err := svc.SendMessage(context.Background(), session.ID, userID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionState.Status != models.TherapyStatusActive || first.SessionState.CrisisWarnings != 1 {
		t.Fatalf("first detection must de-escalate, got %+v", first.SessionState)
	}
	second, err := svc.SendMessage(context.Background(), session.ID, userID, req)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if second.AssistantMessage.ID != first.AssistantMessage.ID {
		t.Error("retry must return the stored de-escalation reply")
	}
	s := repo.sessions[session.ID]
	if s.Status != models.TherapyStatusActive || s.CrisisWarnings != 1 {
		t.Errorf("retry must not count as a second detection: status=%s warnings=%d", s.Status, s.CrisisWarnings)
	}
}

// N4: a retry while the original is still running is told so, not re-run.
func TestTherapy_SendMessage_TurnInProgress(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)
	reqID := uuid.New()
	if _, _, err := repo.ClaimTurn(context.Background(), session.ID, reqID, "still thinking", "text", false); err != nil {
		t.Fatal(err)
	}

	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "still thinking", InputMode: "text", RequestID: reqID.String(),
	})
	if !errors.Is(err, ErrTherapyTurnInProgress) {
		t.Errorf("want ErrTherapyTurnInProgress, got %v", err)
	}
}

func TestTherapy_SendMessage_InvalidRequestID(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)

	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "hi", InputMode: "text", RequestID: "not-a-uuid",
	})
	if !errors.Is(err, errTherapyInvalidRequestID) {
		t.Errorf("want errTherapyInvalidRequestID, got %v", err)
	}
}

// N4: a turn that fails before replying is released, so the same request_id
// can be retried and actually run.
func TestTherapy_SendMessage_FailedTurnReleasedForRetry(t *testing.T) {
	errSrv := claudeErrorServer(t)
	defer errSrv.Close()
	repo := newFakeTherapyRepo()
	svc := NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, newClaudeWithServer(errSrv),
		NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"}),
		&fakeTherapyStorage{}, NewCrisisDetector(nil, nil), nil, true, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)
	req := models.SendTherapyMessageRequest{Content: "Just a normal day.", InputMode: "text", RequestID: uuid.NewString()}

	if _, err := svc.SendMessage(context.Background(), session.ID, userID, req); err == nil {
		t.Fatal("Claude failure must surface as an error")
	}
	if repo.released != 1 || len(repo.messages[session.ID]) != 0 {
		t.Fatalf("failed turn must be released: released=%d messages=%d", repo.released, len(repo.messages[session.ID]))
	}

	ok := newStubTherapyService(repo, nil)
	if _, err := ok.SendMessage(context.Background(), session.ID, userID, req); err != nil {
		t.Fatalf("retry after release must run the turn: %v", err)
	}
}

// N3: a reply generated while a concurrent request hard-stopped the session
// is discarded, never appended after the crisis resources.
func TestTherapy_SendMessage_ReplyAfterConcurrentHardStop_NotStored(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)
	repo.beforeReply = func() { repo.sessions[session.ID].Status = models.TherapyStatusCrisisDetected }

	_, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "Thinking about my week.", InputMode: "text",
	})
	if !errors.Is(err, ErrTherapyNotActive) {
		t.Errorf("want ErrTherapyNotActive, got %v", err)
	}
	if n := countRole(repo.messages[session.ID], "assistant"); n != 0 {
		t.Errorf("no assistant reply may be stored after a hard stop, got %d", n)
	}
	if repo.sessions[session.ID].TurnCount != 0 {
		t.Error("a discarded reply must not count a turn")
	}
}

// endRaceRepo lands a crisis hard stop while EndSession is generating the
// summary (ListMessages runs right before the summary call).
type endRaceRepo struct{ *fakeTherapyRepo }

func (r *endRaceRepo) ListMessages(ctx context.Context, id uuid.UUID) ([]models.TherapySessionMessage, error) {
	_ = r.fakeTherapyRepo.MarkCrisis(ctx, id, time.Now(), 60)
	return r.fakeTherapyRepo.ListMessages(ctx, id)
}

// N3: EndSession racing a crisis hard stop must leave crisis_detected and
// discard the summary (ADR-014 / ADR-016).
func TestTherapy_EndSession_ConcurrentCrisis_KeepsCrisisDetected(t *testing.T) {
	base := newFakeTherapyRepo()
	userID := uuid.New()
	session := startSession(t, newStubTherapyService(base, nil), userID)
	svc := NewTherapyService(&endRaceRepo{base}, &fakeTherapyAnalysisRepo{}, nil,
		NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"}),
		NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"}),
		&fakeTherapyStorage{}, NewCrisisDetector(nil, nil), nil, true, nil)

	if _, err := svc.EndSession(context.Background(), session.ID, userID); !errors.Is(err, ErrTherapyAlreadyEnded) {
		t.Fatalf("want ErrTherapyAlreadyEnded, got %v", err)
	}
	s := base.sessions[session.ID]
	if s.Status != models.TherapyStatusCrisisDetected {
		t.Errorf("crisis status must survive a concurrent End, got %s", s.Status)
	}
	if s.PostSessionSummary != nil || s.SessionMoodScore != nil {
		t.Error("a crisis session must never receive a post-session summary")
	}
}

// staleActiveRepo returns the session as the request first read it: active.
type staleActiveRepo struct{ *fakeTherapyRepo }

func (r *staleActiveRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.TherapySession, error) {
	s, err := r.fakeTherapyRepo.GetByID(ctx, id, userID)
	if s == nil || err != nil {
		return s, err
	}
	cp := *s
	cp.Status = models.TherapyStatusActive
	return &cp, nil
}

// N3: a crisis message whose session was ended a moment earlier still hard-stops it.
func TestTherapy_SendMessage_CrisisAfterConcurrentEnd_HardStops(t *testing.T) {
	base := newFakeTherapyRepo()
	userID := uuid.New()
	session := startSession(t, newStubTherapyService(base, nil), userID)
	svc := NewTherapyService(&staleActiveRepo{base}, &fakeTherapyAnalysisRepo{}, nil,
		NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"}),
		NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"}),
		&fakeTherapyStorage{}, NewCrisisDetector(nil, nil), nil, true, nil)
	base.sessions[session.ID].Status = models.TherapyStatusCompleted
	summary := "should be cleared"
	base.sessions[session.ID].PostSessionSummary = &summary

	resp, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I want to kill myself", InputMode: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.SessionState.Status != models.TherapyStatusCrisisDetected {
		t.Errorf("want crisis_detected response, got %s", resp.SessionState.Status)
	}
	s := base.sessions[session.ID]
	if s.Status != models.TherapyStatusCrisisDetected || s.PostSessionSummary != nil {
		t.Errorf("crisis must win over a concurrent End: status=%s summary=%v", s.Status, s.PostSessionSummary)
	}
}
