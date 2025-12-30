import React, { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
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
  const [hoveredMessageId, setHoveredMessageId] = useState(null)
  const [openMenuId, setOpenMenuId] = useState(null)

  useEffect(() => {
    fetchTeamData()
    if (token) connectWebSocket()
    
    return () => {
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

  const fetchTeamData = async () => {
    try {
      setLoading(true)
      
      // Fetch team info
      const teamResponse = await api.get(`/teams/${teamId}`)
      setTeam(teamResponse.data.data)
      
      // Fetch messages
      const messagesResponse = await api.get(`/teams/${teamId}/messages?limit=100`)
      setMessages(messagesResponse.data.data || [])
      
    } catch (error) {
      console.error('Failed to fetch team data:', error)
      if (error.response?.status === 403) {
        navigate('/dashboard')
      }
    } finally {
      setLoading(false)
    }
  }

  const connectWebSocket = () => {
    // Close existing connection if any
    if (wsRef.current) {
      try { wsRef.current.close() } catch {}
    }
    const wsUrl = `ws://${window.location.hostname}:${window.location.port}/api/ws/${teamId}?token=${encodeURIComponent(token)}`
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
      
      // Try to reconnect after 3 seconds
      setTimeout(() => {
        if (wsRef.current === websocket || wsRef.current === null) {
          // only reconnect if not replaced by a newer socket
          connectWebSocket()
        }
      }, 1200)
    }
    
    websocket.onerror = (error) => {
      console.error('WebSocket error:', error)
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
    <div className="h-[calc(100vh-8rem)] flex flex-col">
      {/* Team header */}
      <div className="bg-white border-b border-gray-200 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-br from-gray-400 to-gray-500 rounded-lg flex items-center justify-center">
              <Users className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-semibold text-gray-900">{team.name}</h1>
              {team.description && (
                <p className="text-sm text-gray-500">{team.description}</p>
              )}
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            {/* Connection status */}
            <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
            <span className="text-xs text-gray-500">
              {isConnected ? 'Connected' : 'Disconnected'}
            </span>
            
            {/* Team menu */}
            <div className="relative">
              <button
                onClick={() => setShowTeamMenu(!showTeamMenu)}
                className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <MoreVertical className="w-5 h-5" />
              </button>
              
              {showTeamMenu && (
                <div className="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-2 z-10">
                  {team && user?.id === team.owner_id && (
                    <button
                      onClick={() => { setShowTeamMenu(false); setShowManageMembers(true) }}
                      className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center space-x-2"
                    >
                      <UserPlus className="w-4 h-4" />
                      <span>Invite members</span>
                    </button>
                  )}
                  <button
                    onClick={() => setShowTeamMenu(false)}
                    className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-50 flex items-center space-x-2"
                  >
                    <Settings className="w-4 h-4" />
                    <span>Team settings</span>
                  </button>
                  <hr className="my-2" />
                  <button
                    onClick={leaveTeam}
                    className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 flex items-center space-x-2"
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
      <div className="flex-1 overflow-y-auto p-4 space-y-4 scrollbar-hide">
        {messages.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <Users className="w-8 h-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">No messages yet</h3>
            <p className="text-gray-500">Start the conversation by sending a message!</p>
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
              <div className={`relative max-w-xs lg:max-w-md ${message.user_id === user.id ? 'text-right' : ''} ${message.user_id === user.id ? 'pl-6' : ''}`}>
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
                <div className={`mt-1 text-xs text-gray-500 ${
                  message.user_id === user.id ? 'text-right' : ''
                }`}>
                  <span>{message.name || message.username}</span>
                  <span className="mx-2">•</span>
                  <span>{formatTime(message.created_at || message.timestamp)}</span>
                </div>
                {message.user_id === user.id && hoveredMessageId === message.id && (
                  <div
                    className="absolute"
                    style={{ left: '-16px', top: '8px' }}
                  >
                    <button
                      onClick={() => setOpenMenuId(openMenuId === message.id ? null : message.id)}
                      className="p-1 text-gray-400 hover:text-gray-600"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.75a.75.75 0 110-1.5.75.75 0 010 1.5zM12 12.75a.75.75 0 110-1.5.75.75 0 010 1.5zM12 18.75a.75.75 0 110-1.5.75.75 0 010 1.5z" />
                      </svg>
                    </button>
                    {openMenuId === message.id && (
                      <div className="absolute left-0 mt-1 w-28 bg-white border border-gray-200 rounded shadow-md text-left text-sm z-10">
                        {message.type !== 'image' && (
                          <button className="w-full text-left px-3 py-2 text-gray-700 hover:bg-gray-100" disabled>
                            Edit (coming soon)
                          </button>
                        )}
                        <button
                          onClick={() => {
                            deleteMessage(message.id)
                            setOpenMenuId(null)
                          }}
                          className="w-full text-left px-3 py-2 text-red-600 hover:bg-red-50"
                        >
                          Delete
                        </button>
                      </div>
                    )}
                  </div>
                )}
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
      <div className="bg-white border-t border-gray-200 p-4 sticky bottom-0">
        <div className="max-w-5xl mx-auto w-full space-y-3">
          {pendingPreview && (
            <div className="flex items-start space-x-3 rounded-lg border border-gray-200 p-3 bg-gray-50">
              <div className="w-24 h-24 overflow-hidden rounded-md bg-white border">
                <img src={pendingPreview} alt="preview" className="w-full h-full object-cover" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-gray-700">Image ready to send</p>
                <p className="text-xs text-gray-500">Press Send to upload with your next message.</p>
              </div>
              <button
                onClick={() => {
                  if (pendingPreview) URL.revokeObjectURL(pendingPreview)
                  setPendingPreview(null)
                  setPendingImage(null)
                }}
                className="text-gray-500 hover:text-red-500 text-sm"
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
    </div>
  )
}

export default Chat
