package effects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

// Operation describes one provider call. TenantID is stamped by
// Perform from the authenticated tenant (never caller-supplied):
// channels attribute every send to the tenant that authorized it.
type Operation struct {
	CapabilityID string
	Operation    string
	Payload      json.RawMessage
	TenantID     string
}

// Receipt carries provider-issued identifiers for a completed call.
type Receipt struct {
	ProviderRequestID string
	ProviderReply     string
}

// PermanentError marks a provider failure that retry cannot fix.
type PermanentError struct{ Err error }

func (e PermanentError) Error() string { return "permanent: " + e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

// UnknownError marks a call whose outcome cannot be established
// (typically timeout-after-send). The effect must quarantine, never
// blindly retry.
type UnknownError struct{ Err error }

func (e UnknownError) Error() string { return "unknown outcome: " + e.Err.Error() }
func (e UnknownError) Unwrap() error { return e.Err }

// ErrAttemptInProgress reports a duplicate Perform while another
// worker holds a live attempt. The caller should retry after a short
// delay; the attempt resolves on its own. It is never terminal.
var ErrAttemptInProgress = errors.New("effects: another attempt is in progress")

// ErrAttemptSuperseded reports that our terminal transition lost to
// a concurrent adopt: the row moved while we were sending. The
// returned record is current; reconciliation evidence (not blind
// retry) decides what happens next.
var ErrAttemptSuperseded = errors.New("effects: attempt superseded by a concurrent transition")

// DefaultAttemptLease bounds how long an ATTEMPTING row looks live.
// A duplicate Perform adopts a stale-looking attempt into UNKNOWN
// and answers contention for a fresh-looking one. The lease must
// exceed any reasonable provider send duration; raise it (via the
// Service field) before enabling slow real providers.
const DefaultAttemptLease = 2 * time.Minute

// Channel performs one provider call. Implementations are capability
// connectors (mock channels in Phase 1, real providers later).
type Channel interface {
	Send(ctx context.Context, op Operation) (Receipt, error)
}

// Intent declares one business operation before any provider contact.
// BusinessOpID must be tenant-scoped (`tenant:local`); unscoped IDs
// are rejected so two tenants' identical local IDs never share a
// ledger row. InvocationID links the effect to its canonical
// invocation for cancellation gating and attribution.
type Intent struct {
	BusinessOpID   string
	AttemptID      string
	IdempotencyKey string
	InvocationID   string
	// EffectToken is the invocation's opaque claim token, passed
	// through from the executor's call. The policy check verifies
	// it against the invocation row.
	EffectToken string
	Operation   Operation
}

// ErrInvocationCancelled reports an effect claim for an invocation
// that is no longer live. No new attempt may begin after
// cancellation is durably recorded.
var ErrInvocationCancelled = errors.New("effects: invocation is cancelled or terminal")

// Cancellation gates effect claims on invocation liveness. The
// runtime implements it with a locked invocation-row read, so the
// check linearizes against concurrent cancellation.
type Cancellation interface {
	CheckCancelled(ctx context.Context, tenantID, invocationID string) error
}

// Service executes effects through the ledger. Attempts are
// at-least-once; confirmed effects are at-most-once per business
// operation; ambiguous outcomes quarantine as UNKNOWN and are never
// blindly retried. Every state change commits inside the tenant
// context with an expected-state predicate, so concurrent workers
// serialize on the row instead of clobbering each other.
type Service struct {
	pool      *pgxpool.Pool
	channels  map[string]Channel
	canceller Cancellation
	policy    PolicyChecker
	// AttemptLease overrides DefaultAttemptLease when positive.
	AttemptLease time.Duration
	// TestFault injects failures at named effect boundaries for
	// tests (see testkit/fault). It is set only by tests, after
	// construction; production constructors leave it nil, and a
	// nil injector never fires, so production behavior is unchanged.
	TestFault *fault.Injector
}

// Config wires an effects service. Every dependency is required:
// canceller gates claims on liveness and policy revalidates
// authority; either missing fails closed at Perform time.
type Config struct {
	Pool      *pgxpool.Pool
	Channels  map[string]Channel
	Canceller Cancellation
	Policy    PolicyChecker
}

// New returns a service dispatching by capability ID.
func New(cfg Config) *Service {
	return &Service{pool: cfg.Pool, channels: cfg.Channels, canceller: cfg.Canceller,
		policy: cfg.Policy, AttemptLease: DefaultAttemptLease}
}

func (s *Service) lease() time.Duration {
	if s.AttemptLease > 0 {
		return s.AttemptLease
	}
	return DefaultAttemptLease
}

// Perform executes one business operation. A repeated call with the
// same business operation ID returns the recorded outcome without
// re-sending; a call racing a live attempt gets ErrAttemptInProgress;
// rows stranded past the attempt lease move to UNKNOWN exactly once.
// Cancellation and policy gate every live path but never settled
// reads; refused claims insert nothing.
func (s *Service) Perform(ctx context.Context, tenantID string, intent Intent) (contract.EffectRecord, error) {
	if tenantID == "" || intent.BusinessOpID == "" || intent.InvocationID == "" {
		return contract.EffectRecord{}, fmt.Errorf("effects: tenant, business operation ID, and invocation ID are required")
	}
	if !strings.HasPrefix(intent.BusinessOpID, contract.ScopePrefix(tenantID)) {
		return contract.EffectRecord{}, fmt.Errorf("effects: business operation ID must be tenant-scoped (%q:local)", tenantID)
	}
	if s.canceller == nil {
		return contract.EffectRecord{}, fmt.Errorf("effects: cancellation checks unavailable (nil canceller)")
	}
	if s.policy == nil {
		return contract.EffectRecord{}, fmt.Errorf("effects: policy checks unavailable (nil policy)")
	}
	// Stamp server-derived tenant attribution onto the operation
	// before any channel can send it. Intent travels by value, so
	// the caller's copy is untouched.
	intent.Operation.TenantID = tenantID
	if rec, err := s.load(ctx, tenantID, intent.BusinessOpID); err == nil {
		if err := s.checkOperation(ctx, tenantID, intent.Operation, rec); err != nil {
			return contract.EffectRecord{}, err
		}
		if settled(rec.State) {
			return rec, nil
		}
	} else if !isNoRows(err) {
		return contract.EffectRecord{}, err
	}
	if err := s.canceller.CheckCancelled(ctx, tenantID, intent.InvocationID); err != nil {
		return contract.EffectRecord{}, err
	}
	if err := s.policy.CheckPolicy(ctx, tenantID, intent); err != nil {
		return contract.EffectRecord{}, err
	}
	rec, created, err := s.claim(ctx, tenantID, intent)
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if !created {
		if err := s.checkOperation(ctx, tenantID, intent.Operation, rec); err != nil {
			return contract.EffectRecord{}, err
		}
		return s.adopt(ctx, tenantID, intent, rec)
	}
	return s.attempt(ctx, tenantID, intent, rec)
}

// checkOperation refuses a repeated business operation ID carrying a
// materially different operation. Attempt IDs legitimately differ;
// the capability, operation, and payload must not.
func (s *Service) checkOperation(ctx context.Context, tenantID string, op Operation, rec contract.EffectRecord) error {
	if op.CapabilityID != rec.CapabilityID || op.Operation != rec.Operation {
		return fmt.Errorf("effects: business operation reused with a different operation")
	}
	stored, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (string, error) {
		var payload string
		err := tx.QueryRow(ctx, `SELECT payload FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, rec.BusinessOpID, tenantID).Scan(&payload)
		return payload, err
	})
	if err != nil {
		return err
	}
	if !samePayload(op.Payload, json.RawMessage(stored)) {
		return fmt.Errorf("effects: business operation reused with a different payload")
	}
	return nil
}

func samePayload(a, b json.RawMessage) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == 0 && len(b) == 0
	}
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		return string(a) == string(b)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return string(a) == string(b)
	}
	ca, err := json.Marshal(va)
	if err != nil {
		return false
	}
	cb, err := json.Marshal(vb)
	if err != nil {
		return false
	}
	return string(ca) == string(cb)
}

// settled reports states that end an operation without further sends.
// Repeating a settled operation is a read, so cancellation never
// blocks it; reconciliation depends on reading terminal outcomes.
func settled(s contract.EffectState) bool {
	switch s {
	case contract.EffectConfirmed, contract.EffectFailed, contract.EffectUnknown,
		contract.EffectReconciling, contract.EffectNotApplied:
		return true
	default:
		return false
	}
}

// attempt moves a PENDING record to ATTEMPTING and performs the
// provider send, recording the outcome. Both the winner (first
// claim) and the take-over path (stale PENDING adopt) execute here.
func (s *Service) attempt(ctx context.Context, tenantID string, intent Intent, rec contract.EffectRecord) (contract.EffectRecord, error) {
	ch, ok := s.channels[intent.Operation.CapabilityID]
	if !ok {
		return s.transition(ctx, tenantID, rec, contract.EffectFailed, "", "unknown capability")
	}
	if err := s.TestFault.Fire("attempt-begin"); err != nil {
		return contract.EffectRecord{}, err
	}
	rec, err := s.transition(ctx, tenantID, rec, contract.EffectAttempting, "", "")
	if err != nil {
		return contract.EffectRecord{}, err
	}
	return s.sendAndRecord(ctx, tenantID, ch, intent, rec)
}

// sendAndRecord performs one provider send for a record already in
// ATTEMPTING and persists the outcome.
func (s *Service) sendAndRecord(ctx context.Context, tenantID string, ch Channel, intent Intent, rec contract.EffectRecord) (contract.EffectRecord, error) {
	receipt, err := ch.Send(ctx, intent.Operation)
	if ferr := s.TestFault.Fire("settle-commit"); ferr != nil {
		return contract.EffectRecord{}, ferr
	}
	switch {
	case err == nil:
		rec, err = s.transition(ctx, tenantID, rec, contract.EffectConfirmed, receipt.ProviderRequestID, receipt.ProviderReply)
	case errors.As(err, &UnknownError{}):
		rec, err = s.transition(ctx, tenantID, rec, contract.EffectUnknown, "", err.Error())
	default:
		rec, err = s.transition(ctx, tenantID, rec, contract.EffectFailed, "", err.Error())
	}
	if err != nil {
		return rec, err
	}
	// settle-response models a lost response after a committed
	// outcome settlement: the outcome stands, and a repeat Perform
	// observes it without re-sending.
	if ferr := s.TestFault.Fire("settle-response"); ferr != nil {
		return contract.EffectRecord{}, ferr
	}
	return rec, nil
}

// claim inserts the PENDING record, or loads the existing one when
// this business operation was already attempted. The boolean reports
// whether this call owns the first attempt.
func (s *Service) claim(ctx context.Context, tenantID string, intent Intent) (contract.EffectRecord, bool, error) {
	if err := s.TestFault.Fire("effect-reserve"); err != nil {
		return contract.EffectRecord{}, false, err
	}
	inserted, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (bool, error) {
		tag, err := tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id, capability_id, operation, payload, state)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PENDING')
			ON CONFLICT (business_op_id) DO NOTHING`,
			intent.BusinessOpID, tenantID, intent.AttemptID, intent.IdempotencyKey, intent.InvocationID,
			intent.Operation.CapabilityID, intent.Operation.Operation, string(intent.Operation.Payload))
		if err != nil {
			return false, err
		}
		return tag.RowsAffected() == 1, nil
	})
	if err != nil {
		return contract.EffectRecord{}, false, err
	}
	rec, err := s.load(ctx, tenantID, intent.BusinessOpID)
	if err != nil {
		return contract.EffectRecord{}, false, err
	}
	return rec, inserted, nil
}

