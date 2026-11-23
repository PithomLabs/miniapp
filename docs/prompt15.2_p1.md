## deepseek

# P1 Exit Review — Acceptance Decision for P2

## What I can and cannot verify from here

**I can verify:** that the contract addresses each of the seven conditions I raised at the prior plan review (C1–C7); that the design decisions (RC ratification, six rules, state machines, X-11 sweeper, X-08 attach, R1–R3 refinements) are internally coherent; that residuals are honestly recorded rather than hidden; that the evidence register's P0 rows are specific and reproducible; and that the self-review disclosure is correctly stated.

**I cannot verify:** the file:line evidence against RC-1a–d (I do not have the tree); the v1.9.0 SDK citations at the line level; the "no bypass found" claim in the choke-point audit; whether the 168 PASS / 0 FAIL / 5 SKIP baseline is what the sanctioned command actually produces today. Any reviewer — human or otherwise — faces the same limitation unless they open the repository.

The consequence for this decision: **my review of the contract is a review of the contract as a document, not of the contract as an accurate description of the codebase.** The accuracy of the codebase claims will be established empirically by P2's red/green tests, not by my reading.

That reframes the C2 decision. The question is not "should I accept the self-review as ground truth." It is: **is the contract strong enough as a discipline that P2's empirical tests will surface its inaccuracies, given that the contract was authored by the same agent who will implement against it?**

## Verification of the seven prior conditions

| Condition | Status | Evidence |
|---|---|---|
| **C1** — reproducible digest command | **Closed** | P0 register row contains the exact `find … sha256sum` command; twice-reproduced; delta vs prior digest is exactly +7 docs, zero source drift. Manifest preserved at a documented path. |
| **C2** — contract review authority | **At issue** | Self-review disclosed; independent review waived only by owner acceptance. This document is the acceptance decision. |
| **C3** — six normative rules | **Closed** | §2 enumerates all six (predicated single-writer, guard-and-act, authorization at execution boundary, no unreachable operational states, truthful lifecycle/audit, falsifiable evidence). Each is enforceable and referenced by the transition tables. |
| **C4** — choke-point contingency | **Closed** | §5 documents the audit result (2 + 2 call sites, no bypass found) and states the contingency: "a bypass found during P2 is fixed in place when the correction stays within this contract; escalation to the owner happens only if the fix needs new architecture or policy." This is the correct boundary — P2 is not authorized to invent architecture to cover a bypass. |
| **C5** — WS-H and WS-I one-liners | **Closed** | §13 states both precisely. WS-H: "CI provisions pinned PostgreSQL + runs the sanctioned `go test -p 1 ./...` (plus race) with DB env — validated by a local CI-equivalent run; remote execution stays unobserved, so G1 row 12 stays PARTIAL." WS-I: "every new evidence entry carries pre/post-run digests; historical undigested claims are marked irreproducible, never retrofitted." |
| **C6** — X-11 mechanism selection | **Closed** | §8 evaluates all three candidates, selects the sweeper with justification (runtime-enforced, no miniapp cooperation, no contract change), rejects the other two for stated reasons, and describes the three sweep operations (S1 lost-signal, S2 expiry observation, S3 settlement discovery) with bounded convergence. |
| **C7** — acceptance matrix reference | **Closed** | §15 contains the exit self-check table mapping each of the ten gate conditions to its P1 evidence location. |

Six of seven conditions are closed. The seventh is the decision requested.

## The specific question: accept self-review or require independent review?

I do **not** recommend requiring an independent reviewer before P2, and I recommend the owner **accept the self-review limitation with explicit recorded conditions**. The reasoning:

**1. The contract does not ask to be trusted as ground truth — it asks to be tested empirically.** §13 commits P2 to red/green discipline: every HIGH finding test must fail against the pre-remediation tree and pass against the remediated tree, with both runs digested. This is not self-attestation. It is an empirical claim that the contract makes and that reality will verify or falsify. If the file:line evidence in RC-1a–d is wrong, the X-01–X-05 tests that depend on it will fail to produce the expected red-to-green transition, and P2 will surface the inaccuracy. An independent pre-P2 review could theoretically catch this earlier, but it cannot catch it more reliably than the red/green tests will.

**2. The self-review disclosure is itself the discipline working.** The prior cycle failed because the implementation report's strong claims were not distinguished from verified claims. The P1 contract's first-page disclosure — "Authored by the P2 implementer. This is a discipline check, not an independent review" — and the evidence register's honest recording of residuals (R2 superuser limitation, OpenCode NOT RUN, parallel-red root cause traced not labeled) are exactly what the prior cycle lacked. Rejecting the self-review would punish the agent for disclosing honestly.

**3. The contract's design decisions are recorded with the discipline a later reviewer needs.** R1 (`attached_run_id`), R2 (superuser REVOKE limitation), R3 (gated tools claim as `pending`), X-11 sweeper selection, X-08 attach semantics — each is stated with alternatives considered and the reason for the choice. If the owner disagrees with any, it can be caught here, before P2 implements. If the owner agrees, the record of *why* is preserved for whatever review P2's output receives. This is the substrate an independent reviewer would use; requiring the reviewer to be present *now* rather than available *at the next gate* does not add information.

**4. Independent review of the contract would not catch the class of error most likely to matter.** The prior cycle's failure was not "the contract was wrong." It was "the implementation and the self-assessment were both authored by the same agent who was incentivized to consider the work complete." An independent pre-P2 contract reviewer would face the same limitation I face — they do not have P2's tests to check against. The reviewer at P2 *exit*, with red/green evidence in hand, can verify what the contract claims against what the tests actually exercised. That reviewer's authority must be established in the P2 acceptance criteria, not waived in exchange for a pre-P2 review that cannot do the same work.

**Therefore:** the cost of accepting the self-review is that P2 may implement a contract with inaccuracies that P2's own red/green tests do not surface because the tests are also designed against the contract. The cost of requiring independent pre-P2 review is that the review cannot verify anything the empirical tests will not verify more reliably, and it delays P2. The former cost is real and material; the latter is not.

Accept the self-review with the conditions below.

## Conditions for acceptance

The following conditions must be recorded in the P2 exit evidence. Each one closes a specific gap that self-review leaves open.

### Condition 1 — Red/green tests are the empirical verification of the contract

P2 must produce, for each of X-01, X-02, X-03, X-04, X-05, X-07 (and optionally the X-08/X-11/X-13/X-14/X-15 tests where the red side is feasible), a documented red/green transition:

- **Red:** the test fails against an isolated pre-remediation snapshot, with the failure message showing the specific failure mode the contract predicted (e.g., "X-01: gated invocation executed effects with no approval linkage" not just "test failed").
- **Green:** the test passes against the remediated tree.
- **Both runs:** bound to exact digests (the `ba359270…` baseline and the post-P2 digest), with the same database names except for isolation, and the snapshots preserved.

This is not a new condition — the contract's §13 already commits to it. The P2 exit review must verify that it happened and that the failure messages match the contract's predicted failure modes. If a red test's failure message does not match the contract, the contract is wrong and P2 must stop and report it — not adjust the test.

### Condition 2 — P2 exit requires independent review of the implementation

The self-review limitation is being waived at the contract boundary. It is **not** being waived at the implementation boundary. P2 exit must include an independent reviewer (not the P2 implementer) who:

- Reads the red/green evidence and confirms the failure modes match the contract.
- Spot-checks the transition tables against the implemented transitions (at minimum, one M1 transition, one M2, one M3, one M6).
- Spot-checks the choke-point audit by grepping the post-P2 tree for `Async.Submit` and `dbos.RunWorkflow` and confirming the caller set matches.
- Confirms the six rules are enforced in code, not just documented.

This reviewer's scope is bounded: not a full re-review, but a verification that the contract's claims survived the empirical test. It is the check that would have prevented the prior cycle's failure.

