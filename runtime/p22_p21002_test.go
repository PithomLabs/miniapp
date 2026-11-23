// P21-002 fail-closed characterization: the approval gate refuses
// submission on pending rows without a live linkage, and concurrent
// float-during-park converges instead of sticking. These paths are
// what the specified stuck-state trigger actually encounters;
// they pass on both trees by design (the trigger as specified
// cannot produce a pending-plus-workflow record — see the Stage 1
// evidence row for the reachability proof).
//
// RED/GREEN-COMPATIBLE SOURCE: public APIs, SQL fixtures, and
// pre-existing helpers only.
package runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

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
	p22Miniapp  = "test-p22"
	p22Tool     = "probe.p22"
	p22AMiniapp = "test-p22a"
	p22ATool    = "probe.p22a"
)

func p22Policy() *auth.Policy {
	return auth.NewPolicy(
		map[string]auth.Binding{
			credAgentA: {AgentID: "agent-a", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "agent-a", TenantID: tenantA,
			}},
		},
		map[string]map[string]contract.TrustMode{
			tenantA: {
				p22Miniapp:  contract.TrustAutonomous,
				p22AMiniapp: contract.TrustAutonomous,
			},
		},
	)
}

type p22Fixture struct {
	svc     *runtime.Service
	async   *fakeAsync
	effects *effects.Service
}

func newP22Fixture(t *testing.T, db *testkit.DB) p22Fixture {
	t.Helper()
	reg := registry.New()
	registerProbeWith(t, reg, p22Miniapp, p22Tool, true)
	testkit.RegisterClassATool(t, reg, p22AMiniapp, p22ATool, "", nil,
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		})
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	policy := p22Policy()
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
	return p22Fixture{svc: svc, async: async, effects: effSvc}
}

// p22SeedRow inserts an invocation in the given state for the
// Class D probe tool. Trust is confirm (gated tool); the effect
// token is fixed for Perform intents.
func p22SeedRow(t *testing.T, db *testkit.DB, key, status, runID string) string {
	t.Helper()
	return p22SeedRowTool(t, db, key, p22Miniapp, p22Tool, status, runID)
}

func p22SeedRowTool(t *testing.T, db *testkit.DB, key, miniapp, tool, status, runID string) string {
	t.Helper()
	id := uuid.NewString()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token, run_id)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', $6, 'confirm', 'tok-p22', $7)`,
			id, tenantA, miniapp, tool, key, status, runID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func p22Truncate(t *testing.T, db *testkit.DB) {
	t.Helper()
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
}

func p22RunID(t *testing.T, db *testkit.DB, key string) string {
	t.Helper()
	run, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (string, error) {
		var r string
		err := tx.QueryRow(context.Background(),
			`SELECT run_id FROM app.invocations WHERE idempotency_key = $1`, key).Scan(&r)
		return r, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return run
}

// TestP22PendingSubmitRequiresLinkage pins the fail-closed gate:
// a floated same-key retry on a pending record without a live
// approval is refused with no run adopted and no status change.
// This is the X-01-crash-shape leg of the specified trigger.
func TestP22PendingSubmitRequiresLinkage(t *testing.T) {
	db := newDB(t)
	p22Truncate(t, db)
	fx := newP22Fixture(t, db)
	key := "key-p22-linkage"
	p22SeedRow(t, db, key, string(contract.RunPending), "")
	_, err := invoke(fx.svc, p22Miniapp, p22Tool, `{}`, key, credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "approval linkage required") {
		t.Fatalf("float retry err = %v; want approval-linkage refusal", err)
	}
	if r := p22RunID(t, db, key); r != "" {
		t.Fatalf("run_id = %q after refused submit; want empty (nothing adopted)", r)
	}
	if st := recordStatus(t, db, key); st != contract.RunPending {
		t.Fatalf("record = %s; want pending (refused submit changes nothing)", st)
	}
	fx.async.mu.Lock()
	submits := len(fx.async.submits)
	fx.async.mu.Unlock()
	if submits != 0 {
		t.Fatalf("workflow submits = %d, want 0", submits)
	}
}

// TestP22ConcurrentFloatDuringParkConverges stages the
// millisecond-race leg deterministically: a park blocked behind
// the row gate while a floated retry reads the still-pending row.
// The float is refused (no linkage yet); after release the park
// commits normally with exactly one submit and no stuck state.
func TestP22ConcurrentFloatDuringParkConverges(t *testing.T) {
	db := newDB(t)
	p22Truncate(t, db)
	fx := newP22Fixture(t, db)
	key := "key-p22-race"
	invID := p22SeedRow(t, db, key, string(contract.RunPending), "")
	holder, release := rowGateHolder(t, db, tenantA, invID)
	var wg sync.WaitGroup
	type res struct {
		out runtime.InvokeResult
		err error
	}
	parkOut := make(chan res, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		out, err := invoke(fx.svc, p22Miniapp, p22Tool, `{}`, key, credAgentA, contract.TrustConfirm)
		parkOut <- res{out, err}
	}()
	// The park must be blocked inside its commit-tx (at the
	// invocation row write) before the float reads.
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
		release()
		wg.Wait()
		t.Fatal("park gate never formed")
	}
	_, err := invoke(fx.svc, p22Miniapp, p22Tool, `{}`, key, credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "approval linkage required") {
		release()
		wg.Wait()
		t.Fatalf("float-during-park err = %v; want approval-linkage refusal", err)
	}
	release()
	pr := <-parkOut
	wg.Wait()
	if pr.err != nil || pr.out.Status != contract.RunWaitingApproval {
		t.Fatalf("park = %+v, %v; want waiting_approval", pr.out, pr.err)
	}
	if r := p22RunID(t, db, key); r == "" {
		t.Fatal("park-time submit missing run (want waiting+run normal shape)")
	}
	if st := recordStatus(t, db, key); st != contract.RunWaitingApproval {
		t.Fatalf("record = %s, want waiting_approval (converged, never stuck)", st)
	}
	fx.async.mu.Lock()
	submits := len(fx.async.submits)
	fx.async.mu.Unlock()
	if submits != 1 {
		t.Fatalf("workflow submits = %d, want exactly 1", submits)
	}
}

