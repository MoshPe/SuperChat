package httpapi

import (
	"SuperChat/internal/model"
	"SuperChat/internal/service"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type fakeTeamCoreService struct {
	createTeamFn       func(userID string, req model.CreateTeamRequest) (*model.Team, error)
	listTeamsFn        func(userID string) ([]*model.Team, error)
	getTeamFn          func(teamID, userID string) (*model.Team, error)
	updateTeamFn       func(teamID, userID string, input service.UpdateTeamInput) (*model.Team, error)
	deleteTeamFn       func(teamID, userID string) error
	getTeamMembersFn   func(teamID, userID string) ([]service.TeamMemberView, error)
	addTeamMemberFn    func(teamID, userID, targetUsername string) (string, error)
	removeTeamMemberFn func(teamID, userID, targetUserID string) error
	joinTeamFn         func(teamID, userID string) (*model.Team, error)
	leaveTeamFn        func(teamID, userID string) error
	transferOwnerFn    func(teamID, userID, newOwnerID string) (*model.Team, error)
}

func (f *fakeTeamCoreService) CreateTeam(userID string, req model.CreateTeamRequest) (*model.Team, error) {
	return f.createTeamFn(userID, req)
}
func (f *fakeTeamCoreService) ListTeams(userID string) ([]*model.Team, error) {
	return f.listTeamsFn(userID)
}
func (f *fakeTeamCoreService) GetTeam(teamID, userID string) (*model.Team, error) {
	return f.getTeamFn(teamID, userID)
}
func (f *fakeTeamCoreService) UpdateTeam(teamID, userID string, input service.UpdateTeamInput) (*model.Team, error) {
	return f.updateTeamFn(teamID, userID, input)
}
func (f *fakeTeamCoreService) DeleteTeam(teamID, userID string) error {
	return f.deleteTeamFn(teamID, userID)
}
func (f *fakeTeamCoreService) GetTeamMembers(teamID, userID string) ([]service.TeamMemberView, error) {
	return f.getTeamMembersFn(teamID, userID)
}
func (f *fakeTeamCoreService) AddTeamMember(teamID, userID, targetUsername string) (string, error) {
	return f.addTeamMemberFn(teamID, userID, targetUsername)
}
func (f *fakeTeamCoreService) RemoveTeamMember(teamID, userID, targetUserID string) error {
	return f.removeTeamMemberFn(teamID, userID, targetUserID)
}
func (f *fakeTeamCoreService) JoinTeam(teamID, userID string) (*model.Team, error) {
	return f.joinTeamFn(teamID, userID)
}
func (f *fakeTeamCoreService) LeaveTeam(teamID, userID string) error {
	return f.leaveTeamFn(teamID, userID)
}
func (f *fakeTeamCoreService) TransferOwnership(teamID, userID, newOwnerID string) (*model.Team, error) {
	return f.transferOwnerFn(teamID, userID, newOwnerID)
}

func withHTTPAPIAuth(r *http.Request, userID, username string) *http.Request {
	ctx := context.WithValue(r.Context(), "user_id", userID)
	ctx = context.WithValue(ctx, "username", username)
	return r.WithContext(ctx)
}

func decodeAPIResponse(t *testing.T, rr *httptest.ResponseRecorder) model.APIResponse {
	t.Helper()
	var resp model.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	return resp
}

func TestHandleCreateTeam_MapsValidationError(t *testing.T) {
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			createTeamFn: func(userID string, req model.CreateTeamRequest) (*model.Team, error) {
				return nil, &service.ValidationError{Message: "Team name is required"}
			},
		},
	})
	req := withHTTPAPIAuth(httptest.NewRequest(http.MethodPost, "/api/teams", strings.NewReader(`{"name":""}`)), "u1", "alice")
	rr := httptest.NewRecorder()

	s.HandleCreateTeam(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	resp := decodeAPIResponse(t, rr)
	if resp.Error != "Team name is required" {
		t.Fatalf("unexpected error: %#v", resp)
	}
}

func TestHandleGetTeam_MapsForbidden(t *testing.T) {
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			getTeamFn: func(teamID, userID string) (*model.Team, error) {
				return nil, service.ErrForbidden
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/teams/t1", nil)
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleGetTeam(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	resp := decodeAPIResponse(t, rr)
	if resp.Error != "Access denied" {
		t.Fatalf("unexpected error: %#v", resp)
	}
}

func TestHandleUpdateTeam_Success(t *testing.T) {
	team := model.NewTeam("T", "", "u1")
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			updateTeamFn: func(teamID, userID string, input service.UpdateTeamInput) (*model.Team, error) {
				team.Name = "Updated"
				return team, nil
			},
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/teams/"+team.ID, strings.NewReader(`{"name":"Updated"}`))
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": team.ID})
	rr := httptest.NewRecorder()

	s.HandleUpdateTeam(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeAPIResponse(t, rr)
	if !resp.Success || resp.Message != "Team updated successfully" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestHandleDeleteTeam_Success(t *testing.T) {
	var deletedID string
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			deleteTeamFn: func(teamID, userID string) error {
				deletedID = teamID
				return nil
			},
		},
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/teams/t1", nil)
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleDeleteTeam(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if deletedID != "t1" {
		t.Fatalf("expected delete called for t1, got %q", deletedID)
	}
}

func TestHandleGetTeamMembers_MapsForbidden(t *testing.T) {
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			getTeamMembersFn: func(teamID, userID string) ([]service.TeamMemberView, error) {
				return nil, service.ErrForbidden
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/teams/t1/members", nil)
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleGetTeamMembers(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeAPIResponse(t, rr)
	if resp.Error != "Access denied" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestHandleAddTeamMember_MapsConflictStyleValidation(t *testing.T) {
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			addTeamMemberFn: func(teamID, userID, targetUsername string) (string, error) {
				return "", &service.ValidationError{Message: "User already a member"}
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/teams/t1/members", strings.NewReader(`{"username":"bob"}`))
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleAddTeamMember(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleJoinTeam_MapsNotFound(t *testing.T) {
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			joinTeamFn: func(teamID, userID string) (*model.Team, error) { return nil, service.ErrNotFound },
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/teams/t1/join", nil)
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleJoinTeam(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLeaveTeam_MapsOwnerRestriction(t *testing.T) {
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			leaveTeamFn: func(teamID, userID string) error {
				return &service.ValidationError{Message: "Team owner cannot leave. Transfer ownership first."}
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/teams/t1/leave", nil)
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleLeaveTeam(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleTransferOwnership_Success(t *testing.T) {
	team := model.NewTeam("T", "", "u2")
	s := NewServer(HandlerDeps{
		TeamCore: &fakeTeamCoreService{
			transferOwnerFn: func(teamID, userID, newOwnerID string) (*model.Team, error) {
				return team, nil
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/teams/t1/transfer-ownership", strings.NewReader(`{"user_id":"u2"}`))
	req = mux.SetURLVars(withHTTPAPIAuth(req, "u1", "alice"), map[string]string{"id": "t1"})
	rr := httptest.NewRecorder()

	s.HandleTransferOwnership(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeAPIResponse(t, rr)
	if resp.Message != "Ownership transferred successfully" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
