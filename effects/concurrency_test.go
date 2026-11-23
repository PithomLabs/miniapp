package effects_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

const wsTenant = "t-ws"

type allowCancellation struct{}

func (allowCancellation) CheckCancelled(ctx context.Context, tenantID, invocationID string) error {
	return nil
}

type allowPolicy struct{}

func (allowPolicy) CheckPolicy(ctx context.Context, tenantID string, intent effects.Intent) error {
	return nil
}

func setupMock(t *testing.T) (*effects.Service, *testkit.DB) {
	t.Helper()
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	return effects.New(effects.Config{Pool: db.Pool, Channels: map[string]effects.Channel{"mock": mockcap.New(db.Pool)}, Canceller: allowCancellation{}, Policy: allowPolicy{}}), db
}

func mockPayload(mode, opID string) json.RawMessage {
	return json.RawMessage(`{"mock":{"mode":"` + mode + `","opId":"` + opID + `"}}`)
}

func mockIntent(opID, mode string) effects.Intent {
	return effects.Intent{
		BusinessOpID: wsTenant + ":" + opID, AttemptID: "a-" + opID, InvocationID: "inv-" + opID,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: mockPayload(mode, opID)},
	}
}

func TestOpIDScopeEnforced(t *testing.T) {
	svc, _ := setupMock(t)
	ctx := context.Background()
	if _, err := svc.Perform(ctx, wsTenant, effects.Intent{BusinessOpID: "unscoped", InvocationID: "inv-scope"}); err == nil {
		t.Fatal("unscoped business operation ID accepted")
	}
	if _, err := svc.Perform(ctx, wsTenant, effects.Intent{BusinessOpID: "other:op", InvocationID: "inv-scope"}); err == nil {
		t.Fatal("foreign-tenant business operation ID accepted")
	}
	if _, err := svc.Perform(ctx, wsTenant, effects.Intent{BusinessOpID: wsTenant + ":op"}); err == nil {
		t.Fatal("missing invocation ID accepted")
	}
}