### Condition 3 — Contract inaccuracies found during P2 are surfaced, not silently reconciled

If P2 discovers that a file:line citation is wrong, a design decision does not match reality, or an assumption in the contract is false, P2 must:

- Stop the affected workstream.
- Record the inaccuracy in `docs/phase1/invocation-lifecycle-contract.md` as a corrected section with both the original claim and the corrected claim.
- If the correction changes the design (not just the citation), stop P2 entirely and return to owner decision.

The contract is authoritative for P2 as long as it is accurate. It is not a shield against discovering that it is wrong.

### Condition 4 — The three design decisions are recorded as owner-accepted rather than owner-inferred

R1 (`attached_run_id`), R2 (superuser REVOKE limitation), R3 (gated tools claim as `pending`), the X-11 sweeper as the sole enforced bound, and the X-08 genuine-attach default are all recorded as decisions in the contract. For each:

- The contract's reasoning is preserved as the acceptance rationale.
- If any of these is later found to require a different approach, the owner's acceptance is what allowed the current approach — the owner should not be surprised by a future change.

This is a recording discipline, not a decision procedure. The decisions are correct as stated; the record ensures they were made deliberately.

### Condition 5 — The X-11 sweeper is an operational component and must be documented as such

§8 says the sweeper needs "no miniapp cooperation, no decider retry, and no miniapp-contract change (no owner decision)." This is correct about *miniapp authors*. It is less clear about *operators* — the sweeper is a new framework worker with its own interval, startup behavior, and operational profile. P2 must add a line to `docs/phase1/classd-operations.md` (which the contract already mentions for the cooperative-cancel line) documenting:

- The sweeper's schedule (production: serve-loop ticker with default interval; startup: one-shot; opportunistic: bounded work on `Status`/poll).
- The three operations it performs (S1/S2/S3).
- What its failure looks like (a stopped sweeper means lost-signal recovery, expiry observation, and unpolled settlement all regress to their pre-sweeper behavior — recoverable on restart, but visible as a backlog).

This is documentation, not a new decision. It ensures the sweeper is not silently accepted as infrastructure without acknowledging it as infrastructure.

### Condition 6 — The blocked owner questions are not blocking P2

The plan's Q2 (OpenCode) is recorded as **BLOCKED by owner decision** — recorded as a residual rather than resolved. This is correct and does not block P2: G1 row 10 is PARTIAL for this cycle and will remain PARTIAL until the owner authorizes the probe. The evidence register's P0 row already records this honestly ("NOT RUN 2026-11-23: owner authorization unresolved"). No further action required for acceptance.

The plan's Q1 (supersession) is also recorded as reserved. If `docs/prompt15.md` or `docs/prompt15_plan_review.md` contains authorizations that contradict `prompt15.2_plan.md`, P2 must surface that before proceeding. If they do not contradict, P2 proceeds. This is a conditional acceptance: **P2 may begin on the assumption that `prompt15.2_plan.md` is the governing plan**, and if any later inspection reveals a conflict, P2 stops and returns to owner decision.

## What success looks like at P2 exit

The P2 exit review — whichever form it takes — should be able to answer yes to all of the following:

1. Every HIGH finding test (X-01 through X-05) exists, fails against pre-remediation with a failure message matching the contract's prediction, and passes against the remediated tree.
2. X-07's mock measurement integrity test demonstrates the vacuity fix (double-send produces two rows, not one).
3. The choke-point audit holds post-P2 (grep confirms 2 + 2 call sites, no new callers).
4. The transition tables are implemented for all six machines; at least four transitions are spot-checked against their implementations.
5. The sweeper runs at startup, on tick, and on `Status`/poll; its three operations (S1/S2/S3) are exercised by at least one test each.
6. The red/green snapshots are preserved with the exact digests and the runs are reproducible.
7. The X-08/X-11/X-13/X-14/X-15 fixes are implemented and their tests demonstrate the failure-path behavior the contract specifies.
8. The residuals from §11 (F-01–F-14 dispositions, the OSI ownership fix, the cooperative-cancel ops line, the down-file index nit) are implemented or documented as deferred with reasoning.
9. The hard stop on `lead-followup` holds: no implementation, manifest, registration, execution, or ADR approval.
10. No protected document was modified.

If any of these is not true, P2 is not complete. If the contract was wrong in a way that makes one of these impossible, that is the P2 exit report's primary finding.

## Acceptance

**P2 is authorized to begin.** The self-review limitation is accepted with the six conditions above. The conditions are not re-litigations of the contract — they are the empirical verification that makes the self-review acceptable. If P2 cannot satisfy them, the contract was wrong and P2 must say so.

The discipline that the prior cycle failed to establish is now operationalized: diagnose before fixing (RC ratified); establish structural mechanisms (transition tables, choke point, sweeper, fault injector); demonstrate the discipline addresses the findings (red/green); make it reviewable (contract + re-audit). The conditions above extend this to the final piece — **make the self-review verifiable**, not by requiring a reviewer who cannot see the code, but by requiring empirical evidence that the contract's claims survive contact with reality.

The next action is not another review. It is P2 execution against the contract, with the conditions above as the acceptance criteria for P2 exit.

---

**Recorded acceptance:** Self-review limitation accepted by the owner (this document) on 2026-11-23, conditioned on the six items above. Independent contract review waived; independent implementation review required at P2 exit. OpenCode probe remains BLOCKED by owner decision. Supersession question remains reserved; P2 proceeds on the assumption that `prompt15.2_plan.md` is governing, stops if later inspection reveals conflict.



## claude

I'm reviewing the P1 contract independently against the approved plan's ten exit conditions and the governing prompt, and I'm now reading the contract's tail and the Prompt 15.2 sections it cites.

I'm filing this review's context to memory and then writing the independent review of the P1 contract.

## Verdict

I recommend **accepting the P1 contract with conditions**. P1 exit isn't satisfied yet: five document-only amendments are needed before P2 starts, because each one changes what P2 builds or what its tests must assert. The full review is `invocation-lifecycle-contract-review.md`.

**Limits of the review.** It's a document-level review only. I didn't have the repository, the pinned SDK or any test output, so I couldn't check any `file:line` citation in the contract. The review lists seven claims (CR-12), with the file and line to look at, that carry the root-cause ratification. Each takes a minute or two to spot-check.

**What holds up**
- The root-cause ratification is coherent, and the three refinements (R1–R3) are well reasoned.
- All six rules are enumerated, with subsystem-specific operations and no universal `transition()`.
- The submission choke point is specified, with a grep-based proof strategy and a contingency.
- The X-11 mechanism is compared against alternatives and avoids a miniapp-contract change.
- The F-residuals each have their own task or test.
- The agent stopped at the gate instead of starting P2.

**Amend before P2 (Tier 1)**
- **CR-01 (HIGH): attach and cancel.** The loser's cancel behavior in X-08 is written as "defined cancel" but never defined. The P1 exit gate requires this explicitly.
- **CR-02 (HIGH): lock order.** §4 says no path acquires locks out of order, but the Park row inserts the approval and then updates the invocation. That is approval before invocation, which is backwards from the stated order.
- **CR-03 (HIGH): the cancel point of no return.** The claim insert is the reservation, but the transition to ATTEMPTING has no liveness check. A cancelled invocation can still make one more provider call. The review recommends predicating that transition on liveness.
- **CR-04 (MEDIUM): supersede fence.** It is another check-then-act across transactions, the same defect the contract diagnoses elsewhere.
- **CR-05 (MEDIUM): §7.1 completeness.** The contract has no per-machine terminal states or invariants, and no Check-live row.

**Resolve in the first workstream task (Tier 2)**
- **X-11 bound:** it is stated too strongly, since it covers the invocation record and effect gate but not workflow termination.
- **Migration 000006:** the unique index ignores any duplicate run IDs already in the database.
- **X-15:** the kill-switch trigger is called "unbypassable", but a table owner or superuser can disable it.
- **X-09 and X-10:** an alternative that weakens the X-09 test, and how dead-letter workflows are reported.
- **Signals:** a signal to a finished workflow returns "accepted".

