The third review adds several worthwhile findings, particularly the need for **framework-level enforcement of architectural contracts**, rather than relying on each miniapp author or ADR reviewer to catch mistakes.

I would incorporate its findings on effect guarantees, tenant-scoped `entity_key`, touch/effect boundaries, cancellation races, approval-to-signal recovery, and ADR consistency validation. I would not adopt its suggestion that a workflow ID should be the invocation identity, or that a partitioned queue alone guarantees business-level serialization. Those identities and responsibilities must remain distinct.

The following prompt consolidates all three reviews and supersedes Prompt 12.

# Prompt 13 — Phase 1 Class D Framework Readiness: Hardening, Enforcement, and Adversarial Conformance

## 1. Mission

Act as the Principal Architect and senior Go/distributed-systems engineer responsible for `github.com/PithomLabs/miniapp`.

Continue Phase 1 from the accepted G0 baseline. The objective is to establish whether the framework can safely support genuinely DBOS-warranted business processes, not to make the `lead-followup` example artificially complicated.

The central question is:

**Can the framework reliably host long-lived, multi-stage business processes involving durable waits, signals and events, consequential effects, human decisions, cancellation, recovery across restarts and deployments, and tenant-safe concurrency?**

The `lead-followup` example is deliberately simple. It is a forcing fixture, intended to exercise the framework's reusable Class D capabilities using controlled synthetic semantics and mock capabilities.

Do not evaluate the entire DBOS architecture based on whether this particular sales process independently justifies DBOS. A production workload's DBOS adoption is a separate workload-specific decision.

Your responsibility is to strengthen the generic framework, demonstrate its guarantees through adversarial tests, and identify exactly what remains unproven.

## 2. Governing authority and source material

Read and follow:

1. `docs/plan4.2.md` — authoritative and immutable.
2. `docs/phase0/g0-core-adjudication.md`.
3. `docs/phase0/g0-evidence.md` and `docs/evidence/README.md`.
4. `docs/phase0/determinism-rules.md`.
5. `docs/phase1/open-spec-items.md`.
6. `docs/adr/miniapps/lead-followup-adr001.md`.
7. The consolidated review of the first two adversarial reviews: `consolidated_phase1_classd_framework_review.md`.
8. The third review: `adv_review3_prompt11.md`.
9. The original reviews, if available: `adv_review_prompt11.md` and `adv_review2_prompt11.md`.
10. The actual Phase 1 implementation, migrations, tests, CI workflows, scripts, manifests, and pinned dependency source.

If review files are not present in the repository, use the supplied review content and this prompt as inputs. Do not invent repository paths or claim to have inspected files that are unavailable.

The consolidated reviews are analysis, not unquestionable specifications. Verify every implementation finding against the current working tree.

### Governing architectural principles

Preserve:

- One canonical invocation and execution service.
- REST, MCP, CLI, and Web as thin adapters.
- Static miniapp registration and immutable version lookup.
- Server-derived principals, authority, and tenant context.
- Separation between agent principal and authority principal.
- Authorization before dispatch.
- Shared effects, approvals, audit, and lifecycle services.
- Class A inline execution and Class D stable run handles.
- Tenant-safe transaction and entity boundaries.
- DBOS family and version isolation according to `plan4.2`.
- Honest external-effect semantics using `PENDING`, `ATTEMPTING`, `CONFIRMED`, `FAILED`, and `UNKNOWN`.
- No blind retry of UNKNOWN effects.
- No exactly-once external-delivery promise.
- Stop and drain DBOS workers before closing their shared datasource pool.

Do not reopen accepted architectural decisions without concrete evidence of an invariant violation, security defect, or incompatibility with `plan4.2`.

## 3. Absolute hard stop

The `lead-followup` ADR remains DRAFT and NOT APPROVED.

Until an authorized human explicitly approves a revised ADR:

- Do not implement `lead-followup` business workflow code.
- Do not create its manifest.
- Do not register or execute it as a DBOS workflow.
- Do not introduce sales-specific behavior into framework packages.
- Do not modify or sign the original ADR.
- Do not proceed to increment 6.

