import { useAuthStore } from '../stores/auth'

const BASE_URL = import.meta.env.VITE_API_URL

export class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.status = status
  }
}

export async function tryRefresh() {
  const auth = useAuthStore()
  try {
    const res = await fetch(`${BASE_URL}/api/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!res.ok) {
      auth.clear()
      return false
    }
    const data = await res.json()
    auth.setAccessToken(data.access_token)
    return true
  } catch {
    auth.clear()
    return false
  }
}

async function request(path, { method = 'GET', body, skipRetry = false } = {}) {
  const auth = useAuthStore()

  const headers = {}
  let payload = body
  if (body !== undefined && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  if (auth.accessToken) {
    headers['Authorization'] = `Bearer ${auth.accessToken}`
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: payload,
    credentials: 'include',
  })

  if (res.status === 401 && !skipRetry) {
    const refreshed = await tryRefresh()
    if (refreshed) {
      return request(path, { method, body, skipRetry: true })
    }
  }

  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new ApiError(res.status, data.error || res.statusText)
  }

  if (res.status === 204) {
    return null
  }
  return res.json()
}

export const api = {
  get: (path) => request(path),
  post: (path, body) => request(path, { method: 'POST', body }),
  put: (path, body) => request(path, { method: 'PUT', body }),
}
