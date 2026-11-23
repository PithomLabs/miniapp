# Adversarial Review: `lead-followup-adr001.md`

## Preamble — What this review is and isn't

Per `prompt11.md`, this review's surface is the `lead-followup` ADR, but its subject is the **framework**, not the example. The example is deliberately simple — a multi-touch lead follow-up with waits, eligibility re-checks, mock sends, and an optional approval gate. That simplicity is intentional: a probe.

The question I am answering is not "is `lead-followup` a good ADR?" It is: **does a review of this ADR reveal whether the framework — as specified in `plan4.2.md` and evidenced in the reported Increments 4–5 — is genuinely capable of supporting the Class D workflows it claims to support, once those workflows become worthy of DBOS rather than merely plausible?**

That reframe changes the finding set. A finding that the ADR overstates a guarantee is a **framework capability finding**, because a framework that lets its ADR process overstate guarantees cannot reliably govern the harder workflows it will be asked to host. A finding that the ADR proposes a mechanism the framework rejects is a **framework enforcement finding**, because the framework needs a mechanism to catch that before a human reviewer has to. And a finding that the ADR leaves a specification gap is a **framework template finding**, because the template that produced this ADR should have required the specification.

I have not modified any file. I have not registered or executed anything. The hard stop in `prompt11.md` and in the ADR itself is in force.

---

## A. Executive verdict

**The `lead-followup` ADR cannot be approved as written.** Not because `lead-followup` is a poor example of a DBOS workload — it is actually a reasonable one — but because the ADR contradicts the framework's own contracts in three places, proposes a mechanism the framework explicitly rejects in one place, and leaves five specification gaps that a Class D workflow cannot leave open.

**The framework is architecturally capable of supporting DBOS-warranted workflows.** The primitives exist: DBOS substrate, effect protocol, approval table, `entity_key` contract, per-family versioning, determinism lint, crash-boundary test requirements. The gates exist: G0-Core requires DBOS/Neon atomicity and family-version isolation; G1 requires "no duplicate confirmed effects, version-safe recovery."

**The framework is process-weak.** Nothing in the ADR process prevents an author from:
1. claiming "exactly-once semantics" when the framework's R2 explicitly disclaims that language,
2. proposing "workflow ID as entity serializer" when the framework's §15.2 explicitly rejects that mechanism,
3. leaving business semantics entirely `[PROPOSED]` while assigning a 12/14 score.

Those three failures are visible in this ADR — a simple, well-intentioned probe — and they will be worse in the harder workflows the framework is meant to host.

**Recommendation:** CONDITIONAL. Adopt DBOS for `lead-followup` only after the business playbook is confirmed by the owner, the three blockers below are corrected, and the five HIGH findings are specified. Add an ADR consistency gate to the framework before any second Class D ADR reaches review.

---

## B. Findings

### BLOCKERS

**B1 — "Exactly-once semantics per touch" contradicts the framework's own R2 contract**

- **Location:** ADR §3, "effects 2: external sends … need exactly-once semantics per touch"; §7, "tx boundary: touch record + checkpoint commit atomically."
- **Evidence from framework:** `plan4.2.md` R2 explicitly says: *"No duplicate confirmed business effects, no silently lost operations, no blind retries of unknown outcomes"* is the acceptance language, replacing *"zero duplicate external effects"* because *"a step can execute twice across the provider-complete/checkpoint-missing crash window."* §9.5 reiterates the same discipline. §14.3 and §15 make the effect protocol `PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN` mandatory.
- **Why it matters:** The ADR's "exactly-once" claim is the kind of guarantee a business owner will hear and a reviewer will accept because the framework has "durable execution." Neither hears the honest guarantee: at-least-once attempts, at-most-once confirmed effects when provider idempotency exists, reconcile-by-read when it does not, UNKNOWN quarantine otherwise. For a mock-channel Phase 1 the distinction is academic. For the first real provider send, it is the difference between a defensible incident response and a false promise.
- **Resolution required:** Engineering change — replace "exactly-once" with the R2 language. The framework should refuse to register a Class D tool whose manifest or ADR contains the phrase "exactly-once" applied to external effects without an explicit reconciliation method named. **This is a framework validator rule, not just an ADR correction.**
- **Framework capability signal:** The framework is capable; it lacks enforcement.

