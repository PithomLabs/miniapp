package classd

import (
	"encoding/json"
	"testing"

	"github.com/PithomLabs/miniapp/registry"
)

// TestTranslateRequiresLinkageWhenGated is the workflow-side
// fail-closed proof (§5.6): the gated registration refuses to
// translate a request without approval linkage, while the open
// registration and linked gated requests translate normally.
func TestTranslateRequiresLinkageWhenGated(t *testing.T) {
	base := registry.WorkflowRequest{
		InvocationID: "inv-1", TenantID: "t1",
		Input: json.RawMessage(`{"subject_id":"s1"}`),
	}
	if _, err := translate(base, true); err == nil {
		t.Fatal("gated translate without linkage succeeded")
	}
	linked := base
	linked.ApprovalID = "appr-1"
	if _, err := translate(linked, true); err != nil {
		t.Fatalf("gated translate with linkage: %v", err)
	}
	if _, err := translate(base, false); err != nil {
		t.Fatalf("open translate: %v", err)
	}
}
