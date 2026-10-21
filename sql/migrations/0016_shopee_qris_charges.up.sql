-- Satu charge aktif merepresentasikan tepat satu tagihan lintas domain.
-- payable_amount unik selama belum dibayar/kedaluwarsa, sehingga mutasi Shopee
-- tidak dapat dicocokkan ke dua transaksi yang berbeda.
CREATE TABLE IF NOT EXISTS shopee_qris_charges (
    id              varchar(36) PRIMARY KEY,
    entity_type     varchar(24) NOT NULL CHECK (entity_type IN ('event_order', 'donation', 'shop_order')),
    entity_id       varchar(36) NOT NULL,
    base_amount     bigint NOT NULL CHECK (base_amount > 0),
    unique_code     integer NOT NULL CHECK (unique_code BETWEEN 1 AND 99999),
    payable_amount  bigint NOT NULL CHECK (payable_amount = base_amount + unique_code),
    qris_payload    text NOT NULL,
    status          varchar(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'expired', 'cancelled')),
    shopee_transaction_id varchar(160),
    expires_at      timestamptz NOT NULL,
    paid_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uniq_shopee_qris_entity UNIQUE (entity_type, entity_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_shopee_qris_payable_pending
    ON shopee_qris_charges (payable_amount)
    WHERE status = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS uniq_shopee_qris_transaction
    ON shopee_qris_charges (shopee_transaction_id)
    WHERE shopee_transaction_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_shopee_qris_pending_expiry
    ON shopee_qris_charges (status, expires_at);