// load reads one record inside the tenant context. The tenant
// predicate is explicit: RLS alone does not bind privileged roles.
func (s *Service) load(ctx context.Context, tenantID, opID string) (contract.EffectRecord, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (contract.EffectRecord, error) {
		var r contract.EffectRecord
		err := tx.QueryRow(ctx, `SELECT business_op_id, attempt_id, idempotency_key, capability_id,
			operation, state, provider_request, provider_reply, updated_at
			FROM app.effect_records WHERE business_op_id = $1 AND tenant_id = $2`, opID, tenantID).Scan(
			&r.BusinessOpID, &r.AttemptID, &r.IdempotencyKey, &r.CapabilityID,
			&r.Operation, &r.State, &r.ProviderRequest, &r.ProviderReply, &r.UpdatedAt)
		return r, err
	})
}

// adopt handles a repeated Perform for an already-recorded operation.
// Settled outcomes return untouched. A fresh live row — PENDING or
// ATTEMPTING within the attempt lease — answers contention: another
// worker owns it. A stale ATTEMPTING row quarantines to UNKNOWN. A
// stale PENDING row was never attempted, so the adopter takes it
// over and executes the attempt itself.
func (s *Service) adopt(ctx context.Context, tenantID string, intent Intent, rec contract.EffectRecord) (contract.EffectRecord, error) {
	if settled(rec.State) {
		return rec, nil
	}
	switch rec.State {
	case contract.EffectPending, contract.EffectAttempting:
		if time.Since(rec.UpdatedAt) < s.lease() {
			return rec, ErrAttemptInProgress
		}
		if rec.State == contract.EffectPending {
			return s.takeover(ctx, tenantID, intent, rec)
		}
		return s.adoptStale(ctx, tenantID, rec)
	default:
		return rec, fmt.Errorf("effects: unknown state %q", rec.State)
	}
}

