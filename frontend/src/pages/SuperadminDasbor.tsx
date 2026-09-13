import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { superadmin, type AuditLogEntry, type TenantList } from '../lib/api'

export default function SuperadminDasbor() {
  const nav = useNavigate()
  const [tenants, setTenants] = useState<TenantList[]>([])
  const [audit, setAudit] = useState<AuditLogEntry[]>([])
  const [tab, setTab] = useState<'tenants' | 'audit'>('tenants')
  const [pesan, setPesan] = useState('')

  useEffect(() => {
    superadmin.me().catch(() => nav('/superadmin'))
    superadmin.listTenants().then(setTenants).catch((e) => setPesan(e.message))
    superadmin.auditLog().then(setAudit).catch(() => [])
  }, [nav])

  function keluar() {
    superadmin.clear()
    nav('/superadmin')
  }

  return (
    <div className="min-h-screen bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface px-6 py-4 flex items-center justify-between">
        <div>
          <h1 className="font-bold">Superadmin Platform</h1>
          <p className="text-xs text-ink2">Logikraf — manajemen tenant & audit</p>
        </div>
        <button onClick={keluar} className="btn-ghost">Keluar</button>
      </header>

      <main className="mx-auto max-w-5xl p-6 space-y-6">
        <div className="flex gap-2">
          <button onClick={() => setTab('tenants')} className={`btn-ghost ${tab === 'tenants' ? 'ring-2 ring-brand' : ''}`}>
            Tenant ({tenants.length})
          </button>
          <button onClick={() => setTab('audit')} className={`btn-ghost ${tab === 'audit' ? 'ring-2 ring-brand' : ''}`}>
            Audit Log
          </button>
        </div>

        {pesan && <p className="text-sm text-danger">{pesan}</p>}

        {tab === 'tenants' && (
          <section className="card">
            <h2 className="font-semibold mb-4">Daftar Tenant</h2>
            {tenants.length === 0 ? (
              <p className="text-sm text-ink2">Belum ada tenant.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-line text-left text-ink2">
                      <th className="py-2">Nama</th>
                      <th className="py-2">Slug</th>
                      <th className="py-2">Status</th>
                      <th className="py-2">Dibuat</th>
                    </tr>
                  </thead>
                  <tbody>
                    {tenants.map((t) => (
                      <tr key={t.id} className="border-b border-line">
                        <td className="py-2">{t.name}</td>
                        <td className="py-2 text-ink2">{t.slug}</td>
                        <td className="py-2">
                          <span className={`badge ${t.status === 'active' ? 'badge-success' : 'badge-warning'}`}>
                            {t.status}
                          </span>
                        </td>
                        <td className="py-2 text-ink2">{t.created_at}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {tab === 'audit' && (
          <section className="card">
            <h2 className="font-semibold mb-4">Audit Log Global</h2>
            {audit.length === 0 ? (
              <p className="text-sm text-ink2">Belum ada audit.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-line text-left text-ink2">
                      <th className="py-2">Tenant</th>
                      <th className="py-2">Aksi</th>
                      <th className="py-2">Entitas</th>
                      <th className="py-2">Waktu</th>
                    </tr>
                  </thead>
                  <tbody>
                    {audit.map((a) => (
                      <tr key={a.id} className="border-b border-line">
                        <td className="py-2 text-ink2">{a.tenant_id}</td>
                        <td className="py-2">{a.action}</td>
                        <td className="py-2 text-ink2">{a.entity}{a.entity_id ? `/${a.entity_id}` : ''}</td>
                        <td className="py-2 text-ink2">{a.created_at}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        <section className="card">
          <h2 className="font-semibold mb-2">Reset Password Superadmin</h2>
          <Link to="/superadmin/reset" className="btn-ghost">Ganti kata sandi</Link>
        </section>
      </main>
    </div>
  )
}
