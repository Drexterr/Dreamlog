package models

import "github.com/google/uuid"

// TranscriptionJob is the payload serialized into Redis.
type TranscriptionJob struct {
	EntryID  uuid.UUID `json:"entry_id"`
	AudioKey string    `json:"audio_key"`
	UserID   uuid.UUID `json:"user_id"`
	Attempt  int       `json:"attempt"` // 0-indexed; incremented on retry
}
