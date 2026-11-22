# Adversarial Architecture and Implementation Review — Prompt 5

- **Reviewer role:** independent principal architect / senior Go engineer (read-only review; no source, config, docs, or history modified except this report).
- **Repository:** `/home/chaschel/Documents/go/miniapp` (single Go module `github.com/PithomLabs/miniapp`).
- **Baseline:** `docs/plan4.2.md` (authoritative; plan4.1 preserved as history).
- **Review date (UTC):** 2026-11-22. All commands below were executed fresh on that date unless marked otherwise.
- **Report path:** `docs/adv_review_prompt5.md` (per review request; prompt5 §16 names `docs/reviews/adversarial-implementation-review.md`, which was not created — no other file was touched).
- **This report contains no secrets, tokens, database URLs, or credentials.**

## 1. Executive verdict: CONDITIONAL GO

The implementation is coherent, honestly scoped, and largely faithful to plan4.2 for a Phase 0 freeze: contracts validate real invariants, the `_FILE` secret contract fails closed on the tested paths, Taskfile environment isolation holds up under inspection, the Compose design is sound, and the evidence tracker is unusually honest about what is blocked. No CRITICAL finding was demonstrated.

It is not an unconditional GO. One HIGH finding (production authority is policy, not mechanism) needs an explicit owner decision before any production wiring; ten MEDIUM findings need owned fixes or test executions; and the highest-value behavioral evidence (DBOS/River/RLS spikes) still has no passing execution anywhere. None of the prompt5 §17 NO-GO criteria is triggered **today**, but criterion 1 (untrusted-agent route to production authority) is exactly one environment configuration away from being triggered, which is why this verdict is conditional rather than GO.

**Gates for proceeding to Phase 1** (see §15 for owners): H1 owner decision recorded; M1, M2, M3 fixed; M7 spikes executed once (local Docker suffices); M8 guide consolidated to one voice; M9 CI runs `test:integration`. M5/M6 must be fixed before any production use, not before Phase 1 coding.

## 2. Actual repository inventory and baseline verification

Git state (verified `git status --short`, 2026-11-22): branch `main` with **zero commits**; every file untracked; no staged or unstaged diff to review. The "no commit authorized" posture from prompt5 §3 holds. My own probe directory (`probecheck/`, used for two execution proofs) was created and removed inside the review; `git status` after removal shows only the pre-existing untracked set.

| Claimed | Actual (verified) | Verdict |
|---|---|---|
| 17 ADRs | 17 files `docs/adr/001..017-*.md`, titles match plan4.2 §5, plus `miniapps/adr001-template.md` + `inventory.md` | PASS |
| 11 generated JSON Schemas | 11 files in `contract/schemas/`; `TestSchemasCurrent` regenerates byte-identical | PASS |
| Separate dev/prod Taskfiles | `Taskfile.dev.yml` (16 tasks), `Taskfile.prod.yml` (8 tasks); no `includes:` in either | PASS with note (§5) |
| Local PostgreSQL 19.0 | Default `postgres:19.0` pinned in Taskfile + Compose + `.env.example`; digest verified against Docker Hub (see §6) | PASS with note (§6) |
| Seven dev/Fly scripts | 3 in `scripts/dev/` + 4 in `scripts/fly/` = 7, plus `scripts/checks.sh` as an 8th repo-check script | PASS (counting clarified) |
| Go `_FILE` config contract | `internal/config/secret.go` + unit tests; conflict/missing/empty/trim/redaction covered | PASS with findings M1, M10 |
| Regression + infra tests | `infra/` (taskfile/compose/scripts), `contract/`, `internal/*/` all green | PASS |
| Phase 0 spike harnesses | 5 test files in `spikes/`; only MCP executes here, rest SKIP without env | PARTIAL (see §3) |
| Execution-category guide | `docs/app_category_guidelines.md`, 2296 lines, **four concatenated drafts** | FAIL as a single guide (M8) |
| Secret/env documentation | `secrets-management.md`, `ENV_VARS.md`, rotation + Fly guides | PASS with findings (H1, §11) |
| Dockerfile / `.dockerignore` / `fly.toml` | All absent | Correct at Phase 0; prod/dev deploy preconditions fail closed (verified) |
| `AGENTS.md` | Absent; `.agents/` exists but is empty | L5 |

**Inventory arithmetic check (prompt5 §3):** the "17 tasks" claim is a miscount in the report, not a missing task. `task --taskfile Taskfile.dev.yml --list` prints exactly **16** tasks: `doctor` (1) + `dev:db:*` (6) + `dev:*` (2) + `test`, `test:integration` (2) + `fly:*` (5) = 16. The enumerated list (1+6+2+2+5) sums to 16 on its own terms. Nothing is missing; the "17" is wrong. Recorded as INFO, not a defect. Production lists exactly the 8 expected tasks.

## 3. Architecture and Phase 0 correctness

**Contracts validate invariants, not just shapes — with documented deferrals.** `contract/manifest.go:152-197` enforces class/substrate matrix, JSON-object schemas, retry coherence, entity-key-or-justification for effect tools, and non-negative limits; `contract/run.go:71-79` freezes an effect machine in which UNKNOWN can only go to RECONCILING (verified by `TestEffectUnknownCannotRetryDirectly`); `contract/capability.go:60-112` enforces idempotency-required-for-writes, attribution, and operator-approved https origins; `contract/tasks.go:129-134` gives a non-circular release envelope. `contract/doc.go` honestly states that behavioral enforcement lives in the Phase 1+ owning packages. Gaps are real but disclosed: `ApprovalRecord` (`contract/run.go:123-136`) has **no** `Validate` method, so approval-to-intent binding, expiry, and self-approval rejection exist as fields only; `NoEntityWhy` accepts any non-empty string with review deferred to registry admission; blast-radius limits are optional at the structural layer. For a Phase 0 freeze this matches plan4.2 §23.1 condition 1 ("contracts reviewed/versioned"), provided the deferrals stay tracked — they are (M8's artifact table, inventory.md).

**ADR-001 port is faithful.** `docs/adr/001-dbos-adoption.md` correctly reframes `docs/DBOS.md` as principles-not-verbatim for Go/River/DBOS, requires per-workload records with human review, and limits automated admission to presence/arithmetic/state — exactly plan4.2 §10.1. The template (`docs/adr/miniapps/adr001-template.md`) and inventory (River-vs-DBOS PILOT required before Class C enablement) are consistent with R1.

**DBOS family/version isolation: plan sound, behavioral proof blocked.** `docs/phase0/dbos-family-plan.md` correctly refuses to tie workflow versions to manifest versions, requires per-family `AppName`, and defers process-group retention to the Phase 6 drill. `TestDBOSFamilyIsolation` (`spikes/dbos_test.go:201-257`) tests the right thing (two AppNames, one branch, independent completion). But it has **never executed**: Neon access was lost mid-session (incident note in `g0-evidence.md`), and this review environment has no Docker daemon, so neither the Neon nor the local-Docker path could run. Status: BLOCKED, honestly recorded. The family plan's step 2 (version tagging / patch strategy) has no spike at all — accepted as a Phase 6 obligation, but it means "version isolation" today is AppName-only.

**Determinism analyzer: honestly spec-only.** `docs/phase0/determinism-rules.md` freezes the rule set (goroutines, select, map order, time/rand, globals, I/O in bodies, durable calls inside steps) and defers the `go/analysis` linter to Phase 1. No grep-based enforcement is claimed. Correct posture; the risk is that Class D code gets written in Phase 1 before the linter lands — the guide's "run the determinism linter" rows (e.g. §7.5) will be no-ops until then.

**Effect/approval correctness: machine right, boundaries future.** The effect machine, idempotency-required declarations, and reconcile-before-retry are correctly shaped. Transaction boundaries, compensation ordering, and approval binding have no implementation yet because no executor exists — consistent with Phase 0 scope, but the prompt5 §4.4 items must be re-audited when `effects/` and `approvals/` gain code (both directories are empty).

**Tenant isolation: helper correct, regression coverage incomplete (M7).** `store.WithTenant` (`internal/store/store.go:60-81`) begins an explicit transaction, arms `set_config('app.tenant_id', $1, true)` inside it, rejects empty tenants, and rolls back on all failure paths — the plan4.2 R5 pattern. `spikes/rls_test.go` correctly uses a `NOBYPASSRLS` role (superusers bypass RLS unconditionally) and asserts default-deny plus no cross-checkout leak. What is missing: no test reproduces the two discovered failure modes (superuser bypass succeeding, autocommit context loss), so a future regression that reintroduces them would still pass the suite; and the spike has never executed anywhere (BLOCKED). Details in M7.

## 4. Execution-category guidance review

The file called "the execution-category developer guide" is four independent drafts concatenated under `## claude`, `## gemini`, `## deepseek`, `## chatgpt` headings (lines 1, 516, 731, 1053 of a 2296-line file). Individually each part is largely plan-consistent; as a deliverable it is not a guide. See M8 for the finding; the per-class answers below are assessed against the strongest parts, which a developer cannot be expected to identify.

- **Class A:** synchronous inline result, no queue/workflow, effect ledger for writes — correct in all parts (e.g. lines 72-86). "When NOT" rows present.
- **Class B:** River, transactional enqueue, unique keys vs idempotent workers mostly distinguished; terminal-failure observability required as tests. Adequate.
- **Class C:** contradictory across parts. Claude line 107 says "River by default"; chatgpt line 1375 says "Class C has no automatic substrate" while its own line 2028 says "on River by default"; deepseek lines 793/885 forbid checkpoint-resume for Class C while claude/chatgpt require "resume without replaying completed records" as the Class C failure contract. Plan4.2 R1 ("mandatory comparison … rather than hard-coding one default") sides with line 1375. A developer cannot resolve this without an architect.
- **Class D:** durable promise, determinism rules, workflow IDs, version strategy, at-least-once steps — well covered (claude §7.5 is the best section). AppName/version isolation is covered only in the chatgpt part.
- **Cross-cutting vs categories:** approval, saga, keyed serialization, reconciliation, blast radius are correctly framed as patterns, not categories, in the claude and gemini parts.
- **Compute placement independence:** covered almost exclusively by the chatgpt part (43 matches for placement/Sprite/Machine terms vs 1/0/0 in the other three). Three quarters of the "guide" never tells the developer that Dimension B exists.
- **Skeletons reference a fictional API:** claude §6.3-6.5 show `framework.Invocation`, `h.tx.Run`, `h.cap.Call`, `effects.Intent` — none of which exists (all framework packages are empty directories). The header disclaimer ("illustrative shapes") does not prevent a coding agent from copying them as real usage.

## 5. Development and production Taskfile review

Both files parse under Task 3.48.0; `--list` output matches the inventories in §2. Independence holds: neither file contains `includes:`; dev never references `MINIAPP_PROD_APP`; prod never references `miniapp-demo`, Compose, or Postgres. Dry runs are secret-safe (verified `dev:db:up --dry`: commands carry paths and references, never values). Shared scripts introduce no cross-environment path: `secret-input.sh`/`secret-apply.sh` hard-allowlist `miniapp-demo` for `--env dev` and `$MINIAPP_PROD_APP` for `--env prod`, refusing anything else.

- **Development (16 tasks):** all present and correctly wired. Genuine defects: the `dev:db:up`/`test:integration` readiness loops always exit 0 (M3); `test`'s `gofmt -l .` never fails (L1). `dev:db:reset` scoping is good (typed prompt + pinned current-major volume name; only bypass is Task's own `--yes`, which still cannot retarget the volume). `test:integration` correctly unsets its transient `DBPW` and never prints values; it assumes (but does not ensure) secrets + DB exist — a missing `db_password` fails the task with a path-only error, which is fail-closed if unfriendly.
- **Production (8 tasks):** structure is right — every task preconditions on `MINIAPP_PROD_APP`, deploy additionally on org + `fly.toml` + preflight + HEAD + release-verify + typed confirmation. Verified: `deploy --dry` with no env fails closed on the first precondition. Defects: no dirty-tree check (M6); `release:verify` verifies machine count, not release identity (M5); and none of these gates bind direct `flyctl` usage (H1).
- **`miniapp-prod` posture:** correctly unwired — no default anywhere, prod fails closed. The docs' assertion that such an app "exists" could not be verified here (Fly BLOCKED, L9).

