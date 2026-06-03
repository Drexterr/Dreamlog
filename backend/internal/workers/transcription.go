package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	"go.uber.org/zap"
)

// TranscriptionWorker is the main pipeline worker.
// Phase 2 pipeline:
//   Audio download → Whisper → Crisis screen → Context build → Claude analysis
//   → Store analysis → Schedule nudge → Delete audio
type TranscriptionWorker struct {
	queue           jobQueue
	entryRepo       entryStore
	analysisRepo    analysisStore
	transcriber     audioTranscriber
	crisisDetector  crisisScreener
	contextBuilder  contextAssembler
	claude          aiAnalyzer
	nudgeSvc        nudgeScheduler
	storage         audioStorage
	personExtractor personExtractor
	personRepo      personMentionStore
	log             *zap.Logger
	maxRetries      int
	concurrency     int
}

type TranscriptionWorkerDeps struct {
	Queue           jobQueue
	EntryRepo       entryStore
	AnalysisRepo    analysisStore
	NudgeRepo       *repositories.NudgeRepository // kept for backward compat with cmd/worker; not stored on struct
	Transcriber     audioTranscriber
	CrisisDetector  crisisScreener
	ContextBuilder  contextAssembler
	Claude          aiAnalyzer
	NudgeSvc        nudgeScheduler
	Storage         audioStorage
	PersonExtractor personExtractor
	PersonRepo      personMentionStore
	Log             *zap.Logger
	MaxRetries      int
	Concurrency     int
}

func NewTranscriptionWorker(deps TranscriptionWorkerDeps) *TranscriptionWorker {
	return &TranscriptionWorker{
		queue:           deps.Queue,
		entryRepo:       deps.EntryRepo,
		analysisRepo:    deps.AnalysisRepo,
		transcriber:     deps.Transcriber,
		crisisDetector:  deps.CrisisDetector,
		contextBuilder:  deps.ContextBuilder,
		claude:          deps.Claude,
		nudgeSvc:        deps.NudgeSvc,
		storage:         deps.Storage,
		personExtractor: deps.PersonExtractor,
		personRepo:      deps.PersonRepo,
		log:             deps.Log,
		maxRetries:      deps.MaxRetries,
		concurrency:     deps.Concurrency,
	}
}

