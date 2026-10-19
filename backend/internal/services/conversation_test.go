package services

import (
	"context"
	"errors"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeConvRepo struct {
	conv   *models.Conversation
	msgs   []models.ConversationMessage
	getErr error
	addErr error
}

func (r *fakeConvRepo) GetOrCreate(_ context.Context, entryID, userID uuid.UUID) (*models.Conversation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.conv == nil {
		r.conv = &models.Conversation{
			ID:        uuid.New(),
			EntryID:   entryID,
			UserID:    userID,
			TurnCount: 0,
			IsClosed:  false,
		}
	}
	return r.conv, nil
}

func (r *fakeConvRepo) GetByID(_ context.Context, id, userID uuid.UUID) (*models.Conversation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.conv == nil || r.conv.ID != id || r.conv.UserID != userID {
		return nil, nil
	}
	return r.conv, nil
}

func (r *fakeConvRepo) AddMessage(_ context.Context, convID uuid.UUID, role, content string) (*models.Conversation, *models.ConversationMessage, error) {
	if r.addErr != nil {
		return nil, nil, r.addErr
	}
	if role == "user" && r.conv != nil && (r.conv.IsClosed || r.conv.TurnCount >= models.MaxConversationTurns) {
		return nil, nil, models.ErrConversationTurnUnavailable
	}
	msg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           role,
		Content:        content,
	}
	r.msgs = append(r.msgs, *msg)
	if role == "user" && r.conv != nil {
		r.conv.TurnCount++
		if r.conv.TurnCount >= models.MaxConversationTurns {
			r.conv.IsClosed = true
		}
	}
	return r.conv, msg, nil
}

func (r *fakeConvRepo) ListMessages(_ context.Context, _ uuid.UUID) ([]models.ConversationMessage, error) {
	return r.msgs, nil
}

func (r *fakeConvRepo) Close(_ context.Context, _ uuid.UUID) (*models.Conversation, error) {
	if r.conv != nil {
		r.conv.IsClosed = true
	}
	return r.conv, nil
}

type fakeConvCrisis struct {
	result *CrisisResult
	err    error
}

func (f *fakeConvCrisis) Screen(_ context.Context, _, _ string) (*CrisisResult, error) {
	if f.result == nil && f.err == nil {
		return &CrisisResult{}, nil
	}
	return f.result, f.err
}

type fakeEntryReader struct {
	entry *models.Entry
	err   error
}

func (r *fakeEntryReader) GetByIDInternal(_ context.Context, id uuid.UUID) (*models.Entry, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.entry != nil && r.entry.ID == id {
		return r.entry, nil
	}
	return nil, nil
}

type fakeAnalysisReader struct {
	analysis *models.EntryAnalysis
	err      error
}

func (r *fakeAnalysisReader) GetByEntryID(_ context.Context, _ uuid.UUID) (*models.EntryAnalysis, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.analysis, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func convStubClaude() *ClaudeService {
	return NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "test"})
}

func newConvSvc(cr ConvRepository, er EntryStoreReader, ar AnalysisStoreReader) *ConversationService {
	return NewConversationService(cr, er, ar, convStubClaude())
}

func makeConvEntry(entryID uuid.UUID) *models.Entry {
	transcript := "I had a productive day."
	return &models.Entry{
		ID:         entryID,
		Status:     models.EntryStatusCompleted,
		Transcript: &transcript,
	}
}

func makeConvAnalysis(entryID uuid.UUID) *models.EntryAnalysis {
	return &models.EntryAnalysis{
		EntryID:    entryID,
		Reflection: "Sounds like a good day. What helped you most?",
	}
}

func newOpenConv(convID, entryID, userID uuid.UUID, turns int) *models.Conversation {
	return &models.Conversation{
		ID:        convID,
		EntryID:   entryID,
		UserID:    userID,
		TurnCount: turns,
		IsClosed:  false,
	}
}

// ── GetOrCreate tests ─────────────────────────────────────────────────────────

func TestConvService_GetOrCreate_NewConversation(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	cr := &fakeConvRepo{}
	svc := newConvSvc(cr, &fakeEntryReader{entry: makeOwnedConvEntry(entryID, userID)}, &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)})
	conv, err := svc.GetOrCreate(context.Background(), entryID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv == nil {
		t.Fatal("conversation must not be nil")
	}
	if conv.EntryID != entryID {
		t.Errorf("entry_id: want %v, got %v", entryID, conv.EntryID)
	}
	if conv.TurnCount != 0 {
		t.Errorf("new conversation must start with turn_count=0, got %d", conv.TurnCount)
	}
	if conv.IsClosed {
		t.Error("new conversation must not be closed")
	}
}

func TestConvService_GetOrCreate_IsIdempotent(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	cr := &fakeConvRepo{}
	svc := newConvSvc(cr, &fakeEntryReader{entry: makeOwnedConvEntry(entryID, userID)}, &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)})

	conv1, _ := svc.GetOrCreate(context.Background(), entryID, userID)
	conv2, _ := svc.GetOrCreate(context.Background(), entryID, userID)

	if conv1.ID != conv2.ID {
		t.Error("GetOrCreate must return the same conversation on repeated calls")
	}
}

