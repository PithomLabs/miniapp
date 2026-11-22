# Consolidated Implementation Plan — Go Agent-Invokable Miniapp Framework

- **Status:** Planning only. No code, migrations, or deployments were changed to produce this plan.
- **Module:** `github.com/PithomLabs/miniapp` (single Go module, per instruction)
- **Primary blueprint:** `docs/plan3_z.md`, supplemented by `docs/plan3_deepseek.md`
- **Task source:** `docs/prompt.md` (§5 task: implementation planning only)
- **Plan path:** `docs/plan4.md` (this file)
- **External versions verified in this run:** none — all dependency versions are UNPINNED and must be pinned in Phase 0 with verification dates and source links.

## 1. Executive summary and final architecture recommendation

Build two deliverables in one repository and one Go module:

1. **A reusable Go framework library** that owns contracts, registry, runtime,
   execution adapters (plain Go, River, DBOS), capabilities, identity and
   authorization, token brokering, the external-effect protocol, approvals,
   audit/observability, the three agent-facing adapters (MCP, REST/OpenAPI,
   CLI), and a test kit.
2. **A catalog of reference miniapps** built only against the framework's
   public APIs, proving each execution class and each framework guarantee.

Recommended architecture: a **shared platform runtime** (one Fly app with
process groups for gateway, River workers, and a DBOS executor) backed by one
Neon Postgres primary (separate schemas for app, River, and DBOS); one
**canonical invocation service** fronted by thin MCP/REST/CLI adapters; a
**central MCP router** with namespaced tools; a **two-principal authorization
model** (agent principal + authority principal) with manifest-bound trust
modes; and an **external-effect protocol** with explicit UNKNOWN-outcome
quarantine and reconciliation. Class A runs as plain Go handlers, Class B/C
on River, Class D on DBOS — each workload has exactly one execution owner,
and DBOS must earn adoption per case under the ADR-001 policy in
`docs/DBOS.md`.

The first code proves the thesis with a vertical slice (`crm-read` for
Class A, `lead-followup` for Class D, then `contact-import` for Class C)
before the rest of the catalog is built. No AI miniapp generator, no browser
automation, and no new workflow engine are in scope.

## 2. Product boundaries and explicit non-goals

The product is an API-first business-process automation platform. An agent in
a harness (Codex, Claude Code, Hermes, OpenClaw) discovers and invokes a
miniapp through a typed interface; the miniapp owns credentials, execution,
recovery, and audit. The agent never holds provider credentials, never owns
durable state, and never bypasses server-side permission checks.

A miniapp is a governed, versioned automation unit: contract, execution
lifecycle, permitted capabilities, brokered credentials, authorization and
approval boundaries, idempotency and external-effect protocol, durable
execution when required, audit, and acceptance tests.

Explicit non-goals for the initial scope:

- General-purpose AI agent runtime, harness replacement, or reasoning loop.
- Browser or desktop automation (requires a separate governed capability).
- UI/dashboard generation as the product surface (headless, agent-invoked).
- General BPMN replacement, universal business-rules engine, or rulez engine work.
- A new custom workflow engine competing with DBOS.
- Free-form LLM code execution; generated or authored miniapps must not run
  arbitrary unreviewed Go, shell, or JavaScript.
- AI miniapp generator (deferred; see §22).

## 3. Source-document reconciliation table

Precedence applied: (1) two-deliverable requirement, (2) ADR-001 policy in
`docs/DBOS.md`, (3) `plan3_z.md`, (4) `plan3_deepseek.md`, (5) `rea.md`,
(6) actual repository evidence. The repository is docs-only, so item 6
contributes the greenfield finding, not code constraints.

| # | Disagreement | Sources | Resolution | Rationale |
|---|---|---|---|---|
| R1 | River chain as durable workflow abstraction vs explicit progress model | DeepSeek §4 vs prompt §2A | Explicit per-record progress model and idempotency for Class C; promote to DBOS only when checkpointed recovery genuinely matters | River's verified surface is transactional jobs, retries, scheduling, uniqueness — not a workflow abstraction |
| R2 | "Zero duplicate external effects" as acceptance language | plan3_z §§3,8.7 vs prompt §2B | "No duplicate confirmed business effects, no silently lost operations, no blind retries of unknown outcomes" | A step can execute twice across the provider-complete/checkpoint-missing crash window |
| R3 | Fly Machines stop during waits and resume later | plan3_z §4.4/§10, DeepSeek §8 vs prompt §2C | Baseline: always-available DBOS executor; prove wake topology before any scale-to-zero claim | Sleep suspension may be Enterprise-gated; Fly autostart is request-driven, not timer-driven |
| R4 | Identity: user-delegated token mandatory for every write vs agent principal | DeepSeek §7 vs plan3_z §7.4 | Two-principal model: agent principal plus authority principal (delegated user, approved service identity, or authorized trigger) | Supports interactive and unattended automation without granting agents admin authority |
| R5 | RLS via `auth.user_id()` | DeepSeek §7 | Explicit tenant-context mechanism (session variable, roles, `FORCE ROW LEVEL SECURITY`); RLS is defense in depth behind mandatory app authorization | No such built-in exists in standard Neon Postgres |
| R6 | One Fly app per miniapp vs shared app with process groups | plan3_z §10 vs DeepSeek §8 | Shared platform runtime by default; dedicated apps/groups only on proven isolation, scaling, or customer need | Lower operational cost; preserves independent scaling where justified |
| R7 | Declarative step-DSL runtime vs Go miniapps on a framework library | GPT §3C/§6 vs prompt §7 | Behavior is Go code against public framework APIs; `miniapp.json` is the contract/permission manifest, not a full behavior DSL | Matches the "reusable Go framework library" deliverable; a trusted step DSL is a later, separately-scoped product |
| R8 | Generic MCP tools vs namespaced per-miniapp tools | GPT §3A vs DeepSeek §5 | Staged: generic discovery/invoke/status tools first, namespaced tools (`rpa.<domain>.<verb>_<noun>`) once harness interop is proven | Interoperability first, ergonomics second; router stays central either way |
| R9 | "Shared Neon URL" implying atomic DBOS/app commits | plan3_z §5.1, DeepSeek §8 | Verify handle-level datasource integration (`RunAsTransaction` or version equivalent); prove crash windows with tests | Atomicity depends on pool-handle configuration, not on sharing a URL |
| R10 | Anecdotal ~150ms cross-region latency as universal fact | DeepSeek §8 | Measure; place Fly region with the Neon primary by benchmark, not anecdote | Latency claims must be measured per deployment |
| R11 | rulez/RuleSpec as product foundation | plan.md/plan2.md vs pivot docs | rulez is an optional deterministic decision component, out of initial scope | Pivot redefined the product as agent-invoked execution, not generated UIs |
| R12 | "Entity miniapp" as a fifth category | GPT §2 | Entity state is a composable pattern (Postgres concurrency + optional DBOS orchestration), not a separate class | Prompt §4 and ADR-001 fix the A/B/C/D taxonomy |
| R13 | `agent-toolchain` in the initial catalog | plan3_z §9 | Assess separately with its own scope justification; default is later | Must not duplicate the harness reasoning loop |
| R14 | Async handle with short expiry | DeepSeek §5 envelope | Stable run IDs with no arbitrary active-handle expiry; terminal-history retention is a separate policy | Multi-day waits outlive short expirations |
| R15 | DBOS Conductor / multi-executor recovery assumed | GPT §4 | Start with one dedicated DBOS executor; add Conductor only on measured multi-process need | Keeps licensing and topology explicit |

