import { memo, useState } from 'react'
import { fileUrl, type FileInfo } from './api'
import { extension, fileKind, formatSize } from './format'
import { CheckIcon, CloseIcon, FileIcon, PhotoIcon, PlayIcon, RetryIcon } from './icons'
import type { PendingUpload } from './useUploads'

interface FileGridProps {
  files: FileInfo[] | null
  uploads: PendingUpload[]
  thumbs: Record<string, string>
  onOpen: (file: FileInfo) => void
  onRetry: (key: string) => void
  onDismiss: (key: string) => void
}

const gridClass = 'grid grid-cols-3 sm:grid-cols-4 lg:grid-cols-3 xl:grid-cols-4 gap-2'

function FileGrid({ files, uploads, thumbs, onOpen, onRetry, onDismiss }: FileGridProps) {
  const count = (files?.length ?? 0) + uploads.length

  return (
    <section>
      <div className="flex items-baseline justify-between px-1 mb-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-400">Images &amp; files</h2>
        {count > 0 && <span className="text-xs text-gray-500 tabular-nums">{count}</span>}
      </div>

      {files === null && uploads.length === 0 ? (
        <div className={gridClass} aria-label="Loading files">
          {[0, 1, 2].map((i) => (
            <div key={i} className="aspect-square rounded-xl bg-dark-800 animate-pulse" />
          ))}
        </div>
      ) : count === 0 ? (
        <EmptyState />
      ) : (
        <div className={gridClass}>
          {uploads.map((upload) => (
            <UploadTile key={upload.key} upload={upload} onRetry={onRetry} onDismiss={onDismiss} />
          ))}
          {files?.map((file) => (
            <FileTile key={file.id} file={file} thumb={thumbs[file.id]} onOpen={onOpen} />
          ))}
        </div>
      )}
    </section>
  )
}

function EmptyState() {
  return (
    <div className="rounded-2xl border-2 border-dashed border-dark-700 px-6 py-10 text-center">
      <PhotoIcon className="w-10 h-10 mx-auto text-gray-600" />
      <p className="mt-3 font-semibold text-gray-300">No images or files yet</p>
      <p className="mt-1 text-sm text-gray-500">
        <span className="md:hidden">Tap Paste, Photos or Files below.</span>
        <span className="hidden md:inline">Paste an image, drop files here, or use the buttons above.</span>
      </p>
    </div>
  )
}

function GenericPreview({ name, type, size }: { name: string; type: string; size: number }) {
  const kind = fileKind(type, name)
  const ext = extension(name).slice(0, 4).toUpperCase()
  return (
    <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 p-2 text-center">
      {kind === 'video' ? (
        <PlayIcon className="w-8 h-8 text-accent-500" />
      ) : (
        <div className="relative">
          <FileIcon className="w-9 h-9 text-gray-500" />
          {ext && (
            <span className="absolute left-1/2 top-[58%] -translate-x-1/2 -translate-y-1/2 text-[8px] font-bold text-gray-300">
              {ext}
            </span>
          )}
        </div>
      )}
      <span className="w-full text-xs text-gray-300 line-clamp-2 break-all leading-tight">{name}</span>
      <span className="text-[10px] text-gray-500">{formatSize(size)}</span>
    </div>
  )
}

function FileTile({ file, thumb, onOpen }: { file: FileInfo; thumb?: string; onOpen: (file: FileInfo) => void }) {
  const [broken, setBroken] = useState(false)
  const showImage = fileKind(file.type, file.name) === 'image' && !broken

  return (
    <button
      onClick={() => onOpen(file)}
      title={file.name}
      aria-label={`Open ${file.name}`}
      className="relative aspect-square rounded-xl overflow-hidden bg-dark-800 border border-white/5 hover:border-accent-500 focus-visible:outline-2 focus-visible:outline-accent-500 transition active:scale-[0.97]"
    >
      {showImage ? (
        <img
          src={thumb ?? fileUrl(file.id)}
          alt={file.name}
          loading="lazy"
          decoding="async"
          draggable={false}
          onError={() => setBroken(true)}
          className="absolute inset-0 w-full h-full object-cover"
        />
      ) : (
        <GenericPreview name={file.name} type={file.type} size={file.size} />
      )}
    </button>
  )
}

function ProgressRing({ progress }: { progress: number }) {
  const radius = 18
  const circumference = 2 * Math.PI * radius
  return (
    <svg viewBox="0 0 44 44" className="w-12 h-12 -rotate-90" aria-hidden="true">
      <circle cx="22" cy="22" r={radius} fill="none" stroke="rgba(255,255,255,0.2)" strokeWidth="4" />
      <circle
        cx="22"
        cy="22"
        r={radius}
        fill="none"
        stroke="white"
        strokeWidth="4"
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={circumference * (1 - progress)}
        className="transition-[stroke-dashoffset] duration-200"
      />
    </svg>
  )
}

function UploadTile({
  upload,
  onRetry,
  onDismiss,
}: {
  upload: PendingUpload
  onRetry: (key: string) => void
  onDismiss: (key: string) => void
}) {
  const { file, previewUrl, progress, status } = upload
  const [broken, setBroken] = useState(false)

  return (
    <div className="relative aspect-square rounded-xl overflow-hidden bg-dark-800 border border-white/5" title={file.name}>
      {previewUrl && !broken ? (
        <img
          src={previewUrl}
          alt={file.name}
          onError={() => setBroken(true)}
          className="absolute inset-0 w-full h-full object-cover"
        />
      ) : (
        <GenericPreview name={file.name} type={file.type} size={file.size} />
      )}

      <div
        className={`absolute inset-0 flex flex-col items-center justify-center gap-1 text-white text-xs font-semibold ${
          status === 'error' ? 'bg-red-950/85' : status === 'done' ? 'bg-black/30' : 'bg-black/55'
        }`}
      >
        {status === 'queued' && <span>Waiting…</span>}
        {status === 'uploading' && (
          <>
            <ProgressRing progress={progress} />
            <span className="tabular-nums">{Math.round(progress * 100)}%</span>
          </>
        )}
        {status === 'done' && <CheckIcon className="w-8 h-8" />}
        {status === 'error' && (
          <>
            <span className="px-2 text-center">Upload failed</span>
            <button
              onClick={() => onRetry(upload.key)}
              className="flex items-center gap-1 mt-1 px-3 py-1.5 rounded-lg bg-white/15 hover:bg-white/25"
            >
              <RetryIcon className="w-4 h-4" />
              Retry
            </button>
          </>
        )}
      </div>

      {status !== 'done' && (
        <button
          onClick={() => onDismiss(upload.key)}
          aria-label={status === 'error' ? 'Dismiss' : 'Cancel upload'}
          className="absolute top-1 right-1 p-1.5 rounded-full bg-black/60 text-white hover:bg-black/80"
        >
          <CloseIcon className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  )
}

// Memoized so typing doesn't re-render every tile
export default memo(FileGrid)
