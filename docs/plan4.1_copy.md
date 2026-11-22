# Consolidated Implementation Plan v2 — Go Miniapp Framework, References, and Fly Deployment Platform

- **Status:** Planning only. No code, migrations, or deployments were changed to produce this plan.
- **Module:** `github.com/PithomLabs/miniapp` (single Go module, per instruction)
- **Supersedes:** `docs/plan4.md` (carried forward, not replaced as history)
- **New inputs:** `docs/prompt2.md` (truncated at §7.1 — see Appendix D) and the
  `sprites-go` checkout (`github.com/superfly/sprites-go`, v0.2.2 per CHANGELOG 2026-10-20, inspected 2026-11-22)
- **Primary blueprint:** `docs/plan3_z.md`, supplemented by `docs/plan3_gemini.md` then `docs/plan3_deepseek.md`
- **Plan path:** `docs/plan4.1.md` (this file)
- **External versions verified in this run:** none for our module — all dependency versions are
  UNPINNED and must be pinned in Phase 0 with verification dates and source links.
  The sprites-go v0.2.2 coverage notes are inspected evidence, not our pin.

## 1. Executive summary and final architecture recommendation

Build two deliverables in one repository and one Go module, plus the deployment
platform experience that operates them:

1. **A reusable Go framework library** owning contracts, registry, runtime,
   execution adapters (plain Go, River, DBOS), capabilities, identity and
   authorization, token brokering, the external-effect protocol, approvals,
   audit/observability, agent-facing adapters (MCP, REST/OpenAPI, CLI), the
   deployment-target abstraction (Machines, Sprites, hybrid), and a test kit.
2. **A catalog of reference miniapps** built only against the framework's
   public APIs, proving each execution class, the placement policy, and one
   hybrid DBOS-plus-Sprite flow.

Two independent dimensions govern every miniapp and must never be conflated
into one selector. **Dimension A (execution semantics):** Class A runs as
plain Go handlers, Class B/C on River, Class D on DBOS. **Dimension B
(compute placement):** trusted production code runs on Fly Machines;
agent-authored, generated, or untrusted code runs in isolated Fly Sprites;
durable workflows may orchestrate bounded Sprite tasks in a hybrid pattern
with Neon holding authoritative state. Sprites complement DBOS; a Sprite
checkpoint is a filesystem snapshot, never a workflow checkpoint.

The framework additionally provides a unified Design → Develop → Validate →
Publish → Deploy → Operate → Maintain lifecycle so developers never hand-drive
the Fly APIs for routine work, while consequential operational decisions stay
explicit. The first code still proves the thesis with a vertical slice
(`crm-read`, `lead-followup`, then `contact-import`) before catalog, Sprite
integration, and lifecycle tooling expand.

## 2. Product boundaries and explicit non-goals

The product is an API-first business-process automation platform. An agent in
a harness (Codex, Claude Code, Hermes, OpenClaw, and other compatible
environments) discovers, invokes, monitors, and interacts with miniapps; the
miniapp owns credentials, execution, recovery, audit, and its deployment and
operational configuration. The agent never holds provider credentials, never
owns authoritative durable state, and never bypasses server-side checks.

Explicit non-goals for the initial scope:

- General-purpose agent framework or harness replacement.
- Universal BPMN engine or a new workflow engine replacing DBOS.
- UI/dashboard generator as the product surface.
- Unrestricted shell or browser automation service; browser/desktop automation,
  where necessary, is an explicitly declared and separately secured capability,
  never an undocumented fallback.
- Arbitrary LLM-generated code deployment platform; unreviewed generated Go,
  shell, or JavaScript never executes inside the trusted production runtime
  (isolated Sprites are the venue for untrusted code).
- A rule engine every miniapp must use (rulez stays an optional later
  integration, out of initial scope).
- AI miniapp generator (deferred; see §22).

## 3. Source-document reconciliation table

Precedence applied: (1) explicit deliverables in `prompt2.md`, (2) ADR-001
policy in `docs/DBOS.md`, (3) `plan3_z.md`, (4) `plan3_gemini.md`,
(5) `plan3_deepseek.md`, (6) `rea.md` patterns, (7) repository evidence and
verified upstream behavior. New rows carry prompt2's five-way disposition;
R1–R15 are carried forward from plan4.md unchanged in substance.

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
| R12 | "Entity miniapp" as a fifth category | GPT §2 | Entity state is a composable pattern (Postgres concurrency + optional DBOS orchestration), not a separate class | ADR-001 and prompt2 §6 fix the A/B/C/D taxonomy |
| R13 | `agent-toolchain` in the initial catalog | plan3_z §9 | Assess separately with its own scope justification; default is later | Must not duplicate the harness reasoning loop |
| R14 | Async handle with short expiry | DeepSeek §5 envelope | Stable run IDs with no arbitrary active-handle expiry; terminal-history retention is a separate policy | Multi-day waits outlive short expirations |
| R15 | DBOS Conductor / multi-executor recovery assumed | GPT §4 | Start with one dedicated DBOS executor; add Conductor only on measured multi-process need | Keeps licensing and topology explicit |
| R16 | Single technology selector vs two independent dimensions | Earlier single-taxonomy framing vs prompt2 §6 | Adopt Dimension A (execution) × Dimension B (placement); never one mutually exclusive selector | A durable workflow can orchestrate an isolated task; conflation forbids valid hybrids. Disposition: accepted with modification |
| R17 | "Agent-invoked ⇒ Sprite" vs trust-profile placement | Naive reading of Sprites-for-agents vs prompt2 §2 | Placement follows the trust profile of the code, not the identity of the caller; trusted Go tools stay on Machines | A Sprite per tool call would destroy latency, cost, and auditability. Disposition: accepted with modification |
| R18 | Sprite checkpoint as workflow recovery vs filesystem snapshot | Checkpoint ambiguity vs prompt2 §3 | Sprite checkpoints restore files only; DBOS owns business recovery; Neon holds task identity and results | Conflating them strands workflows on filesystem state. Disposition: accepted with modification |
| R19 | CLI as thin invocation client vs full lifecycle CLI | plan4 §13 vs prompt2 §4 | One `miniapp` binary with two groups — `invoke` (thin REST client) and lifecycle (`init/dev/validate/test/preview/publish/deploy/status/...`); exact surface assessed in Phase 0 | One binary avoids tool sprawl; grouping avoids command sprawl. Disposition: accepted with modification |
| R20 | sprites-go for the full lifecycle vs typed REST client | prompt2 §1/§4 caution | Split by inspected evidence: SDK for lifecycle/exec/filesystem/services/checkpoints/network-policy; typed REST client for connectors/usage/logs/privilege gaps; both behind framework interfaces (Appendix C) | The checked-out SDK verifiably covers the first set and lacks the second. Disposition: requires proof of concept (re-verify at pinned version) |
| R21 | Sprite network default-open vs secure-by-default | Fly default vs framework policy | Framework applies a restrictive network policy before executing any agent-supplied code; default-deny templates per profile | Least privilege cannot wait for a later hardening phase. Disposition: accepted with modification |
| R22 | Sprite→Machine promotion as informal copy vs governed release | prompt2 §7 table row | Promotion produces an immutable versioned artifact with manifest, dependency lock, test results, and provenance, through a review gate | Production trust must be earned per release, not inherited from a workspace. Disposition: accepted unchanged |
| R23 | Hybrid DBOS+Sprite as new pattern | prompt2 §§2–3,7 | DBOS workflow orchestrates; Sprite runs the bounded task; Neon stores task identity/result; Sprite receives no unrestricted credentials | Gives agents isolation without surrendering durability or audit. Disposition: accepted unchanged |

