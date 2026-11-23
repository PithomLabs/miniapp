# `authorizeAttempt` Design Note (Phase 2, Stage 2)

**Status:** design approved pending owner review; Stage 3
implementation NOT YET RELEASED. No code changed for this note.

**Scope:** the canonical effect-time authorization boundary:
revalidate authoritative persisted state and atomically commit
`PENDING → ATTEMPTING` before any provider call. Constraints
C1–C7 are binding; §16 stops on genuine contradiction.

## 0. Constraint compliance (design-level)

| C | Verdict | Mechanism |
|---|---|---|
| C1 no advisory lock | HOLDS | tx takes row locks + Go RLock only; CLAIM/entity locks never acquired here (verified by construction + extended guard) |
| C2 global order | HOLDS | RLock → 1 → 2 → 4 → 5; level 3 taken only `FOR SHARE` where sequence-bound (no prod sequence tables in Phase 2 — order preserved by absence, §5) |
| C3 own tx | HOLDS | fresh pool checkout per authorization; never the attempt's `pgx.Tx` (signature takes pool/services, not tx) |
| C4 no new acquirers | HOLDS | zero new advisory sites; INFO-2 guard extended with a raw-SQL `checks.sh` gate |
| C5 pending semantics | HOLDS | P21-002 closed: pending ⟹ unsubmitted structurally; authorize treats pending as ineligible |
| C6 `ErrAttemptInProgress` | DESIGNED | typed propagation → framework conflict + same-key retry (no indefinite tx hold) |
| C7 orphan window | DESIGNED | liveness + authority revalidated in-tx immediately before the transition |

## 1. API and production callers

New method on `effects.Service` (owns the ledger; already owns
claim/attempt/takeover/transition):

```go
// authorizeAttempt runs the §4-ordered authorization transaction
// for one effect attempt. On success the row is ATTEMPTING and
// the caller must send through the canonical path. Refusals and
// wait signals never transition; infra errors are retryable.
func (s *Service) authorizeAttempt(
    ctx context.Context, tenantID string,
    intent Intent,           // threaded op identity (untrusted until bound)
    mode AttemptMode,        // fresh | takeover | reconcile
    worker WorkerID,         // generation/fencing identity (§5)
) (contract.EffectRecord, error)
```

`AttemptMode` selects the step-5 source predicate: `fresh`
(`PENDING` + attempt match), `takeover` (stale `PENDING` +
age check + adopt new attempt ID), `reconcile`
(`NOT_APPLIED` + full revalidation). One function, three
predicated shapes — never three paths.

Production callers (exhaustive; replaces every bare
`PENDING→ATTEMPTING` writer):

| Caller today | Change |
|---|---|
| `attempt()` (fresh claim winner) | replace bare `transition()` with `authorizeAttempt(fresh)`; `sendAndRecord` unchanged post-commit |
| `takeover()` (stale PENDING) | replace age-UPDATE + send with `authorizeAttempt(takeover)`; send after commit |
| NOT_APPLIED retry (no caller today — reconcile path missing) | new: route through `authorizeAttempt(reconcile)`; wire the single retry entry point found in Stage 3 inventory |
| `adopt()` short-circuits | UNCHANGED (settled reads, fresh-live contention, stale-ATTEMPTING quarantine authorize no new attempt) |

No-bypass inventory method (Stage 3 executes): grep all
`Channel.Send` call sites + all `ATTEMPTING` writers; each must
trace to `authorizeAttempt` success or be deleted. Known
non-bypass: `faultwrap` (test-only decorator),
`mockcap` (channel impl, never self-invokes), probe workflows
(call `Perform`, inherit the boundary).

## 2. Authoritative source per check (§7 mapped)

All reads below execute inside the authorization tx in §4 order
except RLock (in-process, held across). "Server-read" means
loaded from PG inside this tx, never from `intent` except as a
lookup key whose binding is then verified.

