-- Kompatibilitas untuk instalasi yang sudah menerapkan 0016 versi awal.
ALTER TABLE shopee_qris_charges
    ADD COLUMN IF NOT EXISTS settled_at timestamptz,
    ADD COLUMN IF NOT EXISTS settlement_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS settlement_error text,
    ADD COLUMN IF NOT EXISTS settlement_locked_until timestamptz;

CREATE INDEX IF NOT EXISTS idx_shopee_qris_unsettled
    ON shopee_qris_charges (status, settled_at, settlement_locked_until);

INSERT INTO payment_methods (
    id, name, type, code, image_url, account_name, account_number, created_at, updated_at
) VALUES (
    'shopee-qris', 'QRIS ShopeePay', 'QRIS', 'SHOPEE_QRIS', '', 'YN Solo', '-', now(), now()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    code = EXCLUDED.code,
    updated_at = now(),
    deleted_at = NULL;
