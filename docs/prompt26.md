# Prompt26 — Independent Gate 1 Targeted Re-Review

## Role

Act as an independent senior software architect and adversarial reviewer of `github.com/PithomLabs/miniapp`.

You did not implement Prompt25. Your job is to independently establish whether the Gate 1 corrections actually close the findings from Prompt24 and preserve the approved architecture.

Review the working tree, source code, tests, contracts, evidence artifacts, and execution results. Do not accept the implementation report's conclusions merely because the report says they are verified.

## 1. Scope and governing sources

Review only:

- `docs/adv_review_prompt21.md` — original C1–C7 definitions.
- `docs/adv_review_prompt24.md` — Gate 1 findings and owner decisions.
- `docs/phase2/authorize-attempt-design.md`.
- `docs/phase1/invocation-lifecycle-contract.md`.
- `docs/phase2/prompt25-gate1-corrections-report.md`.
- `docs/evidence/p25/README.md`, all referenced diffs and pristine sources.
- `docs/evidence/README.md` and `docs/evidence/digests/p25-closeout-2026-11-23.manifest.txt`.
- The actual implementation and committed-in-worktree regression tests.

The governing architecture remains unchanged:

- The framework owns reliability; miniapps own business logic.
- Authorization is performed at the effect boundary, before provider work.
- Authorization uses its own transaction and preserves the required global lock order.
- No new advisory-lock acquisition paths.
- No provider write before committed authorization.
- Attempt epochs must prevent stale workers from overwriting newer outcomes.
- PostgreSQL 19 is mandatory for this evidence.

This is a targeted re-review, not a general architecture review.

## 2. Reviewer independence and write boundary

Do not modify production source, tests, contracts, design documents, evidence artifacts, or the implementation report.

You may execute tests, inspect source, verify hashes, create disposable databases, and use temporary files under `/tmp`.

Your only permitted repository write is the independent review report:

`docs/phase2/prompt26-gate1-targeted-rereview.md`

Do not commit, stage, push, deploy, migrate, access production secrets, or invoke real provider writes. Use mock providers only.

Do not repair defects while reviewing. If a correction is necessary, report the exact defect and stop.

## 3. Baseline and evidence integrity

Independently establish the actual review baseline:

1. Confirm the current HEAD, worktree state, modified and untracked paths, and whether they agree with the Prompt25 report.
2. Reproduce the reported source and tree digests using the repository scripts. Verify the manifest inventory and exclusions.
3. Inspect the complete `docs/evidence/p25/` package. Verify the red-vs-final diffs, pristine source hashes, test-file hashes, reproduction instructions, and mapping of each behavioral hunk to its finding.
4. Verify that test bytes used on red and green trees are identical. Do not accept a red/green claim based only on matching test names or a narrative description.
5. Reproduce the relevant red and green cases where the package permits. Confirm that each red failure discriminates the intended defect and that green passes for the intended reason.
6. Verify that characterization-only tests are correctly labeled. Never upgrade a characterization result into red/green evidence.

Record any inaccessible artifact, missing file, digest mismatch, or reproduction limitation explicitly.

## 4. Mandatory PostgreSQL 19 environment

Re-resolve the image identity using Docker inspection. The required image is:

`postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`

Verify the running server reports PostgreSQL 19.0 and that the inspected container uses that exact immutable image digest.

Use disposable databases and the project's test fixtures. Do not use or truncate the long-lived development database.

Run database tests serially where required by the shared-database fixtures. At minimum, independently rerun the focused tests for P24-001, P24-002, P24-003, P24-005, P24-007, P24-009, P24-010, and P24-011, followed by the affected package suites. Run the full test battery if practical.

If the exact PostgreSQL 19 environment cannot be established, verdict is **BLOCKED**, not CLEAR. Do not substitute another PostgreSQL major version.

## 5. Finding-by-finding review

### P24-001 — HIGH: terminal withdrawal and truthful audit

