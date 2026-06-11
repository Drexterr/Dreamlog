-- Voice language preference for Therapy Mode TTS output.
-- 'auto' = follow the per-turn detected language (Whisper / script detection).
-- 'english' / 'hindi' = always use that language's voice regardless of detection.
ALTER TABLE users
    ADD COLUMN voice_language TEXT NOT NULL DEFAULT 'auto'
    CHECK (voice_language IN ('auto', 'english', 'hindi'));
