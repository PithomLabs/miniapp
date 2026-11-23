package effects_test

// P24-002 (Prompt 25): every outcome transition must be fenced by
// attempt identity. These tests re-implement POC-2's deterministic
// staging as committed regressions across all three late-outcome
// flavors: epoch A authorizes and blocks in Send, the row ages past
// the lease, a touch quarantines it to UNKNOWN, the operator settles
// NOT_APPLIED, epoch C reconciles and blocks — then A's released
// outcome must lose as superseded while C's outcome lands.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

// p25ScriptOutcome is one epoch's scripted provider result.
type p25ScriptOutcome struct {
	receipt effects.Receipt
	err     error
}

// p25ScriptChannel serves a scripted outcome per send, holding each
// epoch open until its release closes. enteredA/enteredC prove each
// epoch's authorization committed (sends happen only after commit).
type p25ScriptChannel struct {
	mu       sync.Mutex
	calls    int
	enteredA chan struct{}
	releaseA chan struct{}
	enteredC chan struct{}
	releaseC chan struct{}
	outA     p25ScriptOutcome
	outC     p25ScriptOutcome
}

func p25NewScriptChannel(outA, outC p25ScriptOutcome) *p25ScriptChannel {
	return &p25ScriptChannel{
		enteredA: make(chan struct{}), releaseA: make(chan struct{}),
		enteredC: make(chan struct{}), releaseC: make(chan struct{}),
		outA: outA, outC: outC,
	}
}

func (c *p25ScriptChannel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	c.mu.Lock()
	c.calls++
	n := c.calls
	c.mu.Unlock()
	switch n {
	case 1:
		close(c.enteredA)
		select {
		case <-c.releaseA:
			return c.outA.receipt, c.outA.err
		case <-ctx.Done():
			return effects.Receipt{}, ctx.Err()
		}
	case 2:
		close(c.enteredC)
		select {
		case <-c.releaseC:
			return c.outC.receipt, c.outC.err
		case <-ctx.Done():
			return effects.Receipt{}, ctx.Err()
		}
	default:
		return effects.Receipt{}, fmt.Errorf("p25: unexpected third send")
	}
}

func (c *p25ScriptChannel) ncalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type p25RowView struct {
	state   string
	attempt string
	req     string
	reply   string
}

