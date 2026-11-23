The review identifies a common pattern: the framework has many sound individual mechanisms, but several of their guarantees break at the boundaries between them—especially when requests race, a process fails between two durable operations, or cancellation overlaps with effect execution.

The remediation prompt below treats the 24 findings as evidence of underlying invariant failures, not 24 isolated tickets. It requires the coding agent to investigate, repair, and prove the shared causes before claiming Phase 1 readiness. It also preserves the review's distinction between mandatory G1 fixes and strongly advised fixes.&#x20;

adv_review_prompt14.md



# Prompt 15 — Root-Cause Remediation of Prompt 14 Findings

## Role

Act as the **Principal Go Architect, PostgreSQL and Distributed Systems Engineer, and Security Engineer** responsible for bringing the Miniapp framework to genuine Phase 1 / Class D readiness.

You are the implementation owner, not merely a code reviewer. Investigate the findings in `docs/adv_review_prompt14.md`, establish their underlying causes, repair the affected framework boundaries, and demonstrate that the corrected invariants hold under concurrency, crashes, retries, and recovery.

Your objective is not to make the review report look green. Your objective is to make the underlying system correct.

## 1. Governing documents and authority

Read these documents before changing code:

1. `docs/plan4.2.md` — immutable architectural authority and Phase 1 scope.
2. `docs/prompt13.md` — original implementation requirements.
3. `docs/prompt13_imp.md` — implementation report and claims being reassessed.
4. `docs/adv_review_prompt14.md` — independent adversarial findings X-01 through X-24, plus the G1 reassessment.
5. `docs/prompt10.md` — original G1 operational acceptance criteria.
6. `docs/evidence/README.md` — existing evidence register.
7. The relevant Phase 0/G0 adjudication, DBOS documentation, and version register.

Authority order:

- `docs/plan4.2.md` governs architecture and phase boundaries.
- G0 decisions remain accepted and binding.
- Prompt 13 defines the implementation requirements unless they conflict with the governing architecture.
- Prompt 14 supplies the independent findings and evidence challenges.
- This prompt directs remediation; it does not authorize an architecture redesign.

Do not modify `docs/plan4.2.md`, the G0 baseline commit, or accepted G0 evidence.

## 2. Mission and governing engineering principle

**Every adversarial finding must trigger a root-cause investigation. Do not merely patch the code location named in the finding.**

An adversarial review identifies observable failures. A finding is not necessarily the underlying defect. Several findings may originate from one broken state transition, missing transaction boundary, incorrect ownership model, or unproven assumption.

For every finding:

1. Reproduce or independently verify the reported behavior against the current source.
2. Identify the violated system invariant.
3. Trace the complete causal path: entry point, state mutation, transaction boundary, concurrent actors, failure window, recovery path, and externally observable consequence.
4. Identify the underlying design or implementation cause. Determine whether related findings share that cause.
5. Formulate a durable correction at the lowest appropriate shared boundary.
6. Add regression tests that would fail against the original implementation and pass against the corrected implementation.
7. Test relevant competing interleavings and failure windows, not only the reported happy-path reproduction.
8. Inspect adjacent code for the same underlying defect.
9. Update documentation and evidence so that the implemented guarantee is described accurately.
10. Re-run the relevant adversarial tests and record what was actually executed.

**Do not confuse a symptom disappearing with an invariant being restored.**

Examples of unacceptable remediation:

- Adding a retry without fixing the state that makes a retry unsafe.
- Adding a mutex around one method while the same state can be modified through another entry point or process.
- Checking cancellation earlier without closing the race between the check and the effect claim.
- Making an approval signal succeed without proving that the correct workflow exists and can receive it.
- Making a test pass by deduplicating the test provider's records.
- Downgrading a failed guarantee to a weaker claim without reconciling it with the governing specification.
- Adding a comment that describes an unsafe race as an accepted design.
- Marking a finding resolved because the relevant code path was not exercised.

When multiple findings share a root cause, repair the shared mechanism and demonstrate that the correction closes all affected findings.

When the original proposed fix is insufficient, find a better fix. You are not required to follow a reviewer's suggested implementation literally. You are required to satisfy the invariant the reviewer identified.

## 3. Scope and non-negotiable boundaries

### In scope

Remediate all actionable findings X-01 through X-24 in Prompt 14, including the HIGH, MEDIUM, and LOW findings.

The five HIGH findings are the first priority. The additional findings are not optional merely because they have lower severity.

### Out of scope

Do not introduce:

- Production provider integrations, OAuth, customer credentials, or real external provider writes.
- River/Class B execution or a Class C substrate decision.
- Sprite execution or untrusted-code execution.
- Fly deployment, G6 rollout, cross-release artifact-retention drills, or UI work.
- A production business playbook or sales semantics.
- A production effects reconciler that belongs to Phase 4.
- A general BPMN/workflow-engine replacement.
- New architectural abstractions that do not solve a demonstrated requirement.

### Absolute `lead-followup` hard stop

The `lead-followup` ADR remains DRAFT and unapproved.

Do not create, modify, register, execute, or approve:

- A `lead-followup` implementation.
- Its manifest or registration.
- A runnable `lead-followup` workflow.
- Production sales semantics or a production playbook.
- Any bypass around the ADR approval hard stop.

The neutral Class D conformance probe remains the framework fixture.

Do not invent owner decisions for OSI-1 through OSI-4 or the Prompt 13 §22 decisions. Preserve them as open items.

### No commits or pushes

Do not commit, push, amend history, or modify remote state. Leave the entire working tree uncommitted.

## 4. Baseline and working-tree protection

Before implementation:

