// Klien API tipis: token disimpan di localStorage, semua panggilan lewat /api
// (vhost meneruskan ke service Go).
const TOKEN_KEY = 'smarthub.token'
// Token superadmin terpisah: halaman superadmin tidak boleh memakai token tenant
// (dan sebaliknya), kalau tidak semua panggilan /api/superadmin/* akan 401.
const SUPERADMIN_TOKEN_KEY = 'smarthub.superadmin.token'

export type Me = { user_id: string; tenant_id: string; roles: string[]; status: string }

export type Health = {
  status: string
  database: boolean
  redis: boolean
  service: string
}

export const auth = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
  // Keluar: minta server MENCABUT token ini, lalu bersihkan token lokal.
  //
  // Token lokal selalu dihapus — juga saat pencabutan gagal (mis. Redis mati) —
  // supaya pengguna benar-benar keluar dari perangkat ini. Kembaliannya
  // memberi tahu apakah server benar-benar mencabut, bukan sekadar menghapus
  // di sisi peramban.
  logout: async (): Promise<boolean> => {
    const token = localStorage.getItem(TOKEN_KEY)
    let dicabut = false
    if (token) {
      try {
        const r = await request<{ dicabut: boolean }>('/auth/logout', { method: 'POST' })
        dicabut = r.dicabut
      } catch {
        dicabut = false
      }
    }
    localStorage.removeItem(TOKEN_KEY)
    return dicabut
  },
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  mode: 'tenant' | 'superadmin' = 'tenant',
): Promise<T> {
  const token = mode === 'superadmin' ? localStorage.getItem(SUPERADMIN_TOKEN_KEY) : auth.get()
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init.headers || {}),
    },
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : {}
  if (!res.ok) throw new ApiError(res.status, data.error || `Permintaan gagal (${res.status})`)
  return data as T
}

export const api = {
  login: (email: string, password: string) =>
    request<{ token: string; role: string; name: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ identifier: email, password }),
    }),
  me: () => request<Me>('/me'),
  accept: (body: { token: string; otp: string; full_name: string; password: string }) =>
    request<{ token: string }>('/auth/invite/accept', { method: 'POST', body: JSON.stringify(body) }),
  resendOtp: (token: string) =>
    request<{ status: string }>('/auth/invite/resend-otp', {
      method: 'POST',
      body: JSON.stringify({ token }),
    }),
  invite: (body: { email: string; phone: string; role: string }) =>
    request<{ invite_link: string; expires_in_days: number }>('/invite', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
}

// ---- Sensus (Fase 2) ----
export type ResidentProfile = {
  id: string
  full_name: string
  nik_last4: string
  family_role: string
  birth_place?: string
  birth_date?: string
  gender?: string
  religion?: string
  marital_status?: string
  occupation?: string
  education?: string
  verification_status: string
  rejection_reason?: string
  lifecycle_status: string
  has_ktp: boolean
  has_kk: boolean
  house_unit?: string
  occupancy_type?: string
  is_primary_payer?: boolean
}

export const CENSUS_STAFF_ROLES = ['TENANT_MANAGER', 'SECRETARY']

