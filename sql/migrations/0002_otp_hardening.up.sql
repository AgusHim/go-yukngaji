-- OTP sekali pakai dengan batas percobaan dan jejak waktu kirim.
ALTER TABLE IF EXISTS otp_tx ADD COLUMN IF NOT EXISTS attempts integer NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS otp_tx ADD COLUMN IF NOT EXISTS used_at timestamp;
ALTER TABLE IF EXISTS otp_tx ADD COLUMN IF NOT EXISTS last_sent_at timestamp;

-- Pencarian kode aktif terbaru per email.
CREATE INDEX IF NOT EXISTS idx_otp_tx_email_created_at ON otp_tx (email, created_at DESC);
