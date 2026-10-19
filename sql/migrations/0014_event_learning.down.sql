DROP TABLE IF EXISTS ticket_episode_access;
ALTER TABLE tickets DROP COLUMN IF EXISTS access_mode;
DROP TABLE IF EXISTS event_episode_contents;
DROP TABLE IF EXISTS event_episodes;