// ── SendMessage / 3-turn cap tests ────────────────────────────────────────────

func TestConvService_SendMessage_FirstTurn_Succeeds(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 0)}
	er := &fakeEntryReader{entry: makeConvEntry(entryID)}
	ar := &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)}

	svc := newConvSvc(cr, er, ar)
	conv, err := svc.SendMessage(context.Background(), convID, userID, "Hello!", "IN")
	if err != nil {
		t.Fatalf("first turn must succeed: %v", err)
	}
	if conv == nil {
		t.Fatal("returned conversation must not be nil")
	}
}

func TestConvService_SendMessage_SecondTurn_Succeeds(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 1)}
	er := &fakeEntryReader{entry: makeConvEntry(entryID)}
	ar := &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)}

	svc := newConvSvc(cr, er, ar)
	conv, err := svc.SendMessage(context.Background(), convID, userID, "Second message!", "IN")
	if err != nil {
		t.Fatalf("second turn must succeed: %v", err)
	}
	if conv == nil {
		t.Fatal("returned conversation must not be nil")
	}
}

func TestConvService_SendMessage_ThirdTurn_ClosesConversation(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	// TurnCount=2 means the third user message will make it 3 (= MaxConversationTurns)
	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 2)}
	er := &fakeEntryReader{entry: makeConvEntry(entryID)}
	ar := &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)}

	svc := newConvSvc(cr, er, ar)
	conv, err := svc.SendMessage(context.Background(), convID, userID, "Third message!", "IN")
	if err != nil {
		t.Fatalf("third turn must succeed: %v", err)
	}
	if !conv.IsClosed {
		t.Error("conversation must be closed after the 3rd user turn")
	}
	if conv.TurnCount != models.MaxConversationTurns {
		t.Errorf("turn_count: want %d, got %d", models.MaxConversationTurns, conv.TurnCount)
	}
}

func TestConvService_SendMessage_ClosedConversation_ReturnsError(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	closedConv := &models.Conversation{
		ID:        convID,
		EntryID:   entryID,
		UserID:    userID,
		TurnCount: 3,
		IsClosed:  true,
	}
	cr := &fakeConvRepo{conv: closedConv}
	svc := newConvSvc(cr, &fakeEntryReader{}, &fakeAnalysisReader{})

	_, err := svc.SendMessage(context.Background(), convID, userID, "One more!", "IN")
	if err == nil {
		t.Error("closed conversation must reject new messages")
	}
	if !errors.Is(err, ErrConversationClosed) {
		t.Errorf("error: want ErrConversationClosed, got %v", err)
	}
}

func TestConvService_SendMessage_MaxTurnsNotClosed_ReturnsError(t *testing.T) {
	// Edge case: TurnCount is at max but IsClosed flag not yet set.
	// The service must check TurnCount too, not just IsClosed.
	entryID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	cr := &fakeConvRepo{conv: &models.Conversation{
		ID:        convID,
		EntryID:   entryID,
		UserID:    userID,
		TurnCount: models.MaxConversationTurns,
		IsClosed:  false, // not yet closed
	}}
	svc := newConvSvc(cr, &fakeEntryReader{}, &fakeAnalysisReader{})

	_, err := svc.SendMessage(context.Background(), convID, userID, "Extra turn!", "IN")
	if err == nil {
		t.Error("max turns reached must reject new messages")
	}
	if !errors.Is(err, ErrConversationClosed) {
		t.Errorf("error: want ErrConversationClosed, got %v", err)
	}
}

func TestConvService_SendMessage_NotFound_ReturnsError(t *testing.T) {
	cr := &fakeConvRepo{conv: nil}
	svc := newConvSvc(cr, &fakeEntryReader{}, &fakeAnalysisReader{})

	_, err := svc.SendMessage(context.Background(), uuid.New(), uuid.New(), "Hello", "IN")
	if err == nil {
		t.Error("not-found conversation must return an error")
	}
}

func TestConvService_SendMessage_WrongUser_ReturnsError(t *testing.T) {
	entryID := uuid.New()
	convID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()

	// Conv belongs to ownerID; call is made with otherID.
	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, ownerID, 0)}
	svc := newConvSvc(cr, &fakeEntryReader{}, &fakeAnalysisReader{})

	_, err := svc.SendMessage(context.Background(), convID, otherID, "Intrude!", "IN")
	if err == nil {
		t.Error("wrong user must not be able to send messages (ownership check)")
	}
}

func TestConvService_MaxConversationTurns_IsThree(t *testing.T) {
	// Explicitly assert the product invariant - do not change without a product decision.
	if models.MaxConversationTurns != 3 {
		t.Errorf("MaxConversationTurns must be 3 (ADR-006), got %d", models.MaxConversationTurns)
	}
}

// ── Reply content ─────────────────────────────────────────────────────────────

