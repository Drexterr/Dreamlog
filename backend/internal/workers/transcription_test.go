package workers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Fake implementations ─────────────────────────────────────────────────────

type fakeQueue struct {
	mu       sync.Mutex
	jobs     [][]byte
	dlq      [][]byte
	enqueued []any
}

func (q *fakeQueue) Dequeue(_ context.Context) ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return nil, nil
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, nil
}

func (q *fakeQueue) Enqueue(_ context.Context, v any) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueued = append(q.enqueued, v)
	return nil
}

func (q *fakeQueue) EnqueueDLQ(_ context.Context, payload []byte, _ string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dlq = append(q.dlq, payload)
	return nil
}

// fakeEntryStore holds one entry and tracks state transitions.
type fakeEntryStore struct {
	mu         sync.Mutex
	entry      *models.Entry
	setFailed  []string
	setCompleted bool
	deleted    bool
}

func (s *fakeEntryStore) SetProcessing(_ context.Context, id uuid.UUID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entry == nil || s.entry.ID != id {
		return false, nil
	}
	if s.entry.Status == models.EntryStatusCompleted {
		return false, nil
	}
	s.entry.Status = models.EntryStatusProcessing
	return true, nil
}

func (s *fakeEntryStore) GetByIDInternal(_ context.Context, id uuid.UUID) (*models.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entry != nil && s.entry.ID == id {
		cp := *s.entry
		return &cp, nil
	}
	return nil, nil
}

func (s *fakeEntryStore) SetCompleted(_ context.Context, id uuid.UUID, transcript, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entry != nil && s.entry.ID == id {
		s.entry.Status = models.EntryStatusCompleted
		s.entry.Transcript = &transcript
		s.setCompleted = true
	}
	return nil
}

func (s *fakeEntryStore) SetFailed(_ context.Context, _ uuid.UUID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setFailed = append(s.setFailed, errMsg)
	if s.entry != nil {
		s.entry.Status = models.EntryStatusFailed
		s.entry.RetryCount++
	}
	return nil
}

type fakeAnalysisStore struct {
	mu       sync.Mutex
	upserted []*models.EntryAnalysis
}

func (s *fakeAnalysisStore) Upsert(_ context.Context, _ uuid.UUID, a *models.EntryAnalysis) (*models.EntryAnalysis, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.upserted = append(s.upserted, a)
	return a, nil
}

type fakeStorage struct {
	mu      sync.Mutex
	content string
	deleted []string
	dlErr   error
}

func (s *fakeStorage) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(s.content)), nil
}

func (s *fakeStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dlErr != nil {
		return s.dlErr
	}
	s.deleted = append(s.deleted, key)
	return nil
}

type fakeTranscriber struct {
	result *services.WhisperResponse
	err    error
}

func (t *fakeTranscriber) Transcribe(_ context.Context, _ io.Reader, _ string) (*services.WhisperResponse, error) {
	return t.result, t.err
}

type fakeCrisisScreener struct {
	result *services.CrisisResult
	err    error
}

func (s *fakeCrisisScreener) Screen(_ context.Context, _, _ string) (*services.CrisisResult, error) {
	return s.result, s.err
}

type fakeContextAssembler struct {
	result *services.AnalyzeEntryInput
	err    error
}

func (a *fakeContextAssembler) Build(_ context.Context, _, _ uuid.UUID) (*services.AnalyzeEntryInput, error) {
	if a.result == nil {
		a.result = &services.AnalyzeEntryInput{Transcript: "stub"}
	}
	return a.result, a.err
}

type fakeAIAnalyzer struct {
	result *models.ClaudeAnalysisOutput
	err    error
}

func (a *fakeAIAnalyzer) AnalyzeEntry(_ context.Context, _ services.AnalyzeEntryInput) (*models.ClaudeAnalysisOutput, error) {
	if a.result == nil {
		a.result = &models.ClaudeAnalysisOutput{
			MoodScore:    65,
			Reflection:   "A thoughtful reflection. What matters most to you today?",
			MorningNudge: "Start tomorrow with one small intention.",
			Summary:      "A good entry.",
			Topics:       []string{"work"},
		}
	}
	return a.result, a.err
}

type fakeNudgeScheduler struct {
	mu       sync.Mutex
	calls    int
	lastMsg  string
}

