This is the next work order in sequence: **Prompt 22 — WS-B Phase 2: Effect-Time Authorization**. It treats Prompt 21's CLEAR as the release gate, makes P21-002 the first bounded task, and carries all seven Phase 2 compatibility constraints forward.&#x20;

adv_review_prompt21.md

adv_review_prompt21.md



# Prompt 22 — WS-B Phase 2: Effect-Time Authorization

**Project:** `github.com/PithomLabs/miniapp`\
**Workstream:** WS-B — Effect Authorization\
**Phase:** Phase 2\
**Governing review:** Prompt 21 — Independent Adversarial Review of WS-B Phase 1 Closeout\
**Review verdict:** CLEAR\
**Authorization:** The owner has explicitly released Phase 2 through this work order.\
**Primary objective:** Implement and verify atomic, effect-time authorization before a consequential effect can transition from `PENDING` to `ATTEMPTING`.

---

## 1. Your role and responsibility

Act as the senior Go/PostgreSQL architect and implementation engineer responsible for the reliability boundary of Miniapp.

You are not being asked to redesign the framework. You are implementing the next bounded stage of an architecture that has already passed independent adversarial review.

The governing principle is:

> **The framework owns reliability; miniapps own business logic.**

The framework must enforce authorization, lifecycle, tenant isolation, approval binding, concurrency control, effect eligibility, and recovery. Miniapps must not be responsible for recreating these guarantees independently.

Investigate underlying causes and system-level failure modes. Do not merely patch the immediate symptom. Every material correction must have a corresponding invariant, a discriminating regression test, and durable enforcement at the correct shared boundary.

Do not claim a guarantee broader than the actual transaction and provider boundary establishes.

---

## 2. Governing authority and precedence

Before changing code, inspect the complete current versions of these governing materials:

1. `docs/plan4.2.md`
2. `docs/prompt15.2.md`
3. `docs/phase1/invocation-lifecycle-contract.md`
4. `docs/prompt17.md` and its plan-review mirror
5. `docs/phase1/prompt20-phase1-closeout.md`
6. `docs/adv_review_prompt21.md`
7. The Phase 1 Class A execution design and relevant effect-ledger documentation.

The approved architecture and lifecycle contract govern implementation. Prompt 21's independent findings and constraints are binding Phase 2 input.

Resolve any apparent conflict by following the latest explicitly approved governing correction. Do not treat historical prompts, review reports, or superseded instructions as current architecture.

Do not silently reinterpret the contract. When implementation requires a change to a governing invariant, document the reason, impact, alternatives, and required owner decision before proceeding with that change.

### Phase 1 is closed

Prompt 21 independently returned CLEAR. Do not reopen its completed work or start another general remediation cycle.

The following are accepted Phase 1 foundations:

- Predicated lifecycle transitions and stale-result rejection.
- Class A's transaction-scoped execution claim.
- Atomic handler business writes and predicated invocation completion.
- Savepoint handling for Class A handler failures.
- At-most-one concurrent Class A handler execution and at-most-one committed transactional business execution per invocation.
- Durable approval waiting and replay.
- The reviewed lock-order compatibility between Class A execution and Phase 2 authorization.

Do not weaken or casually modify these mechanisms.

---

## 3. Exact Phase 2 objective

Implement one canonical effect-authorization boundary that atomically revalidates authoritative persisted state before an effect is allowed to transition:

`PENDING → ATTEMPTING`

The boundary must prevent an effect from being initiated when the invocation has become ineligible, its approval is invalid, its authority or intent no longer matches, its applicable grant is absent or revoked under the accepted configuration model, a kill switch prohibits execution, its effect token is invalid, or its lease/generation/fencing ownership is stale.

The authorization transaction must commit the `PENDING → ATTEMPTING` transition before the external provider call begins.

The provider call must occur **after the authorization transaction commits**. Never keep the authorization transaction open while waiting for a network call.

