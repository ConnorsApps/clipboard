import { useCallback, useEffect, useEffectEvent, useRef, useState, type ChangeEvent } from 'react'
import type { FileInfo } from './api'
import AddBar, { type PickerKind } from './AddBar'
import { readClipboard, textWinsOverImage } from './clipboard'
import DropOverlay from './DropOverlay'
import FileGrid from './FileGrid'
import FileViewer from './FileViewer'
import { renameGenericFile } from './format'
import { ClipboardIcon, LogoutIcon } from './icons'
import TextPanel from './TextPanel'
import { Toast, useToast } from './Toast'
import { useSync, type ConnectionState } from './useSync'
import { useUploads } from './useUploads'

const isTouch = window.matchMedia('(pointer: coarse)').matches
const pasteShortcut = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘V' : 'Ctrl+V'

function isEditable(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')
  )
}

function ConnectionBadge({ connection }: { connection: ConnectionState }) {
  const styles: Record<ConnectionState, { dot: string; label: string }> = {
    connected: { dot: 'bg-green-400', label: 'Live' },
    connecting: { dot: 'bg-amber-400 animate-pulse', label: 'Connecting' },
    offline: { dot: 'bg-red-400', label: 'Offline' },
  }
  const { dot, label } = styles[connection]
  return (
    <span className="flex items-center gap-1.5 rounded-full bg-white/5 px-2.5 py-1 text-xs text-gray-300">
      <span className={`w-2 h-2 rounded-full ${dot}`} />
      {label}
    </span>
  )
}

