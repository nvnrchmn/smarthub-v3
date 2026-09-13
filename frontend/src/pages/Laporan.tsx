import { useCallback, useEffect, useState } from 'react'
import { laporan, type RekapBulanan, type Tunggakan } from '../lib/api'

const rp = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 })

// Laporan — rekap bulanan + tunggakan, unduh PDF/CSV, kirim ke pengurus.
export default function Laporan() {
  const [periode, setPeriode] = useState(() => new Date().toISOString().slice(0, 7))
  const [rek, setRek] = useState<RekapBulanan | null>(null)
  const [tun, setTun] = useState<Tunggakan | null>(null)
  const [pesan, setPesan] = useState('')
  const [sukses, setSukses] = useState('')
  const [sibuk, setSibuk] = useState(false)

  const muat = useCallback(async (p: string) => {
    setPesan('')
    try {
      const [r, t] = await Promise.all([laporan.rekap(p), laporan.tunggakan()])
      setRek(r)
      setTun(t)
    } catch (e) {
      setPesan(e instanceof Error ? e.message : 'gagal memuat laporan')
    }
  }, [])

  useEffect(() => {
    muat(periode)
  }, [periode, muat])

  async function unduh(format: 'pdf' | 'csv') {
    setPesan('')
    setSukses('')
    try {
      await laporan.unduh(periode, format)
    } catch (e) {
      setPesan(e instanceof Error ? e.message : 'gagal mengunduh')
    }
  }

  async function kirim() {
    setPesan('')
    setSukses('')
    setSibuk(true)
    try {
      const r = await laporan.kirim(periode)
      setSukses(`Ringkasan terkirim ke ${r.terkirim} pengurus via WhatsApp`)
    } catch (e) {
      setPesan(e instanceof Error ? e.message : 'gagal mengirim')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-ink">Laporan Iuran</h1>
          <p className="text-sm text-ink2">Rekap bulanan & daftar tunggakan perumahan.</p>
        </div>
        <input type="month" value={periode} onChange={(e) => setPeriode(e.target.value)} className="field w-auto" />
      </div>

      {pesan && <p className="text-sm text-danger">{pesan}</p>}
      {sukses && <p className="text-sm text-brand">{sukses}</p>}

      <div className="animate-list grid gap-3 sm:grid-cols-3">
        <div className="card">
          <p className="text-xs text-ink2">Ditagih periode ini</p>
          <p className="mt-1 text-lg font-semibold text-ink">{rp.format(rek?.total_ditagih ?? 0)}</p>
          <p className="text-xs text-ink2">{rek?.jumlah_invoice ?? 0} tagihan</p>
        </div>
        <div className="card">
          <p className="text-xs text-ink2">Sudah dibayar</p>
          <p className="mt-1 text-lg font-semibold text-ink">{rp.format(rek?.total_dibayar ?? 0)}</p>
          <p className="text-xs text-ink2">{rek?.jumlah_lunas ?? 0} unit lunas</p>
        </div>
        <div className="card">
          <p className="text-xs text-ink2">Tunggakan periode ini</p>
          <p className="mt-1 text-lg font-semibold text-ink">{rp.format(rek?.total_tunggakan ?? 0)}</p>
          <p className="text-xs text-ink2">{rek?.jumlah_belum ?? 0} unit belum bayar</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <button onClick={() => unduh('pdf')} className="btn-primary">
          Unduh PDF
        </button>
        <button onClick={() => unduh('csv')} className="btn-ghost">
          Unduh CSV (Excel)
        </button>
        <button onClick={kirim} disabled={sibuk} className="btn-ghost">
          {sibuk ? 'Mengirim...' : 'Kirim ke pengurus (WA)'}
        </button>
      </div>

      <section className="card">
        <h2 className="mb-3 font-semibold text-ink">Rincian iuran</h2>
        <div className="divide-y divide-line">
          {(rek?.rincian ?? []).map((it) => (
            <div key={it.label + it.kind} className="flex items-center justify-between py-2 text-sm">
              <span className="text-ink">
                {it.label} <span className="text-ink2">({it.jumlah} tagihan)</span>
              </span>
              <span className="text-ink2">{rp.format(it.amount)}</span>
            </div>
          ))}
          {(rek?.rincian ?? []).length === 0 && <p className="py-2 text-sm text-ink2">Belum ada tagihan pada periode ini.</p>}
        </div>
        <div className="mt-3 flex flex-wrap gap-4 text-xs text-ink2">
          <span>Kas tunai: {rp.format(rek?.kas_tunai ?? 0)}</span>
          <span>QRIS: {rp.format(rek?.kas_qris ?? 0)}</span>
        </div>
      </section>

      <section className="card">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-semibold text-ink">Tunggakan</h2>
          <span className="text-sm text-ink2">
            {rp.format(tun?.total ?? 0)} · {tun?.jumlah_unit ?? 0} unit
          </span>
        </div>
        <div className="mb-3 flex flex-wrap gap-2">
          {(tun?.buckets ?? []).map((b) => (
            <span key={b.label} className="badge border-line text-ink2">
              {b.label}: {b.jumlah} unit · {rp.format(b.amount)}
            </span>
          ))}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-ink2">
              <tr>
                <th className="py-2">Unit</th>
                <th>Kepala keluarga</th>
                <th>Sejak</th>
                <th>Hari</th>
                <th className="text-right">Total</th>
              </tr>
            </thead>
            <tbody>
              {(tun?.baris ?? []).map((b) => (
                <tr key={b.unit} className="border-t border-line">
                  <td className="py-2 text-ink">{b.unit}</td>
                  <td className="text-ink2">{b.kepala_keluarga || '-'}</td>
                  <td className="text-ink2">{b.periode_tertua}</td>
                  <td className="text-ink2">{b.hari_terlambat}</td>
                  <td className="text-right text-ink">{rp.format(b.total_tunggakan)}</td>
                </tr>
              ))}
              {(tun?.baris ?? []).length === 0 && (
                <tr>
                  <td colSpan={5} className="py-3 text-ink2">
                    Tidak ada tunggakan.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
