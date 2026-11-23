The newly attached `prompt15_z.md` adds a useful improvement: it identifies **two systemic root-cause classes**, rather than treating all 24 findings as unrelated defects.

I would incorporate its RC-1/RC-2 diagnosis, the transition-integrity contract, the single submission choke point, and mandatory red/green evidence. Those are stronger than simply requiring a root-cause matrix.

I would also retain several corrections from our previous consolidation:

- Use subsystem-specific transition operations with common guarantees—not one universal transition function that obscures different state-machine semantics.
- Never hold a PostgreSQL transaction or row lock across an external send.
- Retain the full X-01–X-24 remediation scope, not just the HIGH findings and selected MEDIUM findings.
- Require genuine run attachment for X-08 unless an actual conflict with the governing specification is established.
- Preserve the previously approved environment and execution choices.

The consolidated prompt below supersedes the earlier Prompt 15 drafts.

# Prompt 15 — Structural Remediation of the Class D Framework: Root Causes, Transition Integrity, and Adversarial Proof

| Field                            | Value                                                                                  |
| -------------------------------- | -------------------------------------------------------------------------------------- |
| Repository                       | `github.com/PithomLabs/miniapp`                                                        |
| Role                             | Principal Go Architect, PostgreSQL Engineer, Distributed Systems and Security Engineer |
| Governing architecture           | `docs/plan4.2.md` — immutable                                                          |
| Implementation specification     | `docs/prompt13.md`                                                                     |
| Adversarial review               | `docs/adv_review_prompt14.md`                                                          |
| Historical implementation report | `docs/prompt13_imp.md`                                                                 |
| Remediation findings             | X-01 through X-24                                                                      |
| Objective                        | Establish structural correctness and prove the repaired invariants                     |
| Required final report            | `docs/prompt15_imp.md`                                                                 |

---

## 1. Mission

Prompt 14's independent adversarial review found five HIGH findings, thirteen MEDIUM findings, and six LOW findings. Its independent G1 assessment was 4 PASS and 10 PARTIAL, contradicting the original 13/14 claim.

The review did not recommend an architecture redesign. It identified bounded but important failures in the framework's durable state transitions, recovery behavior, lifecycle reporting, testing, and evidence.

This is not a patch round.

Your job is to:

1. Independently diagnose the systemic defects behind the findings.
2. Establish a normative transition-integrity contract before changing source code.
3. Correct the shared mechanisms that allow the failures to occur.
4. Remediate every actionable finding X-01 through X-24.
5. Add discriminating tests that exercise the actual failure boundaries.
6. Demonstrate that the new tests detect the old behavior and pass with the corrected enforcement.
7. Reassess the capability matrix and all 14 G1 criteria using evidence from the corrected source state.
8. Leave behind an engineering discipline that future implementation and adversarial-review cycles must follow.

**Success is not that the findings disappear. Success is that the underlying class of defect cannot silently recur.**

Do not accept the findings as a task list without understanding their causes. Conversely, do not use a broad root-cause explanation to avoid fixing individual findings.

## 2. Standing post-review engineering principle

This principle applies to this remediation and every future cycle.

After every adversarial review, before any code change:

1. **Cluster, don't tally.** Group findings into defect classes and identify the systemic mechanism that produced every instance.
2. **Derive the contract.** State the invariant or small set of invariants whose uniform enforcement prevents the defect class. Express them as enforceable rules: predicate, atomicity scope, authorization boundary, and recovery behavior.
3. **Map, then remediate.** Every finding must map to a contract rule and at least one discriminating regression test. Supplementary tests may be necessary for additional failure windows.
4. **Fix causes, not symptoms.** A local correction is insufficient if another path can still violate the same invariant.
5. **Closure is red/green, not inspection.** For every HIGH finding and X-07, demonstrate that the relevant test fails with the defective enforcement absent and passes with it restored, binding both executions to exact source digests.
6. **Re-review audits the contract.** The next independent review verifies the contract, its enforcement, and the elimination of the defect class—not merely changes to finding statuses.

This is an engineering requirement, not a writing preference.

## 3. Root-cause synthesis — independently assess before coding

The following is the working diagnosis. Verify it against the actual repository and pinned SDK. Ratify, revise, or reject it in writing before changing code.

### RC-1 — Runtime: invariants enforced by procedural path shape instead of durable transition integrity

The five HIGH findings suggest that important guarantees are enforced by the ordering of ordinary operations and comments rather than by explicit, recoverable, database-arbitrated state transitions.

Four related mechanisms appear responsible.

**RC-1a — Non-predicated writes and last-writer-wins behavior**

Contested durable fields can be overwritten without proving that the previous state still matches the expected state.

Examples:

- X-03: unconditional `run_id` overwrite permits concurrent same-key submissions to adopt different workflow identities.
- X-04: competing approval decisions read and update without adequate row locking or state predicates, permitting silent last-writer-wins behavior.

**RC-1b — Check-then-act across separate transaction boundaries**

A condition is checked in one scope, but the operation relying on that condition occurs after the protection has been released.

Example:

- X-05: `CheckLive` releases its lock before effect reservation. Cancellation can commit in the gap while a new attempt continues toward execution.

**RC-1c — Authorization enforced by procedural ordering instead of at the execution boundary**

