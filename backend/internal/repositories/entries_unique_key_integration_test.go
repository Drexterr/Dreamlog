//go:build integration

package repositories

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// N10: concurrent registrations of one upload converge on a single entry,
// and exactly one caller is told it created it (so one job is queued).
func TestIntegration_Entries_CreateOnceConcurrent(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewEntryRepository(db)
	key := "audio/" + user.String() + "/" + uuid.NewString() + ".aac"

	const n = 8
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, n)
	created := make([]bool, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			e, c, err := r.CreateOnce(ctx, user, key, 10, 1, models.EntryModeProcessing)
			if err != nil || e == nil {
				t.Errorf("CreateOnce: %+v %v", e, err)
				return
			}
			ids[i], created[i] = e.ID, c
		}(i)
	}
	wg.Wait()
	creators := 0
	for i := range ids {
		if ids[i] != ids[0] {
			t.Fatalf("registrations diverged: %v vs %v", ids[i], ids[0])
		}
		if created[i] {
			creators++
		}
	}
	if creators != 1 {
		t.Fatalf("exactly one registration may create the entry, got %d", creators)
	}
	var rows int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM entries WHERE user_id=$1 AND audio_key=$2`, user, key).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("rows for one upload: %d %v", rows, err)
	}
	// Another user may of course use the same key text in their own namespace.
	other := regressionUser(t, db)
	if _, c, err := r.CreateOnce(ctx, other, key, 10, 1, models.EntryModeProcessing); err != nil || !c {
		t.Fatalf("other user: created=%v %v", c, err)
	}
}

// N10: migration 000045 folds existing duplicates (keeping the completed row)
// before adding the unique index.
func TestIntegration_Migration045_FoldsDuplicateEntries(t *testing.T) {
	db := regressionDBUpTo(t, "000044")
	ctx := context.Background()
	user := uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,supabase_id,email,name) VALUES($1,$2,$3,'QA')`, user, user.String(), user.String()+"@example.test")
	failed, completed, single := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO entries(id,user_id,audio_key,status,created_at) VALUES
		($1,$4,'audio/dup','failed',NOW()-INTERVAL '2 minutes'),
		($2,$4,'audio/dup','completed',NOW()-INTERVAL '1 minute'),
		($3,$4,'audio/single','completed',NOW())`, failed, completed, single, user)

	sql, err := os.ReadFile("../../migrations/000045_entries_unique_audio_key.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(sql))

	var ids []uuid.UUID
	rows, err := db.Query(ctx, `SELECT id FROM entries WHERE user_id=$1 ORDER BY created_at`, user)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) != 2 || ids[0] != completed || ids[1] != single {
		t.Fatalf("want the completed duplicate and the single entry kept, got %v", ids)
	}
	if _, err := db.Exec(ctx, `INSERT INTO entries(user_id,audio_key,status) VALUES($1,'audio/single','pending')`, user); err == nil {
		t.Fatal("unique index must reject a second entry for the same upload")
	}
}
