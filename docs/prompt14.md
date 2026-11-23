The review should be substantially more demanding than checking whether the implementation report is complete. Prompt 13's agent claims that nearly all framework capabilities are proven, including real-PostgreSQL execution, DBOS recovery, approval recovery, effects safety, and 13 of 14 G1 checks.

The critical task is to establish whether those claims survive independent examination of the actual code, tests, migrations, and evidence—not just whether the agent can point to passing test names.

Here is the adversarial review prompt.

# Prompt 14 — Independent Adversarial Code Review of Prompt 13 Implementation

## 1. Role and mission

Act as an independent Principal Architect, Go runtime engineer, PostgreSQL specialist, and distributed-systems security reviewer.

You are reviewing the implementation delivered under:

- `prompt13.md` — the governing implementation specification.
- `prompt13_imp.md` — the implementation agent's completion report and claims.

The repository is `github.com/PithomLabs/miniapp`.

Your responsibility is to establish whether the implementation actually satisfies Prompt 13 and the governing architecture in `docs/plan4.2.md`.

This is an independent code review, not a continuation of implementation.

**Do not trust the implementation report as evidence. Treat every statement marked PROVEN as a hypothesis until independently verified against source code, tests, execution records, and repository state.**

The central question is:

**Has the implementation genuinely established a reusable, secure, recoverable Class D execution framework, or has it merely assembled enough passing tests and documentation to appear ready?**

Pay particular attention to distributed-systems failure windows, concurrency races, transaction boundaries, tenant isolation, DBOS version affinity, and the distinction between a successful database state transition and a successful external business effect.

The goal is not to criticize `lead-followup` for being a relatively simple workflow. It is to determine whether the framework can support substantially more demanding workflows without requiring each miniapp to reimplement its infrastructure.

## 2. Governing authority and precedence

Read the actual repository files:

1. `docs/plan4.2.md` — immutable architectural authority.
2. `docs/prompt10.md` — original Phase 1 implementation specification and operational G1 checks.
3. `docs/phase0/g0-core-adjudication.md`.
4. `docs/phase0/g0-evidence.md`.
5. `docs/evidence/README.md`.
6. `docs/phase0/determinism-rules.md`.
7. `docs/phase1/classd-capability-matrix.md`.
8. `docs/phase1/classd-framework-readiness.md`.
9. `docs/phase1/open-spec-items.md`.
10. The original `lead-followup` ADR and its reviews.
11. All implementation source, migrations, tests, scripts, analyzer fixtures, CI workflows, and OpenCode demonstration evidence.

Also read the two attached materials:

- `prompt13.md`
- `prompt13_imp.md`

If any referenced file or path does not exist, record that fact. Do not silently substitute a different file or invent a source reference.

### Precedence

Use this order when evaluating conflicts:

1. `docs/plan4.2.md`.
2. Accepted G0 decisions and recorded owner decisions.
3. `prompt13.md` as the authorized Phase 1 implementation specification.
4. Existing approved contracts and ADRs.
5. Repository implementation and executed evidence.
6. `prompt13_imp.md` as the implementation agent's claims.

The implementation report does not amend the architecture or authorize a deviation.

Do not modify `plan4.2.md`.

## 3. Absolute review-only restrictions

This is a read-only adversarial review.

You may create exactly one new review report:

`docs/phase1/classd-framework-adversarial-review.md`

Do not:

- Modify production or framework code.
- Modify tests or fixtures.
- Modify migrations.
- Modify manifests or registrations.
- Modify CI or scripts.
- Modify `plan4.2.md`.
- Modify `prompt13_imp.md` or `prompt13.md`.
- Fix findings discovered during the review.
- Create or register `lead-followup`.
- Execute any `lead-followup` workflow.
- Approve or sign its ADR.
- Perform real provider calls or external writes.
- Modify production or shared Neon state.
- Create, destroy, reset, or migrate the existing Neon project or branch.
- Commit or push changes.
- Clean, reset, stash, overwrite, or revert existing uncommitted work.

You may inspect code, run static analysis and safe read-only commands, inspect test source, examine existing evidence, and execute tests only where the environment is demonstrably disposable and the run cannot damage existing data.

Do not rerun destructive migration or database recovery experiments against the shared Neon environment.

The review must leave the working tree unchanged except for the review report.

## 4. Establish the review baseline

Before reviewing findings:

1. Record current HEAD, branch, remote tracking status, and working-tree status.
2. Record all modified, staged, and untracked paths.
3. Identify the exact source state corresponding to the Prompt 13 implementation evidence.
4. Determine whether the implementation report's test results are bound to a commit, working-tree digest, or merely an unspecified dirty tree.
5. Identify any differences between the reported files changed and the actual repository diff.
6. Verify the original G0 baseline and the accepted G0 commit remain intact.
7. Verify `docs/plan4.2.md` and other immutable governing documents are unchanged.
8. Identify pre-existing user-owned untracked files and distinguish them from implementation-agent additions.

