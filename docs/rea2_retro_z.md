## The principle, restated for our stack

Rea's bet is that the document's complexity is not intrinsic to durable workflows — it is the cost of hand-rolling, per feature, machinery the engine already owns. DBOS provides the same primitive set Restate does (Postgres-backed step journal, workflow-ID identity, durable timers/events/queues, automatic recovery, app-version affinity). So the move is: **the miniapp framework does for DBOS what Rea does for Restate** — wrap the primitives once, with guardrails, so workflow authors write sequential code plus policy, never lifecycle plumbing.

Root cause addressed: the plan4.2 machinery is hard because it is authored per feature and contested at the SQL level. Either delete it where the engine covers it, or move it into one tested framework surface. Per-workflow code shrinks to business logic.

## Mapping: hand-built machinery → DBOS primitive → framework wrapper

| What the document hand-builds | DBOS primitive that replaces it | Miniapp-framework wrapper |
|---|---|---|
| Checkpoint recovery / "stage 4 of 7 failed" | Step journal + automatic recovery on executor start | Rule, not code: *every side effect is a step*; recovery registered at boot |
| Predicated SQL state machine for invocation lifecycle | The engine's own status (`PENDING/RUNNING/SUCCESS/ERROR/CANCELLED`) recorded transactionally with the journal | No app-side state machine; one typed status read / projection |
| Identity conflations (workflow vs. entity vs. dedup scopes) | Workflow ID as the single identity key; queue dedup IDs; idempotency keys | `GenerateDeterministicKey(domain, id)` + one `SubmitOrAttach` so callers cannot double-start |
| Approval table + sweeper / lost signals | Durable events (`set`/`get` with timeout) + durable timers | `WaitForApproval(key, timeout, reminders)` — the wait is engine state, crash-safe, no sweeper |
| Cancellation ordering ("record intent before stop") | Engine writes CANCELLED in the same DB before stopping execution | One cancel entry point with our DETACH-ONLY policy encoded; ordering is not author-owned |
| Saga / compensation ledger | Completed steps are already journaled and LIFO-replayable | `Saga.Add` analog: persist the compensation entry (as a step) *before* the action; one implementation, tested once |
| Generic stuck-invocation sweeper | Engine recovery + queue reaper cover crashes; business expiry is a durable timer inside the flow | Delete the generic sweeper; per-flow expiry = `WaitForApproval` timeouts |
| Observability surface | Engine tables + workflow events | `StatusData` projection (phase / progress / completed steps) written through one call, rendered by the miniapp UI |

## The guardrail layer we add on top (Rea-style)

- **Policy** `strict | warn | disabled`, auto-strict in CI. Runtime checks: raw network/DB I/O in the workflow body (must be steps), `time.Now()`/UUID/rand in the workflow body, direct writes to lifecycle tables, unbounded loops without durable sleep.
- **Deterministic helpers** that route nondeterminism through steps.
- **`RunEffect`** — the one thin ledger we keep. Wraps a step with a provider idempotency key and records an explicit `UNKNOWN` outcome on ambiguous provider timeouts. This is DBOS's honest gap (it replays steps; it cannot know whether the provider executed), so reconciliation stays ours — made uniform instead of bespoke.
- **Bounded loops/retries** (`While(max)`, `Retry(backoff)`) using durable sleep between iterations.
- **Control/data-plane tagging** at handler registration, enforcing the orchestration-vs-side-effect separation structurally.

## What stays developer-owned (honest boundary, unchanged)

Business predicates and stage logic; provider idempotency semantics and UNKNOWN reconciliation (uniform, not automatic); version affinity discipline on workflow changes; ops runbooks. DBOS is replay-based — any step can re-execute after a crash — so "is this side effect idempotent?" remains a human decision.

## What this does to plan4.2

Not deleted — demoted from "machinery every workflow needs" to "acceptance spec for the framework layer." The contested-mutation analysis (cancel-won, claim semantics, lock order) is exactly the correctness spec for the handful of wrapper functions above and for the residual cases the engine doesn't cover (the UNKNOWN ledger, cancelling already-completed work). Concretely: the six per-feature state machines collapse to engine status plus one reconciliation table; the 000006 preflight and sweeper guarantees shrink accordingly; P2 stays gated until a pilot proves the wrapper.

## Suggested next step

One pilot: take the heaviest existing flow, rewrite it on the wrapper (about five helpers plus policy), and diff — LOC deleted, machinery removed, which contract sections survive as framework tests vs. get dropped. That evidence decides the P2 gate.