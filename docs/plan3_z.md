# The Miniapps Pivot

## Agent-Invoked Business Process Automation on Go, DBOS, River, Fly.io, and Neon

*A technical vision for engineers, architects, and business decision-makers — grounded in ADR-001 (OpenLevel's DBOS adoption policy), the Review 12 reliability retrospective, and the app category patterns from rea.md.*

---

## 1. Executive Summary

OpenLevel pivots from building background automation as private application plumbing to shipping it as a product surface: **miniapps** — small, robust, RPA-style business process automations that AI agents can discover, invoke, run, and monitor. A coding agent in a harness like Claude Code or Codex, or a general-purpose agent runtime like Hermes or OpenClaw, calls a miniapp the way it calls any tool: with a typed request, an explicit permission scope, and a contract that promises durable, correct execution even across crashes, deployments, and multi-day waits.

The stack is fixed by this decision:

| Layer | Choice | Role |
| --- | --- | --- |
| Language | **Go** | One static binary per miniapp; typed contracts; fast cold starts on Fly Machines |
| Durable execution | **DBOS Transact Go** (`dbos-transact-golang`) | Class D workflows: checkpoints, durable waits, recovery, queues — on Postgres |
| Job queue | **River** (`riverqueue/river`) | The pg-boss equivalent in Go: transactional Postgres-backed jobs for Class B/C |
| Runtime | **Fly.io Machines** | Per-miniapp deployment, auto-stop/start workers, secrets, private networking |
| Database | **Neon Postgres** | App data + DBOS system tables + River jobs; branching for environments |

This is not a new framework. It is the existing OpenLevel architecture — Hono-era lessons, ADR-001's class model, the Review 12 crash-boundary discipline, and the adoption rubric — repackaged so each business process becomes an independently deployable, agent-addressable unit with a governed security perimeter around SaaS and custom REST integrations.

Everything ADR-001 already decided still applies. The pivot changes **who invokes the execution layer** (agents, not just HTTP handlers) and **what the unit of delivery is** (a catalog of miniapps, not one app's background jobs).

---

## 2. Why Pivot — and Why Now

### 2.1 The reliability work already built the product

The last two development cycles produced, as internal plumbing: workflow state machines, reapers, recovery claims, idempotency primitives, transactional outboxes, and an external-effect protocol — precisely the machinery Review 12 showed is hard to build correctly by hand (ten of eleven remediation findings only partially resolved). That machinery is the core of a durable automation platform. The pivot turns a cost center into the product: instead of every feature re-implementing durability inside one SaaS app, each process ships once as a miniapp with proven guarantees.

### 2.2 Agents are the new integration surface

RPA traditionally meant fragile screen-scraping robots. The modern equivalent is an AI agent that can read APIs, plan, and call tools. What agents lack is exactly what miniapps provide:

- **Durability** — an agent session dies; a multi-day follow-up sequence must not.
- **Correctness under retries** — agents retry and re-plan; consequential side effects need idempotency, not duplicated invoices.
- **Governance** — agents cannot be granted standing admin credentials to a CRM; they need scoped, audited, least-privilege execution.
- **Observability** — "the agent said it did it" is not an audit trail.

So the RPA-style value (eliminating manual multi-system busywork) is delivered not by a brittle bot but by a **durable, permissioned workflow the agent merely triggers and monitors**. The agent does planning and judgment; the miniapp does execution with guarantees.

### 2.3 The category patterns already exist

rea.md maps ADR-001's operation classes onto concrete framework patterns. That mapping — Class A stateless operations, Class B delayed one-shot jobs, Class C journaled multi-step operations, Class D durable workflows with submit/attach/signal, per-entity Virtual-Object-style state, saga compensation, and policy-mode guardrails — transfers cleanly onto the required Go stack. Section 4 performs that translation pattern by pattern.

---

## 3. What Exactly Is a Miniapp

A miniapp is a **deployable automation unit with a contract**. Not a script, not a library function — a governed product component. Every miniapp consists of six parts:

1. **Contract** — a typed input/output schema (JSON Schema generated from Go structs), a human-readable description, worked examples, and an explicit **error taxonomy** (terminal errors vs. retryable errors vs. quarantined outcomes). Agents depend on this to call the miniapp correctly; CI validates that the published schema matches the binary.
2. **Workflow** — the durable execution logic: DBOS workflow for Class D, River job for Class B/C, plain transactional handler for Class A. Deterministic workflow code; all nondeterminism (clocks, randomness, external calls, DB reads) behind steps.
3. **Permission manifest** — a declarative, versioned statement of what the miniapp may do: which SaaS connections and scopes it may use, which tenants it may act on, which trust modes it supports (dry-run, confirm, autonomous), and budget/rate limits. No manifest, no deployment.
4. **Integration bindings** — named connectors (e.g., `crm`, `mail`, `calendar`) resolved through the token broker at step time. Miniapps never hold credentials (ADR-001 §7.6 payload discipline).
5. **Observability** — structured events emitted for every stage transition, external effect, and wait; execution history queryable by tenant, entity, and status (`waiting`, `stuck`, `completed`); OpenTelemetry traces from invocation to effect.
6. **Acceptance tests** — the crash-boundary script (Section 8.7) as an automated suite: kill the worker mid-wait, replace the machine, double-submit the trigger, and prove zero stranded runs and zero duplicate effects before the miniapp version ships.

The RPA contrast is worth stating plainly. Classic RPA automates the **interface** and inherits its fragility; a miniapp automates the **process** and inherits the database's guarantees. When the CRM changes its UI, a screen-scraping bot breaks silently; when the CRM changes its API, a miniapp fails loudly at a typed boundary, quarantines the unknown outcome, and pages an operator — per ADR-001 §7.2.

---

## 4. App Category Patterns → Miniapp Templates on the Go Stack

rea.md's answer section maps ADR-001's classes onto a Restate/Go idiom. The required stack replaces Restate with **DBOS Transact Go + River**, but every pattern survives the translation. This is the load-bearing table of the pivot:

| rea.md pattern | Restate/Rea form | Miniapp form on the required stack |
| --- | --- | --- |
| Class A — ordinary operation | Stateless service; `ValidateInput` + `TerminalError` | Plain Go HTTP handler + Neon transaction via `pgx`; no engine. Validation errors as typed terminal responses |
| Class B — simple background job | `Send` with `WithDelay` + `WithIdempotencyKey` | **River** job: one-shot, scheduled, unique jobs; enqueue **in the same Postgres transaction** as the business write (the pg-boss philosophy, in Go) |
| Class C — multi-step background op | Handler with journaled `Run` side effects, fan-out/fan-in | River job with explicit batch steps + per-record idempotency; promote to a DBOS workflow when the adoption rubric justifies checkpoints |
| Class D — durable workflow | Workflow service: `Submit/Attach/Signal`, durable sleeps, promises | **DBOS Transact Go** workflow: durable steps, `DBOS.Sleep`-style durable waits, events for external signals, workflow queues for concurrency limits, workflow ID as the identity |
| Per-entity state (Virtual Object) | Key-addressable object, single-writer semantics | **Per-entity workflow identity** — `workflow_id = tenant:entity:kind` gives one logical execution per entity; single-writer enforced with workflow identity deduplication plus per-entity River queues or a Postgres advisory lock in the claim step |
| Saga compensation | Pre-registered compensations, LIFO rollback, DLQ | A small platform helper library: compensation registered **before** the action executes, LIFO unwind on error, failures land in the workflow's terminal-failure inspection (the DLQ equivalent), compensations themselves idempotent steps |
| Guardrails / policy modes | `PolicyStrict / Warn / Disabled`; idempotency-key validation; context-usage validation | The same three policy modes as env-driven platform config: **strict** in CI/production (fail on missing idempotency keys, nondeterministic workflow code, missing manifest entries), **warn** in development, **disabled** only in tests. A static check on workflow functions enforces step-wrapping discipline |
| Observability | Metrics collector, OTel tracing, hooks | OTel traces from invocation → workflow → step → HTTP effect; counters for invocations, waits, effects, quarantines per miniapp version |

### 4.1 Class A — ordinary operations (no engine)

Retrieve a contact, check availability, update a setting. These stay ordinary Go handlers over Neon transactions. A miniapp of this class exists mainly to give agents a **safe, typed, permission-checked** way to do CRUD — the automation value is the governance shell, not the execution engine. ADR-001 is explicit: wrapping a plain transaction in a workflow engine adds complexity for nothing (rejection case 1 and 10).

### 4.2 Class B — one-shot and delayed jobs (River)

Send one notification, generate one report, process one webhook. River provides what pg-boss provided in the Node stack: Postgres-backed persistence, scheduled/delayed execution, automatic retries with backoff, and **unique jobs** for deduplication. Its defining property matches ADR-001 §7.3's transactional-boundary requirement in spirit: a job enqueued inside the application transaction only becomes visible if that transaction commits, eliminating the orphaned-job failure mode that motivated the outbox work. River is the **default engine** for this class — DBOS is not introduced merely to run one task asynchronously (ADR-001 Class B decision).

### 4.3 Class C — multi-step background operations (River first, DBOS by rubric)

Contact import with validation and a summary report; syncing a batch of external records. The template: a River job with bounded concurrency, per-record idempotency keys (`tenant:import:{id}:record:{n}`), and a progress record. If the operation needs cross-run checkpointing, cancellation mid-batch with resumption, or durable waits between phases, it scores on the adoption rubric (Amendment 1, four lenses) and may be promoted to Class D. The rubric — not habit — decides.

### 4.4 Class D — durable business workflows (DBOS Transact Go)

The heart of the catalog. The DBOS Go SDK provides library-in-process durable execution on Postgres: workflows with automatic checkpointing to system tables, durable sleeps that persist wake-up times, events for external signals, workflow queues for rate-limited fan-out, and recovery on process start. The Go SDK is Go-native — built around `context.Context` with compile-time typed workflow and step registration — and reached v1.x with continued investment through 2026.

The rea.md patterns translate as follows:

- **Submit/attach/signal** → the agent (or an HTTP caller) starts a workflow with an intentional workflow ID; any later caller attaches to the same ID to poll status instead of starting a duplicate; external responses arrive as DBOS events that resume the waiting workflow.
- **Human-in-the-loop** → the workflow emits a pending-approval record and executes a durable wait on an event; an approver's decision (human, or an agent operating under a trust mode that permits it) is delivered as the event. The workflow holds no machine resources while waiting.
- **Durable waits** → multi-day sleeps are the core fit signal (rubric lens FT3). The wait state persists in Neon; machines can be stopped entirely (Section 10).
- **Versioning** → pending workflows across deploys are handled per ADR-001 §7.5: compatibility-first, else drain or explicitly version; deployment guard checks running workflow versions before promoting a Machine.

### 4.5 The honest costs, carried over unchanged

Adopting DBOS still incurs the obligations ADR-001 §7.8 lists — extra database writes, connection budgets, system-schema migrations, retention policy, worker topology — plus the enterprise-feature boundary: capabilities such as suspending sleeping workflows from memory or distributed recovery with additional coordination have been documented as Enterprise-tier in DBOS's own materials, and the standing rule is to **verify the selected edition per feature** rather than assume open-source availability. Single-process recovery on worker start is the open-source baseline; the miniapp architecture deliberately embraces it (Section 10) rather than pretending distributed recovery is free. None of this changes under the pivot — but the rubric's AC lens now prices it once, centrally, in the platform layer instead of per feature.

---

## 5. Why This Stack Holds Together

### 5.1 One Postgres to rule them all

Neon hosts app data, DBOS system tables, and River's job tables. That single-backbone choice is the pivot's biggest simplification: DBOS's differentiator (per rea.md's own comparison) is that **the workflow checkpoint and the business write commit atomically in one Postgres transaction** — the exact property that Review 12 found hand-rolled boundaries violating (N2, N3, N4). River extends the same property to Class B/C: enqueue commits with the mutation that motivates it. One backup story, one retention policy, one branching tool for environments.

### 5.2 Go as the contract language

Go gives the agent surface typed, generated JSON Schemas; single static binaries that start in milliseconds on Fly Machines; first-class `context.Context` cancellation; and a concurrency model that suits rate-limited fan-out. The DBOS Go SDK's context-native design means workflow code reads like ordinary Go, which lowers the team-cognitive-tax line (rubric AC5) that the TypeScript stack would have kept higher.

### 5.3 The queue and the engine do not overlap

ADR-001 §8's coexistence rule becomes permanent architecture with a clean inventory:

| Workload (miniapp class) | Execution owner | Reason |
| --- | --- | --- |
| Class A operations | Go handler + Neon transaction | No lifecycle to preserve |
| Class B one-shot/delayed jobs | **River** | Transactional, simple, retries suffice |
| Class C bounded batch ops | **River** (promote via rubric) | Job contract + per-record idempotency |
| Class D durable journeys | **DBOS Transact Go** | Checkpoints, waits, events, recovery |
| Ordinary data mutations | Domain code + PostgreSQL | Never an engine |

One authoritative owner per responsibility; no second outbox for the same workflow.

### 5.4 Fly.io and Neon as the runtime pair

Fly Machines give each miniapp its own image, scale, and secrets, with auto-stop for idle workers — a natural fit for workflows that spend most of their life waiting in Neon, not in memory. Neon's serverless model complements this: environments are **branches** of production schema plus seed data, so every preview and CI run gets a real, isolated Postgres — including the DBOS system tables — in seconds. The known interplay to engineer deliberately: when all machines are stopped and the database autosuspends, a resuming workflow pays a cold-start latency (database resume plus machine boot) on wake; for multi-day waits that cost is negligible and the savings are real, while latency-sensitive paths (webhook receivers, the agent API) run on always-on machines with pooled connections (Neon's pooled endpoint) to respect DBOS and River connection budgets.

