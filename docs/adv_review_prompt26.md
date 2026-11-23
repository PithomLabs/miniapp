# Prompt 26 — Independent Gate 1 Targeted Re-Review

**Target:** Prompt 25 Gate 1 corrections for `docs/adv_review_prompt24.md` findings
**Report path note:** prompt26 §2/§9 name `docs/phase2/prompt26-gate1-targeted-rereview.md`;
the owner-directed path `docs/adv_review_prompt26.md` is used instead.
**Verdict: CLEAR** — both HIGHs independently verified closed with discriminating red/green
evidence; C2 restored; P24-005 conforms to the owner decision on both discrepancies; deferrals
accurate; LOWs closed or bounded; traceability corrected in §7 with a tracked rewrite.
Gate 1 is released to resume the remaining Stage 3A work.

## 1. Overall verdict and rationale

**CLEAR.** Every Gate 1 correction was re-verified against source, tests, and independent
PG19 execution — not against the implementation report's claims:

- P24-001 (HIGH): the withdraw+audit now commit atomically via the decision-commit
  restructure; POC-1 cancel-first reproduced red (`PENDING` + false withdrawn audit) and
  green (`CANCELLED` + exactly one truthful `effect-withdrawn`); the authorize-first order,
  repeat, lost, and quarantined legs all behave per the withdraw rule.
- P24-002 (HIGH): all outcome transitions fenced by `attempt_id`; all three late-outcome
  flavors reproduced red (stale win + superseded live attempt + wrong receipt) and green
  (stale superseded, live lands with its own receipt/attempt).
- P24-003 (C2): Settle attribution is a lock-free immutable-column read; the smoking-gun
  waiter + SQLSTATE 57014 reproduced red; effect-row serialization pinned green. Schema
  independently audited: zero FKs, zero triggers; identity columns immutable across all six
  UPDATE sites.
