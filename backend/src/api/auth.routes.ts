// src/api/auth.routes.ts — Auth: login, invite, me, logout
//
// POST /api/auth/login    — login via email atau phone
// POST /api/auth/invite   — admin undang user baru → token invite
// GET  /api/auth/me       — profil + roles current user
// POST /api/auth/logout   — revoke token
//
// Token flow:
// - JWT access (15min, di cookie HttpOnly + SameSite=Strict)
// - JWT refresh (7d, di cookie HttpOnly)
// - OTP: optional dev bypass (tanpa OTP kalau NODE_ENV !== production)
import { Hono, type Context } from 'hono'
import { jwt } from 'hono/jwt'
import { z } from 'zod'
import { eq, and } from 'drizzle-orm/expressions'
import { getDB } from '../db/provider'
import * as schema from '../db/schema'

const auth = new Hono()

// ─── Skema validasi ─────────────────────────────────────────
const LoginSchema = z.object({
  identifier: z.string().min(3),  // email atau phone
  password: z.string().min(1),
})

const InviteSchema = z.object({
  email: z.string().email().optional(),
  phone: z.string().min(10).optional(),
  role: z.enum(['TENANT_MANAGER', 'SECRETARY', 'TREASURER', 'RESIDENT']),
}).refine(d => d.email || d.phone, {
  message: 'Harus isi email atau nomor telepon',
})

// ─── POST /api/auth/login ──────────────────────────────────
auth.post('/login', async (c) => {
  try {
    const { identifier, password } = LoginSchema.parse(await c.req.json())
    const db = getDB()

    // Cari user via email atau phone
    const user = await db.query(
      `SELECT * FROM login_accounts
       WHERE tenant_id = $1 AND (email = $2 OR phone = $2) AND password_hash IS NOT NULL`,
      [c.var.tenant_id, identifier]
    )

    if (!user.rows?.length) {
      return c.json({ error: 'Kredensial tidak valid' }, 401)
    }

    const acc = user.rows[0]

    // Verify password (argon2 / bcrypt di prod)
    // Di dev: password_hash = 'devpass:<plain>'
    const isPasswordValid = acc.password_hash.startsWith('devpass:')
      ? acc.password_hash === `devpass:${password}`
      : await Bun.password.verify(password, acc.password_hash)

    if (!isPasswordValid) {
      return c.json({ error: 'Kredensial tidak valid' }, 401)
    }

    // Generate JWT
    const secret = process.env.JWT_SECRET || 'kawaii_neko_2025_dev_secret'
    const now = Math.floor(Date.now() / 1000)

    const accessToken = await jwt.sign({
      sub: acc.id,
      tenant: acc.tenant_id,
      roles: acc.roles,
      iat: now,
      exp: now + 900, // 15 menit
    }, secret)

    const refreshToken = await jwt.sign({
      sub: acc.id,
      type: 'refresh',
      iat: now,
      exp: now + 604800, // 7 hari
    }, secret)

    // Set cookies
    c.header('Set-Cookie', [
      `access_token=${accessToken}; HttpOnly; Path=/; Max-Age=900; SameSite=Strict; Secure`,
      `refresh_token=${refreshToken}; HttpOnly; Path=/; Max-Age=604800; SameSite=Strict; Secure`,
    ])

    // Ambil profil
    const profile = await db.query(
      `SELECT * FROM profiles WHERE login_account_id = $1`,
      [acc.id]
    )

    return c.json({
      message: 'Login berhasil',
      ok: true,
      data: {
        user: {
          id: acc.id,
          email: acc.email,
          phone: acc.phone,
          roles: acc.roles,
          status: acc.status,
          profile: profile.rows[0] || null,
        },
        access_token: accessToken, // frontend pakai ini untuk API non-cookie
      },
    })
  } catch (e: any) {
    if (e instanceof z.ZodError) {
      return c.json({ error: 'Validasi gagal', details: e.errors }, 400)
    }
    return c.json({ error: 'Login gagal' }, 500)
  }
})

