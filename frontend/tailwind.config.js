/** @type {import('tailwindcss').Config} */
// Token mengikuti dokumen UI/UX: palet diambil dari design system logikraf.id
// (brand #0052FF), motion memakai durasi lambat 400-1200 ms.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'sans-serif'],
      },
      colors: {
        canvas: 'var(--bg-canvas)',
        surface: 'var(--bg-surface)',
        subtle: 'var(--bg-subtle)',
        line: 'var(--border-default)',
        ink: 'var(--text-primary)',
        ink2: 'var(--text-secondary)',
        muted: 'var(--text-muted)',
        // Warna identitas (hex, sama di light & dark) supaya modifier opasitas
        // seperti bg-brand/10 bisa dipakai — var() tidak mendukung itu.
        brand: '#0052FF',
        brandhover: '#0041CC',
        'brand-subtle': '#EFF6FF',
        accent: '#10B981',
        ok: '#10B981',
        warn: '#F59E0B',
        danger: '#EF4444',
        info: '#3B82F6',
      },
      transitionDuration: {
        instant: '120ms',
        fast: '240ms',
        base: '400ms',
        slow: '600ms',
        slower: '800ms',
        ambient: '1200ms',
      },
      transitionTimingFunction: {
        standard: 'cubic-bezier(0.4, 0, 0.2, 1)',
        decelerate: 'cubic-bezier(0, 0, 0.2, 1)',
        accelerate: 'cubic-bezier(0.4, 0, 1, 1)',
        spring: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
      },
      keyframes: {
        'rise-in': {
          '0%': { opacity: '0', transform: 'translateY(16px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        'pulse-badge': {
          '0%': { transform: 'scale(1)' },
          '50%': { transform: 'scale(1.04)' },
          '100%': { transform: 'scale(1)' },
        },
      },
      animation: {
        'rise-in': 'rise-in 600ms cubic-bezier(0, 0, 0.2, 1) both',
        shimmer: 'shimmer 1200ms linear infinite',
        'pulse-badge': 'pulse-badge 600ms cubic-bezier(0.34, 1.56, 0.64, 1)',
      },
    },
  },
  plugins: [],
}
