# Prompt 15 — Root-Cause Remediation of the Class D Framework

## 1. Role and mission

You are the implementation agent authorized to remediate the defects identified in `docs/adv_review_prompt14.md` against the Class D framework delivered under Prompt 13.

This is **not** a remediation-by-finding-list task. Each of the five HIGH findings (X-01 through X-05) is a symptom of a deeper structural gap in how the framework manages durable state transitions under concurrency and failure. Fixing each symptom in isolation will leave the framework structurally brittle in exactly the same way for the next workflow — the framework will appear ready while continuing to hide failure-path defects beneath passing happy-path tests.

Your job is to:

1. Identify and remediate the **root cause** that produces X-01 through X-05 and a substantial subset of the MEDIUM findings.
2. Establish a **structural discipline** that makes this class of defect visible and preventable going forward.
3. Demonstrate that the remediation genuinely addresses the underlying pattern — not merely the individual symptoms.
4. Leave behind a practice that future adversarial reviews can point to and future implementation agents can be held to.

The deliverable is not "the findings are closed." The deliverable is "the class of defect that produced the findings cannot silently recur."

## 2. Root cause analysis (authorized diagnosis; challenge it before accepting it)

The five HIGH findings are symptoms of a single structural gap:

**The framework lacks a unified state-transition discipline over durable state.** Every durable mutation — invocation submission, approval decision, effect claim, cancellation, run binding — is written as an ad-hoc sequence of reads and unpredicated updates that happen to be correct when the happy path runs unopposed. Under concurrency, crash-at-boundary, or retry-from-a-different-state, the same code produces divergent durable state, orphaned workflows, bypassed authority, and non-converging retries.

Concretely, X-01 through X-05 all arise from the **absence of one or more** of the following properties:

| Property | Missing in | Symptom |
|---|---|---|
| **Predicated transitions** — every durable write is compare-and-swap on the current state (`UPDATE … WHERE status = expected`) | `runtime/store.go` `update`, `approvals/approvals.go` `Decide` | X-03 (unpredicated `run_id` overwrite), X-04 (last-writer-wins in Decide) |
| **Retry-as-re-entry** — every retry path reads current state and dispatches on observed state, not on the caller's "this is a retry" flag | `runtime/service.go` `confirmSubmit`, `pollApproval` | X-01 (retry bypasses approval), X-02 (poll never submits) |
| **Locks span the critical section** — `SELECT … FOR UPDATE` is held from state read through state write AND through any external observation that the state serializes against | `runtime/lifecycle.go` `checkLive`, `effects/effects.go` `claim` | X-05 (cancel window between lock release and claim) |
| **Fault injection at every transition boundary** — every state machine has a test for "operation N fails, retry from state M, converge to state K" | Throughout the test suite | All five findings are on failure paths the suite does not cover |
| **Invariant-first tests** — every invariant has a test that hits the failure path, not just the happy path | `TestCancelBeforeClaim`, `TestSubmitIdempotent`, all Decide tests | X-05 (no concurrent cancel-during-claim), X-04 (all Decide tests sequential) |

The MEDIUM findings split roughly into two categories. Approximately half (X-06, X-08, X-11, X-12, X-13, X-14) share the same root cause — missing predication, non-re-entry recovery paths, missing fault coverage, or incomplete state transitions. The remainder (X-07, X-09, X-10, X-15, X-16, X-17, X-18, X-19–X-24) are the specific defects the framework's own discipline would surface if the discipline existed.

**Your first task is to independently assess this root cause diagnosis.** If you conclude it is wrong or incomplete, say so in the required design document (§9) and justify the revision. Do not merely accept it and start coding.

## 3. Governing authority and precedence

Read the actual repository files:

1. `docs/plan4.2.md` — immutable architectural authority.
2. `docs/adv_review_prompt14.md` — the adversarial review this work addresses.
3. `docs/prompt13.md` — the original implementation specification.
4. `docs/prompt13_imp.md` — the implementation agent's claims.
5. `docs/prompt14.md` — the review specification that produced the review.
6. All implementation source, tests, migrations, scripts, analyzer fixtures, CI workflows.
7. The pinned SDK source (`dbos-transact-golang` v1.9.0) — read it, do not assume it.

### Precedence

1. `docs/plan4.2.md`.
2. Accepted G0 decisions and recorded owner decisions.
3. Prompt 13 as the authorized Phase 1 implementation specification.
4. Existing approved contracts and ADRs.
5. The adversarial review's findings (X-01 through X-24, F-01 through F-14 verdicts).
6. Prompt 13's implementation report as historical claims (not authority).
7. This prompt as the authorization for the remediation scope and discipline.

Where this prompt and Prompt 13 disagree, this prompt governs. Where this prompt and `plan4.2.md` disagree, `plan4.2.md` governs, and you must report the conflict rather than proceed.

## 4. Scope

**In scope:**

- Remediation of X-01 through X-05 (BLOCKER-class HIGH findings).
- Remediation of the MEDIUM findings that share the identified root cause.
- Establishment of the structural discipline (§5).
- A state-machine specification document for each subsystem with durable state (§9).
- A fault-injection test harness extension (§7).
- A systemic re-audit document (§10).
- Small hardening items from the LOW findings where the fix is bounded and mechanical.

**Out of scope (do not start these; they belong to other phases or owners):**

- Real provider integrations, OAuth flows, production reconciler, volumetric controls (Phase 4+; OSI-4).
- G6 evidence (Fly deployment overlap, artifact retention, cross-release drills).
- Remote CI green-run observation beyond ensuring the fix will pass when pushed.
- `lead-followup` implementation, manifest, registration, or execution.
- The `lead-followup` ADR review remediation (a separate owner action).
- River/Class C, Sprites, promotion pipeline, full lifecycle CLI.
- Changes to `plan4.2.md` or any immutable governing document.
- Changes to the review report `docs/adv_review_prompt14.md`.

**Hard stop (unchanged):** No `lead-followup` code, manifest, registration, execution, or ADR approval. This prompt does not confer approval.

## 5. Required discipline (the deliverable beyond the fixes)

The remediation must establish, and the design document must specify, five structural properties that apply to every subsystem with durable state (invocations, effects, approvals, runs, sequences, cancellation):

### 5.1 State-machine formalization

Every subsystem's durable state must be defined as an explicit state machine:

- States (the enumerable set).
- Transitions (source state → destination state, with predicate, effect, and audit requirement).
- Invariants (properties that must hold in every reachable state).
- Terminal states (states from which no further transition is permitted without a new generation).

The state machine must live in a specification document (§9) and be represented in code as a single `transition()` primitive per subsystem, not as scattered `UPDATE` statements. A transition primitive has a signature like:

```
Transition(ctx, id, fromStates []State, toState State, predicate func(current) bool) (Result, error)
```

which performs the read-modify-write under a row lock, verifies the current state is in `fromStates`, checks the predicate, writes the destination state, and emits the audit event — atomically, in one transaction.

Every existing ad-hoc `UPDATE` in the affected subsystems must be replaced by a call to the transition primitive. This is the single most important structural change.

### 5.2 Predication on every write

Every durable write must be predicated on the current state. Unpredicated updates are forbidden in these subsystems:

- `UPDATE invocations SET run_id = …` must become `UPDATE invocations SET run_id = … WHERE run_id = ''` (or the equivalent transition call), with `RowsAffected` checked and reload-on-zero-rows behavior.
- `UPDATE approvals SET status = …` must be predicated on `status = 'pending'` (or the required prior state).
- Effect claims, settlements, and takeovers must be predicated on expected prior state and use `RowsAffected` to detect losers.
- Every `WHERE` clause on a state-mutating statement must include the expected current state.

