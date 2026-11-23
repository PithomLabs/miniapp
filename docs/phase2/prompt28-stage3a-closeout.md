# Prompt28 — Stage 3A closeout

Remaining Stage 3A work (authorization matrix, concurrency +
crash matrices, C6 fixture, revocation tooling, Settle crash
safety, bypass assurance) with Gate 2 evidence. One item is
blocked by owner decision (Settle operator wiring — decision
memo filed, boundary not weakened); everything else is
complete and green.

## 1. Baseline and final HEAD/worktree status

- HEAD throughout: `a09b7ff` ("prompt26"). No commits made.
- Starting worktree: 8 Prompt27 paths + `docs/prompt28.md`,
  plus one new outside append to `docs/faq.md` (+117/−1 Q&A
  on app categories; worktree blob `024e26d4…`, mtime
  2026-11-23 23:56:34 +0800) — preserved byte-exact,
  recorded as out of scope, never reset or attributed.
- Source digest at freeze reproduced the Prompt27 baseline
  (`664558be…`) exactly: zero executable drift at start.
- Final worktree: Prompt28 adds 21 new paths and edits 15
  (see §13 for the inventory); faq append still preserved.
- Outside activity DURING this session (not mine, preserved
  as-is, out of scope): new `docs/roadmap_classd.md`,
  `docs/roadmap_observability.md`,
  `docs/roadmap_security.md`, `docs/observability.md`, and
  deletion of tracked `docs/roadmap2.md` (existed at HEAD,
  absent from worktree; looks like an author-side rename —
  not restored, to avoid clobbering intent). The manifest
  and digests below postdate these arrivals.

## 2. Source and tree digests

- Source digest (stable): `2d98ee5f9950530ed494764f993af8245b1e157d419a6c9c976af09a7192bea5`
  (`bash scripts/digest.sh source .`).
- Tree digest: `64278ca01d9c046744f225428f4c29332523641eed953b0ff1fde5caa0669219`
  (`bash scripts/digest.sh tree .`), computed over every
  file except this report (reproducible by moving it aside).
- Manifest: `docs/evidence/digests/p28-closeout-2026-11-23.manifest.txt`
  (complete inventory, self excluded). Historical
  manifests untouched.

## 3. PostgreSQL 19 immutable image and server identity

- Required: `postgres@sha256:bb6e70ad…` (full:
  `postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`),
  re-resolved via `docker inspect` before the evidence
  snapshot (RepoDigest == container image, exact match).
- Server: `PostgreSQL 19.0 (Debian 19.0-1.pgdg13+1)`.
- Go `go1.27.3 linux/amd64`. Disposable container + fresh
  `p28test` database for all runs; long-lived dev DB never
  connected to; container removed after.
- Pinned digest now in Compose, CI, Taskfile default, and
  `postgres-image.sh` fallback (env overrides preserved);
  zero floating refs in functional config (YAML-parsed,
  script syntax-checked). CI execution itself was NOT run
  here — pinning only.

## 4. Authorization matrix results

12 new legs, all green-first (new coverage, no defect
found — characterization, never red/green):
identity/tenant/binding/tool legs, succeeded+failed
terminal sweep, live waiting states, pending-approval wait
(no audit, no state change), 4 kill-switch scopes +
tool-declared switch, typed grant allow/deny, grant-swap
race hammer (-race clean, exact outcome accounting),
capability-reuse conflict, Intent trust-field guard.
Every refusal asserts typed classification, untouched
state, zero sends, and exact audits. Approval dimension
cites M1–M7/P27 (unchanged); spoofing cites M5 + the new
structural guard + P26-001.

## 5. Concurrency/race matrix results

- B1/B3 cite POC-1/POC-2; B6 cites the WS-A swap hammer.
- New, all forced-order (gates, row locks, blocking-pid
  chains, pg_stat_activity needles — never sleeps):
  same-effect full evidence (B2/B9), lease-turnover
  quarantine vs live authorize (B4), restore-vs-dispatch
  both orders (B5), sequence fail-closed + zero-producer
  proof (B7; M5 racing stays accepted residual),
  Settle-vs-Settle FIFO + settle hammer (B8).
  All green; every test asserts winner/loser/state/audit/
  provider-count/retry-class/epochs.

