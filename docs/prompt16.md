# Independent Adversarial Code Review — Miniapp WS-A

## 1. Role and objective

Act as an independent principal engineer conducting an adversarial code review of the completed WS-A implementation in `github.com/PithomLabs/miniapp`.

Your job is not to confirm that the implementation appears reasonable, that the author reports green tests, or that the previous review accepted the work.

Your job is to determine whether the actual source code and tests enforce the governing architecture and lifecycle contract under realistic failure, concurrency, replay, cancellation, and recovery conditions.

Assume that a defect may exist even when:
- The full test suite passes.
- The implementation follows the apparent happy path.
- A test has a convincing name.
- A test passes repeatedly.
- The author has supplied red/green evidence.
- A previous reviewer accepted the work.

Do not assume a defect either. Every finding must be grounded in source code, an executable reproduction, a demonstrable contract violation, or a concrete missing invariant.

**Review only. Do not modify the implementation.**

## 2. Review baseline — freeze and verify first

The submitted WS-A green tree is identified by the abbreviated digest:

`f0b7dfa6…`

The pristine red snapshot is identified by:

`893df60f…`

The author also reports:
- Per-finding red-overlay digests.
- Identical test bytes on red and green sides.
- X-03 and X-04 deterministic red/green results.
- Full suite 21/21 green.
- `checks.sh` passing.
- `gofmt` and `go vet` clean.
- An updated evidence register.

Before reviewing, independently establish the actual checkout and source state.

1. Record the repository path, current revision, branch, full commit SHA, full tree digest, and working-tree status.
2. Verify that the reviewed source corresponds to the submitted WS-A green tree. Obtain the full digest from the authoritative evidence register or repository artifacts; do not infer it from the abbreviated hash.
3. Establish whether the working tree contains uncommitted changes. Identify them explicitly.
4. Locate the exact red snapshot and per-finding overlays. Verify their identity and provenance where the artifacts permit.
5. Identify the exact test files and source files changed by WS-A.
6. Record the exact commands, environment, database setup, and test results you independently reproduce.

Do not silently substitute another revision, a later working tree, or an approximate source snapshot. If the exact submitted baseline cannot be established, report that limitation and do not claim a complete review of that baseline.

Do not edit, reset, rebase, clean, stash, amend, or otherwise mutate the submitted checkout. You may execute read-only inspection and verification commands. Run any necessary experiments in disposable databases and isolated scratch copies. Keep generated files and test artifacts outside the reviewed checkout.

## 3. Governing documents — source of truth

Read and apply the actual repository versions of:

1. `docs/plan4.2.md` — immutable governing architecture.
2. `docs/prompt15.2.md` — governing implementation and review instructions.
3. `docs/phase1/invocation-lifecycle-contract.md` — normative lifecycle contract.
4. The WS-A evidence register and relevant review/decision records.
5. The relevant migration, implementation, and test files.

Do not rely on a summary of these documents in place of reading them.

Preserve the documents' terminology and normative distinctions. Do not silently reinterpret, weaken, reconcile, or replace a requirement with a more convenient implementation assumption.

Where documents appear inconsistent, identify the exact passages and explain the conflict. Do not invent an owner decision to resolve it.

### Owner-approved Rule 3 amendment

The following wording is normative and must be checked against the actual contract and implementation:

> A gated workflow may be submitted with a valid pending approval linkage only to enter its durable approval-wait path. It may proceed to consequential business execution only after the framework verifies a persisted granted decision with matching authority and intent. The effect-authorization boundary independently revalidates that decision. Missing, denied, expired, cancelled, or mismatched approval linkages fail closed.

The distinction is critical:

- **Workflow-start permission:** a valid pending linkage permits submission solely to enter the durable approval-wait path.
- **Business-execution permission:** consequential execution requires a persisted, verified approval grant with matching authority and intent.
- **Effect-authorization permission:** the actual effect boundary independently revalidates the persisted decision.

WS-A claims admission and durable waiting. WS-B owns implementation of the effect-authorization transaction.

Do not report the absence of the WS-B effect-authorization implementation as an accidental WS-A omission. However, do investigate whether WS-A contains any bypass that lets a pending, denied, expired, cancelled, or mismatched approval proceed to consequential business execution before WS-B's boundary can protect it.

