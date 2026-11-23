The net-valid consolidation is: **P2 opening is accepted, WS-C and FaultInjector may proceed, and WS-A remains gated on successful WS-C exit—including empirical X-07 red/green evidence.**

I would make three clarifications to the latest review:

- DETACH-ONLY provenance is already settled by your explicit owner approval in this conversation. It does not depend on who authored the review directive.
- FaultInjector must be independently validated before WS-A, but it should not be used to manufacture the X-07 red/green result. X-07 has a direct, deterministic test.
- The `RECONCILING` removal needs a substantive §16 record and a test of the audit boundary, not merely confirmation that the state disappeared from the table.

Here is the consolidated directive.

# P2 Opening Review — Consolidated WS-C + FaultInjector Directive

**Date:** 2026-11-23\
**P2 status:** OPEN\
**Current workstream:** WS-C (X-07) + FaultInjector\
**Next workstream:** WS-A, gated on WS-C exit\
**Scope:** P2 implementation under the accepted Prompt 15.2 contract and consolidated P2 exit requirements.

## 1. Decision

P2 opening is accepted. The reported documentation updates, red/green snapshot preparation, and workstream sequencing are consistent with the accepted P2 discipline.

**WS-C and FaultInjector are authorized. WS-A must not begin until WS-C exit is demonstrated and reviewed.**

This is not an authorization to weaken the evidence requirements, bypass the red/green gate, or treat the existence of a test as proof that it discriminates.

The reported P2-opening changes are accepted as the current working record. Their empirical correctness remains subject to implementation evidence and review.

## 2. Resolved decisions and outstanding conditions

### Already accepted

- `docs/prompt15.2.md` remains governing.
- DETACH-ONLY cancellation is directly owner-approved: cancelling an attached invocation cancels only itself, preserves its `attached_run_id`, and never cancels the shared target or other observers. This decision's authority comes from the owner's explicit approval, not from the authorship of a review document.
- Removing `RECONCILING` in favor of atomic single-transaction settlement is an accepted design choice, subject to the contract record and crash/retry evidence below.
- OpenCode remains `NOT RUN / BLOCKED`, non-blocking for P2.
- Grant-revocation Conditions A–E remain P2 exit requirements, not WS-C opening tasks.

### Carried forward

| Condition                      | Current disposition       | Required follow-through                                                                                                           |
| ------------------------------ | ------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| 7 — Review-document provenance | Opening evidence recorded | Verify authorship and review status at P2 exit; preserve the direct owner approval of DETACH-ONLY independently                   |
| 8 — Multi-colon probe ID       | Open                      | Resolve concretely in migration 000006; escape with collision proof or fail with actionable diagnostic and documented remediation |
| 9 — Corrections-never-silent   | Addressed                 | Preserve in §13 and follow during implementation                                                                                  |
| 10 — Quarantine visibility     | Addressed in contract     | Verify reason field and explicit resume/fail operator action in the actual Status API                                             |
| 11 — DETACH-ONLY provenance    | Owner decision settled    | Retain the direct approval evidence; no further decision is needed                                                                |
| 12 — `CANCELLED` effect state  | Open                      | Prove it is reachable through a real cancellation path; remove it if it proves unreachable                                        |

Grant-revocation Conditions A–E remain tracked for their relevant implementation and operations workstreams. They must not disappear from the P2 exit checklist.

## 3. Required §16 record — removal of RECONCILING

The removal of `RECONCILING` is a substantive state-machine change and must have an explicit §16 amendment entry.

The entry must state:

**What changed**

`RECONCILING` was removed from the effect state machine. Operator settlement now performs an atomic, single-transaction transition from `UNKNOWN` to the selected terminal outcome, including the attributable audit record.

**Why**

The previous intermediate state could strand settlement between transactions without adding a useful, independently actionable state. A single transaction ensures that a crash before commit leaves the effect `UNKNOWN`, while a committed settlement leaves both the terminal outcome and its audit record durable.