Decisions for you (recommendations, not choices I've made):
- Whether to accept this as the P1 contract review, or ask for a second reviewer who can also verify the citations.
- Attach-cancel semantics: should cancelling the attached loser only detach it, or also cancel the shared target run?
- Whether the transition to ATTEMPTING should be gated on liveness. I recommend yes.
- Whether settling an expired approval should also request cancellation of the run.
- Confirm that Prompt 15.2 supersedes the earlier Prompt 15 drafts.
- Authorize the OpenCode preflight, or mark it BLOCKED. If the named model can't be used under existing authorization, record BLOCKED rather than substituting another.


# Independent Review — P1 Invocation Lifecycle Contract (Prompt 15.2)

- **Intended repository path:** `docs/phase1/invocation-lifecycle-contract-review.md`
- **Subject:** `docs/phase1/invocation-lifecycle-contract.md` (status NORMATIVE-DRAFT, authored by the P2 implementer)
- **Judged against:** `prompt15.2.md` §7 (contract requirements), WS-B/WS-D/WS-E, §10; the approved plan's P1 exit gate (consolidated review §5, ten conditions); the P0/P1 evidence rows in `docs/evidence/README.md`
- **Date:** 2026-11-23
- **Authority of this document:** advisory. It does not approve P2, and it is not the owner's acceptance.

---

## Scope and limits (read first)

**This is an independent document review, not a source verification.** I did not have the repository, the pinned SDK tree, a database, or any test output. I read the contract in full, the P0/P1 evidence rows, the governing prompt's §7, WS-B/D/E and §10, the approved plan and the plan reviews.

- I can check completeness against §7, internal consistency, design soundness, and whether claims overreach.
- I **cannot** confirm that any `file:line` citation in the contract is accurate. Those citations carry the "ratified" verdict on RC-1a–d and RC-2. Section CR-12 lists the seven claims that matter most, with a way to spot-check each in minutes.

The reviewer is not the P2 implementer, so this review answers the "reviewed by someone other than the implementer" part of P1 exit condition 5. Whether it is *sufficient* is the owner's call.

---

## A. Verdict

**ACCEPT WITH CONDITIONS. The contract is a strong, mostly sound design and should not be redesigned. P1 exit is not yet satisfied: five document-only amendments (Tier 1) are needed first, because each changes what P2 will build or what the P2 tests must assert.**

What the contract gets right:

- It ratifies the root causes with specific evidence and adds three well-reasoned refinements. R1 (`attached_run_id`) resolves the X-03/X-22 conflict cleanly; R3 (gated tools claim as `pending`) fixes the claim-state ordering.
- All six rules are enumerated and mapped to enforceable operations. There is no universal `transition()`, as the governing prompt requires.
- The lock discipline (§4) states the right principle: never hold a transaction or lock across a send, a DBOS submission or a wait.
- The choke point (§5) is specified by function, inputs and step order. The audit found two call sites for `Async.Submit` and two for `dbos.RunWorkflow`, with a grep-based test strategy and a contingency.
- The X-11 evaluation (§8) compares three mechanisms and chooses one that needs no miniapp-contract change. It is not the Phase 4 reconciler.
- F-residuals are dispositioned individually (§11), with tasks and tests, not a matrix update.
- The agent stopped at the gate and disclosed the self-review limitation instead of proceeding.

**Tiering.**

| Tier | Meaning | Items |
|---|---|---|
| 1 | Amend the contract before P2 starts (design or test-defining) | CR-01 to CR-05 |
| 2 | Resolve in the contract or as the first task of the affected workstream | CR-06 to CR-10 |
| 3 | Observations and housekeeping | CR-11, CR-12 |

---

## B. P1 exit gate scorecard

Conditions are from the approved plan, §5.

| # | Condition | Assessment | Notes |
|---|---|---|---|
| 1 | Root-cause assessment (RC-1a–d, RC-2) | **Accept (citations unverified)** | Reasoning is coherent and internally consistent. See CR-12 for spot-checks |
| 2 | Full coverage (X-01–X-24, F-residuals) | **Accept with gaps** | §10 and §11 cover everything. Coverage of §7.1's required per-machine content is incomplete (CR-05) |
| 3 | Machine ownership | **Accept** | §3 maps every X-ID and the four disciplines. X-11 names the sweeper as a component, which is fine |
| 4 | Transition rules enumerated and enforceable | **Accept** | §2. Enforcement is by predicates and tests, not by a shared abstraction, as intended |
| 5 | Contract review | **Provided here; owner decides sufficiency** | Document-level only |
| 6 | Submission choke point | **Accept** | Audit is grep-level and the P2 proof is a caller-set test. It does not cover non-test files in `testkit/` (confirm) |
| 7 | X-11 mechanism | **Accept with a truthfulness caveat** | Mechanism is fine; the stated bound overreaches (CR-06) |
| 8 | X-08 semantics | **NOT MET** | The attached loser's cancel behavior is asserted ("defined cancel") but never defined (CR-01). The supersede fence repeats check-then-act (CR-04) |
| 9 | X-17/X-18 acceptance | **Accept** | WS-H and WS-I are one sentence each. Digest scope caveat in CR-11 |
| 10 | Evidence provenance | **Accept with notes** | SDK and tree digests recorded, method reproducible. CR-11 |

---

## C. Conformance to Prompt 15.2 §7.1 and §7.7

### C.1 Per-machine content required by §7.1

| §7.1 element | Present? |
|---|---|
| All enumerable states | Yes (§3) |
| Permitted transitions, source state and predicate | Yes (§7 tables) |
| Atomicity and transaction boundary | Yes |
| Authorization precondition | Yes |
| Locking or arbitration mechanism | Yes, with an inconsistency (CR-02) |
| Required audit event | Yes |
| Failure behavior, retry and recovery | Yes, merged into one column |
| **Terminal states** | **No.** Terminal states are implied (§3 state lists) but never stated per machine |
| **Invariants that must hold in every reachable state** | **No.** No machine has an invariant list |
| Owning component | Yes (§3) |

### C.2 Event rows required by §7.7

Covered: Invoke, Same-key retry, Park, Submit-after-park, Grant, Deny, Expire, Decide, Concurrent decide, Cancel, Decide versus cancel, Claim, Send, Settle, Signal, Status, Completion.

**Missing: Check-live.** `checkLive` appears only as the function being replaced (§1, RC-1b). It needs a row stating where its callers go after the fix (the claim transaction subsumes it for effects; confirm no other caller remains).

### C.3 "No row claims enforcement without a test"

Most rows name a test. Rows without a named test or with a loose reference: Retry-after-reconcile ("reconcile-retry test"), the stale-resume interleaving (cross-reference points to §12, which is the migration section, not §11 where the F-03 residual lives), Approval settlement and Executor stop ("kept"), Policy read ("kept"). Name the test or mark the row as covered by an existing named test.

---

## D. Findings

Severity: **HIGH** (changes design or leaves a P1 exit condition unmet), **MEDIUM** (should be fixed in the contract or first in its workstream), **LOW** (observation).

### Tier 1 — amend before P2

#### CR-01 — Attached loser's cancel behavior is not defined (HIGH)
- **Where:** §10 row X-08 ("defined cancel"); §7/M5 "True attach" row has no cancel rule.
- **Why it matters:** P1 exit condition 8 requires "unconditional cancellation/status requirements defined". Prompt 15.2 WS-E lists "defined cancellation behavior" for the losing invocation, and the plan's Gate 5 and the ChatGPT review (item E) require it explicitly. The contract's M6 cancel table covers only invocations that own a run.
- **The undefined cases:**
  1. Cancelling the loser: does it detach the loser only, or cancel the shared target run (which also serves the winner)?
  2. Cancelling the winner while a loser is attached: what does the loser's status become, and does it re-attach to a newer generation?
  3. Who may cancel (tenant and principal rules), and is it idempotent on redelivery?
  4. What the loser reports when the target is cancelled, superseded or fails.
- **Resolution:** add rows to M5 and M6 for each case, with a predicate, an audit event and a named test. If the choice in case 1 changes published behavior, record it as an owner-visible decision (the contract already lists attach as an owner-decision item). I am not choosing a semantic; the two options are detach-only and cancel-target, and they have different blast radii.

#### CR-02 — The lock order in §4 is contradicted by the Park row (HIGH)
- **Where:** §4 ("No path acquires these out of order, so no lock cycle exists"); §7/M1 Park row ("approval row created first in the same tx", then the invocation update); §8/S2 (expiry marking on the approval, then invocation settlement).
- **Problem:** the global order is invocation (1), approval (2), effect (3). Park inserts the approval row and then updates the invocation row, which is 2 then 1. The S2 expiry path is described as marking the approval and then settling the invocation, also 2 then 1, unless these are separate transactions (the contract does not say). In practice a plain insert of a new row will rarely deadlock with `Cancel` (which updates the invocation, then `CancelPending`), but the document asserts something its own tables contradict, and the P2 deadlock tests will be written from this text.
- **Resolution:** either reorder Park to take the invocation lock first and then insert the approval, or state that the approval insert is exempt and why. State explicitly that expiry marking and invocation settlement are separate transactions. Add a table listing every multi-lock transaction with its acquisition order, and extend the order to include the sequence/generation row (CR-04) and note that kill-switch reads take no lock.

#### CR-03 — The "reservation boundary" for X-05 is ambiguous (HIGH)
- **Where:** §4 effect/cancel ordering; §7/M3 Claim and Send rows.
- **Problem:** the contract defines the reservation as the claim `INSERT` (PENDING) under the invocation row lock. That matches 15.2 WS-B literally. But `PENDING → ATTEMPTING` is a later, separate transaction (the "fossilizing" step) with no liveness predicate, and the send follows it. So this interleaving is permitted: claim commits, cancel commits and reports the PENDING effect as in flight, then the worker moves it to ATTEMPTING and sends. The provider call happens after the cancel was acknowledged.
- **Is it a defect?** It satisfies the letter of §7.5 ("an attempt reserved before cancellation is in flight"), but it makes cancellation weaker than it appears: a cancelled invocation can still make one more provider call, and no listed test exercises the window between PENDING and ATTEMPTING.
- **Resolution (decision needed, then document):**
  1. State which transition is the point of no return: claim (PENDING) or attempt start (ATTEMPTING).
  2. **Recommended:** also predicate `PENDING → ATTEMPTING` on invocation liveness, in one local transaction that takes the invocation lock first (order 1 then 3, consistent with §4). That shrinks the post-cancel window to the unavoidable ATTEMPTING-to-send gap, which cannot be closed without holding a lock across the send.
  3. Add a test for cancel between PENDING and ATTEMPTING, and state in the cancel response contract exactly what "in flight" includes.

#### CR-04 — The supersede fence repeats the check-then-act defect (MEDIUM, Tier 1)
- **Where:** §7/M5 "Supersede" row ("every pre-send step re-checks `status='active' AND generation=?`").
- **Problem:** this is the same guard-and-act split the contract diagnoses in RC-1b. A supersede that commits after the step's check and before the send still lets the fenced run send. Rule 2 says the guard must stay authoritative through the durable operation, but here the generation check and the effect claim are separate transactions, and the generation row is not in the §4 lock order.
- **Why it matters:** the X-08 unconditional requirement is "an obsolete generation cannot continue producing effects".
- **Resolution:** either put the generation-active predicate inside the effect claim transaction (extending the lock order to include the generation row), or document the residual window (at most the sends already past their check), bound it, and add a test that measures it. Do not claim the fence is complete without one of these.

#### CR-05 — §7.1 and §7.7 completeness (MEDIUM, Tier 1, document only)
- Add per-machine **terminal states** and **reachable-state invariants** (for example: an approval is `decided` at most once; a settled invocation never returns to a live state; at most one effect row per operation identity; `run_id` is never overwritten once set).
- Add the **Check-live** row (C.2).
- Name the missing tests or point to existing ones (C.3).
- Fix stale cross-references: the F-03 interleaving points to "§12" and the M4 Cancel row's "§12 ops line" should refer to §11; verify the others.

### Tier 2 — resolve in the contract or first in the workstream

#### CR-06 — The X-11 bound is stated more strongly than the mechanism supports (MEDIUM)
- **Where:** §8 ("converge within one sweep interval"); 15.2 WS-D.
- **What holds:** the sweeper makes the **invocation record and the effect gate** converge within the sweep interval. Even if a workflow ignores the decision, effect claims check invocation liveness, so a settled invocation cannot produce new effects.
- **What does not:**
  1. S2 marks the approval expired, signals, and settles the invocation. It does not say it cancels the executor run, so a workflow that never re-reads the approval table can stay alive indefinitely. That is a resource leak, not an effect risk.
  2. The bound exists only while a sweeper is running. The contract does not say where it runs (which process group), what happens when the gateway is down (the startup sweep covers recovery, not downtime), or how multiple replicas interact (idempotent predicated operations should make this safe, but say so).
  3. S1/S2 rely on workflows re-reading the table after waking. plan4.2 §15.1 already requires a recheck of expiry, hash, authority and self-approval before the consequential action. Cite it so this is not read as a new miniapp obligation (Gate 7).
  4. The rejected option (framework-owned bounded wait) has a feasible defense-in-depth form: an analyzer rule against raw unbounded `Recv` in miniapp packages. Schedule it in WS-G.
- **Resolution:** restate the guarantee truthfully (record and effect gate converge within the interval; workflow termination is best effort) and decide whether S2 requests cooperative cancellation of the run. Name the process group and the tick cadence, and confirm the sweeper's connection use fits the Neon budget.

#### CR-07 — Migration 000006 ignores existing data (MEDIUM)
- **Where:** §12.
- **Problems:** `UNIQUE (run_id) WHERE run_id <> ''` fails to create if the X-03 defect ever produced duplicate run IDs in an existing database (development branches are the likely place). The X-19 escape is declared "identity for all existing rows" without a scan showing that no persisted ID contains `%` or `:`.
- **Resolution:** the upgrade test must seed duplicate run IDs and IDs containing `%` and `:`, and the plan must say what the forward-only remediation is (detect, report, and repair before the index, or fail with an actionable message). Run the scan against the real development and Neon branch data before claiming compatibility.

#### CR-08 — X-15 "unbypassable" overclaims (MEDIUM)
- **Where:** §7/D1 ("fires for ANY writer, unbypassable").
- **Problem:** a table owner can disable a trigger, and a superuser can suppress ordinary triggers with `session_replication_role`. The contract already notes that REVOKEs do not bind the local superuser (R2), but it still calls the trigger unbypassable.
- **Also:** the service process holds the control-plane connection string, and miniapps are statically linked into that same process (ADR-016), so trusted in-process code can reach it. That is acceptable under plan4.2's trust model (untrusted code goes to Sprites), but it should be recorded as a residual.
- **Resolution:** define the trigger as `ENABLE ALWAYS`, reword to "fires for every writer except an owner or superuser who deliberately disables it", test it with the superuser and a restricted role, and record the same-process residual next to R2.

#### CR-09 — X-09 weakened alternative and X-10 dead-letter classification (MEDIUM)
- **X-09:** the contract allows "v1-down/v2-up non-progress (or per-version marker attribution)". Prompt 15.2 specifies five steps: pending v1 workflow, stop v1, run only v2 and prove v2 cannot execute it, restart v1 and prove v1 recovers it. Pin the test to those steps; do not offer a weaker alternative.
- **X-10:** the contract treats `MAX_RECOVERY_ATTEMPTS_EXCEEDED` as terminal and reports it as failure. Operationally it is a dead-letter state that needs an operator, so give it its own visible class in the queryable counts (the approved plan said not to omit it). Verify in the SDK source whether it can be resumed; if it can, it is not strictly terminal. Also define what the invocation record shows when its workflow is dead-lettered, so the invocation does not settle as failed while an operator can still resume the workflow.

#### CR-10 — Signal outcome for a terminal run (LOW to MEDIUM)
- **Where:** §6 and §7/M4 ("terminal → harmless accept").
- **Problem:** the contract introduces the verified / accepted / processed vocabulary, but then returns "accepted" for a workflow that has already finished. Rule 5 says not to report success when the required state was not established.
- **Resolution:** return a distinct no-op outcome (for example, already terminal) for finished runs, an error for unknown runs, and a defined unavailable error for an unreachable family. The audit event name `signal/accepted` should apply only to live runs.

### Tier 3 — observations

#### CR-11 — Digest scope and manifest delta (LOW)
- The tree digest covers every file except `.git` and the manifest directory, so any documentation edit changes it. That is why a mid-P1 owner edit forced a second digest. For red/green evidence, define a **source-scope digest** (Go, SQL, CI and scripts, excluding docs and evidence) in addition to the full-tree digest, and state which one P2's pre-remediation snapshot binds to.
- The contract reports 254 files at P0 and 255 at P1, while naming two new untracked documents (the owner's FAQ and the contract itself). Record the file-level difference between the two manifests so the count is explained.
- The five skipped tests in the baseline are four crash-worker child guards and one environment-gated test, which is expected and disclosed.

#### CR-12 — What I could not verify, and a five-minute spot-check list (LOW)
These claims carry the weight of the root-cause ratification. The owner or any second reviewer can confirm them with `sed -n` or `grep` on the tree and the pinned SDK:

| # | Claim | Where to look |
|---|---|---|
| 1 | `invocations.update` is an unconditional update with no expected-state predicate | `runtime/store.go` lines 126–138 |
| 2 | `confirmSubmit` submits any running Class D record and a retry's request never sets the approval field | `runtime/service.go` lines 354–369 and 186–190 |
| 3 | `Decide` and the expiry marking have no pending-status predicate | `approvals/approvals.go` lines 104–140 |
| 4 | `Executor.Signal` returns nil on the first family | `executor/dbos/executor.go` lines 291–307 |
| 5 | SDK `Send` is a blind insert with no foreign key on the workflow table | pinned SDK v1.9.0, `system_database.go` lines 5037 and 2570 |
| 6 | `Shutdown` returns a timeout error naming pending components, including workflows | pinned SDK v1.9.0, `dbos/dbos.go` lines 1057–1148 |
| 7 | The mock deduplicates with `ON CONFLICT DO NOTHING` and carries a false comment | `testkit/mockcap/mockcap.go` lines 132–136 and 76–77 |

If any of these is wrong, the affected ratification in §1 must be redone before P2.

---

## E. Decisions needed from the owner

These are recommendations, not decisions.

1. **Accept this review as the P1 contract review**, conditional on the Tier 1 amendments, or ask for a second reviewer who can also verify the citations (CR-12). Note the contract's own statement: P2 requires explicit acceptance of the self-review limitation or an independent review.
2. **Attach-cancel semantics (CR-01):** detach-only or cancel-target, and whether the choice is an owner-level product decision.
3. **Point of no return for effects (CR-03):** claim only, or also predicate `PENDING → ATTEMPTING` on liveness. I recommend the latter.
4. **Sweeper behavior (CR-06):** whether settlement of an expired approval also requests cancellation of the run.
5. **Supersession:** confirm that Prompt 15.2 supersedes the earlier Prompt 15 drafts, so the earlier GO is not treated as covering this structure. The consolidated reviews recommended this.
6. **OpenCode:** authorize the isolated-HOME model preflight and E2E rerun, or mark it BLOCKED by owner decision. If the named model cannot be used under existing authorization, record BLOCKED rather than substituting another model.

---

## F. Recommended path to P2

1. The agent amends the contract for CR-01 to CR-05 (document only, no source changes) and records Tier 2 items as first tasks of their workstreams.
2. The owner records acceptance of the review (item 1 above) and answers items 2 to 4.
3. P2 starts in the planned order: `FaultInjector` and the mock integrity fix (X-07) first, then WS-A, WS-B, WS-D, WS-E, WS-F and the rest.
4. Keep every standing boundary: no `lead-followup` work, no commits or pushes, no protected-document edits, mock-only channels, disposable databases, forward-only migrations.



## Z

# P1 Exit-Gate Review — Invocation Lifecycle Contract (Prompt 15.2)

| Field | Value |
|---|---|
| Reviewer role | Independent contract reviewer (P1 exit gate, condition 10 of 10; not the P2 implementer) |
| Object under review | `docs/phase1/invocation-lifecycle-contract.md` (603 lines, §§0–15, NORMATIVE-DRAFT) + P0/P1 evidence rows in `docs/evidence/README.md` |
| Verifiable inputs held | `docs/adv_review_prompt14.md` (full), the Prompt 15 root-cause lineage, this reviewer's own plan review (PR-01–PR-10) |
| **Not verifiable here** | `docs/prompt15.2.md`, `prompt15.2_plan_approved.md`, the "chatgpt review C/E/I" and "consolidated approval" documents the contract cites, the repo, the SDK clone, the manifest files. Section references to those documents are accepted as recorded claims (same disposition as PR-10 in the plan review) |
| Question before this review | The self-review limitation (§0, §15): the contract author is also the P2 implementer. This review supplies the independent branch of that gate. |

## A. Verdict: CONTRACT ACCEPTED — P2 GO, conditional on three amendments (CR-01–CR-03) applied to the contract text before code

The contract is an exceptionally strong P1 deliverable. Every amendment this reviewer required at the plan gate (PR-01 through PR-06, plus the non-gating PR-07/08/09) is demonstrably incorporated; RC-1a–d and RC-2 are ratified with file:line evidence consistent with the Prompt 14 review; the six rules are enumerated and enforceable; the transition tables carry predicate, atomicity, authorization, audit, failure exit, and a named test per row; the choke-point audit (2+2 call sites, no bypass, plus an executor-side fail-closed layer at §5.6) is the strongest part of the document; and the evidence register's P0/P1 rows are digested, twice-reproduced, and honest about what did not run (OpenCode NOT RUN, remote CI unobserved, mid-P1 owner doc drift recorded as docs-only).

The adversarial pass found three MEDIUM defects **inside the contract itself** — all three are contract-text amendments, not re-derivations; none reopens the structure. Two are exactly the defect class this contract exists to eliminate (a lock-order violation and a new potential dead-end state introduced by the fix design); one leaves a documented revocation race that the governing review's invariant appears to forbid. Apply them to the document, cite the amended rows in the P2 opening state, and begin P2.

**Ruling on the self-review question:** this document is the independent review; the limitation is discharged for P1. It is not a blanket waiver: the P2 implementation output still receives an independent exit review (Prompt 16-style), per the standing post-review principle (§2.5 of the lineage). Supersession and OpenCode remain owner-reserved and are unaffected by this review.

## B. Plan-review amendments — verification

| Plan-review item | Required | Contract evidence | Verdict |
|---|---|---|---|
| PR-01 (HIGH): F-residuals missing | Closure list with WS/test/doc per residual | §11: all four named residuals (stale-resume interleaving, `Settle` audit, three F-10 signal cases, lazy-expiry E2E, `settleDenied` wording, cooperative-cancel ops line) **plus** §H/§G/§J/§K/§L extras dispositioned; "a matrix update alone is not sufficient" | INCORPORATED (beyond what was required) |
| PR-02: machine ownership gaps | No ownerless finding | §3: M4 owns signal delivery + foreign-version visibility; D1 owns kill-switch control plane; finding→owner map covers X-01–X-24; "exactly one primary owner" | INCORPORATED |
| PR-03: sanctioned invocation in CI; parallel-red root-caused | `-p 1` baked into CI; root cause named | P0 register row: cross-package `TRUNCATE` traced (`runtime/service_test.go:318`, 7 tables/test, `t.Parallel` count 0 → cross-binary), pair repro (parallel FAIL vs `-p 1` PASS); §10 X-17: "sanctioned `-p 1` (Gate 4 — never bare `./...`)"; §13 WS-H one-liner; G1 row 12 stays PARTIAL | INCORPORATED (root cause is traced, not labeled — exactly what was demanded) |
| PR-04: three missing boundaries | Mock-only, no-weakening, disposable-DB restated | §14: all three verbatim, plus hard stop, no commits, forward-only migrations | INCORPORATED |
| PR-05: X-08 unconditional parts | Supersede linkage, claim race, loser status — regardless of attach choice | §10 X-08: "UNCONDITIONALS regardless: no `Succeeded`-while-active, barrier claim-race test, supersede→cancel (fencing, §7/M5)"; recorded as owner-decision item despite 15.2 pre-deciding | INCORPORATED (see CR-07 on the fence substitution) |
| PR-06: per-part fidelity | Full sub-part lists survive the WS one-liners | §10: X-01 (dangling-row adoption, exactly-one audit), X-02 (grant AND denial AND expiry converge), X-04 (expiry-marking AND `CancelPending` predicate discipline), X-07 (comment + phrasing + re-runs), X-11 (bound + commit-time audit + mailbox strike) — all present | INCORPORATED |
| PR-07: snapshot mechanism named | Worktree at digest, both runs digested | §13 C2: "isolated Git worktree at the recorded pre-remediation digest… both digests recorded; active tree untouched" | INCORPORATED |
| PR-08: SDK clone bound | Digest in register; no machine-local paths | Header: SDK digest `308583f2…` (260 files, manifest preserved); README P1 row: "machine-local paths excluded from evidence" | INCORPORATED |
| PR-09: WS-A rationale fixed | Drop the loose "same files" claim | §10: "shared invocation/approval lifecycle ownership" | INCORPORATED |

## C. §15 self-check — nine conditions independently confirmed

1. **Root-cause assessment (§1):** RATIFY verdicts match this reviewer's own synthesis; each RC carries file:line evidence consistent with Prompt 14 §E (X-03 `store.go:126-138`; X-04 `approvals.go:104-140` with the conforming `CancelPending` contrast; X-05 `lifecycle.go:230-244` + `effects.go:142-182`; X-01's three triggers; X-02's `pollApproval` never-submits; X-06's blind `Send` with SDK cites; X-12/X-13 read-driven/emitted-nothing). The three refinements are genuinely additive, not refutations.
2. **Full coverage (§10 + §11):** every X-ID has sub-parts + test; every F-residual has a task/test/doc disposition.
3. **Machine ownership (§3):** no ownerless finding; X-19's shared-helper footnote is a good cross-machine honesty note.
4. **Rules (§2):** all six present and enforceable; "no universal `transition()`" with a written-justification escape hatch is the right shape.
5. **Choke point (§5):** 2 `Async.Submit` call sites + 2 non-test `dbos.RunWorkflow` sites, all routed; grep-proof test planned; executor-side rejection (§5.6) gives defense in depth — even a future bypassing caller cannot run gated work unapproved.
6. **X-11 mechanism (§8):** sweeper selected with the alternatives' rejection rationale recorded (framework cannot intercept `Recv` at runtime; poll-driven has no provable bound); scope strictly invocation/approval lifecycle; never drives effect outcomes; S2's signal wake covers the 24h-`Recv` workflow; mailbox phrase struck.
7. **X-08 semantics (§7/M5 + §10):** genuine attach + unconditionals + attach re-resolve.
8. **X-17/18 (§13 + §10):** skip-fast gates, CI postgres + sanctioned invocation, digest-per-run, historical claims marked irreproducible/never retrofitted — the register's prompt13 rows visibly follow this.
9. **Provenance (§0 + §13):** tree digest twice-reproduced with documented method; SDK digest; mid-P1 owner drift recorded as docs-only with both manifests.
10. **Contract review:** this document.

P0 claims spot-checked against the register: 168 PASS / 0 FAIL / 5 SKIP serial (row: 20 pkgs ok); preflight 1/2/0/1; Neon probe read-only; OpenCode NOT RUN with the authorization reason; delta vs `5564a37b` = +7 docs, zero source drift. All match the implementer's report.

## D. Findings (contract amendments required before P2 code)

### CR-01 (MEDIUM): attached-run convergence is unspecified — the fix must not create a new dead-end class

R1's `attached_run_id` creates invocation rows whose own `run_id` is empty. The settlement machinery is keyed on `run_id` only: sweeper S3 ("live records **with run_id** whose executor `Status` is terminal") and M1's Completion row ("`running`/`waiting_event` **(+run)**") never see an attached record. M5's "loser status follows target run" states intent for sequence attachments but gives no mechanism, and M1 has **no row at all** for same-key attachments (the X-03 loser). Left as written, an attached invocation can strand live forever when its target settles — an RC-1d dead end manufactured by the fix itself. **Required:** add explicit coverage — (a) `settleTerminal`/S3 predicates cover records with `attached_run_id` set (resolve the owner row's outcome; propagate failure as failure, per M5); (b) Status-by-invocation resolves `attached_run_id` → owner run; (c) an M1/M5 row defining cancel semantics for an attached record ("defined cancel" is currently promised in §10 X-08 but nowhere defined).

