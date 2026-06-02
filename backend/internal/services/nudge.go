package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
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

	// Schedule for the user's preferred nudge hour tomorrow morning.
	// If that time has already passed today, add another day.
	now := time.Now().In(loc)
	scheduled := time.Date(now.Year(), now.Month(), now.Day(),
		user.FCMNudgeHour, 0, 0, 0, loc)
	if !scheduled.After(now) {
		scheduled = scheduled.Add(24 * time.Hour)
	}

	entryIDPtr := &entryID
	if _, err := s.nudgeRepo.Create(ctx, userID, entryIDPtr, message, scheduled.UTC(), user.Timezone); err != nil {
		return fmt.Errorf("nudge: create: %w", err)
	}
	return nil
}
