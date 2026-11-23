The next task should be **Phase 1 Class D Framework Readiness**, not implementation of the `lead-followup` business workflow.

This prompt directs the agent to turn the 14 consolidated findings into verified framework capabilities, correct genuine defects, and build adversarial regression tests. It allows generic framework work to continue while preserving the ADR hard stop. It also prevents the agent from inflating the sales example simply to justify DBOS.

# Prompt 12 — Phase 1 Class D Framework Readiness: Remediation, Adversarial Proofs, and G1 Preparation

## Role

Act as the Principal Architect and senior Go/distributed-systems engineer responsible for `github.com/PithomLabs/miniapp`.

Your task is to bring the reusable Phase 1 execution framework closer to demonstrable Class D readiness by addressing the consolidated findings F-01 through F-14 and producing adversarial, reproducible evidence.

This is a framework engineering task, not a sales-workflow implementation task.

The central question is:

**Can this framework safely support genuinely DBOS-warranted business processes involving long-lived execution, multiple dependent stages, durable waits, external events, human decisions, recovery, consequential effects, and tenant-safe concurrency?**

Do not judge the framework by whether the `lead-followup` example itself is sufficiently complicated. It is a deliberately controlled forcing fixture. Its purpose is to exercise the framework, not to make a production business case for DBOS.

---

## 1. Governing sources and authority

Read and follow these sources:

1. `docs/plan4.2.md` — authoritative, immutable architecture baseline.
2. `docs/phase0/g0-core-adjudication.md` — accepted G0 decisions.
3. `docs/phase0/g0-evidence.md` and `docs/evidence/README.md`.
4. `docs/phase0/determinism-rules.md`.
5. `docs/phase1/open-spec-items.md`.
6. `docs/adr/miniapps/lead-followup-adr001.md` — currently DRAFT, NOT APPROVED.
7. `docs/adv_review_prompt11.md`.
8. `docs/adr/miniapps/lead-followup-adr001-review.md`, if present.
9. `docs/adv_review_prompt11.md` and `adv_review2_prompt11.md` supplied for the consolidated review.
10. `consolidated_phase1_classd_framework_review.md`.
11. The actual Phase 1 Go source, migrations, tests, scripts, CI workflows, manifests, and pinned dependency source.

`plan4.2.md` remains immutable. Do not change it to accommodate an implementation shortcut.

The consolidated review is an input to this task, not an unquestionable specification. Verify every code-level finding against the actual current working tree before accepting it as a defect.

### Accepted decisions to preserve

- One canonical invocation and execution service.
- REST, MCP, CLI, and Web are adapters, not independent execution paths.
- Static registration; no dynamic code loading.
- Server-derived tenant, authority, and principal context.
- Agent principal and authority principal remain distinct.
- One shared PostgreSQL-backed runtime and effects/approval/audit infrastructure.
- DBOS per-family identity and explicit version policy.
- Same-image/process-group topology, subject to the existing G6 proof requirement.
- `WorkflowDef.Version` must not be treated as an effective recovery policy unless the executor actually enforces the policy.
- Class A executes inline; Class D returns a stable run handle.
- `UNKNOWN` external effects are quarantined, never blindly retried.
- No exactly-once recipient-delivery promise.
- Shared-pool DBOS workers must stop and drain before the underlying pool closes.

Do not reopen settled decisions without concrete evidence of a violated invariant, security defect, or incompatibility with the governing plan.

---

## 2. Absolute hard stop: no `lead-followup` implementation

The `lead-followup` ADR remains DRAFT and NOT APPROVED.

Until an authorized human explicitly approves a revised ADR:

**Do not:**

- Create `lead-followup` workflow implementation code.
- Create or register its manifest.
- Register it with the DBOS executor.
- Enable or execute a DBOS workflow named `lead-followup`.
- Add `lead-followup` business semantics to generic framework code.
- Modify or sign the original ADR.
- Claim that the production sales playbook has been approved.
- Proceed to increment 6.

Generic framework work and generic substrate probes may proceed within the previously authorized Phase 1 scope.