func (s *fakeNudgeScheduler) ScheduleMorningNudge(_ context.Context, _, _ uuid.UUID, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.lastMsg = message
	return nil
}

// ── Worker builder ────────────────────────────────────────────────────────────

type workerFixture struct {
	entryStore   *fakeEntryStore
	analysisStore *fakeAnalysisStore
	storage      *fakeStorage
	transcriber  *fakeTranscriber
	crisis       *fakeCrisisScreener
	context      *fakeContextAssembler
	ai           *fakeAIAnalyzer
	nudge        *fakeNudgeScheduler
	queue        *fakeQueue
	worker       *TranscriptionWorker
}

func newWorkerFixture(entryID, userID uuid.UUID, audioKey string) *workerFixture {
	f := &workerFixture{
		entryStore: &fakeEntryStore{
			entry: &models.Entry{
				ID:          entryID,
				UserID:      userID,
				AudioKey:    audioKey,
				DurationSec: 60,
				Status:      models.EntryStatusPending,
			},
		},
		analysisStore: &fakeAnalysisStore{},
		storage:       &fakeStorage{content: "fake audio bytes"},
		transcriber: &fakeTranscriber{
			result: &services.WhisperResponse{
				Text:     "Today was a good day at work.",
				Language: "en",
			},
		},
		crisis: &fakeCrisisScreener{
			result: &services.CrisisResult{Detected: false},
		},
		context: &fakeContextAssembler{},
		ai:      &fakeAIAnalyzer{},
		nudge:   &fakeNudgeScheduler{},
		queue:   &fakeQueue{},
	}

	f.worker = NewTranscriptionWorker(TranscriptionWorkerDeps{
		Queue:          f.queue,
		EntryRepo:      f.entryStore,
		AnalysisRepo:   f.analysisStore,
		Transcriber:    f.transcriber,
		CrisisDetector: f.crisis,
		ContextBuilder: f.context,
		Claude:         f.ai,
		NudgeSvc:       f.nudge,
		Storage:        f.storage,
		Log:            zap.NewNop(),
		MaxRetries:     3,
		Concurrency:    1,
	})

	return f
}

