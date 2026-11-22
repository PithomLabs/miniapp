I recommend making Prompt 8 a **formal G0-Core acceptance and owner-adjudication task**, not another implementation sprint.

The most important correction is condition 4: plan4.2 requires a documented workflow-version, retention/drain and safe deployment-topology strategy at G0. The actual Fly deployment-overlap and recovery drill remains a G6 activity. We should not defer the G0 design requirement just because its full operational proof comes later.

Here is the prompt.

# Prompt 8 — G0-Core Final Adjudication, Owner Sign-Off and Evidence Freeze

## Role

Act as the principal architect and technical acceptance reviewer for:

`github.com/PithomLabs/miniapp`

Your responsibility is to determine whether Phase 0 has satisfied every mandatory G0-Core acceptance condition in `docs/plan4.2.md` §23.1.

This is the final G0-Core closure task.

The repository has undergone:

- Phase 0 architecture and contract implementation.
- Adversarial implementation review and Gate A remediation.
- Neon DBOS/River/RLS verification.
- Additional DBOS shared-handle and concurrent-tenant regression tests.

The objective is no longer to implement more infrastructure. It is to establish a defensible acceptance decision, resolve outstanding owner approvals, and freeze the evidence supporting that decision.

**Do not manufacture a PASS. An honest NOT PASSED verdict is preferable to an unsupported acceptance.**

## 1. Governing authority

Read these documents from the repository before starting:

1. `docs/plan4.2.md` — authoritative architecture and G0-Core acceptance criteria.
2. `docs/phase0/g0-evidence.md` — current G0 tracker.
3. Existing evidence-register files under `docs/evidence/`.
4. `docs/adv_review_prompt5.md` — findings and historical review evidence.
5. The original Prompt 3 report or governing prompt.
6. DBOS, RLS, Neon, family/version and MCP spike reports.
7. The current ADRs and dependency/version register.
8. The Neon and Sprite runbooks.

Locate the exact current paths. Do not rely on earlier conversation summaries where repository evidence is available.

### Precedence

- `docs/plan4.2.md` controls architecture and acceptance.
- Its §23.1 defines the eleven G0-Core pass conditions.
- Its §23.2 defines later phase-specific proof gates.
- The tracker records evidence and status but cannot change the governing acceptance criteria.
- Owner decisions must be explicitly recorded; an agent cannot infer consent.

Do not modify `docs/plan4.2.md`, `docs/adv_review_prompt5.md`, historical ADRs, or accepted Gate A implementation.

## 2. Scope and authorization

This is an acceptance, evidence and documentation task.

Authorized:

- Inspect the full repository and relevant execution artifacts.
- Reconcile the tracker with the fresh Neon and local test evidence.
- Verify that Gate A remains accepted.
- Review the core contract, dependency, security, family/version, harness and lifecycle evidence.
- Prepare precise owner-decision requests.
- Update `docs/phase0/g0-evidence.md`.
- Update the existing evidence register under `docs/evidence/`.
- Create one narrowly scoped adjudication report, if needed:\
  `docs/phase0/g0-core-adjudication.md`.

Not authorized:

- Phase 1 implementation.
- Source-code changes.
- Test modifications or new test additions.
- Dependency changes.
- Architecture redesign.
- ADR rewrites.
- Sprite implementation.
- Fly deployment or production wiring.
- Neon project, branch, schema or credential mutations.
- Secret modifications.
- Git staging, committing, tagging or pushing.

Do not rerun expensive cloud tests merely to repeat already valid evidence. Re-run a test only when the existing evidence cannot be independently verified, is incomplete, or no longer matches the source revision being adjudicated.

If new source changes or tests appear necessary, document the gap and stop that workstream for separate authorization.

## 3. Establish the current repository and evidence baseline

Before changing documentation:

1. Record current branch and `git status --short`.
2. Inventory all repository files.
3. Verify the Gate A changed-file inventory.
4. Confirm that:
   - M1–M3 and L1/L3/L4 were implemented.
   - Their regression tests remain present.
   - The accepted Gate A implementation has not regressed.
   - `docs/plan4.2.md` remains unchanged.
   - `docs/adv_review_prompt5.md` remains unchanged.
5. Confirm the two latest authorized test files exist:
   - `spikes/dbos_shared_test.go`
   - `spikes/rls_concurrent_test.go`
6. Inspect the exact evidence register entries from the latest Neon sprint and Prompt 7 execution.
7. Identify the actual source state against which every test ran.

Do not treat a test report as source-revision evidence unless its source revision can be established.

The repository was previously reported to have zero commits. Reconfirm this.

Do not create the first commit during this task.