Generic framework work and generic substrate probes may proceed within the existing Phase 1 authorization.

Use neutral test-only workflow names, identifiers, entities, and synthetic data. A generic probe must not disguise itself as `lead-followup`.

The human decision between a synthetic PILOT fixture and a production sales workload remains unresolved. Do not infer approval from the architecture plan, review verdict, or this prompt.

## 4. Correct review lens

Keep three distinct questions separate.

| Question             | Purpose                                                                                                                 |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Framework capability | Can the reusable framework execute and govern demanding Class D workflows safely?                                       |
| Fixture adequacy     | Does the controlled `lead-followup` example exercise enough Class D behavior to serve as a meaningful conformance test? |
| Production adoption  | Does a real sales follow-up process justify DBOS over simpler alternatives?                                             |

Phase 1 primarily answers the first question.

A small fixture is adequate when it exercises the difficult framework invariants. Artificially adding business branches, channels, quiet hours, escalations, and regulatory rules would make the test less focused, not more rigorous.

Conversely, a fixture that merely delays a function call without meaningful checkpoints, signals, cancellation, restart recovery, or version behavior is not adequate.

The framework must be capable of supporting more complex workloads without requiring those workloads to reimplement authorization, effects, approvals, durability, or recovery.

## 5. Establish the current baseline

Before modifications:

1. Verify HEAD, branch, remote tracking state, G0 ancestry, and pinned dependencies.
2. Capture `git status --short`, existing diffs, and untracked files.
3. Identify all increment 4–5 work and preserve it.
4. Confirm the exact DBOS Go SDK version, River version, Go toolchain, PostgreSQL driver, and CI environment.
5. Run the existing test suite and distinguish PASS, FAIL, and SKIP.
6. Verify actual database-backed execution separately from unit-only evidence.
7. Preserve the historical successful local PostgreSQL evidence as a separate evidence item.
8. Check whether the authorized Neon test branch is available without changing or creating unrelated resources.

Do not clean, reset, stash, overwrite, or revert existing work.

Record the baseline source revision and working-tree state. Never treat different test environments as contradictory merely because one reviewer could not access a database that was available to another.

## 6. Consolidated framework capability findings

Create or update:

`docs/phase1/classd-capability-matrix.md`

Use stable finding IDs F-01 through F-14. Map the following findings into that matrix.

| ID   | Framework capability or concern                                                          |
| ---- | ---------------------------------------------------------------------------------------- |
| F-01 | Submission atomicity, invocation/run binding, and orphan-workflow prevention             |
| F-02 | Approval decision, signal delivery, cancellation, and workflow lifecycle convergence     |
| F-03 | Effect-ledger concurrency, crash boundaries, UNKNOWN quarantine, and truthful guarantees |
| F-04 | Separation of invocation, workflow, entity, sequence, and effect identities              |
| F-05 | Per-family DBOS application identity, version compatibility, and recovery                |
| F-06 | Effect-time authority, tenant, policy, and blast-radius enforcement                      |
| F-07 | Idempotency-key reuse with different input                                               |
| F-08 | Tenant-scoped identities, SQL predicates, and pooled-connection isolation                |
| F-09 | Approval authority, scope, expiry, duplicate decisions, and self-approval                |
| F-10 | Durable sleep, signal recovery, and measured worker/edition behavior                     |
| F-11 | Fresh versus stale lifecycle status and executor-unavailable behavior                    |
| F-12 | Entity serialization, DBOS partitioned queues, and concurrency semantics                 |
| F-13 | Seven-family determinism analyzer, operations, and CI enforcement                        |
| F-14 | Framework/miniapp boundaries and architectural-contract enforcement                      |

Include the additional third-review concerns:

- Touch/business-operation boundaries must not be confused with DBOS step boundaries.
- Cancellation during an in-flight effect needs explicit semantics.
- Approval must be rechecked and delivered to the waiting workflow reliably.
- Eligibility and policy rechecks must occur at the correct durable boundary.
- A test-only mock must deliberately produce an UNKNOWN outcome.
- ADR/manifest consistency must be validated before a Class D workload is approved or registered.