### CR-02 (MEDIUM): the M1 Park row violates the contract's own global lock order — AB-BA deadlock window against Cancel

§4 fixes acquisition order 1 invocation → 2 approval → 3 effect → 4 advisory. The Park row runs "approval row created **first** in the same tx," i.e. lock 2 (INSERT takes the new row's lock) then lock 1 (predicated invocation UPDATE) — 2→1. M6 Cancel runs 1→2 (invocation UPDATE, then `CancelPending` on the pending approval) in one tx. Since `pending` must be inside Cancel's live predicate set (a gated, not-yet-parked invocation must be cancelable), concurrent Cancel-during-Park on the same invocation is a textbook AB-BA: each holds the row the other needs, until `statement_timeout`. The whole point of §4 is that no such cycle exists. **Required (preferred minimal fix):** reorder within the one transaction — predicated invocation park-UPDATE first (lock 1), then the approval INSERT (lock 2), then the audit; atomicity and the exactly-once `approval-requested` audit are preserved, and §4's consequences text stays true. Alternatively split into two transactions with defined failure exits (§6's approval-create "missing row (create)" row already covers the split), but state which was chosen and re-verify §4's consequence list against the amended rows.

### CR-03 (MEDIUM): the pre-transaction grant check leaves a revocation race the governing invariant forbids — close it or make the residual an explicit owner decision

