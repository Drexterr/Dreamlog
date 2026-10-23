-- N10: one entry per (user, audio_key). Concurrent registrations of the same
-- upload converge on one row (INSERT ... ON CONFLICT DO NOTHING) instead of
-- creating a duplicate whose job fails once the first job deletes the audio.
-- Existing duplicates are folded first, keeping the most useful row: a
-- completed one, else the oldest.
DELETE FROM entries e
USING (
    SELECT id, ROW_NUMBER() OVER (
        PARTITION BY user_id, audio_key
        ORDER BY (status = 'completed') DESC, created_at, id
    ) AS rn
    FROM entries
) ranked
WHERE e.id = ranked.id AND ranked.rn > 1;

DROP INDEX IF EXISTS idx_entries_user_audio_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_entries_user_audio_key ON entries(user_id, audio_key);
