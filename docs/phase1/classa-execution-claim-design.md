# Class A Execution-Claim Design (MEDIUM-2, Prompt 20 Step 1)

**Status:** investigation complete; minimal implementation proposed (no
memo). No code changed in this step.

**Verdict:** the existing Class A transaction architecture CAN provide
the framework-enforced single-execution guarantee with a narrow,
migration-free change: a transaction-scoped advisory execution claim
taken before the handler runs, plus handler and invocation completion
committed atomically in one transaction. Losers never reach the
handler; crash at any point is recoverable without duplicate committed
work; no new durable state, migration, or broad API change is required.

## 1. Current architecture (verified facts)

All citations are to the post-MEDIUM-1 working tree.

- `executeSync` (`runtime/service.go:668`) runs the handler via
  `Sync.Execute`, then records the outcome via `completeSync`
  (`service.go:681`) → `completeIfLive` (`store.go:145`). Handler
  execution and record completion are TWO SEPARATE transactions.
- `classa.Execute` (`executor/classa/classa.go:30`) opens one
  `WithTenant` transaction, takes the advisory entity lock first
  when an entity key exists (lines 40–44), then runs the handler
  with the open transaction (`ToolCall.Tx`, line 45). Handler
  database writes commit when this transaction commits.
- Three paths reach `executeSync` with no interlock between them:
  first-claim dispatch (`service.go:499`, line 510), pending replay
  via `dispatchPending` (`service.go:223`), and the granted-approval
  poll branch (`service.go:365`, line 386).
- `replay` (`service.go:193`) answers terminal records verbatim;
  live Class A records re-enter dispatch/poll (pending/waiting) or
  fail closed (`interrupted` on running).
- Effect-ledger operations (`effects.Perform`, `effects.go:151`)
  run in their OWN transactions via pool checkouts
  (`store.WithTenant` per step); the `Channel` interface
  (`effects.go:69`) takes no caller transaction. Ledger safety
  comes from claim-or-adopt by business operation ID
  (`effects.go:302`, `effects.go:348`), settled short-circuit,
  `ErrAttemptInProgress` for live races, and the age-predicated
  stale→UNKNOWN quarantine (`effects.go:444`, race-safe by
  construction).
- The contract promises exactly-once provider delivery at the
  ledger layer (`invocation-lifecycle-contract.md:347`); the
  ledger — not the handler transaction — is the cross-transaction
  idempotency mechanism.
- The sole production advisory acquisition is
  `store.LockEntity` (`internal/store/store.go:47`,
  `pg_advisory_xact_lock(hashtext($1))`), taken first-in-
  transaction with no prior row locks (review-verified no-cycle
  proof). No `pg_try_*` variant is used today.
- `SyncExecutor` has exactly one production implementor
  (`classa.Executor`); `executor.Request` has exactly one
  production construction site (`service.go:705`,
  field-named); test-site literals are field-named. An additive
  `Request` field breaks no constructor.
- REST maps errors by kind plus a code switch with a generic
  default (`transport/rest/server.go:94`): a new terminal code
  surfaces without adapter changes (optional one-line 409 mapping
  beside `idempotency-key-reuse`).
- Verified live on PG19: `hashtextextended(text, 0)` and
  `pg_try_advisory_xact_lock(bigint)` both work
  (`SELECT hashtextextended('exec-claim:probe', 0),
  pg_try_advisory_xact_lock(...)` → 64-bit value, `t`).
- Crash-test precedent exists: `SpawnWorker` + file rendezvous +
  `Kill9` (`testkit/proc.go:91,172,203`), used by
  `effects/concurrency_test.go:323`.

## 2. Failure analysis

Three crash/interleave windows exist today; all stem from one root
cause: **handler execution is neither mutually excluded nor atomic
with completion.**

- **W1 — concurrent execution.** Two granted polls (or claim-winner
  dispatch plus pending-replay dispatch) both pass observation and
  both run the handler in separate transactions. Business writes
  commit twice; only the record update is single-winner. Direct-SQL
  writes double-apply; ledger-routed sends survive via claim/adopt
  but the execution-count guarantee is violated.
