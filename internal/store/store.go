package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolMode selects the Neon connection mode for one workload. DBOS
// requires direct connections (it uses LISTEN/NOTIFY); other workloads
// choose per Phase 0 compatibility evidence.
type PoolMode string

const (
	// PoolDirect opens direct connections (required for DBOS).
	PoolDirect PoolMode = "direct"
	// PoolPooled uses the pooled endpoint where semantics allow.
	PoolPooled PoolMode = "pooled"
)

// PoolConfig is the frozen connection configuration shape. Secrets stay
// in DSN only in memory; String redacts them for logs.
type PoolConfig struct {
	DSN      string
	Mode     PoolMode
	MaxConns int32
}

// String renders the config with credentials redacted for logs. URL
// and keyword DSN forms are sanitized format-aware; anything that
// cannot be parsed safely redacts entirely rather than leaking.
func (c PoolConfig) String() string {
	return fmt.Sprintf("mode=%s maxConns=%d dsn=%s", c.Mode, c.MaxConns, redactDSN(c.DSN))
}

// TenantContextSQL returns the statement arming the RLS tenant context
// for the current transaction. It is only meaningful inside an
// explicit transaction; in autocommit each statement is its own
// transaction and the setting is discarded immediately. Prefer
// WithTenant, the only approved entry point for tenant transactions.
// LockEntity serializes the current transaction on a tenant-scoped
// entity key. The lock is transaction-scoped (released at commit or
// rollback), so it is safe for single-transaction Class A execution
// and can never be held across a durable wait.
func LockEntity(ctx context.Context, tx pgx.Tx, entityKey string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, entityKey)
	return err
}

func TenantContextSQL() string {
	return `SELECT set_config('app.tenant_id', $1, true)`
}

// WithTenant runs fn inside an explicit transaction with the RLS
// tenant context armed first. The context is transaction-local, so it
// cannot leak across pooled checkouts. RLS remains defense in depth
// behind mandatory application authorization.
func WithTenant[T any](ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(tx pgx.Tx) (T, error)) (T, error) {
	var zero T
	if tenantID == "" {
		return zero, fmt.Errorf("store: tenant ID must not be empty")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback(ctx) // no-op after successful commit
	if _, err := tx.Exec(ctx, TenantContextSQL(), tenantID); err != nil {
		return zero, err
	}
	out, err := fn(tx)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}
	return out, nil
}
