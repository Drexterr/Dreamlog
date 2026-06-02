-- Phase 5a: Therapist share links
-- Read-only, passcode-protected, 72-hour expiry

CREATE TABLE share_links (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token        TEXT        NOT NULL UNIQUE,          -- URL-safe random token (32 bytes hex)
    passcode_hash TEXT       NOT NULL,                 -- bcrypt hash of 4-digit numeric code
    expires_at   TIMESTAMPTZ NOT NULL,                 -- creation + 72h
    revoked      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_share_links_user_id ON share_links(user_id);
CREATE INDEX idx_share_links_token   ON share_links(token);
