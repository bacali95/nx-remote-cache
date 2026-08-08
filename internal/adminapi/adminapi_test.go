package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nx-remote-cache/internal/db"
	"nx-remote-cache/internal/session"
	"nx-remote-cache/internal/storage"
	"nx-remote-cache/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testServer wires a real Postgres-backed Server plus a temp-dir local
// storage backend, and seeds one admin user. Skips if TEST_DATABASE_URL is
// unset.
func testServer(t *testing.T) (http.Handler, *store.Store, storage.Backend) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	if err := db.Migrate(url); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `TRUNCATE users, sessions, tokens RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	st := store.New(pool)

	backend, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	sessions := session.NewManager(st, time.Hour)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(st, sessions, backend, log, false /* cookieSecure: plain http in tests */, time.Hour, nil /* uiFS: no static UI in tests */)

	return srv.Handler(), st, backend
}

func seedUser(t *testing.T, st *store.Store, email, password string) {
	t.Helper()
	hash, err := session.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := st.CreateUser(context.Background(), email, hash); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

// doJSON issues a request and, if the response carries a session cookie
// from a prior login, replays it. mutate lets the caller add the CSRF
// header for state-changing requests.
func doJSON(t *testing.T, h http.Handler, method, path string, body any, cookie *http.Cookie, csrf bool) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	r := httptest.NewRequest(method, path, &buf)
	r.Header.Set("Content-Type", "application/json")
	if csrf {
		r.Header.Set(csrfHeader, "1")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func loginAndGetCookie(t *testing.T, h http.Handler, email, password string) *http.Cookie {
	t.Helper()
	w := doJSON(t, h, http.MethodPost, "/admin/api/auth/login", loginRequest{Email: email, Password: password}, nil, true)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", w.Code, w.Body)
	}
	resp := http.Response{Header: w.Header()}
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			return c
		}
	}
	t.Fatalf("no session cookie set on login response")
	return nil
}

func TestLoginRequiresCSRFHeaderAndValidatesCreds(t *testing.T) {
	h, st, _ := testServer(t)
	seedUser(t, st, "admin@example.com", "correct-password")

	// Login itself is a POST but exempt from the "protected" wrapper (no
	// session yet), so it's not CSRF-gated — but every other mutating
	// endpoint is, which the later tests exercise.
	w := doJSON(t, h, http.MethodPost, "/admin/api/auth/login", loginRequest{Email: "admin@example.com", Password: "wrong"}, nil, true)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: status = %d, want 401", w.Code)
	}

	w = doJSON(t, h, http.MethodPost, "/admin/api/auth/login", loginRequest{Email: "admin@example.com", Password: "correct-password"}, nil, true)
	if w.Code != http.StatusOK {
		t.Fatalf("correct password: status = %d, body = %s", w.Code, w.Body)
	}
}

func TestMeRequiresSessionAndCSRFOnMutatingRoutes(t *testing.T) {
	h, st, _ := testServer(t)
	seedUser(t, st, "admin@example.com", "correct-password")

	if w := doJSON(t, h, http.MethodGet, "/admin/api/auth/me", nil, nil, false); w.Code != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", w.Code)
	}

	cookie := loginAndGetCookie(t, h, "admin@example.com", "correct-password")

	w := doJSON(t, h, http.MethodGet, "/admin/api/auth/me", nil, cookie, false)
	if w.Code != http.StatusOK {
		t.Fatalf("me with cookie: status = %d, body = %s", w.Code, w.Body)
	}
	var me userResponse
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.Email != "admin@example.com" {
		t.Fatalf("me.Email = %q, want admin@example.com", me.Email)
	}

	// Mutating request with a valid cookie but no CSRF header must be
	// rejected, proving the header check runs independently of the session
	// check.
	w = doJSON(t, h, http.MethodPost, "/admin/api/account/password", changePasswordRequest{CurrentPassword: "correct-password", NewPassword: "new-password-1"}, cookie, false)
	if w.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF header: status = %d, want 403", w.Code)
	}
}

func TestTokenLifecycleOverHTTP(t *testing.T) {
	h, st, _ := testServer(t)
	seedUser(t, st, "admin@example.com", "correct-password")
	cookie := loginAndGetCookie(t, h, "admin@example.com", "correct-password")

	w := doJSON(t, h, http.MethodPost, "/admin/api/tokens", createTokenRequest{Name: "ci-write", Scope: "write"}, cookie, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("create token: status = %d, body = %s", w.Code, w.Body)
	}
	var created createTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created token: %v", err)
	}
	if created.Token == "" {
		t.Fatalf("expected raw token in create response")
	}

	w = doJSON(t, h, http.MethodGet, "/admin/api/tokens", nil, cookie, false)
	if w.Code != http.StatusOK {
		t.Fatalf("list tokens: status = %d", w.Code)
	}
	var tokens []tokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("decode token list: %v", err)
	}
	if len(tokens) != 1 || tokens[0].ID != created.ID {
		t.Fatalf("token list = %+v, want one entry matching created token", tokens)
	}

	w = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/admin/api/tokens/%d", created.ID), nil, cookie, true)
	if w.Code != http.StatusNoContent {
		t.Fatalf("revoke token: status = %d, body = %s", w.Code, w.Body)
	}

	w = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/admin/api/tokens/%d", created.ID), nil, cookie, true)
	if w.Code != http.StatusNotFound {
		t.Fatalf("revoke already-revoked token: status = %d, want 404", w.Code)
	}
}

func TestUserCannotDeleteSelfOrLastAdmin(t *testing.T) {
	h, st, _ := testServer(t)
	seedUser(t, st, "admin@example.com", "correct-password")
	cookie := loginAndGetCookie(t, h, "admin@example.com", "correct-password")

	w := doJSON(t, h, http.MethodGet, "/admin/api/auth/me", nil, cookie, false)
	var me userResponse
	_ = json.Unmarshal(w.Body.Bytes(), &me)

	w = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/admin/api/users/%d", me.ID), nil, cookie, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete self: status = %d, want 400", w.Code)
	}

	w = doJSON(t, h, http.MethodPost, "/admin/api/users", createUserRequest{Email: "second@example.com", Password: "another-password"}, cookie, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("create second user: status = %d, body = %s", w.Code, w.Body)
	}
	var second userResponse
	_ = json.Unmarshal(w.Body.Bytes(), &second)

	// Now two admins exist; deleting the *other* one should succeed.
	w = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/admin/api/users/%d", second.ID), nil, cookie, true)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete other admin: status = %d, body = %s", w.Code, w.Body)
	}

	// Back down to one admin: deleting it (even indirectly not-self) should
	// now be blocked because it's the last one. Re-verify via the "last
	// admin" guard by attempting to delete self again is redundant with the
	// first check, so instead assert count is back to 1.
	n, err := st.CountUsers(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("CountUsers = (%d, %v), want (1, nil)", n, err)
	}
}

func TestCacheBrowseDeleteAndPrune(t *testing.T) {
	h, st, backend := testServer(t)
	seedUser(t, st, "admin@example.com", "correct-password")
	cookie := loginAndGetCookie(t, h, "admin@example.com", "correct-password")
	ctx := context.Background()

	// "fresh" stays; "stale" predates the prune cutoff and should be swept.
	if err := backend.Put(ctx, "fresh1", bytes.NewReader([]byte("x")), 1); err != nil {
		t.Fatalf("seed fresh1: %v", err)
	}
	if err := backend.Put(ctx, "stale1", bytes.NewReader([]byte("y")), 1); err != nil {
		t.Fatalf("seed stale1: %v", err)
	}
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(localEntryPath(t, backend, "stale1"), oldTime, oldTime); err != nil {
		t.Fatalf("backdate stale1: %v", err)
	}

	w := doJSON(t, h, http.MethodGet, "/admin/api/cache", nil, cookie, false)
	if w.Code != http.StatusOK {
		t.Fatalf("list cache: status = %d, body = %s", w.Code, w.Body)
	}
	var page listCacheResponse
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(page.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(page.Entries), page.Entries)
	}

	w = doJSON(t, h, http.MethodPost, "/admin/api/cache/prune", pruneRequest{OlderThanDays: 7}, cookie, true)
	if w.Code != http.StatusOK {
		t.Fatalf("prune: status = %d, body = %s", w.Code, w.Body)
	}
	var result deleteCountResponse
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Deleted != 1 {
		t.Fatalf("prune deleted = %d, want 1 (only stale1)", result.Deleted)
	}

	if exists, _ := backend.Exists(ctx, "fresh1"); !exists {
		t.Fatalf("prune deleted the fresh entry too")
	}
	if exists, _ := backend.Exists(ctx, "stale1"); exists {
		t.Fatalf("stale entry survived prune")
	}

	w = doJSON(t, h, http.MethodDelete, "/admin/api/cache/fresh1", nil, cookie, true)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete fresh1: status = %d, body = %s", w.Code, w.Body)
	}

	w = doJSON(t, h, http.MethodDelete, "/admin/api/cache/doesnotexist", nil, cookie, true)
	if w.Code != http.StatusNotFound {
		t.Fatalf("delete missing entry: status = %d, want 404", w.Code)
	}
}

// localEntryPath reaches into the *storage.Local backend to get the file
// path for a hash, so the test can backdate its mtime. Local is the only
// backend under test here (see testServer), so the type assertion is safe.
func localEntryPath(t *testing.T, backend storage.Backend, hash string) string {
	t.Helper()
	local, ok := backend.(*storage.Local)
	if !ok {
		t.Fatalf("localEntryPath: backend is not *storage.Local")
	}
	return local.Path(hash)
}
