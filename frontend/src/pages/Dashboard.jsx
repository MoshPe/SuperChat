import React, { useState, useEffect, useRef } from 'react' 
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { 
  MessageSquare, 
  Users, 
  Plus, 
  Calendar,
  TrendingUp,
  Activity
} from 'lucide-react'
import api from '../services/api'
import CreateTeamModal from '../components/CreateTeamModal'

const Dashboard = () => {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [teams, setTeams] = useState([])
  const [recentMessages, setRecentMessages] = useState([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [avatarUrls, setAvatarUrls] = useState({})
  const avatarUrlsRef = useRef({})
  const [onlineCounts, setOnlineCounts] = useState({})
  const [stats, setStats] = useState({
    totalTeams: 0,
    totalMessages: 0,
    activeTeams: 0
  })

  useEffect(() => {
    fetchDashboardData()
  }, [])

  const fetchDashboardData = async () => {
    try {
      setLoading(true)
      
      // Fetch teams and online counts together
      const [teamsResponse, onlineResponse] = await Promise.all([
        api.get('/teams'),
        api.get('/teams/online').catch(() => ({ data: { data: {} } }))
      ])
      const teamsData = teamsResponse.data.data || []
      const onlineData = onlineResponse.data.data || {}
      setTeams(teamsData)
      // normalize counts to numbers
      const normalizedOnline = Object.fromEntries(
        Object.entries(onlineData).map(([k, v]) => [k, Number(v) || 0])
      )
      setOnlineCounts(normalizedOnline)
      
      // Fetch messages per team (larger window for stats)
      const allMessages = []
      for (const team of teamsData) {
        try {
          const messagesResponse = await api.get(`/teams/${team.id}/messages?limit=200`)
          const teamMessages = messagesResponse.data.data || []
          allMessages.push(...teamMessages.map(msg => ({ ...msg, teamName: team.name })))
        } catch (error) {
          console.error(`Failed to fetch messages for team ${team.id}:`, error)
        }
      }
      
      // Sort by timestamp and take most recent for the activity list
      const sortedMessages = allMessages
        .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        .slice(0, 10)
      
      setRecentMessages(sortedMessages)
      
      // Calculate stats (using fetched messages set)
      const activeCutoff = Date.now() - 24 * 60 * 60 * 1000
      setStats({
        totalTeams: teamsData.length,
        totalMessages: allMessages.length,
        activeTeams: teamsData.filter(team => (normalizedOnline[team.id] || 0) > 0).length
      })
      
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error)
    } finally {
      setLoading(false)
    }
  }

  // Load team avatars for cards (protected)
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

  // Cleanup cached avatars no longer referenced
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

  const formatTimeAgo = (timestamp) => {
    const now = new Date()
    const messageTime = new Date(timestamp)
    const diffInMinutes = Math.floor((now - messageTime) / (1000 * 60))
    
    if (diffInMinutes < 1) return 'Just now'
    if (diffInMinutes < 60) return `${diffInMinutes}m ago`
    if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)}h ago`
    return `${Math.floor(diffInMinutes / 1440)}d ago`
  }

  const handleCreateTeam = async (teamData) => {
    const response = await api.post('/teams', teamData)
    const newTeam = response.data.data
    setTeams(prev => [newTeam, ...prev])
    setShowCreateModal(false)
    navigate(`/chat/${newTeam.id}`)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6" style={{ marginLeft: '10px', marginRight: '10px', marginTop: '5px'}}>
      {/* Welcome header */}
      <div className="bg-gradient-to-r from-primary-500 to-primary-600 rounded-2xl p-6 text-white">
        <div className="flex items-center space-x-4">

          <div>
          <h1 className="text-2xl font-bold">Welcome back, {user?.name || user?.username}!</h1>
            <p className="text-primary-100">Ready to collaborate with your teams?</p>
          </div>
        </div>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="card p-6">
          <div className="flex items-center space-x-3">
            <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center">
              <Users className="w-6 h-6 text-blue-600" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-300">Total Teams</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.totalTeams}</p>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center space-x-3">
            <div className="w-12 h-12 bg-green-100 rounded-xl flex items-center justify-center">
              <MessageSquare className="w-6 h-6 text-green-600" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-300">Total Messages</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.totalMessages}</p>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center space-x-3">
            <div className="w-12 h-12 bg-purple-100 rounded-xl flex items-center justify-center">
              <Activity className="w-6 h-6 text-purple-600" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-300">Active Teams</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.activeTeams}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Teams section */}
      <div className="card p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Your Teams</h2>
          <button
            onClick={() => setShowCreateModal(true)}
            className="btn-primary flex items-center space-x-2"
          >
            <Plus className="w-4 h-4" />
            <span>Create Team</span>
          </button>
        </div>

        {teams.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <Users className="w-8 h-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No teams yet</h3>
            <p className="text-gray-500 dark:text-gray-300 mb-6">Create your first team to start collaborating</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="btn-primary"
            >
              Create Team
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {teams.map((team) => (
              <Link
                key={team.id}
                to={`/chat/${team.id}`}
                className="block p-4 border border-gray-200 dark:border-gray-700 rounded-xl hover:border-primary-300 dark:hover:border-primary-400 hover:shadow-md dark:hover:shadow-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-all duration-200"
              >
                <div className="flex items-center space-x-3 mb-3">
                  <div className="w-10 h-10 bg-gradient-to-br from-gray-400 to-gray-500 rounded-lg flex items-center justify-center overflow-hidden">
                    {team.avatar && avatarUrls[team.avatar] ? (
                      <img src={avatarUrls[team.avatar]} alt={team.name} className="w-full h-full object-cover" />
                    ) : (
                      <Users className="w-5 h-5 text-white" />
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="font-medium text-gray-900 dark:text-gray-100 truncate">{team.name}</h3>
                    {team.description && (
                      <p className="text-sm text-gray-500 dark:text-gray-300 truncate">{team.description}</p>
                    )}
                  </div>
                </div>
                <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                  <span>Created {formatTimeAgo(team.created_at)}</span>
                  <span className="text-primary-600 hover:text-primary-700">Open chat ?</span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>

      {/* Recent activity */}
      {recentMessages.length > 0 && (
        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">Recent Activity</h2>
          <div className="space-y-4">
            {recentMessages.map((message) => (
              <div key={message.id} className="flex items-start space-x-3 p-3 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2 mb-1">
                    <span className="font-medium text-gray-900 dark:text-gray-100">{message.username}</span>
                    <span className="text-sm text-gray-500 dark:text-gray-300">in</span>
                    <span className="text-sm font-medium text-primary-600">{message.teamName}</span>
                  </div>
                  <p className="text-sm text-gray-700 dark:text-gray-200 mb-1">{message.content}</p>
                  <div className="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-300">
                    <Calendar className="w-3 h-3" />
                    <span>{formatTimeAgo(message.created_at)}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

          {/* Create Team Modal */}
      <CreateTeamModal 
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSubmit={handleCreateTeam}
      />
    </div>
  )
}

export default Dashboard
