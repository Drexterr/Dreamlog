//go:build integration

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// N9: an entry still 'pending' was never attempted. However long it waits in
// a backlog it is never failed and never charged a retry; only a 'processing'
// row whose worker was lost counts.
func TestIntegration_ReclaimStuck_PendingNeverCountsAsAttempt(t *testing.T) {
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
	waiting, lost := mk(), mk()
	exec := func(sql string, args ...any) {
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	backdate := func() {
		exec(`ALTER TABLE entries DISABLE TRIGGER trg_entries_updated_at`)
		exec(`UPDATE entries SET status='pending', updated_at=NOW()-INTERVAL '2 hours', retry_count=3 WHERE id=$1`, waiting)
		exec(`UPDATE entries SET status='processing', updated_at=NOW()-INTERVAL '40 minutes', retry_count=1 WHERE id=$1`, lost)
		exec(`ALTER TABLE entries ENABLE TRIGGER trg_entries_updated_at`)
	}
	state := func(id uuid.UUID) (string, int) {
		var s string
		var n int
		if err := db.QueryRow(ctx, `SELECT status, retry_count FROM entries WHERE id=$1`, id).Scan(&s, &n); err != nil {
			t.Fatal(err)
		}
		return s, n
	}

	// Backlogged queue: pending rows are left alone, even at retry_count 3.
	backdate()
	stuck, err := r.ReclaimStuck(ctx, 15*time.Minute, 30*time.Minute, 3, 100, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(stuck) != 1 || stuck[0].ID != lost || stuck[0].RetryCount != 2 {
		t.Fatalf("only the lost processing row is reclaimed, charged one attempt: %+v", stuck)
	}
	if s, n := state(waiting); s != "pending" || n != 3 {
		t.Fatalf("waiting entry must stay pending and uncharged: %s/%d", s, n)
	}

	// Empty queue: the pending row is requeued, still without a charge.
	backdate()
	stuck, err = r.ReclaimStuck(ctx, 15*time.Minute, 30*time.Minute, 3, 100, true)
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]int{}
	for _, s := range stuck {
		got[s.ID] = s.RetryCount
	}
	if n, ok := got[waiting]; !ok || n != 3 {
		t.Fatalf("pending row must be requeued with retry_count unchanged: %+v", stuck)
	}
	if s, n := state(waiting); s != "pending" || n != 3 {
		t.Fatalf("waiting entry: %s/%d", s, n)
	}
}