## 6. Docker PostgreSQL and volume-safety review

**Image selection: pin verified, one doc sentence falsified.** I checked the claims against Docker Hub and postgresql.org live on 2026-11-22. `postgres:19` and `postgres:19.0` both resolve to digest `sha256:bb6e70ad…687c60`, exactly matching the digest recorded in `docs/development/local-postgres.md:15-16`; `19rc1`/`19beta1` resolve to different digests; postgresql.org confirms PG 19 latestMinor 0 with relDate 2026-11-19 and supported=true. The default pin is a genuine stable GA release. However, the doc's claim (lines 11-13) that "`postgres:19`, `postgres:19.0`, and `postgres:latest` resolve to one digest" is **false**: `latest` currently resolves to `sha256:fe9ef805…` — a different digest. The pin is right; the `latest` sentence must be struck (L2). This is also a live demonstration of prompt5 §7's warning not to treat `latest` as newest-stable.

**Volume safety: refusal logic works, but it guards the wrong comparison (M4).** `postgres-volume-check.sh` never deletes anything (asserted by `TestVolumeCheckNeverDeletes`), refuses mismatched majors, preserves all volumes, and requires `MINIAPP_ALLOW_FRESH_DB=1` for a fresh start — all exercised via a fake-docker shim in `TestVolumeCheckBehavior`. The gap: the script compares the **Taskfile's `POSTGRES_MAJOR` variable** against **volume names**, not the image's actual major against the volume's actual `PG_VERSION`. A mis-set `POSTGRES_IMAGE=postgres:18` with `POSTGRES_MAJOR=19` sails through the check and lets PG18 attempt a PG19 data directory (server refuses to start — data preserved, but a confusing failure the check claims to prevent). `postgres-image.sh` prints the image major but nothing compares it to `POSTGRES_MAJOR`. The doc sentence "compares the image major against existing volumes" (local-postgres.md:41-42) is therefore inaccurate.

**Compose security: good.** Base `compose.yaml` publishes no ports (asserted), uses `POSTGRES_PASSWORD_FILE` with a single-file secret mount (directory never mounted), major-isolated volume names, loopback-only host override in `compose.host.yaml` (asserted, no `0.0.0.0`), healthcheck with sane intervals, and no credential literals anywhere. `TestComposeValidates` parses both files; I additionally ran `docker compose config --quiet` for both combinations — exit 0. Live daemon behavior (health, stop/start persistence, wrong-major refusal, reset confirmation) is BLOCKED here: the Docker socket is not accessible from this environment (`permission denied … /var/run/docker.sock`). The recorded PGDATA claim (`/var/lib/postgresql/19/docker`) is plausible but could not be re-verified for the same reason; the Compose mount (data parent) is correct regardless of that detail.

**Digest pinning gap:** the verified digest lives only in prose. Nothing — not the Taskfile, not Compose, not `postgres-image.sh` — pins or enforces it, so a future tag move is silent. Recommend pinning `image: postgres:19.0@sha256:bb6e70ad…` once the daemon is available to confirm pull-by-digest (L2).

## 7. Go configuration and `_FILE` review

`internal/config/secret.go:14-31` implements the required contract well: lookup-based (set-but-empty counts as set), both-set rejected, neither-set rejected with an actionable message, whitespace trimmed, empty-after-trim rejected, file errors classified (`no such file` / `permission denied` / `read error`) without content, and values never appearing in errors. Unit tests cover direct, file (with trailing newline), conflict, missing, unreadable (absent path + directory), whitespace/empty, redaction, and the DBOS pair. No connection-string logging exists; no fallback between environments exists. `config.LoadPolicy` correctly defaults warn/strict-by-CI.

Two findings:

- **M1 (MEDIUM): unknown `MINIAPP_POLICY` fails open to `warn` outside CI**, contradicting the code comment ("Unknown values fail closed to strict"), the test name's intent, and `docs/ENV_VARS.md`. Proven by execution: `MINIAPP_POLICY=bogus` without `CI` returns `warn`. The existing test only exercises the unknown value with `CI=true`, which masks the bug. A typo (`MINIAPP_POLICY=strict ` with trailing space, `STRICT`, `enforce`) silently weakens validation in exactly the environments where strictness was requested.
- **M10 (MEDIUM): the `_FILE` contract has no consumers and no end-to-end proof.** `DatabaseURL()`/`DBOSSystemDatabaseURL()` are called only by their own tests; no `main`, no pool constructor, no DBOS setup reads them. "Same configuration semantics for local Docker, Fly dev, Fly prod" is therefore a unit-tested resolver, not a demonstrated delivery path. Symlinked or world-readable secret files are also silently accepted (no permission-model check), and keyword-format DSNs defeat the redaction helper (M2).

Threat-model note on arbitrary-path reads: `*_FILE` values are operator-controlled paths, not request-controlled, so `os.ReadFile(path)` on an operator-supplied path is acceptable inside the declared threat model — but the first consumer that derives the path from anything less trusted must re-audit this.

## 8. Secret-management and threat-boundary review

**Local secrets: solid.** `local-secrets-init.sh` uses mode 700/600, `openssl rand`/urandom generation, never overwrites, prints paths only; `TestLocalSecretsInit` proves permissions, idempotency, length, and non-leakage in output. `.gitignore` excludes `.env`, key material, and secrets dirs. Compose mounts only the single required file. The secrets-management doc's "honest threat boundary" for Compose (file mounts, not a vault; no protection against privileged host users) is accurate and commendably frank.

**Fly secret workflows: interface-verified, behavior-blocked.** `secret-input.sh` uses hidden input, `set +x`, stdin import, strict allowlists on both `--env`/`--app` and secret names, stage-only semantics, `unset` cleanup, and prod typed confirmation. I verified against the installed flyctl `--help` that `secrets import --stage` and `secrets deploy` exist with the flags the scripts use. `app-preflight.sh` fails closed on auth failure, unknown app, and org mismatch; `secret-apply.sh` separates stage/apply with metadata-only listing plus confirmation. What could not run here: any live Fly command (sandbox blocks `~/.fly`, so even `auth whoami` fails) — all Fly behavior is BLOCKED, and the scripts' runtime behavior (prompt handling, exit codes, signal cleanup) is reviewed statically plus via `infra`'s property tests only.

**The HIGH finding (H1): production authority is policy, not mechanism.** `docs/operations/fly-production.md:40-45` states "The coding agent holds no production deployment privileges: it may prepare and inspect, never release." Nothing in the implementation enforces this. Any process running as the operator — the coding agent included — inherits `MINIAPP_PROD_APP`/`MINIAPP_PROD_ORG`, `~/.fly` credentials, and a shell: it can satisfy every typed confirmation (they read stdin, not an out-of-band approver) and can bypass the Taskfiles entirely with direct `flyctl` commands. The Taskfile gates constrain only the Taskfile path. Today the blast radius is zero because production is unwired and fails closed — but the moment the owner exports the two prod variables on any agent-accessible host, the agent holds production deployment authority and, via deploy/SSH, indirect access to runtime secrets. The required Fly runtime threat-boundary documentation (deployed code can read injected secrets; deployment authority implies secret access) is also missing from every guide — only the Compose half of the honest boundary is written down.

