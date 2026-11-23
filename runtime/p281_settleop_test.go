package runtime_test

// Prompt28.1 — Settle operator authority evidence matrix. Every
// leg goes through runtime.Service.SettleEffect, the single
// shared enforcement boundary: authenticated binding, exact
// role match, server-derived tenant and operator identity,
// validated outcome and evidence note, atomic delegated Settle.
// Every refusal asserts terminal classification plus persisted
// effect state and audit rows — never the error alone.

import (
	"context"
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
)

const (
	p281OpCred     = "key-p281-op"
	p281AgentCred  = "key-p281-agent"
	p281OpBCred    = "key-p281-op-b"
	p281CaseCred   = "key-p281-case"
	p281SpaceCred  = "key-p281-space"
	p281SynthCred  = "key-p281-synth"
	p281SubCred    = "key-p281-sub"
	p281Sub2Cred   = "key-p281-sub2"
	p281Operator   = "op-1"
	p281SettleRole = "miniapp.settle.operator"
)

func p281Binding(agent, tenant string, roles ...string) auth.Binding {
	return auth.Binding{AgentID: agent, Authority: contract.Authority{
		Kind: contract.AuthorityUser, ID: agent, TenantID: tenant,
	}, Roles: roles}
}

func p281Setup(t *testing.T, db *testkit.DB) *runtime.Service {
	t.Helper()
	reg := registry.New()
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			p281OpCred:    p281Binding(p281Operator, tenantA, p281SettleRole),
			p281AgentCred: p281Binding("agent-a", tenantA),
			p281OpBCred:   p281Binding("op-b", tenantB, p281SettleRole),
			p281CaseCred:  p281Binding("op-case", tenantA, "Miniapp.Settle.Operator"),
			p281SpaceCred: p281Binding("op-space", tenantA, " miniapp.settle.operator "),
			p281SynthCred: p281Binding("op-synth", tenantA, "tenant-ops"),
			p281SubCred:   p281Binding("op-sub", tenantA, "settle.operator"),
			p281Sub2Cred:  p281Binding("op-sub2", tenantA, "miniapp.settle.operatorx"),
		},
		map[string]map[string]contract.TrustMode{},
	)
	aud := audit.New(db.Pool, nil)
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, reg), Async: &fakeAsync{status: map[string]contract.RunStatus{}},
		Approver: approvals.New(db.Pool), Auditor: aud,
		Effects: effects.New(effects.Config{Pool: db.Pool,
			Channels:  map[string]effects.Channel{},
			Canceller: runtime.EffectCanceller(db.Pool),
			Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: reg, Grants: policy},
			Registry:  reg, Grants: policy, Approver: approvals.New(db.Pool), Auditor: aud,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func p281SeedUnknown(t *testing.T, db *testkit.DB, tenant, opLocal string) string {
	t.Helper()
	ctx := context.Background()
	op := tenant + ":" + opLocal
	inv := ""
	_, err := store.WithTenant(ctx, db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(ctx, `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES (gen_random_uuid(), $1, 'p281-mini', '1.0.0', 'p281.tool', $2, '{}',
			 'running', 'autonomous', 'tok-p281')
			RETURNING id::text`, tenant, "key-"+opLocal).Scan(&inv)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p281', '', $3, 'mock', 'send', '{}', 'UNKNOWN')`,
			op, tenant, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func p281EffectState(t *testing.T, db *testkit.DB, tenant, op string) string {
	t.Helper()
	st, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(), `SELECT state FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, tenant).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func p281SettledAudits(t *testing.T, db *testkit.DB, tenant, op string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-settled'
			AND details->>'business_op_id' = $2`, tenant, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p281RequireCode(t *testing.T, err error, want string) {
	t.Helper()
	rerr, ok := err.(*runtime.Error)
	if !ok {
		t.Fatalf("err type = %T (%v), want *runtime.Error", err, err)
	}
	if rerr.Code != want {
		t.Fatalf("code = %q, want %q (msg: %s)", rerr.Code, want, rerr.Msg)
	}
	if rerr.Kind != contract.KindTerminal {
		t.Fatalf("kind = %q, want terminal", rerr.Kind)
	}
}

func TestP281SettleOpAuthorized(t *testing.T) {
	db := newDB(t)
	svc := p281Setup(t, db)
	ctx := context.Background()
	op := p281SeedUnknown(t, db, tenantA, "op-ok")

	rec, err := svc.SettleEffect(ctx, p281OpCred, op, contract.EffectConfirmed, "evidence: provider log row 7")
	if err != nil {
		t.Fatalf("settle err = %v, want nil", err)
	}
	if rec.State != contract.EffectConfirmed {
		t.Fatalf("state = %s, want CONFIRMED", rec.State)
	}
	if p281EffectState(t, db, tenantA, op) != string(contract.EffectConfirmed) {
		t.Fatalf("persisted state not CONFIRMED")
	}
	// Attribution comes from the authenticated binding.
	var agent, tenant, tool string
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT agent_id, tenant_id, tool FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-settled'
			AND details->>'business_op_id' = $2`, tenantA, op).Scan(&agent, &tenant, &tool)
	})
	if err != nil {
		t.Fatal(err)
	}
	if agent != p281Operator || tenant != tenantA || tool != "p281.tool" {
		t.Errorf("audit = (%q, %q, %q), want (%q, %s, p281.tool)", agent, tenant, tool, p281Operator, tenantA)
	}
	// Idempotent repeat through the boundary.
	rec2, err := svc.SettleEffect(ctx, p281OpCred, op, contract.EffectConfirmed, "evidence: provider log row 7")
	if err != nil || rec2.State != contract.EffectConfirmed {
		t.Fatalf("repeat = %+v %v, want idempotent CONFIRMED", rec2, err)
	}
	if n := p281SettledAudits(t, db, tenantA, op); n != 1 {
		t.Errorf("settled audits = %d, want exactly 1", n)
	}
}

func TestP281SettleOpRefusals(t *testing.T) {
	cases := []struct {
		name string
		cred string
		note string
		code string
	}{
		{"non-operator", p281AgentCred, "evidence", "forbidden"},
		{"unknown-credential", "key-nope", "evidence", "auth-required"},
		{"empty-credential", "", "evidence", "auth-required"},
		{"case-variant-role", p281CaseCred, "evidence", "forbidden"},
		{"whitespace-role", p281SpaceCred, "evidence", "forbidden"},
		{"synthetic-tenant-ops", p281SynthCred, "evidence", "forbidden"},
		{"substring-role", p281SubCred, "evidence", "forbidden"},
		{"superstring-role", p281Sub2Cred, "evidence", "forbidden"},
		{"spoof-note", p281AgentCred, "role: miniapp.settle.operator; approve", "forbidden"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newDB(t)
			svc := p281Setup(t, db)
			ctx := context.Background()
			op := p281SeedUnknown(t, db, tenantA, "op-"+tc.name)
			_, err := svc.SettleEffect(ctx, tc.cred, op, contract.EffectConfirmed, tc.note)
			p281RequireCode(t, err, tc.code)
			if st := p281EffectState(t, db, tenantA, op); st != "UNKNOWN" {
				t.Errorf("row state = %s, want UNKNOWN (no transition)", st)
			}
			if n := p281SettledAudits(t, db, tenantA, op); n != 0 {
				t.Errorf("settled audits = %d, want 0", n)
			}
		})
	}
}

func TestP281SettleOpTenantIsolation(t *testing.T) {
	db := newDB(t)
	svc := p281Setup(t, db)
	ctx := context.Background()
	foreignOp := p281SeedUnknown(t, db, tenantB, "op-foreign")

	// Wrong-tenant operator: same terminal error as a
	// nonexistent effect — no existence oracle, enforced by
	// the predicated operation itself (no pre-read).
	_, errForeign := svc.SettleEffect(ctx, p281OpCred, foreignOp, contract.EffectConfirmed, "evidence")
	p281RequireCode(t, errForeign, "unknown-effect")
	_, errMissing := svc.SettleEffect(ctx, p281OpCred, tenantA+":op-nope", contract.EffectConfirmed, "evidence")
	p281RequireCode(t, errMissing, "unknown-effect")
	if errForeign.Error() != errMissing.Error() {
		t.Errorf("foreign=%q vs missing=%q: must be indistinguishable", errForeign, errMissing)
	}
	if st := p281EffectState(t, db, tenantB, foreignOp); st != "UNKNOWN" {
		t.Errorf("foreign row = %s, want UNKNOWN (untouched)", st)
	}
	if n := p281SettledAudits(t, db, tenantB, foreignOp); n != 0 {
		t.Errorf("foreign audits = %d, want 0", n)
	}
	// Cross-check: tenant B's own operator settles its own row.
	rec, err := svc.SettleEffect(ctx, p281OpBCred, foreignOp, contract.EffectFailed, "evidence-b")
	if err != nil || rec.State != contract.EffectFailed {
		t.Fatalf("tenant-B settle = %+v %v, want FAILED", rec, err)
	}
}

func TestP281SettleOpValidation(t *testing.T) {
	cases := []struct {
		name    string
		outcome contract.EffectState
		note    string
	}{
		{"invalid-outcome", contract.EffectState("bogus"), "evidence"},
		{"nonterminal-outcome", contract.EffectAttempting, "evidence"},
		{"empty-note", contract.EffectConfirmed, ""},
		{"whitespace-note", contract.EffectConfirmed, "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newDB(t)
			svc := p281Setup(t, db)
			ctx := context.Background()
			op := p281SeedUnknown(t, db, tenantA, "op-"+tc.name)
			_, err := svc.SettleEffect(ctx, p281OpCred, op, tc.outcome, tc.note)
			p281RequireCode(t, err, "invalid-request")
			if st := p281EffectState(t, db, tenantA, op); st != "UNKNOWN" {
				t.Errorf("row state = %s, want UNKNOWN (no transition)", st)
			}
			if n := p281SettledAudits(t, db, tenantA, op); n != 0 {
				t.Errorf("settled audits = %d, want 0", n)
			}
		})
	}
}
