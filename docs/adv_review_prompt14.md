# Prompt 14 — Independent Adversarial Code Review of the Prompt 13 Implementation

- **Role:** independent Principal Architect / Go / PostgreSQL / distributed-systems security reviewer. Read-only; no code, test, migration, manifest, ADR, CI, or history touched except this report.
- **Governing spec:** `docs/prompt13.md` + `docs/plan4.2.md` (immutable authority) + G0 adjudication + `docs/prompt10.md` G1 checks.
- **Implementation report under review:** `docs/prompt13_imp.md` (49 lines) + `docs/phase1/classd-capability-matrix.md` (F-01–F-14) + `docs/phase1/classd-framework-readiness.md` (G1 matrix) + `docs/evidence/README.md` (register).
- **Review date:** 2026-11-23. HEAD `5cf9627` (merge over `e6b5b7b` G0 baseline); Phase 1 tree uncommitted.
- **Report path:** `docs/adv_review_prompt14.md` per review request. (Prompt14 §3/§22 name `docs/phase1/classd-framework-adversarial-review.md`; that file was not created — this is the only file added.)
- **Environment limits (read first):** no Docker daemon, no `TEST_*` URLs, no Neon access in this review environment. `go build`, `go vet`, `gofmt`, analyzer/preflight/contract unit tests, and `scripts/checks.sh` were executed fresh and pass; all 54 DB-backed tests SKIP here (2 kill-test parents FAIL without a DB — X-17). Every behavioral verdict below therefore rests on line-level code inspection + test-assertion inspection + pinned-SDK-source verification, with the agent's evidence register treated as an unconfirmed execution record per prompt14 §4. NOT PROVEN is used exactly where execution was claimed but cannot be reproduced from here.

## A. Executive verdict: NOT ACCEPTABLE (bounded, remediable — not a redesign)

The Prompt 13 agent did substantial, mostly careful work: the effects ledger is genuinely race-hardened at the database layer, per-family DBOS identity with pinned versions is real, the approval bridge/cancellation/kill-switch machinery exists with serious adversarial tests, the analyzer implements all seven rule families, and the evidence register is unusually honest about failures and dispositions. The core design is sound.

It is nevertheless not acceptable as-is, because five HIGH findings break important framework invariants, all on failure paths the test suite does not cover:

1. **X-01 — Approval bypass on retry.** If anything fails between invocation claim and approval-park persistence (approvals-table error, store error, crash), the record stays `running` with no run; the agent's same-key retry takes `confirmSubmit`, which submits the gated tool immediately with no approval linkage. The workflow (ApprovalID "") skips its approval wait and executes consequential effects with no approval ever requested. Deterministic authorization failure on a plausible operational event.
2. **X-02 — Approval-path submit failure strands the invocation permanently.** Record sits `WaitingApproval` naming a run that was never submitted (or no run at all); polls wait forever; grant fails forever; denial retries forever. No recovery path exists.
3. **X-03 — Concurrent same-key submits can orphan.** `store.update` overwrites `run_id` unconditionally, so two racing duplicates mint two run IDs; the loser's overwrite orphans a live, unaddressable workflow. Only sequential submit tests exist.
4. **X-04 — Concurrent conflicting approval decisions last-writer-win.** `Decide` reads without a row lock and updates without a state predicate, so simultaneous approve/reject (or decide-vs-cancel) silently overwrite each other — including a self-approval landing through the race.
5. **X-05 — Cancel-vs-claim window.** `CheckLive`'s `FOR UPDATE` lock is released when its transaction commits, before the claim INSERT and send; a cancellation recorded in that gap does not stop the new attempt. The "linearizes" comment overclaims, and no concurrent cancel-during-claim test exists.

Beyond these: one MEDIUM test-design flaw vacates all provider-side "exactly 1 send" assertions (X-07, mock log dedupes by op_id); "attach" is detect-and-exit, not run attach (X-08); version affinity is SDK-confirmed but not behaviorally discriminated (X-09); approval recovery depends on decider retry plus probe-owned Recv timeouts with no framework-mandated bound (X-11); settlement audit requires Status polling (X-12); evidence binds to an undigested dirty tree with a cited P0 entry that does not exist (X-18).

**The framework is not genuinely Class D-ready within the tested envelope today.** It is close: every finding has a bounded fix at the identified boundary, none reopens an architecture decision, and none requires G6, remote CI, production providers, or the `lead-followup` playbook. §M recommends bounded remediation and re-review, not redesign. The hard stop holds throughout: no `lead-followup` code, manifest, registration, execution, or ADR approval exists or is authorized (verified §B.7).

## B. Baseline and repository integrity

**B.1 HEAD/branch/remote (observed 2026-11-23):** branch `main` at `5cf9627` ("Merge branch 'main' of https://github.com/PithomLabs/miniapp"); history contains `e6b5b7b` (G0 baseline) intact; remote `origin` = `https://github.com/PithomLabs/miniapp`; no upstream tracking shown; stash empty; no commits or pushes performed by this review.

**B.2 Working-tree status:** 10 modified tracked paths, 0 staged, 30+ untracked paths (full list below). The prompt13_imp §J claim of "3 modified paths, 30+ untracked" understates the tracked delta: the actual tracked modifications are `contract/manifest.go`, `contract/manifest_test.go`, `contract/run.go`, `contract/validate.go`, `docs/evidence/README.md`, `docs/phase0/version-register.md`, `go.mod`, `go.sum`, `internal/store/store.go`, `scripts/checks.sh` (10 files, +212/−11). The "3 modified" figure matches the *prompt13-baseline* evidence entry (pre-prompt13 delta); prompt13 itself modified 7 more tracked files. The report's change inventory conflates the two states — minor, but it is exactly the kind of source-state ambiguity §4 warns about.