Do not assume the implementation agent's reported change inventory is correct.

### Critical evidence rule

A test result is meaningful only when its source state and environment can be identified.

For every important test claim, establish:

- Source revision or exact working-tree digest.
- Test file and test name.
- Exact command.
- Database/environment identity.
- Actual pass, fail, and skip counts.
- Whether the test exercised the claimed behavior.
- Whether the result can be reproduced from the current tree.

If evidence predates the implementation being reviewed, it cannot prove the current implementation.

If two test runs occurred in different environments, do not treat their results as contradictory without examining the environment differences.

## 5. Executive claims to challenge

The implementation report claims:

- Class D framework capabilities are proven end to end on local Docker PostgreSQL 19 and in a separate Neon pass.
- F-01 through F-14 are PROVEN except F-06, which is PARTIALLY PROVEN.
- Canonical invocation and submit idempotency are correct.
- Entity-key validation, tenant isolation, and Class A serialization work.
- Class D attachment and durable-sequence identity work.
- The effects ledger is race-hardened.
- Cancellation is durable-first and handles in-flight effects.
- Approval decisions survive crashes and are delivered to waiting workflows.
- Each DBOS family has its own application identity and pinned compatibility version.
- Version affinity and drain visibility are proven.
- Effect-time policy is revalidated.
- Durable sleep and signal recovery are proven.
- Lifecycle status exposes staleness.
- The determinism analyzer implements all seven approved rule families.
- The generic Class D probe has 12 tests.
- REST, MCP, and CLI are thin adapters.
- OpenCode drives a real demonstration.
- ADR preflight and CI enforcement exist.
- Thirteen of fourteen operational G1 checks pass.
- No real provider writes, production integrations, or `lead-followup` implementation were introduced.
- No existing tests were weakened.
- All work remains uncommitted.

Independently verify every claim.

Do not accept the presence of a package, interface, migration, test name, report, or CI job as proof that the capability works.

## 6. Workstream A — Canonical invocation and submission atomicity

Review the entire path:

`adapter → trusted principal → canonical invocation → authorization → idempotency → registry → executor → run binding → audit`

Inspect:

- `runtime/`
- `registry/`
- `contract/`
- `executor/classa/`
- `executor/dbos/`
- `auth/`
- relevant migrations and tests.

### A1. One canonical path

Verify REST, MCP, CLI, and direct framework invocation all use the same canonical service.

Find any duplicated authorization, idempotency, policy, audit, dispatch, or execution logic in adapters.

Try to identify a code path that can invoke an implementation without going through the canonical service.

### A2. Auth-before-dispatch

Verify that:

- Invalid or missing authentication is rejected before implementation dispatch.
- Tenant and authority context are derived from trusted server state.
- Client-supplied identity fields cannot override trusted context.
- Refused calls do not execute miniapp handlers.
- Refusals are attributed and auditable.

### A3. Submission failure window

Independently trace the critical failure:

1. Invocation accepted and recorded.
2. DBOS workflow submission succeeds.
3. Run-binding persistence fails.
4. Caller receives a retryable error.
5. Same-key retry occurs.

Verify that the new record-before-submit or other implemented mechanism actually closes this window.

Challenge the use of deterministic submission identity and DBOS workflow-ID reuse behavior against the pinned SDK.

Test or inspect whether:

- A crash after intent creation but before DBOS submission leaves a recoverable pending intent.
- A crash after DBOS submission does not create an unaddressable workflow.
- Retry returns a valid handle.
- Concurrent duplicate requests cannot produce divergent invocation/run relationships.
- Audit attribution remains intact.

Do not accept the phrase "record-before-submit" as proof of atomic submission. PostgreSQL and DBOS still have distinct failure boundaries.

### A4. Idempotency input comparison

Verify:

- Same key and equivalent canonical input replays the original invocation.
- Same key and materially different input is rejected.
- Canonicalization does not collapse materially different JSON values.
- Tenant, miniapp, tool, version, and other relevant scope dimensions are included in the idempotency namespace.
- Hashes are stable and not dependent on arbitrary serialization whitespace.

Look for duplicate JSON keys, number normalization, null/missing distinctions, nested object ordering, and payload validation issues.

## 7. Workstream B — Identity, entity serialization, and multi-tenancy

Independently verify the five identity distinctions:

1. Invocation identity.
2. Workflow/run identity.
3. Entity identity.
4. Sequence identity.
5. Effect identity.

The implementation must not conflate them.

### B1. Entity-key admission

Inspect contract and registry validation.

