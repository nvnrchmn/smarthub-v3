import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { census, roleLabel, auth, ApiError, type ResidentProfile } from '../lib/api'

// Formulir sensus warga: data dasar + NIK/No. KK + unggah KTP & KK.
// Gerak mengikuti UIUX.md 1.3 (durasi lambat, stagger antar kartu).
export default function Sensus() {
  const nav = useNavigate()
  const [me, setMe] = useState<ResidentProfile | null>(null)
  const [form, setForm] = useState({
    full_name: '', nik: '', kk_number: '', family_role: 'HEAD_OF_FAMILY',
    birth_place: '', birth_date: '', gender: 'M', religion: '',
    marital_status: '', occupation: '', education: '',
  })
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState<{ t: 'ok' | 'err'; s: string } | null>(null)
  const [roles, setRoles] = useState<string[]>([])

  useEffect(() => {
    (async () => {
      try {
        setMe(await census.me())
      } catch (e) {
        // 404 = warga belum pernah mengisi sensus (bukan masalah sesi).
        if (e instanceof ApiError && e.status === 401) return nav('/masuk')
      }
      // peran untuk menu
      try {
        const r = await fetch('/api/me', { headers: { Authorization: `Bearer ${auth.get()}` } })
        const d = await r.json()
        setRoles(d.roles || [])
      } catch { /* menu staff opsional */ }
    })()
  }, [nav])

  const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm({ ...form, [k]: e.target.value })

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true); setMsg(null)
    try {
      const r = await census.submitMe(form)
      setMe(r)
      setMsg({ t: 'ok', s: 'Data tersimpan. Berkas KTP & KK bisa diunggah di bawah.' })
    } catch (err) {
      setMsg({ t: 'err', s: err instanceof Error ? err.message : 'gagal menyimpan' })
    } finally {
      setBusy(false)
    }
  }

  async function unggah(kind: 'KTP' | 'KK', file: File) {
    if (!me) return
    setBusy(true); setMsg(null)
    try {
      await census.upload(me.id, kind, file)
      setMe(await census.me())
      setMsg({ t: 'ok', s: `Berkas ${kind} terunggah, menunggu pemeriksaan sekretaris.` })
    } catch (err) {
      setMsg({ t: 'err', s: err instanceof Error ? err.message : 'gagal mengunggah' })
    } finally {
      setBusy(false)
    }
  }

  const statusStyle: Record<string, string> = {
    UNVERIFIED: 'border-warn/40 bg-warn/10 text-warn',
    VERIFIED: 'border-ok/40 bg-ok/10 text-ok',
    REJECTED: 'border-danger/40 bg-danger/10 text-danger',
  }
  const statusLabel: Record<string, string> = {
    UNVERIFIED: 'Menunggu verifikasi',
    VERIFIED: 'Terverifikasi',
    REJECTED: 'Ditolak',
  }

  return (
    <div className="min-h-dvh bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface backdrop-blur transition-colors duration-slow">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-4 py-3">
          <div>
            <p className="text-xs text-muted">Sensus warga</p>
            <h1 className="text-lg font-bold text-ink">Data kependudukan</h1>
          </div>
          <div className="flex items-center gap-2">
            {roles.some((r) => ['SECRETARY', 'TENANT_MANAGER'].includes(r)) && (
              <button className="btn-ghost" onClick={() => nav('/verifikasi')}>Verifikasi</button>
            )}
            <button className="btn-ghost" onClick={() => { void auth.logout().then(() => nav('/masuk')) }}>Keluar</button>
          </div>
        </div>
      </header>

      <main className="animate-list mx-auto max-w-3xl space-y-4 px-4 py-5">
        {!me && (
          <section className="card border-brand/30">
            <p className="text-sm text-ink2">
              Belum ada data sensus untuk akun ini. Isi formulir di bawah lalu simpan — setelah itu
              berkas KTP &amp; Kartu Keluarga bisa diunggah.
            </p>
          </section>
        )}
        {me && (
          <section className="card flex items-center justify-between">
            <div>
              <p className="text-xs text-muted">Status data Anda</p>
              <p className="font-semibold text-ink">{me.full_name}</p>
              <p className="text-xs text-ink2">NIK tersimpan: ••••••••{me.nik_last4 || '------'}</p>
            </div>
            <span className={`badge ${statusStyle[me.verification_status] || ''}`}>
              {statusLabel[me.verification_status] || me.verification_status}
            </span>
          </section>
        )}

        {msg && (
          <div className={`card ${msg.t === 'ok' ? 'border-ok/40' : 'border-danger/40'}`}>
            <p className={msg.t === 'ok' ? 'text-ok' : 'text-danger'}>{msg.s}</p>
          </div>
        )}

        <form onSubmit={simpan} className="card space-y-4">
          <h2 className="font-semibold text-ink">Data dasar</h2>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Nama lengkap</span>
              <input className="field" value={form.full_name} onChange={set('full_name')} required />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Peran dalam keluarga</span>
              <select className="field" value={form.family_role} onChange={set('family_role')}>
                <option value="HEAD_OF_FAMILY">Kepala keluarga</option>
                <option value="SPOUSE">Istri/suami</option>
                <option value="CHILD">Anak</option>
                <option value="OTHER">Lainnya</option>
              </select>
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">NIK (16 angka, tersimpan terenkripsi)</span>
              <input className="field" inputMode="numeric" maxLength={16} value={form.nik} onChange={set('nik')} required />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Nomor Kartu Keluarga</span>
              <input className="field" inputMode="numeric" maxLength={16} value={form.kk_number} onChange={set('kk_number')} />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Tempat lahir</span>
              <input className="field" value={form.birth_place} onChange={set('birth_place')} />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Tanggal lahir</span>
              <input className="field" type="date" value={form.birth_date} onChange={set('birth_date')} />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Jenis kelamin</span>
              <select className="field" value={form.gender} onChange={set('gender')}>
                <option value="M">Laki-laki</option>
                <option value="F">Perempuan</option>
              </select>
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Pekerjaan</span>
              <input className="field" value={form.occupation} onChange={set('occupation')} />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Agama</span>
              <input className="field" value={form.religion} onChange={set('religion')} />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs text-ink2">Status perkawinan</span>
              <input className="field" value={form.marital_status} onChange={set('marital_status')} />
            </label>
          </div>
          <button className="btn-primary w-full sm:w-auto" disabled={busy}>
            {busy ? 'Menyimpan…' : 'Simpan data'}
          </button>
        </form>

        <section className="card space-y-3">
          <h2 className="font-semibold text-ink">Berkas KTP &amp; Kartu Keluarga</h2>
          <p className="text-xs text-ink2">
            Berkas disimpan di penyimpanan privat dan hanya bisa dibuka sekretaris/pengelola
            (setiap pembukaan tercatat).
          </p>
          {(['KTP', 'KK'] as const).map((kind) => (
            <div key={kind} className="flex items-center justify-between rounded-xl border border-line bg-canvas p-3
                                       transition-all duration-base ease-spring hover:border-brand/40">
              <div>
                <p className="text-sm font-medium text-ink">Foto {kind}</p>
                <p className="text-xs text-ink2">
                  {kind === 'KTP'
                    ? (me?.has_ktp ? 'Sudah terunggah' : 'Belum ada')
                    : (me?.has_kk ? 'Sudah terunggah' : 'Belum ada')}
                </p>
              </div>
              <label className="btn-ghost cursor-pointer">
                {me && ((kind === 'KTP' && me.has_ktp) || (kind === 'KK' && me.has_kk)) ? 'Ganti' : 'Unggah'}
                <input
                  type="file"
                  accept="image/*,application/pdf"
                  className="hidden"
                  onChange={(e) => e.target.files?.[0] && unggah(kind, e.target.files[0])}
                />
              </label>
            </div>
          ))}
          {me?.house_unit && (
            <p className="text-xs text-muted">Rumah terdaftar: {me.house_unit}</p>
          )}
        </section>

        <p className="text-center text-xs text-muted">
          Peran akun: {roles.map((r) => roleLabel[r] || r).join(', ') || '—'}
        </p>
      </main>
    </div>
  )
}