1. Record the current HEAD, branch, remotes, staged changes, tracked modifications, and untracked paths.
2. Compare the current repository state with the Prompt 14 report's recorded baseline (`5cf9627` over G0 `e6b5b7b`). Report any drift.
3. Inspect the contents of every pre-existing modified or untracked file before deciding whether it belongs to this task.
4. Identify owner-authored and prior-review files. Preserve them.
5. Do not use `git clean`, `git reset`, `git checkout`, `git restore`, or stash operations to simplify the working tree.
6. Do not overwrite existing user-owned documents or prior evidence.
7. Never delete an existing migration or rewrite an applied migration to make a test pass.

The ten tracked modifications and 30+ untracked paths reported by Prompt 14 are not permission to overwrite those files. Verify the actual current state rather than assuming the report's snapshot is still current.

If a file's ownership or intended use is unclear, preserve it and work around it.

## 5. First deliverable: root-cause analysis before implementation

Before modifying source code, create a working root-cause matrix. It may initially be recorded in `docs/prompt15_remediation.md` or an equivalent new, non-conflicting document.

For every X-01 through X-24, record:

| Field                        | Required content                                                  |
| ---------------------------- | ----------------------------------------------------------------- |
| Finding                      | Original Prompt 14 ID and severity                                |
| Verification                 | Confirmed, partially confirmed, contradicted, or not reproducible |
| Violated invariant           | The precise guarantee that fails                                  |
| Root cause                   | The underlying design or implementation defect                    |
| Shared causes                | Other findings affected by this same defect                       |
| Repair boundary              | The lowest shared component that should own the correction        |
| Regression test              | A test that fails against the old behavior                        |
| Additional adversarial tests | Other interleavings and failures needed                           |
| Acceptance evidence          | Commands, assertions, and actual results                          |

Do not begin with a collection of local code edits. First establish how the affected state machines and transaction boundaries fit together.

If a finding cannot be independently reproduced because the required environment is unavailable, inspect the source and existing tests, retain the finding as unverified, and implement against the governing invariant where the defect is credible.

Do not dismiss a finding merely because the original review environment could not run the corresponding database test.

## 6. Workstream A — Invocation submission and approval safety

**Findings: X-01, X-02, X-03.**

These findings concern a common framework boundary: durable invocation identity, approval linkage, and workflow submission are not one coherent recoverable state machine.

Investigate `runtime/service.go`, `runtime/store.go`, the approval package, the DBOS executor, and all relevant migrations.

### Required invariants

1. An approval-required tool must never execute consequential effects without a durable, valid approval linkage.
2. A retry must never transform an approval-required invocation into an unapproved submission.
3. An invocation must not remain permanently stranded because approval persistence, audit recording, or DBOS submission failed between operations.
4. Every invocation must have one stable, discoverable workflow identity.
5. Concurrent requests using the same idempotency identity must not create multiple independently executing workflows.
6. A workflow that has been submitted must remain discoverable, addressable, and cancellable through the framework.
7. A failed submission must leave a state from which a later retry or poll can safely converge.

### X-01 — Approval bypass on retry

Investigate the complete sequence:

- Initial invocation claim.
- Determination that approval is required.
- Approval-row creation.
- Approval linkage persistence.
- Invocation transition into the approval-pending state.
- Audit creation.
- Workflow ID assignment.
- DBOS submission.
- Same-key retry and replay.

Trace every failure window, including crashes between durable writes.

Correct the state machine so that `confirmSubmit`, replay, and every retry path independently enforce the approval requirement.

A gated invocation must not be submitted without a valid approval ID and the correct persisted approval state. The workflow itself must fail closed if required approval information is missing; do not rely exclusively on the service method to enforce this.

Retries must idempotently resume or reconstruct the legitimate approval path. They must never fall through to ordinary submission simply because an earlier step failed.

### X-02 — Approval-path submission and audit failures

Repair the lifecycle so that an invocation cannot be permanently stranded in `WaitingApproval` with a run ID for a workflow that does not exist.

Inspect and repair:

- Approval-row creation and retry behavior.
- Persistence of approval linkage.
- Persistence of the stable run ID.
- Audit failure handling.
- Submission failure handling.
- `pollApproval` behavior.
- Grant, rejection, expiration, and cancellation after a failed submission.

The framework must have a deterministic recovery operation that can establish whether the correct run exists and safely submit it if missing.

Do not assume that polling a nonexistent run can eventually succeed. Do not make an operator perform manual database surgery to recover an ordinary transient failure.

If approval state and invocation state are in the same PostgreSQL database, determine which operations should be atomic in one transaction. If DBOS submission necessarily remains a separate boundary, persist enough information to recover safely across that boundary.

### X-03 — Concurrent same-key submission

The current unpredicated `run_id` overwrite is not acceptable.

Repair run-ID allocation and adoption at the database boundary. A conditional update, row lock, or another proven database-arbitrated mechanism may be appropriate.

Do not rely on an in-process mutex: multiple service instances can receive the same request concurrently.

Inspect the pinned DBOS SDK's actual behavior when two callers submit the same workflow identity. Do not assume duplicate submissions attach safely without verifying it.

The corrected behavior must guarantee:

- One canonical run ID per invocation.
- One live workflow for the same invocation identity.
- No last-writer-wins run-ID replacement.
- A losing submitter obtains the existing canonical run identity.
- No unaddressable workflow is left executing in the background.

### Required regression tests

Use real PostgreSQL and barrier-controlled concurrency where appropriate.

