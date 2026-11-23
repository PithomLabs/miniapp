# Class D Framework Readiness (G1)

14 operational acceptance checks derived from the G1 requirements
(`docs/prompt10.md` §9.1–14; `docs/plan4.2.md` remains authoritative
on any discrepancy). Each row cites the governing plan requirement,
the proving implementation and tests, the exact source state and
environment, and one verdict: PASS, PARTIAL, BLOCKED, NOT PROVEN.

Source state for every row: HEAD `5cf9627` plus the uncommitted
Prompt 13 tree (3 modified paths, 30+ untracked paths — full list
in the P0 baseline entry of `docs/evidence/README.md`). No commit
or push was made during this work. Local environment: Docker
`postgres:19.0`, database `miniapp_dev`, Go 1.27.3, pinned
dependencies per `go.mod` (DBOS v1.9.0, River v0.51.2, MCP SDK
v1.8.0, pgx v5.13.0). Neon environment (where noted): project
Pithom Labs, branch `phase0/g0-spikes` — results kept separate.

## 1. Class A works synchronously — PASS

- Plan: §6.1/§19 Phase 1 (synchronous Class A semantics; inline
  Class A tests).
- Proof: `runtime` invoke/dispatch, `executor/classa` (tenant tx +
  entity advisory locks), `miniapps/crm-read`. Tests:
  `TestInvokeClassAHappyPath`, `TestInvokeIdempotentReplay`,
  `TestSearchRoundTrip`, `TestSameEntitySerializes`,
  `TestDistinctEntitiesParallel`, adapter happy-path cases.
- Env: local PG (PASS); Neon re-run recorded separately below.

## 2. Class D works durably — PASS

- Plan: §6.2/§11/§19 Phase 1 (durable waits, restart/replay).
- Proof: `executor/dbos` (per-family applications, pinned
  versions, idempotent submit, signal/cancel), `probe/classd`
  12-test conformance envelope through the canonical service.
- Env: local PG (PASS); Neon re-run recorded separately below.

## 3. Both use the same canonical invocation service — PASS

- Plan: §6.3 (one canonical invocation service; no parallel
  authorization/policy/idempotency/execution paths).
- Proof: all invocations flow through `runtime.Service.Invoke`;
  REST/MCP/CLI adapters translate shapes only. Contract tests
  (`transport`) assert equivalent outcomes across direct, REST,
  MCP, and CLI for success, refusals, approval, and lifecycle.
- Env: local PG (PASS).

## 4. Authorization occurs before dispatch — PASS

- Plan: §9.3/§14 (authenticate, authorize, then dispatch).
- Proof: `auth.Policy` Authenticate+Authorize precede every
  dispatch; `effects.StrictPolicy` revalidates grant, token, and
  kill switches at every effect claim. Tests: `TestInvokeRejections`
  (6 classes), `TestEffectGrantRevoked`, `TestEffectTokenMismatch`,
  `TestServerDerivedPrincipal`, adapter 401/403/404 mappings.
- Env: local PG (PASS).

## 5. Audit attribution is complete — PASS

- Plan: §9.5/§13 (complete audit trail; agent/authority/tenant
  attribution).
- Proof: mandatory audit on dispatch, decisions, signals,
  cancels, and run settlement (`run-settled` on terminal
  discovery). Tests: `TestInvokeClassAHappyPath` (chain),
  `TestProbeAuditReconstruction` (full gated chain with
  per-row attribution assertions).
- Env: local PG (PASS).

## 6. DBOS workflow recovery is version-safe — PASS

- Plan: ADR-016 + §11.2 (per-family AppName, explicit
  compatibility version, drain); G0-4 topology.
- Proof: per-family applications with pinned
  `<family>/<compat>` versions (SDK affinity verified in
  v1.9.0 source); startup foreign-version reporting; run→family
  routing. Tests: `TestVersionOverlap`, `TestRestartRecovery`,
  `TestTwoFamilyRouting`, `TestSleepAcrossRestart`.
  Cross-release artifact retention stays the G6 live-drill item
  (documented, out of Phase 1 scope by plan).
- Env: local PG (PASS).

## 7. There are no duplicate confirmed effects — PASS

- Plan: R2/§9.5 (no duplicate confirmed business effects; no
  silently lost operations; no blind retries of unknown outcomes).
- Proof: predicated ledger transitions, contention-vs-adopt
  lease rule, takeover path, idempotent submit/key handling,
  op-identity comparison. Tests: 8-way duplicate claims (1
  send), kill-at-every-boundary suites, `TestSubmitIdempotent`,
  `TestIdempotencyKeyConflict`, `TestOperationReuseConflict`,
  probe restart/concurrency (exactly 1 send).