For generic DBOS tests, use neutral test-only probes with synthetic data. A generic probe may exercise durable waits, signals, cancellation, recovery, approvals, effects, tenant isolation, and version transitions without representing itself as the `lead-followup` business process.

Do not put synthetic sales assumptions into framework contracts.

The choice between a PILOT forcing fixture and a production workload remains a separate human decision. Do not infer that choice from this prompt.

---

## 3. First establish the actual repository state

Before editing anything:

1. Confirm the current HEAD, branch, remote tracking state, and G0 commit ancestry.
2. Capture `git status --short`, `git diff`, and the existing untracked-file inventory.
3. Identify which files are already modified or untracked from increments 4–5.
4. Verify the pinned DBOS Go SDK version, River version, PostgreSQL driver, Go toolchain, and analyzer dependencies.
5. Confirm that all existing Phase 0 and Phase 1 tests still pass.
6. Separate tests that actually ran against PostgreSQL from tests skipped for missing configuration.
7. Verify which local and Neon test databases are already authorized and configured.

Do not reset, clean, stash, overwrite, or revert the existing working tree.

Do not assume the successful increment 4–5 local PostgreSQL run and a separate reviewer run with skipped database tests are contradictory. They were conducted in different environments. Preserve their exact evidence independently.

Record the baseline source revision and working-tree state.

If an existing file differs from the consolidated review, inspect the current code and update the finding's disposition based on evidence.

---

## 4. Establish the Class D framework capability matrix

Create:

`docs/phase1/classd-capability-matrix.md`

Map every consolidated finding to:

- Finding ID.
- Framework invariant.
- Governing `plan4.2` requirement.
- Relevant package, source files, and current implementation.
- Existing test coverage.
- Evidence status: PROVEN, PARTIALLY PROVEN, NOT PROVEN, or DEFECT CONFIRMED.
- Whether the issue is generic framework behavior or miniapp-specific behavior.
- Required correction or proof.
- Required test and environment.
- Phase 1 completion status.
- Any owner decision still required.

Use the following findings as the initial matrix.

| ID   | Capability or finding                                                  |
| ---- | ---------------------------------------------------------------------- |
| F-01 | Submit/record atomicity and prevention of orphan workflows             |
| F-02 | Approval decision and workflow lifecycle coupling                      |
| F-03 | Effect-ledger concurrency and truthful external-effect guarantees      |
| F-04 | Separation of invocation, run, entity, sequence, and effect identities |
| F-05 | DBOS family/version compatibility and recovery                         |
| F-06 | Effect-time authority, tenant, and blast-radius policy                 |
| F-07 | Same idempotency key with different input                              |
| F-08 | Explicit tenant scoping in identities and mutations                    |
| F-09 | Approval scope/role enforcement                                        |
| F-10 | Durable sleep recovery and capacity/edition behavior                   |
| F-11 | Status freshness when the executor is unavailable                      |
| F-12 | Queue partitioning versus business-level serialization                 |
| F-13 | Determinism analyzer and operational controls                          |
| F-14 | Framework versus miniapp responsibility boundaries                     |

Also retain the secondary hardening findings:

- Explicit tenant predicates on invocation/effect mutations.
- Stale status visibility.
- Canonical JSON intent hashing.

Do not close a finding merely because a field, interface, comment, or configuration option exists. It must be enforced and tested.

---

## 5. Implement and prove the framework invariants

Work incrementally. Each increment must have:

1. A clearly stated invariant.
2. A root-cause investigation.
3. A minimal implementation that preserves the architecture.
4. A regression test that fails against the original defect.
5. A successful test against the correction.
6. An evidence entry.
7. No regression to previously established behavior.

### Workstream A — Identity, tenancy, and invocation correctness

Address F-04, F-07, F-08, and the identity portion of F-14.

Keep these identities distinct:

- **Invocation identity:** one canonical request and its idempotent replays.
- **Run/workflow identity:** one durable execution.
- **Entity identity:** the business object declared through the miniapp's `entity_key`.
- **Sequence identity:** a domain concept owned by a miniapp, where needed.
- **Effect identity:** one consequential operation under the shared effects contract.

