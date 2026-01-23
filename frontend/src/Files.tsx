import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import Uppy from '@uppy/core'
import Dashboard from '@uppy/react/dashboard'
import Tus from '@uppy/tus'

// Import Uppy styles
import '@uppy/core/css/style.min.css'
import '@uppy/dashboard/css/style.min.css'

interface FileInfo {
  id: string
  name: string
  size: number
  uploadedAt: string
}

interface WSMessage {
  type: string
  files?: FileInfo[]
}

function Files() {
  const [files, setFiles] = useState<FileInfo[]>([])
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const hasConnectedRef = useRef(false)
  const isUnmountingRef = useRef(false)
  const [uppy] = useState(() => {
    const token = localStorage.getItem('clipboard_token')
    
    const uppyInstance = new Uppy({
      restrictions: {
        maxFileSize: null, // No limit
      },
      autoProceed: true, // Auto-upload when files are selected
    }).use(Tus, {
      allowedMetaFields: true,
      limit: 0,
      endpoint: '/api/uploads/',
      chunkSize: 5 * 1024 * 1024, // 5MB chunks
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
    
    uppyInstance.on('error',(err)=>{
      console.error(err);
    })

    uppyInstance.on('file-removed',(file)=>{
      setFiles(files.filter(f => f.id !== file.id))
    })

    // Ensure filename is passed in metadata
    uppyInstance.on('file-added', (file) => {
      uppyInstance.setFileMeta(file.id, {
        filename: file.name,
      })
    })

    return uppyInstance
  })
  const navigate = useNavigate()

  useEffect(() => {
    isUnmountingRef.current = false

    // WebSocket connection
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

        // Only reconnect if token still exists
        if (localStorage.getItem('clipboard_token')) {
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, 2000)
        }
      }

      ws.onmessage = (event) => {
        try {
          const msg: WSMessage = JSON.parse(event.data)
          if (msg.type === 'files_list' && msg.files) {
            setFiles(msg.files)
          }
        } catch {
          // Ignore parse errors
        }
      }
    }

    // Fetch files on mount
    fetchFiles()
    
    // Connect to WebSocket
    connect()

    // Listen to individual file uploads completing (fallback)
    uppy.on('upload-success', () => {
      // Small delay to allow backend to finish writing metadata
      setTimeout(() => {
        fetchFiles()
      }, 1000)
    })

    return () => {
      isUnmountingRef.current = true
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [uppy, navigate])

  const fetchFiles = async () => {
    try {
      const token = localStorage.getItem('clipboard_token')
      const response = await fetch('/api/files', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (response.status === 401) {
        localStorage.removeItem('clipboard_token')
        navigate('/login')
        return
      }

      if (response.ok) {
        const data = await response.json()
        setFiles(data || [])
      }
    } catch (error) {
      console.error('Failed to fetch files:', error)
    }
  }

  const handleDelete = async (fileId: string) => {
    try {
      const token = localStorage.getItem('clipboard_token')
      const response = await fetch(`/api/files/${fileId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (response.ok) {
        await fetchFiles()
      }
    } catch (error) {
      console.error('Failed to delete file:', error)
    }
  }

  const handleDownload = (fileId: string, fileName: string) => {
    const token = localStorage.getItem('clipboard_token')
    const url = `/api/files/${fileId}?token=${encodeURIComponent(token || '')}`
    
    // Create a temporary link and click it
    const a = document.createElement('a')
    a.href = url
    a.download = fileName
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString)
    return date.toLocaleString()
  }

  return (
    <div className="w-full max-w-5xl px-2 sm:px-0">
      <div className="bg-dark-800 rounded-2xl py-6 px-4 sm:px-6 shadow-xl">
        <div className="flex justify-between items-center mb-4 sm:mb-6">
          <h2 className="text-xl sm:text-2xl font-bold text-accent-500">Upload Files</h2>
          <div className="flex items-center gap-2 text-xs text-gray-400">
            <span
              className={`w-2 h-2 rounded-full ${
                connected ? 'bg-green-400' : 'bg-red-400'
              }`}
            />
            <span>{connected ? 'Connected' : 'Disconnected'}</span>
          </div>
        </div>
        
        {/* Uppy Dashboard */}
        <div className="mb-6 sm:mb-8">
          <Dashboard
            uppy={uppy}
            proudlyDisplayPoweredByUppy={false}
            theme="dark"
            width="100%"
            height={350}
          />
        </div>

        {/* Files List */}
        <div className="max-h-[50vh] overflow-y-auto" style={{ WebkitOverflowScrolling: 'touch' }}>
          <h3 className="text-lg sm:text-xl font-semibold text-white mb-3 sm:mb-4">Your Files</h3>
          {files.length === 0 ? (
            <p className="text-gray-400 text-center py-8">No files uploaded yet</p>
          ) : (
            <div className="space-y-2">
              {files.map((file) => (
                <div
                  key={file.id}
                  className="bg-dark-900 border border-dark-700 rounded-lg p-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 hover:border-accent-500 transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-white font-medium truncate">{file.name}</p>
                    <p className="text-gray-500 text-sm">
                      {formatFileSize(file.size)} • {formatDate(file.uploadedAt)}
                    </p>
                  </div>
                  <div className="flex gap-2 flex-shrink-0">
                    <button
                      onClick={() => handleDownload(file.id, file.name)}
                      className="px-3 py-2 sm:px-4 bg-accent-500 hover:bg-accent-600 text-white rounded-lg transition-colors text-sm font-semibold flex-1 sm:flex-none active:scale-[0.98]"
                    >
                      Download
                    </button>
                    <button
                      onClick={() => handleDelete(file.id)}
                      className="px-3 py-2 sm:px-4 bg-red-700 hover:bg-red-800 text-white rounded-lg transition-colors text-sm font-semibold flex-1 sm:flex-none active:scale-[0.98]"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Files
