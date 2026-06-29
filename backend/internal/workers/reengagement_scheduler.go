package workers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// reengagementMessages is a pool of warm, non-guilt re-engagement nudge messages.
// One is chosen pseudo-randomly per user per day so repeat sends feel varied.
var reengagementMessages = []string{
	"How's your day going? Even 30 seconds of speaking your mind helps.",
	"Your journal is open whenever you're ready. What's on your mind?",
	"It's been a little while. No pressure — just an open door.",
	"What's one thing you're sitting with today? Take a moment to say it aloud.",
	"A quiet minute for your thoughts. Your last reflection is still here.",
	"Even a quick voice note counts. What's alive for you right now?",
	"Your journal misses you. What's been happening?",
}

// ReengagementScheduler finds users who haven't journaled in a while and
// sends them a gentle re-engagement push at their configured nudge hour.
type ReengagementScheduler struct {
	nudgeRepo  reengagementRepo
	fcm        fcmSender
	lapseHours int // how many hours without an entry before re-engaging
	log        *zap.Logger
}

func NewReengagementScheduler(nudgeRepo reengagementRepo, fcm fcmSender, log *zap.Logger) *ReengagementScheduler {
	return &ReengagementScheduler{
		nudgeRepo:  nudgeRepo,
		fcm:        fcm,
		lapseHours: 26, // more than 24h to avoid colliding with morning nudge on the day of last entry
		log:        log,
	}
}

// Run blocks until ctx is cancelled, ticking every minute.
func (s *ReengagementScheduler) Run(ctx context.Context) {
	s.log.Info("reengagement scheduler starting")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("reengagement scheduler stopping")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *ReengagementScheduler) tick(ctx context.Context) {
	users, err := s.nudgeRepo.LapsedUsersAtNudgeHour(ctx, s.lapseHours)
	if err != nil {
		s.log.Error("reengagement scheduler: fetch lapsed users", zap.Error(err))
		return
	}
	if len(users) == 0 {
		return
	}

	s.log.Info("reengagement scheduler: dispatching", zap.Int("count", len(users)))

	for _, u := range users {
		msg := pickMessage(u.UserID.String())

		// Create a nudge row now so the 23-hour dedup check prevents double-sends.
		nudge, err := s.nudgeRepo.CreateWithType(ctx, u.UserID, nil, msg, time.Now().UTC(), u.Timezone, "reengagement")
		if err != nil {
			s.log.Warn("reengagement scheduler: create nudge row",
				zap.String("user_id", u.UserID.String()),
				zap.Error(err),
			)
			continue
		}

		tokens, err := s.nudgeRepo.GetDeviceTokens(ctx, u.UserID)
		if err != nil {
			s.log.Warn("reengagement scheduler: get tokens",
				zap.String("nudge_id", nudge.ID.String()),
				zap.Error(err),
			)
			_ = s.nudgeRepo.MarkFailed(ctx, nudge.ID, "get tokens: "+err.Error())
			continue
		}
		if len(tokens) == 0 {
			_ = s.nudgeRepo.MarkSent(ctx, nudge.ID)
			continue
		}

		var lastErr error
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, "DreamLog", msg, map[string]string{
				"type":     "reengagement",
				"nudge_id": nudge.ID.String(),
			}); err != nil {
				lastErr = err
				s.log.Warn("reengagement scheduler: send failed",
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

// pickMessage selects a message from the pool using the user ID + today's date
// as a seed so the same user gets the same message all day but different ones on
// different days.
func pickMessage(userID string) string {
	seed := int64(0)
	for _, c := range userID {
		seed = seed*31 + int64(c)
	}
	today := time.Now().Format("20060102")
	for _, c := range today {
		seed = seed*31 + int64(c)
	}
	r := rand.New(rand.NewSource(seed)) //nolint:gosec // not security-sensitive
	return reengagementMessages[r.Intn(len(reengagementMessages))]
}
