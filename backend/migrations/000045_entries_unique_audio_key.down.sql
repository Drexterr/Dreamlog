DROP INDEX IF EXISTS uq_entries_user_audio_key;
CREATE INDEX IF NOT EXISTS idx_entries_user_audio_key ON entries(user_id, audio_key);