func p25ReadRow(t *testing.T, db *testkit.DB, op string) p25RowView {
	t.Helper()
	v, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (p25RowView, error) {
		var v p25RowView
		err := tx.QueryRow(context.Background(), `SELECT state, attempt_id,
			provider_request, provider_reply FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, wsTenant).Scan(
			&v.state, &v.attempt, &v.req, &v.reply)
		return v, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func p25AgeRow(t *testing.T, db *testkit.DB, op string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `UPDATE app.effect_records
			SET updated_at = now() - interval '1 hour'
			WHERE business_op_id = $1 AND tenant_id = $2`, op, wsTenant)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// p25StaleFence runs the full POC-2 interleaving: A's late outcome
// (outA) must lose to C's live attempt, whose accept outcome lands.
func p25StaleFence(t *testing.T, flavor string, outA p25ScriptOutcome) {
	t.Helper()
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	outC := p25ScriptOutcome{receipt: effects.Receipt{ProviderRequestID: "p25-rC", ProviderReply: "p25-ackC"}}
	ch := p25NewScriptChannel(outA, outC)
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": ch}, allowCancellation{}, allowPolicy{})

	local := "p25a2-" + flavor
	seedLiveInvocation(t, db, wsTenant, local)
	inv := authzInvID(wsTenant, local)
	op := wsTenant + ":p25a2-" + flavor
	intent := effects.Intent{BusinessOpID: op, AttemptID: "a-p25a2",
		InvocationID: inv, EffectToken: authzToken,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send"}}

	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	doneA := make(chan outcome, 1)
	go func() {
		rec, err := svc.Perform(ctx, wsTenant, intent)
		doneA <- outcome{rec, err}
	}()
	select {
	case <-ch.enteredA:
	case <-time.After(60 * time.Second):
		t.Fatal("epoch A send never entered (authorize did not commit ATTEMPTING)")
	}
	rowA := p25ReadRow(t, db, op)
	if rowA.state != "ATTEMPTING" {
		t.Fatalf("epoch A row state = %s, want ATTEMPTING", rowA.state)
	}

	// Age past the lease; a touch quarantines to UNKNOWN.
	p25AgeRow(t, db, op)
	if rec, err := svc.Perform(ctx, wsTenant, intent); err != nil || rec.State != contract.EffectUnknown {
		t.Fatalf("touch = %+v err=%v, want UNKNOWN adopt", rec, err)
	}
	if v := p25ReadRow(t, db, op); v.state != "UNKNOWN" {
		t.Fatalf("quarantined row state = %s, want UNKNOWN", v.state)
	}

	// Operator settles NOT_APPLIED; epoch C reconciles and blocks.
	if _, err := svc.Settle(ctx, wsTenant, op, contract.EffectNotApplied, "p25-note", "p25-op"); err != nil {
		t.Fatalf("settle: %v", err)
	}
	doneC := make(chan outcome, 1)
	go func() {
		rec, err := svc.Perform(ctx, wsTenant, intent)
		doneC <- outcome{rec, err}
	}()
	select {
	case <-ch.enteredC:
	case <-time.After(60 * time.Second):
		t.Fatal("epoch C send never entered (reconcile did not commit ATTEMPTING)")
	}
	rowC := p25ReadRow(t, db, op)
	if rowC.state != "ATTEMPTING" {
		t.Fatalf("epoch C row state = %s, want ATTEMPTING", rowC.state)
	}
	if rowC.attempt == "" || rowC.attempt == rowA.attempt {
		t.Fatalf("epoch C attempt = %q, want a fresh epoch distinct from %q", rowC.attempt, rowA.attempt)
	}

	// Release the stale epoch first: its outcome must lose.
	close(ch.releaseA)
	var errA error
	select {
	case o := <-doneA:
		errA = o.err
	case <-time.After(60 * time.Second):
		t.Fatal("epoch A never returned after release")
	}
	close(ch.releaseC)
	var recC contract.EffectRecord
	var errC error
	select {
	case o := <-doneC:
		recC, errC = o.rec, o.err
	case <-time.After(60 * time.Second):
		t.Fatal("epoch C never returned after release")
	}
	_ = recC

	if ch.ncalls() != 2 {
		t.Errorf("provider sends = %d, want exactly 2 (both epochs attempted)", ch.ncalls())
	}
	if !errors.Is(errA, effects.ErrAttemptSuperseded) {
		t.Errorf("epoch A err = %v, want superseded (stale outcome fenced)", errA)
	}
	if errC != nil {
		t.Errorf("epoch C err = %v, want nil (live outcome lands)", errC)
	}
	final := p25ReadRow(t, db, op)
	if final.state != "CONFIRMED" {
		t.Errorf("final state = %s, want CONFIRMED (epoch C's outcome)", final.state)
	}
	if final.attempt != rowC.attempt {
		t.Errorf("final attempt = %s, want epoch C's %s", final.attempt, rowC.attempt)
	}
	if final.req != "p25-rC" || final.reply != "p25-ackC" {
		t.Errorf("final receipt = (%q,%q), want epoch C's (p25-rC,p25-ackC)", final.req, final.reply)
	}
}

func TestP25POC2StaleConfirmedFenced(t *testing.T) {
	p25StaleFence(t, "confirmed", p25ScriptOutcome{
		receipt: effects.Receipt{ProviderRequestID: "p25-rA", ProviderReply: "p25-ackA"}})
}

func TestP25POC2StaleFailedFenced(t *testing.T) {
	p25StaleFence(t, "failed", p25ScriptOutcome{err: errors.New("p25-epochA-failed")})
}

func TestP25POC2StaleUnknownFenced(t *testing.T) {
	p25StaleFence(t, "unknown", p25ScriptOutcome{err: effects.UnknownError{Err: errors.New("p25-epochA-unknown")}})
}