1. Approval creation fails once; retry same key; verify no unapproved execution.
2. Approval-link persistence fails once; retry; verify the same approval identity is recovered.
3. Crash between approval creation and invocation update; retry; verify convergence.
4. Crash after invocation update but before DBOS submission; poll/retry; verify safe submission.
5. DBOS submission fails once after the approval record is persisted; grant later; verify convergence.
6. Denial after failed submission; verify the invocation settles correctly.
7. Same-key concurrent invocations from separate database connections; verify exactly one workflow identity and no orphan.
8. Different input with the same idempotency key; verify the existing conflict behavior remains intact.
9. Verify that the neutral probe cannot execute effects with a missing or invalid approval linkage.

## 7. Workstream B — Approval decision concurrency and bounded recovery

**Findings: X-04 and X-11.**

Investigate `approvals/approvals.go`, `runtime/lifecycle.go`, the DBOS signal path, approval-wait behavior in the probe, and approval-related audit writes.

### X-04 — Conflicting decisions and cancellation race

Make approval decisions database-atomic.

The database must arbitrate the transition from `pending` to its terminal decision. Use a row lock or a correctly predicated state transition, with an affected-row check. Apply equivalent state-predicate discipline to expiration and cancellation.

Required behavior:

- Two identical decisions: one transition; the retry is idempotent and returns the existing decision.
- Approve versus reject: exactly one wins; the loser receives a conflict.
- Decide versus cancel: cancellation and decision cannot overwrite each other silently.
- Approval after cancellation: rejected.
- Requester self-approval: rejected under concurrent as well as sequential execution.
- Expiration versus approval: one valid state transition wins according to the persisted state and policy.
- Material hash and authority checks remain enforced.

Do not treat sequential idempotency tests as evidence of concurrency safety.

### X-11 — Lost signal and unbounded approval wait

The approval database record is authoritative. A DBOS signal is a notification, not the approval decision itself.

The current design relies on a retrying decider or a probe-owned receive timeout. That is not a framework-level bounded recovery guarantee.

Choose and implement a framework-enforced recovery strategy. Suitable approaches may include:

- A framework-owned approval wait primitive with a mandatory maximum observation interval, followed by a database re-read.
- A bounded framework-controlled notification-redelivery mechanism.
- Another explicitly justified approach that guarantees recovery after the decision transaction commits but the signal is lost.

Do not introduce the Phase 4 production effects reconciler under another name. Keep the solution narrowly scoped to approval lifecycle correctness.

The implementation must not allow a Class D miniapp to accidentally park indefinitely on an unbounded raw approval receive operation while claiming that approval expiration and decision recovery are guaranteed.

Inspect the pinned DBOS SDK to determine the exact timeout and signal behavior. Do not assume that a timeout or buffered signal works differently from what the SDK actually implements.

Persist the approval decision audit event at the durable decision transaction boundary. A crash after committing the decision but before sending the signal must not erase the audit record.

Remove or correct the inaccurate "durable mailbox" description. Do not claim a mailbox exists unless there is an actual persisted mailbox implementation.

### Required regression tests

- Concurrent approve/reject with a synchronization barrier.
- Concurrent decide/cancel.
- Self-approval race.
- Expiry versus decision race.
- Decision commits; process dies before signal; decider does not retry; workflow recovers within the framework-enforced bound.
- Approval expiration is observed within that bound.
- A miniapp cannot bypass the framework's approval-wait guarantees using an unbounded approval wait.
- Audit exists even when signal delivery fails.

## 8. Workstream C — Cancellation and effect-claim linearization

**Finding: X-05.**

Investigate the complete path across:

- `runtime/lifecycle.go`
- `effects/effects.go`
- Effect claim and transition SQL.
- Policy checks.
- Cancellation persistence.
- In-flight reporting.
- Provider invocation.
- Effect settlement.

The current `CheckLive` lock is released before the subsequent effect claim. That leaves a race between the liveness check and the claim.

### Required invariant

**After cancellation is durably recorded, no new consequential effect attempt may begin.**

Repair the claim boundary so that cancellation and effect claiming are ordered by a database-enforced synchronization mechanism.

A transaction that checks invocation liveness and claims the effect must be serialized against the transaction that durably records cancellation.

Possible mechanisms include a conditional claim tied to a live invocation row or a row-lock protocol shared by claim and cancellation. Verify the actual lock and isolation semantics in PostgreSQL.

Do not merely repeat `CheckLive` immediately before the existing claim. A check followed by a separate transaction retains a race.

Do not hold a PostgreSQL transaction or lock across an external provider request.

Define the precise linearization point at which an effect becomes an in-flight attempt. A claim that wins before cancellation must be distinguishable from a claim attempted after cancellation. Cancellation must refuse every new post-cancel claim, while an already in-flight effect must settle according to evidence and remain accurately reported as in-flight.

Preserve R2:

- No duplicate confirmed business effects.
- No silently lost operations.
- No blind retries of unknown outcomes.

Do not claim exactly-once provider delivery.

### Required tests

Use two independent database connections and deterministic barriers.

1. Claim wins before cancellation: effect is visible as in-flight; cancellation reports it accurately.
2. Cancellation wins before claim: claim is rejected.
3. Cancellation commits while the old `CheckLive` stage is paused: no subsequent claim succeeds.
4. Cancellation overlaps with policy evaluation.
5. Cancellation overlaps with a stale lease takeover.
6. A stale worker resumes after takeover and cannot send or overwrite the current owner.
7. A failed or timed-out claim transaction cannot leave a misleading ATTEMPTING record.
8. Verify the external send counter, not only the ledger state.

Correct comments and operational documentation to describe the actual linearization semantics.

## 9. Workstream D — Effect-ledger test integrity

**Finding: X-07.**

Investigate `testkit/mockcap/mockcap.go` and every test that relies on provider-side send counts.

The current mock log deduplicates records by `op_id`. That makes a `SentCount == 1` assertion incapable of proving that only one send occurred.

Fix the measurement, not the assertion.

