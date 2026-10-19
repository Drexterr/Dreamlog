-- Lifetime cap on wrong passcodes. The per-window lockout resets every
-- 15 minutes, which over a 72h link still allows ~1,400 guesses; after
-- share_max_total_failures wrong passcodes in total the link is expired.
ALTER TABLE share_links ADD COLUMN IF NOT EXISTS total_failed_attempts INT NOT NULL DEFAULT 0;
