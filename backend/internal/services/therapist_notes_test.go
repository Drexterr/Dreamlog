package services

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/google/uuid"
)

// ── in-memory fakes ───────────────────────────────────────────────────────────

type fakeNotesRepo struct {
	mu       sync.Mutex
	keys     map[uuid.UUID][]byte
	clients  map[uuid.UUID]*models.ExternalClientRow
	sessions map[uuid.UUID]*models.ClientSessionRow
}

func newFakeNotesRepo() *fakeNotesRepo {
	return &fakeNotesRepo{
		keys:     map[uuid.UUID][]byte{},
		clients:  map[uuid.UUID]*models.ExternalClientRow{},
		sessions: map[uuid.UUID]*models.ClientSessionRow{},
	}
}

func (r *fakeNotesRepo) GetWrappedDEK(_ context.Context, id uuid.UUID) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.keys[id], nil
}

func (r *fakeNotesRepo) InsertWrappedDEK(_ context.Context, id uuid.UUID, wrapped []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.keys[id]; ok {
		return existing, nil // race: stored key wins
	}
	r.keys[id] = wrapped
	return wrapped, nil
}

func (r *fakeNotesRepo) CreateExternalClient(_ context.Context, therapistID uuid.UUID, nameEnc []byte, role string) (*models.ExternalClientRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := &models.ExternalClientRow{
		ID: uuid.New(), TherapistID: therapistID, NameEnc: nameEnc, Role: role,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	r.clients[row.ID] = row
	return row, nil
}

func (r *fakeNotesRepo) ListExternalClients(_ context.Context, therapistID uuid.UUID, includeArchived bool) ([]*models.ExternalClientRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.ExternalClientRow
	for _, c := range r.clients {
		if c.TherapistID == therapistID && (!c.Archived || includeArchived) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeNotesRepo) GetExternalClient(_ context.Context, therapistID, clientID uuid.UUID) (*models.ExternalClientRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[clientID]
	if !ok || c.TherapistID != therapistID {
		return nil, nil
	}
	return c, nil
}

func (r *fakeNotesRepo) UpdateExternalClient(_ context.Context, therapistID, clientID uuid.UUID, nameEnc []byte, role *string, archived *bool) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[clientID]
	if !ok || c.TherapistID != therapistID {
		return false, nil
	}
	if nameEnc != nil {
		c.NameEnc = nameEnc
	}
	if role != nil {
		c.Role = *role
	}
	if archived != nil {
		c.Archived = *archived
	}
	return true, nil
}

func (r *fakeNotesRepo) DeleteExternalClient(_ context.Context, therapistID, clientID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[clientID]
	if !ok || c.TherapistID != therapistID {
		return false, nil
	}
	delete(r.clients, clientID)
	return true, nil
}

func (r *fakeNotesRepo) CreateSession(_ context.Context, s *models.ClientSessionRow) (*models.ClientSessionRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	r.sessions[s.ID] = s
	return s, nil
}

func (r *fakeNotesRepo) GetSession(_ context.Context, therapistID, sessionID uuid.UUID) (*models.ClientSessionRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	if !ok || s.TherapistID != therapistID {
		return nil, nil
	}
	return s, nil
}

func (r *fakeNotesRepo) ListSessionsForExternalClient(_ context.Context, therapistID, clientID uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.ClientSessionRow
	for _, s := range r.sessions {
		if s.TherapistID == therapistID && s.ExternalClientID != nil && *s.ExternalClientID == clientID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeNotesRepo) ListSessionsForLinkedClient(_ context.Context, therapistID, linkedUserID uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.ClientSessionRow
	for _, s := range r.sessions {
		if s.TherapistID == therapistID && s.LinkedUserID != nil && *s.LinkedUserID == linkedUserID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeNotesRepo) ListRecentSessions(_ context.Context, therapistID uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.ClientSessionRow
	for _, s := range r.sessions {
		if s.TherapistID == therapistID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeNotesRepo) UpdateSessionBullets(_ context.Context, therapistID, sessionID uuid.UUID, bulletsEnc []byte) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	if !ok || s.TherapistID != therapistID {
		return false, nil
	}
	s.BulletsEnc = bulletsEnc
	return true, nil
}

func (r *fakeNotesRepo) UpdateSessionSummary(_ context.Context, therapistID, sessionID uuid.UUID, summaryEnc []byte) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	if !ok || s.TherapistID != therapistID {
		return false, nil
	}
	s.SummaryEnc = summaryEnc
	return true, nil
}

func (r *fakeNotesRepo) DeleteSession(_ context.Context, therapistID, sessionID uuid.UUID) (*string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	if !ok || s.TherapistID != therapistID {
		return nil, false, nil
	}
	delete(r.sessions, sessionID)
	return s.ImageKey, true, nil
}

func (r *fakeNotesRepo) Overview(_ context.Context, therapistID uuid.UUID) (*models.TherapistOverview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o := &models.TherapistOverview{}
	for _, c := range r.clients {
		if c.TherapistID == therapistID && !c.Archived {
			o.ExternalClients++
		}
	}
	for _, s := range r.sessions {
		if s.TherapistID == therapistID {
			o.TotalSessions++
		}
	}
	return o, nil
}

type fakeLinkChecker struct {
	activeLinks map[uuid.UUID]bool // clientID → active
}

func (f *fakeLinkChecker) GetClientLink(_ context.Context, therapistID, clientID uuid.UUID) (*models.ClientTherapistLink, error) {
	if f.activeLinks[clientID] {
		return &models.ClientTherapistLink{TherapistID: therapistID, ClientID: clientID, Status: "active"}, nil
	}
	return nil, nil
}

type fakeNotesQueue struct {
	mu       sync.Mutex
	enqueued []any
}

func (q *fakeNotesQueue) Enqueue(_ context.Context, v any) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueued = append(q.enqueued, v)
	return nil
}

type fakeNotesStorage struct {
	deleted []string
}

func (s *fakeNotesStorage) PresignPutKey(_ context.Context, _ string) (string, error) {
	return "https://storage.example/put", nil
}

func (s *fakeNotesStorage) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

type fakeSummarizer struct {
	summary  string
	calls    int
	gotLabel string
}

func (f *fakeSummarizer) GenerateSessionNotesSummary(_ context.Context, clientLabel, _ string, _ []string) (string, error) {
	f.calls++
	f.gotLabel = clientLabel
	return f.summary, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newNotesServiceForTest(t *testing.T) (*TherapistNotesService, *fakeNotesRepo, *fakeNotesQueue, *fakeLinkChecker, *fakeSummarizer) {
	t.Helper()
	repo := newFakeNotesRepo()
	queue := &fakeNotesQueue{}
	links := &fakeLinkChecker{activeLinks: map[uuid.UUID]bool{}}
	summarizer := &fakeSummarizer{summary: "A focused session covering sleep and boundaries."}
	master, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	cipher := NewNotesCipher(master, repo)
	svc := NewTherapistNotesService(repo, links, queue, &fakeNotesStorage{}, summarizer, cipher)
	return svc, repo, queue, links, summarizer
}

func isAPIErrorWithCode(t *testing.T, err error, code int) {
	t.Helper()
	apiErr, ok := apierr.As(err)
	if !ok {
		t.Fatalf("want APIError %d, got %T: %v", code, err, err)
	}
	if apiErr.Code != code {
		t.Fatalf("want status %d, got %d (%s)", code, apiErr.Code, apiErr.Message)
	}
}

// ── external clients ─────────────────────────────────────────────────────────

func TestNotes_CreateExternalClient_NameEncryptedAtRest(t *testing.T) {
	svc, repo, _, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()

	client, err := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	if err != nil {
		t.Fatal(err)
	}
	if client.Name != "Asha K" {
		t.Fatalf("decrypted name mismatch: %q", client.Name)
	}

	row := repo.clients[client.ID]
	if bytes.Contains(row.NameEnc, []byte("Asha")) {
		t.Fatal("client name stored in plaintext")
	}
}

func TestNotes_CreateExternalClient_EmptyNameRejected(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	_, err := svc.CreateExternalClient(context.Background(), uuid.New(), "   ", "")
	isAPIErrorWithCode(t, err, 400)
}

func TestNotes_ExternalClient_OwnershipIsolation(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	therapistA, therapistB := uuid.New(), uuid.New()

	client, err := svc.CreateExternalClient(context.Background(), therapistA, "Asha K", "client")
	if err != nil {
		t.Fatal(err)
	}

	// Therapist B cannot read, update, or delete A's client.
	if _, err := svc.GetExternalClient(context.Background(), therapistB, client.ID); err == nil {
		t.Fatal("therapist B read therapist A's client")
	}
	name := "Hacked"
	if _, err := svc.UpdateExternalClient(context.Background(), therapistB, client.ID, UpdateExternalClientInput{Name: &name}); err == nil {
		t.Fatal("therapist B updated therapist A's client")
	}
	if err := svc.DeleteExternalClient(context.Background(), therapistB, client.ID); err == nil {
		t.Fatal("therapist B deleted therapist A's client")
	}

	// A still sees the client, unchanged.
	got, err := svc.GetExternalClient(context.Background(), therapistA, client.ID)
	if err != nil || got.Name != "Asha K" {
		t.Fatalf("owner lost access: %v", err)
	}
}

// ── sessions ─────────────────────────────────────────────────────────────────

func TestNotes_CreateSession_Validation(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()
	clientID := uuid.New()

	cases := []struct {
		name  string
		input CreateSessionInput
	}{
		{"no client ref", CreateSessionInput{Bullets: []string{"a"}}},
		{"both client refs", CreateSessionInput{ExternalClientID: &clientID, LinkedClientID: &clientID, Bullets: []string{"a"}}},
		{"no content", CreateSessionInput{ExternalClientID: &clientID}},
		{"both content kinds", CreateSessionInput{ExternalClientID: &clientID, ImageKey: "notes/x/y.jpg", Bullets: []string{"a"}}},
		{"bad date", CreateSessionInput{ExternalClientID: &clientID, Bullets: []string{"a"}, SessionDate: "07-07-2026"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateSession(context.Background(), therapistID, tc.input)
			isAPIErrorWithCode(t, err, 400)
		})
	}
}

func TestNotes_CreateSession_ManualBullets_CompletedAndEncrypted(t *testing.T) {
	svc, repo, queue, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()

	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	session, err := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID,
		Bullets:          []string{"  discussed sleep hygiene  ", "", "set one boundary at work"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != models.ClientSessionCompleted {
		t.Fatalf("manual session should be completed, got %s", session.Status)
	}
	if len(session.Bullets) != 2 || session.Bullets[0] != "discussed sleep hygiene" {
		t.Fatalf("bullets not cleaned: %v", session.Bullets)
	}
	if len(queue.enqueued) != 0 {
		t.Fatal("manual session should not enqueue an OCR job")
	}

	// No plaintext at rest.
	row := repo.sessions[session.ID]
	if bytes.Contains(row.BulletsEnc, []byte("sleep hygiene")) {
		t.Fatal("bullets stored in plaintext")
	}
}

func TestNotes_CreateSession_ImageEnqueuesOCRJob(t *testing.T) {
	svc, _, queue, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()

	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	session, err := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID,
		ImageKey:         imageKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != models.ClientSessionPending {
		t.Fatalf("image session should be pending, got %s", session.Status)
	}
	if len(queue.enqueued) != 1 {
		t.Fatalf("want 1 OCR job, got %d", len(queue.enqueued))
	}
	job := queue.enqueued[0].(*models.NoteOCRJob)
	if job.SessionID != session.ID || job.ImageKey != imageKey || job.TherapistID != therapistID {
		t.Fatalf("job fields wrong: %+v", job)
	}
}

func TestNotes_PresignKey_AcceptedByCreateSession(t *testing.T) {
	svc, _, queue, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()
	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")

	// The key issued by presign must pass CreateSession's ownership prefix
	// check verbatim - regression guard for key re-namespacing bugs.
	_, key, err := svc.PresignNoteUpload(context.Background(), therapistID, "my notes.jpg", "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "notes/"+therapistID.String()+"/") {
		t.Fatalf("presign key not therapist-scoped: %q", key)
	}
	if !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("presign key lost extension: %q", key)
	}

	session, err := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID,
		ImageKey:         key,
	})
	if err != nil {
		t.Fatalf("presigned key rejected by CreateSession: %v", err)
	}
	if session.Status != models.ClientSessionPending || len(queue.enqueued) != 1 {
		t.Fatalf("session not queued: %s / %d jobs", session.Status, len(queue.enqueued))
	}
}