## 4. Scope of this review

Review the WS-A implementation and its interaction with existing framework code.

The principal closure areas are:

- X-01 through X-04.
- X-19.
- X-22.
- Rule 3 and the workflow-start/business-execution distinction.
- Durable approval-wait entry and replay.
- Canonical workflow ownership and attachment identity.
- Concurrency controls and transaction boundaries.
- The correctness and discriminatory power of the new tests.
- Relevant migration and identity-encoding behavior.

Follow dependencies into other code when needed to understand the actual behavior. In particular, inspect relevant DBOS integration, lifecycle helpers, executor behavior, transaction management, migrations, and fault-injection boundaries.

Do not expand this into a general architectural redesign or an unbounded review of unrelated packages.

The governing principle remains:

**The framework owns reliability; miniapps own business logic.**

The review must determine whether the framework provides the claimed lifecycle and reliability guarantees without pushing distributed-systems responsibilities onto individual miniapps.

## 5. Adversarial investigation

Do not simply check whether each finding has a test. Trace the real implementation from public entry points through authorization, persistence, DBOS submission, workflow execution, and lifecycle transitions.

### A. X-01 — approval bypass on retry

Investigate whether any retry, replay, repeated submit, or alternate invocation path can bypass the approval requirements.

Specifically examine:
- Whether every submission path passes through the canonical checked-submission mechanism.
- Whether persisted approval state is reloaded and re-evaluated on retry.
- Whether stale in-memory state can authorize submission or execution.
- Whether failure between approval validation, workflow submission, and lifecycle persistence creates a bypass.
- Whether already-submitted and not-yet-submitted states are distinguished correctly.
- Whether a pending approval permits only durable waiting, never consequential business execution.

Find every route into the workflow that could evade the intended checks.

### B. X-02 — stranded approval and orphan workflow recovery

Investigate all failure windows between:
- Persisting the invocation.
- Creating the approval record.
- Recording the run identity.
- Submitting the DBOS workflow.
- Entering the durable wait.
- Replaying or polling the invocation.

Determine whether a crash, timeout, lost response, retry, or duplicate request can create:
- A stranded `WaitingApproval` invocation.
- An orphaned DBOS workflow.
- An invocation without its required workflow.
- A workflow with no valid invocation ownership.
- An approval that cannot be resolved or resumed.

Verify whether the recovery paths are idempotent and whether state reconciliation is based on authoritative persisted state rather than assumptions in process memory.

### C. X-03 — concurrent duplicate submission

Independently challenge the same-key submission race.

The reported test uses a DB lock-gate:
- A holder transaction locks the target row.
- Both racing transactions must observe the pre-write state.
- Both racers are proven blocked behind the designated holder on the intended row.
- The proof uses transitive `pg_blocking_pids` chains, statement shape, and zero unrelated lock waiters across the database.
- The holder releases only after the intended blocking conditions are established.

Verify that the test actually proves those conditions.

Investigate:
- Whether both racers truly reach the relevant database boundary before the holder releases.
- Whether the blocker-chain analysis correctly handles waiter-behind-waiter tuple-lock queues.
- Whether unrelated waiters, unrelated statements, or other locks can make the gate pass incorrectly.
- Whether the gate can release prematurely.
- Whether teardown always releases the holder and joins all racers, including on timeout and assertion failure.
- Whether the test fails against the exact red implementation for the actual defect.
- Whether green guarantees one canonical run identity and no orphan workflow.

Challenge the test helper itself. A concurrency test whose synchronization mechanism is unreliable cannot establish a concurrency guarantee.

### D. X-04 — conflicting decisions and cancellation

Review approve/reject and Decide/cancel races independently.

For approve/reject:
- Verify both racers reach the intended blocked boundary before release.
- Confirm the database transition is genuinely single-winner.
- Check affected-row handling and conditional state predicates.
- Verify that two successful API responses cannot represent two conflicting committed decisions.
- Reproduce the reported red result if possible.

