# plan4.1 — Adversarial Review and Vision Pivot

- **Date:** 2026-11-22
- **Subject:** `docs/plan4.1.md` (Consolidated Implementation Plan v2, "frozen baseline")
- **Judged against:** ADR-001 in `DBOS.md` (Gates 1–4, §7 obligations, §8 inventory, §9 decision template), the app-category patterns in `rea.md`, and your restated requirements
- **Requirements (restated):** miniapps for business process automation in the style of RPA, invoked by AI agents inside harnesses (Codex, Claude Code, Hermes, OpenClaw, …); Go; DBOS; a Go-based pg-boss equivalent; Fly.io; Neon Postgres; integrations via SaaS APIs and custom REST APIs, compliant with SaaS authentication, authorization and permissions
- **Output:** review (§1–§5), pivoted vision and blueprint (§6–§8), re-sequenced roadmap (§9–§10), decisions needed (§11)

---

## 0. Verdict

**Do not start Phase 0 as written.**

plan4.1 has a strong core: honest at-least-once semantics, UNKNOWN-outcome quarantine, credentials kept away from agents, refusal to claim HA from one Machine, and spike-first gating. But it fails the test it adopts from ADR-001 — *the framework must earn its complexity* — at the platform level. It also contains four technical conflicts with how DBOS, River, and Neon actually behave, and one security gap squarely inside your "custom REST" requirement.

Summary of what to change:

1. **Cut the Sprite / promotion / lifecycle-CLI / CI-parity track out of the core.** It is roughly 40% of the plan, is not in your restated requirements, and drives most of the risk register (K14–K19). Keep a single manifest field (`placement: machine`) so the door stays open.
2. **Fix ADR-016 (shared host, static registration) for Class D.** DBOS recovery is gated by application version; one shared binary for all miniapps couples every miniapp's multi-day workflows to every other miniapp's releases.
3. **Close the SSRF/egress gap** in the capability proxy before any custom-REST connector exists.
4. **Re-sequence:** no external write ships before the effect ledger, reconciler and approval gate exist. Today `crm-write` (Phase 2) precedes both (Phase 4).
5. **Run seven timeboxed spikes (§9)** instead of a 13-condition G0, then build the vertical slice and deploy it to Fly in the first iteration.

### Top findings at a glance

| ID | Sev | Finding | Basis |
|---|---|---|---|
| F01 | Critical | Shared host + static registration (ADR-016) vs DBOS application-version recovery | V |
| F02 | Critical | Scope inflation: Sprites, promotion pipeline, lifecycle CLI, CI parity before any miniapp ships | P/J |
| F03 | Critical | No SSRF / egress controls for custom REST connectors | P |
| F04 | High | External writes (Phase 2) ship before reconciliation and approvals (Phase 4) | P |
| F05 | High | `confirm` trust mode turns *any* tool into a human-in-the-loop process; taxonomy never reconciled | P |
| F06 | High | Uniform async run handle for Class A contradicts ADR-001 Class A | P |
| F07 | High | Class C "progress model" is a shadow workflow engine; River OSS lacks global concurrency limits | V |
| F08 | High | Entity serialization unspecified; DBOS partitioned queues fill the Virtual-Object gap rea.md claimed | V |
| F09 | High | ADR-001 "admission gate" cannot be automated; no per-miniapp §9 decision record; no §8 inventory | P |
| F10 | High | No blast-radius limits, kill switch, or per-principal rate limits for ordinary miniapps | P |
| F11 | High | No treatment of untrusted tool results (prompt injection through returned data) | P/J |
| F12 | High | SaaS permission pass-through, re-auth channel, and out-of-band approver channel unspecified | P/J |
| F13 | High | Neon: disaster-recovery replay hazard, pooled-mode limits on RLS context, autosuspend vs polling, connection budget | V/J |
| F14 | High | Fly: overlapping deploys of the single DBOS executor, drain, static egress IP | V/J |
| F15 | High | Phase 0 overloaded; golden path verified last; "frozen" with unanswered gating questions | P |
| F16–F24 | Med/Low | Pins, determinism lint, manifest/build drift, circular verification, retention vs erasure, catalog shape, identity sprawl, invented requirements, SQLite dev backend | see §3.7 |

**Basis legend:** **V** = verified externally on 2026-11-22 (§12); **P** = from plan text; **J** = reviewer judgment.

---

## 1. Method and limits of this review

- Every miniapp and platform component was scored against ADR-001's questions: *Is this a genuine business process? Must it survive its initiator? Would doing it without the framework require more custom machinery? Does the net complexity go down?*
- rea.md was used as a **pattern source** (stateless service, delayed job, journaled fan-out, workflow, keyed entity, saga, guardrails), not as an engine choice. Restate is not on your stack.
- I checked DBOS Go, River, and Neon behavior against current documentation (list in §12). I did **not** verify Fly.io specifics, the current MCP authorization revision, harness-specific MCP behavior, or any SaaS provider's OAuth/ToS quirks. Those claims are marked "verify" wherever they appear.
- plan4.1 says every dependency version is UNPINNED because none were verified. That is no longer a reason to stay unpinned — see F16.

---

## 2. What plan4.1 gets right (keep these)

1. **Two independent dimensions** (execution semantics × placement) is a clean idea; only the second dimension needs cutting for now (F02).
2. **R2 acceptance language** — "no duplicate *confirmed* effects, no silently lost operations, no blind retries of unknown outcomes" — is the honest contract. Keep verbatim.
3. **Effect protocol** `PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN` with quarantine and reconciliation.
4. **Agent never holds provider credentials**; two-principal model (agent + authority).
5. **Honest availability posture** (K21): a single always-on executor is not HA.
6. **R9**: atomicity of app write + DBOS checkpoint depends on the pool handle, not on "sharing a URL". Verified correct — see F13/§6.5.
7. **Spike-first gating with failure dispositions** (§23.2): a failed spike forces a re-plan, never a workaround.
8. **R14**: no arbitrary expiry on run handles for multi-day waits.

---

## 3. Findings

### 3.1 Critical

