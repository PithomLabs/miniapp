# Phase 1 Class D Capability Matrix (F-01–F-14)

Framework-readiness findings consolidated from the three prompt11
reviews, each verified against the working tree at prompt13 baseline
(`5cf9627` + increments 4–5 uncommitted). A finding is listed as a
defect only where the cited code was read; plan citations were
re-checked verbatim. Dispositions: PROVEN, PARTIALLY PROVEN,
NOT PROVEN, DEFECT CONFIRMED.

Conventions: R1 = `docs/adv_review_prompt11.md`, R2 =
`docs/adv_review2_prompt11.md`, R3 =
`docs/adv_review3_prompt11.md`, RC =
`docs/review2.md` (consolidation). Workstream tags (WS-A…J) refer to
`docs/prompt13.md` §§7–16.

## F-01 — Submission atomicity, invocation/run binding, orphan prevention

- Original finding: R1 H4; RC F-01. DBOS submit succeeds, then the
  `run_id` persistence fails; retry returns a handle with an empty
  run ID while a live workflow runs orphaned.
- Plan requirement: §19 Phase 1 "stable run handle for Class D";
  §6.3 canonical invocation; prompt10 §9.2–9.3.
- Source locations: `runtime/service.go:237-244` (dispatch),
  `:186-195` (approval-path submit),
  `runtime/service.go:155-159` (replay returns `rec.RunID` verbatim).
- Root cause: submit-then-record without intent persistence or
  compensation; `AsyncExecutor` has no idempotent-submit or
  lookup-by-invocation operation.
- Category: framework defect.
- Required correction: durable submission intent recorded before
  submit (or cancel-on-record-failure with audit); retry must yield
  a valid live handle or a settled terminal state.
- Regression test: fault-injected store failure between submit and
  record; same-key retry; concurrent duplicates; orphan audit.
- Evidence environment: unit fault injection + local PostgreSQL;
  Neon where available.
- Disposition: DEFECT CONFIRMED.
- WS-B correction: runtime-minted run IDs persisted before submit
  (`submitRecorded`), submit errors leave the record live, and
  duplicate polls re-submit idempotently (`confirmSubmit`); both
  executors honor RunID idempotency. `TestSubmitFailureRetryAttaches`
  (1 run, valid handle), `TestConfirmSubmitStableHandle`,
  `TestSubmitIdempotent` (1 marker row) PASS on local PG.
  Disposition now PROVEN (local; Neon re-run in P7).

## F-02 — Approval decision, signal delivery, cancellation, lifecycle convergence

- Original finding: R1 H2/H3/M6; R2 LF-08; R3 H3; RC F-02.
  (a) Denial/expiry marks the invocation failed without cancelling
  a live Class D run. (b) No decision→workflow signal bridge exists.
  (c) Runtime never sets approval expiry. (d) Approved Class D polls
  executor status without verifying resumption.
- Plan requirement: §15.1 (approval table authoritative; Class D
  waits on durable message with timeout; recheck before action);
  §14.4 (out-of-band approval); §9.4 (poll/signal/cancel).
- Source locations: `runtime/service.go:208-228` (pollApproval);
  `runtime/lifecycle.go:93-113` (DecideApproval: decision + audit,
  no signal); `runtime/service.go:171-175` (no Expiry set).
- Root cause: approval and workflow lifecycles were built as
  independent state machines with no coupling protocol.
- Category: framework defect.
- Required correction: decision→signal bridge with crash-safe
  redelivery/reconciliation; terminal non-approval cancels live
  runs; explicit-policy expiry + settlement path; poll reconciles
  invocation/run status.
- Regression test: decision-commit→crash→restart→resume; duplicate
  signals; reject/expire/cancel races; denial cancels run.
