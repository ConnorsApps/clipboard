import { useNavigate } from 'react-router-dom'

type Tab = 'clipboard' | 'files'

interface LayoutProps {
  children: React.ReactNode
  currentTab: Tab
}

function Layout({ children, currentTab }: LayoutProps) {
  const navigate = useNavigate()

  const handleLogout = () => {
    localStorage.removeItem('clipboard_token')
    navigate('/login')
  }

  const handleTabClick = (tab: Tab) => {
    navigate(tab === 'clipboard' ? '/' : '/files')
  }

  return (
    <div className="min-h-screen bg-dark-900 flex flex-col">
      {/* Top Navigation Bar - Fixed to top with safe area support */}
      <div className="fixed top-0 left-0 right-0 z-50 bg-dark-800/95 backdrop-blur-md border-b border-dark-700 pt-[env(safe-area-inset-top)]">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          {/* Tab Navigation */}
          <div className="flex gap-1 bg-dark-900 rounded-lg p-1">
            <button
              onClick={() => handleTabClick('clipboard')}
              className={`px-6 py-2 rounded-md font-semibold transition-colors ${
                currentTab === 'clipboard'
                  ? 'bg-accent-500 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-dark-700'
              }`}
            >
              Clipboard
            </button>
            <button
              onClick={() => handleTabClick('files')}
              className={`px-6 py-2 rounded-md font-semibold transition-colors ${
                currentTab === 'files'
                  ? 'bg-accent-500 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-dark-700'
              }`}
            >
              Files
            </button>
          </div>

          {/* Logout Button */}
          <button
            onClick={handleLogout}
            className="text-sm text-gray-500 hover:text-accent-500 transition-colors px-4 py-2"
          >
            Logout
          </button>
        </div>
      </div>

      {/* Main Content - Adjusted for fixed header with safe area */}
      <div className="flex-1 overflow-y-auto pt-[calc(5.5rem+env(safe-area-inset-top))] pb-[env(safe-area-inset-bottom)] px-4 pb-4" style={{ WebkitOverflowScrolling: 'touch' }}>
        <div className="flex items-center justify-center min-h-full">
          {children}
        </div>
      </div>
    </div>
  )
}

export default Layout
