package server

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"nx-remote-cache/internal/auth"
	"nx-remote-cache/internal/storage"
)

const (
	readToken  = "read-token"
	writeToken = "write-token"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	backend, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("new local backend: %v", err)
	}
	tokens := auth.NewTokenStore([]string{readToken}, []string{writeToken})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(backend, tokens, log, 10*1024*1024)
}

func doRequest(t *testing.T, h http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestPutThenGetRoundtrip(t *testing.T) {
	h := newTestServer(t).Handler()
	payload := []byte("cached task output")

	putResp := doRequest(t, h, http.MethodPut, "/v1/cache/abc123", writeToken, payload)
	if putResp.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body=%s", putResp.Code, putResp.Body)
	}

	getResp := doRequest(t, h, http.MethodGet, "/v1/cache/abc123", readToken, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200; body=%s", getResp.Code, getResp.Body)
	}
	if getResp.Body.String() != string(payload) {
		t.Fatalf("GET body = %q, want %q", getResp.Body.String(), payload)
	}
}

func TestGetMissingReturns404(t *testing.T) {
	h := newTestServer(t).Handler()
	resp := doRequest(t, h, http.MethodGet, "/v1/cache/doesnotexist", readToken, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.Code)
	}
}

func TestPutDuplicateReturns409(t *testing.T) {
	h := newTestServer(t).Handler()
	payload := []byte("first write")

	first := doRequest(t, h, http.MethodPut, "/v1/cache/dup", writeToken, payload)
	if first.Code != http.StatusOK {
		t.Fatalf("first PUT status = %d, want 200", first.Code)
	}

	second := doRequest(t, h, http.MethodPut, "/v1/cache/dup", writeToken, []byte("second write"))
	if second.Code != http.StatusConflict {
		t.Fatalf("second PUT status = %d, want 409", second.Code)
	}
}

func TestNoTokenReturns401(t *testing.T) {
	h := newTestServer(t).Handler()

	get := doRequest(t, h, http.MethodGet, "/v1/cache/abc123", "", nil)
	if get.Code != http.StatusUnauthorized {
		t.Fatalf("GET without token status = %d, want 401", get.Code)
	}

	put := doRequest(t, h, http.MethodPut, "/v1/cache/abc123", "", []byte("x"))
	if put.Code != http.StatusUnauthorized {
		t.Fatalf("PUT without token status = %d, want 401", put.Code)
	}
}

func TestUnknownTokenReturns401(t *testing.T) {
	h := newTestServer(t).Handler()
	resp := doRequest(t, h, http.MethodGet, "/v1/cache/abc123", "not-a-real-token", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}
}

func TestReadOnlyTokenCannotWrite(t *testing.T) {
	h := newTestServer(t).Handler()
	resp := doRequest(t, h, http.MethodPut, "/v1/cache/abc123", readToken, []byte("nope"))
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.Code)
	}
}

func TestWriteTokenCanAlsoRead(t *testing.T) {
	h := newTestServer(t).Handler()
	payload := []byte("written by write token")
	if resp := doRequest(t, h, http.MethodPut, "/v1/cache/xyz", writeToken, payload); resp.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", resp.Code)
	}
	resp := doRequest(t, h, http.MethodGet, "/v1/cache/xyz", writeToken, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", resp.Code)
	}
}

func TestPutWithoutContentLengthReturns411(t *testing.T) {
	h := newTestServer(t).Handler()
	r := httptest.NewRequest(http.MethodPut, "/v1/cache/nolen", bytes.NewReader([]byte("x")))
	r.ContentLength = 0
	r.Header.Set("Authorization", "Bearer "+writeToken)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusLengthRequired {
		t.Fatalf("status = %d, want 411", w.Code)
	}
}

func TestInvalidHashRejected(t *testing.T) {
	h := newTestServer(t).Handler()
	resp := doRequest(t, h, http.MethodGet, "/v1/cache/abc%2e%2e%24def", readToken, nil)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for hash with disallowed characters", resp.Code)
	}
}

func TestHealthNoAuth(t *testing.T) {
	h := newTestServer(t).Handler()
	resp := doRequest(t, h, http.MethodGet, "/health", "", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
}
