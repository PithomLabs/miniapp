# Prompt 15.2 Implementation Plan

| Field | Value |
|---|---|
| Governing prompt | `docs/prompt15.2.md` (supersedes all earlier Prompt 15 drafts) |
| Review inputs | `docs/adv_review_prompt14.md`; `docs/prompt15_z.md` (RC-1/RC-2 source) |
| SDK reference | `/home/chaschel/Documents/go/dbos-transact-golang-1.9.0` — verified byte-identical to the pinned v1.9.0 module source (`diff -rq` clean); authoritative for all §12 SDK claims |
| Policy context | `docs/DBOS.md`; `docs/plan4.2.md` remains the immutable authority |
| Status | PLAN ONLY — no implementation authorized yet |

## 1. SDK reading results

Read `/home/chaschel/Documents/go/dbos-transact-golang-1.9.0` in full for the
behaviors 15.2 §12 requires. It is byte-identical to the pinned v1.9.0 module
source, so all citations below are authoritative (the older
`dbos-transact-golang` sibling clone is superseded for this work and must not
be cited — it predates v1.9.0 suspend support).

| Finding | Verified v1.9.0 behavior (exact citations) |
|---|---|
| X-03 duplicate submit | `InsertWorkflowStatus` (`dbos/internal/sysdb/system_database.go:1471`, upsert `:1593–1627`) uses `ON CONFLICT (workflow_uuid) DO UPDATE` + `CreatorXID` self-write detection with `RejectExisting` enforcement (`:1700`); same-ID resubmit attaches to the existing row, but concurrent same-key framework submits mint **different** run IDs, so SDK dedup cannot save them — the predicated-adoption fix must live in `runtime/store.go`. |
| X-06 blind signal | `SysDB.Send` (`system_database.go:5037`) is a blind `INSERT INTO notifications`, and notifications have "no foreign key on workflow_status" (`:2570`) — send-to-nonexistent returns nil. Framework-side existence verification is mandatory; the fix API is `RetrieveWorkflow` (`dbos/workflow.go:5242/5296`), which returns `NonExistentWorkflowError` for unknown IDs. Residual check-then-send race must be documented per 15.2 §8/D. |
| X-14 shutdown | `Shutdown` (`dbos/dbos.go:1057`) returns a timeout error enumerating pending components including `"workflows"` (`:1148`) — the framework can detect workers-still-active and must never silently `pool.Close()` on that path. |
| X-09 affinity | `ReenqueueForRecovery` (`system_database.go:6349`) re-enqueues only matching `application_version` — native mechanism the behavioral test (v1 pending, v2-only worker, restart v1) must discriminate. |
| X-10 statuses | v1.9.0 enum (`dbos/internal/models/workflow_status.go:13–20`): PENDING, ENQUEUED, DELAYED, **SUSPENDED**, SUCCESS, ERROR, CANCELLED, **MAX_RECOVERY_ATTEMPTS_EXCEEDED** — the X-10 enumeration scope. |
| X-11 bounded wait | `Recv` takes a timeout plus `RecvOption`s (`dbos/workflow.go:3976`) — candidate mechanism for the framework-enforced recovery bound (P1 selects exactly one enforceable mechanism per 15.2 §8/D). |

Superseded / non-authoritative: `docs/prompt15.md`, `docs/prompt15.1.md`,
`docs/prompt15_plan_review.md` (structure; its environment/checkpoint/OpenCode
decisions are restated in 15.2 §5), and `docs/adv_review2_prompt14.md` (despite
its name, a competing Prompt 15 draft mandating one universal `transition()`
primitive and the two `classd-state-transition-*.md` docs — both rejected /
forbidden by 15.2).

## 2. P0 — Baseline (15.2 §6)

