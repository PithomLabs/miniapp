package executor

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

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

// ErrExecutionInProgress reports that another attempt currently
// holds the invocation's execution claim. Nothing was executed
// and nothing was written: the caller must retry with the same
// idempotency key, converging via replay once the holder commits
// (or claiming and executing if the holder crashed — for pending
// or waiting records; running records answer interrupted/new-key
// per replay semantics).
var ErrExecutionInProgress = errors.New("executor: execution in progress")

// IsExecutionInProgress reports a lost execution claim.
func IsExecutionInProgress(err error) bool {
	return errors.Is(err, ErrExecutionInProgress)
}

// ErrExecutionSuperseded reports that the attempt rolled back
// before committing anything: either the invocation was already
// terminal when the claim was acquired, or a concurrent
// settlement won while this attempt executed. The caller must
// reload and answer from the recorded state, never by
// overwriting it.
var ErrExecutionSuperseded = errors.New("executor: execution superseded")

// IsExecutionSuperseded reports a rolled-back attempt.
func IsExecutionSuperseded(err error) bool {
	return errors.Is(err, ErrExecutionSuperseded)
}

// ErrAttemptContention reports that the handler's ledger attempt
// lost contention: another attempt owns the operation. The whole
// attempt transaction rolled back — nothing committed, no FAILED
// recorded. The caller must retry with the same idempotency key,
// converging via replay or adopting the winner (for pending or
// waiting records; running records answer interrupted/new-key
// per replay semantics).
var ErrAttemptContention = errors.New("executor: effect attempt contention")

// IsAttemptContention reports a rolled-back ledger contention.
func IsAttemptContention(err error) bool {
	return errors.Is(err, ErrAttemptContention)
}

// HandlerFailedError reports that the handler returned an error:
// its database writes rolled back to savepoint, the FAILED
// outcome committed, and the claim released. The invocation is
// terminally failed; audit it and answer execution-failed.
type HandlerFailedError struct {
	Err error
}

// Error returns the failure with the handler's message preserved.
func (e *HandlerFailedError) Error() string {
	return "executor: handler failed: " + e.Err.Error()
}

// Unwrap returns the handler's original error.
func (e *HandlerFailedError) Unwrap() error {
	return e.Err
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
	// Claim carries the invocation-scoped callbacks the sync
	// executor runs inside the attempt transaction (liveness
	// re-check before the handler, terminal completion after).
	// Mandatory for production Class A execution: a nil Claim
	// (or nil callback) fails closed and the handler never runs.
	Claim *ClaimHooks
}

// ClaimHooks are invocation-scoped callbacks the sync executor
// runs inside the attempt transaction. The runtime owns
// invocation SQL; the executor owns transaction orchestration.
type ClaimHooks struct {
	// Recheck reloads invocation liveness after the execution
	// claim is held. False rolls the attempt back without
	// reaching the handler.
	Recheck func(tx pgx.Tx) (live bool, err error)
	// Complete records the terminal outcome. False rolls the
	// attempt back: a concurrent settlement won.
	Complete func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error)
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
