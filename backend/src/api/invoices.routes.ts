// src/api/invoices.routes.ts — CRUD tagihan (invoices) per rumah
import { Hono } from 'hono'

export const invoiceRoutes = new Hono()

// GET /api/invoices
// Query: ?status=PENDING&house_id=...&month=2025-01
invoiceRoutes.get('/', async (c) => {
  return c.json({ data: [], meta: { total: 0 } })
})

// POST /api/invoices
// Generate invoice (manual atau via cron)
invoiceRoutes.post('/', async (c) => {
  return c.json({ error: 'create handler not yet implemented' }, 501)
})

// GET /api/invoices/:id
invoiceRoutes.get('/:id', async (c) => {
  return c.json({ data: null })
})

// PATCH /api/invoices/:id
// Update status (mark as PAID, dll)
invoiceRoutes.patch('/:id', async (c) => {
  return c.json({ error: 'patch handler not yet implemented' }, 501)
})