A committed `ATTEMPTING` transition means the operation has crossed the framework's authorization boundary and is in flight. It does not prove that a provider completed the operation.

The design must handle ambiguity, retries, cancellation, worker loss, stale ownership, and recovery without claiming exactly-once external delivery.

---

## 4. Mandatory Phase 2 constraints — Prompt 21 §J.5

These seven constraints are binding. Treat them as invariants, not suggestions.

### C1 — No advisory lock in `authorizeAttempt`

`authorizeAttempt` must acquire **no PostgreSQL advisory lock**.

This is essential to the independently verified no-cycle argument between the Class A attempt transaction and the Phase 2 authorization transaction.

Do not add an advisory lock as a shortcut for concurrency control.

### C2 — Preserve the global lock order

Authorization must follow the exact contract-defined order:

`RLock → 1 → 2 → 3 → 4 → 5`

Use the definitions in the current lifecycle contract. The reviewed sequence includes:

1. Invocation row.
2. Approval row.
3. Sequence/generation state.
4. Policy-control/switch and grant state.
5. Effect row and its predicated transition.

The implementation must preserve the specified lock modes and order. Do not reverse the order, acquire a later lock early, or introduce an unreviewed lock path.

Any deviation reopens the deadlock analysis and requires an explicit owner decision.

### C3 — Authorization uses its own transaction

`authorizeAttempt` must run in its own transaction and pool checkout.

It must not receive, reuse, or execute on the Class A attempt's `pgx.Tx`.

This preserves the reviewed separation between:

- Class A's execution transaction, which holds its execution claim and entity lock while performing handler work; and
- the effect-authorization transaction, which follows C1 and C2 and commits its own authorization decision.

Revisit the complete interaction graph before changing this separation.

### C4 — No new advisory-lock acquisition paths

Do not introduce new advisory-lock acquisition sites.

Preserve the existing claim-before-entity order in Class A. Extend the mechanical guard or add an appropriate `checks.sh` raw-SQL gate if the Phase 2 implementation introduces SQL-based paths that could bypass the existing AST guard.

Do not weaken the existing guard to make a new implementation pass.

### C5 — Resolve pending-record semantics before authorization relies on them

P21-002 found a pre-existing, narrow stuck state:

A Class D invocation can remain `pending` after a workflow has been submitted and attached. The workflow then runs against a record that `checkLive`, `cancelIfLive`, and `syncIfLive` treat as ineligible. The result is an effect-paralyzed workflow, an ineffective cancellation request, and a record that may never synchronize.

Resolve this before effect authorization relies on pending-state semantics.

The required invariant is:

> A submitted, attached workflow must have one coherent lifecycle interpretation across effect eligibility, cancellation, synchronization, restoration, and status reporting.

First reproduce and understand the precise trigger. Then select the smallest durable correction consistent with the lifecycle contract.

Evaluate the two review-approved approaches:

- Transition `pending` to `running` through a properly predicated update when a workflow run is successfully adopted/attached; or
- Explicitly support `pending`-with-run as live in every relevant eligibility, cancellation, and synchronization path.

Prefer neither by assumption. Inspect all creation, restoration, submission, adoption, replay, cancellation, and status paths. Choose the approach that preserves the framework's lifecycle invariants and avoids creating a new intermediate inconsistency.

In particular:

- A Class A invocation without a workflow run must retain its correct `pending` semantics.
- A restoration-submitted Class D workflow must not be effect-paralyzed.
- Run identity adoption and any status transition must be atomic and predicated.
- Terminal and quarantined records must remain ineligible.
- No stale settlement may overwrite an adopted run identity.
- Do not add a sweeper or a new lifecycle subsystem as a workaround.

Add a discriminating regression test that reproduces the restoration case and verifies the full behavior, including successful effect eligibility and correct cancellation/status behavior. Test both the live and terminal controls.

