import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { census, type ResidentProfile } from '../lib/api'

const STATUS_COLOR: Record<string, string> = {
  verified: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
  pending: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
  rejected: 'bg-rose-500/15 text-rose-400 border-rose-500/30',
}

function StatusBadge({ status }: { status: string }) {
  const base = 'inline-flex rounded-md border px-2.5 py-0.5 text-xs font-medium'
  const color = STATUS_COLOR[status] || STATUS_COLOR.pending
  return <span className={`${base} ${color}`}>{status}</span>
}

export default function Sensus() {
  const [residents, setResidents] = useState<ResidentProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [err, setErr] = useState('')

  useEffect(() => {
    void census.list()
      .then((r) => setResidents(r))
      .catch((e: unknown) =>
        setErr(e instanceof Error ? e.message : 'Gagal memuat sensus')
      )
      .finally(() => setLoading(false))
  }, [])

  const filtered = residents.filter(
    (r) =>
      r.full_name.toLowerCase().includes(search.toLowerCase()) ||
      (r.house_unit ?? '').includes(search)
  )

  return (
    <div className="min-h-screen bg-[#0f0f14] text-slate-100 font-sans selection:bg-indigo-500/30">
      <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-[#0f0f14]/80 backdrop-blur-xl">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-4">
          <h1 className="text-lg font-bold">Sensus Warga</h1>
          <Link to="/dashboard" className="text-sm text-slate-400 hover:text-slate-200">
            ← Dashboard
          </Link>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-5 py-8">
        <div className="mb-6">
          <input
            type="text"
            placeholder="Cari nama / NIK / unit ..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full max-w-md rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 transition-all"
          />
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
        ) : filtered.length === 0 ? (
          <p className="text-sm text-slate-500">Belum ada sensus hari ini.</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-white/[0.05]">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b border-white/[0.05]">
                  <th className="pb-3 text-left font-medium text-slate-400">Nama</th>
                  <th className="pb-3 text-left font-medium text-slate-400">Unit Rumah</th>
                  <th className="pb-3 text-left font-medium text-slate-400">Status</th>
                  <th className="pb-3 text-right font-medium text-slate-400">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((r) => (
                  <tr
                    key={r.id}
                    className="border-b border-white/[0.03] transition-colors hover:bg-white/[0.03]"
                  >
                    <td className="py-3 text-slate-200">
                      {r.full_name}{' '}
                      <span className="text-slate-500">
                        ({r.nik_last4 ?? '—'})
                      </span>
                    </td>
                    <td className="py-3 text-slate-300">
                      {r.house_unit ?? '-'}
                    </td>
                    <td className="py-3">
                      <StatusBadge status={r.verification_status} />
                    </td>
                    <td className="py-3 text-right">
                      <button
                        onClick={() => {}}
                        className="rounded-md border border-white/[0.08] px-2.5 py-1 text-xs text-slate-300 hover:bg-white/[0.05]"
                      >
                        Reset PW
                      </button>
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