**B2 — Workflow ID as entity serializer is a mechanism the framework rejects**

- **Location:** ADR §3 "coordination 1," proposing per-lead serialization via "a deterministic workflow ID derived from the lead, or a partitioned DBOS queue keyed by lead."
- **Evidence from framework:** `plan4.2.md` §15.2 states plainly: *"Workflow IDs alone do not serialize access to an entity."* The supported mechanisms are: row lock or transaction-scoped advisory lock (Class A), stable unique job/effect identity plus entity lock (Class B/C), or DBOS partitioned queue with per-partition concurrency one (Class D).
- **Why it matters:** The ADR is choosing between an endorsed mechanism and an explicitly rejected one as though they were equivalents. This is precisely the failure mode the framework's own §15.2 was written to prevent. A reviewer who trusts the ADR's "or" will approve a design that does not enforce the invariant it claims.
- **Resolution required:** Engineering change — remove the workflow-ID option. The ADR must specify the partitioned-queue mechanism (concurrency 1, keyed by the tenant-scoped entity key) or a transaction-scoped advisory lock held across the workflow's critical sections, and must test it. This is not a stylistic choice.
- **Framework capability signal:** The framework is capable; the ADR process allows rejected mechanisms to appear as equals.

**B3 — Entity identity is not tenant-scoped**

- **Location:** ADR §3 "Entity identity: the sales lead (`lead_id`)" with a note that the mechanism "requires owner confirmation."
- **Evidence from framework:** `plan4.2.md` §9.1 and §15.2 require each consequential write tool to declare `entity_key` with serialization scope. The framework's tenant isolation model (§14, R5) requires transaction-local tenant context. A `lead_id` alone is not globally unique across tenants — two tenants can legitimately have the same lead ID.
- **Why it matters:** A partitioned queue keyed by raw `lead_id` would serialize two different tenants' leads against each other, and a database uniqueness constraint on raw `lead_id` would reject a second tenant's legitimate lead. This is a data-corruption bug in the multi-tenant case that a single-tenant Phase 1 would not surface.
- **Resolution required:** Engineering change — the ADR must specify `entity_key = tenant_id:lead_id` (or an equivalent scoping). The framework's manifest validator must reject unscoped entity keys for any tool whose substrate is Class D.
- **Framework capability signal:** The framework is capable; the ADR process permits unscoped keys.

**B4 — Business playbook is entirely `[PROPOSED]`; the 12/14 score is pre-justified**

- **Location:** ADR preamble and §1 — every business semantic (number of touches, timing, channels, quiet hours, eligibility, exit conditions, escalation, approval role, cancellation, failure behavior) is marked `[PROPOSED]`. Reviewer and date are blank.
- **Why it matters:** The ADR is honest that the business case is unapproved, then assigns 12/14 — a score that presumes the business case is what was proposed. This is circular. The framework's ADR-001 gate rubric (`docs/DBOS.md` Gates 1–4) is written for a workload whose business consequences are established. It is not a rubric for scoring proposed-but-unconfirmed semantics.
- **Resolution required:** Human business decision — the ADR must carry **no score** until the playbook is confirmed. The framework should refuse to compute or record a Class C/D score for an ADR whose business semantics are `[PROPOSED]`. Once the playbook resolves, the score can be recomputed and reviewed.
- **Framework capability signal:** The framework's ADR-001 adoption rubric is capable; the ADR process allows the rubric to be applied to hypotheticals.

### HIGH

**H1 — "Resume from the last completed touch" conflates business touch with DBOS step**

- **Location:** ADR §3 "recovery 2."
- **Why it matters:** A "touch" is a business event (a send). A DBOS step is an atomic unit of durable execution. Resuming "from the last completed touch" is well-defined only if the workflow's step boundaries align with touch boundaries. If a touch involves multiple steps (e.g., `record_intent`, `call_provider`, `confirm`, `emit_audit`), then resuming "from the last completed touch" is a request that the step structure be touch-aligned, and that mid-touch crashes be handled at the touch boundary, not the step boundary.
- **Resolution required:** Engineering change — the ADR must state, per touch, which steps are pre-send (idempotent, safe to replay), which are send-attempting (subject to UNKNOWN handling), and which are post-send (idempotent, gated on effect state). Without this, "resume" is undefined.

**H2 — Cancel-during-in-flight-touch is unspecified**