---

## 6. The Agent Harness Integration Layer

This is the new surface the pivot exists to build. It must serve two very different agent worlds with one contract.

### 6.1 Invocation surfaces

- **CLI-first.** Coding-agent harnesses (Claude Code, Codex, and similar) run shell commands fluently. Every miniapp publishes a subcommanded binary (`miniapp lead-followup start --tenant acme --lead 123 --dry-run`) whose `--help`, JSON output, and exit codes are part of the contract. The agent experience of a miniapp is a *good CLI*.
- **MCP tool exposure.** A platform MCP server lists each miniapp as a typed tool — name, description, JSON Schema input, structured output — so MCP-speaking harnesses get native tool-calling without shell parsing. The MCP server is a thin, stateless projection of the same contract; it forwards to the platform API and never executes workflows in-process.
- **HTTP for general agent runtimes.** Hermes, OpenClaw, and custom agents integrate via a signed HTTP API: `POST /v1/miniapps/{id}/invocations` with an agent principal, an idempotency key, and the typed payload; `GET .../invocations/{id}` for status; `POST .../invocations/{id}:cancel`. One OpenAPI spec, generated, versioned.

All three surfaces share one execution path. The CLI and MCP layer are adapters; the API owns authorization, identity, and audit.

### 6.2 The invocation lifecycle

