# Prompt29 — Independent Gate 2 Adversarial Review of Prompt28 Stage 3A

## Mission

Act as an independent principal engineer and adversarial architecture reviewer of Miniapp.

Your assignment is to independently review Prompt28 Stage 3A implementation and evidence, determine whether the Gate 2 exit criteria have actually been satisfied, and issue a defensible release verdict.

You are not the implementation author. Do not assume that the implementation report is correct merely because it says the tests passed. Verify the underlying source, tests, execution evidence, architectural contracts, and residuals.

**This is a review-only work order. Do not implement fixes.**

The objective is to determine whether Miniapp can formally close Stage 3A and proceed to the next foundational architecture workstreams.

## 1. Governing architectural decisions

The following decisions are binding review criteria:

1. The framework owns execution reliability; miniapps own business logic.
2. The existing Classes A–D retain their distinct execution and recovery semantics.
3. Authorization must be enforced at the appropriate business-operation and consequential external-effect boundaries.
4. Authorization must not acquire advisory locks or violate the established global lock order.
5. External side effects must not be sent without the required authorization and durable effect-state transition.
6. Ambiguous provider outcomes must be quarantined and resolved using evidence. Never blindly resend an operation whose outcome is unknown.
7. Class A recovery must preserve transaction rollback, stable BusinessOpID semantics, bounded recovery, and absence of held locks while waiting.
8. PostgreSQL 19 is mandatory for the relevant local, integration, concurrency, crash-recovery, and DBOS verification.
9. M5 sequence-verification infrastructure remains deferred. Its documented fail-closed behavior and activation triggers must remain intact.
10. Class B and C adapter conformance is not part of Prompt28 and must not be claimed as complete.

Two additional architectural priorities are now locked:

**Security and Observability are co-equal, critical foundational capabilities.**

They are to be developed as coordinated contracts after this Gate 2 review. Both must be established before accelerated Class B/C implementation. This review must not begin their implementation or silently expand Prompt28.

## 2. Review inputs

Read and cross-reference the actual repository contents, including:

- `docs/prompt28.md` — Prompt28 governing work order.
- `docs/phase2/prompt28-stage3a-closeout.md` — implementation closeout.
- `docs/phase2/prompt28-settle-operator-decision.md` — owner-blocked authority decision.
- `docs/evidence/p28/README.md` — reviewer evidence package.
- `docs/evidence/README.md` — evidence register.
- `docs/evidence/digests/p28-closeout-2026-11-23.manifest.txt` — evidence manifest.
- The original Prompt21 C1–C7 definitions.
- Prompt22 Phase 2 requirements, particularly its crash matrix and authorization criteria.
- Prompt24, Prompt25, Prompt26, and Prompt27 findings, dispositions, and residuals.
- The actual implementation, tests, migration history, CI configuration, Compose configuration, Taskfile, scripts, and relevant architecture documents.

Locate the governing files if their paths differ. Do not assume that a filename or report reference proves that the corresponding implementation exists.

Use the implementation report as an index into the evidence, not as a substitute for inspecting that evidence.

## 3. Preserve repository integrity

Before reviewing:

1. Record the exact HEAD, branch, and worktree status.
2. Record tracked and untracked changes.
3. Reproduce the claimed source and tree digest where possible.
4. Identify files changed by activity outside Prompt28.
5. Compare the actual changes against the Prompt28 manifest and report.

The closeout report records outside author activity, including roadmap document additions and deletion of `docs/roadmap2.md`. It also identifies previously preserved FAQ activity.

Treat these as potentially legitimate owner-side changes. Do not reset, restore, delete, overwrite, reformat, or attribute them to Prompt28 without evidence and authorization.

No commits, staging, pushes, releases, deployments, migrations, dependency changes, or real provider writes are authorized.

## 4. Independent authorization-matrix review

Inspect every new Prompt28 authorization-matrix leg.

Verify:

- Identity and tenant binding.
- Principal, invocation, operation, and effect ownership.
- Successful and failed terminal-state behavior.
- Pending and waiting-state behavior.
- Pending approval semantics.
- All kill-switch scopes, including tool-declared switches.
- Typed grant allow and deny behavior.
- Grant-swap race behavior.
- Capability reuse and conflict behavior.
- Intent trust-field spoofing protection.
- Persisted trust and approval semantics from the earlier Gate 1 dispositions.

For each refusal case, inspect whether the test genuinely proves:

1. The correct typed refusal.
2. No unauthorized business-state mutation.
3. No provider send.
4. No unauthorized transition to ATTEMPTING.
5. Correct and truthful audit behavior.

Verify that the five-point refusal evidence is asserted by the tests themselves rather than inferred from test names or report summaries.

