package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/monitoring"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NoteOCRWorker processes therapist note-photo jobs:
//
//	download image → Claude vision OCR → encrypt → store bullets → delete image
//
// Deliberately NO crisis screening: these are a professional's clinical
// records about a client, not a person journaling their own state. Showing
// the treating therapist hotline resources for their own notes would be
// wrong, and flagging their records as crises would pollute nothing useful.
type NoteOCRWorker struct {
	queue      jobQueue
	repo       noteSessionStore
	storage    audioStorage // same Download/Delete interface as the entry pipeline
	ocr        noteOCRClient
	cipher     *services.NotesCipher
	log        *zap.Logger
	maxRetries int
}

type noteSessionStore interface {
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.ClientSessionRow, error)
	SetSessionProcessing(ctx context.Context, sessionID uuid.UUID) (bool, error)
	CompleteSessionOCR(ctx context.Context, sessionID uuid.UUID, rawTextEnc, bulletsEnc []byte) error
	SetSessionFailed(ctx context.Context, sessionID uuid.UUID, errMsg string, final bool) error
}

type noteOCRClient interface {
	ExtractNotesFromImage(ctx context.Context, imageData []byte, contentType string) (*models.NotesOCROutput, error)
}

type NoteOCRWorkerDeps struct {
	Queue      jobQueue
	Repo       noteSessionStore
	Storage    audioStorage
	OCR        noteOCRClient
	Cipher     *services.NotesCipher
	Log        *zap.Logger
	MaxRetries int
}

func NewNoteOCRWorker(deps NoteOCRWorkerDeps) *NoteOCRWorker {
	return &NoteOCRWorker{
		queue:      deps.Queue,
		repo:       deps.Repo,
		storage:    deps.Storage,
		ocr:        deps.OCR,
		cipher:     deps.Cipher,
		log:        deps.Log,
		maxRetries: deps.MaxRetries,
	}
}