Record each send attempt independently. A serial attempt identifier or another non-deduplicating design should permit the test suite to distinguish:

- Number of calls to the mock provider.
- Number of accepted provider attempts.
- Number of confirmed business effects.
- Number of unknown outcomes.
- Number of retries and duplicate attempts.

The mock must not conceal duplicate sends through an upsert or uniqueness constraint on the business operation ID.

Add a deliberate double-send test demonstrating that the test suite detects two sends for the same operation.

Re-run every test identified by X-07, including the effect-ledger takeover, kill/restart, and Class D probe send-count assertions.

Remove inaccurate "exactly-once-at-provider" language. Provider attempt counts must remain measurable even when the same operation ID is submitted repeatedly.

## 10. Workstream E — DBOS signal existence and workflow routing

**Finding: X-06.**

Investigate `executor/dbos/executor.go` and the exact signal APIs in the pinned DBOS SDK.

A successful signal call must not be reported as proof that a nonexistent workflow received the signal.

The current fan-out behavior and SDK notification buffering must be examined carefully. Determine how the framework can identify the owning workflow family and verify that the workflow exists and is addressable, including after process restart.

Repair routing and verification so that:

- Signalling an unknown run returns a meaningful error.
- A known run is routed to the correct DBOS application/family.
- A signal after restart reaches the owning run, not an accidental first-match application.
- An unavailable or incompatible run is reported honestly.
- The framework does not silently accumulate orphan notifications while returning success.

Inspect SDK behavior before choosing the implementation. If existence verification and signal delivery cannot be made one atomic SDK operation, explicitly characterize the remaining race and use the durable invocation/run state to make the result recoverable.

### Required tests

- Signal a fabricated run ID; verify an error.
- Signal a valid run after restart with the in-memory run map empty.
- Signal a run owned by a different family from the first registered family.
- Signal a completed run; verify defined harmless behavior.
- Signal an unavailable or incompatible run; verify an honest result.
- Verify approval decisions do not report successful delivery to a nonexistent workflow.

## 11. Workstream F — Actual run attachment and lifecycle truth

**Finding: X-08.**

Inspect the sequence claim, durable-generation identity, invocation records, probe workflow, and runtime lifecycle.

The review found that the current "attach" behavior is detect-and-exit: a second invocation observes an existing sequence and completes successfully without referencing the actual workflow.

That is not equivalent to attaching to the live run.

Follow the governing Prompt 13 requirement that a losing invocation references the correct existing workflow.

Prefer genuine run attachment:

- Persist an explicit relationship between the new invocation and the canonical existing workflow.
- Preserve tenant scope.
- Return the correct stable run handle.
- Ensure the attached invocation's observable lifecycle does not falsely claim that the underlying sequence completed while it is still executing.
- Ensure failure, cancellation, and completion of the target run are reflected accurately.
- Ensure cancellation of an attached invocation cannot accidentally cancel another tenant's or unrelated run.

If implementation reveals that true attachment conflicts with an explicit governing requirement, document the conflict rather than silently changing semantics. Do not relabel detect-and-exit as attach.

Also investigate superseded active generations. A superseded workflow must not continue sending consequential effects when the intended generation has replaced it. Implement and test the appropriate cancellation linkage.

### Required tests

- Barrier-synchronized sequence claims from concurrent invocations.
- Both invocations resolve to one canonical workflow.
- Losing invocation reports a live/attached state, not false success.
- Target workflow success and failure propagate correctly.
- Cancellation of the target and cancellation of an attachment have defined, tested behavior.
- Superseded workflow cannot continue producing effects.
- Cross-tenant sequences remain independent.

## 12. Workstream G — Behavioral DBOS version-affinity proof

**Finding: X-09.**

The per-family AppName and pinned ApplicationVersion mechanism is a good design direction. The existing test, however, uses indistinguishable code and cannot prove that the new version refrains from executing the old version's workflow.

Create a behaviorally discriminating version-transition test.

At minimum:

1. Start a version-1 worker and create a durable pending workflow under version 1.
2. Stop the version-1 worker.
3. Start only the version-2 worker.
4. Demonstrate that the old version-1 pending workflow does not execute version-2 steps.
5. Verify that the correct version can still recover and execute its own workflow.

Use version-distinguishable step markers or another direct source of executor attribution. A single DBOS deduplication marker is insufficient.

Test using the real pinned DBOS SDK and the actual executor configuration. Do not replace the test with parsing a version string or reading SDK source alone.

Retain SDK source inspection as supporting mechanism evidence, not behavioral proof.

## 13. Workstream H — Foreign-version lifecycle visibility

**Finding: X-10.**

Inspect `reportForeignPending` and the SDK's available workflow-status filters.

The current report only covers a subset of statuses and runs once at startup. That is not sufficient for durable workflows that can remain delayed, suspended, enqueued, or otherwise nonterminal.

Make foreign-version visibility cover all relevant nonterminal states.

At minimum:

- Enumerate or query all nonterminal statuses supported by the pinned SDK.
- Identify the owning AppName and ApplicationVersion.
- Include workflows that become nonterminal or otherwise drain-relevant after startup.
- Expose useful counts or records through a queryable operational surface, not only an ephemeral startup log.
- Make incompatible or unknown versions actionable for an operator.

Do not report terminal workflows as stranded work.

Document the exact status set and its SDK meaning. Add tests covering ENQUEUED, DELAYED, SUSPENDED, PENDING, and every other relevant state available in the pinned SDK.

Cross-release artifact retention remains a G6 concern and must not be falsely claimed as solved here.

## 14. Workstream I — Completion and refusal audit completeness

**Findings: X-12 and X-13.**

