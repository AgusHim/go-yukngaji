DROP INDEX IF EXISTS idx_shopee_qris_unsettled;

ALTER TABLE shopee_qris_charges
    DROP COLUMN IF EXISTS settlement_locked_until,
    DROP COLUMN IF EXISTS settlement_error,
    DROP COLUMN IF EXISTS settlement_attempts,
    DROP COLUMN IF EXISTS settled_at;

-- Payment method sengaja dipertahankan bila migrasi dibalik: baris transaksi
-- lama dapat masih mereferensikannya dan riwayat pembayaran tidak boleh rusak.