Update the lifecycle contract to record the chosen semantics and all relevant state predicates.

**P21-002 must be resolved before `authorizeAttempt` starts depending on a pending-record decision.** If it cannot be resolved safely within this bounded scope, stop and report the exact blocker rather than silently treating `pending` as live.

### C6 — Class A handlers that call `Perform`

Class A handlers that invoke `Perform` must handle `ErrAttemptInProgress` correctly.

The reviewed ledger design has a bounded lease delay before a same-operation takeover after rollback. A handler must not blindly convert this result into success, permanently strand the business operation, or derive a different operation identity on retry.

Verify that:

- The same logical operation retains the same `BusinessOpID` across retries.
- Recovery adopts or observes the existing ledger operation instead of creating a second logical operation.
- `ErrAttemptInProgress` is handled through an explicit bounded retry/recovery policy appropriate to the operation.
- The handler does not hold its Class A transaction indefinitely while waiting for a ledger lease.
- Any retry policy is visible, bounded, and testable.

There are currently no in-tree Class A handlers calling `Perform`. Do not invent a production use case to conceal this latent integration requirement. Add the necessary contract and test coverage using an appropriate fixture, and document the exact boundary.

### C7 — Close the O-01 orphan window at the effect boundary

Phase 1 retained the possibility that a workflow is submitted after settlement wins the adoption race.

`CheckLive` is interim protection, not the durable solution.

The Phase 2 authorization transaction must revalidate invocation liveness and all relevant authority immediately before the effect crosses into `ATTEMPTING`. A workflow submitted for a now-terminal invocation must not be able to initiate a consequential effect.

Do not claim that the submit-time race has disappeared merely because the workflow can still start. Prove that the effect boundary rejects it.

---

## 5. Phase 2 entry task: P21-002

Complete this as the first implementation task.

Required sequence:

1. Inspect the exact lifecycle predicates and every production caller.
2. Reproduce the pending-plus-attached-workflow condition using the real dispatch/restoration path and mock effects.
3. Demonstrate the failure before correction.
4. Select and document the lifecycle interpretation.
5. Implement the smallest durable fix at the shared lifecycle boundary.
6. Add regression tests for restoration, `Perform`, cancellation, and status synchronization.
7. Update the lifecycle contract and relevant operational documentation.
8. Run the focused PostgreSQL 19 tests and the relevant full regression suite.

Do not make the implementation depend on a particular status interpretation until this task is closed.

---

## 6. Contract and evidence maintenance

P21-003 identified missing Class A lifecycle and lock-order rows in the normative contract.

Before building on Class A replay semantics, add the missing rows to the lifecycle contract. At minimum, represent:

- Class A attempt execution and predicated in-transaction completion.
- `execution-in-progress` as a request conflict, not a persisted invocation state.
- Same-key retry behavior for an interrupted Class A execution.
- Class A grant-to-`executeSync`-to-terminal behavior.
- The Class A attempt transaction's advisory claim/entity lock footprint and invocation-row completion lock.

Make the rows match the shipped code and its tests. Do not rewrite historical review reports to create false provenance.

Also address P21-001 where it directly affects the completion evidence: `TestPredicatedOpsFromStates` must not be presented as proof of a dead `completeIfLive` path. Retarget the relevant test to the live `completeTx` behavior, or remove truly dead code and obsolete tests with a clear explanation. Keep the behavioral integration proof intact.

P21-004's one-line traceability and comment corrections are ordinary cleanup. Correct them opportunistically when touching the relevant files, but do not allow them to expand this workstream.

Do not reopen HIGH-1, MEDIUM-1, MEDIUM-2, or MEDIUM-3 absent newly discovered evidence of a real regression.

---

## 7. Authorization transaction design

Before coding, produce a short design note that maps every required check to its authoritative source, lock, transaction, and test.

The authorization transaction must atomically evaluate the following before committing `PENDING → ATTEMPTING`:

### 7.1 Invocation liveness

Re-read the authoritative invocation state under the required lock order.

Verify:

- Correct tenant and invocation identity.
- Invocation is eligible under the chosen lifecycle semantics.
- Invocation is not cancelled, expired, failed, succeeded, or quarantined where those states are ineligible.
- No stale workflow or worker can authorize an effect for a terminal invocation.
- Pending semantics follow the completed P21-002 correction.

Do not rely solely on an earlier `CheckLive` performed when the workflow was submitted or dispatched.

### 7.2 Persisted approval decision

Independently revalidate the persisted approval decision.

Verify the decision is:

- Persisted and granted.
- Bound to the correct tenant.
- Bound to the correct invocation.
- Bound to the correct tool and operation.
- Bound to the correct authority principal and intent.
- Within its validity/expiry conditions.
- Consistent with the authoritative approval record.

A pending approval may be accepted only to enter the durable approval-wait path. It is not authorization to perform consequential business work.

Missing, denied, expired, cancelled, or mismatched approval linkages must fail closed.

A caller-provided approval claim, workflow input, agent assertion, or stale in-memory value is not a substitute for the persisted decision.

### 7.3 Grant and policy snapshot

Preserve the accepted P2 grant model:

- Grants are loaded from configuration snapshots.
- Do not add hot reload.
- Pin the selected in-process snapshot with the approved mutex across authorization.
- Cross-replica coherence remains an explicit deployment property.
- Expose the configuration hash through `/healthz`.
- Verify the expected revocation hash on every replica using the supported `miniapp auth verify-revocation <expected-hash>` operation.
- Preserve the ten-minute maximum staleness baseline.
- A rollback must not restore an older grant that has already been revoked.
- Never claim immediate revocation when the model provides bounded propagation.

Inspect the existing implementation and operational tooling. If any accepted safeguard is missing, record the exact gap. Do not weaken the revocation contract or claim that a partial implementation satisfies it.

### 7.4 Kill switches and policy controls

Read the authoritative policy-control state using the contract's required lock mode and order.

A disabled tool, effect class, tenant, or other applicable kill switch must prevent the effect from becoming `ATTEMPTING`.

The check must be performed at effect authorization, not merely at invocation admission or workflow submission.

### 7.5 Sequence, generation, lease, and fencing

Revalidate the authoritative sequence/generation state and effect ownership.

Verify:

- The effect token identifies the intended effect.
- The effect is eligible for a new attempt.
- The worker owns the current generation/fencing identity.
- The lease is valid under the approved state machine.
- No stale worker can use an old generation to authorize a new attempt.
- The transition's predicates make concurrent attempts converge on one valid winner.

Do not weaken fencing to make retries easier. Do not treat a local process lock as cross-process ownership.

### 7.6 Atomic effect transition

All authorization checks and the predicated `PENDING → ATTEMPTING` transition must occur within the same authorization transaction.

The transition must be conditional on the authoritative state and ownership that were just verified.

A failed predicate, missing row, zero-row update, stale generation, invalid approval, or revoked authority must not be interpreted as success.

The transaction must return a typed and actionable result that distinguishes, at minimum, an ineligible invocation, approval denial/expiry/mismatch, policy denial, stale ownership, an already-in-progress attempt, and infrastructure failure. Align names and API behavior with the existing contract; do not create a redundant error taxonomy without justification.

---

## 8. Provider-call boundary and delivery semantics

The provider call happens only after the authorization transaction commits.

The required sequence is:

1. Begin authorization transaction.
2. Acquire locks in the required order.
3. Revalidate authoritative state.
4. Conditionally transition the effect to `ATTEMPTING`.
5. Commit.
6. Release authorization snapshot protection and database resources as required by the design.
7. Invoke the provider through the canonical framework path.
8. Persist the provider outcome using the established effect state machine and fencing rules.

No network call may occur while holding the authorization transaction open.

