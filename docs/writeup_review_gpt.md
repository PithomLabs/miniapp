# Writing Class D Miniapps: Complexity, Failure Modes, and Framework Mitigations

**Audience:** Developers building miniapps on the framework, and operations teams running them.
**Purpose:** A candid account of why Class D workflows are genuinely hard, what specifically makes them hard, and how this framework removes a bounded subset of that difficulty while leaving the rest as testable, well-defined work.
**Grounding:** The concrete failure modes found during the Class D framework's own adversarial review (X-01 through X-24), the mitigations specified by the invocation lifecycle contract, and the discipline established by the prompt-driven review cycles.

**Implementation-status note:** This document describes the accepted Class D target contract, not a claim that every capability below is already shipped. P2 implementation is in progress. Treat a capability as available in a running framework only after its implementation, tests, and applicable P2 exit evidence have been accepted. Operator procedures described here are operating requirements, not claims that the framework performs them automatically.

---

## Executive summary

**Class D miniapps are durable business workflows** — multi-day, multi-stage processes with waits, approvals, external effects, and requirements that progress survive process death, deployment, and arbitrary time gaps. They are not "async with retries." The gap between the surface ("submit a workflow, poll status") and the depth (checkpoint recovery, effect reconciliation, version drift, cancellation ordering, tenant isolation, approval durability) is where most defects live.

**The framework is designed to remove three categories of complexity:** state-machine plumbing (predicated transitions, retry-as-re-entry, lock ordering), effect lifecycle (`PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN`, with provisional withdrawal to `CANCELLED` and evidence-based handling of `NOT_APPLIED`), and lifecycle orchestration (lost-signal recovery, approval-expiry observation, settlement discovery, and foreign-version visibility).

**The framework does not remove — and cannot remove — five categories of work** that remain the miniapp developer's responsibility: business semantics, fixtures, diagnostics, deployment discipline, and operational readiness.

**The mindset shift** is from "does it work" to "how does it fail." A Class D developer's job is to ensure every failure path terminates in a defined, observable, recoverable state. The framework provides the vocabulary and the primitives; the developer provides the enumeration of what "defined" means for their workflow.

**The honest bar:** Class D is genuinely hard. The framework makes the hard parts bounded and testable rather than open-ended and uncertain. The remaining work is real work — but it is well-defined work with concrete acceptance criteria.

---

## Part 1: What makes Class D genuinely hard

### 1.1 The surface vs. the depth

A developer's first encounter with Class D reads like this:

```go
handle, err := framework.Invoke(ctx, Invocation{
    Tool:           "hr.onboard_employee",
    Input:          input,
    IdempotencyKey: "tenant:" + tenant + ":employee:" + empID + ":onboard",
})
// Poll or signal to observe progress
status, _ := framework.Status(ctx, handle.RunID)
```

That is the surface. The depth is everything the framework is doing to make that surface honest:

- The invocation was claimed idempotently against a unique key.
- The workflow was submitted to a specific DBOS family with a specific application version.
- The run ID was adopted (not overwritten) if a concurrent duplicate raced.
- The approval requirement was resolved from the manifest, not the caller's flags.
- The submission passed a choke point that is the only path to `Async.Submit`.
- The workflow will checkpoint its stages; a crash mid-run resumes from the last completed step.
- External effects will be recorded as intents in a ledger before any provider call.
- The ledger's state transitions are predicated on expected prior state and use affected-row checks.
- If the outcome of a send is ambiguous, it will be quarantined as `UNKNOWN`, not blindly retried.
- If the workflow waits for an approval, the approval table is the source of truth; the DBOS signal is a hint.
- If a signal is lost in a crash, the sweeper is designed to redeliver it within the configured interval while a healthy sweeper and its PostgreSQL/DBOS dependencies are available and the backlog remains within processing capacity.
- If an approval expires during a long wait, the sweeper is designed to observe the expiry and settle the invocation, subject to the same health, dependency, and backlog conditions.
- If the workflow completes and nobody polls, the sweeper is designed to discover and record settlement independently of polling, subject to those conditions.
- Cancellation is durable-first; an in-flight attempt is reported honestly, not hidden.
- The invocation row has a `run_id` unique constraint when nonempty.
- Attached invocations (losers of a race) follow the owning workflow's outcome; cancelling an attached invocation cancels only that invocation, preserves `attached_run_id`, and leaves the shared target and other observers untouched.
- Foreign versions' pending workflows are visible to operators; dead-lettered workflows are quarantined, not failed.
- Shutdown failure preserves the pool rather than silently closing it under active workers.

A developer who sees only the surface and assumes "async with retries" may build a miniapp that passes its happy-path test but fails on the first process kill, version change, or concurrent duplicate submit. The target framework contract is designed to make such failures observable, diagnosable, and traceable; it does not make an incorrectly designed business workflow safe by itself.

