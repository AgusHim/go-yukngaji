-- Sesi event, materi, dan rekaman. Akses tidak disimpan pada user secara
-- langsung; ia selalu diturunkan dari tiket berstatus paid agar refund atau
-- pembatalan order langsung menutup akses.
CREATE TABLE IF NOT EXISTS event_episodes (
  id varchar PRIMARY KEY,
  event_id varchar NOT NULL REFERENCES events(id),
  position integer NOT NULL DEFAULT 1,
  title varchar NOT NULL,
  description text NOT NULL DEFAULT '',
  start_at timestamp,
  end_at timestamp,
  is_published boolean NOT NULL DEFAULT false,
  created_at timestamp NOT NULL DEFAULT now(),
  updated_at timestamp NOT NULL DEFAULT now(),
  deleted_at timestamp
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_event_episodes_position
  ON event_episodes(event_id, position) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_event_episodes_event_id
  ON event_episodes(event_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS event_episode_contents (
  id varchar PRIMARY KEY,
  episode_id varchar NOT NULL REFERENCES event_episodes(id),
  position integer NOT NULL DEFAULT 1,
  type varchar NOT NULL CHECK (type IN ('material', 'recording')),
  title varchar NOT NULL,
  description text NOT NULL DEFAULT '',
  url text NOT NULL,
  is_published boolean NOT NULL DEFAULT false,
  created_at timestamp NOT NULL DEFAULT now(),
  updated_at timestamp NOT NULL DEFAULT now(),
  deleted_at timestamp
);

CREATE INDEX IF NOT EXISTS idx_episode_contents_episode_id
  ON event_episode_contents(episode_id) WHERE deleted_at IS NULL;

ALTER TABLE tickets ADD COLUMN IF NOT EXISTS access_mode varchar NOT NULL DEFAULT 'all';
CREATE TABLE IF NOT EXISTS ticket_episode_access (
  ticket_id varchar NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  episode_id varchar NOT NULL REFERENCES event_episodes(id) ON DELETE CASCADE,
  created_at timestamp NOT NULL DEFAULT now(),
  PRIMARY KEY (ticket_id, episode_id)
);
CREATE INDEX IF NOT EXISTS idx_ticket_episode_access_episode_id
  ON ticket_episode_access(episode_id);
