package models

import (
	"github.com/google/uuid"
	"strings"
	"testing"
)

func TestTextEntryValidation(t *testing.T) {
	for _, tc := range []struct {
		name, content, token string
		mode                 EntryMode
		valid                bool
	}{
		{"normal", " A whole journal ", uuid.NewString(), "", true},
		{"unicode boundary", strings.Repeat("心", MaxTextEntryCharacters), uuid.NewString(), EntryModeRant, true},
		{"too long", strings.Repeat("心", MaxTextEntryCharacters+1), uuid.NewString(), "", false},
		{"empty", " \n\t", uuid.NewString(), "", false},
		{"bad token", "journal", "not-uuid", "", false},
		{"zero token", "journal", uuid.Nil.String(), "", false},
		{"bad mode", "journal", uuid.NewString(), "invalid", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := CreateTextEntryInput{Content: tc.content, RequestID: tc.token, Mode: tc.mode}
			if err := input.Validate(); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			if tc.valid && input.Content != strings.TrimSpace(tc.content) {
				t.Fatal("content not trimmed")
			}
		})
	}
}

func TestTextEntryRetryIDsAreStableAndUserScoped(t *testing.T) {
	user, other, token := uuid.New(), uuid.New(), uuid.NewString()
	if TextEntryID(user, token) != TextEntryID(user, token) {
		t.Fatal("unstable ID")
	}
	if TextEntryID(user, token) == TextEntryID(other, token) {
		t.Fatal("cross-user ID collision")
	}
}
