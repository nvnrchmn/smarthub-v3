-- 0004_census.sql — skema sensus sesuai ERD (house_units, family_cards,
-- resident_profiles, house_occupancies). Idempoten.
BEGIN;

-- PRD: tenant punya kode lingkungan, mis. GRAHA-ASRI-RW08
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS code TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS tenants_code_key ON tenants (code) WHERE code IS NOT NULL;

-- Tabel dari 0001 (houses/residents) tidak dipakai dan kosong; diganti struktur ERD.
-- urutan penting: residents punya FK ke houses
DROP TABLE IF EXISTS residents;
DROP TABLE IF EXISTS houses;

CREATE TABLE IF NOT EXISTS house_units (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    block            VARCHAR(10) NOT NULL,
    unit_number      VARCHAR(10) NOT NULL,
    occupancy_status TEXT NOT NULL DEFAULT 'VACANT'
                     CHECK (occupancy_status IN ('OCCUPIED','VACANT','RENOVATION')),
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, block, unit_number)
);

CREATE TABLE IF NOT EXISTS family_cards (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- PRD 281: nomor KK wajib terenkripsi (AES-256-GCM, AAD = tenant_id)
    family_card_number_enc BYTEA,
    number_last4           TEXT,
    kk_file_path           TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS resident_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    family_card_id      UUID REFERENCES family_cards(id) ON DELETE SET NULL,
    full_name           VARCHAR(150) NOT NULL,
    nik_enc             BYTEA,
    nik_last4           TEXT, -- untuk tampilan tersamar tanpa perlu dekripsi
    family_role         TEXT NOT NULL DEFAULT 'OTHER'
                        CHECK (family_role IN ('HEAD_OF_FAMILY','SPOUSE','CHILD','OTHER')),
    birth_place         TEXT,
    birth_date          DATE,
    gender              TEXT CHECK (gender IN ('M','F')),
    religion            TEXT,
    marital_status      TEXT,
    occupation          TEXT,
    education           TEXT,
    ktp_file_path       TEXT,
    verification_status TEXT NOT NULL DEFAULT 'UNVERIFIED'
                        CHECK (verification_status IN ('UNVERIFIED','VERIFIED','REJECTED')),
    lifecycle_status    TEXT NOT NULL DEFAULT 'ACTIVE'
                        CHECK (lifecycle_status IN ('ACTIVE','MOVED_OUT','DECEASED')),
    rejection_reason    TEXT,
    verified_at         TIMESTAMPTZ,
    verified_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- kolom untuk basis data yang sudah dibuat sebelum penambahan kolom ini
ALTER TABLE resident_profiles ADD COLUMN IF NOT EXISTS nik_last4 TEXT;
ALTER TABLE family_cards      ADD COLUMN IF NOT EXISTS number_last4 TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS resident_profiles_account_key
    ON resident_profiles (tenant_id, account_id) WHERE account_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS house_occupancies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resident_id     UUID NOT NULL REFERENCES resident_profiles(id) ON DELETE CASCADE,
    house_unit_id   UUID NOT NULL REFERENCES house_units(id) ON DELETE CASCADE,
    occupancy_type  TEXT NOT NULL DEFAULT 'OWNER_OCCUPANT'
                    CHECK (occupancy_type IN ('OWNER_OCCUPANT','TENANT','OWNER_NON_RESIDENT')),
    is_primary_payer BOOLEAN NOT NULL DEFAULT false,
    start_date      DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS house_occupancies_active_key
    ON house_occupancies (house_unit_id, resident_id) WHERE end_date IS NULL;

-- RLS + policy isolasi tenant untuk seluruh tabel baru
DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['house_units','family_cards','resident_profiles','house_occupancies'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = t AND policyname = 'tenant_isolation') THEN
            EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid) WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)', t);
        END IF;
        EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO smarthub_app', t);
    END LOOP;
END $$;

-- audit_logs bersifat append-only: peran aplikasi tidak boleh mengubah/menghapus
REVOKE UPDATE, DELETE ON audit_logs FROM smarthub_app;
GRANT INSERT, SELECT ON audit_logs TO smarthub_app;
GRANT USAGE, SELECT ON SEQUENCE audit_logs_id_seq TO smarthub_app;

COMMIT;
