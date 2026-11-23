package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/internal/store"
)

// Status reports one invocation. Live Class D runs refresh from the
// executor and sync the record; terminal runs answer from the
// record. When the executor cannot be reached for a live run, the
// persisted record is served with Stale set — staleness is always
// explicit, never silent. Cross-tenant lookups fail identically to
// missing records.
func (s *Service) Status(ctx context.Context, credential, invocationID string) (InvokeResult, error) {
	binding, err := s.cfg.Policy.Authenticate(credential)
	if err != nil {
		return InvokeResult{}, terminal("auth-required", "authentication failed")
	}
	tenantID := binding.Authority.TenantID
	rec, err := s.store.getByID(ctx, tenantID, invocationID)
	if err != nil {
		return InvokeResult{}, terminal("unknown-invocation", "no such invocation")
	}
	out := InvokeResult{InvocationID: rec.ID, Status: rec.Status, Result: rec.Result}
	if rec.RunID != "" {
		status := rec.Status
		if status == contract.RunRunning || status == contract.RunWaitingEvent || status == contract.RunWaitingApproval {
			live, err := s.cfg.Async.Status(ctx, rec.RunID)
			if err != nil {
				out.Stale = true
			} else {
				status = live
				_ = s.store.update(ctx, tenantID, rec.ID, live, nil, rec.RunID, "")
				if live == contract.RunSucceeded || live == contract.RunFailed || live == contract.RunCancelled {
					// Settlement audit failures propagate (retryable) so a
					// failed audit is never silently mistaken for a
					// complete chain; the next read converges on the
					// persisted terminal record. (Fault point
					// "audit:run-settled"; full sync-error preservation
					// lands with X-24a.)
					req := InvokeRequest{MiniappID: rec.MiniappID, Version: rec.Version, Tool: rec.Tool}
					if err := s.audit(ctx, req, binding, "run-settled", string(live), rec.ID, "", nil); err != nil {
						return InvokeResult{}, retryable("audit-unavailable", "%v", err)
					}
				}
			}
		}
		if status == contract.RunCancelled {
			// Best-effort cancel redelivery: a cancel recorded
			// while the executor was unreachable converges here.
			// Errors stay silent — the record is already terminal
			// and the next read retries.
			_ = s.cfg.Async.Cancel(ctx, rec.RunID, "cancel redelivery")
		}
		out.Status = status
		out.Handle = s.handleFromRecord(rec, status)
	}
	return out, nil
}

// Signal delivers an external input to a live run.
func (s *Service) Signal(ctx context.Context, credential, runID string, sig contract.Signal) error {
	binding, err := s.cfg.Policy.Authenticate(credential)
	if err != nil {
		return terminal("auth-required", "authentication failed")
	}
	if runID == "" {
		return terminal("invalid-request", "run ID is required")
	}
	tenantID := binding.Authority.TenantID
	rec, err := s.store.getByRun(ctx, tenantID, runID)
	if err != nil {
		return terminal("unknown-run", "no such run")
	}
	if rec.Status != contract.RunRunning && rec.Status != contract.RunWaitingEvent && rec.Status != contract.RunWaitingApproval {
		return terminal("run-not-live", "run is %s", rec.Status)
	}
	if err := s.cfg.Async.Signal(ctx, runID, sig); err != nil {
		return retryable("signal-failed", "%v", err)
	}
	return s.auditRec(ctx, binding, rec, "signal", "delivered", map[string]any{"signal": sig.Type})
}

// CancelResult reports what a cancellation settled. InFlight names
// business operations unfinished at cancel time; they settle by
// evidence (CONFIRMED, FAILED, or UNKNOWN) afterwards, never as
// "cancelled".
type CancelResult struct {
	RunID    string   `json:"run_id"`
	InFlight []string `json:"in_flight,omitempty"`
}

// Cancel stops a live run. Cancellation records durably FIRST —
// no new effect attempt may begin afterwards — then settles pending
// approvals and stops the executor run. A repeated Cancel on an
// already-cancelled invocation redelivers the executor stop instead
// of failing, so cancel-while-unavailable converges on retry (and
// via Status reads) rather than stranding the intent.
func (s *Service) Cancel(ctx context.Context, credential, runID, reason string) (CancelResult, error) {
	binding, err := s.cfg.Policy.Authenticate(credential)
	if err != nil {
		return CancelResult{}, terminal("auth-required", "authentication failed")
	}
	if runID == "" {
		return CancelResult{}, terminal("invalid-request", "run ID is required")
	}
	tenantID := binding.Authority.TenantID
	rec, err := s.store.getByRun(ctx, tenantID, runID)
	if err != nil {
		return CancelResult{}, terminal("unknown-run", "no such run")
	}
	if rec.Status == contract.RunCancelled {
		return s.redeliverCancel(ctx, binding, rec)
	}
	if rec.Status != contract.RunRunning && rec.Status != contract.RunWaitingEvent && rec.Status != contract.RunWaitingApproval {
		return CancelResult{}, terminal("run-not-live", "run is %s", rec.Status)
	}
	if reason == "" {
		reason = "cancelled by operator"
	}
	if err := s.store.update(ctx, tenantID, rec.ID, contract.RunCancelled, nil, rec.RunID, reason); err != nil {
		return CancelResult{}, retryable("store-unavailable", "%v", err)
	}
	if err := s.settleCancel(ctx, binding, rec, reason); err != nil {
		return CancelResult{}, err
	}
	return s.cancelReport(ctx, binding, rec, reason)
}

