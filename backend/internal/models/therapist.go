package models

import (
	"time"

	"github.com/google/uuid"
)

type Therapist struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Credentials string    `json:"credentials,omitempty"`
	Plan        string    `json:"plan"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ClientTherapistLink struct {
	ID           uuid.UUID  `json:"id"`
	TherapistID  uuid.UUID  `json:"therapist_id"`
	ClientID     uuid.UUID  `json:"client_id"`
	Status       string     `json:"status"` // active | revoked
	LinkedAt     time.Time  `json:"linked_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// ClientSummary is returned by GET /therapists/clients - one row per linked client.
type ClientSummary struct {
	ClientID    uuid.UUID `json:"client_id"`
	Name        string    `json:"name"`
	LinkedAt    time.Time `json:"linked_at"`
	LastEntryAt *time.Time `json:"last_entry_at,omitempty"`
	AvgMood30d  *int      `json:"avg_mood_30d,omitempty"`
	EntryCount  int       `json:"entry_count"`
}

// ClientBrief is the pre-session brief Claude generates for a specific client.
type ClientBrief struct {
	ClientID    uuid.UUID `json:"client_id"`
	ClientName  string    `json:"client_name"`
	GeneratedAt time.Time `json:"generated_at"`
	Brief       string    `json:"brief"`         // 3-sentence Claude summary
	TopEmotions []string  `json:"top_emotions"`
	MoodTrend   string    `json:"mood_trend"`    // "improving" | "declining" | "stable" | "insufficient_data"
	AvgMood7d   *int      `json:"avg_mood_7d,omitempty"`
	EntryCount  int       `json:"entry_count"`
	RecentEntries []*ExportEntrySummary `json:"recent_entries"` // last 5
}
