package effects_test

// Prompt28 workstream F — provider-path bypass assurance for the
// reconcile, unknown-state, and unknown-capability initiation
// paths: authorization precedes every provider touch, refusals
// send nothing, UNKNOWN is never resent, and unknown
// capabilities fail closed without sending.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

func p28SeedState(t *testing.T, db *testkit.DB, op, inv, state string) {
	t.Helper()
	p28SeedStateCap(t, db, op, inv, state, "mock")
}

func p28SeedStateCap(t *testing.T, db *testkit.DB, op, inv, state, cap string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p28bp', '', $3, $4, 'send',
			 '{"mock":{"mode":"accept","opId":"' || $5 || '"}}', $6)`,
			wsTenant+":"+op, wsTenant, inv, cap, op, state)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Reconciliation runs fresh effect-time authorization: a revoked
// grant refuses the NOT_APPLIED retry (no shortcut to send).
func TestP28ReconcileRevokedRefuses(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	policy := auth.NewPolicy(nil, map[string]map[string]contract.TrustMode{})
	svc := p28PolicySvc(t, db, policy)
	seedLiveInvocation(t, db, wsTenant, "inv-p28rec")
	inv := authzInvID(wsTenant, "inv-p28rec")
	p28SeedState(t, db, "p28rec", inv, "NOT_APPLIED")

	_, err := svc.Perform(context.Background(), wsTenant, mockIntent("p28rec", mockcap.ModeAccept))
	var refusal *effects.RefusalError
	if !errors.As(err, &refusal) || refusal.Reason != effects.RefusalGrantDenied {
		t.Fatalf("err = %v, want grant-denied refusal", err)
	}
	if v := p28ReadRow(t, db, wsTenant+":p28rec"); v.state != "NOT_APPLIED" {
		t.Errorf("row = %s, want NOT_APPLIED (untouched)", v.state)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28rec"); n != 0 {
		t.Errorf("sends = %d, want 0", n)
	}
	if n := p28RefusedAudits(t, db, wsTenant+":p28rec"); n != 1 {
		t.Errorf("refused audits = %d, want exactly 1", n)
	}
}

// Reconciliation against a terminal invocation refuses through
// the terminal branch (withdrawal loses on NOT_APPLIED).
func TestP28ReconcileTerminalRefuses(t *testing.T) {
	svc, db := setupMock(t)
	inv := authzInvID(wsTenant, "inv-p28rect")
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, 't-auth-fix', '1.0.0', 't.auth', 'key-p28rect', '{}',
			 'cancelled', 'autonomous', $3)`, inv, wsTenant, authzToken)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	p28SeedState(t, db, "p28rect", inv, "NOT_APPLIED")
	intent := mockIntent("p28rect", mockcap.ModeAccept)
	intent.InvocationID = inv
	_, err = svc.Perform(context.Background(), wsTenant, intent)
	if !errors.Is(err, effects.ErrInvocationCancelled) {
		t.Fatalf("err = %v, want terminal cancellation refusal", err)
	}
	if v := p28ReadRow(t, db, wsTenant+":p28rect"); v.state != "NOT_APPLIED" {
		t.Errorf("row = %s, want NOT_APPLIED (withdrawal loses, never resurrected)", v.state)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28rect"); n != 0 {
		t.Errorf("sends = %d, want 0", n)
	}
	if n := p28RefusedAudits(t, db, wsTenant+":p28rect"); n != 1 {
		t.Errorf("refused audits = %d, want exactly 1 (lost-withdraw refusal)", n)
	}
}

// UNKNOWN rows are never blindly resent: repeat reads the
// terminal outcome without provider contact.
func TestP28UnknownNeverResends(t *testing.T) {
	svc, db := setupMock(t)
	seedLiveInvocation(t, db, wsTenant, "inv-p28unk")
	p28SeedState(t, db, "p28unk", authzInvID(wsTenant, "inv-p28unk"), "UNKNOWN")
	intent := mockIntent("p28unk", mockcap.ModeAccept)
	intent.InvocationID = authzInvID(wsTenant, "inv-p28unk")
	for i := 0; i < 3; i++ {
		rec, err := svc.Perform(context.Background(), wsTenant, intent)
		if err != nil || rec.State != contract.EffectUnknown {
			t.Fatalf("repeat %d = %+v %v, want UNKNOWN read", i, rec, err)
		}
	}
	if n := mockcap.SentCount(t, db.Pool, "p28unk"); n != 0 {
		t.Errorf("sends = %d, want 0", n)
	}
}

// Unknown capabilities fail closed on every initiation path
// that reaches dispatch: the FAILED recording the code attempts
// is rejected by the state machine (PENDING/NOT_APPLIED have no
// FAILED edge), so the call errors, the row is unchanged, and
// nothing sends. Characterization: fail-closed but
// taxonomy-noisy (deterministic condition, infra-shaped
// error) — proper capability-membership refusal follows the
// P24-008 trigger with the first real provider.
func TestP28UnknownCapabilityFailsClosed(t *testing.T) {
	setup := []struct {
		name      string
		state     string
		aged      bool
		wantState string
	}{
		{"fresh", "", false, "PENDING"},
		{"takeover", "PENDING", true, "PENDING"},
		{"reconcile", "NOT_APPLIED", false, "NOT_APPLIED"},
	}
	for _, tc := range setup {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := setupMock(t)
			seedLiveInvocation(t, db, wsTenant, "inv-p28uc-"+tc.name)
			inv := authzInvID(wsTenant, "inv-p28uc-"+tc.name)
			op := "p28uc-" + tc.name
			if tc.state != "" {
				p28SeedStateCap(t, db, op, inv, tc.state, "no-such-cap")
				if tc.aged {
					p28AgeRow(t, db, wsTenant+":"+op)
				}
			}
			intent := mockIntent(op, mockcap.ModeAccept)
			intent.InvocationID = inv
			intent.Operation.CapabilityID = "no-such-cap"
			_, err := svc.Perform(context.Background(), wsTenant, intent)
			if err == nil {
				t.Errorf("%s: unknown capability accepted", tc.name)
			}
			if v := p28ReadRow(t, db, wsTenant+":"+op); v.state != tc.wantState {
				t.Errorf("%s row = %s, want %s (unchanged)", tc.name, v.state, tc.wantState)
			}
			if n := mockcap.SentCount(t, db.Pool, op); n != 0 {
				t.Errorf("%s sends = %d, want 0", tc.name, n)
			}
		})
	}
}