## 9. Fly and Neon isolation review

- **App selection:** dev target `miniapp-demo` is a hardcoded literal (not overridable); prod target requires explicit env config. Preflight checks app existence/accessibility and org match. No confusion path found in the Taskfile/script layer. Live verification BLOCKED (no flyctl auth in this environment).
- **Neon separation:** there is no Neon configuration anywhere in the repo — correct by design. No code path references a Neon URL; no fallback between local/dev/prod can occur because each delivery path carries only its own credential. `test:integration` builds both `TEST_DATABASE_URL` and `TEST_DBOS_DATABASE_URL` from the same local database, which is fine for spikes (DBOS and app share one branch by design in Phase 0) but means River/RLS/DBOS spikes share one database locally — acceptable, documented nowhere, worth one line in the integration docs.
- **Evidence hygiene:** `docs/evidence/README.md` records exactly two behavioral entries (MCP pass; DBOS atomicity observed-once on Neon) and refuses to record goals as results. The g0 tracker keeps local-PG, observed-once-Neon, and blocked-Neon distinct. The one overclaim found is the `latest`-digest sentence (L2), not a Neon conflation.
- **Unresolved:** whether `miniapp-demo`/`miniapp-prod` exist and in which org (docs assert both; Fly BLOCKED, L9). The g0 incident note (Neon access lost, projects invisible) remains an owner action.

## 10. CI, supply-chain, and script review

**CI (`.github/workflows/ci.yml`):** minimal and safe — no Fly/Neon credentials, strict policy env, pinned actions, `gofmt`/`vet`/`test`/`race`/`checks.sh`. Two gaps: (a) `gofmt -l .` never fails (same flaw as the `test` task, L1); (b) the integration suite that is the *only* executable form of the DBOS/River/RLS spikes never runs in CI (M9) — `ubuntu-latest` runners have Docker and could run `test:integration` today.

**Supply chain:** `go.mod` pins match the version register exactly (DBOS v1.9.0, River + driver v0.51.2, MCP SDK v1.8.0, sprites-go v0.2.2, pgx v5.13.0, jsonschema v0.14.0); `scripts/checks.sh` enforces all seven pins plus enterprise-DBOS exclusion, framework→miniapps import ban, and fixture-credential scan — `ALL-CHECKS-PASS` verified. License notes (River MPL-2.0 file-level, MCP Apache-2.0) are handled. `go vet` clean; `gofmt -l .` clean; race tests pass on `contract` + `internal/...`. The migrate-tool pin (v4.20.1) is docs-only until Phase 1 creates `migrations/` content — appropriate.

**Scripts:** all 8 pass `bash -n`; all set `-u`/`-eu` and never `set -x` (asserted by tests). No `eval`, no unquoted expansions on secret paths, no destructive command under a harmless name, no production default, no secret interpolation in any Taskfile command or dry-run output. Minor hardening nits (no secret exposure): `$APP` interpolated into a python one-liner in preflight, substring tag filter over-matches (`*rc*` refuses any path containing "rc"), single-line secret reads, secrets-init TOCTOU — collected in L7.

## 11. Documentation and developer-experience review

Walking the 14 prompt5 §12 questions as a new developer:

1. Start local PG — yes (`dev:db:up`, local-postgres.md). 2-3. Credential location/names — yes (state dir + `db_password`). 4. Which environment — yes (hardcoded `miniapp-demo` vs env-required prod). 5. `_FILE` vs env — yes, accurately (plus M10 caveat: nothing consumes it yet). 6. Local tests without Neon — yes (`test:integration`), though SKIP texts still say "needs a Neon branch" (L3). 7-8. Fly dev + secret provisioning — yes, copy-pasteable. 9. Prod separation — yes, modulo H1's overclaim. 10-11. Categories and selection — **no**: four contradictory drafts (M8). 12. When NOT to use DBOS — yes in fragments, no as one answer. 13. Where rules/persistence/provider/execution live — partially (domain-core rule in one part only; framework packages don't exist yet). 14. Who authorizes production — stated ("owner"/"operator") but the mechanism is a stdin prompt, not an authorization (H1).

Other doc notes: ADRs are numbered and plan-faithful (all 17 titles match §5; texts are short but sufficient for Phase 0). Phase0 specs are crisp and honestly gated. `docs/plan3_review.md` is still empty (known Q5). Duplicate sources of truth are limited to the guide's four voices and the `latest`-digest sentence. Stale command names: none found — every documented command matches its Taskfile task. `fly.toml` location disagrees between plan4.2 §7 (`fly/`) and both Taskfiles (`./fly.toml`) — L6.

## 11b. `docs/ENV_VARS.md` review (additionally requested)

I checked every row against the code that reads it. The document is accurate, complete, and well-scoped — with one false statement inherited from a code bug.

| Variable | Doc claim | Code truth | Verdict |
|---|---|---|---|
| `DATABASE_URL` / `_FILE` pair | Mutually exclusive; both/neither = error; trim; empty fails; errors name vars/paths only | `secret.go:14-31` + tests: exactly as described | PASS |
| `DBOS_SYSTEM_DATABASE_URL` / `_FILE` | Same pair semantics | Same `Secret()` helper | PASS |
| `MINIAPP_POLICY` | Default warn / strict under CI; **unknown fails closed to strict** | Unknown + no CI returns `warn` (M1, proven by execution) | **FAIL** — doc matches the comment, not the behavior |
| `CI` | `true` selects strict default | `config.go:27` exact match on `"true"` | PASS |
| `MINIAPP_PROD_APP` / `MINIAPP_PROD_ORG` | Required, never defaulted, fail closed | Preconditions in every prod task + script allowlists | PASS |
| `POSTGRES_IMAGE`/`MAJOR`/`USER`/`DB` | Defaults `postgres:19.0`/`19`/`miniapp`/`miniapp_dev` | Match Taskfile vars + Compose defaults + `.env.example` | PASS |
| `MINIAPP_SECRETS_DIR` | State-dir default, mode 700, holds `db_password` 600 | `local-secrets-init.sh:8-10,33` + Compose `:?…}` requirement | PASS |
| `MINIAPP_DB_HOST_PORT` | 5433, loopback via override | `compose.host.yaml:8` | PASS |
| `MINIAPP_ALLOW_FRESH_DB` | `1` acknowledges fresh volume | `postgres-volume-check.sh:26` | PASS |
| `XDG_STATE_HOME`, `HOME` | Derivation chain | Matches scripts + Taskfiles | PASS |
| `DBPW` | Transient inside `test:integration`, unset after | `Taskfile.dev.yml:89`: set, used in URL assembly, `unset DBPW` before `go test` | PASS |
| `TEST_DATABASE_URL`, `TEST_DBOS_DATABASE_URL`, `SPRITE_TOKEN` | Consumers + skip semantics | Match `spikes/*_test.go` gates; skips are explicit with reasons | PASS |
| `MINIAPP_SPIKE_WORKER/DIR/WFID/MARKER` | Crash-probe internals | All four read in `dbos_test.go:261-277` | PASS |
| `SECRET=name`, `db_password` | Correctly classified as non-env-var | Task CLI var + filename respectively | PASS (good trap-avoidance) |

Completeness: nothing material is missing. (`POSTGRES_PASSWORD_FILE` is a Compose literal, not an input; `GOPATH`/`GOWORK` overrides are machine-specific sandbox notes.) Two recommendations, neither a defect: (a) fix the `MINIAPP_POLICY` row together with M1; (b) add one clarifying line that `SECRET: no (path)` on `_FILE`/`MINIAPP_SECRETS_DIR` means the *variable* holds a path while the *file contents* are secret — the current table is technically right but a hurried reader could mis-triage.

## 12. Verification matrix with exact commands and observed results

Environment notes: repo has no commits; Go runs needed `GOWORK=off` (parent `/home/chaschel/Documents/go/go.work`) plus writable `GOPATH`/`GOCACHE` (default cache read-only) — the same overrides `g0-evidence.md` documents. Nothing was committed, deployed, or mutated; no secret-bearing command was run.

