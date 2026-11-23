# **Consolidated P1 review — Prompt 15.2**

Prompt 15.2 confirmed as governing

Independent document review satisfied

Contract amendments required before P2

The two attached reviews are substantially compatible. I would combine their findings, retain the useful conditions, and remove duplicated or overly cautious requirements.

**My consolidated verdict: ACCEPT THE P1 DESIGN WITH AMENDMENTS. Do not begin P2 until the contract amendments below are incorporated.**

The independent-review requirement is now satisfied at the document-design level. The attached reviewers have examined the contract independently of its implementer. That does not verify the implementation or source citations independently; the P2 exit must still have a separate reviewer inspecting the actual code, tests, and evidence.

Your confirmation also resolves the supersession question: `prompt15.2.md` is the governing specification. Earlier Prompt 15 drafts no longer provide separate authorization.

## **1. Findings that should be retained**

The reviews agree on the strongest parts of the contract:

- RC-1a–d and RC-2 are a coherent root-cause diagnosis.
- Six transition-integrity rules are appropriate.
- Subsystem-specific transitions are preferable to a universal `transition()` primitive.
- R1 `attached_run_id` is a good solution to the run ownership/attachment identity problem.
- R3, making approval-gated invocations begin in `pending`, is an important structural correction.
- The submission choke point plus executor-side fail-closed validation is particularly strong.
- The sweeper is a reasonable framework-owned approval recovery mechanism.
- The F-01–F-14 residuals now have explicit dispositions.
- The P0 evidence and digest methodology are substantially improved.
- P2 must receive an independent implementation review, even though the P1 self-review limitation is now discharged.

I would preserve those decisions.&#x20;

prompt15.2_p1.md



## **2. Consolidated amendment list**

I have merged the overlapping findings from the attached reviews and my previous contract review into ten amendments.

| Priority | Amendment                                                       | Sources              |
| -------- | --------------------------------------------------------------- | -------------------- |
| HIGH     | Define attachment cancellation and lifecycle propagation        | CR-01, previous R-04 |
| HIGH     | Correct Park and sweeper transaction lock ordering              | CR-02, previous R-03 |
| HIGH     | Define the true effect-attempt authorization boundary           | CR-03, previous R-01 |
| HIGH     | Make generation fencing atomic with effect authorization        | CR-04, previous R-02 |
| HIGH     | Complete state-machine invariants and transition-table coverage | CR-05                |
| MEDIUM   | Correct the sweeper's guarantee and workflow termination claims | CR-06, previous R-05 |
| MEDIUM   | Preflight existing data before migration 000006                 | CR-07                |
| MEDIUM   | Correct kill-switch privilege and credential boundaries         | CR-08, previous R-06 |
| MEDIUM   | Strengthen version, dead-letter, and signal outcome semantics   | CR-09, CR-10         |
| LOW      | Improve source/test digest provenance                           | CR-11                |

There is one additional important issue from the earlier review: **grant revocation must be synchronized with effect authorization**, not merely checked before a transaction starts.

### **Three particularly important design corrections**

**Effect reservation versus actual attempt**

The contract must distinguish `PENDING` from `ATTEMPTING`.

I recommend defining `ATTEMPTING` as the logical point of no return for a provider attempt. The transition from `PENDING` to `ATTEMPTING` must revalidate invocation liveness, approval/authority, grant state, kill switches, and generation ownership under the appropriate database synchronization protocol.

A cancellation or supersession that wins first must prevent that transition. A transition to `ATTEMPTING` that wins first is an in-flight attempt, which cancellation cannot recall.

This also requires defining what happens to a pending effect when cancellation wins. The contract should not leave such records permanently pending or falsely count them as provider attempts. Reassess the existing `CANCELLED` state rather than automatically striking it.

**Generation fencing**

A check immediately before sending is still vulnerable to a check-then-act race.

Generation ownership must be validated at the same durable boundary that authorizes the effect attempt. This is the framework-level primitive future Class D miniapps should inherit rather than reimplement.

**Approval parking**

The contract currently says approval creation, invocation parking, and audit occur in one transaction, but it also describes recovering a dangling approval row produced by park failure.

Those descriptions need reconciliation. I recommend preserving atomic parking: lock/update the invocation first, then create the approval and audit in the same transaction. A failed park rolls all three back. Keep dangling-row adoption as defensive recovery for legacy or pre-existing records, and test it using an appropriate fixture.

