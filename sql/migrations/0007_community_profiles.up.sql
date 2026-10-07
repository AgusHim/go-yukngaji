-- Profil komunitas: identitas publik (alias), preferensi privasi, dan izin
-- tampil leaderboard. Satu baris per akun, relasi 1:1 ke users.
CREATE TABLE IF NOT EXISTS community_profiles (
  user_id varchar PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  public_id varchar(32) NOT NULL,
  alias varchar(40),
  bio text,
  avatar_url varchar,
  profile_visibility varchar NOT NULL DEFAULT 'public',
  leaderboard_opt_out boolean NOT NULL DEFAULT false,
  is_blocked boolean NOT NULL DEFAULT false,
  show_badges boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- public_id adalah satu-satunya identifier yang boleh tampil di response
-- publik; id akun internal tidak pernah dikirim ke luar.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_community_profiles_public_id
  ON community_profiles (public_id);

-- Alias harus unik tanpa peduli huruf besar/kecil, tetapi boleh kosong.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_community_profiles_alias_lower
  ON community_profiles (lower(alias))
  WHERE alias IS NOT NULL AND alias <> '';

CREATE INDEX IF NOT EXISTS idx_community_profiles_visibility
  ON community_profiles (leaderboard_opt_out, is_blocked);

-- ALTER TABLE ... ADD CONSTRAINT tidak idempotent dan akan gagal bila file
-- ini dijalankan ulang, jadi dibungkus supaya aman diulang.
DO $$
BEGIN
  ALTER TABLE community_profiles ADD CONSTRAINT chk_community_profiles_visibility
    CHECK (profile_visibility IN ('public', 'members', 'private'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

-- Backfill untuk akun yang sudah ada. alias sengaja dibiarkan NULL: username
-- lama bisa duplikat, dan kegagalan backfill tidak boleh menghalangi migrasi.
-- Tampilan jatuh ke users.username selama alias belum diisi.
INSERT INTO community_profiles (user_id, public_id)
SELECT u.id, substr(replace(gen_random_uuid()::text, '-', ''), 1, 16)
FROM users u
WHERE u.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM community_profiles cp WHERE cp.user_id = u.id)
ON CONFLICT (user_id) DO NOTHING;