Investigate the lifecycle audit contract, workflow completion path, effect-policy refusal paths, and every audit event emitted by the framework.

### X-12 — Settlement must not depend on Status polling

A run's settlement audit event must not require a user or agent to call `Status`.

Determine the smallest durable framework-owned completion mechanism that provides complete audit reconstruction for successful and failed Class D executions.

Prefer a durable completion hook or executor completion callback when it can establish the required guarantee. Consider the crash window between workflow completion and audit persistence. The implementation must not leave an unpolled terminal workflow without its settlement record after ordinary recovery.

If a DBOS workflow wrapper can persist settlement before its workflow becomes terminal, investigate that path carefully. Ensure that failures and recovery are also represented correctly. Do not claim a complete audit chain merely because a test polls after completion.

Do not silently introduce the Phase 4 production effects reconciler. A narrowly scoped Phase 1 lifecycle completion mechanism is acceptable when required by G1.

### X-13 — Policy refusals must be attributable

Refused or blocked effect claims must leave an auditable trace.

Record enough structured information to distinguish:

- No effect attempt was made.
- An effect was refused by a kill switch.
- A grant was revoked.
- A token or authority binding was invalid.
- Another policy rule refused the operation.

The refusal record must carry appropriate tenant, invocation, workflow, business-operation, policy-rule, and timestamp attribution.

Do not create an ATTEMPTING effect record for an operation that was never attempted. Use a distinct refusal event or a separately defined refused state.

Do not place secrets or credential material in audit details.

### Required tests

- Complete a workflow without ever calling `Status`; verify the terminal audit chain is complete.
- Fail a workflow without polling; verify its settlement is auditable.
- Kill and recover a workflow near its completion boundary; verify no permanently missing settlement.
- Engage each applicable policy refusal path; verify the correct refusal event exists.
- Verify that refusal events do not masquerade as provider attempts.
- Scan recorded audit events for known test secrets and credential patterns.

## 15. Workstream J — Kill-switch control-plane security

**Finding: X-15.**

Investigate `migrations/000004_policy_controls.up.sql`, `effects.SetKillSwitch`, database role ownership, transaction contexts, miniapp handler privileges, and operational interfaces.

The application database role must not be able to modify the safety control that governs that same application merely because it can execute ordinary miniapp SQL.

The control plane must be separated from ordinary miniapp execution.

Required properties:

1. Miniapp execution cannot engage or disengage kill switches through arbitrary SQL.
2. Only an authorized control-plane identity can mutate switch state.
3. Every switch mutation is attributable to an actor, reason, scope, previous state, and new state.
4. Switch mutation and its audit record are committed consistently.
5. The operator interface cannot bypass canonical authentication and authorization.
6. Missing or invalid control-plane authority fails closed.

Consider role separation, explicit grants, and narrowly scoped database functions. Do not assume that a `SECURITY DEFINER` function is safe without reviewing ownership, search path, privileges, and caller identity.

Provide a minimal operator command or equivalent supported control surface that reuses the framework's authority model. Do not invent a privileged unauthenticated CLI escape hatch.

### Required tests

- A Class A miniapp handler attempts direct `UPDATE` of the kill-switch table; it must fail.
- An untrusted runtime identity cannot call the privileged switch operation.
- An authorized operator can engage and disengage a switch.
- Each successful mutation produces an audit record in the same durable operation.
- Invalid scope, invalid actor, and failed audit persistence cannot silently change switch state.
- Verify the effect-policy path reads the authoritative switch state.

## 16. Workstream K — Determinism analyzer correctness and explicit boundaries

**Finding: X-16.**

Review the analyzer implementation, its fixtures, the seven rule families, and the documented rules in `docs/phase0/determinism-rules.md`.

Do not simply add more pattern matches. Determine whether the analyzer's current traversal model consistently analyzes the code it claims to analyze.

Required corrections:

1. Named step functions and inline step literals must follow a consistent rule set.
2. Add negative fixtures for `RunAsTransaction` and `Enqueue` when used inside forbidden step contexts.
3. Add fixtures for unsupported nondeterminism such as `uuid.NewString()` and other relevant randomness APIs.
4. Add fixtures for depth-two helper calls and cross-package helper calls.
5. Add fixtures for `errgroup.Go`, `reflect.Select`, `context.WithTimeout`, and other identified boundary cases.
6. Add fixtures for any map iterator or library-based iteration behavior that the current AST rules miss.
7. Pin the expected behavior for `dbos.Go` and other DBOS APIs so the analyzer does not reject valid framework constructs indiscriminately.
8. Reconcile fixture counts and rule-family documentation.

For every behavior the analyzer does not detect, choose one of:

- Implement reliable detection.
- Emit a clear diagnostic that the construct is outside supported analysis.
- Explicitly document the boundary and ensure the affected code cannot silently be represented as fully analyzed.

Do not claim that an analyzer proves general workflow determinism. State its actual supported detection envelope.

Verify the DBOS SDK signatures against the pinned dependency. Keep the seven rule families intact unless the governing specification explicitly requires a change.

### Required tests

Add positive and negative fixtures for each relevant behavior and ensure each fixture is actually discovered and analyzed by the strict gate.

A green fixture set that fails to exercise a rule is not evidence that the rule works.

## 17. Workstream L — CI must execute the tests it claims to enforce

**Finding: X-17.**

Investigate the DB-backed kill/restart tests and all GitHub Actions workflows.

The review established that two restart-test parents fail when the DB environment is absent, while the remote CI configuration does not provide a PostgreSQL service.

Correct both sides.

### Test behavior

- Every database-dependent test must detect missing prerequisites before spawning child processes.
- A genuinely unavailable database must produce a clear, fast `SKIP`, not a delayed failure.
- A provisioned but broken database must produce a `FAIL`, not a skip.
- Do not weaken assertions to make CI green.

