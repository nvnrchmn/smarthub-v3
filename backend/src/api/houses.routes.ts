// src/api/houses.routes.ts — CRUD manajemen rumah (houses)
//
// GET    /api/houses        — list + filter + pagination
// POST   /api/houses        — buat rumah baru
// GET    /api/houses/:id    — detail + occupants
// PUT    /api/houses/:id    — update
// DELETE /api/houses/:id    — soft delete (set VACANT)
//
// Query params:
//   ?page=1&limit=50&status=OCCUPIED&search=budi
//   (nama kepala / nomor rumat / blok di-search)
import { Hono } from 'hono'
import { z } from 'zod'
import { getDB } from '../db/provider'
import * as schema from '../db/schema'

const houses = new Hono()

// Skema validasi (zod)
const HouseSchema = z.object({
  blok: z.string().min(1),
  nomor_rumah: z.string().min(1),
  daya_listrik: z.number().int().min(100).max(10000),
  luas_bangunan: z.number().min(1).max(5000),
  luas_tanah: z.number().min(1).max(10000),
  address: z.string().min(3),
})

const HouseUpdateSchema = HouseSchema.partial().extend({
  status: z.enum(['OCCUPIED', 'VACANT']).optional(),
})

// ─── GET /api/houses — list + filter + pagination ─────────────
houses.get('/', async (c) => {
  const db = getDB()
  const tenant_id = c.var.tenant_id

  // Parse query params
  const page = Math.max(1, Number(c.req.query('page') || 1))
  const limit = Math.min(100, Math.max(1, Number(c.req.query('limit') || 50)))
  const offset = (page - 1) * limit

  const status = c.req.query('status')
  const search = c.req.query('search')

  // Build WHERE conditions
  const conditions: string[] = []
  const values: any[] = [tenant_id]
  let i = 2

  if (status) {
    conditions.push(`status = $${i++}`)
    values.push(status)
  }
  if (search) {
    conditions.push(`(nama_kepala ILIKE $${i} OR blok ILIKE $${i+1} OR nomor_rumah ILIKE $${i+1})`)
    values.push(`%${search}%`, `%${search}%`)
    i += 2
  }
  conditions.push('deleted_at IS NULL')

  const where = conditions.length > 1 ? `WHERE ${conditions.join(' AND ')}` : 'WHERE deleted_at IS NULL'

  // Get total count
  const countResult = await db.query(`
    SELECT COUNT(*) FROM profiles p
    JOIN houses h ON p.house_id = h.id AND p.is_primary_contact = true
    ${where.replace('deleted_at IS NULL', 'h.deleted_at IS NULL').replace('deleted_at', 'h.deleted_at').replace('status = ', 'h.status = ').replace('nama_kepala', 'p.nama_kepala').replace('blok', 'h.blok').replace('nomor_rumah', 'h.nomor_rumah')}
  `, values.slice(0, -1))

  // Wait — better approach: simple direct query
  const simpleWhere = conditions.filter(c => !c.includes('nama_kepala')).join(' AND ')
  const total = await db.query(
    `SELECT COUNT(*) FROM houses WHERE tenant_id = $1 AND (${simpleWhere})`,
    [tenant_id]
  )

  // Get houses with primary contact
  const result = await db.query(`
    SELECT h.*, p.full_name as nama_kepala, p.phone as no_hp
    FROM houses h
    LEFT JOIN profiles p ON p.house_id = h.id AND p.is_primary_contact = true
    WHERE h.tenant_id = $1 AND h.deleted_at IS ${status ? 'NULL AND h.status = $2' : 'NULL'}
    ORDER BY h.blok, h.nomor_rumah
    LIMIT $${status ? 3 : 2} OFFSET $${status ? 4 : 3}
  `, status ? [tenant_id, status, limit, offset] : [tenant_id, limit, offset])

  return c.json({
    data: result.rows || [],
    meta: {
      page,
      limit,
      total: Number(total.rows[0]?.count || 0),
      total_pages: Math.ceil(Number(total.rows[0]?.count || 0) / limit),
    },
    message: 'Daftar rumah berhasil diambil',
  })
})

