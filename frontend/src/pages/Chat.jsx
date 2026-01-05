import React, { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useStatus } from '../contexts/StatusContext'
import { 
  Send, 
  Users, 
  MoreVertical,
  LogOut,
  UserPlus,
  Settings
} from 'lucide-react'
import api from '../services/api'
import ManageMembersModal from '../components/ManageMembersModal'
import TeamSettingsModal from '../components/TeamSettingsModal'

const Chat = () => {
  const { teamId } = useParams()
  const navigate = useNavigate()
  const { user, token } = useAuth()
  const [team, setTeam] = useState(null)
  const [messages, setMessages] = useState([])
  const [newMessage, setNewMessage] = useState('')
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [pendingImage, setPendingImage] = useState(null)
  const [pendingPreview, setPendingPreview] = useState(null)
  const [imageUrls, setImageUrls] = useState({})
  const imageUrlsRef = useRef({})
  const wsRef = useRef(null)
  const [isConnected, setIsConnected] = useState(false)
  const [typingUsers, setTypingUsers] = useState(new Set())
  const messagesEndRef = useRef(null)
  const [showTeamMenu, setShowTeamMenu] = useState(false)
  const typingTimeoutsRef = useRef(new Map())
  const [showManageMembers, setShowManageMembers] = useState(false)
  const [showTeamSettings, setShowTeamSettings] = useState(false)
  const [hoveredMessageId, setHoveredMessageId] = useState(null)
  const [openMenuId, setOpenMenuId] = useState(null)
  const shouldReconnectRef = useRef(true)
  const [removalNotice, setRemovalNotice] = useState('')
  const { setServerDown } = useStatus()
  const [teamAvatarUrl, setTeamAvatarUrl] = useState('')
  const teamAvatarObjRef = useRef(null)

  const checkMembership = async () => {
    try {
      await api.get(`/teams/${teamId}`)
      setServerDown('')
      return true
    } catch (err) {
      if (!err.response) {
        setServerDown('Not connected to server.')
      }
      if (err.response?.status === 403 || err.response?.status === 404) {
        handleRemoved('You were removed from this team.')
        return false
      }
      return false
    }
  }

  useEffect(() => {
    shouldReconnectRef.current = true
    if (token) connectWebSocket()
    fetchTeamData()
    
    return () => {
      shouldReconnectRef.current = false
      if (wsRef.current) {
        try { wsRef.current.onopen = null; wsRef.current.onmessage = null; wsRef.current.onclose = null; wsRef.current.onerror = null } catch {}
        try { wsRef.current.close() } catch {}
        wsRef.current = null
        setIsConnected(false)
      }
      // Clear any pending typing timeouts
      typingTimeoutsRef.current.forEach((t) => clearTimeout(t))
      typingTimeoutsRef.current.clear()
    }
  }, [teamId, token])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  // Cleanup pending preview URL on unmount
  useEffect(() => {
    return () => {
      if (pendingPreview) URL.revokeObjectURL(pendingPreview)
    }
  }, [pendingPreview])

  // Fetch protected image blobs with auth header and cache object URLs
  useEffect(() => {
    const loadImages = async () => {
      for (const msg of messages) {
        const isImage = msg.type === 'image' || (typeof msg.content === 'string' && msg.content.startsWith('/api/uploads/'))
        if (!isImage) continue
        const key = msg.id || msg.content
        if (!key || imageUrlsRef.current[key]) continue
        try {
          // Normalize path to avoid double baseURL when msg.content already has /api prefix
          let requestPath = msg.content
          if (requestPath.startsWith('/api/')) {
            requestPath = requestPath.replace(/^\/api/, '')
          }
          const res = await api.get(requestPath, { responseType: 'blob' })
          const objectUrl = URL.createObjectURL(res.data)
          imageUrlsRef.current[key] = objectUrl
          setImageUrls(prev => ({ ...prev, [key]: objectUrl }))
        } catch (err) {
          console.error('Failed to load image', err)
        }
      }
    }
    loadImages()
  }, [messages])

  // Cleanup cached object URLs on unmount
  useEffect(() => {
    return () => {
      Object.values(imageUrlsRef.current).forEach(url => URL.revokeObjectURL(url))
    }
  }, [])

  // Load team avatar (protected endpoint) to use as img src
  useEffect(() => {
    let cancelled = false
    const loadAvatar = async () => {
      if (!team?.avatar) {
        setTeamAvatarUrl('')
        return
      }
      let path = team.avatar
      if (path.startsWith('/api/')) {
        path = path.replace(/^\/api/, '')
      }
      try {
        const res = await api.get(path, { responseType: 'blob' })
        const url = URL.createObjectURL(res.data)
        if (teamAvatarObjRef.current) {
          URL.revokeObjectURL(teamAvatarObjRef.current)
        }
        teamAvatarObjRef.current = url
        if (!cancelled) setTeamAvatarUrl(url)
      } catch (err) {
        console.error('Failed to load team avatar', err)
        setTeamAvatarUrl('')
      }
    }
    loadAvatar()
    return () => {
      cancelled = true
      if (teamAvatarObjRef.current) {
        URL.revokeObjectURL(teamAvatarObjRef.current)
        teamAvatarObjRef.current = null
      }
    }
  }, [team?.avatar])

  const handleRemoved = (reason) => {
    shouldReconnectRef.current = false
    if (wsRef.current) {
      try { wsRef.current.close() } catch {}
    }
    setRemovalNotice(reason)
  }

  const fetchTeamData = async () => {
    try {
      setLoading(true)
      setServerDown('')
      
      // Fetch team info
      const teamResponse = await api.get(`/teams/${teamId}`)
      setTeam(teamResponse.data.data)
      
      // Fetch messages
      const messagesResponse = await api.get(`/teams/${teamId}/messages?limit=100`)
      setMessages(messagesResponse.data.data || [])
      
    } catch (error) {
      console.error('Failed to fetch team data:', error)
      if (!error.response) {
        setServerDown('Not connected to server.')
      }
      if (error.response?.status === 403) {
        handleRemoved('You were removed from this team.')
      }
    } finally {
      setLoading(false)
    }
  }

  // If disconnected for more than a short window, re-verify membership to avoid lingering
  useEffect(() => {
    if (isConnected) return
    const timer = setTimeout(() => {
      checkMembership()
    }, 1500)
    return () => clearTimeout(timer)
  }, [isConnected, teamId])

  const connectWebSocket = () => {
    if (!shouldReconnectRef.current) return
    // Close existing connection if any
    if (wsRef.current) {
      try { wsRef.current.close() } catch {}
    }
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const wsUrl = `${protocol}://${window.location.hostname}:${window.location.port}/api/ws/${teamId}?token=${encodeURIComponent(token)}`
    const websocket = new WebSocket(wsUrl)
    
    websocket.onopen = () => {
      console.log('WebSocket connected')
      setIsConnected(true)
    }
    
    websocket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        handleWebSocketMessage(data)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }
    
    websocket.onclose = () => {
      console.log('WebSocket disconnected')
      setIsConnected(false)
      if (wsRef.current === websocket) {
        wsRef.current = null
      }
      
      // Check membership; if removed, stop reconnects and redirect
      checkMembership().then((stillMember) => {
        if (!stillMember) return
        // Try to reconnect after 3 seconds (if allowed)
        setTimeout(() => {
          if (!shouldReconnectRef.current) return
          if (wsRef.current === websocket || wsRef.current === null) {
            // only reconnect if not replaced by a newer socket
            connectWebSocket()
          }
        }, 300)
      })
    }
    
    websocket.onerror = (error) => {
      console.error('WebSocket error:', error)
      setServerDown('Not connected to server.')
      checkMembership()
      // Try a quick reconnect if still allowed
      setTimeout(() => {
        if (!shouldReconnectRef.current) return
        if (!wsRef.current) {
          connectWebSocket()
        }
      }, 300)
    }
    
    wsRef.current = websocket
  }

  const handleWebSocketMessage = (data) => {
    switch (data.type) {
      case 'chat_message': {
        const incoming = { ...data.payload, created_at: data.payload.created_at || data.payload.timestamp }
        setMessages(prev => {
          if (incoming.id && prev.some(m => m.id === incoming.id)) return prev
          return [...prev, incoming]
        })
        break
      }
      case 'message_deleted': {
        const id = data.payload?.id || data.payload?.message_id
        const deleter = data.payload?.deleted_by_name || 'User'
        if (!id) break
        setMessages(prev => prev.map(m => {
          if (m.id !== id) return m
          return {
            ...m,
            content: `${deleter} has deleted the message`,
            type: 'system',
          }
        }))
        break
      }
      case 'removed_from_team': {
        const reason = data.payload?.reason || 'You were removed from the team.'
        handleRemoved(reason)
        break
      }
      case 'user_joined':
        // Handle user joined notification
        break
      case 'user_left':
        // Handle user left notification
        break
      case 'typing':
        if (data.payload.user_id !== user.id) {
          const name = data.payload.username
          // Add to set
          setTypingUsers(prev => {
            const ns = new Set(prev)
            ns.add(name)
            return ns
          })
          // Reset the 3s timeout per user
          const existing = typingTimeoutsRef.current.get(name)
          if (existing) clearTimeout(existing)
          const timer = setTimeout(() => {
            setTypingUsers(prev => {
              const ns = new Set(prev)
              ns.delete(name)
              return ns
            })
            typingTimeoutsRef.current.delete(name)
          }, 3000)
          typingTimeoutsRef.current.set(name, timer)
        }
        break
      default:
        console.log('Unknown message type:', data.type)
    }
  }

  const sendMessage = async () => {
    if (sending) return
    if (!pendingImage && !newMessage.trim()) return

    setSending(true)

    try {
      // Send pending image first (if any)
      if (pendingImage) {
        await uploadImageAndSend(pendingImage)
        setPendingImage(null)
        setPendingPreview(null)
      }

      // Send text message if provided
      if (newMessage.trim()) {
        if (wsRef.current && isConnected) {
          wsRef.current.send(JSON.stringify({
            type: 'chat_message',
            payload: {
              content: newMessage.trim(),
              type: 'text'
            }
          }))
        } else {
          const response = await api.post(`/teams/${teamId}/messages`, {
            content: newMessage.trim(),
            type: 'text'
          })
          setMessages(prev => [...prev, response.data.data])
        }
        setNewMessage('')
      }
    } catch (error) {
      console.error('Failed to send message:', error)
    } finally {
      setSending(false)
    }
  }

  const uploadImageAndSend = async (file) => {
    try {
      setUploading(true)
      const form = new FormData()
      form.append('file', file)
      const res = await api.post(`/upload?team_id=${teamId}`, form, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      const url = res.data.data?.url
      if (!url) return
      // Seed local image cache with preview if available
      if (pendingPreview) {
        imageUrlsRef.current[url] = pendingPreview
        setImageUrls(prev => ({ ...prev, [url]: pendingPreview }))
        setPendingPreview(null)
      }
      // Prefer WS if connected
      if (wsRef.current && isConnected) {
        wsRef.current.send(JSON.stringify({
          type: 'chat_message',
          payload: { content: url, type: 'image' }
        }))
      } else {
        await api.post(`/teams/${teamId}/messages`, { content: url, type: 'image' })
      }
    } catch (e) {
      console.error('Upload failed', e)
    } finally {
      setUploading(false)
    }
  }

  const handlePaste = async (e) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      const it = items[i]
      if (it.kind === 'file' && it.type.startsWith('image/')) {
        const file = it.getAsFile()
        if (file) {
          e.preventDefault()
          if (pendingPreview) URL.revokeObjectURL(pendingPreview)
          setPendingImage(file)
          setPendingPreview(URL.createObjectURL(file))
          break
        }
      }
    }
  }

  const deleteMessage = async (messageId) => {
    try {
      await api.delete(`/teams/${teamId}/messages/${messageId}`)
      setMessages(prev => prev.map(m => {
        if (m.id !== messageId) return m
        return {
          ...m,
          content: `${user.name || user.username} has deleted the message`,
          type: 'system',
        }
      }))
    } catch (err) {
      console.error('Failed to delete message', err)
    }
  }

  const handleTyping = () => {
    if (wsRef.current && isConnected) {
      wsRef.current.send(JSON.stringify({
        type: 'typing',
        payload: {}
      }))
    }
  }

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const formatTime = (timestamp) => {
    const date = new Date(timestamp)
    if (isNaN(date.getTime())) return ''
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  const leaveTeam = async () => {
    try {
      await api.post(`/teams/${teamId}/leave`)
      navigate('/dashboard')
    } catch (error) {
      console.error('Failed to leave team:', error)
    }
  }

  const saveTeamSettings = async (payload) => {
    try {
      const res = await api.put(`/teams/${teamId}`, payload)
      setTeam(res.data.data)
      setShowTeamSettings(false)
    } catch (error) {
      console.error('Failed to update team:', error)
    }
  }

  const deleteTeam = async () => {
    if (!window.confirm('Are you sure you want to delete this team? This cannot be undone.')) return
    try {
      await api.delete(`/teams/${teamId}`)
      navigate('/dashboard')
    } catch (error) {
      console.error('Failed to delete team:', error)
    }
  }

  const uploadTeamAvatar = async (file) => {
    const formData = new FormData()
    formData.append('file', file)
    const res = await api.post(`/upload?team_id=${teamId}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return res.data?.data?.url
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  if (!team) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Team not found</h2>
        <p className="text-gray-500 mb-4">The team you're looking for doesn't exist or you don't have access.</p>
        <button
          onClick={() => navigate('/dashboard')}
          className="btn-primary"
        >
          Back to Dashboard
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 min-h-0 h-full dark:bg-gray-900 overflow-hidden">
      {removalNotice && (
        <div className="fixed inset-0 bg-gray-900 bg-opacity-60 z-50 flex items-center justify-center">
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl border border-gray-200 dark:border-gray-700 max-w-sm w-full mx-4 p-6 text-center">
            <div className="mx-auto mb-4 w-12 h-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center text-xl font-bold">
              !
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">Removed from team</h3>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">{removalNotice}</p>
            <button
              onClick={() => navigate('/dashboard')}
              className="w-full btn-primary py-2"
            >
              OK
            </button>
          </div>
        </div>
      )}
      {/* Team header */}
      <div className="bg-white border-b border-gray-200 dark:bg-gray-800 dark:border-gray-700 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-gray-400 to-gray-500 flex items-center justify-center overflow-hidden">
              {teamAvatarUrl ? (
                <img src={teamAvatarUrl} alt={team.name} className="w-full h-full object-cover" />
              ) : (
                <Users className="w-5 h-5 text-white" />
              )}
            </div>
            <div>
              <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{team.name}</h1>
              {team.description && (
                <p className="text-sm text-gray-500 dark:text-gray-300">{team.description}</p>
              )}
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            {/* Connection status */}
            <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
            <span className="text-xs text-gray-500 dark:text-gray-300">
              {isConnected ? 'Connected' : 'Disconnected'}
            </span>
            
            {/* Team menu */}
            <div className="relative">
              <button
                onClick={() => setShowTeamMenu(!showTeamMenu)}
                className="p-2 text-gray-400 hover:text-gray-600 dark:text-gray-300 dark:hover:text-white rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              >
                <MoreVertical className="w-5 h-5" />
              </button>
              
              {showTeamMenu && (
                <div className="absolute right-0 top-full mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-2 z-10">
                  {team && user?.id === team.owner_id && (
                    <button
                      onClick={() => { setShowTeamMenu(false); setShowManageMembers(true) }}
                      className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center space-x-2"
                    >
                      <UserPlus className="w-4 h-4" />
                      <span>Invite members</span>
                    </button>
                  )}
                  <button
                    onClick={() => { setShowTeamMenu(false); setShowTeamSettings(true) }}
                    className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center space-x-2"
                  >
                    <Settings className="w-4 h-4" />
                    <span>Team settings</span>
                  </button>
                  <hr className="my-2" />
                  <button
                    onClick={leaveTeam}
                    className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/40 flex items-center space-x-2"
                  >
                    <LogOut className="w-4 h-4" />
                    <span>Leave team</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4 pb-3 custom-scrollbar">
        {messages.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <Users className="w-8 h-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No messages yet</h3>
            <p className="text-gray-500 dark:text-gray-300">Start the conversation by sending a message!</p>
          </div>
        ) : (
          messages.map((message) => (
            <div
              key={message.id}
              onMouseEnter={() => setHoveredMessageId(message.id)}
              onMouseLeave={() => {
                if (openMenuId !== message.id) {
                  setHoveredMessageId(null)
                }
              }}
              className={`flex items-start space-x-3 ${
                message.user_id === user.id ? 'flex-row-reverse space-x-reverse' : ''
              }`}
            >
              <div className={`relative max-w-xs lg:max-w-md ${message.user_id === user.id ? 'text-right' : ''}`}>
                {(() => {
                  const isImage = message.type === 'image' || (typeof message.content === 'string' && message.content.startsWith('/api/uploads/'))
                  const key = message.id || message.content
                  if (isImage) {
                    const src = key ? imageUrls[key] : null
                    return (
                      <div className={`message-bubble ${message.user_id === user.id ? 'own' : 'other'} p-1`}>
                        {src ? (
                          <a href={src} target="_blank" rel="noreferrer">
                            <img src={src} alt="uploaded" className="rounded-lg max-w-full h-auto" loading="lazy" />
                          </a>
                        ) : (
                          <div className="text-xs text-gray-500 px-2 py-1">Loading image...</div>
                        )}
                      </div>
                    )
                  }
                  return (
                  <div className={`message-bubble ${
                    message.user_id === user.id ? 'own' : 'other'
                  }`}>
                    <p className="text-sm whitespace-pre-wrap break-words">{message.content}</p>
                  </div>
                  )
                })()}
                <div className={`mt-1 text-xs text-gray-500 dark:text-gray-300 ${message.user_id === user.id ? 'text-right' : ''}`}>
                  <span>{message.name || message.username}</span>
                  <span className="mx-2">&middot;</span>
                  <span>{formatTime(message.created_at || message.timestamp)}</span>
                  {message.user_id === user.id && (
                    <>
                      <span className="mx-2">&middot;</span>
                      <button
                        onClick={() => deleteMessage(message.id)}
                        className="text-red-500 hover:text-red-700"
                      >
                        Delete
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
        
        {/* Typing indicators */}
        {typingUsers.size > 0 && (
          <div className="flex items-center space-x-2 text-sm text-gray-500 italic">
            <div className="flex space-x-1">
              <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
              <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
              <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
            </div>
            <span>{Array.from(typingUsers).join(', ')} typing...</span>
          </div>
        )}
        
        <div ref={messagesEndRef} />
      </div>

      {/* Message input */}
      <div className="shrink-0 bg-white border-t border-gray-200 dark:bg-gray-800 dark:border-gray-700 p-4 pt-3 pb-3">
        <div className="max-w-5xl mx-auto w-full space-y-3">
          {pendingPreview && (
            <div className="flex items-start space-x-3 rounded-lg border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900">
              <div className="w-24 h-24 overflow-hidden rounded-md bg-white dark:bg-gray-800 border dark:border-gray-700">
                <img src={pendingPreview} alt="preview" className="w-full h-full object-cover" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-gray-700 dark:text-gray-100">Image ready to send</p>
                <p className="text-xs text-gray-500 dark:text-gray-300">Press Send to upload with your next message.</p>
              </div>
              <button
                onClick={() => {
                  if (pendingPreview) URL.revokeObjectURL(pendingPreview)
                  setPendingPreview(null)
                  setPendingImage(null)
                }}
                className="text-gray-500 dark:text-gray-300 hover:text-red-500 text-sm"
              >
                Clear
              </button>
            </div>
          )}

          <div className="flex space-x-3 items-center">
            <input
              type="text"
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              onPaste={handlePaste}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  sendMessage()
                }
              }}
              onInput={handleTyping}
              placeholder="Type a message..."
              className="flex-1 input h-12"
              disabled={!isConnected}
            />
            <button
              onClick={sendMessage}
              disabled={( !newMessage.trim() && !pendingImage) || sending || uploading || !isConnected}
              className="btn-primary px-6 disabled:opacity-50 disabled:cursor-not-allowed h-12"
            >
              {sending || uploading ? (
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
              ) : (
                <Send className="w-5 h-5" />
              )}
            </button>
          </div>
          
          {!isConnected && (
            <p className="text-xs text-red-600">
              Connection lost. Trying to reconnect...
            </p>
          )}
        </div>
      </div>
      {/* Manage Members Modal */}
      {team && user?.id === team.owner_id && (
        <ManageMembersModal
          isOpen={showManageMembers}
          onClose={() => setShowManageMembers(false)}
          teamId={teamId}
          ownerId={team.owner_id}
        />
      )}
      {team && (
        <TeamSettingsModal
          isOpen={showTeamSettings}
          onClose={() => setShowTeamSettings(false)}
          team={team}
          onSave={saveTeamSettings}
          onDelete={deleteTeam}
          onLeave={() => { setShowTeamSettings(false); leaveTeam() }}
          isOwner={user?.id === team.owner_id}
          onUploadAvatar={uploadTeamAvatar}
        />
      )}
    </div>
  )
}

export default Chat