- **Location:** ADR §3 "recovery 2 (cancel-safe)" and §4 references to cancellation.
- **Why it matters:** A cancellation that arrives after a touch is `ATTEMPTING` but before it is `CONFIRMED | FAILED | UNKNOWN` is a three-way race. The framework's §15 mandates UNKNOWN quarantine, but the ADR does not say what the workflow does: does it drain, does it stop attempting new touches, does it attempt to recall an in-flight send? For Phase 1 mock channels, this is testable; for real channels, it becomes the difference between "stop the sequence" and "stop the sequence but the last touch is possibly lost."
- **Resolution required:** Engineering change — specify cancellation semantics: what happens to the current touch, what the workflow does while the effect is `ATTEMPTING`, what evidence permits resumption after cancellation, and what constitutes a "safe cancel."

**H3 — Approval-to-DBOS-event mapping is unspecified**

- **Location:** ADR §3 "human 1" and §6–7 references to approval-pending recovery.
- **Why it matters:** `plan4.2.md` §15.1 says approval is a policy gate, not a class; for Class D the approval table is the source of truth and the workflow waits on a DBOS message/event with timeout. The ADR describes "a touch gated on approval parks durably and resumes on decision or expiry" but does not specify: which approvals gate which touches; which approval channel is used; what happens on approval-after-expiry; how material-hash invalidation interacts with a mid-workflow approval; whether the approver can be the same principal as the requester.
- **Resolution required:** Engineering change — the ADR must define the approval-to-DBOS mapping per touch, including the approval-table state transitions and the DBOS wait condition, and reference the framework's out-of-band approval channel (§14.4).

**H4 — Version-transition behavior for pending workflows is not addressed**

- **Location:** ADR §7 "versioning plan: `WorkflowDef.Version "1"`; recovery guard rejects unknown versions."
- **Why it matters:** Rejecting unknown versions is the correct guard, but it does not answer the operational question: when a `lead-followup` v1 workflow is waiting on a multi-day durable wait and v2 ships, what happens to the v1 workflow? `plan4.2.md` ADR-016 prescribes per-family `AppName` and explicit application version, with old artifacts retained until drain. The ADR should either cite that contract or state its own drain strategy.
- **Resolution required:** Engineering change — reference the framework's ADR-016 family-versioning policy, or specify an equivalent drain strategy for this workflow family.

**H5 — UNKNOWN handling for mock sends is not specified**

- **Location:** ADR §4 "mock connectors in Phase 1, no production channels."
- **Why it matters:** The framework's effect protocol (`PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN`) is mandatory. Even with mock channels, the ADR should specify how a mock send can produce `UNKNOWN` (deliberately injected, in test), how `UNKNOWN` quarantines, and what evidence permits reconciliation. Otherwise the Phase 1 test suite will not exercise the state that will dominate Phase 2 and beyond.
- **Resolution required:** Engineering change — specify how mock-channel tests produce and resolve `UNKNOWN`, and require that a "poisoned" mock (succeeds externally, fails to acknowledge) be part of the Phase 1 test suite.

### MEDIUM

**M1 — The 12/14 score is unearned; the fair score is 0 or "not yet scoreable."**

If scored honestly under the rubric — where points are earned by *confirmed* business requirements, not proposed ones — the score cannot be finalized until B4 resolves. The aspirational score, *if* all proposed semantics are confirmed and B2 is corrected (partitioned queue), is 11/14 (coordination drops from 1 to 0 until mechanism is fixed; other points provisional). The ADR should state this openly.

**M2 — "~2-3 weeks vs ~1 week" estimates are unfalsifiable and omit long-term costs**

- **Location:** ADR §3 "Custom machinery estimate."
- **Why it matters:** The comparison does not account for: DBOS operational cost (upgrades, version transitions, `AppName` registration, family deployment boundaries), the determinism lint's ongoing enforcement cost, the effect reconciler's operator burden, the approval-table audit, or the version-transition drain procedure. These are real and recurring. The framework's ADR-001 Gate 4 requires comparing total complexity, not initial implementation effort.
- **Resolution required:** Engineering change — add the lifecycle-cost lines the framework's own Gate 4 requires. This is not a critique of the example; it is a test of whether the ADR template captures the framework's intended discipline.

**M3 — "Eligibility re-check" is undefined**