## **3. Ready-to-send contract amendment directive**

The following is the consolidated instruction I recommend giving the implementation agent. It authorizes document amendments only, not P2 source changes.

# P1 Contract Amendment Directive — Consolidated Independent Review

## Governing authority

The owner has confirmed that `docs/prompt15.2.md` is the governing specification and supersedes all earlier Prompt 15 drafts.

The P1 lifecycle contract has received independent document-level review. The self-review limitation is discharged for P1 design review, but the P2 implementation must receive an independent exit review.

**P2 remains unauthorized until the contract amendments below are completed and accepted.**

Do not modify implementation source code while performing these amendments.

Do not modify `docs/plan4.2.md`, accepted G0 documents, prior adversarial-review documents, or other protected documents.

Preserve all P0/P1 evidence and existing modifications.

## 1. Attached invocation lifecycle and cancellation

**Finding: CR-01 and attached-run convergence concerns.**

Complete the M1, M5, and M6 contract tables.

Define the lifecycle of an invocation with `attached_run_id`:

- Status resolution must follow the canonical owning workflow.
- An attached invocation must be settled when the target workflow reaches a terminal state.
- Target success, failure, and cancellation must propagate truthfully.
- The sweeper must discover attached runs, not just records with their own `run_id`.
- `Status` must resolve `attached_run_id` to the owning run.
- Duplicate settlement audits must be prevented through the shared `settleTerminal` operation.
- Attachment must retain tenant and principal authorization boundaries.

Define all cancellation cases:

1. Cancellation of an attached invocation.
2. Cancellation of the owning invocation.
3. Cancellation of the target workflow.
4. Cancellation following supersession.
5. Cancellation redelivery.
6. Status reported by attached invocations after target termination.

Recommended default: cancelling an attached invocation cancels or detaches only that invocation's observer relationship. It must not cancel a shared workflow owned by another invocation. The historical attachment should remain attributable. Cancelling the owning invocation may request cancellation of the shared workflow; attached observers then follow its terminal outcome.

Record the final policy as an owner-visible decision. Do not assume the recommendation has been approved.

Add named tests for each applicable behavior.

## 2. Global database lock order and approval parking

**Finding: CR-02.**

The stated global lock order is:

1. Invocation.
2. Approval.
3. Effect.
4. Advisory entity lock.

The current Park row inserts the approval first and then updates the invocation. Correct this contradiction.

Preferred Park sequence, within one transaction:

1. Lock or conditionally update the invocation row.
2. Insert or adopt the approval row.
3. Record the approval-requested audit event.
4. Commit.

A failed operation rolls back the invocation transition, approval creation, and audit together.

A retry creates or adopts the same legitimate approval identity. A dangling approval row caused by this atomic Park operation should not be expected from a failed transaction; retain dangling-row adoption for legacy/pre-existing records and construct its test explicitly.

For sweeper S2, either:

- Acquire invocation then approval locks in one transaction, perform expiry and invocation settlement atomically, and release both before external calls; or
- Use separate sequential transactions, explicitly releasing the approval lock before acquiring the invocation lock.

Prefer the first where supported by the actual storage architecture.

Add a complete table of all multi-row transaction lock orders. Include the sequence-generation row and any authorization/grant rows added under amendment 3.

Verify with barrier-synchronized cancellation-versus-Park and expiry-versus-cancellation tests. Use statement timeouts to detect deadlocks.

## 3. Effect-attempt authorization and cancellation boundary

**Finding: CR-03 and the X-05 cancellation boundary.**

Clarify the difference between `PENDING` and `ATTEMPTING`.

Recommended semantic definition:

- `PENDING` is a provisional/reserved operation that has not yet been authorized to call the provider.
- `ATTEMPTING` is the durable logical point of no return for a provider attempt.
- `CONFIRMED`, `FAILED`, and `UNKNOWN` describe outcomes after that point.
- A pre-attempt reservation must never be reported as a provider call.

The `PENDING → ATTEMPTING` transition must, in one local PostgreSQL transaction:

1. Acquire locks in the documented global order.
2. Verify the invocation is still live.
3. Verify the relevant approval linkage and authority.
4. Revalidate grants and revocation state.
5. Revalidate token and kill-switch state.
6. Verify the operation remains eligible.
7. Verify any generation ownership/fencing requirement.
8. Verify the expected effect state and owner/lease.
9. Commit the `ATTEMPTING` transition before the external send.

