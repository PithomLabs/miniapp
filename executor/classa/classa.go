package classa

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
)

// Executor runs Class A tools inside one tenant transaction.
type Executor struct {
	pool     *pgxpool.Pool
	registry *registry.Registry
}

// New returns an executor over pool resolving handlers from reg.
func New(pool *pgxpool.Pool, reg *registry.Registry) *Executor {
	return &Executor{pool: pool, registry: reg}
}

// Execute resolves the tool handler and runs one claimed execution
// attempt inside the request's tenant transaction: execution
// claim, entity lock, liveness re-check, handler, then terminal
// completion commit atomically. At most one concurrent attempt per
// invocation reaches the handler; the record is terminal if and
// only if business work committed.
func (e *Executor) Execute(ctx context.Context, req executor.Request) (json.RawMessage, error) {
	_, impl, err := e.registry.Resolve(req.MiniappID, req.Version)
	if err != nil {
		return nil, err
	}
	handler, ok := impl.ClassATools[req.Tool]
	if !ok {
		return nil, fmt.Errorf("classa: tool %q has no Class A handler", req.Tool)
	}
	// No legacy bypass: without invocation claim hooks the
	// executor refuses to run. The handler never executes.
	if req.Claim == nil || req.Claim.Recheck == nil || req.Claim.Complete == nil {
		return nil, fmt.Errorf("classa: execution claim hooks required")
	}
	var handlerErr error
	out, err := store.WithTenant(ctx, e.pool, req.TenantID, func(tx pgx.Tx) (json.RawMessage, error) {
		held, err := store.TryClaimExecution(ctx, tx, req.TenantID, req.InvocationID)
		if err != nil {
			return nil, err
		}
		if !held {
			return nil, executor.ErrExecutionInProgress
		}
		if req.EntityKey != "" {
			if err := store.LockEntity(ctx, tx, req.EntityKey); err != nil {
				return nil, err
			}
		}
		live, err := req.Claim.Recheck(tx)
		if err != nil {
			return nil, err
		}
		if !live {
			return nil, fmt.Errorf("%w: invocation terminal before execution", executor.ErrExecutionSuperseded)
		}
		if _, err := tx.Exec(ctx, `SAVEPOINT framework_handler_exec`); err != nil {
			return nil, err
		}
		result, herr := handler(ctx, registry.ToolCall{
			Context:      ctx,
			TenantID:     req.TenantID,
			Principal:    req.Principal,
			Input:        req.Input,
			Tx:           tx,
			InvocationID: req.InvocationID,
			EntityKey:    req.EntityKey,
			EffectToken:  req.EffectToken,
		})
		if herr != nil {
			// Business writes roll back to savepoint while the
			// FAILED outcome still records below and the claim
			// stays held: transaction advisory locks survive
			// savepoint rollback (verified on PG19). Never
			// commit partial handler writes.
			if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT framework_handler_exec`); err != nil {
				return nil, err
			}
			won, err := req.Claim.Complete(tx, contract.RunFailed, nil, herr.Error())
			if err != nil {
				return nil, err
			}
			if !won {
				return nil, fmt.Errorf("%w: completion lost to concurrent settlement", executor.ErrExecutionSuperseded)
			}
			// Committed failure: record the outcome through the
			// return channel (a nil error commits) and report
			// the handler error out-of-band.
			handlerErr = herr
			return nil, nil
		}
		won, err := req.Claim.Complete(tx, contract.RunSucceeded, result, "")
		if err != nil {
			return nil, err
		}
		if !won {
			return nil, fmt.Errorf("%w: completion lost to concurrent settlement", executor.ErrExecutionSuperseded)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	if handlerErr != nil {
		return nil, &executor.HandlerFailedError{Err: handlerErr}
	}
	return out, nil
}
