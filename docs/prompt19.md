Prompt 19 should independently review the completed Phase 1 before Phase 2 begins. The most important challenge is whether the seven lifecycle mutations are genuinely safe under competing transactions—not merely whether the new tests pass.

# Prompt 19 — Independent Adversarial Review of WS-B Phase 1

## M-01 Lifecycle Integrity and Contract Corrections

**Repository:** `github.com/PithomLabs/miniapp`\
**Review scope:** Prompt 17, WS-B Phase 1 only\
**Review mode:** Independent, adversarial, read-only\
**Next implementation phase:** Phase 2 — Effect-Authorization Transaction\
**Phase 2 status:** BLOCKED pending independent review and owner disposition

---

## 1. Role and mission

Act as an independent principal engineer conducting an adversarial review of the completed WS-B Phase 1 implementation.

Your task is to determine whether the implementation actually closes M-01 and whether the associated contract, evidence, and test-harness corrections are accurate and complete.

Do not act as the implementation author defending the work. Treat the submitted completion report and green test results as claims requiring independent verification.

The central question is:

**Can any competing lifecycle operation overwrite a valid terminal state, produce an incorrect final status, or create duplicate or misleading audit evidence despite the new predicated transitions?**

The second question is whether the implementation and normative documentation now express the actual lock-order, identity, migration, and fault-injection behavior accurately.

Do not begin Phase 2. Do not implement the effect-authorization transaction.

Do not fix findings during the review.

The objective is to determine whether Phase 1 is an independently trustworthy foundation for Phase 2.

## 2. Governing documents and source of truth

Read the actual repository versions of:

1. `docs/plan4.2.md` — immutable governing architecture.
2. `docs/prompt15.2.md` — governing implementation instructions.
3. `docs/phase1/invocation-lifecycle-contract.md` — normative lifecycle contract.
4. `docs/prompt15.2_p2.md` — accepted P2 decisions and constraints.
5. `docs/prompt17.md` — WS-B work order.
6. `docs/adv_review_prompt16.md` — independent WS-A review and owner disposition.
7. `docs/evidence/README.md` — evidence register.
8. Relevant manifests, pre-images, test reports, and Phase 1 implementation artifacts.

Read the actual source and tests, not summaries or report descriptions.

Preserve the distinction between:

- Normative contract requirements.
- Owner-approved decisions.
- Implementation behavior.
- Test evidence.
- Known residual limitations.
- Deferred Phase 2 responsibilities.

Do not silently repair a contradiction between documentation and code by interpreting one as non-binding.

If a requirement is ambiguous, identify the exact passages and report the conflict.

### Governing architectural principle

**The framework owns reliability; miniapps own business logic.**

The framework must enforce lifecycle consistency through shared mechanisms. Individual miniapps must not have to independently prevent stale status overwrites, duplicate cancellation audits, or inconsistent recovery outcomes.

## 3. Baseline verification — mandatory before review

The submitted Phase 1 evidence identifies these abbreviated states:

| State                           | Reported digest | Purpose                                                 |
| ------------------------------- | --------------- | ------------------------------------------------------- |
| WS-A acceptance file set        | `e20a4689…`     | Immutable historical foundation                         |
| Phase 1 red tree                | `c9c08fe5…`     | Pre-remediation behavior with regression tests overlaid |
| Phase 1 implementation worktree | `5afb6e46…`     | Submitted green implementation                          |
| M-01 regression test            | `222f5ff5…`     | Identical red/green test source                         |

These values are identifiers, not sufficient evidence by themselves.

Before reviewing implementation:

1. Recompute the complete WS-A acceptance digest using the authoritative digest procedure.
2. Verify the Phase 1 red-tree digest and its relationship to the accepted baseline.
3. Recompute the complete Phase 1 green worktree digest.
4. Verify the complete SHA-256 of `runtime/m01_test.go`.
5. Verify that the exact same test bytes were used in the red and green overlays.
6. Verify that the red overlay changes only the intended test file and contains the expected pre-remediation implementation.
7. Verify the green tree contains the submitted implementation and no unreported changes.
8. Verify the evidence-register pre-image and appended rows.
9. Record the current Git revision, branch, working-tree status, and complete file inventory.

The reviewer must identify every source or test difference between the accepted baseline, the red overlay, and the submitted green tree.