### 5.3 Retry as re-entry, not as flag

Any operation that can be retried (invocation submit, approval-poll, effect claim) must, on entry, read current state and dispatch based on **observed state**, not on caller-supplied "is retry" flags. This is the pattern that makes X-01 and X-02 structurally impossible:

```
func (s *service) submitInvocation(ctx, req) (Handle, error) {
    record := loadInvocation(ctx, req.idempotencyKey)
    switch record.state {
    case StateClaimedNoRun:
        return s.beginApprovalOrSubmit(ctx, record)   // requires fresh approval check
    case StateWaitingApproval:
        return s.resumeApprovalPoll(ctx, record)       // requires submitting if run missing
    case StateRunning:
        return s.attachOrResubmit(ctx, record)         // requires idempotent re-submit
    case StateTerminal:
        return record.result, nil
    }
}
```

Every state a retry can land in must have a defined handler. Every handler must be idempotent under the predicated transitions from §5.2. There is no separate "retry code path" — retry is just entering the state machine again.

### 5.4 Lock scope spans the critical section

Any check-then-act sequence where the check's outcome determines the act must be executed with a lock held from the check through the act, **and the lock must cover any external observation the act's correctness depends on**.

For X-05 specifically: `checkLive` + `claim` + `attempt` + `send` must be serialized against `cancel`. The predicated transition (§5.2) on the effect state must include "is the invocation live?" as part of its predicate, checked under the same row lock that the cancellation uses.

If holding a lock across an external call is infeasible (which it usually is), the external call must be preceded by a transition that would fail if cancellation had landed, and the cancellation must be predicated on the effect state to prevent racing.

### 5.5 Fault injection and invariant tests

Every state machine must ship with:

- A **fault-injection test** for each transition: fail the operation, then retry from the pre-transition state, and assert the transition either completes or converges to a defined error state.
- An **invariant test** for each stated invariant: hit the failure path that would violate the invariant, assert the invariant holds.
- A **concurrency test** for each pair of operations that can race (using barrier synchronization, not sequential submission).

The testkit must grow a `FaultInjector` type that can be configured to fail a named operation on its Nth attempt. The affected tests must use it. This is the mechanism that prevents "the suite passes but the failure path is untested."

## 6. Required implementation order

Do not start with the code fixes. Follow this order:

### Step 1 — Write the root-cause diagnosis and design document

Produce `docs/phase1/classd-state-transition-discipline.md`. It must contain:

- Your independent assessment of the root cause (§2 of this prompt). Agree, disagree, or revise, with reasoning.
- Per-subsystem state machine specification: states, transitions, predicates, invariants, terminal states.
- The transition-primitive design: signature, semantics, failure modes.
- The retry-as-re-entry design: what "current state" means for each subsystem, what handler runs per state.
- The lock-scope specification for each check-then-act sequence.
- The fault-injection harness design.
- The list of every existing test that must be added, strengthened, or replaced.
- The mapping from each HIGH and root-cause-sharing MEDIUM finding to the structural change that makes it impossible or caught.

This document is the primary deliverable. If it is wrong, everything downstream is wrong. Have it reviewed internally before proceeding.

### Step 2 — Implement the transition primitive per subsystem

Build the `transition()` primitive for invocations, approvals, effects, runs, and sequences. Do not implement ad-hoc transitions and refactor them later — implement the primitive first and route every durable mutation through it.

Do not merge this step with Step 3. The primitive must be merged and tested standalone before the call sites change.

### Step 3 — Refactor call sites

Replace every ad-hoc durable write in the affected subsystems with a `transition()` call. This is a mechanical change, but it will surface every place where the current state is unclear or unstated — which is exactly the point.

### Step 4 — Implement retry-as-re-entry

Rewrite `confirmSubmit`, `pollApproval`, and any other "retry" or "recover" path as state-dispatch handlers. Delete any "is this a retry" flag that influences control flow (calling it a flag whose state is ignored is not the same as removing it).

