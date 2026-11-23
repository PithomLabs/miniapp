## issue

what is the underlying problem per below https://github.com/PithomLabs/miniapp/actions/runs/45134002993/job/137555800027, what insights can you give me? Run go test -p 1 ./...
  go test -p 1 ./...
  shell: /usr/bin/bash -e {0}
  env:
    MINIAPP_POLICY: strict
    CI: true
    TEST_DATABASE_URL: ***localhost:5432/miniapp_dev
    TEST_DBOS_DATABASE_URL: ***localhost:5432/miniapp_dev
ok  	github.com/PithomLabs/miniapp/analyzer/determinism	1.152s
ok  	github.com/PithomLabs/miniapp/approvals	1.011s
ok  	github.com/PithomLabs/miniapp/audit	0.047s
ok  	github.com/PithomLabs/miniapp/auth	0.003s
?   	github.com/PithomLabs/miniapp/cmd/determinism-lint	[no test files]
?   	github.com/PithomLabs/miniapp/cmd/miniapp	[no test files]
?   	github.com/PithomLabs/miniapp/cmd/preflight	[no test files]
ok  	github.com/PithomLabs/miniapp/contract	0.007s
?   	github.com/PithomLabs/miniapp/deploy	[no test files]
ok  	github.com/PithomLabs/miniapp/effects	0.720s
?   	github.com/PithomLabs/miniapp/executor	[no test files]
ok  	github.com/PithomLabs/miniapp/executor/classa	0.661s
ok  	github.com/PithomLabs/miniapp/executor/dbos	37.687s
ok  	github.com/PithomLabs/miniapp/infra	0.349s
ok  	github.com/PithomLabs/miniapp/internal/config	0.003s
ok  	github.com/PithomLabs/miniapp/internal/store	0.003s
ok  	github.com/PithomLabs/miniapp/miniapps/crm-read	0.032s
ok  	github.com/PithomLabs/miniapp/preflight	0.004s
ok  	github.com/PithomLabs/miniapp/probe/classd	39.047s
ok  	github.com/PithomLabs/miniapp/registry	0.005s
ok  	github.com/PithomLabs/miniapp/runtime	3.243s
?   	github.com/PithomLabs/miniapp/spikeharness/mcpstdio	[no test files]
time=2026-11-23T07:52:58.669Z level=INFO msg="Initializing DBOS context" app_name=spike dbos_version=unknown
time=2026-11-23T07:52:58.669Z level=INFO msg="Connecting to system database" database_url=***localhost:5432/miniapp_dev schema=dbos
time=2026-11-23T07:52:58.695Z level=INFO msg="DBOS launched" app_version=ccfef224ee28a5a84610b87c29dc96e5ec6bd0ca5395753e869717ae5701ef16 executor_id=local
time=2026-11-23T07:52:58.744Z level=INFO msg="Initializing DBOS context" app_name=spike dbos_version=unknown
time=2026-11-23T07:52:58.744Z level=INFO msg="Connecting to system database" database_url=***localhost:5432/miniapp_dev schema=dbos
time=2026-11-23T07:52:58.770Z level=INFO msg="DBOS launched" app_version=ccfef224ee28a5a84610b87c29dc96e5ec6bd0ca5395753e869717ae5701ef16 executor_id=local
time=2026-11-23T07:52:58.780Z level=WARN msg="workflow error type cannot be gob-encoded; persisting its message only: errors.Is/errors.As will not match this error when read back from the database" error_type=*errors.errorString error="spike forced failure" encode_error="failed to gob encode data: gob: type not registered for interface: errors.errorString"
time=2026-11-23T07:52:58.785Z level=WARN msg="workflow error type cannot be gob-encoded; persisting its message only: errors.Is/errors.As will not match this error when read back from the database" error_type=*errors.errorString error="spike forced failure" encode_error="failed to gob encode data: gob: type not registered for interface: errors.errorString"
time=2026-11-23T07:52:58.811Z level=INFO msg="Initializing DBOS context" app_name=spike-fam-a dbos_version=unknown
time=2026-11-23T07:52:58.811Z level=INFO msg="Connecting to system database" database_url=***localhost:5432/miniapp_dev schema=dbos
time=2026-11-23T07:52:58.838Z level=INFO msg="DBOS launched" app_version=027f789f1319d01c371d53c0ddd8a8e5982c536f8dfdad41b829e3934e97efc5 executor_id=local
time=2026-11-23T07:52:58.856Z level=INFO msg="Initializing DBOS context" app_name=spike-fam-b dbos_version=unknown
time=2026-11-23T07:52:58.856Z level=INFO msg="Connecting to system database" database_url=***localhost:5432/miniapp_dev schema=dbos
time=2026-11-23T07:52:58.880Z level=INFO msg="DBOS launched" app_version=7e14f8e5bef01349e462723f616c561c34a3d45b1fc3cd7c85c8a3502f4ecc92 executor_id=local
time=2026-11-23T07:52:59.134Z level=INFO msg="Initializing DBOS context" app_name=spike dbos_version=unknown
time=2026-11-23T07:52:59.134Z level=INFO msg="Connecting to system database" database_url=***localhost:5432/miniapp_dev schema=dbos
time=2026-11-23T07:52:59.162Z level=INFO msg="Recovered pending workflows" count=1
time=2026-11-23T07:52:59.162Z level=INFO msg="DBOS launched" app_version=ccfef224ee28a5a84610b87c29dc96e5ec6bd0ca5395753e869717ae5701ef16 executor_id=local
--- FAIL: TestConcurrentTenantIsolation (180.01s)
    rls_concurrent_test.go:196: deadlock waiting for armed transactions: context deadline exceeded
