import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { type Health } from '../lib/api'

/*
 * Landing.tsx — halaman publik (statis). Dapatkan tema dari system preference,
 * tidak perlu login. Semua data hard-coded (mockup) untuk demo.
 */

async function fetchHealth(): Promise<Health | null> {
  const res = await fetch('/api/health', { credentials: 'omit' })
  if (!res.ok) return null
  return (await res.json()) as Health
}

const ROLES = [
  { name: 'Warga', desc: 'Lihat tagihan iuran, bayar QRIS, ajukan surat, dan akses lapak warga.', icon: '🏠' },
  { name: 'Sekretaris', desc: 'Verifikasi sensus (NIK/KK terenkripsi), kelola data rumah & warga.', icon: '📝' },
  { name: 'Bendahara', desc: 'Terbitkan tagihan, kuitensi digital, dan buku kas masuk.', icon: '💰' },
  { name: 'Pengelola', desc: 'Kelola tenant, pengurus, kebijakan, dan laporan perumahan.', icon: '⚙️' },
]

const FEATURES = [
  {
    title: 'Pembayaran via QRIS Dinamis',
    desc: 'QR code unik per faktur. Rekonsiliasi otomatis tiap 5 menit.',
    icon: (
      <svg className="h-5 w-5 text-brand" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 4.5v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591 1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z"
        />
      </svg>
    ),
  },
  {
    title: 'Buku Kas Tunai',
    desc: 'Catat setoran fisik & kas harian. Setiap transaksi tercatat dengan persetujuan Ketua.',
    icon: (
      <svg className="h-5 w-5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 8c-2.21 0-4-.896-4-2s1.79-2 4-2 4 .896 4 2-1.79 2-4 2z"
        />
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v10m-6-4h12" />
      </svg>
    ),
  },
  {
    title: 'Sensus Warga Terenkripsi',
    desc: 'NIK & KK dienkripsi AES-256-GCM. Hanya sekretaris yang bisa lihat setelah verifikasi.',
    icon: (
      <svg className="h-5 w-5 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M16.5 10.5V7.5a4.5 4.5 0 10-9 0v3m.75 4.5h7.5m-7.5 2.25h7.5M9 18h6a2 2 0 100-4H9a2 2 0 100 4z"
        />
      </svg>
    ),
  },
  {
    title: 'Pengingat via WhatsApp',
    desc: 'OTP, aktivasi, dan tagihan menunggak dikirim langsung lewat GoWA terhubung.',
    icon: (
      <svg className="h-5 w-5 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 18v.75a2.25 2.25 0 002.25 2.25H15a3 3 0 003-3v-2.25M9.75 9h.008v.008H9.75zM15 9h.008v.008H15m-3 3h.008v.008H12z"
        />
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M8.25 4.5A4.25 4.25 0 0112 3a4.25 4.25 0 014.25 4.25V8.5a2.25 2.25 0 01-2.25 2.25H9.75A2.25 2.25 0 017.5 8.5v-1A4.25 4.25 0 018.25 4.5z"
        />
      </svg>
    ),
  },
]

const STEPS = [
  { num: '1', title: 'Undang Warga', desc: 'Pengelola mengirim undangan via email + nomor WhatsApp.' },
  { num: '2', title: 'Verifikasi OTP', desc: 'Warga menerima kode 6 digit, masukkan nama, buat kata sandi.' },
  { num: '3', title: 'Mulai Pakai', desc: 'Akses sesuai peran — dasbor, tagihan, laporan, sensus.' },
]

const TRUST_ITEMS = [
  { label: 'AES-256-GCM', subtitle: 'Enkripsi NIK & KK' },
  { label: 'PostgreSQL RLS', subtitle: 'Isolasi tenant 100%' },
  { label: 'JWT + Cabut Token', subtitle: 'Sesi aman' },
  { label: 'Invite-only', subtitle: 'Akses oleh pengelola' },
  { label: 'GoWA Aktif', subtitle: 'Notifikasi WhatsApp langsung' },
  { label: 'Audit Trail', subtitle: 'Setiap aksi tercatat' },
]

