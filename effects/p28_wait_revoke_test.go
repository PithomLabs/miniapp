package effects_test

// Prompt28 workstream D — §H-7 wait→revoke→refuse end to end. A
// gated confirm claim waits on a durable pending approval; the
// grant is revoked mid-wait via the setter API; subsequent
// claims refuse grant-denied and audit the refusal — even after
// the approval is later granted, because revocation wins. The
// pending approval row itself is untouched by revocation.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

func TestWaitRevokeRefuseEndToEnd(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	ctx := context.Background()

	reg := registry.New()
	mf := testkit.ClassAManifest("t-p28wait", "t.p28wait", "", nil)
	mf.Tools[0].Approval = contract.ApprovalRequirement{Required: true}
	mf.Tools[0].TrustModes = []contract.TrustMode{contract.TrustConfirm, contract.TrustAutonomous}
	impl := testkit.ClassAImpl("test", "t.p28wait",
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		})
	if err := reg.Register(mf, impl, testkit.Envelope(t, mf, "test")); err != nil {
		t.Fatal(err)
	}
	policy := auth.NewPolicy(nil, map[string]map[string]contract.TrustMode{
		wsTenant: {"t-p28wait": contract.TrustConfirm},
	})
	approver := approvals.New(db.Pool)
	svc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: allowCancellation{}, Policy: allowPolicy{},
		Registry: reg, Grants: policy,
		Approver: approver, Auditor: audit.New(db.Pool, nil),
	})
	inv := authzInvID(wsTenant, "inv-p28wait")
	seedInvocationFor(t, db, wsTenant, inv, "t-p28wait", "t.p28wait")
	// The shared seeder persists autonomous; this scenario needs
	// confirm trust to force the approval path.
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE app.invocations SET trust_mode = 'confirm'
			WHERE id = $1::uuid`, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	intentFor := func(op string) effects.Intent {
		return effects.Intent{BusinessOpID: wsTenant + ":" + op, AttemptID: "a-" + op,
			InvocationID: inv, EffectToken: authzToken,
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: mockPayload(mockcap.ModeAccept, op)}}
	}

	// Durable wait: pending approval parks the claim.
	apprID, err := approver.Submit(ctx, approvals.Request{
		TenantID: wsTenant, MiniappID: "t-p28wait", Version: "1.0.0",
		InvocationID: inv, Operation: "t.p28wait",
		IntentHash:    contract.IntentHash("t-p28wait", "1.0.0", "t.p28wait", json.RawMessage(`{}`)),
		ApproverScope: "", RequestedBy: "agent-a",
		Expiry: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Perform(ctx, wsTenant, intentFor("p28wait1")); !errors.Is(err, effects.ErrApprovalPending) {
		t.Fatalf("wait err = %v, want ErrApprovalPending", err)
	}
	if st := p28ApprovalStatus(t, db, apprID); st != "pending" {
		t.Fatalf("approval = %s, want pending (durable wait)", st)
	}

	// Revoke mid-wait via the setter API. Revocation settles
	// nothing: the approval row stays pending.
	policy.SetGrants(map[string]map[string]contract.TrustMode{})
	if st := p28ApprovalStatus(t, db, apprID); st != "pending" {
		t.Fatalf("approval = %s, want still pending (revocation settles nothing)", st)
	}

	// The wait resolves (approval granted), but the post-wait
	// claim refuses grant-denied and audits it: revocation wins
	// over linkage.
	decider := approvals.Decider{
		Caller: contract.Caller{AgentID: "human-op", Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: "human-op", TenantID: wsTenant,
		}},
	}
	hash := contract.IntentHash("t-p28wait", "1.0.0", "t.p28wait", json.RawMessage(`{}`))
	if err := approver.Decide(ctx, wsTenant, apprID, decider, contract.DecisionApprove, hash); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Perform(ctx, wsTenant, intentFor("p28wait2"))
	var refusal *effects.RefusalError
	if !errors.As(err, &refusal) || refusal.Reason != effects.RefusalGrantDenied {
		t.Fatalf("post-wait err = %v, want grant-denied refusal", err)
	}
	if n := p28RefusedAudits(t, db, wsTenant+":p28wait2"); n != 1 {
		t.Fatalf("refused audits = %d, want exactly 1", n)
	}
}

func p28ApprovalStatus(t *testing.T, db *testkit.DB, id string) string {
	t.Helper()
	var st string
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(context.Background(), `SELECT status FROM app.approvals
			WHERE id = $1::uuid`, id).Scan(&st)
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
}