## 4. Repository inventory and reuse/adapt/replace/retire assessment

The workspace at `/home/chaschel/Documents/go/miniapp` contains only
`docs/` (now 15 Markdown files including plan4.md, plan4.1.md, and prompt2.md). There is
still no `go.mod`, no Go source, no migrations, no Fly/Neon configuration,
no CI, no git checkout, and no ADRs, runbooks, or `AGENTS.md`.

| Item | Finding | Disposition |
|---|---|---|
| `docs/prompt.md` | Original task definition and corrections (§§1–15) | Reuse; superseded as task contract by prompt2 where they overlap |
| `docs/prompt2.md` | Updated task: two dimensions, Sprites, lifecycle DX; **truncated at §7.1** (446 lines, ends mid-list, no trailing newline) | Reuse §§1–7.1 as governing task contract; treat the missing remainder per Appendix D |
| `docs/plan4.md` | Prior consolidated plan (this file's baseline) | Reuse as baseline; revised here, retained as history |
| `docs/plan3_z.md` | Primary product/implementation architecture | Reuse as primary blueprint, subject to §3 corrections |
| `docs/plan3_gemini.md` | Now #4 in precedence: allocation and responsibility separation | Reuse as supplementary design (promoted per prompt2) |
| `docs/plan3_deepseek.md` | Now #5 in precedence: MCP/manifest/security supplements | Reuse as supplementary design |
| `docs/plan3_gpt.md` | Corroborating architecture | Reuse as cross-check; no separate authority |
| `docs/DBOS.md` | ADR-001 selective-adoption policy text | Reuse as mandatory adoption policy; formalize as ADR-001 file in Phase 0 |
| `docs/rea.md` | Restate-vs-DBOS comparison; Virtual Object, Saga, guardrail patterns | Reuse patterns only; Restate is not on the stack |
| `docs/plan.md`, `plan2.md`, reviews | Superseded UI-generator framing | Retire as product direction; retain durable lessons |
| `docs/plan3_review.md` | Empty file | Investigate in Phase 0 (fill or remove) |
| sprites-go checkout at `/home/chaschel/Documents/go/sprites-go` | v0.2.2 per CHANGELOG (2026-10-20), MIT, Go 1.26.0+; exec-first SDK plus lifecycle/filesystem/services/checkpoint/network-policy APIs; no usage/logs/connectors/privilege APIs (Appendix C) | Reuse as inspected evidence; adapt via wrapper interfaces; re-verify at pin — never a blind mandatory dependency |
| ADR-001 file, Review 12, OpenLevel code | Referenced but absent | Missing: treat cited claims as unverified |
| Go toolchain, modules, CI, Fly/Neon config | Absent | Greenfield: create in Phase 0/1 under `github.com/PithomLabs/miniapp` |

## 5. Key architecture decisions and ADR references

Phase 0 formalizes these as ADR files under `docs/adr/` (new directory).
ADR-001..010 are carried forward from plan4.md; ADR-011..015 are new.

| ID | Decision | Choice |
|---|---|---|
| ADR-001 | DBOS adoption policy | Adopt `docs/DBOS.md` verbatim: default reject; Class D earns DBOS through Gates 1–4 and the complexity score |
| ADR-002 | Module and repo layout | One repo, one Go module `github.com/PithomLabs/miniapp`; framework packages plus `miniapps/` references; split modules only if versioning later demands it |
| ADR-003 | Deployment unit | Shared platform runtime (one Fly app, process groups) by default; separately deployable miniapp binaries/modules; dedicated apps on justification |
| ADR-004 | Invocation architecture | One canonical invocation service; MCP router, REST/OpenAPI, and CLI are thin adapters on the same execution, authorization, and audit path |
| ADR-005 | Authorization model | Two principals (agent + authority), trust modes (`dry-run`, `confirm`, `autonomous`) bound per miniapp version per tenant, server-side enforcement |
| ADR-006 | External-effect semantics | Honest at-least-once with UNKNOWN quarantine and reconciliation; corrected acceptance language per R2 |
| ADR-007 | Connection strategy | DBOS on Neon direct connection initially; app and River modes decided per workload after compatibility testing; explicit connection budgets |
| ADR-008 | DBOS executor topology | Single dedicated executor, always available for waits; no scale-to-zero for the DBOS worker until the wake test passes; Conductor on measured need |
| ADR-009 | Manifest authority | `miniapp.json` is the versioned artifact of record: identity, tools, substrate, capabilities, policy, fixtures, deployment profile, exact dependency manifest |
| ADR-010 | Miniapp-toolchain scope | `agent-toolchain` and any generator work need separate scope justification; default is deferred |
| ADR-011 | Two-dimension classification | Dimension A (execution class) and Dimension B (compute placement) decided independently; admission checks enforce each against its own policy |
| ADR-012 | Compute-placement policy | Machines for trusted production; Sprites for untrusted/generated/interactive work; hybrid DBOS+Sprite for isolated tasks inside durable workflows (table in §10) |
| ADR-013 | Sprite SDK vs REST client | sprites-go (pinned, wrapped) for lifecycle/exec/filesystem/services/checkpoints/network-policy; framework-owned typed REST client for connectors/usage/logs/privilege gaps; both behind interfaces with fakes |
| ADR-014 | Lifecycle CLI | One `miniapp` binary, two command groups (invoke + lifecycle); exact surface frozen in Phase 0 from prompt2's proposals (§13) |
| ADR-015 | Hybrid execution pattern | DBOS orchestrates, Sprite executes the bounded task, Neon holds task identity/result, no unrestricted credentials enter the Sprite |

## 6. Target architecture diagrams

### 6.1 System context

```text
[Agent harnesses: Codex, Claude Code, Hermes, OpenClaw, ...]
        |  MCP (Streamable HTTP / stdio)   |  signed REST   |  CLI
        v                                  v                v
+------------------+  +------------------------------------------+  +------------------+
| Central MCP      |  | Canonical invocation service (Machine)   |  | CLI: invoke +    |
| router           |->| registry, auth, dispatch, run handles,   |<-| lifecycle groups |
| (namespaced      |  | audit, approvals                         |  |                  |
|  tools + generic |  +-----+--------+------------+------------+--+  +------------------+
|  discovery)      |        |        |            |            |
+------------------+        v        v            v            v
                     +--+------+ +--+-------+ +--+--------+ +--+---------+
                     | Class A | | River    | | DBOS      | | Capability |
                     | handler | | workers  | | executor  | | proxy +    |
                     | + tx    | | (B/C)    | | (D)       | | token      |
                     +--+------+ +--+-------+ +--+---+----+ | broker     |
                        |           |            |   |      +--+---------+
                        |           |            |   | Bounded Sprite task
                        |           |            |   v      (hybrid only)
                        |           |            | +--+----------+  +-----------+
                        |           |            | | Sprite task |->| Fly Sprite|
                        |           |            | | runner      |  | (isolated,|
                        |           |            | | (deploy/)   |  |  policy-  |
                        |           |            | +--+----------+  |  gated)   |
                        v           v            v    |             +-----------+
+---------------------------------------------------------------------+
| Neon Postgres (single primary, same region as Fly)                  |
|   app schema: domain, runs, effects, approvals, audit, registry,    |
|               sprite tasks, promotion records                       |
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
                       auth, effects, approvals, audit, deploy, testkit)
                             |
transport adapters --->  invocation service  --->  executor adapters
(MCP / REST / CLI)       (auth + dispatch)        (classa / river / dbos)
                             |                            |
                             v                            v
                      security/policy/persistence    capability clients
                                                      (no credential leaks
                                                       into workflows)

deploy/ target adapters ---> Fly Machines API / sprites-go (wrapped) /
                               framework REST client (gaps)
```

Rules (plan4 rules retained): the framework never imports reference
miniapps; transports own no business logic and no separate authorization;
provider clients never leak credentials into workflow code; DBOS, River,
and now the Sprite clients sit behind stable framework interfaces that keep
their real semantics visible. CLI lifecycle commands are thin over
`deploy/`, `runtime/`, and `testkit/` — they own no policy.

### 6.3 Invocation lifecycle

Submit (idempotency key = stable operation ID) -> validate manifest, resolve
trust mode, record invocation with agent identity -> return run handle
immediately -> execute on the declared substrate and placement -> agent polls
status or receives webhook -> terminal typed result plus execution summary ->
every transition audited.

### 6.4 Unified dev-to-ops lifecycle

Design (`init` from class + profile template) -> Develop (local or isolated
Sprite workspace, typed contracts, fixtures, mocks, live feedback) ->
Validate (`validate` + `test`: unit, contract, security, concurrency,
failure, crash-boundary; disposable Neon branches and Sprite previews) ->
Publish (immutable versioned artifact: manifest, dependency lock, test
results, provenance) -> Deploy (Machines for trusted services/workers,
Sprites for approved workspaces/isolated execution) -> Operate (unified
status, logs, traces, costs, history, checkpoints, approvals, retries,
recovery, rollback) -> Maintain (safe upgrades, workflow migration,
connection rotation, posture inspection, environment decommission, restore
to known-good). The framework hides repetitive infrastructure work, never
consequential operational decisions.

## 7. Framework library responsibilities and package/module structure

Proposed layout under the single module `github.com/PithomLabs/miniapp`
(all paths are proposals; nothing exists yet). New vs plan4.md: `deploy/`,
CLI command groups, promotion records.

```text
go.mod                          module github.com/PithomLabs/miniapp
contract/                       manifest + schemas + descriptors (public)
  manifest.go                   miniapp.json parse/validate (incl. §9.1 profile)
  schemas.go                    Go structs -> JSON Schema generation
  descriptors.go                tool descriptors, error taxonomy
registry/                       discovery, registration, version resolution
runtime/                        invocation lifecycle, dispatch, run handles
executor/
  executor.go                   substrate interfaces + admission (ADR-001 gate)
  classa/                       plain handler + tx adapter
  river/                        Class B/C adapter (progress, uniqueness)
  dbos/                         Class D adapter (workflows, events, recovery)
deploy/                         Dimension B: targets + lifecycle (public)
  profile.go                    deployment profiles (machine/sprite/hybrid/local)
  machine/                      Fly Machines target adapter
  sprite/                       Sprite target adapter: sprites-go wrapper (pinned)
  rest/                         framework typed REST client (gap APIs)
  tasks.go                      bounded Sprite-task primitive for hybrid flows
  promotion.go                  publish/promote/rollback records + gates
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
  cli/                          `miniapp` binary: invoke group + lifecycle group
testkit/                        fixtures, doubles, crash harness, policy modes
internal/
  store/                        Postgres access, tx boundaries, RLS context
  config/                       env-driven config incl. strict/warn/disabled
migrations/                     versioned SQL (app schema; DBOS/River via pins)
miniapps/<name>/                one reference miniapp per directory (see §16)
  miniapp.json  tools/  capabilities/  fixtures/  policy/  README.md
fly/                            fly.toml, process groups, health checks, profiles
docs/adr/                       ADR-001..015 formalized in Phase 0
```

Public API surface adds `deploy` to plan4's list. Miniapps import only
public packages; CI enforces this with an import checker.

## 8. Public Go API proposals, with representative signatures

Illustrative only; Phase 0 freezes exact signatures. plan4's runtime,
registry, capability, effects, and approvals sketches are retained; the
additions below cover Dimension B. Error values stay typed (terminal vs
retryable vs quarantined).

```go
// deploy: placement profiles and targets.
type Placement int // PlacementMachine, PlacementSprite, PlacementHybrid, PlacementLocal
type DeployProfile struct {
  Name string // e.g. "prod-machine", "agent-sandbox", "preview"
  Placement Placement
  SpritePolicy *SpritePolicy // network policy, resources, lifetime
  Promotion *PromotionPolicy // review gate, required checks
}
func LoadProfile(name string) (*DeployProfile, error)

// deploy/sprite: bounded task primitive (hybrid flows call this from DBOS steps).
type SpriteTask struct {
  TaskID string // stable, stored in Neon by the orchestrator
  Snapshot string // base snapshot, empty for blank
  Command []string
  Files map[string][]byte // staged inputs, size-capped
  NetworkPolicy NetworkPolicy // applied BEFORE execution (R21)
  Timeout time.Duration
  ResultRef string // where the result lands (Neon, not Sprite FS)
}
func RunTask(ctx context.Context, task SpriteTask) (TaskResult, error)
func TaskStatus(ctx context.Context, taskID string) (TaskStatus, error)
func CancelTask(ctx context.Context, taskID string, reason string) error

// deploy/sprite: workspace and checkpoint management.
func ProvisionWorkspace(ctx context.Context, spec WorkspaceSpec) (Workspace, error)
func DecommissionWorkspace(ctx context.Context, id string) error
func Snapshot(ctx context.Context, id, comment string) (CheckpointID, error)
func Restore(ctx context.Context, id string, cp CheckpointID) error

// deploy/promotion: governed Sprite -> Machine path.
func Publish(ctx context.Context, miniappID, version string) (Artifact, error)
func Promote(ctx context.Context, artifact Artifact, target string) error
func Rollback(ctx context.Context, miniappID, toVersion string) error
```

Consistency rule (retained): Go structs generate JSON Schema; CI drift
tests prove Go contracts, published schemas, MCP tool schemas, OpenAPI,
and docs agree.

## 9. Manifest, capability, identity, policy, run, and external-effect contracts

### 9.1 `miniapp.json` (manifest, per miniapp version)

plan4 fields retained (identity, version, description, examples, class per
tool, schemas, descriptors, substrate, capability dependencies, permissions,
trust modes, approvals, idempotency, timeouts/retry/cancel, effect and saga
declarations, audit, fixtures, exact dependency manifest, compatibility
notes), plus a **`deployment` section**: Dimension B placement per tool
(machine/sprite/hybrid/local), named deployment profile reference, Sprite
policy reference (network rules, resources, lifetime), and promotion policy
(review gate, required checks). Immutable once published.

### 9.2 `CapabilitySpec` (per approved operation)

Unchanged from plan4: provider, operation, pinned API version, schemas, auth,
connection ref, scopes, tenant/resource scope, read/write and risk class,
rate limits, timeout, retry, idempotency, error taxonomy, audit, fixtures,
compensation/reconciliation op. Discovery yields candidates; approval is a
separate reviewed step.

### 9.3 Identity and policy

Unchanged from plan4: agent principal plus authority principal, tenant,
scopes, trust mode bound per miniapp version per tenant, server-side
enforcement for every consequential action. Sprite workspaces and tasks
authenticate as scoped platform actors, never as users or tenants (§14).

### 9.4 Run envelope (uniform async contract)

Unchanged from plan4 (`run_id`, `miniapp_id`, `version`, `status`,
`poll/signal/cancel` tools). Sprite tasks reuse the same envelope shape
with `task_id`, so agents learn one async pattern across Machines and
Sprites. No arbitrary active-handle expiry; terminal-history retention is
separate policy.

### 9.5 External-effect record

Unchanged from plan4: `PENDING -> ATTEMPTING -> CONFIRMED | FAILED |
UNKNOWN` with reconciliation toward `CONFIRMED`/`NOT_APPLIED`, quarantine,
or operator intervention.

### 9.6 Sprite-task and promotion records (new)

A Sprite task declares: stable task ID, parent run/workflow ID, snapshot
base, staged inputs (capped), command, network policy (applied before
execution), timeout, expected result schema, and result destination in
Neon. Lifecycle: `STAGED -> POLICY_APPLIED -> RUNNING -> SUCCEEDED |
FAILED | TIMED_OUT | CANCELLED`, with terminal results copied to Neon
before the workspace may be decommissioned. A promotion record binds the
immutable artifact (manifest hash, dependency lock, test results,
provenance including source Sprite snapshot ID) to its review decision,
promoter identity, and target.

## 10. Execution-class decision matrix, DBOS adoption policy, and compute placement

### 10.1 Dimension A — execution semantics (unchanged)

ADR-001 (`docs/DBOS.md`) remains mandatory and enforced by the executor
admission check.

| Class | Form | Substrate | Governing rule |
|---|---|---|---|
| A | Ordinary operation in one request | Plain Go handler + Postgres tx | No queue, no workflow engine |
| B | One-shot, independently retryable job | River | Transactional enqueue, unique keys, retries |
| C | Bounded multi-step operation | River by default; DBOS only when checkpointed recovery genuinely matters | Explicit progress + per-record idempotency; rubric decides promotion |
| D | Durable business workflow | DBOS | Durable waits, events, checkpoints, recovery |

Async, scheduled, or multi-step alone never justifies DBOS.

### 10.2 Dimension B — compute placement (new)

Decided independently of Dimension A; the manifest admission check enforces
this table, and exceptions require a written justification.

| Workload | Execution substrate | Preferred Fly target |
|---|---|---|
| Trusted Class A API action | Go handler + PostgreSQL | Machine |
| Class B simple job | River | Machine |
| Class C bounded batch | River, selectively DBOS | Machine |
| Class D durable business workflow | DBOS | Machine |
| Trusted production miniapp API | Trusted Go runtime | Machine |
| Agent-authored or generated code | Isolated execution | Sprite |
| Persistent agent development workspace | Sprite filesystem/process environment | Sprite |
| Per-invocation isolated code task | Isolated execution | Sprite |
| Untrusted code evaluation and testing | Isolated execution | Sprite |
| Agent task inside a durable business workflow | DBOS orchestration + isolated execution | Hybrid |
| Agent-generated prototype promoted to production | Sprite in development, immutable artifact on release | Sprite → Machine |

Placement follows the trust profile of the code, never the identity of the
caller (R17). A persistent Sprite is not business-workflow persistence (R18).

## 11. River, DBOS, and Sprite-task integration design

### 11.1 River (Class B, default Class C — unchanged)

Transactional enqueue in the same transaction as the motivating write;
unique jobs on stable business IDs; retries with backoff and dead-letter
inspection; scheduling. Class C adds a framework-owned progress record,
per-record idempotency keys (`tenant:batch:{id}:row:{n}`), bounded worker
concurrency, and resume-without-replay. Phase 0 verifies the pinned River
APIs against its own source.

### 11.2 DBOS (Class D only — unchanged)

Deterministic workflow code; nondeterminism only in steps/transactions;
stable workflow IDs (`tenant:entity:kind:op`); durable sleeps; events for
signals/approvals; queues for rate-limited fan-out; per-deployment version
compatibility (compatibility-first, else drain or version routing, plus a
deployment guard); payload discipline (IDs and compact context, no
credentials). Single-executor recovery on start is the open-source
baseline. Phase 0 verifies the pinned DBOS Go SDK against its source and
docs, never via ported examples.

### 11.3 Sprite tasks and the hybrid pattern (new)

A hybrid flow keeps the DBOS workflow authoritative on a Machine. When it
needs isolated execution, a step calls `deploy.RunTask` with the §9.6 task
declaration; the runner provisions (or reuses) a policy-gated Sprite,
stages capped inputs, applies the network policy **before** execution,
runs the bounded command with a timeout, copies the terminal result to
Neon, and records the outcome. Failure modes are explicit: task timeout,
policy rejection, Sprite unavailability, and result-schema mismatch each
map to typed outcomes the workflow can retry (new task ID, same parent),
compensate, or escalate — a lost Sprite never strands the workflow because
Neon, not the Sprite filesystem, holds authoritative state. Sprite
checkpoints may seed reproducible task environments but never substitute
for workflow checkpoints. All Sprite clients sit behind framework
interfaces with fakes so hybrid flows are unit-testable without Fly.

## 12. Neon connection and transaction design

Unchanged from plan4: one primary with `app` (now also sprite tasks and
promotion records), `river`, and `dbos` schemas; Neon branches for
environments; DBOS on direct connection initially; app and River modes
decided per workload after testing; migrations on direct connections in one
gate; handle-level transaction integration proven by crash tests, never
assumed; explicit connection budgets across Machines.

## 13. MCP, HTTP/OpenAPI, and CLI architecture

One canonical invocation service; thin adapters sharing execution,
authorization, and audit. REST/OpenAPI (canonical, versioned) and the
central MCP router (generic tools first, namespaced tools once proven) are
unchanged from plan4.

The CLI becomes one `miniapp` binary with two groups (R19, ADR-014):

- **Invoke group** (thin REST client, plan4 behavior retained): run a
  miniapp tool, poll status, signal, cancel — with stable `--help`, JSON
  output, and exit codes as contract.
- **Lifecycle group** (new): assessed in Phase 0 from prompt2's proposals
  against a no-proliferation rule — every command must map to exactly one
  lifecycle stage in §6.4 and own no policy.

Proposed assessment (Phase 0 freezes this):

| Proposal | Assessment |
|---|---|
| `init --class --profile` | Keep: scaffold from class + deployment-profile template |
| `dev [--sprite]` | Keep: local dev server or isolated Sprite workspace |
| `validate` | Keep: manifest, schema, policy, and placement admission checks |
| `test` | Keep: thin over `testkit/` (unit → crash-boundary) |
| `preview` | Keep: disposable Neon branch + Sprite preview environment |
| `publish` | Keep: immutable versioned artifact + provenance |
| `deploy` | Keep: Machines services/workers, approved Sprite workspaces |
| `status`, `runs`, `logs` | Keep as one `status` family: `status`, `runs`, `logs` share one query path |
| `promote`, `rollback` | Keep: governed release moves with recorded decisions |
| `doctor` | Keep: infrastructure, dependency, and policy diagnostics |

Harness validation still covers one coding-agent harness and one
general-purpose runtime minimum, with transports and approval behavior
verified per harness.

## 14. OAuth, tenant security, and authorization architecture

plan4's dual boundary is retained: agent-to-platform authorization is
separate from platform-to-SaaS capability authorization; agent principal
plus authority principal with fail-closed handlers; token broker with
OAuth code+PKCE, refresh rotation, revocation, and RFC 8693 only where
supported; envelope-encrypted credentials with references (never secrets)
in payloads/logs; RLS as defense in depth behind mandatory app
authorization; verified, deduplicated webhooks. Sprite-specific additions:

- **Network policy before execution (R21):** outbound networking starts
  open, so the framework applies the profile's restrictive network policy
  (domain allow/deny rules) before any agent-supplied code runs. A task
  whose policy cannot be confirmed never starts. Default-deny templates
  ship per deployment profile.
- **Credential boundary:** a Sprite never receives unrestricted platform
  or SaaS credentials. Tasks get staged inputs and, only where the
  manifest allows, short-lived downscoped tokens for explicitly listed
  endpoints — resolved at dispatch, never tenant master credentials.
- **Workspace identity:** Sprites authenticate as scoped platform actors
  with per-task/per-workspace leases; leases expire, every action is
  audited, and decommissioning revokes access and wipes staged secrets.
- **Promotion review:** moving Sprite-built code to Machines requires the
  §9.6 promotion record: manifest hash, dependency lock, test results,
  provenance (source snapshot ID), reviewer identity, and decision. No
  silent promotion path exists.

## 15. External-effect, idempotency, reconciliation, and Saga design

Unchanged from plan4: framework-owned `PENDING -> ATTEMPTING ->
CONFIRMED | FAILED | UNKNOWN` protocol with reconciliation, quarantine for
unknown outcomes, and the corrected acceptance target; explicit idempotent
LIFO saga compensation that never auto-fires on UNKNOWN outcomes without a
safety basis; Postgres mechanisms (not workflow IDs) for entity
serialization. Sprite tasks that perform external effects go through the
same protocol — the effect intent is recorded by the orchestrator before
dispatch, so a Sprite-side send inherits idempotency and reconciliation
rather than inventing its own.

## 16. Reference miniapp catalog and implementation sequence

Each reference keeps plan4's full content bar (manifest, Go implementation,
contracts, bindings, permissions, substrate declaration, fixtures with
independent expected results, unit/integration/failure tests, local run
docs, Fly config, example invocations, operational notes) and gains a
deployment profile (§9.1). All references default to Machine placement;
hybrid is demonstrated once, deliberately.