M3's Claim (new) row re-reads token and kill-switches **in-tx** but keeps the grant-map check **pre-tx**, with a parenthetical residual: "revocation landing mid-tx is observed at the next claim; the attempt completes cooperatively." Prompt 14 §12 (Workstream G) states the invariant verbatim: **"A race between policy read and effect claim cannot bypass revocation."** A revocation committing between the pre-tx grant read and the claim's commit is exactly such a race — bounded to one transaction, but the invariant is unconditional. **Required:** either fold the grant check into the claim transaction (the tx already re-reads policy inputs; a grants read/join is the same shape and likely trivial), **or** promote the residual to an explicit, named owner-decision entry that cites the governing invariant and records why in-tx grants were not chosen — not a parenthetical in a table cell. The §H-7 wait→revoke→refuse E2E covers the durable-wait case; it does not cover this race.

### CR-04 (LOW): state the sweeper's transaction boundaries explicitly

S2 touches both machines (approval expiry marking, then invocation settlement). The M2 Expiry row says "one tx (Decide path + sweeper path share it)" and settlement is a separate settle transaction — which respects §4's order *across* transactions — but the contract never says the sweeper releases the approval lock before acquiring the invocation row. One sentence ("sweep steps run as separate sequential transactions; no lock is held across steps") closes the only unwritten order question in the document.

