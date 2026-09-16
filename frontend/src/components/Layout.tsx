import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { api, auth, type Me } from '../lib/api'

const PENGURUS = ['TENANT_MANAGER', 'TREASURER', 'SECRETARY']
const ITEM = [
  { to: '/dasbor', label: 'Dasbor', icon: '◉', peran: 'all' },
  { to: '/tagihan', label: 'Tagihan', icon: '⬡', peran: 'all' },
  { to: '/keuangan', label: 'Keuangan', icon: '◇', peran: 'pengurus' },
  { to: '/laporan', label: 'Laporan', icon: '▣', peran: 'pengurus' },
  { to: '/rumah', label: 'Rumah', icon: '⌂', peran: 'pengurus' },
  { to: '/sensus', label: 'Sensus', icon: '⊟', peran: 'pengurus' },
]

// Layout Linear-style: sidebar + main
export default function Layout() {
  const nav = useNavigate()
  const loc = useLocation()
  const [me, setMe] = useState<Me | null>(null)
  const [dark, setDark] = useState(() => {
    if (typeof window === 'undefined') return false
    return localStorage.getItem('theme') === 'dark' ||
      (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  }, [dark])

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
      {/* Sidebar desktop */}
      <aside className="fixed inset-y-0 left-0 z-20 hidden w-56 border-r border-line bg-surface lg:block">
        <div className="flex h-12 items-center gap-2 border-b border-line px-4">
          <span className="grid h-7 w-7 place-items-center rounded-lg bg-brand text-sm font-bold text-white">S</span>
          <span className="text-sm font-bold text-ink">Smarthub</span>
        </div>

        <nav className="space-y-0.5 p-3 animate-list">
          {items.map((i) => (
            <Link
              key={i.to}
              to={i.to}
              className={`nav-item ${loc.pathname === i.to ? 'active' : ''}`}
            >
              <span className="text-base opacity-70">{i.icon}</span>
              <span>{i.label}</span>
            </Link>
          ))}
        </nav>

        <div className="absolute inset-x-0 bottom-0 border-t border-line p-3">
          <button onClick={() => setDark(!dark)} className="btn-icon w-full" title="Toggle theme">
            {dark ? '☀' : '☾'}
          </button>
          <button onClick={keluar} className="btn-ghost mt-2 w-full">
            Keluar
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="lg:pl-56">
        {/* Mobile header */}
        <header className="sticky top-0 z-10 border-b border-line bg-surface/85 backdrop-blur">
          <div className="flex items-center gap-3 px-4 py-3">
            <Link to="/" className="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-bold text-white lg:hidden">
              S
            </Link>
            <span className="text-sm font-bold text-ink lg:hidden">Smarthub</span>
            <span className="ml-auto text-xs text-ink2">
              {pengurus ? 'Pengurus' : 'Warga'}
            </span>
            <button onClick={() => setDark(!dark)} className="btn-icon text-sm" title="Toggle theme">
              {dark ? '☀' : '☾'}
            </button>
            <button onClick={keluar} className="btn-ghost !px-3 !py-1.5 text-xs">
              Keluar
            </button>
          </div>
        </header>

        <main className="mx-auto max-w-3xl px-4 py-6">
          <Outlet />
        </main>

        {/* Mobile bottom nav */}
        <nav className="fixed inset-x-0 bottom-0 z-10 flex border-t border-line bg-surface lg:hidden">
          {items.slice(0, 5).map((i) => (
            <Link
              key={i.to}
              to={i.to}
              className={`flex-1 py-2.5 text-center text-xs ${loc.pathname === i.to ? 'font-semibold text-brand' : 'text-ink2'}`}
            >
              <div className="text-base">{i.icon}</div>
              {i.label}
            </Link>
          ))}
        </nav>
      </div>
    </div>
  )
}
