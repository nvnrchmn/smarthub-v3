import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { api, auth, type Me } from '../lib/api'

const PENGURUS = ['TENANT_MANAGER', 'TREASURER', 'SECRETARY']
const ITEM = [
  { to: '/dasbor', label: 'Dasbor', peran: 'all' },
  { to: '/tagihan', label: 'Tagihan', peran: 'all' },
  { to: '/keuangan', label: 'Keuangan', peran: 'pengurus' },
  { to: '/laporan', label: 'Laporan', peran: 'pengurus' },
  { to: '/rumah', label: 'Rumah', peran: 'pengurus' },
  { to: '/sensus', label: 'Sensus', peran: 'pengurus' },
]

// Layout — header + menu navigasi (sidebar di desktop, bilah bawah di ponsel).
export default function Layout() {
  const nav = useNavigate()
  const loc = useLocation()
  const [me, setMe] = useState<Me | null>(null)

  useEffect(() => {
    api.me().then(setMe).catch(() => nav('/masuk'))
  }, [nav])

  const pengurus = !!me && (me.roles || []).some((r) => PENGURUS.includes(r))
  const items = ITEM.filter((i) => i.peran === 'all' || pengurus)

  function keluar() {
    auth.clear()
    nav('/masuk')
  }

  return (
    <div className="min-h-screen bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface/85 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <Link to="/" className="font-bold text-ink">
            Smarthub
          </Link>
          <div className="flex items-center gap-2">
            <span className="hidden text-xs text-ink2 sm:inline">
              {pengurus ? 'Pengurus' : 'Warga'}
            </span>
            <button onClick={keluar} className="btn-ghost !px-3 !py-1.5 text-xs">
              Keluar
            </button>
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-5xl gap-6 px-4 py-5">
        <nav className="hidden w-48 shrink-0 flex-col gap-1 md:flex">
          {items.map((i) => (
            <Link
              key={i.to}
              to={i.to}
              className={`rounded-xl px-3 py-2 text-sm transition-colors ${
                loc.pathname === i.to ? 'bg-brand/10 font-semibold text-brand' : 'text-ink2 hover:bg-subtle'
              }`}
            >
              {i.label}
            </Link>
          ))}
        </nav>
        <main className="min-w-0 flex-1 pb-24 md:pb-2">
          <Outlet />
        </main>
      </div>

      <nav className="fixed inset-x-0 bottom-0 z-10 flex border-t border-line bg-surface md:hidden">
        {items.slice(0, 5).map((i) => (
          <Link
            key={i.to}
            to={i.to}
            className={`flex-1 py-2.5 text-center text-xs ${
              loc.pathname === i.to ? 'font-semibold text-brand' : 'text-ink2'
            }`}
          >
            {i.label}
          </Link>
        ))}
      </nav>
    </div>
  )
}
