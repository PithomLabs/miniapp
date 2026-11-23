// Migration upgrade-path tests for 000006 (invocation attachment
// identity). Each test runs against a scratch database so fresh
// install, clean upgrade, and duplicate-data diagnostic paths are
// all exercised without touching shared test state.
package testkit

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
)

// scratchDSN creates an empty database and returns its DSN,
// dropping it on cleanup. Requires a superuser-capable server role
// (local Docker development).
func scratchDSN(t *testing.T, name string) string {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	maint := base
	if i := strings.LastIndex(maint, "/"); i >= 0 {
		maint = maint[:i] + "/postgres"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, maint)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP DATABASE IF EXISTS `+name); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), time.Minute)
		defer ccancel()
		cpool, err := pgxpool.New(cctx, maint)
		if err != nil {
			return
		}
		defer cpool.Close()
		_, _ = cpool.Exec(cctx, `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
	})
	dsn := base
	if i := strings.LastIndex(dsn, "/"); i >= 0 {
		dsn = dsn[:i] + "/" + name
	}
	return dsn
}

func migrateTo(t *testing.T, dsn string, version uint) {
	t.Helper()
	m, err := migrate.New("file://"+migrationsDir(t), pgx5URL(dsn))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Migrate(version); err != nil && err.Error() != "no change" {
		t.Fatal(err)
	}
}

func migrateUpErr(t *testing.T, dsn string) error {
	t.Helper()
	m, err := migrate.New("file://"+migrationsDir(t), pgx5URL(dsn))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	err = m.Up()
	if err != nil && err.Error() == "no change" {
		return nil
	}
	return err
}

// TestMigration000006FreshInstall migrates an empty database to
// latest and asserts the new identity objects exist.
func TestMigration000006FreshInstall(t *testing.T) {
	dsn := scratchDSN(t, "miniapp_mig_fresh")
	MigrateUp(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var col bool
	err = pool.QueryRow(ctx, `SELECT count(*) > 0 FROM information_schema.columns
		WHERE table_schema = 'app' AND table_name = 'invocations'
		AND column_name = 'attached_run_id'`).Scan(&col)
	if err != nil || !col {
		t.Fatalf("attached_run_id present = %v, %v", col, err)
	}
	var idx bool
	err = pool.QueryRow(ctx, `SELECT count(*) > 0 FROM pg_indexes
		WHERE schemaname = 'app' AND indexname = 'invocations_run_id_uidx'`).Scan(&idx)
	if err != nil || !idx {
		t.Fatalf("unique run_id index present = %v, %v", idx, err)
	}
}

// TestMigration000006DuplicateDiagnostic seeds duplicate nonempty
// run IDs at version 5, then proves the upgrade fails with the
// actionable preflight diagnostic and applies nothing.
func TestMigration000006DuplicateDiagnostic(t *testing.T) {
	dsn := scratchDSN(t, "miniapp_mig_dup")
	migrateTo(t, dsn, 5)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, key := range []string{"dup-a", "dup-b"} {
		_, err = pool.Exec(ctx, `INSERT INTO app.invocations
			(tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input, status, run_id)
			VALUES ('t1', 'm', '1.0.0', 'tool', $1, '{}', 'running', 'run-dup')`, key)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = migrateUpErr(t, dsn)
	if err == nil || !strings.Contains(err.Error(), "duplicate nonempty run_id") {
		t.Fatalf("upgrade err = %v, want the duplicate-run_id diagnostic", err)
	}
	if !strings.Contains(err.Error(), "run-dup") {
		t.Fatalf("diagnostic does not name the offending ID: %v", err)
	}
	var col bool
	_ = pool.QueryRow(ctx, `SELECT count(*) > 0 FROM information_schema.columns
		WHERE table_schema = 'app' AND table_name = 'invocations'
		AND column_name = 'attached_run_id'`).Scan(&col)
	if col {
		t.Fatal("attached_run_id applied despite refused migration (not atomic)")
	}
}

// TestMigration000006UniqueEnforced proves the partial unique index
// rejects duplicate nonempty run IDs while permitting any number
// of empty ones.
func TestMigration000006UniqueEnforced(t *testing.T) {
	dsn := scratchDSN(t, "miniapp_mig_uidx")
	MigrateUp(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	mkrow := func(key, run string) error {
		_, err := pool.Exec(ctx, `INSERT INTO app.invocations
			(tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input, status, run_id)
			VALUES ('t1', 'm', '1.0.0', 'tool', $1, '{}', 'running', $2)`, key, run)
		return err
	}
	if err := mkrow("u-empty-1", ""); err != nil {
		t.Fatal(err)
	}
	if err := mkrow("u-empty-2", ""); err != nil {
		t.Fatalf("second empty run_id rejected: %v", err)
	}
	if err := mkrow("u-one", "run-x"); err != nil {
		t.Fatal(err)
	}
	if err := mkrow("u-two", "run-x"); err == nil {
		t.Fatal("duplicate nonempty run_id accepted")
	} else if !strings.Contains(err.Error(), "invocations_run_id_uidx") {
		t.Fatalf("violation names %v, want the partial unique index", err)
	}
}

// TestMigration000006DownUpCycle proves the down file reverses
// 000006 cleanly and the upgrade re-applies after it.
func TestMigration000006DownUpCycle(t *testing.T) {
	dsn := scratchDSN(t, "miniapp_mig_cycle")
	MigrateUp(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	present := func() bool {
		var col bool
		err := pool.QueryRow(ctx, `SELECT count(*) > 0 FROM information_schema.columns
			WHERE table_schema = 'app' AND table_name = 'invocations'
			AND column_name = 'attached_run_id'`).Scan(&col)
		if err != nil {
			t.Fatal(err)
		}
		return col
	}
	if !present() {
		t.Fatal("attached_run_id missing after up")
	}
	migrateTo(t, dsn, 5)
	if present() {
		t.Fatal("attached_run_id survives down to 5")
	}
	MigrateUp(t, dsn)
	if !present() {
		t.Fatal("attached_run_id missing after re-upgrade")
	}
}

// TestMigration000006DelimiterCompat seeds delimiter-bearing
// identity values (the CR-07 multi-colon probe shape) and proves
// the upgrade applies: the X-19 escape handles them without a data
// rewrite, and no production rows exist to migrate.
func TestMigration000006DelimiterCompat(t *testing.T) {
	dsn := scratchDSN(t, "miniapp_mig_delim")
	migrateTo(t, dsn, 5)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `INSERT INTO app.effect_records
		(business_op_id, tenant_id, attempt_id, capability_id, operation, payload, state)
		VALUES ('local:subj:7:effect1', 'local', 'a1', 'mock', 'send', '{}', 'CONFIRMED')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateUpErr(t, dsn); err != nil {
		t.Fatalf("upgrade with delimiter-bearing IDs: %v", err)
	}
}
