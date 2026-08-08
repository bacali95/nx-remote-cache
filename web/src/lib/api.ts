// Thin fetch wrapper for the admin API. Every request is same-origin
// (served from the same binary under /admin/), so cookies go along for free
// with credentials: "same-origin"; mutating requests also carry the CSRF
// header the server requires (see internal/adminapi/middleware.go).

const CSRF_HEADER = "X-Nxcache-Admin"

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const isMutating = method !== "GET" && method !== "HEAD"
  const headers: Record<string, string> = {}
  if (body !== undefined) headers["Content-Type"] = "application/json"
  if (isMutating) headers[CSRF_HEADER] = "1"

  const res = await fetch(`/admin/api${path}`, {
    method,
    headers,
    credentials: "same-origin",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const data = await res.json()
      if (data?.error) message = data.error
    } catch {
      // response had no JSON body; fall back to statusText
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body ?? {}),
  del: <T>(path: string) => request<T>("DELETE", path),
}

export interface User {
  id: number
  email: string
  createdAt: string
}

export interface Token {
  id: number
  name: string
  scope: "read" | "write"
  createdAt: string
  lastUsedAt?: string
  revokedAt?: string
}

export interface CacheEntry {
  hash: string
  size: number
  modTime: string
}

export interface ListCacheResponse {
  entries: CacheEntry[] | null
  nextCursor?: string
}