### Step 5 — Implement lock-scope corrections

Fix X-05 (`checkLive` / `claim` boundary). Audit every other check-then-act sequence for the same pattern.

### Step 6 — Implement fault-injection harness and add tests

Add `FaultInjector` to the testkit. Add the fault-injection and invariant tests identified in the design document. The tests must fail against the pre-remediation code and pass against the remediated code. Demonstrate this in the evidence register.

### Step 7 — Remediate MEDIUM findings that share the root cause

X-06, X-08, X-11, X-12, X-13, X-14, X-15. Each should be resolvable by a transition primitive (X-06: presence-verified transition in `Signal`; X-08: attach as a state machine in the probe; X-11: mandatory Recv-timeout as a transition policy; X-12: settlement transition on completion; X-13: refusal transition; X-14: shutdown transition; X-15: authorization-gated transition on `kill_switches`).

Where a MEDIUM finding is genuinely a distinct class of problem (X-07 provider counting, X-09 affinity discrimination, X-10 drain visibility, X-16 analyzer coverage, X-17 CI DB-service, X-18 tree digest), address it in this cycle if bounded; note it explicitly in the evidence register if deferred with reasoning.

### Step 8 — Re-audit

Produce `docs/phase1/classd-state-transition-reaudit.md` that:

- Maps every X finding to the change that addresses it.
- For each HIGH finding, argues why the structural change makes the finding impossible rather than merely absent.
- For each MEDIUM finding, states whether it is fixed, mitigated, or deferred, with reasoning.
- Lists any findings from the review that are NOT addressed and why.
- Lists any new concerns that the remediation itself raised.
- Cites the specific tests (by name and file) that verify each claim.

This re-audit is the primary artifact for the next adversarial review. It must be honest.

## 7. Test discipline

Every test added or modified under this prompt must:

- Run against real PostgreSQL.
- Be runnable in CI (either against a service container or against the configured test database).
- Use barrier synchronization for concurrency, not sequential submit.
- Use `FaultInjector` for failure-path tests.
- Assert the invariant, not the implementation.
- Fail loudly if the invariant is violated, rather than passing because the failure path was not exercised.
- Document, in a comment, the finding it addresses.

### Required test additions (minimum)

| Test | Addresses | Asserts |
|---|---|---|
| `TestApprovalRetryAfterSubmitFailure` | X-01 | Retry after `Approver.Submit` fails re-enters approval; never executes unapproved |
| `TestApprovalRetryAfterParkFailure` | X-01 | Retry after `WaitingApproval` update fails re-enters approval; never executes unapproved |
| `TestApprovalPollSubmitsIfMissing` | X-02 | Poll converges; grant takes effect after a park-time submit failure |
| `TestApprovalDenialSettlesAfterParkFailure` | X-02 | Denial settles a stranded invocation |
| `TestConcurrentSubmitSameKey` | X-03 | Barrier-synced duplicates produce one workflow; loser adopts winner's `run_id` |
| `TestConcurrentApproveReject` | X-04 | Barrier-synced approve+reject: exactly one wins, loser errors |
| `TestConcurrentDecideAndCancel` | X-04 | Decision after cancellation is refused |
| `TestCancelDuringClaim` | X-05 | Cancellation during claim (gated by fault injection) prevents the send |
| `TestCancelDuringAttempt` | X-05 | Cancellation during `ATTEMPTING` observed via in-flight report |
| `TestSignalUnknownRun` | X-06 | Signal for nonexistent run returns error |
| `TestSettlementAuditedWithoutPoll` | X-12 | Settlement event emitted on workflow completion, not only on Status |
| `TestRefusalAudited` | X-13 | Kill-switch/revocation/token refusals appear in ledger and audit |
| `TestShutdownFailurePreservesPool` | X-14 | Shutdown failure does not silently close pool with active workers |
| `TestKillSwitchNotHandlerWritable` | X-15 | Handler tx cannot flip kill switch |

