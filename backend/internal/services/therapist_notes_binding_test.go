package services

import (
	"context"
	"testing"

	"github.com/dreamlog/backend/internal/repositories"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/google/uuid"
)

// M10: two clients of the SAME therapist share a key, so without row binding a
// DB-level attacker could swap their encrypted names undetected.
func TestNotes_SwappedCiphertextBetweenRows_Rejected(t *testing.T) {
	svc, repo, _, _, _ := newNotesServiceForTest(t)
	ctx := context.Background()
	therapist := uuid.New()
	a, err := svc.CreateExternalClient(ctx, therapist, "Asha K", "client")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateExternalClient(ctx, therapist, "Ravi M", "client")
	if err != nil {
		t.Fatal(err)
	}
	if !pkgcrypto.IsSealed(repo.clients[a.ID].NameEnc) {
		t.Fatal("new ciphertexts must be written in the bound (v2) format")
	}

	repo.clients[a.ID].NameEnc, repo.clients[b.ID].NameEnc = repo.clients[b.ID].NameEnc, repo.clients[a.ID].NameEnc
	if _, err := svc.GetExternalClient(ctx, therapist, a.ID); err == nil {
		t.Fatal("a ciphertext swapped in from another row must not decrypt")
	}
}

func TestNotes_CiphertextMovedBetweenColumns_Rejected(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	ctx := context.Background()
	therapist, session := uuid.New(), uuid.New()
	summary, err := svc.cipher.EncryptField(ctx, therapist, BindSessionSummary(session), "summary text")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.cipher.DecryptField(ctx, therapist, BindSessionRawText(session), summary); err == nil {
		t.Fatal("a summary ciphertext must not decrypt as raw text")
	}
}

// upgradeRepo adds the legacy-store methods to the in-memory fake.
type upgradeRepo struct{ *fakeNotesRepo }

func (r *upgradeRepo) ListExternalClientCiphers(context.Context) ([]repositories.CipherRow, error) {
	var out []repositories.CipherRow
	for _, c := range r.clients {
		out = append(out, repositories.CipherRow{ID: c.ID, TherapistID: c.TherapistID, Fields: [3][]byte{c.NameEnc}})
	}
	return out, nil
}
func (r *upgradeRepo) SetExternalClientNameEnc(_ context.Context, id uuid.UUID, enc []byte) error {
	r.clients[id].NameEnc = enc
	return nil
}
func (r *upgradeRepo) ListSessionCiphers(context.Context) ([]repositories.CipherRow, error) {
	var out []repositories.CipherRow
	for _, s := range r.sessions {
		out = append(out, repositories.CipherRow{ID: s.ID, TherapistID: s.TherapistID, Fields: [3][]byte{s.RawTextEnc, s.BulletsEnc, s.SummaryEnc}})
	}
	return out, nil
}
func (r *upgradeRepo) SetSessionCiphers(_ context.Context, id uuid.UUID, raw, bullets, summary []byte) error {
	s := r.sessions[id]
	s.RawTextEnc, s.BulletsEnc, s.SummaryEnc = raw, bullets, summary
	return nil
}

// M10: rows written before binding are re-sealed in place and stay readable.
func TestNotes_UpgradeLegacyCiphertexts(t *testing.T) {
	base := newFakeNotesRepo()
	repo := &upgradeRepo{base}
	master, _ := pkgcrypto.NewDEK()
	cipher := NewNotesCipher(master, repo)
	svc := NewTherapistNotesService(repo, &fakeLinkChecker{activeLinks: map[uuid.UUID]bool{}}, &fakeNotesQueue{}, &fakeNotesStorage{}, &fakeSummarizer{}, cipher)
	ctx := context.Background()
	therapist := uuid.New()

	// A client stored in the legacy (unbound) format.
	dek, err := cipher.dek(ctx, therapist)
	if err != nil {
		t.Fatal(err)
	}
	legacy, _ := pkgcrypto.Encrypt(dek, []byte("Legacy Name"))
	clientID := uuid.New()
	if _, err := base.CreateExternalClient(ctx, clientID, therapist, legacy, "client"); err != nil {
		t.Fatal(err)
	}

	n, err := svc.UpgradeLegacyCiphertexts(ctx)
	if err != nil || n != 1 {
		t.Fatalf("upgrade: n=%d err=%v", n, err)
	}
	if !pkgcrypto.IsSealed(base.clients[clientID].NameEnc) {
		t.Fatal("legacy row must be re-sealed")
	}
	got, err := svc.GetExternalClient(ctx, therapist, clientID)
	if err != nil || got.Name != "Legacy Name" {
		t.Fatalf("upgraded row must still decrypt: %+v %v", got, err)
	}
	if n, err := svc.UpgradeLegacyCiphertexts(ctx); err != nil || n != 0 {
		t.Fatalf("second run must be a no-op: n=%d err=%v", n, err)
	}
}

// N15: with RejectLegacy on, an unbound (v1) ciphertext - e.g. restored from a
// backup or copied in from another row - is refused instead of decrypted,
// while the startup upgrader can still re-seal it.
func TestNotes_RejectLegacyCiphertext(t *testing.T) {
	base := newFakeNotesRepo()
	repo := &upgradeRepo{base}
	master, _ := pkgcrypto.NewDEK()
	cipher := NewNotesCipher(master, repo).RejectLegacy(true)
	svc := NewTherapistNotesService(repo, &fakeLinkChecker{activeLinks: map[uuid.UUID]bool{}}, &fakeNotesQueue{}, &fakeNotesStorage{}, &fakeSummarizer{}, cipher)
	ctx := context.Background()
	therapist := uuid.New()
	dek, err := cipher.dek(ctx, therapist)
	if err != nil {
		t.Fatal(err)
	}
	legacy, _ := pkgcrypto.Encrypt(dek, []byte("Legacy Name"))
	clientID := uuid.New()
	if _, err := base.CreateExternalClient(ctx, clientID, therapist, legacy, "client"); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GetExternalClient(ctx, therapist, clientID); err == nil {
		t.Fatal("a legacy ciphertext must not be readable once legacy is rejected")
	}
	if n, err := svc.UpgradeLegacyCiphertexts(ctx); err != nil || n != 1 {
		t.Fatalf("the upgrader must still re-seal legacy rows: n=%d err=%v", n, err)
	}
	got, err := svc.GetExternalClient(ctx, therapist, clientID)
	if err != nil || got.Name != "Legacy Name" {
		t.Fatalf("re-sealed row must decrypt: %+v %v", got, err)
	}
}
