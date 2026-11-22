# ADR-015: Hybrid DBOS-plus-Sprite Execution Pattern

Status: Adopted.

## Context

Durable workflows sometimes need isolated execution for bounded
untrusted work without surrendering durability, audit, or credential
boundaries.

## Decision

The DBOS workflow stays authoritative on a Machine; the Sprite runs
the bounded task; Neon holds task identity and results; no
unrestricted credentials enter the Sprite. Provisioning is idempotent
on the task ID with a durable registry, lease, and reaper; policy is
confirmed before execution; failures map to typed outcomes the
workflow retries (new task ID), compensates, or escalates.

## Consequences

- A lost Sprite never strands the workflow; an orphaned Sprite is
  reaped, never silently billed.
- The pattern is proven once by `hybrid-task-demo`, not per miniapp.
