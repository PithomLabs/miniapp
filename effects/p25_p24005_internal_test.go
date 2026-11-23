package effects

// P24-005 (Prompt 25 §6) authorization matrix: owner option (a) — a
// gated tool skips the human-approval linkage only when the
// persisted invocation trust is autonomous AND the admitted tool
// manifest explicitly permits autonomous execution. Persisted
// trust is validated against the manifest independently of the
// linkage exception; caller claims (Intent carries no trust) can
// never establish it.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
)

const (
	p25mTenant   = "tenant-p25m"
	p25mToken    = "tok-p25m"
	p25mGated    = "p25m-gated"
	p25mGatedTl  = "p25m.gated"
	p25mConfOnly = "p25m-confonly"
	p25mConfTl   = "p25m.conf"
)

// p25mRegisterGated admits a minimal gated Class D tool with the
// given explicit trust modes.
func p25mRegisterGated(t *testing.T, reg *registry.Registry, id, tool string, modes []contract.TrustMode) {
	t.Helper()
	sum := sha256.Sum256([]byte(`{"name":"default","placement":"machine","substrate":"dbos"}`))
	profile := contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
	schema := json.RawMessage(`{"type":"object"}`)
	mf := &contract.MiniappManifest{
		ID: contract.Name(id), Version: contract.Version("1.0.0"),
		Description: "P24-005 matrix probe.",
		Tools: []contract.ToolManifest{{
			Name: contract.ToolName(tool), Description: "Gated matrix probe.",
			Class: contract.ClassD, Substrate: contract.SubstrateDBOS,
			InputSchema: schema, OutputSchema: schema,
			TrustModes:  modes,
			Approval:    contract.ApprovalRequirement{Required: true, ApproverScope: "tenant-ops", Expiry: "1h"},
			NoEntityWhy: "Probe: no effects.",
		}},
		Deployment: contract.DeploymentSection{Tools: map[string]contract.ToolDeployment{
			tool: {Placement: contract.PlacementMachine,
				Profile: contract.ProfileRef{Name: "default", Digest: profile}},
		}},
	}
	if err := mf.Validate(); err != nil {
		t.Fatalf("matrix manifest: %v", err)
	}
	digest, err := contract.ProfileDigest(mf)
	if err != nil {
		t.Fatal(err)
	}
	impl := &registry.Implementation{Build: "test", Workflows: map[string]registry.WorkflowDef{
		tool: {Name: tool, Version: "1",
			Register: func(env registry.WorkflowEnv) error { return nil },
			Start: func(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
				return nil
			}},
	}}
	env := contract.ReleaseEnvelope{Manifest: digest, Implementation: "test",
		Build: "test", SourceRevision: "test", Artifact: digest}
	if err := reg.Register(mf, impl, env); err != nil {
		t.Fatal(err)
	}
}

func p25mService(t *testing.T, db *testkit.DB) *Service {
	t.Helper()
	reg := registry.New()
	p25mRegisterGated(t, reg, p25mGated, p25mGatedTl,
		[]contract.TrustMode{contract.TrustConfirm, contract.TrustAutonomous})
	p25mRegisterGated(t, reg, p25mConfOnly, p25mConfTl,
		[]contract.TrustMode{contract.TrustConfirm})
	grants := map[string]map[string]contract.TrustMode{
		p25mTenant: {p25mGated: contract.TrustAutonomous, p25mConfOnly: contract.TrustAutonomous},
	}
	return New(Config{Pool: db.Pool,
		Registry: reg,
		Grants:   auth.NewPolicy(map[string]auth.Binding{}, grants),
		Approver: approvals.New(db.Pool),
		Auditor:  audit.New(db.Pool, nil),
	})
}

