package services

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/pkg/queue"
	"github.com/google/uuid"
)

type EntryService struct {
	repo    *repositories.EntryRepository
	storage *StorageService
	queue   *queue.Queue
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
		return nil, fmt.Errorf("entryService.Create: recording exceeds 30-minute limit")
	}

	// Verify the client actually uploaded before we create a DB row.
	exists, err := s.storage.Exists(ctx, input.AudioKey)
	if err != nil {
		return nil, fmt.Errorf("entryService.Create: storage check: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("entryService.Create: audio file not found in storage: %s", input.AudioKey)
	}

	// Persist the entry.
	entry, err := s.repo.Create(ctx, userID, input.AudioKey, input.AudioSizeBytes, input.DurationSec, input.Mode)
	if err != nil {
		return nil, fmt.Errorf("entryService.Create: repo: %w", err)
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
		// Entry is created but not queued — the worker can recover these via a reconciler (Phase 2).
		// Log the error and return the entry so the client knows the upload succeeded.
		// Do not roll back: the entry row is the source of truth.
		return entry, fmt.Errorf("entryService.Create: enqueue (entry created, will retry): %w", err)
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
		HasMore:  (page*pageSize) < total,
	}, nil
}
