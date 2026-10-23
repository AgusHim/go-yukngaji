DROP TABLE IF EXISTS voucher_redemptions;
DROP TABLE IF EXISTS vouchers;
ALTER TABLE orders DROP COLUMN IF EXISTS voucher_code;
ALTER TABLE orders DROP COLUMN IF EXISTS discount_amount;
