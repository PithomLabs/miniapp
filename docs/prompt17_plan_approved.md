**Decision: (a) Accept the CI delta as explained and expected. Do not revert it or set it aside.**

I am explicitly waiving the original source-delta hard stop for this precisely identified, documented change set. This is a controlled exception, not a general relaxation of the baseline-verification rule.

The reason is that the reported changes are confined to CI configuration, test code, and test-process infrastructure, and have been reconciled file-for-file against `ci-restart-root-cause.md`. The investigation is valuable evidence and should be preserved.

## **1. Conditions of acceptance**

The exception covers exactly:

- Five modified files: `ci.yml`, `concurrency_test.go`, `dbosexec_test.go`, `probe_test.go`, `proc.go`.
- Four untracked files: `proc_test.go` and the three investigation documents.

No additional files are implicitly accepted.

Before closing Phase 0, verify and record that:

1. Every changed Go file is test code or test-process infrastructure, not production runtime behavior.
2. All nine files match the inventory and explanations in `ci-restart-root-cause.md`.
3. No hidden production behavior change exists in the diff.
4. The current working tree remains clean.
5. Full file hashes and the complete current-tree digest are preserved.

Any discrepancy outside that inventory reactivates the hard stop.

## **2. Preserve three distinct source states**

Do not collapse these into one baseline.

| State                    | Identity                      | Purpose                                    |
| ------------------------ | ----------------------------- | ------------------------------------------ |
| WS-A acceptance baseline | `e20a4689…`                   | Immutable historical foundation            |
| CI-investigation delta   | Nine identified files         | Separately preserved test/CI investigation |
| WS-B starting tree       | `982aecf` plus verified delta | Actual implementation starting state       |

The acceptance baseline must remain byte-exact.

The current tree is the authorized starting point for WS-B because the investigation delta is accepted.

**Red/green discipline:** For the X-07 smoke test, use the reconstructed acceptance baseline as proposed. For subsequent WS-B regression tests, any CI/test infrastructure required by the test must be present identically on both red and green sides. The regression test bytes must also be identical. Only the implementation under test may differ.

This prevents a test-harness difference from being mistaken for an implementation correction.

## **3. Complete the remaining Phase 0 work**

PHASE 0 — CONTINUE

I authorize the four remaining steps:

1. Freeze the exact `e20a4689…` directory copy and separately freeze the CI-investigation delta.
2. Validate the per-finding overlay procedure using X-07 red/green.
3. Execute all three GitHub diagnostic configurations:
   - No database, parallel execution.
   - Database-backed, parallel execution.
   - Database-backed, serial execution with `-p 1`.

Record actual results, including failures and timing. The expected outcomes are hypotheses to test, not results to manufacture.

4\. Produce immutable evidence-register copies and the Phase 0 completion report.

For the CI diagnostic, retain both possible contributing causes as distinct hypotheses:

- Missing PostgreSQL in the no-database CI configuration.
- Shared-database contention under parallel package execution.

Do not change CI configuration, worker behavior, or test semantics during this diagnostic.

## **4. Phase boundary**

No Phase 1 implementation starts until the Phase 0 report establishes that all checks passed and the baselines are reproducible.

After a clean Phase 0, proceed with the previously approved Prompt 17 sequence, beginning with M-01. No further approval is needed for routine execution within that scope.

The advisory-lock correction, L-01 through L-04, O-05, and the remaining phases retain their previous authorization.

**Final disposition:** Preserve the CI investigation. Freeze the historical acceptance baseline. Use the verified current tree as the WS-B starting tree. Complete Phase 0 without changing implementation behavior.

No commits, pushes, Neon access, or OpenCode access are authorized.