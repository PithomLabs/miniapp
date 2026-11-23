# Consolidated Review — Phase 1 Class D Framework Readiness

**Subject:** `lead-followup` ADR-001 and the Phase 1 DBOS execution framework  
**Review inputs:** `adv_review_prompt11.md` and `adv_review2_prompt11.md`  
**Governing architecture:** `docs/plan4.2.md` (authoritative and immutable)  
**Review date represented by the source reviews:** 22 November 2026  
**Status:** Independent review synthesis; not an ADR approval or implementation authorization

---

## 1. Executive conclusion

**The framework direction is capable in principle of supporting workloads for which durable orchestration is justified, but the current Phase 1 implementation has not yet demonstrated several foundational Class D guarantees. The correct conclusion is “not yet proven ready,” not “DBOS is unjustified because `lead-followup` is too simple.”**

The two reviews are strongest when read together:

- `plan4.2` deliberately names `lead-followup` in the Phase 1 forcing set. Its role is to exercise the framework's Class D contracts: durable waits, eligibility re-checks, events, cancellation, recovery, and lifecycle operations.
- The example need not itself be a sophisticated production workflow or independently prove that a sales follow-up process deserves DBOS. A compact, controlled fixture is useful when it tests the framework's generic guarantees without introducing real providers, real customer data, or speculative business rules.
- The real question is whether the framework can safely host more demanding, genuinely DBOS-warranted workflows: long-lived multi-stage processes, durable waits and signals, human decisions, recovery across restarts and deployments, controlled external effects, tenant-safe concurrency, and observable operations.
- Review 1's source-level findings expose potentially serious defects in generic runtime, effects, approval, identity, and DBOS-version behavior. Review 2 correctly grounds the review in `plan4.2`, separates the forcing fixture from a production business process, and identifies framework obligations the ADR must bind to.

**Recommended disposition:** reframe the ADR as a **Class D framework pilot/conformance fixture**, not as a production adoption decision for a fully specified sales playbook. Approve only explicit synthetic fixture semantics sufficient to test the required contracts. Keep production channels, real external writes, and any claim of exactly-once delivery out of scope. Fix and prove the framework-level blockers below before claiming G1 readiness.

This is not a rejection of DBOS. Nor is it approval to proceed to increment 6. The existing ADR hard stop remains in force until the human owner reviews and explicitly approves the revised pilot ADR.

## 2. Review lens and boundaries

The consolidated review separates three questions that were partly conflated in the source reviews.

| Question | What it asks | How it should be decided |
|---|---|---|
| **Framework capability** | Can the framework provide correct, reusable Class D execution for demanding workflows? | Test generic runtime, DBOS integration, durability, versioning, effects, approvals, security, tenancy, and operations. This is the primary Phase 1 objective. |
| **Fixture adequacy** | Does `lead-followup` exercise enough of those contracts to be a meaningful Class D proof? | Use a small, explicitly synthetic sequence with enough stages, durable waits, event/signal handling, cancellation, recovery, and mock effects to exercise the contracts. Avoid unnecessary sales-domain complexity. |
| **Production business adoption** | Should an actual sales follow-up process use DBOS rather than a simpler scheduler or queue? | Decide later, when there is an authoritative playbook, real consequences, volume, operational requirements, and a fair alternative comparison. This is not what the Phase 1 fixture needs to prove. |

A simple fixture is not a weak test by definition. It becomes a weak test only if it avoids the hard properties that distinguish Class D from Class A or B—for example, if it is merely a delayed function call with no event wait, no recovery boundary, no lifecycle transition, or no meaningful state to preserve.

Conversely, adding quiet hours, channel fallbacks, regulatory rules, escalations, multiple campaign generations, or elaborate sales policy merely to make the example look “complex” would be counterproductive. Those are production-playbook questions, not prerequisites for a framework conformance fixture.

## 3. Reconciliation of the two reviews

