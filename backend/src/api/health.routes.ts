// src/api/health.routes.ts — Health check (DB + Redis connection)
import { Hono } from 'hono'

export const healthRoute = new Hono().get('/', async (c) => {
  try {
    const { getDB, getRedis } = await import('../db/provider')
    const db = getDB()
    const redis = getRedis()
    // Ping PostgreSQL
    await db.query('SELECT 1')
    // Ping Redis
    await redis.ping()
    return c.json({
      status: 'ok',
      service: 'smarthub-api',
      database: true,
      redis: true,
    })
  } catch (err: any) {
    return c.json({
      status: 'error',
      service: 'smarthub-api',
      database: false,
      redis: false,
      error: err?.message || 'unknown error',
    }, 500)
  }
})