- **Location:** ADR §1 and §3 "stages 2."
- **Why it matters:** An eligibility re-check before a touch could be: a Class A read via canonical invocation, a capability call, a DBOS step wrapping either, or a local predicate over workflow state. The choice determines whether the check is durable, whether it appears in the audit trail as a step, and whether it can be retried independently. None of these are specified.
- **Resolution required:** Engineering change — declare each re-check as a named DBOS step with a named capability, and specify the failure path when eligibility cannot be determined.

**M4 — Approval vs execution timing separation is unclear**

- **Location:** ADR §3 "human 1" and §6–7.
- **Why it matters:** An approval that binds to material input (per `plan4.2.md` §15.1) must be re-checked immediately before execution. The ADR does not say what the workflow does between approval and execution if the lead's state changes — and eligibility re-checks are exactly the mechanism the workflow already has. The ADR should state that approval is a valid precondition, not a substitute for the last re-check.
- **Resolution required:** Engineering change — specify the ordering: approval → last eligibility re-check → execution, and what happens if the re-check fails after approval.

### LOW

**L1 — Missing references to the framework's §15.2 entity-serialization rules.** The ADR mentions coordination but should cite the framework section that governs it, so future readers know the invariant is framework-enforced (once the framework enforces it).

**L2 — Missing references to R2's corrected acceptance language.** The ADR's effect semantics should quote the framework's language directly, not paraphrase it.

**L3 — Missing reference to ADR-016 family-versioning.** If H4 is resolved by citing ADR-016, this finding becomes moot. If not, the ADR's versioning story stands alone and needs its own defense.

**L4 — `§7 checks` label "planned" is honest but incomplete.** The ADR does not map each planned test to a framework-enforced crash-boundary test category (§18 of `plan4.2.md`). Mapping would let the framework's test harness drive the ADR's evidence, not require bespoke tests.

---

## C. Four-gate assessment

| Gate | Governing threshold | Proposed by ADR | My assessment |
|---|---|---|---|
| **Gate 1 — Durable need** | Multi-day waits, resumption from checkpoint, or durable event waits are genuine requirements | Multi-day waits with exactly-once touch semantics | **Provisionally met.** The multi-day wait and recovery requirements are real *if* the playbook confirms them. The "exactly-once" framing (B1) weakens the claim; the honest requirement — "no duplicate confirmed sends, no silently lost sends, no blind retries of unknown sends" — is still a genuine durable need. |
| **Gate 2 — Workflow characteristics** | At least two of: multiple independent stages, durable waits, external events, human approval, multiple dependent services, retry between stages, fan-out, compensation, cancellation, execution history | waiting, recovery, effects, coordination | **Partially met.** Waiting and recovery are established. Effects are established subject to B1/H5. Coordination (B2, B3) is broken as proposed. The count should be **2 (waiting, recovery) + 1 provisional (effects) + 0 (coordination until fixed) = 3**, not 4. Still meets the ≥2 threshold; the ADR overstates the margin. |
| **Gate 3 — Business consequence** | Failure would break a customer-facing promise, cause duplicate effects, or create expensive reconciliation | Lost follow-up = lost revenue; duplicate touch = spam; out-of-window = playbook violation | **OWNER DECISION REQUIRED.** All three consequences are plausible but none is confirmed by an authoritative source. Under the framework's own ADR-001 policy, an unconfirmed business consequence cannot satisfy Gate 3. |
| **Gate 4 — Net complexity** | DBOS reduces total complexity vs. the bespoke alternative, accounting for full lifecycle | ~2–3 weeks (bespoke) vs ~1 week (DBOS) | **NOT PROVEN.** Estimates are initial-effort only and omit lifecycle costs (M2). Even if correct, the comparison does not satisfy the framework's own gate as written. |

**Gate assessment:** The ADR passes Gates 1 and 2 provisionally; Gate 3 requires owner input; Gate 4 is unproven. This is not a "yes" — it is a "yes if the business case confirms and the ADR is corrected to match framework contracts."

---

## D. Revised score