- P24-005: the persisted-trust admission + linkage exception implement owner option (a)
  exactly, with all 7 owner-required matrix outcomes. Discrepancy 1 (auto+denied-present →
  authorized) conforms to the owner disposition's twice-scoped anti-spoofing rule; the
  contradicting sentence lives in the agent plan/plan-review, not the work order —
  [P26-001](#p26-001) (LOW) requires a one-line owner confirmation, not code. Discrepancy 2
  (empty-mode legacy) is compatible but unpinned — [P26-002](#p26-002) (LOW) requires the
  matrix leg.
- P24-004/P24-008 deferrals recorded accurately with enforceable triggers; P24-006 (gates
  green), P24-007 (mint + 5 tests), P24-009 (AST guard + negative control), P24-010 (message +
  matrix), P24-011 (terminal taxonomy; audit skip structurally forced by schema NOT NULL)
  all verified closed.
- C1–C7: the report's §11 relabels C3–C7 ([P26-003](#p26-003), LOW — rewrite required, true
  table published in §7). No constraint actually regressed; C6's fixture deferral stands
  explicitly unproven.

Full battery green on exact-parity PG19 (all packages incl. dbos/classd, race subset,
checks.sh). No CRITICAL or HIGH remains; no new MEDIUM. The three LOWs are tracked with
exact closure criteria; none reopens a completed foundation.

## 2. Baseline, digests, manifest, environment

- HEAD `1a67ba22b50e2ff54568db308a34d75b14ed2a0d` (main); worktree matches the report's
  inventory (31 modified + 29 untracked at report time; current adds only `docs/prompt26.md`
  and this review). No commits/pushes by any party; this review wrote only this file.
- Source digest `e4b517e85026e8ff558fa2ba5a57eecc59028a12f3ac236d155f703a032837d8` —
  reproduced EXACTLY via `bash scripts/digest.sh source .`.
- Tree digest `dab311379e7e54e8035413a04f98fb1c7041f5f25647c42fb2d2816db0d14b78` —
  reproduced EXACTLY via `bash scripts/digest.sh tree` on the manifest-era file set
  (final tree minus the postdating report and minus `docs/prompt26.md`, which arrived after
  the manifest). Note: the recorded 381-entry manifest is a *superset inventory* (includes
  the report row + `docs/evidence/digests/`, with `./` path forms) while the digest covers
  the script-defined set — every manifest entry's file hash verifies against current bytes,
  so the inventory is true; only the §1 wording ("reproducible by moving this file aside")
  omits that post-manifest arrivals must also be set aside. Expected, not a defect.
- All 9 test-file hashes verified byte-exact (`47937ede…` … `83c8feb6…`); pristine sources
  verified genuine pre-fix bytes (`f3e0ebb6…` has the old `Withdrawn` shape, no decision/
  trust/phantom code; `900fb6bc…` has no attempt predicate/loader/mint/phantom code);
  p24001 red-vs-final diff verified byte-identical to pristine→current (red1 = pristine).
- Environment: Go 1.27.3 linux/amd64, `GOWORK=off`; PostgreSQL `19.0
  (Debian 19.0-1.pgdg13+1)` in a disposable container whose inspected image equals the
  required `postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
  (RepoDigest == container image, verified); fresh `miniapp_dev`/`miniapp_red`/`miniapp_dbos`
  databases. The long-lived dev container was never touched by this review (the implementers'
  use of it is their methodology note, not mine) and the review container was removed after.
- Pre-fix snapshot tarball `471cc3d5…` is not in the repo (was `/tmp`, reaped) — the durable
  path (pristine files + diffs + recipes) substitutes for it and was validated end to end
  below. No other inaccessible artifact.
- Worktree-vs-report delta: the report counts 31 modified paths; this review observes 32.
  The extra path is `docs/faq.md` (+64/−1 Q&A append, mtime 23:18 — after the manifest at
  22:43 and after `docs/prompt26.md` at 23:01), edited by activity outside this review
  (this review wrote only its report; test runs don't touch the tree). Out of review scope;
  every manifest entry otherwise verifies byte-exact. Current tree = manifest tree +
  `docs/prompt26.md` + the `faq.md` append + this report.

## 3. Findings register

| ID | Severity | Subject | Disposition |
|---|---|---|---|
| P24-001 | HIGH (was) | Withdraw rollback + false audit | CLOSED — verified §5.1 |
| P24-002 | HIGH (was) | Stale overwrite | CLOSED — verified §5.2 |
| P24-003 | MEDIUM→closed | Settle 5→1 inversion | CLOSED — verified §5.3 |
| P24-004 | MEDIUM→deferral | M5 scope | Accurate bounded deferral — §5.4 |
| P24-005 | MEDIUM→decision | Gated+autonomous | Conforms to owner decision — §5.5/§6 |
| P24-006..011 | LOW (were) | Cleanup items | All closed/dispositioned — §5.6 |
| P26-001 | LOW (new) | Plan-review hostile sentence vs owner scope | Needs one-line owner confirmation — §6 |
| P26-002 | LOW (new) | Empty-mode residual unpinned | Needs one matrix leg — §6 |
| P26-003 | LOW (new) | §11 C3–C7 relabeling | Needs §11 rewrite; true table in §7 |

No CRITICAL, HIGH, or new MEDIUM. Observations (no finding): manifest-vs-digest
methodology note (§2); guard same-name false-positive shape (errs loud, zero instances);
gate-channel double-close shape (fails loudly); reconcile unknown-capability map rejection
(pre-existing quirk, fail-closed).

## 4. Test commands and observed results

CWD repo root (reds in `/tmp` copies), `GOWORK=off`, PG19 above. `gofmt -l .` empty;
`go vet ./...` clean; `scripts/checks.sh` ALL-CHECKS-PASS. Focused: all P25 tests green
(POC-1 ×2, withdraw internals ×4, POC-2 ×3, Settle ×2, M1–M7, Mint ×5, NoRows ×2,
Contention ×2+matrix, auth guard/snapshot ×5). Affected suites + FULL `go test -p 1 ./...`
all ok (dbos 37.9s, classd 39.7s, zero skips beyond design entrypoints); `-race`
effects/runtime/approvals green.

Red reproductions (recipes from `docs/evidence/p25/README.md`, test bytes hash-verified
identical in each red tree): red1 — CancelFirst + Won FAIL with `row state after terminal
refusal: PENDING (want CANCELLED)`; LostRepeat/LostNotApplied/Quarantined/AuthorizeFirst
PASS (characterization, correctly labeled). Red2 — all 3 flavors FAIL (A err nil,
C superseded, wrong receipts). Red3 — TakesNoInvocationLock FAILS with the smoking-gun
`FOR UPDATE` waiter + SQLSTATE 57014; Serializes PASSES (characterization). Red5 — M3/M4
(reason)/M5-auto/M6-auto FAIL for the intended reasons; M1/M2/M6-confirm/M7 + Mint controls
PASS. Red7 — all 5 mint tests FAIL (caller values persisted verbatim). Red10 — message test
FAILS showing the overclaim (matrix helper kept inert, correctly non-discriminating).
Red11 — both FAIL with raw `no rows in result set`. P24-009 — old patterns match 3/7
fixture lines, missing whole-map/delete/range/len exactly as claimed; new guard greens +
negative control greens.

## 5. Finding-by-finding assessments

### 5.1 P24-001 — CLOSED

Source: terminal branch → `terminalWithdrawTx` (predicated flip + won-conditional in-tx
audit via `InsertTx`) → commit → typed refusal post-commit; commit failure surfaces as
infra, never a withdrawal. `withdrawIfPending` returns (bool, error) with no swallowing;
`RefusalError.Withdrawn` removed (zero non-test references); non-terminal refusals keep the
post-rollback `effect-refused` path. Repeat-after-cancel: flip loses → in-tx `effect-refused`
+ commit (no resurrection, no second withdrawal). Quarantined/mismatch/transient/wait/
contention/infra never reach the withdraw (verified branch order); no ATTEMPTING/provider
after terminal (refusal precedes step 5).

Tests: cancel-first staging independently validated — FIFO row-gate (each waiter observed
via query-needle before the next starts; grant misorder fails loudly on row+audit
assertions, not silently), speculative claim-INSERT gate (speculative tuple blocks `claim`
until Cancel is *joined*, so authorize deterministically reads terminal), exact counts
(1 withdrawn / 0 refused / 0 sends) + repeat assertions. Authorize-first: entered-signal +
ATTEMPTING row double-prove commit-before-cancel; in-flight reporting + CONFIRMED-with-
receipt prove non-recall. Internal legs pin won/lost-repeat/lost-NOT_APPLIED/quarantined
audit pairing exactly; red/green labeling verified correct per leg above.

### 5.2 P24-002 — CLOSED

Source: `transition()` UPDATE gains `AND attempt_id = $7` from the in-hand (pre-transition)
record; zero rows → existing Superseded path, which no caller misreads as success (all six
call sites audited: three `sendAndRecord` outcome legs carry authorize-loaded epochs; three
unknown-capability legs carry claim/adopt-loaded epochs or map-reject). `adoptStale` stays
age-only — re-derived safe independently (a cycled epoch has a fresh timestamp, defeating a
stale-window UPDATE; quarantining a genuinely stale row is the designed disposition at any
epoch).

Tests: all three flavors force the full interleave (entered markers prove per-epoch commit;
distinct framework attempt IDs asserted from the row; aged+quarantined+settled staging
verified step by step; A released first, Superseded asserted; C lands with C's state,
attempt, and receipt; exactly-2-sends via entered markers + channel counter). The
mockcap-rows → channel-counter deviation is sound and documented (per-epoch scripted
outcomes are incompatible with payload-driven modes; entered markers prove execution more
directly). Red/green verified per flavor above.

### 5.3 P24-003 — CLOSED (C2 restored)

Source: full Settle tx re-read — loadTx (plain) → effect UPDATE (5) → `loadIdentityTx`
(plain SELECT of miniapp/version/tool) → audit INSERT → loadTx (plain). `loadInvocationTx`
now used only at authorize position 1 (verified callers). Schema independently audited:
zero `REFERENCES`/foreign keys, zero triggers/rules in all migrations; all six
`UPDATE app.invocations` sites touch only status/run/result/error/timestamps (identity
columns immutable post-claim; claim verified separately) — the unlocked read cannot tear.
No advisory SQL anywhere in the touched paths. §4.2 "5 only" now true.

Tests: TakesNoInvocationLock discriminates (red: smoking-gun waiter + 57014; green: 0.06s
+ correct outcome/note/attribution — the attribution assertion pins the lock-free read's
correctness, not just its lock-freedom). Serializes pins the unchanged effect footprint on
both trees (characterization, correctly labeled).

### 5.4 P24-004 — Accurate bounded deferral

Design §5 + §12.1 and contract M5-row + §7/M5 note verified against all six prompt26 checks:
mock-only named limitation stated; never presented as an implemented guarantee ("explicit
scope deferral, NOT an implemented generation-fencing guarantee"); framework-declared
requirement marker (never inferred from `Intent.Sequence` presence; presence-based
classification marked INTERIM); trusted propagation bound to tenant/invocation/effect;
four-gate activation trigger (verifier + propagation + supersede validation + X-08) with
mechanical admission refusal; fail-closed when active at position 3 with no tx/I-O/advisory.
No code or migration, as approved. (§12.1's placement bullet describes the activated
end-state; the deferral bounds it in three places — acceptable.)

### 5.5 P24-005 — Conforms to the owner decision

Implementation verified against all 7 prompt26 checks: persisted row trust (immutable —
no UPDATE touches `trust_mode`) drives the exception; Intent carries no trust field;
standalone admission refuses manifest-disallowed trust before and regardless of the
exception; gated+Confirm requires full linkage; gated+Autonomous+explicit-modes skips
linkage only; tenant/binding/grants/switches/token/ownership/sequence all still apply;
no caller input can flip persisted trust; terminal short-circuits first. Matrix M1–M7
verified leg by leg (M3/M4-reason/M5-auto/M6-auto discriminate on red; M1/M2/M6-confirm/
M7 correctly characterize). Both discrepancies resolved in §6.

### 5.6 P24-006 through P24-011 — All closed/dispositioned

- P24-006: `gofmt -l .` empty, `go vet` clean, checks ALL-CHECKS-PASS (this review's runs).
- P24-007: unconditional claim-time mint (fail-closed), first attempt authorizes against the
  persisted epoch (line-verified), Intent field deprecated-but-compatible; 5 tests green,
  all fail red with caller values persisted verbatim.
- P24-008: prerequisite recorded in design §11 + contract §7/M3 (verified text); docs-only
  as approved.
- P24-009: AST guard covers subscript/whole-map/element assignment/delete/len/range +
  qualifier-aware (incl. dot-import) literals; full suite green proves zero false positives
  (including the same-name `StrictPolicy.Grants` plumbing, which errs loud-if-colliding —
  noted); 7-line planted fixture all flagged with vacuity guards; aliasing residual
  documented and grep-verified absent (re-verified: only two *Policy-pointer* plumbing
  references exist outside auth, neither map access).
- P24-010: helper qualifies by (status, class) per replay semantics; call site threads
  in-scope `rec.Status`/`tool.Class`; execution-in-progress untouched; stub-sync test
  discriminates on red with the overclaim verbatim; 6-leg matrix covers all live states ×
  both classes (terminal correctly out of matrix — contention unreachable there).
- P24-011: no-rows → terminal binding refusal (authorize) / `ErrPhantomBinding` (Settle);
  only `isNoRows` mapped, genuine errors stay retryable; tests discriminate red (raw noise)
  vs green. Audit skip for phantoms ACCEPTED AS FORCED: `InsertTx` rejects empty
  miniapp/tool and the schema enforces NOT NULL, while phantom invocations have no knowable
  miniapp/tool — no schema-valid non-fabricated audit exists. Full attribution genuinely
  needs the out-of-scope schema change; the residual is bounded and pinned by test.

## 6. P24-005 decision analysis (both discrepancies)

**Discrepancy 1 — auto+denied-present → authorized.** The exact owner disposition
(prompt25.md, stated twice identically) scopes the hostile-approval rule to bypass **"by
changing the caller's trust claim"** / **"through a caller-supplied trust claim"** — an
anti-spoofing rule — and lists 7 required tests, all present with correct outcomes (test 6's
denied/expired/mismatched legs evaluate on the Confirm path). Spoofing is structurally
impossible (no trust field in Intent; row trust immutable post-claim — both verified), and
M5 pins the persisted-trust discriminator. The implementation therefore conforms to the
owner decision exactly. Data-model establishment (not assumption): hostile approvals are
inapplicable to persisted-Autonomous invocations — approvals attach only via park, and all
three park callers (first-claim NeedsApproval, dispatchPending-on-pending, linkage-required
re-entry) imply non-autonomous stored trust, proven caller by caller; the submit choke
likewise never loads linkage for autonomous trust, so the exception mirrors admission
completely. Honoring synthetic hostile rows would let test/legacy residue veto legitimate
execution — skipping is correct, not expedient.
The contradicting "any present hostile approval" sentence lives in the agent plan and its
review (prompt25_plan.md:17, plan-review §5), which broadened the owner rule without
answering the undecided sub-semantics (pending-on-autonomous?) and without authority to
override the twice-scoped work order. [P26-001](#p26-001) (LOW): record a one-line owner
confirmation of the scoped reading and cite it on the M6 leg. No code change.

**Discrepancy 2 — empty-mode legacy residual.** Compatible with the owner decision: the
decision requires *explicit* manifest permission, which empty modes do not give, so authorize
correctly demands linkage; the submit side predates the decision and is explicitly out of
Gate 1 scope. Bounded (undeclared manifests only), recoverable (declare modes — authorize
resolves the live manifest, no record change needed), and consistent with the fail-closed
direction. But unpinned: no matrix leg covers empty modes. [P26-002](#p26-002) (LOW): add
the empty-mode gated+auto → approval-missing leg and record the submit-side-validation
trigger (before any empty-mode production manifest ships gated tools).

## 7. C1–C7 traceability (true table — supersedes report §11)

The report's §11 relabels C3–C7 (C3–C4 as "tx atomicity, grant snapshot", C5 as
"fencing/leases", C6 as "retry semantics", C7 as "terminal authority") against the P21
definitions. This is a traceability error ([P26-003](#p26-003), LOW — §11 rewrite required):
in particular it obscures C6's still-deferred fixture behind the P24-010 message fix. No
constraint actually regressed. The accurate gate status, verified by this review:

| Constraint (P21 definition) | Gate 1 status | Evidence |
|---|---|---|
| C1 no advisory in authorize | HOLDS (re-verified) | Statement-level re-read: plain DML only; no new advisory SQL |
| C2 global lock order | HOLDS (restored by P24-003) | Lock-free attribution; schema audit; discriminating test |
| C3 own authorization tx | HOLDS (preserved) | Signature unchanged; WithTenant checkout; callers pass no tx |
| C4 no new advisory acquirers | HOLDS (preserved) | Exhaustive grep; INFO-2 + AST guards green |
| C5 explicit pending semantics | HOLDS (preserved) | P21-002 flip intact; authorize treats pending ineligible (re-read branches) |
| C6 Class A Perform recovery | MECHANISM SOUND, INTEGRATION UNPROVEN (still deferred) | Contention path intact; fixture + op-ID guidance still outstanding per P24 — P24-010 message work does not close this |
| C7 O-01 closes at effect boundary | HOLDS, strengthened (P24-001) | Terminal refuses + now actually withdraws; orphan cannot send and now terminalizes |

## 8. Accepted residuals and Gate 1 release

Retained residuals (all re-examined): O-01 submit race (effect-time protection + withdrawal
now both hold); 32-bit hash; ledger boundary; X-12; T5-style characterizations; FIFO staging
(loud on deviation, used again soundly in POC-1); aggregate LOW-1 counts; 2-min lease;
INFO-2 raw-SQL bypass (C4 conditional correctly untriggered); miniapp SQL discipline;
M5 probe deferral (now with enforceable trigger); X-10 resumability (re-verified refusing
without withdrawing); empty-mode coherence ([P26-002](#p26-002)); plan-sentence fidelity
([P26-001](#p26-001)); §11 rewrite ([P26-003](#p26-003)); mint-failure branch (defensive,
unforceable — covers `uuid.NewRandom` error + empty-string impossibility); aliasing beyond
the AST guard (documented, absent).

**Gate 1 is released to resume the remaining Stage 3A work** (full authorization/crash
matrix, C6 fixture, revocation tooling, Settle wiring + operator auth, bypass suite).
P26-001/002/003 must land before Gate 2 reliance on the affected records (disposition
confirmation, one matrix leg, §11 rewrite) — small, bounded, non-blocking for resumption.
If a BLOCKED/CLEAR dispute arises on the §11 rewrite: the accurate traceability record is
this review's §7; the report correction is editorial follow-through, not re-verification.
