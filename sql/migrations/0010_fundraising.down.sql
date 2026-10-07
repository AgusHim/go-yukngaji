-- Pembalikan 0010_fundraising.
--
-- Urutannya kebalikan dari up: tabel anak lebih dulu, lalu tabel induk, supaya
-- tidak ada foreign key yang menggantung. Semua pernyataan idempoten.

DELETE FROM xp_rules WHERE id = 'xpr-donation';

ALTER TABLE IF EXISTS xp_rules DROP COLUMN IF EXISTS monthly_cap;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS campaign_updates;
DROP TABLE IF EXISTS donations;
DROP TABLE IF EXISTS campaigns;
