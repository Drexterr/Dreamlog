//go:build integration

package repositories

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// Master-key rotation: the wrapped data keys can be listed, and replaced with
// compare-and-swap so exactly one of several concurrent re-wrappers wins.
func TestIntegration_WrappedDEK_ListAndCompareAndSwap(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	r := NewTherapistNotesRepository(db)

	mk := func() uuid.UUID {
		u := regressionUser(t, db)
		id := uuid.New()
		if _, err := db.Exec(ctx, `INSERT INTO therapists(id,user_id,name,email) VALUES($1,$2,'T',$3)`, id, u, id.String()+"@example.test"); err != nil {
			t.Fatal(err)
		}
		return id
	}
	a, b := mk(), mk()
	wrapA, wrapB := []byte("wrapped-a-old-master"), []byte("wrapped-b-old-master")
	for id, w := range map[uuid.UUID][]byte{a: wrapA, b: wrapB} {
		if _, err := r.InsertWrappedDEK(ctx, id, w); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := r.ListWrappedDEKs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := map[uuid.UUID][]byte{}
	for _, row := range rows {
		found[row.TherapistID] = row.Wrapped
	}
	if !bytes.Equal(found[a], wrapA) || !bytes.Equal(found[b], wrapB) {
		t.Fatalf("ListWrappedDEKs must return both keys: %v", found)
	}

	// Concurrent re-wrappers all start from wrapA; only one swap succeeds.
	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := r.ReplaceWrappedDEK(ctx, a, wrapA, []byte{'n', 'e', 'w', byte('0' + i)})
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("exactly one concurrent re-wrap may win, got %d", wins)
	}
	// A stale caller cannot overwrite the winner, and the other therapist is untouched.
	if ok, err := r.ReplaceWrappedDEK(ctx, a, wrapA, []byte("stale")); err != nil || ok {
		t.Fatalf("stale swap must be rejected: %v %v", ok, err)
	}
	if got, _ := r.GetWrappedDEK(ctx, b); !bytes.Equal(got, wrapB) {
		t.Fatalf("other therapist's key changed: %q", got)
	}
}