func TestNotes_Presign_RejectsBadContentType(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	_, _, err := svc.PresignNoteUpload(context.Background(), uuid.New(), "notes.pdf", "application/pdf")
	isAPIErrorWithCode(t, err, 400)
}

func TestNotes_CreateSession_ForeignImageKeyRejected(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()
	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")

	// Key under another therapist's prefix must be rejected.
	_, err := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID,
		ImageKey:         "notes/" + uuid.New().String() + "/stolen.jpg",
	})
	isAPIErrorWithCode(t, err, 400)
}

func TestNotes_CreateSession_UnknownExternalClient404(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	unknown := uuid.New()
	_, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionInput{
		ExternalClientID: &unknown,
		Bullets:          []string{"a"},
	})
	isAPIErrorWithCode(t, err, 404)
}

func TestNotes_CreateSession_LinkedClientRequiresActiveLink(t *testing.T) {
	svc, _, _, links, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()
	linkedUser := uuid.New()

	// No consented link → 404.
	_, err := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		LinkedClientID: &linkedUser,
		Bullets:        []string{"a"},
	})
	isAPIErrorWithCode(t, err, 404)

	// Active link → allowed.
	links.activeLinks[linkedUser] = true
	session, err := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		LinkedClientID: &linkedUser,
		Bullets:        []string{"discussed progress"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.LinkedUserID == nil || *session.LinkedUserID != linkedUser {
		t.Fatal("linked user not recorded")
	}
}