For every finding include:

- Original finding and source review.
- Governing `plan4.2` requirement.
- Exact source and test locations.
- Root cause, if confirmed.
- Whether it is a framework defect, an unproven capability, an ADR/process weakness, or a miniapp-specific question.
- Required correction or proof.
- Regression test.
- Evidence environment.
- Current disposition: PROVEN, PARTIALLY PROVEN, NOT PROVEN, or DEFECT CONFIRMED.

Do not accept a finding merely because a review asserts it. Inspect the code.

## 7. Workstream A — Identity and entity serialization

Keep these identities strictly distinct:

| Identity              | Meaning                                                                |
| --------------------- | ---------------------------------------------------------------------- |
| Invocation identity   | One accepted canonical request and its retries                         |
| Workflow/run identity | One durable execution registered with DBOS                             |
| Entity identity       | The business entity declared through `entity_key`                      |
| Sequence identity     | A domain-specific unit of work or generation, where applicable         |
| Effect identity       | One consequential business operation under the shared effects contract |

Do not conflate the workflow ID with invocation identity or entity identity.

A workflow ID alone is not an entity lock.

### Required behavior

- Consequential write tools declare a tenant-scoped `entity_key`.
- The framework rejects missing or unscoped entity keys where required by the governing contract.
- Different tenants may legitimately use identical local entity IDs.
- Cross-tenant IDs must never collide, attach, or leak run handles.
- Same-entity concurrency follows an explicit serialization contract.
- Different entities can execute concurrently.
- Any requirement for one active business sequence per entity has a durable business-level uniqueness mechanism.
- A completed sequence can be followed by a legitimate later sequence, subject to miniapp-defined generation semantics.
- Invocation records preserve their own idempotency and audit identities even when multiple invocations attach to one existing run.

### Queue decision

Inspect the pinned DBOS v1.9.0 implementation before selecting a queue mechanism.

A partitioned DBOS queue with concurrency one may provide Class D execution ordering for a tenant-scoped entity key. However, it does not automatically provide deduplication or business-level active-sequence uniqueness.

Evaluate:

1. Partition key must include tenant scope.
2. Does a long-sleeping workflow occupy its partition slot?
3. What happens to a second workflow submitted to the same partition?
4. What are the fairness and starvation characteristics?
5. Can different partitions execute concurrently?
6. What happens across process restart and deployment overlap?

Do not assume queue ordering equals request deduplication.

Do not rely on a deterministic workflow ID as the sole serialization mechanism.

Do not hold a database transaction or advisory lock open across a multi-day durable wait.

Where one active sequence must be enforced, use a durable entity/sequence record and an appropriate database uniqueness or transactional claim. Let the workflow ID reference that durable sequence. The exact design must follow verified framework contracts and avoid introducing sales-specific identity concepts into generic code.

## 8. Workstream B — Invocation submission and orphan prevention

Address F-01.

Investigate the partial-failure window:

1. Canonical invocation is accepted.
2. DBOS successfully submits the workflow.
3. Persistence of the workflow/run binding fails.
4. The caller receives an error.
5. The caller retries with the original idempotency key.

The framework must not leave an unaddressable workflow that cannot be observed, cancelled, attributed, audited, or safely retried.

Inspect the pinned SDK and existing storage design before selecting the solution. Consider durable submission intent, idempotent workflow identity, transactional coordination, or compensation only where technically justified.

Required tests:

- Successful submit followed by injected run-binding persistence failure.
- Same-key retry.
- Concurrent duplicate submissions.
- Workflow status reconciliation.
- Cancellation or compensation of an orphan if required.
- Audit attribution of partial failure and recovery.

The retry must yield a valid handle or a correctly settled terminal state.

Never claim atomicity across PostgreSQL and DBOS unless the implementation actually proves it.

Also enforce:

- Same idempotency key plus same canonical input returns the same accepted invocation/result.
- Same key with materially different input returns an explicit conflict.
- Do not silently return a previous result for different input.

## 9. Workstream C — Shared effects protocol and step boundaries

