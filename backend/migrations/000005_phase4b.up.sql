-- Phase 4b: Weekly Emotional Review
CREATE TYPE weekly_review_status AS ENUM ('pending', 'completed', 'failed');

CREATE TABLE weekly_reviews (
    id            UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID                  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start    DATE                  NOT NULL,   -- Sunday that triggered this review
    narrative     TEXT,
    top_emotions  TEXT[]                DEFAULT '{}',
    mood_arc      JSONB                 DEFAULT '[]',
    entry_count   INT                   NOT NULL DEFAULT 0,
    status        weekly_review_status  NOT NULL DEFAULT 'pending',
    scheduled_at  TIMESTAMPTZ           NOT NULL,   -- Sunday 10 AM in user's local time
    generated_at  TIMESTAMPTZ,
    error_msg     TEXT,
    created_at    TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, week_start)
);

CREATE INDEX idx_weekly_reviews_pending ON weekly_reviews(scheduled_at)
    WHERE status = 'pending';
