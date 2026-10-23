package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

// fakeDLQList mirrors queue.Queue's DLQ list: index 0 is the newest element.
type fakeDLQList struct {
	items      []string
	enqueued   []any
	enqueueErr error
	stealOnRm  bool // simulate a concurrent claim: LREM finds nothing
	restored   int
}

func (f *fakeDLQList) DLQRange(_ context.Context, start, stop int64) ([]string, error) {
	n := int64(len(f.items))
	if stop < 0 || stop >= n {
		stop = n - 1
	}
	if start >= n || start > stop {
		return nil, nil
	}
	return append([]string(nil), f.items[start:stop+1]...), nil
}

func (f *fakeDLQList) DLQLen(context.Context) (int64, error) { return int64(len(f.items)), nil }

func (f *fakeDLQList) DLQRemove(_ context.Context, raw string) (bool, error) {
	if f.stealOnRm {
		return false, nil
	}
	for i, it := range f.items {
		if it == raw {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeDLQList) DLQRestore(_ context.Context, raw string) error {
	f.restored++
	f.items = append([]string{raw}, f.items...)
	return nil
}

func (f *fakeDLQList) Enqueue(_ context.Context, v any) error {
	if f.enqueueErr != nil {
		return f.enqueueErr
	}
	f.enqueued = append(f.enqueued, v)
	return nil
}

// pushDead appends an older dead letter (the list is newest first).
func (f *fakeDLQList) pushDead(t *testing.T, payload any, errMsg string, failedAt time.Time) {
	t.Helper()
	p, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"payload": json.RawMessage(p), "error": errMsg, "failed_at": failedAt})
	f.items = append(f.items, string(raw))
}

type fakeDLQEntryRepo struct {
	entries  map[uuid.UUID]*models.Entry
	resetErr error
	resets   int
}

func (r *fakeDLQEntryRepo) GetByIDInternal(_ context.Context, id uuid.UUID) (*models.Entry, error) {
	return r.entries[id], nil
}

func (r *fakeDLQEntryRepo) ResetForRequeue(_ context.Context, id uuid.UUID) (bool, error) {
	if r.resetErr != nil {
		return false, r.resetErr
	}
	e := r.entries[id]
	if e == nil || e.Status != models.EntryStatusFailed {
		return false, nil
	}
	r.resets++
	e.Status, e.RetryCount, e.ErrorMsg = models.EntryStatusPending, 0, nil
	return true, nil
}

type fakeDLQNotesRepo struct {
	sessions map[uuid.UUID]*models.ClientSessionRow
	resets   int
}

func (r *fakeDLQNotesRepo) GetSessionByID(_ context.Context, id uuid.UUID) (*models.ClientSessionRow, error) {
	return r.sessions[id], nil
}

func (r *fakeDLQNotesRepo) ResetSessionForRequeue(_ context.Context, id uuid.UUID) (bool, error) {
	s := r.sessions[id]
	if s == nil || s.Status != models.ClientSessionFailed {
		return false, nil
	}
	r.resets++
	s.Status = models.ClientSessionPending
	return true, nil
}

type fakeObjects map[string]bool

func (f fakeObjects) Exists(_ context.Context, key string) (bool, error) { return f[key], nil }

type dlqFixture struct {
	svc     *DLQAdminService
	entries *fakeDLQList
	notes   *fakeDLQList
	entryDB *fakeDLQEntryRepo
	notesDB *fakeDLQNotesRepo
	objects fakeObjects
}

func newDLQFixture() *dlqFixture {
	f := &dlqFixture{
		entries: &fakeDLQList{},
		notes:   &fakeDLQList{},
		entryDB: &fakeDLQEntryRepo{entries: map[uuid.UUID]*models.Entry{}},
		notesDB: &fakeDLQNotesRepo{sessions: map[uuid.UUID]*models.ClientSessionRow{}},
		objects: fakeObjects{},
	}
	f.svc = NewDLQAdminService(f.entries, f.notes, f.entryDB, f.notesDB, f.objects, nil)
	return f
}

const secretTranscript = "SECRET-TRANSCRIPT-never-listed"

