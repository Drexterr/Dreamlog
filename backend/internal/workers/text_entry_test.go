package workers

import (
	"context"
	"errors"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
	"testing"
)

func TestTypedJournalUsesSafetyPipelineWithoutAudioTranscription(t *testing.T) {
	for _, crisis := range []bool{false, true} {
		id, user := uuid.New(), uuid.New()
		f := newWorkerFixture(id, user, "text/marker")
		content := "A complete typed journal."
		f.entryStore.entry.Transcript = &content
		f.entryStore.entry.DurationSec = 0
		f.transcriber.err = errors.New("must not transcribe typed text")
		f.crisis.result = &services.CrisisResult{Detected: crisis, Response: "safe response"}
		f.worker.processJob(context.Background(), makeJobPayload(t, id, user, "text/marker"))
		if f.entryStore.entry.Status != models.EntryStatusCompleted {
			t.Fatalf("status=%s", f.entryStore.entry.Status)
		}
		if len(f.analysisStore.upserted) != 1 || f.analysisStore.upserted[0].IsCrisis != crisis {
			t.Fatal("wrong safety outcome")
		}
		if len(f.storage.deleted) != 0 {
			t.Fatal("typed journal touched audio storage")
		}
	}
}
