// src/db/provider.ts — DB & Redis provider (dev mode, pg native
// Diprod gunakan drizzle-orm + connection pool via Bun PG driver)
import { Pool } from 'pg'
import IORedis from 'ioredis'

let pool: Pool | null = null
let redis: IORedis | null = null

export function getDB(): Pool {
  if (!pool) {
    pool = new Pool({
      connectionString: process.env.DATABASE_URL,
      max: Number(process.env.DB_POOL_SIZE) || 10,
    })
  }
  return pool
}

export function getRedis(): IORedis {
  if (!redis) {
    redis = new IORedis(process.env.REDIS_URL || 'redis://localhost:6379')
  }
  return redis
}

export type DB = Pool