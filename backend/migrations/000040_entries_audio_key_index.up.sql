-- Lookup index for idempotent POST /entries (retry with the same audio_key).
CREATE INDEX IF NOT EXISTS idx_entries_user_audio_key ON entries(user_id, audio_key);
