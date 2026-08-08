package db

import (
	"context"
	"os"
	"testing"
)

// testDatabaseURL skips the test unless TEST_DATABASE_URL is set, since
// these tests need a real Postgres instance (see docker-compose.yml, or run
// one manually: docker run -e POSTGRES_PASSWORD=test -p 5432:5432 postgres).
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	return url
}

func TestMigrateIsIdempotent(t *testing.T) {
	url := testDatabaseURL(t)

	if err := Migrate(url); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(url); err != nil {
		t.Fatalf("second Migrate (should be a no-op): %v", err)
	}

	pool, err := Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()

	for _, table := range []string{"users", "sessions", "tokens"} {
		var exists bool
		err := pool.QueryRow(context.Background(),
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`,
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %q: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q was not created by migrations", table)
		}
	}
}

func TestConnectRejectsMalformedURL(t *testing.T) {
	_, err := Connect(context.Background(), "not a valid connection string")
	if err == nil {
		t.Fatal("expected an error for a malformed connection string")
	}
}

func TestConnectFailsPingAgainstUnreachableHost(t *testing.T) {
	// A syntactically valid DSN pointing at a port nothing listens on:
	// pgxpool.New succeeds (it doesn't dial eagerly), so this exercises
	// Connect's Ping error path specifically.
	_, err := Connect(context.Background(), "postgres://user:pass@127.0.0.1:1/db?connect_timeout=1")
	if err == nil {
		t.Fatal("expected a ping error for an unreachable host")
	}
}

func TestMigrateRejectsMalformedURL(t *testing.T) {
	if err := Migrate("not a valid connection string"); err == nil {
		t.Fatal("expected an error for a malformed connection string")
	}
}

func TestMigrateFailsAgainstUnreachableHost(t *testing.T) {
	err := Migrate("postgres://user:pass@127.0.0.1:1/db?connect_timeout=1")
	if err == nil {
		t.Fatal("expected an error for an unreachable host")
	}
}
