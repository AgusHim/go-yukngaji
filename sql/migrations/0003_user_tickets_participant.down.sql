ALTER TABLE IF EXISTS user_tickets DROP CONSTRAINT IF EXISTS fk_user_tickets_participant;
DROP INDEX IF EXISTS idx_user_tickets_participant_user_id;
ALTER TABLE IF EXISTS user_tickets DROP COLUMN IF EXISTS participant_user_id;
