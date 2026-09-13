-- 0008_superadmin_rls.sql — superadmin (platform) boleh membaca lintas tenant.
--
-- Masalah: role aplikasi (smarthub_app) tunduk RLS, sehingga query global
-- (daftar semua tenant, audit log global) mengembalikan 0 baris.
-- Solusi: policy tambahan bersifat PERMISSIVE (di-OR dengan policy tenant),
-- hanya berlaku saat transaksi menyetel GUC app.superadmin='on'.
-- GUC tidak diset → current_setting(...,true) = NULL → policy tidak cocok.
--
-- Hak baca superadmin sengaja dibatasi: tenants + audit_logs saja (least
-- privilege). Tabel warga (house_units, resident_profiles, invoices, dst.)
-- tetap tidak terlihat lintas tenant.
--
-- Idempoten, aman dijalankan berulang.

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'tenants' AND policyname = 'tenants_superadmin'
    ) THEN
        CREATE POLICY tenants_superadmin ON tenants
            USING (current_setting('app.superadmin', true) = 'on');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'audit_logs' AND policyname = 'audit_logs_superadmin'
    ) THEN
        CREATE POLICY audit_logs_superadmin ON audit_logs
            USING (current_setting('app.superadmin', true) = 'on');
    END IF;
END $$;

COMMIT;
