Prompt 13 per docs/prompt13.md is complete. Final report (§24):

## A. What was proven

The reusable framework's Class D capabilities are proven end to end on real PostgreSQL (local Docker PG19, plus a separate Neon pass): canonical invocation with idempotent submit and key-conflict rejection; tenant-scoped entity keys with Class A serialization and Class D attach semantics; a race-hardened effect ledger (contention/adopt/takeover/settle, exactly-once confirmed effects); durable-first cancellation with redelivery and in-flight reporting; a crash-safe approval bridge (idempotent Decide, decision signals, workflow table re-read, scope enforcement, lazy expiry); per-family DBOS applications with pinned compatibility versions, native affinity, and drain visibility; effect-time revalidation (live grants, token binding, five-level kill switches); durable sleep/signal behavior with measured envelope; explicit stale status; a seven-family determinism analyzer with strict CI; a 12-test neutral conformance probe; thin REST/MCP/CLI adapters with an agent-driven OpenCode demo; and a structural ADR preflight. Matrix: [classd-framework-readiness.md](/home/chaschel/Documents/go/miniapp/docs/phase1/classd-framework-readiness.md). Findings: [classd-capability-matrix.md](/home/chaschel/Documents/go/miniapp/docs/phase1/classd-capability-matrix.md). Evidence: [evidence register](/home/chaschel/Documents/go/miniapp/docs/evidence/README.md).

## B. Exact commands

Baseline and validation: `gofmt -l .`, `go vet ./...`, `go test -p 1 ./... -count=1` (plus `-race` for the final run), `bash scripts/checks.sh`, targeted `go test -run` invocations, `go run ./cmd/preflight --check-all`, `opencode run --format json -m opencode/deepseek-v4.1-flash-free -- "<script>"`, `neon projects list`, `neon branches list`, and a throwaway `migrate` down/up driver. DB URLs came from the secrets dir (local) and a branch-scoped connection string (Neon, destroyed after). Full per-run commands live in the evidence register.

## C. Finding verdicts

F-01 submit atomicity — PROVEN (record-before-submit, idempotent re-submit). F-02 approval/signal/cancel convergence — PROVEN (denial settlement, bridge, redelivery). F-03 ledger races — PROVEN (predicated transitions, lease rule, settle). F-04 identity separation — PROVEN (derivation contract + durable-sequence demo). F-05 family/version — PROVEN (per-family apps, pinned versions, overlap/drain tests). F-06 effect-time policy — PARTIALLY PROVEN (grant/token/kill-switch enforced; volumetric controls need owner input, OSI-4). F-07 key reuse — PROVEN (canonical comparison). F-08 tenant predicates — PROVEN (incl. BYPASSRLS proof). F-09 approval authority — PROVEN (scope, idempotency, gates). F-10 sleep/signals — PROVEN (kill/resume, matrix, envelope, edition gate). F-11 status freshness — PROVEN (`Stale` marker). F-12 serialization — PROVEN (queue evaluated ordering-only; DB-arbitrated uniqueness demonstrated). F-13 analyzer — PROVEN (checker, fixtures, strict gate, ops doc). F-14 contract enforcement — PROVEN (preflight + CI gate). Root causes, corrections, and test links are in the capability matrix.

## D. Issues fixed during execution

Orphan-on-submit-failure; silent input-mismatch replay; ledger clobber race; denial without run cancel; missing decision bridge and expiry; single-AppName/no-version executor; unenforced limits; silent stale status; no analyzer/preflight/transports/serve binary. Plus incident dispositions (all in evidence): pool-close deadlock order, envelope identity, jsonb spacing, package-parallel truncation, applied-migration edit (forward-fixed via 000005), parallel-run interference, three Neon-only test assumptions, boolean MCP schemas, DB-global queue/version namespaces, MCP env verification.

## E. Pinned-SDK behavior (v1.9.0, source-verified)

Recovery and dequeue filter on application version (native affinity); partition keys are free strings; resident sleepers hold partition slots (suspend is Enterprise-gated — proven by test); same-partition FIFO; partition key and deduplication ID mutually exclusive; queue names and version names are DB-global; steps don't retry by default; recovery dead-letters after 100 attempts; no automatic history GC; `Send` buffers without state checks.

## F. What cannot be claimed yet

Volumetric/allowlist/circuit-breaker enforcement (OSI-4, needs formats); remote CI observation (no push authorized); cross-release artifact retention (G6 drill); any `lead-followup` workload readiness (hard stop holds; ADR still DRAFT).

## G. Refusal confirmation

No real providers, OAuth, writes, or credentials anywhere (mock-only channels, gate 3 clean); no reconciler worker (test-only settle demo); no River/Class C, Sprites, promotion, Fly/G6 work, UI, sales playbook, or `lead-followup` code/manifest/registration/execution (verified: the sole Go mention is the probe's disclaimer comment); no commits or pushes; no baseline edits; no production or Neon state mutations beyond the test branch's disposable fixtures.

## H. G1 readiness

13 of 14 operational checks PASS; criterion 12 (CI enforcement) is PARTIAL — gates wired and green locally, remote CI unobserved without a push. No overall G1 PASS is declared here; that verdict belongs with the owner and the named reviewer.

## I. Corrected premature items

"Exactly-once" ledger language → R2-honest at-most-once-confirmed (code comments fixed, lint added); workflow-ID-as-lock and queue-alone serialization → rejected per prompt13, replaced by durable-record uniqueness; `Sleep` residency assumptions → measured; preflight overclaim warnings now flag the ADR draft's guarantee phrases for reviewer judgment.

## J. Files changed/added

New packages: `runtime` lifecycle/entity/canonical, `executor/dbos` rewrite + `executor/classa` locks, `effects` rewrite + `policy.go`, `approvals` bridge APIs, `auth` roles, `contract` entity/expiry/signal additions, `registry` admission rules, `testkit/{mockcap,stack,proc,miniapp}`, `probe/classd` (12-test fixture), `transport/{rest,mcp,cli}` + contract tests, `analyzer/determinism` + fixtures, `preflight`, `cmd/{miniapp,determinism-lint,preflight}`, migrations 000003–000005, five Phase 1 docs, demo transcript. Modified: contract validation, store predicates, service/lifecycle, checks.sh (+3 gates), go.mod/go.sum (x/tools v0.52.0), evidence register. Untouched: plan4.2, reviews, committed baseline. Note: `docs/faq.md` is untracked and not mine — left alone.

## K. Open items and owner decisions

OSI-1 (playbook semantics), OSI-2 (role vocabulary), OSI-3 (no-entity variance), OSI-4 (volumetric semantics) all OPEN. §22 decisions (pilot path, named reviewer, synthetic values) all OPEN — they block increment 6 only, not this framework result.

## L. Close-out

Prompt 13's framework-proof objective is met: every assigned finding is either proven with executed evidence or explicitly bounded with a recorded reason. No test was weakened to pass (three were strengthened after failures taught better assertions); no failure was left uninvestigated; local and Neon evidence are separate and complete. The hard stop stands, the tree is uncommitted for review, and the next step is yours: the three owner decisions.
