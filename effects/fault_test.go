// Fault-injection tests for effect boundaries: reservation,
// attempt begin, takeover begin, outcome settlement (commit and
// response), operator settlement, provider send, and policy check.
// Each test proves firing, propagation, and persisted-state
// correctness; pre-commit failure means rollback, post-commit
// response loss means idempotent recovery.
package effects_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/fault"
	"github.com/PithomLabs/miniapp/testkit/faultwrap"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

var errFault = errors.New("injected fault")

// TestEffectsNilTestFault proves production construction leaves
// injection disabled.
func TestEffectsNilTestFault(t *testing.T) {
	if svc := effects.New(effects.Config{}); svc.TestFault != nil {
		t.Fatal("TestFault != nil after production New")
	}
}

// effectState reads one ledger row directly (a read-only probe;
// never performs). ok is false when no row exists.
func effectState(t *testing.T, db *testkit.DB, tenant, businessOpID string) (contract.EffectState, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	st, err := store.WithTenant(ctx, db.Pool, tenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(ctx, `SELECT state FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, businessOpID, tenant).Scan(&s)
		if err != nil {
			return "", err
		}
		return s, nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false
		}
		t.Fatal(err)
	}
	return contract.EffectState(st), true
}

func ageEffectRow(t *testing.T, db *testkit.DB, tenant, businessOpID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := store.WithTenant(ctx, db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE app.effect_records SET updated_at = now() - interval '1 hour'
			WHERE business_op_id = $1`, businessOpID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestFaultEffectReserve: claim INSERT fails → error, no row, no
// send; retry converges.
func TestFaultEffectReserve(t *testing.T) {
	svc, db := setupMock(t)
	inj := fault.New()
	inj.FailAt("effect-reserve", 1, errFault)
	svc.TestFault = inj
	ctx := context.Background()
	intent := mockIntent("op-freserve", mockcap.ModeAccept)

	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, errFault) {
		t.Fatalf("Perform err = %v, want injected fault", err)
	}
	if n := inj.Count("effect-reserve"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if _, ok := effectState(t, db, wsTenant, wsTenant+":op-freserve"); ok {
		t.Fatal("effect row exists despite failed reservation")
	}
	if n := mockcap.SentCount(t, db.Pool, "op-freserve"); n != 0 {
		t.Fatalf("sends = %d, want 0", n)
	}
	if rec, err := svc.Perform(ctx, wsTenant, intent); err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("retry = %+v, %v; want CONFIRMED", rec, err)
	}
}

// TestFaultAttemptBegin: PENDING→ATTEMPTING fails → error, row stays
// PENDING, no send; retry converges.
func TestFaultAttemptBegin(t *testing.T) {
	svc, db := setupMock(t)
	inj := fault.New()
	inj.FailAt("attempt-begin", 1, errFault)
	svc.TestFault = inj
	ctx := context.Background()
	intent := mockIntent("op-fattempt", mockcap.ModeAccept)

	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, errFault) {
		t.Fatalf("Perform err = %v, want injected fault", err)
	}
	st, ok := effectState(t, db, wsTenant, wsTenant+":op-fattempt")
	if !ok || st != contract.EffectPending {
		t.Fatalf("state = %s/%v, want PENDING (no attempt began)", st, ok)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fattempt"); n != 0 {
		t.Fatalf("sends = %d, want 0", n)
	}
	// An immediate retry meets fresh-lease contention (correct: the
	// row may belong to a live worker); the contention path sends
	// nothing and changes no state. After the lease lapses the
	// retry takes over and converges without a duplicate send.
	// Every phase transition is state-driven (lease timestamps,
	// direct row reads); no sleeps gate any step.
	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, effects.ErrAttemptInProgress) {
		t.Fatalf("immediate retry err = %v, want ErrAttemptInProgress", err)
	}
	if st, _ := effectState(t, db, wsTenant, wsTenant+":op-fattempt"); st != contract.EffectPending {
		t.Fatalf("state after contention = %s, want PENDING (unchanged)", st)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fattempt"); n != 0 {
		t.Fatalf("sends after contention = %d, want 0", n)
	}
	ageEffectRow(t, db, wsTenant, wsTenant+":op-fattempt")
	rec, err := svc.Perform(ctx, wsTenant, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("aged retry = %+v, %v; want CONFIRMED", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fattempt"); n != 1 {
		t.Fatalf("sends = %d, want 1", n)
	}
}

// TestFaultTakeoverBegin: stale-PENDING takeover fails before its
// transition → error, row stays stale PENDING, no send.
func TestFaultTakeoverBegin(t *testing.T) {
	svc, db := setupMock(t)
	ctx := context.Background()
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, capability_id, operation, payload, state, updated_at)
			VALUES ($1, $2, 'a0', 'mock', 'send', '{"mock":{"mode":"accept","opId":"op-ftake"}}', 'PENDING', now() - interval '1 hour')`,
			wsTenant+":op-ftake", wsTenant)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	inj := fault.New()
	inj.FailAt("takeover-begin", 1, errFault)
	svc.TestFault = inj

	if _, err := svc.Perform(ctx, wsTenant, mockIntent("op-ftake", mockcap.ModeAccept)); !errors.Is(err, errFault) {
		t.Fatalf("Perform err = %v, want injected fault", err)
	}
	st, _ := effectState(t, db, wsTenant, wsTenant+":op-ftake")
	if st != contract.EffectPending {
		t.Fatalf("state = %s, want PENDING (takeover refused)", st)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-ftake"); n != 0 {
		t.Fatalf("sends = %d, want 0", n)
	}
}

// TestFaultSettleCommit: outcome settlement fails after the send →
// error, provider sent once, row stays ATTEMPTING; aging converges
// to UNKNOWN with no second send.
func TestFaultSettleCommit(t *testing.T) {
	svc, db := setupMock(t)
	inj := fault.New()
	inj.FailAt("settle-commit", 1, errFault)
	svc.TestFault = inj
	ctx := context.Background()
	intent := mockIntent("op-fsettle", mockcap.ModeAccept)

	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, errFault) {
		t.Fatalf("Perform err = %v, want injected fault", err)
	}
	st, _ := effectState(t, db, wsTenant, wsTenant+":op-fsettle")
	if st != contract.EffectAttempting {
		t.Fatalf("state = %s, want ATTEMPTING (outcome unrecorded)", st)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fsettle"); n != 1 {
		t.Fatalf("sends = %d, want 1 (send preceded the failed settle)", n)
	}
	ageEffectRow(t, db, wsTenant, wsTenant+":op-fsettle")
	rec, err := svc.Perform(ctx, wsTenant, intent)
	if err != nil || rec.State != contract.EffectUnknown {
		t.Fatalf("aged retry = %+v, %v; want UNKNOWN", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fsettle"); n != 1 {
		t.Fatalf("sends after recovery = %d, want 1 (no blind retry)", n)
	}
}

// TestFaultSettleResponse: outcome commits, then the response is
// lost → caller sees the error but the outcome stands; a repeat
// observes CONFIRMED without re-sending.
func TestFaultSettleResponse(t *testing.T) {
	svc, db := setupMock(t)
	inj := fault.New()
	inj.FailAt("settle-response", 1, errFault)
	svc.TestFault = inj
	ctx := context.Background()
	intent := mockIntent("op-fresp", mockcap.ModeAccept)

	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, errFault) {
		t.Fatalf("Perform err = %v, want injected fault", err)
	}
	st, _ := effectState(t, db, wsTenant, wsTenant+":op-fresp")
	if st != contract.EffectConfirmed {
		t.Fatalf("state = %s, want CONFIRMED (commit stood)", st)
	}
	rec, err := svc.Perform(ctx, wsTenant, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("repeat = %+v, %v; want CONFIRMED without error", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fresp"); n != 1 {
		t.Fatalf("sends = %d, want 1 (no resend on recovery)", n)
	}
}

// TestFaultSettleOperatorCommit: operator settlement fails before
// its transition → error, row stays UNKNOWN; retry converges.
func TestFaultSettleOperatorCommit(t *testing.T) {
	svc, db := setupMock(t)
	ctx := context.Background()
	if rec, err := svc.Perform(ctx, wsTenant, mockIntent("op-fopsettle", mockcap.ModeLostAck)); err != nil || rec.State != contract.EffectUnknown {
		t.Fatalf("lost-ack setup = %+v, %v; want UNKNOWN", rec, err)
	}
	inj := fault.New()
	inj.FailAt("settle-operator-commit", 1, errFault)
	svc.TestFault = inj

	if _, err := svc.Settle(ctx, wsTenant, wsTenant+":op-fopsettle", contract.EffectFailed, "evidence"); !errors.Is(err, errFault) {
		t.Fatalf("Settle err = %v, want injected fault", err)
	}
	st, _ := effectState(t, db, wsTenant, wsTenant+":op-fopsettle")
	if st != contract.EffectUnknown {
		t.Fatalf("state = %s, want UNKNOWN (settlement rolled back)", st)
	}
	rec, err := svc.Settle(ctx, wsTenant, wsTenant+":op-fopsettle", contract.EffectFailed, "evidence")
	if err != nil || rec.State != contract.EffectFailed {
		t.Fatalf("retry = %+v, %v; want FAILED", rec, err)
	}
}

// TestFaultProviderSend: the channel wrapper fails before delegating
// → the provider call never happens (0 sends) and the attempt
// settles FAILED, the honest lifecycle outcome for a failed call.
func TestFaultProviderSend(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	inj := fault.New()
	inj.FailAt(faultwrap.OpProviderSend, 1, errFault)
	svc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": faultwrap.Channel{Inner: mockcap.New(db.Pool), Inj: inj}},
		Canceller: allowCancellation{}, Policy: allowPolicy{}})
	ctx := context.Background()

	rec, err := svc.Perform(ctx, wsTenant, mockIntent("op-fsend", mockcap.ModeAccept))
	if err != nil {
		t.Fatalf("Perform err = %v; provider failure settles the attempt, want (FAILED, nil)", err)
	}
	if rec.State != contract.EffectFailed {
		t.Fatalf("state = %s, want FAILED", rec.State)
	}
	if n := inj.Count(faultwrap.OpProviderSend); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-fsend"); n != 0 {
		t.Fatalf("delegate sends = %d, want 0 (wrapper fired first)", n)
	}
}

// TestFaultPolicyCheck: policy evaluation fails → the claim is
// refused before any reservation; no row, no send.
func TestFaultPolicyCheck(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	inj := fault.New()
	inj.FailAt(faultwrap.OpPolicyCheck, 1, errFault)
	svc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: allowCancellation{}, Policy: faultwrap.Policy{Inner: allowPolicy{}, Inj: inj}})
	ctx := context.Background()

	if _, err := svc.Perform(ctx, wsTenant, mockIntent("op-fpolicy", mockcap.ModeAccept)); !errors.Is(err, errFault) {
		t.Fatalf("Perform err = %v, want injected fault", err)
	}
	if n := inj.Count(faultwrap.OpPolicyCheck); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	if _, ok := effectState(t, db, wsTenant, wsTenant+":op-fpolicy"); ok {
		t.Fatal("effect row exists despite refused policy")
	}
}
