// src/db/schema.ts — Drizzle ORM schema untuk Smarthub V3
//
// Stack: Bun runtime + Hono + Drizzle ORM → PostgreSQL 16 + Redis
// RLS aktif di semua tabel tenant-scoped (policy otomatis via trigger)
// UUID v7 time-sortable untuk semua primary key

import {
  pgTable,
  text,
  uuid,
  boolean,
  timestamp,
  date,
  numeric,
  jsonb,
  inet,
  index,
  uniqueIndex,
} from 'drizzle-orm/pg-core'

// ──────────────────────────────────────────────
// tenants
// └─ Superadmin mengelola banyak tenant (RT/RW perumahan)
// ──────────────────────────────────────────────
export const tenants = pgTable(
  'tenants',
  {
    id: uuid('id').primaryKey().defaultRandom(), // gen_random_uuid() → v4; cukup untuk dev
    name: text('name').notNull(),
    slug: text('slug').notNull().unique(),
    createdBy: uuid('created_by').references(() => loginAccounts.id),
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
    updatedAt: timestamp('updated_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_tenants_slug').on(table.slug),
  ]
)

// ──────────────────────────────────────────────
// login_accounts
// └─ Hanya untuk auth: email OR phone + password hash
//    (salah satu wajib diisi, bukan keduanya kosong)
// ──────────────────────────────────────────────
export const loginAccounts = pgTable(
  'login_accounts',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    email: text('email').unique(), // nullable — bisa pakai phone saja
    phone: text('phone').unique(), // nullable — bisa pakai email saja
    passwordHash: text('password_hash').notNull(),
    status: text('status').notNull().default('invited'), // active | invited | suspended
    roles: text('roles').array().notNull().default([]), // string[]
    lastLogin: timestamp('last_login', { mode: 'string' }),
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
    updatedAt: timestamp('updated_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    uniqueIndex('uk_login_email').on(table.email),
    uniqueIndex('uk_login_phone').on(table.phone),
  ]
)

// ──────────────────────────────────────────────
// profiles
// └─ 1:1 ke login_accounts — data pribadi + avatar
// ──────────────────────────────────────────────
export const profiles = pgTable(
  'profiles',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    loginAccountId: uuid('login_account_id')
      .notNull()
      .references(() => loginAccounts.id)
      .unique(),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    fullName: text('full_name').notNull(),
    noKtp: text('no_ktp'), // 16 digit, nullable (opsional)
    noKk: text('no_kk'), // 16 digit, nullable
    birthDate: date('birth_date'),
    gender: text('gender'), // L | P
    address: text('address'),
    avatarUrl: text('avatar_url'), // MinIO: /avatars/<uuid>.webp
    isPrimaryContact: boolean('is_primary_contact').default(false),
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
    updatedAt: timestamp('updated_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_profiles_tenant').on(table.tenantId),
    uniqueIndex('uk_profiles_ktp').on(table.noKtp),
    uniqueIndex('uk_profiles_kk').on(table.noKk),
  ]
)

// ──────────────────────────────────────────────
// houses
// └─ Unit rumah per tenant — fokus manajemen rumah
// ──────────────────────────────────────────────
export const houses = pgTable(
  'houses',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    blok: text('blok'), // "Blok A", dst.
    nomorRumah: text('nomor_rumah'), // "03", "12A", dst.
    dayaListrik: numeric('daya_listrik', { precision: 5 }), // 900, 1300, 2200
    luasBangunan: numeric('luas_bangunan'), // m²
    luasTanah: numeric('luas_tanah'), // m²
    status: text('status').notNull().default('OCCUPIED'), // OCCUPIED | VACANT
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
    updatedAt: timestamp('updated_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_houses_tenant').on(table.tenantId),
    uniqueIndex('uk_houses_rt').on(table.tenantId, table.blok, table.nomorRumah),
  ]
)

