import { useCallback, useEffect, useRef, useState } from 'react'
import { getToken, type FileInfo } from './api'

export type ConnectionState = 'connecting' | 'connected' | 'offline'

// After this long in the background a socket may look open but be dead: phones suspend the
// page and the server drops it after missed pings, without the page ever seeing a close
const STALE_AFTER_HIDDEN_MS = 20_000
// While visible, check the connection is alive; a socket can also die silently in the
// foreground, e.g. when a phone switches from Wi-Fi to cellular
const HEARTBEAT_MS = 25_000
const PONG_TIMEOUT_MS = 10_000
const MAX_REMEMBERED_SENDS = 5000

interface WSMessage {
  type: string
  content?: string
  version?: number
  files?: FileInfo[]
}

// FNV-1a; only used to recognize texts this device sent
function hashText(text: string): number {
  let hash = 0x811c9dc5
  for (let i = 0; i < text.length; i++) {
    hash ^= text.charCodeAt(i)
    hash = Math.imul(hash, 0x01000193)
  }
  return hash >>> 0
}

interface SyncOptions {
  onAuthError: () => void
  // Called when edits made offline replaced newer text from another device; restore puts
  // that text back
  onConflict: (restore: () => void) => void
}

// useSync keeps one WebSocket open for both the shared text and the file list.
// files is null until the server has sent the first list; synced is false until the
// shared text has been received once.
export function useSync({ onAuthError, onConflict }: SyncOptions) {
  const [content, setContent] = useState('')
  const [files, setFiles] = useState<FileInfo[] | null>(null)
  const [connection, setConnection] = useState<ConnectionState>('connecting')
  const [synced, setSynced] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  // The current socket has delivered the server's copy, so edits can be sent on it
  const socketSyncedRef = useRef(false)
  const contentRef = useRef('')
  // Last text received from the server (null until the first sync)
  const serverTextRef = useRef<string | null>(null)
  // Hashes of texts sent since then. After a reconnect, a server copy that is one of these (or
  // the last received text) means nobody else changed it, so any difference is our edits that
  // never arrived and they should be sent again.
  const sentHashesRef = useRef(new Set<number>())
  const lastSentHashRef = useRef<number | null>(null)
  // Edited while the socket couldn't take updates
  const dirtyRef = useRef(false)
  const callbacksRef = useRef({ onAuthError, onConflict })

  useEffect(() => {
    callbacksRef.current = { onAuthError, onConflict }
  }, [onAuthError, onConflict])

  const sendUpdate = useCallback((ws: WebSocket, text: string) => {
    ws.send(JSON.stringify({ type: 'update', content: text }))
    const sent = sentHashesRef.current
    if (sent.size >= MAX_REMEMBERED_SENDS) sent.delete(sent.values().next().value!)
    const hash = hashText(text)
    sent.add(hash)
    lastSentHashRef.current = hash
  }, [])

  const updateContent = useCallback(
    (next: string) => {
      contentRef.current = next
      setContent(next)
      const ws = wsRef.current
      // Until this socket has delivered the server's copy, an update could be overwritten by
      // that copy arriving after it; hold it and push it once synced instead
      if (ws && ws.readyState === WebSocket.OPEN && socketSyncedRef.current) {
        sendUpdate(ws, next)
      } else {
        dirtyRef.current = true
      }
    },
    [sendUpdate],
  )

  useEffect(() => {
    let disposed = false
    let attempt = 0
    let retryTimer: number | undefined
    let stopSocket: (() => void) | undefined

    const connect = () => {
      if (disposed) return
      window.clearTimeout(retryTimer)

      const token = getToken()
      if (!token) {
        callbacksRef.current.onAuthError()
        return
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`)
      wsRef.current = ws
      socketSyncedRef.current = false
      setConnection('connecting')
      let receivedInitialContent = false
      // Version of the newest content seen on this socket
      let version = 0
      let heartbeat: number | undefined
      let pongTimer: number | undefined

      const stop = () => {
        window.clearInterval(heartbeat)
        window.clearTimeout(pongTimer)
      }
      stopSocket = stop

      ws.onopen = () => {
        heartbeat = window.setInterval(() => {
          if (document.visibilityState !== 'visible' || pongTimer !== undefined) return
          ws.send(JSON.stringify({ type: 'ping' }))
          pongTimer = window.setTimeout(() => {
            pongTimer = undefined
            replaceSocket(ws)
          }, PONG_TIMEOUT_MS)
        }, HEARTBEAT_MS)
      }

      ws.onmessage = (event) => {
        // Any message proves the connection is alive
        window.clearTimeout(pongTimer)
        pongTimer = undefined

        let msg: WSMessage
        try {
          msg = JSON.parse(event.data)
        } catch {
          return
        }

        if (msg.type === 'content') {
          const incoming = msg.content ?? ''
          const show = (text: string) => {
            contentRef.current = text
            setContent(text)
          }

          if (receivedInitialContent) {
            // Versions only grow, so an older one arrived out of order and is already superseded
            if (msg.version !== undefined) {
              if (msg.version <= version) return
              version = msg.version
            }
            serverTextRef.current = incoming
            const hash = hashText(incoming)
            if (sentHashesRef.current.has(hash)) {
              // The server echoing our own edit. Older echoes are skipped so typing isn't rolled
              // back; the latest is applied in case another device's text was shown before it
              if (hash === lastSentHashRef.current && contentRef.current !== incoming) show(incoming)
              return
            }
            sentHashesRef.current.clear()
            show(incoming)
            return
          }

          // The server's copy, sent right after connecting
          receivedInitialContent = true
          version = msg.version ?? 0
          // Only now is the server really serving us; it can accept and then close
          // (e.g. token store down), and the backoff must keep growing in that case
          attempt = 0
          socketSyncedRef.current = true
          setConnection('connected')

          const local = contentRef.current
          const known = serverTextRef.current
          const serverHasOurs =
            known !== null && (incoming === known || sentHashesRef.current.has(hashText(incoming)))
          serverTextRef.current = incoming
          sentHashesRef.current.clear()
          if (known !== null && local !== incoming && (dirtyRef.current || serverHasOurs)) {
            // Our edits never made it; push them (last writer wins)
            dirtyRef.current = false
            sendUpdate(ws, local)
            if (!serverHasOurs) callbacksRef.current.onConflict(() => updateContent(incoming))
            return
          }
          dirtyRef.current = false
          show(incoming)
          setSynced(true)
        } else if (msg.type === 'files_list') {
          setFiles(msg.files ?? [])
        }
      }

      ws.onerror = () => {
        // onclose always fires after onerror and handles all reconnection
      }

      ws.onclose = (event) => {
        stop()
        // Ignore sockets that have already been replaced by a newer connection
        if (disposed || wsRef.current !== ws) return
        wsRef.current = null
        socketSyncedRef.current = false
        setConnection('offline')

        // 4001/1008 = explicit auth rejection from the server — token is bad
        if (event.code === 4001 || event.code === 1008) {
          callbacksRef.current.onAuthError()
          return
        }

        // Any other close (server restart, network blip, proxy drop) — keep retrying.
        // Backoff: 1s, 2s, 4s, … capped at 30s.
        const delay = Math.min(1000 * 2 ** attempt, 30000)
        attempt++
        retryTimer = window.setTimeout(connect, delay)
      }
    }

    // Drop a socket we no longer trust and start a fresh one. Clearing wsRef first turns
    // its onclose into a no-op.
    const replaceSocket = (ws: WebSocket) => {
      if (disposed || wsRef.current !== ws) return
      stopSocket?.()
      wsRef.current = null
      socketSyncedRef.current = false
      ws.close()
      attempt = 0
      connect()
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
        replaceSocket(ws)
      } else {
        reconnectNow()
      }
    }

    connect()
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('online', reconnectNow)

    return () => {
      disposed = true
      window.clearTimeout(retryTimer)
      stopSocket?.()
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('online', reconnectNow)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [sendUpdate, updateContent])

  const getContent = useCallback(() => contentRef.current, [])

  return { content, files, connection, synced, updateContent, getContent }
}