| Miniapp | Class | Placement | Purpose: what it proves | Sequence |
|---|---|---|---|---|
| `crm-read` | A | Machine | Typed permission-checked reads; invocation + auth + audit path | First (vertical slice) |
| `lead-followup` | D | Machine | Durable waits, eligibility re-checks, events, cancel-safe recovery | First (vertical slice) |
| `contact-import` | C | Machine | River batch, per-record idempotency, progress, resume | Second |
| `notify-send` | B | Machine | One-shot job, idempotency key, delivery receipt | Early |
| `report-generate` | B | Machine | Scheduled one-shot render + delivery | Early |
| `data-sync` | C | Machine | Pull-merge-push, conflict policy | Early |
| `approval-flow` | D | Machine | Durable approval events, escalation, timeout | Early |
| `appointment-journey` | D | Machine | Multi-touch waits, quiet hours, reschedule-cancel | After first durable slice |
| `hybrid-task-demo` | D + task | Hybrid | DBOS workflow orchestrating one bounded Sprite task with Neon-held result | With Phase 5 Sprite work |
| `campaign-orchestration` | D | Machine | Rate-limited fan-out via queues | Gated on real-provider effect-protocol evidence |
| `agent-toolchain` | D | Assessed | Resumable authorized agent/tool series | Later; needs separate scope justification |
| `crm-write` | A | Machine | Write authorization boundary | First wave, after write-auth tests pass |

