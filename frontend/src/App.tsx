import { useCallback, useState } from 'react'
import Login from './Login'
import Home from './Home'
import { clearToken, getToken, setToken } from './api'

// One screen: drop old routes (/files, /login). Reloaded with the viewer open: leave its
// history entry so Back isn't a no-op.
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
