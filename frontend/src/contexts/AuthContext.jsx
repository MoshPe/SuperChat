import React, { createContext, useContext, useState, useEffect } from 'react'
import api from '../services/api'

const AuthContext = createContext()

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null)
  const [token, setToken] = useState(null)
  const [loading, setLoading] = useState(false)

  const login = async (username, password) => {
    try {
      const response = await api.post('/auth/login', { username, password })
      const { user, token } = response.data.data
      
      // Set in state and API headers
      setUser(user)
      setToken(token)
      api.defaults.headers.common['Authorization'] = `Bearer ${token}`
      
      return { success: true, user }
    } catch (error) {
      const message = error.response?.data?.error || 'Login failed'
      const status = error.response?.status
      return { success: false, error: message, status }
    }
  }

  const register = async (username, name, password) => {
    try {
      const response = await api.post('/auth/register', { username, name, password })
      const { user, token } = response.data.data
      
      // Set in state and API headers
      setUser(user)
      setToken(token)
      api.defaults.headers.common['Authorization'] = `Bearer ${token}`
      
      return { success: true, user }
    } catch (error) {
      const message = error.response?.data?.error || 'Registration failed'
      return { success: false, error: message }
    }
  }

  const logout = async () => {
    try {
      await api.post('/auth/logout')
    } catch (error) {
      console.error('Logout API call failed:', error)
    } finally {
      // Clear state and headers (no persistence)
      setUser(null)
      setToken(null)
      delete api.defaults.headers.common['Authorization']
    }
  }

  const updateProfile = async (profileData) => {
    try {
      const response = await api.put('/user/profile', profileData)
      const updatedUser = response.data.data
      
      // Update state only (no persistence)
      setUser(updatedUser)
      
      return { success: true, user: updatedUser }
    } catch (error) {
      const message = error.response?.data?.error || 'Profile update failed'
      return { success: false, error: message }
    }
  }

  const changePassword = async (currentPassword, newPassword) => {
    try {
      await api.put('/user/password', {
        current_password: currentPassword,
        new_password: newPassword,
      }, {
        skipAuthRedirect: true,
      })
      return { success: true }
    } catch (error) {
      const message = error.response?.data?.error || 'Password change failed'
      return { success: false, error: message }
    }
  }

  const value = {
    user,
    token,
    loading,
    login,
    register,
    logout,
    updateProfile,
    changePassword,
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}
