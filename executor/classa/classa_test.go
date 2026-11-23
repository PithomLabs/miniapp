package classa_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
)

const (
	probeID = "ent-probe"
	tool    = "probe.write"
)

func setup(t *testing.T, handler registry.ToolFunc) (*classa.Executor, *testkit.DB) {
	t.Helper()
	db := testkit.NewDB(t)
	reg := registry.New()
	testkit.RegisterClassATool(t, reg, probeID, tool, "subject_id", []string{"mock.send"}, handler)
	return classa.New(db.Pool, reg), db
}

func req(entity, inv string) executor.Request {
	return executor.Request{
		InvocationID: inv, TenantID: "tenant-a",
		Principal: contract.Caller{AgentID: "agent-a",
			Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "agent-a", TenantID: "tenant-a"}},
		MiniappID: probeID, Version: "1.0.0", Tool: tool,
		Input: json.RawMessage(`{"subject_id":"x"}`), EntityKey: entity,
		Claim: stubClaim(),
	}
}

// stubClaim stands in for the runtime's invocation hooks: the
// entity-serialization tests below exercise lock orchestration,
// not invocation lifecycle (covered at the runtime level).
func stubClaim() *executor.ClaimHooks {
	return &executor.ClaimHooks{
		Recheck: func(tx pgx.Tx) (bool, error) { return true, nil },
		Complete: func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
			return true, nil
		},
	}
}

// TestSameEntitySerializes proves concurrent executions for one
// entity never overlap inside the handler: peak concurrency is 1.
func TestSameEntitySerializes(t *testing.T) {
	var inflight, peak atomic.Int32
	handler := func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		cur := inflight.Add(1)
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}
		time.Sleep(300 * time.Millisecond)
		inflight.Add(-1)
		return json.RawMessage(`{"ok":true}`), nil
	}
	ex, _ := setup(t, handler)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, inv := range []string{"i1", "i2"} {
		wg.Add(1)
		go func(inv string) {
			defer wg.Done()
			_, err := ex.Execute(context.Background(), req("tenant-a:sub1", inv))
			errs <- err
		}(inv)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
	}
	if got := peak.Load(); got != 1 {
		t.Fatalf("peak concurrency = %d, want 1 (same entity must serialize)", got)
	}
}

// TestDistinctEntitiesParallel proves different entities do not
// block each other: both handlers must rendezvous inside the lock
// region, which is impossible under serialization (the first would
// time out waiting for a second that can never enter).
func TestDistinctEntitiesParallel(t *testing.T) {
	var entered atomic.Int32
	handler := func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		entered.Add(1)
		deadline := time.Now().Add(5 * time.Second)
		for entered.Load() < 2 {
			if time.Now().After(deadline) {
				return nil, context.DeadlineExceeded
			}
			time.Sleep(10 * time.Millisecond)
		}
		return json.RawMessage(`{"ok":true}`), nil
	}
	ex, _ := setup(t, handler)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i, ent := range []string{"tenant-a:sub1", "tenant-a:sub2"} {
		wg.Add(1)
		go func(i int, ent string) {
			defer wg.Done()
			_, err := ex.Execute(context.Background(), req(ent, string(rune('a'+i))))
			errs <- err
		}(i, ent)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("execute: %v (distinct entities must proceed in parallel)", err)
		}
	}
}