### CR-05 (LOW): the one-tx Park makes the dangling-approval branch defensive/legacy-only — the X-01 test must be built accordingly

Under the new park (approval INSERT + park + audit in one transaction — which is itself the right design), a park failure rolls back the approval row too, so the X-01 "adopt a legitimate dangling pending row" branch (§5.3) can no longer be produced by a live fault injection; it is reachable only from pre-existing/legacy rows or fixture state. Keep the branch (robustness), and construct `TestApprovalRetryAfterSubmitFailure`'s adoption leg from fixture state rather than expecting fault injection to yield a dangling row. Note it in §10 so P2 does not discover this mid-build.

### CR-06 (LOW): X-15's ephemeral-role proof must run through a real restricted-role connection

"Proven with ephemeral test roles" is the right intent; pin the standard: the handler-attempt test must connect **as** the ephemeral restricted role and receive the denial from PostgreSQL (the same standard the review applied to the BYPASSRLS proof — "genuine privileged execution, not text-matching"). Catalog inspection of the REVOKE statements is not proof.

### CR-07 (LOW): record the supersede→fence substitution and enumerate the six acceptance checks

The review's X-08 resolution asked for a "supersede→cancel linkage"; the contract delivers generation fencing instead (every pre-send step re-checks `status='active' AND generation=?`; fenced runs exit without sending). That is arguably **stronger** than cooperative cancel for the invariant that matters (no send after supersede — cooperative cancel cannot stop an in-flight step anyway), but the substitution is a deliberate design deviation from the resolution text and should be recorded as such with the one-line rationale, and the "six acceptance checks (chatgpt E)" referenced in §10 should be enumerated in the P2 report so they are checkable.

