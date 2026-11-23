# Review — `lead-followup` ADR-001 (Independent Adversarial Review), Revision 2

- **Intended repository path:** `docs/adr/miniapps/lead-followup-adr001-review.md`
- **Subject:** `docs/adr/miniapps/lead-followup-adr001.md` — status DRAFT, proposed decision PROPOSED ADOPT, score 12/14
- **Review brief:** Prompt 11
- **Governing plan:** `docs/plan4.2.md` (read for this revision; see scope below)
- **Date:** 2026-11-22
- **Revision 2:** re-run against plan4.2.md. Revision 1 was written without it and said its gate rows would need re-running.
- **Finding IDs** use the prefix `LF-` so they do not collide with plan4.2 Appendix F (`F01–F24`, which dispositions an earlier review of plan4.1).
- **Reviewer stance:** independent; the ADR author's score and wording are treated as claims, not evidence.
- **Authority of this document:** none. It confers no approval. The increment-6 hard stop remains in force. The ADR's reviewer and approval-date fields are not signed here.

---

## Scope, limits and evidence conventions

| Material named in Prompt 11 | Available? | Consequence |
|---|---|---|
| `lead-followup-adr001.md` | Yes, full text | Primary subject |
| `prompt11.md` | Yes | Review brief |
| `docs/plan4.2.md` | **Yes (new in Rev 2)**; read: §§1–2, 5, 7–10, 11, 12, 14.2–14.4, 15, 16, 18, 19 (Phases 1 and 4), 23, App. B, App. F. **Not read in full:** §§3–4, 6, 13, 14.1, 14.5, 17, 19 (other phases), 20–22, App. A, C–E | Governing text for plan-level claims below |
| ADR-001 as `DBOS.md` (gates, rubric, §7 obligations, §9 template) | Yes | plan4.2 §10.1 adopts it "by principle" and requires it to be ported to Go/PostgreSQL/River in Phase 0 (G0-Core condition 9). Until that port exists, `DBOS.md` is the only text of the gates and the 14-point rubric |
| `adr001-template.md`, G0 adjudication | No | Not reviewed |
| `runtime/*`, `executor/dbos/*`, `approvals/`, `effects/`, `audit/`, tests, PostgreSQL spike evidence | **No** | Every statement about Phase 1 code or test coverage is **NOT INSPECTED / NOT PROVEN** by this review |

No test was run and no source revision was examined, so no test claim here is bound to a revision or environment. DBOS behavior claims come from public DBOS Go documentation checked 2026-11-22 (listed at the end); v1.9.0-specific behavior and this repository's integration were **not** verified.

**Conventions.** **NOT PROVEN**: evidence absent or not inspected. **OWNER DECISION REQUIRED**: a business semantic is unknown; no value or default is invented here. In tables, effect states are written with "or" instead of a pipe character.

**Change inventory.** This file replaces Revision 1 (a copy of Revision 1 was kept outside the deliverable folder). Earlier in this engagement I also edited my own `app_category_guidelines.md` in two places: a Class D sample no longer sets automatic retries on a send step, and the follow-up-sequence example is marked conditional on a confirmed playbook. No repository file was read or modified, nothing was committed, and no `lead-followup` code, manifest, registration or execution was created.

---

## Revision 2 — what plan4.2 changed

| # | Topic | Revision 1 | Revision 2 (after plan4.2) |
|---|---|---|---|
| 1 | What the quoted descriptor is | "Authoritative-as-quoted"; circular if used as a requirement | Verified in §16: it sits in a column headed "Purpose: what it proves", and `lead-followup` belongs to the "forcing set". It is a **framework forcing fixture**, not a business requirement. This strengthens LF-01 and creates a fork the owner must resolve (O-00) |
| 2 | "Exactly-once" | Unachievable as worded | Also **contradicts the governing plan**: ADR-006 ("honest at-least-once"), §9.5 and §15 define the effect states and the target "no duplicate confirmed effects, no silently lost operations, no blind retries" |
| 3 | Workflow ID as serialization | Not equivalent to entity serialization | §15.2 states it outright: "Workflow IDs alone do not serialize access to an entity." The ADR's first mechanism is exactly that |
| 4 | Gate 2 definition | "plan4.2 may differ" | plan4.2 does **not** restate the gates or the rubric. `DBOS.md` governs until ported; the ADR's cited basis is still the wrong list |
| 5 | River availability | Open question (NOT VERIFIED) | `executor/river` is Phase 3 (§19, G3); Phase 1 has only `executor/classa` and `executor/dbos`. Alternative A′ is therefore **out of Phase 1 scope**, so Gate 4 is partly a scope argument, not a workload argument |
| 6 | Framework controls | Listed as possible gaps (authority revalidation, kill switch, approvals, version drain) | plan4.2 already **requires** these (§9.3, §9.1, §14.2, §14.4, §15.1, §11.2, §23.1). Findings LF-06, LF-07, LF-08, LF-12 are downgraded from HIGH to MEDIUM: the ADR must bind to them; implementation is NOT INSPECTED |
| 7 | Phase 1 dependency | Not known | §19 Phase 1 and gate G1 name `lead-followup` as the Class D vehicle. The hard stop blocks G1 as written (LF-21) |
| 8 | Scope of the ADR | Business promise accepted as proposed | The descriptor mentions only durable waits, eligibility re-checks, events, cancel-safe recovery. Quiet hours (§16 puts them in `appointment-journey`, Phase 5), escalation, channels and per-touch approval are **ADR additions** that earn score points without authority (LF-01) |
| 9 | Named reviewer | Blank, noted | §10.1 requires a **named reviewer** for the business-consequence and net-complexity gates (LF-19) |
| 10 | Verdict | DEFER | Still **DEFER**, with a recommended re-scope to **PILOT on a forcing fixture** (§A) |

---

## A. Executive verdict

**Recommendation: DEFER. The ADR is not ready for human sign-off. Recommended re-scope: record the Phase 1 decision as a PILOT of Class D on a forcing fixture, and defer any ADOPT decision for a production follow-up playbook.**

It is not a REJECT: DBOS remains a plausible fit if the playbook turns out to be long-lived and event-driven. It is not an ADOPT or CONDITIONAL ADOPT because:

1. **The ADR treats a framework test vehicle as a business workload.** plan4.2 §16 lists `lead-followup` in the forcing set, and its descriptor answers "what it proves" about the framework. The ADR then scores 12/14 on business semantics that no authority has approved, several of which are not even in the descriptor (LF-01).
2. **The reliability promise contradicts the governing plan.** "Exactly-once semantics per touch" conflicts with ADR-006, §9.5 and §15. The ADR describes no effect-state discipline and puts sends "inside transaction steps" (LF-02).
3. **Identity and serialization are contradictory and violate §15.2.** The run handle is "derived per invocation" while serialization is "derived from the lead"; the latter is a workflow ID alone, which §15.2 says does not serialize an entity; neither mentions the tenant (LF-03, LF-04).
4. **The Gate 4 comparison is not a fair comparison.** Alternative A lists work B also needs, omits a sequence-table alternative (A′), gives unsourced estimates and omits DBOS lifecycle costs. plan4.2 makes A′ unavailable in Phase 1 scope (River is Phase 3), which is a legitimate reason, but the ADR does not give it (LF-05).

**Two compliant paths (owner chooses, O-00):**

| | Path P — pilot on a forcing fixture | Path W — production workload |
|---|---|---|
| What lead-followup is | A reference sequence that exercises Class D under test connectors (plan4.2 §16, Phase 1) | A real business follow-up playbook |
| Decision label | **PILOT** (ADR-001 §9: limited proof with measurable acceptance criteria) | ADOPT, only on confirmed requirements and proof |
| Business semantics | A minimal reference sequence, explicitly labelled test semantics; the owner approves it. No production promise | The full register in §E |
| Gate 3 | Assessed as a framework consequence (G1: no duplicate confirmed effects, version-safe recovery) | Assessed as a business consequence |
| Gate 4 | Alternative A′ unavailable in Phase 1; state the scope reason and what the pilot must measure | Fair comparison including A′, which needs River (Phase 3, G3) |
| Real channels | Impossible before G4 (ledger, reconciler, out-of-band approval, blast-radius controls, first-write tests) | Same |
| Effect on G1 | Unblocks the Class D vehicle once approved | Blocks G1 until the real playbook is confirmed |

I recommend Path P for increment 6 because plan4.2 itself places `lead-followup` in Phase 1 with test connectors and no real provider writes. That is a recommendation, not an authorization; the choice belongs to the owner.

