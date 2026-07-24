const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8000'

export async function getHealth(): Promise<unknown> {
  const res = await fetch(`${BASE_URL}/health`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}
