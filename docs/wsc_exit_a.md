**WS-A exit submitted. Stopping — WS-B not started, awaiting review.**

## WS-A implementation (X-01–X-04, X-19, X-22)

- **Invocation/approval state machine:** R3 pending-claim with pending-aware replay (deny-on-recompute fails closed without settling); atomic Park (invocation lock → approval INSERT → audit INSERT, one tx, adopt-on-race, no duplicates); submission choke point (`submitChecked`: reload, live-linkage enforcement, mint-or-adopt, attached-aware); submit-if-missing on poll **and** Decide (approve-only; reject skips notification of never-submitted runs); unknown-tolerant `cancelRun` (double-unknown absorbs, everything else retries); framework `executor.ErrUnknownRun` sentinel with SDK code-based matching (no string parsing); executor + workflow fail-closed layers behind the choke.
- **Decide:** row lock + predicated writes + affected-row reconciliation (identical converges, conflicts defined); predicated expiry marking.
- **Identity:** `contract.ScopeID` canonical escape shared by entity/lock/ledger/switch paths (collision + precedence + legacy-identity tests); empty run-ID rejection at Signal/Cancel; `attached_run_id` + partial unique index via migration 000006 (preflight diagnostic, dup/delimiter/unique/down-up tests).

## WS-A red/green (all HIGH findings)

| Finding | RED (snapshot `893df60f…` copy, fresh `miniapp_red`) | GREEN (worktree `1acc277c…`, `miniapp_dev`) |
|---|---|---|
| X-01 | Both discriminators FAIL predicted: blind submit with empty `ApprovalID` (X-01 bypass); dangling row ignored | PASS: choke refuses, parks/adopts, submits bound to linkage |
| X-02 | All 4 FAIL predicted: status/signal/cancel against unknown run (grant/denial/expiry stranded) | PASS: submit-if-missing + tolerant absorb |
| X-03 | FAILS round 1: divergent handles (orphan) | PASS incl. `-race`, 30/30 rounds single-workflow |
| X-04 | ApproveReject FAILS round 1 (8 successes, silent overwrites); DecideCancel FAILS round 1 (decide-nil + cancelled + final decided) | All 4 PASS: exactly one winner, defined conflicts |

Identical test bytes both sides (content hashes verified at overlay); impl/harness/full-tree/source-scope digests recorded per finding. Full suite 21/21 green, `checks.sh` PASS, preflight unchanged, snapshot re-verified pristine.

## Honest divergences (all recorded)

1. **Rule 3 interpretation** (§16-logged): choke requires *live* linkage (pending/granted), not granted-only — granted-only would forbid the reviewed park-submit-wait flow and force a forbidden redesign; prompt mechanics (S1, X-02, workflow wait) presume pending-linkage submits. Forced by the no-redesign boundary, offered for owner review.
2. **X-04 self-approval/expiry tests pass on both trees** — investigated: no race exists (immutable self-check fields; single marker pre-sweeper). Documented guard tests, not discriminators.
3. **X-03/X-04 reds are probabilistic-in-principle** (round-1 failures observed); green sides deterministic by construction. Documented per test.
4. **Two contract-driven test updates** (not weakenings): `TestFaultParkAudit` → atomic-rollback expectations; `TestFaultAttemptBegin` retry via lease-takeover. Both §16-logged.
5. **Snapshot handling correction**: the X-07 overlay ran inside the snapshot dir; removed afterward with digest re-verified exact. All WS-A reds ran in per-finding copies; snapshot untouched since.

**WS-A exit gate:** all X-01–X-04 remediations implemented, red/green proven, no regressions. Awaiting review before WS-B.
