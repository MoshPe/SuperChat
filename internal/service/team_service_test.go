package service

import (
	"SuperChat/internal/model"
	"SuperChat/internal/store"
	"errors"
	"testing"
)

type fakeTeamStore struct {
	createTeamFn            func(*model.Team) error
	getTeamByIDFn           func(string) (*model.Team, error)
	getUserTeamsFn          func(string) ([]*model.Team, error)
	updateTeamFn            func(*model.Team) error
	deleteTeamFn            func(string) error
	addTeamMemberFn         func(*model.TeamMember) error
	removeTeamMemberFn      func(string, string) error
	getTeamMembersFn        func(string) ([]*model.TeamMember, error)
	isTeamMemberFn          func(string, string) (bool, error)
	transferTeamOwnershipFn func(string, string, string) error
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
	return nil, store.ErrTeamNotFound
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

func (f *fakeTeamStore) AddTeamMember(member *model.TeamMember) error {
	if f.addTeamMemberFn != nil {
		return f.addTeamMemberFn(member)
	}
	return nil
}

func (f *fakeTeamStore) RemoveTeamMember(teamID, userID string) error {
	if f.removeTeamMemberFn != nil {
		return f.removeTeamMemberFn(teamID, userID)
	}
	return nil
}

func (f *fakeTeamStore) GetTeamMembers(teamID string) ([]*model.TeamMember, error) {
	if f.getTeamMembersFn != nil {
		return f.getTeamMembersFn(teamID)
	}
	return nil, nil
}

func (f *fakeTeamStore) IsTeamMember(teamID, userID string) (bool, error) {
	if f.isTeamMemberFn != nil {
		return f.isTeamMemberFn(teamID, userID)
	}
	return false, nil
}

func (f *fakeTeamStore) TransferTeamOwnership(teamID, currentOwnerID, newOwnerID string) error {
	if f.transferTeamOwnershipFn != nil {
		return f.transferTeamOwnershipFn(teamID, currentOwnerID, newOwnerID)
	}
	return nil
}

type fakeUserStore struct {
	getUserByUsernameFn func(string) (*model.User, error)
	getUserByIDFn       func(string) (*model.User, error)
}

func (f *fakeUserStore) CreateUser(user *model.User) error { return nil }

func (f *fakeUserStore) GetUserByUsername(username string) (*model.User, error) {
	if f.getUserByUsernameFn != nil {
		return f.getUserByUsernameFn(username)
	}
	return nil, store.ErrUserNotFound
}

func (f *fakeUserStore) GetUserByID(userID string) (*model.User, error) {
	if f.getUserByIDFn != nil {
		return f.getUserByIDFn(userID)
	}
	return nil, store.ErrUserNotFound
}

func validationMsg(err error) string {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Message
	}
	return ""
}

