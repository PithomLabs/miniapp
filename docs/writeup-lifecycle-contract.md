# The Invocation Lifecycle Contract — A Developer's Guide

**Audience:** Developers building Class D miniapps on the framework, and operations teams running them.
**Companion document:** *"Writing Class D Miniapps: Complexity, Failure Modes, and Framework Mitigations"* — the writeup explains *why* Class D is hard and what the framework gives you. This guide explains *the contract that specifies those mitigations*: what the states are, what every transition must obey, and what is guaranteed — and not guaranteed.
**Source:** `Invocation Lifecycle Contract (Prompt 15.2 P1 — Normative)`, verified against tree `5cf9627` and pinned SDK `dbos-transact-golang` v1.9.0. Section references below (§1–§16) point into that document.

**Status note:** The contract is a normative draft whose amendments gate P2 implementation. This guide describes the accepted target contract, not a claim that every capability is already shipped. Treat a capability as available in a running framework only after its implementation, tests, and P2 exit evidence are accepted.

---

## 1. What this contract is, in one page

The contract is the rulebook for every durable lifecycle mutation in the framework: claiming an invocation, parking it for approval, submitting a workflow, sending an external effect, cancelling, settling. It exists because the framework's own adversarial review found that every HIGH defect lived in exactly one place — a contested mutation with no predicate, no lock, or no defined failure branch.

Its core discipline is a single sentence:

> **No lifecycle correction ships without a contract row. No row claims enforcement without a test.**

Every transition in this guide corresponds to a row in the contract's normative transition tables (§7). Every row names its predicate (the exact `WHERE` and lock scope), its atomicity (one transaction or explicitly separate ones), its authorization, its audit event, its failure exit, and its test. If a developer adds an ad-hoc lifecycle mutation that has no row, review rejects it. That is the whole game: **the state machine is the code, and every edge of it is written down and tested.**

Three reading aids before the detail:

1. **"Predicated" means the write proves its expected prior state.** `UPDATE ... WHERE status='pending'` plus an affected-row check. Zero rows affected = you lost a race; you reload and reconcile or return a defined conflict. You never overwrite.

2. **"Point of no return" means a durable boundary, not a hope.** For effects it is the `ATTEMPTING` state: once the authorization transaction commits, the send happens, and cancellation cannot recall it. The contract is explicit that the enforceable invariant is *no new `ATTEMPTING` transition after durable cancel* — never *no send can still be in flight after cancel*.

3. **Every guarantee has stated conditions.** Bounded recovery requires a healthy sweeper and available dependencies. Deadlock-freedom requires every transaction to respect the lock order. Nothing is guaranteed by intention; everything is guaranteed by construction plus a named test.

---

## 2. The bugs that shaped the rules

The writeup's five HIGH findings are the contract's root-cause input (§1, RC-1a–d). Each one is a generalizable shape; the rules exist to make each shape impossible.

| Finding | Shape | What went wrong in the code |
|---|---|---|
| X-01 | Authorization by path ordering | The gate and the submit were different functions; a retry after a partial failure entered submit without re-passing the gate, so a gated workflow ran unapproved. |
| X-02 | Failure branches not modeled as states | A submit-after-park failure left the invocation `waiting_approval` naming a run that was never submitted — polls waited forever, grants failed forever, denials looped forever. The failure was a gap between states. |
| X-03 | Non-predicated writes | Two concurrent same-key submits both saw `run_id = ''`, both minted IDs, last write won; one workflow stayed live and unaddressable. |
| X-04 | Read-modify-write without locks | Two concurrent `Decide` calls both read `pending`, both wrote; approve and reject could both "succeed." |
| X-05 | Check-then-act across boundaries | Liveness was checked in one transaction (lock released at commit); the effect claim INSERT happened in a later transaction with no liveness predicate. A cancel committing in the gap stopped nothing. |

Two meta-findings matter as much as the five: **X-07** (a mock that deduplicated its own log made "at least one send" the strongest assertion possible — evidence built to demonstrate, not falsify) and **RC-2** generally (tests never covered interleavings or failure branches). Hence Rule 6 below: evidence must discriminate.

---

## 3. The six transition-integrity rules (§2)

Every contested lifecycle mutation must obey all six. These are the rules a developer can assume the framework enforces — and must mirror in their own durable code.

