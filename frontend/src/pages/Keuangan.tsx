import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ApiError,
  auth,
  billing,
  rupiah,
  SASARAN_IURAN,
  STATUS_TAGIHAN,
  SUMBER_KAS,
  type CashLedger,
  type FeeItem,
  type Invoice,
  type SaldoKas,
} from '../lib/api'

const bolehKelola = (roles: string[]) => roles.some((r) => ['TENANT_MANAGER', 'TREASURER'].includes(r))
const ketua = (roles: string[]) => roles.includes('TENANT_MANAGER')

// Keuangan Bendahara: tagihan, master iuran, pengaturan, buku kas, setoran bank.
export default function Keuangan() {
  const nav = useNavigate()
  const [tab, setTab] = useState<'tagihan' | 'iuran' | 'pengaturan' | 'kas'>('tagihan')
  const [roles, setRoles] = useState<string[]>([])
  const [daftar, setDaftar] = useState<Invoice[]>([])
  const [iuran, setIuran] = useState<FeeItem[]>([])
  const [kas, setKas] = useState<CashLedger[]>([])
  const [saldo, setSaldo] = useState<SaldoKas | null>(null)
  const [pesan, setPesan] = useState('')
  const [galat, setGalat] = useState('')
  const [sibuk, setSibuk] = useState(false)
  const [periode, setPeriode] = useState(() => new Date().toISOString().slice(0, 7))
  const [form, setForm] = useState({ code: '', label: '', amount: '', applies_to: 'ALL', is_active: true })
  const [hari, setHari] = useState('1')
  const [tempo, setTempo] = useState('14')
  const [setor, setSetor] = useState({ nominal: '', catatan: '' })

  useEffect(() => {
    if (!auth.get()) {
      nav('/masuk')
      return
    }
    ;(async () => {
      try {
        const me = await fetch('/api/me', { headers: { Authorization: `Bearer ${auth.get()}` } }).then((r) => r.json())
        setRoles(me.roles || [])
        if (!bolehKelola(me.roles || [])) {
          setGalat('Halaman ini untuk Bendahara dan Ketua.')
          return
        }
        await muat(true)
      } catch {
        setGalat('gagal memuat data keuangan')
      }
    })()
  }, [nav])

  async function muat(awal = false) {
    const r = await billing.invoices()
    setDaftar(r.items)
    setIuran(await billing.feeItems())
    const k = await billing.kas()
    setKas(k.items)
    setSaldo(k.saldo)
    if (awal) {
      const p = await billing.pengaturan()
      setHari(String(p.billing_day))
      setTempo(String(p.due_days))
    }
  }

  async function terbitkan() {
    setSibuk(true)
    setPesan('')
    setGalat('')
    try {
      const h = await billing.generate(periode)
      setPesan(`Periode ${h.periode}: ${h.dibuat} tagihan dibuat, ${h.dilewati} dilewati (dari ${h.total_unit} rumah).`)
      await muat()
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'gagal menerbitkan tagihan')
    } finally {
      setSibuk(false)
    }
  }

  async function bayarTunai(inv: Invoice) {
    const jawab = window.prompt(`Nominal uang tunai yang diterima untuk rumah ${inv.house_unit}?`, String(inv.total_amount))
    if (!jawab) return
    setSibuk(true)
    setPesan('')
    setGalat('')
    try {
      await billing.tunai(inv.id, Number(jawab), 'penerimaan tunai')
      setPesan(`Kas tunai dicatat untuk ${inv.invoice_number}. Kuitansi digital dikirim ke WhatsApp penghuni.`)
      await muat()
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'gagal mencatat pembayaran tunai')
    } finally {
      setSibuk(false)
    }
  }

  async function tambahIuran() {
    setGalat('')
    setPesan('')
    try {
      await billing.simpanIuran({
        code: form.code.toUpperCase(),
        label: form.label,
        amount: Number(form.amount),
        applies_to: form.applies_to,
        is_active: form.is_active,
      })
      setForm({ code: '', label: '', amount: '', applies_to: 'ALL', is_active: true })
      setPesan('Komponen iuran disimpan.')
      setIuran(await billing.feeItems())
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'gagal menyimpan iuran')
    }
  }

  async function simpanPengaturan() {
    setGalat('')
    setPesan('')
    try {
      await billing.simpanPengaturan(Number(hari), Number(tempo))
      setPesan('Pengaturan penagihan disimpan.')
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'gagal menyimpan pengaturan')
    }
  }

  async function kirimSetor() {
    setGalat('')
    setPesan('')
    try {
      await billing.setor(Number(setor.nominal), setor.catatan)
      setSetor({ nominal: '', catatan: '' })
      setPesan('Setoran dicatat, menunggu persetujuan Ketua.')
      await muat()
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'gagal mencatat setoran')
    }
  }

  async function setujui(grup: string) {
    setGalat('')
    setPesan('')
    try {
      await billing.setujuiSetor(grup)
      setPesan('Setoran disetujui.')
      await muat()
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'gagal menyetujui setoran')
    }
  }

  return (
    <div className="min-h-screen bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface/95 backdrop-blur">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
          <button onClick={() => nav('/dasbor')} className="text-sm text-ink2 transition-colors duration-240 hover:text-brand">
            ← Dasbor
          </button>
          <p className="font-semibold text-ink">Keuangan</p>
        </div>
        <div className="mx-auto flex max-w-4xl gap-2 overflow-x-auto px-4 pb-3">
          {(() => {
            const tabs: Array<[typeof tab, string]> = [
              ['tagihan', 'Tagihan'],
              ['iuran', 'Master iuran'],
              ['pengaturan', 'Pengaturan'],
              ['kas', 'Buku kas'],
            ]
            return tabs.map(([k, label]) => (
              <button
                key={k}
                onClick={() => setTab(k)}
                className={
                  'whitespace-nowrap rounded-full px-4 py-2 text-sm transition-all duration-400 ' +
                  (tab === k ? 'bg-brand text-white shadow-lg shadow-brand/25' : 'bg-surface text-ink2 border border-line hover:border-brand/40')
                }
              >
                {label}
              </button>
            ))
          })()}
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-6">
        {pesan && <p className="mb-3 rounded-xl border border-ok/30 bg-ok/10 p-3 text-sm text-ok">{pesan}</p>}
        {galat && <p className="mb-3 rounded-xl border border-danger/30 bg-danger/10 p-3 text-sm text-danger">{galat}</p>}

        {tab === 'tagihan' && (
          <>
            <section className="card">
              <h2 className="font-semibold text-ink">Terbitkan tagihan bulanan</h2>
              <p className="mt-1 text-xs text-ink2">
                Rumah kosong hanya dikenai komponen dasar; tunggakan periode sebelumnya ikut ditagihkan.
              </p>
              <div className="mt-3 flex gap-2">
                <input value={periode} onChange={(e) => setPeriode(e.target.value)} placeholder="2026-09" className="field flex-1" />
                <button onClick={terbitkan} disabled={sibuk} className="btn-primary disabled:opacity-60">
                  {sibuk ? 'Memproses…' : 'Terbitkan'}
                </button>
              </div>
            </section>

            <div className="animate-list mt-4 space-y-3">
              {daftar.map((inv) => (
                <section key={inv.id} className="card">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-semibold text-ink">Rumah {inv.house_unit || '—'} · {inv.period}</p>
                      <p className="text-xs text-ink2">{inv.invoice_number} · jatuh tempo {inv.due_date}</p>
                      <p className="mt-1 text-[11px] text-muted">
                        {inv.items?.map((i) => `${i.label} ${rupiah(i.amount)}`).join(' · ')}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="font-bold text-ink">{rupiah(inv.total_amount)}</p>
                      <span className={'mt-1 inline-block rounded-badge px-2 py-1 text-[11px] ' + (inv.status === 'PAID' ? 'bg-ok/10 text-ok' : 'bg-warn/10 text-warn')}>
                        {STATUS_TAGIHAN[inv.status] || inv.status}
                      </span>
                    </div>
                  </div>
                  {inv.status === 'UNPAID' && (
                    <button onClick={() => bayarTunai(inv)} className="btn-secondary mt-3 w-full sm:w-auto">
                      Catat bayar tunai
                    </button>
                  )}
                  {inv.status === 'PAID' && inv.paid_at && (
                    <p className="mt-2 text-[11px] text-ok">Lunas pada {inv.paid_at}</p>
                  )}
                </section>
              ))}
              {daftar.length === 0 && (
                <section className="card">
                  <p className="text-sm text-ink2">Belum ada tagihan pada tenant ini.</p>
                </section>
              )}
            </div>
          </>
        )}

        {tab === 'iuran' && (
          <>
            <section className="card">
              <h2 className="font-semibold text-ink">Komponen iuran</h2>
              <div className="animate-list mt-3 space-y-2">
                {iuran.map((f) => (
                  <div key={f.id} className="flex items-center justify-between rounded-xl border border-line bg-canvas p-3">
                    <div>
                      <p className="text-sm font-medium text-ink">{f.label}</p>
                      <p className="text-[11px] text-muted">
                        {f.code} · {SASARAN_IURAN[f.applies_to] || f.applies_to} {f.is_active ? '' : '· nonaktif'}
                      </p>
                    </div>
                    <p className="text-sm font-semibold text-ink">{rupiah(f.amount)}</p>
                  </div>
                ))}
                {iuran.length === 0 && <p className="text-sm text-ink2">Belum ada komponen iuran.</p>}
              </div>
            </section>

            <section className="card mt-4">
              <h2 className="font-semibold text-ink">Tambah / ubah iuran</h2>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                <input value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} placeholder="Kode (IPL)" className="field" />
                <input value={form.label} onChange={(e) => setForm({ ...form, label: e.target.value })} placeholder="Nama iuran" className="field" />
                <input value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} inputMode="numeric" placeholder="Nominal (150000)" className="field" />
                <select value={form.applies_to} onChange={(e) => setForm({ ...form, applies_to: e.target.value })} className="field">
                  {Object.entries(SASARAN_IURAN).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </select>
              </div>
              <label className="mt-2 flex items-center gap-2 text-xs text-ink2">
                <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
                Aktif (ikut ditagihkan)
              </label>
              <button onClick={tambahIuran} className="btn-primary mt-3 w-full sm:w-auto">Simpan iuran</button>
            </section>
          </>
        )}

        {tab === 'pengaturan' && (
          <section className="card">
            <h2 className="font-semibold text-ink">Jadwal penagihan</h2>
            <p className="mt-1 text-xs text-ink2">
              Tagihan terbit otomatis pada tanggal ini setiap bulan (cron harian 06:10).
            </p>
            <div className="mt-3 grid gap-2 sm:grid-cols-2">
              <label className="text-xs text-ink2">
                Tanggal terbit (1–28)
                <input value={hari} onChange={(e) => setHari(e.target.value)} inputMode="numeric" className="field mt-1" />
              </label>
              <label className="text-xs text-ink2">
                Jatuh tempo (hari setelah terbit)
                <input value={tempo} onChange={(e) => setTempo(e.target.value)} inputMode="numeric" className="field mt-1" />
              </label>
            </div>
            <button onClick={simpanPengaturan} className="btn-primary mt-3 w-full sm:w-auto">Simpan pengaturan</button>
          </section>
        )}

        {tab === 'kas' && (
          <>
            <div className="grid gap-3 sm:grid-cols-3">
              {([
                ['bank_gateway', 'Kas Bank / Gateway'],
                ['petty_cash', 'Kas Fisik Bendahara'],
                ['bank_account', 'Rekening Paguyuban'],
              ] as Array<[keyof SaldoKas, string]>).map(([k, label]) => (
                <section key={k} className="card">
                  <p className="text-xs text-muted">{label}</p>
                  <p className="mt-1 text-lg font-bold text-ink">{rupiah(Number(saldo?.[k] || 0))}</p>
                </section>
              ))}
            </div>

            <section className="card mt-4">
              <h2 className="font-semibold text-ink">Setor kas fisik ke bank</h2>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                <input value={setor.nominal} onChange={(e) => setSetor({ ...setor, nominal: e.target.value })} inputMode="numeric" placeholder="Nominal setoran" className="field" />
                <input value={setor.catatan} onChange={(e) => setSetor({ ...setor, catatan: e.target.value })} placeholder="Catatan / nomor slip" className="field" />
              </div>
              <button onClick={kirimSetor} className="btn-primary mt-3 w-full sm:w-auto">Catat setoran</button>
              {(saldo?.menunggu_persetujuan || 0) > 0 && (
                <p className="mt-2 text-[11px] text-warn">{saldo?.menunggu_persetujuan} mutasi menunggu persetujuan Ketua.</p>
              )}
            </section>

            <section className="mt-4">
              <h2 className="font-semibold text-ink">Mutasi terakhir</h2>
              <div className="animate-list mt-3 space-y-2">
                {kas.map((k) => (
                  <div key={k.id} className="card flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm text-ink">
                        {SUMBER_KAS[k.source_type] || k.source_type} · {k.transaction_type === 'IN' ? 'masuk' : 'keluar'}
                      </p>
                      <p className="text-[11px] text-muted">{k.ledger_date} {k.invoice_number ? `· ${k.invoice_number}` : ''} {k.notes ? `· ${k.notes}` : ''}</p>
                    </div>
                    <div className="text-right">
                      <p className={'text-sm font-semibold ' + (k.transaction_type === 'IN' ? 'text-ok' : 'text-danger')}>
                        {k.transaction_type === 'IN' ? '+' : '−'}{rupiah(k.amount)}
                      </p>
                      {k.transfer_group && !k.approved_at && ketua(roles) && (
                        <button onClick={() => setujui(k.transfer_group!)} className="mt-1 text-[11px] text-brand hover:underline">
                          Setujui
                        </button>
                      )}
                      {k.transfer_group && k.approved_at && <p className="text-[11px] text-ok">disetujui</p>}
                    </div>
                  </div>
                ))}
                {kas.length === 0 && <p className="text-sm text-ink2">Belum ada mutasi kas.</p>}
              </div>
            </section>
          </>
        )}
      </main>
    </div>
  )
}
