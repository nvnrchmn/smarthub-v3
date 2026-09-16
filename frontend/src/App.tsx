import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import Landing from './pages/Landing'
import Login from './pages/Login'
import Activate from './pages/Activate'
import SuperadminLogin from './pages/SuperadminLogin'
import SuperadminDasbor from './pages/SuperadminDasbor'
import Layout from './components/Layout'
import ErrorBoundary from './components/ErrorBoundary'
import NotFound from './pages/NotFound'

// Lazy load untuk halaman berat — tampilkan skeleton saat loading.
const Dashboard = lazy(() => import('./pages/Dashboard'))
const Sensus = lazy(() => import('./pages/Sensus'))
const Verifikasi = lazy(() => import('./pages/Verifikasi'))
const Rumah = lazy(() => import('./pages/Rumah'))
const Tagihan = lazy(() => import('./pages/Tagihan'))
const Keuangan = lazy(() => import('./pages/Keuangan'))
const Laporan = lazy(() => import('./pages/Laporan'))

function PageLoader() {
  return (
    <div className="flex items-center justify-center min-h-[60vh]" role="status" aria-label="Memuat halaman">
      <div className="flex flex-col items-center gap-3">
        <div className="w-8 h-8 border-4 border-blue-200 border-t-blue-600 rounded-full animate-spin" aria-hidden="true" />
        <span className="text-sm text-slate-500">Memuat...</span>
      </div>
    </div>
  )
}

export default function App() {
  return (
    <ErrorBoundary>
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
            <Route path="/dasbor" element={<Suspense fallback={<PageLoader />}><Dashboard /></Suspense>} />
            <Route path="/tagihan" element={<Suspense fallback={<PageLoader />}><Tagihan /></Suspense>} />
            <Route path="/keuangan" element={<Suspense fallback={<PageLoader />}><Keuangan /></Suspense>} />
            <Route path="/laporan" element={<Suspense fallback={<PageLoader />}><Laporan /></Suspense>} />
            <Route path="/rumah" element={<Suspense fallback={<PageLoader />}><Rumah /></Suspense>} />
            <Route path="/sensus" element={<Suspense fallback={<PageLoader />}><Sensus /></Suspense>} />
            <Route path="/verifikasi" element={<Suspense fallback={<PageLoader />}><Verifikasi /></Suspense>} />
          </Route>

          {/* 404: halaman nyata, bukan redirect. */}
          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  )
}
