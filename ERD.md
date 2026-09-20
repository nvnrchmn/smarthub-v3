ERD.md — Database Schema
Smarthub V3 — Fullstack TypeScript

Stack: Bun + Hono + Drizzle ORM → PostgreSQL 16
Multi-tenant: setiap tenant = 1 organisasi (RT/RW). Data diisolasi via PostgreSQL Row Level Security + Drizzle tenant-aware query.

=== ENTITIES === 14 tabel

--- tenants ---
id (PK, UUID)
name — nama organisasi  
slug — unik, URL-friendly
created_by — user ID, superadmin pertama
created_at, updated_at
[Superadmin 1 table, banyak tenant]

--- users ---
id (PK, UUID)
tenant_id (FK → tenants)
email
full_name
phone
password_hash
roles[] — SET<Role> (bisa ganda: TENANT_MANAGER, SECRETARY, TREASURER, SECRETARY, RESIDENT)
status — active / suspended / invited
invite_token
invite_expires_at
last_login
created_by — siapa yg invite
created_at, updated_at
[RLS: tenant-scoped. 5 roles per tenant]

--- houses ---
id (PK, UUID)
tenant_id (FK)
nomor_rumah — contoh "03/12"
blok — contoh "Blok A"
daya_listrik — 900, 1300, 2200, 3500, dst
luas_bangunan — meter²
luas_tanah — meter²
nama_kepala — warga kepala
nik_nokk — 16 digit
alamat_lengkap
status — OCCUPIED / VACANT
created_at, updated_at
[RLS: tenant-scoped]

--- kartu_keluarga ---
id (PK, UUID)
house_id (FK → houses)
no_kk — 16 digit
nama_kepala
total_anggota
created_at, updated_at
[1 rumah bisa 1 KK, atau 0 jika vacant]

--- census_records ---
id (PK, UUID)
house_id (FK → houses)
tenant_id (FK)
responden_user_id (FK → users, Sekretaris/Pendata)
isi — JSONB (pertanyaan dinamis, struktur fleksibel)
submitted_at
created_at
[Data sensus dari Sekretaris/Pendata, bisa berisi foto/KK]

--- billing_items ---
id (PK, UUID)
tenant_id (FK)
name — "Iuran Wajib", "Pasar Minggu", dst
amount — nominal
type — RECURRING / ONE_TIME / VARIABLE
freq — MONTHLY / QUARTERLY / YEARLY  
month_day — untuk recurring
is_mandatory — boolean
start_at, end_at
created_by — user
created_at, updated_at
[Master data iuran per tenant]

--- invoices ---
id (PK, UUID)
tenant_id (FK)
house_id (FK → houses)
billing_item_id (FK → billing_items)
amount — rupiah
status — DRAFT / PENDING / PAID / OVERDUE
due_date
paid_at
created_at, updated_at
[Tagihan khusus untuk 1 rumah]

--- invoice_items ---
id (PK, UUID)
invoice_id (FK → invoices)
billing_item_id (FK → billing_items)
amount
description
[Item breakdown dalam 1 invoice]

--- payments ---
id (PK, UUID)
invoice_id (FK → invoices)
tenant_id (FK)
house_id (FK)
user_id — pembayar (warga)
amount
payment_method — QRIS / XCARD / EDC
external_ref — Xendit charge ID
paid_at
created_at
[Setiap pembayaran link ke 1 invoice, support partial payment]

--- audit_log ---
id (PK, UUID)
tenant_id (FK)
user_id — aktor
action — "create_invoice", "mark_paid", "census_submit"
entity — "invoices", "houses"
entity_id — UUID yang dimaksud
detail — JSONB
ip_address
user_agent — truncated
created_at
[Wajib semua aksi bisnis ter-log]

--- cash_balances ---
id (PK, UUID)
tenant_id (FK)
period — 2025-01, dst
total_paid — jumlah diterima bulan itu
total_unpaid — tunggakan
total_saldo_awal — saldo awal kas
[Cache rekap periode, di-refresh pas ada payment]

--- notifications ---
id (PK, UUID)
tenant_id (FK)
user_id (FK → users)
title
message
url_path — deep-link ke halaman
is_read
created_at
[Notifikasi sistem ke user (misal overdue, konfirmasi pembayaran)]

