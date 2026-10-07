ALTER TABLE IF EXISTS orders DROP CONSTRAINT IF EXISTS chk_orders_amounts;

DROP INDEX IF EXISTS idx_events_code;
DROP INDEX IF EXISTS idx_events_slug;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_user_tickets_email;
DROP INDEX IF EXISTS idx_user_tickets_event_id;
DROP INDEX IF EXISTS idx_user_tickets_order_id;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP INDEX IF EXISTS idx_orders_event_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS uniq_user_tickets_public_id;
DROP INDEX IF EXISTS uniq_orders_public_id;
