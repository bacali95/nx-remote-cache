package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorize(t *testing.T) {
	store := NewTokenStore([]string{"reader"}, []string{"writer"})

	cases := []struct {
		name        string
		token       string
		scope       Scope
		wantValid   bool
		wantAllowed bool
	}{
		{"reader can read", "reader", ScopeRead, true, true},
		{"reader cannot write", "reader", ScopeWrite, true, false},
		{"writer can read", "writer", ScopeRead, true, true},
		{"writer can write", "writer", ScopeWrite, true, true},
		{"unknown token", "nope", ScopeRead, false, false},
		{"empty token", "", ScopeRead, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, allowed := store.Authorize(tc.token, tc.scope)
			if valid != tc.wantValid || allowed != tc.wantAllowed {
				t.Fatalf("Authorize(%q) = (%v, %v), want (%v, %v)", tc.token, valid, allowed, tc.wantValid, tc.wantAllowed)
			}
		})
	}
}

func TestExtractBearer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := ExtractBearer(r); got != "" {
		t.Fatalf("no header: got %q, want empty", got)
	}

	r.Header.Set("Authorization", "Bearer abc123")
	if got := ExtractBearer(r); got != "abc123" {
		t.Fatalf("got %q, want abc123", got)
	}

	r.Header.Set("Authorization", "Basic abc123")
	if got := ExtractBearer(r); got != "" {
		t.Fatalf("non-bearer scheme: got %q, want empty", got)
	}
}
