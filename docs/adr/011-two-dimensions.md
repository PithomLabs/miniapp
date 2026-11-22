# ADR-011: Two-Dimension Classification

Status: Adopted.

## Context

Execution semantics (how a process recovers) and compute placement
(where code runs under what isolation) answer different questions.
One mutually exclusive selector would forbid valid hybrids such as a
durable workflow orchestrating an isolated task.

## Decision

Dimension A (execution class A/B/C/D) and Dimension B (placement
machine/sprite/hybrid/local) are decided independently. Admission
checks enforce each against its own policy: ADR-001 for execution,
the placement table for compute.

## Consequences

- Hybrid DBOS-plus-Sprite flows are first-class, with Neon holding
  authoritative state.
- Reviews must evaluate both dimensions; a pass on one never implies
  the other.