## 4. Build the authoritative G0-Core adjudication matrix

Before updating any status, create a matrix for all eleven conditions in plan4.2 §23.1.

Use only these six evidence states:

- PASS — mandatory requirement satisfied by current, independently verifiable evidence.
- PARTIAL — some requirements satisfied, but at least one required element remains unresolved.
- BLOCKED — cannot proceed because a specific external decision, prerequisite or access requirement is unavailable.
- FAIL — executed evidence demonstrates that the requirement is not satisfied.
- SKIPPED — the relevant check was intentionally not executed or was skipped.
- NOT RUN — no execution evidence exists.

Each condition must include:

| Field                   | Required content                                  |
| ----------------------- | ------------------------------------------------- |
| Condition               | Exact plan4.2 §23.1 number and wording            |
| Governing requirement   | What must actually be true                        |
| Existing implementation | Relevant source, contract, ADR or design artifact |
| Behavioral evidence     | Exact test/report and environment                 |
| Current status          | One of the six states                             |
| Outstanding gap         | Exact missing item, if any                        |
| Owner dependency        | Named decision or sign-off required               |
| Completion evidence     | What would allow promotion to PASS                |
| Next action             | Specific and executable                           |

Do not collapse multiple conditions into a generic statement such as "Phase 0 is complete."

Do not use later-phase requirements to hold G0-Core hostage.

Do not waive a G0 requirement because a later test is difficult.

## 5. Adjudicate the eleven G0-Core conditions

Use the following as the authoritative acceptance checklist.

### Condition 1 — Core contracts reviewed and versioned

Required:

Core Class A/D contracts, identity/authority model, synchronous/asynchronous invocation, effect record, entity-key semantics, registry model, and non-circular manifest-to-build release binding are reviewed and versioned.

Inspect:

- `contract/`
- ADRs governing the relevant architecture.
- JSON Schemas.
- Contract regression tests.
- Release-envelope implementation.
- Entity-key and `no_entity` semantics.
- Identity and authority documentation.

Produce a review checklist mapping every required contract element to its implementation and evidence.

Distinguish:

- Implementation exists.
- Automated contract tests pass.
- Architecture was reviewed.
- Human acceptance was recorded.

Do not equate successful Go tests with human review sign-off.

Required owner action: obtain an explicit review/sign-off from the authorized architect or owner, with a dated acceptance record.

### Condition 2 — Phase 1 dependencies pinned

Required:

Every Phase 1 dependency has its exact version, license, verification date, upstream source, and upgrade owner recorded.

Inspect:

- `go.mod` and `go.sum`.
- Dependency/version register.
- Upstream-source and license evidence.
- CI dependency checks.
- Phase 1-specific dependencies versus later optional dependencies.

Verify that every dependency actually needed by Phase 1 is covered.

Do not pull optional Sprite or Phase 2 dependencies into the core pin requirement merely because they appear in the module.

Report any missing upgrade owner or stale verification record.

Promote to PASS only when every required field is established.

### Condition 3 — DBOS/Neon shared-handle atomicity and crash probes

Required:

Prove application transaction integration at the actual shared pool-handle boundary through crash probes.

Review the latest evidence from:

- Existing DBOS atomic commit and rollback tests.
- `spikes/dbos_shared_test.go`.
- The separate-pool negative control.
- Neon direct-connection execution.
- Local PostgreSQL execution.
- Crash-recovery execution and its durable-state assertions.

Verify:

- Actual shared-engine detection.
- Commit durable state.
- Rollback durable state.
- Completed-step metadata.
- Negative control distinguishing separate-pool construction.
- The exact DBOS SDK version.
- Crash/recovery assertions.

Local and Neon evidence must remain separate.

Preserve the historical November 22, 16:19 atomicity observation and the later fresh Neon execution as separate events.

The report's shared-pool shutdown-ordering observation must remain documented as a Phase 1 executor lifecycle constraint. Do not characterize it as a proven framework defect without supporting evidence.

Promote to PASS only for the exact shared-handle and crash-probe requirements in §23.1.

### Condition 4 — DBOS family/version/deployment isolation strategy

Required:

Each Class D family has:

- A stable DBOS `AppName`.
- An explicit workflow-version policy.
- An old-version retention/drain strategy.
- Proven same-image worker process-group isolation, or an explicitly revised safe deployment unit.

This condition requires careful adjudication.

Do not incorrectly defer all family/version requirements to G6.

Separate two levels:

**G0 design and architecture requirement**

- Per-family stable `AppName`.
- Explicit workflow-version policy.
- Compatibility and old-version retention rules.
- Drain strategy.
- Safe deployment topology.
- Evidence or an explicit architectural revision if the selected topology cannot guarantee isolation.