Address F-03 and the effect-related findings from Review 3.

The governing state machine is:

`PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN`

Investigate the current ledger under real concurrent PostgreSQL transactions.

### Required framework invariants

- Duplicate attempts cannot race into two sends.
- A losing worker cannot overwrite a winning terminal state.
- State transitions require expected-state predicates or equivalent locking.
- Effect identity carries tenant scope.
- UNKNOWN is never blindly retried.
- Ambiguous external outcomes remain UNKNOWN until supported evidence resolves them.
- Every attempt and transition is attributable and auditable.
- A completed DBOS step is not automatically proof that an external effect was confirmed.

### Distinguish business-operation boundaries from DBOS step boundaries

A business operation or "touch" may consist of several durable steps. Do not assume that a business action and a DBOS step are the same unit.

The generic effect pattern must distinguish:

1. Pre-effect validation and policy checks.
2. Durable intent recording.
3. Attempt claim.
4. External capability invocation.
5. Durable outcome recording.
6. Post-effect audit and continuation.

Use the shared framework effects contract. Do not implement a second miniapp-owned ledger or call a provider directly from workflow control flow.

The workflow's recovery logic must consult the authoritative effect state before deciding whether an interrupted operation may continue.

The framework must not promise that a business operation resumes "from the last completed step" when the external effect associated with that step may be UNKNOWN.

### Required adversarial tests

- Parallel duplicate effect claims with synchronized workers.
- Intent recorded, process terminated before send.
- Mock capability accepts a send but acknowledgement is lost.
- Process terminates before outcome persistence.
- Ambiguous timeout.
- Concurrent competing state transitions.
- Duplicate effect identity under two tenants.
- Interrupted DBOS step replay.
- Cancellation racing the effect attempt.

Use a deterministic mock capability supporting confirmed acceptance, rejection, timeout, and acceptance-with-lost-acknowledgement.

The poisoned mock must demonstrate UNKNOWN quarantine and no blind retry.

Phase 1 does not implement the production Phase 4 reconciliation worker. It must correctly persist, expose, and quarantine UNKNOWN; a test-only mock reconciliation path may demonstrate the contract without creating production reconciliation infrastructure.

No real provider or external write is permitted.

## 10. Workstream D — Cancellation and lifecycle semantics

Address F-02 and cross-cutting lifecycle behavior.

A cancellation request cannot recall an external message already accepted by a provider.

Define and test the generic framework behavior for:

- Cancellation before an effect claim.
- Cancellation while waiting.
- Cancellation while approval is pending.
- Cancellation racing with an effect claim.
- Cancellation during ATTEMPTING.
- Cancellation when the effect is UNKNOWN.
- Cancellation followed by a duplicate submission.
- Cancellation while the executor is unavailable.

The core invariant is:

**After cancellation is durably recorded, no new consequential effect attempt may begin.**

An effect already in progress must settle as CONFIRMED, FAILED, or UNKNOWN according to evidence. Do not claim cancellation guarantees that an external effect can be recalled.

The response and lifecycle status must distinguish a clean cancellation from a cancellation with an unresolved in-flight effect.

Do not invent business-specific cancellation roles or re-entry rules. Implement the generic authority and lifecycle contract, and record undefined policy semantics as Open Spec Items.

## 11. Workstream E — Approval-to-workflow event protocol

Address F-02 and F-09, including the third review's approval-mapping findings.

Approval is a policy gate, not a separate execution class. The approval record is authoritative; a DBOS signal is a notification that a decision exists.

Prove this sequence:

1. Approval decision commits to PostgreSQL.
2. Process terminates before the DBOS signal is delivered.
3. Framework restarts.
4. The waiting workflow eventually observes the authoritative decision.
5. Duplicate signal delivery does not create duplicate decisions or effects.
6. A duplicate or conflicting approval decision is consistently rejected or idempotently handled.
7. Approval rejection or expiry settles the live workflow according to the generic contract.
8. A late decision cannot approve a terminal/cancelled invocation.
9. Invocation and workflow statuses converge or expose an actionable recoverable discrepancy.
10. Material-hash changes invalidate stale approvals.
11. Self-approval and unauthorized approver scope are rejected.
12. Approver identity, agent principal, and authority principal remain correctly attributed.

