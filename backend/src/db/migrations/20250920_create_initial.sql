-- migration/20250920_create_initial.sql
-- Smarthub V3 — Initial schema (Drizzle-generated, manually reviewed)
--
-- RLS aktif di semua tabel tenant-scoped.
-- Trigger update_updated_at otomatis di semua tabel yang punya updated_at.

-- ────── Extension ──────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ────── Tenants ──────
CREATE TABLE IF NOT EXISTS "tenants" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "name" TEXT NOT NULL,
  "slug" TEXT NOT NULL UNIQUE,
  "created_by" UUID REFERENCES login_accounts(id),
  "created_at" TIMESTAMPTZ DEFAULT now(),
  "updated_at" TIMESTAMPTZ DEFAULT now()
);

-- ────── Login Accounts ──────
CREATE TABLE IF NOT EXISTS "login_accounts" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "email" TEXT UNIQUE,
  "phone" TEXT UNIQUE,
  "password_hash" TEXT NOT NULL,
  "status" TEXT NOT NULL DEFAULT 'invited',
  "roles" TEXT[] NOT NULL DEFAULT '{}',
  "last_login" TIMESTAMPTZ,
  "created_at" TIMESTAMPTZ DEFAULT now(),
  "updated_at" TIMESTAMPTZ DEFAULT now()
);

-- ────── Profiles ──────
CREATE TABLE IF NOT EXISTS "profiles" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "login_account_id" UUID NOT NULL UNIQUE REFERENCES login_accounts(id) ON DELETE CASCADE,
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "full_name" TEXT NOT NULL,
  "no_ktp" TEXT UNIQUE,
  "no_kk" TEXT UNIQUE,
  "birth_date" DATE,
  "gender" TEXT,
  "address" TEXT,
  "avatar_url" TEXT,
  "is_primary_contact" BOOLEAN DEFAULT false,
  "created_at" TIMESTAMPTZ DEFAULT now(),
  "updated_at" TIMESTAMPTZ DEFAULT now()
);

-- ────── Houses ──────
CREATE TABLE IF NOT EXISTS "houses" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "blok" TEXT,
  "nomor_rumah" TEXT,
  "daya_listrik" NUMERIC(5),
  "luas_bangunan" NUMERIC,
  "luas_tanah" NUMERIC,
  "status" TEXT NOT NULL DEFAULT 'OCCUPIED',
  "created_at" TIMESTAMPTZ DEFAULT now(),
  "updated_at" TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS "uk_houses_rt" ON "houses" ("tenant_id", "blok", "nomor_rumah");

-- ────── House Occupants (pivot) ──────
CREATE TABLE IF NOT EXISTS "house_occupants" (
  "house_id" UUID NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
  "profile_id" UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  "relationship" TEXT NOT NULL,
  "moved_in" TIMESTAMPTZ,
  "moved_out" TIMESTAMPTZ,
  PRIMARY KEY ("house_id", "profile_id")
);

CREATE INDEX IF NOT EXISTS "idx_occupants_profile" ON "house_occupants" ("profile_id");
CREATE INDEX IF NOT EXISTS "idx_occupants_house" ON "house_occupants" ("house_id");

-- ────── Billing Items ──────
CREATE TABLE IF NOT EXISTS "billing_items" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "name" TEXT NOT NULL,
  "amount" NUMERIC(12,2) NOT NULL,
  "type" TEXT NOT NULL DEFAULT 'RECURRING',
  "is_mandatory" BOOLEAN NOT NULL DEFAULT true,
  "created_at" TIMESTAMPTZ DEFAULT now(),
  "updated_at" TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS "uk_billing_name" ON "billing_items" ("tenant_id", "name");