// ──────────────────────────────────────────────
// house_occupants (pivot)
// └─ M-n: 1 rumah punya M profile; 1 profile bisa jadi primary contact
// ──────────────────────────────────────────────
export const houseOccupants = pgTable(
  'house_occupants',
  {
    houseId: uuid('house_id')
      .notNull()
      .references(() => houses.id)
      .primaryKey(),
    profileId: uuid('profile_id')
      .notNull()
      .references(() => profiles.id)
      .primaryKey(),
    relationship: text('relationship').notNull(), // KEPALA_RUMAH | ANGGOTA_KELUARGA
    movedIn: timestamp('moved_in', { mode: 'string' }),
    movedOut: timestamp('moved_out', { mode: 'string' }),
  },
  (table) => [
    index('idx_occupants_profile').on(table.profileId),
    index('idx_occupants_house').on(table.houseId),
  ]
)

// ──────────────────────────────────────────────
// billing_items
// └─ Master template iuran (RECURRING / ONE_TIME)
// ──────────────────────────────────────────────
export const billingItems = pgTable(
  'billing_items',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    name: text('name').notNull(),
    amount: numeric('amount', { precision: 12, scale: 2 }).notNull(),
    type: text('type').notNull().default('RECURRING'), // RECURRING | ONE_TIME
    isMandatory: boolean('is_mandatory').notNull().default(true),
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
    updatedAt: timestamp('updated_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_billing_tenant').on(table.tenantId),
    uniqueIndex('uk_billing_name').on(table.tenantId, table.name),
  ]
)

// ──────────────────────────────────────────────
// invoices
// └─ Tagihan per house + periode + billing_item
// ──────────────────────────────────────────────
export const invoices = pgTable(
  'invoices',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    houseId: uuid('house_id')
      .notNull()
      .references(() => houses.id),
    billingItemId: uuid('billing_item_id')
      .notNull()
      .references(() => billingItems.id),
    amount: numeric('amount', { precision: 12, scale: 2 }).notNull(),
    periodMonth: date('period_month').notNull(), // 1st day of month
    dueDate: date('due_date').notNull(),
    paidAt: timestamp('paid_at', { mode: 'string' }),
    status: text('status').notNull().default('PENDING'), // DRAFT | PENDING | PAID | OVERDUE
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
    updatedAt: timestamp('updated_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_invoices_tenant').on(table.tenantId),
    index('idx_invoices_status').on(table.status),
    index('idx_invoices_house_period').on(table.houseId, table.periodMonth),
    uniqueIndex('uk_invoices').on(
      table.tenantId,
      table.houseId,
      table.billingItemId,
      table.periodMonth
    ),
  ]
)

// ──────────────────────────────────────────────
// payments
// └─ Konfirmasi pembayaran (QRIS via Xendit)
// ──────────────────────────────────────────────
export const payments = pgTable(
  'payments',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    invoiceId: uuid('invoice_id')
      .notNull()
      .references(() => invoices.id),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    houseId: uuid('house_id')
      .notNull()
      .references(() => houses.id),
    loginAccountId: uuid('login_account_id')
      .notNull()
      .references(() => loginAccounts.id),
    amount: numeric('amount', { precision: 12, scale: 2 })
      .notNull()
      .$check((col) => col.amount.gte(0)), // CHECK amount >= 0
    externalRef: text('external_ref'), // Xendit charge ID
    paidAt: timestamp('paid_at', { mode: 'string' }).notNull().defaultNow(),
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_payments_tenant').on(table.tenantId),
    index('idx_payments_paid_at').on(table.paidAt),
    index('idx_payments_invoice').on(table.invoiceId),
  ]
)

// ──────────────────────────────────────────────
// audit_log
// └─ Auto-triggered via trigger function (lihat migration)
// ──────────────────────────────────────────────
export const auditLog = pgTable(
  'audit_log',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    tenantId: uuid('tenant_id')
      .notNull()
      .references(() => tenants.id),
    loginAccountId: uuid('login_account_id').references(() => loginAccounts.id),
    action: text('action').notNull(), // "create_invoice", "mark_paid", dst.
    entity: text('entity').notNull(), // "invoices", "houses", dst.
    entityId: uuid('entity_id'),
    detail: jsonb('detail'), // JSONB diff data
    ipAddress: inet('ip_address'),
    createdAt: timestamp('created_at', { mode: 'string' }).defaultNow(),
  },
  (table) => [
    index('idx_audit_tenant').on(table.tenantId),
    index('idx_audit_action').on(table.action),
  ]
)
