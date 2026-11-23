// MEDIUM-2 classa-level tests: claim orchestration, pre-handler
// liveness gate, handler-error savepoint semantics, and mandatory
// hooks.
//
// GREEN-ONLY SOURCE: references ClaimHooks/Request.Claim, which do
// not exist pre-fix. Excluded from the red overlay per discipline
// (behavioral red/green lives in runtime/p20_medium2_test.go,
// which uses only stable APIs).
package classa_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
)

// TestP20NilClaimFailsClosed (condition 4): without claim hooks
// the executor refuses to run. The handler never executes: no
// silent legacy two-transaction path exists.
func TestP20NilClaimFailsClosed(t *testing.T) {
	var calls atomic.Int32
	ex, _ := setup(t, func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		calls.Add(1)
		return json.RawMessage(`{}`), nil
	})
	r := req("tenant-a:sub1", "inv-nilclaim")
	r.Claim = nil
	if _, err := ex.Execute(context.Background(), r); err == nil {
		t.Fatal("Execute without Claim succeeded; must fail closed")
	} else if got := err.Error(); !strings.Contains(got, "claim hooks required") {
		t.Fatalf("err = %q, want claim-hooks-required refusal", got)
	}
	r.Claim = &executor.ClaimHooks{}
	if _, err := ex.Execute(context.Background(), r); err == nil {
		t.Fatal("Execute with empty Claim succeeded; must fail closed")
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("handler executions = %d, want 0", n)
	}
}

// TestP20TerminalRecheckSkipsHandler (condition 1, unit aspect):
// when the invocation is already terminal at claim time, the
// attempt rolls back before the handler. The full liveness path
// (real record states) is proven at the runtime level.
func TestP20TerminalRecheckSkipsHandler(t *testing.T) {
	var calls atomic.Int32
	ex, _ := setup(t, func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		calls.Add(1)
		return json.RawMessage(`{}`), nil
	})
	r := req("tenant-a:sub1", "inv-termrecheck")
	r.Claim = &executor.ClaimHooks{
		Recheck: func(tx pgx.Tx) (bool, error) { return false, nil },
		Complete: func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
			t.Error("Complete ran after a terminal recheck; must not")
			return false, nil
		},
	}
	_, err := ex.Execute(context.Background(), r)
	if !errors.Is(err, executor.ErrExecutionSuperseded) {
		t.Fatalf("err = %v, want ErrExecutionSuperseded", err)
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("handler executions = %d, want 0", n)
	}
}

// TestP20HandlerErrorRollsBackBusiness (condition 2): a failing
// handler's database writes roll back to savepoint while the
// FAILED outcome commits and the claim releases. Partial handler
// writes are never committed.
func TestP20HandlerErrorRollsBackBusiness(t *testing.T) {
	exFail, db := setup(t, func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		if _, err := call.Tx.Exec(ctx, `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
			VALUES ('p20c2', 'tenant-a', 'p20c2', 'p20c2@example.com')`); err != nil {
			return nil, err
		}
		return nil, errP20Boom
	})
	db.Truncate(t, "app.crm_contacts")
	var completed atomic.Int32
	var gotStatus contract.RunStatus
	var gotErr string
	hooks := &executor.ClaimHooks{
		Recheck: func(tx pgx.Tx) (bool, error) { return true, nil },
		Complete: func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
			completed.Add(1)
			gotStatus, gotErr = status, errMsg
			return true, nil
		},
	}
	r := req("", "inv-handlerfail")
	r.Claim = hooks
	_, err := exFail.Execute(context.Background(), r)
	var herr *executor.HandlerFailedError
	if !errors.As(err, &herr) {
		t.Fatalf("err = %v, want HandlerFailedError", err)
	}
	if !errors.Is(herr.Err, errP20Boom) {
		t.Fatalf("inner err = %v, want the handler error preserved", herr.Err)
	}
	// Tenant context is mandatory: forced RLS would hide a
	// committed row and let the assertion pass vacuously.
	n, err := store.WithTenant(context.Background(), db.Pool, "tenant-a", func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.crm_contacts WHERE id = 'p20c2'`).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("business rows = %d, want 0 (handler writes must roll back)", n)
	}
	if completed.Load() != 1 || gotStatus != contract.RunFailed || gotErr != errP20Boom.Error() {
		t.Fatalf("complete calls = %d status = %s err = %q; want exactly one failed completion",
			completed.Load(), gotStatus, gotErr)
	}
	// The claim released with the commit: the same invocation
	// executes normally afterwards.
	exOK, _ := setup(t, func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		return json.RawMessage(`{"ok":true}`), nil
	})
	r2 := req("", "inv-handlerfail")
	r2.Claim = hooks
	if _, err := exOK.Execute(context.Background(), r2); err != nil {
		t.Fatalf("post-failure execute err = %v (claim must have released)", err)
	}
}

var errP20Boom = errors.New("p20 boom")

// TestP20ConcurrentClaimSingleWinner pins claim mutual exclusion
// at the executor level: the loser gets ErrExecutionInProgress
// without reaching the handler, deterministically (the winner's
// handler-entry signal proves overlap before the loser starts).
func TestP20ConcurrentClaimSingleWinner(t *testing.T) {
	entered := make(chan struct{}, 16)
	release := make(chan struct{})
	var once sync.Once
	shut := func() { once.Do(func() { close(release) }) }
	defer shut()
	var calls atomic.Int32
	ex, _ := setup(t, func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		return json.RawMessage(`{"ok":true}`), nil
	})
	type out struct {
		res json.RawMessage
		err error
	}
	wOut := make(chan out, 1)
	go func() {
		res, err := ex.Execute(context.Background(), req("tenant-a:sub1", "inv-claimrace"))
		wOut <- out{res, err}
	}()
	select {
	case <-entered:
	case <-time.After(30 * time.Second):
		shut()
		t.Fatal("winner never entered the handler")
	}
	// Loser runs while the winner holds the claim: prompt
	// conflict, no handler entry. (A hang here fails the suite
	// loudly instead of passing vacuously.)
	lDone := make(chan out, 1)
	go func() {
		_, err := ex.Execute(context.Background(), req("tenant-a:sub1", "inv-claimrace"))
		lDone <- out{nil, err}
	}()
	var l out
	select {
	case l = <-lDone:
	case <-time.After(30 * time.Second):
		shut()
		t.Fatal("loser never returned (must conflict promptly, not wait)")
	}
	if !errors.Is(l.err, executor.ErrExecutionInProgress) {
		shut()
		t.Fatalf("loser err = %v, want ErrExecutionInProgress", l.err)
	}
	shut()
	var w out
	select {
	case w = <-wOut:
	case <-time.After(30 * time.Second):
		t.Fatal("winner never returned")
	}
	if w.err != nil {
		t.Fatalf("winner err = %v", w.err)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("handler executions = %d, want exactly 1", n)
	}
}
