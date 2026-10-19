package workers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var expansionErr = errors.New("injected failure")

type expansionEntryStore struct {
	*fakeEntryStore
	stage string
}

func (s *expansionEntryStore) SetProcessing(ctx context.Context, id uuid.UUID) (bool, error) {
	if s.stage == "processing" {
		return false, expansionErr
	}
	if s.stage == "unclaimed" {
		return false, nil
	}
	return s.fakeEntryStore.SetProcessing(ctx, id)
}
func (s *expansionEntryStore) GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Entry, error) {
	if s.stage == "fetch" {
		return nil, expansionErr
	}
	if s.stage == "missing" {
		return nil, nil
	}
	return s.fakeEntryStore.GetByIDInternal(ctx, id)
}
func (s *expansionEntryStore) SetTranscript(ctx context.Context, id uuid.UUID, text, lang string) error {
	if s.stage == "transcript" {
		return expansionErr
	}
	return s.fakeEntryStore.SetTranscript(ctx, id, text, lang)
}
func (s *expansionEntryStore) SetCompleted(ctx context.Context, id uuid.UUID, text, lang string) error {
	if s.stage == "complete" {
		return expansionErr
	}
	return s.fakeEntryStore.SetCompleted(ctx, id, text, lang)
}

type expansionStorage struct {
	*fakeStorage
	downloadErr error
	reader      io.ReadCloser
}

func (s *expansionStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.downloadErr != nil {
		return nil, s.downloadErr
	}
	if s.reader != nil {
		return s.reader, nil
	}
	return s.fakeStorage.Download(ctx, key)
}

type expansionReader struct{ closed bool }

func (*expansionReader) Read([]byte) (int, error) { return 0, expansionErr }
func (r *expansionReader) Close() error           { r.closed = true; return nil }

type expansionQueue struct {
	fakeQueue
	cancel     context.CancelFunc
	dequeueErr error
	enqueueErr error
}

func (q *expansionQueue) Enqueue(ctx context.Context, v any) error {
	_ = q.fakeQueue.Enqueue(ctx, v)
	return q.enqueueErr
}
func (q *expansionQueue) Dequeue(context.Context) ([]byte, error) {
	q.cancel()
	return nil, q.dequeueErr
}

type expansionInsight struct {
	err   error
	calls int
}

func (b *expansionInsight) BuildConnectionInsight(context.Context, uuid.UUID, uuid.UUID, []string) (string, error) {
	b.calls++
	return "A recurring connection", b.err
}

type expansionNudge struct{}

func (expansionNudge) ScheduleMorningNudge(context.Context, uuid.UUID, uuid.UUID, string) error {
	return expansionErr
}

func TestExpansionTranscriptionFailureBoundaries(t *testing.T) {
	for _, tc := range []struct{ stage, want string }{
		{"processing", "set processing"}, {"unclaimed", "not in processable state"}, {"fetch", "fetch entry"}, {"missing", "fetch entry"},
		{"download", "download audio"}, {"empty", "empty transcript"}, {"transcript", "store transcript"},
		{"context", "context build"}, {"analysis", "store analysis"}, {"complete", "complete entry"}, {"crisis complete", "complete crisis entry"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			id, user := uuid.New(), uuid.New()
			f := newWorkerFixture(id, user, "audio/test")
			store := &expansionEntryStore{fakeEntryStore: f.entryStore, stage: tc.stage}
			f.worker.entryRepo = store
			switch tc.stage {
			case "download":
				f.worker.storage = &expansionStorage{fakeStorage: f.storage, downloadErr: expansionErr}
			case "empty":
				f.transcriber.result.Text = ""
			case "context":
				f.context.err = expansionErr
			case "analysis":
				f.analysisStore.err = expansionErr
			case "crisis complete":
				store.stage = "complete"
				f.crisis.result.Detected = true
			}
			err := f.worker.handle(context.Background(), &models.TranscriptionJob{EntryID: id, UserID: user, AudioKey: "audio/test"}, zap.NewNop())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if f.entryStore.setCompleted || f.nudge.calls != 0 {
				t.Fatal("failed pipeline completed or ran success side effects")
			}
			// Audio is deleted as soon as the transcript is stored (ADR-005), so
			// only failures before that point may leave it in storage.
			afterTranscript := map[string]bool{"context": true, "analysis": true, "complete": true, "crisis complete": true}[tc.stage]
			if afterTranscript != (len(f.storage.deleted) == 1) {
				t.Fatalf("stage %q: unexpected audio deletion state %v", tc.stage, f.storage.deleted)
			}
		})
	}
}

