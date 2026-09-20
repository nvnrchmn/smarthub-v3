// src/api/houses.routes.ts — CRUD untuk manajemen rumah (houses)
import { Hono } from 'hono'

export const houseRoutes = new Hono()

// GET /api/houses
// Query: ?blok=A&status=OCCUPIED&page=1&limit=50
houseRoutes.get('/', async (c) => {
  return c.json({ data: [], meta: { page: 1, limit: 50, total: 0 } })
})

// POST /api/houses
// Create rumah baru
houseRoutes.post('/', async (c) => {
  return c.json({ error: 'create handler not yet implemented' }, 501)
})

// GET /api/houses/:id
houseRoutes.get('/:id', async (c) => {
  return c.json({ data: null })
})

// PATCH /api/houses/:id
houseRoutes.patch('/:id', async (c) => {
  return c.json({ error: 'patch handler not yet implemented' }, 501)
})

// DELETE /api/houses/:id
houseRoutes.delete('/:id', async (c) => {
  return c.json({ error: 'delete handler not yet implemented' }, 501)
})
