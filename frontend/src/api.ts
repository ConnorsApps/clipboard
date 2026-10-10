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

// Retries once on 401/503/network errors: the token store can lag or blip
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

// For <img>, <video> and download links, which can't send an Authorization header
export function fileUrl(id: string): string {
  return `/api/files/${encodeURIComponent(id)}?token=${encodeURIComponent(getToken() ?? '')}`
}

export async function fetchFileBlob(id: string, signal?: AbortSignal): Promise<Blob> {
  // Same URL as the thumbnail, so usually cached
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
  // Starts, or resumes after an error
  start: () => void
  // Stops and deletes the partial upload
  abort: () => void
}

export function createUpload(file: File, callbacks: UploadCallbacks): UploadHandle {
  const metadata: Record<string, string> = { filename: file.name }
  if (file.type) {
    metadata.filetype = file.type
  }

  const upload = new tus.Upload(file, {
    endpoint: '/api/uploads/',
    chunkSize: 5 * 1024 * 1024,
    retryDelays: [0, 1000, 3000, 5000, 10000],
    // Resume via start(); stored fingerprints would make duplicate files share one upload
    storeFingerprintForResuming: false,
    metadata,
    headers: { Authorization: `Bearer ${getToken()}` },
    onProgress: (sent, total) => callbacks.onProgress(total ? sent / total : 0),
    // tusd names the file after the upload ID, the URL's last segment
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