Cancellation must serialize against that transition:

- Cancellation commits first: the transition fails; no provider attempt is authorized.
- `ATTEMPTING` commits first: the attempt is in flight and is reported as such.
- A failed or superseded worker cannot send after losing the attempt ownership predicate.

Do not hold a database lock across the external provider send.

Define and implement a terminal disposition for PENDING effects cancelled before attempt authorization. Reassess `CANCELLED` as a legitimate state transition instead of retaining the earlier decision to strike it automatically.

The cancellation response must distinguish provisional pending reservations from actual ATTEMPTING operations.

Add a deterministic test that pauses between PENDING and ATTEMPTING, commits cancellation, and verifies that the provider is never called.

## 4. Generation fencing at the effect boundary

**Finding: CR-04.**

A pre-send check of `status='active' AND generation=?` is insufficient because supersession can commit after the check.

Move generation validation into the same transaction that authorizes `PENDING → ATTEMPTING`.

Define one global acquisition order covering:

- Invocation.
- Approval where relevant.
- Sequence/generation.
- Grant and policy-control records.
- Effect.
- Advisory entity lock.

Supersession must serialize against effect-attempt authorization.

If supersession commits first, the old generation cannot authorize another attempt. If the attempt transition commits first, that attempt is already in flight and must be reported honestly.

Add a barrier test:

1. Old generation reaches the pre-attempt stage.
2. Pause it.
3. Commit supersession.
4. Resume the old generation.
5. Assert that it cannot transition to ATTEMPTING or call the provider.

The superseded workflow must not produce subsequent effects. Record that existing in-flight provider work cannot be recalled.

Generation fencing is the chosen framework mechanism; do not describe it as cooperative cancellation.

## 5. Grant revocation and policy consistency

**Finding: Previous independent review — effect-time grant race.**

The current contract leaves grant-map validation before the claim transaction while performing some other policy checks inside the transaction.

Resolve this against the Prompt 14 requirement:

"A race between policy read and effect claim cannot bypass revocation."

Where grants are stored in the same PostgreSQL database, include the grant read in the effect-attempt authorization transaction and use an appropriate shared row-lock protocol so that grant revocation and effect authorization have a defined order.

A plain unlocked read inside a transaction is not necessarily sufficient if revocation may commit after the read and before the effect authorization transaction commits.

Ensure the revocation path participates in the same synchronization protocol.

If a grant source is external or cannot join the transaction, specify the trusted version/token mechanism and explicitly identify the remaining guarantee. Do not silently retain the existing residual.

Add a test for grant revocation racing with effect authorization, in addition to the existing wait→revoke→refuse E2E.

## 6. State-machine completeness

**Finding: CR-05.**

For all six durable machines, explicitly document:

- Terminal states.
- Reachable-state invariants.
- Transitions prohibited from terminal states.
- The expected source-state predicate.
- Failure and recovery semantics.

At minimum include:

- An approval decision cannot be substituted after the first terminal decision.
- A settled invocation cannot return to a live state.
- A run ID cannot be overwritten after canonical ownership is established.
- A nonempty run ID maps to one owning invocation.
- An attached invocation resolves to one authoritative target.
- At most one valid effect owner can authorize a send for a business operation.
- A superseded generation cannot authorize new attempts.
- UNKNOWN outcomes cannot be blindly retried.

Add the missing Check-live event row. Explain which callers are eliminated or migrated to the atomic claim/authorization path.

Name every test supporting every contract row. References such as "kept tests" or "reconcile-retry test" must resolve to a real named test and file.

Correct stale section references and ensure the residual cross-references point to the actual F-residual section.

## 7. Sweeper recovery and bounded guarantees

**Finding: CR-06 and X-11.**

Preserve the selected framework-owned lifecycle sweeper and its strictly limited scope. It is not the Phase 4 production effects reconciler.

Clarify that the recovery guarantee is conditional:

- A configured bound applies while at least one healthy sweeper instance is running.
- Dependencies, including PostgreSQL and the relevant DBOS family, must be available.
- Startup recovery operates after service restart.
- Concurrent replicas must be safe through predicated, idempotent transitions.
- Batch limits, backlog handling, and failure visibility must be specified.
- A complete system outage does not have a finite recovery bound; recovery resumes when service becomes available.

