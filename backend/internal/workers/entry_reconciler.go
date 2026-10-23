package workers

import (
	"context"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"go.uber.org/zap"
)

// Stuck-entry thresholds. An entry still 'pending' after stalePendingAge was
// never picked up (lost enqueue, Redis flush); one still 'processing' after
// staleProcessingAge lost its worker (crash, deploy, OOM). Both are far above
// normal pipeline latency, so live jobs are never touched.
const (
	reconcileInterval  = 5 * time.Minute
	stalePendingAge    = 15 * time.Minute
	staleProcessingAge = 30 * time.Minute
	reconcileBatch     = 100
)

type stuckEntryStore interface {
	ReclaimStuck(ctx context.Context, pendingAge, processingAge time.Duration, maxRetries, limit int, includePending bool) ([]repositories.StuckEntry, error)
}

// reconcileQueue is the job queue plus its depth.
type reconcileQueue interface {
	Enqueue(ctx context.Context, v any) error
	Len(ctx context.Context) (int64, error)
}

// EntryReconciler requeues entries whose job was lost, so no entry spins in
// 'pending'/'processing' forever (ADR-004: a crashed worker loses nothing).
// The reclaim is an atomic UPDATE … RETURNING, so N worker replicas running
// this concurrently never requeue the same entry twice per cycle.
type EntryReconciler struct {
	store      stuckEntryStore
	queue      reconcileQueue
	maxRetries int
	log        *zap.Logger
}

func NewEntryReconciler(store stuckEntryStore, queue reconcileQueue, maxRetries int, log *zap.Logger) *EntryReconciler {
	if log == nil {
		log = zap.NewNop()
	}
	return &EntryReconciler{store: store, queue: queue, maxRetries: maxRetries, log: log}
}

func (r *EntryReconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	for {
		r.reconcileOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *EntryReconciler) reconcileOnce(ctx context.Context) {
	// A stale 'pending' row is most likely still waiting in a backlogged
	// queue: requeueing it would add a duplicate job. Only while the queue is
	// empty can a pending row's job really be lost (N9). An unknown depth
	// leaves pending rows for the next cycle.
	depth, depthErr := r.queue.Len(ctx)
	if depthErr != nil {
		r.log.Warn("reconciler: queue depth unavailable", zap.Error(depthErr))
	} else {
		r.log.Info("reconciler: cycle", zap.Int64("queue_depth", depth))
	}
	includePending := depthErr == nil && depth == 0

	stuck, err := r.store.ReclaimStuck(ctx, stalePendingAge, staleProcessingAge, r.maxRetries, reconcileBatch, includePending)
	if err != nil {
		r.log.Error("reconciler: reclaim stuck entries failed", zap.Error(err))
		return
	}
	for _, e := range stuck {
		job := &models.TranscriptionJob{
			EntryID:     e.ID,
			UserID:      e.UserID,
			AudioKey:    e.AudioKey,
			Attempt:     e.RetryCount,
			UserCountry: e.Country,
		}
		if err := r.queue.Enqueue(ctx, job); err != nil {
			// The entry stays 'pending' with a fresh updated_at; the next
			// cycle after stalePendingAge picks it up again.
			r.log.Error("reconciler: requeue failed", zap.String("entry_id", e.ID.String()), zap.Error(err))
			continue
		}
		r.log.Warn("reconciler: requeued stuck entry", zap.String("entry_id", e.ID.String()), zap.Int("attempt", e.RetryCount))
	}
}
