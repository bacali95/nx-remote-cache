// Package server implements the Nx self-hosted remote cache HTTP contract:
// https://nx.dev/docs/kb/self-hosted-caching#build-your-own-caching-server
package server

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"nx-remote-cache/internal/auth"
	"nx-remote-cache/internal/storage"
)

type Server struct {
	backend       storage.Backend
	tokens        *auth.TokenStore
	log           *slog.Logger
	maxEntryBytes int64
}

func New(backend storage.Backend, tokens *auth.TokenStore, log *slog.Logger, maxEntryBytes int64) *Server {
	return &Server{backend: backend, tokens: tokens, log: log, maxEntryBytes: maxEntryBytes}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/cache/{hash}", s.handleGet)
	mux.HandleFunc("PUT /v1/cache/{hash}", s.handlePut)
	return withLogging(s.log, mux)
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
	valid, allowed := s.tokens.Authorize(token, auth.ScopeRead)
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
	valid, allowed := s.tokens.Authorize(token, auth.ScopeWrite)
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
	if r.ContentLength > s.maxEntryBytes {
		http.Error(w, "artifact exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
		return
	}

	err := s.backend.Put(r.Context(), hash, r.Body, r.ContentLength)
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
