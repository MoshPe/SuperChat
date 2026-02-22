import React, { useEffect, useRef, useState } from 'react'
import { X, Trash2, LogOut, Upload, XCircle } from 'lucide-react'
import api from '../services/api'

const TeamSettingsModal = ({
  isOpen,
  onClose,
  team,
  onSave,
  onDelete,
  onLeave,
  isOwner,
  onUploadAvatar,
}) => {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [avatar, setAvatar] = useState('')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [previewUrl, setPreviewUrl] = useState('')
  const fileInputRef = useRef(null)
  const previewObjectUrlRef = useRef('')

  useEffect(() => {
    if (isOpen && team) {
      setName(team.name || '')
      setDescription(team.description || '')
      setAvatar(team.avatar || '')
    }
  }, [isOpen, team])

  // Build a preview URL for secured uploads
  useEffect(() => {
    let cancelled = false
    const load = async () => {
      if (!avatar) {
        if (previewObjectUrlRef.current) {
          URL.revokeObjectURL(previewObjectUrlRef.current)
          previewObjectUrlRef.current = ''
        }
        setPreviewUrl('')
        return
      }
      // If absolute URL, just use it
      if (/^https?:\/\//i.test(avatar)) {
        if (previewObjectUrlRef.current) {
          URL.revokeObjectURL(previewObjectUrlRef.current)
          previewObjectUrlRef.current = ''
        }
        setPreviewUrl(avatar)
        return
      }
      try {
        let path = avatar
        if (path.startsWith('/api/')) {
          path = path.replace(/^\/api/, '')
        }
        const res = await api.get(path, { responseType: 'blob' })
        const url = URL.createObjectURL(res.data)
        if (cancelled) {
          URL.revokeObjectURL(url)
          return
        }
        if (previewObjectUrlRef.current) {
          URL.revokeObjectURL(previewObjectUrlRef.current)
        }
        previewObjectUrlRef.current = url
        setPreviewUrl(url)
      } catch (e) {
        console.error('Failed to load avatar preview', e)
        setPreviewUrl('')
      }
    }
    load()
    return () => {
      cancelled = true
    }
  }, [avatar])

  useEffect(() => {
    return () => {
      if (previewObjectUrlRef.current) {
        URL.revokeObjectURL(previewObjectUrlRef.current)
        previewObjectUrlRef.current = ''
      }
    }
  }, [])

  if (!isOpen) return null

  const handleSave = async (e) => {
    e.preventDefault()
    setSaving(true)
    try {
      await onSave({
        name: name.trim(),
        description: description.trim(),
        avatar: avatar.trim(),
      })
    } finally {
      setSaving(false)
    }
  }

  const handleFileSelect = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const url = await onUploadAvatar(file)
      if (url) setAvatar(url)
    } catch (err) {
      console.error(err)
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const confirmLeave = () => {
    if (window.confirm('Leave this team?')) {
      onLeave()
    }
  }

  const confirmDelete = () => {
    if (window.confirm('Delete this team for everyone? This cannot be undone.')) {
      onDelete()
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center px-4 py-6">
      <div className="absolute inset-0 bg-gray-900 bg-opacity-60" onClick={onClose} />
      <div className="relative w-full max-w-xl bg-white dark:bg-gray-800 rounded-2xl shadow-2xl border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Team settings</h3>
            <p className="text-sm text-gray-600 dark:text-gray-300">Update team details or leave the team.</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">Name</label>
            <input
              className="input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              disabled={!isOwner}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">Description</label>
            <textarea
              className="input"
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={!isOwner}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">Avatar (image upload)</label>
            <div className="mt-2 flex items-center space-x-3">
              <div className="w-14 h-14 min-w-[3.5rem] rounded-lg bg-gray-100 dark:bg-gray-700 overflow-hidden flex items-center justify-center">
                {previewUrl ? (
                  <img src={previewUrl} alt="avatar" className="w-full h-full object-cover" />
                ) : (
                  <span className="text-[11px] text-gray-500 text-center px-1 leading-tight">No image</span>
                )}
              </div>
              <div className="flex items-center space-x-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={handleFileSelect}
                  disabled={!isOwner || uploading}
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={!isOwner || uploading}
                  className="btn-outline flex items-center space-x-2 disabled:opacity-50"
                >
                  <Upload className="w-4 h-4" />
                  <span>{uploading ? 'Uploading...' : 'Upload image'}</span>
                </button>
                {avatar && (
                  <button
                    type="button"
                    onClick={() => { setAvatar(''); setPreviewUrl('') }}
                    className="btn-outline flex items-center space-x-1 text-red-600 border-red-300 hover:bg-red-50 dark:text-red-300 dark:border-red-700 dark:hover:bg-red-900/40"
                  >
                    <XCircle className="w-4 h-4" />
                    <span>Remove</span>
                  </button>
                )}
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end pt-4 space-x-2">
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={saving || !isOwner} className="btn-primary">
              {saving ? 'Saving...' : 'Save changes'}
            </button>
          </div>

          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <button
              type="button"
              onClick={confirmLeave}
              className="btn-outline flex items-center space-x-2"
            >
              <LogOut className="w-4 h-4" />
              <span>Leave team</span>
            </button>
            {isOwner && (
              <button
                type="button"
                onClick={confirmDelete}
                className="btn bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 flex items-center space-x-2"
              >
                <Trash2 className="w-4 h-4" />
                <span>Delete team</span>
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  )
}

export default TeamSettingsModal
