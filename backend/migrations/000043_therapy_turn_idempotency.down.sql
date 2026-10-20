DROP INDEX IF EXISTS uq_therapy_messages_reply_to;
DROP INDEX IF EXISTS uq_therapy_messages_request;
ALTER TABLE therapy_session_messages DROP COLUMN IF EXISTS reply_to;
ALTER TABLE therapy_session_messages DROP COLUMN IF EXISTS request_id;
