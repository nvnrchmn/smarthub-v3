import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, auth, roleLabel, rupiah, type Me, billing, type Invoice } from '../lib/api'

const STATUS_COLOR: Record<string, string> = {
  PAID: 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30',
  OVERDUE: 'bg-rose-500/15 text-rose-400 border border-rose-500/30',
  UNPAID: 'bg-amber-500/15 text-amber-400 border border-amber-500/30',
}
const DEFAULT_STATUS_COLOR = 'bg-slate-500/15 text-slate-400 border border-slate-500/30'

function statusBadge(status: string) {
  const cls = STATUS_COLOR[status] ?? DEFAULT_STATUS_COLOR
  const label = status === 'PAID' ? 'Lunas' : status === 'OVERDUE' ? 'Terlambat' : status === 'UNPAID' || status === 'DRAFT' ? 'Belum bayar' : status
  return <span className={`inline-flex items-center rounded-md px-2.5 py-0.5 text-xs font-medium ${cls}`}>{label}</span>
}

export default function Dashboard() {
  const [me, setMe] = useState<Me | null>(null)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const nav = useNavigate()

  useEffect(() => {
    void auth.get()
    if (!auth.get()) {
      auth.clear()
      nav('/masuk')
      return
    }
    api
      .me()
      .then(setMe)
      .catch(() => {
        auth.clear()
        nav('/masuk')
      })
    void billing
      .invoices('')
      .then((r) => setInvoices(r.items))
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Gagal memuat tagihan'))
      .finally(() => setLoading(false))
  }, [nav])

  if (loading || !me) return <div className="min-h-screen bg-[#0f0f14] flex items-center justify-center"><span className="text-slate-400 text-sm">Memuat…</span></div>

  const total = invoices.reduce((s, i) => s + i.total_amount, 0)
  const unpaid = invoices.filter((i) => i.status !== 'PAID')
  const belumBayar = unpaid.filter((i) => i.status === 'UNPAID' || i.status === 'DRAFT').length
  const terlambat = unpaid.filter((i) => i.status === 'OVERDUE').length
  const isPengurus = me.roles.includes('TENANT_MANAGER') || me.roles.includes('SECRETARY') || me.roles.includes('TREASURER')

  return (
    <div className="min-h-screen bg-[#0f0f14] text-slate-100 font-sans selection:bg-indigo-500/30">
      <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-[#0f0f14]/80 backdrop-blur-xl">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-5 py-4">
          <Link to="/" className="font-bold text-slate-100"><span className="text-indigo-400">Smarthub</span></Link>
          <button
            onClick={() => void auth.logout().then(() => nav('/masuk'))}
            className="rounded-lg border border-white/[0.08] px-3 py-1.5 text-xs font-semibold text-slate-300 hover:bg-white/[0.05]"
          >
            Keluar
          </button>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-5 py-8">
        <div className="mb-6">
          <h1 className="text-xl font-bold text-slate-100">Dasbor</h1>
          <p className="mt-1 text-sm text-slate-400">
            {roleLabel[me.roles[0] as keyof typeof roleLabel] ?? me.roles.join(', ')}
          </p>
        </div>
        <div className="mb-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <SummaryCard title="Tagihan Aktif" value={invoices.length} subtitle="invoice bulan ini" icon="📄" />
          <SummaryCard title="Total Tagihan" value={rupiah(total)} subtitle={`${belumBayar} belum dibayar`} icon="💰" />
          <SummaryCard title="Harus Bayar" value={terlambat} subtitle="jatuh tempo lewat" icon="⚠️" danger />
          <SummaryCard title="Saldo Kas" value={rupiah(0)} subtitle="terhubung ke kas" icon="🏦" />
        </div>
        {err && <p className="mb-4 text-sm text-rose-400">{err}</p>}
        <section>
          <h2 className="mb-3 text-sm font-semibold text-slate-300">Tagihan Terbaru</h2>
          {loading ? (
            <div className="animate-pulse space-y-2">
              <div className="h-12 bg-slate-800/50 rounded-lg"></div>
              <div className="h-12 bg-slate-800/50 rounded-lg"></div>
              <div className="h-12 bg-slate-800/50 rounded-lg"></div>
            </div>
          ) : invoices.length === 0 ? (
            <p className="text-sm text-slate-500">Belum ada tagihan.</p>
          ) : (
            <div className="space-y-1.5">
              {invoices.slice(0, 8).map((inv) => (
                <InvoiceRow key={inv.id} inv={inv} />
              ))}
            </div>
          )}
        </section>
        {isPengurus && (
          <section className="mt-6 grid gap-3 sm:grid-cols-3">
            <ActionButton label="Verifikasi sensus" icon="👥" onClick={() => nav('/verifikasi')} />
            <ActionButton label="Kelola rumah" icon="🏠" onClick={() => nav('/rumah')} />
            <ActionButton label="Keuangan" icon="💸" onClick={() => nav('/keuangan')} />
          </section>
        )}
      </main>
    </div>
  )
}

function SummaryCard({ title, value, subtitle, icon, danger }: {
  title: string; value: string | number; subtitle?: string; icon?: string; danger?: boolean
}) {
  return (
    <div className="rounded-xl border border-white/[0.06] bg-[#16161d] p-4 transition-colors hover:border-white/[0.1]">
      <div className="flex items-center gap-3">
        <span className="text-xl">{icon}</span>
        <div>
          <p className={`text-2xl font-semibold ${danger ? 'text-rose-400' : 'text-slate-100'}`}>{value}</p>
          <p className="text-xs text-slate-500">{title}</p>
        </div>
      </div>
      {subtitle && <p className="mt-2 text-xs text-slate-500">{subtitle}</p>}
    </div>
  )
}

function InvoiceRow({ inv }: { inv: Invoice }) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-white/[0.06] bg-[#16161d] px-4 py-3">
      <div className="min-w-0">
        <p className="text-sm font-medium text-slate-200 truncate">{inv.invoice_number}</p>
        <p className="text-xs text-slate-500">
          {inv.house_unit} · {new Date(inv.due_date).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' })}
        </p>
      </div>
      <div className="ml-3 text-right shrink-0">
        <p className="text-sm font-medium text-slate-100">{rupiah(inv.total_amount)}</p>
        <div className="mt-1">{statusBadge(inv.status)}</div>
      </div>
    </div>
  )
}

function ActionButton({ label, icon, onClick }: { label: string; icon: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="flex items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-[#16161d] px-4 py-3 text-sm font-medium text-slate-200 hover:border-white/[0.15] hover:bg-[#1a1a22]"
    >
      <span>{icon}</span>
      {label}
    </button>
  )
}
