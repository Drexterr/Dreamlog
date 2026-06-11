package services

import (
	"context"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// ── ListSessions ──────────────────────────────────────────────────────────────

func TestTherapy_ListSessions_ReturnsSummariesNewestFirst(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()

	first := startSession(t, svc, userID)
	if _, err := svc.EndSession(context.Background(), first.ID, userID); err != nil {
		t.Fatal(err)
	}
	second := startSession(t, svc, userID)

	resp, err := svc.ListSessions(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(resp.Sessions))
	}
	ids := map[uuid.UUID]bool{}
	for _, s := range resp.Sessions {
		ids[s.ID] = true
	}
	if !ids[first.ID] || !ids[second.ID] {
		t.Error("list must contain both sessions")
	}
}

func TestTherapy_ListSessions_EmptyForNewUser(t *testing.T) {
	svc := newStubTherapyService(newFakeTherapyRepo(), nil)
	resp, err := svc.ListSessions(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("expected empty list, got %d", len(resp.Sessions))
	}
}

// ── GetSession ────────────────────────────────────────────────────────────────

func TestTherapy_GetSession_ReturnsMessages(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)

	if _, err := svc.SendMessage(context.Background(), session.ID, userID, models.SendTherapyMessageRequest{
		Content: "I had a calm day.", InputMode: "text",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.GetSession(context.Background(), session.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("expected user + assistant messages, got %d", len(got.Messages))
	}
	if got.Messages[0].Role != "user" || got.Messages[1].Role != "assistant" {
		t.Error("messages must be in order: user then assistant")
	}
}

func TestTherapy_GetSession_NotFound(t *testing.T) {
	svc := newStubTherapyService(newFakeTherapyRepo(), nil)
	if _, err := svc.GetSession(context.Background(), uuid.New(), uuid.New()); err != ErrTherapyNotFound {
		t.Fatalf("expected ErrTherapyNotFound, got %v", err)
	}
}

func TestTherapy_GetSession_AutoExpiresPastActiveSession(t *testing.T) {
	repo := newFakeTherapyRepo()
	svc := newStubTherapyService(repo, nil)
	userID := uuid.New()
	session := startSession(t, svc, userID)

	// Force the session past its expiry.
	repo.sessions[session.ID].ExpiresAt = time.Now().Add(-time.Minute)

	got, err := svc.GetSession(context.Background(), session.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.TherapyStatusExpired {
		t.Errorf("active session past expiry must auto-expire on read, got %s", got.Status)
	}
	if got.TimeRemainingSec != 0 {
		t.Errorf("expired session must report 0 time remaining, got %d", got.TimeRemainingSec)
	}
}
