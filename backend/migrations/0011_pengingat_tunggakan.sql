-- 0011_pengingat_tunggakan.sql — tambahkan kolom pengingat tunggakan ke invoices.

-- last_reminder_at: kapan terakhir pengingat dikirim (untuk rate limiting).
alter table invoices add column if not exists last_reminder_at timestamp with time zone;

-- Indeks untuk query pencarian invoice yang perlu diingatkan.
create index if not exists idx_invoices_reminder
    on invoices (tenant_id, status, due_date)
    where status = 'UNPAID';
