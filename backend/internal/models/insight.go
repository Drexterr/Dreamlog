package models

import (
	"time"

	"github.com/google/uuid"
)

// InsightCardData is the payload returned by GET /insights/card.
// It contains everything the mobile ShareInsightModal needs to render and share.
type InsightCardData struct {
	WeekLabel   string       `json:"week_label"`   // e.g. "May 26 – Jun 1"
	WeekStart   string       `json:"week_start"`   // YYYY-MM-DD (Monday)
	MoodArc     []MoodArcDay `json:"mood_arc"`     // daily avg mood Mon–Sun
	TopEmotions []string     `json:"top_emotions"` // up to 3 emotions this week
	Streak      int          `json:"streak"`       // current journaling streak
	EntryCount  int          `json:"entry_count"`  // entries made this week
	ShareCount  int          `json:"share_count"`  // total insight cards shared by user
}

// InsightShare is one recorded share event.
type InsightShare struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	WeekStart string    `json:"week_start"` // YYYY-MM-DD
	CreatedAt time.Time `json:"created_at"`
}

// InsightShareResult is returned after POST /insights/share.
type InsightShareResult struct {
	TotalShares int    `json:"total_shares"` // user's all-time share count
	WeekStart   string `json:"week_start"`
}
