//go:build integration

package repositories

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
)

const testMonth = 30 * 24 * time.Hour

// N5: concurrent same-user replays of one purchase grant it exactly once, so
// one payment can never extend a pass twice.
func TestIntegration_Payments_ConcurrentReplayGrantsOnce(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewPaymentRepository(db)
	if _, err := r.Record(ctx, user, "GPA.n5", models.PlanPlus, "google", "com.talktoode.app.plus.monthly", false); err != nil {
		t.Fatal(err)
	}

	const n = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	grants := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, granted, err := r.GrantPlan(ctx, "GPA.n5", user, models.PlanPlus, testMonth)
			if err != nil || u == nil || u.Plan != models.PlanPlus {
				t.Errorf("grant: %+v %v", u, err)
				return
			}
			if granted {
				mu.Lock()
				grants++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if grants != 1 {
		t.Fatalf("one purchase granted %d times, want 1", grants)
	}

	u, err := NewUserRepository(db).GetByID(ctx, user)
	if err != nil || u.PlanExpiresAt == nil {
		t.Fatalf("user: %+v %v", u, err)
	}
	if got := time.Until(*u.PlanExpiresAt); got > testMonth+time.Hour || got < testMonth-time.Hour {
		t.Fatalf("expiry must be one pass (~30 days) out, got %v", got)
	}
}

// N5: two different purchases of the same plan granted concurrently both
// count - the expiry is computed under the user row lock, never from a stale read.
func TestIntegration_Payments_ConcurrentDistinctPurchasesBothExtend(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewPaymentRepository(db)
	for _, tx := range []string{"GPA.n5a", "GPA.n5b"} {
		if _, err := r.Record(ctx, user, tx, models.PlanPlus, "google", "com.talktoode.app.plus.monthly", false); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for _, tx := range []string{"GPA.n5a", "GPA.n5b"} {
		wg.Add(1)
		go func(tx string) {
			defer wg.Done()
			if _, granted, err := r.GrantPlan(ctx, tx, user, models.PlanPlus, testMonth); err != nil || !granted {
				t.Errorf("%s: granted=%v %v", tx, granted, err)
			}
		}(tx)
	}
	wg.Wait()
	u, err := NewUserRepository(db).GetByID(ctx, user)
	if err != nil || u.PlanExpiresAt == nil {
		t.Fatalf("user: %+v %v", u, err)
	}
	if got := time.Until(*u.PlanExpiresAt); got < 2*testMonth-time.Hour || got > 2*testMonth+time.Hour {
		t.Fatalf("two passes must stack to ~60 days, got %v", got)
	}
}

// N5: a transaction recorded for another user is never granted to the caller.
func TestIntegration_Payments_GrantRejectsOtherOwner(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewPaymentRepository(db)
	if _, err := r.Record(ctx, owner, "GPA.n5owner", models.PlanPro, "apple", "com.talktoode.app.pro.monthly", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.GrantPlan(ctx, "GPA.n5owner", other, models.PlanPro, testMonth); !errors.Is(err, ErrPaymentNotOwned) {
		t.Fatalf("want ErrPaymentNotOwned, got %v", err)
	}
	u, err := NewUserRepository(db).GetByID(ctx, other)
	if err != nil || u.Plan != models.PlanFree {
		t.Fatalf("other user must stay free: %+v %v", u, err)
	}
}
