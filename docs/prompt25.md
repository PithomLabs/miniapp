Gate 1 decision: NOT CLEAR — targeted correction required
I accept the independent review's verdict. The good news is that it found two narrowly scoped HIGH defects, not a failure of the overall authorization architecture. The reviewer confirms that the central no-send-without-committed-ATTEMPTING invariant holds, but terminality and ledger integrity are compromised. 
adv_review_prompt24.md

Do not continue the remaining Stage 3A expansion yet. First correct the two HIGH findings, the C2 lock-order violation, and resolve the two owner decisions. Then we conduct a targeted independent re-review—not another full review of the entire architecture.
1. My owner decisions
Finding	Decision
P24-001 — withdrawal rolls back	MUST FIX
P24-002 — stale attempt overwrites newer outcome	MUST FIX
P24-003 — Settle lock order 5 → 1	MUST FIX
P24-004 — M5 sequence boundary	Defer verifier implementation under an explicit activation gate
P24-005 — gated + autonomous trust	Choose option (a), with strict server-derived trust validation
P24-006 — formatting/checks	Fix before new evidence
P24-007 — caller-minted attempt IDs	Framework must mint them
P24-008 — capability membership	Mandatory before the first non-mock provider
P24-009 — policy guard gap	Strengthen now
P24-010 — contention message	Correct now
P24-011 — missing invocation classification	Correct now

P24-004: M5 deferral
Keep sequence fencing probe-scoped. Do not introduce a generic production sequence subsystem or migration now.

However, the current presence-based classification is not an acceptable future design. The activation trigger must require a framework-controlled, tool/workload-declared sequence requirement. When that requirement is active, missing trusted identity or verifier must fail closed. An ordinary caller must never be able to omit Intent.Sequence and thereby downgrade a sequence-bound effect.

Defer the actual production verifier and probe wiring while the sequence path remains mock-only. Before any non-mock sequence-bound provider is enabled, require trusted identity propagation, the verifier, authorization-time generation validation, and the X-08 supersession barrier test. The current probe's missing generation fence must be explicitly recorded as a mock-only limitation, not a proven guarantee. 
adv_review_prompt24.md

P24-005: autonomous trust
Choose option (a). Preserve the existing trust model.

For a gated tool, human approval is required when the authoritative persisted invocation trust is confirm. The autonomous path may proceed without an approval linkage only when:
- The persisted invocation trust is autonomous.
- The admitted manifest explicitly permits autonomous trust for that tool.
- The trust decision comes from the authoritative framework record, never from caller-supplied Intent.
- All other authorization checks remain mandatory: tenant, invocation, tool, operation, grant, kill switch, token, ownership, and any required sequence fence.
- A denied, expired, or mismatched approval must never be bypassed by changing the caller's trust claim.
This preserves the already-offered Required + Confirm/Autonomous configuration while making its effect-time semantics coherent. Add explicit authorization-matrix coverage. 
adv_review_prompt24.md

2. Next work order — Prompt 25
Prompt 25 — WS-B Phase 2: Gate 1 Blocking Corrections
Repository: github.com/PithomLabs/miniapp
Workstream: WS-B Phase 2
Previous review: Prompt 24 — Independent Adversarial Review, Gate 1
Previous verdict: NOT CLEAR
Objective: Correct P24-001 and P24-002, resolve P24-003, implement the owner dispositions for P24-004/P24-005, and prepare a narrowly scoped independent re-review.
1. Mission
Perform bounded, root-cause corrections against the actual working tree.

The authorization architecture is accepted as the foundation. Do not redesign it.

The two HIGH findings are release blockers:
- P24-001: terminal withdrawal does not persist, but its audit claims it did.
- P24-002: stale workers can overwrite newer attempts because outcome transitions are not fenced by attempt identity.
P24-003 violates the mandatory global lock order.

Correct these defects at their shared enforcement boundaries, with discriminating PostgreSQL 19 regression tests.

Do not continue the outstanding Stage 3A expansion until the targeted independent re-review returns CLEAR.
2. P24-001 — Make withdrawal and its audit genuinely atomic
Required invariant:

When irreversible terminal invocation invalidation wins before effect authorization, a pending effect must become terminal CANCELLED, and its audit must truthfully record that withdrawal.

