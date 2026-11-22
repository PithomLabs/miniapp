package contract

import (
	"encoding/json"
	"time"
)

// RunStatus is the lifecycle state of an invocation or task handle.
type RunStatus string

const (
	RunPending         RunStatus = "pending"
	RunRunning         RunStatus = "running"
	RunWaitingApproval RunStatus = "waiting_approval"
	RunWaitingEvent    RunStatus = "waiting_event"
	RunSucceeded       RunStatus = "succeeded"
	RunFailed          RunStatus = "failed"
	RunCancelled       RunStatus = "cancelled"
	RunQuarantined     RunStatus = "quarantined"
)

// RunHandle is the uniform async envelope returned for every
// asynchronous invocation. Active handles carry no arbitrary expiry;
// terminal-history retention is governed by the retention policy.
type RunHandle struct {
	RunID      string    `json:"run_id"`
	MiniappID  string    `json:"miniapp_id"`
	Version    Version   `json:"version"`
	Status     RunStatus `json:"status"`
	PollPath   string    `json:"poll_tool"`
	SignalPath string    `json:"signal_tool"`
	CancelPath string    `json:"cancel_tool"`
}

// Signal delivers an external input (approval, event, cancellation
// detail) to a running execution.
type Signal struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// UntrustedText wraps provider-originated free text. The flag is always
// true by construction: provider content is data for the agent to
// interpret, never instructions and never trusted error prose.
type UntrustedText struct {
	Value     string `json:"value"`
	Untrusted bool   `json:"untrusted"`
}

// Untrusted marks provider text as explicitly untrusted data.
func Untrusted(s string) UntrustedText {
	return UntrustedText{Value: s, Untrusted: true}
}

// EffectState is the outcome state of one external-effect attempt.
type EffectState string

const (
	EffectPending     EffectState = "PENDING"
	EffectAttempting  EffectState = "ATTEMPTING"
	EffectConfirmed   EffectState = "CONFIRMED"
	EffectFailed      EffectState = "FAILED"
	EffectUnknown     EffectState = "UNKNOWN"
	EffectReconciling EffectState = "RECONCILING"
	EffectNotApplied  EffectState = "NOT_APPLIED"
)

// ValidEffectTransitions is the frozen effect state machine. UNKNOWN
// never transitions directly back to ATTEMPTING: reconciliation must
// establish the outcome first.
var ValidEffectTransitions = map[EffectState][]EffectState{
	EffectPending:     {EffectAttempting, EffectCancelled()},
	EffectAttempting:  {EffectConfirmed, EffectFailed, EffectUnknown},
	EffectUnknown:     {EffectReconciling},
	EffectReconciling: {EffectConfirmed, EffectNotApplied, EffectFailed},
	EffectConfirmed:   {},
	EffectFailed:      {},
	EffectNotApplied:  {EffectAttempting},
}

// EffectCancelled is a sentinel for intents withdrawn before dispatch.
func EffectCancelled() EffectState { return "CANCELLED" }

// EffectRecord is the frozen attempt/outcome record for one
// consequential outbound effect.
type EffectRecord struct {
	BusinessOpID    string      `json:"businessOpId"`
	AttemptID       string      `json:"attemptId"`
	IdempotencyKey  string      `json:"idempotencyKey,omitempty"`
	CapabilityID    string      `json:"capabilityId"`
	Operation       string      `json:"operation"`
	State           EffectState `json:"state"`
	ProviderRequest string      `json:"providerRequestId,omitempty"`
	ProviderReply   string      `json:"providerReplyRef,omitempty"`
	UpdatedAt       time.Time   `json:"updatedAt,omitempty"`
}

// Transition moves the record to the next state when the machine allows
// it, and rejects anything else as a terminal contract error.
func (r *EffectRecord) Transition(next EffectState) error {
	for _, allowed := range ValidEffectTransitions[r.State] {
		if allowed == next {
			r.State = next
			r.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return Terminal("state", "illegal effect transition %s -> %s", r.State, next)
}

// ApprovalDecision is the recorded outcome of an approval request.
type ApprovalDecision string

const (
	DecisionApprove ApprovalDecision = "approve"
	DecisionReject  ApprovalDecision = "reject"
	DecisionExpired ApprovalDecision = "expired"
)

// ApprovalRecord binds a decision to the exact requested operation. Any
// material change to the operation invalidates approvals bound to the
// previous intent hash.
type ApprovalRecord struct {
	ID             string           `json:"id"`
	MiniappID      string           `json:"miniappId"`
	MiniappVersion Version          `json:"miniappVersion"`
	InvocationID   string           `json:"invocationId"`
	Operation      string           `json:"operation"`
	Resource       string           `json:"resource,omitempty"`
	IntentHash     Digest           `json:"intentHash"`
	Approver       string           `json:"approver,omitempty"`
	AuthorityKind  AuthorityKind    `json:"authorityKind,omitempty"`
	ExpiresAt      time.Time        `json:"expiresAt,omitempty"`
	Decision       ApprovalDecision `json:"decision,omitempty"`
	DecidedAt      time.Time        `json:"decidedAt,omitempty"`
}
