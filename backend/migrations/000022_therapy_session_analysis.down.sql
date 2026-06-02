ALTER TABLE therapy_sessions
    DROP COLUMN IF EXISTS session_mood_score,
    DROP COLUMN IF EXISTS session_emotional_tone,
    DROP COLUMN IF EXISTS session_topics,
    DROP COLUMN IF EXISTS session_key_insights;