Check whether each relevant test would fail if the authorization boundary were removed or bypassed. Identify vacuous assertions, shared-fixture assumptions, missing negative controls, and tests that exercise only mocked behavior.

## 5. Concurrency and race review

Independently examine the nine reported race cases and their underlying test implementations.

Pay particular attention to:

- Same-effect worker contention.
- Cross-generation and attempt-epoch fencing.
- Lease turnover.
- Quarantine versus live authorization.
- Restoration versus dispatch in both orders.
- Sequence-verification fail-closed behavior and the M5 deferral.
- Settle-versus-Settle ordering.
- Shared-row contention.
- Provider outcome and audit consistency.

Determine whether the tests force the relevant interleavings through actual synchronization, locks, barriers, process state, and database observations.

Reject timing-based tests that merely sleep and hope a race occurs.

For every race, identify:

- The intended invariant.
- The forced ordering.
- The expected winner and loser.
- The durable database state.
- The audit result.
- Provider-send accounting.
- Attempt and lease epoch behavior.
- Retry or recovery classification.

Verify that the tests cannot pass while the underlying race remains possible.

## 6. Crash-recovery review

Independently verify the six Prompt22 §10.3 crash boundaries.

The review must distinguish genuine process termination from injected errors, returned errors, simulated exceptions, and transaction rollbacks.

For every boundary, verify:

- Where the process terminates.
- Whether termination is genuine.
- Which database transaction has committed.
- Which locks and leases remain.
- Durable effect and invocation state after restart.
- Whether a provider call may already have occurred.
- Whether the subsequent action is safe.
- Whether the implementation can accidentally resend an ambiguous effect.

The report explains that B4/B5 share one stall-kill test because their externally observable boundaries are identical. Verify that equivalence against the source and state evidence. Do not require duplicate tests merely for different labels, but do not accept an unsupported equivalence claim.

Ambiguous outcomes must remain quarantined. No recovery path may silently treat UNKNOWN as proof that a provider did nothing.

## 7. C6 Class A recovery integration

Inspect `TestP28C6ClassARecovery` and all associated implementation code.

Independently verify the reported eleven C6 claims, especially:

- Real Class A handler contention can reach `ErrAttemptInProgress`.
- The losing transaction completely rolls back.
- No partial business writes survive.
- The invocation is not incorrectly terminalized.
- The losing request holds no transaction, database lock, or grant RLock while waiting.
- Recovery is bounded.
- BusinessOpID remains stable across attempts.
- Framework-generated attempt epochs remain authoritative.
- The interrupted/new-key guidance is accurate.
- New-key adoption does not resend a previously completed external effect.

Check that the fixture tests the actual execution path rather than a simplified imitation.

Verify that developer documentation describes the required caller behavior accurately and does not promise indefinite or automatic retries.

## 8. Grant hash, health, and revocation

Inspect the implementation of:

- `Service.GrantConfigHash()`.
- `GET /healthz`.
- `miniapp auth verify-revocation`.
- Startup hash logging.
- Grant hash canonicalization.
- Revocation verification and operator documentation.

Verify:

- Hash order invariance.
- Hash changes when relevant grant configuration changes.
- Replica comparison is accurate.
- Stale and unreachable replicas cannot produce a false success.
- Empty inventory cannot produce a successful verification result.
- Exit codes match the documented result matrix.
- Secrets, tokens, credentials, and sensitive grant material are not exposed.
- The implementation preserves the explicitly documented revocation staleness and restart semantics.

Do not interpret `/healthz` as a comprehensive operational-health endpoint unless its contract actually promises that behavior.

Do not claim immediate global revocation unless the underlying implementation proves it.

## 9. Settle crash safety and operator authority

Review the Settle crash matrix separately from the authenticated operator wiring.

For crash safety, independently examine:

- Precommit process termination.
- Lost-response behavior.
- Idempotent repeat and audit backfill.
- Concurrent backfill serialization.
- Conflicting second decisions.
- Interaction with B8 and the effect/attempt lifecycle.

Reproduce the reported backfill audit race and inspect the migration-free fix. Verify that the red and green tests use identical test bytes and that the behavior changes because of the actual fix.

For authenticated operator wiring, preserve the STOP.

Read `docs/phase2/prompt28-settle-operator-decision.md` and independently inspect the source audit.

Determine whether a trusted, server-derived administrative authority actually exists.

Do not invent a role, assume `tenant-ops` is production-approved, treat tenant ownership as operator authority, or convert a caller-controlled string into trusted identity.

Do not wire `settle` or add a service-level Settle method.

The reviewer is not authorized to choose among owner options A, B, and C. Record this as an owner decision, not an implementation defect.

