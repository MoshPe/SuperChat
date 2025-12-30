import React, { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { 
  MessageSquare, 
  Users, 
  User, 
  LogOut, 
  Menu, 
  X,
  Plus,
  Search
} from 'lucide-react'
import Sidebar from './Sidebar'

const Layout = () => {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const isActive = (path) => {
    return location.pathname === path
  }

  return (
    <div className="main-container min-h-screen bg-gray-50">
      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div 
          className="fixed inset-0 z-40 bg-gray-600 bg-opacity-75 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <Sidebar 
        isOpen={sidebarOpen} 
        onClose={() => setSidebarOpen(false)}
        user={user}
        onLogout={handleLogout}
      />

      {/* Main content */}
      <div className="right-container">
        {/* Top navigation */}
        <div className="sticky top-0 z-30 bg-white border-b border-gray-200">
          <div className="flex items-center justify-between px-4 py-3">
            {/* Mobile menu button */}
            <button
              type="button"
              className="lg:hidden p-2 rounded-md text-gray-400 hover:text-gray-500 hover:bg-gray-100"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu className="h-6 w-6" />
            </button>

            {/* Page title */}
            <div className="flex-1 min-w-0">
              <h1 className="text-lg font-semibold text-gray-900 truncate">
                {location.pathname === '/dashboard' && 'Dashboard'}
                {location.pathname.startsWith('/chat/') && 'Team Chat'}
                {location.pathname === '/profile' && 'Profile'}
              </h1>
            </div>

            {/* User menu */}
            <div className="flex items-center space-x-3">
              <button
                onClick={() => navigate('/profile')}
                className="flex items-center space-x-2 p-2 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <span className="hidden sm:block text-sm font-medium text-gray-700">
                  {user?.name || user?.username}
                </span>
              </button>
              
              <button
                onClick={handleLogout}
                className="p-2 text-gray-400 hover:text-gray-500 hover:bg-gray-100 rounded-lg transition-colors"
                title="Logout"
              >
                <LogOut className="h-5 w-5" />
              </button>
            </div>
          </div>
        </div>

        {/* Page content */}
        <main>
          <Outlet />
        </main>
      </div>
    </div>
  )
}

export default Layout