## E. What to preserve unchanged

The choke point with its executor-side fail-closed layer (§5.6); the sweeper's strictly-lifecycle scope with the "not the Phase 4 reconciler" boundary; R3 (gated tools claim as `pending`), which removes the X-01 bypass surface by construction rather than by patching the replay route; the X-19 percent-escape with legacy identity compatibility; the X-23 four-part enforce/wire/strike/fence decisions; the evidence discipline (§13) including "historical claims marked irreproducible, never retrofitted"; and the honest register rows (OpenCode NOT RUN with reason; Neon probe read-only; G1 row 12 stays PARTIAL).

## F. P2 GO — conditions

1. Apply CR-01, CR-02, CR-03 to `docs/phase1/invocation-lifecycle-contract.md` before P2 code; cite the amended rows in the P2 opening state (a section-level diff is sufficient — no full contract re-review is required for these three textual amendments). CR-04–CR-07 are to be applied while editing or honored at implementation time.
2. The §14 standing boundaries bind all of P2 as written.
3. Supersession and OpenCode remain owner-reserved: OpenCode stays NOT RUN/BLOCKED with G1 row 10 adjusted accordingly unless the owner authorizes; no model call without authorization.
4. The self-review limitation is discharged **for P1 only** by this review; P2's implementation and report still receive an independent exit review before any G1 re-adjudication is trusted.
5. Remote CI remains honestly unobserved: G1 row 12 stays PARTIAL unless remote execution is actually observed (already encoded in §13).

## G. Scope limits of this review

Document-level review of the contract and register as uploaded; no repo, SDK clone, or execution access. The contract's citations to `docs/prompt15.2.md`, the approved plan, and the "chatgpt review" series are accepted as recorded claims; where this review's verdicts depend on those documents (e.g., Gate numbering, "consolidated approval" contingencies), the dependence is stated rather than hidden. All findings above are grounded in the contract's own text and in `docs/adv_review_prompt14.md` held by this review.



## chatgpt

P1 independent review — Invocation Lifecycle Contract
P0 accepted
P1 substantially complete
P2 not yet authorized

I reviewed the actual invocation-lifecycle-contract.md and README.md evidence register, not just the agent's completion summary.

My verdict: CONDITIONALLY ACCEPTABLE. The root-cause diagnosis, architectural direction, and most of the transition contract are sound. I would accept the independent-review requirement as satisfied by this review of the design document, but I would require several contract corrections before P2 begins.

I do not recommend restarting P0 or rewriting P1. The work completed is valuable.
1. P0 — Accepted
The evidence register provides a credible baseline.
Item	Assessment
Worktree digest	PASS
Source drift against prior baseline	PASS
Serial full suite	PASS
Static gates	PASS
Parallel test failure investigation	PASS
Neon branch identity	PASS
OpenCode preflight	NOT RUN

The digest ba3592701e696c59746ac923895edaee7b203cc660c1574543560167b4dfe091 is accompanied by a reproducible manifest. The difference from the previous digest is reported as seven additional planning/review documents, with no source changes.

The serial suite reports 168 PASS, zero FAIL, and five SKIP. The parallel failure is traced to cross-package TRUNCATE operations against a shared database, with a reproducing test pair. This is an important distinction: the current parallel failure is an integration-test isolation defect, not evidence of a framework concurrency defect.

The sanctioned -p 1 execution must remain part of CI until the fixtures are made parallel-safe. 
README(20261123-002115).md

2. P1 — What I accept
The contract makes several important architectural decisions correctly.

RC-1 and RC-2 are well diagnosed. The four runtime mechanisms—non-predicated writes, check-then-act boundaries, authorization by path ordering, and unrepresented failure states—are a persuasive explanation of X-01 through X-05. RC-2 correctly identifies that some earlier tests could pass while the defective behavior remained possible.

The six transition-integrity rules are appropriate. In particular, predicated transitions, authorization at the execution boundary, retry-as-re-entry, truthful lifecycle reporting, and falsifiable evidence should become standing framework requirements.

The subsystem-specific approach is correct. I agree with rejecting a universal transition() primitive. Different subsystems need their own state-transition operations while obeying shared guarantees.

The submission choke point is a strong design decision. It makes approval enforcement a property of the framework's actual execution boundary rather than an expectation placed on each caller.

R1 is especially valuable. Separating run_id from attached_run_id resolves the conflict between unique workflow ownership and genuine attachment. It gives X-03, X-08, and X-22 one coherent identity model.

The sweeper is a reasonable X-11 choice. A narrowly scoped framework-owned approval/settlement sweeper is preferable to requiring every miniapp developer to implement signal recovery or approval-expiry loops independently.

The contract also gives all 24 findings and the additional F-01–F-14 residuals explicit owners and tests. That is a substantial improvement in traceability. 
invocation-lifecycle-contract.md

3. Required corrections before P2
I found six issues that deserve attention. The first four affect the correctness of important framework guarantees; the remaining two affect the completeness and security of the proposed mechanisms.
R-01 — Clarify the effect reservation and cancellation boundary
Required before P2
The contract describes:
1. Lock invocation row.
2. Verify liveness.
3. Insert an effect row in PENDING.
4. Commit.
5. Subsequently transition PENDING → ATTEMPTING.
6. Send the provider operation.
The cancellation operation locks the invocation row and subsequently enumerates effects.

The contract calls the reservation the point at which the effect becomes in flight, but it does not make the meaning of PENDING sufficiently explicit.

This matters when cancellation commits after the PENDING insertion but before the ATTEMPTING transition.

The contract needs to establish exactly which state transition authorizes the provider attempt and whether a worker can resume an existing PENDING record after cancellation.