-- ────── Invoices ──────
CREATE TABLE IF NOT EXISTS "invoices" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "house_id" UUID NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
  "billing_item_id" UUID NOT NULL REFERENCES billing_items(id),
  "amount" NUMERIC(12,2) NOT NULL,
  "period_month" DATE NOT NULL,
  "due_date" DATE NOT NULL,
  "paid_at" TIMESTAMPTZ,
  "status" TEXT NOT NULL DEFAULT 'PENDING',
  "created_at" TIMESTAMPTZ DEFAULT now(),
  "updated_at" TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT "ck_inv_amount" CHECK ("amount" >= 0)
);

CREATE INDEX IF NOT EXISTS "idx_invoices_tenant" ON "invoices" ("tenant_id");
CREATE INDEX IF NOT EXISTS "idx_invoices_status" ON "invoices" ("status");
CREATE INDEX IF NOT EXISTS "idx_invoices_house_period" ON "invoices" ("house_id", "period_month");
CREATE UNIQUE INDEX IF NOT EXISTS "uk_invoices" ON "invoices" ("tenant_id", "house_id", "billing_item_id", "period_month");

-- ────── Payments ──────
CREATE TABLE IF NOT EXISTS "payments" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "invoice_id" UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "house_id" UUID NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
  "login_account_id" UUID NOT NULL REFERENCES login_accounts(id),
  "amount" NUMERIC(12,2) NOT NULL,
  "external_ref" TEXT,
  "paid_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "created_at" TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT "ck_pay_amount" CHECK ("amount" >= 0)
);

CREATE INDEX IF NOT EXISTS "idx_payments_tenant" ON "payments" ("tenant_id");
CREATE INDEX IF NOT EXISTS "idx_payments_paid_at" ON "payments" ("paid_at");
CREATE INDEX IF NOT EXISTS "idx_payments_invoice" ON "payments" ("invoice_id");

-- ────── Audit Log ──────
CREATE TABLE IF NOT EXISTS "audit_log" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "login_account_id" UUID REFERENCES login_accounts(id),
  "action" TEXT NOT NULL,
  "entity" TEXT NOT NULL,
  "entity_id" UUID,
  "detail" JSONB,
  "ip_address" INET,
  "created_at" TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS "idx_audit_tenant" ON "audit_log" ("tenant_id");
CREATE INDEX IF NOT EXISTS "idx_audit_action" ON "audit_log" ("action");

-- ──────────────────────────────────────────────
-- Trigger: update_updated_at
-- Di semua tabel yang punya kolom updated_at.
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION "update_updated_at"()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' THEN
    NEW."updated_at" := now();
  END IF;
  RETURN NEW;
END $$;

CREATE TRIGGER "trg_tenants_updated"
  BEFORE UPDATE ON "tenants"
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER "trg_login_accounts_updated"
  BEFORE UPDATE ON "login_accounts"
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER "trg_profiles_updated"
  BEFORE UPDATE ON "profiles"
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER "trg_houses_updated"
  BEFORE UPDATE ON "houses"
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER "trg_billing_items_updated"
  BEFORE UPDATE ON "billing_items"
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER "trg_invoices_updated"
  BEFORE UPDATE ON "invoices"
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ──────────────────────────────────────────────
-- Trigger: log_audit
-- Otomatis log aksi bisnis di invoices + payments + houses.
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION "log_audit"()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
  v_tenant_id UUID;
  v_login_account_id UUID;