func TestExpansionTranscriptionOptionalFailures(t *testing.T) {
	for _, failInsight := range []bool{false, true} {
		t.Run(map[bool]string{false: "insight stored", true: "insight unavailable"}[failInsight], func(t *testing.T) {
			id, user := uuid.New(), uuid.New()
			f := newWorkerFixture(id, user, "audio/test")
			insight := &expansionInsight{}
			if failInsight {
				insight.err = expansionErr
			}
			f.worker.insightBuilder = insight
			f.worker.nudgeSvc = expansionNudge{}
			f.storage.dlErr = expansionErr
			people := &fakePersonMentionStore{err: expansionErr}
			f.worker.personRepo = people
			f.worker.personExtractor = &fakePersonExtractor{out: &models.PersonExtractionOutput{People: []models.ExtractedPerson{{Name: "Alex"}}}}
			err := f.worker.handle(context.Background(), &models.TranscriptionJob{EntryID: id, UserID: user, AudioKey: "audio/test"}, zap.NewNop())
			if err != nil || !f.entryStore.setCompleted || insight.calls != 1 || len(people.upserted) != 1 {
				t.Fatalf("optional failure blocked completion: %v", err)
			}
			want := "A recurring connection"
			if failInsight {
				want = ""
			}
			if f.analysisStore.upserted[0].ConnectionInsight != want {
				t.Fatal("incorrect persisted connection insight")
			}
		})
	}
}

func TestExpansionTranscriptionReenqueueFailure(t *testing.T) {
	id, user := uuid.New(), uuid.New()
	f := newWorkerFixture(id, user, "audio/test")
	f.transcriber.err = expansionErr
	q := &expansionQueue{enqueueErr: expansionErr}
	f.worker.queue = q
	f.worker.processJob(context.Background(), makeJobPayload(t, id, user, "audio/test"))
	if len(q.enqueued) != 1 || q.enqueued[0].(*models.TranscriptionJob).Attempt != 1 || len(f.entryStore.setFailed) != 1 || f.entryStore.setFailed[0] != "re-enqueue failed: injected failure" {
		t.Fatalf("retry failure lost: enqueued=%v failures=%v", q.enqueued, f.entryStore.setFailed)
	}
}

type expansionNoteStore struct {
	*fakeNoteSessionStore
	stage string
}

func (s *expansionNoteStore) SetSessionProcessing(ctx context.Context, id uuid.UUID) (bool, error) {
	if s.stage == "processing" {
		return false, expansionErr
	}
	return s.fakeNoteSessionStore.SetSessionProcessing(ctx, id)
}
func (s *expansionNoteStore) CompleteSessionOCR(ctx context.Context, id uuid.UUID, raw, bullets []byte) error {
	if s.stage == "store" {
		return expansionErr
	}
	return s.fakeNoteSessionStore.CompleteSessionOCR(ctx, id, raw, bullets)
}

type expansionKeyStore struct {
	calls  int
	failAt int
}

func (s *expansionKeyStore) GetWrappedDEK(context.Context, uuid.UUID) ([]byte, error) {
	s.calls++
	if s.calls == s.failAt {
		return nil, expansionErr
	}
	return nil, nil
}
func (*expansionKeyStore) InsertWrappedDEK(_ context.Context, _ uuid.UUID, key []byte) ([]byte, error) {
	return key, nil
}

func TestExpansionOCRFailureBoundaries(t *testing.T) {
	for _, tc := range []struct{ stage, want string }{
		{"processing", "set processing"}, {"busy", "not in processable state"}, {"download", "download image"},
		{"read", "read image"}, {"empty", "image object is empty"}, {"raw encryption", "encrypt raw text"}, {"bullets encryption", "encrypt bullets"}, {"store", "store OCR result"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			user := uuid.New()
			base := &fakeNoteSessionStore{session: pendingNoteSession(user, "notes/test.png")}
			storage := &fakeStorage{content: "image"}
			ocr := &fakeNoteOCRClient{out: &models.NotesOCROutput{RawText: "notes", Bullets: []string{"notes"}}}
			w, _ := newNoteWorkerForTest(t, base, storage, ocr, &fakeQueue{})
			w.repo = &expansionNoteStore{fakeNoteSessionStore: base, stage: tc.stage}
			r := &expansionReader{}
			switch tc.stage {
			case "busy":
				base.session.Status = models.ClientSessionProcessing
			case "download":
				w.storage = &expansionStorage{fakeStorage: storage, downloadErr: expansionErr}
			case "read":
				w.storage = &expansionStorage{fakeStorage: storage, reader: r}
			case "empty":
				storage.content = ""
			case "raw encryption", "bullets encryption":
				failAt := 1
				if tc.stage == "bullets encryption" {
					failAt = 2
				}
				w.cipher = services.NewNotesCipher(make([]byte, 32), &expansionKeyStore{failAt: failAt})
			}
			err := w.handle(context.Background(), &models.NoteOCRJob{SessionID: base.session.ID, TherapistID: user, ImageKey: "notes/test.png"}, zap.NewNop())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if base.session.Status == models.ClientSessionCompleted || len(storage.deleted) != 0 {
				t.Fatal("failure completed session or deleted retry image")
			}
			if tc.stage == "read" && !r.closed {
				t.Fatal("failed reader was not closed")
			}
		})
	}
}

