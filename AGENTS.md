# Logikraf AI Studio — Smarthub V3 (Fullstack TypeScript)

Satu sumber kebenaran untuk semua agent AI yang mengerjakaan project Smarthub V3 di VPS ini.

> Stack: **Bun runtime + Hono (backend) + React 19/Vite (frontend) + Drizzle ORM + PostgreSQL + Redis + MinIO**
> Deploy: **GitHub Actions → Docker build → docker compose (prod) → VPS**

## 🏢 Tim (peran → alat)

| Peran | Alat | Tanggung jawab |
|---|---|---|
| Project Manager (orkestrator) | Hermes Agent | Brief, pecah sprint, delegasi, tracking, laporan |
| Analis & Arsitek | llm-council (5 advisor) | Spesifikasi besar → keputusan arsitektur, prioritas, trade-off |
| Developer | delegate_task / sub-agent Hermes | Implementasi fitur per sprint, isolasi git worktree |
| QA / Reviewer | Ponytail config FULL + review | Gate sebelum merge; UI diverifikasi headless Chrome |
| DevOps / Release | GitHub Actions + docker compose | Build di CI, deploy VPS, backup PostgreSQL |

## 🔄 Alur Kerja (standar industri)

1. Brief → 2. Spec (Analis/Arsitek) → 3. Sprint plan (staged)
→ 4. Coding (dev, paralel per worktree) → 5. Review gate (Ponytail FULL)
→ 6. Build di GitHub Actions (JANGAN build di server) → 7. Verifikasi (headless Chrome / curl live)
→ 8. Deploy docker compose → 9. Laporan ke PM (verified output)

Aturan sprint: staged; tiap sprint selesai = laporan terverifikasi. UI harus SIGNIFIKAN.

## 🗺️ Peta Deployment

Jalankan `peta --cek` untuk lihat service + port hidup. Cache di-refresh tiap jam via `/etc/cron.d/peta-refresh`.

- Lokasi kode: `/home/nvnrchmn/Projects/smarthub-v3/`
- Deploy target: `/opt/smarthub-v3/`
- Service systemd: `smarthub-api-v3.service` (backend Bun di port 8082), frontend statis di nginx vhost
- DB: PostgreSQL (`smarthub` database)
- Cache: Redis (session + rate limiter)
- Storage: MinIO (S3 compatible)

## 💻 Stack Final — Fullstack TypeScript

| Layer | Teknologi | Versi | Catatan |
|---|---|---|---|
| Runtime | Bun | latest | Built-in TS transpile, test runner, npm |
| Backend Framework | Hono.js | latest | Multi-tenant pattern (domain middleware) |
| Frontend | React + Vite | 19 + latest | Component-based, canvas-heavy UI ready |
| ORM | Drizzle | latest | TypeScript-first, query builder per tenant |
| Database | PostgreSQL | 16 | RLS mandatory, PII terenkripsi |
| Cache | Redis | 7-alpine | Session + rate limiter login |
| Storage | MinIO | latest | S3 compatible, uploads kaca film |
| Testing | Bun test + Playwright | latest | Unit + UI verification headless |
| Containerization | Docker + Compose | latest | Multi-stage Dockerfile, non-root user |
| CI/CD | GitHub Actions | — | Build → push image → deploy VPS |
| Deploy Method | docker compose prod | — | `docker compose -f compose.prod.yml up -d` |

## 🔐 Security & Conventions

- Env secrets: selalu via environment variables, belum pernah hardcode di file
- Rate limiter login: maks 5 attempt/IP/15menit → block (Redis backend)
- Audit trail: mandatory di semua endpoint write (log ke PostgreSQL + optional webhook)
- Non-root user di container (`USER bun`)
- `.env.production` **tidak di-commit** → gunakan gunakan `docker compose --env-file`
- Backup PostgreSQL: `/usr/local/bin/mysql-backup.sh` (harian 02:30) → `/var/backups/mysql/` + off-site B2
- Health check: semua service wajib implement `/health` endpoint

## 📁 Struktur Project (akan dibuat)

```
smarthub-v3/
├── backend/
│   ├── src/
│   │   ├── routes/
│   │   ├── middleware/ (auth, tenant, rate-limit)
│   │   ├── services/
│   │   └── index.ts
│   ├── Dockerfile
│   └── drizzle.config.ts
├── frontend/
│   ├── src/
│   │   ├── pages/ (5 layer role)
│   │   ├── components/
│   │   └── App.tsx
│   ├── Dockerfile
│   └── vite.config.ts
├── docker-compose.yml (dev)
├── docker-compose.prod.yml (prod)
├── .env.example
├── AGENTS.md
├── PRD.md
└── README.md
```

## 🔑 Layer Role Mapping

| Layer | Role di Sistem |
|---|---|
| 1 | Warga |
| 2 | Sekretaris + Bendahara |
| 3 | Tenant Manager / Owner |
| 4 | Superadmin |
| 5 | API consumer (Xendit, GoWA) |

## 🚫 Larangan

- JANGAN build di server (hanya GitHub Actions)
- JANGAN hardcode credential di file atau chat
- JANGAN gunakan `reload` untuk service — pake `docker compose up/down`
- JANGAN ubah `/etc/nginx/nginx.conf` — pakai vhost di `/www/server/panel/vhost/nginx/`
