import { useCallback, useState } from 'react'
import Login from './Login'
import Home from './Home'
import { clearToken, getToken, setToken } from './api'

// Everything lives on one screen: tidy up old routes like /files and /login. If the page was
// reloaded with the file viewer open, step back off its history entry so Back isn't a no-op.
if (window.history.state?.viewer) {
  window.history.back()
} else {
  window.history.replaceState(null, '', '/')
}

function App() {
  const [token, setTokenState] = useState(getToken)

  const handleLogin = (newToken: string) => {
    setToken(newToken)
    setTokenState(newToken)
  }

  const handleLogout = useCallback(() => {
    clearToken()
    setTokenState(null)
  }, [])

  return token ? <Home key={token} onLogout={handleLogout} /> : <Login onLogin={handleLogin} />
}

export default App