| Dimension | ADR | My score | Rationale |
|---|---|---|---|
| lifetime | 2 | 2 (provisional) | Multi-day lead lifecycle is plausible; confirmed only if playbook confirms | |
| stages | 2 | 2 | Eligibility → touch → wait → re-check → touch pattern is real | |
| waiting | 2 | 2 | Durable waits across restarts are genuine | |
| recovery | 2 | 2 (provisional) | "Cancel-safe recovery" requires H1/H2 specification | |
| effects | 2 | 2 (provisional) | Requires B1/H5 correction — the guarantee is at-least-once + reconciliation, not exactly-once | |
| coordination | 1 | 0 (until fixed) | B2 is a blocker; the proposed mechanisms include one the framework rejects | |
| human | 1 | 1 (provisional) | Requires H3 specification | |
| **Total** | **12/14** | **11/14 provisional, 0/14 scoreable** | The ADR is not scoreable until B4 resolves. If playbook confirms the proposed semantics and B2/B3/H1/H2/H3/H4/H5 are corrected, 11/14 is defensible. |

The framework's rubric asks the ADR to earn points. The ADR's 12/14 cannot survive contact with the framework's own contracts. That is the finding.

---

## E. Business-owner decision register

All of the following are **OWNER DECISION REQUIRED** and block B4.

| # | Decision | Options the sources support | Consequence of each | Owner input needed |
|---|---|---|---|---|
| 1 | Number of touches | 1–N (N unspecified in any source) | Determines workflow duration and stage count | Exact N, or "campaign-bounded by segment" |
| 2 | Touch timing | Not specified | Determines wait durations; affects test harness clocking | Intervals and deadlines |
| 3 | Channels | Mock only in Phase 1; production channel TBD | Determines whether the effect protocol's UNKNOWN handling is exercised in Phase 1 | Channel list and order-of-attempts |
| 4 | Quiet hours | Timezone and holidays unspecified | Determines durable wait boundaries and whether "wait" spans business-day boundaries | Timezone; days; holiday calendar source |
| 5 | Eligibility at each stage | Not specified | Determines number of DBOS steps and capability calls; determines failure path when eligibility is unreadable | Predicate per stage |
| 6 | Exit conditions | Not specified | Determines cancellation triggers and completion semantics | Enumerated exit causes |
| 7 | Escalation | Not specified | Determines additional workflow branches and approver roles | Escalation conditions and target roles |
| 8 | Approval applicability | Which touches require approval, if any, unspecified | Determines H3 scope: if approval gates all touches, the workflow is approval-heavy; if none, the human dimension should score 0 | Approval policy |
| 9 | Cancellation semantics | Not specified | Determines H2 scope and what "cancel-safe" means | Cancel authority and in-flight-touch behavior |
| 10 | Failure behavior | Channel unavailable → unknown; send fails → unknown; outcome unknown → unknown (per framework). Owner must confirm what the business will tolerate | Determines whether "no duplicate confirmed sends" is acceptable or whether "at most one attempt ever" is required | Failure tolerance per touch |
| 11 | Consent and suppression | Not specified | Determines whether a per-lead suppression check is a required step | Consent source and suppression list |
| 12 | Duplicate-contact limits | Not specified | Determines whether the coordination dimension is per-lead only or also per-lead-per-channel-per-window | Constraints |
| 13 | Tenant scoping | Framework requires tenant-scoped entity keys; owner must confirm lead IDs are tenant-local or globally unique | Determines B3 resolution | Confirmation or override with justification |

The framework's ADR-001 template is capable of holding this register. The ADR as drafted does not populate it — it marks everything `[PROPOSED]` and assigns a score. That is the gap.

---

## F. Technical design recommendations

**Minimum viable Class D design for `lead-followup`, contingent on owner decisions above:**

1. **Entity identity:** `entity_key = {tenant_id}:{lead_id}`, declared in the manifest, enforced by the framework's manifest validator (B3).

2. **Coordination:** Partitioned DBOS queue, concurrency 1, partition key = `entity_key`. Advisory-lock fallback only if partitioned-queue fairness is insufficient (per §15.2). Workflow ID is the invocation identity, *not* the entity-serialization mechanism (B2).

3. **Effect protocol per touch:** Each touch is a multi-step block:
   - `step:check_eligibility` (reads via capability, idempotent)
   - `step:record_intent` (durably writes `PENDING` + idempotency identity)
   - `step:attempt_send` (writes `ATTEMPTING`, calls capability proxy, returns provider receipt or ambiguity)
   - `step:confirm` (writes `CONFIRMED` on receipt, or `UNKNOWN` on ambiguity)
   - `step:audit` (idempotent)
   
   The step boundaries are touch-aligned so that "resume from last completed touch" is well-defined (H1). Mock channels must be able to produce `UNKNOWN` deliberately (H5).

