# Evidence Register (F19)

Behavioral proof lives here, indexed by gate. Design-consistency
review is never recorded as behavioral evidence. Every entry links:
evidence artifact, test name, source revision, environment, exact
command, result, and date.

## Index format

| Gate | Claim | Artifact | Test | Revision | Environment | Command | Result | Date |
|---|---|---|---|---|---|---|---|---|

## Entries

| G0-Core | MCP protocol round-trip passes | spikes output | `TestMCPRoundTrip` | _pending first commit_ | local, in-process | `go test ./spikes/ -run TestMCPRoundTrip` | PASS 2026-11-22 | 2026-11-22 |
| G0-Core | DBOS atomic commit/rollback on Neon direct | spikes output | `TestDBOSAtomicCommit`, `TestDBOSAtomicRollback` | _pending_ | Neon `openlevel-staging`, branch `phase0/g0-spikes`, direct | `go test ./spikes/ -run 'TestDBOSAtomic(Commit\|Rollback)'` | PASS observed 2026-11-22 16:19 +08:00; re-run required after access restoration | 2026-11-22 |
| G0-Core (supporting, local only) | Full spike suite on local PG19 | spikes output | `TestDBOS*`, `TestRiver*`, `TestRLS*`, `TestMCPRoundTrip` | _pending first commit_ | local Docker postgres:19.0, miniapp_dev | `go test ./spikes/ -v -timeout 4m` with local `TEST_*` URLs | PASS 2026-11-22 20:02 +08:00 (7 pass, 2 expected skips); NOT Neon evidence | 2026-11-22 |
| G0-Core | Neon API unreachable at closure probe | CLI output | n/a (probe, not a test) | n/a | operator workstation | `neon projects list` | BLOCKED 2026-11-22 20:02 +08:00: stored OAuth expired, refresh out of authorization; no remote state changed | 2026-11-22 |
| G0-Core | DBOS atomic commit/rollback/crash/family on Neon direct (separate pools) | spikes output | `TestDBOSAtomicCommit` (100.4s), `TestDBOSAtomicRollback` (27.3s), `TestDBOSFamilyIsolation` (42.5s), `TestDBOSCrashRecovery` (60.4s) | _pending first commit_ | Neon `Pithom Labs`, branch `phase0/g0-spikes` (`br-frosty-pine-bdwmx50b`), direct, PG 18.7 | `go test ./spikes/ -v -timeout 4m` with branch DSNs (suite timeout hit after DBOS+MCP; River/RLS rerun separately) | PASS 2026-11-22 20:10:47–20:14:48 +08:00; `dbos.transaction_completion` present → separate-pool path confirmed; shared-handle variant NOT covered | 2026-11-22 |
| G0-Core | River tx enqueue + RLS tenant isolation on Neon direct | spikes output | `TestRiverTransactionalEnqueue` (11.3s), `TestRLSTenantContext` (10.3s) | _pending first commit_ | Neon `Pithom Labs`, branch `phase0/g0-spikes`, direct | `go test ./spikes/ -run 'TestRiverTransactionalEnqueue\|TestRLSTenantContext' -v -timeout 4m` | PASS 2026-11-22 20:15:16–20:15:39 +08:00; RLS single-threaded only, concurrent leg outstanding | 2026-11-22 |
| G0-Core | DBOS basic tx on pooled endpoint (C5 probe) | spikes output | `TestDBOSAtomicCommit` | _pending first commit_ | Neon pooled endpoint, same branch | `go test ./spikes/ -run TestDBOSAtomicCommit` with pooled DSN | PASS 30.0s 20:16:07–20:16:38 +08:00; proves basic tx only — notification paths unproven, ADR-007 direct-only retained | 2026-11-22 |
| G0-Core | Spike output contains no credentials | grep count over evidence logs | n/a (hygiene check) | n/a | operator workstation | `grep -c -F <password> <logs>` without printing | PASS 2026-11-22: 0 occurrences in both logs | 2026-11-22 |

| G0-Core | DBOS shared-handle durability + detection negative | spikes output | `TestDBOSSharedHandle`, `TestDBOSSharedDetectionNegative` | _pending first commit_ | local Docker PG19 + Neon `Pithom Labs`, branch `phase0/g0-spikes`, direct | `go test ./spikes/ -run 'TestDBOSShared(Handle\|DetectionNegative)' -v` with local, then branch DSNs | PASS local + Neon 2026-11-22 20:42–20:43 +08:00; re-confirmed local 21:04. Detection log present (shared) / absent (separate); commit+rollback markers + step metadata asserted | 2026-11-22 |
| G0-Core | Concurrent-tenant RLS isolation | spikes output | `TestConcurrentTenantIsolation` | _pending first commit_ | local Docker PG19 + Neon `Pithom Labs`, branch `phase0/g0-spikes`, direct | `go test ./spikes/ -run TestConcurrentTenantIsolation -race -v` locally; same without `-race` on Neon | PASS local + Neon 2026-11-22 20:42–20:43 +08:00; re-confirmed local 21:04. 8 workers × 4 tenants, barrier overlap, durable counts, unarmed leg at 0 rows; RLS-off experiment failed at foreign-row assertion as designed (file restored byte-identical) | 2026-11-22 |
| Phase 1 constraint (observation, not behavioral proof) | Shared-pool shutdown ordering | spike debug log + code comment | `TestDBOSSharedHandle` development | _pending first commit_ | local Docker PG19 | n/a (found during prompt7 implementation) | OBSERVED 2026-11-22: closing a pool shared with a running DBOS executor deadlocks on its background workers. Constraint: stop and drain DBOS workers before closing a pool they depend on. Not a framework defect | 2026-11-22 |

No other behavioral entries exist yet. Goals, not results, stay out.
