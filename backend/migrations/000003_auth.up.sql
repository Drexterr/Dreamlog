-- Phase 3: Local email/password authentication
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