Verify that Class D write tools requiring an entity key cannot be admitted with:

- Missing `entity_key`.
- Empty entity key.
- Raw unscoped identifiers.
- Client-controlled tenant scope.
- An invalid entity-key declaration.

Check that the tenant scope is bound by trusted server context, not supplied as an authoritative client value.

### B2. Class A serialization

Verify the actual locking/transaction implementation:

- Lock scope is tenant-safe.
- Locks are acquired and released correctly.
- No lock is held across a durable wait.
- Errors do not strand locks.
- Concurrent operations on the same entity serialize.
- Different entities can proceed concurrently.

### B3. Class D attach and uniqueness semantics

The report claims "Class D attach semantics" and a durable-sequence demonstration.

Inspect what "attach" actually means.

Verify:

- Multiple invocations do not accidentally create multiple active sequences when one sequence is required.
- The losing invocation has its own durable identity and audit attribution.
- It references the correct existing workflow.
- It receives a valid run handle.
- Same-tenant entity identity is enforced in PostgreSQL.
- Cross-tenant entity IDs never attach or collide.
- A completed sequence can be followed by a legitimate new sequence where the domain permits it.
- Cancellation and concurrent re-submission do not create inconsistent generations.

Determine whether business-level uniqueness is enforced by the durable database record or incorrectly delegated to the workflow ID or queue.

### B4. RLS and explicit tenant predicates

Inspect all relevant SELECT, INSERT, UPDATE, DELETE, claim, transition, and lifecycle queries.

Verify tenant predicates remain correct even under privileged roles that bypass RLS.

The implementation report claims a `BYPASSRLS` proof. Inspect the exact test and determine whether it tests real privileged execution rather than merely checking SQL text.

Inspect tenant transaction-local context under pooled connections.

Test or examine whether one transaction can leak tenant context into the next pooled connection user.

Look for missing tenant predicates in less obvious paths such as cleanup, audit, approval, effects, and run-status reconciliation.

## 8. Workstream C — Effects ledger and truthful external guarantees

This is a critical review area.

Inspect all effect code and migrations, particularly:

- Claim/attempt acquisition.
- `PENDING`, `ATTEMPTING`, `CONFIRMED`, `FAILED`, `UNKNOWN` transitions.
- Lease ownership and expiry.
- Takeover.
- Settlement.
- Retry eligibility.
- Provider/mock invocation.
- Tenant-scoped effect identity.
- SQL concurrency controls.

### C1. Challenge the "exactly-once confirmed effects" claim

The report uses the phrase "exactly-once confirmed effects."

Determine whether it means:

- One confirmed database record per logical effect.
- One successful provider invocation.
- One accepted external message.
- One recipient-visible delivery.

These are different guarantees.

The framework must not confuse a unique CONFIRMED record with exactly-once external delivery.

Evaluate the implementation against the governing R2 language:

**No duplicate confirmed business effects, no silently lost operations, and no blind retries of unknown outcomes.**

Any stronger guarantee must be precisely qualified by the actual provider capability and transaction boundary.

### C2. Lease takeover and stale-worker race

Challenge the most dangerous race:

1. Worker A claims an effect.
2. Worker A pauses after acquiring the lease but before or during the external operation.
3. The lease expires.
4. Worker B takes over.
5. Worker A resumes.
6. Both workers may now perform or settle the same effect.

Determine whether there is a real fencing mechanism, attempt token, state predicate, provider idempotency boundary, or other control that prevents a stale worker from sending or overwriting a newer attempt.

A lease alone does not prove exclusive external execution.

### C3. Claim/transition race

Verify that:

- Concurrent duplicate claims cannot produce two independent attempts.
- Losing workers cannot overwrite terminal states.
- `UPDATE` statements enforce expected current state.
- State transitions are atomic under PostgreSQL concurrency.
- Transaction isolation and lock acquisition are correctly scoped.
- Deadlock and serialization retries do not repeat unsafe external effects.

### C4. UNKNOWN behavior

Inspect the poisoned mock and the test-only settlement path.

Verify actual behavior for:

1. Intent persisted, no send attempted.
2. Send accepted, acknowledgement lost.
3. Timeout with ambiguous outcome.
4. Process termination after acceptance but before confirmation.
5. DBOS step replay.
6. Lease takeover while the first worker remains alive.
7. Cancellation during an attempt.

An UNKNOWN effect must not be blindly retried.

Check that the test-only reconciliation/settlement path does not masquerade as the Phase 4 production reconciliation worker.

Verify that operators or callers can observe unresolved UNKNOWN state and its attribution.

### C5. Effect-time identity and audit

Verify the identity scope prevents collision between identical local effect IDs in different tenants.