Preserve the distinction between authorization and delivery:

- A successful authorization transaction is not proof of provider success.
- A timeout or connection loss after dispatch may leave an ambiguous outcome.
- `UNKNOWN` must not be blindly retried.
- Use the established `BusinessOpID`, provider idempotency support, attempt records, and reconciliation behavior where applicable.
- Never claim exactly-once provider delivery.
- Do not silently change the meaning of `ATTEMPTING`, `UNKNOWN`, `SETTLED`, or other established ledger states.

The framework must not allow a stale worker to overwrite a newer attempt's result. Demonstrate how the generation/fencing rules apply to outcome persistence and recovery.

The cancellation linearization rule is:

- **Cancellation wins first:** the authorization transaction observes the cancelled/ineligible state; the effect does not transition to `ATTEMPTING` and no provider call occurs.
- **Authorization wins first:** the effect is already `ATTEMPTING`; it is in flight and cannot be recalled merely by changing invocation state. Cancellation must prevent subsequent unauthorized effects, but must not falsely claim to have recalled an already-authorized provider call.

Make this distinction explicit in the contract, tests, and operational error reporting.

---

## 9. Canonical execution path and no-bypass requirement

Inventory all production paths that can initiate an external effect, including `Perform`, `Channel`, retry/recovery paths, and any direct provider adapters.

Every consequential provider initiation must pass through the canonical authorization boundary.

Do not fix one path while leaving another path able to call the provider without equivalent checks.

Verify that:

- No adapter independently decides that approval or invocation liveness is sufficient.
- No workflow-side or miniapp-side shortcut bypasses authorization.
- The provider is not called when authorization fails.
- The authorization transaction does not receive the Class A attempt transaction.
- No new advisory lock is acquired.
- Mock providers remain the only providers used for Phase 2 verification.

Do not introduce real customer credentials or real provider writes.

---

## 10. Required test and adversarial verification plan

Derive tests from contract invariants before implementing the production correction. Each important invariant needs a discriminating test at the layer that owns it.

### 10.1 Authorization matrix

Cover at least:

| Condition                                                          | Required behavior                                      |
| ------------------------------------------------------------------ | ------------------------------------------------------ |
| Valid invocation, matching persisted grant, valid granted approval | Authorization may proceed                              |
| Missing approval linkage                                           | Fail closed                                            |
| Pending approval                                                   | May park/wait; must not authorize consequential effect |
| Denied approval                                                    | Fail closed                                            |
| Expired approval                                                   | Fail closed                                            |
| Cancelled approval or invocation                                   | Fail closed                                            |
| Wrong tenant                                                       | Fail closed                                            |
| Wrong invocation                                                   | Fail closed                                            |
| Wrong tool or operation                                            | Fail closed                                            |
| Wrong authority principal                                          | Fail closed                                            |
| Wrong intent                                                       | Fail closed                                            |
| Grant absent or revoked under accepted snapshot model              | Fail closed                                            |
| Kill switch disabled                                               | Fail closed                                            |
| Invalid effect token                                               | Fail closed                                            |
| Effect not `PENDING`                                               | Do not start a new attempt                             |
| Stale generation/fencing owner                                     | Refuse stale attempt                                   |
| Invalid or expired lease                                           | Follow the established ownership/recovery contract     |
| Concurrent authorization for the same effect                       | One valid transition winner                            |
| Terminal invocation with an attached workflow                      | No effect authorization                                |

### 10.2 Deterministic concurrency tests

Use independent PostgreSQL sessions and the actual production transactions. Do not substitute process-local mutexes or mock synchronization for database guarantees.

Force both cancellation race outcomes:

1. Cancellation/settlement commits first; authorization must lose and no provider call may occur.
2. Authorization commits first; cancellation must observe the in-flight state, and the test must verify the documented non-recallable boundary.

Also test:

- Two workers competing for the same `PENDING` effect.
- Two workers with different generations.
- Stale ownership after lease turnover.
- Concurrent approval/policy changes where permitted by the state machine.
- Concurrent restoration and effect dispatch.
- Cross-invocation contention for shared policy or sequence rows.

Assert persisted rows, affected-row counts, attempt records, provider mock logs, and audit outcomes. Do not use a test's goroutine count as proof of database exclusion.

### 10.3 Crash and ambiguous-outcome tests

Inject process failures at meaningful boundaries:

- Before authorization transaction commit.
- Immediately after authorization commit but before provider dispatch.
- During or immediately after provider dispatch.
- After provider response but before outcome persistence.
- During outcome persistence.
- During recovery or lease takeover.

For every point, establish what durable state remains, which worker is allowed to continue, whether a provider call may already have happened, and whether the next action is retry, adopt, reconcile, or remain `UNKNOWN`.

Never turn an ambiguous external outcome into a fabricated success or a blind retry.

### 10.4 P21-002 regression

The regression must reproduce the pending-plus-attached-workflow sequence through the actual dispatch/restoration path.

Verify:

- The workflow is adopted consistently.
- The invocation is eligible for its intended effects.
- `Perform` is not rejected solely because the invocation remains in an incoherent `pending` state.
- Cancellation and status synchronization behave correctly.
- The fix does not make Class A pending records live without a run.
- Terminal and quarantined records remain ineligible.

### 10.5 Class A integration boundary

Add coverage for the C6 `ErrAttemptInProgress` behavior using a Class A fixture that invokes `Perform`.

Demonstrate stable `BusinessOpID` derivation and correct bounded recovery/adoption. Do not claim the issue is closed merely because no current production handler exercises it.

### 10.6 Red/green evidence

For every newly discovered material defect:

1. Freeze the discriminating test source.
2. Demonstrate the defect against the appropriate pre-fix behavior where technically possible.
3. Apply the smallest durable correction.
4. Run the identical test source against the corrected implementation.
5. Record exact test names, commands, exit status, and evidence hashes.

If a true red cannot be staged, state precisely why and classify the evidence as source-proven, composition-proven, characterization, or unverified. Never label a test that passes on both sides as a red/green discriminator.

---

## 11. Mandatory PostgreSQL 19 environment

PostgreSQL 19 is mandatory across:

- Local development.
- Integration and concurrency tests.
- Migration tests.
- DBOS-backed workflow tests.
- Crash-recovery tests.
- CI.

Use the exact PostgreSQL 19.0 environment established by the Prompt 21 review as the parity baseline:

- PostgreSQL 19.0.
- Debian package build `19.0-1.pgdg13+1`.
- The same immutable container image digest across local and CI environments.

Inspect the available local Docker image and record its full immutable digest. Do not mistake a shortened image ID or floating tag for a digest. Configure CI to use the same pinned image identity.

Do not silently substitute PostgreSQL 16, 17, 18, or another version. Do not use a floating `latest` tag as the reproducibility mechanism.

Use disposable test databases. Never run destructive migrations against a user's working or production database.

For shared-database test suites, preserve the established serial execution discipline, including `go test -p 1` where required. Set `TEST_DBOS_DATABASE_URL` when running the DBOS suite so its tests are not silently skipped.

If the exact PostgreSQL 19 baseline cannot be provisioned in an environment, mark the affected verification **BLOCKED**. Do not report PASS based on a different PostgreSQL major version.

Record:

- `SELECT version()` output.
- Image reference and full digest.
- Go version and architecture.
- Relevant environment variables, with secrets redacted.
- Exact commands and working directories.
- Test counts, skips, failures, and duration.

---

## 12. Migration, implementation, and scope discipline

Migrations must be forward-only.

Any schema change must be justified by a Phase 2 invariant and tested against fresh and upgraded disposable PostgreSQL 19 databases. Do not rewrite historical migrations or use destructive shortcuts.

