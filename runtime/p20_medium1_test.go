// MEDIUM-1 discriminating tests: CR-05 run-identity integrity.
//
// INTERNAL TEST SOURCE (package runtime): drives store ops and the
// submit choke directly. Red/green-compatible: the MEDIUM-1 red
// tree is the pre-fix working copy, so every referenced API exists
// identically on both sides (the fix changes only predicate SQL,
// one ignored write, and service branching — no signatures). API-
// level deterministic staging of the clobber is unforceable (see
// the Phase 0 reachability row); these tests pin the boundary that
// owns the guarantee with committed, ordered operations.
package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

// p20SeedRow inserts an invocation in the given state. runID "" means unbound.
func p20SeedRow(t *testing.T, db *testkit.DB, key, status, runID string) string {
	t.Helper()
	id := uuid.NewString()
	_, err := store.WithTenant(context.Background(), db.Pool, "tenant-a", func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token, run_id)
			VALUES ($1, 'tenant-a', 'm', '1.0.0', 't.t', $2, '{}', $3, 'confirm', 'tok-p20m1', $4)`,
			id, key, status, runID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func p20RunID(t *testing.T, db *testkit.DB, id string) string {
	t.Helper()
	run, err := store.WithTenant(context.Background(), db.Pool, "tenant-a", func(tx pgx.Tx) (string, error) {
		var r string
		err := tx.QueryRow(context.Background(),
			`SELECT run_id FROM app.invocations WHERE id = $1::uuid`, id).Scan(&r)
		return r, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return run
}

// TestP20SettlePreservesAdoptedRun proves the stale-settle leg: an
// adoption commits run R, then a settlement carrying a stale empty
// run wins on the still-waiting row. R must survive (CR-05). Red:
// failIfWaiting writes the stale value and clobbers R.
func TestP20SettlePreservesAdoptedRun(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx := context.Background()
	id := p20SeedRow(t, db, "m1-clobber", string(contract.RunWaitingApproval), "")
	adopted, err := s.adoptRunID(ctx, "tenant-a", id, "run-p20m1")
	if err != nil || !adopted {
		t.Fatalf("adoptRunID(waiting) = %v, %v; want true, nil (both trees)", adopted, err)
	}
	won, err := s.failIfWaiting(ctx, "tenant-a", id, "", "approval expired")
	if err != nil || !won {
		t.Fatalf("failIfWaiting(waiting) = %v, %v; want true, nil (both trees)", won, err)
	}
	got, err := s.getByID(ctx, "tenant-a", id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.RunID != "run-p20m1" {
		t.Fatalf("run_id = %q, want run-p20m1 (stale settle clobbered the adoption)", got.RunID)
	}
}

// TestP20AdoptRefusesTerminal proves terminal and quarantined
// records can never acquire a run through adoption. Red:
// adoptRunID has no status predicate and wins. A live control
// proves the predicate still permits legitimate adoption.
func TestP20AdoptRefusesTerminal(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx := context.Background()
	for _, tc := range []struct {
		key    string
		status contract.RunStatus
	}{
		{"m1-term", contract.RunFailed},
		{"m1-quar", contract.RunQuarantined},
	} {
		id := p20SeedRow(t, db, tc.key, string(tc.status), "")
		adopted, err := s.adoptRunID(ctx, "tenant-a", id, "run-p20m1x")
		if err != nil {
			t.Fatalf("adoptRunID(%s) errored: %v", tc.status, err)
		}
		if adopted {
			t.Fatalf("adoptRunID(%s) won; terminal/quarantined records must refuse adoption", tc.status)
		}
		if r := p20RunID(t, db, id); r != "" {
			t.Fatalf("adoptRunID(%s) left run_id %q, want empty", tc.status, r)
		}
	}
	live := p20SeedRow(t, db, "m1-live", string(contract.RunWaitingApproval), "")
	if adopted, err := s.adoptRunID(ctx, "tenant-a", live, "run-p20m1y"); err != nil || !adopted {
		t.Fatalf("adoptRunID(waiting) = %v, %v; want true, nil (control)", adopted, err)
	}
	pend := p20SeedRow(t, db, "m1-pend", string(contract.RunPending), "")
	if adopted, err := s.adoptRunID(ctx, "tenant-a", pend, "run-p20m1z"); err != nil || !adopted {
		t.Fatalf("adoptRunID(pending) = %v, %v; want true, nil (restoration control)", adopted, err)
	}
}

// p20Async counts workflow submissions; all other operations are inert.
type p20Async struct {
	mu      sync.Mutex
	submits int
}

func (f *p20Async) Submit(_ context.Context, req executor.Request) (contract.RunHandle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submits++
	return contract.RunHandle{RunID: req.RunID}, nil
}

func (f *p20Async) Status(_ context.Context, _ string) (contract.RunStatus, error) {
	return contract.RunRunning, nil
}

func (f *p20Async) Signal(_ context.Context, _ string, _ contract.Signal) error { return nil }

func (f *p20Async) Cancel(_ context.Context, _ string, _ string) error { return nil }

func (f *p20Async) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.submits
}

func p20ChokeFixture(t *testing.T, db *testkit.DB) (*Service, *p20Async) {
	t.Helper()
	reg := registry.New()
	async := &p20Async{}
	policy := auth.Policy{}
	effSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: reg, Grants: policy},
	})
	svc, err := New(Config{
		Registry: reg, Policy: policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, reg), Async: async,
		Approver: approvals.New(db.Pool), Auditor: audit.New(db.Pool, nil), Effects: effSvc,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, async
}

func p20Grant(t *testing.T, db *testkit.DB, invID string) {
	t.Helper()
	ctx := context.Background()
	hash := IntentHash("m", "1.0.0", "t.t", []byte(`{}`))
	appr := approvals.New(db.Pool)
	id, err := appr.Submit(ctx, approvals.Request{
		TenantID: "tenant-a", MiniappID: "m", Version: "1.0.0",
		InvocationID: invID, Operation: "t.t", IntentHash: hash,
		RequestedBy: "agent-a", Expiry: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	decider := approvals.Decider{
		Caller: contract.Caller{AgentID: "human-op",
			Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "human-op", TenantID: "tenant-a"}},
	}
	if err := appr.Decide(ctx, "tenant-a", id, decider, contract.DecisionApprove, hash); err != nil {
		t.Fatal(err)
	}
}

// TestP20LostAdoptionSubmitsNothing proves a losing or rejected
// adoption launches no DBOS workflow: on a failed record with no
// run (rejected adoption), and on a failed record already bound
// to a run (losing adoption onto an ineligible winner). Red:
// both legs submit. The with-run leg additionally fails on a
// green tree without the submitChecked liveness re-check.
func TestP20LostAdoptionSubmitsNothing(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.approvals")
	svc, async := p20ChokeFixture(t, db)
	ctx := context.Background()
	tool := contract.ToolManifest{
		Approval: contract.ApprovalRequirement{Required: true},
	}
	binding := auth.Binding{AgentID: "agent-a", Authority: contract.Authority{
		Kind: contract.AuthorityUser, ID: "agent-a", TenantID: "tenant-a",
	}}
	req := InvokeRequest{MiniappID: "m", Version: "1.0.0", Tool: "t.t",
		Input: []byte(`{}`), IdempotencyKey: "m1-nosub"}

	// Leg 1: failed record, no run — adoption rejected.
	// Non-aborting asserts: red must show both legs failing.
	id1 := p20SeedRow(t, db, "m1-nosub", string(contract.RunFailed), "")
	p20Grant(t, db, id1)
	if _, err := svc.submitChecked(ctx, req, binding, tool, invocation{ID: id1}); err == nil {
		t.Errorf("submitChecked(failed, no run) succeeded; must refuse without submitting")
	}
	if n := async.count(); n != 0 {
		t.Errorf("submits = %d, want 0 (rejected adoption submitted)", n)
	}
	if r := p20RunID(t, db, id1); r != "" {
		t.Errorf("run_id = %q, want empty (terminal gained a run)", r)
	}

	// Leg 2: failed record already bound — adoption loses onto an
	// ineligible winner; the winner's run must not be submitted.
	id2 := p20SeedRow(t, db, "m1-nosub2", string(contract.RunFailed), "run-prior")
	p20Grant(t, db, id2)
	if _, err := svc.submitChecked(ctx, req, binding, tool, invocation{ID: id2}); err == nil {
		t.Errorf("submitChecked(failed, bound) succeeded; must refuse without submitting")
	}
	if n := async.count(); n != 0 {
		t.Errorf("submits = %d, want 0 (losing adoption submitted the winner)", n)
	}
	if r := p20RunID(t, db, id2); r != "run-prior" {
		t.Errorf("run_id = %q, want run-prior", r)
	}
}