The framework must not assume that a workflow ID is an entity lock.

Requirements:

1. Verify that `entity_key` is represented and enforced according to `plan4.2`.
2. Include tenant scope in entity and effect identities.
3. Demonstrate serialization for the same entity where the contract requires it.
4. Demonstrate parallel execution for distinct entities.
5. Add explicit tenant predicates to privileged and ordinary mutation paths, rather than relying solely on RLS.
6. Prove that identical entity IDs in different tenants cannot collide, attach, or leak handles.
7. Hash a canonical representation of request input for idempotency comparison.
8. The same idempotency key and same input must replay the original result or handle.
9. The same key with different input must fail explicitly.
10. Ensure any attached invocation has an explicit and auditable relationship to its existing run.

Do not hard-code `lead_id` or sales-sequence generation into the framework.

A DBOS partitioned queue may be evaluated as an execution-ordering mechanism, but it must not substitute for business-level uniqueness or entity serialization.

### Workstream B — Run submission and stable handle guarantees

Address F-01.

Inspect the exact failure window:

1. The canonical invocation is accepted.
2. DBOS successfully submits a workflow.
3. Persistence of the workflow/run binding fails.
4. The caller receives an error.
5. The caller retries using the original idempotency key.

The framework must not leave a running workflow that cannot be found, observed, cancelled, attributed, or safely retried.

Investigate and implement the correct root-level solution. Options may involve durable submission intent, deterministic idempotent submission identity, transactional outbox-like coordination, or compensation, but do not select one without inspecting DBOS v1.9.0 semantics and the existing storage model.

Required tests:

- DBOS submission succeeds, then invocation persistence is fault-injected to fail.
- Same-key retry occurs.
- No unaddressable workflow remains.
- Retry returns a valid live handle or a properly compensated terminal result.
- Audit history reflects the partial failure and recovery.
- Concurrent duplicate submissions do not create inconsistent invocation/run relationships.

Do not claim atomicity across PostgreSQL and the DBOS runtime unless the implementation actually provides it.

### Workstream C — Shared effects protocol and concurrency

Address F-03 and the effect portion of F-08.

The framework must preserve the governing effect-state discipline:

`PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN`

Investigate the current claim/load/transition behavior under real concurrent transactions.

Required invariants:

- One logical effect identity cannot be concurrently claimed by two independent attempts without an explicit safe rule.
- A losing worker cannot overwrite a winning worker's terminal state.
- State transitions must be guarded by the expected current state or equivalent locking.
- All identity and mutation paths must be tenant-safe, including under privileged database roles.
- Retries must not blindly resend an effect with an UNKNOWN outcome.
- An ambiguous provider outcome remains UNKNOWN until supported evidence resolves it.
- Audit records preserve the attempt and outcome history.

Use transaction/row-lock or compare-and-set techniques as justified by the actual concurrency model.

Required adversarial tests:

1. Parallel duplicate attempts using barrier synchronization.
2. Crash after intent creation but before the send.
3. Crash after mock-channel acceptance but before confirmation persistence.
4. Ambiguous timeout.
5. Concurrent terminal transition attempts.
6. Same effect identifier in two tenants.
7. Replayed DBOS step after an interrupted effect attempt.

Use a deterministic mock capability that can simulate acceptance, rejection, timeout, and crash windows.

Do not add real provider integration, OAuth, or external writes.

Do not implement the Phase 4 reconciliation worker. Phase 1 must correctly represent and quarantine UNKNOWN, make the state observable, and refuse blind retries. Record reconciliation as a later-phase obligation.

Do not promise exactly-once delivery to an external recipient.

### Workstream D — Approval as a durable framework protocol

Address F-02 and F-09.

Treat approval as a generic policy and lifecycle capability, not a sales-specific business rule.

Inspect the actual approval table, runtime lifecycle, DBOS signal implementation, and transaction boundaries.

The decision record must remain authoritative. A workflow signal is a notification that a decision exists, not the decision itself.

