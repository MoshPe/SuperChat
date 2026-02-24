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
  Settings,
  Paperclip,
  File,
  FileText,
  FileArchive,
  FileCode,
  FileSpreadsheet,
  FileAudio,
  FileVideo,
  Shield
} from 'lucide-react'
import api from '../services/api'
import ManageMembersModal from '../components/ManageMembersModal'
import TeamSettingsModal from '../components/TeamSettingsModal'

const SCREEN_VIEWER_MAX_QUEUED_CHUNKS = 48
const SCREEN_VIEWER_RECONNECT_DELAYS_MS = [400, 800, 1200, 1600]

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
  const [pendingAttachments, setPendingAttachments] = useState([])
  const pendingAttachmentsRef = useRef([])
  const [imageUrls, setImageUrls] = useState({})
  const imageUrlsRef = useRef({})
  const [attachmentMeta, setAttachmentMeta] = useState({})
  const attachmentMetaRef = useRef({})
  const fileInputRef = useRef(null)
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
  const [uploadError, setUploadError] = useState('')
  const [isDragOver, setIsDragOver] = useState(false)
  const [activeScreenShare, setActiveScreenShare] = useState(null)
  const [screenShareDeniedInfo, setScreenShareDeniedInfo] = useState(null)
  const [screenShareError, setScreenShareError] = useState('')
  const [screenSharePendingStart, setScreenSharePendingStart] = useState(false)
  const [isScreenSharePublisher, setIsScreenSharePublisher] = useState(false)
  const activeScreenShareRef = useRef(null)
  const screenPublishWsRef = useRef(null)
  const screenMediaStreamRef = useRef(null)
  const screenMediaRecorderRef = useRef(null)
  const screenPublisherStartingRef = useRef(false)
  const screenPublisherStoppingRef = useRef(false)
  const screenViewerVideoRef = useRef(null)
  const screenViewerWsRef = useRef(null)
  const screenViewerSessionRef = useRef('')
  const screenViewerStoppingRef = useRef(false)
  const screenViewerReconnectTimerRef = useRef(null)
  const screenViewerReconnectAttemptRef = useRef(0)
  const screenViewerMediaSourceRef = useRef(null)
  const screenViewerSourceBufferRef = useRef(null)
  const screenViewerSourceOpenHandlerRef = useRef(null)
  const screenViewerUpdateEndHandlerRef = useRef(null)
  const screenViewerObjectUrlRef = useRef('')
  const screenViewerChunkQueueRef = useRef([])
  const screenViewerPendingInitRef = useRef(null)
  const screenViewerTrimInProgressRef = useRef(false)
  const [screenViewerConnected, setScreenViewerConnected] = useState(false)
  const [screenViewerReady, setScreenViewerReady] = useState(false)
  const [screenViewerError, setScreenViewerError] = useState('')
  const [screenViewerMimeType, setScreenViewerMimeType] = useState('')

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
      setActiveScreenShare(null)
      setScreenShareDeniedInfo(null)
      setScreenShareError('')
      setScreenSharePendingStart(false)
      setIsScreenSharePublisher(false)
      stopLocalScreenSharePublisher({ notifyServer: false, clearLocalState: true })
      stopLocalScreenShareViewer({ clearState: true, preserveError: false })
    }
  }, [teamId, token])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    pendingAttachmentsRef.current = pendingAttachments
  }, [pendingAttachments])

  useEffect(() => {
    activeScreenShareRef.current = activeScreenShare
  }, [activeScreenShare])

  const clearScreenViewerReconnectTimer = ({ resetAttempts = false } = {}) => {
    if (screenViewerReconnectTimerRef.current) {
      clearTimeout(screenViewerReconnectTimerRef.current)
      screenViewerReconnectTimerRef.current = null
    }
    if (resetAttempts) {
      screenViewerReconnectAttemptRef.current = 0
    }
  }

  // Cleanup pending preview URLs on unmount
  useEffect(() => {
    return () => {
      pendingAttachmentsRef.current.forEach(item => {
        if (item.previewUrl) URL.revokeObjectURL(item.previewUrl)
      })
    }
  }, [])

  // Fetch protected image blobs with auth header and cache object URLs
  useEffect(() => {
    const parseFilename = (headers) => {
      const cd = headers?.['content-disposition'] || ''
      const utf8Match = cd.match(/filename\*=UTF-8''([^;]+)/i)
      if (utf8Match?.[1]) {
        try { return decodeURIComponent(utf8Match[1]) } catch {}
      }
      const plainMatch = cd.match(/filename=\"?([^\";]+)\"?/i)
      if (plainMatch?.[1]) return plainMatch[1]
      const fromHeader = headers?.['x-upload-filename']
      return fromHeader || ''
    }

    const loadImages = async () => {
      for (const msg of messages) {
        const isUpload = typeof msg.content === 'string' && msg.content.startsWith('/api/uploads/')
        if (!isUpload) continue
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
          const meta = {
            filename: parseFilename(res.headers),
            contentType: res.data?.type || res.headers?.['content-type'] || '',
            size: Number(res.headers?.['content-length']) || res.data?.size || 0,
          }
          attachmentMetaRef.current[key] = meta
          setAttachmentMeta(prev => ({ ...prev, [key]: meta }))
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

  const isImageFile = (file) => !!file && typeof file.type === 'string' && file.type.startsWith('image/')

  const attachmentVisual = ({ filename = '', contentType = '' } = {}) => {
    const name = String(filename || '').toLowerCase()
    const type = String(contentType || '').toLowerCase()
    const ext = name.includes('.') ? name.split('.').pop() : ''

    if (type.startsWith('audio/') || ['mp3', 'wav', 'ogg', 'm4a', 'flac'].includes(ext)) {
      return { Icon: FileAudio, label: 'Audio' }
    }
    if (type.startsWith('video/') || ['mp4', 'mov', 'mkv', 'avi', 'webm'].includes(ext)) {
      return { Icon: FileVideo, label: 'Video' }
    }
    if (type === 'application/pdf' || ext === 'pdf') {
      return { Icon: FileText, label: 'PDF' }
    }
    if (
      type.includes('zip') ||
      type.includes('compressed') ||
      ['zip', 'rar', '7z', 'tar', 'gz', 'tgz'].includes(ext)
    ) {
      return { Icon: FileArchive, label: 'Archive' }
    }
    if (
      type.includes('json') || type.includes('xml') || type.includes('javascript') || type.includes('typescript') ||
      type.includes('x-shellscript') || type.startsWith('text/') ||
      ['js', 'ts', 'tsx', 'jsx', 'json', 'xml', 'yml', 'yaml', 'html', 'css', 'md', 'txt', 'go', 'py', 'java', 'c', 'cpp', 'rs', 'sh'].includes(ext)
    ) {
      return { Icon: FileCode, label: 'Code/Text' }
    }
    if (
      type.includes('spreadsheet') || type.includes('excel') ||
      ['xls', 'xlsx', 'csv'].includes(ext)
    ) {
      return { Icon: FileSpreadsheet, label: 'Spreadsheet' }
    }
    if (
      type.includes('word') || type.includes('document') || type.includes('presentation') ||
      ['doc', 'docx', 'ppt', 'pptx', 'odt'].includes(ext)
    ) {
      return { Icon: FileText, label: 'Document' }
    }
    return { Icon: File, label: 'File' }
  }

  const formatFileSize = (bytes) => {
    const n = Number(bytes)
    if (!Number.isFinite(n) || n <= 0) return ''
    if (n < 1024) return `${n} B`
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
    return `${(n / (1024 * 1024)).toFixed(1)} MB`
  }

  const buildPendingAttachment = (file) => ({
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    file,
    previewUrl: isImageFile(file) ? URL.createObjectURL(file) : '',
    progress: null,
    error: '',
  })

  const appendPendingFiles = (filesLike) => {
    const files = Array.from(filesLike || []).filter(Boolean)
    if (!files.length) return
    setUploadError('')
    setPendingAttachments(prev => [...prev, ...files.map(buildPendingAttachment)])
  }

  const removePendingAttachment = (attachmentId, { revokePreview = true } = {}) => {
    setPendingAttachments(prev => {
      const next = []
      for (const item of prev) {
        if (item.id !== attachmentId) {
          next.push(item)
          continue
        }
        if (revokePreview && item.previewUrl) URL.revokeObjectURL(item.previewUrl)
      }
      return next
    })
  }

  const clearPendingAttachments = () => {
    setPendingAttachments(prev => {
      prev.forEach(item => {
        if (item.previewUrl) URL.revokeObjectURL(item.previewUrl)
      })
      return []
    })
  }

  const updatePendingAttachment = (attachmentId, patch) => {
    setPendingAttachments(prev => prev.map(item => (
      item.id === attachmentId ? { ...item, ...patch } : item
    )))
  }

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
    const wsUrl = `${protocol}://${window.location.host}/api/ws/${teamId}?token=${encodeURIComponent(token)}`
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
          const name = data.payload.name || data.payload.username
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
          }, 500)
          typingTimeoutsRef.current.set(name, timer)
        }
        break
      case 'user_profile_updated': {
        const updatedUserId = data.payload?.user_id
        if (!updatedUserId) break
        const nextName = data.payload?.name || data.payload?.username
        if (!nextName) break
        setMessages(prev => prev.map(message => {
          if (message.user_id !== updatedUserId) return message
          return {
            ...message,
            name: nextName,
            username: nextName,
          }
        }))
        break
      }
      case 'screen_share_start_granted': {
        const sessionId = data.payload?.session_id
        const sharerUserId = data.payload?.sharer_user_id || user?.id
        const sharerName = data.payload?.sharer_name || user?.name || user?.username || 'User'
        setScreenSharePendingStart(false)
        setScreenShareDeniedInfo(null)
        setScreenShareError('')
        if (!sessionId) {
          setScreenShareError('Screen share start was granted, but no session ID was provided.')
          break
        }
        setIsScreenSharePublisher(true)
        setActiveScreenShare({
          sessionId,
          sharerUserId,
          sharerName,
          status: 'granted',
        })
        break
      }
      case 'screen_share_denied': {
        setScreenSharePendingStart(false)
        setIsScreenSharePublisher(false)
        setScreenShareDeniedInfo({
          sharerUserId: data.payload?.sharer_user_id || '',
          sharerName: data.payload?.sharer_name || 'Another member',
          reason: data.payload?.reason || 'already_active',
          activeSession: data.payload?.active_session || '',
        })
        if (data.payload?.sharer_user_id !== activeScreenShare?.sharerUserId) {
          setActiveScreenShare((prev) => prev && prev.status === 'started' ? prev : null)
        }
        break
      }
      case 'screen_share_started': {
        const sessionId = data.payload?.session_id
        const sharerUserId = data.payload?.sharer_user_id
        const sharerName = data.payload?.sharer_name || 'Member'
        if (!sessionId || !sharerUserId) break
        setScreenSharePendingStart(false)
        setScreenShareDeniedInfo(null)
        setScreenShareError('')
        setActiveScreenShare({
          sessionId,
          sharerUserId,
          sharerName,
          status: 'started',
        })
        setIsScreenSharePublisher(sharerUserId === user?.id)
        break
      }
      case 'screen_share_stopped': {
        const stoppedSessionId = data.payload?.session_id
        const reason = data.payload?.reason
        const shouldStopLocalPublisher =
          isScreenSharePublisher &&
          (!stoppedSessionId || stoppedSessionId === activeScreenShareRef.current?.sessionId)
        const shouldStopLocalViewer =
          !!screenViewerWsRef.current &&
          (!stoppedSessionId || stoppedSessionId === activeScreenShareRef.current?.sessionId)
        if (shouldStopLocalPublisher) {
          stopLocalScreenSharePublisher({ notifyServer: false, clearLocalState: false })
        }
        if (shouldStopLocalViewer) {
          stopLocalScreenShareViewer({ clearState: true, preserveError: false })
        }
        setScreenSharePendingStart(false)
        setIsScreenSharePublisher(false)
        setActiveScreenShare((prev) => {
          if (!prev) return null
          if (stoppedSessionId && prev.sessionId && prev.sessionId !== stoppedSessionId) return prev
          return null
        })
        if (reason && reason !== 'stopped_by_sharer') {
          setScreenShareError('Screen sharing ended.')
        }
        break
      }
      case 'screen_share_error': {
        setScreenSharePendingStart(false)
        const msg = data.payload?.message || 'Screen sharing error'
        setScreenShareError(msg)
        break
      }
      default:
        console.log('Unknown message type:', data.type)
    }
  }

  const pickScreenShareMimeType = () => {
    if (typeof window === 'undefined' || typeof window.MediaRecorder === 'undefined') return ''
    const candidates = [
      'video/webm;codecs=vp9',
      'video/webm;codecs=vp8',
      'video/webm',
    ]
    for (const mime of candidates) {
      try {
        if (window.MediaRecorder.isTypeSupported?.(mime)) return mime
      } catch {}
    }
    return ''
  }

  const buildScreenStreamWsUrl = (sessionId, role) => {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    return `${protocol}://${window.location.host}/api/ws/${teamId}/screen?token=${encodeURIComponent(token)}&role=${encodeURIComponent(role)}&session_id=${encodeURIComponent(sessionId)}`
  }

  const stopLocalScreenSharePublisher = ({ notifyServer = false, clearLocalState = false } = {}) => {
    if (screenPublisherStoppingRef.current) return
    screenPublisherStoppingRef.current = true

    const recorder = screenMediaRecorderRef.current
    const stream = screenMediaStreamRef.current
    const publishWs = screenPublishWsRef.current

    screenMediaRecorderRef.current = null
    screenMediaStreamRef.current = null
    screenPublishWsRef.current = null
    screenPublisherStartingRef.current = false

    try {
      if (recorder && recorder.state !== 'inactive') recorder.stop()
    } catch {}
    try {
      if (stream) {
        stream.getTracks().forEach((track) => {
          try { track.onended = null } catch {}
          try { track.stop() } catch {}
        })
      }
    } catch {}
    try {
      if (publishWs) {
        try { publishWs.onopen = null; publishWs.onmessage = null; publishWs.onclose = null; publishWs.onerror = null } catch {}
        if (publishWs.readyState === WebSocket.OPEN || publishWs.readyState === WebSocket.CONNECTING) {
          publishWs.close()
        }
      }
    } catch {}

    if (notifyServer && wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      try {
        wsRef.current.send(JSON.stringify({
          type: 'screen_share_stop',
          payload: {},
        }))
      } catch {}
    }

    if (clearLocalState) {
      setScreenSharePendingStart(false)
      setIsScreenSharePublisher(false)
      setActiveScreenShare(null)
    }

    // Release stop guard on next macrotask so nested callbacks don't immediately re-enter.
    setTimeout(() => { screenPublisherStoppingRef.current = false }, 0)
  }

  const startScreenSharePublisher = async (share) => {
    if (!share?.sessionId || !token) return
    if (screenPublisherStartingRef.current || screenMediaRecorderRef.current || screenPublishWsRef.current) return

    if (!navigator.mediaDevices?.getDisplayMedia) {
      setScreenSharePendingStart(false)
      setScreenShareError('Screen capture is not supported in this browser.')
      setIsScreenSharePublisher(false)
      return
    }

    if (typeof window === 'undefined' || typeof window.MediaRecorder === 'undefined') {
      setScreenSharePendingStart(false)
      setScreenShareError('MediaRecorder is not available in this browser.')
      setIsScreenSharePublisher(false)
      return
    }

    screenPublisherStartingRef.current = true
    setScreenShareError('')
    setActiveScreenShare((prev) => {
      if (!prev || prev.sessionId !== share.sessionId) return prev
      return { ...prev, status: 'starting' }
    })

    let stream = null
    let publishWs = null
    let recorder = null

    try {
      stream = await navigator.mediaDevices.getDisplayMedia({
        video: {
          frameRate: { ideal: 30, max: 30 },
        },
        audio: false,
      })

      const currentShare = activeScreenShareRef.current
      if (!currentShare || currentShare.sessionId !== share.sessionId) {
        stream.getTracks().forEach((track) => { try { track.stop() } catch {} })
        return
      }

      screenMediaStreamRef.current = stream
      const videoTrack = stream.getVideoTracks?.()[0]
      if (videoTrack) {
        videoTrack.onended = () => {
          stopLocalScreenSharePublisher({ notifyServer: true, clearLocalState: false })
        }
      }

      const preferredMimeType = pickScreenShareMimeType()
      const recorderOptions = {
        videoBitsPerSecond: 4_000_000,
        ...(preferredMimeType ? { mimeType: preferredMimeType } : {}),
      }
      recorder = new window.MediaRecorder(stream, recorderOptions)
      screenMediaRecorderRef.current = recorder

      const wsUrl = buildScreenStreamWsUrl(share.sessionId, 'publisher')
      publishWs = new WebSocket(wsUrl)
      publishWs.binaryType = 'arraybuffer'
      screenPublishWsRef.current = publishWs

      await new Promise((resolve, reject) => {
        let settled = false
        publishWs.onopen = () => {
          if (settled) return
          settled = true
          resolve()
        }
        publishWs.onerror = () => {
          if (settled) return
          settled = true
          reject(new Error('Failed to connect screen stream WebSocket'))
        }
        publishWs.onclose = () => {
          if (!settled) {
            settled = true
            reject(new Error('Screen stream WebSocket closed before opening'))
            return
          }
          if (!screenPublisherStoppingRef.current) {
            setScreenShareError('Screen streaming connection ended.')
            stopLocalScreenSharePublisher({ notifyServer: false, clearLocalState: false })
          }
        }
      })

      publishWs.onclose = () => {
        if (!screenPublisherStoppingRef.current) {
          setScreenShareError('Screen streaming connection ended.')
          stopLocalScreenSharePublisher({ notifyServer: false, clearLocalState: false })
        }
      }
      publishWs.onerror = () => {
        if (!screenPublisherStoppingRef.current) {
          setScreenShareError('Screen streaming connection error.')
        }
      }

      const settings = videoTrack?.getSettings?.() || {}
      publishWs.send(JSON.stringify({
        type: 'init',
        mime_type: recorder.mimeType || preferredMimeType || 'video/webm',
        width: Number(settings.width) || undefined,
        height: Number(settings.height) || undefined,
        fps_target: 30,
      }))

      recorder.ondataavailable = (event) => {
        if (!event?.data || event.data.size <= 0) return
        if (!screenPublishWsRef.current || screenPublishWsRef.current.readyState !== WebSocket.OPEN) return
        event.data.arrayBuffer()
          .then((buf) => {
            if (!screenPublishWsRef.current || screenPublishWsRef.current.readyState !== WebSocket.OPEN) return
            screenPublishWsRef.current.send(buf)
          })
          .catch((err) => {
            console.error('Failed to read screen share chunk', err)
          })
      }
      recorder.onerror = () => {
        setScreenShareError('Screen recorder error.')
      }
      recorder.start(150)

      setScreenSharePendingStart(false)
      setActiveScreenShare((prev) => {
        if (!prev || prev.sessionId !== share.sessionId) return prev
        return { ...prev, status: 'started' }
      })
    } catch (err) {
      console.error('Failed to start screen sharing publisher', err)
      const msg = err?.message || 'Failed to start screen sharing'
      setScreenSharePendingStart(false)
      setIsScreenSharePublisher(false)
      setScreenShareError(msg)
      stopLocalScreenSharePublisher({ notifyServer: true, clearLocalState: true })
    } finally {
      screenPublisherStartingRef.current = false
    }
  }

  useEffect(() => {
    if (!isScreenSharePublisher) return
    if (!activeScreenShare?.sessionId) return
    if (activeScreenShare.status !== 'granted') return
    void startScreenSharePublisher(activeScreenShare)
  }, [activeScreenShare, isScreenSharePublisher])

  const stopLocalScreenShareViewer = ({ clearState = false, preserveError = true, keepSocket = false } = {}) => {
    if (screenViewerStoppingRef.current) return
    screenViewerStoppingRef.current = true
    if (!keepSocket) {
      clearScreenViewerReconnectTimer({ resetAttempts: !!clearState })
    }

    if (!keepSocket) {
      const viewerWs = screenViewerWsRef.current
      screenViewerWsRef.current = null
      screenViewerSessionRef.current = ''

      try {
        if (viewerWs) {
          try { viewerWs.onopen = null; viewerWs.onmessage = null; viewerWs.onclose = null; viewerWs.onerror = null } catch {}
          if (viewerWs.readyState === WebSocket.OPEN || viewerWs.readyState === WebSocket.CONNECTING) {
            viewerWs.close()
          }
        }
      } catch {}
    }

    const sourceBuffer = screenViewerSourceBufferRef.current
    const mediaSource = screenViewerMediaSourceRef.current
    const sourceOpenHandler = screenViewerSourceOpenHandlerRef.current
    const updateEndHandler = screenViewerUpdateEndHandlerRef.current

    try {
      if (sourceBuffer && updateEndHandler) {
        sourceBuffer.removeEventListener('updateend', updateEndHandler)
      }
    } catch {}
    try {
      if (mediaSource && sourceOpenHandler) {
        mediaSource.removeEventListener('sourceopen', sourceOpenHandler)
      }
    } catch {}
    try {
      if (mediaSource && mediaSource.readyState === 'open') {
        mediaSource.endOfStream()
      }
    } catch {}

    screenViewerSourceBufferRef.current = null
    screenViewerMediaSourceRef.current = null
    screenViewerSourceOpenHandlerRef.current = null
    screenViewerUpdateEndHandlerRef.current = null
    screenViewerChunkQueueRef.current = []
    screenViewerPendingInitRef.current = null
    screenViewerTrimInProgressRef.current = false

    if (screenViewerObjectUrlRef.current) {
      const url = screenViewerObjectUrlRef.current
      screenViewerObjectUrlRef.current = ''
      try {
        const videoEl = screenViewerVideoRef.current
        if (videoEl) {
          try { videoEl.pause() } catch {}
          try { videoEl.removeAttribute('src') } catch {}
          try { videoEl.load() } catch {}
        }
      } catch {}
      try { URL.revokeObjectURL(url) } catch {}
    }

    if (clearState) {
      if (!keepSocket) setScreenViewerConnected(false)
      setScreenViewerReady(false)
      setScreenViewerMimeType('')
      if (!preserveError) setScreenViewerError('')
    } else {
      if (!keepSocket) setScreenViewerConnected(false)
      setScreenViewerReady(false)
    }

    setTimeout(() => { screenViewerStoppingRef.current = false }, 0)
  }

  const maybeTrimScreenViewerBuffer = () => {
    if (screenViewerTrimInProgressRef.current) return false
    const sb = screenViewerSourceBufferRef.current
    const videoEl = screenViewerVideoRef.current
    if (!sb || !videoEl || sb.updating) return false
    let buffered
    try {
      buffered = sb.buffered
    } catch {
      return false
    }
    if (!buffered || buffered.length === 0) return false
    try {
      const start = buffered.start(0)
      const end = buffered.end(buffered.length - 1)
      if ((end - start) < 8) return false
      const trimTo = Math.max(0, (videoEl.currentTime || 0) - 1)
      if (trimTo <= start + 0.5) return false
      screenViewerTrimInProgressRef.current = true
      sb.remove(start, trimTo)
      return true
    } catch {
      screenViewerTrimInProgressRef.current = false
      return false
    }
  }

  const flushScreenViewerChunkQueue = () => {
    const ms = screenViewerMediaSourceRef.current
    const sb = screenViewerSourceBufferRef.current
    if (!ms || !sb) return
    if (ms.readyState !== 'open' || sb.updating) return
    const next = screenViewerChunkQueueRef.current.shift()
    if (!next) {
      const videoEl = screenViewerVideoRef.current
      if (videoEl && videoEl.paused) {
        videoEl.play().catch(() => {})
      }
      return
    }
    try {
      sb.appendBuffer(next)
    } catch (err) {
      console.error('Failed to append screen share chunk', err)
      setScreenViewerError('Failed to render screen share stream.')
      stopLocalScreenShareViewer({ clearState: true, preserveError: true })
    }
  }

  const initializeScreenViewerMse = (initPayload) => {
    const mimeType = initPayload?.mime_type
    if (!mimeType) {
      setScreenViewerError('Screen share init is missing a mime type.')
      stopLocalScreenShareViewer({ clearState: true, preserveError: true })
      return
    }
    if (typeof window === 'undefined' || typeof window.MediaSource === 'undefined') {
      setScreenViewerError('MediaSource playback is not supported in this browser.')
      stopLocalScreenShareViewer({ clearState: true, preserveError: true })
      return
    }
    if (window.MediaSource.isTypeSupported && !window.MediaSource.isTypeSupported(mimeType)) {
      setScreenViewerError(`Unsupported screen stream format: ${mimeType}`)
      stopLocalScreenShareViewer({ clearState: true, preserveError: true })
      return
    }

    stopLocalScreenShareViewer({ clearState: true, preserveError: true, keepSocket: true })
    screenViewerPendingInitRef.current = initPayload
    setScreenViewerError('')
    setScreenViewerReady(false)
    setScreenViewerMimeType(mimeType)

    const mediaSource = new window.MediaSource()
    screenViewerMediaSourceRef.current = mediaSource
    const objectUrl = URL.createObjectURL(mediaSource)
    screenViewerObjectUrlRef.current = objectUrl

    const onSourceOpen = () => {
      const ms = screenViewerMediaSourceRef.current
      if (!ms || ms !== mediaSource) return
      try {
        const sb = ms.addSourceBuffer(mimeType)
        screenViewerSourceBufferRef.current = sb
        const onUpdateEnd = () => {
          if (screenViewerTrimInProgressRef.current) {
            screenViewerTrimInProgressRef.current = false
          }
          if (!screenViewerReady) setScreenViewerReady(true)
          if (maybeTrimScreenViewerBuffer()) return
          flushScreenViewerChunkQueue()
        }
        screenViewerUpdateEndHandlerRef.current = onUpdateEnd
        sb.addEventListener('updateend', onUpdateEnd)
        flushScreenViewerChunkQueue()
      } catch (err) {
        console.error('Failed to initialize MSE source buffer', err)
        setScreenViewerError('Failed to initialize viewer playback.')
        stopLocalScreenShareViewer({ clearState: true, preserveError: true })
      }
    }
    screenViewerSourceOpenHandlerRef.current = onSourceOpen
    mediaSource.addEventListener('sourceopen', onSourceOpen)

    const videoEl = screenViewerVideoRef.current
    if (videoEl) {
      try {
        videoEl.muted = true
        videoEl.playsInline = true
        videoEl.autoplay = true
        videoEl.src = objectUrl
      } catch (err) {
        console.error('Failed to attach viewer video element', err)
        setScreenViewerError('Failed to attach viewer video element.')
        stopLocalScreenShareViewer({ clearState: true, preserveError: true })
      }
    }
  }

  const scheduleScreenShareViewerReconnect = (share) => {
    if (!share?.sessionId) return
    if (screenViewerReconnectTimerRef.current) return
    const attempt = screenViewerReconnectAttemptRef.current
    const delay = SCREEN_VIEWER_RECONNECT_DELAYS_MS[
      Math.min(attempt, SCREEN_VIEWER_RECONNECT_DELAYS_MS.length - 1)
    ]
    screenViewerReconnectAttemptRef.current = attempt + 1
    screenViewerReconnectTimerRef.current = setTimeout(() => {
      screenViewerReconnectTimerRef.current = null
      const currentShare = activeScreenShareRef.current
      if (!currentShare) return
      if (currentShare.sessionId !== share.sessionId) return
      if (currentShare.status !== 'started') return
      if (currentShare.sharerUserId === user?.id) return
      connectScreenShareViewer(currentShare)
    }, delay)
  }

  const connectScreenShareViewer = (share) => {
    if (!share?.sessionId || !token) return
    if (screenViewerWsRef.current && screenViewerSessionRef.current === share.sessionId) return

    clearScreenViewerReconnectTimer()
    stopLocalScreenShareViewer({ clearState: true, preserveError: false })
    setScreenViewerError('')
    setScreenViewerReady(false)
    setScreenViewerConnected(false)
    screenViewerChunkQueueRef.current = []

    const viewerWs = new WebSocket(buildScreenStreamWsUrl(share.sessionId, 'viewer'))
    viewerWs.binaryType = 'arraybuffer'
    screenViewerWsRef.current = viewerWs
    screenViewerSessionRef.current = share.sessionId

    viewerWs.onopen = () => {
      if (screenViewerWsRef.current !== viewerWs) return
      clearScreenViewerReconnectTimer({ resetAttempts: true })
      setScreenViewerConnected(true)
      setScreenViewerError('')
    }

    viewerWs.onmessage = (event) => {
      if (screenViewerWsRef.current !== viewerWs) return
      const data = event.data
      if (typeof data === 'string') {
        try {
          const parsed = JSON.parse(data)
          if (parsed?.type === 'init') {
            initializeScreenViewerMse(parsed)
          }
        } catch (err) {
          console.error('Failed to parse screen viewer init', err)
        }
        return
      }

      const enqueueChunk = (buf) => {
        if (!buf || buf.byteLength === 0) return
        screenViewerChunkQueueRef.current.push(new Uint8Array(buf))
        if (screenViewerChunkQueueRef.current.length > SCREEN_VIEWER_MAX_QUEUED_CHUNKS) {
          const dropCount = screenViewerChunkQueueRef.current.length - SCREEN_VIEWER_MAX_QUEUED_CHUNKS
          screenViewerChunkQueueRef.current.splice(0, dropCount)
        }
        flushScreenViewerChunkQueue()
      }

      if (data instanceof ArrayBuffer) {
        enqueueChunk(data)
        return
      }
      if (data instanceof Blob) {
        data.arrayBuffer().then(enqueueChunk).catch((err) => {
          console.error('Failed reading viewer chunk blob', err)
        })
      }
    }

    viewerWs.onerror = () => {
      if (screenViewerWsRef.current !== viewerWs) return
      if (!screenViewerStoppingRef.current) {
        setScreenViewerError('Viewer stream connection error.')
      }
    }

    viewerWs.onclose = () => {
      if (screenViewerWsRef.current !== viewerWs) return
      const wasIntentional = screenViewerStoppingRef.current
      const currentSession = activeScreenShareRef.current?.sessionId
      const shouldStillBeViewing =
        activeScreenShareRef.current &&
        activeScreenShareRef.current.status === 'started' &&
        activeScreenShareRef.current.sharerUserId !== user?.id &&
        currentSession === share.sessionId
      stopLocalScreenShareViewer({ clearState: true, preserveError: true })
      if (shouldStillBeViewing && !wasIntentional) {
        setScreenViewerError('Viewer stream disconnected. Reconnecting...')
        scheduleScreenShareViewerReconnect(share)
      }
    }
  }

  useEffect(() => {
    const share = activeScreenShare
    const shouldView =
      !!share?.sessionId &&
      share.status === 'started' &&
      share.sharerUserId &&
      share.sharerUserId !== user?.id &&
      !!token

    if (!shouldView) {
      stopLocalScreenShareViewer({ clearState: true, preserveError: false })
      return
    }

    connectScreenShareViewer(share)
  }, [activeScreenShare, user?.id, token])

  const requestScreenShareStart = () => {
    if (!wsRef.current || !isConnected) {
      setScreenShareError('You must be connected to start screen sharing.')
      return
    }
    if (activeScreenShare && activeScreenShare.sharerUserId !== user?.id) {
      setScreenShareDeniedInfo({
        sharerUserId: activeScreenShare.sharerUserId,
        sharerName: activeScreenShare.sharerName || 'Another member',
        reason: 'already_active',
        activeSession: activeScreenShare.sessionId || '',
      })
      return
    }
    setScreenShareError('')
    setScreenShareDeniedInfo(null)
    setScreenSharePendingStart(true)
    wsRef.current.send(JSON.stringify({
      type: 'screen_share_request_start',
      payload: {},
    }))
  }

  const stopScreenShare = () => {
    if (!wsRef.current || !isConnected) {
      stopLocalScreenSharePublisher({ notifyServer: false, clearLocalState: true })
      setScreenShareError('Connection lost. Unable to stop screen sharing.')
      return
    }
    setActiveScreenShare((prev) => {
      if (!prev) return prev
      return { ...prev, status: 'stopping' }
    })
    stopLocalScreenSharePublisher({ notifyServer: true, clearLocalState: false })
  }

  const sendMessage = async () => {
    if (sending) return
    if (!pendingAttachments.length && !newMessage.trim()) return

    setSending(true)
    setUploadError('')

    try {
      const attachmentsToSend = [...pendingAttachments]

      // Send pending attachments first (if any)
      for (const attachment of attachmentsToSend) {
        await uploadAttachmentAndSend(attachment)
        removePendingAttachment(attachment.id, { revokePreview: false })
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
      const msg = error.response?.data?.error || error.response?.data?.message || error.message || 'Failed to send message'
      setUploadError(msg)
    } finally {
      setSending(false)
    }
  }

  const uploadAttachmentAndSend = async (attachment) => {
    const file = attachment.file
    try {
      setUploading(true)
      updatePendingAttachment(attachment.id, { progress: 0, error: '' })
      const form = new FormData()
      form.append('file', file)
      const res = await api.post(`/upload?team_id=${teamId}`, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (event) => {
          if (!event) return
          if (!event.total || event.total <= 0) return
          const next = Math.max(0, Math.min(100, Math.round((event.loaded / event.total) * 100)))
          updatePendingAttachment(attachment.id, { progress: next })
        },
      })
      const url = res.data.data?.url
      if (!url) return
      const msgType = isImageFile(file) ? 'image' : 'file'
      // Seed local cache with preview for images and metadata for all attachments
      if (msgType === 'image' && attachment.previewUrl) {
        imageUrlsRef.current[url] = attachment.previewUrl
        setImageUrls(prev => ({ ...prev, [url]: attachment.previewUrl }))
      }
      const meta = { filename: file.name || '', contentType: file.type || '', size: file.size || 0 }
      attachmentMetaRef.current[url] = meta
      setAttachmentMeta(prev => ({ ...prev, [url]: meta }))
      // Prefer WS if connected
      if (wsRef.current && isConnected) {
        wsRef.current.send(JSON.stringify({
          type: 'chat_message',
          payload: { content: url, type: msgType }
        }))
      } else {
        const msgRes = await api.post(`/teams/${teamId}/messages`, { content: url, type: msgType })
        if (msgRes.data?.data) {
          setMessages(prev => [...prev, msgRes.data.data])
        }
      }
    } catch (e) {
      console.error('Upload failed', e)
      const msg = e.response?.data?.error || e.response?.data?.message || 'Upload failed'
      updatePendingAttachment(attachment.id, { error: msg, progress: null })
      setUploadError(msg)
      throw e
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
          appendPendingFiles([file])
          break
        }
      }
    }
  }

  const handleFileInputChange = (e) => {
    const files = e.target.files
    if (!files?.length) return
    appendPendingFiles(files)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleDragOver = (e) => {
    if (!e.dataTransfer?.types?.includes('Files')) return
    e.preventDefault()
    setIsDragOver(true)
  }

  const handleDragLeave = (e) => {
    if (e.currentTarget.contains(e.relatedTarget)) return
    setIsDragOver(false)
  }

  const handleDrop = (e) => {
    if (!e.dataTransfer?.files?.length) return
    e.preventDefault()
    setIsDragOver(false)
    appendPendingFiles(e.dataTransfer.files)
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
      setShowTeamSettings(false)
      navigate('/dashboard')
    } catch (error) {
      console.error('Failed to leave team:', error)
      const msg = error.response?.data?.error || error.response?.data?.message || 'Failed to leave team'
      throw new Error(msg)
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
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{team.name}</h1>
                {user?.id === team.owner_id && (
                  <span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-200">
                    <Shield className="mr-1 h-3 w-3" />
                    Owner
                  </span>
                )}
              </div>
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
                    onClick={() => {
                      void leaveTeam().catch((err) => {
                        window.alert(err?.message || 'Failed to leave team')
                      })
                    }}
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

      <div className="px-4 pt-3">
        <div className="rounded-xl border border-gray-200 bg-white/90 px-4 py-3 dark:border-gray-700 dark:bg-gray-800/90">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              <div className="text-sm font-semibold text-gray-900 dark:text-gray-100">Screen sharing</div>
              {activeScreenShare ? (
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  <span className="font-medium text-gray-800 dark:text-gray-100">
                    {activeScreenShare.sharerName || 'A member'}
                  </span>{' '}
                  is sharing their screen.
                  {isScreenSharePublisher ? ' You are the active sharer.' : ''}
                </p>
              ) : (
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  One member can share their screen at a time.
                </p>
              )}
              {screenShareDeniedInfo && !activeScreenShare && (
                <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                  {screenShareDeniedInfo.sharerName} is already sharing a screen.
                </p>
              )}
              {screenShareError && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-300">{screenShareError}</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              {isScreenSharePublisher ? (
                <button
                  type="button"
                  onClick={stopScreenShare}
                  disabled={!isConnected}
                  className="btn-outline border-red-300 text-red-600 hover:bg-red-50 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/40 disabled:opacity-50"
                >
                  Stop sharing
                </button>
              ) : (
                <button
                  type="button"
                  onClick={requestScreenShareStart}
                  disabled={
                    !isConnected ||
                    screenSharePendingStart ||
                    (activeScreenShare && activeScreenShare.sharerUserId !== user?.id)
                  }
                  className="btn-primary disabled:opacity-50"
                >
                  {screenSharePendingStart ? 'Requesting...' : 'Share screen'}
                </button>
              )}
            </div>
          </div>
          {activeScreenShare && (
            isScreenSharePublisher ? (
              <div className="mt-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-3 py-4 text-xs text-gray-500 dark:border-gray-600 dark:bg-gray-900/40 dark:text-gray-300">
                Your screen is being shared. Viewer playback is shown to other team members.
              </div>
            ) : (
              <div className="mt-3 space-y-2">
                <div className="overflow-hidden rounded-xl border border-gray-200 bg-black dark:border-gray-700">
                  <video
                    ref={screenViewerVideoRef}
                    autoPlay
                    muted
                    playsInline
                    className="block w-full h-auto max-h-[28rem] bg-black"
                  />
                </div>
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
                  <span className={`${screenViewerConnected ? 'text-green-600 dark:text-green-300' : 'text-gray-500 dark:text-gray-300'}`}>
                    {screenViewerConnected ? 'Viewer connected' : 'Viewer disconnected'}
                  </span>
                  <span className={`${screenViewerReady ? 'text-blue-600 dark:text-blue-300' : 'text-gray-500 dark:text-gray-300'}`}>
                    {screenViewerReady ? 'Rendering stream' : 'Waiting for stream data'}
                  </span>
                  {screenViewerMimeType && (
                    <span className="text-gray-500 dark:text-gray-300 truncate">
                      {screenViewerMimeType}
                    </span>
                  )}
                </div>
                {screenViewerError && (
                  <p className="text-xs text-red-600 dark:text-red-300">{screenViewerError}</p>
                )}
              </div>
            )
          )}
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
                  const isUpload = typeof message.content === 'string' && message.content.startsWith('/api/uploads/')
                  const isImage = message.type === 'image'
                  const isFileAttachment = message.type === 'file' || (isUpload && !isImage)
                  const key = message.id || message.content
                  if (isImage && isUpload) {
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
                  if (isFileAttachment) {
                    const src = key ? imageUrls[key] : null
                    const meta = key ? attachmentMeta[key] : null
                    const fileLabel = meta?.filename || 'Attachment'
                    const { Icon, label } = attachmentVisual(meta || {})
                    return (
                      <div className={`message-bubble ${message.user_id === user.id ? 'own' : 'other'} p-3`}>
                        <div className="flex items-center space-x-3">
                          <div className="w-9 h-9 rounded-lg bg-white/70 dark:bg-gray-800/70 border border-gray-200 dark:border-gray-700 flex items-center justify-center">
                            <Icon className="w-4 h-4" />
                          </div>
                          <div className="min-w-0">
                            <div dir="auto" className="text-sm font-medium truncate">{fileLabel}</div>
                            <div className="text-xs opacity-80">
                              {label}{meta?.size ? ` | ${formatFileSize(meta.size)}` : (src ? '' : ' | Loading file...')}
                            </div>
                          </div>
                        </div>
                        {src && (
                          <a
                            href={src}
                            target="_blank"
                            rel="noreferrer"
                            download={meta?.filename || undefined}
                            className="mt-2 inline-flex text-xs underline underline-offset-2"
                          >
                            Open / Download
                          </a>
                        )}
                      </div>
                    )
                  }
                  return (
                  <div className={`message-bubble ${
                    message.user_id === user.id ? 'own' : 'other'
                  }`}>
                    <p dir="auto" className="text-sm whitespace-pre-wrap break-words">{message.content}</p>
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
      <div
        className={`shrink-0 bg-white border-t border-gray-200 dark:bg-gray-800 dark:border-gray-700 p-4 pt-3 pb-3 transition-colors ${
          isDragOver ? 'bg-blue-50 dark:bg-blue-950/20' : ''
        }`}
        onDragOver={handleDragOver}
        onDragEnter={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <div className={`max-w-5xl mx-auto w-full space-y-3 ${isDragOver ? 'ring-2 ring-blue-400 rounded-xl p-2 -m-2' : ''}`}>
          {pendingAttachments.length > 0 && (
            <div className="space-y-2 max-h-72 overflow-y-auto pr-1 custom-scrollbar">
              {pendingAttachments.map((attachment) => {
                const { file, id, previewUrl, progress, error } = attachment
                const { Icon, label } = attachmentVisual({ filename: file?.name, contentType: file?.type })
                return (
                  <div
                    key={id}
                    className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex items-start space-x-3 min-w-0 flex-1">
                        {previewUrl ? (
                          <div className="w-20 h-20 overflow-hidden rounded-md bg-white dark:bg-gray-800 border dark:border-gray-700 shrink-0">
                            <img src={previewUrl} alt={file?.name || 'preview'} className="w-full h-full object-cover" />
                          </div>
                        ) : (
                          <div className="w-10 h-10 rounded-lg bg-white dark:bg-gray-800 border dark:border-gray-700 flex items-center justify-center shrink-0">
                            <Icon className="w-4 h-4 text-gray-500 dark:text-gray-300" />
                          </div>
                        )}
                        <div className="min-w-0 flex-1">
                          <p dir="auto" className="text-sm text-gray-700 dark:text-gray-100 truncate">{file?.name || 'Attachment'}</p>
                          <p className="text-xs text-gray-500 dark:text-gray-300 truncate">
                            {previewUrl ? 'Image' : label}
                            {file?.size ? ` | ${formatFileSize(file.size)}` : ''}
                            {file?.type ? ` | ${file.type}` : ''}
                          </p>
                          {progress != null && (
                            <div className="mt-2">
                              <div className="flex items-center justify-between text-xs text-gray-600 dark:text-gray-300 mb-1">
                                <span>Uploading</span>
                                <span>{progress}%</span>
                              </div>
                              <div className="h-2 rounded-full bg-gray-200 dark:bg-gray-700 overflow-hidden">
                                <div
                                  className="h-full bg-blue-500 transition-all"
                                  style={{ width: `${progress}%` }}
                                />
                              </div>
                            </div>
                          )}
                          {error && (
                            <p className="mt-1 text-xs text-red-600">{error}</p>
                          )}
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => removePendingAttachment(id)}
                        className="text-gray-500 dark:text-gray-300 hover:text-red-500 text-sm shrink-0"
                        disabled={sending || uploading}
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                )
              })}
              {pendingAttachments.length > 1 && (
                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={clearPendingAttachments}
                    className="text-xs text-gray-500 dark:text-gray-300 hover:text-red-500"
                    disabled={sending || uploading}
                  >
                    Clear all
                  </button>
                </div>
              )}
            </div>
          )}

          <div className="flex space-x-3 items-end">
            <input
              ref={fileInputRef}
              type="file"
              multiple
              className="hidden"
              onChange={handleFileInputChange}
            />
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="btn-outline px-3 h-12"
              disabled={sending || uploading || !isConnected}
              title="Attach file"
            >
              <Paperclip className="w-5 h-5" />
            </button>
            <textarea
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              onPaste={handlePaste}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  sendMessage()
                }
              }}
              onInput={handleTyping}
              placeholder="Type a message..."
              dir="auto"
              rows={1}
              className="flex-1 input h-auto min-h-[3rem] max-h-32 py-3 resize-none leading-5 overflow-y-auto"
              disabled={!isConnected}
            />
            <button
              onClick={sendMessage}
              disabled={(!newMessage.trim() && pendingAttachments.length === 0) || sending || uploading || !isConnected}
              className="btn-primary px-6 disabled:opacity-50 disabled:cursor-not-allowed h-12"
            >
              {sending || uploading ? (
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
              ) : (
                <Send className="w-5 h-5" />
              )}
            </button>
          </div>
          {uploadError && (
            <p className="text-xs text-red-600">
              {uploadError}
            </p>
          )}
          {isDragOver && (
            <p className="text-xs text-blue-600 dark:text-blue-300">
              Drop file(s) to attach them to your next message
            </p>
          )}
          
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
          onLeave={leaveTeam}
          isOwner={user?.id === team.owner_id}
          onUploadAvatar={uploadTeamAvatar}
          onTeamUpdated={setTeam}
        />
      )}
    </div>
  )
}

export default Chat
