package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
)

// invocation is one durable idempotency record.
type invocation struct {
	ID             string
	TenantID       string
	MiniappID      string
	Version        string
	Tool           string
	IdempotencyKey string
	Input          json.RawMessage
	Status         contract.RunStatus
	Result         json.RawMessage
	RunID          string
	// AttachedRunID names an adopted winner run for attached
	// invocations (R1). It carries no uniqueness constraint:
	// many observers may reference one owning run.
	AttachedRunID string
	Err           string
	TrustMode     contract.TrustMode
	EffectToken   string
}

// invocations persists idempotency records. All access runs inside
// the tenant context.
type invocations struct {
	pool *pgxpool.Pool
}

// KeyConflict reports an idempotency-key reuse with materially
// different input. Replaying the first record would attribute the
// wrong result to the wrong request, so the conflict fails closed.
type KeyConflict struct {
	Key string
}

func (e *KeyConflict) Error() string {
	return "idempotency key " + e.Key + " was already used with different input"
}

// claim inserts a record in the given initial state, or returns the
// existing record when this idempotency key was already seen.
// created reports first claim. A repeated key with materially
// different input is a *KeyConflict, never a silent replay of the
// first record. Gated tools claim as pending (R3) so no approval
// requirement can be skipped between claim and park.
func (s *invocations) claim(ctx context.Context, tenantID string, req InvokeRequest, invocationID, effectToken string, initial contract.RunStatus) (invocation, bool, error) {
	created, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (bool, error) {
		tag, err := tx.Exec(ctx, `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input, status, trust_mode, effect_token)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (tenant_id, miniapp_id, miniapp_version, tool, idempotency_key) DO NOTHING`,
			invocationID, tenantID, req.MiniappID, req.Version, req.Tool, req.IdempotencyKey, string(req.Input),
			string(initial), string(req.RequestedTrust), effectToken)
		if err != nil {
			return false, err
		}
		return tag.RowsAffected() == 1, nil
	})
	if err != nil {
		return invocation{}, false, err
	}
	rec, err := s.get(ctx, tenantID, req)
	if err != nil {
		return invocation{}, false, err
	}
	if !created && !sameInput(rec.Input, req.Input) {
		return invocation{}, false, &KeyConflict{Key: req.IdempotencyKey}
	}
	return rec, created, nil
}

func (s *invocations) get(ctx context.Context, tenantID string, req InvokeRequest) (invocation, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (invocation, error) {
		var r invocation
		var result []byte
		err := tx.QueryRow(ctx, `SELECT id::text, tenant_id, miniapp_id, miniapp_version, tool,
			idempotency_key, input::text, status, COALESCE(result::text, ''), run_id, attached_run_id, error,
			trust_mode, effect_token
			FROM app.invocations
			WHERE tenant_id = $1 AND miniapp_id = $2 AND miniapp_version = $3
			  AND tool = $4 AND idempotency_key = $5`,
			tenantID, req.MiniappID, req.Version, req.Tool, req.IdempotencyKey).Scan(
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

// getByRun loads an invocation by run ID, verifying tenant ownership
// explicitly. RLS defense-in-depth cannot be relied on here because
// privileged roles bypass it; the application check is mandatory.
func (s *invocations) getByRun(ctx context.Context, tenantID, runID string) (invocation, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (invocation, error) {
		var r invocation
		var result []byte
		err := tx.QueryRow(ctx, `SELECT id::text, tenant_id, miniapp_id, miniapp_version, tool,
			idempotency_key, input::text, status, COALESCE(result::text, ''), run_id, attached_run_id, error,
			trust_mode, effect_token
			FROM app.invocations WHERE run_id = $1`, runID).Scan(
			&r.ID, &r.TenantID, &r.MiniappID, &r.Version, &r.Tool,
			&r.IdempotencyKey, &r.Input, &r.Status, &result, &r.RunID, &r.AttachedRunID, &r.Err,
			&r.TrustMode, &r.EffectToken)
		if err != nil {
			return invocation{}, err
		}
		if r.TenantID != tenantID {
			return invocation{}, fmt.Errorf("invocation belongs to another tenant")
		}
		r.Result = json.RawMessage(result)
		return r, nil
	})
}

func (s *invocations) update(ctx context.Context, tenantID, id string, status contract.RunStatus, result json.RawMessage, runID, errMsg string) error {
	resultArg := ""
	if len(result) > 0 {
		resultArg = string(result)
	}
	_, err := store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE app.invocations SET status = $1,
			result = NULLIF($2, '')::jsonb, run_id = $3, error = $4, updated_at = now()
			WHERE id = $5::uuid AND tenant_id = $6`, string(status), resultArg, runID, errMsg, id, tenantID)
		return true, err
	})
	return err
}

// parkTx transitions one invocation to waiting_approval inside the
// caller's transaction (the atomic Park's first statement, lock
// order §4.2 position 1). The predicate accepts pending records
// and pre-R3 legacy running records (which claimed before gated
// tools started as pending); it reports whether this call won the
// transition so the caller can adopt rather than overwrite.
func parkTx(ctx context.Context, tx pgx.Tx, tenantID, id string) (bool, error) {
	tag, err := tx.Exec(ctx, `UPDATE app.invocations SET status = 'waiting_approval', updated_at = now()
		WHERE id = $1::uuid AND tenant_id = $2 AND status IN ('pending', 'running')`,
		id, tenantID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// adoptRunID records a minted run ID only when the invocation has
// neither its own run nor an adopted one. Concurrent same-key
// submitters race here: exactly one wins the predicate; losers
// reload and adopt the winner's run instead of minting their own
// (X-03). Reports whether this call won.
func (s *invocations) adoptRunID(ctx context.Context, tenantID, id, runID string) (bool, error) {
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) (bool, error) {
		tag, err := tx.Exec(ctx, `UPDATE app.invocations SET run_id = $1, updated_at = now()
			WHERE id = $2::uuid AND tenant_id = $3 AND run_id = '' AND attached_run_id = ''`,
			runID, id, tenantID)
		if err != nil {
			return false, err
		}
		return tag.RowsAffected() == 1, nil
	})
}
