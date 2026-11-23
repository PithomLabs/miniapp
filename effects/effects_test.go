package effects

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

// scriptChannel is a scripted mock provider. Each Send consumes the
// next scripted outcome (or repeats the last); sends counts every
// provider contact so tests can assert the intended send count.
// The count measures calls; exactly-once delivery is never claimed.
type scriptChannel struct {
	sends int
	steps []func() (Receipt, error)
}

func (c *scriptChannel) Send(ctx context.Context, op Operation) (Receipt, error) {
	c.sends++
	i := c.sends - 1
	if i >= len(c.steps) {
		i = len(c.steps) - 1
	}
	return c.steps[i]()
}

func okStep(id string) func() (Receipt, error) {
	return func() (Receipt, error) { return Receipt{ProviderRequestID: id}, nil }
}

// allowCancellation approves every claim. Unit tests that do not
// exercise cancellation use it; cancellation behavior has its own
// tests with real and denying cancellers.
type allowCancellation struct{}

func (allowCancellation) CheckCancelled(ctx context.Context, tenantID, invocationID string) error {
	return nil
}

type allowPolicy struct{}

func (allowPolicy) CheckPolicy(ctx context.Context, tenantID string, intent Intent) error {
	return nil
}

func setup(t *testing.T, ch Channel) (*Service, string) {
	t.Helper()
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	return New(Config{Pool: db.Pool, Channels: map[string]Channel{"mock": ch}, Canceller: allowCancellation{}, Policy: allowPolicy{}}), "t-effect"
}

func TestPerformHappyPath(t *testing.T) {
	ch := &scriptChannel{steps: []func() (Receipt, error){okStep("req-1")}}
	svc, tenant := setup(t, ch)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	rec, err := svc.Perform(ctx, tenant, Intent{BusinessOpID: "t-effect:op-happy", InvocationID: "inv-test", AttemptID: "a1", Operation: Operation{CapabilityID: "mock", Operation: "send"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != contract.EffectConfirmed || rec.ProviderRequest != "req-1" {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if ch.sends != 1 {
		t.Fatalf("provider contacted %d times, want 1", ch.sends)
	}
	// Repeating the same operation must not re-send.
	rec2, err := svc.Perform(ctx, tenant, Intent{BusinessOpID: "t-effect:op-happy", InvocationID: "inv-test", AttemptID: "a2", Operation: Operation{CapabilityID: "mock", Operation: "send"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec2.State != contract.EffectConfirmed || ch.sends != 1 {
		t.Fatalf("repeat re-sent or changed state: %+v sends=%d", rec2, ch.sends)
	}
}

func TestPerformFailureAndUnknown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	failCh := &scriptChannel{steps: []func() (Receipt, error){
		func() (Receipt, error) { return Receipt{}, PermanentError{errors.New("rejected")} },
	}}
	svc, tenant := setup(t, failCh)
	rec, err := svc.Perform(ctx, tenant, Intent{BusinessOpID: "t-effect:op-fail", InvocationID: "inv-test", AttemptID: "a1", Operation: Operation{CapabilityID: "mock"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != contract.EffectFailed {
		t.Fatalf("state = %s, want FAILED", rec.State)
	}

	timeoutCh := &scriptChannel{steps: []func() (Receipt, error){
		func() (Receipt, error) { return Receipt{}, UnknownError{errors.New("timeout after send")} },
	}}
	svc2, _ := setup(t, timeoutCh)
	rec2, err := svc2.Perform(ctx, tenant, Intent{BusinessOpID: "t-effect:op-unknown", InvocationID: "inv-test", AttemptID: "a1", Operation: Operation{CapabilityID: "mock"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec2.State != contract.EffectUnknown {
		t.Fatalf("state = %s, want UNKNOWN", rec2.State)
	}
	// Unknown outcomes never blind-retry on repeat.
	if _, err := svc2.Perform(ctx, tenant, Intent{BusinessOpID: "t-effect:op-unknown", InvocationID: "inv-test", AttemptID: "a2", Operation: Operation{CapabilityID: "mock"}}); err != nil {
		t.Fatal(err)
	}
	if timeoutCh.sends != 1 {
		t.Fatalf("unknown outcome re-sent: %d sends", timeoutCh.sends)
	}
}

func TestPerformAdoptsStrandedAttempt(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.effect_records")
	svc := New(Config{Pool: db.Pool, Channels: map[string]Channel{"mock": &scriptChannel{steps: []func() (Receipt, error){okStep("x")}}}, Canceller: allowCancellation{}, Policy: allowPolicy{}})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// Simulate a worker that crashed between intent and outcome: an
	// ATTEMPTING row older than the attempt lease, which no live
	// worker could still hold.
	if _, err := store.WithTenant(ctx, db.Pool, "t-effect", func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, capability_id, operation, state, updated_at)
			VALUES ('t-effect:op-crash', 't-effect', 'a1', 'mock', 'send', 'ATTEMPTING', now() - interval '1 hour')`)
		return true, err
	}); err != nil {
		t.Fatal(err)
	}
	rec, err := svc.Perform(ctx, "t-effect", Intent{BusinessOpID: "t-effect:op-crash", InvocationID: "inv-test", AttemptID: "a2", Operation: Operation{CapabilityID: "mock", Operation: "send"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != contract.EffectUnknown {
		t.Fatalf("stranded attempt adopted as %s, want UNKNOWN", rec.State)
	}
}
