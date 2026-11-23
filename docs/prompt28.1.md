# Owner Decision — Settle Operator Authority

**Decision:** OPTION A — Approved

**Status:** Owner-approved architectural decision; implementation remains pending.

## 1. Canonical role

Approved role identifier:

`miniapp.settle.operator`

This is the dedicated authority required to execute the Settle control operation.

The role must be checked server-side against the authenticated `auth.Binding.Roles`. Matching is exact and case-sensitive.

## 2. Membership and assignment

The role is assigned explicitly to designated operator principals within a tenant.

Assignment must originate from trusted, server-controlled identity or credential provisioning.

The following are not sufficient to establish operator authority:

- Caller-supplied role names or identity claims.
- Tenant ownership alone.
- The generic `user`, `service`, or `trigger` authority kind.
- Approval authority or approver status.
- The synthetic `tenant-ops` role.
- An agent's assertion that it represents an administrator.

No caller may grant itself operator authority.

The tenant associated with the authenticated binding must match the tenant owning the effect being settled.

## 3. Enforcement requirements

The Settle operation must:

1. Require an authenticated binding.
2. Require the exact `miniapp.settle.operator` role.
3. Verify tenant ownership of the target effect.
4. Derive operator identity from the trusted binding, never from caller-supplied attribution.
5. Deny by default when identity, role, or tenant verification fails.
6. Preserve the existing atomic Settle state-transition and audit guarantees.
7. Record truthful attribution of the operator, tenant, effect, and decision.
8. Use the same enforcement boundary regardless of whether Settle is eventually invoked through an API, MCP, CLI, or internal service interface.

The operator role grants Settle authority only. It does not automatically grant unrelated administrative privileges, approval authority, or general effect-execution permissions.

## 4. Production identity boundary

The current API-key configuration is demonstration scaffolding, not a production identity-management system.

It may be used for isolated development and testing, but production operator membership must come from an explicitly trusted provisioning mechanism.

Do not represent demo API-key configuration as production-grade authentication.

The future Miniapp Security Contract must formalize role issuance, membership changes, revocation, tenant scope, and authenticated identity propagation.

## 5. Scope and restrictions

This decision:

- Closes OSI-2 for the specific Settle operator role vocabulary and authority requirement.
- Does not close the broader Miniapp identity and authorization architecture.
- Does not authorize a generic RBAC system.
- Does not authorize new migrations or dependencies.
- Does not weaken existing effect, audit, locking, or recovery guarantees.
- Does not authorize unrelated Phase 3 work.

The implementation must be a minimal, durable enforcement of this decision, not a workaround.

## 6. Required evidence

Before considering Settle operator wiring complete, independently verify:

- Authorized operator succeeds.
- Non-operator is refused.
- Missing or malformed identity is refused.
- Wrong-tenant operator is refused.
- Caller-supplied identity or role spoofing cannot grant authority.
- Audit attribution comes from the authenticated binding.
- Failed authorization causes no Settle state transition.
- Concurrent and repeated Settle operations preserve existing atomicity and idempotency.
- Existing crash-safety tests remain green.

**Owner disposition:** Option A is approved. The implementation must follow this decision without weakening the existing security boundary.