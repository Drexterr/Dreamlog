//go:build integration

package repositories

import (
	"context"
	"sync"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// DLQ requeue resets only FAILED entries, gives them a fresh retry budget,
// and concurrent resets of one entry succeed exactly once.
func TestIntegration_Entries_ResetForRequeue(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewEntryRepository(db)
	entry, err := r.Create(ctx, user, "audio/"+user.String()+"/"+uuid.NewString()+".m4a", 100, 3, models.EntryModeProcessing)
	if err != nil {
		t.Fatal(err)
	}

	if ok, err := r.ResetForRequeue(ctx, entry.ID); err != nil || ok {
		t.Fatalf("a pending entry must not be reset: ok=%v %v", ok, err)
	}
	for i := 0; i < 3; i++ {
		if err := r.SetFailed(ctx, entry.ID, "max retries exceeded: boom"); err != nil {
			t.Fatal(err)
		}
	}

	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	resets := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := r.ResetForRequeue(ctx, entry.ID)
			if err != nil {
				t.Error(err)
			}
			if ok {
				mu.Lock()
				resets++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if resets != 1 {
		t.Fatalf("concurrent resets succeeded %d times, want 1", resets)
	}
	got, err := r.GetByIDInternal(ctx, entry.ID)
	if err != nil || got.Status != models.EntryStatusPending || got.RetryCount != 0 || got.ErrorMsg != nil {
		t.Fatalf("after reset: %+v %v", got, err)
	}

	if _, err := db.Exec(ctx, `UPDATE entries SET status = 'completed' WHERE id = $1`, entry.ID); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.ResetForRequeue(ctx, entry.ID); err != nil || ok {
		t.Fatalf("a completed entry must never be reset: ok=%v %v", ok, err)
	}
	if ok, err := r.ResetForRequeue(ctx, uuid.New()); err != nil || ok {
		t.Fatalf("unknown entry: ok=%v %v", ok, err)
	}
}

func TestIntegration_NoteSessions_ResetSessionForRequeue(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	therapist, client, session := uuid.New(), uuid.New(), uuid.New()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO therapists(id,user_id,name,email) VALUES($1,$2,'T',$3)`, []any{therapist, user, therapist.String() + "@example.test"}},
		{`INSERT INTO external_clients(id,therapist_id,name_enc) VALUES($1,$2,'\x00')`, []any{client, therapist}},
		{`INSERT INTO client_sessions(id,therapist_id,external_client_id,status,image_key,error_msg,retry_count) VALUES($1,$2,$3,'failed','notes/x/n.jpg','boom',3)`, []any{session, therapist, client}},
	} {
		if _, err := db.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	r := NewTherapistNotesRepository(db)
	if ok, err := r.ResetSessionForRequeue(ctx, session); err != nil || !ok {
		t.Fatalf("failed session must reset: ok=%v %v", ok, err)
	}
	row, err := r.GetSessionByID(ctx, session)
	if err != nil || row.Status != models.ClientSessionPending || row.ImageKey == nil {
		t.Fatalf("after reset: %+v %v", row, err)
	}
	var retries int
	var errMsg *string
	if err := db.QueryRow(ctx, `SELECT retry_count, error_msg FROM client_sessions WHERE id = $1`, session).Scan(&retries, &errMsg); err != nil || retries != 0 || errMsg != nil {
		t.Fatalf("retry budget not reset: %d %v %v", retries, errMsg, err)
	}
	if ok, err := r.ResetSessionForRequeue(ctx, session); err != nil || ok {
		t.Fatalf("a pending session must not reset again: ok=%v %v", ok, err)
	}
}
