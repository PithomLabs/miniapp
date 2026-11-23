package runtime_test

// P24-001 (Prompt 25): terminal-invocation withdrawal must persist
// atomically with exactly one truthful audit. These tests re-implement
// POC-1's deterministic staging as committed regressions: the
// cancel-first leg forces checkLive(live) → cancel-commit →
// authorize-terminal through FIFO lock staging plus a speculative
// claim-INSERT gate; the authorize-first leg pins the opposite order
// through a provider gate channel.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

// p25GateChannel blocks inside Send until released, so tests can hold
// an authorized attempt open deterministically. It mirrors the
// effects-package gate used by the concurrency suite.
type p25GateChannel struct {
	entered chan struct{}
	release chan struct{}
	sends   int
}

func (c *p25GateChannel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	c.sends++
	close(c.entered)
	select {
	case <-c.release:
		return effects.Receipt{ProviderRequestID: "p25-r1", ProviderReply: "p25-ok"}, nil
	case <-ctx.Done():
		return effects.Receipt{}, ctx.Err()
	}
}

// p25StagePool opens a dedicated pool for staging transactions and
// lock-queue polling, so gate holders never starve the fixture pool.
func p25StagePool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 4
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// p25WaitLockWaiter blocks until a backend other than the poller is
// actively waiting on a lock while running a query containing
// needle. The caller starts waiters in a known order, so observing
// each waiter before starting the next proves FIFO queue order.
func p25WaitLockWaiter(t *testing.T, pool *pgxpool.Pool, needle, what string) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND pid <> pg_backend_pid()
			AND state = 'active' AND wait_event_type = 'Lock'
			AND position($1 in query) > 0`, needle).Scan(&n)
		if err != nil {
			t.Fatalf("pg_stat_activity poll (%s): %v", what, err)
		}
		if n > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s (needle %q) to queue on its lock", what, needle)
}

func p25RowState(t *testing.T, pool *pgxpool.Pool, tenant, op string) contract.EffectState {
	t.Helper()
	st, err := store.WithTenant(context.Background(), pool, tenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(), `SELECT state FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, tenant).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatalf("row state read: %v", err)
	}
	return contract.EffectState(st)
}

