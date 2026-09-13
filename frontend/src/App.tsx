import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import Landing from './pages/Landing'
import Login from './pages/Login'
import Activate from './pages/Activate'
import Dashboard from './pages/Dashboard'
import Sensus from './pages/Sensus'
import Verifikasi from './pages/Verifikasi'
import Rumah from './pages/Rumah'
import Tagihan from './pages/Tagihan'
import Keuangan from './pages/Keuangan'
import Laporan from './pages/Laporan'
import SuperadminLogin from './pages/SuperadminLogin'
import SuperadminDasbor from './pages/SuperadminDasbor'
import Layout from './components/Layout'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Di luar layout: halaman publik & area superadmin. */}
        <Route path="/" element={<Landing />} />
        <Route path="/masuk" element={<Login />} />
        <Route path="/aktivasi" element={<Activate />} />
        <Route path="/superadmin" element={<SuperadminLogin />} />
        <Route path="/superadmin/dasbor" element={<SuperadminDasbor />} />

        {/* Di dalam layout: semua halaman ber-login (dapat menu navigasi). */}
        <Route element={<Layout />}>
          <Route path="/dasbor" element={<Dashboard />} />
          <Route path="/tagihan" element={<Tagihan />} />
          <Route path="/keuangan" element={<Keuangan />} />
          <Route path="/laporan" element={<Laporan />} />
          <Route path="/rumah" element={<Rumah />} />
          <Route path="/sensus" element={<Sensus />} />
          <Route path="/verifikasi" element={<Verifikasi />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