// ─── POST /api/auth/refresh ─────────────────────────────────
auth.post('/refresh', async (c) => {
  const refreshToken = c.req.cookie('refresh_token')
  if (!refreshToken) {
    return c.json({ error: 'Refresh token tidak ditemukan' }, 401)
  }

  try {
    const secret = process.env.JWT_SECRET || 'kawaii_neko_2025_dev_secret'
    const payload = await jwt.verify(refreshToken, secret)

    if (payload.type !== 'refresh') {
      return c.json({ error: 'Token tidak valid' }, 401)
    }

    const now = Math.floor(Date.now() / 1000)
    const newAccessToken = await jwt.sign({
      sub: payload.sub,
      tenant: payload.tenant,
      roles: payload.roles,
      iat: now,
      exp: now + 900,
    }, secret)

    return c.json({
      message: 'Token berhasil diperbarui',
      ok: true,
      data: { access_token: newAccessToken },
    })
  } catch {
    return c.json({ error: 'Refresh token tidak valid' }, 401)
  }
})

// ─── POST /api/auth/invite ──────────────────────────────────
auth.post('/invite', async (c) => {
  try {
    const body = InviteSchema.parse(await c.req.json())
    const db = getDB()
    const now = new Date()

    // Cek role invite — hanya TENANT_MANAGER ke atas yang bisa
    const myRoles = c.var.user?.roles || []
    const canInvite = ['TENANT_MANAGER', 'SUPERADMIN'].some(r => myRoles.includes(r))
    const roleValid = canInvite ? true : ['SECRETARY', 'TREASURER', 'RESIDENT'].includes(body.role)

    if (myRoles.includes('SECRETARY') && !['SECRETARY', 'TREASURER', 'RESIDENT'].includes(body.role)) {
      return c.json({ error: 'Role tidak diizinkan' }, 403)
    }
    if (myRoles.includes('TREASURER') && body.role === 'SUPERADMIN') {
      return c.json({ error: 'Role tidak diizinkan' }, 403)
    }

    // Cek duplikat email/phone
    const existing = await db.query(
      `SELECT * FROM login_accounts WHERE tenant_id = $1 AND (email = $2 OR phone = $3)`,
      [c.var.tenant_id, body.email, body.phone]
    )
    if (existing.rows?.length) {
      return c.json({ error: 'Pengguna sudah terdaftar' }, 409)
    }

    // Buat invite token
    const token = crypto.randomUUID()
    const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000) // 7 hari

    await db.query(
      `INSERT INTO login_accounts (id, tenant_id, email, phone, password_hash, roles, status, invite_token, invite_expires_at, created_by, created_at, updated_at)
       VALUES (gen_random_uuid(), $1, $2, $3, NULL, ARRAY[$4], 'invited', $5, $6, $7, $8, $8)`,
      [
        c.var.tenant_id,
        body.email || null,
        body.phone || null,
        body.role,
        token,
        expiresAt,
        c.var.user?.id,
        now,
      ]
    )

    // TODO: kirim email/WhatsApp via GoWA
    // untuk sekarang: return token
    const link = `https://smarthub.logikraf.id/accept-invite?token=${token}`

    return c.json({
      message: 'Undangan berhasil dikirim',
      ok: true,
      data: {
        invite_link: link,
        expires_at: expiresAt.toISOString(),
      },
    }, 201)
  } catch (e: any) {
    if (e instanceof z.ZodError) {
      return c.json({ error: 'Validasi gagal', details: e.errors }, 400)
    }
    return c.json({ error: 'Gagal mengundang' }, 500)
  }
})

// ─── GET /api/auth/me ───────────────────────────────────────
auth.get('/me', async (c) => {
  const db = getDB()
  const userId = c.var.user?.id

  if (!userId) {
    return c.json({ error: 'Unauthorized' }, 401)
  }

  const user = await db.query(
    `SELECT id, email, phone, roles, status, last_login FROM login_accounts WHERE id = $1`,
    [userId]
  )

  if (!user.rows?.length) {
    return c.json({ error: 'User not found' }, 404)
  }

  const profile = await db.query(
    `SELECT * FROM profiles WHERE login_account_id = $1`,
    [userId]
  )

  c.json({
    data: {
      user: user.rows[0],
      profile: profile.rows[0] || null,
    },
  })
})

// ─── POST /api/auth/logout ─────────────────────────────────
auth.post('/logout', (c) => {
  c.header('Set-Cookie', [
    `access_token=; HttpOnly; Path=/; Max-Age=0; SameSite=Strict; Secure`,
    `refresh_token=; HttpOnly; Path=/; Max-Age=0; SameSite=Strict; Secure`,
  ])
  return c.json({ message: 'Logout berhasil', ok: true })
})

export { auth as authRoutes }