## 10. Provider bypass and effect-boundary assurance

Inspect the complete provider-send inventory.

Verify the claim that there is one canonical production send site and that every consequential provider path must pass through it.

Review:

- Fresh Perform.
- Takeover and adoption.
- Reconciliation.
- Resume and retry.
- DBOS workflow paths.
- Channel implementations.
- Settle and administrative paths.
- Any other provider-facing code discovered during review.

Inspect the AST guard, its negative control, allowlists, and test coverage.

Determine whether a new provider-send call introduced outside the canonical site would be detected.

Verify that authorization precedes the provider call, that durable ATTEMPTING state precedes the send, and that stale outcomes cannot overwrite a newer attempt epoch.

Verify that UNKNOWN is not blindly resent.

The unknown-capability taxonomy issue and P24-008 activation trigger must remain accurately characterized. Do not convert a fail-closed behavior into a stronger guarantee than the evidence supports.

No real provider calls are authorized.

## 11. Revalidate C1–C7 using their original meanings

Use the original Prompt21 definitions. Do not rely on the old mislabeled table in historical reports.

| Constraint | Required invariant |
|---|---|
| C1 | `authorizeAttempt` acquires no advisory locks. |
| C2 | Authorization follows the global lock order. |
| C3 | Authorization uses its own transaction/pool checkout, never the Class A attempt's `pgx.Tx`. |
| C4 | No new advisory-lock acquisition paths are introduced. |
| C5 | Pending semantics are explicit and correct. |
| C6 | Class A handlers invoking Perform handle `ErrAttemptInProgress` with bounded recovery, stable BusinessOpID, and no indefinite transaction hold. |
| C7 | The O-01 orphan window closes at the effect boundary. |

For every constraint, inspect the source and its discriminating tests. State whether it is source-proven, execution-proven, composition-proven, characterization-only, accepted residual, or blocked/unverified.

Do not equate the presence of an interface or test name with proof of an invariant.

## 12. PostgreSQL 19, build, and verification audit

The required PostgreSQL image digest is:

`sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`

Independently verify the resolved digest and PostgreSQL server version used for review runs.

Inspect whether the exact immutable image is consistently configured in:

- Compose.
- CI.
- Taskfile defaults.
- `postgres-image.sh`.
- Relevant integration and test scripts.

Do not substitute another PostgreSQL major version.

Reproduce the applicable test and verification commands using a disposable PostgreSQL 19 database. Never connect to or modify a long-lived development database.

Where environment constraints prevent a test, record it as not executed. Do not convert source inspection or pinned CI configuration into evidence that CI actually ran.

The existing report states that CI was pinned but not executed. Preserve that distinction.

Re-run, where the environment permits:

- Full Go test suite.
- Race-enabled relevant packages and race tests.
- DBOS and Class D suites.
- `go vet ./...`.
- `gofmt` verification.
- `bash scripts/checks.sh`.

Record exact commands, versions, exit codes, and relevant output. No test result may be claimed without actual execution evidence.

## 13. Residual and deferral review

Revalidate every residual carried forward from Gate 1 and Prompt28, including:

- M5 sequence verifier deferral and activation trigger.
- Unknown-capability taxonomy behavior.
- Empty trust-mode coherence.
- Mint-failure branch.
- AST aliasing concerns.
- X-12.
- Ledger boundary.
- Two-minute lease semantics.
- Settle operator authority.
- Class B/C adapter conformance.
- CI not executed.

For each, classify:

- Is it still real?
- Is it bounded?
- Is it explicitly accepted or blocked?
- Does it weaken an invariant?
- What event or workstream activates it?
- Does it block Stage 3A closure?

Do not silently close residuals because tests are green.

Do not expand the scope to implement deferred work.

## 14. Finding severity and verdict rules

Use these evidence classifications:

- Execution-proven.
- Source-proven.
- Composition-proven.
- Characterization-only.
- Accepted residual.
- Blocked/unverified.

Use these finding severities:

- CRITICAL: fundamental security, data-integrity, or correctness failure.
- HIGH: material failure of an invariant, authorization boundary, or recovery guarantee.
- MEDIUM: important gap that needs correction or explicit owner disposition before progression.
- LOW: bounded hygiene or documentation concern that does not compromise the governing guarantees.

Group findings by underlying defect class. Identify the violated invariant, actual failure mode, affected paths, and missing discriminating tests.

Do not inflate severity to demand cosmetic changes. Do not downgrade a real security or correctness failure because the normal test suite is green.

The verdict must be one of:

