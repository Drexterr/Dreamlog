ALTER TABLE users
  DROP COLUMN IF EXISTS is_deleted,
  DROP COLUMN IF EXISTS deleted_at,
  DROP COLUMN IF EXISTS first_joined_at,
  DROP COLUMN IF EXISTS reregistered_at,
  DROP COLUMN IF EXISTS reregistration_count,
  DROP COLUMN IF EXISTS country;
