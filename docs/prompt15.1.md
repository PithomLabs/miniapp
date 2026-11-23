I would consolidate the two Prompt 15 versions, but **not adopt every architectural prescription in the second version literally**.

The strongest addition is its central principle: the five HIGH findings expose a lack of disciplined, recoverable state transitions. That should become the foundation of remediation and future implementation reviews.

There are three important corrections:

1. **One transition discipline, not necessarily one universal&#x20;**`transition()`**&#x20;function.** Invocations, approvals, effects, and workflow runs have different concurrency and persistence semantics. Require subsystem-specific transition operations with common guarantees, rather than forcing an abstraction that may obscure those differences.
2. **Never hold database locks across external calls.** Cancellation and effect claiming must be serialized at the database reservation boundary. An attempt reserved before cancellation is in flight; a reservation attempted after cancellation must fail.
3. **Do not let the structural diagnosis excuse the remaining findings.** The approved scope remains all X-01–X-24, including CI, evidence, security, analyzer, and identity defects—not just the five HIGH findings and their related MEDIUM findings.

I've consolidated the requirements into one implementation prompt. It retains your approved execution choices, root-cause-first sequence, and full remediation scope. The two design/re-audit documents serve as the primary deliverables; the re-audit also serves as the final implementation report, avoiding a third overlapping report. The uploaded Prompt 15 supplies the structural-discipline requirements.&#x20;

adv_review2_prompt14.md

&#x20;The original adversarial findings remain the acceptance basis.&#x20;

adv_review_prompt14.md



# Consolidated Prompt 15 — Structural Root-Cause Remediation of the Class D Framework

## 1. Role and mission

Act as the Principal Go Architect, PostgreSQL Engineer, and Distributed Systems Engineer responsible for remediating the Prompt 14 findings in the Miniapp framework.

You own the implementation, not merely the analysis.

Your mission is to make the existing Phase 1/Class D framework correct at the failure boundaries identified by the independent adversarial review.

This is not a task to close a checklist of 24 findings with 24 local patches.

The deliverable is:

1. A defensible diagnosis of the structural problems that produced the findings.
2. A durable state-transition discipline that prevents the HIGH failure class from silently recurring.
3. Corrected framework behavior across all 24 findings.
4. Adversarial regression tests proving the repaired invariants.
5. Evidence bound to the exact source state that was tested.
6. An honest reassessment of all 14 G1 operational criteria.

**Find the root cause before fixing the symptom. Fix the shared mechanism before fixing individual call sites. Prove the invariant, not merely the implementation.**

## 2. Governing authority

Read the actual repository documents and source. Do not rely solely on previous summaries or implementation reports.

Authority order:

1. `docs/plan4.2.md` — immutable architectural authority.
2. Accepted G0 decisions and recorded owner decisions.
3. `docs/prompt13.md` — authorized Phase 1 implementation specification.
4. `docs/adv_review_prompt14.md` — independent adversarial review and findings X-01 through X-24.
5. `docs/prompt14.md` — review specification.
6. Existing approved contracts and ADRs.
7. Pinned DBOS Go SDK source, version 1.9.0.
8. Previous Prompt 13 implementation report and evidence as historical claims, not authority.
9. This consolidated Prompt 15 as remediation authorization.

Where this prompt conflicts with `plan4.2.md` or an accepted G0 decision, the governing document wins. Record the conflict and do not invent an owner decision.

Do not modify `docs/plan4.2.md`, the G0 adjudication, or any immutable governing document.

## 3. Scope and hard boundaries

### 3.1 Required scope

Address every actionable finding X-01 through X-24, including all six X-24 subfindings.

The five HIGH findings X-01 through X-05 are blockers and must receive first priority.

The MEDIUM and LOW findings remain in scope. Lower severity does not mean optional.

A finding may remain PARTIAL, BLOCKED, or NOT RUN only when a genuine environmental, specification, or authorization constraint prevents its closure. Explain the limitation and its consequence. Do not describe a finding as fixed when the required behavior has not been proven.

### 3.2 Explicitly out of scope

Do not introduce:

- Real external provider integrations, OAuth, customer credentials, or production provider writes.
- Production effects reconciliation or Phase 4 machinery.
- River/Class B execution or a Class C substrate decision.
- Sprite/untrusted-code execution.
- Promotion pipelines.
- Fly deployment or G6 artifact-retention work.
- A general workflow-engine replacement.
- UI or miniapp-generator work.
- A full lifecycle CLI.

A narrow authenticated operator interface for kill-switch management is permitted because X-15 requires an operational control surface. It must use the existing authority model.

### 3.3 `lead-followup` hard stop

The `lead-followup` ADR remains DRAFT and unapproved.

Do not create, modify, register, approve, or execute:

