// HIGH-1 discriminating tests: settlement-versus-completion races.
//
// RED/GREEN-COMPATIBLE SOURCE: uses only public service APIs, SQL
// fixtures, pg_stat_activity observation, the shared gate helpers
// from x03_test.go, and prod/testkit APIs verified present on the
// acceptance tree. The red run overlays exactly this file onto the
// frozen pre-remediation copy. A granted poll cannot race a settle
// on one invocation (approval-state monotonicity — see the Phase 0
// reachability row), so the completion leg runs through
// trust-float dispatch: same executeSync/completion contention,
// reachable and API-level.
package runtime_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
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
	p20Miniapp = "test-p20a"
	p20Tool    = "probe.p20a"
	p20Result  = `{"p20":"settled"}`
)

// p20Gate is a channel-blocked Class A handler: entry is
// signaled, the handler parks until release closes, exactly one
// canned result returns, and every execution is counted.
type p20Gate struct {
	entered chan string
	release chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func newP20Gate() *p20Gate {
	return &p20Gate{entered: make(chan string, 1), release: make(chan struct{})}
}

func (g *p20Gate) handler() registry.ToolFunc {
	return func(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
		g.calls.Add(1)
		g.entered <- call.InvocationID
		<-g.release
		return json.RawMessage(p20Result), nil
	}
}

func (g *p20Gate) shutdown() {
	g.once.Do(func() { close(g.release) })
}

// p20Policy grants the probe miniapp autonomous trust, so the
// completion leg floats past the approval requirement while the
// park leg (confirm) still parks.
func p20Policy() *auth.Policy {
	return auth.NewPolicy(
		map[string]auth.Binding{
			credAgentA: {AgentID: "agent-a", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "agent-a", TenantID: tenantA,
			}},
		},
		map[string]map[string]contract.TrustMode{
			tenantA: {p20Miniapp: contract.TrustAutonomous},
		},
	)
}

type p20Fixture struct {
	svc      *runtime.Service
	async    *fakeAsync
	approver *approvals.Service
}

// newP20Fixture wires a service around one gated Class A probe
// tool with a channel-blocked handler. Self-contained for red
// compatibility (prod constructor shapes match newRedFixture).
func newP20Fixture(t *testing.T, db *testkit.DB, gate *p20Gate) p20Fixture {
	t.Helper()
	reg := registry.New()
	mf := testkit.ClassAManifest(p20Miniapp, p20Tool, "", nil)
	mf.Tools[0].Approval = contract.ApprovalRequirement{
		Required: true, ApproverScope: "tenant-ops", Expiry: "1h",
	}
	mf.Tools[0].TrustModes = []contract.TrustMode{contract.TrustConfirm, contract.TrustAutonomous}
	if err := reg.Register(mf, testkit.ClassAImpl("test", p20Tool, gate.handler()),
		testkit.Envelope(t, mf, "test")); err != nil {
		t.Fatal(err)
	}
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	aud := audit.New(db.Pool, nil)
	approver := approvals.New(db.Pool)
	policy := p20Policy()
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
	return p20Fixture{svc: svc, async: async, approver: approver}
}

