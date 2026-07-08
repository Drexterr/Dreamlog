-- Therapist Notes: external clients + session notes (encrypted at rest) + ToS acceptance
--
-- Sensitive columns (client names, note text, bullets, summaries) are stored as
-- BYTEA ciphertext, encrypted application-side with AES-256-GCM under a
-- per-therapist data key (DEK). Each DEK is stored wrapped (encrypted) by the
-- server master key in therapist_keys - the database never holds a usable key.

-- ── Terms of Service acceptance ──────────────────────────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS tos_accepted_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS tos_version     TEXT;

-- Therapist-specific acceptance: "I have my client's consent and am responsible
-- for client data I upload". Recorded separately from the general ToS.
ALTER TABLE therapists ADD COLUMN IF NOT EXISTS client_consent_accepted_at TIMESTAMPTZ;
ALTER TABLE therapists ADD COLUMN IF NOT EXISTS client_consent_version     TEXT;

-- ── Per-therapist wrapped data keys (envelope encryption) ────────────────────
CREATE TABLE therapist_keys (
    therapist_id UUID        PRIMARY KEY REFERENCES therapists(id) ON DELETE CASCADE,
    wrapped_dek  BYTEA       NOT NULL,   -- 32-byte DEK encrypted with the master key
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── External clients (therapist-owned, NOT app users) ────────────────────────
CREATE TABLE external_clients (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    therapist_id  UUID        NOT NULL REFERENCES therapists(id) ON DELETE CASCADE,
    name_enc      BYTEA       NOT NULL,  -- encrypted display name / initials
    role          TEXT        NOT NULL DEFAULT 'client', -- free-form label, e.g. client | couple | minor
    archived      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_external_clients_therapist ON external_clients(therapist_id) WHERE NOT archived;

-- ── Client sessions (one row per consultation; holds the notes) ──────────────
CREATE TYPE client_session_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE client_sessions (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    therapist_id       UUID        NOT NULL REFERENCES therapists(id) ON DELETE CASCADE,
    external_client_id UUID        REFERENCES external_clients(id) ON DELETE CASCADE,
    linked_user_id     UUID        REFERENCES users(id) ON DELETE CASCADE,
    session_date       DATE        NOT NULL DEFAULT CURRENT_DATE,
    status             client_session_status NOT NULL DEFAULT 'pending',
    image_key          TEXT,       -- storage key of the uploaded note photo; object deleted after OCR
    raw_text_enc       BYTEA,      -- encrypted OCR raw text (reference for re-edits)
    bullets_enc        BYTEA,      -- encrypted JSON array of bullet strings (the editable notes)
    summary_enc        BYTEA,      -- encrypted AI session summary
    error_msg          TEXT,
    retry_count        INT         NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- exactly one client reference: an external client OR a linked app user
    CONSTRAINT client_sessions_one_client CHECK (
        (external_client_id IS NOT NULL AND linked_user_id IS NULL) OR
        (external_client_id IS NULL AND linked_user_id IS NOT NULL)
    )
);

CREATE INDEX idx_client_sessions_therapist ON client_sessions(therapist_id, session_date DESC);
CREATE INDEX idx_client_sessions_external  ON client_sessions(external_client_id) WHERE external_client_id IS NOT NULL;
CREATE INDEX idx_client_sessions_linked    ON client_sessions(linked_user_id) WHERE linked_user_id IS NOT NULL;

-- RLS: deny-all for Supabase PostgREST, same as migration 000024. The Go
-- backend connects as the service role and is unaffected.
ALTER TABLE therapist_keys   ENABLE ROW LEVEL SECURITY;
ALTER TABLE external_clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_sessions  ENABLE ROW LEVEL SECURITY;
