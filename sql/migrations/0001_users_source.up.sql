-- Menandai dari mana akun berasal (register, otp, google, presence) tanpa
-- menebak dari keberadaan kolom lain.
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS source varchar;
