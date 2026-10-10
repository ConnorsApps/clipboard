export type FileKind = 'image' | 'video' | 'audio' | 'text' | 'other'

const IMAGE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'avif', 'bmp', 'svg', 'heic', 'heif']
const VIDEO_EXTENSIONS = ['mp4', 'mov', 'm4v', 'webm']
const AUDIO_EXTENSIONS = ['mp3', 'm4a', 'wav', 'ogg', 'aac', 'flac']

export function extension(name: string): string {
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : ''
}

export function fileKind(type: string, name: string): FileKind {
  const ext = extension(name)
  if (type.startsWith('image/') || IMAGE_EXTENSIONS.includes(ext)) return 'image'
  if (type.startsWith('video/') || VIDEO_EXTENSIONS.includes(ext)) return 'video'
  if (type.startsWith('audio/') || AUDIO_EXTENSIONS.includes(ext)) return 'audio'
  if (type.startsWith('text/')) return 'text'
  return 'other'
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes / 1024
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`
}

export function formatRelative(dateString: string): string {
  const date = new Date(dateString)
  const seconds = (Date.now() - date.getTime()) / 1000
  if (seconds < 60) return 'just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} h ago`
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

// Replaces generic names like "image.png" from pastes and iOS camera shots
export function generatedFileName(type: string, prefix = 'paste', date = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const stamp =
    `${date.getFullYear()}${pad(date.getMonth() + 1)}${pad(date.getDate())}` +
    `-${pad(date.getHours())}${pad(date.getMinutes())}${pad(date.getSeconds())}`
  const sub = type.split('/')[1]?.split(/[+;]/)[0] ?? 'bin'
  return `${prefix}-${stamp}.${sub === 'jpeg' ? 'jpg' : sub}`
}

export function renameGenericFile(file: File, prefix: string): File {
  if (file.name && !/^image\.\w+$/i.test(file.name)) return file
  return new File([file], generatedFileName(file.type || 'image/png', prefix), {
    type: file.type,
    lastModified: file.lastModified,
  })
}
