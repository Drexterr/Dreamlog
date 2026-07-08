package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeNoteSessionStore struct {
	mu       sync.Mutex
	session  *models.ClientSessionRow
	failures []string
	final    bool
}

func (s *fakeNoteSessionStore) GetSessionByID(_ context.Context, id uuid.UUID) (*models.ClientSessionRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil || s.session.ID != id {
		return nil, nil
	}
	cp := *s.session
	return &cp, nil
}

func (s *fakeNoteSessionStore) SetSessionProcessing(_ context.Context, id uuid.UUID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil || s.session.ID != id || s.session.Status != models.ClientSessionPending {
		return false, nil
	}
	s.session.Status = models.ClientSessionProcessing
	return true, nil
}

func (s *fakeNoteSessionStore) CompleteSessionOCR(_ context.Context, id uuid.UUID, rawTextEnc, bulletsEnc []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil && s.session.ID == id {
		s.session.Status = models.ClientSessionCompleted
		s.session.RawTextEnc = rawTextEnc
		s.session.BulletsEnc = bulletsEnc
		s.session.ImageKey = nil
	}
	return nil
}

func (s *fakeNoteSessionStore) SetSessionFailed(_ context.Context, id uuid.UUID, errMsg string, final bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = append(s.failures, errMsg)
	s.final = final
	if s.session != nil && s.session.ID == id {
		if final {
			s.session.Status = models.ClientSessionFailed
		} else {
			s.session.Status = models.ClientSessionPending
		}
	}
	return nil
}

type fakeNoteOCRClient struct {
	out   *models.NotesOCROutput
	err   error
	calls int
}

func (f *fakeNoteOCRClient) ExtractNotesFromImage(_ context.Context, _ []byte, _ string) (*models.NotesOCROutput, error) {
	f.calls++
	return f.out, f.err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newNoteWorkerForTest(t *testing.T, store *fakeNoteSessionStore, storage *fakeStorage, ocr *fakeNoteOCRClient, queue *fakeQueue) (*NoteOCRWorker, *services.NotesCipher) {
	t.Helper()
	master, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	keys := &fakeNoteKeyStore{keys: map[uuid.UUID][]byte{}}
	cipher := services.NewNotesCipher(master, keys)
	w := NewNoteOCRWorker(NoteOCRWorkerDeps{
		Queue:      queue,
		Repo:       store,
		Storage:    storage,
		OCR:        ocr,
		Cipher:     cipher,
		Log:        zap.NewNop(),
		MaxRetries: 3,
	})
	return w, cipher
}

type fakeNoteKeyStore struct {
	mu   sync.Mutex
	keys map[uuid.UUID][]byte
}

func (s *fakeNoteKeyStore) GetWrappedDEK(_ context.Context, id uuid.UUID) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keys[id], nil
}

func (s *fakeNoteKeyStore) InsertWrappedDEK(_ context.Context, id uuid.UUID, wrapped []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.keys[id]; ok {
		return existing, nil
	}
	s.keys[id] = wrapped
	return wrapped, nil
}

func pendingNoteSession(therapistID uuid.UUID, imageKey string) *models.ClientSessionRow {
	clientID := uuid.New()
	return &models.ClientSessionRow{
		ID:               uuid.New(),
		TherapistID:      therapistID,
		ExternalClientID: &clientID,
		SessionDate:      time.Now(),
		Status:           models.ClientSessionPending,
		ImageKey:         &imageKey,
	}
}

