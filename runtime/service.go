package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

// InvokeRequest is the canonical invocation envelope. Credential is
// the presented API key; agent, authority, and tenant resolve
// server-side and are never accepted from the caller. RequestedTrust
// above the tenant grant is denied, never silently downgraded.
type InvokeRequest struct {
	MiniappID      string
	Version        string
	Tool           string
	Input          json.RawMessage
	IdempotencyKey string
	Credential     string
	RequestedTrust contract.TrustMode
	// entityKey is the tenant-scoped entity derived at dispatch.
	// Callers never set it; Invoke resolves it server-side.
	entityKey string
	// approvalID names the approval gating this invocation.
	// Set only on the approval path, before submit.
	approvalID string
}

// InvokeResult is either an inline result (Class A), a run handle
// (Class D), or a pending approval pointer. Stale reports that a
// live status could not be refreshed because the executor was
// unreachable, so Status carries the last persisted record —
// never mistaken for a fresh live reading.
type InvokeResult struct {
	InvocationID string              `json:"invocation_id"`
	Status       contract.RunStatus  `json:"status"`
	Result       json.RawMessage     `json:"result,omitempty"`
	Handle       *contract.RunHandle `json:"handle,omitempty"`
	ApprovalID   string              `json:"approval_id,omitempty"`
	Stale        bool                `json:"stale,omitempty"`
}

// Config wires the service. Every dependency is required; New fails
// closed on any nil so a half-wired service cannot serve.
type Config struct {
	Registry *registry.Registry
	Policy   *auth.Policy
	Pool     *pgxpool.Pool
	Sync     executor.SyncExecutor
	Async    executor.AsyncExecutor
	Approver *approvals.Service
	Auditor  *audit.Service
	Effects  *effects.Service
}

// Service is the canonical invocation service.
//
// TestFault injects failures at named lifecycle boundaries for
// tests (see testkit/fault). It is set only by tests, after
// construction; production constructors leave it nil, and a nil
// injector never fires, so production behavior is unchanged.
type Service struct {
	cfg       Config
	store     *invocations
	validator *validator
	TestFault *fault.Injector
}

// New builds a service, rejecting incomplete wiring.
func New(cfg Config) (*Service, error) {
	if cfg.Registry == nil || cfg.Pool == nil || cfg.Sync == nil ||
		cfg.Async == nil || cfg.Approver == nil || cfg.Auditor == nil || cfg.Effects == nil {
		return nil, fmt.Errorf("runtime: registry, pool, executors, approvals, audit, and effects are all required")
	}
	return &Service{cfg: cfg, store: &invocations{pool: cfg.Pool}, validator: newValidator()}, nil
}

// IntentHash delegates to the shared contract implementation
// (single definition; this wrapper preserves existing call sites,
// including frozen regression tests).
func IntentHash(miniapp, version, tool string, input json.RawMessage) contract.Digest {
	return contract.IntentHash(miniapp, version, tool, input)
}

