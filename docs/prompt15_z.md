# Prompt 15 — Root-Cause Remediation of the Prompt 14 Adversarial Review

| Field | Value |
|---|---|
| Role | Implementation agent: Go runtime, PostgreSQL, distributed-systems engineer |
| Repository | `github.com/PithomLabs/miniapp` |
| Governing authority | `docs/plan4.2.md` (immutable), G0 adjudication, `docs/prompt13.md`, `docs/prompt10.md` |
| Review being remediated | `docs/adv_review_prompt14.md` — verdict **NOT ACCEPTABLE (bounded and remediable — not a redesign)** |
| Inputs | `docs/prompt13_imp.md`, `docs/evidence/README.md`, `docs/phase1/classd-capability-matrix.md`, `docs/phase1/classd-framework-readiness.md` |
| Deliverables | `docs/phase1/invocation-lifecycle-contract.md`, remediated code/tests/migrations/CI, `docs/prompt15_imp.md`, updated evidence register and matrices |
| Date | 2026-11-23 |

---

## 1. Role and mission

Prompt 14's independent adversarial review found five HIGH defects (X-01 through X-05), all breaking core lifecycle invariants on failure paths and interleavings the test suite never covers; thirteen MEDIUM defects; six LOW defects; and an independently adjudicated G1 matrix of 4 PASS / 10 PARTIAL against the reported 13 of 14. The review also established that the core design is sound and that every finding has a bounded fix at an identified boundary — no architecture decision reopens.

**This is a remediation round, not a patch round.** Do not fix findings one by one. The five HIGH findings are one root cause wearing five costumes; the MEDIUM evidence findings share a second root cause. Your job, in order:

1. Ratify or refute the root-cause synthesis in §3 (if you refute it, say why, in writing, before touching code).
2. Establish the transition-integrity contract it implies — first as a normative document (§6), then as code.
3. Close every finding by mapping it to a contract rule and a discriminating test that fails without the fix.
4. Report honestly (§11) for an independent re-review.

## 2. Standing post-review principle (standing text — lift verbatim into Prompt 16 and every future post-review prompt)

After every adversarial review, before any code change:

1. **Cluster, don't tally.** Group all findings into defect classes and identify the systemic mechanism that produced every instance in the class. A finding is a symptom; the missing contract is the cause.
2. **Derive the contract.** Write the invariant, or small set of invariants, whose uniform enforcement eliminates the entire class — stated as enforceable rules (predicate, atomicity scope, enforcement boundary), not as advice.
3. **Map, then remediate.** Every finding maps to exactly one contract rule plus one discriminating test. Remediate against the contract. A fix that closes an instance while leaving the class reachable from another path is not accepted.
4. **Closure is red/green, not inspection.** A finding is closed only when a test is shown to fail with the enforcement removed and pass with it restored — both runs recorded, bound to source digests. "The code now looks right" is not closure.
5. **Re-review audits the contract.** The next independent review verifies the contract document, its enforcement in code, and class elimination — not merely that each finding ID changed state.

## 3. Root-cause synthesis (ratify before coding; normative for this round)

### RC-1 (runtime): invariants enforced by path shape, not by predicated atomic transitions

The Phase 1 lifecycle code enforces its guarantees by *procedural sequencing and comments* instead of by *predicated, atomic transitions on durable rows*. Four mechanisms produce all five HIGH findings:

- **RC-1a — Non-predicated writes (last-writer-wins).** Contested durable fields are written by unconditional `UPDATE`. X-03: `runtime/store.go:126-138` overwrites `run_id` unconditionally, so two racing same-key submits mint two run IDs and the loser's overwrite orphans a live, unaddressable workflow. X-04: `approvals/approvals.go:104-140` reads without a row lock and updates without a state predicate, so concurrent conflicting `Decide` calls (and decide-vs-cancel) silently overwrite; self-approval can land through the race.
- **RC-1b — Check-then-act across a boundary.** A guard is verified in one transaction or lock scope; the act it guards happens in another. X-05: `runtime/lifecycle.go:226-255` releases the `FOR UPDATE` lock when its transaction commits, before the policy check, claim INSERT, attempt transition, and send in `effects/effects.go:165-181` — so a cancellation recorded in the gap does not stop a new attempt, violating the Workstream D invariant ("after cancellation is durably recorded, no new consequential effect attempt may begin"). The mint window inside X-03 is the same shape: guard (claim) and act (mint + record) in separate scopes.
- **RC-1c — Authorization enforced by path ordering instead of at the execution boundary.** The approval gate lives in `requestApproval`'s sequencing (`runtime/service.go:199-235`); the submit transition itself (`confirmSubmit`, `runtime/service.go:354-369`) checks nothing. Three failure triggers leave a gated invocation `Running` with no run — (a) `Approver.Submit` fails, (b) the `WaitingApproval` park update fails, (c) crash between claim commit and park — and the same-key retry then routes `Running` Class D records (`runtime/service.go:186-194` → `:163-167`) into `confirmSubmit`, which submits with `ApprovalID: ""` and a fresh `InvokeRequest`, so the workflow skips its approval wait and executes consequential effects with no approval requested, returned, or recorded. The gate must live in the submit transition, not in one path's ordering.
- **RC-1d — Failure branches are not states of the machine.** Error exits strand records in states with no convergence, or lie about outcomes. X-02: a failed `submitRecorded` after park leaves the record `WaitingApproval` naming a run that was never submitted (or, if the park audit failed, no run at all); `pollApproval` (`runtime/service.go:243-287`) polls but never submits, so polls wait forever, grant fails forever, denial retries `settleDenied` forever. X-06: `Signal` returns success for nonexistent runs (`executor/dbos/executor.go:291-307` fan-out returns on first nil; SDK `sysdb.Send` checks nothing), so stranded runs are masked and orphan notifications accumulate. X-11: decisions are audited only after a successful signal, so a committed-but-unsignalled decision is unaudited. X-12: settlement audit appears only when someone calls `Status`; never-polled runs have permanently incomplete audit chains. X-13: refused effect claims leave no ledger row and no audit event, so kill-switch and revocation refusals are invisible to operators.

### RC-2 (evidence): tests and records built to demonstrate, not to falsify

- X-07: the mock provider log dedupes by `op_id` (`ON CONFLICT DO NOTHING`, `testkit/mockcap/mockcap.go:132-136`), so `SentCount == 1` proves "at least one acceptance" regardless of how many sends occurred — every provider-side "exactly 1 send" assertion in takeover, kill-recovery, restart, and the probe is vacuous (the 8-way race test is unaffected; it uses an in-process counter).
- X-09: `TestVersionOverlap` runs identical code with both workers alive; step dedup guarantees the same marker count with or without affinity, so the test cannot discriminate (the real proof today is the pinned-SDK source read).
- X-17: the kill/restart test parents FAIL (not skip) without a DB, and remote CI has no postgres service — remote CI as configured can never pass the suites that carry the framework guarantees.
- X-18: every prompt13 behavioral claim binds to "`5cf9627` + prompt13 tree" — no digest, no per-file hashes — and the readiness doc cites a P0 full-list baseline entry that does not exist.
- Underlying both: race-sensitive code was tested sequentially almost everywhere the HIGH findings live (submit, Decide, claim-vs-cancel, sequence claim), and failure branches were untested everywhere RC-1d lives.

**Why five "PROVEN" rows shipped with one root cause:** the suite covers success paths, sequential retries, and crash-recovery-once; it never covers interleavings (concurrent same-key submit, concurrent Decide, cancel-during-claim) or failure branches (park failure, submit-after-park failure, lost signals). §7 and §10 make both coverage classes mandatory for every fix.

### The transition-integrity contract (normative; materialized per §6)

