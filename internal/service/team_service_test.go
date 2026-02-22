package service

import (
	"SuperChat/internal/model"
	"SuperChat/internal/store"
	"errors"
	"testing"
)

type fakeTeamStore struct {
	createTeamFn   func(*model.Team) error
	getTeamByIDFn  func(string) (*model.Team, error)
	getUserTeamsFn func(string) ([]*model.Team, error)
	updateTeamFn   func(*model.Team) error
	deleteTeamFn   func(string) error
}

func (f *fakeTeamStore) CreateTeam(team *model.Team) error {
	if f.createTeamFn != nil {
		return f.createTeamFn(team)
	}
	return nil
}

func (f *fakeTeamStore) GetTeamByID(teamID string) (*model.Team, error) {
	if f.getTeamByIDFn != nil {
		return f.getTeamByIDFn(teamID)
	}
	return nil, errors.New("team not found")
}

func (f *fakeTeamStore) GetUserTeams(userID string) ([]*model.Team, error) {
	if f.getUserTeamsFn != nil {
		return f.getUserTeamsFn(userID)
	}
	return nil, nil
}

func (f *fakeTeamStore) UpdateTeam(team *model.Team) error {
	if f.updateTeamFn != nil {
		return f.updateTeamFn(team)
	}
	return nil
}

func (f *fakeTeamStore) DeleteTeam(teamID string) error {
	if f.deleteTeamFn != nil {
		return f.deleteTeamFn(teamID)
	}
	return nil
}

func TestTeamServiceCreateTeam_ValidatesNameAndAddsOwnerMembership(t *testing.T) {
	var created *model.Team
	var addedMember *model.TeamMember

	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			createTeamFn: func(team *model.Team) error {
				created = team
				return nil
			},
		},
		AddTeamMember: func(member *model.TeamMember) error {
			addedMember = member
			return nil
		},
	})

	if _, err := svc.CreateTeam("u1", model.CreateTeamRequest{}); err == nil {
		t.Fatal("expected validation error for missing team name")
	}

	got, err := svc.CreateTeam("u1", model.CreateTeamRequest{Name: "Team", Description: "desc"})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if got == nil || created == nil {
		t.Fatal("expected created team")
	}
	if addedMember == nil || addedMember.TeamID != got.ID || addedMember.UserID != "u1" || addedMember.Role != "owner" {
		t.Fatalf("unexpected owner membership: %#v", addedMember)
	}
}

func TestTeamServiceGetTeam_EnforcesMembership(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
		},
		IsTeamMember: func(teamID, userID string) bool { return false },
	})

	_, err := svc.GetTeam(team.ID, "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTeamServiceUpdateTeam_RejectsNonOwner(t *testing.T) {
	svc := NewTeamService(TeamServiceDeps{
		Teams:       &fakeTeamStore{},
		IsTeamOwner: func(teamID, userID string) bool { return false },
	})

	_, err := svc.UpdateTeam("t1", "u1", UpdateTeamInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTeamServiceUpdateTeam_UpdatesFields(t *testing.T) {
	team := model.NewTeam("Old", "old", "u1")
	team.Avatar = "/a"
	var updated *model.Team
	newName := "  New Name  "
	newDesc := "  desc  "
	newAvatar := "  /new  "

	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
			updateTeamFn: func(t *model.Team) error {
				updated = t
				return nil
			},
		},
		IsTeamOwner: func(teamID, userID string) bool { return true },
	})

	got, err := svc.UpdateTeam(team.ID, "u1", UpdateTeamInput{
		Name:        &newName,
		Description: &newDesc,
		Avatar:      &newAvatar,
	})
	if err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	if got.Name != "New Name" || got.Description != "desc" || got.Avatar != "/new" {
		t.Fatalf("unexpected updated values: %#v", got)
	}
	if updated == nil {
		t.Fatal("expected UpdateTeam store call")
	}
}

func TestTeamServiceDeleteTeam_RequiresOwner(t *testing.T) {
	var deletedID string
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			deleteTeamFn: func(teamID string) error {
				deletedID = teamID
				return nil
			},
		},
		IsTeamOwner: func(teamID, userID string) bool { return false },
	})

	err := svc.DeleteTeam("t1", "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if deletedID != "" {
		t.Fatalf("delete should not be called, got %s", deletedID)
	}
}

func TestTeamServiceGetAndListTeams_HappyPaths(t *testing.T) {
	team := model.NewTeam("T", "", "u1")
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
			getUserTeamsFn: func(userID string) ([]*model.Team, error) {
				return []*model.Team{team}, nil
			},
			deleteTeamFn: func(teamID string) error { return nil },
		},
		IsTeamMember: func(teamID, userID string) bool { return true },
		IsTeamOwner:  func(teamID, userID string) bool { return true },
	})

	list, err := svc.ListTeams("u1")
	if err != nil || len(list) != 1 || list[0].ID != team.ID {
		t.Fatalf("ListTeams unexpected result: list=%#v err=%v", list, err)
	}
	got, err := svc.GetTeam(team.ID, "u1")
	if err != nil || got.ID != team.ID {
		t.Fatalf("GetTeam unexpected result: team=%#v err=%v", got, err)
	}
	if err := svc.DeleteTeam(team.ID, "u1"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
}

func TestTeamServiceGetTeam_MapsStoreNotFound(t *testing.T) {
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			getTeamByIDFn: func(id string) (*model.Team, error) { return nil, store.ErrTeamNotFound },
		},
		IsTeamMember: func(teamID, userID string) bool { return true },
	})

	_, err := svc.GetTeam("missing", "u1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
