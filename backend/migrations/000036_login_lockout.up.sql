-- Brute-force protection for the local email/password login path
-- (POST /auth/login - internal/services/auth.go, ADR-001). Mirrors the
-- share-link passcode lockout added in 000032: failed_login_attempts is
-- incremented on each wrong password, and login_locked_until temporarily
-- blocks further attempts once a threshold is crossed.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS failed_login_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS login_locked_until     TIMESTAMPTZ;
