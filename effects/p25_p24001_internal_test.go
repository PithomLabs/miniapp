package effects

// P24-001 (Prompt 25) direct authorization-path tests. These call
// authorizeAttempt without the pre-claim gate, covering the terminal
// branch outcomes a live-then-cancel race can reach: withdrawal won,
// withdrawal lost (already terminal or never pending), and the
// quarantined never-withdraw rule. The forced end-to-end race lives in
// runtime/p25_p24001_test.go; these pin the exact audit pairing per
// outcome without concurrency.

import (
	"context"
	"errors"
	"testing"

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
	p25Tenant = "tenant-p25"
	p25Token  = "tok-p25"
)

func p25AuthService(t *testing.T, db *testkit.DB) *Service {
	t.Helper()
	return New(Config{Pool: db.Pool,
		Registry: registry.New(),
		Grants:   auth.NewPolicy(map[string]auth.Binding{}, map[string]map[string]contract.TrustMode{}),
		Approver: approvals.New(db.Pool),
		Auditor:  audit.New(db.Pool, nil),
	})
}

func p25SeedInvocation(t *testing.T, db *testkit.DB, local, status string) string {
	t.Helper()
	id := "11111111-2222-3333-4444-" + local
	_, err := store.WithTenant(context.Background(), db.Pool, p25Tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, 'p25-mini', '1.0.0', 'p25.tool', $3, '{}', $4, 'autonomous', $5)`,
			id, p25Tenant, "key-"+local, status, p25Token)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func p25SeedEffect(t *testing.T, db *testkit.DB, op, inv, state string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, p25Tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p25-seed', '', $3, 'mock', 'send', '{}', $4)`,
			op, p25Tenant, inv, state)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func p25State(t *testing.T, db *testkit.DB, op string) string {
	t.Helper()
	st, err := store.WithTenant(context.Background(), db.Pool, p25Tenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(), `SELECT state FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, p25Tenant).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func p25Audits(t *testing.T, db *testkit.DB, op, action string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, p25Tenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = $2 AND details->>'business_op_id' = $3`,
			p25Tenant, action, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p25RefusalReason(t *testing.T, db *testkit.DB, op string) string {
	t.Helper()
	reason, err := store.WithTenant(context.Background(), db.Pool, p25Tenant, func(tx pgx.Tx) (string, error) {
		var r string
		err := tx.QueryRow(context.Background(), `SELECT details->>'reason' FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-refused'
			AND details->>'business_op_id' = $2 ORDER BY id DESC LIMIT 1`,
			p25Tenant, op).Scan(&r)
		return r, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return reason
}

func p25Authorize(t *testing.T, svc *Service, op, inv string) error {
	t.Helper()
	_, err := svc.authorizeAttempt(context.Background(), p25Tenant, Intent{
		BusinessOpID: op, InvocationID: inv, EffectToken: p25Token,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
	}, AuthorizeFresh, "a-p25")
	return err
}

// Terminal invocation plus PENDING row: the withdrawal wins, persists,
// and pairs with exactly one effect-withdrawn audit.
func TestP25TerminalWithdrawWon(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25AuthService(t, db)
	op := p25Tenant + ":p25-won"
	inv := p25SeedInvocation(t, db, "555555555555", "cancelled")
	p25SeedEffect(t, db, op, inv, "PENDING")

	if err := p25Authorize(t, svc, op, inv); !errors.Is(err, ErrInvocationCancelled) {
		t.Fatalf("authorize err = %v, want ErrInvocationCancelled", err)
	}
	if st := p25State(t, db, op); st != "CANCELLED" {
		t.Errorf("row state = %s, want CANCELLED", st)
	}
	if n := p25Audits(t, db, op, "effect-withdrawn"); n != 1 {
		t.Errorf("effect-withdrawn audits = %d, want exactly 1", n)
	}
	if n := p25Audits(t, db, op, "effect-refused"); n != 0 {
		t.Errorf("effect-refused audits = %d, want 0", n)
	}
}

// Terminal invocation plus already-CANCELLED row (an authorize-reaching
// repeat): nothing resurrects, nothing re-withdraws; the refusal
// audits once as effect-refused with the ineligibility reason.
func TestP25TerminalWithdrawLostRepeat(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25AuthService(t, db)
	op := p25Tenant + ":p25-lost"
	inv := p25SeedInvocation(t, db, "666666666666", "cancelled")
	p25SeedEffect(t, db, op, inv, "CANCELLED")

	if err := p25Authorize(t, svc, op, inv); !errors.Is(err, ErrInvocationCancelled) {
		t.Fatalf("authorize err = %v, want ErrInvocationCancelled", err)
	}
	if st := p25State(t, db, op); st != "CANCELLED" {
		t.Errorf("row state = %s, want CANCELLED", st)
	}
	if n := p25Audits(t, db, op, "effect-withdrawn"); n != 0 {
		t.Errorf("effect-withdrawn audits = %d, want 0 (never re-withdraw)", n)
	}
	if n := p25Audits(t, db, op, "effect-refused"); n != 1 {
		t.Errorf("effect-refused audits = %d, want exactly 1", n)
	}
	if r := p25RefusalReason(t, db, op); r != "invocation-ineligible" {
		t.Errorf("refusal reason = %q, want invocation-ineligible", r)
	}
}

// Terminal invocation plus a never-pending row: withdrawal loses, the
// row is untouched, and the refusal audits once as effect-refused.
func TestP25TerminalWithdrawLostNotApplied(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25AuthService(t, db)
	op := p25Tenant + ":p25-na"
	inv := p25SeedInvocation(t, db, "777777777777", "failed")
	p25SeedEffect(t, db, op, inv, "NOT_APPLIED")

	if err := p25Authorize(t, svc, op, inv); !errors.Is(err, ErrInvocationCancelled) {
		t.Fatalf("authorize err = %v, want ErrInvocationCancelled", err)
	}
	if st := p25State(t, db, op); st != "NOT_APPLIED" {
		t.Errorf("row state = %s, want NOT_APPLIED (untouched)", st)
	}
	if n := p25Audits(t, db, op, "effect-withdrawn"); n != 0 {
		t.Errorf("effect-withdrawn audits = %d, want 0", n)
	}
	if n := p25Audits(t, db, op, "effect-refused"); n != 1 {
		t.Errorf("effect-refused audits = %d, want exactly 1", n)
	}
}

// Quarantined invocations stay resumable: refusal without withdrawal,
// the PENDING row untouched, no withdrawal audit.
func TestP25QuarantinedNeverWithdraws(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25AuthService(t, db)
	op := p25Tenant + ":p25-quar"
	inv := p25SeedInvocation(t, db, "888888888888", "quarantined")
	p25SeedEffect(t, db, op, inv, "PENDING")

	if err := p25Authorize(t, svc, op, inv); !errors.Is(err, ErrInvocationCancelled) {
		t.Fatalf("authorize err = %v, want ErrInvocationCancelled", err)
	}
	if st := p25State(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	if n := p25Audits(t, db, op, "effect-withdrawn"); n != 0 {
		t.Errorf("effect-withdrawn audits = %d, want 0", n)
	}
	if n := p25Audits(t, db, op, "effect-refused"); n != 1 {
		t.Errorf("effect-refused audits = %d, want exactly 1", n)
	}
}
