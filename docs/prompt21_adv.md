I recommend naming this **Prompt 21 — Independent Adversarial Review of WS-B Phase 1 Closeout**. It evaluates whether the entire closeout is genuinely ready to serve as the foundation for Phase 2, with particular attention to the five review priorities identified in §10 of the report.

# Prompt 21 — Independent Adversarial Review of WS-B Phase 1 Closeout

**Repository:** `github.com/PithomLabs/miniapp`\
**Review target:** Prompt 20 — WS-B Phase 1 Closeout\
**Primary report:** `docs/phase1/prompt20-phase1-closeout.md`\
**Review protocol:** Independent, adversarial, read-only\
**Next phase:** WS-B Phase 2 — Effect-Authorization Transaction\
**Phase 2 status:** BLOCKED pending independent CLEAR and explicit owner release.

---

## 1. Mission

Act as an independent principal engineer conducting a final adversarial review of the completed Prompt 20 Phase 1 closeout.

Your responsibility is to determine whether the framework now has a sufficiently sound foundation for Phase 2 effect authorization.

Do not simply validate that:

- All seven implementation tasks are marked complete.
- All reported tests pass.
- The evidence register contains the expected rows.
- Previous findings have been assigned dispositions.
- The author reports that no architectural decisions remain.

Read and independently challenge the actual implementation, tests, contracts, and evidence.

The central question is:

**Does the completed Phase 1 implementation establish trustworthy lifecycle integrity, single Class A execution, workflow ownership, and transaction semantics on which Phase 2 can safely build?**

This is a bounded release-gate review, not an invitation to redesign the framework.

Be especially skeptical of the interaction between the newly implemented Class A single-transaction execution mechanism and the forthcoming Phase 2 effect-authorization transaction.

Do not implement Phase 2.

Do not fix findings during the review.

## 2. Governing sources

Read the actual repository versions of:

1. `docs/plan4.2.md` — immutable governing architecture.
2. `docs/prompt15.2.md` — governing implementation instructions.
3. `docs/prompt17.md` — corrected WS-B work order.
4. `docs/prompt20.md` — bounded Phase 1 closeout authorization.
5. `docs/adv_review_prompt19.md` — independent Phase 1 review.
6. `docs/phase1/invocation-lifecycle-contract.md` — current normative contract.
7. `docs/phase1/classa-execution-claim-design.md` — MEDIUM-2 approved design.
8. `docs/phase1/prompt20-phase1-closeout.md` — submitted implementation and evidence report.
9. `docs/evidence/README.md` — evidence register and provenance.
10. All referenced manifests, register pre-images, red/green overlays, and relevant test files.

Do not rely on report summaries in place of source inspection.

Preserve the distinction between:

- Normative contract requirements.
- Owner-approved decisions.
- Implemented behavior.
- Independently reproduced tests.
- Source-correct but unproven claims.
- Accepted residual risks.
- Deferred work.

Do not silently reconcile contradictions or promote historical review material into governing instructions.

The governing architectural principle remains:

**The framework owns reliability; miniapps own business logic.**

## 3. Baseline and provenance verification

The closeout report identifies:

| Artifact                        | Reported identity |
| ------------------------------- | ----------------- |
| Phase 1 original green manifest | `5afb6e46…`       |
| Phase 1 original red tree       | `c9c08fe5…`       |
| Prompt 20 frozen green          | `d953a400…`       |
| Prompt 20 frozen red            | `c9c08fe5…`       |
| Prompt 20 closeout manifest     | `a8ffe2f4…`       |
| Final implementation tree       | `e237c51e…`       |
| Register pre-image              | `acff00c6…`       |
| HEAD                            | `4945478`         |

These are submitted claims, not substitutes for independent verification.

Before inspecting behavior:

1. Recompute the final closeout manifest using its recorded procedure.
2. Verify the full final worktree identity.
3. Verify the original Phase 1 red and green baselines remain intact.
4. Verify the original register pre-image and manifest provenance.
5. Verify all five register-bound test files remain byte-identical where required.
6. Reconstruct the relevant per-finding red overlays and verify their source/test composition.
7. Confirm the actual source and test inventory matches the closeout report.
8. Identify all source, test, contract, and documentation changes since the frozen Phase 1 baseline.
9. Record the full Git revision, working-tree status, environment, and exact commands.

