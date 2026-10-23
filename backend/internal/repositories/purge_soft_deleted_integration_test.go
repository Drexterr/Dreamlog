//go:build integration

package repositories

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
)

// N2: accounts soft-deleted before hard delete must be purged by migration
// 000042 - with all their data - and stop being served to linked therapists.
func TestIntegration_Migration042_PurgesLegacySoftDeletedUsers(t *testing.T) {
	db := regressionDBUpTo(t, "000041")
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}

	deleted, active, therapistUser, therapist := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, u := range []uuid.UUID{deleted, active, therapistUser} {
		exec(`INSERT INTO users(id,supabase_id,email,name) VALUES($1,$2,$3,'QA')`, u, u.String(), u.String()+"@example.test")
	}
	exec(`UPDATE users SET is_deleted = true, deleted_at = NOW() WHERE id = $1`, deleted)
	exec(`INSERT INTO entries(user_id,audio_key,status,transcript) VALUES($1,'audio/x/' || gen_random_uuid()::text,'completed','private')`, deleted)
	exec(`INSERT INTO entries(user_id,audio_key,status,transcript) VALUES($1,'audio/y/' || gen_random_uuid()::text,'completed','kept')`, active)
	exec(`INSERT INTO therapists(id,user_id,name,email) VALUES($1,$2,'T','t@example.test')`, therapist, therapistUser)
	exec(`INSERT INTO client_therapist_links(therapist_id,client_id,status) VALUES($1,$2,'active'),($1,$3,'active')`, therapist, deleted, active)

	sql, err := os.ReadFile("../../migrations/000042_purge_soft_deleted_users.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(sql))

	var n int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM users WHERE id = $1`, deleted).Scan(&n); err != nil || n != 0 {
		t.Fatalf("soft-deleted user must be purged: %d %v", n, err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM entries WHERE user_id = $1`, deleted).Scan(&n); err != nil || n != 0 {
		t.Fatalf("their entries must cascade: %d %v", n, err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM users WHERE id = $1`, active).Scan(&n); err != nil || n != 1 {
		t.Fatalf("active users must be untouched: %d %v", n, err)
	}
	var cols int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name='users' AND column_name IN ('is_deleted','deleted_at') AND table_schema = current_schema()`).Scan(&cols); err != nil || cols != 0 {
		t.Fatalf("soft-delete columns must be dropped: %d %v", cols, err)
	}

	// Checked in SQL rather than through TherapistRepository: the schema here
	// stops at 000042, and repository queries track the latest schema.
	var linked []uuid.UUID
	rows, err := db.Query(ctx, `SELECT client_id FROM client_therapist_links WHERE therapist_id = $1 AND status = 'active'`, therapist)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		linked = append(linked, id)
	}
	rows.Close()
	if len(linked) != 1 || linked[0] != active {
		t.Fatalf("therapist must only see the active client, got %v", linked)
	}
}
