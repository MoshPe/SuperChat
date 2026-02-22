package service

import (
	"SuperChat/internal/model"
	"SuperChat/internal/store"
	"errors"
	"strings"
	"time"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func newValidationError(message string) error {
	return &ValidationError{Message: message}
}

type TeamServiceDeps struct {
	Teams         store.TeamStore
	IsTeamMember  func(teamID, userID string) bool
	IsTeamOwner   func(teamID, userID string) bool
	AddTeamMember func(*model.TeamMember) error
}

type TeamService struct {
	deps TeamServiceDeps
}

type UpdateTeamInput struct {
	Name        *string
	Description *string
	Avatar      *string
}

func NewTeamService(deps TeamServiceDeps) *TeamService {
	return &TeamService{deps: deps}
}

func (s *TeamService) CreateTeam(userID string, req model.CreateTeamRequest) (*model.Team, error) {
	if s.deps.Teams == nil {
		return nil, errors.New("team store not configured")
	}
	if req.Name == "" {
		return nil, newValidationError("Team name is required")
	}

	team := model.NewTeam(req.Name, req.Description, userID)
	if err := s.deps.Teams.CreateTeam(team); err != nil {
		return nil, err
	}

	if s.deps.AddTeamMember != nil {
		member := model.NewTeamMember(team.ID, userID, "owner")
		if err := s.deps.AddTeamMember(member); err != nil {
			return nil, err
		}
	}

	return team, nil
}

func (s *TeamService) ListTeams(userID string) ([]*model.Team, error) {
	if s.deps.Teams == nil {
		return nil, errors.New("team store not configured")
	}
	return s.deps.Teams.GetUserTeams(userID)
}

func (s *TeamService) GetTeam(teamID, userID string) (*model.Team, error) {
	if s.deps.IsTeamMember != nil && !s.deps.IsTeamMember(teamID, userID) {
		return nil, ErrForbidden
	}
	team, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrNotFound
	}
	return team, err
}

func (s *TeamService) UpdateTeam(teamID, userID string, input UpdateTeamInput) (*model.Team, error) {
	if s.deps.IsTeamOwner != nil && !s.deps.IsTeamOwner(teamID, userID) {
		return nil, ErrForbidden
	}

	team, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, newValidationError("Team name cannot be empty")
		}
		team.Name = name
	}
	if input.Description != nil {
		team.Description = strings.TrimSpace(*input.Description)
	}
	if input.Avatar != nil {
		team.Avatar = strings.TrimSpace(*input.Avatar)
	}
	team.UpdatedAt = time.Now()

	if err := s.deps.Teams.UpdateTeam(team); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TeamService) DeleteTeam(teamID, userID string) error {
	if s.deps.IsTeamOwner != nil && !s.deps.IsTeamOwner(teamID, userID) {
		return ErrForbidden
	}
	return s.deps.Teams.DeleteTeam(teamID)
}