The final closeout report was written after the reported manifest generation. Resolve this through the documented manifest/pre-image methodology rather than assuming the report contains its own digest.

Do not infer full SHA-256 values from abbreviated identifiers.

If the submitted source cannot be established exactly, report the discrepancy and assess whether it prevents a CLEAR verdict.

Do not modify, reset, clean, or otherwise mutate the submitted working tree.

Use isolated copies and disposable databases for experiments.

## 4. Scope of review

Review all Prompt 20 closeout claims, with five high-priority investigation areas.

### Priority 1 — MEDIUM-2: Class A single-transaction execution

This is the most consequential change in the closeout.

The implementation combines:

- Invocation-scoped 64-bit namespaced advisory transaction claim.
- Existing Class A entity advisory lock.
- Persisted invocation liveness revalidation.
- Savepoint-guarded handler execution.
- Predicated completion.
- Handler and transactional business writes committed atomically.
- Terminal conflict and same-key retry semantics.

The reported guarantee is:

**At most one concurrent handler execution, and at most one committed transactional business execution per invocation.**

Independently verify that this is actually enforced.

#### A. Execution-claim correctness

Inspect:

- `internal/store` execution-claim implementation.
- `executor.Request` claim hooks.
- `executor/classa/classa.go`.
- `runtime/service.go`.
- Runtime completion and recheck store methods.
- All production callers of `Sync.Execute`.

Verify:

1. The claim is acquired before handler execution.
2. The claim is database-global and coordinates separate processes.
3. Competing attempts cannot both reach the handler.
4. Claim failure is returned without entering the handler.
5. A claim automatically releases on commit, rollback, process death, or connection loss.
6. The namespaced hash domain cannot collide with the entity-lock domain in a way that creates lock-order violations.
7. No production path silently bypasses the execution claim.
8. Nil claim/completion hooks fail closed.
9. The sole production caller and all constructor paths are accurately inventoried.

Do not accept process-local mutexes or test-only synchronization as proof of distributed ownership.

#### B. Single-transaction atomicity

Establish that the actual transaction contains:

`execution claim → entity advisory lock → liveness recheck → handler → predicated completion → commit`

Verify that handler business writes and invocation completion cannot commit separately.

Challenge:

- Error propagation.
- Savepoint creation and rollback.
- Failed handler semantics.
- Completion predicate failure.
- Transaction rollback.
- Database connection loss.
- Process termination.
- Lost response after commit.

Specifically verify the handler-error behavior:

- Handler business writes roll back to the savepoint.
- The failure result can still be recorded.
- The transaction-scoped execution claim remains held until the final outcome commits or the whole transaction aborts.
- Partial handler writes cannot accidentally survive.

Independently test the PostgreSQL savepoint/advisory-lock interaction where possible.

#### C. The `execution-in-progress` result

The implementation introduces:

- `ErrExecutionInProgress`.
- `ErrExecutionSuperseded`.
- `HandlerFailedError`.

Verify their actual classifications and behavior.

The execution-in-progress result must not become a persisted terminal invocation state.

A competing caller must receive a truthful conflict that identifies same-key retry as the convergence path.

Verify:

- The loser returns promptly.
- No handler runs on the losing attempt.
- The invocation row remains unchanged by the losing request.
- Explicit retry after winner completion replays the recorded outcome.
- No implicit retry loop causes repeated handler execution.
- REST's 409 mapping is accurate and does not misclassify unrelated failures.

#### D. Crash and recovery evidence

The closeout identifies:

- T1: concurrent granted polls.
- T2: dispatch-versus-replay.
- C3: retryable conflict.
- T5: pre-commit crash characterization.
- T6: post-handler-commit crash.
- C1/C2/C4: Class A claim and savepoint unit tests.

Inspect the actual test implementations and subprocess harness.

T5 is reported as passing on both red and green. It is a characterization test, not a red/green discriminator. Preserve this distinction.

T6 is the discriminating crash-after-commit regression. Verify that Kill9 occurs at the claimed completion boundary and that persisted business-row counts establish the original double-apply defect.

Determine whether the tests genuinely prove:

- One concurrent handler entry.
- One committed terminal outcome.
- No duplicate committed handler writes.
- Correct recovery after a process crash.
- Correct behavior after a committed transaction whose response is lost.

Do not accept in-memory execution counters as proof of crash-recovery cardinality.

