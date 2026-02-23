import React, { useEffect, useMemo, useState } from 'react'
import { X, UserPlus, UserMinus, Shield } from 'lucide-react'
import api from '../services/api'

const ManageMembersModal = ({ isOpen, onClose, teamId, ownerId }) => {
  const [users, setUsers] = useState([])
  const [members, setMembers] = useState([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [busyIds, setBusyIds] = useState(new Set())

  const memberIdSet = useMemo(() => new Set(members.map(m => m.user_id)), [members])

  const loadData = async () => {
    setLoading(true)
    try {
      const [usersRes, membersRes] = await Promise.all([
        api.get(`/users?team_id=${encodeURIComponent(teamId)}`),
        api.get(`/teams/${teamId}/members`),
      ])
      setUsers(usersRes.data.data || [])
      setMembers(membersRes.data.data || [])
    } catch (e) {
      console.error('Failed to load members/users', e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen && teamId) {
      loadData()
    }
  }, [isOpen, teamId])

  const filteredUsers = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return users
    return users.filter(u => u.username.toLowerCase().includes(term))
  }, [users, search])

  const addMember = async (username) => {
    setBusyIds(prev => new Set(prev).add(username))
    try {
      await api.post(`/teams/${teamId}/members`, { username })
      await loadData()
    } catch (e) {
      console.error('Failed to add member', e)
    } finally {
      setBusyIds(prev => { const n = new Set(prev); n.delete(username); return n })
    }
  }

  const removeMember = async (userId) => {
    setBusyIds(prev => new Set(prev).add(userId))
    try {
      await api.delete(`/teams/${teamId}/members/${userId}`)
      setMembers(prev => prev.filter(m => m.user_id !== userId))
    } catch (e) {
      console.error('Failed to remove member', e)
    } finally {
      setBusyIds(prev => { const n = new Set(prev); n.delete(userId); return n })
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        {/* Backdrop */}
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onClick={onClose} />

        <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-2xl sm:w-full">
          <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium text-gray-900">Manage Members</h3>
              <button onClick={onClose} className="text-gray-400 hover:text-gray-500">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mb-4">
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search users by username"
                className="input"
              />
            </div>

            <div className="max-h-96 overflow-y-auto divide-y divide-gray-100">
              {loading ? (
                <div className="py-8 text-center text-gray-500">Loading...</div>
              ) : (
                filteredUsers.map(u => {
                  const isMember = memberIdSet.has(u.id)
                  const isOwner = u.id === ownerId
                  return (
                    <div key={u.id} className="flex items-center justify-between py-3">
                      <div className="flex items-center space-x-3 min-w-0">
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-gray-900 truncate">{u.name}</p>
                          {isMember && (
                            <p className="text-xs text-gray-500 truncate">Member</p>
                          )}
                        </div>
                        {isOwner && (
                          <span className="ml-2 inline-flex items-center text-xs text-amber-700 bg-amber-100 px-2 py-0.5 rounded-full">
                            <Shield className="w-3 h-3 mr-1" /> Owner
                          </span>
                        )}
                      </div>
                      <div>
                        {isMember ? (
                          <button
                            onClick={() => removeMember(u.id)}
                            className="btn-outline flex items-center space-x-2 disabled:opacity-50"
                            disabled={isOwner || busyIds.has(u.id)}
                          >
                            <UserMinus className="w-4 h-4" />
                            <span>Remove</span>
                          </button>
                        ) : (
                          <button
                            onClick={() => addMember(u.username)}
                            className="btn-primary flex items-center space-x-2 disabled:opacity-50"
                            disabled={busyIds.has(u.username)}
                          >
                            <UserPlus className="w-4 h-4" />
                            <span>Add</span>
                          </button>
                        )}
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ManageMembersModal
