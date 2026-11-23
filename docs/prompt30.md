**Gate 2 accepted. Stage 3A is released.**

I accept the independent verdict: **CLEAR WITH SCOPED EXCEPTIONS**.

This is a meaningful milestone. The report independently establishes the important authorization, concurrency, crash-recovery, Class A recovery, revocation, Settle, and provider-boundary guarantees. There are no outstanding CRITICAL, HIGH, or MEDIUM findings.

The Prompt28.1 Settle implementation is also accepted as an authorized supersession of the earlier STOP. Its service boundary, CLI integration, and evidence are verified. We should not reopen that decision.

adv_review_prompt29.md

## **1. My disposition of the five findings**

| Finding | My decision                     | Required action                                                                  |
| ------- | ------------------------------- | -------------------------------------------------------------------------------- |
| P29-001 | Close through stronger evidence | Add persisted-state and audit assertions to the six policy-refusal legs.         |
| P29-002 | Close through stronger evidence | Add a genuine planted-violation negative-control test to the AST guard.          |
| P29-003 | Close through stronger evidence | Strengthen trust-field name detection and test adversarial variants.             |
| P29-004 | Acknowledged                    | Preserve the historical closeout; Prompt28.1 supersession is already documented. |
| P29-005 | Close through documentation     | Document the abandoned-PENDING boundary and expected-hash sourcing.              |

These are bounded improvements, not reasons to reopen Gate 2.&#x20;

adv_review_prompt29.md



## **2. Next instruction: Prompt30**

I recommend a short, strictly bounded residual-closure workstream. Do not let five LOW findings become another large architectural project.

# Prompt30 — Gate 2 Residual Closure and Operational Readiness

## Mission

Close the five LOW findings from `docs/adv_review_prompt29.md`, execute the documented revocation drill if a suitable isolated multi-replica environment is available, and preserve all existing Stage 3A guarantees.

This is a bounded evidence-hardening and documentation workstream.

Stage 3A is already released. Do not reopen its architecture or redesign working components merely to address these findings.

## 1. Governing decisions

- Prompt29 verdict CLEAR WITH SCOPED EXCEPTIONS is accepted.
- Prompt28 Stage 3A is released.
- Prompt28.1 Option A is approved and implemented.
- `runtime.Service.SettleEffect` remains the single shared Settle authorization boundary.
- No second Settle authority vocabulary or independent adapter authorization is permitted.
- Security and Observability remain co-equal critical foundational priorities.
- Developer Experience and Enforcement Architecture remains a foundational requirement.
- No Phase 3 work or Class B/C implementation is included here.

## 2. P29-001 — Complete authorization refusal evidence

Target: `effects/p28_matrix_policy_test.go`.

Inspect the six policy-refusal legs identified by Prompt29.

Add assertions establishing:

- Effect remains PENDING.
- No unauthorized business-state mutation occurs.
- Exactly one truthful `effect-refused` audit is recorded, where prescribed by the existing contract.
- Provider sends remain zero.
- The typed refusal is correct.

Do not weaken or replace existing assertions.

If a particular policy path legitimately has different audit semantics, inspect the governing contract and correct the evidence-register claim rather than manufacturing an audit expectation.

The objective is to eliminate the gap between test coverage and the five-point evidence claims.

## 3. P29-002 — Add a planted AST-guard negative control

Target: `effects/p28_bypass_guard_test.go`.

Use the established P24-009 planted-control pattern.

Create a controlled invalid source fixture containing:

- A prohibited provider-send implementation.
- A prohibited call site.

Run the actual guard against the fixture and assert that it fails for the expected reasons.

Also retain the positive control proving legitimate production code passes.

The test must detect a broken or weakened matcher, not merely assert that expected production symbols exist.

Do not introduce production bypasses or modify the canonical production send site.

## 4. P29-003 — Strengthen the trust-name guard

Target: `effects/p28_matrix_policy_test.go` and the relevant Intent structural guard.

Current enforcement recognizes exact names such as `Trust`, `TrustMode`, and `RequestedTrust`.

Strengthen it so that plausible future trust-bearing names cannot evade detection.

At minimum, test:

- `TrustLevel`
- `RequestedTrustMode`
- `trustMode`
- `TrustPolicy`
- A deliberately adversarial trust-related field name.

Choose the smallest durable approach: normalized, case-insensitive trust-token detection or an explicit documented name registry.

Do not create false positives for ordinary unrelated fields.

Keep the current Intent API unchanged. This is a structural regression guard, not permission to add trust fields.

## 5. P29-004 — Verify decision supersession record

No implementation change is authorized or required unless inspection discovers that the existing record is incomplete.

Verify:

- Prompt28's original STOPPED statement remains historically intact.
- Prompt28.1 Option A approval is recorded.
- The implementation authorization and GO are traceable.
- The memo addendum and evidence-register rows establish the supersession chain.
- Future documentation identifies `SettleEffect` as the single authoritative Settle boundary.

Do not rewrite historical closeout verdicts.

## 6. P29-005 — Document the abandoned-PENDING boundary