The approval requirement is enforced by the path that happens to call the submission method, not by submission itself.

Example:

- X-01: a retry can reach `confirmSubmit` without the approval linkage that the initial request path was supposed to establish.

**RC-1d — Failure branches are not represented as recoverable machine states**

Operations can fail between durable writes, leaving records that have no reliable convergence path or misrepresent their status.

Examples:

- X-02: approval-path failures can strand invocations.
- X-06: signaling nonexistent workflows reports false success.
- X-11: approval recovery depends on behavior not mandated by the framework.
- X-12: settlement audit depends on callers polling.
- X-13: policy refusals leave no durable attributable record.

Do not assume that these four mechanisms exhaust the runtime root causes. Inspect related identity, lifecycle, authorization, and shutdown boundaries.

### RC-2 — Evidence: tests and records demonstrate expected outcomes without always being able to falsify incorrect behavior

Several findings indicate that the evidence mechanisms themselves are inadequate.

- X-07: mock provider records deduplicate by business operation ID, concealing repeated sends.
- X-09: version-affinity tests use indistinguishable code and cannot prove executor attribution.
- X-17: restart-test parents fail without a database, while remote CI does not provision PostgreSQL.
- X-18: important historical evidence lacks exact source digests.
- The HIGH findings were predominantly tested sequentially rather than under the competing interleavings that could break their guarantees.

The underlying issue is not simply insufficient test count. It is that some tests cannot distinguish a correct implementation from the specific defective implementation they are intended to catch.

A passing test is useful only when its assertions would fail under the relevant defect.

### 3.1 Required root-cause matrix

Before source changes, create the following matrix in the normative design document:

| Finding | Defect class | Violated invariant | Root cause | Contract rule | Primary test |
| ------- | ------------ | ------------------ | ---------- | ------------- | ------------ |

Cover X-01 through X-24.

Findings may share one contract rule. However, every finding must have at least one identifiable test that discriminates the defective behavior from the corrected behavior.

Record findings whose original behavior cannot be independently reproduced because the environment lacks the required prerequisites. Do not misrepresent code inspection as execution evidence.

## 4. Governing authority and repository boundaries

Read the actual files before proceeding.

Authority:

1. `docs/plan4.2.md`.
2. Accepted G0 decisions and recorded owner decisions.
3. `docs/prompt13.md`.
4. Existing approved contracts and ADRs.
5. `docs/adv_review_prompt14.md` and its findings.
6. Actual implementation source, tests, migrations, scripts, analyzer fixtures, and CI workflows.
7. Pinned DBOS Go SDK v1.9.0 source.
8. Historical reports and evidence as claims, not proof of current behavior.
9. This prompt as the authorization for remediation.

`docs/plan4.2.md` is immutable.

The adversarial review is authoritative for remediation targets, not for changing architectural decisions.

If remediation appears to require an architecture change, record the exact conflict and request an owner decision. Do not invent a reconciliation.

### 4.1 Protected files and paths

Do not modify:

- `docs/plan4.2.md`.
- `docs/prompt13.md`.
- `docs/prompt13_imp.md`.
- `docs/adv_review_prompt14.md`.
- `docs/faq.md`.
- Prior adversarial-review documents.
- G0 baseline history or accepted G0 evidence.

Preserve all existing tracked modifications and untracked files. Verify the actual repository state before touching files.

Do not use destructive Git operations to simplify the working tree.

### 4.2 Scope

All actionable findings X-01 through X-24 are in scope.

Authorized implementation areas include:

- `runtime/`
- `approvals/`
- `effects/`
- `executor/`
- `contract/`
- `internal/store/`
- `cmd/`
- `transport/`
- `analyzer/`
- `preflight/`
- `testkit/`
- `probe/`
- `scripts/`
- `.github/workflows/`
- `migrations/`

Update `docs/evidence/README.md`, `docs/phase1/classd-capability-matrix.md`, `docs/phase1/classd-framework-readiness.md`, and the version register where required.

Do not add a new miniapp. The neutral Class D probe remains the framework fixture.

### 4.3 Explicitly outside this remediation

Do not introduce:

- Real provider integrations, OAuth, customer credentials, or external production writes.
- Production reconciliation machinery belonging to Phase 4.
- River/Class B or a Class C substrate decision.
- Sprites, untrusted-code execution, or promotion pipelines.
- Fly deployment or G6 retention drills.
- A general BPMN replacement.
- UI or miniapp-generator work.
- A full lifecycle CLI.

A narrowly scoped authenticated kill-switch operator command is allowed because X-15 requires an operational control surface. It must reuse canonical authorization.

### 4.4 `lead-followup` hard stop

The ADR remains DRAFT and unapproved.

No `lead-followup` implementation, manifest, registration, execution, or ADR approval is authorized.

Do not invent its sales semantics or owner decisions.

Preserve OSI-1 through OSI-4 and the Prompt 13 §22 decisions as open where applicable.

## 5. Approved execution decisions

Proceed in one continuous pass through baseline, diagnosis, implementation, adversarial validation, and close-out.

Do not stop after the root-cause matrix or after fixing only HIGH findings.

### 5.1 Local execution

Use local Docker/PostgreSQL with appropriate host permissions for database, concurrency, and real process-kill tests.

Do not weaken host-wide security settings or access unrelated data.

### 5.2 Neon

Best effort only:

1. Perform a read-only availability and branch-identity probe.
2. Use `phase0/g0-spikes` only if its identity and test-only authorization are verified.
3. Do not create a project or another branch.
4. Do not mutate a shared or production branch.
5. Do not destroy, recreate, or reset Neon resources.
6. If unreachable or unsuitable, mark Neon BLOCKED and continue with local PostgreSQL.

Use disposable local databases or explicitly authorized test resources.

### 5.3 OpenCode E2E

Repeat Prompt 13's evidence scope:

- Transport.
- MCP.
- One genuine agent turn.

Do not expand into a multi-turn scenario.

Record OpenCode version, model/version, source digest, invocation/run identifiers, and corresponding server-side audit evidence. Redact credentials.

State exactly what the demo proves and does not prove.

### 5.4 No commits or pushes

All work remains uncommitted. No remote mutation is authorized.

## 6. P0 — Baseline

Before any implementation change, record:

- HEAD, branch, remote configuration, and relevant history.
- Staged, modified, and untracked paths.
- Go, PostgreSQL, DBOS SDK, and dependency versions.
- Docker and database prerequisites.
- Full baseline test results.
- A deterministic initial worktree digest.

Prompt 14 recorded HEAD `5cf9627` over G0 baseline `e6b5b7b`. Verify actual state and report drift.

Do not discard current changes to reproduce the review snapshot.

Run the available baseline checks:

```
gofmt -l .
go vet ./...
go test ./... -count=1
bash scripts/checks.sh
```

Run DB-backed tests when prerequisites are available.

Label outcomes accurately: PASS, FAIL, SKIP, BLOCKED, or NOT RUN.

Record source digests before and after the baseline run.

No source implementation changes before P1 is complete.

## 7. P1 — Normative transition-integrity contract

Create:

`docs/phase1/invocation-lifecycle-contract.md`

This is the primary design deliverable and must be completed before code changes.

### 7.1 State-machine specification

Specify durable lifecycle state machines for:

- Invocations.
- Approvals.
- Effects and effect attempts.
- Workflow/run identity and lifecycle.
- Durable sequences and attachments.
- Cancellation.

For each subsystem, document:

- All enumerable states.
- Permitted state transitions.
- Source state and exact transition predicate.
- Atomicity and transaction boundary.
- Required authorization precondition.
- Database locking or concurrency-arbitration mechanism.
- Required audit event.
- Expected failure behavior.
- Retry and recovery behavior.
- Terminal states.
- Invariants that must hold in every reachable state.
- Owning implementation component.

### 7.2 Transition-integrity rules

The following rules are normative.

**Rule 1 — Predicated single-writer transitions**

Every contested lifecycle mutation must prove its expected prior state.

Use a conditional update, row-lock protocol, or equivalent database arbitration. Check affected rows. A losing transition must reload, reconcile, or return a defined conflict.

Never use unconditional last-writer-wins updates for contested lifecycle state.

For `run_id`, adopt the canonical value rather than replacing an existing one.

**Rule 2 — Guard-and-act consistency**

A guard must remain authoritative through the durable operation it protects.

The guard and act must share a transaction or a proven synchronization protocol.

For cancellation/effect claiming, the effect reservation is the critical database boundary. Cancellation and claim must serialize against each other.

**Rule 3 — Authorization at the execution boundary**

All executor submissions must pass through one authoritative choke point.

The choke point must resolve the invocation's actual policy and approval requirements from trusted persisted state. It must reject submission when a required approval is missing, invalid, ungranted, or unrelated to that invocation.

No replay, retry, poll, or alternate caller may bypass it.

**Rule 4 — No unreachable operational states**

Every nonterminal state must have a defined convergence path or an explicit operator-actionable exception.

Every retry re-enters the state machine, reads current persisted state, and chooses its behavior from that state.

A retry flag must not determine whether authorization or lifecycle requirements are enforced.

**Rule 5 — Truthful lifecycle and audit reporting**

Do not report success for operations whose target does not exist or whose required state was not established.

Persist decision audit at the durable decision boundary.

Persist settlement independently of Status polling.

Record policy refusals as attributable events distinct from provider attempts.

**Rule 6 — Falsifiable evidence**

A claim ships only with a discriminating test.

Mocks must measure actual operations without hiding duplicates.

Version-affinity tests must distinguish the executor that ran a workflow.

Every important result must bind to an exact source digest.

### 7.3 Subsystem-specific transition operations

Implement subsystem-specific transition functions or methods with the common properties above.

A single universal `transition()` function is not required and must not be forced onto subsystems whose state, transaction, or concurrency semantics differ.

However:

- Every durable lifecycle mutation must have a documented transition rule.
- Existing ad-hoc updates in affected lifecycle paths must be replaced by the appropriate transition operation.
- Any exception must have a written explanation and a test where relevant.
- Do not retain an unpredicated special case merely because it is inconvenient to refactor.

Do not build an abstraction that hides authorization, expected-state predicates, transaction ownership, or failure handling.

### 7.4 Retry-as-re-entry

All retryable paths must load current persisted state and dispatch accordingly.

Include:

- Invocation submit/replay.
- Approval creation/poll.
- Submit-if-missing recovery.
- Workflow signal redelivery.
- Effect claim/settlement.
- Cancellation redelivery.
- Run attachment.
- Other recovery paths discovered during investigation.

