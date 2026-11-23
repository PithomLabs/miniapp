# Prompt 20 — WS-B Phase 1 Closeout Report

**Work order:** `docs/prompt20.md` (governing: `docs/prompt17.md`)
**Review closed:** `docs/adv_review_prompt19.md` (NOT CLEAR → this closeout)
**Status:** all authorized items complete; submitted for focused
independent adversarial review. **Phase 2 remains blocked** until
that review returns CLEAR with explicit owner release.

## 1. Baselines and provenance

| Artifact | Digest / identity |
|---|---|
| Phase 1 green manifest (`p17-phase1`) | `5afb6e46…` — 300 entries verified (299 identical; README = 5 documented rows appended, pure append) |
| Phase 1 red tree | `c9c08fe5…` recomputed exactly; accept-vs-red diff = `m01_test.go` only |
| `runtime/m01_test.go` full SHA-256 | `222f5ff5…` both sides |
| P20 freeze-green (`/tmp/p20-freeze-green`, read-only) | `d953a400…` (re-verified pristine at closeout) |
| P20 freeze-red (`/tmp/p20-freeze-red`, read-only) | `c9c08fe5…` (re-verified pristine at closeout) |
| Closeout tree + manifest (`p20-closeout-2026-11-23`) | `a8ffe2f4…` |
| Register pre-image (`p20-phase0.README.md`) | `acff00c6…` |
| HEAD / branch | `4945478`, `main` (no commits by this workstream) |
| Environment | Go 1.27.3, PG `19.0 (Debian 19.0-1.pgdg13+1)`, serial `-p 1`, `GOWORK=off` |

§2.7 discrepancy (reported before proceeding, owner-accepted):
docs-only additions, zero source files — tracked review/diagram
materials at HEAD plus untracked `prompt20.md` (this work order),
`prompt20_plan.md` (owner-saved approved plan), `prompt21.md`
(owner CI scratch, separate workstream, not acted upon).
Planning→freeze delta proven = exactly the latter two files
(deltacheck re-digests `0c8f25b4…`). All red/green runs executed
in disposable per-finding copies; frozen baselines never written
(re-verified above); all five register-bound test files
byte-identical at closeout.

## 2. HIGH-1 — settlement versus completion (CLOSED)

Owner rescoped the unreachable granted-poll race to
trust-float-dispatched Class A completion versus expiry
settlement (Q1 approved). Phase 0 proved granted and settle
observations mutually exclusive per invocation (`Decide`
enforces expiry, decided immutable, single live approval).

- Production change: two comment corrections only
  (`completeIfLive`, `completeSync` cite settlement, not Cancel).
- Tests (`runtime/p20_high1_test.go`, `f2613224…` identical):
  `TestP20SettleWinsOverCompletion` (strictly sequential forced
  order, no queue assumption) and
  `TestP20CompletionWinsOverSettle` (forced queue + post-hoc
  winner proof), 3 rounds each, persisted-state + exact-audit
  assertions. Store boundary re-verified via
  `TestPredicatedOpsFromStates`.
- Red (`fb7a17f0…`, acceptance + overlay): both FAIL round 0 —
  false success over settlement / false failure over completion.
- Green (`d42b6c25…`): both PASS 3/3, race-clean. No mechanism
  defect found; no redesign per scope restriction.

## 3. MEDIUM-1 — CR-05 run identity (CLOSED)

- Fix A: `failIfWaiting` drops the `run_id` write (parameter
  retained blanked for identical test sources).
- Fix B: `adoptRunID` gains the live-state predicate
  (`pending`/`running`/`waiting_event`/`waiting_approval`).
- Fix C: `submitChecked` refuses submission for ineligible
  invocations after reload, via `adoptEligible` — covering both
  the adoption-loss path and the already-bound path (the latter
  exposed by the bound-leg test during green; same predicate and
  location, within authorized scope).
- Tests (`runtime/p20_medium1_test.go`, `04632e7c…` identical):
  stale-settle-preserves-R, adopt-refuses-terminal/quarantined
  (+ live controls), losing/rejected-adoption-submits-nothing
  (both legs, submit counter). Sequential committed-operation
  order; API-level deterministic staging proven unforceable
  (Phase 0), store/service boundary owns the guarantee.
