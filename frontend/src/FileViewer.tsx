import { useEffect, useRef, useState, type ReactNode } from 'react'
import { deleteFile, fetchFileBlob, fileUrl, type FileInfo } from './api'
import { canCopyImages, copyImage } from './clipboard'
import { extension, fileKind, formatRelative, formatSize } from './format'
import { CloseIcon, CopyIcon, DownloadIcon, FileIcon, ShareIcon, TrashIcon } from './icons'
import type { ToastOptions } from './Toast'

// Files up to this size are fetched up front so Copy and Share can run inside the tap itself,
// which iOS Safari requires
const MAX_PREFETCH_BYTES = 50 * 1024 * 1024
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
  const prefetch = file.size <= MAX_PREFETCH_BYTES && (canCopy || canShare || kind === 'text')

  const [blob, setBlob] = useState<Blob | null>(null)
  const [text, setText] = useState<string | null>(null)
  const [previewFailed, setPreviewFailed] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const blobPromise = useRef<Promise<Blob> | null>(null)
  const closeRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!prefetch) return
    const controller = new AbortController()
    const promise = fetchFileBlob(file.id, controller.signal)
    blobPromise.current = promise
    promise
      .then(async (b) => {
        setBlob(b)
        if (kind === 'text' && b.size <= MAX_TEXT_PREVIEW_BYTES) setText(await b.text())
      })
      .catch(() => {})
    return () => controller.abort()
  }, [file.id, kind, prefetch])

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
      await copyImage(blob ?? blobPromise.current ?? fetchFileBlob(file.id))
      showToast('Image copied')
    } catch {
      showToast(`Couldn't copy the image. ${isTouch ? 'Long-press' : 'Right-click'} it to copy instead.`, {
        tone: 'error',
      })
    }
  }

  const handleShare = async () => {
    if (!blob) return
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
        {canShare && prefetch && (
          <button onClick={handleShare} disabled={!blob} className={`${actionClass} bg-white/10 hover:bg-white/20`}>
            <ShareIcon />
            {blob ? 'Share' : 'Loading…'}
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
