import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

type Health = { status: string; database: boolean; service: string }

const roles = [
  { name: 'Warga', desc: 'Lihat tagihan iuran, bayar QRIS, ajukan surat, dan akses lapak warga.' },
  { name: 'Sekretaris', desc: 'Verifikasi sensus (NIK/KK terenkripsi), kelola data rumah & warga.' },
  { name: 'Bendahara', desc: 'Terbitkan tagihan, kuitansi digital, dan buku kas masuk.' },
  { name: 'Pengelola', desc: 'Kelola tenant, pengurus, kebijakan, dan laporan perumahan.' },
]

export default function Landing() {
  const [health, setHealth] = useState<Health | null>(null)
  const [dark, setDark] = useState(false)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
  }, [dark])

  useEffect(() => {
    fetch('/api/health')
      .then((r) => (r.ok ? r.json() : null))
      .then(setHealth)
      .catch(() => setHealth(null))
  }, [])

  return (
    <div className="min-h-screen bg-canvas text-ink">
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-5 py-4">
          <span className="text-lg font-bold">Smarthub<span className="text-brand">.</span></span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setDark((v) => !v)}
              className="rounded-lg border border-line px-3 py-1.5 text-xs font-semibold text-ink2"
            >
              {dark ? 'Mode terang' : 'Mode gelap'}
            </button>
            <Link
              to="/masuk"
              className="rounded-lg bg-brand px-3 py-1.5 text-xs font-semibold text-white"
            >
              Masuk
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-5 py-10">
        <h1 className="text-3xl font-bold sm:text-4xl">Satu platform untuk RT/RW & perumahan</h1>
        <p className="mt-3 max-w-2xl text-ink2">
          Iuran bulanan dengan QRIS dinamis, sensus warga dengan data pribadi terenkripsi,
          persuratan, forum, dan lapak warga — dalam satu sistem multi-perumahan.
        </p>

        <div className="mt-6 flex items-center gap-2 text-xs text-muted">
          <span className={'h-2 w-2 rounded-full ' + (health?.database ? 'bg-ok' : 'bg-warn')} />
          {health ? `API ${health.status} · database ${health.database ? 'terhubung' : 'bermasalah'}` : 'memeriksa API…'}
        </div>

        <section className="mt-8 grid gap-4 sm:grid-cols-2">
          {roles.map((r) => (
            <article key={r.name} className="rounded-2xl border border-line bg-surface p-5">
              <h2 className="font-semibold">{r.name}</h2>
              <p className="mt-1 text-sm text-ink2">{r.desc}</p>
            </article>
          ))}
        </section>
      </main>

      <footer className="border-t border-line py-6 text-center text-xs text-muted">
        Smarthub v3 · dibangun ulang 2026
      </footer>
    </div>
  )
}
