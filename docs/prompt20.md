I recommend **Prompt 20: Bounded Phase 1 Closeout**. It authorizes the three remaining correctness issues, prohibits architectural wandering, and makes a focused independent re-review the final gate before Phase 2.

# Prompt 20 — Bounded WS-B Phase 1 Closeout

**Repository:** `github.com/PithomLabs/miniapp`\
**Governing work order:** `docs/prompt17.md`\
**Independent review:** `docs/adv_review_prompt19.md`\
**Status:** Phase 1 closeout authorized; Phase 2 remains blocked.

## 1. Mission

Close only the structural and foundational findings identified by Prompt 19.

The owner has reviewed the findings and made the following decisions:

1. HIGH-1: Replace the unreachable Cancel-versus-Class-A-completion race requirement with a reachable settlement-versus-completion race.
2. MEDIUM-1: Preserve CR-05 workflow identity immutability by fixing `failIfWaiting` and `adoptRunID`.
3. MEDIUM-2: Preserve framework-enforced single execution. Do not downgrade Class A handler execution to at-least-once.
4. MEDIUM-3: Correct the contradictory advisory-lock instruction in Prompt 17 and its plan-review mirror.

The goal is not another comprehensive remediation cycle. It is to close these specific structural gaps with the smallest durable corrections supported by the existing architecture.

**Do not expand scope merely because additional improvements can be identified.**

The governing principle remains:

**The framework owns reliability; miniapps own business logic.**

## 2. Baseline and evidence freeze

Before implementation:

1. Read Prompt 19 and its complete finding details.
2. Verify the submitted Phase 1 green tree, reported as `5afb6e46…`.
3. Verify the Phase 1 red tree, reported as `c9c08fe5…`.
4. Verify `runtime/m01_test.go` and its full SHA-256.
5. Preserve immutable directory copies of the red and green baselines.
6. Record the working-tree digest, register pre-image, Git revision, and environment.
7. Do not modify the historical Prompt 19 review report.

Use the exact commands and path conventions recorded in the evidence procedure.

Every new regression test must be run against the appropriate pre-fix implementation and the corrected implementation, using identical test bytes.

Do not use Git branches, commits, pushes, resets, or rebases. Continue the established isolated-directory-copy discipline.

If the actual source differs from the identified baseline beyond documented evidence additions, report the discrepancy before proceeding.

## 3. HIGH-1 — settlement versus completion

### Owner-approved requirement change

The original public Cancel-versus-Class-A-completion scenario is unreachable because Class A in-flight records do not have a `run_id`, while the current cancellation API addresses invocations by `run_id`.

Do not redesign cancellation merely to manufacture this race.

Replace the original race requirement with:

**Deterministically prove that approval settlement and synchronous Class A completion cannot produce incoherent lifecycle or audit outcomes when competing for the same invocation.**

Implement two deliberately forced winner scenarios.

**Scenario A — settlement wins**

- Approval denial or expiry settlement commits first.
- Synchronous handler completion subsequently loses.
- Completion discards the stale handler result.
- The response reflects authoritative persisted state.
- No false success or completion audit is emitted.

**Scenario B — completion wins**

- Synchronous completion commits first.
- Approval settlement subsequently loses.
- Settlement reloads the authoritative record.
- The response reflects the completed outcome.
- No false denial or expiry audit is emitted.

### Test requirements

Use:

- The actual Class A granted execution path.
- A channel-blocked handler.
- The established PostgreSQL lock-gate methodology.
- Both explicitly controlled winner orders.
- Persisted invocation and audit assertions.

Do not rely on sleeps, scheduling luck, or repeated probabilistic races.

Also verify terminal immutability at the store boundary. Do not claim public cancellation coverage for run-less Class A records.

Correct the `completeIfLive` comment to describe the actual competing settlement path rather than Cancel.

**Exit condition:** both winner scenarios fail against the appropriate pre-remediation behavior and pass against the corrected implementation.

## 4. MEDIUM-1 — CR-05 workflow identity integrity

Implement the two narrow corrections identified in Prompt 19.

### A. `failIfWaiting`

Remove the `run_id` write entirely.

Approval settlement owns the approval-related status and error transition. It must not overwrite workflow identity using a value captured from an earlier read.

### B. `adoptRunID`

Add a database-level live-state predicate.

Permitted states must include `pending`, because restoration dispatch can legitimately adopt from that state.

Terminal and quarantined records must be rejected.

If adoption loses, the service must reload authoritative state and must not submit a workflow for an ineligible invocation.