4. **Approval:** Approval is a policy overlay, not a step class. Class D workflow waits on a DBOS message with a timeout; the approval table is the source of truth. The wait is per-touch when owner confirms approval policy; material-hash invalidation and last-eligibility-recheck-before-execution are mandatory (H3, M4).

5. **Cancellation:** Workflow receives cancel signal; behavior is:
   - If no touch `ATTEMPTING` → cancel cleanly, record `CANCELLED` with last completed touch.
   - If a touch is `ATTEMPTING` → wait for its outcome (success, failure, or UNKNOWN) within a bounded time, then cancel; `UNKNOWN` outcomes are quarantined and surfaced for reconciliation, not retried (H2).
   - New-touch scheduling halts immediately upon cancel receipt.

6. **Versioning:** Per-family `AppName` = `lead-followup`; explicit `ApplicationVersion`; deployment retains old version's worker until pending workflows drain. Reference ADR-016 (H4).

7. **Touch-level audit:** Every touch writes an audit event with `(tenant_id, lead_id, touch_index, effect_id, provider_receipt_or_UNKNOWN, timestamp)` so an operator can reconstruct the sequence's history and reconcile UNKNOWNs.

8. **Framework vs. miniapp boundary:** All of the above are miniapp-specific configurations of framework primitives. No new framework mechanism is required. **The framework's capability is confirmed; the ADR's specification is not yet aligned.**

---

## G. Acceptance-test matrix

| Invariant | Test | Expected result | Evidence required |
|---|---|---|---|
| No duplicate confirmed sends | Kill process after provider accepted but before `CONFIRMED`; restart; replay | `UNKNOWN` quarantined; operator reconciliation path exercised; no second send attempted without explicit evidence | Pre-enablement |
| No silently lost sends | Kill process after `PENDING` before `ATTEMPTING`; restart | Resume attempt from `PENDING` | Pre-enablement |
| `entity_key` serializes concurrent submissions | Two concurrent invocations for same `(tenant_id, lead_id)` | One workflow; second submission returns the live handle | Pre-enablement |
| Tenant isolation for entity key | Same `lead_id` in two tenants | Two workflows, run concurrently | Pre-enablement |
| Approval material-hash invalidation | Approve; mutate input; attempt execution | Approval invalidated; workflow pauses for re-approval | Pre-enablement |
| Approval out-of-band | Attempt self-approval from the invoking agent | Rejected | Pre-enablement |
| Cancel-during-in-flight-touch | Cancel signal while `ATTEMPTING` | Bounded wait; `UNKNOWN` quarantined; cancellation recorded with last completed touch | Pre-enablement |
| Version transition with pending workflow | Deploy v2 while v1 waiting | v1 resumes on v1 worker (drain) or is drained; no silent replay against v2 | Pre-enablement |
| Determinism lint | CI on Class D package | Pass | Pre-enablement |
| UNKNOWN quarantine in mock channels | Poisoned mock: success without ack | Effect enters `UNKNOWN`; reconciler flags; no blind retry | Pre-enablement |
| Resume from last completed touch | Kill mid-touch at each step; restart | Resume at touch boundary; no replay of completed touch steps | Pre-enablement |
| PII discipline | Inspect workflow checkpoints after a run | No provider text or PII in checkpoint payloads | Pre-enablement |
| RLS tenant isolation under pooled connections | Concurrent writes across tenants on the shared pool | Transaction-local context enforces correct tenant scoping | Pre-enablement |
| Provider receipt absent + reconciliation method defined | Mock returns no receipt; reconciliation method is exercised | Effect remains `UNKNOWN` until reconciliation; outcome determines retry vs. terminal | Pre-enablement |

These tests must run on real Postgres (per framework §18) and — for the crash-boundary tests — on the Fly deployment shape (per §19 Phase 1 exit gate G1). Mock-channel tests alone do not satisfy B1's "at-least-once + reconciliation" discipline; they satisfy the Phase 1 Phase-1-substrate requirement and set up the pattern for Phase 2.

---

## H. Proposed ADR corrections

The following are proposed replacement texts. I have not edited the ADR; these are patch proposals for the ADR's author.

