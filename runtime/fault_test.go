// Fault-injection tests for runtime lifecycle boundaries. Each test
// establishes three properties: the injector fired at the named
// operation, the expected error reached the actual caller, and the
// persisted state satisfies the lifecycle contract (rollback on
// pre-commit failure; idempotent recovery on post-commit response
// loss). FaultInjector itself is validated in testkit/fault.
package runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/fault"
	"github.com/PithomLabs/miniapp/testkit/faultwrap"

	"github.com/jackc/pgx/v5"
)

var errFault = errors.New("injected fault")

// TestServiceNilTestFault proves production construction leaves
// injection disabled.
func TestServiceNilTestFault(t *testing.T) {
	db := testkit.NewDB(t)
	reg := registry.New()
	policy := testPolicy()
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, reg), Async: &fakeAsync{},
		Approver: approvals.New(db.Pool), Auditor: audit.New(db.Pool, nil),
		Effects: effects.New(effects.Config{Pool: db.Pool,
			Channels: map[string]effects.Channel{}, Canceller: runtime.EffectCanceller(db.Pool)}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if svc.TestFault != nil {
		t.Fatal("TestFault != nil after production New")
	}
}

// newFaultFixture mirrors newFixture with the async executor wrapped
// for fault injection.
func newFaultFixture(t *testing.T, db *testkit.DB, inj *fault.Injector) (fixture, *faultwrap.Async) {
	t.Helper()
	fx := newFixture(t, db)
	wrapped := &faultwrap.Async{Inner: fx.async, Inj: inj}
	svc, err := runtime.New(runtime.Config{
		Registry: fx.reg, Policy: fx.policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, fx.reg), Async: wrapped,
		Approver: fx.approver, Auditor: fx.aud, Effects: fx.effects,
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.svc = svc
	return fx, wrapped
}

func approvalCount(t *testing.T, db *testkit.DB, tenant string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.approvals`).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func invocationStatus(t *testing.T, db *testkit.DB, tenant, key, miniapp, tool string) (contract.RunStatus, string) {
	t.Helper()
	type row struct {
		status contract.RunStatus
		runID  string
	}
	r, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (row, error) {
		var s string
		var run string
		err := tx.QueryRow(context.Background(), `SELECT status, run_id FROM app.invocations
			WHERE tenant_id = $1 AND miniapp_id = $2 AND tool = $3 AND idempotency_key = $4`,
			tenant, miniapp, tool, key).Scan(&s, &run)
		return row{contract.RunStatus(s), run}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return r.status, r.runID
}

func auditCount(t *testing.T, db *testkit.DB, tenant, action string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.audit_events WHERE action = $1`, action).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestFaultApprovalCreate: Approver.Submit fails → retryable, no
// approval row, invocation not parked (X-01 trigger a setup).
func TestFaultApprovalCreate(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	inj := fault.New()
	inj.FailAt("approval-create", 1, errFault)
	fx.approver.TestFault = inj

	_, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fc", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "store-unavailable") {
		t.Fatalf("err = %v, want retryable store-unavailable", err)
	}
	if n := inj.Count("approval-create"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if n := approvalCount(t, db, tenantA); n != 0 {
		t.Fatalf("approval rows = %d, want 0 (rolled back)", n)
	}
	if st, _ := invocationStatus(t, db, tenantA, "key-fc", probeGatedID, probeGatedTool); st == contract.RunWaitingApproval {
		t.Fatalf("record parked despite failed approval creation")
	}
}

// TestFaultParkUpdate: park persistence fails → retryable, nothing
// parked (no approval row orphaned by the atomic park's failure,
// no waiting record).
func TestFaultParkUpdate(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	inj := fault.New()
	inj.FailAt("park-update", 1, errFault)
	fx.svc.TestFault = inj

	_, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fp", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "store-unavailable") {
		t.Fatalf("err = %v, want retryable store-unavailable", err)
	}
	if n := inj.Count("park-update"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	// Rollback property (holds pre- and post-R3; WS-A pins the
	// initial state value to pending): no park, and the approval
	// created before the failed park is visible for adoption.
	if st, _ := invocationStatus(t, db, tenantA, "key-fp", probeGatedID, probeGatedTool); st == contract.RunWaitingApproval {
		t.Fatalf("record parked despite failed park update")
	}
}

// TestFaultParkAudit: the audit leg of the atomic Park fails →
// the whole park rolls back (record stays pending, no approval
// row, no audit row) and the caller retries; the retry parks
// cleanly. Updated for atomic Park (WS-A): pre-atomic code left
// the park committed with a missing audit; the atomic operation
// fails as one store-unavailable unit. Distinguishes audit failure
// from park-update failure per R-03 (separate ops, separate
// rollback assertions).
func TestFaultParkAudit(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	inj := fault.New()
	inj.FailAt("audit:approval-requested", 1, errFault)
	fx.svc.TestFault = inj

	_, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fa", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "store-unavailable") {
		t.Fatalf("err = %v, want retryable store-unavailable", err)
	}
	if n := inj.Count("audit:approval-requested"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if st, _ := invocationStatus(t, db, tenantA, "key-fa", probeGatedID, probeGatedTool); st != contract.RunPending {
		t.Fatalf("status = %s, want pending (atomic rollback)", st)
	}
	if n := approvalCount(t, db, tenantA); n != 0 {
		t.Fatalf("approval rows = %d, want 0 (rolled back with the park)", n)
	}
	if n := auditCount(t, db, tenantA, "approval-requested"); n != 0 {
		t.Fatalf("approval-requested audits = %d, want 0", n)
	}
	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fa", credAgentA, contract.TrustConfirm)
	if err != nil || out.Status != contract.RunWaitingApproval || out.ApprovalID == "" {
		t.Fatalf("retry = %+v, %v; want parked with approval", out, err)
	}
	if n := auditCount(t, db, tenantA, "approval-requested"); n != 1 {
		t.Fatalf("approval-requested audits after retry = %d, want exactly 1", n)
	}
}

// TestFaultRunIDPersist: run-ID persistence fails → retryable, no
// run recorded, executor never submitted (boundary before submit).
func TestFaultRunIDPersist(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	inj := fault.New()
	inj.FailAt("runid-persist", 1, errFault)
	fx.svc.TestFault = inj

	_, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-fr", credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "store-unavailable") {
		t.Fatalf("err = %v, want retryable store-unavailable", err)
	}
	if n := inj.Count("runid-persist"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if _, runID := invocationStatus(t, db, tenantA, "key-fr", probeID, probeTool); runID != "" {
		t.Fatalf("run_id = %q, want empty (persistence failed first)", runID)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if fx.async.submitCalls != 0 {
		t.Fatalf("submitCalls = %d, want 0 (fault before submit)", fx.async.submitCalls)
	}
}

// TestFaultAsyncSubmit: executor submission fails → retryable
// submit-failed; the run ID stays persisted and live for retry
// (X-02 setup); the delegate executor sees nothing.
func TestFaultAsyncSubmit(t *testing.T) {
	db := newDB(t)
	inj := fault.New()
	inj.FailAt(faultwrap.OpAsyncSubmit, 1, errFault)
	fx, _ := newFaultFixture(t, db, inj)

	_, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-fs", credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "submit-failed") {
		t.Fatalf("err = %v, want retryable submit-failed", err)
	}
	if n := inj.Count(faultwrap.OpAsyncSubmit); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	st, runID := invocationStatus(t, db, tenantA, "key-fs", probeID, probeTool)
	if runID == "" {
		t.Fatal("run_id empty, want persisted for retry")
	}
	if st != contract.RunRunning {
		t.Fatalf("status = %s, want running (live for retry)", st)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if fx.async.submitCalls != 0 {
		t.Fatalf("delegate submitCalls = %d, want 0 (wrapper fired first)", fx.async.submitCalls)
	}
}

// TestFaultAsyncSignal: signal transmission fails after a committed
// decision → retryable; the decision stands and the retry
// converges idempotently with exactly one delivery.
func TestFaultAsyncSignal(t *testing.T) {
	db := newDB(t)
	inj := fault.New()
	inj.FailAt(faultwrap.OpAsyncSignal, 1, errFault)
	fx, _ := newFaultFixture(t, db, inj)

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fsig", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(`{}`))
	decide := func() error {
		return fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash)
	}
	if err := decide(); err == nil || !strings.Contains(err.Error(), "signal-failed") {
		t.Fatalf("decide err = %v, want retryable signal-failed", err)
	}
	if n := inj.Count(faultwrap.OpAsyncSignal); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if err := decide(); err != nil {
		t.Fatalf("retry: %v (must converge idempotently)", err)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.signal) != 1 {
		t.Fatalf("deliveries = %d, want exactly 1", len(fx.async.signal))
	}
}

// TestFaultApprovalLinkage: approval lookup fails on poll →
// retryable; the waiting record is untouched.
func TestFaultApprovalLinkage(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fl", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	inj := fault.New()
	inj.FailAt("approval-linkage", 1, errFault)
	fx.approver.TestFault = inj
	if waiting.ApprovalID == "" {
		t.Fatal("no approval parked")
	}

	_, err = invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-fl", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "store-unavailable") {
		t.Fatalf("re-invoke err = %v, want retryable store-unavailable", err)
	}
	if n := inj.Count("approval-linkage"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if st, _ := invocationStatus(t, db, tenantA, "key-fl", probeGatedID, probeGatedTool); st != contract.RunWaitingApproval {
		t.Fatalf("status = %s, want waiting_approval (untouched)", st)
	}
}

// TestFaultCompletionAudit: settlement audit fails on Status →
// retryable once; the persisted terminal record converges on the
// next read. (Pre-WS-D behavior: the record update stands; WS-D's
// sweeper backfills the missing audit.)
func TestFaultCompletionAudit(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-fca", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	fx.async.mu.Lock()
	fx.async.status[out.Handle.RunID] = contract.RunSucceeded
	fx.async.mu.Unlock()
	inj := fault.New()
	inj.FailAt("audit:run-settled", 1, errFault)
	fx.svc.TestFault = inj

	if _, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID); err == nil ||
		!strings.Contains(err.Error(), "audit-unavailable") {
		t.Fatalf("status err = %v, want retryable audit-unavailable", err)
	}
	if n := inj.Count("audit:run-settled"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	done, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil {
		t.Fatalf("second status: %v (must converge)", err)
	}
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s, want succeeded", done.Status)
	}
	if n := auditCount(t, db, tenantA, "run-settled"); n != 0 {
		t.Fatalf("run-settled audits = %d, want 0 (failed audit left no row)", n)
	}
}

