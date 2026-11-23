// Canonical effect-time authorization: revalidate authoritative
// persisted state and atomically transition PENDING → ATTEMPTING.
//
// Every consequential provider initiation passes through
// authorizeAttempt. It runs in its own transaction and pool
// checkout (never the caller's tx), takes no advisory lock, and
// follows global order RLock → 1 → 2 → 4 → 5 (level 3 joins only
// for sequence-bound operations; see SequenceVerifier). The
// provider call always happens after commit, outside this
// transaction.
package effects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
)

// RefusalReason names why authorization refused. Terminality is
// carried by error identity, never inferred from the name.
type RefusalReason string

const (
	RefusalApprovalMissing   RefusalReason = "approval-missing"
	RefusalApprovalDenied    RefusalReason = "approval-denied"
	RefusalApprovalExpired   RefusalReason = "approval-expired"
	RefusalApprovalCancelled RefusalReason = "approval-cancelled"
	RefusalApprovalMismatch  RefusalReason = "approval-mismatch"
	RefusalGrantDenied       RefusalReason = "grant-denied"
	RefusalTrustModeMismatch RefusalReason = "trust-mode-mismatch"
	RefusalKillSwitch        RefusalReason = "kill-switch-engaged"
	RefusalTokenMismatch     RefusalReason = "token-mismatch"
	RefusalStaleOwnership    RefusalReason = "stale-ownership"
	RefusalBindingMismatch   RefusalReason = "binding-mismatch"
)

// RefusalError is a terminal fail-closed authorization refusal.
// Terminal-invocation withdrawals commit inside the authorization
// transaction with their paired audit (see terminalWithdrawTx);
// they are reported through the decision, never through this type.
type RefusalError struct {
	Reason RefusalReason
	Msg    string
}

// Error renders the refusal with its reason code.
func (e *RefusalError) Error() string {
	return fmt.Sprintf("effects: authorization refused (%s): %s", string(e.Reason), e.Msg)
}

// ErrApprovalPending signals the wait path: the approval is
// pending and valid, so the effect must park — never authorize.
// Callers catch this and enter durable waiting; it is not a
// failure and withdraws nothing.
var ErrApprovalPending = errors.New("effects: approval pending; effect waits, never authorizes")

// SequenceRef carries sequence identity as pure data. It proves
// nothing by itself; the installed SequenceVerifier checks it
// against authoritative durable state inside the transaction.
type SequenceRef struct {
	Subject    string
	Generation int64
}

// SequenceBinding is the fully qualified fence check: the
// claimed identity plus the authorizing context it must match.
type SequenceBinding struct {
	TenantID     string
	InvocationID string
	OpID         string
	Ref          SequenceRef
}

// SequenceVerifier checks entity-sequence generation ownership.
// Implementations live with their table owner (today: the probe
// package owns the only sequence domain); effects owns the
// interface, the binding rules, and the fail-closed defaults.
// The check runs inside the authorization transaction at
// lock-order position 3: it must read its generation row
// FOR SHARE, open no transaction, call no provider, and take no
// advisory lock.
type SequenceVerifier interface {
	VerifyGeneration(ctx context.Context, tx pgx.Tx, binding SequenceBinding) error
}

// AttemptMode selects the step-5 source predicate. One function,
// three predicated shapes — never three authorization paths.
type AttemptMode int

const (
	// AuthorizeFresh authorizes a first attempt: PENDING row,
	// attempt ID matches the claim.
	AuthorizeFresh AttemptMode = iota
	// AuthorizeTakeover authorizes a stale-PENDING takeover past
	// the lease, adopting a fresh attempt ID.
	AuthorizeTakeover
	// AuthorizeReconcile re-authorizes a NOT_APPLIED row with a
	// fresh attempt ID after full revalidation.
	AuthorizeReconcile
)

