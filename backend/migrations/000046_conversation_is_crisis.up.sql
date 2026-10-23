-- N12: remember that a follow-up conversation ended in crisis, so reopening
-- it shows the crisis resources instead of an ordinary closed conversation.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS is_crisis BOOLEAN NOT NULL DEFAULT false;