**G6 operational proof**

- Live Fly deployment overlap.
- Graceful shutdown and worker drain.
- Actual deployment/recovery interaction.
- Verification that incompatible versions cannot race.

G0 does not automatically require the full G6 live production drill.

Inspect the family/version plan, ADR-016 and relevant spikes.

Prepare a concise owner adjudication explaining:

1. What has already been proven.
2. What is documented but not yet behaviorally demonstrated.
3. What must be decided now for G0.
4. What is specifically deferred to G6.

Do not mark PASS solely because AppName isolation passed if the required workflow-version and retention strategy remains undefined.

### Condition 5 — Tenant transaction-local context and RLS under concurrency

Required:

Tenant transaction-local context and RLS behavior are verified under the chosen connection mode and concurrency.

Inspect:

- `internal/store/WithTenant`.
- Existing RLS tests.
- `spikes/rls_concurrent_test.go`.
- Default-deny evidence.
- `NOBYPASSRLS` role configuration.
- Concurrent barrier execution.
- Negative controls.
- Durable per-tenant assertions.

Verify that the evidence demonstrates actual concurrent overlap rather than sequential execution.

Preserve the earlier local and Neon test results separately.

Promote to PASS only if the current implementation and evidence establish the required tenant isolation invariant.

### Condition 6 — Initial MCP/REST invocation round trip with first target harness

Required:

The initial MCP/REST invocation round trip works with the first target harness.

The larger two-family harness matrix is explicitly a G6 requirement.

Inspect:

- Existing MCP round-trip evidence.
- Authentication and authorization path.
- Transport and registration setup.
- Harness-specific configuration and invocation.
- Actual result or run-handle response.

Identify the exact harness that has been selected.

Do not silently choose one on behalf of the owner.

An in-process MCP SDK test is not automatically equivalent to a round trip through the chosen external harness.

If the harness has not been selected, classify the condition as PARTIAL or BLOCKED according to what evidence exists.

Required owner decision: name the first target harness.

After selection, define the smallest necessary acceptance check and record its actual evidence.

Do not require the full Phase 6 two-family compatibility matrix.

### Condition 7 — Named security owner signs the core threat model

Required:

The named security owner signs the core threat model.

Current known owner question: Q9.

Do not infer a security owner from the person who wrote the threat model, implemented security controls, or ran tests.

The owner must explicitly name the accountable security owner.

Acceptance evidence must include:

- Named individual.
- Role and accountability.
- Exact threat-model document and revision.
- Dated approval.
- Any remaining reservations.

Without an explicit owner and signature, this condition cannot pass.

Do not sign the threat model on the owner's behalf.

### Condition 8 — Prompt 2 truncation resolved

Required:

The truncated `prompt2.md` must be resolved through either:

- An owner-supplied missing remainder; or
- Explicit owner acceptance of Appendix D.

Current owner question: Q8.

Inspect the exact truncation point and Appendix D.

Prepare an explicit acceptance request that states:

- `prompt2.md` ends at §7.1.
- The remainder has not been supplied.
- Appendix D contains architect-authored completions D1–D4.
- Those completions are not original prompt text.
- Accepting Appendix D means accepting those completions as the governing replacement for the missing instructions.

The owner must either supply the missing text or explicitly accept Appendix D.

Silence, previous discussion, or an agent's recommendation is not acceptance.

Do not modify `prompt2.md` or Appendix D.

### Condition 9 — ADR-001 Go/Postgres/River port and decision responsibilities

Required:

ADR-001 is ported for Go/Postgres/River, and per-Class-C/D decision records and human approval responsibilities are defined.

Inspect:

- `docs/adr/001-dbos-adoption.md`.
- Miniapp ADR template and inventory.
- Class C/D execution decision rules.
- Human review and approval responsibility.

Verify alignment with plan4.2 and the implemented stack.

The Phase 3 River atomic-enqueue proof is not required for G0-Core.

Do not defer the definition of the decision process merely because a later workload-specific decision happens in Phase 3.

### Condition 10 — A15/A16 design recorded

Required:

A15 binary/registry and A16 golden-path/CI design are recorded.

Inspect:

- Relevant ADRs and architecture records.
- CLI and registry design.
- Golden-path/lifecycle design.
- CI/CD service and policy parity.
- Environment and migration ordering.
- Rollback responsibilities.

Do not claim full A16 operational implementation. G0 requires the design to be recorded; the complete golden path is later exercised in G6.

Promote to PASS only if both designs are actually represented in the authoritative artifacts.

