-- Phase 4f extension: Dream Decoder mode
-- 1. Expand the entries.mode CHECK to allow 'dream'.
-- 2. Add dream-specific columns to entry_analysis (NULL for non-dream entries).

ALTER TABLE entries DROP CONSTRAINT IF EXISTS entries_mode_check;
ALTER TABLE entries
    ADD CONSTRAINT entries_mode_check
        CHECK (mode IN ('processing', 'rant', 'gratitude', 'decision', 'dream'));

ALTER TABLE entry_analysis
    ADD COLUMN IF NOT EXISTS dream_symbols TEXT[],
    ADD COLUMN IF NOT EXISTS dream_type    TEXT
        CHECK (dream_type IN ('nightmare', 'lucid', 'recurring', 'vivid', 'surreal', 'mundane') OR dream_type IS NULL);
