# CI Failure Root Cause — DBOS Restart Tests (prompt18)

- **Failing commit:** `982aecfb4435031f9ffd68bca90e06f7f2f1dad5` (HEAD at investigation; tree was clean)
- **GitHub Actions run:** `45115104815`
- **Failures:** `TestSleepAcrossRestart` (`executor/dbos`) and `TestProbeRestart` (`probe/classd`) — both `worker never produced .../submitted` after 120s
- **Investigation date:** 2026-11-23. No commits or pushes made.

## 1. Exact failing step

Step 4 of prompt18's 10-step sequence — **worker initializes DBOS** — never happens. Both workers exit successfully in ~0s during harness setup, before any DBOS initialization, workflow registration, or submission. The parent tests then poll a dead worker's marker path for the full 120 seconds.

Concretely: `TestSleepWorker` reaches `newPool(t)`, which calls `t.Skip("TEST_DBOS_DATABASE_URL unset")`; `TestProbeCrashWorker` reaches `newStack(t)` → `testkit.NewDB(t)`, which skips when `TEST_DATABASE_URL` is unset. CI sets neither variable and provides no PostgreSQL, so both workers skip instantly. Nothing DBOS-related fails; DBOS is never reached.

## 2. Evidence

- **CI config:** `.github/workflows/ci.yml` at the failing commit defines no `services:` block and no `TEST_*` env vars (verified by reading).
- **Local red reproduction:** `go test ./executor/dbos/ -run TestSleepAcrossRestart` with both vars unset fails after exactly 120.00s with the identical message (`worker never produced .../submitted`). Same structure in `TestProbeRestart`; CI log shows both.
- **Worker-side proof (not inference):** each worker entrypoint run directly without DB vars skips in 0.00s (`TEST_DBOS_DATABASE_URL unset` / `TEST_DATABASE_URL unset ...`), exit 0, no marker. This is the earliest divergent step, observed — not the missing marker alone.
- **Green control:** with a local PostgreSQL and both vars set, each test passes in ~15s with assertions byte-identical to the failing run.
- **Contrast case:** `TestKillMidSendAdoptsUnknown` (`effects/`) spawns a worker the same way but calls `testkit.NewDB(t)` *before* spawning, so it skips fast without a DB. Only the two DBOS restart parents spawn before any DB touch.

## 3. Why both tests shared the symptom

Identical structure over shared helpers: parent `TempDir` → `testkit.SpawnWorker` → `testkit.WaitForFile(..., 2*time.Minute)` with no DB preflight, and workers whose first DB touch skips when unconfigured. One missing environment input (a database) produces one shared symptom (absent marker after 120s). Business logic in both tests is innocent.

Three harness defects shared the blame for the 120s mystery:

1. **No parent preflight.** Unlike every other DB test in the suite (50+ skip via `NewDB`/equivalent), these parents spawned first.
2. **Worker output discarded.** `SpawnWorker` set no stdout/stderr pipes, so the skip lines above were invisible; `WaitForFile` never inspected worker liveness or exit status.
3. **No reaping on the timeout path.** `WaitForFile` killed without `Wait()`, leaking a zombie until parent exit.

## 4. Durable correction

No timeout change, no selective skips, no weakened assertions, no business-semantics changes. All changes are CI config + test harness (5 modified files, 1 new test file):

- **CI provides a real database** (`.github/workflows/ci.yml`): pinned `postgres:19.0` service (exact tag matches local `POSTGRES_IMAGE`; comment records the sync point), `miniapp_dev` database, both `TEST_*` URLs, health-checked startup. The restart tests now genuinely execute in CI.
- **Serial test packages in CI** (`go test -p 1 ./...`): packages share one database and truncate tables; parallel packages interfere (proven separately during WS-A review). Required for correctness with a shared service DB.
- **Must-run guard step:** re-runs both restart tests verbosely and greps for `--- PASS`, so a future skip-regression fails CI loudly instead of going green silently.
- **Parent preflight** (both restart tests): new `testkit.RequireDB(t, vars...)` skips fast when its DB vars are unset — the same suite-wide convention as `NewDB`, not a selective skip. Execution is guaranteed by CI providing the DB, per above.
- **Observable, reaping harness** (`testkit/proc.go`): `SpawnWorker` returns a `Worker` handle capturing bounded (64KB tail) stdout/stderr, reaping exactly once via a `Wait` goroutine, with `t.Cleanup` kill+reap on every exit path. `WaitForFile` fails fast when the worker exits before the marker, attaching exit status and output tails; on timeout it kills, reaps, and attaches diagnostics. Marker file remains the sole readiness signal. `Kill9` reaps idempotently. All three `SpawnWorker` callers (incl. the effects crash test) updated mechanically.

## 5. Regression coverage

- `testkit/proc_test.go` (new, 7 tests, no DB needed): early nonzero exit fails fast with status + stderr; clean early exit (the CI skip shape) fails fast with the skip reason visible; present marker still succeeds; timeout kills and reaps (process provably gone); `Kill9` reaps; `Kill9` on an exited worker is clean; output buffer bounded to tail.
- Existing suites unchanged in assertions: both restart tests (kill/recovery/identity), effects crash test, full package suites.
- Live negative control: with bogus (unreachable) DB URLs, `TestSleepAcrossRestart` now fails in **0.20s** carrying the worker's `connection refused` diagnostics — previously this shape of failure cost 120s and said only the marker was missing.

## 6. Reproduced red and green results

| Case | Command / env | Result |
|---|---|---|
| Red (pre-fix shape) | no DB vars, original parents | FAIL after 120.00s, `worker never produced .../submitted` (identical to CI) |
| Worker mechanism | worker entrypoints directly, no DB vars | SKIP in 0.00s with unset-var message, exit 0 |
| Green | local PG + both vars, post-fix | both restart tests PASS (~15s each), assertions unchanged |
| No-DB post-fix | vars unset | both SKIP in 0.00s |
| Crash post-fix | bogus DB URLs | FAIL in 0.20s with worker diagnostics attached |
| Repeats | `-count=3` each restart test, with DB | all PASS |
| Full suite | `go test -p 1 ./...` with DB | 21 pkgs ok, 0 fail |
| Full suite | same without DB | 21 pkgs ok (skip-fast), 0 fail |
| Race | CI's `go test -race ./contract/ ./internal/... ./spikes/` | ok |
| Static | `gofmt`, `go vet ./...`, `bash scripts/checks.sh` | clean / clean / ALL-CHECKS-PASS |
| Guard simulation | CI guard-step grep pipeline locally | both `--- PASS` matched |

Environment for local runs: Go 1.27.3 (CI uses 1.26; `go.mod` floor 1.26.0), stock PostgreSQL 16.15 user-space cluster (CI will run pinned 19.0 per the workflow; the exercised surface is version-independent).

## 7. Remaining risks

- **First CI run on the fix is the true proof.** Everything above is local; the workflow YAML is validated but the service wiring (health check, DNS `localhost`, startup ordering) is only proven by a green Actions run showing both tests executing (the guard step enforces the "executing, not skipping" part).
- **Expanded CI execution surface.** With a database present, previously-skipping DB tests (spikes, River/RLS, DBOS suites) will now execute in CI. They pass locally; any environment-sensitive failure will surface on the first run — that is intended (real coverage), but expect possible follow-up.
- **Go version skew** (CI 1.26 vs newer local toolchains) remains as before; unrelated to this failure.
- **No governing-document conflict found:** no contract, ADR, or plan text specifies harness/CI behavior contradicted by this fix, so no doc changes were made.