Do not claim that the sweeper guarantees workflow termination merely because it settles the invocation and sends a signal.

Recommended expiry behavior:

1. Persist expiry and settle the invocation authoritatively.
2. Ensure the effect gate rejects subsequent consequential attempts.
3. Send the expiry notification.
4. Request cooperative cancellation of the workflow on a best-effort basis.
5. Expose cancellation failure and retry it on subsequent sweep/startup.
6. Never claim that the workflow is confirmed stopped merely because `CancelWorkflow` was requested.

The safety guarantee is the durable invocation/effect gate. Resource cleanup is best-effort cooperative cancellation.

Add operator documentation covering the sweeper's process group, schedule, operations, failure behavior, backlog monitoring, and restart recovery.

Use the documented interval, but do not promise unconditional wall-clock convergence during total service unavailability.

## 8. Migration 000006 data preflight

**Finding: CR-07.**

Before applying the unique run-ID index or changing identity encoding:

- Scan for duplicate nonempty `run_id` values.
- Scan relevant existing identity fields for `%` and `:` characters.
- Check for collisions under the proposed canonical escaping.
- Record the scan results for the disposable local database and authorized test branch.
- Add upgrade fixtures that deliberately contain duplicate run IDs and delimiter-bearing IDs.

If duplicates or collisions exist:

- Detect and report them before applying the constraint.
- Do not silently choose an owner or rewrite ambiguous workflow identities.
- Fail migration with an actionable diagnostic or execute an explicitly authorized deterministic remediation.

Test fresh install and upgrade paths.

Do not mutate shared or production Neon resources.

## 9. Kill-switch control-plane security

**Finding: CR-08 and X-15.**

Remove the claim that the trigger is "unbypassable."

Use accurate wording:

- The restricted application role cannot change kill-switch state directly.
- The authorized setter and audit trigger cover ordinary permitted writers.
- A table owner or superuser with sufficient privilege can still alter or disable the mechanism.

The serving process must not receive or retain the control-plane DSN if miniapp code runs in that same process.

Prefer:

- Restricted role for normal serving.
- Separate authenticated operator command for control-plane changes.
- Operator credentials loaded only by that command, not by the serving process.
- Explicit deployment instructions for the production role configuration.

The operator function must protect actor, reason, and scope attribution.

Test direct writes using a genuine restricted-role database connection, not text inspection alone. Also test the privileged setter, audit trigger, and actual runtime connection role.

Record the remaining owner/superuser and trusted in-process-code limitations as explicit residuals.

## 10. Version affinity and signal semantics

**Findings: CR-09 and CR-10.**

### X-09

Keep the exact five-step behavioral test:

1. Create pending v1 work under v1.
2. Stop v1.
3. Run only v2.
4. Prove v2 does not execute or advance the v1 workflow.
5. Restart v1 and prove v1 recovers and executes it.

Do not weaken the test into a source citation or a non-discriminating alternative.

### X-10

Give `MAX_RECOVERY_ATTEMPTS_EXCEEDED` its own visible dead-letter/operator-actionable class.

Confirm from pinned SDK v1.9.0 whether and how it can be resumed.

Define what the framework invocation reports when the DBOS run is dead-lettered, including the consequences for operator recovery. Do not silently settle an invocation as permanently failed if a supported resume mechanism remains available.

### X-06

Distinguish:

- Unknown workflow: error.
- Unavailable workflow family: explicit unavailable error.
- Live workflow: notification accepted by DBOS.
- Already terminal workflow: distinct no-op/already-terminal result.
- Workflow processed signal: only claim this if actual processing evidence exists.

`signal/accepted` must describe a live run whose notification was accepted. Do not return ordinary success for a terminal-run no-op.

Preserve the documented RetrieveWorkflow-to-Send residual TOCTOU.

## 11. Evidence provenance improvements

**Finding: CR-11 and X-18.**

Keep the current reproducible full-tree digest.

Additionally introduce a source-scope digest for the executable content under test:

- Go implementation and tests.
- SQL and migrations.
- CI workflow configuration.
- Scripts.
- Relevant fixtures and testdata.

For red/green testing, bind:

1. The implementation source digest.
2. The test-harness digest.
3. The full-tree digest.
4. The exact command and environment.

Preserve the same discriminating test source across the red and green runs so that the comparison identifies the implementation change rather than a different test.