| # | Command (from repo root) | Result |
|---|---|---|
| V1 | `git status --short` | Untracked-everything, zero commits — PASS (posture holds) |
| V2 | `task --taskfile Taskfile.dev.yml --list` | 16 tasks, all expected names — PASS (report's "17" is a miscount) |
| V3 | `task --taskfile Taskfile.prod.yml --list` | 8 tasks, all expected — PASS |
| V4 | `task --taskfile Taskfile.dev.yml dev:db:up --dry` | 5 commands, no secret material — PASS |
| V5 | `task --taskfile Taskfile.prod.yml deploy --dry` (no env) | Fails closed: `MINIAPP_PROD_APP is not configured` — PASS |
| V6 | `gofmt -l .` | Empty — PASS |
| V7 | `go vet ./...` (with Go env overrides) | Clean — PASS |
| V8 | `go test ./...` | All 5 test packages ok (`deploy` has no test files) — PASS |
| V9 | `go test -race ./contract/ ./internal/...` | ok × 3 — PASS |
| V10 | `go test ./spikes/ -v` (no env) | 1 PASS (MCP), 8 SKIP with reasons — PARTIAL (skips are not passes) |
| V11 | `bash -n` on all 8 scripts | All OK — PASS |
| V12 | `bash scripts/checks.sh` | `ALL-CHECKS-PASS` — PASS |
| V13 | `docker compose -f compose.yaml config --quiet` (+ host override) | Exit 0 both — PASS (parse only, no daemon needed) |
| V14 | `docker image inspect postgres:19.0` / `docker volume ls` | BLOCKED — socket permission denied from this environment |
| V15 | `dev:db:up`, stop/start persistence, wrong-major refusal live, `dev:db:reset`, `test:integration` | BLOCKED (V14) — NOT RUN; never substitute the shim tests for these |
| V16 | `flyctl auth whoami`, `apps list`, status, secrets metadata | BLOCKED — sandbox makes `~/.fly` read-only; even whoami cannot run |
| V17 | `flyctl secrets import --help`, `secrets deploy --help` (with `HOME` redirected to temp) | `--stage` on import confirmed; deploy confirmed — PASS (interface only) |
| V18 | Docker Hub tag API: `19.0`, `19`, `latest`, `19rc1`, `19beta1` digests | 19.0==19==doc digest; rc/beta differ; `latest` DIFFERS — PARTIAL (doc sentence false, pin true) |
| V19 | `postgresql.org/versions.json` | PG19 minor 0 rel 2026-11-19 supported; PG18 minor 7 exists — PASS (doc dates check out) |
| V20 | Wait-loop emulation (`false && break \|\| sleep` × N; `$?`) | Exits 0 after persistent failure — FAIL (M3, behavior proven) |
| V21 | Throwaway `go test` probe (created, run, removed): `LoadPolicy` with `bogus`/no-CI; `PoolConfig.String` on keyword DSN | Returns `warn`; password echoed verbatim — FAIL (M1, M2, behaviors proven) |
| V22 | `grep` callers of `DatabaseURL()`/`WithTenant()`/`PoolConfig` | Tests + spikes only; zero production callers — PASS-as-evidence (M10) |
| V23 | Per-section term counts in category guide | Placement coverage 1/0/0/43 across the four parts — FAIL-as-evidence (M8) |

Labels per prompt5 §13: V1-V9, V11-V13, V17, V19, V22 = PASS; V10, V18, V23 = PARTIAL/FAIL-as-evidence as noted; V14-V16 = BLOCKED; V15 additionally NOT RUN; V20-V21 = FAIL (demonstrated defects, own the fixes in §15).

## 13. Findings sorted by severity

No CRITICAL findings. One HIGH, ten MEDIUM, ten LOW. INFO observations follow the numbered findings.

```
ID: H1
Title: Production deployment authority is policy, not mechanism; agent-exclusion claim is unenforceable
Severity: HIGH

Affected files: docs/operations/fly-production.md:40-45; Taskfile.prod.yml:52-74;
  scripts/fly/secret-input.sh:41-45; scripts/fly/secret-apply.sh:31-39
Exact line references: fly-production.md lines 42-45 ("coding agent holds no
  production deployment privileges"); Taskfile.prod.yml lines 73, 82-91 (stdin
  confirmations, no identity check)

Requirement / invariant: Only an authenticated human operator with release
  authority can mutate production; the coding agent must have no production
  deployment authority (prompt5 §6/§9; plan4.2 §9.6 review gate).
Observed implementation: Every production gate (env presence, org match, typed
  name confirmation) is satisfiable by any process running as the operator,
  including the coding agent, which inherits MINIAPP_PROD_APP/ORG, ~/.fly
  credentials, and a shell. Confirmations read stdin, not an out-of-band
  approver. Direct `flyctl deploy --app …` / `flyctl ssh` bypass the Taskfiles
  entirely. No doc states the Fly runtime boundary (injected secrets are
  readable by deployed code; deployment authority implies secret access).

Attack or failure scenario: Owner exports the two prod variables on this
  (agent-accessible) host to do legitimate prod work. Any later agent session
  can then run the prod Taskfile tasks (typing the app name at prompts) or
  plain flyctl, deploy attacker-influenced code to the prod app, and exfiltrate
  DATABASE_URL from the process environment.
Why the current behavior is unsafe or incorrect: The strongest prod-isolation
  claim in the docs is false as a technical statement; an owner relying on it
  will configure prod exactly where the agent can reach it.

Reproduction: Static (no prod mutation attempted): `grep -rn whoami Taskfile.prod.yml`
  shows identity is displayed (doctor) but never authorized; all confirmations
  are `read -r` from stdin.
Evidence: V5 (fail-closed today only because env is unset); H1 analysis in §8.
Root cause: Authorization modeled as task UX (prompts) instead of as a
  principal boundary; no separate prod operator identity exists.

Recommended correction: (1) Rewrite fly-production.md: agent exclusion is an
  OPERATING RULE, not an enforced property; document the full Fly runtime
  boundary per prompt5 §9 bullets. (2) Owner decision (gate): prod Fly auth +
  prod env must live only on hosts/processes the agent cannot reach (separate
  machine or operator-only context); record the decision. (3) Optional
  hardening: require a hardware/OTP-backed deploy step outside the agent's
  reach for `deploy`/`secrets:apply`.
Required regression test: Static test asserting the Fly runtime boundary text
  exists in the ops docs (no silent reversion to the overclaim); manual owner
  attestation for the hosting decision.
Owning package / phase: docs/operations + owner; gate before any prod wiring.

Whether this blocks the next development phase: Blocks prod wiring, not
  Phase 1 coding. NO-GO criterion 1 is NOT triggered today (prod unwired,
  fail-closed verified) but triggers the moment prod env exists on an
  agent-accessible host — hence CONDITIONAL GO with this gate.
```

```
ID: M1
Title: Unknown MINIAPP_POLICY fails open to warn outside CI
Severity: MEDIUM

Affected files: internal/config/config.go; internal/config/config_test.go;
  docs/ENV_VARS.md
Exact line references: config.go lines 26-31 (default branch returns warn when
  CI != "true"); ENV_VARS.md line 29 ("Unknown values fail closed to strict")

Requirement / invariant: Unknown policy values fail closed to strict
  (config.go comment line 17; ENV_VARS.md; prompt5 §8 "no unintended fallback").
Observed implementation: `MINIAPP_POLICY=bogus` without CI returns
  PolicyWarn. Only under CI does the default branch return strict.

Attack or failure scenario: Operator sets `MINIAPP_POLICY=strict ` (trailing
  space), `STRICT`, or a typo on a non-CI host; validation silently runs in
  warn mode while everyone believes it is strict.
Why the current behavior is unsafe or incorrect: Fail-open on the strictness
  selector defeats the control it selects; documentation asserts the opposite.

Reproduction: V21 probe — `MINIAPP_POLICY=bogus`, CI unset → LoadPolicy()
  returned "warn" (observed). With CI=true → "strict".
Evidence: V21 output; code read of the default branch.
Root cause: Default branch conflates "unset" (legitimate default) with
  "unrecognized value" (must fail closed).

Recommended correction: In the default branch, return strict when
  MINIAPP_POLICY is non-empty (unknown value) and keep warn only when unset
  and not CI. Update the comment to state the rule precisely.
Required regression test: Table test over {unset, strict, warn, disabled,
  bogus, "STRICT", "strict "} × {CI true/unset}; unknown must be strict in
  both CI modes (extends TestLoadPolicy).
Owning package / phase: internal/config; Phase 1 gate.

Whether this blocks the next development phase: Yes as a gate (small fix).
```

```
ID: M2
Title: PoolConfig.String leaks keyword-format DSN passwords into logs
Severity: MEDIUM

Affected files: internal/store/store.go; internal/store/store_test.go
Exact line references: store.go lines 34-45 (redaction covers URL form only;
  keyword form with no "@" falls through unchanged)

Requirement / invariant: Secrets never appear in logs, errors, or diagnostics
  (ENV_VARS.md conventions; prompt5 §8 item 10).
Observed implementation: `DSN: "host=db user=u password=s3cret dbname=x"`
  renders verbatim, password included. pgx accepts keyword DSNs, so this is a
  reachable input shape for the first pool constructor that logs its config.

Attack or failure scenario: Phase 1 adds `NewPool` logging `cfg` on startup
  failure; an operator-supplied keyword DSN (common when copying Neon
  connection strings in keyword form) lands in CI logs / journald.
Why the current behavior is unsafe or incorrect: A helper whose documented job
  is redaction ("String redacts them for logs") silently passes one DSN shape
  through. Latent today (no callers, V22) but the trap is set for Phase 1.

Reproduction: V21 probe — String() output contained `password=s3cret`
  verbatim (observed).
Evidence: V21 output; code read (no "@" → no redaction branch).
Root cause: Redaction implemented for URL form only.

Recommended correction: Redact `password=`/`passfile=` tokens (quote-aware)
  in the keyword form, or reject non-URL DSNs at the boundary; add both shapes
  plus passwordless URL to the redaction test.
Required regression test: Table test: URL with/without password, keyword with
  password (quoted + unquoted), malformed input — output must never contain
  the password and must preserve host/username.
Owning package / phase: internal/store; Phase 1 gate (before first caller).

Whether this blocks the next development phase: Yes as a gate (small fix).
```

```
ID: M3
Title: dev:db:up and test:integration readiness loops always succeed, even against a dead database
Severity: MEDIUM

Affected files: Taskfile.dev.yml
Exact line references: line 41 (dev:db:up wait loop); line 88
  (test:integration wait loop) — `for …; do … pg_isready … && break || sleep
  5; done` with no post-loop gate

Requirement / invariant: `dev:db:up` must leave a healthy database or fail;
  `test:integration` must test against a live database (prompt5 §7: health,
  stop/start persistence, integration suite).
Observed implementation: After 24 failed pg_isready attempts the loop exits
  with the status of `sleep 5` (0) and the task succeeds. A developer gets
  "up" with no database; the integration suite then runs spikes against a
  dead port and every spike verdict from that run is meaningless.

Attack or failure scenario: No attack needed — stale volume, wrong password
  file, or a daemon hiccup produces a green `dev:db:up` followed by confusing
  downstream failures and wasted debugging against a database that was never
  there.
Why the current behavior is unsafe or incorrect: The task's success signal is
  decoupled from the condition it exists to establish.

Reproduction: V20 — loop-body emulation with persistent failure exits 0
  (observed). Live re-run needs a Docker daemon (BLOCKED here).
Evidence: V20; shell semantics of for-loop exit status.
Root cause: Missing final gate after a best-effort retry loop.

Recommended correction: Append a final `pg_isready` (single attempt, no
  sleep) after each loop so the task fails when the DB never became ready;
  same fix in both tasks.
Required regression test: infra test asserting each wait loop is followed by
  a non-looped readiness gate (static), plus a live negative run once a
  daemon is available (stop the DB mid-wait, expect task failure).
Owning package / phase: Taskfile.dev.yml; Phase 1 gate.

Whether this blocks the next development phase: Yes as a gate (small fix).
```

```
ID: M4
Title: Volume check guards the Taskfile variable against volume names, not the image against the data
Severity: MEDIUM

Affected files: scripts/dev/postgres-volume-check.sh; Taskfile.dev.yml:39;
  docs/development/local-postgres.md:41-42
Exact line references: postgres-volume-check.sh lines 9-25 (compares
  $EXPECTED_MAJOR from argv against `docker volume ls` names only);
  local-postgres.md lines 41-42 ("compares the image major against existing
  volumes" — inaccurate)

Requirement / invariant: Startup must refuse when the selected image major
  does not match the existing volume's data version, without deleting anything
  (prompt5 §7: running server vs expected volume; stale-data bypass).
Observed implementation: The check receives POSTGRES_MAJOR (a Taskfile
  variable, default 19) and compares it to volume *names*. Nothing verifies
  that POSTGRES_IMAGE's actual major equals POSTGRES_MAJOR
  (postgres-image.sh prints `major=` but no caller compares it), and nothing
  inspects the volume's real PG_VERSION. A volume named `miniapp-pgdata-19`
  containing PG18 files (mislabelled run, manual `docker volume create`, or
  IMAGE/MAJOR skew) passes the check; the server then fails to start (data
  preserved, but the check's promise is broken and the error is cryptic).

Attack or failure scenario: Developer exports POSTGRES_IMAGE=postgres:18 to
  test something, forgets POSTGRES_MAJOR; check passes against the 19 volume;
  PG18 refuses the data directory with a generic FATAL while the operator
  trusted "startup refuses on mismatch" to have caught it earlier.
Why the current behavior is unsafe or incorrect: The guard's input
  (declared var) can contradict the guarded reality (image + bytes) with no
  detection; documentation overstates what is compared.

Reproduction: Static + V15 BLOCKED live. Logic review suffices: no data path
  connects `postgres-image.sh`'s `major=` output to the volume decision.
Evidence: Script reads; Taskfile.dev.yml:38-39 (image script output unused
  by the volume script).
Root cause: Name-based proxy check without grounding either side (image
  major, on-disk PG_VERSION).

Recommended correction: (1) In dev:db:up, capture the image major and refuse
  when it differs from POSTGRES_MAJOR before the volume check. (2) State the
  residual honestly: name-based only; on-disk PG_VERSION inspection is a
  documented non-goal until a daemon-backed test exists. (3) Fix the
  local-postgres.md sentence to describe the actual comparison.
Required regression test: Extend the fake-docker shim test with an
  IMAGE-major vs MAJOR-var mismatch case that must refuse; live negative test
  once a daemon exists.
Owning package / phase: scripts/dev + Taskfile.dev; Phase 1 gate.

Whether this blocks the next development phase: Yes as a gate.
```

```
ID: M5
Title: release:verify verifies machine count, not release identity; no immutable release identity exists
Severity: MEDIUM

Affected files: scripts/fly/release-verify.sh; Taskfile.prod.yml:60-82
Exact line references: release-verify.sh lines 21-28 (counts machines with
  `wc -l`, compares to zero); Taskfile.prod.yml lines 70-73 (deploy captures
  no digest; nothing to compare against)

Requirement / invariant: Production operations must validate immutable
  build/release identity (prompt5 §6); release:verify must verify "deployed
  release identity and health" (Taskfile.prod.yml:77).
Observed implementation: `--expect-deployed` passes when >=1 machine exists —
  including stopped, draining, or wrong-release machines. The deploy task
  records no image digest, no source revision binding, and no expected value,
  so there is literally nothing for any verifier to compare. "Release
  identity verification" is currently a liveness headcount.

Attack or failure scenario: Deploy pushes revision B but an old machine on
  revision A lingers; release:verify reports OK; rollback:prepare inspects the
  same ambiguous state. No step can answer "is prod running the digest I just
  approved?"
Why the current behavior is unsafe or incorrect: The task name and doc promise
  an identity check that does not exist; operators will trust it during the
  highest-stakes operations (deploy, rollback).

Reproduction: Static (Fly BLOCKED; no fly.toml exists so deploy cannot run
  anyway). Code read of the count-vs-compare logic.
Evidence: V16 BLOCKED; script read.
Root cause: Phase 0 has no artifact/digest pipeline (no Dockerfile, no build
  provenance), so identity verification was stubbed as presence verification.

Recommended correction: (1) Rename or re-scope honestly until Phase 5:
  "release presence check". (2) With Phase 1 deployment work: capture the
  deployed image digest at deploy time, persist it as the expected value, and
  make release:verify compare machine image digests (and stopped/started
  state) against it; fail on any divergence.
Required regression test: Fake-flyctl harness: matching digests pass;
  one stale machine fails; zero machines fails.
Owning package / phase: scripts/fly + Taskfile.prod; before any prod use
  (not before Phase 1 coding).

Whether this blocks the next development phase: No for Phase 1 coding; YES
  before first production deployment.
```

```
ID: M6
Title: Production deploy gates on HEAD existing, not on a clean, recorded tree
Severity: MEDIUM

Affected files: Taskfile.prod.yml
Exact line references: line 71 (`git rev-parse --verify HEAD` — passes with
  uncommitted changes); no `git status --porcelain` cleanliness check, no tag,
  no recorded revision anywhere

Requirement / invariant: Production release evidence must identify the actual
  source revision and build artifact (prompt5 §11); deploy requires "immutable
  build/release identity" (prompt5 §6).
Observed implementation: The deploy task verifies that HEAD exists but never
  checks that the working tree is clean, never requires a tag, and never
  records which revision was deployed. Dirty-tree code can ship while every
  observer assumes "HEAD" describes it.

Attack or failure scenario: Operator has unrelated local edits (debug flags,
  reverted-later experiment) and runs prod deploy; the shipped image contains
  them; release:verify cannot detect the drift (M5); the incident review finds
  "HEAD = clean commit X" and chases the wrong cause.
Why the current behavior is unsafe or incorrect: The one revision gate in the
  prod path attests to less than its name implies, and nothing else records
  provenance.

Reproduction: Static (no fly.toml, so deploy cannot run; inspection of the
  precondition list).
Evidence: Taskfile.prod.yml:60-74 read; V5 (fails closed today only for
  missing env + missing fly.toml).
Root cause: Provenance treated as a presence check instead of a cleanliness +
  identity record.

Recommended correction: Add a cleanliness precondition
  (`test -z "$(git status --porcelain)"`), require the tree to match a tag or
  record `git rev-parse HEAD` + `git status` fingerprint into the deploy log
  output, and carry that identity into M5's release-verify comparison.
Required regression test: Fake-git harness: dirty tree refuses deploy; clean
  tree proceeds and prints the recorded revision.
Owning package / phase: Taskfile.prod; before any prod use.

Whether this blocks the next development phase: No for Phase 1 coding; YES
  before first production deployment.
```

```
ID: M7
Title: RLS failure-mode regressions missing; tenant-isolation spike never executed
Severity: MEDIUM

Affected files: spikes/rls_test.go; internal/store/store.go
Exact line references: rls_test.go lines 96-128 (asserts correct-path
  behavior only); no test demonstrates superuser RLS bypass or autocommit
  context loss

Requirement / invariant: "The previously discovered RLS superuser-bypass and
  autocommit defects must have regression tests that genuinely reproduce the
  failure mode" (prompt5 §4.5); tenant isolation proven with app roles.
Observed implementation: TestRLSTenantContext correctly uses a NOBYPASSRLS
  role and asserts default-deny + isolation + no leak through WithTenant —
  the happy path through the correct helper. It does NOT show that (a) the
  same queries as superuser/table-owner bypass the policy (proving the test
  would catch a privilege regression), or (b) arming context outside a
  transaction loses it (proving the autocommit hazard is real and guarded).
  Further, the test has never executed anywhere: Neon access was lost before
  the first run, and no local-Docker run has been recorded either.

Attack or failure scenario: A Phase 1 refactor runs migrations as a table
  OWNER role for app queries (owners bypass RLS like superusers) — the suite
  stays green because no test pins the bypass behavior; tenant isolation is
  silently gone.
Why the current behavior is unsafe or incorrect: A regression test that cannot
  fail when the defect returns is documentation, not a test. Combined with
  zero executions, tenant isolation is currently trusted, not proven.

Reproduction: Read rls_test.go:96-128 — no superuser/autocommit case exists.
  Execution: V10 shows SKIP; V15 BLOCKED.
Evidence: V10, V15; code read.
Root cause: Spike written to prove the mechanism, not to pin the two
  discovered failure modes; infrastructure loss blocked all execution.

Recommended correction: (1) Add two negative cases: same-table query as the
  bootstrap superuser sees all rows (documents the bypass and why app roles
  must be NOBYPASSRLS non-owners); context armed then committed separately
  from the tenant query loses effect (documents the autocommit hazard).
  (2) Execute the full spike once via test:integration on local Docker and
  record the evidence entry; re-run on Neon when access is restored.
Required regression test: The two negative cases themselves, run green in CI
  via M9.
Owning package / phase: spikes + internal/store; Phase 1 gate (local run).

Whether this blocks the next development phase: Yes as a gate (execute at
  least the local run; Neon re-run stays an owner action).
```

```
ID: M8
Title: Execution-category "guide" is four unconsolidated drafts with direct contradictions
Severity: MEDIUM

Affected files: docs/app_category_guidelines.md (2296 lines)
Exact line references: part boundaries lines 1/516/731/1053; Class C
  substrate: line 107 ("River by default") vs line 1375 ("no automatic
  substrate") vs line 2028 (same part as 1375: "on River by default");
  Class C resume: lines 793/885 ("no requirement to resume from a
  checkpoint") vs §3.3 failure contract ("resume without replaying completed
  records"); placement coverage 1/0/0/43 term matches per part (V23);
  fictional APIs §6.3-6.5 (framework.Invocation, h.tx.Run, h.cap.Call)

Requirement / invariant: One developer guide consistent with plan4.2 and
  ADR-001, covering the A/B/C/D table, when-NOT for each class, cross-cutting
  patterns, and placement independence (prompt5 §5).
Observed implementation: Four model-voiced drafts concatenated without
  editing. A developer asking "what substrate for Class C?" or "does Class C
  resume from checkpoints?" receives mutually exclusive answers, one of which
  contradicts plan4.2 R1 (which sides with line 1375: mandatory comparison,
  no blanket default). 75% of the file never mentions Dimension B. Skeletons
  teach APIs that do not exist anywhere in the repo.

Attack or failure scenario: A coding agent (the primary consumer) ingests the
  file top-down, adopts "River by default" + "no checkpoint resume" for a
  batch with per-record effects, and builds exactly the shadow workflow engine
  ADR-001 Gate 4 exists to prevent — citing the guide.
Why the current behavior is unsafe or incorrect: Contradictory normative
  guidance is worse than sparse guidance: every choice made from it is
  defensible and half of them are wrong per the plan.

Reproduction: Read the cited lines; V23 counts.
Evidence: V23; line citations above.
Root cause: Four parallel drafts merged by concatenation instead of
  consolidated by edit.

Recommended correction: Rewrite as ONE guide (keep the best of each: claude
  structure + §7 tables, chatgpt placement chapter, deepseek Gate-4 caveat),
  resolving: Class C = mandatory comparison, no blanket default; Class C
  resume = per-record progress without DBOS checkpoints (define the boundary
  precisely); placement in every class section; skeletons marked
  pseudo-code-only until the framework API exists, or rewritten against real
  SDK calls.
Required regression test: Static doc test: single H1 voice (no `## <model>`
  part headers), required terms present per class section (substrate,
  when-not, placement pointer), and a banned-fictional-API list
  (framework.Invocation etc.) absent outside a marked pseudo-code fence.
