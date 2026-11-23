package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/internal/store"
)

// Event is one attributable audit record. Details must never contain
// credentials; callers pass redacted payloads only.
type Event struct {
	ID            int64
	At            time.Time
	TenantID      string
	AgentID       string
	AuthorityKind string
	AuthorityID   string
	MiniappID     string
	Version       string
	Tool          string
	InvocationID  string
	RunID         string
	Action        string
	Outcome       string
	Details       json.RawMessage
}

// Service appends and queries audit events.
type Service struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// New returns a service. A nil logger disables structured log output
// (durable writes always happen).
func New(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{pool: pool, logger: logger}
}

// Record appends one event. Tenant, agent, authority, miniapp, tool,
// action, and outcome are required; anything less is rejected rather
// than recorded half-attributed.
func (s *Service) Record(ctx context.Context, e Event) error {
	if err := s.InsertTx(ctx, nil, e); err != nil {
		return err
	}
	s.Log(e)
	return nil
}

// InsertTx validates and inserts one event using an explicit
// transaction when tx is non-nil, or its own tenant transaction
// when tx is nil. Multi-table atomic operations (approval parking)
// pass their transaction so the audit row commits or rolls back
// with the state it describes; they call Log after commit.
func (s *Service) InsertTx(ctx context.Context, tx pgx.Tx, e Event) error {
	if e.TenantID == "" || e.AgentID == "" || e.AuthorityKind == "" || e.AuthorityID == "" {
		return fmt.Errorf("audit: tenant, agent, and authority attribution are required")
	}
	if e.MiniappID == "" || e.Tool == "" || e.Action == "" || e.Outcome == "" {
		return fmt.Errorf("audit: miniapp, tool, action, and outcome are required")
	}
	details := e.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	const q = `INSERT INTO app.audit_events
		(tenant_id, agent_id, authority_kind, authority_id, miniapp_id, miniapp_version,
		 tool, invocation_id, run_id, action, outcome, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid, $9, $10, $11, $12)`
	args := []any{e.TenantID, e.AgentID, e.AuthorityKind, e.AuthorityID, e.MiniappID, e.Version,
		e.Tool, e.InvocationID, e.RunID, e.Action, e.Outcome, string(details)}
	if tx != nil {
		_, err := tx.Exec(ctx, q, args...)
		return err
	}
	_, err := store.WithTenant(ctx, s.pool, e.TenantID, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, q, args...)
		return true, err
	})
	return err
}

// Log emits the structured log line for a recorded event.
// InsertTx callers with an explicit transaction invoke it after a
// successful commit so logging never precedes durability.
func (s *Service) Log(e Event) {
	if s.logger == nil {
		return
	}
	s.logger.Info("audit", "tenant", e.TenantID, "agent", e.AgentID,
		"authority", e.AuthorityKind+"/"+e.AuthorityID, "miniapp", e.MiniappID,
		"version", e.Version, "tool", e.Tool, "action", e.Action, "outcome", e.Outcome)
}

// Query returns a tenant's events newest-first, bounded by limit.
func (s *Service) Query(ctx context.Context, tenantID string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("audit: limit must be 1..1000")
	}
	return store.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) ([]Event, error) {
		rows, err := tx.Query(ctx, `SELECT id, at, tenant_id, agent_id, authority_kind, authority_id,
			miniapp_id, miniapp_version, tool, COALESCE(invocation_id::text, ''), run_id,
			action, outcome, COALESCE(details::text, '{}')
			FROM app.audit_events WHERE tenant_id = $1 ORDER BY id DESC LIMIT $2`, tenantID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []Event
		for rows.Next() {
			var e Event
			var details string
			if err := rows.Scan(&e.ID, &e.At, &e.TenantID, &e.AgentID, &e.AuthorityKind,
				&e.AuthorityID, &e.MiniappID, &e.Version, &e.Tool, &e.InvocationID,
				&e.RunID, &e.Action, &e.Outcome, &details); err != nil {
				return nil, err
			}
			e.Details = json.RawMessage(details)
			out = append(out, e)
		}
		return out, rows.Err()
	})
}
