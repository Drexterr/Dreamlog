-- Phase 6: Therapy Mode

CREATE TYPE therapy_session_status AS ENUM ('active', 'completed', 'expired', 'crisis_detected');

CREATE TABLE therapy_sessions (
    id                    UUID                   PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID                   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status                therapy_session_status NOT NULL DEFAULT 'active',
    started_at            TIMESTAMPTZ            NOT NULL DEFAULT NOW(),
    expires_at            TIMESTAMPTZ            NOT NULL,           -- started_at + 1 hour, enforced server-side
    ended_at              TIMESTAMPTZ,
    duration_sec          INT,                                       -- elapsed seconds at session end
    turn_count            INT                    NOT NULL DEFAULT 0,
    context_snapshot      JSONB                  NOT NULL DEFAULT '{}', -- journal context at session start
    post_session_summary  TEXT,
    billing_amount_paise  INT                    NOT NULL DEFAULT 0,  -- 49900 = ₹499; 0 if Pro/free
    created_at            TIMESTAMPTZ            NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_therapy_sessions_user ON therapy_sessions (user_id, created_at DESC);
CREATE INDEX idx_therapy_sessions_status ON therapy_sessions (status) WHERE status = 'active';

CREATE TABLE therapy_session_messages (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID        NOT NULL REFERENCES therapy_sessions(id) ON DELETE CASCADE,
    role        TEXT        NOT NULL CHECK (role IN ('user', 'assistant')),
    content     TEXT        NOT NULL,
    input_mode  TEXT        NOT NULL CHECK (input_mode IN ('voice', 'text', 'system')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_therapy_messages_session ON therapy_session_messages (session_id, created_at ASC);