Prove these cases:

1. Approval decision commits successfully.
2. Process crashes before the workflow signal is delivered.
3. The process restarts and the decision is eventually delivered or recovered.
4. Duplicate signal delivery does not cause duplicate decisions or effects.
5. Duplicate approval decisions are consistently rejected or idempotently handled.
6. Approval rejection terminates or cancels the live workflow according to the generic contract.
7. Expiry is supported when an explicit expiry was supplied.
8. No implicit default expiry is invented.
9. A late decision cannot approve a terminal or cancelled invocation.
10. Approval, expiry, cancellation, and effect-claim races produce a consistent terminal outcome.
11. Approver scope and role are enforced if those fields are part of the contract.
12. Agent/authority attribution and self-approval restrictions remain intact.

The framework must ensure invocation and workflow lifecycle states converge or expose a recoverable discrepancy.

Do not invent business approver roles, approval thresholds, or expiry periods.

If the existing contract does not define a necessary behavior, record it as an Open Spec Item.

### Workstream E — DBOS family and version-safe recovery

Address F-05.

Inspect the pinned `dbos-transact-golang` v1.9.0 source and behavior. Do not rely on current online documentation or assumptions about another SDK version.

Follow `plan4.2` and the accepted G0 family/version topology.

Requirements:

1. Stable DBOS `AppName` per Class D workflow family.
2. Explicit `ApplicationVersion` behavior tied to the workflow compatibility line.
3. Clear distinction between miniapp manifest version, workflow definition version, and DBOS application version.
4. A defined compatibility and upgrade policy.
5. Old workflow artifacts remain available while old workflows are pending, unless a tested compatible patch is used.
6. Unknown or incompatible versions become visible and operator-actionable.
7. Version rejection must not silently strand durable work.
8. Preserve the accepted same-image/process-group topology; do not introduce an unapproved deployment topology.

Required test:

- Start a workflow under version N.
- Leave it pending at a durable wait.
- Start version N+1 under the accepted topology.
- Verify the documented compatibility/drain behavior.
- Verify the older run completes safely or is explicitly and observably handled.
- Verify unknown versions are not silently accepted or discarded.

Do not treat a string field or a unit test of version parsing as proof of version-safe recovery.

### Workstream F — Effect-time policy and authority

Address F-06.

Trace the complete path from invocation principal through DBOS execution to the final mock capability/effect claim.

Prove:

- Tenant and authority context are derived from trusted server-side state.
- No workflow payload can override trusted tenant, authority, or permission data.
- Authorization happens before dispatch.
- Policy and authority are revalidated before every consequential effect claim according to `plan4.2`.
- A policy change or authority revocation while a workflow is waiting prevents a new prohibited effect.
- Central kill-switch and blast-radius controls are enforced by the framework, not by optional miniapp checks.
- Tenant isolation holds under pooled connections and privileged-role execution.
- All refusals and blocked effects are auditable.

Use mock capabilities only. No production provider or egress work.

### Workstream G — Durable waits and status observability

Address F-10 and F-11.

First establish exactly what DBOS v1.9.0 does for durable waits and sleepers in the deployed edition. Separate verified SDK behavior from unverified assumptions.

Required sleep test:

1. Start a generic workflow.
2. Enter a durable sleep/wait.
3. Terminate the worker process.
4. Restart it against the same PostgreSQL state.
5. Verify the wait resumes according to documented SDK behavior.
6. Verify signal delivery before, during, and after the wait has defined behavior.
7. Verify no completed stage or confirmed effect is repeated.

Also assess representative concurrent sleeping-workflow counts and record:

- Worker memory behavior.
- Connection usage.
- Checkpoint/history growth.
- Restart/recovery time.
- Any edition-dependent limitations.

Do not invent a production capacity target. Record the measured envelope and unresolved scale requirements.

`Status` must distinguish:

- Fresh live executor status.
- Persisted status returned because the executor is unavailable.
- Unknown or stale state.
- Terminal state.

Do not silently swallow executor errors and return stale persisted state as if it were fresh.

### Workstream H — Determinism analyzer and operations

