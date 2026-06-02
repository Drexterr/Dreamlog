package workers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake implementations ──────────────────────────────────────────────────────

type fakeNudgeDispatcher struct {
	mu       sync.Mutex
	pending  []*models.Nudge
	tokens   map[uuid.UUID][]string // userID → FCM tokens
	tokenErr error
	sentIDs  []uuid.UUID
	failedID uuid.UUID
	failedMsg string
	sentErr  error
	failErr  error
}

func (f *fakeNudgeDispatcher) PendingDue(_ context.Context) ([]*models.Nudge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pending, nil
}

func (f *fakeNudgeDispatcher) GetDeviceTokens(_ context.Context, userID uuid.UUID) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tokenErr != nil {
		return nil, f.tokenErr
	}
	return f.tokens[userID], nil
}

func (f *fakeNudgeDispatcher) MarkSent(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sentErr != nil {
		return f.sentErr
	}
	f.sentIDs = append(f.sentIDs, id)
	return nil
}

func (f *fakeNudgeDispatcher) MarkFailed(_ context.Context, id uuid.UUID, msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failErr != nil {
		return f.failErr
	}
	f.failedID = id
	f.failedMsg = msg
	return nil
}

func (f *fakeNudgeDispatcher) wasSentID(id uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sentIDs {
		if s == id {
			return true
		}
	}
	return false
}

type fakeFCMSender struct {
	mu       sync.Mutex
	calls    []fcmCall
	sendErr  error
}

type fcmCall struct {
	token string
	title string
	body  string
	data  map[string]string
}

func (f *fakeFCMSender) SendToToken(_ context.Context, token, title, body string, data map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fcmCall{token: token, title: title, body: body, data: data})
	return f.sendErr
}

func (f *fakeFCMSender) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makeNudge(userID uuid.UUID, message string) *models.Nudge {
	return &models.Nudge{
		ID:          uuid.New(),
		UserID:      userID,
		Message:     message,
		ScheduledAt: time.Now().Add(-time.Minute), // due 1 minute ago
		Status:      models.NudgeStatusPending,
	}
}

func newSchedulerFixture() (*NudgeScheduler, *fakeNudgeDispatcher, *fakeFCMSender) {
	repo := &fakeNudgeDispatcher{
		tokens: make(map[uuid.UUID][]string),
	}
	fcm := &fakeFCMSender{}
	sched := NewNudgeScheduler(repo, fcm, zap.NewNop())
	return sched, repo, fcm
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestNudgeScheduler_NoPendingNudges_FCMNotCalled(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	repo.pending = nil

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("no pending nudges: want 0 FCM calls, got %d", fcm.callCount())
	}
}

func TestNudgeScheduler_DispatchesPendingNudge(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Good morning! Reflect on yesterday.")
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"token-abc"}

	sched.tick(context.Background())

	if fcm.callCount() != 1 {
		t.Fatalf("want 1 FCM call, got %d", fcm.callCount())
	}
	call := fcm.calls[0]
	if call.token != "token-abc" {
		t.Errorf("token: want token-abc, got %s", call.token)
	}
	if call.body != n.Message {
		t.Errorf("body: want %q, got %q", n.Message, call.body)
	}
	if call.data["type"] != "morning_nudge" {
		t.Errorf("data.type: want morning_nudge, got %s", call.data["type"])
	}
	if call.data["nudge_id"] != n.ID.String() {
		t.Errorf("data.nudge_id: want %s, got %s", n.ID.String(), call.data["nudge_id"])
	}
}

func TestNudgeScheduler_MarksNudgeSentAfterSuccess(t *testing.T) {
	sched, repo, _ := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Morning nudge text")
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"tok-1"}

	sched.tick(context.Background())

	if !repo.wasSentID(n.ID) {
		t.Error("nudge must be marked sent after successful FCM dispatch")
	}
}

func TestNudgeScheduler_NoTokens_MarksNudgeSentAnyway(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Some message")
	repo.pending = []*models.Nudge{n}
	// No tokens registered for this user

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("no tokens: want 0 FCM calls, got %d", fcm.callCount())
	}
	if !repo.wasSentID(n.ID) {
		t.Error("nudge with no devices must still be marked sent to avoid re-dispatch")
	}
}

