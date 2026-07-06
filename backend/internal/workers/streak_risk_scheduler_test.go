package workers

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeStreakRiskRepo struct {
	mu         sync.Mutex
	candidates []repositories.StreakRiskUser
	fetchErr   error
	created    []*models.Nudge
	createErr  error
	tokens     map[uuid.UUID][]string
	sentIDs    []uuid.UUID
	failedIDs  []uuid.UUID
}

func (f *fakeStreakRiskRepo) StreakRiskUsersAtLocalHour(_ context.Context, _ int) ([]repositories.StreakRiskUser, error) {
	return f.candidates, f.fetchErr
}

func (f *fakeStreakRiskRepo) CreateWithType(_ context.Context, userID uuid.UUID, entryID *uuid.UUID, message string, scheduledAt time.Time, timezone, nudgeType string) (*models.Nudge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return nil, f.createErr
	}
	n := &models.Nudge{
		ID: uuid.New(), UserID: userID, EntryID: entryID,
		Message: message, ScheduledAt: scheduledAt, Timezone: timezone,
		Status: models.NudgeStatusPending, NudgeType: nudgeType,
	}
	f.created = append(f.created, n)
	return n, nil
}

func (f *fakeStreakRiskRepo) GetDeviceTokens(_ context.Context, userID uuid.UUID) ([]string, error) {
	return f.tokens[userID], nil
}

func (f *fakeStreakRiskRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sentIDs = append(f.sentIDs, id)
	return nil
}

func (f *fakeStreakRiskRepo) MarkFailed(_ context.Context, id uuid.UUID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failedIDs = append(f.failedIDs, id)
	return nil
}

type fakeStreakLookup struct {
	streaks map[uuid.UUID]*models.StreakInfo
	err     error
}

func (f *fakeStreakLookup) StreakInfo(_ context.Context, userID uuid.UUID) (*models.StreakInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	if s, ok := f.streaks[userID]; ok {
		return s, nil
	}
	return &models.StreakInfo{}, nil
}

func newStreakRiskFixture() (*StreakRiskScheduler, *fakeStreakRiskRepo, *fakeStreakLookup, *fakeFCMSender) {
	repo := &fakeStreakRiskRepo{tokens: map[uuid.UUID][]string{}}
	streaks := &fakeStreakLookup{streaks: map[uuid.UUID]*models.StreakInfo{}}
	fcm := &fakeFCMSender{}
	sched := NewStreakRiskScheduler(repo, streaks, fcm, zap.NewNop())
	return sched, repo, streaks, fcm
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestStreakRisk_NoCandidates_NothingSent(t *testing.T) {
	sched, repo, _, fcm := newStreakRiskFixture()

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("want 0 FCM calls, got %d", fcm.callCount())
	}
	if len(repo.created) != 0 {
		t.Errorf("want no nudge rows created, got %d", len(repo.created))
	}
}

func TestStreakRisk_ActiveStreak_SendsWithCountInMessage(t *testing.T) {
	sched, repo, streaks, fcm := newStreakRiskFixture()
	userID := uuid.New()
	repo.candidates = []repositories.StreakRiskUser{{UserID: userID, Timezone: "Asia/Kolkata"}}
	repo.tokens[userID] = []string{"tok-1"}
	streaks.streaks[userID] = &models.StreakInfo{CurrentStreak: 5}

	sched.tick(context.Background())

	if fcm.callCount() != 1 {
		t.Fatalf("want 1 FCM call, got %d", fcm.callCount())
	}
	call := fcm.calls[0]
	if !strings.Contains(call.body, "5 days") {
		t.Errorf("message must mention the streak length, got %q", call.body)
	}
	if call.data["type"] != "streak_risk" {
		t.Errorf("data.type: want streak_risk, got %s", call.data["type"])
	}
	if len(repo.created) != 1 || repo.created[0].NudgeType != models.NudgeTypeStreakRisk {
		t.Error("a streak_risk nudge row must be created before sending")
	}
	if len(repo.sentIDs) != 1 {
		t.Error("nudge must be marked sent after successful dispatch")
	}
}

func TestStreakRisk_ShortStreak_Skipped(t *testing.T) {
	sched, repo, streaks, fcm := newStreakRiskFixture()
	userID := uuid.New()
	repo.candidates = []repositories.StreakRiskUser{{UserID: userID, Timezone: "UTC"}}
	repo.tokens[userID] = []string{"tok-1"}
	streaks.streaks[userID] = &models.StreakInfo{CurrentStreak: minStreakForRiskNudge - 1}

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("streak below minimum: want 0 FCM calls, got %d", fcm.callCount())
	}
	if len(repo.created) != 0 {
		t.Error("streak below minimum: no nudge row must be created")
	}
}

func TestStreakRisk_StreakLookupError_Skipped(t *testing.T) {
	sched, repo, streaks, fcm := newStreakRiskFixture()
	userID := uuid.New()
	repo.candidates = []repositories.StreakRiskUser{{UserID: userID, Timezone: "UTC"}}
	streaks.err = errors.New("db down")

	sched.tick(context.Background())

	if fcm.callCount() != 0 || len(repo.created) != 0 {
		t.Error("streak lookup error: user must be skipped entirely")
	}
}

func TestStreakRisk_NoTokens_MarkedSentWithoutFCM(t *testing.T) {
	sched, repo, streaks, fcm := newStreakRiskFixture()
	userID := uuid.New()
	repo.candidates = []repositories.StreakRiskUser{{UserID: userID, Timezone: "UTC"}}
	streaks.streaks[userID] = &models.StreakInfo{CurrentStreak: 7}

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("no tokens: want 0 FCM calls, got %d", fcm.callCount())
	}
	if len(repo.sentIDs) != 1 {
		t.Error("no tokens: nudge must still be marked sent to prevent re-dispatch")
	}
}

func TestStreakRisk_FCMError_MarkedFailed(t *testing.T) {
	sched, repo, streaks, fcm := newStreakRiskFixture()
	userID := uuid.New()
	repo.candidates = []repositories.StreakRiskUser{{UserID: userID, Timezone: "UTC"}}
	repo.tokens[userID] = []string{"tok-1"}
	streaks.streaks[userID] = &models.StreakInfo{CurrentStreak: 4}
	fcm.sendErr = errors.New("fcm unavailable")

	sched.tick(context.Background())

	if len(repo.failedIDs) != 1 {
		t.Error("FCM error: nudge must be marked failed")
	}
	if len(repo.sentIDs) != 0 {
		t.Error("FCM error: nudge must not be marked sent")
	}
}
