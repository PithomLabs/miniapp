package testkit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationsDir resolves the repository migrations directory
// independent of the test runner's working directory.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate testkit source")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "migrations")
}

// MigrateUp applies all pending migrations to dsn. It uses the pinned
// golang-migrate library with the pgx5 driver over the repository's
// migration files; no migration CLI installation is required.
func MigrateUp(t *testing.T, dsn string) {
	t.Helper()
	m, err := migrate.New("file://"+migrationsDir(t), pgx5URL(dsn))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err.Error() != "no change" {
		t.Fatal(err)
	}
}

func pgx5URL(dsn string) string {
	if s, ok := strings.CutPrefix(dsn, "postgres://"); ok {
		return "pgx5://" + s
	}
	return strings.Replace(dsn, "postgresql://", "pgx5://", 1)
}

// DB is a migrated test database. Tests share one database per process
// run and truncate the tables they touch; table creation is
// migration-owned, never ad-hoc in tests.
type DB struct {
	Pool *pgxpool.Pool
	DSN  string
}

// NewDB connects to TEST_DATABASE_URL, migrates it, and returns a
// pool. It skips (never fails) when the variable is unset.
func NewDB(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	MigrateUp(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return &DB{Pool: pool, DSN: dsn}
}

// Truncate empties the named tables (schema-qualified) for test
// isolation. Callers name exactly the tables their test writes.
func (db *DB) Truncate(t *testing.T, tables ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, table := range tables {
		if _, err := db.Pool.Exec(ctx, "TRUNCATE "+table); err != nil {
			t.Fatal(err)
		}
	}
}
