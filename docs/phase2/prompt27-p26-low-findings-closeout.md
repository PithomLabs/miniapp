# Prompt27 — P26 LOW-findings closeout

Gate 1 is CLEAR (independent Prompt26 review accepted). This
bounded work order closes P26-001/002/003 and reconciles the
working-tree baseline. No Stage 3A implementation was started.

## 1. P26-001 — Owner confirmation and exact scope

Owner confirms the scoped P24-005 reading (prompt27 preamble):

- The prohibition against bypassing denied, expired,
  mismatched, or cancelled approvals through a
  caller-supplied trust claim applies when persisted trust
  is Confirm (anti-spoofing).
- Genuinely persisted Autonomous + explicit manifest
  permission proceeds without approval linkage.
- Inapplicable approval rows create no requirement for a
  legitimate persisted-Autonomous invocation; the exception
  never lets a caller flip persisted Confirm to Autonomous.

Recorded durably in the lifecycle contract §16
(`P26-001 (Prompt27)` row, citing Prompt26 §6 and the
design §2 / contract §7/M3 Authorize rows) and cited on
the M6 autonomous-exception leg comment in
`effects/p25_p24005_internal_test.go` (comment-only; no
implementation change, per the work order).

## 2. P26-002 — Test behavior, results, activation trigger

New `TestP27EmptyModeGatedAutonomousRefused`
(`runtime/p27_p26002_test.go`, `c82c5b19…`) with a
self-contained fixture (gated miniapp, empty `TrustModes`,
autonomous grant; shared fixtures untouched). Observed on
disposable PG19, all as the residual describes:

1. Submit admits: autonomous gated invoke → running + handle.
2. Persisted trust recorded: invocation row `trust_mode =
   autonomous`, status `running`.
3. Authorization refuses (terminal `*RefusalError`).
4. Exact reason: `approval-missing`.
5. No send (effect ledger row `PENDING` — claimed, never
   `ATTEMPTING`; `SentCount == 0`), no loop, no misleading
   success: immediate repeat meets adopt contention (fresh
   PENDING answers before authorize re-runs), post-lease
   repeat returns the terminal refusal — the cycle
   converges to refusal; invocation row undisturbed
   (still `running`).

Invocation-row state (`running`) and effect-ledger-row
state (`PENDING`) are asserted as distinct rows throughout.

Classification: characterization-only (no code change
exists to pair a red; never claimed as red/green). An
early draft of leg 5b expected the repeat to re-refuse
and caught the adopt-contention shape instead; the test
now asserts the full observed cycle.

Activation trigger (recorded in design §11 and contract
§7/M3, next to the P24-008 prerequisite): before any
empty-mode production manifest enables gated tools,
submit-side validation must prevent admission of trust
combinations that cannot pass authorization. Preferred
durable direction: explicit trust-mode declaration for
gated tools, or an equivalent manifest-admission rule
preserving legacy behavior outside that boundary. No
manifest migration in this task.

## 3. P26-003 — Corrected C1–C7 table and C6 status

`prompt25-gate1-corrections-report.md` §11 rewritten to the
Prompt26 §7 authoritative mapping (original Prompt21
definitions). Verified: exactly one diff hunk, §11 lines
only — no other report section changed.

| Constraint (P21 definition) | Gate 1 status |
|---|---|
| C1 — `authorizeAttempt` takes no advisory lock | Holds |
| C2 — Authorization follows the global lock order | Holds; P24-003 restored Settle's effect-lock-only footprint |
| C3 — Authorization uses its own transaction, not the attempt's `pgx.Tx` | Holds |
| C4 — No new advisory-lock acquirers | Holds |
| C5 — Explicit pending semantics | Holds |
| C6 — Class A handlers that call `Perform` handle `ErrAttemptInProgress` with bounded recovery and correct operation-ID behavior | Mechanism sound; integration unproven; fixture and guidance deferred |
| C7 — O-01 orphan window closes at the effect boundary | Holds; strengthened by P24-001 |