## 4. Repository inventory and reuse/adapt/replace/retire assessment

The workspace at `/home/chaschel/Documents/go/miniapp` contains only
`docs/` (12 Markdown files). There is no `go.mod`, no Go source, no
migrations, no Fly/Neon configuration, no CI, no git checkout, and no
ADRs, runbooks, or `AGENTS.md` beyond the docs themselves.

| Item | Finding | Disposition |
|---|---|---|
| `docs/prompt.md` | Task definition and architecture corrections (§§1–15) | Reuse as the governing task contract |
| `docs/plan3_z.md` | Primary product/implementation architecture | Reuse as primary blueprint, subject to §3 corrections |
| `docs/plan3_deepseek.md` | Agent-facing MCP/manifest/security supplements | Reuse as supplementary design |
| `docs/plan3_gpt.md`, `docs/plan3_gemini.md` | Corroborating architectures (gateway, coexistence, envelopes) | Reuse as cross-checks; no separate authority |
| `docs/DBOS.md` | ADR-001 selective-adoption policy text (Classes, Gates, score, template) | Reuse as mandatory adoption policy; formalize as ADR-001 file in Phase 0 |
| `docs/rea.md` | Restate-vs-DBOS comparison; Virtual Object, Saga, guardrail patterns | Reuse patterns only; Restate is not on the stack |
| `docs/plan.md`, `plan2.md`, reviews | Superseded UI-generator framing | Retire as product direction; retain durable lessons (declarative artifact + trusted runtime, CapabilitySpec depth, security-early, falsification discipline, audit minimization) |
| `docs/plan3_review.md` | Empty file | Investigate: confirm whether a review was expected, then either fill or remove in Phase 0 |
| ADR-001 file, Review 12 retrospective, OpenLevel code | Referenced by sources but absent from this repo | Missing: treat cited claims from them as unverified; do not assume Hono/pg-boss-era mechanisms exist here |
| Go toolchain, modules, CI, Fly/Neon config | Absent | Greenfield: create in Phase 0/1 under `github.com/PithomLabs/miniapp` |

Consequence: every framework component is greenfield. Nothing is adapted or
replaced; the docs are the only reuse. Phase 0 therefore carries real
contract and spike work rather than being a code-survey phase.

## 5. Key architecture decisions and ADR references

Phase 0 formalizes these as ADR files under `docs/adr/` (new directory).

| ID | Decision | Choice |
|---|---|---|
| ADR-001 | DBOS adoption policy | Adopt `docs/DBOS.md` verbatim as the governing policy: default reject; Class D earns DBOS through Gates 1–4 and the complexity score |
| ADR-002 | Module and repo layout | One repo, one Go module `github.com/PithomLabs/miniapp`; framework packages plus `miniapps/` references; split modules only if versioning later demands it |
| ADR-003 | Deployment unit | Shared platform runtime (one Fly app, process groups) by default; separately deployable miniapp binaries/modules; dedicated apps on justification |
| ADR-004 | Invocation architecture | One canonical invocation service; MCP router, REST/OpenAPI, and CLI are thin adapters on the same execution, authorization, and audit path |
| ADR-005 | Authorization model | Two principals (agent + authority), trust modes (`dry-run`, `confirm`, `autonomous`) bound per miniapp version per tenant, server-side enforcement |
| ADR-006 | External-effect semantics | Honest at-least-once with UNKNOWN quarantine and reconciliation; corrected acceptance language per R2 |
| ADR-007 | Connection strategy | DBOS on Neon direct connection initially; app and River modes decided per workload after compatibility testing; explicit connection budgets |
| ADR-008 | DBOS executor topology | Single dedicated executor, always available for waits; no scale-to-zero for the DBOS worker until the wake test passes; Conductor on measured need |
| ADR-009 | Manifest authority | `miniapp.json` is the versioned artifact of record: identity, tools, substrate, capabilities, policy, fixtures, exact dependency manifest |
| ADR-010 | Miniapp-toolchain scope | `agent-toolchain` and any generator work need separate scope justification; default is deferred |

## 6. Target architecture diagrams

### 6.1 System context

