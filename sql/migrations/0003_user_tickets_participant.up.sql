-- Tiket rombongan: user_id tetap pembeli (data lama), sedangkan
-- participant_user_id menunjuk akun peserta sebenarnya bila emailnya cocok.
ALTER TABLE IF EXISTS user_tickets ADD COLUMN IF NOT EXISTS participant_user_id varchar;

CREATE INDEX IF NOT EXISTS idx_user_tickets_participant_user_id
  ON user_tickets (participant_user_id);

-- Backfill: cocokkan email peserta ke akun yang sudah ada. Bila satu email
-- dipakai beberapa akun, diambil akun paling awal supaya hasilnya deterministik.
UPDATE user_tickets ut
SET participant_user_id = matched.id
FROM (
  SELECT DISTINCT ON (lower(email)) lower(email) AS email, id
  FROM users
  WHERE deleted_at IS NULL
    AND email IS NOT NULL
    AND email <> ''
  ORDER BY lower(email), created_at ASC NULLS LAST, id ASC
) AS matched
WHERE ut.participant_user_id IS NULL
  AND ut.deleted_at IS NULL
  AND lower(ut.user_email) = matched.email;

ALTER TABLE IF EXISTS user_tickets DROP CONSTRAINT IF EXISTS fk_user_tickets_participant;
ALTER TABLE IF EXISTS user_tickets
  ADD CONSTRAINT fk_user_tickets_participant
  FOREIGN KEY (participant_user_id) REFERENCES users (id) ON DELETE SET NULL;
