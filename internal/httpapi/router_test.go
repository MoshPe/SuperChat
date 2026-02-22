package httpapi

import "testing"

func TestNewRouterReturnsHandler(t *testing.T) {
	h := NewRouter(HandlerDeps{})
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}