FAIL
FAIL	github.com/PithomLabs/miniapp/spikes	181.724s
ok  	github.com/PithomLabs/miniapp/testkit	3.082s
ok  	github.com/PithomLabs/miniapp/testkit/fault	0.002s
?   	github.com/PithomLabs/miniapp/testkit/faultwrap	[no test files]
ok  	github.com/PithomLabs/miniapp/testkit/mockcap	2.127s
?   	github.com/PithomLabs/miniapp/testkit/stack	[no test files]
ok  	github.com/PithomLabs/miniapp/transport	2.531s
?   	github.com/PithomLabs/miniapp/transport/cli	[no test files]
?   	github.com/PithomLabs/miniapp/transport/mcp	[no test files]
?   	github.com/PithomLabs/miniapp/transport/rest	[no test files]
FAIL
Error: Process completed with exit code 1.




## triage

The only failure is `TestConcurrentTenantIsolation` in the `spikes` package: `rls_concurrent_test.go:196: deadlock waiting for armed transactions: context deadline exceeded`. Everything else passed. The 180.01s runtime means a hard-coded 180s context timeout fired, so the test never finished rather than failing an assertion. It also means no isolation violation was observed. The test hung before it could check anything.

I can't see the test source, so what follows is inference from the log.

## What the message says

"Armed transactions" most likely means a barrier pattern: N transactions each `BEGIN`, set the tenant context (e.g. `SET LOCAL app.tenant_id`), signal "armed", then wait until all N are armed before querying at the same moment. The test is stuck because fewer than N transactions ever reached "armed". The barrier is application-level, so Postgres can't see it and its deadlock detector never fires. Only your timeout catches it.

## Likely causes, most to least probable

1. **Pool size smaller than N.** Each armed transaction holds its connection while waiting for the others. If N exceeds the pool's max connections, the remaining goroutines block on acquire and the barrier never completes. pgx's `pgxpool` defaults `MaxConns` to `max(4, runtime.NumCPU())`. A GitHub runner has few cores and a dev laptop has many, which would explain "passes locally, hangs in CI". If you set `MaxConns` explicitly, check it against N plus anything else sharing the pool.

2. **A goroutine failed before arming, and the error was swallowed.** If a worker hits a connection error (e.g. `too many connections`) or a failed `SET LOCAL`, returns early, and doesn't cancel the others, the rest wait forever. You get "deadlock" instead of the real error.

3. **A lock held by an armed transaction blocks another one.** If tenants write rows that collide on a global unique index (the same business key under different tenants, say), tenant B blocks on tenant A's uncommitted insert, while A waits at the barrier for B. If that's what is happening, it is also a real design finding: uniqueness that spans tenants leaks information through blocking. Schema or policy DDL running during the test would produce the same hang, since it takes heavy locks.

4. **Leftover connections or state in the shared database.** Several DBOS contexts launch in this package against the same `miniapp_dev` that `TEST_DATABASE_URL` points to. If those contexts aren't shut down, their pools stay open and eat into the connection budget. The `Recovered pending workflows count=1` line also shows state surviving between tests in this database.

## Smaller observations

- The gob warning is not the cause, but it matters for your durable-workflow design. After a restart, `errors.Is/As` won't match the original error, because the error is persisted as a message only. Use registered error types or stable error codes in workflow results if recovery logic depends on error identity.
- `dbos_version=unknown` suggests build info isn't being embedded. It's harmless but worth fixing for traceability.
- The 180s timeout wastes CI time. A barrier like this should fail fast, within about 15–20s.

## How to confirm

- Reproduce locally with `GOMAXPROCS=2` or a small pool (`pool_max_conns=4` in the DSN), with `-count=1 -race`. If it hangs, it's cause 1.
- Make the barrier fail fast: use `errgroup.WithContext` so any worker error cancels the rest, and have workers report which stage they reached.
- On timeout, dump `pg_stat_activity` and `pg_locks` before failing, so you can see who is blocked on what.
- Log `pool.Stat()` (acquired and max connections) and the connection count at the start of the test.
- Check that every DBOS context in `spikes` is shut down, or give this test its own database or schema.

