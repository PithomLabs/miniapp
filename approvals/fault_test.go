// Fault-injection tests for approval boundaries: creation,
// linkage, decision commit (rollback), and decision response
// (idempotent recovery after a committed decision).
package approvals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

var errFault = errors.New("injected fault")

// TestApprovalsNilTestFault proves production construction leaves
// injection disabled.
func TestApprovalsNilTestFault(t *testing.T) {
	if svc := New(nil); svc.TestFault != nil {
		t.Fatal("TestFault != nil after production New")
	}
}

// TestFaultApprovalCreateUnit: Submit fails before the INSERT; no
// row is created.
func TestFaultApprovalCreateUnit(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	inj := fault.New()
	inj.FailAt("approval-create", 1, errFault)
	svc.TestFault = inj
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if _, err := svc.Submit(ctx, testRequest()); !errors.Is(err, errFault) {
		t.Fatalf("Submit err = %v, want injected fault", err)
	}
	if n := inj.Count("approval-create"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if _, err := svc.ByInvocation(ctx, "t-appr", "inv-1"); err == nil {
		t.Fatal("approval row exists despite failed creation")
	}
}

// TestFaultApprovalLinkageUnit: ByInvocation fails; the caller sees
// the injected error.
func TestFaultApprovalLinkageUnit(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	inj := fault.New()
	inj.FailAt("approval-linkage", 1, errFault)
	svc.TestFault = inj

	if _, err := svc.ByInvocation(ctx, "t-appr", "inv-1"); !errors.Is(err, errFault) {
		t.Fatalf("ByInvocation err = %v, want injected fault", err)
	}
	if n := inj.Count("approval-linkage"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	// The row itself is untouched: a retry without the fault reads it.
	svc.TestFault = nil
	rec, err := svc.ByInvocation(ctx, "t-appr", "inv-1")
	if err != nil || rec.ID != id {
		t.Fatalf("retry = %+v, %v; want the parked row", rec, err)
	}
}

// TestFaultDecideCommit: a validated decision fails before its
// UPDATE; the approval stays pending (rollback) and a retry
// converges.
func TestFaultDecideCommit(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	inj := fault.New()
	inj.FailAt("decide-commit", 1, errFault)
	svc.TestFault = inj

	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); !errors.Is(err, errFault) {
		t.Fatalf("Decide err = %v, want injected fault", err)
	}
	if n := inj.Count("decide-commit"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	rec, err := svc.Get(ctx, "t-appr", id)
	if err != nil || rec.Status != "pending" {
		t.Fatalf("status = %+v, %v; want pending (rolled back)", rec, err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); err != nil {
		t.Fatalf("retry: %v (must converge)", err)
	}
}

// TestFaultDecideResponse: the decision commits, then the response
// is lost; the identical retry converges idempotently without
// duplicating or substituting the decision.
func TestFaultDecideResponse(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	inj := fault.New()
	inj.FailAt("decide-response", 1, errFault)
	svc.TestFault = inj

	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); !errors.Is(err, errFault) {
		t.Fatalf("Decide err = %v, want injected fault", err)
	}
	rec, err := svc.Get(ctx, "t-appr", id)
	if err != nil || rec.Status != "decided" || rec.Decision != contract.DecisionApprove {
		t.Fatalf("status = %+v, %v; want committed approve", rec, err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); err != nil {
		t.Fatalf("identical retry: %v (must converge idempotently)", err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionReject, testHash); err == nil {
		t.Fatal("conflicting retry accepted after committed decision")
	}
}
