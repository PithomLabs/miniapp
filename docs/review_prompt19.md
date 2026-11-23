# **Owner disposition — Prompt 19**

PHASE 2 REMAINS BLOCKED

I accept the **NOT CLEAR** verdict. The review has identified real gaps in the release criteria, but I agree with its assessment that the core predicated-transition mechanism is sound where it has been tested.

I authorize a bounded Phase 1 closeout. I am making the following owner decisions so implementation can proceed without further clarification.

## **1. HIGH-1 — approve the race-requirement rescope**

**Decision: Replace the unreachable Cancel-versus-completion requirement with settlement-versus-completion.**

The review establishes that Class A in-flight invocations have no `run_id`, while public `Cancel` addresses records by `run_id`. Consequently, the originally specified cancellation race cannot occur through that API. We must not modify the public cancellation API merely to manufacture a test scenario.&#x20;

adv_review_prompt19.md



The revised requirement is:

**Prove that&#x20;**`completeSync`**&#x20;behaves correctly when approval settlement and synchronous execution completion compete for the same invocation.**

Require two deterministic winner scenarios:

| Scenario | Winning transition                | Required losing behavior                                                                                        |
| -------- | --------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| A        | Approval denial/expiry settlement | Completion discards its handler result, returns authoritative terminal state, emits no completion-success audit |
| B        | Synchronous completion            | Approval settlement reloads and reports the completed state, without emitting a false denial/expiry audit       |

Use the actual Class A granted execution path, a channel-blocked handler, and the established PostgreSQL lock-gate methodology.

Both winner orders must be deliberately established and tested. Three repetitions of an uncontrolled race are not sufficient proof that both outcomes were exercised.

Also require:

- API-level coverage of the reachable Class A execution path.
- Store-level terminal-immutability coverage for attempts to cancel an already-completed record.
- Correction of the `completeIfLive` comment so that it describes the actual competing settlement path, not public Cancel.

Do not claim that the store-level cancellation test proves a public Class A cancellation capability.

This is an explicit owner amendment to Prompt 17's original Phase 1 race requirement.

## **2. MEDIUM-1 — approve the CR-05 correction**

Mandatory closeout

I approve the reviewer's proposed correction.&#x20;

adv_review_prompt19.md



**A.&#x20;**`failIfWaiting`

Remove the `run_id` write entirely.

Approval settlement owns status and error transitions. Workflow identity ownership belongs to the adoption mechanism. A settlement operation must not write an identity value captured from an earlier read.

**B.&#x20;**`adoptRunID`

Add a database-enforced live-state predicate.

The permitted states must include `pending`, because restoration dispatch can adopt from that state. Terminal and quarantined records must be rejected.

The service must also stop submission when adoption loses and the authoritative record is no longer eligible. A store predicate is the primary guarantee; the service-level check is defense in depth.

**C. Required regression tests**

1. A stale expiry poll loses its race after another path adopts run `R`. It must preserve `R`.
2. Adoption against a terminal record must fail.
3. A failed adoption must not launch a DBOS workflow.
4. Both relevant interleavings must be deterministically exercised.

Assert persisted identity, status, and actual workflow-submission count.

## **3. MEDIUM-2 — choose framework-enforced single execution**

Mandatory architectural correction

**Decision: Preserve the no-re-execution idempotency guarantee. Do not downgrade Class A to at-least-once handler execution.**

The review identifies a serious gap: concurrent granted polls, or duplicate dispatches, can execute the same Class A handler more than once even though the invocation record ultimately has one winning terminal transition. Direct database writes by the handler can therefore happen twice.&#x20;

adv_review_prompt19.md



The framework must own execution mutual exclusion.

I authorize a framework-managed, predicated execution-claim mechanism. However, do not implement a superficial `waiting_approval → running` update without accounting for crashes.

Before implementation, produce a short design addendum answering:

- What is the atomic execution-claim boundary?
- How do concurrent polls and duplicate dispatches establish one winner?
- How does the loser observe the authoritative result?
- What happens if the process dies after claiming but before invoking the handler?
- What happens if the handler transaction commits but the caller loses its response?
- Can a retry execute business writes a second time?
- How does the mechanism behave across separate processes, not merely goroutines?
- How are pending, running, waiting-approval, and terminal records distinguished?

The solution must preserve the Class A transactional guarantees and must not introduce an unrecoverable claimed-but-never-executed state.

If that requires moving claim, business transaction, and lifecycle completion into a coordinated transaction boundary, analyze that explicitly. Do not decide the architecture merely by choosing a convenient status transition.

**Required test:** deterministically race two granted polls and duplicate same-key dispatches using a channel-blocked Class A handler. Assert one handler execution, one coherent terminal transition, and correct audit evidence.

Include crash and recovery tests for any newly introduced claim state.

Do not transfer execution-idempotency responsibility to individual miniapp authors.

## **4. MEDIUM-3 — correct Prompt 17 before Phase 2**

**Decision: Mandatory documentation correction.**

