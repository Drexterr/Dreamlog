package models

import (
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/google/uuid"
	"strings"
	"unicode/utf8"
)

const MaxTextEntryCharacters = 20000

type CreateTextEntryInput struct {
	Content   string    `json:"content" binding:"required"`
	RequestID string    `json:"request_id" binding:"required"`
	Mode      EntryMode `json:"mode"`
	// Language is detected server-side from the text (never client-supplied)
	// and stored like Whisper's language, so typed Hindi journals get the
	// Hindi prompt (N11).
	Language string `json:"-"`
}

func (in *CreateTextEntryInput) Validate() error {
	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" || utf8.RuneCountInString(in.Content) > MaxTextEntryCharacters {
		return apierr.BadRequest("journal must contain 1 to 20000 characters")
	}
	id, err := uuid.Parse(in.RequestID)
	if err != nil || id == uuid.Nil {
		return apierr.BadRequest("request_id must be a non-empty UUID")
	}
	in.RequestID = id.String()
	if in.Mode == "" {
		in.Mode = EntryModeProcessing
	}
	if !in.Mode.Valid() {
		return apierr.BadRequest("invalid journal mode")
	}
	return nil
}

// A retry token never addresses another user's entry.
func TextEntryID(userID uuid.UUID, requestID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("ode/text/"+userID.String()+"/"+requestID))
}

func (in *CreateTextEntryInput) Matches(entry *Entry) bool {
	return entry != nil && entry.Transcript != nil && *entry.Transcript == in.Content && entry.Mode == in.Mode
}
