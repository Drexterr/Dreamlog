-- Soft delete: mark accounts as deleted instead of hard-deleting
ALTER TABLE users
  ADD COLUMN is_deleted           BOOLEAN     NOT NULL DEFAULT false,
  ADD COLUMN deleted_at           TIMESTAMPTZ,
  ADD COLUMN first_joined_at      TIMESTAMPTZ,
  ADD COLUMN reregistered_at      TIMESTAMPTZ,
  ADD COLUMN reregistration_count INT         NOT NULL DEFAULT 0,
  ADD COLUMN country              TEXT;

-- Backfill first_joined_at from created_at for existing users
UPDATE users SET first_joined_at = created_at;
