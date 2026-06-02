package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyRepository struct {
	db *pgxpool.Pool
}

func NewCompanyRepository(db *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// GetBySlug returns the company with the given slug, or nil if not found.
func (r *CompanyRepository) GetBySlug(ctx context.Context, slug string) (*models.Company, error) {
	const q = `
		SELECT id, name, slug, admin_email, plan, seat_limit, created_at, updated_at
		FROM companies WHERE slug = $1`

	c := &models.Company{}
	err := r.db.QueryRow(ctx, q, slug).Scan(
		&c.ID, &c.Name, &c.Slug, &c.AdminEmail, &c.Plan, &c.SeatLimit, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("company.GetBySlug: %w", err)
	}
	return c, nil
}

// IsMember returns the member row if the user belongs to the company.
func (r *CompanyRepository) IsMember(ctx context.Context, companyID, userID uuid.UUID) (*models.CompanyMember, error) {
	const q = `
		SELECT id, company_id, user_id, role, joined_at
		FROM company_members WHERE company_id = $1 AND user_id = $2`

	m := &models.CompanyMember{}
	err := r.db.QueryRow(ctx, q, companyID, userID).Scan(
		&m.ID, &m.CompanyID, &m.UserID, &m.Role, &m.JoinedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("company.IsMember: %w", err)
	}
	return m, nil
}

// TotalMembers returns the total enrolled member count for a company.
func (r *CompanyRepository) TotalMembers(ctx context.Context, companyID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM company_members WHERE company_id = $1`, companyID,
	).Scan(&n)
	return n, err
}

// TeamMoodHistory returns daily mood rows from v_team_daily_mood for the given window.
func (r *CompanyRepository) TeamMoodHistory(ctx context.Context, companyID uuid.UUID, since, until time.Time) ([]*models.TeamDailyMood, error) {
	const q = `
		SELECT day::TEXT, avg_mood, active_members, entry_count
		FROM v_team_daily_mood
		WHERE company_id = $1 AND day >= $2 AND day < $3
		ORDER BY day ASC`

	rows, err := r.db.Query(ctx, q, companyID, since, until)
	if err != nil {
		return nil, fmt.Errorf("company.TeamMoodHistory: %w", err)
	}
	defer rows.Close()

	var result []*models.TeamDailyMood
	for rows.Next() {
		dm := &models.TeamDailyMood{}
		if err := rows.Scan(&dm.Day, &dm.AvgMood, &dm.ActiveMembers, &dm.EntryCount); err != nil {
			return nil, fmt.Errorf("company.TeamMoodHistory scan: %w", err)
		}
		result = append(result, dm)
	}
	return result, rows.Err()
}

// JoinCompany adds a user as a member of a company (idempotent via ON CONFLICT DO NOTHING).
func (r *CompanyRepository) JoinCompany(ctx context.Context, companyID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO company_members (company_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		companyID, userID,
	)
	if err != nil {
		return fmt.Errorf("company.JoinCompany: %w", err)
	}
	return nil
}
