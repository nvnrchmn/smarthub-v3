import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { superadmin, auth, type SuperadminMe } from '../lib/api'

export default function Settings() {
  const nav = useNavigate()
  const [me, setMe] = useState<SuperadminMe | null>(null)
  const [xenditKey, setXenditKey] = useState('')
  const [xenditPrefix, setXenditPrefix] = useState('')
  const [oldPw, setOldPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')

  useEffect(() => {
    const t = auth.get()
    if (!t) { nav('/superadmin'); return }
    superadmin.me().then(setMe).catch(() => nav('/superadmin'))
    superadmin.getSetting('xendit_key').then((v: any) => setXenditKey(v.value || ''))
    superadmin.getSetting('xendit_prefix').then((v: any) => setXenditPrefix(v.value || ''))
  }, [nav])

  async function simpanXendit(e: React.FormEvent) {
    e.preventDefault()
    setMsg(''); setErr('')
    try {
      await superadmin.setSetting('xendit_key', xenditKey)
      await superadmin.setSetting('xendit_prefix', xenditPrefix)
      setMsg('Pengaturan Xendit tersimpan')
    } catch (e: any) {
      setErr(e.message || 'gagal menyimpan')
    }
  }

  async function resetPw(e: React.FormEvent) {
    e.preventDefault()
    setMsg(''); setErr('')
    try {
      await superadmin.resetPassword(oldPw, newPw)
      setMsg('Password direset')
      setOldPw(''); setNewPw('')
    } catch (e: any) {
      setErr(e.message || 'gagal reset password')
    }
  }

  if (!me) return null

  return (
    <div className="min-h-screen bg-canvas">
      <header className="sticky top-0 z-10 border-b border-line bg-surface px-4 py-3 flex items-center justify-between">
        <h1 className="font-semibold text-ink">Pengaturan Platform</h1>
        <button onClick={() => nav('/superadmin/dasbor')} className="btn-ghost">Kembali</button>
      </header>
      <main className="mx-auto max-w-2xl space-y-6 p-4">
        <section className="card">
          <h2 className="font-semibold text-ink mb-4">Kunci Xendit (Payment Hub)</h2>
          <form onSubmit={simpanXendit} className="space-y-3">
            <div>
              <label className="text-sm text-ink2">Internal Key</label>
              <input type="text" value={xenditKey} onChange={e => setXenditKey(e.target.value)} className="form-input" placeholder="key-..." />
            </div>
            <div>
              <label className="text-sm text-ink2">External Prefix</label>
              <input type="text" value={xenditPrefix} onChange={e => setXenditPrefix(e.target.value)} className="form-input" placeholder="sb-" />
            </div>
            <button type="submit" className="btn-primary">Simpan</button>
          </form>
        </section>

        <section className="card">
          <h2 className="font-semibold text-ink mb-4">Reset Password Superadmin</h2>
          <form onSubmit={resetPw} className="space-y-3">
            <div>
              <label className="text-sm text-ink2">Password Lama</label>
              <input type="password" value={oldPw} onChange={e => setOldPw(e.target.value)} className="form-input" />
            </div>
            <div>
              <label className="text-sm text-ink2">Password Baru (min 8)</label>
              <input type="password" value={newPw} onChange={e => setNewPw(e.target.value)} className="form-input" />
            </div>
            <button type="submit" className="btn-primary">Reset</button>
          </form>
        </section>

        {msg && <p className="text-sm text-ok">{msg}</p>}
        {err && <p className="text-sm text-danger">{err}</p>}
      </main>
    </div>
  )
}
