package store

import "SuperChat/internal/model"

type UserStore interface {
	CreateUser(user *model.User) error
	GetUserByUsername(username string) (*model.User, error)
	GetUserByID(userID string) (*model.User, error)
}

type TeamStore interface {
	CreateTeam(team *model.Team) error
	GetTeamByID(teamID string) (*model.Team, error)
	GetUserTeams(userID string) ([]*model.Team, error)
	UpdateTeam(team *model.Team) error
	DeleteTeam(teamID string) error
	AddTeamMember(member *model.TeamMember) error
	RemoveTeamMember(teamID, userID string) error
	GetTeamMembers(teamID string) ([]*model.TeamMember, error)
	IsTeamMember(teamID, userID string) (bool, error)
	TransferTeamOwnership(teamID, currentOwnerID, newOwnerID string) error
}
