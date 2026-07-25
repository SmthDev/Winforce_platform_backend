const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

export interface LoginResponse {
  user?: { id?: string; email?: string; name?: string }
  session?: { id?: string; expires_at?: string }
}

export interface ProfileResponse {
  id: string
  email: string
  first_name: string
  last_name: string
}

export async function getHealth(): Promise<unknown> {
  const res = await fetch(`${BASE_URL}/health`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}


export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await fetch(`${BASE_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ email, password }),
  })

  if (!res.ok) {
    throw new Error(await readError(res))
  }
  return readBody<LoginResponse>(res)
}


export async function getProfile(): Promise<ProfileResponse> {
  const res = await fetch(`${BASE_URL}/profile`, {
    credentials: 'include',
  })

  if (!res.ok) {
    throw new Error(await readError(res))
  }
  return readBody<ProfileResponse>(res)
}

export async function updateProfileName(firstName: string, lastName: string): Promise<ProfileResponse> {
  const res = await fetch(`${BASE_URL}/profile`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ first_name: firstName, last_name: lastName }),
  })

  if (!res.ok) {
    throw new Error(await readError(res))
  }
  return readBody<ProfileResponse>(res)
}

async function readBody<T>(res: Response): Promise<T> {
  const text = await res.text()
  if (!text) return {} as T
  try {
    return JSON.parse(text) as T
  } catch {
    return {} as T
  }
}

async function readError(res: Response): Promise<string> {
  if (res.status === 401 || res.status === 403) return 'Invalid email or password.'
  if (res.status === 429) return 'Too many attempts. Please wait a moment.'

  const body = await readBody<{ message?: string; error?: string }>(res)
  return body.message ?? body.error ?? `HTTP ${res.status}`
}
