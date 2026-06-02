package services

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
	"github.com/google/uuid"
)

type StorageService struct {
	client  *pkgstorage.Client
	cfg     *appconfig.StorageConfig
}

func NewStorageService(client *pkgstorage.Client, cfg *appconfig.StorageConfig) *StorageService {
	return &StorageService{client: client, cfg: cfg}
}

// AudioKey returns the canonical storage key for an audio file.
// Format: audio/{userID}/{entryID}.aac
func AudioKey(userID, entryID uuid.UUID) string {
	return fmt.Sprintf("audio/%s/%s.aac", userID, entryID)
}

// PresignUpload generates a PUT URL and the corresponding object key.
// When STORAGE_PROXY_BASE_URL is set, returns a backend proxy URL instead of a
// direct MinIO presigned URL — used in dev where MinIO isn't reachable from the device.
func (s *StorageService) PresignUpload(ctx context.Context, userID uuid.UUID) (uploadURL, key string, err error) {
	entryID := uuid.New()
	key = AudioKey(userID, entryID)

	if s.cfg.ProxyBaseURL != "" {
		uploadURL = s.cfg.ProxyBaseURL + "/upload?key=" + url.QueryEscape(key)
		return uploadURL, key, nil
	}

	uploadURL, err = s.client.PresignUpload(ctx, key)
	if err != nil {
		return "", "", fmt.Errorf("storage: presign upload: %w", err)
	}
	return uploadURL, key, nil
}

// Upload streams audio data to storage. Used by the backend upload proxy handler.
func (s *StorageService) Upload(ctx context.Context, key string, body io.Reader) error {
	return s.client.Upload(ctx, key, "audio/aac", body)
}

// PresignDownload generates a temporary GET URL for a stored audio file.
func (s *StorageService) PresignDownload(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignDownload(ctx, key, expiry)
	if err != nil {
		return "", fmt.Errorf("storage: presign download: %w", err)
	}
	return url, nil
}

// Exists checks whether the audio object exists in storage.
// Used to validate the client actually uploaded before creating an entry.
func (s *StorageService) Exists(ctx context.Context, key string) (bool, error) {
	return s.client.Exists(ctx, key)
}

// Delete removes an audio file from storage. Called after successful transcription.
func (s *StorageService) Delete(ctx context.Context, key string) error {
	if err := s.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	return nil
}

// PresignExpirySec returns the upload URL expiry in seconds for API responses.
func (s *StorageService) PresignExpirySec() int {
	return int(s.cfg.PresignExpiry.Seconds())
}

// PresignPut generates a PUT URL for an arbitrary key (used for therapy voice uploads).
// Returns (uploadURL, audioKey, error). The key is the filename passed in, prefixed with "therapy/".
func (s *StorageService) PresignPut(ctx context.Context, filename, contentType string, expiry time.Duration) (uploadURL, audioKey string, err error) {
	audioKey = fmt.Sprintf("therapy/%s/%s", uuid.New(), filename)

	if s.cfg.ProxyBaseURL != "" {
		return s.cfg.ProxyBaseURL + "/upload?key=" + url.QueryEscape(audioKey), audioKey, nil
	}

	uploadURL, err = s.client.PresignUpload(ctx, audioKey)
	if err != nil {
		return "", "", fmt.Errorf("storage: presign therapy put: %w", err)
	}
	return uploadURL, audioKey, nil
}

// GetObject downloads an object from storage. Caller must close the returned ReadCloser.
// Used by the therapy service to fetch audio for Whisper transcription.
func (s *StorageService) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	rc, err := s.client.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("storage: get object %q: %w", key, err)
	}
	return rc, nil
}
