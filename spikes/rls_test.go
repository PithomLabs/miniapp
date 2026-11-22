package spikes

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/internal/store"
)

// appRoleDSN derives an application-role DSN from the bootstrap DSN by
// swapping userinfo. Superusers bypass RLS unconditionally, so the
// isolation assertions must run as a NOBYPASSRLS role, never as the
// bootstrap superuser.
func appRoleDSN(t *testing.T, bootstrap, user, password string) string {
	t.Helper()
	u, err := url.Parse(bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(user, password)
	return u.String()
}

// TestRLSTenantContext proves the tenant-context mechanism behind the
// RLS defense-in-depth layer: without app.tenant_id set, a tenant
// policy hides every row; with it set transaction-locally, exactly the
// tenant's rows appear; and the setting does not leak across pooled
// checkouts. Runs on the disposable Phase 0 branch (Q4).
func TestRLSTenantContext(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	// Disposable test-only credential for the NOBYPASSRLS role. This
	// database is local-only or a disposable branch; the value is never
	// logged, and the role is dropped in cleanup.
	const appPassword = "spike-app-only-do-not-reuse"
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		t.Fatal("cannot determine database name from DSN")
	}
	setup := []string{
		`CREATE SCHEMA IF NOT EXISTS spike_rls`,
		`DROP TABLE IF EXISTS spike_rls.docs`,
		`CREATE TABLE spike_rls.docs (id serial primary key, tenant_id text not null, body text)`,
		`INSERT INTO spike_rls.docs (tenant_id, body) VALUES ('a', 'one'), ('b', 'two')`,
		`ALTER TABLE spike_rls.docs ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE spike_rls.docs FORCE ROW LEVEL SECURITY`,
		`DROP POLICY IF EXISTS tenant_isolation ON spike_rls.docs`,
		`CREATE POLICY tenant_isolation ON spike_rls.docs USING (tenant_id = current_setting('app.tenant_id', true))`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'spike_app') THEN EXECUTE 'DROP OWNED BY spike_app'; END IF; END $$`,
		`DROP ROLE IF EXISTS spike_app`,
		`CREATE ROLE spike_app WITH LOGIN PASSWORD '` + appPassword + `' NOBYPASSRLS`,
		`GRANT CONNECT ON DATABASE "` + dbName + `" TO spike_app`,
		`GRANT USAGE ON SCHEMA spike_rls TO spike_app`,
		`GRANT SELECT, INSERT ON spike_rls.docs TO spike_app`,
	}
	for _, q := range setup {
		if _, err := admin.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = admin.Exec(ctx, `DROP TABLE IF EXISTS spike_rls.docs`)
		_, _ = admin.Exec(ctx, `DROP OWNED BY spike_app`)
		_, _ = admin.Exec(ctx, `DROP ROLE IF EXISTS spike_app`)
	})

	pool, err := pgxpool.New(ctx, appRoleDSN(t, dsn, "spike_app", appPassword))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// count uses the production tenant helper: context armed inside
	// the same transaction as the tenant query. An empty tenant runs
	// outside any context to prove the default-deny behavior.
	count := func(ctx context.Context, tenant string) int {
		if tenant == "" {
			var n int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM spike_rls.docs`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			return n
		}
		n, err := store.WithTenant(ctx, pool, tenant, func(tx pgx.Tx) (int, error) {
			var n int
			err := tx.QueryRow(ctx, `SELECT count(*) FROM spike_rls.docs`).Scan(&n)
			return n, err
		})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if got := count(ctx, ""); got != 0 {
		t.Fatalf("unset context exposed %d rows, want 0", got)
	}
	if got := count(ctx, "a"); got != 1 {
		t.Fatalf("tenant a saw %d rows, want 1", got)
	}
	// A fresh checkout must not inherit the previous tenant setting.
	if got := count(ctx, ""); got != 0 {
		t.Fatalf("tenant context leaked across pooled checkouts: %d rows", got)
	}
}
