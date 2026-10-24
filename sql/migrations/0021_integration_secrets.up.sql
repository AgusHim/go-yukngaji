CREATE TABLE IF NOT EXISTS integration_secrets (
  key varchar(80) PRIMARY KEY,
  value text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
