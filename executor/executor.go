package executor

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/PithomLabs/miniapp/contract"
)

// ErrUnknownRun reports that no workflow exists for a run ID.
// Executors wrap it (never bare strings) so the runtime can
// distinguish "never submitted" (converge: submit-if-missing or
// absorb) from "executor unavailable" (retry) without parsing
// messages. Match with IsUnknownRun.
var ErrUnknownRun = errors.New("executor: unknown run")

// IsUnknownRun reports whether err definitively establishes that
// no workflow exists for the requested run ID.
func IsUnknownRun(err error) bool {
	return errors.Is(err, ErrUnknownRun)
}

// Request is a validated, authorized invocation ready for execution.
// Executors trust these fields; all validation and authorization
// happens in the runtime before dispatch.
type Request struct {
	InvocationID   string
	TenantID       string
	Principal      contract.Caller
	MiniappID      string
	Version        string
	Tool           string
	Input          json.RawMessage
	IdempotencyKey string
	TrustMode      contract.TrustMode
	// EntityKey is the tenant-scoped serialization entity derived
	// by the runtime, or empty when the tool declares none.
	EntityKey string
	// EffectToken authorizes effect claims under this invocation.
	// Miniapps pass it through opaquely; the ledger verifies it.
	EffectToken string
	// ApprovalID names the approval gating this invocation, or
	// empty when it was not approval-gated.
	ApprovalID string
	// RunID is the runtime-minted run identity for Class D
	// submissions. Submit is idempotent on it: repeating a Submit
	// with the same RunID attaches to the existing run instead of
	// starting a second one.
	RunID string
}

// SyncExecutor runs request/response tools to completion inside the
// caller's request scope (Class A).
type SyncExecutor interface {
	Execute(ctx context.Context, req Request) (json.RawMessage, error)
}

// AsyncExecutor runs durable tools (Class D) and manages their
// lifecycle by stable run ID. Submit is idempotent on
// Request.RunID: a repeated Submit with an already-submitted ID
// returns the existing run's handle. The runtime relies on this to
// retry across submit-then-crash windows without orphaning runs.
type AsyncExecutor interface {
	Submit(ctx context.Context, req Request) (contract.RunHandle, error)
	Status(ctx context.Context, runID string) (contract.RunStatus, error)
	Signal(ctx context.Context, runID string, sig contract.Signal) error
	Cancel(ctx context.Context, runID string, reason string) error
}