1. **Submit** — the agent calls with an idempotency key (the **stable operation ID** of ADR-001 §7.2, e.g., `acme:lead:123:followup:v1`). Retried submits attach to the same execution instead of duplicating it.
2. **Acknowledge** — the API validates the manifest, resolves the trust mode, records the invocation with agent identity, and returns an execution ID immediately.
3. **Execute** — the durable workflow runs. Long waits persist in Neon; machines may stop.
4. **Report** — the agent polls status, or registers a webhook; terminal output is the typed result plus an execution summary (stages completed, effects performed, anything quarantined).
5. **Audit** — every transition is recorded: which principal, which trust mode, which effect, which provider operation ID.

### 6.3 Trust modes — the safety dial

| Mode | Behavior | Typical use |
| --- | --- | --- |
| `dry-run` (default for consequential ops) | Full workflow executes against real read APIs; every external effect is simulated and recorded in the result | Agent exploration, contract verification |
| `confirm` | Workflow runs to a durable gate and waits for an approval event from an authorized human (or an agent explicitly granted approver scope) | Bookings, sends above thresholds, payments |
| `autonomous` | Effects flow under the manifest's scope, rate, and budget limits; everything audited | Low-risk, high-volume operations |

Trust mode is bound in the permission manifest **per miniapp version per tenant**, not chosen freely by the agent. An agent can never talk a miniapp out of its manifest — this is the miniapp equivalent of rea.md's policy-mode guardrails, applied to the security perimeter rather than the framework internals.