**In plain language for the business owner.** The ADR asks you to approve a technology for a business process you have not yet defined, and it promises "no lost and no duplicated touches", which cannot be literally guaranteed when messages go through an outside provider. The honest promise is: *at most one confirmed send per touch is recorded; if a send's outcome cannot be established it is held for review and not blindly repeated; recipient-side delivery is not guaranteed.* First decide whether this is a test sequence for the platform or a real follow-up process.

**What the ADR does well (keep):** DRAFT labelling; `[PROPOSED]` tagging; a hard stop; explicit Alternative A/B lists; a §7 checklist with named planned tests; a mock-only boundary; entity identity named as the lead; and, structurally, it carries nearly every field §10.1 requires (business promise, score, gates, alternatives, custom machinery, residual work, operational cost) with evidence links still to come.

**What would turn DEFER into a defensible decision:**

- **Path P → PILOT:** C1 the owner approves the reference sequence as test semantics (O-00 and the Path-P rows in §E); C2 the corrected effect protocol (§F.4) and identity model (§F.2) are in the ADR; C3 the ADR binds to the plan4.2 contracts listed in LF-20; C4 the pre-approval substrate probes in §G are green on real PostgreSQL; C5 a named reviewer is recorded.
- **Path W → ADOPT:** all of the above, plus the full §E register, a fair A′ comparison with measured data (after River exists), a version-drain policy bounded by the confirmed maximum sequence duration, and a production-channel plan that waits for G4.
- **REJECT:** a confirmed production playbook that is short, time-driven, and tolerant of restart. Then the workload is Class B/C and fails Gates 1 and 4.

---

## B. Findings

Severity: **BLOCKER** (must be resolved before human review proceeds), **HIGH**, **MEDIUM**, **LOW**. "ADR location" uses the ADR's own headings and bullets (it has no section numbers). "plan4.2 §" cites the governing plan.

### Summary

| ID | Sev | Title | Rev 1 | Needs |
|---|---|---|---|---|
| LF-01 | BLOCKER | Forcing fixture treated as a business workload; score rests on unapproved semantics, some beyond the descriptor | BLOCKER | Human decision |
| LF-02 | BLOCKER | "Exactly-once" overclaim contradicts plan4.2; no effect-state discipline; sends "inside transaction steps" | BLOCKER | Engineering plus owner |
| LF-03 | BLOCKER | Invocation identity conflated with sequence identity; workflow ID alone violates §15.2 | BLOCKER | Engineering plus owner |
| LF-04 | HIGH | Entity identity omits tenant | HIGH | Engineering |
| LF-05 | HIGH | Gate 4 comparison asymmetric and unsourced; scope reason not stated | HIGH | Engineering analysis |
| LF-06 | MEDIUM | ADR does not bind to the plan4.2 version-isolation policy | HIGH | Engineering plus owner |
| LF-07 | MEDIUM | ADR silent on effect-time revalidation, blast radius and kill switch that plan4.2 requires | HIGH | Engineering |
| LF-08 | MEDIUM | Approval requirement unconfirmed; decision-to-resume window; ordering with eligibility | HIGH | Owner plus engineering |
| LF-09 | HIGH | Cancellation semantics undefined | HIGH | Owner plus engineering |
| LF-10 | HIGH | Consent, opt-out, suppression, frequency and ownership rules absent | HIGH | Human decision |
| LF-11 | MEDIUM | Gate 2 basis misstated | HIGH | ADR correction |
| LF-12 | MEDIUM | Ownership of effects, connectors, approvals and serialization not stated; Phase 1 versus Phase 4 capability | MEDIUM | Engineering clarification |
| LF-13 | MEDIUM | Planned tests are weaker than plan4.2 §18 and insufficient | MEDIUM | Engineering |
| LF-14 | MEDIUM | Gate 3 consequences partly unsupported; important ones missing | MEDIUM | Owner confirmation |
| LF-15 | MEDIUM | Partitioned-queue option not analyzed against required behavior | MEDIUM | Engineering analysis |
| LF-16 | MEDIUM | §7.7 operations and §7.8 edition obligations absent | MEDIUM | ADR addition |
| LF-17 | MEDIUM | No policy for playbook changes to in-flight sequences | MEDIUM | Human decision |
| LF-18 | LOW | Restore-from-backup replay hazard (covered at plan level; ADR silent) | LOW | Engineering note |
| LF-19 | LOW | Housekeeping and unnamed reviewer | LOW | ADR correction |
| LF-20 | MEDIUM | ADR does not bind to plan4.2 §9.1 and §10.1 required fields | new | ADR addition |
| LF-21 | MEDIUM | Gate G1 depends on an unapproved ADR | new | Owner decision |

---

#### LF-01 — Forcing fixture treated as a business workload; score rests on unapproved semantics
- **Severity:** BLOCKER
- **ADR location:** header (OSI-1 statement and "Governing descriptors"); *Business promise*; *Score* (all bullets); *Gate 1–3*
- **Problem:**
  1. plan4.2 §16 places `lead-followup` in the table whose last column is "Purpose: what it proves", and states "The forcing set stays `crm-read` + `lead-followup` + `contact-import`". Its descriptor describes what the fixture proves about the framework (durable waits, eligibility re-checks, events, cancel-safe recovery). It does not establish durations, touch counts, channels or consequences. Using it as evidence that a *business* process needs DBOS is circular: the catalog assigns Class D to exercise Class D, and the ADR confirms Class D from the catalog.
  2. The ADR adds semantics the descriptor does not contain and scores them: quiet hours ("out-of-window"; plan4.2 §16 assigns quiet hours to `appointment-journey`, Phase 5), escalation, channels, and per-touch approval parking ("human 1").
  3. The ADR itself states no approved source specifies the sequence, then awards points whose justification is that sequence ("multi-day lead lifecycle", the stage list).
- **Why it matters:** the rubric is "a decision aid, not a selector" and the decisive criteria are durable business requirements and comparative complexity (`DBOS.md` §4). The band also matters: 11–14 requires "explicit justification for a custom workflow engine", while 8–10 requires an "architectural comparison". A 12 pre-loads the argument.
- **Evidence:** ADR header and score bullets; plan4.2 §16, §10.1.
- **Resolution:** the owner decides what `lead-followup` is in Phase 1 (O-00). Under Path P, the ADR becomes a PILOT with explicitly labelled test semantics. Under Path W, a real playbook is confirmed first (§E). Either way the ADR presents a floor/ceiling score (§D), not a single 12.
- **Type:** Human business decision.

#### LF-02 — "Exactly-once" overclaim contradicts plan4.2; effect discipline missing
- **Severity:** BLOCKER
- **ADR location:** *Business promise* ("no lost, duplicated, or out-of-window touches"); *Score › effects 2* ("need exactly-once semantics per touch"); *§7 checks › tx boundary*; *External-effect boundaries* ("sends happen only inside transaction steps recording the touch first")
- **Problem:**
  1. DBOS steps are at-least-once: a step interrupted before its completion is durably recorded runs again on recovery. DBOS transactions give exactly-once for SQL effects in the datasource database only. Neither extends to a message accepted by an outside provider.
  2. **plan4.2 already states the correct contract.** ADR-006 is "honest at-least-once with UNKNOWN quarantine and reconciliation"; §9.5 defines `PENDING → ATTEMPTING → CONFIRMED or FAILED or UNKNOWN` and the target "no duplicate confirmed business effects, no silently lost operations, and no blind retries of unknown outcomes"; §15 says a timeout after provider acceptance is UNKNOWN, not FAILED. The ADR's wording contradicts the plan it is meant to follow and never mentions these states.
  3. "Sends happen only inside transaction steps" is ambiguous. If it means the provider call runs inside a database transaction, a rollback can erase the intent record after the provider has already sent, the transaction stays open across a network call, and replay re-executes the call. If it means "a transaction step records intent first, then a separate step sends", that is acceptable but is not what the text says.
  4. Automatic step retries on the send step would turn a timeout into a duplicate unless the provider honors idempotency keys for the whole retry window.
  5. **Phase boundary.** plan4.2 puts the reconciler, out-of-band approval and first-write tests in Phase 4 (G4); Phase 1 has only "core `effects/`". In Phase 1 an UNKNOWN effect can be recorded and quarantined, not reconciled. The ADR should say so and say real channels are blocked until G4.
