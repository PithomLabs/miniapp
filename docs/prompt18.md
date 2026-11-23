I inspected the GitHub Actions job logs. The failure is **two DBOS restart integration tests timing out**, not a formatting, compilation, or vet error.



CI FAILED

**Commit:** `982aecfb4435031f9ffd68bca90e06f7f2f1dad5`



## **1. Exact failures**

| Package         | Test                     | Failure                                              |
| --------------- | ------------------------ | ---------------------------------------------------- |
| `executor/dbos` | `TestSleepAcrossRestart` | Worker never produced `submitted` within 120 seconds |
| `probe/classd`  | `TestProbeRestart`       | Worker never produced `submitted` within 120 seconds |

Both failures occurred around the same time.

Passed

- `gofmt`
- `go vet ./...`
- Other reported packages, including `runtime`, `effects`, `approvals`, `executor/classa`

Not executed because the test step failed:

- Race tests
- `bash scripts/checks.sh`

## **2. Important observations**

The failure signature is significant:

Both tests expect an external worker to produce a `submitted` marker, but neither marker appears.

That points toward a shared problem in the DBOS restart-test execution path, rather than two unrelated business-logic failures.

Possible causes to investigate include:

1. Worker subprocess startup or environment configuration.
2. DBOS system-database initialization or connectivity.
3. Workflow registration or queue/worker startup ordering.
4. Restart recovery behavior or workflow identity.
5. An assumption in the tests about when the worker becomes ready.

These are hypotheses, not established causes. The logs do not reveal which one is responsible.

## **3. My recommendation**

Do not increase the 120-second timeout or skip these tests. That would mask the underlying issue.

Investigate the shared worker-startup path first, then compare the two restart tests to identify their common dependency.

Here is the implementation prompt I recommend giving your coding agent:

# CI Failure Investigation — DBOS Restart Tests

Repository: `github.com/PithomLabs/miniapp`

Failing commit: `982aecfb4435031f9ffd68bca90e06f7f2f1dad5`

GitHub Actions run: `45115104815`

## Mission

Investigate and fix the root cause of two GitHub Actions integration-test failures:

- `executor/dbos`: `TestSleepAcrossRestart`
- `probe/classd`: `TestProbeRestart`

Both tests time out after 120 seconds because the worker never produces the expected `submitted` marker.

Do not increase the timeout, skip the tests, weaken their assertions, or introduce test-only workarounds that conceal the underlying failure.

## Phase 1 — Establish the exact environment

Inspect:

- The exact failing commit.
- The CI workflow and its environment variables.
- Both test implementations and their subprocess helpers.
- DBOS initialization, worker startup, workflow registration, and shutdown.
- How the tests establish readiness and detect successful submission.
- PostgreSQL and DBOS system-database setup.

Reproduce the failure using the same Go version and equivalent PostgreSQL/DBOS configuration where possible.

Record differences between local and GitHub Actions environments.

## Phase 2 — Find the shared failure mechanism

Trace the complete execution sequence:

1. Parent test process starts.
2. Database and DBOS configuration is established.
3. Worker subprocess is launched.
4. Worker initializes DBOS.
5. Workflow registration completes.
6. Worker becomes ready.
7. Workflow submission is attempted.
8. Worker produces the `submitted` marker.
9. Parent test observes the marker.
10. Process restarts and recovery proceeds.

Identify the earliest step where actual behavior diverges from the expected sequence.

Compare the two failing tests to identify shared code and dependencies.

Investigate:

- Process environment propagation.
- Working directory and executable path.
- Subprocess stdout/stderr and startup errors.
- DBOS initialization failures.
- Database connection readiness.
- Worker registration and lifecycle ordering.
- Workflow queue configuration.
- Restart semantics and persisted workflow identity.
- Race conditions around readiness signaling.
- File-based marker paths and permissions.

Do not infer that a missing marker means DBOS failed to start. Collect evidence showing where execution stopped.

## Phase 3 — Root-cause correction

Fix the underlying shared mechanism at the appropriate framework or test-harness boundary.

The solution must:

- Preserve production DBOS behavior.
- Preserve the test's original assertion and purpose.
- Make worker startup and readiness observable.
- Propagate startup failures rather than hiding them.
- Ensure subprocesses are cleaned up on every exit path.
- Avoid sleep-based synchronization where deterministic readiness signaling is possible.
- Preserve restart/recovery semantics.

Do not alter business workflow semantics to make the integration tests pass.

## Phase 4 — Regression tests

Add tests that distinguish the actual root cause from the corrected behavior.

At minimum:

- Worker startup failure is reported with actionable diagnostics.
- Readiness is not reported before the worker is actually ready.
- Successful worker startup produces the expected marker.
- Restart recovery uses the intended persisted workflow identity.
- Parent-test failure reliably terminates child processes.
- Both original restart tests remain discriminating.

If the failure is environment-specific, add a reproducible CI-equivalent test or diagnostic check. Do not claim local reproduction if it was not achieved.

## Phase 5 — Verification

Run:

1. Both failing tests independently.
2. Both tests repeatedly.
3. Their respective package suites.
4. `go test ./...`.
5. Relevant race tests.
6. `gofmt`, `go vet ./...`, and `bash scripts/checks.sh`.

Record exact commands, environment, outputs, and commit/tree identity.

## Constraints

- No timeout increases as a substitute for fixing the cause.
- No skipped tests.
- No weakened assertions.
- No unrelated architectural redesign.
- No changes to governing documents without identifying a genuine contract conflict.
- No commits or pushes without explicit authorization.

## Deliverable

Provide a root-cause report containing:

- The exact startup/recovery step that failed.
- Evidence establishing the underlying cause.
- Why both tests exhibited the same symptom.
- The durable correction.
- Regression coverage.
- Reproduced red and green results.
- Remaining risks.

The objective is to fix the underlying failure mechanism, not merely make this particular CI run green.

## **4. CI reference**

[Open the failed GitHub Actions run](https://github.com/PithomLabs/miniapp/actions/runs/45115104815?utm_source=chatgpt.com).

**My architectural concern:** these are restart-recovery tests. They protect one of the central claims of the framework: durable workflows survive process restarts. Treat this as a correctness issue, not ordinary flaky-test cleanup.