// redeliverCancel retries the executor stop and approval settlement
// for an already-cancelled invocation.
func (s *Service) redeliverCancel(ctx context.Context, binding auth.Binding, rec invocation) (CancelResult, error) {
	if err := s.settleCancel(ctx, binding, rec, "cancel redelivery"); err != nil {
		return CancelResult{}, err
	}
	live, err := s.cfg.Effects.LiveEffects(ctx, binding.Authority.TenantID, rec.ID)
	if err != nil {
		return CancelResult{}, retryable("store-unavailable", "%v", err)
	}
	return CancelResult{RunID: rec.RunID, InFlight: opIDs(live)}, nil
}

// settleCancel stops the executor run and settles pending approvals.
// An executor error against an already-terminal run is absorbed (the
// run is down regardless); any other executor failure is retryable
// with the durable cancel intent standing.
func (s *Service) settleCancel(ctx context.Context, binding auth.Binding, rec invocation, reason string) error {
	if _, err := s.cfg.Approver.CancelPending(ctx, binding.Authority.TenantID, rec.ID); err != nil {
		return retryable("store-unavailable", "%v", err)
	}
	return s.cancelRun(ctx, rec.RunID, reason)
}

func (s *Service) cancelReport(ctx context.Context, binding auth.Binding, rec invocation, reason string) (CancelResult, error) {
	tenantID := binding.Authority.TenantID
	live, err := s.cfg.Effects.LiveEffects(ctx, tenantID, rec.ID)
	if err != nil {
		return CancelResult{}, retryable("store-unavailable", "%v", err)
	}
	out := CancelResult{RunID: rec.RunID, InFlight: opIDs(live)}
	details := map[string]any{"reason": reason, "in_flight": out.InFlight}
	if err := s.auditRec(ctx, binding, rec, "cancel", "cancelled", details); err != nil {
		return CancelResult{}, retryable("audit-unavailable", "%v", err)
	}
	return out, nil
}

func opIDs(recs []contract.EffectRecord) []string {
	var out []string
	for _, r := range recs {
		out = append(out, r.BusinessOpID)
	}
	return out
}

// DecideApproval records a human decision on a pending approval. The
// decider recomputes the intent hash independently; a mismatch fails
// closed and the waiting invocation keeps waiting. Decisions on
// non-waiting invocations are refused. After the decision commits,
// the waiting run (if any) is notified on the approval topic; the
// signal is a redeliverable notification — Decide is idempotent, so
// a decider that crashes after commit retries to completion, and the
// workflow re-reads the authoritative table on wake.
func (s *Service) DecideApproval(ctx context.Context, credential, approvalID string, decision contract.ApprovalDecision, intentHash contract.Digest) error {
	binding, err := s.cfg.Policy.Authenticate(credential)
	if err != nil {
		return terminal("auth-required", "authentication failed")
	}
	tenantID := binding.Authority.TenantID
	appr, err := s.cfg.Approver.Get(ctx, tenantID, approvalID)
	if err != nil {
		return terminal("unknown-approval", "no such approval")
	}
	rec, err := s.store.getByID(ctx, tenantID, appr.InvocationID)
	if err != nil {
		return retryable("store-unavailable", "%v", err)
	}
	if rec.Status == contract.RunCancelled {
		return terminal("approval-cancelled", "the invocation was cancelled")
	}
	if rec.Status != contract.RunWaitingApproval {
		return terminal("decision-rejected", "invocation is %s, not waiting for approval", rec.Status)
	}
	decider := approvals.Decider{
		Caller: contract.Caller{AgentID: binding.AgentID, Authority: binding.Authority},
		Roles:  binding.Roles,
	}
	if err := s.cfg.Approver.Decide(ctx, tenantID, approvalID, decider, decision, intentHash); err != nil {
		return terminal("decision-rejected", "%v", err)
	}
	if rec.RunID != "" {
		payload, err := json.Marshal(contract.ApprovalSignal{ApprovalID: approvalID, Decision: decision})
		if err != nil {
			return terminal("internal", "cannot encode approval signal")
		}
		sig := contract.Signal{Type: contract.SignalApprovalDecided, Payload: payload}
		if err := s.cfg.Async.Signal(ctx, rec.RunID, sig); err != nil {
			// X-02: the run ID was persisted but the park-time
			// submit never landed. On approve, submit through the
			// choke and notify once. On any other decision there
			// is nothing to notify (no run exists), so the
			// committed decision stands without a signal and
			// poll settles without a run. Any other signal
			// failure stays retryable.
			if !executor.IsUnknownRun(err) {
				return retryable("signal-failed", "%v", err)
			}
			if decision == contract.DecisionApprove {
				req := InvokeRequest{MiniappID: rec.MiniappID, Version: rec.Version,
					Tool: rec.Tool, Input: rec.Input, IdempotencyKey: rec.IdempotencyKey,
					RequestedTrust: rec.TrustMode}
				tool, terr := s.toolFor(req)
				if terr != nil {
					return retryable("submit-failed", "%v", terr)
				}
				key, kerr := resolveEntity(tool.EntityKey, rec.Input, tenantID)
				if kerr != nil {
					return terminal("internal", "persisted input no longer derives its entity: %v", kerr)
				}
				req.entityKey = key
				handle, serr := s.submitRecorded(ctx, req, binding, rec)
				if serr != nil {
					return retryable("submit-failed", "%v", serr)
				}
				if err := s.cfg.Async.Signal(ctx, handle.RunID, sig); err != nil {
					return retryable("signal-failed", "%v", err)
				}
			}
		}
	}
	return s.auditRec(ctx, binding, rec, "approval-decided", string(decision),
		map[string]any{"approval_id": approvalID})
}

