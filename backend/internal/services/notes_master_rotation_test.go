package services

import (
	"context"
	"testing"

	"github.com/dreamlog/backend/internal/repositories"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/google/uuid"
)

// rotateRepo adds the production repository's key-listing and
// compare-and-swap methods to the in-memory notes fake.
type rotateRepo struct {
	*fakeNotesRepo
	// beforeSwap runs inside ReplaceWrappedDEK, to simulate a concurrent re-wrap.
	beforeSwap func(id uuid.UUID)
}

func (r *rotateRepo) ListWrappedDEKs(context.Context) ([]repositories.WrappedKeyRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []repositories.WrappedKeyRow
	for id, w := range r.keys {
		out = append(out, repositories.WrappedKeyRow{TherapistID: id, Wrapped: append([]byte(nil), w...)})
	}
	return out, nil
}

func (r *rotateRepo) ReplaceWrappedDEK(_ context.Context, id uuid.UUID, old, replacement []byte) (bool, error) {
	if r.beforeSwap != nil {
		r.beforeSwap(id)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if string(r.keys[id]) != string(old) {
		return false, nil
	}
	r.keys[id] = replacement
	return true, nil
}

func newKey(t *testing.T) []byte {
	t.Helper()
	k, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func opensUnder(master, wrapped []byte) bool {
	_, err := pkgcrypto.UnwrapDEK(master, wrapped)
	return err == nil
}

// A therapist's data key written under the old master key stays readable after
// the master key changes, as long as the old key is supplied as the previous
// key, and is re-wrapped under the new key on first use.
func TestNotesRotation_PreviousKeyKeepsDataReadableAndRewrapsOnUse(t *testing.T) {
	ctx := context.Background()
	repo := &rotateRepo{fakeNotesRepo: newFakeNotesRepo()}
	oldKey, newKey1 := newKey(t), newKey(t)
	therapist := uuid.New()

	ct, err := NewNotesCipher(oldKey, repo).EncryptField(ctx, therapist, "bind", "client secret")
	if err != nil {
		t.Fatal(err)
	}
	if !opensUnder(oldKey, repo.keys[therapist]) {
		t.Fatal("precondition: data key is wrapped under the old master key")
	}

	// The new key alone cannot read it - rotating without PREVIOUS would lose the notes.
	if _, err := NewNotesCipher(newKey1, repo).DecryptField(ctx, therapist, "bind", ct); err == nil {
		t.Fatal("the new master key alone must not open old data keys")
	}

	rotated := NewNotesCipher(newKey1, repo).WithPreviousKey(oldKey)
	got, err := rotated.DecryptField(ctx, therapist, "bind", ct)
	if err != nil || got != "client secret" {
		t.Fatalf("with the previous key the data must stay readable: %q %v", got, err)
	}
	if !opensUnder(newKey1, repo.keys[therapist]) || opensUnder(oldKey, repo.keys[therapist]) {
		t.Fatal("using the key must re-wrap it under the new master key only")
	}
	// After the re-wrap the new key alone is enough.
	if got, err := NewNotesCipher(newKey1, repo).DecryptField(ctx, therapist, "bind", ct); err != nil || got != "client secret" {
		t.Fatalf("new key alone after re-wrap: %q %v", got, err)
	}
}

func TestNotesRotation_RewrapMasterKey_AllTherapistsAndIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := &rotateRepo{fakeNotesRepo: newFakeNotesRepo()}
	oldKey, newKey1 := newKey(t), newKey(t)
	old := NewNotesCipher(oldKey, repo)
	var therapists []uuid.UUID
	cts := map[uuid.UUID][]byte{}
	for i := 0; i < 5; i++ {
		id := uuid.New()
		therapists = append(therapists, id)
		ct, err := old.EncryptField(ctx, id, "b", "secret-"+id.String())
		if err != nil {
			t.Fatal(err)
		}
		cts[id] = ct
	}
	// One therapist is already on the new key (e.g. created after the rotation).
	fresh := NewNotesCipher(newKey1, repo)
	freshID := uuid.New()
	freshCT, err := fresh.EncryptField(ctx, freshID, "b", "fresh")
	if err != nil {
		t.Fatal(err)
	}

	rot := NewNotesCipher(newKey1, repo).WithPreviousKey(oldKey)
	n, err := rot.RewrapMasterKey(ctx)
	if err != nil || n != 5 {
		t.Fatalf("want 5 re-wrapped, got %d (%v)", n, err)
	}
	for _, id := range append(therapists, freshID) {
		if !opensUnder(newKey1, repo.keys[id]) || opensUnder(oldKey, repo.keys[id]) {
			t.Fatalf("therapist %s must be wrapped under the new key only", id)
		}
	}
	if n, err := rot.RewrapMasterKey(ctx); err != nil || n != 0 {
		t.Fatalf("second run must be a no-op, got %d (%v)", n, err)
	}
	// Data is intact with ONLY the new key; the old key is no longer needed.
	only := NewNotesCipher(newKey1, repo)
	for _, id := range therapists {
		if got, err := only.DecryptField(ctx, id, "b", cts[id]); err != nil || got != "secret-"+id.String() {
			t.Fatalf("therapist %s after rotation: %q %v", id, got, err)
		}
	}
	if got, err := only.DecryptField(ctx, freshID, "b", freshCT); err != nil || got != "fresh" {
		t.Fatalf("fresh therapist: %q %v", got, err)
	}
}

// A data key that neither master key opens is reported, never silently
// skipped, and does not stop the others from being re-wrapped.
func TestNotesRotation_UnopenableKeyReportedOthersRewrapped(t *testing.T) {
	ctx := context.Background()
	repo := &rotateRepo{fakeNotesRepo: newFakeNotesRepo()}
	oldKey, newKey1, stranger := newKey(t), newKey(t), newKey(t)
	good, bad := uuid.New(), uuid.New()
	if _, err := NewNotesCipher(oldKey, repo).EncryptField(ctx, good, "b", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewNotesCipher(stranger, repo).EncryptField(ctx, bad, "b", "x"); err != nil {
		t.Fatal(err)
	}
	n, err := NewNotesCipher(newKey1, repo).WithPreviousKey(oldKey).RewrapMasterKey(ctx)
	if n != 1 || err == nil {
		t.Fatalf("want 1 re-wrapped and an error for the unopenable key, got %d %v", n, err)
	}
	if !opensUnder(newKey1, repo.keys[good]) {
		t.Fatal("the openable key must still be re-wrapped")
	}
	if !opensUnder(stranger, repo.keys[bad]) {
		t.Fatal("the unopenable key must be left untouched")
	}
}

// If another instance re-wrapped the key first, this instance does not
// overwrite it or count it.
func TestNotesRotation_LostRaceIsNotCounted(t *testing.T) {
	ctx := context.Background()
	repo := &rotateRepo{fakeNotesRepo: newFakeNotesRepo()}
	oldKey, newKey1 := newKey(t), newKey(t)
	id := uuid.New()
	if _, err := NewNotesCipher(oldKey, repo).EncryptField(ctx, id, "b", "x"); err != nil {
		t.Fatal(err)
	}
	var other []byte
	repo.beforeSwap = func(tid uuid.UUID) {
		// Another instance wins the swap first.
		dek, _ := pkgcrypto.UnwrapDEK(oldKey, repo.keys[tid])
		other, _ = pkgcrypto.WrapDEK(newKey1, dek)
		repo.mu.Lock()
		repo.keys[tid] = other
		repo.mu.Unlock()
	}
	n, err := NewNotesCipher(newKey1, repo).WithPreviousKey(oldKey).RewrapMasterKey(ctx)
	if err != nil || n != 0 {
		t.Fatalf("a lost race must not be counted: %d %v", n, err)
	}
	if string(repo.keys[id]) != string(other) {
		t.Fatal("the winner's wrapped key must be kept")
	}
}

func TestNotesRotation_NewTherapistKeysUseTheNewMasterKey(t *testing.T) {
	ctx := context.Background()
	repo := &rotateRepo{fakeNotesRepo: newFakeNotesRepo()}
	oldKey, newKey1 := newKey(t), newKey(t)
	id := uuid.New()
	if _, err := NewNotesCipher(newKey1, repo).WithPreviousKey(oldKey).EncryptField(ctx, id, "b", "x"); err != nil {
		t.Fatal(err)
	}
	if !opensUnder(newKey1, repo.keys[id]) || opensUnder(oldKey, repo.keys[id]) {
		t.Fatal("a data key created during rotation must be wrapped under the new master key")
	}
}

func TestNotesRotation_PreviousKeyDisabledWhenEmptyOrSame(t *testing.T) {
	ctx := context.Background()
	repo := &rotateRepo{fakeNotesRepo: newFakeNotesRepo()}
	k := newKey(t)
	for name, prev := range map[string][]byte{"nil": nil, "same": k} {
		c := NewNotesCipher(k, repo).WithPreviousKey(prev)
		if c.previousKey != nil {
			t.Errorf("%s: previous key must be ignored", name)
		}
		if n, err := c.RewrapMasterKey(ctx); err != nil || n != 0 {
			t.Errorf("%s: re-wrap must be a no-op, got %d %v", name, n, err)
		}
	}
}

// A store that cannot list or swap keys (any test double) simply skips the
// re-wrap; reads through the previous key still work.
func TestNotesRotation_StoreWithoutRewrapSupport(t *testing.T) {
	ctx := context.Background()
	plain := newFakeNotesRepo()
	oldKey, newKey1 := newKey(t), newKey(t)
	id := uuid.New()
	ct, err := NewNotesCipher(oldKey, plain).EncryptField(ctx, id, "b", "x")
	if err != nil {
		t.Fatal(err)
	}
	c := NewNotesCipher(newKey1, plain).WithPreviousKey(oldKey)
	if n, err := c.RewrapMasterKey(ctx); err != nil || n != 0 {
		t.Fatalf("unsupported store: %d %v", n, err)
	}
	if got, err := c.DecryptField(ctx, id, "b", ct); err != nil || got != "x" {
		t.Fatalf("reads must still work via the previous key: %q %v", got, err)
	}
}
