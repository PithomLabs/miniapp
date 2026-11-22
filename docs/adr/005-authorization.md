# ADR-005: Two-Principal Authorization with Manifest-Bound Trust Modes

Status: Adopted.

## Context

Agent identity alone cannot authorize business operations: it would
either block legitimate unattended automation or grant agents implicit
administrator power.

## Decision

Every invocation carries an agent principal (which harness client) and
an authority principal (delegated user, approved tenant service
identity, or authorized trigger). Trust modes (`dry-run`, `confirm`,
`autonomous`) bind per miniapp version per tenant; the calling agent
cannot weaken them. Every consequential action passes a server-side
policy check. MCP annotations and elicitation are interaction aids,
never enforcement. Approval is an explicit policy gate overlaid on the
execution classes, not a new execution class: single-step tools use the
durable approval table, larger processes wait on DBOS events against the
same table.

## Consequences

- Handlers fail closed when authority cannot be resolved.
- Unattended workflows need explicit tenant/service grants with
  bounded permissions.
