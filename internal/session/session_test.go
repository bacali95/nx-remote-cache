package session

import (
	"context"
	"os"
	"testing"
	"time"

	"nx-remote-cache/internal/db"
	"nx-remote-cache/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) *store.Store {
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
	return store.New(pool)
}

func TestLoginLogoutLifecycle(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	m := NewManager(s, time.Hour)

	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := s.CreateUser(ctx, "admin@example.com", hash); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := m.Login(ctx, "admin@example.com", "wrong-password"); err != ErrInvalidCredentials {
		t.Fatalf("wrong password: err = %v, want ErrInvalidCredentials", err)
	}
	if _, err := m.Login(ctx, "nope@example.com", "whatever"); err != ErrInvalidCredentials {
		t.Fatalf("unknown email: err = %v, want ErrInvalidCredentials", err)
	}

	sessionID, err := m.Login(ctx, "admin@example.com", "correct-password")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sessionID == "" {
		t.Fatalf("expected non-empty session id")
	}

	u, err := m.CurrentUser(ctx, sessionID)
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if u.Email != "admin@example.com" {
		t.Fatalf("CurrentUser returned %q, want admin@example.com", u.Email)
	}

	if err := m.Logout(ctx, sessionID); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := m.CurrentUser(ctx, sessionID); err != store.ErrNotFound {
		t.Fatalf("after logout: err = %v, want ErrNotFound", err)
	}
}

func TestCurrentUserEmptySession(t *testing.T) {
	s := newTestStore(t)
	m := NewManager(s, time.Hour)
	if _, err := m.CurrentUser(context.Background(), ""); err != store.ErrNotFound {
		t.Fatalf("empty session: err = %v, want ErrNotFound", err)
	}
}