### 3.1 Findings retained as materially valid

The following concerns are coherent across the reviews or supported by Review 1's direct inspection of the Phase 1 source described in that report:

1. **Exactly-once external delivery is an invalid guarantee.** DBOS workflow/step recovery and PostgreSQL transactions cannot make an external provider send atomic with the database. The framework's `UNKNOWN` state is the right discipline. Wording must align with `plan4.2` and ADR-006: no blind retry of uncertain effects, no duplicate confirmed effects, and no silently abandoned intent. Provider-level idempotency or reconciliation is conditional on the provider and belongs to later real-provider phases.
2. **Workflow identity is not business-entity serialization.** Invocation identity, run/workflow identity, entity identity, and effect identity must remain distinct. A deterministic workflow ID alone does not serialize access to a business entity. The framework needs a clear entity-key contract and a proven concurrency mechanism; the miniapp supplies its business entity and sequence semantics.
3. **Run submission must not create orphan workflows.** A successful DBOS submit followed by a failed persistence of the run handle can leave a live workflow that the invocation record cannot find, return, cancel, or audit correctly. This is a generic framework defect, not a sales-playbook concern.
4. **DBOS family and version policy must be real.** A declarative `WorkflowDef.Version` field is not a recovery policy unless the executor and deployment process enforce it. Per-family `AppName`, explicit application-version behavior, compatibility/drain policy, and pending-workflow transition tests are necessary for long-lived workflows.
5. **Approval is a control-plane-to-workflow protocol, not just an approval row.** A decision must reliably reach the waiting workflow; terminal rejection or expiry must settle/cancel a live run; duplicate decisions and races must be safe; invocation and workflow status must not diverge indefinitely.
6. **Effect-ledger concurrency and tenant scoping need adversarial proof.** Sequential idempotency tests are insufficient if concurrent claims can overwrite one another or tenant identity is not included in effect identity and enforced in SQL predicates.
7. **Operational and recovery behavior is part of Class D capability.** Durable waits, restarts, version transitions, failed workflow inspection, retry limits, retention, alerting, and operator recovery ownership require explicit proof or documented limits.
8. **The existing eight ADR checks are not enough as written.** They need to cover post-submit partial failure, same-key/different-input reuse, cross-key same-entity submissions, cross-tenant identity collisions, approval decision/resume races, denial cancellation, effect uncertainty, old-version pending workflows, and durable-wait restart.

### 3.2 Findings that must be reframed, not carried forward literally

**The missing production playbook is not a blocker to testing the framework.** It is a blocker to claiming that a real sales operation has been approved for DBOS. Under a pilot path, synthetic fixture semantics can be explicitly approved as test inputs and must never be represented as real business policy.

**The claimed production consequences are not required to justify the fixture.** Lost revenue attribution, customer spam, out-of-window contact, opt-out violations, and channel-specific compliance are relevant to a production sales process. They should not be invented to make the Phase 1 fixture appear DBOS-worthy. The framework-level consequence is that a Class D runtime that loses, duplicates, strands, or misattributes durable work is not safe for the kinds of business processes the framework intends to host.

**The absent Phase 4 reconciler is not, by itself, a Phase 1 defect.** `plan4.2` explicitly phases production effect reconciliation and first real writes later. Phase 1 must correctly record and quarantine `UNKNOWN`, refuse blind retries, and keep real provider writes disabled. Do not pull the Phase 4 reconciliation worker into Phase 1 merely to close a review finding. Do not claim that quarantine has been reconciled when it has only been recorded.

**The sales-specific requirements are conditional.** Consent, suppression, quiet hours, lead reassignment, contact preferences, campaign generation, and production escalation belong in a real playbook decision. They are not general framework requirements. The framework must offer extension points and enforce generic authorization, tenancy, effects, audit, and policy boundaries.

