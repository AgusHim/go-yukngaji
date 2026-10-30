-- Membuka link QnA adalah check-in mandiri. Satu user hanya boleh memiliki
-- satu record per event per hari WIB; hari berikutnya tetap mendapat baris baru.
UPDATE presence
SET check_in_date = (created_at AT TIME ZONE 'Asia/Jakarta')::date
WHERE user_ticket_id IS NULL
  AND check_in_date IS NULL
  AND created_at IS NOT NULL;

WITH ranked AS (
  SELECT id,
         row_number() OVER (
           PARTITION BY user_id, event_id, check_in_date
           ORDER BY created_at ASC NULLS LAST, id ASC
         ) AS rn
  FROM presence
  WHERE user_ticket_id IS NULL
    AND check_in_date IS NOT NULL
    AND deleted_at IS NULL
)
UPDATE presence p
SET deleted_at = now()
FROM ranked r
WHERE p.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_presence_qna_user_event_check_in_date
  ON presence (user_id, event_id, check_in_date)
  WHERE user_ticket_id IS NULL
    AND check_in_date IS NOT NULL
    AND deleted_at IS NULL;
