# Smarthub V3 — Fullstack TypeScript Starter

Multi-tenant SaaS untuk manajemen perumahan (RT/RW).

## 🏗️ Stack
- **Backend**: Bun + Hono.js + Drizzle ORM → PostgreSQL
- **Frontend**: React 19 + Vite + TailwindCSS
- **Cache**: Redis (session + rate limiter)
- **Storage**: MinIO (S3 compatible)
- **Deploy**: Docker Compose → GitHub Actions → VPS

## 🚀 Quick Start (dev)

```bash
# 1. Clone
git clone https://github.com/nvnrchmn/smarthub-v3.git

# 2. Start infrastructure
docker compose up -d db redis minio

# 3. Backend
cd backend && bun install && bun run dev

# 4. Frontend
cd ../frontend && npm install && npm run dev
```

## 🐋 Deploy

```bash
docker compose -f compose.prod.yml up -d
```

## 🔐 Security
- `.env.production` is NOT committed. Use `docker compose --env-file`.
- All endpoints require JWT with scope-based auth.
- PostgreSQL RLS isolates tenant data.
- Login rate-limited (5 attempt / 15 min per IP).

## 📖 Docs
- `AGENTS.md` — coding standards & deployment rules
- `PRD.md` — full feature spec & sprint plan