There must be a handler for each state that a legitimate retry can encounter.

### 7.5 Lock discipline and external effects

Never hold a PostgreSQL transaction, row lock, or advisory transaction lock across an external provider request, DBOS submission, long wait, or signal wait.

For the effect/cancel race:

- If cancellation commits first, a new effect reservation must fail.
- If effect reservation commits first, that attempt is already in flight and must be visible to cancellation.
- An in-flight effect settles from evidence.
- A stale worker must not send or overwrite a newer owner.
- No failed reservation may create a false ATTEMPTING state.

Preserve R2:

- No duplicate confirmed business effects.
- No silently lost operations.
- No blind retries of unknown outcomes.

Do not claim exactly-once provider delivery.

### 7.6 Submission choke-point proof

Document the exact submission function and its inputs.

The approval linkage must be part of the decision, not an optional caller convenience.

Prove every DBOS executor submission reaches this choke point using source-level inspection, call-graph analysis, grep or an equivalent automated test.

A test must demonstrate rejection when a gated invocation has no valid approval linkage.

### 7.7 Contract table

Include a transition table that covers each relevant lifecycle state crossed with the events that can change it:

- Invoke.
- Same-key retry.
- Park.
- Submit-after-park.
- Grant.
- Deny.
- Expire.
- Decide.
- Concurrent decide.
- Cancel.
- Decide versus cancel.
- Check-live.
- Claim.
- Send.
- Settle.
- Signal.
- Status.
- Completion.

For each table row specify:

1. Expected source state.
2. Destination state.
3. Exact predicate or lock protocol.
4. Transaction/atomicity scope.
5. Authorization condition.
6. Audit requirement.
7. Failure and crash exit.
8. Recovery behavior.
9. Owning component.
10. Primary discriminating test.

**No lifecycle correction should ship without a contract row. No contract row should claim enforcement without a test.**

## 8. P2 — Remediation in dependency order

Do not have independent agents or parallel edits modify shared runtime state-machine code concurrently.

### Workstream A — Invocation and approval correctness

**X-01, X-02, X-03, X-04**

Investigate:

- `runtime/service.go`
- `runtime/store.go`
- `runtime/lifecycle.go`
- `approvals/`
- `executor/dbos/`
- Associated migrations and tests.

#### X-01 — Approval bypass on retry

Make the submission choke point enforce approval linkage.

A retry must adopt an existing pending approval or re-enter legitimate approval handling. Never submit blindly.

The workflow itself must fail closed if required approval information is missing.

Test the failures of approval creation, park persistence, and crash between claim and park.

#### X-02 — Approval-path failures

Make every approval-path failure recoverable.

Investigate approval-row creation, approval linkage, audit, run-ID persistence, DBOS submission, polling, grant, denial, expiration, and cancellation.

Polling must submit-if-missing using the stored approval linkage before polling an eligible pending workflow.

A denial or expiration of a workflow that was never submitted must still settle the invocation correctly.

No ordinary transient failure may leave a permanently waiting invocation.

#### X-03 — Concurrent same-key submissions

Use database arbitration for canonical run-ID adoption and workflow submission.

Guarantee one canonical run ID and no unaddressable workflow.

Use separate connections and a barrier-synchronized concurrency test.

Verify the exact pinned DBOS duplicate-submission behavior.

#### X-04 — Approval decision races

Make approval decision transitions database-atomic.

Apply equivalent predicate discipline to decision, cancellation, and expiration.

Test concurrent approve/reject, decide/cancel, self-approval races, and expiration/decision races.

Only one conflicting state transition can win.

### Workstream B — Cancellation and effect claim

**X-05**

Repair the claim boundary so cancellation and a new effect reservation cannot cross each other without a defined database order.

Do not merely repeat `CheckLive` before the existing claim.

Test claim-before-cancel and cancel-before-claim, policy-evaluation overlap, and stale-worker takeover.

The actual reservation must be visible to the cancellation response if it committed first.

### Workstream C — Mock provider attempt integrity

**X-07**

Make the mock log every send attempt independently.

Remove deduplication that hides multiple provider calls.

Record enough information to distinguish attempts, acceptance, outcome, and operation identity.

Add a deliberate double-send test that produces two measurable attempts.

Correct false exactly-once comments and assertions.

Run all affected effect and probe tests again.

### Workstream D — Lifecycle and shutdown truth

**X-06, X-11, X-12, X-13, X-14**

#### X-06 — Signal to unknown workflow

Inspect pinned DBOS SDK behavior.

Verify existence and correct family routing before reporting a signal accepted.

Test unknown IDs, cross-family routing, valid signals after restart, completed workflows, and unavailable/incompatible runs.

If existence-check and signal operations cannot be atomic, document the remaining race and make the result/recovery semantics truthful.

#### X-11 — Bounded approval recovery

Implement one framework-enforced recovery mechanism.

Prefer a framework-owned approval-wait abstraction with a mandatory bounded observation interval and authoritative database re-read, provided this can be enforced without allowing miniapps to bypass it through raw receive calls.

If that cannot be made enforceable, choose another bounded mechanism based on the actual SDK capabilities. Document why it was selected.

Do not introduce the Phase 4 production reconciler.

Audit approval decisions when they commit, not only after successful signal delivery.