1. **Predicated single-writer transitions.** Conditional `UPDATE` (or a proven row-lock protocol) + affected-row check; on zero rows, reload and reconcile or return a defined conflict. `run_id` is adopted, never overwritten.
2. **Guard-and-act consistency.** The guard stays authoritative through the durable operation it protects: guard and act share one transaction or a proven lock protocol. Cancellation and effect claims serialize at the claim boundary.
3. **Authorization at the execution boundary.** All executor submissions pass one choke point (§5 below) that resolves the approval requirement from persisted state — never from caller flags, cached values, or a bare signal.
4. **No unreachable operational states.** Every nonterminal state has a defined convergence path. Every retry re-enters the machine and dispatches on persisted state; a retry flag never decides whether authorization applies.
5. **Truthful lifecycle and audit reporting.** Never report success for a nonexistent target. Audit decisions at commit, not after signaling. Settle independently of polling. Refusals are attributable events, distinct from provider attempts.
6. **Falsifiable evidence.** Every invariant claim ships with a discriminating test; mocks measure every operation; affinity tests distinguish executors; results bind to source digests.

### The two permissions (Rule 3, owner-approved)

For gated tools, the contract distinguishes two permissions. **A pending approval is never a business-execution authorization.**

| Permission | Required condition | Permits |
|---|---|---|
| **Workflow-start** | A valid persisted **pending** approval linkage | Submitting the workflow *solely to enter its durable approval-wait path* |
| **Business-execution** | A persisted, verified **granted** decision (matching authority, tenant, tool, intent hash, unexpired) | Authorizing consequential effects — revalidated independently inside the effect-authorization transaction |

The linkage binds tenant, invocation, tool, and intent. A signal, a cached workflow value, or a caller-supplied approval ID alone is insufficient. Missing, denied, expired, cancelled, or mismatched linkages **fail closed**.

---

## 4. The six state machines and who owns them (§3)

Six durable machines plus four operational disciplines (control-plane authorization, CI enforcement, analyzer enforcement, evidence provenance). Every finding X-01–X-24 and every residual F-01–F-14 maps to exactly one primary owner — nothing is ownerless.

| Machine | Subsystem | Owner |
|---|---|---|
| M1 — Invocation lifecycle | invocation row status | `runtime` service/store |
| M2 — Approval | approval rows + the DBOS signal bridge | `approvals` + `runtime` |
| M3 — Effect / attempt | the effect ledger | `effects` |
| M4 — Workflow / run identity | `run_id` / `attached_run_id`, signals, foreign versions | `executor/dbos` + store |
| M5 — Sequence / attachment | probe generations, attach linkage | `probe/classd` |
| M6 — Cancellation | cancel intent, approval settlement, executor stop, in-flight report | `runtime` + `effects.LiveEffects` |

### Per-machine reference (states, terminals, invariants — §7 CR-05)

| Machine | Nonterminal | Terminal | Key invariants / prohibitions |
|---|---|---|---|
| **M1 invocation** | `pending, running, waiting_approval, waiting_event, quarantined` (quarantined = dead-lettered, operator-actionable) | `succeeded, failed, cancelled` | Settled never re-lives; run ID never overwritten; a nonempty `run_id` maps to exactly one owner; attached resolves to exactly one target; quarantined moves only via operator resume/fail. Cancel on `cancelled` redelivers the stop, never re-lives. |
| **M2 approval** | `pending` | `decided, expired, cancelled` | First terminal decision is never substituted (identical re-decide is idempotent); expiry requires a supplied `expires_at`, never invented. |
| **M3 effect** | `PENDING, ATTEMPTING` | `CONFIRMED, FAILED, UNKNOWN` (settled only via Settle), `NOT_APPLIED` (retryable only via fresh Authorize), `CANCELLED` (withdrawn before authorization) | At most one owner authorizes a send per business operation; superseded generations authorize nothing; `UNKNOWN` never blindly retried; `CANCELLED` means the provider was never called; no transitions out of `CONFIRMED/FAILED/CANCELLED`; `UNKNOWN→ATTEMPTING` prohibited. |
| **M4 workflow/run** | `PENDING, ENQUEUED, DELAYED, SUSPENDED` + dead-letter (nonterminal-quarantined) | `SUCCESS, ERROR, CANCELLED` | Every signal outcome is one of the 5-way taxonomy; shutdown failure never silently closes a live pool; `accepted` never reported for terminal runs. |
| **M5 sequence** | `active` | `superseded, done` (per generation: `done`) | At most one `active` generation per (tenant, subject) — partial unique; an attached loser never owns the sequence; new generation = new row. |
| **M6 cancellation** | intent recorded, propagation converging | intent + approvals settled + `PENDING` withdrawn + executor stop absorbed-or-converged | Durable intent precedes every stop; `in_flight` and `withdrawn` are never conflated; no new authorizations after intent commit; attached rows never cancel a shared workflow (DETACH-ONLY). |

---

## 5. The developer-visible lifecycle

