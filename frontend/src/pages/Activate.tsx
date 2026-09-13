import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { api, auth } from '../lib/api'

// Aktivasi undangan: token dari tautan (berlaku 7 hari) + OTP yang dikirim ke
// nomor HP (saat ini lewat kanal WhatsApp tenant), lalu warga membuat kata sandi.
export default function Activate() {
  const [params] = useSearchParams()
  const token = params.get('token') || ''
  const [otp, setOtp] = useState('')
  const [fullName, setFullName] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  const nav = useNavigate()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr('')
    setInfo('')
    setBusy(true)
    try {
      const res = await api.accept({ token, otp: otp.trim(), full_name: fullName.trim(), password })
      auth.set(res.token)
      nav('/dasbor')
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Aktivasi gagal')
    } finally {
      setBusy(false)
    }
  }

  async function resend() {
    setErr('')
    setInfo('')
    try {
      await api.resendOtp(token)
      setInfo('Kode baru sudah dikirim ke nomor Anda.')
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Gagal mengirim ulang kode')
    }
  }

  if (!token) {
    return (
      <div className="mx-auto max-w-md px-5 py-16 text-center text-ink">
        <h1 className="text-xl font-bold">Tautan tidak lengkap</h1>
        <p className="mt-2 text-sm text-ink2">
          Buka tautan aktivasi persis seperti yang dikirim pengelola perumahan.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-10 bg-canvas text-ink">
      <h1 className="text-2xl font-bold">Aktivasi akun</h1>
      <p className="mt-1 text-sm text-ink2">
        Masukkan 6 digit kode yang dikirim ke nomor HP Anda, lalu buat kata sandi.
      </p>

      <form onSubmit={submit} className="mt-6 space-y-4">
        <label className="block">
          <span className="text-sm font-medium">Kode OTP</span>
          <input
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            required
            value={otp}
            onChange={(e) => setOtp(e.target.value.replace(/\D/g, ''))}
            className="mt-1 w-full rounded-xl border border-line bg-surface px-4 py-3 text-center text-xl tracking-[0.4em] outline-none focus:border-brand"
          />
        </label>
        <label className="block">
          <span className="text-sm font-medium">Nama lengkap</span>
          <input
            required
            value={fullName}
            onChange={(e) => setFullName(e.target.value)}
            className="mt-1 w-full rounded-xl border border-line bg-surface px-4 py-3 outline-none focus:border-brand"
          />
        </label>
        <label className="block">
          <span className="text-sm font-medium">Kata sandi baru</span>
          <input
            type="password"
            required
            minLength={8}
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="mt-1 w-full rounded-xl border border-line bg-surface px-4 py-3 outline-none focus:border-brand"
          />
          <span className="mt-1 block text-xs text-muted">Minimal 8 karakter.</span>
        </label>

        {err && <p className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{err}</p>}
        {info && <p className="rounded-xl border border-ok/30 bg-ok/10 px-4 py-3 text-sm text-ok">{info}</p>}

        <button
          disabled={busy}
          className="w-full rounded-xl bg-brand px-4 py-3 font-semibold text-white disabled:opacity-60"
        >
          {busy ? 'Mengaktifkan…' : 'Aktifkan akun'}
        </button>
      </form>

      <button onClick={resend} className="mt-4 text-sm font-medium text-brand underline">
        Kirim ulang kode
      </button>
    </div>
  )
}