The advisory-last lock ordering in `docs/prompt17.md` and its plan-review mirror contradicts the corrected lifecycle contract.&#x20;

adv_review_prompt19.md



Correct both documents:

- Make lifecycle contract §4 authoritative.
- Remove the instruction to acquire an advisory lock after PostgreSQL transactional locks.
- Explicitly state that the Phase 2 `authorizeAttempt` transaction does not acquire an advisory lock.
- Preserve the outermost-first rule for operations that actually require advisory locking.
- Record the correction under the never-silent amendment mechanism.

Run a repository-wide search for remaining advisory-last instructions. Phase 2 must not inherit contradictory instructions.

## **5. LOW findings — disposition**

These are not independent Phase 2 release blockers, but I want the inexpensive corrections included in the closeout.

| Finding                                              | Owner decision                                                                                                                           |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| LOW-1: misleading `duplicate` audit                  | Fix. Emit the marker only when the claim actually identified an existing invocation. Add a losing-first-claim test.                      |
| LOW-2: missing audit backfill after post-win failure | Accept as a residual limitation. Record the future X-12 audit-recovery requirement. No sweeper or effects reconciler in this workstream. |
| LOW-3: contract phasing precision                    | Correct the future X-05 markers, claim-response timing description, and explicit provenance for the advisory-lock wording.               |

The audit-backfill issue must not be represented as solved by M-01's winner-only audit placement.&#x20;

adv_review_prompt19.md



## **6. INFO findings — disposition**

| Finding                                                   | Owner decision                                                                                                                                                                                                 |
| --------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| INFO-1: undefined live set                                | Define nonterminal states separately from operation-specific live/actionable state predicates. Explicitly document why `pending` is accepted by completion but excluded from certain run-addressed operations. |
| INFO-2: direct advisory-lock acquisition                  | Add a lightweight mechanical guard ensuring the production acquisition remains restricted to the Class A framework path. Avoid an unnecessary analyzer redesign.                                               |
| INFO-3: 32-bit `hashtext`                                 | Accept. Monitor contention; no change now.                                                                                                                                                                     |
| INFO-4: Prompt 19 completion-state table                  | Correct the future review documentation. The implementation's broader accepted states are intentional.                                                                                                         |
| INFO-5: unreachable defensive branches                    | Record for the later quarantine workstream. Do not invent current producers or change behavior now.                                                                                                            |
| INFO-6: digest provenance                                 | Adopt exact commands, working directory, and path forms in all future evidence.                                                                                                                                |
| INFO-7: run-less Class A cancellation and trust asymmetry | Document that Class A records are not individually addressable through the current run-ID cancellation API. Document the effective-trust behavior and preserve the verified fail-closed boundary.              |

The review's observations support these dispositions.&#x20;

adv_review_prompt19.md



## **7. Verification requirements**

The independent review ran on PostgreSQL 16.15 rather than the PostgreSQL 19.0 environment specified in the work order. The substitution is honestly documented, and the reviewer provides a rationale based on the exercised SQL and locking semantics.&#x20;

adv_review_prompt19.md



For the closeout, I require the new concurrency tests to run on local PostgreSQL 19.0 as well, where the authorized environment permits.

Retain all previous evidence requirements:

- Frozen Phase 1 red and green copies.
- Byte-identical regression-test sources.
- Per-finding red/green results.
- Deterministic gates with cleanup on every path.
- Persisted database state and audit assertions.
- Full serial test suite.
- Relevant race tests.
- `scripts/checks.sh`.
- Full reproducible digests and evidence-register pre-images.

A code change without a discriminating test does not close a finding.

## **8. Authorization and next gate**

PHASE 1 CLOSEOUT AUTHORIZED

The execution order is:

1. HIGH-1: amend the requirement and implement both deterministic settlement/completion winner proofs.
2. MEDIUM-1: fix `failIfWaiting` and `adoptRunID`, with both race regressions.
3. MEDIUM-2: produce the execution-claim design addendum, then implement framework-enforced single execution and its recovery tests.
4. MEDIUM-3: correct Prompt 17 and its plan-review mirror.
5. Close LOW-1 and LOW-3; record LOW-2 as deferred X-12 work.
6. Apply the INFO documentation and mechanical-guard dispositions.
7. Re-run the relevant red/green tests, PostgreSQL 19.0 concurrency tests, full serial suite, race tests, and checks.
8. Submit a focused closeout report for independent adversarial review.

**Phase 2 remains prohibited until the closeout receives a CLEAR independent review and explicit owner release.**

No effect-authorization implementation, commits, pushes, Neon access, or OpenCode access are authorized yet.

**One architectural point deserves emphasis:** M-01 established that the framework can protect lifecycle transitions from stale writes. MEDIUM-2 asks a different question: can the framework guarantee that the business handler itself executes only once under concurrent replay? We must settle that before designing effect authorization around invocation identity and lifecycle assumptions.

The principle remains unchanged: **the framework owns reliability; miniapps own business logic.**