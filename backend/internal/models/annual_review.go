package models

import (
	"time"

	"github.com/google/uuid"
)

type AnnualReviewStatus string

const (
	AnnualReviewStatusPending   AnnualReviewStatus = "pending"
	AnnualReviewStatusCompleted AnnualReviewStatus = "completed"
	AnnualReviewStatusFailed    AnnualReviewStatus = "failed"
)

// MonthlyMoodArcDay is one month's aggregated mood in an annual review.
type MonthlyMoodArcDay struct {
	Month      string `json:"month"`       // YYYY-MM
	AvgMood    int    `json:"avg_mood"`
	EntryCount int    `json:"entry_count"`
}

// AnnualReview holds the Claude-generated yearly summary for one user.
type AnnualReview struct {
	ID          uuid.UUID          `json:"id"`
	UserID      uuid.UUID          `json:"user_id"`
	Year        int                `json:"year"`          // calendar year reviewed (e.g., 2025)
	Narrative   string             `json:"narrative"`
	TopEmotions []string           `json:"top_emotions"`
	TopTopics   []string           `json:"top_topics"`
	MoodArc     []MonthlyMoodArcDay `json:"mood_arc"`
	EntryCount  int                `json:"entry_count"`
	AvgMood     *int               `json:"avg_mood"`      // nil if no entries
	Status      AnnualReviewStatus `json:"status"`
	ScheduledAt time.Time          `json:"scheduled_at"`
	GeneratedAt *time.Time         `json:"generated_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

// YearSummaryEntry is the per-entry data fed into the annual review prompt.
type YearSummaryEntry struct {
	Date      time.Time
	Summary   string
	MoodScore int
	Emotions  []string
	Topics    []string
}
