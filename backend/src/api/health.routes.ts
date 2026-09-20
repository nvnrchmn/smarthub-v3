// src/api/health.routes.ts — Health check endpoint (tidak butuh auth)
import { Hono } from 'hono'
import { getDB, getRedis } from '../db/provider'

export const healthRoute = new Hono().get('/', async (c) => {
  const db = getDB()
  const redis = getRedis()

  try {
    const dbOk = (await db.execute('SELECT 1')).length > 0
    const redisOk = await redis.ping().then((r) => r === 'PONG')

    return c.json({
      status: 'ok',
      database: dbOk,
      redis: redisOk,
      service: 'smarthub-api',
      timestamp: new Date().toISOString(),
    })
  } catch (e: any) {
    return c.json(
      {
        status: 'error',
        database: false,
        redis: false,
        service: 'smarthub-api',
        error: e.message,
      },
      500
    )
  }
})
