-- Misi harian ini memakai period_key daily Asia/Jakarta, sehingga satu akun
-- hanya dapat mengklaim satu kali per misi setiap hari.
INSERT INTO missions (id, code, title, description, type, verification_mode, reward_xp, claim_limit, version, starts_at, ends_at, is_published, sort_order, created_at, updated_at)
VALUES
  ('d0a10000-0000-4000-8000-000000000001', 'daily_subuh_jamaah', 'Subuh berjamaah di masjid', 'Check-in amalan Subuh berjamaah hari ini.', 'daily', 'auto', 2, 1, 1, '2026-01-01T00:00:00+07:00', '2099-01-01T00:00:00+07:00', true, 1, now(), now()),
  ('d0a10000-0000-4000-8000-000000000002', 'daily_sedekah_subuh', 'Sedekah Subuh', 'Check-in sedekah Subuh hari ini.', 'daily', 'auto', 2, 1, 1, '2026-01-01T00:00:00+07:00', '2099-01-01T00:00:00+07:00', true, 2, now(), now())
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;
