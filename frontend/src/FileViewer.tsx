import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { deleteFile, fetchFileBlob, fileUrl, type FileInfo } from './api'
import { canCopyImages, copyImage } from './clipboard'
import { extension, fileKind, formatRelative, formatSize } from './format'
import { CloseIcon, CopyIcon, DownloadIcon, FileIcon, ShareIcon, TrashIcon } from './icons'
import type { ToastOptions } from './Toast'

// iOS allows Copy/Share only within the tap, so the file must already be in memory. Images load
// on open (usually cached); other files on the first Share tap, shared on the second.
const MAX_SHARE_BYTES = 50 * 1024 * 1024
const MAX_TEXT_PREVIEW_BYTES = 1024 * 1024

const isTouch = window.matchMedia('(pointer: coarse)').matches

interface FileViewerProps {
  file: FileInfo
  thumb?: string
  onClose: () => void
  onDeleted: () => void
  showToast: (message: string, options?: ToastOptions) => void
}

function FileViewer({ file, thumb, onClose, onDeleted, showToast }: FileViewerProps) {
  const kind = fileKind(file.type, file.name)
  const [canShare] = useState(
    () =>
      typeof navigator.canShare === 'function' &&
      navigator.canShare({ files: [new File([''], file.name, { type: file.type || 'application/octet-stream' })] }),
  )
  const canCopy = kind === 'image' && canCopyImages()
  const shareable = canShare && file.size <= MAX_SHARE_BYTES
  const loadOnOpen =
    (kind === 'image' && (canCopy || shareable)) || (kind === 'text' && file.size <= MAX_TEXT_PREVIEW_BYTES)

  const [blob, setBlob] = useState<Blob | null>(null)
  const [loading, setLoading] = useState(false)
  const [text, setText] = useState<string | null>(null)
  const [previewFailed, setPreviewFailed] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const blobPromise = useRef<Promise<Blob> | null>(null)
  const abortLoad = useRef<AbortController | null>(null)
  const closeRef = useRef<HTMLButtonElement>(null)

  const loadBlob = useCallback((): Promise<Blob> => {
    if (!blobPromise.current) {
      const controller = new AbortController()
      abortLoad.current = controller
      setLoading(true)
      // Just uploaded: still in memory
      const promise = thumb
        ? fetch(thumb, { signal: controller.signal }).then((r) => r.blob())
        : fetchFileBlob(file.id, controller.signal)
      blobPromise.current = promise
      promise
        .then(setBlob)
        .catch(() => {
          blobPromise.current = null
        })
        .finally(() => setLoading(false))
    }
    return blobPromise.current
  }, [file.id, thumb])

  useEffect(() => {
    if (loadOnOpen) loadBlob()
  }, [loadOnOpen, loadBlob])

  useEffect(() => () => abortLoad.current?.abort(), [])

  useEffect(() => {
    if (kind === 'text' && blob && blob.size <= MAX_TEXT_PREVIEW_BYTES) {
      blob.text().then(setText, () => {})
    }
  }, [kind, blob])

  // Esc closes; lock page scroll underneath while open
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    const overflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    closeRef.current?.focus()
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = overflow
    }
  }, [onClose])

  useEffect(() => {
    if (!confirmDelete) return
    const timer = window.setTimeout(() => setConfirmDelete(false), 3000)
    return () => window.clearTimeout(timer)
  }, [confirmDelete])

  const handleCopy = async () => {
    try {
      await copyImage(blob ?? loadBlob())
      showToast('Image copied')
    } catch {
      showToast(`Couldn't copy the image. ${isTouch ? 'Long-press' : 'Right-click'} it to copy instead.`, {
        tone: 'error',
      })
    }
  }

  const handleShare = async () => {
    if (!blob) {
      loadBlob().catch(() => showToast("Couldn't load the file to share", { tone: 'error' }))
      return
    }
    try {
      await navigator.share({ files: [new File([blob], file.name, { type: blob.type || file.type })] })
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        showToast('Sharing failed', { tone: 'error' })
      }
    }
  }

  const handleDelete = async () => {
    if (!confirmDelete) {
      setConfirmDelete(true)
      return
    }
    setDeleting(true)
    try {
      await deleteFile(file.id)
      onDeleted()
    } catch {
      setDeleting(false)
      setConfirmDelete(false)
      showToast('Delete failed', { tone: 'error' })
    }
  }

  let preview: ReactNode
  if (kind === 'image' && !previewFailed) {
    preview = (
      <img
        src={thumb ?? fileUrl(file.id)}
        alt={file.name}
        onError={() => setPreviewFailed(true)}
        className="max-w-full max-h-full object-contain rounded-lg shadow-2xl"
      />
    )
  } else if (kind === 'video' && !previewFailed) {
    preview = (
      <video
        src={fileUrl(file.id)}
        controls
        playsInline
        preload="metadata"
        onError={() => setPreviewFailed(true)}
        className="max-w-full max-h-full rounded-lg bg-black"
      />
    )
  } else if (kind === 'audio' && !previewFailed) {
    preview = <audio src={fileUrl(file.id)} controls onError={() => setPreviewFailed(true)} className="w-full max-w-md" />
  } else if (text !== null) {
    preview = (
      <pre className="w-full max-w-3xl max-h-full overflow-auto rounded-xl bg-dark-900 border border-white/10 p-4 text-sm text-gray-200 whitespace-pre-wrap break-words select-text">
        {text}
      </pre>
    )
  } else {
    const ext = extension(file.name).toUpperCase()
    preview = (
      <div className="flex flex-col items-center gap-3 text-center text-gray-400">
        <FileIcon className="w-16 h-16 text-gray-500" />
        {ext && <span className="text-sm font-bold tracking-wide text-gray-300">{ext}</span>}
        <span className="text-sm">No preview available</span>
      </div>
    )
  }

  const actionClass =
    'flex-1 md:flex-none flex flex-col md:flex-row items-center justify-center gap-1 md:gap-2 px-3 md:px-5 py-2.5 rounded-xl text-xs md:text-sm font-semibold transition active:scale-95 disabled:opacity-40'

  return (
    <div role="dialog" aria-modal="true" aria-label={file.name} className="fixed inset-0 z-50 flex flex-col bg-dark-950/95 backdrop-blur-sm overscroll-contain">
      <header className="flex items-center gap-3 px-4 pt-[calc(0.75rem+env(safe-area-inset-top))] pb-3">
        <div className="min-w-0 flex-1">
          <p className="font-semibold truncate">{file.name}</p>
          <p className="text-xs text-gray-400">
            {formatSize(file.size)} · {formatRelative(file.uploadedAt)}
          </p>
        </div>
        <button
          ref={closeRef}
          onClick={onClose}
          aria-label="Close"
          className="p-2.5 rounded-full bg-white/10 hover:bg-white/20 text-white"
        >
          <CloseIcon />
        </button>
      </header>

      <div
        className="flex-1 min-h-0 flex items-center justify-center px-4 pb-2"
        onClick={(e) => {
          if (e.target === e.currentTarget) onClose()
        }}
      >
        {preview}
      </div>

      <footer className="flex justify-center gap-2 px-3 pt-2 pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
        {canCopy && (
          <button onClick={handleCopy} className={`${actionClass} bg-white/10 hover:bg-white/20`}>
            <CopyIcon />
            Copy
          </button>
        )}
        {shareable && (
          <button onClick={handleShare} disabled={loading} className={`${actionClass} bg-white/10 hover:bg-white/20`}>
            <ShareIcon />
            {loading ? 'Preparing…' : 'Share'}
          </button>
        )}
        <a href={fileUrl(file.id)} download={file.name} className={`${actionClass} bg-white/10 hover:bg-white/20`}>
          <DownloadIcon />
          Download
        </a>
        <button
          onClick={handleDelete}
          disabled={deleting}
          className={`${actionClass} ${confirmDelete ? 'bg-red-700 text-white' : 'bg-white/10 hover:bg-red-900/60 text-red-300'}`}
        >
          <TrashIcon />
          {confirmDelete ? 'Confirm' : 'Delete'}
        </button>
      </footer>
    </div>
  )
}

export default FileViewer
