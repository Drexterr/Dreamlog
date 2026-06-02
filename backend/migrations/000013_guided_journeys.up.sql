-- Guided Journeys: structured multi-step voice journaling sequences.
-- journey_id references a template defined in code (services/journey.go).

CREATE TABLE IF NOT EXISTS journey_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    journey_id   TEXT NOT NULL,
    current_step INT  NOT NULL DEFAULT 0,
    total_steps  INT  NOT NULL,
    status       TEXT NOT NULL DEFAULT 'in_progress'
                      CHECK (status IN ('in_progress', 'completed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_journey_sessions_user
    ON journey_sessions (user_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS journey_session_steps (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL REFERENCES journey_sessions(id) ON DELETE CASCADE,
    step_index   INT  NOT NULL,
    prompt       TEXT NOT NULL,
    entry_id     UUID REFERENCES entries(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, step_index)
);
