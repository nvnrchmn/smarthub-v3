// src/db/migrate.ts — Migration runner
//
// Cara pakai:
//   bun run db:migrate  (development)
// atau:
//   docker exec -i smarthub_v3_db psql -U smarthub -d smarthub_dev < src/db/migrations/20250920_create_initial.sql

import { getDB } from './provider'
import { readFileSync, readdirSync } from 'fs'
import { join } from 'path'

async function migrate() {
  const db = getDB()
  const migrateDir = join(import.meta.dir, 'migrations')

  // Urutkan file migration
  const files = readdirSync(migrateDir)
    .filter(f => f.endsWith('.sql'))
    .sort()

  console.log(`[migrate] ${files.length} migration file(s) found.`)

  for (const file of files) {
    // Cek apakah migration sudah dijalankan
    const applied = await db.query(
      `SELECT 1 FROM schema_migrations WHERE filename = $1`,
      [file]
    )

    if (applied.rows.length > 0) {
      console.log(`[migrate] ✓ ${file} — already applied, skip.`)
      continue
    }

    const sql = readFileSync(join(migrateDir, file), 'utf-8')
    console.log(`[migrate] → Running ${file}...`)

    try {
      await db.query(sql)

      // Catat migration selesai
      await db.query(
        `INSERT INTO schema_migrations (filename, applied_at) VALUES ($1, now())`,
        [file]
      )

      console.log(`[migrate] ✓ ${file} — applied.`)
    } catch (e: any) {
      console.error(`[migrate] ✗ ${file} — FAILED: ${e.message}`)

      if (e.code === '23505') {
        console.error('[migrate] Duplicate key — migration mungkin sudah diterapkan.')
      }

      throw e
    }
  }

  console.log('[migrate] Semua migration selesai.')
}

// Jalankan migration otomatis di dev mode
if (import.meta.main) {
  migrate().catch(e => {
    console.error(e)
    process.exit(1)
  })
}