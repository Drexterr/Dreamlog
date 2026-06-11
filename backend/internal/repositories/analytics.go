package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalyticsRepository writes product analytics events.
// All writes are append-only — never UPDATE or DELETE rows.
type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// Insert appends one event. userID may be nil for anonymous events.
func (r *AnalyticsRepository) Insert(ctx context.Context, userID *uuid.UUID, eventName string, properties map[string]any) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO analytics_events (user_id, event_name, properties)
		 VALUES ($1, $2, $3)`,
		userID, eventName, properties,
	)
	if err != nil {
		return fmt.Errorf("analyticsRepo.Insert: %w", err)
	}
	return nil
}
