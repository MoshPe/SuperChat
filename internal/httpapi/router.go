package httpapi

import (
	"SuperChat/internal/model"
	"net/http"

	"github.com/gorilla/mux"
)

// HandlerDeps will hold injected dependencies as the backend refactor progresses.
type HandlerDeps struct {
	AuthMiddleware func(http.HandlerFunc) http.HandlerFunc

	HandleRegister  http.HandlerFunc
	HandleLogin     http.HandlerFunc
	HandleLogout    http.HandlerFunc
	HandleGetTeams  http.HandlerFunc
	HandleGetUpload http.HandlerFunc

	HashPassword      func(string) (string, error)
	CheckPassword     func(string, string) bool
	CreateUser        func(*model.User) error
	GetUserByUsername func(string) (*model.User, error)
	GenerateToken     func(*model.User) (string, error)
	IsUsernameExists  func(error) bool
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
	if deps.HandleGetTeams != nil {
		api.HandleFunc("/teams", applyAuth(deps.AuthMiddleware, deps.HandleGetTeams)).Methods(http.MethodGet)
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
