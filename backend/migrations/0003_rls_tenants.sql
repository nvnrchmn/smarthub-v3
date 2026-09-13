-- 0003_rls_tenants.sql — tutup celah: tabel tenants sebelumnya tanpa RLS.
-- Idempoten, aman dijalankan berulang.
BEGIN;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'tenants' AND policyname = 'tenants_isolation'
    ) THEN
        CREATE POLICY tenants_isolation ON tenants
            USING (id = current_setting('app.tenant_id', true)::uuid)
            WITH CHECK (id = current_setting('app.tenant_id', true)::uuid);
    END IF;
END $$;

COMMIT;
