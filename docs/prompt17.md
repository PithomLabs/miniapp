**WS-B work order issued.** You are authorized to begin implementation, subject to the baseline verification gate below.

The workstream has five ordered priorities. M-01 is first, not because it blocks WS-B authorization, but because we must not carry known lifecycle-state corruption into the effect-authorization implementation.

# WS-B Work Order — Effect Authorization and Lifecycle Integrity

**Project:** `github.com/PithomLabs/miniapp`\
**Workstream:** WS-B\
**Status:** AUTHORIZED TO BEGIN\
**Governing architecture:** `docs/plan4.2.md`\
**Governing implementation prompt:** `docs/prompt15.2.md`\
**Primary contract:** `docs/phase1/invocation-lifecycle-contract.md`

## 1. Mission

Implement and prove the framework's effect-authorization boundary.

The framework must ensure that a workflow cannot perform a consequential external operation merely because it was admitted, submitted, resumed, or signaled.

Before an effect can transition from `PENDING` to `ATTEMPTING`, the framework must atomically revalidate the authorization and lifecycle conditions specified by the accepted contract.

Also close the lifecycle-mutation defect M-01 identified in the independent Prompt 16 review.

The governing principle is unchanged:

**The framework owns reliability; miniapps own business logic.**

Miniapp authors must not have to independently implement approval revalidation, cancellation races, effect fencing, authorization snapshots, or recovery mechanisms.

## 2. Authoritative baseline and preflight

The accepted WS-A state is deliberately distinguished from the earlier submitted and review-execution states.

The review records:

- Submitted green baseline: `f0b7dfa6…`
- Review-execution file set: `e20a4689…`
- Review report delivery state: `a547c0c1…`
- Acceptance state: the `e20a4689…` file set plus the report at `ec981838…`, with the documented recomputation procedure.

Before touching source code:

1. Read `docs/adv_review_prompt16.md`, including its §6 owner-decision addendum.
2. Resolve and record the full hashes and exact recomputation commands from the authoritative document.
3. Verify the accepted file set, report identity, repository revision, and working-tree state.
4. Record Go, DBOS SDK, PostgreSQL, and test-environment versions.
5. Confirm the WS-A review artifacts and evidence register are available.
6. Create an isolated implementation working copy or branch without modifying the frozen acceptance baseline.

Do not substitute the submitted green digest for the acceptance state. Do not treat abbreviated hashes as full identity.

**Hard stop:** If the acceptance state cannot be established, stop and report the discrepancy. Do not silently choose a nearby revision.

Preserve a reproducible pre-implementation snapshot. It becomes the red baseline for newly introduced WS-B regression tests.

## 3. Governing decisions and non-negotiable boundaries

The following owner-approved distinction is normative:

> A gated workflow may be submitted with a valid pending approval linkage only to enter its durable approval-wait path. It may proceed to consequential business execution only after the framework verifies a persisted granted decision with matching authority and intent. The effect-authorization boundary independently revalidates that decision. Missing, denied, expired, cancelled, or mismatched approval linkages fail closed.

WS-A established admission and durable waiting. WS-B must establish consequential execution authorization.

Do not weaken this distinction.

Other binding requirements:

- DBOS remains the durable execution engine.
- Preserve the pinned DBOS Go SDK version and existing architecture.
- All adapters continue through the canonical invocation service.
- Authentication occurs before dispatch; tenant and authority/principal are server-derived.
- The agent principal and authority principal remain distinct.
- Shared PostgreSQL framework services remain authoritative for invocation, approval, authorization, audit, and effect lifecycle state.
- Do not claim exactly-once provider delivery.
- Do not hold database transactions or locks across provider calls, DBOS submissions, or durable waits.
- Effects remain mock-only in this phase. No real provider writes or customer credentials.
- Migrations remain forward-only. Use disposable databases only.
- No commits or pushes without explicit authorization.

Do not implement Class B or C, an effects reconciler, River integration, the `lead-followup` miniapp, or its draft ADR. Do not introduce unrelated architectural redesign.

## 4. Ordered implementation priorities

