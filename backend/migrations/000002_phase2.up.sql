-- Phase 2: AI Analysis, Conversations, Nudges, Search, Mood, Devices

-- ──────────────────────────────────────────────────────────────────
-- EXPAND entry_analysis (was stubbed in Phase 1)
-- ──────────────────────────────────────────────────────────────────
ALTER TABLE entry_analysis
    ADD COLUMN IF NOT EXISTS mood_score     INT          CHECK (mood_score BETWEEN 1 AND 100),
    ADD COLUMN IF NOT EXISTS emotional_tone JSONB        NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS topics         TEXT[]       NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS key_quotes     TEXT[]       NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS summary        TEXT,
    ADD COLUMN IF NOT EXISTS reflection     TEXT,
    ADD COLUMN IF NOT EXISTS morning_nudge  TEXT,
    ADD COLUMN IF NOT EXISTS is_crisis      BOOLEAN      NOT NULL DEFAULT FALSE;

-- ──────────────────────────────────────────────────────────────────
-- CONVERSATIONS  (follow-up "Tell me more" flow)
-- Max 3 turns enforced in application layer.
-- ──────────────────────────────────────────────────────────────────
CREATE TABLE conversations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id    UUID        NOT NULL UNIQUE REFERENCES entries(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    turn_count  INT         NOT NULL DEFAULT 0,   -- 0 = not started, max 3
    is_closed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_entry_id ON conversations(entry_id);
CREATE INDEX idx_conversations_user_id  ON conversations(user_id);

CREATE TABLE conversation_messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT        NOT NULL CHECK (role IN ('user','assistant')),
    content         TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conv_messages_conv_id ON conversation_messages(conversation_id);
CREATE INDEX idx_conv_messages_created ON conversation_messages(conversation_id, created_at ASC);

-- ──────────────────────────────────────────────────────────────────
-- USER DEVICES  (FCM push tokens)
-- ──────────────────────────────────────────────────────────────────
CREATE TABLE user_devices (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token   TEXT        NOT NULL UNIQUE,
    platform    TEXT        NOT NULL DEFAULT 'unknown',  -- ios | android
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_devices_user_id ON user_devices(user_id);
CREATE INDEX idx_user_devices_token   ON user_devices(fcm_token);

-- ──────────────────────────────────────────────────────────────────
-- NUDGES  (morning notifications)
-- ──────────────────────────────────────────────────────────────────
CREATE TYPE nudge_status AS ENUM ('pending', 'sent', 'failed');

CREATE TABLE nudges (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id       UUID         REFERENCES entries(id) ON DELETE SET NULL,
    message        TEXT         NOT NULL,
    scheduled_at   TIMESTAMPTZ  NOT NULL,
    timezone       TEXT         NOT NULL DEFAULT 'UTC',
    status         nudge_status NOT NULL DEFAULT 'pending',
    sent_at        TIMESTAMPTZ,
    error_msg      TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_nudges_user_id       ON nudges(user_id);
CREATE INDEX idx_nudges_scheduled_at  ON nudges(scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_nudges_status        ON nudges(status);

-- ──────────────────────────────────────────────────────────────────
-- ADD user timezone to users
-- ──────────────────────────────────────────────────────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS timezone   TEXT NOT NULL DEFAULT 'UTC',
    ADD COLUMN IF NOT EXISTS fcm_nudge_hour INT NOT NULL DEFAULT 8; -- local hour (0-23)

-- ──────────────────────────────────────────────────────────────────
-- FULL-TEXT SEARCH on entries
-- tsvector indexes transcript + language-aware tokenisation.
-- ──────────────────────────────────────────────────────────────────
ALTER TABLE entries
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Populate existing rows (entries may have NULL transcript for pending ones).
UPDATE entries
SET search_vector = to_tsvector('english', COALESCE(transcript, ''))
WHERE transcript IS NOT NULL;

-- Fast GIN index for FTS.
CREATE INDEX idx_entries_search ON entries USING GIN(search_vector);

-- Trigger: keep search_vector current on INSERT/UPDATE.
CREATE OR REPLACE FUNCTION entries_search_vector_update()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', COALESCE(NEW.transcript, ''));
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_entries_search_vector
    BEFORE INSERT OR UPDATE OF transcript ON entries
    FOR EACH ROW EXECUTE FUNCTION entries_search_vector_update();

-- ──────────────────────────────────────────────────────────────────
-- DAILY MOOD VIEW (convenience)
-- ──────────────────────────────────────────────────────────────────
CREATE OR REPLACE VIEW v_daily_mood AS
SELECT
    e.user_id,
    DATE(e.created_at AT TIME ZONE 'UTC') AS day,
    ROUND(AVG(ea.mood_score))::INT        AS avg_mood,
    COUNT(e.id)                           AS entry_count
FROM entries e
JOIN entry_analysis ea ON ea.entry_id = e.id
WHERE ea.mood_score IS NOT NULL
GROUP BY e.user_id, DATE(e.created_at AT TIME ZONE 'UTC');

-- ──────────────────────────────────────────────────────────────────
-- TRIGGERS: updated_at on new tables
-- ──────────────────────────────────────────────────────────────────
CREATE TRIGGER trg_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER trg_user_devices_updated_at
    BEFORE UPDATE ON user_devices
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
