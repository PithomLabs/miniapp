The ADR is correctly marked **DRAFT — NOT APPROVED**. The review should challenge both the business case for DBOS and the technical guarantees being proposed. In particular, it must scrutinize the phrase “exactly-once semantics” and the proposed per-lead serialization mechanism rather than accepting either as established.&#x20;

lead-followup-adr001.md



Here is a prompt for an independent, adversarial review. It deliberately keeps the increment 5 hard stop in force.

# Prompt 11 — Adversarial Architecture and Business Review: `lead-followup` ADR-001

## Role

Act as an independent Principal Software Architect, distributed-systems reviewer, and business-process analyst reviewing the proposed ADR-001 decision for `lead-followup`.

Your job is not to make the ADR look acceptable. Your job is to determine whether adopting DBOS for this workload is justified by actual business requirements, whether the proposed workflow semantics are sufficiently defined, and whether the design can satisfy the reliability guarantees it claims.

Be skeptical. Challenge assumptions, identify contradictions, distinguish proven capabilities from proposed behavior, and expose hidden complexity.

The final recommendation must be defensible to both a business owner and an experienced distributed-systems engineer.

## 1. Review scope and current state

Review these materials in the repository:

**Primary document**

- `docs/adr/miniapps/lead-followup-adr001.md`

**Governing architecture**

- `docs/plan4.2.md` — authoritative and immutable.
- `docs/adr/miniapps/adr001-template.md`.
- `app_category_guidelines.md` and any applicable ADR-001 guidance.
- Relevant G0 adjudication and accepted architecture decisions.

**Phase 1 implementation and evidence**

- `runtime/service.go`
- `runtime/lifecycle.go`
- `executor/dbos/executor.go`
- DBOS workflow registration and versioning implementation.
- `approvals/`, `effects/`, and `audit/` implementations.
- Relevant tests and PostgreSQL spike evidence, particularly shared-pool behavior, transaction boundaries, tenant isolation, and shutdown ordering.

Inspect the actual repository state. Do not rely solely on the summary below or on claims in the ADR.

### Current implementation status

Increments 4–5 have been reported complete.

**Class A — `crm-read`**

- Canonical invocation service with invocation, idempotent replay, approval gating, and dispatch.
- Lifecycle service for status, signal, cancel, and approval decisions.
- `crm-read` manifest, handler, and static registration.
- Runtime tests covering happy path, replay, rejection classes, approval grant/reject/self-decision, cross-tenant isolation, and Class D lifecycle against a fake executor.
- `crm-read` tests covering manifest validity, registration, and tenant-isolated search.

**DBOS substrate**

- `executor/dbos/executor.go` supports submit, status, signal, and cancel.
- Shared-pool, single-transaction configuration.
- Explicit shutdown-before-pool-close ordering.
- Tenant-context step helper.
- Typed workflow start and per-tool registration using the shared datasource.
- PostgreSQL-backed tests covering durable writes, signal delivery, cancellation, configuration rejection, and shared-handle detection.

Two implementation findings must be considered:

1. Datasource provisioning belongs to the executor rather than individual miniapps, because per-tool provisioning causes duplicate registration.
2. Closing the shared pool before stopping and draining DBOS workers can deadlock.

**ADR status**

- `docs/adr/miniapps/lead-followup-adr001.md` is DRAFT.
- It proposes adoption of DBOS with a score of 12/14.
- Its business playbook semantics are explicitly `[PROPOSED]`.
- Its §7 acceptance tests are planned, not yet written.
- The reviewer and approval date are pending.

### Absolute hard stop

No `lead-followup` workflow code, manifest, registration, or DBOS execution may be created or enabled until an authorized human explicitly approves the workload-specific ADR.

This review does not confer approval. Do not interpret a positive recommendation, a technically complete ADR, or your own assessment as human authorization.

Do not proceed to increment 6.

## 2. Evaluate the ADR-001 decision gates

Use the exact governing definitions and thresholds in `plan4.2.md` and the ADR guidance. Do not substitute your own criteria.

Evaluate all four gates independently:

### Gate 1 — Durable need

Does this workload genuinely require durable execution beyond a request-bound operation or an ordinary background job?

Examine:

- Whether multi-day waits are a real business requirement.
- Whether a sequence of dependent stages is necessary.
- Whether restart and deployment recovery are meaningful business requirements.
- Whether the process could instead be represented as independent Class B jobs or a simpler state machine.

Do not assume that a process is a durable workflow merely because it contains delays or multiple steps.

### Gate 2 — Workflow characteristics

Verify the claimed characteristics individually:

- Waiting.
- Recovery.
- External effects.
- Coordination.

