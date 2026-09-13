import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { superadmin, type AuditLogEntry, type TenantList } from '../lib/api'

// Tanggal ringkas ala Indonesia: 13 Sep 2026, 07:44.
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
    superadmin.listTenants().then(setTenants).catch((e) => setPesan(e.message))
    superadmin.auditLog().then(setAudit).catch(() => [])
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
    } catch (e) {
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
    } catch (e) {
      setPesan(e instanceof Error ? e.message : 'gagal ganti kata sandi')
    }
  }

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
          <button onClick={() => setTab('pengaturan')} className={`btn-ghost ${tab === 'pengaturan' ? 'ring-2 ring-brand' : ''}`}>
            Pengaturan
          </button>
        </div>

        {pesan && <p className="text-sm text-danger">{pesan}</p>}
        {sukses && <p className="text-sm text-brand">{sukses}</p>}

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
                        <td className="py-2 text-ink2">{tgl(t.created_at)}</td>
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
                        <td className="py-2 text-ink2">{tgl(a.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {tab === 'pengaturan' && (
          <>
            <section className="card">
              <h2 className="font-semibold mb-1">Kunci Xendit Platform</h2>
              <p className="text-sm text-ink2 mb-4">
                Dipakai untuk pembuatan QRIS seluruh tenant. Prefix klien hub (mis. <code>sb-</code>).
              </p>
              <form onSubmit={simpanXendit} className="space-y-3 max-w-lg">
                <div>
                  <label className="block text-sm text-ink2 mb-1">Secret Key Xendit</label>
                  <input
                    type="password"
                    className="field"
                    value={xenditKey}
                    onChange={(e) => setXenditKey(e.target.value)}
                    placeholder="xnd_development_..."
                    autoComplete="off"
                  />
                </div>
                <div>
                  <label className="block text-sm text-ink2 mb-1">Prefix Klien Hub</label>
                  <input
                    type="text"
                    className="field"
                    value={xenditPrefix}
                    onChange={(e) => setXenditPrefix(e.target.value)}
                    placeholder="sb-"
                  />
                </div>
                <button type="submit" className="btn-primary">Simpan</button>
              </form>
            </section>

            <section className="card">
              <h2 className="font-semibold mb-1">Ganti Kata Sandi Superadmin</h2>
              <p className="text-sm text-ink2 mb-4">Minimal 8 karakter. Perlu kata sandi lama.</p>
              <form onSubmit={gantiSandi} className="space-y-3 max-w-lg">
                <div>
                  <label className="block text-sm text-ink2 mb-1">Kata Sandi Lama</label>
                  <input
                    type="password"
                    className="field"
                    value={oldPw}
                    onChange={(e) => setOldPw(e.target.value)}
                    autoComplete="current-password"
                  />
                </div>
                <div>
                  <label className="block text-sm text-ink2 mb-1">Kata Sandi Baru</label>
                  <input
                    type="password"
                    className="field"
                    value={newPw}
                    onChange={(e) => setNewPw(e.target.value)}
                    autoComplete="new-password"
                  />
                </div>
                <button type="submit" className="btn-primary">Ganti Kata Sandi</button>
              </form>
            </section>
          </>
        )}
      </main>
    </div>
  )
}
