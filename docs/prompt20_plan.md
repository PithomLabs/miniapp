Goal

  Execute Prompt 20 (Bounded WS-B Phase 1 Closeout): close HIGH-1, MEDIUM-1, MEDIUM-2, MEDIUM-3 plus
  the ten small items, with red/green evidence for every behavioral correction, then STOP for focused
  independent re-review. Phase 2 stays blocked.

  Success Criteria

  • HIGH-1: both forced winner scenarios (settlement-wins, completion-wins) fail on the
    pre-remediation tree and pass on the corrected tree, via the actual Class A execution path,
    channel-blocked handler, lock-gate staging, and persisted-state plus audit assertions.
  • MEDIUM-1: no settlement write can clobber adopted run identity; terminal/quarantined records
    refuse adoption; losing adoption submits nothing. Proven by deterministic red/green tests.
  • MEDIUM-2: framework-enforced single Class A execution with crash/recovery evidence, or a short
    owner decision memo if the minimum safe fix exceeds Phase 1 scope.
  • MEDIUM-3 and all LOW/INFO dispositions applied; full serial suite, race trio, and checks.sh green;
    closeout report plus evidence register complete.

  Approach

  Phase 0 — freeze and reachability adjudication (before any code). Verify the Phase 1 baselines per
  prompt20 §2: green manifest 5afb6e46… (expect 299/300 entries plus the explained README delta and
  prompt19.md/prompt20.md extras — current tree digests 0c8f25b4…, consistent with exactly those
  additions), red tree c9c08fe5… (already recomputed exactly this turn), m01_test.go full hash
  222f5ff5… (matches). Preserve immutable copies of both baselines outside the repo, record digest,
  register pre-image, HEAD, and environment. Then verify the two reachability proofs below against
  code. If either proof fails (a reachable path was missed), follow the reachable path. If either
  proof holds and the owner has not approved the rescope in Q1/Q2, STOP with a short memo per §10 — do
  not build tests for unreachable races.

  Central discovery (verified this turn, shapes HIGH-1 and MEDIUM-1). The specified API-level races
  look unreachable, for the same structural reason HIGH-1 cited for Cancel-vs-completion:

  • Decide enforces expiry unconditionally (approvals.go:129 (`decide`); past-expiry pending → marks
    expired, returns approvals: request expired). The review's HIGH-1 recipe (expiry observation, then
    mid-race Decide(approve), then a granted poll) cannot yield a granted poll.
  • One live approval per invocation via framework paths (parkApproval adopts, never duplicates;
    parkTx accepts pending/running only), ByInvocation reads latest only (approvals.go:278
     (`byinvocation`)), decided approvals are immutable.
  • Therefore a granted observation and a settle observation (denied/expired/cancelled) are mutually
    exclusive per invocation: lazy expiry, denial, and cancellation all require a non-granted approval
    state, and no transition runs from those states back to granted.

  HIGH-1 recommended rescope (needs Q1): the reachable settle-vs-complete race runs the completion leg
  through trust-float dispatch, not a granted poll. Authorize returns Allow for a gated tool when
  autonomous trust is requested within grant (auth.go:108 (`authorize`)), so leg B (same-key Invoke as
  autonomous → dispatchPending → dispatch → executeSync with a channel-blocked handler) executes the
  real executeSync/completeIfLive path while leg C parks (confirm retry) and leg A settles (expiry
  poll behind the row gate). Forced orders via staged gate queueing: queue A first for
  settlement-wins, queue B first (verified waiter quorum before A starts) for completion-wins; FIFO
  grant with post-hoc winner assertion, 3 rounds each. Red is the acceptance tree plus the new test
  file only (blind update overwrites either way); the new file must be self-contained (own Class A
  gated registration via testkit.RegisterClassATool, own blocking handler) and compile on the old tree
  — verify by compiling the overlay on red. Also fix the threat-model comments in completeIfLive
  (store.go:139-142 (`completeiflive`)) and completeSync (service.go:646-650 (`completesync`)) to cite
  settlement, not Cancel, and confirm store-level terminal immutability coverage (existing
  TestPredicatedOpsFromStates).

  MEDIUM-1 (needs Q2 on test level; code fixes proceed regardless). Fix A: remove the run_id write
  from failIfWaiting, keeping the parameter blanked (_) with a comment (dropping it would make the
  defect untestable red/green — see below). Fix B: add
  AND status IN ('pending','running','waiting_event','waiting_approval') to adoptRunID (pending
  required for restoration dispatch; terminal/quarantined rejected). Fix C: in submitChecked, after a
  lost adoption reloads, re-check record liveness before Async.Submit and refuse without submitting
  when ineligible (closes the winner-terminal-with-run case; error must converge via replay on retry).
  API-level deterministic clobber staging is unforceable (A's synchronous Invoke completes before the
  test can interleave an adoption; a row gate blocks both same-row writers), so test at the
  store/service boundary in a new internal file runtime/p20_medium1_test.go: M1-1 stale-value settle
  after committed adopt preserves R; M1-2 adopt onto terminal refused; M1-3 rejected/losing adoption
  submits nothing (both no-run and with-run winner legs), asserting submitCalls == 0. Red is a pre-fix
  working-tree copy. Explicit non-scope: the adopt-wins-then-settle-wins-then-submit orphan stays a
  documented bounded residual (effects refused via CheckLive; full closure belongs to O-01/Phase 2
  authorization) — recorded, not fixed.

  MEDIUM-2. Step 1: write docs/phase1/classa-execution-claim-design.md from direct inspection
  (claim/txn model, classa.Execute single-tx shape, advisory first-statement position, pollApproval
  granted branch, dispatch/dispatchPending, replay, completeSync, Kill9/SpawnWorker harness in
  testkit/proc.go), answering all 8 prompt20 questions. Step 2: implement the minimum mechanism only
  if it needs no new migration, durable state, or broad API change; otherwise stop with the owner memo
  (proven limitation, alternatives, failure/recovery semantics, contract changes, why-not-smaller).
  Tests per §5 (concurrent granted polls, same-key dispatch vs replay, single execution plus coherent
  terminal plus audit, pre-commit crash, post-commit lost response, no duplicate committed writes)
  with handler-execution counters and persisted-state assertions; red is the pre-fix tree.

  MEDIUM-3. Amend docs/prompt17.md:144 and the prompt17_plan_review.md mirror to cite contract §4 as
  authoritative, remove advisory-last ordering, state authorizeAttempt takes no advisory lock,
  preserve outermost-first for real advisory users; repo-wide grep for contradictions; never-silent
  §13/§16 entry. No code change.

  Small items. LOW-1: replay takes a duplicate flag (true only when claim returned existing; loser
  paths pass false), plus a losing-first-claim test on the HIGH-1 scenario-A staging (runs after
  HIGH-1). LOW-2: X-12 backfill-scope note in the contract plus deferred-work entry — no code. LOW-3:
  three doc precision fixes (§4.2 Cancel row, §9 claim-response timing, §16 provenance cite). INFO-1:
  one-line live definition plus op-specific exclusion table. INFO-2: lightweight AST assertion test
  pinning the single LockEntity call site (guard-only, no red). INFO-3: accept 32-bit contention in
  the report, no change. INFO-4: fix the running only row in docs/prompt19.md:197 (the protocol doc —
  never touch adv_review_prompt19.md). INFO-5: quarantine deferral record. INFO-6: log every digest
  with exact command, CWD, and path forms. INFO-7: contract lines on run-less/non-addressable
  cancellation and the effective-trust-float rule with the Class A/D asymmetry.

  Standing constraints. No changes to register-bound files (m01_test.go, store_test.go, Phase 1
  evidence); new tests in new files only (p20_high1_test.go, p20_medium1_test.go, p20_medium2_test.go,
  p20_low1_test.go). Identical test bytes red/green; per-finding isolated copies; fresh disposable
  DBs; no branches, commits, pushes, Neon, or OpenCode.

  Steps

  1. Freeze: verify baselines, preserve immutable copies, record digest/pre-image/HEAD/env.
  2. Reachability adjudication: re-verify the approval-monotonicity proofs; confirm Q1/Q2 rescope or
     stop with memo.
  3. HIGH-1: comments, two forced-order tests, terminal-immutability confirmation, red/green.
  4. MEDIUM-1: three fixes, three tests, red/green.
  5. MEDIUM-2: design doc, then implement-plus-tests with red/green, or owner memo.
  6. MEDIUM-3 plus LOW/INFO dispositions.
  7. Full verification, evidence register, docs/phase1/prompt20-phase1-closeout.md with the five-way
     distinction, STOP for re-review.

  Validation Plan

  • Per finding: red run fails for the intended defect on the isolated pre-fix tree; green passes on
    the corrected tree; overlay hashes and tree digests recorded with exact commands.
  • Full gates: gofmt -l ., go vet ./..., go test -p 1 ./...,
    go test -race -p 1 ./runtime/ ./approvals/ ./effects/, bash scripts/checks.sh — all on local
    Docker PG19 with TEST_* URLs, GOWORK=off, unsandboxed DB runs.
  • Targeted repeats of race tests; skipped/blocked tests recorded as such, never as PASS; cleanup
    verified (no leaked holders, workers, or scratch DBs).

  Risks / Open Questions

  • Risk: PG lock-queue FIFO order is implementation behavior, not SQL-guaranteed; mitigated by staged
    queueing plus post-hoc winner assertions, the same assumption the accepted X-03 methodology
    already rests on.
  • Risk: MEDIUM-2 may require the owner-memo path; treated as a normal outcome, not a failure.
  • Q1 (HIGH-1, needs owner answer): approve rescoping the race to trust-float-dispatched Class A
    completion vs expiry settlement (same completeIfLive contention, API-level, deterministic), since
    granted-poll-vs-settlement is unreachable as specified?
  • Q2 (MEDIUM-1, needs owner answer): approve store/service-boundary red/green tests (with the
    failIfWaiting parameter retained blanked for testability) plus a documented API-unreachability
    proof, since API-level deterministic clobber staging is unforceable?

  Reply go to start, or tell me what to adjust.
