//go:build integration

package testhelpers

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewTestDB starts a Postgres container, applies testdata/schema.sql, and returns a pool.
// The container and pool are closed automatically via t.Cleanup.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	_, file, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "schema.sql")

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithInitScripts(schemaPath),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
