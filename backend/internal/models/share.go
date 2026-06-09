package models

import (
	"time"

	"github.com/google/uuid"
)

// ShareLink is a 72-hour passcode-protected read-only link for a therapist.
type ShareLink struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Token        string
	PasscodeHash string
	ExpiresAt    time.Time
	Revoked      bool
	CreatedAt    time.Time
}

// ShareLinkView is the public payload returned when a therapist opens a valid link.
// Raw transcripts and reflections are excluded - only AI summaries and mood data.
type ShareLinkView struct {
	UserName    string          `json:"user_name"`
	Period      string          `json:"period"` // always "30d"
	MoodArc     []MoodArcDay    `json:"mood_arc"`
	AvgMood     *int            `json:"avg_mood"`
	TopEmotions []string        `json:"top_emotions"`
	Summaries   []EntrySummary  `json:"summaries"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

// EntrySummary is a single entry's anonymized AI output - no raw transcript.
type EntrySummary struct {
	Date      string   `json:"date"`       // YYYY-MM-DD
	Summary   string   `json:"summary"`    // 2-3 sentence AI summary only
	MoodScore int      `json:"mood_score"`
	Topics    []string `json:"topics"`
}

// CreateShareLinkInput holds what the handler passes to the repository.
type CreateShareLinkInput struct {
	UserID       uuid.UUID
	Token        string
	PasscodeHash string
	ExpiresAt    time.Time
}

// CreateShareLinkResult is returned to the mobile client after link creation.
type CreateShareLinkResult struct {
	Token     string    `json:"token"`
	Passcode  string    `json:"passcode"`   // plaintext, shown once
	URL       string    `json:"url"`        // full shareable URL
	ExpiresAt time.Time `json:"expires_at"`
}
