# Prompt 16 — Independent Adversarial Review of WS-A Exit (Prompt 15.2 Phase A)

- **Role:** independent adversarial reviewer (Plan §1 posture). Read-only except this report; no fixes, no WS-B, no commits.
- **Governing spec:** `docs/prompt15.2.md` + `docs/phase1/invocation-lifecycle-contract.md` + `docs/prompt15.2_p2.md` + owner plan `docs/wsc_exit_a_review_plan.md`.
- **Implementation under review:** WS-A (prompt14 findings X-01–X-04, X-19, X-22 + Rule 3 amendment + entry-path wait proofs).
- **Review date:** 2026-11-23. Report: `docs/adv_review_prompt16.md` (the only file added).
- **Verdict: CLEAR — WS-A is accepted as the foundation for WS-B.** No CRITICAL or HIGH finding remains. One MEDIUM, six LOW findings, and four carry-forward observations are recorded below; none blocks WS-B. (§10 condition-by-condition justification at the end.)

## 1. Baseline freeze (Plan §1–§2)

**Submitted green tree.** Full digest `f0b7dfa6458b5bc5e41816a3a6eb9565447bd4b3d20bf27e6c6656f37d820932` — reproduced by hashing the committed manifest file `docs/evidence/digests/p15-wsaexit-2026-11-23.manifest.txt` (290 entries). HEAD at review: `5cf9627` (branch `main`).

**Red snapshot.** Digest `893df60f36a290b7d791e514f2d3f8830f82beae16dd9b12a7d40f36029ad918` — reproduced twice: (a) hashing the committed `p15-p2open` manifest, (b) `scripts/digest.sh tree /tmp/p15-red-893df60f36a2` on the pristine snapshot copy. The snapshot is byte-identical to its recorded digest: pristine, confirmed.

**Current tree vs submitted baseline.** Current tree digest is `e20a4689fd37842f19fdb983db0d271f0e4fa14c153ffef3872abf885f27700f`. The delta to `f0b7dfa6…` is exactly two items, both explained, neither touching reviewed behavior:
1. `docs/evidence/README.md` — differs by more than pure append (proven: no line-boundary prefix/excision of the current file reproduces the submitted hash). Rows 94 (WS-A-exit meta, self-declared post-digest) and 95 (P1 contract note) postdate submission; at least one further in-place edit exists within rows 1–93 whose original bytes are unrecoverable (no pre-image retained). All load-bearing WS-A row contents were independently verified true (hashes, outcomes, commands). Recorded as process finding L-04, not a correctness threat.
2. `docs/prompt16.md` — this review's prompt, added post-submission (expected). No other added/changed/missing files (290/290 manifest entries present, verified by recomputation).

All verification runs below executed against this tree (= submitted baseline + the two explained deltas). The single review-created scratch test used for one live probe was created, run, and deleted; post-removal tree digest equals the pre-probe digest (`e20a4689…`), verified. (`e20a4689…` predates this report file; §6 pins the acceptance file set with the report accounted for.)

**Overlay provenance.** Per-finding red copies verified: `/tmp/p15-redrun-x03b` differs from the pristine snapshot ONLY by `runtime/x01x02_test.go` + `runtime/x03_test.go`; `/tmp/p15-redrun-x04b` ONLY by `approvals/x04_test.go`; `/tmp/p15-redrun-x01x02` ONLY by `runtime/x01x02_test.go` (all via recursive diff). Overlay test bytes are identical to the current tree (`x01x02 8b89c172…`, `x03 f43b4b17…`, `x04 0e8273ac…` — all match the register and `sha256sum` of current files). The older `/tmp/p15-redrun-x03` + `/tmp/p15-redrun-x04` hold the superseded barrier generation (`adc6b578…`, `92a3acd3…`, matching register rows 88–89). A fresh overlay rebuilt by this review (snapshot + current test files) discriminates identically (both FAIL). The register's overlay-digest strings (`adeb94e0…`, `408e12c0…`) and source digest (`b397df5d…`) are not reproducible from the record (computation method undocumented; `digest.sh files` is path-string-dependent) — recorded as observation O-02; per-file hashes and copy diffs carry the actual proof and all verify.