Owning package / phase: docs; Phase 1 gate (agents will build from this).

Whether this blocks the next development phase: Yes as a gate.
```

```
ID: M9
Title: The integration suite — the only executable form of the DBOS/River/RLS spikes — runs nowhere automatically
Severity: MEDIUM

Affected files: .github/workflows/ci.yml; Taskfile.dev.yml:84-89
Exact line references: ci.yml lines 23-27 (unit tests + race on skips only;
  no Docker-based integration job); Taskfile.dev.yml:84-89 (test:integration
  exists but is manual-only)

Requirement / invariant: Regression tests must actually execute; skipped
  tests are not passes (prompt5 §2); the DBOS/River/RLS invariants need
  continuous proof (plan4.2 §23 gates).
Observed implementation: CI runs `go test ./...` (spikes skip without env)
  and a race job over the same skipping tests. `test:integration`, which
  wires local Docker PG into TEST_*_URL so the spikes genuinely execute, is
  a manual task. The highest-value tests in the repo — atomic commit,
  crash recovery, family isolation, transactional enqueue, tenant isolation —
  have no scheduled execution anywhere, on any branch, ever.

Attack or failure scenario: A dependency bump (DBOS v1.9.1, River v0.52)
  subtly changes RunAsTransaction or InsertTx semantics; CI stays green on
  skips; the breakage surfaces months later during a production incident.
