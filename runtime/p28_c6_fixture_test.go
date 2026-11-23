package runtime_test

// Prompt28 workstream C — C6 Class A recovery fixture. A real
// Class A handler invokes Perform and meets ErrAttemptInProgress
// under deterministic contention (a gate-held holder on the same
// BusinessOpID). Proves: reachability through the real execution
// path, complete rollback (no partial business writes), no false
// terminal/success marking, no lock held while recovering,
// correct state-specific retry guidance, stable BusinessOpID,
// framework epochs, lease-bounded recovery, and safe adoption.

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
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
	p28C6Miniapp = "test-c6"
	p28C6Tool    = "probe.c6work"
)

type p28C6Fixture struct {
	svc      *runtime.Service
	effSvc   *effects.Service
	policy   *auth.Policy
	reg      *registry.Registry
	approver *approvals.Service
	aud      *audit.Service
	mu       sync.Mutex
	tried    []string // BusinessOpIDs the handler attempted, in order
}

func p28C6Handler(fx *p28C6Fixture, effSvc *effects.Service) registry.ToolFunc {
	return func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		var in struct {
			SubjectID string `json:"subject_id"`
		}
		if err := json.Unmarshal(call.Input, &in); err != nil {
			return nil, err
		}
		opID := call.TenantID + ":c6:" + in.SubjectID
		fx.mu.Lock()
		fx.tried = append(fx.tried, opID)
		fx.mu.Unlock()
		if _, err := call.Tx.Exec(ctx, `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
			VALUES ($1, $2, 'c6', 'c6@example.com')`,
			"c6-"+call.InvocationID, call.TenantID); err != nil {
			return nil, err
		}
		_, err := effSvc.Perform(ctx, call.TenantID, effects.Intent{
			BusinessOpID: opID, InvocationID: call.InvocationID,
			EffectToken: call.EffectToken,
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: json.RawMessage(`{"mock":{"mode":"accept","opId":"c6s1"}}`)},
		})
		if err != nil {
			return nil, err
		}
		return json.RawMessage(`{"c6":"ok"}`), nil
	}
}

func p28C6Setup(t *testing.T, db *testkit.DB) *p28C6Fixture {
	t.Helper()
	fx := &p28C6Fixture{}
	reg := registry.New()
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	aud := audit.New(db.Pool, nil)
	approver := approvals.New(db.Pool)
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			credAgentA: {AgentID: "agent-a", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "agent-a", TenantID: tenantA,
			}},
		},
		map[string]map[string]contract.TrustMode{
			tenantA: {p28C6Miniapp: contract.TrustAutonomous},
		},
	)
	effSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: reg, Grants: policy},
		Registry:  reg, Grants: policy, Approver: approver, Auditor: aud,
	})
	testkit.RegisterClassATool(t, reg, p28C6Miniapp, p28C6Tool, "", nil, p28C6Handler(fx, effSvc))
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: db.Pool,
		Sync: classa.New(db.Pool, reg), Async: async,
		Approver: approver, Auditor: aud, Effects: effSvc,
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.svc, fx.effSvc, fx.policy = svc, effSvc, policy
	fx.reg, fx.approver, fx.aud = reg, approver, aud
	return fx
}

