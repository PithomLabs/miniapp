# ADR-006: Honest External-Effect Semantics

Status: Adopted.

## Context

Durable execution replays steps across the provider-complete /
checkpoint-missing crash window, so unconditional exactly-once effects
are unachievable at external providers.

## Decision

The framework guarantees no duplicate *confirmed* business effects, no
silently lost operations, and no blind retries of unknown outcomes.
Effects follow PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN with
reconciliation before any retry or compensation of UNKNOWN outcomes.
Saga compensations are explicit, idempotent, LIFO, and never auto-fire
on UNKNOWN outcomes without a safety basis.

## Consequences

- Acceptance language uses the corrected target everywhere; "zero
  duplicate effects" appears nowhere as an unconditional claim.
- Providers without idempotency or status lookup get conservative
  quarantine handling.
