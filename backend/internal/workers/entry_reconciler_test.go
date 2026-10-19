package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeStuckStore struct {
	stuck []repositories.StuckEntry
	err   error
}

func (f *fakeStuckStore) ReclaimStuck(context.Context, time.Duration, time.Duration, int, int) ([]repositories.StuckEntry, error) {
	return f.stuck, f.err
}

// H7: entries stranded by a crash/deploy are requeued with their retry count.
func TestEntryReconciler_RequeuesStuckEntries(t *testing.T) {
	id, user := uuid.New(), uuid.New()
	q := &fakeQueue{}
	r := NewEntryReconciler(&fakeStuckStore{stuck: []repositories.StuckEntry{
		{ID: id, UserID: user, AudioKey: "audio/x", RetryCount: 1, Country: "IN"},
	}}, q, 3, zap.NewNop())

	r.reconcileOnce(context.Background())

	if len(q.enqueued) != 1 {
		t.Fatalf("want 1 requeued job, got %d", len(q.enqueued))
	}
	job := q.enqueued[0].(*models.TranscriptionJob)
	if job.EntryID != id || job.UserID != user || job.Attempt != 1 || job.UserCountry != "IN" {
		t.Errorf("requeued job mismatch: %+v", job)
	}
}

func TestEntryReconciler_StoreErrorIsNonFatal(t *testing.T) {
	q := &fakeQueue{}
	NewEntryReconciler(&fakeStuckStore{err: errors.New("db down")}, q, 3, nil).reconcileOnce(context.Background())
	if len(q.enqueued) != 0 {
		t.Error("nothing must be requeued when the reclaim query fails")
	}
}

// H7: a duplicate job for an entry another worker holds is dropped, not
// counted as a failure (which would revert the live entry to pending).
func TestWorker_DuplicateJobForProcessingEntry_IsDropped(t *testing.T) {
	entryID, userID := uuid.New(), uuid.New()
	f := newWorkerFixture(entryID, userID, "audio/key")
	f.entryStore.entry.Status = models.EntryStatusProcessing

	f.worker.processJob(context.Background(), makeJobPayload(t, entryID, userID, "audio/key"))

	if len(f.entryStore.setRetrying) != 0 || len(f.entryStore.setFailed) != 0 || len(f.queue.enqueued) != 0 {
		t.Error("duplicate job must be dropped without retry/failure side effects")
	}
	if f.entryStore.entry.Status != models.EntryStatusProcessing {
		t.Errorf("live entry status must be untouched, got %s", f.entryStore.entry.Status)
	}
}
