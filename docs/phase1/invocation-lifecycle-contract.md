# Invocation Lifecycle Contract (Prompt 15.2 P1 — Normative)

| Field | Value |
|---|---|
| Status | NORMATIVE-DRAFT (amended per `prompt15.2_p1_review.md` directive) — P2 gated until the owner accepts these amendments |
| Governing prompt | `docs/prompt15.2.md` §§3, 7 (this document is its §7 / §17.1 deliverable) |
| Tree verified | HEAD `5cf9627` (main), worktree digest `ba3592701e696c59746ac923895edaee7b203cc660c1574543560167b4dfe091` (254 files, manifest `docs/evidence/digests/p15-p0-2026-11-23.manifest.txt`); mid-P1 owner docs-only drift recorded as digest `32c6981263d83f9256cbb7b7fb48a18003fbd3d50100ede2ea5218b8f71de033` (255 files, manifest `docs/evidence/digests/p15-p1-2026-11-23.manifest.txt`) — zero source files affected, verification stands |
| SDK verified | `github.com/dbos-inc/dbos-transact-golang` v1.9.0, source digest `308583f25714c7886144da34a8720afc7aa5addf81ca4c53339d2873138eec39` (260 files, manifest `docs/evidence/digests/p15-sdk-dbos-1.9.0.manifest.txt`); method: sorted `sha256sum` manifest, same as tree digest |
| Self-review disclosure | Authored by the P2 implementer. This is a discipline check, not an independent review. P2 requires explicit owner acceptance of this limitation or an independent review (P1 exit gate). |

Section references to `docs/prompt15.2.md` were verified against the actual
file. No implementation code was changed to produce this document.

## 1. Root-cause assessment (15.2 §3 — ratified with refinements)

Each hypothesis was verified against the tree above and pinned SDK source.
Verdict: **RATIFY RC-1a–d and RC-2**, with three refinements (R1–R3) below.
Nothing was refuted.

### RC-1a — Non-predicated writes (RATIFIED)

- `runtime/store.go:126-138` (`invocations.update`): unconditional
  `SET status, run_id ... WHERE id AND tenant_id`, no expected-state
  predicate, no affected-row check. Both `submitRecorded`
  (`runtime/service.go:335-347`) and `confirmSubmit` (`:354-369`) mint a
  fresh run ID when the record shows empty, so two concurrent same-key
  submits mint distinct IDs and the last `update` wins (X-03). The loser's
  workflow is live, sending, and unaddressable.
- `approvals/approvals.go:104-140` (`Decide`): `getTx` reads with no row
  lock (`:173-178`); the decision `UPDATE ... WHERE id` (`:137-139`) and
  the expiry-marking `UPDATE` (`:120`) carry no `status='pending'`
  predicate. Concurrent approve/reject, requester/approver, and
  Decide-vs-`CancelPending` interleave silently (X-04). (`CancelPending`
  itself, `:154-164`, is already predicated with an affected-row check —
  the conforming pattern.)

### RC-1b — Check-then-act across boundaries (RATIFIED)

- `runtime/lifecycle.go:230-244` (`checkLive`): `SELECT ... FOR UPDATE`
  inside one `WithTenant` transaction; the lock releases at commit.
  `effects.Service.Perform` (`effects/effects.go:142-182`) then runs
  policy check (`:168`) and claim `INSERT` (`:171`, no liveness
  predicate) in later transactions, the `PENDING→ATTEMPTING` transition
  in another, and the provider send outside any transaction. A cancel
  committing in the gap does not stop the new attempt (X-05). The
  "linearizes" comment (`lifecycle.go:221-225`) is false and must be
  corrected, not relocated.
- The X-03 mint window is the same shape (guard: claim; act: mint +
  record in separate scopes).

### RC-1c — Authorization by path ordering (RATIFIED)

- `confirmSubmit` (`runtime/service.go:354-369`) submits any `Running`
  Class D record with `ApprovalID: req.approvalID`, and a retry's fresh
  `InvokeRequest` never sets that field. `replay` (`:186-190`) routes
  `Running` Class D records there with no approval check. All three
  triggers verified: `Approver.Submit` failure (`:210-218`) and park
  `update` failure (`:219-221`) leave the claimed record `Running`
  (claim inserts `'running'`, `runtime/store.go:57`); crash between
  claim commit and park does the same. The submitted workflow gets
  `ApprovalID: ""` and `probe/classd/sequence.go:156` skips the wait
  (X-01).

### RC-1d — Failure branches are not states (RATIFIED)

- `pollApproval` (`runtime/service.go:243-287`) never submits: a parked
  record whose submit failed (run ID persisted, workflow never
  submitted) or whose park audit failed (no run ID) waits forever; a
  later grant fails forever at `Async.Status`; denial loops
  `settleDenied`/`cancelRun` when a run ID names nothing submitted
  (X-02).
- `Executor.Signal` (`executor/dbos/executor.go:291-307`) returns nil on
  the first family because SDK `Send` never errors for unknown
  destinations (verified: `SysDB.Send` is a blind `INSERT INTO
  notifications`, v1.9.0 `dbos/internal/sysdb/system_database.go:5037`,
  and notifications carry "no foreign key on workflow_status", `:2570`)
  (X-06).
- `DecideApproval` audits only after a successful signal
  (`runtime/lifecycle.go:213` then `:217`): committed-but-unsignalled
  decisions are unaudited (X-11).
- `run-settled` emits only from `Status` (`runtime/lifecycle.go:43-48`)
  (X-12). The SDK offers no workflow-completion callback (verified by
  source search), so settlement discovery must be framework-owned.
- Refused claims return errors and "insert nothing"
  (`effects/effects.go:165-170` and the `Perform` contract, `:141`)
  (X-13).

### RC-2 — Evidence built to demonstrate, not falsify (RATIFIED)

- X-07: `INSERT ... ON CONFLICT (op_id) DO NOTHING`
  (`testkit/mockcap/mockcap.go:132-136`) plus the false comment (`:76-77`).
- X-09: `TestVersionOverlap` runs identical code with both workers alive;
  step dedup yields identical markers with or without affinity.
- X-17: kill/restart parents spawn workers before checking DB env
  (`executor/dbos/dbosexec_test.go:778-806`, same shape in
  `probe/classd/probe_test.go:587+`); CI (`.github/workflows/ci.yml`)
  provisions no PostgreSQL and runs bare `go test ./...`.
- X-18: prompt13 rows cite an undigested tree; the readiness P0 full-list
  entry is absent (verified by grep).
- All HIGH findings sit on interleavings (concurrent submit, concurrent
  Decide, cancel-during-claim, sequence-claim race) or failure branches
  (park failure, submit-after-park failure, lost signals) the suite
  never covers.

### Refinements (P1 additions, not refutations)

- **R1 — X-03/X-22 interplay.** Predicated run-ID adoption plus X-22's
  `UNIQUE (run_id) WHERE run_id <> ''` conflict if a losing invocation
  stores the winner's run ID in its own `run_id` column (two rows, one
  ID). Design (§§5, 9): `run_id` stays the row's own submitted run
  (unique when nonempty); a new nullable `attached_run_id` column
  (migration 000006) carries an adopted winner. `getByRun` resolves the
  owner row; attachments are explicit. X-08 true attach reuses this
  column — one mechanism for both findings.
- **R2 — X-15 is role-dependent.** `REVOKE` binds only non-superuser
  roles, but local development connects as the superuser `miniapp`.
  The migration ships branch-local hardening (privileged setter +
  `ENABLE ALWAYS` flip-audit trigger — accurate wording per CR-08:
  an owner/superuser can still disable it); role separation is
  proven by tests over genuine restricted connections and required
  of production deployment as a recorded residual (§12). The
  serving process never holds the control DSN (CR-08).
- **R3 — Claim starts gated tools as `Running`.** The initial `'running'`
  status (store.go:57) precedes the approval decision. P2 moves the
  authorize decision before `claim` and inserts gated tools as
  `pending` (`contract.RunPending`, currently unused for invocations);
  `replay` handles `pending` identically to first-claim dispatch.

## 2. Transition-integrity rules (normative)

1. **Predicated single-writer transitions.** Every contested lifecycle
   mutation proves its expected prior state with a conditional `UPDATE`
   (or row-lock protocol) plus an affected-row check. On zero rows the
   loser reloads and reconciles or returns a defined conflict — never
   an unconditional overwrite. `run_id` is adopted, never overwritten.
2. **Guard-and-act consistency.** A guard stays authoritative through the
   durable operation it protects: guard and act share one transaction
   or a proven lock protocol. Cancellation and effect claim serialize
   against each other at the claim boundary.
3. **Authorization at the execution boundary.** All executor submissions
   pass one choke point (§5) that resolves the approval requirement
   from persisted state. Two permissions are distinguished
   (owner-approved WS-A Rule 3 amendment, §16):

   > A gated workflow may be submitted with a valid pending approval
   > linkage only to enter its durable approval-wait path. It may
   > proceed to consequential business execution only after the
   > framework verifies a persisted granted decision with matching
   > authority and intent. The effect-authorization boundary
   > independently revalidates that decision. Missing, denied,
   > expired, cancelled, or mismatched approval linkages fail closed.

   | Permission | Required condition | Permitted behavior |
   |---|---|---|
   | Workflow-start | Valid persisted pending approval linkage | Submit solely to enter the durable approval wait |
   | Business-execution | Persisted, verified granted decision | Authorize consequential effects after revalidation |

   A pending approval is never a business-execution authorization.
   The linkage must bind tenant, invocation, tool, and intent; a
   signal, cached workflow value, or caller-supplied approval ID
   alone is insufficient. No replay, retry, poll, or alternate
   caller bypasses the choke.
4. **No unreachable operational states.** Every nonterminal state has a
   defined convergence path or an explicit operator-actionable
   exception. Every retry re-enters the machine, reads persisted state,
   and dispatches on that state; a retry flag never decides whether
   authorization or lifecycle requirements apply.
5. **Truthful lifecycle and audit reporting.** Never report success for a
   nonexistent target or unestablished state. Audit decisions at commit;
   settle independently of `Status` polling; record refusals as
   attributable events distinct from provider attempts.
6. **Falsifiable evidence.** Every invariant claim ships with a
   discriminating test (red/green for HIGH + X-07); mocks measure every
   operation; affinity tests distinguish executors; every important
   result binds to exact source digests.

Subsystem-specific transition operations implement these rules; no
universal `transition()` is imposed across differing state, transaction,
or concurrency semantics (15.2 §7.3). Every durable lifecycle mutation
in §3's machines routes through its documented operation; any exception
needs a written justification and, where relevant, a test.

## 3. State machines and ownership (Gate 2)

Six durable machines plus four operational disciplines. Every X-01–X-24
finding and every F-01–F-14 residual has exactly one primary owner.

| # | Machine / discipline | States | Owner component |
|---|---|---|---|
| M1 | Invocation lifecycle | `pending, running, waiting_approval, waiting_event, succeeded, failed, cancelled` (`contract.RunStatus`; `quarantined` unused by invocations) | `runtime` service/store |
| M2 | Approval | `pending, decided, expired, cancelled` (rows in `app.approvals`) | `approvals` + `runtime` bridge |
| M3 | Effect / attempt | `PENDING, ATTEMPTING, CONFIRMED, FAILED, UNKNOWN, RECONCILING, NOT_APPLIED, CANCELLED` (`contract.ValidEffectTransitions`; `CANCELLED` = withdrawn before attempt authorization, wired per §10/X-23 reassessment — never attempted, never retried) | `effects` |
| M4 | Workflow / run identity | run ownership (`run_id` / `attached_run_id`), signal delivery (verified / accepted / processed, §7/X-06), foreign-version visibility classes (§7/X-10) | `executor/dbos` + `runtime` store |
| M5 | Sequence / attachment | probe generation claim (`active, superseded, done`), attach linkage — generation FENCING deferred per P24-004 (probe-scoped, mock-only; activation trigger in §7/M5) | `probe/classd` + M1 attach columns |
| M6 | Cancellation | invocation cancel intent, approval settlement, executor stop, in-flight report | `runtime` + `effects.LiveEffects` |
| D1 | Control-plane authorization | kill-switch `engaged` transitions, operator authentication | migration 000006 + `cmd/miniapp switch` |
| D2 | CI enforcement | skip-fast gates, provisioned DB service, sanctioned invocation | `.github/workflows/ci.yml` + testkit |
| D3 | Analyzer enforcement | seven rule families, fixture-pinned boundary | `analyzer/determinism` + `cmd/determinism-lint` |
| D4 | Evidence provenance | digest-per-run, register entries, matrix updates | `docs/evidence` + P2 scripts |

Finding → owner map (primary test in §10):

