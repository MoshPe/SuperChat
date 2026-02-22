package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthHandleRegister_InvalidJSON(t *testing.T) {
	s := NewServer(HandlerDeps{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader("{"))
	rr := httptest.NewRecorder()

	s.handleRegister(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"error":"Invalid request body"`) {
		t.Fatalf("expected invalid body error, got %s", rr.Body.String())
	}
}

func TestAuthHandleRegister_MissingFields(t *testing.T) {
	s := NewServer(HandlerDeps{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"","name":"","password":""}`))
	rr := httptest.NewRecorder()

	s.handleRegister(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"error":"All fields are required"`) {
		t.Fatalf("expected missing fields error, got %s", rr.Body.String())
	}
}

func TestAuthHandleLogin_MissingFields(t *testing.T) {
	s := NewServer(HandlerDeps{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"","password":""}`))
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"error":"Username and password are required"`) {
		t.Fatalf("expected missing fields error, got %s", rr.Body.String())
	}
}

func TestAuthHandleLogout_Success(t *testing.T) {
	s := NewServer(HandlerDeps{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rr := httptest.NewRecorder()

	s.handleLogout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"message":"Logout successful"`) {
		t.Fatalf("expected logout success response, got %s", rr.Body.String())
	}
}
