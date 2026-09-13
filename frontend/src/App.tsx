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

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Landing />} />
        <Route path="/masuk" element={<Login />} />
        <Route path="/aktivasi" element={<Activate />} />
        <Route path="/dasbor" element={<Dashboard />} />
        <Route path="/sensus" element={<Sensus />} />
        <Route path="/verifikasi" element={<Verifikasi />} />
        <Route path="/rumah" element={<Rumah />} />
        <Route path="/tagihan" element={<Tagihan />} />
        <Route path="/keuangan" element={<Keuangan />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
