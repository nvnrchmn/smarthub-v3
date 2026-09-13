-- 0001_init.sql — fondasi multi-tenant Smarthub.
-- Isolasi tenant ditegakkan di level database (RLS), bukan hanya di kode.
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    phone         TEXT,
    full_name     TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('RESIDENT','SECRETARY','TREASURER','TENANT_MANAGER')),
    status        TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','ACTIVE','SUSPENDED')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE TABLE IF NOT EXISTS invite_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL,
    otp_hash   TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS houses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    block      TEXT NOT NULL,
    number     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, block, number)
);
-- Data sensus: NIK & KK disimpan TERENKRIPSI (AES-256-GCM di aplikasi).
-- Kolom *_enc disimpan sebagai BYTEA; plaintext TIDAK pernah masuk database.
CREATE TABLE IF NOT EXISTS residents (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    house_id   UUID REFERENCES houses(id) ON DELETE SET NULL,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    full_name  TEXT NOT NULL,
    nik_enc    BYTEA,
    kk_enc     BYTEA,
    status     TEXT NOT NULL DEFAULT 'PENDING_VERIFICATION',
    verified_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  UUID NOT NULL,
    actor_id   UUID,
    action     TEXT NOT NULL,
    entity     TEXT,
    entity_id  TEXT,
    detail     JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- RLS: setiap koneksi aplikasi menyetel app.tenant_id; baris tenant lain tidak
-- akan terlihat walau query lupa memakai WHERE tenant_id.
ALTER TABLE users        ENABLE ROW LEVEL SECURITY;
ALTER TABLE houses       ENABLE ROW LEVEL SECURITY;
ALTER TABLE residents    ENABLE ROW LEVEL SECURITY;
ALTER TABLE invite_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs   ENABLE ROW LEVEL SECURITY;

DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['users','houses','residents','invite_tokens','audit_logs'] LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = t AND policyname = 'tenant_isolation') THEN
      EXECUTE format(
        'CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid)', t);
    END IF;
  END LOOP;
END $$;
-- Peran aplikasi: BUKAN superuser (superuser melewati RLS) dan tidak memiliki BYPASSRLS.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'smarthub_app') THEN
    CREATE ROLE smarthub_app LOGIN;
  END IF;
END $$;

GRANT USAGE ON SCHEMA public TO smarthub_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO smarthub_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO smarthub_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO smarthub_app;

COMMIT;