// p28SeedRunning seeds a running autonomous invocation for the
// C6 tool (autonomous matches the manifest's TrustModes).
func p28SeedRunning(t *testing.T, db *testkit.DB, key string) string {
	t.Helper()
	invID := ""
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token, run_id)
			VALUES (gen_random_uuid(), $1, $2, '1.0.0', $3, $4, '{}', 'running', 'autonomous', 'tok-c6', 'run-c6-holder')
			RETURNING id::text`, tenantA, p28C6Miniapp, p28C6Tool, key).Scan(&invID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return invID
}

func p28ContactCount(t *testing.T, db *testkit.DB, id string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(), `SELECT count(*) FROM app.crm_contacts
			WHERE id = $1 AND tenant_id = $2`, id, tenantA).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p28AppLocksHeld(t *testing.T, db *testkit.DB) int {
	t.Helper()
	var n int
	err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_locks l
		JOIN pg_stat_activity a ON a.pid = l.pid
		WHERE a.datname = current_database() AND a.state = 'active' AND l.granted
		AND l.relation IN ('app.effect_records'::regclass, 'app.invocations'::regclass)
		AND a.pid <> pg_backend_pid()`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestP28C6ClassARecovery(t *testing.T) {
	db := newDB(t)
	fx := p28C6Setup(t, db)
	ctx := context.Background()
	op := tenantA + ":c6:s1"

	// Holder: a second service sharing pool/registry/policy, with
	// a gate channel, holds the same BusinessOpID ATTEMPTING.
	holdInv := p28SeedRunning(t, db, "key-c6-holder")
	holdTok := effectToken(t, db, tenantA, holdInv)
	gate := &p25GateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	holdSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": gate},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: fx.reg, Grants: fx.policy},
		Registry:  fx.reg, Grants: fx.policy, Approver: fx.approver, Auditor: fx.aud,
	})
	holderDone := make(chan error, 1)
	go func() {
		_, err := holdSvc.Perform(ctx, tenantA, effects.Intent{
			BusinessOpID: op, InvocationID: holdInv, EffectToken: holdTok,
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: json.RawMessage(`{"mock":{"mode":"accept","opId":"c6s1"}}`)},
		})
		holderDone <- err
	}()
	select {
	case <-gate.entered:
	case err := <-holderDone:
		t.Fatalf("holder failed before Send: %v", err)
	case <-time.After(60 * time.Second):
		t.Fatal("holder never reached Send")
	}
	var holderState, holderAttempt string
	_, err := store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT state, attempt_id FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, tenantA).Scan(&holderState, &holderAttempt)
	})
	if err != nil {
		t.Fatal(err)
	}
	if holderState != "ATTEMPTING" {
		t.Fatalf("holder row = %s, want ATTEMPTING", holderState)
	}

	// Contention through the real Class A path: the handler's
	// Perform meets the live attempt and the whole attempt rolls
	// back. Prompt return proves no wait-held recovery.
	start := time.Now()
	_, err = invoke(fx.svc, p28C6Miniapp, p28C6Tool, `{"subject_id":"s1"}`,
		"key-c6-first", credAgentA, contract.TrustAutonomous)
	elapsed := time.Since(start)
	rerr, ok := err.(*runtime.Error)
	if !ok || rerr.Code != "effect-contention" {
		t.Fatalf("invoke err = %v, want terminal effect-contention", err)
	}
	if elapsed >= 30*time.Second {
		t.Errorf("contention invoke took %v; recovery must not wait", elapsed)
	}
	firstID := ""
	_, _ = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT id::text FROM app.invocations
			WHERE idempotency_key = $1 AND tenant_id = $2`, "key-c6-first", tenantA).Scan(&firstID)
	})
	// No partial business writes survive.
	if n := p28ContactCount(t, db, "c6-"+firstID); n != 0 {
		t.Errorf("contact rows = %d, want 0 (handler tx fully rolled back)", n)
	}
	// Not marked successful or terminally failed.
	var st, result string
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT status, COALESCE(result::text, '') FROM app.invocations
			WHERE id = $1::uuid`, firstID).Scan(&st, &result)
	})
	if err != nil {
		t.Fatal(err)
	}
	if st != string(contract.RunRunning) || result != "" {
		t.Errorf("invocation = (%s, %q), want (running, no result)", st, result)
	}
	// No lock held after return: no DB row locks, grant RLock free.
	if n := p28AppLocksHeld(t, db); n != 0 {
		t.Errorf("app row locks held = %d, want 0", n)
	}
	fx.policy.SetGrants(map[string]map[string]contract.TrustMode{
		tenantA: {p28C6Miniapp: contract.TrustAutonomous},
	})

	// Release the holder; it confirms exactly once.
	close(gate.release)
	select {
	case err := <-holderDone:
		if err != nil {
			t.Fatalf("holder: %v", err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("holder never completed after release")
	}

	// Same-key retry guidance: running record answers
	// interrupted/new-key (never a promise of convergence).
	_, err = invoke(fx.svc, p28C6Miniapp, p28C6Tool, `{"subject_id":"s1"}`,
		"key-c6-first", credAgentA, contract.TrustAutonomous)
	rerr2, ok := err.(*runtime.Error)
	if !ok || rerr2.Code != "interrupted" {
		t.Fatalf("same-key retry err = %v, want terminal interrupted", err)
	}
	if !strings.Contains(rerr2.Msg, "new idempotency key") {
		t.Errorf("interrupted msg = %q, want new-key guidance", rerr2.Msg)
	}

	// New-key retry, same business operation: adopts the
	// winner's settled outcome without resending.
	out2, err := invoke(fx.svc, p28C6Miniapp, p28C6Tool, `{"subject_id":"s1"}`,
		"key-c6-second", credAgentA, contract.TrustAutonomous)
	if err != nil || out2.Status != contract.RunSucceeded {
		t.Fatalf("new-key retry = %+v %v, want success", out2, err)
	}
	if n := p28ContactCount(t, db, "c6-"+out2.InvocationID); n != 1 {
		t.Errorf("contact rows = %d, want 1 (success commits business writes)", n)
	}
	fx.mu.Lock()
	tried := append([]string(nil), fx.tried...)
	fx.mu.Unlock()
	if len(tried) != 2 || tried[0] != op || tried[1] != op {
		t.Errorf("attempted op IDs = %v, want [%s %s] (stable BusinessOpID)", tried, op, op)
	}
	var finalAttempt string
	_, err = store.WithTenant(ctx, db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT attempt_id FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, tenantA).Scan(&finalAttempt)
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalAttempt != holderAttempt {
		t.Errorf("final epoch = %s, want adopted holder epoch %s", finalAttempt, holderAttempt)
	}
	if _, err := uuid.Parse(finalAttempt); err != nil {
		t.Errorf("epoch %q is not a framework UUID: %v", finalAttempt, err)
	}
	if gate.sends != 1 {
		t.Errorf("provider sends = %d, want exactly 1 (adopt resends nothing)", gate.sends)
	}
}
