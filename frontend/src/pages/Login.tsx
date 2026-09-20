import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, auth } from '../lib/api'

// Halaman masuk. Tidak ada pendaftaran mandiri: akun hanya lahir dari undangan
// pengelola (invite-only), sehingga halaman ini hanya untuk masuk.
// Dibuat menarik versi Linear: bg gradient animasi, elevated card, input glow,
// password visibility toggle, WhatsApp login toggle, spinner mikro.

export default function Login() {
  const [mode, setMode] = useState<'email' | 'wa'>('email')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const nav = useNavigate()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr('')
    setBusy(true)
    try {
      const res = await api.login(email.trim(), password)
      auth.set(res.token)
      nav('/dasbor')
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Gagal masuk')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-canvas text-ink motion-safe:bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] motion-safe:from-slate-950/80 motion-safe:via-slate-950/80 motion-safe:to-slate-900/90">
      {/* Subtle mesh / noise overlay */}
      <div
        className="pointer-events-none absolute inset-0 -z-10 overflow-hidden"
        aria-hidden="true"
      >
        <div className="absolute -top-1/2 left-1/2 h-[600px] w-[600px] -translate-x-1/2 rounded-full bg-brand/5 blur-3xl filter saturate-150" />
        <div className="absolute bottom-0 right-0 h-[400px] w-[400px] rounded-full bg-indigo-600/5 blur-3xl filter" />
      </div>

      <div className="mx-auto w-full max-w-md">
        {/* Logo */}
        <Link to="/" className="mb-8 flex items-center justify-center gap-2 text-xl font-bold">
          <span className="relative flex h-8 w-3">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-brand opacity-75" />
            <span className="relative inline-flex h-8 w-3 rounded-full bg-brand" />
          </span>
          <span>Smarthub<span className="text-brand">.</span></span>
        </Link>

        {/* Elevated card */}
        <div className="rounded-2xl border border-white/[0.06] bg-surface/80 p-8 shadow-2xl shadow-black/40 backdrop-blur-xl">
          <div className="mb-6 text-center">
            <h1 className="text-2xl font-bold tracking-tight text-ink">Masuk ke Smarthub</h1>
            <p className="mt-1.5 text-sm text-ink2">
              Akun hanya tersedia lewat undangan pengelola perumahan.
            </p>
          </div>

          {/* Toggle email vs WhatsApp */}
          <div className="mb-6 grid grid-cols-2 gap-1.5 rounded-xl bg-slate-800/60 p-1 text-xs font-medium text-ink2">
            <button
              type="button"
              onClick={() => setMode('email')}
              className={`rounded-lg px-3 py-2 transition-all ${mode === 'email' ? 'bg-brand text-white shadow' : 'hover:bg-white/[0.05]'}`}
            >
              Email
            </button>
            <button
              type="button"
              onClick={() => setMode('wa')}
              className={`rounded-lg px-3 py-2 transition-all ${mode === 'wa' ? 'bg-brand text-white shadow' : 'hover:bg-white/[0.05]'}`}
            >
              WhatsApp
            </button>
          </div>

          <form onSubmit={submit} className="space-y-4">
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-ink2">
                {mode === 'wa' ? 'Nomor HP' : 'Email'}
              </span>
              <input
                type={mode === 'wa' ? 'tel' : 'email'}
                required
                autoComplete={mode === 'wa' ? 'tel' : 'email'}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder={mode === 'wa' ? '08xx-xxxx-xxxx' : 'nama@email.com'}
                className="peer w-full rounded-xl border-2 border-line bg-transparent px-4 py-3 text-ink placeholder-transparent outline-none transition-all duration-200 focus:border-brand focus:ring-2 focus:ring-brand/20 autofill:bg-transparent"
              />
              <span className="mt-1 block text-xs text-ink2">Bisa pakai email atau nomor WhatsApp terdaftar.</span>
            </label>

            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-ink2">Kata sandi</span>
              <div className="relative">
                <input
                  type={showPw ? 'text' : 'password'}
                  required
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Kata sandi"
                  className="peer w-full rounded-xl border-2 border-line bg-transparent px-4 py-3 text-ink placeholder-transparent outline-none transition-all duration-200 focus:border-brand focus:ring-2 focus:ring-brand/20"
                />
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => setShowPw(!showPw)}
                  className="absolute inset-y-0 right-0 flex items-center justify-center rounded-r-xl px-3 text-ink2/50 transition-colors hover:text-ink2"
                  aria-label={showPw ? 'Sembunyikan kata sandi' : 'Lihat kata sandi'}
                >
                  {showPw ? '🙈' : '👁️'}
                </button>
              </div>
            </label>

            {err && (
              <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger animate-in fade-in-0 zoom-in-95">
                {err}
              </div>
            )}

            <button
              disabled={busy || !email || !password}
              className="relative flex w-full items-center justify-center gap-2 rounded-xl bg-brand px-4 py-3 font-semibold text-white transition-all duration-200 hover:bg-brand/90 hover:shadow-lg hover:shadow-brand/20 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60"
            >
              {busy ? (
                <>
                  <svg className="-ml-1 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 0116 0M12 5.5v3l2 2" />
                  </svg>
                  Memproses…
                </>
              ) : (
                'Masuk'
              )}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}