// Run blocks on the notes queue until ctx is cancelled. Jobs are processed
// sequentially - note OCR volume is far below entry volume.
func (w *NoteOCRWorker) Run(ctx context.Context) {
	w.log.Info("note OCR worker starting", zap.Int("max_retries", w.maxRetries))
	for {
		select {
		case <-ctx.Done():
			w.log.Info("note OCR worker shutting down")
			return
		default:
		}

		payload, err := w.queue.Dequeue(ctx)
		if err != nil {
			w.log.Error("note queue dequeue error", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}
		if payload == nil {
			continue
		}
		w.processJob(ctx, payload)
	}
}

func (w *NoteOCRWorker) processJob(ctx context.Context, payload []byte) {
	var job models.NoteOCRJob
	if err := json.Unmarshal(payload, &job); err != nil {
		w.log.Error("note worker: unmarshal job", zap.Error(err))
		_ = w.queue.EnqueueDLQ(ctx, payload, "unmarshal failed: "+err.Error())
		return
	}

	log := w.log.With(
		zap.String("session_id", job.SessionID.String()),
		zap.Int("attempt", job.Attempt),
	)
	log.Info("note worker: processing OCR job")

	start := time.Now()
	if err := w.handle(ctx, &job, log); err != nil {
		log.Warn("note worker: job failed", zap.Error(err), zap.Duration("elapsed", time.Since(start)))

		if job.Attempt >= w.maxRetries-1 {
			log.Error("note worker: max retries reached, moving to DLQ")
			monitoring.CaptureErr(err, map[string]string{
				"job":        "note_ocr",
				"session_id": job.SessionID.String(),
			})
			_ = w.repo.SetSessionFailed(ctx, job.SessionID, "max retries exceeded: "+err.Error(), true)
			_ = w.queue.EnqueueDLQ(ctx, payload, err.Error())
			return
		}

		backoff := time.Duration(math.Pow(2, float64(job.Attempt))) * 2 * time.Second
		time.Sleep(backoff)

		_ = w.repo.SetSessionFailed(ctx, job.SessionID, err.Error(), false)
		job.Attempt++
		if enqErr := w.queue.Enqueue(ctx, &job); enqErr != nil {
			log.Error("note worker: re-enqueue failed", zap.Error(enqErr))
			_ = w.repo.SetSessionFailed(ctx, job.SessionID, "re-enqueue failed: "+enqErr.Error(), true)
		}
		return
	}
	log.Info("note worker: job completed", zap.Duration("elapsed", time.Since(start)))
}

func (w *NoteOCRWorker) handle(ctx context.Context, job *models.NoteOCRJob, log *zap.Logger) error {
	// ── 1. Idempotency gate ─────────────────────────────────────────────────
	transitioned, err := w.repo.SetSessionProcessing(ctx, job.SessionID)
	if err != nil {
		return fmt.Errorf("set processing: %w", err)
	}
	if !transitioned {
		session, _ := w.repo.GetSessionByID(ctx, job.SessionID)
		if session == nil {
			// Session deleted before the job ran - clean up the photo and stop.
			log.Info("note worker: session gone, deleting orphaned image")
			_ = w.storage.Delete(ctx, job.ImageKey)
			return nil
		}
		if session.Status == models.ClientSessionCompleted {
			log.Info("note worker: session already completed, skipping")
			return nil
		}
		return fmt.Errorf("session not in processable state (%s)", session.Status)
	}

	// ── 2. Download the note photo ──────────────────────────────────────────
	log.Info("note worker: downloading note image")
	reader, err := w.storage.Download(ctx, job.ImageKey)
	if err != nil {
		return fmt.Errorf("download image: %w", err)
	}
	defer reader.Close()

	imageData, err := io.ReadAll(io.LimitReader(reader, services.MaxNoteImageBytes+1))
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}
	if len(imageData) > services.MaxNoteImageBytes {
		// Permanent failure - retrying won't shrink the file.
		_ = w.repo.SetSessionFailed(ctx, job.SessionID, "note photo exceeds 10 MB limit", true)
		_ = w.storage.Delete(ctx, job.ImageKey)
		return nil
	}
	if len(imageData) == 0 {
		return fmt.Errorf("image object is empty")
	}

	// ── 3. Claude vision OCR ────────────────────────────────────────────────
	log.Info("note worker: running vision OCR")
	out, err := w.ocr.ExtractNotesFromImage(ctx, imageData, contentTypeForKey(job.ImageKey))
	if err != nil {
		return fmt.Errorf("vision OCR: %w", err)
	}
	if len(out.Bullets) == 0 && strings.TrimSpace(out.RawText) == "" {
		// Nothing readable - permanent, surfaced to the therapist as failed.
		_ = w.repo.SetSessionFailed(ctx, job.SessionID, "no readable notes found in the photo", true)
		_ = w.storage.Delete(ctx, job.ImageKey)
		return nil
	}
	if len(out.Bullets) == 0 {
		out.Bullets = []string{strings.TrimSpace(out.RawText)}
	}

	// ── 4. Encrypt + store ──────────────────────────────────────────────────
	rawEnc, err := w.cipher.EncryptField(ctx, job.TherapistID, out.RawText)
	if err != nil {
		return fmt.Errorf("encrypt raw text: %w", err)
	}
	bulletsEnc, err := w.cipher.EncryptBullets(ctx, job.TherapistID, out.Bullets)
	if err != nil {
		return fmt.Errorf("encrypt bullets: %w", err)
	}
	if err := w.repo.CompleteSessionOCR(ctx, job.SessionID, rawEnc, bulletsEnc); err != nil {
		return fmt.Errorf("store OCR result: %w", err)
	}

	// ── 5. Delete the photo (most sensitive artifact; ADR-005 spirit) ───────
	log.Info("note worker: deleting note image")
	if err := w.storage.Delete(ctx, job.ImageKey); err != nil {
		log.Warn("note worker: delete image failed (non-fatal)", zap.Error(err))
	}

	return nil
}

// contentTypeForKey maps the stored key extension back to a MIME type for the
// vision data URL.
func contentTypeForKey(key string) string {
	switch {
	case strings.HasSuffix(key, ".png"):
		return "image/png"
	case strings.HasSuffix(key, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
