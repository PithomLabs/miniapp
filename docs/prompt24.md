**My decision: call Independent Adversarial Review Gate 1 now. Do not continue expanding implementation yet.**

We have reached the review threshold we agreed on: the canonical authorization boundary is implemented, integrated into the reported live paths, and the atomic settlement rewrite is in place.

This is precisely when independent scrutiny is most valuable: before adding the remaining revocation tooling, crash tests, and broader verification on top of a potentially flawed authorization core.

I see no reason to stop the project, but I do see several important claims that need independent verification.

### **What I want challenged**

| Priority | Question                                                                                                               |
| -------- | ---------------------------------------------------------------------------------------------------------------------- |
| Critical | Is the sequence verifier framework-owned and impossible for an untrusted Intent to override?                           |
| Critical | Are all authorization checks and `PENDING → ATTEMPTING` genuinely atomic?                                              |
| High     | Does grant snapshot locking correctly cover the authorization read set without extending across provider calls?        |
| High     | Can `adopt()` be exploited to adopt another invocation's operation?                                                    |
| High     | Do transient refusals and terminal withdrawals preserve the approved semantics?                                        |
| High     | Does Class A `effect-contention` permit correct same-key recovery without stranding or falsely failing the invocation? |
| High     | Is atomic Settle genuinely crash-safe, including its audit and in-memory cleanup?                                      |
| High     | Can any production provider path bypass authorization?                                                                 |

The first question is particularly important. My previous instruction required the sequence verifier to be framework-owned, not a callback that individual callers can replace. The review must verify the actual implementation, not infer compliance from the presence of a sequence check.

## **Next prompt — Independent Adversarial Review Gate 1**

# Prompt 24 — Independent Adversarial Review: WS-B Phase 2 Authorization Core (Gate 1)

**Repository:** `github.com/PithomLabs/miniapp`\
**Review target:** Prompt 23 Stage 3A authorization core implementation\
**Governing architecture:** Prompt 15.2 / Plan 4.2\
**Governing work order:** Prompt 22\
**Approved design:** `docs/phase2/authorize-attempt-design.md`\
**Previous independent review:** Prompt 21 — CLEAR\
**Review type:** Independent, adversarial, read-only\
**Purpose:** Establish whether the new authorization core is structurally sound before the remaining Stage 3A implementation expands.

---

## 1. Mission

Act as an independent senior Go/PostgreSQL security and distributed-systems reviewer.

You did not implement this code.

Your job is not to validate the implementer's narrative or confirm that the tests pass. Your job is to find ways the implementation could violate its authorization, ownership, transaction, lifecycle, concurrency, or recovery guarantees.

Assume that green tests may be incomplete, incorrectly staged, or insufficiently discriminating.

Review the actual repository, source, contract, design, test implementations, and available execution evidence.

The central invariant is:

> No external provider call can begin unless the framework has atomically revalidated authoritative state and committed the corresponding effect's transition to ATTEMPTING.

Your review must determine whether this invariant is enforced by the architecture, not merely followed by current callers.

The framework owns reliability; miniapps own business logic.

---

## 2. Review scope

Review the implementation currently present in the repository.

At minimum inspect:

- `effects/authorize.go`
- Effect service and ledger implementation.
- Approval transaction-scoped read implementation.
- `contract` shared intent hashing and schemas.
- `auth.Policy` snapshot mutex and accessors.
- Class A executor and runtime error mapping.
- `adopt()` and all operation ownership paths.
- Sequence verification abstraction and trusted context propagation.
- Cancellation and withdrawal handling.
- Atomic Settle implementation.
- Every production provider-initiation path.
- Relevant tests, fixtures, and evidence-register entries.
- Updated lifecycle contract.
- `docs/phase2/authorize-attempt-design.md`.

Read Prompt 21 and Prompt 22 in full before assessing compliance.

Do not rely on the implementation report alone.

---

## 3. Verify exact baseline and provenance

Record:

- Repository and branch.
- HEAD commit.
- Full source-tree digest.
- Worktree status.
- Design and contract digests.
- Relevant changed-file inventory.
- PostgreSQL version.
- Full immutable Docker image digest.
- Go version and architecture.
- Exact test commands and working directories.