// authzDecision is the committed outcome of the authorization
// transaction. Exactly one of rec (authorized) or refusal (a
// terminal-invocation decision committed with its paired audit) is
// set. logged carries the in-transaction audit event needing its
// post-commit log line, or nil when nothing was recorded in-tx.
type authzDecision struct {
	rec     contract.EffectRecord
	refusal error
	logged  *audit.Event
}

// authorizeAttempt evaluates the complete authorization read set
// and conditionally transitions the effect to ATTEMPTING in one
// database transaction. On success the returned record is
// ATTEMPTING and the caller must send through the canonical
// path. Refusals and wait signals never transition; infra
// failures are retryable. A terminal-invocation decision commits
// first (withdrawal, if won, with its paired audit in the same
// transaction) and is then returned as a typed refusal; a commit
// failure surfaces as infrastructure failure and never reports a
// withdrawal.
//
// attemptID is the caller's ownership epoch: fresh attempts
// present the claimed ID, takeovers and reconciles present a
// newly minted one. Empty is rejected (fail closed).
func (s *Service) authorizeAttempt(ctx context.Context, tenantID string, intent Intent, mode AttemptMode, attemptID string) (contract.EffectRecord, error) {
	if tenantID == "" || intent.BusinessOpID == "" || intent.InvocationID == "" {
		return contract.EffectRecord{}, fmt.Errorf("effects: tenant, business operation ID, and invocation ID are required")
	}
	if attemptID == "" {
		return contract.EffectRecord{}, fmt.Errorf("effects: attempt ID is required")
	}
	if s.grants == nil || s.approver == nil || s.registry == nil {
		return contract.EffectRecord{}, fmt.Errorf("effects: authorizeAttempt dependencies are not wired")
	}
	s.grants.RLock()
	defer s.grants.RUnlock()
	attr := &invocationAuth{}
	out, rerr := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (authzDecision, error) {
		return s.authorizeTx(ctx, tx, tenantID, intent, mode, attemptID, attr)
	})
	if rerr != nil {
		// Nothing committed: audit genuine refusals only.
		// Contention, waits, supersession, and infra failures
		// are not refusals.
		if reason, ok := refusalReason(rerr); ok {
			s.auditRefusal(ctx, tenantID, intent, *attr, reason)
		}
		return contract.EffectRecord{}, rerr
	}
	if out.logged != nil && s.auditor != nil {
		s.auditor.Log(*out.logged)
	}
	if out.refusal != nil {
		return contract.EffectRecord{}, out.refusal
	}
	return out.rec, nil
}

// refusalReason classifies an authorization outcome for audit:
// terminal refusals (and terminal invocations) audit; waits,
// contention, supersession, and infra noise do not.
func refusalReason(err error) (string, bool) {
	var refusal *RefusalError
	if errors.As(err, &refusal) {
		return string(refusal.Reason), true
	}
	if errors.Is(err, ErrInvocationCancelled) {
		return "invocation-ineligible", true
	}
	return "", false
}

