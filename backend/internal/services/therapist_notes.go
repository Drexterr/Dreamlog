package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/google/uuid"
)

// AllowedNoteImageTypes are the content types accepted for note photo uploads.
// Claude vision handles jpeg/png/webp; HEIC is not supported (mobile converts
// to JPEG before upload).
var AllowedNoteImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// MaxNoteImageBytes caps note photos at 10 MB (also enforced client-side).
const MaxNoteImageBytes = 10 * 1024 * 1024

// notesKeyStore is the subset of the notes repository the cipher needs.
type notesKeyStore interface {
	GetWrappedDEK(ctx context.Context, therapistID uuid.UUID) ([]byte, error)
	InsertWrappedDEK(ctx context.Context, therapistID uuid.UUID, wrapped []byte) ([]byte, error)
}

type notesRepo interface {
	notesKeyStore
	CreateExternalClient(ctx context.Context, therapistID uuid.UUID, nameEnc []byte, role string) (*models.ExternalClientRow, error)
	ListExternalClients(ctx context.Context, therapistID uuid.UUID, includeArchived bool) ([]*models.ExternalClientRow, error)
	GetExternalClient(ctx context.Context, therapistID, clientID uuid.UUID) (*models.ExternalClientRow, error)
	UpdateExternalClient(ctx context.Context, therapistID, clientID uuid.UUID, nameEnc []byte, role *string, archived *bool) (bool, error)
	DeleteExternalClient(ctx context.Context, therapistID, clientID uuid.UUID) (bool, error)
	CreateSession(ctx context.Context, s *models.ClientSessionRow) (*models.ClientSessionRow, error)
	GetSession(ctx context.Context, therapistID, sessionID uuid.UUID) (*models.ClientSessionRow, error)
	ListSessionsForExternalClient(ctx context.Context, therapistID, clientID uuid.UUID, limit int) ([]*models.ClientSessionRow, error)
	ListSessionsForLinkedClient(ctx context.Context, therapistID, linkedUserID uuid.UUID, limit int) ([]*models.ClientSessionRow, error)
	ListRecentSessions(ctx context.Context, therapistID uuid.UUID, limit int) ([]*models.ClientSessionRow, error)
	UpdateSessionBullets(ctx context.Context, therapistID, sessionID uuid.UUID, bulletsEnc []byte) (bool, error)
	UpdateSessionSummary(ctx context.Context, therapistID, sessionID uuid.UUID, summaryEnc []byte) (bool, error)
	DeleteSession(ctx context.Context, therapistID, sessionID uuid.UUID) (imageKey *string, found bool, err error)
	Overview(ctx context.Context, therapistID uuid.UUID) (*models.TherapistOverview, error)
}

// linkChecker verifies an active therapist↔client link (consent gate) before a
// session may reference a linked app user.
type linkChecker interface {
	GetClientLink(ctx context.Context, therapistID, clientID uuid.UUID) (*models.ClientTherapistLink, error)
}

type notesJobQueue interface {
	Enqueue(ctx context.Context, v any) error
}

