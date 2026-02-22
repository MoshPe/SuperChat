package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteSuccessResponseShape(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteSuccess(rr, map[string]string{"x": "y"}, "ok")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["success"] != true {
		t.Fatalf("expected success=true, got %#v", body["success"])
	}
	if body["message"] != "ok" {
		t.Fatalf("expected message=ok, got %#v", body["message"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", body["data"])
	}
	if data["x"] != "y" {
		t.Fatalf("expected data.x=y, got %#v", data["x"])
	}
}

func TestWriteErrorResponseShape(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteError(rr, http.StatusBadRequest, "bad request")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["success"] != false {
		t.Fatalf("expected success=false, got %#v", body["success"])
	}
	if body["error"] != "bad request" {
		t.Fatalf("expected error field, got %#v", body["error"])
	}
}