// TestConcurrentDuplicateClaims hammers one operation from eight
// workers: exactly one sends, the rest answer contention, and the
// recorded outcome is CONFIRMED — never clobbered, never duplicated.
// The winner is held inside its send until every loser has answered,
// so the verdict never depends on scheduling skew.
func TestConcurrentDuplicateClaims(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	gate := &gateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	svc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": gate},
		Canceller: allowCancellation{}, Policy: allowPolicy{}})
	ctx := context.Background()
	intent := effects.Intent{BusinessOpID: wsTenant + ":op-race", AttemptID: "a1",
		InvocationID: "inv-race",
		Operation:    effects.Operation{CapabilityID: "mock", Operation: "send"}}
	const workers = 8
	type outcome struct {
		rec contract.EffectRecord
		err error
	}
	out := make(chan outcome, workers)
	for i := 0; i < workers; i++ {
		go func() {
			rec, err := svc.Perform(ctx, wsTenant, intent)
			out <- outcome{rec, err}
		}()
	}
	deadline := time.After(60 * time.Second)
	contention := 0
	for contention < workers-1 {
		select {
		case o := <-out:
			if !errors.Is(o.err, effects.ErrAttemptInProgress) {
				t.Fatalf("loser outcome = %+v err=%v, want contention", o.rec, o.err)
			}
			contention++
		case <-deadline:
			t.Fatalf("only %d losers answered contention", contention)
		}
	}
	close(gate.release)
	select {
	case o := <-out:
		if o.err != nil || o.rec.State != contract.EffectConfirmed {
			t.Fatalf("winner = %+v err=%v, want CONFIRMED", o.rec, o.err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("winner never completed after release")
	}
	if gate.sends != 1 {
		t.Fatalf("provider sends = %d, want exactly 1", gate.sends)
	}
}

// gateChannel blocks inside Send until released, so tests can hold
// an attempt open deterministically.
type gateChannel struct {
	entered chan struct{}
	release chan struct{}
	sends   int
}

func (c *gateChannel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	c.sends++
	close(c.entered)
	select {
	case <-c.release:
		return effects.Receipt{ProviderRequestID: "r1"}, nil
	case <-ctx.Done():
		return effects.Receipt{}, ctx.Err()
	}
}

func TestLiveRaceContention(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	gate := &gateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	svc := effects.New(effects.Config{Pool: db.Pool, Channels: map[string]effects.Channel{"mock": gate}, Canceller: allowCancellation{}, Policy: allowPolicy{}})
	ctx := context.Background()
	intent := effects.Intent{BusinessOpID: wsTenant + ":op-live", AttemptID: "a1", InvocationID: "inv-live",
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send"}}

	done := make(chan error, 1)
	go func() {
		_, err := svc.Perform(ctx, wsTenant, intent)
		done <- err
	}()
	select {
	case <-gate.entered:
	case err := <-done:
		t.Fatalf("perform failed before send: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("worker never reached Send")
	}
	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, effects.ErrAttemptInProgress) {
		t.Fatalf("duplicate during live attempt err = %v, want ErrAttemptInProgress", err)
	}
	close(gate.release)
	if err := <-done; err != nil {
		t.Fatalf("winner: %v", err)
	}
	if gate.sends != 1 {
		t.Fatalf("sends = %d, want 1", gate.sends)
	}
	rec, err := svc.Perform(ctx, wsTenant, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("repeat = %+v, %v; want CONFIRMED", rec, err)
	}
}

func TestStalePendingTakeoverSendsOnce(t *testing.T) {
	svc, db := setupMock(t)
	ctx := context.Background()
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, capability_id, operation, payload, state, updated_at)
			VALUES ($1, $2, 'a0', 'mock', 'send', '{"mock":{"mode":"accept","opId":"op-takeover"}}', 'PENDING', now() - interval '1 hour')`,
			wsTenant+":op-takeover", wsTenant)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := svc.Perform(ctx, wsTenant, mockIntent("op-takeover", mockcap.ModeAccept))
	if err != nil {
		t.Fatalf("takeover: %v", err)
	}
	if rec.State != contract.EffectConfirmed {
		t.Fatalf("takeover state = %s, want CONFIRMED", rec.State)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-takeover"); n != 1 {
		t.Fatalf("sends = %d, want 1", n)
	}
}

func TestOperationReuseConflict(t *testing.T) {
	svc, _ := setupMock(t)
	ctx := context.Background()
	first := mockIntent("op-reuse", mockcap.ModeAccept)
	if _, err := svc.Perform(ctx, wsTenant, first); err != nil {
		t.Fatalf("first: %v", err)
	}
	second := mockIntent("op-reuse", mockcap.ModeAccept)
	second.Operation.Operation = "different-send"
	if _, err := svc.Perform(ctx, wsTenant, second); err == nil {
		t.Fatal("same op ID with different operation accepted")
	}
	third := mockIntent("op-reuse", mockcap.ModeReject)
	if _, err := svc.Perform(ctx, wsTenant, third); err == nil {
		t.Fatal("same op ID with different payload accepted")
	}
}

func TestCrossTenantOpIDs(t *testing.T) {
	svc, _ := setupMock(t)
	ctx := context.Background()
	mkIntent := func(tenant, local string) effects.Intent {
		return effects.Intent{BusinessOpID: tenant + ":" + local, AttemptID: "a1", InvocationID: "inv-" + local,
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: mockPayload(mockcap.ModeAccept, tenant+"-"+local)}}
	}
	ra, err := svc.Perform(ctx, "t-a", mkIntent("t-a", "opx"))
	if err != nil {
		t.Fatalf("t-a: %v", err)
	}
	rb, err := svc.Perform(ctx, "t-b", mkIntent("t-b", "opx"))
	if err != nil {
		t.Fatalf("t-b: %v", err)
	}
	if ra.State != contract.EffectConfirmed || rb.State != contract.EffectConfirmed {
		t.Fatalf("states = %s, %s; want independent CONFIRMED", ra.State, rb.State)
	}
}

func TestSettleAndListUnknown(t *testing.T) {
	svc, db := setupMock(t)
	ctx := context.Background()
	rec, err := svc.Perform(ctx, wsTenant, mockIntent("op-settle", mockcap.ModeTimeout))
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != contract.EffectUnknown {
		t.Fatalf("state = %s, want UNKNOWN", rec.State)
	}
	found, err := svc.ListUnknown(ctx, wsTenant, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].BusinessOpID != wsTenant+":op-settle" {
		t.Fatalf("quarantine = %+v, want the settled op", found)
	}
	settled, err := svc.Settle(ctx, wsTenant, wsTenant+":op-settle", contract.EffectFailed, "operator: provider confirmed rejection")
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	if settled.State != contract.EffectFailed {
		t.Fatalf("settled state = %s, want FAILED", settled.State)
	}
	found, err = svc.ListUnknown(ctx, wsTenant, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("quarantine still lists settled op: %+v", found)
	}
	if _, err := svc.Settle(ctx, wsTenant, wsTenant+":op-settle", contract.EffectFailed, "again"); err == nil {
		t.Fatal("double settle accepted")
	}
	if _, err := svc.Settle(ctx, wsTenant, wsTenant+":op-settle", contract.EffectAttempting, "bad"); err == nil {
		t.Fatal("settle to non-terminal accepted")
	}
	_ = db
}

// countChannel wraps a Channel and counts Send invocations.
type countChannel struct {
	inner effects.Channel
	count *int
}

func (c *countChannel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	*c.count++
	return c.inner.Send(ctx, op)
}

const workerEnvKey = "MINIAPP_EFFECTS_WORKER"

func TestEffectsCrashWorker(t *testing.T) {
	if !testkit.WorkerMode(workerEnvKey) {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	svc := effects.New(effects.Config{Pool: db.Pool, Channels: map[string]effects.Channel{"mock": mockcap.New(db.Pool)}, Canceller: allowCancellation{}, Policy: allowPolicy{}})
	op := os.Getenv("MINIAPP_EFFECTS_OP")
	_, _ = svc.Perform(context.Background(), wsTenant, effects.Intent{
		BusinessOpID: wsTenant + ":" + op, AttemptID: "a-worker", InvocationID: "inv-kill",
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: mockPayload(mockcap.ModeDelayedAck, op)},
	})
}

// TestKillMidSendAdoptsUnknown kills a worker after the provider
// accepted but before the outcome recorded: the immediate retry
// answers contention (the send may be in flight), and once the row
// ages out the attempt quarantines to UNKNOWN with the provider
// holding exactly one send and no blind retry after.
func TestKillMidSendAdoptsUnknown(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	op := "op-kill"
	cmd := testkit.SpawnWorker(t, "TestEffectsCrashWorker", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": op,
	})
	deadline := time.Now().Add(60 * time.Second)
	for {
		var n int
		err := db.Pool.QueryRow(context.Background(),
			`SELECT count(*) FROM mock_provider_log WHERE op_id = $1`, op).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			testkit.Kill9(t, cmd)
			t.Fatal("worker never reached provider acceptance")
		}
		time.Sleep(100 * time.Millisecond)
	}
	testkit.Kill9(t, cmd)

	var sends int
	svc := effects.New(effects.Config{Pool: db.Pool, Channels: map[string]effects.Channel{
		"mock": &countChannel{inner: mockcap.New(db.Pool), count: &sends}}, Canceller: allowCancellation{}, Policy: allowPolicy{}})
	ctx := context.Background()
	intent := effects.Intent{BusinessOpID: wsTenant + ":" + op, AttemptID: "a-retry", InvocationID: "inv-kill",
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: mockPayload(mockcap.ModeDelayedAck, op)}}
	if _, err := svc.Perform(ctx, wsTenant, intent); !errors.Is(err, effects.ErrAttemptInProgress) {
		t.Fatalf("immediate retry err = %v, want ErrAttemptInProgress", err)
	}
	_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE app.effect_records SET updated_at = now() - interval '1 hour'
			WHERE business_op_id = $1`, wsTenant+":"+op)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := svc.Perform(ctx, wsTenant, intent)
	if err != nil {
		t.Fatalf("aged retry: %v", err)
	}
	if rec.State != contract.EffectUnknown {
		t.Fatalf("aged retry state = %s, want UNKNOWN", rec.State)
	}
	if sends != 0 {
		t.Fatalf("parent sent %d times during recovery, want 0 (no blind retry)", sends)
	}
	if n := mockcap.SentCount(t, db.Pool, op); n != 1 {
		t.Fatalf("provider sends = %d, want exactly 1", n)
	}
}
