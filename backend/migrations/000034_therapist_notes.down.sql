DROP TABLE IF EXISTS client_sessions;
DROP TYPE  IF EXISTS client_session_status;
DROP TABLE IF EXISTS external_clients;
DROP TABLE IF EXISTS therapist_keys;

ALTER TABLE therapists DROP COLUMN IF EXISTS client_consent_version;
ALTER TABLE therapists DROP COLUMN IF EXISTS client_consent_accepted_at;
ALTER TABLE users DROP COLUMN IF EXISTS tos_version;
ALTER TABLE users DROP COLUMN IF EXISTS tos_accepted_at;
