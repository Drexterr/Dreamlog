//go:build integration

package repositories

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// N4: a turn is claimed once per request_id - concurrent claims converge on
// one user message, and a reply is stored at most once per turn.
func TestIntegration_TherapyTurn_IdempotentClaimAndReply(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	r := NewTherapyRepository(db)
	s, err := r.Create(ctx, regressionUser(t, db), models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
	therapistDataOK(t, err)
	reqID := uuid.New()

	const n = 8
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, n)
	created := make([]bool, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, c, err := r.ClaimTurn(ctx, s.ID, reqID, "same turn", "text", false)
			errs[i], created[i] = err, c
			if m != nil {
				ids[i] = m.ID
			}
		}(i)
	}
	wg.Wait()
	winners := 0
	for i := 0; i < n; i++ {
		therapistDataOK(t, errs[i])
		if ids[i] != ids[0] {
			t.Fatalf("concurrent claims must converge on one message: %v vs %v", ids[i], ids[0])
		}
		if created[i] {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("exactly one claim may create the turn, got %d", winners)
	}

	user, reply, err := r.FindTurn(ctx, s.ID, reqID)
	therapistDataOK(t, err)
	if user == nil || user.ID != ids[0] || reply != nil {
		t.Fatalf("claimed turn without reply: %+v %+v", user, reply)
	}

	first, turns, err := r.AddReply(ctx, s.ID, user.ID, "reply one", "text", true, true)
	therapistDataOK(t, err)
	if turns != 1 {
		t.Fatalf("reply must count the turn: %d", turns)
	}
	again, turns, err := r.AddReply(ctx, s.ID, user.ID, "reply two", "text", true, true)
	therapistDataOK(t, err)
	if again.ID != first.ID || again.Content != "reply one" || turns != 1 {
		t.Fatalf("a turn is answered once: %+v turns=%d", again, turns)
	}
	_, reply, err = r.FindTurn(ctx, s.ID, reqID)
	therapistDataOK(t, err)
	if reply == nil || reply.ID != first.ID {
		t.Fatalf("FindTurn must return the stored reply: %+v", reply)
	}
	// An answered turn is never released.
	therapistDataOK(t, r.ReleaseTurn(ctx, user.ID))
	if u, _, _ := r.FindTurn(ctx, s.ID, reqID); u == nil {
		t.Fatal("ReleaseTurn must not remove an answered turn")
	}

	// An unanswered turn is released, and the request id can be claimed again.
	other := uuid.New()
	m, _, err := r.ClaimTurn(ctx, s.ID, other, "will fail", "text", false)
	therapistDataOK(t, err)
	therapistDataOK(t, r.ReleaseTurn(ctx, m.ID))
	if u, _, _ := r.FindTurn(ctx, s.ID, other); u != nil {
		t.Fatal("a released turn must be gone")
	}
	if _, c, err := r.ClaimTurn(ctx, s.ID, other, "retry", "text", false); err != nil || !c {
		t.Fatalf("released request id must be claimable again: %v %v", c, err)
	}
}

// N3: after a crisis hard stop no reply is stored, a normal claim is refused,
// and neither End nor expiry can overwrite crisis_detected.
func TestIntegration_TherapyTurn_HardStopIsFinal(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	r := NewTherapyRepository(db)
	owner := regressionUser(t, db)
	s, err := r.Create(ctx, owner, models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
	therapistDataOK(t, err)

	pending, _, err := r.ClaimTurn(ctx, s.ID, uuid.New(), "ordinary message", "text", false)
	therapistDataOK(t, err)
	therapistDataOK(t, r.MarkCrisis(ctx, s.ID, time.Now(), 30))

	if _, _, err := r.AddReply(ctx, s.ID, pending.ID, "late reply", "text", true, true); !errors.Is(err, models.ErrTherapySessionNotActive) {
		t.Fatalf("reply after hard stop: want ErrTherapySessionNotActive, got %v", err)
	}
	if _, _, err := r.ClaimTurn(ctx, s.ID, uuid.New(), "next", "text", false); !errors.Is(err, models.ErrTherapySessionNotActive) {
		t.Fatalf("claim after hard stop: want ErrTherapySessionNotActive, got %v", err)
	}
	// Crisis turns are still recorded, with their resources reply.
	crisisMsg, _, err := r.ClaimTurn(ctx, s.ID, uuid.New(), "crisis", "text", true)
	therapistDataOK(t, err)
	if _, _, err := r.AddReply(ctx, s.ID, crisisMsg.ID, "resources", "system", false, false); err != nil {
		t.Fatalf("crisis resources reply: %v", err)
	}

	now := time.Now()
	dur := 60
	for _, st := range []models.TherapySessionStatus{models.TherapyStatusCompleted, models.TherapyStatusExpired} {
		ok, err := r.EndActive(ctx, s.ID, st, &now, &dur)
		therapistDataOK(t, err)
		if ok {
			t.Fatalf("%s must not overwrite crisis_detected", st)
		}
	}
	therapistDataOK(t, r.SetSessionAnalysis(ctx, s.ID, &models.TherapySessionAnalysis{MoodScore: 50, SessionNarrative: "must not land"}))
	got, err := r.GetByID(ctx, s.ID, owner)
	therapistDataOK(t, err)
	if got.Status != models.TherapyStatusCrisisDetected || got.PostSessionSummary != nil || got.SessionMoodScore != nil || got.TurnCount != 0 {
		t.Fatalf("crisis session changed: %+v", got)
	}
	msgs, err := r.ListMessages(ctx, s.ID)
	therapistDataOK(t, err)
	for _, m := range msgs {
		if m.Content == "late reply" {
			t.Fatal("a reply generated before the hard stop must not be stored after it")
		}
	}
}

// N3: concurrent End and crisis hard stop always leave crisis_detected, and a
// summary written by the losing End never lands.
func TestIntegration_TherapyTurn_ConcurrentEndAndCrisis(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	r := NewTherapyRepository(db)
	owner := regressionUser(t, db)
	for i := 0; i < 10; i++ {
		s, err := r.Create(ctx, owner, models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
		therapistDataOK(t, err)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			now, dur := time.Now(), 10
			if ok, err := r.EndActive(ctx, s.ID, models.TherapyStatusCompleted, &now, &dur); err == nil && ok {
				_ = r.SetSessionAnalysis(ctx, s.ID, &models.TherapySessionAnalysis{MoodScore: 60, SessionNarrative: "summary"})
			}
		}()
		go func() {
			defer wg.Done()
			_ = r.MarkCrisis(ctx, s.ID, time.Now(), 10)
		}()
		wg.Wait()
		got, err := r.GetByID(ctx, s.ID, owner)
		therapistDataOK(t, err)
		if got.Status != models.TherapyStatusCrisisDetected {
			t.Fatalf("run %d: crisis must win, got %s", i, got.Status)
		}
		if got.PostSessionSummary != nil {
			t.Fatalf("run %d: crisis session kept a summary %q", i, *got.PostSessionSummary)
		}
		summaries, err := r.PastCompletedSummaries(ctx, owner, 10)
		therapistDataOK(t, err)
		if len(summaries) != 0 {
			t.Fatalf("run %d: crisis transcript entered session memory: %v", i, summaries)
		}
	}
}
