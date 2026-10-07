DROP INDEX IF EXISTS uniq_presence_ticket_check_in_date;
ALTER TABLE IF EXISTS presence DROP COLUMN IF EXISTS check_in_date;
