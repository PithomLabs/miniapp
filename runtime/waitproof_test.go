// Rule 3 entry-path proofs for poll and decision (green-only).
// First-submission and replay wait proofs run against the real
// probe workflow in probe/classd; these two pin the remaining
// entry paths at the service boundary with the fake executor.
package runtime_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

func invocationIDForKey(t *testing.T, db *testkit.DB, key string) string {
	t.Helper()
	id, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(),
			`SELECT id::text FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// TestPollWhilePendingSubmitsNothing proves the poll entry path
// obeys workflow-start permission: polling a pending approval
// returns waiting without submitting another run or sending.
func TestPollWhilePendingSubmitsNothing(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-poll-wait", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != contract.RunWaitingApproval {
		t.Fatalf("invoke = %+v; want waiting", out)
	}
	fx.async.mu.Lock()
	before := fx.async.submitCalls
	fx.async.mu.Unlock()
	for i := 0; i < 3; i++ {
		out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-poll-wait", credAgentA, contract.TrustConfirm)
		if err != nil {
			t.Fatalf("poll %d err = %v, want waiting", i, err)
		}
		if out.Status != contract.RunWaitingApproval {
			t.Fatalf("poll %d = %+v; want waiting", i, out)
		}
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if fx.async.submitCalls != before {
		t.Fatalf("submitCalls = %d, want %d (poll must not submit)", fx.async.submitCalls, before)
	}
}

// TestRejectNeverSubmitsMissingRun proves the decision entry path:
// rejecting an approval whose run was never submitted settles
// without submitting anything.
func TestRejectNeverSubmitsMissingRun(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	fx.async.failNextSubmit = true
	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-rej-nosub", credAgentA, contract.TrustConfirm); err == nil {
		t.Fatal("park-time submit succeeded despite injected failure")
	}
	// No run exists; the rejection must settle without creating one.
	fx.async.mu.Lock()
	fx.async.failNextSubmit = false
	before := fx.async.submitCalls
	fx.async.mu.Unlock()
	// Resolve the parked approval directly (same-key re-invoke
	// would poll; the decision path is what this test drives).
	invID := invocationIDForKey(t, db, "key-rej-nosub")
	appr, err := fx.approver.ByInvocation(context.Background(), tenantA, invID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fx.svc.DecideApproval(context.Background(), credDecide, appr.ID, contract.DecisionReject, appr.IntentHash); err != nil {
		t.Fatalf("decide err = %v, want nil (nothing to notify, decision stands)", err)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if fx.async.submitCalls != before {
		t.Fatalf("submitCalls = %d, want %d (reject must not submit)", fx.async.submitCalls, before)
	}
}
