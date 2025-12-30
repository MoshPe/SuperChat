import React, { useState, useEffect } from 'react'
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

  useEffect(() => {
    // Initial fetch
    fetchTeams()
    fetchOnlineCounts()
    // Poll every 10 seconds
    const interval = setInterval(() => {
      fetchTeams()
      fetchOnlineCounts()
    }, 10000)
    return () => clearInterval(interval)
  }, [])

  const fetchTeams = async () => {
    try {
      setLoading(true)
      const response = await api.get('/teams')
      setTeams(response.data.data || [])
    } catch (error) {
      console.error('Failed to fetch teams:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchOnlineCounts = async () => {
    try {
      const response = await api.get('/teams/online')
      setOnlineCounts(response.data.data || {})
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

  const filteredTeams = teams.filter(team =>
    team.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    team.description?.toLowerCase().includes(searchTerm.toLowerCase())
  )

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
        fixed inset-y-0 left-0 z-50 min-w-[16rem] w-80 bg-white border-r border-gray-200 transform transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:inset-0
        ${isOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        {/* Sidebar header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <div className="flex items-center space-x-2">
            <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-primary-600 rounded-lg flex items-center justify-center">
              <MessageSquare className="w-5 h-5 text-white" />
            </div>
            <h1 className="text-xl font-bold text-gradient">SuperChat</h1>
          </div>
          <button
            onClick={onClose}
            className="lg:hidden p-1 rounded-md text-gray-400 hover:text-gray-500 hover:bg-gray-100"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* User quick actions at top */}
        <div className="px-4 py-3 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <button
              onClick={() => navigate('/profile')}
              className="text-sm font-medium text-gray-900 hover:text-primary-600 truncate"
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
                ? 'bg-primary-50 text-primary-700 border border-primary-200'
                : 'text-gray-700 hover:bg-gray-50'
            }`}
          >
            <Users className="w-5 h-5" />
            <span>Dashboard</span>
          </Link>
          
          {/* Profile link removed; username at top now navigates to profile */}
        </nav>

        {/* Teams section */}
        <div className="p-4 border-t border-gray-200">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider">
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
              className="w-full pl-10 pr-3 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
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
                <Link
                  key={team.id}
                  to={`/chat/${team.id}`}
                  className={`flex items-center space-x-3 px-3 py-2 rounded-lg transition-colors ${
                    isTeamActive(team.id)
                      ? 'bg-primary-50 text-primary-700 border border-primary-200'
                      : 'text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  <div className="w-8 h-8 bg-gradient-to-br from-gray-400 to-gray-500 rounded-lg flex items-center justify-center">
                    <Users className="w-4 h-4 text-white" />
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
                        (onlineCounts[team.id] || 0) > 0 ? 'bg-green-500' : 'bg-gray-300'
                      }`}
                    />
                    <span>{onlineCounts[team.id] || 0}</span>
                  </div>
                </Link>
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