The review established that the current withdrawal UPDATE rolls back because its enclosing transaction returns a refusal error. The audit nevertheless commits separately. This produces PENDING plus a false effect-withdrawn audit.
Required correction
Prefer the following transaction structure:
1. Begin the existing authorization transaction.
2. Perform the authoritative checks in the approved lock order.
3. If terminal invocation invalidation requires withdrawal, execute the predicated PENDING → CANCELLED UPDATE.
4. Check the affected-row count.
5. Insert effect-withdrawn only when the withdrawal actually succeeds.
6. Return a successful transaction-closure result containing the terminal refusal decision, allowing the database transaction to commit.
7. After the commit succeeds, return the appropriate typed refusal to the caller.
8. If commit fails, return infrastructure failure. Never report a committed withdrawal or emit a successful withdrawal audit.
An equivalent implementation is acceptable only if it proves the same transactional invariant and cannot authorize the effect between terminal detection and withdrawal.

Do not use a separate audit transaction that can commit independently of the withdrawal.
Required behavior
- Exactly one transition wins.
- Exactly one truthful withdrawal audit accompanies the winning transition.
- Repeated access to an already-CANCELLED effect does not resurrect it or manufacture repeated withdrawal events.
- A concurrent cancellation and authorization converge correctly.
- Quarantined invocations remain resumable and are not withdrawn merely because they are quarantined.
- Approval mismatch, tenant mismatch, operation mismatch, transient refusal, wait, contention, and infrastructure errors never withdraw an unrelated effect.
- Lazy expiry does not invent an irreversible terminal decision.
Regression evidence
Re-implement POC-1 as a committed test with the same deterministic staging shape.

Assert:
- The invocation is terminal.
- The effect row is CANCELLED.
- Exactly one correctly attributed effect-withdrawn audit exists.
- No ATTEMPTING transition occurs.
- No provider call occurs.
- Retry does not resurrect the effect.
Also retain the opposite forced ordering:
- Authorization commits first: effect is ATTEMPTING.
- Cancellation then reports the effect as in-flight.
- Cancellation does not pretend to recall an already-authorized provider call.
Freeze identical test bytes for pre-fix and post-fix runs wherever the historical pre-fix state can be reproduced.
3. P24-002 — Fence every outcome transition by attempt identity
Required invariant:

An attempt may settle only its own attempt epoch. A stale worker must never overwrite the state or outcome of a newer attempt.

The existing transition() predicates on operation, tenant, and state but omits attempt_id.
Required correction
Add the expected attempt identity from the authoritative in-hand record to every outcome transition predicate:

WHERE operation = ... AND tenant = ... AND state = ... AND attempt_id = ...

Verify that every caller passes the actual attempt identity belonging to its execution epoch.

A zero-row transition must be classified as stale/superseded or the existing appropriate non-success outcome. It must not silently succeed or fall through into an unpredicated write.

Do not replace attempt fencing with:
- An in-memory mutex.
- A process-local worker identity.
- A lease-age assumption alone.
- A fresh read followed by an unpredicated UPDATE.
The database predicate is the enforcement boundary.
Required regression evidence
Re-implement POC-2 as a committed deterministic regression.

Force the following sequence:
1. Epoch A authorizes and blocks during provider execution.
2. Epoch A becomes stale.
3. The operation is reconciled as NOT_APPLIED.
4. Epoch C obtains a fresh authorized attempt.
5. Epoch C blocks during provider execution.
6. Epoch A's late result is released.
7. Epoch A must receive a stale/superseded outcome and must not modify epoch C's row.
8. Epoch C settles with its own receipt.
Cover all three late outcomes:
- CONFIRMED.
- FAILED.
- UNKNOWN.
Assert attempt ID, state, receipt attribution, affected-row count, and actual mock-provider attempt records.

No stale worker may overwrite a newer attempt's result.

Use the existing attempt column. No migration.
4. P24-003 — Remove Settle's lock inversion
Preserve the global lock ordering.

Settle currently obtains the effect lock and subsequently obtains an invocation FOR UPDATE lock for audit attribution.

Replace the attribution loader with a plain lock-free read of immutable invocation identity columns.

The Settle transaction should retain its effect-row ownership and atomic state-plus-audit behavior without acquiring invocation lock 1 after effect lock 5.