## 6. Six-boundary crash matrix results (prompt22 §10.3)

Genuine SIGKILL at lock-wait/row-state positions (2 new
worker entrypoints); every boundary records durable state,
provider accounting, allowed retry, epoch retention, and
evidence: B1a pre-claim queue, B1b mid-auth-tx queue, B2
post-commit pre-send, B3 mid-send (extended with epoch
retention), B4/B5 shared stall-kill test (externally
identical boundaries — explicitly recorded, plus
operator-settle-from-evidence completion), B6a takeover
queue, B6b quarantine-adopt queue. Ambiguous outcomes are
quarantined, never blindly resent, never fabricated. All
green.

## 7. C6 integration result and stable BusinessOpID semantics

`TestP28C6ClassARecovery`: real handler + real dispatch
proves all 11 C6 claims — contention reachable, full
rollback (business write absent, then present on success),
no false terminal (stays running), no held locks
(pg_locks zero + RLock free), interrupted/new-key guidance,
stable op IDs across attempts, adopted holder UUID epoch,
lease-untouched bounded recovery (0.08s, no sleeps),
adopt-without-resend. C6 mechanism was already implemented;
its integration is now execution-proven. Op-ID guidance in
contract §7/M3 + new `docs/development/effect-authoring.md`.

## 8. Grant hash, health, and revocation tooling result

New: `Service.GrantConfigHash()` (RLock-pinned),
unauthenticated `GET /healthz` (`{status, grant_config_hash}`,
design §6), startup hash log, `miniapp auth
verify-revocation` (explicit inventory, per-replica
ok/stale/unreachable verdicts, MISMATCHED/STALE/INCOMPLETE
summaries, exit 0 iff all match, never success on empty
inventory), ops revocation section, §H-7 E2E, hash
order-invariance test, immediate-revocation grep audit
(clean). Specified semantics implemented as written;
reporting-label assignment documented in code. All green.

## 9. Settle crash matrix and authenticated operator result

Crash half complete: genuine-kill pre-commit test,
fault-modeled lost-response test (kill adds no observable
distinction past commit — stated), backfill-idempotency
test; scenario 4 cites B8. The backfill audit race was
PROVEN (16 duplicates/640 races) and fixed with a
migration-free row-lock serialization — genuine red/green
on identical bytes (`1d48f257…`), green 3×.
Operator wiring: STOPPED per §7 + guardrail 2 — no
suitable trusted authority exists (OSI-2 role vocabulary
open, `tenant-ops` synthetic, keys file demo scaffolding,
no control DSN/switch/X-15). Decision memo:
`docs/phase2/prompt28-settle-operator-decision.md` with
three minimal options. `effects.Settle` correctly remains
test-only API.

## 10. Provider bypass inventory and test results

Inventory (grep + AST-pinned): exactly one production
Send site (`sendAndRecord`), two Channel impls (mockcap;
faultwrap test-only, zero non-test importers), one
production Perform caller (probe workflow step),
dbos.Send allowlisted as SDK signaling. Per-path tests:
reconcile-revoked/terminal, UNKNOWN-never-resends,
unknown-capability fail-closed (characterized wart:
invalid-transition error, fail-closed, taxonomy follows
P24-008), probe workflow ledger lineage (CONFIRMED +
UUID epoch + invocation binding + receipt). P24-008 and
P24-004 triggers re-verified; no real provider. All green.

## 11. C1–C7 true traceability table

