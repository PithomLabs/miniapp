// X-01/X-02 discriminating tests (approval bypass on retry;
// approval-path convergence).
//
// RED/GREEN-COMPATIBLE SOURCE: this file must compile and run
// against both the pre-remediation snapshot and the remediated
// tree. It uses only APIs present in both (public service surface,
// pre-existing failure drivers, direct SQL fixtures) plus helpers
// defined in this file. The red run overlays exactly this file
// onto the snapshot. Tests needing the fault injector live in
// fault_test.go and are green-only.
package runtime_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"

	"github.com/PithomLabs/miniapp/miniapps/crm-read"
)

// recordingAsync wraps fakeAsync and records submitted requests so
// tests can prove linkage identity. Defined here (not on the shared
// fake) to keep this file red-compatible.
type recordingAsync struct {
	inner   *fakeAsync
	mu      sync.Mutex
	submits []executor.Request
}

func (r *recordingAsync) Submit(ctx context.Context, req executor.Request) (contract.RunHandle, error) {
	h, err := r.inner.Submit(ctx, req)
	if err == nil {
		r.mu.Lock()
		r.submits = append(r.submits, req)
		r.mu.Unlock()
	}
	return h, err
}

func (r *recordingAsync) Status(ctx context.Context, runID string) (contract.RunStatus, error) {
	return r.inner.Status(ctx, runID)
}

func (r *recordingAsync) Signal(ctx context.Context, runID string, sig contract.Signal) error {
	return r.inner.Signal(ctx, runID, sig)
}

func (r *recordingAsync) Cancel(ctx context.Context, runID string, reason string) error {
	return r.inner.Cancel(ctx, runID, reason)
}

func (r *recordingAsync) recorded() []executor.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]executor.Request(nil), r.submits...)
}

// redFixture mirrors newFixture with a recording async executor.
// Self-contained (old-stable helpers only) for red compatibility.
type redFixture struct {
	svc      *runtime.Service
	async    *fakeAsync
	rec      *recordingAsync
	approver *approvals.Service
}

func newRedFixture(t *testing.T, db *testkit.DB) redFixture {
	t.Helper()
	reg := registry.New()
	if err := crmread.Register(reg); err != nil {
		t.Fatal(err)
	}
	registerGated(t, reg)
	registerProbe(t, reg)
	registerProbeWith(t, reg, probeGatedID, probeGatedTool, true)
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	rec := &recordingAsync{inner: async}
	aud := audit.New(db.Pool, nil)
	approver := approvals.New(db.Pool)
	policy := testPolicy()
	effSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: reg, Grants: policy},
	})
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, reg), Async: rec,
		Approver: approver, Auditor: aud, Effects: effSvc,
	})
	if err != nil {
		t.Fatal(err)
	}
	return redFixture{svc: svc, async: async, rec: rec, approver: approver}
}

func gatedApprovalRequest(invID string, hash contract.Digest) approvals.Request {
	return approvals.Request{
		TenantID: tenantA, MiniappID: probeGatedID, Version: "1.0.0",
		InvocationID: invID, Operation: probeGatedTool, IntentHash: hash,
		ApproverScope: "tenant-ops", RequestedBy: "agent-a",
	}
}

// insertLegacyRunningRow simulates the pre-R3 crash/park-failure
// durable state directly: a gated invocation claimed Running with
// no run and no approval. Both trees read this shape; old code
// submits it blindly (the X-01 defect), new code re-enters
// approval handling through the choke point.
func insertLegacyRunningRow(t *testing.T, db *testkit.DB, key string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', 'running', 'confirm', 'tok-legacy')`,
			uuid.NewString(), tenantA, probeGatedID, probeGatedTool, key)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestApprovalLegacyRunningNoBlindSubmit is the X-01 red
// discriminator: a gated record stranded Running without a run
// must never submit without an approval linkage when retried.
func TestApprovalLegacyRunningNoBlindSubmit(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	insertLegacyRunningRow(t, db, "key-legacy")

	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-legacy", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("retry err = %v, want parked result", err)
	}
	for _, s := range fx.rec.recorded() {
		if s.ApprovalID == "" {
			t.Fatalf("submitted gated workflow %s without approval linkage (X-01 bypass)", s.RunID)
		}
	}
	if out.Status != contract.RunWaitingApproval || out.ApprovalID == "" {
		t.Fatalf("retry = %+v; want parked with approval", out)
	}
	submits := fx.rec.recorded()
	if len(submits) != 1 {
		t.Fatalf("submits = %d, want exactly 1 linked submit", len(submits))
	}
	if submits[0].ApprovalID != out.ApprovalID {
		t.Fatalf("submitted linkage %q != parked approval %q", submits[0].ApprovalID, out.ApprovalID)
	}
}

// TestApprovalAdoptsDanglingPendingApproval proves the X-01
// dangling-row case: a legacy Running record with a pre-existing
// pending approval adopts it (no duplicate) and submits bound to it.
func TestApprovalAdoptsDanglingPendingApproval(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	insertLegacyRunningRow(t, db, "key-dangle")
	var invID string
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(context.Background(),
			`SELECT id::text FROM app.invocations WHERE idempotency_key = 'key-dangle'`).Scan(&invID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(`{}`))
	dangling, err := fx.approver.Submit(context.Background(), gatedApprovalRequest(invID, hash))
	if err != nil {
		t.Fatal(err)
	}

	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-dangle", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("retry err = %v, want live result", err)
	}
	if out.Handle == nil {
		t.Fatalf("retry = %+v; want a live handle", out)
	}
	submits := fx.rec.recorded()
	if len(submits) != 1 || submits[0].ApprovalID != dangling {
		t.Fatalf("submits = %+v, want exactly one submit bound to dangling %q (no blind submit)", submits, dangling)
	}
}