export const census = {
  list: async () =>
    (await request<{ items: ResidentProfile[] | null }>('/census/residents')).items ?? [],
  me: () => request<ResidentProfile>('/census/me'),
  submitMe: (body: Record<string, unknown>) =>
    request<ResidentProfile>('/census/me', { method: 'POST', body: JSON.stringify(body) }),
  // Backend membalas { items, jumlah }; UI memakai array-nya saja.
  queue: async (status = 'UNVERIFIED') => {
    const r = await request<{ items: ResidentProfile[]; jumlah: number }>(
      `/census?status=${encodeURIComponent(status)}`,
    )
    return Array.isArray(r) ? (r as unknown as ResidentProfile[]) : r.items || []
  },
  detail: (id: string) => request<ResidentProfile>(`/census/${id}`),
  reveal: (id: string) =>
    request<{ nik: string; kk_number: string }>(`/census/${id}/reveal`, { method: 'POST' }),
  verify: (id: string, status: string, reason = '') =>
    request<{ status: string }>(`/census/${id}/verify`, {
      method: 'POST',
      body: JSON.stringify({ status, reason }),
    }),
  upload: async (id: string, kind: 'KTP' | 'KK', file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    const token = auth.get()
    const res = await fetch(`/api/census/${id}/documents/${kind}`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      body: fd,
    })
    if (!res.ok) throw new Error(((await res.json().catch(() => ({}))) as { error?: string }).error || 'unggah gagal')
    return (await res.json()) as { key: string }
  },
  documentBlobUrl: async (id: string, kind: 'KTP' | 'KK') => {
    const token = auth.get()
    const res = await fetch(`/api/census/${id}/documents/${kind}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    })
    if (!res.ok) throw new Error('dokumen tidak bisa dibuka')
    return URL.createObjectURL(await res.blob())
  },
}

export const roleLabelCensus: Record<string, string> = {
  HEAD_OF_FAMILY: 'Kepala Keluarga',
  SPOUSE: 'Istri/Suami',
  CHILD: 'Anak',
  OTHER: 'Lainnya',
}

export const roleLabel: Record<string, string> = {
  RESIDENT: 'Warga',
  SECRETARY: 'Sekretaris',
  TREASURER: 'Bendahara',
  TENANT_MANAGER: 'Pengelola',
}
// ---- Kelola rumah & kartu keluarga (Fase 2 lanjutan) ----
export type HouseUnit = {
  id: string
  block: string
  unit_number: string
  occupancy_status: string
  notes?: string
  occupant_count?: number
  primary_occupant?: string
}

export type FamilyMember = {
  id: string
  full_name: string
  family_role: string
  verification_status: string
  lifecycle_status: string
}

export type FamilyCard = {
  id: string
  number_last4: string
  has_file: boolean
  member_count: number
  members: FamilyMember[]
}

export const UNIT_STATUS_LABEL: Record<string, string> = {
  OCCUPIED: 'Dihuni',
  VACANT: 'Kosong',
  RENOVATION: 'Renovasi',
}

export const houses = {
  list: async () => {
    const r = await request<{ items: HouseUnit[]; jumlah: number }>('/houses')
    return r.items ?? []
  },
  create: (body: { block: string; number: string; notes?: string }) =>
    request<HouseUnit>('/houses', { method: 'POST', body: JSON.stringify(body) }),
  update: (id: string, body: { block?: string; number?: string; status?: string; notes?: string }) =>
    request<HouseUnit>(`/houses/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  remove: (id: string) => request<{ status: string }>(`/houses/${id}`, { method: 'DELETE' }),
}

export const occupancy = {
  assign: (profileId: string, body: { house_unit_id: string; occupancy_type: string; is_primary_payer: boolean }) =>
    request<{ status: string }>(`/census/${profileId}/occupancy`, { method: 'POST', body: JSON.stringify(body) }),
  end: (profileId: string) =>
    request<{ status: string }>(`/census/${profileId}/occupancy/end`, { method: 'POST' }),
}

export const familyCards = {
  list: async () => {
    const r = await request<{ items: FamilyCard[]; jumlah: number }>('/family-cards')
    return r.items ?? []
  },
  create: (kkNumber: string) =>
    request<{ id: string }>('/family-cards', { method: 'POST', body: JSON.stringify({ kk_number: kkNumber }) }),
  addMember: (cardId: string, residentId: string) =>
    request<{ status: string }>(`/family-cards/${cardId}/members`, {
      method: 'POST',
      body: JSON.stringify({ resident_id: residentId }),
    }),
  removeMember: (cardId: string, residentId: string) =>
    request<{ status: string }>(`/family-cards/${cardId}/members/${residentId}`, { method: 'DELETE' }),
  candidates: async () => {
    const r = await request<{ items: FamilyMember[]; jumlah: number }>('/family-cards/candidates')
    return r.items ?? []
  },
}

// ---- Tagihan iuran (Fase 3) ----
export type InvoiceItem = { label: string; amount: number; kind: string }
export type Invoice = {
  id: string
  house_unit_id: string
  house_unit?: string
  invoice_number: string
  period: string
  base_amount: number
  arrears_amount: number
  total_amount: number
  status: string
  due_date: string
  paid_at?: string
  items: InvoiceItem[]
}
export type FeeItem = {
  id: string
  code: string
  label: string
  amount: number
  applies_to: string
  is_active: boolean
}
export type CashLedger = {
  id: string
  source_type: string
  transaction_type: string
  amount: number
  notes?: string
  invoice_number?: string
  approved_at?: string
  ledger_date: string
  transfer_group?: string
}
export type SaldoKas = {
  bank_gateway: number
  petty_cash: number
  bank_account: number
  menunggu_persetujuan: number
}

export const STATUS_TAGIHAN: Record<string, string> = {
  UNPAID: 'Belum bayar',
  PAID: 'Lunas',
  VOID: 'Dibatalkan',
}

export const SUMBER_KAS: Record<string, string> = {
  BANK_GATEWAY: 'Kas Bank / Gateway',
  PETTY_CASH_TREASURER: 'Kas Fisik Bendahara',
  BANK_ACCOUNT: 'Rekening Paguyuban',
}

export const SASARAN_IURAN: Record<string, string> = {
  ALL: 'Semua rumah',
  OCCUPIED: 'Hanya rumah dihuni',
  VACANT: 'Hanya rumah kosong',
}

export const rupiah = (v: number) =>
  'Rp' + (v ?? 0).toLocaleString('id-ID', { maximumFractionDigits: 0 })

export const billing = {
  invoices: async (status = '') => {
    const r = await request<{ items: Invoice[]; jumlah: number; total_belum_lunas: number }>(
      `/billing/invoices${status ? `?status=${encodeURIComponent(status)}` : ''}`,
    )
    return { items: r.items ?? [], jumlah: r.jumlah ?? 0, total: r.total_belum_lunas ?? 0 }
  },
  invoice: (id: string) => request<Invoice>(`/billing/invoices/${id}`),
  qrisBuat: (id: string) =>
    request<{ reference_id: string; qr_string: string; amount: number; expires_at: string }>(
      `/billing/invoices/${id}/qris`,
      { method: 'POST' },
    ),
  qrisCek: (id: string) =>
    request<{ invoice: Invoice; status_gateway: string }>(`/billing/invoices/${id}/qris`),
  tunai: (id: string, nominal: number, catatan: string) =>
    request<Invoice>(`/billing/invoices/${id}/cash`, {
      method: 'POST',
      body: JSON.stringify({ nominal, catatan }),
    }),
  generate: (periode: string) =>
    request<{ periode: string; dibuat: number; dilewati: number; total_unit: number }>(
      '/billing/generate',
      { method: 'POST', body: JSON.stringify({ periode }) },
    ),
  feeItems: async () => {
    const r = await request<{ items: FeeItem[] }>('/billing/fee-items')
    return r.items ?? []
  },
  simpanIuran: (f: Partial<FeeItem>) =>
    request<{ id: string }>('/billing/fee-items', { method: 'POST', body: JSON.stringify(f) }),
  pengaturan: () => request<{ billing_day: number; due_days: number }>('/billing/settings'),
  simpanPengaturan: (billing_day: number, due_days: number) =>
    request<{ status: string }>('/billing/settings', {
      method: 'POST',
      body: JSON.stringify({ billing_day, due_days }),
    }),
  kas: async () => {
    const r = await request<{ items: CashLedger[]; saldo: SaldoKas }>('/billing/ledger')
    return { items: r.items ?? [], saldo: r.saldo }
  },
  setor: (nominal: number, catatan: string, bukti = '') =>
    request<{ transfer_group: string }>('/billing/ledger/deposit', {
      method: 'POST',
      body: JSON.stringify({ nominal, catatan, bukti }),
    }),
  setujuiSetor: (grup: string) =>
    request<{ status: string }>(`/billing/ledger/deposit/${grup}/approve`, { method: 'POST' }),
}

export const BILLING_STAFF_ROLES = ['TENANT_MANAGER', 'TREASURER']

// ---- Superadmin (Fase 4) ----
export type SuperadminMe = { id: string; email: string; full_name: string; role: string }
export type TenantList = { id: string; name: string; slug: string; status: string; created_at: string }
export type AuditLogEntry = { id: number; tenant_id: string; actor_id?: string; action: string; entity?: string; entity_id?: string; detail?: any; created_at: string }

export const superadmin = {
  TOKEN_KEY: SUPERADMIN_TOKEN_KEY,
  get: () => localStorage.getItem(SUPERADMIN_TOKEN_KEY),
  set: (t: string) => localStorage.setItem(SUPERADMIN_TOKEN_KEY, t),
  clear: () => localStorage.removeItem(SUPERADMIN_TOKEN_KEY),
  login: async (email: string, password: string) => {
    const r = await request<{ token: string; superadmin: SuperadminMe }>(
      '/superadmin/login',
      { method: 'POST', body: JSON.stringify({ email, password }) },
      'superadmin',
    )
    superadmin.set(r.token)
    return r.superadmin
  },
  me: () => request<SuperadminMe>('/superadmin/me', {}, 'superadmin'),
  // Backend mengembalikan {items, jumlah}; UI hanya butuh arraynya.
  listTenants: async () =>
    (await request<{ items: TenantList[] | null }>('/superadmin/tenants', {}, 'superadmin')).items ?? [],
  auditLog: async () =>
    (await request<{ items: AuditLogEntry[] | null }>('/superadmin/audit-log', {}, 'superadmin')).items ?? [],
  allSettings: async () =>
    (await request<{ items: Record<string, string> | null }>('/superadmin/settings', {}, 'superadmin')).items ?? {},
  resetPassword: (oldPw: string, newPw: string) =>
    request<{ status: string }>(
      '/superadmin/reset-password',
      { method: 'POST', body: JSON.stringify({ old_password: oldPw, new_password: newPw }) },
      'superadmin',
    ),
  getSetting: (key: string) =>
    request<{ key: string; value: string }>(`/superadmin/settings?key=${encodeURIComponent(key)}`, {}, 'superadmin'),
  setSetting: (key: string, value: string) =>
    request<{ status: string }>(
      '/superadmin/settings',
      { method: 'POST', body: JSON.stringify({ key, value }) },
      'superadmin',
    ),
}

// ---- Laporan (pengurus) ----
export type RekapItem = { label: string; kind: string; jumlah: number; amount: number }
export type RekapBulanan = {
  periode: string
  tenant_nama: string
  jumlah_invoice: number
  total_ditagih: number
  total_dibayar: number
  total_tunggakan: number
  jumlah_lunas: number
  jumlah_belum: number
  rincian: RekapItem[] | null
  kas_tunai: number
  kas_qris: number
}
export type TunggakanBaris = {
  unit: string
  kepala_keluarga: string
  telepon: string
  periode_tertua: string
  jumlah_tagihan: number
  total_tunggakan: number
  hari_terlambat: number
  bucket: string
}
export type BucketTunggakan = { label: string; jumlah: number; amount: number }
export type Tunggakan = {
  as_of: string
  jumlah_unit: number
  total: number
  buckets: BucketTunggakan[] | null
  baris: TunggakanBaris[] | null
}

export const laporan = {
  rekap: (periode: string) => request<RekapBulanan>(`/reports/monthly?period=${encodeURIComponent(periode)}`),
  tunggakan: () => request<Tunggakan>('/reports/arrears'),
  kirim: (periode: string) =>
    request<{ status: string; terkirim: number }>('/reports/send', {
      method: 'POST',
      body: JSON.stringify({ periode }),
    }),
  unduh: async (periode: string, format: 'pdf' | 'csv') => {
    const token = auth.get()
    const res = await fetch(`/api/reports/download?format=${format}&period=${encodeURIComponent(periode)}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) throw new ApiError(res.status, 'gagal mengunduh laporan')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `laporan-iuran-${periode}.${format}`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  },
}
