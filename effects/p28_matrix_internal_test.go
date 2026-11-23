package effects

// Prompt28 workstream A — authorization matrix legs (direct
// authorizeAttempt). Each refusal asserts: exact terminal
// classification, no ATTEMPTING (row untouched), error shape,
// and audit behavior. No provider invocation is proven by the
// row never leaving PENDING: sends require an ATTEMPTING
// record, which only a successful authorization produces.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

const p28TenantB = "tenant-p28b"

func p28Audits(t *testing.T, db *testkit.DB, op, action string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, p25mTenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = $2 AND details->>'business_op_id' = $3`,
			p25mTenant, action, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// p28SeedInvocation seeds an invocation under an explicit tenant.
func p28SeedInvocation(t *testing.T, db *testkit.DB, tenant, local, status, trust, miniapp, tool string) string {
	t.Helper()
	id := "33333333-4444-5555-6666-" + local
	_, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', $6, $7, $8)`,
			id, tenant, miniapp, tool, "key-"+local, status, trust, p25mToken)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// Wrong tenant: authorizing as a tenant that does not own the
// invocation is a terminal binding refusal with no cross-tenant
// information leak (generic message) and no audit (nothing
// attributable was loaded).
func TestP28AuthorizeWrongTenant(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25mService(t, db)
	op := p25mTenant + ":p28-tenant"
	inv := p25mSeedInvocation(t, db, "000000000011", "running", "confirm", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)

	_, err := svc.authorizeAttempt(context.Background(), p28TenantB, Intent{
		BusinessOpID: op, InvocationID: inv, EffectToken: p25mToken,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
	}, AuthorizeFresh, "a-p28")
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want terminal *RefusalError", err)
	}
	if refusal.Reason != RefusalBindingMismatch {
		t.Errorf("reason = %s, want binding-mismatch", refusal.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	if n := p28Audits(t, db, op, "effect-refused"); n != 0 {
		t.Errorf("effect-refused audits = %d, want 0 (unattributable skip)", n)
	}
}

// Wrong invocation binding: an effect bound to one live
// invocation cannot authorize under another.
func TestP28AuthorizeWrongInvocation(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25mService(t, db)
	op := p25mTenant + ":p28-bind"
	inv1 := p25mSeedInvocation(t, db, "000000000012", "running", "autonomous", p25mGated, p25mGatedTl)
	inv2 := p25mSeedInvocation(t, db, "000000000013", "running", "autonomous", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv1)

	_, err := svc.authorizeAttempt(context.Background(), p25mTenant, Intent{
		BusinessOpID: op, InvocationID: inv2, EffectToken: p25mToken,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
	}, AuthorizeFresh, "a-p25m")
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want terminal *RefusalError", err)
	}
	if refusal.Reason != RefusalBindingMismatch {
		t.Errorf("reason = %s, want binding-mismatch", refusal.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	if n := p28Audits(t, db, op, "effect-refused"); n != 1 {
		t.Errorf("effect-refused audits = %d, want exactly 1", n)
	}
}

// Unknown miniapp/tool: binding fails closed before any lock
// beyond the invocation row is taken.
func TestP28AuthorizeUnknownTool(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25mService(t, db)
	op := p25mTenant + ":p28-tool"
	inv := p25mSeedInvocation(t, db, "000000000014", "running", "autonomous", "nope-missing", "nope.tool")
	p25mSeedEffect(t, db, op, inv)

	_, err := svc.authorizeAttempt(context.Background(), p25mTenant, Intent{
		BusinessOpID: op, InvocationID: inv, EffectToken: p25mToken,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
	}, AuthorizeFresh, "a-p25m")
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want terminal *RefusalError", err)
	}
	if refusal.Reason != RefusalBindingMismatch {
		t.Errorf("reason = %s, want binding-mismatch", refusal.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	if n := p28Audits(t, db, op, "effect-refused"); n != 1 {
		t.Errorf("effect-refused audits = %d, want exactly 1", n)
	}
}

// Terminal sweep: succeeded and failed invocations withdraw
// exactly like cancelled ones (P24-001 mechanics).
func TestP28TerminalSucceededFailedWithdraw(t *testing.T) {
	for _, status := range []string{"succeeded", "failed"} {
		t.Run(status, func(t *testing.T) {
			db := testkit.NewDB(t)
			db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
			svc := p25mService(t, db)
			op := p25mTenant + ":p28-term-" + status
			inv := p25mSeedInvocation(t, db, "000000000015", status, "autonomous", p25mGated, p25mGatedTl)
			p25mSeedEffect(t, db, op, inv)

			if err := p25mAuthorizeTerminal(t, svc, op, inv); !errors.Is(err, ErrInvocationCancelled) {
				t.Fatalf("err = %v, want ErrInvocationCancelled", err)
			}
			if st := p25mState(t, db, op); st != "CANCELLED" {
				t.Errorf("row state = %s, want CANCELLED", st)
			}
			if n := p28Audits(t, db, op, "effect-withdrawn"); n != 1 {
				t.Errorf("effect-withdrawn audits = %d, want exactly 1", n)
			}
			if n := p28Audits(t, db, op, "effect-refused"); n != 0 {
				t.Errorf("effect-refused audits = %d, want 0 (won path audits withdrawn only)", n)
			}
		})
	}
}

func p25mAuthorizeTerminal(t *testing.T, svc *Service, op, inv string) error {
	t.Helper()
	_, err := svc.authorizeAttempt(context.Background(), p25mTenant, Intent{
		BusinessOpID: op, InvocationID: inv, EffectToken: p25mToken,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
	}, AuthorizeFresh, "a-p25m")
	return err
}

// Live waiting states authorize: waiting_approval and
// waiting_event are live like running.
func TestP28LiveWaitingStatesAuthorize(t *testing.T) {
	for _, status := range []string{"waiting_approval", "waiting_event"} {
		t.Run(status, func(t *testing.T) {
			db := testkit.NewDB(t)
			db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
			svc := p25mService(t, db)
			op := p25mTenant + ":p28-live-" + status
			inv := p25mSeedInvocation(t, db, "000000000016", status, "autonomous", p25mGated, p25mGatedTl)
			p25mSeedEffect(t, db, op, inv)

			rec, err := p25mAuthorize(t, svc, op, inv)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if rec.State != contract.EffectAttempting {
				t.Errorf("state = %s, want ATTEMPTING", rec.State)
			}
		})
	}
}

// Pending valid approval: authorize waits — ErrApprovalPending
// is returned (not a refusal), with no state change, no audit,
// and no ATTEMPTING row.
func TestP28PendingApprovalWaits(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25mService(t, db)
	op := p25mTenant + ":p28-wait"
	inv := p25mSeedInvocation(t, db, "000000000017", "running", "confirm", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)
	p25mApproval(t, db, inv, p25mGated, p25mGatedTl, "none")

	_, err := p25mAuthorize(t, svc, op, inv)
	if !errors.Is(err, ErrApprovalPending) {
		t.Fatalf("err = %v, want ErrApprovalPending (wait, not refusal)", err)
	}
	var refusal *RefusalError
	if errors.As(err, &refusal) {
		t.Fatalf("wait classified as refusal (%s); waits must not refuse", refusal.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	if n := p28Audits(t, db, op, "effect-refused"); n != 0 {
		t.Errorf("effect-refused audits = %d, want 0 (waits are not refusals)", n)
	}
	if n := p28Audits(t, db, op, "effect-withdrawn"); n != 0 {
		t.Errorf("effect-withdrawn audits = %d, want 0", n)
	}
}