// Invoke executes one governed invocation: validate, authenticate,
// schema-check, authorize, deduplicate, optionally gate on approval,
// then dispatch by execution class. Audit is mandatory for every
// state-changing outcome; idempotent replay keeps audit failure from
// corrupting the invocation record.
func (s *Service) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	if req.MiniappID == "" || req.Version == "" || req.Tool == "" {
		return InvokeResult{}, terminal("invalid-request", "miniapp, version, and tool are required")
	}
	if len(req.Input) == 0 {
		req.Input = json.RawMessage(`{}`)
	}
	manifest, _, err := s.cfg.Registry.Resolve(req.MiniappID, req.Version)
	if err != nil {
		return InvokeResult{}, terminal("unknown-miniapp", "%v", err)
	}
	tool, err := findTool(manifest, req.Tool)
	if err != nil {
		return InvokeResult{}, terminal("unknown-tool", "%v", err)
	}
	binding, err := s.cfg.Policy.Authenticate(req.Credential)
	if err != nil {
		return InvokeResult{}, terminal("auth-required", "authentication failed")
	}
	tenantID := binding.Authority.TenantID
	if err := s.validator.validate(manifest, tool, req.Input); err != nil {
		s.auditBestEffort(ctx, req, binding, "invoke", "invalid-input", "")
		if re, ok := err.(*Error); ok {
			return InvokeResult{}, re
		}
		return InvokeResult{}, terminal("invalid-input", "%v", err)
	}
	decision, _, err := s.cfg.Policy.Authorize(auth.AuthorizeRequest{
		Binding: binding, TenantID: tenantID, Manifest: manifest, Tool: tool, Requested: req.RequestedTrust,
	})
	if err != nil {
		s.auditBestEffort(ctx, req, binding, "invoke", "forbidden", "")
		return InvokeResult{}, terminal("forbidden", "%v", err)
	}
	entityKey, err := resolveEntity(tool.EntityKey, req.Input, tenantID)
	if err != nil {
		s.auditBestEffort(ctx, req, binding, "invoke", "invalid-entity", "")
		return InvokeResult{}, terminal("invalid-entity", "%v", err)
	}
	req.entityKey = entityKey
	key := req.IdempotencyKey
	if key == "" {
		key = uuid.NewString()
	}
	req.IdempotencyKey = key
	invocationID := uuid.NewString()
	initial := contract.RunRunning
	if decision == auth.NeedsApproval {
		initial = contract.RunPending
	}
	rec, created, err := s.store.claim(ctx, tenantID, req, invocationID, uuid.NewString(), initial)
	if err != nil {
		var conflict *KeyConflict
		if errors.As(err, &conflict) {
			s.auditBestEffort(ctx, req, binding, "invoke", "key-reuse", "")
			return InvokeResult{}, terminal("idempotency-key-reuse", "%v", conflict)
		}
		return InvokeResult{}, retryable("store-unavailable", "%v", err)
	}
	if !created {
		return s.replay(ctx, req, binding, tool, rec, true)
	}
	// claim-response models a lost success response after the claim
	// commit: the row stands, the caller retries, and the retry
	// converges through replay on the same record — never a second
	// run. One-shot per the fault-injection contract.
	if err := s.TestFault.Fire("claim-response"); err != nil {
		return InvokeResult{}, retryable("store-unavailable", "%v", err)
	}
	if decision == auth.NeedsApproval {
		return s.requestApproval(ctx, req, binding, tool, rec)
	}
	return s.dispatch(ctx, req, binding, tool, rec)
}

// replay answers a repeated idempotency key from the record without
// re-executing. Terminal outcomes replay verbatim; live Class D runs
// return their handle; interrupted Class A attempts fail closed with
// instructions to resubmit under a new key. The duplicate marker is
// emitted only when the claim actually identified an existing
// invocation (isDuplicate): lost-race reloads answer from the
// record without mislabeling a first-claim caller as a duplicate.
func (s *Service) replay(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation, isDuplicate bool) (InvokeResult, error) {
	if isDuplicate {
		s.auditBestEffort(ctx, req, binding, "invoke", "duplicate", rec.ID)
	}
	switch rec.Status {
	case contract.RunSucceeded:
		out := InvokeResult{InvocationID: rec.ID, Status: rec.Status, Result: rec.Result}
		if rec.RunID != "" {
			out.Handle = s.handleFor(req, rec.RunID, rec.Status)
		}
		return out, nil
	case contract.RunFailed, contract.RunCancelled:
		return InvokeResult{}, terminal("recorded-failure", "prior attempt %s; resubmit with a new idempotency key to retry", rec.Status)
	case contract.RunPending:
		return s.dispatchPending(ctx, req, binding, tool, rec)
	case contract.RunWaitingApproval:
		return s.pollApproval(ctx, req, binding, tool, rec)
	case contract.RunRunning, contract.RunWaitingEvent:
		if tool.Class == contract.ClassA {
			return InvokeResult{}, terminal("interrupted", "prior attempt did not complete; resubmit with a new idempotency key")
		}
		return s.confirmSubmit(ctx, req, binding, tool, rec)
	default:
		return InvokeResult{}, terminal("internal", "unknown invocation status %q", rec.Status)
	}
}

