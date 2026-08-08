package httplog

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithLoggingRecordsExplicitStatus(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	handler := WithLogging(log, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	r := httptest.NewRequest(http.MethodGet, "/v1/cache/abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	logged := buf.String()
	for _, want := range []string{"method=GET", "path=/v1/cache/abc", "status=404"} {
		if !strings.Contains(logged, want) {
			t.Errorf("log output %q missing %q", logged, want)
		}
	}
}

func TestWithLoggingDefaultsToStatusOKWhenNeverWritten(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	handler := WithLogging(log, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("log output %q should default to status=200", buf.String())
	}
}
