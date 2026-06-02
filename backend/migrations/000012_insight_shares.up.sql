-- Tracks every time a user shares an insight card (viral acquisition loop).
-- week_start is the Monday of the week the card represented (ISO week).
CREATE TABLE IF NOT EXISTS insight_shares (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_insight_shares_user_week
    ON insight_shares (user_id, week_start DESC);
