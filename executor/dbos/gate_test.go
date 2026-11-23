// Executor-side approval gate tests (§5.6 defense in depth).
package dbosexec_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dbos-inc/dbos-transact-golang/dbos"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/registry"
)

// TestSubmitRejectsGatedWithoutLinkage proves the executor refuses
// a gated tool arriving without approval linkage — even a future
// caller bypassing the service choke cannot run gated work
// unapproved. A linked submit proceeds to Start.
func TestSubmitRejectsGatedWithoutLinkage(t *testing.T) {
	pool := newPool(t)
	reg := registry.New()
	mf := probeManifest()
	mf.Tools[0].Approval = contract.ApprovalRequirement{Required: true}
	mf.Tools = mf.Tools[:1]
	for name := range mf.Deployment.Tools {
		if name != toolRecord {
			delete(mf.Deployment.Tools, name)
		}
	}
	var starts atomic.Int32
	impl := &registry.Implementation{Build: "test", Workflows: map[string]registry.WorkflowDef{
		toolRecord: {Name: toolRecord, Version: "1",
			Register: func(env registry.WorkflowEnv) error { return nil },
			Start: func(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
				starts.Add(1)
				return nil
			}},
	}}
	registerProbeFamily(t, reg, probeID, probeVer, mf, impl)
	h := finishHarness(t, pool, reg)
	ctx := context.Background()
	base := executor.Request{
		InvocationID: "inv-g", TenantID: "t-g", MiniappID: probeID,
		Version: probeVer, Tool: toolRecord, Input: json.RawMessage(`{}`),
		IdempotencyKey: "k-g", TrustMode: contract.TrustConfirm,
		RunID: "run-gate-1",
	}
	if _, err := h.ex.Submit(ctx, base); err == nil || !strings.Contains(err.Error(), "requires approval linkage") {
		t.Fatalf("unlinked submit err = %v, want approval-linkage refusal", err)
	}
	if n := starts.Load(); n != 0 {
		t.Fatalf("Start called %d times for refused submit, want 0", n)
	}
	base.ApprovalID = "appr-1"
	base.RunID = "run-gate-2"
	if _, err := h.ex.Submit(ctx, base); err != nil {
		t.Fatalf("linked submit: %v", err)
	}
	if n := starts.Load(); n != 1 {
		t.Fatalf("Start called %d times, want 1", n)
	}
}
