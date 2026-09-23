# Smarthub — Dokumentasi KYC & Payout Module

> **Status**: Blueprint desain · 23 Sep 2026
> **Tujuan**: Memindahkan fitur KYC, Payout/Settlement, dan manajemen rekening dari Partners Portal ke Smarthub sebagai module terintegrasi.

---

## Daftar Isi

1. [Latar Belakang & Keputusan](#1-latar-belakang--keputusan)
2. [Arsitektur Sistem](#2-arsitektur-sistem)
3. [Modul KYC (Verify on Behalf)](#3-modul-kyc-verify-on-behalf)
4. [Modul Payout/Settlement](#4-modul-payoutsettlement)
5. [Modul Rekening Bank](#5-modul-rekening-bank)
6. [Database Schema](#6-database-schema)
7. [API Endpoints](#7-api-endpoints)
8. [Konfigurasi Environment](#8-konfigurasi-environment)
9. [Flow Diagram](#9-flow-diagram)
10. [Keamanan & Compliance](#10-keamanan--compliance)
11. [Roadmap Implementasi](#11-roadmap-implementasi)

---

## 1. Latar Belakang & Keputusan

### 1.1 Keputusan Arsitektur

| Komponen | Smarthub | Partners Portal |
|---|---|---|
| **Target User** | Tenant Smarthub (ketua RT/RW) | Mitra bisnis Logikraf (toko, MG, dll) |
| **KYC & Payout** | ✅ Di Smarthub | ✅ Tetap di partners portal (untuk mitra non-Smarthub) |
| **Branding** | Smarthub | Logikraf |
| **Database** | DB Smarthub sendiri | DB Partners Portal |

### 1.2 Alasan Pemisahan

- Tenant Smarthub adalah **end-user SaaS**, bukan mitra bisnis
- Data tenant sudah ada di Smarthub (nama, email, HP, alamat)
- UX lebih sederhana — semua fitur di satu tempat
- Smarthub menjadi **self-contained** product
- Partners portal tetap melayani mitra bisnis non-Smarthub

---

## 2. Arsitektur Sistem

### 2.1 Diagram Alur KYC

```
┌─────────────────────────────────────────────────────────────────┐
│                        SMARTHUB FRONTEND                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   Dashboard  │  │   Verifikasi │  │   Payout/Settlement  │  │
│  │              │  │     KYC      │  │                      │  │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘  │
└─────────┼─────────────────┼─────────────────────┼───────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                        SMARTHUB BACKEND                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Auth & RBAC │  │  KYC Module  │  │   Payout Module      │  │
│  │              │  │              │  │                      │  │
│  └──────────────┘  └──────┬───────┘  └──────────┬───────────┘  │
└───────────────────────────┼─────────────────────┼───────────────┘
                            │                     │
                            ▼                     ▼
                  ┌─────────────────┐   ┌─────────────────┐
                  │  XENDIT API     │   │   BANK ACCOUNT  │
                  │  (v3/accounts)  │   │   (Database)    │
                  └─────────────────┘   └─────────────────┘
```

### 2.2 Diagram Alur Payout

```
┌─────────────────────────────────────────────────────────────────┐
│                        SMARTHUB FRONTEND                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Tarik Saldo / Payout                   │   │
│  │  - Pilih rekening tujuan                                  │   │
│  │  - Input jumlah                                           │   │
│  │  - Konfirmasi                                             │   │
│  └──────────────────────────┬───────────────────────────────┘   │
└─────────────────────────────┼───────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        SMARTHUB BACKEND                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Payout Module                          │   │
│  │  - Validasi saldo                                         │   │
│  │  - Validasi rekening                                      │   │
│  │  - Call Xendit API                                        │   │
│  │  - Simpan riwayat                                         │   │
│  └──────────────────────────┬───────────────────────────────┘   │
└─────────────────────────────┼───────────────────────────────────┘
                              │
                              ▼
                  ┌─────────────────────────┐
                  │     XENDIT PAYOUT API   │
                  │  POST /v3/payouts       │
                  │  Header: for-user-id    │
                  └─────────────────────────┘
```

---

## 3. Modul KYC (Verify on Behalf)

### 3.1 Konsep

Smarthub bertindak sebagai **Master Account** yang mengumpulkan data KYC tenant dan mengirimkannya ke Xendit atas nama tenant (Verify on Behalf).

### 3.2 Persyaratan Xendit

- ✅ Xendit XenPlatform sudah aktif
- ✅ Verify on Behalf **otomatis aktif** (konfirmasi Xendit via email 23 Sep 2026)
- ✅ Tenant perorangan (INDIVIDUAL) diterima
- ⚠️ **Consent wajib** — tenant harus menyetujui sebelum data dikirim
- ⚠️ **Service Agreement** — wajib lampirkan perjanjian yang ditandatangani tenant

### 3.3 Flow KYC

```
1. Tenant login ke Smarthub
        ↓
2. Buka menu "Verifikasi"
        ↓
3. Isi form data diri:
   - Nama lengkap
   - No. KTP (NIK)
   - Tanggal lahir
   - Jenis kelamin
   - Kewarganegaraan
   - Alamat (sesuai KTP)
   - Email
   - No. HP
   - Tipe entitas (INDIVIDUAL/SOLE_PROPRIETORSHIP/dll)
        ↓
4. Upload dokumen:
   - KTP depan (JPG/PNG/PDF)
   - KTP belakang (JPG/PNG/PDF)
   - Selfie (JPG/PNG) — optional tapi disarankan
   - Perjanjian layanan (PDF) — wajib
        ↓
5. Centang consent checkbox:
   "Saya menyetujui bahwa data pribadi (KTP, selfie, NPWP) 
   dan dokumen KYC saya dikirim ke Xendit oleh Smarthub 
   sebagai platform provider untuk proses verifikasi, 
   sesuai ketentuan XenPlatform."
        ↓
6. Submit → POST /api/v1/me/kyc/initiate
        ↓
7. Server buat MANAGED Sub-Account via Xendit API
   POST https://api.xendit.co/v3/accounts
   {
     "name": "<nama tenant>",
     "email": "<email tenant>",
     "identity": {
       "country_of_incorporation": "ID",
       "entity_type": "INDIVIDUAL"
     },
     "configuration": {
       "webhooks": {
         "recipient": "MASTER_ACCOUNT"
       }
     }
   }
        ↓
8. Simpan sub_account_id ke DB Smarthub
        ↓
9. Xendit kirim undangan email ke tenant
        ↓
10. Tenant isi KYC di dashboard Xendit
    (atau via Verify on Behalf jika diizinkan)
        ↓
11. Xendit verifikasi (3-5 hari kerja)
        ↓
12. Webhook account.verification → Smarthub
    Status: VERIFICATION_IN_PROGRESS / PASSED / FAILED
        ↓
13. Sub-account LIVE → bisa menerima pembayaran
```

### 3.4 Status KYC

| Status | Deskripsi | Aksi |
|---|---|---|
| `INVITED` | Sub-account dibuat, menunggu tenant isi KYC | Tenant cek email |
| `AWAITING_DOCS` | Xendit menunggu dokumen | Tenant upload dokumen |
| `PENDING_VERIFICATION` | Dikirim ke Xendit, menunggu review | Tunggu 3-5 hari |
| `AWAITING_RESUBMISSION` | Ditolak, perlu perbaikan | Tenant upload ulang |
| `LIVE` | KYC approved | Bisa terima pembayaran |
| `DECLINED` | Ditolak permanen | Hubungi Xendit |
| `SUSPENDED` | Ditangguhkan | Hubungi Xendit |

---

## 4. Modul Payout/Settlement

### 4.1 Konsep

Tenant dapat menarik dana dari saldo sub-account ke rekening bank yang terdaftar. Smarthub memanggil Xendit Payout API dengan header `for-user-id`.

### 4.2 Persyaratan

- Sub-account harus **LIVE** (KYC approved)
- **Money-out** harus diaktifkan untuk sub-account (via Dashboard Xendit atau koordinasi tim Xendit)
- Rekening tujuan harus terdaftar dan valid

### 4.3 Flow Payout

```
1. Tenant login ke Smarthub
        ↓
2. Buka menu "Pencairan" atau "Tarik Saldo"
        ↓
3. Lihat saldo tersedia (GET /api/v1/me/balance)
        ↓
4. Pilih rekening tujuan (atau tambah baru)
        ↓
5. Input jumlah penarikan (Rp)
        ↓
6. Konfirmasi
        ↓
7. Submit → POST /api/v1/me/payouts
   {
     "bank_account_id": <id>,
     "amount": <jumlah dalam rupiah>
   }
        ↓
8. Server call Xendit Payout API
   POST https://api.xendit.co/v3/payouts
   Header: for-user-id: <sub_account_id>
   {
     "amount": <jumlah>,
     "bank_account_id": <xendit_bank_account_id>,
     "reference_id": "<unique_id>",
     "description": "Payout Smarthub"
   }
        ↓
9. Simpan ke tabel payouts (status: PENDING)
        ↓
10. Xendit proses → dana masuk rekening tenant
        ↓
11. Webhook payout.* → Smarthub
    Status: PENDING → PROCESSING → COMPLETED/FAILED
        ↓
12. Update status payout di DB
        ↓
13. Notifikasi WA ke tenant (opsional)
```

### 4.4 Status Payout

| Status | Deskripsi |
|---|---|
| `PENDING` | Diajukan, menunggu proses Xendit |
| `PROCESSING` | Sedang diproses Xendit |
| `COMPLETED` | Dana berhasil masuk rekening |
| `FAILED` | Gagal (saldo insufficient, rekening invalid, dll) |
| `CANCELLED` | Dibatalkan |

---

## 5. Modul Rekening Bank

### 5.1 Konfigurasi

Tenant dapat mendaftarkan satu atau lebih rekening bank sebagai tujuan pencairan.

### 5.2 Flow Rekening

```
1. Tenant → Menu "Rekening" / "Tarik Saldo"
        ↓
2. Tambah rekening baru:
   - Nama bank
   - No. rekening
   - Nama pemilik rekening (harus sesuai KTP)
   - Set sebagai utama (opsional)
        ↓
3. Simpan ke DB Smarthub
        ↓
4. Saat payout, pilih rekening tujuan
```

---

## 6. Database Schema

### 6.1 Tabel `smarthub_tenants` (tabel yang sudah ada atau baru)

```sql
-- Kolom yang perlu ditambahkan ke tabel tenants yang sudah ada
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS xendit_sub_account_id VARCHAR(120);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS xendit_kyc_status VARCHAR(40) DEFAULT 'NOT_REGISTERED';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS xendit_entity_type VARCHAR(40) DEFAULT 'INDIVIDUAL';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS xendit_kyc_submitted_at TIMESTAMP;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS xendit_kyc_verified_at TIMESTAMP;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS xendit_money_out_enabled BOOLEAN DEFAULT FALSE;
```

### 6.2 Tabel `smarthub_bank_accounts`

```sql
CREATE TABLE IF NOT EXISTS smarthub_bank_accounts (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id),
  bank_code VARCHAR(20) NOT NULL,          -- BCA, MANDIRI, BRI, dll
  bank_name VARCHAR(100) NOT NULL,
  account_number VARCHAR(50) NOT NULL,
  account_holder VARCHAR(150) NOT NULL,
  is_default BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

### 6.3 Tabel `smarthub_payouts`

```sql
CREATE TABLE IF NOT EXISTS smarthub_payouts (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id),
  xendit_payout_id VARCHAR(120),
  bank_account_id BIGINT REFERENCES smarthub_bank_accounts(id),
  amount BIGINT NOT NULL,                   -- dalam rupiah
  status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
  reference_id VARCHAR(60) UNIQUE,
  failure_reason TEXT,
  completed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

### 6.4 Tabel `smarthub_kyc_submissions`

```sql
CREATE TABLE IF NOT EXISTS smarthub_kyc_submissions (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id),
  entity_type VARCHAR(40),
  legal_name VARCHAR(150),
  email VARCHAR(190),
  ktp_number VARCHAR(30),
  dob DATE,
  gender VARCHAR(10),
  nationality VARCHAR(10),
  address JSONB,
  status VARCHAR(40),
  failure_reasons JSONB,
  submitted_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## 7. API Endpoints

### 7.1 KYC Endpoints

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/me/kyc` | Status KYC tenant |
| `POST` | `/api/v1/me/kyc/initiate` | Submit data awal + buat sub-account |
| `POST` | `/api/v1/me/kyc/documents` | Upload dokumen KYC (multipart) |
| `POST` | `/api/v1/me/kyc/submit` | Submit data + dokumen ke Xendit |

### 7.2 Payout Endpoints

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/me/balance` | Saldo sub-account |
| `GET` | `/api/v1/me/payouts` | Riwayat payout |
| `POST` | `/api/v1/me/payouts` | Ajukan payout baru |
| `GET` | `/api/v1/me/payouts/:id` | Detail payout |

### 7.3 Bank Account Endpoints

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/me/bank-accounts` | Daftar rekening tenant |
| `POST` | `/api/v1/me/bank-accounts` | Tambah rekening baru |
| `DELETE` | `/api/v1/me/bank-accounts/:id` | Hapus rekening |
| `PATCH` | `/api/v1/me/bank-accounts/:id` | Update rekening |

### 7.4 Webhook Endpoints (diterima dari Xendit)

| Method | Endpoint | Event | Deskripsi |
|---|---|---|---|
| `POST` | `/api/v1/webhooks/xendit/account.verification` | KYC status | Update status KYC tenant |
| `POST` | `/api/v1/webhooks/xendit/payout` | Payout status | Update status payout |

---

## 8. Konfigurasi Environment

```env
# Xendit API
XENDIT_API_KEY=<secret_key_master_account>
XENDIT_CALLBACK_TOKEN=<callback_token>

# KYC Portal Mode
KYC_PORTAL_MODE=portal

# JWT Secret (untuk auth tenant Smarthub)
JWT_SECRET=<jwt_secret>

# Database (Smärthub)
DATABASE_URL=<smarthub_db_connection_string>
```

---

## 9. Flow Diagram

### 9.1 Alur Lengkap Onboarding Tenant Smarthub

```
┌─────────────────────────────────────────────────────────────────┐
│                        SMARTHUB TENANT                          │
│                                                                 │
│  1. Login → Dashboard                                           │
│  2. Status: "Verifikasi diperlukan untuk menerima pembayaran"   │
│  3. Klik "Lengkapi Verifikasi"                                 │
│  4. Isi form data diri + upload dokumen                        │
│  5. Centang consent                                            │
│  6. Submit                                                     │
│  7. Status: "Menunggu verifikasi Xendit (3-5 hari kerja)"      │
│  8. Notifikasi WA/email saat status berubah                     │
│  9. Status: "Verified — Sub-account aktif"                     │
│ 10. Dashboard tampil saldo + menu pencairan                    │
└─────────────────────────────────────────────────────────────────┘
```

### 9.2 Alur Pencairan Dana

```
┌─────────────────────────────────────────────────────────────────┐
│                        SMARTHUB TENANT                          │
│                                                                 │
│  1. Login → Dashboard                                           │
│  2. Lihat saldo tersedia                                        │
│  3. Klik "Tarik Saldo"                                          │
│  4. Pilih rekening tujuan                                       │
│  5. Input jumlah                                                │
│  6. Konfirmasi                                                  │
│  7. Status: "Penarikan diproses"                                │
│  8. Notifikasi WA/email saat dana masuk                        │
│  9. Riwayat payout tampil di dashboard                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## 10. Keamanan & Compliance

### 10.1 Konsent

- ✅ Consent checkbox **wajib** sebelum submit KYC
- ✅ Service agreement (PDF) **wajib** dilampirkan
- ✅ Konsent disimpan di DB (timestamp + IP + user agent)

### 10.2 Data Sensitif

- ❌ **Jangan simpan KTP/selfie/NPWP di DB** — langsung forward ke Xendit
- ✅ File upload diteruskan ke Xendit via API, tidak disimpan permanen
- ✅ Database hanya simpan `file_id` dari Xendit (bukan file)

### 10.3 Webhook Security

- ✅ Verifikasi `X-CALLBACK-TOKEN` untuk memastikan webhook berasal dari Xendit
- ✅ Internal key untuk endpoint yang dipanggil internal

### 10.4 Audit Trail

- ✅ Semua request ke Xendit di-log (request_id, timestamp, status)
- ✅ Riwayat payout tersimpan minimal 1 tahun
- ✅ Audit log untuk semua perubahan status KYC

---

## 11. Roadmap Implementasi

### Fase 1 — Foundation (1-2 minggu)
- [ ] Database migration (tabel baru)
- [ ] Xendit client service (v3/accounts, /files, /account_verification, /v3/payouts)
- [ ] Webhook handler (account.verification, payout)
- [ ] Auth & middleware

### Fase 2 — KYC Module (2-3 minggu)
- [ ] Form KYC (data diri + upload dokumen)
- [ ] Consent checkbox
- [ ] Submit ke Xendit
- [ ] Status tracking + notifikasi

### Fase 3 — Payout Module (2-3 minggu)
- [ ] Saldo display
- [ ] Form pencairan
- [ ] Rekening bank management
- [ ] Riwayat payout

### Fase 4 — Dashboard & UX (1-2 minggu)
- [ ] Integrasi ke dashboard tenant
- [ ] Notifikasi WA/email
- [ ] Testing end-to-end

### Fase 5 — Go-Live (1 minggu)
- [ ] Pilot dengan 1-2 tenant
- [ ] Monitoring & bug fixing
- [ ] Dokumentasi final

---

## Referensi

- [Xendit XenPlatform Documentation](https://docs.xendit.co/docs/xenplatform)
- [Create Sub-Account (v3)](https://docs.xendit.co/apidocs/create-account-v3)
- [Submit Account Verification](https://docs.xendit.co/apidocs/submit-account-verification)
- [Upload KYC Document](https://docs.xendit.co/apidocs/upload-file)
- [Create Payout](https://docs.xendit.co/apidocs/create-payout)
- [Xendit Webhook Events](https://docs.xendit.co/docs/webhooks)

---

*Dokumen ini bersifat sebagai blueprint. Implementasi aktual mungkin mengalami penyesuaian berdasarkan hasil testing dan konfirmasi dari Xendit.*