Read `authorizeTx`, `authorizeAttempt`, `terminalWithdrawTx`, `withdrawIfPending`, `auditRefusal`, and every relevant caller.

Establish that:

- The `PENDING → CANCELLED` transition and its corresponding `effect-withdrawn` audit are in the same transaction.
- The withdrawal audit is inserted only when the predicated state transition actually wins.
- The transaction commits before the typed refusal is returned.
- A commit failure cannot be reported as a successful withdrawal.
- A repeated attempt cannot resurrect a cancelled effect or insert a second `effect-withdrawn` event.
- Refusal events are distinguishable from withdrawal events.
- Quarantined, mismatched, transient, waiting, contention, and infrastructure-error paths do not incorrectly withdraw an effect.
- No provider call or committed `ATTEMPTING` transition occurs after terminal cancellation.

Adversarially inspect `TestP25POC1CancelFirstWithdraws` and the opposite-order test. Do not assume the report's ordering narrative is accurate.

Verify that the cancel-first ordering is actually forced by synchronization and observable markers, not sleeps or repetition. Inspect the FIFO row-gate and speculative-insertion gate in detail. Confirm that the test proves the intended sequence, including the point at which the authorization transaction observes the terminal invocation.

Verify the authorize-first test proves that `ATTEMPTING` committed before cancellation is attempted and that the resulting in-flight response does not claim cancellation can recall an already-authorized provider call.

The invariant is exactly one `effect-withdrawn` audit per successful `PENDING → CANCELLED` transition. Verify exact event counts and types.

### P24-002 — HIGH: stale attempt outcome fencing

Inspect `transition()` and all callers.

Verify that every outcome UPDATE is fenced by the authoritative `attempt_id` from the in-hand record, in addition to its existing state and operation predicates.

Inspect all callers, including fresh authorization, claim, takeover, reconciliation, and each `sendAndRecord` outcome path.

For CONFIRMED, FAILED, and UNKNOWN independently, verify:

- Epoch A can block in provider execution.
- A is aged and quarantined as described.
- NOT_APPLIED settlement permits a new epoch C.
- C receives a distinct framework attempt ID.
- A's late outcome cannot overwrite C's row.
- A is treated as superseded.
- C's actual result and receipt remain associated with C's attempt.
- Provider execution counts are proven by reliable entered markers and channel counters or equivalent evidence.

Inspect whether the zero-row transition result reliably becomes the existing Superseded outcome. Ensure no caller interprets it as successful settlement.

Reproduce all three red/green cases.

### P24-003 — C2: Settle lock order

Inspect the entire Settle transaction and its transitive helper calls, not just the query that was changed.

Verify that:

- Attribution uses a plain read of immutable identity columns.
- Settle never obtains an invocation-row lock after obtaining the effect-row lock.
- No indirect helper, trigger, foreign key, or other schema mechanism recreates the `effect → invocation` dependency.
- The identity fields read without a lock are genuinely immutable during the relevant operation.
- The Settle path retains the contractually specified effect-lock-only footprint.
- No new advisory lock path was introduced.

Independently inspect schema definitions and migrations. Do not accept the report's zero-FK/zero-trigger claim without verification.

Run `TestP25SettleTakesNoInvocationLock` and `TestP25SettleSerializesOnEffectRow` on PG19. Confirm that the first test discriminates the old implementation and that the second preserves effect-row serialization.

### P24-004 — M5 sequence deferral

This is an owner-approved scope deferral, not a claim that sequence fencing has been implemented.

Verify that the design note and lifecycle contract:

- Explicitly state that the current probe remains mock-only and has no authorization-time generation validation.
- Never present sequence fencing as a current guarantee.
- Define a framework-owned sequence requirement, not one inferred from `Intent.Sequence` presence.
- Require trusted identity propagation bound to tenant, invocation, and effect.
- Require supersession validation and the X-08 barrier test before a sequence-bound non-mock provider can be enabled.
- Clearly state that missing verifier/identity must fail closed when the feature is activated.