Keep changes localized to the shared framework boundary. Prefer one canonical implementation over duplicated checks in adapters or miniapps.

Do not introduce a new architectural subsystem when a predicated transaction, existing ledger state machine, or shared runtime mechanism can enforce the invariant.

Do not expand this workstream into:

- River or Class B implementation.
- Class C execution-class decisions.
- Phase 4 effects reconciler or X-12 audit-backfill implementation.
- X-10 quarantine operator semantics.
- Sprites, G6, Fly rollout, or production deployment.
- Real provider integrations or customer credentials.
- BPMN replacement or architectural redesign.
- UI, generators, or unrelated miniapp features.

Do not change the accepted DBOS/Class A architecture merely to simplify tests.

No commits, pushes, releases, or deployment actions. Leave the work in the current working tree and report its exact state.

---

## 13. Required implementation sequence

Work in these bounded stages.

### Stage 0 — Baseline and source audit

- Verify repository, branch, commit, and worktree state.
- Read the governing architecture and contract documents.
- Inventory `Perform`, `Channel`, authorization, effect transitions, retry/recovery, provider adapters, policy controls, grants, approvals, and lifecycle predicates.
- Verify PostgreSQL 19.0 parity and test infrastructure.
- Record the exact baseline before editing.

Deliver a concise source map and baseline manifest.

### Stage 1 — P21-002 lifecycle correction

- Reproduce the defect.
- Choose the lifecycle semantics.
- Implement the minimal shared correction.
- Add discriminating regression coverage.
- Update the contract.
- Verify on PostgreSQL 19.

Do not proceed to authorization decisions that rely on pending-state semantics until this stage is closed.

### Stage 2 — Authorization contract and transaction design

Write a concise design note that includes:

- The authorization API and all production callers.
- The authoritative source for each check.
- The exact lock order and lock modes.
- The independent transaction boundary.
- The atomic effect transition.
- Error and retry semantics.
- Cancellation linearization.
- Lease/generation/fencing behavior.
- Grant snapshot and revocation assumptions.
- A lock/transaction graph showing interaction with Class A execution, effect-ledger operations, settlement, cancellation, and recovery.
- The tests that distinguish each invariant.

Have the implementation conform to this design note. If source inspection uncovers a genuine contradiction with C1–C7, stop and report it instead of quietly changing the constraints.

### Stage 3 — Implement the canonical authorization boundary

Implement the smallest shared mechanism that enforces all checks and the predicated `PENDING → ATTEMPTING` transition atomically.

Ensure every production effect path uses it. Ensure the provider call begins only after commit.

Preserve the established error model and ledger semantics.

### Stage 4 — Adversarial verification

Run the full matrix, deterministic race tests, crash tests, recovery tests, Class A integration tests, and canonical-path/bypass inventory.

Use real PostgreSQL 19 sessions and mock providers.

Run the full serial repository suite and required race subset. Include DBOS tests.

### Stage 5 — Documentation and closeout

Update:

- The lifecycle contract.
- The effect-authorization design documentation.
- Relevant operational and revocation documentation.
- Tests and evidence register.
- Any narrowly relevant P21 follow-up notes.

Produce a Phase 2 implementation report with exact evidence.

Stop after Phase 2. Do not begin Phase 3 without a separate owner instruction.

---

## 14. Acceptance criteria

Phase 2 is complete only when all of the following are true:

1. P21-002 is resolved with a discriminating PostgreSQL 19 regression test.
2. The Class A lifecycle rows and attempt-transaction footprint needed by subsequent work are accurately represented in the contract.
3. `authorizeAttempt` is the canonical effect-time authorization boundary.
4. All approval, authority, grant, policy, liveness, token, lease, and generation checks are based on authoritative state.
5. All checks and `PENDING → ATTEMPTING` occur atomically in the separate authorization transaction.
6. The implementation respects C1–C7.
7. Cancellation-first and authorization-first race outcomes are both tested and correct.
8. No production provider path bypasses the boundary.
9. Provider calls occur after authorization commit.
10. Ambiguous provider outcomes are not blindly retried or falsely reported as settled.
11. Class A `Perform` callers have a verified C6 recovery contract.
12. PostgreSQL 19 local and CI parity is demonstrated using the pinned image digest.
13. The full repository suite, DBOS suite, required race subset, and project checks pass without hidden skips.
14. The worktree contains no unexplained changes or stray processes.
15. Every claimed guarantee is linked to source, a discriminating test, and exact execution evidence.

A green unit suite alone is not sufficient. A source review alone is not sufficient. The acceptance decision must combine source inspection, transaction analysis, adversarial tests, and PostgreSQL 19 execution.

---

## 15. Required final report

Create:

`docs/phase2/prompt22-phase2-implementation-report.md`

The report must contain:

1. Executive outcome: COMPLETE, BLOCKED, or INCOMPLETE.
2. Exact repository baseline and final tree identity.
3. PostgreSQL 19 version and pinned image digest.
4. P21-002 diagnosis, selected correction, and evidence.
5. Authorization transaction design and lock-order proof.
6. C1–C7 compliance table with source references and test evidence.
7. Canonical provider-path inventory and bypass analysis.
8. Approval, authority, grant, kill-switch, token, lease, and fencing verification.
9. Cancellation race results for both winner orders.
10. Crash/recovery and ambiguous-outcome behavior.
11. Class A `ErrAttemptInProgress` integration results.
12. Red/green test matrix with exact commands and results.
13. Full test, DBOS, race, vet, formatting, and project-check results.
14. Migrations and operational changes, if any.
15. Remaining defects, residual risks, and deferred work.
16. Exact worktree status; explicitly confirm no commit, push, or deployment occurred.

Classify claims honestly as:

- Verified by execution.
- Verified by source inspection.
- Verified by composition/reasoning.
- Characterization only.
- Accepted residual.
- Blocked or unverified.

Do not turn a hypothesis into a finding, a source inspection into an execution result, or an unavailable test into a pass.

---

## 16. Stop conditions

Stop implementation and report a precise blocker if:

- C1–C7 cannot be preserved.
- The lock-order graph reveals a genuine cycle or an unresolved cross-transaction dependency.
- The authorization boundary cannot be made atomic with the effect transition.
- A provider path cannot be brought under the canonical boundary without an architectural decision.
- P21-002 cannot be resolved without weakening lifecycle safety.
- The accepted grant/revocation safeguards cannot be established.
- PostgreSQL 19 parity cannot be used for required verification.
- A test result contradicts the contract and the discrepancy cannot be explained from authoritative source.
- A necessary design change exceeds this work order.

Do not work around a blocker by weakening a predicate, bypassing a check, introducing an unreviewed lock, broadening a state allowlist, adding a hidden retry, or claiming a stronger delivery guarantee.

When blocked, provide the exact failing invariant, relevant code path, reproduction, evidence, alternatives, and the owner decision required.

---

## 17. Final engineering principle

Phase 1 established that Miniapp can control admission, lifecycle transitions, and single transactional execution.

Phase 2 must establish that a workflow cannot turn an old admission decision into a consequential external action after its authority, approval, invocation state, policy, or ownership has changed.

The critical boundary is not workflow submission. It is the moment the framework authorizes the effect.

**Revalidate at that boundary. Commit the authorization transition before calling the provider. Preserve the distinction between an authorized attempt and a confirmed external outcome.**

Do not weaken C1–C7. Do not claim exactly-once provider delivery. Do not let miniapps become responsible for framework reliability.

**The framework owns reliability; miniapps own business logic.**

**Important:** This prompt authorizes Phase 2 implementation only within the defined WS-B scope. It does not authorize commits, pushes, deployment, real provider writes, or Phase 3.