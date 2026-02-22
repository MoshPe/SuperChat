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
