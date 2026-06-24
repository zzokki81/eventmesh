//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationsDir is the path to the order service migrations, relative to this
// package's directory (order/repository/postgres).
const migrationsDir = "../../migrations"

// testPool is the shared connection pool for this package's integration tests.
// It is backed by a single throwaway PostgreSQL container started once in
// TestMain and torn down when the package's tests finish.
var testPool *pgxpool.Pool

// TestMain starts one PostgreSQL container for the whole package, applies the
// order migrations against it, and exposes a ready pool via testPool. Running
// the container once (rather than per-test) keeps the suite fast; each test
// isolates itself by resetting the tables it touches (see resetTables).
func TestMain(m *testing.M) {
	code, err := runWithPostgres(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// runWithPostgres owns the container lifecycle so its deferred cleanup runs
// before the process exits; TestMain turns the returned code into os.Exit.
func runWithPostgres(m *testing.M) (int, error) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("eventmesh_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		// PostgreSQL restarts once during first-time init, so wait for the
		// "ready" log line to appear twice before treating it as up.
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return 1, fmt.Errorf("start postgres container: %w", err)
	}
	defer func() { _ = ctr.Terminate(ctx) }()

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 1, fmt.Errorf("connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return 1, fmt.Errorf("open pool: %w", err)
	}
	defer pool.Close()

	if err := applyMigrations(ctx, pool); err != nil {
		return 1, fmt.Errorf("apply migrations: %w", err)
	}

	testPool = pool
	return m.Run(), nil
}

// applyMigrations runs every *.up.sql file in migrationsDir in lexical order
// (the files are zero-padded and numbered), so the test schema matches the one
// the service uses in production without duplicating any SQL here.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	pattern := filepath.Join(migrationsDir, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob migrations %q: %w", pattern, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migrations found at %q", pattern)
	}
	sort.Strings(files)

	for _, f := range files {
		stmt, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := pool.Exec(ctx, string(stmt)); err != nil {
			return fmt.Errorf("exec migration %s: %w", f, err)
		}
	}
	return nil
}

// resetTables empties the tables so each test starts from a clean slate,
// regardless of order. It is also registered as a cleanup so a test does not
// leak rows into the next one.
func resetTables(t *testing.T) {
	t.Helper()
	const q = "TRUNCATE orders, outbox_events"
	if _, err := testPool.Exec(context.Background(), q); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), q); err != nil {
			t.Errorf("reset tables (cleanup): %v", err)
		}
	})
}
