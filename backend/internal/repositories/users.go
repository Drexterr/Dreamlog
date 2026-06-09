package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, supabase_id, email, name, timezone, fcm_nudge_hour, nudge_enabled, goal, preferred_name, streak_freeze_count, plan, plan_expires_at, age_range, country, is_deleted, deleted_at, first_joined_at, reregistered_at, reregistration_count, created_at, updated_at`

// rowScanner is satisfied by both pgx.Row and pgx.Rows - avoids a direct pgx type in the signature.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(
		&u.ID, &u.SupabaseID, &u.Email, &u.Name,
		&u.Timezone, &u.FCMNudgeHour, &u.NudgeEnabled,
		&u.Goal, &u.PreferredName,
		&u.StreakFreezeCount,
		&u.Plan, &u.PlanExpiresAt,
		&u.AgeRange, &u.Country,
		&u.IsDeleted, &u.DeletedAt, &u.FirstJoinedAt, &u.ReregisteredAt, &u.ReregistrationCount,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ProfileUpdate holds optional fields for PUT /me.
type ProfileUpdate struct {
	Name          *string
	PreferredName *string
	Timezone      *string
	FCMNudgeHour  *int
	NudgeEnabled  *bool
	Goal          *string
	AgeRange      *string
	Country       *string
}

// UpdateProfile applies whichever fields are non-nil in a single UPDATE.
func (r *UserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, p ProfileUpdate) (*models.User, error) {
	setClauses := []string{}
	args := []any{id} // $1 = user id
	idx := 2

	if p.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", idx))
		args = append(args, *p.Name)
		idx++
	}
	if p.PreferredName != nil {
		setClauses = append(setClauses, fmt.Sprintf("preferred_name = $%d", idx))
		args = append(args, *p.PreferredName)
		idx++
	}
	if p.Timezone != nil {
		setClauses = append(setClauses, fmt.Sprintf("timezone = $%d", idx))
		args = append(args, *p.Timezone)
		idx++
	}
	if p.FCMNudgeHour != nil {
		setClauses = append(setClauses, fmt.Sprintf("fcm_nudge_hour = $%d", idx))
		args = append(args, *p.FCMNudgeHour)
		idx++
	}
	if p.NudgeEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("nudge_enabled = $%d", idx))
		args = append(args, *p.NudgeEnabled)
		idx++
	}
	if p.Goal != nil {
		setClauses = append(setClauses, fmt.Sprintf("goal = $%d", idx))
		args = append(args, *p.Goal)
		idx++
	}
	if p.AgeRange != nil {
		setClauses = append(setClauses, fmt.Sprintf("age_range = $%d", idx))
		args = append(args, *p.AgeRange)
		idx++
	}
	if p.Country != nil {
		setClauses = append(setClauses, fmt.Sprintf("country = $%d", idx))
		args = append(args, *p.Country)
		idx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	q := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $1 RETURNING %s",
		strings.Join(setClauses, ", "),
		userColumns,
	)

	u, err := scanUser(r.db.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.UpdateProfile: %w", err)
	}
	return u, nil
}

// GetBySupabaseID fetches a user by their Supabase UUID.
func (r *UserRepository) GetBySupabaseID(ctx context.Context, supabaseID string) (*models.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE supabase_id = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, supabaseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.GetBySupabaseID: %w", err)
	}
	return u, nil
}

// GetByID fetches a user by internal UUID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.GetByID: %w", err)
	}
	return u, nil
}

// Upsert creates or updates a user record (idempotent on supabase_id).
func (r *UserRepository) Upsert(ctx context.Context, supabaseID, email, name string) (*models.User, error) {
	q := `
		INSERT INTO users (supabase_id, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (supabase_id) DO UPDATE
			SET email = EXCLUDED.email,
			    updated_at = NOW()
		RETURNING ` + userColumns

	u, err := scanUser(r.db.QueryRow(ctx, q, supabaseID, email, name))
	if err != nil {
		return nil, fmt.Errorf("users.Upsert: %w", err)
	}
	return u, nil
}

// UpdateName sets the display name for a user.
func (r *UserRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) (*models.User, error) {
	q := `
		UPDATE users SET name = $2
		WHERE id = $1
		RETURNING ` + userColumns

	u, err := scanUser(r.db.QueryRow(ctx, q, id, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.UpdateName: %w", err)
	}
	return u, nil
}

// GetByEmail fetches an active (non-deleted) user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE email = $1 AND is_deleted = false`
	u, err := scanUser(r.db.QueryRow(ctx, q, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.GetByEmail: %w", err)
	}
	return u, nil
}

// GetByEmailIncDeleted fetches a user by email regardless of deletion status.
// Used during re-registration to detect and reactivate soft-deleted accounts.
func (r *UserRepository) GetByEmailIncDeleted(ctx context.Context, email string) (*models.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.GetByEmailIncDeleted: %w", err)
	}
	return u, nil
}