Correct any nonexistent "durable mailbox" wording.

#### X-12 — Settlement audit

Move settlement recording into a framework-owned completion path, or another proven durable mechanism.

A completed or failed run must have its settlement audit without requiring Status polling.

Test recovery around the completion/audit boundary.

#### X-13 — Refusal audit

Record refusals due to kill switches, grant revocation, invalid tokens, and other applicable policy failures.

Include safe attribution and distinguish refusal from provider attempt.

Do not create false ATTEMPTING records for operations that were never attempted.

#### X-14 — Safe shutdown

Inspect:

- `cmd/miniapp/main.go`
- `testkit/stack/stack.go`
- Executor shutdown.
- Shared database pool lifecycle.

A failed or timed-out executor shutdown must never be followed by a silent pool close while workers may still be using the pool.

Preserve diagnostics and use a clearly reported termination path that does not close a still-needed pool.

Add a fault-injected shutdown failure test that proves the safety property.

### Workstream E — Run attachment and version safety

**X-08, X-09, X-10**

#### X-08 — True run attachment

Implement genuine attachment to the canonical existing workflow.

The losing invocation must:

- Reference the actual target run.
- Preserve tenant and invocation identity.
- Return a valid handle.
- Reflect target-run status.
- Have defined cancellation behavior.
- Not report success while the target sequence remains active.

Connect superseded workflow state to cancellation so an obsolete generation cannot continue producing effects.

Do not silently downgrade true attachment to detect-and-exit.

If inspection proves that the governing contract makes genuine attachment impossible, stop that specific decision and document the exact conflict for owner review. Do not invent replacement semantics.

Use barrier-synchronized concurrent sequence claims.

#### X-09 — Behavioral version affinity

Create a discriminating test with version-specific workflow evidence:

1. Create a pending v1 workflow under v1.
2. Stop v1.
3. Run only v2.
4. Prove v2 cannot execute the v1 workflow.
5. Restart v1 and prove the correct version recovers it.

Source inspection supports but does not substitute for behavioral proof.

#### X-10 — Foreign-version visibility

Expose relevant nonterminal work across the DBOS statuses supported by the pinned SDK.

Cover PENDING, ENQUEUED, DELAYED, SUSPENDED, and other applicable states.

Provide queryable operator visibility where practical. A documented log-alert contract is acceptable only if a queryable surface would require disproportionate or out-of-scope architecture.

Do not claim G6 artifact-retention completion.

### Workstream F — Kill-switch control-plane security

**X-15**

Separate control-plane privileges from ordinary miniapp SQL.

Miniapp code must not directly update the kill-switch table.

Use appropriate role separation or a narrowly scoped privileged database function. Inspect ownership, privileges, search path, and caller identity carefully.

Persist every authorized switch change with audit attribution in a consistent operation.

Provide a minimal authenticated operator command using canonical authorization.

Add a forward-only migration. Use the next valid migration number after inspecting the current migration history; do not assume a number or rewrite an applied migration.

Test direct handler denial, unauthorized mutation denial, authorized operation, and durable audit.

### Workstream G — Determinism analyzer

**X-16**

Fix named-step versus inline-step traversal inconsistencies.

Add positive/negative fixtures for:

- `RunAsTransaction` and `Enqueue` in forbidden contexts.
- `uuid.NewString()` and relevant non-stdlib nondeterminism.
- Depth-two and cross-package helpers.
- `errgroup.Go`.
- `reflect.Select`.
- `context.WithTimeout`.
- Map/iterator behavior.
- Valid DBOS APIs.

Document the actual supported analysis boundary. Do not claim general determinism proof from AST pattern matching.

Reconcile fixture counts and analyzer documentation.

### Workstream H — CI correctness

**X-17**

Gate database-dependent restart-test parents before spawning children.

Missing DB prerequisites must produce fast SKIP. A configured but broken database must FAIL.

Add a pinned PostgreSQL service and health check to the CI integration workflow.

Do not weaken or delete tests.

Do not push to trigger remote CI. Remote execution remains unobserved until it actually happens.

### Workstream I — Evidence integrity

**X-18**

Adopt deterministic worktree digests for every important test run.

Include tracked modifications and relevant untracked implementation files. Use a sorted file-hash manifest or another reproducible method.

Record before and after each important run:

- HEAD and branch.
- Source digest.
- Exact command.
- Environment and database version.
- PASS/FAIL/SKIP/BLOCKED/NOT RUN counts.
- Any fixture or source modifications.
- What the result proves and does not prove.

Correct the missing P0 full-list reference in the readiness documentation.

Do not retroactively claim identical local and Neon source trees without evidence.

### Workstream J — Identity, preflight, repository, and contract hardening

**X-19 through X-24**

#### X-19 — Unambiguous entity identity

Replace ambiguous tenant/entity string concatenation with an unambiguous representation.

Use the same canonical implementation for locking, comparison, and persistence.

Add collision-pair tests.

#### X-20 — ADR preflight

An approved-but-not-yet-registered ADR must not be incorrectly classified as a hard failure.

Document the boundary between ADR preflight, registry admission, manifest validation, and runtime enforcement.

#### X-21 — Repository gates and dependencies

Expand package coverage in `scripts/checks.sh`.

Update direct dependency pins and version-register records according to the actual module graph.

