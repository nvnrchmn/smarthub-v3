import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { census, roleLabelCensus, auth, ApiError, type ResidentProfile } from '../lib/api'

// Antrean verifikasi untuk Sekretaris/Pengelola: daftar data tersamar, buka PII
// (tercatat audit), lihat berkas KTP/KK, lalu setujui atau tolak.
export default function Verifikasi() {
  const nav = useNavigate()
  const [list, setList] = useState<ResidentProfile[]>([])
  const [pilih, setPilih] = useState<ResidentProfile | null>(null)
  const [pii, setPii] = useState<{ nik: string; kk_number: string } | null>(null)
  const [docUrl, setDocUrl] = useState<string | null>(null)
  const [alasan, setAlasan] = useState('')
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState<{ t: 'ok' | 'err'; s: string } | null>(null)
  const [filter, setFilter] = useState('UNVERIFIED')

  async function muat(status = filter) {
    try {
      setList(await census.queue(status))
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) return nav('/masuk')
    }
  }

  useEffect(() => { muat() }, [filter]) // eslint-disable-line react-hooks/exhaustive-deps

  async function buka(id: string) {
    setPii(null); setDocUrl(null); setAlasan(''); setMsg(null)
    const d = await census.detail(id).catch(() => null)
    setPilih(d)
  }

  async function bukaPII() {
    if (!pilih) return
    setBusy(true)
    try {
      setPii(await census.reveal(pilih.id))
      setMsg({ t: 'ok', s: 'NIK & No. KK dibuka — akses ini tercatat di audit log.' })
    } catch (e) {
      setMsg({ t: 'err', s: e instanceof Error ? e.message : 'gagal membuka' })
    } finally {
      setBusy(false)
    }
  }

  async function lihatBerkas(kind: 'KTP' | 'KK') {
    if (!pilih) return
    setBusy(true)
    try {
      if (docUrl) URL.revokeObjectURL(docUrl)
      setDocUrl(await census.documentBlobUrl(pilih.id, kind))
    } catch (e) {
      setMsg({ t: 'err', s: e instanceof Error ? e.message : 'berkas tidak bisa dibuka' })
    } finally {
      setBusy(false)
    }
  }

  async function putuskan(status: 'VERIFIED' | 'REJECTED') {
    if (!pilih) return
    setBusy(true); setMsg(null)
    try {
      await census.verify(pilih.id, status, alasan)
      setMsg({ t: 'ok', s: status === 'VERIFIED' ? 'Data disetujui.' : 'Data ditolak, alasan dikirim ke warga.' })
      setPilih(null); setPii(null); setDocUrl(null)
      await muat()
    } catch (e) {
      setMsg({ t: 'err', s: e instanceof Error ? e.message : 'gagal menyimpan keputusan' })
    } finally {
      setBusy(false)
    }
  }

  const badge: Record<string, string> = {
    UNVERIFIED: 'border-warn/40 bg-warn/10 text-warn',
    VERIFIED: 'border-ok/40 bg-ok/10 text-ok',
    REJECTED: 'border-danger/40 bg-danger/10 text-danger',
  }

  return (
    <div className="min-h-dvh bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface backdrop-blur transition-colors duration-slow">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2 px-4 py-3">
          <div>
            <p className="text-xs text-muted">Sensus</p>
            <h1 className="text-lg font-bold text-ink">Antrean verifikasi</h1>
          </div>
          <div className="flex items-center gap-2">
            <select className="field w-auto" value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="UNVERIFIED">Menunggu</option>
              <option value="VERIFIED">Terverifikasi</option>
              <option value="REJECTED">Ditolak</option>
            </select>
            <button className="btn-ghost" onClick={() => nav('/dasbor')}>Dasbor</button>
          </div>
        </div>
      </header>

      <main className="mx-auto grid max-w-5xl gap-4 px-4 py-5 lg:grid-cols-2">
        <section>
          {msg && (
            <div className={`card mb-3 ${msg.t === 'ok' ? 'border-ok/40' : 'border-danger/40'}`}>
              <p className={msg.t === 'ok' ? 'text-ok' : 'text-danger'}>{msg.s}</p>
            </div>
          )}
          <div className="animate-list space-y-3">
            {list.length === 0 && (
              <div className="card">
                <p className="text-sm text-ink2">Tidak ada data pada status ini.</p>
              </div>
            )}
            {list.map((r) => (
              <button
                key={r.id}
                onClick={() => buka(r.id)}
                className={`card w-full text-left ${pilih?.id === r.id ? 'border-brand/60 shadow-md' : ''}`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold text-ink">{r.full_name}</p>
                    <p className="text-xs text-ink2">
                      NIK ••••••••{r.nik_last4 || '------'} · {roleLabelCensus[r.family_role] || r.family_role}
                    </p>
                    <p className="mt-1 text-xs text-muted">
                      {r.house_unit ? `Rumah ${r.house_unit}` : 'Rumah belum ditautkan'}
                      {' · KTP '}{r.has_ktp ? 'ada' : 'belum'}{' / KK '}{r.has_kk ? 'ada' : 'belum'}
                    </p>
                  </div>
                  <span className={`badge ${badge[r.verification_status] || ''}`}>{r.verification_status}</span>
                </div>
              </button>
            ))}
          </div>
        </section>

        <section className="lg:sticky lg:top-20 lg:self-start">
          {!pilih ? (
            <div className="card">
              <p className="text-sm text-ink2">Pilih satu data di sebelah kiri untuk diperiksa.</p>
            </div>
          ) : (
            <div className="card animate-rise-in space-y-3">
              <div className="flex items-start justify-between">
                <div>
                  <h2 className="font-semibold text-ink">{pilih.full_name}</h2>
                  <p className="text-xs text-ink2">{roleLabelCensus[pilih.family_role] || pilih.family_role}</p>
                </div>
                <span className={`badge ${badge[pilih.verification_status] || ''}`}>{pilih.verification_status}</span>
              </div>

              <dl className="grid grid-cols-2 gap-2 text-xs">
                <div><dt className="text-muted">Tempat/tanggal lahir</dt><dd className="text-ink">{pilih.birth_place || '-'}, {pilih.birth_date || '-'}</dd></div>
                <div><dt className="text-muted">Jenis kelamin</dt><dd className="text-ink">{pilih.gender === 'F' ? 'Perempuan' : pilih.gender === 'M' ? 'Laki-laki' : '-'}</dd></div>
                <div><dt className="text-muted">Pekerjaan</dt><dd className="text-ink">{pilih.occupation || '-'}</dd></div>
                <div><dt className="text-muted">Agama</dt><dd className="text-ink">{pilih.religion || '-'}</dd></div>
                <div><dt className="text-muted">Status kawin</dt><dd className="text-ink">{pilih.marital_status || '-'}</dd></div>
                <div><dt className="text-muted">Rumah</dt><dd className="text-ink">{pilih.house_unit || '-'}</dd></div>
              </dl>

              {pii ? (
                <div className="rounded-xl border border-brand/30 bg-brand/5 p-3 text-sm">
                  <p className="text-xs text-muted">NIK</p>
                  <p className="font-mono text-ink">{pii.nik}</p>
                  <p className="mt-2 text-xs text-muted">Nomor KK</p>
                  <p className="font-mono text-ink">{pii.kk_number || '-'}</p>
                </div>
              ) : (
                <button className="btn-ghost w-full" onClick={bukaPII} disabled={busy}>
                  Buka NIK &amp; Nomor KK (tercatat audit)
                </button>
              )}

              <div className="flex flex-wrap gap-2">
                {(['KTP', 'KK'] as const).map((k) => (
                  <button
                    key={k}
                    className="btn-ghost"
                    onClick={() => lihatBerkas(k)}
                    disabled={busy || (k === 'KTP' ? !pilih.has_ktp : !pilih.has_kk)}
                  >
                    Lihat {k}
                  </button>
                ))}
              </div>

              {docUrl && (
                <div className="overflow-hidden rounded-xl border border-line bg-subtle transition-all duration-slow">
                  <img src={docUrl} alt="Berkas dokumen" className="max-h-72 w-full object-contain" />
                </div>
              )}

              {pilih.verification_status !== 'VERIFIED' && (
                <>
                  <label className="block">
                    <span className="mb-1 block text-xs text-ink2">Catatan/alasan (wajib bila menolak)</span>
                    <input className="field" value={alasan} onChange={(e) => setAlasan(e.target.value)} maxLength={200} />
                  </label>
                  <div className="flex gap-2">
                    <button className="btn-primary flex-1" disabled={busy} onClick={() => putuskan('VERIFIED')}>
                      Setujui
                    </button>
                    <button className="btn-danger flex-1" disabled={busy} onClick={() => putuskan('REJECTED')}>
                      Tolak
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </section>
      </main>

      <footer className="mx-auto max-w-5xl px-4 pb-8">
        <p className="text-xs text-muted">
          Masuk sebagai {auth.get() ? 'pengurus' : '-'} · setiap pembukaan data pribadi dan berkas
          tercatat di audit log.
        </p>
      </footer>
    </div>
  )
}
