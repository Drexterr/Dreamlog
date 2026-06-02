package models

import (
	"time"

	"github.com/google/uuid"
)

type EntryStatus string

const (
	EntryStatusPending    EntryStatus = "pending"
	EntryStatusProcessing EntryStatus = "processing"
	EntryStatusCompleted  EntryStatus = "completed"
	EntryStatusFailed     EntryStatus = "failed"
)

// MaxRecordingSeconds is enforced both client-side and at job processing time.
const MaxRecordingSeconds = 30 * 60 // 30 minutes

// EntryMode controls how Claude analyzes and reflects on the entry.
type EntryMode string

const (
	EntryModeProcessing EntryMode = "processing" // default: full emotional analysis
	EntryModeRant       EntryMode = "rant"       // validation only, no deep analysis
	EntryModeGratitude  EntryMode = "gratitude"  // gratitude framing + follow-up prompts
	EntryModeDecision   EntryMode = "decision"   // Socratic analysis for decisions
	EntryModeDream      EntryMode = "dream"      // dream recounting + symbolic interpretation
)

func (m EntryMode) Valid() bool {
	switch m {
	case EntryModeProcessing, EntryModeRant, EntryModeGratitude, EntryModeDecision, EntryModeDream:
		return true
	}
	return false
}

type Entry struct {
	ID             uuid.UUID   `json:"id"`
	UserID         uuid.UUID   `json:"user_id"`
	AudioKey       string      `json:"audio_key"`
	AudioSizeBytes int64       `json:"audio_size_bytes"`
	DurationSec    float64     `json:"duration_sec"`
	Status         EntryStatus `json:"status"`
	Mode           EntryMode   `json:"mode"`
	Transcript     *string     `json:"transcript,omitempty"`
	Language       *string     `json:"language,omitempty"`
	ErrorMsg       *string     `json:"error_msg,omitempty"`
	RetryCount     int         `json:"retry_count"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// CreateEntryInput is the request body for POST /entries.
// The client calls this after uploading audio to obtain a job.
type CreateEntryInput struct {
	AudioKey       string    `json:"audio_key"        binding:"required"`
	AudioSizeBytes int64     `json:"audio_size_bytes" binding:"required,min=1"`
	DurationSec    float64   `json:"duration_sec"     binding:"required,min=0.1"`
	Mode           EntryMode `json:"mode"`            // optional; defaults to "processing"
}

// PresignResponse is returned by POST /entries/presign.
type PresignResponse struct {
	UploadURL string `json:"upload_url"`
	AudioKey  string `json:"audio_key"`
	ExpiresIn int    `json:"expires_in"` // seconds
}

// ListEntriesResponse wraps paginated entry results.
type ListEntriesResponse struct {
	Entries    []*Entry `json:"entries"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	HasMore    bool     `json:"has_more"`
}