**The scheduler/River comparison is not the primary Phase 1 decision.** Review 2 is right that a table plus scheduled jobs is a plausible alternative for a linear, time-driven process, and River is Phase 3 in `plan4.2`. That alternative should be considered when deciding whether a real production sales process deserves DBOS. It should not be used to invalidate the planned Phase 1 DBOS Class D substrate proof, nor should Phase 1's scope limitation be presented as proof that DBOS is inherently simpler.

### 3.3 Conflicts and evidence limitations

- **The 6/14 versus 9/14 score:** Review 1 scored only conservatively established business semantics and arrived at 6/14. Review 2 credited the plan's explicit “durable waits” and “cancel-safe recovery” descriptor and arrived at a 9/14 floor. The difference comes from how those plan-level fixture characteristics are mapped to the business-workload rubric. Neither score answers the primary framework-capability question. For the pilot path, remove the production-adoption score as the decision criterion and use a framework conformance matrix. For a later production path, rescore against a confirmed playbook and the governing rubric.
- **DB-backed test evidence:** the user-reported increment 4–5 run says all 14 test packages passed against local PostgreSQL. Review 1's independent read-only run reported that 16 database-backed tests were skipped because that review environment had neither `TEST_DATABASE_URL` nor Docker; Review 2 did not inspect the Phase 1 code or run tests. These are different execution environments, not proof that the user's successful run did not happen. Preserve the user's run as reported, but file its exact source state, environment, commands, and results in the evidence register. Re-run on the current uncommitted tree before using it as G1 evidence.
- **DBOS sleep residency/edition behavior:** Review 2 raises a relevant scaling question about durable sleeps and open-source versus Enterprise behavior, while noting that its public-documentation check did not verify v1.9.0 specifics. Treat this as an unresolved pinned-SDK/edition verification and capacity-test item—not as a settled limitation or a reason to reject DBOS.
- **Review 2's code-level observations:** it explicitly did not inspect the Phase 1 source. Its plan4.2 requirements are valid inputs, but source-level findings from that review remain requirements to verify, not independent proof of implementation defects.

## 4. Consolidated findings — framework readiness

Severity here is about **G1/Class D framework readiness**, not whether a production sales process should adopt DBOS.

### Blockers to a credible Class D framework proof

| ID | Finding | Consolidated assessment | Required proof or correction |
|---|---|---|---|
| F-01 | Submit/record atomicity and orphan workflows | Review 1 describes a submit-success / run-handle-persist-failure window that leaves a live, unaddressable workflow. This undermines stable run handles and lifecycle control for every Class D miniapp. | Fault-inject failure after DBOS submit and before invocation persistence. Ensure compensation/cancellation or recoverable attachment; retries must return a valid live handle. Audit the recovery path. |
| F-02 | Approval decision and workflow lifecycle are not coupled | Review 1 reports no reliable decision-to-signal bridge, no production expiry assignment, no expiry settlement path, and denial/expiry marking an invocation failed without cancelling the live run. Review 2 correctly notes that approval is a generic policy contract in `plan4.2`, even if this fixture need not require business approval. | Generic Class D approval probe: decision commit then crash before signal; redelivery; duplicate decision; reject/expire/cancel races; approval after cancellation; verify the workflow resumes or terminates and invocation/run state converges. |
| F-03 | Effect ledger concurrency and truthful external-effect guarantees | “Exactly once per touch” is not achievable as stated. Review 1's code inspection also reports non-serialized `claim → load → transition` operations and unconditional updates that may overwrite a concurrent outcome. | Use state-guarded compare-and-set or transaction/row-lock discipline; tenant-scope all identities and mutations; add parallel duplicate-attempt tests. Assert UNKNOWN quarantine and no blind retry. Never equate DB commit with provider delivery. |
| F-04 | Invocation, workflow, entity, sequence, and effect identities are conflated | The runtime run handle is not the business entity key. A deterministic workflow ID does not lock an entity; a per-lead ID without a generation can also prevent later legitimate sequences. | Define framework-level invocation/run identity separately from the manifest `entity_key`. Demonstrate same-entity concurrency control and different-entity parallelism. Keep sequence-generation semantics inside the fixture/domain, with tenant scope included. |
| F-05 | DBOS family/version compatibility is not enforced | Review 1 reports a shared `AppName`, no explicitly managed `ApplicationVersion`, and a `WorkflowDef.Version` value that is not consumed. This conflicts with the per-family and version-drain requirements in `plan4.2`. | Implement or prove per-family `AppName`, explicit application-version policy, compatibility/drain handling, old-artifact retention, and operator-visible rejection. Test an old workflow pending while a new version is deployed. |
| F-06 | Effect-time authority, tenant and blast-radius policy | `plan4.2` requires policy and authority revalidation around consequential actions, central limits, and kill-switch enforcement. Review 2 could not inspect whether the current code implements these; therefore this is a mandatory verification, not a source-proven defect. | Trace every Class D effect path. Verify trusted tenant/authority derivation, authorization before dispatch, revalidation at effect time, tenant-scoped SQL, blast-radius limits and a central kill switch that workers cannot bypass. |

