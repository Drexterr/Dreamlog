-- The purged accounts cannot be restored; only the columns come back.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_deleted           BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS deleted_at           TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reregistered_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reregistration_count INT NOT NULL DEFAULT 0;