**Patch to B1 — replace the "exactly-once" language throughout:**

> *Current:* "external sends (mock channels in Phase 1) need exactly-once semantics per touch."
>
> *Proposed:* "External sends follow the framework's §9.5 effect protocol: `PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN`. The accepted guarantee is: no duplicate confirmed business effects, no silently lost operations, no blind retries of unknown outcomes. Exactly-once delivery is not claimed; where provider idempotency is unavailable, reconcile-by-read is mandatory before any retry, and `UNKNOWN` outcomes are quarantined for operator decision."

**Patch to B2 — replace the coordination mechanism:**

> *Current:* "per-lead serialization via a deterministic workflow ID derived from the lead, or a partitioned DBOS queue keyed by lead."
>
> *Proposed:* "per-lead serialization via a partitioned DBOS queue, partition key = `entity_key = {tenant_id}:{lead_id}`, per-partition concurrency 1. Workflow ID is the invocation identity (deduplication of submitted requests), not an entity-serialization mechanism."

**Patch to B3 — replace entity identity:**

> *Current:* "Entity identity: the sales lead (`lead_id`)."
>
> *Proposed:* "Entity identity: `entity_key = {tenant_id}:{lead_id}`. Lead IDs are tenant-scoped; the framework requires tenant-scoped entity keys for all Class D write tools."

**Patch to B4 — remove pre-justified score:**

> *Proposed:* Replace "Score (0/1/2): 12/14" with "Score: **not yet scoreable** until the owner-decision register (§E) is populated and confirmed. Provisional aspirational score *if* the proposed semantics are confirmed and B2 is corrected: 11/14."

**Patches to H1–H5:** add specific subsections to §3 covering step/touch alignment, cancellation-during-in-flight-touch semantics, approval-to-DBOS-event mapping, version-transition drain strategy, and UNKNOWN production for mock channels.

---

## I. Approval recommendation

**The ADR is not ready for approval.** It needs:

1. **Owner decisions on all 13 items in §E.** Until those are resolved, B4 applies and the ADR cannot be scored.
2. **Author corrections on B1, B2, B3.** These are not business decisions; they are corrections to match the framework's own contracts.
3. **Specification additions for H1–H5.** These are engineering decisions the ADR author makes, not framework changes.

**The framework itself is capable of supporting DBOS-warranted workflows.** The evidence in this review is that the framework's primitives are correct and sufficient. The `lead-followup` probe does not reveal a missing framework capability; it reveals a missing **framework enforcement mechanism at the ADR boundary** — specifically, the absence of a mechanical validator that refuses an ADR that (a) contradicts framework contracts, (b) proposes mechanisms the framework rejects, (c) leaves required specifications open, or (d) claims a score for unconfirmed semantics.

**For the framework:** this probe argues for adding a `adr-validator` to the Phase 0 contracts — a mechanical check that runs before human review and produces a findings list. The rules are visible in B1–B3 and H1–H5 above. Without it, every future Class D ADR review will repeat this exercise, and the framework's ability to govern DBOS-warranted workflows will be limited by the process rather than the substrate.

**For the ADR:** CONDITIONAL. Approval becomes defensible once:

- The owner confirms the playbook (unblocks B4).
- B1, B2, B3 are corrected in the ADR text.
- H1–H5 are specified in the ADR text.
- The acceptance-test matrix in §G is committed as the pre-enablement evidence set.
- A human reviewer signs the ADR with an evidence-register citation for each test.

Until then: **hard stop remains.** No `lead-followup` workflow code, manifest, or registration is created by this review or by anyone acting on it.

---

## Change inventory

- **Files created by this review:** none.
- **Files modified:** none.
- **Files deleted:** none.
- **Workflows registered or executed:** none.
- **Real provider integrations enabled:** none.
- **Framework code or tests modified:** none.
- **`lead-followup` code or manifest created:** none.
- **Increment 6 started:** none.
- **ADR status:** unchanged — still `DRAFT — NOT APPROVED`.
- **Reviewer field in ADR:** unchanged — still blank. This review does not sign the ADR.

---

**Human owner, the question in front of you is not "should I approve this ADR." It is "which of the 13 business decisions in §E do I resolve, in what order, and do I accept the framework as capable of hosting this workflow once those decisions are made." The framework's answer to the second is yes. The ADR's answer to the first is that the work is yours to do.**