```text
[Agent harnesses: Codex, Claude Code, Hermes, OpenClaw]
        |  MCP (Streamable HTTP / stdio)   |  signed REST   |  CLI
        v                                  v                v
+------------------+  +------------------------------------------+  +------------------+
| Central MCP      |  | Canonical invocation service             |  | CLI (thin client)|
| router           |->| registry, auth, dispatch, run handles,   |<-| over REST        |
| (namespaced      |  | audit, approvals                         |  |                  |
|  tools + generic |  +-----+--------+------------+------------+--+  +------------------+
|  discovery)      |        |        |            |            |
+------------------+        v        v            v            v
                     +--+------+ +--+-------+ +--+--------+ +--+---------+
                     | Class A | | River    | | DBOS      | | Capability |
                     | handler | | workers  | | executor  | | proxy +    |
                     | + tx    | | (B/C)    | | (D)       | | token      |
                     +--+------+ +--+-------+ +--+--------+ | broker     |
                        |           |            |          +--+---------+
                        v           v            v             |
+---------------------------------------------------------------------+
| Neon Postgres (single primary, same region as Fly)                  |
|   app schema: domain, runs, effects, approvals, audit, registry     |
|   river schema: job tables                                          |
|   dbos schema: workflow checkpoints, sleeps, events                 |
+---------------------------------------------------------------------+
                          | SaaS OAuth APIs | custom REST APIs | webhooks
                          v
                 [External business systems]
```

### 6.2 Dependency direction (binding rule)

```text
miniapps/<name>  --->  framework public API (contract, runtime, capability,
                       auth, effects, approvals, audit, testkit)
                             |
transport adapters --->  invocation service  --->  executor adapters
(MCP / REST / CLI)       (auth + dispatch)        (classa / river / dbos)
                             |                            |
                             v                            v
                      security/policy/persistence    capability clients
                                                      (no credential leaks
                                                       into workflows)
```

Rules: the framework never imports reference miniapps; transports own no
business logic and no separate authorization; provider clients never leak
credentials or transport behavior into workflow code; DBOS and River sit
behind stable framework interfaces that keep their real semantics visible
(no leaky generic "job" abstraction that hides replay vs retry).

### 6.3 Invocation lifecycle

Submit (idempotency key = stable operation ID) -> validate manifest, resolve
trust mode, record invocation with agent identity -> return run handle
immediately -> execute on the declared substrate -> agent polls status or
receives webhook -> terminal typed result plus execution summary -> every
transition audited.

## 7. Framework library responsibilities and package/module structure

Proposed layout under the single module `github.com/PithomLabs/miniapp`
(all paths are proposals; nothing exists yet):

```text
go.mod                          module github.com/PithomLabs/miniapp
contract/                       manifest + schemas + descriptors (public)
  manifest.go                   miniapp.json parse/validate
  schemas.go                    Go structs -> JSON Schema generation
  descriptors.go                tool descriptors, error taxonomy
registry/                       discovery, registration, version resolution
runtime/                        invocation lifecycle, dispatch, run handles
executor/
  executor.go                   substrate interfaces + admission (ADR-001 gate)
  classa/                       plain handler + tx adapter
  river/                        Class B/C adapter (progress, uniqueness)
  dbos/                         Class D adapter (workflows, events, recovery)
capability/
  spec.go                       CapabilitySpec contract
  registry.go                   connector registry + allowlists
  clients/                      typed SaaS/custom-REST clients (no creds held)
auth/
  identity.go                   agent + authority principals
  policy.go                     scopes, trust modes, approvals policy
  enforce.go                    server-side authorization checks
tokens/                         token broker: resolve, refresh, revoke, redact
effects/
  protocol.go                   intents, attempts, states, reconciliation
  saga.go                       compensation registration + LIFO unwind
approvals/                      durable approval requests/decisions
audit/                          structured events, retention, redaction, OTel
transport/
  service.go                    canonical invocation service (HTTP handlers)
  openapi/                      generated OpenAPI + versioned routes
  mcp/                          central router + tool adapters
  cli/                          `miniapp` command tree (thin REST client)
testkit/                        fixtures, doubles, crash harness, policy modes
internal/
  store/                        Postgres access, tx boundaries, RLS context
  config/                       env-driven config incl. strict/warn/disabled
migrations/                     versioned SQL (app schema; DBOS/River via pins)
miniapps/<name>/                one reference miniapp per directory (see §16)
  miniapp.json  tools/  capabilities/  fixtures/  policy/  README.md
deploy/fly/                     fly.toml, process groups, health checks
docs/adr/                       ADR-001..010 formalized in Phase 0
```

Public API surface is `contract`, `registry`, `runtime`, `capability`,
`auth`, `tokens`, `effects`, `approvals`, `audit`, `transport/*`,
`testkit`. Everything else is internal. Miniapps import only public
packages; CI enforces this with an import checker (MiniMax lesson: declared
access must match code — see §6.2 rules and §18).

## 8. Public Go API proposals, with representative signatures

Illustrative only; Phase 0 freezes exact signatures. Error values are typed
(terminal vs retryable vs quarantined), never bare strings.

```go
// contract: manifest is the artifact of record.
type MiniappManifest struct { /* §9 fields */ }
func LoadManifest(path string) (*MiniappManifest, error)
func (m *MiniappManifest) Validate() error

// registry: versioned discovery.
func Register(m *MiniappManifest, impl MiniappImpl) error
func Resolve(id, version string) (*MiniappManifest, MiniappImpl, error)

// runtime: one invocation path for all transports.
type InvokeRequest struct {
  MiniappID, MiniappVersion string
  Tool string
  Input json.RawMessage
  IdempotencyKey string // stable operation ID
  Principal auth.Caller // agent + authority
  TrustMode auth.TrustMode
}
type RunHandle struct {
  RunID, MiniappID, Version, Status string
  PollPath, SignalPath, CancelPath string
}
func Invoke(ctx context.Context, req InvokeRequest) (RunHandle, error)
func Status(ctx context.Context, runID string) (RunStatus, error)
func Signal(ctx context.Context, runID string, sig Signal) error
func Cancel(ctx context.Context, runID string, reason string) error

// executor: substrate behind explicit interfaces.
type Executor interface {
  Submit(ctx context.Context, invocation Invocation) (RunHandle, error)
  Attach(ctx context.Context, runID string) (RunStatus, error)
}

// capability: approved operations only.
type Capability interface {
  Spec() CapabilitySpec
  Call(ctx context.Context, call CapabilityCall) (CapabilityResult, error)
}

// effects: the non-negotiable protocol.
type EffectIntent struct {
  BusinessOpID, AttemptID, IdempotencyKey string
  CapabilityID, Operation string
}
func ExecuteEffect(ctx context.Context, intent EffectIntent,
  do func(ctx context.Context) (ProviderOutcome, error)) (EffectRecord, error)

// approvals: bound to exact request hash.
func RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalID, error)
func Decide(ctx context.Context, id ApprovalID, d Decision) error
```

