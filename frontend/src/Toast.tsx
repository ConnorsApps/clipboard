import { useCallback, useEffect, useRef, useState } from 'react'

export interface ToastOptions {
  tone?: 'info' | 'error'
  action?: { label: string; onClick: () => void }
}

interface ToastState extends ToastOptions {
  id: number
  message: string
}

export function useToast() {
  const [toast, setToast] = useState<ToastState | null>(null)
  const nextId = useRef(0)

  const showToast = useCallback((message: string, options: ToastOptions = {}) => {
    setToast({ id: nextId.current++, message, ...options })
  }, [])

  const dismissToast = useCallback(() => setToast(null), [])

  return { toast, showToast, dismissToast }
}

interface ToastProps {
  toast: ToastState | null
  onDismiss: () => void
  // above-dock: clears the phone's add bar; top: clears the viewer's buttons
  placement: 'bottom' | 'above-dock' | 'top'
}

const placementClass = {
  bottom: 'bottom-[calc(1rem+env(safe-area-inset-bottom))] md:bottom-6',
  'above-dock': 'bottom-[calc(5.75rem+env(safe-area-inset-bottom))] md:bottom-6',
  top: 'top-[calc(4.5rem+env(safe-area-inset-top))]',
}

export function Toast({ toast, onDismiss, placement }: ToastProps) {
  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(onDismiss, toast.action ? 5000 : toast.tone === 'error' ? 4000 : 2200)
    return () => window.clearTimeout(timer)
  }, [toast, onDismiss])

  if (!toast) return null

  return (
    <div
      className={`fixed inset-x-0 z-[60] flex justify-center px-4 pointer-events-none ${placementClass[placement]}`}
    >
      <div
        key={toast.id}
        role="status"
        className={`pointer-events-auto flex items-center gap-3 max-w-md rounded-xl px-4 py-3 text-sm shadow-2xl border ${
          toast.tone === 'error'
            ? 'bg-red-950 border-red-800 text-red-100'
            : 'bg-dark-700 border-white/10 text-white'
        }`}
      >
        <span className="min-w-0">{toast.message}</span>
        {toast.action && (
          <button
            onClick={() => {
              toast.action!.onClick()
              onDismiss()
            }}
            className="shrink-0 font-semibold text-accent-500 hover:text-white px-1"
          >
            {toast.action.label}
          </button>
        )}
      </div>
    </div>
  )
}