| Finding | Owner | Finding | Owner | Finding | Owner |
|---|---|---|---|---|---|
| X-01 | M1/M2 | X-09 | M4 | X-17 | D2 |
| X-02 | M1/M2 | X-10 | M4 | X-18 | D4 |
| X-03 | M1/M4 | X-11 | M2 (+sweeper §8) | X-19 | M1/M6* |
| X-04 | M2 | X-12 | M1 (+sweeper §8) | X-20 | preflight (D3-adjacent) |
| X-05 | M3/M6 | X-13 | M3 | X-21 | D2/D3 (checks.sh) |
| X-06 | M4 | X-14 | M4/cmd/stack | X-22 | M1/M4 |
| X-07 | testkit/mockcap (D4 instrument) | X-15 | D1 | X-23 | M1/M3/M5 (per item) |
| X-08 | M5/M1 | X-16 | D3 | X-24a–f | M1/M4/testkit (per item) |

\* X-19's canonical encoding helper lives in `runtime` and is shared by
entity derivation (M1), effect ledger scoping (M3), advisory-lock keys
(M6/Class A), and switch-name scoping (D1).

## 4. Lock discipline and global lock order

Never hold a PostgreSQL transaction, row lock, or advisory transaction
lock across an external provider send, a DBOS submission, a long wait,
or a signal wait (15.2 §7.5). All guard-and-act scopes below hold locks
only across local SQL.

Global acquisition order, PG row locks (deadlock-freedom; extended
per CR-04 — sequence, policy-control, and grant rows added):

1. Invocation row (`app.invocations`, `FOR UPDATE` or predicated
   `UPDATE`).
2. Approval row (`app.approvals`, `FOR UPDATE` / `FOR SHARE` or
   predicated `UPDATE`).
3. Sequence/generation row (`probe_sequences`, `FOR SHARE` for
   fence reads; supersede `UPDATE` takes the write lock).
4. Policy-control rows (`app.kill_switches` matched rows `FOR
   SHARE`; grant content pinned via the §4.1 Go `RLock`, acquired
   before the PG transaction begins).
5. Effect row (`app.effect_records`, predicated `UPDATE` /
   conditional `INSERT`).
6. Advisory entity locks are acquired outermost-first, before
   acquiring transactional row or table locks. An advisory lock
   must never be acquired while holding those transactional
   locks. Once acquired, it may remain held while the protected
   operation obtains its subsequent database locks, subject to
   the established release and failure semantics. (Owner-approved
   WS-B correction; the previous "leaf" wording was reversed —
   see §16. Class A holds its entity lock across handler
   `Perform` calls by design; the no-cycle proof below covers
   that nesting.)

No-cycle proof (from the actual call graph): (a) exactly one
production site acquires advisory locks —
`executor/classa/classa.go` `Execute` (claim call, then entity call — P21-004: was `:41`, refreshed), always as the first lock operations
in a fresh transaction (verified: `LockEntity` precedes all
handler work; `executeSync`/`dispatch` hold no PG locks across
the call); Class B/C/D, runtime, approvals, effects, and sweeper
paths never acquire advisory locks (exhaustive grep). (b) Hence
no path acquires an advisory lock while holding PG row/table
locks. (c) Advisory waiters hold no PG locks (they block before
their handler runs), so a holder — waiting only on PG locks
ordered by §4.2 — can never wait transitively on its own
waiters. Same-key serialization resolves as holders commit.
(d) Nested `Perform` calls inside a Class A handler acquire PG
locks 1→5 while the advisory lock is held; with (a)–(c) this
adds no cycle. QED.

(MEDIUM-2 amendment: Class A execution now takes a second
advisory lock — the invocation execution claim
(`TryClaimExecution`, 64-bit `exec-claim:`-namespaced
`pg_try_advisory_xact_lock`) — before the entity lock, inside
the same transaction that runs the handler and commits
completion. The proof extends unchanged in structure: exactly
one code site acquires both, always in the fixed claim→entity
order, still with no row locks held; the claim is non-blocking
(try), so claim contention never waits at all (losers get a
terminal `execution-in-progress` conflict); entity
serialization keeps part (c). No reverse path exists.
Handler-error savepoint rollback does not release the
transactional claim (verified live on PG19).)

Consequences: the authorization transaction takes 1→2→3→4→5 in
order; `Cancel` takes 1, then settles approvals (2) and withdraws
`PENDING` effects (5) — never 5 before 1; `Settle`, takeover, and
adopt touch effect rows only (5) plus the authorization revalidation
path of §7/M3 where they authorize a send; the X-04 `Decide`
transaction locks exactly one approval row (2). The full multi-row
table is §4.2. No path acquires these out of order, so no lock
cycle exists. P2 proves it with concurrent
cancel-during-claim/takeover/settle/Park/expiry tests run to
completion (a deadlock fails the test via statement timeout, not
via a hang — tests set `statement_timeout`).

### 4.1 Grant authorization protocol (amendment 5)

Grants are file-loaded, process-lifetime configuration
(`auth.Policy` maps, loaded once by `loadPolicy`; no reload path
exists; only tests mutate them). They cannot join the PostgreSQL
transaction, so the trusted mechanism is a pinned snapshot:

1. `auth.Policy` gains an `RWMutex` with guarded accessors;
   `GrantAllows` and credential resolution take `RLock`. Direct map
   writes outside the package are forbidden (P2 migrates the one
   test mutator to a setter API and adds a grep test).
