-- 0010_add_reset_password.sql — tambahkan kolom reset password ke tabel users.
--
-- Token disimpan sebagai hash SHA-256 (hex 64 karakter).
-- Indeks pada hash untuk mempercepat lookup saat verifikasi.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS reset_token_hash text,
    ADD COLUMN IF NOT EXISTS reset_sent_at   timestamptz;

CREATE INDEX IF NOT EXISTS idx_users_reset_token_hash ON users (reset_token_hash);
