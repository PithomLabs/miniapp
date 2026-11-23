package effects_test

// P24-003 (Prompt 25): Settle must never take the invocation lock —
// the §4.2 row is "5 effect" only. These tests force both
// lock-held-first orders with statement timeouts: with the
// invocation row held externally, Settle must still complete
// (proving the 5 → 1 edge is gone); with the effect row held,
// Settle must serialize on it and converge after release (pinning
// the unchanged effect-row footprint).

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
)

// p25SettleSvc builds a Settle-capable service on the given pool,
// mirroring newAuthService's authorization wiring.
func p25SettleSvc(t *testing.T, pool *pgxpool.Pool) *effects.Service {
	t.Helper()
	reg := registry.New()
	testkit.RegisterClassATool(t, reg, authzMiniapp, authzTool, "", nil,
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		})
	return effects.New(effects.Config{Pool: pool,
		Channels:  map[string]effects.Channel{},
		Canceller: allowCancellation{},
		Policy:    allowPolicy{},
		Registry:  reg,
		Grants:    authzPolicy(),
		Approver:  approvals.New(pool),
		Auditor:   audit.New(pool, nil),
	})
}

// p25TimeoutPool clones the DSN with a statement timeout, so a
// lock-waiting backend fails boundedly instead of hanging the suite.
func p25TimeoutPool(t *testing.T, dsn, timeout string) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 2
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func p25SeedUnknown(t *testing.T, db *testkit.DB, op, inv string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p25s', '', $3, 'mock', 'send', '{}', 'UNKNOWN')`,
			op, wsTenant, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// p25LockWaiter reports whether any backend besides the poller is
// lock-waiting on a query containing needle, with the query text.
func p25LockWaiter(pool *pgxpool.Pool, needle string) (bool, string) {
	var found bool
	var query string
	_ = pool.QueryRow(context.Background(), `SELECT true, query FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid()
		AND state = 'active' AND wait_event_type = 'Lock'
		AND position($1 in query) > 0 LIMIT 1`, needle).Scan(&found, &query)
	return found, query
}

// TestP25SettleTakesNoInvocationLock holds the invocation row and
// runs Settle against it: Settle must complete (its attribution
// read takes no lock), with correct outcome and attribution. A
// Settle that waits on the invocation lock fails loudly.
func TestP25SettleTakesNoInvocationLock(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	seedLiveInvocation(t, db, wsTenant, "p25s1")
	inv := authzInvID(wsTenant, "p25s1")
	op := wsTenant + ":p25s1"
	p25SeedUnknown(t, db, op, inv)

	svc := p25SettleSvc(t, p25TimeoutPool(t, db.DSN, "5s"))

	holder, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	if _, err := holder.Exec(ctx, store.TenantContextSQL(), wsTenant); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := holder.QueryRow(ctx, `SELECT id::text FROM app.invocations
		WHERE id = $1::uuid FOR UPDATE`, inv).Scan(&held); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		rec, err := svc.Settle(ctx, wsTenant, op, contract.EffectNotApplied, "p25-note", "p25-op")
		done <- outcome{rec, err}
	}()

	// Watch for the smoking gun (an invocation-lock waiter) while
	// awaiting completion; backstop bounds a wedged run.
	var waited bool
	var waitQuery string
	deadline := time.Now().Add(30 * time.Second)
	var res outcome
completed:
	for {
		select {
		case res = <-done:
			break completed
		default:
		}
		if w, q := p25LockWaiter(db.Pool, "FROM app.invocations"); w {
			waited = true
			waitQuery = q
		}
		if time.Now().After(deadline) {
			t.Fatal("settle neither completed nor was observable within 30s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if waited {
		t.Errorf("settle waited on the invocation lock (5 -> 1 edge present): %q", waitQuery)
	}
	if res.err != nil {
		t.Fatalf("settle err = %v, want nil (invocation lock must not block settle)", res.err)
	}
	if res.rec.State != contract.EffectNotApplied {
		t.Errorf("settled state = %s, want NOT_APPLIED", res.rec.State)
	}
	if res.rec.ProviderReply != "p25-note" {
		t.Errorf("settled note = %q, want p25-note", res.rec.ProviderReply)
	}
	var miniapp, tool string
	err = db.Pool.QueryRow(ctx, `SELECT miniapp_id, tool FROM app.audit_events
		WHERE tenant_id = $1 AND action = 'effect-settled'
		AND details->>'business_op_id' = $2`, wsTenant, op).Scan(&miniapp, &tool)
	if err != nil {
		t.Fatalf("settle audit read: %v", err)
	}
	if miniapp != authzMiniapp || tool != authzTool {
		t.Errorf("settle attribution = %s/%s, want %s/%s (lock-free read must attribute correctly)",
			miniapp, tool, authzMiniapp, authzTool)
	}
}

// TestP25SettleSerializesOnEffectRow holds the effect row and runs
// Settle: it must block on the effect write (pinning the unchanged
// "5 only" footprint), then converge once released.
func TestP25SettleSerializesOnEffectRow(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	seedLiveInvocation(t, db, wsTenant, "p25s2")
	inv := authzInvID(wsTenant, "p25s2")
	op := wsTenant + ":p25s2"
	p25SeedUnknown(t, db, op, inv)

	svc := p25SettleSvc(t, db.Pool)

	holder, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	if _, err := holder.Exec(ctx, store.TenantContextSQL(), wsTenant); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := holder.QueryRow(ctx, `SELECT business_op_id FROM app.effect_records
		WHERE business_op_id = $1 FOR UPDATE`, op).Scan(&held); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		rec, err := svc.Settle(ctx, wsTenant, op, contract.EffectConfirmed, "p25-note2", "p25-op")
		done <- outcome{rec, err}
	}()

	// The effect-write wait must appear (both trees): Settle still
	// takes lock 5 first.
	deadline := time.Now().Add(10 * time.Second)
	seen := false
	for time.Now().Before(deadline) {
		select {
		case o := <-done:
			t.Fatalf("settle completed while the effect row was held: %+v err=%v", o.rec, o.err)
		default:
		}
		if w, _ := p25LockWaiter(db.Pool, "UPDATE app.effect_records"); w {
			seen = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !seen {
		t.Fatal("settle never queued on the held effect row within 10s")
	}
	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case o := <-done:
		if o.err != nil {
			t.Fatalf("settle err = %v, want nil after release", o.err)
		}
		if o.rec.State != contract.EffectConfirmed {
			t.Errorf("settled state = %s, want CONFIRMED", o.rec.State)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("settle never completed after release")
	}
}