Required correction:
- Define whether PENDING is already a reserved, in-flight business attempt.
- Specify which transitions permit the actual send.
- Ensure cancellation sees every reservation that committed before it.
- Ensure retries and stale lease takeovers cannot create a new attempt after cancellation.
- Require an adversarial test that pauses execution between PENDING and ATTEMPTING, commits cancellation, and verifies the specified outcome.
- Test both claim-before-cancel and cancel-before-claim.
The rule must remain:

A reservation that wins before cancellation is already in flight. A new reservation after cancellation is prohibited.

Do not hold the database lock across the provider call.
R-02 — Supersession fencing must reach the effect boundary
Required before P2
M5 says that every pre-send step rechecks:

status = 'active' AND generation = ?

That is better than having no generation check, but it is still potentially a check-then-act race.

An old workflow might pass its pre-send check, get superseded, and subsequently reach the shared effects service.

The framework must make supersession meaningful at the actual effect-reservation boundary.

Required correction:
- Define the generation/fencing identity carried by the workflow's trusted execution context.
- Validate it when the consequential effect reservation is committed.
- Serialize the validation against supersession.
- Reject the reservation from a superseded generation.
- Preserve the existing effect-ledger fencing rules.
Add a test that pauses an old generation after its initial check, supersedes it, then allows it to attempt an effect.

The test must prove that the stale generation cannot send.

This is important beyond the neutral probe: it establishes a reusable guarantee that future Class D miniapps can rely on without implementing their own generation fencing.
R-03 — Resolve the atomicity contradiction in approval parking
Required clarification
There are two descriptions that need reconciliation.

In the submission choke point, the contract says a retry can adopt a dangling pending approval row left by a failed park.

In M1's Park transition, it says approval creation, invocation parking, and audit insertion occur in the same transaction.

If all three writes are in one PostgreSQL transaction, a failed transaction rolls all three back. There should be no dangling pending row from a failed park.

Both approaches can be legitimate, but they describe different transaction models.

Choose and document the actual one:

Option A — Atomic parking

Create the approval record, transition the invocation, and write the audit event in the same transaction. Failure rolls them all back. A retry creates them again using the same idempotency identity.

Option B — Separate durable operations

Create the approval independently, then park the invocation. A crash or failure can leave a dangling approval row, which the retry must discover and adopt idempotently.

If using B, explain why the operations cannot share a transaction.

Ensure that the fault-injection tests distinguish failure of Approver.Submit, park persistence, and audit insertion. Do not keep both transaction models as undocumented assumptions.

Also specify acquisition order for the invocation and approval rows in this transaction to preserve the global lock discipline.
R-04 — Extend settlement discovery to attached runs
Required correction
R1 deliberately gives an attached invocation:
- An empty run_id.
- A populated attached_run_id pointing to the owning workflow.
However, S3 in the sweeper describes settlement discovery for live records with run_id.

That leaves attached invocations outside the stated settlement-discovery query.

Required correction:
- Include attached workflow identity when discovering terminal outcomes.
- Settle the attached invocation from the authoritative target workflow's status.
- Prevent duplicate settlement audits through the shared settleTerminal transition.
- Test success, failure, cancellation, and target-process unavailability for an attached invocation without any caller polling Status.
The rule should be that attachment does not sacrifice lifecycle observability.
R-05 — Clarify the sweeper's actual recovery guarantee
The chosen sweeper is a good direction, but its timing guarantee needs more precise wording.

The contract claims convergence within one sweep interval, using a 30-second default.

That guarantee is achievable while a healthy sweeper instance is running. It cannot be an unconditional wall-clock guarantee while every application instance is stopped or the database is unavailable.

Specify:
- Recovery within the configured interval while at least one healthy sweeper is running and its dependencies are available.
- Startup recovery following process restart.
- Idempotency across concurrent sweeper instances.
- Behavior when a sweep fails or exceeds its interval.
- Bounded batch processing and error visibility.
- Whether attached workflow identities are included.
- The consequences of a prolonged complete service outage.
Do not promise a bound that the framework cannot enforce during total unavailability.

Also ensure the sweeper follows the global invocation → approval → effect lock order when an operation needs multiple records.
R-06 — Tighten X-15 privilege and credential separation
The proposed restricted app role and privileged setter are directionally correct. However, the contract currently describes the serving service as holding a MINIAPP_CONTROL_DATABASE_URL.

A security boundary must not depend on ordinary miniapp code being unable to access a credential that is present in its own serving process.

I recommend:
1. The ordinary serve process must not load or retain the control-plane DSN.
2. The authenticated operator command should use the separate control-plane credential.
3. The ordinary runtime must use a restricted database role, including in local integration tests.
4. The application startup should reject an unsafe role configuration rather than silently run with elevated privileges.
5. The audit trigger must not be described as universally unbypassable: superusers and sufficiently privileged table/function owners remain capable of bypassing ordinary privilege restrictions.
6. Verify actor and reason attribution cannot be arbitrarily supplied by an unauthorized SQL caller.
Test the actual runtime role, not only an artificial restricted role in a separate test.

The superuser limitation is a PostgreSQL privilege boundary, not something an audit trigger can magically eliminate.
4. One additional framework concern: grant revocation
The M3 claim row specifies that grant-map validation happens before the transaction, while token and kill-switch checks are performed inside it.

That leaves an important question:

What happens when a grant is revoked after the pre-transaction check but before the effect reservation commits?

The contract acknowledges this as a residual but does not fully define its authorization semantics.

I recommend that P1 explicitly define the effect authorization linearization point and P2 investigate whether grant validation can participate in the same atomic database boundary.

If the grant source cannot participate in that transaction, define the trusted snapshot/token semantics and their limitations. Do not imply that every revocation committed before reservation necessarily prevents the send unless the implementation actually guarantees it.

Also, the F-03 Settle work requires more than an audit event. The re-audit should identify the actual operator-accessible surface through which authorized settlement can be performed. A test-only API does not constitute operational exposure.
5. Independent-review and owner-decision gates
I will distinguish the following:
Gate	Assessment
P0 baseline	Accepted
RC-1/RC-2 diagnosis	Accepted
State-machine design	Amend as above
Transition rules	Accepted in principle
Contract self-review	Independent document review performed
Submission choke-point audit	Accepted for P1 design
X-11 mechanism selection	Accepted subject to R-05
Evidence provenance	Accepted

My review is independent of the agent that will implement P2, so it satisfies the requested independent document review at this gate. It does not certify implementation behavior before code exists.

Two owner matters remain:

Prompt supersession: explicitly confirm that Prompt 15.2 supersedes the previous Prompt 15 drafts and governs this remediation. The earlier GO should not be treated as authorization for a different prompt structure.

OpenCode: the preflight remains NOT RUN. This need not block unrelated P2 engineering, but G1 row 10 must remain PARTIAL/BLOCKED until the authorized E2E is completed. No model call should be made without your authorization.
6. Final recommendation
Do not restart P0 or P1. Amend the existing contract in place.

The most consequential corrections are:
1. Clarify the precise effect-reservation/cancellation boundary.
2. Extend generation fencing to the actual effect-reservation boundary.
3. Reconcile atomic approval parking with dangling-row recovery.
4. Include attached workflows in settlement discovery.
5. Qualify the sweeper recovery bound.
6. Enforce real control-plane credential and role separation.
These are contract refinements, not requests for architecture redesign.

Once they are addressed, I would be comfortable recommending P2 authorization. The existing dependency order should remain intact.

One important point for the implementation agent: do not treat these revisions as permission to simplify away the framework's reusable primitives. Your standing objective remains that the framework owns reliability and each miniapp owns business logic. Every correction should strengthen a shared primitive, not introduce a one-off workaround in the neutral probe.

Owner decision needed: confirm that prompt15.2.md is the governing and superseding Prompt 15 specification. P2 remains held until the six contract corrections above are made and the revised contract passes the exit review.