#### E. Future Phase 2 compatibility — highest priority

The execution transaction is now longer-lived and carries two advisory locks before transactional row locks.

Phase 2 will introduce `authorizeAttempt`, with its own required transaction and lock ordering:

`invocation → approval → sequence → policy/grants → effect`

The corrected contract states that Phase 2 authorization does not acquire an advisory lock.

Independently examine whether these designs compose safely.

Investigate:

1. Whether effect authorization is performed inside a Class A handler transaction or in a separate transaction.
2. Whether a nested transaction or pool checkout creates an independent lock lifetime.
3. Whether the invocation completion row lock can interact with authorization's invocation lock.
4. Whether approval or grant revalidation can create a reverse wait path.
5. Whether entity serialization can interact with invocation execution claims.
6. Whether advisory execution claims remain outermost relative to all PostgreSQL locks.
7. Whether a transaction can wait on an authorization operation that itself waits on the executing invocation.
8. Whether a deadlock cycle is possible across multiple processes.
9. Whether failed or rolled-back handler attempts leave effect-ledger state in a recoverable condition.
10. Whether Phase 2's `ATTEMPTING` semantics can be preserved without weakening the Class A guarantee.

Produce an explicit lock and transaction interaction graph.

Do not implement Phase 2 or invent its final transaction structure.

Classify each conclusion as:

- Proven safe by current implementation and contract.
- Safe under a specific documented Phase 2 constraint.
- Unknown and requiring a Phase 2 design decision.
- A genuine blocking incompatibility.

A bounded Phase 2 integration constraint is acceptable. A hidden deadlock or atomicity conflict is not.

### Priority 2 — MEDIUM-1: Fix C and run-ID ownership

The closeout reports three changes:

- `failIfWaiting` no longer writes `run_id`.
- `adoptRunID` includes the allowed live-state predicate.
- `submitChecked` performs `adoptEligible` checks after losing adoption and for already-bound records.

Inspect the code and the actual SQL predicates.

Specifically challenge:

1. Whether application and store predicates match exactly.
2. Whether `pending` is correctly allowed for restoration dispatch.
3. Whether terminal and quarantined states are consistently rejected.
4. Whether a lost adoption correctly reloads the authoritative record.
5. Whether a terminal record with an existing run ID is refused before submission.
6. Whether both no-run and already-bound paths are tested.
7. Whether rejected adoption produces zero DBOS submissions.
8. Whether stale settlement can overwrite canonical run identity.
9. Whether a status change can occur between the liveness check and the submission decision.
10. Whether any alternate submission path bypasses the choke.

#### Fix C branch coverage

This is an explicit closeout review priority.

Do not accept a single test that happens to exercise only the adoption-loss branch.

Inspect the following cases independently:

- Adoption loses and the reloaded record is terminal.
- Adoption loses and the reloaded record remains live.
- Existing run identity is present and the record is terminal.
- Existing run identity is present and the record remains live.
- Rejected adoption with no run.
- Rejected adoption with an existing run.

Verify exact submission counts, run identity, status, and error semantics.

The closeout explicitly labels the winner-terminal-with-run loss branch as source-correct but not independently proven.

Determine whether this is a material release risk.

Do not pretend code inspection equals executed regression evidence. Conversely, do not block on an unstageable test if the store invariant and surrounding checks conclusively establish the property. Explain the reasoning.

Preserve the authorized O-01 residual:

`adopt wins → settlement wins → submit`

Do not silently expand the review into implementing the Phase 2 effect-authorization boundary.

### Priority 3 — HIGH-1: forced settlement/completion winner scenarios

Review `runtime/p20_high1_test.go`.

The approved rescope replaces the unreachable public Cancel-versus-Class-A-completion race with trust-float-dispatched Class A completion versus expiry settlement.

The closeout reports:

- Scenario A: sequential settlement-wins.
- Scenario B: completion-first queueing.
- Three rounds each.
- Persisted-state and exact-audit assertions.
- Identical red/green test bytes.

Independently verify both.

#### Scenario A

Establish:

- The actual settlement commit has completed before the handler is released.
- The handler's stale result cannot overwrite settlement.
- The losing completion does not emit a false success audit.
- Persisted invocation state and response are coherent.

#### Scenario B

Challenge the reported FIFO queue assumption.