// failedEntry registers a failed entry and its dead letter; withTranscript
// controls whether a transcript was stored before the failure.
func (f *dlqFixture) failedEntry(t *testing.T, withTranscript bool) *models.Entry {
	t.Helper()
	e := &models.Entry{
		ID: uuid.New(), UserID: uuid.New(),
		AudioKey: "audio/" + uuid.NewString() + "/clip.m4a",
		Status:   models.EntryStatusFailed, RetryCount: 3,
	}
	if withTranscript {
		tr := secretTranscript
		e.Transcript = &tr
	}
	f.entryDB.entries[e.ID] = e
	f.entries.pushDead(t, models.TranscriptionJob{EntryID: e.ID, AudioKey: e.AudioKey, UserID: e.UserID, UserCountry: "IN", Attempt: 2},
		"max retries exceeded: claude: timeout", time.Now())
	return e
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestDLQAdmin_List_PaginatesNewestFirst_PrivacySafe(t *testing.T) {
	f := newDLQFixture()
	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		ids = append(ids, f.failedEntry(t, true).ID)
	}

	items, total, err := f.svc.List(context.Background(), DLQEntries, 1, 2)
	if err != nil || total != 3 || len(items) != 2 {
		t.Fatalf("page 1: %d items total=%d err=%v", len(items), total, err)
	}
	if *items[0].EntryID != ids[0] || items[0].ID != ids[0].String() || items[0].Attempt != 2 || items[0].FailedAt == nil {
		t.Fatalf("item: %+v", items[0])
	}
	page2, _, _ := f.svc.List(context.Background(), DLQEntries, 2, 2)
	if len(page2) != 1 || *page2[0].EntryID != ids[2] {
		t.Fatalf("page 2: %+v", page2)
	}
	empty, _, _ := f.svc.List(context.Background(), DLQEntries, 5, 2)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("past the end must be an empty list, got %+v", empty)
	}

	raw, _ := json.Marshal(items)
	for _, leak := range []string{secretTranscript, "audio/", "user_id", "payload"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("listing leaks %q: %s", leak, raw)
		}
	}
}

func TestDLQAdmin_List_UnparseableAndTruncatedErrors(t *testing.T) {
	f := newDLQFixture()
	f.entries.items = append(f.entries.items, "not json at all")
	f.entries.pushDead(t, map[string]string{"weird": "job"}, strings.Repeat("x", 1000), time.Now())

	items, _, err := f.svc.List(context.Background(), DLQEntries, 1, 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v %v", items, err)
	}
	for _, it := range items {
		if !it.Unparseable || !strings.HasPrefix(it.ID, "raw-") || it.EntryID != nil {
			t.Fatalf("unparseable item: %+v", it)
		}
	}
	if items[0].ID == items[1].ID {
		t.Fatal("distinct raw items need distinct ids")
	}
	if n := len([]rune(items[1].Error)); n > dlqMaxErrorLen+1 {
		t.Fatalf("error not truncated: %d runes", n)
	}
}

func TestDLQAdmin_UnknownQueue(t *testing.T) {
	f := newDLQFixture()
	if _, _, err := f.svc.List(context.Background(), "payments", 1, 10); !errors.Is(err, ErrDLQUnknownQueue) {
		t.Fatalf("list: %v", err)
	}
	if err := f.svc.Requeue(context.Background(), "payments", uuid.NewString()); !errors.Is(err, ErrDLQUnknownQueue) {
		t.Fatalf("requeue: %v", err)
	}
	if err := f.svc.Discard(context.Background(), "", uuid.NewString()); !errors.Is(err, ErrDLQUnknownQueue) {
		t.Fatalf("discard: %v", err)
	}
}

// ── Requeue (entries) ─────────────────────────────────────────────────────────

func TestDLQAdmin_Requeue_EntryWithTranscript(t *testing.T) {
	f := newDLQFixture()
	e := f.failedEntry(t, true)

	if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); err != nil {
		t.Fatal(err)
	}
	if e.Status != models.EntryStatusPending || e.RetryCount != 0 || e.ErrorMsg != nil {
		t.Fatalf("entry not reset: %+v", e)
	}
	if len(f.entries.items) != 0 {
		t.Fatal("the dead letter must be removed")
	}
	if len(f.entries.enqueued) != 1 {
		t.Fatalf("enqueued %d jobs", len(f.entries.enqueued))
	}
	job := f.entries.enqueued[0].(*models.TranscriptionJob)
	if job.EntryID != e.ID || job.UserID != e.UserID || job.AudioKey != e.AudioKey || job.Attempt != 0 || job.UserCountry != "IN" {
		t.Fatalf("fresh job: %+v", job)
	}
}

func TestDLQAdmin_Requeue_NoTranscript_AudioStillThere(t *testing.T) {
	f := newDLQFixture()
	e := f.failedEntry(t, false)
	f.objects[e.AudioKey] = true
	if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); err != nil {
		t.Fatal(err)
	}
	if len(f.entries.enqueued) != 1 {
		t.Fatal("job not enqueued")
	}
}

func TestDLQAdmin_Requeue_NoTranscript_AudioDeleted_Refused(t *testing.T) {
	f := newDLQFixture()
	e := f.failedEntry(t, false) // audio object absent (ADR-005)
	err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String())
	if !errors.Is(err, ErrDLQSourceGone) {
		t.Fatalf("want ErrDLQSourceGone, got %v", err)
	}
	if len(f.entries.items) != 1 || len(f.entries.enqueued) != 0 || e.Status != models.EntryStatusFailed {
		t.Fatal("a refused requeue must change nothing")
	}
}