For each characteristic, identify the actual business requirement and the technical mechanism needed to satisfy it.

Do not award points merely because the ADR describes a feature that DBOS supports. The workload must genuinely need that feature.

### Gate 3 — Business consequence

Challenge the stated consequences:

- A lost follow-up means lost revenue or attribution.
- A duplicate touch means customer-facing spam.
- An out-of-window touch violates the playbook.

Determine which consequences are established business requirements and which are assumptions awaiting owner confirmation.

Assess their likelihood, severity, reversibility, and operational consequences. Identify any missing consequences that should influence the decision.

### Gate 4 — Net complexity comparison

Compare the DBOS proposal with the non-DBOS alternative honestly.

The ADR estimates:

- Alternative A: approximately 2–3 weeks for bespoke durable machinery and ongoing ownership.
- Alternative B: approximately one week for workflow, connectors, eligibility, approval, and audit wiring.

Challenge these estimates.

Account for the full lifecycle cost, not just initial implementation:

- Development and testing.
- Operations and monitoring.
- Recovery and reconciliation.
- Version upgrades and compatibility.
- Database and connection-pool management.
- Incident response.
- Long-term ownership.
- New failure modes introduced by the chosen architecture.

Identify whether any substantial DBOS-related operational costs or custom machinery have been omitted.

Conclude whether DBOS produces a genuine net reduction in complexity for this particular workload.

## 3. Challenge the proposed business playbook

The ADR explicitly says that no approved source currently specifies the follow-up sequence.

Review every unresolved business semantic. Do not invent values or infer owner intent.

Require explicit business-owner confirmation for:

| Business decision | Questions that must be resolved                                                                                |
| ----------------- | -------------------------------------------------------------------------------------------------------------- |
| Number of touches | How many follow-ups constitute one sequence?                                                                   |
| Timing            | What are the exact intervals and deadlines?                                                                    |
| Channels          | Which channels are allowed, in what order, and what happens when one is unavailable?                           |
| Quiet hours       | Which timezone applies? What days, holidays, and local restrictions apply?                                     |
| Eligibility       | What conditions make a lead eligible at submission and before each subsequent touch?                           |
| Exit conditions   | What events terminate the sequence? For example, conversion, opt-out, invalid contact, or loss of eligibility. |
| Escalation        | Under what conditions is a lead escalated, to whom, and through which mechanism?                               |
| Approval          | Which touches require approval? Who is authorized to approve? What happens on rejection or expiry?             |
| Cancellation      | Who can cancel a sequence, and what happens to an in-progress or uncertain touch?                              |
| Failure behavior  | What happens when a channel is unavailable, a send fails, or its outcome cannot be determined?                 |

Also determine whether additional rules are necessary, such as consent, suppression lists, duplicate-contact limits, lead ownership changes, and changes to contact preferences.

For each item, classify it as:

- Already defined by an authoritative business source.
- Explicitly proposed but awaiting approval.
- Missing and required before implementation.
- Not necessary for this workload, with justification.

Do not approve a business playbook simply because the proposed sequence sounds reasonable.

Produce a concise owner-decision table containing the exact decisions needed to make the workflow implementable.

## 4. Challenge the exactly-once claims

This is a critical review area.

The ADR states that external sends require exactly-once semantics per touch and proposes recording the touch before sending through transaction steps.

Determine precisely what guarantee is achievable.

Examine these failure windows:

1. The database records an intended touch, but the process crashes before sending.
2. The channel accepts the message, but the process crashes before recording confirmation.
3. The channel times out and its actual outcome is unknown.
4. DBOS replays a completed or interrupted workflow step.
5. Two workers attempt the same touch concurrently.
6. A retry occurs after a network partition or ambiguous response.
7. A cancellation or approval rejection races with an in-progress send.

Distinguish:

- Exactly-once workflow-step execution.
- Exactly-once database state transition.
- At-most-once external send attempt.
- Provider-supported idempotent delivery.
- Eventual reconciliation of an uncertain external effect.

Do not equate a PostgreSQL transaction with atomicity across an external messaging provider.

Compare the design against the governing effect-state discipline:

`PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN`

In particular:

- Is the effect intent durably recorded before the external operation?
- How is a successful but unacknowledged send reconciled?
- Is an `UNKNOWN` result quarantined rather than blindly retried?
- What evidence permits a retry?
- Is provider idempotency available, or is this guarantee limited to mock connectors?
- What happens when reconciliation cannot establish the outcome?

The design must not make an exactly-once production-delivery promise that its mechanisms cannot support.

Recommend precise replacement language for any overstated guarantee.

Remember that Phase 1 uses mock channels only. Do not introduce production provider integration or external writes.

## 5. Challenge per-lead serialization and identity