### 6.4 What the agent never sees

Credentials, tenant tokens, provider payloads beyond its own result, or another tenant's execution. Workflow payloads carry identifiers and compact context (ADR-001 §7.6); the token broker injects credentials at step time and they are never serialized into checkpoints, logs, or tool outputs.

---

## 7. Integrations and Security

### 7.1 The connection registry and token broker

Each tenant registers connections in a **connection registry**: SaaS products via OAuth 2.0 (authorization-code with refresh tokens where the provider supports user-scoped access; client-credentials for tenant-scoped service integrations), custom apps via REST with API keys or mTLS client certificates. The registry stores secrets encrypted (application-level envelope encryption; keys in Fly secrets/KMS), records the granted **scopes** per connection, and exposes exactly one operation to workflows: *resolve a named binding for a tenant at step time*. The token broker handles refresh, revocation checks, and scope matching — a workflow step asking for `crm` gets a live token only if the manifest's scope requirement is satisfied; otherwise the step fails as a typed, terminal permission error.

### 7.2 Inbound webhooks

Provider callbacks (OAuth redirects, delivery events, agent approval callbacks) hit an always-on Fly endpoint that verifies **signature and timestamp** (HMAC per provider; reject replays), maps the event to a tenant and a pending workflow, and delivers it as a durable event. Webhook handling itself is a River unique job keyed on the provider event ID — a provider that retries the same delivery produces one execution.