1. **Predicated single-writer transitions.** Every durable lifecycle state change is a compare-and-swap: `UPDATE ... WHERE id = ... AND status = 'expected'` with an affected-row check; on 0 rows, reload and reconcile — never an unconditional overwrite. `run_id` is adopted, not overwritten: `UPDATE ... WHERE run_id = ''`, reload on 0 rows affected and adopt the winner's run.
2. **Guard-and-act in one atomic scope.** The guard verified in one transaction must gate the act in the same transaction or under the same held lock: liveness is verified atomically with the claim INSERT (conditional INSERT predicated on the live invocation row, or the row lock held through the claim), and `Decide` arbitrates under a row lock or predicated update inside one scope.
3. **Authorization at the execution boundary, through one choke point.** All executor submissions route through a single choke point that takes the approval linkage as an argument and refuses to submit any approval-gated tool without a resolvable granted-approval linkage — re-entering approval handling (re-request or refuse, never submit blind) when linkage is missing. No alternate path, retry included, may reach executor submission except through the choke point.
4. **No dead-end states.** Every non-terminal state has, by construction, a bounded convergence path: the approval poll submits-if-missing (reusing the stored approval linkage) before polling; failed parks converge to retryable errors or recoverable states; every stranded condition is discoverable, operator-actionable, and testable.
5. **Truthful failure reporting and commit-time audit.** Operations verify the object exists before claiming success (Signal and Decide on unknown runs error); decisions are audited at commit time, not after notification; settlement is emitted by the completion path, not only by reads; refusals are recorded with attribution.
6. **Falsifiable evidence.** An invariant claim ships only with a test demonstrated to fail when the enforcement is absent (red/green recorded); test instruments record reality (every provider attempt logged, no dedup-by-op_id on the log); every behavioral claim binds to a source digest per run.

## 4. Governing authority and precedence

Unchanged from Prompt 14 §2, in the same order: `docs/plan4.2.md`; accepted G0 decisions and recorded owner decisions; `docs/prompt13.md`; approved contracts and ADRs; repository implementation and executed evidence; `docs/prompt13_imp.md` and the Prompt 14 review as claims about the tree. The review report (`docs/adv_review_prompt14.md`) is **normative for remediation targets** — its locations, failure scenarios, and resolution fields (§21 fields 2, 5, 9) are the authoritative statement of what must change — but it does not amend the architecture. `docs/plan4.2.md` remains immutable. If any fix appears to require an architecture change, stop and write the owner decision instead; the review's verdict is that none does.

## 5. Scope, restrictions, and hard stop

**Authorized:** modify implementation code (`runtime/`, `approvals/`, `effects/`, `executor/`, `contract/`, `internal/store/`, `cmd/`, `transport/`, `analyzer/`, `preflight/`, `testkit/`, `probe/`, `scripts/`, `.github/workflows/`), add tests and analyzer fixtures, add migrations where a fix requires one (X-15, X-22), update `scripts/checks.sh`, update `docs/evidence/README.md`, `docs/phase1/classd-capability-matrix.md`, `docs/phase1/classd-framework-readiness.md`, and the version register where findings require it.

**Forbidden (unchanged from Prompt 14):**

- **Hard stop:** no `lead-followup` code, manifest, registration, execution, or ADR approval; the ADR stays DRAFT and unapproved; the neutral probe stays neutral.
- No modification of `docs/plan4.2.md`, `docs/prompt13.md`, `docs/prompt13_imp.md`, `docs/adv_review_prompt14.md`, `docs/faq.md`, or prior review documents.
- No commits, no pushes; all work remains uncommitted, as in prior rounds.
- No real provider calls or external writes; mock-only channels.
- No changes to the shared or production Neon project or branch; disposable databases only (local Docker PostgreSQL or a disposable Neon branch); no destructive migration experiments against shared state.
- No weakening of existing tests to make suites pass: X-17's fix is skip-fast environment gating, not test deletion; predicates added under this round may tighten assertions, never loosen them.
- No new miniapps; no Enterprise DBOS modules; no new credentials or DSNs in the tree.

## 6. First deliverable: the contract document (before code)

Write `docs/phase1/invocation-lifecycle-contract.md` before changing any code. It must contain:

