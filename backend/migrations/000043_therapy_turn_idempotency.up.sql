-- N3/N4 (code review 2026-10-20).
-- request_id: client-generated id of a user turn. Unique per session, so a
--   retried turn (client timeout, flaky network) returns the stored pair
--   instead of a second Claude call, a second turn and - for crisis language -
--   a second crisis detection that would hard-stop the session (ADR-014).
-- reply_to: the user message an assistant reply answers. Unique, so a turn can
--   only ever be answered once.
ALTER TABLE therapy_session_messages ADD COLUMN IF NOT EXISTS request_id UUID;
ALTER TABLE therapy_session_messages ADD COLUMN IF NOT EXISTS reply_to UUID
    REFERENCES therapy_session_messages(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS uq_therapy_messages_request
    ON therapy_session_messages (session_id, request_id) WHERE request_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_therapy_messages_reply_to
    ON therapy_session_messages (reply_to) WHERE reply_to IS NOT NULL;
