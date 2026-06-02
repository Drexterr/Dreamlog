-- Phase 4c: Streak Mechanics Overhaul

-- Add streak freeze tracking to users.
-- streak_freeze_count: available freezes (0-3). Auto-granted 1 per week, max 3 stored.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS streak_freeze_count INT NOT NULL DEFAULT 1
        CHECK (streak_freeze_count >= 0 AND streak_freeze_count <= 3),
    ADD COLUMN IF NOT EXISTS streak_freeze_granted_week DATE;

-- Track individual freeze days (each row = one day "protected" from breaking streak).
CREATE TABLE IF NOT EXISTS streak_freeze_days (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    frozen_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, frozen_date)
);

CREATE INDEX IF NOT EXISTS idx_streak_freeze_days_user ON streak_freeze_days(user_id);