- **W2 — crash after handler commit, before completion.**
  Transaction 1 (handler) commits; the process dies before
  transaction 2 (completion). The record stays live forever while
  business work is durably committed. Any re-execution (new key
  after `interrupted`, or a future recovery path) double-applies.
  Nothing in the record distinguishes this state from
  "never executed".
- **W3 — crash before handler commit.** Handler transaction rolls
  back; the record stays live; re-execution is safe but nothing
  today performs it for a stuck record (only new-key resubmit).

A predicated status-claim transition alone (review's suggestion)
closes W1 but NOT W2: claim-then-execute keeps handler and
completion in separate transactions, so "claimed but not terminal"
is still ambiguous between "never ran" and "ran and committed".
Closing W2 requires atomicity of handler + completion. Similarly,
single-transaction alone (handler + completion atomic, no claim)
closes W2/W3 but not W1's "losers do not execute" (both handlers
would run; the loser's writes would roll back, but the handler
would still have been reached and counted). Both halves are
load-bearing.

## 3. Proposed mechanism

One transaction per execution attempt, shaped:

```
BEGIN (WithTenant, tenant context first — unchanged)
  1. TRY advisory execution claim:
       SELECT pg_try_advisory_xact_lock(
         hashtextextended('exec-claim:' || tenant_id || ':' || invocation_id, 0))
     - false → ROLLBACK, return terminal execution-in-progress
       conflict (caller retries the same key; never reached handler)
     - true  → proceed (exactly one holder per invocation, all processes)
  2. Advisory entity lock (existing LockEntity, unchanged position
     relative to row locks: claim → entity is the single fixed
     global advisory order)
  3. Run handler(tx) — business writes, uncommitted
  4. Predicated completion UPDATE (same predicate as completeIfLive)
     - 0 rows → ROLLBACK everything (a concurrent settlement won;
       business writes discarded, never committed), reload + replay
     - 1 row  → COMMIT (business + terminal commit atomically;
       xact claim auto-releases)
Post-commit: completion audit exactly as today (unchanged; the
LOW-2 X-12 backfill scope still covers post-win audit loss).
```

Why each piece:

- **Advisory xact claim (ephemeral, no migration).** Held if and
  only if the owner's transaction is alive; a crash (connection
  drop, backend death, `Kill9`) releases it automatically. No
  lease column, no sweeper, no stuck state, no clock dependence.
  Database-global, so separate processes coordinate. `TRY`
  (non-blocking) avoids waiter pile-up behind slow handlers;
  losers get a truthful conflict instead of queueing.
- **64-bit namespaced key.** `hashtextextended` over a
  `exec-claim:`-prefixed string keeps the claim domain disjoint
  from the 32-bit entity-lock domain (no cross-talk with
  `LockEntity`, no INFO-3 interaction) and makes cross-invocation
  false contention negligible (liveness-only if it ever happens).
- **Single-tx handler + completion.** Makes "record terminal"
  synonymous with "business committed". This is the fact recovery
  relies on (see Q6 proof).
- **Predicated in-tx completion.** Preserves HIGH-1 semantics: a
  concurrent settlement still wins by committing first; the
  completion then matches zero rows and the whole attempt —
  including already-run business writes — rolls back. HIGH-1
  scenario A stays green with strictly stronger behavior
  (discard-by-rollback rather than discard-by-convention).
- **Retry-converges losers.** The conflict is terminal and
  same-key-retryable: once the winner commits, retry replays the
  terminal; if the winner crashed, retry re-claims and executes
  (safe by §5). No waiter, no sleep, no new API.

Lock-order (§4) impact: the Class A transaction becomes
advisory-claim → advisory-entity (if any) → row locks (handler
writes) → invocation completion row lock. Advisory acquisition
still happens first with no row locks held (outermost-first
preserved). Two advisories are now held simultaneously (claim +
entity), so the review's "at most one" lemma is replaced by a
fixed-global-order argument: exactly one code site acquires both,
always claim-before-entity, and no path acquires entity-before-
claim — no wait-for cycle is constructible. No lock is held
across any new external operation: connection/tx scope across
the handler is unchanged from today (classa already holds its
transaction open across handler execution); the additions are one
cheap SELECT at the start and one UPDATE at the end.