### 7.3 The external-effect protocol (non-negotiable)

Every consequential outbound effect follows the protocol Review 12 made necessary:

1. **Stable operation ID** — `tenant:entity:operation[:attempt]`, deterministic, never a random UUID.
2. **Provider idempotency key** — passed wherever the SaaS supports it.
3. **Attempt records with explicit states** — `pending → sent → accepted | unknown | failed`; an effect is recorded `pending` before the HTTP call.
4. **Unknown outcomes are quarantined, not retried blind** — a timeout after a send leaves state `unknown`, the workflow proceeds on the compensation path, and a reconciliation step (human or agent-assisted, scoped read-only) resolves it.

This protocol is implemented **once** in the platform's effect-helper library; miniapps consume it, and CI's strict policy mode rejects effect steps that bypass it.

### 7.4 Authentication and authorization matrix

| Boundary | Mechanism | Authorization |
| --- | --- | --- |
| Agent → platform | Per-agent principal: API key or OAuth client credentials; optional mTLS for harnesses that support it | Role + tenant scope + trust-mode ceiling + rate/budget limits |
| Human operator → platform | SSO/OIDC | Admin roles, approval authority, registry management |
| Platform → SaaS | OAuth 2.0 per tenant connection | Scopes from consent, checked per step by the token broker |
| Platform → custom app | API key / mTLS per connection | Least-privilege endpoint list in the manifest |
| SaaS → platform | Signed webhooks (HMAC + timestamp) | Event-type allowlist per provider |
| Workflow → data | Neon Postgres role per miniapp + **row-level security** on tenant | Tenant isolation enforced in the database, not only in code |

Two rules make the matrix enforceable rather than aspirational. First, **agents are principals, not admins**: an agent's token encodes which tenants and which miniapp versions it may invoke, and the API rejects anything the manifest does not permit — including `confirm`-mode effects without an approval path. Second, **the manifest is the unit of change control**: a new miniapp version with broader scopes or a new provider binding is a reviewed deployment artifact, diffable in version control, auditable at runtime.

### 7.5 Audit and observability

Every invocation, decision, effect, and quarantine writes an audit event keyed by tenant, principal (human or agent), miniapp version, and operation ID. The execution-history views (who is waiting, what is stuck, what was quarantined) are themselves exposed as **read-only miniapps** — so an agent can ask "what is stuck for tenant acme?" through the same governed surface, without database access.

---

## 8. Reliability Standards (What "Robust" Means Here)

These standards are ADR-001 §7 restated as per-miniapp acceptance criteria; a miniapp version that fails any of them does not ship.

1. **Determinism** — workflow functions call steps in the same order on replay; clocks, randomness, DB reads, and HTTP live inside steps. A static check plus review checklist enforces it (rea.md's context-usage guardrail, made native to the Go SDK's step discipline).
2. **Idempotency** — every effect uses the external-effect protocol; every trigger uses a stable workflow ID; every River producer uses unique-job keys where duplication matters.
3. **Transactional boundaries** — business mutation + checkpoint (or + enqueue) commit together; verified in tests against the real Neon database, not assumed.
4. **Identity and deduplication** — retried submits, double webhooks, and agent re-plans collapse onto one execution.
5. **Versioning** — deployment guard refuses to promote an image with breaking workflow changes while executions of the old version are pending; drain or explicit version routing is planned per ADR-001 §7.5.
6. **Recovery policy** — retry limits, terminal-failure inspection (the DLQ equivalent), cancellation, ownership, alerting, and retention are declared in the manifest, not improvised per incident.
7. **Crash-boundary acceptance** — the automated suite: start a workflow, wait meaningfully, `kill -9` the worker, replace the Machine, resume from the correct checkpoint, finish with **zero stranded runs and zero duplicate external effects**; double-submit the trigger and prove single execution. This is the pilot acceptance of the Amendment 1 rubric, generalized into CI.
8. **Admission** — a new miniapp proposal is scored on the four-lens rubric (BV, FT, AC, CW) like any other Class D candidate. The pivot does not weaken ADR-001; it industrializes its discipline. Notably, the platform's central token broker, effect library, and test harness **lower AC across the whole catalog**, which is precisely the argument the rubric's Gate 4 demands.

