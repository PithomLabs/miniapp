# Prompt 25 — Gate 1 corrections report

Corrections for the Gate 1 review (`docs/adv_review_prompt24.md`):
P24-001 (HIGH), P24-002 (HIGH), P24-003 (C2 lock order), the
P24-004/P24-005 owner dispositions, and the bounded P24-006–P24-011
cleanup. No further Stage 3A expansion was undertaken.

## 1. Baseline and final tree identity

- Git HEAD: `1a67ba2` ("prompt21"), unchanged throughout.
- Pre-fix snapshot (post-`gofmt -w`, whitespace-only verified per
  file): tarball `471cc3d5b9ac16321aa8e4ad40d009ff546c28d9876b9ec79539c979758b7ba6`
  (64 test files hashed), frozen 2026-11-23T13:45:12Z.
- Hygiene first: 10 unformatted files formatted; 9 verified
  whitespace-only via `diff -w`, 1 (`effects/authorize.go`,
  missing trailing newline) via token comparison.
- Final source digest (stable — covers all executable
  content including tests and fixtures, excludes this
  report): `e4b517e85026e8ff558fa2ba5a57eecc59028a12f3ac236d155f703a032837d8`
  (`bash scripts/digest.sh source .`; reviewer-reproducible
  exactly).
- Final tree digest: `dab311379e7e54e8035413a04f98fb1c7041f5f25647c42fb2d2816db0d14b78`
  (`bash scripts/digest.sh tree .`), computed over every file
  except this report (the report postdates it; reproducible by
  moving this file aside).
- Full-tree manifest (complete inventory, self excluded):
  `docs/evidence/digests/p25-closeout-2026-11-23.manifest.txt`
  (381 files).

## 2. Execution environment (PostgreSQL 19)

