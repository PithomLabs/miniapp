The net result is **P1 accepted for P2 commencement, subject to explicit P2 exit conditions**. The grant-revocation limitation is accepted as a deployment residual, not a framework blocker, provided Conditions A–E are carried into P2 exactly as owner-approved.

This reconciles the latest acceptance review, which authorizes P2 and carries Conditions 1–12 to its exit, with your grant-revocation decision.&#x20;

Pasted text(20261123-005435).txt



I am retaining two additional technical findings from my previous review: the stranded `RECONCILING` possibility and incomplete lock-order coverage. They should be added to the P2 exit checklist rather than used to reverse the P1 acceptance.

# P1 Amendment Review — Consolidated Acceptance and P2 Exit Directive

**Date:** 2026-11-23\
**Status:** P1 ACCEPTED — P2 AUTHORIZED TO BEGIN\
**Governing specification:** `docs/prompt15.2.md`\
**Implementation contract:** `docs/phase1/invocation-lifecycle-contract.md`

## 1. Consolidated decision

The amended P1 contract is accepted as the governing implementation specification for P2.

The acceptance is conditional on satisfying the P2 exit requirements in this directive. Those conditions are not preconditions to starting unrelated P2 work. Where a condition concerns an unresolved state transition or authorization boundary, however, the relevant implementation must not proceed on an invented or ambiguous interpretation.

The following decisions are settled:

- `docs/prompt15.2.md` governs; prior Prompt 15 drafts are superseded.
- DETACH-ONLY cancellation of attached invocations is owner-approved. This approval was given directly by the owner, not inferred from a review directive.
- The grant-revocation limitation is accepted as a deployment property, subject to Conditions A–E in Section 4.
- OpenCode remains `NOT RUN / BLOCKED`, non-blocking for P2.
- No shared/production Neon mutations, real provider writes, commits, pushes, or protected architecture-document edits are authorized by this acceptance.

## 2. Accepted architectural amendments

The following amendments are accepted and should not be reopened without new evidence:

1. **Effect authorization:** `PENDING` is a provisional reservation; `ATTEMPTING` is the durable point of no return. All authorization checks occur in the transition transaction, and provider sends occur after commit.
2. **Cancellation:** cancellation intent prevents subsequent authorization. The response distinguishes withdrawn work from in-flight attempts.
3. **Generation fencing:** sequence generation ownership is checked within the effect-authorization transaction.
4. **Approval parking:** Park locks invocation before approval insertion and commits the state change, approval record, and audit atomically.
5. **Attachment lifecycle:** attached invocations participate in Status and settlement discovery. Cancelling an attached invocation cancels only that invocation, preserves `attached_run_id`, and never cancels the shared target or other observers.
6. **Lifecycle recovery:** the framework-owned sweeper handles signal redelivery, approval expiry, and terminal settlement, subject to the documented availability and backlog conditions.
7. **Control-plane separation:** the serving process does not retain the control DSN; switch operations use a separate operator command and canonical authorization.
8. **Affinity and signals:** X-09 retains the exact five-step behavioral test; signal outcomes distinguish unknown, unavailable, live accepted, already-terminal, and evidenced processing.
9. **Evidence discipline:** red/green tests, source-scope digests, full-tree digests, and independent implementation review remain mandatory.

## 3. P2 exit requirements carried forward

These requirements consolidate the accepted review's Conditions 1–12 with additional lifecycle findings. They are P2 completion criteria, not reasons to reopen P1 acceptance.

### 3.1 Evidence, review, and provenance

- HIGH findings and X-07 receive empirical red/green tests against the exact pre-remediation and remediated source snapshots. Both runs use the same discriminating test source and documented harness.
- The P2 exit receives an independent implementation review by someone other than the P2 implementer.
- Every result is bound to the implementation source digest, test-harness digest, full-tree digest, command, and environment.
- `docs/prompt15.2_p1_review.md` is recorded in the evidence register with author, date, and scope. Its independent-review or self-review status must be stated accurately.
- The final manifest and digest claims must reconcile with actual entries. In particular, resolve the contradictory `faq.md` descriptions in §13 of the contract.
- Historical undigested claims remain marked irreproducible; they must not be retrofitted with invented provenance.

