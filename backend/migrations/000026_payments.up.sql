-- Payment audit trail for plan upgrades.
-- payment_intent_id is UNIQUE: a Stripe PaymentIntent can grant a plan exactly
-- once (idempotency / replay protection for POST /billing/upgrade).

CREATE TABLE IF NOT EXISTS payments (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_intent_id TEXT        NOT NULL UNIQUE,
    plan              TEXT        NOT NULL CHECK (plan IN ('plus', 'pro')),
    amount            BIGINT      NOT NULL,
    currency          TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_user ON payments (user_id);