- Evidence environment: local PostgreSQL + controlled restart.
- Disposition: DEFECT CONFIRMED.
- WS-D progress: durable-cancel-first ordering, `CancelPending`
  approval settlement, executor redelivery (Cancel retry + Status
  piggyback), `CancelResult.InFlight` reporting via
  `LiveEffects`, invocation-gated effect claims
  (`EffectCanceller` + locked liveness check), settled reads
  exempt from gating. Tests (`TestCancelBeforeClaim`,
  `TestCancelPendingApproval`, `TestCancelInFlightReported`,
  `TestCancelDuplicateSubmission`,
  `TestCancelExecutorDownRedelivers`, `TestCancelPending`) PASS on
  local PG.
- WS-E correction: denial/expiry/cancelled approvals stop the run
  BEFORE the invocation fails (`settleDenied`); decisions signal
  the waiting run on `approval.decided` with idempotent Decide for
  crashed deciders; `ReadApprovalDecision` gives workflows the
  table re-read; manifest expiry validated + recorded, observed
  lazily at poll (no invented sweeper). No poll-time renotify by
  decision: decide-retry + durable mailbox + workflow re-read
  already cover every window; renotify would only add duplicate
  load. Tests (`TestDenialCancelsRun`, `TestExpiryCancelsRun`,
  `TestManifestExpiryRecorded`, `TestDecideSignalsRun`,
  `TestDecideIdempotentAndConflicting`, `TestDecideCrashRedelivers`,
  `TestApprovalReread`) PASS on local PG. The end-to-end
  wait→restart→timeout→reread→proceed path is demonstrated by the
  P4 probe. Disposition now PROVEN (local; Neon re-run in P7).

## F-03 — Effect-ledger concurrency, crash boundaries, UNKNOWN quarantine

- Original finding: R1 B3/H5; R2 LF-02; R3 B1/H5; RC F-03.
  (a) claim→load→transition run in separate transactions with an
  unpredicated UPDATE: concurrent attempts can clobber terminal
  states. (b) Code comments promise "exactly once", contradicting
  the at-least-once substrate and §9.5. (c) No UNKNOWN drain exists
  (correctly Phase 4; quarantine listing/operator settle unproven).