The ADR proposes per-lead serialization through either:

- A deterministic workflow ID derived from the lead.
- A partitioned DBOS queue keyed by lead.

It also proposes a stable run handle derived per invocation.

These are not automatically equivalent guarantees.

Analyze the distinction between:

- Invocation identity.
- Run/workflow identity.
- Business entity identity.
- Effect/touch identity.
- Idempotency identity.

Determine whether the ADR accidentally conflates the identity of one invocation with the identity of a lead's entire follow-up sequence.

Explicitly test this scenario:

Two agents submit follow-up requests concurrently for the same lead, possibly with different invocation IDs and different request idempotency keys.

Questions:

- Do they create two workflows?
- Does one return the existing active workflow?
- What happens if the existing workflow has completed?
- Can a later, legitimate follow-up campaign be started?
- What defines a new sequence generation?
- What prevents two active sequences from sending concurrently?
- What happens if the sequence is cancelled while another submission arrives?

Evaluate the proposed mechanisms against the governing rule that a DBOS workflow ID does not itself serialize a business entity.

Consider appropriate durable enforcement mechanisms such as:

- PostgreSQL uniqueness constraints.
- Transactional entity locks.
- Advisory locks.
- Explicit per-entity sequence records.
- A partitioned work-queue design.

Do not prescribe a mechanism before establishing the required behavior. Recommend the smallest design that proves the business invariant.

Pay particular attention to tenant identity. Determine whether `lead_id` alone is globally unique or whether entity identity must include tenant scope.

## 6. Review approvals, cancellation, and human intervention

The ADR assigns one point to human involvement and proposes durable approval-pending recovery.

Determine whether approval is actually a business requirement or merely an optional implementation feature.

Review:

- Approval eligibility and required roles.
- Separation of duties.
- Self-approval prevention.
- Tenant boundaries.
- Approval expiry.
- Rejection semantics.
- Duplicate decisions.
- Decision attribution and audit evidence.
- Recovery after a decision is recorded but before workflow resumption.
- Cancellation while waiting for approval.
- Races between approval, expiry, cancellation, and execution.

Verify that these requirements are consistent with the existing Phase 1 approval and audit contracts.

Identify which semantics must be settled in the ADR and which may safely be implemented through existing framework capabilities.

Do not invent approval roles, thresholds, or expiry periods.

## 7. Review DBOS boundaries and version-safe recovery

Inspect the actual DBOS v1.9.0 integration rather than relying on general DBOS descriptions.

Evaluate:

- Workflow family `AppName` isolation.
- Application and workflow version registration.
- The relationship between `WorkflowDef.Version` and DBOS application versioning.
- Behavior when a workflow from an older version is still pending.
- Compatibility and replay after deployment.
- Unknown-version rejection.
- Deterministic workflow behavior.
- Durable wait and signal behavior.
- Cancellation and recovery semantics.
- Tenant-context propagation.
- Transaction boundaries.
- Shared-pool behavior and worker shutdown ordering.

Confirm whether the proposed workflow design can use the executor substrate without bypassing canonical invocation, authorization, audit, or effect controls.

Identify any assumptions that are not demonstrated by existing PostgreSQL evidence.

Do not claim that a test proves more than it actually exercised.

## 8. Evaluate the proposed §7 acceptance tests

The ADR lists these planned checks:

1. Determinism lint.
2. Idempotency keys.
3. Transaction boundary for touch and checkpoint.
4. Stable workflow ID.
5. Versioning guard.
6. Payload discipline.
7. Crash recovery.
8. Operational cost.

Review whether these tests are sufficient and correctly scoped.

For every test, state:

- The invariant it must establish.
- The failure mode it must reproduce.
- The specific regression assertion.
- Whether it can run locally, against real PostgreSQL, or requires a controlled process-restart test.
- Whether its evidence is necessary before approval or before enabling execution.

Identify missing adversarial tests, especially for:

- Concurrent submissions for the same lead.
- Different tenants using identical lead IDs.
- Replayed or duplicated approval decisions.
- Uncertain external-effect outcomes.
- Cancellation races.
- Version transitions with old runs pending.
- Database transaction failure during a workflow step.
- Process termination at every critical effect boundary.

Do not write or enable `lead-followup` code as part of this review. Tests may be specified as acceptance requirements only.

## 9. Check consistency with the framework's scope

Verify the proposal respects the accepted Phase 1 architecture:

- One canonical invocation service.
- Static miniapp registration.
- Authorization before dispatch.
- Server-derived principal and tenant context.
- Idempotency and attributable audit.
- Stable run handles for Class D.
- Shared effect and approval contracts.
- Per-family DBOS identity/version policy.
- Real PostgreSQL validation.
- Mock channels only.
- No real provider writes.