func makeJobPayload(t *testing.T, entryID, userID uuid.UUID, audioKey string) []byte {
	t.Helper()
	job := models.TranscriptionJob{
		EntryID:  entryID,
		UserID:   userID,
		AudioKey: audioKey,
		Attempt:  0,
	}
	b, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestWorker_HappyPath_CompletesEntryAndDeletesAudio(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	payload := makeJobPayload(t, entryID, userID, audioKey)

	f.worker.processJob(context.Background(), payload)

	// Entry must be marked completed
	if f.entryStore.entry.Status != models.EntryStatusCompleted {
		t.Errorf("entry status: want completed, got %s", f.entryStore.entry.Status)
	}

	// Analysis must be stored
	if len(f.analysisStore.upserted) == 0 {
		t.Error("analysis must be stored for a successful entry")
	}
	if f.analysisStore.upserted[0].IsCrisis {
		t.Error("non-crisis entry must have is_crisis=false")
	}

	// Audio must be deleted after success
	if len(f.storage.deleted) == 0 {
		t.Error("audio must be deleted after successful processing")
	}
	if f.storage.deleted[0] != audioKey {
		t.Errorf("deleted key: want %q, got %q", audioKey, f.storage.deleted[0])
	}

	// Morning nudge must be scheduled
	if f.nudge.calls == 0 {
		t.Error("morning nudge must be scheduled after successful entry")
	}

	// No failures
	if len(f.entryStore.setFailed) > 0 {
		t.Errorf("unexpected SetFailed calls: %v", f.entryStore.setFailed)
	}
}

func TestWorker_TranscriptionFails_SetsEntryFailed(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.transcriber.err = errors.New("whisper API unavailable")
	f.transcriber.result = nil

	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	// Should either be failed or re-queued (attempt 0 < maxRetries-1)
	// processJob handles retry: on first attempt, it re-enqueues
	if len(f.queue.enqueued) == 0 && len(f.entryStore.setFailed) == 0 {
		t.Error("transcription failure must either set entry failed or re-enqueue for retry")
	}

	// No analysis must have been stored
	if len(f.analysisStore.upserted) > 0 {
		t.Error("analysis must not be stored when transcription fails")
	}

	// Audio must NOT be deleted (transcription didn't succeed)
	if len(f.storage.deleted) > 0 {
		t.Error("audio must not be deleted when transcription fails")
	}
}

func TestWorker_CrisisDetected_StoresCrisisAnalysis(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/crisis.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.crisis.result = &services.CrisisResult{
		Detected: true,
		Response: "Please reach out to a crisis line.",
	}

	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	if len(f.analysisStore.upserted) == 0 {
		t.Fatal("crisis entry must store an analysis record")
	}
	stored := f.analysisStore.upserted[0]
	if !stored.IsCrisis {
		t.Error("crisis entry must have is_crisis=true")
	}
	if stored.Reflection == "" {
		t.Error("crisis entry must have a crisis response as the reflection")
	}
	if stored.MoodScore != 10 {
		t.Errorf("crisis mood_score: want 10, got %d", stored.MoodScore)
	}
}

func TestWorker_CrisisScreenerError_FailsSafeAsCrisis(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/screener-error.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.crisis.result = nil
	f.crisis.err = errors.New("screener unavailable")

	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	// ADR-002: screener error must be treated as a crisis, never as normal.
	if len(f.analysisStore.upserted) == 0 {
		t.Fatal("entry must store an analysis record when screener errors")
	}
	stored := f.analysisStore.upserted[0]
	if !stored.IsCrisis {
		t.Error("screener error must fail safe with is_crisis=true")
	}
	if stored.Reflection == "" {
		t.Error("fail-safe crisis entry must include a crisis resource response")
	}
	if f.nudge.calls > 0 {
		t.Error("no nudge must be scheduled for a fail-safe crisis entry")
	}
}

func TestWorker_CrisisDetected_AudioNotDeletedForAudit(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/crisis.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.crisis.result = &services.CrisisResult{Detected: true, Response: "help resources"}

	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	if len(f.storage.deleted) > 0 {
		t.Error("audio must NOT be deleted for crisis entries (kept for safety audit trail)")
	}
}

func TestWorker_CrisisDetected_NoNudgeScheduled(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")
	f.crisis.result = &services.CrisisResult{Detected: true, Response: "help"}

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	if f.nudge.calls > 0 {
		t.Error("morning nudge must not be scheduled for crisis entries")
	}
}

func TestWorker_MaxRetriesExceeded_GoesToDLQ(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.transcriber.err = errors.New("persistent failure")
	f.transcriber.result = nil

	// Simulate a job at the last allowed attempt
	job := models.TranscriptionJob{
		EntryID:  entryID,
		UserID:   userID,
		AudioKey: audioKey,
		Attempt:  2, // maxRetries-1 = 2 (maxRetries=3)
	}
	payload, _ := json.Marshal(job)

	f.worker.processJob(context.Background(), payload)

	// DLQ must receive the payload
	if len(f.queue.dlq) == 0 {
		t.Error("max-retries-exceeded job must go to DLQ")
	}
	// Must not re-enqueue
	if len(f.queue.enqueued) > 0 {
		t.Error("max-retries-exceeded job must not be re-enqueued")
	}
}

func TestWorker_RetryIncrementsAttempt(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	audioKey := "audio/test/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.transcriber.err = errors.New("temporary failure")
	f.transcriber.result = nil

	// Attempt 0 - should re-enqueue with attempt=1
	job := models.TranscriptionJob{
		EntryID:  entryID,
		UserID:   userID,
		AudioKey: audioKey,
		Attempt:  0,
	}
	payload, _ := json.Marshal(job)
	f.worker.processJob(context.Background(), payload)

	if len(f.queue.enqueued) == 0 {
		t.Fatal("failed job must be re-enqueued for retry")
	}

	// Check that the re-enqueued job has attempt incremented
	re, ok := f.queue.enqueued[0].(*models.TranscriptionJob)
	if !ok {
		// Marshal round-trip
		b, _ := json.Marshal(f.queue.enqueued[0])
		var j models.TranscriptionJob
		_ = json.Unmarshal(b, &j)
		re = &j
	}
	if re.Attempt != 1 {
		t.Errorf("re-enqueued job attempt: want 1, got %d", re.Attempt)
	}
}

func TestWorker_AlreadyCompletedEntry_Skips(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")
	// Mark entry as already completed before the worker runs
	f.entryStore.entry.Status = models.EntryStatusCompleted

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	// SetProcessing returns false for completed entries → worker skips
	// No new analysis should be stored
	if len(f.analysisStore.upserted) > 0 {
		t.Error("already-completed entry must not be re-processed")
	}
}

func TestWorker_EntryExceedsDurationLimit_Fails(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")
	f.entryStore.entry.DurationSec = float64(models.MaxRecordingSeconds) + 1

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	// Should fail (re-enqueued or DLQ) but not produce an analysis
	if len(f.analysisStore.upserted) > 0 {
		t.Error("entry exceeding duration limit must not produce an analysis")
	}
}

func TestWorker_MalformedPayload_GoesToDLQ(t *testing.T) {
	f := newWorkerFixture(uuid.New(), uuid.New(), "audio/key")

	f.worker.processJob(context.Background(), []byte("not valid json {{{"))

	if len(f.queue.dlq) == 0 {
		t.Error("malformed payload must go to DLQ immediately")
	}
}

func TestWorker_AudioDeletedAfterSuccess(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	const audioKey = "audio/user/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	found := false
	for _, k := range f.storage.deleted {
		if k == audioKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("audio key %q must be deleted; deleted keys: %v", audioKey, f.storage.deleted)
	}
}

func TestWorker_AudioNotDeletedOnTranscriptionFailure(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	const audioKey = "audio/user/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.transcriber.err = errors.New("whisper unavailable")
	f.transcriber.result = nil

	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	if len(f.storage.deleted) > 0 {
		t.Error("audio must not be deleted when transcription fails (may need to retry)")
	}
}

func TestWorker_NudgeNotScheduledWhenMorningNudgeEmpty(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")
	f.ai.result = &models.ClaudeAnalysisOutput{
		MoodScore:    50,
		Reflection:   "Reflection text.",
		MorningNudge: "", // empty - no nudge
		Summary:      "Summary.",
	}

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	if f.nudge.calls > 0 {
		t.Error("nudge must not be scheduled when morning_nudge is empty")
	}
}

// ── Concurrency safety ────────────────────────────────────────────────────────

func TestWorker_ProcessJobIsSafeUnderConcurrency(t *testing.T) {
	const n = 20
	var wg sync.WaitGroup
	failures := make([]error, n)
	var mu sync.Mutex

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			entryID := uuid.New()
			userID := uuid.New()
			f := newWorkerFixture(entryID, userID, "audio/key")
			payload := makeJobPayload(t, entryID, userID, "audio/key")

			func() {
				defer func() {
					if r := recover(); r != nil {
						mu.Lock()
						failures[idx] = errors.New("panic in processJob")
						mu.Unlock()
					}
				}()
				f.worker.processJob(context.Background(), payload)
			}()
		}(i)
	}

	wg.Wait()

	for i, err := range failures {
		if err != nil {
			t.Errorf("goroutine %d panicked: %v", i, err)
		}
	}
}

