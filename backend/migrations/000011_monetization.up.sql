-- Add subscription plan to users table.
-- plan: free | plus | pro | b2b  (default 'free')
-- plan_expires_at: NULL means plan does not expire (lifetime / manual)

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS plan TEXT NOT NULL DEFAULT 'free'
        CHECK (plan IN ('free', 'plus', 'pro', 'b2b')),
    ADD COLUMN IF NOT EXISTS plan_expires_at TIMESTAMPTZ DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_users_plan ON users (plan);
