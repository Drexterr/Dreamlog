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

type fakePlanExpiryRepo struct {
	mu         sync.Mutex
	soon       []repositories.PlanExpiryUser
	soonErr    error
	expired    []repositories.PlanExpiryUser
	expiredErr error
	createErr  error
	created    []*models.Nudge
	tokens     map[uuid.UUID][]string
	sentIDs    []uuid.UUID
	failedIDs  []uuid.UUID
}

func (f *fakePlanExpiryRepo) PlanExpiringSoonUsersAtLocalHour(_ context.Context, _, _ int) ([]repositories.PlanExpiryUser, error) {
	return f.soon, f.soonErr
}

func (f *fakePlanExpiryRepo) PlanExpiredUsersAtLocalHour(_ context.Context, _ int) ([]repositories.PlanExpiryUser, error) {
	return f.expired, f.expiredErr
}

func (f *fakePlanExpiryRepo) CreateWithType(_ context.Context, userID uuid.UUID, entryID *uuid.UUID, message string, scheduledAt time.Time, timezone, nudgeType string) (*models.Nudge, error) {
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

func (f *fakePlanExpiryRepo) GetDeviceTokens(_ context.Context, userID uuid.UUID) ([]string, error) {
	return f.tokens[userID], nil
}

func (f *fakePlanExpiryRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sentIDs = append(f.sentIDs, id)
	return nil
}

func (f *fakePlanExpiryRepo) MarkFailed(_ context.Context, id uuid.UUID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failedIDs = append(f.failedIDs, id)
	return nil
}

func newPlanExpiryFixture() (*PlanExpiryScheduler, *fakePlanExpiryRepo, *fakeFCMSender) {
	repo := &fakePlanExpiryRepo{tokens: map[uuid.UUID][]string{}}
	fcm := &fakeFCMSender{}
	sched := NewPlanExpiryScheduler(repo, fcm, zap.NewNop())
	return sched, repo, fcm
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestPlanExpiry_NoCandidates_NothingSent(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("want 0 FCM calls, got %d", fcm.callCount())
	}
	if len(repo.created) != 0 {
		t.Errorf("want no nudge rows created, got %d", len(repo.created))
	}
}

func TestPlanExpiry_ExpiringSoon_SendsAndCreatesNudge(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()
	userID := uuid.New()
	repo.soon = []repositories.PlanExpiryUser{
		{UserID: userID, Timezone: "Asia/Kolkata", Plan: "pro", PlanExpires: time.Now().Add(48 * time.Hour)},
	}
	repo.tokens[userID] = []string{"tok-1"}

	sched.tick(context.Background())

	if fcm.callCount() != 1 {
		t.Fatalf("want 1 FCM call, got %d", fcm.callCount())
	}
	if fcm.calls[0].data["type"] != models.NudgeTypePlanExpiringSoon {
		t.Errorf("data.type: want %s, got %s", models.NudgeTypePlanExpiringSoon, fcm.calls[0].data["type"])
	}
	if len(repo.created) != 1 || repo.created[0].NudgeType != models.NudgeTypePlanExpiringSoon {
		t.Error("a plan_expiring_soon nudge row must be created before sending")
	}
	if len(repo.sentIDs) != 1 {
		t.Error("nudge must be marked sent after successful dispatch")
	}
}

func TestPlanExpiry_Expired_SendsAndCreatesNudge(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()
	userID := uuid.New()
	repo.expired = []repositories.PlanExpiryUser{
		{UserID: userID, Timezone: "UTC", Plan: "pro", PlanExpires: time.Now().Add(-1 * time.Hour)},
	}
	repo.tokens[userID] = []string{"tok-1"}

	sched.tick(context.Background())

	if fcm.callCount() != 1 {
		t.Fatalf("want 1 FCM call, got %d", fcm.callCount())
	}
	if fcm.calls[0].data["type"] != models.NudgeTypePlanExpired {
		t.Errorf("data.type: want %s, got %s", models.NudgeTypePlanExpired, fcm.calls[0].data["type"])
	}
	if len(repo.created) != 1 || repo.created[0].NudgeType != models.NudgeTypePlanExpired {
		t.Error("a plan_expired nudge row must be created before sending")
	}
}

func TestPlanExpiry_BothListsNonEmpty_SendsBoth(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()
	soonUser, expiredUser := uuid.New(), uuid.New()
	repo.soon = []repositories.PlanExpiryUser{{UserID: soonUser, Timezone: "UTC", Plan: "pro", PlanExpires: time.Now().Add(72 * time.Hour)}}
	repo.expired = []repositories.PlanExpiryUser{{UserID: expiredUser, Timezone: "UTC", Plan: "plus", PlanExpires: time.Now().Add(-2 * time.Hour)}}
	repo.tokens[soonUser] = []string{"tok-soon"}
	repo.tokens[expiredUser] = []string{"tok-expired"}

	sched.tick(context.Background())

	if fcm.callCount() != 2 {
		t.Fatalf("want 2 FCM calls (one per list), got %d", fcm.callCount())
	}
	if len(repo.created) != 2 {
		t.Fatalf("want 2 nudge rows created, got %d", len(repo.created))
	}
}

func TestPlanExpiry_NoTokens_MarkedSentWithoutFCM(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()
	userID := uuid.New()
	repo.soon = []repositories.PlanExpiryUser{{UserID: userID, Timezone: "UTC", Plan: "pro", PlanExpires: time.Now().Add(24 * time.Hour)}}

	sched.tick(context.Background())

	if fcm.callCount() != 0 {
		t.Errorf("no tokens: want 0 FCM calls, got %d", fcm.callCount())
	}
	if len(repo.sentIDs) != 1 {
		t.Error("no tokens: nudge must still be marked sent to prevent re-dispatch")
	}
}

func TestPlanExpiry_FCMError_MarkedFailed(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()
	userID := uuid.New()
	repo.expired = []repositories.PlanExpiryUser{{UserID: userID, Timezone: "UTC", Plan: "pro", PlanExpires: time.Now().Add(-30 * time.Minute)}}
	repo.tokens[userID] = []string{"tok-1"}
	fcm.sendErr = errors.New("fcm unavailable")

	sched.tick(context.Background())

	if len(repo.failedIDs) != 1 {
		t.Error("FCM error: nudge must be marked failed")
	}
	if len(repo.sentIDs) != 0 {
		t.Error("FCM error: nudge must not be marked sent")
	}
}

func TestPlanExpiry_ExpiringSoonFetchError_ExpiredStillProcessed(t *testing.T) {
	sched, repo, fcm := newPlanExpiryFixture()
	userID := uuid.New()
	repo.soonErr = errors.New("db down")
	repo.expired = []repositories.PlanExpiryUser{{UserID: userID, Timezone: "UTC", Plan: "pro", PlanExpires: time.Now().Add(-1 * time.Hour)}}
	repo.tokens[userID] = []string{"tok-1"}

	sched.tick(context.Background())

	if fcm.callCount() != 1 {
		t.Errorf("expiring-soon fetch failing must not block the expired list: want 1 FCM call, got %d", fcm.callCount())
	}
}

func TestExpiringSoonMessage_MentionsDayCount(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"today", -1 * time.Hour, "today"},
		{"tomorrow", 20 * time.Hour, "tomorrow"},
		{"three days", 65 * time.Hour, "3 days"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := expiringSoonMessage(time.Now().Add(tc.in))
			if !strings.Contains(msg, tc.want) {
				t.Errorf("expiringSoonMessage(%v) = %q, want substring %q", tc.in, msg, tc.want)
			}
		})
	}
}