### Priority 1 — M-01: lifecycle mutation integrity

The Prompt 16 review identified unpredicated `store.update` remnants:

- `runtime/service.go`: `executeSync` terminal updates.
- `runtime/service.go`: `pollApproval` failed-state marking.
- `runtime/lifecycle.go`: `Status` synchronization update.

The review found that these writes can overwrite a concurrent cancellation with a success or failure status. Audit evidence remains, but the persisted final status can become misleading.

This is a contract violation. Its impact was assessed as bounded and non-blocking, but the finding retains its MEDIUM identifier for stability.

**Required work:**

1. Trace each mutation to its actual caller, state assumptions, and transaction boundary.
2. Establish the permitted source states for each transition from the lifecycle contract.
3. Replace unsafe unconditional updates with atomic, predicated transitions.
4. Check affected-row counts and handle lost transitions explicitly.
5. Ensure a stale status observation cannot overwrite an authoritative terminal state.
6. Preserve truthful audit attribution and error handling.
7. Avoid blanket predicates that accidentally block legitimate transitions.

**Required regression tests:**

- Cancellation wins before synchronous completion commits: completion cannot overwrite cancellation.
- Synchronous completion wins first: subsequent cancellation cannot overwrite the completed outcome.
- Cancellation races with approval denial/expiry settlement: the final state reflects the valid winning transition.
- A stale `Status` observation cannot overwrite a terminal cancellation.
- Repeated and losing transition attempts behave idempotently or return the documented conflict result.

Use deterministic synchronization at the actual database transition boundary. Sleeps or probabilistic timing alone are insufficient.

The tests must fail against the frozen pre-WS-B baseline for the intended defect and pass against the corrected implementation.

**Exit condition:** all M-01 sites have an explicit, contract-consistent transition rule, affected-row handling, and discriminating regression coverage. Any remaining exception must be written, justified, and tested.

### Priority 2 — effect-authorization transaction

Implement the canonical framework-owned boundary that authorizes an effect and transitions it from `PENDING` to `ATTEMPTING`.

The decision must happen in the required transaction, using the contract's complete lock protocol.

Revalidate, using authoritative persisted state and the approved grant snapshot:

1. Invocation identity, tenant, liveness, and cancellation state.
2. The persisted approval decision.
3. Approval tenant, invocation, tool, authority, and intent binding.
4. Whether the decision is a valid persisted grant, not merely a pending linkage or signal.
5. Applicable grants and authorization snapshot.
6. Effect token and effect eligibility.
7. Kill-switch state.
8. Generation and lease/fencing ownership.
9. Any additional conditions explicitly required by the lifecycle contract.

The approval check must verify the actual persisted decision and its matching authority and intent. Do not accept an in-memory approval result, a prior workflow admission check, or a signal as sufficient authorization.

**Transaction and lock requirements**

The lifecycle contract §4 is authoritative for lock order. (P20
MEDIUM-3 correction: the advisory-last line below is superseded
and retained struck for provenance — the Phase 2
`authorizeAttempt` transaction takes no advisory lock at all.)

`invocation → approval → sequence/generation → policy-control/grant snapshot → effect → ~~advisory lock~~` — REMOVED.

Preserve the normative lock order for the authorization transaction:

`invocation → approval → sequence/generation → policy-control/grant snapshot → effect`

For operations that actually require advisory locking (Class A
entity serialization and execution claims), preserve the §4
outermost-first rule: advisory locks are acquired before any
transactional row or table locks, in the fixed claim→entity order
at the single Class A site.

Inspect the actual code and contract for the precise lock acquisition and transaction boundaries. Do not improvise a different order or add locks without analyzing deadlock implications.

The transaction must revalidate the required state and perform the conditional `ATTEMPTING` transition atomically. Check affected-row results. Persist the required lifecycle/audit evidence within the proper transaction boundary.

The provider operation occurs only after the transaction commits. Never hold the authorization transaction open during the provider call.

**Grant snapshot discipline**

Preserve the accepted P2 grant-revocation model:

- Grants are loaded as configuration snapshots; do not silently introduce hot reload or DB-backed grants.
- Preserve the in-process pinned-snapshot locking requirement across authorization.
- Do not claim immediate cross-replica revocation.
- Preserve the accepted maximum-staleness and operational verification requirements.
- Verify the health hash, replica verification, rollback constraints, and operational documentation obligations recorded in the governing P2 decisions.
- If an accepted grant-coherence safeguard is missing, report the exact gap rather than silently weakening the contract or inventing a new revocation guarantee.

**Required tests:**

- Valid grant and matching approval: effect may cross the boundary once.
- Pending approval: cannot cross.
- Denied approval: cannot cross.
- Expired approval: cannot cross.
- Cancelled invocation or approval: cannot cross.
- Tenant, invocation, tool, authority, or intent mismatch: cannot cross.
- Grant revoked before authorization on a replica holding the updated snapshot: cannot cross.
- Stale generation or lost lease: cannot cross.
- Kill switch active: cannot cross.
- Competing cancellation and effect claim: exactly the documented winner semantics.
- Retry/replay cannot bypass revalidation or create a second logical attempt.

Tests must assert persisted effect state, invocation state, approval state, audit evidence, and actual mock-provider attempt records. An API response alone is insufficient.

### Priority 3 — O-01: close the submit-time TOCTOU at the effect boundary

WS-A proved that submission enters a durable wait path with a valid persisted linkage. It did not prove that the linkage remains valid when consequential work is attempted.

Build explicit tests for changes occurring after workflow submission but before effect authorization.

At minimum, exercise:

1. A valid pending linkage at submission followed by denial before the effect claim.
2. A linkage that expires before the effect claim.
3. Cancellation after submission but before the effect claim.
4. A grant change before authorization.
5. A binding mismatch discovered at the effect boundary.
6. A workflow replay or resumed execution after the authorization conditions have changed.

The expected invariant is:

**No invalid approval, invocation, authority, grant, lease, or effect state can cross the `ATTEMPTING` boundary.**

The test must demonstrate this at the actual authorization transaction, not merely through a workflow-side pre-check.

Do not move the security boundary into individual miniapp code. The framework must enforce it.

### Priority 4 — L-05: trust-mismatch retry classification

The review found a fail-closed trust-mismatch retry that propagates a bare `errApprovalRequired` through `dispatch` and `Invoke`.

Preserve the proven security behavior:

- No inappropriate workflow submission.
- No bypass of the persisted approval linkage.
- Retrying with the original confirm trust converges correctly.

Correct the error classification so adapters receive the documented, meaningful outcome rather than an unclassified internal error.

Add a regression test that asserts both the returned classification and the absence of an unauthorized submission.

Do not fix this by weakening the trust or approval checks.

### Priority 5 — L-06: retry-time denial audit

The review found that a retry-time authorization denial in `dispatchPending` returns `forbidden` without the audit event emitted by the initial authorization-denial path.

Make the audit behavior consistent.

Add a regression test covering grant revocation between initial admission and retry. Assert:

- The caller receives the correct denial.
- No workflow or effect is improperly submitted.
- The denial is attributed to the correct tenant and principal.
- The appropriate audit record is committed exactly according to the documented audit semantics.
- No secrets or sensitive authorization material are placed in the audit event.

## 5. Effect lifecycle semantics must remain truthful

Preserve the contract's distinction between reservation, authorization, and external execution.

- `PENDING` is a provisional reservation, not a provider attempt.
- `ATTEMPTING` is the durable logical point of no return for the effect claim.
- If cancellation wins while an effect remains `PENDING`, withdraw it to terminal `CANCELLED`, with no provider attempt.
- If the `ATTEMPTING` transition wins first, the effect is in flight. Cancellation cannot recall an external operation already in progress.
- Do not claim exactly-once external delivery.
- An ambiguous provider outcome remains `UNKNOWN`; do not blindly retry it.
- `NOT_APPLIED` may be retried only through the documented authorization path.
- Settlement must be based on evidence and follow the existing contract.

Do not introduce a background effects reconciler in WS-B. Do not invent provider semantics.