**Environmental deltas.** No Docker daemon, no Neon access, no `TEST_*` URLs in this environment; PostgreSQL server binaries absent (client-only distro tools). This review procured stock Ubuntu PostgreSQL 16.15 via `apt-get download` (no root; user-space extract to `/tmp`), ran a disposable TCP-only cluster (per-invocation lifecycle: the sandbox gives each shell its own network namespace, so server start → tests → stop were wrapped in single invocations; data dir persisted across invocations). Databases `miniapp_dev` / `miniapp_red` recreated disposable. **All execution results below are on PG16.15; the evidence used PG19.0.** The exercised surface (predicated DML, row locks, `pg_stat_activity`/`pg_blocking_pids`, RLS, partial unique indexes, advisory locks, LISTEN-free flows, `gen_random_uuid`) is semantics-identical across PG16–PG19, and DBOS v1.9.0 supports PG16; the substitution is recorded here, not hidden. Go 1.27.3 toolchain with `GOWORK=off GOTOOLCHAIN=local` (parent `go.work` would otherwise capture the module) and the pre-seeded module cache.

## 2. Findings

Severity scale per prompt16 §10: CRITICAL (blocks, exploitable/loss), HIGH (blocks, invariant broken), MEDIUM (must fix, bounded), LOW (fix or record). No CRITICAL or HIGH findings. Findings marked [WS-B+] are carry-forward items owned by later workstreams, recorded here only to prevent silent loss.

### M-01 — Raw `store.update` remnants have no written exception (MEDIUM)

**Location:** `runtime/service.go` — `executeSync` terminal updates, `pollApproval` failed-marking (3 sites, errors discarded); `runtime/lifecycle.go` — `Status` sync update. **Contract:** §6 mandates predicated per-transition operations for every durable lifecycle mutation ("Any exception must have a written explanation and, where relevant, a test").

**Defect.** These mutations still use the unpredicated `store.update`:
- `executeSync` writes `RunFailed`/`RunSucceeded` unconditionally. A `Cancel` landing between handler completion and the update is clobbered: the invocation reports succeeded/failed though cancelled (both audit events exist, so evidence is preserved but the final status lies).
- `pollApproval` marks `RunFailed` unconditionally (denied/expired), clobbering a concurrent `cancelled` marking. Terminal-over-terminal; audits preserved.
- `Status` sync update likewise unpredicated (benign direction: it only writes observed executor truth).

