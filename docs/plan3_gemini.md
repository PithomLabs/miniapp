# Agent-Invokable RPA Miniapps: Unified Technical Architecture

## 1. Strategic Reframe: The Core Inversion

This pivot shifts miniapps from **human-facing UI surfaces** to **agent-facing executable contracts**—API-first, RPA-style business process automations that coding harnesses (Claude Code, Codex) and general agent runtimes (Hermes, OpenClaw) discover, invoke, and monitor.

> **The agent is the planner and caller. The miniapp is the governed executor. The agent never holds a credential, never owns durability, and never bypasses policy.**

| Responsibility | Owner | Rationale |
| --- | --- | --- |
| **Intent, planning, sequencing across systems** | AI Agent (Harness) | Agents excel at reasoning over messy context and choosing tools.

 |
| **Durable execution across restarts & multi-day waits** | Miniapp Runtime (DBOS / River) | Agent sessions terminate; multi-day business processes cannot.

 |
| **Credentials, OAuth tokens, & refresh rotation** | Capability Proxy / Token Broker | Agents must never touch raw SaaS secrets or launder user scopes.

 |
| **Idempotency & crash-safe external side effects** | Miniapp Step & Effect Protocol | Agent retries and re-plans must never generate duplicate invoices or sends.

 |
| **Tenant isolation, RBAC, & human approval gates** | Miniapp Policy & DB RLS | Server-side enforcement; never rely solely on harness prompts or client flags.

 |

Classic RPA was brittle because it automated fragile UIs and baked state into scripts. Agentic RPA is brittle if the agent holds credentials and state. Packaging processes as declarative, versioned miniapps moves state, secrets, and invariants into a tested, auditable Go runtime.

---

## 2. Stack Allocation & Coexistence Rules

Every component in the required stack has a strict, non-overlapping boundary:

| Layer | Technology | Role & Non-Negotiable Rule |
| --- | --- | --- |
| **Runtime & Contracts** | **Go** (1.26+) | Single static binary per runtime/worker; typed JSON Schemas generated from Go structs; `context.Context`-native cancellation.

 |
| **Agent Interface** | **Official Go MCP SDK** + REST/OpenAPI | Primary MCP surface (Streamable HTTP / stdio) alongside versioned REST endpoints; stateless adapter over the execution router.

 |
| **Simple & Batch Jobs** | **River** (`riverqueue/river`) | The Go/Postgres equivalent of `pg-boss`. Handles Class B/C jobs via transactional `InsertTx`, unique keys, retries, and scheduled runs.

 |
| **Durable Workflows** | **DBOS Transact Go** | Handles Class D long-running processes: deterministic replay, step checkpointing, durable `Sleep`, event signals (`Recv`), and recovery.

 |
| **Database** | **Neon Postgres** | Single transactional backbone for domain data, River queues, and DBOS system tables. Ephemeral branches power CI crash tests.

 |
| **Compute** | **Fly.io Machines** | Co-located in the **exact same region** as the Neon primary to eliminate checkpoint latency. Separate process groups for API/MCP, River, and DBOS.

 |

### The Coexistence & Connection Rules

1. **One Authoritative Owner:** A miniapp declares its substrate in its manifest. Never dispatch the same workload to both River and DBOS.


2. **Neon Connection Pooling Split:** DBOS relies on session-level Postgres features (such as `LISTEN/NOTIFY`) and **must use Neon's direct (unpooled) connection** with a bounded `pgxpool`. Stateless Class A API handlers and high-concurrency River workers should use Neon's pooled endpoint.


3. **DBOS Scale-to-Zero & Recovery:** Open-source DBOS recovers workflows on process startup. Start with a single dedicated DBOS executor group (or auto-stop/start single machine per workload) and adopt DBOS Conductor only when multi-process distributed recovery is empirically required.



---

## 3. Taxonomy: `rea.md` Patterns Mapped to Miniapp Categories

Every miniapp maps to a concrete pattern from `rea.md`. **DBOS is rejected by default for Classes A, B, and C** to avoid unnecessary replay-determinism constraints and checkpoint write amplification under high-frequency agent tool calls.

| Category (`rea.md`) | Miniapp Form | Substrate | Operational Contract & Example |
| --- | --- | --- | --- |
| **Class A: Action** | Single-step synchronous read/write | **Go Handler + Neon Tx** | *Example:* `crm.get_contact`, `billing.create_draft_invoice`. Runs inside a standard Postgres transaction. Typed validation errors return immediately as terminal errors.

 |
