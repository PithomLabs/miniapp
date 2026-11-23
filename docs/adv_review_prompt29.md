# Prompt 29 — Independent Adversarial Review: WS-B Phase 2 Stage 3A Gate 2

**Target:** Prompt28 Stage 3A implementation and evidence (Gate 2), plus the concurrent
Prompt28.1 Settle-operator track reviewed against its own owner decision.
**Report path note:** prompt29 §2 names `docs/phase2/adv-review-prompt28-gate2.md`; the
owner-directed path `docs/adv_review_prompt29.md` is used instead.
**Verdict: CLEAR WITH SCOPED EXCEPTIONS.** The core Stage 3A guarantees — atomic
authorization, single-winner ownership, crash consistency, Class A recovery, revocation
verification, atomic Settle, and no provider bypass — are independently verified by source,
forced-order tests, genuine kills, and full-battery execution, with no CRITICAL/HIGH/MEDIUM
outstanding. Five LOWs are tracked (three evidence-accuracy/hardening items, one decision
supersession broader than the closeout, one trigger note). No unresolved Gate 2 blocker.

## 1. Overall verdict and rationale

**CLEAR WITH SCOPED EXCEPTIONS.** Rationale, per the §15 rules:

- All 12 matrix legs, 9 race cases, 6 crash boundaries, 11 C6 claims, the grant/revocation
  chain, the Settle crash matrix, and the bypass inventory were re-verified by reading every
  test and re-running the battery on exact-parity PG19. No CRITICAL or HIGH finding; no new
  MEDIUM.
