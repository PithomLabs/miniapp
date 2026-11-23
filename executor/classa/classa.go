package classa

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

// Execute resolves the tool handler and runs it inside the request's
// tenant transaction. Handler errors abort the transaction; nothing
// partial commits.
func (e *Executor) Execute(ctx context.Context, req executor.Request) (json.RawMessage, error) {
	_, impl, err := e.registry.Resolve(req.MiniappID, req.Version)
	if err != nil {
		return nil, err
	}
	handler, ok := impl.ClassATools[req.Tool]
	if !ok {
		return nil, fmt.Errorf("classa: tool %q has no Class A handler", req.Tool)
	}
	return store.WithTenant(ctx, e.pool, req.TenantID, func(tx pgx.Tx) (json.RawMessage, error) {
		if req.EntityKey != "" {
			if err := store.LockEntity(ctx, tx, req.EntityKey); err != nil {
				return nil, err
			}
		}
		return handler(ctx, registry.ToolCall{
			Context:      ctx,
			TenantID:     req.TenantID,
			Principal:    req.Principal,
			Input:        req.Input,
			Tx:           tx,
			InvocationID: req.InvocationID,
			EntityKey:    req.EntityKey,
			EffectToken:  req.EffectToken,
		})
	})
}