type notesStorage interface {
	PresignPutKey(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type sessionSummarizer interface {
	GenerateSessionNotesSummary(ctx context.Context, clientLabel string, sessionDate string, bullets []string) (string, error)
}

// NotesCipher performs per-therapist envelope encryption. Shared between the
// API service and the OCR worker.
type NotesCipher struct {
	masterKey []byte
	keys      notesKeyStore
}

func NewNotesCipher(masterKey []byte, keys notesKeyStore) *NotesCipher {
	return &NotesCipher{masterKey: masterKey, keys: keys}
}

// dek returns the therapist's data key, creating one on first use.
func (c *NotesCipher) dek(ctx context.Context, therapistID uuid.UUID) ([]byte, error) {
	wrapped, err := c.keys.GetWrappedDEK(ctx, therapistID)
	if err != nil {
		return nil, err
	}
	if wrapped == nil {
		fresh, err := pkgcrypto.NewDEK()
		if err != nil {
			return nil, err
		}
		wrappedFresh, err := pkgcrypto.WrapDEK(c.masterKey, fresh)
		if err != nil {
			return nil, err
		}
		// On a concurrent race the stored key wins for both callers.
		wrapped, err = c.keys.InsertWrappedDEK(ctx, therapistID, wrappedFresh)
		if err != nil {
			return nil, err
		}
	}
	return pkgcrypto.UnwrapDEK(c.masterKey, wrapped)
}

// EncryptField seals a plaintext string for the therapist. Empty input → nil.
func (c *NotesCipher) EncryptField(ctx context.Context, therapistID uuid.UUID, plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	dek, err := c.dek(ctx, therapistID)
	if err != nil {
		return nil, err
	}
	return pkgcrypto.Encrypt(dek, []byte(plaintext))
}

// DecryptField opens ciphertext for the therapist. Nil input → "".
func (c *NotesCipher) DecryptField(ctx context.Context, therapistID uuid.UUID, ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	dek, err := c.dek(ctx, therapistID)
	if err != nil {
		return "", err
	}
	plaintext, err := pkgcrypto.Decrypt(dek, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptBullets seals a bullet list as encrypted JSON.
func (c *NotesCipher) EncryptBullets(ctx context.Context, therapistID uuid.UUID, bullets []string) ([]byte, error) {
	if len(bullets) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(bullets)
	if err != nil {
		return nil, fmt.Errorf("notes: marshal bullets: %w", err)
	}
	dek, err := c.dek(ctx, therapistID)
	if err != nil {
		return nil, err
	}
	return pkgcrypto.Encrypt(dek, b)
}

// DecryptBullets opens an encrypted bullet list. Nil input → empty slice.
func (c *NotesCipher) DecryptBullets(ctx context.Context, therapistID uuid.UUID, ciphertext []byte) ([]string, error) {
	if len(ciphertext) == 0 {
		return []string{}, nil
	}
	raw, err := c.DecryptField(ctx, therapistID, ciphertext)
	if err != nil {
		return nil, err
	}
	var bullets []string
	if err := json.Unmarshal([]byte(raw), &bullets); err != nil {
		return nil, fmt.Errorf("notes: unmarshal bullets: %w", err)
	}
	return bullets, nil
}

// ── TherapistNotesService ────────────────────────────────────────────────────

type TherapistNotesService struct {
	repo    notesRepo
	links   linkChecker
	queue   notesJobQueue
	storage notesStorage
	claude  sessionSummarizer
	cipher  *NotesCipher
}

func NewTherapistNotesService(repo notesRepo, links linkChecker, queue notesJobQueue, storage notesStorage, claude sessionSummarizer, cipher *NotesCipher) *TherapistNotesService {
	return &TherapistNotesService{
		repo: repo, links: links, queue: queue, storage: storage, claude: claude, cipher: cipher,
	}
}

// Cipher exposes the underlying cipher (worker wiring).
func (s *TherapistNotesService) Cipher() *NotesCipher { return s.cipher }

// ── External clients ─────────────────────────────────────────────────────────

func (s *TherapistNotesService) CreateExternalClient(ctx context.Context, therapistID uuid.UUID, name, role string) (*models.ExternalClient, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierr.BadRequest("name is required")
	}
	if len(name) > 200 {
		return nil, apierr.BadRequest("name too long")
	}
	if role == "" {
		role = "client"
	}
	nameEnc, err := s.cipher.EncryptField(ctx, therapistID, name)
	if err != nil {
		return nil, fmt.Errorf("notes: encrypt name: %w", err)
	}
	row, err := s.repo.CreateExternalClient(ctx, therapistID, nameEnc, role)
	if err != nil {
		return nil, err
	}
	return s.toExternalClient(ctx, row)
}

func (s *TherapistNotesService) ListExternalClients(ctx context.Context, therapistID uuid.UUID, includeArchived bool) ([]*models.ExternalClient, error) {
	rows, err := s.repo.ListExternalClients(ctx, therapistID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]*models.ExternalClient, 0, len(rows))
	for _, row := range rows {
		c, err := s.toExternalClient(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *TherapistNotesService) GetExternalClient(ctx context.Context, therapistID, clientID uuid.UUID) (*models.ExternalClient, error) {
	row, err := s.repo.GetExternalClient(ctx, therapistID, clientID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apierr.NotFound("client")
	}
	return s.toExternalClient(ctx, row)
}

type UpdateExternalClientInput struct {
	Name     *string `json:"name"`
	Role     *string `json:"role"`
	Archived *bool   `json:"archived"`
}

func (s *TherapistNotesService) UpdateExternalClient(ctx context.Context, therapistID, clientID uuid.UUID, input UpdateExternalClientInput) (*models.ExternalClient, error) {
	if input.Name == nil && input.Role == nil && input.Archived == nil {
		return nil, apierr.BadRequest("at least one field must be provided")
	}
	var nameEnc []byte
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, apierr.BadRequest("name cannot be empty")
		}
		var err error
		nameEnc, err = s.cipher.EncryptField(ctx, therapistID, trimmed)
		if err != nil {
			return nil, fmt.Errorf("notes: encrypt name: %w", err)
		}
	}
	found, err := s.repo.UpdateExternalClient(ctx, therapistID, clientID, nameEnc, input.Role, input.Archived)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, apierr.NotFound("client")
	}
	return s.GetExternalClient(ctx, therapistID, clientID)
}

func (s *TherapistNotesService) DeleteExternalClient(ctx context.Context, therapistID, clientID uuid.UUID) error {
	found, err := s.repo.DeleteExternalClient(ctx, therapistID, clientID)
	if err != nil {
		return err
	}
	if !found {
		return apierr.NotFound("client")
	}
	return nil
}

func (s *TherapistNotesService) toExternalClient(ctx context.Context, row *models.ExternalClientRow) (*models.ExternalClient, error) {
	name, err := s.cipher.DecryptField(ctx, row.TherapistID, row.NameEnc)
	if err != nil {
		return nil, fmt.Errorf("notes: decrypt client name: %w", err)
	}
	return &models.ExternalClient{
		ID: row.ID, TherapistID: row.TherapistID, Name: name, Role: row.Role,
		Archived: row.Archived, SessionCount: row.SessionCount, LastSessionAt: row.LastSessionAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// ── Sessions ─────────────────────────────────────────────────────────────────

// PresignNoteUpload returns a PUT URL for a note photo.
func (s *TherapistNotesService) PresignNoteUpload(ctx context.Context, therapistID uuid.UUID, filename, contentType string) (uploadURL, key string, err error) {
	ext, ok := AllowedNoteImageTypes[contentType]
	if !ok {
		return "", "", apierr.BadRequest("content_type must be image/jpeg, image/png or image/webp")
	}
	// Key is fully server-generated (client filename is ignored) and namespaced
	// under the therapist's ID - CreateSession later verifies this prefix so a
	// therapist can only attach objects their own presign issued.
	_ = filename
	key = fmt.Sprintf("notes/%s/%s%s", therapistID, uuid.New(), ext)
	uploadURL, err = s.storage.PresignPutKey(ctx, key)
	if err != nil {
		return "", "", err
	}
	return uploadURL, key, nil
}

type CreateSessionInput struct {
	ExternalClientID *uuid.UUID `json:"external_client_id"`
	LinkedClientID   *uuid.UUID `json:"linked_client_id"`
	SessionDate      string     `json:"session_date"` // YYYY-MM-DD; defaults to today
	ImageKey         string     `json:"image_key"`    // OCR path
	Bullets          []string   `json:"bullets"`      // manual path
}

// CreateSession creates a session for an external or linked client.
// With ImageKey: status=pending and an OCR job is queued.
// With Bullets: notes are stored immediately, status=completed.
func (s *TherapistNotesService) CreateSession(ctx context.Context, therapistID uuid.UUID, input CreateSessionInput) (*models.ClientSession, error) {
	if (input.ExternalClientID == nil) == (input.LinkedClientID == nil) {
		return nil, apierr.BadRequest("provide exactly one of external_client_id or linked_client_id")
	}
	if input.ImageKey == "" && len(input.Bullets) == 0 {
		return nil, apierr.BadRequest("provide image_key (photo of notes) or bullets (typed notes)")
	}
	if input.ImageKey != "" && len(input.Bullets) > 0 {
		return nil, apierr.BadRequest("provide either image_key or bullets, not both")
	}

	sessionDate := time.Now().UTC()
	if input.SessionDate != "" {
		parsed, err := time.Parse("2006-01-02", input.SessionDate)
		if err != nil {
			return nil, apierr.BadRequest("session_date must be YYYY-MM-DD")
		}
		sessionDate = parsed
	}

	// Ownership / consent checks.
	if input.ExternalClientID != nil {
		client, err := s.repo.GetExternalClient(ctx, therapistID, *input.ExternalClientID)
		if err != nil {
			return nil, err
		}
		if client == nil {
			return nil, apierr.NotFound("client")
		}
	} else {
		link, err := s.links.GetClientLink(ctx, therapistID, *input.LinkedClientID)
		if err != nil {
			return nil, err
		}
		if link == nil {
			return nil, apierr.NotFound("client") // no active (consented) link
		}
	}

	// The image key must be one this therapist's presign issued - prevents a
	// therapist pointing OCR at arbitrary storage objects (e.g. another
	// therapist's photo) by guessing keys.
	if input.ImageKey != "" && !strings.HasPrefix(input.ImageKey, fmt.Sprintf("notes/%s/", therapistID)) {
		return nil, apierr.BadRequest("invalid image_key")
	}

	row := &models.ClientSessionRow{
		TherapistID:      therapistID,
		ExternalClientID: input.ExternalClientID,
		LinkedUserID:     input.LinkedClientID,
		SessionDate:      sessionDate,
	}

	if input.ImageKey != "" {
		row.Status = models.ClientSessionPending
		key := input.ImageKey
		row.ImageKey = &key
	} else {
		bullets := cleanBullets(input.Bullets)
		if len(bullets) == 0 {
			return nil, apierr.BadRequest("bullets cannot be empty")
		}
		bulletsEnc, err := s.cipher.EncryptBullets(ctx, therapistID, bullets)
		if err != nil {
			return nil, fmt.Errorf("notes: encrypt bullets: %w", err)
		}
		row.Status = models.ClientSessionCompleted
		row.BulletsEnc = bulletsEnc
	}

	created, err := s.repo.CreateSession(ctx, row)
	if err != nil {
		return nil, err
	}

	if input.ImageKey != "" {
		job := models.NoteOCRJob{SessionID: created.ID, TherapistID: therapistID, ImageKey: input.ImageKey}
		if err := s.queue.Enqueue(ctx, &job); err != nil {
			// Don't leave a forever-pending row behind - remove it so the
			// therapist sees a clean error and can simply retry.
			_, _, _ = s.repo.DeleteSession(ctx, therapistID, created.ID)
			return nil, fmt.Errorf("notes: enqueue OCR job: %w", err)
		}
	}

	return s.toSession(ctx, created)
}

func (s *TherapistNotesService) GetSession(ctx context.Context, therapistID, sessionID uuid.UUID) (*models.ClientSession, error) {
	row, err := s.repo.GetSession(ctx, therapistID, sessionID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apierr.NotFound("session")
	}
	return s.toSession(ctx, row)
}

// ListSessions returns sessions for one client (external or linked).
func (s *TherapistNotesService) ListSessions(ctx context.Context, therapistID uuid.UUID, externalClientID, linkedClientID *uuid.UUID, limit int) ([]*models.ClientSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []*models.ClientSessionRow
	var err error
	switch {
	case externalClientID != nil:
		rows, err = s.repo.ListSessionsForExternalClient(ctx, therapistID, *externalClientID, limit)
	case linkedClientID != nil:
		rows, err = s.repo.ListSessionsForLinkedClient(ctx, therapistID, *linkedClientID, limit)
	default:
		rows, err = s.repo.ListRecentSessions(ctx, therapistID, limit)
	}
	if err != nil {
		return nil, err
	}
	out := make([]*models.ClientSession, 0, len(rows))
	for _, row := range rows {
		sess, err := s.toSession(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, nil
}

// UpdateBullets replaces the editable bullet list on a session.
func (s *TherapistNotesService) UpdateBullets(ctx context.Context, therapistID, sessionID uuid.UUID, bullets []string) (*models.ClientSession, error) {
	bullets = cleanBullets(bullets)
	if len(bullets) == 0 {
		return nil, apierr.BadRequest("bullets cannot be empty")
	}
	bulletsEnc, err := s.cipher.EncryptBullets(ctx, therapistID, bullets)
	if err != nil {
		return nil, fmt.Errorf("notes: encrypt bullets: %w", err)
	}
	found, err := s.repo.UpdateSessionBullets(ctx, therapistID, sessionID, bulletsEnc)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, apierr.NotFound("session")
	}
	return s.GetSession(ctx, therapistID, sessionID)
}

// Summarize generates (and stores) an AI summary of the session's bullets.
func (s *TherapistNotesService) Summarize(ctx context.Context, therapistID, sessionID uuid.UUID) (*models.ClientSession, error) {
	row, err := s.repo.GetSession(ctx, therapistID, sessionID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apierr.NotFound("session")
	}
	if row.Status != models.ClientSessionCompleted {
		return nil, apierr.Conflict("session notes are not ready yet")
	}
	bullets, err := s.cipher.DecryptBullets(ctx, therapistID, row.BulletsEnc)
	if err != nil {
		return nil, fmt.Errorf("notes: decrypt bullets: %w", err)
	}
	if len(bullets) == 0 {
		return nil, apierr.Conflict("session has no notes to summarize")
	}

	// Client label for the prompt: an anonymous placeholder, never the name.
	// The AI does not need the client's identity to summarize the notes.
	summary, err := s.claude.GenerateSessionNotesSummary(ctx, "the client", row.SessionDate.Format("2006-01-02"), bullets)
	if err != nil {
		return nil, fmt.Errorf("notes: summarize: %w", err)
	}
	summaryEnc, err := s.cipher.EncryptField(ctx, therapistID, summary)
	if err != nil {
		return nil, fmt.Errorf("notes: encrypt summary: %w", err)
	}
	if _, err := s.repo.UpdateSessionSummary(ctx, therapistID, sessionID, summaryEnc); err != nil {
		return nil, err
	}
	return s.GetSession(ctx, therapistID, sessionID)
}

// DeleteSession removes a session and any not-yet-processed note photo.
func (s *TherapistNotesService) DeleteSession(ctx context.Context, therapistID, sessionID uuid.UUID) error {
	imageKey, found, err := s.repo.DeleteSession(ctx, therapistID, sessionID)
	if err != nil {
		return err
	}
	if !found {
		return apierr.NotFound("session")
	}
	if imageKey != nil && *imageKey != "" {
		// Best-effort cleanup of an orphaned photo (session deleted before OCR ran).
		_ = s.storage.Delete(ctx, *imageKey)
	}
	return nil
}

func (s *TherapistNotesService) Overview(ctx context.Context, therapistID uuid.UUID) (*models.TherapistOverview, error) {
	return s.repo.Overview(ctx, therapistID)
}

func (s *TherapistNotesService) toSession(ctx context.Context, row *models.ClientSessionRow) (*models.ClientSession, error) {
	rawText, err := s.cipher.DecryptField(ctx, row.TherapistID, row.RawTextEnc)
	if err != nil {
		return nil, fmt.Errorf("notes: decrypt raw text: %w", err)
	}
	bullets, err := s.cipher.DecryptBullets(ctx, row.TherapistID, row.BulletsEnc)
	if err != nil {
		return nil, fmt.Errorf("notes: decrypt bullets: %w", err)
	}
	summary, err := s.cipher.DecryptField(ctx, row.TherapistID, row.SummaryEnc)
	if err != nil {
		return nil, fmt.Errorf("notes: decrypt summary: %w", err)
	}
	errMsg := ""
	if row.ErrorMsg != nil {
		errMsg = *row.ErrorMsg
	}
	return &models.ClientSession{
		ID: row.ID, TherapistID: row.TherapistID,
		ExternalClientID: row.ExternalClientID, LinkedUserID: row.LinkedUserID,
		SessionDate: row.SessionDate.Format("2006-01-02"), Status: row.Status,
		RawText: rawText, Bullets: bullets, Summary: summary, ErrorMsg: errMsg,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// cleanBullets trims entries, drops empties, and caps count/length.
func cleanBullets(in []string) []string {
	out := make([]string, 0, len(in))
	for _, b := range in {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if len(b) > 2000 {
			b = b[:2000]
		}
		out = append(out, b)
		if len(out) >= 100 {
			break
		}
	}
	return out
}
