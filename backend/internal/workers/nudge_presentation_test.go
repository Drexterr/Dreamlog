package workers

import (
	"context"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

func TestNudgePresentation_ByType(t *testing.T) {
	cases := []struct {
		nudgeType string
		wantTitle string
		wantData  string
	}{
		{"", "Your morning reflection", "morning_nudge"},
		{models.NudgeTypeMorning, "Your morning reflection", "morning_nudge"},
		{models.NudgeTypeCheckin, "You asked me to check in", "checkin"},
		{models.NudgeTypeStreakRisk, "Your streak is waiting", "streak_risk"},
		{models.NudgeTypeReengagement, "DreamLog", "reengagement"},
		{"unknown_future_type", "Your morning reflection", "morning_nudge"},
	}
	for _, c := range cases {
		title, dataType := nudgePresentation(c.nudgeType)
		if title != c.wantTitle || dataType != c.wantData {
			t.Errorf("nudgePresentation(%q): want (%q, %q), got (%q, %q)",
				c.nudgeType, c.wantTitle, c.wantData, title, dataType)
		}
	}
}

func TestNudgeScheduler_CheckinNudge_UsesCheckinPresentation(t *testing.T) {
	sched, repo, fcm := newSchedulerFixture()
	userID := uuid.New()
	n := makeNudge(userID, "You asked me to check in about the interview.")
	n.NudgeType = models.NudgeTypeCheckin
	repo.pending = []*models.Nudge{n}
	repo.tokens[userID] = []string{"tok-1"}

	sched.tick(context.Background())

	if fcm.callCount() != 1 {
		t.Fatalf("want 1 FCM call, got %d", fcm.callCount())
	}
	call := fcm.calls[0]
	if call.title != "You asked me to check in" {
		t.Errorf("title: want check-in title, got %q", call.title)
	}
	if call.data["type"] != "checkin" {
		t.Errorf("data.type: want checkin, got %s", call.data["type"])
	}
}