func noteJobPayload(t *testing.T, job *models.NoteOCRJob) []byte {
	t.Helper()
	b, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestNoteOCR_HappyPath_EncryptsStoresAndDeletesImage(t *testing.T) {
	therapistID := uuid.New()
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	store := &fakeNoteSessionStore{session: pendingNoteSession(therapistID, imageKey)}
	storage := &fakeStorage{content: "fake-jpeg-bytes"}
	ocr := &fakeNoteOCRClient{out: &models.NotesOCROutput{
		RawText: "Client slept better. Set a boundary at work.",
		Bullets: []string{"Client slept better", "Set a boundary at work"},
	}}
	queue := &fakeQueue{}
	w, cipher := newNoteWorkerForTest(t, store, storage, ocr, queue)

	job := &models.NoteOCRJob{SessionID: store.session.ID, TherapistID: therapistID, ImageKey: imageKey}
	w.processJob(context.Background(), noteJobPayload(t, job))

	if store.session.Status != models.ClientSessionCompleted {
		t.Fatalf("want completed, got %s (failures: %v)", store.session.Status, store.failures)
	}
	// Image deleted after OCR.
	if len(storage.deleted) != 1 || storage.deleted[0] != imageKey {
		t.Fatalf("note image not deleted: %v", storage.deleted)
	}
	// Stored ciphertext holds no plaintext.
	if bytes.Contains(store.session.BulletsEnc, []byte("boundary")) ||
		bytes.Contains(store.session.RawTextEnc, []byte("boundary")) {
		t.Fatal("notes stored in plaintext")
	}
	// And decrypts back for the owning therapist.
	bullets, err := cipher.DecryptBullets(context.Background(), therapistID, store.session.BulletsEnc)
	if err != nil || len(bullets) != 2 {
		t.Fatalf("decrypt failed: %v %v", err, bullets)
	}
}

func TestNoteOCR_OCRFails_ImageKeptAndRetried(t *testing.T) {
	therapistID := uuid.New()
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	store := &fakeNoteSessionStore{session: pendingNoteSession(therapistID, imageKey)}
	storage := &fakeStorage{content: "fake-jpeg-bytes"}
	ocr := &fakeNoteOCRClient{err: errors.New("vision API down")}
	queue := &fakeQueue{}
	w, _ := newNoteWorkerForTest(t, store, storage, ocr, queue)

	job := &models.NoteOCRJob{SessionID: store.session.ID, TherapistID: therapistID, ImageKey: imageKey}
	w.processJob(context.Background(), noteJobPayload(t, job))

	// Image must NOT be deleted on failure - the retry needs it.
	if len(storage.deleted) != 0 {
		t.Fatalf("image deleted on failure: %v", storage.deleted)
	}
	// Retry re-enqueued with attempt+1.
	if len(queue.enqueued) != 1 {
		t.Fatalf("want 1 retry enqueue, got %d", len(queue.enqueued))
	}
	retry := queue.enqueued[0].(*models.NoteOCRJob)
	if retry.Attempt != 1 {
		t.Fatalf("want attempt 1, got %d", retry.Attempt)
	}
	if store.final {
		t.Fatal("first failure marked final")
	}
}

func TestNoteOCR_MaxRetries_DLQAndFinalFailure(t *testing.T) {
	therapistID := uuid.New()
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	store := &fakeNoteSessionStore{session: pendingNoteSession(therapistID, imageKey)}
	storage := &fakeStorage{content: "fake-jpeg-bytes"}
	ocr := &fakeNoteOCRClient{err: errors.New("vision API down")}
	queue := &fakeQueue{}
	w, _ := newNoteWorkerForTest(t, store, storage, ocr, queue)

	job := &models.NoteOCRJob{SessionID: store.session.ID, TherapistID: therapistID, ImageKey: imageKey, Attempt: 2}
	w.processJob(context.Background(), noteJobPayload(t, job))

	if store.session.Status != models.ClientSessionFailed || !store.final {
		t.Fatalf("want final failed, got %s final=%v", store.session.Status, store.final)
	}
	if len(queue.dlq) != 1 {
		t.Fatalf("want 1 DLQ entry, got %d", len(queue.dlq))
	}
	if len(queue.enqueued) != 0 {
		t.Fatal("job re-enqueued past max retries")
	}
}

func TestNoteOCR_EmptyResult_FinalFailureAndImageDeleted(t *testing.T) {
	therapistID := uuid.New()
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	store := &fakeNoteSessionStore{session: pendingNoteSession(therapistID, imageKey)}
	storage := &fakeStorage{content: "fake-jpeg-bytes"}
	ocr := &fakeNoteOCRClient{out: &models.NotesOCROutput{RawText: "", Bullets: []string{}}}
	queue := &fakeQueue{}
	w, _ := newNoteWorkerForTest(t, store, storage, ocr, queue)

	job := &models.NoteOCRJob{SessionID: store.session.ID, TherapistID: therapistID, ImageKey: imageKey}
	w.processJob(context.Background(), noteJobPayload(t, job))

	if store.session.Status != models.ClientSessionFailed || !store.final {
		t.Fatalf("want final failure for unreadable photo, got %s", store.session.Status)
	}
	if len(storage.deleted) != 1 {
		t.Fatal("unreadable image should still be deleted")
	}
	if len(queue.enqueued) != 0 {
		t.Fatal("unreadable photo should not retry")
	}
}

func TestNoteOCR_SessionDeletedBeforeJob_CleansUpImage(t *testing.T) {
	therapistID := uuid.New()
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	store := &fakeNoteSessionStore{session: nil} // session already deleted
	storage := &fakeStorage{content: "fake-jpeg-bytes"}
	ocr := &fakeNoteOCRClient{}
	queue := &fakeQueue{}
	w, _ := newNoteWorkerForTest(t, store, storage, ocr, queue)

	job := &models.NoteOCRJob{SessionID: uuid.New(), TherapistID: therapistID, ImageKey: imageKey}
	w.processJob(context.Background(), noteJobPayload(t, job))

	if len(storage.deleted) != 1 || storage.deleted[0] != imageKey {
		t.Fatalf("orphaned image not cleaned up: %v", storage.deleted)
	}
	if ocr.calls != 0 {
		t.Fatal("OCR ran for a deleted session")
	}
}

func TestNoteOCR_AlreadyCompleted_Skips(t *testing.T) {
	therapistID := uuid.New()
	imageKey := "notes/" + therapistID.String() + "/photo.jpg"
	session := pendingNoteSession(therapistID, imageKey)
	session.Status = models.ClientSessionCompleted
	store := &fakeNoteSessionStore{session: session}
	storage := &fakeStorage{content: "fake-jpeg-bytes"}
	ocr := &fakeNoteOCRClient{}
	queue := &fakeQueue{}
	w, _ := newNoteWorkerForTest(t, store, storage, ocr, queue)

	job := &models.NoteOCRJob{SessionID: session.ID, TherapistID: therapistID, ImageKey: imageKey}
	w.processJob(context.Background(), noteJobPayload(t, job))

	if ocr.calls != 0 {
		t.Fatal("OCR re-ran for a completed session")
	}
	if len(queue.dlq) != 0 || len(queue.enqueued) != 0 {
		t.Fatal("completed session produced queue activity")
	}
}