func (w *TranscriptionWorker) Run(ctx context.Context) {
	w.log.Info("transcription worker starting",
		zap.Int("concurrency", w.concurrency),
		zap.Int("max_retries", w.maxRetries),
	)

	sem := make(chan struct{}, w.concurrency)
	for {
		select {
		case <-ctx.Done():
			w.log.Info("transcription worker shutting down")
			return
		default:
		}

		payload, err := w.queue.Dequeue(ctx)
		if err != nil {
			w.log.Error("queue dequeue error", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}
		if payload == nil {
			continue
		}

		sem <- struct{}{}
		go func(p []byte) {
			defer func() { <-sem }()
			w.processJob(ctx, p)
		}(payload)
	}
}

func (w *TranscriptionWorker) processJob(ctx context.Context, payload []byte) {
	var job models.TranscriptionJob
	if err := json.Unmarshal(payload, &job); err != nil {
		w.log.Error("worker: unmarshal job", zap.Error(err))
		_ = w.queue.EnqueueDLQ(ctx, payload, "unmarshal failed: "+err.Error())
		return
	}

	log := w.log.With(
		zap.String("entry_id", job.EntryID.String()),
		zap.Int("attempt", job.Attempt),
	)
	log.Info("worker: processing job")

	start := time.Now()
	if err := w.handle(ctx, &job, log); err != nil {
		log.Warn("worker: job failed", zap.Error(err), zap.Duration("elapsed", time.Since(start)))

		if job.Attempt >= w.maxRetries-1 {
			log.Error("worker: max retries reached, moving to DLQ")
			_ = w.entryRepo.SetFailed(ctx, job.EntryID, "max retries exceeded: "+err.Error())
			_ = w.queue.EnqueueDLQ(ctx, payload, err.Error())
			return
		}

		backoff := time.Duration(math.Pow(2, float64(job.Attempt))) * 2 * time.Second
		log.Info("worker: retrying after backoff", zap.Duration("backoff", backoff))
		time.Sleep(backoff)

		job.Attempt++
		if enqErr := w.queue.Enqueue(ctx, &job); enqErr != nil {
			log.Error("worker: failed to re-enqueue", zap.Error(enqErr))
			_ = w.entryRepo.SetFailed(ctx, job.EntryID, "re-enqueue failed: "+enqErr.Error())
		} else {
			_ = w.entryRepo.SetFailed(ctx, job.EntryID, err.Error())
		}
		return
	}
	log.Info("worker: job completed", zap.Duration("elapsed", time.Since(start)))
}

func (w *TranscriptionWorker) handle(ctx context.Context, job *models.TranscriptionJob, log *zap.Logger) error {
	// ── 1. Idempotency check ────────────────────────────────────────────────
	transitioned, err := w.entryRepo.SetProcessing(ctx, job.EntryID)
	if err != nil {
		return fmt.Errorf("set processing: %w", err)
	}
	if !transitioned {
		entry, _ := w.entryRepo.GetByIDInternal(ctx, job.EntryID)
		if entry != nil && entry.Status == models.EntryStatusCompleted {
			log.Info("worker: entry already completed, skipping")
			return nil
		}
		return fmt.Errorf("entry not in processable state")
	}

	// ── 2. Fetch entry + validate duration ─────────────────────────────────
	entry, err := w.entryRepo.GetByIDInternal(ctx, job.EntryID)
	if err != nil || entry == nil {
		return fmt.Errorf("fetch entry: %w", err)
	}
	if entry.DurationSec > models.MaxRecordingSeconds {
		return fmt.Errorf("audio exceeds 30-minute limit")
	}

	// ── 3. Download + Whisper transcription ────────────────────────────────
	log.Info("worker: downloading audio")
	audioReader, err := w.storage.Download(ctx, job.AudioKey)
	if err != nil {
		return fmt.Errorf("download audio: %w", err)
	}
	defer audioReader.Close()

	log.Info("worker: transcribing via whisper")
	whisperResult, err := w.transcriber.Transcribe(ctx, audioReader, job.EntryID.String()+".aac")
	if err != nil {
		return fmt.Errorf("whisper: %w", err)
	}
	if whisperResult.Text == "" {
		return fmt.Errorf("whisper returned empty transcript")
	}

	// Persist transcript immediately — before AI analysis.
	if err := w.entryRepo.SetCompleted(ctx, job.EntryID, whisperResult.Text, whisperResult.Language); err != nil {
		return fmt.Errorf("store transcript: %w", err)
	}

	// ── 4. Crisis detection ─────────────────────────────────────────────────
	log.Info("worker: crisis screening")
	crisis, err := w.crisisDetector.Screen(ctx, whisperResult.Text)
	if err != nil {
		// Non-fatal: log and continue with normal analysis.
		log.Warn("worker: crisis detector error, continuing", zap.Error(err))
		crisis = &services.CrisisResult{Detected: false}
	}

	if crisis.Detected {
		log.Warn("worker: crisis detected, storing safe response")
		analysis := &models.EntryAnalysis{
			EntryID:    job.EntryID,
			IsCrisis:   true,
			Reflection: crisis.Response,
			Summary:    "Entry flagged for immediate support.",
			MoodScore:  10,
		}
		if _, err := w.analysisRepo.Upsert(ctx, job.EntryID, analysis); err != nil {
			log.Error("worker: store crisis analysis", zap.Error(err))
		}
		// No nudge for crisis entries. Skip audio deletion for safety audit trail.
		return nil
	}

	// ── 5. Build context + Claude analysis ─────────────────────────────────
	log.Info("worker: building context")
	ctxInput, err := w.contextBuilder.Build(ctx, job.EntryID, job.UserID)
	if err != nil {
		return fmt.Errorf("context build: %w", err)
	}

	log.Info("worker: calling claude")
	claudeOut, err := w.claude.AnalyzeEntry(ctx, *ctxInput)
	if err != nil {
		return fmt.Errorf("claude analysis: %w", err)
	}

	// ── 6. Persist analysis ─────────────────────────────────────────────────
	analysis := &models.EntryAnalysis{
		EntryID:       job.EntryID,
		MoodScore:     claudeOut.MoodScore,
		EmotionalTone: claudeOut.EmotionalTone,
		Topics:        claudeOut.Topics,
		KeyQuotes:     claudeOut.KeyQuotes,
		Summary:       claudeOut.Summary,
		Reflection:    claudeOut.Reflection,
		MorningNudge:  claudeOut.MorningNudge,
		IsCrisis:          false,
		DreamSymbols:      claudeOut.DreamSymbols,
		DreamType:         claudeOut.DreamType,
		PsychologicalLens: claudeOut.PsychologicalLens,
		VedicLens:         claudeOut.VedicLens,
	}
	if _, err := w.analysisRepo.Upsert(ctx, job.EntryID, analysis); err != nil {
		return fmt.Errorf("store analysis: %w", err)
	}

	// ── 7. Schedule morning nudge ───────────────────────────────────────────
	if claudeOut.MorningNudge != "" {
		if err := w.nudgeSvc.ScheduleMorningNudge(ctx, job.UserID, job.EntryID, claudeOut.MorningNudge); err != nil {
			// Non-fatal.
			log.Warn("worker: schedule nudge failed", zap.Error(err))
		}
	}

	// ── 8. Delete audio from storage ────────────────────────────────────────
	log.Info("worker: deleting audio")
	if err := w.storage.Delete(ctx, job.AudioKey); err != nil {
		log.Warn("worker: delete audio failed (non-fatal)", zap.Error(err))
	}

	// ── 9. Extract people (relationship map) — non-fatal ────────────────────
	if w.personExtractor != nil && w.personRepo != nil && whisperResult.Text != "" {
		extracted, err := w.personExtractor.ExtractPeople(ctx, whisperResult.Text)
		if err != nil {
			log.Warn("worker: person extraction failed (non-fatal)", zap.Error(err))
		} else if len(extracted.People) > 0 {
			if err := w.personRepo.UpsertPersonMentions(ctx, job.UserID, job.EntryID, extracted.People); err != nil {
				log.Warn("worker: store person mentions failed (non-fatal)", zap.Error(err))
			}
		}
	}

	return nil
}