- A `lead-followup` implementation.
- Its manifest or registration.
- Its runnable workflow.
- Its production sales semantics or playbook.

Do not bypass the ADR approval hard stop.

Preserve OSI-1 through OSI-4 and the Prompt 13 §22 owner decisions as open where they remain unresolved. Do not invent their values.

### 3.4 Repository protection

Do not:

- Commit, push, amend history, or mutate remote state.
- Run `git clean`, `git reset`, or destructive checkout/restore operations.
- Overwrite owner-authored or pre-existing untracked files.
- Rewrite an applied migration.
- Destroy or recreate existing Neon resources.
- Modify the G0 baseline or governing documents.

Preserve the working tree except for intended remediation changes.

## 4. Approved execution decisions

Proceed in one continuous implementation pass through baseline, design, remediation, adversarial validation, and final report.

Do not stop after the root-cause matrix. Do not stop after the five HIGH findings.

### Environment

Use local Docker/PostgreSQL with the required host permissions for integration, concurrency, and genuine process-kill tests.

Do not weaken host-wide security settings or access unrelated data to make execution possible.

Neon is best-effort:

- Perform a read-only availability and branch-identity probe first.
- Use `phase0/g0-spikes` only if it exists and is confirmed to be the authorized test branch.
- Do not create another project or branch.
- Do not destroy, recreate, or mutate unverified Neon resources.
- If inaccessible or unsuitable, mark Neon BLOCKED and continue with local PostgreSQL.

### OpenCode E2E

Repeat the scope of Prompt 13:

- Transport integration.
- MCP integration.
- One genuine agent turn.

Do not expand into a multi-turn scenario.

Improve the evidence artifact to include:

- OpenCode version.
- Model/version.
- Source worktree digest.
- Invocation and run identifiers.
- Corresponding server-side audit evidence.
- Redacted credentials.

State exactly what the demonstration proves and what it does not prove.

## 5. P0 — Establish the baseline before changing code

Record:

1. Git HEAD, branch, remotes, and commit history.
2. All staged, modified, and untracked paths.
3. Go version, dependency versions, DBOS SDK version, PostgreSQL availability, and Docker availability.
4. Existing test prerequisites and environment variables.
5. Full baseline test results.
6. First deterministic worktree digest.

The Prompt 14 report records HEAD `5cf9627`, G0 baseline `e6b5b7b`, and a dirty implementation tree. Verify the actual current repository state. Do not assume it remains unchanged.

If the current tree differs:

- Record the exact differences.
- Identify any changed governing document or implementation path.
- Preserve all changes.
- Determine whether the differences affect the remediation baseline.

Do not overwrite or discard work to recreate the review snapshot.

Run available baseline checks, including:

```
gofmt -l .
go vet ./...
go test ./... -count=1
bash scripts/checks.sh
```

Run DB-backed tests where the environment permits. Explicitly distinguish PASS, FAIL, SKIP, BLOCKED, and NOT RUN.

Record the source digest before and after baseline execution.

**No implementation changes are allowed until P1 is complete.**

## 6. P1 — Independently diagnose root causes

Create:

`docs/phase1/classd-state-transition-discipline.md`

This is the first implementation deliverable.

### 6.1 Assess the structural hypothesis

The second Prompt 15 diagnosis proposes that the five HIGH findings arise from a common lack of unified durable state-transition discipline.

Independently assess that hypothesis.

The working hypothesis is:

> The framework has no sufficiently consistent discipline for durable state transitions under concurrency, failure, retry, and recovery. Ad-hoc reads and writes allow different entry paths to produce divergent state, orphaned workflows, bypassed authority, and non-converging retries.

The hypothesis is useful, but do not accept it uncritically.

Determine whether the findings share one root cause, several closely related causes, or distinct causes.

At minimum, investigate these properties:

| Structural property                                  | Findings principally affected |
| ---------------------------------------------------- | ----------------------------- |
| Predicated durable transitions                       | X-03, X-04                    |
| Retry as state-based re-entry                        | X-01, X-02                    |
| Database arbitration across check-and-act boundaries | X-05                          |
| Fault injection at durable boundaries                | X-01 through X-05             |
| Invariant-oriented concurrency tests                 | X-03, X-04, X-05              |
| Honest lifecycle and identity transitions            | X-06, X-08, X-11, X-12        |
| Evidence that measures actual behavior               | X-07, X-17, X-18              |

Identify additional shared causes where justified.

Do not claim that all 24 findings can be solved by one transition primitive. Several are distinct classes of problem, including test-measurement integrity, SDK affinity verification, CI configuration, analyzer coverage, control-plane authorization, and evidence provenance.

### 6.2 Formalize durable state machines

Specify explicit state machines for the affected durable subsystems:

- Invocations.
- Approvals.
- Effects and effect attempts.
- Workflow/run identity and lifecycle.
- Durable sequences and attachments.
- Cancellation.

For each, specify:

- Enumerable states.
- Permitted transitions.
- Expected source state and transition predicates.
- Transaction and locking requirements.
- Effects of successful and failed transitions.
- Audit requirements.
- Idempotency behavior.
- Terminal states.
- Recovery behavior.
- Invariants that must hold in every reachable state.

Include transitions involving external DBOS operations, but keep the application database and DBOS failure boundaries distinct.

### 6.3 Transition discipline

Establish common guarantees across state-machine implementations:

1. Every consequential state change has a defined source state and permitted transition.
2. Mutating database statements use expected-state predicates or an equivalent locking protocol.
3. Every losing concurrent transition detects the loss and returns a defined result.
4. State and its required audit event are committed atomically when they share the same database transaction boundary.
5. Tenant and authority predicates remain explicit.
6. A retry reads current persisted state and dispatches based on that state.
7. Failure between durable operations leaves a recoverable state.
8. Terminal states cannot be silently overwritten.

Use **subsystem-specific transition functions or methods**.

A shared helper or common transition contract may be useful, but do not force invocations, approvals, effects, and workflow runs into one universal abstraction if their persistence or concurrency semantics differ.

Do not create a generic `transition()` function that hides the state-specific rules.

Every affected durable mutation must use an explicit transition discipline. A field update that does not represent a state transition need not be forced through the abstraction, but its concurrency and identity semantics must still be justified.

For each exception, document why the update does not need state-transition predication.

### 6.4 Retry as state-machine re-entry

Every retryable entry point must load the current persisted state and dispatch accordingly.

This includes:

- Invocation submission and replay.
- Approval creation and polling.
- Workflow submission recovery.
- Effect claim and settlement.
- Cancellation redelivery.
- Run attachment.
- Any additional recovery operation discovered during investigation.

There must be a defined handler for every state in which a retry can arrive.

Do not use a caller-provided retry flag to determine whether approval or other mandatory controls are bypassed.

### 6.5 Database lock scope

For every check-then-act operation, establish the serialization boundary that makes the check authoritative.

For cancellation and effect claiming:

- Claim and cancellation must participate in a shared database synchronization protocol.
- If cancellation wins the ordering, no new effect reservation can be committed.
- If the effect reservation wins, that attempt is already in flight and must be visible to cancellation.
- A stale worker cannot send or overwrite the current attempt after losing ownership.
- A failed claim cannot leave a false ATTEMPTING state.

**Never hold a PostgreSQL transaction, row lock, or advisory transaction lock across a provider send, DBOS operation, signal wait, or long-running external call.**

An effect reservation that commits before cancellation is the linearization point for an in-flight attempt. Cancellation may not recall an external send already authorized by that reservation, but it must prevent any new post-cancel reservation.

Document the remaining external-system ambiguity honestly under R2.

### 6.6 Fault-injection discipline

Design a test-only `FaultInjector` facility in the testkit.

It should be able to fail a named operation on its Nth attempt and support deterministic injection at important boundaries.

Keep it out of the public framework API. Do not introduce production feature flags or bypasses to enable it.

Use it for meaningful state transitions and external boundaries, including:

- Approval persistence.
- Invocation linkage.
- Run-ID persistence.
- DBOS submission.
- Signal delivery.
- Effect claim.
- Provider send.
- Effect settlement.
- Completion audit.
- Shutdown.

Do not create meaningless failpoints for every SQL statement. Focus on durable operations whose partial completion changes what a retry must do.

### 6.7 Required design document content

The design document must include:

- Independent root-cause assessment.
- Root-cause matrix for X-01 through X-24.
- Per-subsystem state-machine definitions.
- Transition and recovery semantics.
- Concurrency and lock-order diagrams or tables where useful.
- Fault-injection design.
- Tests to add or strengthen.
- A mapping from each finding to the structural correction.
- Risks the proposed discipline does not eliminate.

Review the design internally before touching source code.

If it exposes a genuine conflict with governing architecture, stop that specific workstream and report the conflict. Do not invent an owner decision.

## 7. P2 — Implementation in dependency order

Implement the changes in the following order. Avoid concurrent edits to shared runtime state-machine code.

### Workstream A — Invocation and approval state

**X-01, X-02, X-03, X-04, X-22**

Investigate the whole invocation and approval transition path across:

- `runtime/service.go`
- `runtime/store.go`
- `runtime/lifecycle.go`
- `approvals/`
- `executor/dbos/`
- Relevant migrations and tests.

#### X-01 — Approval bypass on retry

Every retry of an approval-required invocation must re-enter the approval state machine.

No invocation may be submitted without valid persisted approval linkage when policy requires approval.