### High-priority framework gaps and proof obligations

| ID | Finding | Required proof or correction |
|---|---|---|
| F-07 | Same idempotency key with different input may replay the first result | Store/compare a canonical input hash on idempotency conflict. Identical request replay returns the same invocation/result/handle; different input under the same key fails explicitly. |
| F-08 | Tenant scope may rely too heavily on RLS or globally keyed effect IDs | Include tenant in entity/effect identities and add explicit tenant predicates to load/update paths, including privileged/bypass-role execution. Test identical entity IDs in two tenants. |
| F-09 | Approval scope/role may be stored but not enforced | If `ApproverScope` is part of the contract, enforce it at decision time; reject decisions from unauthorized roles and decisions against terminal/cancelled invocations. Keep business roles configurable rather than hard-coded to the fixture. |
| F-10 | Durable sleep recovery and scale are unproven | Add a pinned-DBOS-v1.9.0 sleep-across-process-restart test. Inspect the exact SDK/edition behavior and measure memory/worker/connection impact at representative concurrent sleeper counts. Do not infer production capacity from a short unit test. |
| F-11 | Workflow lifecycle observability can conceal executor failure | `Status` must distinguish a stale persisted status from a successful live executor query; surface executor errors as retryable/stale information rather than silently returning an apparently current state. |
| F-12 | Entity queue partitioning is not equivalent to submission deduplication | If a partitioned queue is considered, test whether a long-waiting workflow occupies the partition and whether a second submission queues as another workflow. Enforce business-level uniqueness independently of queue ordering. |
| F-13 | Determinism analyzer and operational controls are part of the framework deliverable | Complete the seven rule families with positive/negative fixtures and strict CI for Class D. Add recovery ownership, retry/terminal-failure policy, history retention, inspection, alerting, and DBOS edition/version verification. |
| F-14 | Framework versus miniapp responsibility is not explicit enough | Framework owns canonical invocation, principal/tenant, policy, lifecycle, audit, approvals, effects protocol, DBOS registration/versioning, and entity-key enforcement primitives. Miniapps own domain state, workflow logic, eligibility and business transitions. No miniapp-owned datasource, parallel authorization path, or duplicate effects ledger. |

### Secondary hardening

Review 1 also identifies three lower-severity issues worth retaining in the hardening backlog: (a) explicit tenant predicates on invocation/effect mutations even where RLS exists; (b) surfacing stale status when the executor cannot be queried; and (c) canonicalizing JSON before intent hashing so insignificant whitespace/key-order differences do not invalidate approvals. These should be verified against the current tree and fixed without expanding the Class D scope unnecessarily.

## 5. What `lead-followup` should prove as a fixture

The fixture should be deliberately small in business semantics but non-trivial in framework behavior. It should demonstrate a representative Class D execution path using synthetic data and mock capabilities:

1. A canonical invocation creates a durable run and returns a stable handle.
2. The workflow passes through more than one dependent stage and persists meaningful progress.
3. A durable wait survives process termination and restart.
4. A signal/event can wake the workflow; duplicate or late signals have defined behavior.
5. Eligibility or a configurable predicate is rechecked before the next consequential mock effect.
6. Cancellation prevents future effect attempts after the cancellation decision is durable; an already in-flight effect is settled as confirmed, failed, or unknown rather than misreported as cancelled.
7. The mock effect uses the shared effects ledger, with idempotency and UNKNOWN behavior tested under crash and concurrency.
8. The run remains attributable to the agent, authority, tenant, invocation, and effect throughout its lifecycle.
9. A version-transition test demonstrates the documented old-run policy.
10. The same framework contract works through the canonical service; REST, MCP, and CLI remain thin adapters, and the selected OpenCode demonstration exercises success, refusal, and durable submission.

The fixture does **not** need a realistic sales playbook, real lead records, real provider integration, quiet-hour law, multiple channels, production escalation, or production exactly-once delivery. Those would make the example bigger without necessarily making the framework proof stronger.

A useful acceptance question is: **Could the same framework contracts support a future workflow with more stages, longer waits, external events, multiple human decisions, and several consequential effects without bypassing or rewriting the framework?** The fixture and generic probes should provide evidence for that answer.

## 6. Framework capability matrix

| Capability | Framework responsibility | Minimum Phase 1 evidence |
|---|---|---|
| Canonical invocation | One service owns validation, auth, policy, idempotency, dispatch, results/errors, and audit; transports cannot bypass it | Contract tests across REST/MCP/CLI adapters and direct service; refusal before dispatch |
| Stable Class D handle | Durable mapping from invocation to a discoverable run; safe replay after partial failure | Submit/record fault injection; duplicate submission; lifecycle lookup after restart |
| Durable orchestration | Durable waits, stage progress, signals, cancellation, recovery | Kill/restart during wait and signal; prove continuation from checkpoint |
| Version-safe recovery | Per-family identity, managed application version, old-code retention/drain or compatible recovery | Old run pending during version transition; unknown version is visible/actionable |
| Approval lifecycle | Durable decision record, signal/resume, expiry, rejection, cancellation, separation of duties | Decision-signal crash window; duplicate/late decision; reject/expire/cancel race |
| External effects | Shared effect-state machine, idempotency, no blind retry of UNKNOWN, audit | Concurrent attempt test; crash at intent/send/outcome boundaries; UNKNOWN quarantine |
| Tenant and entity safety | Trusted tenant context, RLS plus explicit tenant scoping, declared entity serialization | Same entity ID across tenants; same-entity concurrency; different entities parallel |
| Policy and safety controls | Effect-time authorization/revalidation, quotas, blast-radius bounds, kill switch | Revoke authority and trip kill switch while a run is waiting; no subsequent effect begins |
| Determinism | Pinned-SDK-aware analyzer and deterministic workflow design | Seven rule families, positive/negative fixtures, strict CI gate, replay test |
| Operability | Inspectability, status freshness, retry limits, terminal failure, retention, alerting, recovery owner | Operator-facing failure drills and documented runbooks/limits |

This matrix is the framework assessment. The `lead-followup` fixture is one vehicle for exercising it; it is not the benchmark for whether all future DBOS workloads should resemble sales follow-up.

## 7. Test and evidence plan

The merged test requirements below are the minimum set to make the code findings falsifiable. They preserve the increment 5 hard stop by using generic substrate probes until the ADR is approved.

