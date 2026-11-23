// MEDIUM-2 runtime tests: framework-enforced single Class A
// execution (concurrent granted polls, dispatch-vs-replay,
// retryable conflict, pre/post-commit crash recovery).
//
// RED/GREEN-COMPATIBLE SOURCE: uses only public service APIs, SQL
// fixtures, pg_stat_activity observation, file rendezvous, and the
// SpawnWorker/Kill9 harness — all stable pre-fix. The red run
// overlays exactly this file onto the pre-fix working copy.
// New-API unit tests (nil hooks, recheck, savepoint) live in
// executor/classa/p20_medium2_test.go and are green-only.
package runtime_test

import (
	"context"
	"encoding/json"
	"os"
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
	p20bMiniapp = "test-p20b"
	p20bTool    = "probe.p20b"
	p20bResult  = `{"p20b":"ok"}`
)

// p20bGate is a channel-blocked Class A handler that writes one
// direct-SQL business row per execution (the double-apply
// detector), signals entry, parks until release, and counts
// executions. The entered buffer is generous so a double-
// executing tree wedges nothing — it fails assertions instead.
type p20bGate struct {
	name    string
	entered chan int32
	release chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func newP20bGate(name string) *p20bGate {
	return &p20bGate{name: name, entered: make(chan int32, 16), release: make(chan struct{})}
}

func (g *p20bGate) handler() registry.ToolFunc {
	return func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		n := g.calls.Add(1)
		if _, err := call.Tx.Exec(ctx, `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
			VALUES ($1, 'tenant-a', $2, 'p20b@example.com')`, uuid.NewString(), g.name); err != nil {
			return nil, err
		}
		g.entered <- n
		<-g.release
		return json.RawMessage(p20bResult), nil
	}
}

func (g *p20bGate) shutdown() {
	g.once.Do(func() { close(g.release) })
}

func p20bPolicy() *auth.Policy {
	return auth.NewPolicy(
		map[string]auth.Binding{
			credAgentA: {AgentID: "agent-a", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "agent-a", TenantID: tenantA,
			}},
		},
		map[string]map[string]contract.TrustMode{
			tenantA: {p20bMiniapp: contract.TrustAutonomous},
		},
	)
}

type p20bFixture struct {
	svc   *runtime.Service
	async *fakeAsync
}

func newP20bFixture(t *testing.T, db *testkit.DB, handler registry.ToolFunc) p20bFixture {
	t.Helper()
	reg := registry.New()
	mf := testkit.ClassAManifest(p20bMiniapp, p20bTool, "", nil)
	mf.Tools[0].Approval = contract.ApprovalRequirement{
		Required: true, ApproverScope: "tenant-ops", Expiry: "1h",
	}
	mf.Tools[0].TrustModes = []contract.TrustMode{contract.TrustConfirm, contract.TrustAutonomous}
	if err := reg.Register(mf, testkit.ClassAImpl("test", p20bTool, handler),
		testkit.Envelope(t, mf, "test")); err != nil {
		t.Fatal(err)
	}
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	policy := p20bPolicy()
	aud := audit.New(db.Pool, nil)
	approver := approvals.New(db.Pool)
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
	return p20bFixture{svc: svc, async: async}
}

func p20bSeedPending(t *testing.T, db *testkit.DB, key string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', 'pending', 'confirm', 'tok-p20b')`,
			uuid.NewString(), tenantA, p20bMiniapp, p20bTool, key)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// p20bBusinessCount counts committed handler business rows.