// takeover claims a stale PENDING row for this worker and executes
// the attempt. The age predicate keeps the take-over race-safe: if
// the row moved (or freshened) under us, contention — not a blind
// send — is the answer.
func (s *Service) takeover(ctx context.Context, tenantID string, intent Intent, rec contract.EffectRecord) (contract.EffectRecord, error) {
	r := rec
	if err := r.Transition(contract.EffectAttempting); err != nil {
		return contract.EffectRecord{}, err
	}
	if err := s.TestFault.Fire("takeover-begin"); err != nil {
		return contract.EffectRecord{}, err
	}
	r.UpdatedAt = time.Now().UTC()
	cutoff := time.Now().UTC().Add(-s.lease())
	affected, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (int64, error) {
		tag, err := tx.Exec(ctx, `UPDATE app.effect_records SET state = 'ATTEMPTING', updated_at = now()
			WHERE business_op_id = $1 AND tenant_id = $2 AND state = 'PENDING' AND updated_at <= $3`,
			r.BusinessOpID, tenantID, cutoff)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	})
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if affected == 0 {
		fresh, lerr := s.load(ctx, tenantID, rec.BusinessOpID)
		if lerr != nil {
			return contract.EffectRecord{}, lerr
		}
		return fresh, ErrAttemptInProgress
	}
	ch, ok := s.channels[intent.Operation.CapabilityID]
	if !ok {
		return s.transition(ctx, tenantID, r, contract.EffectFailed, "", "unknown capability")
	}
	return s.sendAndRecord(ctx, tenantID, ch, intent, r)
}

