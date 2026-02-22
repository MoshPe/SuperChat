package main

import (
	"net/http/httptest"
	"testing"
)

func TestIsAllowedWebSocketOrigin_AllowsSameHostnameDifferentPort(t *testing.T) {
	orig := append([]string(nil), allowedWebSocketOrigins...)
	setAllowedWebSocketOriginsFromCSV("")
	defer func() { allowedWebSocketOrigins = orig }()

	r := httptest.NewRequest("GET", "http://localhost:8080/api/ws/team", nil)
	r.Host = "localhost:8080"
	r.Header.Set("Origin", "http://localhost:3000")

	if !isAllowedWebSocketOrigin(r) {
		t.Fatal("expected same hostname (different port) origin to be allowed")
	}
}

func TestIsAllowedWebSocketOrigin_RejectsDifferentHostname(t *testing.T) {
	orig := append([]string(nil), allowedWebSocketOrigins...)
	setAllowedWebSocketOriginsFromCSV("")
	defer func() { allowedWebSocketOrigins = orig }()

	r := httptest.NewRequest("GET", "http://localhost:8080/api/ws/team", nil)
	r.Host = "localhost:8080"
	r.Header.Set("Origin", "https://evil.example")

	if isAllowedWebSocketOrigin(r) {
		t.Fatal("expected different hostname origin to be rejected")
	}
}

func TestIsAllowedWebSocketOrigin_AllowsExplicitAllowlist(t *testing.T) {
	orig := append([]string(nil), allowedWebSocketOrigins...)
	setAllowedWebSocketOriginsFromCSV("https://chat.example.com, https://admin.example.com")
	defer func() { allowedWebSocketOrigins = orig }()

	r := httptest.NewRequest("GET", "http://localhost:8080/api/ws/team", nil)
	r.Host = "localhost:8080"
	r.Header.Set("Origin", "https://admin.example.com")

	if !isAllowedWebSocketOrigin(r) {
		t.Fatal("expected allowlisted origin to be allowed")
	}
}

func TestSetAllowedWebSocketOriginsFromCSV_TrimsAndSkipsEmpty(t *testing.T) {
	orig := append([]string(nil), allowedWebSocketOrigins...)
	defer func() { allowedWebSocketOrigins = orig }()

	setAllowedWebSocketOriginsFromCSV(" https://a.example , ,https://b.example  ")

	if len(allowedWebSocketOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(allowedWebSocketOrigins))
	}
	if allowedWebSocketOrigins[0] != "https://a.example" || allowedWebSocketOrigins[1] != "https://b.example" {
		t.Fatalf("unexpected parsed origins: %#v", allowedWebSocketOrigins)
	}
}

func TestJWTSecret_HardcodedAndNonEmpty(t *testing.T) {
	if len(jwtSecret) == 0 {
		t.Fatal("expected hardcoded jwt secret to be non-empty")
	}
}
