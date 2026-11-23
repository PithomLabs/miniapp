# **Miniapp roadmap — where we are and what's next**

**Phase 1 — Foundation**

COMPLETE · CLEAR

Independently reviewed under Prompt 21. Lifecycle integrity, approval waiting, Class A single execution, transaction semantics, and concurrency behavior were accepted.

**Phase 2 — Effect-time authorization**

OWNER APPROVED · GO

The active workstream. Implement the canonical `authorizeAttempt` boundary so no consequential effect can proceed without fresh authorization.

**Phase 3 — Class B and C execution**

FUTURE · NOT AUTHORIZED

Extend the execution framework to independently retryable jobs and bounded multistep/batch workloads.

**Phase 4 — Reliability and operational maturity**

FUTURE · NOT AUTHORIZED

Address deferred reconciliation, audit recovery, quarantine operations, and related reliability capabilities.

## **1. What happens now: Phase 2**

The approved Prompt 22 work order divides the work into six stages.

1. **Stage 0 — Baseline and source audit**

   Establish the exact repository state, inventory effect paths, inspect the existing ledger and authorization mechanisms, and verify PostgreSQL 19. Confirm or refute the suspected structural gaps:
   - Settlement still using two transactions.
   - Missing production sequence/generation fencing.
   - Missing grant-snapshot and revocation safeguards.
   - Floating PostgreSQL image tags in Compose and CI.
2. **Stage 1 — Resolve P21-002**

   Fix the pending-plus-attached-workflow stuck state. Reproduce it through the actual restoration path, establish consistent lifecycle semantics, and test `Perform`, cancellation, and status synchronization.

   This is a hard gate: authorization must not rely on pending-state semantics until this is resolved.
3. **Stage 2 — Contract maintenance and authorization design**

   Complete the missing Class A lifecycle contract rows, correct the stale completion-test target, and write the authorization design.

   Map every authorization check to its authoritative data, lock, transaction, and regression test. Prove compatibility with the existing Class A transaction.
4. **Stage 3 — Implement&#x20;**`authorizeAttempt`

   Build the canonical authorization boundary. It must verify invocation liveness, persisted approval, authority, grants, policy controls, effect eligibility, lease, and generation/fencing ownership.

   All checks and `PENDING → ATTEMPTING` must commit atomically in a separate transaction. Provider calls happen only afterward.
5. **Stage 4 — Adversarial verification**

   Exercise concurrent workers, both cancellation winner orders, stale generations, lease turnover, crash boundaries, recovery, and every provider-initiation path.

   Verify the Class A `Perform` integration and run the complete PostgreSQL 19 test battery, including DBOS and race tests.
6. **Stage 5 — Documentation and closeout**

   Update contracts, design documents, operational guidance, and evidence registers. Produce `docs/phase2/prompt22-phase2-implementation-report.md`.

   Stop and await a separate owner instruction before Phase 3.

These stages and their acceptance requirements are defined in the approved work order.&#x20;

Pasted markdown(20261123-101334).md



## **2. What Phase 2 must actually deliver**

The central architectural milestone is this:

## **The effect authorization boundary**

**Workflow requests an effect**

Submission is not authorization

`authorizeAttempt`

Revalidate invocation, approval, authority, grants, kill switches, effect token, lease, and generation.

One atomic PostgreSQL transaction

**Commit&#x20;**`PENDING → ATTEMPTING`

Authorization is now committed

**Call external provider**

After transaction commit

**Record outcome or reconcile ambiguity**

Never blindly retry UNKNOWN

That boundary is the real purpose of Phase 2. It prevents a previously admitted workflow from using stale authority to initiate an effect after approval, policy, invocation state, or ownership has changed.

The seven reviewed constraints C1–C7 remain binding throughout implementation.&#x20;

adv_review_prompt21.md



## **3. What comes after Phase 2**

Once Phase 2 passes its own acceptance review, the broader roadmap resumes.

| Future workstream                    | Purpose                                                                                             |
| ------------------------------------ | --------------------------------------------------------------------------------------------------- |
| **Phase 3 — Class B**                | Independently retryable one-shot jobs, with River as the default execution mechanism.               |
| **Phase 3 — Class C**                | Bounded batch and multistep operations; choose River or DBOS according to workload requirements.    |
| **Phase 4 — Reliability operations** | Durable reconciliation, audit recovery, quarantine handling, and operational recovery capabilities. |

These are future architectural workstreams, not part of the current authorization. Prompt 22 explicitly prohibits starting Phase 3 as part of its closeout.&#x20;

## **4. The engineering rules that apply throughout**

Keep these decisions visible as the project grows:

- **PostgreSQL 19 everywhere:** local Docker, development, integration tests, crash tests, and CI. Pin the same immutable image digest.
- **Framework-enforced reliability:** no delegation of authorization or lifecycle guarantees to individual miniapps.
- **No hidden bypasses:** every consequential provider path uses the canonical boundary.
- **No false exactly-once claims:** database execution guarantees do not imply exactly-once external delivery.
- **Evidence before acceptance:** actual PostgreSQL execution, reproducible red/green tests, exact digests, and explicit residuals.
- **Bounded scope:** no unrelated redesign or premature implementation of later phases.

## **5. The immediate next action**

The next task is **Prompt 22, Stage 0**—not coding `authorizeAttempt` immediately.

The first output should be a verified baseline and source map, with the four suspected gaps confirmed or rejected. Then proceed to P21-002.

**The strategic progression is:**

1. Phase 1 proved the framework can control admission, lifecycle, and transactional execution.
2. Phase 2 proves the framework can control whether an external effect is authorized at the exact moment it is about to happen.
3. Phase 3 expands execution capabilities without weakening those guarantees.
4. Phase 4 strengthens recovery and operational maturity.

The destination remains the same: **a miniapp framework where developers focus on business logic while the framework takes responsibility for execution reliability, authorization, and recovery.**