Why the current behavior is unsafe or incorrect: The suite's green signal
  currently attests to compilation of the spikes, not to any invariant.

Reproduction: Read ci.yml; V10 shows 8/9 spikes SKIP in a default run.
Evidence: V8-V10; ci.yml read.
Root cause: Integration job never added (local-Docker path is new in this
  phase; CI predates it).

Recommended correction: Add a CI job on ubuntu-latest (Docker available):
  start the Compose DB with a throwaway secrets dir, run `test:integration`
  (or its commands), then `dev:db:down`; fail the build on any spike failure
  (skips fail too in this job — env is present so nothing should skip).
Required regression test: The CI job itself; assert in-repo that ci.yml
  references the integration path (static guard against silent removal).
Owning package / phase: CI + Taskfile.dev; Phase 1 gate.

Whether this blocks the next development phase: Yes as a gate.
```

```
ID: M10
Title: _FILE configuration contract is unit-tested but has no consumers and no end-to-end proof
Severity: MEDIUM

Affected files: internal/config/secret.go; internal/store/store.go;
  (absent: any main, pool constructor, DBOS setup)
Exact line references: V22 — DatabaseURL()/DBOSSystemDatabaseURL() callers:
  secret.go (definitions) + secret_test.go only; WithTenant callers:
  rls_test.go only

Requirement / invariant: "Same configuration semantics for local Docker, Fly
  development, and Fly production" with "no unintended fallback" and "no
  environment-specific Go code" (prompt5 §8).
Observed implementation: The resolver is correct in isolation (all 12 prompt5
  §8 items hold at unit level except file-permission-model, which is
  unchecked: world-readable/symlinked secret files are silently accepted).
  But no application entrypoint, pool constructor, or DBOS initializer calls
  it, so cross-environment behavioral identity is unproven: nothing
  demonstrates local `_FILE` delivery and Fly env delivery converging on the
  same running config, or that third-party errors (pgx, DBOS SDK) cannot leak
  the resolved URL indirectly.

Attack or failure scenario: Phase 1 wires the resolver into startup and a
  pgx parse error echoes the DSN into a startup log — the unit-tested
  redaction boundary (our code) holds while the real leak (their error)
  was never probed.
Why the current behavior is unsafe or incorrect: Phase 0 claims a delivery
  contract across three environments with zero delivery paths built; the
  highest-risk items (indirect leaks via third-party errors, permission
  model) sit exactly in the unbuilt region.

Reproduction: V22 grep; read internal/config + internal/store (no main
  package exists in the module).
Evidence: V22; package listing (no cmd/, no main).
Root cause: Correct Phase 0 scoping (resolver before consumers) without a
  tracked E2E obligation.

Recommended correction: (1) Before the first consumer lands: add a startup
  probe test that resolves both pairs via _FILE and via env and asserts
  identical effective config minus source; wrap third-party dial/parse errors
  through a redacting boundary and test with a malformed URL containing a
  canary password. (2) Decide and document the permission model (refuse
  group/world-readable secret files? follow symlinks?) and test it.
Required regression test: The startup-equivalence probe + canary-leak test;
  static test that every *_FILE consumer goes through config.Secret.
Owning package / phase: internal/config + first Phase 1 consumer; gate on
  first consumer (track now).

Whether this blocks the next development phase: No for starting Phase 1;
  YES before the first production-shaped startup path lands.
```

```
ID: L1
Title: gofmt gate is decorative in `task test` and CI; checks.sh skips infra/
Severity: LOW
Affected files: Taskfile.dev.yml:80; .github/workflows/ci.yml:23; scripts/checks.sh:11
Exact line references: `gofmt -l .` exits 0 even when files are listed;
  checks.sh formats-check covers only `contract deploy internal spikes`
Requirement / invariant: Formatting enforced by task + CI (prompt5 §11/§13).
Observed implementation: `task test` and the CI gofmt step print offenders
  but never fail on them. checks.sh does fail, but only over 4 dirs —
  infra/ and Taskfile-adjacent Go files pass everywhere unformatted.
Attack or failure scenario: None security-relevant; slow style drift, and a
  false sense that `task test` covers formatting.
Why the current behavior is unsafe or incorrect: A gate that cannot fail
  teaches developers to ignore gates.
Reproduction: `gofmt -l .` exit code is 0 by design (gofmt(1)); V6/V8 pass
  regardless.
Evidence: V6, V8, V12; checks.sh:11 dir list.
Root cause: `gofmt -l` mistaken for a check.
Recommended correction: `test -z "$(gofmt -l .)"` in the task and CI;
  add `infra` (+ future dirs) to checks.sh, or check `.` there too.
Required regression test: infra static test asserting the task/CI use the
  failing form.