// dispatchPending routes a claimed-but-undispatched record exactly
// as first-claim dispatch would: the decision is recomputed from
// current policy (deterministic given the same inputs). A denial
// fails terminally without settling the record, so a later grant
// restoration can still dispatch it.
func (s *Service) dispatchPending(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (InvokeResult, error) {
	tenantID := binding.Authority.TenantID
	manifest, _, err := s.cfg.Registry.Resolve(req.MiniappID, req.Version)
	if err != nil {
		return InvokeResult{}, terminal("unknown-miniapp", "%v", err)
	}
	decision, _, err := s.cfg.Policy.Authorize(auth.AuthorizeRequest{
		Binding: binding, TenantID: tenantID, Manifest: manifest, Tool: tool, Requested: req.RequestedTrust,
	})
	if err != nil {
		s.auditBestEffort(ctx, req, binding, "invoke", "forbidden", rec.ID)
		return InvokeResult{}, terminal("forbidden", "%v", err)
	}
	if decision == auth.NeedsApproval {
		return s.requestApproval(ctx, req, binding, tool, rec)
	}
	return s.dispatch(ctx, req, binding, tool, rec)
}

// requestApproval records an approval need and parks the invocation.
// Class A proceeds when the agent re-invokes after approval; Class D
// submits immediately and waits on the approval event.
func (s *Service) requestApproval(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (InvokeResult, error) {
	hash := IntentHash(req.MiniappID, req.Version, req.Tool, req.Input)
	var expiry time.Time
	if tool.Approval.Expiry != "" {
		dur, err := time.ParseDuration(tool.Approval.Expiry)
		if err != nil {
			return InvokeResult{}, terminal("invalid-request", "tool approval expiry does not parse: %v", err)
		}
		expiry = time.Now().UTC().Add(dur)
	}
	approvalID, err := s.parkApproval(ctx, req, binding, tool, rec, hash, expiry)
	if err != nil {
		return InvokeResult{}, err
	}
	out := InvokeResult{InvocationID: rec.ID, Status: contract.RunWaitingApproval, ApprovalID: approvalID}
	if tool.Class == contract.ClassD {
		handle, err := s.submitRecorded(ctx, req, binding, rec)
		if err != nil {
			return InvokeResult{}, err
		}
		out.Handle = &handle
	}
	return out, nil
}

// parkApproval atomically parks an invocation on an approval: the
// invocation transition, the approval row, and the audit row commit
// together or roll back together (contract §7/M1, lock order §4.2:
// invocation first, then approval). A legitimate pre-existing
// pending approval is adopted, never duplicated; a lost race
// against a concurrent park adopts the winner. Retries are safe:
// a failed park leaves no partial state.
func (s *Service) parkApproval(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation, hash contract.Digest, expiry time.Time) (string, error) {
	tenantID := binding.Authority.TenantID
	areq := approvals.Request{
		TenantID: tenantID, MiniappID: req.MiniappID, Version: req.Version,
		InvocationID: rec.ID, Operation: req.Tool, IntentHash: hash,
		ApproverScope: tool.Approval.ApproverScope, RequestedBy: binding.AgentID,
		Expiry: expiry,
	}
	type outcome struct {
		id     string
		adopt  bool
		audit  audit.Event
		hasAud bool
	}
	out, err := store.WithTenant(ctx, s.cfg.Pool, tenantID, func(tx pgx.Tx) (outcome, error) {
		parked, err := parkTx(ctx, tx, tenantID, rec.ID)
		if err != nil {
			return outcome{}, err
		}
		if !parked {
			return outcome{adopt: true}, nil
		}
		if err := s.TestFault.Fire("park-update"); err != nil {
			return outcome{}, err
		}
		// Adopt a legitimate pending row (legacy/defensive) rather
		// than duplicating it; the invocation-row lock held above
		// serializes concurrent parks, so at most one row wins.
		var id string
		err = tx.QueryRow(ctx, `SELECT id::text FROM app.approvals
			WHERE tenant_id = $1 AND invocation_id = $2 AND status = 'pending'
			ORDER BY created_at DESC LIMIT 1`, tenantID, rec.ID).Scan(&id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return outcome{}, err
		}
		if err != nil {
			id, err = s.cfg.Approver.InsertTx(ctx, tx, areq)
			if err != nil {
				return outcome{}, err
			}
			// The approval-create op belongs to the Approver: the
			// same injector arms standalone Submit and atomic Park.
			if err := s.cfg.Approver.TestFault.Fire("approval-create"); err != nil {
				return outcome{}, err
			}
		}
		raw, err := json.Marshal(map[string]any{"approval_id": id})
		if err != nil {
			return outcome{}, err
		}
		ev := audit.Event{
			TenantID: tenantID, AgentID: binding.AgentID,
			AuthorityKind: string(binding.Authority.Kind), AuthorityID: binding.Authority.ID,
			MiniappID: req.MiniappID, Version: req.Version, Tool: req.Tool,
			InvocationID: rec.ID, Action: "approval-requested", Outcome: "pending",
			Details: raw,
		}
		if err := s.TestFault.Fire("audit:approval-requested"); err != nil {
			return outcome{}, err
		}
		if err := s.cfg.Auditor.InsertTx(ctx, tx, ev); err != nil {
			return outcome{}, err
		}
		return outcome{id: id, audit: ev, hasAud: true}, nil
	})
	if err != nil {
		return "", retryable("store-unavailable", "%v", err)
	}
	if out.adopt {
		// Lost the park race: adopt the winner's pending approval.
		appr, err := s.cfg.Approver.ByInvocation(ctx, tenantID, rec.ID)
		if err != nil || appr.Status != "pending" {
			return "", retryable("store-unavailable", "park lost race with no discoverable approval")
		}
		return appr.ID, nil
	}
	if out.hasAud {
		s.cfg.Auditor.Log(out.audit)
	}
	return out.id, nil
}

// pollApproval re-checks a waiting invocation: granted approvals
// release Class A execution inline (Class D was already notified at
// decision time); denied, expired, or cancelled approvals settle the
// live run first and then fail terminally; pending approvals keep
// waiting. Expiry is observed lazily here — no sweeper invents
// deadlines; only explicitly supplied expiries settle.
func (s *Service) pollApproval(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (InvokeResult, error) {
	tenantID := binding.Authority.TenantID
	appr, err := s.cfg.Approver.ByInvocation(ctx, tenantID, rec.ID)
	if err != nil {
		return InvokeResult{}, retryable("store-unavailable", "%v", err)
	}
	status := appr.Status
	if status == "pending" && !appr.ExpiresAt.IsZero() && time.Now().UTC().After(appr.ExpiresAt) {
		status = "expired"
	}
	switch status {
	case "decided":
		if appr.Decision != contract.DecisionApprove {
			if err := s.settleDenied(ctx, rec, "approval denied"); err != nil {
				return InvokeResult{}, err
			}
			return s.settlePollTerminal(ctx, req, binding, tool, rec, "denied")
		}
		if tool.Class == contract.ClassD {
			return s.grantedRun(ctx, req, binding, rec)
		}
		return s.executeSync(ctx, req, binding, tool, rec)
	case "expired":
		if err := s.settleDenied(ctx, rec, "approval expired"); err != nil {
			return InvokeResult{}, err
		}
		return s.settlePollTerminal(ctx, req, binding, tool, rec, "expired")
	case "cancelled":
		if err := s.settleDenied(ctx, rec, "approval cancelled"); err != nil {
			return InvokeResult{}, err
		}
		return s.settlePollTerminal(ctx, req, binding, tool, rec, "cancelled")
	default:
		return InvokeResult{InvocationID: rec.ID, Status: contract.RunWaitingApproval, ApprovalID: appr.ID}, nil
	}
}

// grantedRun returns the live handle for a granted Class D
// invocation, submitting first when the park-time submit never
// landed (X-02). A missing run is submitted idempotently through
// the choke with the stored granted linkage; any other executor
// failure stays retryable.
func (s *Service) grantedRun(ctx context.Context, req InvokeRequest, binding auth.Binding, rec invocation) (InvokeResult, error) {
	if rec.RunID == "" {
		handle, err := s.submitRecorded(ctx, req, binding, rec)
		if err != nil {
			return InvokeResult{}, err
		}
		return InvokeResult{InvocationID: rec.ID, Status: contract.RunRunning, Handle: &handle}, nil
	}
	status, err := s.cfg.Async.Status(ctx, rec.RunID)
	if err != nil {
		if !executor.IsUnknownRun(err) {
			return InvokeResult{}, retryable("status-unavailable", "%v", err)
		}
		handle, err := s.submitRecorded(ctx, req, binding, rec)
		if err != nil {
			return InvokeResult{}, err
		}
		status, err = s.cfg.Async.Status(ctx, handle.RunID)
		if err != nil {
			return InvokeResult{}, retryable("status-unavailable", "%v", err)
		}
		return InvokeResult{InvocationID: rec.ID, Status: status, Handle: &handle}, nil
	}
	return InvokeResult{InvocationID: rec.ID, Status: status, Handle: s.handleFor(req, rec.RunID, status)}, nil
}

// settlePollTerminal records a poll settlement (denied, expired,
// or approval-cancelled) from the waiting state only. On losing
// the transition race it reloads and replays on fresh state, so
// the caller always answers from the recorded terminal — never by
// overwriting a concurrent winner. The approval-cancelled branch
// stays unaudited here because Cancel already audited the
// cancellation.
func (s *Service) settlePollTerminal(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation, outcome string) (InvokeResult, error) {
	tenantID := binding.Authority.TenantID
	won, err := s.store.failIfWaiting(ctx, tenantID, rec.ID, rec.RunID, "approval "+outcome)
	if err != nil {
		return InvokeResult{}, retryable("store-unavailable", "%v", err)
	}
	if !won {
		fresh, err := s.store.getByID(ctx, tenantID, rec.ID)
		if err != nil {
			return InvokeResult{}, retryable("store-unavailable", "%v", err)
		}
		return s.replay(ctx, req, binding, tool, fresh, false)
	}
	switch outcome {
	case "denied":
		s.auditBestEffort(ctx, req, binding, "approval-denied", "denied", rec.ID)
		return InvokeResult{}, terminal("approval-denied", "the approval request was denied")
	case "expired":
		s.auditBestEffort(ctx, req, binding, "approval-expired", "expired", rec.ID)
		return InvokeResult{}, terminal("approval-expired", "the approval request expired")
	default:
		return InvokeResult{}, terminal("approval-cancelled", "the approval was cancelled")
	}
}

// settleDenied stops the live run behind a terminally non-approved
// invocation. The invocation is marked failed only after the run is
// confirmed down, so denial can never orphan a sending workflow. A
// run that already reached a terminal state needs no stop.
func (s *Service) settleDenied(ctx context.Context, rec invocation, reason string) error {
	if rec.RunID == "" {
		return nil
	}
	return s.cancelRun(ctx, rec.RunID, reason)
}

// cancelRun stops one executor run, absorbing the error when the run
// is already terminal or never existed (a named-but-never-submitted
// run has nothing to stop, so denial and expiry converge instead of
// looping). Anything else is retryable: the caller keeps its record
// live and tries again. The absorb branch is safe because the choke
// point refuses submissions on non-live approval linkages, so no
// framework path can concurrently submit the run being absorbed.
func (s *Service) cancelRun(ctx context.Context, runID, reason string) error {
	if err := s.cfg.Async.Cancel(ctx, runID, reason); err != nil {
		status, serr := s.cfg.Async.Status(ctx, runID)
		if serr == nil &&
			(status == contract.RunSucceeded || status == contract.RunFailed || status == contract.RunCancelled) {
			return nil
		}
		if executor.IsUnknownRun(err) && executor.IsUnknownRun(serr) {
			return nil
		}
		return retryable("cancel-failed", "%v", err)
	}
	return nil
}

// dispatch runs a first-claim invocation according to its class.
func (s *Service) dispatch(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (InvokeResult, error) {
	if err := s.audit(ctx, req, binding, "invoke", "dispatched", rec.ID, "", nil); err != nil {
		return InvokeResult{}, retryable("audit-unavailable", "%v", err)
	}
	if tool.Class == contract.ClassD {
		handle, err := s.submitRecorded(ctx, req, binding, rec)
		if err != nil {
			return InvokeResult{}, err
		}
		return InvokeResult{InvocationID: rec.ID, Status: contract.RunRunning, Handle: &handle}, nil
	}
	return s.executeSync(ctx, req, binding, tool, rec)
}

// errApprovalRequired signals that a gated tool reached submission
// without a live approval linkage. Callers re-enter approval
// handling (adopt or park); submission without linkage is refused,
// never attempted blindly.
var errApprovalRequired = errors.New("runtime: approval linkage required before submit")

// submitChecked is the single choke point for executor submission
// (Rule 3). Every submit path reloads current state, resolves the
// approval requirement from the admitted manifest plus persisted
// linkage, and mints-or-adopts exactly one run ID:
//
//   - Attached rows follow their target and never submit.
//   - Gated tools require a LIVE approval linkage for this
//     invocation (pending or granted-approve, intent-matched,
//     unexpired): the park-submit-wait architecture submits bound
//     to a pending approval, and the workflow waits on it. Missing
//     linkage refuses with errApprovalRequired (caller re-enters
//     approval handling); decided-against, expired, or mismatched
//     linkage fails closed without submitting.
//   - Run IDs mint only when both run_id and attached_run_id are
//     empty, via predicated adoption: the mint race loser reloads
//     and submits the winner's ID (idempotent attach), never its
//     own second workflow.
//
// Record status is never changed here: callers submit only from
// states whose status is already correct.
// adoptEligible mirrors the adoptRunID live-state predicate: only
// these statuses may gain a workflow binding (winners) or lend one
// to a loser (adoption). Terminal and quarantined records are
// ineligible in both directions.
func adoptEligible(s contract.RunStatus) bool {
	switch s {
	case contract.RunPending, contract.RunRunning, contract.RunWaitingEvent, contract.RunWaitingApproval:
		return true
	default:
		return false
	}
}

func (s *Service) submitChecked(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (contract.RunHandle, error) {
	tenantID := binding.Authority.TenantID
	cur, err := s.store.getByID(ctx, tenantID, rec.ID)
	if err != nil {
		return contract.RunHandle{}, retryable("store-unavailable", "%v", err)
	}
	if cur.AttachedRunID != "" {
		return *s.handleFor(req, cur.AttachedRunID, cur.Status), nil
	}
	if tool.Approval.Required && cur.TrustMode != contract.TrustAutonomous {
		appr, err := s.cfg.Approver.ByInvocation(ctx, tenantID, cur.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return contract.RunHandle{}, errApprovalRequired
			}
			return contract.RunHandle{}, retryable("store-unavailable", "%v", err)
		}
		hash := IntentHash(req.MiniappID, req.Version, req.Tool, cur.Input)
		live := appr.Status == "pending" ||
			(appr.Status == "decided" && appr.Decision == contract.DecisionApprove)
		if !live || appr.IntentHash != hash {
			return contract.RunHandle{}, terminal("approval-not-granted",
				"approval %s is %s; gated execution refused", appr.ID, appr.Status)
		}
		if !appr.ExpiresAt.IsZero() && time.Now().UTC().After(appr.ExpiresAt) {
			return contract.RunHandle{}, terminal("approval-not-granted",
				"approval %s expired; gated execution refused", appr.ID)
		}
		req.approvalID = appr.ID
	}
	runID := cur.RunID
	if runID != "" && !adoptEligible(cur.Status) {
		// Already bound, but the reloaded record is no longer
		// eligible (a concurrent settlement landed after the
		// caller read it live): the bound run must not be
		// submitted. Same convergence as below via replay.
		return contract.RunHandle{}, retryable("run-adoption-ineligible",
			"invocation %s is %s; no workflow submitted", cur.ID, cur.Status)
	}
	if runID == "" {
		if err := s.TestFault.Fire("runid-persist"); err != nil {
			return contract.RunHandle{}, retryable("store-unavailable", "%v", err)
		}
		candidate := "run-" + uuid.NewString()
		adopted, err := s.store.adoptRunID(ctx, tenantID, cur.ID, candidate)
		if err != nil {
			return contract.RunHandle{}, retryable("store-unavailable", "%v", err)
		}
		if !adopted {
			winner, err := s.store.getByID(ctx, tenantID, cur.ID)
			if err != nil || winner.RunID == "" {
				return contract.RunHandle{}, retryable("store-unavailable", "run adoption lost with no winner")
			}
			// The winner may have settled terminally after this
			// request read it live: never submit a workflow for
			// an ineligible invocation. The caller retries the
			// same key and converges via replay on the recorded
			// terminal.
			if !adoptEligible(winner.Status) {
				return contract.RunHandle{}, retryable("run-adoption-ineligible",
					"invocation %s is %s; no workflow submitted", cur.ID, winner.Status)
			}
			runID = winner.RunID
		} else {
			runID = candidate
		}
	}
	handle, err := s.cfg.Async.Submit(ctx, s.executorRequest(req, binding, cur, runID))
	if err != nil {
		return contract.RunHandle{}, retryable("submit-failed", "%v", err)
	}
	handle.RunID = runID
	return handle, nil
}

// submitRecorded submits through the choke point. A submit error
// leaves the record live (never failed), so the agent's same-key
// retry re-enters the choke and attaches to the run if the first
// submit actually landed.
func (s *Service) submitRecorded(ctx context.Context, req InvokeRequest, binding auth.Binding, rec invocation) (contract.RunHandle, error) {
	tool, err := s.toolFor(req)
	if err != nil {
		return contract.RunHandle{}, err
	}
	return s.submitChecked(ctx, req, binding, tool, rec)
}

// confirmSubmit replays a live Class D record through the choke
// point. A gated record without linkage re-enters approval
// handling instead of submitting blind (X-01).
func (s *Service) confirmSubmit(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (InvokeResult, error) {
	handle, err := s.submitChecked(ctx, req, binding, tool, rec)
	if errors.Is(err, errApprovalRequired) {
		return s.requestApproval(ctx, req, binding, tool, rec)
	}
	if err != nil {
		return InvokeResult{}, err
	}
	return InvokeResult{InvocationID: rec.ID, Status: rec.Status, Handle: &handle}, nil
}

// contentionRetryHint qualifies the effect-contention retry
// instruction by record state and tool class, following replay
// semantics: running Class A records answer interrupted/new-key,
// so same-key retry cannot converge there; every other live
// shape converges via same-key replay. Used only by the
// effect-contention mapping; the execution-in-progress message
// is unchanged (out of scope for P24-010).
func contentionRetryHint(status contract.RunStatus, class contract.Class) string {
	if class == contract.ClassA && (status == contract.RunRunning || status == contract.RunWaitingEvent) {
		return "resubmit with a new idempotency key (running Class A records answer interrupted/new-key per replay semantics)"
	}
	return "retry with the same idempotency key"
}

// toolFor resolves the admitted tool for a request. The choke
// point uses it to derive approval requirements from trusted
// registry state, never from caller input.
func (s *Service) toolFor(req InvokeRequest) (contract.ToolManifest, error) {
	manifest, _, err := s.cfg.Registry.Resolve(req.MiniappID, req.Version)
	if err != nil {
		return contract.ToolManifest{}, terminal("unknown-miniapp", "%v", err)
	}
	tool, err := findTool(manifest, req.Tool)
	if err != nil {
		return contract.ToolManifest{}, terminal("unknown-tool", "%v", err)
	}
	return tool, nil
}

func (s *Service) executeSync(ctx context.Context, req InvokeRequest, binding auth.Binding, tool contract.ToolManifest, rec invocation) (InvokeResult, error) {
	tenantID := binding.Authority.TenantID
	ereq := s.executorRequest(req, binding, rec, "")
	// The production Class A path always carries claim hooks
	// (mandatory; the executor fails closed without them).
	ereq.Claim = &executor.ClaimHooks{
		Recheck: func(tx pgx.Tx) (bool, error) {
			return s.store.recheckTx(ctx, tx, tenantID, rec.ID)
		},
		Complete: func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
			return s.store.completeTx(ctx, tx, tenantID, rec.ID, status, result, errMsg)
		},
	}
	result, err := s.cfg.Sync.Execute(ctx, ereq)
	if err == nil {
		if err := s.audit(ctx, req, binding, "invoke", "succeeded", rec.ID, "", nil); err != nil {
			return InvokeResult{}, retryable("audit-unavailable", "%v", err)
		}
		return InvokeResult{InvocationID: rec.ID, Status: contract.RunSucceeded, Result: result}, nil
	}
	var herr *executor.HandlerFailedError
	if errors.As(err, &herr) {
		s.auditBestEffort(ctx, req, binding, "invoke", "failed", rec.ID)
		return InvokeResult{}, terminal("execution-failed", "%v", herr.Err)
	}
	if errors.Is(err, executor.ErrExecutionInProgress) {
		// Loser: nothing executed, nothing written. Terminal
		// conflict naming the same-key retry; no replay (the
		// claim is still held) and no automatic retry loop.
		return InvokeResult{}, terminal("execution-in-progress",
			"another execution holds invocation %s; retry with the same idempotency key", rec.ID)
	}
	if errors.Is(err, executor.ErrAttemptContention) {
		// Ledger contention inside the handler: the attempt
		// rolled back with nothing committed (no FAILED
		// recorded). Terminal conflict; the retry instruction
		// follows replay semantics for this record (C6).
		return InvokeResult{}, terminal("effect-contention",
			"another attempt owns the effect for invocation %s; %s", rec.ID,
			contentionRetryHint(rec.Status, tool.Class))
	}
	if errors.Is(err, executor.ErrExecutionSuperseded) {
		// Rolled back before committing anything: reload and
		// replay on the recorded state.
		fresh, err := s.store.getByID(ctx, tenantID, rec.ID)
		if err != nil {
			return InvokeResult{}, retryable("store-unavailable", "%v", err)
		}
		return s.replay(ctx, req, binding, tool, fresh, false)
	}
	return InvokeResult{}, retryable("store-unavailable", "%v", err)
}

func (s *Service) executorRequest(req InvokeRequest, binding auth.Binding, rec invocation, runID string) executor.Request {
	return executor.Request{
		InvocationID: rec.ID, TenantID: binding.Authority.TenantID,
		Principal: contract.Caller{AgentID: binding.AgentID, Authority: binding.Authority},
		MiniappID: req.MiniappID, Version: req.Version, Tool: req.Tool,
		Input: req.Input, IdempotencyKey: req.IdempotencyKey, TrustMode: req.RequestedTrust,
		EntityKey: req.entityKey, RunID: runID, EffectToken: rec.EffectToken,
		ApprovalID: req.approvalID,
	}
}

func (s *Service) handleFor(req InvokeRequest, runID string, status contract.RunStatus) *contract.RunHandle {
	return &contract.RunHandle{
		RunID: runID, MiniappID: req.MiniappID, Version: contract.Version(req.Version), Status: status,
		PollPath: "runs.status", SignalPath: "runs.signal", CancelPath: "runs.cancel",
	}
}

func findTool(manifest *contract.MiniappManifest, name string) (contract.ToolManifest, error) {
	for _, t := range manifest.Tools {
		if string(t.Name) == name {
			return t, nil
		}
	}
	return contract.ToolManifest{}, fmt.Errorf("tool %q is not declared", name)
}

// audit records a mandatory audit event; callers fail the operation
// when it errors. auditBestEffort records rejections without
// changing their outcome. auditRec attributes lifecycle actions to a
// stored invocation.
func (s *Service) audit(ctx context.Context, req InvokeRequest, binding auth.Binding, action, outcome, invocationID, runID string, details map[string]any) error {
	return s.auditEvent(ctx, binding, req.MiniappID, req.Version, req.Tool, action, outcome, invocationID, runID, details)
}

func (s *Service) auditBestEffort(ctx context.Context, req InvokeRequest, binding auth.Binding, action, outcome, invocationID string) {
	_ = s.audit(ctx, req, binding, action, outcome, invocationID, "", nil)
}

func (s *Service) auditRec(ctx context.Context, binding auth.Binding, rec invocation, action, outcome string, details map[string]any) error {
	return s.auditEvent(ctx, binding, rec.MiniappID, rec.Version, rec.Tool, action, outcome, rec.ID, rec.RunID, details)
}

func (s *Service) auditEvent(ctx context.Context, binding auth.Binding, miniapp, version, tool, action, outcome, invocationID, runID string, details map[string]any) error {
	if err := s.TestFault.Fire("audit:" + action); err != nil {
		return err
	}
	var raw json.RawMessage
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			return err
		}
		raw = b
	}
	return s.cfg.Auditor.Record(ctx, audit.Event{
		TenantID: binding.Authority.TenantID, AgentID: binding.AgentID,
		AuthorityKind: string(binding.Authority.Kind), AuthorityID: binding.Authority.ID,
		MiniappID: miniapp, Version: version, Tool: tool,
		InvocationID: invocationID, RunID: runID, Action: action, Outcome: outcome,
		Details: raw,
	})
}