func TestTeamServiceCreateTeam_ValidatesNameAndAddsOwnerMembership(t *testing.T) {
	var created *model.Team
	var addedMember *model.TeamMember
	teams := &fakeTeamStore{
		createTeamFn: func(team *model.Team) error {
			created = team
			return nil
		},
		addTeamMemberFn: func(member *model.TeamMember) error {
			addedMember = member
			return nil
		},
	}

	svc := NewTeamService(TeamServiceDeps{Teams: teams, Users: &fakeUserStore{}})

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
			getTeamByIDFn:  func(id string) (*model.Team, error) { return team, nil },
			isTeamMemberFn: func(teamID, userID string) (bool, error) { return false, nil },
		},
		Users: &fakeUserStore{},
	})

	_, err := svc.GetTeam(team.ID, "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTeamServiceUpdateTeam_RejectsNonOwner(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil }},
		Users: &fakeUserStore{},
	})

	_, err := svc.UpdateTeam(team.ID, "u1", UpdateTeamInput{})
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
		Users: &fakeUserStore{},
	})

	got, err := svc.UpdateTeam(team.ID, "u1", UpdateTeamInput{Name: &newName, Description: &newDesc, Avatar: &newAvatar})
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
	team := model.NewTeam("T", "", "owner")
	var deletedID string
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
			deleteTeamFn: func(teamID string) error {
				deletedID = teamID
				return nil
			},
		},
		Users: &fakeUserStore{},
	})

	err := svc.DeleteTeam(team.ID, "u1")
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
			getTeamByIDFn:  func(id string) (*model.Team, error) { return team, nil },
			getUserTeamsFn: func(userID string) ([]*model.Team, error) { return []*model.Team{team}, nil },
			isTeamMemberFn: func(teamID, userID string) (bool, error) { return true, nil },
			deleteTeamFn:   func(teamID string) error { return nil },
		},
		Users: &fakeUserStore{},
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
			getTeamByIDFn:  func(id string) (*model.Team, error) { return nil, store.ErrTeamNotFound },
			isTeamMemberFn: func(teamID, userID string) (bool, error) { return true, nil },
		},
		Users: &fakeUserStore{},
	})

	_, err := svc.GetTeam("missing", "u1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTeamServiceGetTeamMembers_RequiresMembership(t *testing.T) {
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{isTeamMemberFn: func(teamID, userID string) (bool, error) { return false, nil }},
		Users: &fakeUserStore{},
	})

	_, err := svc.GetTeamMembers("t1", "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTeamServiceGetTeamMembers_EnrichesUsernames(t *testing.T) {
	members := []*model.TeamMember{{ID: "m1", TeamID: "t1", UserID: "u2", Role: "member"}}
	svc := NewTeamService(TeamServiceDeps{
		Teams: &fakeTeamStore{
			isTeamMemberFn:   func(teamID, userID string) (bool, error) { return true, nil },
			getTeamMembersFn: func(teamID string) ([]*model.TeamMember, error) { return members, nil },
		},
		Users: &fakeUserStore{getUserByIDFn: func(id string) (*model.User, error) {
			return &model.User{ID: id, Username: "bob", Name: "Bob Display"}, nil
		}},
	})

	got, err := svc.GetTeamMembers("t1", "u1")
	if err != nil {
		t.Fatalf("GetTeamMembers: %v", err)
	}
	if len(got) != 1 || got[0].Username != "bob" || got[0].Name != "Bob Display" || got[0].UserID != "u2" {
		t.Fatalf("unexpected members result: %#v", got)
	}
}

