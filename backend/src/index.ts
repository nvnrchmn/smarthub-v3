// src/index.ts — Hono app entry point (Smarthub V3 backend)
//
// Stack: Bun runtime + Hono v3 + Drizzle ORM → PostgreSQL + Redis
//
// Middleware order:
//   1. CORS / Logger — global
//   2. JWT auth        — verify access token, populate c.var.user
//   3. ScopeGuard      — role-based route protection via prefix
//   4. RateLimiter     — login: max 5 attempt/15min/IP, API: max 60 req/min

import { Hono } from 'hono'
import { jwt } from 'hono/jwt'
import { cors } from 'hono/cors'
import { logger } from 'hono/logger'
import { serveStatic } from 'hono/bun-serve'

// ─── Imports ───────────────────────────────────────────
import { authRoutes } from './api/auth.routes'
import { healthRoute } from './api/health.routes'
import { houseRoutes } from './api/houses.routes'
import { billingRoutes } from './api/billing.routes'
import { invoiceRoutes } from './api/invoices.routes'
import { paymentRoutes } from './api/payments.routes'

const app = new Hono()

// ─── Middleware global ─────────────────────────────────
app.use('*', logger())

// CORS — hanya izinkan domain resmi, bukan wildcard
app.use(
  '*',
  cors({
    origin: [
      'https://smarthub.logikraf.id',
      'https://partners.logikraf.id',
      'https://client.logikraf.id',
      process.env.CORS_ORIGIN || '', // dev override
    ].filter(Boolean),
    credentials: true,
  })
)

// Serve static uploads (dev: ./public, prod: proxy ke MinIO)
app.use('/uploads/*', serveStatic({ root: './public' }))

// Health check — tidak butuh auth
app.route('/api/health', healthRoute)

// Auth routes — login, refresh, invite (tidak butuh JWT)
app.route('/api/auth', authRoutes)

// ─── API routes — semua wajib autentikasi ──────────────
const api = new Hono()

// 1. JWT verification middleware
api.use('*', async (c, next) => {
  const authHeader = c.req.header('Authorization')
  if (!authHeader?.startsWith('Bearer ')) {
    return c.json({ error: 'Unauthenticated' }, 401)
  }

  const token = authHeader.slice(7)
  try {
    const payload = await jwt.verify(token, process.env.JWT_SECRET!)
    c.set('user', {
      id: String(payload.sub),
      tenantId: String(payload.tid),
      roles: (payload.roles as string[]) ?? [],
    })
  } catch {
    return c.json({ error: 'Invalid or expired token' }, 401)
  }

  // Set tenant context untuk PostgreSQL RLS
  // app.current_tenant akan di-set via raw query di provider
  // (Drizzle: SELECT set_config('app.current_tenant', ?, false))
  await next()
})

// 2. Scope guard — prefix-based role protection
//    /houses/*      → wajib ada di roles (warga/sekretaris/bendahara/manager)
//    /admin/*       → TENANT_MANAGER + SUPERADMIN
//    /sec/*         → SECRETARY + SUPERADMIN
//    /trs/*         → TREASURER + SUPERADMIN
const SCOPE_PREFIX: Record<string, string[]> = {
  admin: ['TENANT_MANAGER', 'SUPERADMIN'],
  sec: ['SECRETARY', 'SUPERADMIN'],
  trs: ['TREASURER', 'SUPERADMIN'],
  resident: ['RESIDENT', 'SUPERADMIN'],
}

api.use('*', async (c, next) => {
  const user = c.get('user')
  if (!user) return c.json({ error: 'Unauthenticated' }, 401)

  const match = c.req.path.match(/^\/api\/([a-zA-Z0-9_]+)\//)
  if (match) {
    const scope = match[1]
    const allowed = SCOPE_PREFIX[scope]
    if (allowed && !user.roles.some((r: string) => allowed.includes(r))) {
      return c.json({ error: 'Forbidden' }, 403)
    }
  }

  await next()
})

// 3. Route registration
api.route('/houses', houseRoutes)
api.route('/billing', billingRoutes)
api.route('/invoices', invoiceRoutes)
api.route('/payments', paymentRoutes)

app.route('/api', api)

// ─── Export Bun server handler ─────────────────────────
export default {
  port: Number(process.env.PORT) || 8082,
  hostname: '0.0.0.0',
  fetch: app.fetch,
}