## 6. Root-cause and implementation discipline

For every defect:

1. Identify the violated contract invariant.
2. Trace the root cause through all affected call paths.
3. Identify the shared mechanism responsible.
4. Define the regression test before changing the implementation.
5. Demonstrate the defect against the frozen baseline.
6. Correct the responsible mechanism, not only the immediate symptom.
7. Run the regression against the corrected implementation.
8. Inspect alternate paths for the same defect class.

Do not add special-case patches to one miniapp or one adapter.

Prefer framework primitives and canonical transition operations that are reusable across execution classes. Keep business decisions in miniapps; keep reliability and enforcement in the framework.

If a required contract rule is ambiguous or implementation reveals a genuine architectural conflict, stop at that decision point and report the exact conflicting passages, evidence, alternatives, and consequences. Never silently rewrite the contract to fit the implementation.

## 7. Evidence and verification

Use the same standard of evidence as WS-A.

For each newly closed finding:

- Record the exact source files and functions changed.
- Record the exact test files.
- Preserve identical test bytes between red and green where applicable.
- Use a fresh isolated red overlay based on the frozen pre-WS-B baseline.
- Reproduce the intended red failure.
- Re-run the identical test source against the corrected implementation.
- Record full source/tree digests, overlay digests, commands, environment, database version, and results.
- Do not reuse stale overlays or rely on abbreviated hashes.
- Do not weaken assertions or remove existing coverage to obtain green.

Run the appropriate package tests, fault-injection tests, race tests, migration tests, analyzer checks, and full suite serially where the shared-database test harness requires `-p 1`.

Test failure windows, including lost responses, transaction rollback, cancellation races, and recovery after an `ATTEMPTING` transition.

Use disposable databases only. Record environmental substitutions honestly. Never claim a test ran when it did not.

## 8. Required WS-B exit report

Submit a standalone report containing:

1. **Baseline identity:** full accepted-state identity and frozen pre-implementation digest.
2. **M-01 closure:** each original mutation, its new predicate, affected-row behavior, and deterministic red/green proof.
3. **Effect authorization design:** transaction boundary, lock order, revalidation inputs, and `ATTEMPTING` transition.
4. **O-01 closure:** post-submission approval and authorization change tests.
5. **L-05/L-06 closure:** error classification and denial-audit proof.
6. **Failure semantics:** cancellation-first/attempting-first, UNKNOWN, NOT_APPLIED, replay, and recovery.
7. **Test evidence:** exact commands, environments, full results, race coverage, and digests.
8. **Findings register:** every finding mapped to a contract invariant, root cause, and discriminating test.
9. **Unresolved risks:** explicitly separate defects, accepted residuals, unverified assumptions, and deferred work.
10. **Architectural assessment:** whether the framework now owns the consequential effect-authorization guarantee without relying on miniapp-specific enforcement.

Do not mark a finding closed merely because its corresponding code changed. Closure requires evidence that the underlying invariant is enforced.

## 9. WS-B completion criteria

WS-B is complete only when:

- M-01 is closed with deterministic transition tests.
- The canonical effect-authorization transaction implements the full accepted revalidation contract.
- O-01 is closed with tests proving invalid state cannot cross the effect boundary.
- L-05 and L-06 are addressed and tested.
- Existing WS-A guarantees remain intact.
- The full regression and race suites pass.
- No CRITICAL or HIGH finding remains.
- All evidence is reproducible against identified source states.
- No prohibited scope or architecture change has occurred.

A passing test suite alone does not constitute acceptance. Submit the exit report for independent adversarial review before claiming WS-B complete.

**Final instruction:** WS-B must establish that the framework controls the precise point at which a durable workflow becomes authorized to perform consequential work. Do not weaken that guarantee for implementation convenience.

**Execution order:** freeze and verify the acceptance baseline → M-01 → effect-authorization transaction → O-01 race proofs → L-05/L-06 → full evidence package → independent adversarial review.

This work order authorizes WS-B implementation. It does not authorize commits, pushes, production effects, or expansion into later workstreams.