func TestNotes_UpdateBullets_RoundTripAndIsolation(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()
	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	session, _ := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID, Bullets: []string{"original"},
	})

	updated, err := svc.UpdateBullets(context.Background(), therapistID, session.ID, []string{"edited bullet", "new bullet"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Bullets) != 2 || updated.Bullets[0] != "edited bullet" {
		t.Fatalf("bullets not updated: %v", updated.Bullets)
	}

	// Another therapist cannot edit.
	_, err = svc.UpdateBullets(context.Background(), uuid.New(), session.ID, []string{"hijack"})
	isAPIErrorWithCode(t, err, 404)
}

func TestNotes_Summarize_StoresEncryptedAndAnonymizesClient(t *testing.T) {
	svc, repo, _, _, summarizer := newNotesServiceForTest(t)
	therapistID := uuid.New()
	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	session, _ := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID, Bullets: []string{"discussed sleep"},
	})

	got, err := svc.Summarize(context.Background(), therapistID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != summarizer.summary {
		t.Fatalf("summary mismatch: %q", got.Summary)
	}
	if summarizer.gotLabel != "the client" {
		t.Fatalf("client identity leaked to AI prompt: %q", summarizer.gotLabel)
	}
	row := repo.sessions[session.ID]
	if bytes.Contains(row.SummaryEnc, []byte("focused session")) {
		t.Fatal("summary stored in plaintext")
	}
}