The workflow itself must fail closed if required approval data is missing.

#### X-02 — Approval-path submission failures

Make approval-pending invocations recoverable when:

- Approval creation fails.
- Approval linkage persistence fails.
- Audit persistence fails.
- Run-ID persistence fails.
- DBOS submission fails.
- The process crashes between any of those operations.

Polling and retry must be able to establish whether a workflow exists and safely submit it if missing.

Grant and denial must converge from those failure states. No ordinary transient error should require database surgery.

#### X-03 — Concurrent same-key submission

Use database arbitration for run-ID adoption and submission identity.

Prove:

- One canonical run ID per invocation.
- No unconditional overwrite.
- One live workflow per invocation identity.
- The loser reloads and adopts the winning identity.
- No unaddressable workflow remains executing.

Do not rely on an in-process mutex.

Verify actual duplicate-submission semantics in the pinned DBOS SDK.

#### X-04 — Conflicting approval decisions

Make approval decisions atomic through a row lock or predicated state transition.

Test:

- Approve versus reject.
- Decide versus cancel.
- Self-approval races.
- Expiration versus decision.
- Identical decision retries.

Exactly one conflicting transition may win. The loser must receive a defined conflict result.

#### X-22 — Empty and duplicate run IDs

Reject empty run IDs at the service boundary.

Enforce intended nonempty run-ID uniqueness at the database layer after confirming tenant and DBOS identity semantics.

Use a forward-only migration if required.

### Workstream B — Cancellation and effect claim

**X-05**

Repair the database claim boundary.

The liveness check and effect reservation must be serialized against durable cancellation.

Do not merely add another independent `CheckLive` call.

Preserve:

- No duplicate confirmed business effects.
- No silently lost operations.
- No blind retries of unknown outcomes.

Test competing claim/cancel orderings with separate PostgreSQL connections and deterministic synchronization.

### Workstream C — Mock attempt integrity

**X-07**

Change the mock provider so every send attempt creates an independently countable record.

Remove deduplication that conceals repeated sends.

Distinguish:

- Provider call count.
- Accepted attempt count.
- Confirmed operation count.
- Unknown outcome count.
- Retry count.

Add a deliberate double-send test that proves the test suite detects two sends for the same operation.

Re-run every affected effect-ledger and probe assertion.

### Workstream D — Lifecycle and shutdown truth

**X-06, X-11, X-12, X-13, X-14**

#### X-06 — Signal to nonexistent workflow

Use durable invocation/run identity and correct family routing.

Unknown run IDs must return an error. Valid runs must be routable after process restart when the in-memory map is empty.

Do not claim atomic signal delivery merely because `Send` returns nil. Inspect the pinned SDK and characterize existence-check/signal race behavior.

#### X-11 — Bounded approval recovery

Implement a framework-enforced bounded approval wait/recovery strategy.

A miniapp must not accidentally park indefinitely using an unbounded raw receive operation while the framework claims bounded approval recovery.

The approval record remains authoritative. The signal is only a notification.

Persist the decision audit event at the durable decision boundary, not only after successful signal delivery.

Do not claim that a durable mailbox exists unless one is actually implemented.

#### X-12 — Settlement audit

A run's completion audit must not depend on `Status` polling.

Implement a durable completion mechanism and test success, failure, process crash, and recovery around the settlement boundary.

Do not introduce a production Phase 4 reconciler merely to satisfy Phase 1 lifecycle audit.

#### X-13 — Refusal audit

Record policy refusals with appropriate tenant, invocation, workflow, business operation, policy rule, actor, and time attribution.

Do not record an unattempted effect as ATTEMPTING.

Distinguish policy refusal from provider attempt and provider outcome.

Do not place credentials or secrets in the audit.

#### X-14 — Shutdown and shared pool ownership

Inspect:

- `cmd/miniapp/main.go`
- `testkit/stack/stack.go`
- DBOS executor shutdown.
- All relevant tests.

A failed or timed-out DBOS shutdown must not be followed by a silent `pool.Close()` while workers may still use the shared pool.

Preserve diagnostic information and terminate through a clearly reported failure path or another demonstrably safe shutdown strategy.

Add fault-injection tests proving the pool is not closed prematurely.

Do not swallow shutdown errors.

### Workstream E — Run attachment and version safety

**X-08, X-09, X-10**

#### X-08 — Actual run attachment

Inspect sequence claims and lifecycle behavior.

Prefer genuine attachment to the canonical existing workflow:

- Persist the relationship.
- Preserve tenant isolation.
- Return the correct stable run handle.
- Reflect target-run completion and failure.
- Define cancellation semantics for the attachment.
- Ensure superseded workflows cannot continue sending effects.

Do not label detect-and-exit behavior as attachment.