Record the manifest delta between the P0 and mid-P1 snapshots, explaining the additional files.

Bind the SDK digest to pinned v1.9.0 source provenance, not to a machine-local absolute path.

Do not retroactively reconstruct historical evidence whose tested source state cannot be established.

## 12. P1 exit and P2 gate

Update the P1 contract and its self-check table to reflect every amendment.

Record:

- What changed.
- Why it changed.
- The affected state-machine rows.
- The corresponding regression tests.
- Any owner decision still outstanding.
- Any newly discovered conflict.

The independent document-review condition is satisfied. The independent implementation review remains mandatory at P2 exit.

Do not begin source implementation as part of this amendment task.

P2 remains gated until:

1. All contract amendments are incorporated.
2. Attachment cancellation semantics are explicitly owner-approved.
3. Effect-attempt and generation-fencing semantics are explicit.
4. The lock-order tables are internally consistent.
5. Sweeper guarantees are truthful.
6. The data-preflight and control-plane security requirements are included.

Prompt 15.2 remains the governing specification. The `lead-followup` hard stop, no commits/pushes, protected-document restrictions, mock-only channels, disposable databases, and forward-only migration requirements remain binding.

**The desired outcome is still a reusable framework primitive, not a one-off probe fix. The effect, approval, attachment, cancellation, and recovery guarantees must be available to future miniapps without requiring their authors to independently recreate the same distributed-systems safeguards.**

## **4. Owner decisions still needed**

The governing-spec question is resolved. The remaining decision with the clearest product-level consequences is attachment cancellation.

**When a losing invocation is attached to another invocation's workflow, what should cancelling the loser do?**

**Cancel only the attached invocation (recommended).**

Preserve its attachment history, settle its own status as cancelled, and leave the shared target workflow running.

**Cancel the shared target workflow too.**

This affects the winning invocation and every other observer attached to that workflow.

Confirm decision

My recommendation is the first option because an attached invocation does not own the shared workflow. It avoids giving one observer authority to cancel another invocation's work.

**Overall:** P1 has a strong and largely complete design. The amendments are about closing contradictions and ensuring the new fixes do not introduce new failure windows. Once those are incorporated, the next step is P2—not another full architecture redesign.

Owner decision accepted

The attachment cancellation policy is now settled:

**Detach-only cancellation.** An attached invocation does not own the shared workflow and therefore cannot cancel it.

Use the following normative rule in `invocation-lifecycle-contract.md` and carry it into P2 implementation.

### Owner Decision — Cancellation of an Attached Invocation

**Decision: DETACH-ONLY.**

When an invocation is attached to another invocation's canonical workflow through `attached_run_id`, cancelling the attached invocation:

1. Transitions only the attached invocation to its terminal `cancelled` state.
2. Does not call `CancelWorkflow` on the shared target workflow.
3. Does not modify the owning invocation's lifecycle.
4. Does not affect other invocations attached to the same target.
5. Preserves the original `attached_run_id` as immutable historical attachment information. Do not clear it during cancellation.
6. Records an attributable cancellation audit event identifying the attached invocation and its target workflow.
7. Is idempotent under repeated cancellation requests.
8. Requires normal tenant ownership and authority checks.

After cancellation, the attached invocation no longer follows the target's live status for its own lifecycle. Its status remains `cancelled`. The preserved attachment reference remains available for audit and historical inspection.

Conversely, when the owning invocation's target workflow terminates or is cancelled, attached invocations that have not themselves been cancelled must resolve and reflect the target's authoritative outcome according to the attachment lifecycle contract.

Cancelling an attached invocation must never implicitly cancel the shared target workflow.

**Required regression tests:**

- Cancelling an attached invocation settles only the attached invocation.
- The target workflow remains active and can complete normally.
- Other attached invocations remain unaffected.
- `attached_run_id` remains preserved after cancellation.
- Repeated cancellation is idempotent.
- Cross-tenant cancellation is rejected.
- The owning invocation's cancellation behavior remains unchanged.

This resolves the owner decision for X-08/CR-01. The implementation agent should incorporate it into the M5 and M6 transition tables and their tests.

**P2 is still gated** on the other contract amendments: effect-attempt authorization, generation fencing, lock ordering, approval parking atomicity, sweeper guarantees, migration preflight, and control-plane security. No implementation authorization is implied by this decision alone.