Effect-ledger composition (verified, not assumed): ledger entries
live in separate transactions and converge by BusinessOpID
claim-or-adopt (`effects.go:302/348`) with settled short-circuit,
so a rolled-back loser attempt cannot duplicate sends — the
winner's re-Perform adopts the loser's PENDING row or receipt.
Stranded PENDING rows quarantine to UNKNOWN exactly once via the
age predicate (`effects.go:444`). Direct-SQL handler writes roll
back with the loser. Genuinely non-transactional handler side
effects (raw socket/HTTP calls outside the ledger) remain the
miniapp's responsibility via ledger routing — the framework
guarantee covers committed database work and handler-reach
exclusion, which is the documented ledger boundary.

## 4. The eight questions

1. **Atomic execution-claim boundary?** The single execution
   transaction: advisory-claim → entity lock → handler →
   predicated completion → commit. Claim, business, and terminal
   are one atomic unit; any abort leaves no partial state.
2. **One winner among granted polls / duplicate dispatches?** The
   `pg_try_advisory_xact_lock` claim admits exactly one holder
   per invocation across all processes. Every path to
   `executeSync` (dispatch, dispatchPending, granted poll) funnels
   through it before the handler runs; losers get the terminal
   conflict without executing.
3. **Loser reload and authoritative result?** Losers return
   terminal `execution-in-progress` immediately (no wait, no
   handler). The caller retries the same idempotency key: replay
   answers the committed terminal verbatim, or re-dispatches into
   a fresh claim attempt if the winner crashed. Convergence is by
   existing replay, no new protocol.
4. **Crash after claiming, before executing?** The claim is the
   transaction's lock: the crash aborts the transaction and frees
   the lock with zero durable trace. The record never left its
   live state. The next attempt claims and executes normally.
   Nothing to fence, nothing stuck, no sweeper involved.
5. **Handler writes commit, response lost?** Impossible to split:
   handler writes and the terminal commit in the same
   transaction. A lost response retries into replay, which
   returns the recorded terminal; the handler is never re-run
   for a terminal record.
6. **Can recovery execute the handler twice (committed)?** No.
   Proof: commit(handler writes) ⟺ commit(completion) — same
   transaction. Recovery executes only when the record is
   non-terminal, i.e. completion never committed, hence handler
   writes never committed. Re-execution after a crash therefore
   always starts from zero committed business work. (Scope: ledger-
   routed sends additionally converge by claim/adopt; raw
   non-transactional side effects stay outside the framework
   guarantee per the ledger boundary.)
7. **Stale owners fenced?** Stale ownership is unrepresentable: the
   lock exists only while the owner's transaction is alive, and
   the completion commits only inside that same transaction. A
   dead owner's writes are already rolled back; a live-but-slow
   owner still holds the claim, so no second owner exists. No
   generation counters or fencing tokens needed.
8. **State discrimination?** Unchanged from today: pending /
   running / waiting_approval are pre-terminal (claim may proceed;
   the in-tx completion predicate re-verifies liveness at commit
   time, so a state change mid-execution rolls the attempt back);
   succeeded / failed / cancelled are terminal and replay
   verbatim without executing; quarantined never executes
   (CR-05; predicate excludes it, unchanged).

## 5. Constraint check (Prompt 20 Step 2 gate)

- **No migration:** the claim is an ephemeral advisory lock; the
  completion reuses the existing status/result/error columns. No
  schema change. VERIFIED trivially (no new table/column/index in
  the design).
- **No durable state:** no lease, generation, owner, or outbox
  rows; crash recovery falls out of transaction abort, not stored
  data. VERIFIED (mechanism above names none).
- **No broad API change:** (a) one additive `executor.Request`
  field carrying the completion closure (sole production literal
  is field-named, `service.go:705`; single implementor); (b) two
  tx-scoped store methods following the existing `parkTx`
  (`store.go:219`) precedent; (c) `classa.Execute` internal
  restructure (claim → entity → handler → completion in its
  already-open transaction); (d) one new terminal code
  (`execution-in-progress`), adapter-transparent per kind-based
  mapping (`rest/server.go:94`), optional one-line 409 entry.
  No interface method added/removed, no adapter change required,
  no caller-visible behavior change except the new conflict code
  (documented, retry-convergent).

