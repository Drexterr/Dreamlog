DROP TABLE IF EXISTS streak_freeze_days;
ALTER TABLE users
    DROP COLUMN IF EXISTS streak_freeze_count,
    DROP COLUMN IF EXISTS streak_freeze_granted_week;