func TestConvService_SendMessage_ReturnsAssistantReply(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 0)}
	er := &fakeEntryReader{entry: makeConvEntry(entryID)}
	ar := &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)}

	svc := newConvSvc(cr, er, ar)
	_, err := svc.SendMessage(context.Background(), convID, userID, "Hello", "IN")
	if err != nil {
		t.Fatal(err)
	}

	// The fake conv repo accumulates messages: [user, assistant]
	if len(cr.msgs) < 2 {
		t.Errorf("want at least 2 messages (user + assistant), got %d", len(cr.msgs))
	}
	lastMsg := cr.msgs[len(cr.msgs)-1]
	if lastMsg.Role != "assistant" {
		t.Errorf("last message role: want assistant, got %s", lastMsg.Role)
	}
	if lastMsg.Content == "" {
		t.Error("assistant reply must not be empty")
	}
}

func makeOwnedConvEntry(entryID, userID uuid.UUID) *models.Entry {
	e := makeConvEntry(entryID)
	e.UserID = userID
	return e
}

// H4 / API contract: follow-ups only open on completed, non-crisis entries.
func TestConvService_GetOrCreate_RejectsPendingAndCrisisEntries(t *testing.T) {
	entryID, userID := uuid.New(), uuid.New()

	pending := makeOwnedConvEntry(entryID, userID)
	pending.Status = models.EntryStatusPending
	svc := newConvSvc(&fakeConvRepo{}, &fakeEntryReader{entry: pending}, &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)})
	if _, err := svc.GetOrCreate(context.Background(), entryID, userID); err == nil {
		t.Error("pending entry must not open a conversation")
	}

	crisis := makeConvAnalysis(entryID)
	crisis.IsCrisis = true
	svc = newConvSvc(&fakeConvRepo{}, &fakeEntryReader{entry: makeOwnedConvEntry(entryID, userID)}, &fakeAnalysisReader{analysis: crisis})
	if _, err := svc.GetOrCreate(context.Background(), entryID, userID); err == nil {
		t.Error("crisis entry must not open a conversation")
	}
}

// H4: a crisis message gets resources, closes the conversation, and never
// reaches the reflection model.
func TestConvService_SendMessage_CrisisMessage_ClosesWithResources(t *testing.T) {
	entryID, userID, convID := uuid.New(), uuid.New(), uuid.New()
	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 0)}
	svc := newConvSvc(cr, &fakeEntryReader{entry: makeConvEntry(entryID)}, &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)}).
		WithCrisisScreener(&fakeConvCrisis{result: &CrisisResult{Detected: true, Response: "helplines"}}, nil)

	conv, err := svc.SendMessage(context.Background(), convID, userID, "honestly I want to kill myself", "IN")
	if err != nil {
		t.Fatal(err)
	}
	if !conv.IsCrisis || !conv.IsClosed {
		t.Errorf("crisis reply must be flagged and close the conversation: crisis=%v closed=%v", conv.IsCrisis, conv.IsClosed)
	}
	last := cr.msgs[len(cr.msgs)-1]
	if last.Role != "assistant" || last.Content != "helplines" {
		t.Errorf("last message must be the crisis resources, got %+v", last)
	}
}

// H4 + ADR-002: a screener error fails safe to the crisis reply.
func TestConvService_SendMessage_ScreenerError_FailsSafe(t *testing.T) {
	entryID, userID, convID := uuid.New(), uuid.New(), uuid.New()
	cr := &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 0)}
	svc := newConvSvc(cr, &fakeEntryReader{entry: makeConvEntry(entryID)}, &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)}).
		WithCrisisScreener(&fakeConvCrisis{err: context.DeadlineExceeded}, nil)

	conv, err := svc.SendMessage(context.Background(), convID, userID, "it's all too much", "IN")
	if err != nil {
		t.Fatal(err)
	}
	if !conv.IsCrisis {
		t.Error("screener error must be treated as crisis")
	}
}

// H9: losing the turn-claim race surfaces as ErrConversationClosed.
func TestConvService_SendMessage_TurnRaceLost_ReturnsClosed(t *testing.T) {
	entryID, userID, convID := uuid.New(), uuid.New(), uuid.New()
	cr := &raceConvRepo{fakeConvRepo: &fakeConvRepo{conv: newOpenConv(convID, entryID, userID, 2)}}
	svc := newConvSvc(cr, &fakeEntryReader{entry: makeConvEntry(entryID)}, &fakeAnalysisReader{analysis: makeConvAnalysis(entryID)})
	if _, err := svc.SendMessage(context.Background(), convID, userID, "race", "IN"); !errors.Is(err, ErrConversationClosed) {
		t.Errorf("want ErrConversationClosed, got %v", err)
	}
}

// raceConvRepo reports an open conversation on read, but another request has
// already taken the last turn by the time AddMessage runs.
type raceConvRepo struct{ *fakeConvRepo }

func (r *raceConvRepo) AddMessage(_ context.Context, _ uuid.UUID, role, _ string) (*models.Conversation, *models.ConversationMessage, error) {
	if role == "user" {
		return nil, nil, models.ErrConversationTurnUnavailable
	}
	return r.conv, &models.ConversationMessage{}, nil
}
