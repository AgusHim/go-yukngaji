-- Check-in idempotent per hari: satu tiket hanya boleh menghasilkan satu
-- baris presence per tanggal (zona Asia/Jakarta).
ALTER TABLE IF EXISTS presence ADD COLUMN IF NOT EXISTS check_in_date date;

-- Backfill tanggal dari created_at. created_at disimpan sebagai timestamp
-- tanpa zona, jadi dikonversi dengan asumsi waktu Asia/Jakarta.
UPDATE presence
SET check_in_date = (created_at AT TIME ZONE 'Asia/Jakarta')::date
WHERE check_in_date IS NULL
  AND created_at IS NOT NULL;

-- Rapikan duplikat lama (tiket + tanggal sama) dengan soft delete pada semua
-- baris selain yang paling awal, supaya unique index bisa dibuat.
WITH ranked AS (
  SELECT id,
         row_number() OVER (
           PARTITION BY user_ticket_id, check_in_date
           ORDER BY created_at ASC NULLS LAST, id ASC
         ) AS rn
  FROM presence
  WHERE user_ticket_id IS NOT NULL
    AND check_in_date IS NOT NULL
    AND deleted_at IS NULL
)
UPDATE presence p
SET deleted_at = now()
FROM ranked r
WHERE p.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_presence_ticket_check_in_date
  ON presence (user_ticket_id, check_in_date)
  WHERE user_ticket_id IS NOT NULL
    AND check_in_date IS NOT NULL
    AND deleted_at IS NULL;