#### F01 — ADR-016 (shared host, static registration) conflicts with DBOS application-version recovery
- **Basis:** V + P
- **Plan text:** ADR-016: one shared Go runtime, statically registered miniapp implementations, independently versioned manifests; ADR-008: single DBOS executor.
- **Evidence:** DBOS tags every workflow with the application version that started it and, by default, does not recover workflows tagged with a different version. The Go SDK exposes an application-version setting and patching APIs; automatic migration of old-version workflows to the new version is an Enterprise feature (`AutoUpgrade`).
- **Impact:** In a shared host, shipping `crm-read` v1.4 changes the host build. Depending on how the version is derived, every in-flight multi-day `lead-followup` or `employee-provisioning` workflow is either stranded (version mismatch) or replayed against changed code (non-determinism). "Independently versioned manifests" is fiction when the code that runs the workflows is one binary.
- **Fix (pick one, decide in spike S2):**
  1. **Recommended:** keep one shared **gateway host** for Class A/B/C, but run **one DBOS worker process group per Class D miniapp (or family)**, each with its own explicit application version, bumped only on breaking workflow changes. This pulls "independent binaries" forward *only* for Class D.
  2. One shared DBOS worker with a **manually pinned semantic `ApplicationVersion`**, strict use of `Patch`/`DeprecatePatch` for compatible changes, and a deployment guard that refuses a workflow-code change without a version bump or patch. Cheaper, but a human error becomes a production replay bug.
  - Either way, verify in S2 whether the Go SDK scopes queues, recovery, and versions per application, or whether you need separate `dbos` schemas/databases to isolate families.
  - Bind the manifest to the build (see F18).

#### F02 — Scope inflation: the platform is specified before a single miniapp exists
- **Basis:** P + J
- **Plan text:** Sprites (workspaces, bounded tasks, hybrid pattern), promotion records with OCI/SBOM/provenance, a lifecycle CLI with ~14 commands, GitHub Actions parity, 17 ADRs, R1–R23, A1–A16, 13 G0 conditions, 6 gates, 91 KB of text, zero code.
- **ADR-001 applied to the plan itself:** Gate 4 asks whether the framework reduces total complexity versus the simpler alternative. The plan never asks that of its own components. Sprites alone generate A1–A6, A9, K14–K19, §18's longest test list, and Appendix C.
- **Impact:** Your restated stack (Go, DBOS, River-equivalent, Fly, Neon, REST integrations) contains no Sprites. The plan's own Appendix D admits the Sprite requirements were reconstructed from a truncated file. You would be building the largest and riskiest subsystem on inferred requirements.
- **Fix:** disposition table in §4. Sprites, bounded tasks, promotion, and the full lifecycle CLI move to an **optional extension track** (§9, R6), gated on a real need for running untrusted agent-authored code. Keep `placement: machine` as the only legal manifest value in v1.

#### F03 — Custom REST connectors without SSRF and egress controls
- **Basis:** P
- **Plan text:** `capability/registry.go` "connector registry + allowlists"; "typed SaaS/custom-REST clients". Nothing names SSRF, DNS rebinding, redirect handling, or private-range blocking.
- **Impact:** "Custom apps via REST APIs" means a tenant-supplied base URL. Without controls, the capability proxy is an SSRF engine running inside your Fly private network (6PN/`.internal` addresses), next to your Neon endpoints, metadata endpoints, and your own admin ports.
- **Fix (mandatory before any custom-REST connector):**
  - https only; per-connection host allowlist registered by an operator, not by the agent.
  - Resolve DNS yourself, validate the resolved IP against deny ranges (loopback, link-local, RFC 1918, carrier-grade NAT, cloud metadata, Fly internal, Neon hosts), then connect to that IP (no re-resolution → no rebinding).
  - No cross-host redirects; redirect limit; re-validate on every hop.
  - Response size cap, timeout, content-type allowlist.
  - Egress through a dedicated proxy layer so one place enforces all of the above.
  - SSRF probe suite in CI (§7 matrix).

### 3.2 High — reliability and ADR-001 compliance

#### F04 — External writes ship before the machinery that makes them safe
- **Basis:** P
- **Plan text:** `crm-write` (Class A) in Phase 2; reconciliation workers, approvals, and saga in Phase 4.
- **Impact:** A Class A tool that PATCHes a SaaS is not "one Postgres transaction"; it is an external effect with unknown-outcome risk (timeout after the provider applied the change). Between Phase 2 and Phase 4 an UNKNOWN effect has no reconciler and no approval gate.
- **Fix:** the first external-write miniapp lands only after: the effect ledger, a reconcile-by-read sweeper (River periodic job), the approval table (F05), and write-authorization tests. Until then, write tools are `dry-run`-only.

#### F05 — `confirm` mode makes every tool a human-in-the-loop process
- **Basis:** P + ADR-001 §5D, Gate 4
- **Plan text:** trust modes (`dry-run`, `confirm`, `autonomous`) bound per miniapp version per tenant; `approvals/` package; Class A/B/C/D chosen by execution shape.
- **Problem:** a Class A `record-update` in `confirm` mode must wait for a human — ADR-001 calls that Class D territory ("human-in-the-loop processes"). The plan never says what holds the pending state for a Class A/B tool or what resumes it. If approvals live in app tables plus a poller, you are quietly building the "durable timers + recovery + reaper" machinery that ADR Gate 4 lists as the *cost of not using DBOS*.
- **Fix:** define **approval-gated invocation** as a framework invariant with one mechanism:
  - Approval is a single table with a state machine: `PENDING → APPROVED | REJECTED | EXPIRED`, `expires_at`, `material_hash` (invalidates on parameter change), approver identity, channel.
  - Bounded wait (e.g. ≤ 24 h) for single-step tools: approval triggers a one-shot River job (or inline tx). No DBOS.
  - Approval inside a larger multi-stage process: the workflow waits on a DBOS message/event with a timeout; the same approval table remains the source of truth and audit.
  - Record this as the one place where a Class A/B tool borrows a D-style wait without being reclassified.

#### F06 — Uniform async handle for Class A
- **Basis:** P
- **Plan text:** §6.3: *every* invocation returns a run handle immediately; agent polls.
- **ADR-001 Class A:** completes within one request, no independent execution lifecycle.
- **Impact:** a "read a contact" tool costs the agent extra turns, tokens, and latency; harness tool timeouts and model context make this worse.
- **Fix:** **sync-first contract.** `invoke` returns the result inline when the tool is Class A or completes within a sync budget (e.g. 5 s); it returns a run handle only for B/C/D or when the budget is exceeded. Every invocation still gets a run/audit row. Add `wait_ms` long-poll on `get_run` (cap ~20 s) to cut polling.

