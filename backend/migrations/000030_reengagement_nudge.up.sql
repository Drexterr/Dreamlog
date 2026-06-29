-- Add nudge_type to distinguish morning nudges from re-engagement nudges.
ALTER TABLE nudges ADD COLUMN nudge_type TEXT NOT NULL DEFAULT 'morning';