// authorizeTx runs the ordered read set and predicated transition
// inside one transaction. It returns (decision, nil) when the
// transaction may commit — either an authorization or a
// terminal-invocation decision with its paired audit — and
// (zero, outcome) otherwise; outcome is a *RefusalError,
// ErrApprovalPending, a contention error, or an infra error.
func (s *Service) authorizeTx(ctx context.Context, tx pgx.Tx, tenantID string, intent Intent, mode AttemptMode, attemptID string, attr *invocationAuth) (authzDecision, error) {
	// Lock 1: invocation identity + liveness + token + tool/trust.
	inv, err := loadInvocationTx(ctx, tx, tenantID, intent.InvocationID)
	if err != nil {
		if isNoRows(err) {
			// Deterministic permanent condition (forged ID
			// or phantom-bound op): terminal binding
			// refusal, routed through the refusal-audit
			// path (which skips as unattributable — nothing
			// was loaded). Genuine connection errors stay
			// retryable below.
			return authzDecision{}, &RefusalError{Reason: RefusalBindingMismatch,
				Msg: "invocation has no row; the binding is invalid"}
		}
		return authzDecision{}, err
	}
	*attr = inv
	if intent.EffectToken == "" || inv.token == "" || intent.EffectToken != inv.token {
		return authzDecision{}, &RefusalError{Reason: RefusalTokenMismatch, Msg: "effect token does not match invocation"}
	}
	live, terminal := invocationEligible(inv.status)
	if !live {
		if terminal {
			// Irreversible terminal (cancelled/failed/succeeded):
			// withdraw a pending row and pair its audit in
			// this same tx, then refuse after commit.
			return s.terminalWithdrawTx(ctx, tx, tenantID, intent, inv)
		}
		// Pending and quarantined refuse without withdrawing:
		// pending is unsubmitted (nothing to withdraw from the
		// submitter's view — the row may still park), and
		// quarantined is resumable.
		return authzDecision{}, terminalInvocationRefusal(inv.status)
	}
	// Tool resolution (in-memory registry): determines whether an
	// approval linkage is required and which kill switch applies.
	manifest, _, err := s.registry.Resolve(inv.miniapp, inv.version)
	if err != nil {
		return authzDecision{}, &RefusalError{Reason: RefusalBindingMismatch, Msg: fmt.Sprintf("unknown miniapp %s", inv.miniapp)}
	}
	tool, err := findTool(manifest, inv.tool)
	if err != nil {
		return authzDecision{}, &RefusalError{Reason: RefusalBindingMismatch, Msg: fmt.Sprintf("unknown tool %s", inv.tool)}
	}
	// Persisted-trust admission: the manifest's declared TrustModes
	// admit the invocation's persisted trust, independent of the
	// approval linkage below. An empty list declares no restriction
	// (legacy manifests); an explicit list refuses anything
	// unlisted, so the linkage exception can never launder a
	// manifest-disallowed trust mode. Caller claims cannot
	// influence this: Intent carries no trust field.
	if len(tool.TrustModes) > 0 && !trustAdmitted(tool.TrustModes, inv.trust) {
		return authzDecision{}, &RefusalError{Reason: RefusalTrustModeMismatch,
			Msg: fmt.Sprintf("tool %s does not admit %s trust", inv.tool, inv.trust)}
	}
	// Lock 2: approval linkage (gated tools only), skipped when the
	// persisted invocation trust is autonomous and the manifest
	// explicitly permits autonomous execution — mirroring the
	// submit-time choke, which admits autonomous gated workflows
	// without linkage. All other checks still apply.
	if tool.Approval.Required && !autonomousLinkageExempt(tool, inv.trust) {
		appr, err := s.approver.ByInvocationTx(ctx, tx, tenantID, intent.InvocationID)
		if err != nil {
			if isNoRows(err) {
				return authzDecision{}, &RefusalError{Reason: RefusalApprovalMissing, Msg: "no approval linkage for invocation"}
			}
			return authzDecision{}, err
		}
		if out := checkApprovalBinding(inv, appr, tool); out != nil {
			return authzDecision{}, out
		}
	}
	// Lock 3: sequence fence (sequence-bound operations only).
	if intent.Sequence != nil {
		if s.sequences == nil {
			return authzDecision{}, &RefusalError{Reason: RefusalStaleOwnership, Msg: "sequence identity present but no sequence verifier installed"}
		}
		if err := s.sequences.VerifyGeneration(ctx, tx, SequenceBinding{
			TenantID: tenantID, InvocationID: intent.InvocationID,
			OpID: intent.BusinessOpID, Ref: *intent.Sequence,
		}); err != nil {
			return authzDecision{}, &RefusalError{Reason: RefusalStaleOwnership, Msg: err.Error()}
		}
	}
	// Lock 4: kill switches FOR SHARE + pinned grant read (RLock
	// held by the caller across the whole tx).
	if hit, err := s.engagedSwitchTx(ctx, tx, tenantID, inv.miniapp, intent.Operation.CapabilityID, tool.Limits.KillSwitch); err != nil {
		return authzDecision{}, err
	} else if hit != "" {
		return authzDecision{}, &RefusalError{Reason: RefusalKillSwitch, Msg: fmt.Sprintf("kill switch %q is engaged", hit)}
	}
	if !s.grants.GrantAllowsLocked(tenantID, inv.miniapp, inv.trust) {
		return authzDecision{}, &RefusalError{Reason: RefusalGrantDenied, Msg: fmt.Sprintf("no live grant covers %s at this trust", inv.miniapp)}
	}
	// Lock 5: predicated effect transition (last statement).
	rec, err := s.transitionAuthorized(ctx, tx, tenantID, intent, mode, attemptID)
	if err != nil {
		return authzDecision{}, err
	}
	return authzDecision{rec: rec}, nil
}