- Image: `postgres:19.0` —
  `postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
  (`docker inspect`, RepoDigest == container image).
- Server: `PostgreSQL 19.0 (Debian 19.0-1.pgdg13+1)`,
  container `miniapp-dev-db-1`, database `miniapp_dev`.
- Go: `go1.27.3 linux/amd64`. All DB tests ran with
  `TEST_DATABASE_URL`/`TEST_DBOS_DATABASE_URL` on disposable
  truncating fixtures; serial (`-p 1`) unless noted.

## 3. P24-001 — Withdrawal and audit genuinely atomic (HIGH)

Root cause (confirmed): the withdraw `UPDATE` ran inside a
transaction whose function then returned the refusal error;
`WithTenant` commits only on nil error, so the flip always
rolled back — while `withdrawn=true` unconditionally drove a
separate-transaction `effect-withdrawn` audit.

Correction (`effects/authorize.go`, prompt25 §2 8-step shape):
`authorizeTx` now returns an `authzDecision`. The terminal
branch withdraws (predicated `PENDING → CANCELLED`), checks
affected rows, inserts the paired audit (`effect-withdrawn` iff
won, else `effect-refused`/`invocation-ineligible`) in the SAME
transaction via `audit.InsertTx`, and returns the decision for
commit; the typed refusal (`ErrInvocationCancelled`) is
returned post-commit. Commit failure surfaces as infra failure
and never reports a withdrawal. `RefusalError.Withdrawn`
(provably consumer-free) was removed; non-terminal refusals
keep the post-rollback `effect-refused` path.

Regressions (POC-1 re-implemented, deterministic — no
repetition-as-proof; every order forced with barriers and
per-execution markers):

- `TestP25POC1CancelFirstWithdraws` (`runtime`,
  `47937ede…`): FIFO row-gate staging (Perform liveness read
  queued first, Cancel `UPDATE` behind — both observed waiting
  in `pg_stat_activity` before release) + speculative claim-
  `INSERT` gate (stale PENDING row committed only after Cancel
  joins) forces checkLive(live) → cancel-commit →
  authorize-terminal. Asserts terminal invocation, `CANCELLED`
  row, exactly one `effect-withdrawn`, zero refused pre-repeat,
  zero provider sends; repeat refuses silently pre-claim with
  no resurrection and no second withdrawal.
- `TestP25TerminalWithdrawWon` (`effects`, `7cdb4ee3…`):
  direct won-path proof.
- Characterization (pass both sides by design):
  `TestP25TerminalWithdrawLostRepeat` (authorize-reaching
  repeat audits `effect-refused` once),
  `TestP25TerminalWithdrawLostNotApplied`,
  `TestP25QuarantinedNeverWithdraws`,
  `TestP25POC1AuthorizeFirstInFlight` (opposite forced order:
  provider entered-signal proves `ATTEMPTING` committed;
  Cancel reports in-flight; release completes `CONFIRMED` —
  no recall pretense).

Red/green: identical bytes on the frozen pre-fix overlay fail
with `row state after terminal refusal: PENDING (want
CANCELLED)` (+ the false withdrawn audit present); green
passes all six (0.03–0.13s). 5× flakiness repeat green.
`./effects/` + `./runtime/` suites green.

## 4. P24-002 — Attempt-fenced outcome transitions (HIGH)

Correction (`effects/effects.go` `transition()`): added
`AND attempt_id = $7` from the in-hand record; zero-row
landings use the existing Superseded path. All six callers
audited: authorize-loaded (sendAndRecord ×3), claim-loaded
(`attempt`), adopt-loaded (`takeover`, `reconcile`) — each
carries its true epoch; the predicate converts silent
clobbers into Superseded. `adoptStale` stays age-only
(already safe; uniformity churn rejected with rationale).

Regressions (POC-2, all three late-outcome flavors, scripted
per-epoch provider outcomes with entered/release gates):

- `TestP25POC2StaleConfirmedFenced`,
  `TestP25POC2StaleFailedFenced`,
  `TestP25POC2StaleUnknownFenced` (`effects`, `a6fc35f6…`):
  epoch A blocks in Send → SQL aging → touch quarantines →
  direct `Settle(NOT_APPLIED)` → epoch C reconciles and
  blocks → A released first. Assert: distinct epochs, A
  superseded, C lands (`CONFIRMED` + C's receipt + C's
  attempt), exactly 2 sends. (Send execution is proven by
  entered markers + channel counters rather than mockcap
  rows: per-epoch scripted outcomes are incompatible with
  payload-driven mockcap modes. Documented deviation.)

Red/green: identical bytes on the one-line-reverted tree fail
on all flavors (A err nil — stale win; C superseded; wrong
receipts, e.g. `("", "unknown outcome: p25-epochA-unknown")`
instead of C's); green passes all three. 5× repeat green.
`./effects/` + `./runtime/` suites green.

## 5. P24-003 — Settle lock order (C2)

Correction (`effects/effects.go`): `insertSettleAudit` now
attributes via `loadIdentityTx` — a lock-free plain `SELECT`
of immutable identity columns — instead of the
lock-taking `loadInvocationTx … FOR UPDATE`.

Complete lock-path proof: Settle's tx is loadTx (plain) →
effect `UPDATE` (5) → identity `SELECT` (no lock) → audit
`INSERT` → loadTx (plain). Verified: zero `FOR UPDATE` in
`effects.go`; `loadInvocationTx` used only at authorize
position 1; migrations contain zero `REFERENCES`/foreign keys
and zero triggers; miniapp/version/tool untouched by all six
`UPDATE app.invocations` sites (status/run/result/error/
timestamps only) — the unlocked read cannot tear. §4.2 stays
effect-lock-only; Settle never recreates the
effect → invocation edge.

Regressions (forced both lock-held-first orders, statement
timeouts):

- `TestP25SettleTakesNoInvocationLock` (`effects`,
  `9f936547…`, discriminating): invocation row held
  externally; Settle on a 5s-statement-timeout pool must
  complete. Red fails loudly with the smoking-gun waiter
  (`SELECT … FROM app.invocations … FOR UPDATE`) +
  SQLSTATE 57014; green completes in 0.06s with correct
  outcome + attribution.
- `TestP25SettleSerializesOnEffectRow` (characterization):
  effect row held; Settle queues on its `UPDATE` (marker
  observed both trees), converges after release — pins the
  unchanged "5" footprint.

Settle suite green (`TestSettleAndListUnknown`,
`TestFaultSettle*`).

## 6. P24-004 — M5 owner disposition

Recorded (docs-only, no code, no migration): M5 generation
fencing is probe-scoped — an explicit scope deferral, NOT an
implemented guarantee. Named limitation: the probe performs
no authorization-time generation validation (mock-only).
Enforceable activation trigger: a manifest declaring
sequence-bound effects (framework-declared requirement
marker — never inferred from `Intent.Sequence` presence)
is REFUSED at registry admission until verifier +
trusted propagation (framework-originated,
tenant/invocation/effect bound) + supersede validation +
X-08 barrier test all land. Presence-based classification
marked INTERIM. Applied to
`docs/phase2/authorize-attempt-design.md` (§5 area, §12.1)
and `docs/phase1/invocation-lifecycle-contract.md` (M5
module row, §7/M5 note). Preflight green after each batch.

## 7. P24-005 — Autonomous-trust semantics (option a)

Correction (`effects/authorize.go`): (1) standalone
persisted-trust admission — non-empty `TrustModes` must
contain the persisted trust, else terminal
`trust-mode-mismatch` (refuses regardless of the linkage
exception, so invalid trust can never be laundered);
empty list = unrestricted (legacy; verified compatible
with every existing fixture). (2) Linkage exception:
gated + persisted-autonomous + Autonomous explicitly in
modes skips approval linkage, mirroring the submit choke;
all other checks (tenant/binding/grants/switches/token/
ownership/sequence) still apply. Caller claims cannot
trigger it (`Intent` carries no trust field).

Matrix (`effects/p25_p24005_internal_test.go`, `3154d217…`,
11 legs, all green): M1 granted-confirm authorized; M2
missing-confirm → approval-missing; M3 auto+permits →
authorized (RED refused — discriminating); M4
auto+forbids → trust-mode-mismatch (RED refused with the
wrong reason — reason-discriminating); M5 persisted-trust
discriminator (RED auto leg refused — discriminating);
M6 denied/expired/mismatched/cancelled fully evaluated
on confirm (both-sides characterization) + auto+denied-
present → authorized, pinning the choke-mirror exception
shape (RED refused — discriminating); M7 terminal+auto
→ cancelled-withdraw, never authorized
(characterization). Blast-radius audit: every
trust/manifest pairing in existing suites verified
(empty modes, [A]+auto, [C,A], dry-run never reaching
authorize) — `./effects/` + `./runtime/` suites green.
Contract + design language updated (§7/M3 Authorize row,
design §2 table + lock diagram).

Residual (documented, out of scope): empty-mode gated+auto
keeps legacy demand-linkage (submit admits, authorize
refuses) — coherence requires declaring modes; submit-side
TrustModes validation untouched per the authorize-boundary
scope.

## 8. P24-006 through P24-011 dispositions

- P24-006 (LOW): `gofmt -w` before any evidence; diffs
  verified whitespace-only; `gofmt -l .` empty,
  `scripts/checks.sh` → `ALL-CHECKS-PASS` (after routing
  around a sandbox read-only `GOCACHE` — environmental).
- P24-007 (LOW): claim-time unconditional minting
  (`uuid.NewRandom`, fail-closed on mint failure or empty
  ID); first attempt authorizes against the persisted
  epoch (`rec.AttemptID`), never `Intent.AttemptID`
  (deprecated, wire-compat retained). Tests
  (`423f333e…`): empty/malformed/attacker-chosen/
  colliding/retry — all 5 fail red (caller values
  persisted), all pass green (valid distinct UUIDs,
  retry preserves). Suites green.
- P24-008 (LOW): capability-membership prerequisite
  recorded (design §11 gating prerequisite + contract
  §7/M3 note): authorization must verify capability ∈
  declared set before the first non-mock provider; docs
  only.
- P24-009 (LOW): policy guard migrated to AST (mirrors
  the INFO-2 `go/ast` precedent): subscript, whole-map
  and element assignment, `delete`/`range`/`len` over
  map fields, qualifier-aware `auth.Policy` literals.
  Zero false positives (`faultwrap.Policy{}`,
  `cfg.Grants`, `p.Grants.GrantAllows` all pass);
  7-line planted fixture all flagged; old-matcher
  blindness proven mechanically; control observed to
  fail loudly. Residual documented: pure aliasing
  (`m := policy.Grants`) needs type info — no such
  shape exists (grep-verified).
- P24-010 (LOW): `effect-contention` message qualified
  by record state × tool class (`contentionRetryHint`,
  `rec.Status` + `tool.Class` in scope):
  running/waiting-event Class A instructs new-key
  (same sentence shape as the fixed InProgress
  comment); all other shapes keep same-key.
  `ErrAttemptContention` comment qualified identically.
  Tests: stub-sync invoke (RED shows the overclaim,
  GREEN the new-key instruction) + 6-leg matrix.
  Execution-in-progress message untouched per scope.
- P24-011 (LOW): no-rows → terminal taxonomy:
  authorize maps to `RefusalBindingMismatch` routed
  through the refusal-audit path; Settle maps to new
  terminal `ErrPhantomBinding`; only `isNoRows` is
  mapped (genuine errors stay retryable). Tests fail
  red with raw `no rows in result set` noise, pass
  green. Audit note: phantom invocations are
  unattributable (miniapp/tool unknowable), so the
  refusal path skips per the standing attribution
  rule — pinned by test; full attribution would need
  schema/validation changes (out of scope).

## 9. Red/green provenance (hashes + commands)

Identical test bytes verified (`cmp`) between frozen
red-run copies and committed files for all 9 test files:

| File | SHA-256 |
|---|---|
| `runtime/p25_p24001_test.go` | `47937ede…87d` |
| `effects/p25_p24001_internal_test.go` | `7cdb4ee3…f8` |
| `effects/p25_p24002_test.go` | `a6fc35f6…35` |
| `effects/p25_p24003_test.go` | `9f936547…24a` |
| `effects/p25_p24005_internal_test.go` | `3154d217…982` |
| `effects/p25_p24007_test.go` | `423f333e…eac` |
| `runtime/p25_p24010_test.go` | `d3819f3a…4ac` |
| `runtime/p25_p24010_internal_test.go` | `227bc867…6c0` |
| `effects/p25_p24011_internal_test.go` | `83c8feb6…177` |

Per-finding red trees: full-tree `/tmp` copies
(`p25_red`, `p25_red2/3/5/7/10/11`) differing from
green by exactly the finding's behavioral hunk(s)
(delta verified by diff in each case); P24-009 is
guard-only (blindness proof instead of red tree).
Durable reproduction: `docs/evidence/p25/` (README +
red-vs-final diffs + pristine pre-fix sources; the
pristine-file red path was validated end-to-end and
reproduces the exact red1 signature).

Commands (each with local `TEST_*` URLs on the pinned
PG19): focused `-run … -count=1` per finding (quoted
above), then `./effects/` + `./runtime/` suites per
finding, then §10 battery. No test was weakened; no
red/green pair is claimed where both sides pass —
characterization legs are labeled as such.

## 10. Verification battery results

- `gofmt -l .`: clean. `go vet ./...`: clean.
  `bash scripts/checks.sh`: `ALL-CHECKS-PASS`.
- `go test -p 1 ./... -count=1 -timeout 30m`: every
  package ok, zero failures (effects 1.8s, runtime
  5.4s, dbos 37.7s, classd 39.3s, spikes/testkit/
  transport/mockcap < 7s each).
- `go test -race -p 1 ./effects/ ./runtime/
  ./approvals/ -count=1`: all ok (3.8s/8.6s/2.5s).
  (One parallel `-race` run without `-p 1` failed via
  cross-package truncate contention — the documented
  shared-DB serial requirement, not a regression.)
- `go run ./cmd/preflight --check-all`: 1 pass,
  2 warn, 0 fail, 1 skip (warn/skip pre-existing).
- 5× flakiness repeats green for POC-1 cancel-first
  and all POC-2 flavors.
- Docs: contract §4.2/M3/§16 rows + §16 P25 log entry;
  design M5/approval/capability updates; evidence
  register rows appended.

## 11. C1–C7 compliance updates

- C1 (no advisory in authorize): preserved — no new
  advisory SQL (`transition` predicate, withdraw,
  mint, mappings are plain DML).
- C2 (global lock order): P24-003 restores it —
  Settle is now effect-lock-only (proof in §5);
  authorize order RLock → 1 → 2 → 4 → 5 unchanged
  (trust checks use held row + in-memory manifest,
  no new locks).
- C3–C4 (tx atomicity, grant snapshot): P24-001
  extends atomicity to withdrawal+audit; grant
  snapshot untouched.
- C5 (fencing/leases): P24-002 adds attempt-epoch
  fencing; P24-007 makes epochs framework-minted.
- C6 (retry semantics): P24-010 corrects the
  contention instruction; P24-001 repeats refuse
  without resurrection.
- C7 (terminal authority): P24-001 withdrawal +
  P24-011 phantom taxonomy tighten terminal handling;
  no authorization path weakened.

## 12. Remaining Stage 3A work (NOT started)

Full authorization/crash matrix, C6 fixture, revocation
tooling, Settle wiring, bypass suite, and the remaining
matrix/race/crash/bypass suites from the approved plan —
all blocked until the targeted re-review returns CLEAR.

## 13. Worktree state + confirmations

- HEAD `1a67ba2` (prompt21); 31 modified + 29
  untracked paths (Stage 3A + Prompt 25 work, uncommitted
  for review). Production changes: `effects/authorize.go`,
  `effects/effects.go`, `runtime/service.go`,
  `executor/executor.go` (comment), `auth/policy_guard_test.go`
  + fixture; docs: design note, lifecycle contract,
  evidence register, this report, `docs/evidence/p25/`.
- CONFIRMED: no commit, no push, no staging, no
  deployment, no production-secret access, no real
  provider write (mock channels only), no migration,
  no dependency change, no Neon/OpenCode use.
- STOPPING per prompt25 §11: implementation ends here;
  the next gate is the targeted independent re-review
  (Prompt 26).
