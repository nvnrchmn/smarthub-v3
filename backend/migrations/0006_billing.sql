-- 0006_billing.sql — tagihan iuran (QRIS + kas tunai) sesuai PRD/ERD. Idempoten.
BEGIN;

-- Konfigurasi penagihan per tenant (PRD: tanggal terbit default 1, jatuh tempo H+n)
CREATE TABLE IF NOT EXISTS tenant_billing_settings (
    tenant_id  UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    billing_day INTEGER NOT NULL DEFAULT 1 CHECK (billing_day BETWEEN 1 AND 28),
    due_days    INTEGER NOT NULL DEFAULT 14,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Master iuran (Bendahara): IPL, Keamanan, Sampah, dst.
CREATE TABLE IF NOT EXISTS fee_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code       TEXT NOT NULL,
    label      TEXT NOT NULL,
    amount     NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    applies_to TEXT NOT NULL DEFAULT 'ALL' CHECK (applies_to IN ('ALL','OCCUPIED','VACANT')),
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

-- Invoice menginduk pada UNIT rumah (PRD), bukan akun warga.
CREATE TABLE IF NOT EXISTS invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    house_unit_id   UUID NOT NULL REFERENCES house_units(id) ON DELETE CASCADE,
    invoice_number  TEXT NOT NULL UNIQUE,
    period          DATE NOT NULL,
    base_amount     NUMERIC(14,2) NOT NULL DEFAULT 0,
    arrears_amount  NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_amount    NUMERIC(14,2) NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'UNPAID' CHECK (status IN ('UNPAID','PAID','VOID')),
    due_date        DATE NOT NULL,
    paid_at         TIMESTAMPTZ,
    -- jalur QRIS (hub Logikraf): reference dipakai untuk cek status, tidak ada webhook
    qris_reference   TEXT,
    qris_string      TEXT,
    qris_expires_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, house_unit_id, period)
);

CREATE TABLE IF NOT EXISTS invoice_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    label      TEXT NOT NULL,
    amount     NUMERIC(14,2) NOT NULL,
    kind       TEXT NOT NULL DEFAULT 'CURRENT' CHECK (kind IN ('CURRENT','ARREARS'))
);

-- Pembayaran: QRIS (via gateway) atau kas tunai yang diterima Bendahara.
CREATE TABLE IF NOT EXISTS payments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id       UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    payer_resident_id UUID REFERENCES resident_profiles(id) ON DELETE SET NULL,
    cash_receiver_id UUID REFERENCES resident_profiles(id) ON DELETE SET NULL,
    received_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    receipt_number   TEXT NOT NULL UNIQUE,
    payment_method   TEXT NOT NULL CHECK (payment_method IN ('QRIS_DYNAMIC','MANUAL_CASH')),
    amount_paid      NUMERIC(14,2) NOT NULL CHECK (amount_paid > 0),
    gateway_reference TEXT,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- satu invoice hanya boleh punya satu pembayaran aktif (anti dobel bayar)
    UNIQUE (invoice_id)
);

-- Buku kas: sumber dana membedakan uang di gateway vs uang fisik Bendahara.
CREATE TABLE IF NOT EXISTS cash_ledgers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    payment_id    UUID REFERENCES payments(id) ON DELETE SET NULL,
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    source_type   TEXT NOT NULL CHECK (source_type IN ('BANK_GATEWAY','PETTY_CASH_TREASURER','BANK_ACCOUNT')),
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('IN','OUT')),
    amount        NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    transfer_group UUID,
    approved_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at   TIMESTAMPTZ,
    evidence_file_path TEXT,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS invoices_unit_period_idx ON invoices (tenant_id, house_unit_id, period DESC);
CREATE INDEX IF NOT EXISTS invoices_status_idx ON invoices (tenant_id, status);
CREATE INDEX IF NOT EXISTS cash_ledgers_source_idx ON cash_ledgers (tenant_id, source_type);

-- RLS + hak akses peran aplikasi untuk semua tabel baru
DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['tenant_billing_settings','fee_items','invoices','invoice_items','payments','cash_ledgers'] LOOP
    IF t IN ('invoice_items','payments','cash_ledgers') THEN
      -- tabel anak: isolasi lewat tenant_id juga (payments/cash_ledgers) atau induknya (invoice_items)
      IF t = 'invoice_items' THEN
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = t AND policyname = 'tenant_isolation') THEN
          EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (exists (select 1 from invoices i where i.id = invoice_id and i.tenant_id = current_setting(''app.tenant_id'', true)::uuid)) WITH CHECK (exists (select 1 from invoices i where i.id = invoice_id and i.tenant_id = current_setting(''app.tenant_id'', true)::uuid))', t);
        END IF;
      ELSE
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = t AND policyname = 'tenant_isolation') THEN
          EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid) WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)', t);
        END IF;
      END IF;
    ELSE
      EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
      IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = t AND policyname = 'tenant_isolation') THEN
        EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid) WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)', t);
      END IF;
    END IF;
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO smarthub_app', t);
  END LOOP;
END $$;

COMMIT;
