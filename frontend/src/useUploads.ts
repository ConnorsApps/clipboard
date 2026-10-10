import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { createUpload, type FileInfo, type UploadHandle } from './api'
import { fileKind } from './format'

const MAX_PARALLEL = 3

export interface PendingUpload {
  key: string
  file: File
  // Object URL, so images show before the server has them
  previewUrl?: string
  progress: number
  status: 'queued' | 'uploading' | 'error' | 'done'
  error?: string
  fileId?: string
}

let nextKey = 0

// Upload queue. A finished upload stays until the server lists it; its preview becomes the thumbnail.
export function useUploads(files: FileInfo[] | null) {
  const [uploads, setUploads] = useState<PendingUpload[]>([])
  const [thumbs, setThumbs] = useState<Record<string, string>>({})
  // Kept after an error so Retry resumes
  const handles = useRef(new Map<string, UploadHandle>())
  // In flight. A ref, not state, so a lagging render can't start an upload twice
  const active = useRef(new Set<string>())

  const patch = useCallback((key: string, changes: Partial<PendingUpload>) => {
    setUploads((list) => list.map((u) => (u.key === key ? { ...u, ...changes } : u)))
  }, [])

  // Start queued uploads, oldest first
  useEffect(() => {
    const slots = MAX_PARALLEL - active.current.size
    if (slots <= 0) return
    const ready = uploads
      .filter((u) => u.status === 'queued' && !active.current.has(u.key))
      .reverse()
      .slice(0, slots)
    for (const u of ready) {
      active.current.add(u.key)
      let handle = handles.current.get(u.key)
      if (!handle) {
        handle = createUpload(u.file, {
          onProgress: (progress) => patch(u.key, { progress }),
          onSuccess: (fileId) => {
            active.current.delete(u.key)
            handles.current.delete(u.key)
            patch(u.key, { status: 'done', progress: 1, fileId })
          },
          onError: (err) => {
            active.current.delete(u.key)
            patch(u.key, { status: 'error', error: err.message })
          },
        })
        handles.current.set(u.key, handle)
      }
      patch(u.key, { status: 'uploading', error: undefined })
      handle.start()
    }
  }, [uploads, patch])

  // Hand finished uploads over to the file list
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

  // Free previews of deleted files
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

  // On logout: stop uploads, free previews
  const latest = useRef({ uploads, thumbs })
  useEffect(() => {
    latest.current = { uploads, thumbs }
  })
  useEffect(() => {
    const pending = handles.current
    return () => {
      for (const handle of pending.values()) handle.abort()
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

  // Cancel or drop; deletes the partial upload
  const dismiss = useCallback((key: string) => {
    handles.current.get(key)?.abort()
    handles.current.delete(key)
    active.current.delete(key)
    const upload = latest.current.uploads.find((u) => u.key === key)
    if (upload?.previewUrl) URL.revokeObjectURL(upload.previewUrl)
    setUploads((list) => list.filter((u) => u.key !== key))
  }, [])

  // Until the hand-over runs: no duplicate tile, and reuse the local preview
  const fileIds = useMemo(() => new Set(files?.map((f) => f.id)), [files])
  const visibleUploads = useMemo(
    () => uploads.filter((u) => !(u.fileId && fileIds.has(u.fileId))),
    [uploads, fileIds],
  )
  const allThumbs = useMemo(() => {
    const merged = { ...thumbs }
    for (const u of uploads) {
      if (u.fileId && u.previewUrl && !merged[u.fileId]) merged[u.fileId] = u.previewUrl
    }
    return merged
  }, [thumbs, uploads])

  return { uploads: visibleUploads, thumbs: allThumbs, add, retry, dismiss }
}
