-- 0002_invite_auth.sql — dipakai fitur undangan + OTP + login.
-- Idempoten: aman dijalankan ulang setiap deploy.
BEGIN;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ;

ALTER TABLE invite_tokens ADD COLUMN IF NOT EXISTS phone TEXT;
ALTER TABLE invite_tokens ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;
ALTER TABLE invite_tokens ADD COLUMN IF NOT EXISTS otp_sent_at TIMESTAMPTZ;
ALTER TABLE invite_tokens ADD COLUMN IF NOT EXISTS token_hash TEXT;

COMMIT;

-- Login tidak punya konteks tenant sebelum user terverifikasi, sedangkan RLS
-- menyembunyikan semua baris tanpa app.tenant_id. Karena itu lookup kredensial
-- dibuat sebagai fungsi SECURITY DEFINER dengan kolom terbatas — RLS tetap utuh
-- untuk seluruh query normal.
CREATE OR REPLACE FUNCTION auth_lookup_user(p_email TEXT)
RETURNS TABLE (id UUID, tenant_id UUID, password_hash TEXT, role TEXT, status TEXT, full_name TEXT, phone TEXT)
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
  SELECT u.id, u.tenant_id, u.password_hash, u.role, u.status, u.full_name, u.phone
  FROM users u WHERE lower(u.email) = lower(p_email) LIMIT 1;
$$;

-- Aktivasi undangan: pemegang token harus bisa menemukan undangannya tanpa tahu tenant.
CREATE OR REPLACE FUNCTION auth_lookup_invite(p_token_hash TEXT)
RETURNS TABLE (id UUID, tenant_id UUID, email TEXT, phone TEXT, role TEXT, otp_hash TEXT,
               expires_at TIMESTAMPTZ, used_at TIMESTAMPTZ, attempts INT, otp_sent_at TIMESTAMPTZ)
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
  SELECT i.id, i.tenant_id, i.email, i.phone, i.role, i.otp_hash, i.expires_at, i.used_at, i.attempts, i.otp_sent_at
  FROM invite_tokens i WHERE i.token_hash = p_token_hash LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION auth_lookup_user(TEXT) TO smarthub_app;
GRANT EXECUTE ON FUNCTION auth_lookup_invite(TEXT) TO smarthub_app;