Reconstruct an effect's history from audit/ledger records. Determine whether the framework can identify the initiating agent, authority principal, invocation, entity, effect, attempt, and final outcome.

## 9. Workstream D — Cancellation and lifecycle correctness

The report claims durable-first cancellation and in-flight reporting.

Challenge the invariant:

**Once cancellation is durably recorded, no new consequential effect attempt may begin.**

Inspect ordering among:

- Cancellation intent.
- Durable run cancellation.
- Effect claim.
- Provider/mock invocation.
- Effect settlement.
- Invocation status update.
- Audit write.

Review all eight scenarios in Prompt 13 §10.

Particular attention:

- Cancel racing with an effect claim.
- Cancel during an ATTEMPTING effect.
- Cancel when the outcome is UNKNOWN.
- Cancel during approval wait.
- Cancel while the executor is unavailable.
- Duplicate cancellation.
- Submission during cancellation.
- Process crash between cancellation record and DBOS signal.

Verify that the user-visible status does not falsely report clean cancellation when an effect may still have occurred.

A cancellation request cannot recall an already accepted external operation.

Check that status and audit disclose unresolved in-flight effects without falsely claiming they are completed or cancelled.

## 10. Workstream E — Approval bridge and authority enforcement

The report claims a crash-safe approval bridge with idempotent decisions, decision signals, workflow table re-read, scope enforcement, and lazy expiry.

Inspect the actual protocol.

The approval database record must remain authoritative. A DBOS signal is only a notification that the decision may have changed.

### E1. Crash-window correctness

Verify:

1. Decision commits.
2. Process crashes before signal delivery.
3. Process restarts.
4. The waiting workflow eventually observes the authoritative decision.
5. Duplicate signal delivery is safe.
6. A crash after signal delivery but before workflow progress does not lose the decision.

Find the exact redelivery/reconciliation mechanism. A test that only repeats a signal call is insufficient.

### E2. Decision semantics

Verify:

- Duplicate decisions.
- Conflicting decisions.
- Self-approval refusal.
- Scope/role enforcement.
- Tenant isolation.
- Material-hash invalidation.
- Late decision after cancellation.
- Decision after expiry.
- Decision concurrent with cancellation.
- Agent principal and authority principal attribution.

Determine whether the role resolver uses trusted server-side role state or caller-supplied scope claims.

### E3. Lazy expiry

The report specifically claims lazy expiry.

Inspect its actual semantics:

- When is expiry stored?
- Which operation causes expiry to be evaluated?
- Can a workflow remain parked forever without an external poll or decision?
- Does DBOS itself have a bounded wait/timeout?
- Can an expired decision be incorrectly accepted?
- What happens if the process is down during expiry?
- Is an explicit policy value required?

Do not treat lazy evaluation as autonomous expiry unless the implementation demonstrates that the waiting workflow will observe it within a bounded, defined mechanism.

### E4. Approval/run convergence

Verify that a denied or expired approval actually settles or cancels the live workflow and that invocation and executor statuses converge.

Test or inspect whether a denied workflow can continue into a consequential effect.

## 11. Workstream F — DBOS family identity, version affinity, and recovery

The report claims:

- Per-family DBOS applications.
- Pinned compatibility versions.
- Native affinity.
- Drain visibility.
- Pending workflow overlap tests.

Independently inspect the executor and pinned `dbos-transact-golang` v1.9.0 source.

Verify the distinctions among:

- Manifest version.
- Workflow definition version.
- DBOS `AppName`.
- DBOS `ApplicationVersion`.

### F1. Per-family identity

Verify that each Class D family has a stable, distinct `AppName` and that worker process-group selection cannot accidentally register another family's workflows.

### F2. Version management

Verify that the application version is explicitly controlled and tied to workflow compatibility, rather than accidentally changing due to repository or binary changes.

Inspect whether all required workers use the intended version.

### F3. Native affinity

The report states that DBOS recovery/dequeue filters by application version.

Verify this claim against the pinned SDK source and actual tests.

Check namespace behavior for:

- AppName.
- Queue names.
- Application versions.
- Workflow identifiers.
- Partition names.

The report identifies queue names and version names as DB-global. Determine whether this creates real cross-family collision or registration risks.

### F4. Drain and artifact retention

Distinguish:

- SDK-level version affinity.
- A process recognizing which version owns a workflow.
- Actual deployment retaining the old compatible artifact.
- Old workers remaining available until pending workflows drain.

A test that starts two versioned executors in one process does not automatically prove deployment-level artifact retention.

The implementation report itself says cross-release artifact retention remains unproven at G6. Ensure that limitation is not contradicted by the F-05 PROVEN label.

### F5. Shared-pool lifecycle

Verify worker shutdown and drain ordering before closing the shared datasource pool.

