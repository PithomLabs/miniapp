# Prompt 24 — Independent Adversarial Review: WS-B Phase 2 Authorization Core (Gate 1)

**Review target:** Prompt 23 Stage 3A authorization core as present in the working tree
**Review type:** independent, adversarial, read-only (this report is the only file created;
the prompt's §16 names `docs/phase2/adv_review_prompt23_gate1.md` — the owner-directed path
`docs/adv_review_prompt24.md` is used instead)
**Verdict: NOT CLEAR** — two HIGH findings compromise transaction correctness/fencing and the
audit boundary. Both are execution-proven, narrowly scoped, and fixable without broad rework;
a targeted re-review of the corrections is the appropriate next gate, not a rewrite.

## 1. Executive verdict

**NOT CLEAR.** The authorization core is well-architected — the transaction shape, lock order,
ownership predicates, grant snapshot, and no-bypass wiring are sound as designed — but two
structural defects in the delivered code violate binding requirements with execution proofs:

- [P24-001](#p24-001) (HIGH): terminal-invocation withdrawal never persists — the
  `PENDING → CANCELLED` flip runs inside a transaction that always rolls back (the refusal is
  returned as an error), while the audit unconditionally records `effect-withdrawn`. Every
  cancel-raced authorization leaves a PENDING row behind and a false audit. Proven by
  deterministic POC (row PENDING + withdrawn audit, 0.07s).
- [P24-002](#p24-002) (HIGH): outcome settlement is not fenced by attempt — `transition()`'s
  predicated UPDATE constrains state but not `attempt_id`, so a stale worker's late outcome
  overwrites a newer attempt's live ATTEMPTING row. Stage 3A's reconcile path introduces the
  first multi-epoch ATTEMPTING rows, making this reachable. Proven by deterministic POC
  (epoch A's receipt landed on epoch C's attempt). Violates prompt22 §8's explicit prohibition.

Three MEDIUMs require owner disposition (Settle 5→1 lock-order violation; incomplete M5/sequence
trust boundary; gated+autonomous submit/authorize incoherence — a behavior regression for a
manifest-offered production configuration). Six LOWs and bounded residuals follow. No CRITICAL.

The central invariant — *no provider call unless the framework atomically revalidated and
committed ATTEMPTING* — HOLDS for sends: both HIGHs corrupt terminality/ledger integrity, but
neither permits an unauthorized send (all failures run fail-closed). The remaining Stage 3A
test matrix (authorization matrix, both cancel-race orders, crash matrix, bypass tests, C6
proof) is genuinely outstanding — this review verified the absences rather than assuming them.

### Owner questions — direct answers

| Priority | Question | Answer |
|---|---|---|
| Critical | Framework-owned, un-overridable sequence verifier? | SHAPE yes (Config-installed, data-only identity — no Intent path to the callback), but NO verifier exists, nothing presents identity, and presence-based classification is a latent downgrade vector ([P24-004](#p24-004), MEDIUM) |
| Critical | Checks + `PENDING → ATTEMPTING` genuinely atomic? | YES for authorization (single tx, verified query-level); NO for the withdraw half (rolled back) and NO for settlement fencing (attempt-agnostic) ([P24-001](#p24-001), [P24-002](#p24-002), HIGH) |
| High | Grant snapshot covers read set, released before send? | YES — RLock across the tx, `Locked` variant inside, released by defer before any send; setter takes no PG locks |
| High | `adopt()` cross-invocation adoption? | NO — step-5 invocation predicate + mismatch refusal close the confused-deputy gap at the boundary (all ten attacks analyzed, §8) |
| High | Transient vs terminal semantics preserved? | PARTLY — refusal taxonomy is correct, but terminal withdrawal never persists and the audit misreports it ([P24-001](#p24-001)) |
| High | `effect-contention` same-key recovery sound? | MECHANISM yes (full rollback, no FAILED, 409, retry converges); running-record message overclaims; op-ID guidance + fixture outstanding |
| High | Atomic Settle crash-safe incl. audit/cleanup? | ARCHITECTURE yes (single tx, idempotent repeat, defined conflict, no RECONCILING); 5→1 lock inversion ([P24-003](#p24-003)); operator-auth moot (unwired); 4 crash tests outstanding |
| High | Any provider path bypasses authorization? | NO — sole `Send` site post-authorize; sole prod Perform caller (probe); all ATTEMPTING writers inside authorize; capability/tool binding unenforced but moot with one mock channel ([P24-008](#p24-008), LOW, trigger-bound) |

## 2. Exact baseline and provenance

- Repository `github.com/PithomLabs/miniapp`, branch `main`, HEAD `1a67ba22b50e2ff54568db308a34d75b14ed2a0d`
  ("prompt21" commit = P20 closeout implementation + P21 review).
- Worktree digest: `2141559f601f93ff20c932819fa4019f071634d85ce19bc905277eeda750b895`
  (`scripts/digest.sh tree .`, repo root). Delta vs HEAD: 31 tracked modifications + 12 untracked
  paths (full inventory in §B of working notes; every file reviewed or skimmed per §14).
- Design `docs/phase2/authorize-attempt-design.md`: `0f4823564780c582…` (differs from the
  Stage 2 evidence row `5f5506f9…` — owner conditions §12 added after that row; the row predates
  them and is not updated).
- Contract `docs/phase1/invocation-lifecycle-contract.md`: `e188e222cb2974c5…` (matches the
  Stage 2 row — unchanged since).
- PostgreSQL `19.0 (Debian 19.0-1.pgdg13+1)`, disposable Docker container from image
  `postgres:19.0`, full immutable identity
  `sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
  (image ID and RepoDigest coincide for this local image). Fresh `miniapp_dev`, `miniapp_red`,
  `miniapp_dbos` databases. The project's long-lived dev container was untouched; the review
  container was removed afterwards.
- Go `1.27.3 linux/amd64`, `GOWORK=off`. DB-touching commands ran unsandboxed (sandboxed
  network namespaces cannot reach host ports); all reads, digests, and reconstructions ran
  sandboxed. No commits, pushes, or repo mutations; POCs ran in a `/tmp` tree copy only.
- Test commands (CWD repo root unless noted): `gofmt -l .`, `go vet ./...`,
  `go test -p 1 ./... -count=1` (+ `TEST_DBOS_DATABASE_URL`), focused package runs,
  `go test -race -p 1 ./runtime/ ./approvals/ ./effects/`, `bash scripts/checks.sh`,
  and two POC tests in the `/tmp` copy. Full results in §14.

## 3. C1–C7 compliance assessment

| Constraint | Required result | Source evidence | Executed test evidence | Verdict |
|---|---|---|---|---|
| C1 | No advisory lock in authorizeAttempt | All authorize SQL read query-level: `FOR UPDATE`/`FOR SHARE` SELECTs + predicated UPDATEs only; verifier interface forbids advisory; zero implementations exist | None direct (no lock-profile test); INFO-2 guard green (call-based sites unchanged) | HOLDS (source-proven; recommend a `pg_locks` assertion in the matrix) |
| C2 | Correct global lock order | Authorize: RLock → 1 → 2 → 4 → 5 verified statement-by-statement; full wait-for graph in §4 (8 parties) | Forced-order tests: none yet for authorize | CONDITIONAL — authorize holds, but Settle takes 5→1 ([P24-003](#p24-003), MEDIUM) |
| C3 | Independent authorization transaction | Signature takes services+pool, never `pgx.Tx`; `WithTenant` checkout inside; all 3 call sites pass no tx | Indirect: all effects tests pass through it | HOLDS |
| C4 | No new advisory paths / guard intact | Exhaustive grep: zero new sites; INFO-2 guard green | Guard suite green incl. `-race` | HOLDS (raw-SQL gate correctly not added — no new SQL advisory paths exist to trigger the conditional) |
| C5 | P21-002 semantics respected | Atomic adopt-flip verified; pending ⟹ unsubmitted structurally; authorize treats pending ineligible | `TestP22AdoptionMarksPendingRunning` + 4 characterization suites green | HOLDS (reachability-proof accuracy observation, §14) |
| C6 | Class A Perform recovery | Contention → whole-tx rollback → terminal 409 same-key retry verified; record untouched by construction | NO fixture test exists (outstanding per §14); message overclaims for running records ([P24-010](#p24-010), LOW) | MECHANISM SOUND, INTEGRATION UNPROVEN |
| C7 | O-01 closed at effect boundary | Liveness + authority revalidated in-tx before the transition; terminal refuses; orphan cannot send | POC-1 exercises the terminal branch (refusal holds; withdrawal lost) | PROTECTION HOLDS, WITHDRAWAL BROKEN ([P24-001](#p24-001), HIGH) |

## 4. Authorization transaction analysis (Priority 1)

Read `authorizeAttempt`/`authorizeTx` end to end; every SQL statement checked for locks.

1. **Own transaction + pool checkout: YES.** `store.WithTenant(ctx, s.pool, …)` inside
   `authorizeAttempt`; the signature accepts services and pool, never `pgx.Tx` (C3).
2. **No advisory lock, direct or indirect: YES.** No advisory statement in the tx; helpers
   audited: `loadInvocationTx` (SELECT … FOR UPDATE), `ByInvocationTx` (SELECT … FOR SHARE),
   `engagedSwitchTx` (SELECT … FOR SHARE), `transitionAuthorized` (predicated UPDATE),
   `classifyLoss` (SELECT … FOR UPDATE), registry (in-memory), `GrantAllowsLocked`
   (in-memory under held RLock), `SequenceVerifier` (none installed; interface contract
   forbids advisory/tx/I-O). No retries, no callbacks, no nested transactions.
3. **Authoritative persisted state: YES**, with one caller-supplied exception analyzed below:
   invocation row (lock 1), approval row (lock 2), switches (lock 4), grants (pinned snapshot),
   effect row (lock 5) are all server-read. Intent fields are lookup keys whose bindings are
   verified — except `Operation.CapabilityID`, which selects kill switches and the channel
   with no tool-membership check ([P24-008](#p24-008), LOW — one mock channel today).
4. **Lock order RLock → 1 → 2 → 4 → 5: YES**, exact modes: Go RLock held across; 1 invocation
   `FOR UPDATE`; 2 approval `FOR SHARE` (gated tools); 3 conditional (absent — no verifier);
   4 switches `FOR SHARE` + grant-map read; 5 effect predicated UPDATE last. Commit precedes
   send by construction (send lives outside the tx closure, in the caller).
5. **Affected-row checks: YES** — step-5 UPDATE requires exactly 1; zero reloads
   (`FOR UPDATE`, same position) and classifies contention/stale/mismatch — never success.
6. **Stale pre-tx checks: none load-bearing.** `CheckCancelled`/`CheckPolicy`/claim run before
   the tx but every security decision is revalidated inside it (C7). The pre-gates are
   fail-closed hints, correctly treated as such.
7. **Provider before commit: NO.** Single `Send` site (`sendAndRecord`), reachable only after
   `authorizeAttempt` returns success (post-commit record). All three callers (fresh, takeover,
   reconcile) follow authorize-then-send; unknown-capability paths transition FAILED without
   sending (self-row only — analyzed benign in §13).
8. **Success without committed ATTEMPTING: NO.** Success returns the post-UPDATE reloaded row
   from a committed tx; every other outcome is an error return with rollback (except the
   withdraw finding below — a committed flip is *lost*, the reverse failure).
9. **In-memory state implying authorization after failure: NO.** No caches, no out-params on
   error paths; the `withdrawn` flag misreports (see next), but no success state escapes.
10. **Rollback/release on all paths: YES**, except the withdraw's *intent*: pure refusals roll
    back correctly; RLock releases via defer on success/refusal/panic; pool tx closes via
    `WithTenant`. The defect: the terminal-withdraw flip is *meant* to commit but structurally
    cannot ([P24-001](#p24-001)).

Wait-for graph (8 parties; → is waits-on; rows are PG row locks unless noted):

```text
A  attempt tx:     [adv claim?] → [adv entity?] → handler{direct SQL, Perform→TA} → [UPDATE inv row1] → COMMIT
TA authorize tx:   [RLock] → [1 inv FOR UPDATE] → [2 appr FOR SHARE] → [4 sw FOR SHARE] → [5 eff UPDATE] → COMMIT → send
D  cancel/settle:  [1 pred UPDATE] ; [2 CancelPending]  (separate txs; NO effect batch exists — §9)
X  Decide:         [2 appr FOR UPDATE + pred UPDATE]  (approval rows only; never waits on 1)
S  Settle-atomic:  [5 eff UPDATE] → [1 inv FOR UPDATE] → COMMIT   ← ORDER VIOLATION (5→1)
L  lease/takeover: same shape as TA (takeover IS AuthorizeTakeover mode)
Q  supersede:      [3 probe_sequences UPDATE]  (no TA takes 3 — no edges)
G  grant setter:   [Go Lock] (no PG I/O) ; readers hold RLock across PG waits, never upgrade
```

Cycle analysis: TA↔A sequential per invocation, same-object serialization across (no
inversion — TA never waits on advisory; A holds row 1 only at its final single statement).
TA↔X/D/Park same-direction or same-object, predicate-correct either order. TA↔TA serializes
on row 5 with ownership predicates (one winner). RLock↔PG acyclic (setter takes no PG locks;
Go RWMutex writer-preference prevents setter starvation; pool waiters never hold RLock in a
setter-dependent chain). **One constructible cycle: TA↔S** — TA holds 1/wants 5 while S holds
5/wants 1, when TA targets a row S is settling (TA's belief stale: PENDING/NOT_APPLIED at
decision, UNKNOWN when S commits first). Requires a 3-way interleave + operator timing;
PostgreSQL's deadlock detector aborts a victim (retryable, converges) — availability-only,
narrow, but a genuine C2/§4.2 violation ([P24-003](#p24-003), MEDIUM). Fix is trivial
(lock-free attribution read — identity columns are immutable).

## 5. Sequence verifier trust-boundary analysis (Priority 2)

Twelve questions, answered against the actual code (not the design's promises):

1./2. **Created/owned: NOWHERE.** `SequenceVerifier` is an interface in `effects`; zero
   implementations exist repo-wide (verified by exhaustive grep); `Config.Sequences` is never
   set — not in production wiring, not in tests. The §12.1 probe verifier was not implemented.
3./4. **Caller override / miniapp no-op: IMPOSSIBLE BY CONSTRUCTION.** The verifier comes only
   from `effects.Config` (service construction); `Intent` carries `*SequenceRef` data only —
   no behavior, no callback field. There is no callback to supply or replace. The owner's core
   demand (no caller callback) is satisfied structurally.
5./6./7. **Identity change / binding / trusted sourcing: NO ENFORCEMENT.** Nothing sets
   `Intent.Sequence` in production (the probe omits it); no code binds identity to
   tenant/invocation/effect (that logic would live in the missing verifier + §12.1 binding
   rules, unimplemented).
8./9./10. **Tx use / position / side effects: SPECIFIED, UNIMPLEMENTED.** The call site runs
   the verifier inside the tx at position 3 with the right binding struct; the interface
   contract forbids new transactions, provider calls, and advisory locks — but no
   implementation exists to audit.
11./12. **Missing/invalid identity / unavailable verifier: FAIL CLOSED — VERIFIED.** Identity
   present + nil verifier → `RefusalStaleOwnership`; verifier error → same refusal. Any
   sequence-present op refuses today.

Required-semantics assessment: non-sequence effects omit fencing by *presence* of
`intent.Sequence` — a caller-controlled datum. A sequence-bound effect is therefore
*downgradable by omission*: presenting no identity skips all fencing silently. Today this is
unexploitable (zero sequence ops exist; the probe — the fence's only consumer — presents no
identity and is mock-only), but the §5 activation trigger ("introduce tables + validate
generation") does not include a *forcing mechanism* (tool-declared sequence requirement or
op-ID subject parsing), so the first sequence workload inherits the hole. Separately, owner
Decision A ("preserve the probe fence and validate it at the boundary") is half-unimplemented:
the probe workflow has no generation-active re-check before `effectStep`, no fence runs at
authorize, and the X-08 barrier test was never written — a superseded-but-live probe
generation (stale via a slow approval wait, superseded by a concurrent run) would send its
old-generation op. Mock-only today; the reference pattern is unfenced.

Per the §5 blocking criterion ("depending on exploitability"): exploitability for consequential
effects is zero today → not HIGH. Filed as [P24-004](#p24-004) (MEDIUM): the trust-boundary
shape is correct and attendance is safe-today, but the owner must disposition the missing
verifier, the omission vector, the Decision-A gap, and the trigger amendment before any
sequence workload. Details and required correction in §15.

## 6. Approval and intent-binding analysis (Priority 3)

`checkApprovalBinding` verified field-by-field against server-read state: tenant, invocation,
miniapp, version, tool/operation, recomputed `IntentHash` (from the invocation row's input —
never intent-supplied), approver scope vs tool requirement, status machine
(pending→wait-if-unexpired/expired-otherwise; decided→approve-only + non-empty non-self
`decided_by` + post-grant expiry check; cancelled/expired/unknown→refuse). `IntentHash` has
one canonical definition (`contract/intenthash.go`, NUL-separated, canonical-JSON); the
runtime copy delegates (verified diff); all producers (park, choke, authorize) recompute from
authoritative rows. `ByInvocationTx` holds `FOR SHARE` at position 2, latest-only, tenant-
predicated, no fault hook (pure primitive); the `Decide` transaction (approval-row `FOR UPDATE`
only, never waits on row 1) is unweakened — readers share, deciders exclude, no cycle (§4).
Pending approval parks (`ErrApprovalPending` — wait, never authorize, withdraws nothing); no
tx or row lock is held across any workflow/signal wait (verified: waits happen outside txs).
Malformed/missing linkage fails closed in every branch (including unknown approval statuses).
`Resource` is recorded but unset by all producers (always empty) — the unchecked field is moot.

One structural gap: **approval linkage is required for gated tools regardless of row trust**,
while the submit-time choke exempts autonomous-trust records. A gated tool invoked with
autonomous trust (a manifest-offered production configuration — the probe's `ToolRun` declares
`Required: true` + `TrustModes: [Confirm, Autonomous]`) submits without linkage and then has
*every* effect refused with `approval-missing`, permanently (no API can park a running record).
Pre-Stage-3A these effects sent (no effect-time approval check) — this is a behavior
regression for that configuration, in the fail-closed direction. No test covers autonomous
gated effects (zero `TrustAutonomous` references in probe tests). The design never addresses
the trust exception. This needs an owner semantic decision — see [P24-005](#p24-005) (MEDIUM):
(a) authorize honors row-trust autonomous like the choke, (b) the choke drops the autonomous
exemption (fail fast at submit), or (c) manifests forbid Required+Autonomous pairs.

## 7. Grant snapshot and revocation assessment (Priority 4)

`auth.Policy`: mutex owned by the object; all methods pointer-receiver (copylocks-clean —
`go vet` green); all grant reads via accessors (`GrantAllows` self-locking,
`GrantAllowsLocked` under held RLock); credential resolution under RLock; all mutation via
`SetGrants` (Lock, wholesale swap, hash recompute, zero PG I/O — the no-cycle premise holds
mechanically); exhaustive grep finds zero production map accesses outside `auth` and zero bare
`auth.Policy{` literals (sole constructor `NewPolicy` in `main.go`). `Authorize`'s unlock-
before-inner-read is safe (maps never mutated in place, only replaced; happens-before via the
mutex). Authorization holds RLock across the full tx including commit; released by defer on
all paths including panic, strictly before any provider send; no caller invokes the
self-locking form under a held RLock (deadlock-free by inspection + `-race` green). The
`p22` mutator migrated to the setter; the guard test enforces the discipline on prod and test
sources alike.

Config hash: `sha256` over canonical JSON of the grants map only — deterministic (sorted
keys), credential-free (verified by test), sensitive to every grant change (verified by test),
insensitive to map iteration order (by construction). Scope limitation (recorded, not a
finding): role/authority changes in `Keys` do not affect the hash, so hash comparison cannot
detect role-only drift — the hash is a *grant*-configuration hash as documented; revocation
verification inherits that scope. `/healthz` exposure and `verify-revocation` are NOT
implemented (zero references outside `auth.go`) — honestly outstanding per §14, and the core
makes no revocation claim that depends on them: no doc, comment, or status surface promises
immediate revocation (spot-checked §4.1, design §6, contract X-12 rows). The ten-minute
staleness baseline and cross-replica restart requirements remain explicit in the contract and
design. No gap between the claimed model (bounded propagation, restart-driven coherence) and
the core (snapshot pinning + atomic swap).

Guard-test gap ([P24-009](#p24-009), LOW): the line-based matcher catches `.Keys[`/`.Grants[`
and `auth.Policy{` but not whole-map assignment (`policy.Grants = …`, which would bypass the
setter *and* skip hash recompute) or bare `.Grants`/`.Keys` reads. No such usage exists
today (verified); the guard should grow an AST or assignment-pattern rule before the next
policy touch.

## 8. Ownership/adoption attack analysis (Priority 5)

All ten attacks attempted against the *shipped* authorize path (step-5 predicate +
`classifyLoss` + `checkOperation`), not against `adopt()` in isolation (`adopt` is fail-open
by itself — it compares no invocations — and is safe only because every send path funnels
through authorize, which was verified):

1. Cross-tenant adoption: blocked at three layers (op-ID prefix check, tenant predicate in
   every query, tenant-scoped `loadInvocationTx`). Existing `TestCrossTenantOpIDs` green.
2. Cross-invocation adoption: step-5 predicate requires `invocation_id` match;
   `classifyLoss` checks binding *before* state and refuses mismatches without withdrawing.
   The Stage 2 confused-deputy gap is closed where it matters (before ATTEMPTING).
3. Cross-tool adoption: `checkOperation` (capability/operation/payload equality, run before
   every live path) + authorize's tool resolution from the invocation row. A different tool
   under the same op ID is refused.
4. Op-ID reuse with different operation: `checkOperation` refuses (capability/operation/
   payload); covered by `TestOperationReuseConflict` (green).
5. Forged attempt lineage: fresh mode matches the claimed ID (self-consistent for the
   winner); takeover/reconcile mint framework uuids; stale IDs fail predicates. Fresh IDs are
   caller-supplied free text, contrary to design §5 — vacuous-harmless (lease-freshness and
   claim-winner exclusivity carry the safety, not the ID match) but a design-compliance gap
   ([P24-007](#p24-007), LOW).
6. Old lease holder adopts new lease: takeover predicate requires PENDING + age + invocation;
   a newer attempt shows ATTEMPTING → predicate fails → contention/superseded, never adoption.
7. **Stale worker settles newer owner's attempt: SUCCEEDS — [P24-002](#p24-002) (HIGH).**
   `transition()` constrains state, not `attempt_id`; POC-2 proves epoch A CONFIRMs epoch C's
   live row. (The `adoptStale` quarantine path is age-guarded and safe; only outcome
   settlement is exposed.)
8. Terminal revival: settled rows short-circuit to reads; CANCELLED/terminal refuse; no path
   re-enters PENDING (nothing writes it — verified writers inventory); NOT_APPLIED re-enters
   only via full-revalidation reconcile (§11).
9. Retry creates a second logical op: same op ID routes to adopt/reconcile (no duplicate);
   new op ID is definitionally a new op. Cross-key Class A recovery vs invocation binding is
   a guidance gap (op-ID derivation pattern undemonstrated — outstanding C6 work, §10).
10. Settled-read shortcut → new provider op: reads return records without authorization, but
    every subsequent send re-enters `Perform` → full authorize. No initiation flows from a
    read. (Pre-existing, mitigated cross-invocation read disclosure: exact op ID *and* exact
    payload required — noted as accepted residual.)

`BusinessOpID` stability: framework-enforced for the probe (deterministic subject:generation
derivation, verified); caller-supplied in general with tenant-prefix + payload-match binding.
Attempt lineage (`attempt_id` per epoch) is durable and finer than run identity; attached-run
resolution stays deferred per design (authorize treats attached rows by liveness+binding only
— no run comparison — consistent with "no writers exist").

## 9. Withdrawal classification assessment (Priority 6)

Refusal taxonomy verified against the owner table: temporary policy/grant/switch denials
refuse with PENDING retained; infra errors are retryable with PENDING retained; contention
returns `ErrAttemptInProgress` with PENDING retained; stale ownership refuses terminally
(caller mints fresh); cancel-first refuses; authorize-first reports in-flight;
ambiguous outcomes quarantine UNKNOWN without blind retry. Per-question results:

- Terminal invocation withdraws only its own effects: YES (withdraw predicate carries op +
  tenant; invocation binding verified before the branch).
- Approval refusal with mismatched binding: never withdraws (mismatch returns before any
  withdraw; verified order).
- Lazy expiry: treated as refusal (`approval-expired`), not withdrawal — correct (no
  irreversible decision invented).
- Mismatch destroying another owner's row: impossible (no withdraw on any mismatch path).
- Retry resurrecting CANCELLED: impossible — but only because **CANCELLED is currently
  unreachable** (see below).
- Post-cancel takeover authorizing: impossible (terminal invocation refuses before step 5).
- Refusal audited as provider attempt: NO — `effect-refused/<reason>` vs `effect-withdrawn`
  vs attempt records are distinct; no phantom attempt records are manufactured.
- Withdrawal audit pairing: **BROKEN — [P24-001](#p24-001) (HIGH).** The `effect-withdrawn`
  audit fires whenever the in-tx flip *executes*, but the flip always rolls back.
- Quarantined stays resumable: YES — quarantined refuses without withdrawing (verified
  branch), and no path terminalizes it.

The structural fact: **no code path can persist CANCELLED.** `withdrawIfPending` is the sole
writer and always rolls back; the M6 "eager PENDING→CANCELLED batch in cancel tx" from
contract §4.2 does not exist in any form (cancel performs zero effect writes — verified).
Cancel-first protection holds (terminal refuses), but terminality hygiene does not: PENDING
rows linger, `LiveEffects` reports them in-flight forever, and the audit asserts withdrawals
that never happened. Cancellation-first vs authorization-first *ordering* has no forced
deterministic test yet (outstanding §14 item) — POC-1 stages the cancel-first order
adversarially and proves the refusal (correct) alongside the lost withdrawal (defect).

## 10. C6 integration assessment (Priority 7)

The fourth Class A outcome verified end to end: handler errors matching
`effects.ErrAttemptInProgress` return `executor.ErrAttemptContention` *before* the completion
closure runs — the whole attempt transaction rolls back (no FAILED recorded, no business
writes committed, record untouched and live) — and `executeSync` maps it to terminal
`effect-contention` (HTTP 409) naming the same-key retry. The conflict is explicitly terminal
at the request boundary only; the invocation is never terminalized by it (verified: no
completion write on that path). No automatic retry exists (explicit caller retry); no
indefinite hold (prompt refusal, lease never awaited); no orphaned invocation (record stays
live) or operation (ledger row owned by the winner, adopted or settled-read on retry).

Retry-convergence by record state: pending/waiting records re-dispatch into a fresh claim and
adopt or observe the winner (sound); **running** records answer `interrupted`/new-key per
pre-existing replay semantics — so the conflict message's blanket "retry with the same
idempotency key" overclaims exactly as the P21 `execution-in-progress` comment did before its
P21-004 fix ([P24-010](#p24-010), LOW). For new-key retries, invocation binding correctly
refuses adoption of the old invocation's row (confused-deputy prevention working as designed);
convergence then requires the winner to settle (settled-read) or a fresh op ID — which
surfaces the undocumented tension between C6's "stable BusinessOpID" and invocation binding
for cross-key Class A recovery. No C6 fixture test exists (zero references to
`effect-contention`/`ErrAttemptContention` in any test) and no op-ID derivation guidance is
written — the mechanism is sound, the integration pattern is undemonstrated. Correctly filed
as outstanding Stage 3A work (C6 row: mechanism sound, integration unproven), not as a core
defect.

## 11. NOT_APPLIED retry review (Priority 8)

The contract permits the change (`ValidEffectTransitions` gains `NOT_APPLIED → ATTEMPTING`;
RECONCILING edges fully removed — constant retained for stored-data compat with an accurate
comment; `records_test` now asserts UNKNOWN→RECONCILING rejected). Implementation verified:
only `NOT_APPLIED` rows are eligible (exact-state predicate); the same `BusinessOpID` is
preserved (predicate key); lineage is *not* inherited (fresh framework uuid minted — the old
attempt is dead by definition of reconciliation); full revalidation (invocation, approval,
grants, switches, token, ownership) precedes the transition — reconcile shares `authorizeTx`
with all other modes; stale workers cannot resurrect (predicate requires current NOT_APPLIED
+ invocation match; concurrent losers classify to contention/superseded); no send precedes
the ATTEMPTING commit (canonical authorize-then-send); concurrent retries converge on one
winner (row-5 serialization + predicate); disagreement fails closed (`classifyLoss`).

The genuine risk is the one the prompt names: duplicating an *actually applied* operation
misclassified as NOT_APPLIED. That classification is made only by operator `Settle` (evidence-
cited, currently unwired — no production path produces NOT_APPLIED, so reconcile is
unreachable in production today). The authorization core cannot detect operator misjudgment;
it correctly bounds the blast radius to one fresh authorized attempt rather than a blind
resend. The reconciliation-evidence vs authorization-transaction distinction is preserved
(Settle decides the past; authorize decides the next attempt). The reconcile path is also the
trigger that exposes [P24-002](#p24-002): multi-epoch ATTEMPTING rows are new with this path,
and outcome settlement was never fenced for them.

## 12. Atomic Settle review (Priority 9)

Source architecture verified point-by-point: one predicated `UNKNOWN → outcome` UPDATE (only
CONFIRMED/FAILED/NOT_APPLIED accepted; operator string required; auditor must be wired);
state change + `effect-settled` audit commit atomically in one tx (verified statement order);
no RECONCILING in any production path (writer deleted, map edges removed, `adopt` default
rejects it); identical repeat idempotent with audit backfill iff missing (verified predicate);
conflicting repeat returns terminal `ErrAlreadySettled` (first decision stands); crash-before-
commit changes nothing (`TestFaultSettleOperatorCommit` green: UNKNOWN preserved, retry
converges); crash-after-commit leaves state+audit consistent by construction (same tx);
zero network calls inside the tx (verified statements); PENDING/ATTEMPTING explicitly refused
(not misreported as settled).

Gaps, honestly filed. (1) **Lock order: Settle takes 5 then 1** — the effect UPDATE precedes
`loadInvocationTx … FOR UPDATE` (used for audit attribution), inverting global order and
contradicting the §4.2 "5 effect only" row; a TA↔Settle deadlock is constructible (narrow:
stale-belief TA + mid-flight Settle on one invocation; self-healing via detector+retry).
[P24-003](#p24-003) (MEDIUM). The `FOR UPDATE` is gratuitous (identity columns immutable) —
a lock-free attribution read is the fix. (2) **Operator identity is a caller string**,
non-empty-checked only; no authentication or server derivation exists at the Settle boundary.
Moot today (zero production callers — Settle is reachable only from tests), but the wiring
work must authenticate before exposing it — recorded as a wiring constraint, not a current
finding. (3) The four §16.1 crash tests are 1-of-4 executed (operator-commit fault green);
lost-response-after-commit, idempotent-repeat-under-crash, and conflicting-second-settle
crash orderings are outstanding — listed in §17, not counted. (4) Cosmetic: concurrent
identical repeats can double-insert the backfill audit (count-then-insert, no predicate) —
narrow, audit-only, noted. (5) `Settle` on a row whose invocation is missing fails closed
(attribution load fails → rollback) — correct direction; operator-visible error is an infra-
shaped message for a deterministic condition (folded into [P24-011](#p24-011)).

## 13. Provider-path inventory and bypass analysis (Priority 10)

Independent non-test inventory of everything that can initiate provider work:

| Path | Entry | Authorization | Send | Verdict |
|---|---|---|---|---|
| `Perform` fresh claim | `attempt()` | `authorizeAttempt(Fresh)` (claim-winner, attempt match) | `sendAndRecord` post-commit | BOUND |
| `Perform` stale-PENDING adopt | `takeover()` | `authorizeAttempt(Takeover)` (age + invocation + fresh uuid) | post-commit | BOUND |
| `Perform` NOT_APPLIED | `reconcile()` (2 call sites, 1 function) | `authorizeAttempt(Reconcile)` (full revalidation, fresh uuid) | post-commit | BOUND |
| `Perform` settled/live/stale-ATTEMPTING | `adopt()` short-circuits | none — and none sends (read / `ErrAttemptInProgress` / quarantine) | none | NO SEND |
| Unknown capability | 3 sites | none — transitions FAILED (own-row only: reachable solely with self-created or checkOperation-bound rows) | none | NO SEND |
| `Channel.Send` | interface | n/a (callee only) | sole call site `sendAndRecord` | CALLEE |
| `mockcap` | sole prod channel | n/a | via `sendAndRecord` only | CALLEE |
| `faultwrap` | decorator | n/a | test-only wiring | TEST-ONLY |
| Probe workflows | `effectStep` → `Perform` | inherited | inherited | BOUND |
| Reconcile/retry/resume | via `Perform`/`reconcile` | same boundary | same boundary | BOUND |
| Direct adapter calls | — | — | none exist (exhaustive grep) | ABSENT |
| Alternate ATTEMPTING writers | — | all 3 `SET state='ATTEMPTING'` statements are the step-5 predicates | — | ABSENT |

ATTEMPTING-transition audit: `transition()` callers move only to CONFIRMED/FAILED/UNKNOWN
(outcome settlement — the [P24-002](#p24-002) fencing gap, not a bypass). No adapter decides
liveness/approval sufficiency (adapters receive post-authorization `Operation`s only). The
"probe workflow is the sole production Perform caller" claim is VERIFIED (exactly one
non-test `.Perform(` in the repo). A caller inventory alone would not suffice — the call
graph was walked from every entry above to the single `ch.Send` statement, and the existing
effects suite (all green) exercises fresh/takeover/contention/quarantine/crash through the
boundary. Bypass verdict: **no production bypass exists.** Two boundary-adjacent gaps (neither
a bypass today): unenforced tool-capability membership ([P24-008](#p24-008), LOW — one mock
channel; must close before the first real provider) and caller-minted fresh attempt IDs
([P24-007](#p24-007), LOW — vacuous-harmless; lease/claim-winner carry the safety).

## 14. Test-discrimination assessment

Classification of the relevant tests (executed on PG19.0 unless noted):

- **Discriminating red/green:** `TestP22AdoptionMarksPendingRunning` (store-layer P21-002 flip;
  red fails on status, green passes — executed green here; red claimed in evidence, snapshot
  `61603cba…` reported-only).
- **Green-only regression:** all pre-existing effects/runtime/probe/transport suites
  (mechanically rewired to authorize deps; assertions unchanged — verified by diff scan; no
  weakening found); `TestFaultSettleOperatorCommit` (§16.1 case 1 of 4); `TestSettleAndListUnknown`
  (updated basic Settle incl. conflict + bad-outcome legs); auth snapshot/guard/hash tests;
  `TestP23SnapshotPinning` (deterministic, no sleeps — sound).
- **Characterization:** the 4 P22 suites (pass both sides by design, honestly labeled);
  `TestKillMidSendAdoptsUnknown` (still passes through the new boundary — composition signal).
- **Source-only proof:** C1 (no advisory — read + grep, no lock-profile test); C2 authorize
  order (statement read; no forced-order test); C4 (grep + guard); P24-003/004/005/007/008/011
  (code + absence proofs).
- **Composition proof:** full suite green (every package incl. dbos 37.6s, probe 39.3s) +
  `-race` subset + existing contention/takeover/quarantine tests traversing authorize.
- **Not yet implemented (verified absent — zero test references):** authorization matrix
  (approval/grant/switch/token/stale/mismatch rows), authorize concurrency (both cancel orders,
  worker-vs-worker, lease vs authorize, revocation race, supersede vs authorize, settle vs
  ownership), authorize crash matrix (6 boundaries), C6 fixture, bypass tests, 3 of 4 Settle
  crash tests, grant CLI/healthz tests. The `authz_fixture_test.go` comment claims approval
  paths "are covered by the authorization matrix suite" — that suite does not exist; the
  comment overclaims present coverage.
- **Blocked/unverified:** none environmentally (PG19 available); historical red byte-identities
  (`01ad4fd1…`, `61603cba…`, `cf30d406…`, `f460c704…`, `34622b37…`) are reported-only,
  unverifiable without the frozen copies — same honest limitation as P21, non-blocking since
  discrimination that matters was re-proven by POC.

Battery results: `go vet` clean; **gofmt CHECK-FAIL (10 files)** and `checks.sh` CHECK-FAIL
([P24-006](#p24-006), LOW — process); full serial suite all-green (29 ok / skip-clean; DBOS
included via `TEST_DBOS_DATABASE_URL`); `-race` runtime/approvals/effects green; crash-worker
entrypoints skip by design. POC-1 and POC-2 (adversarial, `/tmp` copy, deterministic staging)
both FAIL as designed, proving [P24-001](#p24-001) and [P24-002](#p24-002) by execution
(0.07s/0.08s, no sleeps on success paths).

Evidence-register gap: Stage 3A has rows only through grant snapshot; authorizeAttempt,
ownership/adopt, C6, Settle, and bypass work are implemented but unrecorded. Acceptable Gate 1
input (source + tests reviewed directly), but the rows must be written before Gate 2.
The P21-002 "proven unreachable" claim overreaches in two legs ("trust immutable" — grants are
restart-mutable by design; legacy dangling rows unaddressed) — noted as evidence-accuracy
observation; the structural hardening carries C5 regardless, so no action beyond correcting
the claim.

## 15. Findings

### P24-001 — Terminal withdrawal never persists; audit falsely claims it

| Field | Detail |
|---|---|
| Finding ID | P24-001 |
| Severity | HIGH |
| Contract reference | M3 "Withdraw (cancel-won)" row (predicated `PENDING → CANCELLED`); §8 cancel-first; prompt23 §5 (durable cancellation withdraws); audit-truthfulness (X-13 family) |
| Source location | `effects/authorize.go` `authorizeTx` terminal branch + `withdrawIfPending`; `WithTenant` commit-on-nil-error; `auditRefusal` withdrawn labeling |
| Defect class | Transaction correctness + audit integrity |
| Root cause | The withdraw UPDATE runs inside a transaction whose function then returns the refusal *error*; `WithTenant` commits only on nil error, so the flip always rolls back — while `withdrawn=true` unconditionally drives an `effect-withdrawn` audit. No other code writes CANCELLED (verified writers inventory), so the state is unreachable. |
| Trigger | Exact sequence (POC-1, deterministic): PENDING-stale row + live invocation; Perform blocks at the locked pre-claim liveness read behind a row gate; cancel commits queued behind it; FIFO grants the waiter first (liveness passes live); authorize reads terminal, flips in-tx, refuses, rolls back |
| Failure mode | Row stays PENDING forever; audit records `effect-withdrawn`; `LiveEffects` reports the row in-flight indefinitely; every same-op retry re-refuses |
| Impact | Required lifecycle invariant violated (cancel-win withdrawal); false audit attribution on a security boundary; ledger litter with live-reading rows. No unauthorized send (fail-closed direction preserved — refusals hold) |
| Evidence | Source read (full path) + POC-1 execution: `row state after terminal refusal: PENDING (want CANCELLED)` with `effect-withdrawn audits = 1`, 0.07s, PG19 |
| Discriminating test | POC-1 (`TestP24POCWithdrawPersists`, `/tmp` review artifact — must be re-implemented as a committed regression, not copied) |
| Required correction | Smallest durable: decide-then-withdraw in two steps — authorizeTx returns the refusal *decision* without flipping on error paths; on terminal-invocation refusal, a follow-up predicated `PENDING → CANCELLED` tx (safe: terminal invocations never un-terminal, so the row cannot authorize meanwhile; idempotent on retry); audit `withdrawn` iff the second tx commits, else `refused`. Alternative (riskier): custom tx handling with commit-on-flagged-refusal. Either way add the committed regression + audit-pairing assertions |

### P24-002 — Stale worker outcome overwrites a newer attempt

| Field | Detail |
|---|---|
| Finding ID | P24-002 |
| Severity | HIGH |
| Contract reference | Prompt22 §8 ("must not allow a stale worker to overwrite a newer attempt's result"); M3 attempt/lease ownership; `attempt_id` lineage purpose |
| Source location | `effects/effects.go` `transition()` UPDATE predicate (state-only); all `sendAndRecord` outcome legs (CONFIRMED/FAILED/UNKNOWN) |
| Defect class | Fencing / ledger integrity |
| Root cause | Outcome settlement predicates on `(op, tenant, state)` but not `attempt_id`. Stage 3A's reconcile path introduces the first multi-epoch ATTEMPTING rows (NOT_APPLIED → ATTEMPTING with a fresh ID); a previous epoch's late outcome then matches the newer epoch's live row. Pre-existing single-epoch shape masked this; reconcile exposes it. |
| Trigger | Exact sequence (POC-2, deterministic): epoch A authorizes + blocks in Send; row aged past lease; touch quarantines to UNKNOWN; operator settles NOT_APPLIED; epoch C reconciles + authorizes + blocks; A's send released → A's CONFIRMED lands on C's row |
| Failure mode | Row CONFIRMED with A's stale receipt on C's attempt ID; C's real outcome subsequently lost (Superseded); ledger attributes the wrong attempt's evidence. FAILED/UNKNOWN late outcomes clobber identically |
| Impact | Ledger integrity violation: wrong receipts/outcomes on live attempts; silent loss of the true outcome. Currently triggerable only via operator Settle (unwired in prod — see scope note), so exposure opens the moment Settle wiring lands, an explicitly planned Stage 3A step. The reviewed core is unsound as composed |
| Evidence | Source read (predicate + all callers' `rec.AttemptID` availability) + POC-2 execution: `STALE OVERWRITE: epoch A's receipt landed on epoch C's attempt e717c687…`, 0.08s, PG19 |
| Discriminating test | POC-2 (`TestP24POCStaleSettleFenced`, `/tmp` artifact — re-implement as committed regression across all three outcome flavors) |
| Required correction | Smallest durable: add `AND attempt_id = $N` (from the in-hand `rec`) to `transition()`'s UPDATE; stale landings become zero-row Superseded (existing handling). Optionally add the same conjunct to `adoptStale` (already age-safe; uniformity hardening). Verify all `transition` callers carry true attempt IDs (verified: authorize-loaded, claim-loaded, adopt-loaded — all carry it). No migration (column exists) |

### P24-003 — Settle inverts lock order (5 → 1)

| Field | Detail |
|---|---|
| Finding ID | P24-003 |
| Severity | MEDIUM |
| Contract reference | C2 global order; §4.2 "Settle operator (M3)" row ("5 effect" only) |
| Source location | `effects/effects.go` `Settle` → effect UPDATE then `insertSettleAudit` → `loadInvocationTx … FOR UPDATE` |
| Defect class | Lock order (availability) |
| Root cause | Audit attribution reuses the lock-taking invocation loader after the effect row is already held, instead of a lock-free read (identity columns are immutable) |
| Trigger | TA holds inv-1 / wants eff-5 on a row S holds, while S holds eff-5 / wants inv-1: requires TA with a stale rows-belief (PENDING/NOT_APPLIED at decision, UNKNOWN when S commits first) overlapping a mid-flight operator Settle on the same invocation |
| Failure mode | Deadlock; PostgreSQL aborts a victim (retryable error); retry converges. No wrong outcome possible |
| Impact | Narrow, self-healing availability blip — but a binding-constraint (C2) and contract-row violation requiring owner disposition per Gate 1 rules |
| Evidence | Source read (statement order + lock modes); wait-for graph §4 (sole cycle); no execution (3-way + operator timing; code proof sufficient for MEDIUM) |
| Discriminating test | None yet — recommend a forced-order TA-vs-Settle deadlock-absence test (statement timeouts, both orders converge) alongside the fix |
| Required correction | Lock-free attribution read in `insertSettleAudit` (plain SELECT of immutable identity columns); re-verify the §4.2 row ("5 only") and re-run the Settle suite |

### P24-004 — M5/sequence trust boundary incomplete

| Field | Detail |
|---|---|
| Finding ID | P24-004 |
| Severity | MEDIUM |
| Contract reference | Owner condition §12.1 (framework-owned verifier); Decision A (probe fence at the boundary); M5/X-08 fence rows; §5 activation trigger |
| Source location | `effects/authorize.go` `SequenceVerifier` (interface only) + position-3 call site; `effects.Config.Sequences` (never set); `probe/classd/sequence.go` (no identity presented, no generation re-check) |
| Defect class | Authorization trust boundary (latent) |
| Root cause | Only the interface + call-site shape shipped: zero implementations, zero wiring, zero identity producers. Classification is presence-based (`intent.Sequence != nil`), so fencing is downgradable by omission at the mechanism level; the §5 trigger names tables + validation but no forcing mechanism |
| Trigger | Future: first sequence workload presents identity on some ops and omits it on (or via a bug in) others → unfenced sends. Present: superseded-but-live probe generation sends its old-generation op (no authorize-time or workflow-side generation check) — mock-only |
| Failure mode | Latent downgrade vector; present Decision-A gap for probe reference code; trigger as written does not close the omission hole |
| Impact | Zero consequential exploitability today (no sequence ops; presence refuses fail-closed); structural gap the owner explicitly flagged, in exactly the mechanism asked about. Requires disposition, not code, at Gate 1 |
| Evidence | Exhaustive greps (implementations: 0; `Sequences:` wirings: 0; `Intent.Sequence` producers: 0); probe workflow read (no re-check); trigger text read |
| Discriminating test | None applicable pre-implementation; on implementation: omission-downgrade refusal test + supersede-barrier test (the never-written X-08 test) |
| Required correction | Owner disposition: (a) amend the §5 trigger to require a forcing mechanism (tool-declared sequence requirement or op-ID subject parsing) with the tables; (b) decide whether the §12.1 probe verifier + probe `Sequence` presentation ships before any sequence workload or stays deferred with the mock-only rationale recorded; (c) write the X-08 barrier test when the fence lands. No code now |

### P24-005 — Gated+autonomous submit/authorize incoherence (regression)

| Field | Detail |
|---|---|
| Finding ID | P24-005 |
| Severity | MEDIUM |
| Contract reference | Prompt22 §7.2 (approval revalidation — silent on trust exception); choke gated branch (autonomous exemption); manifest `TrustModes` offers |
| Source location | `runtime/service.go` `submitChecked` (skips linkage when row trust autonomous) vs `effects/authorize.go` `authorizeTx` (requires linkage for all gated tools) |
| Defect class | Authorization completeness (over-refusal trap) |
| Root cause | The design never addresses the trust exception: submit-time admits autonomous gated invocations without linkage while effect-time unconditionally demands it, and no API can create linkage for a running record |
| Trigger | Invoke a gated tool (e.g. probe `ToolRun`: `Required: true`, `TrustModes: [Confirm, Autonomous]`) with autonomous trust → Allow → submit without linkage → workflow runs → every `Perform` refused `approval-missing`, permanently |
| Failure mode | Autonomous gated workflows run and fail at the first effect, unrecoverably. Pre-Stage-3A these effects sent (no effect-time approval check) — behavior regression in the fail-closed direction |
| Impact | Breaks a manifest-offered production configuration (the probe's gated tool). Loud (clear refusal errors), never unsafe — but incoherent, and untested (zero autonomous references in probe tests) |
| Evidence | Source reads (both branches + no-park-for-running); probe manifest read; coverage grep (0 hits); pre/post behavior comparison by code path |
| Discriminating test | None exists — the authorization matrix must include gated+autonomous whichever semantics the owner picks |
| Required correction | Owner semantic decision (this IS the disposition): (a) authorize skips linkage when row trust is autonomous (completes the existing float model; weakens gate-at-effect-time); (b) choke requires linkage for gated tools regardless of trust (fail fast at submit; recommended — coherent and fail-fast); or (c) manifest validation forbids Required+Autonomous pairs. Then implement + matrix-test the chosen semantics |

### P24-006 — Unformatted tree; checks gate red

| Field | Detail |
|---|---|
| Finding ID | P24-006 |
| Severity | LOW |
| Contract reference | Prompt23 §9 battery (`gofmt -l .`, `scripts/checks.sh`); evidence discipline |
| Source location | 10 files (`cmd/miniapp/main.go`, `effects/authorize.go`, `effects/concurrency_test.go`, 3× `p20_*_test.go`, `p22_p21002_test.go`, `service_test.go`, `x01x02_test.go`, `stack.go`) |
| Defect class | Process |
| Root cause | Implementation landed without running gofmt; `scripts/checks.sh` reports CHECK-FAIL |
| Trigger | `gofmt -l .` / `bash scripts/checks.sh` on the worktree |
| Failure mode | Red gate; noisy diffs; no behavioral effect |
| Impact | Process hygiene only — but Gate 1 evidence should be gate-clean; `go vet` is clean and all tests pass independently of formatting |
| Evidence | `gofmt -l .` output (10 files); `scripts/checks.sh` CHECK-FAIL transcript |
| Discriminating test | n/a — `gofmt -l .` must print nothing |
| Required correction | Run `gofmt -w` on the tree before the next evidence snapshot; confirm checks.sh green |

### P24-007 — Fresh attempt IDs are caller free text

| Field | Detail |
|---|---|
| Finding ID | P24-007 |
| Severity | LOW |
| Contract reference | Design §5 (uuids minted by framework code, never caller free text); sole prod caller passes hardcoded `"a1"` |
| Source location | `effects/effects.go` `Perform` → `claim` → `attempt` (no mint/validate); `probe/classd/sequence.go` `AttemptID: "a1"` |
| Defect class | Design compliance (defense in depth) |
| Root cause | The "mint-or-validate" placement was deferred to Stage 3 and never implemented |
| Trigger | Any caller chooses any fresh attempt ID (including empty-adjacent or colliding values) |
| Failure mode | None demonstrated: fresh matching is self-consistent for the claim winner; cross-worker safety rides lease-freshness + claim-winner exclusivity + invocation binding, all of which hold. Takeover/reconcile mint framework uuids correctly |
| Impact | Vacuous-harmless today; weakens the lineage story the design promises (attempt IDs as ownership epochs) |
| Evidence | Source reads (no mint/validate call in Perform/claim); probe hardcode; safety-carrier analysis in §8 attack 5 |
| Discriminating test | A mint/validate unit test once implemented (reject empty/foreign, mint on demand) |
| Required correction | Implement the deferred placement (mint in `claim` when empty + validate shape, or mint in `Perform` and thread through) or amend design §5 to record caller-supplied-with-match as the approved model with the carriers named here |

### P24-008 — Tool-capability membership unenforced

| Field | Detail |
|---|---|
| Finding ID | P24-008 |
| Severity | LOW |
| Contract reference | `ToolManifest.Capabilities` (declared, zero enforcement points); kill-switch capability level; least-privilege adapter selection |
| Source location | `effects/authorize.go` (uses `intent.Operation.CapabilityID` for switches + channel routing); `effects/policy.go` (same); no membership check anywhere |
| Defect class | Authorization scope (latent) |
| Root cause | The manifest declares per-tool capabilities but no production code ever compares the intent's capability against them |
| Trigger | Miniapp presents a registered-but-undeclared capability: capability-scoped kill switches for its own capability don't match; the send routes via the presented capability's adapter |
| Failure mode | Cross-capability adapter invocation; capability-switch evasion (payload still flows, via a different adapter — the operator's "capability X is off" intent holds for adapter X but the miniapp isn't confined to declared adapters) |
| Impact | Nil today (single mock channel: unknown capabilities fail, `mock` is everyone's); structural gap that must close before/with the first real provider, when adapters gain differing authority |
| Evidence | Zero-reference grep for `Capabilities` outside `manifest.go`; channel inventory (mock-only prod); kill-switch name construction reads |
| Discriminating test | Matrix row: undeclared-capability intent refused (once the check exists) |
| Required correction | Add the membership check at authorize (intent capability ∈ row-tool manifest capabilities; refuse otherwise) no later than the first real channel; record the trigger explicitly |

### P24-009 — Policy guard-test pattern gap

| Field | Detail |
|---|---|
| Finding ID | P24-009 |
| Severity | LOW |
| Contract reference | §4.1 mechanical enforcement ("forbids direct map writes"); design §6 grep test |
| Source location | `auth/policy_guard_test.go` (line matcher) |
| Defect class | Test-quality (guard coverage) |
| Root cause | Line-based matcher covers `.Keys[`/`.Grants[` subscript and `auth.Policy{` literals but not whole-map assignment (`policy.Grants = …` — bypasses setter AND skips hash recompute) or bare field reads |
| Trigger | A future `policy.Grants = x` outside `auth` passes the guard silently, racing readers and desyncing the hash |
| Failure mode | Silent discipline bypass; no current instance (verified zero by grep) |
| Impact | Guard is weaker than its charter claims; the covered patterns are genuinely enforced (incl. tests) |
| Evidence | Guard source read; bypass-pattern grep (zero current hits) |
| Discriminating test | Extend the guard (AST or assignment patterns) + re-run its negative control with a planted whole-map assignment |
| Required correction | One-test fix: match `\.Grants\s*=` / `\.Keys\s*=` (and bare field tokens) or migrate the guard to AST; keep the planted-violation negative control green |

### P24-010 — Contention message overclaims for running records

| Field | Detail |
|---|---|
| Finding ID | P24-010 |
| Severity | LOW |
| Contract reference | C6 explicit retry semantics; P21-004 precedent (same overclaim just fixed for `execution-in-progress`) |
| Source location | `runtime/service.go` `effect-contention` terminal message; `replay` running+Class A branch |
| Defect class | Contract/UX accuracy |
| Root cause | The message promises same-key convergence unconditionally, but same-key retry on a running record answers `interrupted`/new-key (pre-existing replay semantics) |
| Trigger | Ledger contention inside a first-claim (running-record) Class A dispatch → conflict says "retry with the same idempotency key" → retry yields `interrupted` |
| Failure mode | Misleading instruction (agents may loop same-key retries that can never converge on running records); underlying behavior is sound (interrupted → new-key → settled-read when the winner completes; binding refusal otherwise, by design) |
| Impact | Message-only; mirrors the exact wart P21-004 fixed next door — likely copy-forward |
| Evidence | Message text + replay branch reads; convergence analysis in §10 |
| Discriminating test | C6 fixture assertion on the running-record message (with the fixture itself, outstanding) |
| Required correction | Qualify the message by record state (same sentence shape as the fixed InProgress comment), or route contention-retry messaging through one shared helper |

### P24-011 — Missing-invocation surfaces as infra noise

| Field | Detail |
|---|---|
| Finding ID | P24-011 |
| Severity | LOW |
| Contract reference | Fail-closed binding (§7.1); error taxonomy (§7.6 — typed, actionable results) |
| Source location | `authorizeTx` → `loadInvocationTx` raw `pgx.ErrNoRows` propagation; `Settle` attribution load (same shape) |
| Defect class | Error classification |
| Root cause | No-rows (deterministic: forged/garbage invocation ID, or op bound to a phantom invocation) returns as an untyped infra-shaped error instead of a terminal binding refusal |
| Trigger | `authorizeAttempt` (any mode) or `Settle` on an op whose invocation ID has no row |
| Failure mode | Callers see retryable-shaped noise for a permanent condition (fail-closed direction — workflows fail rather than loop at runtime level, but the signal is wrong); unaudited (not a refusal); Settle on phantom-bound rows fails opaquely |
| Impact | Observability + taxonomy; no safety impact (nothing authorizes, nothing settles) |
| Evidence | Source reads (raw propagation; `refusalReason` excludes it); no-rows is untestable via public Perform (pre-gates), hence latent |
| Discriminating test | Unit test: direct authorize on missing invocation yields terminal binding refusal + audit; Settle ditto |
| Required correction | Map `isNoRows` from the invocation load to a terminal `RefusalBindingMismatch` (authorize) / terminal settle error, with audit; keep genuine connection errors retryable |

## 16. Required correction and bounded owner decision

Minimum to clear Gate 1 (structural classes only — no broad rewrite):

1. **Fix P24-001** (withdraw commit structure + truthful audit + committed regression incl. the
   POC-1 staging shape). Re-review: POC-1 rewritten as a committed test must pass (row
   CANCELLED + exactly one truthful `effect-withdrawn`).
2. **Fix P24-002** (attempt predicate in `transition()` + committed regression across
   CONFIRMED/FAILED/UNKNOWN flavors). Re-review: POC-2 shape must pass (C's row/attempt/
   receipt all C-owned; A's late outcome Superseded).
3. **Fix P24-003** (lock-free Settle attribution read). Re-review: statement-level re-read +
   Settle suite green.
4. **Disposition P24-004** (sequence trigger amendment + probe-verifier decision + X-08 test
   commitment). No code required at Gate 1; the amended trigger text is the deliverable.
5. **Disposition P24-005** (pick (a)/(b)/(c) semantics for gated+autonomous). Code + matrix
   row for the chosen semantics may ride the same targeted re-review or the Stage 3A matrix
   at owner option — but the decision must precede the matrix (tests encode it).
6. **Fix P24-006..P24-011** opportunistically (all one-liners except P24-011's small mapping);
   none blocks re-review but P24-006 (`gofmt` + checks green) is required before any new
   evidence snapshot.

Targeted re-review scope: the two HIGH fixes + their committed regressions, the Settle lock
read, the two disposition records, and a re-run of the affected suites. The full Stage 3A
matrix remains outstanding work regardless (it cannot substitute for these fixes, and these
fixes do not substitute for it).

## 17. Explicit remaining Stage 3A work

Verified outstanding (absent from tree, not merely unreviewed): full authorization matrix
(§10.1 rows); both forced cancel-race orders + worker/lease/revocation/supersede/settlement
races; 6-boundary crash matrix; C6 fixture + op-ID guidance; bypass tests per path; 3 of 4
Settle crash tests; grant hash consumers (`/healthz`, `verify-revocation` CLI) + tooling
tests; Settle production wiring (+ operator authentication per §12); sequence-verifier
implementation/wiring per disposition; Stage 3A evidence rows for authorize/ownership/C6/
Settle/bypass; final full-suite + CI parity snapshot. The four Settle crash tests are 1/4;
the "authorization matrix suite" is referenced by a fixture comment but does not exist.

## 18. Accepted residuals and deferred work

Retained (all re-examined, boundaries intact): O-01 submit-time race (effect-time protection
holds; terminality pending P24-001's fix); 32-bit entity-hash contention; ledger boundary
(non-transactional effects out of scope); X-12 gap; T5 characterization; FIFO staging
assumption; aggregate LOW-1 counts; 2-minute lease delay (now load-bearing for C6 retry
wording — see §10); INFO-2 raw-SQL bypass (C4 conditional correctly untriggered); miniapp
direct-SQL discipline; M5 probe-scope deferral (pending P24-004's trigger amendment);
X-10 quarantine (resumability re-verified: quarantined refuses without withdrawing);
prompt21 CI scratch. Newly accepted through this review: best-effort refusal audits (denial-
side observability only; grants always evidenced); app-clock lease predicates (liveness-only
skew); settled-read cross-invocation disclosure (pre-existing, payload-match-gated);
unknown-capability FAILED without authorize (own-row only, no send); backfill double-audit
race (cosmetic, concurrent-identical-repeat only); receipt loss on Superseded (UNKNOWN stays
honest; reconcile via provider evidence).

