package services

import (
	"context"

	"github.com/google/uuid"
)

// analyticsRepo is the minimal DB interface needed to persist events.
type analyticsRepo interface {
	Insert(ctx context.Context, userID *uuid.UUID, eventName string, properties map[string]any) error
}

// AnalyticsService logs product events. All methods are fire-and-forget:
// errors are silently dropped so analytics never affects the happy path.
type AnalyticsService struct {
	repo analyticsRepo
}

func NewAnalyticsService(repo analyticsRepo) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// Track logs an event. userID may be nil for pre-auth events.
// Errors are swallowed — analytics must never break the caller.
func (s *AnalyticsService) Track(ctx context.Context, userID *uuid.UUID, event string, props map[string]any) {
	if s == nil || s.repo == nil {
		return
	}
	if props == nil {
		props = map[string]any{}
	}
	_ = s.repo.Insert(ctx, userID, event, props)
}

// TrackUser is a convenience wrapper for authenticated calls.
func (s *AnalyticsService) TrackUser(ctx context.Context, userID uuid.UUID, event string, props map[string]any) {
	s.Track(ctx, &userID, event, props)
}

// ── Event name constants (from PRICING.md §6c minimum event set) ─────────────

const (
	EventSignup                   = "signup"
	EventOnboardingStepCompleted  = "onboarding_step_completed"
	EventOnboardingCompleted      = "onboarding_completed"
	EventEntryRecorded            = "entry_recorded"
	EventEntryCompleted           = "entry_completed"
	EventEntryFailed              = "entry_failed"
	EventReflectionViewed         = "reflection_viewed"
	EventFollowupStarted          = "followup_started"
	EventFollowupTurn             = "followup_turn"
	EventTherapySessionStarted    = "therapy_session_started"
	EventTherapySessionEnded      = "therapy_session_ended"
	EventPaywallViewed            = "paywall_viewed"
	EventPurchaseInitiated        = "purchase_initiated"
	EventPurchaseCompleted        = "purchase_completed"
	EventPurchaseFailed           = "purchase_failed"
	EventPlanChanged              = "plan_changed"
	EventEntryLimitHit            = "entry_limit_hit"
	EventShareCreated             = "share_created"
	EventInsightCardShared        = "insight_card_shared"
	EventExportDownloaded         = "export_downloaded"
)