| Constraint (P21 definition) | Status | Evidence |
|---|---|---|
| C1 — `authorizeAttempt` takes no advisory lock | Holds | No new advisory SQL anywhere in Prompt28 (grep + INFO-2 guard green) |
| C2 — Authorization follows the global lock order | Holds | Backfill fix takes lock 5 only (same as settle path); no new lock levels; healthz/CLI take no PG locks |
| C3 — Authorization uses its own transaction, never the attempt's `pgx.Tx` | Holds | B1b kill test proves claim/auth tx separation (PENDING survives auth rollback) |
| C4 — No new advisory-lock acquirers | Holds | Exhaustive grep; INFO-2 guard green; Class A claim/entity sites unchanged |
| C5 — Explicit pending semantics | Holds | B5 proves the pending window never strands/sends; P21-002 flip untouched |
| C6 — Class A `Perform` recovery: bounded, stable op IDs, no held tx | Holds (integration now proven) | WS-C fixture, all 11 claims green |
| C7 — O-01 closes at the effect boundary | Holds | Refusal-before-send on every new path; zero-send assertions throughout |

## 12. Every remaining residual and activation trigger

- WS-E operator wiring: BLOCKED on owner authority
  decision (memo filed). Nothing else in Prompt28
  depends on it.
- M5 supersede-racing (B7): accepted residual per M5
  deferral; trigger unchanged (verifier + propagation
  + X-08 + admission gate).
- Unknown-capability taxonomy noise: fail-closed,
  characterized; proper refusal follows P24-008.
- B4/B5 shared test: externally identical boundaries,
  one deterministic test, explicitly recorded.
- Startup hash log line: source-proven (main package
  untestable by construction).
- Empty-mode coherence, mint-failure branch, AST
  aliasing, X-12, ledger boundary, 2-min lease:
  carried unchanged from Gate 1.
- Class B/C adapter conformance: explicitly out of
  scope (§12) — a later workstream, not a residual
  of this one.

## 13. Exact commands and results

All on the disposable pinned PG19 (`TEST_*` URLs,
`GOCACHE=/tmp/p28_gocache`, escalated sandbox for
Docker/network); durations observed 2026-11-23:

- New tests per workstream (`-run` focused, `-count=1`,
  `-race` for hammers): all PASS (times in §§4–10).
- `gofmt -l .` clean; `go vet ./...` clean.
- `go test -p 1 ./... -count=1`: 22 ok / 0 fail
  (one infra expectation updated tag→digest mid-run
  per §9, same anti-drift intent; re-run clean).
- `go test -race -p 1 ./runtime/ ./approvals/
  ./effects/`: all ok. `bash scripts/checks.sh`:
  ALL-CHECKS-PASS. DBOS (37s) + Class D (40s) suites
  included in the full run on required URLs.
- New paths (21): 10 effects p28 test files, runtime
  p28 restore/C6 tests, transport/cli revoke.go +
  revoke_test.go, transport healthz test, probe ledger
  test, effect-authoring doc, settle decision memo,
  docs/evidence/p28/, p28 manifest, this report.
  Modified (15): compose/CI/Taskfile/script pins, auth
  hash test (appended), cmd usage/auth group,
  classd-operations (revocation section), contract
  (C6 rule), concurrency test (worker anchor + B3
  epochs), effects.go (backfill lock), infra pin
  expectation, runtime GrantConfigHash, REST healthz
  route, evidence register, Prompt25 report §11 only.
  (Prompt27's 8 paths + prompt28.md + outside
  roadmap/faq activity excluded from these counts.)

## 14. Evidence-register references

`docs/evidence/README.md`: Prompt 28 WS-A, WS-B races,
WS-B crash, WS-C, WS-D, WS-E crash, WS-E wiring, WS-F,
parity, closeout rows. Reviewer package:
`docs/evidence/p28/README.md` (inventory + backfill
red/green record + reproduction).

## 15. Explicit Gate 2 readiness verdict

READY FOR GATE 2 WITH ONE SCOPED EXCEPTION. All
Prompt28 implementation is complete, green, and
evidenced except authenticated Settle operator wiring,
which is correctly STOPPED awaiting the owner authority
decision (memo filed; boundary not weakened). Gate 2
should review everything, note the wiring as
owner-blocked (not implementation-deferred), and must
not claim end-to-end execution conformance across all
four classes (Class B/C adapters remain a later
workstream per §12). CI was pinned, not executed here.
