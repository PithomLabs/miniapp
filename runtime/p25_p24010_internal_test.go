package runtime

// P24-010 qualifier matrix: the effect-contention retry hint
// follows replay semantics per record state and tool class.
import (
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
)

func TestP25ContentionRetryHint(t *testing.T) {
	cases := []struct {
		name   string
		status contract.RunStatus
		class  contract.Class
		want   string
	}{
		{"running Class A", contract.RunRunning, contract.ClassA, "new idempotency key"},
		{"waiting-event Class A", contract.RunWaitingEvent, contract.ClassA, "new idempotency key"},
		{"pending Class A", contract.RunPending, contract.ClassA, "same idempotency key"},
		{"waiting-approval Class A", contract.RunWaitingApproval, contract.ClassA, "same idempotency key"},
		{"running Class D", contract.RunRunning, contract.ClassD, "same idempotency key"},
		{"waiting-event Class D", contract.RunWaitingEvent, contract.ClassD, "same idempotency key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := contentionRetryHint(tc.status, tc.class)
			if !strings.Contains(got, tc.want) {
				t.Errorf("hint = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}