If true attachment conflicts with a governing requirement, document the conflict instead of silently weakening the requirement.

#### X-09 — Discriminating version affinity

Create a behavioral version-transition test:

1. Start v1 and create a pending v1 workflow.
2. Stop v1.
3. Start only v2.
4. Verify v2 does not execute v1's pending work.
5. Restart v1 and verify the appropriate version executes it.

Use executor-distinguishable markers. SDK source inspection is supporting evidence, not a replacement for behavioral proof.

#### X-10 — Foreign-version visibility

Enumerate the nonterminal DBOS states that matter for drain and recovery.

Include relevant PENDING, ENQUEUED, DELAYED, and SUSPENDED work, plus other states supported by the pinned SDK.

Make foreign-version work queryable and actionable beyond a one-time startup log.

Do not claim G6 artifact retention is proven.

### Workstream F — Kill-switch control plane

**X-15**

Separate the control-plane authority from ordinary miniapp database privileges.

Miniapp handlers must not directly modify kill-switch state.

Use appropriate database permissions or narrowly scoped privileged functions, with safe ownership, search path, and caller-authority checks.

Every successful switch mutation must be attributable and audited consistently.

Provide a minimal authenticated operator command or equivalent control surface that reuses canonical authorization.

Add tests proving that:

- Miniapp SQL cannot flip switches.
- Unauthorized callers cannot use the privileged function.
- Authorized operators can change valid switch state.
- Audit and state mutation are consistently persisted.
- Effect-policy reads the authoritative state.

Use a forward-only migration for schema or privilege changes. Use the next valid migration number; never rewrite applied migrations.

### Workstream G — Determinism analyzer

**X-16**

Inspect analyzer traversal consistency and all seven rule families.

Correct named-function versus inline-function divergence.

Add appropriate positive and negative fixtures for:

- `RunAsTransaction` and `Enqueue` in forbidden step contexts.
- `uuid.NewString()` and other relevant nondeterministic calls.
- Depth-two and cross-package helper calls.
- `errgroup.Go`.
- `reflect.Select`.
- `context.WithTimeout`.
- Map iteration and iterator behavior.
- Valid DBOS APIs.

For unsupported analysis boundaries, either implement detection or explicitly surface/document the boundary. Do not silently claim complete coverage.

Correct fixture counts and analyzer documentation.

### Workstream H — CI and test environment

**X-17**

Gate database-dependent tests before spawning child processes.

Missing prerequisites must produce fast, explicit SKIP. A configured but broken database must FAIL.

Configure GitHub Actions with a pinned PostgreSQL service and health check so remote CI will execute DB-backed suites.

Preserve assertions. Do not weaken tests to make CI pass.

Do not push to trigger remote CI. Record remote CI as UNOBSERVED until a real remote run exists.

### Workstream I — Evidence integrity

**X-18**

Every important test run must be tied to the exact source tree.

Use a deterministic sorted file-hash manifest or another reproducible worktree digest that includes tracked modifications and relevant untracked implementation files.

Record:

- HEAD and branch.
- Worktree digest before and after the run.
- Test command.
- Environment and DB version.
- Pass/fail/skip counts.
- Whether the run used local PostgreSQL, Neon, or CI.
- Fixture/harness changes made before the run.
- What the run proves and does not prove.

Correct the missing P0 full-list reference identified in Prompt 14.

Do not claim historical local and Neon runs used the same tree without evidence.

### Workstream J — Identity, preflight, and repository hardening

**X-19, X-20, X-21, X-23, X-24a–f**

#### X-19 — Unambiguous entity identity

Replace ambiguous tenant/entity concatenation with an unambiguous canonical representation.

Use it consistently for locks, comparisons, and persistence.

Add the collision-pair regression test.

#### X-20 — ADR preflight

An approved but not yet registered ADR must not be misclassified as a hard failure simply because implementation has not occurred.

Document the split between structural preflight, registry admission, manifest validation, and runtime enforcement.

#### X-21 — Repository checks

Expand package coverage in `scripts/checks.sh`.

Update direct-dependency pins and the version register according to the actual Go module graph.

Add a fixture proving forbidden imports in newly added packages are caught.

#### X-23 — Declared but unenforced contract

Resolve the semantics of:

- Manifest `Timeout`.
- Unreachable `CANCELLED` effect state.
- Unreachable `NOT_APPLIED → ATTEMPTING` transition.
- Superseded probe workflows.

Implement, enforce, or explicitly retire each surface where permitted by governing specifications.

Do not add dead code just to make a state appear reachable.

#### X-24 — Additional hardening

Resolve every subfinding:

a. Expose status synchronization errors honestly; preserve terminal authority.

b. Authenticate before exposing miniapp-registry existence information, unless a governing rule requires otherwise.

c. Add and test a REST request-body limit.

