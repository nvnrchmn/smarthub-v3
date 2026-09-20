// src/index.ts — Hono app entry point (Smarthub V3 backend)
//
// Stack: Bun runtime + Hono + Drizzle ORM → PostgreSQL + Redis
//
// Middleware order:
//   1. CORS / Logger — global
//   2. JWT auth        — verify access token, populate c.var.user
//   3. TenantContext   — SET LOCAL 'app.current_tenant' untuk RLS
//   4. ScopeGuard      — role-based route protection via prefix
//
import { Hono } from 'hono'
import { jwt } from 'hono/jwt'
import { cors } from 'hono/cors'
import { logger } from 'hono/logger'

// ─── Imports ───────────────────────────────────────────
import { authRoutes } from './api/auth.routes'
import { healthRoute } from './api/health.routes'
import { houseRoutes } from './api/houses.routes'
import { billingRoutes } from './api/billing.routes'
import { invoiceRoutes } from './api/invoices.routes'
import { paymentRoutes } from './api/payments.routes'
import { getDB } from './db/provider'

const app = new Hono()

// ─── Middleware global ─────────────────────────────────
app.use('*', logger())

// CORS — hanya izinkan domain resmi, tidak wildcard
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
    return c.json({ error: 'Unauthenticated', code: 'NO_TOKEN' }, 401)
  }

  const token = authHeader.slice(7)
  try {
    const secret = process.env.JWT_SECRET || 'kawaii_neko_2025_dev_secret'
    const payload = await jwt.verify(token, secret)

    const userId = String(payload.sub)
    const tenantId = String(payload.tenant || payload.tid)

    // Ambil roles + status dari DB (authoritative)
    const db = getDB()
    const userCheck = await db.query(
      `SELECT id, email, phone, roles, status FROM login_accounts WHERE id = $1 AND tenant_id = $2`,
      [userId, tenantId]
    )

    if (!userCheck.rows?.length) {
      return c.json({ error: 'Pengguna tidak valid' }, 401)
    }

    const user = userCheck.rows[0]
    if (user.status === 'suspended') {
      return c.json({ error: 'Akun Anda ditangguhkan' }, 403)
    }

    // Populate context
    c.set('user', {
      id: user.id,
      email: user.email,
      phone: user.phone,
      tenantId: tenantId,
      roles: user.roles,
    })
    c.set('tid', tenantId)

    // 2. Set PostgreSQL tenant context untuk RLS
    await db.query(`SET LOCAL "app.current_tenant" = $1`, [tenantId])

    await next()
  } catch {
    return c.json({ error: 'Invalid or expired token', code: 'TOKEN_EXPIRED' }, 401)
  }
})

// 3. Scope guard — prefix-based role protection
//    /houses/*      → TENANT_MANAGER, SECRETARY, TREASURER, RESIDENT, SUPERADMIN
//    /billing/*     → TENANT_MANAGER, TREASURER, SUPERADMIN
//    /invoices/*    → TENANT_MANAGER, TREASURER, SUPERADMIN (read), ADMIN lain read-only
//    /payments/*    → TENANT_MANAGER, TREASURER, SUPERADMIN
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
      return c.json({ error: 'Forbidden — scope tidak diizinkan', code: 'SCOPE_DENIED' }, 403)
    }
  }

  await next()
})

// 4. Route registration
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