-- Relationship Map management: allow users to hide people (noise / privacy).
-- Rename and merge reuse existing columns; only "hidden" is new state.
ALTER TABLE people ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT false;