### Condition 11 — Every core spike has dated evidence

Required:

Every core spike must have a dated report with:

- Test names.
- Source revision.
- Environment.
- Commands.
- Results.
- Failure disposition.

Review every core spike, not just the three latest tests.

Build a coverage matrix that maps each core spike to its evidence artifact.

The current repository has been reported to have zero commits. Verify whether a source revision can be established independently and durably.

A timestamp, Git branch name, or mutable working-tree checksum is not automatically a Git source revision.

Do not manufacture a commit or source revision.

Prepare a first-commit authorization request that identifies:

- Exact proposed source-tree scope.
- Expected source revision benefit.
- Whether documentation evidence should be included.
- Any paths that should remain excluded.
- The final pre-commit verification procedure.

Do not stage or commit without explicit owner authorization.

Until a valid source revision is recorded and every core spike is tied to the appropriate source state, condition 11 cannot be promoted to PASS.

## 6. Owner decision register

Build a single current owner-decision table.

At minimum, evaluate the following:

| Item          | Decision                                                   | G0 relationship                                                                              |
| ------------- | ---------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Q7            | CLI assessment/keep-list sign-off                          | Verify whether it is actually a mandatory §23.1 condition before treating it as a G0 blocker |
| Q8            | Accept Appendix D or provide missing Prompt 2 text         | Direct G0 condition 8                                                                        |
| Q9            | Name security owner and obtain threat-model signature      | Direct G0 condition 7                                                                        |
| First harness | Select initial MCP/REST harness                            | Direct G0 condition 6                                                                        |
| Condition 1   | Formal core contract review and sign-off                   | Direct G0 condition 1                                                                        |
| Condition 4   | Workflow-version and safe deployment topology adjudication | Direct G0 condition 4                                                                        |
| First commit  | Authorize first Git commit/source revision                 | Evidence dependency for condition 11                                                         |
| Neon Q4       | Neon project/region for G0 testing                         | Resolved according to the latest owner-confirmed sprint report                               |

For Q7, Q1–Q6, and every other Appendix B decision, inspect plan4.2 and distinguish its actual phase dependency.

Do not elevate a later-phase question into a G0 blocker without a direct governing requirement.

In particular:

- Q1 first SaaS belongs to Phase 2.
- Q2 two-family harness matrix belongs to Phase 6.
- Q3 DBOS Enterprise budget is conditional on later wake-test evidence.
- Q4 Neon is resolved; Fly organization/region belongs to later Fly deployment work, while Sprite organization/quota is needed before Phase 4 workspace work.
- Q5 is housekeeping unless another governing requirement makes it mandatory.
- Q6 must be classified according to its actual catalog dependency.

The final owner table must contain:

- Exact decision requested.
- Exact governing-document source.
- Who is authorized to decide.
- Evidence of completion.
- Current status.
- Whether it blocks G0-Core.
- Next action.

Do not record an owner approval until it is explicitly supplied.

## 7. Separate G0 from later phases

This is mandatory to prevent scope creep.

Use `docs/plan4.2.md` §19, §23.1 and §23.2.

| Gate/phase   | Scope                                                                        | Must not be pulled into G0                                              |
| ------------ | ---------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| G0-Core      | Core Class A/D contracts and required proofs                                 | —                                                                       |
| Phase 1 / G1 | Core vertical slice and lifecycle bootstrap                                  | Real provider writes                                                    |
| Phase 2 / G2 | Provider security and read-only integrations                                 | First SaaS OAuth as G0 requirement                                      |
| Phase 3 / G3 | River jobs and Class C comparison                                            | River atomic enqueue as G0 requirement                                  |
| Phase 4 / G4 | Effects, approvals, first external writes, optional Sprite workspace preview | Sprite preview as universal G0 requirement                              |
| Phase 5 / G5 | Catalog expansion, bounded Sprite tasks and promotion                        | Sprite tasks as G0 requirement                                          |
| Phase 6 / G6 | Production falsification and full lifecycle acceptance                       | Two-harness matrix and live Fly deploy-overlap drill as G0 requirements |

A later phase may rely on a G0 design decision without requiring the later phase's operational proof early.

The failure of an optional extension must defer that extension, not block an unrelated core phase.

## 8. Owner decision packet

Prepare a concise, self-contained owner decision packet.

Separate decisions into:

**A. Mandatory G0 decisions**

- Q8.
- Q9.
- Initial harness.
- Condition 1 review/sign-off.
- Condition 4 architecture adjudication.
- First-commit authorization/source revision.

**B. Decisions to classify but not automatically treat as G0 blockers**