- **Why it matters:** a duplicate customer-facing message is irreversible (Gate 3), and the promise is one the mechanisms cannot keep.
- **Evidence:** ADR text; plan4.2 ADR-006, §9.5, §15, §19 (Phase 1, Phase 4); DBOS Go step and transaction documentation. Phase 1 effects implementation: NOT INSPECTED.
- **Resolution:** replace the guarantee language (§H P-02, P-07, P-10); adopt the protocol in §F.4 on the shared effects contract; require a per-channel declaration of provider idempotency and reconciliation capability; define the UNKNOWN policy (O-10).
- **Type:** Engineering change plus owner decision.

#### LF-03 — Invocation identity conflated with sequence identity; workflow ID alone violates §15.2
- **Severity:** BLOCKER
- **ADR location:** *§7 checks › workflow ID* ("stable run handle derived per invocation"); *Entity identity* ("deterministic workflow ID derived from the lead, or a partitioned DBOS queue keyed by lead"); *Score › coordination 1*
- **Problem:**
  - If the ID is derived per invocation, two agents with different idempotency keys create two workflows for one lead: no serialization.
  - If it is derived from the lead, that is "a workflow ID alone", which plan4.2 §15.2 says "do[es] not serialize access to an entity". A completed or cancelled sequence then blocks any later legitimate campaign, and a second submission with different inputs may be answered with the first run's handle (behavior with differing inputs: NOT PROVEN, needs a probe).
  - Neither defines a sequence generation or what a submission should do when a sequence is active, completed or cancelled (O-14).
  - plan4.2 §9.1 requires every consequential write tool to declare an `entity_key` serialization scope or a reviewed `no_entity` reason; the ADR's "Entity identity" is not stated in those terms and lacks the tenant.
- **Resolution:** adopt the five-identity model (§F.2); enforce "at most one active sequence per (tenant, lead)" in PostgreSQL independent of DBOS (§F.3); declare `entity_key` per §9.1; owner defines re-submission semantics (O-14). §15.2's allowed Class D mechanism (partitioned queue, concurrency one per entity) can be added for execution ordering once probed (LF-15).
- **Type:** Engineering change plus owner decision.

#### LF-04 — Entity identity omits tenant
- **Severity:** HIGH
- **ADR location:** *Entity identity* ("the sales lead (`lead_id`)")
- **Problem:** lead IDs are very likely unique only within a tenant. A workflow ID or uniqueness key built from `lead_id` alone lets one tenant's submission collide with, return the handle of, or be blocked by another tenant's sequence. `DBOS.md` §7.4 requires identities scoped by tenant and business operation; plan4.2 §9.3 and §12 make tenant scope and transaction-local tenant context mandatory.
- **Evidence:** ADR text; `DBOS.md` §7.4; plan4.2 §9.3, §12. Whether `lead_id` is globally unique in the source system: NOT PROVEN.
- **Resolution:** entity identity is `(tenant_id, lead_id)` at minimum (plus source system if more than one); every derived identity carries the tenant; add the cross-tenant identical-lead-ID test (§G).
- **Type:** Engineering change.

