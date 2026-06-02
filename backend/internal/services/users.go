package services

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
)


type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// GetOrCreate fetches the user by supabase_id, creating them if they don't exist.
// This is called on every authenticated request via middleware.
func (s *UserService) GetOrCreate(ctx context.Context, supabaseID, email, name string) (*models.User, error) {
	user, err := s.repo.Upsert(ctx, supabaseID, email, name)
	if err != nil {
		return nil, fmt.Errorf("userService.GetOrCreate: %w", err)
	}
	return user, nil
}

// GetMe returns the current user by internal ID.
func (s *UserService) GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userService.GetMe: %w", err)
	}
	return user, nil
}

// UpdateName sets the user's display name.
func (s *UserService) UpdateName(ctx context.Context, userID uuid.UUID, name string) (*models.User, error) {
	user, err := s.repo.UpdateName(ctx, userID, name)
	if err != nil {
		return nil, fmt.Errorf("userService.UpdateName: %w", err)
	}
	return user, nil
}

// UpdateProfile applies whichever fields are non-nil.
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, input models.UpdateUserInput) (*models.User, error) {
	p := repositories.ProfileUpdate{
		Name:          input.Name,
		PreferredName: input.PreferredName,
		Timezone:      input.Timezone,
		FCMNudgeHour:  input.FCMNudgeHour,
		NudgeEnabled:  input.NudgeEnabled,
		Goal:          input.Goal,
		AgeRange:      input.AgeRange,
	}
	user, err := s.repo.UpdateProfile(ctx, userID, p)
	if err != nil {
		return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
	}
	return user, nil
}

// Delete permanently removes the user and all their data.
func (s *UserService) Delete(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("userService.Delete: %w", err)
	}
	return nil
}
