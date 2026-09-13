// Klien API tipis: token disimpan di localStorage, semua panggilan lewat /api
// (vhost meneruskan ke service Go).
const TOKEN_KEY = 'smarthub.token'

export type Me = { user_id: string; tenant_id: string; roles: string[]; status: string }

export const auth = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = auth.get()
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
      body: JSON.stringify({ email, password }),
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
  me: () => request<ResidentProfile>('/census/me'),
  submitMe: (body: Record<string, unknown>) =>
    request<ResidentProfile>('/census/me', { method: 'POST', body: JSON.stringify(body) }),
  queue: (status = 'UNVERIFIED') =>
    request<ResidentProfile[]>(`/census?status=${encodeURIComponent(status)}`),
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
