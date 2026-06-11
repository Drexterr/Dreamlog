package repositories

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentRepository records consumed payment intents so a single payment can
// never grant a plan more than once (replay protection for /billing/upgrade).
type PaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Record inserts the payment if its payment_intent_id has not been seen
// before. Returns false when the intent was already consumed.
func (r *PaymentRepository) Record(ctx context.Context, userID uuid.UUID, paymentIntentID string, plan models.Plan, amount int64, currency string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO payments (user_id, payment_intent_id, plan, amount, currency)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (payment_intent_id) DO NOTHING`,
		userID, paymentIntentID, plan, amount, currency,
	)
	if err != nil {
		return false, fmt.Errorf("paymentRepo.Record: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