Use reliable signal delivery or reconciliation of the authoritative approval table. A one-time signal after a database commit is insufficient if a crash can occur between the two actions.

No default expiry may be invented. Only enforce an expiry that was explicitly supplied under an approved policy.

For any consequential effect, the ordering must ensure that approval is valid and current, then eligibility and policy are rechecked immediately before the effect claim.

Approval is not a substitute for the last authorization, eligibility, or effect-time policy check.

Do not introduce sales-specific approver roles, approval thresholds, or per-touch rules.

## 12. Workstream F — DBOS family and version-safe recovery

Address F-05 and the third review's pending-version concern.

Inspect the actual pinned DBOS Go SDK v1.9.0 behavior.

Distinguish:

- Miniapp manifest version.
- Workflow definition version.
- DBOS `AppName`.
- DBOS `ApplicationVersion`.

A field such as `WorkflowDef.Version` is not a version-recovery policy unless the executor and deployment process actually enforce it.

Requirements:

1. Stable DBOS `AppName` per Class D workflow family.
2. Explicit application-version behavior tied to workflow compatibility.
3. A documented version bump, compatibility, patch, and drain policy.
4. Old workflow artifacts remain available while incompatible old workflows are pending, unless a verified compatible patch is used.
5. Unknown or incompatible versions become visible and operator-actionable.
6. No silent abandonment of pending workflows.
7. Preserve the accepted same-image/process-group topology, subject to the existing G6 proof.
8. The executor must stop and drain workers before closing a shared pool.

Required test:

- Start a generic workflow using version N.
- Leave it at a durable wait.
- Introduce version N+1 using the accepted process topology.
- Verify the old workflow resumes under the documented compatibility policy.
- Verify old artifacts are not retired before the required drain.
- Verify unknown-version behavior.
- Verify all failures remain visible and recoverable.

A parser test or unit assertion on a version string is not proof of version-safe recovery.

## 13. Workstream G — Effect-time policy, authority, and blast-radius controls

Address F-06.

Trace the complete path from canonical invocation to the actual shared effect claim.

Prove that:

- Tenant, authority, and principal data come from trusted server context.
- Workflow payload cannot override those values.
- Authorization occurs before dispatch.
- Authority, policy, resource scope, limits, and kill-switch state are revalidated before a consequential effect claim as required by `plan4.2`.
- A revocation or policy change during a durable wait prevents a newly prohibited effect.
- Kill-switch and blast-radius enforcement is central and cannot be bypassed by a miniapp.
- Tenant isolation survives pooled database connections and privileged-role execution.
- Refusals and blocked effects are attributable and auditable.

Use deterministic mock capabilities only.

Do not implement OAuth, provider egress, or production write support.

## 14. Workstream H — Durable waits, signals, and status

Address F-10 and F-11.

Inspect pinned DBOS v1.9.0 behavior and establish the actual edition limitations. Do not infer current SDK behavior from another version or from unverified documentation.

Required tests:

1. Start a generic workflow and enter a durable wait.
2. Terminate the worker process while waiting.
3. Restart it against the same PostgreSQL state.
4. Verify the wait resumes according to documented behavior.
5. Deliver a signal before the wait, during the wait, and after the wait.
6. Test duplicate signals and signal replay.
7. Verify a completed stage and confirmed effect are not repeated.
8. Test cancellation during the wait.
9. Record actual memory, connection, history/checkpoint, and recovery behavior for representative concurrent sleepers.
10. Verify DBOS edition and licensed capability boundaries.

Do not invent production concurrency targets. Record measured envelopes and remaining scale requirements.

`Status` must distinguish:

- Fresh live executor status.
- Persisted status returned because the executor is unavailable.
- Stale or unknown state.
- Terminal state.

Never silently swallow executor errors and present stale persisted data as fresh live state.

## 15. Workstream I — Determinism analyzer and operational conformance

