package spikes

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/internal/store"
)

// concTenants are the distinct identities whose isolation is proven
// under concurrency.
var concTenants = []string{"conc-a", "conc-b", "conc-c", "conc-d"}

// workersPerTenant is hoisted so the fixture pool can be sized from
// the same constants as the barrier: every participant pins one
// pool connection across it.
const workersPerTenant = 2

// concParticipants is the number of workers that must hold open
// transactions simultaneously at the barrier.
func concParticipants() int { return len(concTenants) * workersPerTenant }

// concFixture owns the concurrent-isolation fixture: its own table and
// role, created in setup and dropped in cleanup. Nothing outside these
// objects is touched.
type concFixture struct {
	pool *pgxpool.Pool
}

func setupConcIsolation(t *testing.T, ctx context.Context) *concFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		t.Fatal("cannot determine database name from DSN")
	}
	const appPassword = "spike-conc-only-do-not-reuse"
	setup := []string{
		`CREATE SCHEMA IF NOT EXISTS spike_rls`,
		`DROP TABLE IF EXISTS spike_rls.conc_docs`,
		`CREATE TABLE spike_rls.conc_docs (id serial primary key, tenant_id text not null, body text)`,
		`ALTER TABLE spike_rls.conc_docs ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE spike_rls.conc_docs FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY tenant_isolation ON spike_rls.conc_docs USING (tenant_id = current_setting('app.tenant_id', true)) WITH CHECK (tenant_id = current_setting('app.tenant_id', true))`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'spike_app_conc') THEN EXECUTE 'DROP OWNED BY spike_app_conc'; END IF; END $$`,
		`DROP ROLE IF EXISTS spike_app_conc`,
		`CREATE ROLE spike_app_conc WITH LOGIN PASSWORD '` + appPassword + `' NOBYPASSRLS`,
		`GRANT CONNECT ON DATABASE "` + dbName + `" TO spike_app_conc`,
		`GRANT USAGE ON SCHEMA spike_rls TO spike_app_conc`,
		`GRANT SELECT, INSERT ON spike_rls.conc_docs TO spike_app_conc`,
		`GRANT USAGE, SELECT ON SEQUENCE spike_rls.conc_docs_id_seq TO spike_app_conc`,
	}
	for _, q := range setup {
		if _, err := admin.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	poolCfg, err := pgxpool.ParseConfig(appRoleDSN(t, dsn, "spike_app_conc", appPassword))
	if err != nil {
		t.Fatal(err)
	}
	// Size explicitly from the test's own concurrency: every barrier
	// participant holds one connection across it, and the pgx default
	// (max(4, NumCPU)) starves the barrier on small machines. +2
	// headroom covers the post-barrier counts and negative control.
	poolCfg.MaxConns = int32(concParticipants() + 2)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	// Seed one committed row per tenant BEFORE spawning workers. Without
	// committed foreign rows, MVCC snapshot isolation alone would hide
	// other tenants' uncommitted writes and the overlap assertions
	// would pass even with RLS disabled. The seed makes every read
	// RLS-sensitive by construction. Seeding runs through armed
	// application transactions (not the bootstrap role) so setup makes
	// no privilege assumptions beyond the grants above.
	for _, tenant := range concTenants {
		if _, err := store.WithTenant(ctx, pool, tenant, func(tx pgx.Tx) (bool, error) {
			_, err := tx.Exec(ctx, `INSERT INTO spike_rls.conc_docs (tenant_id, body) VALUES ($1, 'seed')`, tenant)
			return true, err
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(pool.Close)
	t.Cleanup(admin.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		admin, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return
		}
		defer admin.Close()
		_, _ = admin.Exec(ctx, `DROP TABLE IF EXISTS spike_rls.conc_docs`)
		_, _ = admin.Exec(ctx, `DROP OWNED BY spike_app_conc`)
		_, _ = admin.Exec(ctx, `DROP ROLE IF EXISTS spike_app_conc`)
	})
	return &concFixture{pool: pool}
}

// TestConcurrentTenantIsolation proves tenant isolation under genuine
// overlap: every worker holds an open transaction with its own tenant
// context while all participants proceed together through two
// barriers. Each tenant writes and reads only its own rows; an
// unarmed transaction sees nothing. Coordination is barrier-based,
// never timing-based; timeouts below are deadlock backstops that fail
// loudly, not assertions about speed.
func TestConcurrentTenantIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := setupConcIsolation(t, ctx)
	if got := int(fx.pool.Config().MaxConns); got < concParticipants() {
		t.Fatalf("pool MaxConns=%d < %d barrier participants: barrier would starve", got, concParticipants())
	}

	tenants := concTenants
	type observation struct {
		tenant string
		seen   []string
	}
	ready := make(chan struct{}, len(tenants)*workersPerTenant)
	releaseReads := make(chan struct{})
	doneWrites := make(chan struct{}, len(tenants)*workersPerTenant)
	releaseCommit := make(chan struct{})
	results := make(chan observation, len(tenants)*workersPerTenant)
	errs := make(chan error, len(tenants)*workersPerTenant+1)

	var wg sync.WaitGroup
	for _, tenant := range tenants {
		for w := 0; w < workersPerTenant; w++ {
			wg.Add(1)
			go func(tenant string, worker int) {
				defer wg.Done()
				_, err := store.WithTenant(ctx, fx.pool, tenant, func(tx pgx.Tx) (bool, error) {
					// Phase 1: context is armed; wait until every
					// participant holds an open armed transaction.
					ready <- struct{}{}
					select {
					case <-releaseReads:
					case <-ctx.Done():
						return false, ctx.Err()
					}
					// Phase 2: write then read while all
					// transactions overlap.
					body := fmt.Sprintf("%s-w%d", tenant, worker)
					if _, err := tx.Exec(ctx, `INSERT INTO spike_rls.conc_docs (tenant_id, body) VALUES ($1, $2)`, tenant, body); err != nil {
						return false, err
					}
					rows, err := tx.Query(ctx, `SELECT tenant_id FROM spike_rls.conc_docs`)
					if err != nil {
						return false, err
					}
					var seen []string
					for rows.Next() {
						var got string
						if err := rows.Scan(&got); err != nil {
							rows.Close()
							return false, err
						}
						seen = append(seen, got)
					}
					rows.Close()
					if rows.Err() != nil {
						return false, rows.Err()
					}
					doneWrites <- struct{}{}
					select {
					case <-releaseCommit:
					case <-ctx.Done():
						return false, ctx.Err()
					}
					results <- observation{tenant: tenant, seen: seen}
					return true, nil
				})
				if err != nil {
					errs <- err
				}
			}(tenant, w)
		}
	}

	total := len(tenants) * workersPerTenant
	waitFor := func(ch chan struct{}, n int, what string) {
		t.Helper()
		for got := 0; got < n; {
			select {
			case <-ch:
				got++
			case err := <-errs:
				t.Fatalf("worker failed before %s (%d/%d ready): %v", what, got, n, err)
			case <-ctx.Done():
				t.Fatalf("deadlock waiting for %s (%d/%d ready): %v", what, got, n, ctx.Err())
			}
		}
	}
	waitFor(ready, total, "armed transactions")
	close(releaseReads)
	waitFor(doneWrites, total, "overlapped writes")
	close(releaseCommit)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("worker error: %v", err)
	}
	close(results)

	// Every worker must have seen only its own tenant's rows, and at
	// least its own write (proving the read observed real data, not an
	// empty table).
	counts := map[string]int{}
	for ob := range results {
		if len(ob.seen) == 0 {
			t.Fatalf("tenant %s saw no rows (expected its own writes)", ob.tenant)
		}
		for _, got := range ob.seen {
			if got != ob.tenant {
				t.Fatalf("tenant %s saw foreign row %q: isolation violated", ob.tenant, got)
			}
		}
		counts[ob.tenant]++
	}
	for _, tenant := range tenants {
		if counts[tenant] != workersPerTenant {
			t.Fatalf("tenant %s reported %d observations, want %d", tenant, counts[tenant], workersPerTenant)
		}
	}

	// Final durable state: each tenant counts its own rows through a
	// fresh armed transaction — one seed row plus one row per worker.
	// Counting through RLS (rather than bypassing it) keeps the whole
	// test inside the application role's privileges.
	for _, tenant := range tenants {
		n, err := store.WithTenant(ctx, fx.pool, tenant, func(tx pgx.Tx) (int, error) {
			var n int
			err := tx.QueryRow(ctx, `SELECT count(*) FROM spike_rls.conc_docs`).Scan(&n)
			return n, err
		})
		if err != nil {
			t.Fatal(err)
		}
		if n != 1+workersPerTenant {
			t.Fatalf("tenant %s owns %d rows, want %d", tenant, n, 1+workersPerTenant)
		}
	}

	// Negative control: the identical read through a deliberately
	// unarmed transaction must see nothing. If RLS were absent or the
	// arming step were load-bearing-free, this would return rows and
	// the control would fail — proving the main assertions are
	// sensitive to context arming rather than vacuous.
	conn, err := fx.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var unarmed int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM spike_rls.conc_docs`).Scan(&unarmed); err != nil {
		t.Fatal(err)
	}
	if unarmed != 0 {
		t.Fatalf("unarmed transaction saw %d rows, want 0 (default-deny violated)", unarmed)
	}
}