#### F07 — Class C "progress model" is a shadow workflow engine
- **Basis:** V + P
- **Plan text:** R1/§11.1: framework-owned progress record, per-record idempotency keys, bounded concurrency, resume-without-replay on River.
- **ADR-001:** Gate 4 requires comparing Alternative A (custom state/recovery) with Alternative B (DBOS). The plan asserts the answer for Class C instead of comparing.
- **New evidence:** River's open-source tier does not include global concurrency limits or workflows (Pro features, per River's published plan comparison — verify on riverqueue.com). For RPA against rate-limited SaaS APIs, global/per-tenant/per-provider concurrency control is core. DBOS queues provide worker concurrency, rate limits, deduplication, priority, and partitioned concurrency in the open-source library.
- **Impact:** you may build bespoke concurrency and progress machinery on River that DBOS queues already provide — while still running DBOS anyway for Class D, which makes DBOS's marginal complexity lower than ADR-001 assumed when it was written against pg-boss.
- **Fix:** keep River as the Class B engine. For Class C, run an ADR §9 **PILOT** on `bulk-import` implemented both ways (River + progress table vs DBOS parent workflow + queued child workflows) and record the decision with measured custom-code volume, crash-test results, and rate-limit behavior. Do not bake the answer into R1.

#### F08 — Entity serialization is unspecified; the rea.md "no DBOS equivalent" claim is out of date
- **Basis:** V + P
- **Plan text:** R12 rejects "entity miniapp" as a class (fine) but §15 only says "Postgres mechanisms for entity serialization" with no design.
- **Why it matters for RPA:** two agents (or one agent retrying) acting on the same invoice, lead, or employee is *the* correctness hazard.
- **Evidence:** the DBOS Go reference documents partitioned queues in which at most one workflow per partition key runs at a time. That is the keyed single-writer semantics rea.md attributed only to Restate Virtual Objects (execution serialization; entity *state* still lives in your tables).
- **Fix:** define a framework **Keyed Serial Executor** pattern:
  - Class A: `SELECT … FOR UPDATE` on the entity row (or advisory lock) inside the tx.
  - Class B/C: unique job key + entity row lock in the job.
  - Class D: workflow enqueued on a queue partitioned by `entity_key`, concurrency 1.
  - Manifest declares `entity_key` per tool; the framework refuses to register a write tool without it or an explicit `no_entity` justification.
  - Spike S3 verifies ordering/fairness within a partition, deduplication, and rate-limit interaction.

#### F09 — ADR-001 enforcement is hollow
- **Basis:** P
- **Plan text:** ADR-001 "adopt verbatim"; "executor admission check (ADR-001 gate)"; "rubric admission".
- **Problems:**
  1. ADR-001 is written for OpenLevel's TypeScript/Hono/pg-boss world. "Verbatim" is false; it needs a Go/River port (Hono → net/http, pg-boss → River).
  2. Gates 3 (business consequences) and 4 (net complexity) are judgments; code cannot "admit" them. The check degenerates into "manifest says class D".
  3. No per-miniapp §9 decision record (REJECT/DEFER/PILOT/ADOPT).
  4. No §8 workload inventory (one authoritative owner per responsibility) even though River and DBOS coexist *permanently* by design, whereas ADR §8 treats coexistence as a migration state.
- **Fix:** require `adr001.md` per Class C/D miniapp (template in §5.2). Admission verifies: record present, score arithmetic consistent, decision ∈ {PILOT, ADOPT}, §7.1–7.8 checklist items ticked with links to tests. Maintain the inventory table in §5.1. Review annually whether River still earns its seat.

#### F10 — No blast-radius controls
- **Basis:** P + J
- **Gap:** the plan has budgets for Sprite tasks but nothing equivalent for ordinary miniapps. An agent in a retry loop can send 10,000 emails or update 10,000 records within policy.
- **Fix — per principal × tenant × capability:** rate limits, daily/hourly write caps, max records per run, monetary caps where relevant, recipient/domain allowlists for outbound communication, a **circuit breaker** (auto-trip on error/duplicate spikes), and a **kill switch** at tenant, miniapp, and capability levels that takes effect without redeploy. Limits are declared in the manifest and enforced server-side.

#### F11 — Untrusted tool results (prompt injection through returned data)
- **Basis:** P + J
- **Gap:** server-side authorization stops an injected instruction from *escalating*, but injected text in a CRM note or email subject can still steer the agent into misusing permissions it legitimately has.
- **Fix:** result envelope returns typed fields; free-text provider content is wrapped as `untrusted: true` with size caps; never pass provider text through as instructions or error messages; high-risk capabilities require `confirm` regardless of what the agent claims; poisoned-fixture tests in CI (§7 matrix).

#### F12 — SaaS permission pass-through, re-auth, and approver channels
- **Basis:** P + J
- **Gaps:**
  1. **Permission intersection:** effective authority must be the intersection of manifest-approved scopes, the OAuth scopes actually granted, *and* what the SaaS-side identity can do. Provider 403 is authoritative; never cache privilege decisions beyond a short TTL.
  2. **Attribution:** manifest must declare whether actions appear as the delegated user or as a service identity — SaaS audit logs and compliance regimes care.
  3. **Re-auth:** agents cannot complete browser consent. Expired/revoked grants (`invalid_grant`) must surface as typed `needs_reauth` with an out-of-band link to a human, and the connection moves to `NEEDS_REAUTH` without crashing the miniapp.
  4. **Approver channel:** approvals must happen out of band from the agent session (separate authenticated UI, Slack/email magic link with step-up) with separation of duties. An agent that can approve its own request defeats `confirm`. Harness elicitation is a convenience, never the control.

### 3.3 High — infrastructure

#### F13 — Neon
- **Basis:** V (pooling facts) + J (DR, cost)
- **Findings:**
  1. **Pooled endpoint limits.** Neon's pooler runs PgBouncer in transaction mode; `SET`/`RESET` and `LISTEN`/`NOTIFY` do not work through it. plan4.1 R5 proposes an RLS "session variable". Mandate **transaction-local** context only (`set_config(..., true)` / `SET LOCAL` inside the tx) and make the tenant-context helper the only way to open a tenant tx.
  2. **DBOS on direct, River choice by spike.** Consistent with K5/ADR-007; keep. River offers a poll-only mode for environments without `LISTEN`; confirm in S4 which River mode runs on the pooled endpoint.
  3. **Connection budget** — direct connections are capped by compute size (documented examples: ~104 on 0.25 CU, ~419 on 1 CU). The plan says "explicit budgets" but gives no formula. See §8.2.
  4. **Autosuspend vs polling.** An always-on DBOS executor plus River polling means the Neon compute never idles. Decide up front: production autosuspend **off** and sized for always-on; dev/test branches may autosuspend. The plan has no Neon cost model.
  5. **PITR/restore replay hazard (not in the plan).** Restoring Neon to an earlier point rewinds app tables *and* DBOS checkpoints together, but SaaS-side effects are not rewound. After restore, in-flight workflows will re-execute steps whose effects already happened. Runbook: stop workers → restore → mark all non-terminal runs `UNKNOWN`/paused → reconcile effects by read → resume deliberately. Drill it in R5.

