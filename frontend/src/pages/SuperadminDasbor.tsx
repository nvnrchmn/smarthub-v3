import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { superadmin, type AuditLogEntry, type TenantList } from '../lib/api'

function tgl(iso?: string) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export default function SuperadminDasbor() {
  const nav = useNavigate()
  const [tenants, setTenants] = useState<TenantList[]>([])
  const [audit, setAudit] = useState<AuditLogEntry[]>([])
  const [tab, setTab] = useState<'tenants' | 'audit' | 'pengaturan'>('tenants')
  const [pesan, setPesan] = useState('')
  const [sukses, setSukses] = useState('')
  const [xenditKey, setXenditKey] = useState('')
  const [xenditPrefix, setXenditPrefix] = useState('')
  const [oldPw, setOldPw] = useState('')
  const [newPw, setNewPw] = useState('')

  useEffect(() => {
    superadmin.me().catch(() => nav('/superadmin'))
    superadmin.listTenants().then(setTenants).catch((e: Error) => setPesan(e.message))
    superadmin.auditLog().then(setAudit).catch(() => setAudit([]))
    superadmin
      .allSettings()
      .then((s) => {
        setXenditKey(s.xendit_key ?? '')
        setXenditPrefix(s.xendit_prefix ?? '')
      })
      .catch(() => {})
  }, [nav])

  async function simpanXendit(e: React.FormEvent) {
    e.preventDefault()
    setPesan(''); setSukses('')
    try {
      await superadmin.setSetting('xendit_key', xenditKey)
      await superadmin.setSetting('xendit_prefix', xenditPrefix)
      setSukses('Kunci Xendit tersimpan')
    } catch (e: unknown) {
      setPesan(e instanceof Error ? e.message : 'gagal menyimpan')
    }
  }

  async function gantiSandi(e: React.FormEvent) {
    e.preventDefault()
    setPesan(''); setSukses('')
    if (newPw.length < 8) { setPesan('Kata sandi baru minimal 8 karakter'); return }
    try {
      await superadmin.resetPassword(oldPw, newPw)
      setSukses('Kata sandi superadmin diganti')
      setOldPw(''); setNewPw('')
    } catch (e: unknown) {
      setPesan(e instanceof Error ? e.message : 'gagal ganti kata sandi')
    }
  }

  function keluar() {
    superadmin.clear()
    nav('/superadmin')
  }

  return (
    <div className="min-h-screen bg-[#0a0a0f] text-slate-100 font-sans flex flex-col">
      <header className="sticky top-0 z-50 border-b border-white/10 bg-[#0f0f14]/80 backdrop-blur-xl px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-indigo-500 to-violet-500 flex items-center justify-center shadow-[0_0_15px_rgba(129,140,248,0.4)]">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
          </div>
          <div>
            <h1 className="text-sm font-bold tracking-tight">Superadmin Platform</h1>
            <p className="text-[10px] text-slate-400">Logikraf — manajemen tenant & audit</p>
          </div>
        </div>
        <button onClick={keluar} className="text-xs px-3 py-1.5 rounded-full border border-white/10 hover:bg-white/5 hover:border-white/20 transition-colors text-slate-300 hover:text-white">
          Keluar
        </button>
      </header>

      <main className="mx-auto max-w-4xl px-6 py-8 flex-1 space-y-8">
        {pesan && <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-4 py-3 text-sm text-red-300">{pesan}</div>}
        {sukses && <div className="rounded-lg bg-emerald-500/10 border border-emerald-500/20 px-4 py-3 text-sm text-emerald-300">{sukses}</div>}

        <div className="flex gap-2">
          {(['tenants', 'audit', 'pengaturan'] as const).map((t) => (
            <button key={t} onClick={() => setTab(t)}
              className={`px-4 py-2 rounded-full text-xs font-medium transition-all ${tab === t ? 'bg-white text-black shadow-[0_0_20px_rgba(255,255,255,0.15)]' : 'text-slate-400 hover:text-slate-200 hover:bg-white/5 border border-transparent hover:border-white/10'}`}>
              {t === 'tenants' ? `Tenant (${tenants.length})` : t === 'audit' ? `Audit (${audit.length})` : 'Pengaturan'}
            </button>
          ))}
        </div>

        {tab === 'tenants' && (
          <section>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {tenants.map((t) => (
                <div key={t.id} className="rounded-xl border border-white/10 bg-[#111118]/60 p-4 hover:bg-[#161626] transition-colors group">
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <h3 className="text-sm font-semibold text-white">{t.name || t.slug}</h3>
                      <p className="text-xs text-slate-400">{t.slug}</p>
                    </div>
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium ${t.status === 'active' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'}`}>
                      {t.status}
                    </span>
                  </div>
                  <div className="text-[10px] text-slate-500 space-y-0.5">
                    <p>ID: {t.id}</p>
                    <p>Kode: {t.id}</p>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}

        {tab === 'audit' && (
          <section>
            <table className="w-full text-xs border-collapse">
              <thead>
                <tr className="border-b border-white/10 text-slate-400">
                  <th className="text-left py-2.5 px-3 font-medium">Aksi</th>
                  <th className="text-left py-2.5 px-3 font-medium">Aktor</th>
                  <th className="text-left py-2.5 px-3 font-medium">Tenant</th>
                  <th className="text-left py-2.5 px-3 font-medium">Waktu</th>
                </tr>
              </thead>
              <tbody>
                {audit.map((a) => (
                  <tr key={a.id} className="border-b border-white/[0.05] hover:bg-white/[0.03] transition-colors">
                    <td className="py-2.5 px-3 font-medium text-white">{a.action}</td>
                    <td className="py-2.5 px-3 text-slate-400">{a.tenant_id}</td>
                    <td className="py-2.5 px-3 text-slate-500">{a.tenant_id}</td>
                    <td className="py-2.5 px-3 text-slate-500">{tgl(a.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        )}

        {tab === 'pengaturan' && (
          <section className="space-y-6">
            <form onSubmit={simpanXendit} className="rounded-xl border border-white/10 bg-[#111118]/40 p-6 space-y-4">
              <h2 className="text-sm font-bold text-white">Pengaturan Xendit</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <label className="block">
                  <span className="text-[10px] text-slate-400 mb-1 block">Secret Key</span>
                  <input type="password" value={xenditKey} onChange={e => setXenditKey(e.target.value)}
                    className="w-full rounded-lg bg-[#0f0f14] border border-white/10 px-3 py-2 text-xs text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500/50 transition-all" placeholder="sk_..." />
                </label>
                <label className="block">
                  <span className="text-[10px] text-slate-400 mb-1 block">Prefix</span>
                  <input type="text" value={xenditPrefix} onChange={e => setXenditPrefix(e.target.value)}
                    className="w-full rounded-lg bg-[#0f0f14] border border-white/10 px-3 py-2 text-xs text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500/50 transition-all" placeholder="mg-..." />
                </label>
              </div>
              <button type="submit" className="inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-indigo-600 to-violet-600 px-4 py-2 text-xs font-semibold text-white hover:brightness-110 transition-all shadow-[0_0_20px_rgba(129,140,248,0.25)]">
                Simpan Kunci
              </button>
            </form>

            <form onSubmit={gantiSandi} className="rounded-xl border border-white/10 bg-[#111118]/40 p-6 space-y-4">
              <h2 className="text-sm font-bold text-white">Ganti Sandi Superadmin</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <label className="block">
                  <span className="text-[10px] text-slate-400 mb-1 block">Sandi Lama</span>
                  <input type="password" value={oldPw} onChange={e => setOldPw(e.target.value)}
                    className="w-full rounded-lg bg-[#0f0f14] border border-white/10 px-3 py-2 text-xs text-white focus:outline-none focus:ring-2 focus:ring-amber-500/30 transition-all" />
                </label>
                <label className="block">
                  <span className="text-[10px] text-slate-400 mb-1 block">Sandi Baru</span>
                  <input type="password" value={newPw} onChange={e => setNewPw(e.target.value)}
                    className="w-full rounded-lg bg-[#0f0f14] border border-white/10 px-3 py-2 text-xs text-white focus:outline-none focus:ring-2 focus:ring-amber-500/30 transition-all" />
                </label>
              </div>
              <button type="submit" className="inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-amber-600 to-orange-600 px-4 py-2 text-xs font-semibold text-white hover:brightness-110 transition-all shadow-[0_0_20px_rgba(245,158,11,0.25)]">
                Ganti Sandi
              </button>
            </form>
          </section>
        )}
      </main>
    </div>
  )
}
