-- Jawaban tamu disimpan tanpa user_id dan ditandai belum terverifikasi,
-- sehingga tidak pernah dihitung sebagai aktivitas akun yang sah.
-- Prasyarat: tabel poll_responses sudah ada (sql/poll_migration.sql).
ALTER TABLE poll_responses ADD COLUMN IF NOT EXISTS is_verified boolean NOT NULL DEFAULT false;
ALTER TABLE poll_responses ALTER COLUMN user_id DROP NOT NULL;

-- Baris lama dibuat saat user_id masih wajib diisi dari client.
UPDATE poll_responses
SET is_verified = true
WHERE user_id IS NOT NULL
  AND is_verified = false;

CREATE INDEX IF NOT EXISTS idx_poll_responses_poll_user ON poll_responses (poll_id, user_id);
