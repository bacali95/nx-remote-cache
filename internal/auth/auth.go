// Package auth implements bearer-token authentication with read/write scopes,
// matching the Nx self-hosted remote cache contract: a token either may only
// download artifacts (read) or may also upload them (write).
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type Scope int

const (
	ScopeRead Scope = iota
	ScopeWrite
)

// TokenStore holds the accepted bearer tokens. Every write token implicitly
// grants read access, since a CI job that can publish results can also fetch
// them.
type TokenStore struct {
	read  []string
	write []string
}

func NewTokenStore(readTokens, writeTokens []string) *TokenStore {
	return &TokenStore{read: readTokens, write: writeTokens}
}

func (s *TokenStore) hasWrite(token string) bool {
	return containsConstantTime(s.write, token)
}

func (s *TokenStore) hasRead(token string) bool {
	return containsConstantTime(s.read, token) || containsConstantTime(s.write, token)
}

// Authorize checks whether token is permitted the given scope. It returns
// (validToken, allowed): validToken is false when the token is not
// recognized at all (→ caller should respond 401); allowed is false when the
// token is recognized but lacks the requested scope (→ caller should respond
// 403).
func (s *TokenStore) Authorize(token string, scope Scope) (validToken, allowed bool) {
	known := s.hasRead(token) || s.hasWrite(token)
	if !known {
		return false, false
	}
	switch scope {
	case ScopeWrite:
		return true, s.hasWrite(token)
	default:
		return true, s.hasRead(token)
	}
}

func containsConstantTime(tokens []string, candidate string) bool {
	if candidate == "" {
		return false
	}
	found := false
	for _, t := range tokens {
		if len(t) != len(candidate) {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(t), []byte(candidate)) == 1 {
			found = true
		}
	}
	return found
}

// ExtractBearer pulls the token out of an "Authorization: Bearer <token>"
// header. Returns "" if the header is missing or malformed.
func ExtractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}
