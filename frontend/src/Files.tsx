import { useState, useEffect } from 'react'
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

function Files() {
  const [files, setFiles] = useState<FileInfo[]>([])
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
    // Fetch files on mount
    fetchFiles()

    // Listen to individual file uploads completing
    uppy.on('upload-success', () => {
      // Small delay to allow backend to finish writing metadata
      setTimeout(() => {
        fetchFiles()
      }, 1000)
    })

  }, [uppy])

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
    <div className="w-full max-w-5xl">
      <div className="bg-dark-800 rounded-2xl p-6 shadow-xl">
        <h2 className="text-2xl font-bold text-accent-500 mb-6">Upload Files</h2>
        
        {/* Uppy Dashboard */}
        <div className="mb-8">
          <Dashboard
            uppy={uppy}
            proudlyDisplayPoweredByUppy={false}
            theme="dark"
            height={350}
          />
        </div>

        {/* Files List */}
        <div>
          <h3 className="text-xl font-semibold text-white mb-4">Your Files</h3>
          {files.length === 0 ? (
            <p className="text-gray-400 text-center py-8">No files uploaded yet</p>
          ) : (
            <div className="space-y-2">
              {files.map((file) => (
                <div
                  key={file.id}
                  className="bg-dark-900 border border-dark-700 rounded-lg p-4 flex items-center justify-between hover:border-accent-500 transition-colors"
                >
                  <div className="flex-1 min-w-0 mr-4">
                    <p className="text-white font-medium truncate">{file.name}</p>
                    <p className="text-gray-500 text-sm">
                      {formatFileSize(file.size)} • {formatDate(file.uploadedAt)}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleDownload(file.id, file.name)}
                      className="px-4 py-2 bg-accent-500 hover:bg-accent-600 text-white rounded-lg transition-colors text-sm font-semibold"
                    >
                      Download
                    </button>
                    <button
                      onClick={() => handleDelete(file.id)}
                      className="px-4 py-2 bg-red-700 hover:bg-red-800 text-white rounded-lg transition-colors text-sm font-semibold"
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
