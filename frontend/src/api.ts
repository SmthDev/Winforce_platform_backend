const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

export interface LoginResponse {
  user?: { id?: string; email?: string; name?: string }
  session?: { id?: string; expires_at?: string }
}

export async function getHealth(): Promise<unknown> {
  const res = await fetch(`${BASE_URL}/health`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

/**
 * Signs in through POST /login, the backend alias for Limen's
 * credential-password sign-in. The session comes back as a cookie, so the
 * request must be credentialed.
 */
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
