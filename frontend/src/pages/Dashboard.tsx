import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, auth, roleLabel, type Me } from '../lib/api'

const canInvite = (roles: string[]) => roles.includes('TENANT_MANAGER') || roles.includes('SECRETARY')

export default function Dashboard() {
  const [me, setMe] = useState<Me | null>(null)
  const [err, setErr] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [role, setRole] = useState('RESIDENT')
  const [link, setLink] = useState('')
  const [busy, setBusy] = useState(false)
  const nav = useNavigate()

  useEffect(() => {
    api
      .me()
      .then(setMe)
      .catch(() => {
        auth.clear()
        nav('/masuk')
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function submitInvite(e: React.FormEvent) {
    e.preventDefault()
    setErr('')
    setLink('')
    setBusy(true)
    try {
      const res = await api.invite({ email: email.trim(), phone: phone.trim(), role })
      setLink(res.invite_link)
      setEmail('')
      setPhone('')
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Gagal mengundang')
    } finally {
      setBusy(false)
    }
  }

  if (!me) return <div className="min-h-screen bg-canvas px-5 py-16 text-center text-ink2">Memuat…</div>

  return (
    <div className="min-h-screen bg-canvas text-ink">
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-5 py-4">
          <Link to="/" className="font-bold">
            Smarthub<span className="text-brand">.</span>
          </Link>
          <button
            onClick={() => {
              // Server mencabut token ini lebih dulu, lalu keluar. Berbeda dari
              // sekadar menghapus token di peramban, sesi di server ikut mati.
              void auth.logout().then(() => nav('/masuk'))
            }}
            className="rounded-lg border border-line px-3 py-1.5 text-xs font-semibold text-ink2"
          >
            Keluar
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-5 py-8">
        <h1 className="text-xl font-bold">Dasbor</h1>
        <p className="mt-1 text-sm text-ink2">
          Peran: {me.roles.map((r) => roleLabel[r] || r).join(', ')} · status {me.status}
        </p>

        {canInvite(me.roles) ? (
          <section className="mt-6 rounded-2xl border border-line bg-surface p-5">
            <h2 className="font-semibold">Undang warga / pengurus</h2>
            <p className="mt-1 text-sm text-ink2">
              Tautan aktivasi berlaku 7 hari; kode OTP dikirim ke nomor HP tujuan.
            </p>
            <div className="mb-2 flex flex-wrap justify-end gap-2">
              <button onClick={() => nav('/verifikasi')} className="btn-ghost">Verifikasi sensus</button>
              <button onClick={() => nav('/rumah')} className="btn-ghost">Kelola rumah &amp; KK</button>
              <button onClick={() => nav('/keuangan')} className="btn-ghost">Keuangan</button>
            </div>
            <form onSubmit={submitInvite} className="mt-4 grid gap-3 sm:grid-cols-2">
              <input
                type="email"
                required
                placeholder="email@contoh.id"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="rounded-xl border border-line bg-canvas px-4 py-3 outline-none focus:border-brand"
              />
              <input
                required
                placeholder="08xxxxxxxxxx"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                className="rounded-xl border border-line bg-canvas px-4 py-3 outline-none focus:border-brand"
              />
              <select
                value={role}
                onChange={(e) => setRole(e.target.value)}
                className="rounded-xl border border-line bg-canvas px-4 py-3 outline-none focus:border-brand"
              >
                <option value="RESIDENT">Warga</option>
                <option value="SECRETARY">Sekretaris</option>
                <option value="TREASURER">Bendahara</option>
                <option value="TENANT_MANAGER">Pengelola</option>
              </select>
              <button
                disabled={busy}
                className="rounded-xl bg-brand px-4 py-3 font-semibold text-white disabled:opacity-60"
              >
                {busy ? 'Mengirim…' : 'Kirim undangan'}
              </button>
            </form>
            {err && <p className="mt-3 text-sm text-danger">{err}</p>}
            {link && (
              <div className="mt-4 rounded-xl border border-line bg-canvas p-4">
                <p className="text-xs text-muted">Tautan aktivasi</p>
                <p className="mt-1 break-all text-sm">{link}</p>
              </div>
            )}
          </section>
        ) : (
          <section className="mt-6 space-y-3">
            <h2 className="font-semibold text-ink">Menu warga</h2>
            <div className="animate-list space-y-3">
              <button onClick={() => nav('/sensus')} className="card w-full text-left">
                <p className="font-semibold text-ink">Data sensus &amp; dokumen</p>
                <p className="text-xs text-ink2">
                  Lengkapi NIK, nomor KK, dan unggah foto KTP &amp; Kartu Keluarga untuk diverifikasi sekretaris.
                </p>
              </button>
              <button onClick={() => nav('/tagihan')} className="card w-full text-left">
                <p className="font-semibold text-ink">Tagihan iuran</p>
                <p className="text-xs text-ink2">
                  Lihat tagihan rumahmu, bayar via QRIS, atau setor tunai ke Bendahara.
                </p>
              </button>
            </div>
          </section>
        )}
      </main>
    </div>
  )
}