Consistency rule: Go structs generate JSON Schema; CI proves Go contracts,
published schemas, MCP tool schemas, OpenAPI, and docs agree. A drift test
fails the build on any mismatch.

## 9. Manifest, capability, identity, policy, run, and external-effect contracts

### 9.1 `miniapp.json` (manifest, per miniapp version)

Identity, version, description, examples, execution class per tool,
input/output schema refs, tool descriptors and names, substrate, capability
dependencies with connector versions, permission manifest and trust modes,
approval requirements, idempotency requirements, timeouts/retry/cancel,
external-effect declarations, saga declarations, audit requirements,
fixture refs, exact dependency manifest (SDK, registry, policy, fixture
hashes), compatibility/migration notes. Immutable once published.

### 9.2 `CapabilitySpec` (per approved operation)

Provider, operation, pinned API version, input/output schemas, auth method,
connection ref, OAuth scopes or API permissions, tenant/resource scope,
read/write and risk class, rate limits, timeout, retry semantics,
idempotency contract, error taxonomy, audit requirements, fixtures, and
compensation/reconciliation operation where applicable. Discovery yields
candidates; approval is a separate reviewed step.

### 9.3 Identity and policy

Caller carries agent principal (which harness client) and authority
principal (delegated user, approved tenant service identity, or authorized
trigger) plus tenant, miniapp version, connection, capability/resource
scopes, and trust mode. Trust mode (`dry-run`, `confirm`, `autonomous`) is
bound per miniapp version per tenant; the agent cannot weaken it. Every
consequential action passes a server-side policy check. MCP annotations and
elicitation are interaction aids, not enforcement.

### 9.4 Run envelope (uniform async contract)

```json
{
  "run_id": "run_01J9X8K2M4",
  "miniapp_id": "rpa.billing.chase_invoice",
  "version": "1.2.0",
  "status": "waiting_approval",
  "poll_tool": "rpa.runs.status",
  "signal_tool": "rpa.runs.signal",
  "cancel_tool": "rpa.runs.cancel"
}
```

No arbitrary active-handle expiry. Terminal-history retention is a separate
documented policy. Terminal errors use the shared taxonomy.

### 9.5 External-effect record

Business-op ID, attempt ID, provider idempotency key, intent row persisted
before dispatch, provider request/response IDs, and state transitions
`PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN`, with UNKNOWN
entering reconciliation toward `CONFIRMED`, `NOT_APPLIED`, quarantine, or
operator intervention. See §15 for the full machine.

## 10. Execution-class decision matrix and DBOS adoption policy

ADR-001 (`docs/DBOS.md`) is mandatory and enforced by the executor admission
check: a miniapp tool declares its class and substrate in `miniapp.json`,
and the framework rejects DBOS for tools that do not pass Gates 1–4.

| Class | Form | Substrate | Governing rule |
|---|---|---|---|
| A | Ordinary operation in one request | Plain Go handler + Postgres tx | No queue, no workflow engine |
| B | One-shot, independently retryable job | River | Transactional enqueue, unique keys, retries |
| C | Bounded multi-step operation | River by default; DBOS only when checkpointed recovery genuinely matters | Explicit progress + per-record idempotency; rubric decides promotion |
| D | Durable business workflow | DBOS | Durable waits, events, checkpoints, recovery |

Decision procedure per tool: classify (A/B/C/D) -> run Gates 1–4 and the
complexity score as a decision aid -> record REJECT/DEFER/PILOT/ADOPT with
rationale in the miniapp README -> admission check enforces the declared
substrate at registration. Async, scheduled, or multi-step alone never
justifies DBOS.

## 11. River and DBOS integration design

### 11.1 River (Class B, default Class C)

Transactional enqueue in the same Postgres transaction as the motivating
business write; unique jobs keyed on stable business IDs; retries with
backoff and dead-letter inspection; scheduling for delayed work. Class C
adds a framework-owned progress record (total/pending/succeeded/failed/
quarantined per batch), per-record idempotency keys
(`tenant:batch:{id}:row:{n}`), bounded worker concurrency, and a recovery
contract: resume scans progress rows, never replays completed records.
Phase 0 verifies the pinned River version's exact enqueue, uniqueness,
retry, scheduling, and failure-inspection APIs against its own source.

### 11.2 DBOS (Class D only)

Workflow and step discipline: deterministic workflow code; clocks,
randomness, DB reads, and HTTP only inside steps or transactions; stable
workflow IDs scoped by tenant and business operation
(`tenant:entity:kind:op`, never random); durable sleeps for waits; events
for external signals and approvals; queues for rate-limited fan-out;
explicit version-compatibility strategy per deployment (compatibility-first,
else drain or version routing, plus a deployment guard that refuses breaking
changes while old executions are pending). Payload discipline: IDs and
compact context only — no credentials, tokens, or full customer records.
Single-executor recovery on start is the open-source baseline; distributed
recovery needs explicit coordination (Conductor) on measured need only.
Phase 0 verifies the pinned DBOS Go SDK's workflows, steps, sleep, events,
queues, recovery, and transaction integration against its source and docs —
never by porting TypeScript/Python examples.

## 12. Neon connection and transaction design

Schema layout on one primary: `app` (domain, runs, effects, approvals,
audit, registry, RLS policies), `river` (job tables), `dbos` (system
tables). Environments are Neon branches with masked seed data.

| Workload | Initial approach |
|---|---|
| DBOS system pool | Neon direct connection, bounded pool; transaction-mode pooling is unsuitable (`LISTEN/NOTIFY`) |
| Application datasource | Direct or pooled per transaction/session requirements, decided in Phase 0 |
| River | Direct or pooled after compatibility testing |
| Migrations | Direct connection, same gate for app + DBOS + River schemas |