func p25AuditCount(t *testing.T, pool *pgxpool.Pool, tenant, op, action string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), pool, tenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = $2 AND details->>'business_op_id' = $3`,
			tenant, action, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatalf("audit count read: %v", err)
	}
	return n
}

func p25InvocationStatus(t *testing.T, pool *pgxpool.Pool, tenant, inv string) contract.RunStatus {
	t.Helper()
	st, err := store.WithTenant(context.Background(), pool, tenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(), `SELECT status FROM app.invocations
			WHERE id = $1::uuid AND tenant_id = $2`, inv, tenant).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatalf("invocation status read: %v", err)
	}
	return contract.RunStatus(st)
}

// TestP25POC1CancelFirstWithdraws forces cancel-before-authorize:
// Perform's pre-claim liveness read queues first behind a held
// invocation-row lock, Cancel's UPDATE queues behind it, FIFO grants
// the liveness read first (it passes live), then the speculative
// claim-INSERT gate holds Perform until Cancel has fully committed.
// authorize must then withdraw PENDING → CANCELLED in the same
// transaction as its effect-withdrawn audit, refuse, and commit.
func TestP25POC1CancelFirstWithdraws(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	ctx := context.Background()

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-p25-poc1", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	tok := effectToken(t, db, tenantA, out.InvocationID)
	intent := effIntent(out.InvocationID, tok, "p25poc1", mockcap.ModeAccept)
	op := intent.BusinessOpID

	stage := p25StagePool(t, db.DSN)

	// Speculative-insertion gate: the stale PENDING row exists only
	// inside this open transaction, so claim's INSERT ... ON
	// CONFLICT blocks until the test commits — strictly after
	// Cancel has completed. Payload/capability mirror the intent so
	// the post-claim operation check passes.
	stale, err := stage.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer stale.Rollback(ctx)
	if _, err := stale.Exec(ctx, store.TenantContextSQL(), tenantA); err != nil {
		t.Fatal(err)
	}
	if _, err := stale.Exec(ctx, `INSERT INTO app.effect_records
		(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
		 capability_id, operation, payload, state, updated_at)
		VALUES ($1, $2, 'a-stale-p25poc1', '', $3, 'mock', 'send', $4, 'PENDING',
			now() - interval '1 hour')`,
		op, tenantA, out.InvocationID, string(intent.Operation.Payload)); err != nil {
		t.Fatal(err)
	}

	// FIFO row gate: hold the invocation row; Perform's locked
	// liveness read queues first, Cancel's UPDATE queues behind.
	gate, err := stage.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Rollback(ctx)
	if _, err := gate.Exec(ctx, store.TenantContextSQL(), tenantA); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := gate.QueryRow(ctx, `SELECT id::text FROM app.invocations
		WHERE id = $1::uuid FOR UPDATE`, out.InvocationID).Scan(&held); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		err error
	}
	performDone := make(chan outcome, 1)
	go func() {
		_, err := fx.effects.Perform(ctx, tenantA, intent)
		performDone <- outcome{err}
	}()
	p25WaitLockWaiter(t, stage, "SELECT status FROM app.invocations", "Perform liveness read")

	type cancelOutcome struct {
		res runtime.CancelResult
		err error
	}
	cancelDone := make(chan cancelOutcome, 1)
	go func() {
		res, err := fx.svc.Cancel(ctx, credAgentA, out.Handle.RunID, "p25-poc1")
		cancelDone <- cancelOutcome{res, err}
	}()
	p25WaitLockWaiter(t, stage, "UPDATE app.invocations SET status", "Cancel UPDATE")

	// Release: FIFO grants the waiter first (liveness passes live),
	// then Cancel commits. Join Cancel before opening the
	// speculative gate so authorize deterministically reads terminal.
	if err := gate.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case o := <-cancelDone:
		if o.err != nil {
			t.Fatalf("cancel: %v", o.err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("cancel never completed after gate release")
	}
	if err := stale.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case o := <-performDone:
		if !errors.Is(o.err, effects.ErrInvocationCancelled) {
			t.Fatalf("perform err = %v, want ErrInvocationCancelled", o.err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("perform never completed after speculative release")
	}

	// Forced-order proof: the winning transition persisted with
	// exactly one truthful audit; nothing attempted or sent.
	if st := p25InvocationStatus(t, db.Pool, tenantA, out.InvocationID); st != contract.RunCancelled {
		t.Errorf("invocation status = %s, want cancelled", st)
	}
	if st := p25RowState(t, db.Pool, tenantA, op); st != contract.EffectCancelled() {
		t.Errorf("row state after terminal refusal = %s, want CANCELLED", st)
	}
	if n := p25AuditCount(t, db.Pool, tenantA, op, "effect-withdrawn"); n != 1 {
		t.Errorf("effect-withdrawn audits = %d, want exactly 1", n)
	}
	if n := p25AuditCount(t, db.Pool, tenantA, op, "effect-refused"); n != 0 {
		t.Errorf("effect-refused audits = %d, want 0 before any repeat", n)
	}
	if n := mockcap.SentCount(t, db.Pool, op); n != 0 {
		t.Errorf("provider sends = %d, want 0 (never ATTEMPTING)", n)
	}

	// Repeat: refused, never resurrected, never re-withdrawn.
	if _, err := fx.effects.Perform(ctx, tenantA, intent); !errors.Is(err, effects.ErrInvocationCancelled) {
		t.Errorf("repeat perform err = %v, want ErrInvocationCancelled", err)
	}
	if st := p25RowState(t, db.Pool, tenantA, op); st != contract.EffectCancelled() {
		t.Errorf("row state after repeat = %s, want CANCELLED", st)
	}
	if n := p25AuditCount(t, db.Pool, tenantA, op, "effect-withdrawn"); n != 1 {
		t.Errorf("effect-withdrawn audits after repeat = %d, want exactly 1", n)
	}
	// The plain repeat refuses at the pre-claim liveness gate
	// (never reaching authorize), which audits nothing — unchanged
	// behavior. Authorize-reaching repeats audit effect-refused;
	// see effects/p25_p24001_internal_test.go.
	if n := p25AuditCount(t, db.Pool, tenantA, op, "effect-refused"); n != 0 {
		t.Errorf("effect-refused audits after repeat = %d, want 0 (silent pre-claim refusal)", n)
	}
}

// TestP25POC1AuthorizeFirstInFlight pins the opposite forced order:
// authorization commits ATTEMPTING first (proven by Send entry),
// Cancel then reports the effect in-flight, and releasing the
// provider lets the authorized send complete — cancellation never
// pretends to recall it.
func TestP25POC1AuthorizeFirstInFlight(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	ctx := context.Background()

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-p25-poc1b", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	tok := effectToken(t, db, tenantA, out.InvocationID)
	intent := effIntent(out.InvocationID, tok, "p25poc1b", mockcap.ModeAccept)
	op := intent.BusinessOpID

	gate := &p25GateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	gated := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": gate},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: fx.reg, Grants: fx.policy},
		Registry:  fx.reg, Grants: fx.policy, Approver: fx.approver, Auditor: fx.aud,
	})

	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		rec, err := gated.Perform(ctx, tenantA, intent)
		done <- outcome{rec, err}
	}()
	select {
	case <-gate.entered:
	case <-time.After(60 * time.Second):
		t.Fatal("provider send never entered (authorize did not commit ATTEMPTING)")
	}
	// Send entry proves the authorization transaction committed:
	// sends happen only after commit, outside that transaction.
	if st := p25RowState(t, db.Pool, tenantA, op); st != contract.EffectAttempting {
		t.Fatalf("row state while send held = %s, want ATTEMPTING", st)
	}

	res, err := fx.svc.Cancel(ctx, credAgentA, out.Handle.RunID, "p25-poc1b")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	found := false
	for _, id := range res.InFlight {
		if id == op {
			found = true
		}
	}
	if !found {
		t.Errorf("cancel in-flight = %v, want it to contain %s", res.InFlight, op)
	}

	close(gate.release)
	select {
	case o := <-done:
		if o.err != nil {
			t.Fatalf("perform err = %v, want nil (authorized send completes)", o.err)
		}
		if o.rec.State != contract.EffectConfirmed {
			t.Errorf("final state = %s, want CONFIRMED (no recall of an authorized send)", o.rec.State)
		}
		if o.rec.ProviderRequest == "" {
			t.Errorf("confirmed without provider receipt (recall pretense)")
		}
	case <-time.After(60 * time.Second):
		t.Fatal("perform never completed after release")
	}
	if gate.sends != 1 {
		t.Errorf("provider sends = %d, want exactly 1", gate.sends)
	}
}
