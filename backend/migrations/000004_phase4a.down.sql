-- Phase 4a rollback
ALTER TABLE users DROP COLUMN IF EXISTS preferred_name;
ALTER TABLE users DROP COLUMN IF EXISTS goal;