Add fixtures proving newly added packages cannot bypass import restrictions.

#### X-22 — Empty run IDs

Reject empty IDs at Signal and Cancel service boundaries.

Verify and enforce the intended uniqueness of nonempty run IDs, using a forward-only migration where needed.

#### X-23 — Declared but unenforced contracts

Audit:

- Manifest `Timeout`.
- Unreachable `CANCELLED`.
- `NOT_APPLIED → ATTEMPTING`.
- Superseded probe workflows.

Implement, enforce, or explicitly retire each surface where governing specifications permit. Do not retain misleading declarations.

#### X-24 — Additional hardening

Resolve all six subfindings:

a. Do not silently swallow Status synchronization errors.

b. Authenticate before exposing registry existence where the governing policy permits.

c. Limit REST request-body size and test oversized requests.

d. Reject trailing JSON values and garbage. Use strict decoding that verifies EOF after the expected document; do not rely solely on `dec.More()` for top-level trailing data.

e. Record and test actual tenant attribution in mock provider attempts.

f. Bound or replace the process-local run-routing map. Durable workflow identity remains authoritative.

Also scan recorded audit events for known test secrets and credential patterns.

## 9. Fault-injection and adversarial test discipline

Extend the testkit with a test-only `FaultInjector` that can fail a named operation on its Nth attempt.

Use it at meaningful durable boundaries, not indiscriminately on every SQL statement.

Required fault points include:

- Approval creation.
- Approval linkage.
- Park update/audit.
- Run-ID persistence.
- DBOS submission.
- Approval decision commit.
- Signal transmission.
- Effect reservation.
- Provider send.
- Settlement.
- Completion audit.
- Shutdown.

### 9.1 Red/green evidence

For every HIGH finding and X-07, demonstrate:

1. The discriminating test fails when the relevant defective enforcement is absent.
2. The test passes when the corrected enforcement is restored.

Use an isolated pre-remediation snapshot or a controlled reversible test variant. Do not damage the active working tree, rewrite history, or destroy evidence to produce a red result.

Bind both runs to exact source digests.

Where an authentic red run cannot be produced safely, document the limitation and do not claim full red/green closure.

### 9.2 Concurrency tests

Use independent PostgreSQL connections and deterministic barriers, not sleep-based race tests.

At minimum:

- Concurrent same-key submission.
- Competing approval decisions.
- Decide versus cancellation.
- Approval versus expiration.
- Cancellation versus claim.
- Cancellation versus in-flight attempt.
- Stale lease takeover versus stale worker resumption.
- Sequence-claim race.
- Cross-tenant identity collision/isolation.

Run relevant cases under `-race`.

### 9.3 Process-kill tests

Use the genuine Prompt 13 process-kill harness.

Do not substitute mocks for real child-process termination and recovery.

Test crashes between important durable writes and around submission, approval signal, effect execution, settlement, and shutdown.

### 9.4 Test boundaries

Database invariants require real PostgreSQL evidence.

Pure functions and analyzer AST rules may use unit tests where that is the correct boundary.

Every new test must assert an invariant rather than merely exercise a code path.

## 10. Required regression coverage

At minimum, add or strengthen tests for:

| Test                                        | Finding | Required proof                                    |
| ------------------------------------------- | ------- | ------------------------------------------------- |
| `TestApprovalRetryAfterSubmitFailure`       | X-01    | Retry re-enters approval; no unapproved execution |
| `TestApprovalRetryAfterParkFailure`         | X-01    | Park failure cannot bypass approval               |
| Approval crash-between-claim-and-park test  | X-01    | Recovery remains gated                            |
| `TestApprovalPollSubmitsIfMissing`          | X-02    | Poll safely submits missing run                   |
| `TestApprovalDenialSettlesAfterParkFailure` | X-02    | Denial converges                                  |
| `TestConcurrentSubmitSameKey`               | X-03    | One canonical workflow and handle                 |
| `TestConcurrentApproveReject`               | X-04    | Exactly one conflicting decision wins             |
| `TestConcurrentDecideAndCancel`             | X-04    | Decision cannot overwrite cancellation            |
| `TestCancelDuringClaim`                     | X-05    | Post-cancel reservation is rejected               |
| `TestCancelDuringAttempt`                   | X-05    | Pre-cancel in-flight attempt is visible           |
| `TestSignalUnknownRun`                      | X-06    | Unknown run returns error                         |
| `TestMockRecordsEveryAttempt`               | X-07    | Double-send is measured as two attempts           |
| `TestProbeAttachLifecycle`                  | X-08    | Attachment follows target workflow lifecycle      |
| `TestVersionV2DoesNotExecuteV1Run`          | X-09    | Version affinity is behaviorally discriminated    |
| `TestForeignVersionNonterminalVisibility`   | X-10    | Relevant states are exposed                       |
| `TestApprovalLostSignalBoundedRecovery`     | X-11    | Lost signal recovers within framework bound       |
| `TestSettlementAuditedWithoutPoll`          | X-12    | Settlement audit independent of Status            |
| `TestRefusalAudited`                        | X-13    | Policy refusal has durable attribution            |
| `TestShutdownFailurePreservesPool`          | X-14    | Active workers never lose their pool prematurely  |
| `TestKillSwitchNotHandlerWritable`          | X-15    | Handler cannot change safety controls             |
| Analyzer positive/negative fixtures         | X-16    | Supported rules are pinned by tests               |
| DB-less restart test                        | X-17    | Missing DB produces fast SKIP                     |
| Evidence digest test/script                 | X-18    | Exact source state is reproducible                |
| Entity collision test                       | X-19    | Distinct identities remain distinct               |
| Approved-unregistered ADR test              | X-20    | Correct non-failure status                        |
| Repository gate fixture                     | X-21    | New packages cannot bypass restrictions           |
| Empty run ID test                           | X-22    | Signal/Cancel reject empty identifiers            |
| Contract-state coverage                     | X-23    | Declared behavior is implemented or retired       |
| Hardening tests                             | X-24a–f | Each subfinding is independently tested           |