// transition moves one record through the contract state machine,
// guarded by an expected-state predicate: the write lands only when
// the row still shows the state we based our decision on. A lost
// race returns the current record with ErrAttemptSuperseded instead
// of overwriting the winner.
func (s *Service) transition(ctx context.Context, tenantID string, rec contract.EffectRecord, next contract.EffectState, req, reply string) (contract.EffectRecord, error) {
	r := rec
	if err := r.Transition(next); err != nil {
		return contract.EffectRecord{}, err
	}
	r.ProviderRequest, r.ProviderReply = req, reply
	r.UpdatedAt = time.Now().UTC()
	affected, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (int64, error) {
		tag, err := tx.Exec(ctx, `UPDATE app.effect_records SET state = $1, provider_request = $2,
			provider_reply = $3, updated_at = now()
			WHERE business_op_id = $4 AND tenant_id = $5 AND state = $6`,
			string(r.State), req, reply, r.BusinessOpID, tenantID, string(rec.State))
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	})
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if affected == 0 {
		fresh, lerr := s.load(ctx, tenantID, rec.BusinessOpID)
		if lerr != nil {
			return contract.EffectRecord{}, lerr
		}
		return fresh, fmt.Errorf("%w: %s moved while transitioning", ErrAttemptSuperseded, rec.BusinessOpID)
	}
	return r, nil
}

// adoptStale quarantines a claim older than the lease. The age
// predicate makes the quarantine itself race-safe: a worker that
// touched the row after our read keeps its attempt.
func (s *Service) adoptStale(ctx context.Context, tenantID string, rec contract.EffectRecord) (contract.EffectRecord, error) {
	r := rec
	if err := r.Transition(contract.EffectUnknown); err != nil {
		return contract.EffectRecord{}, err
	}
	r.ProviderReply = "adopted stale attempt"
	r.UpdatedAt = time.Now().UTC()
	cutoff := time.Now().UTC().Add(-s.lease())
	affected, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (int64, error) {
		tag, err := tx.Exec(ctx, `UPDATE app.effect_records SET state = 'UNKNOWN',
			provider_reply = $1, updated_at = now()
			WHERE business_op_id = $2 AND tenant_id = $3 AND state = $4 AND updated_at <= $5`,
			r.ProviderReply, r.BusinessOpID, tenantID, string(rec.State), cutoff)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	})
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if affected == 0 {
		fresh, lerr := s.load(ctx, tenantID, rec.BusinessOpID)
		if lerr != nil {
			return contract.EffectRecord{}, lerr
		}
		return fresh, ErrAttemptInProgress
	}
	return r, nil
}

