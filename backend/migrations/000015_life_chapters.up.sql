CREATE TABLE life_chapters (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    description TEXT,
    start_date  DATE        NOT NULL,
    end_date    DATE,                                         -- NULL = ongoing chapter
    emoji       TEXT,                                         -- optional icon e.g. "🌱"
    color       TEXT        NOT NULL DEFAULT '#7C3AED',       -- hex color for UI
    summary     TEXT,                                         -- Claude-generated (populated on demand)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_life_chapters_user ON life_chapters (user_id, start_date DESC);