func p25mSeedInvocation(t *testing.T, db *testkit.DB, local, status, trust, miniapp, tool string) string {
	t.Helper()
	id := "22222222-3333-4444-5555-" + local
	_, err := store.WithTenant(context.Background(), db.Pool, p25mTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', $6, $7, $8)`,
			id, p25mTenant, miniapp, tool, "key-"+local, status, trust, p25mToken)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func p25mSeedEffect(t *testing.T, db *testkit.DB, op, inv string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, p25mTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p25m', '', $3, 'mock', 'send', '{}', 'PENDING')`,
			op, p25mTenant, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// p25mApproval submits linkage for an invocation and applies the
// given disposition: approve, reject, expire (approved then
// backdated), mismatch (wrong intent hash), or none (left pending).
func p25mApproval(t *testing.T, db *testkit.DB, inv, miniapp, tool, disposition string) {
	t.Helper()
	ctx := context.Background()
	appr := approvals.New(db.Pool)
	hash := contract.IntentHash(miniapp, "1.0.0", tool, json.RawMessage(`{}`))
	if disposition == "mismatch" {
		wrong := sha256.Sum256([]byte("p25m-wrong-intent"))
		hash = contract.Digest("sha256:" + hex.EncodeToString(wrong[:]))
	}
	id, err := appr.Submit(ctx, approvals.Request{
		TenantID: p25mTenant, MiniappID: miniapp, Version: "1.0.0",
		InvocationID: inv, Operation: tool, IntentHash: hash,
		ApproverScope: "tenant-ops", RequestedBy: "agent-a",
		Expiry: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	decider := approvals.Decider{
		Caller: contract.Caller{AgentID: "human-op",
			Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "human-op", TenantID: p25mTenant}},
		Roles: []string{"tenant-ops"},
	}
	switch disposition {
	case "approve", "expire", "mismatch":
		if err := appr.Decide(ctx, p25mTenant, id, decider, contract.DecisionApprove, hash); err != nil {
			t.Fatal(err)
		}
	case "reject":
		if err := appr.Decide(ctx, p25mTenant, id, decider, contract.DecisionReject, hash); err != nil {
			t.Fatal(err)
		}
	case "cancel":
		if _, err := appr.CancelPending(ctx, p25mTenant, inv); err != nil {
			t.Fatal(err)
		}
	case "none":
	default:
		t.Fatalf("unknown disposition %q", disposition)
	}
	if disposition == "expire" {
		_, err := store.WithTenant(ctx, db.Pool, p25mTenant, func(tx pgx.Tx) (bool, error) {
			_, err := tx.Exec(ctx, `UPDATE app.approvals SET expires_at = now() - interval '1 hour'
				WHERE id = $1::uuid`, id)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func p25mAuthorize(t *testing.T, svc *Service, op, inv string) (contract.EffectRecord, error) {
	t.Helper()
	return svc.authorizeAttempt(context.Background(), p25mTenant, Intent{
		BusinessOpID: op, InvocationID: inv, EffectToken: p25mToken,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
	}, AuthorizeFresh, "a-p25m")
}

func p25mRefusal(t *testing.T, err error) *RefusalError {
	t.Helper()
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want *RefusalError", err)
	}
	return refusal
}

func p25mState(t *testing.T, db *testkit.DB, op string) string {
	t.Helper()
	st, err := store.WithTenant(context.Background(), db.Pool, p25mTenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(), `SELECT state FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, p25mTenant).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func p25mSetup(t *testing.T) (*testkit.DB, *Service) {
	t.Helper()
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.effect_records")
	return db, p25mService(t, db)
}

// 1. Gated + Confirm + valid approval → authorized.
func TestP25M1GatedConfirmGranted(t *testing.T) {
	db, svc := p25mSetup(t)
	op := p25mTenant + ":p25m1"
	inv := p25mSeedInvocation(t, db, "000000000001", "running", "confirm", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)
	p25mApproval(t, db, inv, p25mGated, p25mGatedTl, "approve")

	rec, err := p25mAuthorize(t, svc, op, inv)
	if err != nil {
		t.Fatalf("authorize err = %v, want nil", err)
	}
	if rec.State != contract.EffectAttempting {
		t.Errorf("state = %s, want ATTEMPTING", rec.State)
	}
}

// 2. Gated + Confirm + missing approval → refused approval-missing.
func TestP25M2GatedConfirmMissing(t *testing.T) {
	db, svc := p25mSetup(t)
	op := p25mTenant + ":p25m2"
	inv := p25mSeedInvocation(t, db, "000000000002", "running", "confirm", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)

	_, err := p25mAuthorize(t, svc, op, inv)
	if r := p25mRefusal(t, err); r.Reason != RefusalApprovalMissing {
		t.Errorf("reason = %s, want approval-missing", r.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
}

// 3. Gated + Autonomous + manifest permits → authorized, no linkage.
func TestP25M3GatedAutonomousPermitted(t *testing.T) {
	db, svc := p25mSetup(t)
	op := p25mTenant + ":p25m3"
	inv := p25mSeedInvocation(t, db, "000000000003", "running", "autonomous", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)

	rec, err := p25mAuthorize(t, svc, op, inv)
	if err != nil {
		t.Fatalf("authorize err = %v, want nil (autonomous exception)", err)
	}
	if rec.State != contract.EffectAttempting {
		t.Errorf("state = %s, want ATTEMPTING", rec.State)
	}
}

// 4. Gated + Autonomous + manifest forbids → refused
// trust-mode-mismatch (never authorized via the exception).
func TestP25M4GatedAutonomousForbidden(t *testing.T) {
	db, svc := p25mSetup(t)
	op := p25mTenant + ":p25m4"
	inv := p25mSeedInvocation(t, db, "000000000004", "running", "autonomous", p25mConfOnly, p25mConfTl)
	p25mSeedEffect(t, db, op, inv)

	_, err := p25mAuthorize(t, svc, op, inv)
	if r := p25mRefusal(t, err); r.Reason != RefusalTrustModeMismatch {
		t.Errorf("reason = %s, want trust-mode-mismatch", r.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
}

// 5. The discriminator is persisted trust: same tool, same caller
// shape — persisted-confirm without linkage refuses via the
// confirm path, persisted-autonomous without linkage authorizes.
// Intent carries no trust field, so no caller claim can flip this.
func TestP25M5PersistedTrustDecides(t *testing.T) {
	db, svc := p25mSetup(t)
	opC := p25mTenant + ":p25m5c"
	invC := p25mSeedInvocation(t, db, "000000000005", "running", "confirm", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, opC, invC)
	opA := p25mTenant + ":p25m5a"
	invA := p25mSeedInvocation(t, db, "000000000006", "running", "autonomous", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, opA, invA)

	_, errC := p25mAuthorize(t, svc, opC, invC)
	if r := p25mRefusal(t, errC); r.Reason != RefusalApprovalMissing {
		t.Errorf("confirm leg reason = %s, want approval-missing (exception not applied)", r.Reason)
	}
	recA, errA := p25mAuthorize(t, svc, opA, invA)
	if errA != nil {
		t.Fatalf("autonomous leg err = %v, want nil", errA)
	}
	if recA.State != contract.EffectAttempting {
		t.Errorf("autonomous leg state = %s, want ATTEMPTING", recA.State)
	}
}

// 6. Hostile approvals on the confirm path are fully evaluated —
// never bypassed. Plus the option-(a) exception shape: a present
// hostile approval under persisted-autonomous does not block the
// linkage exception (mirrors the submit choke, which never loads
// linkage for autonomous trust) — discriminating, since pre-fix
// code evaluates the present denial and refuses.
// Owner scope confirmed (P26-001, Prompt27; contract §16):
// the anti-bypass rule binds persisted-Confirm invocations
// (anti-spoofing); inapplicable rows create no requirement
// for genuinely persisted-Autonomous invocations.
func TestP25M6HostileApprovalsEvaluated(t *testing.T) {
	cases := []struct {
		name        string
		disposition string
		want        RefusalReason
	}{
		{"denied", "reject", RefusalApprovalDenied},
		{"expired", "expire", RefusalApprovalExpired},
		{"mismatched", "mismatch", RefusalApprovalMismatch},
		{"cancelled", "cancel", RefusalApprovalCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, svc := p25mSetup(t)
			op := p25mTenant + ":p25m6"
			inv := p25mSeedInvocation(t, db, "000000000007", "running", "confirm", p25mGated, p25mGatedTl)
			p25mSeedEffect(t, db, op, inv)
			p25mApproval(t, db, inv, p25mGated, p25mGatedTl, tc.disposition)

			_, err := p25mAuthorize(t, svc, op, inv)
			if r := p25mRefusal(t, err); r.Reason != tc.want {
				t.Errorf("reason = %s, want %s", r.Reason, tc.want)
			}
			if st := p25mState(t, db, op); st != "PENDING" {
				t.Errorf("row state = %s, want PENDING (untouched)", st)
			}
		})
	}
	t.Run("autonomous-exception-ignores-present-denial", func(t *testing.T) {
		db, svc := p25mSetup(t)
		op := p25mTenant + ":p25m6a"
		inv := p25mSeedInvocation(t, db, "000000000008", "running", "autonomous", p25mGated, p25mGatedTl)
		p25mSeedEffect(t, db, op, inv)
		p25mApproval(t, db, inv, p25mGated, p25mGatedTl, "reject")

		rec, err := p25mAuthorize(t, svc, op, inv)
		if err != nil {
			t.Fatalf("authorize err = %v, want nil (exception mirrors submit choke)", err)
		}
		if rec.State != contract.EffectAttempting {
			t.Errorf("state = %s, want ATTEMPTING", rec.State)
		}
	})
}

// 7. A terminal invocation cannot exploit the autonomous
// exception: refusal with P24-001 withdrawal semantics, never
// authorization.
func TestP25M7TerminalCannotExploit(t *testing.T) {
	db, svc := p25mSetup(t)
	op := p25mTenant + ":p25m7"
	inv := p25mSeedInvocation(t, db, "000000000009", "cancelled", "autonomous", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)

	_, err := p25mAuthorize(t, svc, op, inv)
	if !errors.Is(err, ErrInvocationCancelled) {
		t.Fatalf("authorize err = %v, want ErrInvocationCancelled", err)
	}
	if st := p25mState(t, db, op); st != "CANCELLED" {
		t.Errorf("row state = %s, want CANCELLED (withdrawn, never authorized)", st)
	}
}