The test stages completion as the sole waiter before settlement begins, then uses post-hoc winner assertions.

Determine:

1. Whether the blocking statement is the intended `completeIfLive` operation.
2. Whether the waiter is genuinely established before the competing operation starts.
3. Whether the queue order is demonstrably controlled or merely inferred.
4. Whether PostgreSQL's observed queue behavior is being treated as a guarantee it does not provide.
5. Whether the post-hoc assertions reject an incorrect winner rather than silently accepting it.
6. Whether all three rounds exercise the intended interleaving.
7. Whether red and green use identical test bytes.

Do not confuse a test that eventually observes the desired winner with one that deterministically forces it.

### Priority 4 — LOW-1: duplicate-audit attribution

The closeout changes `replay` to accept an `isDuplicate` flag.

It reports two red/green tests with aggregate marker counts of four versus three.

Inspect:

- Every `replay` call site.
- The true-duplicate claim path.
- The completion-loser path.
- The settlement-loser path.
- The status-sync-loser path.
- The cancellation-loser path.

Verify that only a genuine existing idempotency key produces a `duplicate` audit.

Challenge the aggregate-count approach.

Determine whether a test could pass while assigning the duplicate marker to the wrong request or invocation.

Prefer assertions tied to:

- Invocation identity.
- Request identity.
- Audit actor.
- Event type.
- Actual claim outcome.

Report whether aggregate counts are sufficient evidence or whether per-invocation attribution remains unproven.

Do not demand a broad audit refactor for a bounded labeling issue.

### Priority 5 — MEDIUM-3: governing-versus-historical lock-order scoping

The closeout reports:

- Corrected Prompt 17.
- Corrected plan-review mirror.
- §4 authoritative.
- Advisory-last instructions removed from governing documents.
- Historical reviews and writeups preserved.

Verify this distinction.

Search the entire repository for:

- Advisory-last instructions.
- Conflicting lock-order diagrams.
- References implying Phase 2 acquires an advisory lock.
- Historical material that is clearly marked as superseded.
- Any remaining governing document that could reasonably be mistaken as authoritative.

Inspect the triage inventory.

Do not require historical reports to be rewritten merely because they preserve old findings. They must, however, be distinguishable from current governing instructions.

Verify the corrections-never-silent provenance.

### Priority 6 — Remaining closeout items

Verify the reported dispositions without reopening their architecture.

| Item   | Required verification                                                                                      |
| ------ | ---------------------------------------------------------------------------------------------------------- |
| LOW-2  | X-12 audit-backfill scope is recorded as deferred, not represented as implemented                          |
| LOW-3  | Contract phase markers, claim-response timing, and provenance wording are accurate                         |
| INFO-1 | Live-state terminology matches operation-specific predicates                                               |
| INFO-2 | AST guard pins the sole production advisory acquisition site and has meaningful positive/negative coverage |
| INFO-3 | 32-bit entity hash contention remains an accepted liveness limitation                                      |
| INFO-4 | Prompt 19 protocol table correction matches actual completion states                                       |
| INFO-5 | Quarantine behavior is explicitly deferred to X-10                                                         |
| INFO-6 | Digest commands, CWD, and path forms are reproducible                                                      |
| INFO-7 | Run-less cancellation and trust-float behavior are accurately documented                                   |

Do not turn accepted residuals into additional implementation projects.

## 5. Test and evidence discipline

Assess the quality of each test independently.

For every claimed regression, ask:

1. Does the test exercise the real production path?
2. Does it force the intended boundary?
3. Could defective code still pass?
4. Are test source bytes identical across red and green?
5. Does red fail for the intended reason?
6. Does green assert persisted state, not just return values?
7. Are concurrency conditions established with deterministic synchronization?
8. Does the test account for all required winner/loser outcomes?
9. Does cleanup execute on every failure path?
10. Is `-race` evidence supplementary or genuinely relevant?
11. Is the test a regression discriminator, characterization test, or guard test?

Do not:

- Treat a green test as proof of test discrimination.
- Treat repeated probabilistic races as deterministic.
- Treat a characterization test as red/green evidence.
- Treat source inspection as an executed test.
- Treat omitted execution as a pass.
- Accept weakened assertions.
- Accept test helpers whose own synchronization can wedge or create false evidence.

### Red/green standard

Verify the exact per-finding source identities and overlay contents.