None has the contract-required written exception. (Cancel's own update is specified predicated under M6/WS-B — correctly future, not part of this finding.)

**Why MEDIUM, not HIGH:** window is narrow (in-request for Class A), impact is status-label truthfulness with full audit evidence retained, no unauthorized execution or data loss is reachable through any of these sites (verified by tracing each caller: all write terminal states consistent with an observed outcome).

**Resolution:** convert to predicated transitions (e.g., only-from-live for completion; only-from-waiting for poll settlement) with affected-row handling, or record written exceptions per §6. Suggested owner: WS-B (it owns M6/settleTerminal and the lifecycle-mutation sweep will touch the same lines anyway).

**Severity clarification (owner-directed, 2026-11-23; see §6).** Two statements must be kept distinct, and the original report blurred them: (1) As a normative matter, M-01 violates the contract's lifecycle-mutation requirement (§6: predicated transitions or a written exception — neither holds), which meets the letter of the HIGH rubric wording ("violations of normative lifecycle or concurrency guarantees"). A terminal-state overwrite is documented, and the report must not be read as claiming no invariant is implicated. (2) As an impact matter, the race is narrow, audit evidence is fully retained, and no unauthorized execution or data loss is reachable — bounded enough to be non-blocking for WS-A acceptance. **Owner disposition:** WS-A remains the accepted baseline AND M-01 is a **mandatory** WS-B lifecycle-correctness item (Priority 1 intake), not optional cleanup. The MEDIUM label is retained for identifier stability; its normative force is "contract violation, must fix in WS-B," not "fix at leisure."

### L-01 — Contract `claim-response` op specified but unimplemented and untested (LOW)

**Location:** contract §6 op table (row: "`claim-response` | `claim` post-INSERT, pre-return (WS-A, new)") vs implementation (no `Fire("claim-response")` anywhere — verified by exhaustive grep; no dedicated fault test).

**Defect.** The contract names 20 fault ops (the table gained `claim-response` during WS-A without a §13 drift entry), but only 19 exist in code. The *behavior* the op would pin — lost-success claim response returns the prior handle, never a second run — is covered by pre-existing replay tests (`TestInvokeIdempotentReplay` and the X-01 replay paths), so nothing behavioral is missing; what is missing is the named hook, its dedicated fault test, and the drift-table record of the addition.

**Resolution:** either implement the hook + `TestFaultClaimResponse` (one-shot, assert replay-converges + single run + audit coherence), or amend the contract to drop the row with a §13 entry citing the covering replay tests. Either closes it; the current state (specified-but-absent) is the defect.

### L-02 — Contract §12 overclaims migration delimiter/escape scanning (LOW)

**Location:** contract §12 ("The migration preflight scans identity fields for `%/:` ... check escape collisions ... on detection the migration fails") vs `migrations/000006_*.up.sql` (scans duplicate nonempty `run_id` ONLY) and `TestMigration000006DelimiterCompat` (pins upgrade-SUCCEEDS-with-delimiters).

**Defect.** The migration deliberately does not scan delimiters and cannot "detect" escape collisions — correctly so: `ScopeID` is provably injective (verified: `%`-first percent-encoding makes both parts `:`-free, so the join has exactly one separator; independently re-proven in §4), making collisions impossible, and delimiter-bearing legacy rows coexist compatibly (proven by the DelimiterCompat test). There is no production data to migrate. The contract sentence describes a reject-on-delimiter design that was replaced by delimiter-compat without updating §12 or §13.

**Resolution:** rewrite §12 to state actual behavior (run_id-dup detection + delimiter compat + no-prod-data premise). No code change needed.

### L-03 — Legacy raw-ID rows orphan on re-attempt; contract says lookups "keep working" (LOW)

**Location:** contract §12 ("persisted `BusinessOpID` lookups keep working") vs the encoding cutover (new writes escaped via `ScopeID`; legacy rows raw).

**Defect.** A legacy raw multi-colon op ID re-attempted post-upgrade computes a different (escaped) PK, misses the legacy row, and inserts a fresh row — a re-send across the upgrade boundary. Reachability today: nil in practice (test DBs only, no prod data; old invocations are terminal so their ops are never re-attempted; verified no prod data exists per §16.1 premise). The contract sentence is strictly false and the cutover needs one explicit paragraph: legacy rows are never matched again (orphaned, not migrated), which is safe only because no production rows exist.

**Resolution:** correct §12 with the orphaning statement + a standing rule: any future encoding change with production data present requires dual-read or backfill. No code change for WS-A.

### L-04 — Evidence register edited in place post-submission; original bytes unrecoverable (LOW, process)

**Location:** `docs/evidence/README.md` (differs from the submitted manifest hash by more than rows 94–95; proven: no line excision reproduces the submitted hash).

**Defect.** At least one in-place edit within rows 1–93 occurred after the digest, and no pre-image was retained, so the edit itself cannot be identified. All load-bearing WS-A row contents were independently verified true by this review (test-file hashes match byte-for-byte, red/green outcomes reproduced, commands match), so no WS-A claim depends on the unedited-register assumption. The failure is purely procedural: an append-only register must not have unidentifiable edits.

**Resolution:** retain manifest-generation-time copies of the register going forward (or commit it); treat the current register as accurate-but-unverifiable-as-append-only. No WS-A content action needed.

### L-05 — Trust-mismatch retry returns unclassified `errApprovalRequired` (LOW)

**Location:** `runtime/service.go` `submitChecked` → `dispatch` → `Invoke` (error escapes raw, neither terminal nor retryable).

**Defect (proven live with a throwaway probe, since removed; tree digest verified unchanged after removal).** A pending confirm-trust record retried as autonomous (grant permitting) reaches `dispatch` → `submitRecorded` → choke refusal, which propagates as bare `runtime: approval linkage required before submit`. Fail-closed (verified: no submit, record stays pending, retry-as-confirm converges), but adapters cannot classify it — it will surface as an internal error rather than a retryable signal, and the record sits pending until retried with original trust.

**Resolution:** classify (retryable is correct — a confirm retry converges) or route `dispatch`'s choke refusal back into approval handling. One test (the probe's shape) pins it.

### L-06 — `dispatchPending` denial emits no audit event (LOW)

**Location:** `runtime/service.go` `dispatchPending` deny branch (returns terminal `forbidden` without audit) vs `Invoke`'s authorize-deny path (audits `forbidden`).

**Defect.** First-request denials are audited; retry-after-grant-revocation denials (pending record + revoked grant + retry) are silent. Reachable, narrow, attribution-only impact: the terminal error reaches the caller correctly; only the audit trail lacks the event.