Address F-13.

Read `docs/phase0/determinism-rules.md`. Implement exactly the seven approved rule families using a Go analyzer compatible with the pinned toolchain and DBOS v1.9.0 APIs.

Requirements:

- Positive and negative fixtures for every rule family.
- Correct recognition of actual DBOS workflow boundaries.
- Actionable diagnostics with source location and rationale.
- Avoid false positives on allowed deterministic patterns.
- Detect prohibited calls hidden behind trivial helpers where the specification requires it.
- Run analyzer tests in repository checks.
- Enforce strict mode for Class D in CI.
- Do not silently add, omit, or reinterpret a rule family.
- Put ambiguous specification issues into Open Spec Items.

Also document and verify framework operations:

- Retry limits and terminal failure.
- Cancellation.
- Workflow inspection.
- Recovery ownership.
- History/checkpoint retention.
- Stuck-workflow and stale-status alerts.
- Safe operator intervention.
- DBOS edition and version verification.

Do not introduce Enterprise-only dependencies or behavior without explicit owner approval.

## 16. Workstream J — Architectural contract and ADR enforcement

The third review identifies an important process weakness: an ADR can contradict the framework's own contracts while still reaching human review.

Address this, but do not build an unreliable natural-language approval engine.

Inspect the existing registry, manifest validation, ADR template, and repository check structure.

Determine the smallest maintainable mechanism for a Class D conformance preflight.

Where the architecture permits, validate structured facts such as:

- Required manifest fields.
- Tenant-scoped `entity_key`.
- Class D workflow family identity.
- DBOS application-version policy reference.
- Approved ADR reference and approval state.
- Effect-state and retry contract declarations.
- Required evidence/test references.
- Open Spec Items and owner-decision status.
- Required ADR sections and named reviewer.

A structured contract must be the source of truth. Do not pretend that substring matching can understand arbitrary prose.

A narrow static lint may flag suspicious statements such as "exactly-once external delivery" for human review, but it must not silently approve or reject semantic claims on the basis of text matching alone.

Any enforcement mechanism must:

- Reject objectively invalid structured configuration.
- Report actionable diagnostics.
- Distinguish warning, error, and owner decision.
- Avoid false positives that make the tool unusable.
- Run in CI or the existing validation command where justified.
- Preserve human authority over business decisions.

Do not create a general-purpose policy language or AI-driven ADR reviewer.

If a reliable ADR consistency validator is not justified within Phase 1, deliver a structured checklist and a documented proposal, then record the remaining process risk.

## 17. Framework versus miniapp responsibilities

Make the boundary explicit and enforce it where required.

**Framework owns:**

- Canonical invocation and adapter-independent execution.
- Trusted principals and tenant context.
- Authorization and policy enforcement.
- Idempotency and invocation/run binding.
- Static registration and workflow family/version infrastructure.
- Entity-key and generic serialization primitives.
- Shared effects protocol and ledger.
- Approval state and reliable decision delivery.
- Audit attribution.
- Lifecycle operations.
- Determinism analysis and strict CI.
- Generic operational and recovery controls.

**Miniapps own:**

- Domain entities.
- Domain state and sequence/generation semantics.
- Business eligibility and domain-specific transitions.
- Input/output contracts.
- Declared entity keys and policy requirements.
- Use of framework effects and approval protocols.

Do not allow miniapps to introduce alternate datasources, authorization, lifecycle engines, or duplicate effect ledgers.

Do not put `lead_id`, sales campaign generations, contact preferences, quiet hours, or sales-specific stages into generic framework packages.

## 18. Generic Class D conformance probe

After fixing framework defects that prevent meaningful testing, build a compact, neutral Class D probe.

It must not be named `lead-followup` and must not contain sales semantics.

Exercise:

1. Canonical authenticated invocation.
2. Server-derived tenant and principal attribution.
3. Stable workflow/run handle.
4. Multiple dependent checkpointed stages.
5. Durable wait.
6. External event/signal wait and resume.
7. Policy revalidation after a wait.
8. Mock effect through the shared ledger.
9. UNKNOWN outcome via poisoned mock.
10. Approval wait and recovery if required by the generic Phase 1 approval contract.
11. Cancellation and in-flight-effect behavior.
12. Process restart/replay.
13. Version transition with a pending workflow.
14. Tenant-scoped entity concurrency.
15. Complete audit reconstruction.

