package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func utf8AsLatin1Mojibake(s string) string {
	b := []byte(s)
	runes := make([]rune, len(b))
	for i, v := range b {
		runes[i] = rune(v)
	}
	return string(runes)
}

func uploadAuthReq(req *http.Request, userID, username string) *http.Request {
	ctx := context.WithValue(req.Context(), "user_id", userID)
	ctx = context.WithValue(ctx, "username", username)
	return req.WithContext(ctx)
}

func makeUploadRequest(t *testing.T, fieldName, filename string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func withUploadLimitBytes(t *testing.T, n int64, fn func()) {
	t.Helper()
	orig := maxUploadBytes
	maxUploadBytes = n
	t.Cleanup(func() { maxUploadBytes = orig })
	fn()
}

func TestHandleUpload_AcceptsNonImageFileTypes(t *testing.T) {
	withTempDB(t, func() {
		withUploadLimitBytes(t, 2<<20, func() {
			pdfBytes := []byte("%PDF-1.4\n%test pdf content\n")
			req := makeUploadRequest(t, "file", "doc.pdf", pdfBytes)
			req = uploadAuthReq(req, "u1", "alice")
			rr := httptest.NewRecorder()

			handleUpload(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
			}

			var resp APIResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			data, ok := resp.Data.(map[string]interface{})
			if !ok {
				t.Fatalf("expected response data map, got %#v", resp.Data)
			}
			if _, ok := data["url"].(string); !ok {
				t.Fatalf("expected upload url, got %#v", data)
			}
		})
	})
}

func TestHandleUpload_UsesConfiguredMaxUploadBytes(t *testing.T) {
	withTempDB(t, func() {
		// Very small limit so multipart payload exceeds it.
		withUploadLimitBytes(t, 64, func() {
			req := makeUploadRequest(t, "file", "big.bin", bytes.Repeat([]byte("a"), 1024))
			req = uploadAuthReq(req, "u1", "alice")
			rr := httptest.NewRecorder()

			handleUpload(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
			}
			var resp APIResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if resp.Error != "File too large or invalid form" {
				t.Fatalf("unexpected error message: %#v", resp)
			}
		})
	})
}

func TestNormalizeUploadFilename_DecodesHebrewMojibake(t *testing.T) {
	want := "מסמך טקסט חדש.txt"
	got := normalizeUploadFilename(utf8AsLatin1Mojibake(want))
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
