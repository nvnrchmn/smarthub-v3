import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

type Health = { status: string; database: boolean; redis: boolean; service: string }

const roles = [
  { name: 'Warga', desc: 'Lihat tagihan iuran, bayar QRIS, ajukan surat, dan akses lapak warga.', icon: '🏠' },
  { name: 'Sekretaris', desc: 'Verifikasi sensus (NIK/KK terenkripsi), kelola data rumah & warga.', icon: '📝' },
  { name: 'Bendahara', desc: 'Terbitkan tagihan, kuitensi digital, dan buku kas masuk.', icon: '💰' },
  { name: 'Pengelola', desc: 'Kelola tenant, pengurus, kebijakan, dan laporan perumahan.', icon: '⚙️' },
]

const FEATURES = [
  {
    title: 'Pembayaran via QRIS Dinamis',
    desc: 'QR code unik per faktur. Rekonsiliasi otomatis tiap 5 menit via cron.',
    icon: (
      <svg className="h-5 w-5 text-brand" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 4.5v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591 1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />
      </svg>
    ),
  },
  {
    title: 'Buku Kas Tunai',
    desc: 'Catat setoran fisik & kas harian. Setiap transaksi tercatat dengan persetujuan Ketua.',
    icon: (
      <svg className="h-5 w-5 text-ok" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 8c-2.21 0-4-.896-4-2s1.79-2 4-2 4 .896 4 2-1.79 2-4 2z" />
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 8v10m-6-4h12" />
      </svg>
    ),
  },
  {
    title: 'Sensus Warga Terenkripsi',
    desc: 'NIK & KK dienkripsi AES-256-GCM. Hanya sekretaris yang bisa lihat setelah verifikasi.',
    icon: (
      <svg className="h-5 w-5 text-info" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M16.5 10.5V7.5a4.5 4.5 0 10-9 0v3m.75 4.5h7.5m-7.5 2.25h7.5M9 18h6a2 2 0 100-4H9a2 2 0 100 4z" />
      </svg>
    ),
  },
  {
    title: 'Pengingat via WhatsApp',
    desc: 'OTP, aktivasi, dan tagihan menunggak dikirim langsung lewat GoWA terhubung.',
    icon: (
      <svg className="h-5 w-5 text-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M12 18v.75a2.25 2.25 0 002.25 2.25H15a3 3 0 003-3v-2.25M9.75 9h.008v.008H9.75zM15 9h.008v.008H15m-3 3h.008v.008H12z" />
        <path strokeLinecap="round" strokeLinejoin="round"
          d="M8.25 4.5A4.25 4.25 0 0112 3a4.25 4.25 0 014.25 4.25V8.5a2.25 2.25 0 01-2.25 2.25H9.75A2.25 2.25 0 017.5 8.5v-1A4.25 4.25 0 018.25 4.5z" />
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

export default function Landing() {
  const [health, setHealth] = useState<Health | null>(null)
  const [dark, setDark] = useState(true)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    document.documentElement.classList.toggle('light', !dark)
  }, [dark])

  useEffect(() => {
    fetch('/api/health')
      .then((r) => (r.ok ? r.json() : null))
      .then(setHealth)
      .catch(() => setHealth(null))
  }, [])

  return (
    <div className="min-h-screen bg-canvas text-ink">
      {/* ==================== HEADER ==================== */}
      <header className="sticky top-0 z-50 border-b border-line bg-surface/80 backdrop-blur-lg">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link to="/" className="text-xl font-bold tracking-tight">
            Smarthub<span className="text-brand">.</span>
          </Link>
          <nav className="hidden items-center gap-6 text-sm md:flex">
            <a href="#fitur" className="text-ink2 hover:text-ink transition-colors">
              Fitur
            </a>
            <a href="#cara-kerja" className="text-ink2 hover:text-ink transition-colors">
              Cara Kerja
            </a>
            <a href="#faq" className="text-ink2 hover:text-ink transition-colors">
              FAQ
            </a>
          </nav>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setDark((v) => !v)}
              className="btn-ghost"
              aria-label="Toggle theme"
            >
              {dark ? (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591 1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />
                </svg>
              ) : (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z" />
                </svg>
              )}
            </button>
            <Link to="/masuk" className="btn-primary text-sm">
              Masuk
            </Link>
          </div>
        </div>
      </header>

      {/* ==================== HERO ==================== */}
      <main>
        <section className="mx-auto max-w-6xl px-6 py-16 sm:py-24">
          <div className="grid items-center gap-12 lg:grid-cols-2 lg:gap-16">
            {/* Text */}
            <div className="animate-rise">
              {/* Status Badge */}
              <div className="mb-6 flex items-center gap-2">
                <span className={`status-dot ${health?.database ? 'ok' : 'warn'}`} />
                <span className="text-sm font-medium text-ink2">
                  {health
                    ? `API ${health.status} · database ${health.database ? 'terhubung' : 'bermasalah'}`
                    : 'memeriksa…'}
                </span>
              </div>

              <h1 className="text-4xl font-bold leading-tight tracking-tight sm:text-5xl lg:text-6xl">
                Satu platform untuk{' '}
                <span className="bg-gradient-to-r from-brand to-accent bg-clip-text text-transparent">
                  iuran, sensus, dan buku kas
                </span>{' '}
                perumahan RT/RW.
              </h1>

              <p className="mt-6 max-w-xl text-lg leading-relaxed text-ink2 sm:text-xl">
                Iuran bulanan via QRIS dinamis + kas tunai, survei warga dengan data pribadi
                terenkripsi AES-256-GCM, pengingat tunggakan via WhatsApp, dan buku kas
                transparan — dalam satu sistem multi-perumahan yang aman.
              </p>

              <div className="mt-10 flex flex-wrap items-center gap-4">
                <Link to="/masuk" className="btn-primary">
                  Mulai Sekarang
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
                  </svg>
                </Link>
                <Link to="/aktivasi" className="btn-secondary">
                  Aktivasi Akun
                </Link>
              </div>

              {/* Mini trust bar */}
              <div className="mt-8 flex flex-wrap items-center gap-6 text-xs text-muted">
                <span className="flex items-center gap-1">
                  <span className="h-1.5 w-1.5 rounded-full bg-ok" />
                  Data tetap di server lokal
                </span>
                <span className="flex items-center gap-1">
                  <span className="h-1.5 w-1.5 rounded-full bg-ok" />
                  Tanpa iklan, tanpa berbagi data
                </span>
              </div>
            </div>

            {/* Hero Visual: mockup card grid */}
            <div className="animate-rise animate-rise-delay-2">
              <div className="relative">
                {/* Background glow */}
                <div className="absolute -inset-8 bg-brand/5 rounded-full filter blur-3xl opacity-60" />

                {/* Mockup: stacked cards representing app screens */}
                <div className="relative mx-auto grid gap-4">
                  {/* Main card: invoice/payment */}
                  <div className="card border-brand/20">
                    <div className="mb-3 flex items-center justify-between">
                      <h3 className="font-semibold text-ink">Tagihan Iuran</h3>
                      <span className="badge badge-warning">Belum bayar</span>
                    </div>
                    <p className="text-xs text-muted">
                      Rumah A-01 · Periode 2026-09
                    </p>
                    <p className="mt-2 text-2xl font-bold text-ink">Rp275.000</p>
                    <div className="mt-3 grid grid-cols-2 gap-2">
                      <div className="rounded-xl border border-line bg-canvas p-2 text-center">
                        <svg
                          className="mx-auto h-6 w-6 text-brand"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          strokeWidth={1.5}
                        >
                          <rect x="3" y="5" width="18" height="14" rx="2" />
                          <path strokeLinecap="round" strokeLinejoin="round" d="M8 9l4 4 4-4" />
                        </svg>
                        <span className="block text-[10px] text-ink2 mt-1">QRIS</span>
                      </div>
                      <div className="rounded-xl border border-line bg-canvas p-2 text-center">
                        <svg
                          className="mx-auto h-6 w-6 text-ok"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          strokeWidth={1.5}
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 8c1.654 0 3 1.346 3 3s-1.346 3-3 3-3-1.346-3-3 1.346-3 3-3z" />
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            d="M12 14V4m0 10v4m8-8h-2.4A2.4 2.4 0 0 0 14 8.4V6"
                          />
                        </svg>
                        <span className="block text-[10px] text-ink2 mt-1">Tunai</span>
                      </div>
                    </div>
                  </div>

                  {/* Floating card: kas tunai */}
                  <div className="-mt-4 card border-line/50 lg:ml-auto lg:max-w-sm">
                    <h3 className="mb-2 font-semibold text-ink">Buku Kas</h3>
                    <div className="grid grid-cols-2 gap-2 text-center">
                      <div>
                        <p className="text-xs text-muted">Kas Bank</p>
                        <p className="font-bold text-ok text-sm">Rp1.125.000</p>
                      </div>
                      <div>
                        <p className="text-xs text-muted">Kas Fisik</p>
                        <p className="font-bold text-warn text-sm">Rp250.000</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ==================== FEATURE GRID ==================== */}
        <section id="fitur" className="mx-auto max-w-6xl px-6 py-16">
          <div className="text-center">
            <h2 className="text-3xl font-bold text-ink">Fitur Unggulan</h2>
            <p className="mt-3 max-w-2xl text-sm text-ink2 mx-auto">
              Semua kebutuhan administrasi perumahan dalam satu sistem yang aman dan
              terintegrasi.
            </p>
          </div>

          <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {FEATURES.map((f, i) => (
              <div
                key={f.title}
                className={`card-interactive animate-rise animate-rise-delay-${i + 1}`}
              >
                <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-brand/10">
                  {f.icon}
                </div>
                <h3 className="font-semibold">{f.title}</h3>
                <p className="mt-1 text-sm leading-relaxed text-ink2">{f.desc}</p>
              </div>
            ))}
          </div>
        </section>

        {/* ==================== HOW IT WORKS ==================== */}
        <section id="cara-kerja" className="border-t border-line bg-subtle/40 py-16">
          <div className="mx-auto max-w-4xl px-6">
            <div className="text-center">
              <h2 className="text-3xl font-bold text-ink">Cara Kerja</h2>
              <p className="mt-3 text-sm text-ink2">
                Tiga langkah sederhana untuk seluruh perumahan siap digital.
              </p>
            </div>

            <div className="mt-12 grid gap-6 sm:grid-cols-3">
              {STEPS.map((s) => (
                <div key={s.num} className="text-center">
                  <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-brand text-sm font-bold text-white">
                    {s.num}
                  </div>
                  <h3 className="mt-3 font-semibold">{s.title}</h3>
                  <p className="mt-1 text-sm text-ink2">{s.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ==================== ROLE CARDS ==================== */}
        <section className="mx-auto max-w-6xl px-6 py-16">
          <div className="text-center">
            <h2 className="text-3xl font-bold text-ink">Peran & Akses</h2>
            <p className="mt-3 max-w-2xl text-sm text-ink2 mx-auto">
              Setiap peran memiliki akses yang tepat sesuai tanggung jawabnya.
            </p>
          </div>

          <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {roles.map((r, i) => (
              <article
                key={r.name}
                className={`card-interactive animate-rise animate-rise-delay-${i + 1}`}
              >
                <div className="mb-3 text-2xl">{r.icon}</div>
                <h2 className="font-semibold">{r.name}</h2>
                <p className="mt-1 text-sm leading-relaxed text-ink2">{r.desc}</p>
              </article>
            ))}
          </div>
        </section>

        {/* ==================== TRUST / SECURITY ==================== */}
        <section className="border-t border-line bg-subtle/30 py-12">
          <div className="mx-auto max-w-4xl px-6">
            <div className="text-center">
              <h2 className="text-2xl font-bold text-ink">Keamanan & Privasi</h2>
              <p className="mt-2 text-sm text-ink2">
                Data perumahan Anda dilindungi dengan standar keamanan tinggi.
              </p>
            </div>

            <div className="mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {TRUST_ITEMS.map((t, i) => (
                <div
                  key={t.label}
                  className={`card text-center animate-rise animate-rise-delay-${i + 1}`}
                >
                  <p className="text-sm font-semibold text-ink">{t.label}</p>
                  <p className="mt-0.5 text-xs text-ink2">{t.subtitle}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ==================== FAQ ==================== */}
        <section id="faq" className="mx-auto max-w-3xl px-6 py-16">
          <div className="text-center">
            <h2 className="text-3xl font-bold text-ink">Pertanyaan Umum</h2>
          </div>

          <div className="mt-10 space-y-3">
            <details className="card group">
              <summary className="cursor-pointer list-none font-medium text-ink">
                <span className="flex items-center justify-between">
                  <span>Apakah data kami aman?</span>
                  <svg
                    className="h-5 w-5 text-ink2 transition-transform group-open:rotate-180"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={1.5}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </span>
              </summary>
              <p className="mt-2 text-sm leading-relaxed text-ink2">
                Ya. Semua data pribadi (NIK, KK) dienkripsi AES-256-GCM sebelum
                tersimpan, dan database menggunakan PostgreSQL Row-Level Security
                untuk isolasi antar-perumahan.
              </p>
            </details>

            <details className="card group">
              <summary className="cursor-pointer list-none font-medium text-ink">
                <span className="flex items-center justify-between">
                  <span>Bagaimana cara mendaftar?</span>
                  <svg
                    className="h-5 w-5 text-ink2 transition-transform group-open:rotate-180"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={1.5}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </span>
              </summary>
              <p className="mt-2 text-sm leading-relaxed text-ink2">
                Smarthub bersifat invite-only. Pengelola perumahan mengundang
                warga via email + WhatsApp, lalu warga menerima OTP 6 digit untuk
                mengaktifkan akun.
              </p>
            </details>

            <details className="card group">
              <summary className="cursor-pointer list-none font-medium text-ink">
                <span className="flex items-center justify-between">
                  <span>Apakah ada biaya berlangganan?</span>
                  <svg
                    className="h-5 w-5 text-ink2 transition-transform group-open:rotate-180"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={1.5}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </span>
              </summary>
              <p className="mt-2 text-sm leading-relaxed text-ink2">
                Hubungi PT Logika Kreatif Indonesia untuk konseultasi harga.
                Paket fleksibel per perumahan, sesuai jumlah unit hunian.
              </p>
            </details>
          </div>
        </section>

        {/* ==================== SECONDARY CTA ==================== */}
        <section className="border-t border-line bg-subtle/30 py-16">
          <div className="mx-auto max-w-3xl px-6 text-center">
            <h2 className="text-2xl font-bold text-ink">Siap digitalkan perumahan Anda?</h2>
            <p className="mt-3 text-sm text-ink2">
              Hubungi kami untuk demo gratis.
            </p>
            <div className="mt-6 flex flex-col justify-center gap-3 sm:flex-row">
              <Link to="/masuk" className="btn-primary">
                Coba Sekarang
              </Link>
              <a
                href="https://wa.me/628983342429"
                target="_blank"
                rel="noopener noreferrer"
                className="btn-secondary"
              >
                Hubungi Kami
              </a>
            </div>
          </div>
        </section>
      </main>

      {/* ==================== FOOTER ==================== */}
      <footer className="border-t border-line py-8">
        <div className="mx-auto max-w-6xl px-6">
          <div className="grid gap-6 sm:grid-cols-3">
            <div>
              <span className="text-xl font-bold">
                Smarthub<span className="text-brand">.</span>
              </span>
              <p className="mt-2 text-xs text-muted">
                Platform digital untuk perumahan RT/RW &amp; komunitas.
                Dikembangkan oleh PT Logika Kreatif Indonesia.
              </p>
            </div>
            <div className="text-sm">
              <h4 className="font-semibold text-ink mb-2">Halaman</h4>
              <ul className="space-y-1 text-ink2">
                <li>
                  <Link to="/" className="hover:text-ink transition-colors">
                    Beranda
                  </Link>
                </li>
                <li>
                  <a href="#fitur" className="hover:text-ink transition-colors">
                    Fitur
                  </a>
                </li>
                <li>
                  <a href="#cara-kerja" className="hover:text-ink transition-colors">
                    Cara Kerja
                  </a>
                </li>
                <li>
                  <a href="#faq" className="hover:text-ink transition-colors">
                    FAQ
                  </a>
                </li>
              </ul>
            </div>
            <div className="text-sm">
              <h4 className="font-semibold text-ink mb-2">Perusahaan</h4>
              <ul className="space-y-1 text-ink2">
                <li>
                  <a
                    href="https://logikraf.id"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-ink transition-colors"
                  >
                    Logikraf
                  </a>
                </li>
                <li>
                  <a href="mailto:contact@logikraf.id" className="hover:text-ink transition-colors">
                    Kontak
                  </a>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-8 border-t border-line pt-4 text-center text-xs text-muted">
            Smarthub v3 · PT Logika Kreatif Indonesia · smarthub.logikraf.id
          </div>
        </div>
      </footer>
    </div>
  )
}