### 1.2 The five failure modes we found (and they are not exotic)

The framework's own adversarial review surfaced five HIGH findings. All five were in code that had passed its test suite. All five were on failure paths the suite did not cover. None was exotic — each is the "obvious" mistake a competent developer makes when they focus on the happy path.

**X-01 — Authorization by path ordering.** The code checked "is this invocation gated?" in one function and "submit the workflow" in another. A retry after a partial failure entered the submit function without going back through the gate. The workflow skipped its approval wait and executed consequential effects unapproved. The bug required no attacker, only a transient error in a normal path.

**X-02 — Failure branches not modeled as states.** A failed submit-after-park left the invocation in `WaitingApproval` naming a run that was never submitted. Polls waited forever. A later grant failed forever. A denial looped forever. The "failure" was not a state; it was a gap between states.

**X-03 — Non-predicated writes.** Two concurrent same-key submits both saw `run_id == ""`, both minted a run ID, both wrote (last writer wins). One workflow was orphaned — live, sending, unaddressable. The `UPDATE ... WHERE id = ?` had no `WHERE run_id = ''` predicate.

**X-04 — Read-modify-write without locks.** Two concurrent `Decide` calls on the same approval both read `pending`, both passed the state check, both wrote (last writer wins). An approve and a reject could both "succeed." A self-approval could land through the race.

**X-05 — Check-then-act across boundaries.** `CheckLive` acquired a lock, read the invocation, and released. The effect claim INSERT happened in a different transaction, with no liveness predicate. A cancel committing in the gap did not stop the new attempt.

**These are the five failure modes developers should expect to write.** They are not signs of incompetence. They are the natural consequences of building a stateful system one happy path at a time. The framework's job is to make each of them either impossible (because the primitive doesn't allow it) or caught (because the test suite would fail).

### 1.3 Why "async with retries" is a trap

The naive mental model — "submit a job, retry if it fails" — fails on five specific challenges that Class D workflows face by construction:

**1. Cancellation.** In "async with retries," cancel means "stop scheduling new attempts." In a Class D workflow, cancel means "stop the current attempt if possible, report what was in flight, settle pending stages, notify attached observers, and do not start new attempts." The ordering matters: durable intent must be recorded *before* any stop is attempted, or a crash between "stop requested" and "stop recorded" leaves the workflow running with no way to observe that a cancel was intended.

**2. Approval.** In "async with retries," approval is "wait for a callback." In a Class D workflow, approval must survive: the callback crashing before it signals, the approver going offline for a day, the approval expiring while the approver is unreachable, the workflow's process being restarted with the approval decision already committed but unsignalled, and the approval's material inputs changing between request and grant. The approval table is not a callback — it is a durable source of truth that must be re-read at decision points and re-checked immediately before execution.

**3. Version changes.** In "async with retries," version changes are "don't change the code." In a Class D workflow, version changes are unavoidable: the workflow may run for days, during which the code changes. Old workflow instances must resume on compatible old code (or be deliberately patched to new code), and new instances must start on new code. If the workflow's checkpointed state is incompatible with the new code's replay logic, the instance can fail or behave incorrectly. The framework's per-family `AppName` and pinned `ApplicationVersion` provide affinity and visibility; deployment policy must retain compatible old artifacts until pending workflows drain or are deliberately migrated.

**4. Partial failure.** In "async with retries," partial failure is "retry from the start." In a Class D workflow, partial failure is the normal case: three stages completed, the fourth failed, the fifth and sixth never ran. Retrying from the start repeats the first three, which may produce duplicate external effects. The framework's effect ledger and checkpointed stages support safe recovery; any business-specific compensating actions must still be designed and implemented by the miniapp.

**5. Coordination.** In "async with retries," coordination is "don't run two workers." In a Class D workflow, coordination is unavoidable: the same lead, invoice, or employee can receive concurrent invocations from agents, retries, and background processes. The framework's entity key, partitioned queues, and predicated adoption exist because "the caller won't call twice" is not a guarantee.

---

## Part 2: The cognitive load

### 2.1 What the developer has to hold in their head

A Class D miniapp developer works with:

- **States** — the enumerable set for each subsystem (invocation, approval, effect, workflow, sequence, cancellation)
- **Transitions** — source state, destination state, predicate, effect, audit requirement
- **Invariants** — properties that must hold in every reachable state
- **Terminal states** — states from which no further transition is permitted
- **Lock order** — the total order in which any multi-row transaction acquires locks
- **Failure modes** — what happens at every transition boundary when the operation fails
- **Cancellation races** — for each check-then-act sequence, what happens if cancel lands between
- **Version compatibility** — what happens to pending workflows when code changes
- **Idempotency identity** — the stable key per business operation that makes retries safe

