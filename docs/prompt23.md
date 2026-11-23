The next step is **Prompt 23 — Stage 3A: Implement the Authorization Core**.

This prompt directs the coding agent to implement the actual `authorizeAttempt` boundary, preserve the two owner-approved design decisions, and prepare the implementation for the first independent adversarial review. It deliberately stops at that review gate rather than allowing the work to expand unchecked.

# Prompt 23 — WS-B Phase 2, Stage 3A: Authorization Core Implementation

**Repository:** `github.com/PithomLabs/miniapp`\
**Workstream:** WS-B — Effect Authorization\
**Governing implementation order:** Prompt 22\
**Approved design:** `docs/phase2/authorize-attempt-design.md`\
**Current status:** Stage 0, Stage 1, and Stage 2 complete. Stage 3 explicitly released by the owner.\
**Next review gate:** Independent Adversarial Review Gate 1 — Authorization Core.

---

## 1. Mission

Act as the senior Go/PostgreSQL engineer implementing Miniapp's canonical effect-time authorization boundary.

Implement the approved `authorizeAttempt` design so that an external provider operation cannot begin unless the framework has atomically revalidated authoritative persisted state and committed the corresponding effect's transition into `ATTEMPTING`.

The governing principle is:

> The framework owns reliability; miniapps own business logic.

This implementation must enforce authorization and concurrency guarantees at the shared framework boundary. Individual miniapps, workflows, and adapters must not have to reproduce these checks.

This is implementation work, not another architecture-discovery exercise. The owner has approved Stage 3.

However, do not interpret that approval as permission to redesign the framework, weaken the contract, or implement unrelated future work.

## 2. Governing sources

Read the current repository versions before making changes:

1. `docs/plan4.2.md`
2. `docs/prompt15.2.md`
3. `docs/phase1/invocation-lifecycle-contract.md`
4. `docs/prompt17.md`
5. `docs/phase1/prompt20-phase1-closeout.md`
6. `docs/adv_review_prompt21.md`
7. `docs/phase2/authorize-attempt-design.md`
8. Relevant effects, approval, policy, cancellation, and ledger implementation.

The authorization design note is the approved Stage 2 engineering deliverable. Use its actual transaction design, API contracts, source mappings, and test decisions. Do not reconstruct or simplify the design from earlier summaries.

### Owner decisions already resolved

The following decisions are binding and do not need another owner clarification.

**Decision A — M5 sequence deferral**

Entity-sequence fencing remains probe-scoped. Do not introduce a generic production sequence subsystem or migration merely to anticipate hypothetical workloads.

The distinction between per-operation ownership and entity-sequence ownership must remain explicit:

- BusinessOpID and attempt lineage protect the same logical business operation.
- Lease and ownership predicates protect attempt ownership.
- Invocation binding prevents an operation from being adopted or authorized under an unrelated invocation.
- M5 sequence generation protects entity-level supersession.

The first three do not substitute for the fourth.

For existing non-sequence effects, use the approved durable operation-ownership model.

For probe-scoped sequence effects, preserve the existing authoritative sequence-generation fence and validate it at the effect-authorization boundary.

For any future production workload that requires entity-level sequence supersession, the activation trigger is explicit: introduce authoritative durable sequence ownership and validate the generation before authorization. Until that capability exists, an effect requiring such a fence must fail closed. Never silently skip a required sequence check.

Do not create a general sequence table, migration, or framework abstraction without a concrete requirement and separate owner approval.

**Decision B — Withdraw semantics**

Preserve the distinction between a transient authorization refusal and terminal effect withdrawal.

- Temporary policy denial, kill-switch engagement, infrastructure failure, and retryable contention do not automatically withdraw a pending operation.
- Durable invocation cancellation that wins before authorization withdraws the pending effect through the predicated `PENDING → CANCELLED` transition.
- Other terminal invalidations may withdraw an effect only where the governing contract explicitly defines that behavior.
- `CANCELLED` is terminal, never retried, and never counted as a provider attempt.
- Once `ATTEMPTING` commits, the operation is in flight. It cannot be recalled merely by changing invocation state.

Map every refusal reason to its precise outcome. Do not infer terminality from an error name or convenience.

If the approved design note does not yet record these owner decisions, update it before implementing the affected behavior. Do not silently diverge from the owner-approved design.

## 3. Mandatory invariants

The implementation must satisfy Prompt 21 constraints C1–C7 and the existing lifecycle contract.

### C1 — No advisory lock in `authorizeAttempt`