// TestP22PendingWithoutRunStaysIneligible characterizes the
// preserved shape on both Class D and Class A tools: a pending
// record with no run refuses Perform and sends nothing.
func TestP22PendingWithoutRunStaysIneligible(t *testing.T) {
	db := newDB(t)
	p22Truncate(t, db)
	mockcap.EnsureLog(t, db.Pool)
	fx := newP22Fixture(t, db)
	for _, tc := range []struct {
		key     string
		miniapp string
		tool    string
		op      string
	}{
		{"key-p22-norun-d", p22Miniapp, p22Tool, "p22nr1"},
		{"key-p22-norun-a", p22AMiniapp, p22ATool, "p22nr2"},
	} {
		invID := p22SeedRowTool(t, db, tc.key, tc.miniapp, tc.tool, string(contract.RunPending), "")
		if _, err := fx.effects.Perform(context.Background(), tenantA, effects.Intent{
			BusinessOpID: contract.ScopeID(tenantA, tc.op), AttemptID: "a1",
			InvocationID: invID, EffectToken: "tok-p22",
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: json.RawMessage(`{"mock":{"mode":"accept","opId":"` + tc.op + `"}}`)},
		}); !errors.Is(err, effects.ErrInvocationCancelled) {
			t.Fatalf("%s: Perform err = %v; want ErrInvocationCancelled (both trees)", tc.key, err)
		}
		if n := mockcap.SentCount(t, db.Pool, tc.op); n != 0 {
			t.Fatalf("%s: provider sends = %d, want 0", tc.key, n)
		}
	}
}

// TestP22TerminalStaysIneligible characterizes preserved refusal
// for failed and quarantined records on both trees.
func TestP22TerminalStaysIneligible(t *testing.T) {
	db := newDB(t)
	p22Truncate(t, db)
	mockcap.EnsureLog(t, db.Pool)
	fx := newP22Fixture(t, db)
	for _, tc := range []struct {
		key    string
		status contract.RunStatus
		op     string
		run    string
	}{
		{"key-p22-term-f", contract.RunFailed, "p22t1", "run-p22tf"},
		{"key-p22-term-q", contract.RunQuarantined, "p22t2", "run-p22tq"},
	} {
		invID := p22SeedRow(t, db, tc.key, string(tc.status), tc.run)
		if _, err := fx.effects.Perform(context.Background(), tenantA, effects.Intent{
			BusinessOpID: contract.ScopeID(tenantA, tc.op), AttemptID: "a1",
			InvocationID: invID, EffectToken: "tok-p22",
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: json.RawMessage(`{"mock":{"mode":"accept","opId":"` + tc.op + `"}}`)},
		}); !errors.Is(err, effects.ErrInvocationCancelled) {
			t.Fatalf("%s: Perform err = %v; want ErrInvocationCancelled (both trees)", tc.key, err)
		}
		if _, err := fx.svc.Cancel(context.Background(), credAgentA, tc.run, "x"); err == nil ||
			!strings.Contains(err.Error(), "run-not-live") {
			t.Fatalf("%s: Cancel err = %v; want run-not-live (both trees)", tc.key, err)
		}
	}
}