func TestTeamServiceAddTeamMember_RulesAndSuccess(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	var added *model.TeamMember
	teams := &fakeTeamStore{
		getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
		isTeamMemberFn: func(teamID, userID string) (bool, error) {
			return userID == "owner", nil
		},
		addTeamMemberFn: func(m *model.TeamMember) error { added = m; return nil },
	}
	users := &fakeUserStore{getUserByUsernameFn: func(username string) (*model.User, error) {
		return &model.User{ID: "u2", Username: username}, nil
	}}
	svc := NewTeamService(TeamServiceDeps{Teams: teams, Users: users})

	if _, err := svc.AddTeamMember(team.ID, "not-owner", "bob"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if _, err := svc.AddTeamMember(team.ID, "owner", ""); validationMsg(err) != "Username is required" {
		t.Fatalf("expected username validation, got %v", err)
	}

	// Already member case
	teams.isTeamMemberFn = func(teamID, userID string) (bool, error) {
		if userID == "owner" || userID == "u2" {
			return true, nil
		}
		return false, nil
	}
	if _, err := svc.AddTeamMember(team.ID, "owner", "bob"); validationMsg(err) != "User already a member" {
		t.Fatalf("expected already-member validation, got %v", err)
	}

	teams.isTeamMemberFn = func(teamID, userID string) (bool, error) { return userID == "owner", nil }
	userID, err := svc.AddTeamMember(team.ID, "owner", "bob")
	if err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	if userID != "u2" || added == nil || added.UserID != "u2" || added.Role != "member" {
		t.Fatalf("unexpected add result userID=%q added=%#v", userID, added)
	}
}

func TestTeamServiceRemoveTeamMember_RulesAndSuccess(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	var removedTeamID, removedUserID string
	teams := &fakeTeamStore{
		getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
		isTeamMemberFn: func(teamID, userID string) (bool, error) {
			if userID == "owner" || userID == "u2" {
				return true, nil
			}
			return false, nil
		},
		removeTeamMemberFn: func(teamID, userID string) error { removedTeamID, removedUserID = teamID, userID; return nil },
	}
	svc := NewTeamService(TeamServiceDeps{Teams: teams, Users: &fakeUserStore{}})

	if err := svc.RemoveTeamMember(team.ID, "not-owner", "u2"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if err := svc.RemoveTeamMember(team.ID, "owner", "owner"); validationMsg(err) != "Cannot remove team owner" {
		t.Fatalf("expected owner-protect validation, got %v", err)
	}
	if err := svc.RemoveTeamMember(team.ID, "owner", "u3"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
	if err := svc.RemoveTeamMember(team.ID, "owner", "u2"); err != nil {
		t.Fatalf("RemoveTeamMember: %v", err)
	}
	if removedTeamID != team.ID || removedUserID != "u2" {
		t.Fatalf("unexpected remove args: %s %s", removedTeamID, removedUserID)
	}
}

func TestTeamServiceJoinTeam_RulesAndSuccess(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	var added *model.TeamMember
	teams := &fakeTeamStore{
		getTeamByIDFn:   func(id string) (*model.Team, error) { return team, nil },
		isTeamMemberFn:  func(teamID, userID string) (bool, error) { return userID == "owner", nil },
		addTeamMemberFn: func(m *model.TeamMember) error { added = m; return nil },
	}
	svc := NewTeamService(TeamServiceDeps{Teams: teams, Users: &fakeUserStore{}})

	if _, err := svc.JoinTeam(team.ID, "owner"); validationMsg(err) != "Already a member of this team" {
		t.Fatalf("expected already-member validation, got %v", err)
	}
	joined, err := svc.JoinTeam(team.ID, "u2")
	if err != nil {
		t.Fatalf("JoinTeam: %v", err)
	}
	if joined.ID != team.ID || added == nil || added.UserID != "u2" || added.Role != "member" {
		t.Fatalf("unexpected join result: team=%#v added=%#v", joined, added)
	}
}

func TestTeamServiceLeaveTeam_RulesAndSuccess(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	var removed string
	teams := &fakeTeamStore{
		getTeamByIDFn: func(id string) (*model.Team, error) { return team, nil },
		isTeamMemberFn: func(teamID, userID string) (bool, error) {
			return userID == "owner" || userID == "u2", nil
		},
		removeTeamMemberFn: func(teamID, userID string) error { removed = userID; return nil },
	}
	svc := NewTeamService(TeamServiceDeps{Teams: teams, Users: &fakeUserStore{}})

	if err := svc.LeaveTeam(team.ID, "owner"); validationMsg(err) != "Team owner cannot leave. Transfer ownership first." {
		t.Fatalf("expected owner leave validation, got %v", err)
	}
	if err := svc.LeaveTeam(team.ID, "u3"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-member leave, got %v", err)
	}
	if err := svc.LeaveTeam(team.ID, "u2"); err != nil {
		t.Fatalf("LeaveTeam: %v", err)
	}
	if removed != "u2" {
		t.Fatalf("expected removed user u2, got %s", removed)
	}
}

func TestTeamServiceTransferOwnership_RulesAndSuccess(t *testing.T) {
	team := model.NewTeam("T", "", "owner")
	var transferred bool
	teams := &fakeTeamStore{
		getTeamByIDFn: func(id string) (*model.Team, error) {
			if transferred {
				return &model.Team{ID: team.ID, OwnerID: "u2", Name: team.Name}, nil
			}
			return team, nil
		},
		isTeamMemberFn: func(teamID, userID string) (bool, error) {
			return userID == "owner" || userID == "u2", nil
		},
		transferTeamOwnershipFn: func(teamID, currentOwnerID, newOwnerID string) error {
			transferred = true
			return nil
		},
	}
	svc := NewTeamService(TeamServiceDeps{Teams: teams, Users: &fakeUserStore{}})

	if _, err := svc.TransferOwnership(team.ID, "not-owner", "u2"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if _, err := svc.TransferOwnership(team.ID, "owner", ""); validationMsg(err) != "Target user is required" {
		t.Fatalf("expected target-required validation, got %v", err)
	}
	if _, err := svc.TransferOwnership(team.ID, "owner", "owner"); validationMsg(err) != "Cannot transfer ownership to yourself" {
		t.Fatalf("expected self-transfer validation, got %v", err)
	}
	updated, err := svc.TransferOwnership(team.ID, "owner", "u2")
	if err != nil {
		t.Fatalf("TransferOwnership: %v", err)
	}
	if !transferred || updated.OwnerID != "u2" {
		t.Fatalf("unexpected transfer result: transferred=%v updated=%#v", transferred, updated)
	}
}
