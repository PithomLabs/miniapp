package runtime_test

// P24-010 (Prompt 25): the effect-contention retry instruction must
// follow replay semantics for the record at hand. Ledger
// contention inside a first-claim (running) Class A dispatch
// answers interrupted/new-key on same-key retry, so the message
// must not promise same-key convergence there.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/runtime"
)

// p25ContentionSync stubs the sync executor to lose ledger
// contention on every execution.
type p25ContentionSync struct{}

func (p25ContentionSync) Execute(ctx context.Context, req executor.Request) (json.RawMessage, error) {
	return nil, executor.ErrAttemptContention
}

func TestP25ContentionMessageRunningClassA(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	// Rebuild the service with a contention-losing sync executor;
	// every other dependency matches the standard fixture.
	svc, err := runtime.New(runtime.Config{
		Registry: fx.reg, Policy: fx.policy, Pool: db.Pool,
		Sync: p25ContentionSync{}, Async: fx.async,
		Approver: fx.approver, Auditor: fx.aud, Effects: fx.effects,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = invoke(svc, "entity-probe", "probe.capture", `{"subject_id":"p25c"}`,
		"key-p25-contention", credAgentA, contract.TrustAutonomous)
	if err == nil {
		t.Fatal("invoke err = nil, want effect-contention")
	}
	rerr, ok := err.(*runtime.Error)
	if !ok {
		t.Fatalf("err type = %T, want *runtime.Error", err)
	}
	if rerr.Code != "effect-contention" {
		t.Fatalf("code = %q, want effect-contention", rerr.Code)
	}
	if !strings.Contains(rerr.Msg, "new idempotency key") {
		t.Errorf("msg = %q, want new-key instruction for running Class A", rerr.Msg)
	}
	if strings.Contains(rerr.Msg, "retry with the same idempotency key") {
		t.Errorf("msg = %q, must not promise same-key convergence for running Class A", rerr.Msg)
	}
}
