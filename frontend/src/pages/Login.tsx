import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, auth } from '../lib/api'

// Halaman masuk. Tidak ada pendaftaran mandiri: akun hanya lahir dari undangan
// pengelola (invite-only), sehingga halaman ini hanya untuk masuk.
export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
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
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-10 bg-canvas text-ink">
      <Link to="/" className="text-lg font-bold">
        Smarthub<span className="text-brand">.</span>
      </Link>
      <h1 className="mt-6 text-2xl font-bold">Masuk</h1>
      <p className="mt-1 text-sm text-ink2">
        Akun dibuat oleh pengelola perumahan. Belum punya undangan? Hubungi pengelola Anda.
      </p>

      <form onSubmit={submit} className="mt-6 space-y-4">
        <label className="block">
          <span className="text-sm font-medium">Email</span>
          <input
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="mt-1 w-full rounded-xl border border-line bg-surface px-4 py-3 outline-none focus:border-brand"
          />
        </label>
        <label className="block">
          <span className="text-sm font-medium">Kata sandi</span>
          <input
            type="password"
            required
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="mt-1 w-full rounded-xl border border-line bg-surface px-4 py-3 outline-none focus:border-brand"
          />
        </label>

        {err && <p className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{err}</p>}

        <button
          disabled={busy}
          className="w-full rounded-xl bg-brand px-4 py-3 font-semibold text-white disabled:opacity-60"
        >
          {busy ? 'Memproses…' : 'Masuk'}
        </button>
      </form>
    </div>
  )
}
