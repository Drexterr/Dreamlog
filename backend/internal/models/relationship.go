package models

import (
	"time"

	"github.com/google/uuid"
)

// PersonRole classifies the relationship between the user and a person.
type PersonRole string

const (
	PersonRoleFamily    PersonRole = "family"
	PersonRoleFriend    PersonRole = "friend"
	PersonRoleColleague PersonRole = "colleague"
	PersonRoleRomantic  PersonRole = "romantic"
	PersonRoleOther     PersonRole = "other"
)

// PersonSentiment is the valence of a single mention.
type PersonSentiment string

const (
	PersonSentimentPositive PersonSentiment = "positive"
	PersonSentimentNeutral  PersonSentiment = "neutral"
	PersonSentimentNegative PersonSentiment = "negative"
)

// Person is an extracted person tracked in the user's relationship map.
type Person struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	Name            string     `json:"name"`
	Role            PersonRole `json:"role"`
	MentionCount    int        `json:"mention_count"`
	PositiveCount   int        `json:"positive_count"`
	NegativeCount   int        `json:"negative_count"`
	LastMentionedAt time.Time  `json:"last_mentioned_at"`
	Hidden          bool       `json:"hidden"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UpdatePersonInput carries optional edits to a person. Nil fields are left
// unchanged. Used for rename, re-categorize, and hide/unhide.
type UpdatePersonInput struct {
	Name   *string `json:"name"`
	Role   *string `json:"role"`
	Hidden *bool   `json:"hidden"`
}

// PersonMention is one extracted mention of a person in an entry.
type PersonMention struct {
	ID        uuid.UUID       `json:"id"`
	PersonID  uuid.UUID       `json:"person_id"`
	EntryID   uuid.UUID       `json:"entry_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Sentiment PersonSentiment `json:"sentiment"`
	Context   string          `json:"context"`
	CreatedAt time.Time       `json:"created_at"`
}

// PersonDetail combines a person with their recent mentions.
type PersonDetail struct {
	Person   *Person         `json:"person"`
	Mentions []PersonMention `json:"mentions"`
}

// ExtractedPerson is returned by the Claude person-extraction call.
// Defined here (not in services) to avoid import cycles with repositories.
type ExtractedPerson struct {
	Name      string `json:"name"`
	Role      string `json:"role"`      // family|friend|colleague|romantic|other
	Sentiment string `json:"sentiment"` // positive|neutral|negative
	Context   string `json:"context"`
}

// PersonExtractionOutput is the full JSON response from the extraction prompt.
type PersonExtractionOutput struct {
	People []ExtractedPerson `json:"people"`
}