The authorization transaction acquires no PostgreSQL advisory lock.

Do not introduce advisory locks as a concurrency shortcut.

### C2 — Preserve global lock ordering

Follow the exact reviewed ordering:

`RLock → 1 invocation → 2 approval → 3 sequence/generation → 4 policy controls/grants → 5 effect`

Use the specific lock modes defined in the contract and design note.

Do not acquire locks out of order or add an unreviewed cross-transaction dependency.

### C3 — Separate authorization transaction

Authorization runs in its own transaction and pool checkout.

Do not pass the Class A attempt's `pgx.Tx` into `authorizeAttempt`. Do not merge the Class A attempt transaction and the authorization transaction.

### C4 — No new advisory-lock paths

Preserve the existing advisory-lock mechanism and its guard.

Extend mechanical enforcement if new acquisition paths or raw-SQL advisory mechanisms could bypass the current guard.

### C5 — Pending semantics

P21-002 is closed. The adopted workflow transition and updated lifecycle contract are authoritative.

An ordinary `pending` invocation is not automatically eligible for effects. Respect the chosen `pending → running` adoption semantics.

Class A pending records remain unchanged.

### C6 — Class A Perform recovery

Preserve the stable BusinessOpID and the approved bounded handling of `ErrAttemptInProgress`.

A Class A handler must not hold its transaction indefinitely while waiting for an effect-ledger lease.

### C7 — Effect-time protection

A submit-time or workflow-time authorization check is not sufficient.

Revalidate authoritative state at the effect boundary immediately before the transition to `ATTEMPTING`.

The implementation must prevent the O-01 stale-workflow path from authorizing a new effect after settlement or cancellation.

---

## 4. First implementation objective: canonical authorization core

Implement the canonical `authorizeAttempt` mechanism described in the design note.

Its responsibility is to evaluate the complete authorization read set and conditionally transition the effect to `ATTEMPTING` in one database transaction.

At minimum, it must revalidate:

### 4.1 Invocation

- Tenant identity.
- Invocation identity.
- Authoritative persisted status.
- Cancellation and terminal state.
- Eligibility under the completed P21-002 semantics.
- Correct association between invocation and effect.

Do not treat an earlier liveness check as authoritative authorization.

### 4.2 Approval

- Persisted approval linkage.
- Granted decision.
- Matching tenant.
- Matching invocation.
- Matching tool and operation.
- Matching authority principal.
- Matching intent.
- Validity and expiry.
- Any required approval-generation or binding information.

Missing, denied, expired, cancelled, or mismatched approval must fail closed.

Pending approval may enter the durable approval-wait path but never authorizes consequential execution.

### 4.3 Grants and policy

Implement the approved pinned configuration snapshot.

- Add the `RWMutex` using the approved pointer-receiver migration.
- Provide a setter API for controlled policy mutation.
- Protect all grant reads and credential resolution.
- Hold the required read lock across the authorization read set.
- Prevent unguarded map mutation.
- Preserve the absence of hot reload.
- Preserve the ten-minute maximum staleness model.
- Respect cross-replica restart/coherence requirements.

Implement the credential-free configuration hash.

Expose the hash through `/healthz`.

Implement `miniapp auth verify-revocation <expected-hash>` using the explicit replica inventory described in the design note.

The command must report which replicas were verified, which were not reachable, and whether the required hash matches. A partial inventory must never be represented as complete verification.

Do not expose credentials, secret material, or sensitive policy contents through hashes, logs, or health endpoints.

Do not claim immediate distributed revocation.

### 4.4 Policy controls

Read authoritative kill-switch and policy-control state using the approved lock order and lock modes.

An engaged applicable switch must prevent authorization.

The effect must not enter `ATTEMPTING` when policy refuses it.

### 4.5 Effect identity and durable ownership

Use the approved ledger identity model:

- Stable BusinessOpID.
- Durable operation primary key.
- Attempt lineage.
- Lease.
- Invocation binding.
- Predicated ownership transitions.

Inspect `adopt()` carefully.

The confused-deputy gap identified in the Stage 2 design must be closed through authoritative binding verification. An existing operation must not be adopted merely because a caller knows its operation ID.

Verify the tenant, invocation, operation identity, authority, and other design-defined bindings before adoption.

Do not accept arbitrary caller-supplied identity as proof of ownership.

The same logical business operation must retain its BusinessOpID across retries and recovery.

### 4.6 M5 sequence ownership

Preserve the owner-approved M5 scope.

