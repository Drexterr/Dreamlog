package models

import (
	"time"

	"github.com/google/uuid"
)

// CurrentToSVersion is bumped whenever the Terms of Service / Privacy Policy
// materially change; clients compare it against the user's accepted version
// and re-prompt when they differ.
const CurrentToSVersion = "1.0"

// ClientSessionStatus mirrors the client_session_status Postgres ENUM.
type ClientSessionStatus string

const (
	ClientSessionPending    ClientSessionStatus = "pending"
	ClientSessionProcessing ClientSessionStatus = "processing"
	ClientSessionCompleted  ClientSessionStatus = "completed"
	ClientSessionFailed     ClientSessionStatus = "failed"
)

// ExternalClient is a client the therapist manages who is NOT an Ode user.
// Name is decrypted by the service layer before this struct is returned.
type ExternalClient struct {
	ID            uuid.UUID  `json:"id"`
	TherapistID   uuid.UUID  `json:"therapist_id"`
	Name          string     `json:"name"` // plaintext (decrypted); stored encrypted
	Role          string     `json:"role"`
	Archived      bool       `json:"archived"`
	SessionCount  int        `json:"session_count"`
	LastSessionAt *time.Time `json:"last_session_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ExternalClientRow is the raw DB shape with ciphertext, used only between
// repository and service.
type ExternalClientRow struct {
	ID            uuid.UUID
	TherapistID   uuid.UUID
	NameEnc       []byte
	Role          string
	Archived      bool
	SessionCount  int
	LastSessionAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ClientSession is one consultation's notes. Text fields are decrypted by the
// service layer; they are stored as ciphertext.
type ClientSession struct {
	ID               uuid.UUID           `json:"id"`
	TherapistID      uuid.UUID           `json:"therapist_id"`
	ExternalClientID *uuid.UUID          `json:"external_client_id,omitempty"`
	LinkedUserID     *uuid.UUID          `json:"linked_user_id,omitempty"`
	SessionDate      string              `json:"session_date"` // YYYY-MM-DD
	Status           ClientSessionStatus `json:"status"`
	RawText          string              `json:"raw_text,omitempty"`
	Bullets          []string            `json:"bullets"`
	Summary          string              `json:"summary,omitempty"`
	ErrorMsg         string              `json:"error_msg,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// ClientSessionRow is the raw DB shape with ciphertext.
type ClientSessionRow struct {
	ID               uuid.UUID
	TherapistID      uuid.UUID
	ExternalClientID *uuid.UUID
	LinkedUserID     *uuid.UUID
	SessionDate      time.Time
	Status           ClientSessionStatus
	ImageKey         *string
	RawTextEnc       []byte
	BulletsEnc       []byte
	SummaryEnc       []byte
	ErrorMsg         *string
	RetryCount       int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NoteOCRJob is the payload serialized into the notes Redis queue.
type NoteOCRJob struct {
	SessionID   uuid.UUID `json:"session_id"`
	TherapistID uuid.UUID `json:"therapist_id"`
	ImageKey    string    `json:"image_key"`
	Attempt     int       `json:"attempt"`
}

// NotesOCROutput is the structured result of the Claude vision OCR call.
type NotesOCROutput struct {
	RawText string   `json:"raw_text"`
	Bullets []string `json:"bullets"`
}

// TherapistOverview is the caseload metrics block for the therapist dashboard.
type TherapistOverview struct {
	ExternalClients   int        `json:"external_clients"`
	LinkedClients     int        `json:"linked_clients"`
	SessionsThisWeek  int        `json:"sessions_this_week"`
	SessionsThisMonth int        `json:"sessions_this_month"`
	TotalSessions     int        `json:"total_sessions"`
	LastSessionAt     *time.Time `json:"last_session_at,omitempty"`
}