// TestApprovalRetryAfterSubmitFailure (X-01 trigger a): approval
// creation fails once; the same-key retry parks cleanly and the
// Class D submit carries the parked linkage — never a blind
// unapproved submit.
func TestApprovalRetryAfterSubmitFailure(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	inj := fault.New()
	inj.FailAt("approval-create", 1, errFault)
	fx.approver.TestFault = inj

	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-x01a", credAgentA, contract.TrustConfirm); err == nil {
		t.Fatal("first invoke succeeded despite injected failure")
	}
	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-x01a", credAgentA, contract.TrustConfirm)
	if err != nil || out.Status != contract.RunWaitingApproval || out.ApprovalID == "" {
		t.Fatalf("retry = %+v, %v; want parked with approval", out, err)
	}
	if n := approvalCount(t, db, tenantA); n != 1 {
		t.Fatalf("approval rows = %d, want exactly 1", n)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.submits) != 1 || fx.async.submits[0].ApprovalID != out.ApprovalID {
		t.Fatalf("submits = %+v, want one submit bound to %q", fx.async.submits, out.ApprovalID)
	}
}

// TestApprovalRetryAfterParkFailure (X-01 trigger b): identical
// convergence when the park transition itself fails once.
func TestApprovalRetryAfterParkFailure(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	inj := fault.New()
	inj.FailAt("park-update", 1, errFault)
	fx.svc.TestFault = inj

	if _, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-x01b", credAgentA, contract.TrustConfirm); err == nil {
		t.Fatal("first invoke succeeded despite injected failure")
	}
	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-x01b", credAgentA, contract.TrustConfirm)
	if err != nil || out.Status != contract.RunWaitingApproval || out.ApprovalID == "" {
		t.Fatalf("retry = %+v, %v; want parked with approval", out, err)
	}
	if n := approvalCount(t, db, tenantA); n != 1 {
		t.Fatalf("approval rows = %d, want exactly 1", n)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.submits) != 1 || fx.async.submits[0].ApprovalID == "" {
		t.Fatalf("submits = %+v, want one linked submit", fx.async.submits)
	}
}

// TestApprovalCrashBetweenClaimAndPark (X-01 trigger c): a crash
// after claim commit leaves a pending record with no approval; the
// retry must park (not submit blindly). The SQL fixture is the
// crash's exact durable outcome, so the test is equivalent to a
// kill at that boundary without the process machinery.
func TestApprovalCrashBetweenClaimAndPark(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES (gen_random_uuid(), $1, $2, '1.0.0', $3, 'key-x01c', '{}', 'pending', 'confirm', 'tok-x01c')`,
			tenantA, probeGatedID, probeGatedTool)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-x01c", credAgentA, contract.TrustConfirm)
	if err != nil || out.Status != contract.RunWaitingApproval || out.ApprovalID == "" {
		t.Fatalf("retry = %+v, %v; want parked with approval", out, err)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.submits) != 1 || fx.async.submits[0].ApprovalID != out.ApprovalID {
		t.Fatalf("submits = %+v, want one submit bound to %q", fx.async.submits, out.ApprovalID)
	}
}
