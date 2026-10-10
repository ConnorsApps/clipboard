import { useState, type FormEvent } from 'react'
import { login } from './api'
import { ClipboardIcon } from './icons'

function Login({ onLogin }: { onLogin: (token: string) => void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      onLogin(await login(password))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
      setLoading(false)
    }
  }

  return (
    <div className="min-h-dvh flex flex-col items-center justify-center px-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]">
      <div className="w-full max-w-sm bg-dark-800 rounded-2xl p-6 shadow-xl border border-white/5">
        <ClipboardIcon className="w-10 h-10 mx-auto text-accent-500" />
        <h1 className="mt-3 text-2xl font-bold text-center">Clipboard Sync</h1>
        <p className="mt-1 mb-6 text-sm text-center text-gray-400">Share text, images and files between your devices</p>
        <form onSubmit={handleSubmit}>
          {error && <div className="text-red-400 text-center text-sm mb-4">{error}</div>}
          <input
            type="password"
            placeholder="Password"
            aria-label="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoFocus
            autoComplete="current-password"
            className="w-full px-4 py-3.5 bg-dark-900 border-2 border-dark-700 rounded-xl text-white placeholder-gray-500 outline-none focus:border-accent-500 transition-colors"
          />
          <button
            type="submit"
            disabled={loading || !password}
            className="w-full mt-4 py-3.5 font-semibold bg-accent-500 hover:bg-accent-600 text-white rounded-xl transition active:scale-[0.98] disabled:opacity-50 disabled:active:scale-100"
          >
            {loading ? 'Logging in…' : 'Log in'}
          </button>
        </form>
      </div>
      <a
        href="https://www.flaticon.com/free-icons/clipboard"
        title="clipboard icons"
        target="_blank"
        rel="noopener noreferrer"
        className="mt-6 text-xs text-gray-500 hover:text-gray-400 transition-colors"
      >
        Clipboard icons created by itim2101 - Flaticon
      </a>
    </div>
  )
}

export default Login