The closeout report says frozen baselines remained untouched and all five register-bound test files remained byte-identical.

Reproduce the most consequential red/green claims where practical, especially:

- HIGH-1.
- MEDIUM-1 Fix C.
- MEDIUM-2 T1/T2/C3.
- MEDIUM-2 T6.
- LOW-1 attribution.

If the exact historical overlay cannot be reconstructed, say so and assess the surviving evidence honestly.

## 6. Full verification

Independently reproduce the available verification battery against the exact submitted closeout tree.

The reported commands include:

```
gofmt -l .
go vet ./...
go test -p 1 ./... -count=1
go test -race -p 1 ./runtime/ ./approvals/ ./effects/ -count=1
bash scripts/checks.sh
```

Also run the relevant targeted Class A concurrency, crash-recovery, MEDIUM-1, HIGH-1, and audit-attribution tests.

Expected environment:

- Go 1.27.3.
- DBOS Go SDK v1.9.0.
- Local PostgreSQL 19.0.
- `GOWORK=off`.
- Serial shared-database package execution with `-p 1`.

Use disposable databases only.

Record exact commands, actual results, full digests, and any environment substitutions.

Do not claim a full-suite verification if only targeted packages ran.

## 7. Review boundaries

This is an independent audit, not a new implementation phase.

Do not:

- Modify production code.
- Modify tests.
- Modify migrations.
- Rewrite historical review reports.
- Change the accepted WS-A baseline.
- Start Phase 2.
- Implement `authorizeAttempt`.
- Introduce new effects or provider behavior.
- Implement X-12 audit reconciliation.
- Implement X-10 quarantine semantics.
- Introduce Class B or C work.
- Implement `lead-followup`.
- Approve its ADR.
- Change CI or worker startup unrelated to a verified defect.
- Use Neon or OpenCode.
- Commit, push, rebase, reset, amend, or create branches.

Create only the new review report inside the repository. Keep all scratch files, overlays, and test artifacts outside the submitted tree.

If a genuine architectural conflict is discovered, identify it and stop the affected review area. Do not redesign around it.

## 8. Finding classification

Use these severities:

- **CRITICAL:** unauthorized consequential execution, cross-tenant violation, irreversible data corruption, or fundamental failure of the framework's security boundary.
- **HIGH:** a required lifecycle, execution-cardinality, atomicity, or recovery invariant is violated; a race allows duplicate committed business work; a required release proof is invalid; or a deadlock/ownership defect can materially compromise correctness.
- **MEDIUM:** a meaningful correctness, reliability, authorization, or auditability defect remains with bounded impact.
- **LOW:** limited documentation, process, or test-quality issue.
- **INFORMATIONAL:** accepted limitation, future constraint, or clarification.

Do not downgrade an execution or atomicity defect simply because the test suite is green.

Do not inflate accepted residual limitations into blockers without demonstrating why their accepted safeguards are inadequate.

Every finding must contain:

| Field               | Required detail                                                                      |
| ------------------- | ------------------------------------------------------------------------------------ |
| Finding ID          | Stable identifier, e.g. P21-001                                                      |
| Severity            | CRITICAL / HIGH / MEDIUM / LOW / INFORMATIONAL                                       |
| Contract reference  | Exact section or invariant                                                           |
| Source location     | File, function, line                                                                 |
| Defect class        | Execution, atomicity, recovery, identity, authorization, audit, lock order, evidence |
| Root cause          | Underlying mechanism                                                                 |
| Trigger             | Exact operation sequence                                                             |
| Failure mode        | Actual incorrect behavior                                                            |
| Impact              | Concrete consequence                                                                 |
| Evidence            | Source, SQL, test, reproduction                                                      |
| Discriminating test | Test that detects the behavior                                                       |
| Required correction | Smallest durable correction                                                          |

Distinguish confirmed defects from unverified hypotheses.

## 9. Required report

Create:

`docs/adv_review_prompt21.md`

Use the following structure.

### A. Executive verdict

Choose exactly one:

- **CLEAR:** No CRITICAL or HIGH finding remains; Phase 1 is an independently verified foundation for Phase 2.
- **NOT CLEAR:** A CRITICAL or HIGH finding remains, or a foundational guarantee is contradicted.
- **INCONCLUSIVE:** The exact baseline or essential verification could not be established.