Do not infer full hashes from abbreviated values.

Do not substitute a nearby revision or silently rebuild a different baseline.

If any discrepancy is found, stop the affected verification and report it.

### Environment

The intended verification environment is:

- Go 1.27.3.
- DBOS Go SDK v1.9.0.
- Local PostgreSQL 19.0.
- Serial package execution using `go test -p 1`.
- Disposable databases only.

No Neon, OpenCode, real provider writes, or customer credentials.

Record any environmental substitution explicitly. An unexecuted test is not a passing test.

### Evidence provenance

The review must distinguish:

- Evidence independently reproduced.
- Evidence verified by reading source or test code.
- Evidence supplied by the author but not independently reproduced.
- Claims that remain unknown.

Do not treat a green report as proof of a correctly established baseline.

## 4. Review scope

Review only the completed Phase 1 work:

### M-01

Seven lifecycle mutation sites, their supporting store operations, audit behavior, and regression tests.

### L-01

The `claim-response` fault operation, dedicated test, replay behavior, and production-path isolation.

### L-02

Migration delimiter and escape compatibility wording.

### L-03

Legacy identity-row behavior after the encoding change.

### L-04

Immutable manifest-time evidence-register copies and provenance.

### O-05

Serial shared-database test methodology documentation.

### Advisory-lock contract correction

The corrected §4 lock-order rule and its no-cycle proof.

Do not expand into Phase 2 effect authorization, even if adjacent code or tests already contain scaffolding for it.

If an adjacent issue is discovered, record it separately as a deferred observation.

## 5. M-01 — principal adversarial review

### 5.1 Expected implementation

The submitted report claims that all seven mutation sites now use predicated transitions and loser-reload-replay behavior.

The reported implementation locations are:

- `runtime/service.go`
  - `executeSync` / `completeSync`
  - `settlePollTerminal`
- `runtime/lifecycle.go`
  - `Status` synchronization
  - `Cancel`
- `runtime/store.go`
  - Four transition operations.

The previous unpredicated `store.update` abstraction is reported fully eliminated:

- Zero definitions.
- Zero production callers.

Verify every statement independently.

Search the complete production tree. Do not limit inspection to the named files.

Determine whether any equivalent unconditional write remains under another name or through a lower-level query.

### 5.2 Expected transition semantics

The submitted design states:

| Operation                         | Required source state   | Losing transition behavior                                  |
| --------------------------------- | ----------------------- | ----------------------------------------------------------- |
| `executeSync` terminal completion | `pending`/`running`/`waiting_approval` (implementation-correct: undispatched pending and granted waiting Class A records must complete) | Reload and replay the recorded terminal; discard stale result |
| `settlePollTerminal`              | `waiting_approval` only | Re-observe approval and invocation; answer truthfully       |
| `Status` synchronization          | Live states only        | Return the current persisted record                         |
| `Cancel`                          | Live states only        | Already-terminal state becomes the documented no-op         |

These are expectations to verify, not assumptions to accept.

For every mutation, establish:

1. The exact SQL predicate.
2. The transaction boundary.
3. The rows and locks involved.
4. Whether the update is atomic with its associated audit.
5. What happens when zero rows are affected.
6. Whether the current record is reloaded after losing.
7. Whether the returned API result reflects the authoritative persisted state.
8. Whether a stale result can escape to the caller.
9. Whether an audit event can be emitted for a transition that did not win.
10. Whether repeated requests remain idempotent.

The store-level operations must enforce the invariant. Do not accept an implementation in which service-layer checks are the only protection.

### 5.3 Challenge loser-reload-replay behavior

The new design relies on losing operations reloading persisted state and returning the truthful outcome.

Challenge this carefully.

Investigate:

- Whether the reload actually obtains fresh committed state.
- Whether stale transaction snapshots can survive the losing update.
- Whether a caller can accidentally return the old in-memory object.
- Whether retries can invoke the handler again.
- Whether a stale successful handler result can overwrite a cancellation.
- Whether a terminal state can be changed by another terminal operation.
- Whether a losing operation can incorrectly emit a success audit.
- Whether concurrent requests can each believe they won.

Verify that loser handling cannot become an unbounded replay loop or conceal a persistent state conflict.

### 5.4 `executeSync` and `completeSync`

