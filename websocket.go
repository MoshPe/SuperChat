package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client
type Client struct {
	ID       string
	UserID   string
	Username string
	TeamID   string
	Conn     *websocket.Conn
	Send     chan []byte
	once     sync.Once
}

// TeamRoom represents a chat room for a team
type TeamRoom struct {
	ID      string
	Clients map[*Client]bool
	Mutex   sync.RWMutex
}

// Hub manages all team rooms
type Hub struct {
	Rooms map[string]*TeamRoom
	Mutex sync.RWMutex
}

var hub = &Hub{
	Rooms: make(map[string]*TeamRoom),
}

// handleWebSocket handles WebSocket connections for team chat
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["teamId"]
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	claims, err := validateToken(token)
	if err != nil {
		log.Println("Invalid token")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userID, username := claims.UserID, claims.Username
	displayName := claims.Name
	if displayName == "" {
		if u, err := getUserByID(userID); err == nil && u.Name != "" {
			displayName = u.Name
		} else {
			displayName = username
		}
	}

	// Check if user is member of the team
	if !isTeamMember(teamID, userID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client
	client := &Client{
		ID:       fmt.Sprintf("%s-%s", userID, teamID),
		UserID:   userID,
		Username: displayName,
		TeamID:   teamID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	// Get or create team room
	hub.Mutex.Lock()
	room, exists := hub.Rooms[teamID]
	if !exists {
		room = &TeamRoom{
			ID:      teamID,
			Clients: make(map[*Client]bool),
		}
		hub.Rooms[teamID] = room
	}
	hub.Mutex.Unlock()

	// Add client to room
	room.Mutex.Lock()
	room.Clients[client] = true
	room.Mutex.Unlock()

	// Send join notification
	joinMessage := WebSocketMessage{
		Type: "user_joined",
		Payload: map[string]interface{}{
			"user_id":   userID,
			"username":  username,
			"name":      displayName,
			"timestamp": time.Now(),
		},
	}
	room.broadcast(joinMessage, client)

	// Start goroutines for reading and writing
	go client.writePump(room)
	go client.readPump(room)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(room *TeamRoom) {
	defer func() {
		c.disconnect(room)
	}()

	c.Conn.SetReadLimit(512) // Max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Parse message
		var wsMessage WebSocketMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		// Handle different message types
		switch wsMessage.Type {
		case "chat_message":
			c.handleChatMessage(room, wsMessage)
		case "typing":
			c.handleTyping(room, wsMessage)
		case "ping":
			c.handlePing()
		default:
			log.Printf("Unknown message type: %s", wsMessage.Type)
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump(room *TeamRoom) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.disconnect(room)
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleChatMessage processes chat messages
func (c *Client) handleChatMessage(room *TeamRoom, wsMessage WebSocketMessage) {
	payload, ok := wsMessage.Payload.(map[string]interface{})
	if !ok {
		return
	}

	content, ok := payload["content"].(string)
	if !ok || content == "" {
		return
	}

	// Re-check membership; if removed, notify and disconnect
	if !isTeamMember(c.TeamID, c.UserID) {
		removalNotice := WebSocketMessage{
			Type: "removed_from_team",
			Payload: map[string]interface{}{
				"reason": "You were removed from the team.",
			},
		}
		if data, err := json.Marshal(removalNotice); err == nil {
			c.Send <- data
		}
		c.disconnect(room)
		return
	}

	// Determine message type (default to text)
	msgType := "text"
	if t, ok := payload["type"].(string); ok && t != "" {
		msgType = t
	}

	// Create and save message to database
	message := NewMessage(c.TeamID, c.UserID, c.Username, content, msgType)
	if err := saveMessage(message); err != nil {
		log.Printf("Failed to save message: %v", err)
		return
	}

	// Broadcast message to all clients in the room
	chatMessage := WebSocketMessage{
		Type: "chat_message",
		Payload: map[string]interface{}{
			"id":        message.ID,
			"user_id":   message.UserID,
			"username":  c.Username,
			"name":      c.Username,
			"content":   message.Content,
			"type":      message.Type,
			"timestamp": message.CreatedAt,
		},
	}
	room.broadcast(chatMessage, nil) // Send to all clients including sender
}

// handleTyping processes typing indicators
func (c *Client) handleTyping(room *TeamRoom, wsMessage WebSocketMessage) {
	typingMessage := WebSocketMessage{
		Type: "typing",
		Payload: map[string]interface{}{
			"user_id":   c.UserID,
			"username":  c.Username,
			"name":      c.Username,
			"timestamp": time.Now(),
		},
	}
	room.broadcast(typingMessage, c) // Send to all clients except sender
}

// handlePing responds to ping messages
func (c *Client) handlePing() {
	pongMessage := WebSocketMessage{
		Type:    "pong",
		Payload: map[string]interface{}{"timestamp": time.Now()},
	}
	if data, err := json.Marshal(pongMessage); err == nil {
		c.Send <- data
	}
}

// disconnect removes the client from the room
func (c *Client) disconnect(room *TeamRoom) {
	// Ensure disconnect logic runs only once per client
	c.once.Do(func() {
		room.Mutex.Lock()
		delete(room.Clients, c)
		room.Mutex.Unlock()

		// Safe close of send channel
		close(c.Send)
		// Close websocket connection
		_ = c.Conn.Close()

		// Send leave notification
		leaveMessage := WebSocketMessage{
			Type: "user_left",
			Payload: map[string]interface{}{
				"user_id":   c.UserID,
				"username":  c.Username,
				"timestamp": time.Now(),
			},
		}
		room.broadcast(leaveMessage, nil)

		// Clean up empty rooms
		room.Mutex.RLock()
		if len(room.Clients) == 0 {
			room.Mutex.RUnlock()
			hub.Mutex.Lock()
			delete(hub.Rooms, room.ID)
			hub.Mutex.Unlock()
		} else {
			room.Mutex.RUnlock()
		}
	})
}

// kickUserFromTeam notifies and disconnects a user from a team room if connected.
func kickUserFromTeam(teamID, userID, reason string) {
	hub.Mutex.RLock()
	room, ok := hub.Rooms[teamID]
	hub.Mutex.RUnlock()
	if !ok {
		return
	}
	room.Mutex.RLock()
	var targets []*Client
	for client := range room.Clients {
		if client.UserID == userID {
			targets = append(targets, client)
		}
	}
	room.Mutex.RUnlock()

	for _, client := range targets {
		notice := WebSocketMessage{
			Type: "removed_from_team",
			Payload: map[string]interface{}{
				"reason": reason,
			},
		}
		if data, err := json.Marshal(notice); err == nil {
			client.Send <- data
		}
		client.disconnect(room)
	}
}

// broadcast sends a message to all clients in the room
func (r *TeamRoom) broadcast(message WebSocketMessage, exclude *Client) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	// Collect clients that are blocked so we can disconnect after releasing lock
	var toRemove []*Client

	r.Mutex.RLock()
	for client := range r.Clients {
		if client != exclude {
			select {
			case client.Send <- data:
			default:
				// Mark for removal; don't close here to avoid double-close race
				toRemove = append(toRemove, client)
			}
		}
	}
	r.Mutex.RUnlock()

	// Disconnect blocked clients outside of lock
	for _, client := range toRemove {
		client.disconnect(r)
	}
}
