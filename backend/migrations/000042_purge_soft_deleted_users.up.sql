-- N2 (code review 2026-10-20): accounts deleted before hard delete (000037)
-- were only flagged is_deleted=true and kept all their journal, therapy and
-- relationship data - and approved therapist links kept serving it. Those
-- users asked for deletion: remove them now. Every user-owned table cascades;
-- payments are detached (ON DELETE SET NULL, 000037).
DELETE FROM users WHERE is_deleted = true;

-- Drop the soft-delete state entirely so it cannot come back.
ALTER TABLE users
    DROP COLUMN IF EXISTS is_deleted,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS reregistered_at,
    DROP COLUMN IF EXISTS reregistration_count;
