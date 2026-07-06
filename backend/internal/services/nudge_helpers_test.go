package services

import (
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
)

func TestNextOccurrenceOfHour_StillAheadToday(t *testing.T) {
	now := time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC)
	got := nextOccurrenceOfHour(now, 21)
	want := time.Date(2026, 7, 6, 21, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextOccurrenceOfHour_AlreadyPassed_Tomorrow(t *testing.T) {
	now := time.Date(2026, 7, 6, 22, 5, 0, 0, time.UTC)
	got := nextOccurrenceOfHour(now, 21)
	want := time.Date(2026, 7, 7, 21, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextOccurrenceOfHour_ExactHour_Tomorrow(t *testing.T) {
	// At exactly the target hour, the occurrence is not strictly after now.
	now := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	got := nextOccurrenceOfHour(now, 8)
	want := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestCheckinMessage_UsesMorningNudge(t *testing.T) {
	a := &models.EntryAnalysis{MorningNudge: "Notice how the walk felt this morning."}
	if got := CheckinMessage(a); got != a.MorningNudge {
		t.Errorf("want morning nudge reused, got %q", got)
	}
}

func TestCheckinMessage_FallsBackToTopic(t *testing.T) {
	a := &models.EntryAnalysis{Topics: []string{"the job interview"}}
	got := CheckinMessage(a)
	if !strings.Contains(got, "the job interview") {
		t.Errorf("want topic referenced, got %q", got)
	}
}

func TestCheckinMessage_NilAnalysis_GenericLine(t *testing.T) {
	got := CheckinMessage(nil)
	if got == "" {
		t.Error("nil analysis must still produce a message")
	}
}