Inspect both success and failure completion paths.

The key invariant:

**Once cancellation wins, a stale synchronous handler result cannot change the invocation into succeeded or failed.**

Conversely, if completion wins first, a later cancellation must not overwrite the completed outcome.

Inspect whether:

- Only the permitted `running` state can transition.
- The handler result is discarded when cancellation already committed.
- The caller receives the correct terminal cancellation result.
- Successful and failed completion produce the correct audit only when their transition wins.
- Completion and cancellation preserve the required lock order.
- Neither path holds database locks across external operations.

Do not accept a test that only proves the final database state while ignoring an incorrect response to the losing caller.

### 5.5 `settlePollTerminal`

Review approval-denied and approval-expired settlement.

The key invariant:

**An approval-poll result based on stale information must not overwrite a concurrent cancellation or another authoritative terminal transition.**

Check:

- Only `waiting_approval` can be settled by this path.
- Approval state is re-observed when the update loses.
- The invocation state is re-observed.
- The response matches the actual persisted terminal or pending state.
- No stale approval observation creates an incorrect failure response.
- No duplicate or false audit occurs.

Examine error and retry behavior when approval and invocation reads disagree or a concurrent transition commits between reads.

### 5.6 `Status` synchronization

Challenge the assumption that synchronization with executor truth is always harmless.

The key invariant:

**Status may synchronize an observed executor result only while the invocation remains eligible for that transition.**

Inspect:

- The live-state predicate.
- The update's affected-row handling.
- The reload path after losing.
- Whether a stale executor result can overwrite cancellation.
- Whether terminal-state immutability is maintained.
- Whether the status-sync audit is emitted only if synchronization wins.

The submitted report specifically states that a correctness fix was made so the Status-sync settlement audit runs only on the winning path.

Verify the exact control flow.

A losing synchronization must not emit a success audit merely because it observed a successful executor result.

### 5.7 `Cancel`

Inspect cancellation as an independently competing lifecycle operation.

The key invariants:

- Cancellation cannot overwrite a previously committed terminal state.
- Repeated cancellation cannot produce duplicate cancellation audits.
- A losing cancellation returns the documented terminal-already no-op.
- Cancellation cannot accidentally trigger another handler or workflow execution.
- The invocation response reflects the authoritative current state.

Inspect all affected-row and reload behavior.

Pay particular attention to transaction boundaries where a cancellation transition and its audit are committed.

### 5.8 Store-level transition discipline

The four reported `runtime/store.go` operations must be inspected directly.

Determine whether they:

- Encode the required source-state predicates.
- Enforce tenant and identity scope.
- Check affected-row counts correctly.
- Return useful results to callers.
- Make illegal transitions impossible at the shared store boundary.
- Preserve audit and lifecycle invariants.
- Avoid allowing service callers to bypass the predicated mechanism.

Review the green-only `runtime/store_test.go` tests separately.

The submitted report says those tests pin:

- Tenant predicates.
- From-state matrix.
- Terminal immutability.
- CR-05.

Do not assume these tests are red/green discriminators. Assess them according to what they actually prove.

## 6. M-01 red/green test discrimination

### 6.1 Test identity

The reported regression file is:

`runtime/m01_test.go`

Reported hash:

`222f5ff5…`

Verify the complete hash against the exact red and green source bytes.

Verify:

- Identical source.
- Same test names and assertions.
- Same test helper implementation, except where the overlay methodology explicitly documents a required shared harness file.
- No environment-specific branch that weakens red assertions.
- No source substitution that changes the failure boundary.

### 6.2 Red behavior

The submitted report says the red implementation fails because:

1. The two race tests produce incoherent state/audit pairs.
2. The synchronous race discriminates under both winner orders across runs.
3. The double-cancel test produces duplicate cancellation audits.
4. Three sequential tests pass on both sides as documented characterizations.

Verify all of these claims.

A red failure must be caused by the intended old behavior, not an unrelated startup failure, database issue, test timeout, or broken harness.

### 6.3 Deterministic concurrency proof

The original WS-B work order requires deterministic lock-gate tests.

Independently examine the synchronization helper and every race test.

For each required race:

- Identify the exact row and transaction boundary.
- Verify that the holder owns the intended lock.
- Verify that the racer reaches the intended SQL statement.
- Verify that the racer has actually observed the expected pre-transition state before being released.
- Establish blocking through the intended lock chain.
- Where tuple-lock queues require it, examine transitive `pg_blocking_pids` relationships rather than only direct blockers.
- Verify statement shape and relevant lock identity.
- Exclude unrelated lock waiters and statements from satisfying the gate.
- Release the holder only after the required conditions are established.
- Ensure cleanup releases the holder and joins all racers on every exit path.

Do not accept sleeps, repeated scheduling, or probabilistic overlap as a substitute for the required proof.

A test that happens to exercise both winner orders across multiple runs is not automatically deterministic.

### 6.4 The five required race cases

Reconstruct the five required race cases from Prompt 17, the Phase 1 implementation, and the tests.

Produce a mapping with:

| Race | Competing operations | Required winning/losing outcomes | Gate proof | Red result | Green result |
| ---- | -------------------- | -------------------------------- | ---------- | ---------- | ------------ |

Do not invent missing cases. If the submitted test inventory does not cover all five required cases, report the gap.

Examine both sides of each race where both orders are permitted.

### 6.5 Sequential characterization tests

The report states that three sequential tests pass on both red and green.

Verify that they are honestly documented as characterization or guard tests, not as regression discriminators.

Explain:

- What behavior they establish.
- Why the red implementation is expected to pass them.
- Whether they add meaningful coverage.
- Whether their names or report wording could mislead a later reviewer.

### 6.6 Audit correctness

Independently reconstruct the audit expectations for every race.

Assert that:

- A winning transition emits its appropriate audit.
- A losing transition does not emit an audit claiming it won.
- Duplicate cancellation requests do not duplicate the cancellation audit.
- Tenant and principal attribution are correct.
- Audit ordering and transaction atomicity match the contract.
- Failed or rolled-back transitions do not leave misleading committed audit evidence.

Do not treat the presence of an audit row as sufficient. Verify that its semantics match the transition actually committed.

## 7. Advisory-lock contract correction

The submitted Phase 1 work corrects the advisory-lock wording in §4.

The corrected rule is:

**Advisory locks are acquired outermost-first, before acquiring transactional PostgreSQL locks. They must never be acquired while already holding PostgreSQL locks.**

The earlier wording that described advisory locking as a leaf operation was backwards for the actual Class A implementation.

The report states that `executor/classa/classa.go` acquires the advisory lock first and holds it across handler `Perform` calls.

Verify this from source, not comments.

### Required investigation

1. Locate every production advisory-lock acquisition.
2. Trace the complete caller chain for each acquisition.
3. Identify all PostgreSQL row, table, and transaction locks acquired afterward.
4. Identify whether any path acquires an advisory lock while already holding a transactional PostgreSQL lock.
5. Trace Class A's advisory-lock lifetime across `Perform`.
6. Inspect nested framework operations invoked by handlers.
7. Inspect other execution classes and shared service operations that can participate in the same lock graph.
8. Construct a lock-order graph and establish that it has no cycle.
9. Account for errors, retries, cancellation, and transaction boundaries.
10. Verify release semantics and lock ownership.

The proof must address the actual nesting, not merely show that one function calls `pg_advisory_lock` before one SQL statement.

If any path can acquire locks in the reverse order, identify the concrete cycle and its possible deadlock.

### Documentation verification

Confirm that §4 accurately expresses the implementation and that the change is recorded through the corrections-never-silent process.

The historical §13 drift table contains near-duplicate WS-B rows for L-02, L-03, and advisory locking.

The author reports that these rows were left intact to preserve the historical record.

Determine whether:

- Their content is accurate.
- Their duplication creates ambiguity about the normative rule.
- Their historical provenance is preserved.
- A future reader can identify which text is authoritative.

Do not demand historical deletion merely to improve presentation. If normalization is warranted, report it as a future documentation action rather than editing the source.

## 8. L-01 — claim-response fault operation

The Prompt 16 review identified a specified-but-unimplemented `claim-response` fault operation.

The submitted Phase 1 report says the fault operation and its test have now been implemented.

Verify:

1. The operation is defined in the fault-injection inventory.
2. Its name and firing point match the contract.
3. The hook fires after the claim is persisted but before the response is returned.
4. It supports the required one-shot behavior.
5. Replay returns the original canonical handle.
6. Replay does not create a second workflow.
7. The persisted run identity remains stable.
8. Audit evidence remains coherent.
9. The dedicated `TestFaultClaimResponse` actually exercises the lost-response condition.
10. The test fails or otherwise distinguishes the missing-hook behavior on the appropriate baseline, where the historical source permits such a comparison.

### Production-path isolation

Verify:

- The hook can be invoked only through the approved test fault-injection mechanism.
- No production configuration or assignment path enables it.
- No environment variable, manifest setting, public API, or miniapp input can activate it.
- Nil-injector behavior remains safe.
- The hook does not change production behavior when fault injection is absent.

Do not accept a test that merely calls the hook directly without proving the real claim/replay path.

## 9. L-02 and L-03 — identity and migration documentation

These are contract-correction items, not authorization to redesign migration behavior.

### L-02 — delimiter compatibility

The accepted review identified that the migration preflight checks duplicate nonempty `run_id` values but does not perform the delimiter/escape collision scans previously described in §12.

The submitted correction is expected to state the actual behavior:

- `ScopeID` uses an injective encoding.
- Delimiter-bearing legacy IDs remain compatible with the documented encoding behavior.
- Migration preflight checks the conditions it actually implements.
- It does not claim to scan for delimiter or escape collisions if no such scan exists.

Verify the encoding's injectivity and its supported input domain.

Inspect the implementation and tests, not just the prose.

### L-03 — legacy identity-row behavior

The accepted review identified that a legacy raw multi-colon operation ID can fail to match its old primary key when retried through the new escaped encoding.

The documented residual behavior is that such a legacy row can remain orphaned rather than transparently matching the new identity representation.

Verify:

- The actual legacy/new-key lookup behavior.
- Whether old rows can be re-attempted in the current lifecycle.
- Whether the no-production-data premise is accurately documented and supported by the available evidence.
- Whether the new documentation clearly states the compatibility boundary.
- Whether future encoding changes with production data present require dual-read or backfill.

Do not silently claim automatic migration or backward-compatible lookup if the code does not provide it.

### Corrections-never-silent discipline

Verify that both corrections are recorded in §13 with:

- What was wrong.
- Why the correction is necessary.
- The actual implementation behavior.
- Relevant supporting tests.
- Any accepted residual limitation.

The historical duplicate rows must not obscure the corrected rule.

## 10. L-04 — evidence-register immutability

The accepted Prompt 16 review identified an in-place modification of the evidence register whose original bytes could not be recovered.

The Phase 1 report says the new process retains a manifest and register pre-image.

Verify:

- The pre-image exists and is immutable for the submitted review cycle.
- The pre-image hash is correct.
- The manifest records the correct file set.
- The digest command, current directory, path form, and exclusions are recorded.
- New evidence rows can be tied to the exact register version used.
- The final worktree digest is reproducible without self-reference ambiguity.
- No historical artifact was silently replaced.

A register that contains its own digest cannot straightforwardly include that final digest in its own hashed bytes. Verify that the submitted method avoids this self-reference problem explicitly.

Do not attempt to reconstruct missing historical bytes where no pre-image exists. Distinguish the historical limitation from the new control.

## 11. O-05 — serial shared-database testing

The accepted review found that separate Go test package processes can interfere when they share one database.

The Phase 1 work adds a one-line `testkit` documentation note stating the serial-package requirement.

Verify:

- The note accurately describes the shared-database hazard.
- The recommended `-p 1` behavior matches the test harness.
- It does not overstate the requirement for packages using isolated databases.
- It gives a future developer enough information to avoid the known lock contention.
- The actual Phase 1 verification commands followed the documented methodology.

Do not require a test-harness redesign merely because a misuse scenario exists. However, if the documentation misstates the isolation assumptions, report it.

## 12. Full regression and independent verification

Independently run the available verification battery against the exact Phase 1 worktree.

At minimum:

```
gofmt -l .
go vet ./...
go test -p 1 ./...
```

Run the targeted M-01 tests separately, including repeated execution and `-race` where applicable.

Run the lock-gate concurrency tests repeatedly enough to expose synchronization defects, but do not mistake repetition for determinism.

Run:

- M-01 red overlay.
- M-01 green tests.
- Store-level transition tests.
- L-01 fault-injection tests.
- Relevant approval and runtime suites.
- Full serial suite.
- Relevant migration and identity tests.
- Existing `scripts/checks.sh` gates, where applicable.

The exact environment is subject to the Phase 1 baseline. Use local PostgreSQL 19.0 and Go 1.27.3 where available.

Record actual results and commands.

If a test cannot run:

- Identify the test.
- State the exact environmental blocker.
- Mark it NOT RUN or BLOCKED.
- Do not claim coverage based on a related test.

Do not modify source code or tests to make this review run.

## 13. Finding classification

Use these severities:

- **CRITICAL:** unauthorized execution, cross-tenant access, irreversible state corruption, or fundamental compromise of the framework boundary.
- **HIGH:** a lifecycle or concurrency invariant remains broken; a terminal state can be corrupted; a transition can falsely claim success; duplicate or misleading committed audit evidence remains possible; or a required correctness proof is invalid.
- **MEDIUM:** a bounded correctness, auditability, contract, or reliability defect remains without demonstrated compromise of a critical invariant.
- **LOW:** limited documentation, process, test, or maintainability defect.
- **INFORMATIONAL:** accepted limitation, clarification, or improvement opportunity.

Do not downgrade a terminal-state integrity defect solely because its race window is narrow.

Do not inflate a wording issue into a code defect.

Every finding must include:

| Field               | Required detail                                                                              |
| ------------------- | -------------------------------------------------------------------------------------------- |
| Finding ID          | Stable identifier, e.g. `P19-001`                                                            |
| Severity            | CRITICAL / HIGH / MEDIUM / LOW / INFORMATIONAL                                               |
| Contract reference  | Exact section or invariant                                                                   |
| Source location     | File, function, line range                                                                   |
| Defect class        | State transition, audit, concurrency, lock order, fault injection, documentation, provenance |
| Root cause          | Underlying mechanism                                                                         |
| Trigger             | Exact operation sequence                                                                     |
| Failure mode        | Actual incorrect behavior                                                                    |
| Impact              | Concrete consequence                                                                         |
| Evidence            | Source, SQL, test output, or reproduction                                                    |
| Discriminating test | Test that distinguishes defective from correct behavior                                      |
| Required correction | Durable fix at the responsible mechanism                                                     |

Separate established defects from hypotheses requiring more evidence.

## 14. Required report

Create one standalone report:

`docs/adv_review_prompt19.md`

Do not modify implementation source, tests, migrations, the lifecycle contract, or historical evidence artifacts during the review.

The report must contain the following sections.

### A. Executive verdict

Choose exactly one:

- **CLEAR:** No CRITICAL or HIGH finding remains; all required M-01 invariants and Phase 1 exit conditions are independently verified.
- **NOT CLEAR:** A CRITICAL or HIGH finding remains, or a required M-01 guarantee is not established.
- **INCONCLUSIVE:** The exact baseline or essential verification could not be established.

A CLEAR verdict is bounded to this Phase 1 review.

### B. Baseline and provenance

Record:

- Full acceptance baseline digest.
- Full Phase 1 red-tree digest.
- Full Phase 1 green-tree digest.
- Full M-01 test-file digest.
- Git revision and working-tree status.
- Register pre-image and manifest identity.
- Environment and toolchain versions.
- Exact commands and independent reproductions.

### C. M-01 transition matrix

For every reported mutation, document:

| Mutation | SQL predicate | Transaction boundary | Losing behavior | Audit behavior | Test | Verified |
| -------- | ------------- | -------------------- | --------------- | -------------- | ---- | -------- |

Explicitly account for all seven mutation sites and all four store operations.

### D. Five-race assessment

Provide the reconstructed five-case race matrix, including the lock-gate proof, red behavior, green behavior, winner/loser outcomes, and remaining gaps.

Distinguish deterministic synchronization from repeated probabilistic execution.

### E. Audit integrity

Report the exact audit guarantees for completion, approval settlement, status synchronization, and cancellation.

Include the Status-sync audit-on-win correction and the double-cancel proof.

### F. L-01 through L-04 and O-05 assessment

For each item state:

- Original finding.
- Required correction.
- Actual implementation/documentation.
- Verification evidence.
- Residual limitation.
- Final disposition.

### G. Advisory-lock proof

Include:

- Lock acquisition graph.
- Actual Class A nesting.
- Every relevant acquisition site.
- No-cycle argument.
- Contract wording assessment.
- Remaining assumptions.

### H. Test-discrimination matrix

At minimum:

| Claim | Red behavior | Green behavior | Independent execution | What is actually proven | Gap |
| ----- | ------------ | -------------- | --------------------- | ----------------------- | --- |

Do not claim that green-only store tests are red/green discriminators.

Do not call a sequential characterization test a concurrency regression test.

### I. Residual risks and deferred scope

Separate:

1. Confirmed defects.
2. Unverified claims.
3. Accepted residual limitations.
4. Phase 2 responsibilities.
5. Out-of-scope observations.

Do not treat Phase 2 scaffolding as completed effect authorization.

### J. Final recommendation

State whether Phase 1 is independently accepted as the foundation for Phase 2.

List all findings that require correction or explicit owner disposition before Phase 2 can start.

## 15. Review boundaries

This is a read-only review.

Do not:

- Modify implementation source.
- Modify tests.
- Modify migrations.
- Rewrite historical contract decisions.
- Rebuild or replace the accepted baseline.
- Start Phase 2.
- Implement `authorizeAttempt`.
- Introduce grant mutexes or generation fencing.
- Implement provider behavior.
- Add an effects reconciler.
- Introduce Class B or C work.
- Implement the `lead-followup` miniapp.
- Approve its ADR.
- Change CI or worker startup behavior.
- Use Neon or OpenCode.
- Commit, push, reset, rebase, amend, or create branches.

One new review report is permitted. Keep all other review-generated artifacts outside the submitted worktree.

If a defect is found, describe the root cause and required test. Do not patch it.

## 16. Permanent review discipline

For every finding:

1. Determine whether it is a symptom, distinct defect, or member of a larger failure class.
2. Identify the root cause before proposing a correction.
3. Map it to an enforceable contract invariant.
4. Identify the shared mechanism responsible.
5. Specify a test that exercises the actual failure boundary.
6. Distinguish a regression discriminator from a guard or characterization test.
7. Verify that the test cannot pass while the defective behavior remains.
8. Require a durable correction rather than a local workaround.
9. Bind the conclusion to exact source and evidence identities.
10. Preserve accepted limitations and owner decisions without silently altering them.

Do not judge the implementation by whether the original finding identifiers have disappeared.

Judge whether the underlying failure mechanisms have been eliminated or reliably detected.

## 17. Phase 2 release gate

Phase 2 remains blocked until:

- M-01 is independently verified closed.
- All seven mutation sites have correct predicated transitions.
- All five required concurrency cases are adequately tested.
- Losing operations reload and report authoritative state correctly.
- Audit-on-win behavior is proven.
- Double-cancel cannot produce duplicate cancellation audits.
- Advisory-lock order and no-cycle reasoning are sound.
- L-01 through L-04 and O-05 have truthful, reproducible dispositions.
- No CRITICAL or HIGH finding remains.
- The full serial verification is independently reproduced.
- The final report clearly distinguishes proven, unverified, and deferred behavior.

A CLEAR report supports Phase 2 release, subject to the owner's explicit decision.

A NOT CLEAR report blocks Phase 2 until the findings are resolved and re-reviewed.

An INCONCLUSIVE report does not release the gate.

**Do not begin Phase 2 merely because Phase 1's author declared completion or because the full serial test suite is green.**

The purpose of Prompt 19 is to establish independently that lifecycle state and audit integrity are now protected by the framework's actual database transition mechanisms.

## **My review emphasis**

I would pay particular attention to three things when evaluating the result:

1. **The five race proofs:** the report says the sync race discriminated under both winner orders across runs. The reviewer must establish whether the gates actually force and prove those orders or whether repetition merely makes both outcomes likely.
2. **Audit atomicity:** the Status-sync audit-on-win correction and double-cancel behavior deserve direct transaction-level examination. A correct final status with incorrect audit evidence is not closure.
3. **The advisory-lock graph:** because Class A holds the advisory lock across handler `Perform` calls, the reviewer must inspect nested database operations and prove the absence of reverse acquisition paths.

I recommend keeping Phase 2 blocked until Prompt 19 returns CLEAR and the resulting findings have been explicitly dispositioned.