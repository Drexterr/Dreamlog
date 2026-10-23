package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Dead-letter queue administration (operator tooling, /admin/dlq).
//
// Jobs that exhaust WORKER_MAX_RETRIES are LPUSHed by the workers onto Redis
// DLQ lists as {"payload": <job>, "error": "...", "failed_at": "..."}: entry
// pipeline jobs on WORKER_DLQ_KEY, therapist note OCR jobs on
// WORKER_NOTES_DLQ_KEY. (The dead_letter_jobs table from migration 000001 is
// not written by anything.) This service lists those items without exposing
// any journal content, requeues one, or discards one.

// DLQ names accepted by the admin API.
const (
	DLQEntries = "entries"
	DLQNotes   = "notes"
)

const (
	// dlqMaxScan bounds how many dead letters a lookup reads.
	dlqMaxScan = 10000
	// dlqMaxErrorLen truncates stored worker errors in listings.
	dlqMaxErrorLen = 300
	// dlqRawIDPrefix marks items whose payload could not be parsed; they are
	// addressed by a hash of the raw element instead of an entry/session id.
	dlqRawIDPrefix = "raw-"
)

var (
	ErrDLQUnknownQueue = errors.New("unknown dead-letter queue")
	ErrDLQItemNotFound = errors.New("dead-letter item not found")
	ErrDLQUnparseable  = errors.New("dead-letter job payload cannot be parsed, so it cannot be requeued; discard it")
	ErrDLQTargetGone   = errors.New("the entry or session no longer exists; discard this item")
	ErrDLQCompleted    = errors.New("the entry or session already completed; discard this item")
	ErrDLQNotFailed    = errors.New("the entry or session is not in a failed state")
	ErrDLQSourceGone   = errors.New("the source audio or photo was already deleted and there is no transcript, so a retry cannot succeed; discard this item")
	ErrDLQClaimed      = errors.New("this item was already requeued or discarded")
)

// dlqList is one Redis dead-letter list plus its main queue (*queue.Queue).
type dlqList interface {
	DLQRange(ctx context.Context, start, stop int64) ([]string, error)
	DLQLen(ctx context.Context) (int64, error)
	DLQRemove(ctx context.Context, raw string) (bool, error)
	DLQRestore(ctx context.Context, raw string) error
	Enqueue(ctx context.Context, v any) error
}

type dlqEntryRepo interface {
	GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Entry, error)
	ResetForRequeue(ctx context.Context, id uuid.UUID) (bool, error)
}

