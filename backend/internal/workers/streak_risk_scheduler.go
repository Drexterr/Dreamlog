package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"go.uber.org/zap"
)

// streakRiskLocalHour is the local evening hour at which streak-at-risk pushes
// go out - late enough that the day is clearly slipping away, early enough
// that recording is still realistic.
const streakRiskLocalHour = 21

// minStreakForRiskNudge avoids nagging brand-new users: only streaks of this
// length or more get an at-risk push.
const minStreakForRiskNudge = 3

// StreakRiskScheduler finds users whose active streak has no entry yet today
// and sends a loss-aversion (but non-guilt) push at 21:00 local time.
type StreakRiskScheduler struct {
	nudgeRepo streakRiskRepo
	streaks   streakLookup
	fcm       fcmSender
	log       *zap.Logger
}

func NewStreakRiskScheduler(nudgeRepo streakRiskRepo, streaks streakLookup, fcm fcmSender, log *zap.Logger) *StreakRiskScheduler {
	return &StreakRiskScheduler{nudgeRepo: nudgeRepo, streaks: streaks, fcm: fcm, log: log}
}

// Run blocks until ctx is cancelled, ticking every minute.
func (s *StreakRiskScheduler) Run(ctx context.Context) {
	s.log.Info("streak risk scheduler starting")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("streak risk scheduler stopping")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *StreakRiskScheduler) tick(ctx context.Context) {
	users, err := s.nudgeRepo.StreakRiskUsersAtLocalHour(ctx, streakRiskLocalHour)
	if err != nil {
		s.log.Error("streak risk scheduler: fetch candidates", zap.Error(err))
		return
	}
	if len(users) == 0 {
		return
	}

	s.log.Info("streak risk scheduler: candidates", zap.Int("count", len(users)))

	for _, u := range users {
		streak, err := s.streaks.StreakInfo(ctx, u.UserID)
		if err != nil {
			s.log.Warn("streak risk scheduler: streak info",
				zap.String("user_id", u.UserID.String()), zap.Error(err))
			continue
		}
		if streak.CurrentStreak < minStreakForRiskNudge {
			continue
		}

		msg := fmt.Sprintf(
			"%d days of showing up for yourself. One quiet minute tonight keeps it going.",
			streak.CurrentStreak,
		)

		// Create the row first so the 20-hour dedup check prevents double-sends.
		nudge, err := s.nudgeRepo.CreateWithType(ctx, u.UserID, nil, msg, time.Now().UTC(), u.Timezone, models.NudgeTypeStreakRisk)
		if err != nil {
			s.log.Warn("streak risk scheduler: create nudge row",
				zap.String("user_id", u.UserID.String()), zap.Error(err))
			continue
		}

		tokens, err := s.nudgeRepo.GetDeviceTokens(ctx, u.UserID)
		if err != nil {
			_ = s.nudgeRepo.MarkFailed(ctx, nudge.ID, "get tokens: "+err.Error())
			continue
		}
		if len(tokens) == 0 {
			_ = s.nudgeRepo.MarkSent(ctx, nudge.ID)
			continue
		}

		var lastErr error
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, "Your streak is waiting", msg, map[string]string{
				"type":     "streak_risk",
				"nudge_id": nudge.ID.String(),
			}); err != nil {
				lastErr = err
				s.log.Warn("streak risk scheduler: send failed",
					zap.String("nudge_id", nudge.ID.String()),
					zap.String("token", truncateToken(token)),
					zap.Error(err),
				)
			}
		}

		if lastErr != nil {
			_ = s.nudgeRepo.MarkFailed(ctx, nudge.ID, fmt.Sprintf("send error: %v", lastErr))
		} else {
			_ = s.nudgeRepo.MarkSent(ctx, nudge.ID)
		}
	}
}