C6 is explicitly NOT closed by P24-010; its fixture and
guidance remain deferred to the Stage 3A work.

## 4. Worktree reconciliation (incl. faq delta)

- HEAD at freeze: `a09b7ff` ("prompt26", 2026-11-23
  23:30 +0800, author ibmendoza), clean tree. (HEAD
  moved since Prompt25: the Prompt25 work, review
  artifacts, and faq append were committed by the
  owner outside this work order.)
- `docs/faq.md`: `1a67ba2 → a09b7ff` is +158/−1 — the
  single deletion re-added with a trailing newline;
  the rest is a pure append (Datadog/Kibana
  observability-comparison Q&A). The review saw
  +64/−1 at 23:18; outside activity continued before
  the 23:30 commit. Blob `df1e33244f…`; worktree ==
  HEAD — preserved exactly, no reset/discard/overwrite.
  Status: out of scope for Prompt25/26/27 (no
  attribution claimed either way).
- `docs/prompt26.md` (295 lines, `599a166b…`) and
  `docs/adv_review_prompt26.md` (291 lines,
  `c55b6022…`): separate review artifacts, not
  implementation evidence.
- Historical Prompt25 digests/manifest left untouched;
  new baseline below supersedes nothing — it extends.

## 5. New baseline digests and manifest

- Source digest (stable; excludes this report):
  `664558bed1390c6d3d982809eabb588a52fee3a03164e2b09ea478d2abf80b2f`
  (`bash scripts/digest.sh source .`;
  reviewer-reproducible exactly).
- Tree digest: `2ee4c9eaf038c289bfd74dacb06aa6808a0c65fc805464735456de6a3abd8827`
  (`bash scripts/digest.sh tree .`), computed over
  every file except this report (reproducible by
  moving this file aside).
- Manifest (complete inventory, self excluded):
  `docs/evidence/digests/p27-closeout-2026-11-23.manifest.txt`.
- Baseline note: source digest at freeze,
  `e4b517e8…`, exactly equaled the Prompt25 closeout
  source digest — zero executable drift between
  Prompt25 green and the Prompt27 baseline.

## 6. Environment and test evidence

- Image re-resolved via `docker inspect`:
  `postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
  (exact match). Server: `PostgreSQL 19.0 (Debian
  19.0-1.pgdg13+1)`.
- Disposable container (`--rm`, ephemeral password,
  port 5444), fresh `p27test` database; long-lived dev
  DB never connected to; container removed after the
  runs.
- `go test ./runtime/ -run TestP27EmptyMode -count=1`:
  PASS (0.07s).
- `go test -p 1 ./runtime/ ./effects/ -count=1`:
  PASS (5.5s / 1.8s).
- `gofmt -l .` clean; `go vet ./...` clean;
  `bash scripts/checks.sh` ALL-CHECKS-PASS.
- Full serial suite skipped per prompt27 §4: the test
  is new-file-local (own fixture), contract/design
  edits are doc-only, the M6 touch is comment-only —
  no shared package behavior affected.

## 7. Residuals and activation triggers

- P26-002 empty-mode coherence: accepted bounded
  residual, pinned by test; trigger in §2 above.
- P26-001 scope: closed (disposition recorded, no code).
- P26-003 traceability: closed (§11 rewritten, C6
  still deferred).
- Carried Prompt25 residuals unchanged (X-12, ledger
  boundary, mint-failure branch, aliasing beyond the
  AST guard, M5/X-08, C6 fixture).

## 8. Confirmations

No Stage 3A implementation, no C6 fixture, no Settle
wiring, no operator auth, no revocation tooling, no
bypass/crash matrices, no migrations, no dependency
changes, no commits, staging, pushes, deployments, or
real provider writes. Production changes: none (one new
test file, one test comment, docs/evidence only).
Stopping here per prompt27 §6; next gate is the owner's
Stage 3A direction.