Address F-13.

Read `docs/phase0/determinism-rules.md` and implement the exact seven rule families defined by the approved specification.

Requirements:

- Use a Go analyzer compatible with the pinned toolchain and repository conventions.
- Understand the pinned DBOS v1.9.0 API and workflow boundaries.
- Produce positive and negative fixtures for every rule family.
- Avoid false positives on allowed deterministic patterns.
- Avoid relying on naming conventions that workflow authors can trivially bypass.
- Provide actionable diagnostics with source position and rationale.
- Run analyzer tests in the standard repository check.
- Add strict CI enforcement for Class D workflows.
- Verify that a violation cannot be hidden by moving a call through a trivial helper.

Do not invent additional banned constructs or silently omit a rule family. Record ambiguous rules as Open Spec Items.

Also document and prove the framework's operational controls:

- Retry and terminal-failure handling.
- Cancellation semantics.
- Workflow inspection.
- Recovery ownership.
- History retention.
- Stuck-workflow and stale-status alerting.
- Operator intervention and safe recovery.
- DBOS edition and version verification.

Do not introduce Enterprise-only functionality or dependencies without explicit owner approval.

### Workstream I — Framework/miniapp boundary

Address F-14.

Document the division of responsibility.

**Framework owns:**

- Canonical invocation.
- Trusted principals and tenant context.
- Authorization and policy enforcement.
- Idempotency and invocation/run binding.
- Static registration and workflow family/version registration.
- Generic entity-key/serialization enforcement primitives.
- Shared effects protocol and ledger.
- Approval protocol and lifecycle coupling.
- Audit attribution.
- Lifecycle operations.
- Determinism enforcement.
- Generic operational and recovery controls.

**Miniapps own:**

- Domain entities and state.
- Domain-specific workflow steps.
- Business eligibility and state transitions.
- Business-specific sequence/generation semantics.
- Tool input/output contracts.
- Declared entity keys and policy requirements.
- Use of framework effects and approval contracts.

Do not permit miniapps to create alternative datasources, authorization paths, lifecycle engines, or duplicate effects ledgers.

Update the architecture/implementation documentation where necessary, without changing `plan4.2`.

---

## 6. Queue partitioning: investigate, do not assume

Address F-12.

If the current framework does not use DBOS partitioned queues, do not introduce them merely to close a review item.

If partitioned queues are proposed:

- Inspect pinned SDK behavior.
- Test whether a long-waiting workflow occupies a partition slot.
- Test a second workflow submission targeting the same partition.
- Test whether queue ordering provides serialization, deduplication, both, or neither.
- Test different-entity parallelism.
- Measure any queue fairness or starvation effects.

The framework must enforce required entity uniqueness independently of workflow IDs and queue ordering.

Document the decision and evidence. Use the simplest mechanism that satisfies the invariant.

---

## 7. Build the generic Class D conformance probe

Once the shared framework defects that block a reliable probe are corrected, construct a generic test-only Class D conformance probe.

Do not call it `lead-followup`. Do not use sales-domain semantics.

The probe should be deliberately compact but exercise:

1. Canonical authenticated submission.
2. Durable run handle.
3. More than one dependent, checkpointed stage.
4. Durable wait.
5. Signal/event wait and resume.
6. Effect-time policy revalidation.
7. Mock effect through the shared effects ledger.
8. Approval wait and durable decision recovery, if approval is part of the generic contract being proved.
9. Cancellation with defined in-flight behavior.
10. Restart/replay from a meaningful checkpoint.
11. Version transition with a pending workflow.
12. Complete tenant/agent/authority/effect attribution.

This fixture exists only to test reusable framework behavior.

Do not introduce artificial fan-out, multiple providers, elaborate business rules, production quiet hours, or other domain complexity simply to make the workflow appear more DBOS-worthy.

The question is not whether this one probe justifies DBOS. The question is whether the generic framework contracts are strong enough to support more complex workloads without being bypassed or rewritten.

---

## 8. Preserve the original Phase 1 vertical slices