| §7 check | Source (server-read) | Lock |
|---|---|---|
| 7.1 tenant + invocation identity | `app.invocations` row by (tenant, invocation_id) | 1 `FOR UPDATE` |
| 7.1 liveness (incl. P21-002: pending ineligible; cancelled/failed/succeeded/quarantined ineligible) | same row, `status` | 1 (held) |
| 7.1 effect↔invocation binding | `app.effect_records.invocation_id` == intent == authorizing invocation; tenant match on both rows | 1 + 5 (predicate) |
| 7.1 no stale authorizer | run binding informational from row; attempt ownership at step 5 (§5) | 1 + 5 |
| 7.2 approval persisted + granted | `app.approvals` latest row for invocation — SKIPPED iff persisted invocation trust is autonomous AND the admitted tool manifest explicitly lists autonomous in `TrustModes` (P24-005 option (a); mirrors the submit choke; all other checks still apply) | 2 `FOR SHARE` (when evaluated) |
| 7.2 persisted-trust admission | persisted `trust_mode` ∈ declared `TrustModes` (empty list = no restriction); violation → `trust-mode-mismatch` refusal, independent of linkage | 1 (held) + manifest |
| 7.2 tenant/invocation/tool/operation binding | approval row columns vs invocation row + intent | 1 + 2 |
| 7.2 authority principal + intent | approval `decided_by`/scope (recorded at Decide) + recomputed `IntentHash(miniapp, version, tool, input)` from the invocation row (same inputs as park/submit choke) | 1 + 2 |
| 7.2 expiry/validity | approval `expires_at` + status, lazy-expiry rule shared with `pollApproval` | 2 |
| 7.3 grant coverage | pinned in-process snapshot under RLock (§6) | RLock (held) |
| 7.4 kill switches | `app.kill_switches` matched names (global, tenant, miniapp, capability, tool-declared — same name list as `StrictPolicy`) | 4 `FOR SHARE` |
| 7.4 token | invocation row `effect_token` vs intent token | 1 (held) |
| 7.5 effect eligibility + ownership | effect row: state/attempt/lease + worker generation (§5) | 5 predicated UPDATE |

`StrictPolicy.CheckPolicy` stays as the pre-claim coarse gate
(unchanged semantics); `authorizeAttempt` is the binding
revalidation — the two share check names and the kill-switch
name list (single source: unexported helper used by both, to
avoid drift).

## 3. Lock order, modes, transaction (§4/C1–C3)

```
RLock (pinned grants; in-process, first, held across)
  → 1 invocation FOR UPDATE (identity + liveness + token + tool/trust)
  → 2 approval FOR SHARE (latest linkage; skipped under the P24-005 autonomous exception)
  → [3 sequence FOR SHARE — conditional, absent in Phase 2 (§5)]
  → 4 kill_switches FOR SHARE (matched names) + grant-map read (RLock held)
  → 5 effect predicated UPDATE (state + attempt + lease + invocation binding)
→ COMMIT → release RLock → provider send (canonical path) → settle
```

- C1: no advisory statement exists in this tx (grep-verified +
  extended guard). RLock is a Go mutex: §4's no-hold rule
  governs PG locks; the contract's no-cycle argument (setter
  takes no PG locks) is re-stated with the actual setter in §6.