d. Make canonical JSON reject trailing values and garbage.

e. Record the actual tenant in mock provider attempts and test cross-tenant attribution.

f. Bound or replace the process-local run-routing map. Durable identity, not the map, must remain authoritative.

Also scan recorded audit output for test credentials and secrets.

## 8. Database and migration integrity

All schema and privilege changes must be forward-only.

1. Never rewrite an applied migration.
2. Add new migrations for schema changes.
3. Test fresh installation and upgrades from relevant prior schema states.
4. Record rollback behavior or intentional irreversibility.
5. Preserve explicit tenant predicates and accepted G0 RLS behavior.
6. Never hold DB transactions or locks across durable waits or external calls.
7. Do not change DBOS-owned tables.
8. Do not mutate Neon state without authorization.

Document every migration's purpose, fresh-install behavior, upgrade behavior, and evidence.

## 9. P3 — Adversarial validation

The remediation is not complete when tests pass under ordinary execution.

### 9.1 Failure injection

Inject failures after each meaningful durable boundary:

- Invocation claim.
- Approval creation.
- Approval linkage.
- Audit persistence.
- Run-ID persistence.
- DBOS submission.
- Approval decision commit.
- Signal send.
- Effect reservation.
- Provider send.
- Effect settlement.
- Workflow completion.
- Settlement audit.
- Shutdown.

Verify that the next legitimate retry or recovery operation converges without authorization bypass, orphaning, silent loss, or unsafe retries.

### 9.2 Concurrency

Use independent PostgreSQL connections and synchronization barriers, never sleep-based race tests.

Test:

- Concurrent same-key submission.
- Conflicting approval decisions.
- Decision versus cancellation.
- Expiration versus approval.
- Cancellation versus effect reservation.
- Cancellation versus in-flight attempt.
- Stale lease takeover versus old worker resumption.
- Concurrent sequence claims.
- Cross-tenant identity collisions and isolation.

### 9.3 Process failure

Use the genuine Prompt 13 process-kill harness.

Test real subprocess termination and restart, not mocked restart behavior.

At minimum:

- After approval commit but before signal.
- After run-ID persistence but before DBOS submission.
- During effect execution.
- After ATTEMPTING persistence.
- Near workflow completion and settlement audit.
- During shutdown with an unresponsive worker.

### 9.4 Red/green regression evidence

For important regression tests, demonstrate that the test detects the original defective behavior and passes after remediation.

Use a preserved pre-remediation snapshot or another reproducible isolated method. Do not rewrite history or damage the active working tree to produce a red result.

If a red/green comparison is technically impossible, explain why and mark the evidence appropriately. Do not fabricate a pre-remediation result.

### 9.5 Evidence types

Use real PostgreSQL for database concurrency and database-state invariants.

Pure functions and AST analyzer behavior may use unit tests where that is the correct test boundary.

Do not require every unit test to depend on PostgreSQL. Do require every database guarantee to have real database evidence.

## 10. Required test coverage

At minimum, add or strengthen tests covering:

| Test                                        | Finding | Required assertion                                      |
| ------------------------------------------- | ------- | ------------------------------------------------------- |
| `TestApprovalRetryAfterSubmitFailure`       | X-01    | Retry re-enters approval; never executes unapproved     |
| `TestApprovalRetryAfterParkFailure`         | X-01    | Approval failure cannot bypass the gate                 |
| `TestApprovalPollSubmitsIfMissing`          | X-02    | Poll converges after failed submission                  |
| `TestApprovalDenialSettlesAfterParkFailure` | X-02    | Denial settles correctly                                |
| `TestConcurrentSubmitSameKey`               | X-03    | One workflow; stable run ID                             |
| `TestConcurrentApproveReject`               | X-04    | One decision wins                                       |
| `TestConcurrentDecideAndCancel`             | X-04    | Cancellation cannot be overwritten                      |
| `TestCancelDuringClaim`                     | X-05    | Claim after cancellation is rejected                    |
| `TestCancelDuringAttempt`                   | X-05    | Pre-cancel in-flight attempt is accurately reported     |
| `TestSignalUnknownRun`                      | X-06    | Unknown run returns error                               |
| `TestMockRecordsEveryAttempt`               | X-07    | Duplicate sends are counted                             |
| `TestProbeAttachLifecycle`                  | X-08    | Attachment reflects target run                          |
| `TestVersionV2DoesNotExecuteV1Run`          | X-09    | Version affinity is behaviorally discriminated          |
| `TestForeignVersionNonterminalVisibility`   | X-10    | Relevant nonterminal states are visible                 |
| `TestApprovalLostSignalBoundedRecovery`     | X-11    | Recovery occurs within framework bound                  |
| `TestSettlementAuditedWithoutPoll`          | X-12    | Completion audit is independent of Status               |
| `TestRefusalAudited`                        | X-13    | Policy refusal has durable attribution                  |
| `TestShutdownFailurePreservesPool`          | X-14    | Pool remains open while workers may use it              |
| `TestKillSwitchNotHandlerWritable`          | X-15    | Handler cannot modify safety state                      |
| Analyzer fixtures                           | X-16    | Every supported boundary has positive/negative coverage |
| DB-less restart tests                       | X-17    | Fast explicit skip, not delayed failure                 |
| Digest verification                         | X-18    | Evidence identifies exact source state                  |
| Entity collision test                       | X-19    | Encoded identities cannot collide                       |
| ADR approved-unregistered test              | X-20    | Correct warning/status                                  |
| Repository gate fixtures                    | X-21    | New package restrictions are enforced                   |
| Empty run ID test                           | X-22    | Signal/Cancel reject empty identifiers                  |
| Contract-state coverage                     | X-23    | Declared fields and transitions have defined behavior   |
| Hardening tests                             | X-24    | All six subfindings are tested                          |