- Non-sequence effects do not require a nonexistent generic sequence table.
- Probe-scoped sequence effects retain their authoritative generation fence.
- A required but unavailable or unverifiable sequence fence fails closed.
- No generic M5 production migration is introduced.

Record the future activation trigger in the implementation documentation.

### 4.7 Atomic effect transition

Within the same authorization transaction:

1. Revalidate the invocation.
2. Revalidate approval and authority.
3. Validate the pinned grant snapshot.
4. Validate policy controls.
5. Validate sequence/generation ownership when applicable.
6. Validate the effect token, state, lease, owner, and attempt lineage.
7. Perform the predicated transition into `ATTEMPTING`.
8. Verify the affected-row count.
9. Commit.

Zero rows, stale ownership, mismatched identity, or invalid authority must never be treated as successful authorization.

The provider call must occur only after commit.

---

## 5. Transient refusal versus terminal withdrawal

Implement the distinction as a first-class behavior.

Before coding, create or update an explicit reason-to-disposition table in the design or contract.

The table must distinguish:

| Condition                              | Required disposition                                  |
| -------------------------------------- | ----------------------------------------------------- |
| Temporary policy denial                | Refuse; pending effect is not automatically withdrawn |
| Engaged kill switch                    | Refuse; no provider call                              |
| Grant revoked                          | Refuse under the approved snapshot/revocation model   |
| Infrastructure failure                 | Retryable refusal; preserve pending state             |
| Concurrent owner/lease conflict        | Defined contention result; no duplicate attempt       |
| Stale worker                           | Reject stale ownership                                |
| Durable cancellation committed first   | Withdraw pending effect to terminal `CANCELLED`       |
| Other terminal invocation invalidation | Follow only an explicitly defined contract transition |
| Authorization committed first          | `ATTEMPTING`; in-flight, not recallable               |
| Ambiguous provider outcome             | `UNKNOWN`; never blindly retry                        |

Ensure audits preserve this distinction:

- `effect-refused` identifies authorization refusal.
- `effect-withdrawn` identifies terminal withdrawal before an attempt.
- Provider attempt records describe actual provider initiation, not authorization refusals.

Do not manufacture provider-attempt records for work that never crossed the authorization boundary.

---

## 6. Atomic settlement correction

Implement the approved Settle correction from the Stage 2 design note.

The Stage 2 design identifies the existing `RECONCILING` intermediate state and the need for atomic settlement.

The required correction must ensure that:

- The operator's authoritative settlement transition is atomic.
- Audit creation is consistent with the committed settlement.
- In-memory map cleanup cannot contradict durable state.
- A crash cannot leave an unhandled intermediate state that violates the approved contract.
- Existing operation ownership and fencing remain respected.

Use the exact four named crash tests in the design note. Preserve their intended crash boundaries.

Do not invent a new settlement state machine or automatic reconciliation sweeper.

Do not implement X-12 or X-10 as part of this correction.

---

## 7. Canonical provider boundary

Inventory every production provider-initiation path.

Include:

- `Perform`.
- `Channel`.
- Claim/adopt paths.
- Lease takeover.
- Retry-after-reconciliation.
- Recovery and resume paths.
- Any other adapter or executor capable of initiating a provider operation.

Every consequential provider call must pass through the same canonical authorization mechanism.

Do not add independent policy checks to individual adapters as substitutes for the shared boundary.

Demonstrate that authorization failure on each production path means:

- No `ATTEMPTING` transition.
- No provider call.
- No false success.
- Correct audit attribution.
- Correct retry or terminal disposition.

The provider call must happen after transaction commit.

Keep providers mocked throughout this workstream. No real customer credentials or real provider writes.

---

## 8. Test requirements

Tests must prove database behavior, not merely return values.

Use real PostgreSQL 19 sessions for transaction ordering, row locking, concurrency, and recovery.

### 8.1 Authorization matrix

Verify:

- Valid granted approval proceeds.
- Missing approval refuses.
- Pending approval refuses consequential execution.
- Denied and expired approvals refuse.
- Wrong tenant or invocation refuses.
- Wrong operation, tool, authority principal, or intent refuses.
- Revoked grant refuses.
- Engaged kill switch refuses.
- Invalid effect token refuses.
- Terminal invocation refuses.
- Invalid lease refuses.
- Stale generation refuses.
- Wrong owner or operation identity refuses.
- Duplicate concurrent authorization has one winner.
- Retry-after-reconciliation performs a fresh authorization.

### 8.2 Concurrency

Use independent database sessions and deterministically force:

1. Cancellation commits before authorization.
2. Authorization commits before cancellation.
3. Two workers authorize the same effect.
4. A stale worker races with a new owner.
5. Lease takeover races with authorization.
6. Revocation races with authorization under the defined snapshot ordering.
7. Generation supersession races with a sequence-bound authorization.
8. Settlement races with competing effect ownership.

Assert persisted state, transaction outcomes, attempt lineage, affected-row counts, audit records, and actual mock-provider calls.

Do not use sleep-based timing assumptions, goroutine counts, or in-memory locks as evidence of PostgreSQL guarantees.

### 8.3 Crash and recovery

Test the authorization and effect lifecycle at meaningful boundaries:

- Before authorization commit.
- Immediately after authorization commit, before provider dispatch.
- During provider dispatch.
- After provider response, before outcome persistence.
- During settlement.
- During ownership takeover/recovery.

Verify the durable state and the correct recovery action.

Distinguish:

- Safe retry.
- Same-operation adoption.
- Lease contention.
- Reconciliation.
- `UNKNOWN`.
- Terminal withdrawal.

Never silently retry an ambiguous external operation.

### 8.4 Grant snapshot

Verify:

- Concurrent policy reads.
- Setter/revocation ordering.
- Snapshot pinning during authorization.
- Credential-free configuration hash stability.
- Hash changes for relevant grant changes.
- `/healthz` hash correctness.
- Explicit replica-inventory verification.
- Missing or unreachable replicas are reported as incomplete verification.
- No direct map writes bypass the setter.

### 8.5 C6 Class A integration

Use a Class A test fixture invoking `Perform`.

Verify:

- Stable BusinessOpID.
- Correct operation adoption.
- Bounded `ErrAttemptInProgress` recovery.
- No indefinite transaction hold.
- No duplicate logical operation.
- Correct behavior after rollback and subsequent recovery.

Do not claim C6 is proven because no current production handler uses `Perform`.

### 8.6 Red/green discipline

For material defects:

1. Freeze the discriminating test bytes.
2. Preserve the exact pre-fix source snapshot.
3. Reproduce the defect against the pre-fix implementation where possible.
4. Apply the durable correction.
5. Run identical test bytes against the corrected implementation.
6. Record hashes, commands, exit codes, and exact outputs.

If a true red is technically impossible, state why and classify the evidence honestly.

Tests that pass on both sides are characterization tests, not discriminating red/green evidence.

---

## 9. PostgreSQL 19 requirements

PostgreSQL 19 remains mandatory throughout development, local testing, integration, CI, and crash verification.

Use the exact pinned image identity established by the project. Do not use floating `postgres:19.0` tags as the only image identity.

Verify the full image digest using Docker inspection. Ensure Compose and CI refer to the same immutable image.

Record:

- `SELECT version()`.
- Full image digest.
- Go version and architecture.
- Database names and disposable test configuration.
- Exact commands and working directories.
- Test duration, count, skips, and failures.

Run the serial repository suite where required:

```
gofmt -l .
go vet ./...
go test -p 1 ./... -count=1
go test -race -p 1 ./runtime/ ./approvals/ ./effects/
bash scripts/checks.sh
```

Set `TEST_DBOS_DATABASE_URL` for DBOS verification.

If PostgreSQL 19 parity is unavailable, mark affected tests BLOCKED. Never substitute another major version and report PASS.

---

## 10. Implementation sequence

### Step A — Verify Stage 2 baseline

Confirm:

- Current branch and HEAD.
- Current worktree digest.
- Design note and contract versions.
- Stage 1 and Stage 2 evidence entries.
- PostgreSQL 19 image identity.
- Existing test infrastructure.

Do not overwrite or silently modify previous evidence.

### Step B — Implement the authorization core

Implement the canonical transaction and its store/API contracts.

Keep transaction ownership explicit.

Prove the lock order and C1–C7 compatibility in source and through database tests.

### Step C — Implement the grant-snapshot mechanism

Implement the approved mutex, setter API, hash, health exposure, and revocation verification command.

### Step D — Implement ownership, fencing, and adoption protections

Implement the approved durable per-operation ownership model and close the `adopt()` confused-deputy gap.

Preserve the M5 deferral.

### Step E — Integrate and enforce provider initiation

Route every production initiation path through the canonical authorization boundary.

Prove there is no bypass.

### Step F — Implement approved atomic settlement correction

Apply the approved atomic-settlement design and its four named crash tests.

### Step G — Run focused and full verification