- Env: local PG (PASS).

## 8. Entity identity and serialization are demonstrated where required — PASS

- Plan: §9.1/§15.2 (tenant-scoped `entity_key`; IDs alone do not
  serialize).
- Proof: `contract.EntityKey` derivation contract, registry
  admission strictness, dispatch-time tenant prefixing, Class A
  advisory-lock serialization, probe durable-sequence record
  with DB-arbitrated claims and attach semantics; partitioned
  queues evaluated (ordering-only) on the pinned SDK. Tests:
  entity suite, serialization pair, queue probe,
  `TestProbeConcurrency`, `TestProbeCrossTenant`.
- Env: local PG (PASS).

## 9. Approval-pending recovery works with mock channels — PASS

- Plan: §15.1/§19 Phase 1 (approval-pending recovery with mock
  channels).
- Proof: decision→signal bridge with idempotent Decide and
  crash redelivery, workflow-side table re-read helper, lazy
  expiry observation, denial/expiry run settlement, scope/role
  enforcement. Tests: `TestDenialCancelsRun`,
  `TestDecideCrashRedelivers`, `TestApprovalReread`,
  `TestProbeApprovalFlow/Timeout`, `TestProbeRejectExits`.
- Env: local PG (PASS).

## 10. The OpenCode integration works end to end — PASS

- Plan: §19 Phase 1 (harness end-to-end).
- Proof: MCP stdio adapter served from `cmd/miniapp`; agent
  session `ses_dfba059baffeS49dPms5qed1Xw` executed contact
  search, honoring a wrong-key refusal without retry, then a
  full Class D submit→status→signal→succeeded lifecycle with
  agent-reported IDs. Redacted transcript:
  `docs/evidence/opencode-phase1-demo-2026-11-23.jsonl`.
- Env: operator workstation, OpenCode 1.19.6, isolated HOME,
  local PG (PASS).

## 11. The determinism analyzer works against its positive and negative fixtures — PASS

- Plan: §19 Phase 1 (pinned-SDK-aware analyzer, fixtures,
  replay tests).
- Proof: `analyzer/determinism` implements all seven
  `determinism-rules.md` families; `TestAnalyzer` (detbad 13
  violations flagged, detgood clean); dogfood clean on all repo
  workflow code; replay equivalence via
  `TestProbeReplayDeterminism`.
- Env: static analysis, no DB (PASS).

## 12. CI enforces the required Class D rules — PARTIAL

- Plan: §19 Phase 1 (strict CI for Class D).
- Proof: analyzer strict gate 6 and preflight gate 7 wired
  into `scripts/checks.sh`, which CI runs; `ALL-CHECKS-PASS`
  observed locally. Limitation: CI itself runs only on push,
  and no push is authorized — remote enforcement is configured
  but not yet observed.
- Env: local checks PASS; remote CI NOT PROVEN.

## 13. Real provider writes remain disabled — PASS

- Plan: K24/§9.5 (dry-run only; mock channels in Phase 1).
- Proof: effects channels are mock-only (`mockcap`); checks.sh
  gate 3 scans for Enterprise/provider imports; no production
  credentials exist in tree, config, or logs.
- Env: static + all runs (PASS).

## 14. All required evidence is recorded and independently verifiable — PASS

- Plan: §20/§22 evidence discipline.
- Proof: one register entry per run in
  `docs/evidence/README.md` (artifact, test, revision,
  environment, exact command, result, date, failure
  dispositions); this matrix; the capability matrix with
  per-finding dispositions. Local and Neon results kept
  separate throughout.
- Env: n/a (record-keeping; complete).

## Neon validation (§20.9)

Branch `phase0/g0-spikes`, full framework suite (16 packages):
13 PASS outright (contract, auth, registry, approvals 85s,
audit, classa, dbosexec 531s, crm-read, testkit, mockcap,
transport 209s, preflight, analyzer). Three initial failures,
all test-harness issues with zero product defects, each fixed
and re-verified green on the branch: concurrency exact-overlap
assumption under latency (rewritten gate-held, 23s), Neon
password policy on the bypass-test role plus revoke-based
cleanup (28s, ran — not skipped), eligibility flip racing row
creation (row-wait, 65s). Spikes not re-run (G0 Neon evidence
stands; current local pass recorded). Production branch and
project state untouched throughout.

## Owner decisions still required (§22)

1. Pilot fixture selection (Path P vs Path W for any
   `lead-followup` work): OPEN — blocks increment 6 only.
2. Named human reviewer for workload decisions: OPEN.
3. Synthetic vs production values for any pilot: OPEN;
   conformance work used synthetic values throughout.
