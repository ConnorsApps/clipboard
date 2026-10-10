import { useCallback, useEffect, useRef, useState } from 'react'
import { getToken, type FileInfo } from './api'

export type ConnectionState = 'connecting' | 'connected' | 'offline'

// After this long in the background a socket may look open but be dead: phones suspend the
// page and the server drops it after missed pings, without the page ever seeing a close
const STALE_AFTER_HIDDEN_MS = 20_000

interface WSMessage {
  type: string
  content?: string
  files?: FileInfo[]
}

// useSync keeps one WebSocket open for both the shared text and the file list.
// files is null until the server has sent the first list.
export function useSync(onAuthError: () => void) {
  const [content, setContent] = useState('')
  const [files, setFiles] = useState<FileInfo[] | null>(null)
  const [connection, setConnection] = useState<ConnectionState>('connecting')
  const wsRef = useRef<WebSocket | null>(null)
  const contentRef = useRef('')
  // Set when the text was edited while no socket was open, so the edit isn't lost on reconnect
  const dirtyRef = useRef(false)
  const onAuthErrorRef = useRef(onAuthError)

  useEffect(() => {
    onAuthErrorRef.current = onAuthError
  }, [onAuthError])

  useEffect(() => {
    let disposed = false
    let attempt = 0
    let retryTimer: number | undefined

    const connect = () => {
      if (disposed) return
      window.clearTimeout(retryTimer)

      const token = getToken()
      if (!token) {
        onAuthErrorRef.current()
        return
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`)
      wsRef.current = ws
      setConnection('connecting')
      let receivedInitialContent = false

      ws.onopen = () => {
        attempt = 0
        setConnection('connected')
      }

      ws.onmessage = (event) => {
        let msg: WSMessage
        try {
          msg = JSON.parse(event.data)
        } catch {
          return
        }

        if (msg.type === 'content') {
          // The first content message after connecting is the server's copy. If the text was
          // edited while offline, keep the local edit and push it instead (last writer wins).
          if (!receivedInitialContent && dirtyRef.current) {
            receivedInitialContent = true
            dirtyRef.current = false
            ws.send(JSON.stringify({ type: 'update', content: contentRef.current }))
            return
          }
          receivedInitialContent = true
          contentRef.current = msg.content ?? ''
          setContent(contentRef.current)
        } else if (msg.type === 'files_list') {
          setFiles(msg.files ?? [])
        }
      }

      ws.onerror = () => {
        // onclose always fires after onerror and handles all reconnection
      }

      ws.onclose = (event) => {
        // Ignore sockets that have already been replaced by a newer connection
        if (disposed || wsRef.current !== ws) return
        wsRef.current = null
        setConnection('offline')

        // 4001/1008 = explicit auth rejection from the server — token is bad
        if (event.code === 4001 || event.code === 1008) {
          onAuthErrorRef.current()
          return
        }

        // Any other close (server restart, network blip, proxy drop) — keep retrying.
        // Backoff: 1s, 2s, 4s, … capped at 30s.
        const delay = Math.min(1000 * 2 ** attempt, 30000)
        attempt++
        retryTimer = window.setTimeout(connect, delay)
      }
    }

    // Phones drop sockets while the app is in the background; reconnect right away
    // when it comes back instead of waiting out the backoff
    const reconnectNow = () => {
      if (document.visibilityState !== 'visible') return
      const ws = wsRef.current
      if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
      attempt = 0
      connect()
    }

    let hiddenAt = 0
    const onVisibilityChange = () => {
      if (document.visibilityState === 'hidden') {
        hiddenAt = Date.now()
        return
      }
      const ws = wsRef.current
      if (ws && hiddenAt && Date.now() - hiddenAt > STALE_AFTER_HIDDEN_MS) {
        // Don't trust it; replacing wsRef first makes its onclose a no-op
        wsRef.current = null
        ws.close()
      }
      reconnectNow()
    }

    connect()
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('online', reconnectNow)

    return () => {
      disposed = true
      window.clearTimeout(retryTimer)
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('online', reconnectNow)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [])

  const updateContent = useCallback((next: string) => {
    contentRef.current = next
    setContent(next)
    const ws = wsRef.current
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'update', content: next }))
    } else {
      dirtyRef.current = true
    }
  }, [])

  return { content, files, connection, updateContent }
}