### CI behavior

Configure a pinned PostgreSQL service with an appropriate health check and required environment variables for the integration test job.

Remote CI must actually execute the DB-backed suites. Do not treat a run in which all DB tests skipped as a passing Class D integration suite.

Preserve the existing Go version and dependency pinning requirements. Make the workflow execute the same relevant integration, contract, analyzer, migration, and restart checks expected locally.

Do not push to trigger remote CI. Record the local CI-equivalent execution and accurately label remote execution as unobserved until it actually runs.

## 18. Workstream M — Evidence integrity and reproducibility

**Finding: X-18.**

Investigate `docs/evidence/README.md`, the Phase 1 readiness matrix, the claim that local and Neon tests ran against the same tree, and the missing cited P0 baseline entry.

The existing evidence register is useful and honest about many incidents. Preserve that quality.

### Required evidence discipline

Before each important test run, compute a deterministic digest of the exact source state being tested.

The digest must cover the actual relevant working tree, including tracked modifications and relevant untracked source files. A Git commit SHA alone is insufficient for an uncommitted implementation.

Use a reproducible sorted file manifest with SHA-256 hashes or an equivalent deterministic worktree digest.

For each important run, record:

- Git HEAD and branch.
- Worktree digest.
- Relevant test command.
- Environment and PostgreSQL/DBOS version.
- Whether tests passed, failed, skipped, or were not run.
- Exact test packages and counts where available.
- Whether the run used local PostgreSQL, Neon, or CI.
- Any fixture or harness modification required before the successful run.
- A clear statement of what the run proves and does not prove.

Compute the digest both before and after the headline test runs. If source files changed during the process, bind each test result to the actual source snapshot it used. Do not claim that two test runs used the same tree without evidence.

Create the missing P0-style baseline full-list entry or correct the references that claim one exists.

Do not retrofit certainty into historical tests for which the exact tested tree cannot be reconstructed. Mark those results as historically recorded but not independently reproducible.

Re-run the important suites against a digested final implementation state, subject to actual environment availability and authorization. Do not create Neon projects or branches, or mutate existing Neon state without authorization.

## 19. Workstream N — Remaining identity, preflight, gate, and contract defects

Resolve all remaining findings in this section. Do not omit them from the completion report.

### X-19 — Unambiguous tenant/entity identity

Replace the ambiguous `tenant:local` concatenation with an unambiguous encoding or a rigorously validated representation.

Use one canonical implementation everywhere entity identity is derived, compared, locked, or persisted.

Do not assume tenant IDs cannot contain a separator unless that constraint is actually enforced at every relevant boundary.

Add the collision-pair regression test identified in Prompt 14.

### X-20 — Preflight orphan-FAIL behavior

Correct preflight behavior for the legitimate approve-then-build sequence.

An approved ADR without a registered miniapp should not be classified as a hard failure merely because implementation has not yet occurred. Use a warning or another accurate status.

Document the boundary between:

- ADR structural preflight.
- Registry admission.
- Manifest validation.
- Runtime enforcement.

Do not duplicate validation without a clear reason, and do not silently leave a required check unenforced.

### X-21 — Repository checks and dependency register

Audit the package lists in `scripts/checks.sh`.

Ensure new framework packages, probe packages, command packages, executor packages, and runtime packages are included in the relevant import and security restrictions.

Update direct dependency checks to include the dependencies introduced by Phase 1 where the project intends to pin them.

Correct the stale version-register entry for `migrate`.

Add fixtures or tests that demonstrate that a forbidden import introduced into a newly added package is caught by the gate.

Do not blindly pin transitive dependencies as direct dependencies. Reflect the actual Go module dependency graph.

### X-22 — Empty run IDs and uniqueness

Reject empty run IDs at the service boundary for `Signal` and `Cancel`.

Inspect the database model and enforce the intended uniqueness of nonempty run IDs. A partial unique index may be appropriate:

`UNIQUE (run_id) WHERE run_id <> ''`

Verify tenant scoping and DBOS workflow identity semantics before applying it.

Add migration coverage for fresh installation and upgrade paths.

### X-23 — Declared but unenforced contract surface

Audit each declared field, state, and transition.

Specifically investigate:

- Manifest `Timeout`.
- The unreachable `CANCELLED` effect state.
- The machine-allowed but operationally unreachable `NOT_APPLIED → ATTEMPTING` transition.
- Superseded probe workflows that continue running.

For each item, either implement and test the promised behavior or remove/clarify the unused contract surface where the governing specification permits.

Do not add dead code solely to make a declared state appear reachable. Resolve the semantic inconsistency.

### X-24 — Additional boundary hardening

Resolve each subfinding separately:

**a. Status synchronization errors**

Do not silently discard synchronization failures. Preserve authoritative terminal status, expose staleness or synchronization failure accurately, and allow safe self-recovery.

**b. Authentication order**

Authenticate before exposing miniapp-registry existence information, unless an explicit governing policy requires otherwise.

**c. REST request size**

Apply an explicit request-body limit and test oversized requests.

**d. Canonical JSON**

Reject trailing JSON values or garbage. Use strict decoding and confirm the decoder reaches EOF after the expected value.

**e. Mock tenant attribution**

Record and assert the actual tenant identity in mock provider attempts. Add cross-tenant attribution tests.

**f. Run-routing map**

Inspect the process-local `runs` map for unbounded growth and correctness after restart. Bound it, expire entries safely, or replace it with a durable routing mechanism. A cache must not become the authority for run existence or identity.

Also add a test that scans recorded audit data for test credentials and secrets, as required by the Prompt 14 security assessment.

