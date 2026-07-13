package repositories

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentRepository records consumed store transactions so a single purchase
// can never grant a plan more than once (replay protection for /billing/upgrade).
type PaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Record inserts the purchase if its store transaction_id has not been seen
// before. Returns false when the transaction was already consumed.
// Amount and currency are store-managed (the store collected the money), so
// the row records provenance only: store + product_id + transaction_id.
func (r *PaymentRepository) Record(ctx context.Context, userID uuid.UUID, transactionID string, plan models.Plan, store, productID string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO payments (user_id, transaction_id, plan, amount, currency, store, product_id)
		VALUES ($1, $2, $3, 0, '', $4, $5)
		ON CONFLICT (transaction_id) DO NOTHING`,
		userID, transactionID, plan, store, productID,
	)
	if err != nil {
		return false, fmt.Errorf("paymentRepo.Record: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