### 5.1 An invocation's life (M1)

**Ungated tool:** `Invoke` → `running` (→ `waiting_event` if it waits on signals) → `succeeded | failed | cancelled`.

**Gated tool** (gating resolved from the admitted manifest, never caller flags): `Invoke` → `pending` → park → `waiting_approval` → on grant, submit + converge → `running` → terminal; on deny/expire/cancel → `failed`. The park is **one transaction** that locks the invocation row, writes the predicated park update, inserts the approval, and emits the `approval-requested` audit — all commit together or all roll back. A retry after any partial failure re-parks under the same idempotency identity; adoption of a dangling row exists only as defensive recovery for legacy records (built from a fixture — it cannot be produced by the atomic park).

**Dead-lettered run:** a workflow that exhausts recovery attempts is **not failed** — it maps to invocation `quarantined` (nonterminal, operator-actionable). The operator resumes (`dbos.ResumeWorkflow` first, then the predicated invocation update `WHERE status='quarantined'`) or settles it as failed (predicated, audited with attribution). Resume after cancel is refused; concurrent resume-vs-fail serialize on the predicate — one wins, the loser gets a defined conflict. The status surface shows `status: quarantined` with the reason and the required action; it never auto-settles and is never silently failed while resume is available.

**Settlement (Completion):** `settleTerminal` is a predicated update `WHERE status IN (live)`, shared by the `Status` read path and the sweeper — so settlement is audited (`run-settled/<terminal>`) exactly once, whether or not anyone polls. Attached invocations resolve the **owner** row's outcome; success/failure/cancel propagate truthfully (never `Succeeded` while the target is active).

### 5.2 An approval's life (M2)

The approval table is the durable source of truth; the DBOS signal is a hint. `Decide` takes `SELECT ... FOR UPDATE`, re-checks scope, intent hash, non-self, and tenant **under the lock**, then writes the predicated `UPDATE ... WHERE status='pending'`. Audit is written at commit, before any signal. Losers of a concurrent decide reload: identical outcome → idempotent success; conflicting outcome → a defined conflict error, never a silent overwrite. Expiry marking is predicated (`status='pending' AND expires_at passed`) and shared by the poll path and the sweeper. Signal-after-decide uses the verified-signal operation: a signal failure is retryable and the sweeper redelivers — the decision is never lost with the signal.

### 5.3 An effect's life (M3) — the heart of the contract

This machine encodes the honest answer to "did the provider actually do it?"

```
            cancel (durable intent)
   ┌──────────────────────────────────────┐
   │                                      ▼
(claim) → PENDING ──authorize-tx──→ ATTEMPTING ──send──→ CONFIRMED | FAILED | UNKNOWN
             │  (provisional,        (point of no return;      │
             │   never authorized       send strictly after    └─ UNKNOWN: operator settle only
             │   to call the provider)  commit)                   (atomic, evidence-backed)
             ▼
          CANCELLED (withdrawn before authorization; never attempted)

  Refusal before any claim → no row at all, only an `effect-refused/<reason>` audit.
  Provider reports "not applied" → NOT_APPLIED → retry only via a fresh Authorize.
```

The rules that make this machine safe:

- **Claim is provisional.** `PENDING` is a reservation; it implies no authorization. A conditional INSERT on the business-op primary key — the developer's stable effect identity — is the only dedup mechanism.
- **`PENDING → ATTEMPTING` is one transaction.** In global lock order it locks the invocation row (`FOR UPDATE`), then revalidates **everything** inside the transaction: liveness, approval linkage and authority (`FOR SHARE`), grants (pinned-snapshot protocol), token, kill switches (`FOR SHARE`), eligibility and generation ownership (sequence fence where sequence-bound), and the expected effect state/owner/lease. It commits — **then** the provider send happens, outside any transaction.
- **Cancellation serializes against that transition.** Cancel-first: the transition fails, the `PENDING` record is withdrawn to terminal `CANCELLED`, nothing is attempted. Attempt-first: the attempt is in flight and is reported as such — it settles from provider evidence. Stale workers cannot send or overwrite a newer owner. A failed authorization creates no `ATTEMPTING` row at all.
- **Refusals are audited, not faked.** Kill switch, revoked grant, expired token → `effect-refused/<reason>` with tenant, invocation, op, rule, timestamp. Never an `ATTEMPTING` row, never secrets. That is what keeps "never attempted" distinguishable from "blocked by policy."
- **`UNKNOWN` is quarantine, not a retryable failure.** A timeout after possible provider acceptance settles `UNKNOWN`. Blind retry is prohibited; `UNKNOWN→ATTEMPTING` direct is prohibited. **Settle is atomic and operator-only** (§16.1): one predicated transaction `UNKNOWN → terminal`, including the `effect-settled/<outcome>` audit. Idempotent on (op, outcome, note) — an identical retry succeeds and backfills a missing audit; a different outcome/note is a conflict. Crash-before-commit leaves `UNKNOWN` (retry); commit-then-crash is covered by audit backfill. The two-transaction `RECONCILING` intermediate was removed because it could strand a row whose intended outcome lived only in memory.
- **`NOT_APPLIED` is the settled form of "the provider definitively did not do it."** It is retryable only through the full Authorize-attempt transaction — a fresh authorization, never a shortcut.
- **The honesty clause:** the contract claims *no duplicate confirmed business effects, no silently lost operations, no blind retries of unknown outcomes* — and explicitly claims **no exactly-once provider delivery**. Idempotency at the provider remains the developer's job.

