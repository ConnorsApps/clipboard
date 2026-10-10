import type { ReactNode } from 'react'
import { AttachIcon, CameraIcon, PasteIcon, PhotoIcon } from './icons'

export type PickerKind = 'photos' | 'camera' | 'files'

interface AddBarProps {
  onPaste: () => void
  onPick: (kind: PickerKind) => void
  // dock: fixed thumb-reach bar on phones; inline: a row above the grid on larger screens
  variant: 'dock' | 'inline'
  hidden?: boolean
}

// Camera capture only makes sense on phones and tablets
const hasCamera = window.matchMedia('(pointer: coarse)').matches
const isMac = /Mac|iPhone|iPad/.test(navigator.platform)

function AddBar({ onPaste, onPick, variant, hidden }: AddBarProps) {
  const buttons: { label: string; icon: ReactNode; onClick: () => void; primary?: boolean }[] = [
    { label: 'Paste', icon: <PasteIcon className="w-6 h-6" />, onClick: onPaste, primary: true },
    { label: 'Photos', icon: <PhotoIcon className="w-6 h-6" />, onClick: () => onPick('photos') },
    ...(hasCamera ? [{ label: 'Camera', icon: <CameraIcon className="w-6 h-6" />, onClick: () => onPick('camera') }] : []),
    { label: 'Files', icon: <AttachIcon className="w-6 h-6" />, onClick: () => onPick('files') },
  ]

  if (variant === 'dock') {
    return (
      <nav
        aria-label="Add to clipboard"
        className={`md:hidden fixed inset-x-0 bottom-0 z-30 bg-dark-800/95 backdrop-blur-md border-t border-white/10 pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)] pr-[env(safe-area-inset-right)] ${
          hidden ? 'hidden' : ''
        }`}
      >
        <div className="flex gap-1 px-2 pt-2 pb-2">
          {buttons.map((b) => (
            <button
              key={b.label}
              onClick={b.onClick}
              className={`flex-1 flex flex-col items-center justify-center gap-1 py-2 rounded-xl text-xs font-semibold transition active:scale-95 ${
                b.primary ? 'bg-accent-500 text-white' : 'text-gray-300 active:bg-dark-700'
              }`}
            >
              {b.icon}
              {b.label}
            </button>
          ))}
        </div>
      </nav>
    )
  }

  return (
    <div className="hidden md:flex flex-wrap items-center gap-2">
      {buttons.map((b) => (
        <button
          key={b.label}
          onClick={b.onClick}
          className={`flex items-center gap-2 px-4 py-2.5 rounded-xl font-semibold text-sm transition active:scale-[0.98] ${
            b.primary ? 'bg-accent-500 hover:bg-accent-600 text-white' : 'bg-dark-700 hover:bg-dark-700/70 text-gray-200'
          }`}
        >
          {b.icon}
          {b.label}
        </button>
      ))}
      <span className="text-sm text-gray-500 ml-1">
        or drop files anywhere · <kbd className="font-sans">{isMac ? '⌘' : 'Ctrl+'}V</kbd>
      </span>
    </div>
  )
}

export default AddBar
