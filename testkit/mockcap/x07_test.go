// Discriminating test for X-07 (mock measurement integrity).
//
// RED/GREEN-COMPATIBLE SOURCE: this file must compile and run
// against both the pre-remediation snapshot and the remediated
// tree. Use only mockcap/effects APIs present in both (Send,
// SentCount, ModeAccept, Operation{CapabilityID, Operation,
// Payload}, the channel/payload helpers). The red run overlays
// this exact file onto the snapshot; any new-API use breaks the
// red build loudly instead of producing the predicted failure.
package mockcap_test

import (
	"context"
	"testing"

	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

// TestMockRecordsEveryAttempt: two actual Channel.Send invocations
// for one logical operation must produce two measured sends.
// Red (old op_id-PK log with ON CONFLICT DO NOTHING): the second
// send is suppressed, SentCount = 1, FAIL. Green (per-attempt log):
// SentCount = 2, PASS. Row-level identity and attribution are
// covered by TestMockAttemptRowsAttributed on the remediated tree.
func TestMockRecordsEveryAttempt(t *testing.T) {
	ch, db := channel(t)
	ctx := context.Background()
	op := effects.Operation{CapabilityID: "mock", Operation: "send",
		Payload: payload(mockcap.ModeAccept, "op-double")}

	if _, err := ch.Send(ctx, op); err != nil {
		t.Fatalf("first send = %v", err)
	}
	if _, err := ch.Send(ctx, op); err != nil {
		t.Fatalf("second send = %v", err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-double"); n != 2 {
		t.Fatalf("SentCount = %d, want 2 (one row per actual send)", n)
	}
}
