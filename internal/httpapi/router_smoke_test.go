package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterRegistersAuthLoginRoute(t *testing.T) {
	r := NewRouter(HandlerDeps{
		HandleRegister: func(w http.ResponseWriter, r *http.Request) {},
		HandleLogin: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		},
		HandleLogout: func(w http.ResponseWriter, r *http.Request) {},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected auth login route handler to run, got status %d", rr.Code)
	}
}

func TestRouterRegistersAuthRegisterRoute(t *testing.T) {
	r := NewRouter(HandlerDeps{
		HandleRegister: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
		HandleLogin:  func(w http.ResponseWriter, r *http.Request) {},
		HandleLogout: func(w http.ResponseWriter, r *http.Request) {},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected auth register route handler to run, got status %d", rr.Code)
	}
}

func TestRouterRegistersAuthLogoutRoute(t *testing.T) {
	r := NewRouter(HandlerDeps{
		AuthMiddleware: func(next http.HandlerFunc) http.HandlerFunc { return next },
		HandleRegister: func(w http.ResponseWriter, r *http.Request) {},
		HandleLogin:    func(w http.ResponseWriter, r *http.Request) {},
		HandleLogout: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusResetContent)
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusResetContent {
		t.Fatalf("expected auth logout route handler to run, got status %d", rr.Code)
	}
}

func TestRouterRegistersTeamsRoute(t *testing.T) {
	r := NewRouter(HandlerDeps{
		AuthMiddleware: func(next http.HandlerFunc) http.HandlerFunc { return next },
		HandleGetTeams: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/teams", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected teams route handler to run, got status %d", rr.Code)
	}
}

func TestRouterRegistersTeamByIDRoute(t *testing.T) {
	r := NewRouter(HandlerDeps{
		AuthMiddleware: func(next http.HandlerFunc) http.HandlerFunc { return next },
		HandleGetTeam: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/teams/team-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusPartialContent {
		t.Fatalf("expected team by id route handler to run, got status %d", rr.Code)
	}
}

func TestRouterRegistersTeamMembershipRoutes(t *testing.T) {
	r := NewRouter(HandlerDeps{
		AuthMiddleware: func(next http.HandlerFunc) http.HandlerFunc { return next },
		HandleGetTeamMembers: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
		HandleAddTeamMember: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		},
		HandleRemoveTeamMember: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})

	req1 := httptest.NewRequest(http.MethodGet, "/api/teams/team-1/members", nil)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("expected get members route handler to run, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/teams/team-1/members", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusAccepted {
		t.Fatalf("expected add member route handler to run, got %d", rr2.Code)
	}

	req3 := httptest.NewRequest(http.MethodDelete, "/api/teams/team-1/members/u2", nil)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusNoContent {
		t.Fatalf("expected remove member route handler to run, got %d", rr3.Code)
	}
}

func TestRouterRegistersTeamJoinLeaveTransferRoutes(t *testing.T) {
	r := NewRouter(HandlerDeps{
		AuthMiddleware: func(next http.HandlerFunc) http.HandlerFunc { return next },
		HandleJoinTeam: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
		HandleLeaveTeam: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		},
		HandleTransferOwnership: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusResetContent)
		},
	})

	req1 := httptest.NewRequest(http.MethodPost, "/api/teams/team-1/join", nil)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("expected join route handler to run, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/teams/team-1/leave", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusAccepted {
		t.Fatalf("expected leave route handler to run, got %d", rr2.Code)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/teams/team-1/transfer-ownership", nil)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusResetContent {
		t.Fatalf("expected transfer route handler to run, got %d", rr3.Code)
	}
}

func TestRouterRegistersUploadDownloadRoute(t *testing.T) {
	r := NewRouter(HandlerDeps{
		HandleGetUpload: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/uploads/abc123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected upload download route handler to run, got status %d", rr.Code)
	}
}