**Tests**

`TestSettleCrashConverges` must exercise the actual settlement failure boundaries, including:

1. Failure at the audit insertion boundary before transaction commit: no partial settlement remains; a subsequent Settle attempt converges.
2. Loss of the caller's response after successful commit: a repeated Settle observes the committed outcome and does not duplicate the audit.
3. Repeated identical Settle requests are idempotent.
4. Conflicting settlement requests cannot overwrite the first committed decision.

If the implementation includes an explicit audit-backfill path, test that path directly rather than inferring its correctness from a normal successful settlement.

**Preserved guards**

- `UNKNOWN` remains non-retryable for provider attempts until an operator settles it.
- `UNKNOWN → ATTEMPTING` directly remains prohibited.
- Business-attempt retry must pass through the full authorization protocol.
- The removal of `RECONCILING` must not weaken the prohibition against blind retries.

The test must demonstrate rollback, retry, and idempotency at the relevant transaction boundaries. Merely checking that the state is absent from a transition table is insufficient.

## 4. WS-C — X-07 mock measurement integrity

### Required implementation

The mock provider-attempt log must represent actual send attempts:

- One durable row per actual mock `Channel.Send` invocation.
- A serial primary key.
- No deduplication by logical operation ID.
- Correct tenant attribution for each attempt.
- No false comments or exactly-once claims.
- `SentCount` must measure actual recorded sends, not unique business operations or reservations.

### Required discriminating test

`TestMockRecordsEveryAttempt` must:

1. Invoke the real mock `Channel.Send` boundary twice for the same logical operation.
2. Assert that two separate attempt rows exist.
3. Assert that the rows have distinct serial identifiers.
4. Assert correct tenant and operation attribution.
5. Demonstrate that the count represents actual send calls.

The test must not merely inspect the schema or verify the primary-key type.

### Affected-suite review

Re-run every affected suite under the corrected log semantics.

Read every assertion that previously used `SentCount == 1` as evidence of exactly-once behavior.

Distinguish two separate questions:

- Does the mock record every actual send? X-07 establishes this.
- Does the framework prevent an unintended duplicate send? The affected framework tests must establish this independently using the now-trustworthy measurement.

A failure that reveals an actual duplicate send is a framework defect to fix, not a mock assertion to weaken. A test that was previously vacuous must be rewritten so that it proves the intended invariant.

### X-07 red/green exit evidence

Run the identical `TestMockRecordsEveryAttempt` source against:

- The isolated pre-remediation snapshot at digest `893df60f…`, using `miniapp_red`.
- The remediated implementation, using `miniapp_dev`.

The red result must fail because the old mock log suppresses the second attempt. The green result must pass because both actual send invocations are recorded.

Record:

- Complete implementation/source-scope digest.
- Test-harness digest.
- Full-tree digest.
- Exact test command and environment.
- Database identity for each run.
- Expected and observed failure messages.

The test must fail for the predicted X-07 defect, not for an unrelated build, environment, or database problem.

## 5. FaultInjector — independent trust requirements

FaultInjector is the instrumentation that WS-A will rely on to demonstrate failure and recovery behavior. Its own correctness must therefore be established before WS-A begins.

### Required behavior

The injector must support:

- A named operation boundary.
- Failure on the configured Nth invocation.
- One-shot and persistent failure modes.
- Deterministic counters and outcomes.
- Per-test isolation; no shared mutable state across unrelated tests.
- Explicit propagation of the injected error through the actual caller.
- No production behavior change when the injector is absent.

The injector must fail at the named operation boundary, not at an arbitrary earlier point that happens to produce a similar error.

### Production-constructor safety

`Service.TestFault`, `effects.Service.TestFault`, and `approvals.Service.TestFault` must remain nil in production constructors.

Required proof:

- A test asserts nil immediately after every production `New` constructor.
- No environment variable, configuration file, miniapp manifest, request field, or public runtime operation can populate the injector.
- Test setup is the only intended source of injection configuration.
- Tests cannot accidentally reuse a prior test's injector state.

The nil-after-`New` assertion is necessary, but it does not alone prove that no production configuration path can populate the field. Verify both properties.

### Fault-point inventory

Reconcile the implementation and test inventory against every §9 fault point, including:

- Approval creation.
- Approval linkage.
- Park update and audit.
- Run-ID persistence.
- DBOS submission.
- Approval decision commit.
- Signal transmission.
- Effect reservation.
- Provider send.
- Settlement.
- Completion audit.
- Shutdown.

Each fault point must have a named test showing that the intended operation was reached, the configured failure occurred, and the resulting lifecycle behavior is correct.

### Injector validation

Before WS-A, demonstrate:

- Exact-Nth invocation behavior.
- One-shot behavior.
- Persistent behavior.
- Independent test instances.
- Correct failure propagation.
- No firing when not configured.
- Nil-after-production-construction.
- No cross-test contamination under race-sensitive execution.

Do not claim that the injector is trustworthy merely because it compiles or because one injected failure works.

## 6. Keep the two evidence tracks separate

**X-07 red/green** proves that the mock measurement defect existed and was corrected. It must use the actual mock channel and does not depend on FaultInjector.

**FaultInjector contract tests** prove that the new test instrumentation is deterministic and trustworthy. The new utility is not expected to compile or run against a pre-remediation snapshot in which it does not exist. Do not manufacture a red result by overlaying incompatible new test utilities onto an old source tree.

Where FaultInjector is used to test an implementation defect, its firing and boundary must be independently demonstrated.

## 7. Pre-remediation snapshot integrity

Before using the red snapshot as evidence:

- Recompute its full-tree and source-scope digests.
- Confirm the isolated worktree matches the recorded `893df60f…` snapshot.
- Verify its manifest and record any differences.
- Confirm that no source change occurred in the snapshot after its recorded creation.
- Keep the red and green database identities separate.

Expected green implementation changes do not invalidate the red snapshot. The requirement is that the red snapshot itself remains the exact, immutable pre-remediation baseline.

## 8. WS-C exit gate — mandatory before WS-A

WS-C is complete only when all of the following are true:

1. The per-attempt mock log records every actual send, without deduplication.
2. `TestMockRecordsEveryAttempt` proves double-send produces two rows through the actual mock Channel.
3. The test fails against the exact pre-remediation snapshot for the predicted reason.
4. The same test passes against the remediated implementation.
5. All affected `SentCount` and attempt assertions have been reviewed and re-run.
6. Tenant attribution is correct.
7. False exactly-once claims and comments are corrected.
8. FaultInjector passes its independent deterministic contract tests.
9. All §9 fault points are inventoried and reconciled.
10. Production constructors leave injection disabled, with no production configuration path to enable it.
11. Source-scope, harness, and full-tree digests are recorded.
12. There are no unexplained source changes or weakened tests.

**WS-A remains gated until this evidence is submitted and reviewed.**

If X-07 red/green cannot be demonstrated, stop and surface the result. Determine whether the old defect was absent, the test is nondiscriminating, or the remediation addresses the wrong mechanism. Do not proceed by changing the expected result.

## 9. Operating principle

The framework owns reliability; miniapps own business logic.

The mock must measure actual attempts. FaultInjector must provide reusable, deterministic failure primitives. Downstream tests must prove framework guarantees rather than rely on unverified instrumentation.

**Authorization:** Proceed with WS-C and FaultInjector only. WS-A begins after WS-C exit acceptance.

One final distinction: the `893df60f…` snapshot's self-digest proves its recorded identity only when independently reproduced. It does not by itself prove the working tree has remained unchanged; the manifest and source-scope checks above are the relevant evidence.