import { useState, useEffect, useRef, ChangeEvent } from 'react'
import { useNavigate } from 'react-router-dom'

interface WSMessage {
  type: string
  content?: string
}

function Clipboard() {
  const [content, setContent] = useState('')
  const [connected, setConnected] = useState(false)
  const [copyFeedback, setCopyFeedback] = useState('')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const hasConnectedRef = useRef(false)
  const isUnmountingRef = useRef(false)
  const navigate = useNavigate()

  useEffect(() => {
    isUnmountingRef.current = false

    const connect = () => {
      // Don't connect if component is unmounting
      if (isUnmountingRef.current) return

      const token = localStorage.getItem('clipboard_token')
      if (!token) {
        navigate('/login')
        return
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`

      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        hasConnectedRef.current = true
        setConnected(true)
      }

      ws.onerror = () => {
        // If we never successfully connected, treat as auth failure
        // (WebSocket upgrade returned 401)
        if (!hasConnectedRef.current) {
          localStorage.removeItem('clipboard_token')
          navigate('/login')
        }
      }

      ws.onclose = (event) => {
        setConnected(false)

        // Don't reconnect if component is unmounting
        if (isUnmountingRef.current) return

        // Auth failure codes
        if (event.code === 1008 || event.code === 4001) {
          localStorage.removeItem('clipboard_token')
          navigate('/login')
          return
        }

        // Only reconnect if token still exists (not logged out)
        if (localStorage.getItem('clipboard_token')) {
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, 2000)
        }
      }

      ws.onmessage = (event) => {
        try {
          const msg: WSMessage = JSON.parse(event.data)
          if (msg.type === 'content') {
            setContent(msg.content ?? '')
          }
        } catch {
          // Ignore parse errors
        }
      }
    }

    connect()

    return () => {
      isUnmountingRef.current = true
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const sendUpdate = (newContent: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'update',
        content: newContent
      }))
    }
  }

  const handleContentChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value
    setContent(newContent)
    sendUpdate(newContent)
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content)
      setCopyFeedback('Copied!')
      setTimeout(() => setCopyFeedback(''), 2000)
    } catch {
      setCopyFeedback('Failed to copy')
      setTimeout(() => setCopyFeedback(''), 2000)
    }
  }

  const handleClear = () => {
    setContent('')
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'clear' }))
    }
  }

  return (
    <div className="w-full max-w-xl bg-dark-800 rounded-2xl p-5 shadow-xl">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-xl font-bold text-accent-500">Clipboard</h1>
        <div className="flex items-center gap-2 text-xs text-gray-400">
          <span
            className={`w-2 h-2 rounded-full ${
              connected ? 'bg-green-400' : 'bg-red-400'
            }`}
          />
          <span>{connected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </div>

      {copyFeedback && (
        <div className="text-green-400 text-center text-sm mb-4">
          {copyFeedback}
        </div>
      )}

      <textarea
        value={content}
        onChange={handleContentChange}
        placeholder="Paste or type content here..."
        autoFocus
        className="w-full min-h-[300px] max-h-[60vh] p-4 text-base bg-dark-900 border-2 border-dark-700 rounded-xl text-white placeholder-gray-500 outline-none focus:border-accent-500 transition-colors resize-y touch-pan-y"
        style={{ fontSize: '16px' }}
      />

      <div className="flex gap-3 mt-4">
        <button
          onClick={handleCopy}
          className="flex-1 py-4 text-base font-semibold bg-accent-500 hover:bg-accent-600 text-white rounded-xl transition-colors active:scale-[0.98]"
        >
          Copy
        </button>
        <button
          onClick={handleClear}
          className="flex-1 py-4 text-base font-semibold bg-red-700 hover:bg-red-800 text-white rounded-xl transition-colors active:scale-[0.98]"
        >
          Clear
        </button>
      </div>
    </div>
  )
}

export default Clipboard