func TestDLQAdmin_Requeue_RefusesWrongEntryStates(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(f *dlqFixture, e *models.Entry)
		want   error
	}{
		{"entry deleted", func(f *dlqFixture, e *models.Entry) { delete(f.entryDB.entries, e.ID) }, ErrDLQTargetGone},
		{"already completed", func(_ *dlqFixture, e *models.Entry) { e.Status = models.EntryStatusCompleted }, ErrDLQCompleted},
		{"pending again", func(_ *dlqFixture, e *models.Entry) { e.Status = models.EntryStatusPending }, ErrDLQNotFailed},
		{"processing", func(_ *dlqFixture, e *models.Entry) { e.Status = models.EntryStatusProcessing }, ErrDLQNotFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDLQFixture()
			e := f.failedEntry(t, true)
			tc.mutate(f, e)
			if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if len(f.entries.items) != 1 || len(f.entries.enqueued) != 0 {
				t.Fatal("nothing may be claimed or enqueued")
			}
		})
	}
}

func TestDLQAdmin_Requeue_DoubleClickEnqueuesOnce(t *testing.T) {
	f := newDLQFixture()
	e := f.failedEntry(t, true)
	if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); err != nil {
		t.Fatal(err)
	}
	err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String())
	if !errors.Is(err, ErrDLQItemNotFound) {
		t.Fatalf("second requeue: %v", err)
	}
	if len(f.entries.enqueued) != 1 || f.entryDB.resets != 1 {
		t.Fatalf("enqueued=%d resets=%d, want 1/1", len(f.entries.enqueued), f.entryDB.resets)
	}
}

func TestDLQAdmin_Requeue_LostClaimRace_NoSideEffects(t *testing.T) {
	f := newDLQFixture()
	e := f.failedEntry(t, true)
	f.entries.stealOnRm = true // another request removed it between find and LREM
	if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); !errors.Is(err, ErrDLQClaimed) {
		t.Fatalf("want ErrDLQClaimed, got %v", err)
	}
	if f.entryDB.resets != 0 || len(f.entries.enqueued) != 0 {
		t.Fatal("the losing request must not reset or enqueue")
	}
}

func TestDLQAdmin_Requeue_ResetFailureRestoresItem(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		f := newDLQFixture()
		e := f.failedEntry(t, true)
		f.entryDB.resetErr = errors.New("db down")
		if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); err == nil {
			t.Fatal("want error")
		}
		if f.entries.restored != 1 || len(f.entries.items) != 1 || len(f.entries.enqueued) != 0 {
			t.Fatalf("item must be restored: restored=%d items=%d", f.entries.restored, len(f.entries.items))
		}
	})
	t.Run("state changed after the check", func(t *testing.T) {
		f := newDLQFixture()
		e := f.failedEntry(t, true)
		// The conditional reset sees a non-failed row.
		f.svc.entryDB = &racingEntryRepo{fakeDLQEntryRepo: f.entryDB, entry: e}
		if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); !errors.Is(err, ErrDLQNotFailed) {
			t.Fatalf("want ErrDLQNotFailed, got %v", err)
		}
		if f.entries.restored != 1 || len(f.entries.enqueued) != 0 {
			t.Fatal("item must be restored and nothing enqueued")
		}
	})
}

// racingEntryRepo flips the entry to completed right after it is read.
type racingEntryRepo struct {
	*fakeDLQEntryRepo
	entry *models.Entry
}

func (r *racingEntryRepo) GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Entry, error) {
	e, err := r.fakeDLQEntryRepo.GetByIDInternal(ctx, id)
	snapshot := *e
	r.entry.Status = models.EntryStatusCompleted
	return &snapshot, err
}

func TestDLQAdmin_Requeue_EnqueueFailureReported(t *testing.T) {
	f := newDLQFixture()
	e := f.failedEntry(t, true)
	f.entries.enqueueErr = errors.New("redis down")
	if err := f.svc.Requeue(context.Background(), DLQEntries, e.ID.String()); err == nil {
		t.Fatal("want error")
	}
	// The row is pending, so the reconciler picks it up; it is not failed again.
	if e.Status != models.EntryStatusPending {
		t.Fatalf("status %s", e.Status)
	}
}