The plan distinguishes the DBOS system pool, the application datasource,
the exact shared pool handle, and the transaction-completion mechanism when
handles differ. Crash windows between business writes and DBOS checkpoints
are specified and tested; atomicity is proven, not assumed from a shared
URL. Connection budgets are computed across Fly Machines and horizontal
scaling, with pooled endpoints reserved for workloads that tolerate them.

## 13. MCP, HTTP/OpenAPI, and CLI architecture

One canonical invocation service; three thin adapters sharing execution,
authorization, and audit.

- **REST/OpenAPI (canonical):** versioned routes —
  `POST /v1/miniapps/{id}/invocations`, `GET .../invocations/{run}`,
  `POST .../invocations/{run}:signal`, `POST ...:cancel` — with generated,
  versioned OpenAPI. Works independently of MCP.
- **MCP router (primary agent surface):** one central router using the
  official Go MCP SDK (pinned, compatibility-tested). Stage 1 exposes
  generic `list/describe/invoke/status/signal/cancel` tools; stage 2 adds
  dynamically registered namespaced tools (`rpa.<domain>.<verb>_<noun>`)
  with tool-sprawl controls (per-tenant allowlists, manifest-bound exposure:
  the router lists only miniapps/operations authorized for the caller).
  Correctness and security never depend on optional task, elicitation, or
  other client-specific features; approvals are server-enforced.
- **CLI (developer/coding-agent surface):** `miniapp <id> <verb> --json
  --idempotency-key ...` with stable `--help`, JSON output, and exit codes
  as contract; a thin REST client, usable without MCP.

Harness validation covers at least one coding-agent harness and one
general-purpose runtime (see §19 Phase 6 and Appendix B for the
exact pair). Transports, elicitation, tasks, and approval behavior are
verified per harness, never assumed identical.

## 14. OAuth, tenant security, and authorization architecture

Dual boundary: agent-to-platform authorization (agent principal, tenant
membership, miniapp/version access, execution policy) is separate from
platform-to-SaaS capability authorization (approved operation, connection,
API scope, record access, risk level, idempotency). The agent's right to
invoke never implies the right to execute every SaaS operation.

- **Principals:** agent identity (API key or client credentials per
  harness client) plus authority (delegated user, approved tenant service
  identity, or authorized trigger). Handlers fail closed when authority
  cannot be resolved for a consequential tool.
- **Token broker:** resolves named bindings at step time; checks manifest
  scopes; handles OAuth authorization-code + PKCE, refresh rotation with
  reuse detection and absolute lifetime, revocation; RFC 8693 exchange only
  where the provider supports it. Credentials are envelope-encrypted
  server-side; workflow inputs, checkpoints, logs, and MCP outputs carry
  connection references, never secrets.
- **Tenant isolation:** Postgres RLS with an explicit session-variable
  context mechanism, least-privilege database roles, and `FORCE ROW LEVEL
  SECURITY` on tenant tables — defense in depth behind mandatory
  application authorization. Project-per-tenant reserved for
  compliance-heavy tiers.
- **Webhooks:** always-on receiver verifying HMAC signatures and
  timestamps, rejecting replays, deduplicating on provider event IDs via
  unique jobs, and delivering to tenants/workflows as durable events.
- **Non-negotiables:** no raw tokens to agents; no client-supplied tenant
  IDs trusted as identity; read/write permissions independent; high-risk
  operations need approval from an authorized principal (never the calling
  agent alone); every mutation has idempotency, audit, and reconciliation.

## 15. External-effect, idempotency, reconciliation, and Saga design

The framework-owned protocol every consequential outbound effect must use:

```text
PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN
                                              |
                                         RECONCILIATION
                                         /            \
                                  CONFIRMED        NOT_APPLIED
```

Rules: stable business-operation and attempt IDs; provider idempotency
keys wherever supported; intent persisted before dispatch; provider
request/response IDs recorded; timeouts-after-send land in UNKNOWN and
quarantine for reconciliation instead of blind retry; reconciliation
before retry or compensation whenever the outcome cannot be established.
Acceptance target: no duplicate confirmed business effects, no silently
lost operations, no blind retries of unknown outcomes — and completed DBOS
steps are never re-executed on recovery.

Saga compensation is explicit and idempotent: compensations are registered
before their forward actions, unwind LIFO on error, and never fire
automatically against UNKNOWN outcomes unless reconciliation or an
idempotency guarantee makes them safe. Irreversible effects declare that
plainly and route to operator intervention. Entity concurrency uses
Postgres uniqueness, row locking, or optimistic version checks; a DBOS
workflow ID supplies stable identity, not serialization.

## 16. Reference miniapp catalog and implementation sequence

Each reference is a real consumer of the public framework APIs with its
manifest, Go implementation, contracts, capability bindings, permissions,
substrate declaration, fixtures with independent expected results, unit/
integration/failure tests, local run docs, Fly config, example
MCP/REST/CLI invocations, and operational notes.

| Miniapp | Class | Purpose: what it proves | Sequence |
|---|---|---|---|
| `crm-read` | A | Typed, permission-checked reads; invocation + auth + audit path | First (vertical slice) |
| `lead-followup` | D | Durable waits, eligibility re-checks, events, cancel-safe recovery | First (vertical slice) |
| `contact-import` | C | River batch, per-record idempotency, progress, resume | Second |
| `notify-send` | B | One-shot job, idempotency key, delivery receipt | Early |
| `report-generate` | B | Scheduled one-shot render + delivery | Early |
| `data-sync` | C | Pull-merge-push, conflict policy | Early |
| `approval-flow` | D | Durable approval events, escalation, timeout | Early |
| `appointment-journey` | D | Multi-touch waits, quiet hours, reschedule-cancel | After first durable slice |
| `campaign-orchestration` | D | Rate-limited fan-out via queues | Gated on real-provider effect-protocol evidence |
| `agent-toolchain` | D | Resumable authorized agent/tool series | Later; needs separate scope justification |
| `crm-write` | A | Write authorization boundary | First wave, after write-auth tests pass |

The forcing set is `crm-read` + `lead-followup` + `contact-import`: they
establish core contracts across all three substrates without building the
whole catalog first. The remainder then evidences reusability.

## 17. Fly deployment and operations model

One Fly app, same region as the Neon primary (placement confirmed by a
measured latency benchmark, not anecdote):

