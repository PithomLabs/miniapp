// M-01 discriminating tests: predicated invocation transitions.
//
// RED/GREEN-COMPATIBLE SOURCE: uses only public service APIs, SQL
// fixtures, pg_stat_activity observation, and the shared gate
// helpers from x03_test.go plus the red fixture from x01x02_test.go
// (all present identically in the acceptance tree). The red run
// overlays exactly this file onto the frozen acceptance copy.
package runtime_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
)

// seedWaitingRunWithReject inserts a waiting Class D record with a
// submitted run and a decided-reject approval, returning the run ID.
func seedWaitingRunWithReject(t *testing.T, db *testkit.DB, fx redFixture, key string) string {
	t.Helper()
	runID := "run-m01-" + uuid.NewString()[:8]
	invID := uuid.NewString()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token, run_id)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', 'waiting_approval', 'confirm', 'tok-m01', $6)`,
			invID, tenantA, probeGatedID, probeGatedTool, key, runID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.async.mu.Lock()
	fx.async.status[runID] = contract.RunRunning
	fx.async.mu.Unlock()
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, []byte(`{}`))
	apprID, err := fx.approver.Submit(context.Background(), gatedApprovalRequest(invID, hash))
	if err != nil {
		t.Fatal(err)
	}
	// Decide directly (not via the service): the decision commits
	// without touching the invocation row, setting up the race.
	decider := approvals.Decider{
		Caller: contract.Caller{AgentID: "human-op",
			Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "human-op", TenantID: tenantA}},
		Roles: []string{"tenant-ops"},
	}
	if err := fx.approver.Decide(context.Background(), tenantA, apprID, decider, contract.DecisionReject, hash); err != nil {
		t.Fatal(err)
	}
	return runID
}

// TestPollSettleVsCancelRace gates a denial-settle poll against a
// Cancel on the same record: exactly one terminal wins and the
// loser answers truthfully. Red: both unpredicated writes succeed
// silently (denied terminal + cancel success), which is incoherent.
func TestPollSettleVsCancelRace(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
		fx := newRedFixture(t, db)
		key := "key-m01-settle"
		runID := seedWaitingRunWithReject(t, db, fx, key)
		var invID string
		_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
			err := tx.QueryRow(context.Background(),
				`SELECT id::text FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&invID)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
		holder, release := rowGateHolder(t, db, tenantA, invID)
		start := make(chan struct{})
		var wg sync.WaitGroup
		var pollErr, cancelErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, pollErr = invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, key, credAgentA, contract.TrustConfirm)
		}()
		go func() {
			defer wg.Done()
			<-start
			_, cancelErr = fx.svc.Cancel(context.Background(), credAgentA, runID, "m01 race")
		}()
		close(start)
		// Both sides must be blocked behind OUR holder on the
		// invocations write before either proceeds.
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 2); !ok {
			release()
			wg.Wait()
			t.Fatalf("round %d: gate never formed", round)
		}
		release()
		wg.Wait()
		pollDenied := pollErr != nil && strings.Contains(pollErr.Error(), "approval-denied")
		cancelOK := cancelErr == nil
		st := recordStatus(t, db, key)
		// Coherent outcomes only: poll-denied with Failed, or
		// cancel-won with Cancelled. Both succeeding is the
		// silent-overwrite defect.
		pollWon := pollDenied && st == contract.RunFailed && !cancelOK
		cancelWon := cancelOK && st == contract.RunCancelled && !pollDenied
		if !pollWon && !cancelWon {
			t.Fatalf("round %d: pollErr=%v cancelErr=%v status=%s (incoherent pair)", round, pollErr, cancelErr, st)
		}
	}
}

// TestStatusSyncVsCancelRace gates an executor-truth sync against
// a Cancel: exactly one terminal wins and the loser answers from
// fresh state. Red: both succeed silently (Status claims the
// synced terminal while Cancel claims cancellation).
func TestStatusSyncVsCancelRace(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
		fx := newRedFixture(t, db)
		runID := "run-m01-sync"
		invID := uuid.NewString()
		_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
			_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
				(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
				 status, trust_mode, effect_token, run_id)
				VALUES ($1, $2, $3, '1.0.0', $4, 'key-m01-sync', '{}', 'running', 'autonomous', 'tok-m01s', $5)`,
				invID, tenantA, probeID, probeTool, runID)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
		fx.async.mu.Lock()
		fx.async.status[runID] = contract.RunSucceeded
		fx.async.mu.Unlock()
		holder, release := rowGateHolder(t, db, tenantA, invID)
		start := make(chan struct{})
		var wg sync.WaitGroup
		var statusOut runtime.InvokeResult
		var statusErr, cancelErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			statusOut, statusErr = fx.svc.Status(context.Background(), credAgentA, invID)
		}()
		go func() {
			defer wg.Done()
			<-start
			_, cancelErr = fx.svc.Cancel(context.Background(), credAgentA, runID, "m01 sync race")
		}()
		close(start)
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 2); !ok {
			release()
			wg.Wait()
			t.Fatalf("round %d: gate never formed", round)
		}
		release()
		wg.Wait()
		if statusErr != nil {
			t.Fatalf("round %d: status err = %v (sync must answer, never error)", round, statusErr)
		}
		st := recordStatus(t, db, "key-m01-sync")
		syncWon := st == contract.RunSucceeded && statusOut.Status == contract.RunSucceeded && cancelErr != nil
		cancelWon := st == contract.RunCancelled && statusOut.Status == contract.RunCancelled && cancelErr == nil
		if !syncWon && !cancelWon {
			t.Fatalf("round %d: status=%s cancelErr=%v record=%s (incoherent pair)",
				round, statusOut.Status, cancelErr, st)
		}
	}
}

