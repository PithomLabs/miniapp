# Prompt 25 reviewer package (docs/evidence/p25)

Targeted re-review inputs for the Gate 1 corrections (prompt25).
The original Prompt 24 report is `docs/adv_review_prompt24.md`; the
correction report is `docs/phase2/prompt25-gate1-corrections-report.md`;
the full-tree manifest is
`docs/evidence/digests/p25-closeout-2026-11-23.manifest.txt`.

## Red-vs-final diffs

Each `p24XXX-red-vs-final.diff` is `diff -u` of the frozen red tree
against the final corrected tree for the finding's production
file(s). Red trees were full-tree copies under `/tmp` (since
reaped with the session); every red is exactly reproducible from
the final tree by reverting the finding's behavioral hunk(s) —
see "Reproducing a red tree" below. Later findings' hunks ride
along in earlier diffs; the map names each hunk's owner.

- `p24001-red-vs-final.diff` (`effects/authorize.go`, 9 hunks):
  P24-001 decision-commit restructure (RefusalError/RefusalReason
  comments, `authzDecision`, `authorizeAttempt`, `authorizeTx`,
  `terminalWithdrawTx`, `withdrawIfPending`, `auditRefusal`) +
  P24-005 hunks (reason const, standalone trust check, linkage
  exception, `trustAdmitted`/`autonomousLinkageExempt`) + P24-011
  hunks (invocation no-rows mapping).
- `p24002-red-vs-final.diff` (`effects/effects.go`, 7 hunks):
  P24-002 attempt predicate (`transition`, hunk `@@ -448`) +
  P24-003 attribution loader + P24-007 mint (claim/attempt/Intent
  note) + P24-011 (`ErrPhantomBinding`, settle mapping).
- `p24003-red-vs-final.diff` (`effects/effects.go`, 6 hunks):
  P24-003 `loadIdentityTx` + attribution call-site hunks, plus
  the P24-007 and P24-011 hunks above (red3 already carries the
  P24-002 predicate).
- `p24005-red-vs-final.diff` (`effects/authorize.go`, 2 hunks):
  P24-005 standalone trust admission + linkage exception
  (`@@ -221`) + P24-011 no-rows mapping (`@@ -191`). The reason
  const and helpers exist inertly on red5 (behavior-neutral).
- `p24007-red-vs-final.diff` (`effects/effects.go`, 4 hunks):
  P24-007 claim mint + attempt-site hunks + P24-011 hunks. (The
  Intent deprecation note exists on red7; inert comment.)
- `p24010-red-vs-final.diff` (`runtime/service.go`, 1 hunk):
  P24-010 effect-contention call site. Precise: the helper
  exists inertly on red10.
- `p24011a-red-vs-final.diff` (`effects/authorize.go`, 1 hunk)
  and `p24011b-red-vs-final.diff` (`effects/effects.go`,
  1 hunk): P24-011 no-rows mappings. Precise.
- P24-009 (AST policy guard) has no red tree: guard-only change.
  Old-matcher blindness is proven mechanically — the planted
  whole-map assignment line contains zero old-pattern
  substrings — and the negative control fails loudly when any
  planted line goes unflagged (observed during development).

## Pristine pre-fix sources (durable red references)

- `authorize.go.pristine` (`f3e0ebb6…`) — byte-identical to the
  frozen pre-fix `effects/authorize.go`.
- `effects.go.pristine` (`900fb6bc…`) — byte-identical to the
  frozen pre-fix `effects/effects.go`.

## Reproducing a red tree

Copy the final tree to a scratch dir (test files overlay
unchanged; identical bytes — hashes in the correction report),
then apply ONE of these reverts. Two newer test files reference
symbols their reds predate; move them aside first as noted —
they postdate the red under test, and every other P25 test
file compiles against every red:

- P24-001: move `effects/p25_p24005_internal_test.go` and
  `effects/p25_p24011_internal_test.go` aside; `cp
  docs/evidence/p25/authorize.go.pristine effects/authorize.go`.
  Exact for the P24-001 tests: they use empty-mode manifests
  (standalone check irrelevant), no phantom rows, and
  claim-conflict/takeover/direct-authorize paths (mint
  irrelevant).
- P24-002: move `effects/p25_p24011_internal_test.go` aside;
  `cp docs/evidence/p25/effects.go.pristine effects/effects.go`.
  Exact for the POC-2 tests: live autonomous ungated paths
  (standalone check passes, exception uninvolved), caller-ID
  claim (mint absent, as on the original red2).
- P24-003: move `effects/p25_p24011_internal_test.go` aside;
  same `effects.go.pristine` copy. Exact for the Settle
  tests: Settle path only (transition predicate uninvolved).
- P24-007: move `effects/p25_p24011_internal_test.go` aside;
  same `effects.go.pristine` copy. Exact for the mint tests:
  caller-ID claim + authorize (later loader/predicate hunks
  uninvolved in these paths).
- P24-005: move `effects/p25_p24011_internal_test.go` aside;
  drop the standalone trust block; restore the bare `if
  tool.Approval.Required` condition (keep the reason const +
  helpers inert).
- P24-010: move `effects/p25_p24011_internal_test.go` aside;
  restore the unconditional same-key message at the
  effect-contention site (keep the helper inert).
- P24-011: drop both isNoRows mappings (keep
  `ErrPhantomBinding` inert). All test files compile.

Run red, then green, e.g. `go test ./effects/ -run
'TestP25POC2' -count=1` against
`TEST_DATABASE_URL`/`TEST_DBOS_DATABASE_URL` on the pinned
PG19 digest (`postgres@sha256:bb6e70ad…`, see report).

## Committed regression tests (all in-tree)

`runtime/p25_p24001_test.go`, `effects/p25_p24001_internal_test.go`,
`effects/p25_p24002_test.go`, `effects/p25_p24003_test.go`,
`effects/p25_p24005_internal_test.go`, `effects/p25_p24007_test.go`,
`runtime/p25_p24010_test.go`, `runtime/p25_p24010_internal_test.go`,
`effects/p25_p24011_internal_test.go`,
`auth/policy_guard_test.go`,
`auth/testdata/policy_guard_violations.go`.