The store predicate is the primary guarantee. Service-level revalidation is defense in depth.

### Required tests

1. A stale expiry poll loses its race after another path adopts run `R`; `R` remains persisted.
2. Adoption against a terminal record is rejected.
3. A losing or rejected adoption cannot launch a DBOS workflow.
4. Both relevant interleavings are deterministically exercised.

Assert persisted run identity, invocation state, and workflow-submission count.

**Exit condition:** no settlement operation can overwrite canonical run ownership, and no terminal record can acquire a new run through adoption.

## 5. MEDIUM-2 — framework-enforced single Class A execution

This is the most important architectural closeout item.

The existing behavior can permit concurrent granted polls or duplicate dispatches to execute the same Class A handler more than once, even though only one terminal record update wins.

The owner explicitly rejects downgrading the framework guarantee to at-least-once handler execution.

### Step 1 — focused design investigation

Before implementation, produce:

`docs/phase1/classa-execution-claim-design.md`

Keep this document focused on the smallest safe solution.

Inspect the existing Class A transaction model, advisory-lock behavior, handler invocation, approval polling, same-key dispatch, and lifecycle completion.

Determine how to establish a framework-owned, database-coordinated execution guarantee across separate processes.

The design must answer:

1. What is the atomic execution-claim boundary?
2. How do competing granted polls and duplicate dispatches establish one winner?
3. How does a loser reload and return the authoritative result?
4. What happens if a worker crashes after claiming but before executing?
5. What happens if handler database writes commit but the response is lost?
6. Can recovery accidentally execute the business handler a second time?
7. How are stale execution owners prevented from committing?
8. How are pending, running, waiting-approval, and terminal states distinguished?

Do not assume an in-memory mutex, process-local single-flight mechanism, or advisory lock alone provides distributed execution ownership.

Analyze whether the existing Class A transaction can atomically coordinate handler database writes with invocation completion.

Preserve transactional semantics and existing lock ordering. Do not hold database locks across external operations.

### Step 2 — minimum durable implementation

Implement the smallest mechanism that provides the approved guarantee.

Required behavior:

- At most one concurrent execution of a given invocation reaches the handler.
- Losers do not execute the handler.
- Losing requests observe or wait for the authoritative outcome.
- A crash cannot leave a permanently claimed invocation that cannot recover.
- A committed handler operation cannot be silently executed again because the caller lost its response.
- Stale owners cannot commit after losing execution ownership.
- The behavior works across separate processes, not just goroutines.

Prefer existing database transaction and lifecycle mechanisms. Avoid introducing new durable states, migrations, or broad API changes unless the existing architecture genuinely requires them.

Do not solve this by transferring idempotency responsibility to individual miniapp authors.

If the investigation establishes that the minimum safe solution requires a substantial architectural change, new persistent state, or a migration beyond the existing Phase 1 scope, stop this item and submit a short owner decision memo containing:

- The proven limitation.
- The smallest safe alternatives.
- Their failure and recovery semantics.
- The required contract changes.
- Why a smaller correction cannot satisfy the guarantee.

Do not start a broad redesign.

### Required discriminating tests

Use a channel-blocked Class A test handler and deterministic synchronization.

Test:

1. Two concurrent granted polls for the same invocation.
2. Concurrent same-key dispatch and replay.
3. One handler execution and one coherent terminal transition.
4. Accurate audit behavior.
5. Process failure before handler transaction commit.
6. Process failure after successful handler commit but before response delivery.
7. Recovery cannot cause duplicate committed business writes.

Assert actual handler execution counts and persisted database state, not API responses alone.

**Exit condition:** the framework enforces one execution without introducing an unrecoverable claim or allowing duplicate committed business work.

## 6. MEDIUM-3 — correct the Phase 2 lock-order instructions

Before any Phase 2 design begins:

- Correct `docs/prompt17.md`.
- Correct the matching advisory-lock instruction in `docs/prompt17_plan_review.md`.
- Make lifecycle contract §4 authoritative.
- Remove advisory-last ordering.
- Explicitly state that the Phase 2 `authorizeAttempt` transaction does not acquire an advisory lock.
- Preserve the outermost-first rule for operations that actually require advisory locking.
- Record the correction through the never-silent amendment process.

Perform a repository-wide search for contradictory advisory-last instructions.

Do not modify the verified advisory-lock implementation.

## 7. Small closeout items

Complete these inexpensive items without expanding the workstream.