| Process group | Responsibility | Scaling note |
|---|---|---|
| `gateway` | REST/MCP/CLI service, Class A, webhook receiver | Always on; MCP/HTTP never depend on worker availability |
| `river` | Class B/C workers | Scales with queue depth; may idle-stop if enqueue latency allows |
| `dbos` | Single dedicated DBOS executor | Always on until the wake test (§19 G6) proves otherwise |

Plus: health checks per group; graceful shutdown (drain in-flight steps,
no new claims); versioned migrations applied in the deployment gate;
explicit machine sizing validated by load tests; autoscaling and auto-stop
limits documented per group (no scale-to-zero for `dbos` by default).

Secrets and data: platform secrets in Fly secrets; tenant OAuth/API
credentials in the encrypted connection registry (envelope encryption with
a managed key service and a rotation model — recommended over Fly secrets
alone, which are static per-app config, not a tenant vault); audit
retention policy with redaction and access controls; backup/restore via
Neon branching plus tested restore drills. Operations: DBOS/River failure
inspection runbooks, alerts for stuck/quarantined/stranded work and
approval backlogs, execution-history views exposed as read-only miniapps,
and release/rollback procedures honoring the workflow-version deployment
guard.

## 18. Test architecture and acceptance criteria

Test pyramid: unit tests per package; contract tests (Go structs =
schemas = MCP/OpenAPI/docs, enforced by drift tests); integration tests
against real Postgres (disposable Neon branches); crash-boundary suite on
real Fly Machines; security probes; harness-compatibility tests on two
harness families. Critical persistence and workflow tests never rely on
mocks alone.

Framework test kit (`testkit/`): manifest/schema validation, fixture
runner, capability doubles, strict/warn policy modes (strict in CI and
production: missing idempotency keys, nondeterministic workflow code, and
undeclared capability use fail the build), idempotency tests, replay
checks, version-compatibility tests, tenant-isolation tests, concurrency
tests, and an import checker proving miniapps use only public APIs.

Acceptance gates (measurable, honest language):

- Invocation/identity: duplicate submits collapse to one run; invalid
  agent identity, tenant, miniapp/version, authority, grant, connection,
  or scope fails closed with typed terminal errors.
- Multi-tenant: cross-tenant read/write probes fail at both app and RLS
  layers; manipulated IDs, RLS bypass attempts, and over-privileged roles
  are covered; concurrent tenant operations stay isolated.
- Integrations: token expiry/refresh-failure/revocation, rate limits,
  outages, timeouts, duplicate/replayed/invalid webhooks, and unknown
  provider outcomes behave per §15 (quarantine, never blind retry).
- Reliability: double-submit, worker kill mid-step, kill between effect
  and checkpoint, River retry/terminal failure, connection interruption,
  DBOS recovery, cancel, approval-while-stopped, deploy-with-pending, and
  same-entity concurrency all meet: no unauthorized execution, no
  cross-tenant access, no lost committed enqueue, no duplicate
  confirmed effects, unknown outcomes quarantined, completed steps never
  re-executed, interrupted workflows resume under the documented model,
  and no active invocation is silently discarded.

## 19. Phased implementation roadmap

The prompt's Phase 0–6 structure is adopted unchanged: it proves the
framework with a vertical slice before expanding security, River support,
hardening, catalog, and deployment falsification. Each phase is specified
to become an independent coding-agent task.

### Phase 0 — Repository investigation and architecture contracts

- Objective: freeze contracts, pin the stack with evidence, resolve
  identity/transaction/threat-model questions, and run integration spikes.
- Packages: `contract/`, `internal/store`, `migrations/` (skeleton),
  `docs/adr/`, `go.mod` + CI skeleton.
- Dependencies: none (first).
- Tasks: formalize ADR-001..010; freeze `miniapp.json`, CapabilitySpec,
  identity/policy, run envelope, and effect-record contracts; pin every
  dependency version with verification date + source links; verify DBOS
  Go APIs, River APIs, MCP SDK transports, Neon direct/pooled behavior,
  and Fly process groups/autostart; spike DBOS+Neon commit atomicity and
  River `InsertTx` atomicity; define the threat model and RLS context
  mechanism; set up Neon branching and CI with strict policy mode.
- Artifacts: ADR files, contract specs with JSON Schema, version register
  (§20 filled), spike reports, threat model, CI running drift tests.
- Tests: contract validation tests, spike crash probes on a Neon branch.
- Acceptance: every §21 PoC question has a dated evidence note.
- Exit gate G0: contracts reviewed, versions pinned, spikes green, threat
  model signed off. Known risks: SDK APIs differ from source-doc claims;
  Neon/Fly behaviors need workarounds.

### Phase 1 — Minimal framework vertical slice

- Objective: prove the full path with one Class A and one Class D miniapp
  on test connectors.
- Packages: `registry/`, `runtime/`, `executor/classa`, `executor/dbos`,
  `auth/`, `effects/` (core), `approvals/` (core), `audit/`,
  `transport/*`, `testkit/` (core), `miniapps/crm-read`,
  `miniapps/lead-followup`.
- Dependencies: G0.
- Tasks: invocation service with validation, auth-before-dispatch, run
  handles, status/signal/cancel; Class A and DBOS adapters; generic MCP
  tools + REST + CLI; audit events; mock SaaS endpoints; both miniapps
  end to end.
- Artifacts: working slice callable via MCP, REST, and CLI with audit trail.
- Tests: contract drift, duplicate-submit collapse, worker-kill recovery,
  approval-while-stopped, tenant probes on mocks.
- Acceptance: agent invokes both tools; durable tool survives a mid-run
  machine stop and resumes from the correct step with no duplicate
  confirmed effects.
- Exit gate G1: vertical slice green on real Postgres. Risks: DBOS
  replay/versioning surprises; MCP transport quirks per harness.

### Phase 2 — Security and real SaaS integration

- Objective: production-grade identity, credentials, and one real SaaS
  sandbox plus one custom REST API.
- Packages: `capability/`, `tokens/`, `internal/store` (RLS),
  `transport/service` (webhooks), connector clients.