BEGIN
  -- Ambil tenant_id dari record yang dioperasikan
  CASE TG_TABLE_NAME
    WHEN 'invoices' THEN
      SELECT house_id INTO v_tenant_id FROM houses WHERE id = NEW.house_id LIMIT 1;
    WHEN 'payments' THEN
      SELECT tenant_id INTO v_tenant_id FROM payments WHERE id = NEW.id;
    WHEN 'houses' THEN
      SELECT tenant_id INTO v_tenant_id FROM houses WHERE id = NEW.id;
  END CASE;

  IF TG_OP = 'DELETE' THEN
    INSERT INTO audit_log (tenant_id, login_account_id, action, entity, entity_id, detail)
    VALUES (v_tenant_id, current_setting('app.current_user_id')::UUID, 'delete', TG_TABLE_NAME, OLD.id, row_to_json(OLD));
    RETURN OLD;
  ELSIF TG_OP = 'INSERT' THEN
    INSERT INTO audit_log (tenant_id, login_account_id, action, entity, entity_id, detail)
    VALUES (v_tenant_id, current_setting('app.current_user_id')::UUID, 'insert', TG_TABLE_NAME, NEW.id, row_to_json(NEW));
    RETURN NEW;
  ELSIF TG_OP = 'UPDATE' THEN
    INSERT INTO audit_log (tenant_id, login_account_id, action, entity, entity_id, detail)
    VALUES (v_tenant_id, current_setting('app.current_user_id')::UUID, 'update', TG_TABLE_NAME, NEW.id, row_to_json(NEW));
    RETURN NEW;
  END IF;
  RETURN NEW;
END $$;

CREATE TRIGGER "trg_invoices_audit"
  AFTER INSERT OR UPDATE OR DELETE ON "invoices"
  FOR EACH ROW EXECUTE FUNCTION log_audit();

CREATE TRIGGER "trg_payments_audit"
  AFTER INSERT OR UPDATE OR DELETE ON "payments"
  FOR EACH ROW EXECUTE FUNCTION log_audit();

CREATE TRIGGER "trg_houses_audit"
  AFTER INSERT OR UPDATE OR DELETE ON "houses"
  FOR EACH ROW EXECUTE FUNCTION log_audit();

-- ──────────────────────────────────────────────
-- Trigger: update_cash_balance
-- Otomatis update aggregate balance setiap payment.
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION "update_cash_balance"()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
    INSERT INTO cash_balances (tenant_id, period, total_paid)
    VALUES (NEW.tenant_id, to_char(NEW.paid_at, 'YYYY-MM'), NEW.amount)
    ON CONFLICT (tenant_id, period)
    DO UPDATE SET total_paid = cash_balances.total_paid + NEW.amount;
  END IF;
  RETURN NEW;
END $$;

CREATE TABLE IF NOT EXISTS "cash_balances" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenant_id" UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  "period" TEXT NOT NULL, -- "2025-01"
  "total_paid" NUMERIC(12,2) DEFAULT 0,
  "total_unpaid" NUMERIC(12,2) DEFAULT 0,
  "total_saldo_awal" NUMERIC(12,2) DEFAULT 0,
  "updated_at" TIMESTAMPTZ DEFAULT now(),
  "created_at" TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS "uk_cash_balances" ON "cash_balances" ("tenant_id", "period");
CREATE INDEX IF NOT EXISTS "idx_cash_balances_tenant" ON "cash_balances" ("tenant_id");

CREATE TRIGGER "trg_payments_balance"
  AFTER INSERT OR UPDATE ON "payments"
  FOR EACH ROW EXECUTE FUNCTION update_cash_balance();

-- ──────────────────────────────────────────────
-- RLS Policy Template (harus diterapkan per tabel)
-- ──────────────────────────────────────────────
-- ENABLE ROW LEVEL SECURITY pada semua tabel tenant-scoped
-- Policy: tenant_isolation => current_setting('app.current_tenant')::UUID = tenant_id

ALTER TABLE "login_accounts" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "profiles" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "houses" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "house_occupants" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "billing_items" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "invoices" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "payments" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "audit_log" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "cash_balances" ENABLE ROW LEVEL SECURITY;

-- Superadmin role bypass RLS
CREATE POLICY "superadmin_bypass" ON "login_accounts"
  FOR ALL TO PUBLIC
  USING (
    (SELECT current_setting('app.current_tenant', true)::UUID) = tenant_id
    OR current_setting('app.current_user_roles', true)::TEXT LIKE '%SUPERADMIN%'
  )
  WITH CHECK (true);

-- ... policy yang sama diterapkan ke semua tabel di atas
-- (Drizzle Kit + RLS generator akan otomatisasi ini)