Confirm the proposal does not silently introduce:

- A second workflow or invocation engine.
- Miniapp-owned datasource provisioning.
- An alternative authorization path.
- A separate effects ledger.
- Client-supplied tenant or authority identity.
- Unapproved production effects.
- Infrastructure outside the approved Phase 1 scope.

If a design requirement cannot be satisfied by the existing framework, identify the architectural gap instead of adding an unapproved workaround.

## 10. Review the decision itself

After completing the analysis, independently assess the proposal:

**PROPOSED ADOPT**

Is DBOS justified for this workload?

**PROPOSED DEFER**

Are important business requirements still unknown, making a defensible decision premature?

**PROPOSED REJECT**

Does the workload not justify DBOS, or does the proposed design introduce complexity or guarantees that cannot be justified?

**CONDITIONAL**

Is adoption defensible only after specific conditions are satisfied?

Give a clear explanation. Do not preserve the current 12/14 score merely because it is already in the document. Recalculate the score using the governing rubric and provide evidence and rationale for every point.

Separate:

- Business requirements established by an authoritative source.
- Existing framework capabilities proven by code and tests.
- Technical requirements inferred from those business requirements.
- Proposed decisions awaiting human approval.

## 11. Required output

Create a review report at:

`docs/adr/miniapps/lead-followup-adr001-review.md`

The report must contain:

### A. Executive verdict

A clear recommendation and concise explanation for the business owner.

### B. Findings

Numbered findings classified as:

- BLOCKER
- HIGH
- MEDIUM
- LOW

For each finding include:

- Finding ID.
- Severity.
- Exact ADR section or source location.
- What is wrong, missing, or unproven.
- Why it matters.
- Evidence from the repository.
- Required resolution.
- Whether it requires a human business decision or an engineering change.

Do not manufacture findings just to fill categories.

### C. Four-gate assessment

A gate-by-gate decision with the governing threshold, score, evidence, and unresolved conditions.

### D. Revised score

A transparent 14-point breakdown. Explain any score change and do not award points for unconfirmed semantics.

### E. Business-owner decision register

A table of every unresolved business decision, the available choices where the sources support them, the consequence of each choice, and the exact owner input required.

Do not invent default choices.

### F. Technical design recommendations

Describe the minimum viable design needed to meet the confirmed requirements. Explicitly separate required framework behavior from miniapp-specific behavior.

### G. Acceptance-test matrix

Map each required invariant to a proposed adversarial test, expected result, and evidence required before workflow enablement.

### H. Proposed ADR corrections

Provide exact replacement wording or a clearly identified patch proposal for inaccurate, ambiguous, or unsupported claims.

Do not silently change the original ADR during the review.

### I. Approval recommendation

State whether the ADR is ready for human review, needs further information, or should be rejected.

Leave the human reviewer and approval date blank. Do not sign the ADR.

## 12. Review discipline and restrictions

- Treat `plan4.2.md` as authoritative and immutable.
- Do not modify framework code, tests, manifests, workflow registrations, or execution configuration.
- Do not implement `lead-followup`.
- Do not register or execute a `lead-followup` DBOS workflow.
- Do not create a manifest for `lead-followup`.
- Do not enable any real external writes or provider integrations.
- Do not commit or push changes.
- Do not weaken, delete, or rewrite existing tests.
- Do not convert unapproved business assumptions into requirements.
- Do not treat the ADR author's proposed score as an independently validated result.
- Do not claim that mock-channel tests prove production-provider delivery guarantees.
- Do not treat workflow identity as entity serialization.
- Do not confuse a successful test of the DBOS substrate with proof that the proposed business workflow is correctly designed.

When evidence is absent, say **NOT PROVEN**. When semantics are unknown, say **OWNER DECISION REQUIRED**. When a proposed guarantee cannot be achieved as written, explain the failure mechanism and recommend precise, achievable language.

For every significant technical conclusion, cite the exact source file and relevant code or test location. Bind test claims to the source revision and environment actually tested.

The existing work is uncommitted. Preserve that state and report your change inventory.

## Final instruction

This is an ADR review, not implementation authorization.

The deliverable must help the human owner make an informed decision about the business playbook and whether DBOS is justified.

**Stop after producing the review report. Increment 6 remains blocked until the human owner has reviewed the findings, resolved the business decisions, and explicitly signed the ADR.**

**My recommendation:** pay particular attention to three possible blockers in the review: the absence of an authoritative playbook, the unjustified exactly-once external-send claim, and the ambiguity between per-invocation workflow identity and per-lead serialization. Those are substantive design questions, not documentation polish.