type dlqNotesRepo interface {
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.ClientSessionRow, error)
	ResetSessionForRequeue(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

type dlqObjectStore interface {
	Exists(ctx context.Context, key string) (bool, error)
}

// DLQItem is the privacy-safe view of a dead letter: identifiers, the worker
// error and timing only - never transcripts, note text or storage keys.
type DLQItem struct {
	// ID addresses the item in requeue/discard calls: the entry or session
	// id, or "raw-<hash>" for an unparseable payload.
	ID          string     `json:"id"`
	Queue       string     `json:"queue"`
	EntryID     *uuid.UUID `json:"entry_id,omitempty"`
	SessionID   *uuid.UUID `json:"session_id,omitempty"`
	Error       string     `json:"error"`
	Attempt     int        `json:"attempt"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
	Unparseable bool       `json:"unparseable,omitempty"`
}

// DLQAdminService implements the /admin/dlq operations.
type DLQAdminService struct {
	entries dlqList
	notes   dlqList
	entryDB dlqEntryRepo
	notesDB dlqNotesRepo
	storage dlqObjectStore
	log     *zap.Logger
}

func NewDLQAdminService(entries, notes dlqList, entryDB dlqEntryRepo, notesDB dlqNotesRepo, storage dlqObjectStore, log *zap.Logger) *DLQAdminService {
	if log == nil {
		log = zap.NewNop()
	}
	return &DLQAdminService{entries: entries, notes: notes, entryDB: entryDB, notesDB: notesDB, storage: storage, log: log}
}

func (s *DLQAdminService) list(name string) (dlqList, error) {
	switch name {
	case DLQEntries:
		if s.entries != nil {
			return s.entries, nil
		}
	case DLQNotes:
		if s.notes != nil {
			return s.notes, nil
		}
	}
	return nil, ErrDLQUnknownQueue
}

// dlqRecord is the envelope queue.EnqueueDLQ writes.
type dlqRecord struct {
	Payload  json.RawMessage `json:"payload"`
	Error    string          `json:"error"`
	FailedAt time.Time       `json:"failed_at"`
}

// parsedDLQ is a decoded dead letter plus the job fields needed to requeue.
type parsedDLQ struct {
	item  DLQItem
	entry *models.TranscriptionJob
	note  *models.NoteOCRJob
}

func rawItemID(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return dlqRawIDPrefix + hex.EncodeToString(sum[:8])
}

func truncateDLQError(msg string) string {
	r := []rune(msg)
	if len(r) <= dlqMaxErrorLen {
		return msg
	}
	return string(r[:dlqMaxErrorLen]) + "…"
}

func parseDLQ(queueName, raw string) parsedDLQ {
	p := parsedDLQ{item: DLQItem{Queue: queueName, ID: rawItemID(raw), Unparseable: true}}
	var rec dlqRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return p
	}
	p.item.Error = truncateDLQError(rec.Error)
	if !rec.FailedAt.IsZero() {
		t := rec.FailedAt
		p.item.FailedAt = &t
	}
	switch queueName {
	case DLQEntries:
		var job models.TranscriptionJob
		if json.Unmarshal(rec.Payload, &job) == nil && job.EntryID != uuid.Nil {
			id := job.EntryID
			p.entry = &job
			p.item.ID, p.item.EntryID, p.item.Attempt, p.item.Unparseable = id.String(), &id, job.Attempt, false
		}
	case DLQNotes:
		var job models.NoteOCRJob
		if json.Unmarshal(rec.Payload, &job) == nil && job.SessionID != uuid.Nil {
			id := job.SessionID
			p.note = &job
			p.item.ID, p.item.SessionID, p.item.Attempt, p.item.Unparseable = id.String(), &id, job.Attempt, false
		}
	}
	return p
}

// List returns one page of dead letters, newest first, plus the total count.
func (s *DLQAdminService) List(ctx context.Context, queueName string, page, pageSize int) ([]DLQItem, int64, error) {
	q, err := s.list(queueName)
	if err != nil {
		return nil, 0, err
	}
	total, err := q.DLQLen(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("dlqAdmin.List: %w", err)
	}
	start := int64((page - 1) * pageSize)
	items := []DLQItem{}
	if start >= total {
		return items, total, nil
	}
	raws, err := q.DLQRange(ctx, start, start+int64(pageSize)-1)
	if err != nil {
		return nil, 0, fmt.Errorf("dlqAdmin.List: %w", err)
	}
	for _, raw := range raws {
		items = append(items, parseDLQ(queueName, raw).item)
	}
	return items, total, nil
}

// find returns the newest dead letter with the given id, and its raw element.
func (s *DLQAdminService) find(ctx context.Context, q dlqList, queueName, id string) (string, parsedDLQ, error) {
	raws, err := q.DLQRange(ctx, 0, dlqMaxScan-1)
	if err != nil {
		return "", parsedDLQ{}, fmt.Errorf("dlqAdmin.find: %w", err)
	}
	for _, raw := range raws {
		if p := parseDLQ(queueName, raw); p.item.ID == id {
			return raw, p, nil
		}
	}
	return "", parsedDLQ{}, ErrDLQItemNotFound
}

// Discard removes one dead letter without retrying it.
func (s *DLQAdminService) Discard(ctx context.Context, queueName, id string) error {
	q, err := s.list(queueName)
	if err != nil {
		return err
	}
	raw, _, err := s.find(ctx, q, queueName, id)
	if err != nil {
		return err
	}
	removed, err := q.DLQRemove(ctx, raw)
	if err != nil {
		return fmt.Errorf("dlqAdmin.Discard: %w", err)
	}
	if !removed {
		return ErrDLQClaimed
	}
	s.log.Info("dlq: item discarded", zap.String("queue", queueName), zap.String("id", id))
	return nil
}

// Requeue retries one dead letter: the entry/session is reset to pending with
// a fresh retry budget and a new attempt-0 job is enqueued. The item is
// claimed with an atomic LREM first, so a double click enqueues once.
func (s *DLQAdminService) Requeue(ctx context.Context, queueName, id string) error {
	q, err := s.list(queueName)
	if err != nil {
		return err
	}
	raw, p, err := s.find(ctx, q, queueName, id)
	if err != nil {
		return err
	}
	if p.item.Unparseable {
		return ErrDLQUnparseable
	}

	var job any
	var reset func() (bool, error)
	switch {
	case p.entry != nil:
		j, err := s.prepareEntry(ctx, p.entry)
		if err != nil {
			return err
		}
		job = j
		reset = func() (bool, error) { return s.entryDB.ResetForRequeue(ctx, j.EntryID) }
	case p.note != nil:
		j, err := s.prepareNote(ctx, p.note)
		if err != nil {
			return err
		}
		job = j
		reset = func() (bool, error) { return s.notesDB.ResetSessionForRequeue(ctx, j.SessionID) }
	default:
		return ErrDLQUnparseable
	}

	removed, err := q.DLQRemove(ctx, raw)
	if err != nil {
		return fmt.Errorf("dlqAdmin.Requeue claim: %w", err)
	}
	if !removed {
		return ErrDLQClaimed
	}
	restore := func() {
		if rerr := q.DLQRestore(context.WithoutCancel(ctx), raw); rerr != nil {
			s.log.Error("dlq: failed to restore claimed item", zap.String("queue", queueName), zap.String("id", id), zap.Error(rerr))
		}
	}
	ok, err := reset()
	if err != nil {
		restore()
		return fmt.Errorf("dlqAdmin.Requeue reset: %w", err)
	}
	if !ok {
		// Changed state between the check and the claim.
		restore()
		return ErrDLQNotFailed
	}
	if err := q.Enqueue(ctx, job); err != nil {
		// The row is pending again; for entries the reconciler requeues stale
		// pending rows, so the retry is not lost.
		return fmt.Errorf("dlqAdmin.Requeue enqueue: %w", err)
	}
	s.log.Info("dlq: item requeued", zap.String("queue", queueName), zap.String("id", id))
	return nil
}

func (s *DLQAdminService) prepareEntry(ctx context.Context, old *models.TranscriptionJob) (*models.TranscriptionJob, error) {
	if s.entryDB == nil {
		return nil, ErrDLQUnknownQueue
	}
	entry, err := s.entryDB.GetByIDInternal(ctx, old.EntryID)
	if err != nil {
		return nil, fmt.Errorf("dlqAdmin.Requeue load entry: %w", err)
	}
	if entry == nil {
		return nil, ErrDLQTargetGone
	}
	switch entry.Status {
	case models.EntryStatusCompleted:
		return nil, ErrDLQCompleted
	case models.EntryStatusFailed:
	default:
		return nil, ErrDLQNotFailed
	}
	// ADR-005: audio is deleted once a transcript is stored. With a stored
	// transcript (typed journals always have one) the retry reuses it; without
	// one, the recording must still exist or the retry is pointless.
	if entry.Transcript == nil || *entry.Transcript == "" {
		if err := s.requireObject(ctx, entry.AudioKey); err != nil {
			return nil, err
		}
	}
	return &models.TranscriptionJob{
		EntryID:     entry.ID,
		AudioKey:    entry.AudioKey,
		UserID:      entry.UserID,
		UserCountry: old.UserCountry,
		Attempt:     0,
	}, nil
}

func (s *DLQAdminService) prepareNote(ctx context.Context, old *models.NoteOCRJob) (*models.NoteOCRJob, error) {
	if s.notesDB == nil {
		return nil, ErrDLQUnknownQueue
	}
	session, err := s.notesDB.GetSessionByID(ctx, old.SessionID)
	if err != nil {
		return nil, fmt.Errorf("dlqAdmin.Requeue load session: %w", err)
	}
	if session == nil {
		return nil, ErrDLQTargetGone
	}
	switch session.Status {
	case models.ClientSessionCompleted:
		return nil, ErrDLQCompleted
	case models.ClientSessionFailed:
	default:
		return nil, ErrDLQNotFailed
	}
	// ADR-019: the photo is the only OCR source.
	if session.ImageKey == nil || *session.ImageKey == "" {
		return nil, ErrDLQSourceGone
	}
	if err := s.requireObject(ctx, *session.ImageKey); err != nil {
		return nil, err
	}
	return &models.NoteOCRJob{
		SessionID:   session.ID,
		TherapistID: session.TherapistID,
		ImageKey:    *session.ImageKey,
		Attempt:     0,
	}, nil
}

func (s *DLQAdminService) requireObject(ctx context.Context, key string) error {
	if key == "" || s.storage == nil {
		return ErrDLQSourceGone
	}
	ok, err := s.storage.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("dlqAdmin.Requeue storage check: %w", err)
	}
	if !ok {
		return ErrDLQSourceGone
	}
	return nil
}
