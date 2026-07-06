package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
)

// Typed errors surfaced to handlers.
var (
	ErrNudgesDisabled = errors.New("nudges are disabled for this user")
	ErrCheckinExists  = errors.New("a check-in is already scheduled for this entry")
)

// NudgeService schedules morning nudges based on user timezone preference.
type NudgeService struct {
	nudgeRepo *repositories.NudgeRepository
	userRepo  *repositories.UserRepository
}

func NewNudgeService(nudgeRepo *repositories.NudgeRepository, userRepo *repositories.UserRepository) *NudgeService {
	return &NudgeService{nudgeRepo: nudgeRepo, userRepo: userRepo}
}

// ScheduleMorningNudge computes the next morning delivery time in the user's
// timezone and persists the nudge record. The cron scheduler sends it later.
func (s *NudgeService) ScheduleMorningNudge(ctx context.Context, userID, entryID uuid.UUID, message string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("nudge: fetch user: %w", err)
	}
	if !user.NudgeEnabled {
		return nil
	}

	loc, err := time.LoadLocation(user.Timezone)
	if err != nil {
		loc = time.UTC
	}

	scheduled := nextOccurrenceOfHour(time.Now().In(loc), s.nudgeHourFor(ctx, user, loc))

	entryIDPtr := &entryID
	if _, err := s.nudgeRepo.Create(ctx, userID, entryIDPtr, message, scheduled.UTC(), user.Timezone); err != nil {
		return fmt.Errorf("nudge: create: %w", err)
	}
	return nil
}

// ScheduleCheckinNudge schedules a user-requested "check in on this tomorrow"
// nudge for the entry, delivered at the user's nudge hour tomorrow. Returns the
// scheduled time. One pending check-in per entry.
func (s *NudgeService) ScheduleCheckinNudge(ctx context.Context, userID, entryID uuid.UUID, message string) (time.Time, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return time.Time{}, fmt.Errorf("nudge: fetch user: %w", err)
	}
	if !user.NudgeEnabled {
		return time.Time{}, ErrNudgesDisabled
	}

	exists, err := s.nudgeRepo.HasPendingCheckinForEntry(ctx, entryID)
	if err != nil {
		return time.Time{}, fmt.Errorf("nudge: checkin dedup: %w", err)
	}
	if exists {
		return time.Time{}, ErrCheckinExists
	}

	loc, err := time.LoadLocation(user.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// "Check in tomorrow" means tomorrow - never later the same day.
	now := time.Now().In(loc)
	tomorrow := now.AddDate(0, 0, 1)
	scheduled := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
		s.nudgeHourFor(ctx, user, loc), 0, 0, 0, loc)

	entryIDPtr := &entryID
	if _, err := s.nudgeRepo.CreateWithType(ctx, userID, entryIDPtr, message, scheduled.UTC(), user.Timezone, models.NudgeTypeCheckin); err != nil {
		return time.Time{}, fmt.Errorf("nudge: create checkin: %w", err)
	}
	return scheduled, nil
}

// CheckinMessage composes the push body for a user-requested check-in from the
// entry's analysis. Falls back to a generic caring line when no analysis exists.
func CheckinMessage(analysis *models.EntryAnalysis) string {
	if analysis != nil {
		if analysis.MorningNudge != "" {
			return analysis.MorningNudge
		}
		if len(analysis.Topics) > 0 {
			return fmt.Sprintf("You asked me to check in about %s. How did it go?", analysis.Topics[0])
		}
	}
	return "You asked me to check in on yesterday's entry. How are you feeling about it today?"
}

// nudgeHourFor resolves the delivery hour: the user's learned typical recording
// hour when auto-timing is on and the signal is strong, else the configured
// fcm_nudge_hour. Errors in the learning query are non-fatal (fallback applies).
func (s *NudgeService) nudgeHourFor(ctx context.Context, user *models.User, loc *time.Location) int {
	if user.NudgeAutoTime {
		if hour, ok, err := s.nudgeRepo.TypicalEntryHour(ctx, user.ID, loc.String()); err == nil && ok {
			return hour
		}
	}
	return user.FCMNudgeHour
}

// nextOccurrenceOfHour returns the next time the given local hour occurs
// strictly after now (today if still ahead, else tomorrow).
func nextOccurrenceOfHour(now time.Time, hour int) time.Time {
	scheduled := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !scheduled.After(now) {
		scheduled = scheduled.Add(24 * time.Hour)
	}
	return scheduled
}
