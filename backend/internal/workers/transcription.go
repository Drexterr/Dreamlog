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
	"github.com/dreamlog/backend/pkg/monitoring"
	"go.uber.org/zap"
)

// TranscriptionWorker is the main pipeline worker.
// Phase 2 pipeline:
//   Audio download → Whisper → Store transcript → Delete audio → Crisis screen
//   → Context build → Claude analysis → Store analysis → Schedule nudge
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
	insightBuilder  connectionInsightBuilder
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
	InsightBuilder  connectionInsightBuilder // optional; nil = no connection insights
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
		insightBuilder:  deps.InsightBuilder,
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
			// Report a job panic to Sentry before it crashes the process.
			defer monitoring.RecoverRepanic()
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

		// Final status writes must survive a shutdown-cancelled ctx, otherwise a
		// deploy leaves the entry stuck in 'processing' forever.
		wctx := context.WithoutCancel(ctx)

		if job.Attempt >= w.maxRetries-1 {
			log.Error("worker: max retries reached, moving to DLQ")
			monitoring.CaptureErr(err, map[string]string{
				"job":      "entry_pipeline",
				"entry_id": job.EntryID.String(),
			})
			_ = w.entryRepo.SetFailed(wctx, job.EntryID, "max retries exceeded: "+err.Error())
			_ = w.queue.EnqueueDLQ(wctx, payload, err.Error())
			return
		}

		backoff := time.Duration(math.Pow(2, float64(job.Attempt))) * 2 * time.Second
		log.Info("worker: retrying after backoff", zap.Duration("backoff", backoff))
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
		}

		// Put the entry back to 'pending' BEFORE re-enqueueing, so whichever
		// worker picks the job up can claim it. 'failed' is reserved for the
		// terminal DLQ path - the client stops polling on it.
		if serr := w.entryRepo.SetRetrying(wctx, job.EntryID, err.Error()); serr != nil {
			log.Error("worker: failed to mark entry for retry", zap.Error(serr))
		}
		job.Attempt++
		if enqErr := w.queue.Enqueue(wctx, &job); enqErr != nil {
			log.Error("worker: failed to re-enqueue", zap.Error(enqErr))
			_ = w.entryRepo.SetFailed(wctx, job.EntryID, "re-enqueue failed: "+enqErr.Error())
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

	// ── 3. Transcript: reuse from a previous attempt, or Whisper it ────────
	var transcript, language string
	if entry.Transcript != nil && *entry.Transcript != "" {
		// A previous attempt already transcribed (and deleted) the audio.
		// Don't re-download or pay for Whisper again.
		log.Info("worker: reusing transcript from previous attempt")
		transcript = *entry.Transcript
		if entry.Language != nil {
			language = *entry.Language
		}
	} else {
		log.Info("worker: downloading audio")
		audioReader, err := w.storage.Download(ctx, job.AudioKey)
		if err != nil {
			return fmt.Errorf("download audio: %w", err)
		}
		log.Info("worker: transcribing via whisper")
		whisperResult, err := w.transcriber.Transcribe(ctx, audioReader, job.EntryID.String()+".m4a")
		audioReader.Close()
		if err != nil {
			return fmt.Errorf("whisper: %w", err)
		}
		if whisperResult.Text == "" {
			return fmt.Errorf("whisper returned empty transcript")
		}
		transcript, language = whisperResult.Text, whisperResult.Language

		// Persist transcript immediately - before AI analysis.
		if err := w.entryRepo.SetTranscript(ctx, job.EntryID, transcript, language); err != nil {
			return fmt.Errorf("store transcript: %w", err)
		}

		// ADR-005: audio is deleted as soon as the transcript is safely stored -
		// before crisis screening or analysis, so no later failure (crash, DLQ,
		// crisis branch) can leave the recording behind.
		log.Info("worker: deleting audio")
		if err := w.storage.Delete(ctx, job.AudioKey); err != nil {
			log.Warn("worker: delete audio failed (non-fatal)", zap.Error(err))
		}
	}

	// ── 4. Crisis detection ─────────────────────────────────────────────────
	log.Info("worker: crisis screening")
	crisis, err := w.crisisDetector.Screen(ctx, transcript, job.UserCountry)
	if err != nil {
		// ADR-002 fail-safe: if the screener errors, uncertain = crisis.
		log.Warn("worker: crisis detector error, failing safe (treating as crisis)", zap.Error(err))
		crisis = &services.CrisisResult{Detected: true, Response: services.CrisisResponse(job.UserCountry)}
	}

	if crisis.Detected {
		log.Warn("worker: crisis detected, storing safe response")
		return w.storeCrisis(ctx, job, transcript, language, crisis.Response)
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
	if claudeOut.Crisis {
		// The analysis model's SAFETY OVERRIDE fired even though the screener
		// didn't - this is a crisis, never a blank normal reflection (ADR-002).
		log.Warn("worker: analysis model flagged crisis, storing safe response")
		return w.storeCrisis(ctx, job, transcript, language, services.CrisisResponse(job.UserCountry))
	}

	// ── 5b. Connection insight (cross-entry pattern) - non-fatal ────────────
	// Deterministic; only fires when a topic recurs, so the reward stays variable.
	connectionInsight := ""
	if w.insightBuilder != nil && len(claudeOut.Topics) > 0 {
		insight, err := w.insightBuilder.BuildConnectionInsight(ctx, job.UserID, job.EntryID, claudeOut.Topics)
		if err != nil {
			log.Warn("worker: connection insight failed (non-fatal)", zap.Error(err))
		} else {
			connectionInsight = insight
		}
	}

	// ── 6. Persist analysis ─────────────────────────────────────────────────
	analysis := &models.EntryAnalysis{
		EntryID:           job.EntryID,
		MoodScore:         claudeOut.MoodScore,
		EmotionalTone:     claudeOut.EmotionalTone,
		Topics:            claudeOut.Topics,
		KeyQuotes:         claudeOut.KeyQuotes,
		Summary:           claudeOut.Summary,
		Reflection:        claudeOut.Reflection,
		MorningNudge:      claudeOut.MorningNudge,
		IsCrisis:          false,
		ConnectionInsight: connectionInsight,
		DreamSymbols:      claudeOut.DreamSymbols,
		DreamType:         claudeOut.DreamType,
		PsychologicalLens: claudeOut.PsychologicalLens,
		VedicLens:         claudeOut.VedicLens,
	}
	if _, err := w.analysisRepo.Upsert(ctx, job.EntryID, analysis); err != nil {
		return fmt.Errorf("store analysis: %w", err)
	}
	if err := w.entryRepo.SetCompleted(ctx, job.EntryID, transcript, language); err != nil {
		return fmt.Errorf("complete entry: %w", err)
	}

	// ── 7. Schedule morning nudge ───────────────────────────────────────────
	if claudeOut.MorningNudge != "" {
		if err := w.nudgeSvc.ScheduleMorningNudge(ctx, job.UserID, job.EntryID, claudeOut.MorningNudge); err != nil {
			// Non-fatal.
			log.Warn("worker: schedule nudge failed", zap.Error(err))
		}
	}

	// ── 8. Extract people (relationship map) - non-fatal ────────────────────
	if w.personExtractor != nil && w.personRepo != nil && transcript != "" {
		extracted, err := w.personExtractor.ExtractPeople(ctx, transcript)
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

// storeCrisis persists the crisis-resource analysis and completes the entry.
// No nudge is scheduled for crisis entries. The audio was already deleted
// right after transcription - ADR-005 makes no exception for crisis entries.
func (w *TranscriptionWorker) storeCrisis(ctx context.Context, job *models.TranscriptionJob, transcript, language, response string) error {
	analysis := &models.EntryAnalysis{
		EntryID:    job.EntryID,
		IsCrisis:   true,
		Reflection: response,
		Summary:    "Entry flagged for immediate support.",
		MoodScore:  10,
		Topics:     []string{},
		KeyQuotes:  []string{},
	}
	if _, err := w.analysisRepo.Upsert(ctx, job.EntryID, analysis); err != nil {
		return fmt.Errorf("store crisis analysis: %w", err)
	}
	if err := w.entryRepo.SetCompleted(ctx, job.EntryID, transcript, language); err != nil {
		return fmt.Errorf("complete crisis entry: %w", err)
	}
	return nil
}