### 3.2 Contract corrections are never silent

Add this rule to §13:

> If P2 discovers a contract inaccuracy, it records the correction in this document with both the original claim and the corrected claim, and resumes with the corrected discipline. Contract inaccuracies are not silently reconciled.

Do not edit protected architecture or review documents to make evidence appear consistent. Surface conflicts and preserve their provenance.

### 3.3 Effect reconciliation recovery

The current M3 contract describes operator settlement as `UNKNOWN → RECONCILING → terminal` across two transactions, but does not specify recovery if the process stops after entering `RECONCILING`.

P2 must close this lifecycle gap by either:

- Making the settlement transition atomic; or
- Defining a recoverable, predicated, and fenced `RECONCILING` state with explicit retry/adoption semantics.

Add `RECONCILING` to retry-as-re-entry handling. Add fault injection that terminates settlement between the two transactions and proves eventual convergence without allowing conflicting operator decisions.

A `RECONCILING` record must not become an indefinitely stranded state.

### 3.4 Quarantine and dead-letter lifecycle

The SDK evidence establishes that dead-lettered workflows can be resumed. P2 must translate that capability into an explicit framework lifecycle rather than only a status mapping.

Specify:

- The exact DBOS v1.9.0 resume API and eligible workflow states.
- The framework-authorized `quarantined → running` resume transition.
- The operator-authorized `quarantined → failed` transition.
- Predicated concurrency, audit, and recovery semantics for both.
- Rejection of attempts to resume an invocation whose framework lifecycle is already `cancelled`.
- A clear operator-visible Status representation containing `status: quarantined`, a reason, and the required operator action (resume or fail).

Do not let an SDK resume operation silently revive a framework-terminal invocation.

Required tests cover successful resume, operator failure, status visibility, concurrent decisions, and rejection of resume after cancellation.

### 3.5 Lock-order completeness

Complete §4.2's lock-order table for every relevant path, explicitly covering:

- Sequence claim.
- True attachment.
- Existing effect takeover/adoption.
- Retry authorization.
- Any path that transitions from effect ownership into invocation or sequence authorization.

For each path, state whether locks are acquired together, which transaction owns them, and precisely where locks are released.

In particular, effect takeover must not retain an effect-row lock while beginning the higher-order invocation/sequence authorization path. If the work uses separate transactions, document that separation.

Do not claim the global order is cycle-free until these paths are accounted for. Keep the barrier/concurrency tests and statement timeouts.

### 3.6 Migration preflight

The local and Neon CR-07 scans are recorded as clean except for the identified expected multi-colon probe operation ID.

P2 must choose and document the actual handling policy for that record:

- Deterministic, collision-safe encoding, with a test proving the encoded identity remains unambiguous; or
- A fail-closed migration diagnostic requiring explicitly documented manual intervention.

Do not leave the migration behavior as an unresolved choice between automatic remediation and failure. Never silently choose an owner or rewrite an ambiguous identity.

Prove both fresh-install and upgrade paths.

### 3.7 Cancellation withdraw state

`CANCELLED` is accepted as the M3 terminal state for a provisional effect withdrawn before attempt authorization.

P2 must demonstrate that a real cancellation path produces it. Required evidence includes `TestPendingWithdrawnOnCancel` and the pause-between-`PENDING`-and-`ATTEMPTING` test.

If empirical testing shows that the state cannot be produced under the actual cancellation protocol, reconsider the state rather than retaining an unreachable transition.

### 3.8 Other carried requirements

- Preserve the exact five-step X-09 affinity test; do not substitute weaker simultaneous-worker or marker-dedup tests.
- Complete X-08's attachment lifecycle and DETACH-ONLY regression tests.
- Provide the operator-accessible `settle` verb with canonical authorization and attributable audit.
- Document the lifecycle sweeper as an operational component, including failure visibility, backlog, startup recovery, and its conditional guarantee.
- Keep the OpenCode model preflight blocked and unrun.
- Preserve the original P2 exit requirement for independent implementation review.

## 4. Grant-revocation limitation — owner acceptance with Conditions A–E

**Decision: ACCEPTED AS A DEPLOYMENT RESIDUAL, WITH CONDITIONS.**

