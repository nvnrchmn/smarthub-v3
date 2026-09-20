// src/api/payments.routes.ts — CRUD pembayaran (payments) via QRIS/Xendit
import { Hono } from 'hono'

export const paymentRoutes = new Hono()

// GET /api/payments
// Query: ?invoice_id=...&house_id=...&method=QRIS
paymentRoutes.get('/', async (c) => {
  return c.json({ data: [], meta: { total: 0 } })
})

// POST /api/payments
// Membuat payment record setelah konfirmasi dari Xendit webhook
paymentRoutes.post('/', async (c) => {
  return c.json({ error: 'create handler not yet implemented' }, 501)
})

// GET /api/payments/:id
paymentRoutes.get('/:id', async (c) => {
  return c.json({ data: null })
})
