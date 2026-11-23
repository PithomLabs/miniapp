# ADR-001 decision — lead-followup v1.0.0

Status: **DRAFT — NOT APPROVED.** This record proposes the workload
decision; it confers no authority to implement, register, or execute
`lead-followup` as a DBOS workflow. Human review (business
consequence + net complexity) is required first. All business
semantics below are `[PROPOSED]` per OSI-1: no approved source
specifies the follow-up sequence, and nothing here invents an owner
decision.

Template: `docs/adr/miniapps/adr001-template.md`.
Governing descriptors (plan4.2): "Business-outcome workflow: durable
waits, eligibility re-checks, events, cancel-safe recovery."

- Business promise: every sales lead receives its follow-up touches
  exactly per the confirmed playbook — no lost, duplicated, or
  out-of-window touches — with every touch attributable to the
  invoking agent and authority. `[PROPOSED: the playbook itself —
  number/timing of touches, channels, quiet hours, exit and
  escalation conditions — requires owner confirmation.]`
- Score (0/1/2): lifetime 2, stages 2, waiting 2, recovery 2,
  effects 2, coordination 1, human 1 = 12/14.
  - lifetime 2: multi-day lead lifecycle, far beyond one request.
  - stages 2: eligibility check → touch 1 → wait → eligibility
    re-check → touch 2 → … → exit/escalate. `[PROPOSED stage list.]`
  - waiting 2: durable waits between touches must survive restarts.
  - recovery 2: cancel-safe recovery is an explicit requirement;
    interrupted runs resume from the last completed touch.
  - effects 2: external sends (mock channels in Phase 1) need
    exactly-once semantics per touch.
  - coordination 1: per-lead serialization so concurrent triggers
    for one lead collapse into one sequence. `[PROPOSED mechanism.]`
  - human 1: approval-pending recovery — a touch gated on approval
    parks durably and resumes on decision or expiry.
- Gate 1 durable need: multi-day waits with exactly-once touch
  semantics across process restarts and deployments.
- Gate 2 characteristics (need ≥2): waiting, recovery, effects,
  coordination — four present.
- Gate 3 failure consequence: a lost follow-up is silently lost
  revenue attribution; a duplicated touch is customer-facing spam;
  an out-of-window touch violates the playbook. All three are
  business-visible and hard to remediate after the fact.
- Alternative A (without DBOS) custom machinery: lead-state table +
  stage machine, timer/delay queue, reaper for stuck sequences,
  per-lead locking, idempotency ledger per touch, manual recovery
  runbook for mid-sequence crashes.
- Alternative B (DBOS) covers: durable waits, crash recovery from
  last completed step, events/signals, cancellation, workflow
  identity, versioned recovery. Still needs us to build: tool
  manifest + workflow functions, mock channel connectors,
  eligibility re-check logic, approval parking integration, audit
  attribution, per-lead serialization wiring.
- §7 checks (each links to a test before enablement; tests are
  PLANNED, not written):
  - determinism lint: analyzer clean on the workflow package
    (planned: `TestLeadFollowupDeterminismLint`).
  - idempotency keys: duplicate submission returns the live handle
    without a second sequence (planned:
    `TestLeadFollowupDuplicateSubmit`).
  - tx boundary: touch record + checkpoint commit atomically
    (planned: `TestLeadFollowupAtomicTouch`).
  - workflow ID: stable run handle derived per invocation
    (planned: `TestLeadFollowupStableHandle`).
  - versioning plan: `WorkflowDef.Version "1"`; recovery guard
    rejects unknown versions (planned:
    `TestLeadFollowupVersionGuard`).
  - payload discipline: inputs/outputs JSON-serializable, bounded
    size, no credentials (planned:
    `TestLeadFollowupPayloadDiscipline`).
  - recovery policy: kill mid-wait, relaunch, sequence resumes
    without repeating the completed touch (planned:
    `TestLeadFollowupCrashRecovery`).
  - op cost: one workflow family, shared-pool durability, no new
    infrastructure beyond the Phase 1 stack.
- Entity identity: the sales lead (`lead_id`). `[PROPOSED:
  per-lead serialization via a deterministic workflow ID derived
  from the lead, or a partitioned DBOS queue keyed by lead —
  mechanism requires owner confirmation alongside the playbook.]`
- External-effect boundaries: channel sends happen only inside
  transaction steps recording the touch first; mock connectors in
  Phase 1, no production channels.
- Custom machinery estimate (Alternative A): a bespoke durable
  state machine (~2–3 weeks) plus permanent ownership of timers,
  reaping, and recovery runbooks.
- Residual work under Alternative B: workflow + connectors +
  eligibility + approval/audit wiring (~1 week), reusing the
  tested Phase 1 DBOS substrate.
- Decision: PROPOSED ADOPT (pending human review).
- Reviewer (human): <awaiting owner>   Date: <pending>

## Hard stop

Until the reviewer signs above: no `lead-followup` workflow code, no
registration, no DBOS execution, no manifest. The DBOS substrate
(`executor/dbos`), approvals, effects, and audit proceed
independently and are proven by substrate probes carrying no
business semantics.