---

## 9. Reference Miniapp Catalog

Initial catalog, mapped to classes and engines. Contracts are sketches; the rubric governs admission and the manifest governs scope.

| Miniapp | Class / engine | Contract sketch |
| --- | --- | --- |
| `lead-followup` | D / DBOS | In: tenant, lead ID, sequence plan. Waits days, re-checks eligibility via `crm`, sends via `mail`, honors reply/unsubscribe events; cancel-safe; confirm mode for sends |
| `appointment-journey` | D / DBOS | Multi-touch reminders before/after an appointment; quiet-hours policy; reschedules cancel the journey |
| `approval-flow` | D / DBOS | In: request + approver set; durable wait on decision event; escalates on timeout; typed outcome |
| `agent-toolchain` | D / DBOS | Wraps a multi-step AI agent execution: each tool call a step, expensive calls checkpointed, replay without re-spend |
| `campaign-orchestration` | D / DBOS, **K4-gated** | Fan-out with rate limits via workflow queues; blocked until the external-effect protocol proves out on real sends (knockout K4 from the rubric) |
| `contact-import` | C / River | Batch validate + upsert + report; per-record idempotency; progress query |
| `data-sync` | C / River | Pull-merge-push for one external record type; conflict policy in manifest |
| `report-generate` | B / River | One-shot or scheduled report render + delivery |
| `notify-send` | B / River | Single notification with idempotency key and delivery receipt |
| `crm-read`, `crm-write` | A / handler | Typed, permission-checked CRUD for agents; the governance shell is the product |

---

## 10. Deployment Topology on Fly.io + Neon

- **Per-miniapp Fly app** (or a shared host app per tenant tier, when volumes are small): its own image, machine count, and secrets; Class B/C workers are River-only consumers; Class D hosts register DBOS workflows and run recovery on start.
- **Always-on edge**: the agent API, MCP server, and webhook receiver run as small always-on Machines; everything else auto-stops when idle.
- **Environments as Neon branches**: `main` → production, every PR → a branch database with the full schema (app + DBOS + River) and masked seed data; CI runs the crash-boundary suite against the branch before a release is allowed to promote.
- **Migrations**: versioned SQL migrations for app schema, DBOS system tables, and River schema, applied by the same tool in the same gate; production migration runs are a deployment-guard step, per the environment-isolation discipline already adopted.
- **Secrets and keys**: static platform secrets in Fly secrets; tenant connection secrets in the encrypted registry (not Fly secrets, which are per-app static config); encryption keys in KMS.
- **Observability**: Fly logs → aggregator; OTel traces from invocation to effect; DBOS/River dashboard-equivalents queried through the read-only inspection miniapps; alerts on quarantines, stranded-run detection, and stuck-wait ages.

---

## 11. Roadmap

| Phase | Scope | Exit criteria |
| --- | --- | --- |
| **0 — Platform foundation** | Go platform library: contract schema generation, manifest validation, token broker, effect protocol helper, policy modes, River+DBOS bootstrap, crash-boundary test harness | Harness runs green against a throwaway Neon branch; policy strict mode enforced in CI |
| **1 — First miniapp pilot** | `lead-followup` (Class D) end-to-end: CLI + HTTP, `dry-run` and `confirm` modes, one SaaS OAuth connection | Crash-boundary suite passes on real infra; zero stranded runs, zero duplicate sends; rubric sheet ratified |
| **2 — Agent surface + catalog breadth** | MCP server; agent principal + manifest enforcement; add `approval-flow`, `appointment-journey`, `contact-import`, `report-generate` | Two external agent harnesses (one coding agent, one general runtime) invoke the catalog through the published contract |
| **3 — Multi-tenant GA** | RLS isolation audited; connection registry self-serve; budgets/rate limits; autonomous mode for low-risk ops; `campaign-orchestration` unblocked by K4 protocol evidence | Onboarding a new tenant + SaaS connection requires no code change; audit trail complete per invocation |
| **4 — Consolidation** | Retire redundant in-app plumbing per the workload inventory; pg-boss-era machinery decommissioned where the inventory shows miniapps own the workload | One authoritative execution owner per responsibility; simplest reliable architecture |