- Plan requirement: §9.5 (state machine + "no duplicate confirmed
  business effects, no silently lost operations, no blind retries");
  §15 (UNKNOWN quarantine); §18 crash-boundary tests.
- Source locations: `effects/effects.go:75-103` (Perform),
  `:108-129` (claim), `:144-155` (adopt), `:159-176` (transition,
  unpredicated UPDATE); `:71`, `effects/doc.go:3` (overclaim).
- Root cause: ledger written for sequential use; concurrency
  control and honest guarantee language deferred.
- Category: framework defect (a, b); unproven capability (c).
- Required correction: row-lock or state-predicated transitions;
  R2-accurate guarantee language; quarantine exposure + operator
  settle path; per-boundary crash tests with a poisoned mock.
- Regression test: barrier-synced duplicate claims; kill at every
  intent/send/outcome boundary; ambiguous timeout; UNKNOWN settle.
- Evidence environment: local PostgreSQL + process kill; `-race`.
- Disposition: DEFECT CONFIRMED.
- WS-C correction: expected-state-predicated transitions with
  affected-row checks; fresh-live duplicates answer
  ErrAttemptInProgress (never adopt); stale rows adopt (ATTEMPTING)
  or take over (PENDING) under age predicates; tenant-scoped op
  IDs enforced; R2-honest guarantee language; ListUnknown +
  operator Settle. Tests (`TestConcurrentDuplicateClaims` 8-way,
  `TestLiveRaceContention`, `TestStalePendingTakeoverSendsOnce`,
  `TestKillMidSendAdoptsUnknown`, `TestSettleAndListUnknown`,
  `TestCrossTenantOpIDs`, `TestOpIDScopeEnforced`) PASS with
  `-race` on local PG: exactly 1 provider send, no clobber, no
  blind retry. Disposition now PROVEN (local; Neon re-run in P7).

## F-04 — Invocation, workflow, entity, sequence, effect identity separation

- Original finding: R1 B2/M4; R2 LF-03/LF-04; R3 B2/B3; RC F-04.
  Invocation and run identities are distinct in the record (good),
  but entity identity has no tenant-scoped contract: `entity_key`
  is free text, unvalidated for scope, unenforced at admission.
- Plan requirement: §15.2 ("Workflow IDs alone do not serialize
  access to an entity"); §9.1 (consequential writes declare
  `entity_key` or reviewed `no_entity`); §12 tenant scope.
- Source locations: `contract/manifest.go:135,191`
  (presence-only check); no admission check in `registry/`;
  `executor/dbos/executor.go:104` (random run IDs).
- Root cause: identity framework built bottom-up (invocation/run
  first); entity contract declared but not enforced.
- Category: framework defect (missing enforcement).
- Required correction: tenant-scoped `entity_key` format validated
  in contract + enforced at registry admission; documented
  five-identity model; workflow ID references durable sequence,
  never equals entity.
- Regression test: unscoped/missing key rejected; same entity ID
  across tenants fully independent; attach/audit semantics.
- Evidence environment: unit + local PostgreSQL.
- Disposition: DEFECT CONFIRMED.
- WS-A correction: `contract.EntityKey` (input-field-path shape) +
  admission strictness for effectful tools + dispatch-time
  tenant-prefix derivation + Class A advisory-lock auto-serialization
  implemented and tested.
- P4 demonstration: the probe's durable `probe_sequences` record
  (partial unique on the active generation, DB-arbitrated claims,
  attach-on-conflict with observation rows, wake_at-bounded
  supersede) proves the identity split end to end:
  `TestProbeConcurrency` (one winner + one attached),
  `TestProbeCrossTenant` (independent sequences).
  Disposition now PROVEN (local; Neon re-run in P7).

## F-05 — Per-family AppName, version compatibility, recovery

- Original finding: R1 H1; R2 LF-06; R3 H4; RC F-05. One AppName
  serves the whole registry; no application-version management;
  `WorkflowDef.Version` is admission-checked but never read.
- Plan requirement: ADR-016 + §11.2 (stable distinct AppName per
  Class D family; explicit compatibility version; retain old
  artifact until drain); G0-4 topology; §19 Phase 1 version
  isolation/drain test.
- Source locations: `executor/dbos/executor.go:41-58` (single
  AppName, no version config); `registry/registry.go:133`
  (only Version read site); SDK `computeApplicationVersion`
  (binary hash default — verify override in v1.9.0 source).
- Root cause: single-family Phase 1 shortcut; version policy
  deferred to later work.
- Category: framework defect (missing policy + enforcement).
- Required correction: per-family AppName + managed version tied
  to workflow compatibility; drain/retain policy; operator-visible
  unknown-version handling; live N→N+1 pending-run transition test.
- Regression test: old run pending across deploy; unknown version
  visible/recoverable; artifact retention.
- Evidence environment: local PostgreSQL + two builds/processes.
- Disposition: DEFECT CONFIRMED.
- WS-F correction (verified against v1.9.0 source: recovery and
  dequeue both filter on application_version, so affinity is
  native): one DBOS application per miniapp family
  (`miniapp-<id>`), `ApplicationVersion` pinned to the declared
  workflow-compatibility version (admission enforces one version
  per family), startup warning naming foreign pending versions
  with counts, run→family routing with fan-out fallback.
  Tests (`TestVersionOverlap`, `TestRestartRecovery`,
  `TestTwoFamilyRouting`, registry `family-version-agreement`)
  PASS on local PG: v2 reports v1's pending run without touching
  it, v1 drains exactly once, restarts resume, two families route
  independently. Cross-release artifact retention stays a G6
  deployment drill (documented in `executor/dbos/doc.go`).
  Disposition now PROVEN (local; Neon re-run in P7).

## F-06 — Effect-time authority, tenant, policy, blast-radius enforcement

- Original finding: R2 LF-07; RC F-06 (verification-required, not
  source-proven by R2). Revalidation at effect time, central
  limits, kill switch.
- Plan requirement: §9.3 (recheck before provider action); §15.1
  (expiry/material/authority recheck); §9.1/§14.2 (blast-radius
  limits + kill switches enforced centrally).
- Source locations: `contract/manifest.go:84-96`
  (`BlastRadiusLimits`/`KillSwitch` declared, zero non-contract
  references); no enforcement in `runtime/` or `effects/`;
  pre-dispatch `Authorize` present (`runtime/service.go:112-118`).
- Root cause: policy model declared in contract; enforcement
  wiring never built.
- Category: unproven capability (mechanism absent, not broken).
- Required correction: central revalidation gate on the
  invocation→effect-claim path; revocation/kill-switch during
  waits blocks new effects; attributable refusals.
- Regression test: revoke authority / trip kill switch mid-wait;
  attempt effect; assert refusal + audit.
- Evidence environment: local PostgreSQL.
- Disposition: NOT PROVEN.
- WS-G correction: `StrictPolicy` resolves tool/trust from the
  invocation row (never the Intent), re-checks the live grant,
  verifies the per-invocation effect token, and enforces
  global/tenant/miniapp/capability/tool kill switches from
  central state; trust mode + token persisted at claim
  (migration 000004); op-identity comparison on ledger conflicts.
  Tests (`TestEffectGrantRevoked`, `TestKillSwitchBlocks`,
  `TestEffectTokenMismatch`, `TestServerDerivedPrincipal`,
  `TestOperationReuseConflict`) PASS on local PG.
  Deliberately deferred (OSI-4): rate/volume counters, amount
  parsing, recipient/domain payload inspection, circuit breaker —
  each needs owner/product decisions on formats, windows, and
  sources that no approved source supplies. Disposition now
  PARTIALLY PROVEN.

## F-07 — Idempotency-key reuse with different input

- Original finding: R1 M3; RC F-07. Claim-conflict path never
  compares input; replay serves the first record for any input.
- Plan requirement: §15 (idempotency); prompt10 §9 (duplicate
  submit proving same-request replay).
- Source locations: `runtime/store.go:38-58` (ON CONFLICT
  DO NOTHING, no input check); `runtime/service.go:142-163`.
- Root cause: input stored but never compared on conflict.
- Category: framework defect.
- Required correction: canonical-input-hash comparison on
  claim conflict; mismatch → terminal `idempotency-key-reuse`.
- Regression test: same key/same input replays; same
  key/different input errors explicitly.
- Evidence environment: local PostgreSQL.
- Disposition: DEFECT CONFIRMED.
- WS-B correction: canonical-input comparison on claim conflict
  (`KeyConflict` → terminal `idempotency-key-reuse`); `TestIdempotencyKeyConflict`
  PASS (different input errors, whitespace variant replays). The
  same canonicalizer now backs `IntentHash` (R1 L3 fixed;
  whitespace/key-order equivalence tested). Disposition now PROVEN
  (local; Neon re-run in P7).

## F-08 — Tenant-scoped identities, SQL predicates, pooled isolation

- Original finding: R1 H5/M4/L1; RC F-08. RLS + `WithTenant`
  pervasive, but `invocations.update`, `effects` load/transition,
  and approvals lookups rely on RLS alone; effect PK is global.
- Plan requirement: §12 (transaction-local tenant context; RLS);
  §14 (tenant isolation incl. privileged roles); K25.
- Source locations: `runtime/store.go:103-115` (update: no tenant
  predicate — contrast `getByRun` comment on RLS bypass);
  `effects/effects.go:132-142,166-171`;
  `migrations/000001_framework_init.up.sql:62` (global PK).
- Root cause: defense-in-depth applied inconsistently; tenant
  scoping of effect identity deferred to WS-A.
- Category: framework defect (hardening gap).
- Required correction: explicit tenant predicates on all
  mutation paths; tenant-scoped effect identity (with WS-A);
  bypass-role tests.
- Regression test: cross-tenant update refusal under privileged
  role; identical IDs across tenants.
- Evidence environment: local PostgreSQL (NOBYPASSRLS + bypass).
- Disposition: PARTIALLY PROVEN.
- WS-A correction: tenant predicates added to
  `invocations.update` and approvals lookups
  (`TestUpdateRejectsCrossTenant` PASS). Effects load/transition
  predicates land with WS-C state-predicate hardening.
- WS-C correction: effects load/transition carry explicit tenant
  predicates; op IDs tenant-prefixed by enforcement;
  `TestUpdateRejectsCrossTenantBypassRole` proves the predicate
  (not RLS) rejects under BYPASSRLS. Disposition now PROVEN
  (local; Neon re-run in P7).

## F-09 — Approval authority, scope, expiry, duplicates, self-approval

- Original finding: R1 H3/M2; R2 LF-08; R3 H3; RC F-09.
  Self-approval + tenant + hash + lazy-expiry checks enforced
  (good); `ApproverScope` stored but never enforced; runtime sets
  no expiry; no invocation-status gate before Decide.
- Plan requirement: §14.4/§15.1 (approval policy, expiry,
  out-of-band channel in Phase 4); separation of duties.
- Source locations: `approvals/approvals.go:82-132` (no scope
  check); `runtime/lifecycle.go:93-113` (no invocation check);
  `runtime/service.go:171-175` (no Expiry).
- Root cause: scope/role model deferred; expiry left to callers
  that never set it.
- Category: framework defect (partial enforcement).
- Required correction: scope/role enforcement at Decide (owner
  defines roles — Open Spec Item if unconfirmed); refuse decisions
  on non-waiting invocations; duplicate-decision consistency rule.
- Regression test: unauthorized role refused; decide-after-cancel
  refused; double-decide consistent.
- Evidence environment: local PostgreSQL.
- Disposition: DEFECT CONFIRMED.
- WS-E correction: `Binding.Roles` + scope matching at Decide;
  invocation-status gate (cancelled/terminal refused); identical
  re-decide idempotent, conflicting re-decide rejected;
  self-approval retained. Tests (`TestDecideScopeAndIdempotency`,
  `TestScopeEnforced`, `TestDecideIdempotentAndConflicting`,
  `TestDecideAfterCancelRefused`) PASS on local PG. Owner-defined
  role vocabulary stays an Open Spec Item (mechanism is generic).
  Disposition now PROVEN (local; Neon re-run in P7).

## F-10 — Durable sleep, signal recovery, measured worker/edition behavior

- Original finding: R1 M5; R2 §7.7/DBOS.md:622 note; RC F-10. No
  durable sleep used or proven; open-source sleep residency and
  edition boundaries unverified on the pinned SDK.
- Plan requirement: §19 Phase 1 (durable waits; harness e2e);
  §7.7 retry/retention/alerting; prompt10 §9.2.
- Source locations: no `Sleep(` workflow usage in non-test code
  (verified by grep); crash spike blocks in a step, not a sleep.
- Root cause: capability never exercised.
- Category: unproven capability.
- Required correction: sleep-across-restart probe on pinned
  v1.9.0; signal timing/duplication matrix; measured sleeper
  envelope (memory/connections/checkpoints); edition check.
- Regression test: kill mid-sleep → wake on time; pre/during/post
  signals; duplicate signals; cancel during wait.
- Evidence environment: local PostgreSQL + process kill.
- Disposition: NOT PROVEN.
- WS-H correction: `TestSleepAcrossRestart` (SIGKILL mid-15s-sleep,
  relaunch, re-submit attaches, wakes and completes exactly once),
  `TestSignalMatrix` (pre-wait buffered, duplicates single-apply,
  post-completion accepted harmlessly with no state change),
  `TestSleepSuspendRequiresEnterprise` (OSS boundary proven via
  the gating error), `TestSleeperEnvelope` (20 sleepers in 2.1s;
  see evidence entry for measurements). All PASS on local PG.
  Disposition now PROVEN (local; Neon re-run in P7).

## F-11 — Fresh vs stale lifecycle status, executor-unavailable behavior

- Original finding: R1 L2; RC F-11. `Status` swallows executor
  errors and serves the record as if fresh.
- Plan requirement: §9.4 lifecycle operations; operability
  (inspectable, honest status).
- Source locations: `runtime/lifecycle.go:28-34` (`err == nil`
  guard discards failures).
- Root cause: optimistic refresh without staleness channel.
- Category: framework defect.
- Required correction: freshness levels (fresh / persisted while
  unavailable / stale / terminal); executor errors surfaced as
  retryable or stale-marked, never silent.
- Regression test: executor-down Status returns stale-marked
  record or retryable error (failing fake).
- Evidence environment: unit + local PostgreSQL.
- Disposition: DEFECT CONFIRMED.
- WS-H correction: `InvokeResult.Stale` marks persisted records
  served during executor outage; terminal records stay
  authoritative. `TestStatusStaleOnExecutorDown` PASS on local
  PG. Disposition now PROVEN (local; Neon re-run in P7).

## F-12 — Entity serialization, partitioned queues, concurrency semantics

- Original finding: R1 B2; R2 LF-03/LF-15; R3 B2 (partially
  rejected: prompt13 excludes workflow-ID-as-identity and
  queue-alone serialization); RC F-12. No serialization mechanism
  exists at all; queue behavior on the pinned SDK unverified.
- Plan requirement: §15.2 (Class D may use partitioned queue,
  concurrency one, keyed by entity; IDs alone do not serialize);
  §18 keyed-serialization tests.
- Source locations: no queue/lock serialization code anywhere
  (verified by grep); `executor/dbos/executor.go` (no queue API).
- Root cause: deferred to WS-A by design.
- Category: unproven capability.
- Required correction: SDK-source queue evaluation (slot
  occupancy, second-submit, fairness, restart/overlap) + durable
  business-level uniqueness independent of ordering; tenant in
  every key.
- Regression test: same-entity serializes; different entities
  parallel; second submission defined behavior.
- Evidence environment: local PostgreSQL + pinned SDK probes.
- Disposition: NOT PROVEN.
- WS-A evaluation (v1.9.0 source + `TestPartitionedQueueEvaluation`
  PASS): (1) partition key is a free string — tenant scoping is our
  construction (use the tenant-scoped entity key); (2) resident
  sleepers HOLD the slot (PENDING counted; suspend/Enterprise
  would free it — out of scope without owner approval);
  (3) second same-partition workflow waits ENQUEUED, then RUNS
  (ordering, not dedup); (4) FIFO per partition
  (priority, created_at) — head-of-line blocking inherent, no
  starvation at equal priority; (5) partitions independent;
  (6) ENQUEUED survives restart; deploy overlap follows the
  application_version clause. Partition key and DeduplicationID
  are mutually exclusive: ordering XOR attach, never both. Queue
  names are unique across applications sharing one system database
  (ConflictingRegistration otherwise). Decision: queue is
  ordering-only; active-sequence uniqueness stays a durable
  miniapp-side record (P4 probe demonstrates).
- P4 demonstration: `TestProbeConcurrency` proves business-level
  uniqueness independent of queue ordering (no queue involved;
  the partial unique index plus attach semantics is the
  mechanism). Disposition now PROVEN (local; Neon re-run in P7).

## F-13 — Seven-family determinism analyzer, operations, CI enforcement

- Original finding: R1 H6/G1-lint-half; R2 LF-06/LF-16; RC F-13.
  Rules specified; no implementation, fixtures, or CI gate.
- Plan requirement: `docs/phase0/determinism-rules.md` (7
  families); §19 Phase 1 (pinned-SDK-aware analyzer; strict CI
  for Class D); §7.7 operations.
- Source locations: no `analyzer/` package; CI
  (`.github/workflows/ci.yml`) has no analyzer step.
- Root cause: scheduled Phase 1 work not yet built.
- Category: unproven capability (greenfield).
- Required correction: `go/analysis` implementation faithful to
  the spec; positive/negative fixtures per family; strict CI;
  operations doc (retry/retention/inspection/alerting/runbooks).
- Regression test: fixture suite; known-bad flagged,
  valid-accepted; replay tests per workflow.
- Evidence environment: unit/static CI + local runs.
- Disposition: NOT PROVEN.
- P3 correction: `go/analysis` checker implementing all seven
  `determinism-rules.md` families (goroutine, select, map order,
  nondeterministic stdlib, mutable globals with DBOS-handle
  exemption, direct I/O, step-body APIs) with one-level helper
  descent and RegisterWorkflow boundary recognition; fixture
  packages detbad (13 violations) / detgood (clean incl.
  dbos.Go/Select, step-local time, allowed sentinels);
  `cmd/determinism-lint` multichecker wired as strict gate 6 in
  `scripts/checks.sh` (CI runs it); dogfood clean on all repo
  workflow code; operations doc at
  `docs/phase1/classd-operations.md`. `TestAnalyzer` PASS.
  Disposition now PROVEN (local; CI itself runs on push).

## F-14 — Framework/miniapp boundary, architectural-contract enforcement

- Original finding: R3 process weakness (ADR overclaims pass
  uncaught); R2 LF-12/LF-20; RC F-14. Canonical-service shape is
  real (good); no mechanical enforcement of entity_key scope,
  family/version refs, ADR refs, or guarantee language.
- Plan requirement: §6.3/§9.3 (canonical invocation); §9.1
  (entity_key, limits, kill switch); §10.1 (ADR required fields);
  §11.2 (family/version).
- Source locations: `registry/registry.go` (admission: no entity
  or version-policy checks); no ADR/manifest consistency tooling.
- Root cause: enforcement tooling never built; human review the
  only gate.
- Category: unproven capability (process + partial code).
- Required correction: structured-fact preflight (reject invalid
  config; warn/error/owner-decision diagnostics); overclaim lint
  warning-only; checklist + proposal where unreliable.
- Regression test: invalid fixtures rejected; valid admitted
  without false positives; CI/checks wiring.
- Evidence environment: unit + repo checks.
- Disposition: PARTIALLY PROVEN.
- P6 correction: `preflight` package + `cmd/preflight` checking
  title/scope binding, status/decision vocabulary, reviewer/score
  confirmation, draft-state machine (draft+unregistered SKIP,
  draft+registered FAIL, approved+unregistered FAIL),
  evidence-register references, wording/temporal recheck warnings,
  and warning-only overclaim lint with an empty always-fail list.
  Nine fixture tests PASS; live `--check-all` reports the
  lead-followup draft SKIP plus two overclaim warnings; wired as
  strict gate 7 in `scripts/checks.sh` (CI runs it).
  Disposition now PROVEN.

## Third-review additions (mapped, not duplicated)

- Touch/step boundary distinction → F-03 correction + WS-C tests.
- Cancel-during-ATTEMPTING semantics → F-02 + WS-D matrix.
- Approval recheck + reliable delivery → F-02 + WS-E protocol.
- Eligibility/policy recheck placement → F-06 + WS-G (durable
  boundary: last recheck inside the effect-claim transaction).
- Poisoned UNKNOWN mock → F-03 harness + F-10/WS-C tests.
- ADR/manifest consistency validation → F-14 + WS-J preflight.
- Rejected per prompt13: workflow-ID-as-invocation-identity,
  partitioned-queue-alone serialization (see F-04/F-12).

## Disposition roll-up

- DEFECT CONFIRMED: none.
- PARTIALLY PROVEN: F-06 (WS-G; volumetric controls in OSI-4).
- NOT PROVEN: none.
- PROVEN (local): F-01 (WS-B), F-02 (WS-D/E), F-03 (WS-C), F-04 (WS-A/P4), F-05 (WS-F), F-07 (WS-B), F-08 (WS-A/C), F-09 (WS-E), F-10 (WS-H), F-11 (WS-H), F-12 (WS-A/P4), F-13 (P3), F-14 (P6).
- PROVEN: none at capability level (component behaviors such as
  auth-before-dispatch, self-approval refusal, shared-pool
  atomicity, and tenant RLS are proven and stay proven as
  regression anchors).