func TestNudgeScheduler_GetTokensError_MarksNudgeFailed(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Nudge message")
	repo.pending = []*models.Nudge{n}
	repo.tokenErr = errors.New("db connection lost")

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("token error: want 0 FCM calls, got %d", fcm.callCount())
	}
	if repo.failedID != n.ID {
		t.Errorf("nudge must be marked failed when GetDeviceTokens errors; got id %v", repo.failedID)
	}
	if repo.failedMsg == "" {
		t.Error("failed message must not be empty")
	}
}

func TestNudgeScheduler_FCMError_MarksNudgeFailed(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Morning nudge")
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"tok-1"}
	fcm.sendErr = errors.New("FCM service unavailable")

	sched.tick(context.Background())

	if repo.wasSentID(n.ID) {
		t.Error("nudge with FCM error must not be marked sent")
	}
	if repo.failedID != n.ID {
		t.Errorf("nudge must be marked failed on FCM error; got id %v", repo.failedID)
	}
}

func TestNudgeScheduler_MultipleTokens_AllReceiveFCM(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Morning message")
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"tok-1", "tok-2", "tok-3"}

	sched.tick(context.Background())

	if fcm.callCount() != 3 {
		t.Errorf("3 tokens: want 3 FCM calls, got %d", fcm.callCount())
	}
}

func TestNudgeScheduler_MultipleNudges_AllDispatched(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()

	uid1, uid2 := uuid.New(), uuid.New()
	n1 := makeNudge(uid1, "msg 1")
	n2 := makeNudge(uid2, "msg 2")
	repo.pending = []*models.Nudge{n1, n2}
	repo.tokens[uid1] = []string{"tok-a"}
	repo.tokens[uid2] = []string{"tok-b"}

	sched.tick(context.Background())

	if fcm.callCount() != 2 {
		t.Errorf("2 nudges: want 2 FCM calls, got %d", fcm.callCount())
	}
	if !repo.wasSentID(n1.ID) {
		t.Error("nudge 1 must be marked sent")
	}
	if !repo.wasSentID(n2.ID) {
		t.Error("nudge 2 must be marked sent")
	}
}

func TestNudgeScheduler_PartialFCMFailure_FailsEntireNudge(t *testing.T) {
	// If any token fails, the nudge is marked failed (lastErr wins).
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "message")
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"tok-good", "tok-bad"}

	callNum := 0
	fcm.sendErr = nil
	// Intercept: first call succeeds, second fails.
	// We simulate this by making SendToToken stateful via a closure mock.
	// Since fakeFCMSender uses a fixed error, use a custom one here.
	type partialFCM struct{ callCount int }
	partialSender := &partialFakeFCM{}
	sched.fcm = partialSender
	_ = callNum // keep linter quiet

	sched.tick(context.Background())

	// Second token always fails with partialFakeFCM. Nudge must be marked failed.
	if repo.wasSentID(n.ID) {
		t.Error("nudge must not be marked sent when any token fails")
	}
}

// partialFakeFCM fails on the second and subsequent calls.
type partialFakeFCM struct {
	mu    sync.Mutex
	calls int
}

func (p *partialFakeFCM) SendToToken(_ context.Context, token, _, _ string, _ map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.calls >= 2 {
		return errors.New("FCM error on second token: " + token)
	}
	return nil
}

// ── Run loop: fires immediately on start ──────────────────────────────────────

func TestNudgeScheduler_RunFiresOnStart(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "Morning nudge")
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"tok-x"}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Give the goroutine a moment to fire the initial tick.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if fcm.callCount() == 0 {
		t.Error("Run must dispatch nudges immediately on startup (not wait for first tick)")
	}
}

// ── truncateToken ─────────────────────────────────────────────────────────────

func TestTruncateToken_LongToken(t *testing.T) {
	token := "abcdefghijklmnopqrstuvwxyz"
	result := truncateToken(token)
	if len([]rune(result)) > 12 {
		t.Errorf("truncated token must be ≤ 12 runes, got %d: %q", len([]rune(result)), result)
	}
	if result[:8] != token[:8] {
		t.Error("truncated token must start with first 8 chars of original")
	}
}

func TestTruncateToken_ShortToken_Unchanged(t *testing.T) {
	token := "short"
	if truncateToken(token) != token {
		t.Errorf("short token must be unchanged: want %q, got %q", token, truncateToken(token))
	}
}

func TestTruncateToken_ExactlyTwelveChars_Unchanged(t *testing.T) {
	token := "123456789012"
	if truncateToken(token) != token {
		t.Errorf("12-char token must be unchanged: want %q, got %q", token, truncateToken(token))
	}
}
