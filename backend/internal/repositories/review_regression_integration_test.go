//go:build integration

package repositories

// Regression tests for the 2026-10-19 code review (docs/CODE_REVIEW_2026-10-19.md,
// "Test gaps"): invariants that only hold under real Postgres concurrency or
// cascades, which unit fakes cannot prove.

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func reviewCompletedEntry(t *testing.T, db *pgxpool.Pool, user uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	entry, err := NewEntryRepository(db).Create(ctx, user, "audio/"+user.String()+"/"+uuid.NewString(), 10, 1, models.EntryModeProcessing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `UPDATE entries SET status='completed', transcript='t' WHERE id=$1`, entry.ID); err != nil {
		t.Fatal(err)
	}
	return entry.ID
}

// H9: with one turn left, many concurrent sends must claim exactly one turn.
func TestIntegration_Review_ConversationTurnCapUnderConcurrency(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	entry := reviewCompletedEntry(t, db, user)
	r := NewConversationRepository(db)
	conv, err := r.GetOrCreate(ctx, entry, user)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `UPDATE conversations SET turn_count = $2 WHERE id = $1`, conv.ID, models.MaxConversationTurns-1); err != nil {
		t.Fatal(err)
	}

	const senders = 10
	var wg sync.WaitGroup
	results := make(chan error, senders)
	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := r.AddMessage(ctx, conv.ID, "user", "parallel")
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok, rejected := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, models.ErrConversationTurnUnavailable):
			rejected++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || rejected != senders-1 {
		t.Fatalf("want exactly 1 claimed turn, got ok=%d rejected=%d", ok, rejected)
	}
	got, err := r.GetByID(ctx, conv.ID, user)
	if err != nil || got.TurnCount != models.MaxConversationTurns || !got.IsClosed {
		t.Fatalf("conversation after race: %+v %v", got, err)
	}
	var userMsgs int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND role='user'`, conv.ID).Scan(&userMsgs); err != nil || userMsgs != 1 {
		t.Fatalf("rejected sends must write nothing: %d %v", userMsgs, err)
	}
}

// H11: a store transaction is recorded once even under parallel redemption,
// same-user replays see their own row, and the grant marker is reported.
func TestIntegration_Review_PaymentReplayUnderConcurrency(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	other := regressionUser(t, db)
	r := NewPaymentRepository(db)

	const redeemers = 10
	var wg sync.WaitGroup
	results := make(chan RecordResult, redeemers)
	for i := 0; i < redeemers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := r.Record(ctx, user, "GPA.race", models.PlanPlus, "google", "com.talktoode.app.plus.monthly", false)
			if err != nil {
				t.Error(err)
			}
			results <- res
		}()
	}
	wg.Wait()
	close(results)
	inserted := 0
	for res := range results {
		if res.Inserted {
			inserted++
		} else if res.OwnerID != user || res.Granted {
			t.Errorf("replay must report owner=user, granted=false: %+v", res)
		}
	}
	if inserted != 1 {
		t.Fatalf("transaction recorded %d times, want 1", inserted)
	}

	if _, granted, err := r.GrantPlan(ctx, "GPA.race", user, models.PlanPlus, 30*24*time.Hour); err != nil || !granted {
		t.Fatalf("first grant: granted=%v %v", granted, err)
	}
	if res, err := r.Record(ctx, user, "GPA.race", models.PlanPlus, "google", "x", false); err != nil || res.Inserted || !res.Granted {
		t.Fatalf("granted replay: %+v %v", res, err)
	}
	if res, err := r.Record(ctx, other, "GPA.race", models.PlanPlus, "google", "x", false); err != nil || res.Inserted || res.OwnerID != user {
		t.Fatalf("other user's replay must see the original owner: %+v %v", res, err)
	}
	if _, err := r.Record(ctx, user, "GPA.sandbox", models.PlanPlus, "apple", "x", true); err != nil {
		t.Fatal(err)
	}
	var env string
	if err := db.QueryRow(ctx, `SELECT environment FROM payments WHERE transaction_id='GPA.sandbox'`).Scan(&env); err != nil || env != "sandbox" {
		t.Fatalf("sandbox purchase must be recorded as sandbox: %q %v", env, err)
	}
}

// H10: two concurrent crisis detections get distinct warning counts, so exactly
// one de-escalates and the other hard-stops.
func TestIntegration_Review_TherapyCrisisWarningIsAtomic(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewTherapyRepository(db)
	session, err := r.Create(ctx, user, models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	counts := make([]int, 2)
	for i := range counts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n, err := r.IncrementCrisisWarning(ctx, session.ID)
			if err != nil {
				t.Error(err)
			}
			counts[i] = n
		}(i)
	}
	wg.Wait()
	sort.Ints(counts)
	if counts[0] != 1 || counts[1] != 2 {
		t.Fatalf("concurrent increments must return 1 and 2, got %v", counts)
	}
}

// M12: concurrent self-joins never exceed the seat limit.
func TestIntegration_Review_SeatLimitUnderConcurrency(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	company := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO companies(id,name,slug,admin_email,seat_limit) VALUES($1,'Seats','seats','hr@example.test',3)`, company); err != nil {
		t.Fatal(err)
	}
	r := NewCompanyRepository(db)
	const joiners = 10
	users := make([]uuid.UUID, joiners)
	for i := range users {
		users[i] = regressionUser(t, db)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	joined := 0
	for _, u := range users {
		wg.Add(1)
		go func(u uuid.UUID) {
			defer wg.Done()
			ok, err := r.JoinCompany(ctx, company, u, 3)
			if err != nil {
				t.Error(err)
			}
			if ok {
				mu.Lock()
				joined++
				mu.Unlock()
			}
		}(u)
	}
	wg.Wait()
	total, err := r.TotalMembers(ctx, company)
	if err != nil || total != 3 || joined != 3 {
		t.Fatalf("seat limit 3: members=%d joined=%d err=%v", total, joined, err)
	}
}