The forcing set stays `crm-read` + `lead-followup` + `contact-import`.
`hybrid-task-demo` is intentionally small: its job is to prove the §11.3
pattern (policy-before-exec, bounded task, Neon-held result, typed failure
mapping), not to be a second product surface.

## 17. Fly deployment and operations model

Two first-class targets under one lifecycle (§6.4), same region as the
Neon primary (confirmed by measured benchmark):

| Target | Process / unit | Responsibility | Scaling note |
|---|---|---|---|
| Machine | `gateway` | REST/MCP/CLI service, Class A, webhook receiver | Always on; never depends on worker availability |
| Machine | `river` | Class B/C workers | Scales with queue depth; may idle-stop if latency allows |
| Machine | `dbos` | Single dedicated DBOS executor | Always on until the wake test (§19 G6) proves otherwise |
| Sprite | Workspaces | Agent dev/prototype environments, previews | Per-workspace leases; decommissioned when idle past policy |
| Sprite | Tasks | Bounded isolated execution for hybrid flows | Ephemeral per task; result lands in Neon, then teardown |
| Either | Promotion | Governed Sprite → Machine release path | Review-gated; recorded per §9.6 |

Machine operations keep plan4's bar: health checks, graceful shutdown,
gated versioned migrations, validated sizing, documented auto-stop limits
(no scale-to-zero for `dbos` by default). Sprite operations add: lease
lifetimes and idle decommissioning, checkpoint/restore runbooks
(filesystem scope only), network-policy templates per profile, cost
tracking per workspace/task, and restore-to-known-good for dev
environments. Secrets keep the split: platform secrets in Fly secrets,
tenant credentials in the encrypted connection registry with envelope
encryption and rotation — Sprites hold neither beyond staged,
short-lived, downscoped task inputs. Unified operate view covers status,
logs, traces, costs, execution history, checkpoints, approvals, retries,
recovery, and rollback across both targets, with alerts for
stuck/quarantined/stranded work, approval backlogs, lease violations, and
policy-application failures.