// TestApprovalAdoptionCreatesNoDuplicate proves the adopted path
// never inserts a second pending approval for the invocation.
func TestApprovalAdoptionCreatesNoDuplicate(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	insertLegacyRunningRow(t, db, "key-dangle-dup")
	var invID string
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(context.Background(),
			`SELECT id::text FROM app.invocations WHERE idempotency_key = 'key-dangle-dup'`).Scan(&invID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(`{}`))
	if _, err := fx.approver.Submit(context.Background(), gatedApprovalRequest(invID, hash)); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-dangle-dup", credAgentA, contract.TrustConfirm); err != nil {
		t.Fatalf("retry err = %v", err)
	}
	n, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.approvals WHERE invocation_id = $1`, invID).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("approval rows = %d, want exactly 1 (adopted, not duplicated)", n)
	}
}

// TestApprovalPollSubmitsIfMissing is the X-02 grant discriminator:
// a granted approval whose run was never submitted must converge
// by submitting idempotently on poll — not fail forever.
func TestApprovalPollSubmitsIfMissing(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	// Waiting record with no run (park-audit-failure shape), plus a
	// decided approval, built from durable state both trees read.
	invID := uuid.NewString()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, 'key-poll-missing', '{}', 'waiting_approval', 'confirm', 'tok-pm')`,
			invID, tenantA, probeGatedID, probeGatedTool)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(`{}`))
	apprID, err := fx.approver.Submit(context.Background(), gatedApprovalRequest(invID, hash))
	if err != nil {
		t.Fatal(err)
	}
	if err := fx.svc.DecideApproval(context.Background(), credDecide, apprID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("decide: %v", err)
	}

	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-poll-missing", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("poll err = %v, want converged Running handle (submit-if-missing)", err)
	}
	if out.Status != contract.RunRunning || out.Handle == nil {
		t.Fatalf("poll = %+v; want Running with handle", out)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if _, ok := fx.async.status[out.Handle.RunID]; !ok {
		t.Fatalf("run %s missing from executor (submit-if-missing did not submit)", out.Handle.RunID)
	}
}

// TestApprovalGrantAfterSubmitFailure: the park-time submit fails,
// then a human approves. The grant must take effect on the
// actually-submitted run (X-02): Decide recovers the missing run,
// and poll converges to a live handle. Red: the grant fails
// forever (signal/poll against a run that does not exist).
func TestApprovalGrantAfterSubmitFailure(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	fx.async.failNextSubmit = true
	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-grant-missing", credAgentA, contract.TrustConfirm); err == nil {
		t.Fatal("park-time submit succeeded despite injected failure")
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(`{}`))
	approvalID := parkedApprovalID(t, db, "key-grant-missing")
	if err := fx.svc.DecideApproval(context.Background(), credDecide, approvalID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("decide err = %v, want nil (grant must take effect on the recovered run)", err)
	}
	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-grant-missing", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("poll err = %v, want converged Running handle", err)
	}
	if out.Status != contract.RunRunning || out.Handle == nil {
		t.Fatalf("poll = %+v; want Running with handle", out)
	}
}

// TestApprovalDenialSettlesAfterParkFailure: the park-time submit
// fails, then a human denies. The invocation must settle terminally
// (a named-but-never-submitted run has nothing to stop) instead of
// retrying cancellation forever. Red: poll loops retryable.
func TestApprovalDenialSettlesAfterParkFailure(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	fx.async.failNextSubmit = true
	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-deny-missing", credAgentA, contract.TrustConfirm); err == nil {
		t.Fatal("park-time submit succeeded despite injected failure")
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(`{}`))
	approvalID := parkedApprovalID(t, db, "key-deny-missing")
	// The decision commits on both trees (signal delivery differs);
	// convergence is judged at poll, where the defect strands red.
	_ = fx.svc.DecideApproval(context.Background(), credDecide, approvalID, contract.DecisionReject, hash)
	if rec, err := fx.approver.Get(context.Background(), tenantA, approvalID); err != nil || rec.Decision != contract.DecisionReject {
		t.Fatalf("decision = %+v, %v; want committed reject", rec, err)
	}
	_, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-deny-missing", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "approval-denied") {
		t.Fatalf("poll err = %v, want terminal approval-denied (denial must settle, not loop)", err)
	}
}

// TestApprovalExpirySettlesAfterParkFailure: same as denial, via
// lazy expiry observation on poll.
func TestApprovalExpirySettlesAfterParkFailure(t *testing.T) {
	db := newDB(t)
	fx := newRedFixture(t, db)
	fx.async.failNextSubmit = true
	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-exp-missing", credAgentA, contract.TrustConfirm); err == nil {
		t.Fatal("park-time submit succeeded despite injected failure")
	}
	approvalID := parkedApprovalID(t, db, "key-exp-missing")
	backdateExpiry(t, db, approvalID)
	_, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-exp-missing", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "approval-expired") {
		t.Fatalf("poll err = %v, want terminal approval-expired (expiry must settle, not loop)", err)
	}
}

// parkedApprovalID returns the pending approval parked for one
// invocation key.
func parkedApprovalID(t *testing.T, db *testkit.DB, key string) string {
	t.Helper()
	var invID string
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(context.Background(),
			`SELECT id::text FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&invID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	var apprID string
	_, err = store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(context.Background(),
			`SELECT id::text FROM app.approvals WHERE invocation_id = $1 ORDER BY created_at DESC LIMIT 1`, invID).Scan(&apprID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return apprID
}
