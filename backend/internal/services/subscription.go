package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// subscriptionUserRepo is the DB contract SubscriptionService needs.
type subscriptionUserRepo interface {
	UpdatePlan(ctx context.Context, id uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error)
	CountMonthlyEntries(ctx context.Context, userID uuid.UUID) (int, error)
}

// subscriptionShareRepo is the share quota contract.
type subscriptionShareRepo interface {
	CountMonthlyByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// SubscriptionService handles plan gating and quota enforcement.
type SubscriptionService struct {
	userRepo  subscriptionUserRepo
	shareRepo subscriptionShareRepo
}

func NewSubscriptionService(userRepo subscriptionUserRepo, shareRepo subscriptionShareRepo) *SubscriptionService {
	return &SubscriptionService{userRepo: userRepo, shareRepo: shareRepo}
}

// GetPlanDetails returns the limits for a plan.
func (s *SubscriptionService) GetPlanDetails(plan models.Plan) *models.PlanLimits {
	return models.GetPlanLimits(plan)
}

// UpgradePlan sets the user's plan in the DB and returns the updated user.
func (s *SubscriptionService) UpgradePlan(ctx context.Context, userID uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error) {
	user, err := s.userRepo.UpdatePlan(ctx, userID, plan, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("subscriptionService.UpgradePlan: %w", err)
	}
	return user, nil
}

// CheckEntryQuota returns a non-nil error when the user has reached their monthly entry limit.
// Non-free plans have no limit.
func (s *SubscriptionService) CheckEntryQuota(ctx context.Context, userID uuid.UUID, plan models.Plan) error {
	if plan != models.PlanFree {
		return nil
	}
	count, err := s.userRepo.CountMonthlyEntries(ctx, userID)
	if err != nil {
		return fmt.Errorf("subscriptionService.CheckEntryQuota: %w", err)
	}
	if count >= models.FreeMonthlyEntries {
		return errEntryQuotaExceeded
	}
	return nil
}

// CheckShareQuota returns a non-nil error when the user cannot create another share link.
// Free: not allowed. Plus: 5/month. Pro/B2B: unlimited.
func (s *SubscriptionService) CheckShareQuota(ctx context.Context, userID uuid.UUID, plan models.Plan) error {
	switch plan {
	case models.PlanPro, models.PlanB2B:
		return nil
	case models.PlanPlus:
		count, err := s.shareRepo.CountMonthlyByUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("subscriptionService.CheckShareQuota: %w", err)
		}
		if count >= models.PlusMonthlyShares {
			return errShareQuotaExceeded
		}
		return nil
	default: // PlanFree
		return errShareNotAllowed
	}
}

// sentinel errors — checked by handlers via errors.Is.
var (
	errEntryQuotaExceeded = fmt.Errorf("monthly entry limit reached; upgrade to DreamLog+ for unlimited entries")
	errShareNotAllowed    = fmt.Errorf("therapist share links require DreamLog+ or higher")
	errShareQuotaExceeded = fmt.Errorf("monthly share link limit reached; upgrade to DreamLog Pro for unlimited shares")
)

// IsEntryQuotaExceeded returns true for the free-plan entry limit error.
func IsEntryQuotaExceeded(err error) bool { return err == errEntryQuotaExceeded }

// IsShareError returns true for any share quota / plan error.
func IsShareError(err error) bool {
	return err == errShareNotAllowed || err == errShareQuotaExceeded
}
