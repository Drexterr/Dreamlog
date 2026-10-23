//go:build integration

package repositories

import (
	"context"
	"sync"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

func TestIntegration_TypedEntryAtomicRetryAndOwnership(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewEntryRepository(db)
	input := models.CreateTextEntryInput{Content: "Complete fictional journal", RequestID: uuid.NewString(), Mode: models.EntryModeGratitude}
	const callers = 8
	var group sync.WaitGroup
	created := make(chan bool, callers)
	for i := 0; i < callers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			entry, fresh, err := r.CreateText(ctx, owner, &input)
			if err != nil {
				t.Error(err)
				return
			}
			if !input.Matches(entry) || entry.DurationSec != 0 || entry.AudioSizeBytes != 0 || entry.Status != models.EntryStatusPending {
				t.Error("incorrect typed journal")
			}
			created <- fresh
		}()
	}
	group.Wait()
	close(created)
	count := 0
	for fresh := range created {
		if fresh {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("created %d entries", count)
	}
	id := models.TextEntryID(owner, input.RequestID)
	if foreign, err := r.GetByID(ctx, id, other); err != nil || foreign != nil {
		t.Fatal("ownership failure")
	}
	entry, fresh, err := r.CreateText(ctx, other, &input)
	if err != nil || !fresh || entry.ID == id {
		t.Fatal("retry token not user-scoped")
	}
	changed := input
	changed.Content = "Changed content"
	old, fresh, err := r.CreateText(ctx, owner, &changed)
	if err != nil || fresh || changed.Matches(old) {
		t.Fatal("a conflicting retry replaced the original journal")
	}
}

// N11: the detected language is stored on a typed journal.
func TestIntegration_TextEntry_StoresLanguage(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner := regressionUser(t, db)
	r := NewEntryRepository(db)
	input := models.CreateTextEntryInput{Content: "आज मैं बहुत थका हुआ हूँ", RequestID: uuid.NewString(), Mode: models.EntryModeProcessing, Language: "hindi"}
	entry, fresh, err := r.CreateText(ctx, owner, &input)
	if err != nil || !fresh {
		t.Fatalf("create: %v %v", fresh, err)
	}
	if entry.Language == nil || *entry.Language != "hindi" {
		t.Fatalf("language not stored: %v", entry.Language)
	}
}
