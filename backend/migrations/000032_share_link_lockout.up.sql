-- Passcode brute-force protection for public share links.
-- A share link protects clinical journal data behind a 4-digit passcode; without
-- a lockout an attacker who obtains the (forwardable) token can brute-force the
-- 10,000-value space. failed_attempts is incremented on each wrong passcode and
-- locked_until temporarily blocks validation once a threshold is crossed.
ALTER TABLE share_links
    ADD COLUMN IF NOT EXISTS failed_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until    TIMESTAMPTZ;