This is not a lot of vocabulary by Class D standards — DBOS, Temporal, and Camunda all require similar breadth. But it is a lot of vocabulary for a developer who has primarily written Class A/B code, where the failure modes are "the transaction rolled back" and "the request returned an error."

**The cognitive load is not the number of concepts.** It is the number of *interactions* between them. Predicates interact with lock order. Idempotency interacts with retry. Cancellation interacts with every pre-send transition. Version changes interact with every checkpoint. The developer must reason about pairs, not just concepts.

### 2.2 The five identity distinctions

The single most common source of Class D bugs is conflating identity scopes. There are five distinct identities in a well-designed workflow, and conflating any two of them produces a specific class of defect:

| Identity | What it identifies | Conflating produces |
|---|---|---|
| **Invocation identity** | One call from one caller with one idempotency key | Duplicate submits from the same key produce two workflows |
| **Workflow identity** | One DBOS workflow instance (the `run_id`) | Attached invocations orphan the workflow |
| **Entity identity** | One business object (`tenant_id:lead_id`) | Cross-tenant leads collide |
| **Sequence identity** | One generation of an entity's workflow | Superseded generations send duplicate touches |
| **Effect identity** | One external side effect (`tenant:invoice:123:post`) | Duplicate provider sends |

Every pair of these has a specific collision mode. The framework's design uses:
- **Invocation identity** as the unique key on the invocation row (idempotent claims)
- **Workflow identity** as `run_id`, separate from `attached_run_id` (attached invocations follow the owner's run, never overwrite it)
- **Entity identity** as `tenant_id:lead_id`, declared in the manifest, enforced by admission, and used as the partition key for the DBOS queue that serializes per-entity work
- **Sequence identity** as a partial unique generation record, with a fence checked at the authorization boundary
- **Effect identity** as `tenant:business_op_id`, with a global PK and a prefix gate that fails closed on cross-tenant collisions

**A Class D developer must know which identity scope applies to which operation.** "Delete this lead" is an entity operation. "Send this specific touch" is a sequence-and-effect operation. "Submit a workflow that advances this sequence" is a workflow-and-sequence-and-entity operation. Getting the scope wrong is not a typo; it is a correctness bug that manifests only under concurrency or retry.

### 2.3 The mindset shift

Class A/B developers ask: "Does this work?" Class D developers must ask: **"How does this fail?"**

The mindset shift has three components:

**From imperative to state machine.** A Class A function is a sequence of statements. A Class D workflow is a state machine where every transition has a predicate, an effect, an audit requirement, and a failure mode. The developer's mental model shifts from "what the code does" to "which states are reachable and how do we get from one to the next."

**From happy path to boundaries.** A Class A test checks that the function returns the right answer. A Class D test checks that the function returns the right answer, that a crash at each boundary leaves the system in a defined state, and that a retry after the crash converges correctly. The test suite is larger, slower, and tests things the happy path never sees.

**From "the framework handles it" to "the framework gives me the tools."** The framework removes state-machine plumbing, effect lifecycle, and lifecycle orchestration. It does not remove the developer's responsibility for business semantics, fixtures, diagnostics, deployment discipline, and operational readiness. The boundary matters because the developer who assumes the framework does everything will build a workflow that the framework supports but does not govern.

---

## Part 3: The framework's specific mitigations

### 3.1 State machine formalization

**What the framework gives you:** Six explicit state machines — invocation, approval, effect, workflow/run, sequence, cancellation — each with enumerated states, defined transitions, predicates, and terminal states. Each transition is implemented by a *subsystem-specific* operation, not by scattered ad-hoc `UPDATE` statements.

**What the developer still owns:** The business semantics of the workflow stages. The framework knows that an invocation is `running` or `succeeded`; it does not know that "eligibility re-check must run before each touch" or "escalation triggers after 3 failed touches." The developer declares these as DBOS steps; the framework checkpoints them.

**Why it matters:** Without formalization, every developer reinvents the state machine per workflow, and the state machines diverge. With formalization, every workflow uses the same primitives and inherits the same failure-path coverage.

### 3.2 Predicated transitions

**What the framework is designed to provide:** Every contested lifecycle mutation proves its expected prior state through a predicated `UPDATE` with an affected-row check or an explicit row-lock protocol. On a lost race, the loser reloads and reconciles or returns a defined conflict. The `run_id` is *adopted*, never overwritten.

**What this closes:** The predicated-transition and row-lock protocols prevent the original X-03 (concurrent submits orphan workflows) and X-04 (concurrent decisions last-writer-win) failure shapes when every relevant path uses the protocol. Two concurrent submits either have one winner whose `run_id` the other adopts, or serialize and reconcile against persisted state.

**What the developer still owns:** Knowing which state is expected. If the developer writes `UPDATE ... WHERE id = ?` without a state predicate, the primitive doesn't help — the discipline is on the developer.

**The concrete pattern:**

```go
// Wrong
result, _ := db.Exec(ctx, `UPDATE invocations SET run_id = $1 WHERE id = $2`, runID, invID)

// Right
result, _ := db.Exec(ctx, `UPDATE invocations SET run_id = $1 WHERE id = $2 AND run_id = '' AND attached_run_id IS NULL`, runID, invID)
if result.RowsAffected() == 0 {
    current, _ := loadInvocation(ctx, invID)
    if current.RunID != "" {
        // Adopt the owning invocation's canonical run ID; never overwrite it.
        return Handle{RunID: current.RunID}, nil
    }
    if current.AttachedRunID != nil {
        // An attached invocation follows its target; it does not acquire run_id.
        return Handle{RunID: *current.AttachedRunID}, nil
    }
    return ConflictError{}
}
```

Every contested lifecycle transition in the framework's subsystems must follow this pattern or its documented row-lock equivalent. This does not mean every insert or append-only audit write is an `UPDATE`. When a developer adds an ad-hoc lifecycle mutation, the review discipline requires a contract row that specifies its predicate, lock scope, and failure behavior.

### 3.3 Retry-as-re-entry

**What the framework gives you:** Every retryable path loads persisted state and dispatches on observed state. There is no "is this a retry" flag that changes control flow. A retry enters the same state machine, reads the same row, and dispatches to the same handler that would have run if the operation had succeeded the first time.

**What this closes:** The retry-as-re-entry protocol, combined with the submission choke point and persisted approval checks, prevents the original X-01 (retry bypasses approval) and X-02 (failed submit strands the invocation) failure shapes. This depends on every submission path passing through the enforced choke point and every retry dispatching from persisted state.

**The concrete pattern:**

```go
func (s *service) submitInvocation(ctx context.Context, req InvokeRequest) (Handle, error) {
    record := loadInvocation(ctx, req.idempotencyKey)
    switch record.state {
    case StatePending:
        return s.dispatchFirstClaim(ctx, record, req)
    case StateWaitingApproval:
        return s.resumeApprovalPoll(ctx, record)   // submits if run missing
    case StateRunning:
        return s.chokePointSubmit(ctx, record, req)  // requires approval linkage
    case StateTerminal:
        return record.result, nil
    }
}
```

**What the developer still owns:** The business handlers that execute per state. If the developer adds a new state ("escalated") to their workflow stages, the handlers must be defined for retry-as-re-entry.

### 3.4 The lock order and ATTEMPTING as point of no return

**What the accepted contract specifies:** A global lock acquisition order — invocation → approval → sequence → policy-control → effect → advisory-lock — that every multi-row transaction must respect. The order is designed to prevent lock cycles; deadlock-freedom is not established by the list alone. Every acquisition path must be audited against it and exercised by concurrency tests.

**What the ATTEMPTING rework changes:** The prior design treated `PENDING → ATTEMPTING` as "claimed but sendable after." The rework makes `ATTEMPTING` the durable point of no return, with full revalidation (liveness, approval, grants, token, switches, eligibility, generation, lease) executed *inside* the transition transaction and the send strictly after commit.

**What this closes:** X-05 (check-then-act across boundaries) by making authorization and the durable `PENDING → ATTEMPTING` transition one transaction. The provider send still occurs after commit, so cancellation cannot recall an attempt whose authorization already committed. The enforceable invariant is "no new ATTEMPTING transition after durable cancel," not "no provider send can still be in flight after durable cancel."

**What the developer still owns:** Choosing the effect identity. If the developer uses an unstable effect identity (a UUID generated per attempt), predication cannot prevent duplicate sends. The identity must be stable across retries.

### 3.5 Submission choke point

**What the accepted contract requires:** One enforced choke point, `submitChecked`, is the sole path to `Async.Submit`. Every executor submission must pass through it. The choke point resolves the tool's approval requirement from the persisted manifest (never from caller flags), checks for a valid approval linkage, and refuses gated tools without one.

**What this closes:** The class of authorization-by-path-ordering bugs, provided the caller-set audit continues to show that every non-test `Async.Submit` call routes through the choke point. Multiple service paths may call the choke point; none may bypass it.

**What the developer still owns:** The manifest. If the developer declares a tool as ungated when it should be gated, the choke point will correctly submit it. The manifest is the source of truth for gating; the developer is responsible for declaring it correctly.

**How to verify in review:** A grep test confirms the non-test caller set of `Async.Submit` matches the audited set. Any new caller is a review violation.

### 3.6 The effect ledger and UNKNOWN quarantine

**What the accepted contract specifies:** Every external effect is recorded as an intent before any provider call. A provisional `PENDING` reservation advances to `ATTEMPTING` only after full authorization; cancellation may withdraw a still-pending reservation to `CANCELLED`. An attempt may settle as `CONFIRMED`, `FAILED`, `UNKNOWN`, or `NOT_APPLIED`. A timeout after possible provider acceptance is `UNKNOWN`, not `FAILED`; blind retry is prohibited. `NOT_APPLIED` may be retried only through a fresh authorization. Settlement requires provider-side evidence or an authorized operator decision with evidence. Phase 1 does not include an automatic effects-reconciliation sweep. Refusals (kill switch, revoked grant, expired token) are recorded as `effect-refused` with attribution.

**What this closes:** The class of defects where an ambiguous failure is retried, producing duplicate sends, or where a silent refusal makes "never attempted" indistinguishable from "blocked by policy."

**What the developer still owns:** Provider idempotency. If the provider supports idempotency keys, the developer should use them. If not, the developer must specify a reconcile-by-read method. The framework cannot tell the developer which method applies to their specific provider; it can only enforce that the developer specified one.

### 3.7 Cancellation semantics

**What the framework gives you:** Durable intent first. Cancel is recorded in the invocation row before any stop is attempted. Pending approvals settle. `PENDING` effects move to `CANCELLED` (withdrawn before authorization, never attempted). `ATTEMPTING` effects are reported as in-flight and settle from evidence. Cancellation of an attached invocation is DETACH-ONLY: the attached row cancels, the shared target is untouched, other observers are unaffected, `attached_run_id` is preserved immutably.

**What this closes:** The class of defects where cancel silently fails to stop a new attempt, or where an attached invocation accidentally cancels a shared workflow that other observers depend on.

**What the developer still owns:** The cancel authority policy. Who can cancel what? The framework enforces tenant ownership and authorization; the business policy on which roles can cancel which workflows is in the manifest.

### 3.8 Approval bridge

**What the accepted contract specifies:** The approval table is the durable source of truth. The DBOS signal is a hint that the decision may have changed. Decisions are audited at commit, not after signal. The approval binds to the admitted miniapp version, tenant, authority, input material hash, and expiry. An out-of-band approver channel is the control; MCP elicitation is convenience, not the control. A healthy sweeper redelivers lost signals, observes expiry, and settles invocations independently of polling, subject to dependency availability and backlog capacity.

**What this closes:** The class of defects where a decision is committed but unsignalled (and thus unobserved), where an approval expires during a long wait (and the workflow parks forever), or where a self-approval can land through a race.

**What the developer still owns:** The approval policy. Who approves what? Is it a single approver or dual approval? What happens on rejection? What happens on expiry? The framework provides the mechanism; the business rules are in the manifest.

### 3.9 Version safety

**What the accepted contract specifies:** Per-family `AppName` and explicit `ApplicationVersion`, with SDK version affinity so v2 workers do not execute v1 workflows. Foreign-version visibility is surfaced on startup and via periodic checks. Dead-letter workflows are represented as quarantined (nonterminal and operator-actionable) rather than silently failed. Operators must retain compatible old artifacts until pending workflows drain or are deliberately migrated; the framework does not by itself guarantee artifact retention.

**What this closes:** The class of defects where a version change silently strands or corrupts pending workflows, or where dead-lettered workflows are silently marked failed when they could have been resumed.

**What the developer still owns:** The version strategy. Does a workflow code change require a version bump? Is it a compatible patch or an incompatible change? The framework provides the mechanism; the developer makes the call.

### 3.10 Grant and policy enforcement

**What the accepted contract specifies:** In-process grant revocation is synchronized by a pinned-snapshot protocol: an `RWMutex` in `auth.Policy`, `RLock` held across the authorization read set, and `Lock` for the setter. This closes the in-process race. Cross-replica coherence remains a deployment residual. The P2 exit requirements call for a rolling-restart procedure, bounded staleness window, config-version visibility, and a verification command; do not describe those surfaces as available until implemented and verified.

**What this closes:** The in-process race between revocation and authorization. The cross-replica window is explicitly named as a deployment residual, not silently accepted.

**What the developer still owns:** The deployment discipline. The operator must verify revocation completed, bound the staleness window, and have a rollback criterion. The framework provides the surface; the operator uses it.

### 3.11 The sweeper

**What the accepted contract specifies:** A framework worker that runs at startup, on a tick, and opportunistically on `Status`/poll. Three operations: S1 lost-signal redelivery, S2 expiry observation, S3 settlement discovery. The target is observation within the configured sweep interval while at least one healthy sweeper and the required PostgreSQL/DBOS dependencies are available and the batch processor can keep up with the backlog. Outages, failures, or excess backlog delay convergence and must be visible to operators.

**What this closes:** The class of defects where a decision is committed but never observed, where expiry is never marked, or where an unpolled workflow's settlement is never recorded.

**What the developer still owns:** Nothing for the sweeper itself (it is framework-owned). But the developer must not construct workflows whose guarantees *depend* on polling — the sweeper means polling is optional for correctness, not just convenience.

---

## Part 4: What the developer still owns

The framework removes three categories of complexity. Five categories remain the developer's responsibility. These are not "nice to have" — they are the boundary between a demo and a production system.

### 4.1 The business logic

The framework does not know that "a lead is eligible for follow-up if they have not unsubscribed, have not converted, and have not received a touch in the last 48 hours." The developer does.

**What the developer must specify:**
- Eligibility predicates (re-checked at every stage, not just at submit time)
- Stage logic (what each stage does, what it produces, what it depends on)
- Exit conditions (what terminates the workflow, what happens to in-flight work)
- Escalation rules (when is a human needed, who, via what channel)
- Approval policy (which stages require approval, from whom, with what expiry)

**How to make this testable:** Each predicate becomes a fixture. Each stage becomes a DBOS step with a crash boundary. Each exit condition becomes a test that hits the exit path and asserts the correct terminal state.

### 4.2 The fixtures

The framework's red/green discipline means every HIGH finding must ship with a discriminating test that fails pre-remediation and passes post. The developer inherits this discipline for their own workflows.

**What the developer must specify:**
- Fixtures for every stage boundary (input → expected output, with the DBOS step boundaries)
- Fixtures for every failure path (what happens when the DBOS submit fails, when the signal send fails, when the provider returns UNKNOWN)
- Fixtures for every concurrency race (barrier-synced, not sequential)

**How to make this tractable:** The `FaultInjector` is the tool. It fails a named operation on its Nth attempt. The developer configures it per fixture; the test asserts the failure path converges to the expected terminal state.

### 4.3 The diagnostics

The framework records events, but it does not decide which events matter for the developer's workflow.

**What the developer must specify:**
- Decision-point logging (the audit record for each approval, not just "a decision happened")
- Distinguishing refusals from attempts (a kill-switch refusal is not the same as a provider attempt that failed)
- Never silent success (if the workflow reports success, the developer must verify the success is real, not assumed)

**How to make this tractable:** Each audit event maps to a fixture. The developer writes the fixture that asserts the audit record exists, contains the expected attribution, and distinguishes the event class correctly.

### 4.4 The deployment discipline

The accepted P2 target includes a grant-revocation verification surface and an associated deployment procedure. Until those surfaces are implemented and verified, the operator cannot infer fleet-wide revocation completion from a restart alone.

**What the operator must specify:**
- Config-version visibility on health checks
- Revocation verification command in the deployment runbook
- Maximum staleness window (the accepted P2 baseline is 10 minutes)
- Rollback criterion if the window is exceeded
- Restricted role enforcement in production

**How to make this tractable:** The accepted P2 exit conditions require these surfaces and procedures to be implemented and verified. Until then, a restart alone does not prove fleet-wide revocation completion.

### 4.5 The operational readiness

The framework emits status; the operator must monitor it.

**What the operator must monitor:**
- Quarantine backlog (dead-lettered workflows awaiting operator action)
- Approval backlog (pending approvals past their expected window)
- Sweeper health (a stopped sweeper regresses lost-signal recovery, expiry observation, and settlement discovery to pre-sweeper behavior)
- Foreign-version pending count (workflows that a version change has left stranded)
- Executor-down events (the sweeper and status sync compensate, but alerts matter)
- Settlement failures (an invocation that completed but was never settled is a data integrity issue)

**How to make this tractable:** For production, map every operationally significant state and failure condition to an alert, an owner, a runbook, and a verification command. Treat missing coverage as an operational-readiness gap rather than assuming the framework automatically supplies every alert.

---

## Part 5: Testing Class D properly

### 5.1 Barrier-synced concurrency, not sequential

A sequential test — "submit, wait for completion, submit again" — cannot exercise a race. The two submits happen at different times; the race requires them to happen at the same time.

**The barrier pattern:**

```go
// import "sync"
var ready, done sync.WaitGroup
ready.Add(N)
done.Add(N)
start := make(chan struct{})
for i := 0; i < N; i++ {
    go func(i int) {
        defer done.Done()
        ready.Done()
        <-start // All workers wait here until the test releases them
        invoke(ctx, request(i))
    }(i)
}
ready.Wait() // Every worker has reached the start gate
close(start)  // Release them together
done.Wait()
```

Concurrency tests should use a real synchronization barrier with separate database connections where the race requires them. Sequential tests that merely look like concurrency tests do not exercise the interleaving being claimed.

### 5.2 Fault injection at the real boundary

"Make the function return an error" is not fault injection. The failure must occur at the real boundary: the DBOS submit, the signal send, the commit of the approval decision, the provider call.

**The FaultInjector pattern:** create a test-scoped injector, then configure the named operation (for example, `dbos.Submit`) to fail on its first invocation. Exercise the real submit path, assert that the injected error propagates, and retry through the normal lifecycle path. Use the concrete configuration API exposed by the testkit version in use; the operation name and attempt number here describe the contract, not a promise about method spelling.

The injector is a test-only seam. In production, it is nil. A test asserts nil-after-`New` for every service constructor.

### 5.3 Red/green for framework remediations and reproducible regressions

When fixing an existing defect or validating a risk for which a meaningful pre-fix baseline exists, use the red/green discipline:

- **Fail** against the pre-remediation snapshot with a failure message that matches the expected failure mode.
- **Pass** against the remediated tree.
- Run the **same test source** on both sides where the test can be expressed against APIs available in both snapshots.
- Record **both digests** and the **exact command** in the evidence register.

A newly introduced miniapp feature may have no pre-remediation implementation to test. In that case, provide discriminating tests for its invariants and failure paths; do not manufacture an artificial red run. A test that passes on both sides does not discriminate when it is intended to prove an existing defect was fixed. A test that fails for a different reason than expected is testing the wrong thing.

### 5.4 Real PostgreSQL, real process kills

In-memory fakes cannot exercise transaction isolation, row-level security, or advisory locks. They also cannot exercise process-kill recovery, because the killed "process" is a goroutine, not a process.

**The kill/restart pattern:**

```go
// Submit a workflow that sleeps for 15 seconds
// Spawn a subprocess that runs the workflow worker
// SIGKILL the subprocess after 5 seconds
// Spawn a new subprocess (new process, same PostgreSQL state)
// Assert the workflow resumes from the last completed step
```

The kill/restart harness must run against real PostgreSQL and a real process. Because it is slow, a project may reserve the full harness for defined release gates and scheduled CI rather than every commit. The required cadence must be explicit in the project's CI and release policy; this paragraph is a testing recommendation, not a claim about an already-configured schedule.

### 5.5 The prohibited tests

Certain test patterns are forbidden because they produce false confidence:

- **Sequential "concurrency" tests** — they exercise the wrong code path
- **Mocks that do not measure each call** — `ON CONFLICT DO NOTHING` on a mock log means "at least one send" is the only assertion possible, not "exactly one"
- **Tests that pass on both sides of a fix** — the fix is either unnecessary or the test is not measuring the fix
- **Silent skips of required DB-backed tests in a database-enabled CI job** — the CI job must provision PostgreSQL and prove those tests ran. An explicit skip-fast path may be used in a deliberately DB-less local environment, but it must be visible and must not substitute for the database-enabled CI run
- **Tests that assert on implementation details** — "the function returned nil" is not the same as "no duplicate send occurred"; the latter is what matters

---

## Part 6: Operational awareness

### 6.1 What "stuck" looks like

The framework's states distinguish several kinds of "not progressing":

| State | Meaning | Operator action |
|---|---|---|
| `running` | Normal progress | None |
| `waiting_event` | Waiting for a signal | None unless the wait has been long (alert threshold) |
| `waiting_approval` | Waiting for a decision | None unless the approval has been long (backlog alert) |
| `quarantined` | Dead-lettered; operator action required, with reason and resume/fail guidance | Resume only after verifying the cause is resolved and the workflow version is compatible; otherwise make an explicit, audited fail decision |
| Foreign version pending | Worker version mismatch | Drain old version or patch workflows |
| `UNKNOWN` effect | Ambiguous external outcome | Reconcile by read; settle if possible |
| Sweeper stopped | Lost-signal recovery, expiry observation, settlement discovery regressed | Restart sweeper |

The operator must distinguish these. "The workflow is running" and "the workflow is waiting for a signal that will never come" look the same in a naive dashboard. They require different actions.

### 6.2 What to alert on

A production deployment should define alerts for operationally significant conditions, including:

- Quarantine backlog (dead-lettered workflows past a threshold)
- Approval backlog (pending approvals past their expected window)
- Sweeper health (last sweep timestamp; alert if stale)
- Foreign-version pending (any non-zero count, if that is the chosen policy)
- Executor down (for DBOS families)
- Settlement failures (invocations completed but not settled)
- Attached invocation orphans (attached rows whose target has disappeared)

Each alert should be actionable, identify the responsible subsystem, and link to a runbook and verification command. A stuck sweeper is not merely a "DBOS problem"; it means lost-signal recovery, expiry observation, and settlement discovery may be regressed.

### 6.3 Recovery procedures

**Quarantine resume vs. fail.** A quarantined workflow can be resumed (re-enqueued on its compatible version, resuming from its last checkpoint) or failed (settled as failed). Neither action is an automatic safe default. Resume only after verifying that the underlying cause is resolved and the workflow's code version is available and compatible; fail only through an explicit, authorized, audited operator decision.

**Lost signal recovery (automatic).** While a healthy sweeper and its dependencies are available and backlog remains within processing capacity, redelivery is targeted within the configured sweep interval. Failures and backlog must be visible; the operator should not manually re-signal unless the sweeper is unavailable or recovery is otherwise blocked.

**PITR restore with effect reconciliation.** A point-in-time restore rewinds only the application and DBOS records included in the restore scope; it does not rewind external provider effects. Do not assume the restored effect ledger contains every operation whose provider call may have happened after the restore point—the restore may have removed the corresponding intent, attempt, or audit rows. Before restarting workers: stop all workers; preserve and consult pre-restore inventories, backups, and provider-side records; identify operations whose external outcomes may no longer match restored state; reconstruct and quarantine ambiguous operations as `UNKNOWN`; reconcile by provider read or other evidence; and resume only after the discrepancies are resolved. If the candidate set cannot be established reliably, keep affected workflows stopped and escalate rather than guessing. **This procedure must be rehearsed before GA.**

**Version drain.** Before retiring an old version, verify no pending workflows of that version remain (via the foreign-version pending count). If some remain, either deliberately patch them to the new version (only if compatible and tested) or keep the old artifact and worker capability available until they complete.

### 6.4 What NOT to do

- **Manually retry UNKNOWN effects.** The UNKNOWN state exists because the outcome is genuinely unknown. Retrying can produce a duplicate. Reconcile by reading provider-side evidence and use the authorized `settle` operation with a documented evidence note. Phase 1 does not provide an automatic effects-reconciliation sweep; absent sufficient evidence, leave the operation quarantined.
- **Force-settle without evidence.** The `settle` verb requires an evidence note. "I think it probably worked" is not evidence. Read the provider's records or accept the UNKNOWN.
- **Restore a database without reconciling external effects.** A PITR restore rewinds local state but not the provider's. Any workflow that completed between the restore point and the restore time has produced real external effects that are now inconsistent with local state.
- **Skip the deployment discipline for revocation.** Cross-replica revocation requires restart coordination and the accepted P2 conditions for config-hash visibility, fleet verification, and a bounded staleness window. Until those controls are implemented and verified, do not claim that a restart proves fleet-wide revocation is complete.

---

## Part 7: The mindset

### 7.1 From "does it work" to "how does it fail"

The developer's job is not to make the workflow succeed. It is to ensure every failure path terminates in a defined, observable, recoverable state. The workflow succeeding is the trivial case. The workflow failing at step 4 of 7, after producing two external effects, during a version transition, with an approval pending — that is where the developer earns their keep.

### 7.2 From "the framework handles it" to "the framework gives me the tools"

The framework removes three categories of complexity. It does not remove the five categories the developer still owns. The boundary matters because a developer who assumes the framework handles everything will build a workflow the framework supports but does not govern — and the first incident will be in a category the framework never claimed to handle.

### 7.3 From "I'll fix it when it breaks" to "I'll test the failure before I ship"

The red/green discipline is not optional. A fix that is not tested against the pre-remediation state is a hope, not a guarantee. A test that does not fail on the pre-remediation state is testing the wrong thing. The barrier-synced test that fails for the right reason on the pre-remediation state, then passes on the remediated tree, is the minimum evidence that the failure path is closed.

### 7.4 From "operators will figure it out" to "operators have a runbook"

A production-ready deployment maps every operationally significant state and failure condition to an actionable alert, an owner, a runbook, and a verification command. This is what "production-grade" means: not "it works in the happy path," but "when it fails, the operator knows what to do and can verify it worked."

---

## Conclusion: The honest bar

Class D is genuinely hard. The framework removes a specific subset of the hardness — the state-machine plumbing, the effect lifecycle, the lifecycle orchestration — and leaves the rest as bounded, well-defined, testable work. That remaining work is:

- **Business semantics** — what the workflow actually does
- **Fixtures** — the tests that prove every failure path converges correctly
- **Diagnostics** — the audit records that make incidents reconstructable
- **Deployment discipline** — the operator-side coordination that makes revocation and version transitions safe
- **Operational readiness** — the alerts, runbooks, and procedures that make production production-grade

The framework makes the hard parts bounded rather than open-ended. The developer's job is to fill in the bounded parts with the same discipline the framework was built with: predicated transitions, retry-as-re-entry, fault-injected tests, real-PostgreSQL evidence, and the honest recording of residuals.

**The measure of a robust Class D miniapp** is not that it works. It is that when it fails — and it will — the failure is observable, diagnosable, recoverable, and attributable. That is what the framework was built for, and that is what the developer must verify their workflow inherits.

The discipline pays off in operations, not in the first version. The first version will feel slower to write than a Class A handler. The second version will feel faster than a bespoke distributed system. The third version will be a template that other developers copy. That is the trajectory the framework is designed to make possible.