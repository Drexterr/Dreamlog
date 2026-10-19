//go:build integration

package repositories

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

func TestIntegration_ShareOwnershipExpiryAndRevocation(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	repo := NewShareRepository(db)
	makeLink := func(user uuid.UUID, expiry time.Time) *models.ShareLink {
		t.Helper()
		link, err := repo.Create(ctx, models.CreateShareLinkInput{UserID: user, Token: uuid.NewString(), PasscodeHash: "test-hash-fixture", ExpiresAt: expiry})
		if err != nil {
			t.Fatal(err)
		}
		return link
	}
	active := makeLink(owner, time.Now().Add(72*time.Hour))
	expired := makeLink(owner, time.Now().Add(-time.Second))
	makeLink(other, time.Now().Add(72*time.Hour))
	links, err := repo.ListByUser(ctx, owner)
	if err != nil || len(links) != 1 || links[0].ID != active.ID {
		t.Fatalf("owner list: %+v %v", links, err)
	}
	if got, err := repo.GetByToken(ctx, expired.Token); err != nil || got != nil {
		t.Fatalf("expired link accessible: %+v %v", got, err)
	}
	if got, err := repo.GetByToken(ctx, "missing-fixture"); err != nil || got != nil {
		t.Fatalf("missing link: %+v %v", got, err)
	}
	if err := repo.Revoke(ctx, active.ID, other); err == nil {
		t.Fatal("foreign user revoked owner's link")
	}
	if got, err := repo.GetByToken(ctx, active.Token); err != nil || got == nil {
		t.Fatalf("foreign revoke changed link: %+v %v", got, err)
	}
	if err := repo.Revoke(ctx, uuid.New(), owner); err == nil {
		t.Fatal("missing link revoke claimed success")
	}
	if err := repo.Revoke(ctx, active.ID, owner); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetByToken(ctx, active.Token); err != nil || got != nil {
		t.Fatalf("revoked link accessible: %+v %v", got, err)
	}
	links, err = repo.ListByUser(ctx, owner)
	if err != nil || len(links) != 0 {
		t.Fatalf("revoked links listed: %+v %v", links, err)
	}
	count, err := repo.CountMonthlyByUser(ctx, owner)
	if err != nil || count != 2 {
		t.Fatalf("revocation/expiry must not erase quota usage: %d %v", count, err)
	}
}

func TestIntegration_SharePasscodeLockoutConcurrentAndReset(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	repo := NewShareRepository(db)
	link, err := repo.Create(ctx, models.CreateShareLinkInput{UserID: regressionUser(t, db), Token: uuid.NewString(), PasscodeHash: "test-hash-fixture", ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan *time.Time, 5)
	failures := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locked, err := repo.RecordFailedPasscode(ctx, link.ID, 5, 15*time.Minute)
			results <- locked
			failures <- err
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	locks := 0
	for until := range results {
		if until != nil {
			locks++
			if !until.After(time.Now().Add(14 * time.Minute)) {
				t.Fatalf("short lock window: %v", until)
			}
		}
	}
	if locks != 1 {
		t.Fatalf("expected one threshold crossing, got %d", locks)
	}
	got, err := repo.GetByToken(ctx, link.Token)
	if err != nil || got.LockedUntil == nil || got.FailedAttempts != 0 {
		t.Fatalf("lock state: %+v %v", got, err)
	}
	if err := repo.ResetPasscodeAttempts(ctx, link.ID); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetByToken(ctx, link.Token)
	if err != nil || got.LockedUntil != nil || got.FailedAttempts != 0 {
		t.Fatalf("reset state: %+v %v", got, err)
	}
	until, err := repo.RecordFailedPasscode(ctx, link.ID, 5, 15*time.Minute)
	if err != nil || until != nil {
		t.Fatalf("first attempt after reset should not lock: %v %v", until, err)
	}
	if _, err := repo.RecordFailedPasscode(ctx, uuid.New(), 5, time.Minute); err == nil {
		t.Fatal("missing link accepted failure counter update")
	}
}

func TestIntegration_ShareViewFiltersPrivateAndIneligibleRecords(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	entries, analyses, shares := NewEntryRepository(db), NewAnalysisRepository(db), NewShareRepository(db)
	fixtures := []struct {
		name              string
		user              uuid.UUID
		age               time.Duration
		crisis, completed bool
	}{
		{"allowed-summary", owner, 0, false, true},
		{"foreign-summary", other, 0, false, true},
		{"old-summary", owner, 31 * 24 * time.Hour, false, true},
		{"crisis-summary", owner, 0, true, true},
		{"pending-summary", owner, 0, false, false},
	}
	for _, fixture := range fixtures {
		e, err := entries.Create(ctx, fixture.user, "private-audio-fixture", 100, 3, models.EntryModeProcessing)
		if err != nil {
			t.Fatal(err)
		}
		if fixture.completed {
			if err := entries.SetCompleted(ctx, e.ID, "raw-transcript-private-marker", "english"); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.Exec(ctx, "UPDATE entries SET created_at=$2 WHERE id=$1", e.ID, time.Now().Add(-fixture.age)); err != nil {
			t.Fatal(err)
		}
		_, err = analyses.Upsert(ctx, e.ID, &models.EntryAnalysis{MoodScore: 70, Summary: fixture.name, Reflection: "private-reflection-marker", Topics: []string{"work"}, KeyQuotes: []string{"private-quote-marker"}, IsCrisis: fixture.crisis, EmotionalTone: []models.EmotionalTone{{Emotion: "hope", Intensity: 0.8}, {Emotion: "weak-emotion", Intensity: 0.2}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	view, err := shares.ShareView(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Summaries) != 1 || view.Summaries[0].Summary != "allowed-summary" || len(view.MoodArc) != 1 || view.AvgMood == nil || *view.AvgMood != 70 {
		t.Fatalf("unexpected filtered view: %+v", view)
	}
	if len(view.TopEmotions) != 1 || view.TopEmotions[0] != "hope" {
		t.Fatalf("emotion filtering: %+v", view.TopEmotions)
	}
	if view.UserName != "QA" || view.Period != "30d" {
		t.Fatalf("shared display name/period: %+v", view)
	}
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"foreign-summary", "old-summary", "crisis-summary", "pending-summary", "raw-transcript-private-marker", "private-reflection-marker", "private-quote-marker", "private-audio-fixture", "weak-emotion"} {
		if strings.Contains(string(payload), marker) {
			t.Fatalf("private/ineligible marker leaked: %s", marker)
		}
	}
	if _, err := shares.ShareView(ctx, uuid.New()); err == nil {
		t.Fatal("missing user generated a public view")
	}
}

func TestIntegration_ShareViewEmptyAccount(t *testing.T) {
	db := regressionDB(t)
	view, err := NewShareRepository(db).ShareView(context.Background(), regressionUser(t, db))
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Summaries) != 0 || len(view.MoodArc) != 0 || len(view.TopEmotions) != 0 || view.AvgMood != nil {
		t.Fatalf("empty account fabricated data: %+v", view)
	}
}
