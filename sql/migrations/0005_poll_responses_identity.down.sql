DROP INDEX IF EXISTS idx_poll_responses_poll_user;

UPDATE poll_responses SET user_id = '' WHERE user_id IS NULL;

ALTER TABLE poll_responses ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE poll_responses DROP COLUMN IF EXISTS is_verified;
