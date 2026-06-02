ALTER TABLE entry_analysis
    DROP COLUMN IF EXISTS dream_symbols,
    DROP COLUMN IF EXISTS dream_type;

ALTER TABLE entries DROP CONSTRAINT IF EXISTS entries_mode_check;
ALTER TABLE entries
    ADD CONSTRAINT entries_mode_check
        CHECK (mode IN ('processing', 'rant', 'gratitude', 'decision'));