Inspect error and timeout paths: can shutdown failure still cause the pool to close while DBOS workers are active?

## 12. Workstream G — Effect-time policy and kill switches

F-06 is marked PARTIALLY PROVEN. Verify both the completed and incomplete parts.

Trace the full path:

`canonical invocation → trusted principal → policy/grant → workflow payload → resumed workflow → effect claim → mock capability`

Verify:

- Authority and tenant are not trusted from workflow payload.
- Current grant state is checked at the actual effect boundary.
- Revoked or changed grants prevent a new effect after a durable wait.
- Token/authority binding cannot be substituted by a workflow argument.
- Kill switches are authoritative and centrally enforced.
- The five reported kill-switch levels have clear precedence and fail-closed behavior.
- A race between policy read and effect claim cannot bypass revocation.
- Refusals are audited.
- Tenant isolation holds under privileged roles.

### Volumetric controls

The report says volumetric, allowlist, and circuit-breaker enforcement are not yet proven because OSI-4 requires owner-defined formats.

Verify that the implementation fails closed where required and does not silently treat missing limits as unlimited permission.

Do not invent values or close OSI-4 on behalf of the owner.

Distinguish an implemented enforcement mechanism from an owner-defined configuration that is still missing.

## 13. Workstream H — Durable sleeps, signals, and status

The report claims pinned-SDK sleep/signal behavior and a measured sleeper envelope.

### H1. Process restart

Inspect the actual process-kill/restart runner. Verify that the worker is a separate process and that the termination is real, not merely cancellation of a goroutine.

Verify that:

1. A generic workflow reaches a durable sleep.
2. The process is terminated.
3. The framework restarts against the same PostgreSQL state.
4. The sleeping workflow resumes correctly.
5. Completed stages are not repeated.
6. Confirmed effects are not repeated.
7. Cancellation during sleep works.

### H2. Signal timing matrix

Verify actual tests for:

- Signal before wait.
- Signal during wait.
- Signal after wait.
- Duplicate signal.
- Multiple signals.
- Signal followed by process termination.
- Signal delivered while the workflow's process group is unavailable.

Do not generalize beyond the SDK semantics actually tested.

### H3. Edition boundary and sleeper envelope

Verify the claim that resident sleepers hold partition slots and that suspension is Enterprise-gated for the pinned v1.9.0 edition.

Inspect the measured test, concurrency count, duration, memory readings, pool use, and measurement method.

Distinguish a small measured test envelope from a production capacity guarantee.

Do not infer scale-to-zero or high availability.

### H4. Status freshness

Verify the reported `Stale` behavior.

Ensure consumers can distinguish:

- Fresh live status.
- Last persisted state because the executor is unavailable.
- Unknown status.
- Terminal state.

Inspect all paths for swallowed errors, stale cache responses, misleading success states, and incorrect terminal-state transitions.

## 14. Workstream I — Determinism analyzer

The report claims all seven approved rule families are implemented and strict CI is enabled.

Read `docs/phase0/determinism-rules.md` first. Do not invent additional rule families or replace approved ones.

Inspect:

- Analyzer implementation.
- Analyzer fixtures.
- Workflow registration recognition.
- Helper-call analysis.
- Command-line tool.
- Repository checks.
- CI workflow.
- The exact Class D strict-mode invocation.

For every rule family, verify:

1. Positive fixture passes.
2. Negative fixture fails.
3. Diagnostic has correct position and explanation.
4. A permitted deterministic pattern is not falsely rejected.
5. The analyzer recognizes actual DBOS workflow boundaries.
6. A prohibited call hidden through the helper patterns covered by the specification is still detected.

Check whether aliases, imported wrappers, package boundaries, indirect calls, and helper functions can bypass the rules where the specification requires detection.

Do not demand whole-program semantic analysis if the approved specification does not require it. State the actual detection boundary honestly.

### Strict CI

Distinguish:

- Analyzer exists.
- Local strict-mode check runs.
- CI workflow is wired.
- CI workflow actually executes successfully on a remote runner.

The report says remote CI was not observed because no push was authorized. Therefore remote execution cannot be marked proven solely because the local gate passes.

## 15. Workstream J — Generic Class D conformance probe

The report claims a 12-test neutral conformance probe.

Inspect the actual probe, not just the test count.

Map every required Prompt 13 §18 item to a specific test and assertion:

1. Canonical authenticated invocation.
2. Server-derived tenant and principal attribution.
3. Stable workflow/run handle.
4. Multiple dependent checkpointed stages.
5. Durable wait.
6. External event/signal wait and resume.
7. Policy revalidation after a wait.
8. Mock effect through the shared ledger.
9. UNKNOWN outcome via poisoned mock.
10. Approval wait and recovery, if required by the generic contract.
11. Cancellation and in-flight-effect behavior.
12. Process restart/replay.
13. Version transition with a pending workflow.
14. Tenant-scoped entity concurrency.
15. Complete audit reconstruction.

