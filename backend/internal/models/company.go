package models

import (
	"time"

	"github.com/google/uuid"
)

type Company struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	AdminEmail string    `json:"admin_email"`
	Plan       string    `json:"plan"`
	SeatLimit  int       `json:"seat_limit"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CompanyMember struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"` // member | admin
	JoinedAt  time.Time `json:"joined_at"`
}

// TeamDailyMood is one row from v_team_daily_mood — fully anonymised.
type TeamDailyMood struct {
	Day           string `json:"day"`            // YYYY-MM-DD
	AvgMood       int    `json:"avg_mood"`
	ActiveMembers int    `json:"active_members"` // distinct users who journalled
	EntryCount    int    `json:"entry_count"`
}

// TeamMoodSummary is the response for GET /b2b/companies/:slug/mood.
type TeamMoodSummary struct {
	CompanyID     uuid.UUID        `json:"company_id"`
	CompanyName   string           `json:"company_name"`
	TotalMembers  int              `json:"total_members"`
	Days          []*TeamDailyMood `json:"days"`
	AvgMood       *int             `json:"avg_mood"`        // weighted, nil if no data
	PrevAvgMood   *int             `json:"prev_avg_mood"`   // prior equal period
	MoodDelta     *int             `json:"mood_delta"`
	AlertThreshold int             `json:"alert_threshold"` // 0 = no alert configured
	IsAlerted     bool             `json:"is_alerted"`      // avg_mood < alert_threshold
}