// CheckLive reports whether an invocation may start new effect
// attempts. The row is locked during the read so the check
// linearizes against a concurrent Cancel: either the check observes
// the cancellation and refuses, or the cancellation lands after the
// check and the in-flight attempt settles by evidence.
func (s *Service) CheckLive(ctx context.Context, tenantID, invocationID string) error {
	return checkLive(ctx, s.cfg.Pool, tenantID, invocationID)
}

func checkLive(ctx context.Context, pool *pgxpool.Pool, tenantID, invocationID string) error {
	live, err := store.WithTenant(ctx, pool, tenantID, func(tx pgx.Tx) (bool, error) {
		var status string
		err := tx.QueryRow(ctx, `SELECT status FROM app.invocations
			WHERE id = $1::uuid AND tenant_id = $2 FOR UPDATE`, invocationID, tenantID).Scan(&status)
		if err != nil {
			return false, err
		}
		switch contract.RunStatus(status) {
		case contract.RunRunning, contract.RunWaitingEvent, contract.RunWaitingApproval:
			return true, nil
		default:
			return false, nil
		}
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return effects.ErrInvocationCancelled
		}
		return fmt.Errorf("runtime: liveness check unavailable: %w", err)
	}
	if !live {
		return effects.ErrInvocationCancelled
	}
	return nil
}

type effectCanceller struct{ pool *pgxpool.Pool }

// EffectCanceller adapts invocation liveness to the effects
// ledger's cancellation gate. Production wiring passes
// EffectCanceller(pool) to effects.New; every claim then honors
// durable cancellation. It takes the pool (not the Service) so
// construction order never cycles.
func EffectCanceller(pool *pgxpool.Pool) effects.Cancellation {
	return effectCanceller{pool}
}

func (c effectCanceller) CheckCancelled(ctx context.Context, tenantID, invocationID string) error {
	return checkLive(ctx, c.pool, tenantID, invocationID)
}

func (s *Service) handleFromRecord(rec invocation, status contract.RunStatus) *contract.RunHandle {
	return &contract.RunHandle{
		RunID: rec.RunID, MiniappID: rec.MiniappID, Version: contract.Version(rec.Version), Status: status,
		PollPath: "runs.status", SignalPath: "runs.signal", CancelPath: "runs.cancel",
	}
}

// getByID loads one invocation by ID, scoped to the tenant.
func (s *invocations) getByID(ctx context.Context, tenantID, id string) (invocation, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (invocation, error) {
		var r invocation
		var result []byte
		err := tx.QueryRow(ctx, `SELECT id::text, tenant_id, miniapp_id, miniapp_version, tool,
			idempotency_key, input::text, status, COALESCE(result::text, ''), run_id, attached_run_id, error,
			trust_mode, effect_token
			FROM app.invocations WHERE id = $1::uuid AND tenant_id = $2`,
			id, tenantID).Scan(
			&r.ID, &r.TenantID, &r.MiniappID, &r.Version, &r.Tool,
			&r.IdempotencyKey, &r.Input, &r.Status, &result, &r.RunID, &r.AttachedRunID, &r.Err,
			&r.TrustMode, &r.EffectToken)
		if err != nil {
			return invocation{}, err
		}
		r.Result = json.RawMessage(result)
		return r, nil
	})
}
