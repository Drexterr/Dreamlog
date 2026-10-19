-- environment: 'production' | 'sandbox' (App Store sandbox / TestFlight / Play
--   license-tester purchases - no money changed hands).
-- granted_at: set once the plan grant for this transaction committed. A replay
--   of the same transaction by the same user is idempotent: if granted_at is
--   set it returns the current plan, if NULL (the grant failed after the
--   payment was recorded) the grant is retried instead of being refused.
ALTER TABLE payments ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'production';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS granted_at  TIMESTAMPTZ;
UPDATE payments SET granted_at = created_at WHERE granted_at IS NULL;
