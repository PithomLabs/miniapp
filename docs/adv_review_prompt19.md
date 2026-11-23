# Adversarial Review: WS-B Phase 1 (M-01 Lifecycle Races + L-01–L-04, O-05)

**Verdict: NOT CLEAR** — one HIGH finding (missing required race proof for the headline mutation, plus a requirement that needs owner rescoping) and three MEDIUM findings block Phase 2 release. The M-01 mechanism itself is sound as far as it is proven: all delivered race tests genuinely discriminate, all green claims reproduced, no demonstrated defect in the predicated-transition design. Remediation is bounded (two tests, two small code fixes, one prompt correction, one owner disposition).

- **Reviewer role:** independent adversarial reviewer (Plan §1 posture). Read-only except this report; no fixes, no commits.
- **Governing spec:** `docs/prompt17.md` (WS-B work order), `docs/phase1/invocation-lifecycle-contract.md`, `docs/prompt19.md` (this review's protocol).
- **Review date:** 2026-11-23. HEAD `982aecfb4435031f9ffd68bca90e06f7f2f1dad5`, branch `main`.
- **Environment deltas (recorded, not hidden):** Go 1.27.3 toolchain (repo floor 1.26.0; CI uses 1.26); disposable PostgreSQL 16.15 user-space cluster instead of local Docker PG19 (no Docker daemon, no Neon access in this environment). Every exercised surface (predicated DML, row locks, `pg_stat_activity`/`pg_blocking_pids`, RLS, partial unique indexes, advisory locks, `gen_random_uuid`) is semantics-identical across PG16–PG19; DBOS v1.9.0 supports PG16. Each shell runs in a private network namespace here, so every DB run wrapped server start → tests → stop in a single invocation; data directories persisted across invocations.

## 1. Baseline verification (§3)

| # | Check | Result |
|---|---|---|
| 1 | Recompute WS-A acceptance `e20a4689…` | **VERIFIED.** `e20a4689fd37842f19fdb983db0d271f0e4fa14c153ffef3872abf885f27700f` reproduced over the frozen copy `/tmp/p15-wsb-accept-e20a4689` (291 files) using the prompt16 exclusion set (prune `.git`, `docs/evidence/digests/`, `docs/adv_review_prompt16.md`; normalized relative paths). A naive `tree` digest differs (`7ca9261c…`) solely because it includes the review file — expected, and this confirms the exclusion discipline rather than undermining it. |
| 2 | Red-tree digest `c9c08fe5…` + relationship to baseline | **VERIFIED.** `c9c08fe5f06c0a9579244e28cfd4930da3a3189575f4d9537d06e497dd2d5031` recomputed over `/tmp/m01-red`. Recursive diff accept-vs-red: exactly one delta — `runtime/m01_test.go` added. Red impl files byte-identical to acceptance (`service.go`/`store.go`/`lifecycle.go`/`store_test.go` hashes match). |
| 3 | Green digest `5afb6e46…` | **VERIFIED with one explained delta.** The manifest file hashes to `5afb6e4643dc…`; all 300 manifest entries present, 299 byte-identical. Sole mismatch: `docs/evidence/README.md` (the 5 Phase 1 rows appended post-digest as self-declared meta-records). Sole extra file in tree: `docs/prompt19.md` (this review's prompt). Current tree digest `888c6257…` is fully explained by those two deltas. |
| 4 | Full SHA-256 of `runtime/m01_test.go` | `222f5ff5a77bc5d59417d54c0922b5667e839e1901086f2e8efc11fdba3002bc` — matches the register's `222f5ff5…` exactly. |
| 5 | Same test bytes red/green | **VERIFIED.** Both copies hash identically (above). |
| 6 | Red overlay = that file only + pre-remediation impl | **VERIFIED.** Recursive diff shows only the added test file; red `service.go`/`store.go`/`lifecycle.go` byte-match acceptance (pre-remediation code: blind `update`, no choke helpers — confirmed by reading). |
| 7 | Green tree contains submitted impl, no unreported changes | **VERIFIED.** `service.go 2be8f329…`, `lifecycle.go 4a6db8e9…`, `store.go 81b69323…`, `store_test.go c9052f85…` all match the register. Implementation diff accept→green confined to the claimed files (runtime ×5 + contract + testkit/doc + CI-delta files, all previously baselined). |
| 8 | Register pre-image + appended rows | **VERIFIED.** `p17-phase1.README.md` hashes to `a721b71e…` ✓; current README = pre-image + exactly 5 appended Phase 1 rows, pure append (`100a101,105`, no in-place edits). L-04 process honored. |
| 9 | Git revision / branch / status / inventory | HEAD `982aecf`, branch `main`, 13 modified + 14 untracked (all reviewed; inventory in §8). No commits or pushes by this review. |

One baseline caveat (INFO-6): the register's overlay-digest strings (`adeb94e0…`, `408e12c0…`) and source digest (`b397df5d…`) are not reproducible from the record — `digest.sh files` output is path-string-dependent and the exact invocations were not recorded (two reconstruction attempts missed). This affects ornament strings only: per-file hashes, full-tree digests, and copy diffs — the proof-carrying artifacts — all verify independently.

## 2. M-01 verdicts

### 2.1 Per-mutation-site verdicts

| # | Mutation site | Verdict | Basis |
|---|---|---|---|
| 1 | `executeSync` terminal completion via `completeSync` (`service.go:651`, op `store.go:143`) | **PARTIAL — mechanism sound, race proof missing (HIGH-1).** Predicate (`pending`/`running`/`waiting_approval` + tenant + id) + affected-row check + loser-reload-replay is correct as written and the predicate is pinned at store level. But no test exercises the loser path or any contention on this op, and prompt17's first race case names it explicitly. The op's own comment misidentifies the threat (Cancel, which cannot address run-less Class A records — see HIGH-1). |
| 2 | `pollApproval` settle via `settlePollTerminal` (`service.go:440`, op `store.go:163`) | **VERIFIED with a bounded defect (MEDIUM-1).** Predicated settle + affected-row check + loser-reload-replay verified by lock-gate race (`TestPollSettleVsCancelRace`), idempotency, and sequential tests, all reproduced red→FAIL/green→PASS. Defect: unconditionally writes back possibly-stale `run_id`, violating CR-05 run-ID immutability under a narrow 3-way race (MEDIUM-1). |
| 3 | Status sync via `syncIfLive` (`lifecycle.go:26`, op `store.go:179`) | **VERIFIED.** Predicated sync + reload-on-loss + audit-only-on-win verified by `TestStatusSyncVsCancelRace` (reproduced both sides) and `TestFaultCompletionAudit`. Old code's unpredicated overwrite + observe-and-audit-anyway is gone (confirmed by accept-vs-green diff). |
| 4 | Cancel via `cancelIfLive` (`lifecycle.go:123`, op `store.go:193`) | **VERIFIED.** Predicated cancel + pre-check + single-audit-winner verified by `TestConcurrentCancelSingleAudit`, `TestSettleThenCancelRefused`, `TestCancelQuarantinedRefused` (all reproduced). The `run_id` write-back is provably a no-op (it always rewrites the resolved ID: `getByRun` matched it, and no concurrent path can change `run_id` while the record stays live — adopt requires empty, settle kills liveness). |
| 5 | `claim-response` hook (`service.go:180`) | **VERIFIED.** Fires post-claim-commit, pre-return, created-path only; one-shot; nil-safe; test-only arming (exhaustive grep: zero production assignments, no env/manifest/API path). `TestFaultClaimResponse` proves lost-response convergence on the real path; additionally verified FAIL on red (hook absent) via independent overlay run. |
| 6 | `update` elimination | **VERIFIED.** Zero definitions, zero callers tree-wide (production); no equivalent unconditional write under another name (all six `UPDATE app.invocations` sites in `store.go` carry predicates + affected-row checks — verified by exhaustive grep). |
| 7 | Lock order / no locks across external ops | **VERIFIED.** All four ops are single-statement transactions (at most one row lock each); `parkTx` holds one row lock while inserting new rows only (same-key parks serialize on the invocation update first); `claim` uses separate txns per step; executor calls happen strictly outside DB transactions (Cancel: op → settle → executor; Status: executor read → op; complete: handler-tx commits → op). No lock is held across any external operation. |

### 2.2 Red/green discrimination verdict: GENUINE

- Overlay identity: `m01_test.go` byte-identical both sides (`222f5ff5…` full hash verified); red copy differs from acceptance by that file alone; helpers (`x03`/`x01x02`/`service_test`) byte-identical across accept/red/green (hash-verified per file).
- Red behavior reproduced independently on a fresh DB: `TestPollSettleVsCancelRace` FAIL round 0 (`pollErr=terminal approval-denied … cancelErr=<nil> status=failed (incoherent pair)`); `TestStatusSyncVsCancelRace` FAIL (`status=succeeded cancelErr=<nil> record=cancelled` — the cancel-wins-record order; the evidence's both-orders observation is consistent with order-dependent last-writer-wins); `TestConcurrentCancelSingleAudit` FAIL (`cancel audits = 2`); all three sequential tests PASS. Each red outcome follows necessarily from red sources (blind `update` + unconditional audit), which I read directly — not from harness artifacts.
- Green: all six PASS (reproduced), plus 3× repeats of the race tests, race detector clean on `runtime`/`approvals`/`effects`, full serial suite (21 pkgs ok), `checks.sh` ALL-CHECKS-PASS — all reproduced in this environment (PG16.15 substitution recorded in §0).
- Gate methodology (prompt19 §6.5): the holder takes `FOR UPDATE` on the pre-seeded committed row; racers' first blocking statement is the predicated `UPDATE` in every gated path (claim `INSERT…ON CONFLICT` against a committed row does not block — speculative insertion waits only on in-flight inserts, of which there are none; all pre-block steps are lock-free reads, in-memory checks, or fake-executor calls — verified per path). `waitBlockedByHolder` requires statement-shape match (`app.invocations`, case-insensitive), transitive `pg_blocking_pids` chains rooted at the holder (waiter-behind-waiter safe), exact-count quorum with zero unrelated lock waiters, 30s deadline with lock dump, and release+join on every path (verified: `release()` + `wg.Wait()` precede every `Fatalf`; goroutines never call `Fatalf`). No `t.Parallel` anywhere in the affected packages, and Phase 1 verification ran `-p 1`, so the quorum cannot see sibling-test waiters.

### 2.3 Five required race cases (§6.4) vs prompt17 §4

| # | Prompt17 required case | Covering test | Verdict |
|---|---|---|---|
| 1 | Cancellation wins before sync completion commits | **NONE** — and untestable as specified (see HIGH-1) | **GAP — required proof missing** |
| 2 | Completion wins first; subsequent cancel cannot overwrite | Store-level only (`TestPredicatedOpsFromStates` terminal-immutability incl. `cancelIfLive` losing) | **PARTIAL** — predicate proven; no API-level test (folded into HIGH-1) |
| 3 | Cancel vs denial/expiry settlement | `TestPollSettleVsCancelRace` (+ `TestSettleThenCancelRefused` sequential) | **COVERED** (denial flavor exercises the shared `failIfWaiting` mechanism; expiry differs only in the approval-state observation, covered sequentially by kept `TestExpiryCancelsRun`) |
| 4 | Stale Status cannot overwrite terminal | `TestStatusSyncVsCancelRace` | **COVERED** |
| 5 | Repeated/losing idempotent-or-conflict | `TestPollSettleIdempotent` + `TestConcurrentCancelSingleAudit` + `TestSettleThenCancelRefused` + `TestCancelQuarantinedRefused` | **COVERED** |

Cases #1 (missing) and #2 (API-level missing) are the substance of HIGH-1. No cases were invented; the gap is reported as found.

## 3. Audit integrity verdict (§6.6)

**Verdict: VERIFIED with two bounded residuals (LOW-1, LOW-2).** Audit placement was traced call-by-call, not inferred from test names:

- **Status-sync settlement** (`lifecycle.go:36-81`): settlement audit fires only inside the won branch when the synced observation is terminal; the lost branch reloads and serves without auditing. Old code audited on executor observation regardless of the write outcome (confirmed by reading red source) — that defect is gone. Reproduced: green `TestStatusSyncVsCancelRace` passes with coherent outcomes; red fails as predicted.
- **Double-cancel single audit** (`lifecycle.go:123-200`): audit lives only in `cancelReport`, reachable solely via the `cancelIfLive`-won path; losers route to `redeliverCancel`, which settles and reports without auditing. Reproduced: red yields 2 audits (both blind updates succeed), green yields exactly 1 across 3 rounds.
- **Poll-settle audits**: `settlePollTerminal` audits denied/expired only on win; the cancelled outcome stays unaudited with the winner's-cancel-audit rationale documented in code. Verified the won+cancelled path is unreachable in practice (see INFO-5) — the rationale holds vacuously.
- **Loser paths emit no misleading evidence**: losers reload and replay the authoritative record; terminal replays answer recorded outcomes. Two precision residuals: (a) replay unconditionally emits a `duplicate` audit even for first-claim losers (LOW-1); (b) post-win audit failure leaves the audit permanently missing since retries answer from the terminal record without re-auditing — pre-existing pattern, now needs an X-12 sweeper scope note (LOW-2).
- Attribution: all four paths audit under the request binding's tenant/agent (never caller-supplied identity); tenant scoping enforced by RLS + explicit predicates (store-level `RejectCrossTenant[BypassRole]` tests pass).

## 4. Advisory-lock verdict (§7)

**Verdict: VERIFIED — the no-cycle proof holds as stated, with two bounded residuals (MEDIUM-3 is the prompt17 conflict, filed separately; INFO-2/INFO-3 below).**

The §4 corrected rule ("outermost-first, never while holding transactional locks") was verified against implementation, not prose:

- **Single acquirer**: exhaustive grep proves exactly one production acquisition site (`executor/classa/classa.go:41`); zero references in tests, miniapps, or any other package.
- **First-in-transaction**: `classa.Execute` opens a fresh `WithTenant` tx (BEGIN + `set_config`, neither of which takes locks — verified in `internal/store/store.go`) and the advisory `SELECT pg_advisory_xact_lock` is the first statement before any handler work. The sole caller chain (`executeSync` → `Sync.Execute`) holds no PG locks at call time (all prior ops committed; verified per caller).
- **No reverse paths**: no code path acquires PG locks and then calls `LockEntity` (only one caller, positioned first). Nested operations inside the handler (`Perform` → `CheckLive` invocation `FOR UPDATE` + effect-claim `INSERT` + lock-free `StrictPolicy` reads — lock inventory: the only `FOR UPDATE` sites tree-wide are `lifecycle.go:305` and `approvals.go:239`) all run strictly after the advisory acquisition, consistent with §4.2 order.
- **No-cycle proof (independent):** an advisory waiter holds zero PG locks (nothing precedes the acquisition that can block), so no PG waiter can ever chain through it; advisory-vs-advisory cycles are impossible because at most one advisory lock is held per transaction (single `EntityKey` per request; re-acquisition by the same session is a no-op); M-01 ops are single-statement (≤1 row lock each); `parkTx` holds one row lock while inserting new rows only, with same-key contention serialized on its own first update. No wait-for cycle is constructible. QED.
- **Test sufficiency**: no dedicated advisory test exists; the contract's stated proof basis is call-graph + existing Class A serialization and effects suites. That basis is adequate *for the current single-acquirer shape* because the safety property is structural (acquisition position), not behavioral — but it is brittle: a second acquirer or a reordered call would silently void the proof. INFO-2 recommends a mechanical guard (analyzer rule or an assertion test pinning the single call site).
- **Residuals**: (a) miniapps *could* import `internal/store` and take advisory locks directly (same module, exported function) — no mechanical bar (INFO-2); (b) `hashtext` is 32-bit — cross-entity false contention possible at scale, liveness-only, no correctness impact (INFO-3); (c) prompt17:144 still specifies advisory-LAST order for the Phase 2 authorization transaction — direct contradiction of corrected §4, must be superseded before Phase 2 (MEDIUM-3).

## 5. L-01–L-04 and O-05 dispositions

| Item | Disposition | Basis |
|---|---|---|
| L-01 claim-response hook | **VERIFIED.** Placement post-claim-commit/pre-return/created-path-only (`service.go:180`); one-shot (`FailAt`, count asserted); nil-safe (`Fire` returns nil on nil receiver); test-only arming (exhaustive grep: zero production assignments, no env/manifest/API path); `TestFaultClaimResponse` exercises the real Invoke→fault→retry path and asserts single row, single run, audit coherence (0 dispatched / 1 duplicate) — reproduced green; independently verified FAIL on red (hook absent → first invoke succeeds) via a scratch overlay run. | Code + grep + green run + red run |
| L-02/L-03 contract corrections | **VERIFIED accurate.** §12 now states dup-only detection + delimiter compat + legacy-row orphaning + the standing dual-read/backfill rule — each clause matches implementation (`000006` scans run_id dups only; `TestMigration000006DelimiterCompat` pins upgrade-succeeds; `ScopeID` injectivity independently re-proven below; no prod data per deployment evidence). `ScopeID` injectivity: verified by independent proof — `%`-first percent-encoding leaves both parts `:`-free, so the join has exactly one separator and decoding is unambiguous; prefix checks are collision-free (a longer escaped tenant can never smuggle a raw `:` after the prefix). Domain: arbitrary UTF-8 strings, no length/charset restriction; empty parts encode degenerately but injectively (callers reject empties upstream — tenant via `Caller.Validate`, entity local via `resolveEntity`). One theoretical edge (new escaped form colliding with a legacy raw row containing literal `%3A`) requires adversarial legacy bytes and no prod data exists — INFORMATIONAL, covered by the standing rule. | Code proof + migration suite reproduced (5/5) + scope unit tests reproduced (3/3) |
| Advisory contract correction | **VERIFIED accurate** (see §4). Drift-row provenance remark ("owner-approved exact wording") is adequate: the wording matches prompt19 §7's authoritative quotation and the plan approval covering the correction; suggesting an explicit cite, not disputing authorization (folded into LOW-3). | Code + docs |
| O-05 `-p 1` note | **VERIFIED.** `testkit/doc.go` states serial execution, shared-DB rationale, and the per-process-DB escape hatch — accurate, scoped, no overclaim. All Phase 1 verification in this review ran `-p 1` per the note. | Doc read + adherence |

## 6. Claim-by-claim audit (prompt19 §5)

| # | Submitted claim | Verdict | Basis |
|---|---|---|---|
| 1 | Seven mutation sites remediated (§5.1) | **VERIFIED.** All seven use predicated ops with affected-row handling; `update` has zero defs/callers; no equivalent unconditional write exists (exhaustive `UPDATE app.invocations` grep: 6 sites, all in `store.go`, all predicated). | Grep + code read + green suite |
| 2 | Transition semantics per the §5.2 table | **VERIFIED with one correction.** Six rows match implementation exactly. Exception: the table's `executeSync` source state ("running only") is narrower than the implementation (`pending`/`running`/`waiting_approval`) — and the implementation is RIGHT: undispatched pending Class A records (via `dispatchPending`) and granted waiting records could never complete otherwise. The table oversimplifies; INFO-4 records the discrepancy resolved in favor of the code. Cancel/Status/sync rows match. |
| 3 | Red/green discrimination (§5.3) | **VERIFIED.** 3 race FAILs + 3 characterization PASSes on red, all 6 PASS green — each reproduced independently with matching failure signatures; overlay = test file only; helpers byte-identical across all three trees. |
| 4 | Race coverage adequacy (§5.4) | **NOT MET — see HIGH-1.** Three of five required cases have deterministic race coverage; the `completeSync` contention case is entirely absent and the completion-wins case lacks API-level coverage. Mechanism shared-pattern argument is plausible but is not the required proof. |
| 5 | Double-cancel single audit (§5.5) | **VERIFIED.** Winner-only audit placement proven by code path analysis (losers route to audit-free `redeliverCancel`) and by `TestConcurrentCancelSingleAudit` (red: 2 audits; green: exactly 1; reproduced both, plus 3× green repeats). |
| 6 | Audit integrity (§5.6) | **VERIFIED with residuals (LOW-1, LOW-2).** No path audits a lost transition; no path duplicates a won audit; attribution server-derived; tenant-scoped. Residuals: `duplicate` label on non-duplicate loser paths; no backfill after post-win audit failure. |
| 7 | Advisory no-cycle + O-05 + L-02/L-03 (§5.7) | **VERIFIED** (see §4–§5), except the prompt17:144 contradiction (MEDIUM-3), which lives outside Phase 1's changed files but inside its safety perimeter. |

## 7. Findings

### HIGH-1 — Required race proof missing for `completeSync` contention; prompt17 case #1 unreachable as specified

- **ID:** HIGH-1. **Severity:** HIGH (blocks Phase 2 release per §17 gate "all five required concurrency cases are adequately tested").
- **Location:** Absent test (would live in `runtime/m01_test.go`); implicated code `runtime/service.go:651` (`completeSync`), `store.go:143` (`completeIfLive`), `service.go:638` (`executeSync`); requirement source prompt17 §4 bullets 1–2.
- **Governing requirement:** Prompt17 "Required regression tests" bullets 1–2 with "deterministic synchronization at the actual database transition boundary"; prompt19 §6.4 (all five cases) and §17 gate.
- **Observed implementation:** `completeIfLive` + loser-reload-replay is implemented and its predicate is pinned at store level (`TestPredicatedOpsFromStates`), but no test — race or sequential — exercises `completeSync` contention or its loser path. Exhaustive search: no test file references `completeSync`; no test races sync completion against anything; no API-level test completes-then-cancels.
- **Reproduction:** N/A (absence-of-proof finding). Corroborating structural fact, verified by exhaustive call analysis: prompt17 case #1 as specified ("Cancellation wins before synchronous completion") is unreachable via public API — `Cancel` addresses records by `run_id` (`lifecycle.go:123-128`), and Class A in-flight completions never carry one (`completeIfLive` never sets `run_id`; no path sets it on the Class A flow). The reachable analog is settle-vs-complete (lazy-expiry/deny poll racing a granted poll's `executeSync`), which is likewise untested.
- **Attack/failure scenario:** Not a demonstrated defect — the mechanism mirrors three proven paths and is likely sound. The failure mode is epistemic: the M-01 headline mutation ships its most consequential path (loser discards result, answers cancellation, emits no success audit) without ever executing it under contention. A subtle bug there (e.g., auditing on loss, answering success after loss) would survive to production.
- **Why unsafe/incorrect:** (a) A release-gate criterion explicitly requires this proof; shipping without it normalizes gate waivers. (b) The `completeIfLive` doc comment ("A concurrent Cancel wins over the completion") misidentifies the threat model — Cancel cannot address these records — indicating the path was reasoned about incorrectly, which is precisely when a test is most valuable.
- **Evidence:** file inventory greps (zero references); call-graph analysis above; prompt17 §4 text; full-suite green run proving no hidden coverage (no test fails if the loser path is broken — established by inspection, since no test reaches it).
- **Root cause:** Test-planning gap — the five-case matrix was covered 3.5/5, with the two `completeSync` cases assumed covered by analogy to the proven paths.
- **Recommended correction:** (1) Add a deterministic settle-vs-complete race test following the established gate pattern: seed `waiting_approval` + pending past-expiry approval; gate-holder locks the row; racer 1 = poll (blocks at `failIfWaiting` after computing expired — gate on 1 waiter to prove it); main thread `Decide(approve)` (different row, unblocked); racer 2 = poll (granted → channel-blocked test handler); gate on 2 waiters; release; assert exactly-once terminal coherence + single audit + discarded loser result *regardless of winner order* (same order-agnostic style as the three existing race tests — do not assert a specific winner), 3 rounds. (2) Add API-level complete-then-cancel-refused coverage (sequential suffices given the predicate proof — frame as terminal-immutability-after-completion, since Cancel cannot address run-less records). Note the race test must use a Class A gated tool (the `crm-gated` fixture, not the Class D probe tool — only the Class A granted path reaches `executeSync`) with a channel-blocked test handler fixture following the `captureStore` pattern. (3) Fix the `completeIfLive` comment to cite the settle race, not Cancel.
- **Required regression test:** The test in (1) — it must FAIL if `completeSync` audited on loss or answered success after loss (assert both explicitly), and PASS green; run it against the current tree to confirm it passes (it should — mechanism believed sound).
- **Owning package/phase:** `runtime`, WS-B Phase 1 follow-up (pre-Phase-2-gate).
- **Blocks Phase 2:** Yes — §17 gate explicitly.

### MEDIUM-1 — `failIfWaiting` overwrites `run_id` with stale values; `adoptRunID` lacks a status predicate (CR-05 violation)

- **ID:** MEDIUM-1. **Severity:** MEDIUM (bounded: no wrong terminal, no unauthorized effects, self-terminating orphan; but a stated invariant provably breaks).
- **Location:** `runtime/store.go:163-177` (`failIfWaiting` writes `run_id=$1` from the stale in-memory record); `store.go:228-244` (`adoptRunID`: no status predicate); invariant source: contract CR-05 ("a run ID is never overwritten once canonical ownership is established").
- **Governing requirement:** CR-05 run-ID immutability; prompt17 M-01 (transition integrity at all seven sites); `getByRun` routing depends on stable bindings.
- **Observed implementation:** `failIfWaiting` sets `run_id` to `rec.RunID` as read at request start. Under a lazy-expiry-vs-grant race (poll A observes expired pre-grant; `Decide(approve)` commits; poll B observes granted and `adoptRunID` sets R; A's `failIfWaiting` then wins on still-`waiting_approval` status), A overwrites R with `""`. The adopted workflow R is orphaned: unroutable via `getByRun`, uncancellable via API, running until self-termination. Symmetrically, `adoptRunID` (no status predicate) can adopt onto an already-terminal record: `grantedRun` reads `waiting_approval`/no-run, a concurrent settle wins (`failed`), then `submitChecked` reloads (failed) yet proceeds — the approval check passes on the immutable `decided/approve` linkage and nothing re-checks record liveness — adopting R and launching a workflow for a failed invocation. (The reverse order — adopt landing first — is the clobber direction above; both orders corrupt the binding, neither corrupts the verdict.)
- **Reproduction:** Code-path analysis (all three legs verified in source: stale `rec` flows unrefreshed into the write; adopt has no status gate; `getByRun` matches `run_id` only). Not executed (narrow 3-way race; no test harness exists — that absence is part of the finding).
- **Attack/failure scenario:** Operator denies/expiry-settles while an agent's grant-poll submits, with `Decide(approve)` landing mid-race. Outcome: coherent terminal verdict (failed/expired — defensible either way), but a live orphan workflow the framework can no longer address. Effects stay refused via `CheckLive` (record not live), so no unauthorized work escapes; blast radius is wasted compute + operational confusion + an unstoppable-via-API run.
- **Why unsafe:** CR-05 exists precisely so routing (`getByRun`) and orphan-prevention hold; this breaks both under concurrency. `cancelIfLive`'s identical-looking write-back is benign (it always writes the run ID it resolved, never a stale empty — verified), which makes this asymmetric defect easy to miss in review.
- **Evidence:** `store.go:163-177` vs `adoptRunID` predicate (`store.go:228-244`); CR-05 text; interleaving traced through `pollApproval`→`settlePollTerminal` and `grantedRun`→`submitRecorded`.
- **Root cause:** `run_id` treated as record payload to preserve, rather than as owned-by-adoption state excluded from settle writes.
- **Recommended correction:** (1) Remove the `run_id` write from `failIfWaiting` (settle changes status/error only; run binding is owned by adoption — no legitimate case needs settle to change it). (2) Add a live-status predicate to `adoptRunID`: `status IN ('pending','running','waiting_event','waiting_approval')` — `pending` must be included (dispatch-after-grant-restoration adopts from `pending`; verified reachable via `dispatchPending`), `quarantined`/terminal excluded. Optionally mirror with a liveness re-check in `submitChecked` after reload (defense in depth; the store predicate is the load-bearing fix). (3) Deterministic regression test: holder-gated poll blocked at `failIfWaiting` while the test thread adopts R directly (ordering-forcing variant, no sleeps), release, assert `run_id=R` preserved + status failed; plus an adopt-onto-terminal refusal test (seed `failed` + empty run, attempt adopt, assert refusal). Both orders must preserve R.
- **Required regression test:** As above, failing on current code (clobber observed), passing after fix; plus an adopt-onto-terminal refusal test.
- **Owning package/phase:** `runtime`, WS-B Phase 1 follow-up (pre-Phase-2; the fix is two predicates + tests).
- **Blocks Phase 2:** Yes — CR-05 is load-bearing for Phase 2's authorization work (routing + orphan-freedom assumptions).

### MEDIUM-2 — Concurrent granted polls (and same-key dispatches) double-execute Class A handlers

- **ID:** MEDIUM-2. **Severity:** MEDIUM (bounded: ledger-protected effects stay safe via op contention; direct-SQL handler writes can double-apply; violates the documented no-re-execution guarantee).
- **Location:** `runtime/service.go:638` (`executeSync`, no mutual exclusion) reached from `pollApproval` granted branch (`service.go:398-405`) and `dispatch` (`service.go:499-511`); violated guarantee: `replay` doc comment (`service.go:189-192`, "answers a repeated idempotency key from the record without re-executing").
- **Governing requirement:** Idempotency-key semantics (same key → single execution; replay answers from record); M-01 transition integrity extended to execution (prompt17 gate context).
- **Observed implementation:** Two polls observing the same granted approval both proceed to `executeSync` — no interlock exists between the approval observation and handler execution. Both handlers run to completion in separate transactions; only the *record* update is single-winner (`completeIfLive` predicate). Same shape for two concurrent same-key invokes on a `pending` record (claim winner + replay loser both dispatch). Pre-existing (WS-A and earlier had it worse: double update too); M-01 narrowed it to double-execute-single-record but did not close it.
- **Reproduction:** Code-path analysis (both paths traced end-to-end; no lock, predicate, or single-flight between observation and `handler()` invocation at `classa.go:45`).
- **Attack/failure scenario:** Aggressive agent retries (or two pollers) on a granted approval-gated Class A tool with direct-SQL side effects (e.g., ledger-external writes): the side effect applies twice while the record shows one success. Ledger-routed effects are protected by op-level contention; non-ledger writes are not.
- **Why unsafe:** The framework advertises idempotent invocation; duplicate execution with partial protection (ledger yes, direct writes no) is the worst combination — silent, data-dependent, and invisible to the record (single success terminal).
- **Evidence:** Call-graph analysis above; current-test-suite gap (no test invokes the same granted approval concurrently and counts handler executions — verified by inventory).
- **Root cause:** Execution mutual exclusion was never designed; M-01 scoped to record transitions only.
- **Recommended correction:** Gate granted Class A execution on winning a predicated execution-claim transition (e.g., `waiting_approval`→`running` claim before handler runs; losers reload+replay), or — if the owner prefers — explicitly downgrade the guarantee to at-least-once execution with a handler-idempotency requirement, stated in the contract. Either way, add the deterministic concurrent-granted-poll test (channel-blocked test handler; assert single handler run + coherent record).
- **Required regression test:** As above — must FAIL on current code (handler runs twice) and pass after the fix.
- **Owning package/phase:** `runtime`, WS-B (Phase 1 follow-up if the claim-transition fix is chosen; contract amendment + handler guidance if the disclosure path is chosen — either needs owner disposition).
- **Blocks Phase 2:** Yes — Phase 2 builds effect authorization on invocation lifecycle assumptions; execution cardinality must be settled first.

### MEDIUM-3 — Prompt17:144 lock order contradicts corrected §4 (advisory-last vs advisory-first)

- **ID:** MEDIUM-3. **Severity:** MEDIUM (no code impact today — Phase 2 unstarted; direct hazard to the next phase's design).
- **Location:** `docs/prompt17.md:144` ("`… → effect → advisory lock`") vs `docs/phase1/invocation-lifecycle-contract.md` §4 (`§4`, lines 239-248: outermost-first, never while holding transactional locks).
- **Governing requirement:** Contract §4 (corrected, verified against implementation in §4 of this report); prompt15.2 §5 lock-order discipline.
- **Observed:** The binding WS-B work order specifies the authorization transaction's lock order ending with the advisory lock — the exact order the §4 correction forbids (acquiring advisory while holding PG locks enables the deadlock cycle the correction eliminates). Additionally, the target M3 transaction takes no advisory lock at all, so the line is both stale-directioned and inapplicable.
- **Reproduction:** Documentary (both passages quoted; contradiction is direct, not interpretive).
- **Why unsafe:** Phase 2 implements from prompt17. Followed literally, it builds a deadlock-prone authorization transaction. The longer the contradiction stands, the more likely it propagates into Phase 2 design docs.
- **Evidence:** The two passages; §4 implementation verification in this report §4.
- **Root cause:** Prompt17 frozen before the §4 correction landed; no supersession pass run.
- **Recommended correction:** Strike or amend prompt17:144 (and the matching quote in `prompt17_plan_review.md:144`) to reference corrected §4 as authoritative, with a one-line note that the M3 transaction takes no advisory lock. No code change.
- **Required regression test:** N/A (documentation). Verification: grep shows no remaining advisory-last ordering statements in governing docs.
- **Owning phase:** Docs, before Phase 2 design begins.
- **Blocks Phase 2:** Yes — as a documentation prerequisite, not a code gate.

### LOW-1 — `replay` emits `duplicate` audit on non-duplicate loser paths

- **ID:** LOW-1. **Severity:** LOW (best-effort label precision; terminal answers remain truthful).
- **Location:** `runtime/service.go:193-194` (unconditional `duplicate` audit at `replay` entry), reached from loser paths `completeSync` (`service.go:657-663`), `settlePollTerminal`, `syncIfLive`-loss, `cancelIfLive`-loss.
- **Defect:** A first-claim caller whose completion loses a race gets a `duplicate` audit though the key was invoked once. Forensics could misread it as key reuse.
- **Recommended correction:** Pass a `duplicate bool` (true only when `claim` returned existing) and emit the marker conditionally. Test: losing first-claim completion emits no `duplicate` marker.
- **Blocks Phase 2:** No (advisory; fold into follow-up).

### LOW-2 — No audit backfill on post-win audit failure

- **ID:** LOW-2. **Severity:** LOW (narrow: requires audit-store failure with healthy transition path; pre-existing pattern).
- **Location:** `completeSync` success branch (`service.go:666-670`), Status settlement (`lifecycle.go:55-61`), `cancelReport` (`lifecycle.go:191-200`).
- **Defect:** Win committed + audit fails → retryable; retry answers from the terminal record without re-auditing, so the settlement/cancel/completion audit is permanently missing. Pre-existing shape, now load-bearing for three more paths.
- **Recommended correction:** Scope X-12 sweeper explicitly to backfill missing settlement audits on settled records (or document best-effort acceptance). No Phase 1 code change required.
- **Blocks Phase 2:** No, but the X-12 scope note must exist before Phase 2 closes audit work.

### LOW-3 — Contract phasing precision (§4.2 "one tx", §9 wording, drift provenance)

- **ID:** LOW-3. **Severity:** LOW (documentation only; behaviors verified correct).
- **Location:** Contract §4.2 Cancel row ("one tx" — true only post-X-05; current implementation is correctly multi-tx); §9 `claim-response` row ("pre-dispatch" — also pre-park); §16 "owner-approved exact wording" remark (adequate via prompt19 §7 quotation + plan approval, but no standalone cite).
- **Defect:** Three precision nits that could mislead a Phase 2 implementer reading the contract cold.
- **Recommended correction:** Phase markers on future-state §4.2 rows ("(X-05)"); "pre-dispatch/park" wording; one explicit cite for the §4 wording approval.
- **Blocks Phase 2:** No.

### INFO-1 — `live` set never pinned; implementation excludes `pending`

- The contract uses `status IN (live)` without a one-line definition (CR-05 defines nonterminal *including* `pending`). Implementation excludes `pending` from `cancelIfLive`/`syncIfLive`/check-live while including it in `completeIfLive`. Each choice verified correct in context (pending records carry no runs, so Cancel/Status cannot address them — `getByRun`/`RunID` gating; completion must accept pending for undispatched Class A). Recommend a one-line `live` definition to prevent future drift. No behavior change.

### INFO-2 — No mechanical bar on miniapp direct advisory acquisition

- The no-cycle proof assumes miniapps never call `store.LockEntity` directly (same module, exported function). True today (exhaustive grep: zero non-`classa` references); unenforced tomorrow. Recommend an analyzer rule or an assertion test pinning the single call site. No current violation.

### INFO-3 — `hashtext` is 32-bit

- Cross-entity false contention possible at scale; liveness-only, zero correctness impact (false sharing serializes, never corrupts). Consider `hashtextextended` if contention is ever observed. No action.

### INFO-4 — Prompt19 §5.2 table vs implementation on completion source states

- The review-protocol table says "running only"; the implementation accepts `pending`/`running`/`waiting_approval` — and the implementation is right (undispatched pending and granted waiting Class A records must complete). Discrepancy resolved in favor of the code; recorded so the table is not cited against the implementation later.

### INFO-5 — Unreachable defensive paths with misleading taxonomy

- `settlePollTerminal` won+`cancelled` branch is unreachable (CancelPending runs only post-cancel-win; cancelled is terminal-immutable — verified). `replay` on `quarantined` falls to `default` → `terminal("internal", …)`; no producer writes `quarantined` today (verified: zero writers). Both are harmless defense-in-depth; flagging so X-10/quarantine work assigns them real semantics (especially the `internal` label, which would misreport a legitimate operator state).

### INFO-6 — Unreproducible digest strings (baseline caveat, no action on code)

- Register overlay/source digest strings lack recorded computation methods (see §1). Per-file and full-tree verification is complete regardless. Future runs should log exact digest commands.

### INFO-7 — Pending/Class-A records uncancellable via API; trust-float asymmetry

- `Cancel` addresses records by `run_id`; pending and Class A records never carry one, so they cannot be cancelled through the API (verified by exhaustive path analysis). Plausibly intended (nothing to stop), but unstated — recommend one contract line. Related: `submitChecked` re-checks persisted trust while `executeSync` honors the fresh decision, so a confirm→autonomous trust flip-flop executes Class A without approval linkage but is refused (fail-closed `errApprovalRequired`) on the Class D path. Safe in both directions (grant-capped; verified no privilege escalation), but asymmetric — recommend documenting the effective-trust-floats rule or symmetric re-check.

## 8. Phase scope recommendation

**Phase 2 is NOT released by this review.** The mechanism delivered in Phase 1 is sound and well-proven where proven — but the release gate of prompt19 §17 is explicit ("all five required concurrency cases are adequately tested"), and it is not met: HIGH-1 (missing `completeSync` race proof + requirement rescope) blocks, and MEDIUM-1 (CR-05 run-ID write coherence), MEDIUM-2 (execution mutual exclusion), MEDIUM-3 (prompt17 lock-order contradiction) must be resolved first lest Phase 2 build on broken assumptions.

Recommended close-out sequence (all bounded, no redesign):

1. **HIGH-1:** add the deterministic settle-vs-complete race test + API-level complete-then-cancel coverage; correct the `completeIfLive` comment; obtain owner disposition rescoping prompt17 case #1 (Cancel-vs-completion is unreachable as specified — replace with the settle race or document why Cancel must become able to address these records).
2. **MEDIUM-1:** drop the `run_id` write from `failIfWaiting`; add a live-status predicate to `adoptRunID`; add the deterministic clobber regression test (specified in the finding).
3. **MEDIUM-2:** owner disposition first (predicated execution-claim vs documented at-least-once + handler-idempotency requirement), then implement + channel-blocked-handler regression test.
4. **MEDIUM-3:** strike/amend prompt17:144 (+ plan-review mirror) to reference corrected §4; verify by grep.
5. **LOW-1/LOW-2/LOW-3:** fold into the same close-out (all one-line-to-small changes + tests/notes).
6. Re-run: red overlay (must still FAIL identically — proving the new tests discriminate), green full suite + race + checks.sh; update the register with a close-out row citing this review.

On completion of 1–6 with a clean re-verification of the touched areas only (no full re-review needed), Phase 2 may proceed. Nothing in this review questions the WS-A baseline, the predicated-transition design, the lock-gate methodology, or the evidence discipline — all verified genuine.

## 9. Residual risks

- **Requirement-implementation drift on `live` states (INFO-1).** The `pending` exclusion is correct but undocumented at the contract level; a future contributor could "fix" it into a bug. One-line definition closes this permanently.
- **Execution cardinality (MEDIUM-2).** Until closed, concurrent duplicate requests can double-execute Class A handlers. Ledger effects are safe; direct-SQL writes are not. Any new Class A tool with direct writes inherits this exposure in the interim.
- **Run-binding races (MEDIUM-1).** Narrow but real; orphaned runs self-terminate with effects refused, so the blast radius is wasted compute plus operational confusion — until fixed.
- **Phase 2 design inputs (MEDIUM-3).** The prompt17 lock-order line is a live trap for the authorization-transaction design. Resolve before any Phase 2 design doc cites lock order.
- **Audit completeness under partial failure (LOW-2).** Settlement audits can go permanently missing when the audit write fails post-win; acceptable only with the X-12 backfill scope explicitly assigned.
- **Provenance hygiene (INFO-6/L-04 lineage).** Digest strings without recorded methods, and the pre-existing register in-place edit, are process debts. The discipline otherwise held (pre-image + pure append verified).
- **Environmental substitution.** All execution in this review ran on PG16.15 (evidence: PG19.0 Docker). The exercised surface is version-independent SQL/locking semantics; a PG19 confirmation run of the three race tests would close this formally at negligible cost and is recommended but not required.