**Resolution:** audit the denial (same `forbidden` shape as the Invoke path) + assert it in a revocation-retry test. Natural WS-B companion (it owns grant-revocation semantics).

### O-01 — Submit-time linkage TOCTOU is contained, not closed (OBSERVATION, residual)

`submitChecked` validates the linkage, then submits; revocation/denial/expiry landing in between yields a run bound to a dead linkage. Containment (verified by reading + wait-proof tests): the run enters only the durable wait path; the workflow re-reads the authoritative table on wake; poll convergence settles the invocation; WS-B's in-transaction effect revalidation (specified, owner-approved) is the backstop that makes wait-path entry safe. The choke's refusal of check-time-dead linkages is complete (pending/granted-only + hash + expiry + tenant/invocation/tool binding all re-read from store). No action for WS-A; WS-B must not weaken the revalidation list.

### O-02 — Unreproducible digest strings (OBSERVATION, provenance)

The register's overlay digests (`adeb94e0…`, `408e12c0…`) and source digest (`b397df5d…`) cannot be reproduced from the record (computation inputs undocumented; `digest.sh files` is path-string-dependent — two attempted reconstructions both miss). This does not weaken any claim: per-file hashes, full-tree digests, and copy diffs — all independently verified — carry the proof. Recommendation: record exact digest commands (with CWD and path forms) or drop the ornament strings.

### O-03 — Future-scope scaffolding present in the WS-A tree (OBSERVATION, scope transparency)

The submitted tree contains behavior-neutral scaffolding beyond strict WS-A scope: WS-B hooks (`effect-reserve`, `takeover-begin`), Phase-4 hooks (`settle-commit`, `settle-response`, `settle-operator-commit`), WS-J's `TenantID` stamp on `effects.Operation`, and the `executor-shutdown` hook. All verified nil-safe (nil injector never fires; no production assignments — exhaustive grep), all covered by their own fault tests, none reachable in production behavior. The WS-A exit row should have noted the tree is "WS-A plus inert future hooks" rather than pure WS-A. No action; WS-B+ must treat these as untested-in-integration until their workstreams land (hooks are unit-proven, not path-proven).

### O-04 — `attached_run_id` column unwired; early-return dormant but correct (OBSERVATION)

No production code writes a non-`''` `attached_run_id` (verified by exhaustive grep — column is R1 scaffolding for WS-E true attach). `submitChecked`'s attached early-return and `adoptRunID`'s `attached_run_id = ''` predicate are therefore dormant-but-correct forward guards. The X-03 "adoption" proven here is run-ID adoption within one invocation, not cross-invocation attach — the register's wording ("adoption by the loser") is accurate as written. WS-E must wire writers + resolution (`Status`/`Signal`/`Cancel` do not resolve attached IDs today).

### O-05 — Parallel test packages against one shared DB hang; serial (`-p 1`) is the supported invocation (OBSERVATION)

This review initially ran `go test ./runtime/ ./approvals/ -race` WITHOUT `-p 1`: the `approvals` package hung to the 600s `go test` timeout (same class of error as an earlier parallel full-suite run that failed fast with cross-talk symptoms). Re-run serially per the evidence methodology: both packages pass race-clean in ~7s. Root cause class: separate test binaries sharing one database (open gate-holder transactions vs cross-package `TRUNCATE`, which needs `ACCESS EXCLUSIVE`), outside the documented practice (`testkit`: "one database per process run"; evidence commands all `-p 1`). Not a product defect and not a WS-A finding — but the hang (rather than fast failure) on misuse is worth one line in `testkit`'s doc comment stating the serial-packages requirement explicitly.

### Rule 3 / §4.6 boundary statement (normative, for WS-B handoff)

WS-A proves: gated workflows are submitted ONLY with a live persisted linkage (choke, single `Async.Submit` site verified by exhaustive call-graph grep; two deeper fail-closed layers: §5.6 executor gate + workflow `translate` gate, both tested); submitted runs enter the durable wait live and send nothing while pending (E2E wait proofs with real DBOS); replay/poll/decision paths obey the same permission (waitproof + fault + X-01/X-02 suites). WS-A does NOT prove — by owner-approved design, not by omission — effect-time grant revalidation: the workflow-side re-read is miniapp code (`ReadApprovalDecision` in the probe), and a future miniapp that skips re-read proceeds on signal alone until WS-B's in-transaction authorization boundary lands. The wait-vs-execution distinction is the load-bearing handoff: nothing in WS-A authorizes consequential execution, and nothing in WS-A may be cited for it.