func TestDLQAdmin_Requeue_UnparseableAndMissing(t *testing.T) {
	f := newDLQFixture()
	f.entries.items = append(f.entries.items, "garbage")
	items, _, _ := f.svc.List(context.Background(), DLQEntries, 1, 10)
	if err := f.svc.Requeue(context.Background(), DLQEntries, items[0].ID); !errors.Is(err, ErrDLQUnparseable) {
		t.Fatalf("unparseable: %v", err)
	}
	if err := f.svc.Requeue(context.Background(), DLQEntries, uuid.NewString()); !errors.Is(err, ErrDLQItemNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

// ── Requeue (notes) ───────────────────────────────────────────────────────────

func failedNote(t *testing.T, f *dlqFixture, imageKey *string) *models.ClientSessionRow {
	t.Helper()
	s := &models.ClientSessionRow{ID: uuid.New(), TherapistID: uuid.New(), Status: models.ClientSessionFailed, ImageKey: imageKey}
	f.notesDB.sessions[s.ID] = s
	key := ""
	if imageKey != nil {
		key = *imageKey
	}
	f.notes.pushDead(t, models.NoteOCRJob{SessionID: s.ID, TherapistID: s.TherapistID, ImageKey: key, Attempt: 2}, "vision OCR: timeout", time.Now())
	return s
}

func TestDLQAdmin_Notes_ListAndRequeue(t *testing.T) {
	f := newDLQFixture()
	key := "notes/" + uuid.NewString() + "/n.jpg"
	f.objects[key] = true
	s := failedNote(t, f, &key)

	items, _, err := f.svc.List(context.Background(), DLQNotes, 1, 10)
	if err != nil || len(items) != 1 || items[0].SessionID == nil || *items[0].SessionID != s.ID || items[0].EntryID != nil {
		t.Fatalf("notes listing: %+v %v", items, err)
	}
	if raw, _ := json.Marshal(items); strings.Contains(string(raw), "notes/") {
		t.Fatalf("listing leaks the image key: %s", raw)
	}
	if err := f.svc.Requeue(context.Background(), DLQNotes, s.ID.String()); err != nil {
		t.Fatal(err)
	}
	job := f.notes.enqueued[0].(*models.NoteOCRJob)
	if job.SessionID != s.ID || job.TherapistID != s.TherapistID || job.ImageKey != key || job.Attempt != 0 {
		t.Fatalf("notes job: %+v", job)
	}
	if s.Status != models.ClientSessionPending || len(f.entries.enqueued) != 0 {
		t.Fatal("note session must be reset and only the notes queue used")
	}
}

func TestDLQAdmin_Notes_PhotoGone_Refused(t *testing.T) {
	t.Run("no image key", func(t *testing.T) {
		f := newDLQFixture()
		s := failedNote(t, f, nil)
		if err := f.svc.Requeue(context.Background(), DLQNotes, s.ID.String()); !errors.Is(err, ErrDLQSourceGone) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("photo deleted", func(t *testing.T) {
		f := newDLQFixture()
		key := "notes/x/gone.jpg"
		s := failedNote(t, f, &key)
		if err := f.svc.Requeue(context.Background(), DLQNotes, s.ID.String()); !errors.Is(err, ErrDLQSourceGone) {
			t.Fatalf("got %v", err)
		}
		if f.notesDB.resets != 0 || len(f.notes.items) != 1 {
			t.Fatal("refused requeue changed state")
		}
	})
}

// ── Discard ───────────────────────────────────────────────────────────────────

func TestDLQAdmin_Discard(t *testing.T) {
	f := newDLQFixture()
	keep := f.failedEntry(t, true)
	drop := f.failedEntry(t, true)
	if err := f.svc.Discard(context.Background(), DLQEntries, drop.ID.String()); err != nil {
		t.Fatal(err)
	}
	items, _, _ := f.svc.List(context.Background(), DLQEntries, 1, 10)
	if len(items) != 1 || *items[0].EntryID != keep.ID {
		t.Fatalf("wrong item discarded: %+v", items)
	}
	if drop.Status != models.EntryStatusFailed || len(f.entries.enqueued) != 0 {
		t.Fatal("discard must not touch the entry or enqueue anything")
	}
	if err := f.svc.Discard(context.Background(), DLQEntries, drop.ID.String()); !errors.Is(err, ErrDLQItemNotFound) {
		t.Fatalf("second discard: %v", err)
	}

	f.entries.items = append(f.entries.items, "garbage")
	items, _, _ = f.svc.List(context.Background(), DLQEntries, 1, 10)
	if err := f.svc.Discard(context.Background(), DLQEntries, items[1].ID); err != nil {
		t.Fatalf("unparseable items must be discardable: %v", err)
	}

	f.entries.stealOnRm = true
	if err := f.svc.Discard(context.Background(), DLQEntries, keep.ID.String()); !errors.Is(err, ErrDLQClaimed) {
		t.Fatalf("lost race: %v", err)
	}
}