- Dependencies: G1.
- Tasks: connection registry with envelope encryption; OAuth
  code+PKCE, refresh rotation, revocation; fail-closed authorization;
  RLS policies + roles + context injection; webhook verification;
  per-harness auth; first real adapters.
- Artifacts: live SaaS + custom-REST capabilities with fixtures.
- Tests: full §18 identity/tenant/integration probes against the sandbox.
- Acceptance: no tool callable without resolvable authority; cross-tenant
  probes fail at app and RLS layers; zero secrets in payloads/logs.
- Exit gate G2: security probes green on real integrations. Risks:
  provider OAuth/scope quirks; RLS performance; vault operational cost.

### Phase 3 — River Class B/C support

- Objective: job adapter plus representative Class B and C miniapps.
- Packages: `executor/river`, `miniapps/notify-send`,
  `miniapps/contact-import`, `miniapps/report-generate` (optional order).
- Dependencies: G1 (core runtime), G2 for authorized sends.
- Tasks: River adapter with progress model, uniqueness, retries,
  scheduling, dead-letter inspection; batch resume contract; miniapps.
- Artifacts: working B/C miniapps with progress queries and runbooks.
- Tests: batch kill/resume, per-record idempotency, duplicate webhooks,
  retry/terminal-failure behavior.
- Acceptance: a Class C batch completes, retries correctly, resumes
  without replaying completed records, and is observable.
- Exit gate G3: B/C suite green on real Postgres. Risks: River API gaps;
  progress-table contention at scale.

### Phase 4 — Approval and external-effect hardening

- Objective: complete approvals, reconciliation, compensation, and
  unknown-outcome handling against real provider behavior.
- Packages: `approvals/`, `effects/` (full), `miniapps/approval-flow`
  (plus hardening of existing miniapps).
- Dependencies: G2, G3.
- Tasks: durable approval binding (version, invocation, operation,
  resource, intent hash, approver, expiry, decision); material-change
  invalidation; self-approval policy; reconciliation workers and
  quarantine UX/runbooks; saga LIFO unwind; namespaced MCP tools if
  harness interop is proven.
- Artifacts: approval-flow miniapp, reconciliation runbooks, quarantine
  dashboards as read-only views.
- Tests: approval expiry/replay/material-change, compensation success
  and failure, UNKNOWN injection (timeout-after-send doubles),
  provider-without-idempotency behavior.
- Acceptance: consequential writes pause across restarts until decided;
  unknown outcomes quarantine rather than repeat; compensations are
  idempotent and audited.
- Exit gate G4: hardening suite green on real provider sandbox. Risks:
  providers without status lookup force conservative quarantine; saga
  design complexity for irreversible effects.

### Phase 5 — Reference catalog expansion

- Objective: build the remaining catalog on the stabilized framework
  without changing public APIs casually.
- Packages: `miniapps/data-sync`, `miniapps/appointment-journey`,
  `miniapps/crm-write`, `miniapps/campaign-orchestration` (gated),
  `miniapps/agent-toolchain` (only with approved scope note).
- Dependencies: G4.
- Tasks: one miniapp at a time with manifest, fixtures, tests, docs,
  and Fly config; any framework gap needs an ADR note and a backwards-
  compatible API change with migration notes.
- Artifacts: complete catalog per §16 with per-miniapp READMEs.
- Tests: per-miniapp suites plus full regression of the catalog.
- Acceptance: every miniapp uses only public APIs (import check),
  passes fixtures, and documents its purpose-per-§16 mapping.
- Exit gate G5: catalog complete except explicitly gated/deferred
  entries. Risks: API churn pressure; catalog sprawl without rubric
  admission.

### Phase 6 — Deployment, harness compatibility, and falsification

- Objective: prove real Fly/Neon behavior, crash recovery, and two
  harness families end to end.
- Packages: `deploy/fly/`, runbooks, benchmark harnesses (cross-cutting).
- Dependencies: G5.
- Tasks: deploy the shared runtime; run the wake test (start multi-day
  workflow, stop worker, pass due time, reactivate, verify resume with
  no repeated completed effects); measure region latency and connection
  budgets; run crash-boundary, deploy-with-pending, and rollback drills;
  invoke one SaaS miniapp, one custom-REST miniapp, and one cross-system
  flow from two harness families via MCP plus REST and CLI.
- Artifacts: benchmark report, wake-test evidence, harness matrix,
  production runbooks, release/rollback procedures.
- Tests: full §18 acceptance on production-like topology.
- Acceptance: wake test passes under the documented recovery model;
  same contract works across MCP/REST/CLI and both harnesses;
  server-side approvals hold where harnesses lack elicitation.
- Exit gate G6: falsification report signed; GA readiness per §23.
  Risks: wake topology fails (keeps `dbos` always-on); harness gaps
  need adapter work; Neon/Fly limits need resizing.

## 20. Dependency/version/license register

All entries UNPINNED at plan time. Phase 0 pins exact versions with
verification dates and source links; `latest` is never a strategy. The
stack below restates the task's required constraints (prompt §4), not
independent recommendations.

| Role | Candidate source | Version | License | Verify in Phase 0 |
|---|---|---|---|---|
| Language toolchain | Go (version TBD) | UNPINNED | — | Minimum required by pinned DBOS/River/MCP SDKs |
| Durable workflows | DBOS Transact Go | UNPINNED | Verify edition terms (open-source vs Enterprise-gated sleep suspension, Conductor) | Workflows, steps, sleep, events, queues, recovery, tx integration APIs |
| Background jobs | River | UNPINNED | Verify | Enqueue, uniqueness, retries, scheduling, inspection APIs |
| Agent protocol | Official Go MCP SDK | UNPINNED | Verify | Transports, protocol revision, auth, elicitation support |
| Database | Neon Postgres | n/a (service) | Verify plan terms | Direct vs pooled behavior, branching, budgets |
| Compute | Fly.io Machines | n/a (service) | Verify plan terms | Process groups, health checks, autostart/stop semantics |
| Schema generation | TBD in Phase 0 | UNPINNED | Verify | Go structs to JSON Schema fidelity |
| OpenAPI tooling | TBD in Phase 0 | UNPINNED | Verify | Spec generation from the canonical service |
| Migrations | TBD in Phase 0 | UNPINNED | Verify | App + DBOS + River schema gating |
| Key management | TBD in Phase 0 | UNPINNED | Verify | Envelope encryption + rotation for tenant vault |