For Decide/cancel:
- Examine both concurrent and sequential orderings.
- Verify whether cancellation can be overwritten by a later decision.
- Verify whether a decision can be overwritten by cancellation or vice versa.
- Check the underlying transaction, row lock, predicates, and affected-row assertions.
- Examine the twenty-round post-release race and sequential overwrite proof.
- Confirm that the tests cover genuinely competing operations rather than merely executing two operations in a convenient sequence.

Distinguish concurrency discriminators from guard tests. Self-approval and expiry checks may be important, but they do not independently prove approve/reject serialization.

### E. X-19 — canonical identity encoding

Review the `ScopeID` encoding and all relevant call sites.

Investigate:
- Whether the encoding is injective over the supported input domain.
- Whether delimiter and escape collisions are impossible.
- Whether decoding is unambiguous.
- Whether empty, malformed, or adversarial identifiers are rejected.
- Whether multi-colon local probe identifiers are handled as intended.
- Whether all relevant lookup, lock, uniqueness, routing, and audit paths use the same canonical representation.
- Whether old persisted identifiers require a safe migration or compatibility path.

Do not accept an encoding simply because ordinary examples round-trip. Challenge its full permitted input domain.

### F. X-22 — empty workflow identities and uniqueness

Inspect the migration and all code paths that create, assign, persist, query, or signal workflow identities.

Verify:
- Empty run IDs cannot enter states that require an owned workflow.
- Uniqueness constraints match the lifecycle contract.
- Partial uniqueness semantics are correct.
- Application checks and database constraints reinforce rather than contradict one another.
- Concurrent assignment cannot silently overwrite ownership.
- Existing records are handled safely and deterministically.
- No ambiguous ownership is resolved by arbitrarily choosing one candidate.

Inspect migration preflight behavior and failure modes. A migration must fail visibly and safely when the data does not support its assumptions.

### G. Durable approval waiting and replay

Treat this as a primary review area, not a superficial marker check.

The reported tests establish:
- A wait-entry marker immediately before `Recv`.
- Authoritative DBOS liveness.
- Zero sends while approval is pending.
- First submission and replay reach the live wait.
- Grant permits exactly one continuation.
- Poll-while-pending does not submit a workflow.
- Rejection or expiry with a missing run does not submit an inappropriate workflow.

Independently determine whether these tests prove the intended behavior.

Investigate:
- Whether the marker can be written even though the workflow never reaches a live wait.
- Whether DBOS liveness is authoritative for the exact workflow being tested.
- Whether the test could pass because the workflow is merely scheduled, not actually waiting for the approval signal.
- Whether replay can enter an incorrect path or duplicate the workflow.
- Whether a lost or repeated signal creates incorrect progress.
- Whether a grant can be observed more than once or trigger more than one continuation.
- Whether rejection and expiry are enforced on all relevant paths.
- Whether a workflow can continue after its invocation or approval has become invalid.

A marker alone is not sufficient proof of a durable wait. Explain precisely what the combined evidence proves and what it does not prove.

### H. Shared lifecycle and transaction correctness

Inspect all relevant lifecycle transitions and their database boundaries.

Challenge:
- Check-then-act races.
- Missing or weak conditional updates.
- Missing affected-row checks.
- Incorrect transaction isolation assumptions.
- Locks acquired too late or released too early.
- Inconsistent lock ordering.
- Deadlocks or retry loops.
- State changes performed before required authorization.
- State changes performed without corresponding audit evidence.
- Stale reads from process memory.
- Incorrect use of tenant, principal, authority, invocation, or workflow identity.
- Error handling that converts failed or ambiguous operations into success.

Trace the complete transaction and lock order. Do not evaluate individual SQL statements without examining how they compose with their callers and surrounding transactions.

### I. Public and alternate entry points

Search for all paths that can:
- Submit or re-submit a workflow.
- Replay an invocation.
- Poll status.
- Decide an approval.
- Cancel an invocation.
- Signal a workflow.
- Assign or alter workflow identity.
- Resume execution or enter consequential work.

Verify that the public API and internal helpers cannot bypass canonical checks by taking a different route.

A correct canonical helper does not provide a guarantee if another reachable path avoids it.

## 6. Adversarial test-quality requirements

Evaluate the tests as rigorously as the implementation.

For every claimed regression test, ask:

1. What exact defect would make this test fail?
2. Does the test force the system into the relevant failure or concurrency boundary?
3. Could the test pass if the defective behavior still existed?
4. Does it test the real public or production execution path?
5. Does it assert externally observable state, not merely internal markers?
6. Does it distinguish one correct outcome from plausible incorrect outcomes?
7. Is its synchronization deterministic?
8. Does cleanup execute on every outcome?
9. Does it use a disposable database and isolate its state?
10. Does it remain meaningful under `-race` and repeated execution?

Do not accept:
- Sleeps used as substitutes for synchronization.
- Tests whose two racers never demonstrably overlap.
- Assertions that check only that an operation returned nil.
- Assertions that do not inspect canonical ownership or persisted state.
- Tests that prove a guard exists but not that the competing operation is excluded.
- Tests that pass because the interesting code path was never reached.
- Test changes that weaken an assertion or remove coverage to obtain green.
- Test doubles that materially differ from production semantics without justification.

For X-03 and X-04, examine the red and green tests byte-for-byte. Confirm that the same test source was used on both sides. Independently determine whether the red outcome demonstrates the original defect, rather than an unrelated failure.

Do not interpret a successful green run as evidence that the test is discriminating. Red failure and green success must be attributable to the intended invariant.

Where a test is not discriminating, say so explicitly, even if the implementation appears correct.

## 7. Required methods

Use multiple complementary methods where useful:

- Source-level call-graph and data-flow inspection.
- SQL and transaction-boundary analysis.
- Migration and constraint inspection.
- Lock-order and concurrency analysis.
- Test-harness inspection.
- Reproduction against the exact red snapshot.
- Reproduction against the exact green snapshot.
- Race-detector runs.
- Fault-injection analysis where relevant.
- Failure-window analysis around DBOS submission and persistence.
- Review of error propagation and recovery behavior.

Inspect the actual pinned DBOS Go SDK version and the repository's integration behavior when a conclusion depends on DBOS semantics. Do not rely on remembered API behavior or assumptions. Clearly separate what the SDK guarantees from what this framework implements around it.

Do not use production databases, real customer credentials, or real provider writes. Use disposable local test resources only.

If the environment prevents a command or test from running, record the exact command and observed failure. Do not infer the result and do not represent an unexecuted test as verified.

## 8. Scope boundaries — do not cross them

This is an audit, not an implementation phase.

Do not:
- Modify source code, tests, migrations, or documentation.
- Start WS-B or implement effect-authorization behavior.
- Implement an effects reconciler.
- Introduce Class B or Class C work.
- Implement the `lead-followup` miniapp.
- Approve or promote the draft `lead-followup` ADR.
- Change the governing architecture.
- Redesign the workflow engine or replace DBOS.
- Introduce provider integrations or real external effects.
- Expand into UI, generators, or unrelated framework work.
- Commit, amend, rebase, reset, push, or otherwise change repository history.
- Update the evidence register to make the review appear closed.

If you discover an adjacent issue outside WS-A, record it separately as an out-of-scope observation. Do not expand the audit or change implementation to address it.

## 9. Finding classification

Use these severities:

- **CRITICAL:** A defect permits unauthorized consequential execution, cross-tenant access, irreversible state corruption, or a fundamental violation of the framework's security or ownership boundary.
- **HIGH:** A defect violates a normative lifecycle or concurrency guarantee, permits conflicting committed outcomes, strands or orphans workflows, bypasses approval, or defeats a required recovery guarantee.
- **MEDIUM:** A meaningful correctness, reliability, auditability, or test-quality defect exists, but its impact is bounded or a reliable containment mechanism remains.
- **LOW:** A limited defect or maintainability issue with no demonstrated material violation of the governing guarantees.
- **INFORMATIONAL:** A clarification, residual limitation, or improvement opportunity that is not a defect.

Do not inflate severity to make the report sound adversarial. Do not downgrade a lifecycle or authorization defect merely because it is difficult to reproduce.

Every finding must contain:

| Field | Required content |
|---|---|
| Finding ID | Stable identifier, e.g. `WSA-001` |
| Severity | CRITICAL / HIGH / MEDIUM / LOW / INFORMATIONAL |
| Contract reference | Exact section, rule, or invariant |
| Source location | File, function, and line range |
| Defect class | Approval, ownership, concurrency, recovery, identity, migration, test quality, etc. |
| Root cause | Underlying mechanism, not the visible symptom |
| Trigger | Exact sequence or condition required |
| Failure mode | What the system actually does incorrectly |
| Impact | Concrete consequence |
| Evidence | Source trace, SQL, test output, reproduction, or proof |
| Discriminating test | Test that would fail for this defect and pass once the root cause is fixed |
| Recommended correction | Durable correction at the responsible shared mechanism |

Do not report speculative findings as established defects. Mark them as hypotheses requiring verification and state what evidence is missing.

## 10. Required review outputs

Produce a review report with these sections.

### A. Executive verdict

Choose exactly one:

- **CLEAR:** No CRITICAL or HIGH defect found; the reviewed WS-A implementation and its evidence meet the examined contract requirements.
- **NOT CLEAR:** At least one CRITICAL or HIGH finding remains, or a critical part of the review could not be verified.
- **INCONCLUSIVE:** The exact baseline or necessary execution environment could not be established sufficiently to support either verdict.

Be explicit that CLEAR is a bounded review of the submitted WS-A scope, not a guarantee that no undiscovered defects exist.

### B. Baseline and provenance

Record:
- Exact revision and full tree digest.
- Working-tree status.
- Red snapshot identity.
- Relevant overlay identities.
- Exact source and test scope.
- Commands and environment used.
- Which evidence was independently reproduced and which was only supplied by the author.

### C. Architecture and contract compliance

For each applicable normative invariant, state:
- The requirement.
- The source mechanism that enforces it.
- The tests that establish it.
- The independent evidence obtained.
- Any residual limitation.

Do not mark a requirement verified merely because a related test exists.

### D. Findings

List findings by severity, highest first. Include all required fields. Provide precise source references and reproducible evidence.

### E. Test-discrimination assessment

Provide a separate table for the principal claims:

| Claim | Test | Red discriminates? | Green passes? | What is actually proven | Remaining gap |
|---|---|---|---|---|---|

At minimum assess:
- X-01.
- X-02.
- X-03.
- X-04 approve/reject.
- X-04 Decide/cancel.
- X-19.
- X-22.
- Rule 3 workflow admission and waiting.
- Replay and grant continuation.
- Relevant recovery and failure-window tests.

Keep guard tests distinct from concurrency discriminators.

### F. Residual risks and accepted boundaries

Separate:
1. Confirmed defects.
2. Unverified claims.
3. Known residual limitations.
4. Explicitly deferred WS-B responsibilities.
5. Out-of-scope observations.

Do not reclassify a documented, owner-accepted residual as a new defect without showing why its accepted assumptions or safeguards are insufficient.

### G. Required remediation before WS-B

List only findings that must be resolved before treating WS-A as independently cleared.

For each, specify:
- The violated invariant.
- The underlying root cause.
- The required durable correction.
- The exact red/green test needed.
- The evidence needed for closure.

Do not prescribe broad refactoring where a narrower shared-mechanism correction is sufficient.

### H. Final recommendation

State whether WS-A is independently cleared to serve as the baseline for WS-B.

Distinguish:
- The previous owner/architectural acceptance of WS-A.
- Your independent review verdict.
- Whether any findings must be fixed before WS-B starts.

Do not claim that WS-B is authorized by this review. That authorization is a separate owner decision.

## 11. Review discipline

Be skeptical, precise, and fair.

- Read the source, not just the report.
- Trace behavior across layers, not just individual functions.
- Challenge tests, not just implementation.
- Prefer reproducible evidence over plausible explanations.
- Identify root causes and shared failure mechanisms.
- Do not silently weaken normative requirements.
- Do not convert missing evidence into a pass.
- Do not claim tests ran unless you ran them and captured their results.
- Do not claim the baseline was verified unless the full identity was established.
- Do not fix defects during the review.
- Do not begin WS-B.

The objective is to establish whether WS-A is a trustworthy, independently examined foundation for the next implementation phase.

**The review is complete only when the report makes it possible for the owner to distinguish what is proven, what is merely claimed, what is defective, and what remains unknown.**