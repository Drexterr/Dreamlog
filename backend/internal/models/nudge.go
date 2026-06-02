package models

import (
	"time"

	"github.com/google/uuid"
)

type NudgeStatus string

const (
	NudgeStatusPending NudgeStatus = "pending"
	NudgeStatusSent    NudgeStatus = "sent"
	NudgeStatusFailed  NudgeStatus = "failed"
)

type Nudge struct {
	ID          uuid.UUID   `json:"id"`
	UserID      uuid.UUID   `json:"user_id"`
	EntryID     *uuid.UUID  `json:"entry_id,omitempty"`
	Message     string      `json:"message"`
	ScheduledAt time.Time   `json:"scheduled_at"`
	Timezone    string      `json:"timezone"`
	Status      NudgeStatus `json:"status"`
	SentAt      *time.Time  `json:"sent_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type UserDevice struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	FCMToken  string    `json:"fcm_token"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterDeviceInput struct {
	FCMToken string `json:"fcm_token" binding:"required"`
	Platform string `json:"platform"  binding:"required,oneof=ios android"`
}

// DailyMood is returned by the mood chart API.
type DailyMood struct {
	Day        string `json:"day"`        // YYYY-MM-DD
	AvgMood    int    `json:"avg_mood"`
	EntryCount int    `json:"entry_count"`
}

type StreakInfo struct {
	CurrentStreak int `json:"current_streak"` // consecutive days
	LongestStreak int `json:"longest_streak"`
	TotalDays     int `json:"total_days"`
	NextMilestone int `json:"next_milestone"` // next milestone target (7/21/50/100); 0 when all reached
	FreezeCount   int `json:"freeze_count"`   // available streak freezes
}

// Milestone thresholds in ascending order.
var StreakMilestones = []int{7, 21, 50, 100}

// NextStreakMilestone returns the next milestone above current, or 0 if all reached.
func NextStreakMilestone(current int) int {
	for _, m := range StreakMilestones {
		if current < m {
			return m
		}
	}
	return 0
}

// IsStreakMilestone reports whether current exactly hits a milestone.
func IsStreakMilestone(current int) bool {
	for _, m := range StreakMilestones {
		if current == m {
			return true
		}
	}
	return false
}

// MoodHistoryResponse is returned by GET /mood/history.
type MoodHistoryResponse struct {
	Days        []*DailyMood `json:"days"`           // daily avg mood, oldest first
	Range       string       `json:"range"`           // "30d" | "90d" | "365d"
	AvgMood     *int         `json:"avg_mood"`        // overall average, null when no data
	PrevAvgMood *int         `json:"prev_avg_mood"`   // average for prior equivalent period, null when no data
	MoodDelta   *int         `json:"mood_delta"`      // AvgMood - PrevAvgMood, null when insufficient data
	TopEmotions []string     `json:"top_emotions"`    // up to 3 most frequent emotions
	EntryCount  int          `json:"entry_count"`     // total entries in the period
}
