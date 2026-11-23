## Goal

Fix the CI hang in `TestConcurrentTenantIsolation` (`spikes/rls_concurrent_test.go`, run `45115104815`) — proven pool starvation, not an RLS failure — and harden the test harness so this failure class surfaces fast with diagnostics instead of burning 180s silently.

## Success Criteria

- The test passes deterministically on 4-vCPU runners (CI shape), not just high-core machines.
- Pool sizing is derived from the test's own concurrency, with a guard asserting `MaxConns ≥ participants`.
- A worker failure before/during arming surfaces its real error in seconds, with an armed-count diagnostic — never a bare 120s+ timeout.
- No other suite test parks more transactions than its pool allows (sweep complete, each audited file either fixed or documented-clean).
- Timeouts, assertions, and the test's isolation semantics are unchanged (no weakened checks, no skips, no timing-based coordination).

## Approach

Accept prompt21's diagnosis — I verified every structural leg of it against the tree: 8 workers (`4 tenants × workersPerTenant = 2`) each pin one pool connection across the `ready` barrier via `store.WithTenant` → `pool.Begin`; the fixture pool uses bare `pgxpool.New` (no `MaxConns`, and `appRoleDSN` adds no `pool_max_conns`); pgx v5.13.0 source (the pinned version, read from module cache) defaults `MaxConns` to `max(4, NumCPU)` → 4 on CI runners; CI sets a plain DSN. Four workers arm, four block on acquire forever, the barrier never opens, the 180s backstop fires. Not a Postgres deadlock (no lock cycle exists; the detector would fire in ~1s), not RLS, not DBOS leakage.

The fix follows prompt21's prescription with two judgment calls of mine: (1) scope is the hang + harness hardening + suite sweep in; gob sentinel + dbos_version traceability out (follow-ups, explicitly not the cause). (2) The guard lives in-test with derived sizing rather than in CI env, so protection travels with the code; no CI workflow changes needed.

## Steps

1. **Reproduce red first.** `taskset -c 0-3 go test ./spikes/ -run TestConcurrentTenantIsolation -count=1 -v` (16-CPU local box needs constraining; `GOMAXPROCS` does not affect pgx). Needs a database: `/tmp` PG16 cluster persists, or Docker PG19 via `task test:integration`. Capture the 120–180s timeout as the red baseline. Do not proceed until red is observed.
2. **Size the pool from the test's concurrency** (`spikes/rls_concurrent_test.go`): hoist `workersPerTenant` to package scope, add a `concParticipants()` helper (`len(concTenants) * workersPerTenant`), build the app-role pool via `ParseConfig` + `NewWithConfig` with `MaxConns = participants + 2` (8 barrier-held + headroom for final counts/negative control). Derive, never hardcode, so a future tenant/worker change can't reintroduce starvation silently.
3. **Add the misconfiguration guard** in the test: fail immediately if `pool.Config().MaxConns < concParticipants()` with a message naming both numbers (pgx always materializes a concrete positive `MaxConns` at parse, so no zero-handling needed — verified in v5.13.0 source).
4. **Fail-fast barrier**: extend `waitFor` to also select on `errs` (already buffered at `total+1`, one send max per worker — no blocking risk), failing immediately with the worker's error plus armed count (`k/N`); cut the ctx backstop 180s→30s.
5. **Sweep the suite** for the same shape (connections pinned across a barrier vs pool size): `effects/concurrency_test.go` (8 workers through `gateChannel` — verify whether a pool conn is pinned while blocked), `runtime/service_test.go` (`blockChannel`), `runtime/x03_test.go`, `runtime/m01_test.go`, `approvals/x04_test.go` (DB lock-gate holder + racers vs default pools). Fix same-shape hits via explicit `MaxConns` + guard; document each audited-clean file in the evidence row.
6. **Record evidence**: append a row to `docs/evidence/README.md` per repo convention (red/green runs, revision, environment). Note: the file has 3 uncommitted lines from in-flight work — append after them, rewrite nothing.

## Validation Plan

- Red proof: step 1 output showing the hang under constraint (120s+ timeout, `deadlock waiting for armed transactions`).
- Green proof: same constrained command passes post-fix (~seconds); repeat `-count=3`.
- Full package: `go test ./spikes/ -count=1` with DB; full suite `go test -p 1 ./...` with DB; `go test -race` on touched packages; `gofmt`, `go vet ./...`, `bash scripts/checks.sh`.
- Guard simulation: temporarily set `MaxConns` below participants (or point at it in review) to show immediate clear failure — do via a scratch run, not a committed test, unless a cheap deterministic unit test falls out naturally.
- Final proof the local runs can't give: a CI run on the pushed branch showing `spikes` green. This step requires push authorization and is tracked as follow-up, not part of `go`.

## Risks

- **Flaky-timing accusation on the 30s backstop**: mitigated — all known paths now fail fast deterministically (worker error → immediate; success path takes seconds locally); the backstop only guards unknowns. Repeat runs + race detector in validation.
- **Sweep scope creep**: bounded — the five files above are the complete inventory of barrier/gate patterns found by search; any new pattern discovered during implementation gets reported before being pulled into scope.
- **WG interplay**: `docs/evidence/README.md` has uncommitted lines from parallel work; step 6 appends only.
- **Explicitly out of scope** (named follow-ups, not this change): typed DBOS error sentinel for the gob warning, `dbos_version=unknown` traceability, any CI workflow edits, any business-semantics changes.

Open questions: none.
