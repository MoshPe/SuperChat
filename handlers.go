package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Auth handlers
func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	name := strings.TrimSpace(req.Name)
	username := strings.TrimSpace(req.Username)
	if username == "" || name == "" || req.Password == "" {
		writeErrorResponse(w, http.StatusBadRequest, "All fields are required")
		return
	}

	if len(req.Password) < 6 {
		writeErrorResponse(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Hash password
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Create user
	user := NewUser(username, name, hashedPassword)

	if err := createUser(user); err != nil {
		if errors.Is(err, errUsernameExists) {
			writeErrorResponse(w, http.StatusConflict, "Username already exists")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate token
	token, err := generateToken(user)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Remove password from response
	user.Password = ""

	response := map[string]interface{}{
		"user":  user,
		"token": token,
	}

	writeSuccessResponse(w, response, "User registered successfully")
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Get user by Username
	user, err := getUserByUsername(req.Username)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if !checkPassword(req.Password, user.Password) {
		writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate token
	token, err := generateToken(user)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Remove password from response
	user.Password = ""

	response := map[string]interface{}{
		"user":  user,
		"token": token,
	}

	writeSuccessResponse(w, response, "Login successful")
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	// In a stateless JWT system, logout is handled client-side
	// But we could implement a blacklist if needed
	writeSuccessResponse(w, nil, "Logout successful")
}

// User handlers
func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	user, err := getUserByID(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Remove password from response
	user.Password = ""
	writeSuccessResponse(w, user, "Profile retrieved successfully")
}

func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	var updateData struct {
		Name     *string `json:"name"`
		Username *string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := getUserByID(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Username is immutable
	if updateData.Username != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Username cannot be changed")
		return
	}

	// Update allowed fields
	if updateData.Name != nil {
		name := strings.TrimSpace(*updateData.Name)
		if name == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Name cannot be empty")
			return
		}
		user.Name = name
	}

	user.UpdatedAt = time.Now()

	if err := updateUser(user); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// Remove password from response
	user.Password = ""
	writeSuccessResponse(w, user, "Profile updated successfully")
}

// Change password handler
func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeErrorResponse(w, http.StatusBadRequest, "All fields are required")
		return
	}
	if len(req.NewPassword) < 6 {
		writeErrorResponse(w, http.StatusBadRequest, "New password must be at least 6 characters")
		return
	}

	user, err := getUserByID(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify current password
	if !checkPassword(req.CurrentPassword, user.Password) {
		writeErrorResponse(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash and update
	hashed, err := hashPassword(req.NewPassword)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
		return
	}
	user.Password = hashed
	user.UpdatedAt = time.Now()

	if err := updateUser(user); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	writeSuccessResponse(w, nil, "Password updated successfully")
}

// Team handlers
func handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Team name is required")
		return
	}

	// Create team
	team := NewTeam(req.Name, req.Description, userID)
	if err := createTeam(team); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create team")
		return
	}

	// Add user as owner
	member := NewTeamMember(team.ID, userID, "owner")
	if err := addTeamMember(member); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to add team member")
		return
	}

	writeSuccessResponse(w, team, "Team created successfully")
}

func handleGetTeams(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	teams, err := getUserTeams(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get teams")
		return
	}

	writeSuccessResponse(w, teams, "Teams retrieved successfully")
}

// handleGetOnlineCounts returns the number of online users per team for the current user
func handleGetOnlineCounts(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	teams, err := getUserTeams(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get teams")
		return
	}

	counts := make(map[string]int)

	// Read hub rooms safely and count unique users per team
	hub.Mutex.RLock()
	for _, team := range teams {
		if room, exists := hub.Rooms[team.ID]; exists {
			room.Mutex.RLock()
			users := make(map[string]struct{})
			for client := range room.Clients {
				users[client.UserID] = struct{}{}
			}
			room.Mutex.RUnlock()
			counts[team.ID] = len(users)
		} else {
			counts[team.ID] = 0
		}
	}
	hub.Mutex.RUnlock()

	writeSuccessResponse(w, counts, "Online counts retrieved successfully")
}

func handleGetTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	// Check if user is member of the team
	if !isTeamMember(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	team, err := getTeamByID(teamID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Team not found")
		return
	}

	writeSuccessResponse(w, team, "Team retrieved successfully")
}

func handleUpdateTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	if !isTeamOwner(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Only team owner can update the team")
		return
	}

	team, err := getTeamByID(teamID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Team not found")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Avatar      *string `json:"avatar"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Team name cannot be empty")
			return
		}
		team.Name = name
	}
	if body.Description != nil {
		team.Description = strings.TrimSpace(*body.Description)
	}
	if body.Avatar != nil {
		team.Avatar = strings.TrimSpace(*body.Avatar)
	}
	team.UpdatedAt = time.Now()

	if err := updateTeam(team); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update team")
		return
	}

	writeSuccessResponse(w, team, "Team updated successfully")
}

func handleDeleteTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	if !isTeamOwner(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Only team owner can delete the team")
		return
	}

	if err := deleteTeam(teamID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete team")
		return
	}

	writeSuccessResponse(w, nil, "Team deleted successfully")
}

// List all users (id, username) - for owner to pick members
func handleListUsers(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)
	teamID := strings.TrimSpace(r.URL.Query().Get("team_id"))
	if teamID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "team_id is required")
		return
	}
	if !isTeamOwner(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Only team owner can list users")
		return
	}

	users, err := getAllUsers()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get users")
		return
	}
	// Strip password
	for _, u := range users {
		u.Password = ""
	}
	writeSuccessResponse(w, users, "Users retrieved successfully")
}

// Get team members (with roles)
func handleGetTeamMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	if !isTeamMember(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	members, err := getTeamMembers(teamID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get team members")
		return
	}

	// Enrich with username
	type MemberDTO struct {
		ID       string    `json:"id"`
		TeamID   string    `json:"team_id"`
		UserID   string    `json:"user_id"`
		Username string    `json:"username"`
		Role     string    `json:"role"`
		JoinedAt time.Time `json:"joined_at"`
	}
	var result []MemberDTO
	for _, m := range members {
		u, _ := getUserByID(m.UserID)
		username := ""
		if u != nil {
			username = u.Username
		}
		result = append(result, MemberDTO{
			ID: m.ID, TeamID: m.TeamID, UserID: m.UserID, Username: username, Role: m.Role, JoinedAt: m.JoinedAt,
		})
	}

	writeSuccessResponse(w, result, "Team members retrieved successfully")
}

// Add a member by username (owner only)
func handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	if !isTeamOwner(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Only team owner can add members")
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	target, err := getUserByUsername(body.Username)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	if isTeamMember(teamID, target.ID) {
		writeErrorResponse(w, http.StatusConflict, "User already a member")
		return
	}

	member := NewTeamMember(teamID, target.ID, "member")
	if err := addTeamMember(member); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to add member")
		return
	}

	writeSuccessResponse(w, map[string]string{"user_id": target.ID}, "Member added successfully")
}

// Remove a member by user ID (owner only, cannot remove owner)
func handleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	targetUserID := vars["userId"]
	userID, _ := getUserFromContext(r)

	if !isTeamOwner(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Only team owner can remove members")
		return
	}

	// Prevent removing owner
	if isTeamOwner(teamID, targetUserID) {
		writeErrorResponse(w, http.StatusBadRequest, "Cannot remove team owner")
		return
	}

	if !isTeamMember(teamID, targetUserID) {
		writeErrorResponse(w, http.StatusNotFound, "User is not a member")
		return
	}

	if err := removeTeamMember(teamID, targetUserID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to remove member")
		return
	}

	// Kick active WS connections for removed user with notification
	kickUserFromTeam(teamID, targetUserID, "You were removed from the team by an admin.")

	writeSuccessResponse(w, nil, "Member removed successfully")
}

func handleJoinTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	// Check if team exists
	team, err := getTeamByID(teamID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Team not found")
		return
	}

	// Check if already a member
	if isTeamMember(teamID, userID) {
		writeErrorResponse(w, http.StatusConflict, "Already a member of this team")
		return
	}

	// Add user as member
	member := NewTeamMember(teamID, userID, "member")
	if err := addTeamMember(member); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to join team")
		return
	}

	writeSuccessResponse(w, team, "Successfully joined team")
}

func handleLeaveTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	// Check if user is owner
	if isTeamOwner(teamID, userID) {
		writeErrorResponse(w, http.StatusBadRequest, "Team owner cannot leave. Transfer ownership first.")
		return
	}

	// Check if user is member
	if !isTeamMember(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Not a member of this team")
		return
	}

	// Remove user from team
	if err := removeTeamMember(teamID, userID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to leave team")
		return
	}

	writeSuccessResponse(w, nil, "Successfully left team")
}

// Chat handlers
func handleGetMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, _ := getUserFromContext(r)

	// Check if user is member of the team
	if !isTeamMember(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Get limit from query params
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := getTeamMessages(teamID, limit)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get messages")
		return
	}

	writeSuccessResponse(w, messages, "Messages retrieved successfully")
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	userID, username := getUserFromContext(r)
	displayName := username
	if u, err := getUserByID(userID); err == nil && u.Name != "" {
		displayName = u.Name
	}

	// Check if user is member of the team
	if !isTeamMember(teamID, userID) {
		writeErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Message content is required")
		return
	}

	if req.Type == "" {
		req.Type = "text"
	}

	// Create message
	message := NewMessage(teamID, userID, displayName, req.Content, req.Type)
	if err := saveMessage(message); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to send message")
		return
	}

	writeSuccessResponse(w, message, "Message sent successfully")
}

// Delete a message (author or team owner)
func handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	messageID := vars["messageId"]
	userID, _ := getUserFromContext(r)

	msg, err := getMessageByID(messageID)
	if err != nil || msg == nil || msg.TeamID != teamID {
		writeErrorResponse(w, http.StatusNotFound, "Message not found")
		return
	}

	// Only author can delete
	if msg.UserID != userID {
		writeErrorResponse(w, http.StatusForbidden, "Not allowed to delete this message")
		return
	}

	if err := deleteMessage(messageID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete message")
		return
	}

	// Determine display name for notification
	deleterName := ""
	if u, err := getUserByID(userID); err == nil && u != nil {
		deleterName = u.Name
		if deleterName == "" {
			deleterName = u.Username
		}
	}
	if deleterName == "" {
		deleterName = "User"
	}

	// Broadcast deletion to connected clients in the team room
	hub.Mutex.RLock()
	room, ok := hub.Rooms[teamID]
	hub.Mutex.RUnlock()
	if ok {
		room.broadcast(WebSocketMessage{
			Type: "message_deleted",
			Payload: map[string]interface{}{
				"id":              messageID,
				"deleted_by":      userID,
				"deleted_by_name": deleterName,
			},
		}, nil)
	}

	writeSuccessResponse(w, map[string]string{"id": messageID}, "Message deleted successfully")
}

// handleUpload handles image uploads and returns a public URL
func handleUpload(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserFromContext(r)

	// Limit upload size to 8MB
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "File too large or invalid form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Missing file")
		return
	}
	defer file.Close()

	// Sniff content type
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	// Reset reader to start
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Allow only images
	switch contentType {
	case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif":
	default:
		writeErrorResponse(w, http.StatusBadRequest, "Unsupported file type")
		return
	}

	// Read entire file (guarded by MaxBytesReader and ParseMultipartForm size)
	data, err := io.ReadAll(file)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Save to BoltDB uploads bucket (associate owner; team linkage optional)
	teamID := r.URL.Query().Get("team_id")
	uploadID, err := saveUpload(header.Filename, contentType, userID, teamID, data)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to store file")
		return
	}

	// Public URL (served via API); access requires Authorization header
	url := "/api/uploads/" + uploadID
	writeSuccessResponse(w, map[string]string{"url": url}, "Upload successful")
}

// handleGetUpload streams an uploaded image by ID
func handleGetUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing id")
		return
	}

	// Require Authorization header (no more token query param)
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := validateToken(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	meta, data, err := getUpload(id)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Not found")
		return
	}

	// Access control: owner or member of the upload team (if set)
	if meta.OwnerID != "" && meta.OwnerID != claims.UserID {
		if meta.TeamID == "" || !isTeamMember(meta.TeamID, claims.UserID) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=31536000")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
