package models

import (
	"time"

	"github.com/google/uuid"
)

// LifeChapter is a user-defined named period of time.
type LifeChapter struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartDate   string     `json:"start_date"` // YYYY-MM-DD
	EndDate     *string    `json:"end_date"`   // YYYY-MM-DD or null (ongoing)
	Emoji       string     `json:"emoji"`
	Color       string     `json:"color"`      // hex e.g. "#7C3AED"
	Summary     string     `json:"summary"`    // Claude-generated, empty until requested
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ChapterDetail is LifeChapter enriched with aggregated entry data.
type ChapterDetail struct {
	LifeChapter
	EntryCount  int          `json:"entry_count"`
	AvgMood     *int         `json:"avg_mood"`     // nil if no entries
	TopEmotions []string     `json:"top_emotions"`
	MoodArc     []MoodArcDay `json:"mood_arc"`     // daily avg mood within the chapter
}

// CreateChapterInput is the request body for POST /chapters.
type CreateChapterInput struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	StartDate   string  `json:"start_date" binding:"required"` // YYYY-MM-DD
	EndDate     *string `json:"end_date"`
	Emoji       string  `json:"emoji"`
	Color       string  `json:"color"`
}

// UpdateChapterInput is the request body for PUT /chapters/:id.
type UpdateChapterInput struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	EndDate     *string  `json:"end_date"` // empty string = clear end date
	Emoji       *string  `json:"emoji"`
	Color       *string  `json:"color"`
}