// terminalWithdrawTx executes the terminal-invocation decision
// inside the authorization transaction: the predicated
// PENDING → CANCELLED flip, the affected-row check, and the paired
// audit insert commit or roll back together. The withdrawal wins
// exactly when the row is still PENDING; otherwise the refusal
// audits as refused, never as a second withdrawal. The returned
// decision carries the typed refusal for post-commit return; any
// error rolls back with no audit and surfaces as infra failure.
func (s *Service) terminalWithdrawTx(ctx context.Context, tx pgx.Tx, tenantID string, intent Intent, inv invocationAuth) (authzDecision, error) {
	won, err := s.withdrawIfPending(ctx, tx, tenantID, intent.BusinessOpID)
	if err != nil {
		return authzDecision{}, err
	}
	var logged *audit.Event
	if s.auditor != nil {
		action, result := "effect-refused", "refused"
		if won {
			action, result = "effect-withdrawn", "withdrawn"
		}
		details, _ := json.Marshal(map[string]any{
			"business_op_id": intent.BusinessOpID,
			"invocation_id":  intent.InvocationID,
			"reason":         "invocation-ineligible",
		})
		e := audit.Event{
			TenantID: tenantID, AgentID: "framework",
			AuthorityKind: string(contract.AuthorityService), AuthorityID: "effects.authorize",
			MiniappID: inv.miniapp, Version: inv.version, Tool: inv.tool,
			Action: action, Outcome: result, InvocationID: intent.InvocationID,
			Details: details,
		}
		if err := s.auditor.InsertTx(ctx, tx, e); err != nil {
			return authzDecision{}, err
		}
		logged = &e
	}
	return authzDecision{refusal: terminalInvocationRefusal(inv.status), logged: logged}, nil
}

// invocationEligible reports (live, terminal) under P21-002
// semantics: pending is ineligible (unsubmitted by construction),
// terminal/quarantined are ineligible, and only irreversible
// terminals (cancelled/failed/succeeded) may withdraw effects —
// quarantined is resumable, so it refuses without withdrawing.
func invocationEligible(status contract.RunStatus) (live, terminal bool) {
	switch status {
	case contract.RunRunning, contract.RunWaitingEvent, contract.RunWaitingApproval:
		return true, false
	case contract.RunCancelled, contract.RunFailed, contract.RunSucceeded:
		return false, true
	default:
		return false, false
	}
}

// terminalInvocationRefusal maps an irreversible terminal invocation
// to the legacy cancellation error existing callers match.
func terminalInvocationRefusal(status contract.RunStatus) error {
	return fmt.Errorf("%w: invocation is %s", ErrInvocationCancelled, status)
}

// trustAdmitted reports whether a declared trust-mode list admits a
// persisted trust mode.
func trustAdmitted(modes []contract.TrustMode, trust contract.TrustMode) bool {
	for _, m := range modes {
		if m == trust {
			return true
		}
	}
	return false
}

// autonomousLinkageExempt reports whether the persisted
// autonomous trust skips the human-approval linkage: only when the
// admitted tool manifest explicitly permits autonomous execution.
// An empty TrustModes list never grants the exemption.
func autonomousLinkageExempt(tool contract.ToolManifest, trust contract.TrustMode) bool {
	return trust == contract.TrustAutonomous && trustAdmitted(tool.TrustModes, contract.TrustAutonomous)
}