## 18. Test architecture and acceptance criteria

plan4's pyramid is retained (unit → contract drift → real-Postgres
integration on Neon branches → crash-boundary on real Fly → security
probes → two-harness compatibility), with Sprite and lifecycle additions:

- **Placement admission tests:** manifests declaring the wrong target for
  their trust profile are rejected at `validate` with typed errors.
- **Policy-before-exec tests:** a Sprite task whose network policy cannot
  be confirmed never starts; outbound attempts outside the allowlist fail;
  the default-open Fly behavior is verified first, then locked down.
- **Checkpoint-semantics tests:** restore returns files only; workflow
  progress is unaffected and is proven to come from DBOS/Neon, not the
  restored filesystem.
- **Hybrid failure tests:** task timeout, policy rejection, Sprite loss,
  and result-schema mismatch each produce the typed outcome the workflow
  handles; no stranded workflow, no lost result, no credential residue.
- **Promotion tests:** promotion without manifest hash, lock, test
  results, provenance, or review decision is refused; rollback restores
  the prior version and its workflow compatibility.
- **Lifecycle CLI tests:** every `miniapp` command maps to one §6.4 stage,
  owns no policy, and round-trips through the same services as MCP/REST.

Acceptance language keeps plan4's four gates (invocation/identity,
multi-tenant, integrations, reliability) and adds: no agent-supplied code
runs on a Sprite without a confirmed restrictive policy; no unrestricted
credential ever enters a Sprite; no Sprite state is authoritative for any
business outcome.

