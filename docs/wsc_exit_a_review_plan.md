**Both decisions approved. Phase 1–5 are authorized under the conditions below. WS-A remains the active workstream; WS-B must not start yet.**

## **1. Owner decision: Rule 3 amendment**

Approved

Adopt the review's revised wording verbatim into §2 and §5:

> A gated workflow may be submitted with a valid pending approval linkage only to enter its durable approval-wait path. It may proceed to consequential business execution only after the framework verifies a persisted granted decision with matching authority and intent. The effect-authorization boundary independently revalidates that decision. Missing, denied, expired, cancelled, or mismatched approval linkages fail closed.

The distinction is now normative:

| Permission         | Required condition                       | Permitted behavior                                          |
| ------------------ | ---------------------------------------- | ----------------------------------------------------------- |
| Workflow-start     | Valid persisted pending approval linkage | Submit workflow solely to enter durable approval wait       |
| Business-execution | Persisted, verified granted decision     | Authorize consequential business effects after revalidation |

A pending approval is never a business-execution authorization.

The pending linkage must be bound to the correct tenant, invocation, tool and intent. A signal, cached workflow value, or caller-supplied approval ID alone is not sufficient evidence of approval.

## **2. Effect-boundary revalidation**

Specify now; implement in WS-B

Do not block WS-A exit on a mechanism owned by WS-B.

Add the requirement to the contract now. WS-B must implement it in the `PENDING → ATTEMPTING` authorization transaction.

That transaction must independently verify:

- Approval record exists and is `decided/approve`.
- Approval belongs to the correct tenant and invocation.
- Tool and intent hash match the persisted invocation.
- Approver authority is valid.
- Approval has not expired or been cancelled.
- All other existing authorization conditions remain satisfied.

The check must occur inside the same authorization transaction as the effect transition. No cached approval decision or workflow-local state may substitute for it.

**WS-A must not claim this effect-boundary guarantee is implemented or proven.** WS-A proves correct workflow admission and approval waiting. WS-B proves consequential-effect authorization.

## **3. Approved execution plan**

I approve the five phases, with these additional precision requirements.

**Phase 1 — Contract amendment**

Record the two permissions in §2 and §5, amend the Rule 3 transition language, and reference WS-B's authorization-transaction requirement. No implementation behavior changes during this phase.

**Phase 2 — Entry-path proofs**

Approve the four deterministic proofs. For first submission and replay, the observation marker must be placed at the actual approval-wait entry, immediately before the durable wait. The test must establish that the workflow remains live and no provider send occurs while approval is pending.

The marker is evidence of reaching the wait branch, not by itself proof that the workflow has successfully suspended in `Recv`. Use the DBOS status or another authoritative wait-state observation where available.

Poll while pending must produce neither another submission nor a provider send. Rejection and expiry of a never-submitted run must remain non-submitting.

**Phase 3 — X-03 deterministic red**

Approve the DB lock-gate design and the two-racer, three-round test.

One important requirement: the wait observation must prove that both racers are blocked by the designated holder transaction on the intended invocation row. Counting unrelated `pg_stat_activity` lock waiters is not sufficient.

The red must reliably demonstrate two distinct run-ID writes and divergent workflows. The green must demonstrate one canonical workflow and adoption by the loser.

**Phase 4 — X-04 deterministic red**

Approve the approval-row gate.

For approve/reject, establish that both racers reached the intended blocked boundary before releasing the holder. Preserve identical test source across red and green.

Keep the 20-round Decide/cancel test and the sequential `TestDecideAfterCancelRefused` guard test. Record the difference between deterministic overwrite evidence and genuinely race-dependent ordering. Continue treating self-approval and expiry tests as guard tests where they pass on both snapshots.

**Phase 5 — Evidence and resubmission**

Approve fresh isolated red copies, fresh databases, harness hashes, identical-source verification, green reruns, full suite, static gates, evidence-register updates and the final stop for WS-A exit review.

No WS-B implementation work during this sequence.

## **4. Final boundaries**

The approved changes resolve the Rule 3 ambiguity without redesigning the submission flow.

The acceptance criteria remain:

1. Pending approval can start a workflow only to enter its durable wait path.
2. No consequential effect can be authorized without a persisted grant.
3. Approval is revalidated at the effect authorization boundary.
4. Every entry path obeys the same permission distinction.
5. Red/green tests demonstrate the original concurrency defects rather than merely exercising concurrent code.

**Proceed with Phase 1, then Phases 2–5. Stop and submit the WS-A exit evidence for review. WS-B remains gated until that review is complete.**