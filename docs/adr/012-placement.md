# ADR-012: Compute-Placement Policy

Status: Adopted.

## Context

"Agent-invoked" does not imply "isolated": placing every tool call in
a Sprite would destroy latency, cost, and auditability.

## Decision

Placement follows the trust profile of the code, never the identity of
the caller. Trusted production code runs on Machines; agent-authored,
generated, or untrusted code runs in isolated Sprites; durable
workflows may orchestrate bounded Sprite tasks (hybrid). Persistent
Sprites are never business-workflow persistence.

## Consequences

- The §10.2 table is enforced by manifest admission; exceptions need
  written justification.
- Sprite checkpoints restore filesystems only.
