package effects_test

// Prompt28 workstream B — deterministic race cases with complete
// evidence (winner, loser, persisted state, audit, provider
// count, retry classification, attempt epochs). Ordering is
// forced with gates and row locks; never sleeps or repetition.

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
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

type p28RowView struct {
	state   string
	attempt string
	req     string
	reply   string
}

func p28ReadRow(t *testing.T, db *testkit.DB, op string) p28RowView {
	t.Helper()
	v, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (p28RowView, error) {
		var v p28RowView
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

func p28RefusedAudits(t *testing.T, db *testkit.DB, op string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-refused'
			AND details->>'business_op_id' = $2`, wsTenant, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// B2/B9 — worker-vs-worker on one effect, full evidence. The
// gate-held winner commits ATTEMPTING first (entered marker);
// every loser answers contention; the winner's receipt lands
// exactly once and contention audits nothing.
func TestP28SameEffectRaceFullEvidence(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	gate := &gateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": gate}, allowCancellation{}, allowPolicy{})
	seedLiveInvocation(t, db, wsTenant, "inv-p28race")
	op := wsTenant + ":p28race"
	intent := mockIntent("p28race", mockcap.ModeAccept)

	const losers = 5
	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	winner := make(chan outcome, 1)
	go func() {
		rec, err := svc.Perform(ctx, wsTenant, intent)
		winner <- outcome{rec, err}
	}()
	select {
	case <-gate.entered:
	case <-time.After(60 * time.Second):
		t.Fatal("winner never reached Send")
	}
	winnerRow := p28ReadRow(t, db, op)
	if winnerRow.state != "ATTEMPTING" {
		t.Fatalf("winner row = %s, want ATTEMPTING", winnerRow.state)
	}
	var wg sync.WaitGroup
	lost := make(chan outcome, losers)
	for i := 0; i < losers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec, err := svc.Perform(ctx, wsTenant, intent)
			lost <- outcome{rec, err}
		}()
	}
	wg.Wait()
	close(lost)
	for o := range lost {
		if !errors.Is(o.err, effects.ErrAttemptInProgress) {
			t.Errorf("loser err = %v, want contention", o.err)
		}
	}
	close(gate.release)
	var w outcome
	select {
	case w = <-winner:
	case <-time.After(60 * time.Second):
		t.Fatal("winner never completed after release")
	}
	if w.err != nil || w.rec.State != contract.EffectConfirmed {
		t.Fatalf("winner = %+v %v, want CONFIRMED", w.rec, w.err)
	}
	final := p28ReadRow(t, db, op)
	if final.state != "CONFIRMED" || final.attempt != winnerRow.attempt {
		t.Errorf("final = (%s, %s), want (CONFIRMED, winner epoch %s)",
			final.state, final.attempt, winnerRow.attempt)
	}
	if final.req != "r1" {
		t.Errorf("final receipt = %q, want the winner's r1", final.req)
	}
	if gate.sends != 1 {
		t.Errorf("sends = %d, want exactly 1", gate.sends)
	}
	if n := p28RefusedAudits(t, db, op); n != 0 {
		t.Errorf("refused audits = %d, want 0 (contention is not refusal)", n)
	}
}

// B4 — lease turnover during a live authorization. A holds
// ATTEMPTING behind a gate; the row ages; B's touch
// quarantines to UNKNOWN; A's released outcome loses as
// superseded. No operator Settle, no reconcile: pure
// turnover-vs-authorization.
func TestP28LeaseTurnoverDuringAuthorize(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	gate := &gateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": gate}, allowCancellation{}, allowPolicy{})
	seedLiveInvocation(t, db, wsTenant, "inv-p28turn")
	op := wsTenant + ":p28turn"
	intent := mockIntent("p28turn", mockcap.ModeAccept)

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
	case <-gate.entered:
	case <-time.After(60 * time.Second):
		t.Fatal("epoch A never reached Send")
	}
	rowA := p28ReadRow(t, db, op)
	if rowA.state != "ATTEMPTING" {
		t.Fatalf("epoch A row = %s, want ATTEMPTING", rowA.state)
	}
	p28AgeRow(t, db, op)
	touch, err := svc.Perform(ctx, wsTenant, intent)
	if err != nil || touch.State != contract.EffectUnknown {
		t.Fatalf("touch = %+v %v, want UNKNOWN adopt", touch, err)
	}
	close(gate.release)
	var a outcome
	select {
	case a = <-doneA:
	case <-time.After(60 * time.Second):
		t.Fatal("epoch A never returned after release")
	}
	if !errors.Is(a.err, effects.ErrAttemptSuperseded) {
		t.Errorf("epoch A err = %v, want superseded", a.err)
	}
	final := p28ReadRow(t, db, op)
	if final.state != "UNKNOWN" {
		t.Errorf("final state = %s, want UNKNOWN", final.state)
	}
	if final.attempt != rowA.attempt {
		t.Errorf("final attempt = %s, want retained epoch %s", final.attempt, rowA.attempt)
	}
	if gate.sends != 1 {
		t.Errorf("sends = %d, want exactly 1 (A attempted once, outcome fenced)", gate.sends)
	}
}

// B8 — concurrent Settle-vs-Settle, forced order. S1 queues
// first behind a held effect row, S2 queues behind S1; FIFO
// grants S1 the win and S2 the deterministic conflict. The
// audit records exactly one settlement.
func TestP28ConcurrentSettleSingleWinner(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": mockcap.New(db.Pool)}, allowCancellation{}, allowPolicy{})
	seedLiveInvocation(t, db, wsTenant, "inv-p28settle")
	op := wsTenant + ":p28settle"
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, $2, 'a-p28s', '', $3, 'mock', 'send', '{}', 'UNKNOWN')`,
			op, wsTenant, authzInvID(wsTenant, "inv-p28settle"))
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	holder, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	if _, err := holder.Exec(ctx, store.TenantContextSQL(), wsTenant); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := holder.QueryRow(ctx, `SELECT business_op_id FROM app.effect_records
		WHERE business_op_id = $1 FOR UPDATE`, op).Scan(&held); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	done1 := make(chan outcome, 1)
	go func() {
		rec, err := svc.Settle(ctx, wsTenant, op, contract.EffectConfirmed, "p28-first", "p28-op")
		done1 <- outcome{rec, err}
	}()
	p28WaitLockWaiter(t, db.Pool, "UPDATE app.effect_records", "first Settle")
	done2 := make(chan outcome, 1)
	go func() {
		rec, err := svc.Settle(ctx, wsTenant, op, contract.EffectFailed, "p28-second", "p28-op")
		done2 <- outcome{rec, err}
	}()
	p28WaitLockWaiterN(t, db.Pool, "UPDATE app.effect_records", 2, "both Settles")
	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var o1, o2 outcome
	select {
	case o1 = <-done1:
	case <-time.After(60 * time.Second):
		t.Fatal("first Settle never completed")
	}
	select {
	case o2 = <-done2:
	case <-time.After(60 * time.Second):
		t.Fatal("second Settle never completed")
	}
	if o1.err != nil || o1.rec.State != contract.EffectConfirmed {
		t.Fatalf("winner = %+v %v, want CONFIRMED", o1.rec, o1.err)
	}
	// S2 read UNKNOWN then lost the predicated UPDATE: superseded,
	// while the winner's decision stands untouched.
	if !errors.Is(o2.err, effects.ErrAttemptSuperseded) {
		t.Errorf("loser err = %v, want ErrAttemptSuperseded (first decision wins)", o2.err)
	}
	final := p28ReadRow(t, db, op)
	if final.state != "CONFIRMED" || final.reply != "p28-first" {
		t.Errorf("final = (%s, %q), want (CONFIRMED, p28-first)", final.state, final.reply)
	}
	n, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-settled'
			AND details->>'business_op_id' = $2`, wsTenant, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("settled audits = %d, want exactly 1", n)
	}
}

// B8 hammer — settlers racing touchers converge exactly-once:
// terminal state, at most one settlement audit, no lost
// updates, and only terminal-shaped errors.
func TestP28SettleHammerConverges(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	ctx := context.Background()
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": mockcap.New(db.Pool)}, allowCancellation{}, allowPolicy{})
	seedLiveInvocation(t, db, wsTenant, "inv-p28shammer")
	op := wsTenant + ":p28shammer"
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state, updated_at)
			VALUES ($1, $2, 'a-p28h', '', $3, 'mock', 'send', '{}', 'UNKNOWN', now() - interval '1 hour')`,
			op, wsTenant, authzInvID(wsTenant, "inv-p28shammer"))
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 12; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			note := fmt.Sprintf("p28-hammer-%d", i)
			_, err := svc.Settle(ctx, wsTenant, op, contract.EffectConfirmed, note, "p28-op")
			errs <- err
		}(i)
		go func(i int) {
			defer wg.Done()
			_, err := svc.Settle(ctx, wsTenant, op, contract.EffectNotApplied, fmt.Sprintf("p28-hammer-na-%d", i), "p28-op")
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	wins := 0
	for err := range errs {
		if err == nil {
			wins++
			continue
		}
		if !errors.Is(err, effects.ErrAlreadySettled) && !errors.Is(err, effects.ErrAttemptSuperseded) {
			t.Errorf("hammer err = %v, want nil, ErrAlreadySettled, or ErrAttemptSuperseded", err)
		}
	}
	if wins != 1 {
		t.Errorf("settlement wins = %d, want exactly 1", wins)
	}
	final := p28ReadRow(t, db, op)
	if final.state != "CONFIRMED" && final.state != "NOT_APPLIED" {
		t.Errorf("final state = %s, want a settled terminal", final.state)
	}
	n, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM app.audit_events
			WHERE tenant_id = $1 AND action = 'effect-settled'
			AND details->>'business_op_id' = $2`, wsTenant, op).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("settled audits = %d, want exactly 1", n)
	}
}

func p28WaitLockWaiter(t *testing.T, pool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, needle, what string) {
	t.Helper()
	p28WaitLockWaiterN(t, pool, needle, 1, what)
}

func p28WaitLockWaiterN(t *testing.T, pool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, needle string, want int, what string) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND pid <> pg_backend_pid()
			AND state = 'active' AND wait_event_type = 'Lock'
			AND position($1 in query) > 0`, needle).Scan(&n)
		if err != nil {
			t.Fatalf("pg_stat_activity poll (%s): %v", what, err)
		}
		if n >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s (needle %q, want %d waiters)", what, needle, want)
}
