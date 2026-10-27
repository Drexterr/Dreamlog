//go:build integration

package repositories

import (
	"context"
	"sync"
	"testing"
)

func TestIntegrationWaitlistConcurrentJoin(t *testing.T) {
	db := regressionDB(t)
	repo := NewWaitlistRepository(db)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := repo.Join(context.Background(), "waitlist@example.test"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var count int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM launch_waitlist WHERE email='waitlist@example.test' AND consent_version='launch-notification-v1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	var enabled bool
	if err := db.QueryRow(context.Background(), `SELECT relrowsecurity FROM pg_class WHERE oid='launch_waitlist'::regclass`).Scan(&enabled); err != nil || !enabled {
		t.Fatal("RLS disabled", err)
	}
}