A single test may cover multiple requirements, but each mapping must have a real assertion.

For each item, distinguish:

- Fully exercised end to end.
- Exercised only at component level.
- Simulated with a fake.
- Indirectly inferred.
- Not tested.

The conformance probe must demonstrate integrated behavior through the canonical invocation service. A collection of isolated unit tests does not prove the same thing.

Verify the probe is genuinely neutral and contains no hidden `lead-followup` business implementation.

Produce a clear "proves / does not prove" statement.

## 16. Workstream K — REST, MCP, CLI, and OpenCode

Inspect every transport adapter.

Verify:

- Each adapter delegates to the canonical service.
- No adapter implements independent authorization or execution.
- Trusted principal and tenant context are constructed server-side.
- Error and refusal behavior is consistent.
- Idempotency works across adapters.
- Run handles and lifecycle operations are preserved.
- MCP schemas are valid and accurately represent booleans, required fields, and input constraints.
- MCP stdio output is not polluted by logs or diagnostics.
- CLI credentials and environment variables do not leak into process output or audit payloads.

### OpenCode demonstration

Inspect the actual transcript and supporting harness files.

Verify:

- OpenCode 1.19.6.
- Isolated HOME and configuration.
- Exact model used.
- Successful `crm-read`.
- One refusal before dispatch.
- Class D submission.
- Stable run handle.
- Status and signal/cancel operation.
- Agent principal and authority attribution.

Determine whether the demonstration proves actual agent-driven tool use or merely executes a scripted sequence outside the harness.

Do not infer successful remote model availability from a transcript if the run did not occur against that model.

Global OpenCode configuration must remain untouched.

## 17. Workstream L — Migrations, database integrity, and repository quality

The implementation report mentions migrations 000003–000005 and an applied-migration edit that was forward-fixed using 000005.

Inspect this carefully.

Verify:

- Existing applied migrations were not silently rewritten.
- Migration 000005 correctly repairs the intended database state.
- Fresh database installation works from migration 000001 onward.
- Upgrade from the previously applied schema works.
- Rollback behavior is documented and safe.
- Migration checksums and ordering are correct.
- No migration destroys existing user data.
- DBOS, application, and other library-owned tables remain correctly separated.
- Constraints and indexes match the actual runtime concurrency model.
- Unique constraints include the required tenant scope.
- RLS policies and privileged roles behave as intended.

Inspect the "throwaway migrate down/up driver" to determine whether it ran on an actually disposable database.

Do not rerun destructive migration tests on the existing Neon branch.

### Build and dependencies

Verify:

- `go.mod` and `go.sum` changes.
- `golang.org/x/tools` version compatibility.
- Pinned SDK source versions.
- No unnecessary dependencies.
- No unapproved Enterprise DBOS modules.
- No credential or DSN leakage.
- No generated binaries or test artifacts committed accidentally.
- Correct `gofmt`, `go vet`, and test behavior.

### Existing test preservation

The report says no test was weakened and three tests were strengthened.

Independently inspect the diffs for:

- Weakened assertions.
- Removed assertions.
- Tests converted to skip.
- Broadened error matching.
- Reduced concurrency.
- Lowered race/failure pressure.
- New environment-dependent bypasses.
- Test-only behavior that differs materially from production code.

Compare against the G0 baseline and the pre-Prompt-13 tests.

## 18. Workstream M — ADR preflight and architectural enforcement

The report claims a structural ADR preflight and warning-only overclaim detection.

Inspect the implementation.

Verify that the preflight:

- Validates structured fields, not arbitrary prose as if it understood semantics.
- Rejects missing required Class D manifest fields.
- Validates tenant-scoped `entity_key`.
- Validates workflow family/version references.
- Validates ADR presence and approval state.
- Checks required evidence/test references.
- Reports unresolved Open Spec Items and owner decisions.
- Distinguishes hard errors, warnings, and owner decisions.
- Does not approve a business or architectural decision automatically.
- Does not treat a textual warning match as proof of a violation or proof of correctness.

The original `lead-followup` ADR remains DRAFT and NOT APPROVED.

Verify that the preflight refuses to admit or enable it.

Confirm it does not contain a hidden bypass or a hard-coded exception.

The owner and named human reviewer retain authority over the PILOT-vs-production decision and ADR approval.

## 19. Workstream N — Evidence integrity and G1 adjudication

This is a critical area. The implementation report says 13 of 14 operational checks PASS, with criterion 12 PARTIAL because remote CI was not observed.

Independently reconstruct the matrix.

### N1. Exact 14 operational checks