If you paste the barrier section of `rls_concurrent_test.go` (around line 196) and the pool config, I can pin down which of these it is.

This is now confirmed as the first cause on my list: pool starvation. It isn't a Postgres deadlock, and the other hypotheses are ruled out by what the file shows.

## Root cause

The test needs 8 connections held open at once (`4 tenants × workersPerTenant = 2`). Each worker's `WithTenant` transaction holds its connection while parked on the `releaseReads` barrier.

But the pool is created with `pgxpool.New(ctx, appRoleDSN(...))`, and unless `appRoleDSN` adds `pool_max_conns` (I can't see it), pgx defaults `MaxConns` to `max(4, runtime.NumCPU())`. On a CI runner that is 4. So:

1. Four workers get connections, arm, and send on `ready`.
2. The other four block inside `WithTenant`, waiting to acquire a connection. They can't, because the first four won't release until the barrier opens.
3. The main goroutine waits for 8 `ready` signals and only ever gets 4.
4. After 180s, `waitFor` fires: `deadlock waiting for armed transactions`.

It passes on any machine with 8 or more logical CPUs and hangs on anything smaller, which is why it looks fine locally.

## What's ruled out

- **Lock collisions:** there are none. The unique key is a `serial` id, and sequences don't block.
- **RLS or policy problems:** the test never got far enough to evaluate any.
- **DBOS or connection leaks from other tests:** they aren't needed to explain this.

## Correction to my earlier advice

I suggested reproducing with `GOMAXPROCS=2`. That won't work, because pgx uses `runtime.NumCPU()`, which ignores `GOMAXPROCS`. Use `taskset -c 0-3 go test ...` or `pool_max_conns=4` in the DSN.

## Fix

Make the pool size explicit and derived from the test's own concurrency. This also needs `workersPerTenant` hoisted out of the function.

```go
const workersPerTenant = 2

func concParticipants() int { return len(concTenants) * workersPerTenant }

// in setupConcIsolation, replacing the pgxpool.New for the app role:
cfg, err := pgxpool.ParseConfig(appRoleDSN(t, dsn, "spike_app_conc", appPassword))
if err != nil {
	t.Fatal(err)
}
cfg.MaxConns = int32(concParticipants()) + 2 // barrier holds one conn per worker; +2 headroom
pool, err := pgxpool.NewWithConfig(ctx, cfg)
```

Then add a guard in the test so a misconfigured pool fails immediately with a clear message instead of hanging:

```go
total := concParticipants()
if got := int(fx.pool.Config().MaxConns); got < total {
	t.Fatalf("pool MaxConns=%d < %d barrier participants: barrier would starve", got, total)
}
```

## Two latent weaknesses

1. **Errors before arming still cause a hang.** `errs` is only read after `wg.Wait()`. If any worker fails before sending on `ready`, for example if `WithTenant` fails on `BEGIN`, `waitFor(ready, ...)` blocks until the timeout and the real error is never shown. Select on `errs` inside `waitFor`:

   ```go
   select {
   case <-ch:
   case err := <-errs:
   	t.Fatalf("worker failed before %s: %v", what, err)
   case <-ctx.Done():
   	t.Fatalf("timed out waiting for %s: %v", what, ctx.Err())
   }
   ```

2. **The 3-minute backstop is too generous.** Nothing in this test should take more than a few seconds, so 30s would fail faster in CI without risking false positives.

Also note that `t.Fatalf` from the main goroutine leaves workers blocked until the deferred `cancel()` runs. That is fine here, but it means the fixture's cleanup runs while their connections may still be open.

The same rule applies to the rest of your suite: any test that parks N transactions at a barrier needs `MaxConns >= N`. It's worth grepping the other spikes for the same pattern.


## Z

I'll ground this in the actual test rather than guessing — let me find the failing test file in the sandbox.
The repo isn't in this sandbox (only my `download/`, `scripts/`, `skills/`, `upload/`). Let me try pulling the actual test source from GitHub before diagnosing:
Repo is reachable via curl — the WebFetch timeout was a tool issue. Downloading the test file for diagnosis (read-only):
The test source confirms the failure point — line 196 is the first barrier `waitFor(ready, 8, "armed transactions")`. Workers must all hold open transactions simultaneously, which means each pins a pool connection across the barrier. Let me verify `store.WithTenant`'s connection behavior and the pool sizing before concluding:
`WithTenant` confirmed: `pool.Begin` pins one connection for the whole transaction — and `fn` spans both barrier waits, so each armed worker holds a connection across them. The test creates its pool with bare `pgxpool.New(ctx, dsn)` (no `MaxConns`), so the pgx v5 default applies: `max(4, NumCPU)`. Let me confirm the CI runner's CPU count and the DSN the workflow actually passes:
API rate limit hit — but the raw host may still respond. One quick attempt at the workflow file, then I'll conclude with what I have (the diagnosis doesn't depend on it):
I pulled the actual test (`spikes/rls_concurrent_test.go`), `internal/store/store.go`, and the workflow file off the repo. This is a **deterministic test-fixture deadlock: connection-pool starvation**, not an RLS failure and not a product bug.

