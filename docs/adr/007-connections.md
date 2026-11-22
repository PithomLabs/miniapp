# ADR-007: Neon Connection Strategy

Status: Adopted (pending Phase 0 spike confirmation of pool-handle behavior).

## Context

DBOS uses PostgreSQL LISTEN/NOTIFY, which transaction-mode poolers do
not support. A uniform pooled connection strategy would break the DBOS
system database.

## Decision

DBOS uses a direct Neon connection with a bounded pool. Application and
River connection modes are decided per workload after compatibility
testing. Migrations use direct connections in one gate. Connection
budgets are computed across Fly Machines and horizontal scaling.
Tenant RLS context is transaction-local only, set inside the same
transaction as tenant queries; session state is never used through the
transaction pooler.

## Consequences

- The Phase 0 spike must prove handle-level behavior, including whether
  the pinned DBOS SDK tolerates pooled endpoints (C5).
- Pooled endpoints are reserved for workloads proven to tolerate them.