#### LF-05 — Gate 4 comparison is asymmetric, unsourced, and does not state its scope reason
- **Severity:** HIGH
- **ADR location:** *Alternative A*, *Alternative B*, *Custom machinery estimate*, *Residual work*
- **Problem:**
  1. **Alternative A is inflated.** Per-lead locking, the per-touch idempotency ledger and a recovery runbook are needed under B too (the ADR's own B list includes per-lead serialization wiring, and the effect ledger is a shared framework contract). Counting them only against A biases the comparison.
  2. **The simpler Alternative A′ is missing:** a `lead_followup_sequences` row per sequence (`status`, `next_touch_no`, `next_due_at`) plus a scheduled job per next touch with execution-time re-checks. For a linear, time-driven playbook it is plausibly simpler: state is visible in SQL, playbook changes are data rather than code versions, and there is no replay or version-drain problem. Events matter only at the next touch decision, which re-checks the row anyway. **I do not conclude A′ wins; the ADR must show it does not.**
  3. **plan4.2 changes what is possible.** River is Phase 3 (§19, G3) and `executor/river` is not in the Phase 1 package list. In Phase 1, DBOS is the only durable substrate in scope; A′ would need infrastructure outside approved scope. That is a **scope** reason, not a **workload** reason, and ADR-001 Gate 4 asks about total complexity for the workload. The ADR should say which it is relying on. If the owner wants a genuine production Gate 4 (Path W), it must be re-run after River exists.
  4. **Estimates are unsourced:** "2–3 weeks" versus "~1 week" has no breakdown. B excludes work that must precede enabling execution (§7 tests including restart tests, version-drain handling, approval resume handling, tenant context in steps).
  5. **DBOS lifecycle costs are omitted:** application-version and patch discipline, drain on deploy, determinism analyzer and replay tests, executor shutdown ordering (the Phase 1 deadlock finding shows this class of risk), checkpoint growth and retention, stuck-workflow alerting, replay debugging, and edition/licence verification.
  6. **Sunk cost.** Reusing the Phase 1 substrate lowers marginal cost legitimately, but the substrate evidence carries "no business semantics" (the ADR's own words), so it does not de-risk lead-followup-specific windows (LF-02, LF-03).
- **Resolution:** rewrite the comparison (§F.6): identical required behaviors in both columns, include A′ with the Phase 1 scope limit stated, task-level estimates with a basis, lifecycle costs, and a clear split between workload-driven and scope-driven advantages.
- **Type:** Engineering analysis.

#### LF-06 — ADR does not bind to the plan4.2 version-isolation policy
- **Severity:** MEDIUM (was HIGH)
- **ADR location:** *§7 checks › versioning plan* ("`WorkflowDef.Version "1"`; recovery guard rejects unknown versions")
- **Problem:** plan4.2 supplies the policy: each Class D family has a stable DBOS `AppName` and an explicitly managed `ApplicationVersion` tied to the workflow compatibility line, not the manifest or repository version (§11.2, ADR-016); old artifacts are retained until old workflows drain or a verified patch exists; a Phase 1 test requires "version isolation/drain with an old workflow pending"; G0-Core condition 4 requires per-family `AppName`, version policy and drain strategy. The ADR names no family, does not relate `WorkflowDef.Version` to `ApplicationVersion`, and offers a guard that only *detects* mismatch. For multi-day sequences, a rejected workflow is a stranded sequence: a lost follow-up. Also unaddressed:
  - the drain window is bounded by the maximum sequence duration, an unconfirmed playbook parameter (O-02);
  - determinism hazards specific to this workload: time-zone database changes, holiday calendars and playbook configuration read inside workflow control flow can change decisions on replay unless computed in a step and recorded (plan4.2 §11.2 requires external I/O and nondeterminism in durable steps);
  - what happens to a workflow rejected by the guard.
- **Evidence:** ADR text; plan4.2 §11.2, §19, §23.1; v1.9.0 integration NOT INSPECTED.
- **Resolution:** ADR states family `AppName`, version bump rules, patch use, drain procedure, operator action on rejection, and cites the Phase 1 version test.
- **Type:** Engineering change; max duration needs owner input.

#### LF-07 — ADR silent on effect-time revalidation, blast radius and kill switch
- **Severity:** MEDIUM (was HIGH)
- **ADR location:** *Business promise* ("attributable to the invoking agent and authority"); *Residual work* ("audit attribution")
- **Problem:** a touch is sent days after invocation. plan4.2 already requires, for every provider action, a recheck of manifest policy, granted scopes, SaaS-side permissions, tenant/resource scope, quotas and risk controls (§9.3), a recheck of expiry, material hash, authority and self-approval immediately before the consequential action (§15.1), and manifest-declared blast-radius limits and kill switches enforced centrally so workers cannot bypass them (§9.1, §14.2). So this is not a framework gap by plan. But the ADR neither cites these requirements nor declares lead-followup's limits (recipient allowlist, per-lead frequency, max records), and the business rule for revoked or changed authority mid-sequence is unspecified (O-16). Whether Phase 1 implements effect-time revalidation: **NOT PROVEN**.
- **Resolution:** ADR declares the limits and revalidation points; O-16 answered; T-10 probe required.
- **Type:** Engineering plus owner decision.

#### LF-08 — Approval requirement unconfirmed; resume window and ordering undefined
- **Severity:** MEDIUM (was HIGH)
- **ADR location:** *Score › human 1*; *Alternative B › approval parking integration*
- **Problem:**
  1. plan4.2 defines the mechanism: approval is a policy gate, not a class (ADR-005); the approval table is the source of truth; a Class D workflow waits on a durable message with timeout; recheck before the consequential action (§15.1, §14.4). It also lists "approval-pending recovery with mock channels" as a Phase 1 test, so that capability must be demonstrated in Phase 1. It does **not** say lead-followup contains per-touch approvals, and the descriptor does not mention approval. Which touches need approval, who approves, and what rejection or expiry means are open (O-08).
  2. **Decision recorded but workflow not resumed.** If the decision commits and the signal is a separate operation, a crash between them leaves the workflow waiting until expiry, where it may conclude EXPIRED although the table says APPROVED. The table must be re-read on wake, with a resume sweeper. plan4.2 §18's approval tests do not list this window. NOT PROVEN for the lifecycle code.
  3. **Ordering.** Eligibility is a business check not covered by §15.1's recheck list; it must run after approval and before send.
  4. Races among approval, expiry, cancellation and the send claim; duplicate decisions; approver tenant boundary.
- **Resolution:** O-08; keep the approval table authoritative; add the T-06 probe; do not invent roles, thresholds or expiries.
- **Type:** Owner decision plus engineering.

#### LF-09 — Cancellation semantics undefined
- **Severity:** HIGH
- **ADR location:** header descriptor ("cancel-safe recovery"); *Score › recovery 2*
- **Problem:** "Cancel-safe" is undefined. A cancel cannot recall a message already handed to a provider. Open: who may cancel; whether cancel stops new attempts only or also the in-flight touch; what happens to an UNKNOWN touch; whether cancel is effective during pending approval; what a submission arriving mid-cancel does. plan4.2 §9.4 provides poll/signal/cancel operations but not these semantics.
- **Achievable guarantee (to be confirmed):** no *new* send attempt starts after cancellation is durably recorded, provided the send-claim transaction re-checks sequence status under the same row lock. An attempt already begun may complete or become UNKNOWN, and the cancel response must say so.
- **Resolution:** O-09; adopt claim-under-lock (§F.4); T-07 probe.
- **Type:** Owner decision plus engineering.

#### LF-10 — Consent, opt-out, suppression, frequency and ownership rules absent
- **Severity:** HIGH
- **ADR location:** *Business promise*; *Score › stages*
- **Problem:** the ADR lists no consent state, opt-out handling, suppression lists, duplicate-contact limits, lead-ownership changes or contact-preference changes. A touch after an opt-out is likely the highest-severity, least reversible failure and constrains design (how fresh the check must be; what happens to an in-flight send). Regulatory exposure for unsolicited, duplicate or quiet-hour messages may apply depending on channel and jurisdiction; that is for the owner and counsel to determine and is not asserted here. plan4.2 contains no such rules.
- **Resolution:** O-11 to O-13. Required for Path W; under Path P they are out of scope but the fixture must be labelled so no one mistakes it for a compliant playbook.
- **Type:** Human business decision.

#### LF-11 — Gate 2 basis misstated
- **Severity:** MEDIUM (was HIGH)
- **ADR location:** *Gate 2 characteristics* ("waiting, recovery, effects, coordination — four present")
- **Problem:** those are score *dimensions* (`DBOS.md` §4), not the Gate 2 characteristics (`DBOS.md` §3: multiple stages, long waits, external events, human approval, multiple dependent services, retries between stages, fan-out, compensation, cancellation or operator intervention, execution-history visibility). plan4.2 §10.1 does not restate Gate 2, so `DBOS.md` governs until the Go port (G0-Core condition 9). The conclusion survives: the descriptor plausibly yields at least three characteristics (long waits, events, cancellation), so Gate 2 is met at the floor, on a different basis. Fan-out and compensation are absent (a sent message cannot be compensated).
- **Resolution:** correct the basis (P-04). **Type:** ADR correction.

#### LF-12 — Ownership of effects, connectors, approvals and serialization not stated
- **Severity:** MEDIUM
- **ADR location:** *§7 checks › tx boundary* ("touch record"); *Alternative B › mock channel connectors, approval parking integration, per-lead serialization wiring*
- **Problem:** "touch record" suggests a miniapp-owned table for what plan4.2 treats as a framework effect (`effects/protocol.go`: intents, attempts, states, reconciliation). Phase 1 also lists a framework "mock capability endpoint" and `testkit/`, so "mock channel connectors" as miniapp work may duplicate or bypass them. Approval parking and serialization are listed as miniapp work; both have framework contracts (§15.1, §15.2). Each could become an alternative to a Phase 1 contract.
- **Resolution:** the ADR states which are framework contracts used as-is and which are miniapp business state (§F.5). Sequence state is miniapp business state; effect state belongs in the shared ledger; connectors are capabilities behind the proxy; approvals use the approval table. Anything the framework cannot express is an architectural gap to adjudicate, not a workaround.
- **Type:** Engineering clarification.

#### LF-13 — Planned tests are weaker than plan4.2 §18 and insufficient
- **Severity:** MEDIUM
- **ADR location:** *§7 checks*
- **Problem:** see §G.1. plan4.2 §18 requires, among others, keyed-serialization tests (same-entity serializes, different entities parallel, retries and deduplication do not violate order), approval tests (expiry, material change, self-approval, replay, out-of-band approver), version tests ("old pending workflows resume only on compatible old code or a tested patch", old artifact not retired before drain), PITR tests and PII-out-of-checkpoint tests. The ADR's version check only rejects unknown versions; its crash test covers a kill during a wait, not during the send window; its duplicate-submit test covers one key; and "op cost" is not a test.
- **Type:** Engineering change.

#### LF-14 — Gate 3 consequences partly unsupported; important ones missing
- **Severity:** MEDIUM
- **ADR location:** *Gate 3 failure consequence*
- **Problem:** "silently lost revenue attribution" conflates a missed outreach with a reporting problem, and a lost touch is detectable by query (an active sequence with an overdue due-time). Missing: touches after opt-out; follow-ups to a converted or reassigned lead; wrong-contact sends; sender-reputation damage from duplicates; per-message provider cost; irreversibility of every send; quiet-hour semantics (send time versus delivery time). Under Path P the relevant consequence is the framework's (G1), not these.
- **Resolution:** owner confirms (O-17). **Type:** Owner confirmation.

#### LF-15 — Partitioned-queue option not analyzed against required behavior
- **Severity:** MEDIUM
- **ADR location:** *Entity identity*
- **Problem:** a queue partitioned by lead with concurrency one serializes *execution*, not *submission*; two submissions would queue as two workflows and run one after another unless deduplicated. **NOT PROVEN:** whether a long-sleeping workflow occupies its partition slot for its whole life (likely, which turns "second submission" into "queued behind a multi-day sequence"). plan4.2 §11.2 and §15.2 already say ordering, fairness and interaction with queue limits must be tested.
- **Resolution:** do not choose a mechanism before O-14; run probe P-5. **Type:** Engineering analysis.

#### LF-16 — §7.7 and §7.8 obligations absent
- **Severity:** MEDIUM
- **ADR location:** *§7 checks* (eight items)
- **Problem:** `DBOS.md` §7.7 requires retry limits, terminal-failure handling, cancellation, failed-workflow inspection, recovery ownership, history retention, operator intervention and alerting; the ADR's "recovery policy" covers only kill-and-resume. §7.8 requires verifying the actual DBOS edition and version; plan4.2 forbids importing Enterprise modules without explicit owner approval (App. F, F16) and leaves Enterprise scope as open owner question 3.
- **Resolution:** add both to the checklist with linked evidence. **Type:** ADR addition.

#### LF-17 — No policy for playbook changes affecting in-flight sequences
- **Severity:** MEDIUM
- **Problem:** if timing, channels or quiet hours change mid-flight, does a sequence follow the version it started under or the latest? Code-versioned workflows tend toward the former; table-driven designs toward the latter. It is a business choice that also constrains LF-06 and the Gate 4 comparison.
- **Resolution:** O-15. **Type:** Human business decision.

#### LF-18 — Restore-from-backup replay hazard
- **Severity:** LOW (HIGH before any production channel)
- **Problem:** plan4.2 §12 and §18 already treat PITR as an external-effect recovery event with a runbook and a pre-GA drill. The ADR is silent; Phase 1 uses mock channels, so no production impact now.
- **Type:** Engineering note.

#### LF-19 — Housekeeping and unnamed reviewer
- **Severity:** LOW
- **Items:** OSI-1 is cited without a link and appears nowhere in plan4.2; the workflow family and `AppName` are not named; "manual recovery runbook" appears in Alternative A though B needs one too; "op cost" is a claim, not a measurement; the ADR has no explicit "Gate 4" conclusion (it presents Alternatives A and B but never states whether Gate 4 is met); the reviewer is `<awaiting owner>` whereas plan4.2 §10.1 says the business-consequence and net-complexity gates "require a named reviewer" and G0-Core condition 9 requires human approval responsibilities to be defined; the 12/14 arithmetic is correct (2+2+2+2+2+1+1) but its inputs are the issue.
- **Type:** ADR correction.

#### LF-20 — ADR does not bind to plan4.2 §9.1 and §10.1 required fields
- **Severity:** MEDIUM (new)
- **Problem:** plan4.2 §10.1 requires the record to contain business promise, score, Gates 1–4, alternatives, custom machinery, residual work, operational cost, and links to evidence. Present: promise, score, Gates 1–3, alternatives, estimates, op-cost bullet. Missing or weak: an explicit Gate 4 conclusion, evidence links (tests are PLANNED), a named reviewer. plan4.2 §9.1 additionally requires, for a consequential write tool, `entity_key` (or reviewed `no_entity`), blast-radius limits, a kill-switch policy reference, and a reference to the approved decision record. The ADR declares none of the first three.
- **Resolution:** P-13. **Type:** ADR addition.

#### LF-21 — Gate G1 depends on an unapproved ADR
- **Severity:** MEDIUM (new; consequence, not defect)
- **Problem:** plan4.2 §19 Phase 1 lists `lead-followup` as the Class D vehicle and requires "a reviewed `adr001.md` decision record … before enabling its DBOS execution". G1 requires Class D working on real Postgres with no duplicate confirmed effects and version-safe recovery. The hard stop means G1 cannot pass until the ADR is signed. The ADR's own note that substrate probes proceed independently shows G1's Class D evidence could come from a generic probe workflow, but plan4.2 names `lead-followup`.
- **Resolution:** O-00 (choose path); if Path W is chosen, the owner may also re-designate the Phase 1 Class D vehicle. Do not weaken the hard stop.
- **Type:** Owner decision.

---

## C. Four-gate assessment

Thresholds: `DBOS.md` §3 (plan4.2 §10.1 adopts the business questions as authoritative and does not restate them).

| Gate | Governing threshold | Assessment | Evidence | Unresolved |
|---|---|---|---|---|
| **1 Durable need** | One or more of: recovery after termination, durable waits, resume from checkpoint, progress across deployments, multiple independent stages, durable wait for event or human. Not met if the operation "can safely restart from the beginning and has no meaningful progress to preserve" | **CONDITIONALLY MET.** Path P: met as a fixture requirement (plan4.2 §16). Path W: depends on real wait durations. Preserved progress (touches already sent) is also derivable from a sequence row plus the effect ledger, so Gate 1 does not by itself distinguish DBOS from a table-driven scheduler | plan4.2 §16; durations NOT CONFIRMED | O-00, O-01, O-02, O-06 |
| **2 Orchestration complexity** | At least two substantial characteristics from the §3 list | **MET AT FLOOR, on a different basis than the ADR cites.** Long waits, events and cancellation or operator intervention follow from the descriptor. Fan-out and compensation absent. Multiple stages and human approval unconfirmed | Descriptor; LF-11 | Basis correction; O-05, O-08 |
| **3 Business consequence** | Failure can break a business promise or create expensive reconciliation work | **Path P: assessed as a framework consequence (G1), plausible. Path W: PLAUSIBLE, NOT ESTABLISHED.** Duplicates and out-of-window or post-opt-out sends are credible, severe and irreversible; the promise is unconfirmed; "silently lost" is an assumption | ADR Gate 3; LF-10, LF-14 | O-11 to O-13, O-17 (Path W) |
| **4 Net complexity** | "Choose DBOS only when Alternative B is genuinely simpler and provides stronger guarantees" | **NOT MET AS PRESENTED / NOT PROVEN.** Unfair comparison, unsourced estimates, no A′, DBOS lifecycle costs omitted. **plan4.2 puts River in Phase 3, so A′ is unavailable in Phase 1**: a scope reason the ADR does not state | LF-05; plan4.2 §19 | Fair comparison; scope reason stated; production Gate 4 only after G3 |

**Honest preliminary view (not a decision):** for a linear, time-driven sequence with exits checked at the next touch, A′ is competitive and probably simpler to operate. DBOS gets stronger as the playbook becomes branching or event-reactive, parks per-touch approvals, or when the uniform Class D lifecycle (status, signal, cancel handles) is itself a framework requirement. In Phase 1 the framework requirement is the operative reason, which is why Path P is the natural fit.

---

## D. Revised score (14-point rubric, `DBOS.md` §4)

Points are awarded only for semantics supported by an authoritative source. "Ceiling" is what the ADR author's proposals would earn **if** the owner confirms them; it is not a score today.

| Dimension | ADR | Floor (supportable now) | Ceiling if proposals confirmed | Rationale |
|---|---|---|---|---|
| Process lifetime | 2 | **1** | 2 | Descriptor implies waits beyond a request, but "multi-day" is `[PROPOSED]` (O-02) |
| Execution stages | 2 | **1** | 2 | "Eligibility re-checks" implies dependent operations (1). Two requires the confirmed touch count and stage list (O-01, O-05) |
| Waiting | 2 | **2** | 2 | Descriptor: "durable waits" and "events" |
| Recovery | 2 | **2** | 2 | Descriptor: "cancel-safe recovery"; resume from prior progress |
| External effects | 2 | **1** | 2 | Sends are consequential, but "multiple" customer-facing sends are unconfirmed; Phase 1 is mock-only (O-01, O-03) |
| Coordination | 1 | **1** | 1 (2 if provider rate limits or cross-lead caps are confirmed) | Per-lead serialization is basic concurrency and needed under any design |
| Human involvement | 1 | **1** | 1 (2 only if per-touch approvals are part of the process) | Operator cancellation is "occasional manual action". The ADR's justification (approval parking) is not in the descriptor (O-08) |
| **Total** | **12** | **9** | **12** | Floor falls in the 8–10 band ("architectural comparison"); ceiling in the 11–14 band |

At 9 the ADR must run a real architectural comparison. At 12 the rubric shifts the burden toward justifying a custom engine. The ADR currently claims the second without evidence for it. Under Path P the score is not the selector; the PILOT criteria are.

---

## E. Business-owner decision register

Nothing below has a default. "Structural options" are shapes the sources support, not recommended values. **Path:** P = needed for the Path P pilot fixture (values may be explicitly labelled test values); W = needed only for a production workload. Status classes: **Defined**, **Proposed**, **Missing**, **Not necessary**.

| ID | Decision | Path | Status | Structural options | Consequence | Exact owner input required |
|---|---|---|---|---|---|---|
| O-00 | What is `lead-followup` in Phase 1? | P, W | Defined as fixture by plan4.2 §16; treated as workload by the ADR | Forcing fixture (PILOT); production workload (ADOPT); re-designate the Class D vehicle | Decides the ADR label, which gates apply, whether G1 is blocked | Path P or W (or a different vehicle) |
| O-01 | Number of touches | P, W | Missing | Fixed count; varies by lead attribute; open-ended until an exit | Drives stage and effect counts, score, test volume | The count or rule (test value acceptable under P) |
| O-02 | Timing: intervals, deadlines, maximum sequence duration | P, W | Missing | Fixed offsets; computed from events; business-day based | Sets lifetime, version-drain window, DBOS-versus-table balance | Intervals and longest sequence (test values acceptable under P) |
| O-03 | Channels: allowed set, order, fallback | W | Missing | Single channel; ordered fallback; per-lead preference | Determines provider idempotency needs | Channels, order, fallback, provider per channel |
| O-04 | Quiet hours: zone, days, holidays, basis | W | Missing (plan4.2 §16 puts quiet hours in `appointment-journey`) | Lead, tenant or sender zone; basis send time or delivery time; calendar source | Out-of-window prevention is only as good as the basis | Zone rule, calendar, basis |
| O-05 | Eligibility at submission and before each touch | P, W | Proposed (stage exists, conditions missing) | Condition list; source of truth; staleness tolerance | Defines the re-check stage and its data dependencies | The conditions and freshness (a trivial condition suffices under P) |
| O-06 | Exit conditions | P, W | Missing | Conversion, opt-out, invalid contact, loss of eligibility, reply, ownership change | Whether events must preempt waits or are checked at next touch | Exact events and whether any needs immediate effect |
| O-07 | Escalation: trigger, recipient, mechanism | W | Missing | None; operator task; notification to a named role | Adds an effect and audit path | Whether it exists; condition, target, mechanism |
| O-08 | Approval | P, W | Proposed (existence only); Phase 1 test requires approval-pending recovery to be demonstrated | None; on sequence start only; on specific touches | Per-touch approval is the main reason the human dimension can reach 2 | Which touches (if any), approver roles, rejection, expiry period and behavior |
| O-09 | Cancellation | P, W | Descriptor level only | By initiator; by tenant admin; by system on exit | Cancel and claim contract; in-flight behavior | Who may cancel; what happens to an in-flight or UNKNOWN touch |
| O-10 | Failure behavior: channel unavailable, send failed, outcome UNKNOWN | P, W | Missing | Skip and continue; pause; escalate; stop; hold for manual review | Whether an unknown outcome costs a touch or risks a duplicate | The rule for each of the three failure kinds |
| O-11 | Consent, opt-out, suppression | W | Missing | Source of truth; check timing; in-flight behavior | Strictest consistency requirement | Authoritative consent source and required freshness |
| O-12 | Frequency and duplicate-contact limits | W | Missing | None; per lead; across sequences; across channels | Cross-sequence coordination | Whether limits exist and their scope |
| O-13 | Lead ownership or preference changes mid-sequence | W | Missing | Continue; pause; end; re-evaluate | Which events must reach the sequence | Rule per change type |
| O-14 | Re-submission semantics | P, W | Missing | Reject while active; return existing handle; queue next generation; replace; different for completed versus cancelled | Defines active-uniqueness, generation and identity model | Rule for a submission when a sequence is active, completed, cancelled or cancelling |
| O-15 | Playbook version binding for in-flight sequences | W | Missing | Start version; latest; per change | Constrains version policy and design choice | Which applies and whether it varies |
| O-16 | Authority lifetime | P, W | Missing | Continue regardless; revalidate at each send; stop on revocation | Revalidation requirement (LF-07) | Rule when authority is revoked, changed or policy trips mid-sequence |
| O-17 | Confirmed consequences and severities | W | Proposed | Owner ranks lost, duplicate, out-of-window, post-opt-out, wrong-contact | Calibrates Gate 3 | Confirmation and ranking |
| O-18 | Named reviewer for Gates 3 and 4 | P, W | Missing (plan4.2 §10.1) | A named person or role | Required before the ADR can be approved | The name |

**Not necessary:** none identified; each row affects a score input, a state transition or a test.

---

## F. Technical design recommendations

Design follows behavior. These are the smallest mechanisms that prove the invariants once §E is answered; they are not implementation authorization.

### F.1 Invariants

| ID | Invariant |
|---|---|
| I1 | At most one active sequence per (tenant, lead) |
| I2 | At most one CONFIRMED send per touch identity; an UNKNOWN outcome is never auto-retried without evidence the channel's declared capability permits |
| I3 | No new send attempt begins after cancellation, exit, ineligibility, opt-out or approval rejection is durably recorded |
| I4 | Authority and policy state (kill switch, limits) are valid at every send claim |
| I5 | Every identity (sequence, workflow, touch, idempotency, approval) is tenant-scoped |
| I6 | A sequence started under version N completes under compatible code or is drained explicitly, never silently stranded |
| I7 | An approval resolves exactly once, and the workflow never concludes EXPIRED while the table says APPROVED |
| I8 | Every touch is attributable to agent, authority and, where applicable, approver |

### F.2 Identity model (five identities that must not be conflated)

| Identity | Meaning | Candidate derivation (structure only) |
|---|---|---|
| Business entity | The lead | `(tenant_id, lead_id)` plus source system if several; declared as the manifest `entity_key` (plan4.2 §9.1) |
| Sequence or generation | One follow-up campaign for a lead | `sequence_id` minted in PostgreSQL at accept; new generation per accepted campaign |
| Invocation and idempotency | One request and its replays | `(tenant, tool, idempotency_key)`, mapping to a `sequence_id` or a rejection |
| Run and workflow | One DBOS execution | Workflow ID derived from the tenant-scoped `sequence_id`; one workflow per sequence |
| Effect or touch | One send | `(tenant, sequence_id, touch_no, channel)`; also the provider idempotency or client reference where supported |

### F.3 Serialization: required behavior versus candidate mechanisms

Scenario: two agents submit follow-ups for the same lead concurrently with different invocation IDs and idempotency keys.

| Question | Workflow ID from lead only | Partitioned queue by lead | PostgreSQL active-uniqueness plus sequence record |
|---|---|---|---|
| Two workflows created? | No (second maps to same ID) | Yes, queued; run one after another | No: a partial unique index on active sequences admits one |
| Second request outcome | Existing handle; differing inputs may be silently ignored (NOT PROVEN) | Accepted; runs after the first ends unless deduplicated | Defined by O-14 |
| Existing sequence completed | Old ID blocks a new campaign | Second campaign runs | New generation allowed |
| Later legitimate campaign | Blocked unless the ID adds a generation | Allowed | Allowed |
| What defines a new generation | Nothing | Nothing | An accepted sequence row |
| Two active sequences sending at once | Prevented only while the ID is unique | Prevented by concurrency one | Prevented by the constraint; also by claim-under-lock |
| Cancel while another submission arrives | Undefined | Undefined | Governed by which statuses count as active (O-09, O-14) |
| plan4.2 §15.2 position | "Workflow IDs alone do not serialize" | Allowed for Class D; tests required | Consistent with the entity-lock family for A, B and C; a business-level uniqueness check for D |

The first two alone do not prove I1; the third does, independent of DBOS. The smallest design proving I1 is the constraint plus the sequence record, with the workflow ID derived from `sequence_id`. A partitioned queue can be added for execution ordering if probe P-5 shows a need.

### F.4 Effect protocol per touch

DBOS rules forbid calling transactions, enqueues or message operations inside a step body, so the protocol is a sequence of separate durable units on the shared effect contract (plan4.2 §9.5, §15):

1. **Record intent** (transaction): effect `PENDING` for the touch identity.
2. **Claim** (transaction, under the sequence row lock): re-check sequence status, eligibility, consent, quiet-hours basis, approval state, authority and kill switch or limits. Move `PENDING` to `ATTEMPTING` with an attempt number. If a check fails, no send; record the outcome.
3. **Begin attempt and send** (one step): first an ordinary compare-and-set marking the attempt as started. If the marker was already set, an earlier execution passed this point: outcome is UNKNOWN and the provider is **not** called unless the channel's declared idempotency makes replay safe. Otherwise call the provider through the capability proxy with the touch identity as idempotency or client reference. **No automatic step retries** unless the provider honors the key for the whole retry window; timeouts and ambiguous responses map to UNKNOWN.
4. **Record outcome** (transaction): CONFIRMED, FAILED or UNKNOWN with provider reference and evidence.
5. **UNKNOWN handling:** quarantine. In Phase 1 only recording and quarantine exist; the reconcile-by-read worker is a Phase 4 deliverable (§19). Retry is permitted only on definitive evidence: provider-asserted idempotent replay, a provider-reported terminal rejection, or a reconciliation read with documented consistency. Otherwise apply the owner's O-10 rule.

Whether the Phase 1 effects package already provides the begin-attempt compare-and-set: **NOT INSPECTED**. If it does not, that is an architectural gap to adjudicate, not a miniapp workaround.

| # | Failure window | State on recovery | Safe action | Guarantee | Residual risk or owner input |
|---|---|---|---|---|---|
| 1 | Intent recorded, crash before send | PENDING, or ATTEMPTING with no marker | Re-run claim (re-check everything; time may have passed), then send | No send occurred, so no duplicate | Touch may now be out-of-window or cancelled; the re-check decides |
| 2 | Channel accepted, crash before confirmation recorded | ATTEMPTING, marker set | Treat as UNKNOWN; do not resend; quarantine | At-most-once attempt; touch possibly delivered but unrecorded | Lost-touch versus duplicate trade-off is the owner's (O-10) |
| 3 | Timeout, outcome unknown | Same as 2 | Same as 2 | Same | Same |
| 4 | DBOS replays an interrupted step | Completed steps return recorded outputs; the interrupted send step re-executes | Re-entry sees the marker and does not resend | Step execution is at-least-once; the CAS makes the attempt at-most-once | NOT PROVEN for this workload; needs the crash probe |
| 5 | Two workers on one touch | Same row | Claim CAS and unique touch identity let one proceed | The database arbitrates; DBOS alone does not | Overlapping executors during deploy: fencing behavior NOT PROVEN (plan4.2 requires Fly overlap tests, G6) |
| 6 | Retry after partition or ambiguous response | UNKNOWN | Provider-idempotent replay or reconciliation only | Effectively-once only where the provider guarantees it | Per-channel capability must be declared; mock channels prove nothing about it |
| 7 | Cancel or approval rejection races an in-flight send | Sequence status changed; send may be in flight | Claim-under-lock stops new attempts; in-flight attempt resolves to CONFIRMED or UNKNOWN | No new attempt after the durable cancel; a sent message cannot be recalled | Response to the canceller must state the in-flight outcome |

**Guarantee ladder:**

| Guarantee | Achievable? | Condition |
|---|---|---|
| Exactly-once workflow step execution | No | DBOS steps are at-least-once |
| Exactly-once database state transition | Yes | SQL in the datasource database with the transaction integration; NOT PROVEN for these transitions |
| At-most-once external send attempt | Yes | CAS marker before the call; cost is a possible lost touch (UNKNOWN) |
| Effectively-once delivery | Only per channel | Provider idempotency honored for the whole retry window; mock channels cannot demonstrate it |
| Reconciliation of an uncertain effect | Best effort, Phase 4 | Depends on provider read API and consistency |
| Exactly-once delivery to the recipient | No | Outside our control |

### F.5 Framework behavior versus miniapp behavior

| Concern | Framework (plan4.2 source) | Miniapp-specific |
|---|---|---|
| Canonical invocation, authorization before dispatch, server-derived principal and tenant | Required (§6.3, §9.3, ADR-004); Phase 1; NOT INSPECTED | none |
| Effect ledger: five states, attempts, UNKNOWN quarantine | `effects/protocol.go` (§7, §9.5); core in Phase 1, reconciler Phase 4 | Touch identity derivation |
| Per-channel capability declaration (idempotent replay, window, reconcile support) | `CapabilitySpec` carries idempotency and reconciliation operation (§9.2) | Choice of channel and fallback |
| Effect-time revalidation of authority, scopes, quotas; kill switch and limits | Required (§9.3, §14.2, §15.1); implementation NOT INSPECTED | Declared limits; which events stop a sequence |
| Approval table, decisions, expiry, self-approval block, out-of-band channel | `approvals/` (§14.4, §15.1); core Phase 1, channel Phase 4 | Which touches need approval; resume-after-decision handling |
| DBOS executor, datasource, shutdown ordering, per-family `AppName` and version isolation | `executor/dbos` (§11.2, ADR-016) | Family name; version policy; maximum duration |
| Tenant-scoped transaction helper | `internal/store` (§12) | none |
| Sequence state, active-uniqueness constraint, playbook configuration, eligibility and quiet-hours rules (pure domain), generation allocation | none | All of it |
| Mock channels | Framework mock capability endpoint and `testkit/` (§19 Phase 1) | Behavior configuration only; never miniapp-owned connectors |

### F.6 What a fair Gate 4 comparison must contain

Both columns implement the same confirmed behaviors (I1–I8). A′ is a sequence table plus scheduled jobs with execution-time re-checks; B is DBOS. **A′ requires the River substrate (Phase 3, G3), so in Phase 1 it can be assessed on paper only.**

| Cost item | A′ | B (DBOS) | Basis required |
|---|---|---|---|
| Sequence state model | Needed | Needed anyway for I1 and reporting | Task list |
| Per-lead uniqueness | Constraint | Same constraint | Same |
| Effect protocol, ledger, claim | Shared | Shared | Same |
| Durable wait | Scheduled job | Durable sleep | Measured |
| Stuck or overdue detection | Query plus alert | Workflow-state monitoring plus business reconciliation | Operations plan |
| Cancellation and exit | Status update, honored at next claim | Cancel plus signals plus status update (still needs claim checks) | Both need the claim rule |
| Playbook change | Data change, effective at next touch | Version or patch policy, drain | O-15 |
| Version and deploy risk | Low | Drain window bounded by max duration | O-02 |
| Per-touch approval parking | Approval table plus one-shot job | Workflow wait on a durable message | O-08 |
| Lifecycle API (status, signal, cancel handles) | Must exist for Class C or B | Existing Class D lifecycle | Substrate availability |
| Replay and determinism discipline | None | Analyzer (plan4.2 Phase 1) and replay tests | Measured effort |
| Checkpoint growth and retention | None | Per-step rows to retain | Measured |
| Availability in approved phase scope | Phase 3 | Phase 1 | plan4.2 §19 |
| Estimate | Task level with basis | Task level with basis | Required |

---

## G. Acceptance-test matrix

### G.1 Review of the ADR's eight planned checks

| # | Planned check | Invariant | Failure to reproduce | Specific assertion | Environment | Needed |
|---|---|---|---|---|---|---|
| 1 | Determinism lint | Workflow body deterministic | Time, randomness, goroutines, map iteration, DB or provider calls in workflow code | Analyzer flags a known-bad fixture and is clean on the package; replay test reproduces the step sequence. plan4.2 requires strict-mode analyzer in CI for Class D | Local | Before enabling. Lint alone is not proof: add replay |
| 2 | Idempotency keys (DuplicateSubmit) | I1 plus request replay | Same key twice; **different keys for one lead (not covered)** | Same key gives same handle; different key gives the O-14 outcome; exactly one active sequence row | Real PostgreSQL | Before enabling; extend scope |
| 3 | Transaction boundary (AtomicTouch) | Exactly-once database transition of touch or effect row plus checkpoint | Process kill between write and checkpoint | Row appears exactly once; no orphan | Real PostgreSQL with kill | Before enabling. Does **not** cover the external send (T-02) |
| 4 | Stable workflow ID | I5, LF-03 identity model | Retry, cross-tenant collision, re-submission | ID is a pure function of tenant and sequence; cross-tenant identical lead IDs never collide | Real PostgreSQL | Before enabling; ADR text says "per invocation" and must change |
| 5 | Versioning guard | I6 | Old workflow pending while new code deploys | plan4.2 §18: old pending workflows resume only on compatible old code or a tested patch; old artifact not retired before drain; unknown version surfaces as operator-actionable state | Controlled restart with two builds | Before enabling; detection alone is not enough |
| 6 | Payload discipline | Compact identifiers, no credentials or content | Large or secret-bearing input or output | Serialized inputs and outputs of every step bounded and free of secret patterns, message bodies and full contact records; plan4.2 §12 and §18: PII not copied into checkpoints | Local plus real PostgreSQL | Before enabling |
| 7 | Crash recovery | I2, resume | Kill mid-wait **and at every effect boundary (not covered)** | Resumes without repeating a completed touch; a kill in each of windows 1–7 never yields two sends under a non-deduplicating mock | Controlled process-restart, real PostgreSQL | Before enabling; expand |
| 8 | Operational cost | Net complexity | none | **Not a test.** Replace with measured quantities: checkpoint rows per touch, connections (plan4.2 §12 budget formula), recovery time, retention growth | Measured | Before approval (feeds Gate 4) |

### G.2 Required adversarial tests

"Pre-approval probe" means a generic substrate probe that carries no `lead-followup` business semantics, consistent with the hard stop and the ADR's own statement that substrate probes proceed independently. Nothing here creates or enables `lead-followup` code.

| ID | Invariant | Adversarial test | Expected result | Evidence required |
|---|---|---|---|---|
| T-01 (P-1) | I1 | N concurrent submissions for one tenant and lead with different invocation IDs and keys | Exactly one active sequence; the rest follow O-14 | Pre-approval probe; then workload test before enabling. plan4.2 §18 keyed-serialization tests |
| T-02 (P-2) | I2 | Kill the process at every boundary of F.4 against a mock channel that does **not** deduplicate | No window yields two sends; windows 2 and 3 end UNKNOWN, not retried | Pre-approval probe; repeat per workload |
| T-03 | I2 | Provider timeout and ambiguous response | UNKNOWN, quarantined, no automatic retry | Before enabling |
| T-04 | I2 | Mock channel with idempotency keys and a finite window; retry inside and outside the window | Retry only inside; outside is UNKNOWN | Before enabling; labelled mock-only proof |
| T-05 | I5 | Two tenants with identical lead IDs; concurrent submissions and cancellations | No collision, no cross-visibility of handles, approvals or audit | Before enabling; plan4.2 §12 isolation under pooled connections |
| T-06 (P-4) | I7 | Replayed and duplicated approval decisions; crash after decision commit before resume signal; decision racing expiry | One resolution; never EXPIRED when APPROVED; resume sweeper works | Pre-approval probe for the generic window; per workload before enabling. plan4.2 §18 approval tests |
| T-07 (P-6) | I3 | Cancel racing a send claim; cancel during pending approval; submission during cancel | No new attempt after durable cancel; in-flight outcome reported | Pre-approval probe; before enabling |
| T-08 (P-3) | I6 | Old-version workflow sleeping; deploy new code; unknown-version recovery | Drain or compatible completion; rejection visible and recoverable | Pre-approval probe; before enabling |
| T-09 | I2 | Database transaction failure (serialization error, deadlock, connection loss) in a transaction step | Retry or fail cleanly, no partial state, no double effect | Before enabling |
| T-10 | I4 | Revoke authority or trip the kill switch mid-sequence | Behavior matches O-16 | Before enabling; may reveal an implementation gap |
| T-11 | I3 | Opt-out arriving during wait, during claim, during an in-flight send | No new attempt after the durable opt-out | Path W only |
| T-12 | I6 | Time-zone and DST boundaries; tzdata change on replay | Decisions on replay equal recorded decisions | Path W only |
| T-13 (P-5) | LF-15 | Partitioned queue with a multi-day sleeping workflow and a second submission | Documents slot occupancy and deduplication behavior | Pre-approval probe |
| T-14 | I8 | Audit reconstruction for a sequence | Every touch shows agent, authority and approver | Before enabling |
| T-15 | LF-18 | Restore-from-backup replay with mock sends already recorded externally | Sequences paused; reconciliation prevents re-send | Before any production channel (plan4.2 G6 drill) |
| T-16 | I2, plan4.2 §14.2 | Retry storm and duplicate-spike against blast-radius limits | Circuit breaker trips; kill switch holds under concurrent workers | Before enabling |

---

## H. Proposed ADR corrections

The original ADR is not modified. Apply as a patch only after the owner decisions in §E. Table cells avoid the pipe character.

| ID | Location | Current | Proposed |
|---|---|---|---|
| P-01 | Header, descriptor | "Governing descriptors (plan4.2): …" | "The plan4.2 §16 descriptor states what `lead-followup` proves for the framework (forcing set). It does not specify a business playbook. Evidence that depends on sequence parameters is conditional on owner confirmation (§E). Scope: Path P (pilot fixture) or Path W (production workload), per owner decision O-00." |
| P-02 | *Business promise* | "…no lost, duplicated, or out-of-window touches…" | "For each touch identity (tenant, sequence, touch number) the system records at most one CONFIRMED send. A send whose outcome cannot be established is recorded as UNKNOWN, is not retried without evidence permitted by the channel's declared idempotency or reconciliation capability, and is surfaced to operators. The system does not guarantee exactly-once delivery to the recipient. New send attempts are not begun after cancellation, exit or ineligibility is durably recorded. Phase 1 mock channels demonstrate state-machine behavior only; real channels are blocked until gate G4." |
| P-03 | *Score* | "= 12/14" | Replace with the floor and ceiling table in §D; state "Floor 9/14 from authoritative sources; ceiling 12/14 if `[PROPOSED]` semantics are confirmed." |
| P-04 | *Gate 2* | "waiting, recovery, effects, coordination — four present" | "Characteristics per `DBOS.md` §3 supported by the descriptor: long-running waits, external events, cancellation or operator intervention (two or more required). Multiple stages and human approval are unconfirmed. Re-verify against the Go-ported ADR-001 when it exists." |
| P-05 | *Gate 3* | "silently lost revenue attribution … business-visible and hard to remediate" | "Under Path P: framework consequence (no duplicate confirmed effects; version-safe recovery) per plan4.2 G1. Under Path W: consequences awaiting owner confirmation: missed touch (detectable by query), duplicate, out-of-window, touch after opt-out, follow-up to a converted or reassigned lead, wrong-contact send. All sends are irreversible." |
| P-06 | *Alternative A / B / estimates* | Shared items counted only in A; "2–3 weeks" versus "~1 week" | Compare Alternative A′ (sequence table plus scheduled jobs) with B on identical required behaviors; state that A′ requires River (Phase 3) and is assessed on paper in Phase 1; list lifecycle costs of each; give task-level estimates with their basis, or remove the figures; add an explicit Gate 4 conclusion. |
| P-07 | *§7 checks › tx boundary* | "touch record + checkpoint commit atomically" | "Effect intent and outcome records commit atomically with their DBOS checkpoints as database transitions. This proves a database transition, not external delivery. The provider call is a separate step on the shared effects ledger (`PENDING`, `ATTEMPTING`, then CONFIRMED, FAILED or UNKNOWN), with no automatic retry unless the channel declares idempotent replay." |
| P-08 | *§7 checks › workflow ID* | "stable run handle derived per invocation" | "Invocation identity (idempotency key) maps to a sequence; the run handle and DBOS workflow ID derive from the tenant-scoped sequence identity, one workflow per sequence." |
| P-09 | *Entity identity* | "the sales lead (`lead_id`) … deterministic workflow ID derived from the lead, or a partitioned queue" | "Manifest `entity_key` is (tenant_id, lead_id). At most one active sequence per entity is enforced by a PostgreSQL constraint on a sequence record. A workflow ID is not relied on for serialization (plan4.2 §15.2). A partitioned queue may be added for execution ordering after probing. Re-submission semantics are defined by owner decision O-14." |
| P-10 | *External-effect boundaries* | "sends happen only inside transaction steps recording the touch first" | "Intent is recorded in a transaction first; the provider call runs in a separate step through the capability proxy and the shared effects ledger. No provider call executes inside a database transaction." |
| P-11 | *§7 checks › versioning plan* | "recovery guard rejects unknown versions" | "Workflow family `<name>` with its own DBOS `AppName`; explicit `ApplicationVersion` tied to the workflow compatibility line; bump rules; patch use; old artifact retained until drain; maximum sequence duration; operator action for a rejected workflow. Time-zone, holiday and playbook decisions are computed in steps and recorded." |
| P-12 | *Decision* | "PROPOSED ADOPT (pending human review)" | "DEFER. Recommended re-scope: PILOT of Class D on the plan4.2 forcing fixture (Path P), with ADOPT for any production playbook deferred until the playbook is confirmed, River exists (G3) and G4 is passed." (Owner may choose Path W.) |
| P-13 | *Checklist and manifest fields* | n/a | Add explicit Gate 4 conclusion; evidence links; named reviewer; `entity_key`; blast-radius limits and kill-switch reference (plan4.2 §9.1, §14.2); checks for authority and kill-switch revalidation, approval decision and resume window, cancellation race, cross-tenant identity, opt-out race (Path W); `DBOS.md` §7.7 operations and §7.8 edition verification |

---

## I. Approval recommendation

- **Status: NEEDS FURTHER INFORMATION. The ADR is not ready for human sign-off in its current form.**
- **Sequence:** (1) owner chooses Path P or W (O-00) and names the reviewer (O-18); (2) owner answers the relevant §E rows; (3) the ADR is patched per §H; (4) generic pre-approval substrate probes P-1 to P-6 (§G.2) run on real PostgreSQL; (5) Gate 4 is restated per §F.6; (6) the ADR returns for human review with the recalculated score.
- **Hard stop unchanged:** no `lead-followup` workflow code, registration, manifest or DBOS execution, and no increment 6, until an authorized human signs the revised ADR.

**Reviewer (human): ______________________   Date: ______________**

*(Left blank. This review does not sign or approve the ADR.)*

---

## Sources checked

Files read: `lead-followup-adr001.md`, `prompt11.md`, `DBOS.md` (ADR-001), `plan4_2.md` (sections listed in the scope table), `app_category_guidelines.md`. Not available: `adr001-template.md`, G0 adjudication, all Phase 1 source and test files.

DBOS Go documentation (public, checked 2026-11-22; v1.9.0 specifics not verified):
- Steps are at-least-once; transaction and step rules: https://docs.dbos.dev/golang/tutorials/step-tutorial
- Transactions and datasources: https://docs.dbos.dev/golang/tutorials/transaction-tutorial and https://docs.dbos.dev/golang/reference/datasources
- Workflows, queues, partitioned concurrency, deduplication: https://docs.dbos.dev/golang/reference/workflows-steps
- Application version, patching, configuration: https://docs.dbos.dev/golang/reference/configuration
- Recovery and versions: https://docs.dbos.dev/production/self-hosting/workflow-recovery
- Determinism rules: https://docs.dbos.dev/golang/prompting
