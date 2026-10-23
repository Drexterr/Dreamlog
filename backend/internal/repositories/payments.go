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
	var kind string
	var grantedAt *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT user_id, plan, granted_at FROM payments WHERE transaction_id = $1 FOR UPDATE`, transactionID,
	).Scan(&owner, &kind, &grantedAt); err != nil {
		return nil, false, fmt.Errorf("paymentRepo.GrantPlan lock payment: %w", err)
	}
	if owner == nil || *owner != userID {
		return nil, false, ErrPaymentNotOwned
	}
	if kind == string(models.PaymentItemTherapyCredit) {
		return nil, false, ErrPaymentWrongKind
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

// ErrPaymentWrongKind means a transaction was recorded for something other
// than what is being granted (a plan pass redeemed as therapy credits).
var ErrPaymentWrongKind = errors.New("payment was recorded for a different product kind")

// TherapyCredits returns the user's spendable therapy session credit balance
// (0 for an unknown user).
func (r *PaymentRepository) TherapyCredits(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT therapy_credits FROM users WHERE id = $1`, userID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("paymentRepo.TherapyCredits: %w", err)
	}
	return n, nil
}

// GrantTherapyCredits adds the credits bought by a recorded store transaction
// exactly once - the GrantPlan pattern (N5). In one transaction it locks the
// payment row, refuses a row owned by someone else or recorded for a plan,
// returns the balance unchanged (granted=false) if the grant already
// committed, and otherwise adds the credits, writes the ledger row and marks
// the payment granted. Concurrent replays serialize on the payment row.
func (r *PaymentRepository) GrantTherapyCredits(ctx context.Context, transactionID string, userID uuid.UUID, credits int) (balance int, granted bool, err error) {
	if credits <= 0 {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits: credits must be positive, got %d", credits)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var owner *uuid.UUID
	var kind string
	var grantedAt *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT user_id, plan, granted_at FROM payments WHERE transaction_id = $1 FOR UPDATE`, transactionID,
	).Scan(&owner, &kind, &grantedAt); err != nil {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits lock payment: %w", err)
	}
	if owner == nil || *owner != userID {
		return 0, false, ErrPaymentNotOwned
	}
	if kind != string(models.PaymentItemTherapyCredit) {
		return 0, false, ErrPaymentWrongKind
	}

	if grantedAt != nil {
		if err := tx.QueryRow(ctx, `SELECT therapy_credits FROM users WHERE id = $1`, userID).Scan(&balance); err != nil {
			return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits read balance: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits commit: %w", err)
		}
		return balance, false, nil
	}

	if err := tx.QueryRow(ctx,
		`UPDATE users SET therapy_credits = therapy_credits + $2, updated_at = NOW() WHERE id = $1 RETURNING therapy_credits`,
		userID, credits,
	).Scan(&balance); err != nil {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits add credits: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO therapy_credit_ledger (user_id, delta, reason, transaction_id) VALUES ($1, $2, 'purchase', $3)`,
		userID, credits, transactionID); err != nil {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits ledger: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE payments SET granted_at = NOW() WHERE transaction_id = $1`, transactionID); err != nil {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits mark granted: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, false, fmt.Errorf("paymentRepo.GrantTherapyCredits commit: %w", err)
	}
	return balance, true, nil
}

// DevGrantTherapyCredits adds credits without a store purchase. Only the dev
// stub (no store credentials) calls it - production refuses to start
// without IAP verification configured.
func (r *PaymentRepository) DevGrantTherapyCredits(ctx context.Context, userID uuid.UUID, credits int) (int, error) {
	if credits <= 0 {
		return 0, fmt.Errorf("paymentRepo.DevGrantTherapyCredits: credits must be positive, got %d", credits)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("paymentRepo.DevGrantTherapyCredits begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var balance int
	if err := tx.QueryRow(ctx,
		`UPDATE users SET therapy_credits = therapy_credits + $2, updated_at = NOW() WHERE id = $1 RETURNING therapy_credits`,
		userID, credits,
	).Scan(&balance); err != nil {
		return 0, fmt.Errorf("paymentRepo.DevGrantTherapyCredits add credits: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO therapy_credit_ledger (user_id, delta, reason) VALUES ($1, $2, 'dev_grant')`,
		userID, credits); err != nil {
		return 0, fmt.Errorf("paymentRepo.DevGrantTherapyCredits ledger: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("paymentRepo.DevGrantTherapyCredits commit: %w", err)
	}
	return balance, nil
}
