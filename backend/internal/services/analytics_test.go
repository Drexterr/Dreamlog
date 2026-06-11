package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// ── Fake repo ─────────────────────────────────────────────────────────────────

type fakeAnalyticsRepo struct {
	events []struct {
		userID     *uuid.UUID
		event      string
		properties map[string]any
	}
	err error
}

func (r *fakeAnalyticsRepo) Insert(_ context.Context, userID *uuid.UUID, event string, props map[string]any) error {
	r.events = append(r.events, struct {
		userID     *uuid.UUID
		event      string
		properties map[string]any
	}{userID, event, props})
	return r.err
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestAnalytics_Track_StoresEvent(t *testing.T) {
	repo := &fakeAnalyticsRepo{}
	svc := NewAnalyticsService(repo)
	uid := uuid.New()

	svc.TrackUser(context.Background(), uid, EventEntryCompleted, map[string]any{"mode": "processing"})

	if len(repo.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(repo.events))
	}
	ev := repo.events[0]
	if ev.event != EventEntryCompleted {
		t.Errorf("wrong event name: %s", ev.event)
	}
	if *ev.userID != uid {
		t.Errorf("wrong user id")
	}
	if ev.properties["mode"] != "processing" {
		t.Errorf("properties not stored correctly")
	}
}

func TestAnalytics_Track_NilPropsBecomesEmptyMap(t *testing.T) {
	repo := &fakeAnalyticsRepo{}
	svc := NewAnalyticsService(repo)

	svc.TrackUser(context.Background(), uuid.New(), EventSignup, nil)

	if len(repo.events) == 0 {
		t.Fatal("event not recorded")
	}
	if repo.events[0].properties == nil {
		t.Error("nil props must be converted to empty map")
	}
}

func TestAnalytics_Track_AnonymousEvent_NilUserID(t *testing.T) {
	repo := &fakeAnalyticsRepo{}
	svc := NewAnalyticsService(repo)

	svc.Track(context.Background(), nil, EventPaywallViewed, nil)

	if len(repo.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(repo.events))
	}
	if repo.events[0].userID != nil {
		t.Error("anonymous event must have nil userID")
	}
}

func TestAnalytics_Track_RepoError_DoesNotPanic(t *testing.T) {
	repo := &fakeAnalyticsRepo{err: errors.New("db down")}
	svc := NewAnalyticsService(repo)

	// Must not panic or return error even when repo fails.
	svc.TrackUser(context.Background(), uuid.New(), EventEntryCompleted, nil)
}

func TestAnalytics_Track_NilService_DoesNotPanic(t *testing.T) {
	var svc *AnalyticsService
	// Calling Track on a nil service must not panic.
	svc.TrackUser(context.Background(), uuid.New(), EventSignup, nil)
}

func TestAnalytics_Track_NilRepo_DoesNotPanic(t *testing.T) {
	svc := NewAnalyticsService(nil)
	svc.TrackUser(context.Background(), uuid.New(), EventSignup, nil)
}

func TestAnalytics_EventConstants_Defined(t *testing.T) {
	// Guard: the minimum event set from PRICING.md §6c must all be defined.
	required := []string{
		EventSignup, EventOnboardingCompleted, EventEntryRecorded, EventEntryCompleted,
		EventEntryFailed, EventReflectionViewed, EventFollowupStarted, EventFollowupTurn,
		EventTherapySessionStarted, EventTherapySessionEnded, EventPaywallViewed,
		EventPurchaseInitiated, EventPurchaseCompleted, EventPurchaseFailed,
		EventPlanChanged, EventEntryLimitHit, EventShareCreated, EventInsightCardShared,
		EventExportDownloaded,
	}
	for _, name := range required {
		if name == "" {
			t.Errorf("event constant is empty string — all minimum events must have non-empty names")
		}
	}
}
