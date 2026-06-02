CREATE TABLE annual_reviews (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    year         INT         NOT NULL,                                -- calendar year being reviewed (e.g., 2025)
    narrative    TEXT,
    top_emotions TEXT[],
    top_topics   TEXT[],
    mood_arc     JSONB,                                              -- [{month:"2025-01",avg_mood:65,entry_count:5}, ...]
    entry_count  INT         NOT NULL DEFAULT 0,
    avg_mood     INT,                                               -- overall avg mood for the year
    status       TEXT        NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending','completed','failed')),
    scheduled_at TIMESTAMPTZ NOT NULL,
    generated_at TIMESTAMPTZ,
    error_msg    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, year)
);

CREATE INDEX idx_annual_reviews_pending ON annual_reviews (scheduled_at, status)
    WHERE status = 'pending';