Keep the fixture small. It exists to exercise reusable framework capabilities, not to simulate a production business.

Explicitly record what this probe does and does not prove.

Passing this probe does not establish that every DBOS use case is automatically safe. It establishes only the tested conformance envelope.

## 19. Preserve Class A, transports, and OpenCode demonstration

The `crm-read` Class A vertical slice remains the synchronous conformance example.

Prove that Class A and the generic Class D probe use the same canonical invocation and policy path.

Complete:

- REST adapter.
- MCP adapter.
- CLI adapter.
- Contract tests preventing policy/execution bypass.
- One OpenCode demonstration using the accepted version 1.19.6.
- Isolated OpenCode HOME/config.
- No modification to global OpenCode configuration.
- Successful `crm-read`.
- One authorization refusal before dispatch.
- Generic Class D submission with stable run handle.
- Attribution intact.
- Lifecycle status and at least one signal/cancel operation.

Do not build a UI or enter Phase 2+ integration work.

## 20. Validation discipline

For each workstream:

1. State the invariant.
2. Trace the complete call path and trusted boundaries.
3. Write a regression test that fails against the defect.
4. Investigate the root cause.
5. Implement the smallest durable fix.
6. Run the targeted test.
7. Run adjacent tests.
8. Run local PostgreSQL integration tests.
9. Run on the already-authorized Neon branch where available.
10. Record evidence before moving to the next workstream.

Final validation must include:

- `gofmt`.
- `go vet ./...`.
- `go test ./...`.
- Race detector.
- Repository checks.
- All seven analyzer families.
- Analyzer fixtures and strict CI.
- Invocation/run partial-failure tests.
- Same-key/different-input tests.
- Cross-tenant and entity concurrency tests.
- Concurrent effects and crash-boundary tests.
- Approval decision/signal/recovery tests.
- Effect-time policy and kill-switch tests.
- Durable wait/restart and signal tests.
- Version transition with old workflow pending.
- Generic Class D conformance probe.
- REST, MCP, and CLI contract tests.
- OpenCode E2E.
- Evidence-register validation.

A skipped database test is not a pass. A unit test is not a substitute for a required real-PostgreSQL or process-restart test.

Never fabricate evidence or claim a test ran when it did not.

## 21. G1 evidence matrix

Create or update:

`docs/phase1/classd-framework-readiness.md`

Evaluate the exact 14 G1 exit criteria in `plan4.2`. Do not invent a substitute list.

For every criterion record:

- Exact governing requirement.
- Implementation location.
- Test or operational drill.
- Exact source revision or working-tree digest.
- Execution environment.
- Exact command.
- Test names and results.
- Skips and failure disposition.
- Evidence-register reference.
- Verdict: PASS, PARTIAL, BLOCKED, or NOT PROVEN.

The capability matrix must link each F-01 through F-14 finding to its implementation, regression test, and evidence.

Do not issue an overall G1 PASS if a mandatory requirement lacks proof.

Keep local PostgreSQL, Neon, unit, race, process-restart, and OpenCode evidence separate.

## 22. ADR and owner-decision boundary

Do not assume the production playbook has been approved.

The `lead-followup` ADR may ultimately be reframed as a synthetic Class D PILOT/conformance fixture, but that requires an explicit human decision.

For a pilot:

- Synthetic semantics are approved only for the test fixture.
- They must be clearly labelled non-production.
- They must exercise framework invariants, not invent sales policy.
- A production workload score must not be derived from synthetic assumptions.
- If the governing ADR template requires a score, explicitly distinguish a pilot/conformance classification from a production adoption score. Do not fabricate business evidence.

For a production workload:

- A confirmed playbook and real business consequences are required.
- Gate 3 requires owner judgment.
- Gate 4 requires a fair comparison with alternatives over full lifecycle cost.
- River/Class C alternatives must be considered at the correct phase boundary.

