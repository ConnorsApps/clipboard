import * as tus from 'tus-js-client'

export interface FileInfo {
  id: string
  name: string
  type: string
  size: number
  uploadedAt: string
}

const TOKEN_KEY = 'clipboard_token'

export const getToken = () => localStorage.getItem(TOKEN_KEY)
export const setToken = (token: string) => localStorage.setItem(TOKEN_KEY, token)
export const clearToken = () => localStorage.removeItem(TOKEN_KEY)

export class AuthError extends Error {}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export async function login(password: string): Promise<string> {
  const response = await fetch('/api/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!response.ok) {
    const text = await response.text()
    throw new Error(text.trim() || 'Login failed')
  }
  const data = await response.json()
  return data.token
}

// authFetch retries once on 401/503 and network errors: the token store (MongoDB) can be
// briefly unavailable, and a new token may not be visible to every replica yet
async function authFetch(path: string, init: RequestInit = {}, attempts = 2): Promise<Response> {
  for (let attempt = 1; ; attempt++) {
    try {
      const response = await fetch(path, {
        ...init,
        headers: { ...init.headers, Authorization: `Bearer ${getToken()}` },
      })
      if ((response.status === 401 || response.status === 503) && attempt < attempts) {
        await sleep(1000)
        continue
      }
      if (response.status === 401) {
        throw new AuthError('Unauthorized')
      }
      return response
    } catch (err) {
      if (err instanceof AuthError || init.signal?.aborted || attempt >= attempts) {
        throw err
      }
      await sleep(1000)
    }
  }
}

// fileUrl is used for <img>/<video> sources and download links, which can't send an
// Authorization header, so the token rides in the query string
export function fileUrl(id: string): string {
  return `/api/files/${encodeURIComponent(id)}?token=${encodeURIComponent(getToken() ?? '')}`
}

export async function fetchFileBlob(id: string, signal?: AbortSignal): Promise<Blob> {
  // Same URL as the thumbnail so the browser cache can serve it
  const response = await fetch(fileUrl(id), { signal })
  if (!response.ok) {
    throw new Error(`Download failed (${response.status})`)
  }
  return response.blob()
}

export async function deleteFile(id: string): Promise<void> {
  const response = await authFetch(`/api/files/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!response.ok && response.status !== 404) {
    throw new Error(`Delete failed (${response.status})`)
  }
}

export interface UploadCallbacks {
  onProgress: (fraction: number) => void
  onSuccess: (fileId: string) => void
  onError: (error: Error) => void
}

export interface UploadHandle {
  // Starts the upload, or resumes it from where the server has it after an error
  start: () => void
  // Stops the upload and deletes what the server has received so far
  abort: () => void
}

// createUpload prepares a resumable tus upload of file
export function createUpload(file: File, callbacks: UploadCallbacks): UploadHandle {
  const metadata: Record<string, string> = { filename: file.name }
  if (file.type) {
    metadata.filetype = file.type
  }

  const upload = new tus.Upload(file, {
    endpoint: '/api/uploads/',
    chunkSize: 5 * 1024 * 1024,
    retryDelays: [0, 1000, 3000, 5000, 10000],
    // Resuming works through this Upload object (start() again after an error). Fingerprints
    // in localStorage would make two copies of the same file share one server upload.
    storeFingerprintForResuming: false,
    metadata,
    headers: { Authorization: `Bearer ${getToken()}` },
    onProgress: (sent, total) => callbacks.onProgress(total ? sent / total : 0),
    // tusd stores each upload under its upload ID, which is the last segment of the upload URL
    onSuccess: () => callbacks.onSuccess(upload.url?.split('/').pop() ?? ''),
    onError: callbacks.onError,
  })

  return {
    start: () => upload.start(),
    abort: () => {
      upload.abort(true).catch(() => {})
    },
  }
}