**Settlement surface (operators):**

```
miniapp settle --op <business-op-id> --to confirmed|failed|not-applied --note <evidence>
```

Authenticates via canonical API keys, refuses non-`UNKNOWN` rows, writes the evidence note into the audit. Phase 1 includes **no automatic effects-reconciliation sweep** — the lifecycle sweeper never drives effect outcomes.

---

## 6. Identity: which key means what

Five identity scopes exist and conflating any two is a correctness bug (the writeup's §2.2 covers the taxonomy). The contract pins the mechanics:

- **Invocation identity** — the idempotency key on the invocation row's unique constraint. Claims are `INSERT ... ON CONFLICT DO NOTHING`; affected=1 wins.
- **Workflow identity** — `run_id`, the row's *own* submitted run, `UNIQUE (run_id) WHERE run_id <> ''` (migration 000006, with a preflight that fails with an actionable diagnostic on duplicate values rather than silently picking an owner).
- **Attachment** — a loser of a same-entity race never mints a second workflow and never overwrites the winner: it stores the winner's run in the nullable **`attached_run_id`** and follows the target's lifecycle. One mechanism serves both dedup (X-22) and genuine attach (X-08).
- **Effect identity** — `tenant:business_op_id`, global primary key, prefix gate that fails closed on cross-tenant collisions.
- **Canonical encoding (X-19)** — one helper, shared by entity derivation, effect scoping, advisory-lock keys, and switch-name scoping: percent-escape each part (`%`→`%25`, `:`→`%3A`), join with `:`. Injective, so escape collisions are impossible. Identity for legacy IDs without `%`/`:`; legacy raw multi-colon rows are orphaned on re-attempt (safe only because no production data exists), and any future encoding change requires dual-read or backfill before cutover.

**The pattern every durable write follows** (this is Rule 1 in code, including the attach case):

```go
// Mint only when the row owns no run at all.
result, _ := db.Exec(ctx,
    `UPDATE invocations SET run_id = $1
       WHERE id = $2 AND run_id = '' AND attached_run_id IS NULL`, runID, invID)
if result.RowsAffected() == 0 {
    current, _ := loadInvocation(ctx, invID)
    if current.RunID != "" {
        return Handle{RunID: current.RunID}, nil        // adopt the owner's run
    }
    if current.AttachedRunID != nil {
        return Handle{RunID: *current.AttachedRunID}, nil // attached: follow the target
    }
    return ConflictError{}
}
```

---

## 7. The submission choke point and retry-as-re-entry

### 7.1 One door: `submitChecked` (§5)

Exactly one function is the sole path to `Async.Submit`. It:

1. Loads the current invocation row (re-entry, not a fresh decision).
2. Resolves the tool's approval requirement **from the admitted manifest** — never from caller flags.
3. If gated: requires a **live persisted linkage** — `pending` (workflow-start permission: submit solely to enter the durable wait) or `decided/approve` with matching intent hash, authority, tenant, and unexpired (business-execution permission). Missing linkage → `errApprovalRequired` (the caller re-parks under the atomic park). Decided-against, expired, or mismatched → terminal refuse, never submit blind.
4. If ungated: submit.
5. Mints a run ID only when both `run_id` and `attached_run_id` are empty, via predicated adoption (§6 above).
6. Fails closed *inside the workflow too*: the executor's `Start`/`translate` rejects a gated tool whose request carries no approval ID — so no executor-side path can run unapproved work even if a future service caller bypasses the check.

**How the "only one door" claim is proved:** an audit verified `Async.Submit` has exactly two non-test call sites (both routed through the choke point), `dbos.RunWorkflow` has exactly two (reachable only via `Executor.Submit`, which requires a runtime-minted run ID), and no `Enqueue` callers exist outside tests. A grep test asserts the non-test caller set equals the audited set — any new call site fails CI. That is Rule 6 applied to architecture.

### 7.2 Every retry is a re-entry (§6)

There is no `isRetry` flag anywhere in the contract. A retry loads persisted state and dispatches on what it observes:

| Path | Dispatches on |
|---|---|
| Invoke submit/replay | `pending` (first claim), `running`/`waiting_event` (choke-point re-entry), `waiting_approval` (approval poll), terminal (verbatim replay / terminal error) |
| Approval create/poll | missing (create), `pending` (wait), decided (grant/deny/expire converge — including runs never submitted), `cancelled` |
| Submit-if-missing recovery | parked-without-run → submit via the choke point with the stored linkage, then poll |
| Signal redelivery | unknown run (error), unavailable family (explicit error), live run (`signal/accepted`), terminal run (distinct `already-terminal` no-op), quarantined run (`run-quarantined`; resume first) |
| Effect claim/settle | no row (provisional claim), `PENDING` (authorize-or-withdraw), `ATTEMPTING` (adopt/quarantine by lease), settled incl. `CANCELLED` (read), `NOT_APPLIED` (retryable re-attempt via Authorize), `UNKNOWN` (operator settle only) |
| Cancellation redelivery | live (cancel), `cancelled` (redeliver stop + report), terminal-other (refuse) |
| Run attachment | unattached (claim/attach), attached (follow target; re-resolve on supersede) |

This is what closes X-01 and X-02 structurally: a retry after a partial failure cannot skip a gate (there is only one gate, and it re-reads persisted state) and cannot strand a state (every observed state has a defined convergence handler, including "parked but never submitted").

---

## 8. Lock discipline (§4)

Two rules, then the order.

1. **Never hold a PostgreSQL transaction, row lock, or advisory lock across an external provider send, a DBOS submission, a long wait, or a signal wait.** All guard-and-act scopes hold locks only across local SQL.
2. **Global acquisition order** for any multi-row transaction:
   1. Invocation row
   2. Approval row
   3. Sequence/generation row
   4. Policy-control rows (kill switches; grant content pinned via the §10 Go `RLock` *before* the transaction begins)
   5. Effect row
   6. Advisory entity locks — acquired **outermost-first**, before any transactional row/table lock, and never acquired while holding them.

The no-cycle argument is short and worth internalizing: exactly one production site acquires advisory locks (the Class A entity lock, always as the first lock operation of a fresh transaction); therefore no path ever takes an advisory lock while holding PG locks; advisory waiters hold no PG locks, so a holder waiting on PG locks (ordered 1→5) can never wait transitively on its own waiters. Nested `Perform` calls inside a Class A handler take PG locks 1→5 under the held advisory lock, adding no cycle. Deadlock-freedom comes from this construction **plus** concurrency tests (cancel-during-claim/takeover/settle/park/expiry) run to completion under `statement_timeout` — a deadlock fails the test, it does not hang it. The order list alone is not the proof.

Representative orders from the transaction table (§4.2): effect authorization = Go `RLock` → 1 → 2 → 3 → 4 → 5; cancel = 1 → 2 → 5 in one transaction, executor stop after commit; `Decide` = row lock on one approval row; sweep steps run as separate sequential transactions, never holding a lock across steps; a takeover never retains an effect-row lock while beginning a fresh authorization; reporting paths are lock-free reads and never mutate.

---

## 9. The lifecycle sweeper (§8)

A framework worker — on startup, on a tick (default 30s, configurable), and opportunistically on `Status`/poll paths; tests invoke it directly for determinism. Three operations, strictly invocation/approval scope:

- **S1 — lost-signal redelivery.** Approval `decided` + invocation still `waiting_approval` + workflow nonterminal → re-send the decision hint via the verified-signal op. Harmless duplicates are fine: workflows re-read the table, which is authoritative.
- **S2 — expiry observation.** Approval past `expires_at` + invocation `waiting_approval` → predicated expiry marking + signal + invocation settlement, in one transaction taking locks 1→2, signaling only after commit. Covers the workflow parked on a long wait: the wake comes from the sweeper, not its own timer.
- **S3 — settlement discovery.** Live records with a run (own or attached) whose target executor status is terminal → the shared `settleTerminal` op. Satisfies "workflow completed and nobody polled" without any caller polling; crash windows close on the next sweep.

**The guarantee, stated honestly:** an event is observed within one sweep interval *while* at least one healthy sweeper runs with PostgreSQL and the relevant DBOS family available, and the batch processor keeps up with the backlog. Outages and excess backlog delay convergence — visibly, with counts, never silently. A complete outage has no finite bound. Safety does not depend on convergence: the durable gates (a settled invocation authorizes no new effect) hold regardless.

**What the sweeper never does:** drive effect outcomes. No `UNKNOWN` settlement, no reconciliation sweep — `Settle` stays an operator action with evidence. Expiry behavior is a fixed six-step sequence (persist expiry and settle the invocation authoritatively → the effect gate henceforth rejects → send the expiry notification → request cooperative cancellation best-effort → expose and retry failures → never claim confirmed-stopped from a requested cancel).

**Operational consequence:** a stopped sweeper regresses lost-signal recovery, expiry observation, and settlement discovery to pre-sweeper behavior. It is recoverable on restart and visible as backlog — alert on it.

---

## 10. Workflow/run identity and the executor surface (M4)

- **Verified signals, 5-way outcome taxonomy.** The signal operation verifies workflow existence (`RetrieveWorkflow`) *then* sends, and reports one of: unknown run (error); unavailable family (explicit error); `signal/accepted` (live runs only); `already-terminal` (a distinct no-op, never ordinary success); processed (claimed only with actual processing evidence, never from a nil `Send`). A residual Retrieve→Send TOCTOU is documented: completion between the two yields a harmless buffered no-op, reported as such when detectable. This taxonomy is what killed the false-success class (X-06).
- **Status mapping.** Dead-lettered runs map to invocation `quarantined` — nonterminal and operator-actionable — never silently failed; all sync errors are preserved, never swallowed (terminal state stays authoritative; staleness is explicit).
- **Shutdown honesty.** On `dbos.Shutdown` failure, the pool is **not** closed: loud diagnostics, non-zero exit in production, `t.Errorf` with no hang in tests. Closing a pool under active workers is a prohibited invariant.
- **Version affinity.** Per-family `AppName` and pinned `ApplicationVersion`; the behavioral proof is exact five steps: v1 pending → stop v1 → run only v2 → prove v2 neither executes nor advances the v1 workflow → restart v1 and prove it recovers and executes. SDK citations are mechanism notes, never substitutes for the test. Foreign-version pending workflows are visible at startup, periodically, and as queryable counts.
- **Run routing cache.** The in-memory `runs` map is bounded (cap + eviction); the durable `run_id`/`attached_run_id` columns are authoritative — the cache never is.

---

## 11. Sequences, attachment, and the generation fence (M5)

A Class D probe advances one entity through **generations**. Claiming a generation is a partial-unique `INSERT ... ON CONFLICT DO NOTHING` — at most one `active` generation per (tenant, subject); the loser of the claim race attaches (below). A stale `active` generation (past its wake time + grace, no heartbeat) is superseded by a predicated update on the sequence row.

**Genuine attach (X-08).** The losing invocation stores the winner's run in `attached_run_id`; no second workflow is submitted for the sequence. The loser's status follows the target's terminal outcome truthfully. Six acceptance checks are enumerated in the contract; the headline ones: concurrent invocations produce one canonical workflow; the loser gets a valid target-linked handle; target failure is never reported as success; superseded generations send nothing further.

**The generation fence (CR-04).** Supersede alone cannot stop an in-flight provider call — so the fence is enforced *at the authorization boundary*, inside the M3 authorize transaction: the sequence row is taken `FOR SHARE` and `status='active' AND generation=?` is verified against the fence identity carried in the workflow's trusted execution context (never caller-supplied free text). Supersede-first → the old generation can never authorize another attempt, and a refusal audit (`effect-refused/superseded`) is written. Attempt-first → the attempt is in flight and reported honestly; in-flight provider work at supersede time cannot be recalled — the contract records this rather than describing cancellation as cooperative recall. The fence is a framework primitive: future Class D miniapps inherit it through `Perform`, never reimplement it.

**DETACH-ONLY cancel (owner-approved).** Cancelling an *attached* invocation: (1) transitions only that row to `cancelled`; (2) never calls `CancelWorkflow` on the shared target; (3) never modifies the owning invocation; (4) never affects other observers; (5) preserves `attached_run_id` immutable; (6) audits attribution (attached invocation + target); (7) is idempotent; (8) requires normal tenant/authority checks (cross-tenant refused). The row stays `cancelled` and stops following the target. Cancelling the *owner* cancels the target (the owner owns the workflow) and observers follow its outcome.

---

## 12. Cancellation (M6) — durable intent first

Cancel is one transaction in lock order 1 → 2 → 5: the predicated invocation update `WHERE status IN (live)` (the durable intent) commits **together with** pending-approval settlement and the batch withdrawal of `PENDING` effects to `CANCELLED`. Only after commit is the executor stop requested (cooperative: in-flight steps complete per the SDK; the ops doc says "cancelled workflows stop at the next step" — never "interrupted"). The cancel response reports `in_flight` (`ATTEMPTING`, settles from evidence) and `withdrawn` (`PENDING→CANCELLED`, never attempted) as **separate, non-conflatable** categories. Executor down? The intent stands, the cancel is retryable, and redelivery via repeat Cancel / Status / sweep converges. Post-termination, an attached row reports its own `cancelled` while other observers report the target's outcome; historical `attached_run_id` remains for audit.

---

## 13. Control plane and grants

- **Kill switch (D1).** Flipping is a privileged setter function (owner-only `EXECUTE`) invoked by a separate `switch` operator command over the **control connection** — the serving process never holds the control DSN (`MINIAPP_CONTROL_DATABASE_URL`). The serving app role has `REVOKE`d write access (proved with ephemeral roles over genuine restricted connections). A flip-audit trigger is `ENABLE ALWAYS` — and the wording stays honest: it fires for every writer **except an owner/superuser who deliberately disables it**; it is not "unbypassable." Startup rejects unsafe role configuration instead of silently running elevated. Production running as the restricted role is a recorded deployment residual.
- **Grants (§4.1, pinned snapshot).** Grants are file-loaded, process-lifetime configuration — they cannot join a PostgreSQL transaction. The trusted mechanism: an `RWMutex` in `auth.Policy`; authorization takes `RLock` across its read set (short, local, a Go mutex — so the no-PG-lock-across-external-calls rule is unaffected); the revocation setter takes `Lock`. In-process revocation races are closed. **Named residual:** no hot reload; cross-replica coherence is a deployment property (rolling-restart window). A wait→revoke→refuse E2E test and a revocation-racing-authorization test pin the behavior.

---

## 14. Fault injection and evidence discipline (§9, §13)

- **The injector.** `testkit/fault.Injector` — test-only, per-test isolated, race-safe, nil-safe (nil never fires; a test asserts nil-after-`New` for every constructor). It fails a named operation on its Nth attempt (`FailAt` one-shot, `FailFrom` persistent). Seams: test-side wrappers (`Async`, `Channel`, `Policy`) plus one field per service (`TestFault`), including the executor's — `Shutdown` sits outside the `AsyncExecutor` interface, so the concrete executor carries the fourth seam.
- **What the ops model** — the contract names ~20 of them, each with a firing/boundary/propagation test. The instructive ones: `async-submit` (run persisted, submit failed — the X-02 setup), `claim-response` (lost response after claim commit — retry converges on the same record), `settle-commit` (sent once, stays `ATTEMPTING`, ages to `UNKNOWN`), `provider-send` (zero delegate sends, attempt settles `FAILED`), `executor-shutdown` (error reaches caller, workers untouched). Each one corresponds to a real crash boundary a miniapp will hit in production.
- **Red/green.** HIGH + X-07 tests run red against an isolated pre-remediation snapshot and green on the remediated tree — same test source both sides, all four digests recorded (implementation source, test harness, full tree, exact command). No artificial red runs for brand-new features; discriminate on invariants instead.
- **Provenance.** Every evidence entry binds to digests (sorted-`sha256sum` manifest method, byte-identical across phases); the SDK is pinned at v1.9.0 with its own digest; historical undigested claims are marked irreproducible, never retrofitted.
- **Test hygiene.** Barrier synchronization + separate connections, never sleeps; `-race` on race-sensitive suites; `statement_timeout` so deadlocks fail tests; real-PostgreSQL kill/restart harness for DB invariants; CI provisions a database and proves DB-backed tests ran.
- **Corrections are never silent.** If implementation discovers a contract inaccuracy, the correction is recorded in the contract with the original claim, the corrected claim, and the rationale (the amendment log in §16 is exactly this ledger — including reversed wordings like the advisory-lock rule and removed machinery like `RECONCILING`).

---

## 15. Guarantees and non-guarantees

Read the left column as what you may build on; the right column as what you must design around.

| Guaranteed (with stated conditions) | Not guaranteed — deliberately |
|---|---|
| Contested mutations are single-writer (predicate + affected-row check, or row lock under the documented protocol) | Exactly-once provider delivery — idempotency keys and reconcile-by-read stay developer work |
| `ATTEMPTING` is the durable point of no return: no new `ATTEMPTING` transition after durable cancel, full revalidation inside the transition transaction | Killing a send already in flight after cancel commits — the contract reports it in-flight and settles from evidence instead |
| `UNKNOWN` is never blindly retried; settlement is atomic, evidence-backed, operator-only | Automatic effect reconciliation in Phase 1 — no sweep settles effects |
| Lost signals, expiry, and unpolled settlement converge within a sweep interval while a healthy sweeper + PostgreSQL + DBOS family are available and backlog is within capacity | Convergence under a complete outage or unbounded backlog — delayed, visible, no finite bound |
| Every nonterminal state has a defined convergence path (retry-as-re-entry) | Silent self-healing — every failure path is audited and attributable, some paths end in "operator must act" |
| Run IDs are adopted, never overwritten; attachments follow their target truthfully | An attached invocation implicitly cancelling a shared workflow — DETACH-ONLY, always |
| Dead-lettered workflows stay resumable (`quarantined`) until an operator decides | Automatic resume — quarantine moves only via explicit, audited operator action |
| In-process grant revocation has no race (pinned snapshot) | Cross-replica instant revocation — rolling-restart window, a recorded deployment residual |

---

## 16. Rules of engagement for miniapp developers

**Do:**

- Use the stable effect identity per business operation (`tenant:business_op_id`) — it is the dedup key the whole ledger leans on.
- Treat the approval table as truth and the signal as a hint; re-check approval state immediately before consequential actions (plan4.2 §15.1 requirement).
- Dispatch every retry on persisted state; never branch on a "retry" flag.
- Handle the defined conflict errors: on a lost race you reload and reconcile — the error names exactly what happened.
- Declare gating, expiry, and authority in the manifest — it is the source of truth at the choke point.
- Design for idempotent redelivery: signals, cancels, and settlements may arrive more than once and must converge.
- Use the `FaultInjector` in your fixtures at the real boundaries (submit, signal, commit, provider call), barrier-synced, with real PostgreSQL.

**Don't:**

- Write ad-hoc lifecycle `UPDATE`s. Every transition needs a contract row (predicate, lock scope, atomicity, audit, failure exit, test); review rejects the rest.
- Treat a standalone liveness read as authorization — the check-live gate is inside the authorization transaction; standalone liveness callers were eliminated.
- Assume a pending approval authorizes execution — only a verified granted decision does, revalidated in the authorize transaction.
- Poll for correctness — the sweeper makes polling optional; but do not make guarantees *depend* on your own timers either (a long `Recv` is woken by the S2 signal, not by your timeout).
- Retry `UNKNOWN`, settle without evidence, or restore a database without reconciling external effects — these are the contract's named "never" operations.
- Assume cooperative cancellation interrupts a running step — it stops at the next step boundary; in-flight provider work cannot be recalled.

---

## Appendix A — Audit vocabulary

| Audit event | When | Property |
|---|---|---|
| `invoke/dispatched` | first claim | mandatory; failure → retryable |
| `invoke/duplicate` | same-key retry | best-effort |
| `approval-requested/pending` | atomic park | exactly once per invocation |
| `approval-decided` | `Decide` commit | at commit, before signal |
| `approval-expired` | predicated expiry marking | shared by poll + sweeper paths |
| `run-settled/<terminal>` | `settleTerminal` win | exactly once, shared by Status + sweeper S3 |
| `effect-refused/<reason>` | refusal before any claim | attributed; no `ATTEMPTING` row; never secrets |
| `effect-withdrawn` | cancel-won `PENDING→CANCELLED` | distinct from failure |
| `effect-settled/<outcome>` | operator Settle | evidence note; idempotent; backfill on retry |
| `cancel/cancelled` | durable cancel intent | reason + `in_flight` + `withdrawn` listed separately |
| `attach/attached` | genuine attach | target run identified |
| `resume/resumed` | quarantine resume | operator + run attributed |
| `signal/accepted` / `signal/already-terminal` | verified-signal outcomes | `accepted` for live runs only |
| `switch-flipped` | kill-switch flip | actor, reason, scope, before/after; trigger-written |

## Appendix B — Where things live in the contract

| Topic | Contract section |
|---|---|
| Root-cause assessment (the five bugs) | §1 |
| Six transition-integrity rules + two permissions | §2 |
| Machines, ownership, finding→owner map | §3 |
| Lock order, no-cycle proof, transaction table | §4–§4.2 |
| Submission choke point | §5 |
| Retry-as-re-entry handler table | §6 |
| Normative transition tables (M1–M6, D1) | §7 |
| Terminals, invariants, prohibitions | §7 (CR-05) |
| Sweeper design, bounds, ops | §8–§8.3 |
| Fault injection | §9 |
| Per-finding map (X-01–X-24) | §10 |
| Residual dispositions (F-01–F-14) | §11 |
| Migrations, roles, control DSN, settle verb | §12 |
| Evidence discipline | §13 |
| Amendment log (what changed and why) | §16 |
