-- 0005_house_kk.sql — kolom pendukung kelola rumah & kartu keluarga. Idempoten.
BEGIN;

-- 4 digit terakhir nomor KK untuk tampilan tanpa dekripsi
ALTER TABLE family_cards ADD COLUMN IF NOT EXISTS number_last4 TEXT;

-- jejak perubahan rumah
ALTER TABLE house_units ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- satu kartu keluarga tidak boleh dipakai dua tenant (jaga isolasi)
CREATE INDEX IF NOT EXISTS house_units_tenant_idx ON house_units (tenant_id);
CREATE INDEX IF NOT EXISTS family_cards_tenant_idx ON family_cards (tenant_id);
CREATE INDEX IF NOT EXISTS resident_profiles_card_idx ON resident_profiles (family_card_id);

COMMIT;
