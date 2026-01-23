import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

function Login() {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const response = await fetch('/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ password }),
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text || 'Login failed')
      }

      const data = await response.json()
      localStorage.setItem('clipboard_token', data.token)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-dark-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md bg-dark-800 rounded-2xl p-6 shadow-xl">
        <h1 className="text-2xl font-bold text-center text-accent-500 mb-6">
          Clipboard Sync
        </h1>
        <form onSubmit={handleSubmit}>
          {error && (
            <div className="text-red-400 text-center text-sm mb-4">{error}</div>
          )}
          <div className="mb-4">
            <input
              type="password"
              placeholder="Enter password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoFocus
              autoComplete="current-password"
              className="w-full px-4 py-4 text-base bg-dark-900 border-2 border-dark-700 rounded-xl text-white placeholder-gray-500 outline-none focus:border-accent-500 transition-colors"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-4 text-base font-semibold bg-accent-500 hover:bg-accent-600 text-white rounded-xl transition-colors active:scale-[0.98] disabled:opacity-50"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  )
}

export default Login
