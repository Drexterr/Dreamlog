//go:build integration

package repositories

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// N7: account deletion erases exactly the user's own prefixes: journal audio,
// therapy voice turns, the TTS of their sessions and their note photos.
func TestIntegration_Users_StoragePrefixes(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user, other := regressionUser(t, db), regressionUser(t, db)
	therapy := NewTherapyRepository(db)
	mine, err := therapy.Create(ctx, user, models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
	therapistDataOK(t, err)
	if _, err := therapy.Create(ctx, other, models.PersonaComforting, models.TherapyContextSnapshot{}, 0); err != nil {
		t.Fatal(err)
	}
	therapist := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO therapists(id,user_id,name,email) VALUES($1,$2,'T',$3)`, therapist, user, therapist.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}

	got, err := NewUserRepository(db).StoragePrefixes(ctx, user)
	therapistDataOK(t, err)
	want := []string{
		"audio/" + user.String() + "/",
		"notes/" + therapist.String() + "/",
		"therapy/" + user.String() + "/",
		"tts/" + mine.ID.String() + "/",
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("prefixes:\n got  %v\n want %v", got, want)
	}
}
