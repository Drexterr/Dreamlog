package services

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/pkg/monitoring"
	"github.com/google/uuid"
	"go.uber.org/zap"
)


type UserService struct {
	repo    *repositories.UserRepository
	objects objectPrefixDeleter // optional; nil = storage is not cleaned on deletion
	log     *zap.Logger
}

// objectPrefixDeleter erases every stored object under a key prefix.
type objectPrefixDeleter interface {
	DeletePrefix(ctx context.Context, prefix string) (int, error)
}

func NewUserService(repo *repositories.UserRepository, objects objectPrefixDeleter, log *zap.Logger) *UserService {
	if log == nil {
		log = zap.NewNop()
	}
	return &UserService{repo: repo, objects: objects, log: log}
}

// GetOrCreate fetches the user by supabase_id, creating them if they don't exist.
// This is called on every authenticated request via middleware.
func (s *UserService) GetOrCreate(ctx context.Context, supabaseID, email, name string, emailVerified bool) (*models.User, error) {
	user, err := s.repo.Upsert(ctx, supabaseID, email, name, emailVerified)
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
		NudgeAutoTime: input.NudgeAutoTime,
		Goal:          input.Goal,
		AgeRange:      input.AgeRange,
		Country:       input.Country,
		VoiceLanguage: input.VoiceLanguage,
	}
	// Explicitly picking a nudge hour is a manual choice - it wins over the
	// learned typical-hour scheduling unless auto is re-enabled in the same call.
	if input.FCMNudgeHour != nil && input.NudgeAutoTime == nil {
		manual := false
		p.NudgeAutoTime = &manual
	}
	user, err := s.repo.UpdateProfile(ctx, userID, p)
	if err != nil {
		return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
	}
	return user, nil
}

// Delete permanently removes the user and all their data, including the
// objects they left in storage (N7).
func (s *UserService) Delete(ctx context.Context, userID uuid.UUID) error {
	return deleteAccount(ctx, s.repo, s.objects, s.log, userID)
}

// accountStore is what account deletion needs from UserRepository.
type accountStore interface {
	StoragePrefixes(ctx context.Context, userID uuid.UUID) ([]string, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

// deleteAccount deletes the user's rows first - the deletion the user asked
// for must not wait on, or fail because of, object storage - then erases
// their storage prefixes. A storage failure is logged and reported; the
// bucket lifecycle rules remain the backstop for anything left behind.
func deleteAccount(ctx context.Context, repo accountStore, objects objectPrefixDeleter, log *zap.Logger, userID uuid.UUID) error {
	var prefixes []string
	if objects != nil {
		var err error
		if prefixes, err = repo.StoragePrefixes(ctx, userID); err != nil {
			return fmt.Errorf("userService.Delete prefixes: %w", err)
		}
	}
	if err := repo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("userService.Delete: %w", err)
	}
	if log == nil {
		log = zap.NewNop()
	}
	cleanup := context.WithoutCancel(ctx)
	for _, p := range prefixes {
		if _, err := objects.DeletePrefix(cleanup, p); err != nil {
			log.Error("account deletion: storage cleanup failed", zap.String("prefix", p), zap.Error(err))
			monitoring.CaptureErr(err, map[string]string{"component": "account_storage_cleanup"})
		}
	}
	return nil
}