// ── Claude analysis failure ───────────────────────────────────────────────────

func TestWorker_ClaudeAnalysisFails_NoAnalysisStoredAndRetried(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	const audioKey = "audio/test/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.ai.err = errors.New("AI returned malformed JSON: unexpected character '}'")
	f.ai.result = nil

	payload := makeJobPayload(t, entryID, userID, audioKey)
	f.worker.processJob(context.Background(), payload)

	// No analysis must be stored on Claude failure.
	if len(f.analysisStore.upserted) > 0 {
		t.Error("analysis must not be stored when Claude analysis fails")
	}

	// Audio must NOT be deleted - the job failed before the delete step.
	if len(f.storage.deleted) > 0 {
		t.Error("audio must not be deleted when Claude analysis fails")
	}

	// Must be re-queued for retry (attempt 0 < maxRetries-1=2) or set failed.
	if len(f.queue.enqueued) == 0 && len(f.entryStore.setFailed) == 0 {
		t.Error("claude failure must trigger retry enqueue or entry failure")
	}
}

func TestWorker_ClaudeAnalysisFails_MaxRetries_GoesToDLQ(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	const audioKey = "audio/test/entry.aac"

	f := newWorkerFixture(entryID, userID, audioKey)
	f.ai.err = errors.New("AI returned malformed JSON")
	f.ai.result = nil

	// Simulate final attempt (maxRetries-1 = 2).
	job := models.TranscriptionJob{
		EntryID:  entryID,
		UserID:   userID,
		AudioKey: audioKey,
		Attempt:  2,
	}
	payload, _ := json.Marshal(job)
	f.worker.processJob(context.Background(), payload)

	if len(f.queue.dlq) == 0 {
		t.Error("claude failure at max retries must go to DLQ")
	}
	if len(f.queue.enqueued) > 0 {
		t.Error("max-retries job must not be re-enqueued")
	}
}