- Red (`dd3a7db3…`, pre-fix snapshot + overlay): all 3 FAIL —
  clobbered R, adoption onto failed winning, both legs
  submitting. Green (`0799839a…`): all 3 PASS, race-clean; full
  `./runtime/` green (X-01/X-02/X-03 unaffected).
- Retained residual (authorized, untouched): adopt-wins →
  settle-wins → submit orphan belongs to Phase 2/O-01.

## 4. MEDIUM-2 — single Class A execution (CLOSED)

Design-first per authorization: `classa-execution-claim-design.md`
answered all 8 questions and proposed the claim + single-tx
mechanism (no memo needed); owner approved with 4 conditions, all
implemented and tested.

- Mechanism: 64-bit namespaced advisory xact claim (ephemeral,
  crash-safe, cross-process) + handler and predicated completion
  committed atomically + savepoint handler-error semantics +
  terminal 409 `execution-in-progress` conflict with same-key
  retry; nil hooks fail closed; sole production caller audited.
- Guarantee as authorized: at most one concurrent handler
  execution, and at most one committed transactional business
  execution per invocation. Non-transactional side effects stay
  outside it (ledger boundary, stated in code, contract, and
  report).
- Tests: 5 runtime (`0bd1fb60…` identical, red overlaid) — T1
  granted polls, T2 dispatch-vs-replay, C3 retryable conflict
  (promptness + untouched row + explicit retry), T5 pre-commit
  crash (Kill9, characterization), T6 post-handler-commit crash
  (Kill9 at gate-blocked completion); 4 classa unit (green-only,
  new-API: nil-claim, terminal recheck, savepoint rollback,
  contention).
- Red (`6dbd8362…`): T1/T2/C3 FAIL with `handler entries so far:
  2`; T6 FAILS `business rows = 2` (two-tx double-apply);
  T5 PASSES both sides as documented characterization.
- Green (`2008f533…`): all 9 PASS incl. `-race`; full runtime
  green (HIGH-1 scenario A holds under rollback semantics).
- Contract: §4 re-proof (two ordered advisories) + §13/§16
  entries. Savepoint-claim survival verified live on PG19.

## 5. MEDIUM-3 — lock-order instructions (CLOSED)

`prompt17.md` and its plan-review mirror corrected identically:
§4 authoritative, advisory-last struck with REMOVED provenance,
`authorizeAttempt` takes no advisory lock, outermost-first
preserved. Repo-wide grep: no advisory-last instruction remains
in governing docs; remaining mentions triaged (prompt20/plan =
fix description; historical reviews/writeups preserved as
records, listed in evidence). Zero code change; verified
advisory implementation untouched.

## 6. Small items (CLOSED per approved dispositions)

| Item | Disposition | Proof |
|---|---|---|
| LOW-1 duplicate marker | `replay` gains `isDuplicate` (Invoke: true; both loss paths: false) | 2 red/green tests (`be385e54…`): red 4-vs-3 markers both directions; green PASS; true-duplicate pin unaffected |
| LOW-2 audit backfill | X-12 row extended with backfill scope; no code | Contract text; deferred-work entry (§9) |
| LOW-3 phasing precision | §4.2 Cancel (X-05) + §9 pre-dispatch/park + §16 provenance cite | Contract text |
| INFO-1 live terminology | `live` defined + op exclusions at CR-05 | Contract text; all sets verified against predicates |
| INFO-2 advisory guard | AST single-site test (`168e5e40…`) | Green run (both primitives found in classa.go); guard-only, no red applies |
| INFO-3 32-bit hash | Accepted residual, no change | §9 |
| INFO-4 completion table | `prompt19.md` §5.2 corrected (protocol doc; historical review untouched) | Doc text |
| INFO-5 quarantine paths | Deferred to X-10 workstream, no code | §9 |
| INFO-6 digest discipline | Every digest logged with exact command + CWD + path forms | §7–§8 |
| INFO-7 run-less/trust docs | M6: uncancellable-by-API + effective-trust-floats rule | Contract text |

No X-12/quarantine implementation; no architecture reopened.

## 7. Verification battery (all green, this tree)

CWD repo root unless noted; DB URLs per run (`miniapp_dev`
green, fresh `miniapp_red` red); `GOWORK=off`.

