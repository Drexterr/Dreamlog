package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/dreamlog/backend/pkg/monitoring"
	"github.com/dreamlog/backend/pkg/queue"
	"github.com/google/uuid"
)

type entryRepository interface {
	CreateOnce(context.Context, uuid.UUID, string, int64, float64, models.EntryMode) (*models.Entry, bool, error)
	CreateText(context.Context, uuid.UUID, *models.CreateTextEntryInput) (*models.Entry, bool, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.Entry, error)
	GetByAudioKey(context.Context, uuid.UUID, string) (*models.Entry, error)
	List(context.Context, repositories.ListEntriesOpts) ([]*models.Entry, int, error)
}
type entryJobQueue interface {
	Enqueue(context.Context, any) error
}

type EntryService struct {
	repo    entryRepository
	storage *StorageService
	queue   entryJobQueue
}

func NewEntryService(
	repo *repositories.EntryRepository,
	storage *StorageService,
	q *queue.Queue,
) *EntryService {
	return &EntryService{repo: repo, storage: storage, queue: q}
}

// PresignUpload generates a pre-signed URL for direct client audio upload.
func (s *EntryService) PresignUpload(ctx context.Context, userID uuid.UUID) (*models.PresignResponse, error) {
	uploadURL, key, err := s.storage.PresignUpload(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("entryService.PresignUpload: %w", err)
	}
	return &models.PresignResponse{
		UploadURL: uploadURL,
		AudioKey:  key,
		ExpiresIn: s.storage.PresignExpirySec(),
	}, nil
}

// Create validates the upload exists, writes the entry row, and enqueues a transcription job.
// userCountry is the ISO 3166-1 alpha-2 code from the user's profile; empty string is fine.
func (s *EntryService) Create(ctx context.Context, userID uuid.UUID, input *models.CreateEntryInput, userCountry string) (*models.Entry, error) {
	// Validate max duration.
	if input.DurationSec > models.MaxRecordingSeconds {
		return nil, apierr.BadRequest("recording exceeds 30-minute limit")
	}

	// Enforce audio_key ownership: the key must live under this user's prefix
	// (audio/{userID}/...), the same path our own presign generates. Without
	// this check a client could point an entry at another user's audio key and
	// have its transcript stored under their account.
	if !strings.HasPrefix(input.AudioKey, fmt.Sprintf("audio/%s/", userID)) {
		return nil, apierr.BadRequest("audio_key does not belong to user")
	}

	// Idempotent per audio_key: a client retrying registration after a
	// timeout gets the entry it already created, never a duplicate.
	if existing, err := s.repo.GetByAudioKey(ctx, userID, input.AudioKey); err != nil {
		return nil, apierr.InternalErr("could not register the recording", fmt.Errorf("entryService.Create: lookup: %w", err))
	} else if existing != nil {
		return existing, nil
	}

	// Verify the client actually uploaded before we create a DB row.
	exists, err := s.storage.Exists(ctx, input.AudioKey)
	if err != nil {
		return nil, apierr.InternalErr("could not register the recording", fmt.Errorf("entryService.Create: storage check: %w", err))
	}
	if !exists {
		return nil, apierr.BadRequest("audio file not found in storage - upload it before creating the entry")
	}

	// Persist the entry.
	// A concurrent registration of the same upload may have won the insert:
	// return its entry and leave the job it queued as the only one (N10).
	entry, created, err := s.repo.CreateOnce(ctx, userID, input.AudioKey, input.AudioSizeBytes, input.DurationSec, input.Mode)
	if err != nil {
		return nil, apierr.InternalErr("could not register the recording", fmt.Errorf("entryService.Create: repo: %w", err))
	}
	if !created {
		return entry, nil
	}

	// Enqueue transcription job.
	job := &models.TranscriptionJob{
		EntryID:     entry.ID,
		AudioKey:    entry.AudioKey,
		UserID:      userID,
		UserCountry: userCountry,
		Attempt:     0,
	}
	if err := s.queue.Enqueue(ctx, job); err != nil {
		// The entry row is the source of truth and the upload succeeded, so
		// this is still a success for the client: workers.EntryReconciler
		// requeues entries left pending. Returning an error here made clients
		// re-record and created duplicate entries.
		monitoring.CaptureErr(err, map[string]string{"component": "entry_enqueue", "entry_id": entry.ID.String()})
		return entry, nil
	}

	return entry, nil
}

// CreateText uses the existing safe analysis pipeline without Whisper or storage.
// FindByAudioKey returns the user's entry already registered for an upload,
// or nil. POST /entries resolves retries with it before checking the quota.
func (s *EntryService) FindByAudioKey(ctx context.Context, userID uuid.UUID, audioKey string) (*models.Entry, error) {
	e, err := s.repo.GetByAudioKey(ctx, userID, audioKey)
	if err != nil {
		return nil, apierr.InternalErr("could not register the recording", fmt.Errorf("entryService.FindByAudioKey: %w", err))
	}
	return e, nil
}

func (s *EntryService) CreateText(ctx context.Context, userID uuid.UUID, input *models.CreateTextEntryInput, country string) (*models.Entry, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	input.Language = DetectTextLanguage(input.Content)
	entry, created, err := s.repo.CreateText(ctx, userID, input)
	if err != nil || entry == nil {
		return nil, apierr.InternalErr("could not save journal", err)
	}
	if !input.Matches(entry) {
		return nil, apierr.Conflict("request_id was already used for a different journal")
	}
	if created {
		job := &models.TranscriptionJob{EntryID: entry.ID, UserID: userID, AudioKey: entry.AudioKey, UserCountry: country}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			// The existing stuck-entry reconciler recovers a lost enqueue.
			monitoring.CaptureErr(err, map[string]string{"component": "text_entry_enqueue", "entry_id": entry.ID.String()})
		}
	}
	return entry, nil
}

// Get returns a single entry, enforcing user ownership.
func (s *EntryService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Entry, error) {
	entry, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("entryService.Get: %w", err)
	}
	return entry, nil
}

// List returns paginated entries for a user.
func (s *EntryService) List(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.ListEntriesResponse, error) {
	entries, total, err := s.repo.List(ctx, repositories.ListEntriesOpts{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("entryService.List: %w", err)
	}
	if entries == nil {
		entries = []*models.Entry{}
	}
	return &models.ListEntriesResponse{
		Entries:  entries,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  (page * pageSize) < total,
	}, nil
}
