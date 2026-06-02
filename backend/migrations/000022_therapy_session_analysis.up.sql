-- Add structured analysis fields to therapy_sessions.
-- Populated by Claude at session end; equivalent to entry_analysis for journal entries.
-- These fields feed into mood history, emotion pattern radar, and topic trends.
ALTER TABLE therapy_sessions
    ADD COLUMN session_mood_score       INT CHECK (session_mood_score BETWEEN 1 AND 100),
    ADD COLUMN session_emotional_tone   JSONB,   -- [{"emotion": str, "intensity": float}]
    ADD COLUMN session_topics           TEXT[],  -- 2-5 themes discussed in the session
    ADD COLUMN session_key_insights     TEXT[];  -- patterns, breakthroughs, unresolved threads for next session