function Home({ onLogout }: { onLogout: () => void }) {
  const { toast, showToast, dismissToast } = useToast()
  const handleConflict = useCallback(
    (restore: () => void) =>
      showToast('Your offline edits replaced newer text from another device', {
        action: { label: 'Undo', onClick: restore },
      }),
    [showToast],
  )
  const { content, files, connection, synced, updateContent, getContent } = useSync({
    onAuthError: onLogout,
    onConflict: handleConflict,
  })
  const { uploads, thumbs, add, retry, dismiss } = useUploads(files)
  const [viewingId, setViewingId] = useState<string | null>(null)
  const [textFocused, setTextFocused] = useState(false)
  const [dragging, setDragging] = useState(false)
  const historyPushed = useRef(false)
  const pickers = {
    photos: useRef<HTMLInputElement>(null),
    camera: useRef<HTMLInputElement>(null),
    files: useRef<HTMLInputElement>(null),
  }

  const viewing = viewingId ? (files?.find((f) => f.id === viewingId) ?? null) : null

  const addFiles = (list: File[], prefix: string) => {
    if (list.length > 0) add(list.map((file) => renameGenericFile(file, prefix)))
  }

  const replaceText = (text: string, message: string) => {
    // Before the first sync, this would overwrite whatever the server has
    if (!synced) {
      showToast('Still connecting — try again in a moment', { tone: 'error' })
      return
    }
    // Read now, not from this render: the Paste button awaits the clipboard first
    const previous = getContent()
    updateContent(text)
    showToast(message, previous ? { action: { label: 'Undo', onClick: () => updateContent(previous) } } : {})
  }

  const handlePasteButton = async () => {
    try {
      const { text, files: images } = await readClipboard()
      if (images.length > 0) {
        addFiles(images, 'paste')
      } else if (text) {
        replaceText(text, 'Pasted from clipboard')
      } else {
        showToast('Your clipboard is empty')
      }
    } catch {
      showToast(
        isTouch
          ? "Couldn't read the clipboard. Long-press the text box and tap Paste instead."
          : `Couldn't read the clipboard. Press ${pasteShortcut} instead.`,
        { tone: 'error' },
      )
    }
  }

  const handlePickerChange = (kind: PickerKind) => (e: ChangeEvent<HTMLInputElement>) => {
    addFiles(Array.from(e.target.files ?? []), kind === 'files' ? 'file' : 'photo')
    // Let the same file be picked again
    e.target.value = ''
  }

  // Ctrl/Cmd+V anywhere: images are uploaded; text outside the text box replaces the shared text.
  // When both are present, textWinsOverImage decides.
  const onDocumentPaste = useEffectEvent((e: ClipboardEvent) => {
    const data = e.clipboardData
    if (!data) return
    const text = data.types.includes('text/plain') ? data.getData('text/plain') : ''
    const pasted = Array.from(data.files)
    if (pasted.length > 0 && !(text && textWinsOverImage(text))) {
      e.preventDefault()
      addFiles(pasted, 'paste')
      return
    }
    if (text && !isEditable(e.target)) {
      e.preventDefault()
      replaceText(text, 'Pasted text')
    }
  })

  useEffect(() => {
    const listener = (e: ClipboardEvent) => onDocumentPaste(e)
    document.addEventListener('paste', listener)
    return () => document.removeEventListener('paste', listener)
  }, [])

  // Drag files anywhere onto the page
  const onDropFiles = useEffectEvent((list: File[]) => addFiles(list, 'paste'))

  useEffect(() => {
    let depth = 0
    // Dragging an image that's already on the page (e.g. out of the viewer to save it) also
    // reports 'Files'; ignore drags that start here so they don't upload a duplicate
    let internalDrag = false
    const onStart = () => {
      internalDrag = true
    }
    const onEnd = () => {
      internalDrag = false
    }
    const hasFiles = (e: DragEvent) => !internalDrag && (e.dataTransfer?.types.includes('Files') ?? false)
    const onEnter = (e: DragEvent) => {
      if (!hasFiles(e)) return
      e.preventDefault()
      depth++
      setDragging(true)
    }
    const onOver = (e: DragEvent) => {
      if (!hasFiles(e)) return
      e.preventDefault()
      e.dataTransfer!.dropEffect = 'copy'
    }
    const onLeave = (e: DragEvent) => {
      if (!hasFiles(e)) return
      depth = Math.max(0, depth - 1)
      if (depth === 0) setDragging(false)
    }
    const onDrop = (e: DragEvent) => {
      if (!hasFiles(e)) return
      e.preventDefault()
      depth = 0
      setDragging(false)
      onDropFiles(Array.from(e.dataTransfer!.files))
    }
    window.addEventListener('dragstart', onStart)
    window.addEventListener('dragend', onEnd)
    window.addEventListener('dragenter', onEnter)
    window.addEventListener('dragover', onOver)
    window.addEventListener('dragleave', onLeave)
    window.addEventListener('drop', onDrop)
    return () => {
      window.removeEventListener('dragstart', onStart)
      window.removeEventListener('dragend', onEnd)
      window.removeEventListener('dragenter', onEnter)
      window.removeEventListener('dragover', onOver)
      window.removeEventListener('dragleave', onLeave)
      window.removeEventListener('drop', onDrop)
    }
  }, [])

  // The viewer gets a history entry so the phone's Back gesture closes it instead of leaving
  const openViewer = useCallback((file: FileInfo) => {
    window.history.pushState({ viewer: true }, '')
    historyPushed.current = true
    setViewingId(file.id)
  }, [])

  const closeViewer = useCallback(() => {
    setViewingId(null)
    if (historyPushed.current) {
      historyPushed.current = false
      window.history.back()
    }
  }, [])

  useEffect(() => {
    const onPop = () => {
      historyPushed.current = false
      setViewingId(null)
    }
    window.addEventListener('popstate', onPop)
    return () => {
      window.removeEventListener('popstate', onPop)
      // Logged out with the viewer open: don't leave its entry behind
      if (historyPushed.current) window.history.back()
    }
  }, [])

  // The open file was deleted on another device
  useEffect(() => {
    if (viewingId && files && !files.some((f) => f.id === viewingId)) closeViewer()
  }, [files, viewingId, closeViewer])

  return (
    <div className="min-h-dvh">
      <header className="sticky top-0 z-20 bg-dark-900/85 backdrop-blur-md border-b border-white/5 pt-[env(safe-area-inset-top)]">
        <div className="max-w-6xl mx-auto px-4 h-14 flex items-center gap-3">
          <ClipboardIcon className="w-6 h-6 text-accent-500" />
          <h1 className="text-lg font-bold">Clipboard</h1>
          <ConnectionBadge connection={connection} />
          <button
            onClick={onLogout}
            className="ml-auto flex items-center gap-2 rounded-lg px-2.5 py-2 text-sm text-gray-400 hover:text-white hover:bg-white/5"
            aria-label="Log out"
          >
            <LogoutIcon className="w-4.5 h-4.5" />
            <span className="hidden sm:inline">Log out</span>
          </button>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 pt-4 pb-[calc(7rem+env(safe-area-inset-bottom))] md:pb-10 grid gap-6 lg:grid-cols-2 lg:items-start">
        <TextPanel
          value={content}
          readOnly={!synced}
          onChange={updateContent}
          onClear={() => replaceText('', 'Text cleared')}
          onFocusChange={setTextFocused}
          onError={(message) => showToast(message, { tone: 'error' })}
        />
        <div className="space-y-4 min-w-0">
          <AddBar variant="inline" onPaste={handlePasteButton} onPick={(kind) => pickers[kind].current?.click()} />
          <FileGrid
            files={files}
            uploads={uploads}
            thumbs={thumbs}
            onOpen={openViewer}
            onRetry={retry}
            onDismiss={dismiss}
          />
        </div>
      </main>

      <AddBar
        variant="dock"
        hidden={textFocused}
        onPaste={handlePasteButton}
        onPick={(kind) => pickers[kind].current?.click()}
      />

      <input ref={pickers.photos} type="file" accept="image/*,video/*" multiple hidden onChange={handlePickerChange('photos')} />
      <input ref={pickers.camera} type="file" accept="image/*" capture="environment" hidden onChange={handlePickerChange('camera')} />
      <input ref={pickers.files} type="file" multiple hidden onChange={handlePickerChange('files')} />

      {viewing && (
        <FileViewer
          key={viewing.id}
          file={viewing}
          thumb={thumbs[viewing.id]}
          onClose={closeViewer}
          onDeleted={() => {
            closeViewer()
            showToast('Deleted')
          }}
          showToast={showToast}
        />
      )}

      {dragging && <DropOverlay />}
      <Toast
        toast={toast}
        onDismiss={dismissToast}
        placement={viewing ? 'top' : textFocused ? 'bottom' : 'above-dock'}
      />
    </div>
  )
}

export default Home
