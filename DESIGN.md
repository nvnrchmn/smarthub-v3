# DESIGN.md — Smarthub V3

> Linear-app style, dark mode default (brand reason: dev tool ergonomics), 2 core colors + 1 accent. Draft reviewed by owner.

## Identity & Personality

**Product:** Smarthub V3 — multi-tenant residential SaaS (Go Fiber v3 + React + TypeScript + Tailwind)
**Tonality:** Profesional tapi tidak kaku. Developer-friendly. Tenang tapi tidak datar.
**Audience:** Tenant manager (owner/secretary/treasurer/resident) — B2B2C, usia 25-55, familiar dengan dashboard digital.

## Visual Language

**Style:** Linear.app — minimalisme dengan depth halus, typography hierarchy jelas, spacing konsisten. Bukan kloning mekanis; kami gunakan spacing & hierarchy Linear sebagai inspirasi, tapi semua komponen kami dirancang ulang dengan karakter sendiri.

**Color Palette (max 3 core + 1 accent):**
- Primary: `#1e1e27` (dark canvas)
- Secondary: `#2a2a3a` (elevated surfaces)
- Neutral: `#e8e8ed` (text primary)
- Accent: `#6366f1` → `#818cf8` (indigo gradient — aksi, link, state aktif)

**Typography:**
- Headings: Inter (atau DM Sans) — semi-bold 600, weight hierarchy jelas
- Body: Inter — regular 400
- Code: JetBrains Mono — untuk angka/ID unit

## Dials

| Dial | Nilai | Alasan |
|------|-------|--------|
| ENERGY | 1 (Linear, GOV.UK) | Produk B2B, fokus fungsionalitas. Tak perlu visual "Awwwards". |
| RHYTHM | 2 (Stripe, Vercel) | Konsisten dengan variasi — section dem section mirip tapi dengan break halus di titik penting. |
| MOTION | 1 (Hover only) | Performa lebih penting. Hanya hover state & transisi halus, tidak ada parallax. |

## Components

- **Kartu utama**: radius `12px`, border `1px solid rgba(255,255,255,0.04)`, shadow `0 4px 24px rgba(0,0,0,0.25)`
- **Tombol aksi**: gradient indigo, ring `rgba(99,102,241,0.3)`, glow hanya saat hover
- **Sidebar**: lebar `240px`, collapse ikon di mobile, hover expand di desktop
- **Navbar**: fixed top, elevation `rgba(0,0,0,0.15)`, backdrop-blur `12px`

## Motion Policy

- Transisi default: `all 0.2s cubic-bezier(0.4,0,0.2,1)`
- Page load: fade-in + translateY(4px) untuk konten utama
- Interactive: hanya hover state, tidak ada entrance animation berlapis

## Dark Mode

- Default dark (`#1e1e27`) — alasan: tool dev, mengurangi eye strain di environment terang sepenuhnya
- Toggle light/dark — kedua mode ditest, contrast WCAG AA ≥ 4.5:1

## Notes

- Ini bukan draft — ini arah eksplisit yang diminta oleh owner. Gunakan sebagai sumber keputusan visual.
- Anti-slop core akan diterapkan. R-30 tidak berlaku (kloning tidak dilakukan — gaya Linear hanya inspirasi).
