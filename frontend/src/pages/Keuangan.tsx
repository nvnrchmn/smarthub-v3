import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

// Keuangan Kas — Linear.app style
// Backend /api/finance/* belum siap → data sementara (mock).

type Tx = {
  id: string
  date: string
  desc: string
  amount: number
}

// TODO: ganti ke api.finance.list() saat backend ready.
async function fetchFinance(): Promise<Tx[]> {
  return [
    { id: '1', date: '12 Sep', desc: 'Pembayaran iuran warga', amount: 5000000 },
    { id: '2', date: '10 Sep', desc: 'Listrik gedung', amount: -850000 },
    { id: '3', date: '08 Sep', desc: 'Pembayaran air', amount: -320000 },
  ]
}

export default function Keuangan() {
  const [txs, setTxs] = useState<Tx[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')

  useEffect(() => {
    void fetchFinance()
      .then(setTxs)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Gagal memuat keuangan'))
      .finally(() => setLoading(false))
  }, [])

  const total = txs.reduce((sum, t) => sum + t.amount, 0)

  return (
    <div className="min-h-screen bg-[#0f0f14] text-slate-100 font-sans selection:bg-indigo-500/30">
      <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-[#0f0f14]/80 backdrop-blur-xl">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-4">
          <h1 className="text-lg font-bold">Keuangan</h1>
          <Link to="/dashboard" className="text-sm text-slate-400 hover:text-slate-200">
            ← Dashboard
          </Link>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-5 py-8">
        <div className="mb-6 grid grid-cols-2 gap-4 md:grid-cols-4">
          <div className="rounded-lg border border-white/[0.05] bg-white/[0.03] p-4">
            <p className="text-xs text-slate-500">Total Kas</p>
            <p className="mt-1 text-xl font-bold text-indigo-400">
              {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(total)}
            </p>
          </div>
          <div className="rounded-lg border border-white/[0.05] bg-white/[0.03] p-4">
            <p className="text-xs text-slate-500">Pemasukan</p>
            <p className="mt-1 text-xl font-bold text-emerald-400">
              {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(
                txs.filter((t) => t.amount > 0).reduce((s, t) => s + t.amount, 0)
              )}
            </p>
          </div>
          <div className="rounded-lg border border-white/[0.05] bg-white/[0.03] p-4">
            <p className="text-xs text-slate-500">Pengeluaran</p>
            <p className="mt-1 text-xl font-bold text-rose-400">
              {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(
                Math.abs(txs.filter((t) => t.amount < 0).reduce((s, t) => s + t.amount, 0))
              )}
            </p>
          </div>
          <div className="rounded-lg border border-white/[0.05] bg-white/[0.03] p-4">
            <p className="text-xs text-slate-500">Transaksi</p>
            <p className="mt-1 text-xl font-bold text-slate-200">{txs.length}</p>
          </div>
        </div>

        {err && (
          <p className="mb-4 rounded-lg bg-rose-500/10 border border-rose-500/30 p-3 text-sm text-rose-400">
            {err}
          </p>
        )}

        {loading ? (
          <div className="animate-pulse space-y-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="h-12 bg-slate-800/50 rounded-lg" />
            ))}
          </div>
        ) : txs.length === 0 ? (
          <p className="text-sm text-slate-500">Belum ada transaksi kas.</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-white/[0.05]">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b border-white/[0.05]">
                  <th className="pb-3 text-left font-medium text-slate-400">Tanggal</th>
                  <th className="pb-3 text-left font-medium text-slate-400">Deskripsi</th>
                  <th className="pb-3 text-right font-medium text-slate-400">Nominal</th>
                </tr>
              </thead>
              <tbody>
                {txs.map((t) => (
                  <tr
                    key={t.id}
                    className="border-b border-white/[0.03] transition-colors hover:bg-white/[0.03]"
                  >
                    <td className="py-3 text-slate-300">{t.date}</td>
                    <td className="py-3 text-slate-200">{t.desc}</td>
                    <td
                      className={`py-3 text-right font-medium ${
                        t.amount > 0 ? 'text-emerald-400' : 'text-rose-400'
                      }`}
                    >
                      {t.amount > 0 ? '+' : ''}{' '}
                      {new Intl.NumberFormat('id-ID', {
                        style: 'currency',
                        currency: 'IDR',
                        minimumFractionDigits: 0,
                      }).format(t.amount)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  )
}