## 3. Verification log (what was run, where, result)

All DB runs on disposable PG16.15 (see §1 environmental note), `TEST_DATABASE_URL`/`TEST_DBOS_DATABASE_URL` pointed per run, `go test -p 1` for suites unless noted. "Reproduced" = this review executed it and observed the outcome.

| # | Run | Location / DB | Result |
|---|---|---|---|
| V1 | `gofmt -l .`, `go vet ./...`, `go build ./...` | current tree | clean / clean / exit 0 |
| V2 | `bash scripts/checks.sh` | current tree | ALL-CHECKS-PASS (incl. analyzer dogfood + live preflight) |
| V3 | X-01/X-02 suite (7 tests) green | tree / miniapp_dev | 7/7 PASS |
| V4 | X-01/X-02 suite red | `/tmp/p15-redrun-x01x02` / miniapp_red | 6 FAIL with predicted messages (bypass, unbound submit, status-unavailable loop, signal-failed, 2× cancel-failed loops) + 1 documented vacuous PASS |
| V5 | `TestConcurrentSubmitSameKey` green, incl. `-race`, incl. 3× repeat | tree / miniapp_dev | PASS (3 rounds), race-clean, 3/3 |
| V6 | Same test red | `/tmp/p15-redrun-x03b` + fresh self-built overlay copy / miniapp_red | FAIL round 0, divergent handles (orphan) — both copies |
| V7 | X-04 suite (4 tests) green, incl. `-race`, incl. 3× repeat | tree / miniapp_dev | 4/4 PASS, race-clean, 3/3 |
| V8 | X-04 suite red | `/tmp/p15-redrun-x04b` / miniapp_red | ApproveReject FAIL round 0 (2 successes); DecideAndCancel FAIL round 1 (both won — order-dependent, near-certain over 20 rounds); 2 documented guard PASSes |
| V9 | Fault suites (runtime 14, approvals 5, effects, executor) | tree / miniapp_dev | all PASS |
| V10 | ScopeID unit tests (3) | tree | PASS |
| V11 | Migration suite (fresh/dup-diagnostic/unique/down-up/delimiter) | tree / scratch DBs | 5/5 PASS |
| V12 | E2E wait proofs (2, real DBOS) | tree / miniapp_dev | 2/2 PASS |
| V13 | Empty-runID rejection (2) + waitproof (2) | tree / miniapp_dev | 4/4 PASS |
| V14 | Full suite `go test -p 1 ./...` | tree / miniapp_dev | 21 pkgs ok, 0 fail (matches evidence) |
| V15 | Full `runtime`+`approvals` with `-race`, serial (`-p 1`) | tree / miniapp_dev | PASS, race-clean (4.8s + 2.0s) |
| V16 | Scratch trust-mismatch probe (throwaway, removed; tree digest verified unchanged after) | tree / miniapp_dev | fail-closed confirmed; bare `errApprovalRequired` (L-05) |
| V17 | Index-schema proof (`CREATE INDEX` unqualified → table's schema) | direct SQL | `idxdemo` — confirms migration NOTE; refutes prompt14 nit (see §5) |
| V18 | Red-defect presence | red snapshot source | confirmed: blind `confirmSubmit`, unpredicated run-ID overwrite, lockless `Decide`, no choke/adopt/park primitives |

Red/green discrimination is therefore reproduced for X-01 (3 triggers), X-02 (4 paths), X-03 (lock-gate), X-04 (2 discriminating tests + 2 honest guards) with byte-identical test sources both sides. The two pre-existing red failures that look odd on first sight (V4's PASS, V8's two PASSes) are each documented in-test as vacuous/guard with the reason stated — verified accurate, not weasel-wording.

## 4. Scope conclusions (prompt16 §4–§5)

**X-01 approval bypass — CLOSED.** The choke (`submitChecked`) refuses every gated submission without a live persisted linkage; legacy `running` records re-enter approval handling (park or adopt, never blind submit); three triggers proven (fault-injected approval-create/park failures + SQL crash fixture); E2E replay proof with real DBOS. Two deeper layers (executor §5.6 gate, workflow `translate` gate) independently refuse unlinked gated runs. Residuals: L-05 (error classification), O-01 (submit-time TOCTOU contained).

**X-02 stranding — CLOSED.** Submit-if-missing on both recovery paths (`grantedRun` for polls, `DecideApproval` for the approve-with-never-submitted race), run-less denial/expiry absorb via unknown-run-verified `cancelRun`, 7 green tests + 6 red FAILs all reproduced. Note for WS-B: `DecideApproval`'s in-call recovery currently activates only with X-06-fixed `Signal` semantics (real `Send` buffers without error); convergence via `grantedRun` poll is what actually fires today — both paths exist and the end state converges either way (verified by tracing both orders).

**X-03 same-key race — CLOSED.** Predicated `adoptRunID` (`run_id = '' AND attached_run_id = ''` + affected-row check + reload) plus the DB partial unique index as backstop; the lock-gate test genuinely forces both racers to observe-empty-then-block (verified: holder `FOR UPDATE`, transitive `pg_blocking_pids` chains, statement-shape match, exact-count quorum, release+join on all paths including gate timeout; teardown audited for wedges). Red FAILs deterministically with divergent handles; green shows shared handle + 1 workflow; race-clean; 3× stable. Loser adopts via idempotent re-submit (fake fidelity to SDK reuse semantics verified by inspection; real-executor path covered by grantedRun/Status tests).

**X-04 Decide races — CLOSED.** `FOR UPDATE` + predicated UPDATE + reconcile-after-loss gives single-winner semantics for approve/reject, Decide/cancel (both orders consistent), self-approval (deterministic refusal — immutable-identity argument verified), and expiry (single marking). The 20-round Decide/cancel test is honestly near-certain rather than deterministic (post-release order genuinely racy; per-round consistency is what is asserted — correct design, accurately documented), backed by the deterministic sequential guard which still exists and passes. Red FAILs reproduced for both discriminating tests.

**X-19 encoding — CLOSED with doc residuals (L-02, L-03).** `ScopeID` injectivity independently re-proven; no raw joins remain in production code (exhaustive grep); collision/legacy/precedence unit tests pass; kill-switch names sound (disjoint literal level-prefixes — the raw joins there are safe by construction, verified); entity binding test passes. Residuals are contract-wording precision only.

**X-22 uniqueness — CLOSED.** Partial unique index + `IsUnknownRun` taxonomy + empty-ID rejection at the service boundary, all with passing tests (reproduced); migration preflight diagnostic + down/up cycle reproduced; `getByRun` determinism restored by the index. Residual: none in WS-A scope (attached-ID resolution is WS-E).

**Rule 3 ordering — CLOSED for admission+waiting (owner-amended scope).** Claim(pending) → park(waiting) → submit-bound-to-live-linkage → wait-live-without-sending → grant → proceed-exactly-once is proven E2E for first-submission AND replay entry paths with real DBOS (wait marks + authoritative executor status + 0/1 send counts, all reproduced); poll and decision paths proven non-submitting at the service boundary. The amendment provenance is solid (verbatim owner wording, mechanical match verified). The effect-authorization half is specified-now/WS-B-later per the same owner decision — correctly unclaimed.

**Lock-gate methodology (§5.C) — SOUND.** Both racers provably reach the intended boundary (statement-shape + transitive-chain + exact-count quorum rules out unrelated waiters and premature release); teardown is wedge-free on all paths (audited); red discrimination is deterministic (X-03, ApproveReject) or near-certain-with-honest-labeling (Decide/cancel); green determinism follows from lock-serialization + predicates. The one over-read risk — counting waiters as proof of observation — is closed because the blocked statement is itself downstream of the observation read in every gated path (verified per path: claim→replay→adopt; Decide checks→UPDATE).

**Migrations/preflight/lint (§5.F) — SOUND.** 000006 upgrade paths all reproduced (fresh/dup-diagnostic/unique/down-up/delimiter); preflight and determinism-lint untouched by WS-A and green via checks.sh (reproduced); the migration NOTE's index-schema claim empirically confirmed (V17). L-02 records the §12 wording gap.

**Alternate entry points (§4.7) — verifiably single-choked.** Exhaustive grep: production `Async.Submit` has exactly one call site (`submitRecorded`, which funnels all five entry paths through `submitChecked`); production `dbos.RunWorkflow` sites are probe `Start` funcs invoked only by the executor post-choke; no production enqueue paths exist. No bypass route found.

**Durable wait/replay (§4.5–§4.6) — proven within the amended scope.** See Rule 3 paragraph + §2 boundary statement. The marker-placement requirement ("immediately before the durable wait") verified in source (mark tx-step directly precedes `Recv`, nothing between); the authoritative-status requirement satisfied (live-non-terminal assertion on the real executor handle); the marker-vs-suspension distinction from the owner plan is respected (tests assert wait-branch-entry + liveness + 0 sends, never claim Recv-suspension proof from the mark alone).

**Prompt14 correction.** This review retracts the prompt14 down-file nit (X-draft "unqualified CREATE INDEX lands in public"): PostgreSQL creates the index in the parent table's schema (documented behavior), the migration NOTE states this correctly, `TestMigration000006FreshInstall` asserts `schemaname='app'` (reproduced green), and V17 proves it with direct SQL. The down files are correct as written.

## 5. Verdict justification (prompt16 §10)

**Verdict: CLEAR.** Condition-by-condition:

1. **Baseline fully established.** Green digest recomputed from the committed manifest (`f0b7dfa6…`); red snapshot re-digested pristine (`893df60f…`); current-tree delta fully enumerated (README post-submission edit + `prompt16.md`, §1); overlays verified test-only with byte-identical sources; a fresh self-built overlay reproduces red discrimination. The README's unrecoverable in-place edit (L-04) does not impair baseline identity: every load-bearing register content was independently re-verified.
2. **Red/green reproduced for all listed findings.** X-01 (3 triggers + 4 X-02 paths): 6 red FAILs with predicted messages, 7 green PASSes. X-03: red FAIL (divergent handles/orphan), green PASS (shared handle/1 workflow), race-clean, 3× stable. X-04: red FAILs for both discriminating tests (ApproveReject deterministic; Decide/cancel order-dependent as documented), green 4/4 PASS, race-clean, 3× stable. X-19/X-22: green suites pass (encoding unit tests, migration fresh/dup/unique/down-up/delimiter, empty-ID rejection); red behavior confirmed by source inspection (raw joins, no index) — no red runs were claimed for these. Rule 3 wait/replay: both E2E proofs green with real DBOS.
3. **No CRITICAL or HIGH finding remains open against WS-A acceptance — with the M-01 correction below.** Strongest finding is M-01. Precision correction (owner-directed, see §6): M-01 documents a terminal-state overwrite that violates the contract's normative lifecycle-mutation requirement — the report's original "nothing found breaks an invariant" phrasing overstated. What holds is narrower and sufficient for the verdict: nothing found loses work, permits unauthorized execution, or contradicts a load-bearing evidence claim, and M-01's impact is bounded (narrow race, full audit retention). All other items are LOW (contract wording, error hygiene, audit completeness at the margins) or observations. CLEAR stands on impact-boundedness plus the owner's explicit disposition making M-01 mandatory in WS-B — not on a claim that the contract is fully satisfied today.
4. **No other condition requires NOT CLEAR or INCONCLUSIVE.** Environmental substitution (PG16 for PG19) is recorded with its rationale and touches no version-sensitive semantics; every execution-dependent claim was executed (nothing INCONCLUSIVE-by-omission); the hard stop holds (no `lead-followup` implementation, manifest, registration, execution, or ADR approval — verified); WS-B remains correctly gated on this review, which now releases it.

**What is proven** (independently executed or line-verified): the §4 closure list — X-01/X-02/X-03/X-04/X-19/X-22 remediations, the amended Rule 3 admission+waiting permission across all five entry paths, lock-gate determinism, migration upgrade behavior, and the full green suite.

**What is merely claimed** (accepted on design + unit evidence, not E2E-proven here): submit-time linkage atomicity (contained, O-01); effect-time revalidation (WS-B, explicitly future); cross-invocation attach (WS-E, scaffolding only); production-provider behavior (no prod data exists — mock-only verified).

**What is defective:** M-01, L-01–L-06 (all bounded, none blocking).

**What remains unknown:** the exact bytes of the README edit (L-04, immaterial); behavior on PG19 specifically (substitution rationale in §1 — a PG19 confirmation run would close this formally if the owner wants belt-and-suspenders, but no version-sensitive surface is involved).

**WS-B is released.** Recommended WS-B intake: M-01 (fold into the lifecycle-mutation sweep), L-05/L-06 (error/audit hygiene in paths WS-B touches), O-01 + the §2 boundary statement (do not weaken the revalidation list; the wait-vs-execution distinction is load-bearing).

## 6. Addendum — owner acceptance and WS-B release (2026-11-23)

This section records the owner's decision on this review. It amends dispositions and pins baselines; it does not alter any verification result in §§1–5.

**Decision: WS-A ACCEPTED as the engineering baseline for WS-B; WS-B RELEASED.** The owner accepts that the review establishes the evidence needed to proceed, citing specifically: baseline verification, red/green reproductions, real-DBOS wait proofs, migration tests, and the explicit admission-vs-execution permission separation.

**M-01 disposition (mandatory, not optional).** The owner directs that M-01's contract-requirement violation (§6 predicated-transitions-or-written-exception) be stated plainly alongside its impact-bounded non-blocking status — see the corrections in §2 (M-01) and §5 condition 3 — and that M-01 is a **mandatory WS-B lifecycle-correctness item**, Priority 1 intake, not cleanup to be scheduled at convenience.

**WS-B intake priorities (owner-sequenced, verbatim):**

| Priority | Item | Required outcome |
|---|---|---|
| 1 | M-01 | Predicated terminal transitions, affected-row handling, deterministic cancellation race tests |
| 2 | Effect authorization | Atomic revalidation at the `ATTEMPTING` boundary |
| 3 | O-01 | Prove that approval changes between submission and effect authorization cannot result in an unauthorized effect |
| 4 | L-05 | Correctly classify trust-mismatch retry failures |
| 5 | L-06 | Audit retry-time authorization denials |

**Effect-authorization proof standard (owner-stated).** The critical proof is not that a granted workflow can execute. It is that **no invalid approval state can cross the effect boundary**, even when approval, cancellation, grants, or invocation state changes after workflow submission.

**Boundaries to preserve through WS-B (owner-stated, cf. §2 boundary statement):**

1. WS-A proves workflow admission and durable approval waiting. It does not prove consequential execution authorization.
2. DBOS durability does not imply exactly-once provider delivery.
3. A workflow entering a live wait does not grant permission to perform business operations.

**Pinned tree states (do not conflate):**

| State | Digest (full tree, `scripts/digest.sh tree`) | Content |
|---|---|---|
| Submitted green (WS-A exit) | `f0b7dfa6…` (full: `f0b7dfa6458b5bc5e41816a3a6eb9565447bd4b3d20bf27e6c6656f37d820932`) | 290-file manifest `docs/evidence/digests/p15-wsaexit-2026-11-23.manifest.txt` |
| Review-execution tree | `e20a4689…` (full: `e20a4689fd37842f19fdb983db0d271f0e4fa14c153ffef3872abf885f27700f`) | Submitted + README post-submission edit + `docs/prompt16.md`; all §3 runs executed here |
| Review-delivery tree | `a547c0c1…` (full: `a547c0c1b837ac96e5430745a06154575b8b87b6c7462b1d04982fd766100401`) | Review-execution + this report as first written (§§1–5); historical |
| Acceptance-record tree | `e20a4689…` file set + this report file | WS-B starts from the review-execution file set (291 files, digest above — recomputation command below) plus `docs/adv_review_prompt16.md` in its delivered form. A self-containing digest is impossible by construction (the digest string lives inside the hashed file); pin instead the immutable file set digest + the report's live hash (`sha256sum docs/adv_review_prompt16.md`, stated in the delivery message). No production, test, migration, or evidence file differs from the review-execution state. |

Reproduce the acceptance file-set digest (normalized relative paths, `docs/evidence/digests/` + the report itself excluded, mirroring `scripts/digest.sh` traversal):

```sh
cd /home/chaschel/Documents/go/miniapp
find . -path ./.git -prune -o -path ./docs/evidence/digests -prune \
  -o -path ./docs/adv_review_prompt16.md -prune -o -type f -print \
  | sed 's|^\./||' | sort | tr '\n' ' ' > /tmp/acclist.txt
bash scripts/digest.sh files $(cat /tmp/acclist.txt)
# expect e20a4689fd37842f19fdb983db0d271f0e4fa14c153ffef3872abf885f27700f
```

**Architectural milestone (owner-stated).** The mechanism that admits and parks workflows has now been independently examined. The next phase must establish that the framework — not individual miniapp authors — controls whether consequential work is allowed to happen.

**Standing directive (owner-stated).** Do not weaken the approval revalidation requirements to make WS-B easier to implement. That boundary is the reason this framework can safely own reliability while miniapps own business logic.

**Process note.** WS-B is released by this decision; implementation begins on the owner's explicit WS-B work order, not on this addendum alone. The reviewer's scratch probe file was removed before delivery (tree digest verified unchanged across its lifecycle); no production, test, migration, or evidence file was modified by the review.
