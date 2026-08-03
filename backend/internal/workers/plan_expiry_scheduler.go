package workers

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"go.uber.org/zap"
)

// planExpiryLocalHour is the local hour at which plan-expiry pushes go out -
// reuses the same hour convention as other daily nudges.
const planExpiryLocalHour = 10

// PlanExpiryScheduler warns users before their paid plan (a one-time IAP pass,
// not an auto-renewing subscription - see docs/API_CONTRACT.md) lapses, and
// nudges them once shortly after it does. Ode has no OS-level renewal
// reminder for this pass model, so the app is the only place this can be
// surfaced.
type PlanExpiryScheduler struct {
	nudgeRepo planExpiryRepo
	fcm       fcmSender
	log       *zap.Logger
}

func NewPlanExpiryScheduler(nudgeRepo planExpiryRepo, fcm fcmSender, log *zap.Logger) *PlanExpiryScheduler {
	return &PlanExpiryScheduler{nudgeRepo: nudgeRepo, fcm: fcm, log: log}
}

// Run blocks until ctx is cancelled, ticking every minute.
func (s *PlanExpiryScheduler) Run(ctx context.Context) {
	s.log.Info("plan expiry scheduler starting")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("plan expiry scheduler stopping")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *PlanExpiryScheduler) tick(ctx context.Context) {
	soon, err := s.nudgeRepo.PlanExpiringSoonUsersAtLocalHour(ctx, planExpiryLocalHour, models.PlanExpiryWarnDays)
	if err != nil {
		s.log.Error("plan expiry scheduler: fetch expiring-soon candidates", zap.Error(err))
	} else {
		for _, u := range soon {
			s.send(ctx, u, models.NudgeTypePlanExpiringSoon,
				"Your plan is about to expire",
				expiringSoonMessage(u.PlanExpires),
			)
		}
	}

	expired, err := s.nudgeRepo.PlanExpiredUsersAtLocalHour(ctx, planExpiryLocalHour)
	if err != nil {
		s.log.Error("plan expiry scheduler: fetch expired candidates", zap.Error(err))
		return
	}
	for _, u := range expired {
		s.send(ctx, u, models.NudgeTypePlanExpired,
			"Your plan has expired",
			"Your Ode plan just lapsed. Renew anytime to pick up right where you left off.",
		)
	}
}

func (s *PlanExpiryScheduler) send(ctx context.Context, u repositories.PlanExpiryUser, nudgeType, title, msg string) {
	// Create the row first so the dedup check on the next tick prevents double-sends.
	nudge, err := s.nudgeRepo.CreateWithType(ctx, u.UserID, nil, msg, time.Now().UTC(), u.Timezone, nudgeType)
	if err != nil {
		s.log.Warn("plan expiry scheduler: create nudge row",
			zap.String("user_id", u.UserID.String()), zap.String("nudge_type", nudgeType), zap.Error(err))
		return
	}

	tokens, err := s.nudgeRepo.GetDeviceTokens(ctx, u.UserID)
	if err != nil {
		_ = s.nudgeRepo.MarkFailed(ctx, nudge.ID, "get tokens: "+err.Error())
		return
	}
	if len(tokens) == 0 {
		_ = s.nudgeRepo.MarkSent(ctx, nudge.ID)
		return
	}

	var lastErr error
	for _, token := range tokens {
		if err := s.fcm.SendToToken(ctx, token, title, msg, map[string]string{
			"type":     nudgeType,
			"nudge_id": nudge.ID.String(),
		}); err != nil {
			lastErr = err
			s.log.Warn("plan expiry scheduler: send failed",
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

// expiringSoonMessage reports how many days remain, rounded (not floored) to
// the nearest whole day so e.g. 20 hours away reads as "tomorrow" rather than
// "today", regardless of exactly when within the warning window the tick fires.
func expiringSoonMessage(expiresAt time.Time) string {
	days := int(math.Round(time.Until(expiresAt).Hours() / 24))
	switch {
	case days <= 0:
		return "Your Ode plan expires today. Renew to keep your streak of features going."
	case days == 1:
		return "Your Ode plan expires tomorrow. Renew to keep everything unlocked."
	default:
		return fmt.Sprintf("Your Ode plan expires in %d days. Renew to keep everything unlocked.", days)
	}
}
