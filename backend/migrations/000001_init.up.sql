-- Phase 1: DreamLog Foundation Schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ──────────────────────────────────────────
-- USERS
-- ──────────────────────────────────────────
CREATE TABLE users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    supabase_id TEXT        NOT NULL UNIQUE,
    email       TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_supabase_id ON users(supabase_id);
CREATE INDEX idx_users_email       ON users(email);

-- ──────────────────────────────────────────
-- ENTRIES
-- ──────────────────────────────────────────
CREATE TYPE entry_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE entries (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    audio_key        TEXT         NOT NULL,            -- R2/MinIO object key
    audio_size_bytes BIGINT       NOT NULL DEFAULT 0,
    duration_sec     FLOAT        NOT NULL DEFAULT 0,
    status           entry_status NOT NULL DEFAULT 'pending',
    transcript       TEXT,
    language         TEXT,
    error_msg        TEXT,
    retry_count      INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entries_user_id    ON entries(user_id);
CREATE INDEX idx_entries_status     ON entries(status);
CREATE INDEX idx_entries_created_at ON entries(created_at DESC);
CREATE INDEX idx_entries_user_created ON entries(user_id, created_at DESC);

-- ──────────────────────────────────────────
-- ENTRY ANALYSIS  (scaffolded, used in Phase 2)
-- ──────────────────────────────────────────
CREATE TABLE entry_analysis (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID        NOT NULL UNIQUE REFERENCES entries(id) ON DELETE CASCADE,
    summary    TEXT,
    themes     TEXT[]      NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entry_analysis_entry_id ON entry_analysis(entry_id);

-- ──────────────────────────────────────────
-- EMOTIONAL METRICS  (scaffolded, used in Phase 2)
-- ──────────────────────────────────────────
CREATE TABLE emotional_metrics (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID        NOT NULL UNIQUE REFERENCES entries(id) ON DELETE CASCADE,
    sentiment  TEXT,
    score      FLOAT,
    emotions   JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emotional_metrics_entry_id ON emotional_metrics(entry_id);

-- ──────────────────────────────────────────
-- DEAD LETTER QUEUE (failed jobs audit trail)
-- ──────────────────────────────────────────
CREATE TABLE dead_letter_jobs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID        REFERENCES entries(id) ON DELETE SET NULL,
    payload    JSONB       NOT NULL,
    error      TEXT,
    attempt    INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dlq_entry_id   ON dead_letter_jobs(entry_id);
CREATE INDEX idx_dlq_created_at ON dead_letter_jobs(created_at DESC);

-- ──────────────────────────────────────────
-- TRIGGERS: updated_at
-- ──────────────────────────────────────────
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER trg_entries_updated_at
    BEFORE UPDATE ON entries
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER trg_entry_analysis_updated_at
    BEFORE UPDATE ON entry_analysis
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
