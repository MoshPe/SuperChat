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
	Teams store.TeamStore
	Users store.UserStore
}

type TeamService struct {
	deps TeamServiceDeps
}

type UpdateTeamInput struct {
	Name        *string
	Description *string
	Avatar      *string
}

type TeamMemberView struct {
	ID       string    `json:"id"`
	TeamID   string    `json:"team_id"`
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Name     string    `json:"name,omitempty"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
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
	if err := s.deps.Teams.AddTeamMember(model.NewTeamMember(team.ID, userID, "owner")); err != nil {
		return nil, err
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
	isMember, err := s.deps.Teams.IsTeamMember(teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}
	team, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrNotFound
	}
	return team, err
}

func (s *TeamService) UpdateTeam(teamID, userID string, input UpdateTeamInput) (*model.Team, error) {
	team, ownerErr := s.getOwnedTeamOrForbidden(teamID, userID)
	if ownerErr != nil {
		return nil, ownerErr
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
	if _, err := s.getOwnedTeamOrForbidden(teamID, userID); err != nil {
		return err
	}
	return s.deps.Teams.DeleteTeam(teamID)
}

func (s *TeamService) GetTeamMembers(teamID, requesterID string) ([]TeamMemberView, error) {
	isMember, err := s.deps.Teams.IsTeamMember(teamID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}

	members, err := s.deps.Teams.GetTeamMembers(teamID)
	if err != nil {
		return nil, err
	}
	result := make([]TeamMemberView, 0, len(members))
	for _, m := range members {
		username := ""
		name := ""
		if s.deps.Users != nil {
			if u, err := s.deps.Users.GetUserByID(m.UserID); err == nil && u != nil {
				username = u.Username
				name = strings.TrimSpace(u.Name)
			}
		}
		result = append(result, TeamMemberView{
			ID:       m.ID,
			TeamID:   m.TeamID,
			UserID:   m.UserID,
			Username: username,
			Name:     name,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		})
	}
	return result, nil
}

func (s *TeamService) AddTeamMember(teamID, requesterID, targetUsername string) (string, error) {
	if _, err := s.getOwnedTeamOrForbidden(teamID, requesterID); err != nil {
		return "", err
	}

	targetUsername = strings.TrimSpace(targetUsername)
	if targetUsername == "" {
		return "", newValidationError("Username is required")
	}

	target, err := s.deps.Users.GetUserByUsername(targetUsername)
	if errors.Is(err, store.ErrUserNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	alreadyMember, err := s.deps.Teams.IsTeamMember(teamID, target.ID)
	if err != nil {
		return "", err
	}
	if alreadyMember {
		return "", newValidationError("User already a member")
	}

	member := model.NewTeamMember(teamID, target.ID, "member")
	if err := s.deps.Teams.AddTeamMember(member); err != nil {
		return "", err
	}
	return target.ID, nil
}

func (s *TeamService) RemoveTeamMember(teamID, requesterID, targetUserID string) error {
	team, err := s.getOwnedTeamOrForbidden(teamID, requesterID)
	if err != nil {
		return err
	}

	if team.OwnerID == targetUserID {
		return newValidationError("Cannot remove team owner")
	}

	isMember, err := s.deps.Teams.IsTeamMember(teamID, targetUserID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotFound
	}

	return s.deps.Teams.RemoveTeamMember(teamID, targetUserID)
}

func (s *TeamService) JoinTeam(teamID, requesterID string) (*model.Team, error) {
	team, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	alreadyMember, err := s.deps.Teams.IsTeamMember(teamID, requesterID)
	if err != nil {
		return nil, err
	}
	if alreadyMember {
		return nil, newValidationError("Already a member of this team")
	}

	if err := s.deps.Teams.AddTeamMember(model.NewTeamMember(teamID, requesterID, "member")); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TeamService) LeaveTeam(teamID, requesterID string) error {
	team, err := s.deps.Teams.GetTeamByID(teamID)
	if err == nil && team != nil && team.OwnerID == requesterID {
		return newValidationError("Team owner cannot leave. Transfer ownership first.")
	}
	if err != nil && !errors.Is(err, store.ErrTeamNotFound) {
		return err
	}

	isMember, err := s.deps.Teams.IsTeamMember(teamID, requesterID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrForbidden
	}

	return s.deps.Teams.RemoveTeamMember(teamID, requesterID)
}

func (s *TeamService) TransferOwnership(teamID, requesterID, newOwnerID string) (*model.Team, error) {
	team, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if team.OwnerID != requesterID {
		return nil, ErrForbidden
	}

	newOwnerID = strings.TrimSpace(newOwnerID)
	if newOwnerID == "" {
		return nil, newValidationError("Target user is required")
	}
	if newOwnerID == requesterID {
		return nil, newValidationError("Cannot transfer ownership to yourself")
	}

	isMember, err := s.deps.Teams.IsTeamMember(teamID, newOwnerID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, newValidationError("New owner must be an existing team member")
	}

	if err := s.deps.Teams.TransferTeamOwnership(teamID, requesterID, newOwnerID); err != nil {
		return nil, err
	}

	updated, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrNotFound
	}
	return updated, err
}

func (s *TeamService) getOwnedTeamOrForbidden(teamID, userID string) (*model.Team, error) {
	team, err := s.deps.Teams.GetTeamByID(teamID)
	if errors.Is(err, store.ErrTeamNotFound) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	if team.OwnerID != userID {
		return nil, ErrForbidden
	}
	return team, nil
}