Verify that PostgreSQL 19 is actually being used.

The mandatory baseline is PostgreSQL 19.0, Debian build `19.0-1.pgdg13+1`, using the project's pinned immutable image identity.

Do not substitute another major version.

Identify which tests were executed, which were skipped, and which are not yet implemented. Never count an absent test as a passing test.

Do not modify production code, tests, contract, or existing evidence.

Temporary review artifacts must remain outside the repository.

Create only the independent review report inside the repository.

No commits or pushes.

---

## 4. Priority 1 — Authorization transaction correctness

Read `authorizeAttempt` end to end.

Reconstruct the actual execution sequence and answer:

1. Is the authorization operation performed in its own transaction and pool checkout?
2. Does it acquire any advisory lock, directly or indirectly?
3. Are all required checks performed using authoritative persisted state?
4. Does it lock the required rows in the correct order?
5. Does it check affected-row counts on contested transitions?
6. Can any check occur outside the transaction and become stale before the effect transition?
7. Can any path call the provider before the authorization transaction commits?
8. Can authorization return success without a committed ATTEMPTING transition?
9. Can a failed transaction leave an in-memory state that falsely implies authorization?
10. Are transaction rollback and resource release correct on every refusal and error path?

Verify this exact order:

`grant RLock → invocation → approval → sequence/generation → policy controls/grants → effect`

The authorization transaction must take no advisory lock.

Inspect query-level locks, not just function call order.

Check whether helpers hide extra queries, locks, retries, callbacks, or transactions.

Construct an explicit wait-for graph for:

- Class A execution.
- Effect authorization.
- Cancellation.
- Approval decision.
- Settlement.
- Lease takeover.
- Sequence supersession.
- Grant setter/revocation.

Identify any constructible deadlock cycle.

A statement that the lock order is correct is not sufficient. Show the actual lock acquisition paths.

---

## 5. Priority 2 — Sequence verifier trust boundary

This is a specific owner concern and must be examined rigorously.

The approved design requires effects to remain independent of probe-owned tables.

However, the implementation must not allow an ordinary caller or miniapp to invent its own sequence authorization.

Inspect the actual sequence abstraction.

Determine:

1. Where is the sequence verifier created?
2. Who owns it?
3. Can an arbitrary Intent supply or replace its callback?
4. Can a miniapp provide a no-op verifier?
5. Can a caller change the sequence identity after the trusted workflow context establishes it?
6. Is the identity bound to tenant, invocation, and effect?
7. Is the identity sourced from trusted framework workflow context?
8. Does the verifier use the existing authorization transaction?
9. Does it execute at lock-order position 3?
10. Can it acquire an advisory lock, perform external I/O, or start another transaction?
11. What happens when the identity is missing, invalid, or mismatched?
12. What happens when a required verifier is unavailable?

The required semantics are:

- A non-sequence effect may omit M5 fencing only when its framework-controlled classification establishes that it is not sequence-bound.
- A sequence-bound effect must have trusted identity and trusted verification.
- Missing or invalid verification fails closed.
- An ordinary Intent cannot downgrade a sequence-bound effect into an unfenced effect.

**Blocking criterion:** If the sequence callback is caller-controlled, overridable by an Intent, or can silently become a no-op, report a HIGH or CRITICAL finding depending on exploitability. Do not accept it as a test-only weakness.

---

## 6. Priority 3 — Approval and intent binding

Verify the complete persisted approval relationship.

Challenge:

- Tenant binding.
- Invocation binding.
- Tool and operation binding.
- Authority principal.
- Intent hash.
- Approval status.
- Expiry.
- Approval linkage integrity.
- Agent principal versus authority principal.

Verify that the shared `IntentHash` implementation has one canonical definition.

Inspect the move from runtime to contract. Verify that all producers and consumers use the same canonical encoding and that no divergent legacy implementation remains.

Check pending approval behavior carefully:

- Pending approval may signal or enter the durable wait path.
- It cannot authorize a provider call.
- No database transaction or row lock may remain held during a workflow wait or signal wait.
- Denied, expired, cancelled, and mismatched approval outcomes must be correctly classified.
- A malformed or missing linkage must fail closed.

Verify that the new approval `FOR SHARE` implementation does not accidentally weaken the established approval-decision transaction.

---

## 7. Priority 4 — Grant snapshot and revocation

Inspect the actual `auth.Policy` implementation.

Verify:

- The mutex is owned by the Policy object.
- Pointer-receiver migration prevents accidental copying of mutex-bearing state.
- All grant reads use the protected accessor.
- Credential resolution uses the protected accessor.
- All mutation goes through the setter.
- No production map write bypass exists.
- Authorization holds the read lock for the full required authorization read set.
- The lock is released on all success, refusal, panic, and error paths.
- The lock is released before the provider call.
- Setter/revocation does not invert database lock order.

Review the configuration hash:

- Canonical and deterministic serialization.
- Includes all authorization-relevant grant data.
- Excludes credentials and secrets.
- Changes when an authorization-relevant grant changes.
- Does not change due to irrelevant map iteration order.

The remaining `/healthz` and `verify-revocation` work is not required to be implemented for this Gate 1 source review, but verify that the current core does not make misleading revocation guarantees.

Confirm that the ten-minute maximum staleness and cross-replica deployment requirements remain explicit.

Report any gap between the claimed revocation model and the actual core.

---

## 8. Priority 5 — Effect ownership and confused-deputy protection

Inspect `adopt()` and every adoption/takeover path.

Try to construct these attacks:

- Tenant A adopts Tenant B's operation.
- Invocation A adopts Invocation B's operation.
- One tool adopts another tool's effect.
- A different operation reuses a known BusinessOpID.
- A caller supplies a forged attempt lineage.
- An old lease holder adopts a new lease.
- A stale worker settles a newer owner's attempt.
- A terminal or settled operation is improperly revived.
- An authorization retry creates another logical business operation.
- A caller obtains access through a settled-read shortcut and then initiates a new provider operation.

Verify that all bindings are checked against authoritative persisted data.

Distinguish safe read-only settled-result access from permission to initiate or resume an operation.

Inspect all state transitions for affected-row checks.

Do not accept "the caller knows the operation ID" as proof of ownership.

Verify stable BusinessOpID derivation across retries and Class A recovery.

---

## 9. Priority 6 — Cancellation and withdraw semantics

Review every refusal reason and its disposition.

Use the owner-approved distinction:

| Condition                       | Required behavior                                               |
| ------------------------------- | --------------------------------------------------------------- |
| Temporary policy denial         | Refuse; retain PENDING                                          |
| Engaged kill switch             | Refuse; retain PENDING unless an explicit terminal rule applies |
| Grant revocation                | Follow the defined grant model; no invented withdrawal          |
| Infrastructure failure          | Retryable; retain PENDING                                       |
| Contention                      | Conflict; retain PENDING                                        |
| Stale worker                    | Reject stale owner                                              |
| Durable cancellation wins first | PENDING → CANCELLED                                             |
| Other terminal invalidation     | Withdraw only when explicitly defined by contract               |
| Authorization wins first        | ATTEMPTING; in-flight                                           |
| Ambiguous provider outcome      | UNKNOWN; no blind retry                                         |

Challenge specifically:

- Does a terminal invocation withdraw only its own effects?
- Can terminal approval refusal withdraw an operation whose approval binding does not match?
- Is lazy expiry incorrectly treated as an irreversible terminal decision?
- Can a mismatch destroy another owner's pending effect?
- Can an idempotent retry resurrect CANCELLED?
- Can a post-cancellation takeover authorize a new attempt?
- Can a refusal be audited as a provider attempt?
- Is every actual withdrawal accompanied by the correct `effect-withdrawn` audit?

Verify that quarantined invocations remain resumable under the established semantics and are not treated as irreversibly terminal.

Both cancellation-first and authorization-first behavior must be supported by real PostgreSQL evidence when such tests exist.

