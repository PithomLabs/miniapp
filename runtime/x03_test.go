// X-03 discriminator: concurrent same-key submissions.
//
// RED/GREEN-COMPATIBLE SOURCE: uses only public Invoke, SQL
// fixtures, pg_stat_activity observation, and pre-existing fake
// inspection, plus the shared red fixture from x01x02_test.go
// (also red-compatible). The red run overlays this file and
// x01x02_test.go onto the snapshot.
//
// Determinism mechanism (DB lock-gate, not a timing barrier): a
// holder transaction locks the pre-claimed invocation row; both
// racers must observe the empty run_id (nothing is committed)
// before either can write. The test proves both racers are
// blocked behind the holder on the intended row (transitive
// pg_blocking_pids chains rooting at the holder + statement shape
// + no other lock waiters database-wide) before releasing. Red:
// both updates land distinct IDs (orphan), every round. Green:
// predicated adoption yields one canonical run, every round.
package runtime_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

var errNoHandle = errors.New("missing handle")

// rowGateHolder locks one invocation row in an uncommitted
// transaction on a dedicated connection. Callers must invoke the
// returned release exactly once (rollback; nothing was written).
// Duplicated per red-compatible test file on purpose: shared
// helpers would need overlaying into the snapshot's non-test
// files, which red/green forbids.
func rowGateHolder(t *testing.T, db *testkit.DB, tenant, invID string) (holderPID int32, release func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	pool, err := pgxpool.New(ctx, db.DSN)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, store.TenantContextSQL(), tenant); err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	var id string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM app.invocations
		WHERE id = $1::uuid FOR UPDATE`, invID).Scan(&id); err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holderPID); err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	return holderPID, func() {
		defer cancel()
		defer pool.Close()
		_ = tx.Rollback(ctx)
	}
}

// waitBlockedByHolder waits until exactly want backends are
// lock-blocked behind the holder on statements touching tableLike,
// with no other lock waiters database-wide. Blockage is proven
// transitively: each waiter's pg_blocking_pids chain must root at
// the holder pid (tuple-lock queues chain waiter-behind-waiter, so
// direct linkage alone is insufficient). Reports success; the
// caller releases the holder and joins racers on every path.
func waitBlockedByHolder(t *testing.T, db *testkit.DB, holderPID int32, tableLike string, want int) ([]int32, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	deadline := time.Now().Add(30 * time.Second)
	for {
		matched, total, err := blockedBehind(t, ctx, db, holderPID, tableLike)
		if err != nil {
			t.Fatal(err)
		}
		if len(matched) == want && total == want {
			return matched, true
		}
		if time.Now().After(deadline) {
			dumpActivity(t, ctx, db)
			t.Logf("blocked-behind-holder waiters = %d (pids %v), total lock waiters = %d; want %d on %s (holder %d)",
				len(matched), matched, total, want, tableLike, holderPID)
			return nil, false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// blockedBehind returns the pids whose statements touch tableLike
// and whose blocking chain roots at the holder, plus the total
// lock-waiter count database-wide.
func blockedBehind(t *testing.T, ctx context.Context, db *testkit.DB, holderPID int32, tableLike string) ([]int32, int, error) {
	t.Helper()
	type backend struct {
		pid      int32
		query    string
		blocking []int32
	}
	rows, err := db.Pool.Query(ctx, `SELECT pid, query, pg_blocking_pids(pid) FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid()
		AND wait_event_type = 'Lock'`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	byPID := map[int32]backend{}
	var waiters []int32
	for rows.Next() {
		var b backend
		if err := rows.Scan(&b.pid, &b.query, &b.blocking); err != nil {
			return nil, 0, err
		}
		byPID[b.pid] = b
		waiters = append(waiters, b.pid)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// rootsAt reports whether any blocking chain from pid ends
	// at the holder (the holder itself is idle, hence absent
	// here). Breadth-first over all reported blockers.
	rootsAt := func(pid int32) bool {
		seen := map[int32]bool{pid: true}
		queue := []int32{pid}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if cur == holderPID {
				return true
			}
			b, ok := byPID[cur]
			if !ok {
				continue
			}
			for _, next := range b.blocking {
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
		return false
	}
	var matched []int32
	for _, pid := range waiters {
		q := byPID[pid].query
		if strings.Contains(strings.ToUpper(q), strings.ToUpper(tableLike)) && rootsAt(pid) {
			matched = append(matched, pid)
		}
	}
	return matched, len(waiters), nil
}

func dumpActivity(t *testing.T, ctx context.Context, db *testkit.DB) {
	t.Helper()
	diag, err := db.Pool.Query(ctx, `SELECT pid, state, wait_event_type, wait_event,
		pg_blocking_pids(pid), left(query, 120) FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid()
		ORDER BY pid`)
	if err != nil {
		return
	}
	defer diag.Close()
	for diag.Next() {
		var pid int32
		var state, q string
		var wet, we *string
		var blocking []int32
		_ = diag.Scan(&pid, &state, &wet, &we, &blocking, &q)
		t.Logf("backend pid=%d state=%s wait=%v:%v blocking=%v query=%.100s",
			pid, state, strptr(wet), strptr(we), blocking, q)
	}
}

func strptr(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

// TestConcurrentSubmitSameKey gates two same-key Class D invokes
// behind a row lock so both must observe the empty run_id before
// either writes, then requires exactly one canonical workflow:
// one executor run, one record run ID, both handles naming it.
func TestConcurrentSubmitSameKey(t *testing.T) {
	const racers = 2
	const rounds = 3
	// One database for all rounds (fresh fixture + truncated tables
	// per round): opening a pool per round exhausts connections.
	db := newDB(t)
	for round := 0; round < rounds; round++ {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
		fx := newRedFixture(t, db)
		invID, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (string, error) {
			var id string
			err := tx.QueryRow(context.Background(), `INSERT INTO app.invocations
				(tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
				 status, trust_mode, effect_token)
				VALUES ($1, $2, '1.0.0', $3, 'key-x03-gate', '{}', 'running', 'autonomous', 'tok-x03')
				RETURNING id::text`,
				tenantA, probeID, probeTool).Scan(&id)
			return id, err
		})
		if err != nil {
			t.Fatal(err)
		}
		holder, release := rowGateHolder(t, db, tenantA, invID)
		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make([]error, racers)
		handles := make([]string, racers)
		for i := 0; i < racers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-x03-gate", credAgentA, contract.TrustAutonomous)
				if err != nil {
					errs[i] = err
					return
				}
				if out.Handle == nil {
					errs[i] = errNoHandle
					return
				}
				handles[i] = out.Handle.RunID
			}(i)
		}
		close(start)
		// Both racers must be blocked behind OUR holder on the
		// invocations write before either proceeds: at this
		// point both have observed the empty run_id and neither
		// has written. Release and join on every path so a gate
		// failure can never wedge teardown.
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", racers); !ok {
			release()
			wg.Wait()
			t.Fatalf("round %d: gate never formed; see activity dump above", round)
		}
		release()
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d racer %d: %v", round, i, err)
			}
		}
		if handles[0] != handles[1] {
			t.Fatalf("round %d: handles diverge %q vs %q (orphan workflow minted)", round, handles[0], handles[1])
		}
		fx.async.mu.Lock()
		n := len(fx.async.status)
		fx.async.mu.Unlock()
		if n != 1 {
			t.Fatalf("round %d: executor runs = %d, want exactly 1 (same-key orphan)", round, n)
		}
	}
}