Owning package / phase: Taskfile.dev + CI; opportunistic.
Whether this blocks the next development phase: No.
```

```
ID: L2
Title: `latest`-digest claim falsified; verified digest enforced nowhere
Severity: LOW
Affected files: docs/development/local-postgres.md:11-16; Taskfile.dev.yml:7;
  compose.yaml:17
Exact line references: local-postgres.md lines 11-13 ("19, 19.0, latest
  resolve to one digest")
Requirement / invariant: Docs describe what is true; mutable tags distinguished
  from immutable digests (prompt5 §7).
Observed implementation: V18 proves `latest` → sha256:fe9ef805… while
  19/19.0 → sha256:bb6e70ad… (the recorded digest, which IS correct). The pin
  is right; the sentence is wrong. Separately, the recorded digest appears in
  no enforcement point (Taskfile/Compose/script all use the mutable tag).
Attack or failure scenario: A future tag move changes the local DB version
  silently; a reader trusting the `latest` sentence misjudges tag semantics.
Why the current behavior is unsafe or incorrect: A "verified <date>" claim
  that is false on its own verification date erodes trust in adjacent claims.
Reproduction: V18 (Docker Hub tag API, observed 2026-11-22).
Evidence: V18 digests; V19 (release-date claims check out).
Root cause: Digest comparison presumably run once against a transient state,
  or `latest` assumed without checking.
Recommended correction: Strike the `latest` clause; pin
  `image: postgres:19.0@sha256:bb6e70ad…` in Compose once a daemon confirms
  pull-by-digest; make postgres-image.sh warn on tag/digest drift.
Required regression test: Static test that Compose pins a digest; periodic
  (non-blocking) Hub re-verification note.
Owning package / phase: docs/development + Compose; opportunistic.
Whether this blocks the next development phase: No.
```

```
ID: L3
Title: Spike skip texts and spike docs still say "needs a Neon branch" though local Docker now works
Severity: LOW
Affected files: spikes/doc.go; spikes/dbos_test.go:94; spikes/river_test.go:45;
  spikes/rls_test.go:39
Exact line references: skip strings "…unset (Q4): … spike needs a Neon branch"
Requirement / invariant: Docs describe what the code actually does (prompt5 §12).
Observed implementation: Since test:integration wires local Docker PG into
  TEST_*_URL, the Neon branch is one option, not a requirement — but every
  skip message and the package doc still name Neon as the need.
Attack or failure scenario: Developer without Neon access reads the skip,
  concludes spikes are unrunnable, and never runs test:integration.
Why the current behavior is unsafe or incorrect: Stale guidance suppresses the
  exact test execution M7/M9 require.
Reproduction: V10 output shows the Neon-branch skip strings.
Evidence: V10; file reads.
Root cause: Docs predate the local-Docker integration path.
Recommended correction: Reword to "unset: run test:integration or export …
  (local Docker or Neon branch)".
Required regression test: Static test that skip strings mention
  test:integration.
Owning package / phase: spikes; with M7.
Whether this blocks the next development phase: No (fix alongside M7).
```

```
ID: L4
Title: Stray zero-byte `spikes/started` artifact sits in the repo
Severity: LOW
Affected files: spikes/started (0 bytes, dated 2026-11-22 18:38)
Exact line references: file itself (no content)
Requirement / invariant: No junk artifacts; nothing unintended gets committed
  (prompt5 §3: record discrepancies).
Observed implementation: An empty file named `started` — almost certainly a
  misdirected crash-probe marker (cf. dbos_test.go "started"/"release" files,
  which belong in a TempDir, never in the repo).
Attack or failure scenario: Committed by accident; confuses the next reader;
  if a future test writes relative-path `started`, repo state interferes.
Why the current behavior is unsafe or incorrect: Hygiene; also hints a spike
  once ran with a repo-relative dir.
Reproduction: `ls -la spikes/` (observed).
Evidence: Directory listing (0-byte file among 6 test files).
Root cause: Manual or crashed test run with wrong working directory.
Recommended correction: Delete the file (owner-authorized cleanup); add a
  static test that spikes/ contains only *.go + allowlisted files.
Required regression test: That directory-hygiene test.
Owning package / phase: spikes; opportunistic.
Whether this blocks the next development phase: No.
```

```
ID: L5
Title: No AGENTS.md though agents are the primary builders
Severity: LOW
Affected files: (absent) AGENTS.md; .agents/ (empty)
Exact line references: n/a — `ls AGENTS.md` fails; prompt5 §3 asks to inspect it
Requirement / invariant: Repository instructions for the agent workflow the
  project actually uses.
Observed implementation: An agent-built repo with no agent instructions:
  no go.work/GOWORK note (every Go command needs overrides here), no
  no-commit rule, no secret-handling pointer.
Attack or failure scenario: A fresh agent session improvises Go flags,
  secret handling, or commit behavior inconsistently.
Why the current behavior is unsafe or incorrect: Process gap, not code gap —
  but this repo's velocity depends on agents.
Reproduction: `ls AGENTS.md` → no such file (observed).
Evidence: Directory listing.
Root cause: Never written.
Recommended correction: Add a short AGENTS.md: build/test commands with the
  required env overrides, never-commit + never-print-secrets rules, pointers
  to ENV_VARS.md and the (consolidated, M8) category guide.
Required regression test: Static test that AGENTS.md exists and names the
  Go overrides.
Owning package / phase: docs; opportunistic.
Whether this blocks the next development phase: No.
```

```
ID: L6
Title: fly.toml location disagrees between plan (§7: fly/) and Taskfiles (./fly.toml)
Severity: LOW
Affected files: docs/plan4.2.md §7 layout; Taskfile.dev.yml:118;
  Taskfile.prod.yml:67
Exact line references: plan shows `fly/  fly.toml, …`; both Taskfiles test
  `-f fly.toml` at repo root
Requirement / invariant: One agreed location (prompt5 §6/§12: no stale names,
  no duplicate sources of truth).
Observed implementation: Both must agree before Phase 1 creates the file —
  today they do not. Failure direction is safe (preconditions fail closed),
  but Phase 1 will trip over it.
Attack or failure scenario: Phase 1 adds fly/fly.toml per the plan; both
  fly:deploy and prod deploy refuse with "no fly.toml exists yet"; operator
  confusion, or a hasty second copy at root that then drifts.
Why the current behavior is unsafe or incorrect: A certain future papercut at
  the worst moment (first deploy attempt).
Reproduction: Read the three references (observed).
Evidence: File reads.
Root cause: Layout plan vs task implementation written independently.
Recommended correction: Owner picks one location; update the other two
  references + both guides in the same change.
Required regression test: Static test asserting plan text and Taskfile
  preconditions name the same path.
Owning package / phase: docs + Taskfiles; with Phase 1 deploy work.
Whether this blocks the next development phase: No.
```

```
ID: L7
Title: Script hardening nits (preflight interpolation, tag filter, single-line reads, init race)
Severity: LOW
Affected files: scripts/fly/app-preflight.sh:19-24;
  scripts/dev/postgres-image.sh:11-17; scripts/fly/secret-input.sh:48-51;
  scripts/dev/local-secrets-init.sh:20-31
Exact line references: (a) `'$APP'` interpolated into a python -c program;
  (b) `*rc*` (also matches any path containing "rc", e.g. registries);
  `*dev*` (refuses registries containing "dev"); (c) `read -rs` reads one
  line; (d) write_once check-then-write without locking
Requirement / invariant: No injection, no surprising refusals, correct secret
  handling under concurrency (prompt5 §11).
Observed implementation: (a) A quote in --app breaks the python (fail-closed:
  empty org → REFUSED) — self-injection only, no privilege boundary crossed.
  (b) Over-broad substring filters can refuse legitimate registries and say
  nothing about official-image provenance. (c) Multi-line values silently
  truncate (fine for URLs; undocumented). (d) Two concurrent inits can both
  pass the existence check and one overwrites (mid-start password churn).
Attack or failure scenario: All require operator-shaped input or operator-run
  concurrency; none crosses a trust boundary. Worst case is a refused
  legitimate image or a churned dev password (dev-only, recoverable).
Why the current behavior is unsafe or incorrect: Minor robustness gaps in
  otherwise careful scripts; worth fixing while the scripts are small.
Reproduction: Static reads (observed).
Evidence: File reads.
Root cause: Normal first-pass scripting.
Recommended correction: (a) Pass APP via argv/env to python; (b) match the
  tag component after the last `:`/`/` instead of substrings, and document
  that official-image provenance is operator responsibility; (c) document
  single-line limitation; (d) `set -o noclobber` or lockfile in write_once.
Required regression test: Unit cases per sub-item (quote in app name;
  registry-path tags; concurrent init leaves one stable password).
Owning package / phase: scripts/; opportunistic.
Whether this blocks the next development phase: No.
```

```
ID: L8
Title: Contract validation nits (capability index dropped, negative retries, envelope Matches partial)
Severity: LOW
Affected files: contract/manifest.go:174-178; contract/manifest.go:73-76;
  contract/tasks.go:129-134
Exact line references: `_ = i` discards the capability index from error
  paths; RetryPolicy accepts negative MaxAttempts; Matches() compares
  Manifest + Implementation only, ignoring Build/SourceRevision/Artifact
Requirement / invariant: Precise errors; no invalid policy accepted; envelope
  integrity actually checked (prompt5 §4.1).
Observed implementation: Three small gaps: capability errors all share one
  field path (harder debugging); `maxAttempts: -1` validates; and the only
  existing envelope check ignores 3 of 5 bound fields, so tampering with the
  artifact digest would still "match".
Attack or failure scenario: Debugging friction (index); a nonsense retry
  policy admitted (negative); a registry built naively on Matches() would
  accept artifact substitution.
Why the current behavior is unsafe or incorrect: Each is a small hole in an
  otherwise tight validation layer; the Matches() one sets a trap for the
  Phase 1 registry author.
Reproduction: Static reads (observed).
Evidence: File reads.
Root cause: Normal first-pass contracts.
Recommended correction: Include the index in capability error fields;
  reject MaxAttempts < 0; either make Matches compare all bound fields or
  rename/document it as an identity pre-check with a stronger Verify for the
  registry to call.
Required regression test: Negative-retry rejection; index in error paths;
  envelope mismatch on artifact digest rejected.
Owning package / phase: contract; opportunistic (Matches before registry).
Whether this blocks the next development phase: No.
```

```
ID: L9
Title: miniapp-demo / miniapp-prod existence-and-org claims unverifiable here, cited as fact in docs
Severity: LOW
Affected files: docs/development/fly-development.md:3-4;
  docs/operations/fly-production.md:15-17
Exact line references: "(org `personal`, verified present)"; "A Fly
  application named `miniapp-prod` exists but is explicitly unwired"
Requirement / invariant: Do not assume a cloud resource is correctly
  configured because it exists; do not invent infrastructure state (prompt5 §2).
Observed implementation: Both claims may be true, but this review could not
  verify either (V16 BLOCKED) and no evidence artifact (apps-list output,
  date, command) is recorded. The claims are load-bearing: preflight org
  checks and the whole dev/prod story assume them.
Attack or failure scenario: Stale claims (app deleted, moved orgs) surface
  as mysterious preflight REFUSEDs later.
Why the current behavior is unsafe or incorrect: Citations without artifacts
  for infrastructure state the plan explicitly says not to assume.
Reproduction: V16 BLOCKED (observed); no artifact under docs/evidence/.
Evidence: V16; evidence-index read (no Fly entries).
Root cause: Claims recorded from a live session without persisting output.
Recommended correction: Run `flyctl apps list --json` once from an
  authenticated shell, redact nothing (metadata only), and file the output
  hash + date as an evidence entry; re-verify on any preflight surprise.
Required regression test: Evidence entry exists (manual, one-time).
Owning package / phase: owner + docs/evidence; opportunistic.
Whether this blocks the next development phase: No.
```

```
ID: L10
Title: RLS spike hardcodes a role password; test:integration shares one DB for all spikes without saying so
Severity: LOW
Affected files: spikes/rls_test.go:52,72; Taskfile.dev.yml:89
Exact line references: `const appPassword = "spike-app-only-do-not-reuse"`;
  both TEST_*_URL built from the same local database
Requirement / invariant: No credential material in tracked files; test
  topology documented (prompt5 §7/§11).
Observed implementation: The hardcoded password is for a disposable,
  cleanup-dropped role on a disposable database, is never logged, and is
  clearly named — acceptable, but it is still a password-shaped literal in
  git that someone will copy-paste one day. The shared-DB topology is fine
  (DBOS+app share one branch by design) but undocumented.
Attack or failure scenario: Copy-paste of the literal into a non-disposable
  context; spike cross-talk (river schema vs dbos tables vs spike tables)
  confusing a future failure.
Why the current behavior is unsafe or incorrect: Low-probability hygiene
  issues in test code.
Reproduction: Static reads (observed).
Evidence: File reads.
Root cause: Test convenience.
Recommended correction: Generate the role password at runtime (crypto/rand
  + hex); add one line to local-postgres.md noting the shared local DB.
Required regression test: Static test that no `PASSWORD '` literal with a
  constant survives in spikes/ (allowlist-free).
Owning package / phase: spikes + docs; opportunistic.
Whether this blocks the next development phase: No.
```

### INFO observations (no defect)

| ID | Observation |
|---|---|
| I1 | Empty package dirs (`approvals audit auth capability effects executor/* fly miniapps registry runtime testkit tokens transport/*`) are placeholders, not packages. Expected at Phase 0; nothing enforces their future shape — fine, but the M8 doc test should also ban guide references to not-yet-existing package APIs. |
| I2 | 16-vs-17 task arithmetic reconciled: the report miscounted (1+6+2+2+5=16); `task --list` prints 16. No missing task. |
| I3 | PG 19.0 GA status, release date (2026-11-19), and recorded digest all verified against postgresql.org + Docker Hub; PG 18.7 (g0 incident note) is a real released minor. Only the `latest` clause (L2) is wrong. |
| I4 | `flyctl secrets import --stage` + `secrets deploy` flags verified against installed flyctl `--help`. The stage/apply split rests on real CLI support. |
| I5 | Determinism analyzer honestly spec-only; threat model honestly DRAFT unsigned; g0 tracker honestly BLOCKED/PARTIAL where blocked. The repo's evidence discipline is good — preserve it. |
| I6 | `Secret()` conflict/missing/empty/trim/redaction semantics are exactly as documented; preflight org-check logic is sound (modulo L7a). |
| I7 | Toolchain note: local `go1.27.3` vs `go.mod` 1.26 vs CI 1.26 — a floor, not a mismatch; no action. The parent-dir `go.work` + read-only GOCACHE overrides are already documented in `g0-evidence.md` (promote to AGENTS.md per L5). |
| I8 | `dev:db:reset`'s Task `prompt:` can be auto-answered by `task --yes`, but the volume name is pinned, so the blast radius cannot be retargeted. Acceptable. |
| I9 | Compose auto-loads a repo-root `.env` if a developer creates one from `.env.example`; `.gitignore` covers it. Consider one line warning against putting `DATABASE_URL` there (a sourced direct value + `_FILE` set = confusing conflict error). |

## 14. Unresolved risks and owner actions

| # | Risk / action | Owner | Needed by |
|---|---|---|---|
| O1 | Decide and record where prod Fly auth + prod env may live (never on an agent-accessible host); see H1 | Owner | Before any prod wiring (gate) |
| O2 | Restore Neon access (Q4-Neon) and re-run DBOS/River/RLS spikes on the `phase0/g0-spikes` branch; file evidence | Owner | G0-Core close (M7 local run unblocks Phase 1 meanwhile) |
| O3 | Name the security owner (Q9) and sign the threat model; adopt the Fly runtime boundary text from H1 | Owner | G0-Core-7 |
| O4 | Accept prompt2 Appendix D or supply the remainder (Q8); name first harness (Q2); remaining Q1/Q3/Q5-Q7 per tracker | Owner | Respective gates (unchanged) |
| O5 | File the Fly apps-list evidence artifact (L9) from an authenticated shell | Owner/operator | Opportunistic |
| O6 | Authorize the first git commit (repo still has zero commits; M6's provenance story starts there) | Owner | Before Phase 1 work |
| O7 | Verify live Docker behaviors V15 (health, persistence, wrong-major refusal, reset, integration) on a daemon host | Operator | Phase 1 gate (with M3/M4 fixes) |

## 15. Prioritized corrective-action backlog

**Gate A — before Phase 1 coding starts (small, mechanical):** M1 (fail-closed policy + test), M2 (keyword-DSN redaction + test), M3 (readiness gates + static test), L1 (failing gofmt form), L3 (skip texts), L4 (remove stray file).

**Gate B — before Phase 1 produces its vertical slice:** M4 (image-major grounding + doc fix), M7 (RLS negative cases + one local `test:integration` execution with evidence), M8 (single consolidated category guide + doc tests), M9 (CI integration job), O7 (live Docker verification V15).

**Gate C — before any production use (not before Phase 1):** H1 (boundary docs + O1 decision), M5 (real release-identity verification), M6 (clean-tree + recorded revision), L6 (fly.toml location), M10-second-half (startup E2E + permission model at first consumer).

**Opportunistic (any order):** L2 (strike `latest` clause; digest pin), L5 (AGENTS.md), L7 (script nits), L8 (contract nits), L9/L10 (evidence + test hygiene), ENV_VARS clarification line (§11b-b). Implementation of all corrections requires separate authorization per prompt5 §1 — this report makes no code changes.

## 16. Readiness recommendation for the next development phase

**CONDITIONAL GO for Phase 1.** The Phase 0 contracts, config boundary, environment isolation, and evidence discipline are sound enough to build on, and every NO-GO criterion in prompt5 §17 currently holds in the safe direction — narrowly, in the case of criterion 1, and only because production is unwired. The conditions are Gates A and B above; Gate C guards production, not Phase 1.

Three things to carry into Phase 1 explicitly:

1. **The secret boundary is now documented for Compose but not for Fly.** Write the Fly runtime boundary down (H1) before anyone is tempted to configure prod on a development host. The typed confirmations are UX, not authorization.
2. **The first consumers of `config` and `store` are where the latent findings detonate** (M2 redaction, M10 permission model and third-party error leaks, L8 Matches). Build the redacting error boundary and the startup-equivalence probe in the same change as the first caller, not after.
3. **Agents will build from the category guide — it must first become one guide** (M8). Until consolidation lands, point agents at plan4.2 §10 + ADR-001 + the contract package, not at the four-draft file.

What was reviewed: every Go file, all 8 scripts, both Taskfiles, both Compose files, CI, all 17 ADRs + miniapps records, all 10 phase0 specs, all 6 dev/ops guides, ENV_VARS.md, the evidence tracker, and the plan4.2 baseline — with fresh execution of every check this sandbox permits (V1–V23). What was not verified: live Docker behavior, live Fly behavior, and any Neon-backed execution (all BLOCKED here, all owned above). No file other than this report was created or modified.
