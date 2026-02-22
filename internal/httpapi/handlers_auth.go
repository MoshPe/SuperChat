package httpapi

import (
	"SuperChat/internal/model"
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	deps HandlerDeps
}

func NewServer(deps HandlerDeps) *Server {
	return &Server{deps: deps}
}

func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	s.handleRegister(w, r)
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	s.handleLogin(w, r)
}

func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	s.handleLogout(w, r)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if s.deps.HashPassword == nil || s.deps.CreateUser == nil || s.deps.GenerateToken == nil || s.deps.IsUsernameExists == nil {
		if s.deps.HandleRegister != nil {
			s.deps.HandleRegister(w, r)
			return
		}
	}

	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	username := strings.TrimSpace(req.Username)
	if username == "" || name == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "All fields are required")
		return
	}
	if len(req.Password) < 6 {
		WriteError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	hashedPassword, err := s.deps.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	user := model.NewUser(username, name, hashedPassword)
	if err := s.deps.CreateUser(user); err != nil {
		if s.deps.IsUsernameExists(err) {
			WriteError(w, http.StatusConflict, "Username already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	token, err := s.deps.GenerateToken(user)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	user.Password = ""
	WriteSuccess(w, map[string]interface{}{
		"user":  user,
		"token": token,
	}, "User registered successfully")
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.deps.GetUserByUsername == nil || s.deps.CheckPassword == nil || s.deps.GenerateToken == nil {
		if s.deps.HandleLogin != nil {
			s.deps.HandleLogin(w, r)
			return
		}
	}

	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := s.deps.GetUserByUsername(req.Username)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if !s.deps.CheckPassword(req.Password, user.Password) {
		WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := s.deps.GenerateToken(user)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	user.Password = ""
	WriteSuccess(w, map[string]interface{}{
		"user":  user,
		"token": token,
	}, "Login successful")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT logout remains a client-side operation.
	if s.deps.HandleLogout != nil {
		// Keep legacy handler path available during migration, but preserve contract if unset.
		s.deps.HandleLogout(w, r)
		return
	}
	WriteSuccess(w, nil, "Logout successful")
}
