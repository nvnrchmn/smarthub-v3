import { useEffect, useState } from 'react'
import { superadmin, type SuperadminMe, type TenantList } from '../lib/api'

export default function SuperadminDasbor() {
  const [me, setMe] = useState<SuperadminMe | null>(null)
  const [tenants, setTenants] = useState<TenantList[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')

  useEffect(() => {
    void superadmin
      .me()
      .then(setMe)
      .catch(() => (window.location.href = '/admin/login'))
    void superadmin.listTenants()
      .then((r: TenantList[]) => setTenants(r))
      .catch((e: unknown) =>
        setErr(e instanceof Error ? e.message : 'Gagal memuat tenant')
      )
      .finally(() => setLoading(false))
  }, [])

  const totalTenants = tenants.length
  const activeTenants = tenants.filter((t) => t.status === 'active').length
  const suspendedTenants = totalTenants - activeTenants

  if (loading || !me)
    return (
      <div className="min-h-screen bg-[#0f0f14] flex items-center justify-center">
        <span className="text-slate-400 text-sm">Memuat…</span>
      </div>
    )

  return (
    <div className="min-h-screen bg-[#0f0f14] text-slate-100 font-sans selection:bg-indigo-500/30">
      <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-[#0f0f14]/80 backdrop-blur-xl">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-5 py-4">
          <h1 className="text-lg font-bold">Superadmin — Smarthub</h1>
          <button
            onClick={() => {
              void localStorage.removeItem('smarthub.superadmin.token')
              window.location.href = '/admin/login'
            }}
            className="rounded-lg border border-white/[0.08] px-3 py-1.5 text-xs font-semibold text-slate-300 hover:bg-white/[0.05]"
          >
            Keluar
          </button>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-5 py-8">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold text-slate-100">Ringkasan Tenant</h2>
            <p className="mt-1 text-sm text-slate-400">
              Total: {totalTenants} | Aktif: {activeTenants} | Suspended:{' '}
              {suspendedTenants}
            </p>
          </div>
          <button
            onClick={() => (window.location.href = '/admin/tenant/new')}
            className="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-indigo-600"
          >
            + Tenant Baru
          </button>
        </div>
        <div className="mb-8 grid gap-3 sm:grid-cols-3">
          <SummaryCard title="Aktif" value={activeTenants} icon="✅" />
          <SummaryCard title="Suspended" value={suspendedTenants} icon="⛔" />
          <SummaryCard title="Total" value={totalTenants} icon="🏢" />
        </div>
        {err && (
          <p className="mb-4 rounded-lg bg-rose-500/10 border border-rose-500/30 p-3 text-sm text-rose-400">
            {err}
          </p>
        )}
        <section>
          <h3 className="mb-3 text-sm font-semibold text-slate-300">
            Daftar Tenant
          </h3>
          {loading ? (
            <div className="animate-pulse space-y-2">
              <div className="h-12 bg-slate-800/50 rounded-lg"></div>
              <div className="h-12 bg-slate-800/50 rounded-lg"></div>
              <div className="h-12 bg-slate-800/50 rounded-lg"></div>
            </div>
          ) : tenants.length === 0 ? (
            <p className="text-sm text-slate-500">Belum ada tenant.</p>
          ) : (
            <div className="space-y-1.5">
              {tenants.map((t) => (
                <TenantRow key={t.id} t={t} />
              ))}
            </div>
          )}
        </section>
      </main>
    </div>
  )
}

function SummaryCard({
  title,
  value,
  icon,
}: {
  title: string
  value: string | number
  icon?: string
}) {
  return (
    <div className="rounded-xl border border-white/[0.06] bg-[#16161d] p-4 text-center transition-colors hover:border-white/[0.1]">
      <div className="mb-1 text-2xl">{icon}</div>
      <div className="text-2xl font-semibold text-slate-100">{value}</div>
      <p className="text-xs text-slate-500">{title}</p>
    </div>
  )
}

function TenantRow({ t }: { t: TenantList }) {
  const sc =
    t.status === 'active'
      ? 'text-emerald-400 bg-emerald-500/10'
      : 'text-rose-400 bg-rose-500/10'
  return (
    <div className="flex items-center justify-between rounded-lg border border-white/[0.06] bg-[#16161d] px-4 py-3 transition-colors hover:border-white/[0.1]">
      <div className="min-w-0">
        <p className="font-medium text-slate-200">{t.name}</p>
        <p className="text-xs text-slate-500">Slug: {t.slug}</p>
      </div>
      <div className="ml-3 flex items-center gap-2 shrink-0">
        <span
          className={`inline-flex rounded-md border px-2 py-0.5 text-xs ${sc} border-current/20`}
        >
          {t.status}
        </span>
        <span className="text-xs text-slate-500">
          {new Date(t.created_at).toLocaleDateString('id-ID')}
        </span>
      </div>
    </div>
  )
}
