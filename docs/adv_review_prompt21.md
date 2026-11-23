# Prompt 21 — Independent Adversarial Review of WS-B Phase 1 Closeout

**Review target:** Prompt 20 — WS-B Phase 1 Closeout (`docs/phase1/prompt20-phase1-closeout.md`)
**Review protocol:** independent, adversarial, read-only (this report is the only file created)
**Verdict: CLEAR** — Phase 1 is an independently verified foundation for Phase 2, subject to the
explicit Phase 2 constraints in §J.5. No CRITICAL or HIGH finding remains. One MEDIUM reliability
wart (pre-existing, narrow, non-blocking) and three LOW documentation/test-quality findings remain.
Phase 2 still requires explicit owner release; this review recommends it.

## A. Executive verdict

**CLEAR.** Phase 1 is an independently verified foundation for Phase 2 effect authorization.

What was independently established (not taken on report):

- Full provenance chain recomputed: closeout manifest `a8ffe2f4…`, final tree `e237c51e…`,
  freeze-green `d953a400…`, all five register-bound test files, the register pre-image, and the
  current-tree delta (exactly three post-manifest meta-records). Byte-exact historical red trees
  are unrecoverable (frozen `/tmp` copies gone; README L-04 edit unrecoverable) — honestly
  reported in §B; functional reds rebuilt from HEAD plus documented overlays reproduce every
  reported red failure with verbatim signatures.
- MEDIUM-2 single Class A execution: claim ordering, single-transaction atomicity, savepoint
  semantics, all three new error types, sole-caller inventory, and crash recovery verified in
  source and by execution. All 9 MEDIUM-2 tests pass on green (incl. `-race`); T1/T2/C3/T6 fail
  on the functional red with the reported verbatim signatures; T5 passes both sides as documented
  characterization. A two-session live probe confirms the advisory claim survives savepoint
  rollback and releases at commit.
- Phase 2 compatibility (highest-priority check): all ten §4.E questions answered with a lock
  and transaction interaction graph (§C). No hidden deadlock or atomicity conflict. Composition
  is safe under seven bounded, explicit Phase 2 constraints (§J.5) — most already normative in
  the contract (§4/§4.2/M3).
- MEDIUM-1 Fix C: all six branch cases dispositioned; five are executed-test proven, the sixth
  (winner-terminal-with-run loss) is a composition of two proven halves with a mild failure mode
  and is genuinely unstageable — the closeout's filing is accepted with reasoning in §D.
- HIGH-1: Scenario A is strictly sequential with no queue assumption; Scenario B's FIFO reliance
  is on the safest possible shape (identical mode, single lock object, front-to-back grant loop)
  with loud failure on any deviation, and is load-bearing only for test determinism, never for
  mechanism correctness. Both scenarios fail on functional red with verbatim signatures and pass
  12/12 forced executions on green.
- LOW-1: all three `replay` callers enumerated; the two non-replay loss paths (status-sync,
  cancellation) verified marker-free by construction; both red directions reproduce 4-vs-3.
  Aggregate counts are sufficient for this bounded labeling issue (§F records the exact residual).
- MEDIUM-3: both governing docs corrected identically with REMOVED provenance; no governing
  advisory-last instruction remains; every `authorizeAttempt`+advisory reference states it takes
  no advisory lock; historical materials are distinguishable and triaged.
- Full battery on exact-parity PostgreSQL 19.0: `gofmt`/`vet` clean, full serial suite green
  (incl. `executor/dbos` 37.8s with `TEST_DBOS_DATABASE_URL`), `-race` subset green,
  `scripts/checks.sh` ALL-CHECKS-PASS, targeted `-count=2` repeats green, zero stray backends.

