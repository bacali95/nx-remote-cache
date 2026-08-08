// Package server implements the Nx self-hosted remote cache HTTP contract:
// https://nx.dev/docs/kb/self-hosted-caching#build-your-own-caching-server
package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"nx-remote-cache/internal/auth"
	"nx-remote-cache/internal/httplog"
	"nx-remote-cache/internal/storage"
)

// ReadTracker records cache hits (count + timestamp) for the admin UI and
// the background janitor's "unread" prune rule. Recording is best-effort
// and off the request's critical path — see handleGet.
type ReadTracker interface {
	RecordCacheRead(ctx context.Context, hash string) error
}

type Server struct {
	backend       storage.Backend
	tokens        *auth.CacheTokenAuth
	reads         ReadTracker
	log           *slog.Logger
	maxEntryBytes atomic.Int64
}

func New(backend storage.Backend, tokens *auth.CacheTokenAuth, reads ReadTracker, log *slog.Logger, maxEntryBytes int64) *Server {
	s := &Server{backend: backend, tokens: tokens, reads: reads, log: log}
	s.SetMaxEntryBytes(maxEntryBytes)
	return s
}

// SetMaxEntryBytes updates the upload size limit enforced on new PUTs.
// Safe to call concurrently — see internal/settings, which calls this when
// an admin changes the setting from the UI.
func (s *Server) SetMaxEntryBytes(n int64) {
	s.maxEntryBytes.Store(n)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/cache/{hash}", s.handleGet)
	mux.HandleFunc("PUT /v1/cache/{hash}", s.handlePut)
	return httplog.WithLogging(s.log, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleGet implements "Download a task output": 200 + binary body on hit,
// 404 on miss, 401/403 on auth failure.
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if !storage.ValidHash(hash) {
		http.Error(w, "invalid hash", http.StatusBadRequest)
		return
	}

	token := auth.ExtractBearer(r)
	valid, allowed, err := s.tokens.Authorize(r.Context(), token, auth.ScopeRead)
	if err != nil {
		s.log.Error("token authorize failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "Missing or invalid authentication token", http.StatusUnauthorized)
		return
	}
	if !allowed {
		http.Error(w, "Access forbidden", http.StatusForbidden)
		return
	}

	body, size, err := s.backend.Get(r.Context(), hash)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Error("get artifact failed", "hash", hash, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = body.Close() }()

	// Best-effort and off the critical path: a slow or failing tracking
	// write must never slow down or fail a cache download. Uses its own
	// context rather than the request's, since the request's is cancelled
	// the moment this handler returns — likely before this goroutine runs.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.reads.RecordCacheRead(ctx, hash); err != nil {
			s.log.Warn("record cache read failed", "hash", hash, "error", err)
		}
	}()

	w.Header().Set("Content-Type", "application/octet-stream")
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		s.log.Warn("stream artifact interrupted", "hash", hash, "error", err)
	}
}

// handlePut implements "Upload a task output": 200 on success, 409 if the
// hash is already stored (cache entries are immutable/content-addressed),
// 401/403 on auth failure.
func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if !storage.ValidHash(hash) {
		http.Error(w, "invalid hash", http.StatusBadRequest)
		return
	}

	token := auth.ExtractBearer(r)
	valid, allowed, err := s.tokens.Authorize(r.Context(), token, auth.ScopeWrite)
	if err != nil {
		s.log.Error("token authorize failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "Missing or invalid authentication token", http.StatusUnauthorized)
		return
	}
	if !allowed {
		http.Error(w, "Access forbidden", http.StatusForbidden)
		return
	}

	if r.ContentLength <= 0 {
		http.Error(w, "Content-Length header is required", http.StatusLengthRequired)
		return
	}
	if r.ContentLength > s.maxEntryBytes.Load() {
		http.Error(w, "artifact exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
		return
	}

	err = s.backend.Put(r.Context(), hash, r.Body, r.ContentLength)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, storage.ErrAlreadyExists):
		http.Error(w, "artifact already exists", http.StatusConflict)
	default:
		s.log.Error("put artifact failed", "hash", hash, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
