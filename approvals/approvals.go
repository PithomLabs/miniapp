package approvals

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

// Request declares one approval need. IntentHash binds the exact
// operation, resource, and arguments digest computed by the caller.
type Request struct {
	TenantID      string
	MiniappID     string
	Version       string
	InvocationID  string
	Operation     string
	Resource      string
	IntentHash    contract.Digest
	ApproverScope string
	RequestedBy   string
	Expiry        time.Time
}

// Decider is the authenticated principal recording a decision: the
// contract caller plus owner-defined roles for scope matching.
type Decider struct {
	contract.Caller
	Roles []string
}

// Record is a durable approval with its decision, if any.
type Record struct {
	ID            string
	TenantID      string
	MiniappID     string
	Version       string
	InvocationID  string
	Operation     string
	Resource      string
	IntentHash    contract.Digest
	ApproverScope string
	Status        string
	RequestedBy   string
	DecidedBy     string
	Decision      contract.ApprovalDecision
	ExpiresAt     time.Time
}

// Service stores approvals in Postgres.
//
// TestFault injects failures at named approval boundaries for
// tests (see testkit/fault). It is set only by tests, after
// construction; production constructors leave it nil, and a nil
// injector never fires, so production behavior is unchanged.
type Service struct {
	pool      *pgxpool.Pool
	TestFault *fault.Injector
}

// New returns a service over pool.
func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Submit records a pending approval request and returns its ID.
func (s *Service) Submit(ctx context.Context, req Request) (string, error) {
	if err := validateRequest(req); err != nil {
		return "", err
	}
	if err := s.TestFault.Fire("approval-create"); err != nil {
		return "", err
	}
	return store.WithTenant(ctx, s.pool, req.TenantID, func(tx pgx.Tx) (string, error) {
		return s.InsertTx(ctx, tx, req)
	})
}

func validateRequest(req Request) error {
	if req.TenantID == "" || req.MiniappID == "" || req.InvocationID == "" || req.Operation == "" {
		return fmt.Errorf("approvals: tenant, miniapp, invocation, and operation are required")
	}
	if err := req.IntentHash.Validate("intentHash"); err != nil {
		return fmt.Errorf("approvals: %w", err)
	}
	if req.RequestedBy == "" {
		return fmt.Errorf("approvals: requesting agent is required")
	}
	return nil
}

