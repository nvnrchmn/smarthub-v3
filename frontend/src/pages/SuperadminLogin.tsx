import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { superadmin } from '../lib/api'

export default function SuperadminLogin() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [galat, setGalat] = useState('')
  const [sibuk, setSibuk] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setGalat('')
    setSibuk(true)
    try {
      await superadmin.login(email, password)
      nav('/superadmin/dasbor')
    } catch (err) {
      setGalat(err instanceof Error ? err.message : 'gagal masuk')
    } finally {
      setSibuk(false)
    }
  }

  return (
    <div className="min-h-screen bg-canvas flex items-center justify-center p-4">
      <div className="card w-full max-w-md">
        <div className="text-center mb-6">
          <h1 className="text-2xl font-bold text-ink">Superadmin</h1>
          <p className="mt-1 text-sm text-ink2">Logikraf Platform</p>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <div>
            <label className="block text-sm text-ink2 mb-1">Email</label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-xl border border-line bg-canvas px-4 py-3 outline-none focus:border-brand"
              placeholder="superadmin@logikraf.id"
            />
          </div>
          <div>
            <label className="block text-sm text-ink2 mb-1">Kata sandi</label>
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-xl border border-line bg-canvas px-4 py-3 outline-none focus:border-brand"
              placeholder="••••••••"
            />
          </div>
          {galat && <p className="text-sm text-danger">{galat}</p>}
          <button type="submit" disabled={sibuk} className="btn-primary w-full justify-center">
            {sibuk ? 'Memproses…' : 'Masuk'}
          </button>
        </form>
      </div>
    </div>
  )
}