Verify:
- No hidden helper reacquires invocation locks.
- No new advisory locks appear.
- The updated implementation conforms to the contract's §4.2 Settle row.
- Existing Settle idempotence and conflict semantics remain unchanged.
Add a deterministic TA-versus-Settle lock-order test where appropriate. Both transactions must converge without a deadlock cycle.
5. P24-004 — Apply the M5 owner disposition
Keep the M5 sequence subsystem probe-scoped.

Do not add a generic production sequence table or migration.

Update docs/phase2/authorize-attempt-design.md and the lifecycle contract to establish:
1. The current M5 verifier and probe identity wiring are deferred because sequence effects remain mock-only.
2. This is an explicit scope deferral, not an implemented generation-fencing guarantee.
3. The current probe's absence of authorization-time generation validation is a named limitation.
4. Sequence requirement must eventually be declared by trusted framework/tool/workload configuration, not inferred from the presence of caller-supplied Intent.Sequence.
5. When sequence-required classification is active, missing identity or verifier fails closed.
6. Identity must be framework-originated and bound to tenant, invocation, and effect.
7. The verifier must use the authorization transaction at lock-order position 3.
8. It must not start transactions, perform I/O, or acquire advisory locks.
9. Before any non-mock sequence-bound provider is enabled, implement the verifier, trusted identity propagation, supersession validation, and X-08 barrier test.
The future activation trigger must be explicit and enforceable. Do not use a vague phrase such as "when sequence workloads are needed."
6. P24-005 — Implement the autonomous-trust semantic decision
Owner choice: option (a).

Human approval is conditional on the authoritative persisted invocation trust.

For gated tools:
- Confirm trust requires a valid persisted granted approval and full binding verification.
- Autonomous trust may omit the human approval linkage only when the persisted invocation trust is autonomous and the admitted tool manifest explicitly permits autonomous execution.
- Caller-provided trust or Intent fields cannot establish this exception.
- Tenant, invocation, tool, operation, authority, grants, kill switches, token, effect ownership, and any applicable sequence requirements remain mandatory.
- A denied, expired, mismatched, or cancelled approval must not be bypassed through a caller-supplied trust claim.
- A manifest that does not permit autonomous trust must refuse autonomous execution.
Correct the inconsistency at the shared authorization boundary without weakening the other authorization checks.

Add explicit tests for:
1. Gated + Confirm trust + valid approval → authorized.
2. Gated + Confirm trust + missing approval → refused.
3. Gated + Autonomous trust + manifest permits autonomous → authorized without human approval linkage, subject to all other controls.
4. Gated + Autonomous trust + manifest does not permit autonomous → refused.
5. Caller-supplied autonomous claim with persisted Confirm trust → refused.
6. Denied/expired/mismatched approval cannot be bypassed.
7. Terminal or cancelled invocation cannot exploit the autonomous exception.
Update Prompt 22-derived contract language and the authorization design.
7. P24-006 through P24-011 — Bounded cleanup
Apply these corrections within the same work order where they remain local and do not expand architectural scope.
P24-006 — Formatting and project checks
Run gofmt on the identified files.

Verify gofmt -l . produces no output and scripts/checks.sh passes before creating new evidence snapshots.
P24-007 — Framework-owned attempt IDs
Implement the Stage 2 design requirement that attempt IDs are framework-minted ownership epochs.

Remove caller free-text control of fresh attempt IDs. Generate or validate them at the shared framework boundary.

Preserve attempt IDs when an existing logical operation is adopted. New ownership epochs must receive fresh framework-generated identifiers.

Test empty, malformed, colliding, and retry cases.
P24-008 — Capability membership
Record this as a mandatory prerequisite before enabling the first non-mock provider.

Do not expand the mock-only provider workstream unnecessarily. Before real provider support, authorization must verify that the requested capability belongs to the admitted tool's declared capability set.
P24-009 — Strengthen the Policy guard
Extend the guard to detect whole-map assignments and direct field access that could bypass the setter/hash discipline.

Prefer AST-based checking if it is the smallest durable solution. Add the negative-control fixture for whole-map assignment.
P24-010 — Correct contention messaging
Qualify effect-contention instructions according to invocation state.

Do not universally promise same-key convergence where running Class A records follow interrupted/new-key semantics.

Keep the distinction between request-level conflict and persisted invocation terminality explicit.
P24-011 — Classify missing invocation
Map a missing authoritative invocation row to a terminal binding refusal rather than an infrastructure-shaped transient error.

