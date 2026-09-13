import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ApiError,
  auth,
  census,
  familyCards,
  houses,
  occupancy,
  roleLabelCensus,
  UNIT_STATUS_LABEL,
  type FamilyCard,
  type FamilyMember,
  type HouseUnit,
  type ResidentProfile,
} from '../lib/api'

// Kelola rumah & Kartu Keluarga (pengurus). Dua tab, semua aksi tercatat audit
// di server. Gerak mengikuti dokumen UI/UX: durasi lambat + stagger.
export default function Rumah() {
  const nav = useNavigate()
  const [tab, setTab] = useState<'rumah' | 'kk'>('rumah')
  const [units, setUnits] = useState<HouseUnit[]>([])
  const [cards, setCards] = useState<FamilyCard[]>([])
  const [warga, setWarga] = useState<ResidentProfile[]>([])
  const [kandidat, setKandidat] = useState<FamilyMember[]>([])
  const [err, setErr] = useState('')
  const [msg, setMsg] = useState('')
  const [busy, setBusy] = useState(false)

  const [blok, setBlok] = useState('')
  const [nomor, setNomor] = useState('')
  const [kkNumber, setKkNumber] = useState('')

  const [pilihRumah, setPilihRumah] = useState('')
  const [pilihWarga, setPilihWarga] = useState('')
  const [tipeHunian, setTipeHunian] = useState('OWNER_OCCUPANT')
  const [pembayarUtama, setPembayarUtama] = useState(true)
  const [pilihKK, setPilihKK] = useState('')

  async function muat() {
    try {
      const [u, c, w, k] = await Promise.all([
        houses.list(),
        familyCards.list(),
        census.queue('VERIFIED').catch(() => [] as ResidentProfile[]),
        familyCards.candidates(),
      ])
      setUnits(u)
      setCards(c)
      setWarga(w)
      setKandidat(k)
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) nav('/masuk')
      else setErr(e instanceof Error ? e.message : 'gagal memuat data')
    }
  }

  useEffect(() => {
    if (!auth.get()) {
      nav('/masuk')
      return
    }
    muat()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function jalan(fn: () => Promise<unknown>, pesan: string) {
    setBusy(true)
    setErr('')
    setMsg('')
    try {
      await fn()
      setMsg(pesan)
      await muat()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'gagal')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="min-h-screen bg-canvas text-ink">
      <header className="sticky top-0 z-10 border-b border-line bg-surface">
        <div className="mx-auto flex max-w-3xl items-center justify-between gap-4 px-4 py-3">
          <div>
            <p className="text-xs text-muted">Smarthub</p>
            <h1 className="font-semibold leading-tight">Rumah &amp; Kartu Keluarga</h1>
          </div>
          <button className="btn" onClick={() => nav('/dasbor')}>
            Dasbor
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-4 py-5 pb-16">
        <div className="flex gap-2">
          <button
            className={tab === 'rumah' ? 'btn-primary flex-1' : 'btn flex-1'}
            onClick={() => setTab('rumah')}
          >
            Rumah ({units.length})
          </button>
          <button className={tab === 'kk' ? 'btn-primary flex-1' : 'btn flex-1'} onClick={() => setTab('kk')}>
            Kartu Keluarga ({cards.length})
          </button>
        </div>

        {err && <p className="mt-4 rounded-xl bg-danger/10 px-4 py-3 text-sm text-danger">{err}</p>}
        {msg && <p className="mt-4 rounded-xl bg-ok/10 px-4 py-3 text-sm text-ok">{msg}</p>}

        {tab === 'rumah' && (
          <>
            <section className="card mt-5">
              <h2 className="font-semibold">Tambah rumah</h2>
              <form
                className="mt-3 flex flex-wrap items-end gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  jalan(() => houses.create({ block: blok, number: nomor }), 'Rumah ditambahkan.').then(() => {
                    setBlok('')
                    setNomor('')
                  })
                }}
              >
                <label className="field w-24">
                  <span className="text-xs text-muted">Blok</span>
                  <input value={blok} onChange={(e) => setBlok(e.target.value)} required maxLength={10} />
                </label>
                <label className="field w-24">
                  <span className="text-xs text-muted">Nomor</span>
                  <input value={nomor} onChange={(e) => setNomor(e.target.value)} required maxLength={10} />
                </label>
                <button className="btn-primary" disabled={busy}>
                  Tambah
                </button>
              </form>
            </section>

            <section className="card mt-5">
              <h2 className="font-semibold">Tempatkan warga</h2>
              <form
                className="mt-3 grid gap-3 sm:grid-cols-2"
                onSubmit={(e) => {
                  e.preventDefault()
                  jalan(
                    () =>
                      occupancy.assign(pilihWarga, {
                        house_unit_id: pilihRumah,
                        occupancy_type: tipeHunian,
                        is_primary_payer: pembayarUtama,
                      }),
                    'Warga ditempatkan.',
                  )
                }}
              >
                <label className="field">
                  <span className="text-xs text-muted">Rumah</span>
                  <select value={pilihRumah} onChange={(e) => setPilihRumah(e.target.value)} required>
                    <option value="">— pilih rumah —</option>
                    {units.map((u) => (
                      <option key={u.id} value={u.id}>
                        {u.block}-{u.unit_number} ({UNIT_STATUS_LABEL[u.occupancy_status] ?? u.occupancy_status})
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  <span className="text-xs text-muted">Warga</span>
                  <select value={pilihWarga} onChange={(e) => setPilihWarga(e.target.value)} required>
                    <option value="">— pilih warga —</option>
                    {warga.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.full_name} {w.house_unit ? `· ${w.house_unit}` : ''}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  <span className="text-xs text-muted">Jenis hunian</span>
                  <select value={tipeHunian} onChange={(e) => setTipeHunian(e.target.value)}>
                    <option value="OWNER_OCCUPANT">Pemilik yang tinggal</option>
                    <option value="TENANT">Penyewa</option>
                    <option value="OWNER_NON_RESIDENT">Pemilik tidak tinggal</option>
                  </select>
                </label>
                <label className="flex items-center gap-2 pt-5 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-[#0052FF]"
                    checked={pembayarUtama}
                    onChange={(e) => setPembayarUtama(e.target.checked)}
                  />
                  Penanggung jawab iuran
                </label>
                <button className="btn-primary sm:col-span-2" disabled={busy}>
                  Tempatkan
                </button>
              </form>
              {warga.length === 0 && (
                <p className="mt-2 text-xs text-muted">
                  Belum ada warga terverifikasi. Data muncul setelah sensus warga disetujui.
                </p>
              )}
            </section>

            <ul className="animate-list mt-5 space-y-3">
              {units.map((u) => (
                <li key={u.id} className="card">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-semibold">
                        {u.block}-{u.unit_number}
                      </p>
                      <p className="text-xs text-muted">{u.notes || 'tanpa catatan'}</p>
                    </div>
                    <span
                      className={
                        'badge ' +
                        (u.occupancy_status === 'OCCUPIED'
                          ? 'bg-ok/10 text-ok'
                          : u.occupancy_status === 'RENOVATION'
                            ? 'bg-warn/10 text-warn'
                            : 'bg-brand/10 text-brand')
                      }
                    >
                      {UNIT_STATUS_LABEL[u.occupancy_status] ?? u.occupancy_status}
                    </span>
                  </div>

                  <div className="mt-3 flex flex-wrap items-center gap-2">
                    <select
                      className="rounded-lg border border-line bg-surface px-2 py-1 text-sm"
                      value={u.occupancy_status}
                      onChange={(e) => jalan(() => houses.update(u.id, { status: e.target.value }), 'Status rumah diperbarui.')}
                    >
                      <option value="VACANT">Kosong</option>
                      <option value="OCCUPIED">Dihuni</option>
                      <option value="RENOVATION">Renovasi</option>
                    </select>
                    {u.primary_occupant && (
                      <button
                        className="btn text-xs"
                        onClick={() => {
                          const w = warga.find((x) => x.full_name === u.primary_occupant)
                          if (!w) {
                            setErr('Warga penghuni tidak ditemukan di daftar terverifikasi.')
                            return
                          }
                          jalan(() => occupancy.end(w.id), 'Hunian diakhiri.')
                        }}
                      >
                        Akhiri hunian
                      </button>
                    )}
                    <button
                      className="btn text-xs text-danger"
                      onClick={() => jalan(() => houses.remove(u.id), 'Rumah dihapus.')}
                    >
                      Hapus
                    </button>
                  </div>
                </li>
              ))}
              {units.length === 0 && <li className="card text-sm text-ink2">Belum ada rumah terdaftar.</li>}
            </ul>
          </>
        )}

        {tab === 'kk' && (
          <>
            <section className="card mt-5">
              <h2 className="font-semibold">Kartu keluarga baru</h2>
              <form
                className="mt-3 flex flex-wrap items-end gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  jalan(() => familyCards.create(kkNumber), 'Kartu keluarga dibuat.').then(() => setKkNumber(''))
                }}
              >
                <label className="field flex-1">
                  <span className="text-xs text-muted">Nomor KK (16 digit, opsional)</span>
                  <input
                    value={kkNumber}
                    onChange={(e) => setKkNumber(e.target.value)}
                    inputMode="numeric"
                    maxLength={16}
                    placeholder="3276xxxxxxxxxxxx"
                  />
                </label>
                <button className="btn-primary" disabled={busy}>
                  Buat
                </button>
              </form>
              <p className="mt-2 text-xs text-muted">
                Nomor KK disimpan terenkripsi; daftar hanya menampilkan 4 digit terakhir.
              </p>
            </section>

            <ul className="animate-list mt-5 space-y-3">
              {cards.map((c) => (
                <li key={c.id} className="card">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-semibold">
                        KK ····{c.number_last4 || '????'}
                      </p>
                      <p className="text-xs text-muted">
                        {c.member_count} anggota · berkas KK {c.has_file ? 'terunggah' : 'belum ada'}
                      </p>
                    </div>
                  </div>

                  <ul className="mt-3 space-y-2">
                    {c.members.map((m) => (
                      <li key={m.id} className="flex items-center justify-between gap-2 rounded-lg bg-subtle px-3 py-2">
                        <div>
                          <p className="text-sm font-medium">{m.full_name}</p>
                          <p className="text-xs text-muted">
                            {roleLabelCensus[m.family_role] ?? m.family_role} · {m.verification_status}
                          </p>
                        </div>
                        <button
                          className="btn text-xs"
                          onClick={() => jalan(() => familyCards.removeMember(c.id, m.id), 'Anggota dikeluarkan.')}
                        >
                          Keluarkan
                        </button>
                      </li>
                    ))}
                    {c.members.length === 0 && <li className="text-xs text-muted">Belum ada anggota.</li>}
                  </ul>

                  <form
                    className="mt-3 flex flex-wrap items-end gap-2"
                    onSubmit={(e) => {
                      e.preventDefault()
                      if (!pilihKK) {
                        setErr('Pilih warga dulu.')
                        return
                      }
                      jalan(() => familyCards.addMember(c.id, pilihKK), 'Anggota ditambahkan.').then(() => setPilihKK(''))
                    }}
                  >
                    <label className="field flex-1">
                      <span className="text-xs text-muted">Tambah anggota (belum punya KK)</span>
                      <select value={pilihKK} onChange={(e) => setPilihKK(e.target.value)}>
                        <option value="">— pilih warga —</option>
                        {kandidat.map((m) => (
                          <option key={m.id} value={m.id}>
                            {m.full_name} · {roleLabelCensus[m.family_role] ?? m.family_role}
                          </option>
                        ))}
                      </select>
                    </label>
                    <button className="btn" disabled={busy || kandidat.length === 0}>
                      Tambah
                    </button>
                  </form>
                </li>
              ))}
              {cards.length === 0 && <li className="card text-sm text-ink2">Belum ada kartu keluarga.</li>}
            </ul>
          </>
        )}
      </main>
    </div>
  )
}