// Tenant context is mandatory (forced RLS).
func p20bBusinessCount(t *testing.T, db *testkit.DB, name string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM app.crm_contacts WHERE name = $1`, name).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p20bTruncate(t *testing.T, db *testkit.DB) {
	t.Helper()
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
}

// p20bRound owns cleanup (handler release + join) on every path.
type p20bRound struct {
	t    *testing.T
	gate *p20bGate
	wg   *sync.WaitGroup
}

func (r *p20bRound) cleanup() {
	r.gate.shutdown()
	r.wg.Wait()
}

func (r *p20bRound) recv(out <-chan p20Out, what string) p20Out {
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

func (r *p20bRound) recvInt(ch <-chan int32, what string) int32 {
	r.t.Helper()
	select {
	case n := <-ch:
		return n
	case <-time.After(30 * time.Second):
		r.cleanup()
		r.t.Fatalf("timed out waiting for %s", what)
		return 0
	}
}

// recvPrompt bounds the conflict-promptness assertion: a correct
// loser answers in milliseconds while the winner is still
// blocked. Only a wait-loop or missing-conflict implementation
// hits the bound (red fails here).
func (r *p20bRound) recvPrompt(out <-chan p20Out, what string) p20Out {
	r.t.Helper()
	select {
	case o := <-out:
		return o
	case <-time.After(10 * time.Second):
		entries := r.gate.calls.Load()
		r.cleanup()
		r.t.Fatalf("no prompt answer for %s (loser must conflict, not wait or execute; handler entries so far: %d)",
			what, entries)
		return p20Out{}
	}
}

func (r *p20bRound) failf(format string, args ...any) {
	r.t.Helper()
	r.cleanup()
	r.t.Fatalf(format, args...)
}

func p20bApprove(t *testing.T, db *testkit.DB, approvalID string) {
	t.Helper()
	appr := approvals.New(db.Pool)
	hash := runtime.IntentHash(p20bMiniapp, "1.0.0", p20bTool, []byte(`{}`))
	decider := approvals.Decider{
		Caller: contract.Caller{AgentID: "human-op",
			Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "human-op", TenantID: tenantA}},
		Roles: []string{"tenant-ops"},
	}
	if err := appr.Decide(context.Background(), tenantA, approvalID, decider, contract.DecisionApprove, hash); err != nil {
		t.Fatal(err)
	}
}

// TestP20ConcurrentGrantedPollsSingleExecution: two polls observe
// the same granted approval. Exactly one executes the handler and
// commits; the loser conflicts without executing; the loser's
// explicit retry converges on the recorded success. Business rows
// prove once-only committed work.
func TestP20ConcurrentGrantedPollsSingleExecution(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		p20bTruncate(t, db)
		gate := newP20bGate("p20b-t1")
		fx := newP20bFixture(t, db, gate.handler())
		key := "key-p20b-t1"
		p20bSeedPending(t, db, key)
		parked, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustConfirm)
		if err != nil || parked.Status != contract.RunWaitingApproval || parked.ApprovalID == "" {
			t.Fatalf("round %d: park = %+v, %v", round, parked, err)
		}
		p20bApprove(t, db, parked.ApprovalID)
		var wg sync.WaitGroup
		rd := &p20bRound{t: t, gate: gate, wg: &wg}
		p1Out := make(chan p20Out, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustConfirm)
			p1Out <- p20Out{res, err}
		}()
		rd.recvInt(gate.entered, "winner handler entry (overlap proof)")
		p2Out := make(chan p20Out, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustConfirm)
			p2Out <- p20Out{res, err}
		}()
		p2 := rd.recvPrompt(p2Out, "loser conflict")
		if p2.err == nil || !strings.Contains(p2.err.Error(), "execution-in-progress") {
			rd.failf("round %d: loser = %+v, %v; want execution-in-progress conflict", round, p2.res, p2.err)
		}
		if n := gate.calls.Load(); n != 1 {
			rd.failf("round %d: handler executions = %d, want exactly 1 (loser must not execute)", round, n)
		}
		gate.shutdown()
		p1 := rd.recv(p1Out, "winner return")
		wg.Wait()
		if p1.err != nil || p1.res.Status != contract.RunSucceeded {
			t.Fatalf("round %d: winner = %+v, %v; want success", round, p1.res, p1.err)
		}
		retry, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustConfirm)
		if err != nil || retry.Status != contract.RunSucceeded {
			t.Fatalf("round %d: loser retry = %+v, %v; want replayed success", round, retry, err)
		}
		if n := p20bBusinessCount(t, db, "p20b-t1"); n != 1 {
			t.Fatalf("round %d: business rows = %d, want exactly 1", round, n)
		}
		if n := p20AuditCount(t, db, "invoke", "succeeded"); n != 1 {
			t.Fatalf("round %d: invoke/succeeded audits = %d, want exactly 1", round, n)
		}
		if st := recordStatus(t, db, key); st != contract.RunSucceeded {
			t.Fatalf("round %d: record = %s, want succeeded", round, st)
		}
	}
}

// TestP20DispatchVsReplaySingleExecution: the claim winner
// dispatches while a same-key replay re-dispatches concurrently.
// Exactly one handler execution commits; the loser conflicts.
func TestP20DispatchVsReplaySingleExecution(t *testing.T) {
	db := newDB(t)
	for round := 0; round < 3; round++ {
		p20bTruncate(t, db)
		gate := newP20bGate("p20b-t2")
		fx := newP20bFixture(t, db, gate.handler())
		key := "key-p20b-t2"
		p20bSeedPending(t, db, key)
		var wg sync.WaitGroup
		rd := &p20bRound{t: t, gate: gate, wg: &wg}
		p1Out := make(chan p20Out, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
			p1Out <- p20Out{res, err}
		}()
		rd.recvInt(gate.entered, "winner handler entry (overlap proof)")
		p2Out := make(chan p20Out, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
			p2Out <- p20Out{res, err}
		}()
		p2 := rd.recvPrompt(p2Out, "loser conflict")
		if p2.err == nil || !strings.Contains(p2.err.Error(), "execution-in-progress") {
			rd.failf("round %d: loser = %+v, %v; want execution-in-progress conflict", round, p2.res, p2.err)
		}
		gate.shutdown()
		p1 := rd.recv(p1Out, "winner return")
		wg.Wait()
		if p1.err != nil || p1.res.Status != contract.RunSucceeded {
			t.Fatalf("round %d: winner = %+v, %v; want success", round, p1.res, p1.err)
		}
		if n := gate.calls.Load(); n != 1 {
			t.Fatalf("round %d: handler executions = %d, want exactly 1", round, n)
		}
		retry, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
		if err != nil || retry.Status != contract.RunSucceeded {
			t.Fatalf("round %d: loser retry = %+v, %v; want replayed success", round, retry, err)
		}
		if n := p20bBusinessCount(t, db, "p20b-t2"); n != 1 {
			t.Fatalf("round %d: business rows = %d, want exactly 1", round, n)
		}
		if st := recordStatus(t, db, key); st != contract.RunSucceeded {
			t.Fatalf("round %d: record = %s, want succeeded", round, st)
		}
	}
}

// TestP20ConflictIsRetryable (condition 3): the loser conflicts
// promptly while the winner is still blocked (no automatic retry
// loop, no wait), writes nothing (the row is untouched), names
// the same-key retry, and an explicit retry converges after the
// winner commits.
func TestP20ConflictIsRetryable(t *testing.T) {
	db := newDB(t)
	p20bTruncate(t, db)
	gate := newP20bGate("p20b-c3")
	fx := newP20bFixture(t, db, gate.handler())
	key := "key-p20b-c3"
	p20bSeedPending(t, db, key)
	var wg sync.WaitGroup
	rd := &p20bRound{t: t, gate: gate, wg: &wg}
	p1Out := make(chan p20Out, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
		p1Out <- p20Out{res, err}
	}()
	rd.recvInt(gate.entered, "winner handler entry")
	p2Out := make(chan p20Out, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
		p2Out <- p20Out{res, err}
	}()
	p2 := rd.recvPrompt(p2Out, "loser conflict (must precede winner release)")
	if p2.err == nil || !strings.Contains(p2.err.Error(), "execution-in-progress") {
		rd.failf("loser = %+v, %v; want execution-in-progress", p2.res, p2.err)
	}
	if !strings.Contains(p2.err.Error(), "same idempotency key") {
		rd.failf("loser err = %q; must name the same-key retry", p2.err.Error())
	}
	// Untouched row, observed while the winner is still blocked:
	// the loser wrote nothing and persisted no terminal.
	if st := recordStatus(t, db, key); st != contract.RunPending {
		rd.failf("record = %s during contention, want pending (loser wrote nothing)", st)
	}
	if n := p20AuditCount(t, db, "invoke", "succeeded"); n != 0 {
		rd.failf("invoke/succeeded audits = %d during contention, want 0", n)
	}
	gate.shutdown()
	p1 := rd.recv(p1Out, "winner return")
	wg.Wait()
	if p1.err != nil {
		t.Fatalf("winner = %v", p1.err)
	}
	retry, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
	if err != nil || retry.Status != contract.RunSucceeded {
		t.Fatalf("explicit retry = %+v, %v; want replayed success", retry, err)
	}
	if n := p20bBusinessCount(t, db, "p20b-c3"); n != 1 {
		t.Fatalf("business rows = %d, want exactly 1", n)
	}
}

const p20bWorkerEnv = "MINIAPP_P20B_WORKER"

// TestP20M2CrashWorker is the crash-worker entrypoint: it runs one
// autonomous dispatch in a child process and is killed by the
// parent at a rendezvous point. Skipped in normal runs.
func TestP20M2CrashWorker(t *testing.T) {
	if os.Getenv(p20bWorkerEnv) == "" {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	db := testkit.NewDB(t)
	key := os.Getenv("MINIAPP_P20B_KEY")
	name := os.Getenv("MINIAPP_P20B_NAME")
	mode := os.Getenv("MINIAPP_P20B_MODE")
	marker := os.Getenv("MINIAPP_P20B_MARKER")
	handler := func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		if _, err := call.Tx.Exec(ctx, `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
			VALUES ($1, 'tenant-a', $2, 'p20b@example.com')`, uuid.NewString(), name); err != nil {
			return nil, err
		}
		if mode == "block" {
			if err := os.WriteFile(marker, []byte("entered"), 0644); err != nil {
				return nil, err
			}
			deadline := time.Now().Add(60 * time.Second)
			for {
				if _, err := os.Stat(marker + ".release"); err == nil {
					break
				}
				if time.Now().After(deadline) {
					return nil, context.DeadlineExceeded
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		return json.RawMessage(p20bResult), nil
	}
	fx := newP20bFixture(t, db, handler)
	_, _ = invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
}

// TestP20CrashBeforeCommitRecovery kills a worker parked inside
// the handler transaction, then recovers in-process: the killed
// attempt committed nothing, and exactly one business row exists
// after recovery. Characterization: both trees roll back
// uncommitted work (pins the behavior, not the fix).
func TestP20CrashBeforeCommitRecovery(t *testing.T) {
	db := newDB(t)
	p20bTruncate(t, db)
	key := "key-p20b-t5"
	p20bSeedPending(t, db, key)
	marker := t.TempDir() + "/entered"
	w := testkit.SpawnWorker(t, "TestP20M2CrashWorker", map[string]string{
		p20bWorkerEnv: "1", "MINIAPP_P20B_KEY": key, "MINIAPP_P20B_NAME": "p20b-t5",
		"MINIAPP_P20B_MODE": "block", "MINIAPP_P20B_MARKER": marker,
	})
	testkit.WaitForFile(t, w, marker, 60*time.Second)
	testkit.Kill9(t, w)
	if n := p20bBusinessCount(t, db, "p20b-t5"); n != 0 {
		t.Fatalf("business rows after kill = %d, want 0 (killed attempt committed nothing)", n)
	}
	if st := recordStatus(t, db, key); st != contract.RunPending {
		t.Fatalf("record after kill = %s, want pending", st)
	}
	gate := newP20bGate("p20b-t5")
	gate.shutdown()
	fx := newP20bFixture(t, db, gate.handler())
	out, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
	if err != nil || out.Status != contract.RunSucceeded {
		t.Fatalf("recovery = %+v, %v; want success", out, err)
	}
	if n := p20bBusinessCount(t, db, "p20b-t5"); n != 1 {
		t.Fatalf("business rows after recovery = %d, want exactly 1", n)
	}
}

// TestP20CrashAfterHandlerCommitRecovery kills a worker blocked
// at the completion write — after its business work committed on
// the old two-transaction path. Recovery must not double-apply:
// green rolls the whole attempt back (single transaction) and
// commits exactly once; red has two committed business rows.
func TestP20CrashAfterHandlerCommitRecovery(t *testing.T) {
	db := newDB(t)
	p20bTruncate(t, db)
	key := "key-p20b-t6"
	p20bSeedPending(t, db, key)
	invID := p20InvID(t, db, key)
	holder, release := rowGateHolder(t, db, tenantA, invID)
	w := testkit.SpawnWorker(t, "TestP20M2CrashWorker", map[string]string{
		p20bWorkerEnv: "1", "MINIAPP_P20B_KEY": key, "MINIAPP_P20B_NAME": "p20b-t6",
		"MINIAPP_P20B_MODE": "run",
	})
	// The worker must be blocked at the completion write behind
	// OUR holder: handler work is done, the terminal is not.
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
		release()
		testkit.Kill9(t, w)
		t.Fatal("worker never blocked at completion")
	}
	testkit.Kill9(t, w)
	release()
	gate := newP20bGate("p20b-t6")
	gate.shutdown()
	fx := newP20bFixture(t, db, gate.handler())
	out, err := invoke(fx.svc, p20bMiniapp, p20bTool, `{}`, key, credAgentA, contract.TrustAutonomous)
	if err != nil || out.Status != contract.RunSucceeded {
		t.Fatalf("recovery = %+v, %v; want success", out, err)
	}
	if n := p20bBusinessCount(t, db, "p20b-t6"); n != 1 {
		t.Fatalf("business rows after recovery = %d, want exactly 1 (no double-apply)", n)
	}
	if st := recordStatus(t, db, key); st != contract.RunSucceeded {
		t.Fatalf("record = %s, want succeeded", st)
	}
}
