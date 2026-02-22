package httpapi

import (
	"SuperChat/internal/model"
	"SuperChat/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (s *Server) HandleCreateTeam(w http.ResponseWriter, r *http.Request) { s.handleCreateTeam(w, r) }
func (s *Server) HandleGetTeams(w http.ResponseWriter, r *http.Request)   { s.handleGetTeams(w, r) }
func (s *Server) HandleGetTeam(w http.ResponseWriter, r *http.Request)    { s.handleGetTeam(w, r) }
func (s *Server) HandleUpdateTeam(w http.ResponseWriter, r *http.Request) { s.handleUpdateTeam(w, r) }
func (s *Server) HandleDeleteTeam(w http.ResponseWriter, r *http.Request) { s.handleDeleteTeam(w, r) }

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleCreateTeam != nil {
			s.deps.HandleCreateTeam(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}

	userID, _ := getUserFromContext(r)
	var req model.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	team, err := s.deps.TeamCore.CreateTeam(userID, req)
	if err != nil {
		s.writeTeamCoreError(w, err,
			map[error]string{},
			"Failed to create team",
		)
		return
	}
	WriteSuccess(w, team, "Team created successfully")
}

func (s *Server) handleGetTeams(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleGetTeams != nil {
			s.deps.HandleGetTeams(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	userID, _ := getUserFromContext(r)
	teams, err := s.deps.TeamCore.ListTeams(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get teams")
		return
	}
	WriteSuccess(w, teams, "Teams retrieved successfully")
}

func (s *Server) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleGetTeam != nil {
			s.deps.HandleGetTeam(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	team, err := s.deps.TeamCore.GetTeam(teamID, userID)
	if err != nil {
		s.writeTeamCoreError(w, err,
			map[error]string{
				service.ErrForbidden: "Access denied",
				service.ErrNotFound:  "Team not found",
			},
			"Failed to get team",
		)
		return
	}
	WriteSuccess(w, team, "Team retrieved successfully")
}

func (s *Server) handleUpdateTeam(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleUpdateTeam != nil {
			s.deps.HandleUpdateTeam(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}

	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Avatar      *string `json:"avatar"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	team, err := s.deps.TeamCore.UpdateTeam(teamID, userID, service.UpdateTeamInput{
		Name:        body.Name,
		Description: body.Description,
		Avatar:      body.Avatar,
	})
	if err != nil {
		s.writeTeamCoreError(w, err,
			map[error]string{
				service.ErrForbidden: "Only team owner can update the team",
				service.ErrNotFound:  "Team not found",
			},
			"Failed to update team",
		)
		return
	}
	WriteSuccess(w, team, "Team updated successfully")
}

func (s *Server) handleDeleteTeam(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleDeleteTeam != nil {
			s.deps.HandleDeleteTeam(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	if err := s.deps.TeamCore.DeleteTeam(teamID, userID); err != nil {
		s.writeTeamCoreError(w, err,
			map[error]string{
				service.ErrForbidden: "Only team owner can delete the team",
			},
			"Failed to delete team",
		)
		return
	}
	WriteSuccess(w, nil, "Team deleted successfully")
}

func (s *Server) writeTeamCoreError(w http.ResponseWriter, err error, known map[error]string, fallback500 string) {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		WriteError(w, http.StatusBadRequest, ve.Message)
		return
	}

	for target, msg := range known {
		if errors.Is(err, target) {
			status := http.StatusInternalServerError
			switch target {
			case service.ErrForbidden:
				status = http.StatusForbidden
			case service.ErrNotFound:
				status = http.StatusNotFound
			}
			WriteError(w, status, msg)
			return
		}
	}

	if strings.TrimSpace(fallback500) == "" {
		fallback500 = "Internal server error"
	}
	WriteError(w, http.StatusInternalServerError, fallback500)
}

func getUserFromContext(r *http.Request) (string, string) {
	userID, _ := r.Context().Value("user_id").(string)
	username, _ := r.Context().Value("username").(string)
	return userID, username
}
