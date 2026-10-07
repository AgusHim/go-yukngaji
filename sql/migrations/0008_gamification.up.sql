-- Gamifikasi: aturan level, aturan XP per sumber, dan ledger XP.
--
-- Ledger bersifat append-only: tidak ada deleted_at dan tidak ada jalur
-- update. Koreksi dicatat sebagai baris penyesuaian baru dengan alasan dan
-- pelaksana, bukan dengan menghapus jejak. Total XP selalu SUM(delta).

CREATE TABLE IF NOT EXISTS level_rules (
  id varchar PRIMARY KEY,
  level integer NOT NULL,
  name varchar NOT NULL,
  min_xp integer NOT NULL,
  badge_label varchar,
  badge_icon_url varchar,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_level_rules_level
  ON level_rules (level);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_level_rules_min_xp
  ON level_rules (min_xp);

CREATE TABLE IF NOT EXISTS xp_rules (
  id varchar PRIMARY KEY,
  source_type varchar NOT NULL,
  xp integer NOT NULL,
  description text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_xp_rules_source_type
  ON xp_rules (source_type);

CREATE TABLE IF NOT EXISTS xp_ledger (
  id varchar PRIMARY KEY,
  user_id varchar NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  delta integer NOT NULL CHECK (delta <> 0),
  source_type varchar NOT NULL,
  ref_type varchar,
  ref_id varchar,
  dedup_key varchar(200) NOT NULL,
  note text,
  actor_user_id varchar REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Satu-satunya jaminan anti-pemberian-ganda. Penulisan memakai
-- INSERT ... ON CONFLICT (dedup_key) DO NOTHING lalu memeriksa RowsAffected.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_xp_ledger_dedup_key
  ON xp_ledger (dedup_key);

CREATE INDEX IF NOT EXISTS idx_xp_ledger_user_created
  ON xp_ledger (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_xp_ledger_created
  ON xp_ledger (created_at);
CREATE INDEX IF NOT EXISTS idx_xp_ledger_source_ref
  ON xp_ledger (source_type, ref_id);

-- Nilai awal di bawah ini adalah default yang dapat diubah pengurus lewat
-- API admin, bukan keputusan produk final.
INSERT INTO level_rules (id, level, name, min_xp, badge_label) VALUES
  ('lvl-1', 1, 'Pemula', 0, 'Pemula'),
  ('lvl-2', 2, 'Pejuang', 100, 'Pejuang'),
  ('lvl-3', 3, 'Aktif', 300, 'Aktif'),
  ('lvl-4', 4, 'Kontributor', 700, 'Kontributor'),
  ('lvl-5', 5, 'Inspirator', 1500, 'Inspirator'),
  ('lvl-6', 6, 'Legenda', 3000, 'Legenda')
ON CONFLICT DO NOTHING;

-- Reward misi disimpan per misi (missions.reward_xp), bukan di sini.
INSERT INTO xp_rules (id, source_type, xp, description) VALUES
  ('xpr-profile', 'profile_complete', 50, 'Profil komunitas dilengkapi (sekali per akun)'),
  ('xpr-checkin', 'checkin', 25, 'Check-in terverifikasi (sekali per akun per event)')
ON CONFLICT DO NOTHING;