- **NOT CLEAR** — unresolved CRITICAL/HIGH findings or other material Gate 2 blockers.
- **CLEAR WITH SCOPED EXCEPTIONS** — core guarantees verified, with explicitly bounded owner-blocked or accepted residuals.
- **CLEAR** — all Gate 2 requirements satisfied with no blocking exceptions.

The Settle operator wiring may remain a scoped owner-blocked exception if the review confirms that the existing implementation correctly stops at the authority boundary and the remaining work does not invalidate the reviewed Stage 3A guarantees.

A reviewer must not declare all four execution classes conformant. Class B/C adapter conformance remains outside Prompt28.

## 15. Required deliverable

Create one independent review report:

`docs/phase2/adv-review-prompt28-gate2.md`

The report must include:

1. Executive verdict.
2. Exact reviewed HEAD, branch, and worktree state.
3. Evidence and digest verification.
4. Findings ranked by severity.
5. Authorization-matrix assessment.
6. Concurrency and crash-recovery assessment.
7. C6 integration assessment.
8. Grant hash, health, and revocation assessment.
9. Settle crash-safety and operator-authority assessment.
10. Provider bypass and effect-boundary assessment.
11. Correct C1–C7 traceability table.
12. PostgreSQL 19 and test-verification results.
13. Residuals, accepted limitations, and activation triggers.
14. Explicit statement of what has not been proven.
15. Gate 2 release recommendation.
16. Recommended next architectural work order.

For every finding, provide:

- Unique finding ID.
- Severity.
- Relevant invariant or contract.
- Exact source/test/document references.
- Reproducible evidence.
- Concrete failure mode and impact.
- Why current coverage does or does not catch it.
- Required disposition.
- Whether it blocks Gate 2.

For every closed finding, explain the evidence that justifies closure.

Do not merely restate the Prompt28 report. The value of this deliverable is independent verification.

## 16. What follows Gate 2 — recommendation only

Do not implement the following work in this review.

If Gate 2 is clear, the recommended next architecture work is to establish two coordinated foundational contracts:

### Workstream S — Miniapp Security Contract

Define the common security model for:

- Human, AI-agent, and service identities.
- Authentication adapters for API, MCP, and CLI.
- Verified principal and actor context.
- Tenant isolation.
- Authorization and capabilities.
- Agent delegation, least privilege, expiration, and revocation.
- Credential and secret handling.
- Background-worker authority propagation.
- Execution-time authorization.
- Relationship to the existing `authorizeAttempt` and effect boundary.
- Security audit attribution and negative tests.

Miniapp owns authorization enforcement. Host applications and identity providers may own identity issuance. MCP must not become a privileged backdoor.

### Workstream O — Miniapp Observability Contract

Define the common model for:

- Durable audit.
- Structured operational logs.
- Metrics.
- Distributed traces.
- Correlation identifiers.
- OpenTelemetry instrumentation and OTLP export.
- Trace propagation across API, MCP, CLI, workers, and execution classes.
- Redaction and security boundaries.
- Metric cardinality.
- Sampling and retention.
- Operational diagnostics, dashboards, alerts, and SLOs.
- Instrumentation testability and cross-class conformance.

The authoritative audit and execution ledgers remain the source of truth for consequential state. External telemetry systems provide search, aggregation, visualization, alerting, and operational diagnosis.

### Joint sequencing

Security and Observability are co-equal, critical foundational capabilities. Their contracts must be designed together before accelerating Class B and C.

The shared foundations should first be integrated with existing Class A and D paths. Class B and C must implement both contracts from their first working versions.

No class is production-ready without security and observability conformance.

The next implementation work order must not be drafted until the Gate 2 review is complete and its findings have been dispositioned.

## 17. Explicit prohibitions

- No Phase 3 work.
- No Security or Observability implementation in this review.
- No Class B/C adapter implementation.
- No production operator role invented or assigned.
- No Settle service or CLI wiring.
- No sequence subsystem implementation.
- No migrations or dependency changes.
- No real provider writes.
- No commits, staging, pushes, releases, or deployments.
- No overwriting or resetting outside author activity.
- No weakening existing authorization, audit, lock-order, effect, or recovery boundaries.
- No unsupported claims of all-four-class conformance.

If a material question cannot be answered from source and reproducible evidence, record it as blocked or unverified. Do not guess, patch around it, or seek owner approval for a decision that the review does not need to make.

## Final instruction

Be adversarial about the guarantees, not adversarial toward the implementation author.

A green test suite is evidence, not a verdict. A source-level guard is evidence, not proof of every runtime path. A report is a claim, not independent verification.

Determine whether the system actually preserves its invariants under failure, contention, revocation, ambiguous outcomes, and adversarial invocation.

Stop after producing the review report and its evidence. Do not proceed into the next workstream.