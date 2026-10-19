//go:build integration

package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestIntegration_JourneyPersistenceAndRollback(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewJourneyRepository(db)
	prompts := []string{"first QA prompt", "second QA prompt"}
	s, err := r.CreateSession(ctx, owner, "stress_relief", prompts)
	if err != nil || s == nil || s.UserID != owner || s.TotalSteps != 2 || s.CurrentStep != 0 || s.Status != models.JourneyStatusInProgress {
		t.Fatalf("create: %+v %v", s, err)
	}
	got, err := r.GetSession(ctx, s.ID, owner)
	if err != nil || len(got.Steps) != 2 || got.Steps[0].Prompt != prompts[0] || got.Steps[1].StepIndex != 1 || got.Steps[0].Completed || got.Steps[0].EntryID != nil {
		t.Fatalf("steps: %+v %v", got, err)
	}
	if got, err := r.GetSession(ctx, s.ID, other); !errors.Is(err, pgx.ErrNoRows) || got != nil {
		t.Fatalf("foreign: %+v %v", got, err)
	}
	entry := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/journey','completed')`, entry, owner); err != nil {
		t.Fatal(err)
	}
	if err := r.AdvanceStep(ctx, s.ID, 0, entry, 1, false); err != nil {
		t.Fatal(err)
	}
	got, err = r.GetSession(ctx, s.ID, owner)
	if err != nil || got.CurrentStep != 1 || got.Status != models.JourneyStatusInProgress || !got.Steps[0].Completed || got.Steps[0].EntryID == nil || *got.Steps[0].EntryID != entry || got.Steps[1].Completed {
		t.Fatalf("advance: %+v %v", got, err)
	}
	if err := r.AdvanceStep(ctx, s.ID, 1, uuid.New(), 2, true); err == nil {
		t.Fatal("missing entry accepted")
	}
	got, err = r.GetSession(ctx, s.ID, owner)
	if err != nil || got.CurrentStep != 1 || got.Steps[1].Completed {
		t.Fatalf("rollback: %+v %v", got, err)
	}
	if err := r.AdvanceStep(ctx, s.ID, 1, entry, 2, true); err != nil {
		t.Fatal(err)
	}
	got, err = r.GetSession(ctx, s.ID, owner)
	if err != nil || got.CurrentStep != 2 || got.Status != models.JourneyStatusCompleted || !got.Steps[1].Completed {
		t.Fatalf("complete: %+v %v", got, err)
	}
	list, err := r.ListSessions(ctx, owner)
	if err != nil || len(list) != 1 || list[0].ID != s.ID {
		t.Fatalf("list: %+v %v", list, err)
	}
	if list, err := r.ListSessions(ctx, other); err != nil || len(list) != 0 {
		t.Fatalf("foreign list: %+v %v", list, err)
	}
	if _, err := r.CreateSession(ctx, uuid.New(), "stress_relief", prompts); err == nil {
		t.Fatal("missing user accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.CreateSession(canceled, owner, "stress_relief", prompts); !errors.Is(err, context.Canceled) {
		t.Fatalf("create cancel: %v", err)
	}
	if _, err := r.GetSession(canceled, s.ID, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("get cancel: %v", err)
	}
	if _, err := r.ListSessions(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancel: %v", err)
	}
	if err := r.AdvanceStep(canceled, s.ID, 0, entry, 1, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("advance cancel: %v", err)
	}
}

func TestIntegration_ConversationPersistenceAndTurnClosure(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	entry := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/conversation','completed')`, entry, owner); err != nil {
		t.Fatal(err)
	}
	r := NewConversationRepository(db)
	c, err := r.GetOrCreate(ctx, entry, owner)
	if err != nil || c == nil || c.EntryID != entry || c.UserID != owner || c.TurnCount != 0 || c.IsClosed {
		t.Fatalf("create: %+v %v", c, err)
	}
	again, err := r.GetOrCreate(ctx, entry, owner)
	if err != nil || again.ID != c.ID {
		t.Fatalf("retry: %+v %v", again, err)
	}
	if got, err := r.GetByID(ctx, c.ID, other); err != nil || got != nil {
		t.Fatalf("foreign: %+v %v", got, err)
	}
	if msgs, err := r.ListMessages(ctx, c.ID); err != nil || len(msgs) != 0 {
		t.Fatalf("empty: %+v %v", msgs, err)
	}
	ids := []uuid.UUID{}
	for i := 1; i <= models.MaxConversationTurns; i++ {
		got, msg, err := r.AddMessage(ctx, c.ID, "user", "QA user message")
		if err != nil || got == nil || msg == nil || got.TurnCount != i || got.IsClosed != (i == models.MaxConversationTurns) || msg.Content != "QA user message" || msg.Role != "user" {
			t.Fatalf("user turn %d: %+v %+v %v", i, got, msg, err)
		}
		ids = append(ids, msg.ID)
		got, msg, err = r.AddMessage(ctx, c.ID, "assistant", "QA reply")
		if err != nil || got.TurnCount != i || msg.Role != "assistant" || got.IsClosed != (i == models.MaxConversationTurns) {
			t.Fatalf("reply %d: %+v %+v %v", i, got, msg, err)
		}
		ids = append(ids, msg.ID)
	}
	msgs, err := r.ListMessages(ctx, c.ID)
	if err != nil || len(msgs) != 6 {
		t.Fatalf("messages: %+v %v", msgs, err)
	}
	for i, msg := range msgs {
		if msg.ID != ids[i] || (i > 0 && msg.CreatedAt.Before(msgs[i-1].CreatedAt)) {
			t.Fatalf("order: %+v", msgs)
		}
	}
	if _, _, err := r.AddMessage(ctx, c.ID, "invalid-role", "bad"); err == nil {
		t.Fatal("invalid role accepted")
	}
	loaded, err := r.GetByID(ctx, c.ID, owner)
	if err != nil || loaded.TurnCount != 3 || !loaded.IsClosed {
		t.Fatalf("invalid insert changed state: %+v %v", loaded, err)
	}
	if msgs, err := r.ListMessages(ctx, c.ID); err != nil || len(msgs) != 6 {
		t.Fatalf("invalid insert leaked: %+v %v", msgs, err)
	}
	if _, _, err := r.AddMessage(ctx, uuid.New(), "user", "missing"); err == nil {
		t.Fatal("missing conversation accepted")
	}
	if _, err := r.GetOrCreate(ctx, uuid.New(), owner); err == nil {
		t.Fatal("missing entry accepted")
	}
	if got, err := r.GetByID(ctx, uuid.New(), owner); err != nil || got != nil {
		t.Fatalf("missing: %+v %v", got, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.GetOrCreate(canceled, entry, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("create cancel: %v", err)
	}
	if _, err := r.GetByID(canceled, c.ID, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("get cancel: %v", err)
	}
	if _, err := r.ListMessages(canceled, c.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancel: %v", err)
	}
	if _, _, err := r.AddMessage(canceled, c.ID, "user", "cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("add cancel: %v", err)
	}
}
