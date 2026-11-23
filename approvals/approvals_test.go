package approvals

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/testkit"
)

const testHash = contract.Digest("sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

func testRequest() Request {
	return Request{
		TenantID: "t-appr", MiniappID: "crm-read", Version: "1.0.0",
		InvocationID: "inv-1", Operation: "rpa.crm.read_contact",
		IntentHash: testHash, RequestedBy: "agent-1",
		Expiry: time.Now().UTC().Add(time.Hour),
	}
}

func reviewer() Decider {
	return Decider{Caller: contract.Caller{AgentID: "agent-2", Authority: contract.Authority{
		Kind: contract.AuthorityUser, ID: "u9", TenantID: "t-appr",
	}}}
}

func TestApprovalRoundTrip(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	rec, err := svc.Get(ctx, "t-appr", id)
	if err != nil || rec.Status != "pending" {
		t.Fatalf("fresh approval = %+v, %v", rec, err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); err != nil {
		t.Fatal(err)
	}
	rec, err = svc.Get(ctx, "t-appr", id)
	if err != nil || rec.Status != "decided" || rec.Decision != contract.DecisionApprove {
		t.Fatalf("decided approval = %+v, %v", rec, err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionReject, testHash); err == nil {
		t.Fatal("double decision accepted")
	}
}

func TestApprovalGuards(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	otherHash := contract.Digest("sha256:" + strings.Repeat("f", 64))
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, otherHash); err == nil {
		t.Fatal("decide with changed intent accepted")
	}
	self := reviewer()
	self.AgentID = "agent-1"
	if err := svc.Decide(ctx, "t-appr", id, self, contract.DecisionApprove, testHash); err == nil {
		t.Fatal("self-approval accepted")
	}
	foreign := reviewer()
	foreign.Authority.TenantID = "t-evil"
	if err := svc.Decide(ctx, "t-appr", id, foreign, contract.DecisionApprove, testHash); err == nil {
		t.Fatal("cross-tenant decision accepted")
	}
	expired := testRequest()
	expired.InvocationID = "inv-expired"
	expired.Expiry = time.Now().UTC().Add(-time.Minute)
	expID, err := svc.Submit(ctx, expired)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Decide(ctx, "t-appr", expID, reviewer(), contract.DecisionApprove, testHash); err == nil {
		t.Fatal("expired approval decided")
	}
	rec, err := svc.Get(ctx, "t-appr", expID)
	if err != nil || rec.Status != "expired" {
		t.Fatalf("expired approval status = %+v, %v", rec, err)
	}
}

func TestCancelPending(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	settled, err := svc.CancelPending(ctx, "t-appr", "inv-1")
	if err != nil || !settled {
		t.Fatalf("cancel pending = %v, %v; want true, nil", settled, err)
	}
	rec, err := svc.Get(ctx, "t-appr", id)
	if err != nil || rec.Status != "cancelled" {
		t.Fatalf("status = %+v, %v; want cancelled", rec, err)
	}
	settled, err = svc.CancelPending(ctx, "t-appr", "inv-1")
	if err != nil || settled {
		t.Fatalf("second cancel = %v, %v; want false, nil", settled, err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); err == nil {
		t.Fatal("decide on cancelled approval accepted")
	}
}

func TestDecideScopeAndIdempotency(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	req := testRequest()
	req.ApproverScope = "tenant-ops"
	id, err := svc.Submit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash); err == nil {
		t.Fatal("unscoped decider accepted on scoped approval")
	}
	ops := reviewer()
	ops.Roles = []string{"tenant-ops"}
	if err := svc.Decide(ctx, "t-appr", id, ops, contract.DecisionApprove, testHash); err != nil {
		t.Fatalf("scoped decide: %v", err)
	}
	if err := svc.Decide(ctx, "t-appr", id, ops, contract.DecisionApprove, testHash); err != nil {
		t.Fatalf("identical re-decide: %v (must be idempotent)", err)
	}
	if err := svc.Decide(ctx, "t-appr", id, ops, contract.DecisionReject, testHash); err == nil {
		t.Fatal("conflicting re-decide accepted")
	}
}
