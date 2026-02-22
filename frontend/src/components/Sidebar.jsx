import React, { useState, useEffect, useMemo, useRef } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { 
  MessageSquare, 
  Users, 
  User, 
  LogOut, 
  X,
  Plus,
  Search
} from 'lucide-react'
import api from '../services/api'
import CreateTeamModal from './CreateTeamModal'

const Sidebar = ({ isOpen, onClose, user, onLogout }) => {
  const navigate = useNavigate()
  const location = useLocation()
  const [teams, setTeams] = useState([])
  const [loading, setLoading] = useState(true)
  const [searchTerm, setSearchTerm] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [onlineCounts, setOnlineCounts] = useState({})
  const firstLoadRef = useRef(true)
  const [avatarUrls, setAvatarUrls] = useState({})
  const avatarUrlsRef = useRef({})

  const teamsEqual = (a = [], b = []) => {
    if (a.length !== b.length) return false
    for (let i = 0; i < a.length; i++) {
      if (
        a[i].id !== b[i].id ||
        a[i].name !== b[i].name ||
        a[i].description !== b[i].description ||
        a[i].avatar !== b[i].avatar
      ) {
        return false
      }
    }
    return true
  }

  useEffect(() => {
    // Initial fetch
    fetchTeams()
    fetchOnlineCounts()
    // Poll every 2 seconds to reflect membership changes quickly
    const interval = setInterval(() => {
      fetchTeams()
      fetchOnlineCounts()
    }, 2000)

    // Refresh on tab focus/visibility change
    const handleVisibility = () => {
      if (document.visibilityState === 'visible') {
        fetchTeams()
        fetchOnlineCounts()
      }
    }
    document.addEventListener('visibilitychange', handleVisibility)

    return () => {
      clearInterval(interval)
      document.removeEventListener('visibilitychange', handleVisibility)
    }
  }, [])

  // Fetch and cache avatar blobs for teams (protected endpoints)
  useEffect(() => {
    avatarUrlsRef.current = avatarUrls
  }, [avatarUrls])

  useEffect(() => {
    const loadAvatars = async () => {
      for (const t of teams) {
        if (!t.avatar) continue
        if (avatarUrlsRef.current[t.avatar]) continue
        let path = t.avatar
        if (path.startsWith('/api/')) {
          path = path.replace(/^\/api/, '')
        }
        try {
          const res = await api.get(path, { responseType: 'blob' })
          const url = URL.createObjectURL(res.data)
          setAvatarUrls(prev => ({ ...prev, [t.avatar]: url }))
        } catch (e) {
          console.error('Failed to load avatar', e)
        }
      }
    }
    loadAvatars()
  }, [teams])

  useEffect(() => {
    return () => {
      Object.values(avatarUrlsRef.current).forEach(url => URL.revokeObjectURL(url))
    }
  }, [])

  // Cleanup cached avatars that are no longer referenced
  useEffect(() => {
    setAvatarUrls(prev => {
      const used = new Set(teams.filter(t => t.avatar).map(t => t.avatar))
      const next = { ...prev }
      Object.keys(next).forEach(k => {
        if (!used.has(k)) {
          URL.revokeObjectURL(next[k])
          delete next[k]
        }
      })
      return next
    })
  }, [teams])

  const fetchTeams = async () => {
    try {
      if (firstLoadRef.current) {
        setLoading(true)
      }
      const response = await api.get('/teams')
      const incoming = (response.data.data || []).slice().sort((a, b) => a.name.localeCompare(b.name))
      setTeams(prev => {
        const sortedPrev = (prev || []).slice().sort((a, b) => a.name.localeCompare(b.name))
        if (teamsEqual(sortedPrev, incoming)) return prev
        return incoming
      })
    } catch (error) {
      console.error('Failed to fetch teams:', error)
    } finally {
      firstLoadRef.current = false
      setLoading(false)
    }
  }

  const fetchOnlineCounts = async () => {
    try {
      const response = await api.get('/teams/online')
      const data = response.data.data || {}
      setOnlineCounts(prev => {
        const same = Object.keys(data).length === Object.keys(prev).length &&
          Object.keys(data).every(k => data[k] === prev[k])
        return same ? prev : data
      })
    } catch (error) {
      console.error('Failed to fetch online counts:', error)
    }
  }

  const handleCreateTeam = async (teamData) => {
    try {
      const response = await api.post('/teams', teamData)
      const newTeam = response.data.data
      setTeams(prev => [newTeam, ...prev])
      setShowCreateModal(false)
      navigate(`/chat/${newTeam.id}`)
    } catch (error) {
      console.error('Failed to create team:', error)
      throw error
    }
  }

  const filteredTeams = useMemo(() => {
    return teams.filter(team =>
      team.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      team.description?.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [teams, searchTerm])

  const isActive = (path) => {
    return location.pathname === path
  }

  const isTeamActive = (teamId) => {
    return location.pathname === `/chat/${teamId}`
  }

  return (
    <>
      {/* Sidebar */}
      <div className={`
        fixed inset-y-0 left-0 z-50 min-w-[16rem] w-80 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 transform transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:inset-0
        ${isOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        {/* Sidebar header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center space-x-2">
            <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-primary-600 rounded-lg flex items-center justify-center">
              <MessageSquare className="w-5 h-5 text-white" />
            </div>
            <h1 className="text-xl font-bold text-gradient">SuperChat</h1>
          </div>
          <button
            onClick={onClose}
            className="lg:hidden p-1 rounded-md text-gray-400 hover:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* User quick actions at top */}
        <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <button
              onClick={() => navigate('/profile')}
              className="text-sm font-medium text-gray-900 dark:text-gray-100 hover:text-primary-600 truncate"
              title="View profile"
            >
              {user?.name || user?.username}
            </button>
            <button
              onClick={onLogout}
              className="p-1 text-gray-400 hover:text-red-500 transition-colors"
              title="Logout"
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Navigation */}
        <nav className="p-4 space-y-2">
          <Link
            to="/dashboard"
            className={`flex items-center space-x-3 px-3 py-2 rounded-lg transition-colors ${
              isActive('/dashboard')
                ? 'bg-primary-50 text-primary-700 border border-primary-200 dark:bg-gray-700 dark:text-primary-100 dark:border-gray-600'
                : 'text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-700'
            }`}
          >
            <Users className="w-5 h-5" />
            <span>Dashboard</span>
          </Link>
          
          {/* Profile link removed; username at top now navigates to profile */}
        </nav>

        {/* Teams section */}
        <div className="p-4 border-t border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100 uppercase tracking-wider">
              Teams
            </h2>
            <button
              onClick={() => setShowCreateModal(true)}
              className="p-1 text-gray-400 hover:text-primary-500 transition-colors"
              title="Create new team"
            >
              <Plus className="w-4 h-4" />
            </button>
          </div>

          {/* Search */}
          <div className="relative mb-3">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search teams..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-3 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent dark:bg-gray-900 dark:border-gray-700 dark:text-gray-100"
            />
          </div>

          {/* Teams list */}
          <div className="space-y-1">
            {loading ? (
              <div className="flex items-center justify-center py-4">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary-600"></div>
              </div>
            ) : filteredTeams.length === 0 ? (
              <div className="text-center py-4 text-sm text-gray-500">
                {searchTerm ? 'No teams found' : 'No teams yet'}
              </div>
            ) : (
              filteredTeams.map((team) => (
                <TeamListItem
                  key={team.id}
                  team={team}
                  active={isTeamActive(team.id)}
                  online={onlineCounts[team.id] || 0}
                  avatarUrl={team.avatar ? avatarUrls[team.avatar] : ''}
                />
              ))
            )}
          </div>
        </div>

        {/* User info moved to top */}
      </div>

      {/* Create Team Modal */}
      <CreateTeamModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSubmit={handleCreateTeam}
      />
    </>
  )
}

export default Sidebar

const TeamListItem = React.memo(function TeamListItem({ team, active, online, avatarUrl }) {
  return (
    <Link
      to={`/chat/${team.id}`}
      className={`flex items-center space-x-3 px-3 py-2 rounded-lg transition-colors ${
        active
          ? 'bg-primary-50 text-primary-700 border border-primary-200 dark:bg-gray-700 dark:text-primary-100 dark:border-primary-300'
          : 'text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-700 dark:hover:border-gray-600 border border-transparent'
      }`}
    >
      <div className="w-8 h-8 bg-gradient-to-br from-gray-400 to-gray-500 rounded-lg flex items-center justify-center overflow-hidden">
        {avatarUrl ? (
          <img src={avatarUrl} alt={team.name} className="w-full h-full object-cover" />
        ) : (
          <Users className="w-4 h-4 text-white" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{team.name}</p>
        {team.description && (
          <p className="text-xs text-gray-500 truncate">{team.description}</p>
        )}
      </div>
      <div className="ml-auto flex items-center space-x-1 text-xs text-gray-500">
        <span
          className={`inline-block w-2 h-2 rounded-full ${
            online > 0 ? 'bg-green-500' : 'bg-gray-300'
          }`}
        />
        <span>{online}</span>
      </div>
    </Link>
  )
})