Findings: [P21-002](#p21-002) (MEDIUM, pre-existing pending+workflow stuck state, narrow trigger,
does not block CLEAR), [P21-001](#p21-001), [P21-003](#p21-003), [P21-004](#p21-004) (LOW).
No finding contradicts a foundational guarantee.

## B. Baseline and provenance

Environment: Go 1.27.3 linux/amd64, `GOWORK=off`, PostgreSQL 19.0
(Debian 19.0-1.pgdg13+1) disposable Docker container (image `postgres:19.0` `bb6e70add742`),
fresh `miniapp_dev` / `miniapp_red` / `miniapp_dbos` databases. The project's long-lived
`miniapp-dev-db-1` container was left untouched. DB-touching commands ran unsandboxed (the
sandboxed network namespace cannot reach host ports); all reads, digests, and reconstructions
ran sandboxed. Git: HEAD `4945478dbec418227cd007b08208d20305e62ac1` (`main`), no commits made,
worktree unmodified by this review.

### B.1 Digest verification (full values independently recomputed)

| Artifact | Reported | Independent result |
|---|---|---|
| Closeout manifest digest | `a8ffe2f4…` | `a8ffe2f4b7a0c5b9d007bd6f3cd245427f6cc81283e57d0177490deb416b0730` EXACT |
| Manifest content vs current tree | 321 entries | diff = exactly 3 post-manifest meta-deltas (§B.2) |
| Final tree | `e237c51e…` | `e237c51e971353b1e5297d9d8fee985141d4d4cc7c9f000ec5f3a7eb04fbf154` EXACT (reconstructed: manifest tree + closeout report) |
| Manifest-era README | `0f1f85ff…` | `0f1f85ff63f603daffdbdcb489c69e048d898016091eb7a9784c7781d5b82f79` EXACT (reconstructed: current minus closeout row + blank) |
| Freeze-green | `d953a400…` | `d953a400e577b22ad81f8ac397fa4695fdeea20b6827aa80f1fce257975ad649` EXACT (reconstructed: HEAD + `prompt20.md` + `prompt20_plan.md` + `prompt21.md`) |
| Register pre-image | `acff00c6…` | `acff00c60ce807de3aa0420ebec34cb8e03aa755923fe0efe3808922e02380d1` EXACT, and byte-identical to HEAD's README (pre-image methodology verified) |
| `runtime/p20_high1_test.go` | `f2613224…` | `f261322457dbf1c4050b4b274b53143341518e2e580f3c89241741fe0405fb36` EXACT |
| `runtime/p20_medium1_test.go` | `04632e7c…` | `04632e7ca447f220ca35150f7890b28a88f2f158e768cd08fe791a6b6201fe0c` EXACT |
| `runtime/p20_medium2_test.go` | `0bd1fb60…` | `0bd1fb6079caa69126640ec5144a5fc704832004a7e600cc4351f01fda906fff` EXACT |
| `runtime/p20_low1_test.go` | `be385e54…` | `be385e548c8b8edf793e48d309f76db8255787d75a9151a4ec4b3264c3b308f3` EXACT |
| `internal/store/p20_info2_test.go` | `168e5e40…` | `168e5e40cc7cc9d4546db3c9b12adb00fcf7b58e280a2913124d04bdc7ca2972` EXACT |
| Closeout report / design doc | `d6b04f99…` / `35b218b7…` | both EXACT |
| `runtime/m01_test.go` | `222f5ff5…` both sides | `222f5ff5a77bc5d59417d54c0922b5667e839e1901086f2e8efc11fdba3002bc` EXACT |
| HEAD tree | — (new) | `c60960f0737edf40ce8311650c8de61589a54924df99c6af6f1b547076aa832a` (stable across two archives) |
| Current tree | — (new) | `568e0dff23a35d1032ad51cc698673c75ac08393315acf3a9a9a774e744d87a2` |

Not verifiable (abbreviated, reported-only, never expanded here): `c9c08fe5` (freeze-red),
`fb7a17f0`, `dd3a7db3`, `6dbd8362`, `797897be` (per-finding reds), `fbf8cc20`, `f7378ff6`,
`68dfcfd9` (pre-fix snapshots), `d42b6c25`, `0799839a`, `2008f533`, `c7b64f93`, `656b4290`
(intermediate greens). The frozen `/tmp` copies are gone with the prior session, and the WS-A
README L-04 in-place edit is documented-unrecoverable, so byte-exact historical reds cannot be
rebuilt. This does not prevent CLEAR: §I reproduces every red failure functionally (HEAD +
documented overlays, plus a defect-class patch for HIGH-1) with verbatim signatures, which is
what reds exist to prove — test discrimination, not byte archaeology.

### B.2 Current-tree delta (what this review actually ran against)

Current tree = closeout manifest tree + exactly: (1) `docs/phase1/prompt20-phase1-closeout.md`
added (predicted by the prompt: the report postdates its manifest), (2) `docs/prompt21_adv.md`
added (this review's work order), (3) `docs/evidence/README.md` pure append (+19 lines, zero
modifications — the prompt20 evidence rows). No production, test, migration, or contract byte
differs from the closeout manifest. The prompt21 pool fix (`spikes/rls_concurrent_test.go`) is
inside the closeout tree, not after it. Full worktree inventory: 14 tracked modifications +
15 untracked files per `git status` (all enumerated in working notes; the manifest diff above
is the authoritative statement).

### B.3 Source inventory since the frozen baseline

Freeze-green = HEAD + 3 untracked owner docs. Changes since, by finding: HIGH-1 (2 comment
corrections + 1 test file); MEDIUM-1 (`store.go` Fix A/B + `service.go` Fix C + 1 test file);
MEDIUM-2 (`internal/store.go` claim, `executor.go` hooks/errors, `classa.go` orchestration,
`store.go` recheckTx/completeTx, `service.go` executeSync rewrite, `rest/server.go` 409 line,
2 test files + design doc); LOW-1 (`service.go` isDuplicate + 1 test file); MEDIUM-3 (2 docs);
small items (contract text, `prompt19.md` §5.2 line, INFO-2 guard test); prompt21 (1 spike test
+ evidence row). Test-helper updates for the mandatory hooks (`classa_test.go`,
`crmread_test.go`) verified as the only other test edits. Nothing else changed.

## C. MEDIUM-2 Class A execution review

### C.1 Claim and lock semantics (Priority 1A — all 9 verified)

- **Acquisition order** (`executor/classa/classa.go` `Execute`): tenant context → `nil`-hook
  refusal → `TryClaimExecution` (non-blocking) → `LockEntity` (only when an entity key exists)
  → `Recheck` → savepoint → handler → `Complete` → commit. The claim precedes the handler on
  every path; claim failure returns `ErrExecutionInProgress` before any handler code runs.
- **Database-global, cross-process**: the claim is `pg_try_advisory_xact_lock` on a 64-bit
  namespaced key. No process-local mutex or test-only synchronization is load-bearing. T5/T6
  use real `Kill9` subprocesses; the claim holder is a PG backend, releasable only by tx end.
- **Mutual exclusion**: one holder per (tenant, invocation). Proven by `TestP20ConcurrentClaimSingleWinner`
  (deterministic entry-signal overlap, prompt loser conflict, exactly 1 handler execution) and
  T1/T2 at the service level.
- **Release**: xact-scoped — commit, rollback, savepoint-rollback-safe (proven live, §C.3),
  backend death, connection loss all release. No lease, no sweeper, no stuck state. A dead
  holder's uncommitted work is already rolled back; no fencing needed (unrepresentable stale
  ownership — the design's answer stands as implemented).
- **Domain separation**: claim key `hashtextextended('exec-claim:'||tenant||':'||invocation, 0)`
  (64-bit) vs entity key `hashtext(entity)` (32-bit, sign-extended). Cross-domain collision is
  2^-32 per acquisition and benign when it happens (both xact-scoped; false serialization only,
  no data keyed by lock identity). Within-domain claim collisions are 64-bit-negligible with the
  same benign consequence. No lock-order violation is constructible from a collision.
- **No bypass**: exhaustive caller inventory — `Sync.Execute` has exactly one production caller
  (`runtime/service.go` `executeSync`) and one production implementation (`classa.Executor`,
  wired in `cmd/miniapp/main.go` and `testkit/stack/stack.go`). `faultwrap` decorates but does
  not bypass (test-only wiring; absent from production constructors).
- **Fail-closed hooks**: `nil` claim or `nil` callback refuses before any lock is taken
  (`TestP20NilClaimFailsClosed` pins both shapes, 0 handler executions). The sole production
  caller always sets both hooks (verified at `service.go:678`).
- **Advisory-site inventory**: case-insensitive repo-wide grep finds exactly two acquisition
  statements, both in `internal/store/store.go` (`LockEntity`, `TryClaimExecution`), both
  reachable in production only from `classa.go` `Execute`. The §4 "exactly one production site"
  premise holds. (The contract's `classa.go:41` line citation is stale — actual call sites are
  the claim/entity lines of `Execute`; folded into [P21-004](#p21-004).)

### C.2 Transaction boundary (Priority 1B)

The attempt transaction is exactly `claim → entity lock → liveness recheck → handler →
predicated completion → commit`, verified by reading `classa.go` `Execute` end to end.
Handler business writes and invocation completion cannot commit separately (same `pgx.Tx`;
T6 proves it by `Kill9`). Reordered/off-path variants examined and excluded:

- `Recheck` is a lock-free `SELECT` (no row lock, so it cannot deadlock against the row gate
  or a concurrent settlement); missing rows read as not-live (fail-closed).
- `Complete` (`completeTx`) carries textually the same predicate as `completeIfLive`
  (`pending`/`running`/`waiting_approval` + tenant + id) and reports affected rows; zero rows
  rolls the whole attempt back (`ErrExecutionSuperseded` → reload + replay, never overwrite).
- Settlement-vs-completion serializes on the single invocation row lock; PG grants one writer,
  the loser sees the committed state. Either order is correct; the tests force each (§E).
- Error propagation: hook errors, savepoint errors, commit errors, and connection loss all
  surface as `retryable("store-unavailable")` except the three typed outcomes. The fall-through
  mapping was audited error by error: every reachable error is genuinely retryable (unreachable
  defensive paths — unknown-tool-inside-Execute, nil-hooks-in-production — are misclassified but
  unreachable, since `toolFor` validates upstream and the sole caller always sets hooks).
- `completeIfLive` now has zero production callers (retired by this redesign) and `completeSync`
  is deleted — see [P21-001](#p21-001) for the dead-code/test-target consequences.

### C.3 Savepoint and handler-error behavior (Priority 1B)

Verified in source and live: savepoint created only around the handler; handler error rolls
back to savepoint (business writes discarded), then FAILED completes in the same tx and commits
with the claim still held; `HandlerFailedError` preserves the handler's error (`errors.As` +
`Unwrap`); the claim releases with the commit (post-failure re-execution succeeds in
`TestP20HandlerErrorRollsBackBusiness`, which also asserts 0 business rows under mandatory
tenant context and exactly-once failed completion).

Independent two-session probe on PG19 (this review): holder acquires → savepoint →
rollback-to-savepoint → observer `pg_try` on the same key returns `f` (claim survived) →
holder commits → observer returns `t` (xact release). The closeout's "verified live on PG19"
claim is independently confirmed. (A naive single-session probe would return `t` vacuously —
advisory re-acquisition is session-reentrant — so this review used two sessions.)

### C.4 The `execution-in-progress` result (Priority 1C)

- `ErrExecutionInProgress`: terminal `execution-in-progress` conflict naming the same-key
  retry; loser writes nothing (C3 asserts the row stays `pending` and zero success audits
  while the winner is blocked); loser returns promptly (10s promptness bound; red hits it);
  no implicit retry loop (explicit retry only, converging via replay after commit).
- `ErrExecutionSuperseded`: never persisted; always converted to reload + `replay(fresh, false)`.
- `HandlerFailedError`: terminal `execution-failed` with the handler message; FAILED committed.
- In-progress is never a persisted invocation state (no status value, no row write on the
  losing path — verified by code path and C3's untouched-row assertion).
- REST mapping: `execution-in-progress` added to the 409 case only; retryable stays 503; no
  other code's mapping touched — accurate, no misclassification.
- One doc wart: `ErrExecutionInProgress`'s comment promises "claiming and executing if the
  holder crashed" — true for `pending`/`waiting_approval` records (re-dispatch re-claims) but
  not for `running` records, which answer `interrupted`/new-key. See [P21-004](#p21-004).

### C.5 Crash and recovery evidence (Priority 1D)

| Test | What it proves | Red | Green (this review) |
|---|---|---|---|
| T1 granted polls | one handler entry, loser conflicts, retry replays success, 1 business row, 1 success audit | FAIL `handler entries so far: 2` (verbatim) | PASS 3/3 |
| T2 dispatch-vs-replay | same via pending re-dispatch | FAIL entries==2 (verbatim) | PASS 3/3 |
| C3 retryable conflict | promptness, untouched row, same-key naming, explicit retry converges | FAIL entries==2 (verbatim) | PASS |
| T5 pre-commit Kill9 | killed attempt commits nothing; recovery → exactly 1 row | PASS (characterization, both sides — confirmed) | PASS |
| T6 completion-blocked Kill9 | single-tx rollback; no double-apply | FAIL `business rows after recovery = 2` (verbatim) | PASS |
| C1/C2/C4 + contention units | nil-hooks, terminal recheck, savepoint rollback, executor-level exclusion | n/a (green-only new-API) | PASS incl. `-race` |

T6's rendezvous is sound: the worker's only block point is the completion UPDATE (claim and
entity are advisory, recheck is a lock-free SELECT), established via the transitive
`pg_blocking_pids` quorum with a zero-other-waiters guard. No in-memory counter is used for
crash cardinality — business-row counts and record states are the evidence. Kill9 lands
pre-commit in both T5 and T6; post-commit response loss converges via the T1/T2/C3 retry legs
(replay-verbatim on terminal records). Note the design-doc T6 (post-commit Kill9) was swapped
for the implemented T6 (pre-commit discriminator) without a written trace — justified (the
specified test is not deterministically stageable without a fault hook; the substitute is
strictly stronger), but undocumented: [P21-004](#p21-004).

### C.6 At-most-one guarantee scope and side-effect boundary

The implemented guarantee — at most one concurrent handler execution and at most one committed
transactional business execution per invocation — holds for database work inside the attempt
transaction. Boundaries (all verified as stated): non-transactional handler side effects
(sockets/HTTP outside the ledger) are outside the guarantee; ledger-routed sends converge by
`BusinessOpID` claim-or-adopt in separate transactions (settled short-circuit, fresh-live
`ErrAttemptInProgress`, stale takeover/quarantine — read in `effects/effects.go`); post-commit
audit loss still needs X-12 (deferred, recorded). No in-tree Class A handler calls `Perform`
today, so the ledger-interaction caveats (2-minute lease delay before same-op takeover after
a rollback) are currently latent — recorded as Phase 2 constraint C6.

### C.7 Future Phase 2 compatibility (Priority 1E — highest priority)

Effect authorization today runs in **separate transactions**: every effects step is its own
`store.WithTenant` pool checkout; `Perform`/`Channel` take no caller transaction (verified
signatures). The attempt transaction therefore contains handler direct-SQL only, plus time-
nested (not lock-nested) Perform transactions. Answers to the ten questions:

1. **Inside or separate?** Separate today; the contract's `authorizeAttempt` is specified as
   one tx in §4 order without a nesting statement. → Safe under constraint C3 (below).
2. **Independent lifetimes?** Yes — pool checkout per step, separate backends. Proven safe by
   implementation (no tx handle crosses into effects).
3. **Completion vs authorization invocation locks?** Sequential for one invocation
   (authorization runs inside the handler, `completeTx` after it returns); cross-invocation
   contention serializes on the single row lock with predicate re-evaluation. No inversion. →
   Proven safe, provided C2.
4. **Approval/grant reverse path?** The attempt tx takes no approval/grant/sequence/effect
   locks (`recheck` is lock-free; `completeTx` takes lock 1 only). No reverse edge exists. →
   Proven safe (miniapp direct-SQL on framework tables remains a pre-existing discipline
   residual, not a Phase 2 blocker).
5. **Entity vs claim interaction?** Fixed claim→entity order, single site, try-claim never
   waits. Proven safe by implementation + `TestSameEntitySerializes`.
6. **Claims outermost?** Yes — acquired before any PG lock; only site (exhaustive grep +
   INFO-2 guard for call-based acquisition). → Safe under C4 (raw-SQL bypass noted).
7. **Tx waiting on authorization that waits on the executing invocation?** Excluded: same-
   invocation Perform transactions complete before `completeTx`; cross-invocation waits
   terminate at short ordered transactions that never wait on advisories. → Proven safe
   modulo the miniapp-discipline residual in (4).
8. **Cross-process deadlock?** No constructible cycle: advisory subgraph acyclic (try-only
   claim + fixed order), row subgraph globally ordered 1→5, cross edges run only
   advisory-holder→rows, never rows→advisory. → Proven safe, provided C1+C2.
9. **Rolled-back attempts and ledger state?** Ledger rows persist (separate txs) and recovery
   converges (short-circuit/takeover/quarantine — code-read verified). Recoverable with a
   bounded 2-minute lease-delay caveat for same-op immediate re-Perform. → Safe under C6.
10. **ATTEMPTING preserved without weakening Class A?** Yes — ATTEMPTING transitions commit
    in ledger-owned transactions, orthogonal to attempt-tx atomicity. Neither layer weakens
    the other. → Proven safe by layering.

Lock and transaction interaction graph (text):

```text
Attempt tx (backend A, tx T1) — Class A execution
  [adv claim(inv)] → [adv entity(E)?] → [SELECT inv, lock-free]
    → handler: {direct SQL (row locks ?*)} + {Perform → pool checkout → backend B}
    → [UPDATE inv, lock 1, predicated] → COMMIT
Perform/authorize txs (backend B, txs T2..Tn) + Phase 2 authorizeAttempt (backend C, tx TA)
  [§4.1 RLock?] → [1 inv FOR UPDATE] → [2 appr FOR SHARE] → [3 seq FOR SHARE]
    → [4 switches FOR SHARE + grants] → [5 effect predicated UPDATE] → COMMIT → send
Cancel/settle/sync (backend D): [1 predicated UPDATE] (+2, +5 per §4.2) → COMMIT

Wait-for edges: A→{row locks, only at end, lock 1}; B/C/D→{rows in 1→5 order}.
Nobody→advisory except A-at-start (claim = try-only, never waits).
Cycles: none — advisory subgraph acyclic; row subgraph ordered; cross edges one-way.
* Miniapp direct-SQL residual: handler SQL could lock arbitrary rows (discipline; pre-existing).
```

No conclusion is "genuine blocking incompatibility" or "unknown requiring design" — every item
is either proven safe or safe under a bounded documented constraint (§J.5). The §4.2 table
should gain the Class A attempt-tx row before Phase 2 builds on it ([P21-003](#p21-003)).

## D. MEDIUM-1 Fix C and run-ID ownership review

### D.1 The three fixes (all verified in source)

- **Fix A**: `failIfWaiting` no longer writes `run_id` (parameter retained blanked so red/green
  test sources stay identical). The write is eliminated, not narrowed — a stale carry cannot
  clobber even in principle. `TestP20SettlePreservesAdoptedRun` pins it: adopt R → settle wins
  → `run_id` still R, status failed. Red: `run_id = "", want run-p20m1` (verbatim, reproduced).
- **Fix B**: `adoptRunID` gains `AND status IN ('pending','running','waiting_event',
  'waiting_approval')`. Terminal and quarantined refused; `pending` included for restoration
  (owner-directed). `TestP20AdoptRefusesTerminal` pins failed/quarantined refusal plus
  waiting/pending live controls. Red: `adoptRunID(failed) won` (verbatim, reproduced). The
  allowlist shape makes per-state testing of succeeded/cancelled redundant (rejected by
  construction).
- **Fix C**: `submitChecked` refuses submission for ineligible invocations via `adoptEligible`,
  on both the adoption-loss path and the already-bound path (the latter added during green when
  leg 2 exposed the bypass — same predicate and location, within authorized scope).

Predicate match (Priority 2 Q1): the Go `adoptEligible` switch and the `adoptRunID` SQL
`IN`-list are textually identical four-state sets. Q2/Q3: pending allowed (restoration
control passes), terminal/quarantined rejected (both tested). Q4: lost adoption reloads via
`getByID` (fresh authoritative read) before deciding. Q7: submit counters assert zero in both
rejection legs. Q8: stale settle cannot overwrite identity (Fix A eliminates the write). Q10:
exactly one production `Async.Submit` site (inside `submitChecked`; `faultwrap` is test-only
wiring) — no choke bypass exists.

### D.2 Fix C branch coverage (all six cases)

| # | Branch | Predicate / behavior | Test evidence |
|---|---|---|---|
| 1a | Adoption loses → winner live | adopts winner run, submits (happy path) | `TestConcurrentSubmitSameKey` (X-03, green) |
| 1b | Adoption loses → winner terminal with run | reloads, `adoptEligible` refuses, zero submits | **source-correct, unproven** (see D.3) |
| 2 | Existing run + terminal record | already-bound guard refuses before submit | Leg 2, green PASS; red submits (counter 1→2 verbatim) |
| 3 | Existing run + live record | submits bound run (happy path) | X-01/X-02/X-03 flows, green |
| 4 | Rejected adoption, no run | `winner.RunID == ""` refuses | Leg 1, green PASS; red submits + terminal gains run (verbatim) |
| 5 | Rejected adoption, existing run | same as #2 via bound guard | Leg 2 (same lines) |

(The prompt's six cases collapse to five code branches; 1a/1b split case 1 by winner state.)

### D.3 Disposition of the source-correct-but-unproven branch (1b)

The branch (`!adopted` → winner has run → `!adoptEligible(winner.Status)` → refuse) composes
two independently proven halves: reload mechanics (proven by Leg 1) and the `adoptEligible`
predicate semantics (proven by Leg 2 at the sibling call site). The composition itself (3
lines) has no executed test because it is genuinely unstageable: hitting it requires the
request's `adoptRunID` UPDATE to observe a winner run that its own earlier `getByID` did not
see, with the winner terminal — no fault hook exists between those statements, and any API-
level pause lands the request on the already-bound path instead. Forcing it would require a
new fault-injection point for a 3-line composition — disproportionate and itself a behavior
change. Failure mode if the branch were wrong is mild and bounded: `Async.Submit` is idempotent
on run ID, so the worst case re-attaches the winner's existing run and returns a stale-live
status — no new workflow, no new effects (the O-01 `CheckLive` backstop refuses sends for the
terminal record), record stays terminal. **Accepted as filed**: code inspection + sibling-branch
proofs conclusively establish this non-safety composition. Not a release risk. (Traceability
wart: the closeout's "sibling P3 guard proof" phrase resolves to no labeled artifact — the
substance is Legs 1+2; see [P21-004](#p21-004).)

### D.4 Q9 — status change between liveness check and submission (O-01 residual)

Real and retained as authorized: after a winning adoption (or a live bound-read), a settlement
can land before `Async.Submit`, submitting a workflow for a now-terminal record. Backstop as
described: the effects pre-claim `CheckLive` gate refuses sends for non-live invocations, so
the orphan cannot produce unauthorized effects. One precision note: the residual is named for
the adopt leg ("adopt-wins→settle→submit") but the already-bound leg has the identical
reload→submit window; same phenomenon, same backstop — the name is narrower than the coverage.
Phase 2 owns the durable fix (effect-time authorization); constraint C7 carries it.

## E. HIGH-1 race proof review

### E.1 Scope and mechanism notes

The Q1 rescope (Cancel-vs-completion → settlement-vs-completion) is sound and owner-authorized:
Class A in-flight records carry no `run_id` while public Cancel addresses by `run_id`, so the
original race is unreachable without redesigning cancellation. The trust-float substitution for
the completion leg (granted polls cannot race settlement — approval monotonicity: decided
immutable, expiry enforced at `Decide`, no insert-while-waiting) was verified, including the
load-bearing invariant: production creates approvals only via `parkApproval→InsertTx`
(adopt-or-insert under row lock; `Approver.Submit` has zero production callers), at most one
park wins per invocation lifetime (status never returns to parkable states), and all observers
read the same latest row — so granted-vs-settle observations are mutually exclusive per
invocation. Both legs funnel through the same `executeSync`+completion mechanism, so the proof
transfers. (The owner disposition names `completeSync`; the mechanism is now `completeTx` with
the same predicate — naming drift only.)

The closeout's "two comment corrections only" production change is accurate but easily
misread: HIGH-1's mechanism is the M-01 predicated transitions (prompt17); prompt20 added the
missing proofs plus the comment corrections. In the final tree one corrected comment sits on
`completeIfLive` (now dead code — [P21-001](#p21-001)) and the other retired with `completeSync`;
the live behavior rests on `completeTx`, whose predicate is textually identical. The tests
exercise the live path via public `Invoke`, so the proof binds the shipped mechanism.

### E.2 Scenario A — settlement wins (strictly sequential, no queue assumption)

Verified: A's settle return proves the failed commit landed before the handler is released;
B's completion then loses the predicate, discards the stale result (record result stays
empty), answers `recorded-failure` replay, and emits zero `invoke/succeeded` audits while
exactly one `approval-expired` audit exists. Red (functional: blind completion): `round 0:
completion err = <nil>, want recorded-failure replay` — verbatim match, round 0. Green: 12/12
forced executions pass (3 rounds × 2 test bodies × `-count=2`).

### E.3 Scenario B — completion wins (forced queue + post-hoc proof)

Staging verified: B established as the SOLE waiter (quorum 1, only request in flight — the
blocking statement is the completion UPDATE by elimination: claim/entity are advisory,
recheck is lock-free), then A queues behind (quorum 2, zero-other-waiters guard), then release.
Answers to the seven questions: (1) the blocking statement is `completeTx`'s UPDATE on green
(`completeSync`'s on historical red) — the test comments saying `completeIfLive` are stale on
both trees ([P21-004](#p21-004)); the test itself is mechanism-agnostic (row-lock gate), so the
proof binds either way. (2) The waiter is genuinely established (pg_stat_activity quorum with
transitive blocking-chain proof). (3) Queue order is controlled as far as PG observably
provides (sole→behind sequence). (4) PG's grant loop processes the wait queue front-to-back
with no reordering for identical-mode waiters on one lock object — the test relies on this
practical FIFO, which is the safest possible shape for the assumption; crucially, any other
grant order fails LOUDLY (B would answer `recorded-failure`, A would emit settle audits —
every deviation trips an assertion), so a violation can never pass vacuously. (5) Post-hoc
assertions admit only completion-first (row succeeded with B's exact result, both callers
answered success, zero settle audits). (6) All three rounds independently re-stage the full
gate sequence. (7) Test bytes identical (hash-verified, §B.1).

Red (functional: blind settle): `settle returns approval-expired instead of replayed success`
— verbatim match, round 0. Bonus evidence from this review's battery: with only the completion
predicate blinded, A fails while B passes; B fails only once the settle predicate is also
blinded — each test discriminates its own predicate independently.

FIFO reliance is load-bearing ONLY for test determinism (producing the completion-first
interleave to assert its outcome shape). Mechanism correctness holds under either grant order
(predicated transitions admit exactly one winner; the loser replays). The assumption is
explicitly labeled in the test ("No retries: any other grant order fails loudly") and is
accepted.

### E.4 Store boundary

`TestPredicatedOpsFromStates` re-run green. Caveat: it pins `completeIfLive` (dead code) rather
than the live `completeTx` — same predicate text, but the unit pinning should move; see
[P21-001](#p21-001).

## F. LOW-1 audit attribution review

### F.1 Call-site inventory (all five prompt paths resolved)

Exhaustive grep over non-test `runtime/` finds exactly three `replay` callers: `Invoke` on
`!created` (`isDuplicate=true` — the genuine existing-key path), `settlePollTerminal` on lost
race (`false`), and `executeSync` on `ErrExecutionSuperseded` (`false`). The prompt's
"status-sync-loser" path (`Status`/`syncIfLive` loss) reloads and answers with explicitly no
audit ("the winner owns the settlement record"); the "cancellation-loser" path
(`Cancel`/`cancelIfLive` loss) redelivers or answers `run-not-live` — neither calls `replay`,
so both are marker-free by construction. Only a genuine claim-found-existing emits `duplicate`.
The same-key-different-input conflict emits `key-reuse`, not `duplicate` — correct (a conflict,
not a duplicate).

### F.2 What 4-vs-3 proves and what it cannot

Both tests interleave-check each request's response (completion-loss: `approval-expired` +
`recorded-failure`; settle-loss: success + replayed success with the completed result) and then
assert exactly 3 `invoke/duplicate` markers. Red reproduces `duplicate audits = 4, want exactly
3` in both directions (verbatim). What could pass while misattributing? Only a compensating
defect: a dropped claim marker plus a marked loss-reload in the same run — i.e., a different
bug than the one fixed, introduced by the fix in exact arithmetic compensation, while all
response-shape interleaving checks still pass. The marker is one boolean threaded from three
call sites (directly inspected); all requests share one invocation and actor, so per-request
attribution within the run is nearly unobservable anyway; the truncated-table count admits no
stray-invocation leakage. Aggregate counts are sufficient evidence for this bounded labeling
issue; per-request attribution inside one invocation remains unproven in principle but would
require per-call marker identities to observe — disproportionate, and the prompt forbids
demanding a broad audit refactor here. The `duplicate` marker stays best-effort (auditor
outage can drop it) — consistent with the accepted X-12 post-win gap, not a new hole.

## G. MEDIUM-3 governing-document review

Verified by direct read, not by report: `docs/prompt17.md` and `docs/prompt17_plan_review.md`
carry byte-identical corrected regions — §4 declared authoritative, the advisory-last line
struck (`~~advisory lock~~`, REMOVED) with supersede provenance, the corrected five-lock order
stated, outermost-first preserved for real advisory users (claim→entity at the single Class A
site). Every `authorizeAttempt`+advisory reference in the repo (prompt17 ×2, contract §16 row,
closeout, prompt20 authorization, prompt20 plan, owner disposition) states it takes NO advisory
lock — zero references imply Phase 2 acquires one. `Do not modify the verified advisory-lock
implementation` (prompt20 §6) honored: no production advisory line changed (the INFO-2 guard
run plus this review's exhaustive acquisition grep confirm two statements, one site).

Remaining "advisory-last" mentions triaged and accepted: fix descriptions (prompt20 work
order/plan, closeout, evidence row, this review's own work order), struck lines with provenance
(prompt17 + mirror), and preserved records (`adv_review_prompt19.md`, owner disposition
`review_prompt19.md` — a point-in-time decision document, clearly titled and evidence-listed).
The contract §16 row records the correction under the never-silent rule. No governing document
could reasonably be mistaken: titles, REMOVED markers, and the evidence triage distinguish
current instructions from history. (The evidence row's "§13/§16 row" phrasing overclaims §13 —
§13 has no P20 entries; see [P21-004](#p21-004). The correction itself is complete.)

## H. Remaining dispositions

| Item | Required verification | Result |
|---|---|---|
| LOW-2 | X-12 backfill recorded as deferred, not implemented | PASS — contract X-12 row: "post-win settlement-audit backfill scope (P20 LOW-2 …; deferred, no Phase 1 code)"; no sweeper/reconciler code exists (verified: no `settleTerminal`, no sweeper constructor in prod) |
| LOW-3 | Phase markers, claim-response timing, provenance wording accurate | PASS — §4.2 Cancel marked "(X-05 future state; current implementation is correctly multi-tx)"; `claim-response` timed "post-claim-commit, pre-dispatch/park" (matches the `Fire` site after `claim` returns, before dispatch/park); §16 provenance rows present |
| INFO-1 | Live terminology matches predicates | PASS with omission — `live` = nonterminal minus quarantined; exclusions for cancel/syncIfLive (pending), completeIfLive/completeTx (include pending), failIfWaiting (waiting-only), adoptRunID (all four) all verified against SQL. **Omission**: `checkLive` (effects gate) excludes `pending` but is absent from the exclusion list — folded into [P21-002](#p21-002) |
| INFO-2 | AST guard pins sole site, meaningful ± coverage | PASS — green run finds both primitives in `classa.go`; this review's negative control (rogue `LockEntity` call in a copied tree) fails loudly naming the file. Limitation (accepted): raw-SQL `pg_advisory_*` via `tx.Exec` bypasses the AST matcher; no `checks.sh` gate covers it. Miniapp discipline residual, pre-existing |
| INFO-3 | 32-bit hash contention accepted liveness-only | PASS — no code change; entity-lock contention is serialization-only (verified: no data keyed by lock identity). Residual retained |
| INFO-4 | Prompt19 §5.2 correction matches implementation | PASS — `executeSync` row now lists `pending`/`running`/`waiting_approval` + "reload and replay the recorded terminal", matching `completeTx` + `ErrExecutionSuperseded` handling. Historical review untouched (edit is in the protocol doc `prompt19.md`) |
| INFO-5 | Quarantine deferred to X-10 | PASS — recorded in closeout §9; no quarantine semantics implemented (verified: no resume/fail operator paths beyond the accepted baseline) |
| INFO-6 | Digest commands/CWD/path forms reproducible | PASS — every digest this review recomputed used the recorded `scripts/digest.sh` procedure at repo root; §B reproduces seven digests exactly |
| INFO-7 | Run-less cancellation + trust-float documented | PASS with precision note — run-less uncancellable-by-API ("intended: nothing is running to stop") and the effective-trust rule verified accurate for unparked records; the parked-record exception (waiting records poll, never float) is unstated — see [P21-004](#p21-004) |

## I. Test-discrimination and verification matrix

All "independent execution" entries ran on PG19.0 (Debian 19.0-1.pgdg13+1), Go 1.27.3,
`GOWORK=off`, disposable `miniapp_dev`/`miniapp_red`/`miniapp_dbos` databases. Reds are
functional reconstructions (HEAD archive + documented overlays; HIGH-1 adds a defect-class
unpredicate patch) since byte-exact historical reds are unrecoverable (§B.1).

| Claim | Red result | Green result | Independent execution | What is proven | Remaining uncertainty |
|---|---|---|---|---|---|
| HIGH-1 A settle-wins | FAIL round 0, false success (reported) | PASS 3/3 (reported) | FAIL `completion err = <nil>` verbatim; PASS 12/12 forced | blind completion discriminated; predicated loss + replay correct | none (byte-exact red unrecoverable; discrimination proven) |
| HIGH-1 B completion-wins | FAIL round 0, false failure (reported) | PASS 3/3 (reported) | FAIL `approval-expired over completion` verbatim; PASS 12/12 | blind settle discriminated; FIFO staging + post-hoc proof sound | practical-FIFO assumption (loud on violation; safest shape) |
| MEDIUM-1 stale settle | FAIL clobber (reported) | PASS (reported) | FAIL `run_id = ""` verbatim; PASS | settle can never write identity | none |
| MEDIUM-1 adopt terminal | FAIL adopt won (reported) | PASS (reported) | FAIL `adoptRunID(failed) won` verbatim; PASS | predicate rejects terminal/quarantined | succeeded/cancelled by construction (allowlist) |
| MEDIUM-1 submit on loss | FAIL both legs submit (reported) | PASS (reported) | FAIL counter 1→2 verbatim; PASS, zero submits | Fix C both legs refuse | branch 1b composition unproven (accepted, §D.3) |
| MEDIUM-2 T1/T2/C3 | FAIL entries==2 (reported) | PASS (reported) | FAIL `handler entries so far: 2` verbatim ×3; PASS | single handler execution + prompt conflict + retry convergence | none |
| MEDIUM-2 T6 | FAIL rows==2 (reported) | PASS (reported) | FAIL `business rows = 2` verbatim; PASS | single-tx atomicity (no double-apply) | none |
| MEDIUM-2 T5 | PASS both sides (characterization) | PASS | PASS on red (confirmed characterization) | pre-commit crash rolls back; recovery converges | not a discriminator (filed correctly) |
| MEDIUM-2 C1/C2/C4+units | n/a green-only | PASS incl. race (reported) | PASS incl. `-race` | hooks/recheck/savepoint/exclusion | none |
| LOW-1 both directions | FAIL 4-vs-3 (reported) | PASS (reported) | FAIL `duplicate audits = 4` verbatim ×2; PASS | marker only on true claim-found | per-request attribution in-principle (accepted, §F.2) |
| MEDIUM-3 grep | n/a (doc) | zero governing hits (reported) | zero governing hits (reproduced); triage verified | governing instructions unambiguous | none |
| INFO-2 guard | n/a (guard-only) | PASS (reported) | PASS + negative control FAILS naming rogue file | pins sole site; catches additions/renames | raw-SQL bypass (accepted residual) |
| Full suite + gates | — | all green (reported) | `gofmt`/`vet` clean; all pkgs ok incl. dbos 37.8s; `-race` subset ok; checks ALL-CHECKS-PASS; `-count=2` repeats ok | no regressions; battery reproduced | none (zero skips: `TEST_DBOS_DATABASE_URL` set) |
| Savepoint/advisory | — | "verified live" (reported) | two-session probe: `f` held post-rollback, `t` after commit | claim survives savepoint rollback; xact release | none |

Classification: verified red/green — HIGH-1 A/B, MEDIUM-1 ×3, T1/T2/C3/T6, LOW-1 ×2;
source-correct but unproven — Fix C branch 1b (accepted, §D.3); characterization — T5;
guard-only — INFO-2; accepted residuals — O-01 orphan (+bound-leg variant note), 32-bit hash,
ledger boundary, X-12 gap, T5, FIFO staging, aggregate counts, lease delay, raw-SQL bypass,
miniapp discipline; deferred — X-12 implementation, X-10 quarantine, prompt21 CI scratch.

## Findings

### P21-001

| Field | Detail |
|---|---|
| Finding ID | P21-001 |
| Severity | LOW |
| Contract reference | Contract §7 rule "No row claims enforcement without a test" (inverse: no test should pin dead enforcement); prompt20 §8 (evidence binds claims to exact source state) |
| Source location | `runtime/store.go` `completeIfLive` (def); `runtime/store_test.go` `TestPredicatedOpsFromStates` et al.; `runtime/p20_high1_test.go` comments (lines 10, 353–355, 373–375) |
| Defect class | Evidence (test-target staleness) |
| Root cause | MEDIUM-2 retired all production callers of `completeIfLive` (verified: zero non-test callers) and deleted `completeSync`, but the store boundary tests still exercise the dead function and HIGH-1 comments still name it as the live mechanism |
| Trigger | Reading test coverage as unit-proof of the shipped completion path |
| Failure mode | `TestPredicatedOpsFromStates` pins a predicate that no production path executes; a future `completeTx` predicate regression would pass unit tests and be caught only at integration level (HIGH-1/MEDIUM-2 races) |
| Impact | Bounded: the live predicate is proven behaviorally (same text + 12 forced HIGH-1 executions + 9 MEDIUM-2 tests green); the gap is unit-level pinning + misleading comments |
| Evidence | Caller grep (zero prod callers); `completeSync` zero repo-wide references; HIGH-1 green battery; predicate textual identity |
| Discriminating test | None exists (that is the finding); a `completeTx`-through-`pgx.Tx` unit test or deletion of `completeIfLive` would close it |
| Required correction | Smallest durable: either delete `completeIfLive` (and its tests) with a one-line note that `completeTx` is the sole completion, or retarget the boundary tests to drive `completeTx` inside a `WithTenant` tx; fix the stale `completeIfLive` test comments to name the row-gate mechanism |

### P21-002

| Field | Detail |
|---|---|
| Finding ID | P21-002 |
| Severity | MEDIUM (pre-existing WS-A baseline; narrowed but not introduced by prompt20, which deliberately kept `pending` adoptable) |
| Contract reference | CR-05 `live` definition + INFO-1 exclusion list (omits `checkLive`); "pending records carry no runs" rationale (false post-restoration) |
| Source location | `runtime/service.go` `submitChecked` (no status change on submit) + `runtime/lifecycle.go` `checkLive` (excludes `pending`) + `cancelIfLive`/`syncIfLive` (exclude `pending`) |
| Defect class | Reliability (stuck state), bounded impact |
| Root cause | Nothing transitions `pending`→`running` on submit (no production `SET status='running'` exists; workflows don't write back; no sweeper), while `checkLive`, `cancelIfLive`, and `syncIfLive` all exclude `pending` — so a restoration-submitted `pending`+workflow record is effect-paralyzed (Perform refused), un-cancellable by API (`run-not-live`), and never synced |
| Trigger | Exact sequence: Class D record `pending` (claim-then-park gap: park failure, crash X-01, or millisecond retry race) + trust floats to Allow + same-key retry → `dispatchPending` → submit → workflow runs against a `pending` record indefinitely |
| Failure mode | Workflow runs but every `Perform` fails `ErrInvocationCancelled`; `Cancel` answers misleading `run-not-live … pending`; `Status` never syncs; record stuck until trust floats back (re-park converges) or operator SQL |
| Impact | Bounded: narrow trigger (requires a grant change racing the pending window), loud symptoms, zero safety violation (over-refusal, never unauthorized execution), one invocation at a time, pre-existing since WS-A |
| Evidence | Source reads (submit path, three predicates, zero running-transitions, no sweeper); contract text; no test covers pending-record Perform (verified by absence) |
| Discriminating test | None exists; a restoration-submit-then-Perform-then-Cancel test would demonstrate it |
| Required correction | Smallest durable: define pending+workflow semantics explicitly — either transition to `running` at submit time (one predicated UPDATE in `submitChecked`'s adopt path) or extend `checkLive`/`cancelIfLive`/`syncIfLive` to `pending`-with-run; correct the "carry no runs" rationale; add `checkLive` to INFO-1's exclusion list with its justification. Phase 2 constraint C5 carries the pending-semantics decision into `authorizeAttempt` regardless |

### P21-003

| Field | Detail |
|---|---|
| Finding ID | P21-003 |
| Severity | LOW |
| Contract reference | §7 rule "No lifecycle correction ships without a contract row" (MEDIUM-2 shipped none in M1); §4.2 lock-order table (missing attempt-tx row) |
| Source location | `docs/phase1/invocation-lifecycle-contract.md` §7/M1 + §4.2 (absent rows) |
| Defect class | Evidence (normative-record incompleteness) |
| Root cause | The MEDIUM-2 contract amendment updated §4 prose + §16 log but added no §7/M1 transition rows (attempt-tx completion, `execution-in-progress` conflict, `interrupted` replay, Class A grant→executeSync) and no §4.2 attempt-tx lock row; pre-existing Class A replay rows were already missing |
| Trigger | A Phase 2 designer (or auditor) looking up the normative Class A execution/replay machine |
| Failure mode | Sound-but-unspecified behavior: the `interrupted`/new-key semantics and conflict semantics have no normative row; the §6/M1 same-key-retry/running row prescribes choke re-entry without a Class A exception, which the code (correctly) does not perform for Class A |
| Impact | Bounded: behavior verified sound by this review; risk is future misbuilding on assumed semantics. Does not block Phase 2 (which needs the attempt-tx lock footprint — specified — not replay semantics) |
| Evidence | Full M1 table + §4.2 table reads (no Class A rows); `replay` source (Class A `interrupted` branch, pre-existing at HEAD, unspecified in contract) |
| Discriminating test | n/a (documentation); verification is the suggested rows matching shipped behavior |
| Required correction | Add M1 rows: (1) `Class A attempt | pending/running/waiting_approval | terminal | advisory claim + entity, then predicated in-tx completion; zero rows → rollback + replay` (tests T1/T2/C3/T6); (2) `execution-in-progress | attempt | terminal-409 conflict | claim held; names same-key retry; loser writes nothing` (test C3); (3) `Same-key retry | running + Class A | interrupted (terminal) | fail-closed resubmit-new-key` (existing `interrupted` tests); (4) Class A grant→`executeSync`→terminal. Add §4.2 row: `Class A attempt | advisory claim → advisory entity → 1 invocation (predicated completion) | advisories outermost; handler direct-SQL nests inside` |

### P21-004

| Field | Detail |
|---|---|
| Finding ID | P21-004 |
| Severity | LOW |
| Contract reference | Corrections-never-silent (§13 rule, §16 log); INFO-6 exactness |
| Source location | Closeout report + evidence rows + design doc + test comments + `executor/executor.go` + contract §4 + INFO-7 rule |
| Defect class | Evidence (traceability/provenance warts; bundled — each trivial alone) |
| Root cause | Seven small overclaims/stale references, each verified individually: (1) "§13/§16 entries" — §13 has no P20 entries (only §16); (2) "sibling P3 guard proof" resolves to no labeled artifact (substance = Fix C Legs 1+2); (3) design-T6 (post-commit Kill9) swapped for implemented-T6 (pre-commit discriminator) without a written trace; (4) design §1 claims the contract "promises exactly-once provider delivery" — the contract explicitly disavows it ("No claim of exactly-once provider delivery"); the design's conclusion doesn't depend on the premise; (5) HIGH-1 test comments name `completeIfLive` as the blocking mechanism (stale on both trees; tests are mechanism-agnostic so proofs bind); (6) contract §4 cites `classa.go:41` (stale line; acquisitions verified at the `Execute` claim/entity lines); (7) `ErrExecutionInProgress` comment promises post-crash claim-and-execute unconditionally (false for `running` records → `interrupted`); (8) INFO-7 trust-float rule omits the parked-record exception (waiting records poll, never float) |
| Trigger | Auditing any cited reference against its target |
| Failure mode | Misdirected readers; overstated cross-references; none affects behavior or proof validity (each verified as wording-only with substance intact) |
| Impact | Minimal: all substance verified; corrections are one-line each |
| Evidence | Direct reads cited per sub-item during this review |
| Discriminating test | n/a (documentation) |
| Required correction | One-line fixes per sub-item: correct the §13 citation; label the sibling proof as Fix C Legs 1+2; add a design→implementation T6 swap note; soften the exactly-once premise to the contract's at-most-once-per-owner + adoption language; reword test comments to "completion UPDATE (row gate)"; refresh the §4 line citation; qualify the InProgress comment by record state; qualify INFO-7 with "unparked records" |

## J. Residual risks and Phase 2 compatibility

### J.1 Confirmed defects

P21-002 (MEDIUM, pre-existing, narrow, tracked above). No CRITICAL or HIGH. P21-001/003/004
are LOW documentation/test-quality items.

### J.2 Unverified claims

None outstanding. Every closeout "source-correct but not independently proven" item is
dispositioned: Fix C branch 1b accepted with composition reasoning (§D.3); T5 filed correctly
as characterization (confirmed passing on red). The closeout's "verified live" advisory claims
were re-proven live. Historical red byte-identities are unverifiable (honest limitation, §B.1)
but their probative content (discrimination) is reproduced.

### J.3 Accepted residuals (all re-verified as described)

O-01 adopt→settle→submit orphan (+bound-leg variant note, §D.4) with `CheckLive` backstop;
INFO-3 32-bit entity-hash contention (liveness-only); non-transactional handler side effects
outside the guarantee (ledger boundary); post-win audit gap pending X-12; practical-FIFO
staging assumption (loud on violation); aggregate-count attribution (§F.2); 2-minute lease
delay for same-op ledger takeover after rollback (latent for Class A — no Perform callers);
raw-SQL advisory bypass of the INFO-2 guard; miniapp direct-SQL discipline (framework tables
off-limits by convention). Each was challenged against its safeguard; all hold.

### J.4 Deferred items

X-12 audit-backfill sweeper (scope recorded, no code — verified absent); X-10 quarantine
semantics (recorded, no code — verified absent); prompt21 CI scratch (separate workstream,
untouched by closeout logic). Correctly filed; no deferral misrepresents implemented work.

### J.5 Explicit Phase 2 design constraints

1. **C1 — `authorizeAttempt` takes no advisory lock.** Already normative (MEDIUM-3, contract
   §4.2/M3). Load-bearing for the no-cycle proof; re-verify at Phase 2 implementation.
2. **C2 — `authorizeAttempt` follows global order §4.1 RLock → 1 → 2 → 3 → 4 → 5.**
   Already normative. Any deviation re-opens the deadlock analysis.
3. **C3 — authorization runs in its own transaction (pool checkout), never on the attempt's
   `pgx.Tx` handle.** Current layering; moving authorization inside the attempt tx would need
   a fresh atomicity analysis (lock order would still hold: advisories outermost).
4. **C4 — no new advisory-lock acquirers.** The INFO-2 guard enforces this for call-based
   acquisition; extend it (or add a `checks.sh` raw-SQL gate) if Phase 2 adds SQL paths.
5. **C5 — define `pending` semantics explicitly in `authorizeAttempt`.** The pending+workflow
   question ([P21-002](#p21-002)) must be answered before authorization revalidation can treat
   pending records; recommend fixing P21-002 first (one predicated UPDATE).
6. **C6 — Class A handlers that `Perform` must handle `ErrAttemptInProgress`** (bounded retry
   across the 2-minute lease) after attempt rollback; op-ID derivation must make same-invocation
   recovery adopt rather than strand.
7. **C7 — the O-01 orphan window closes at the effect boundary.** Phase 2's authorization
   revalidation is the durable fix for submit-after-settle; the `CheckLive` backstop is
   interim coverage, not the design.

### J.6 Genuine incompatibilities

None found. The Class A claim transaction and the specified Phase 2 authorization transaction
compose without hidden lock-order or atomicity conflicts (§C.7).

## K. Final recommendation

Phase 1 is independently cleared to serve as the foundation for Phase 2. The lifecycle
integrity, single Class A execution, workflow ownership, and transaction semantics are proven
by a combination of predicated SQL (read and hash-verified), deterministic forced-interleave
tests (reproduced red and green with verbatim signatures on exact-parity PG19), crash-recovery
tests with real `Kill9` subprocesses, a live two-session savepoint/advisory probe, and an
exhaustive lock-order analysis with no constructible cycle.

Minimum required closeout work: none is required before owner release. The four findings
should be tracked as ordinary follow-ups, in this order: P21-002 alongside Phase 2 entry
(C5 — the pending decision is needed anyway), P21-003 before any work that builds on Class A
replay semantics, P21-001/P21-004 opportunistically (comment/row hygiene). If the owner prefers
zero open MEDIUMs at release, P21-002's smallest correction (one predicated UPDATE + tests,
estimated well under a day) is the only candidate — but its narrow trigger, loud symptoms,
pre-existing status, and absence of any safety violation do not justify holding Phase 2.

**Recommendation: the owner may release Phase 2.** The framework owns reliability; Phase 1 now
proves it for admission, lifecycle, and single execution. Do not weaken the C1–C7 constraints
during Phase 2 design — they are the load-bearing interface between what Phase 1 proved and
what Phase 2 will build.

