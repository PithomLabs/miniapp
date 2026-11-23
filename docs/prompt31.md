# Prompt31 — Adversarial Code Review: Prompt28.1 Settle Operator Boundary

## Mission

Act as an independent principal Go engineer and security reviewer. Adversarially review the completed Prompt28.1 implementation of the Settle operator authority.

Do not trust the implementation report. Inspect the actual source, tests, diffs, contracts, and evidence independently.

**Review only. Do not modify production code or fix findings.**

## Review scope

Inspect:

- `runtime/lifecycle.go` — `Service.SettleEffect`
- `transport/cli/cli.go` — settle command and flags
- `cmd/miniapp/main.go` — command dispatch
- `runtime/p281_settleop_test.go`
- `transport/cli/settle_test.go`
- Contract §16, F-03, operator decision memo addendum, and evidence-register entries.

## Critical invariants to challenge

1. **Authentication:** No unauthenticated, empty, malformed, or unknown credential can settle an effect.
2. **Exact authority:** Only the exact, case-sensitive `miniapp.settle.operator` role from a server-resolved `auth.Binding.Roles` can authorize Settle. No caller-supplied role, identity, or tenant can confer authority.
3. **Tenant isolation:** Tenant comes exclusively from the authenticated binding and is enforced in the authoritative database operation. No pre-read race or cross-tenant access.
4. **Non-oracle behavior:** Nonexistent and foreign-tenant effects produce indistinguishable terminal denials without leaking effect existence.
5. **Validation:** Invalid outcomes, missing/blank notes, and malformed requests fail closed without state transitions.
6. **Attribution:** Operator, tenant, effect, and decision attribution are derived from trusted server-side values. Inspect actual persisted audit records.
7. **Atomicity:** `effects.Settle` retains its original atomicity, idempotency, conflict, and crash-safety semantics. Verify its implementation is untouched.
8. **Shared boundary:** The CLI delegates to `SettleEffect` and contains no independent authorization or attribution logic. Check for alternative production paths that bypass the service method.

## Adversarial test assessment

Inspect whether tests genuinely prove:

- Authorized success and binding-derived attribution.
- All nine refusal legs, including case, whitespace, substring, `tenant-ops`, and smuggled-field attempts.
- Wrong-tenant and nonexistent-effect indistinguishability.
- Zero state mutation and truthful audit behavior on every refusal.
- Outcome/note validation.
- Idempotent repeat and conflicting decisions.
- CLI flag validation, misuse handling, and authorized end-to-end execution.

Look for vacuous assertions, shared-fixture assumptions, missing negative controls, caller-controlled identity, TOCTOU gaps, and tests that verify error returns without inspecting persisted database state.

## Regression and scope verification

Verify the reported PostgreSQL 19 digest, test results, and evidence where reproducible.

Confirm:

- Existing Settle crash-test sources are byte-identical.
- No unauthorized changes to `effects.Settle`.
- No migration, dependency change, generic RBAC, REST/MCP endpoint, or unrelated production modification.
- No commits, deployments, or real provider writes.
- The original STOPPED record remains intact and the Prompt28.1 owner-decision supersession is accurately documented.

## Findings and verdict

Classify findings as CRITICAL, HIGH, MEDIUM, or LOW. For every finding provide the affected invariant, exact source references, concrete failure scenario, evidence, and whether it blocks acceptance.

Use one verdict:

- **CLEAR** — all material guarantees independently verified.
- **CLEAR WITH SCOPED EXCEPTIONS** — only bounded, non-blocking findings remain.
- **NOT CLEAR** — a material security, authorization, tenant-isolation, attribution, or correctness guarantee is violated or unproven.

Do not downgrade a security defect because tests are green. Do not invent findings or demand unrelated redesign.

## Deliverable

Write:

`docs/adv_review_prompt31.md`

Include the review baseline, evidence inspected, findings, test results, invariant-by-invariant assessment, residuals, and final verdict.

Stop after the report. Do not implement fixes, expand scope, or begin Phase 3.

**Final standard:** Prove that an unauthorized caller cannot settle an effect, a caller cannot cross tenant boundaries, attribution cannot be forged, and every authorized settlement preserves the existing atomic effect guarantees.