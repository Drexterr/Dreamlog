DROP TRIGGER IF EXISTS trg_conversations_updated_at ON conversations;
DROP TRIGGER IF EXISTS trg_user_devices_updated_at ON user_devices;
DROP TRIGGER IF EXISTS trg_entries_search_vector ON entries;
DROP FUNCTION IF EXISTS entries_search_vector_update();

DROP VIEW IF EXISTS v_daily_mood;

DROP TABLE IF EXISTS nudges;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS user_devices;

DROP TYPE IF EXISTS nudge_status;

ALTER TABLE entries DROP COLUMN IF EXISTS search_vector;
ALTER TABLE users DROP COLUMN IF EXISTS timezone;
ALTER TABLE users DROP COLUMN IF EXISTS fcm_nudge_hour;

ALTER TABLE entry_analysis
    DROP COLUMN IF EXISTS mood_score,
    DROP COLUMN IF EXISTS emotional_tone,
    DROP COLUMN IF EXISTS topics,
    DROP COLUMN IF EXISTS key_quotes,
    DROP COLUMN IF EXISTS summary,
    DROP COLUMN IF EXISTS reflection,
    DROP COLUMN IF EXISTS morning_nudge,
    DROP COLUMN IF EXISTS is_crisis;