Run the complete relevant test matrix and full PostgreSQL 19 battery.

Confirm no hidden skips, stray processes, or unexplained worktree changes.

---

## 11. First adversarial review gate — mandatory stop

After the authorization core and its initial production integration are implemented, STOP.

Do not proceed into unrelated cleanup or expand the workstream before the first independent adversarial review.

Create a review-ready checkpoint with:

- Exact source and tree digests.
- The authorization design note.
- The complete `authorizeAttempt` implementation.
- Store transaction implementation.
- Lock-order and transaction graph.
- Grant snapshot implementation.
- Ownership/adoption implementation.
- Production provider-path inventory.
- Focused PostgreSQL 19 test results.
- Red/green artifacts.
- Known residuals and unimplemented items.

The independent reviewer must challenge at least:

1. Whether every authorization check uses authoritative persisted state.
2. Whether all checks and `PENDING → ATTEMPTING` are atomic.
3. Whether C1–C7 are actually preserved.
4. Whether the grant snapshot and revocation behavior match the owner-approved model.
5. Whether the adoption path can become a confused deputy.
6. Whether ownership, attempt lineage, and lease fencing survive concurrency and crash recovery.
7. Whether M5 deferral is safe within the stated scope.
8. Whether cancellation and withdrawal semantics distinguish transient refusal from terminal withdrawal.
9. Whether a provider path can bypass authorization.
10. Whether the tests discriminate the claimed defects.

The reviewer must be independent of the implementer.

### Review gate rules

- CRITICAL or HIGH findings block further progression.
- MEDIUM findings require explicit owner disposition.
- LOW findings are tracked without automatically reopening completed foundations.
- Accepted residuals must be named and bounded.
- A green test suite does not override a structural defect.
- No automatic acceptance based on the implementer's own assessment.

Produce the review input package and request independent review. Do not start the next integration expansion until the owner has received and dispositioned Gate 1.

---

## 12. Scope restrictions

Do not implement:

- A generic production M5 sequence subsystem.
- River/Class B.
- Class C workload decisions.
- X-12 audit-backfill sweeper.
- X-10 quarantine operator semantics.
- Phase 4 effects reconciler.
- Sprites, G6, or Fly production rollout.
- Real provider integrations or customer credentials.
- Unrelated architecture changes.
- UI or generator work.

No commits, pushes, releases, or deployments.

Keep migrations forward-only. Do not introduce a migration unless the approved design proves it necessary. The expected design outcome is no migration.

---

## 13. Required deliverables

Produce:

1. The canonical `authorizeAttempt` implementation.
2. Grant snapshot, hash, health, and revocation safeguards.
3. Durable operation ownership and adoption protections.
4. Canonical provider-path enforcement.
5. The approved atomic settlement correction.
6. PostgreSQL 19 integration and concurrency tests.
7. Crash and recovery evidence.
8. Updated lifecycle contract and implementation documentation.
9. Updated evidence register.
10. Review checkpoint for Independent Adversarial Review Gate 1.

Create the implementation report:

`docs/phase2/prompt23-stage3a-implementation-report.md`

The report must contain:

- Exact baseline and final tree identity.
- PostgreSQL 19 version and full image digest.
- C1–C7 compliance matrix.
- Authorization transaction and lock-order explanation.
- Grant-snapshot and revocation results.
- Ownership and adoption proof.
- M5 scope and future activation trigger.
- Transient-refusal versus terminal-withdrawal mapping.
- Atomic settlement evidence.
- Provider-path inventory.
- Test and crash/recovery matrix.
- Exact commands, outputs, and hashes.
- Unresolved findings and residuals.
- Worktree state and confirmation of no commit/push/deployment.

Classify every claim as execution-proven, source-proven, composition-proven, characterization-only, accepted residual, or blocked/unverified.

---

## 14. Final instruction

Implement the smallest durable mechanism that enforces the approved design.

Do not weaken the architecture to make a test pass. Do not use process-local synchronization as a substitute for database ownership. Do not confuse a refusal with a withdrawal. Do not confuse authorization with provider delivery. Do not claim exactly-once external execution.

The first deliverable is not merely code that compiles. It is a reviewable, transactionally correct authorization core with evidence strong enough to survive independent adversarial scrutiny.

**Proceed with Stage 3A. Stop at Adversarial Review Gate 1.**

After this prompt, the sequence is: **Stage 3A implementation → independent Gate 1 review → remaining Stage 3 integration → Gate 2 review → Stage 4 full verification → final Gate 3 review and Phase 2 acceptance.**