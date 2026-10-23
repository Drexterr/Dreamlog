-- Therapy pay-per-use (LAUNCH_CHECKLIST §1): sessions beyond the free first
-- session and Pro's monthly allowance are paid with prepaid session credits,
-- bought as consumable In-App Purchases (POST /billing/therapy-credits).
--
-- users.therapy_credits is the spendable balance. It is only ever changed in
-- the same transaction as a ledger row: +N when a verified store purchase is
-- granted, -1 when a session is started with a credit. The CHECK makes an
-- overdraft impossible even under concurrent session starts.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS therapy_credits INT NOT NULL DEFAULT 0
    CONSTRAINT users_therapy_credits_nonnegative CHECK (therapy_credits >= 0);

-- Audit trail of every balance change. Cascades with the user (account
-- deletion erases it); the payment rows themselves are detached, not deleted
-- (000037), so reconciliation still works.
CREATE TABLE IF NOT EXISTS therapy_credit_ledger (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delta          INT         NOT NULL CHECK (delta <> 0),
    reason         TEXT        NOT NULL CHECK (reason IN ('purchase', 'session', 'dev_grant')),
    transaction_id TEXT,
    session_id     UUID        REFERENCES therapy_sessions(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_therapy_credit_ledger_user
    ON therapy_credit_ledger (user_id, created_at DESC);

-- Belt and braces on top of payments.granted_at: one store transaction can
-- credit an account at most once.
CREATE UNIQUE INDEX IF NOT EXISTS idx_therapy_credit_ledger_transaction
    ON therapy_credit_ledger (transaction_id) WHERE transaction_id IS NOT NULL;

-- Credit purchases are recorded in payments like plan passes (replay
-- protection on the unique transaction_id); their "plan" is 'therapy'.
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_plan_check;
ALTER TABLE payments
    ADD CONSTRAINT payments_plan_check CHECK (plan IN ('plus', 'pro', 'therapy'));