func TestNotes_Summarize_PendingSessionConflicts(t *testing.T) {
	svc, _, _, _, _ := newNotesServiceForTest(t)
	therapistID := uuid.New()
	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	session, _ := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID,
		ImageKey:         "notes/" + therapistID.String() + "/photo.jpg",
	})

	_, err := svc.Summarize(context.Background(), therapistID, session.ID)
	isAPIErrorWithCode(t, err, 409)
}

func TestNotes_DeleteSession_RemovesOrphanImage(t *testing.T) {
	repo := newFakeNotesRepo()
	storage := &fakeNotesStorage{}
	master, _ := pkgcrypto.NewDEK()
	cipher := NewNotesCipher(master, repo)
	svc := NewTherapistNotesService(repo, &fakeLinkChecker{activeLinks: map[uuid.UUID]bool{}}, &fakeNotesQueue{}, storage, &fakeSummarizer{}, cipher)

	therapistID := uuid.New()
	client, _ := svc.CreateExternalClient(context.Background(), therapistID, "Asha K", "client")
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	session, _ := svc.CreateSession(context.Background(), therapistID, CreateSessionInput{
		ExternalClientID: &client.ID, ImageKey: imageKey,
	})

	if err := svc.DeleteSession(context.Background(), therapistID, session.ID); err != nil {
		t.Fatal(err)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != imageKey {
		t.Fatalf("orphan image not deleted: %v", storage.deleted)
	}
}

// ── cipher ───────────────────────────────────────────────────────────────────

func TestNotesCipher_PerTherapistKeys(t *testing.T) {
	repo := newFakeNotesRepo()
	master, _ := pkgcrypto.NewDEK()
	cipher := NewNotesCipher(master, repo)

	a, b := uuid.New(), uuid.New()
	ctA, err := cipher.EncryptField(context.Background(), a, "secret note")
	if err != nil {
		t.Fatal(err)
	}

	// Therapist B's key cannot open A's ciphertext.
	if _, err := cipher.DecryptField(context.Background(), b, ctA); err == nil {
		t.Fatal("cross-therapist decryption succeeded")
	}
	// A's own round trip works.
	got, err := cipher.DecryptField(context.Background(), a, ctA)
	if err != nil || got != "secret note" {
		t.Fatalf("round trip failed: %v %q", err, got)
	}
}

func TestNotesCipher_BulletsRoundTrip(t *testing.T) {
	repo := newFakeNotesRepo()
	master, _ := pkgcrypto.NewDEK()
	cipher := NewNotesCipher(master, repo)
	id := uuid.New()

	bullets := []string{"point one", "point two — हिंदी"}
	enc, err := cipher.EncryptBullets(context.Background(), id, bullets)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(enc), "point one") {
		t.Fatal("bullets ciphertext contains plaintext")
	}
	got, err := cipher.DecryptBullets(context.Background(), id, enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != bullets[1] {
		t.Fatalf("bullets mismatch: %v", got)
	}
}