Update `docs/development/effect-authoring.md` and the appropriate contract pointer.

Document the precise failure scenario:

1. A Class A invocation creates an effect in PENDING.
2. The process terminates during the claim-to-authorization window.
3. The owning invocation is abandoned.
4. A different invocation retries using a new BusinessOpID.
5. Cross-invocation adoption is refused because the pending effect remains bound to its original invocation.
6. The operation can remain blocked until same-invocation recovery or appropriate operator intervention.

Explain:

- Why this is an approved, fail-closed design trade-off.
- Why a new BusinessOpID must not silently adopt the old pending effect.
- What recovery path is supported.
- What limitations developers must understand.

Do not claim this wedge has been execution-tested. Prompt29 explicitly classifies it as analyzed but not executed.

Also add one line to `docs/phase1/classd-operations.md` explaining how operators obtain the expected grant hash from the startup log or live `/healthz` response before running `verify-revocation`.

Do not change the documented ten-minute revocation-staleness semantics.

## 7. Operational revocation drill

The Prompt29 review recommends executing the revocation procedure at least once against actual running replicas before operational reliance.

First inspect whether an isolated, controlled multi-replica environment exists.

If it does:

- Use disposable or explicitly authorized test credentials.
- Record the initial grant hash.
- Revoke a test grant.
- Observe the documented propagation behavior.
- Execute `miniapp auth verify-revocation`.
- Verify correct stale, mismatched, and successful outcomes.
- Record replica identity, timestamps, expected and observed hashes, and exit codes.
- Verify no credentials or sensitive information enter logs or evidence.

If no suitable environment exists, report BLOCKED/NOT EXECUTED with the precise missing prerequisite.

Do not create a deployment, touch production credentials, restart production instances, or alter production grants without explicit owner authorization.

A simulated unit test is not a substitute for this operational drill.

## 8. Validation

For code changes:

- Run the new targeted tests.
- Run the relevant existing runtime, effects, and transport tests.
- Run the full serial suite.
- Run relevant race tests.
- Run gofmt, go vet, and scripts/checks.sh.
- Use the required immutable PostgreSQL 19 digest.
- Use disposable databases only.

Verify that:

- Existing Settle crash-test source remains byte-identical.
- The `effects.Settle` implementation remains untouched.
- No advisory-lock paths or lock-order semantics change.
- No authorization, effect, audit, or recovery guarantees are weakened.

Classify new tests as green-first unless a genuine identical-byte red/green experiment demonstrates a defect and its correction.

## 9. Evidence and deliverables

Create:

`docs/phase2/prompt30-residual-closeout.md`

Update:

- `docs/evidence/README.md`
- Relevant test and developer documentation.
- Relevant contract references, without rewriting historical decisions.

The report must contain:

- Baseline HEAD and worktree state.
- P29-001 through P29-005 dispositions.
- Exact tests and results.
- Digest and PostgreSQL verification.
- Operational revocation drill results or explicit blocker.
- Residuals that remain.
- Confirmation that Stage 3A guarantees remain unchanged.

## 10. Explicit prohibitions

- No new architecture or execution classes.
- No Security Contract implementation.
- No Observability Contract implementation.
- No Developer Experience architecture implementation.
- No Class B/C adapters.
- No generic RBAC.
- No migrations or dependency changes.
- No real provider writes.
- No REST/MCP Settle endpoint.
- No Phase 3 work.
- No commits, pushes, deployments, or staging without explicit authorization.
- Do not overwrite outside author activity.

## Final instruction

Solve the underlying evidence and coverage gaps rather than patching report wording alone.

Keep the work tightly scoped. Stop once all five findings have an evidence-backed disposition and the operational drill has either been genuinely completed or honestly marked blocked.

The next major workstream after this closeout is the coordinated Miniapp Security, Observability, and Developer Experience and Enforcement Architecture specification.

## **3. What follows Prompt30**

This is where I want us to shift the center of gravity.

We have spent considerable effort proving Miniapp's execution and effect-safety foundations. The next major architectural investment should be making those guarantees automatic for application developers.

## **Next major architectural program**

**Security Contract**

Authentication, authorization, delegation, tenant isolation, identity propagation, and credential protection.

**Observability Contract**

Audit, structured logs, metrics, distributed tracing, correlation, export, and operational diagnostics.

**Developer Experience and Enforcement Architecture**

Make security and observability automatic, mandatory, testable, and invisible to ordinary business-logic development.

These must be designed as one coordinated architectural program, not three unrelated feature projects.

There are also three constraints from Prompt29 that must remain load-bearing:

- No sequence workload until all four M5 activation conditions are satisfied.
- No real provider until the P24-008 capability-membership requirement is implemented.
- Every future Settle adapter must use `SettleEffect`.&#x20;

  adv_review_prompt29.md



**My direction:** Complete Prompt30 as a short cleanup, then move immediately to the coordinated foundational architecture specification. We should not begin Class B/C implementation before that specification is agreed.

This is the transition from proving that Miniapp is reliable to ensuring that developers can use it correctly without having to understand all its internal complexity.