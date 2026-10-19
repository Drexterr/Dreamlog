//go:build integration

package repositories

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIntegration_RecentUsersSchedulerSelection(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	eligible, ineligible := regressionUser(t, db), regressionUser(t, db)
	since := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	if _, err := db.Exec(ctx, `INSERT INTO entries(user_id,audio_key,status,created_at) VALUES($1,'fixture/recent','completed',$3),($1,'fixture/retry','completed',$3),($2,'fixture/old','completed',$3-INTERVAL '1 day'),($2,'fixture/pending','pending',$3)`, eligible, ineligible, since); err != nil {
		t.Fatal(err)
	}
	r := NewUserRepository(db)
	users, err := r.ListWithRecentEntries(ctx, since)
	if err != nil || len(users) != 1 || users[0].ID != eligible || users[0].Name != "QA" || users[0].Email == "" {
		t.Fatalf("eligible users once: %+v %v", users, err)
	}
	users, err = r.ListWithRecentEntries(ctx, since.Add(time.Second))
	if err != nil || len(users) != 0 {
		t.Fatalf("outside window: %+v %v", users, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.ListWithRecentEntries(canceled, since); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
