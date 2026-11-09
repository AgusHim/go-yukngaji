ALTER TABLE presence DROP COLUMN IF EXISTS verifier_session_id;
DROP TABLE IF EXISTS event_verifier_sessions;
DROP TABLE IF EXISTS event_verifier_credentials;