## 19. Phased implementation roadmap

plan4's Phase 0–6 structure is retained and extended: the vertical slice
still comes first, while Sprite integration and lifecycle tooling enter at
the phases where their dependencies are proven. Each phase remains an
independent coding-agent task.

### Phase 0 — Repository investigation and architecture contracts

- Objective: freeze contracts (both dimensions), pin the stack with
  evidence, assess the CLI surface and Sprite clients, resolve
  identity/transaction/threat-model questions, and run integration spikes.
- Packages: `contract/`, `deploy/` (profiles + interfaces), `internal/store`,
  `migrations/` (skeleton), `docs/adr/`, `go.mod` + CI skeleton.
- Dependencies: none (first).
- Tasks: formalize ADR-001..015; freeze `miniapp.json` (incl. deployment
  section), CapabilitySpec, identity/policy, run envelope, effect record,
  Sprite-task and promotion records; pin every dependency version with
  verification date + source links; verify DBOS/River/MCP-SDK/Neon/Fly
  behavior; re-verify sprites-go coverage at the pinned version and scope
  the typed REST client for the gaps; spike DBOS+Neon and River atomicity
  plus one Sprite provision/policy/exec/teardown round-trip; define the
  threat model (incl. Sprite boundaries), RLS context mechanism, and the
  frozen CLI surface per §13; set up Neon branching and CI with strict
  policy mode.
- Artifacts: ADR files, contract specs with JSON Schema, filled version
  register (§20), spike reports, threat model, frozen CLI spec, CI running
  drift tests.
- Tests: contract validation tests, spike crash probes on a Neon branch,
  Sprite round-trip probe.
- Acceptance: every §21 PoC question has a dated evidence note.
- Exit gate G0: contracts reviewed, versions pinned, spikes green, CLI
  surface frozen, threat model signed off. Risks: SDK/API drift from the
  inspected checkout; Neon/Fly/Sprite behaviors need workarounds.

### Phase 1 — Minimal framework vertical slice (+ lifecycle bootstrap)

- Objective: prove the full path with one Class A and one Class D miniapp
  on test connectors, with the first lifecycle commands supporting them.
- Packages: `registry/`, `runtime/`, `executor/classa`, `executor/dbos`,
  `auth/`, `effects/` (core), `approvals/` (core), `audit/`,
  `transport/*` (incl. `init/validate/test` + invoke group), `testkit/`
  (core), `miniapps/crm-read`, `miniapps/lead-followup`.
- Dependencies: G0.
- Tasks: invocation service with validation, auth-before-dispatch, run
  handles, status/signal/cancel; Class A and DBOS adapters; generic MCP
  tools + REST + CLI; audit events; mock SaaS endpoints; both miniapps
  end to end; `init/dev/validate/test` sufficient to scaffold and check
  the slice itself (the slice dogfoods the lifecycle from day one).
- Artifacts: working slice callable via MCP, REST, and CLI with audit
  trail, scaffolded and validated by the lifecycle CLI.
- Tests: contract drift, duplicate-submit collapse, worker-kill recovery,
  approval-while-stopped, tenant probes on mocks, CLI round-trip tests.
- Acceptance: agent invokes both tools; durable tool survives a mid-run
  machine stop and resumes from the correct step with no duplicate
  confirmed effects.
- Exit gate G1: vertical slice green on real Postgres. Risks: DBOS
  replay/versioning surprises; MCP transport quirks per harness.

### Phase 2 — Security and real SaaS integration

- Objective: production-grade identity, credentials, and one real SaaS
  sandbox plus one custom REST API — including the Sprite credential
  boundary design.
- Packages: `capability/`, `tokens/`, `internal/store` (RLS),
  `transport/service` (webhooks), connector clients.
- Dependencies: G1.
- Tasks: connection registry with envelope encryption; OAuth code+PKCE,
  refresh rotation, revocation; fail-closed authorization; RLS policies +
  roles + context injection; webhook verification; per-harness auth;
  first real adapters; downscoped task-token design for future Sprite
  tasks (no Sprite execution in this phase).
- Artifacts: live SaaS + custom-REST capabilities with fixtures.
- Tests: full §18 identity/tenant/integration probes against the sandbox.
- Acceptance: no tool callable without resolvable authority; cross-tenant
  probes fail at app and RLS layers; zero secrets in payloads/logs.
- Exit gate G2: security probes green on real integrations. Risks:
  provider OAuth/scope quirks; RLS performance; vault operational cost.

### Phase 3 — River Class B/C support

- Objective: job adapter plus representative Class B and C miniapps.
  Unchanged from plan4.
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

### Phase 4 — Approval, external-effect hardening, preview, and publish

- Objective: complete approvals, reconciliation, compensation, and
  unknown-outcome handling — plus Sprite previews and immutable publish.
- Packages: `approvals/`, `effects/` (full), `deploy/` (preview, publish),
  `transport/cli` (`preview`, `publish`), `miniapps/approval-flow`.
- Dependencies: G2, G3.
- Tasks: durable approval binding with material-change invalidation and
  self-approval policy; reconciliation workers, quarantine views, saga
  LIFO unwind; namespaced MCP tools if interop is proven; `preview`
  (disposable Neon branch + Sprite preview) and `publish` (immutable
  artifact + provenance) commands.
- Artifacts: approval-flow miniapp, reconciliation runbooks, quarantine
  views, working preview/publish path.