2. The effect-authorization transaction holds `RLock` across its
   authorization read set (map content pinned; short and local-only
   — a Go mutex, not a PG lock, so §4's no-hold rule is unaffected).
   Revocation (setter `Lock`) fully precedes or follows the claim;
   no in-process revocation race remains. No cycle exists: the
   setter takes no PG locks, so no path orders PG-then-Go against
   Go-then-PG.
3. Production revocation lifecycle: config change + process restart
   on all replicas. In-flight transactions complete under their
   start-of-tx grants (bounded by local-tx duration).

Explicit named residual (not a parenthetical): no hot-reload;
cross-replica coherence is a deployment property (rolling-restart
window); a future DB-backed grant source would join the
authorization transaction — recorded, out of scope. Tests: §H-7
wait→revoke→refuse E2E via the setter API, plus a revocation-racing-
authorization test asserting the defined order.

### 4.2 Multi-row transaction lock-order table (CR-02)

| Transaction | Locks in order | Notes |
|---|---|---|
| Park (M1) | 1 invocation (`FOR UPDATE`) → 2 approval (`INSERT`) | §7/M1; audit is lock-free append |
| Choke submit / adoptRunID (M1) | 1 invocation | single-row predicated update; adoption on `pending` atomically marks `running` in the same UPDATE (P21-002: pending ⟹ unsubmitted, so liveness predicates need no pending-with-run case) |
| Decide (M2) | 2 approval (`FOR UPDATE`) | single-row + predicated update |
| CancelPending (M2/M6) | 2 approval (predicated `UPDATE`) | |
| Cancel intent + settle (M6) | 1 invocation → 2 approval → 5 effect (`PENDING→CANCELLED` batch) | one tx (X-05 future state; current implementation is correctly multi-tx); executor stop after commit |
| Effect authorization (M3) | §4.1 `RLock` → 1 → 2 → 3 → 4 → 5 | send strictly after commit |
| Class A attempt (M1, P21-003) | advisory claim → advisory entity → 1 invocation (predicated completion) | advisories outermost in fixed claim→entity order at the single Class A site; handler direct-SQL + completion row lock nest inside; no advisory wait (try-claim) |
| Sweeper S1 redelivery (M2) | 2 approval (read) | signal after commit; no invocation lock held |
| Sweeper S2 expiry (M2+M1) | 1 invocation → 2 approval, one tx (discovery reads lock-free first) | §7/M2; signal + settle after commit |
| Sweeper S3 settlement (M1) | 1 invocation (predicated `settleTerminal`) | status read precedes, lock-free |
| Supersede (M5) | 3 sequence (`UPDATE`) | fence enforced at M3 authorization, not here |
| Settle operator (M3) | 5 effect ONLY (single predicated `UPDATE`, atomic — directive §3.3; attribution via lock-free immutable-identity read, never `FOR UPDATE` — P24-003; verified: zero `FOR UPDATE` in `effects.go`, no FKs/triggers) | + idempotent `effect-settled` audit |
| Sequence claim (M5) | 3 sequence only (`UPDATE` supersede + partial-unique `INSERT`) | one step-tx; never touches 1/2/5 |
| True attachment (M5+M1) | 3 sequence (claim-step tx, released) THEN 1 invocation (attach tx) | separate sequential txs; generation read released before attach write |
| Takeover / adopt (M3) | 5 effect only per tx (predicated); re-authorization opens a FRESH tx at §4 order (never holds 5 across it) | separate txs; a takeover never retains an effect-row lock while beginning invocation/sequence authorization |
| Retry authorization (M3) | full §4 order (same as Authorize-attempt) | every retry is a new authorization, never a lock inheritance |
| Effect→invocation reads (report paths) | lock-free reads only (`LiveEffects`, status resolution) | reporting never acquires 1/2/5; mutations always re-lock in order |

Effect/cancel ordering (Rule 2; amended per CR-03/R-01): `PENDING`
is a provisional reservation — not authorized to call the provider
and never reported as a provider call. `ATTEMPTING` is the durable
logical point of no return. The `PENDING → ATTEMPTING` transition
runs in one local transaction that, in global order: locks the
invocation row (`FOR UPDATE`), revalidates liveness, approval
linkage/authority (approval row `FOR SHARE`), grants (pinned-snapshot
protocol, §4.1), token (on the locked invocation row), kill switches
(matched rows `FOR SHARE`), eligibility and generation ownership
(sequence row `FOR SHARE` where sequence-bound, §7/M5), and expected
effect state/owner/lease — then commits before the external send.
Cancellation serializes against that transition: cancel-first → the
transition fails and no attempt is authorized (the `PENDING` record
moves to terminal `CANCELLED`, never attempted); `ATTEMPTING`-first
→ the attempt is in flight and reported as such. In-flight attempts
settle from evidence; stale workers cannot send or overwrite a newer
owner; failed authorizations create no `ATTEMPTING` row. R2
preserved: no duplicate confirmed business effects, no silently
lost operations, no blind retries of unknown outcomes. No claim of
exactly-once provider delivery.

## 5. Submission choke point (Rule 3)

One function, `submitChecked` (name fixed in P2; semantics fixed
here), is the sole path to `Async.Submit`. Signature shape:
`(ctx, req, binding, tool, rec) → (handle, error)`. It:

1. Loads the current invocation row (re-entry, §6).
2. Resolves the tool's approval requirement from the admitted manifest
   (never from caller flags).
3. If gated: requires a LIVE persisted approval linkage
   (`ByInvocation`): `pending` (workflow-start permission: submit
   solely to enter the durable approval wait), or
   `decided/approve` with matching intent hash, authority, tenant,
   and unexpired (business-execution permission) — owner-approved
   WS-A Rule 3 amendment (§16). A pending linkage never
   authorizes consequential execution; the effect-authorization
   boundary independently revalidates the granted decision inside
   the WS-B `PENDING → ATTEMPTING` transaction (specified here,
   implemented in WS-B): approval exists and is `decided/approve`,
   belongs to the correct tenant and invocation, tool and intent
   hash match the persisted invocation, approver authority is
   valid, and the approval is neither expired nor cancelled — with
   all other authorization conditions still satisfied, and no
   cached decision or workflow-local state substituting for the
   in-transaction verification. Missing linkage →
   `errApprovalRequired` (caller re-enters approval handling:
   re-park under the atomic Park row, §7/M1; a legitimate dangling
   pending row from a legacy/pre-existing record is adopted
   defensively — atomic Park cannot produce one).
   Decided-against, expired, or mismatched linkage → terminal
   refuse, never submit blind. Live linkage → submit bound to it
   (linkage ID in the run request).
4. If ungated: submit.
5. Mints a run ID only when both `run_id` and `attached_run_id` are
   empty, via predicated adoption (§7/M1): `UPDATE ... WHERE run_id =
   '' AND attached_run_id IS NULL`; on zero rows, reload and adopt
   the winner (attach, never overwrite).
6. Gated tools with an empty approval linkage fail closed inside the
   workflow too: `Start`/`translate` rejects a gated tool whose
   request carries no approval ID, so no executor-side path can run
   unapproved work even if the service check is bypassed by a future
   caller.

P1 audit result (non-test callers, verified by grep): `Async.Submit`
has exactly two call sites, `submitRecorded`
(`runtime/service.go:341`) and `confirmSubmit` (`:363`); both route
through the choke point in P2 (one may become a thin wrapper).
`dbos.RunWorkflow` has exactly two non-test call sites, the probe's
`Start` bindings (`probe/classd/sequence.go:89,98`), reachable only
via `Executor.Submit`, which requires a runtime-minted run ID and is
reachable only via `Async.Submit`. No `Enqueue` callers exist outside
tests. No bypass found.

Proof strategy (P2): a test asserting the non-test caller set of
`Async.Submit` and `dbos.RunWorkflow` equals the audited set (grep
over the tree, failing on new call sites), plus the X-01 rejection
test (gated invocation with no valid linkage is refused, never
submitted). Bypass contingency (per the consolidated approval): a
bypass found during P2 is fixed in place when the correction stays
within this contract; escalation to the owner happens only if the
fix needs new architecture or policy.

## 6. Retry-as-re-entry (Rule 4)

Every retryable path loads persisted state and dispatches on observed
state. Required handlers:

| Path | States handled |
|---|---|
| Invoke submit/replay | `pending` (dispatch as first claim), `running`/`waiting_event` (choke point re-entry), `waiting_approval` (approval poll), terminal (verbatim replay / terminal error) |
| Approval create/poll | missing row (create), `pending` (wait), decided (grant/deny/expire convergence incl. never-submitted runs), `cancelled` |
| Submit-if-missing recovery | parked-without-run (submit via choke with stored linkage, then poll) |
| Signal redelivery | unknown run (error), unavailable family (explicit error), live run (verified send → `accepted`), terminal run (distinct `already-terminal` no-op, no state change), quarantined run (`run-quarantined`, resume first) |
| Effect claim/settle | no row (provisional claim), `PENDING` (authorize-or-withdraw), `ATTEMPTING` (adopt/quarantine by lease; send-then-settle), settled incl. `CANCELLED` (read), `NOT_APPLIED` (retryable re-attempt via Authorize, §10/X-23), `UNKNOWN` (operator settle only) |
| Cancellation redelivery | live (cancel), `cancelled` (redeliver stop + report), terminal-other (refuse) |
| Run attachment | unattached (claim/attach), attached (follow target; re-resolve on supersede, §7/X-08) |

No `isRetry` flag may influence authorization or lifecycle dispatch.
`store.update` is replaced by predicated per-transition operations
(`adoptRunID`, `parkApproval`, `settleTerminal`, `markCancelled`,
...); each checks affected rows and reloads on zero.

## 7. Transition tables (normative; 15.2 §7.7)

"No lifecycle correction ships without a contract row. No row claims
enforcement without a test." Predicate = exact `WHERE` / lock scope.
All transitions run inside the tenant context with explicit tenant
predicates; RLS stays defense in depth.

### M1 — Invocation lifecycle (`runtime` service/store)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Invoke (first claim) | — | `pending` (gated) / `running` | `INSERT ... ON CONFLICT DO NOTHING` on idempotency unique; affected=1 wins | one tx | authenticate + authorize + entity admission before claim (X-24b: auth before registry resolve) | `invoke/dispatched` (mandatory; failure → retryable) | store error → retryable, no row | `TestInvokeClassAHappyPath`, `TestInvokeRejections` (kept), `TestSubmitChokePointCallerSet` (§5 proof), `TestInvokeEntityDerived`, `TestInvokeEntityMissingFails` (kept) |
| Same-key retry | `pending` | dispatch | reload row; dispatch on observed state | read + handler tx | same as first claim | `invoke/duplicate` (best-effort) | — | `TestInvokeIdempotentReplay` (kept) |
| Same-key retry | `running`/`waiting_event` | same + handle | choke point (§5) + predicated `adoptRunID` (Class D; Class A excepted — see interrupted row below) | handler tx | choke approval check | `invoke/duplicate` | submit error → record stays live → retry re-enters | `TestConcurrentSubmitSameKey` (X-03) |
| Class A attempt (P21-003) | `pending`/`running`/`waiting_approval` | terminal | advisory execution claim (try, non-blocking) + entity lock, then handler + predicated in-tx completion; zero rows → whole attempt rolls back + reload/replay | single attempt tx | entry authorization (claim carries no authority) | `invoke/succeeded` or `invoke/failed` on win, exactly once | claim busy → terminal `execution-in-progress` conflict (same-key retry); pre-handler terminal → replay without executing | T1/T2/C3/T6 + C1/C2/C4 units |
| `execution-in-progress` (P21-003) | attempt | terminal 409 conflict | claim held by another attempt; names same-key retry; loser writes nothing and executes nothing | no tx held (prompt refusal) | — | no audit on the conflict itself | explicit same-key retry converges via replay (success) or re-claims (post-crash) | C3 promptness + untouched-row + retry test |
| Same-key retry, Class A (P21-003) | `running` + Class A | `interrupted` (terminal) | fail-closed: prior attempt did not complete; resubmit under a new key | read | — | `invoke/duplicate` | never re-dispatches the live attempt | existing `interrupted` replay tests |
| Class A grant→execute (P21-003) | `waiting_approval` + granted | terminal | granted poll enters the Class A attempt above (no workflow submit for Class A) | attempt tx | stored granted linkage | win audits as attempt row | denial/expiry settle instead (converge rows) | T1 granted-poll test |
| Park | `pending` (gated) | `waiting_approval` | ONE tx in §4 order (CR-02): 1 invocation `FOR UPDATE` + predicated park-`UPDATE`, THEN 2 approval `INSERT`, then `approval-requested` audit; commit rolls all three together | one tx | approval requirement from manifest | `approval-requested/pending` exactly once per invocation | any failure → all three roll back → retryable; retry re-parks under the same idempotency identity (dangling-row adoption retained ONLY as defensive recovery for legacy/pre-existing rows, built from fixture — Z CR-05) | `TestApprovalRetryAfterSubmitFailure`, `TestApprovalRetryAfterParkFailure` (adoption leg from fixture), crash-between-claim-and-park (X-01) |
| Submit-after-park | `waiting_approval` + approval `pending` | same + run | choke point with stored linkage; run-ID adoption | submit tx | stored linkage | submit failure → `submit-failed` retryable | `TestApprovalPollSubmitsIfMissing` (X-02) |
| Grant converge | `waiting_approval`, approval decided/approve | `running` + handle | submit-if-missing (stored linkage) then executor `Status` | submit tx + status read | linkage intent-hash match | grant observed (`approval-denied` n/a) | never-submitted run → submit first, then converge | `TestApprovalGrantAfterSubmitFailure` (new, X-02) |
| Deny/expire/cancel converge | `waiting_approval` | `failed` | `settleTerminal` predicated on nonterminal | settle tx + audit | — | `approval-denied/expired/cancelled` + `run-settled` | run-less denial settles directly (no loop) | `TestApprovalDenialSettlesAfterParkFailure`, `TestApprovalExpirySettlesAfterParkFailure` (new), `TestDenialCancelsRun`, `TestExpiryCancelsRun` (kept) |
| Completion | `running`/`waiting_event` (+run or +attached_run) | terminal | `settleTerminal`: predicated `UPDATE ... WHERE status IN (live)`; attached records resolve the OWNER row's outcome (success/failure/cancel propagate truthfully — never `Succeeded` while the target is active); winner audits | settle tx | — | `run-settled/<terminal>` exactly once (Status read and sweeper share this op — Rule 1 kills double-audit, incl. across owner + attached rows) | discover via Status read or lifecycle sweep (§8, S3 covers `attached_run_id`); crash between → next sweep settles | `TestSettlementAuditedWithoutPoll` (X-12), `TestAttachedSettlementFollowsTarget` (CR-01) |
| Status sync | any + run/attached_run | synced | resolve `attached_run_id` → owner run first; same `settleTerminal` when live→terminal; sync errors preserved, never swallowed (X-24a) | read + conditional settle | auth first | settle audit only on transition win | executor down → `Stale` + last record (terminal stays authoritative) | `TestStatusStaleOnExecutorDown` (kept) + `TestStatusSyncErrorPreserved` (X-24a) |
| Check-live (legacy callers migrated) | any | — | `checkLive`'s locked-read-then-release shape is ELIMINATED as an authorization gate: the effects pre-claim gate migrates into the M3 Authorize-attempt tx; the `EffectCanceller` adapter is removed with it; any remaining direct `CheckLive` caller is migrated to a lock-free status read or removed — P2 greps `CheckLive\|CheckCancelled` callers to prove none remains | n/a | — | — | no caller may treat a standalone liveness read as authorization | caller-migration grep test (X-05) |

### M2 — Approval (`approvals` + bridge)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Decide | `pending` | `decided` | `SELECT ... FOR UPDATE`, then predicated `UPDATE ... WHERE status='pending'` + affected check | one tx | scope + intent hash + non-self + tenant (all re-checked under lock) | `approval-decided` at COMMIT (before signal) — Gate 6 | loser reloads: identical → idempotent success; conflicting → defined conflict error | `TestConcurrentApproveReject` (X-04) |
| Concurrent decide | `pending` | one `decided` | `SELECT ... FOR UPDATE` + predicated update; exactly one winner | one tx each, serialized | scope + hash + non-self + tenant under lock | winner audits | loser errors (never silent overwrite) | `TestConcurrentApproveReject` (X-04) |
| Decide vs cancel | `pending` | `decided` XOR `cancelled` | Decide's row lock serializes with `CancelPending`'s predicated `UPDATE ... WHERE status='pending'` | one tx each | scope + hash + non-self + tenant under lock | winner audits | Decide-after-cancel refused deterministically; cancel-after-decide no-ops | `TestConcurrentDecideAndCancel` (X-04), `TestDecideAfterCancelRefused` (kept) |
| Expiry marking | `pending` + past expiry | `expired` | predicated `UPDATE ... WHERE status='pending' AND expires_at passed` | one tx (Decide path + sweeper path share it) | — | `approval-expired` | lazy (poll/Decide/sweep); never invented without a supplied expiry | `TestConcurrentExpiryAndDecision` (new, X-04), `TestManifestExpiryRecorded` (kept) |
| `CancelPending` | `pending` | `cancelled` | existing predicated update (conforming, kept) | one tx | cancel intent (M6) | via M6 cancel audit | idempotent no-op when none pending | `TestCancelPending` (approvals), `TestCancelPendingApproval` (runtime) (kept) |
| Signal-after-decide | decided + run | notified | verified-signal op (M4): existence check + `Send`; signal is a hint, table authoritative | signal call (no tx across it) | — | commit audit already written; signal failure → retryable, sweeper redelivers | kill-before-signal → sweep redelivers (§8) | `TestApprovalLostSignalBoundedRecovery` (X-11) |

### M3 — Effect / attempt (`effects`)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Claim (new) | no row | `PENDING` (provisional) | conditional `INSERT` (idempotency on business-op PK; no authorization implied) | one tx | BusinessOpID scope gate only | — | conflict → existing-row path | `TestCancelDuringClaim` (X-05) |
| Authorize attempt | `PENDING` | `ATTEMPTING` (point of no return) | ONE tx in §4 order: §4.1 `RLock` → 1 invocation `FOR UPDATE` (live?) → persisted-trust admission (trust ∈ declared `TrustModes`, empty = unrestricted; else `trust-mode-mismatch`) → 2 approval `FOR SHARE` (linkage? — SKIPPED iff persisted trust autonomous AND manifest explicitly permits autonomous, P24-005 option (a)) → 3 sequence `FOR SHARE` (generation fence, if sequence-bound; deferred per P24-004) → 4 switches `FOR SHARE` + pinned grants → 5 predicated `UPDATE ... WHERE state='PENDING'` + lease/owner check | one tx; send strictly after commit | full StrictPolicy in-tx + fence identity from trusted context | refusal → `effect-refused/<reason>` audit (no `ATTEMPTING` row — Rule 5, X-13) | cancel/supersede/revoke-first → transition fails → `PENDING` withdrawn to `CANCELLED` (terminal, never attempted); attempt-first → in flight | `TestCancelBetweenPendingAndAttempting` (new, pause-based), `TestRefusalAudited` (X-13), `TestSupersededGenerationCannotAttempt` (X-08/CR-04), `TestP25M1–M7` (P24-005 matrix) |
| Withdraw (cancel-won) | `PENDING` | `CANCELLED` (terminal) | predicated `UPDATE ... WHERE state='PENDING'` (eager batch in M6 cancel tx; lazy backstop on next touch) | cancel tx / touch tx | cancel intent (M6) | `effect-withdrawn` with invocation + op (distinct from failure), committed atomically with the winning transition in the authorize tx — exactly one per win, repeats refuse (P24-001) | never retried, never counted as a provider attempt | cancel-path tests (M6), `TestP25POC1*`, `TestP25TerminalWithdraw*` |
| Claim (existing) | `PENDING`/`ATTEMPTING` | adopt/takeover/quarantine/withdraw | lease + age-predicated transitions; stale-`PENDING` takeover and `NOT_APPLIED` retry route through the Authorize-attempt tx (a new authorization, not a shortcut); post-cancel touch withdraws to `CANCELLED` | per-op tx | settled reads skip gates; live paths re-gate | — | contention → `ErrAttemptInProgress`; stale → takeover/UNKNOWN | `TestConcurrentDuplicateClaims`, `TestStalePendingTakeoverSendsOnce`, `TestPerformAdoptsStrandedAttempt` (kept); stale-resume interleaving (F-03, §11) |
| Send | `ATTEMPTING` | `CONFIRMED`/`FAILED`/`UNKNOWN` | authorized `PENDING→ATTEMPTING` precedes every send (§4/CR-03); settlement predicated on `ATTEMPTING` + attempt epoch — a stale worker's outcome lands zero rows and reports superseded, never overwrites (P24-002) | authorize tx, then send, then settle tx | — | outcome via ledger; mock log per attempt (X-07) | unknown → quarantine, never blind retry | `TestCancelDuringAttempt` (X-05), `TestMockRecordsEveryAttempt` (X-07), `TestP25POC2*` |
| Settle (operator) | `UNKNOWN` | terminal (direct) | ATOMIC single predicated `UPDATE ... WHERE state='UNKNOWN'` (directive §3.3 decision — the two-tx `RECONCILING` intermediate is removed from the machine; no strandable middle state exists) | one tx | operator surface only — CLI `settle` verb with canonical auth (doc-spec §12; never a worker) | `effect-settled/<outcome>` with evidence note; idempotent on (op, outcome, note): same → success + audit backfill if missing; different → conflict error (covers crash between state commit and audit) | non-UNKNOWN refused; concurrent settlers serialize on the predicate (one wins) | `TestSettleAndListUnknown` (updated), `TestSettleAudited` (new), `TestSettleCrashConverges` (new, fault-injected kill between commit and audit) |
| Retry-after-reconcile | `NOT_APPLIED` | `ATTEMPTING` | via Authorize-attempt tx (predicated on `NOT_APPLIED` + full revalidation — makes the §10/X-23 edge reachable) | authorize tx, then normal send path | full authorization re-run | — | — | `TestReconcileRetryAfterNotApplied` (new) |
| Refusal | any pre-claim | (no row) | n/a — refusal is the absence of a claim | audit tx | reason: cancelled / token / grant / switch (attributed, X-13) | `effect-refused` with tenant, invocation, op, rule, timestamp; never an `ATTEMPTING` row; never secrets | — | `TestRefusalAudited` per reason |

P24-008 gating prerequisite: before the first non-mock
provider is enabled, authorization must verify that the
requested capability belongs to the admitted tool's declared
capability set. Mock-only operation is unaffected;
real-provider admission is refused until this membership
check lands with its tests.

### M4 — Workflow / run identity (`executor/dbos` + store)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Signal | any run | 5-way outcome | verified-signal op: `RetrieveWorkflow` (existence) THEN `Send`. Taxonomy (CR-10): unknown workflow → `unknown-run` error; unavailable family → explicit `family-unavailable` error; live workflow + `Send` nil → `signal/accepted` (LIVE RUNS ONLY); already-terminal workflow → distinct `already-terminal` no-op outcome (never ordinary success); processed-by-workflow → claimed ONLY with actual processing evidence (re-read/table state), never from `Send` nil. Residual Retrieve→Send TOCTOU documented (completion between check and send yields a harmless buffered no-op, reported as such when detectable) | two SDK calls + status read | service auth + tenant ownership first | `signal/accepted` (live) / `signal/already-terminal` / errors per outcome | never false success | `TestSignalUnknownRun`, `TestSignalTerminalNoOp`, `TestSignalUnavailableFamily` (X-06) + post-restart routing |
| Status | run | mapped status | fan-out `RetrieveWorkflow` + `GetStatus`; mapping amended per CR-09: `MAX_RECOVERY_ATTEMPTS_EXCEEDED` → `quarantined` (was: failed) — dead-letter is resumable, so the invocation stays nonterminal pending operator resume/fail; all other mappings kept | reads | auth + tenant | — | unknown → error; down → stale (M1) | `TestSubmitStatusRoundTrip`, `TestTwoFamilyRouting` (kept) + quarantined-mapping assertions in `TestForeignVersionNonterminalVisibility` |
| Cancel (executor) | live run | cancelling | `dbos.CancelWorkflow` (cooperative: in-flight steps complete per SDK doc — the §11 ops line) | SDK call | via M6 | via M6 | unknown → SDK error (surfaced, never silent) | `TestCancelStops` (kept) + F-02 wording fix |
| Shutdown | serving | stopped | `dbos.Shutdown` per family; error enumerates pending incl. `workflows` (v1.9.0 `dbos/dbos.go:1057-1148`) | call (no tx) | — | loud diagnostics (never swallowed) | §8/X-14: on error skip `pool.Close`, report components, exit non-zero (`cmd/miniapp`); test stack `t.Errorf` + no-hang (C6) | `TestShutdownFailurePreservesPool` (fault-injected + real-timeout variants) |
| Foreign visibility | any version | reported | list nonterminal `PENDING, ENQUEUED, DELAYED, SUSPENDED` grouped by version (C5) + `MAX_RECOVERY_ATTEMPTS_EXCEEDED` as a separate DEAD-LETTER class (resumable per SDK `ResumeWorkflows` — NOT terminal-failed); dead-lettered run maps to invocation `quarantined` (M1 nonterminal; operator resume/fail only) | reads | operator surface | startup + periodic (sweep piggyback) + queryable counts | — | `TestForeignVersionNonterminalVisibility` (X-10) |
| Version affinity | v1 pending + v2 worker | unmoved | SDK `ReenqueueForRecovery` version predicate (v1.9.0 `:6349`) | SDK | — | — | behavioral proof, not citation | `TestVersionV2DoesNotExecuteV1Run` (X-09) |
| Quarantine resume | `quarantined` | `running` | exact API `dbos.ResumeWorkflow(ctx, runID)` (eligible: any status except SUCCESS/ERROR — v1.9.0 `system_database.go:2877-2918`; framework gates to dead-lettered runs); SDK resume FIRST, then predicated invocation `UPDATE ... WHERE status='quarantined'`; crash between converges via sync (quarantined + live workflow → `running`, audited) | SDK call + settle tx | operator authority | `resume/resumed` with operator + run | resume-after-cancel refused (predicate excludes non-quarantined); SDK resume NEVER revives a framework-terminal invocation (sync only settles live records; terminal rows are authoritative) | `TestQuarantineResume`, `TestResumeAfterCancelRefused` |
| Quarantine operator-fail | `quarantined` | `failed` | predicated `UPDATE ... WHERE status='quarantined'` | settle tx | operator authority | `run-settled/failed` + operator attribution | concurrent resume-vs-fail serialize on the predicate (one wins, loser gets defined conflict) | `TestQuarantineOperatorFail`, `TestQuarantineConcurrentDecisions` |
| Quarantined status | `quarantined` | reported | operator-visible representation: `status: quarantined` + reason (dead-lettered: recovery attempts exhausted on workflow `<runID>`) + required action (operator: resume or fail) | read | auth first | — | never auto-settles; never silently failed while resume is available | `TestQuarantinedStatusVisible` |
| Run routing | submit | map entry | `runs` map bounded (cap + eviction) or replaced (X-24f); durable `run_id`/`attached_run_id` authoritative, cache never is | mutex | — | — | post-restart fan-out (existing) | `TestRunRoutingMapBounded` (new) |

### M5 — Sequence / attachment (`probe/classd` + M1 columns)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Sequence claim | no live generation | `active` (winner) | partial-unique `INSERT ... ON CONFLICT DO NOTHING` + `RETURNING` (existing, kept) | one step-tx | tenant-armed | — | conflict → attach path | `TestSequenceClaimRace` (new, barrier; replaces sequential `TestProbeConcurrency` for the race) |
| True attach | live generation exists | loser → `attached_run_id` = winner run | loser invocation stores winner run ID; no second workflow submits for the sequence (loser's own `run_id` stays empty); attachment retains tenant + principal boundaries (attach tx verifies both) | attach tx | tenant match + principal authorization enforced (cross-tenant attach refused) | `attach/attached` with target run | loser status follows target run (M1 Completion); target failure/cancel reported truthfully (never `Succeeded` while target active); cancel per M6 attached-cancel (DETACH-ONLY) | `TestProbeAttachLifecycle` (X-08) |
| Supersede | stale `active` (past `wake_at` + grace, no heartbeat) | `superseded` | supersede `UPDATE` on the sequence row (lock 3) | supersede tx | — | supersede noted | live runs heartbeat and are never superseded | `TestSupersededGenerationCannotAttempt` (barrier: old gen paused pre-authorization → supersede commits → resume → refused, provider never called) |
| Generation fence (CR-04) | any attempt by a superseded generation | refused (no `ATTEMPTING`) | fence enforced INSIDE the M3 Authorize-attempt tx: sequence row `FOR SHARE`, verify `status='active' AND generation=?` against the fence identity carried in the workflow's trusted execution context (sequence key + generation from the recorded claim-step output, threaded through the effect Intent by framework workflow code — never caller-supplied free text); non-sequence effects skip the fence (no row) | authorize tx (§4/CR-03) | fence identity bound to tenant + invocation | refusal → `effect-refused/superseded` | supersede-first → old generation cannot authorize another attempt; attempt-first → in flight, reported honestly; in-flight provider work at supersede time cannot be recalled (recorded, not described as cooperative cancellation) | same barrier test; reusable framework primitive — future Class D miniapps inherit it via `Perform`, never reimplement |
| Attach re-resolve | target terminal + newer live generation | re-attached | Status/attach read resolves current live generation | read | tenant match | — | no live generation → report target outcome | `TestProbeAttachLifecycle` (X-08) |

P24-004 owner disposition (explicit scope deferral, not an
implemented guarantee): the Generation-fence row above is
PROBE-SCOPED — no verifier implementation, no trusted identity
propagation, and no authorization-time generation validation
exist today (named limitation; sequence effects remain
mock-only). Activation trigger (explicit and enforceable): a
miniapp manifest declaring sequence-bound effects (the
framework-declared sequence-requirement marker in
tool/workload configuration — never inferred from
caller-supplied `Intent.Sequence` presence) submitted for
registry admission is REFUSED until verifier + trusted
propagation (framework-originated, tenant/invocation/effect
bound) + supersede validation + the X-08 barrier test all
land. When active, missing identity/verifier fails closed at
lock-order position 3 with no new transaction, I/O, or
advisory lock.

### M6 — Cancellation (`runtime` + effects)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Cancel | live | `cancelled` | predicated `UPDATE ... WHERE status IN (live)` (first — durable intent), then approval settlement (2) + `PENDING→CANCELLED` withdraw batch (5) in the same tx (§4.2) | one tx; executor stop after commit | auth + tenant ownership | `cancel/cancelled` with reason + `in_flight` (`ATTEMPTING`, settle-by-evidence) + `withdrawn` (`PENDING→CANCELLED`, never attempted) — never conflated | executor down → retryable with intent standing; redeliver via repeat Cancel / Status / sweep | `TestCancelBeforeClaim`, `TestCancelPendingApproval`, `TestCancelDuplicateSubmission`, `TestCancelExecutorDownRedelivers` (kept) + `TestCancelDuringClaim`/`TestCancelDuringAttempt`/`TestCancelBetweenPendingAndAttempting` (X-05) |
| Approval settlement | cancel intent | approvals `cancelled` | existing predicated `CancelPending` (kept) | cancel tx | — | — | — | `TestCancelPendingApproval`, `TestDenialCancelsRun` (kept) |
| Executor stop | cancel intent | run cancelling | `cancelRun` (M4 cooperative) | SDK call | — | — | terminal-absorb; else retryable | `TestCancelStops`, `TestCancelExecutorDownRedelivers` (kept) |
| In-flight report | post-record | list | `LiveEffects` AFTER the durable record (a claim committing first is always visible — Rule 2) | read | — | in `cancel/cancelled` details | — | `TestCancelInFlightReported` (kept) + X-05 visibility assertions |
| Attached-cancel (DETACH-ONLY, owner-approved) | attached + live target | attached row `cancelled`, target untouched | predicated `UPDATE attached row ... WHERE status IN (live)`; NEVER calls `CancelWorkflow` on the shared target; `attached_run_id` preserved immutable | cancel tx | normal tenant ownership + authority checks; cross-tenant refused | `cancel/cancelled` identifying attached invocation + target workflow | idempotent on redelivery; owner row + other observers unaffected; attached row stops following target (stays `cancelled`) | `TestAttachedCancelDetachesOnly`, `TestAttachedCancelIdempotent`, `TestAttachedCancelCrossTenantRefused` (CR-01) |
| Owner-cancel with observers | owner live + N attached | owner `cancelled`, target cancelling, observers follow | standard M6 cancel on the owner row; target `CancelWorkflow` requested (owner owns the workflow); attached rows follow the terminal outcome via M1 Completion | owner cancel tx + propagation | owner authority | owner `cancel/cancelled`; observers settle with `run-settled/<target-outcome>` | observers never Succeeded-while-target-active; redelivery converges | `TestOwnerCancelPropagatesToAttached` (CR-01) |
| Target-cancel propagation | target terminal | observers settle | M1 Completion resolves `attached_run_id` → owner outcome | settle tx | — | shared `settleTerminal` (no duplicate audits) | success/failure/cancel all propagate truthfully | `TestAttachedSettlementFollowsTarget` (CR-01) |
| Post-supersede cancel | superseded target + attached | observers re-resolve or settle | attach re-resolve row (§7/M5); cancel of a fenced target follows target-cancel propagation | read + settle | tenant match | — | no observer strands on a dead generation | supersede-cancel test (CR-01) |
| Post-termination status (attached) | attached + terminal target | reported outcome | resolve + report; cancelled-attached rows report `cancelled` (own lifecycle ended), others report target outcome | read | auth first | — | historical `attached_run_id` available for audit | attach-lifecycle tests |

Run-less records (`pending`, Class A pre-completion) carry no
`run_id` and cannot be addressed by the `run_id`-keyed Cancel API
(P20 INFO-7 — intended: nothing is running to stop).
Effective-trust rule (unparked records only — P21-004: waiting
records poll their approval and never float): authorization
evaluates the requested trust within grant on every entry, so a
confirm→autonomous retry floats past the approval requirement on
the Class A path (executes without linkage) while the Class D
choke refuses it (`errApprovalRequired`, fail-closed); both
directions are grant-capped (no privilege escalation). Specified
behavior, not a defect; Phase 2 treats it as such.

### Owner Decision — Cancellation of an Attached Invocation (ACCEPTED)

Recorded verbatim from the P1 review directive: **DETACH-ONLY.**
Cancelling an attached invocation (1) transitions only that row to
`cancelled`; (2) never calls `CancelWorkflow` on the shared target;
(3) never modifies the owning invocation; (4) never affects other
observers; (5) preserves `attached_run_id` immutable (never cleared);
(6) audits attribution (attached invocation + target); (7) is
idempotent; (8) requires normal tenant/authority checks. Afterward
the row stays `cancelled` and stops following the target. Conversely,
non-cancelled observers resolve the target's authoritative outcome.
An attached invocation can never implicitly cancel the shared
workflow. Required tests (§10/X-08): attached-only settlement,
target completes normally, other observers unaffected, `attached_run_id`
preserved, idempotency, cross-tenant refusal, owner behavior unchanged.

### D1 — Control-plane authorization (migration 000006 + operator)

| Event | Source | Dest | Predicate / lock | Atomicity | Auth | Audit | Failure exit → recovery | Test |
|---|---|---|---|---|---|---|---|---|
| Flip (operator) | `engaged` either | flipped | privileged setter function (owner-only `EXECUTE`), invoked by the separate `switch` operator command over the control connection (serving process never holds the control DSN — CR-08) | setter tx | canonical API-key auth at the verb; actor/reason/scope validated server-side | trigger-written `switch-flipped` (actor, reason, scope, before/after) — `ENABLE ALWAYS`; fires for every writer except an owner/superuser who deliberately disables it (accurate wording, NOT "unbypassable") | invalid scope/actor → refused, no state change | `TestKillSwitchNotHandlerWritable`, `TestKillSwitchOperatorVerb`, `TestKillSwitchFlipAudited` (X-15) |
| Handler attempt | any | refused | `REVOKE` on `app.kill_switches` from restricted roles (proven with ephemeral test roles over genuine restricted connections; R2/CR-08) | n/a | role separation | n/a (refusal is the DB error) | direct `UPDATE` fails | `TestKillSwitchNotHandlerWritable` (X-15) |
| Policy read | any | engaged set | `GRANT SELECT` to app role | read | — | — | missing authority fails closed | `TestKillSwitchBlocks`, `TestEffectGrantRevoked`, `TestEffectTokenMismatch` (kept) |

### Per-machine terminal states, invariants, and prohibitions (CR-05)

- **M1 invocation.** Terminal: `succeeded, failed, cancelled`.
  Nonterminal: `pending, running, waiting_approval, waiting_event,
  quarantined` (quarantined = dead-lettered, operator-actionable;
  §10/X-10). Invariants: a settled invocation never returns to a
  live state; a run ID is never overwritten once canonical
  ownership is established; a nonempty `run_id` maps to exactly
  one owning invocation; an attached invocation resolves to exactly
  one authoritative target; quarantined moves only via operator
  resume/fail. Prohibited from terminal: submit, park, decide,
  signal-accept, cancel (cancel on `cancelled` redelivers the stop,
  never re-lives).
  `live` (operation term, P20 INFO-1): `pending, running,
  waiting_approval, waiting_event` — nonterminal minus
  `quarantined`. Operation-specific exclusions (all verified in
  implementation): `cancelIfLive`/`syncIfLive`/`checkLive`
  exclude `pending` (pending records carry no runs — enforced
  structurally: adoption atomically marks pending→running, so
  the exclusion can never strand a submitted workflow; P21-002);
  `completeIfLive`/`completeTx` include `pending` (undispatched
  Class A must complete); `failIfWaiting` is waiting-only (poll
  settlement); `adoptRunID` takes all four live states and marks
  pending→running on win (restoration adopts from `pending`).
  Terminal/quarantined are excluded everywhere.
- **M2 approval.** Terminal: `decided, expired, cancelled`.
  Invariants: a decision is never substituted after the first
  terminal decision (identical re-decide is idempotent, never a
  substitution); expiry requires a supplied `expires_at` (never
  invented). Prohibited from terminal: `Decide`, re-marking.
- **M3 effect.** Terminal: `CONFIRMED, FAILED, UNKNOWN(settled
  only via Settle), NOT_APPLIED(settled form; retryable only via
  Authorize), CANCELLED`. Nonterminal: `PENDING, ATTEMPTING`.
  (`RECONCILING` removed from the machine per directive §3.3:
  settlement is atomic `UNKNOWN→terminal`; no intermediate state
  can strand.) Invariants: at most one valid owner authorizes a
  send per business operation; a superseded generation authorizes
  no new attempt; `UNKNOWN` is never blindly retried; `CANCELLED`
  means withdrawn before authorization (provider never called).
  Prohibited: any transition out of `CONFIRMED/FAILED/CANCELLED`;
  `UNKNOWN→ATTEMPTING` direct (must settle first).
- **M4 workflow/run.** Terminal (framework view): `SUCCESS, ERROR,
  CANCELLED` (+ dead-letter class, nonterminal-quarantined, §10/X-10).
  Invariants: every signal outcome is one of the 5-way taxonomy
  (§10/X-06); shutdown failure never silently closes a live pool.
  Prohibited: reporting `accepted` for terminal runs; closing the
  pool under active workers.
- **M5 sequence.** Terminal per generation: `done`; row terminal:
  `superseded` (no further attempts), `done`. Invariants: at most
  one `active` generation per (tenant, subject) (partial unique);
  an attached loser never owns the sequence. Prohibited from
  `superseded`/`done`: new generation claims on the same row
  version (a new generation is a new row).
- **M6 cancellation.** Terminal: intent recorded + propagation
  converged (executor stop absorbed-or-converged, approvals
  settled, `PENDING` withdrawn). Invariants: durable intent
  precedes every stop; `in_flight` and `withdrawn` are never
  conflated. Prohibited: new authorizations after intent commit
  (Rule 2); cancelling another invocation's owned workflow from
  an attached row (DETACH-ONLY).

## 8. Lifecycle sweeper (X-11 mechanism selection — Gate 7)

P1 evaluated the three 15.2 §8/D candidates:

1. **Framework-owned approval-wait primitive + mandatory bound.**
   Rejected as the primary mechanism: miniapps invoke `dbos.Recv`
   directly and the framework cannot intercept SDK calls at runtime;
   enforcement would rest on analyzer/admission gating (build-time)
   plus a new miniapp-contract obligation — which Gate 7 requires
   surfacing as an owner decision. Sound as defense in depth, not as
   the enforceable bound.
2. **Narrowly-scoped renotify/expiry/settlement sweeper. SELECTED.**
   Runtime-enforced, needs no miniapp cooperation, no decider retry,
   and no miniapp-contract change (no owner decision). It is NOT the
   Phase 4 production effects reconciler: it never drives effect
   outcomes (no `UNKNOWN` settlement; `Settle` stays operator/test
   API). Scope is strictly invocation/approval lifecycle.
3. **Poll-driven renotify only.** Rejected: couples convergence to
   external poll cadence, which the framework does not control — no
   bound is provable.

Selected design — `sweepOnce(ctx, now)` (framework worker; also
invoked opportunistically on `Status`/poll paths with bounded work,
on serve-loop ticker in production, on startup recovery, and
directly by tests for determinism):

- **S1 lost-signal redelivery:** approvals `decided` + invocation
  still `waiting_approval` + workflow nonterminal → re-`Send` the
  `approval.decided` hint via the verified-signal op (M4). Harmless
  duplicates (workflows re-read the table). Bound: sweep interval
  (default 30s, configurable; tests invoke directly).
- **S2 expiry observation:** approvals past `expires_at` + invocation
  `waiting_approval` → predicated expiry marking (M2) + signal +
  invocation settlement (M1 deny/expire row). Covers the
  long/unbounded-`Recv` workflow: the wake comes from the signal,
  not from its own timeout.
- **S3 settlement discovery:** live records with `run_id` OR
  `attached_run_id` whose owner-target executor `Status` is terminal
  → shared `settleTerminal` op (M1 Completion — attached rows
  resolve the owner outcome; CR-01/R-04). Satisfies X-12 without any
  caller polling; crash windows close on the next sweep (startup
  sweep covers "ordinary recovery").

Kill-before-signal, decider-never-retries, and 24h-`Recv` cases all
converge within one sweep interval *while the guarantee's conditions
hold* (below). If the decision commits immediately before
termination, the startup sweep redelivers (S1). The probe keeps its
30s `Recv` convention as a compatible wake hint; the "durable
mailbox" phrase (`classd-capability-matrix.md:86`) is struck and
replaced with this sweeper description (Gate 6).

### 8.1 Transaction boundaries (Z CR-04)

Sweep steps run as separate sequential transactions; no lock is
held across steps. Within S2, one transaction takes locks 1→2
(invocation then approval, §4.2) for expiry marking + invocation
settlement, commits, and only then signals. S1 holds no invocation
lock (lock-free discovery + SDK send). S3's discovery reads are
lock-free; each settlement is a single-row predicated `settleTerminal`.

### 8.2 Conditional recovery guarantee (CR-06/R-05)

The interval bound applies while at least one healthy sweeper
instance runs with PostgreSQL and the relevant DBOS family
available. Startup recovery runs one sweep after service restart.
Concurrent replicas are safe (all sweep writes are predicated and
idempotent). Each sweep is batch-bounded with backlog accounting;
sweep failures are logged with counts and retried on the next
sweep/startup (never silent). A complete system outage has no finite
bound — recovery resumes when service becomes available. The safety
guarantee is the durable invocation/effect gate (a settled
invocation authorizes no new effect); resource cleanup is
best-effort cooperative cancellation. The sweeper never claims a
workflow terminated merely because it settled the invocation and
sent a signal.

Recommended expiry behavior (directed): (1) persist expiry + settle
the invocation authoritatively; (2) the effect gate henceforth
rejects consequential attempts; (3) send the expiry notification;
(4) request cooperative workflow cancellation best-effort; (5)
expose cancellation failure and retry on subsequent sweep/startup;
(6) never claim confirmed-stopped from a requested cancel.

Table re-read after wake is required by plan4.2 §15.1 ("Recheck
expiry, material hash, authority, and self-approval policy
immediately before the consequential action") — cited here so it is
not read as a new miniapp obligation (Gate 7). Defense in depth:
WS-G schedules an analyzer rule against raw unbounded `Recv` in
miniapp packages.

### 8.3 Sweeper operations (P2 ops-doc spec)

`docs/phase1/classd-operations.md` gains: serving process group +
tick schedule (default 30s, configurable) + startup one-shot +
opportunistic bounded work on `Status`/poll; S1/S2/S3 operation
reference; failure/backlog monitoring (stopped sweeper regresses
lost-signal recovery, expiry observation, and unpolled settlement
to pre-sweeper behavior — recoverable on restart, visible as
backlog); restart-recovery procedure.

## 9. Fault-injection design (15.2 §9)

`testkit/fault.Injector`: test-only, configured per test to fail a
named operation on its Nth attempt (`FailAt` one-shot, `FailFrom`
persistent). Per-test isolated, race-safe, no package globals;
nil-safe (nil never fires). Seams (no production behavior change
when unset):

- Test-side wrappers (`testkit/faultwrap`, no prod edits):
  `Async` (`async-submit/status/signal/cancel`),
  `Channel` (`provider-send`), `Policy` (`policy-check`). Each
  fires immediately before delegating.
- Prod-visible seam (one field per service):
  `Service.TestFault`, `effects.Service.TestFault`,
  `approvals.Service.TestFault`, `dbosexec.Executor.TestFault`
  (`*fault.Injector`, nil in every production constructor; a test
  asserts nil-after-`New` for each). Checked at the named points
  below. Never fires when nil.
- Fourth-seam traceability (WS-A carry-forward A): the fourth
  declaration is `dbosexec.Executor.TestFault`, owned by M4,
  purpose-built for the §9 `executor-shutdown` fault point.
  `Shutdown` is outside the `AsyncExecutor` interface (Submit /
  Status / Signal / Cancel only) and is invoked directly on the
  concrete executor, so no interface wrapper can interpose; the
  field seam is the only boundary-faithful option. Proven by
  `TestExecutorNilTestFault` + `TestFaultExecutorShutdown`.

CORRECTION (P2, per §13 never-silent rule). Original claim:
"faulting `Approver` (`Submit`/`Decide`)" via interface wrap, and
`TestFault` on three services. Corrected: `Approver` is a concrete
type (no interface), so approval points use the `TestFault` field
seam, not a wrapper; audit uses action-qualified ops
(`audit:<action>`) at the service `auditEvent` choke; `Shutdown`
is outside the `AsyncExecutor` interface, so the executor carries
the fourth `TestFault` field. Design intent (named boundaries,
nil-safe, tests-only) is unchanged.

Final op inventory (each with a firing/boundary/propagation test):

| Op | Site | Models |
|---|---|---|
| `approval-create` | `Approver.Submit` pre-INSERT | rollback (no row) |
| `approval-linkage` | `Approver.ByInvocation` pre-read | retryable lookup failure |
| `park-update` | `requestApproval` pre-park | rollback (nothing parked) |
| `audit:approval-requested` | service audit choke | park stands, audit missing → retryable |
| `runid-persist` | `submitRecorded` + `confirmSubmit` pre-update | rollback (no run recorded, no submit) |
| `claim-response` | `Invoke` post-claim-commit, pre-dispatch/park | lost response → retry converges on same record (L-01) |
| `async-submit` | wrapper pre-`Submit` | run persisted, submit failed (X-02 setup) |
| `decide-commit` | `Decide` post-validation, pre-UPDATE | rollback (stays pending) |
| `decide-response` | `Decide` post-commit, pre-return | commit stands; identical retry converges |
| `async-signal` | wrapper pre-`Signal` | decision stands; retry redelivers once |
| `effect-reserve` | `claim` pre-INSERT | rollback (no row, no send) |
| `attempt-begin` | `attempt` pre-transition | stays PENDING; lease→takeover converges |
| `takeover-begin` | `takeover` pre-transition | stays stale PENDING |
| `settle-commit` | `sendAndRecord` post-send, pre-transition | sent-once, ATTEMPTING; aging→UNKNOWN |
| `settle-response` | `sendAndRecord` post-transition, pre-return | commit stands; repeat observes, no resend |
| `settle-operator-commit` | `Settle` pre-transition | stays UNKNOWN; retry converges |
| `provider-send` | wrapper pre-`Send` | 0 delegate sends; attempt settles FAILED |
| `policy-check` | wrapper pre-`CheckPolicy` | refused before reservation |
| `audit:run-settled` | service audit choke on settle path | retryable once; terminal record converges |
| `executor-shutdown` | `Executor.Shutdown` entry | error reaches caller; workers untouched |

Required fault points (§9 list) all covered; WS-B keeps op names
stable when paths move into the authorize transaction. Red/green
(15.2 §9.1): HIGH + X-07 tests run red against an isolated
pre-remediation snapshot and green on the remediated tree —
snapshot method C2 (§13), both runs digested. FaultInjector
itself has no red run (it does not exist in the snapshot);
its contract tests are its proof.

## 10. Per-finding map (Gate 6 — full sub-part fidelity)

WS allocation (chatgpt review I): WS-A = X-01–X-04, X-19, X-22
(shared invocation/approval lifecycle ownership — C7); WS-B = X-05;
WS-C = X-07; WS-D = X-06, X-11, X-12, X-13, X-14; WS-E = X-08, X-09,
X-10; WS-F = X-15; WS-G = X-16; WS-H = X-17; WS-I = X-18; WS-J =
X-20, X-21, X-23, X-24a–f. Test names follow 15.2 §10 (exact names
may shift to repo conventions; every requirement maps to assertions
in a named file).

| Finding | Class | Rule | Required sub-parts (all ship) | Primary test |
|---|---|---|---|---|
| X-01 | RC-1c | 3, 4 | choke-point enforcement; atomic Park (§7/M1) with rollback-together; dangling-row adoption defensive/legacy-only via fixture (R-03); park-failure + crash-window coverage; exactly one `approval-requested` audit per invocation | `TestApprovalRetryAfterSubmitFailure`, `TestApprovalRetryAfterParkFailure` (incl. fixture-built adoption leg), `TestApprovalCrashBetweenClaimAndPark` |
| X-02 | RC-1d | 4 | submit-if-missing on poll (stored linkage); grant AND denial AND expiry converge incl. never-submitted runs | `TestApprovalPollSubmitsIfMissing`, `TestApprovalGrantAfterSubmitFailure` (new), `TestApprovalDenialSettlesAfterParkFailure`, `TestApprovalExpirySettlesAfterParkFailure` (new) |
| X-03 | RC-1a/1b | 1 | predicated `adoptRunID` (+ `attached_run_id`, R1); barrier test vs real executor; race detector | `TestConcurrentSubmitSameKey` |
| X-04 | RC-1a | 1, 2 | `FOR UPDATE` + predicated `Decide`; SAME predicate discipline for expiry marking and `CancelPending`; approve/reject, decide/cancel, self-approval, expiry/decision races | `TestConcurrentApproveReject`, `TestConcurrentDecideAndCancel`, `TestConcurrentSelfApprovalRefused` (new), `TestConcurrentExpiryAndDecision` (new) |
| X-05 | RC-1b | 2 | authorize-tx (§4/CR-03: PENDING provisional, ATTEMPTING point of no return) + lock order §4–§4.2; claim-before-cancel AND cancel-before-claim AND pause-between-PENDING-and-ATTEMPTING; in-flight vs withdrawn distinguished in cancel response; stale-takeover overlap; comment corrected | `TestCancelDuringClaim`, `TestCancelDuringAttempt`, `TestCancelBetweenPendingAndAttempting` (new) |
| X-06 | RC-1d | 5 | existence verification via `RetrieveWorkflow`; 5-way outcome taxonomy (§7/M4 — unknown / unavailable-family / accepted-live-only / already-terminal / processed-with-evidence); residual TOCTOU documented | `TestSignalUnknownRun`, `TestSignalTerminalNoOp`, `TestSignalUnavailableFamily` + post-restart routing |
| X-07 | RC-2 | 6 | per-attempt log rows (serial PK, no dedup); false comment + `effects_test.go:18` phrasing fixed; affected suites re-run | `TestMockRecordsEveryAttempt` (double-send → 2 rows) |
| X-08 | design | 4, 5 | GENUINE ATTACH: loser → `attached_run_id`, follows target lifecycle; cancel = DETACH-ONLY (owner-approved §7/M6: attached-only settle, target untouched, `attached_run_id` immutable, idempotent, cross-tenant refused); UNCONDITIONALS: no `Succeeded`-while-active, barrier claim-race test, supersede→fence (§7/M5 CR-04); six acceptance checks enumerated: (1) concurrent invocations → one canonical workflow, (2) loser gets a valid target-linked handle, (3) loser status follows the target, (4) target failure never reported as success, (5) cancel per DETACH-ONLY, (6) superseded generations send nothing further | `TestProbeAttachLifecycle`, `TestAttachedSettlementFollowsTarget`, `TestAttachedCancelDetachesOnly`, `TestAttachedCancelIdempotent`, `TestAttachedCancelCrossTenantRefused`, `TestOwnerCancelPropagatesToAttached`, barrier claim-race + `TestSupersededGenerationCannotAttempt` |
| X-09 | RC-2 | 6 | EXACT five steps, no weaker alternative: (1) pending v1 work under v1, (2) stop v1, (3) run only v2, (4) prove v2 neither executes nor advances the v1 workflow, (5) restart v1 and prove v1 recovers + executes it; SDK citation is mechanism note only | `TestVersionV2DoesNotExecuteV1Run` |
| X-10 | gap | 5 | enumerate PENDING/ENQUEUED/DELAYED/SUSPENDED by version (C5 classes) + DEAD-LETTER as its own operator-actionable class (verified resumable: SDK `ResumeWorkflows` re-enqueues every status except SUCCESS/ERROR — v1.9.0 `system_database.go:2877-2918`); dead-lettered run → invocation `quarantined` (nonterminal; operator resume re-lives, operator fail settles — never silently settled failed while resume is available); startup + periodic + queryable counts | `TestForeignVersionNonterminalVisibility` (incl. seeded dead-letter + quarantined-mapping), `TestQuarantineResume`, `TestQuarantineOperatorFail`, `TestQuarantinedStatusVisible`, `TestQuarantineConcurrentDecisions`, `TestResumeAfterCancelRefused` |
| X-11 | RC-1d | 4, 5 | sweeper §8 (bound + alternatives recorded); commit-time decision audit; mailbox phrase struck; lost-signal + long-wait test | `TestApprovalLostSignalBoundedRecovery` |
| X-12 | RC-1d | 5 | sweeper S3 + shared `settleTerminal`; no-poll + kill-near-completion tests; post-win settlement-audit backfill scope (P20 LOW-2: win committed + audit write failed → sweeper backfills the missing settlement audit on settled records; deferred, no Phase 1 code) | `TestSettlementAuditedWithoutPoll` |
| X-13 | RC-1d | 5 | `effect-refused/<reason>` audit per refusal class; no false `ATTEMPTING`; no secrets | `TestRefusalAudited` |
| X-14 | gap | 5 | skip `pool.Close` on failed shutdown; loud diagnostics + non-zero exit (prod); `t.Errorf` + no-hang (tests); deferred-cleanup audit (no unsafe late close); fault-injected AND real-timeout tests (C6) | `TestShutdownFailurePreservesPool` (fault-injected) + `TestShutdownTimeoutPreservesPool` (new, unresponsive worker) |
| X-15 | gap | 3, 5 | 000006 setter + `ENABLE ALWAYS` audit trigger; REVOKEs + production role instructions; proof over genuine restricted connections (ephemeral roles + actual runtime role); separate `switch` operator command holding the control DSN (canonical auth; serving process excluded); startup rejects unsafe role config; every flip audited; owner/superuser + same-process residuals recorded | `TestKillSwitchNotHandlerWritable`, `TestKillSwitchOperatorVerb`, `TestKillSwitchFlipAudited`, `TestRuntimeRoleIsRestricted` |
| X-16 | RC-2 | 6 | Ident/FuncLit reconciliation (shared node checks; `inStep` gates only nondet/directio allowance); fixtures for RunAsTransaction/Enqueue-in-step, uuid + non-stdlib randomness, depth-2 + cross-package, errgroup.Go, reflect.Select, context.WithTimeout, map/iterator gaps, valid-DBOS pins; detbad 12-vs-13 reconciled; detection boundary documented | analyzer fixture suite |
| X-17 | RC-2 | 6 | skip-fast DB gates on both kill/restart parents (reuse `NewDB` message); CI postgres service + env + sanctioned `-p 1` (Gate 4 — never bare `./...`); DB-less suite skip-clean; WS-H one-liner §13 | DB-less restart test + CI-equivalent run |
| X-18 | RC-2 | 6 | digest-per-run (C1 method); P0-style full-list entry corrected; historical claims marked irreproducible, never retrofitted; WS-I one-liner §13 | digest script test |
| X-19 | gap | 1 | canonical escape (`%`→`%25`, `:`→`%3A` per part, `:` join) in one helper shared by M1/M3/M6/D1; identity for legacy IDs without `%`/`:` (backward compatible, §12-compat note); comment softened | collision-pair test |
| X-20 | gap | 5 | orphan FAIL→WARN; admission-vs-preflight split documented | approved-unregistered test |
| X-21 | gap | 6 | checks.sh: enterprise scan += `executor runtime effects probe transport cmd`; miniapps scan += `probe preflight cmd`; pins += migrate/uuid/santhosh/x-tools (values verified vs go.mod); register migrate row refreshed; bypass fixtures per new package | gate fixture tests |
| X-22 | gap | 1 | empty run-ID rejection at `Signal`/`Cancel` service boundary (covers all adapters); `attached_run_id` + `UNIQUE (run_id) WHERE run_id <> ''` in 000006; `getByRun` owner-only + ambiguity-safe | empty-ID + uniqueness tests |
| X-23 | gap | 5 | `Timeout`: ENFORCE via submit ctx-deadline (admission parses, invalid → terminal; SDK timeout monitor verified present) + `TestManifestTimeoutEnforced` (new); `CANCELLED`: WIRE (reassessed per CR-03 — legitimate `PENDING→CANCELLED` withdraw transition; never attempted/retried) + `TestPendingWithdrawnOnCancel` (new); `NOT_APPLIED→ATTEMPTING`: WIRE (reconcile-retry via Authorize, §7/M3) + `TestReconcileRetryAfterNotApplied`; supersede→cancel: FENCE at authorization boundary (§7/M5 CR-04) + `TestSupersededGenerationCannotAttempt` (substitution from the review's linkage text recorded with rationale: fencing is stronger — cooperative cancel cannot stop an in-flight step) | named tests per item |
| X-24a | gap | 5 | sync errors preserved (terminal authoritative, staleness explicit, self-heal documented) | `TestStatusSyncErrorPreserved` (new) |
| X-24b | gap | 3 | auth before registry resolve (or explicit governing-policy exception) | existence-oracle test |
| X-24c | gap | 3 | `MaxBytesReader` on all REST decodes + oversized test | oversized-body test |
| X-24d | gap | 5 | strict decode + EOF check (15.2 explicitly: not `dec.More()` alone) | trailing-garbage test |
| X-24e | gap | 5 | mock records actual tenant; cross-tenant attribution test | attribution test |
| X-24f | gap | 4 | bound/evict `runs` map (durable IDs authoritative) | `TestRunRoutingMapBounded` (new) |
| X-24secrets | gap | 5 | audit secret-pattern scan in one E2E test | scan test |

## 11. F-01–F-14 residual dispositions (Gate 1)

Every residual without an X-ID gets a task, test, or documentation
disposition — a matrix update alone is not sufficient.

| Origin | Residual | Disposition (WS / owner) |
|---|---|---|
| F-03 | stale-resume-after-takeover interleaving untested | ADD `TestStaleResumeAfterTakeover` (new, barrier): stale worker resumed post-takeover cannot send or overwrite (WS-B, M3) |
| F-03 | `Settle` unaudited/unexposed | ADD `effect-settled/<outcome>` audit (§7/M3) + `TestSettleAudited` (new); expose via operator CLI `settle` verb with canonical auth (doc-spec §12 — a test-only API does not constitute exposure) (WS-B) |
| F-02 | lazy-expiry E2E with real workflow missing | ADD `TestLazyExpiryEndToEnd` (new): gated probe run, approval expires, expiry observed + settled via sweep (WS-D, M2) |
| F-02 | `settleDenied` "confirmed down" overclaims cooperative cancel | CORRECT wording to "cancel requested; in-flight steps complete cooperatively" + assert in `TestDenialCancelsRun`, `TestExpiryCancelsRun` (kept) (WS-D, M6) |
| F-10 | signal-after-timeout | ADD `TestSignalAfterTimeout` (new, WS-D, M4) |
| F-10 | signal-then-kill | ADD `TestSignalThenKillRecovery` (new, kill/recovery, WS-D, M4) |
| F-10 | signal while process group unavailable | ADD `TestSignalUnavailableGroup` (new, routing/recovery, WS-E, M4 — single named owner) |
| Prompt 14 §F | cooperative-cancel reality undocumented | ADD ops-doc line: "Cancelled workflows stop at the next step; executing steps are not interrupted (SDK `CancelWorkflow`)" (`docs/phase1/classd-operations.md`) |
| §H-7 | no wait→revoke→refuse E2E | ADD `TestWaitRevokeRefuseEndToEnd` (new, via §4.1 setter API): durable wait, grant revoked mid-wait, post-wait claim refused + audited (WS-D, M3) |
| §H-11 | no E2E cancel-during-`ATTEMPTING` | ADD `TestProbeCancelDuringAttempting` (new, probe E2E variant of `TestCancelDuringAttempt`) (WS-B, M3/M6) |
| §H-10 | timeout/expiry conflated | COVERED by F-02 lazy-expiry E2E (distinguishes Recv-timeout wake from approval expiry) |
| §G | `TestProbeReplayDeterminism` name overclaims | RENAME to cross-run equivalence + doc note (WS-G-adjacent, test-only) |
| §J | audit secret scan | COVERED by X-24secrets row (§10) |
| §K | down-file index-schema nit | FIX down-file qualification (rollback behavior only; up history untouched) + down/up cycle test (WS-F migration pass) |
| §K | advisory-lock 32-bit `hashtext` | DOCUMENT as liveness-only (false-sharing serializes, never corrupts); no change (LOW observation, out of X-scope) |
| §L | OSI-3 future owner = preflight (wrong) | FIX matrix line: owner is registry/admission design (docs pass, P2) |
| F-07/F-11/F-12/F-05/F-13/F-14 residuals | — | COVERED by X-24d / X-24a / O-5 no-action / G6 no-action / X-17 / X-20 respectively |

## 12. Migration and deployment plan

- **000006** (WS-A identity; forward-only; fresh-install + upgrade
  tested): `attached_run_id` column + `UNIQUE (run_id) WHERE run_id
  <> ''` (X-22/R1) with the CR-07 preflight diagnostic. **000007**
  (WS-F control plane): kill-switch setter function + flip-audit
  trigger `ENABLE ALWAYS` (X-15/D1). Split decided at WS-A: one
  migration per workstream keeps ownership and upgrade behavior
  clear (§16). No other schema changes: approvals, effects, and
  audit tables already carry the needed columns; refusal and
  settlement records are audit-vocabulary extensions.
- **000006 data preflight (CR-07, runs BEFORE the constraint):**
  the migration detects duplicate nonempty `run_id` values and
  fails with an actionable diagnostic listing them (never
  silently picks an owner). Delimiter-bearing IDs are NOT
  scanned or rejected: `ScopeID` percent-encoding is injective
  (both parts `:`-free after escaping, so the join has exactly
  one separator), making escape collisions impossible, and
  legacy delimiter-bearing rows coexist compatibly (corrected
  per L-02 — the earlier "scan identity fields / check escape
  collisions" wording described a reject-on-delimiter design that
  was replaced; see §16). The P1 read-only scans (local: 0
  duplicates, 0 tenant delimiters, 1 expected multi-colon probe
  op ID; Neon: 0/0) were design reconnaissance, not migration
  gates. Upgrade fixtures deliberately contain duplicate run IDs
  (diagnostic asserted) and delimiter-bearing IDs (upgrade
  asserted to SUCCEED). No shared/production Neon mutation.
- **Roles/REVOKEs (X-15)** ship as a documented deployment step, not
  migration content (roles are project-global on Neon; the migration
  stays branch-local): restricted app role (`SELECT` on
  `kill_switches`, full app-schema otherwise), operator role (setter
  `EXECUTE`), local compose support, explicit production role
  instructions. P2 proves the mechanism over genuine restricted
  connections (ephemeral test roles + the actual runtime role —
  Z CR-06 standard). **Recorded residuals:** production must run
  the app as the restricted role (startup rejects unsafe role
  configuration instead of silently running elevated); a table
  owner/superuser can disable the trigger mechanism (it is NOT
  "unbypassable" — accurate wording per CR-08); trusted in-process
  miniapp code shares the serving process (plan4.2 trust model:
  untrusted code goes to Sprites).
- **Control-plane credentials (CR-08/R-06):** the serving process
  MUST NOT load or retain the control DSN. The operator `switch`
  verb is a separate command loading `MINIAPP_CONTROL_DATABASE_URL`
  (new `_FILE`-pair env var, same semantics as `DATABASE_URL`;
  documented in `ENV_VARS.md` in P2) and authenticating the
  operator via canonical API keys; actor/reason/scope attribution
  is validated server-side and cannot be forged by an unauthorized
  SQL caller. The ordinary runtime connects as the restricted
  role, including in local integration tests.
- **X-19 compatibility:** the escape is identity for IDs without
  `%`/`:`; collision-pair test covers the new behavior. Legacy
  raw multi-colon rows are ORPHANED on re-attempt, never matched
  again: a post-upgrade lookup computes the escaped PK, misses
  the raw row, and inserts a fresh row (corrected per L-03 — the
  earlier "lookups keep working" was strictly false). This is
  safe ONLY because no production rows exist (test databases
  re-seed; old invocations are terminal and never re-attempted).
  Standing rule: any future encoding change with production data
  present requires dual-read or backfill before cutover.
- **Operator `settle` verb (F-03 Settle disposition, doc-spec only —
  P2 implements):** `miniapp settle --op <business-op-id> --to
  <confirmed|failed|not-applied> --note <evidence>`; authenticates
  the operator via canonical API keys; invokes the predicated
  `UNKNOWN→RECONCILING→terminal` pair (§7/M3) over the normal
  runtime connection (effect tables are app-writable; no control
  DSN needed); refuses non-`UNKNOWN` rows; writes the
  `effect-settled/<outcome>` audit with the evidence note. A
  test-only Go API does not constitute exposure — the verb is the
  operator-accessible surface.

## 13. Test and evidence discipline

- **Digest method (C1):** sorted `sha256sum` manifest over all files
  except `.git`, digest = SHA-256 of the manifest; exact command in
  the P0 register row; manifests preserved under
  `docs/evidence/digests/`. P2 formalizes it as a tested script
  (WS-I); method stays byte-identical.
- **Red/green snapshots (C2):** isolated Git worktree at the recorded
  pre-remediation digest (or equivalent preserved snapshot); red and
  green runs use separate database names; both digests recorded;
  active tree untouched. The SAME discriminating test source runs on
  both sides, so red→green identifies the implementation change.
- **Source-scope digest (CR-11):** in addition to the full-tree
  digest, P2 binds a source-scope digest covering executable content
  under test (Go implementation + tests, SQL/migrations, CI
  workflows, scripts, fixtures/testdata — excluding docs and
  evidence). Red/green runs record all four: implementation source
  digest, test-harness digest, full-tree digest, exact command +
  environment. P0→P1 manifest delta recorded precisely (directive
  §3.1 faq resolution): both owner-edited files were already present in
  the P0 manifest — `app_framework.md` as tracked-modified (+128
  lines mid-P1) and `faq.md` as pre-existing untracked (hash
  changed mid-P1; no git pre-image exists, so its content delta is
  not reconstructible). The only ADDED file was the contract
  itself, hence 254→255. Current `git status` confirms: `faq.md`
  remains untracked and unignored; `app_framework.md` remains
  tracked-modified.
- **SDK identity (C3):** cite pinned `v1.9.0` + source digest
  `308583f2...` (manifest preserved); machine-local paths stay out
  of the register.
- **WS-H one-liner:** CI provisions pinned PostgreSQL + runs the
  sanctioned `go test -p 1 ./...` (plus race) with DB env —
  validated by a local CI-equivalent run; remote execution stays
  unobserved, so G1 row 12 stays PARTIAL.
- **WS-I one-liner:** every new evidence entry carries pre/post-run
  digests; historical undigested claims are marked irreproducible,
  never retrofitted.
- **Concurrency:** barrier synchronization + separate connections,
  never sleeps; `-race` on the race-sensitive suites; statement
  timeouts so deadlocks fail tests instead of hanging them.
- **Process kills:** genuine kill/restart harness only (15.2 §9.3);
  DB invariants need real-PostgreSQL evidence; analyzer rules may
  use unit tests at their correct boundary.
- **Contract corrections are never silent (directive §3.2):** If P2
  discovers a contract inaccuracy, it records the correction in
  this document with both the original claim and the corrected
  claim, and resumes with the corrected discipline. Contract
  inaccuracies are not silently reconciled. Protected
  architecture/review documents are never edited to make evidence
  appear consistent; conflicts surface with provenance intact.

## 14. Standing boundaries (restated, approval §4)

No `lead-followup` implementation, manifest, registration, execution,
or ADR approval. No commits, pushes, or remote mutation. No edits to
protected architecture/review documents. Mock-only channels — no real
provider writes or integrations. No weakening of existing tests
(skip-fast gating only). Disposable databases only — no destructive
experiments against shared/production Neon resources. Forward-only
migrations. Unresolved owner decisions surfaced, never invented.

## 15. P1 exit self-check

| Condition (approval §5) | P1 evidence in this document (amended) |
|---|---|
| Root-cause assessment | §1: RC-1a–d + RC-2 ratified with file:line evidence; R1–R3 refinements (unchanged) |
| Full coverage | §10: X-01–X-24 + §11: every F-residual assigned; every row carries named tests (CR-05) |
| Machine ownership | §3: every finding mapped; none ownerless; per-machine terminals/invariants/prohibitions added (CR-05) |
| Transition rules | §2: all six enumerated and enforceable (unchanged) |
| Contract review | SATISFIED at document level by the consolidated P1 review (`prompt15.2_p1_review.md`); supersession resolved (15.2 governs); P2-exit independent implementation review still mandatory |
| Submission choke point | §5: audit (2+2 call sites, no bypass) + proof strategy + contingency; dangling-row adoption demoted to defensive/legacy (R-03) |
| X-11 mechanism | §8–§8.3: sweeper selected; conditional guarantees, 6-step expiry, tx boundaries, ops spec (CR-06/R-05/Z CR-04) |
| X-08 semantics | §7/M5+M6+§10: genuine attach + DETACH-ONLY owner decision + 3 unconditionals + 6 enumerated checks (CR-01/R-04) |
| X-17/18 acceptance | §13 one-liners + §10 rows + source-scope digest (CR-11) |
| Evidence provenance | §0 bindings + §13 methods; P0/P1-amend rows in register; preflight scans recorded (CR-07) |

**STOP — P2 gated on owner acceptance of these amendments.** No
source file was modified; only this document and evidence changed.

## 16. P1 amendment log (directive §12)

| Amendment | What changed | Why | Affected rows | Tests | Outstanding decisions / conflicts |
|---|---|---|---|---|---|
| CR-03/R-01 | `ATTEMPTING` = point of no return; full-revalidation tx; `CANCELLED` reassessed as withdraw state; `in_flight` vs `withdrawn` split | Claim-INSERT alone left a cancelable-yet-sendable window | §4, §7/M3+M6, §10/X-05+X-23 | `TestCancelBetweenPendingAndAttempting`, `TestPendingWithdrawnOnCancel` (new) | none |
| Amendment 5 | Pinned-snapshot grant protocol (§4.1) + named no-hot-reload residual | Grants are in-memory; closes the in-process race explicitly | §4.1, §7/M3, §11/§H-7 | setter-API E2E + revocation-race test | cross-replica coherence = deployment property (recorded residual) |
| CR-04/R-02 | Fence inside authorize tx; trusted context; extended lock order | Pre-send check repeated check-then-act | §4–§4.2, §7/M3+M5 | `TestSupersededGenerationCannotAttempt` | none |
| CR-02/R-03/Z CR-05 | Park reordered 1→2; dangling-row defensive/legacy + fixture tests | AB-BA vs Cancel; atomic Park cannot dangle | §4.2, §5 step 3, §7/M1, §10/X-01 | adoption leg from fixture | none |
| CR-01/R-04/Z CR-01 | Attached coverage (settle/Status/S3) + 6 cancel cases + DETACH-ONLY | Attached rows were unsettleable; cancel undefined | §7/M1+M5+M6, §8/S3, §10/X-08 | 7 CR-01 tests (new) | none — DETACH-ONLY owner-approved |
| CR-05 | Terminals/invariants/prohibitions; Check-live row; all tests named; stale refs fixed | §7.1/§7.7 completeness | §3–§11 passim | 20+ newly named tests | none |
| CR-06/R-05/Z CR-04 | Conditional guarantees; 6-step expiry; tx boundaries; ops spec; §15.1 cite; WS-G analyzer rule | Bound overstated; S2 order unwritten | §8–§8.3 | `TestLazyExpiryEndToEnd` covers expiry path | none |
| CR-07 | Preflight spec + local/Neon scan evidence + fixture/diagnostic rule | 000006 must not blind-apply constraints | §12, register rows | upgrade fixtures with dups/delimiters | none (both scans clean) |
| CR-08/R-06/Z CR-06 | Accurate trigger wording; serving excludes control DSN; startup role rejection; genuine-connection proof | "Unbypassable" false; credential in serving process | §7/D1, §10/X-15, §12 | `TestKillSwitchOperatorVerb`, `TestKillSwitchFlipAudited`, `TestRuntimeRoleIsRestricted` | prod role config = deployment residual |
| CR-09 | X-09 exact 5 steps; dead-letter class + SDK-verified resume; `quarantined` mapping | Weaker alternative offered; dead-letter mishandled as failed | §7/M4, §10/X-09+X-10, CR-05 invariants | extended `TestForeignVersionNonterminalVisibility` | none |
| CR-10 | 5-way signal taxonomy; `accepted` live-only | Terminal runs misreported success | §6, §7/M4, §10/X-06 | `TestSignalTerminalNoOp`, `TestSignalUnavailableFamily` | none |
| CR-11 | Source-scope digest; 4-part red/green binding; manifest delta | Full-tree digest too coarse for red/green | §13 | digest script test (WS-I) | none |
| Settle surface | CLI `settle` verb doc-spec (canonical auth) | Test-only API ≠ exposure | §7/M3, §11/F-03, §12 | `TestSettleAudited` | none |
| P2-open §3.2 | Corrections-never-silent rule added to §13 | Directive requires the standing rule | §13 | n/a (process) | none |
| P2-open §3.1 | faq manifest wording resolved precisely | "Untracked" misread as "absent from P0" | §13 | n/a (wording) | none |
| P2-open §3.5 | §4.2 completed: claim/attach/takeover/retry/report paths | Omitted paths left order claims unproven | §4.2 | existing barrier tests extended | none |
| P2-open §3.3 | Settlement made atomic; `RECONCILING` removed from machine; idempotent Settle + audit backfill | Two-tx settle strands on crash-between | §7/M3, CR-05 invariants | `TestSettleCrashConverges` (new: rollback, retry, idempotency at tx boundaries — §16.1 for the full record) | pre-existing `RECONCILING` rows: test DBs only, none in prod (no prod data exists) |
| P2-open §3.4 | Quarantine lifecycle specified (resume/fail/status) | Dead-letter needs operator transitions, not just mapping | §7/M4, §10/X-10 | 5 quarantine tests (new) | none |
| WS-C §9 correction | Approver/audit use field+action-qualified seams (not interface wraps); 4th field on Executor; final 19-op inventory | Original seam paragraph assumed interfaces that do not exist (concrete `Approver`, `Shutdown` outside `AsyncExecutor`) | §9 | op inventory tests (all 19 ops) | none — design intent unchanged |
| WS-A carry A | Fourth seam traced explicitly (owner M4, purpose `executor-shutdown`, §9 point) | Exit review required owner/purpose/points record | §9 | `TestExecutorNilTestFault` + `TestFaultExecutorShutdown` | none |
| WS-A carry B | `TestFaultAttemptBegin` pins per-phase state (PENDING held through contention, 0 sends) + sleep-free structure | Exit review required deterministic phase proof | — (test) | the test itself | none |
| WS-A 000006/7 | Migration split by workstream (identity vs control plane) | Clear migration ownership per §20 | §12 | `TestMigration000006*` (fresh, dup-diagnostic, delimiter-compat) | 000007 lands in WS-F |
| WS-A Rule-3 interpretation | Choke requires LIVE linkage (pending/granted), not granted-only | Original §2/§5.3 wording ("granted approval linkage") forbids the reviewed park-submit-wait flow: park submits bound to a PENDING approval; granted-only would force an architecture redesign (submit-on-grant, obsoleting the signal bridge and sweeper S1) — explicitly forbidden. Prompt 15.2's mechanics (S1 redelivery to waiting runs, X-02 submit-if-missing-before-poll, workflow wait + fail-closed-on-missing) all presume pending-linkage submits. Reading "ungranted" as "not covered by a live approval process" | §2 Rule 3, §5 step 3 | X-01/X-02 discriminators (submit bound to pending linkage; missing/denied/expired refused) | OWNER-APPROVED with narrowing (wsc_exit_a_review_plan.md): two permissions recorded verbatim in §2/§5; effect-boundary revalidation specified now, implemented in WS-B |
| WS-A Rule-3 amendment | Two permissions normative (workflow-start vs business-execution) + authorize-tx verification list | Exit review required the narrowed amendment + owner decision | §2, §5 | entry-path wait proofs (Phase 2); effect test lands in WS-B | none — WS-A proves admission+waiting only, never claims the effect guarantee |
| WS-B advisory | §4 advisory rule corrected to outermost-first + no-cycle proof (was: "leaf, never held while acquiring") | Old wording reversed the actual Class A order (lock held across handler `Perform` calls); owner-approved exact wording adopted (wording: `docs/prompt19.md` §7 quotation; authorization: `docs/prompt17_plan_approved.md`) | §4 | call-graph proof (single acquirer, first statement; no other acquirers) | none |
| WS-B L-01 | `claim-response` op implemented as specified (§9 row added) | Was specified but absent (review L-01) | §9 | `TestFaultClaimResponse` (one-shot, replay convergence, single run, audit coherence) | none |
| WS-B L-02 | §12 preflight wording: dup-only detection + delimiter compat | Described a replaced reject-on-delimiter design | §12 | existing migration tests (dup-diagnostic + delimiter-compat-SUCCEEDS) | none — wording only |
| WS-B L-03 | §12 compat: legacy raw rows orphaned on re-attempt + dual-read/backfill standing rule | "Lookups keep working" strictly false for multi-colon legacy rows | §12 | existing compat test; no prod rows exist | future encoding changes gated |
| WS-B O-05 | `testkit` doc notes the serial `-p 1` requirement | Methodology was tribal knowledge | testkit/doc.go | none (doc only) | none |
| WS-B L-02 | §12 preflight wording: dup-only detection + delimiter compat (was: reject-on-delimiter scanning) | Contract described a design that was replaced; migration never scanned delimiters | §12 | existing `TestMigration000006*` (dup-diagnostic + delimiter-compat-SUCCEEDS) | none — wording only |
| WS-B L-03 | §12 compatibility: legacy raw rows orphaned on re-attempt + dual-read/backfill standing rule (was: "lookups keep working") | Strictly false for multi-colon legacy rows under the escaped PK | §12 | existing compat test (upgrade succeeds); no prod rows exist | future encoding changes gated on the standing rule |
| WS-B advisory | §4 advisory rule corrected to outermost-first + no-cycle proof (was: "leaf, never held while acquiring") | Old wording reversed the actual Class A order (lock held across handler `Perform` calls) | §4 | no new test (call-graph proof; covered by existing Class A + effects suites) | none |
| P20 MEDIUM-2 | Class A single execution: advisory xact claim + single-tx handler/completion + savepoint handler-error semantics + execution-in-progress conflict | Concurrent granted polls / duplicate dispatches double-executed handlers; crash-after-handler-commit double-applied on recovery | §4, §5 | T1/T2/C1/C2/C3/C4/T5/T6 (new) + HIGH-1/M-01/full-suite regressions | non-transactional handler side effects stay outside the guarantee (ledger boundary); post-commit audit still needs X-12 backfill |
| P20 MEDIUM-2 §4 | No-cycle proof extended to two ordered advisories (claim→entity, single site, try-claim never waits) + savepoint-claim survival | Second advisory lock added by the execution claim | §4 | entity serialization suites + claim contention tests | none |
| P20 MEDIUM-3 | prompt17:144 + plan-review mirror corrected: §4 authoritative, advisory-last struck, authorizeAttempt takes no advisory lock, outermost-first preserved | Binding work order contradicted corrected §4 (deadlock-prone direction + inapplicable lock) | prompt17.md, prompt17_plan_review.md | repo-wide grep: no advisory-last instructions remain in governing docs (historical reviews/writeups describing the contradiction preserved as records, listed in evidence) | none |
| P22 P21-002 | Pending semantics fixed structurally: adoption marks pending→running atomically; pending ⟹ unsubmitted, so checkLive/cancelIfLive/syncIfLive keep excluding pending and authorizeAttempt treats pending as ineligible | Specified stuck state (pending+workflow) proven unreachable — approval gate refuses pre-linkage submits, park atomicity couples approval to waiting — but the invariant was argument-only | §4.2, CR-05/INFO-1 (+checkLive listed) | adoption-flip red/green + 4 fail-closed characterization suites + full-suite regressions | none; Class A pending (never adopts) untouched by construction |
| P25 Gate-1 corrections | P24-001 commit-on-decision withdrawal with same-tx paired audit; P24-002 attempt-fenced outcome transitions; P24-003 lock-free Settle attribution (§4.2 "5 only" now true); P24-005 option (a) autonomous exception + independent trust-modes admission; P24-004 M5 probe-scoped deferral + enforceable activation trigger; P24-007 claim-time framework minting; P24-008 capability-membership prerequisite; P24-009 AST policy guard; P24-010 state-qualified contention hint; P24-011 phantom-binding terminal taxonomy | Gate 1 review (adv_review_prompt24) found 2 HIGH + 1 lock-order + 2 dispositions + 6 LOW/INFO against the Stage 3A authorization core | §4.2 Settle row, §7/M3 Authorize/Withdraw/Send rows, §7/M5, §7/M3 P24-008 prerequisite, §16 | `TestP25POC1*`, `TestP25TerminalWithdraw*`, `TestP25POC2*`, `TestP25Settle*`, `TestP25M1–M7`, `TestP25Mint*`, `TestP25Contention*`, `TestP25NoRows*`, `TestP23GuardNegativeControl` (all red/green or guard-pinned per finding) | empty-TrustModes manifests keep legacy demand-linkage (declare modes for coherence); mint-failure branch defensive (unforceable); pure map aliasing beyond the AST guard (documented) |

No new architectural conflicts discovered. No owner decision
invented. OpenCode remains NOT RUN/BLOCKED (unchanged, non-blocking).

### 16.1 Removal of RECONCILING — substantive record (directive §3)

**What changed.** `RECONCILING` was removed from the M3 effect
state machine. Operator settlement now performs an atomic,
single-transaction transition from `UNKNOWN` directly to the
selected terminal outcome (`CONFIRMED`, `FAILED`, or `NOT_APPLIED`),
including the attributable `effect-settled/<outcome>` audit record
via the idempotent Settle path (same (op, outcome, note) converges;
different outcome/note conflicts).

**Why.** The previous `UNKNOWN → RECONCILING → terminal` pair ran
in two transactions with the target outcome held only in memory: a
crash between them stranded a `RECONCILING` row whose intended
outcome was unrecoverable, with no retry/adoption semantics. The
intermediate state added no independently actionable operator
meaning. A single predicated transaction ensures crash-before-commit
leaves the effect `UNKNOWN` (operator retries) while a committed
settlement leaves both the terminal outcome and its audit durable
(audit backfill on idempotent retry covers commit-then-crash-before-
audit).

**Tests.** `TestSettleCrashConverges` exercises the actual
settlement failure boundaries: (1) failure at the audit-insertion
boundary before commit → no partial settlement; subsequent Settle
converges; (2) loss of the caller's response after commit → repeat
observes the committed outcome without duplicating the audit;
(3) repeated identical Settle is idempotent; (4) conflicting
settlement cannot overwrite the first committed decision. An
explicit audit-backfill path is tested directly, never inferred
from a clean-path settlement.

**Preserved guards.** `UNKNOWN` stays non-retryable for provider
attempts until an operator settles it; `UNKNOWN → ATTEMPTING`
direct stays prohibited; business-attempt retry passes the full
authorization protocol; removing `RECONCILING` weakens nothing
about blind retries.
