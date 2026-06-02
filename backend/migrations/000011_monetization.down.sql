DROP INDEX IF EXISTS idx_users_plan;

ALTER TABLE users
    DROP COLUMN IF EXISTS plan_expires_at,
    DROP COLUMN IF EXISTS plan;