The documentation must preserve the owner's decision to defer production sequence implementation. Do not demand a generic sequence table or migration during this review.

### P24-005 — gated + autonomous trust

This area needs particularly careful review.

Read the actual submission choke, persisted invocation trust, admitted manifest, `authorizeTx`, approval lookup, and all related tests.

Verify independently that:

1. Persisted trust—not caller input—determines whether the autonomous exception applies.
2. Persisted trust is independently admitted by the manifest. A disallowed trust mode must refuse even if approval linkage would otherwise be skipped.
3. Gated + Confirm requires a valid approval.
4. Gated + Autonomous skips approval linkage only when the persisted trust is Autonomous and the admitted manifest explicitly permits Autonomous.
5. Tenant, binding, grants, switches, token, ownership, and applicable sequence checks remain mandatory.
6. A caller cannot turn a persisted Confirm invocation into Autonomous by changing an input field or request claim.
7. Terminal invocations cannot use the exception.

**Resolve this apparent discrepancy explicitly:**

The approved Prompt25 plan says any present hostile approval must be fully evaluated, and that denied, expired, mismatched, or cancelled approvals cannot be bypassed via autonomous trust. The implementation report's matrix, however, says `auto+denied-present → authorized`.

Determine whether that test result is consistent with the exact owner disposition and the approved implementation work order. Distinguish an invocation that legitimately persisted Autonomous trust from a Confirm invocation whose trust is being spoofed. Do not silently reinterpret the requirement. If an existing adverse approval must still be honored, identify the precise missing check and classify it. If the approval is genuinely inapplicable to a persisted Autonomous invocation, establish that from the governing contract and actual data model, not assumption.

Also investigate the reported legacy residual:

- Empty `TrustModes` is treated as unrestricted for trust admission.
- The autonomous linkage exception requires explicit `Autonomous ∈ TrustModes`.
- The report says submission can admit a gated autonomous invocation under the empty-mode legacy behavior while authorization later refuses it for missing approval.

Determine whether this creates a permanently unperformable invocation or another workflow inconsistency. Decide whether the residual is compatible with the approved owner decision and Gate 1 scope. Do not dismiss it merely because authorization ultimately fails closed.

### P24-006 through P24-011

Verify each correction independently:

- **P24-006:** `gofmt -l .`, `go vet ./...`, and `scripts/checks.sh`; confirm the reported formatting and check results.
- **P24-007:** Framework-generated attempt IDs are authoritative; caller-provided values cannot become persisted IDs; generation failure fails closed; retry and successive-claim behavior preserve the intended epoch semantics.
- **P24-008:** Capability-membership validation is recorded as a prerequisite before the first non-mock provider. Confirm it is documented as deferred rather than implemented.
- **P24-009:** Inspect the AST policy guard and planted-violation control. Verify the negative control fails loudly, ordinary valid code is not flagged, and the documented alias-analysis residual is bounded.
- **P24-010:** Confirm contention retry guidance is correct for record state and tool class. Ensure `execution-in-progress` behavior was not altered.
- **P24-011:** Confirm `pgx.ErrNoRows` is mapped to a terminal binding refusal in authorization and a deterministic terminal error in Settle, while genuine database errors remain retryable.

For P24-011 specifically, the report says authorization skips the refusal audit for phantom invocations because miniapp/tool attribution is unavailable. The original review requested a terminal binding refusal with audit. Verify the governing attribution rule and the actual available data before accepting that deviation. If an audit can be safely attributed, the omission remains open. If it cannot, require a clear, bounded, explicitly dispositioned residual; do not claim full audit closure without evidence.

## 6. C1–C7 traceability audit

Re-read the original definitions in `docs/adv_review_prompt21.md` and the Prompt24 compliance table.