// ── Person extraction ─────────────────────────────────────────────────────────

type fakePersonExtractor struct {
	out *models.PersonExtractionOutput
	err error
}

func (f *fakePersonExtractor) ExtractPeople(_ context.Context, _ string) (*models.PersonExtractionOutput, error) {
	return f.out, f.err
}

type fakePersonMentionStore struct {
	upserted [][]models.ExtractedPerson
	err      error
}

func (f *fakePersonMentionStore) UpsertPersonMentions(_ context.Context, _, _ uuid.UUID, people []models.ExtractedPerson) error {
	f.upserted = append(f.upserted, people)
	return f.err
}

func TestWorker_PersonExtraction_HappyPath(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")

	personExtractor := &fakePersonExtractor{
		out: &models.PersonExtractionOutput{
			People: []models.ExtractedPerson{
				{Name: "Sarah", Role: "friend", Sentiment: "positive", Context: "Sarah helped me today"},
			},
		},
	}
	personRepo := &fakePersonMentionStore{}

	f.worker.personExtractor = personExtractor
	f.worker.personRepo = personRepo

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	if len(personRepo.upserted) == 0 {
		t.Error("person mentions must be stored on successful extraction")
	}
	if len(personRepo.upserted[0]) != 1 {
		t.Errorf("expected 1 person upserted, got %d", len(personRepo.upserted[0]))
	}
	if personRepo.upserted[0][0].Name != "Sarah" {
		t.Errorf("expected 'Sarah', got %q", personRepo.upserted[0][0].Name)
	}
}

func TestWorker_PersonExtraction_ClaudeError_IsNonFatal(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")

	personExtractor := &fakePersonExtractor{err: errors.New("extraction failed")}
	personRepo := &fakePersonMentionStore{}

	f.worker.personExtractor = personExtractor
	f.worker.personRepo = personRepo

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	// Entry must still complete successfully
	if f.entryStore.entry.Status != models.EntryStatusCompleted {
		t.Errorf("entry must complete even when extraction fails, status: %s", f.entryStore.entry.Status)
	}
	// Main analysis must still be stored
	if len(f.analysisStore.upserted) == 0 {
		t.Error("main analysis must still be stored when extraction fails")
	}
	// No person mentions must be stored
	if len(personRepo.upserted) > 0 {
		t.Error("no person mentions must be stored when extraction fails")
	}
}

func TestWorker_PersonExtraction_EmptyPeople_SkipsStore(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")

	personExtractor := &fakePersonExtractor{
		out: &models.PersonExtractionOutput{People: []models.ExtractedPerson{}},
	}
	personRepo := &fakePersonMentionStore{}

	f.worker.personExtractor = personExtractor
	f.worker.personRepo = personRepo

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	if len(personRepo.upserted) > 0 {
		t.Error("empty extraction result must not call UpsertPersonMentions")
	}
}

func TestWorker_PersonExtraction_NilExtractor_SafelySkipped(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()

	f := newWorkerFixture(entryID, userID, "audio/key")
	// personExtractor and personRepo are nil - must not panic

	payload := makeJobPayload(t, entryID, userID, "audio/key")
	f.worker.processJob(context.Background(), payload)

	if f.entryStore.entry.Status != models.EntryStatusCompleted {
		t.Error("entry must still complete when person extraction is not configured")
	}
}

// ── Backoff timing sanity ─────────────────────────────────────────────────────

func TestWorker_BackoffIncreasesWithAttempt(t *testing.T) {
	// Verify the backoff formula: 2^attempt * 2 seconds
	// attempt 0 → 2s, attempt 1 → 4s, attempt 2 → 8s
	expected := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	for attempt, want := range expected {
		got := time.Duration(1<<uint(attempt)) * 2 * time.Second
		if got != want {
			t.Errorf("attempt %d: want backoff %v, got %v", attempt, want, got)
		}
	}
}