- `gofmt -l .` → clean (no output).
- `go vet ./...` → clean.
- `go test -p 1 ./... -count=1` → every package ok (21 with
  tests; dbos 37.5s, probe/classd 39.1s, runtime 4.9s).
- `go test -race -p 1 ./runtime/ ./approvals/ ./effects/
  -count=1` → all ok.
- `bash scripts/checks.sh` → ALL-CHECKS-PASS.
- Targeted repeats: crash/recovery + HIGH-1 races `-count=2`,
  classa suite → ok (repetition supplements, never replaces,
  the gated/deterministic staging).
- Per-finding red/green commands in evidence rows (identical
  test bytes; overlays listed per row; frozen baselines
  untouched).
- Cleanup verified: 0 idle-in-transaction backends, 0 stray
  test processes, DBs = `miniapp_dev` + `miniapp_red` scratch
  only. Skipped/blocked: none (crash-worker entrypoints skip
  by design outside parent-spawned runs; they RAN under T5/T6).

## 8. Findings register

| Finding | Invariant | Root cause | Discriminating test |
|---|---|---|---|
| HIGH-1 missing proof | M-01 transition integrity at every site | Test-planning gap (sync completion assumed covered) | Forced-order settle/completion races |
| MEDIUM-1 clobber | CR-05 run-ID immutability | Settle treated run_id as preservable payload | Stale-settle-preserves-R |
| MEDIUM-1 adopt-terminal | CR-05 + routing stability | No status predicate on adoption | Adopt-refuses-terminal |
| MEDIUM-1 submit-ineligible | Choke never submits for dead records | No liveness check on stale paths | Lost-adoption-submits-nothing (both legs) |
| MEDIUM-2 double-execute | Single execution per invocation | No interlock before handler | T1/T2/C3 (entries==2 on red) |
| MEDIUM-2 crash double-apply | At-most-once committed business | Two-tx handler/completion split | T6 Kill9 (rows==2 on red) |
| MEDIUM-3 contradiction | §4 authoritative lock order | Work order frozen pre-correction | Governing-doc grep + triage |
| LOW-1 mislabel | Truthful audit attribution | Unconditional marker at replay entry | 4-vs-3 marker counts |

## 9. Five-way distinction

**Completed and red/green verified:** HIGH-1 (2), MEDIUM-1
(3 + have-run extension), MEDIUM-2 (T1/T2/C3/T6 + C1/C2/C4/unit),
LOW-1 (2), MEDIUM-3 (grep), all doc dispositions. Full battery
green; frozen baselines and bound files intact.

**Source-correct but not independently proven:** the
loss-path winner-terminal-with-run branch (reachable only via
an unstageable 3-way interleave; covered by code inspection +
the sibling P3 guard proof); T5 pre-commit crash (passes both
trees by construction — pins rollback recovery, does not
discriminate).

**Accepted residual limitations:** adopt-wins→settle→submit
orphan (Phase 2/O-01 owns it; effects refused via CheckLive);
INFO-3 32-bit entity-hash contention (liveness-only);
non-transactional handler side effects outside the framework
guarantee (ledger boundary); post-win audit gap pending X-12.

**Deferred work:** X-12 audit-backfill sweeper implementation
(scope recorded); quarantine defensive-path semantics (X-10
workstream); `prompt21.md` CI scratch (separate workstream,
untouched).

**Remaining architectural decisions:** none for Phase 1. Owner
decisions recorded: Q1/Q2 rescopes, MEDIUM-2 architecture + 4
conditions, P3-guard scope confirmation, residual acceptances,
prompt21 exclusion.

## 10. Re-review handoff

Suggested focus (highest value first): MEDIUM-2 single-tx +
claim interaction with future Phase 2 authorization (lock
footprint, contention shape); Fix C loss/have-run branch
coverage; HIGH-1 scenario B FIFO reliance (forced queue +
post-hoc proof — confirm the assumption is load-bearing only
where claimed); LOW-1 aggregate-count attribution; the
governing-vs-historical scoping of the MEDIUM-3 grep triage;
anything in §9's middle three categories the reviewer believes
is misfiled. All red runs reproduce from the frozen copies +
overlay hashes in the evidence register.

**Phase 2 remains blocked until this closeout receives
independent adversarial CLEAR and explicit owner release.**
No commits or pushes were made.
