import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import QRCode from 'qrcode'
import { ApiError, auth, billing, rupiah, STATUS_TAGIHAN, type Invoice } from '../lib/api'

// Tagihan warga: daftar tagihan unit yang dihuni + bayar QRIS / info kas tunai.
export default function Tagihan() {
  const nav = useNavigate()
  const [daftar, setDaftar] = useState<Invoice[]>([])
  const [total, setTotal] = useState(0)
  const [pilih, setPilih] = useState<Invoice | null>(null)
  const [qr, setQr] = useState('')
  const [pesan, setPesan] = useState('')
  const [galat, setGalat] = useState('')
  const [sibuk, setSibuk] = useState(false)

  useEffect(() => {
    if (!auth.get()) {
      nav('/masuk')
      return
    }
    billing
      .invoices()
      .then((r) => {
        setDaftar(r.items)
        setTotal(r.total)
      })
      .catch((e) => {
        if (e instanceof ApiError && e.status === 401) nav('/masuk')
        else setGalat('gagal memuat tagihan')
      })
  }, [nav])

  async function muat() {
    const r = await billing.invoices()
    setDaftar(r.items)
    setTotal(r.total)
  }

  async function bukaQR(inv: Invoice) {
    setPilih(inv)
    setQr('')
    setPesan('')
    setGalat('')
    setSibuk(true)
    try {
      const r = await billing.qrisBuat(inv.id)
      const gambar = await QRCode.toDataURL(r.qr_string, { margin: 1, width: 320 })
      setQr(gambar)
      setPesan(`Berlaku sampai ${new Date(r.expires_at).toLocaleTimeString('id-ID')}. Setelah membayar, tekan "Cek status".`)
    } catch (e) {
      setGalat(e instanceof ApiError ? e.message : 'QRIS tidak bisa dibuat sekarang')
    } finally {
      setSibuk(false)
    }
  }

  async function cek() {
    if (!pilih) return
    setSibuk(true)
    try {
      const r = await billing.qrisCek(pilih.id)
      if (r.status_gateway === 'PAID') {
        setPesan('Pembayaran diterima. Kuitansi digital dikirim ke WhatsApp penghuni.')
        setQr('')
        await muat()
        setPilih(r.invoice)
      } else {
        setPesan(`Status di gateway: ${r.status_gateway}. Kalau baru membayar, tunggu sebentar lalu cek lagi.`)
      }
    } catch {
      setGalat('gagal memeriksa status ke gateway')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <div className="min-h-screen bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface/95 backdrop-blur transition-colors duration-400">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-4 py-3">
          <button onClick={() => nav('/dasbor')} className="text-sm text-ink2 transition-colors duration-240 hover:text-brand">
            ← Dasbor
          </button>
          <p className="font-semibold text-ink">Tagihan iuran</p>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-4 py-6">
        <section className="card">
          <p className="text-xs text-muted">Total belum lunas</p>
          <p className="mt-1 text-2xl font-bold text-ink">{rupiah(total)}</p>
        </section>

        {galat && <p className="mt-4 rounded-xl border border-danger/30 bg-danger/10 p-3 text-sm text-danger">{galat}</p>}

        <div className="animate-list mt-4 space-y-3">
          {daftar.map((inv) => (
            <button key={inv.id} onClick={() => (inv.status === 'UNPAID' ? bukaQR(inv) : setPilih(inv))} className="card w-full text-left">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-semibold text-ink">Rumah {inv.house_unit || '—'}</p>
                  <p className="text-xs text-ink2">Periode {inv.period} · {inv.invoice_number}</p>
                  <p className="mt-1 text-xs text-muted">Jatuh tempo {inv.due_date}</p>
                </div>
                <div className="text-right">
                  <p className="font-bold text-ink">{rupiah(inv.total_amount)}</p>
                  <span
                    className={
                      'mt-1 inline-block rounded-badge px-2 py-1 text-[11px] transition-colors duration-400 ' +
                      (inv.status === 'PAID' ? 'bg-ok/10 text-ok' : 'bg-warn/10 text-warn')
                    }
                  >
                    {STATUS_TAGIHAN[inv.status] || inv.status}
                  </span>
                </div>
              </div>
              {inv.status === 'UNPAID' && (
                <p className="mt-3 text-xs text-brand">Ketuk untuk bayar via QRIS →</p>
              )}
            </button>
          ))}
          {daftar.length === 0 && (
            <section className="card">
              <p className="font-semibold text-ink">Belum ada tagihan</p>
              <p className="mt-1 text-xs text-ink2">
                Tagihan terbit otomatis setiap bulan. Pastikan data rumahmu sudah diverifikasi sekretaris.
              </p>
            </section>
          )}
        </div>

        {(qr || (pilih && pilih.status === 'PAID')) && (
          <div className="fixed inset-0 z-20 flex items-end justify-center bg-ink/40 backdrop-blur-sm sm:items-center">
            <section className="card mx-4 mb-4 w-full max-w-md transition-all duration-800">
              <div className="flex items-start justify-between">
                <div>
                  <p className="font-semibold text-ink">Rumah {pilih?.house_unit}</p>
                  <p className="text-xs text-ink2">Periode {pilih?.period}</p>
                </div>
                <button onClick={() => { setPilih(null); setQr(''); setPesan('') }} className="text-sm text-ink2 hover:text-brand">
                  Tutup
                </button>
              </div>
              <p className="mt-2 text-xl font-bold text-ink">{rupiah(pilih?.total_amount || 0)}</p>

              {qr && (
                <div className="mt-4 text-center">
                  <img src={qr} alt="QRIS" className="mx-auto h-56 w-56 rounded-xl border border-line bg-white p-2" />
                  <p className="mt-2 text-xs text-muted">Scan dengan aplikasi bank / e-wallet apa pun (QRIS)</p>
                </div>
              )}

              {pesan && <p className="mt-3 rounded-xl border border-line bg-canvas p-3 text-xs text-ink2">{pesan}</p>}
              {galat && <p className="mt-3 rounded-xl border border-danger/30 bg-danger/10 p-3 text-xs text-danger">{galat}</p>}

              {qr && (
                <button onClick={cek} disabled={sibuk} className="btn-primary mt-4 w-full disabled:opacity-60">
                  {sibuk ? 'Memeriksa…' : 'Cek status pembayaran'}
                </button>
              )}
              <p className="mt-3 text-[11px] text-muted">
                Belum bisa QRIS? Setor tunai ke Bendahara, kuitansi digital akan dikirim ke WhatsApp-mu.
              </p>
            </section>
          </div>
        )}
      </main>
    </div>
  )
}