---

## 12. Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| DBOS Go SDK is younger than the TS/Python SDKs | Pin versions; keep workflow logic thin over a stable platform layer; run the crash-boundary suite per upgrade; watch the release cadence (v1.x actively maintained) |
| Enterprise-gated capabilities (sleep suspension, distributed recovery) | Design to the open-source baseline: waits persist in Neon, machines stop, recovery on start; re-evaluate Enterprise only with measured need (rubric AC4) |
| Neon autosuspend cold starts on wake | Accept latency for multi-day waits; keep latency-sensitive paths always-on; pooled endpoints and connection budgets per §5.4 |
| Agent over-permissioning | Manifest-bound trust modes; agents are scoped principals; confirm-mode gates for consequential effects; full audit |
| Webhook spoofing/replay | Signature + timestamp verification, event-ID uniqueness, allowlists |
| Duplicate effects from retries everywhere | External-effect protocol enforced by CI strict mode; unique jobs in River; stable workflow IDs in DBOS |
| Catalog sprawl / miniapp-per-task bloat | Rubric admission; Class A/B stay handlers/jobs; workload inventory review per quarter |
| Version drift between contract, binary, and docs | Generated schemas, CI contract tests, immutable versioned manifests |

---

## 13. Decision and Next Steps

The pivot asks for three commitments:

1. **Ratify the miniapp platform as the delivery model** for business process automation — Go, DBOS Transact Go, River, Fly.io, Neon — with ADR-001, its Amendment 1 rubric, and the crash-boundary standards carried forward as the miniapp acceptance bar.
2. **Fund Phase 0/1** as defined above: the platform library and the `lead-followup` pilot, the same workload the adoption analysis already scored as the strongest candidate (V=23, D=+4).
3. **Adopt the security perimeter as a product feature**: agent principals, manifests, trust modes, and the external-effect protocol ship in Phase 1, not later — because "robust" for agent-invoked automation means governed, not merely reliable.

The strategic bet is simple to state. Agents will increasingly be the ones asking work to be done. The winners in that world are not the systems that let agents do anything, but the ones that make **doing the right thing durably** the easiest path. Miniapps are OpenLevel's version of that path: the reliability machinery we already learned to build, packaged as the thing agents call first.

---

## References

- DBOS Transact Go SDK — https://github.com/dbos-inc/dbos-transact-golang
- DBOS Transact (durable execution library; Python/TypeScript/Go/Java/Rust) — https://docs.dbos.dev/
- DBOS Architecture — https://docs.dbos.dev/architecture
- DBOS Workflow Tutorial (determinism, steps) — https://docs.dbos.dev/typescript/tutorials/workflow-tutorial
- DBOS Transactions and Datasources — https://docs.dbos.dev/typescript/tutorials/transaction-tutorial
- DBOS Workflow Recovery — https://docs.dbos.dev/production/workflow-recovery
- DBOS Queues — https://docs.dbos.dev/typescript/reference/queues
- DBOS Upgrading Workflows (versioning) — https://docs.dbos.dev/typescript/tutorials/upgrading-workflows
- River — background jobs for Go backed by Postgres — https://riverqueue.com
- River transactional enqueueing — https://riverqueue.com/docs/transactional-enqueueing
- River repository — https://github.com/riverqueue/river
- Fly.io Machines — https://fly.io/docs/machines/
- Fly.io Machine configuration (fly.toml) — https://fly.io/docs/reference/configuration/
- Fly.io Secrets — https://fly.io/docs/flyctl/secrets/
- Neon Postgres (serverless Postgres, branching) — https://neon.com/docs/introduction
- Neon connection pooling — https://neon.com/docs/connect/connection-pooling
- OpenLevel internal: ADR-001 — Selective Adoption of DBOS; ADR-001 Amendment 1 — The DBOS Adoption Rubric; Adversarial Review 12 retrospective
