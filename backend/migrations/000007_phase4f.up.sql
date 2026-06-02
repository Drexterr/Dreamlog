-- Phase 4f: Prompt Modes
-- Add mode column to entries; valid values: processing (default), rant, gratitude, decision.

ALTER TABLE entries
    ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'processing'
        CHECK (mode IN ('processing', 'rant', 'gratitude', 'decision'));
