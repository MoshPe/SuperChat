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
func (s *Server) HandleGetTeamMembers(w http.ResponseWriter, r *http.Request) {
	s.handleGetTeamMembers(w, r)
}
func (s *Server) HandleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	s.handleAddTeamMember(w, r)
}
func (s *Server) HandleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	s.handleRemoveTeamMember(w, r)
}
func (s *Server) HandleJoinTeam(w http.ResponseWriter, r *http.Request)  { s.handleJoinTeam(w, r) }
func (s *Server) HandleLeaveTeam(w http.ResponseWriter, r *http.Request) { s.handleLeaveTeam(w, r) }
func (s *Server) HandleTransferOwnership(w http.ResponseWriter, r *http.Request) {
	s.handleTransferOwnership(w, r)
}

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

func (s *Server) handleGetTeamMembers(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleGetTeamMembers != nil {
			s.deps.HandleGetTeamMembers(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	members, err := s.deps.TeamCore.GetTeamMembers(teamID, userID)
	if err != nil {
		s.writeTeamCoreError(w, err,
			map[error]string{service.ErrForbidden: "Access denied"},
			"Failed to get team members",
		)
		return
	}
	WriteSuccess(w, members, "Team members retrieved successfully")
}

func (s *Server) handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleAddTeamMember != nil {
			s.deps.HandleAddTeamMember(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" {
		WriteError(w, http.StatusBadRequest, "Username is required")
		return
	}
	targetID, err := s.deps.TeamCore.AddTeamMember(teamID, userID, body.Username)
	if err != nil {
		if s.writeTeamAddMemberError(w, err) {
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to add member")
		return
	}
	WriteSuccess(w, map[string]string{"user_id": targetID}, "Member added successfully")
}

func (s *Server) handleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleRemoveTeamMember != nil {
			s.deps.HandleRemoveTeamMember(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	vars := mux.Vars(r)
	teamID := vars["id"]
	targetUserID := vars["userId"]
	userID, _ := getUserFromContext(r)
	if err := s.deps.TeamCore.RemoveTeamMember(teamID, userID, targetUserID); err != nil {
		if s.writeTeamRemoveMemberError(w, err) {
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to remove member")
		return
	}
	if s.deps.KickUserFromTeam != nil {
		s.deps.KickUserFromTeam(teamID, targetUserID, "You were removed from the team by an admin.")
	}
	WriteSuccess(w, nil, "Member removed successfully")
}

func (s *Server) handleJoinTeam(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleJoinTeam != nil {
			s.deps.HandleJoinTeam(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	team, err := s.deps.TeamCore.JoinTeam(teamID, userID)
	if err != nil {
		if s.writeTeamJoinError(w, err) {
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to join team")
		return
	}
	WriteSuccess(w, team, "Successfully joined team")
}

func (s *Server) handleLeaveTeam(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleLeaveTeam != nil {
			s.deps.HandleLeaveTeam(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	if err := s.deps.TeamCore.LeaveTeam(teamID, userID); err != nil {
		if s.writeTeamLeaveError(w, err) {
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to leave team")
		return
	}
	WriteSuccess(w, nil, "Successfully left team")
}

func (s *Server) handleTransferOwnership(w http.ResponseWriter, r *http.Request) {
	if s.deps.TeamCore == nil {
		if s.deps.HandleTransferOwnership != nil {
			s.deps.HandleTransferOwnership(w, r)
			return
		}
		WriteError(w, http.StatusNotFound, "Not found")
		return
	}
	teamID := mux.Vars(r)["id"]
	userID, _ := getUserFromContext(r)
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	team, err := s.deps.TeamCore.TransferOwnership(teamID, userID, body.UserID)
	if err != nil {
		if s.writeTeamTransferOwnershipError(w, err) {
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to transfer ownership")
		return
	}
	WriteSuccess(w, team, "Ownership transferred successfully")
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

func (s *Server) writeTeamAddMemberError(w http.ResponseWriter, err error) bool {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		if ve.Message == "User already a member" {
			WriteError(w, http.StatusConflict, ve.Message)
			return true
		}
		WriteError(w, http.StatusBadRequest, ve.Message)
		return true
	}
	if errors.Is(err, service.ErrForbidden) {
		WriteError(w, http.StatusForbidden, "Only team owner can add members")
		return true
	}
	if errors.Is(err, service.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "User not found")
		return true
	}
	return false
}

func (s *Server) writeTeamRemoveMemberError(w http.ResponseWriter, err error) bool {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		WriteError(w, http.StatusBadRequest, ve.Message)
		return true
	}
	if errors.Is(err, service.ErrForbidden) {
		WriteError(w, http.StatusForbidden, "Only team owner can remove members")
		return true
	}
	if errors.Is(err, service.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "User is not a member")
		return true
	}
	return false
}

func (s *Server) writeTeamJoinError(w http.ResponseWriter, err error) bool {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		if ve.Message == "Already a member of this team" {
			WriteError(w, http.StatusConflict, ve.Message)
			return true
		}
		WriteError(w, http.StatusBadRequest, ve.Message)
		return true
	}
	if errors.Is(err, service.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "Team not found")
		return true
	}
	return false
}

func (s *Server) writeTeamLeaveError(w http.ResponseWriter, err error) bool {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		WriteError(w, http.StatusBadRequest, ve.Message)
		return true
	}
	if errors.Is(err, service.ErrForbidden) {
		WriteError(w, http.StatusForbidden, "Not a member of this team")
		return true
	}
	return false
}

func (s *Server) writeTeamTransferOwnershipError(w http.ResponseWriter, err error) bool {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		WriteError(w, http.StatusBadRequest, ve.Message)
		return true
	}
	if errors.Is(err, service.ErrForbidden) {
		WriteError(w, http.StatusForbidden, "Only the team owner can transfer ownership")
		return true
	}
	if errors.Is(err, service.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "Team not found")
		return true
	}
	return false
}

func getUserFromContext(r *http.Request) (string, string) {
	userID, _ := r.Context().Value("user_id").(string)
	username, _ := r.Context().Value("username").(string)
	return userID, username
}