| **Class B: Job** | Fire-and-forget or delayed async task | **River** (`InsertTx`) | *Example:* `notify.send_receipt`, `reports.generate`. Enqueued atomically in the same Postgres transaction as the domain mutation; returns a job handle immediately.

 |
| **Class C: Pipeline** | Bounded multi-step batch operation | **River** (Batch / Unique Jobs) | *Example:* `crm.import_contacts_batch`. Uses per-record idempotency keys (`tenant:batch:{id}:row:{n}`) and bounded worker concurrency. Promoted to DBOS only if mid-batch crash resumption is required.

 |
| **Class D: Workflow** | Multi-day durable process or approval loop | **DBOS Transact Go** | *Example:* `hr.onboard_employee`, `sales.lead_followup`. Persists sleeps across days, pauses on human approval events without holding compute, and resumes from the exact step after a crash.

 |
| **Virtual Object** *(Composable)* | Per-entity serialized state machine | **Postgres Row Lock / Advisory Lock** | *Example:* `order.lifecycle` (`workflow_id = tenant:entity:kind`). Enforces single-writer serialization via Postgres optimistic version checks or advisory locks rather than assuming native actor primitives.

 |
| **Saga** *(Composable)* | Multi-system reversible mutation | **DBOS Steps + LIFO Unwind** | *Example:* Create CRM account → Create Billing → Provision Workspace. Pre-registers explicit, idempotent compensation steps *before* each forward side effect; unresolvable failures quarantine to manual review.

 |

---

## 4. The Agent Harness Integration Layer (MCP + CLI + REST)

Agents fail when forced to block on long-running synchronous tool calls. All async miniapps (Classes B, C, D) expose a uniform **handle-and-signal envelope** across all agent harnesses (Claude Code, Codex, Hermes, OpenClaw):

### 4.1 Uniform Async Invocation Envelope

```json
{
  "run_id": "run_01J9X8K2M4...",
  "miniapp_id": "rpa.billing.chase_invoice",
  "version": "1.2.0",
  "status": "waiting_approval",
  "poll_tool": "rpa.runs.status",
  "signal_tool": "rpa.runs.signal",
  "cancel_tool": "rpa.runs.cancel",
  "expires_at": "2026-12-06T13:55:00Z"
}

```

### 4.2 Harness Surface Strategy