// TestSettleThenCancelRefused proves completion-wins-first
// sequentially: after a denial settles Failed, Cancel is refused
// without overwriting.
func TestSettleThenCancelRefused(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	runID := seedWaitingRunWithReject(t, db, fx, "key-m01-t4")
	_, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-m01-t4", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "approval-denied") {
		t.Fatalf("poll err = %v, want terminal approval-denied", err)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, runID, "late cancel"); err == nil {
		t.Fatal("Cancel after Failed settle succeeded; must refuse without overwriting")
	}
	if st := recordStatus(t, db, "key-m01-t4"); st != contract.RunFailed {
		t.Fatalf("record = %s, want Failed (preserved)", st)
	}
}

// TestPollSettleIdempotent proves terminal repeats never re-settle:
// the second poll answers from the recorded terminal without
// touching the row again (single approval-denied audit).
func TestPollSettleIdempotent(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	seedWaitingRunWithReject(t, db, fx, "key-m01-t5")
	for i := 0; i < 2; i++ {
		_, _ = invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-m01-t5", credAgentA, contract.TrustConfirm)
	}
	var denied int
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.audit_events WHERE action = 'approval-denied'`).Scan(&denied)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if denied != 1 {
		t.Fatalf("approval-denied audits = %d, want exactly 1 (no re-settle)", denied)
	}
	if st := recordStatus(t, db, "key-m01-t5"); st != contract.RunFailed {
		t.Fatalf("record = %s, want Failed", st)
	}
}

// TestCancelQuarantinedRefused pins the CR-05 invariant: Cancel on
// a quarantined (dead-lettered) invocation is refused without
// touching the row; exit is via operator resume/fail only.
func TestCancelQuarantinedRefused(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token, run_id)
			VALUES (gen_random_uuid(), $1, $2, '1.0.0', $3, 'key-m01-q', '{}', 'quarantined', 'confirm', 'tok-m01q', 'run-quar-1')`,
			tenantA, probeGatedID, probeGatedTool)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, "run-quar-1", "test"); err == nil ||
		!strings.Contains(err.Error(), "run-not-live") {
		t.Fatalf("Cancel err = %v, want run-not-live refusal", err)
	}
	if st := recordStatus(t, db, "key-m01-q"); st != contract.RunQuarantined {
		t.Fatalf("record = %s, want quarantined (preserved)", st)
	}
}

// TestConcurrentCancelSingleAudit gates two concurrent Cancels on
// the same live record: exactly one wins the transition and audits;
// the loser redelivers without a duplicate cancel audit. Red: both
// blind updates succeed and both audit (duplicate cancel record).
func TestConcurrentCancelSingleAudit(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
		fx := newRedFixture(t, db)
		runID := "run-m01-ccancel"
		invID := uuid.NewString()
		_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
			_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
				(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
				 status, trust_mode, effect_token, run_id)
				VALUES ($1, $2, $3, '1.0.0', $4, 'key-m01-ccancel', '{}', 'running', 'autonomous', 'tok-m01c', $5)`,
				invID, tenantA, probeID, probeTool, runID)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
		fx.async.mu.Lock()
		fx.async.status[runID] = contract.RunRunning
		fx.async.mu.Unlock()
		holder, release := rowGateHolder(t, db, tenantA, invID)
		start := make(chan struct{})
		var wg sync.WaitGroup
		var err1, err2 error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, err1 = fx.svc.Cancel(context.Background(), credAgentA, runID, "m01 c1")
		}()
		go func() {
			defer wg.Done()
			<-start
			_, err2 = fx.svc.Cancel(context.Background(), credAgentA, runID, "m01 c2")
		}()
		close(start)
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 2); !ok {
			release()
			wg.Wait()
			t.Fatalf("round %d: gate never formed", round)
		}
		release()
		wg.Wait()
		if err1 != nil || err2 != nil {
			t.Fatalf("round %d: err1=%v err2=%v (both cancels must succeed; loser redelivers)", round, err1, err2)
		}
		var cancels int
		_, err = store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
			err := tx.QueryRow(context.Background(),
				`SELECT count(*) FROM app.audit_events WHERE action = 'cancel' AND outcome = 'cancelled'`).Scan(&cancels)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
		if cancels != 1 {
			t.Fatalf("round %d: cancel audits = %d, want exactly 1 (loser must not re-audit)", round, cancels)
		}
		if st := recordStatus(t, db, "key-m01-ccancel"); st != contract.RunCancelled {
			t.Fatalf("round %d: record = %s, want cancelled", round, st)
		}
	}
}

func recordStatus(t *testing.T, db *testkit.DB, key string) contract.RunStatus {
	t.Helper()
	st, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(),
			`SELECT status FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return contract.RunStatus(st)
}
