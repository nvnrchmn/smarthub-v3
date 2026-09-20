// src/api/billing.routes.ts — CRUD master billing items (iuran)
import { Hono } from 'hono'

export const billingRoutes = new Hono()

// GET /api/billing
// List semua billing item template
billingRoutes.get('/', async (c) => {
  return c.json({ data: [], meta: { total: 0 } })
})

// POST /api/billing
// Create new billing item
billingRoutes.post('/', async (c) => {
  return c.json({ error: 'create handler not yet implemented' }, 501)
})

// PATCH /api/billing/:id
billingRoutes.patch('/:id', async (c) => {
  return c.json({ error: 'patch handler not yet implemented' }, 501)
})

// DELETE /api/billing/:id
billingRoutes.delete('/:id', async (c) => {
  return c.json({ error: 'delete handler not yet implemented' }, 501)
})