The exact test names may differ to match repository conventions, but every requirement must map to actual assertions and test files.

## 11. Database and migration requirements

All schema and privilege changes are forward-only.

1. Inspect the actual migration sequence before choosing a new number.
2. Never rewrite an applied migration.
3. Test fresh install and upgrade from relevant prior schema states.
4. Record rollback limitations and intentionally irreversible changes.
5. Preserve RLS and explicit tenant predicates.
6. Never hold transactions or locks across external calls or durable waits.
7. Do not modify DBOS-owned tables.
8. Do not mutate unverified Neon state.
9. Verify uniqueness constraints against the intended tenant and identity model.

X-15 and X-22 may require migrations. Bundle changes only where doing so preserves clear migration ownership and upgrade behavior.

## 12. P3 — Full adversarial validation

After implementation, exercise the repaired system under:

### Concurrent operation

All races specified in §9.2, using barriers and separate connections.

### Failure between durable operations

Inject failures at the meaningful boundaries in §9 and verify safe re-entry and convergence.

### Process failure

Use genuine kill/restart execution for approval, submission, effects, completion, and shutdown.

### Security

Verify:

- Server-derived tenant.
- Authority/agent principal separation.
- Auth-before-dispatch.
- Approval binding and self-approval rejection.
- Effect-time policy and grant checks.
- Kill-switch privilege separation.
- Cross-tenant invocation, run, approval, and effect isolation.
- Secret-free workflow inputs, checkpoints, logs, and audit events.

### SDK behavior

Verify all DBOS claims against v1.9.0 source and behavior, particularly duplicate submit, workflow retrieval, signal behavior, version affinity, status filtering, and shutdown.

Do not overstate what the SDK guarantees.

## 13. P4 — Verification battery

Run:

```
gofmt -l .
go vet ./...
go test ./... -count=1
go test -race ./...
bash scripts/checks.sh
go run ./cmd/preflight --check-all
```

Also run every targeted regression test, migration suite, DBOS recovery test, concurrency test, transport/MCP test, and OpenCode E2E.

Report separate evidence for:

1. Static checks.
2. Unit tests.
3. Local PostgreSQL integration.
4. Concurrency and race tests.
5. Process-kill recovery.
6. Migration fresh-install and upgrade.
7. Analyzer and preflight.
8. REST/MCP/CLI integration.
9. OpenCode E2E.
10. Neon, if authorized and available.
11. Remote CI, only after an actual run.
12. Skipped, failed, blocked, and not-run tests.

A skipped test is not PASS. A source inspection is not execution. A local CI-equivalent run is not a remote CI result.

Do not weaken assertions or add broad skips.

## 14. Acceptance matrix

Reassess all findings:

| Finding | Required outcome                                           |
| ------- | ---------------------------------------------------------- |
| X-01    | Approval cannot be bypassed on retry                       |
| X-02    | Approval-path failure converges                            |
| X-03    | Concurrent same-key submit produces one canonical workflow |
| X-04    | Conflicting decisions serialize                            |
| X-05    | No new effect reservation after durable cancel             |
| X-06    | Unknown signal fails; known signal routes correctly        |
| X-07    | Every provider attempt is independently measured           |
| X-08    | True run attachment has truthful lifecycle                 |
| X-09    | Version affinity has behavioral proof                      |
| X-10    | Foreign-version nonterminal work is visible                |
| X-11    | Approval recovery is framework-bounded                     |
| X-12    | Settlement audit does not require polling                  |
| X-13    | Refusals are attributable and audited                      |
| X-14    | Shutdown failure preserves active pool safety              |
| X-15    | Kill-switch control plane is protected and audited         |
| X-16    | Analyzer behavior is consistent and documented             |
| X-17    | DB tests skip correctly; CI provisions PostgreSQL          |
| X-18    | Evidence binds to exact source digests                     |
| X-19    | Entity identity encoding is unambiguous                    |
| X-20    | ADR preflight handles approved-unregistered correctly      |
| X-21    | Repository checks cover new packages and dependencies      |
| X-22    | Empty run IDs are rejected; nonempty identity is unique    |
| X-23    | Declared contract semantics are enforced or retired        |
| X-24a–f | All six additional hardening findings are resolved         |

Use:

- **PASS:** implemented and supported by executed regression evidence.
- **PARTIAL:** implementation exists but required proof or guarantee remains incomplete.
- **FAIL:** invariant remains violated.
- **NOT RUN:** verification was not performed.
- **BLOCKED:** a genuine environment or authorization constraint prevents execution.

