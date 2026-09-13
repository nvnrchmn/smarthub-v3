-- 0007_superadmin.sql — akun superadmin + pengaturan global platform.
-- Idempoten.
BEGIN;

-- Akun superadmin: peran global yang tidak terikat tenant.
CREATE TABLE IF NOT EXISTS superadmins (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    full_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'ACTIVE',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Pengaturan global platform (reset password, Xendit key, dll).
CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Superadmin tidak punya batas tenant, jadi tidak pakai RLS.
-- Pastikan hanya ada satu baris untuk setiap key settings.

COMMIT;
