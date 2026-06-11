-- Analytics event stream. Append-only; never UPDATE or DELETE rows.
-- Privacy rule: events carry IDs and metadata only — never transcript or
-- reflection content. See docs/PRICING.md §6c for the minimum event set.

CREATE TABLE IF NOT EXISTS analytics_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        REFERENCES users(id) ON DELETE SET NULL,
    event_name  TEXT        NOT NULL,
    properties  JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookups by user and by event type (funnel queries).
CREATE INDEX IF NOT EXISTS idx_analytics_user       ON analytics_events (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_event_name ON analytics_events (event_name, created_at DESC);
