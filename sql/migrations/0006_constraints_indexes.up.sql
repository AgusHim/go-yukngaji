-- Batasan dan index yang menopang validasi di sisi aplikasi.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_orders_public_id ON orders (public_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_tickets_public_id ON user_tickets (public_id);

-- NOT VALID: hanya berlaku untuk baris baru, sehingga data lama yang mungkin
-- bernilai negatif tidak menggagalkan migrasi. Validasi penuh bisa dijalankan
-- manual dengan ALTER TABLE orders VALIDATE CONSTRAINT chk_orders_amounts.
ALTER TABLE IF EXISTS orders DROP CONSTRAINT IF EXISTS chk_orders_amounts;
ALTER TABLE IF EXISTS orders
  ADD CONSTRAINT chk_orders_amounts
  CHECK (amount >= 0 AND donation >= 0 AND admin_fee >= 0) NOT VALID;

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_event_id ON orders (event_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);

CREATE INDEX IF NOT EXISTS idx_user_tickets_order_id ON user_tickets (order_id);
CREATE INDEX IF NOT EXISTS idx_user_tickets_event_id ON user_tickets (event_id);
CREATE INDEX IF NOT EXISTS idx_user_tickets_email ON user_tickets (lower(user_email));

CREATE INDEX IF NOT EXISTS idx_users_email ON users (lower(email));
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

CREATE INDEX IF NOT EXISTS idx_events_slug ON events (slug);
CREATE INDEX IF NOT EXISTS idx_events_code ON events (code);
