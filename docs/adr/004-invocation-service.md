# ADR-004: One Canonical Invocation Service

Status: Adopted.

## Context

Separate MCP, REST, and CLI execution paths would triple the
authorization, audit, and recovery surface and guarantee drift.

## Decision

All invocations flow through one canonical invocation service that owns
validation, authorization-before-dispatch, run handles, audit, and
approvals. The MCP router, REST/OpenAPI handlers, and CLI are thin
adapters with no business logic and no separate authorization rules.

## Consequences

- A behavior proven over one transport holds for all three.
- Transport adapters are independently replaceable; the service is the
  compatibility boundary.