## 20. Database and migration requirements

All database changes must preserve PostgreSQL correctness and existing data.

1. Never rewrite an already applied migration to make the current test suite pass.
2. Add forward-only migrations for schema changes.
3. Test fresh installation and upgrade from the actual prior schema states.
4. Verify rollback behavior where it is meaningful; document intentionally irreversible migrations.
5. Ensure unique constraints and indexes enforce the intended concurrency model.
6. Keep tenant predicates explicit even where RLS is active.
7. Preserve `FORCE ROW LEVEL SECURITY` and the accepted G0 transaction-local tenant context behavior.
8. Do not hold transactions or advisory locks across durable waits or external calls.
9. Do not alter DBOS-owned tables directly.
10. Do not change the existing database or Neon project/branch structure without authorization.

For each migration, record its purpose, upgrade behavior, fresh-install behavior, and relevant test evidence.

## 21. Cross-cutting adversarial validation

The individual tests above are necessary but not sufficient.

After implementation, conduct a focused adversarial validation of the repaired system.

Test the following classes of behavior:

### Concurrent requests

- Same invocation identity.
- Different input under the same idempotency key.
- Competing approval decisions.
- Approval versus cancellation.
- Effect claim versus cancellation.
- Sequence generation claims.
- Concurrent effect claims and stale lease takeover.

Use separate PostgreSQL connections and deterministic synchronization barriers. Do not rely on timing sleeps to produce races.

### Failure between durable operations

Inject failures at each meaningful boundary:

- Invocation creation.
- Approval creation.
- Approval linkage.
- Audit insertion.
- Run-ID persistence.
- DBOS submission.
- Signal transmission.
- Approval polling.
- Effect claim.
- Provider send.
- Effect settlement.
- Workflow completion audit.

For each injected failure, demonstrate that the next legitimate retry or recovery operation converges without bypassing authorization, creating duplicate workflows, losing an operation, or hiding an unknown outcome.

### Process failure and recovery

Use real process-kill tests where required by the governing specification. In particular:

- Kill after approval decision commit but before signal.
- Kill after stable run-ID persistence but before workflow submission.
- Kill during effect execution.
- Kill after an effect is durably marked ATTEMPTING.
- Kill near workflow completion and settlement audit.
- Restart with the relevant DBOS version and application identity.

Do not substitute a mocked process-restart test for a real process-kill test.

### Security and tenant boundaries

Verify:

- Server-derived tenant identity.
- Authority-principal and agent-principal separation.
- Auth-before-dispatch.
- Effect-time grant/token/policy checks.
- Kill-switch privilege separation.
- Cross-tenant invocation, approval, run, and effect isolation.
- Audit attribution.
- Absence of credentials and secrets in workflow inputs, checkpoints, logs, and audit events.

## 22. Verification requirements

Execute the available checks in a meaningful order. Adapt commands to the repository's actual scripts without removing assertions.

At minimum, run:

```
gofmt -l .
go vet ./...
go test ./... -count=1
go test -race ./...
bash scripts/checks.sh
go run ./cmd/preflight --check-all
```

Also execute targeted tests for every repaired finding and all relevant migration, DBOS recovery, transport, and OpenCode tests.

Run DB-backed suites against real PostgreSQL. Run them against authorized Neon infrastructure only when it is available and permitted.

Separate and report:

1. Static checks.
2. Unit tests.
3. PostgreSQL integration tests.
4. DBOS process-recovery tests.
5. Concurrency/race tests.
6. Analyzer and preflight tests.
7. Transport and OpenCode E2E tests.
8. CI execution.
9. Tests skipped because prerequisites were genuinely unavailable.
10. Tests that failed.

A test that did not execute is NOT PASS.

A test that skipped because the DB was unavailable is not proof of database behavior.

A test that failed because prerequisites were incorrectly configured is still a failure until the underlying test or CI configuration is repaired.

Do not weaken tests, add broad skips, or change test expectations merely to obtain a green result.

If the environment cannot execute a required test, implement the regression test, validate its setup as far as possible, and mark the behavioral guarantee NOT PROVEN. Do not invent a result.

## 23. Required acceptance matrix

Close every Prompt 14 finding. Use the following matrix as the minimum tracking structure:

| Finding | Required outcome                                                              |
| ------- | ----------------------------------------------------------------------------- |
| X-01    | Approval-required invocation cannot bypass approval on retry                  |
| X-02    | Approval-path failures recover; invocation cannot remain permanently stranded |
| X-03    | Concurrent same-key submit resolves to one discoverable workflow              |
| X-04    | Competing approval decisions are database-serialized                          |
| X-05    | No effect claim begins after durable cancellation                             |
| X-06    | Unknown workflow signals fail; valid signals route correctly                  |
| X-07    | Mock measures every provider attempt independently                            |
| X-08    | Actual run attachment or fully spec-compliant lifecycle semantics             |
| X-09    | Version affinity is behaviorally discriminated                                |
| X-10    | All relevant foreign-version nonterminal work is visible and actionable       |
| X-11    | Approval recovery and expiry observation are framework-bounded                |
| X-12    | Settlement audit does not depend on Status polling                            |
| X-13    | Policy refusals have durable, attributable audit records                      |
| X-14    | Shutdown failure cannot silently close the shared pool under active workers   |
| X-15    | Miniapp code cannot mutate kill switches; authorized changes are audited      |
| X-16    | Analyzer behavior is consistent, fixture-backed, and accurately bounded       |
| X-17    | Tests skip correctly without prerequisites; configured CI executes DB tests   |
| X-18    | Important evidence binds to exact source digests                              |
| X-19    | Tenant/entity identity encoding is unambiguous                                |
| X-20    | ADR preflight correctly handles approved-but-not-yet-registered ADRs          |
| X-21    | Repository checks cover new packages and dependency register is accurate      |
| X-22    | Empty run IDs rejected; nonempty run identity uniqueness enforced             |
| X-23    | Declared contract fields and states are implemented or explicitly retired     |
| X-24    | All six boundary-hardening subfindings are resolved                           |