// Reactivate restores a soft-deleted account for the given user ID and updates credentials.
func (r *UserRepository) Reactivate(ctx context.Context, id uuid.UUID, name, passwordHash string) (*models.User, error) {
	q := `
		UPDATE users
		SET is_deleted           = false,
		    deleted_at           = NULL,
		    name                 = $2,
		    password_hash        = $3,
		    reregistered_at      = NOW(),
		    reregistration_count = reregistration_count + 1,
		    updated_at           = NOW()
		WHERE id = $1
		RETURNING ` + userColumns

	u, err := scanUser(r.db.QueryRow(ctx, q, id, name, passwordHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.Reactivate: %w", err)
	}
	return u, nil
}

// CreateLocal creates a user with a bcrypt password hash for local auth.
func (r *UserRepository) CreateLocal(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	supabaseID := "local-" + uuid.New().String()
	q := `
		INSERT INTO users (supabase_id, email, name, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + userColumns

	u, err := scanUser(r.db.QueryRow(ctx, q, supabaseID, email, name, passwordHash))
	if err != nil {
		return nil, fmt.Errorf("users.CreateLocal: %w", err)
	}
	return u, nil
}

// GetPasswordHash returns the stored bcrypt hash for the given email.
func (r *UserRepository) GetPasswordHash(ctx context.Context, email string) (string, error) {
	var hash string
	err := r.db.QueryRow(ctx, `SELECT COALESCE(password_hash, '') FROM users WHERE email = $1`, email).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("users.GetPasswordHash: %w", err)
	}
	return hash, nil
}

// ListWithRecentEntries returns distinct users who have had at least one completed entry
// since the given time. Used by the weekly review scheduler.
func (r *UserRepository) ListWithRecentEntries(ctx context.Context, since time.Time) ([]*models.User, error) {
	q := `
		SELECT DISTINCT ON (u.id) ` + userColumns + `
		FROM users u
		JOIN entries e ON e.user_id = u.id
		WHERE e.status = 'completed' AND e.created_at >= $1
		ORDER BY u.id`

	rows, err := r.db.Query(ctx, q, since)
	if err != nil {
		return nil, fmt.Errorf("users.ListWithRecentEntries: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("users.ListWithRecentEntries scan: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// StreakFreezeCount returns the user's remaining streak freeze count.
func (r *UserRepository) StreakFreezeCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT streak_freeze_count FROM users WHERE id = $1`, userID).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("users.StreakFreezeCount: %w", err)
	}
	return count, nil
}

// UseStreakFreeze inserts a freeze day and decrements the user's freeze count atomically.
// Returns error if the user has no freezes remaining.
func (r *UserRepository) UseStreakFreeze(ctx context.Context, userID uuid.UUID, frozenDate time.Time) error {
	// Decrement count and record the freeze day in one transaction.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users.UseStreakFreeze begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx,
		`UPDATE users SET streak_freeze_count = streak_freeze_count - 1
		 WHERE id = $1 AND streak_freeze_count > 0
		 RETURNING streak_freeze_count`, userID,
	).Scan(&count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("no streak freezes remaining")
		}
		return fmt.Errorf("users.UseStreakFreeze update: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO streak_freeze_days (user_id, frozen_date) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, frozenDate.Format("2006-01-02"),
	)
	if err != nil {
		return fmt.Errorf("users.UseStreakFreeze insert: %w", err)
	}

	return tx.Commit(ctx)
}

// GrantWeeklyFreezes increments streak_freeze_count (up to 3) for all users who haven't
// received their automatic weekly freeze this week yet. Safe to call idempotently.
func (r *UserRepository) GrantWeeklyFreezes(ctx context.Context) error {
	const q = `
		UPDATE users
		SET streak_freeze_count      = LEAST(streak_freeze_count + 1, 3),
		    streak_freeze_granted_week = DATE_TRUNC('week', CURRENT_DATE)::DATE
		WHERE COALESCE(streak_freeze_granted_week, '1970-01-01'::DATE)
		          < DATE_TRUNC('week', CURRENT_DATE)::DATE
		  AND streak_freeze_count < 3`
	_, err := r.db.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("users.GrantWeeklyFreezes: %w", err)
	}
	return nil
}

// UpdatePlan sets the user's subscription plan and optional expiry.
func (r *UserRepository) UpdatePlan(ctx context.Context, id uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error) {
	q := `
		UPDATE users SET plan = $2, plan_expires_at = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + userColumns

	u, err := scanUser(r.db.QueryRow(ctx, q, id, string(plan), expiresAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.UpdatePlan: %w", err)
	}
	return u, nil
}

// Delete soft-deletes the user by setting is_deleted=true and recording deleted_at.
// Data is retained for audit and re-registration purposes.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users SET is_deleted = true, deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND is_deleted = false`,
		id,
	)
	if err != nil {
		return fmt.Errorf("users.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("users.Delete: not found or already deleted")
	}
	return nil
}

// CountMonthlyEntries returns the number of entries created by the user in the current calendar month.
func (r *UserRepository) CountMonthlyEntries(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM entries
		WHERE user_id = $1
		  AND DATE_TRUNC('month', created_at AT TIME ZONE 'UTC') = DATE_TRUNC('month', NOW() AT TIME ZONE 'UTC')`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("users.CountMonthlyEntries: %w", err)
	}
	return count, nil
}

// UpdatePreferences sets timezone and nudge hour.
func (r *UserRepository) UpdatePreferences(ctx context.Context, id uuid.UUID, timezone string, nudgeHour int) (*models.User, error) {
	q := `
		UPDATE users SET timezone = $2, fcm_nudge_hour = $3
		WHERE id = $1
		RETURNING ` + userColumns

	u, err := scanUser(r.db.QueryRow(ctx, q, id, timezone, nudgeHour))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.UpdatePreferences: %w", err)
	}
	return u, nil
}