const FAQ_ITEMS = [
  {
    q: 'Apakah data kami aman?',
    a: 'Ya. Semua data pribadi (NIK, KK) dienkripsi AES-256-GCM sebelum tersimpan, dan database menggunakan PostgreSQL Row-Level Security untuk isolasi antar-perumahan.',
  },
  {
    q: 'Bagaimana cara mendaftar?',
    a: 'Smarthub bersifat invite-only. Pengelola perumahan mengundang warga via email + WhatsApp, lalu warga menerima OTP 6 digit untuk mengaktifkan akun.',
  },
  {
    q: 'Apakah ada biaya berlangganan?',
    a: 'Hubungi PT Logika Kreatif Indonesia untuk konseultasi harga. Paket fleksibel per perumahan, sesuai jumlah unit hunian.',
  },
]

export default function Landing() {
  const [health, setHealth] = useState<Health | null>(null)
  const [dark, setDark] = useState(true)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    document.documentElement.classList.toggle('light', !dark)
  }, [dark])

  useEffect(() => {
    fetchHealth().then(setHealth).catch(() => setHealth(null))
  }, [])

  return (
    <div className="font-sans text-sm text-secondary antialiased">
      {/* ==================== HEADER ==================== */}
      <header className="sticky top-0 z-50 border-b border-slate-800 bg-slate-900/70 backdrop-blur-lg">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
          <Link to="/" className="text-xl font-bold tracking-tight text-primary">
            Smarthub<span className="text-brand">.</span>
          </Link>

          <nav className="hidden items-center gap-6 text-sm md:flex">
            <a href="#fitur" className="text-secondary hover:text-primary transition-colors">Fitur</a>
            <a href="#cara-kerja" className="text-secondary hover:text-primary transition-colors">Cara Kerja</a>
            <a href="#faq" className="text-secondary hover:text-primary transition-colors">FAQ</a>
          </nav>

          <div className="flex items-center gap-3">
            <button
              onClick={() => setDark(!dark)}
              className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-slate-700 text-slate-400 hover:bg-slate-800 hover:text-slate-200 transition"
              aria-label="Toggle theme"
            >
              {dark ? '☀' : '☾'}
            </button>
            <Link
              to="/masuk"
              className="inline-flex items-center justify-center rounded-md bg-gradient-to-r from-indigo-500 to-purple-600 px-4 py-2 text-sm font-medium text-white shadow hover:opacity-90 focus:outline-none"
            >
              Masuk
            </Link>
          </div>
        </div>
      </header>

      {/* ==================== HERO ==================== */}
      <main>
        <section className="mx-auto max-w-5xl px-6 py-16 sm:py-20">
          <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-12">

            {/* Text */}
            <div>
              {/* Status Badge */}
              <div className="mb-5 flex items-center gap-2">
                <span className={`h-1.5 w-1.5 rounded-full ${health?.database ? 'bg-emerald-400' : 'bg-amber-400'}`} />
                <span className="text-xs text-slate-500">
                  {health
                    ? `API ${health.status} · database ${health.database ? 'terhubung' : 'bermasalah'}`
                    : 'memeriksa…'}
                </span>
              </div>

              <h1 className="text-3xl font-bold leading-tight tracking-tight sm:text-4xl lg:text-5xl text-primary">
                Satu platform untuk{' '}
                <span className="bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent">
                  iuran, sensus, dan buku kas
                </span>{' '}
                perumahan RT/RW.
              </h1>

              <p className="mt-5 max-w-md text-base leading-relaxed text-secondary">
                Iuran bulanan via QRIS dinamis + kas tunai, survei warga dengan data pribadi terenkripsi AES-256-GCM,
                pengingat tunggakan via WhatsApp, dan buku kas transparan — dalam satu sistem multi-perumahan yang aman.
              </p>

              <div className="mt-8 flex flex-wrap items-center gap-3">
                <Link
                  to="/masuk"
                  className="inline-flex items-center justify-center rounded-md bg-gradient-to-r from-indigo-500 to-purple-600 px-5 py-2.5 text-sm font-medium text-white shadow hover:opacity-90 focus:outline-none"
                >
                  Mulai Sekarang
                  <svg className="ml-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
                  </svg>
                </Link>
                <Link
                  to="/aktivasi"
                  className="inline-flex items-center justify-center rounded-md border border-slate-700 px-5 py-2.5 text-sm font-medium text-slate-300 hover:bg-slate-800 hover:text-slate-100 transition focus:outline-none"
                >
                  Aktivasi Akun
                </Link>
              </div>

              {/* Trust mini bar */}
              <div className="mt-6 flex flex-wrap items-center gap-5 text-xs text-slate-500">
                <span className="flex items-center gap-1">
                  <span className="h-1 w-1.5 rounded-full bg-emerald-400" />
                  Data tetap di server lokal
                </span>
                <span className="flex items-center gap-1">
                  <span className="h-1 w-1.5 rounded-full bg-emerald-400" />
                  Tanpa iklan, tanpa berbagi data
                </span>
              </div>
            </div>

            {/* Hero Visual: minimal card grid */}
            <div>
              <div className="relative">
                {/* Soft glow behind mockup */}
                <div className="absolute -inset-6 bg-gradient-to-r from-indigo-500/5 via-purple-500/5 to-transparent rounded-full blur-3xl opacity-70" />

                <div className="relative mx-auto grid gap-4 max-w-sm">
                  {/* Main card: invoice / payment */}
                  <div className="rounded-xl border border-slate-800 bg-slate-900 p-4 shadow-lg">
                    <div className="mb-3 flex items-center justify-between">
                      <h3 className="text-sm font-semibold text-primary">Tagihan Iuran</h3>
                      <span className="inline-flex items-center rounded-full border border-amber-900/50 bg-amber-900/20 px-2 py-0.5 text-xs font-medium text-amber-300">
                        Belum bayar
                      </span>
                    </div>

                    <p className="text-xs text-slate-500">Rumah A-01 · Periode 2026-09</p>
                    <p className="mt-1.5 text-xl font-bold text-primary">Rp275.000</p>

                    <div className="mt-3 grid grid-cols-2 gap-2">
                      <div className="flex flex-col items-center justify-center rounded-lg border border-slate-800 bg-slate-950 py-2.5 text-center">
                        <svg
                          className="mx-auto h-5 w-5 text-indigo-400"
                          fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}
                        >
                          <rect x="3" y="5" width="18" height="14" rx="2" />
                          <path strokeLinecap="round" strokeLinejoin="round" d="M8 9l4 4 4-4" />
                        </svg>
                        <span className="block text-[10px] text-slate-500 mt-1">QRIS</span>
                      </div>
                      <div className="flex flex-col items-center justify-center rounded-lg border border-slate-800 bg-slate-950 py-2.5 text-center">
                        <svg
                          className="mx-auto h-5 w-5 text-emerald-400"
                          fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}
                        >
                          <path strokeLinecap="round" strokeLinejoin="round"
                            d="M12 8c1.654 0 3 1.346 3 3s-1.346 3-3 3-3-1.346-3-3 1.346-3 3-3z"
                          />
                          <path strokeLinecap="round" strokeLinejoin="round"
                            d="M12 14V4m0 10v4m8-8h-2.4A2.4 2.4 0 0 0 14 8.4V6"
                          />
                        </svg>
                        <span className="block text-[10px] text-slate-500 mt-1">Tunai</span>
                      </div>
                    </div>
                  </div>

                  {/* Floating offset card: kas */}
                  <div className="-mt-3 rounded-xl border border-slate-800/60 bg-slate-900/60 p-4 shadow-lg">
                    <h3 className="mb-2 text-sm font-semibold text-primary">Buku Kas</h3>
                    <div className="grid grid-cols-2 gap-2 text-center">
                      <div>
                        <p className="text-xs text-slate-500">Kas Bank</p>
                        <p className="text-sm font-bold text-emerald-400">Rp1.125.000</p>
                      </div>
                      <div>
                        <p className="text-xs text-slate-500">Kas Fisik</p>
                        <p className="text-sm font-bold text-amber-400">Rp250.000</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ==================== FEATURE GRID ==================== */}
        <section id="fitur" className="mx-auto max-w-5xl px-6 py-14">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-primary">Fitur Unggulan</h2>
            <p className="mt-2 max-w-xl text-sm text-secondary mx-auto">
              Semua kebutuhan administrasi perumahan dalam satu sistem yang aman dan terintegrasi.
            </p>
          </div>

          <div className="mt-10 grid gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
            {FEATURES.map((f, _i) => (
              <div
                key={f.title}
                className="group rounded-xl border border-slate-800 bg-slate-900/70 p-4 transition-all duration-200 hover:border-slate-700 hover:bg-slate-900"
              >
                <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10">
                  {f.icon}
                </div>
                <h3 className="text-sm font-semibold text-primary">{f.title}</h3>
                <p className="mt-1 text-xs leading-relaxed text-slate-400">{f.desc}</p>
              </div>
            ))}
          </div>
        </section>

        {/* ==================== HOW IT WORKS ==================== */}
        <section id="cara-kerja" className="border-t border-slate-800 py-14">
          <div className="mx-auto max-w-3xl px-6">
            <div className="text-center">
              <h2 className="text-2xl font-bold text-primary">Cara Kerja</h2>
              <p className="mt-2 text-sm text-secondary">
                Tiga langkah sederhana untuk seluruh perumahan siap digital.
              </p>
            </div>

            <div className="mt-10 grid gap-6 sm:grid-cols-3">
              {STEPS.map((s) => (
                <div key={s.num} className="text-center">
                  <div className="mx-auto flex h-9 w-9 items-center justify-center rounded-full bg-indigo-500 text-sm font-bold text-white">
                    {s.num}
                  </div>
                  <h3 className="mt-3 text-sm font-semibold text-primary">{s.title}</h3>
                  <p className="mt-1 text-xs text-secondary">{s.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ==================== ROLE CARDS ==================== */}
        <section className="mx-auto max-w-5xl px-6 py-14">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-primary">Peran & Akses</h2>
            <p className="mt-2 max-w-xl text-sm text-secondary mx-auto">
              Setiap peran memiliki akses yang tepat sesuai tanggung jawabnya.
            </p>
          </div>

          <div className="mt-10 grid gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
            {ROLES.map((r) => (
              <article
                key={r.name}
                className="group rounded-xl border border-slate-800 bg-slate-900/70 p-4 text-center transition-all duration-200 hover:border-slate-700 hover:bg-slate-900"
              >
                <div className="mb-3 text-2xl">{r.icon}</div>
                <h2 className="text-sm font-semibold text-primary">{r.name}</h2>
                <p className="mt-1 text-xs leading-relaxed text-slate-400">{r.desc}</p>
              </article>
            ))}
          </div>
        </section>

        {/* ==================== TRUST / SECURITY ==================== */}
        <section className="border-t border-slate-800 py-12">
          <div className="mx-auto max-w-4xl px-6">
            <div className="text-center">
              <h2 className="text-xl font-bold text-primary">Keamanan & Privasi</h2>
              <p className="mt-1.5 text-sm text-secondary">
                Data perumahan Anda dilindungi dengan standar keamanan tinggi.
              </p>
            </div>

            <div className="mt-8 grid gap-2.5 sm:grid-cols-2 lg:grid-cols-3">
              {TRUST_ITEMS.map((t) => (
                <div
                  key={t.label}
                  className="rounded-lg border border-slate-800 bg-slate-900/50 px-3.5 py-2.5 text-center"
                >
                  <p className="text-sm font-semibold text-primary">{t.label}</p>
                  <p className="mt-0.5 text-xs text-slate-500">{t.subtitle}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ==================== FAQ ==================== */}
        <section id="faq" className="mx-auto max-w-3xl px-6 py-14">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-primary">Pertanyaan Umum</h2>
          </div>

          <div className="mt-8 space-y-2.5">
            {FAQ_ITEMS.map((item) => (
              <details
                key={item.q}
                className="rounded-xl border border-slate-800 bg-slate-900/50 p-4 [&>summary]:cursor-pointer [&>summary]:list-none [&>summary]:font-medium [&>summary]:text-primary [&>summary]:flex [&>summary]:items-center [&>summary]:justify-between [&_svg]:transition-transform [&[open]>summary>svg]:rotate-180"
              >
                <summary>
                  <span>{item.q}</span>
                  <svg
                    className="h-5 w-5 text-slate-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={1.5}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </summary>
                <p className="mt-2 text-xs leading-relaxed text-slate-400">{item.a}</p>
              </details>
            ))}
          </div>
        </section>

        {/* ==================== SECONDARY CTA ==================== */}
        <section className="border-t border-slate-800 py-14">
          <div className="mx-auto max-w-3xl px-6 text-center">
            <h2 className="text-xl font-bold text-primary">Siap digitalkan perumahan Anda?</h2>
            <p className="mt-2 text-sm text-secondary">
              Hubungi kami untuk demo gratis.
            </p>
            <div className="mt-5 flex flex-col justify-center gap-2.5 sm:flex-row">
              <Link
                to="/masuk"
                className="inline-flex items-center justify-center rounded-md bg-gradient-to-r from-indigo-500 to-purple-600 px-5 py-2 text-sm font-medium text-white shadow hover:opacity-90 focus:outline-none"
              >
                Coba Sekarang
              </Link>
              <a
                href="https://wa.me/628983342429"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center rounded-md border border-slate-700 px-5 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800 hover:text-slate-100 transition focus:outline-none"
              >
                Hubungi Kami
              </a>
            </div>
          </div>
        </section>
      </main>

      {/* ==================== FOOTER ==================== */}
      <footer className="border-t border-slate-800 py-8">
        <div className="mx-auto max-w-5xl px-6">
          <div className="grid gap-6 sm:grid-cols-3">
            <div>
              <span className="text-xl font-bold text-primary">
                Smarthub<span className="text-brand">.</span>
              </span>
              <p className="mt-2 text-xs text-slate-500">
                Platform digital untuk perumahan RT/RW &amp; komunitas.
                Dikembangkan oleh PT Logika Kreatif Indonesia.
              </p>
            </div>

            <div>
              <h4 className="font-semibold text-sm text-primary mb-2">Halaman</h4>
              <ul className="space-y-1 text-xs text-slate-400">
                <li>
                  <Link to="/" className="hover:text-primary transition-colors">Beranda</Link>
                </li>
                <li>
                  <a href="#fitur" className="hover:text-primary transition-colors">Fitur</a>
                </li>
                <li>
                  <a href="#cara-kerja" className="hover:text-primary transition-colors">Cara Kerja</a>
                </li>
                <li>
                  <a href="#faq" className="hover:text-primary transition-colors">FAQ</a>
                </li>
              </ul>
            </div>

            <div>
              <h4 className="font-semibold text-sm text-primary mb-2">Perusahaan</h4>
              <ul className="space-y-1 text-xs text-slate-400">
                <li>
                  <a
                    href="https://logikraf.id"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-primary transition-colors"
                  >Logikraf.id</a>
                </li>
                <li>
                  <a
                    href="https://wa.me/628983342429"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-primary transition-colors"
                  >Kontak</a>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-8 pt-4 border-t border-slate-800 text-center text-xs text-slate-600">
            © {new Date().getFullYear()} PT Logika Kreatif Indonesia. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  )
}
