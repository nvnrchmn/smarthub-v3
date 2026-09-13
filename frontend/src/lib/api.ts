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

export const roleLabel: Record<string, string> = {
  RESIDENT: 'Warga',
  SECRETARY: 'Sekretaris',
  TREASURER: 'Bendahara',
  TENANT_MANAGER: 'Pengelola',
}
