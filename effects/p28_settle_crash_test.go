package effects_test

// Prompt28 workstream E — Settle crash scenarios (atomic
// settlement architecture):
//  1. Crash before commit: genuine SIGKILL queued on the
//     settlement UPDATE; nothing survives; retry settles.
//  2. Commit-ok/lost-response: fault-modeled response loss;
//     the retry discovers the committed outcome (idempotent).
//  3. Identical repeat under crash/retry: idempotent success
//     with truthful audit backfill.
//  4. Conflicting second settlement: covered by
//     TestP28ConcurrentSettleSingleWinner + TestP28SettleHammerConverges
//     (first decision wins, conflict deterministic, audit once).

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/fault"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

func TestEffectsSettleWorker(t *testing.T) {
	if !testkit.WorkerMode(workerEnvKey) {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": mockcap.New(db.Pool)}, allowCancellation{}, allowPolicy{})
	op := os.Getenv("MINIAPP_EFFECTS_OP")
	_, _ = svc.Settle(context.Background(), wsTenant, wsTenant+":"+op,
		contract.EffectNotApplied, "p28-crash-note", "p28-op")
}

func p28SeedUnknown(t *testing.T, db *testkit.DB, op, inv string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p28settle', '', $3, 'mock', 'send', '{}', 'UNKNOWN')`,
			wsTenant+":"+op, wsTenant, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func p28SettledAudits(t *testing.T, db *testkit.DB, op string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-settled'
			AND details->>'business_op_id' = $2`, wsTenant, wsTenant+":"+op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Scenario 1 — genuine kill queued on the settlement UPDATE:
// no partial state or audit survives; the retry settles.
func TestP28SettleCrashBeforeCommit(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	ctx := context.Background()
	seedLiveInvocation(t, db, wsTenant, "inv-p28s1")
	p28SeedUnknown(t, db, "p28s1", authzInvID(wsTenant, "inv-p28s1"))
	release := p28HoldEffectRow(t, db, "p28s1")
	defer release()
	w := testkit.SpawnWorker(t, "TestEffectsSettleWorker$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28s1",
	})
	// The worker's loadTx reads UNKNOWN (no lock), then its
	// predicated UPDATE queues behind the holder.
	p28WaitLockWaiter(t, db.Pool, "UPDATE app.effect_records", "settlement UPDATE queue")
	testkit.Kill9(t, w)
	release()
	if st, _ := effectState(t, db, wsTenant, wsTenant+":p28s1"); st != contract.EffectUnknown {
		t.Fatalf("post-kill state = %s, want UNKNOWN (nothing committed)", st)
	}
	if n := p28SettledAudits(t, db, "p28s1"); n != 0 {
		t.Fatalf("settled audits = %d, want 0", n)
	}
	svc := p28Svc(t, db)
	rec, err := svc.Settle(ctx, wsTenant, wsTenant+":p28s1", contract.EffectNotApplied, "p28-crash-note", "p28-op")
	if err != nil || rec.State != contract.EffectNotApplied {
		t.Fatalf("retry = %+v %v, want NOT_APPLIED", rec, err)
	}
	if n := p28SettledAudits(t, db, "p28s1"); n != 1 {
		t.Errorf("settled audits = %d, want exactly 1", n)
	}
}

// Scenario 2 — commit succeeds but the response is lost: the
// fault models the lost response after a committed settle; the
// retry discovers the committed outcome idempotently. A genuine
// kill adds no observable distinction past the commit point
// (durability is the database's guarantee, proven by scenario
// 1's complement), so fault injection is the honest mechanism.
func TestP28SettleCommitLostResponse(t *testing.T) {
	svc, db := setupMock(t)
	seedLiveInvocation(t, db, wsTenant, "inv-p28s2")
	p28SeedUnknown(t, db, "p28s2", authzInvID(wsTenant, "inv-p28s2"))
	ctx := context.Background()
	inj := fault.New()
	inj.FailAt("settle-operator-response", 1, errFault)
	svc.TestFault = inj
	_, err := svc.Settle(ctx, wsTenant, wsTenant+":p28s2", contract.EffectFailed, "evidence", "p28-op")
	if !errors.Is(err, errFault) {
		t.Fatalf("Settle err = %v, want injected response loss", err)
	}
	// The outcome stands despite the lost response.
	if st, _ := effectState(t, db, wsTenant, wsTenant+":p28s2"); st != contract.EffectFailed {
		t.Fatalf("state = %s, want FAILED (committed)", st)
	}
	if n := p28SettledAudits(t, db, "p28s2"); n != 1 {
		t.Fatalf("settled audits = %d, want 1", n)
	}
	svc.TestFault = nil
	rec, err := svc.Settle(ctx, wsTenant, wsTenant+":p28s2", contract.EffectFailed, "evidence", "p28-op")
	if err != nil || rec.State != contract.EffectFailed {
		t.Fatalf("retry = %+v %v, want idempotent FAILED", rec, err)
	}
	if n := p28SettledAudits(t, db, "p28s2"); n != 1 {
		t.Errorf("settled audits after retry = %d, want still 1", n)
	}
}

// Scenario 3 — identical repeat under crash/retry: success is
// idempotent, and a missing audit row is backfilled truthfully
// (the crash-between shape this covers: state committed, audit
// row lost before write).
func TestP28SettleRepeatBackfillsAudit(t *testing.T) {
	svc, db := setupMock(t)
	seedLiveInvocation(t, db, wsTenant, "inv-p28s3")
	p28SeedUnknown(t, db, "p28s3", authzInvID(wsTenant, "inv-p28s3"))
	ctx := context.Background()
	if _, err := svc.Settle(ctx, wsTenant, wsTenant+":p28s3", contract.EffectConfirmed, "evidence", "p28-op"); err != nil {
		t.Fatal(err)
	}
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `DELETE FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-settled'
			AND details->>'business_op_id' = $2`, wsTenant, wsTenant+":p28s3")
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := svc.Settle(ctx, wsTenant, wsTenant+":p28s3", contract.EffectConfirmed, "evidence", "p28-op")
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("repeat = %+v %v, want idempotent CONFIRMED", rec, err)
	}
	if n := p28SettledAudits(t, db, "p28s3"); n != 1 {
		t.Errorf("settled audits = %d, want exactly 1 (backfilled once)", n)
	}
}
