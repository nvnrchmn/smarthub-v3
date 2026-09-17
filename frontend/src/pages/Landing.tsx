import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

type Health = { status: string; database: boolean; redis: boolean; service: string }

const roles = [
  { name: 'Warga', desc: 'Lihat tagihan iuran, bayar QRIS, ajukan surat, dan akses lapak warga.', icon: '🏠' },
  { name: 'Sekretaris', desc: 'Verifikasi sensus (NIK/KK terenkripsi), kelola data rumah & warga.', icon: '📝' },
  { name: 'Bendahara', desc: 'Terbitkan tagihan, kuitansi digital, dan buku kas masuk.', icon: '💰' },
  { name: 'Pengelola', desc: 'Kelola tenant, pengurus, kebijakan, dan laporan perumahan.', icon: '⚙️' },
]

export default function Landing() {
  const [health, setHealth] = useState<Health | null>(null)
  const [dark, setDark] = useState(true) // dark mode default

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    document.documentElement.classList.toggle('light', !dark)
  }, [dark])

  useEffect(() => {
    fetch('/api/health')
      .then((r) => (r.ok ? r.json() : null))
      .then(setHealth)
      .catch(() => setHealth(null))
  }, [])

  return (
    <div className="min-h-screen bg-canvas text-ink">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-line bg-surface/80 backdrop-blur-lg">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <span className="text-xl font-bold tracking-tight">
            Smarthub<span className="text-brand">.</span>
          </span>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setDark((v) => !v)}
              className="btn-ghost"
              aria-label="Toggle theme"
            >
              {dark ? (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />
                </svg>
              ) : (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z" />
                </svg>
              )}
            </button>
            <Link to="/masuk" className="btn-primary text-sm">
              Masuk
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <main className="mx-auto max-w-6xl px-6 py-16 sm:py-24">
        <div className="animate-rise">
          {/* Status Badge */}
          <div className="mb-6 flex items-center gap-2">
            <span className={`status-dot ${health?.database ? 'ok' : 'warn'}`} />
            <span className="text-sm font-medium text-ink2">
              {health ? `API ${health.status} · database ${health.database ? 'terhubung' : 'bermasalah'}` : 'memeriksa…'}
            </span>
          </div>

          {/* Headline */}
          <h1 className="max-w-3xl text-4xl font-bold leading-tight tracking-tight sm:text-5xl lg:text-6xl">
            Satu platform untuk
            <span className="bg-gradient-to-r from-brand to-accent bg-clip-text text-transparent"> RT/RW & perumahan</span>
          </h1>

          {/* Sub-headline */}
          <p className="mt-6 max-w-2xl text-lg leading-relaxed text-ink2 sm:text-xl">
            Iuran bulanan dengan QRIS dinamis, sensus warga dengan data pribadi terenkripsi,
            pengingat tunggakan, dan buku kas — dalam satu sistem multi-perumahan.
          </p>

          {/* CTA */}
          <div className="mt-10 flex flex-wrap items-center gap-4">
            <Link to="/masuk" className="btn-primary">
              Mulai Sekarang
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
              </svg>
            </Link>
            <Link to="/aktivasi" className="btn-secondary">
              Aktivasi Akun
            </Link>
          </div>
        </div>

        {/* Feature Grid */}
        <section className="mt-20 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {roles.map((r, i) => (
            <article
              key={r.name}
              className={`card-interactive animate-rise animate-rise-delay-${i + 1}`}
            >
              <div className="mb-3 text-2xl">{r.icon}</div>
              <h2 className="font-semibold">{r.name}</h2>
              <p className="mt-1 text-sm leading-relaxed text-ink2">{r.desc}</p>
            </article>
          ))}
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-line py-8">
        <div className="mx-auto max-w-6xl px-6 text-center text-sm text-muted">
          Smarthub v3 · dibangun ulang 2026 · PT Logika Kreatif Indonesia
        </div>
      </footer>
    </div>
  )
}
