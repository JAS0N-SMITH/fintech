//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// authBootstrap creates the minimal auth schema that Supabase migrations depend on.
// In production this is provided by Supabase; in tests we stub it out.
const authBootstrap = `
CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.users (
    id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text
);

-- Stub returning NULL — integration tests connect as superuser (RLS bypassed).
CREATE OR REPLACE FUNCTION auth.uid() RETURNS uuid
    LANGUAGE sql STABLE AS $$ SELECT NULL::uuid $$;
`

// migrationsDir returns the absolute path to backend/migrations.
// Uses the test file's own location so it works regardless of working directory.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	// file = .../backend/internal/repository/testhelper_integration_test.go
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
}

// setupTestDB starts a Postgres testcontainer, bootstraps the auth schema,
// runs all goose migrations, and returns a ready pool.
// The container is terminated when t completes.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	// Bootstrap the auth schema before running migrations.
	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if _, err := sqlDB.ExecContext(ctx, authBootstrap); err != nil {
		t.Fatalf("bootstrap auth schema: %v", err)
	}

	// Run goose migrations via the sql.DB driver.
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose set dialect: %v", err)
	}
	migrationsPath := migrationsDir()
	if _, err := os.Stat(migrationsPath); err != nil {
		t.Fatalf("migrations dir not found at %s: %v", migrationsPath, err)
	}
	if err := goose.UpContext(ctx, sqlDB, migrationsPath); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	// Open pgxpool for the repository under test.
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("open pgxpool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

// insertTestUser inserts a row directly into auth.users and returns its UUID.
func insertTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO auth.users DEFAULT VALUES RETURNING id::text`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	return id
}

// ensure stdlib is registered (pgxpool uses pgx driver for sql.Open via stdlib tag).
var _ = stdlib.OpenDBFromPool