Carry over completed work: HEAD `5cf9627`, worktree digest `5564a37b`
(247 files), serial suite green / parallel red on the shared local database
(sanctioned invocation stays `go test -p 1 ./... -count=1` per Prompt 13 —
15.2 §6's bare `go test ./...` needs that qualification recorded),
`bash scripts/checks.sh` PASS, preflight 1 pass / 2 warn / 1 skip, Neon
`phase0/g0-spikes` branch-identity probe PASS (production untouched).
Remaining: OpenCode model preflight (pending owner authorization).
No code changes.

## 3. P1 — Contract before code (15.2 §§3, 7)

Ratify, revise, or reject RC-1a–d / RC-2 in writing against source + v1.9.0
(§1 above is the head start, not the verdict), then author
`docs/phase1/invocation-lifecycle-contract.md`:

- Six durable state machines (invocations, approvals, effects/attempts,
  workflow/run identity, sequences/attachments, cancellation).
- The six normative transition-integrity rules; subsystem-specific transition
  operations (no universal `transition()`).
- Retry-as-re-entry handlers for every retryable path.
- Lock discipline: never hold a PostgreSQL transaction or lock across an
  external send (constrains the X-05 claim-gate design).
- Submission choke-point specification with call-graph / grep / automated-test
  proof that every executor submission passes through it.
- Full transition table (state × event) with predicate, atomicity scope,
  authorization, audit, failure exit, recovery, owner, and one discriminating
  test per row.
- Testkit `FaultInjector` design.
- X-01–X-24 → defect class → contract rule → primary test map.

No source edits until this document exists.

## 4. P2 — Remediation (15.2 §§8–10)

Sequential at shared state-machine boundaries; dependency order:

1. `FaultInjector` + WS-C (X-07 mock measurement integrity first — later
   effect evidence depends on it).
2. WS-A (X-01–X-04) together with X-19/X-22 (same `runtime/` store/service
   files).
3. WS-B (X-05 cancellation-vs-claim gate at the claim boundary).
4. WS-D (X-06, X-11, X-12, X-13, X-14 lifecycle/shutdown truth).
5. WS-E (X-08 genuine attachment default — stop for owner decision only on a
   proven governing-spec conflict; X-09 behavioral affinity; X-10
   enumeration).
6. WS-F (X-15 control-plane separation; forward-only migration after
   inspecting the actual sequence).
7. WS-G (X-16 analyzer), WS-H (X-17 CI), WS-I (X-18 evidence), WS-J
   (X-19–X-24 remainder).

Red/green evidence for every HIGH finding and X-07 via an isolated
pre-remediation snapshot (separate database names, both runs bound to exact
source digests, active tree untouched).

## 5. P3/P4 — Proof and close-out (15.2 §§12–17)

Barrier-synchronized concurrency suites under `-race`, genuine kill/restart
recovery tests, security re-verification (tenant derivation, principal
separation, auth-before-dispatch, kill-switch separation, secret-free audit),
full verification battery with per-environment evidence, acceptance matrix
(§14), F-01–F-14 + G1 rebuild from `prompt10.md` §9 (13/14 claim withdrawn),
digested evidence-register entries, and `docs/prompt15_imp.md` (§17.2 A–M).

Standing boundaries: ADR-001 `lead-followup` hard stop, no commits or pushes,
no protected-document edits, forward-only migrations, Neon/remote-CI honestly
recorded as BLOCKED / unobserved when not run, stop only for genuine
governing-spec conflicts, unresolved owner decisions, or unsafe/unauthorized
mutations.

## 6. Open questions for the owner

1. **Supersession:** confirm 15.2 supersedes `prompt15.md` /
   `prompt15_plan_review.md` in full — keeping their environment/checkpoint /
   OpenCode decisions, which 15.2 §5 restates — so the old GO does not
   authorize this structure.
2. **OpenCode:** authorize the isolated-HOME model probe + Prompt 13-scope E2E
   rerun, or mark OpenCode BLOCKED by owner decision (G1 row 10 accordingly)?
