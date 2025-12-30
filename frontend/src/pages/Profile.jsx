import React, { useEffect, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { User, Save, X, Lock, Shield } from 'lucide-react'

const Profile = () => {
  const { user, updateProfile, changePassword } = useAuth()
  const [isEditing, setIsEditing] = useState(false)
  const [formData, setFormData] = useState({
    username: user?.username || '',
    name: user?.name || '',
  })
  const [loading, setLoading] = useState(false)
  const [pwdLoading, setPwdLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [passwordSuccess, setPasswordSuccess] = useState('')
  const [passwordData, setPasswordData] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!formData.name.trim()) {
      setError('Name cannot be empty')
      return
    }

    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const result = await updateProfile({ name: formData.name.trim() })
      if (result.success) {
        setSuccess('Profile updated successfully!')
        setIsEditing(false)
        // Update local form data
        setFormData({
          username: result.user.username,
          name: result.user.name,
        })
      } else {
        setError(result.error)
      }
    } catch (error) {
      setError('Failed to update profile. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const handleChange = (e) => {
    const { name, value } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: value
    }))
    if (error) setError('')
    if (success) setSuccess('')
  }

  const handleCancel = () => {
    setFormData({
      username: user?.username || '',
      name: user?.name || '',
    })
    setIsEditing(false)
    setError('')
    setSuccess('')
  }

  const handlePasswordChange = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    setPasswordSuccess('')

    if (!passwordData.currentPassword || !passwordData.newPassword || !passwordData.confirmPassword) {
      setError('All password fields are required')
      return
    }
    if (passwordData.newPassword.length < 6) {
      setError('New password must be at least 6 characters')
      return
    }
    if (passwordData.newPassword !== passwordData.confirmPassword) {
      setError('Passwords do not match')
      return
    }

    setPwdLoading(true)
    try {
      const result = await changePassword(passwordData.currentPassword, passwordData.newPassword)
      if (result.success) {
        setPasswordSuccess('Password updated successfully!')
        setPasswordData({ currentPassword: '', newPassword: '', confirmPassword: '' })
      } else {
        setError(result.error)
      }
    } catch (err) {
      setError('Failed to change password. Please try again.')
    } finally {
      setPwdLoading(false)
    }
  }

  const handlePasswordInputChange = (e) => {
    const { name, value } = e.target
    setPasswordData((prev) => ({ ...prev, [name]: value }))
    if (error) setError('')
    if (passwordSuccess) setPasswordSuccess('')
  }

  // Sync local form state when user changes
  useEffect(() => {
    if (user) {
      setFormData({
        username: user.username,
        name: user.name || '',
      })
    }
  }, [user])

  if (!user) {
    return (
      <div className="flex items-center justify-center min-h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Profile</h1>
        <p className="text-gray-600">Manage your account settings and preferences</p>
      </div>

      {/* Profile card */}
      <div className="card p-8">
        <form onSubmit={handleSubmit} className="space-y-6">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          {success && (
            <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg text-sm">
              {success}
            </div>
          )}

          {/* Name field (editable) */}
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-2">
              Name
            </label>
            <input
              id="name"
              name="name"
              type="text"
              value={isEditing ? formData.name : user.name}
              onChange={handleChange}
              disabled={!isEditing}
              className="input disabled:bg-gray-50 disabled:text-gray-500"
              placeholder="Enter your name"
            />
          </div>

          {/* Username field (immutable) */}
          <div>
            <label htmlFor="username" className="block text-sm font-medium text-gray-700 mb-2">
              Username (cannot be changed)
            </label>
            <input
              id="username"
              name="username"
              type="text"
              value={user.username}
              disabled
              className="input disabled:bg-gray-50 disabled:text-gray-500"
              placeholder="Enter username"
            />
          </div>

          {/* Account info */}
          <div className="bg-gray-50 rounded-lg p-4 space-y-3">
            <h3 className="font-medium text-gray-900">Account Information</h3>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-500">Member since:</span>
                <p className="font-medium text-gray-900">
                  {new Date(user.created_at).toLocaleDateString()}
                </p>
              </div>
              <div>
                <span className="text-gray-500">Last updated:</span>
                <p className="font-medium text-gray-900">
                  {new Date(user.updated_at).toLocaleDateString()}
                </p>
              </div>
            </div>
          </div>

          {/* Action buttons */}
          <div className="flex justify-end space-x-3 pt-4">
            {!isEditing ? (
              <button
                type="button"
                onClick={() => setIsEditing(true)}
                className="btn-primary flex items-center space-x-2"
              >
                <User className="w-4 h-4" />
                <span>Edit Profile</span>
              </button>
            ) : (
              <>
                <button
                  type="button"
                  onClick={handleCancel}
                  className="btn-secondary flex items-center space-x-2"
                  disabled={loading}
                >
                  <X className="w-4 h-4" />
                  <span>Cancel</span>
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="btn-primary flex items-center space-x-2"
                >
                  {loading ? (
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                  ) : (
                    <Save className="w-4 h-4" />
                  )}
                  <span>{loading ? 'Saving...' : 'Save Changes'}</span>
                </button>
              </>
            )}
          </div>
        </form>
      </div>

      {/* Security section */}
      <div className="card p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Security</h3>
        <form className="space-y-4" onSubmit={handlePasswordChange}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="currentPassword" className="block text-sm font-medium text-gray-700 mb-2">
                Current password
              </label>
              <input
                id="currentPassword"
                name="currentPassword"
                type="password"
                value={passwordData.currentPassword}
                onChange={handlePasswordInputChange}
                className="input"
                placeholder="Enter current password"
              />
            </div>
            <div>
              <label htmlFor="newPassword" className="block text-sm font-medium text-gray-700 mb-2">
                New password
              </label>
              <input
                id="newPassword"
                name="newPassword"
                type="password"
                value={passwordData.newPassword}
                onChange={handlePasswordInputChange}
                className="input"
                placeholder="Enter new password"
              />
              <p className="text-xs text-gray-500 mt-1">Minimum 6 characters</p>
            </div>
            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-2">
                Confirm new password
              </label>
              <input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                value={passwordData.confirmPassword}
                onChange={handlePasswordInputChange}
                className="input"
                placeholder="Re-enter new password"
              />
            </div>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center text-sm text-gray-600 space-x-2">
              <Lock className="w-4 h-4" />
              <span>We never store plaintext passwords.</span>
            </div>
            <button
              type="submit"
              disabled={pwdLoading}
              className="btn-primary flex items-center space-x-2"
            >
              {pwdLoading ? (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              ) : (
                <Shield className="w-4 h-4" />
              )}
              <span>{pwdLoading ? 'Updating...' : 'Change Password'}</span>
            </button>
          </div>
          {passwordSuccess && (
            <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg text-sm">
              {passwordSuccess}
            </div>
          )}
        </form>
        <p className="text-xs text-gray-500 mt-3">
          Two-factor authentication and other security features coming soon.
        </p>
      </div>

      {/* Danger zone */}
      <div className="card p-6 border-red-200 bg-red-50">
        <h3 className="text-lg font-medium text-red-900 mb-4">Danger Zone</h3>
        <div className="space-y-3">
          <div className="flex items-center justify-between p-3 bg-red-100 rounded-lg">
            <div>
              <p className="font-medium text-red-900">Delete Account</p>
              <p className="text-sm text-red-700">
                Permanently delete your account and all associated data
              </p>
            </div>
            <button
              type="button"
              className="btn bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 text-sm"
              disabled
            >
              Delete Account
            </button>
          </div>
        </div>
        <p className="text-xs text-red-600 mt-3">
          Account deletion is not available in this version
        </p>
      </div>
    </div>
  )
}

export default Profile