Each test must document the invariant or finding it protects.

Do not rely on test names alone. Inspect assertions to ensure they genuinely establish the expected behavior.

## 11. P4 — Verification and acceptance

Run the verification battery:

```
gofmt -l .
go vet ./...
go test ./... -count=1
go test -race ./...
bash scripts/checks.sh
go run ./cmd/preflight --check-all
```

Also run targeted regression, migration, DBOS, process-restart, transport, MCP, and OpenCode tests.

Separate results into:

1. Static checks.
2. Unit tests.
3. PostgreSQL integration tests.
4. Concurrency tests.
5. Process-kill and DBOS recovery tests.
6. Analyzer and preflight tests.
7. Transport and OpenCode E2E tests.
8. Local CI-equivalent execution.
9. Neon execution, if authorized and available.
10. Remote CI, only if an actual remote run exists.
11. Skipped, blocked, failed, and not-run tests.

A skipped test is not PASS.

A test that did not execute is not evidence of behavioral correctness.

Do not weaken assertions or introduce broad skips to obtain a green suite.

## 12. Finding acceptance matrix

Reassess every finding:

| Finding | Required outcome                                                   |
| ------- | ------------------------------------------------------------------ |
| X-01    | Approval cannot be bypassed on retry                               |
| X-02    | Approval-path failures converge                                    |
| X-03    | Same-key concurrent submit produces one canonical workflow         |
| X-04    | Conflicting approval decisions serialize                           |
| X-05    | Post-cancel effect reservation is impossible                       |
| X-06    | Unknown signal fails; valid routing works                          |
| X-07    | Mock measures every actual send attempt                            |
| X-08    | Run attachment has truthful lifecycle semantics                    |
| X-09    | Version affinity has discriminating behavioral evidence            |
| X-10    | Foreign-version nonterminal work is visible                        |
| X-11    | Approval recovery and expiry observation are framework-bounded     |
| X-12    | Settlement audit does not depend on polling                        |
| X-13    | Refusals are attributable and auditable                            |
| X-14    | Shutdown failure cannot silently close a pool under active workers |
| X-15    | Kill-switch state is control-plane protected and audited           |
| X-16    | Analyzer traversal and boundaries are consistent and tested        |
| X-17    | DB-dependent tests skip correctly; CI provisions PostgreSQL        |
| X-18    | Evidence is tied to exact source digests                           |
| X-19    | Tenant/entity identity is unambiguous                              |
| X-20    | ADR preflight handles approved-but-unregistered state correctly    |
| X-21    | Repository gates and dependency register are accurate              |
| X-22    | Empty run IDs rejected; nonempty run identity is unique            |
| X-23    | Declared contract semantics are implemented or explicitly retired  |
| X-24a–f | All six hardening findings are resolved                            |

Classify every result:

- **PASS** — implemented and supported by executed regression evidence.
- **PARTIAL** — implementation exists but a required test or guarantee remains unproven.
- **FAIL** — invariant remains violated.
- **NOT RUN** — required verification was not performed.
- **BLOCKED** — genuine environment or authorization constraint prevents execution.

Do not use PASS to mean that code merely appears correct.

## 13. G1 reassessment

Rebuild all 14 operational checks from `docs/prompt10.md` §9 and map them to `docs/plan4.2.md`.

1. Class A synchronous execution.
2. Class D durable execution.
3. Single canonical invocation service.
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

Prompt 14's independent result was 4 PASS and 10 PARTIAL. Withdraw the earlier 13/14 claim.

Reassess every row against the corrected source digest and executed evidence.

Do not claim overall G1 PASS while a required row remains PARTIAL, FAIL, NOT RUN, or BLOCKED.