A CLEAR verdict is limited to Phase 1 and its compatibility with the approved Phase 2 design constraints. It does not approve Phase 2 implementation itself.

### B. Baseline and provenance

Record full source identities, manifest verification, worktree deltas, register pre-image, exact environment, and reproduced commands.

### C. MEDIUM-2 Class A execution review

Include:

- Claim and lock semantics.
- Transaction boundary.
- Savepoint/error behavior.
- Crash/recovery proof.
- At-most-one guarantee scope.
- Handler side-effect boundary.
- Interaction with the future Phase 2 authorization transaction.

Include a lock-order and transaction-interaction graph.

### D. MEDIUM-1 Fix C and run-ID ownership review

Show each branch and its actual predicate, reload behavior, submission result, and test evidence.

Explicitly disposition the source-correct-but-unproven loss-path branch.

### E. HIGH-1 race proof review

Document both winner scenarios, the actual lock-gate synchronization, queue assumptions, red/green behavior, and persisted audit evidence.

### F. LOW-1 audit attribution review

Explain what the aggregate 4-vs-3 proof establishes and what it cannot establish.

### G. MEDIUM-3 governing-document review

Document the distinction between current governing instructions and preserved historical materials.

### H. Remaining dispositions

Verify LOW-2, LOW-3, and INFO-1 through INFO-7 without reopening accepted scope.

### I. Test-discrimination and verification matrix

For each major claim:

| Claim | Red result | Green result | Independent execution | What is proven | Remaining uncertainty |
| ----- | ---------- | ------------ | --------------------- | -------------- | --------------------- |

Separate:

- Verified red/green.
- Source-correct but unproven.
- Characterization.
- Guard-only.
- Accepted residuals.
- Deferred work.

### J. Residual risks and Phase 2 compatibility

Separate:

1. Confirmed defects.
2. Unverified claims.
3. Accepted residuals.
4. Deferred items.
5. Explicit Phase 2 design constraints.
6. Genuine incompatibilities, if any.

### K. Final recommendation

State whether Phase 1 is independently cleared to serve as the foundation for Phase 2.

If NOT CLEAR or INCONCLUSIVE, list only the minimum required closeout work. Avoid restarting broad remediation.

## 10. Review philosophy

The objective is to determine whether the framework is structurally defensible, not whether the implementation is perfect.

Challenge the most consequential guarantees first.

In particular:

- Can two processes execute the same Class A handler concurrently?
- Can handler database writes commit without the invocation completion?
- Can a failed handler leave partial database writes committed?
- Can recovery repeat already-committed business work?
- Can a terminal invocation acquire or lose canonical workflow identity?
- Can settlement and completion produce inconsistent outcomes?
- Can the new execution transaction introduce a lock-order cycle with Phase 2 authorization?
- Can tests pass without actually exercising the claimed boundary?

Be skeptical but fair.

Do not demand a new design merely because a future improvement is conceivable.

Accept bounded residuals when their consequences are genuinely understood and do not violate the governing contract.

## 11. Phase 2 release gate

Phase 2 is released for owner consideration only if:

1. The final Prompt 20 baseline and evidence are independently established.
2. MEDIUM-2 single Class A execution and crash-recovery guarantees are verified.
3. The future Phase 2 transaction can compose with the Class A claim transaction without a hidden lock-order or atomicity conflict.
4. MEDIUM-1 run-identity protections are adequate.
5. Both HIGH-1 race scenarios are genuinely discriminating and correctly synchronized.
6. LOW-1 audit attribution is sufficiently trustworthy.
7. Governing lock-order instructions are unambiguous.
8. No CRITICAL or HIGH finding remains.
9. Required verification is independently reproduced or its limitations are explicitly accepted.
10. All remaining risks and deferred items are honestly classified.

A passing full suite is necessary evidence, not sufficient acceptance.

A CLEAR review recommends Phase 2 release but does not automatically authorize implementation. The owner must explicitly release Phase 2.

**Final instruction: determine whether Prompt 20 has established a sufficiently reliable lifecycle and Class A execution foundation for effect authorization. Do not pursue perfection, do not reopen accepted architecture, and do not let a green test suite substitute for a genuine proof.**

**My recommendation:** keep Phase 2 blocked until this review specifically clears the Class A transaction's interaction with future `authorizeAttempt`. That is the highest-value final architectural check; the other items are primarily verification of the closeout's reported evidence.