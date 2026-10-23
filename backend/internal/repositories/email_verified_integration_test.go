//go:build integration

package repositories

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// N6: migration 000044 marks existing Supabase accounts verified and leaves
// local signups unverified.
func TestIntegration_Migration044_BackfillsEmailVerified(t *testing.T) {
	db := regressionDBUpTo(t, "000043")
	ctx := context.Background()
	supa, local := uuid.New(), uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO users(id,supabase_id,email,name) VALUES($1,$2,$3,'S'),($4,$5,$6,'L')`,
		supa, "supabase-"+supa.String(), supa.String()+"@example.test",
		local, "local-"+local.String(), local.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	sql, err := os.ReadFile("../../migrations/000044_email_verified.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, string(sql)); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[uuid.UUID]bool{supa: true, local: false} {
		var got bool
		if err := db.QueryRow(ctx, `SELECT email_verified FROM users WHERE id=$1`, id).Scan(&got); err != nil || got != want {
			t.Fatalf("%s: email_verified=%v (%v), want %v", id, got, err, want)
		}
	}
}

// N6: local signups are unverified; a verified sign-in takes the address over
// from an unverified squatter (which can then no longer log in), but never
// from another verified account.
func TestIntegration_Upsert_VerifiedSignInTakesOverUnverifiedEmail(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	r := NewUserRepository(db)
	email := "victim-" + uuid.NewString()[:8] + "@example.test"

	squatter, err := r.CreateLocal(ctx, email, "Squatter", "fixture-hash")
	if err != nil || squatter.EmailVerified {
		t.Fatalf("local signup must be unverified: %+v %v", squatter, err)
	}

	victim, err := r.Upsert(ctx, "supabase-"+uuid.NewString(), email, "Victim", true)
	if err != nil {
		t.Fatalf("verified sign-in must not fail on a squatted email: %v", err)
	}
	if victim.ID == squatter.ID || victim.Email != email || !victim.EmailVerified {
		t.Fatalf("victim must own the verified email: %+v", victim)
	}
	released, err := r.GetByID(ctx, squatter.ID)
	if err != nil || released == nil || !strings.HasPrefix(released.Email, "released+") {
		t.Fatalf("squatter must lose the address: %+v %v", released, err)
	}
	if hash, err := r.GetPasswordHash(ctx, email); err != nil || hash != "" {
		t.Fatalf("the squatter's password must not log in as the email: %q %v", hash, err)
	}

	// A verified account is never displaced.
	if _, err := r.Upsert(ctx, "supabase-"+uuid.NewString(), email, "Other", true); err == nil {
		t.Fatal("a second verified sign-in must not take a verified email")
	}
	if got, err := r.GetByEmail(ctx, email); err != nil || got.ID != victim.ID {
		t.Fatalf("verified owner changed: %+v %v", got, err)
	}

	// An existing unverified row becomes verified when its provider says so.
	local, err := r.Upsert(ctx, "supabase-"+uuid.NewString(), "later-"+email, "Later", false)
	if err != nil || local.EmailVerified {
		t.Fatalf("unverified upsert: %+v %v", local, err)
	}
	again, err := r.Upsert(ctx, local.SupabaseID, local.Email, "Later", true)
	if err != nil || again.ID != local.ID || !again.EmailVerified {
		t.Fatalf("verification must be recorded: %+v %v", again, err)
	}
}
