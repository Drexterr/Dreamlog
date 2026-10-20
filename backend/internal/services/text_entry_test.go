package services

import (
	"context"
	"errors"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/google/uuid"
	"testing"
)

type textRepoFake struct {
	entryRepository
	created  bool
	err      error
	conflict bool
}

func (f *textRepoFake) CreateText(_ context.Context, user uuid.UUID, in *models.CreateTextEntryInput) (*models.Entry, bool, error) {
	content := in.Content
	if f.conflict {
		content = "different journal"
	}
	return &models.Entry{ID: models.TextEntryID(user, in.RequestID), UserID: user, AudioKey: "text/marker", Transcript: &content, Mode: in.Mode, Status: models.EntryStatusPending}, f.created, f.err
}

type textQueueFake struct {
	calls int
	job   *models.TranscriptionJob
	err   error
}

func (f *textQueueFake) Enqueue(_ context.Context, value any) error {
	f.calls++
	f.job = value.(*models.TranscriptionJob)
	return f.err
}

func TestCreateTextEntryQueuesOnlyNewEntriesAndSurvivesEnqueueFailure(t *testing.T) {
	for _, tc := range []struct {
		name            string
		created, outage bool
		want            int
	}{
		{"new", true, false, 1}, {"retry", false, false, 0}, {"queue outage recovered by reconciler", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			queue := &textQueueFake{}
			if tc.outage {
				queue.err = errors.New("redis down")
			}
			svc := &EntryService{repo: &textRepoFake{created: tc.created}, queue: queue}
			user := uuid.New()
			input := &models.CreateTextEntryInput{Content: "  my full journal  ", RequestID: uuid.NewString()}
			entry, err := svc.CreateText(context.Background(), user, input, "IN")
			if err != nil || entry == nil {
				t.Fatalf("entry=%v err=%v", entry, err)
			}
			if queue.calls != tc.want {
				t.Fatalf("enqueue calls=%d", queue.calls)
			}
			if tc.want > 0 && (queue.job.UserID != user || queue.job.UserCountry != "IN" || queue.job.EntryID != entry.ID) {
				t.Fatal("wrong job scope")
			}
		})
	}
}

func TestCreateTextEntryRejectsConflictAndPersistenceFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo *textRepoFake
		code int
	}{
		{"conflicting retry", &textRepoFake{conflict: true}, 409},
		{"persistence failure", &textRepoFake{err: errors.New("database unavailable")}, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			queue := &textQueueFake{}
			svc := &EntryService{repo: tc.repo, queue: queue}
			_, err := svc.CreateText(context.Background(), uuid.New(), &models.CreateTextEntryInput{Content: "journal", RequestID: uuid.NewString()}, "")
			apiError, ok := apierr.As(err)
			if !ok || apiError.Code != tc.code {
				t.Fatalf("error=%v", err)
			}
			if queue.calls != 0 {
				t.Fatal("enqueued failed entry")
			}
		})
	}
}