1. **The transition table**: every lifecycle state (`created`/claimed, `Running`, `WaitingApproval`, terminal states, effect states `PENDING/ATTEMPTING/CONFIRMED/FAILED/UNKNOWN`, approval states) crossed with every event (invoke, same-key retry, park, submit-after-park, grant, deny, expire, decide, concurrent decide, cancel, decide-vs-cancel, check-live, claim, send, settle, signal, status) → next state, **the predicate** (the exact `WHERE` clause or lock scope), **the atomicity scope** (one transaction? lock held through which statement?), **the authorization precondition** enforced at that transition, **the failure exit** (what happens when the transition's writes fail or crash), and the owning component.
2. **The choke-point statement**: the single function through which every executor submission passes, its signature (approval linkage in, no linkage → refuse or re-enter approval), and the grep/probe that proves no other call site submits.
3. **Per-row falsification**: each rule row names the test ID that fails when the enforcement is removed.
4. **Coverage of the review's failure windows** (Prompt 14 §F): submission, effects/leases, cancellation, approval, durable waits/signals, version transition, tenant/entity concurrency.

Rules of engagement: no code change ships for a transition not in the table; no table row ships without its discriminating test. This document is the re-review's primary audit artifact.

## 7. Required remediations — HIGH (each maps to one contract rule; closure per §2.4)

### X-01 — Approval bypass on retry (Rule 3: authorization at the boundary)

- **Fix:** `confirmSubmit` (`runtime/service.go:354-369`) must never submit an approval-gated tool without a resolvable granted-approval linkage. Route all executor submissions through the single choke point of §6.2; when a gated tool arrives with no linkage, re-enter approval handling — adopt the pending approval row left by the failed park if one exists (X-01's dangling-row case), otherwise re-request or refuse, never submit blind. The `replay → Running → confirmSubmit` routing (`runtime/service.go:186-194`, `:163-167`) must pass through the same gate.
- **Tests (all three triggers):** fault-inject `Approver.Submit` to fail once on a gated invoke → same-key retry → assert no submission without approval linkage and no workflow without `ApprovalID` in run input; fault-inject the park update to fail once → same; crash between claim commit and park (kill test) → retry → same. Assert exactly one `approval-requested` audit event per invocation and full audit reconstruction of the request.
- **Acceptance:** no path exists — current or future — by which a gated tool reaches effects without a granted approval; proven by the choke-point proof (§6.2), not by patching the one replay route.

### X-02 — Approval-path failure strands the invocation (Rule 4: no dead-end states)

- **Fix:** `pollApproval` (`runtime/service.go:243-287`) submits-if-missing before polling — reusing `confirmSubmit` semantics with the stored approval linkage — so a record parked without a run converges when the decision arrives; denial or expiry of a never-submitted run settles the invocation terminally instead of looping `settleDenied`/`cancelRun` forever; a failed park audit leaves the record recoverable, never silently permanent.
- **Tests:** fail submit-after-park once on a gated invoke → retry → human approves later → poll converges and the grant takes effect on the actually-submitted run; fail the park audit once → record recoverable per the contract table; deny a never-submitted run → invocation settles terminal with an honest status.
- **Acceptance:** no reachable state in which a record waits forever on a run that does not exist; grant, denial, and expiry all converge within bounded, tested mechanisms.

### X-03 — Concurrent same-key submits orphan a workflow (Rule 1: predicated single-writer)

- **Fix:** predicated `run_id` adoption in `runtime/store.go:126-138` (`UPDATE ... WHERE run_id = ''`, reload on 0 rows affected and adopt the winner's run) or `SELECT FOR UPDATE` on the invocation row across mint + submit-record (`runtime/service.go:335-369`); the loser submits the winner's run ID (executor attach semantics) rather than minting its own.
- **Tests:** barrier-synced concurrent same-key double `Invoke` against the real executor — assert exactly one workflow exists, the record names it, both callers receive valid handles to the same run; run with the race detector.
- **Acceptance:** concurrent same-key submits cannot produce two workflows; orphaning is impossible by construction, and the existing sequential tests (`TestSubmitFailureRetryAttaches`, `TestConfirmSubmitStableHandle`, `TestSubmitIdempotent`) gain concurrent counterparts.

### X-04 — Concurrent conflicting Decide calls last-writer-win (Rules 1 + 2)

- **Fix:** `Decide` (`approvals/approvals.go:104-140`) arbitrates under `SELECT ... FOR UPDATE` or a predicated `UPDATE ... WHERE status = 'pending'` with an affected-row check, inside one scope; the same predicate discipline applies to the expiry-marking update and to `CancelPending`; decision-after-cancel is refused deterministically.
- **Tests:** barrier-synced approve + reject on one approval — exactly one wins, the loser errors; barrier-synced `Decide` + `CancelPending` — deterministic outcome, no overwrite, post-cancel decision refused; requester + approver race — self-approval cannot land through timing. Race detector on.
- **Acceptance:** authorization decisions cannot be substituted by interleaving; every previously sequential `Decide` test has a concurrent counterpart.

### X-05 — Cancel-vs-claim window breaks the Workstream D invariant (Rule 2: guard-and-act atomic)

- **Fix:** liveness is verified atomically with the claim — conditional claim INSERT predicated on the live invocation row, or the `CheckLive` row lock held through the claim INSERT (`runtime/lifecycle.go:226-255`, `effects/effects.go:165-181`); correct the "linearizes" comment to describe the actual mechanism, per the contract table.
- **Tests:** gated race test — block inside `CheckPolicy` (or pre-INSERT), commit a concurrent `Cancel`, release the gate — assert the claim refuses once the cancel commits; assert the cancel response's `InFlight` list cannot miss a claim that began after the cancel, per the contract's ordering rows.
- **Acceptance:** no interleaving exists in which a new consequential effect attempt begins after cancellation is durably recorded — the invariant holds by predicate, not by ordering or comment.

## 8. Required remediations — MEDIUM (X-06 through X-18; fix + discriminating test each)

- **X-06 (Rule 5):** `Executor.Signal` verifies run existence (`RetrieveWorkflow`) before `Send` (`executor/dbos/executor.go:291-307`); tests: signalling a bogus run ID errors (currently nil); post-restart signal with an emptied routing map delivers to the owning family.
- **X-07 (Rule 6):** `testkit/mockcap/mockcap.go:132-136` logs every attempt as its own row (serial PK, `attempted_at`, outcome; no `ON CONFLICT (op_id) DO NOTHING` on the log); fix the false comment (`:77`) and the surviving exactly-once phrasing (`effects_test.go:18`); re-run the affected suites (`TestStalePendingTakeoverSendsOnce`, kill test, probe send-count tests); discriminator: a deliberate double-`Send` now yields 2 rows (the old log yields 1).
- **X-08 (decide-and-document):** either implement true run-attach (loser invocation references the winner's run ID; status follows the winner) or re-scope to detect-and-exit with explicit lifecycle semantics — the loser's terminal status must no longer read `Succeeded` while the sequence is still active (`probe/classd/sequence.go:141-146`) — plus a supersede→cancel linkage. Either way: a barrier-synced claim-race test replaces the sequential `TestProbeConcurrency` (`probe_test.go:392-441`) and a supersede test is added. Record the choice as an owner-decision item; the misleading status is fixed regardless of the option chosen.
- **X-09 (Rule 6):** discriminating affinity test — shut down the v1 worker with v1 pending runs, leave v2 up, assert v1's pending run does NOT progress (or per-version step markers asserting executor identity) (`dbosexec_test.go:621-662`); the pinned-SDK citation becomes mechanism explanation, not proof.
- **X-10:** `reportForeignPending` (`executor/dbos/executor.go:143-164`) covers all non-terminal states (drop the `PENDING`-only filter; group by version/state) and exposes queryable counts or documents the log-pattern alert contract; the overlap test seeds ENQUEUED + SUSPENDED foreign-version runs and asserts full coverage.
- **X-11 (Rules 4 + 5):** mandate the recovery bound in framework code — pick exactly one and enforce it: a required Recv-timeout ceiling for approval waits, a reconcile sweeper, or poll-driven renotify; audit the decision at commit time, not after signal (`runtime/lifecycle.go:180-219`); strike/replace the nonexistent "durable mailbox" phrase in the capability matrix. Tests: decide-then-kill-before-signal with a no-retry decider and a long-Recv workflow → bounded observation of expiry (currently unbounded); long-wait + lost-signal regression test.
- **X-12 (Rule 5):** emit `run-settled` from a completion hook or executor callback on terminal discovery, not only from `Status` reads (`runtime/lifecycle.go:43-48`); test: complete a run with no `Status` call and assert the audit chain contains settlement.
- **X-13 (Rule 5):** record effect-claim refusals — a refused-state ledger row or, at minimum, a mandatory audit event with switch/grant/op attribution (`effects/effects.go:165-170`); test: engage a switch, attempt a claim, list ledger + audit, assert the refusal is attributable and distinguishable from "never attempted."
- **X-14:** on shutdown failure do not silently close the pool (`cmd/miniapp/main.go:91-98`, `testkit/stack/stack.go:87-91`): log loudly at fatal level, skip the close or exit non-zero with worker state; document the operator action; fault-injected shutdown-failure regression test.
- **X-15:** revoke direct write on `app.kill_switches` from the app role (dedicated setter role or `SECURITY DEFINER` function; migration against `migrations/000004_policy_controls.up.sql` lineage, validated on disposable DBs only); audit every flip (`effects/effects.go:499-507`); ship a minimal operator verb (CLI `switch` command) reusing canonical auth. Tests: a handler transaction cannot flip a switch; every flip is audited; the operator verb works end to end.
- **X-16:** add the missing analyzer fixtures (RunAsTransaction-in-step, Enqueue-in-step, `uuid`/non-stdlib randomness, depth-2 helper, `errgroup.Go`, named step function containing `go`), reconcile the `Ident` vs `FuncLit` rule divergence (`determinism.go:185-252`), document the exact detection boundary (depth, packages, randomness providers) in the analyzer documentation, and fix the detbad 12-vs-13 count discrepancy.
- **X-17:** gate both kill/restart parents on DB environment with skip-fast (`dbosexec_test.go:778-806`, `probe_test.go:587+`); add a postgres service (or disposable Neon-branch fixture) to the CI workflows so remote runs execute — not skip — the DB suites. You cannot push; record remote execution as owner-observed-pending and leave G1 row 12 PARTIAL unless remote execution is actually observed. After this fix, a DB-less `go test ./...` must be skip-clean (verified by running it).
- **X-18 (Rule 6):** snapshot the tree digest before changes and after remediation (`git stash create` or file-hash manifest); produce the missing P0-style full-list entry the readiness doc cites (`docs/phase1/classd-framework-readiness.md:9-11`); re-run the headline suites (effects race, dbosexec, probe, transport) at a digested state; adopt digest-per-run discipline for every new evidence entry.

## 9. LOW findings — explicit dispositions required (X-19 through X-24)

For each: fix it, or document-and-defer with a one-paragraph justification recorded in the report. None may be silently dropped. Recommendations (you may defer with reasons, but silence is not a disposition):

- **X-22 — recommend fixing:** reject empty run IDs at the service boundary (`runtime/lifecycle.go:64-81`, `:98-124`) and add `UNIQUE (run_id) WHERE run_id <> ''` — cheap, and it closes an X-03-adjacent hole.
- **X-21 — recommend fixing:** extend `scripts/checks.sh` package lists (`:32`, `:38`), pin the missing direct deps (`:19-29`), refresh the stale `docs/phase0/version-register.md` migrate row.
- **X-24(b), (c) — recommend fixing:** resolve the registry only after authentication (`service.go:112-120`) and add a `MaxBytesReader` to REST — both align with the boundary discipline of Rule 3.
- **X-19:** constrain tenant charset at binding validation or escape the `tenant:local` join, soften the overclaiming comment, add the collision-string test.
- **X-20:** downgrade the preflight orphan-FAIL to WARN and document the admission-vs-preflight split explicitly.
- **X-23:** decide each item's fate explicitly — manifest `Timeout` enforcement, `CANCELLED` reachability, `NOT_APPLIED → ATTEMPTING`, supersede→cancel linkage — and enforce, wire, or strike each with a test; "declared but unenforced" is not a shippable state for production use.
- **X-24 (remaining):** document or fix each bundled item (status-sync error swallow, `canonicalJSON` trailing-garbage check via `dec.More()`, mockcap `tenant_id` attribution, `runs` routing map growth).

## 10. Test-integrity rules for this round

1. **Red/green proof:** for every HIGH fix and for X-07, run the new test with the enforcement reverted (must fail) and restored (must pass); record both runs — command, environment, digest — in the report. Inspection-only closure is rejected.
2. **Barrier-synced races:** X-03, X-04, X-05, and X-08's claim race use the gate-channel pattern of the existing 8-way test; run under `go test -race`.
3. **No weakening:** existing assertions may tighten, never loosen; X-17's gating is skip-fast, not deletion; the DB-less suite must be skip-clean after the fix.
4. **Real execution:** every new discriminating test runs against real PostgreSQL (local Docker or a disposable Neon branch) with the environment recorded per run — DB identity, exact command, pass/fail/skip counts. Never claim an unrun test; a claim without a digest is a claim without a test.
5. **Evidence register:** every new entry carries its tree digest; the missing P0-style full-list entry is produced; failure dispositions stay honest (Prompt 14 O-7 remains the standard).
6. **Phase boundary:** recommend the owner commit the remediated boundary before further hardening (O-1); you still do not commit.

## 11. Report format — `docs/prompt15_imp.md`

- **A. Self-assessed verdict** (final verdict belongs to the re-review; recommend CONDITIONALLY ACCEPTABLE or better only if every HIGH finding is closed with red/green evidence).
- **B. Root-cause → contract mapping:** every finding → defect class (RC-1a/1b/1c/1d or RC-2) → contract rule → enforcement location → test ID.
- **C. Per-finding closure:** X-01 through X-24 plus the F-01–F-14 residuals named in the review (§D), each with red/green evidence or an explicit, justified disposition.
- **D. The contract table as shipped**, with test IDs per row and the choke-point proof.
- **E. Re-adjudicated matrices:** capability matrix F-01–F-14 and the G1 operational rows, re-labelled honestly — rows whose proof remains unobserved (remote CI execution) stay PARTIAL; do not upgrade a row without its evidence.
- **F. Change inventory:** exact modified/added paths matching `git status` — report the real tracked-file delta (the Prompt 13 report's "3 modified paths" vs. the actual 10 is the cautionary example).
- **G. Evidence register updates** with digests and per-run environments.
- **H. Residuals and owner decisions:** the X-08 option chosen, the X-11 bound mechanism chosen, the X-15 operator surface, remote-CI observation (owner action), and every §9 deferral with its justification.

## 12. Discipline

- **Fix the class, not the instance.** A fix that closes X-01 while `confirmSubmit` remains reachable without approval linkage from any other path has not closed RC-1c; the choke-point proof, not the patch, is the deliverable.
- **Bounded fixes only.** All fixes land at the boundaries the review identified. If a fix appears to require an architecture change, stop and write the owner decision instead.
- **No scope creep.** New issues discovered while fixing are recorded as findings and fixed only when they block a required fix; everything else goes to §11.H.
- **Hard stop preserved:** no `lead-followup` code, manifest, registration, execution, or ADR approval; the probe stays neutral; `plan4.2.md` untouched; no commits or pushes; no real provider writes; disposable databases only.
- **Do not self-certify.** G1 closure is claimed only per §11.E's rules; the independent re-review (Prompt 16) audits the contract per §2.5.

## 13. Final instruction

The five HIGH findings are one root cause wearing five costumes: invariants enforced by path shape and comments instead of by predicated, atomic transitions on durable state, with failure branches that were never states of the machine. The evidence findings are the second root cause: tests built to demonstrate, not to falsify. The remediation that matters is the transition-integrity contract — its document (§6), its enforcement in code (§7), and its falsifying tests (§10) — and the finding list is the closure checklist, not the design.

Ratify §3 before coding. Write the contract before changing code. Close every finding red/green, with digests. Report honestly. Leave the tree uncommitted. Preserve the hard stop.

Stop after producing the contract, the remediation, and `docs/prompt15_imp.md`. The re-review that follows will audit the contract, not the confidence.
