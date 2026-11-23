package runtime_test

// Prompt28 workstream B, case B5 — restoration versus dispatch,
// both forced orders. Restoration is the submit-time adoptRunID
// path (pending→running attach for a run-less record); dispatch
// is effect Perform/authorize on the same invocation. Row-gate
// staging with waiter observation forces each order; FIFO grant
// order plus exact assertions prove convergence (never stuck
// pending, never a pre-adoption send).

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

func p28ReadInvocation(t *testing.T, pool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, tenant, invID string) (status, runID, trust string) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `SELECT status, run_id, trust_mode
		FROM app.invocations WHERE id = $1::uuid AND tenant_id = $2`,
		invID, tenant).Scan(&status, &runID, &trust)
	if err != nil {
		t.Fatal(err)
	}
	return status, runID, trust
}

// Order A — restore first: adoption completes, then effect
// dispatch authorizes and sends exactly once on the adopted run.
func TestP28RestoreThenDispatch(t *testing.T) {
	db := newDB(t)
	p22Truncate(t, db)
	fx := newFixture(t, db)
	key := "key-p28-restore-first"
	invID := p22SeedRowTool(t, db, key, probeID, probeTool, string(contract.RunPending), "")

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, key, credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("restore invoke: %v", err)
	}
	if out.Status != contract.RunRunning || out.Handle == nil || out.Handle.RunID == "" {
		t.Fatalf("restore = %+v, want running with adopted run", out)
	}
	st, run, _ := p28ReadInvocation(t, db.Pool, tenantA, invID)
	if st != string(contract.RunRunning) || run == "" {
		t.Fatalf("record = (%s, %s), want (running, adopted run)", st, run)
	}
	tok := effectToken(t, db, tenantA, invID)
	rec, err := fx.effects.Perform(context.Background(), tenantA,
		effIntent(invID, tok, "p28restore", mockcap.ModeAccept))
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("dispatch = %+v %v, want CONFIRMED", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28restore"); n != 1 {
		t.Errorf("sends = %d, want exactly 1", n)
	}
}

// Order B — dispatch queued first: the pre-claim liveness read
// observes pending and refuses (inserting nothing); restoration
// then adopts; the retry authorizes and sends. The pending
// window never strands and never sends early.
func TestP28DispatchQueuedBeforeRestore(t *testing.T) {
	db := newDB(t)
	p22Truncate(t, db)
	fx := newFixture(t, db)
	ctx := context.Background()
	key := "key-p28-dispatch-first"
	invID := p22SeedRowTool(t, db, key, probeID, probeTool, string(contract.RunPending), "")
	tok := effectToken(t, db, tenantA, invID)
	intent := effIntent(invID, tok, "p28dispatch", mockcap.ModeAccept)

	holder, release := rowGateHolder(t, db, tenantA, invID)
	type outcome struct {
		err error
	}
	performDone := make(chan outcome, 1)
	go func() {
		_, err := fx.effects.Perform(ctx, tenantA, intent)
		performDone <- outcome{err}
	}()
	// Perform's locked liveness read queues first...
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
		release()
		t.Fatal("perform liveness gate never formed")
	}
	invokeDone := make(chan outcome, 1)
	go func() {
		_, err := invoke(fx.svc, probeID, probeTool, `{}`, key, credAgentA, contract.TrustAutonomous)
		invokeDone <- outcome{err}
	}()
	// ...restoration's adopt queues behind it.
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 2); !ok {
		release()
		<-performDone
		<-invokeDone
		t.Fatal("adopt gate never formed behind liveness")
	}
	release()
	pOut := <-performDone
	iOut := <-invokeDone
	// FIFO: liveness reads pending and refuses pre-claim.
	if !errors.Is(pOut.err, effects.ErrInvocationCancelled) {
		t.Errorf("dispatch err = %v, want pending-ineligible refusal", pOut.err)
	}
	var refused int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM app.effect_records
		WHERE business_op_id = $1`, intent.BusinessOpID).Scan(&refused); err != nil {
		t.Fatal(err)
	}
	if refused != 0 {
		t.Errorf("effect rows = %d, want 0 (refused claims insert nothing)", refused)
	}
	// Restoration adopted behind it.
	if iOut.err != nil {
		t.Fatalf("restore invoke: %v", iOut.err)
	}
	st, run, _ := p28ReadInvocation(t, db.Pool, tenantA, invID)
	if st != string(contract.RunRunning) || run == "" {
		t.Fatalf("record = (%s, %s), want (running, adopted run)", st, run)
	}
	// Retry converges: authorize + send exactly once.
	rec, err := fx.effects.Perform(ctx, tenantA, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("retry = %+v %v, want CONFIRMED", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28dispatch"); n != 1 {
		t.Errorf("sends = %d, want exactly 1", n)
	}
	if rec.AttemptID == "" {
		t.Errorf("confirmed without a framework attempt epoch")
	}
}