// seedPendingRow inserts the crash-window shape: a claimed but
// undispatched (never parked) Class A invocation.
func seedPendingRow(t *testing.T, db *testkit.DB, key string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', 'pending', 'confirm', 'tok-p20')`,
			uuid.NewString(), tenantA, p20Miniapp, p20Tool, key)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func p20InvID(t *testing.T, db *testkit.DB, key string) string {
	t.Helper()
	invID, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (string, error) {
		var id string
		err := tx.QueryRow(context.Background(),
			`SELECT id::text FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&id)
		return id, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return invID
}

func p20ExpireApproval(t *testing.T, db *testkit.DB, approvalID string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(),
			`UPDATE app.approvals SET expires_at = now() - interval '1 hour' WHERE id = $1::uuid`, approvalID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func p20AuditCount(t *testing.T, db *testkit.DB, action, outcome string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.audit_events WHERE action = $1 AND outcome = $2`,
			action, outcome).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p20Record(t *testing.T, db *testkit.DB, key string) (contract.RunStatus, json.RawMessage) {
	t.Helper()
	type row struct {
		status contract.RunStatus
		result json.RawMessage
	}
	r, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (row, error) {
		var s string
		var res []byte
		err := tx.QueryRow(context.Background(),
			`SELECT status, result FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&s, &res)
		return row{contract.RunStatus(s), json.RawMessage(res)}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return r.status, r.result
}

type p20Out struct {
	res runtime.InvokeResult
	err error
}

// p20Round owns one round's cleanup: holder release, handler
// release, and goroutine join run on every path, including
// timeout failures. Timeouts only bound failure; they never
// branch success logic.
type p20Round struct {
	t       *testing.T
	release func()
	once    sync.Once
	gate    *p20Gate
	wg      *sync.WaitGroup
}

func (r *p20Round) cleanup() {
	r.once.Do(func() { r.release() })
	r.gate.shutdown()
	r.wg.Wait()
}

func (r *p20Round) recv(out <-chan p20Out, what string) p20Out {
	r.t.Helper()
	select {
	case o := <-out:
		return o
	case <-time.After(30 * time.Second):
		r.cleanup()
		r.t.Fatalf("timed out waiting for %s", what)
		return p20Out{}
	}
}

func (r *p20Round) recvStr(ch <-chan string, what string) string {
	r.t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(30 * time.Second):
		r.cleanup()
		r.t.Fatalf("timed out waiting for %s", what)
		return ""
	}
}

func (r *p20Round) failf(format string, args ...any) {
	r.t.Helper()
	r.cleanup()
	r.t.Fatalf(format, args...)
}

// p20Stage runs the shared prefix: B (trust-float dispatch) parks
// in the channel-blocked handler, C (confirm) parks the approval,
// the approval flips to past-expiry, and the row gate is taken.
// B's handler-entry signal proves B reached execution before C
// parked; C's synchronous return proves the park committed.
func p20Stage(t *testing.T, db *testkit.DB, fx p20Fixture, gate *p20Gate, key string,
	bOut chan<- p20Out, wg *sync.WaitGroup) (holder int32, rd *p20Round) {
	t.Helper()
	holder, release := 0, func() {}
	rd = &p20Round{t: t, gate: gate, wg: wg}
	rd.release = func() { release() }
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := invoke(fx.svc, p20Miniapp, p20Tool, `{}`, key, credAgentA, contract.TrustAutonomous)
		bOut <- p20Out{res, err}
	}()
	rd.recvStr(gate.entered, "handler entry (B must reach execution before C parks)")
	parked, err := invoke(fx.svc, p20Miniapp, p20Tool, `{}`, key, credAgentA, contract.TrustConfirm)
	if err != nil || parked.Status != contract.RunWaitingApproval || parked.ApprovalID == "" {
		rd.failf("park leg = %+v, %v; want waiting_approval with approval ID", parked, err)
	}
	p20ExpireApproval(t, db, parked.ApprovalID)
	invID := p20InvID(t, db, key)
	holder, release = rowGateHolder(t, db, tenantA, invID)
	return holder, rd
}

// TestP20SettleWinsOverCompletion forces settlement-first: the
// expiry settle commits while the handler is parked, then the
// completion loses, discards the handler result, and answers the
// recorded terminal with no success audit. Commits are strictly
// sequential (A's return proves its commit before the handler is
// released), so no queue-order assumption is involved. Red: the
// blind completion overwrites the settlement with false success.
func TestP20SettleWinsOverCompletion(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
		gate := newP20Gate()
		fx := newP20Fixture(t, db, gate)
		key := "key-p20h1-A"
		seedPendingRow(t, db, key)
		var wg sync.WaitGroup
		bOut := make(chan p20Out, 1)
		holder, rd := p20Stage(t, db, fx, gate, key, bOut, &wg)
		aOut := make(chan p20Out, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := invoke(fx.svc, p20Miniapp, p20Tool, `{}`, key, credAgentA, contract.TrustConfirm)
			aOut <- p20Out{res, err}
		}()
		// A must be blocked at failIfWaiting behind OUR holder
		// before it may proceed.
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
			rd.failf("round %d: settle gate never formed", round)
		}
		rd.once.Do(func() { rd.release() })
		a := rd.recv(aOut, "settle return (proves the failed commit landed)")
		gate.shutdown()
		b := rd.recv(bOut, "completion return (must lose to the settlement)")
		wg.Wait()
		if a.err == nil || !strings.Contains(a.err.Error(), "approval-expired") {
			t.Fatalf("round %d: settle err = %+v, want terminal approval-expired", round, a.err)
		}
		if b.err == nil || !strings.Contains(b.err.Error(), "recorded-failure") {
			t.Fatalf("round %d: completion err = %+v, want recorded-failure replay", round, b.err)
		}
		st, res := p20Record(t, db, key)
		if st != contract.RunFailed {
			t.Fatalf("round %d: record = %s, want failed (settlement stands)", round, st)
		}
		if len(res) != 0 && string(res) != "null" {
			t.Fatalf("round %d: record result = %s, want empty (stale handler result discarded)", round, res)
		}
		if n := p20AuditCount(t, db, "approval-expired", "expired"); n != 1 {
			t.Fatalf("round %d: approval-expired audits = %d, want exactly 1", round, n)
		}
		if n := p20AuditCount(t, db, "invoke", "succeeded"); n != 0 {
			t.Fatalf("round %d: invoke/succeeded audits = %d, want 0 (no false success)", round, n)
		}
		if n := gate.calls.Load(); n != 1 {
			t.Fatalf("round %d: handler executions = %d, want 1", round, n)
		}
		fx.async.mu.Lock()
		submits := len(fx.async.submits)
		fx.async.mu.Unlock()
		if submits != 0 {
			t.Fatalf("round %d: workflow submits = %d, want 0 (Class A only)", round, submits)
		}
	}
}

// TestP20CompletionWinsOverSettle forces completion-first: B's
// completion UPDATE queues at the row gate (verified as the sole
// waiter while it is the only request in flight), then A's settle
// queues behind it, and the release commits in queue order. The
// post-hoc assertions admit only the completion-first outcome:
// row succeeded with B's result, B answered success, A answered
// the same success via replay, and zero settle audits. No
// retries: any other grant order fails loudly. Red: the blind
// settle overwrites the completion with false failure.
func TestP20CompletionWinsOverSettle(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
		gate := newP20Gate()
		fx := newP20Fixture(t, db, gate)
		key := "key-p20h1-B"
		seedPendingRow(t, db, key)
		var wg sync.WaitGroup
		bOut := make(chan p20Out, 1)
		holder, rd := p20Stage(t, db, fx, gate, key, bOut, &wg)
		// B proceeds first and must be the SOLE waiter: it is the
		// only request in flight, so the quorum proves B reached
		// the completion UPDATE before A starts.
		gate.shutdown()
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
			rd.failf("round %d: completion gate never formed", round)
		}
		aOut := make(chan p20Out, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := invoke(fx.svc, p20Miniapp, p20Tool, `{}`, key, credAgentA, contract.TrustConfirm)
			aOut <- p20Out{res, err}
		}()
		// A queues behind B; the release then grants in queue
		// order (completion first).
		if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 2); !ok {
			rd.failf("round %d: settle gate never formed behind completion", round)
		}
		rd.once.Do(func() { rd.release() })
		b := rd.recv(bOut, "completion return (must win)")
		a := rd.recv(aOut, "settle return (must lose and replay success)")
		wg.Wait()
		if b.err != nil || b.res.Status != contract.RunSucceeded {
			t.Fatalf("round %d: completion = %+v, %v; want success", round, b.res, b.err)
		}
		var got map[string]any
		if err := json.Unmarshal(b.res.Result, &got); err != nil || got["p20"] != "settled" {
			t.Fatalf("round %d: completion result = %s, want the canned handler result", round, b.res.Result)
		}
		var replayed map[string]any
		if err := json.Unmarshal(a.res.Result, &replayed); err != nil || replayed["p20"] != "settled" {
			t.Fatalf("round %d: settle = %+v, %v; want replayed success with the completed result", round, a.res, a.err)
		}
		if a.err != nil || a.res.Status != contract.RunSucceeded {
			t.Fatalf("round %d: settle = %+v, %v; want replayed success", round, a.res, a.err)
		}
		st, res := p20Record(t, db, key)
		if st != contract.RunSucceeded {
			t.Fatalf("round %d: record = %s, want succeeded (completion stands)", round, st)
		}
		var rec map[string]any
		if err := json.Unmarshal(res, &rec); err != nil || rec["p20"] != "settled" {
			t.Fatalf("round %d: record result = %s, want the completed result", round, res)
		}
		if n := p20AuditCount(t, db, "approval-expired", "expired"); n != 0 {
			t.Fatalf("round %d: approval-expired audits = %d, want 0 (loser settles nothing)", round, n)
		}
		if n := p20AuditCount(t, db, "invoke", "succeeded"); n != 1 {
			t.Fatalf("round %d: invoke/succeeded audits = %d, want exactly 1", round, n)
		}
		if n := gate.calls.Load(); n != 1 {
			t.Fatalf("round %d: handler executions = %d, want 1", round, n)
		}
		fx.async.mu.Lock()
		submits := len(fx.async.submits)
		fx.async.mu.Unlock()
		if submits != 0 {
			t.Fatalf("round %d: workflow submits = %d, want 0 (Class A only)", round, submits)
		}
	}
}