Every one of these tests must fail against the pre-remediation code. This is the verification that they address the finding.

## 8. Constraints and prohibitions

### Do not

- Fix findings one at a time without the structural change.
- Introduce a transition primitive and leave ad-hoc writes for "special cases."
- Add fault-injection tests that pass against the pre-remediation code (they must fail first).
- Strengthen existing tests to make them pass — fix the code, not the test.
- Add "temporary" bypasses, feature flags, or environment-gated paths to the affected subsystems.
- Change the public interface of the framework.
- Change the manifest or contract schemas.
- Modify `lead-followup` ADR or implementation.
- Modify `plan4.2.md`, the G0 adjudication, or any immutable document.
- Commit or push.
- Approve the `lead-followup` ADR or register the workflow.
- Address findings not in scope by "while I was there" changes — note them and stop.
- Defer a HIGH finding to a later cycle without an owner decision recorded in the re-audit.

### Do

- Report any conflict between this prompt and `plan4.2.md` as a conflict, not as a reconciliation.
- Report any finding where the root-cause analysis is wrong or the proposed structural change is insufficient.
- Report any new finding the remediation surfaced.
- Report any test that cannot be made to fail-then-pass.
- Report any state machine whose transitions cannot be predicated as specified.
- Keep the working tree clean except for the two required documents and the code/test changes.
- Bind every claim in the re-audit to a specific test file and test name.

## 9. Required documents

Create exactly two new documents (in addition to code and tests):

1. **`docs/phase1/classd-state-transition-discipline.md`** — the design and root-cause document from Step 1.

2. **`docs/phase1/classd-state-transition-reaudit.md`** — the re-audit from Step 8.

Both must be substantive. Neither may be a summary of the other.

Do not create additional documents. If a finding warrants its own write-up, include it as a section in the re-audit.

## 10. Evidence register requirements

Update `docs/evidence/README.md` with:

- A row for the state-transition discipline document, citing source tree digest (see §11).
- A row for each new or strengthened test, with source tree digest, environment, command, pass/skip/fail counts, and the finding addressed.
- A row for the re-audit document.
- An explicit digest for the working tree at the time of the evidence entries.

**The undigested tree problem (X-18) must be fixed in this cycle.** The pre-remediation evidence cited "`5cf9627` + prompt13 tree" with no digest. Every evidence entry from this cycle must carry either a commit SHA (if you commit, which you must not) or a working-tree digest computed via `git stash create` (record the resulting SHA even if you do not check the stash in) or a file-hash manifest written into the evidence register. Pick one method and document it.

## 11. Verification requirements

Before producing the re-audit:

- Run `gofmt -l .` — must be clean.
- Run `go vet ./...` — must be clean.
- Run `go test ./...` with a real PostgreSQL — every test must pass, none may skip except where the environment genuinely cannot provide the DB (and every skip must be documented).
- Run `scripts/checks.sh` — must pass.
- Run the specific fault-injection tests in a demonstration that they fail against the pre-remediation code. Record this in the evidence register: "Test X fails against SHA Y, passes against SHA Z."
- Run the concurrency tests with `-race`.
- Confirm the hard stop holds: `grep -r "lead-followup" --include="*.go" --include="*.json"` must return only comments and doc references, no implementation.

## 12. Ongoing principle (mandatory for all future cycles)

**This prompt establishes a rule that applies to every subsequent adversarial review cycle.**

After each adversarial review, the implementation cycle that follows must:

1. **Diagnose before fixing.** Identify the structural pattern that produced the findings. Do not accept the finding list as the work list. Produce a written root-cause diagnosis as the first deliverable of the cycle.

2. **Ask: is this finding a symptom or a class?** If multiple findings share a cause, the remediation is the cause, not the symptoms. Fixing symptoms without the cause is prohibited.