// ─── POST /api/houses — create ───────────────────────────────
houses.post('/', async (c) => {
  try {
    const body = HouseSchema.parse(await c.req.json())
    const db = getDB()
    const tenant_id = c.var.tenant_id
    const now = new Date()

    const result = await db.query(
      `INSERT INTO houses
        (id, tenant_id, blok, nomor_rumah, daya_listrik, luas_bangunan, luas_tanah, nama_kepala, address, status, created_at, updated_at)
       VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NULL, $7, 'VACANT', $8, $8)
       RETURNING *`,
      [tenant_id, body.blok, body.nomor_rumah, body.daya_listrik, body.luas_bangunan, body.luas_tanah, body.address, now]
    )

    return c.json({
      data: result.rows[0],
      message: 'Rumah berhasil ditambahkan',
      ok: true,
    }, 201)
  } catch (e: any) {
    if (e instanceof z.ZodError) {
      return c.json({ error: 'Validasi gagal', details: e.errors }, 400)
    }
    return c.json({ error: 'Gagal menambahkan rumah' }, 500)
  }
})

// ─── GET /api/houses/:id — detail + occupants ────────────────
houses.get('/:id', async (c) => {
  const db = getDB()
  const tenant_id = c.var.tenant_id
  const id = c.req.param('id')

  const detail = await db.query(
    `SELECT h.*, p.* FROM houses h
     LEFT JOIN (
       SELECT * FROM profiles WHERE is_primary_contact = true
     ) p ON p.house_id = h.id
     WHERE h.id = $1 AND h.tenant_id = $2 AND h.deleted_at IS NULL`,
    [id, tenant_id]
  )

  if (!detail.rows.length) {
    return c.json({ error: 'Rumah tidak ditemukan' }, 404)
  }

  // Ambil semua occupants
  const occupants = await db.query(
    `SELECT p.*, ho.relationship, ho.moved_in, ho.moved_out
     FROM house_occupants ho
     JOIN profiles p ON p.login_account_id = ho.profile_id
     WHERE ho.house_id = $1`,
    [id]
  )

  return c.json({
    data: {
      ...detail.rows[0],
      occupants: occupants.rows,
    },
    message: 'Detail rumah berhasil diambil',
  })
})

// ─── PUT /api/houses/:id — update ────────────────────────────
houses.put('/:id', async (c) => {
  try {
    const body = HouseUpdateSchema.parse(await c.req.json())
    const db = getDB()
    const tenant_id = c.var.tenant_id
    const id = c.req.param('id')
    const now = new Date()

    const setParts = []
    const values = [id, tenant_id, now]
    let i = 4

    for (const [key, val] of Object.entries(body)) {
      if (val !== undefined) {
        setParts.push(`${key} = $${i++}`)
        values.push(val)
      }
    }

    if (!setParts.length) {
      return c.json({ error: 'Tidak ada data untuk diupdate' }, 400)
    }

    setParts.push(`updated_at = $${i}`)

    const result = await db.query(
      `UPDATE houses SET ${setParts.join(', ')}
       WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
       RETURNING *`,
      values
    )

    if (!result.rows.length) {
      return c.json({ error: 'Rumah tidak ditemukan' }, 404)
    }

    return c.json({
      data: result.rows[0],
      message: 'Rumah berhasil diupdate',
      ok: true,
    })
  } catch (e: any) {
    if (e instanceof z.ZodError) {
      return c.json({ error: 'Validasi gagal', details: e.errors }, 400)
    }
    return c.json({ error: 'Gagal mengupdate rumah' }, 500)
  }
})

// ─── DELETE /api/houses/:id — soft delete ────────────────────
houses.delete('/:id', async (c) => {
  const db = getDB()
  const tenant_id = c.var.tenant_id
  const id = c.req.param('id')
  const now = new Date()

  const result = await db.query(
    `UPDATE houses SET deleted_at = $1, status = 'VACANT', updated_at = $1
     WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
     RETURNING id`,
    [now, id, tenant_id]
  )

  if (!result.rows.length) {
    return c.json({ error: 'Rumah tidak ditemukan' }, 404)
  }

  return c.json({
    message: 'Rumah berhasil dihapus (soft-delete)',
    ok: true,
  })
})

export { houses as houseRoutes }