- The closeout's evidence classifications hold up under adversarial audit with three
  exceptions: two register rows overclaim (five-point matrix evidence [P29-001](#p29-001),
  guard negative control [P29-002](#p29-002)), and the guard's trust-name list is exact-match
  only ([P29-003](#p29-003)). All are LOW: behavior verified correct; tests and rows need
  tightening, not redesign.
- A concurrent owner-authorized track (Prompt28.1, Option A) implemented the Settle operator
  boundary + CLI + full evidence *during this review*. It conforms to its decision text on
  all 8 enforcement requirements with all 9 evidence items green. This supersedes the
  closeout's STOPPED claim and prompt29's STOP premise — recorded as a decision
  supersession ([P29-004](#p29-004), LOW/doc), not a violation: the closeout was true when
  written (proven by manifest forensics), and the newer owner direction governs.
- C1–C7 revalidated: all hold; the §11 table uses correct P21 labels with the C6 deferral
  explicit (P26-003 resolved).
- Residuals (§13) accepted with one addition: the abandoned-PENDING cross-invocation wedge
  ([P29-005](#p29-005), LOW/doc) — an approved-design trade-off needing one boundary paragraph.

The verdict is CLEAR (not NOT CLEAR) because every exception is bounded, documented, and
non-blocking: no guarantee is compromised, no proof is missing that the gate requires, and
every required correction is editorial or additive test tightening.

## 2. Exact baseline and provenance

- HEAD `a09b7ff2c639f4bdcd61e7e7192c1597187b8037` (main, "prompt26"); no commits by any party
  during review; worktree read-only (this report is the only file added).
- Closeout digests reproduced EXACTLY: source `2d98ee5f…` via `digest.sh source`;
  tree `64278ca0…` via `digest.sh tree` on the manifest-era file set. Manifest entry-by-entry
  comparison: all 412 shared entries byte-identical; post-manifest arrivals enumerated
  (`prompt28.1.md`, `prompt29.md`, `review4.md`).
- Reviewed-tree digests (final bytes, includes the Prompt28.1 track): source
  `21ff485b…`, tree `55585ca6…`. Production diff vs HEAD: 6 files (effects backfill hunk,
  SettleEffect service method, settle CLI verb, main dispatch, hash accessor, /healthz) —
  all additive or single-hunk; test files: 4 modified (comment/expectation only) + 17 new.
- Environment: Go 1.27.3, `GOWORK=off`; PostgreSQL `19.0 (Debian 19.0-1.pgdg13+1)` in a
  disposable container whose inspected image equals required
  `postgres@sha256:bb6e70ad…c60`; fresh `p29test`/`miniapp_dbos` databases. Long-lived dev
  resources untouched; review container removed after.
- Outside activity (prompt's list confirmed, preserved): roadmap/observability/development
  docs, `review4.md`, `roadmap2.md` deletion, `faq.md` appends — all out of Gate 2 scope,
  none touched.
- Mid-review implementation: the Prompt28.1 track (service + CLI + 2 test files + plan/decision
  docs) landed during this review. Provenance forensics (§9) establish the closeout predates
  it; every affected area was re-verified on final bytes (full suite re-run post-landing).

## 3. Findings register

| ID | Severity | Subject | Disposition |
|---|---|---|---|
| P29-001 | LOW | Matrix row overclaims five-point evidence (policy legs lack row/audit asserts) | Add assertions or narrow row |
| P29-002 | LOW | Bypass-guard "negative-controlled with a plant" untrue (no plant exists) | Add planted control (shape proven by this review) or correct row |
| P29-003 | LOW | Intent trust-name guard is exact-match (novel trust field names evade) | Substring match or name registry |
| P29-004 | LOW/doc | Prompt28.1 supersedes closeout STOP + prompt29 STOP premise | Recorded here; addendum exists; no action beyond acknowledgment |
| P29-005 | LOW/doc | Abandoned-PENDING cross-invocation wedge undocumented | One boundary paragraph in guide (+contract pointer) |

No CRITICAL, HIGH, or MEDIUM. Observations (no finding): gate-channel double-close shape
(fails loudly); reconcile unknown-capability map rejection (pre-existing, fail-closed);
`Settled` idempotent-repeat on CANCELLED-row shape (correct, pinned); CLI credential
process-listing limitation (documented as required).

### P29-001 — Matrix evidence overclaim (policy legs)

- **ID/Severity:** P29-001, LOW. **Invariant:** Prompt28 §3 five-point refusal evidence.
- **References:** `effects/p28_matrix_policy_test.go` (kill-switch ×4, tool-declared,
  grant-deny legs); register row WS-A; file header promising "typed classification,
  untouched state, 0 sends, exact audits".
- **Reproducible evidence:** test source reads — legs assert typed error + zero sends only;
  no row-state read, no audit count. Register row claims all five points.
- **Failure mode/impact:** a future audit-mislabeling or row-mutating regression on these
  paths would pass the matrix. Bounded: security property proven; behavior source-verified.
- **Coverage:** current coverage does NOT catch audit/row regressions on these legs (gap).
- **Disposition:** add `PENDING`-untouched + `effect-refused`×1 assertions (6 legs), or
  narrow the header/row claims. **Blocks Gate 2: No.**

### P29-002 — Guard negative-control overclaim

- **ID/Severity:** P29-002, LOW. **Invariant:** register row WS-F ("negative-controlled
  with a plant").
- **References:** `effects/p28_bypass_guard_test.go` (112 lines, single test, no fixture);
  `effects/testdata/` (absent); register row WS-F.
- **Reproducible evidence:** file read + directory listing (no plant exists); this review's
  plant run (impl + call site both flagged, FAIL naming both) proves the guard works and
  demonstrates the missing control's shape.
- **Failure mode/impact:** matcher-logic rot caught only by presence assertions, not by a
  planted violation. Bounded: anti-blindness checks cover both directions; guard verified
  working live.
- **Coverage:** presence assertions catch blindness; no planted control exists (gap).
- **Disposition:** commit a planted-violation test mirroring P24-009's fixture, or correct
  the row. **Blocks Gate 2: No.**

### P29-003 — Trust-name guard is exact-match

- **ID/Severity:** P29-003, LOW. **Invariant:** `TestP28IntentCarriesNoTrust`
  (anti-spoofing structural guard, P26-001).
- **References:** `effects/p28_matrix_policy_test.go` name list; `effects.Intent` struct.
- **Reproducible evidence:** guard source (exact names `Trust`, `TrustMode`,
  `RequestedTrust`); struct enumeration (zero trust fields today).
- **Failure mode/impact:** a future trust-carrying field under another name (e.g.
  `TrustLevel`) evades silently, reopening spoof shape. Bounded: zero today; plausible names
  covered.
- **Coverage:** catches listed names only (gap for novel names).
- **Disposition:** substring match (`Trust`, zero current false positives) or a documented
  name registry. **Blocks Gate 2: No.**

### P29-004 — Decision supersession broader than the closeout

- **ID/Severity:** P29-004, LOW/doc. **Invariant:** prompt28.1 (Option A) + prompt28.1_plan
  GO vs closeout §9/§12 STOPPED + prompt29 §9 STOP premise.
- **References:** `docs/prompt28.1.md`, `docs/prompt28.1_plan.md`, `runtime/lifecycle.go`
  (`SettleEffect`), `transport/cli/cli.go` (settle verb), `runtime/p281_settleop_test.go`,
  `transport/cli/settle_test.go`, memo addendum, register rows 166–167.
- **Reproducible evidence:** mtime/manifest/byte forensics (manifest-era tree: zero
  SettleEffect references; current tree: service + CLI + tests); 8-requirement source audit
  (all hold); 9 evidence items executed green.
- **Failure mode/impact:** reading closeout/prompt29 in isolation suggests the wiring is
  unauthorized. No behavior impact: closeout true-at-write, addendum chain complete,
  implementation conforming.
- **Coverage:** n/a (traceability record).
- **Disposition:** this review is the acknowledgment; future Gates cite prompt28.1.
  **Blocks Gate 2: No.**

### P29-005 — Abandoned-PENDING wedge undocumented (+ hash sourcing line)

- **ID/Severity:** P29-005, LOW/doc. **Invariant:** C6 op-ID rule; authoring guide
  ("adopts the winner's settled outcome").
- **References:** `docs/development/effect-authoring.md`; contract §7/M3 C6 rule;
  `docs/phase1/classd-operations.md` revocation section (hash sourcing).
- **Reproducible evidence:** path enumeration (adopt/takeover/authorize/Settle — no
  cross-invocation PENDING adoption exists); C6 fixture covers settled case only;
  ops doc lacks expected-hash sourcing.
- **Failure mode/impact:** crash-in-claim→authorize-window + owner abandonment + new-key
  retry (Class A) wedges the op (InProgress then terminal binding-mismatch) until operator
  SQL. Narrow trigger, fail-closed, approved trade-off — but unstated.
- **Coverage:** settled path proven; wedge path analyzed, not executed (shape argument).
- **Disposition:** one boundary paragraph in the guide (+contract pointer) + one
  expected-hash sourcing line in ops doc. **Blocks Gate 2: No.**

## 4. Authorization-matrix assessment (§4 work order)

All 12 legs read and executed green. Direct-authorize legs (wrong tenant/invocation/tool,
terminal sweep, live waiting states, pending wait) assert the full five points with honest
framing (the no-send inference from never-leaving-PENDING is valid for direct-authorize
legs and explicitly stated). Perform-level legs (kill-switch ×4, tool-declared, grant
allow/deny, swap race, capability reuse, trust-name guard) genuinely exercise the boundary
through public paths with real PG + mock sends + audits — but the three refusal legs
**do not assert row-state or audit counts** despite the file header and register row
promising "typed classification, untouched state, 0 sends, exact audits"
([P29-001](#p29-001)). The security property (refusal + no send) is proven; the hygiene
assertions are missing. The swap race is correctly labeled (timing race + `-race` detector
+ exact accounting is the right proof shape for an in-memory mutex; no DB order exists to
force). The trust-name guard is structurally sound but exact-match ([P29-003](#p29-003)).
Capability-reuse is thin but correctly defers to the pre-existing full test. All legs
honestly labeled green-first/characterization — no red/green claimed, none required (no
defect found in matrix scope).

## 5. Race-case assessment (B1–B9)

All nine accounted: B1/B3 cite the P25 POC suites (verified present and green in this
tree), B6 cites the WS-A swap hammer, B2/B4/B5/B7/B8/B9 read in full:

- B2/B9 (same-effect race): winner entered-marker + ATTEMPTING row, 5 losers all typed
  contention, release → winner-epoch CONFIRMED + receipt, sends==1, zero refused audits.
  Complete winner/loser/convergence evidence. Sound.
- B4 (lease turnover): aged ATTEMPTING → touch → UNKNOWN + retained epoch + 1 send, release
  lands Superseded. Sound; complements POC-2 (fence without reconcile).
- B5 (restore race): both orders forced — sequential invoke-then-Perform, and FIFO row-gate
  (liveness read queued first by quorum, adopt behind; release → pre-claim refusal with zero
  rows inserted, then adopt wins, retry adopts). FIFO deviation fails loudly on the refusal
  + zero-row assertions. Sound. Seeds genuine pending shape; ungated probe correctly isolates
  restoration from approval.
- B7 (sequence vacuity): fail-closed refusal + PENDING + refused×1; zero-producer proof
  re-verified by grep (zero `Intent.Sequence` producers). Supersede-racing honestly recorded
  untestable; deferral intact.
- B8 (concurrent settle): forced-order FIFO (holder + sequential waiter observation; loser
  path narrated exactly — pre-commit lock-free read then lost predicate → Superseded) plus
  24-way hammer (exactly 1 win, losers typed, terminal final, exactly 1 audit). Sound.
- Forced-ordering audit: every ordering claim with a correctness consequence uses gates,
  row-locks with pg_stat_activity quorums (transitive), or entered markers; sleeps appear
  only as failure bounds or suite-conventional assists with loud timeouts. FIFO reliance
  mirrors the accepted HIGH-1 pattern with loud deviation in each case (verified per test,
  not assumed).

## 6. Crash-boundary assessment (six boundaries)

All six boundaries use genuine `SIGKILL` worker termination (testkit `Kill9`), real DBOS-less
Perform paths, and lock-wait/row-state positioning — verified worker entrypoints,
`WorkerMode` gating, and kill calls:

- B1a (pre-claim queue): 0 rows post-kill; live-simulate → CONFIRMED + 1 send. Sound.
- B1b (mid-auth queue): PENDING + same attempt + 0 audits post-kill; aged takeover →
  CONFIRMED + fresh epoch + 1 send. Sound.
- B2 (post-commit pre-dispatch): ATTEMPTING polled + **0 mock rows proven before kill**
  (post-hoc placement proof, loud on misfire); immediate contention; aged UNKNOWN + epoch +
  0 sends. The gate holds the worker pre-dispatch deterministically; the mock check is
  belt-and-braces. Sound.
- B3 (mid-send): pre-existing test extended additively (worker-anchor `$` fix is genuine —
  required now that sibling entrypoints exist — plus epoch retention). Kill inside the 2s
  DelayedAck window (worker mode verified); contention → UNKNOWN; 0 parent sends, 1 total,
  epoch retained. Sound.
- B4/B5 (shared): killed while the settlement UPDATE queues behind a held effect row with
  the mock response already recorded. The equivalence claim was challenged and ACCEPTED:
  durable state (ATTEMPTING + epoch + 1 mock row), recovery (age → UNKNOWN with adopt
  marker, operator Settle from evidence), and allowed evidence are identical; the B4/B5
  distinction (pre-statement vs queued-statement) has no post-kill observable a test could
  separate deterministically. Explicitly recorded, not quiet. Sound.
- B6a/B6b (takeover/quarantine authorize): queued-step kills leave rows untouched;
  retries converge (CONFIRMED+fresh epoch+1 send / UNKNOWN+0 sends). Sound.
- B2's stated exception (no forced order — durability order) is correct: once ATTEMPTING
  commits, recovery needs no interleave.

## 7. C6 Class A recovery assessment (11 claims)

The fixture drives a REAL service → dispatch → Class A executor → real handler → Perform
path (only the contending holder's send is gated — a legitimate second worker). All 11
claims verified: reachable integration (1); full rollback incl. handler INSERT (2,3); record
stays live `running` with empty result (4); zero held locks post-return + SetGrants
liveness proving RLock free + prompt return (5); bounded lease-untouched recovery, no aging
step (6 — the 30s bound is loose but the no-wait mechanism is what's proven); identical
BusinessOpID across both attempts (7); winner UUID epoch on the final row (8);
interrupted/new-key guidance on same-key retry (9); new-key settled-adopt without resend +
exactly 1 send + exactly 1 business row (10,11). Docs verified: the authoring guide's
derivation rules, no-wait/no-retry-in-handler, state-specific retry paths, and no-promises
all match the proved behavior; contract §7/M3 C6 rule matches. One gap:
[P29-005](#p29-005) — the guide promises new-key adoption of the winner's outcome without
noting the abandoned-owner boundary (PENDING row under a dead live invocation refuses
cross-invocation takeover by binding; same-invocation retry or operator action is the path).
Approved-design trade-off, narrow trigger (crash in the claim→authorize window +
abandonment + new key), needs one boundary paragraph — not a behavior defect.

## 8. Grant hash, health, and revocation assessment

- `Service.GrantConfigHash()` delegates to the self-locking `ConfigHash`; both prod callers
  (startup log, /healthz) verified lock-free. Canonicalization test pins rotated-insertion +
  100-read bit agreement. Sensitivity/consumers verified: /healthz returns exactly
  `{status, grant_config_hash}` (test pins 2-key body + hash identity + flip-on-swap);
  consumers string-compare with no re-canonicalization (comment states it; correct — both
  ends use one function).
- verify-revocation: explicit-inventory-only (empty → exit 2, never vacuous success),
  complete verdict partition (ok/stale/unreachable → verified/STALE/MISMATCHED/INCOMPLETE —
  all four shapes tested incl. file inventory, dedup, timeouts, malformed input), deterministic
  ordered output, REST-default port matches server default (8080, verified), TLS-safe (no
  skip-verify), credential-free polling. Sound.
- Ops doc: accurate rollout procedure, honest 10-minute staleness, rolling-restart coherence,
  no immediate-revocation promise (grep audit clean — only historical/residual mentions).
  One gap: expected-hash sourcing (startup log / live /healthz read) is workable but
  undocumented — fold into [P29-005](#p29-005)-class doc follow-ups (one line; tracked as an
  observation under §8, no separate finding).
- Wait-then-revoke E2E (§H-7): wait leg, mid-wait revoke leaving the approval row untouched,
  post-approval grant-denied refusal + exactly 1 audit, fresh-op isolation from lease
  contention — sound and correctly designed (different op forces fresh authorize).

## 9. Settle crash matrix and operator authority

Crash scenarios: (1) genuine kill with UNKNOWN + 0 audits + converging retry ✓; (2)
fault-modeled lost-response, explicitly labeled with sound justification (post-commit kill
adds no caller-observable distinction — durability is PG's) ✓ honest mechanism choice;
(3) backfill truthfulness via simulated legacy shape ✓; (4) correctly cites B8 + the
conflict test ✓. Backfill fix verified: single effect-lock re-read, C2-safe (one lock,
lock-free appends after), migration-free; red reproduced (`round 1: settled audits = 16`)
matching the reported signature; green 3× (this review: 1× plus full suite).

Operator authority — decision supersession ([P29-004](#p29-004)): the closeout's STOPPED
claim and prompt29's STOP premise predate owner prompt28.1 (Option A approved) and its
prompt28.1_plan GO. Manifest forensics prove the closeout was TRUE when written (manifest-era
tree contains zero SettleEffect references) — no false report. The Prompt28.1 track (service
method + CLI verb + 2 test files + memo addendum + register rows + F-03 correction) then
landed, partly during this review; every affected area was re-verified on final bytes.
Conformance to prompt28.1's 8 requirements verified point-by-point: binding authentication,
exact case-sensitive role match, tenant-from-binding with predicated enforcement,
binding-derived operator identity, deny-by-default at every step (incl. empty agent/tenant),
outcome/note validation, atomicity delegation (effects.Settle untouched — diff-verified),
truthful attribution; CLI carries zero authorization logic and documents the credential
process-listing limitation. All 9 evidence items present and green (authorized + idempotency +
attribution; 9-leg refusal matrix incl. spoof/case/tenant-ops/substring; tenant indistinguish-
ability incl. cross-tenant-operator; validation matrix; CLI flags + e2e). No REST/MCP endpoint,
no RBAC expansion, no migration. The STOP is therefore MOOT (superseded by a newer owner
decision + conforming implementation), not violated — recorded here with the addendum chain
intact.

## 10. Provider-path inventory and bypass analysis

Independent inventory (call-graph walk from every entry to the single `ch.Send` statement):
fresh/takeover/reconcile all authorize-then-send; adopt short-circuits never send;
unknown-capability paths fail closed pre-authorize (channel lookup precedes authorize —
verified order) with rows unchanged; settled/UNKNOWN reads never send; faultwrap/mockcap
have zero non-test importers (re-verified); `dbos.Send` has exactly one call site, verified
workflow signaling. Per-path tests pin revoked-reconcile refusal, terminal-reconcile
lost-withdraw pairing, UNKNOWN non-resend (3×), and unknown-capability fail-closed on all
three initiation paths (honestly labeled characterization incl. the taxonomy noise and the
P24-008 pointer; op-ID burn on bogus capability is fail-closed and acceptable). DBOS
workflow-called lineage proven end to end (CONFIRMED + UUID epoch + invocation binding +
receipt + exactly 1 send). AST guard reviewed: impl/call-site inventories with qualifier
handling, dbos allowlist matching the verified single site, anti-blindness presence checks
both directions. Detection PROVEN by this review (planted impl + call site → FAIL naming
both). Missing: a committed planted negative control (the register claims one —
[P29-002](#p29-002)). New-call detection verdict: YES, mechanically enforced.

## 11. C1–C7 revalidation

| Constraint | Gate 2 status | Evidence |
|---|---|---|
| C1 no advisory in authorize | HOLDS | No advisory SQL in any touched diff; backfill/CLI/health/hash add none (re-grepped) |
| C2 lock order | HOLDS | Only new lock is the backfill row-5 re-read (single level, verified order); settleop adds no PG locks |
| C3 own tx | HOLDS | No signature changes (all prod diffs additive/internal); SettleEffect delegates into existing txns |
| C4 no new advisory paths | HOLDS | Exhaustive grep + guard green + plant proof |
| C5 pending semantics | HOLDS | adoptRunID untouched (no diff); B5 + P22 suites green |
| C6 Class A recovery | PROVEN (fixture closes the deferral) | 11/11 claims executed; lease code untouched; op-ID rule normative + guide accurate |
| C7 O-01 at effect boundary | HOLDS | Terminal-reconcile + protection legs green; orphan cannot send |

Closeout §11 uses correct P21 labels with C6's closure explicitly tied to the fixture
(P26-003 resolved; verified text).

## 12. Battery, pins, and environment

- Full battery on exact-parity PG19 (digest-verified image, fresh DBs): gofmt/vet clean,
  checks ALL-CHECKS-PASS, full serial suite 22 ok / 0 fail (dbos 38s, classd 40s),
  `-race` effects/runtime/approvals green, all P28/P281 focused suites green (incl. crash
  workers, matrix, races, settle, revoke, healthz, ledger proof, C6).
- PG19 parity chain: PGDATA probe + Compose default (digest, env-overridable) + CI service
  (digest) + Taskfile default (digest) + `postgres-image.sh` (digest default, pre-release
  refusal, report-only) + compose_test digest expectation — all verified byte-level; zero
  floating refs in functional config. CI execution itself not run (honestly stated).
- No hidden skips: crash entrypoints skip-by-design outside parent runs and RAN under the
  matrix (verified in output); DBOS/Class D ran on required URLs.

## 13. Residuals and spec confirmations

Accepted (re-examined, boundaries intact): O-01 submit race (effect-time protection +
withdrawal hold); 32-bit hash; ledger boundary; X-12; T5-style characterizations; FIFO
staging (loud-on-deviation, soundly reused in POC-1/B5/B8); aggregate LOW-1 counts; 2-min
lease (now load-bearing for C6 wording — accurate); INFO-2 raw-SQL bypass; miniapp SQL
discipline; M5 probe deferral + trigger (wording intact, re-grepped); X-10 resumability;
mint-failure branch; aliasing beyond the guard; unknown-capability taxonomy noise
(characterized, trigger-bound); op-ID burn on bogus capability (fail-closed); prompt21
scratch (untouched); B4/B5 shared-test equivalence (accepted with stated reasoning);
expected-hash sourcing (one-line doc follow-up, §8 observation).
Spec confirmations: no non-additive prod change beyond the backfill hunk; no migration;
no new advisory/lock levels; no REST/MCP settle surface; role vocabulary single-purpose;
closeout STOPPED claim true-at-write with addendum supersession recorded.

## 14. Required corrections and Gate 2 release recommendation

Required (all LOW, none blocking release): [P29-001](#p29-001) complete the policy-leg
assertions (or narrow the row); [P29-002](#p29-002) commit a planted guard control (or
correct the row); [P29-003](#p29-003) strengthen the trust-name guard; [P29-004](#p29-004)
acknowledge the Prompt28.1 supersession chain (already addendum-recorded — this review is
the acknowledgment); [P29-005](#p29-005) document the abandoned-PENDING boundary (+contract
pointer) and the expected-hash sourcing line. Suggested sequencing: fold all five into the
next workstream's first commit (each under an hour); none requires re-review beyond
confirming the artifacts exist.

**Recommendation: release Stage 3A (Gate 2 CLEAR).** The authorization core is now proven
across matrix, races, crashes, recovery, revocation, settlement, and bypass dimensions
with honest evidence labeling throughout; the one genuine defect found was fixed with a
real red/green pair; the operator boundary arrived via explicit owner decision with
conforming implementation and full evidence. Do not weaken the M5/P24-008 triggers or the
single-boundary settle rule as Stage 3B begins — they are the load-bearing interfaces the
next phase builds on.

## 15. Explicit statement of what has not been proven

- CI execution itself (pinning verified byte-level; workflows not run — honestly stated).
- M5 supersede-racing (accepted residual; fail-closed + zero-producer proven, racing
  untestable without the deferred verifier).
- Multi-replica revocation propagation timing (procedure + verification proven; the
  10-minute window itself unmeasured).
- Operator CLI against production identity systems (demo keys file; process-listing
  limitation documented; no production credential store exists yet).
- Load/performance characteristics (no perf bounds claimed or tested).
- `-race` on dbos/classd suites (subset matches the project's own race scope).
- The abandoned-PENDING wedge beyond shape analysis (narrow, fail-closed; documented
  residual per [P29-005](#p29-005)).
- Byte-identical historical red trees (red *signatures* reproduced functionally, which is
  what discrimination requires).

## 16. Recommended next architectural work order

1. Land the five P29 LOWs (first commit of the next workstream; each under an hour).
2. Execute the revocation drill end to end at least once against real replicas (procedure
   proven, practice unpracticed) before relying on it operationally.
3. Keep the M5/P24-008 activation triggers load-bearing in Stage 3B planning: no sequence
   workload without the four-gate trigger (framework-declared requirement, trusted
   propagation, supersede validation, X-08 barrier); no real provider without the
   capability-membership check.
4. Hold the single-boundary settle rule: all future settle adapters (if any) enter through
   `SettleEffect`; no second authority vocabulary without a new owner decision.
5. When the first production credential store lands, retire the demo keys file and re-prove
  the operator boundary against it (the p281 matrix is the regression net).
