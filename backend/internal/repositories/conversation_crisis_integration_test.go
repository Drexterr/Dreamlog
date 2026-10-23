//go:build integration

package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// N12: a conversation closed by crisis screening stays flagged, so reopening
// it (GetOrCreate / GetByID) still tells the client to show crisis resources.
func TestIntegration_Conversation_CrisisFlagPersists(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	owner := regressionUser(t, db)
	entry := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/crisis-followup/' || gen_random_uuid()::text,'completed')`, entry, owner); err != nil {
		t.Fatal(err)
	}
	r := NewConversationRepository(db)
	conv, err := r.GetOrCreate(ctx, entry, owner)
	if err != nil || conv.IsCrisis {
		t.Fatalf("a new conversation is not a crisis: %+v %v", conv, err)
	}
	closed, err := r.Close(ctx, conv.ID)
	if err != nil || !closed.IsClosed || !closed.IsCrisis {
		t.Fatalf("crisis close: %+v %v", closed, err)
	}
	reopened, err := r.GetOrCreate(ctx, entry, owner)
	if err != nil || !reopened.IsCrisis || !reopened.IsClosed {
		t.Fatalf("reopened conversation lost its crisis flag: %+v %v", reopened, err)
	}
	byID, err := r.GetByID(ctx, conv.ID, owner)
	if err != nil || byID == nil || !byID.IsCrisis {
		t.Fatalf("GetByID lost the crisis flag: %+v %v", byID, err)
	}
}