* **MCP Gateway (Primary):** Exposes namespaced tools (`rpa.<domain>.<verb>_<noun>`) with JSON Schemas, MCP tool annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`), and task/elicitation support for human-in-the-loop approvals.


* **CLI Wrapper (Coding Harnesses):** A thin, subcommanded Go CLI (`rpa run <miniapp> --json --idempotency-key ...`) that calls the same platform gateway, giving Claude Code and Codex clean exit codes and structured stdout without shell-parsing hacks.


* **Signed REST/OpenAPI (General Harnesses):** Standard endpoints (`POST /v1/miniapps/{id}/runs`) for Hermes, OpenClaw, and webhook triggers.



### 4.3 Declarative Miniapps Over Generated Code

LLMs may author or propose a declarative **`MiniappSpec` (`miniapp.json`)**, never arbitrary Go code deployed directly to production. The trusted Go runtime compiles and interprets `MiniappSpec` against pre-vetted, typed capability adapters. No miniapp version can be published unless its deterministic fixture suite (`fixtures/*.json`) passes in CI.

---

## 5. Security, Auth, Tenancy & SaaS Integration Governance

Because agents are autonomous callers acting on behalf of users and tenants, security cannot rely on prompt instructions or client-supplied metadata.

### 5.1 Dual-Boundary Identity & Token Broker

1. **Inbound (`Agent → Platform`):** Every invocation authenticates an **agent principal** bound to a **delegated user/tenant context**. If a write tool cannot resolve a valid `(agent_id, user_id, tenant_id)` principal, it fails closed.


2. **Outbound (`Platform → SaaS / Custom REST`):** Workflow payloads only ever store opaque **connection references** (e.g., `conn_hubspot_prod`), never tokens. At step execution time, the **Token Broker** validates that the miniapp's manifest permits the requested operation and scope, resolves the envelope-encrypted credential, handles OAuth 2.0 refresh-token rotation with reuse detection, and injects the header into the outbound HTTP client.



### 5.2 Manifest-Bound Trust Modes

Each miniapp version is bound per tenant to an explicit trust mode in its permission manifest:

* `dry-run`: Executes real read steps; simulates all external writes and returns a diff preview.


* `confirm`: Executes up to a consequential write, persists state, and pauses on a durable DBOS event (`Recv`) until an authorized human principal approves via MCP elicitation or the control plane.


* `autonomous`: Executes end-to-end within strict manifest ceilings (allowed endpoints, rate limits, spend/record caps).



### 5.3 The External-Effect Protocol & Database Isolation

* **External-Effect Protocol:** Every outbound SaaS or custom REST mutation generates a deterministic operation ID (`tenant:entity:step:attempt`), records an intent row (`pending`) *before* dispatching the HTTP request, passes the provider's `Idempotency-Key` header, and transitions to `accepted`, `failed`, or `unknown`. Timeouts after dispatch land in `unknown` and **quarantine** for reconciliation rather than blind retries.


* **Inbound Webhooks:** Verified via provider HMAC signature and timestamp window, deduplicated via River unique jobs keyed on the provider's event ID, and routed to waiting DBOS workflows.


* **Neon Tenant Isolation:** Every tenant-scoped table enforces Postgres Row-Level Security (`FORCE ROW LEVEL SECURITY`) keyed on the verified `tenant_id` session variable, with Neon project-per-tenant isolation available for high-compliance tiers.



---

## 6. Deployment Topology (Fly.io + Neon)

```text
                       [ AI Agent Harnesses ]
             (Claude Code · Codex · Hermes · OpenClaw)
                                 │
                   MCP (Streamable HTTP) / REST
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ Fly.io App: rpa-platform (Single Region matching Neon Primary)  │
│                                                                 │
│  ┌───────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │ Process: gateway  │  │ Process: river   │  │ Process: dbos│  │
│  │ (MCP, REST, Auth, │  │ (Class B/C Jobs, │  │ (Class D     │  │
│  │  Webhooks, ClassA)│  │  Batch, Webhooks)│  │  Workflows)  │  │
│  └─────────┬─────────┘  └────────┬─────────┘  └──────┬───────┘  │
└────────────┼─────────────────────┼───────────────────┼──────────┘
             │ (Neon Pooler)       │ (Neon Pooler)     │ (Direct pgxpool)
             ▼                     ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│ Neon Postgres (Single Primary)                                  │
│  ├── schema: app    (Domain tables, RLS, Effect log, Registry)  │
│  ├── schema: river  (Transactional job queues & unique locks)   │
│  └── schema: dbos   (Workflow checkpoints, sleeps, events)      │
└─────────────────────────────────────────────────────────────────┘

```

---

## 7. Phased Execution & Exit Gates

| Phase | Deliverable | Mandatory Exit Gate |
| --- | --- | --- |
| **0. Contracts & Schema** | Define `MiniappSpec` JSON schema, MCP tool envelope, Capability Connector interface, Effect Protocol library, and Neon schema layout (direct vs. pooled connections).

 | Schemas ratified; DBOS Go + River verified side-by-side against a Neon branch.

 |
| **1. Crash-Proof Vertical Slice** | Build one Class A read miniapp and one Class D durable miniapp (`lead-followup` or `customer-onboarding`) callable via MCP and CLI, with mock SaaS endpoints.

 | **Crash-Boundary Suite:** Kill the Fly Machine (`fly machine stop` / `SIGKILL`) mid-workflow; prove recovery from exact step with zero stranded runs and zero duplicate effects.

 |
| **2. Token Broker & First SaaS** | Implement envelope-encrypted Connection Registry, OAuth 2.0 refresh rotation, fail-closed agent/user auth, Postgres RLS, and one live SaaS + one custom REST adapter.

 | Cross-tenant probe fails at DB RLS layer; unauthenticated or scope-exceeding agent call fails closed; zero tokens in logs/checkpoints.

 |
| **3. River Batch & Webhooks** | Enable River workers for Class B/C miniapps (`contact-import`) and HMAC-verified inbound SaaS webhooks.

 | Duplicate webhook deliveries and retried agent batch calls collapse into single idempotent executions.

 |
| **4. Approval Gates & Sagas** | Wire `confirm` trust mode (`Recv` + MCP elicitation) and LIFO compensation helpers for multi-system writes.

 | Consequential write pauses across machine restarts until human approval event arrives; simulated downstream failure triggers clean compensation.

 |
| **5. Multi-Harness & Spec Gen** | Validate across Claude Code, Codex, Hermes, and OpenClaw; add LLM-assisted `MiniappSpec` generation gated by automated fixture validation.

 | Two distinct agent harnesses invoke the same miniapp catalog; LLM-generated specs pass fixture CI gates before publishing.

 |