package runtime_test

// P26-002 (Prompt27) characterization: empty-TrustModes legacy
// residual. Submit admits gated+autonomous (the choke skips
// linkage for autonomous trust); authorization then refuses
// approval-missing (empty modes do not grant the explicit
// autonomous exception). This pins the accepted residual
// end to end: submit → persisted trust → authorize decision.
// NOT red/green — no behavior change exists in this work
// order that could produce a different result.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

const (
	p27GatedID   = "test-p27-gated"
	p27GatedTool = "probe.p27gated"
)

// p27EmptyModeFixture wires a gated miniapp whose manifest
// declares NO TrustModes (legacy empty list) with an
// autonomous grant, so submit admits what authorization
// must refuse. Self-contained: shared fixtures are untouched.
func p27EmptyModeFixture(t *testing.T, db *testkit.DB) (*runtime.Service, *effects.Service) {
	t.Helper()
	reg := registry.New()
	registerProbeWith(t, reg, p27GatedID, p27GatedTool, true)
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	aud := audit.New(db.Pool, nil)
	approver := approvals.New(db.Pool)
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			credAgentA: {AgentID: "agent-a", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "agent-a", TenantID: tenantA,
			}},
		},
		map[string]map[string]contract.TrustMode{
			tenantA: {p27GatedID: contract.TrustAutonomous},
		},
	)
	effSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: reg, Grants: policy},
		Registry:  reg, Grants: policy, Approver: approver, Auditor: aud,
	})
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, reg), Async: async,
		Approver: approver, Auditor: aud, Effects: effSvc,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, effSvc
}

func TestP27EmptyModeGatedAutonomousRefused(t *testing.T) {
	db := newDB(t)
	svc, effSvc := p27EmptyModeFixture(t, db)
	ctx := context.Background()

	// 1. Submit admits: autonomous gated invoke is allowed and
	// submits without approval linkage.
	out, err := invoke(svc, p27GatedID, p27GatedTool, `{}`, "key-p27-empty", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke err = %v, want nil (submit admits)", err)
	}
	if out.Status != contract.RunRunning || out.Handle == nil {
		t.Fatalf("invoke = %+v, want running with handle", out)
	}

	// 2. Persisted trust is recorded as autonomous on the
	// INVOCATION row (status running — live, not pending).
	var persistedTrust, invStatus string
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT trust_mode, status FROM app.invocations
			WHERE id = $1::uuid`, out.InvocationID).Scan(&persistedTrust, &invStatus)
	})
	if err != nil {
		t.Fatal(err)
	}
	if persistedTrust != string(contract.TrustAutonomous) {
		t.Fatalf("persisted trust = %q, want autonomous", persistedTrust)
	}
	if invStatus != string(contract.RunRunning) {
		t.Fatalf("invocation status = %q, want running", invStatus)
	}

	// 3+4. Authorization refuses with the exact reason.
	tok := effectToken(t, db, tenantA, out.InvocationID)
	intent := effIntent(out.InvocationID, tok, "p27empty", mockcap.ModeAccept)
	_, err = effSvc.Perform(ctx, tenantA, intent)
	var refusal *effects.RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("perform err = %v, want terminal *RefusalError", err)
	}
	if refusal.Reason != effects.RefusalApprovalMissing {
		t.Fatalf("reason = %s, want approval-missing", refusal.Reason)
	}

	// 5. No unauthorized send: the EFFECT ledger row stays
	// PENDING (claimed provisionally, never ATTEMPTING) and
	// the provider was never called.
	var effectState string
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT state FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`,
			intent.BusinessOpID, tenantA).Scan(&effectState)
	})
	if err != nil {
		t.Fatal(err)
	}
	if effectState != "PENDING" {
		t.Errorf("effect row state = %s, want PENDING (claimed, never authorized)", effectState)
	}
	if n := mockcap.SentCount(t, db.Pool, "p27empty"); n != 0 {
		t.Errorf("provider sends = %d, want 0", n)
	}

	// 5b. No indefinite retry loop, no misleading success: an
	// immediate repeat meets adopt contention (the just-claimed
	// PENDING row is fresh, so adopt answers before authorize
	// re-runs) — never success, never a send. Past the lease,
	// the row re-enters authorization and the terminal refusal
	// returns: the cycle converges to refusal, it does not loop
	// forever and it never flips to success.
	_, err = effSvc.Perform(ctx, tenantA, intent)
	if !errors.Is(err, effects.ErrAttemptInProgress) {
		t.Fatalf("immediate repeat err = %v, want contention (fresh PENDING adopt)", err)
	}
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE app.effect_records SET updated_at = now() - interval '1 hour'
			WHERE business_op_id = $1 AND tenant_id = $2`, intent.BusinessOpID, tenantA)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = effSvc.Perform(ctx, tenantA, intent)
	var repeat *effects.RefusalError
	if !errors.As(err, &repeat) {
		t.Fatalf("post-lease repeat err = %v, want terminal *RefusalError", err)
	}
	if repeat.Reason != effects.RefusalApprovalMissing {
		t.Errorf("post-lease repeat reason = %s, want approval-missing", repeat.Reason)
	}
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT status FROM app.invocations
			WHERE id = $1::uuid`, out.InvocationID).Scan(&invStatus)
	})
	if err != nil {
		t.Fatal(err)
	}
	if invStatus != string(contract.RunRunning) {
		t.Errorf("invocation status after refusals = %q, want running (undisturbed)", invStatus)
	}
}
