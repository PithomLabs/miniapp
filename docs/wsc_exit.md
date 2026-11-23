**WS-C exit submitted. Stopping — WS-A not started, awaiting review.**

## WS-C exit gate self-check (directive §8)

| # | Requirement | Evidence |
|---|---|---|
| 1 | Per-attempt log, no dedup | Serial PK, plain INSERT, legacy-table upgrade; `mockcap.go` |
| 2 | Double-send → 2 rows via real Channel | `TestMockRecordsEveryAttempt` PASS (green) |
| 3 | Fails on snapshot for predicted reason | RED: `SentCount = 1, want 2`, snapshot `893df60f…` / `miniapp_red` |
| 4 | Same test passes remediated | GREEN on worktree / `miniapp_dev`; test bytes identical (`0e022bbb…` both sides) |
| 5 | Affected assertions reviewed + re-run | effects 11/11, probe 12/12, mockcap 4/4; `==1` now genuine, `==0` still valid; no dup-send revealed; none weakened |
| 6 | Tenant attribution | `TestMockTenantAttribution` (incl. cross-tenant) + `Attempts()` rows |
| 7 | False claims corrected | Mock comment, `effects_test.go` phrasing, probe helper comment |
| 8 | Injector contract tests | 8/8 properties green under `-race` (exact-Nth, one-shot, persistent, isolation, propagation identity, unconfigured-nil, nil-safe, concurrent) |
| 9 | §9 inventory reconciled | 19 named ops, each with firing/boundary/propagation test (incl. commit/response pairs proving rollback vs idempotent recovery separately) |
| 10 | Production safety | 4 nil-after-`New` tests; grep audit: 4 field decls + 13 consults, 0 non-test assignments, `fault.New` in 5 test files only, 0 constructor touches, 0 env/manifest/config paths |
| 11 | Digests recorded | Harness/impl/full-tree/source-scope digests for both runs in the register |
| 12 | No unexplained changes | Full `-p 1` suite 21/21 green, `checks.sh` ALL-CHECKS-PASS, race clean; one specified behavior refinement (Status surfaces settlement-audit failure as retryable — X-24a precursor, tested) |

Two judgment calls to surface: (1) the X-07 discriminator lives in its own red/green-compatible file using only snapshot-stable APIs, with row-level assertions in a companion green-only test — identical-source red/green would otherwise be unbuildable; (2) `TestFaultAttemptBegin`'s retry converges via lease-expiry takeover (immediate retry correctly meets contention) — specified lease behavior, not a weakening.

No FaultInjector red run was manufactured, per §6. Contract §9 + §16 log updated under the never-silent rule.
