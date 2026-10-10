import { useLayoutEffect, useRef, useState, type ChangeEvent } from 'react'
import { copyText } from './clipboard'
import { CheckIcon, CopyIcon, EraseIcon } from './icons'

interface TextPanelProps {
  value: string
  onChange: (value: string) => void
  onClear: () => void
  onFocusChange: (focused: boolean) => void
  onError: (message: string) => void
}

// Don't pop the on-screen keyboard over everything when the page opens on a phone
const autoFocus = window.matchMedia('(pointer: fine)').matches

function TextPanel({ value, onChange, onClear, onFocusChange, onError }: TextPanelProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [copied, setCopied] = useState(false)

  // Grow with the content; CSS caps the height and scrolls beyond that
  useLayoutEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return
    textarea.style.height = 'auto'
    textarea.style.height = `${textarea.scrollHeight + 4}px`
  }, [value])

  const handleCopy = async () => {
    try {
      await copyText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      onError('Copy failed — select the text and copy it manually')
    }
  }

  return (
    <section className="bg-dark-800 rounded-2xl p-3 sm:p-4 shadow-xl border border-white/5">
      <div className="flex items-baseline justify-between px-1 mb-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-400">Text</h2>
        {value && (
          <span className="text-xs text-gray-500 tabular-nums">{value.length.toLocaleString()} chars</span>
        )}
      </div>

      <textarea
        ref={textareaRef}
        value={value}
        onChange={(e: ChangeEvent<HTMLTextAreaElement>) => onChange(e.target.value)}
        onFocus={() => onFocusChange(true)}
        onBlur={() => onFocusChange(false)}
        placeholder="Type or paste text to share across your devices…"
        autoFocus={autoFocus}
        spellCheck={false}
        className="block w-full min-h-32 lg:min-h-64 max-h-[45dvh] lg:max-h-[65dvh] p-3 sm:p-4 bg-dark-900 border-2 border-dark-700 rounded-xl text-white placeholder-gray-500 outline-none focus:border-accent-500 transition-colors resize-none overflow-y-auto leading-relaxed"
      />

      <div className="flex gap-2 mt-3">
        <button
          onClick={handleCopy}
          disabled={!value}
          className="flex-1 flex items-center justify-center gap-2 py-3.5 font-semibold bg-accent-500 hover:bg-accent-600 text-white rounded-xl transition active:scale-[0.98] disabled:opacity-40 disabled:active:scale-100"
        >
          {copied ? <CheckIcon /> : <CopyIcon />}
          {copied ? 'Copied' : 'Copy text'}
        </button>
        <button
          onClick={onClear}
          disabled={!value}
          className="flex items-center justify-center gap-2 px-5 py-3.5 font-semibold bg-dark-700 hover:bg-dark-700/70 text-gray-200 rounded-xl transition active:scale-[0.98] disabled:opacity-40 disabled:active:scale-100"
        >
          <EraseIcon />
          Clear
        </button>
      </div>
    </section>
  )
}

export default TextPanel
