package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Password  string    `json:"password"` // Never send password in JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Team represents a chat team.
type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	Avatar      string    `json:"avatar"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeamMember represents a user's membership in a team.
type TeamMember struct {
	ID       string    `json:"id"`
	TeamID   string    `json:"team_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"` // "owner", "admin", "member"
	JoinedAt time.Time `json:"joined_at"`
}

// Message represents a chat message.
type Message struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // "text", "image", "file"
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// CreateTeamRequest represents a team creation request.
type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SendMessageRequest represents a message sending request.
type SendMessageRequest struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}

// APIResponse represents a standard API response.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// WebSocketMessage represents a WebSocket message.
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// UploadMeta stores upload metadata in BoltDB.
type UploadMeta struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	OwnerID     string    `json:"owner_id"`
	TeamID      string    `json:"team_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewUser(username, name, password string) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New().String(),
		Username:  username,
		Name:      name,
		Password:  password, // Will be hashed before storage
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewTeam(name, description, ownerID string) *Team {
	now := time.Now()
	return &Team{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		Avatar:      "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func NewMessage(teamID, userID, username, content, msgType string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		TeamID:    teamID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		Type:      msgType,
		CreatedAt: time.Now(),
	}
}

func NewTeamMember(teamID, userID, role string) *TeamMember {
	return &TeamMember{
		ID:       uuid.New().String(),
		TeamID:   teamID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}
}

// Serialization helpers.
func (u *User) ToJSON() ([]byte, error) {
	return json.Marshal(u)
}

func (t *Team) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (tm *TeamMember) ToJSON() ([]byte, error) {
	return json.Marshal(tm)
}
