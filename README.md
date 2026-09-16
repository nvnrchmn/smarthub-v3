# Smarthub v3

Platform multi-tenant untuk perumahan: iuran (QRIS dinamis + kas tunai),
sensus warga dengan data pribadi terenkripsi, dan pengingat tunggakan.

## Modul yang Sudah Ada

| Modul | Status |
|---|---|
| Rekonsiliasi QRIS otomatis | ✅ Cron tiap 5 menit |
| Kas tunai + setoran bank | ✅ |
| Sensus warga + enkripsi AES | ✅ |
| Sesi JWT + logout/cabut token | ✅ |
| Lupa sandi via WhatsApp | ✅ |
| Pengingat tunggakan via WhatsApp | ✅ |
| Paginasi & validasi upload | ✅ |
| Redis limiter (fallback memori) | ✅ |

## Roadmap

| Prioritas | Modul |
|---|---|
| Tinggi | Test jalur uang (invoice → bayar → rekonsiliasi) |
| Tinggi | 2FA admin |
| Tinggi | Limiter login per-IP |
| Sedang | Laporan PDF (tunggakan, kas, sensus) |
| Sedang | Health endpoint + rollback otomatis |
| Rendah | Sitemap/PWA/analytics |
| Undur | Persuratan, forum, lapak warga |

## Struktur

- `backend/` — Go (Fiber), clean architecture. `cmd/api` = HTTP API, `internal/` = config, db, domain, usecase, repository, delivery.
- `backend/migrations/` — SQL idempoten (aman dijalankan berulang oleh CI).
- `frontend/` — React + Vite + TypeScript + Tailwind.
- `deploy/` — unit systemd + vhost nginx + compose datastore.

## Infrastruktur (VPS)

- Datastore dijalankan sebagai container (`/opt/smarthub/compose.yml`): PostgreSQL 17, Redis 7, MinIO — semuanya hanya bind ke 127.0.0.1.
- Aplikasi berjalan sebagai service systemd `smarthub-api` (port 8096) di belakang nginx aaPanel.
- Kredensial ada di `/etc/smarthub.env` (600) dan `/opt/smarthub/.env` (600) — **tidak pernah** masuk repo.

## Isolasi tenant

RLS PostgreSQL aktif di semua tabel ber-`tenant_id`. Koneksi aplikasi memakai peran
`smarthub_app` (bukan superuser) dan menyetel `app.tenant_id` per request; baris tenant
lain tidak terlihat walaupun query lupa memakai filter.

## Deploy

Push ke `main` → GitHub Actions membangun frontend + binary Go, menyalinnya ke VPS,
menjalankan migrasi, lalu me-restart `smarthub-api`. Jangan build di server.