// InsertTx validates and inserts one approval request inside the
// caller's transaction, returning its ID. Multi-table atomic
// operations (approval parking) call it within their own
// transaction; Submit wraps it in a single-statement one.
func (s *Service) InsertTx(ctx context.Context, tx pgx.Tx, req Request) (string, error) {
	if err := validateRequest(req); err != nil {
		return "", err
	}
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO app.approvals
		(tenant_id, miniapp_id, miniapp_version, invocation_id, operation, resource,
		 intent_hash, approver_scope, requested_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id::text`,
		req.TenantID, req.MiniappID, req.Version, req.InvocationID, req.Operation,
		req.Resource, string(req.IntentHash), req.ApproverScope, req.RequestedBy,
		nullTime(req.Expiry)).Scan(&id)
	return id, err
}

// Decide records an approval decision. It fails closed when the
// approval is missing, already decided (differently), expired, bound
// to a different intent hash, decided by the requesting agent
// itself, decided under a different tenant, or decided by a caller
// outside the required approver scope. Repeating the identical
// decision (same decider, decision, and intent) succeeds idempotently
// so crashed deciders can retry to completion.
//
// Concurrency (X-04): the approval row is locked on read and every
// decision write is predicated on `pending` with an affected-row
// check, so exactly one of concurrent conflicting decisions wins
// and the losers receive a defined conflict — never a silent
// last-writer-wins overwrite. Decide serializes against expiry
// marking and CancelPending through the same row lock.
func (s *Service) Decide(ctx context.Context, tenantID, id string, decider Decider, decision contract.ApprovalDecision, intentHash contract.Digest) error {
	if decision != contract.DecisionApprove && decision != contract.DecisionReject {
		return fmt.Errorf("approvals: decision must be approve or reject")
	}
	// Outcome codes let the transaction commit bookkeeping (such as
	// marking expiry) while still reporting the outcome as an error to
	// the caller. Returning an error from the closure would roll the
	// marking back.
	const (
		outcomeDecided = "decided"
		outcomeExpired = "expired"
	)
	outcome, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (string, error) {
		rec, err := s.getTxLocked(ctx, tx, tenantID, id)
		if err != nil {
			return "", err
		}
		if rec.TenantID != tenantID {
			return "", fmt.Errorf("approvals: tenant mismatch")
		}
		if rec.Status != "pending" {
			return "", s.reconcileDecided(rec, decider, decision, intentHash)
		}
		if !rec.ExpiresAt.IsZero() && time.Now().UTC().After(rec.ExpiresAt) {
			marked, err := s.markExpiredTx(ctx, tx, id)
			if err != nil {
				return "", err
			}
			if !marked {
				return "", s.reconcileAfterLoss(ctx, tx, tenantID, id, decider, decision, intentHash)
			}
			return outcomeExpired, nil
		}
		if rec.IntentHash != intentHash {
			return "", fmt.Errorf("approvals: intent changed since request; new approval required")
		}
		if decider.AgentID == rec.RequestedBy {
			return "", fmt.Errorf("approvals: the requesting agent cannot approve its own request")
		}
		if decider.Authority.TenantID != tenantID {
			return "", fmt.Errorf("approvals: decider tenant mismatch")
		}
		if rec.ApproverScope != "" && !hasRole(decider.Roles, rec.ApproverScope) {
			return "", fmt.Errorf("approvals: decider lacks required scope %q", rec.ApproverScope)
		}
		if err := s.TestFault.Fire("decide-commit"); err != nil {
			return "", err
		}
		tag, err := tx.Exec(ctx, `UPDATE app.approvals SET status = 'decided', decided_by = $1,
			decision = $2, decided_at = now() WHERE id = $3::uuid AND status = 'pending'`,
			decider.AgentID, string(decision), id)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() != 1 {
			return "", s.reconcileAfterLoss(ctx, tx, tenantID, id, decider, decision, intentHash)
		}
		return outcomeDecided, nil
	})
	if err != nil {
		return err
	}
	if outcome == outcomeExpired {
		return fmt.Errorf("approvals: request expired")
	}
	// decide-response models a lost response after a committed
	// decision: the commit stands, the caller retries, and the
	// identical retry converges idempotently.
	if err := s.TestFault.Fire("decide-response"); err != nil {
		return err
	}
	return nil
}

// CancelPending marks a waiting invocation's pending approvals
// cancelled. Cancellation calls it while settling the run; the
// boolean reports whether any approval was settled.
func (s *Service) CancelPending(ctx context.Context, tenantID, invocationID string) (bool, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (bool, error) {
		tag, err := tx.Exec(ctx, `UPDATE app.approvals SET status = 'cancelled', decided_at = now()
			WHERE tenant_id = $1 AND invocation_id = $2 AND status = 'pending'`,
			tenantID, invocationID)
		if err != nil {
			return false, err
		}
		return tag.RowsAffected() > 0, nil
	})
}

// Get loads one approval by ID within the tenant context.
func (s *Service) Get(ctx context.Context, tenantID, id string) (Record, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (Record, error) {
		return s.getTx(ctx, tx, tenantID, id)
	})
}

func (s *Service) getTx(ctx context.Context, tx pgx.Tx, tenantID, id string) (Record, error) {
	row := tx.QueryRow(ctx, `SELECT id::text, tenant_id, miniapp_id, miniapp_version, invocation_id,
		operation, resource, intent_hash, approver_scope, status, requested_by,
		decided_by, decision, expires_at FROM app.approvals WHERE id = $1::uuid AND tenant_id = $2`, id, tenantID)
	return scanRecord(row)
}

// getTxLocked reads one approval row holding its lock through the
// caller's transaction, so the checks that follow observe a state
// no concurrent decider can change under them.
func (s *Service) getTxLocked(ctx context.Context, tx pgx.Tx, tenantID, id string) (Record, error) {
	row := tx.QueryRow(ctx, `SELECT id::text, tenant_id, miniapp_id, miniapp_version, invocation_id,
		operation, resource, intent_hash, approver_scope, status, requested_by,
		decided_by, decision, expires_at FROM app.approvals
		WHERE id = $1::uuid AND tenant_id = $2 FOR UPDATE`, id, tenantID)
	return scanRecord(row)
}

// reconcileDecided classifies a Decide against an already-terminal
// approval: the identical decision converges idempotently, anything
// else is a defined conflict. A nil return means idempotent success.
func (s *Service) reconcileDecided(rec Record, decider Decider, decision contract.ApprovalDecision, intentHash contract.Digest) error {
	if rec.Status == "decided" && rec.DecidedBy == decider.AgentID &&
		rec.Decision == decision && rec.IntentHash == intentHash {
		return nil
	}
	return fmt.Errorf("approvals: already decided (%s)", rec.Status)
}

// reconcileAfterLoss reloads the row that a predicated write just
// lost and classifies this Decide against the winner. Callers hold
// the row lock, so the reload observes the committed winner.
func (s *Service) reconcileAfterLoss(ctx context.Context, tx pgx.Tx, tenantID, id string, decider Decider, decision contract.ApprovalDecision, intentHash contract.Digest) error {
	rec, err := s.getTx(ctx, tx, tenantID, id)
	if err != nil {
		return err
	}
	return s.reconcileDecided(rec, decider, decision, intentHash)
}

// markExpiredTx transitions a pending, past-expiry approval to
// expired. Reports whether this call won the transition.
func (s *Service) markExpiredTx(ctx context.Context, tx pgx.Tx, id string) (bool, error) {
	tag, err := tx.Exec(ctx, `UPDATE app.approvals SET status = 'expired'
		WHERE id = $1::uuid AND status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ByInvocation loads the latest approval for one invocation within the
// tenant context. The runtime uses it to release waiting invocations.
func (s *Service) ByInvocation(ctx context.Context, tenantID, invocationID string) (Record, error) {
	if err := s.TestFault.Fire("approval-linkage"); err != nil {
		return Record{}, err
	}
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (Record, error) {
		row := tx.QueryRow(ctx, `SELECT id::text, tenant_id, miniapp_id, miniapp_version, invocation_id,
			operation, resource, intent_hash, approver_scope, status, requested_by,
			decided_by, decision, expires_at FROM app.approvals
			WHERE tenant_id = $1 AND invocation_id = $2
			ORDER BY created_at DESC LIMIT 1`, tenantID, invocationID)
		return scanRecord(row)
	})
}

func scanRecord(row pgx.Row) (Record, error) {
	var r Record
	var hash, decision string
	var expiresAt *time.Time
	err := row.Scan(
		&r.ID, &r.TenantID, &r.MiniappID, &r.Version, &r.InvocationID, &r.Operation,
		&r.Resource, &hash, &r.ApproverScope, &r.Status, &r.RequestedBy,
		&r.DecidedBy, &decision, &expiresAt)
	if err != nil {
		return Record{}, fmt.Errorf("approvals: %w", err)
	}
	r.IntentHash = contract.Digest(hash)
	r.Decision = contract.ApprovalDecision(decision)
	if expiresAt != nil {
		r.ExpiresAt = *expiresAt
	}
	return r, nil
}

func hasRole(roles []string, scope string) bool {
	for _, r := range roles {
		if r == scope {
			return true
		}
	}
	return false
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
