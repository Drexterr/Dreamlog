//go:build integration

package repositories

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func creditBalance(t *testing.T, db *pgxpool.Pool, user uuid.UUID) int {
	t.Helper()
	n, err := NewPaymentRepository(db).TherapyCredits(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func ledgerSum(t *testing.T, db *pgxpool.Pool, user uuid.UUID) (sum, rows int) {
	t.Helper()
	if err := db.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(delta),0), COUNT(*) FROM therapy_credit_ledger WHERE user_id = $1`, user,
	).Scan(&sum, &rows); err != nil {
		t.Fatal(err)
	}
	return sum, rows
}

func recordCreditPurchase(t *testing.T, r *PaymentRepository, user uuid.UUID, tx string) {
	t.Helper()
	if _, err := r.Record(context.Background(), user, tx, models.PaymentItemTherapyCredit, "google", "com.talktoode.app.therapy.single", false); err != nil {
		t.Fatal(err)
	}
}

// Concurrent same-user replays of one credit purchase credit it exactly once.
func TestIntegration_TherapyCredits_ConcurrentReplayGrantsOnce(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewPaymentRepository(db)
	recordCreditPurchase(t, r, user, "GPA.credit-race")

	const n = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	grants := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bal, granted, err := r.GrantTherapyCredits(ctx, "GPA.credit-race", user, 1)
			if err != nil || bal != 1 {
				t.Errorf("grant: balance=%d %v", bal, err)
				return
			}
			if granted {
				mu.Lock()
				grants++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if grants != 1 {
		t.Fatalf("one purchase credited %d times, want 1", grants)
	}
	if got := creditBalance(t, db, user); got != 1 {
		t.Fatalf("balance %d, want 1", got)
	}
	if sum, rows := ledgerSum(t, db, user); sum != 1 || rows != 1 {
		t.Fatalf("ledger sum=%d rows=%d, want 1/1", sum, rows)
	}
	var grantedAt *string
	if err := db.QueryRow(ctx, `SELECT granted_at::text FROM payments WHERE transaction_id = 'GPA.credit-race'`).Scan(&grantedAt); err != nil || grantedAt == nil {
		t.Fatalf("payment must be marked granted: %v", err)
	}
}

func TestIntegration_TherapyCredits_GrantRejectsOtherOwnerAndPlanPayments(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewPaymentRepository(db)
	recordCreditPurchase(t, r, owner, "GPA.credit-owner")
	if _, _, err := r.GrantTherapyCredits(ctx, "GPA.credit-owner", other, 1); !errors.Is(err, ErrPaymentNotOwned) {
		t.Fatalf("want ErrPaymentNotOwned, got %v", err)
	}

	// A plan pass can never be redeemed as credits, nor a credit as a plan.
	if _, err := r.Record(ctx, owner, "GPA.plan-pass", models.PlanPlus, "google", "com.talktoode.app.plus.monthly", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.GrantTherapyCredits(ctx, "GPA.plan-pass", owner, 1); !errors.Is(err, ErrPaymentWrongKind) {
		t.Fatalf("plan payment as credits: want ErrPaymentWrongKind, got %v", err)
	}
	if _, _, err := r.GrantPlan(ctx, "GPA.credit-owner", owner, models.PlanPro, testMonth); !errors.Is(err, ErrPaymentWrongKind) {
		t.Fatalf("credit payment as plan: want ErrPaymentWrongKind, got %v", err)
	}
	if creditBalance(t, db, owner) != 0 || creditBalance(t, db, other) != 0 {
		t.Fatal("nothing may be credited")
	}
	u, err := NewUserRepository(db).GetByID(ctx, owner)
	if err != nil || u.Plan != models.PlanFree {
		t.Fatalf("owner plan must be unchanged: %+v %v", u, err)
	}
}

func TestIntegration_TherapyCredits_DevGrantAndLedger(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	user := regressionUser(t, db)
	r := NewPaymentRepository(db)
	bal, err := r.DevGrantTherapyCredits(context.Background(), user, 2)
	if err != nil || bal != 2 {
		t.Fatalf("dev grant: %d %v", bal, err)
	}
	if _, err := r.DevGrantTherapyCredits(context.Background(), user, 0); err == nil {
		t.Fatal("non-positive grants must be refused")
	}
	if sum, rows := ledgerSum(t, db, user); sum != 2 || rows != 1 {
		t.Fatalf("ledger sum=%d rows=%d", sum, rows)
	}
	if got, err := r.TherapyCredits(context.Background(), uuid.New()); err != nil || got != 0 {
		t.Fatalf("unknown user balance: %d %v", got, err)
	}
}

// Concurrent session starts never spend more credits than sessions created,
// and never overdraw the balance.
func TestIntegration_TherapyCredits_ConcurrentConsumption(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	if _, err := NewPaymentRepository(db).DevGrantTherapyCredits(ctx, user, 3); err != nil {
		t.Fatal(err)
	}
	repo := NewTherapyRepository(db)

	const n = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	created, refused := 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := repo.CreateWithCredit(ctx, user, models.PersonaComforting, models.TherapyContextSnapshot{}, models.TherapySessionPricePaise)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && s != nil:
				created++
			case errors.Is(err, models.ErrNoTherapyCredits):
				refused++
			default:
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	if created != 3 || refused != n-3 {
		t.Fatalf("created=%d refused=%d, want 3/%d", created, refused, n-3)
	}
	if got := creditBalance(t, db, user); got != 0 {
		t.Fatalf("balance %d, want 0", got)
	}
	var sessions int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM therapy_sessions WHERE user_id = $1`, user).Scan(&sessions); err != nil || sessions != 3 {
		t.Fatalf("sessions=%d %v, want 3", sessions, err)
	}
	if sum, rows := ledgerSum(t, db, user); sum != 0 || rows != 4 {
		t.Fatalf("ledger sum=%d rows=%d, want 0/4 (one grant, three sessions)", sum, rows)
	}
	var linked int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM therapy_credit_ledger WHERE user_id = $1 AND reason = 'session' AND session_id IS NOT NULL`, user).Scan(&linked); err != nil || linked != 3 {
		t.Fatalf("every consumption must reference its session: %d %v", linked, err)
	}
}

// A failed session insert rolls the consumed credit back.
func TestIntegration_TherapyCredits_FailedSessionInsertKeepsCredit(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	if _, err := NewPaymentRepository(db).DevGrantTherapyCredits(ctx, user, 1); err != nil {
		t.Fatal(err)
	}
	// An invalid persona violates the therapy_sessions CHECK after the credit
	// was decremented inside the transaction.
	if _, err := NewTherapyRepository(db).CreateWithCredit(ctx, user, models.TherapyPersona("not-a-persona"), models.TherapyContextSnapshot{}, 0); err == nil {
		t.Fatal("an invalid persona must fail the session insert (therapy_sessions persona CHECK)")
	}
	if got := creditBalance(t, db, user); got != 1 {
		t.Fatalf("failed start must not spend the credit, balance %d", got)
	}
	if sum, rows := ledgerSum(t, db, user); sum != 1 || rows != 1 {
		t.Fatalf("no consumption may be logged: sum=%d rows=%d", sum, rows)
	}
}

func TestIntegration_TherapyCredits_BalanceCannotGoNegative(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	user := regressionUser(t, db)
	if _, err := db.Exec(context.Background(), `UPDATE users SET therapy_credits = -1 WHERE id = $1`, user); err == nil {
		t.Fatal("the schema must refuse a negative balance")
	}
	if _, err := NewTherapyRepository(db).CreateWithCredit(context.Background(), user, models.PersonaComforting, models.TherapyContextSnapshot{}, 0); !errors.Is(err, models.ErrNoTherapyCredits) {
		t.Fatalf("want ErrNoTherapyCredits, got %v", err)
	}
}

// Migration 000047 is additive and its down migration fully reverses it,
// including credit purchases that the old payments CHECK cannot hold.
func TestIntegration_Migration047_TherapyCreditsUpDown(t *testing.T) {
	db := regressionDBUpTo(t, "000046")
	ctx := context.Background()
	user := regressionUser(t, db)
	exec := func(file string) {
		t.Helper()
		sql, err := os.ReadFile("../../migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	if _, err := db.Exec(ctx, `INSERT INTO payments (user_id, transaction_id, plan, amount, currency, store, product_id) VALUES ($1, 'GPA.before', 'plus', 0, '', 'google', 'com.talktoode.app.plus.monthly')`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO payments (user_id, transaction_id, plan, amount, currency, store) VALUES ($1, 'GPA.too-early', 'therapy', 0, '', 'google')`, user); err == nil {
		t.Fatal("before 000047 a therapy payment must be refused")
	}

	exec("000047_therapy_credits.up.sql")
	if got := creditBalance(t, db, user); got != 0 {
		t.Fatalf("existing users start with 0 credits, got %d", got)
	}
	r := NewPaymentRepository(db)
	recordCreditPurchase(t, r, user, "GPA.credit-mig")
	if _, _, err := r.GrantTherapyCredits(ctx, "GPA.credit-mig", user, 1); err != nil {
		t.Fatal(err)
	}

	exec("000047_therapy_credits.down.sql")
	var n int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'therapy_credits'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("down must drop users.therapy_credits: %d %v", n, err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'therapy_credit_ledger'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("down must drop the ledger: %d %v", n, err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM payments WHERE transaction_id = 'GPA.before'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("plan payments must survive the down migration: %d %v", n, err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO payments (user_id, transaction_id, plan, amount, currency, store) VALUES ($1, 'GPA.after-down', 'therapy', 0, '', 'google')`, user); err == nil {
		t.Fatal("down must restore the original payments plan CHECK")
	}

	// And it re-applies cleanly.
	exec("000047_therapy_credits.up.sql")
}

// Account deletion erases the credit ledger with the user.
func TestIntegration_TherapyCredits_LedgerCascadesWithUser(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	if _, err := NewPaymentRepository(db).DevGrantTherapyCredits(ctx, user, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM users WHERE id = $1`, user); err != nil {
		t.Fatal(err)
	}
	if _, rows := ledgerSum(t, db, user); rows != 0 {
		t.Fatalf("ledger rows must cascade, %d left", rows)
	}
}