// H7: stuck entries are requeued (or failed when out of retries); live ones
// are left alone, and a second pass does not reclaim the same rows.
func TestIntegration_Review_ReclaimStuckEntries(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewEntryRepository(db)
	mk := func() uuid.UUID {
		e, err := r.Create(ctx, user, "audio/"+user.String()+"/"+uuid.NewString(), 1, 1, models.EntryModeProcessing)
		if err != nil {
			t.Fatal(err)
		}
		return e.ID
	}
	stalePending, staleProcessing, liveProcessing, exhausted, freshPending := mk(), mk(), mk(), mk(), mk()

	// Backdate updated_at without the trigger bumping it back to NOW().
	exec := func(sql string, args ...any) {
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`ALTER TABLE entries DISABLE TRIGGER trg_entries_updated_at`)
	exec(`UPDATE entries SET status='pending',    updated_at=NOW()-INTERVAL '20 minutes' WHERE id=$1`, stalePending)
	exec(`UPDATE entries SET status='processing', updated_at=NOW()-INTERVAL '40 minutes' WHERE id=$1`, staleProcessing)
	exec(`UPDATE entries SET status='processing', updated_at=NOW()-INTERVAL '20 minutes' WHERE id=$1`, liveProcessing)
	exec(`UPDATE entries SET status='processing', updated_at=NOW()-INTERVAL '40 minutes', retry_count=3 WHERE id=$1`, exhausted)
	exec(`ALTER TABLE entries ENABLE TRIGGER trg_entries_updated_at`)

	stuck, err := r.ReclaimStuck(ctx, 15*time.Minute, 30*time.Minute, 3, 100, true)
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]bool{}
	for _, s := range stuck {
		got[s.ID] = true
	}
	if len(stuck) != 2 || !got[stalePending] || !got[staleProcessing] {
		t.Fatalf("want stale pending + stale processing reclaimed, got %+v", stuck)
	}
	status := func(id uuid.UUID) string {
		var s string
		if err := db.QueryRow(ctx, `SELECT status FROM entries WHERE id=$1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if status(exhausted) != "failed" || status(liveProcessing) != "processing" || status(freshPending) != "pending" || status(staleProcessing) != "pending" {
		t.Fatal("reclaim touched the wrong rows")
	}
	if again, err := r.ReclaimStuck(ctx, 15*time.Minute, 30*time.Minute, 3, 100, true); err != nil || len(again) != 0 {
		t.Fatalf("second pass must not reclaim the same rows: %+v %v", again, err)
	}
}

// M6: POST /entries idempotency relies on finding the user's entry by key.
func TestIntegration_Review_GetByAudioKey(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user, other := regressionUser(t, db), regressionUser(t, db)
	r := NewEntryRepository(db)
	key := "audio/" + user.String() + "/clip.m4a"
	created, err := r.Create(ctx, user, key, 1, 1, models.EntryModeProcessing)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetByAudioKey(ctx, user, key); err != nil || got == nil || got.ID != created.ID {
		t.Fatalf("own key: %+v %v", got, err)
	}
	if got, err := r.GetByAudioKey(ctx, other, key); err != nil || got != nil {
		t.Fatalf("another user's key must not match: %+v %v", got, err)
	}
}

// H1: account deletion removes all user content and detaches payments.
func TestIntegration_Review_HardDeleteCascades(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	entry := reviewCompletedEntry(t, db, user)
	if _, err := NewConversationRepository(db).GetOrCreate(ctx, entry, user); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTherapyRepository(db).Create(ctx, user, models.PersonaComforting, models.TherapyContextSnapshot{}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPaymentRepository(db).Record(ctx, user, "GPA.deleted", models.PlanPlus, "google", "x", false); err != nil {
		t.Fatal(err)
	}

	if err := NewUserRepository(db).Delete(ctx, user); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"entries", "conversations", "therapy_sessions"} {
		var n int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE user_id=$1`, user).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s must be deleted with the account: %d %v", table, n, err)
		}
	}
	var owner *uuid.UUID
	if err := db.QueryRow(ctx, `SELECT user_id FROM payments WHERE transaction_id='GPA.deleted'`).Scan(&owner); err != nil || owner != nil {
		t.Fatalf("payment must survive with user_id NULL (replay protection): %v %v", owner, err)
	}
}

// M15: a share link dies permanently after the lifetime cap of wrong passcodes.
func TestIntegration_Review_ShareLinkLifetimeFailureCap(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewShareRepository(db)
	link, err := r.Create(ctx, models.CreateShareLinkInput{UserID: user, Token: uuid.NewString(), PasscodeHash: "x", ExpiresAt: time.Now().Add(72 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < ShareMaxTotalFailures; i++ {
		if _, err := r.RecordFailedPasscode(ctx, link.ID, 5, time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	var expires time.Time
	if err := db.QueryRow(ctx, `SELECT expires_at FROM share_links WHERE id=$1`, link.ID).Scan(&expires); err != nil {
		t.Fatal(err)
	}
	if expires.After(time.Now()) {
		t.Fatalf("link must be expired after %d wrong passcodes, expires_at=%v", ShareMaxTotalFailures, expires)
	}
}
