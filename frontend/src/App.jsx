import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import { useStatus } from './contexts/StatusContext'
import Login from './pages/Login'
import Register from './pages/Register'
import Dashboard from './pages/Dashboard'
import Chat from './pages/Chat'
import Profile from './pages/Profile'
import Layout from './components/Layout'

const ServerBanner = () => {
  const { serverDown } = useStatus()
  if (!serverDown) return null
  return (
    <div className="fixed top-4 left-1/2 transform -translate-x-1/2 z-50">
      <div className="bg-white border border-red-200 text-red-700 shadow-lg rounded-lg px-4 py-3 text-sm font-medium flex items-center space-x-2">
        <span className="inline-block w-2 h-2 rounded-full bg-red-500"></span>
        <span>{serverDown}</span>
      </div>
    </div>
  )
}

// Protected route component
const ProtectedRoute = ({ children }) => {
  const { user, loading } = useAuth()
  const { serverDown } = useStatus()
  
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }
  
  if (serverDown) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <div className="bg-white border border-red-200 text-red-700 shadow-lg rounded-lg px-6 py-4 text-center space-y-2">
          <div className="text-lg font-semibold">Not connected to server</div>
          <p className="text-sm text-gray-600">Please check your connection and try again.</p>
        </div>
      </div>
    )
  }
  
  return user ? children : <Navigate to="/login" replace />
}

// Public route component (redirects if already authenticated)
const PublicRoute = ({ children }) => {
  const { user, loading } = useAuth()
  
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }
  
  return user ? <Navigate to="/dashboard" replace /> : children
}

function AppRoutes() {
  return (
    <>
      <ServerBanner />
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={
          <PublicRoute>
            <Login />
          </PublicRoute>
        } />
        <Route path="/register" element={
          <PublicRoute>
            <Register />
          </PublicRoute>
        } />

        {/* Redirect bare /chat paths to login to avoid loops on refresh without team context */}
        <Route path="/chat" element={<Navigate to="/login" replace />} />
        <Route path="/chat/*" element={<Navigate to="/login" replace />} />
        
        {/* Protected routes */}
        <Route path="/" element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="chat/:teamId" element={<Chat />} />
          <Route path="profile" element={<Profile />} />
        </Route>
        
        {/* Catch all route */}
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </>
  )
}

function App() {
  return (
    <AuthProvider>
      <AppRoutes />
    </AuthProvider>
  )
}

export default App