- Q7 CLI sign-off.
- Q1–Q6 and any other Appendix B items.
- Future Fly and Sprite organization decisions.

For each mandatory decision:

- State the issue in plain English.
- Quote or precisely reference the governing requirement.
- Explain available alternatives.
- Explain what each choice means.
- Identify the exact evidence needed afterward.

Do not recommend accepting a decision merely to achieve a PASS.

Where an architectural decision requires human judgment, present the evidence and tradeoffs without silently deciding.

## 9. Evidence freeze and documentation updates

After completing the audit and adjudication:

Update only:

- `docs/phase0/g0-evidence.md`
- Existing appropriate files under `docs/evidence/`
- `docs/phase0/g0-core-adjudication.md` if required.

The report must:

- Preserve the full historical Neon incident timeline.
- Preserve the original 16:19 DBOS observation and later fresh runs as distinct evidence.
- Reference both new tests:
  - `spikes/dbos_shared_test.go`
  - `spikes/rls_concurrent_test.go`
- Reference the exact Neon and local executions.
- Record SDK and database version where available.
- Distinguish design review from behavioral evidence.
- Record source-revision limitations.
- Keep owner approvals pending until actually obtained.

Do not overwrite an old test result with a newer one. Add a new dated entry and retain history.

If the tracker already accurately records an item, do not rewrite it merely for stylistic consistency.

## 10. Final G0-Core verdict

There are exactly two valid verdicts:

### G0-Core PASS

Only when all eleven mandatory conditions in plan4.2 §23.1 are independently satisfied.

List the evidence for every condition.

### G0-Core NOT PASSED

If any mandatory condition is PARTIAL, BLOCKED, FAIL, SKIPPED or NOT RUN.

Identify the exact outstanding requirements and owner actions.

Do not introduce a third verdict such as "almost passed" or "technical pass."

You may summarize technical readiness separately, but the formal gate verdict must be one of the two above.

## 11. Final report — seven deliverables

Follow Prompt 3's seven-deliverable structure.

### 1. Repository audit

- Current source state.
- Gate A status.
- File inventory and Git state.
- Governing document integrity.
- Evidence inventory.

### 2. Change summary

- Exact documentation files updated.
- Exact tracker/evidence records added.
- No implementation changes.

### 3. Verification

- Evidence reviewed and any tests rerun.
- Exact commands and environment.
- PASS/FAIL/BLOCKED/SKIPPED/NOT RUN.
- No conflation of historical, local and Neon results.

### 4. G0 tracker

- All eleven conditions.
- Six-state status classification.
- Required completion evidence.
- Outstanding gaps.

### 5. Owner actions

- Mandatory G0 decisions.
- Non-G0 owner questions.
- Exact acceptance wording and evidence needed.

### 6. Runbooks

- Neon runbook status and current test-branch details.
- Sprite runbook status and explicit later-phase boundary.
- No unperformed work presented as complete.

### 7. Readiness

- Exactly one formal verdict.
- Decisive evidence.
- Remaining actions.
- Whether Phase 1 is authorized by the governing G0 rule.

## 12. Integrity and safety constraints

- Do not modify `docs/plan4.2.md`.
- Do not modify `docs/adv_review_prompt5.md`.
- Do not alter the accepted Gate A implementation.
- Do not modify source code.
- Do not add tests.
- Do not modify dependencies.
- Do not modify Neon, Fly, Sprite, or secrets.
- Do not stage, commit, tag or push.
- Do not infer owner consent.
- Do not claim full G0 merely because the technical spikes are green.
- Do not pull G2–G6 proof requirements into G0.
- Do not remove or weaken any acceptance criterion.
- Preserve every historical evidence record.

**Final instruction:** G0-Core is an acceptance gate, not a paperwork exercise. Your job is to establish exactly which requirements are satisfied, which require human decisions, and which lack behavioral evidence. Close only what the evidence supports. If the owner must act, give the owner a precise decision request and leave the gate NOT PASSED.

## **What I expect this task to accomplish**

The likely outcome is a clean distinction between three categories:

**Technically proven:** DBOS shared-handle transaction behavior, crash probes, tenant/RLS concurrency, dependency pins, ADR-001 and A15/A16 records, subject to final evidence verification.

**Human acceptance required:** core contract sign-off, condition-4 topology adjudication, Q8, Q9 and first-harness selection.

**Provenance required:** condition 11's dated spike reports bound to a real source revision. This is why first-commit authorization matters.

One important point: **Q7 should not automatically block G0-Core.** Prompt8 tells the agent to trace it to the actual eleven conditions rather than elevate every open Appendix B question into a core gate.

The desired result is a defensible G0 acceptance record—not another round of implementation.