- C2: relative order preserved exactly; level 3 is taken only
  where sequence-bound (contract's own conditional), so its
  Phase 2 absence removes edges without reordering.
- C3: `authorizeAttempt` opens its own `store.WithTenant`
  checkout; the signature accepts services + pool, never a
  `pgx.Tx`. Callers pass no transaction; nothing to misuse.
- The effect UPDATE is the LAST statement: single predicated
  write, affected-row == 1 wins, 0 reloads + classifies (§4).
  No lock is held across the provider call (commit precedes
  send, enforced by construction: send lives outside the tx
  closure).

## 4. Outcomes, refusal, withdraw, audit (§7.6, §8, §10.1)

Result taxonomy (extends existing kinds; no new top-level kind):

| Outcome | Meaning | Error / signal |
|---|---|---|
| authorized | row is ATTEMPTING, caller must send | `nil` + record |
| wait | approval pending + valid (park path only) | `ErrApprovalPending` (wait, not failure; caller parks — never authorizes) |
| refuse/ineligible | terminal invocation, dead approval, mismatch | terminal fail-closed errors (below) |
| refuse/policy | grant/switch denial | terminal fail-closed |
| refuse/stale | generation/lease/ownership mismatch | terminal fail-closed (retry mints a fresh attempt) |
| contention | lost the step-5 predicate race | `ErrAttemptInProgress` (existing) |
| infra | SQL/connection/commit failure | retryable wrap (nothing committed) |

Refusal reasons reuse existing errors where they fit
(`ErrInvocationCancelled` for terminal invocation,
`ErrAttemptInProgress` for contention) and add narrow
effects-level terminal errors only where none exists
(approval/grant/switch/ownership/token — one var each,
`contract.Terminal` kind, REST 4xx passthrough like existing
terminal codes). No redundant taxonomy: names mirror the
`StrictPolicy`/choke vocabulary.

Withdraw rule (M3 failure-exit + §8 cancel-first): on
terminal-invocation or terminally-against-approval refusal
(denied/cancelled/decided-expired/lazily-expired), the SAME tx
also flips `PENDING → CANCELLED` (predicated, idempotent).
Transient refusals (grant/switch/policy, pending-approval wait,
mismatch, contention, infra) NEVER withdraw — the row stays
`PENDING` for restoration/retry. Mismatch never withdraws (a
wrong caller must not destroy another owner's row).

Refusal audit (X-13): best-effort `effect-refused/<reason>`
after rollback (refusal commits nothing, so the audit cannot
ride the tx). Attributed with tenant, invocation, op, reason;
never secrets. Success path audits nothing new at authorize
time (attempt evidence = the ATTEMPTING row; outcome audits
ride settlement as today).

Cancellation linearization (§8): cancel-first → authorize
reads terminal/cancelled under lock 1 → refuses (+withdraws);
no provider call. Authorize-first → ATTEMPTING committed;
Cancel observes in-flight via `LiveEffects` (settle-by-evidence,
existing), prevents subsequent authorizations, and never claims
recall. Tests assert both orders plus the non-recall wording in
error/report surfaces.

## 5. Lease, generation, fencing — owner requirement A (§7.5)

**Resolution: the effect ledger already provides sufficient
durable ownership; no new tables, no migration.** Rationale:

- Ownership root: `app.effect_records` PK (`business_op_id`) +
  `attempt_id` + `updated_at` lease + `invocation_id` binding —
  all durable PG columns, all read/written inside the
  authorization tx. No process-local state anywhere in the
  ownership decision.
- Worker identity (`WorkerID` param): the calling attempt's
  `(invocation_id, attempt_id)` pair. Fresh attempts present
  the ID just claimed; takeovers present a newly minted ID
  with the age check; reconcile retries present a new ID for
  a `NOT_APPLIED` row. Attempt IDs are uuids minted by
  framework code at the attempt boundary (never caller free
  text — `Perform` mints or validates; exact placement set in
  Stage 3).
- Stale refusal: the step-5 predicate requires state +
  attempt match (+ age for takeover, + invocation binding
  always). A superseded attempt ID, an expired lease claimed
  by another owner, or a mismatched invocation fails the
  predicate → typed stale refusal. No stale worker can use
  an old generation: old IDs never match a live row.
- Invocation binding closes the adopt() confused-deputy gap:
  `load`/`adopt` today never compare invocation IDs; the
  authorization tx verifies
  `row.(tenant_id, invocation_id)` against the authorizing
  invocation and refuses mismatches (without withdraw, §4).
- Run binding: one run per invocation is enforced by
  adopt-or-attach at the choke; the invocation row's `run_id`
  is read (informational, audit) but the effect fence keys on
  (op, attempt lineage), which is strictly finer. Attached
  (`attached_run_id`) resolution stays deferred to WS-E: no
  writers exist, and authorize treats attached like any
  invocation (liveness + binding, no run comparison).
- M5 entity sequences (multi-generation lifecycles,
  supersede/attach across invocations) stay probe-scoped
  (P24-004 owner disposition — explicit scope deferral, NOT
  an implemented generation-fencing guarantee):
  `probe_sequences` is fixture-only and no production flow
  needs cross-invocation generations. NAMED LIMITATION: the
  current probe performs no authorization-time generation
  validation (no verifier implementation, no identity
  propagation — mock-only effects). Activation trigger
  (explicit and enforceable): a miniapp manifest declaring
  sequence-bound effects — the framework-declared
  sequence-requirement marker in tool/workload
  configuration (exact field fixed at activation design;
  NEVER inferred from caller-supplied `Intent.Sequence`
  presence) — submitted for registry admission. Admission
  REFUSES it until all four land: (1) the framework-owned
  `SequenceVerifier` implemented and installed at service
  construction, (2) trusted identity propagation
  (framework-originated, bound to tenant + invocation +
  effect), (3) supersede→fence validation, (4) the X-08
  barrier test green. The refusal is mechanical (admission
  gate), not a judgment call. When active: missing identity
  or verifier fails closed; the verifier runs inside the
  authorization tx at lock-order position 3 (generation row
  `FOR SHARE`) and opens no transaction, performs no I/O,
  and acquires no advisory lock. Until then the contract's
  "if sequence-bound" conditional covers the absence, and
  any effect presenting sequence identity the installed
  verifier cannot authoritatively resolve refuses.

## 6. Grant snapshot and revocation — owner requirement B (§7.3)

Missing safeguards confirmed in Stage 0; all implemented in
Stage 3 (mandatory, not documented-away):

- `auth.Policy` gains `RWMutex` + guarded accessors per §4.1;
  value receivers migrate to pointer receivers (compile-
  enforced; `go vet` copylocks stays clean); the single test
  mutator migrates to a `SetGrantsForTest`-style setter API;
  a grep test forbids direct map writes outside `auth`.
- Setter API takes `Lock`, swaps maps, recomputes the hash;
  authorization holds `RLock` across its read set. Setter
  performs zero PG I/O (contract's no-cycle premise,
  mechanically true by construction).
- Config hash: `sha256` over canonical JSON of the grants
  map only (Go map marshal sorts keys — deterministic;
  keys/credentials never hashed or exposed).
- `GET /healthz` on the REST server returns
  `{status, grant_config_hash}` via a new
  `Service.GrantConfigHash()` RLock accessor; startup logs the
  same hash. Unauthenticated (hash reveals nothing), exact
  shape in Stage 3.
- `miniapp auth verify-revocation <expected-hash>`: new CLI
  subcommand; replica inventory from an explicit operator-
  provided source (`--replicas host,...` and/or file —
  documented as THE inventory mechanism, no magic discovery);
  polls each `/healthz` with timeout; reports stale /
  unreachable / mismatched; exits nonzero unless every listed
  replica matches (never success on incomplete inventory).
- Model: 10-minute maximum staleness baseline; rolling
  restarts exceeding it are deployment failures; rollback
  preserves restrictive grants (procedural); honest claims
  only ("rejected on the next authorization by a replica
  holding current config"). D: grep-audit contract/comments/
  status surfaces for immediate-revocation language. E:
  `docs/phase1/classd-operations.md` (verify path in Stage 3;
  create the revocation section wherever operations docs live
  if relocated) gains routing/drain, command, window,
  rollback, and verification procedure.
- Tests: §H-7 wait→revoke→refuse E2E via setter API +
  revocation-racing-authorization order test + hash stability
  test + CLI inventory-incomplete test.

## 7. Settlement atomicity — owner requirement C

Stage 0 confirmed the drift: `Settle` still runs load →
`RECONCILING` → outcome across transactions, the state map
retains the edges, and the §16.1-recorded tests do not exist.
Correction (minimal, Stage 3 with the ledger work):

- One tx: predicated `UPDATE ... SET state = outcome WHERE
  business_op_id AND state = 'UNKNOWN'` + `effect-settled`
  audit insert in the same tx (per §16.1: commit leaves both
  durable; crash-before leaves UNKNOWN for operator retry).
- Idempotent retry: same (op, outcome, note) → success, with
  audit backfill when the audit row is missing (explicit
  backfill path, tested directly); different outcome/note →
  terminal conflict (first decision stands).
- Map cleanup: remove `RECONCILING` edges from
  `ValidEffectTransitions` after verifying zero remaining
  references (Settle rewritten; `adopt` default already
  rejects it); keep the constant only if a scan proves a
  reader needs it, else delete.
- Guards preserved: UNKNOWN stays non-retryable for provider
  attempts; `UNKNOWN → ATTEMPTING` stays prohibited; blind
  retries unchanged.
- Tests: fault-injected failure at the audit boundary
  pre-commit (new `Fire` point inside the tx closure),
  lost-response-after-commit convergence, idempotent repeat,
  conflicting second Settle — the four §16.1 cases by name.

## 8. Transaction compatibility — owner requirement D (C1–C7)

Interaction graph (extends the P21 §C.7 graph; `TA` is new):

```text
Attempt tx (A): [adv claim] → [adv entity?] → [SELECT inv, lock-free]
  → handler {direct SQL + Perform → pool checkout} → [UPDATE inv, lock 1] → COMMIT
Authorize tx (TA): [RLock] → [1 inv FOR UPDATE] → [2 appr FOR SHARE]
  → [4 switches FOR SHARE + grants] → [5 effect predicated UPDATE] → COMMIT → send
Cancel/settle (D, current multi-tx): [1 pred UPDATE] ; [2 UPDATE] ; [5 batch] (separate txs)
Settle-atomic (S): [5 pred UPDATE + audit, one tx]
```

Re-proof (no constructible cycle):

- TA takes no advisory lock (C1 — grep + extended guard), so
  no TA→advisory edge exists; A→TA waits run only on row 1
  (single shared object, predicate-correct either order).
- TA holds rows in global order 1→2→4→5 (C2; level 3 absent
  by conditional). No other multi-row holder acquires out of
  order: D holds ≤1 row lock per statement/tx (current
  multi-tx shape), S holds row 5 alone. Any D/S↔TA wait is
  same-object serialization, never a cycle (a cycle needs two
  objects held in opposite orders — unconstructible here).
- RLock↔PG: RLock is held across PG waits, but the setter
  takes no PG locks and readers never upgrade (C2/no-hold
  argument restated with the §6 setter shape). No Go↔PG cycle.
- Same-effect TA↔TA contention serializes on row 5 with the
  ownership predicate: exactly one winner, losers reload +
  classify (contention/stale/terminal). Cross-effect TAs touch
  disjoint row-5 objects; shared 1/2/4 rows serialize in
  order (cross-invocation contention test in Stage 4).
- Attempt-tx (A) vs TA on row 1: sequential per invocation
  (authorize runs inside the handler, `completeTx` after it
  returns); cross-invocation contention serializes with
  predicate re-evaluation. No inversion (TA never waits on an
  advisory; A never waits on 2/4/5 while holding 1... precisely:
  A holds row 1 only at its final UPDATE, a single statement
  whose only wait is same-object).
- X-05 trigger (recorded, not implemented): if Cancel ever
  becomes one tx (1→2→5), re-run this proof — the order is
  compatible by inspection (same direction as TA) but must be
  re-verified with the real code, not assumed.

C6 composition: the C6 conflict (typed `ErrAttemptInProgress`
→ framework conflict, same-key retry, nothing committed) adds
no edge — it returns before any row lock is taken on the
retry path's behalf; the retried attempt is a fresh TA.

## 9. Per-invariant test list (maps §10)

- Matrix (§10.1): one test per row; each refusal asserts the
  typed reason + no ATTEMPTING + no send (mock log empty) +
  X-13 audit where required; the park row asserts wait (no
  authorize, workflow parks); contention rows assert single
  winner (affected-row counts + one send).
- Races (§10.2): cancel-first/authorize-first both orders
  (lock-gate forced); worker-vs-worker same effect; stale
  generation (old attempt ID refused); lease turnover;
  approval/policy mid-flight changes; restoration-vs-dispatch;
  shared-row contention — all on real PG19 sessions with
  persisted-row + mock-log + audit assertions.
- Crash (§10.3): Kill9/fault at all six boundaries with
  retry/adopt/reconcile/UNKNOWN dispositions; never blind
  retry, never fabricated success.
- C6 fixture: Class A tool invoking `Perform` with
  deterministic op IDs; InProgress → conflict (not FAILED);
  same-key retry adopts; bounded policy visible in code +
  contract; tx-hold assertion (conflict returns without
  waiting out the lease).
- Bypass: per-path test that failed authorization sends
  nothing (mock log empty) for every inventoried path.
- Settle: the four §16.1 cases. Grants: §H-7 + race-order +
  hash + CLI tests. Image pins: digest assertion procedure
  (recorded commands, local + CI config diff).

## 10. Migration assessment

Expected: NO migration. Ownership (attempt/lease/binding),
approval linkage, switches, invocation state, and refusal
audits all use existing columns. Triggers that would reopen
this with an owner decision: production entity-sequence
fencing (§5 deferral trigger), or a Stage 3 discovery that a
required predicate cannot be expressed on current columns
(stop per §16, do not improvise).

## 11. Open points for Stage 3 (mechanical, non-blocking)

- `Perform` attempt-ID mint-or-validate placement (fresh
  uuid per attempt boundary; exact function named in Stage 3).
- `ErrApprovalPending` wait-signal plumbing into the
  workflow park path (probe pattern exists; production shape
  set in Stage 3).
- `Service.GrantConfigHash()` accessor shape + `/healthz`
  route wiring + CLI flag naming (per §6 decisions).
- `classd-operations.md` path verification (E).
- Exact `Fire` point names for Settle crash tests
  (following the `claim-response`/`settle-response`
  precedent).

Gating prerequisite (P24-008 — NOT deferred open work): before
the first non-mock provider is enabled, authorization must
verify that the requested capability belongs to the admitted
tool's declared capability set. Mock-only operation is
unaffected; real-provider admission is refused until this
membership check lands with its tests.

Stages 3–5 proceed only on owner release of this design.

## 12. Owner conditions (Stage 3A release; binding)

### 12.1 Framework-owned sequence verifier (no caller callback)

The per-call check callback sketched during planning is
REJECTED: a caller-supplied check could declare its own
sequence valid. Instead:

- `effects` defines `SequenceVerifier` (interface) + `Intent`
  carries sequence *identity* only
  (`Sequence *SequenceRef{Subject, Generation}` — data,
  no behavior).
- The verifier is installed at service construction
  (`effects.Config`), like `Channels`/`Canceller`/`Policy` —
  never from an `Intent`. It receives the authorization tx
  and verifies at lock-order position 3: generation row
  `FOR SHARE`, active status, generation match, tenant +
  invocation + effect binding. It opens no transaction,
  calls no provider, acquires no advisory lock.
- Implementation placement: the generation-row read lives
  with the table owner (`probe/classd`, the sole sequence
  domain); `effects` owns the interface, the binding checks,
  and the fail-closed rules — no `effects → probe` table
  dependency. Production wiring (`cmd/miniapp`, which
  already serves the probe) installs the probe verifier;
  its absence where required fails closed.
- Fail-closed rules: sequence identity present but verifier
  missing/unavailable → refuse; binding mismatch → refuse;
  stale generation → refuse. An operation omits fencing only
  when no sequence identity is present AND the deployment's
  installed verifier set covers every sequence domain it
  serves (today: probe-only; production serves zero
  sequence ops — a true statement about the current
  system, re-verified by the no-bypass inventory, not an
  assumption callers can exploit). Presence-based
  classification is INTERIM: at §5 activation the
  sequence requirement becomes framework-declared
  (tool/workload configuration), never inferred from
  caller-supplied identity presence.
- Activation follows the §5 trigger (sequence-bound
  manifest declaration refused at admission until verifier +
  propagation + supersede validation + X-08 barrier land);
  until then, any effect presenting sequence identity the
  installed verifier cannot authoritatively resolve refuses.

### 12.2 Implementation order (grant infrastructure first)

1. Shared helpers + tx-scoped approval groundwork.
2. Grant snapshot mutex, setter API, hash infrastructure.
3. `authorizeAttempt` with the real pinned snapshot.
4. Ownership/provider-path wiring, revocation tooling,
   settlement, pins, verification.
5. No temporary no-op locking; no authorization path may
   claim snapshot protection before the mutex exists.