func TestExpansionOCROversizeAndRawFallback(t *testing.T) {
	for _, oversize := range []bool{true, false} {
		t.Run(map[bool]string{true: "oversize", false: "raw fallback"}[oversize], func(t *testing.T) {
			user := uuid.New()
			base := &fakeNoteSessionStore{session: pendingNoteSession(user, "notes/test.webp")}
			storage := &fakeStorage{content: "image", dlErr: expansionErr}
			if oversize {
				storage.content = strings.Repeat("x", services.MaxNoteImageBytes+1)
				storage.dlErr = nil
			}
			ocr := &fakeNoteOCRClient{out: &models.NotesOCROutput{RawText: "  readable notes  "}}
			w, cipher := newNoteWorkerForTest(t, base, storage, ocr, &fakeQueue{})
			job := &models.NoteOCRJob{SessionID: base.session.ID, TherapistID: user, ImageKey: "notes/test.webp"}
			if err := w.handle(context.Background(), job, zap.NewNop()); err != nil {
				t.Fatal(err)
			}
			if oversize {
				if !base.final || ocr.calls != 0 || len(storage.deleted) != 1 || !strings.Contains(base.failures[0], "10 MB") {
					t.Fatal("oversize input did not fail permanently before OCR")
				}
			} else {
				bullets, err := cipher.DecryptBullets(context.Background(), user, base.session.BulletsEnc)
				if err != nil || len(bullets) != 1 || bullets[0] != "readable notes" || base.session.Status != models.ClientSessionCompleted {
					t.Fatalf("raw fallback not stored despite nonfatal delete failure: %v %v", bullets, err)
				}
			}
		})
	}
	for key, want := range map[string]string{"a.png": "image/png", "a.webp": "image/webp", "a.jpg": "image/jpeg", "unknown": "image/jpeg"} {
		if got := contentTypeForKey(key); got != want {
			t.Errorf("%s: got %s, want %s", key, got, want)
		}
	}
}

func TestExpansionOCRMalformedAndReenqueueFailure(t *testing.T) {
	user := uuid.New()
	base := &fakeNoteSessionStore{session: pendingNoteSession(user, "notes/test")}
	w, _ := newNoteWorkerForTest(t, base, &fakeStorage{content: "image"}, &fakeNoteOCRClient{err: expansionErr}, &fakeQueue{})
	q := &expansionQueue{enqueueErr: expansionErr}
	w.queue = q
	bad := []byte("{not-json")
	w.processJob(context.Background(), bad)
	if len(q.dlq) != 1 || string(q.dlq[0]) != string(bad) || base.session.Status != models.ClientSessionPending {
		t.Fatal("malformed OCR job not isolated to DLQ")
	}
	job := &models.NoteOCRJob{SessionID: base.session.ID, TherapistID: user, ImageKey: "notes/test"}
	payload, _ := json.Marshal(job)
	w.processJob(context.Background(), payload)
	if !base.final || len(base.failures) != 2 || base.failures[1] != "re-enqueue failed: injected failure" || len(q.enqueued) != 1 || q.enqueued[0].(*models.NoteOCRJob).Attempt != 1 {
		t.Fatalf("OCR enqueue failure not terminal: %v", base.failures)
	}
}

func TestExpansionWorkerRunCancellation(t *testing.T) {
	for _, kind := range []string{"transcription empty", "transcription dequeue error", "OCR empty", "OCR dequeue error"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			q := &expansionQueue{cancel: cancel}
			if strings.Contains(kind, "error") {
				q.dequeueErr = expansionErr
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				if strings.HasPrefix(kind, "OCR") {
					NewNoteOCRWorker(NoteOCRWorkerDeps{Queue: q, Log: zap.NewNop()}).Run(ctx)
				} else {
					NewTranscriptionWorker(TranscriptionWorkerDeps{Queue: q, Log: zap.NewNop(), Concurrency: 1}).Run(ctx)
				}
			}()
			select {
			case <-done:
			case <-time.After(4 * time.Second):
				t.Fatal("worker did not stop after dequeue canceled context")
			}
		})
	}
}
