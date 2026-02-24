package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	screenShareStatusReserved = "reserved"
	screenShareStatusActive   = "active"
	screenShareMaxViewers     = 10
	screenShareMaxChunkBytes  = 2 << 20
)

var (
	errScreenShareSessionNotFound          = errors.New("screen share session not found")
	errScreenSharePublisherUserMismatch    = errors.New("screen share publisher user mismatch")
	errScreenSharePublisherSessionMismatch = errors.New("screen share publisher session mismatch")
	errScreenShareViewerLimitReached       = errors.New("screen share viewer limit reached")
	errScreenShareChunkTooLarge            = errors.New("screen share chunk too large")
	errScreenShareInitNotSet               = errors.New("screen share init metadata not set")
)

type screenShareOutboundKind string

const (
	screenShareOutboundInit  screenShareOutboundKind = "init"
	screenShareOutboundChunk screenShareOutboundKind = "chunk"
)

type screenShareOutboundMessage struct {
	Kind screenShareOutboundKind
	Data []byte
}

type screenShareInit struct {
	MimeType  string `json:"mime_type"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	FPSTarget int    `json:"fps_target,omitempty"`
}

type screenShareAlreadyActiveError struct {
	TeamID        string
	SharerUserID  string
	SharerName    string
	ActiveSession string
}

func (e *screenShareAlreadyActiveError) Error() string {
	return fmt.Sprintf("screen share already active for team %s", e.TeamID)
}

type screenShareSession struct {
	SessionID    string
	TeamID       string
	SharerUserID string
	SharerName   string
	Status       string
	MimeType     string
	Width        int
	Height       int
	FPSTarget    int
	ReservedAt   time.Time
	StartedAt    time.Time
	LastSeenAt   time.Time

	initPayload []byte
	viewers     map[string]chan screenShareOutboundMessage
}

type screenShareRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*screenShareSession // teamID -> current session (reserved or active)
}

func newScreenShareRegistry() *screenShareRegistry {
	return &screenShareRegistry{
		sessions: make(map[string]*screenShareSession),
	}
}

func (r *screenShareRegistry) reserve(teamID, sharerUserID, sharerName string) (*screenShareSession, error) {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing := r.sessions[teamID]; existing != nil {
		return nil, &screenShareAlreadyActiveError{
			TeamID:        teamID,
			SharerUserID:  existing.SharerUserID,
			SharerName:    existing.SharerName,
			ActiveSession: existing.SessionID,
		}
	}

	sess := &screenShareSession{
		SessionID:    newScreenShareSessionID(now),
		TeamID:       teamID,
		SharerUserID: sharerUserID,
		SharerName:   sharerName,
		Status:       screenShareStatusReserved,
		ReservedAt:   now,
		LastSeenAt:   now,
		viewers:      make(map[string]chan screenShareOutboundMessage),
	}
	r.sessions[teamID] = sess
	return cloneScreenShareSession(sess), nil
}

func (r *screenShareRegistry) get(teamID string) *screenShareSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneScreenShareSession(r.sessions[teamID])
}

func (r *screenShareRegistry) release(teamID, sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing := r.sessions[teamID]
	if existing == nil {
		return false
	}
	if sessionID != "" && existing.SessionID != sessionID {
		return false
	}
	delete(r.sessions, teamID)
	return true
}

func (r *screenShareRegistry) validatePublisherAttach(teamID, sessionID, userID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	existing := r.sessions[teamID]
	if existing == nil {
		return errScreenShareSessionNotFound
	}
	if existing.SessionID != sessionID {
		return errScreenSharePublisherSessionMismatch
	}
	if existing.SharerUserID != userID {
		return errScreenSharePublisherUserMismatch
	}
	return nil
}

func newScreenShareSessionID(now time.Time) string {
	return fmt.Sprintf("ss-%d", now.UnixNano())
}

func cloneScreenShareSession(sess *screenShareSession) *screenShareSession {
	if sess == nil {
		return nil
	}
	clone := *sess
	return &clone
}

func (r *screenShareRegistry) publisherInit(teamID, sessionID, userID string, init screenShareInit) ([]byte, error) {
	now := time.Now()

	r.mu.Lock()
	sess := r.sessions[teamID]
	if sess == nil {
		r.mu.Unlock()
		return nil, errScreenShareSessionNotFound
	}
	if sess.SessionID != sessionID {
		r.mu.Unlock()
		return nil, errScreenSharePublisherSessionMismatch
	}
	if sess.SharerUserID != userID {
		r.mu.Unlock()
		return nil, errScreenSharePublisherUserMismatch
	}

	payload, err := json.Marshal(map[string]interface{}{
		"type":       "init",
		"mime_type":  init.MimeType,
		"width":      init.Width,
		"height":     init.Height,
		"fps_target": init.FPSTarget,
	})
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}

	sess.MimeType = init.MimeType
	sess.Width = init.Width
	sess.Height = init.Height
	sess.FPSTarget = init.FPSTarget
	sess.Status = screenShareStatusActive
	if sess.StartedAt.IsZero() {
		sess.StartedAt = now
	}
	sess.LastSeenAt = now
	sess.initPayload = append([]byte(nil), payload...)

	viewers := make([]chan screenShareOutboundMessage, 0, len(sess.viewers))
	for _, ch := range sess.viewers {
		viewers = append(viewers, ch)
	}
	r.mu.Unlock()

	msg := screenShareOutboundMessage{Kind: screenShareOutboundInit, Data: payload}
	for _, ch := range viewers {
		sendScreenShareOutbound(ch, msg)
	}
	return payload, nil
}

func (r *screenShareRegistry) addViewer(teamID, sessionID, viewerID string, ch chan screenShareOutboundMessage) error {
	if ch == nil {
		return fmt.Errorf("viewer channel required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	sess := r.sessions[teamID]
	if sess == nil {
		return errScreenShareSessionNotFound
	}
	if sess.SessionID != sessionID {
		return errScreenSharePublisherSessionMismatch
	}
	if sess.viewers == nil {
		sess.viewers = make(map[string]chan screenShareOutboundMessage)
	}
	if _, exists := sess.viewers[viewerID]; !exists && len(sess.viewers) >= screenShareMaxViewers {
		return errScreenShareViewerLimitReached
	}
	sess.viewers[viewerID] = ch

	if len(sess.initPayload) > 0 {
		sendScreenShareOutbound(ch, screenShareOutboundMessage{
			Kind: screenShareOutboundInit,
			Data: append([]byte(nil), sess.initPayload...),
		})
	}
	return nil
}

func (r *screenShareRegistry) removeViewer(teamID, sessionID, viewerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	sess := r.sessions[teamID]
	if sess == nil || sess.SessionID != sessionID || sess.viewers == nil {
		return false
	}
	if _, ok := sess.viewers[viewerID]; !ok {
		return false
	}
	delete(sess.viewers, viewerID)
	return true
}

func (r *screenShareRegistry) relayPublisherChunk(teamID, sessionID, userID string, chunk []byte) error {
	if len(chunk) > screenShareMaxChunkBytes {
		return errScreenShareChunkTooLarge
	}
	if len(chunk) == 0 {
		return nil
	}

	now := time.Now()
	r.mu.Lock()
	sess := r.sessions[teamID]
	if sess == nil {
		r.mu.Unlock()
		return errScreenShareSessionNotFound
	}
	if sess.SessionID != sessionID {
		r.mu.Unlock()
		return errScreenSharePublisherSessionMismatch
	}
	if sess.SharerUserID != userID {
		r.mu.Unlock()
		return errScreenSharePublisherUserMismatch
	}
	if len(sess.initPayload) == 0 {
		r.mu.Unlock()
		return errScreenShareInitNotSet
	}
	sess.LastSeenAt = now
	viewers := make([]chan screenShareOutboundMessage, 0, len(sess.viewers))
	for _, ch := range sess.viewers {
		viewers = append(viewers, ch)
	}
	r.mu.Unlock()

	msg := screenShareOutboundMessage{Kind: screenShareOutboundChunk, Data: append([]byte(nil), chunk...)}
	for _, ch := range viewers {
		sendScreenShareOutbound(ch, msg)
	}
	return nil
}

func (r *screenShareRegistry) expireStale(now time.Time, reservedTTL, activeIdleTTL time.Duration) []*screenShareSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	var expired []*screenShareSession
	for teamID, sess := range r.sessions {
		if sess == nil {
			delete(r.sessions, teamID)
			continue
		}
		shouldExpire := false
		switch sess.Status {
		case screenShareStatusReserved:
			if reservedTTL > 0 && now.Sub(sess.ReservedAt) > reservedTTL {
				shouldExpire = true
			}
		case screenShareStatusActive:
			if activeIdleTTL > 0 && now.Sub(sess.LastSeenAt) > activeIdleTTL {
				shouldExpire = true
			}
		}
		if !shouldExpire {
			continue
		}
		expired = append(expired, cloneScreenShareSession(sess))
		delete(r.sessions, teamID)
	}
	return expired
}

func (r *screenShareRegistry) releaseBySharer(teamID, userID string) (*screenShareSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sess := r.sessions[teamID]
	if sess == nil || sess.SharerUserID != userID {
		return nil, false
	}
	clone := cloneScreenShareSession(sess)
	delete(r.sessions, teamID)
	return clone, true
}

func sendScreenShareOutbound(ch chan screenShareOutboundMessage, msg screenShareOutboundMessage) {
	if ch == nil {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

func broadcastScreenShareStopped(room *TeamRoom, sess *screenShareSession, reason string) {
	if room == nil || sess == nil {
		return
	}
	payload := map[string]interface{}{
		"session_id":     sess.SessionID,
		"sharer_user_id": sess.SharerUserID,
		"sharer_name":    sess.SharerName,
		"timestamp":      time.Now(),
	}
	if reason != "" {
		payload["reason"] = reason
	}
	room.broadcast(WebSocketMessage{
		Type:    "screen_share_stopped",
		Payload: payload,
	}, nil)
}

func broadcastScreenShareStarted(room *TeamRoom, sess *screenShareSession) {
	if room == nil || sess == nil {
		return
	}
	room.broadcast(WebSocketMessage{
		Type: "screen_share_started",
		Payload: map[string]interface{}{
			"session_id":     sess.SessionID,
			"sharer_user_id": sess.SharerUserID,
			"sharer_name":    sess.SharerName,
			"timestamp":      time.Now(),
		},
	}, nil)
}

func handleScreenSharePublisherDisconnected(room *TeamRoom, teamID, sessionID, userID string) bool {
	sess := screenShareSessions.get(teamID)
	if sess == nil {
		return false
	}
	if sess.SessionID != sessionID || sess.SharerUserID != userID {
		return false
	}
	if !screenShareSessions.release(teamID, sessionID) {
		return false
	}
	broadcastScreenShareStopped(room, sess, "publisher_disconnected")
	return true
}

func releaseScreenShareForChatClientDisconnect(room *TeamRoom, c *Client) bool {
	if c == nil {
		return false
	}
	sess, ok := screenShareSessions.releaseBySharer(c.TeamID, c.UserID)
	if !ok {
		return false
	}
	broadcastScreenShareStopped(room, sess, "sharer_disconnected")
	return true
}

var screenShareSessions = newScreenShareRegistry()

func (c *Client) handleScreenShareRequestStart(room *TeamRoom) bool {
	if c == nil {
		return true
	}

	sess, err := screenShareSessions.reserve(c.TeamID, c.UserID, c.Username)
	if err != nil {
		if activeErr, ok := err.(*screenShareAlreadyActiveError); ok {
			c.sendWSControlMessage(WebSocketMessage{
				Type: "screen_share_denied",
				Payload: map[string]interface{}{
					"reason":         "already_active",
					"sharer_user_id": activeErr.SharerUserID,
					"sharer_name":    activeErr.SharerName,
					"active_session": activeErr.ActiveSession,
					"timestamp":      time.Now(),
				},
			})
			return true
		}
		c.sendWSControlMessage(WebSocketMessage{
			Type: "screen_share_error",
			Payload: map[string]interface{}{
				"message":   "Failed to start screen sharing",
				"timestamp": time.Now(),
			},
		})
		return true
	}

	c.sendWSControlMessage(WebSocketMessage{
		Type: "screen_share_start_granted",
		Payload: map[string]interface{}{
			"session_id":     sess.SessionID,
			"team_id":        sess.TeamID,
			"sharer_user_id": sess.SharerUserID,
			"sharer_name":    sess.SharerName,
			"status":         sess.Status,
			"timestamp":      time.Now(),
		},
	})
	return true
}

func (c *Client) handleScreenShareStop(room *TeamRoom) bool {
	if c == nil {
		return true
	}

	sess := screenShareSessions.get(c.TeamID)
	if sess == nil {
		c.sendWSControlMessage(WebSocketMessage{
			Type: "screen_share_error",
			Payload: map[string]interface{}{
				"message":   "No active screen share session",
				"timestamp": time.Now(),
			},
		})
		return true
	}
	if sess.SharerUserID != c.UserID {
		c.sendWSControlMessage(WebSocketMessage{
			Type: "screen_share_error",
			Payload: map[string]interface{}{
				"message":        "Only the current sharer can stop screen sharing",
				"sharer_user_id": sess.SharerUserID,
				"sharer_name":    sess.SharerName,
				"timestamp":      time.Now(),
			},
		})
		return true
	}

	if !screenShareSessions.release(c.TeamID, sess.SessionID) {
		c.sendWSControlMessage(WebSocketMessage{
			Type: "screen_share_error",
			Payload: map[string]interface{}{
				"message":   "Failed to stop screen sharing",
				"timestamp": time.Now(),
			},
		})
		return true
	}

	broadcastScreenShareStopped(room, sess, "stopped_by_sharer")

	return true
}

func (c *Client) sendWSControlMessage(msg WebSocketMessage) {
	if c == nil || c.Send == nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.Send <- data:
	default:
	}
}

func handleScreenStreamWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["teamId"]
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	claims, err := validateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if !isTeamMember(teamID, claims.UserID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	role := r.URL.Query().Get("role")
	sessionID := r.URL.Query().Get("session_id")
	if role == "" || sessionID == "" {
		http.Error(w, "role and session_id are required", http.StatusBadRequest)
		return
	}

	switch role {
	case "viewer":
		sess := screenShareSessions.get(teamID)
		if sess == nil || sess.SessionID != sessionID {
			http.Error(w, "No active screen share session", http.StatusConflict)
			return
		}
	case "publisher":
		if err := screenShareSessions.validatePublisherAttach(teamID, sessionID, claims.UserID); err != nil {
			switch {
			case errors.Is(err, errScreenSharePublisherUserMismatch):
				http.Error(w, "Publisher is not session owner", http.StatusForbidden)
			case errors.Is(err, errScreenShareSessionNotFound), errors.Is(err, errScreenSharePublisherSessionMismatch):
				http.Error(w, "Invalid screen share session", http.StatusConflict)
			default:
				http.Error(w, "Failed to validate screen share session", http.StatusInternalServerError)
			}
			return
		}
	default:
		http.Error(w, "Invalid screen stream role", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("screen stream websocket upgrade failed: %v", err)
		return
	}

	switch role {
	case "publisher":
		handleScreenStreamPublisher(conn, teamID, sessionID, claims.UserID)
	case "viewer":
		handleScreenStreamViewer(conn, teamID, sessionID, claims.UserID)
	default:
		_ = conn.Close()
	}
}

func handleScreenStreamPublisher(conn *websocket.Conn, teamID, sessionID, userID string) {
	room := getTeamRoom(teamID)
	defer func() {
		_ = conn.Close()
		handleScreenSharePublisherDisconnected(room, teamID, sessionID, userID)
	}()

	conn.SetReadLimit(screenShareMaxChunkBytes + 4096)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("screen publisher read error: %v", err)
			}
			return
		}

		switch msgType {
		case websocket.TextMessage:
			var raw map[string]interface{}
			if err := json.Unmarshal(payload, &raw); err != nil {
				log.Printf("screen publisher invalid text message: %v", err)
				continue
			}
			typ, _ := raw["type"].(string)
			if typ != "init" {
				continue
			}
			init := screenShareInit{
				MimeType:  asString(raw["mime_type"]),
				Width:     asInt(raw["width"]),
				Height:    asInt(raw["height"]),
				FPSTarget: asInt(raw["fps_target"]),
			}
			if init.MimeType == "" {
				log.Printf("screen publisher init missing mime_type")
				continue
			}
			if _, err := screenShareSessions.publisherInit(teamID, sessionID, userID, init); err != nil {
				log.Printf("screen publisher init failed: %v", err)
				return
			}
			broadcastScreenShareStarted(room, screenShareSessions.get(teamID))

		case websocket.BinaryMessage:
			if err := screenShareSessions.relayPublisherChunk(teamID, sessionID, userID, payload); err != nil {
				if errors.Is(err, errScreenShareChunkTooLarge) {
					log.Printf("screen publisher chunk too large: %d bytes", len(payload))
					continue
				}
				log.Printf("screen publisher relay error: %v", err)
				return
			}

		case websocket.PingMessage:
			_ = conn.WriteControl(websocket.PongMessage, nil, time.Now().Add(5*time.Second))
		}
	}
}

func handleScreenStreamViewer(conn *websocket.Conn, teamID, sessionID, userID string) {
	viewerID := fmt.Sprintf("%s-%d", userID, time.Now().UnixNano())
	outbound := make(chan screenShareOutboundMessage, 32)

	if err := screenShareSessions.addViewer(teamID, sessionID, viewerID, outbound); err != nil {
		status := websocket.ClosePolicyViolation
		msg := "viewer join rejected"
		if errors.Is(err, errScreenShareViewerLimitReached) {
			msg = "viewer limit reached"
		} else if errors.Is(err, errScreenShareSessionNotFound) || errors.Is(err, errScreenSharePublisherSessionMismatch) {
			msg = "invalid session"
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(status, msg), time.Now().Add(5*time.Second))
		_ = conn.Close()
		return
	}

	defer func() {
		screenShareSessions.removeViewer(teamID, sessionID, viewerID)
		_ = conn.Close()
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn.SetReadLimit(256)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		for {
			msgType, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if msgType == websocket.PingMessage {
				_ = conn.WriteControl(websocket.PongMessage, nil, time.Now().Add(5*time.Second))
			}
		}
	}()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-done:
			return
		case msg, ok := <-outbound:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			switch msg.Kind {
			case screenShareOutboundInit:
				if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
					return
				}
			case screenShareOutboundChunk:
				if err := conn.WriteMessage(websocket.BinaryMessage, msg.Data); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func getTeamRoom(teamID string) *TeamRoom {
	hub.Mutex.RLock()
	defer hub.Mutex.RUnlock()
	return hub.Rooms[teamID]
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