---

## 10. Priority 7 — Class A C6 integration

Review the fourth Class A outcome introduced for ledger contention.

Determine whether the mapping from effect-ledger contention to `effect-contention` and HTTP 409 preserves:

- Complete Class A transaction rollback.
- No FAILED settlement caused solely by contention.
- No committed business writes from the abandoned attempt.
- Stable BusinessOpID.
- Explicit same-key retry semantics.
- No automatic unbounded retry.
- No indefinite transaction hold.
- No orphaned invocation or operation.
- Correct behavior when a subsequent attempt finds the ledger operation settled, in progress, or stale.

Challenge whether the conflict is merely terminal at the request/workflow boundary or accidentally terminalizes the underlying invocation.

The distinction must be explicit.

A 409 response is not proof that the retry can safely converge.

---

## 11. Priority 8 — NOT_APPLIED retry semantics

The implementation changes `NOT_APPLIED` from a settled-read short-circuit into a fresh authorization path.

Independently verify:

- The contract permits this change.
- Only the correct `NOT_APPLIED` state is eligible.
- The same BusinessOpID is preserved.
- The previous attempt lineage is retained.
- A new attempt is authorized only after current invocation, approval, grant, switch, and ownership checks.
- A stale worker cannot resurrect an obsolete attempt.
- No provider call occurs before the new ATTEMPTING commit.
- A concurrent retry has one winner.
- A disagreement or unexpected state fails closed.

Challenge whether the new behavior can duplicate an operation that was actually applied but incorrectly classified as NOT_APPLIED.

Do not confuse provider reconciliation evidence with the authorization transaction.

---

## 12. Priority 9 — Atomic Settle

Review the new single-transaction Settle implementation.

Verify:

- One predicated state transition.
- State change and settlement audit commit atomically.
- No externally visible RECONCILING intermediate state remains in production paths.
- The operator identity is authenticated and server-derived, not trusted merely because a caller supplied a string.
- Repeated identical settlement is idempotent.
- Conflicting repeated settlement returns a defined terminal conflict.
- Missing audit backfill is explicit, authorized, and idempotent.
- In-memory map cleanup follows durable commit and cannot falsely imply settlement.
- Crash before commit changes nothing.
- Crash after commit leaves state and audit consistent.
- No network calls occur inside the transaction.

The four named crash tests are not yet reported complete. Do not count them as passed.

Assess the source architecture now and list the exact execution evidence still required.

---

## 13. Priority 10 — Production provider-path inventory

Independently enumerate all non-test code that can initiate provider work.

Include:

- `Perform`.
- `Channel`.
- Claim.
- Adoption.
- Takeover.
- Retry.
- Reconcile.
- Recovery/resume.
- Direct provider adapter calls.
- Workflow-specific send paths.

Verify that all consequential provider initiation reaches the same canonical authorization boundary.

Search for direct provider calls and alternate transitions into ATTEMPTING.

Verify that no adapter can perform the send based solely on an earlier liveness or approval check.

The reported claim that the probe workflow is the sole production Perform caller must be verified.

Distinguish current production callers from interface implementations that are test-only or future extension points.

A caller inventory alone is not proof of no bypass. Verify the call graph and relevant tests.

---

## 14. Evidence audit

Review existing tests and their claims.

Classify each relevant test as:

- Discriminating red/green.
- Green-only regression.
- Characterization.
- Source-only proof.
- Composition proof.
- Not yet implemented.
- Blocked/unverified.

Verify that red and green use identical test bytes where claimed.

Verify pre-fix snapshots and hashes.

Identify tests that pass vacuously, rely on sleeps, use goroutine counts instead of persisted database evidence, or fail to prove the claimed transaction boundary.

Run the focused available tests on PostgreSQL 19.

Do not demand that all remaining Stage 3A work be completed before this Gate 1 review. However, do not issue CLEAR if a foundational guarantee is asserted without sufficient evidence.

In particular, record the outstanding items honestly:

- Four Settle crash tests.
- Full authorization matrix.
- Both forced cancellation race orders.
- Full crash/recovery matrix.
- Grant hash, health, and revocation tooling.
- Full provider bypass tests.
- C6 Class A recovery verification.
- Final full-suite and CI parity.

These are still Stage 3A work unless the implementation has already completed them.

---

## 15. Mandatory C1–C7 assessment

Produce a table:

| Constraint | Required result                       | Source evidence | Executed test evidence | Verdict |
| ---------- | ------------------------------------- | --------------- | ---------------------- | ------- |
| C1         | No advisory lock in authorizeAttempt  | <br>            | <br>                   | <br>    |
| C2         | Correct global lock order             | <br>            | <br>                   | <br>    |
| C3         | Independent authorization transaction | <br>            | <br>                   | <br>    |
| C4         | No new advisory paths / guard intact  | <br>            | <br>                   | <br>    |
| C5         | P21-002 semantics respected           | <br>            | <br>                   | <br>    |
| C6         | Class A Perform recovery              | <br>            | <br>                   | <br>    |
| C7         | O-01 closed at effect boundary        | <br>            | <br>                   | <br>    |

Do not use a check mark merely because a symbol or method exists.

---

## 16. Required report and verdict

Create:

`docs/phase2/adv_review_prompt23_gate1.md`

Include:

1. Executive verdict.
2. Exact baseline and provenance.
3. C1–C7 compliance assessment.
4. Authorization transaction analysis.
5. Sequence verifier trust-boundary analysis.
6. Approval and intent-binding analysis.
7. Grant snapshot and revocation assessment.
8. Ownership/adoption attack analysis.
9. Withdrawal classification assessment.
10. C6 integration assessment.
11. NOT_APPLIED retry review.
12. Atomic Settle review.
13. Provider-path inventory and bypass analysis.
14. Test-discrimination assessment.
15. Findings with severity, root cause, trigger, impact, evidence, and discriminating test.
16. Required correction and bounded owner decision.
17. Explicit remaining Stage 3A work.
18. Accepted residuals and deferred work.

Use one of these verdicts:

**CLEAR** — No CRITICAL or HIGH foundational/structural finding remains. Remaining work is bounded and can safely proceed.

**NOT CLEAR** — At least one CRITICAL or HIGH finding compromises authorization, ownership, transaction correctness, fencing, or provider-path enforcement.

**INCONCLUSIVE** — A critical guarantee cannot be established due to missing source, inaccessible environment, or insufficient evidence.

A CLEAR verdict does not mean Stage 3A is complete. It means the reviewed core is safe to continue building on, subject to the remaining work and owner disposition.

Severity discipline:

- CRITICAL/HIGH: block progression.
- MEDIUM: require explicit owner disposition.
- LOW: track without reopening completed foundations.
- Accepted residual: retain explicit boundaries and activation triggers.

---

## 17. Restrictions

Read-only review.

No production code changes.\
No test changes.\
No migrations.\
No contract changes.\
No cleanup.\
No commits or pushes.\
No real provider calls.

Only the independent review report may be created in the repository. Scratch files remain outside the repository.

Do not quietly fix anything discovered. Report it to the owner.

---

## 18. Final instruction

Challenge the implementation, not the people who wrote it.

Do not assume that because the design was approved, its implementation is correct.

Do not assume that because tests are green, the authorization boundary is safe.

Do not expand the architecture to satisfy hypothetical future workloads.

The purpose of this review is to establish whether the actual shared framework boundary is trustworthy enough to support the remaining Phase 2 implementation.

**The framework owns reliability; miniapps own business logic.**

### **What happens after Gate 1**

If the independent verdict is CLEAR and there are no unresolved blocking findings, I will authorize the remaining Stage 3A work:

- Four Settle crash tests.
- Grant hash, `/healthz`, and revocation tooling.
- Full authorization/concurrency/crash matrix.
- Provider bypass tests.
- C6 recovery proof.
- PostgreSQL 19 local/Compose/CI digest parity.
- Final Stage 3A report and Gate 2 preparation.

If NOT CLEAR, we fix only the structural defect classes, then request a targeted re-review. No broad rewrite.