## 21. Risk register and unresolved technical questions

| # | Risk / question | Impact | Mitigation / resolution path |
|---|---|---|---|
| K1 | DBOS Go SDK is younger than TS/Python SDKs; APIs may drift from source-doc claims | High | Pin version; keep workflow logic thin over the platform layer; crash suite per upgrade |
| K2 | Sleep suspension / distributed recovery may be Enterprise-gated | High | Baseline open-source single executor, always on; re-evaluate Enterprise only on measured need |
| K3 | Wake topology unproven (stopped worker + due timer + no HTTP request) | High | Phase 6 wake test decides; external scheduler only if the baseline fails |
| K4 | DBOS/app atomicity depends on pool-handle configuration | High | Phase 0 spike with crash probes; document exact handles |
| K5 | Neon pooled endpoint incompatible with DBOS (`LISTEN/NOTIFY`) | High | Direct connection for DBOS; split modes per workload |
| K6 | MCP approval semantics underspecified across harnesses | High | Server-side enforcement always; host approval is an additional gate |
| K7 | Harness interop (transports, elicitation, tasks) differs per family | Medium | Verify two families in Phase 6; generic tools first |
| K8 | Providers lacking idempotency keys or status lookup | Medium | Conservative quarantine + reconciliation; gate `campaign-orchestration` on evidence |
| K9 | Agent over-permissioning / tool sprawl | Medium | Manifest-bound trust modes, per-tenant allowlists, rubric admission |
| K10 | Tenant vault operational cost (encryption, rotation, revocation) | Medium | Phase 2 compares options; envelope+KMS recommended |
| K11 | First SaaS choice constrains Phase 2 fixtures and OAuth paths | Medium | Appendix B, Q1 |
| K12 | Missing Review 12 / OpenLevel code means cited reliability lessons are unverified here | Low | Re-derive every boundary with local crash tests; cite nothing on faith |
| K13 | `docs/plan3_review.md` is empty — unknown whether a review is owed | Low | Resolve in Phase 0 (fill or remove) |

Proof-of-concept evidence still required before coding beyond Phase 0:
DBOS+Neon atomicity spike, River atomicity spike, wake/resume behavior
note, MCP round-trip on one harness, and the pinned version register.

## 22. Explicit out-of-scope items

AI miniapp generator; browser/desktop automation; UI generation; BPMN,
rules-engine, or custom-workflow-engine work; rulez engine development;
free-form LLM code execution; per-miniapp Fly apps by default; Neon
project-per-tenant except for compliance tiers; any connector beyond the
first SaaS + first custom REST until Phase 5; marketplace, billing, and
metering (productization later).

## 23. Final readiness assessment and implementation gates

- Ready after this plan: Phase 0 can start immediately (contracts, pins,
  spikes, threat model). No implementation phase is ready until G0 passes.
- Gates: G0 contracts+evidence, G1 vertical slice, G2 security+real SaaS,
  G3 River B/C, G4 approval/effect hardening, G5 catalog, G6 deployment
  falsification. Each gate needs its exit evidence; failures stop the
  line rather than being rescued by later phases.
- Contradictions resolved: R1–R15 in §3, with R2 (effect semantics), R3
  (wake topology), R4 (two principals), and R7 (Go library, not step DSL)
  as the load-bearing four.
- Blockers: none for Phase 0. Later phases block on their gate evidence
  (notably the wake test and real-provider effect evidence).

## Appendix A. Insights derived from MiniMax-Code-MiniApps (reference repo)

The MiniMax reference (`MiniMax-Code-MiniApps-main`) is a desktop
(Electron/Node/Chromium) plugin system, not our stack — but its governance
mechanics transfer directly:

1. **Host owns credentials, ports, auth, and publishing; the miniapp owns
   business logic.** Adopted as our token broker + capability proxy: agents
   and miniapps never touch raw secrets.
2. **Manifest as artifact of record** (`miniapp.json` + payload roots +
   lifecycle + declared access). Adopted: our `miniapp.json` declares
   tools, substrate, capabilities, policy, fixtures, and exact dependency
   pins; nothing undeclared is callable.
3. **Declared access must match code** (their `ACCESS_UNDECLARED` checker).
   Adopted as CI strict mode: undeclared capability use, missing
   idempotency keys, and nondeterministic workflow code fail the build.
4. **Fail closed at runtime when permission is revoked** (their permission
   switches). Adopted: revoked connections, scopes, or trust modes fail
   in-flight and future calls with typed errors; the app keeps running.
5. **Durable state written incrementally** because the host may stop the
   process without notice (their lifecycle/dispose discipline). Adopted:
   intent rows before dispatch, progress rows per record, atomic writes —
   never deferred to shutdown.
6. **Per-feature minimum-version matrix** (their client-version table).
   Adopted: manifest compatibility notes plus harness-capability detection
   for optional MCP features.
7. **Content-addressed payload hashing for reproducibility.** Adopted: the
   dependency manifest pins every input needed to reproduce a miniapp.

Not adopted: the presentation layer, page/bridge APIs, device access, and
desktop install mechanics — our surface is headless and agent-invoked.

## Appendix B. Open questions for the owner

1. Which SaaS should be the first real integration (needs OAuth2 + sandbox
   tenant; idempotency keys and status lookup strongly preferred)?
2. Which two harness families must pass Phase 6 (default proposal: one
   coding-agent harness and one general-purpose runtime)?
3. Is DBOS Enterprise (sleep suspension, Conductor) budgetarily in scope
   if the open-source wake test fails, or must the design stay
   open-source-only?
4. Do Fly.io and Neon organizations/regions already exist, or should
   Phase 0 create them (which region is preferred)?
5. Is `docs/plan3_review.md` (currently empty) owed a review, or may
   Phase 0 remove it?
6. May `agent-toolchain` be deferred past the initial catalog, or does a
   specific use case require it early?