// ListUnknown returns a tenant's quarantined records oldest-first,
// bounded by limit. Operators work the quarantine from this listing;
// nothing here re-drives an outcome on its own.
func (s *Service) ListUnknown(ctx context.Context, tenantID string, limit int) ([]contract.EffectRecord, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("effects: limit must be 1..1000")
	}
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) ([]contract.EffectRecord, error) {
		rows, err := tx.Query(ctx, `SELECT business_op_id, attempt_id, idempotency_key, capability_id,
			operation, state, provider_request, provider_reply, updated_at
			FROM app.effect_records WHERE tenant_id = $1 AND state = 'UNKNOWN'
			ORDER BY updated_at ASC LIMIT $2`, tenantID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []contract.EffectRecord
		for rows.Next() {
			var r contract.EffectRecord
			if err := rows.Scan(&r.BusinessOpID, &r.AttemptID, &r.IdempotencyKey, &r.CapabilityID,
				&r.Operation, &r.State, &r.ProviderRequest, &r.ProviderReply, &r.UpdatedAt); err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, rows.Err()
	})
}

// LiveEffects returns one invocation's unfinished records
// (PENDING or ATTEMPTING). Cancellation consults it to distinguish
// a clean cancel from one with in-flight effects.
func (s *Service) LiveEffects(ctx context.Context, tenantID, invocationID string) ([]contract.EffectRecord, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) ([]contract.EffectRecord, error) {
		rows, err := tx.Query(ctx, `SELECT business_op_id, attempt_id, idempotency_key, capability_id,
			operation, state, provider_request, provider_reply, updated_at
			FROM app.effect_records WHERE tenant_id = $1 AND invocation_id = $2
			AND state IN ('PENDING', 'ATTEMPTING') ORDER BY updated_at ASC`, tenantID, invocationID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []contract.EffectRecord
		for rows.Next() {
			var r contract.EffectRecord
			if err := rows.Scan(&r.BusinessOpID, &r.AttemptID, &r.IdempotencyKey, &r.CapabilityID,
				&r.Operation, &r.State, &r.ProviderRequest, &r.ProviderReply, &r.UpdatedAt); err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, rows.Err()
	})
}

// SetKillSwitch engages or releases a named kill switch, effective
// on the next claim without a redeploy. Names follow the level
// convention (global, tenant:<id>, miniapp:<id>,
// capability:<id>) or a manifest-declared switch.
func (s *Service) SetKillSwitch(ctx context.Context, name string, engaged bool) error {
	if name == "" {
		return fmt.Errorf("effects: kill-switch name is required")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO app.kill_switches (name, engaged, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (name) DO UPDATE SET engaged = $2, updated_at = now()`, name, engaged)
	return err
}

// Settle records an operator decision on a quarantined record,
// following the frozen machine (UNKNOWN → RECONCILING → terminal).
// Only CONFIRMED, FAILED, or NOT_APPLIED settle; anything else (and
// any non-UNKNOWN row) is rejected. The note should cite the
// evidence behind the decision.
func (s *Service) Settle(ctx context.Context, tenantID, opID string, outcome contract.EffectState, note string) (contract.EffectRecord, error) {
	switch outcome {
	case contract.EffectConfirmed, contract.EffectFailed, contract.EffectNotApplied:
	default:
		return contract.EffectRecord{}, fmt.Errorf("effects: cannot settle to %s", outcome)
	}
	rec, err := s.load(ctx, tenantID, opID)
	if err != nil {
		return contract.EffectRecord{}, err
	}
	if rec.State != contract.EffectUnknown {
		return contract.EffectRecord{}, fmt.Errorf("effects: only UNKNOWN records settle (have %s)", rec.State)
	}
	if err := s.TestFault.Fire("settle-operator-commit"); err != nil {
		return contract.EffectRecord{}, err
	}
	rec, err = s.transition(ctx, tenantID, rec, contract.EffectReconciling, rec.ProviderRequest, note)
	if err != nil {
		return contract.EffectRecord{}, err
	}
	return s.transition(ctx, tenantID, rec, outcome, rec.ProviderRequest, note)
}
