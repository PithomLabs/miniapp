## **Verdict: Conditional approval — revise four points before execution**

The plan is fundamentally sound. It stays within Prompt25’s Gate 1 scope, preserves the PostgreSQL 19 evidence requirement, and correctly stops before the remaining Stage 3A work. The red/green discipline and honest distinction between discriminating tests and characterization tests are also good.&#x20;

prompt25_plan.md

prompt25_plan.md



However, **I would not approve it unchanged**. Four areas need tightening, plus one important trust-boundary check.

### **1. P24-001: Do not use 50 repetitions as the ordering proof**

The proposed FIFO staging strategy, followed by 50 repetitions and a fallback if any round misorders, is not sufficiently deterministic. Passing 50 repetitions does not establish that the intended interleaving was actually forced.&#x20;

prompt25_plan.md



Require deterministic synchronization from the outset:

- Force and verify the cancel-first ordering.
- Force and verify the authorize-first ordering.
- Use explicit barriers or observable transaction/lock milestones, not sleeps or probabilistic scheduling.
- Assert the actual order markers in each test execution.

A test-only synchronization mechanism is acceptable if the existing fault injector cannot pause execution. Do not add a production locking mechanism merely to make the test work.

### **2. P24-001: Resolve the audit-count contradiction**

The plan says both:

- Audit in the same transaction only when the withdrawal UPDATE wins.
- Repeated attempts should audit as refused/ineligible.

Those statements need separate semantics.&#x20;

prompt25_plan.md



The invariant should be:

**Exactly one&#x20;**`effect-withdrawn`**&#x20;audit event for the successful&#x20;**`PENDING → CANCELLED`**&#x20;transition.**

A repeated authorization against an already-cancelled effect must not generate another `effect-withdrawn` event. A separate refusal audit is acceptable only if it has a distinct event type and does not masquerade as another withdrawal.

The regression must assert the exact count and event types, not merely that an audit row exists.

### **3. P24-007: Fix the attempt-ID contract wording**

The plan says `claim()` will mint a fresh UUID, ignore caller-supplied IDs, and fail closed on empty IDs. Those requirements are ambiguous together.&#x20;

prompt25_plan.md



Make the distinction explicit:

- The framework-generated UUID is the authoritative attempt ID.
- A caller-provided `Intent.AttemptID` must never become the persisted attempt ID.
- An empty legacy caller field is not an error if the framework is expected to mint the ID.
- UUID generation failure or an invalid/empty generated ID must fail closed.
- Test that a caller-supplied value cannot influence the persisted ID and that successive claims receive distinct framework-generated IDs.

Retaining the field for wire compatibility is reasonable. Retaining caller authority over it is not.

### **4. P24-003: Prove the complete Settle path is lock-free with respect to invocation**

The proposed plain SELECT for immutable invocation identity is the correct intended correction. But a grep showing no `FOR UPDATE` in the Settle path is not sufficient proof.

Inspect the entire transaction, including `insertSettleAudit`, relevant foreign keys, triggers, and helper queries. A plain SELECT can be lock-free while a later write in the same transaction introduces an implicit lock on a referenced row.

The forced-order test should demonstrate that Settle never recreates the `effect → invocation` dependency, and that the §4.2 contract remains **effect lock only**.&#x20;

prompt25_plan.md



### **5. P24-005: Make manifest trust-mode validity an independent check**

The proposed approval predicate is consistent with the owner decision: gated autonomous execution can skip approval linkage only when the persisted invocation trust is Autonomous and the manifest permits Autonomous.

But do not let that predicate become the only manifest trust check.

The implementation must independently reject a persisted trust mode that the admitted manifest does not allow. The autonomous exception changes the approval-linkage requirement; it must not turn an invalid trust mode into an acceptable invocation.

The matrix should prove that an Autonomous invocation with a denied, expired, or mismatched approval cannot use the exception to bypass that approval decision.&#x20;

prompt25_plan.md



## **What to do next**

Have the implementation agent amend **the plan only** to incorporate these points. No source edits should begin until those amendments are reflected.

After that, proceed with Prompt25 in its existing order. Do not reopen the architecture or expand into the explicitly excluded Stage 3A work. At closeout, require the correction report and complete reviewer package, then run **Prompt26: targeted independent Gate 1 re-review**.

The core approach is good. The main danger is accepting evidence that appears green without actually proving the concurrency and ownership invariants. Those are precisely the properties this correction stage exists to establish.&#x20;