| Test group | Adversarial scenario | Pass condition | Evidence environment |
|---|---|---|---|
| Invocation/run binding | DBOS submit succeeds; invocation `run_id` persistence fails; same-key retry | No orphan run; retry returns a valid handle or a compensated terminal state with audit | Unit fault injection + real PostgreSQL integration |
| Idempotency | Same key/same input; same key/different input; different keys/same entity | Exact replay for identical request; explicit key-reuse error for mutated input; entity conflict follows declared policy | Real PostgreSQL, concurrent barriers |
| Entity and tenant isolation | Same entity ID in two tenants; two requests to same entity; two different entities | No cross-tenant attach/read; one active sequence per declared entity where required; different entities can proceed in parallel | Real PostgreSQL, non-owner RLS role and privileged-role tests |
| Effects | Concurrent duplicate attempt; crash before send; crash after provider acceptance but before confirm; ambiguous timeout | State transitions are serialized; UNKNOWN is quarantined; no blind retry; no duplicate confirmed effects | Real PostgreSQL + deterministic mock capability + process kill |
| Approval lifecycle | Approve/reject/expire/cancel races; crash after decision commit before signal; duplicate decision | Decision is authoritative and idempotent; workflow resumes or terminates; live run is cancelled on terminal rejection; states converge | Real PostgreSQL + controlled restart |
| Durable waits/events | Kill process during wait; restart; deliver signal before and after wait; duplicate signal | Wait and signal survive/reconcile according to documented semantics; workflow makes progress once | Pinned DBOS v1.9.0 + real PostgreSQL |
| Versioning | Old run pending while new application version is deployed; unknown version | Compatible old run resumes or drains; unsupported version is visible and recoverable; artifacts are retained | Two builds/process versions + real PostgreSQL |
| Determinism | Seven banned/unsafe construct families in positive and negative fixtures | Analyzer flags each prohibited fixture, accepts valid code, and runs in CI strict mode | Unit/static CI |
| Attribution and policy | Agent/authority separation, revoked authority, kill switch, audit reconstruction | Trusted tenant/authority are used; no effect after policy stop; every run/effect/approval is attributable | Integration tests + OpenCode transcript |
| Operations | Failed workflow, stuck wait, expired approval, stale status, retry exhaustion | Operator can inspect, distinguish stale state, intervene, and recover under documented policy | Runbook drill and evidence register |

For evidence integrity, each run must record the tested source revision or working-tree digest, environment, database identity/class (never credentials), exact command, test names, result, skips, and failure disposition. Keep local PostgreSQL, Neon, OpenCode, and unit evidence separate. The successful local-PostgreSQL run reported for increments 4–5 should be entered as its own evidence item; it should not be conflated with Review 1's independent run where database tests were skipped for lack of a configured database.

## 8. Recommended ADR reframe

The ADR should stop presenting `lead-followup` as a fully specified production sales process or using an unconfirmed 12/14 business score to argue for DBOS.

Recommended framing:

> **Decision type: PILOT / Class D framework conformance fixture.** `lead-followup` is selected by `plan4.2` as a forcing fixture to exercise durable waits, eligibility re-checks, event signaling, cancellation, recovery, version-safe execution, approval integration where applicable, and shared effect handling. Its sequence values and business rules are synthetic test semantics approved only for this pilot. They are not a production sales playbook and do not establish a business case for DBOS adoption by a sales organization.
>
> The pilot's success criterion is whether the framework exposes and enforces the required Class D contracts through its canonical invocation path, with real PostgreSQL, deterministic mock capabilities, adversarial restart/concurrency/version tests, and attributable audit evidence. Any production workload's DBOS adoption decision requires a separate workload-specific ADR based on confirmed business requirements and a fair comparison with simpler alternatives.

Correct the external-effect wording to match `plan4.2`:

> The framework records effect intent and attempts through the shared ledger. Confirmed, failed, and unknown outcomes are represented explicitly. An unknown external outcome is quarantined and is not blindly retried. Database checkpoint atomicity does not imply atomicity with an external provider, and the framework does not promise exactly-once recipient delivery. Phase 1 uses mock capabilities only; production reconciliation and real writes remain gated by the later phase requirements.

