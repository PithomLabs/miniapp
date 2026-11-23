GO — approved to start

Your approach is architecturally sound. It closes the owner-approved Settle authority gap with a minimal shared enforcement boundary, without introducing generic RBAC or modifying the underlying atomic Settle implementation.

I approve the work order with the following **mandatory implementation guardrails**.

## **1. Authorization boundary**

The canonical role is:

`miniapp.settle.operator`

Keep it unexported in `runtime`. Require an exact, case-sensitive match against the roles in the authenticated server-resolved binding.

The caller must never be able to supply or override the role, operator identity, or tenant.

The implementation must derive:

- Operator identity from the authenticated binding's trusted identity field.
- Tenant from the authenticated binding.
- Effect identity and authoritative effect metadata from the persisted effect record.
- Decision outcome from the validated request, then persist it atomically with the effect transition.
- Note from the request, as operator-provided evidence.

Do not use the credential itself as operator identity.

**Important:** The existing API-key configuration remains demo scaffolding. Completing this work does not establish production-grade identity provisioning.

## **2. Wrong-tenant protection**

This is the most important implementation detail I want reinforced.

Do not perform a separate effect-existence lookup and then call `effects.Settle`. That introduces an unnecessary race and risks leaking whether an effect exists.

The tenant restriction must be enforced by the authoritative database operation itself, using the tenant derived from the binding.

Map both nonexistent and foreign-tenant effects to the same terminal `unknown-effect` error. Do not expose underlying no-row errors.

Add an explicit nonexistent-effect test alongside the wrong-tenant test. Both must prove indistinguishable external behavior and no state or audit mutation.

## **3. Strengthen the validation matrix**

Add these service tests if they are not already covered:

| Additional case                            | Required result                                     |
| ------------------------------------------ | --------------------------------------------------- |
| Nonexistent effect                         | Same denial as wrong tenant                         |
| Invalid outcome                            | Terminal invalid-request                            |
| Empty or whitespace-only note              | Terminal invalid-request                            |
| Role with case variation                   | Refused                                             |
| Role with leading/trailing whitespace      | Refused                                             |
| `tenant-ops` or role substring             | Refused                                             |
| Caller attempts to supply identity or role | Structurally impossible through the method contract |

For every refusal, inspect persisted effect state and audit rows, not just the returned error.

## **4. CLI security**

Implement the exact CLI hierarchy and flag names specified by F-03. Do not invent a second command syntax.

Mirror existing credential handling, but never print, log, or include credentials in errors. Be aware that passing a credential directly as a command-line argument can expose it through process listings; document that limitation rather than silently treating it as secure credential storage. Carry its proper resolution into the Security Contract.

## **5. Preserve existing guarantees**

The following are non-negotiable:

- `effects.Settle` remains untouched.
- Existing Settle crash-test source remains unchanged.
- Atomicity, idempotency, and conflict behavior remain unchanged.
- No migration, dependency change, generic RBAC, REST/MCP endpoint, or Phase 3 work.
- No historical closeout verdict rewriting.
- The decision memo gets an addendum; the evidence register gets new rows.
- New coverage is classified green-first, not incorrectly called red/green.

For the crash tests, verify their source is byte-for-byte unchanged and execute them again.

## **6. Completion criteria**

The work is complete only when:

1. `runtime.Service.SettleEffect` is the single shared authorization boundary.
2. CLI invokes that method and contains no independent authorization logic.
3. Authorized and refused cases pass, including tenant isolation and spoofing.
4. Audit attribution is verified against actual persisted records.
5. Existing crash-safety and concurrency tests remain green.
6. Full regression, race, formatting, vet, and checks pass on disposable digest-pinned PostgreSQL 19.
7. The contract, decision memo, and evidence register accurately record the implementation.

**Final direction: Proceed with implementation.**

Do not stop merely because a test needs refinement. Investigate the underlying behavior and fix defects within the authorized scope. However, if the implementation cannot obtain a trustworthy server-derived operator identity or enforce tenant isolation without changing the existing guarantees, stop and report the precise blocker.

The goal is not simply to expose a `settle` command. It is to establish a reusable, trustworthy administrative boundary that future API, MCP, and CLI adapters can share.