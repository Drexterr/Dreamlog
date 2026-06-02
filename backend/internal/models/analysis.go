package models

import (
	"time"

	"github.com/google/uuid"
)

// EmotionalTone is one element in the emotional_tone JSONB array.
type EmotionalTone struct {
	Emotion   string  `json:"emotion"`
	Intensity float64 `json:"intensity"` // 0.0 – 1.0
}

// EntryAnalysis is the full AI output for a single entry.
type EntryAnalysis struct {
	ID            uuid.UUID       `json:"id"`
	EntryID       uuid.UUID       `json:"entry_id"`
	MoodScore     int             `json:"mood_score"`      // 1–100
	EmotionalTone []EmotionalTone `json:"emotional_tone"`
	Topics        []string        `json:"topics"`
	KeyQuotes     []string        `json:"key_quotes"`
	Summary       string          `json:"summary"`
	Reflection    string          `json:"reflection"`
	MorningNudge  string          `json:"morning_nudge"`
	IsCrisis      bool            `json:"is_crisis"`
	// Dream Decoder fields — non-nil only when entry.mode = 'dream'.
	DreamSymbols  []string        `json:"dream_symbols,omitempty"`
	DreamType     string          `json:"dream_type,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ClaudeAnalysisOutput is the JSON structure Claude must return.
// Field names match the prompt schema exactly.
type ClaudeAnalysisOutput struct {
	EmotionalTone []EmotionalTone `json:"emotional_tone"`
	Topics        []string        `json:"topics"`
	MoodScore     int             `json:"mood_score"`
	KeyQuotes     []string        `json:"key_quotes"`
	Summary       string          `json:"summary"`
	Reflection    string          `json:"reflection"`
	MorningNudge  string          `json:"morning_nudge"`
	// Dream Decoder fields — populated only when mode = 'dream'.
	DreamSymbols  []string        `json:"dream_symbols,omitempty"`
	DreamType     string          `json:"dream_type,omitempty"`
}

// ── Pattern Radar ────────────────────────────────────────────────────────────

// EmotionPattern is a single emotion axis in the pattern radar.
type EmotionPattern struct {
	Emotion      string  `json:"emotion"`
	Frequency    int     `json:"frequency"`     // count of entries where this emotion appeared
	AvgIntensity float64 `json:"avg_intensity"` // average intensity when it appeared (0.0-1.0)
	Score        float64 `json:"score"`         // normalized combined score for radar axis (0.0-1.0)
}

// MoodDistribution splits entries into high/neutral/low buckets.
type MoodDistribution struct {
	High    int `json:"high"`    // mood_score >= 70
	Neutral int `json:"neutral"` // 40-69
	Low     int `json:"low"`     // < 40
}

// PatternRadarResponse is returned by GET /mood/patterns.
type PatternRadarResponse struct {
	Range            string           `json:"range"`
	Emotions         []EmotionPattern `json:"emotions"`          // top 8 emotions, sorted by score desc
	TotalEntries     int              `json:"total_entries"`
	MoodDistribution MoodDistribution `json:"mood_distribution"`
}