| Finding | Required disposition                                                                                                  |
| ------- | --------------------------------------------------------------------------------------------------------------------- |
| LOW-1   | Emit `duplicate` audit only when the claim actually identified an existing invocation; add a losing-first-claim test. |
| LOW-2   | Record missing post-win audit backfill as deferred X-12 work. No reconciler or sweeper implementation.                |
| LOW-3   | Correct contract phase markers, claim-response timing wording, and advisory-decision provenance.                      |
| INFO-1  | Define the live-state terminology and document operation-specific exclusions.                                         |
| INFO-2  | Add a lightweight mechanical guard restricting production advisory acquisition to the Class A framework path.         |
| INFO-3  | Accept 32-bit hash contention as a liveness-only residual. No change.                                                 |
| INFO-4  | Correct the review documentation's completion-state table to match implementation semantics.                          |
| INFO-5  | Defer quarantine defensive-path semantics to the appropriate later workstream.                                        |
| INFO-6  | Record exact digest commands, working directory, and path forms.                                                      |
| INFO-7  | Document run-less Class A cancellation and effective-trust behavior accurately.                                       |

Do not turn informational observations into additional architectural projects.

Do not add Phase 2 functionality while addressing these items.

## 8. Verification and exit evidence

For every corrected finding:

- Record the exact source and test changes.
- Preserve identical test source on red and green sides.
- Use isolated pre-fix copies.
- Demonstrate intended red failure.
- Demonstrate corrected green behavior.
- Record full source and tree digests.
- Record exact commands, environment, PostgreSQL version, and results.
- Verify race behavior and cleanup.
- Assert persisted state, handler execution counts, and audit evidence.

Run:

```
gofmt -l .
go vet ./...
go test -p 1 ./...
go test -race -p 1 ./runtime/ ./approvals/ ./effects/
bash scripts/checks.sh
```

Run targeted tests independently and repeatedly where useful. Do not mistake repetition for deterministic synchronization.

Record any tests that are skipped, blocked, or not run. Never represent them as PASS.

Preserve local PostgreSQL 19.0, mock-only external effects, forward-only migrations, and all other existing safety boundaries.

## 9. Required deliverables

Produce:

1. `docs/phase1/classa-execution-claim-design.md` — focused design investigation.
2. `docs/phase1/prompt20-phase1-closeout.md` — implementation and evidence report.
3. Updated evidence register with immutable pre-image and reproducible digests.
4. Corrected governing documentation and tests within the approved scope.

The closeout report must distinguish:

- Completed and red/green verified.
- Source-correct but not independently proven.
- Accepted residual limitations.
- Deferred work.
- Any remaining architectural decision.

Do not rewrite `docs/adv_review_prompt19.md`. Preserve it as the historical independent review.

## 10. Stop conditions and Phase 2 release

This is a bounded closeout, not an invitation to reopen the entire architecture.

Do not:

- Re-review already accepted WS-A work.
- Rework the predicated-transition mechanism where its correctness is already established.
- Redesign DBOS integration generally.
- Introduce Class B or C execution.
- Add an effects reconciler.
- Implement real provider integrations.
- Implement `lead-followup` or approve its ADR.
- Add unrelated framework primitives.
- Change CI or worker startup behavior unrelated to a proven defect.
- Commit, push, create branches, or use Neon/OpenCode.

Stop only for:

- A genuine conflict with governing architecture.
- A newly demonstrated CRITICAL or HIGH defect outside the authorized corrections.
- A safety or authorization boundary that cannot be preserved.
- A MEDIUM-2 design that cannot satisfy the guarantee within the existing architecture without a substantial owner decision.

Do not stop for cosmetic findings or routine documentation improvements within the approved scope.

### Phase 2 release criteria

Phase 2 may be proposed for release only when:

1. HIGH-1 is closed with both deterministic winner scenarios.
2. MEDIUM-1 is closed with CR-05 race proofs.
3. MEDIUM-2 is closed with a framework-enforced single-execution guarantee and crash/recovery evidence.
4. MEDIUM-3 is corrected in all governing instructions.
5. The small closeout items have their approved dispositions.
6. No CRITICAL or HIGH finding remains.
7. The full serial verification passes.
8. The evidence binds every important claim to an exact source state.
9. The focused independent re-review returns CLEAR.

A passing test suite alone is insufficient.

**The objective is to close the foundational blockers and proceed. Do not seek perfection, invent new phases, or turn accepted residual risks into endless remediation.**

**Phase 2 remains prohibited until independent closeout review and explicit owner release.**

**My recommendation:** use this as the next coding-agent instruction. It makes MEDIUM-2 the only item requiring a focused design decision, while keeping the other corrections bounded and test-driven.