All three constraints hold. No owner memo; implementation
proposal below.

## 6. Risks and residuals

- **Long-held advisory claim.** A slow handler holds the claim
  (and its connection/transaction) exactly as long as today's
  handler transaction — no regression. Contention is per-
  invocation (same-key duplicates), not global; try-fail keeps
  losers cheap.
- **Audit stays post-commit** (LOW-2 residual retained): a crash
  between completion commit and audit still needs the X-12
  backfill scope. Moving audit into the attempt transaction
  would close it for this path but widens the change; explicitly
  deferred, not forgotten.
- **Settle-during-execution rolls back business work.** Correct
  (settlement won) but wasteful under chronic contention; same-
  key settle-vs-execute contention is rare by construction.
- **Non-transactional handler effects** stay outside the
  guarantee (ledger boundary, §3). No in-tree Class A handler
  performs ledger effects today (sole non-test `Perform` caller
  is the Class D probe), so no existing handler changes shape.
- **Two-advisory §4 re-proof** (§3) must be recorded in the
  contract amendment log when implemented (never-silent rule).

## 7. Minimal implementation proposal

Production edits (all internal to runtime/executor; frozen
evidence files untouched):

1. `internal/store`: add `TryClaimExecution(tx, tenant,
   invocationID) (bool, error)` issuing the namespaced 64-bit
   try-claim; keep `LockEntity` untouched.
2. `runtime/store.go`: add tx-scoped `completeTx(tx, ...)` with
   the `completeIfLive` predicate (parkTx precedent); keep
   `completeIfLive` for existing callers.
3. `executor`: add additive completion-closure field to
   `Request` (nil = legacy two-tx path, preserving current
   behavior for any non-migrated caller during the transition;
   removed once all Class A paths migrate — or keep the field
   permanent and always set; decide at implementation).
4. `executor/classa`: `Execute` takes the claim first, then the
   entity lock, runs the handler, runs the completion closure,
   and rolls back (returning a typed loss signal) on zero rows;
   claim failure returns typed `execution-in-progress` without
   running the handler.
5. `runtime/service.go`: `executeSync` builds the completion
   closure (status/result/error capture), maps claim failure to
   terminal `execution-in-progress`, maps in-tx loss to the
   existing reload+replay path; audit placement unchanged.
6. Optional one-line REST 409 mapping for the new code.
7. Contract: §4 re-proof note (two ordered advisories) + §13/§16
   never-silent entries; error-code table entry.

Regression tests (new files only; identical bytes red/green;
red = pre-fix working copy + overlay; green = fixed tree):

- T1 concurrent granted polls: channel-blocked handler, one
  winner executes, loser gets the conflict without executing;
  assert calls == 1, coherent terminal, exact audits; loser
  retry converges via replay.
- T2 same-key dispatch vs replay: claim winner dispatches while
  replay loser re-dispatches; assert calls == 1, loser conflict,
  convergence on retry.
- T3–T4 folded into T1/T2 assertions (single execution +
  coherent terminal + audit exactness).
- T5 pre-commit crash: `SpawnWorker` + channel-blocked handler,
  `Kill9` mid-execution, retry in a fresh worker; assert zero
  committed business rows from the killed attempt, exactly one
  after recovery, coherent terminal.
- T6 post-commit lost response: kill after commit before
  delivery (file-rendezvous barrier); retry returns the recorded
  terminal with handler-execution count (persisted counting
  table, not in-memory) still exactly one.
- T7 duplicate-commit accounting: persisted business-row counts
  across T5/T6 prove no double-apply (asserted inside T5/T6).
- Full `./runtime/` + HIGH-1 + M-01 suites re-run (no
  regressions; HIGH-1 scenario A must stay green with rollback
  semantics).

Estimated shape: ~120 lines production, ~400 lines tests, zero
migrations. The design's load-bearing claims (single-tx
atomicity, claim mutual exclusion, ledger composition) are each
pinned by a dedicated test above.