The original definitions are:

- C1: `authorizeAttempt` takes no advisory lock.
- C2: global lock order.
- C3: authorization uses its own transaction, not the attempt's `pgx.Tx`.
- C4: no new advisory-lock acquirers.
- C5: explicit pending semantics.
- C6: Class A handlers that call `Perform` handle `ErrAttemptInProgress` with bounded retry and correct operation-ID behavior.
- C7: O-01 orphan window closes at the effect boundary.

The Prompt25 report's §11 appears to relabel C5 as fencing/leases, C6 as retry-message semantics, and C7 as terminal authority. Determine whether this is a traceability error.

Do not claim C6 is closed merely because P24-010 corrected a contention message. Prompt24 explicitly classified C6 as mechanism sound but integration unproven, and the Prompt25 scope explicitly deferred the C6 fixture.

The closeout must distinguish:

- Constraints actually re-verified in this gate.
- Constraints improved by Prompt25.
- Constraints still unproven and explicitly deferred.
- The owner-approved scope deferrals.

A documentation correction may be sufficient, but the review must not accept misleading compliance labels.

## 7. Scope discipline

Do not start the remaining Stage 3A work.

Explicitly out of scope:

- Full authorization/crash matrix.
- C6 fixture implementation.
- Revocation tooling.
- Production Settle wiring or operator authentication.
- Provider bypass suite.
- CI changes.
- New migrations or dependencies.
- Any real provider operation.

Do not turn a targeted Gate 1 review into a broad architecture redesign.

## 8. Verdict rules

Return exactly one overall verdict:

**CLEAR**
- P24-001 and P24-002 are independently verified closed with discriminating red/green evidence.
- P24-003 restores C2 with source and test evidence.
- P24-005 fully conforms to the owner decision, including both discrepancies investigated above.
- P24-004 and P24-008 are accurately recorded as bounded, owner-approved deferrals.
- P24-006 through P24-011 are fixed or have explicit, justified, bounded dispositions.
- C1–C7 traceability is accurate; no unproven constraint is represented as closed.
- No unresolved CRITICAL or HIGH finding remains.
- Every MEDIUM finding is either closed or has an explicit owner disposition.
- The required PG19 evidence is reproducible.

**NOT CLEAR**
- Any CRITICAL or HIGH defect remains.
- A required owner decision is not implemented.
- A claimed correction is contradicted by source or test evidence.
- A required red/green result does not discriminate the intended defect.
- A security, authorization, lock-order, or ledger-integrity guarantee is materially weaker than claimed.

**BLOCKED**
- Required source, artifacts, or PG19 environment cannot be inspected or reproduced.
- Do not use BLOCKED to disguise a code defect; report defects as NOT CLEAR.

Do not lower severity to obtain CLEAR. Do not turn an implementation gap into a documentation-only residual without proving that the owner explicitly accepted it.

## 9. Required report

Write:

`docs/phase2/prompt26-gate1-targeted-rereview.md`

Include:

1. Overall verdict and concise rationale.
2. Exact HEAD, worktree state, source/tree digests, and manifest verification.
3. Environment details, exact PostgreSQL image digest, server version, and test database identity.
4. Findings register with severity, source locations, violated invariant, evidence, and disposition.
5. Separate assessments for P24-001 through P24-011.
6. Explicit P24-005 decision analysis, including both discrepancies.
7. C1–C7 traceability comparison against Prompt21/Prompt24.
8. Test commands and observed results; distinguish independently executed tests from source-proven claims and characterization.
9. Any accepted residuals with exact boundaries and activation triggers.
10. A clear statement of whether Gate 1 is released to resume the remaining Stage 3A work.

If NOT CLEAR, provide the smallest corrective work order needed to resolve the remaining findings. Do not implement it.

If BLOCKED, identify exactly what evidence or environment is missing.

Stop after writing the review report. Do not proceed to Stage 3A or Phase 3.