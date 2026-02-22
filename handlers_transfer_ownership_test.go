package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func withAuthUser(r *http.Request, userID, username string) *http.Request {
	ctx := context.WithValue(r.Context(), "user_id", userID)
	ctx = context.WithValue(ctx, "username", username)
	return r.WithContext(ctx)
}

func TestHandleTransferOwnership_UpdatesOwner(t *testing.T) {
	withTempDB(t, func() {
		owner := NewUser("owner", "Owner", "hashed")
		member := NewUser("member", "Member", "hashed")
		if err := createUser(owner); err != nil {
			t.Fatalf("create owner: %v", err)
		}
		if err := createUser(member); err != nil {
			t.Fatalf("create member: %v", err)
		}

		team := NewTeam("Team", "desc", owner.ID)
		if err := createTeam(team); err != nil {
			t.Fatalf("create team: %v", err)
		}
		if err := addTeamMember(NewTeamMember(team.ID, owner.ID, "owner")); err != nil {
			t.Fatalf("add owner member: %v", err)
		}
		if err := addTeamMember(NewTeamMember(team.ID, member.ID, "member")); err != nil {
			t.Fatalf("add target member: %v", err)
		}

		body := `{"user_id":"` + member.ID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/teams/"+team.ID+"/transfer-ownership", strings.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": team.ID})
		req = withAuthUser(req, owner.ID, owner.Username)
		rr := httptest.NewRecorder()

		handleTransferOwnership(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
		}
		updated, err := getTeamByID(team.ID)
		if err != nil {
			t.Fatalf("getTeamByID: %v", err)
		}
		if updated.OwnerID != member.ID {
			t.Fatalf("expected owner_id %s, got %s", member.ID, updated.OwnerID)
		}
	})
}

func TestHandleTransferOwnership_RejectsNonMemberTarget(t *testing.T) {
	withTempDB(t, func() {
		owner := NewUser("owner", "Owner", "hashed")
		outsider := NewUser("outsider", "Outsider", "hashed")
		_ = createUser(owner)
		_ = createUser(outsider)
		team := NewTeam("Team", "desc", owner.ID)
		_ = createTeam(team)
		_ = addTeamMember(NewTeamMember(team.ID, owner.ID, "owner"))

		body := `{"user_id":"` + outsider.ID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/teams/"+team.ID+"/transfer-ownership", strings.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": team.ID})
		req = withAuthUser(req, owner.ID, owner.Username)
		rr := httptest.NewRecorder()

		handleTransferOwnership(rr, req)

		if rr.Code == http.StatusOK {
			t.Fatalf("expected non-200 for outsider transfer, got body=%s", rr.Body.String())
		}
	})
}