// checkApprovalBinding verifies the linkage against server-read
// invocation state. It returns nil to proceed, ErrApprovalPending
// to wait, or a terminal *RefusalError.
func checkApprovalBinding(inv invocationAuth, appr approvals.Record, tool contract.ToolManifest) error {
	if appr.TenantID != inv.tenantID || appr.InvocationID != inv.id {
		return &RefusalError{Reason: RefusalApprovalMismatch, Msg: "approval bound to a different tenant or invocation"}
	}
	if appr.MiniappID != inv.miniapp || appr.Version != inv.version || appr.Operation != inv.tool {
		return &RefusalError{Reason: RefusalApprovalMismatch, Msg: "approval bound to a different miniapp, version, or tool"}
	}
	want := contract.IntentHash(inv.miniapp, inv.version, inv.tool, inv.input)
	if appr.IntentHash != want {
		return &RefusalError{Reason: RefusalApprovalMismatch, Msg: "approval intent does not match invocation"}
	}
	if appr.ApproverScope != tool.Approval.ApproverScope {
		return &RefusalError{Reason: RefusalApprovalMismatch, Msg: "approval scope does not match tool requirement"}
	}
	switch appr.Status {
	case "pending":
		if appr.ExpiresAt.IsZero() || time.Now().UTC().Before(appr.ExpiresAt) {
			return ErrApprovalPending
		}
		return &RefusalError{Reason: RefusalApprovalExpired, Msg: "approval expired"}
	case "decided":
		if appr.Decision != contract.DecisionApprove {
			return &RefusalError{Reason: RefusalApprovalDenied, Msg: fmt.Sprintf("approval decided %s", appr.Decision)}
		}
		if appr.DecidedBy == "" || appr.DecidedBy == appr.RequestedBy {
			return &RefusalError{Reason: RefusalApprovalMismatch, Msg: "approval decision lacks valid principal attribution"}
		}
		if !appr.ExpiresAt.IsZero() && time.Now().UTC().After(appr.ExpiresAt) {
			return &RefusalError{Reason: RefusalApprovalExpired, Msg: "approval expired"}
		}
		return nil
	case "cancelled":
		return &RefusalError{Reason: RefusalApprovalCancelled, Msg: "approval cancelled"}
	case "expired":
		return &RefusalError{Reason: RefusalApprovalExpired, Msg: "approval expired"}
	default:
		return &RefusalError{Reason: RefusalApprovalMismatch, Msg: fmt.Sprintf("approval in unexpected status %q", appr.Status)}
	}
}

// invocationAuth is the server-read invocation state an
// authorization decision rests on.
type invocationAuth struct {
	id       string
	tenantID string
	miniapp  string
	version  string
	tool     string
	trust    contract.TrustMode
	token    string
	status   contract.RunStatus
	runID    string
	input    json.RawMessage
}

// loadInvocationTx reads the invocation row FOR UPDATE inside the
// authorization transaction (lock-order position 1).
func loadInvocationTx(ctx context.Context, tx pgx.Tx, tenantID, invocationID string) (invocationAuth, error) {
	var inv invocationAuth
	var trust, status, input string
	err := tx.QueryRow(ctx, `SELECT id::text, miniapp_id, miniapp_version, tool,
		trust_mode, effect_token, status, run_id, input::text
		FROM app.invocations WHERE id = $1::uuid AND tenant_id = $2 FOR UPDATE`,
		invocationID, tenantID).Scan(
		&inv.id, &inv.miniapp, &inv.version, &inv.tool,
		&trust, &inv.token, &status, &inv.runID, &input)
	if err != nil {
		return invocationAuth{}, err
	}
	inv.tenantID = tenantID
	inv.trust = contract.TrustMode(trust)
	inv.status = contract.RunStatus(status)
	inv.input = json.RawMessage(input)
	return inv, nil
}

