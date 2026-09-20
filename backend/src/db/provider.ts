// src/db/provider.ts — Database & Redis provider
//
// Inisialisasi singleton connection pool.
// Di dev/test: langsung pakai Pool (Bun native). Di prod: gunakan pg-bun + ioredis.

import { Pool } from 'pg'
import { drizzle } from 'drizzle-orm/node-postgres'
import * as schema from './schema'
import IORedis from 'ioredis'

let db: ReturnType<typeof drizzle> | null = null
let redis: IORedis | null = null

export function getDB() {
  if (!db) {
    const pool = new Pool({
      connectionString: process.env.DATABASE_URL,
      max: Number(process.env.DB_POOL_SIZE) || 10,
    })
    db = drizzle(pool, { schema })
  }
  return db
}

export function getRedis() {
  if (!redis) {
    redis = new IORedis(process.env.REDIS_URL || 'redis://localhost:6379')
  }
  return redis
}

export type DB = ReturnType<typeof getDB>
export type Redis = ReturnType<typeof getRedis>