Do not use PASS merely because code appears correct.

## 15. Rebuild the capability and G1 matrices

Update:

- `docs/phase1/classd-capability-matrix.md`
- `docs/phase1/classd-framework-readiness.md`

Reassess all F-01 through F-14 claims against the final source state.

Rebuild all 14 G1 operational criteria from `docs/prompt10.md` §9 and map each correctly to `docs/plan4.2.md`:

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

Prompt 14 found 4 PASS and 10 PARTIAL. Withdraw the earlier 13/14 claim.

Do not claim overall G1 PASS if a required row remains PARTIAL, FAIL, NOT RUN, or BLOCKED.

Remote CI remains unobserved until an actual remote run occurs. Do not push to obtain that result.

G6 retention, production playbook decisions, OSI-4 volumetric semantics, and Phase 4 reconciliation remain outside scope.

## 16. Evidence integrity

Update `docs/evidence/README.md`.

Every important evidence entry must identify:

- Source-tree digest.
- HEAD and branch.
- Test command and environment.
- PostgreSQL/DBOS version where applicable.
- PASS/FAIL/SKIP counts.
- Finding and invariant addressed.
- Whether the execution was local, Neon, or remote CI.

The digest must include tracked modifications and relevant untracked implementation files. Use a deterministic sorted hash manifest or another reproducible approach.

Record before and after important runs.

Correct the missing P0 full-list evidence reference.

For red/green tests, record the source digest and result of both the defective-enforcement and corrected-enforcement runs.

Do not rewrite historical evidence as though an old, undigested tree can now be reconstructed.

## 17. Required documents

Create exactly these two new substantive documents, in addition to implementation and tests:

### 17.1 Normative contract

`docs/phase1/invocation-lifecycle-contract.md`

Written before code changes.

Contains:

- Root-cause synthesis and independent assessment.
- RC-1/RC-2 mapping.
- State-machine definitions.
- Transition-integrity rules.
- Complete transition table.
- Submission choke-point specification and proof strategy.
- Retry-as-re-entry design.
- Database lock and atomicity boundaries.
- Fault-injection design.
- Mapping from every finding to contract rule and primary discriminating test.

### 17.2 Implementation report and re-audit

`docs/prompt15_imp.md`

Contains:

A. Executive result and self-assessed disposition.

B. Root-cause diagnosis and any changes to the initial hypothesis.

C. Root-cause-to-contract mapping for X-01 through X-24.

D. Per-finding closure evidence, including red/green results where required.

E. The shipped contract table and choke-point proof.

F. Changes to the capability matrix and complete G1 reassessment.

G. Exact test commands and results, separated by environment.

H. Source digests and evidence-register references.

I. Migrations and database changes.

J. Security assessment.

K. Exact change inventory matching Git status.

L. Residuals, blockers, and owner decisions.

M. Explicit phase-boundary and `lead-followup` hard-stop confirmation.

Do not create overlapping alternative documents such as `prompt15_remediation.md`, `classd-state-transition-discipline.md`, or a separate duplicate re-audit.

## 18. Completion criteria

The cycle is complete only when:

1. P0 baseline and digest are recorded.
2. The root-cause synthesis is independently assessed.
3. The normative contract is written before source changes.
4. All five HIGH findings are closed with executed adversarial evidence.
5. Every X-01 through X-24 finding has an explicit, honest disposition.
6. All mandatory full-scope remediations are completed or explicitly blocked with justification.
7. Database invariants have real PostgreSQL regression evidence.
8. Required process-kill tests use the actual harness.
9. Version affinity has discriminating behavior tests.
10. The provider mock cannot hide duplicate sends.
11. Settlement audit does not depend on Status polling.
12. Shutdown error handling cannot silently close a pool under active workers.
13. CI is configured to run DB-backed tests.
14. Evidence identifies the exact source state tested.
15. The capability and G1 matrices are updated honestly.
16. `docs/prompt15_imp.md` is complete.
17. The `lead-followup` hard stop remains intact.
18. No governing or owner-authored document has been overwritten.
19. No commits or pushes have been made.

If a criterion cannot be satisfied, state exactly why and mark the cycle incomplete. Never manufacture a PASS or weaken an invariant.

## 19. Final instruction

Start with P0. Then independently diagnose RC-1 and RC-2 and write `docs/phase1/invocation-lifecycle-contract.md`.

Do not change implementation code before that document is complete.

Afterward proceed through the entire approved remediation and verification scope without pausing for routine confirmation.

Stop only for a genuine conflict with governing architecture, an unresolved owner decision, unsafe or unauthorized resource mutation, or an invariant violation that cannot be resolved within the authorized scope.

The five HIGH findings are not five isolated bugs. They reveal failures in how the framework protects durable state under concurrency, retries, and crashes. The evidence findings reveal failures in how the framework proves its own guarantees.

**Repair the transition discipline. Repair the evidence discipline. Then prove that both work.**

The objective is a structurally defensible Class D framework—not a collection of patches that makes the current review report disappear.

**Important net decisions:** this version adopts the strongest parts of `prompt15_z.md`, keeps the wider remediation scope from our earlier Prompt 15, and avoids the risky requirement that every subsystem use one identical transition function. It also keeps X-08 as true attachment, makes X-11 framework-enforced, and preserves the approved no-push and Neon safeguards.