**B.3 Untracked paths (implementation + user-owned):** `analyzer/`, `approvals/`, `audit/`, `auth/`, `cmd/`, `docs/adr/miniapps/lead-followup-adr001.md`, `docs/adv_review2_prompt11.md`, `docs/adv_review3_prompt11.md`, `docs/adv_review_prompt11.md`, `docs/evidence/opencode-phase1-demo-2026-11-23.jsonl`, `docs/faq.md`, `docs/phase1/`, `docs/prompt10.md`–`docs/prompt14.md`, `docs/review2.md`, `effects/`, `executor/`, `migrations/000001*`–`000005*` (000003–000005 are prompt13's), `miniapps/`, `preflight/`, `probe/`, `registry/`, `runtime/`, `testkit/`, `transport/`.

**B.4 Source state vs evidence binding:** every prompt13 evidence entry cites "`5cf9627` + prompt13 tree" — an undigested dirty tree. No per-file hashes, no `git stash create`, no worktree digest. Worse, the readiness doc cites "the P0 baseline entry … (full list …)" but **no P0 entry exists** in `docs/evidence/README.md` (verified by grep; the closest is the "prompt13 baseline" entry, which gives counts, not paths). Consequences: (a) whether the tree changed between the Neon run and this review is unknowable; (b) local-vs-Neon "same tree" claims are asserted, not demonstrated. See X-18. This review's own runs bind to the current tree as inspected.

**B.5 G0 baseline intact:** `e6b5b7b` present in history; `spikes/`, `docs/phase0/` (except the version-register row addition), and all G0 evidence entries untouched — verified via `git status` (no modifications under those paths).

**B.6 Immutable documents unchanged:** `docs/plan4.2.md` unmodified (absent from the modified list) ✓. `docs/DBOS.md`, ADR-001 port, G0 adjudication unmodified ✓.

**B.7 User-owned vs agent files:** `docs/faq.md` (owner Q&A, per prompt13_imp §J — content reads owner-authored, accepted as user-owned), `docs/adv_review*_prompt11.md` (prior reviews), `docs/review2.md`, prompts 10–14, and the `lead-followup` ADR draft are non-implementation artifacts. `lead-followup` appears in Go sources only as comments (`runtime/service_test.go:264`, `cmd/preflight/main.go:41`, `probe/classd/doc.go:9` — verified by grep): no implementation, manifest, registration, or execution. Hard stop holds.

**B.8 Review-created files:** exactly one: `docs/adv_review_prompt14.md` (this report).

## C. Implementation report claim audit

Verdicts: VERIFIED (independently established from code/tests/evidence/SDK source), PARTIALLY VERIFIED (true within a narrower envelope than claimed), NOT VERIFIED (no adequate basis found), CONTRADICTED (evidence refutes it). "Code-verified" means established by line-level inspection with DB execution taken from the agent's register (not reproducible here — see header).

| # | Claim (prompt13_imp + §5 list) | Verdict | Basis |
|---|---|---|---|
| C1 | Proven end to end on local PG19 + separate Neon pass | PARTIALLY VERIFIED | Register separates the passes honestly; local green + Neon green-with-3-harness-fixes-then-green is well-documented. But: source binds to an undigested dirty tree (X-18); nothing here re-executes DB paths; 2 kill tests FAIL (not skip) DB-less (X-17). "Proven" overstates; "executed and recorded by the agent" is accurate. |
| C2 | F-01–F-14 PROVEN except F-06 PARTIAL | CONTRADICTED (as stated) | Independent §D verdicts: zero of fourteen fully verified; F-07/F-08/F-11 come closest but carry residuals (see rows). The three headline defects (X-01, X-03, X-04) sit inside "PROVEN" findings. |
| C3 | Canonical invocation + submit idempotency correct | PARTIALLY VERIFIED | Canonical path is real and exclusive (§A1/A2 verified); record-before-submit + idempotent re-submit correct on the dispatch path (code + sequential tests). False on the approval path (X-01 bypass, X-02 strand) and under same-key concurrency (X-03). |
| C4 | Entity-key validation, tenant isolation, Class A serialization work | VERIFIED (code) | Contract format + admission strictness + dispatch derivation + advisory-lock serialization + genuine peak/rendezvous tests + genuine BYPASSRLS test. Residuals: X-19 (":" ambiguity), X-08 (attach semantics). |
| C5 | Class D attach + durable-sequence identity work | PARTIALLY VERIFIED | Durable-sequence record with partial unique index + DB-arbitrated claims + cross-tenant independence are real. "Attach" is detect-and-exit with an observation row, not run attach (X-08); concurrency tested sequentially, not barrier-synced. |
| C6 | Effects ledger race-hardened | PARTIALLY VERIFIED | Predicated transitions, contention-vs-adopt lease rule, takeover, settle, tenant scoping, genuine 8-way gate-held + kill tests — all verified by reading. Gaps: X-05 (cancel window), X-07 (provider-count vacuity), stale-resume-after-takeover interleaving untested, Settle unaudited. |
| C7 | Cancellation durable-first, handles in-flight | PARTIALLY VERIFIED | Durable-first ordering, pending-approval settlement, redelivery via re-cancel + Status piggyback, InFlight reporting, settle-by-evidence — verified by reading + strong `TestCancelInFlightReported`. Gaps: X-05 (post-record claim window), X-14 (shutdown-failure ordering). |
| C8 | Approval decisions survive crashes, delivered to waiting workflows | PARTIALLY VERIFIED | Idempotent Decide, decision signal, table re-read helper, scope enforcement, lazy expiry observation, denial/expiry settlement — verified by reading. Gaps: X-02 (strand), X-04 (Decide race), X-11 (recovery = decider-retry + probe-owned timeout; no framework bound; "durable mailbox" nonexistent). |
| C9 | Per-family app identity + pinned compat version | VERIFIED (mechanism) | `miniapp-<id>` AppNames, explicit `ApplicationVersion` (`family/compat`), admission agreement rule, foreign-pending reporting — verified by reading + SDK source (affinity filters confirmed). Gaps: X-09 (affinity not behaviorally discriminated), X-10 (PENDING-only visibility). |
| C10 | Version affinity + drain visibility proven | PARTIALLY VERIFIED | Affinity mechanism SDK-confirmed; overlap/drain/restart tests genuine but same-process, same-code (X-09). Drain visibility exists but is PENDING-only + startup-only (X-10). Artifact retention correctly deferred to G6 (no contradiction). |
| C11 | Effect-time policy revalidated | PARTIALLY VERIFIED | StrictPolicy (invocation-row tool/trust, live grants, token binding, 5 kill-switch levels) + revocation/kill/token-mismatch tests — verified by reading. Gaps: X-13 (refusals unaudited), X-15 (switch-table hardening/operability). Volumetric correctly deferred (OSI-4). |
| C12 | Durable sleep + signal recovery proven | PARTIALLY VERIFIED | Real SIGKILL subprocess kill + relaunch + attach (verified mechanics), signal matrix subset, suspend gate, honest envelope — verified by reading. Gaps: matrix missing signal-after-timeout / signal-then-kill / unavailable-group cases; envelope is 20×2s, not a capacity claim (honestly presented). |
| C13 | Lifecycle status exposes staleness | VERIFIED | `Stale` marker, terminal-authoritative, failing-fake test — verified by reading. Residual: sync-error swallow (X-LOW), cancelled-with-undelivered-stop reads terminal (documented redelivery). |
| C14 | Analyzer implements all seven rule families | PARTIALLY VERIFIED | All seven families implemented, fixtures green (verified by running `TestAnalyzer`), dogfood clean (verified via checks.sh), arg indices match real SDK (verified). Gaps: X-16 (untested step APIs, named-vs-literal divergence, documented-boundary misses, 12-vs-13 count). |
| C15 | Generic probe has 12 tests | VERIFIED (count + neutrality) | 12 `TestProbe*` + 1 worker entrypoint; neutral naming/identifiers; no sales semantics; hard-stop disclaimer present. Coverage mapping in §H (several §18 items partial). |
| C16 | REST/MCP/CLI thin adapters | VERIFIED | All three delegate every operation to `runtime.Service`; no independent auth/policy/execution (verified by reading all three servers + equivalence tests). Residuals: REST no body limit, MCP credential-in-args (demo scaffolding), CLI `--credential` flag option (LOW). |
| C17 | OpenCode drives a real demonstration | PARTIALLY VERIFIED | Transcript shows genuine agent-driven tool use (6 calls: search, refusal-without-retry, submit→status→signal→succeeded; summary echoes observed IDs; credentials redacted). Gaps: server-audit linkage absent, model/version not in artifact (entry asserts textually). |
| C18 | ADR preflight + CI enforcement exist | PARTIALLY VERIFIED | Preflight exists with 9 fixture tests, verified live behavior shape, hard-stop FAIL logic, CI-wired via checks.sh (verified by running). Gaps: X-20 (orphan-FAIL, no manifest-side checks); X-17 (remote CI unobserved + would fail DB-less). |
| C19 | 13/14 G1 checks pass (12 PARTIAL) | CONTRADICTED (as stated) | Independent §I matrix: 4 PASS, 9 PARTIAL, 1 PARTIAL-with-stronger-caveat (12). The report's own rows are well-formed; the PASS labels overclaim on rows 2,4,5,6,7,8,9,10,14. |
| C20 | No real provider writes / integrations / lead-followup | VERIFIED | mock-only channels (verified wiring in stack + cmd), gate 3 green (verified run; stale package list noted X-21), no credentials in tree (verified JSON-scan gate ran), lead-followup comment-only (verified by grep). |
| C21 | No existing tests weakened | NOT VERIFIED (unverifiable as stated) | The only tracked test modified (`contract/manifest_test.go`) is purely additive (verified by diff) ✓. But all increment 4–5 tests are untracked (no git pre-image), so "weakened" has no baseline; the three Neon rewrites read as equivalent-or-stronger (gate-held determinism, runnable bypass, row-wait) but that is judgment, not diffing. |
| C22 | All work remains uncommitted | VERIFIED | `git status` shows the full tree uncommitted; no new commits by the agent (HEAD still `5cf9627`); nothing pushed (no evidence of push; remote untouched — remote state itself not probed, per restrictions). |
| C23 | Pinned-SDK behaviors (report §E) | VERIFIED (material items) | Independently confirmed in v1.9.0 source: recovery + dequeue filter on (version, appName); partition/dedup mutual exclusion; FIFO dequeue order; Send buffers without state checks; queue-name conflicts (also empirical). FIFO/pending-count/slot claims consistent with the queue-eval test. Remaining micro-claims (100-attempt dead-letter, no GC, steps-don't-retry) not re-verified — accepted as cited, low risk. |
| C24 | "Exactly-once" language corrected + lint added | PARTIALLY VERIFIED | Ledger comments fixed (verified `effects/doc.go`); preflight ADR-text lint exists (verified). BUT: no Go-comment lint exists in checks.sh (the "lint added" overreads), and `mockcap.go:77` + `effects_test.go:18` still carry exactly-once phrasing — the mockcap one factually false (X-07). |
| C25 | Files changed/added (§J) | PARTIALLY VERIFIED | Package inventory matches the tree (verified by find). "3 modified paths" understates (10 tracked files modified — §B.2). `docs/faq.md` accepted as not-theirs (content reads owner-authored; left alone per note). |

## D. F-01 through F-14 independent verdicts

Scale: VERIFIED / PARTIALLY VERIFIED / NOT VERIFIED / CONTRADICTED. Detail lives in §E findings and §§F–M; this table is the continuity register.

| ID | Report | Independent | Basis |
|---|---|---|---|
| F-01 | PROVEN | PARTIALLY VERIFIED | Dispatch-path record-before-submit + idempotent re-submit correct (code + sequential tests). Approval path broken: X-01 (bypass on retry), X-02 (strand). Same-key concurrency untested and racy: X-03. |
| F-02 | PROVEN | PARTIALLY VERIFIED | Durable-cancel-first, denial/expiry settlement, idempotent Decide, decision signal, re-read helper, lazy observation — all real. Gaps: X-02, X-04, X-11; lazy-expiry E2E with a real workflow missing; `settleDenied` "confirmed down" overclaims cooperative cancel. |
| F-03 | PROVEN | PARTIALLY VERIFIED | Predicated transitions, lease contention-vs-adopt, takeover, settle/list, tenant scoping, 8-way + kill tests — genuinely strong. Gaps: X-05, X-07, stale-resume interleaving untested, Settle unaudited/unexposed. |
| F-04 | PROVEN | PARTIALLY VERIFIED | Five-identity separation in records; entity contract + admission + derivation enforced and tested. "Attach" is detect-and-exit (X-08); sequence-claim race untested; X-19 residual. |
| F-05 | PROVEN | PARTIALLY VERIFIED | Per-family AppName, pinned `family/compat` versions, admission agreement, foreign-pending reporting — verified incl. SDK affinity filters. Gaps: X-09, X-10. G6 retention correctly unclaimed. |
| F-06 | PARTIAL | PARTIALLY VERIFIED (agree, wider residuals) | Grant/token/kill-switch enforcement + tests verified; volumetric correctly deferred per OSI-4. Add X-13, X-15. Fail-closed on missing limits verified (no silent unlimited). |
| F-07 | PROVEN | VERIFIED | Canonical compare + `KeyConflict`→terminal + whitespace-replay test + IntentHash canonicalization — verified by reading. Residual: `canonicalJSON` single-Decode trailing leniency unreachable via validated Invoke (LOW, noted). |
| F-08 | PROVEN | VERIFIED (with residual) | Explicit predicates on all mutation/load paths checked; BYPASSRLS test is genuine privileged execution (not text-matching). Residual: X-19. |
| F-09 | PROVEN | PARTIALLY VERIFIED | Scope/roles, idempotent-vs-conflicting Decide, self-approval, expiry format/record, invocation gate — verified. Gap: X-04 (concurrent Decide race defeats all of these). |
| F-10 | PROVEN | PARTIALLY VERIFIED | Real SIGKILL restart + attach, signal subset, suspend gate, honest envelope — verified mechanics. Gaps: matrix missing 3 timing cases; envelope is 20×2s (correctly non-guarantee). |
| F-11 | VERIFIED-equivalent | VERIFIED | `Stale` marker, terminal-authoritative, executor-down test — verified. Minor residual: status-sync error swallow. |
| F-12 | PROVEN | PARTIALLY VERIFIED | Queue evaluation genuine (behavioral + SDK-confirmed); Class A serialization genuinely proven (peak + rendezvous); durable uniqueness via probe record. Gaps: X-08; queue convention unenforced (no framework queue use yet — correctly scoped). |
| F-13 | PROVEN | PARTIALLY VERIFIED | Seven families implemented; fixtures green (executed); dogfood clean (executed); CI-wired (executed locally). Gaps: X-16; remote CI unobserved (X-17). |
| F-14 | PROVEN | PARTIALLY VERIFIED | Preflight + fixtures + hard-stop FAIL + CI gate all real (executed). Gaps: X-20; manifest-side enforcement lives in admission instead (acceptable split, narrower than prompt13 §16's list).

## E. New findings

IDs X-01… (HIGH first, then MEDIUM, LOW, OBSERVATION). Each carries the §21 ten fields.

### HIGH

**X-01 — Approval bypass: retry after approval-path failure submits the gated tool with no approval (HIGH, blocks G1)**
1. ID/severity: X-01 / HIGH.
2. Location: `runtime/service.go:163-167` (dispatch on `created`), `:199-235` (`requestApproval`), `:354-369` (`confirmSubmit` — no approval check), `:186-194` (replay routes `Running` Class D to `confirmSubmit`); probe skips wait when `ApprovalID == ""` (`probe/classd/sequence.go:156`).
3. Governing requirement: prompt13 §2 (authorization before dispatch), §11 (approval valid and current before consequential effects), plan4.2 §9.3/§14, G1 row 4.
4. Defect: `confirmSubmit` submits any `Running` Class D record unconditionally. Three triggers leave a gated invocation `Running` with no run: (a) `Approver.Submit` fails in `requestApproval` (line 210-218) → retryable, record still `Running`; (b) the `WaitingApproval` update fails (line 219-221) → same; (c) crash between claim commit and park. The same-key retry then takes `replay → confirmSubmit`, which mints a run ID and submits with `ApprovalID: ""` (fresh `InvokeRequest`), and the workflow skips its approval wait entirely — consequential effects execute with no approval requested, returned, or recorded (any orphan approval row from (b) dangles pending, undiscoverable).
5. Failure scenario: transient approvals-table error (failover, deploy) during a gated invoke → agent retries per the documented retryable contract → unapproved execution. No attacker needed; normal operations suffice.
6. Consequence: authorization-before-dispatch violated for every approval-gated Class D tool under failure; customer-facing effects without required human approval; silent (no audit names the bypass).
7. Reproduction: fault-inject `Approver.Submit` (or the park update) to fail once on a gated invoke, retry same key, assert the workflow ran effects with no `approval-requested` audit and no `ApprovalID` in the run input.
8. Class: code defect + test gap (no approval-path submit-failure test exists).
9. Resolution: `confirmSubmit` must re-enter approval handling for tools requiring approval when no granted approval exists (re-request or refuse — never submit blind); add the fault-injection regression test above plus crash-window coverage.
10. Blocks: G1 (rows 2, 4, 9) and Class D readiness.

**X-02 — Approval-path submit/audit failure strands the invocation permanently (HIGH, blocks G1)**
1. ID/severity: X-02 / HIGH.
2. Location: `runtime/service.go:226-234` (submit after park), `:219-224` (park/audit updates), `:243-287` (`pollApproval` never submits).
3. Governing requirement: prompt13 §8 (retry must yield a valid handle or correctly settled terminal state), §11.7 (rejection/expiry settles the run), G1 rows 2/9.
4. Defect: if `submitRecorded` fails inside `requestApproval`, the record sits `WaitingApproval` naming a run that was never submitted; if the park audit fails, it sits `WaitingApproval` with no run at all. `pollApproval` only polls — it never submits — so: polls wait forever; a later grant fails forever at `Async.Status` (unknown run); a denial retries `settleDenied` forever (`cancelRun` on a nonexistent run fails closed-loop). No path converges except Cancel (which terminates the record while still erroring on redelivery).
5. Failure scenario: executor down at park time (deploy, crash) → gated invocation accepted → approved later by a human → grant can never take effect; the approval, the human action, and the invocation are all stranded.
6. Consequence: permanently wedged invocations requiring DB surgery; denied approvals that cannot settle; operator-facing retryable errors that never converge.
7. Reproduction: `failNextSubmit` on a gated invoke → retry → approve → assert poll converges (currently fails forever).
8. Class: code defect + test gap.
9. Resolution: `pollApproval` must submit-if-missing (reuse `confirmSubmit` semantics with the stored approval linkage) before polling; add park-failure and grant-after-failure regression tests.
10. Blocks: G1 (rows 2, 9) and Class D readiness.

**X-03 — Concurrent same-key submits race on unpredicated `run_id` overwrite → orphan workflow (HIGH, blocks G1 readiness)**
1. ID/severity: X-03 / HIGH.
2. Location: `runtime/store.go:126-138` (`update` overwrites `run_id` unconditionally); `runtime/service.go:335-369` (both `submitRecorded` and `confirmSubmit` mint on empty).
3. Governing requirement: prompt13 §8 (same-key retry; concurrent duplicates; "never claim atomicity unless proven"), F-01.
4. Defect: two in-flight same-key requests: winner claims (created), loser loads the record before the winner's `run_id` update lands. Both see empty `run_id`; both mint distinct IDs; both update (last wins) and both submit. Result: two DBOS workflows, record naming one, the other orphaned — live, sending, unobservable, uncancellable through the framework. The existing submit tests (`TestSubmitFailureRetryAttaches`, `TestConfirmSubmitStableHandle`, `TestSubmitIdempotent`) are all sequential; no barrier-synced same-key test exists.
5. Failure scenario: agent timeout → duplicate submit; double-click; two harness replicas sharing a key namespace.
6. Consequence: orphan workflow executing effects outside framework observability — the exact failure F-01 claims to close.
7. Reproduction: barrier-synced same-key double `Invoke` against the real (or DBOS) executor; assert one workflow exists and the record names it.
8. Class: code defect + test gap.
9. Resolution: predicated `run_id` adoption (`UPDATE … WHERE run_id = ''`, reload on 0 rows affected) or `SELECT FOR UPDATE` on the invocation row across mint+submit-record; add the concurrent regression test.
10. Blocks: Class D readiness (submission atomicity is load-bearing for every Class D claim); G1 row 2 contributor.

**X-04 — Concurrent conflicting `Decide` calls last-writer-win; decide-vs-cancel overwrites cancellation (HIGH, blocks G1)**
1. ID/severity: X-04 / HIGH.
2. Location: `approvals/approvals.go:104-140` (read without row lock; two unpredicated `UPDATE … WHERE id = $3::uuid`).
3. Governing requirement: prompt13 §11.6 (duplicate/conflicting decisions consistently rejected or idempotent), §15.1, G1 row 9.
4. Defect: two simultaneous `Decide` calls both read `pending`, both pass checks, both UPDATE — last commit wins silently. Approve+reject → either outcome with no conflict error. Requester+approver racing → self-approval can land. Decide racing `CancelPending` → the decision overwrites `cancelled` (or vice versa) with no error. The identical-re-decide idempotency only covers sequential retries.
5. Failure scenario: two operators (or an operator + automation) decide simultaneously; operator decides while another cancels.
6. Consequence: authorization decisions silently substituted; self-approval and post-cancel approval achievable through timing; audit records a decision process that never happened.
7. Reproduction: barrier-synced approve+reject on one approval; assert exactly one wins and the loser errors (currently both succeed, last wins). Barrier-synced Decide + Cancel; assert decision-after-cancel refused.
8. Class: code defect + test gap (all Decide tests sequential).
9. Resolution: `SELECT … FOR UPDATE` (or predicated `UPDATE … WHERE status = 'pending'` + affected-row check) inside `Decide`; same predicate discipline for the expiry-marking update; add race regression tests.
10. Blocks: G1 (row 9) and Class D readiness.

**X-05 — Post-cancel effect claims: `CheckLive` lock does not cover the claim; new attempts can begin after durable cancel (HIGH, blocks Class D readiness)**
1. ID/severity: X-05 / HIGH.
2. Location: `runtime/lifecycle.go:226-255` (`checkLive`: `FOR UPDATE` then commit/release); `effects/effects.go:165-181` (check → policy → claim in separate transactions; `claim` verifies nothing about invocation liveness).
3. Governing requirement: prompt13 §10 core invariant — "After cancellation is durably recorded, no new consequential effect attempt may begin."
4. Defect: the `FOR UPDATE` lock is released when the check transaction commits, before the policy check, claim INSERT, attempt transition, and send. A cancellation recorded in that gap does not stop the in-flight claim: INSERT proceeds, transition proceeds, provider send proceeds. The comment ("either the check observes the cancellation and refuses, or the cancellation lands after the check and the in-flight attempt settles by evidence") redescribes the hole as the design — but the second case IS a new attempt beginning after durable cancel, which the invariant forbids. `TestCancelBeforeClaim` is sequential (cancel-then-claim) and cannot catch this; no concurrent cancel-during-claim test exists.
5. Failure scenario: long effect pipeline (slow policy read) with an operator cancel landing mid-claim → provider send after cancel recorded; the send is also invisible to the cancel response's `InFlight` list if the INSERT lands after `LiveEffects` reads.
6. Consequence: the single strongest Workstream D invariant does not hold; post-cancel sends with misleadingly clean cancel receipts.
7. Reproduction: gate-held claim (block inside `CheckPolicy` or pre-INSERT) + concurrent Cancel; assert the claim refuses once cancel commits.
8. Class: code design defect + test gap.
9. Resolution: verify liveness atomically with the claim (e.g., conditional INSERT predicated on a live invocation row, or re-check under the same row lock held through the claim INSERT); add the gated race regression test; correct the comment.
10. Blocks: Class D readiness; G1 row 2 contributor (cancel-safe durability).

### MEDIUM

**X-06 — `Signal` falsely succeeds for unknown runs; masks stranded runs, litters orphan notifications (MEDIUM)**
1. ID/severity: X-06 / MEDIUM.
2. Location: `executor/dbos/executor.go:291-307` (fan-out returns on first nil); SDK `sysdb.Send` inserts into `notifications` with no existence/state check (verified in v1.9.0 source; notifications keyed by workflow UUID globally).
3. Governing requirement: prompt13 §19 (lifecycle operations honest); G1 rows 2/9 (signal operations truthful).
4. Defect: because `dbos.Send` never errors for unknown destinations, the fan-out loop always succeeds on the first family tried. Signalling a nonexistent run returns success and buffers an orphan notification row. Cross-family delivery works accidentally (UUID-addressed notifications), so the defect is not misrouting but false success: `DecideApproval` on an X-02-stranded run reports the decision "notified," and operator signals to dead runs report "delivered."
5. Failure scenario: grant path on a stranded invocation (X-02) — decision commits, signal "succeeds," audit claims notification, workflow (nonexistent) never wakes; poll then fails forever with no hint the signal went nowhere.
6. Consequence: silent lifecycle lies at exactly the failure boundary the bridge exists to cover; unbounded orphan-notification accumulation (no GC per ops doc).
7. Reproduction: signal a bogus run ID against the real executor; assert an error (currently nil). Then signal-after-restart with an emptied `runs` map; assert delivery to the owning family (currently accidental-success).
8. Class: code defect (missing existence verification).
9. Resolution: verify run existence (`RetrieveWorkflow`) before `Send` in `Executor.Signal` (or check-and-send atomically if the SDK offers it); add unknown-run and post-restart routing tests.
10. Blocks: Class D readiness polish (compounds X-02); not standalone G1-blocking.

**X-07 — Mock provider log dedupes by `op_id`: all "exactly 1 send" assertions via `SentCount` are vacuous (MEDIUM)**
1. ID/severity: X-07 / MEDIUM.
2. Location: `testkit/mockcap/mockcap.go:77` (comment), `:132-136` (`INSERT … ON CONFLICT (op_id) DO NOTHING`); consumers: `TestStalePendingTakeoverSendsOnce`, `TestKillMidSendAdoptsUnknown` (partial), `TestProbeHappyPath/ReplayDeterminism/PoisonQuarantine/Restart/EligibilityExit/CancelMidWait`, `providerSendsSubject`.
3. Governing requirement: prompt13 §9 (adversarial tests must prove no duplicate sends), §20 (regression tests prove invariants).
4. Defect: with `ON CONFLICT DO NOTHING` on the PK, the second through Nth sends insert nothing — `SentCount == 1` proves "at least one acceptance," never "exactly one send." The comment "Exactly-once-at-provider means exactly one row survives" is false as written: exactly one row survives regardless of send count. (The 8-way race test is NOT affected — it uses an in-process send counter; the kill test's no-blind-retry is saved by its parent-side counter. The vacuity hits every `SentCount == 1` "exactly once" assertion and the `== 0` assertions remain meaningful.)
5. Failure scenario: a ledger bug double-sends → log still shows 1 row → suite stays green.
6. Consequence: the provider-side exactly-once evidence for takeover, kill-recovery, restart, and all probe send-counts is weaker than claimed; F-03/F-07-adjacent "1 send" rows rest partly on assertion design, not behavior.
7. Reproduction: instrument a double-`Send` call in a scratch test; observe `SentCount == 1` (demonstrates vacuity without touching the tree — or reason from the SQL).
8. Class: test-design defect + documentation defect (false comment + surviving "exactly-once" phrasing).
9. Resolution: log every attempt as its own row (serial PK, no dedup; keep an `attempted_at` + outcome column), count rows for exactly-once assertions; fix the comment; re-run the affected suites.
10. Blocks: G1 row 7 contributor (duplicate-effect evidence quality).

**X-08 — "Attach" is detect-and-exit, not run attach; loser succeeds while the sequence still runs (MEDIUM)**
1. ID/severity: X-08 / MEDIUM.
2. Location: `probe/classd/sequence.go:141-146` (attached path runs `observeStep`, returns success `"attached"`); `probe/classd/probe_test.go:392-441` (`TestProbeConcurrency` sequential, asserts 1 observation row).
3. Governing requirement: prompt13 §7 (invocation records preserve identity when multiple invocations attach to one existing run), §18.14; prompt14 B3 (loser references the correct existing workflow; receives a valid handle).
4. Defect: the losing invocation runs its own full DBOS workflow, which detects the live generation, writes an observation row, and completes SUCCESSFULLY — while the winner's sequence continues for seconds/minutes. The loser holds no reference to the winner's run (observation row carries no run linkage), and its `Succeeded` status misdescribes a sequence that is still active. B3's "references the correct existing workflow" is not implemented at any layer (no runtime attach API exists). The concurrency test is also sequential by construction (waits for active, then submits) — the true claimStep race is untested.
5. Failure scenario: agent polls the loser invocation → `succeeded` → reports completion to the user while touches are still being sent (or while the winner later fails/quarantines — outcomes the loser never reflects).
6. Consequence: misleading lifecycle for every attached duplicate; generation/supersede behavior (un-cancelled superseded runs keep sending — sequence.go has no supersede→cancel linkage) untested.
7. Reproduction: run `TestProbeConcurrency` and inspect the loser's terminal status vs the winner's still-active sequence (visible in the existing test's own flow).
8. Class: design gap (probe) + test gap (no barrier-synced claim race; no supersede test).
9. Resolution: either implement true run-attach (loser invocation references winner run ID; status follows the winner) or re-scope the claim to "detect-and-exit" with explicit lifecycle semantics + a supersede→cancel linkage; add a barrier-synced claim-race test and a supersede test.
10. Blocks: Class D readiness polish; G1 row 8 contributor.

**X-09 — Version affinity SDK-confirmed but not behaviorally discriminated (MEDIUM)**
1. ID/severity: X-09 / MEDIUM.
2. Location: `executor/dbos/dbosexec_test.go:621-662` (`TestVersionOverlap`); SDK filters verified at `ReenqueueForRecovery` + dequeue (`application_version = $3`, `application_name` clause).
3. Governing requirement: prompt13 §12 (live N→N+1 pending-run transition test; "a parser test or unit assertion on a version string is not proof"), plan4.2 §11.2.
4. Defect: both versions run IDENTICAL code with both workers alive. Cross-version step execution would still yield exactly 1 marker row (DBOS step dedup), so `n == 1` cannot discriminate whether v2 touched v1's run. The affinity evidence is therefore the SDK source reading (which this review confirms is correct) plus a test that would pass with or without affinity.
5. Failure scenario: a future SDK/config regression disables version filtering → v2 executes v1 steps → this suite stays green.
6. Consequence: the "native affinity PROVEN" label rests on inspection, not behavior — precisely the substitution prompt14 N2 forbids.
7. Reproduction/required evidence: shut down v1 worker, leave v2 up, assert v1's pending run does NOT progress (or use version-distinguishable step markers and assert executor identity).
8. Class: test gap (mechanism itself verified in SDK source).
9. Resolution: add the v1-down/v2-up non-progress test (or per-version marker attribution); keep the SDK citation as mechanism explanation, not as the proof.
10. Blocks: G1 row 6 contributor (not standalone — affinity mechanism is real).

**X-10 — Foreign-version visibility is PENDING-only and startup-only (MEDIUM)**
1. ID/severity: X-10 / MEDIUM.
2. Location: `executor/dbos/executor.go:143-164` (`reportForeignPending`: `WithFilterStatus(Pending)`, called once in `New`).
3. Governing requirement: prompt13 §12.5 (unknown/incompatible versions visible and operator-actionable), §7.7 (alerting).
4. Defect: ENQUEUED, DELAYED, SUSPENDED (sleeping/signalled-waiting — i.e., most genuinely pending Class D work) runs under foreign versions are never reported; runs that become pending after startup are never reported; the report is a log line with no queryable surface (alerting explicitly future work per ops doc).
5. Failure scenario: deploy v2 while v1 runs sleep for days → no startup warning (sleepers not PENDING-listed... — actually Recv-waiters empirically appear; ENQUEUED/DELAYED provably do not) → v1 retired → sleepers stranded with no alert.
6. Consequence: drain visibility covers a subset of drain-relevant states; silent abandonment remains possible for non-PENDING foreign runs.
7. Reproduction: seed ENQUEUED + SUSPENDED foreign-version runs, start the worker, observe the absence from the warning.
8. Class: code gap (partial implementation).
9. Resolution: list all non-terminal states (or drop the status filter and group by version/state), expose counts queryably (or document the log-pattern alert contract), and re-run the overlap test asserting full-state coverage.
10. Blocks: Class D readiness polish; G1 row 6 contributor.

**X-11 — Approval recovery has no framework-mandated bound; "durable mailbox" does not exist (MEDIUM)**
1. ID/severity: X-11 / MEDIUM.
2. Location: `runtime/lifecycle.go:180-219` (signal-after-commit, no redelivery); capability matrix F-02 ("decide-retry + durable mailbox + workflow re-read" — no mailbox exists; verified by grep); `probe/classd/sequence.go:335-366` (probe-owned 30s-default Recv timeout + re-read); `approvals` (no sweeper — deliberate).
3. Governing requirement: prompt13 §11 (crash between commit and signal must still reach the workflow; bounded observation of expiry), prompt14 E1/E3.
4. Defect: `TestDecideCrashRedelivers` only proves decider-retry convergence (the exact insufficient shape prompt14 E1 names). Framework-initiated redelivery does not exist (poll-time renotify explicitly rejected in the matrix). Recovery therefore depends on (a) the decider retrying, or (b) the workflow's Recv timing out and re-reading — and (b) is probe convention (default 30s), not framework enforcement: a miniapp with a long/unbounded Recv parks past expiry with no observation. Related: `DecideApproval` audits only after a successful signal, so a committed-but-unsignalled decision is also unaudited until retry.
5. Failure scenario: decide commits → process dies before signal → decider never retries (human closed the tab) → workflow with a 24h Recv timeout parks a full day past a 1h approval expiry with no timeout, no escalation, no audit.
6. Consequence: "crash-safe approval bridge" holds only under miniapp discipline the framework does not require; expiry is observable, not enforced, on the workflow side.
7. Reproduction: decide-then-kill-before-signal with a no-retry decider and a long-Recv workflow; assert bounded observation (currently unbounded).
8. Class: design gap + documentation defect ("durable mailbox" phrase) + test gap (no long-Recv + lost-signal test).
9. Resolution: mandate the bound in framework code (required Recv-timeout ceiling for approval waits, or a reconcile sweeper, or poll-driven renotify — pick one and enforce it); audit the decision at commit time, not after signal; strike/replace the mailbox phrase; add the lost-signal + long-wait regression test.
10. Blocks: Class D readiness (approval recovery is load-bearing); G1 row 9 contributor.

**X-12 — Settlement audit requires Status polling; never-polled runs lack `run-settled` records (MEDIUM)**
1. ID/severity: X-12 / MEDIUM.
2. Location: `runtime/lifecycle.go:43-48` (only `run-settled` emission site; all probe/transport audit-chain tests poll via `pollTerminal` first).
3. Governing requirement: plan4.2 §9.5/§13 complete audit trail; G1 row 5.
4. Defect: workflow completion emits no audit event; the settlement record appears only when someone calls `Status` afterward. Fire-and-forget invocations that nobody polls have permanently incomplete chains. Every passing audit test masks this by polling first.
5. Failure scenario: agent submits, never polls (or polls a different invocation ID) → compliance reconstruction shows dispatch with no outcome.
6. Consequence: G1 row 5 "complete" overstates; attribution exists, settlement does not, for unpolled runs.
7. Reproduction: complete a run without any `Status` call; query the audit chain; observe the missing settlement event.
8. Class: code gap.
9. Resolution: emit settlement from a completion hook (or executor callback on terminal discovery) rather than only from reads; add a no-poll audit-completeness test.
10. Blocks: G1 row 5 contributor.

**X-13 — Refused effect claims leave no ledger or audit trace (MEDIUM)**
1. ID/severity: X-13 / MEDIUM.
2. Location: `effects/effects.go:165-170` (refusals return errors; "refused claims insert nothing" per the `Perform` contract); no audit call on any refusal path.
3. Governing requirement: prompt13 §13 ("refusals and blocked effects are attributable and auditable"), prompt14 §12.
4. Defect: a kill-switch/revocation/token refusal surfaces only as a workflow step error (visible in DBOS history and, eventually, as a generic failed run). No ledger row, no audit event names the switch, the revoked grant, or the blocked op. Operators cannot distinguish "never attempted" from "blocked by policy," and cannot audit who/what was refused.
5. Failure scenario: kill switch engaged during an incident → effects silently (from the framework's view) stop → post-incident review cannot reconstruct what was blocked when.
6. Consequence: the "refusals audited" half of F-06 is unimplemented; operational response during exactly the incidents kill switches exist for.
7. Reproduction: engage a switch, attempt a claim, then list ledger + audit for the op; observe total absence.
8. Class: code gap (observability).
9. Resolution: record refusals (ledger row in a refused state or, at minimum, a mandatory audit event with switch/grant/op attribution); add the refusal-visibility regression test.
10. Blocks: G1 row 5 contributor (audit completeness).

**X-14 — Shutdown-failure path closes the pool anyway; workers may still be active (MEDIUM)**
1. ID/severity: X-14 / MEDIUM.
2. Location: `cmd/miniapp/main.go:91-98` (`_ = w.executor.Shutdown(...)` then unconditional `pool.Close()`); `testkit/stack/stack.go:87-91`, `executor/dbos/dbosexec_test.go:427-431` (same shape in tests).
3. Governing requirement: prompt13 §12.8 + plan4.2 (stop and drain before pool close); prompt14 F5 (can shutdown failure still strand active workers?).
4. Defect: the ordering discipline holds only on the success path. On `Shutdown` error/timeout, every call site proceeds to `pool.Close()` — the documented deadlock (workers holding connections the close waits on). The production serve path additionally swallows the error, so the hang arrives without diagnosis. On SIGTERM with an unresponsive worker, the process hangs instead of exiting.
5. Failure scenario: deploy with a wedged step → SIGTERM → 30s drain timeout → ignored error → pool close deadlocks → orchestrator SIGKILLs (or the old revision lingers).
6. Consequence: ungraceful shutdown exactly when graceful shutdown matters; hangs mask the underlying worker fault.
7. Reproduction: shutdown with an unresponsive worker (fault-injected `Shutdown` error) and assert pool-close is skipped or loudly fatal (currently: silent close attempt).
8. Class: code defect (error handling).
9. Resolution: on shutdown failure, do not close the pool silently — log loudly at fatal level, skip the close (leak connections deliberately on the terminating path) or exit non-zero with the worker state; document the operator action; add the fault-injected regression test.
10. Blocks: Class D readiness polish (G6-adjacent operability, but the defect is in Phase 1 code).

**X-15 — Kill-switch table writable by any in-transaction miniapp handler; flips unaudited; no operator interface (MEDIUM)**
1. ID/severity: X-15 / MEDIUM.
2. Location: `migrations/000004_policy_controls.up.sql` (`app.kill_switches`, no RLS by design); `effects/effects.go:499-507` (`SetKillSwitch`, no audit); zero non-test callers of `SetKillSwitch` (verified by grep).
3. Governing requirement: prompt13 §13 (kill-switch/blast-radius enforcement central and unbypassable by a miniapp).
4. Defect: (a) any Class A handler (arbitrary miniapp SQL in-tx) can `UPDATE app.kill_switches` — engage `global` (DoS: all effects refused) or disengage a switch mid-incident — because the table has no RLS and the app role owns it. The *check* cannot be bypassed, but the *state* can be flipped by the very code it governs. (b) Flips emit no audit event. (c) No CLI/transport/operator surface exists for flipping (direct SQL only), so the tested Go API is unreachable in operation.
5. Failure scenario: compromised or buggy miniapp handler flips switches; or an operator flips via psql with no attribution trail.
6. Consequence: safety-control integrity + incident-response gaps around the control designed for incidents.
7. Reproduction: from a test handler tx, `UPDATE app.kill_switches SET engaged` and observe success (currently); attempt to flip via any delivered operator interface (none exists).
8. Class: code gap (authorization/operability) + test gap.
9. Resolution: revoke direct write from the app role (dedicated setter role or `SECURITY DEFINER` function), audit every flip, and ship a minimal operator verb (CLI `switch` command) reusing canonical auth; add handler-cannot-flip + flip-audited tests.
10. Blocks: Class D readiness polish; G1 row 5 contributor (attribution).

**X-16 — Analyzer: untested step APIs, named-vs-literal step divergence, undocumented detection boundaries (MEDIUM)**
1. ID/severity: X-16 / MEDIUM.
2. Location: `analyzer/determinism/determinism.go:185-252` (`walkCall`/`walkStepArg`); fixtures `detbad`/`detgood`.
3. Governing requirement: prompt13 §15 (seven families faithful to spec; fixtures per family), `docs/phase0/determinism-rules.md`.
4. Defect: (a) `RunAsTransaction`-in-step and `Enqueue`-in-step have no negative fixtures (only Send/Recv tested) — half of rule family 7 is unpinned. (b) Named step functions (`Ident` path) are walked with full body rules (go/select/map/global flagged), while step literals (`FuncLit` path) check only the four step APIs — the two paths implement different rules and the named path is entirely untested; one of them is wrong on any consistent reading of "workflow bodies." (c) Documented-boundary misses with no fixture or doc note: `errgroup.Go`/`reflect.Select` (not AST statements), depth-2+ and cross-package helpers (one-level same-package descent), `google/uuid` and other non-stdlib randomness, `context.WithTimeout` clock use. (d) Count nit: detbad carries 12 `want` markers, not the claimed 13.
5. Failure scenario: a workflow calling `uuid.NewString()` or a two-deep `time.Now()` helper passes strict CI and breaks replay in production.
6. Consequence: strict gate weaker than advertised; future miniapps can carry replay hazards through a green gate.
7. Reproduction: add the missing fixtures (uuid call, depth-2 helper, errgroup.Go, named step with `go`, RunAsTransaction-in-step) and observe misses/false-divergence.
8. Class: test gap + documentation defect (detection boundary unstated).
9. Resolution: add the fixtures (pinning intended behavior either way), reconcile the Ident/FuncLit paths, and document the exact detection boundary (depth, packages, randomness providers) in the analyzer doc + operations guide.
10. Blocks: Class D readiness polish (analyzer is enforcement, not runtime); G1 row 11 contributor.

**X-17 — Kill/restart parents FAIL (not skip) without a DB; remote CI has no postgres service (MEDIUM)**
1. ID/severity: X-17 / MEDIUM.
2. Location: `executor/dbos/dbosexec_test.go:778-806`, `probe/classd/probe_test.go:587+` (parents spawn workers without checking DB env; 2-minute `WaitForFile` timeout → FAIL); `.github/workflows/*.yml` (no `services: postgres`).
3. Governing requirement: prompt13 §20 (distinguish PASS/FAIL/SKIP; never claim unrun tests), G1 row 12 (CI enforces Class D rules).
4. Defect: verified by execution in this review — full `go test ./...` without `TEST_*` URLs FAILS `TestSleepAcrossRestart` and `TestProbeRestart` after 120s each (worker skips instantly; parent times out). So (a) the suite is not skip-clean: DB-less runs are red, not green-with-skips; (b) remote CI as configured (no postgres service) would not merely "not observe" enforcement — it would FAIL these two tests while skipping all 54 DB-backed tests. The report's criterion-12 PARTIAL ("wired and green locally, remote unobserved") understates: remote as-configured cannot pass at all.
5. Failure scenario: first push → red CI → pressure to "fix" by weakening tests or ignoring CI.
6. Consequence: the strict-CI story is local-only in a stronger sense than reported; no remote runner can execute the DB suites that carry the framework guarantees.
7. Reproduction: run `go test ./executor/dbos/ -run TestSleepAcrossRestart` with no DB env (observed FAIL, this review).
8. Class: test defect (missing env gate) + CI configuration gap.
9. Resolution: gate both parents on DB env (skip-fast without a DB); add a postgres service (or Neon-branch fixture) to CI so remote runs execute — not skip — the DB suites; re-run and record.
10. Blocks: G1 row 12 (cannot pass until remote CI executes or the gap is owner-accepted).

**X-18 — Evidence binds to an undigested dirty tree; cited P0 full-list entry does not exist (MEDIUM)**
1. ID/severity: X-18 / MEDIUM.
2. Location: `docs/evidence/README.md` (all prompt13 rows cite "`5cf9627` + prompt13 tree"); `docs/phase1/classd-framework-readiness.md:9-11` (cites "the P0 baseline entry … (full list …)" — verified absent by grep).
3. Governing requirement: prompt14 §4 critical evidence rule (source revision or exact working-tree digest per important claim); prompt13 §21 (exact source revision or working-tree digest per criterion).
4. Defect: no per-file hashes, no `git stash create` digest, no worktree snapshot for any prompt13 run; the "full list" entry the readiness doc depends on was never written. Whether the tree changed between the Neon pass and this review — or between the local pass and the Neon pass — is unknowable. "Local and Neon ran the same tree" is asserted, not demonstrated.
5. Failure scenario: a file edited between runs (e.g., the Neon harness fixes, which demonstrably changed test files mid-pass) silently redefines what "PASS" attaches to.
6. Consequence: every prompt13 behavioral verdict is one unrecorded edit away from unattributable; independent reproduction cannot confirm it started from the same sources.
7. Reproduction: attempt to reconstruct the Neon-run tree from the register (impossible — no digest).
8. Class: evidence gap (process).
9. Resolution: snapshot the tree now (`git stash create` or file-hash manifest), record it, and re-run at least the headline suites (effects race, dbosexec, probe, transport) at a digested state; add the missing P0-style full-list entry; adopt digest-per-run discipline (the G0 phase did this better with `e6b5b7b`).
10. Blocks: G1 row 14 (verifiability is the row's entire content).

### LOW

**X-19 — Tenant/entity join uses unescaped `tenant:local`; "cannot collide" overclaims (LOW).** `runtime/entity.go:48`, `contract/validate.go:62-67` (comment), `effects/effects.go:146`. Tenant IDs are server-configured but charset-unconstrained, and local entity values may contain `:` (`resolveEntity` bounds length only). Tenant `a` + local `b:c` and tenant `a:b` + local `c` produce the same scoped string. Impact is bounded: effects fail closed (explicit tenant predicate → confusing error, no cross-read — verified by reading), advisory locks false-share (liveness only), probe tables use separate columns (immune). Fix: constrain tenant charset at binding validation or escape the join; soften the comment. Test: same-collision-string pair across tenants.

**X-20 — Preflight orphan-FAIL misclassifies a legitimate state; manifest-side checks absent (LOW).** `preflight/preflight.go:180-182` (APPROVED + unregistered → FAIL, though approve-then-build is the natural sequence — should be WARN/SKIP), `:115-208` (no manifest/entity_key/family-version/effect-contract/OSI/required-sections checks vs prompt13 §16's list). The shipped scope (title/scope/state/reviewer/score/evidence/overclaim) matches the matrix honestly; the manifest-side items live in registry admission instead (acceptable split). Fix: downgrade orphan to WARN; document the admission-vs-preflight split explicitly.

**X-21 — Repository-gate package lists stale; pin gate misses new direct deps (LOW).** `scripts/checks.sh:32` (enterprise scan omits `executor runtime effects probe transport cmd` — the likeliest homes of a future Enterprise import), `:38` (miniapps-import scan omits `probe preflight cmd` — `cmd/miniapp` imports `miniapps/crm-read` unexamined), `:19-29` (pin list lacks `migrate`, `uuid`, `santhosh-tekuri/jsonschema`, `x/tools`), plus `docs/phase0/version-register.md` migrate row stale ("not imported" — `testkit` imports it). Fix: extend the package lists, pin the new directs, refresh the register row.

**X-22 — Empty run ID not rejected on Signal/Cancel; `getByRun` assumes uniqueness without enforcing it (LOW).** `runtime/lifecycle.go:64-81,98-124` (no emptiness check), `runtime/store.go:104-124` (`WHERE run_id = $1`, no tenant predicate in SQL — app check after — and no partial unique index on non-empty `run_id`). `Signal("")`/`Cancel("")` resolve an arbitrary same-tenant run-less row (fail-closed-ish: self-tenant only, liveness re-checked). Fix: reject empty run IDs at the service boundary; add `UNIQUE (run_id) WHERE run_id <> ''`.

**X-23 — Declared-but-unenforced contract surface (LOW).** Manifest `Timeout` (`probe` declares `"10m"`) is never read outside tests (verified by grep); `CANCELLED` effect state is unreachable (no transition site produces it); `NOT_APPLIED → ATTEMPTING` is machine-allowed (`contract/run.go:90`) but `settled()` makes it unreachable via `Perform` (reconcile→retry dead-ends at settle-time); superseded probe runs are never cancelled (no supersede→cancel linkage). None is load-bearing in Phase 1; each should be enforced, wired, or struck before production use. Fix: decide each item's fate explicitly and test it.

**X-24 — Minor hardening notes (LOW, bundled).** (a) `Status` status-sync ignores update errors (`lifecycle.go:44`) — self-heals on next read; fine, but undocumented. (b) `Invoke` resolves the registry before authenticating (`service.go:112-120`) — unauthenticated miniapp-existence oracle; swap order or accept explicitly. (c) REST has no request body limit (unbounded `Decode`) — add `MaxBytesReader`. (d) `canonicalJSON` single-`Decode` accepts trailing garbage — unreachable via schema-validated `Invoke`, but inconsistent with `sameInput`'s fail-closed posture; check `dec.More()`. (e) mockcap always records `tenant_id ""` — cross-tenant provider attribution untested. (f) `runs` routing map grows unboundedly and is process-local (fan-out fallback covers restarts).

### OBSERVATIONS (not defects)

- **O-1 — "No test weakened" is unverifiable for untracked files.** Only `contract/manifest_test.go` has a git pre-image (purely additive ✓). All increment 4–5 tests are untracked, so weakening has no baseline; the three Neon rewrites read as equivalent-or-stronger. Future phases should commit phase boundaries before hardening.
- **O-2 — Neon test branch reuse.** Phase 1 tests ran on G0's `phase0/g0-spikes` branch (TRUNCATEs, role creation) rather than a fresh branch; "destroyed after" (prompt13_imp §B) is ambiguous and undocumented in the register. G0 spike tables untouched; acceptable, but branch-per-phase would be cleaner.
- **O-3 — Hard stop fully holds.** No `lead-followup` code/manifest/registration/execution; preflight contains no bypass (generic id-based logic; DRAFT+registered FAIL verified by reading); demo uses the neutral probe.
- **O-4 — Demo attribution/model identity live outside the artifact.** The transcript proves agent-driven tool use; server-audit linkage and model/version identity are asserted in prose only. Future demos should bundle the audit query output + harness version line.
- **O-5 — Queue use needs a naming convention before first adoption.** Names are DB-global (SDK + empirical `ConflictingRegistration`); no family-prefix rule is enforced anywhere. Correctly out of Phase 1 scope (no framework queue use); do not adopt queues without one.
- **O-6 — `decide`-after-`Status`-discovered-terminal and other narrow orderings were inspected and behave sanely** (terminal replay, redelivery idempotency); no finding.
- **O-7 — Evidence-register failure dispositions are exemplary** (pool order, envelope identity, jsonb spacing, parallel truncation, migration edit, interference, Neon assumptions, MCP schemas, namespaces, env verification) — preserved as a practice worth keeping.

## F. Failure-window analysis

Verified behavior per area (code-inspected; DB execution per the agent's register, not reproduced here):

- **Submission.** Dispatch path: mint run ID → persist (live + run_id) → submit → retry re-submits same ID and attaches (executor honors `req.RunID`; SDK default reuse attaches). Crash before submit → retry creates; crash after submit → retry attaches; false-negative update → retry re-reads (self-healing). VERIFIED except: approval-path failures bypass (X-01) or strand (X-02); same-key concurrency orphans (X-03). DBOS and PG remain distinct failure boundaries — no atomicity claimed, correctly.
- **Effects and leases.** One send per business op enforced by predicated PENDING→ATTEMPTING before every send; losers get contention (fresh) or quarantine/takeover (stale, age-predicated); terminal states fenced by expected-state predicates with `ErrAttemptSuperseded` on loss. Stale-worker resume cannot send (its transition fails first) and cannot clobber (predicated settlement fails). No fencing token exists, but none is needed: the send is always preceded by a fossilizing state transition. VERIFIED except: X-05 (cancel gap), X-07 (count vacuity), X-15 (switch integrity), untested stale-resume interleaving.
- **Cancellation.** Durable record first → pending-approval settlement → executor stop → in-flight report → audit; re-cancel and Status redeliver. In-flight ATTEMPTING settles by evidence (proven by the blockChannel test). VERIFIED except: X-05 (claim gap), X-14 (shutdown-failure ordering), and the cooperative-cancel reality behind "confirmed down" (in-flight steps complete by SDK design — correctly disclosed nowhere; add one line to the ops doc).
- **Approval.** Decide (idempotent, scoped, self-approval refused, hash-bound, lazy expiry marking) → signal → workflow re-read; denial/expiry/cancel settle the run before failing the invocation. VERIFIED except: X-02 (strand), X-04 (Decide race), X-11 (no framework redelivery/bound; decider-retry + probe timeout only).
- **Durable waits/signals.** Real kill/resume with attach; buffered pre-wait signals; duplicate single-apply; post-completion harmless; suspend Enterprise-gated (behavioral); 20-sleeper envelope measured, honestly non-guarantee. VERIFIED except: 3 matrix cases missing (signal-after-timeout, signal-then-kill, unavailable-group); X-06 (false-success signals).
- **Version transition.** Per-family AppName, pinned `family/compat` versions, SDK-native affinity (source-verified), overlap/drain/restart tests, foreign-pending warning. VERIFIED except: X-09 (non-discriminating test), X-10 (PENDING-only visibility); artifact retention correctly G6.
- **Tenant/entity concurrency.** Tenant-scoped derivation + admission + predicates + genuine BYPASSRLS proof; Class A advisory serialization with peak/rendezvous proofs; probe durable-sequence uniqueness + cross-tenant independence. VERIFIED except: X-08 (attach semantics), X-19 (join ambiguity), untested sequence-claim race.

## G. Determinism analyzer assessment

| Family (spec) | Source | Neg fixture | Pos fixture | Boundary | Verdict |
|---|---|---|---|---|---|
| 1. `go` statements | `determinism.go:124-126` | `badGo` ✓ | — (no `dbos.Go` misuse case) | `errgroup.Go`, method-spawned goroutines missed | PARTIAL (X-16) |
| 2. `select` | `:127-129` | `badSelect` ✓ | `dbos.Select` accepted ✓ | `reflect.Select` missed | PARTIAL (X-16) |
| 3. map order | `:130-136` | `badMap` ✓ | index-only use clean ✓ (no order-free *range* case) | all ranges flagged (conservative); `maps.Keys` iterators missed | PARTIAL (X-16) |
| 4. nondet stdlib | `:206-210` + tables | `badTime`, `badRand`, `nowHelper` (depth-1) ✓ | step-local `time.Now` clean ✓ | depth-2+, cross-package, uuid, `context.WithTimeout` missed | PARTIAL (X-16) |
| 5. globals | `:254-275` | write + read ✓ | const + DBOS-typed read clean ✓ | immutable-read over-approx untested; DBOS-read exemption spec-unblessed but sound | PARTIAL (X-16) |
| 6. direct I/O | `:211-214` + tables | http + env ✓ | step-local I/O clean ✓ | `syscall`/`runtime` unlisted; otherwise good incl. carveouts | VERIFIED (minor residual) |
| 7. step APIs | `:202-204`, `:227-252` | Send/Recv-in-literal ✓ | — | RunAsTransaction/Enqueue untested; named-vs-literal divergence untested | PARTIAL (X-16) |

Recognition: `RegisterWorkflow` roots, same-package one-level helper descent, alias-robust type resolution — verified by reading; arg indices verified against real v1.9.0 signatures (RunAsStep[1], RunAsTransaction[2], Go[1]). Cross-package workflows/helpers silently unanalyzed (no diagnostic) — must be documented. CI: strict gate 6 wired via checks.sh, executed green locally by both agent and this review; remote unobserved (X-17). Replay tests: `TestProbeReplayDeterminism` is cross-run equivalence, not replay-across-recovery (name overclaims; restart coverage is implicit via marker counts).

## H. Generic Class D conformance assessment (§18 → test assertions)

Scale per item: E2E (through canonical service with real assertion), COMP (component-level), FAKE (simulated), INFERRED, MISSING.

| # | §18 requirement | Test + assertion | Verdict |
|---|---|---|---|
| 1 | Canonical authenticated invocation | All 12 probe tests invoke via `Svc.Invoke` with credentials; refusal paths in transport tests | E2E ✓ |
| 2 | Server-derived tenant/principal | `TestProbeCrossTenant` (independent sequences) + capture tests; tenant from binding everywhere | E2E ✓ |
| 3 | Stable run handle | `TestConfirmSubmitStableHandle` + submit idempotency (sequential); X-03 breaks the concurrent case | E2E-sequential only |
| 4 | Multiple dependent checkpointed stages | HappyPath stage 0→3 progression with sleep/eligibility/effect/signal gates | E2E ✓ |
| 5 | Durable wait | `TestSleepAcrossRestart` (real kill) + `TestProbeRestart` (E2E kill) | E2E ✓ |
| 6 | Signal wait/resume | `TestSignalReleases`, `TestSignalMatrix` (subset), probe signal legs | E2E-partial (3 timing cases missing) |
| 7 | Policy revalidation after wait | Revocation/kill/token tests — but at claim-time on demand, not *after a durable wait* in one run | COMP (no wait→revoke→refuse E2E) |
| 8 | Mock effect via shared ledger | HappyPath effect leg + `Confirmed` + audit | E2E ✓ |
| 9 | UNKNOWN via poisoned mock | `TestProbePoisonQuarantine` (lost-ack → quarantined + listed + settled) | E2E ✓ |
| 10 | Approval wait/recovery | ApprovalFlow/Timeout/RejectExits + runtime bridge tests | E2E-partial (X-02/X-11; no lazy-expiry E2E; timeout/expiry conflated) |
| 11 | Cancellation + in-flight | `TestProbeCancelMidWait` (pre-effect) + `TestCancelInFlightReported` (component, ATTEMPTING) | E2E-partial (no E2E cancel-during-ATTEMPTING; X-05) |
| 12 | Restart/replay | `TestProbeRestart` (kill + re-invoke + attach + complete, 1 send) | E2E ✓ (send-count vacuity X-07 applies) |
| 13 | Version transition w/ pending | `TestVersionOverlap` (same-process, same-code) | COMP (X-09; needs v1-down non-progress) |
| 14 | Tenant-scoped entity concurrency | CrossTenant (independent) + Concurrency (sequential attach) | E2E-partial (X-08; no barrier race) |
| 15 | Audit reconstruction | `TestProbeAuditReconstruction` (4-event chain w/ attribution) | E2E-partial (poll-driven; X-12) |

**Proves:** a neutral multi-stage workflow with sleep/signal/approval/eligibility/effect/poison/cancel/restart/cross-tenant paths executes through one canonical service with tenant isolation, ledgered effects, and attributable audit — within the fixture's modest envelope. **Does not prove:** behavior under same-key concurrency, approval-path failures, Decide races, post-cancel claims, version-discriminated affinity, unpolled settlement, or provider-side exactly-once counts. Neutrality verified (no sales semantics; hard-stop disclaimer present).

## I. Fourteen-row G1 operational matrix (independent adjudication)

Rows quote prompt10 §9; plan mapping per the readiness doc (accepted where accurate).

| # | Check (prompt10 §9) | Plan anchor | Independent verdict | Basis |
|---|---|---|---|---|
| 1 | Class A works synchronously | §6.1/§19 | **PASS** | Happy path, replay, entity serialization pair, adapters; sync inline semantics verified. |
| 2 | Class D works durably | §6.2/§11/§19 | **PARTIAL** | Waits/restart/replay/signal/version-mechanism verified; fails on X-01, X-02, X-03, X-05 (approval/submission/cancel windows). |
| 3 | Same canonical invocation service | §6.3 | **PASS** | All adapters delegate; no production bypass found; equivalence tests genuine. |
| 4 | Authorization before dispatch | §9.3/§14 | **PARTIAL** | Pre-dispatch + effect-time enforcement verified; fails on X-01 (approval bypass under failure). |
| 5 | Audit attribution complete | §9.5/§13 | **PARTIAL** | Attribution on all emitted events verified; fails on X-12 (unpolled settlement), X-13 (refusals), X-15 (switch flips). |
| 6 | Version-safe recovery | ADR-016/§11.2 | **PARTIAL** | Family/version/affinity-mechanism/overlap/drain-warning verified; fails on X-09 (discrimination), X-10 (visibility scope). Retention correctly G6, not counted against. |
| 7 | No duplicate confirmed effects | R2/§9.5 | **PARTIAL** | Ledger contention/takeover/kill/adopt verified strong; fails on X-07 (count vacuity) and X-05 (post-cancel send window). |
| 8 | Entity identity/serialization demonstrated | §9.1/§15.2 | **PARTIAL** | Contract/admission/Class A/probe-record/cross-tenant verified; fails on X-08 (attach semantics) + untested claim race. |
| 9 | Approval-pending recovery w/ mocks | §15.1/§19 | **PARTIAL** | Bridge/signal/idempotency/scope/denial-settlement verified; fails on X-02, X-04, X-11 + missing lazy-expiry E2E. |
| 10 | OpenCode E2E | §19 harness | **PARTIAL** | Genuine agent tool use incl. refusal + full lifecycle verified from transcript; fails on audit linkage + model/version-in-artifact (C17). |
| 11 | Analyzer vs fixtures | §19 | **PASS** | Fixtures green (executed), dogfood clean (executed), CI-wired (executed locally). X-16 records missing coverage as analyzer limitations, not fixture failures. |
| 12 | CI enforces Class D rules | §19 strict CI | **PARTIAL** (stronger caveat than reported) | Gates wired + green locally (executed). Remote unobserved (agent's caveat) AND remote-as-configured cannot pass (X-17: no DB service + 2 tests fail DB-less). |
| 13 | Provider writes disabled | K24/§9.5 | **PASS** | Mock-only wiring verified; credential scan executed green; no real integrations. (Gate-3 staleness X-21 noted, currently green.) |
| 14 | Evidence recorded + verifiable | §20/§22 | **PARTIAL** | Register thorough with honest failure dispositions and separated passes. Fails on X-18 (no digest; missing cited P0 entry) and X-24/O-1 (untracked unverifiable). |

Result: **4 PASS, 10 PARTIAL, 0 FAIL-as-category** (PARTIALs include blocking defects per §E "Blocks" lines — the label describes evidence state, not readiness; see §M).

## J. Security assessment

Threat review across invocation, workflow, effects, approvals, audit, DB, and transports (prompt14 §20 list worked in order):

- **Tenant spoofing:** defeated — tenant derives from server-side binding at every boundary (Invoke, resolveEntity, StrictPolicy invocation-row lookup, workflow input translation from validated requests); client input never supplies tenant. VERIFIED by reading + cross-tenant tests. Residual X-19.
- **Privileged-role RLS bypass:** mitigated by explicit predicates on all checked paths + genuine BYPASSRLS test for the highest-risk write. VERIFIED for covered paths. Residual: kill_switches intentionally unscoped (X-15).
- **Confused-deputy invocation:** no production path invokes implementations outside the canonical service (verified call-graph: transports + CLI → svc; executors trust Request by documented contract; tests only otherwise). VERIFIED.
- **Workflow payload authority substitution:** defeated — StrictPolicy resolves tool/trust from the invocation row, verifies EffectToken, ignores Intent-carried authority; `TestServerDerivedPrincipal`/`TestEffectTokenMismatch` cover it. VERIFIED by reading.
- **Cross-tenant effect-ID collision:** defeated by construction (prefix gate + explicit predicates + global-PK fail-closed). VERIFIED by reading + `TestCrossTenantOpIDs`.
- **Duplicate submission, altered input:** defeated sequentially (`KeyConflict` terminal). VERIFIED. Concurrent variant open only via X-03 (orphan, not confusion).
- **Concurrent effect claims:** defeated (predicated transitions + lease rule + genuine 8-way test). VERIFIED. Residual X-05 (cancel race), X-07 (counting).
- **Lease takeover, stale worker alive:** defeated for sends (transition-before-send ordering + predicates) — verified by reading; untested interleaving noted (test gap, mechanism sound).
- **Approval after cancellation:** defeated sequentially (`approval-cancelled` + `CancelPending` + status gate + test). Concurrent variant OPEN (X-04).
- **Approval signal loss:** mitigated by decider-retry + Recv-timeout + re-read; NOT bounded by framework mandate (X-11).
- **Expiry without poller:** workflow parks until its own Recv timeout (probe convention) or forever (unbounded miniapp Recv); lazy observation needs a poll or Decide. OPEN per X-11.
- **Cancellation/effect races:** pre-claim and in-flight-settle paths verified; post-record claim window OPEN (X-05).
- **Old-version workflow on incompatible code:** prevented by SDK affinity (source-verified) + pinned versions + admission agreement; behaviorally undiscriminated (X-09); visibility partial (X-10).
- **Queue namespace collision:** no framework queue use; convention needed before adoption (O-5). Empirical + SDK evidence correctly scoped as evaluation.
- **Run handle exposure across tenants:** `getByRun` verifies tenant explicitly; cross-tenant lookups fail as unknown (tested). VERIFIED. Residual X-22 (empty-ID handling).
- **Secrets into DBOS inputs/checkpoints/logs/audit:** payload discipline asserted by probe tests (IDs-only inputs); audit `Details` redacted-payloads-only by convention (no test scans audit for secrets — minor gap, LOW: add a secret-pattern scan over recorded audit in one E2E test); MCP credential-in-args is demo scaffolding with redacted transcript (verified). No DSN/credential in tree (gate 5 executed green).
- **MCP stdio pollution:** serve path logs to stderr; SDK stdio on stdout; demo ran stdio successfully (transcript exists and parses). VERIFIED as far as statically possible.
- **Migration partial failure:** forward-only repair honestly handled (000005); down-cycle evidence agent-side only; down-file index-schema nit (LOW, §K). NOT re-run here per restrictions.
- **Preflight bypass:** no bypass found (generic logic, hard-stop FAIL, no hardcoded exceptions); operator-flag evasion is operator choice, not a bypass. VERIFIED.
- **Untrusted input into SQL/identity/capability:** SQL fully parameterized (verified by reading all queries — no string-interpolated values except static identifiers in tests/roles with fixed names); workflow IDs runtime-minted; entity values length-bounded (charset open, X-19); capability payloads pass through to mock only.

**Residual risks (accepted or Phase-bound):** volumetric/allowlist/circuit controls absent per OSI-4 (fail-closed verified — no silent unlimited); production reconciler absent per plan (quarantine + operator settle delivered); retention/absent alerting documented as pre-production requirements (ops doc honest); Enterprise boundaries respected (suspend test proves refusal; no Enterprise imports).

## K. Database and migration integrity

- **History:** 000001–000002 (increments 4–5) + 000003 (invocation link) + 000004 (kill switches, trust/token, payload) + 000005 (forward-only payload backfill). The applied-migration edit incident is honestly disclosed with the correct forward-only remedy; 000005's intentional down-noop is correct (avoids dropping 000004-owned column on fresh DBs).
- **Fresh install:** all statements `IF NOT EXISTS`; convergent. **Upgrade:** early-000004 DBs gain the column via 000005; complete-000004 DBs no-op. Verified by reading. Down/up cycle evidence is agent-side only (NOT re-run here per §3 restrictions).
- **Down-file nit (LOW):** unqualified `CREATE INDEX` lands in `public`, but down files address `app.*` names → `DROP INDEX IF EXISTS` silently misses (000003; same pre-existing pattern in 000001). Functionally convergent (re-up skips on existing index), but downs don't fully revert. Schema-qualify index names.
- **Constraints match the concurrency model:** invocation idempotency unique (tenant, miniapp, version, tool, key) ✓; effect PK global with tenant-prefix enforcement + explicit predicates ✓; probe partial-unique active generation ✓. Missing: partial unique on non-empty `run_id` (X-22).
- **RLS:** FORCE + tenant policies on app tables ✓; audit USING-only (documented write-availability choice) ✓; kill_switches intentionally unscoped (X-15); probe/mock tables follow suit (mock log global by provider-simulation design).
- **Pool context:** `WithTenant` unchanged (tx-local `set_config`); G0-5 concurrency evidence stands; no session-state pooling hazards introduced. LockEntity is `pg_advisory_xact_lock` (tx-scoped, cannot cross waits) keyed on the scoped string — 32-bit `hashtext` false-sharing noted (LOW: consider 64-bit).
- **Library separation:** DBOS system tables SDK-owned; app tables `app.*`; probe/mock tables test-owned; no app SQL touches DBOS internals. ✓
- **Dependencies:** `go.mod` adds migrate (library use — register row stale, X-21), uuid (promoted direct, correct), santhosh/jsonschema (runtime validation, registered), x/tools v0.52.0 (go floor 1.26.0 = CI's 1.26 ✓ compatible). Indirect bumps (oauth2, protobuf) transitive. No Enterprise DBOS modules (verified: gate 3 green + import grep scope noted in X-21). No binaries/artifacts in tree (verified by status + find for executables: none added).

## L. Phase boundaries and Open Spec Items

OSI-1 (playbook), OSI-2 (role vocabulary), OSI-3 (no-entity variance), OSI-4 (volumetric semantics) all preserved OPEN with no invented values — verified: scope enforcement is mechanism-generic with synthetic `tenant-ops` only in tests; admission rejects no-entity effectful tools per OSI-3 impact; volumetric enforcement absent and fail-closed (no silent unlimited — verified: no counters/amount/allowlist/breaker code paths exist). No owner decisions invented anywhere in implementation or docs. The §22 pilot/reviewer/synthetic decisions remain open and correctly scoped to increment 6, not framework readiness. One correction: the matrix's F-04 line implies WS-J will define the variance representation — WS-J is the preflight workstream, which does not define it (preflight has no variance representation); reassign OSI-3's future owner to registry/admission design, not preflight.

Phase discipline verified: no River/Class C (dependency present, unused outside spikes), no Sprites, no promotion, no Fly/G6, no real providers/OAuth, no production reconciler (Settle is operator/test API, correctly not a worker), no UI/generator. The probe stays neutral and test-only (serve-binary import gated by `MINIAPP_DEMO_PROBE` with a loud warning — acceptable).

## M. Final recommendation

**Require bounded remediation and re-review. Do not accept as-is. Do not reopen any architecture decision. Do not hold G1 for G6/remote-CI/playbook scope beyond what is specified below.**

Mandatory before G1 (code + tests, all bounded): X-01 (approval re-entry on confirmSubmit), X-02 (submit-if-missing on poll), X-03 (predicated run_id adoption), X-04 (Decide row lock/predicate), X-05 (atomic claim liveness), X-06 (signal existence check), X-07 (per-attempt provider rows), X-09 (discriminating affinity test), X-11 (mandated approval-wait bound + commit-time decision audit), X-12 (completion-hook settlement), X-13 (refusal records), X-15 (switch-table hardening + operator verb + audit), X-17 (DB env gates + CI postgres service), X-18 (tree digest + headline re-runs + P0 entry). Strongly advised in the same pass: X-08, X-10, X-14, X-16, X-19–X-24 (all small).

What is explicitly NOT required for G1: cross-release artifact retention (G6 drill, correctly deferred), remote CI observation beyond a passing configured run (X-17 fix + one green remote run suffices), production playbook or `lead-followup` decisions (OSI-1 + §22 stay open), volumetric enforcement (OSI-4), production reconciler (Phase 4), River/Sprites/promotion/Fly (later phases), Enterprise features.

Re-review scope: the remediated boundaries only (approval/submission paths, Decide concurrency, claim gating, signal existence, provider counting, affinity discrimination, audit completeness points, CI service, digested re-runs) — not a fourth full reading. The framework's design direction is right; the remaining work is failure-path correctness at boundaries the current suite does not cover. A small probe with correct failure semantics remains the goal — the probe is already small; this review itemizes exactly where its semantics are not yet correct.

## N. Review change inventory

- Added exactly one file: `docs/adv_review_prompt14.md` (this report; user-specified path — see header note re prompt14 §3/§22's `docs/phase1/` path, which was not created).
- Modified: nothing. Deleted: nothing. Committed/pushed: nothing. Stash: untouched (empty).
- Executed (read-only / safe): `gofmt -l .` (clean), `go vet ./...` (clean), `go test ./analyzer/... ./preflight/... ./contract/...` (pass), `go test ./...` (pass-or-skip except X-17's two DB-less FAILs), `bash scripts/checks.sh` (ALL-CHECKS-PASS — includes analyzer dogfood + live preflight), SDK-source reads from a disposable module cache (`/tmp`, outside the tree), transcript JSON parsing, `git status/log/diff/branch/remote` reads.
- Not executed: any DB-backed test (no Docker, no URLs, no Neon access — nothing to run against); any migration up/down; any workflow/process-kill test; any Fly/Neon mutation; any OpenCode session; global OpenCode config untouched (never read).
- Working tree after review: identical to before except this report (verified: `git status --short` shows only the pre-existing 10 modified + untracked set plus this file).
