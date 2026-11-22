# ADR-008: One Active Executor per DBOS Application/Family

Status: Adopted.

## Context

Distributed DBOS recovery needs explicit coordination (Conductor);
assuming it for free would hide licensing and topology cost. A single
global executor would also couple every workflow family's availability
and releases.

## Decision

Run one active executor per DBOS application/workflow family in the
initial topology, always available for waits. No scale-to-zero until
the wake test passes. Add standby capacity, multiple executors, or
Conductor only on measured need. One executor per family is the
baseline; it is explicitly not an HA claim.

## Consequences

- Restart, health-check replacement, graceful drain, executor-down
  alerting, and stuck-workflow detection are owned per family.
- Capacity and Conductor decisions are measured per family, never
  assumed platform-wide.
