CREATE TABLE IF NOT EXISTS event_verifier_credentials (
    event_id       varchar(36) PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    password_hash  text NOT NULL,
    enabled        boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS event_verifier_sessions (
    id          varchar(36) PRIMARY KEY,
    event_id    varchar(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    token_hash  varchar(64) NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    last_used_at timestamptz NOT NULL DEFAULT now(),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_event_verifier_sessions_event_expiry
    ON event_verifier_sessions (event_id, expires_at);

ALTER TABLE presence
    ADD COLUMN IF NOT EXISTS verifier_session_id varchar(36)
    REFERENCES event_verifier_sessions(id) ON DELETE SET NULL;
