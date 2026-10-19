package workers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type expansionReengagementRepo struct {
	fakeNudgeDispatcher
	users     []repositories.LapsedUser
	fetchErr  error
	createErr map[uuid.UUID]error
	created   []*models.Nudge
	hours     int
	queries   int
	onFetch   func(context.Context)
}

type expansionFCMFunc func(context.Context, string, string, string, map[string]string) error

func (f expansionFCMFunc) SendToToken(ctx context.Context, token, title, body string, data map[string]string) error {
	return f(ctx, token, title, body, data)
}

func TestExpansionReengagementPartialPushFailure(t *testing.T) {
	u := uuid.New()
	r := &expansionReengagementRepo{users: []repositories.LapsedUser{{UserID: u}}}
	r.tokens = map[uuid.UUID][]string{u: {"bad", "good"}}
	var tokens []string
	fcm := expansionFCMFunc(func(_ context.Context, token, _, _ string, _ map[string]string) error {
		if len(r.created) != 1 {
			t.Fatal("push attempted before dedup row was created")
		}
		tokens = append(tokens, token)
		if token == "bad" {
			return errors.New("first device rejected")
		}
		return nil
	})
	NewReengagementScheduler(r, fcm, zap.NewNop()).tick(context.Background())
	if len(tokens) != 2 || tokens[1] != "good" || len(r.sentIDs) != 0 || r.failedID != r.created[0].ID || r.failedMsg != "send error: first device rejected" {
		t.Fatalf("later success erased partial failure: tokens=%v failed=%s message=%q", tokens, r.failedID, r.failedMsg)
	}
}

func (r *expansionReengagementRepo) LapsedUsersAtNudgeHour(ctx context.Context, hours int) ([]repositories.LapsedUser, error) {
	r.hours, r.queries = hours, r.queries+1
	if r.onFetch != nil {
		r.onFetch(ctx)
	}
	return r.users, r.fetchErr
}

func (r *expansionReengagementRepo) CreateWithType(_ context.Context, user uuid.UUID, entry *uuid.UUID, msg string, at time.Time, zone, kind string) (*models.Nudge, error) {
	if err := r.createErr[user]; err != nil {
		return nil, err
	}
	n := &models.Nudge{ID: uuid.New(), UserID: user, EntryID: entry, Message: msg, ScheduledAt: at, Timezone: zone, NudgeType: kind}
	r.created = append(r.created, n)
	return n, nil
}

func TestExpansionReengagementEligibilityAndPayload(t *testing.T) {
	u := uuid.New()
	r := &expansionReengagementRepo{users: []repositories.LapsedUser{{UserID: u, Timezone: "Asia/Kolkata"}}}
	r.tokens = map[uuid.UUID][]string{u: {"device-a", "device-b"}}
	fcm := &fakeFCMSender{}
	s := NewReengagementScheduler(r, fcm, zap.NewNop())
	before := time.Now().UTC()
	s.tick(context.Background())
	if r.hours != 26 || r.queries != 1 || len(r.created) != 1 {
		t.Fatalf("eligibility query/rows: hours=%d queries=%d rows=%d", r.hours, r.queries, len(r.created))
	}
	n := r.created[0]
	if n.UserID != u || n.EntryID != nil || n.Timezone != "Asia/Kolkata" || n.NudgeType != "reengagement" || n.ScheduledAt.Before(before) || n.ScheduledAt.After(time.Now().UTC()) || n.ScheduledAt.Location() != time.UTC {
		t.Fatalf("incorrect nudge: %+v", n)
	}
	if fcm.callCount() != 2 || len(r.sentIDs) != 1 || r.sentIDs[0] != n.ID {
		t.Fatalf("dispatch not completed: calls=%d sent=%v", fcm.callCount(), r.sentIDs)
	}
	for i, call := range fcm.calls {
		if call.token != r.tokens[u][i] || call.title != "Ode" || call.body != n.Message || call.data["type"] != "reengagement" || call.data["nudge_id"] != n.ID.String() {
			t.Fatalf("incorrect push: %+v", call)
		}
	}
	if pickMessage(u.String()) != n.Message {
		t.Fatal("message is not stable within the day")
	}
	found := false
	for _, msg := range reengagementMessages {
		found = found || msg == n.Message
	}
	if !found {
		t.Fatalf("message outside curated pool: %q", n.Message)
	}
}

func TestExpansionReengagementFailures(t *testing.T) {
	for _, mode := range []string{"no eligible users", "fetch", "create", "no devices", "tokens", "push", "mark sent", "mark failed"} {
		t.Run(mode, func(t *testing.T) {
			u, next := uuid.New(), uuid.New()
			r := &expansionReengagementRepo{users: []repositories.LapsedUser{{UserID: u}, {UserID: next}}, createErr: map[uuid.UUID]error{}}
			r.tokens = map[uuid.UUID][]string{u: {"a", "b"}, next: {"c"}}
			fcm := &fakeFCMSender{}
			switch mode {
			case "no eligible users":
				r.users = nil
			case "fetch":
				r.fetchErr = errors.New("query unavailable")
			case "create":
				r.createErr[u] = errors.New("insert unavailable")
			case "no devices":
				r.tokens = nil
			case "tokens":
				r.tokenErr = errors.New("token lookup unavailable")
			case "push":
				fcm.sendErr = errors.New("push rejected")
			case "mark sent":
				r.sentErr = errors.New("update unavailable")
			case "mark failed":
				r.failErr = errors.New("update unavailable")
				fcm.sendErr = errors.New("push rejected")
			}
			NewReengagementScheduler(r, fcm, zap.NewNop()).tick(context.Background())
			switch mode {
			case "no eligible users", "fetch":
				if len(r.created) != 0 || fcm.callCount() != 0 {
					t.Fatal("ineligible/query-error batch dispatched")
				}
			case "create":
				if len(r.created) != 1 || r.created[0].UserID != next || fcm.callCount() != 1 || len(r.sentIDs) != 1 {
					t.Fatal("create failure did not isolate the first user")
				}
			case "no devices":
				if len(r.sentIDs) != 2 || fcm.callCount() != 0 {
					t.Fatal("device-free nudges must be consumed without pushes")
				}
			case "tokens":
				if fcm.callCount() != 0 || r.failedID != r.created[1].ID || r.failedMsg != "get tokens: token lookup unavailable" {
					t.Fatalf("token failure state: %+v", &r.fakeNudgeDispatcher)
				}
			case "push":
				if fcm.callCount() != 3 || len(r.sentIDs) != 0 || !strings.Contains(r.failedMsg, "send error: push rejected") {
					t.Fatal("push failure must try all devices and fail nudge")
				}
			case "mark sent", "mark failed":
				if fcm.callCount() != 3 || len(r.created) != 2 {
					t.Fatal("status write error stopped the batch")
				}
			}
		})
	}
}

func TestExpansionReengagementRunCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &expansionReengagementRepo{onFetch: func(got context.Context) {
		if got != ctx {
			t.Error("scheduler did not propagate context")
		}
		cancel()
	}}
	done := make(chan struct{})
	go func() { defer close(done); NewReengagementScheduler(r, &fakeFCMSender{}, zap.NewNop()).Run(ctx) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after initial tick cancellation")
	}
	if r.queries != 1 {
		t.Fatalf("want one immediate eligibility query, got %d", r.queries)
	}
}