Apply the equivalent deterministic classification to Settle attribution.

Keep genuine database connectivity and execution failures retryable.

Ensure the resulting audit does not falsely attribute an operation to another invocation.
8. Required test discipline
Use PostgreSQL 19.0, the exact pinned image digest:

sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60

Re-resolve and record the full digest using Docker inspection. Do not rely on the abbreviated image ID or floating tag alone.

Use disposable databases and real independent PostgreSQL sessions.

For P24-001 and P24-002, preserve the exact deterministic POC staging shape.

For P24-003, verify the lock acquisition graph after the correction.

For P24-005, verify the trust decision derives from authoritative persisted state.

Do not use sleeps, goroutine counts, or process-local locks as proof of database concurrency behavior.

Maintain the evidence distinction:
- Discriminating red/green.
- Green-only regression.
- Characterization.
- Source-only proof.
- Composition proof.
- Accepted residual.
- Blocked/unverified.
Never claim a red/green pair where the same test passes on both sides.
9. Required order of work
1. Freeze Stage 3A pre-fix source and test snapshots.
2. Fix P24-001 and add its committed regression.
3. Fix P24-002 and add all three late-outcome regressions.
4. Fix P24-003 and verify lock order.
5. Implement P24-005 semantics and tests.
6. Record P24-004's explicit deferral and activation trigger.
7. Complete bounded P24-006 through P24-011 actions.
8. Run focused tests and affected package suites.
9. Run go vet, formatting, and scripts/checks.sh.
10. Verify PostgreSQL 19 image identity and execution environment.
11. Update contract, design, and evidence register.
12. Produce the targeted correction report.
Do not proceed to the outstanding full authorization/crash matrix during this correction work.
10. Required deliverable
Create:

docs/phase2/prompt25-gate1-corrections-report.md

Include:
- Exact baseline and final tree identity.
- PostgreSQL 19 version and full image digest.
- P24-001 source correction and committed regression evidence.
- P24-002 attempt-fencing correction and all outcome-flavor regressions.
- P24-003 lock-order correction.
- P24-004 owner disposition and activation trigger.
- P24-005 trust semantics and matrix results.
- P24-006 through P24-011 dispositions.
- Exact red/green test hashes and execution commands.
- Full affected-suite, vet, formatting, and checks results.
- Remaining Stage 3A work.
- Current worktree state.
- Explicit confirmation: no commit, push, deployment, or real provider write.
11. Stop and re-review
Once corrections are complete, stop.

Do not continue the rest of Stage 3A before targeted independent re-review.

Prepare a reviewer package containing:
- The original Prompt 24 report.
- The correction report.
- P24-001 and P24-002 pre-fix and corrected snapshots.
- Committed regression tests.
- Relevant source diffs.
- Updated design and lifecycle contract.
- Exact PostgreSQL 19 execution evidence.
- C1–C7 compliance updates.
The targeted reviewer must independently verify the two HIGH corrections, P24-003 lock order, owner dispositions, and affected regressions.

A CLEAR result from that targeted re-review releases the project to complete the remaining Stage 3A matrix. It does not itself mean Phase 2 is complete.

No CRITICAL or HIGH defect may remain.

12. Final instruction
Fix the underlying state-transition and ownership defects. Do not paper over them with error mapping, retry flags, test-only hooks, or additional logging.

Keep the solution localized to the shared framework boundary.

Do not reopen Phase 1. Do not broaden M5. Do not weaken the autonomous-trust, tenant, approval, or grant model.

Stop at targeted Gate 1 re-review.




3. What happens after Prompt 25
1. Prompt 25 — Corrections
   
   Fix the two HIGH findings, C2, and the bounded owner dispositions.
2. Prompt 26 — Targeted independent re-review
   
   Independently verify the corrections. CLEAR releases the remaining Stage 3A work.
3. Complete Stage 3A
   
   Finish the full authorization matrix, cancellation and worker races, crash matrix, C6 integration, revocation tooling, settlement crash tests, provider bypass verification, and CI parity.
4. Gate 2 review
   
   Independent review of the integrated Stage 3A implementation.
5. Stage 4 and final Gate 3
   
   Complete full adversarial verification, then independently assess Phase 2 acceptance.
Your immediate next action is Prompt 25. Do not ask the coding agent to continue broad implementation until the targeted re-review clears the blockers.