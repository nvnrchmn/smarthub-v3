// src/api/auth.routes.ts — Auth routes: login, refresh, invite, logout
import { Hono } from 'hono'
import { jwt } from 'hono/jwt'
// import { getDB } from '../db/provider'
// import * as schema from '../db/schema'
// import { eq, and } from 'drizzle-orm'
// import * as argon2 from 'argon2'

export const authRoutes = new Hono()

// POST /api/auth/login
// Input: { email|string, phone|string, password: string }
// Output: { accessToken, refreshToken }
authRoutes.post('/login', async (c) => {
  const { email, phone, password } = await c.req.json()
  return c.json({ error: 'login handler not yet implemented' }, 501)
})

// POST /api/auth/refresh
// Input: { refreshToken }
// Output: { accessToken }
authRoutes.post('/refresh', async (c) => {
  return c.json({ error: 'refresh handler not yet implemented' }, 501)
})

// POST /api/auth/invite
// Scope: TENANT_MANAGER
// Input: { email|string, phone|string, role: string }
// Output: { token: string } — invite link token
authRoutes.post('/invite', async (c) => {
  return c.json({ error: 'invite handler not yet implemented' }, 501)
})

// POST /api/auth/logout
// Revokes current token + clears cookie
authRoutes.post('/logout', async (c) => {
  return c.json({ error: 'logout handler not yet implemented' }, 501)
})