Use the 14 operational checks enumerated in `docs/prompt10.md` §9.1–14 as the row set, unless the actual file establishes a different enumeration.

These are operational checks derived from the G1 requirements, not 14 criteria explicitly listed as such in `plan4.2`.

For every row:

- Quote or accurately summarize the original operational check.
- Map it to the exact G1 text and tests in `plan4.2`.
- Identify source implementation.
- Identify the specific test or operational drill.
- Check the actual evidence artifact.
- Verify source-state binding.
- Verify environment and command.
- Record skips and failures.
- Independently adjudicate PASS, PARTIAL, BLOCKED, or NOT PROVEN.

Do not inherit the implementation agent's verdict.

### N2. No evidence laundering

Challenge these claims:

- Local Docker PostgreSQL 19 complete.
- Separate Neon pass complete.
- Race detector complete.
- OpenCode demonstration complete.
- Strict CI gate green locally.
- Migration down/up successful.
- Version N-to-N+1 and drain tests passed.

For each claim, establish what actually ran and what it proved.

A passing unit test cannot prove a process-restart test.

A successful migration against a throwaway database cannot prove safe upgrade of an existing environment.

An SDK-source inspection cannot prove runtime version affinity.

A local CI script cannot prove remote CI execution.

A database integration test cannot prove Fly deployment-level artifact retention.

An OpenCode transcript cannot prove agent attribution unless the transcript and server audit evidence can be linked.

### N3. Source and environment binding

Check whether the local and Neon test results ran against the same implementation state.

If the Neon test used a branch-scoped connection string and the branch was destroyed afterwards, verify:

- The branch was the previously authorized target.
- No other branch/project was altered.
- The evidence identifies the branch and environment without credentials.
- The code digest or source revision matches the reviewed tree.
- The destruction of the disposable test branch is documented and authorized.
- No important state was silently removed from the evidence record.

### N4. G1 versus Phase 6

Do not confuse G1 with later G6 requirements.

In particular:

- Runtime version affinity is not the same as deployment artifact retention.
- Process restart is not the same as Fly deployment overlap.
- Local strict CI is not remote CI execution.
- Generic PostgreSQL behavior is not proof of production deployment safety.

A G1 item should not be marked FAIL merely because a G6 activity is out of scope. Equally, G6 evidence must not be claimed from a G1 local test.

## 20. Security and adversarial threat review

Perform a focused threat review across the integrated invocation, workflow, effects, approvals, audit, database, and transports.

At minimum, attempt to reason through:

- Tenant spoofing.
- Privileged-role RLS bypass.
- Confused-deputy invocation.
- Workflow payload authority substitution.
- Cross-tenant effect-ID collision.
- Duplicate submission with altered input.
- Concurrent effect claims.
- Lease takeover while stale worker remains alive.
- Approval after cancellation.
- Approval signal loss.
- Expiry without an active poller.
- Cancellation/effect races.
- Old-version workflow handled by incompatible code.
- Queue namespace collision.
- Run handle exposure across tenants.
- Secrets accidentally serialized into DBOS inputs, checkpoints, logs, or audit.
- MCP stdio pollution.
- Database migration partial failure.
- Preflight bypass.
- Untrusted user input crossing into SQL, workflow identity, or capability execution.

For each credible path, identify whether a test exists and what it demonstrates.

Do not claim the application is secure simply because the code uses RLS or a test asserts unauthorized access is rejected.

## 21. Required consolidated findings

Create a findings register with stable IDs.

Retain F-01 through F-14 for continuity with the existing capability matrix. Add new identifiers only for genuinely new issues.

Classify findings:

- **CRITICAL** — exploitable security boundary failure, data corruption, unbounded duplicate consequential effects, or silent loss of durable work.
- **HIGH** — material Class D invariant or recovery failure.
- **MEDIUM** — partial, misleading, or insufficiently proven capability.
- **LOW** — hardening or maintainability concern.
- **OBSERVATION** — not a defect, but an important limitation or future phase boundary.

For every finding include:

1. ID and severity.
2. Exact source file and relevant function/line.
3. Governing Prompt 13 or `plan4.2` requirement.
4. The actual defect, contradiction, or evidence gap.
5. A concrete failure scenario.
6. Security or business consequence.
7. Reproduction test or evidence required.
8. Whether it is a code defect, test gap, evidence gap, documentation defect, or owner decision.
9. Required resolution.
10. Whether it blocks G1, Class D readiness, or only a later phase.

Do not inflate severity to make the review appear adversarial. Do not suppress a serious finding to preserve the implementation agent's verdict.

## 22. Required review report

Write exactly one new file:

`docs/phase1/classd-framework-adversarial-review.md`

It must contain:

### A. Executive verdict

Choose one:

- **ACCEPTABLE** — implementation satisfies Prompt 13 within its stated Phase 1 envelope.
- **CONDITIONALLY ACCEPTABLE** — core design is sound but specified, bounded defects or evidence gaps remain.
- **NOT ACCEPTABLE** — one or more important framework invariants are not met.
- **INSUFFICIENT EVIDENCE** — source may be sound, but the implementation report's claims cannot be independently established.

State clearly whether the framework is genuinely Class D-ready within the tested envelope.

Do not use an unqualified "Class D-ready" label if mandatory capabilities remain unproven.

### B. Baseline and repository integrity

Record HEAD, branch, source digest, dirty state, modified/untracked paths, immutable baseline integrity, and review-created files.

### C. Implementation report claim audit

For every material claim in `prompt13_imp.md`, classify:

- VERIFIED.
- PARTIALLY VERIFIED.
- NOT VERIFIED.
- CONTRADICTED.

Cite the exact code, test, or evidence supporting the verdict.

### D. F-01 through F-14 findings

Provide the independent finding-by-finding verdicts.

### E. New findings

Include any material issue not covered by the original 14 findings.

### F. Failure-window analysis

Summarize the verified behavior for:

- Submission.
- Effects and leases.
- Cancellation.
- Approval.
- Durable waits/signals.
- Version transition.
- Tenant/entity concurrency.

### G. Determinism analyzer assessment

Map all seven rule families to source, fixtures, diagnostics, CI enforcement, and limitations.

### H. Generic Class D conformance assessment

Map all 15 Prompt 13 §18 requirements to exact test assertions and verdicts.

### I. Fourteen-row G1 operational matrix

Reconstruct and independently adjudicate every row.

Keep G1 and G6 evidence distinct.

### J. Security assessment

Report attack paths, mitigations, missing controls, and residual risks.

### K. Database and migration integrity

Review migration history, fresh-install and upgrade behavior, RLS, pool context, and privileged-role behavior.

### L. Phase boundaries and Open Spec Items

Preserve OSI-1 through OSI-4 and any additional valid items. Do not invent owner decisions.

### M. Final recommendation

State exactly what should happen next:

- Accept Prompt 13 implementation as-is.
- Require bounded remediation and re-review.
- Reopen a specific architecture decision.
- Hold G1 because required proof is missing.

The recommendation must distinguish code defects from unobserved remote CI, G6 deployment proof, production playbook approval, and other deliberately deferred scope.

### N. Review change inventory

Confirm only the review report was added and that no code, test, migration, ADR, manifest, or configuration was modified.

## 23. Review discipline

- Inspect the actual implementation, not merely the completion report.
- Trace complete call paths and database transactions.
- Read tests to understand assertions, not just test names.
- Verify pinned SDK behavior from the actual v1.9.0 source.
- Use exact source references and evidence IDs.
- Report NOT PROVEN when evidence is absent.
- Report PARTIALLY PROVEN when only component behavior is demonstrated.
- Distinguish implementation defects from owner decisions and later-phase requirements.
- Do not require real provider integrations or Fly/G6 drills to pass G1.
- Do not let a generic probe claim to prove every possible Class D workload.
- Do not turn the simple fixture into an artificially complex sales process.
- Do not treat a green unit suite as proof of the full distributed execution contract.
- Do not treat the implementation agent's own review matrix as independent validation.
- Do not create a fix while reviewing.
- No commits or pushes.

## 24. Final instruction

The implementation report makes unusually strong claims: nearly every core framework capability is marked proven, the local and Neon runs are reported green, and 13 of 14 operational G1 checks are marked PASS.

Those claims must withstand independent verification.

**Your job is not to find faults at any cost. Your job is to discover whether the framework's claimed guarantees are real, reproducible, and enforced at the boundaries where failure can occur.**

A small Class D probe with correct failure semantics is more valuable than an elaborate workflow with misleading guarantees.

Preserve the hard stop: no `lead-followup` implementation, manifest, registration, execution, or ADR approval.

Stop after producing the adversarial review report. Do not implement fixes.

### **The three areas I would scrutinize most**

1\. **Effect-ledger lease takeover:** a race-hardened database ledger can still permit two actual sends if a stale worker resumes after another worker takes over. The review must establish whether the implementation has genuine fencing at the external-effect boundary.

2\. **Approval expiry and signal recovery:** lazy expiry and signal redelivery must be shown to work even when the process is unavailable. A decision stored in PostgreSQL is not enough if the workflow can remain parked indefinitely.

3\. **The 13/14 G1 claim:** independently trace every operational check to its actual test and evidence. In particular, SDK affinity, local process recovery, remote CI execution, and deployment-level artifact retention are different proofs.

The intended result is an independent acceptance decision, not another round of implementation or architectural redesign.