=== RELATIONSHIPS ===

tenants (1) → (M) users        — 1 tenant punya M user
tenants (1) → (M) houses      — 1 RT/RW punya M rumah
tenants (1) → (M) billing_items
tenants (1) → (M) invoices
tenants (1) → (M) payments
tenants (1) → (M) audit_log
tenants (1) → (M) cash_balances
tenants (1) → (M) notifications

houses (1) → (M) kartu_keluarga
houses (1) → (M) invoices
houses (1) → (M) payments
houses (1) → (M) census_records

users (1) → (M) census_records  (responden_user_id)
users (1) → (M) audit_log
users (1) → (M) notifications

billing_items (1) → (M) invoices
billing_items (1) → (M) invoice_items

invoices (1) → (M) invoice_items
invoices (1) → (M) payments

payments (1) → (M) audit_log    (via payment_id)

=== RLS RULES (PostgreSQL) ===
- Semua tabel dengan tenant_id: RLS aktif, policy `tenant_isolation`: `current_setting('app.current_tenant')::uuid = tenant_id`
- Set app.current_tenant di Hono middleware setelah auth JWT verified
- Audit log: policy `audit_isolation` juga memakai tenant scoping
- Superadmin (role TENANT_MANAGER di global level): bisa query SELECT semua tenant via `auth.uid() IN (global admins)`

=== CONSTRAINTS ===
- Unique: tenants.slug (case-insensitive) + tenants.name
- Unique: users.email + tenant_id
- Unique: houses.nomor_rumah + block + tenant_id
- Unique: no_kk + tenant_id
- payments.amount check (amount 0)
- invoices.due_date required jika status != DRAFT
- No soft-delete — gunakan status / archive flag

=== RBAC SCOPES (Role Permission Matrix) ===
Role            | User Mgmt | House Data | Census | Billing Create | Invoice Edit | Payments | Audit Log | Superadmin
Superadmin      | ✅ all     | ✅ all      | ✅ all  | ✅ global        | ✅ global    | ✅ all   | ✅ read   | ✅
Tenant Manager  | ✅ tenant  | ✅ tenant   | ✅ all  | ✅ tenant        | ✅ tenant    | ✅ all   | ✅ read   | ✅
Secretary       | ❌         | readonly    | ✅ all  | ✅ all           | ✅ all       | readonly | ❌        | ❌
Treasurer       | ❌         | readonly    | ❌      | ❌               | ✅ (approve) | ✅       | ✅ read   | ❌
Resident        | ❌         | own house   | own     | ❌               | own invoice  | own      | ❌        | ❌

=== INDEXING STRATEGY ===
- Primary key: UUID v7 (time-sortable) via `gen_random_uuidv7()`
- Index: tenant_id + created_at pada invoices, payments (untuk laporan)
- Index: houses.blocked + nomor_rumah (autocomplete)
- Index: users.email (login lookup)
- Partial index: invoices(status = 'PENDING') (reminders query)

=== TRIGGER NOTE ===
- cash_balances di-update via DB trigger AFTER INSERT/UPDATE payments
- audit_log otomatis terisi via trigger atau middleware central (pilih: trigger lebih reliable)

=== CONNECTIONS ===
- Hono context: app.current_tenant di-set dari JWT tenant field
- Drizzle: `db.select().from(tenants)` → otomatis ada clause `WHERE tenant_id = current_setting(...)`
- Migration via Drizzle Kit: `bun run migrate` → `pg_migrate`

=== DEMO DATA ===
- Sample tenant: "Perumahan Griya Asri Residence, RT 03/RW 07"
- Sample house: "Blok A No. 03" — kepala keluarga "Suharto" (NIK 1234567890123456)
- Sample user: "bendahara@griyaasri.test" (role TREASURER); "warga@griyaasri.test" (RESIDENT)

=== NOT ===
- Ini schema final — siap migrate ke Docker Postgres (port 15432)
- Semua tabel punya tenant_id kecuali tenants & audit_log (audit punya)
- receipts penuh di tabel payments, bukan di invoices