## The underlying problem

`TestConcurrentTenantIsolation` spawns **8 workers** (4 tenants × 2), and each worker must hold an **open transaction across two barriers** (`ready` → `releaseReads` → `doneWrites` → `releaseCommit`). Since `store.WithTenant` does `pool.Begin(ctx)` and the transaction spans the whole barrier wait, every armed worker pins one pool connection until release. So the test *requires* 8 simultaneous connections.

But the fixture builds its pool with bare `pgxpool.New(ctx, dsn)` — no `MaxConns` — and pgx v5 defaults to **`max(4, runtime.NumCPU())`**. The workflow runs on `ubuntu-latest` (4 vCPUs) with a plain DSN (`postgres://miniapp:miniapp-ci@localhost:5432/miniapp_dev`, no `pool_max`), so the pool is **capped at 4 connections**. The circular wait:

1. 4 workers acquire a connection, arm, and signal `ready` — then hold their connections while waiting on `releaseReads`
2. The other 4 workers block forever in `pool.Acquire` — no connection will ever be freed, because the armed 4 only commit after `releaseReads` closes
3. `releaseReads` only closes after all 8 signal `ready` — which never happens
4. The backstop at `rls_concurrent_test.go:196` fires at the context deadline → `deadlock waiting for armed transactions: context deadline exceeded` after **180.01s = exactly the 3-minute context** (rls_concurrent_test.go:114, 196)

The failure is **CPU-count-dependent**: on a 16-core dev machine the default is 16 ≥ 8 and the test passes; on any runner with <8 vCPUs it is guaranteed to fail. Classic "green locally, red in CI."

## Why the evidence fits

- The timeout lands on the *first* barrier (`armed transactions`), before any write/read assertions or the negative control ever run — matching an arming stall, not an isolation violation.
- It is **not** a real PostgreSQL lock cycle: PG's deadlock detector (`deadlock_timeout`, ~1s) breaks true cycles with an error within seconds. A silent 180.01s wall means participants never *arrived* — a coordination stall, exactly as observed.
- Everything else passing (`executor/dbos` 37s, `probe/classd` 39s) is consistent: the product machinery is fine; only this fixture starves.
- The gob-encode WARNs and `Recovered pending workflows count=1` just before the failure belong to earlier fault-injection spikes in the same package — they share the process, not the tables or pool. Unrelated.

## The fix

Size the pool explicitly in `setupConcIsolation` (rls_concurrent_test.go:70):

```go
cfg, err := pgxpool.ParseConfig(dsn)
if err != nil {
    t.Fatal(err)
}
cfg.MaxConns = int32(len(concTenants)*workersPerTenant + 2) // 8 armed + final counts + negative control
pool, err := pgxpool.NewWithConfig(ctx, cfg)
```

The barrier's concurrency requirement must be provisioned, not inherited from an ambient default that varies by machine.

## Insights

1. **Barrier-synchronized tests consume a resource the barrier implicitly budgets.** The contract's testing discipline (X-07 lesson) says barrier-synced concurrency with separate connections — this test does that right, but a barrier that holds N connections turns the pool into a synchronization point. Any fixture that parks transactions mid-flight must set `MaxConns ≥ concurrent parked transactions + follow-up traffic`, or the pool deadlocks *for* the test.
2. **Fail fast, with observability.** `waitFor` only selects on the barrier channel and the context. If it also selected on `errs` (buffered, already plumbed through) and reported *how many* workers armed, this failure would have surfaced in milliseconds as "4/8 armed — pool exhausted" instead of burning 180s and naming no suspect. Same evidence-discipline the framework demands of workflows applies to the harness itself. The 180s backstop is fine as a backstop; nothing should ever reach it.
3. **Ambient defaults are hidden nondeterminism.** A test whose outcome depends on `runtime.NumCPU()` will flake across runner upgrades, self-hosted runners, and local machines. Pin every resource the test depends on (pool size, timeouts) — the same philosophy as the contract's "no timing-based coordination."
4. **Separate, smaller finding from the log:** the fault-injection spike forces `errors.New("spike forced failure")` (`*errors.errorString`), which DBOS persists message-only — `errors.Is/As` won't match after recovery. Use a registered typed sentinel error so recovered workflows can classify failures; that's the contract's faithful-recovery requirement. The "Recovered pending workflows count=1" line, meanwhile, is the recovery machinery *working* as designed.