#### F14 — Fly
- **Basis:** V (fencing note in release notes) + J
- **Findings:**
  1. **Overlapping deploys.** A single DBOS Machine that is blue/green or surge-deployed can briefly run two executors that both attempt recovery. Release notes for the Go SDK list workflow ownership / fencing tokens as of v1.6.0 — verify that this closes the overlap window. Independently: pin `max 1` machine per DBOS app, use a stop-then-start (non-surge) deploy strategy for it, set kill signal/timeout to allow graceful `Shutdown`, and test restart-with-pending in S5.
  2. **Static egress IP.** Many enterprise SaaS tenants require IP allowlisting. Plan for a static egress IP on the machines that call providers (verify current Fly offering and cost).
  3. **Ingress.** The MCP/REST gateway is public; see §7 for the agent-authentication ladder.
  4. **Process groups.** `gateway`, `river`, `dbos-<family>`; same region as Neon (measure, don't assume — R10 stays).

### 3.4 High — process

#### F15 — Phase structure and "frozen" claims
- **Basis:** P
- **Problems:**
  1. **Phase 0 is Phases 1–5 in miniature**: Sprite round-trip spike, CLI freeze, CI/CD design, retire procedures, threat-model sign-off, 13 G0 conditions. It will not end.
  2. **Golden path verified last** (G6), so the thing that matters most to developer experience is the last thing learned.
  3. **"Frozen architecture baseline" with Q1–Q9 unanswered**, a missing security owner, no first SaaS, no confirmed Fly/Neon orgs, and a "no blockers for Phase 0" statement that contradicts G0 conditions 9 and 11.
  4. **Harness proof is a Phase 6 claim** while MCP/harness quirks are rated High risk (K6/K7). Prove one harness in the first spike.
- **Fix:** §9 — seven timeboxed spikes, vertical slice deployed to Fly in the first build iteration, golden path grown with the slice, "freeze" only what the slice proves.

### 3.5 Medium and Low

| ID | Sev | Finding | Fix |
|---|---|---|---|
| F16 | Med | Versions stay "UNPINNED" although current docs are available: DBOS Go is at v1.x (breaking v1.0 release; later minor releases add fencing, partitioned queues, datasources, SQLite). Enterprise features (Conductor, auto-upgrade) live in a **separate module under a different license** (DBOS Conductor Software License) — ADR §7.8 licensing check is now answerable. | Pin in the first spike; record the license of every module imported; forbid importing the enterprise module without an explicit decision. |
| F17 | Med | ADR §7.1 determinism is stated generically; Go has specific traps (goroutines in workflows, `select`, map iteration order, `time.Now`, `rand`, globals). DBOS Go provides durable `Go`/`Select` helpers; and docs forbid calling `RunAsTransaction`, `Enqueue`, `Send`/`Recv` inside step bodies. | Ship a `go/analysis` linter run in CI **strict** policy mode that flags these inside workflow functions; add replay tests. |
| F18 | Med | Static registration + "independently versioned manifests": manifest version and binary can drift. | Manifest carries `impl_digest` (build ID/commit) and the registry refuses a manifest whose digest does not match the running build. |
| F19 | Med | Appendix E marks all amendments "Verified: Yes" by re-reading the plan for internal consistency. That is self-consistency, not evidence. | Replace with a table of *evidence artifacts* (spike report, test name, commit). |
| F20 | Med | Retention is defined (A10) but erasure is not: DBOS checkpoints persist step inputs/outputs; audit is append-only. | Keep PII out of checkpoints (IDs only); store PII in app tables with a deletion path; document which records are retained vs erased and why; test erasure end to end. |
| F21 | Med | Catalog is API-wrapper-shaped (`crm-read`, `crm-write`, `notify-send`); no tool-granularity guidance; no result size/pagination rules. RPA value is in *business outcomes* ("post this invoice"). | §6.2 granularity rule; §5/§9 flagship catalog; result caps and cursors in the envelope. |
| F22 | Med | Four overlapping identity constructs: agent principal, authority principal, trust mode, proxy-issued task authority. | Without Sprites: two principals + trust mode only. |
| F23 | Low | Appendix D: requirements "inferred" from a truncated prompt; a typed REST client planned for Sprite APIs whose existence is unverified (violates the plan's own "do not code against imagined APIs"). | Moot if F02 is accepted. |
| F24 | Low | DBOS Go has a SQLite backend (v0.17+); River SQLite support is described as experimental. | Optional fast local loop for `dev`/unit tests; keep every crash/atomicity test on Postgres. |

---

## 4. Disposition of plan4.1 components

| Component | Verdict | Reason |
|---|---|---|
| ADR-001 adoption | **Keep, port** | Governing rule; write the Go/River edition and the per-miniapp record (F09) |
| ADR-002 one repo, one module | Keep | Fine |
| ADR-003 shared runtime default | **Modify** | Shared gateway for A/B/C; per-family worker group for D (F01) |
| ADR-004 canonical invocation service | Keep | Thin adapters over one path is right |
| ADR-005 two principals + trust modes | **Simplify** | Drop task authority until Sprites exist (F22) |
| ADR-006 honest at-least-once | Keep | |
| ADR-007 connection strategy | **Keep + fix** | Transaction-local RLS context; budget formula (F13) |
| ADR-008 single DBOS executor | **Keep + add** | Deploy-overlap and drain rules (F14) |
| ADR-009 manifest authority | Keep, slim | See §6.7; add `impl_digest` |
| ADR-010 agent-toolchain deferred | Keep | Harness owns the reasoning loop |
| ADR-011 two dimensions | **Simplify** | Dimension B = manifest field, one legal value (`machine`) |
| ADR-012/013/015 Sprite placement/SDK/hybrid | **Defer (extension track)** | F02 |
| ADR-014 lifecycle CLI (14 commands) | **Simplify** | `init`, `validate`, `test`, `dev`, `deploy`, `status`, `invoke` — add others only when used twice |
| ADR-016 binary/registry model | **Modify** | F01 |
| ADR-017 golden path + CI parity | **Simplify** | Golden path = slice deployed to Fly in R1; GitHub Actions calls the same `miniapp` commands, no extra service layer |
| Promotion records, OCI/SBOM/provenance | **Defer** | Plain container build + digest pinned in `fly deploy`; add SBOM when a customer asks |
| Sprite task registry, reaper, policy barrier | **Defer** | |
| Retention (A10), DBOS availability (A11), failure dispositions (A13) | Keep | |
| 13-condition G0 | **Replace** | Seven spikes (§9) |
| Catalog (12 miniapps) | **Replace** | §5.3 pattern-driven catalog |

---

## 5. ADR-001 re-applied

### 5.1 Workload inventory (ADR §8) — one authoritative owner each

| Workload pattern | Execution owner | Reason |
|---|---|---|
| Sync read / validated command | Go handler + Postgres tx | Class A; no engine |
| Approval-gated single-step command | Approval table + one-shot River job | Bounded wait; F05 |
| One-shot job, webhook processing, scheduled report | River | Class B |
| Bounded batch with per-record effects | **PILOT: River vs DBOS queue** | Class C; F07 |
| Multi-day, waits, approvals inside process, saga | DBOS | Class D |
| Per-entity serialization | Row lock (A/B/C); partitioned DBOS queue (D) | F08 |
| Effect reconciliation sweeper | River periodic job | Not a workflow |

### 5.2 Per-miniapp decision record (`adr001.md`, required for Class C/D)

```markdown
# ADR-001 decision — <miniapp id> v<version>
- Business promise: <what must never break>
- Score (0/1/2 each): lifetime _, stages _, waiting _, recovery _, effects _, coordination _, human _  = __/14
- Gate 1 durable need: <which>   Gate 2 (≥2 characteristics): <list>   Gate 3 failure consequence: <text>
- Alternative A (without DBOS) custom machinery list: <state, timers, reaper, ...>
- Alternative B (DBOS) what it covers / what we still build: <list>
- §7 checks: determinism lint ✔ | idempotency keys ✔ | tx boundary ✔ | workflow ID ✔ | versioning plan ✔ | payload discipline ✔ | recovery policy ✔ | op-cost ✔  (link each to a test)
- Decision: REJECT | DEFER | PILOT | ADOPT   Date / owner:
```

### 5.3 Pattern-driven catalog with ADR scores

Scores are my estimates for illustration; the owner records real ones. Dimensions: lifetime, stages, waiting, recovery, effects, coordination, human.

| Miniapp (business verb) | Pattern (rea.md → Go) | Score | ADR class → substrate |
|---|---|---|---|
| `record-lookup` | Stateless service (`ServiceClient` in rea.md) | 0 | A → handler |
| `record-update` (+ `confirm`) | Command + approval-gated invocation | 1 → ~5 with approval | A (+ approval table) → handler / River |
| `notify-send` | Send with idempotency key (+ delay) | 2 | B → River |
| `report-run` | Scheduled job | 4 | B → River periodic |
| `bulk-import` | Journaled batch with fan-out/fan-in | 6–7 | C → **PILOT** River vs DBOS |
| `system-sync` | Pull-merge-push with keyed serialization + conflict policy | ~8 | C/D → architectural comparison required |
| `follow-up-sequence` | Durable workflow, waits, cancel | 11–12 | D → DBOS |
| `invoice-to-ledger` | Workflow + approval + saga + reconcile | ~13 | D → DBOS |
| `employee-provisioning` | Saga across SaaS + manager approval wait | 12–13 | D → DBOS |

---

## 6. Pivoted vision and architecture

### 6.1 Vision

> **Miniapps are agent-invokable business-process automations — RPA for agents, API-first.**
> An agent in any harness requests a *business outcome* ("post this invoice", "onboard this employee", "follow up with these leads"). The platform executes it as a durable, permissioned, audited process against SaaS and custom REST APIs. The agent supplies intent and parameters; the platform owns credentials, authorization, execution, recovery, approvals, and audit.

Explicit non-goals (unchanged): UI/screen-scraping automation (a later, separately secured capability if ever), a general agent framework, a new workflow engine, arbitrary agent-generated code in the trusted runtime.

### 6.2 What a miniapp is — and the granularity rule

A miniapp is a **versioned, typed, permissioned business verb with declared execution semantics**.

- **Granularity rule:** a tool should map to one business outcome with a stable idempotency identity (`tenant:entity:kind:op`), not to one provider endpoint. If an agent must chain three tools to achieve one outcome and a crash between them leaves a mess, those three belong inside one miniapp (usually Class D).
- **Thin wrappers are not miniapps.** If the agent needs raw API access, expose the capability through a read-only, policy-bounded `capability.call` tool, not through a catalog entry.

### 6.3 Principles

1. Simplest mechanism that satisfies the requirement (ADR-001 §10). Every component and every miniapp has a recorded reason to exist.
2. Honest delivery semantics: at-least-once steps, idempotency keys, UNKNOWN quarantine, reconcile-by-read.
3. The agent is untrusted for authority and for content: it never holds credentials, never approves its own actions, and never receives unmarked provider text as instructions.
4. Everything consequential is bounded: rate, volume, blast radius, kill switch.
5. One invocation path (REST canonical; MCP and CLI are adapters) with one audit trail.
6. Prove with crash tests on real Postgres before claiming a guarantee.

### 6.4 Architecture (core v1)

```text
 Agent harnesses (Codex, Claude Code, Hermes, OpenClaw, ...)
        |  MCP (Streamable HTTP)  |  REST/OpenAPI  |  CLI (invoke)
        v
+--------------------------------------------------------------+
| gateway  (Fly Machines, 1-2)                                 |
|  authN (agent keys/OAuth) -> authZ (policy, trust mode,      |
|  limits, kill switch) -> registry -> dispatch -> audit       |
|  Class A handlers (sync, tx + effect ledger)                 |
|  approvals API, webhook receiver, capability proxy,          |
|  token broker  (SSRF-hardened egress)                        |
+------+-------------------+-------------------+---------------+
       | InsertTx          | start/enqueue     | provider calls
       v                   v                   v
+-------------+   +------------------+   +-------------------------+
| river       |   | dbos-<family>    |   | SaaS APIs / custom REST |
| (B, C, sweep|   | (D; one process  |   | (OAuth2, API keys, HMAC)|
|  reconcile) |   | group per family)|   +-------------------------+
+------+------+   +---------+--------+
       \                    /
        v                  v
 +--------------------------------------------------+
 | Neon Postgres (same region as Fly)               |
 |  app schema | river schema | dbos schema(s)      |
 |  RLS defense-in-depth (txn-local tenant context) |
 +--------------------------------------------------+
```

### 6.5 Stack decisions

| Concern | Decision | Constraint / note |
|---|---|---|
| Language | Go | Minimum version = max of pinned SDKs |
| Durable workflows (Class D) | DBOS Transact Go | Pin v1.x; open-source only; enterprise module forbidden until decided |
| Jobs (Class B, maybe C) | River (Go, Postgres-backed; the pg-boss equivalent) | OSS tier; know the Pro-only limits (F07) |
| Atomic app write + checkpoint | `RunAsTransaction` with a datasource **built on the same pool handle DBOS uses as its system DB** | Then no separate completion table; spike S1 proves it |
| Atomic enqueue | River `InsertTx` in the domain tx | Spike S4 |
| Compute | Fly Machines | Process groups: `gateway`, `river`, `dbos-<family>` |
| Database | Neon, one primary, schemas `app`/`river`/`dbos` | DBOS on direct endpoint; RLS context txn-local |
| HTTP/MCP | `net/http` + official Go MCP SDK | Pin and verify transports/auth |
| Config/policy | env-driven `strict`/`warn`/`disabled` (rea.md guardrail) | CI = strict |

### 6.6 App-category patterns (rea.md → ADR-001 → Go)

| Pattern | rea.md source | ADR class | Substrate | Required guardrails |
|---|---|---|---|---|
| Query | Stateless service | A | Handler | Authz before dispatch; result caps; untrusted-field marking |
| Command | Stateless service + `ValidateInput`/terminal error | A | Handler + tx + effect ledger | Idempotency key; entity lock; blast-radius limits |
| Approval-gated command | Awakeable/promise + timeout race | A/B (borrowed wait) | Approval table + River | Out-of-band approver; material-change invalidation; expiry |
| Delayed / one-shot job | `Send` + delay + idempotency key | B | River | Unique key; retry policy; terminal-failure inspection |
| Batch | Journaled side effects + fan-out/fan-in | C | River **or** DBOS queue (pilot) | Per-record idempotency; bounded concurrency; resume without replay |
| Durable workflow | Workflow + timers + status/signal | D | DBOS | Stable workflow ID; version strategy; recovery policy; alerting |
| Keyed entity serialization | Virtual Object | A–D | Row lock / DBOS partitioned queue | `entity_key` required on write tools |
| Saga | Pre-registered compensation, LIFO, DLQ | D | DBOS + effect ledger | Register compensation **before** the action; idempotent compensators; never auto-compensate UNKNOWN without a safety basis |
| Sync / reconcile | Fan-out + compare | C/D | Queue/workflow | Reconcile-by-read; cursor checkpoints; conflict policy |
| Policy guardrails | Policy modes, idempotency-key validation, control/data-plane split | cross-cutting | Framework | Strict in CI/prod |

### 6.7 Miniapp contract

`miniapp.json` — slim v1 sketch (illustrative):

```json
{
  "id": "invoice-to-ledger",
  "version": "1.0.0",
  "impl_digest": "build:7f3c9a2",
  "placement": "machine",
  "adr001": { "class": "D", "score": 13, "decision": "ADOPT", "record": "adr001.md" },
  "tools": [
    {
      "name": "post_invoice",
      "kind": "async",
      "input_schema": "schemas/post_invoice.in.json",
      "output_schema": "schemas/post_invoice.out.json",
      "idempotency_key": "tenant:invoice_id",
      "entity_key": "invoice_id",
      "risk": "write",
      "trust_modes": ["dry-run", "confirm"],
      "limits": { "max_runs_per_hour": 60, "max_amount": 10000, "max_records_per_run": 1 },
      "approval": { "required_in": ["confirm"], "expires_in": "24h", "separation_of_duties": true }
    }
  ],
  "capabilities": [
    { "ref": "accounting.bill.create", "scopes": ["bills.write"], "attribution": "service" },
    { "ref": "accounting.bill.get",    "scopes": ["bills.read"],  "attribution": "service" }
  ],
  "fixtures": "fixtures/",
  "compat": { "workflow_version": "1" }
}
```

Go surface — framework shapes (illustrative; the DBOS calls follow current Go docs, verify against the pinned version):

```go
// Framework-owned contracts. Miniapps import only these packages.
type SyncHandler func(ctx context.Context, inv Invocation) (Result, error)          // Class A
type JobHandler  func(ctx context.Context, job JobContext) error                     // Class B/C (River)
type Workflow    func(ctx dbos.Context, in WorkflowInput) (WorkflowOutput, error)    // Class D (DBOS)

// External effects always go through the ledger + capability proxy.
res, err := cap.Call(ctx, "accounting.bill.create", req, effects.Intent{
    Key: inv.IdempotencyKey("bill.create"), // stable business identity
})

// Class D skeleton (shape only):
func PostInvoice(ctx dbos.Context, in PostInvoiceIn) (PostInvoiceOut, error) {
    parsed, err := dbos.RunAsStep(ctx, func(c context.Context) (Parsed, error) { return extract(c, in) },
        dbos.WithStepName("extract"))
    if err != nil { return PostInvoiceOut{}, err }
    // approval: wait on a DBOS message with timeout; approval table stays source of truth
    // post:     dbos.RunAsStep(... capability proxy, effect intent recorded first ...)
    // compensate: registered BEFORE the action, run LIFO on failure, idempotent
    // NO goroutines, NO map iteration, NO time.Now/rand inside workflow body (lint-enforced)
    return finalize(ctx, parsed)
}
```

### 6.8 Invocation contract (agent-facing)

| Item | Contract |
|---|---|
| `invoke(tool, input, idempotency_key?, wait_ms?)` | Returns `{run_id, status, result?, next_actions?}`; **inline result** for Class A or when done within the sync budget |
| `get_run(run_id, wait_ms≤20s)` | Long-poll to reduce polling; stable `run_id`, no active-handle expiry |
| `signal`, `cancel` | Authorized per tool; audited |
| Result data | Typed fields; free text wrapped `{value, untrusted:true}`; size caps; cursors for lists |
| Error taxonomy | `invalid_input`, `unauthorized`, `forbidden`, `needs_reauth` (human action), `needs_approval` (approval id + channel), `rate_limited` (`retry_after`), `limit_exceeded`, `in_flight_duplicate`, `unknown_outcome` (quarantined; do not retry), `retryable`, `terminal` |
| Discovery | `list_tools`/`describe_tool` + OpenAPI + an `AGENTS.md`/skill snippet harnesses can drop into context; generic MCP tools first, namespaced tools after interop is proven (R8 stands) |

---

## 7. Security, authentication and authorization blueprint

### 7.1 Two boundaries

1. **Agent → platform** (who may call which tool, in which trust mode).
2. **Platform → SaaS / custom REST** (what the platform may do with a connected account).

Each boundary has its own authentication, authorization, limits, and audit.

### 7.2 Agent → platform authentication ladder (verify against the current MCP authorization revision)

1. **OAuth 2.1-style flow per MCP authorization** for harnesses that support it (platform acts as a protected resource; scoped, short-lived access tokens).
2. **Scoped API keys** for harnesses without OAuth: hashed at rest, per agent principal, per tenant, per miniapp/tool scope, expiry + rotation, last-used tracking.
3. **Local stdio proxy** for coding-agent harnesses that prefer it, using the same keys/tokens.

Server-side authorization runs for every invocation regardless of what the harness approves in its own UI.

### 7.3 Platform → SaaS

| Concern | Rule |
|---|---|
| Grant type | OAuth 2.0 authorization code + PKCE for delegated access; client credentials / service accounts where the provider supports them; API keys only where nothing else exists |
| Scope policy | Request the least scope per capability; incremental consent; manifest scopes ⊆ granted scopes ∩ SaaS-side permissions (F12) |
| Token handling | Envelope encryption with a KMS key; refresh rotation with reuse detection; revoke on disconnect; tokens never enter workflow payloads, logs, run results, or agent-visible errors |
| Re-auth | `invalid_grant`/revocation → connection `NEEDS_REAUTH`, typed `needs_reauth` to agent, human link out of band |
| Attribution | Manifest declares `user` or `service`; documented per provider |
| Rate limits | Provider policy file per connector; honor `Retry-After`; enforce through queue rate limits or limiter |
| Idempotency | Provider idempotency keys where available; otherwise reconcile-by-read before any retry of an UNKNOWN effect |
| Webhooks | Signature verification, timestamp tolerance, replay dedupe, Class B processing |
| Provider terms | Record per-connector notes on automation/AI-use terms, bulk limits, marketplace/app-verification requirements and lead time (affects schedule) |
| Egress | SSRF-hardened proxy (F03); static egress IP where tenants require allowlisting |

### 7.4 Threat → control → proof

| Threat | Control | Proof |
|---|---|---|
| Agent over-permission / tool sprawl | Per-tool scopes, per-tenant allowlists, trust modes | Authz probes: every tool × principal × mode |
| Confused deputy | Authority principal bound at invocation; approvals for writes | Tests where the agent supplies a forged authority |
| Prompt injection via results | Typed results, `untrusted` marking, caps, forced `confirm` for high-risk | Poisoned-fixture suite |
| SSRF via custom REST | §F03 controls | Probe suite: private IPs, rebinding, redirects, metadata, `.internal` |
| Credential theft/leak | Envelope encryption, no secrets in payloads/logs | Grep-style scan of checkpoints, logs, run records |
| Runaway agent | Rate/volume caps, circuit breaker, kill switch | Load test + kill-switch drill |
| Duplicate/lost effects | Effect ledger, idempotency keys, UNKNOWN quarantine | Crash-boundary tests on real Postgres |
| Approval bypass | Out-of-band approver, separation of duties, material-change invalidation | Self-approval and replay tests |
| Cross-tenant access | App-level authz + txn-local RLS context + per-tenant keys | Cross-tenant probes at app **and** RLS layers |
| Webhook spoofing/replay | Signature + timestamp + dedupe | Replay tests |
| Audit tampering / PII | Append-only audit role (no UPDATE/DELETE), redaction, retention/erasure | Permission tests + erasure test |
| Supply chain | Pinned deps, vulnerability scan in CI | CI gate |

---

## 8. Fly.io and Neon operating model

### 8.1 Process groups

| Group | Runs | Scaling | Connection mode |
|---|---|---|---|
| `gateway` | REST/MCP, Class A, approvals, webhooks, proxy, token broker | 1–2 Machines, always on | Pooled for short txs; direct if a feature needs session state |
| `river` | Class B (and C), reconciliation sweeper | 1+ Machines; scale on queue depth | Direct, or pooled with poll-only (S4 decides) |
| `dbos-<family>` | Class D workers | **Exactly 1 Machine per DBOS app**, always on | Direct; same pool handle as the datasource |

### 8.2 Connection budget (fill in real numbers in S1/S4)

```text
direct_connections_needed =
    Σ(gateway_machines × gateway_pool)      # if using direct
  + Σ(river_machines   × river_pool)
  + Σ(dbos_apps        × dbos_pool)         # shared-handle datasource keeps this low
  + ops/migrations (≥ 3) + headroom (≥ 20%)
must be ≤ Neon max_connections for the chosen compute size (documented examples: ~104 at 0.25 CU, ~419 at 1 CU)
```

### 8.3 Deploy and recovery rules

- DBOS groups: `max 1` Machine, non-surge deploy, graceful `Shutdown` window, health check, executor-down alert, stuck-workflow alert with named owner.
- Workflow-code change policy (F01): version bump or patch required; deployment guard refuses otherwise; drain or run old version until pending workflows of the old version complete.
- Neon: production autosuspend off; dev/test branches disposable; PITR **restore runbook with effect reconciliation** (F13.5), drilled before GA.
- Secrets: platform secrets in Fly secrets; tenant credentials only in the encrypted connection registry.
- Observability: structured audit events, OpenTelemetry traces, queue depth/age, workflow states, effect states (UNKNOWN count is a paging alert), approval backlog.

---

## 9. Re-sequenced roadmap

Each phase is an independent coding-agent task. **Gates require evidence artifacts, not re-reads.**

### R0 — Seven timeboxed spikes (days, not weeks)

| Spike | Question | Pass condition | Failure disposition |
|---|---|---|---|
| S1 | DBOS `RunAsTransaction` on Neon direct with a shared pool: app write and checkpoint atomic? | Kill -9 between write and checkpoint at every boundary: exactly-once SQL effect | Redesign persistence boundary; no atomicity claims |
| S2 | DBOS versioning: deploy v2 while a v1 workflow sleeps; compare shared-worker + pinned version + `Patch` vs per-family worker; check schema/app isolation | Old workflow completes correctly under chosen topology | Choose per-family workers; revisit ADR-016 |
| S3 | DBOS partitioned queue as keyed single-writer + dedup + rate limit | Concurrent same-key requests serialize; no duplicates; rate limit holds | Fall back to row-lock pattern |
| S4 | River: `InsertTx` atomic with domain write; mode on pooled endpoint; measure missing global limits; compare vs DBOS queue for batch | Atomic enqueue proven; mode decided; pilot data captured | Use direct connection for River |
| S5 | Fly: single DBOS Machine restart/deploy with pending workflows; graceful shutdown; overlap behavior | Recovery within target; no duplicate step execution beyond declared at-least-once | Adjust deploy strategy; re-plan topology |
| S6 | MCP round trip on **two** harnesses (one coding agent, one general-purpose): auth, sync result, run handle, long-poll | Both complete `invoke` + `get_run` with auth | Narrow compatibility claim; adapt transport |
| S7 | One SaaS sandbox + one custom REST: OAuth code+PKCE, refresh rotation, revoke → `needs_reauth`; SSRF probe suite against the proxy | All probes pass | Block custom REST until fixed |

**Exit:** all spikes pass or have a recorded disposition; pins recorded with license per module; ADR-001 Go edition written; security owner named (can be you, in writing).

### R1 — Vertical slice, deployed

`record-lookup` (A) + `follow-up-sequence` (D) through REST + MCP + CLI invoke; agent auth; audit; determinism linter; mock SaaS; **deployed to Fly with restart-with-pending test** (first golden-path evidence).

**Gate:** worker-kill recovery with no duplicate *confirmed* effects; both harnesses call both tools.

### R2 — Real integrations and security

Token broker, OAuth, capability proxy with SSRF controls, custom REST adapter, policy/limits/kill switch, webhooks, RLS with txn-local context, `needs_reauth` flow.

**Gate:** §7.4 probes green on a real SaaS sandbox and a custom REST target.

### R3 — Effects and approvals

Effect ledger, reconcile-by-read sweeper, approval table + out-of-band channel, saga support, first write miniapp `record-update` in `confirm` mode.

**Gate:** UNKNOWN injection quarantines; approvals survive restarts; self-approval blocked.

### R4 — Jobs and batch

`notify-send`, `report-run`, `bulk-import` pilot with recorded ADR decision.

**Gate:** batch resumes without replaying completed records; decision record complete.

### R5 — Flagship RPA workflows and operations

`invoice-to-ledger`, `employee-provisioning`, `system-sync`; PITR restore drill; load and blast-radius tests; harness matrix.

**Gate:** all R2–R4 probes still green on production-like Fly + Neon; runbooks owned.

### R6 — Optional extension track (only on demonstrated need)

Sprites for untrusted agent-authored code, promotion pipeline with SBOM/provenance, richer lifecycle CLI, CI parity. Each item needs its own ADR-001-style justification.

---

## 10. Definition of done for a robust miniapp

- [ ] Manifest validates; schemas generated from Go types; drift test green
- [ ] `adr001.md` present (C/D) with real score and decision; §7 checklist linked to tests
- [ ] Idempotency identity defined; entity key declared (or justified absent)
- [ ] Every external effect goes through the ledger and capability proxy; UNKNOWN path tested
- [ ] Fixtures with independent expected results; mocks for each provider
- [ ] Crash-boundary tests at every workflow/job transition on real Postgres
- [ ] Determinism lint clean (D); replay test passes
- [ ] Authorization probes pass for every principal × mode
- [ ] Limits, circuit breaker and kill switch configured and tested
- [ ] Untrusted fields marked; result caps and cursors present
- [ ] Approvals (if any): out-of-band, expiring, material-change invalidation
- [ ] Retention/erasure behavior documented and tested
- [ ] Runbook: stuck, quarantined, re-auth, rollback; alerts wired
- [ ] Compatible-version strategy recorded (D)
- [ ] `impl_digest` matches the deployed build

---

## 11. Decisions needed from you

1. **Sprite/extension track:** confirm it is out of the core (recommended), or state the concrete untrusted-code use case that justifies it.
2. **First SaaS** with OAuth2 and a sandbox (idempotency keys and status lookup strongly preferred) — blocks S7/R2.
3. **Two harnesses** for S6 (suggest one coding agent and one general-purpose runtime).
4. **Tenancy for v1:** one tenant per deployment (simpler, strongest isolation, Neon branch/project per customer) vs multi-tenant with RLS from day one. Recommendation: put `tenant_id` and the tenant-context helper everywhere now, ship single-tenant, and turn on multi-tenant when a second tenant appears.
5. **DBOS edition:** open-source only (recommended) or is the enterprise module acceptable if sleep suspension/recovery proves necessary?
6. **Security owner** accountable for the threat model and for capability/approval-boundary changes (can be you, recorded).
7. **Static egress IP** budget if target SaaS tenants require IP allowlisting.
8. **First flagship process:** which real business process (invoice-to-ledger, employee provisioning, lead follow-up, …) should R5 target?

---

## 12. Sources checked (2026-11-22)

Verified in this review (paraphrased, not quoted):

- DBOS Go reference — workflows & steps, queues incl. partitioned concurrency, enterprise auto-upgrade: https://docs.dbos.dev/golang/reference/workflows-steps
- DBOS Go transactions and datasources (`RunAsTransaction`, shared-pool behavior): https://docs.dbos.dev/golang/tutorials/transaction-tutorial and https://docs.dbos.dev/golang/reference/datasources
- DBOS Go steps (at-least-once; calls forbidden inside steps): https://docs.dbos.dev/golang/tutorials/step-tutorial
- DBOS Go determinism rules (goroutines, `select`, map iteration): https://docs.dbos.dev/golang/prompting
- DBOS Go configuration (application version, patching, enterprise module and license, Conductor): https://docs.dbos.dev/golang/reference/configuration
- DBOS Go upgrade notes (v1.0 breaking release; fencing/ownership; Conductor split): https://docs.dbos.dev/golang/upgrading and https://github.com/dbos-inc/dbos-transact-golang/releases/tag/v1.0.0
- DBOS workflow recovery and application versions: https://docs.dbos.dev/production/self-hosting/workflow-recovery
- DBOS June 2026 update (Go step factories, SQLite backend): https://dbos.dev/blog/new-in-dbos-june-2026
- Neon pooling (transaction-mode limits; direct vs pooled; max connections): https://neon.com/docs/connect/connection-pooling and https://neon.com/docs/connect/choose-connection
- River OSS vs Pro features (third-party summary of riverqueue.com plans — **re-verify on riverqueue.com**): https://www.techbloat.com/software/river/ ; issue showing Pro global limits: https://github.com/riverqueue/river/issues/1145

**Not verified here (verify in the spikes):** Fly.io deploy strategies, kill/timeouts, static egress IP offering and pricing; the current MCP authorization specification revision; harness-specific MCP/OAuth behavior; provider-specific OAuth, verification, and API-terms constraints; DBOS Go default version derivation and per-application scoping of queues/recovery; River poll-only behavior on Neon's pooled endpoint.