G6 retention, production playbook decisions, OSI-4 volumetric semantics, and Phase 4 reconciliation remain outside G1 unless governing architecture explicitly requires otherwise.

## 14. Required documents and evidence

Create exactly two new substantive documents, in addition to code and tests:

### 14.1 Design and root-cause document

`docs/phase1/classd-state-transition-discipline.md`

This document is written before source changes.

It contains the P1 root-cause diagnosis, the state-machine definitions, transition discipline, retry semantics, lock boundaries, fault-injection design, and findings-to-invariant mapping.

### 14.2 Final implementation and re-audit

`docs/phase1/classd-state-transition-reaudit.md`

This is the final implementation report and independent re-audit artifact. It replaces a separate `docs/prompt15_imp.md` report to avoid duplicate and potentially inconsistent reports.

It must contain:

- Executive result.
- Root-cause diagnosis and any revisions.
- Changes for X-01 through X-24.
- Before/after invariant descriptions.
- Test names and actual results.
- Red/green evidence where available.
- Full G1 matrix.
- Security reassessment.
- Migration inventory.
- Evidence and worktree digests.
- Open risks and blockers.
- Change inventory.
- Phase-boundary confirmation.
- Owner decisions still required.

Do not create a separate `docs/prompt15_remediation.md` or `docs/prompt15_imp.md`. Fold that material into the two documents above.

### 14.3 Evidence register

Update `docs/evidence/README.md` with:

- The state-transition discipline document.
- Every important new or strengthened test.
- The final re-audit.
- Exact worktree digests.
- Environment and test commands.
- Pass, fail, skip, and blocked counts.
- The invariant or finding each test protects.

Adopt digest-per-run discipline for future implementation and review cycles.

## 15. Ongoing principle for every future adversarial-review cycle

This requirement is permanent.

After every adversarial review:

1. Diagnose before fixing.
2. Determine whether each finding is a symptom, a distinct defect, or a member of a larger failure class.
3. Write the root-cause diagnosis before changing code.
4. Fix shared mechanisms rather than accumulating local patches.
5. Establish structural discipline that prevents recurrence or reliably detects violations.
6. Add tests that exercise the actual failure boundary.
7. Demonstrate that the test detects the old behavior and validates the correction where practical.
8. Produce a re-audit that argues why the finding is now impossible or reliably caught.
9. Bind every important claim to executed evidence and the exact source state.
10. Preserve unresolved owner decisions rather than inventing values.

This is not a writing preference. It is an engineering requirement.

A future reviewer should evaluate whether the underlying discipline exists, not merely whether the old finding IDs have disappeared.

## 16. Completion criteria

The cycle is complete only when:

1. P0 baseline and source digest are recorded.
2. P1 root-cause diagnosis is documented before code changes.
3. The diagnosis has been independently assessed, not merely adopted.
4. The state-transition discipline is implemented at the appropriate subsystem boundaries.
5. All five HIGH findings are closed with executed adversarial regression evidence.
6. Every X-01 through X-24 finding has a documented, honest disposition.
7. The approved full-scope remediation is completed, not reduced to HIGH-only work.
8. Real PostgreSQL tests cover the database invariants and concurrency.
9. Genuine process-kill tests validate restart/recovery where required.
10. DBOS version affinity has discriminating behavioral evidence.
11. Provider-attempt counts cannot conceal duplicate sends.
12. Audit completeness does not depend on a caller polling the run.
13. CI is configured to run its database-backed tests.
14. Evidence binds the claims to exact source digests.
15. The G1 matrix is rebuilt honestly.
16. The two required documents and evidence register are complete.
17. The `lead-followup` hard stop remains intact.
18. No immutable or owner-authored files were overwritten.
19. No commits or pushes were made.

If a criterion cannot be satisfied, state precisely why and mark the cycle incomplete. Do not manufacture a PASS or weaken a requirement to obtain completion.

## 17. Final instruction

Begin with P0, then produce the root-cause and state-transition design document.

Do not write remediation code before that design document is complete.

After the design is written, continue through the entire approved remediation and verification scope without pausing for routine owner confirmation.

Stop only for a genuine conflict with governing architecture, an unresolved owner decision, unsafe or unauthorized resource mutation, or an invariant violation that cannot be resolved within the authorized scope.

Do not stop after the matrix. Do not stop after the HIGH findings. Do not stop when the tests merely turn green.

**The objective is a structurally defensible Class D framework—not a collection of patches that makes the current review report disappear.**

**The key consolidation:** Prompt 15's state-transition discipline becomes the architectural backbone; the earlier Prompt 15's full X-01–X-24 remediation and approved execution plan remain the scope. The universal transition-function requirement is replaced by subsystem-specific transitions with common invariants, and the cancellation fix explicitly avoids holding database locks across external sends.