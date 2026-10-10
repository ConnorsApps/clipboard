import { useCallback, useEffect, useRef, useState } from 'react'
import { uploadFile, type FileInfo } from './api'
import { fileKind } from './format'

const MAX_PARALLEL = 3

export interface PendingUpload {
  key: string
  file: File
  // Local object URL so images show instantly, before the server has them
  previewUrl?: string
  progress: number
  status: 'queued' | 'uploading' | 'error' | 'done'
  error?: string
  fileId?: string
}

let nextKey = 0

// useUploads queues files for upload and tracks their progress. A finished upload stays in the
// list until the server's file list includes it, then its local preview becomes the thumbnail.
export function useUploads(files: FileInfo[] | null) {
  const [uploads, setUploads] = useState<PendingUpload[]>([])
  const [thumbs, setThumbs] = useState<Record<string, string>>({})
  const cancels = useRef(new Map<string, () => void>())

  const patch = useCallback((key: string, changes: Partial<PendingUpload>) => {
    setUploads((list) => list.map((u) => (u.key === key ? { ...u, ...changes } : u)))
  }, [])

  // Start queued uploads while under the parallel limit
  useEffect(() => {
    const active = uploads.filter((u) => u.status === 'uploading').length
    const ready = uploads.filter((u) => u.status === 'queued').slice(0, Math.max(0, MAX_PARALLEL - active))
    for (const u of ready) {
      patch(u.key, { status: 'uploading', progress: 0, error: undefined })
      const cancel = uploadFile(u.file, {
        onProgress: (progress) => patch(u.key, { progress }),
        onSuccess: (fileId) => {
          cancels.current.delete(u.key)
          patch(u.key, { status: 'done', progress: 1, fileId })
        },
        onError: (err) => {
          cancels.current.delete(u.key)
          patch(u.key, { status: 'error', error: err.message })
        },
      })
      cancels.current.set(u.key, cancel)
    }
  }, [uploads, patch])

  // Hand finished uploads over to the server's file list
  useEffect(() => {
    if (!files) return
    const ids = new Set(files.map((f) => f.id))
    const arrived = uploads.filter((u) => u.status === 'done' && u.fileId && ids.has(u.fileId))
    if (arrived.length === 0) return

    setThumbs((current) => {
      const next = { ...current }
      for (const u of arrived) {
        if (u.previewUrl) next[u.fileId!] = u.previewUrl
      }
      return next
    })
    const keys = new Set(arrived.map((u) => u.key))
    setUploads((list) => list.filter((u) => !keys.has(u.key)))
  }, [files, uploads])

  // Release previews of files that have been deleted
  useEffect(() => {
    if (!files) return
    const ids = new Set(files.map((f) => f.id))
    const stale = Object.keys(thumbs).filter((id) => !ids.has(id))
    if (stale.length === 0) return
    for (const id of stale) URL.revokeObjectURL(thumbs[id])
    setThumbs((current) => {
      const next = { ...current }
      for (const id of stale) delete next[id]
      return next
    })
  }, [files, thumbs])

  // Stop uploads and free previews when leaving (e.g. logout)
  const latest = useRef({ uploads, thumbs })
  useEffect(() => {
    latest.current = { uploads, thumbs }
  })
  useEffect(() => {
    const pending = cancels.current
    return () => {
      for (const cancel of pending.values()) cancel()
      for (const u of latest.current.uploads) if (u.previewUrl) URL.revokeObjectURL(u.previewUrl)
      for (const url of Object.values(latest.current.thumbs)) URL.revokeObjectURL(url)
    }
  }, [])

  const add = useCallback((newFiles: File[]) => {
    const items: PendingUpload[] = newFiles.map((file) => ({
      key: `upload-${nextKey++}`,
      file,
      previewUrl: fileKind(file.type, file.name) === 'image' ? URL.createObjectURL(file) : undefined,
      progress: 0,
      status: 'queued',
    }))
    setUploads((list) => [...items, ...list])
  }, [])

  const retry = useCallback((key: string) => patch(key, { status: 'queued', error: undefined }), [patch])

  const dismiss = useCallback((key: string) => {
    cancels.current.get(key)?.()
    cancels.current.delete(key)
    const upload = latest.current.uploads.find((u) => u.key === key)
    if (upload?.previewUrl) URL.revokeObjectURL(upload.previewUrl)
    setUploads((list) => list.filter((u) => u.key !== key))
  }, [])

  return { uploads, thumbs, add, retry, dismiss }
}