Correct the identity wording:

> Invocation identity, run/workflow identity, business entity identity, and effect identity are distinct. The miniapp declares its tenant-scoped `entity_key`; the framework provides and tests the relevant entity-serialization contract. A DBOS workflow ID is not itself a lock on a business entity. Any sequence or generation semantics belong to the miniapp domain and must be stated by the fixture, not inferred from the workflow ID.

Keep reviewer and approval fields blank until the authorized human reviews and signs the revised pilot ADR.

## 9. Decisions and scope boundaries

### Required owner decisions for the pilot

1. Confirm that `lead-followup` is being used as a **synthetic Phase 1 Class D forcing fixture**, not an approved production sales playbook.
2. Approve the minimum test-only sequence semantics required for the fixture: stage progression, wait behavior, signal/event behavior, cancellation behavior, and whether the fixture exercises approval parking. Mark each value as synthetic and non-production.
3. Name the human ADR reviewer required by `plan4.2` and explicitly approve or defer the revised ADR.

The owner does **not** need to invent production values for consent, quiet hours, real channel order, escalation, or campaign policy to approve a synthetic framework pilot. Those remain required only if a real production sales process is later proposed.

### Keep out of Phase 1

- Real providers, OAuth, and real external writes.
- Production send reconciliation worker (Phase 4); Phase 1 must still record/quarantine UNKNOWN correctly and never blind-retry it.
- River/Class B or Class C substrate work (Phase 3).
- Sprites, workspaces, promotion, arbitrary user code, UI generation, or a general workflow engine.
- Production sales-policy claims not supported by an approved playbook.
- Any claim that the fixture proves all DBOS workloads should use DBOS.

### Hard stop

This consolidated review grants no implementation approval. Do not create, register, or execute `lead-followup` as a DBOS workflow until the authorized human approves the revised pilot ADR. Generic framework probes may proceed only within the previously authorized Phase 1 scope and without introducing `lead-followup` business semantics.

## 10. Final verdict

**Framework architecture: promising and aligned with the stated Class D objective.** The framework has the right conceptual boundaries: canonical invocation, a DBOS executor, shared effects and approvals, trusted principals, tenant isolation, audit, and lifecycle APIs.

**Framework implementation: not yet proven Class D-ready.** The source-inspected findings around orphan workflow submission, approval/run lifecycle coupling, effect-ledger concurrency, identity and tenant scope, and DBOS family/version policy are material. Durable-sleep restart/scale behavior, effect-time policy enforcement, analyzer coverage, and operator readiness still need explicit evidence.

**`lead-followup`: retain it as the planned forcing fixture.** Do not reject it because it may be simpler than a real-world durable business process. Make it simple in domain semantics and rigorous in the framework behaviors it exercises.

**DBOS adoption for future production workloads: remain workload-specific.** A genuinely DBOS-warranted workload should be judged by its actual orchestration requirements, operational consequences, and alternatives—not by whether this particular fixture is an elaborate business process.

The next work should therefore be driven by the framework capability matrix and the consolidated test plan, not by making the sales example artificially complex.

---

## Source review provenance

- `adv_review_prompt11.md` (Review 1): direct source/code inspection reported for runtime, DBOS executor, approvals, effects, migrations, and tests; independent DB-test execution was limited by the review environment having no configured `TEST_DATABASE_URL` or Docker.
- `adv_review2_prompt11.md` (Review 2, revision 2): read `plan4.2` sections listed in its scope and the ADR/DBOS guidance; explicitly did not inspect Phase 1 source code or run tests. Its source-derived code assertions are therefore treated as unverified requirements, not independent source findings.
- Original ADR: `docs/adr/miniapps/lead-followup-adr001.md`, DRAFT and NOT APPROVED.
- No source code, tests, manifests, ADR, or repository history were modified in creating this synthesis. This is a standalone review artifact only.