The `crm-read` Class A slice remains the synchronous conformance case.

The Class D probe remains the generic durable-execution conformance case until the ADR's status is decided.

Verify that both use the same canonical invocation and policy path.

Then complete the Phase 1 transport and harness requirements:

- REST adapter.
- MCP adapter.
- CLI adapter.
- One OpenCode demonstration using the accepted OpenCode version 1.19.6.
- Isolated OpenCode configuration and HOME directory.
- No changes to global OpenCode configuration.
- Demonstrate successful `crm-read`.
- Demonstrate an authorization refusal before dispatch.
- Demonstrate durable Class D submission and stable run handle.
- Preserve agent and authority attribution.
- Demonstrate lifecycle status and at least one signal/cancel operation through the supported interface.

All adapters must delegate to the same canonical invocation service. Add contract tests proving that adapters cannot bypass authorization, policy, audit, or idempotency.

Do not add a UI or any Phase 2+ integrations.

---

## 9. Validation and adversarial discipline

For every finding:

1. Write the regression test first where feasible.
2. Demonstrate that it fails against the existing defect.
3. Fix the root cause, not merely the observed symptom.
4. Demonstrate that the new test passes.
5. Run adjacent tests to detect collateral failures.
6. Run local PostgreSQL integration tests.
7. Run against the already-authorized Neon test branch if configured and available.
8. Keep the environments and evidence separate.

The full validation battery must include:

- `gofmt`.
- `go vet ./...`.
- `go test ./...`.
- Race detector.
- Repository checks.
- All seven analyzer rule families.
- Analyzer positive/negative fixtures.
- Strict CI mode for Class D.
- Canonical invocation contract tests.
- Auth-before-dispatch and cross-tenant tests.
- Invocation/run partial-failure tests.
- Approval decision/signal/recovery tests.
- Concurrent effects tests.
- Effect-time policy and kill-switch tests.
- Durable sleep/restart tests.
- Version-transition tests.
- Generic Class D conformance probe.
- REST/MCP/CLI contract tests.
- OpenCode end-to-end demonstration.
- Evidence-register validation.

A green unit suite is not evidence that skipped database integration tests passed.

A successful DBOS substrate test does not prove approval coupling, effect safety, or version-safe workflow recovery unless it explicitly exercises those behaviors.

Do not claim exact-once external delivery.

---

## 10. G1 evidence and acceptance

Create or update:

`docs/phase1/classd-framework-readiness.md`

It must provide a condition-by-condition assessment against the 14 G1 criteria in `plan4.2`.

For every G1 criterion include:

- Exact governing requirement.
- Implementation location.
- Test or operational drill.
- Source revision/working-tree digest.
- Execution environment.
- Exact command.
- Test result.
- Any skips.
- Evidence register reference.
- Verdict: PASS, PARTIAL, BLOCKED, or NOT PROVEN.

Do not report an overall G1 PASS if any mandatory criterion lacks evidence.

Also update `docs/phase1/classd-capability-matrix.md` so that each F-01 through F-14 finding links to its corrective implementation and regression evidence.

### Evidence recording rules

Every database or integration test record must identify:

- Source revision or exact working-tree digest.
- Local PostgreSQL or Neon environment.
- Database identity/class, without secrets.
- Exact command.
- Test names.
- Pass/fail/skip counts.
- Failure disposition.

Never place passwords, DSNs, access tokens, or credentials in reports or logs.

If a configured environment is unavailable, record the limitation and leave the related criterion unproven. Do not substitute a unit test for a required real-PostgreSQL test.

Keep the earlier successful increment 4–5 local PostgreSQL run as a distinct historical evidence item. Re-run the current tree before using it for G1 acceptance.

---

## 11. Scope restrictions

Remain within Phase 1.

Explicitly out of scope:

- Real provider integrations or OAuth.
- Real external writes.
- Production effect reconciliation worker from Phase 4.
- River and Class B/Class C work from Phase 3.
- Sprite workspace or arbitrary user code.
- Promotion/trusted-build pipeline.
- Production Fly deployment drills from Phase 6.
- General workflow-engine or BPMN replacement.
- UI framework or UI generator.
- AI miniapp generator.
- Real sales playbook and production lead-follow-up policy.
- Any changes to `plan4.2.md`.
- Any ADR approval or reviewer signature.