The framework's in-process grant race is addressed by the pinned-snapshot `RWMutex` protocol. Cross-replica coherence remains a deployment responsibility because grants are loaded once and have no hot-reload path.

A DB-backed grant source or shared revocation mechanism is not pulled into P2. It remains a future architectural direction.

The residual is accepted only with the following conditions:

### A. Config-version visibility

Expose a deterministic grant-configuration hash in process health and startup reporting.

`GET /healthz` returns:

`{ "status": "...", "grant_config_hash": "..." }`

A startup log line records the same hash.

The hash must identify the effective grant configuration without exposing credentials or configuration contents.

### B. Revocation verification

Provide:

`miniapp auth verify-revocation <expected-hash>`

The command polls the health endpoints for all replicas and reports stale, unreachable, or mismatched replicas. It must not report successful completion when the replica inventory is incomplete or a replica cannot be verified.

The implementation must document how it obtains the complete replica inventory.

### C. Bounded deployment window

Use **10 minutes** as the P2 baseline maximum staleness window unless the owner explicitly changes it.

A rolling deployment that exceeds the window is a deployment failure, not an indefinitely extended revocation window.

Rollback must never restore the revoked grant. It may restore a safe application artifact or deployment configuration only while preserving the new restrictive grant configuration.

If the deployment topology cannot enforce a bounded window, production use remains blocked pending a fleet-wide barrier or shared revocation mechanism.

### D. Honest revocation claims

Audit the contract, comments, operator documentation, and status reporting for claims implying immediate or globally instantaneous revocation.

Use the accurate guarantee:

> Revoked grants are rejected on the next authorization performed by a replica holding the current grant configuration.

Do not claim that revocation immediately affects replicas still running an older configuration.

If a required invariant cannot be satisfied without immediate revocation, stop and report the dependency instead of weakening the invariant.

### E. Production operations guidance

Add the requirements to `docs/phase1/classd-operations.md`:

- Config-hash-aware routing, drain-before-cutover, or both.
- The `miniapp auth verify-revocation` command.
- The maximum staleness window.
- The rollback and failure criteria.
- The procedure for verifying that every replica has loaded the expected configuration.

**Failure handling:** If A or B cannot be implemented in P2, record the residual as `verification pending` and request an owner decision at P2 exit. If C cannot be bounded, production deployment is blocked. If D reveals a non-downgradable immediate-revocation dependency, stop and report it.

The residual remains recorded in the amendment log and must appear in the independent re-audit. Acceptance does not remove it.

## 5. P2 exit decision rule

P2 exit is not accepted merely because the listed tests exist or because implementation reports them as passing.

The exit report must provide:

1. Test results and exact red/green source and harness bindings.
2. The updated contract and amendment log for any corrections discovered during implementation.
3. Migration preflight evidence and the concrete multi-colon handling decision.
4. Operator-visible quarantine and settlement behavior.
5. Completed lock-order analysis and concurrency evidence.
6. Grant-revocation Conditions A–E, or the exact unresolved condition and its required owner decision.
7. Independent implementation review findings and dispositions.
8. An updated evidence register and source-scope digest.
9. Explicit confirmation that standing boundaries were preserved.

Any unmet condition is reported as unmet. Do not mark it complete by reclassifying it, weakening its test, or silently changing the contract.

## 6. Authorization and boundaries

P1 is accepted. P2 is authorized to begin against the amended contract and this consolidated exit directive.

This acceptance does not authorize unrelated architecture redesign, Class B/C implementation, Phase 4 effects reconciliation, real provider integrations, production database mutation, or changes to protected documents.

OpenCode remains `NOT RUN / BLOCKED`. No commit, push, or remote mutation is implied.

**Final rule: the framework owns reliability; miniapps own business logic.** P2 must demonstrate that reusable lifecycle, authorization, recovery, and cancellation guarantees are enforced by framework primitives rather than delegated to each miniapp.

**P2 exit remains subject to owner acceptance of the evidence and independent implementation review.**

The two findings I added beyond the supplied acceptance review are narrowly scoped: settlement must recover from `RECONCILING`, and lock-order documentation must account for the omitted paths. They do not reverse P1 acceptance; they prevent those gaps from being implemented on assumptions.