// withdrawIfPending flips a still-PENDING row to CANCELLED. It
// reports whether this call won the withdrawal; a statement error
// is infrastructure failure, never a lost withdrawal.
func (s *Service) withdrawIfPending(ctx context.Context, tx pgx.Tx, tenantID, opID string) (bool, error) {
	tag, err := tx.Exec(ctx, `UPDATE app.effect_records SET state = 'CANCELLED', updated_at = now()
		WHERE business_op_id = $1 AND tenant_id = $2 AND state = 'PENDING'`,
		opID, tenantID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// transitionAuthorized performs the step-5 predicated effect
// transition (the last statement of the authorization
// transaction) and loads the committed row. Zero rows reloads
// and classifies: contention, supersession, staleness, or
// binding mismatch — never misread as success.
func (s *Service) transitionAuthorized(ctx context.Context, tx pgx.Tx, tenantID string, intent Intent, mode AttemptMode, attemptID string) (contract.EffectRecord, error) {
	var tag struct {
		query string
		args  []any
	}
	switch mode {
	case AuthorizeFresh:
		tag.query = `UPDATE app.effect_records SET state = 'ATTEMPTING', updated_at = now()
			WHERE business_op_id = $1 AND tenant_id = $2 AND state = 'PENDING'
			AND attempt_id = $3 AND invocation_id = $4`
		tag.args = []any{intent.BusinessOpID, tenantID, attemptID, intent.InvocationID}
	case AuthorizeTakeover:
		tag.query = `UPDATE app.effect_records SET state = 'ATTEMPTING', attempt_id = $3, updated_at = now()
			WHERE business_op_id = $1 AND tenant_id = $2 AND state = 'PENDING'
			AND updated_at <= $4 AND invocation_id = $5`
		tag.args = []any{intent.BusinessOpID, tenantID, attemptID,
			time.Now().UTC().Add(-s.lease()), intent.InvocationID}
	case AuthorizeReconcile:
		tag.query = `UPDATE app.effect_records SET state = 'ATTEMPTING', attempt_id = $3, updated_at = now()
			WHERE business_op_id = $1 AND tenant_id = $2 AND state = 'NOT_APPLIED'
			AND invocation_id = $4`
		tag.args = []any{intent.BusinessOpID, tenantID, attemptID, intent.InvocationID}
	default:
		return contract.EffectRecord{}, fmt.Errorf("effects: unknown authorization mode %d", mode)
	}
	res, err := tx.Exec(ctx, tag.query, tag.args...)
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if res.RowsAffected() == 1 {
		return s.loadTx(ctx, tx, tenantID, intent.BusinessOpID)
	}
	return s.classifyLoss(ctx, tx, tenantID, intent, attemptID)
}

// loadTx reads one effect row inside the caller's transaction,
// including the invocation binding the public load omits.
func (s *Service) loadTx(ctx context.Context, tx pgx.Tx, tenantID, opID string) (contract.EffectRecord, error) {
	return s.loadTxMode(ctx, tx, tenantID, opID, "")
}

// loadTxLocked reads one effect row FOR UPDATE inside the
// caller's transaction (same lock position as the step-5 write,
// so classifiers serialize instead of racing each other).
func (s *Service) loadTxLocked(ctx context.Context, tx pgx.Tx, tenantID, opID string) (contract.EffectRecord, error) {
	return s.loadTxMode(ctx, tx, tenantID, opID, " FOR UPDATE")
}

func (s *Service) loadTxMode(ctx context.Context, tx pgx.Tx, tenantID, opID, lock string) (contract.EffectRecord, error) {
	var r contract.EffectRecord
	var inv string
	err := tx.QueryRow(ctx, `SELECT business_op_id, attempt_id, idempotency_key, capability_id,
		operation, state, provider_request, provider_reply, updated_at, invocation_id
		FROM app.effect_records WHERE business_op_id = $1 AND tenant_id = $2`+lock,
		opID, tenantID).Scan(
		&r.BusinessOpID, &r.AttemptID, &r.IdempotencyKey, &r.CapabilityID,
		&r.Operation, &r.State, &r.ProviderRequest, &r.ProviderReply, &r.UpdatedAt, &inv)
	if err != nil {
		return contract.EffectRecord{}, err
	}
	r.InvocationID = inv
	return r, nil
}

// classifyLoss explains a zero-row step-5 write from current
// committed state, read FOR UPDATE (same lock position —
// serializes classifiers; the tx ends immediately after).
// Every outcome converges on caller retry: contention backs off,
// supersession reloads and re-routes, staleness mints a fresh
// attempt, mismatch fails closed.
func (s *Service) classifyLoss(ctx context.Context, tx pgx.Tx, tenantID string, intent Intent, attemptID string) (contract.EffectRecord, error) {
	rec, err := s.loadTxLocked(ctx, tx, tenantID, intent.BusinessOpID)
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if rec.InvocationID != "" && rec.InvocationID != intent.InvocationID {
		return contract.EffectRecord{}, &RefusalError{Reason: RefusalBindingMismatch,
			Msg: "effect is bound to a different invocation"}
	}
	fresh := time.Since(rec.UpdatedAt) < s.lease()
	switch rec.State {
	case contract.EffectAttempting:
		if fresh {
			return rec, ErrAttemptInProgress
		}
		return rec, ErrAttemptSuperseded
	case contract.EffectPending:
		// Same attempt still owning a live row: contention
		// (back off; a duplicate same-attempt authorize races
		// its twin). Anything else on a fresh row is also
		// live contention; only a stale row with a foreign
		// attempt means our ownership died.
		if rec.AttemptID == attemptID || fresh {
			return rec, ErrAttemptInProgress
		}
		return contract.EffectRecord{}, &RefusalError{Reason: RefusalStaleOwnership,
			Msg: "attempt ownership no longer holds"}
	default:
		return rec, ErrAttemptSuperseded
	}
}

// engagedSwitchTx returns the first engaged applicable kill
// switch inside the authorization transaction (lock-order
// position 4), or "" when none is engaged.
func (s *Service) engagedSwitchTx(ctx context.Context, tx pgx.Tx, tenantID, miniapp, capabilityID, toolSwitch string) (string, error) {
	var hit string
	err := tx.QueryRow(ctx, `SELECT name FROM app.kill_switches
		WHERE name = ANY($1) AND engaged LIMIT 1 FOR SHARE`,
		killSwitchNames(tenantID, miniapp, capabilityID, toolSwitch)).Scan(&hit)
	if err != nil {
		if isNoRows(err) {
			return "", nil
		}
		return "", err
	}
	return hit, nil
}

// auditRefusal best-effort records a refusal after a rolled-back
// authorization transaction. Terminal-invocation decisions audit
// inside that transaction instead (see terminalWithdrawTx), so
// this path only ever emits effect-refused. A nil auditor skips
// silently; production and refusal-asserting tests always wire one.
func (s *Service) auditRefusal(ctx context.Context, tenantID string, intent Intent, inv invocationAuth, reason string) {
	// Unattributable (invocation never loaded) or unwired
	// auditor: skip rather than fabricate attribution.
	if s.auditor == nil || inv.miniapp == "" || inv.tool == "" {
		return
	}
	const action, result = "effect-refused", "refused"
	details, _ := json.Marshal(map[string]any{
		"business_op_id": intent.BusinessOpID,
		"invocation_id":  intent.InvocationID,
		"reason":         reason,
	})
	_ = s.auditor.Record(ctx, audit.Event{
		TenantID: tenantID, AgentID: "framework",
		AuthorityKind: string(contract.AuthorityService), AuthorityID: "effects.authorize",
		MiniappID: inv.miniapp, Version: inv.version, Tool: inv.tool,
		Action: action, Outcome: result, InvocationID: intent.InvocationID,
		Details: details,
	})
}
