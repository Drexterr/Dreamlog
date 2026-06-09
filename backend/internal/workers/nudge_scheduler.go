package workers

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// NudgeScheduler polls for due nudges every minute and dispatches them via FCM.
// Run as a separate goroutine alongside the API or worker process.
type NudgeScheduler struct {
	nudgeRepo nudgeDispatcher
	fcm       fcmSender
	log       *zap.Logger
}

func NewNudgeScheduler(
	nudgeRepo nudgeDispatcher,
	fcm fcmSender,
	log *zap.Logger,
) *NudgeScheduler {
	return &NudgeScheduler{nudgeRepo: nudgeRepo, fcm: fcm, log: log}
}

// Run blocks until ctx is cancelled, ticking every minute.
func (s *NudgeScheduler) Run(ctx context.Context) {
	s.log.Info("nudge scheduler starting")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// Fire immediately on start, then every minute.
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("nudge scheduler stopping")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *NudgeScheduler) tick(ctx context.Context) {
	nudges, err := s.nudgeRepo.PendingDue(ctx)
	if err != nil {
		s.log.Error("nudge scheduler: fetch pending", zap.Error(err))
		return
	}
	if len(nudges) == 0 {
		return
	}

	s.log.Info("nudge scheduler: dispatching", zap.Int("count", len(nudges)))

	for _, n := range nudges {
		tokens, err := s.nudgeRepo.GetDeviceTokens(ctx, n.UserID)
		if err != nil {
			s.log.Warn("nudge scheduler: get tokens", zap.String("nudge_id", n.ID.String()), zap.Error(err))
			_ = s.nudgeRepo.MarkFailed(ctx, n.ID, "get tokens: "+err.Error())
			continue
		}

		if len(tokens) == 0 {
			// No device registered - mark sent anyway to avoid repeated attempts.
			_ = s.nudgeRepo.MarkSent(ctx, n.ID)
			continue
		}

		var lastErr error
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, "Your morning reflection", n.Message, map[string]string{
				"type":    "morning_nudge",
				"nudge_id": n.ID.String(),
			}); err != nil {
				lastErr = err
				s.log.Warn("nudge scheduler: send failed",
					zap.String("nudge_id", n.ID.String()),
					zap.String("token", truncateToken(token)),
					zap.Error(err),
				)
			}
		}

		if lastErr != nil {
			_ = s.nudgeRepo.MarkFailed(ctx, n.ID, fmt.Sprintf("send error: %v", lastErr))
		} else {
			_ = s.nudgeRepo.MarkSent(ctx, n.ID)
		}
	}
}

func truncateToken(token string) string {
	if len(token) > 12 {
		return token[:8] + "…"
	}
	return token
}