- Tests: approval expiry/replay/material-change, compensation paths,
  UNKNOWN injection, provider-without-idempotency behavior, preview
  isolation and publish integrity.
- Acceptance: consequential writes pause across restarts until decided;
  unknown outcomes quarantine; compensations idempotent and audited;
  every publish is reproducible from its lock.
- Exit gate G4: hardening suite green on real provider sandbox. Risks:
  providers without status lookup force conservative quarantine; saga
  complexity for irreversible effects.

### Phase 5 — Reference catalog expansion + Sprite tasks and promotion

- Objective: build the remaining catalog on the stabilized framework and
  prove the hybrid pattern and governed promotion exactly once.
- Packages: `deploy/` (tasks, promotion, REST gap client), `transport/cli`
  (`deploy`, `promote`, `rollback`), `miniapps/data-sync`,
  `miniapps/appointment-journey`, `miniapps/crm-write`,
  `miniapps/hybrid-task-demo`, `miniapps/campaign-orchestration` (gated),
  `miniapps/agent-toolchain` (only with approved scope note).
- Dependencies: G4.
- Tasks: one miniapp at a time with manifest, fixtures, tests, docs, Fly
  config, and deployment profile; Sprite-task runner with
  policy-before-exec and Neon-held results; `hybrid-task-demo` proving
  §11.3; promotion pipeline with review gates; framework gaps need ADR
  notes and backwards-compatible API changes.
- Artifacts: complete catalog per §16, working hybrid demo, governed
  promotion path.
- Tests: per-miniapp suites, full catalog regression, hybrid failure
  matrix, promotion/rollback drills.
- Acceptance: every miniapp uses only public APIs (import check), passes
  fixtures, and maps to its §16 purpose; hybrid demo meets §18's Sprite
  gates.
- Exit gate G5: catalog complete except explicitly gated/deferred entries.
  Risks: API churn pressure; catalog sprawl without rubric admission;
  Sprite API drift at promotion time.

### Phase 6 — Deployment, harness compatibility, and falsification

- Objective: prove real Fly/Neon/Sprite behavior, crash recovery, and two
  harness families end to end, with the full operate surface.
- Packages: `fly/`, `transport/cli` (`status`, `runs`, `logs`,
  `doctor`), runbooks, benchmark harnesses (cross-cutting).
- Dependencies: G5.
- Tasks: deploy the shared runtime; run the DBOS wake test (start
  multi-day workflow, stop worker, pass due time, reactivate, verify
  resume with no repeated completed effects); verify Sprite default-open
  networking then confirm lockdown; measure region latency and connection
  budgets; run crash-boundary, deploy-with-pending, rollback, Sprite-loss,
  and checkpoint-semantics drills; invoke one SaaS miniapp, one
  custom-REST miniapp, and one cross-system flow from two harness
  families via MCP plus REST and CLI; complete `status/runs/logs/doctor`.
- Artifacts: benchmark report, wake-test evidence, Sprite security
  evidence, harness matrix, production runbooks, release/rollback
  procedures.
- Tests: full §18 acceptance on production-like topology.
- Acceptance: wake test passes under the documented recovery model; same
  contract works across MCP/REST/CLI and both harnesses; server-side
  approvals hold where harnesses lack elicitation; all §18 Sprite gates
  hold on real Sprites.
- Exit gate G6: falsification report signed; GA readiness per §23. Risks:
  wake topology fails (keeps `dbos` always-on); harness gaps need adapter
  work; Sprite quota/limits need resizing.

## 20. Dependency/version/license register

All entries UNPINNED at plan time for our module. Phase 0 pins exact
versions with verification dates and source links; `latest` is never a
strategy. The stack restates the tasks' required constraints, not
independent recommendations.

| Role | Candidate source | Version | License | Verify in Phase 0 |
|---|---|---|---|---|
| Language toolchain | Go (version TBD; note sprites-go requires 1.26.0+) | UNPINNED | — | Minimum required by pinned DBOS/River/MCP/Sprite SDKs |
| Durable workflows | DBOS Transact Go | UNPINNED | Verify edition terms (sleep suspension, Conductor) | Workflows, steps, sleep, events, queues, recovery, tx integration APIs |
| Background jobs | River | UNPINNED | Verify | Enqueue, uniqueness, retries, scheduling, inspection APIs |
| Agent protocol | Official Go MCP SDK | UNPINNED | Verify | Transports, protocol revision, auth, elicitation support |
| Sprite operations | sprites-go (inspected v0.2.2, MIT, 2026-10-20 — evidence, not the pin) | UNPINNED | MIT (checked-out copy; re-confirm at pin) | Re-verify Appendix C coverage at pinned version; wrap behind interfaces |
| Sprite gap APIs | Framework-owned typed REST client | n/a (our code) | n/a | Connectors/usage/logs/privilege endpoints against live API + docs |
| Database | Neon Postgres | n/a (service) | Verify plan terms | Direct vs pooled behavior, branching, budgets |
| Compute | Fly.io Machines + Sprites | n/a (services) | Verify plan terms | Process groups, autostart/stop, Sprite lifecycle/quotas/regions |
| Schema generation | TBD in Phase 0 | UNPINNED | Verify | Go structs to JSON Schema fidelity |
| OpenAPI tooling | TBD in Phase 0 | UNPINNED | Verify | Spec generation from the canonical service |
| Migrations | TBD in Phase 0 | UNPINNED | Verify | App + DBOS + River schema gating |
| Key management | TBD in Phase 0 | UNPINNED | Verify | Envelope encryption + rotation for tenant vault |

## 21. Risk register and unresolved technical questions

K1–K13 carried forward from plan4; K14–K20 are new.

| # | Risk / question | Impact | Mitigation / resolution path |
|---|---|---|---|
| K1 | DBOS Go SDK is younger than TS/Python SDKs; APIs may drift from source-doc claims | High | Pin version; keep workflow logic thin; crash suite per upgrade |
| K2 | Sleep suspension / distributed recovery may be Enterprise-gated | High | Baseline open-source single executor, always on; Enterprise only on measured need |
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
| K14 | Sprite checkpoint mistaken for workflow durability by future developers | High | §11.3 pattern + checkpoint-semantics tests + code review checklist item |
| K15 | Sprite network default-open window before policy applies | High | Policy-before-exec mandatory (R21); task never starts unconfirmed; verify-then-lock drill in Phase 6 |
| K16 | sprites-go coverage gaps (connectors, usage, logs, privilege/resource policies) | Medium | Framework REST client for gaps (ADR-013); re-verify at pinned version |
| K17 | Sprite API drift between inspected v0.2.2 and the pinned version | Medium | Wrap behind interfaces; coverage spike in Phase 0; examples/ dir as interop probes |
| K18 | CLI command proliferation beyond the assessed surface | Low | ADR-014 one-stage-per-command rule; new commands need ADR notes |
| K19 | Promotion integrity (unreviewed Sprite code reaching Machines) | High | §9.6 promotion record mandatory; no silent path; refusal tests |
| K20 | prompt2.md truncated at §7.1 — later task sections unknown | Medium | Appendix D completions, all labeled; owner confirms or supplies the remainder |