Do not expand scope because a proposed solution seems architecturally interesting.

When a missing capability belongs to a later phase, document the boundary and ensure the current phase enforces the correct fail-closed behavior.

---

## 12. Required final report

Return a detailed report with these sections:

### A. Executive result

State whether the Phase 1 framework is now:

- Not Class D-ready.
- Partially Class D-ready.
- Class D-ready for the defined Phase 1 conformance envelope.

Do not generalize beyond the tested envelope.

### B. Baseline and change inventory

Report HEAD, ancestry, initial dirty state, modified/added files, and confirmation that no earlier work was overwritten.

### C. F-01 to F-14 disposition

For every finding:

- Original finding.
- Verified or disproven by source inspection.
- Root cause.
- Code changes.
- Regression test.
- Evidence.
- Remaining risk.
- Final disposition.

### D. Framework capability matrix

Provide the completed matrix and identify any capability not demonstrated.

### E. Adversarial test results

Include the actual test command, environment, results, and fault-injection behavior. Separate local, Neon, unit, race, and OpenCode evidence.

### F. Generic Class D probe

Describe its execution path, lifecycle, failure tests, and proven guarantees. Explicitly state what it does not prove.

### G. Determinism analyzer

Map the seven approved rule families to implementation, fixtures, diagnostics, and CI enforcement.

### H. G1 matrix

Provide all 14 criteria with independent PASS/PARTIAL/BLOCKED/NOT PROVEN verdicts.

### I. Open Spec Items and owner decisions

List unresolved semantics, technical policy choices, and named reviewer requirements. Do not invent decisions.

### J. Phase boundary and recommendation

Identify exactly what remains before:

- G1 can be accepted.
- A human can approve a revised PILOT ADR.
- Increment 6 can begin.

### K. Change and evidence inventory

No commits or pushes. Report all uncommitted changes and all evidence files.

---

## 13. Execution rules

- Work in coherent vertical slices; do not scaffold every package before proving an integrated behavior.
- Investigate root causes and system-level failure modes.
- Require regression tests that demonstrate the invariant.
- Avoid superficial workarounds that leave the underlying defect intact.
- Preserve existing tests and contracts.
- Never silently weaken an assertion, delete a failing test, or turn a required integration test into a skip.
- Do not hide a failure behind a mock if the governing requirement calls for real PostgreSQL or process restart.
- Do not fabricate test results or claim commands ran when they did not.
- Do not use the review findings as permission to change settled architecture.
- Do not introduce framework abstractions without a concrete Phase 1 invariant they enforce.
- If the source leaves behavior undefined, create or update an Open Spec Item rather than inventing the answer.
- If a design requires an owner decision, stop that specific design branch and continue independent framework work.
- Preserve all existing uncommitted user work.

**Do not commit or push anything.**

---

## 14. Final decision boundary

This task may remediate and prove the generic Class D framework. It does not approve the `lead-followup` pilot or a production sales workload.

At completion, explicitly present these two unresolved owner decisions:

1. Should `lead-followup` be approved as a synthetic Phase 1 PILOT/conformance fixture, or is a different Class D fixture preferred?
2. Who is the named human reviewer responsible for the ADR decision required by `plan4.2`?

Do not block independent generic framework work while these decisions are pending.

However, do not revise/sign the ADR, create the `lead-followup` manifest, or start increment 6 until the owner has explicitly resolved the fixture choice and the authorized reviewer has approved the revised ADR.

**The objective is not to make `lead-followup` look like a complex workflow. The objective is to make the framework demonstrably capable of safely supporting workflows for which durable orchestration is genuinely warranted.**

The key sequencing decision is deliberate: fix and prove the shared framework first; keep the sales fixture's business semantics out of those fixes. The pilot-vs-production choice and reviewer designation remain owner decisions, but they need not block generic framework remediation.