Classify each result as:

- **PASS** — invariant implemented and supported by executed regression evidence.
- **PARTIAL** — implementation exists, but a required test, environment, or guarantee remains unproven.
- **FAIL** — invariant is still violated.
- **NOT RUN** — required verification was not performed.

Do not use PASS to mean merely "the code looks correct."

## 24. G1 reassessment

Rebuild the full 14-row G1 matrix from `docs/prompt10.md` and map it accurately to `docs/plan4.2.md`.

At minimum reassess:

1. Class A synchronous execution.
2. Class D durable execution.
3. One canonical invocation service.
4. Authorization before dispatch.
5. Complete audit attribution.
6. Version-safe recovery.
7. No duplicate confirmed effects.
8. Entity identity and serialization.
9. Approval-pending recovery.
10. OpenCode E2E.
11. Analyzer and fixtures.
12. CI enforcement.
13. Provider writes disabled.
14. Evidence recorded and verifiable.

Prompt 14 found 4 PASS and 10 PARTIAL. Do not carry forward the former 13/14 claim.

A corrected code path does not automatically upgrade a G1 result. Each row must have evidence from the corrected, identified source state.

Do not claim overall G1 PASS if a required row remains PARTIAL, FAIL, or NOT RUN.

Remote CI remains unobserved until an actual configured remote run occurs. Do not push merely to obtain that observation.

G6 artifact retention, production playbook decisions, OSI-4 volumetric semantics, and Phase 4 reconciliation remain outside G1 unless the governing specification explicitly says otherwise.

## 25. Evidence and implementation report

Create `docs/prompt15_imp.md`, unless that path already exists—in which case preserve the existing file and select a non-conflicting filename.

The report must include:

### A. Executive result

- What was repaired.
- Whether all 24 findings were addressed.
- Whether the five HIGH findings are closed.
- Whether the Class D guarantees are now proven.
- Whether G1 passes, remains partial, or is blocked.

### B. Root-cause analysis

For every X-01 through X-24:

- Original finding and severity.
- Independent verification.
- Actual root cause.
- Shared underlying causes.
- Corrected implementation boundary.
- Regression tests added.
- Actual test results.
- Remaining limitations.

### C. Design corrections

Explain the important state-machine, transaction, concurrency, recovery, authority, and audit changes.

Describe the invariants in terms of the corrected system, not just the code changes.

### D. Test and evidence register

List exact commands and results. Bind important runs to source digests. Separate local PostgreSQL, Neon, and remote CI evidence.

Include all failures and skipped tests, with reasons.

### E. G1 matrix

Provide all 14 rows with PASS, PARTIAL, FAIL, or NOT RUN and the evidence supporting each result.

### F. Security assessment

Reassess tenant isolation, approval authority, cancellation, effect claims, kill-switch privileges, workflow identity, and audit/secret handling.

### G. Change inventory

List every modified, created, and deleted path. Explain migration additions. Explicitly confirm that no user-owned files, G0 baseline documents, or governing architecture documents were overwritten.

### H. Phase and scope declaration

Confirm:

- No `lead-followup` implementation or ADR approval.
- No production provider writes or credentials.
- No River/Class B or Class C decision.
- No Sprite, promotion, Fly, or G6 work.
- OSI-1 through OSI-4 and Prompt 13 §22 decisions remain open.
- No commits or pushes.

## 26. Completion criteria

The task is complete only when:

1. All 24 findings have been independently reassessed.
2. Their root causes have been corrected or their residual status is explicitly justified against the governing specification.
3. All five HIGH findings are closed with regression evidence.
4. All mandatory Prompt 14 G1 remediations are implemented and tested.
5. The strongly advised remaining findings have also been addressed, or any genuine blocker is clearly documented.
6. The relevant concurrency and failure-window tests execute against real PostgreSQL.
7. DBOS version affinity has a discriminating behavioral test.
8. The mock provider cannot conceal duplicate sends.
9. Audit completeness no longer depends on the caller polling a run.
10. The CI workflow is configured to execute the database-backed tests.
11. The evidence register identifies the exact source state used for important results.
12. The working tree remains uncommitted and all phase boundaries are respected.

**Do not stop after producing the root-cause matrix. Implement the corrections.**

**Do not stop after making the tests green. Verify that the tests actually prove the invariant they claim to prove.**

**Do not stop after the five HIGH findings. Address the remaining findings in the same remediation pass.**

The intended outcome is not a larger framework. It is the existing framework, with its current architecture, made correct at the failure boundaries that Prompt 14 exposed.

## **Why this prompt is structured this way**

The most important decision is to group several findings by their shared failure mechanisms instead of asking the agent to fix them independently:

- **Invocation and approval state:** X-01 through X-04 are connected. Fixing only the retry branch could leave a second approval or submission failure window open.
- **Cancellation and effects:** X-05 must be solved at the database claim boundary. Another earlier liveness check would only move the race.
- **Evidence quality:** X-07 and X-18 address a deeper problem: tests and reports must measure actual behavior and bind it to the source that was tested.
- **Lifecycle truth:** X-06, X-11, X-12, and X-14 concern whether the framework can truthfully report delivery, recovery, completion, and shutdown.

The prompt also requires the agent to finish the implementation rather than return another analysis-only report. The next adversarial review should then focus on the corrected boundaries and the new regression evidence—not repeat the entire architecture review.