3. **Establish discipline, not patches.** Every remediation must leave behind a structural mechanism (primitive, harness, invariant test) that makes the class of finding impossible or caught, not just absent.

4. **Demonstrate the discipline addresses the findings.** The re-audit must argue why each finding is impossible or caught under the new discipline, not merely list the tests that were added.

5. **Make the discipline reviewable.** Every cycle produces a design document and a re-audit document. Future reviewers check whether the discipline was actually established, not whether findings were closed.

This principle applies to this cycle and every future cycle. It must be recorded in the evidence register as a standing practice, cited by each subsequent cycle's re-audit.

## 13. Acceptance criteria for this cycle

The cycle is complete when:

1. The state-transition discipline document is written and internally reviewed.
2. The transition primitive is implemented and tested for each of: invocations, approvals, effects, runs, sequences.
3. Every ad-hoc durable write in those subsystems is replaced with a `transition()` call.
4. Retry paths dispatch on observed state (`confirmSubmit`, `pollApproval`, effect claim, and any other retry/recovery path).
5. X-05 lock scope is corrected.
6. The fault-injection harness exists and is used by at least the tests enumerated in §7.
7. Every test enumerated in §7 exists, fails against pre-remediation code, passes against remediated code, and is cited in the evidence register.
8. The root-cause-sharing MEDIUM findings (X-06, X-08, X-11, X-12, X-13, X-14, X-15) are remediated or explicitly deferred with owner-grade reasoning in the re-audit.
9. The non-root-cause MEDIUM findings (X-07, X-09, X-10, X-16, X-17, X-18) are addressed or explicitly deferred with reasoning.
10. The re-audit maps every X finding to its disposition and cites the tests that verify each claim.
11. The evidence register includes source tree digests for every new entry and fixes the X-18 undigested-tree problem for future evidence.
12. The hard stop on `lead-followup` holds: no implementation, manifest, registration, execution, or ADR approval.
13. No immutable document is modified.
14. Only two new documents are created (`classd-state-transition-discipline.md`, `classd-state-transition-reaudit.md`) plus code and test changes.
15. `gofmt`, `go vet`, `go test ./...` with a real PostgreSQL, and `scripts/checks.sh` all pass.
16. The working tree is clean except for the intended changes.
17. No commits, no pushes.

**Do not declare the cycle complete until all seventeen are satisfied.** If any cannot be satisfied, the cycle is incomplete, and the specific blocker must be documented in the re-audit with an owner-decision request rather than worked around.

## 14. What success looks like

Success is not "the findings are closed." Success is:

- A reviewer reads the re-audit and can independently verify, from the state-transition discipline document, that the five HIGH findings are **structurally impossible** rather than merely absent from the current test suite.
- A future adversarial review of a different framework subsystem finds a similar class of finding, and the discipline in this cycle's design document is the template that catches it before it becomes a HIGH finding.
- The `FaultInjector` harness is used by every subsystem going forward, not just the ones fixed in this cycle.
- The "find the root cause before fixing" principle survives in the evidence register and is cited by every subsequent cycle's re-audit.

If the remediation produces a framework that no longer has the five HIGH findings but is structurally identical (same ad-hoc writes, same unpredicated updates, same non-re-entrant retries, same narrow lock scopes, same untested failure paths), the cycle has failed regardless of test outcomes.

The framework's claim to be Class D-ready must be defensible at the structural level, not at the test-coverage level. This cycle is where that structural defensibility is established.

## 15. Final instruction

Begin with the root-cause diagnosis and design document. Do not write a single line of remediation code until the design document is written and reviewed.

If the design document reveals that the root-cause analysis in §2 is wrong or insufficient, report that finding rather than proceed. The structural discipline is the deliverable; the individual fixes follow from it.

Stop after producing the two required documents and the code/tests that satisfy §13. Do not start the next cycle's work. Do not address findings outside the scope of §4.

The hard stop on `lead-followup` remains in force throughout.