The missing production playbook is not a blocker to generic framework remediation. It is a blocker to production adoption.

The following owner decisions remain separate:

1. Is `lead-followup` approved as a synthetic Phase 1 PILOT fixture, or should another Class D fixture be selected?
2. Who is the named human reviewer responsible for the ADR decision?
3. What synthetic values are approved for the pilot's stage, wait, event, approval, and cancellation tests?

Do not invent answers.

## 23. Phase boundaries

Remain within Phase 1.

Out of scope:

- Real SaaS providers or OAuth.
- Real external writes.
- Production UNKNOWN reconciliation worker from Phase 4.
- River and Class B/Class C substrate work from Phase 3.
- Sprite workspaces or untrusted code execution.
- Promotion and trusted-build pipeline.
- Production Fly deployment and full G6 drills.
- General workflow-engine/BPMN replacement.
- UI or UI generator.
- AI miniapp generator.
- Production sales-playbook implementation.

Where a control belongs to a later phase, document the boundary and fail closed in Phase 1. Do not pull later-phase infrastructure into the current milestone simply to clear a review item.

## 24. Required final report

Return:

### A. Executive verdict

State whether the framework is:

- Not Class D-ready.
- Partially Class D-ready.
- Class D-ready within the tested Phase 1 conformance envelope.

Do not generalize beyond the evidence.

### B. Baseline and change inventory

HEAD, ancestry, initial dirty state, files changed, and preservation of earlier work.

### C. F-01 through F-14 disposition

Root cause, source verification, correction, regression test, evidence, remaining risk.

### D. Capability matrix

All 14 findings mapped to framework invariants and evidence.

### E. Architectural enforcement

What manifest, registry, ADR, and runtime checks are now mechanically enforced; what still depends on human review.

### F. Adversarial validation

Actual commands, environments, failures, recoveries, results, and skips.

### G. Generic Class D conformance probe

Lifecycle, failure windows, proven guarantees, and limitations.

### H. Determinism analyzer

Seven approved rule families, fixtures, diagnostics, CI strict mode.

### I. G1 matrix

All 14 governing criteria with independent verdicts.

### J. Open Spec Items and owner decisions

No invented semantics or approvals.

### K. Phase boundary and next recommendation

Exactly what remains before G1 acceptance, human pilot-ADR approval, and increment 6.

### L. Evidence and change inventory

All uncommitted changes and evidence files. Confirm no commits or pushes.

## 25. Non-negotiable execution rules

- Preserve the G0 baseline.
- Do not modify `plan4.2.md`.
- Do not overwrite earlier increment 4–5 work.
- Do not weaken, delete, or skip existing tests to obtain a green suite.
- Fix underlying causes, not just visible symptoms.
- Require regression tests for invariants.
- Do not use a mock where real PostgreSQL or restart behavior is required.
- Do not confuse step completion with external-effect confirmation.
- Do not confuse workflow ID with entity serialization.
- Do not confuse invocation identity with workflow identity.
- Do not hold a transaction or lock across a durable wait.
- Do not promote proposed business semantics into requirements.
- Do not claim more than the tests prove.
- Continue independent framework work when a business decision blocks only one branch.
- Preserve all existing uncommitted work.

**No commits. No pushes.**

## 26. Final instruction

The purpose of this work is to make the framework enforce and demonstrate its own Class D contracts, so that future miniapps do not need to rediscover or reimplement them.

`lead-followup` is a deliberately modest forcing fixture. Keep its domain semantics modest and its framework tests rigorous.

**Do not make the example look worthy of DBOS. Make the framework worthy of the complex workflows it intends to host.**

**What changed from Prompt 12:** the third review's valid additions are now incorporated into the effects/step-boundary tests, cancellation semantics, approval-signal recovery, poisoned mock, tenant-scoped entity validation, and a lightweight architectural-contract preflight. The incorrect identity conflation and overly prescriptive queue recommendation were deliberately excluded. The ADR hard stop and phase boundaries remain unchanged.