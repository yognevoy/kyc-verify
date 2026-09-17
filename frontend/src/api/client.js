import { fetchEventSource } from '@microsoft/fetch-event-source'

import { useAuthStore } from '../stores/auth'

const BASE_URL = import.meta.env.VITE_API_URL

export class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.status = status
  }
}

class RetriableError extends Error {}

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

function authHeaders() {
  const auth = useAuthStore()
  return auth.accessToken ? { Authorization: `Bearer ${auth.accessToken}` } : {}
}

async function request(path, { method = 'GET', body, skipRetry = false } = {}) {
  const headers = { ...authHeaders() }
  let payload = body
  if (body !== undefined && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
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

export async function subscribeToStream(path, onEvent, { signal } = {}) {
  const headers = authHeaders()

  await fetchEventSource(`${BASE_URL}${path}`, {
    headers,
    credentials: 'include',
    signal,
    async onopen(response) {
      if (response.ok) {
        return
      }
      if (response.status === 401 && (await tryRefresh())) {
        Object.assign(headers, authHeaders())
        throw new RetriableError()
      }
      const data = await response.json().catch(() => ({}))
      throw new ApiError(response.status, data.error || response.statusText)
    },
    onmessage(event) {
      onEvent(JSON.parse(event.data))
    },
    onerror(err) {
      if (err instanceof ApiError) {
        throw err
      }
    },
  })
}