PoC evidence still required before coding beyond Phase 0: DBOS+Neon and
River atomicity spikes, Sprite provision/policy/exec/teardown round-trip,
MCP round-trip on one harness, REST gap-endpoint availability check, and
the pinned version register.

## 22. Explicit out-of-scope items

AI miniapp generator; unrestricted shell/browser automation; arbitrary
LLM-code deployment; UI generation; BPMN, rules-engine, or
custom-workflow-engine work; rulez engine development; free-form LLM code
execution in the trusted runtime; per-miniapp Fly apps by default; Neon
project-per-tenant except for compliance tiers; connectors beyond the
first SaaS + first custom REST until Phase 5; marketplace, billing, and
metering (productization later).

## 23. Final readiness assessment and implementation gates

- Ready after this plan: Phase 0 can start immediately (contracts for both
  dimensions, pins, spikes incl. Sprite round-trip, CLI freeze, threat
  model). No implementation phase is ready until G0 passes.
- Gates: G0 contracts+evidence, G1 vertical slice (+lifecycle bootstrap),
  G2 security+real SaaS, G3 River B/C, G4 hardening+preview/publish, G5
  catalog+Sprite tasks/promotion, G6 deployment falsification. Each gate
  needs its exit evidence; failures stop the line.
- Contradictions resolved: R1–R23 in §3, with R2 (effect semantics), R3
  (wake topology), R4 (two principals), R7 (Go library), R16 (two
  dimensions), R18 (checkpoint scope), and R20 (SDK/REST split) as the
  load-bearing seven.
- Blockers: none for Phase 0. Later phases block on gate evidence —
  notably the wake test, real-provider effect evidence, and Sprite
  policy-before-exec evidence.

## Appendix A. Insights derived from MiniMax-Code-MiniApps (reference repo)

Carried forward from plan4.md: (1) host owns credentials/ports/auth/
publishing, miniapp owns business logic → token broker + capability proxy;
(2) manifest as artifact of record → `miniapp.json` declares everything
callable; (3) declared access must match code → CI strict mode;
(4) fail closed on revocation → typed errors, app keeps running;
(5) incremental durable writes (host may stop the process without notice)
→ intent/progress rows, never deferred to shutdown; (6) per-feature
minimum-version matrix → manifest compatibility notes + harness detection;
(7) content-addressed payload hashing → dependency manifest pins. Not
adopted: presentation layer, page/bridge APIs, device access, desktop
install mechanics.

## Appendix B. Open questions for the owner

1. Which SaaS should be the first real integration (needs OAuth2 + sandbox
   tenant; idempotency keys and status lookup strongly preferred)?
2. Which two harness families must pass Phase 6 (default: one coding-agent
   harness and one general-purpose runtime)?
3. Is DBOS Enterprise (sleep suspension, Conductor) budgetarily in scope
   if the open-source wake test fails, or must the design stay
   open-source-only?
4. Do Fly.io and Neon organizations/regions already exist, or should
   Phase 0 create them (which region; is there a Sprite org with quota)?
5. Is `docs/plan3_review.md` (currently empty) owed a review, or may
   Phase 0 remove it?
6. May `agent-toolchain` be deferred past the initial catalog, or does a
   specific use case require it early?
7. (New) Is the §13 CLI assessment (one binary, two groups, proposed
   keep-list) approved as the Phase 0 freezing basis, or should any
   command be cut/renamed now?
8. (New) prompt2.md ends at §7.1 — is there a remainder to supply, or
   should the Appendix D completions stand?

## Appendix C. sprites-go inspection evidence (2026-11-22)

Checkout: `/home/chaschel/Documents/go/sprites-go`, v0.2.2 per CHANGELOG
(2026-10-20), MIT license, `go 1.26.0` in go.mod. README focus is
exec-first (`sprite.Cmd` mirrors `exec.Cmd`), but the inspected code
covers substantially more than the README claims.

| Area | Inspected coverage (file refs) | Verdict |
|---|---|---|
| Lifecycle | `CreateSprite/WithOrg`, `GetSprite/WithOrg`, `ListSprites`, `ListAllSprites(+WithOrg/Result)`, `DeleteSprite`/`DestroySprite`, `UpdateSprite`, `UpgradeSprite`, `UpdateURLSettings` (`management.go`); `Sprite()`/`SpriteWithOrg()` handles (`client.go`) | Covered — README "future functionality" labels for Create/List are stale |
| Exec | `Command/CommandContext`, `Run/Start/Wait/Output/CombinedOutput`, pipes, TTY + resize, `Signal`, `ExitCode` (`exec.go`); `ListSessions`, `AttachSession*`, `SignalSession` (`session.go`, `client.go`) | Covered |
| Filesystem | `FS` interface: open/stat/read/write/mkdir/remove/rename/copy/chmod + context variants (`filesystem.go`); `Fs*Control` ops (`filesystem_control.go`) | Covered |
| Services | List/Get/Create/Delete/Start/Stop/Signal + log streams (`services.go`); no explicit Restart (stop+start composes it) | Covered except Restart-by-name |
| Checkpoints | Create(+comment)/List(+options)/Get/Restore + streams (`checkpoint.go`) | Covered |
| Network policy | Get/Update, domain allow/deny rules (`policy.go`, `types.go`) | Covered; privilege/resource policies absent |
| Proxy | `ProxySocket/Port/Ports`, `ProxyManager` (`proxy.go`) | Covered (useful for `dev` port-forwarding) |
| Tokens | `CreateToken` FlyV1-macaroon exchange (`client.go`) | Covered |
| Errors | `APIError` with rate-limit helpers, request IDs (`errors.go`) | Covered |
| Usage / logs / connectors | No matching APIs found in non-test sources | Gap → framework REST client (R20) |
| Examples | 19 runnable endpoint examples (`examples/`), incl. `policy_get/set`, checkpoint, service, session, sprite CRUD | Reusable as interop probes in Phase 0 |

Re-verify every row at the pinned version in Phase 0; the checkout may
lag upstream.

## Appendix D. prompt2.md truncation note and labeled completions

`docs/prompt2.md` ends abruptly at §7.1 (line 446, mid-list, no trailing
newline). Everything from §7.2 onward — presumably Sprite placement
details, hybrid rules, lifecycle stage requirements, CLI/package/test
requirements, and any revised phase plan — is missing. This plan completes
those gaps as follows (all labeled architect's completion, needing owner
confirmation per Appendix B Q8):

- D1: Lifecycle stage requirements inferred from prompt2 §4's seven-stage
  list; encoded as §6.4 with CLI mappings in §13.
- D2: Hybrid failure semantics (§11.3) and Sprite security rules (§14)
  inferred from prompt2 §§2–3 plus the inspected SDK's actual primitives.
- D3: Phase integration (§19) keeps plan4's Phase 0–6 skeleton and folds
  Sprite/lifecycle work into the phases owning their dependencies, since
  no revised phase plan survives in the truncated file.
- D4: Precedence (§3) follows prompt2's PART I ordering (gemini above
  deepseek) as the last complete instruction on the subject.
