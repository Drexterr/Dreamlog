package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// RecordResult describes what Record found for a store transaction.
type RecordResult struct {
	Inserted bool      // first time this transaction was seen
	OwnerID  uuid.UUID // user the transaction was recorded for (when !Inserted)
	Granted  bool      // the plan grant for it already committed (when !Inserted)
}

// Record inserts the purchase if its store transaction_id has not been seen
// before. When it has, it reports who owns it and whether its grant already
// committed, so /billing/upgrade can be idempotent for the same user and
// refuse it for anyone else. Amount and currency are store-managed.
func (r *PaymentRepository) Record(ctx context.Context, userID uuid.UUID, transactionID string, plan models.Plan, store, productID string, sandbox bool) (RecordResult, error) {
	env := "production"
	if sandbox {
		env = "sandbox"
	}
	tag, err := r.db.Exec(ctx, `
		INSERT INTO payments (user_id, transaction_id, plan, amount, currency, store, product_id, environment)
		VALUES ($1, $2, $3, 0, '', $4, $5, $6)
		ON CONFLICT (transaction_id) DO NOTHING`,
		userID, transactionID, plan, store, productID, env,
	)
	if err != nil {
		return RecordResult{}, fmt.Errorf("paymentRepo.Record: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return RecordResult{Inserted: true}, nil
	}

	var owner *uuid.UUID
	var grantedAt *time.Time
	if err := r.db.QueryRow(ctx,
		`SELECT user_id, granted_at FROM payments WHERE transaction_id = $1`, transactionID,
	).Scan(&owner, &grantedAt); err != nil {
		return RecordResult{}, fmt.Errorf("paymentRepo.Record lookup: %w", err)
	}
	res := RecordResult{Granted: grantedAt != nil}
	if owner != nil {
		res.OwnerID = *owner
	}
	return res, nil
}

// GrantPlan grants the plan bought by a recorded store transaction exactly
// once (N5). In one transaction it locks the payment row, returns the user
// unchanged (granted=false) if the grant already committed, and otherwise
// sets the plan and marks the payment granted. Concurrent requests for the
// same transaction serialize on the payment row, so one purchase can never
// extend a pass twice. The new expiry is computed under the user row lock: a
// pass for the plan the user already holds extends from the current expiry,
// any other plan starts now.
func (r *PaymentRepository) GrantPlan(ctx context.Context, transactionID string, userID uuid.UUID, plan models.Plan, duration time.Duration) (user *models.User, granted bool, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("paymentRepo.GrantPlan begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var owner *uuid.UUID
	var grantedAt *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT user_id, granted_at FROM payments WHERE transaction_id = $1 FOR UPDATE`, transactionID,
	).Scan(&owner, &grantedAt); err != nil {
		return nil, false, fmt.Errorf("paymentRepo.GrantPlan lock payment: %w", err)
	}
	if owner == nil || *owner != userID {
		return nil, false, ErrPaymentNotOwned
	}

	var currentPlan models.Plan
	var currentExpiry *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT plan, plan_expires_at FROM users WHERE id = $1 FOR UPDATE`, userID,
	).Scan(&currentPlan, &currentExpiry); err != nil {
		return nil, false, fmt.Errorf("paymentRepo.GrantPlan lock user: %w", err)
	}

	if grantedAt == nil {
		now := time.Now()
		base := now
		if currentPlan == plan && currentExpiry != nil && currentExpiry.After(now) {
			base = *currentExpiry
		}
		if _, err := tx.Exec(ctx,
			`UPDATE users SET plan = $2, plan_expires_at = $3, updated_at = NOW() WHERE id = $1`,
			userID, string(plan), base.Add(duration)); err != nil {
			return nil, false, fmt.Errorf("paymentRepo.GrantPlan update plan: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE payments SET granted_at = NOW() WHERE transaction_id = $1`, transactionID); err != nil {
			return nil, false, fmt.Errorf("paymentRepo.GrantPlan mark granted: %w", err)
		}
		granted = true
	}

	user, err = scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, userID))
	if err != nil {
		return nil, false, fmt.Errorf("paymentRepo.GrantPlan read user: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("paymentRepo.GrantPlan commit: %w", err)
	}
	return user, granted, nil
}

// ErrPaymentNotOwned means a transaction is recorded for a different user.
var ErrPaymentNotOwned = errors.New("payment belongs to another user")
