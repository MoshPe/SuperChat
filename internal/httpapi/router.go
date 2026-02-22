package httpapi

import (
	"SuperChat/internal/model"
	"SuperChat/internal/service"
	"net/http"

	"github.com/gorilla/mux"
)

// HandlerDeps will hold injected dependencies as the backend refactor progresses.
type HandlerDeps struct {
	AuthMiddleware func(http.HandlerFunc) http.HandlerFunc

	HandleRegister          http.HandlerFunc
	HandleLogin             http.HandlerFunc
	HandleLogout            http.HandlerFunc
	HandleCreateTeam        http.HandlerFunc
	HandleGetTeams          http.HandlerFunc
	HandleGetTeam           http.HandlerFunc
	HandleUpdateTeam        http.HandlerFunc
	HandleDeleteTeam        http.HandlerFunc
	HandleGetTeamMembers    http.HandlerFunc
	HandleAddTeamMember     http.HandlerFunc
	HandleRemoveTeamMember  http.HandlerFunc
	HandleJoinTeam          http.HandlerFunc
	HandleLeaveTeam         http.HandlerFunc
	HandleTransferOwnership http.HandlerFunc
	HandleGetUpload         http.HandlerFunc

	TeamCore         TeamCoreService
	KickUserFromTeam func(teamID, userID, reason string)

	HashPassword      func(string) (string, error)
	CheckPassword     func(string, string) bool
	CreateUser        func(*model.User) error
	GetUserByUsername func(string) (*model.User, error)
	GenerateToken     func(*model.User) (string, error)
	IsUsernameExists  func(error) bool
}

type TeamCoreService interface {
	CreateTeam(userID string, req model.CreateTeamRequest) (*model.Team, error)
	ListTeams(userID string) ([]*model.Team, error)
	GetTeam(teamID, userID string) (*model.Team, error)
	UpdateTeam(teamID, userID string, input service.UpdateTeamInput) (*model.Team, error)
	DeleteTeam(teamID, userID string) error
	GetTeamMembers(teamID, requesterID string) ([]service.TeamMemberView, error)
	AddTeamMember(teamID, requesterID, targetUsername string) (string, error)
	RemoveTeamMember(teamID, requesterID, targetUserID string) error
	JoinTeam(teamID, requesterID string) (*model.Team, error)
	LeaveTeam(teamID, requesterID string) error
	TransferOwnership(teamID, requesterID, newOwnerID string) (*model.Team, error)
}

// NewRouter returns the HTTP handler for the API/SPA server. Phase 1 starts with
// route registration for representative endpoints and will be expanded
// incrementally to own the full server routing tree.
func NewRouter(deps HandlerDeps) http.Handler {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	s := NewServer(deps)

	if deps.HandleRegister != nil {
		api.HandleFunc("/auth/register", s.handleRegister).Methods(http.MethodPost)
	}
	if deps.HandleLogin != nil {
		api.HandleFunc("/auth/login", s.handleLogin).Methods(http.MethodPost)
	}
	if deps.HandleLogout != nil {
		api.HandleFunc("/auth/logout", applyAuth(deps.AuthMiddleware, s.handleLogout)).Methods(http.MethodPost)
	}
	if deps.TeamCore != nil || deps.HandleCreateTeam != nil {
		api.HandleFunc("/teams", applyAuth(deps.AuthMiddleware, s.handleCreateTeam)).Methods(http.MethodPost)
	}
	if deps.TeamCore != nil || deps.HandleGetTeams != nil {
		api.HandleFunc("/teams", applyAuth(deps.AuthMiddleware, s.handleGetTeams)).Methods(http.MethodGet)
	}
	if deps.TeamCore != nil || deps.HandleGetTeam != nil {
		api.HandleFunc("/teams/{id}", applyAuth(deps.AuthMiddleware, s.handleGetTeam)).Methods(http.MethodGet)
	}
	if deps.TeamCore != nil || deps.HandleUpdateTeam != nil {
		api.HandleFunc("/teams/{id}", applyAuth(deps.AuthMiddleware, s.handleUpdateTeam)).Methods(http.MethodPut)
	}
	if deps.TeamCore != nil || deps.HandleDeleteTeam != nil {
		api.HandleFunc("/teams/{id}", applyAuth(deps.AuthMiddleware, s.handleDeleteTeam)).Methods(http.MethodDelete)
	}
	if deps.TeamCore != nil || deps.HandleGetTeamMembers != nil {
		api.HandleFunc("/teams/{id}/members", applyAuth(deps.AuthMiddleware, s.handleGetTeamMembers)).Methods(http.MethodGet)
	}
	if deps.TeamCore != nil || deps.HandleAddTeamMember != nil {
		api.HandleFunc("/teams/{id}/members", applyAuth(deps.AuthMiddleware, s.handleAddTeamMember)).Methods(http.MethodPost)
	}
	if deps.TeamCore != nil || deps.HandleRemoveTeamMember != nil {
		api.HandleFunc("/teams/{id}/members/{userId}", applyAuth(deps.AuthMiddleware, s.handleRemoveTeamMember)).Methods(http.MethodDelete)
	}
	if deps.TeamCore != nil || deps.HandleJoinTeam != nil {
		api.HandleFunc("/teams/{id}/join", applyAuth(deps.AuthMiddleware, s.handleJoinTeam)).Methods(http.MethodPost)
	}
	if deps.TeamCore != nil || deps.HandleLeaveTeam != nil {
		api.HandleFunc("/teams/{id}/leave", applyAuth(deps.AuthMiddleware, s.handleLeaveTeam)).Methods(http.MethodPost)
	}
	if deps.TeamCore != nil || deps.HandleTransferOwnership != nil {
		api.HandleFunc("/teams/{id}/transfer-ownership", applyAuth(deps.AuthMiddleware, s.handleTransferOwnership)).Methods(http.MethodPost)
	}
	if deps.HandleGetUpload != nil {
		api.HandleFunc("/uploads/{id}", deps.HandleGetUpload).Methods(http.MethodGet)
	}

	return r
}

func applyAuth(mw func(http.HandlerFunc) http.HandlerFunc, h http.HandlerFunc) http.HandlerFunc {
	if h == nil {
		return nil
	}
	if mw == nil {
		return h
	}
	return mw(h)
}
