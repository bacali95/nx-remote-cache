package store

import (
	"context"
	"testing"
	"time"
)

func TestListUsers(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if users, err := s.ListUsers(ctx); err != nil || len(users) != 0 {
		t.Fatalf("ListUsers on empty table = (%d, %v), want (0, nil)", len(users), err)
	}

	first, err := s.CreateUser(ctx, "a@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser a: %v", err)
	}
	second, err := s.CreateUser(ctx, "b@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser b: %v", err)
	}

	users, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 || users[0].ID != first.ID || users[1].ID != second.ID {
		t.Fatalf("ListUsers = %+v, want [a, b] in creation order", users)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetUserByID(context.Background(), 999999); err != ErrNotFound {
		t.Fatalf("GetUserByID(missing) = %v, want ErrNotFound", err)
	}
}

func TestUpdatePasswordNotFound(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpdatePassword(context.Background(), 999999, "new-hash"); err != ErrNotFound {
		t.Fatalf("UpdatePassword(missing) = %v, want ErrNotFound", err)
	}
}

func TestCreateTokenConflict(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	u, err := s.CreateUser(ctx, "admin@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := s.CreateToken(ctx, "ci-write", ScopeWrite, "same-hash", u.ID); err != nil {
		t.Fatalf("first CreateToken: %v", err)
	}
	if _, err := s.CreateToken(ctx, "ci-write-2", ScopeWrite, "same-hash", u.ID); err != ErrConflict {
		t.Fatalf("duplicate token hash: err = %v, want ErrConflict", err)
	}
}

func TestListTokensEmpty(t *testing.T) {
	s := newTestStore(t)
	tokens, err := s.ListTokens(context.Background())
	if err != nil || len(tokens) != 0 {
		t.Fatalf("ListTokens on empty table = (%d, %v), want (0, nil)", len(tokens), err)
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	u, err := s.CreateUser(ctx, "admin@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := s.CreateSession(ctx, "expired1", u.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("CreateSession (expired): %v", err)
	}
	if err := s.CreateSession(ctx, "active1", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession (active): %v", err)
	}

	if err := s.DeleteExpiredSessions(ctx); err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}

	if _, err := s.GetSession(ctx, "active1"); err != nil {
		t.Fatalf("active session should survive: %v", err)
	}

	// GetSession already excludes expired rows from its own WHERE clause,
	// so query the sessions table directly to prove the row is actually
	// gone rather than just filtered out.
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE id_hash = 'expired1'`).Scan(&n); err != nil {
		t.Fatalf("count expired1: %v", err)
	}
	if n != 0 {
		t.Fatalf("expired session row still present after DeleteExpiredSessions")
	}
}

func TestGetCacheReadStatsBatchEmptyInput(t *testing.T) {
	s := newTestStore(t)
	batch, err := s.GetCacheReadStatsBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetCacheReadStatsBatch(nil): %v", err)
	}
	if len(batch) != 0 {
		t.Fatalf("GetCacheReadStatsBatch(nil) = %+v, want empty map", batch)
	}
}

func TestListAllCacheReadStatsEmpty(t *testing.T) {
	s := newTestStore(t)
	all, err := s.ListAllCacheReadStats(context.Background())
	if err != nil {
		t.Fatalf("ListAllCacheReadStats: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("ListAllCacheReadStats on empty table = %+v, want empty map", all)
	}
}
