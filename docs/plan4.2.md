# Consolidated Implementation Plan v3 — Agent-Invokable Go Miniapp Framework and Fly Runtime

- **Status:** Revised architecture baseline following selective consolidation of `plan4.1_adv_review2.md` on 2026-11-22. This revision supersedes the frozen `plan4.1.md`; it does not erase its review history.
- **Revision:** A1–A16 retained. Net-valid findings from F01–F24 are incorporated into the architecture, roadmap, and proof gates; Appendix F records every disposition. No new ADR numbering is introduced for these refinements.
- **Module:** `github.com/PithomLabs/miniapp` (single Go module)
- **Supersedes:** `docs/plan4.1.md` as the active implementation baseline; preserve plan4.1 as historical record.
- **New review input:** `docs/plan4.1_adv_review2.md` (adversarial review dated 2026-11-22)
- **External verification:** DBOS Go, River, Neon, OWASP SSRF guidance, and Fly deployment/egress documentation checked 2026-11-22; exact dependency pins still belong in Phase 0.

## 1. Executive summary and final architecture recommendation

Build a reusable Go framework and a deliberately small catalog of business-outcome miniapps. The agent is the caller and planner; the miniapp is the permissioned executor. The platform owns identity checks, credentials, durable state, effect reconciliation, approvals, and audit.

The baseline remains API-first and uses Go, PostgreSQL/Neon, DBOS for Class D durable workflows, River for Class B jobs, and Fly Machines for trusted production workloads. It keeps two independent dimensions: execution semantics (A/B/C/D) and compute placement (Machine/Sprite/hybrid/local). They must not be collapsed into one selector.

**Core delivery is the Machine-based vertical slice.** Phase 1 proves synchronous Class A invocation and a Class D durable workflow using test connectors. The broader Sprite, hybrid-task, promotion, and full lifecycle capabilities remain in the roadmap because the owner previously approved A1–A16; they are staged behind their own prerequisites and must not block the core vertical slice merely because a later extension spike is incomplete.

The new review identifies real reliability and security gaps. This revision incorporates the valid findings: Class D DBOS application/version isolation; SSRF-safe outbound REST; safe ordering of external writes; synchronous Class A semantics; explicit approval and entity-serialization contracts; blast-radius controls; untrusted-result handling; OAuth permission intersection; Neon restore/reconciliation; Fly deploy-overlap controls; evidence-based gates; and PII erasure. It does **not** adopt the review's blanket removal of Sprites, its assumption that every Class C workload belongs on DBOS, or a wholesale replacement of the accepted lifecycle roadmap.

The governing discipline remains: prove the critical invariant at the boundary where it can fail, record the evidence, and do not delay unrelated phases for optional capabilities.

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

Sprites have two distinct uses with different scope (A8): preview and
development workspaces are in scope for the developer lifecycle (Phase 4),
while untrusted or generated code execution via bounded tasks is
infrastructure built in Phase 5+ behind the A1–A6 security gates. The
generator itself stays out of scope either way.

## 3. Source-document reconciliation table

Precedence applied: (1) explicit deliverables in `prompt2.md`, (2) ADR-001
policy in `docs/DBOS.md`, (3) `plan3_z.md`, (4) `plan3_gemini.md`,
(5) `plan3_deepseek.md`, (6) `rea.md` patterns, (7) repository evidence and
verified upstream behavior. New rows carry prompt2's five-way disposition;
R1–R15 are carried forward from plan4.md, with explicit refinements to R1 (Class C substrate choice) and R5 (transaction-local tenant context) based on review2 evidence.

| # | Disagreement | Sources | Resolution | Rationale |
|---|---|---|---|---|
| R1 | River chain as durable workflow abstraction vs explicit progress model | DeepSeek §4 vs prompt §2A | River remains the Class B default. Class C requires a per-workload comparison: River plus explicit progress/idempotency versus DBOS queues/partitioning; record the decision and evidence rather than hard-coding one default | DBOS Go queues provide queue-wide and per-partition concurrency/rate controls; River global concurrency limits are a Pro feature, so the trade-off is workload- and license-sensitive |
| R2 | "Zero duplicate external effects" as acceptance language | plan3_z §§3,8.7 vs prompt §2B | "No duplicate confirmed business effects, no silently lost operations, no blind retries of unknown outcomes" | A step can execute twice across the provider-complete/checkpoint-missing crash window |
| R3 | Fly Machines stop during waits and resume later | plan3_z §4.4/§10, DeepSeek §8 vs prompt §2C | Baseline: one always-available executor per active DBOS application/workflow family; prove wake topology before any scale-to-zero claim | Sleep suspension may be Enterprise-gated; Fly autostart is request-driven, not timer-driven |
| R4 | Identity: user-delegated token mandatory for every write vs agent principal | DeepSeek §7 vs plan3_z §7.4 | Two-principal model: agent principal plus authority principal (delegated user, approved service identity, or authorized trigger) | Supports interactive and unattended automation without granting agents admin authority |
| R5 | RLS via `auth.user_id()` | DeepSeek §7 and review2 F13 | Explicit transaction-local tenant context set inside the same transaction as tenant queries; use RLS as defense in depth behind mandatory app authorization | Neon transaction pooling does not preserve session `SET` state; Neon documents `SET LOCAL` inside a transaction as the safe scoped pattern, which must still be proven with the selected driver/pool |
| R6 | One Fly app per miniapp vs shared app with process groups | plan3_z §10 vs DeepSeek §8 | Shared platform runtime by default; dedicated apps/groups only on proven isolation, scaling, or customer need | Lower operational cost; preserves independent scaling where justified |
| R7 | Declarative step-DSL runtime vs Go miniapps on a framework library | GPT §3C/§6 vs prompt §7 | Behavior is Go code against public framework APIs; `miniapp.json` is the contract/permission manifest, not a full behavior DSL | Matches the "reusable Go framework library" deliverable; a trusted step DSL is a later, separately-scoped product |
| R8 | Generic MCP tools vs namespaced per-miniapp tools | GPT §3A vs DeepSeek §5 | Staged: generic discovery/invoke/status tools first, namespaced tools (`rpa.<domain>.<verb>_<noun>`) once harness interop is proven | Interoperability first, ergonomics second; router stays central either way |
| R9 | "Shared Neon URL" implying atomic DBOS/app commits | plan3_z §5.1, DeepSeek §8 | Verify handle-level datasource integration (`RunAsTransaction` or version equivalent); prove crash windows with tests | Atomicity depends on pool-handle configuration, not on sharing a URL |
| R10 | Anecdotal ~150ms cross-region latency as universal fact | DeepSeek §8 | Measure; place Fly region with the Neon primary by benchmark, not anecdote | Latency claims must be measured per deployment |
| R11 | rulez/RuleSpec as product foundation | plan.md/plan2.md vs pivot docs | rulez is an optional deterministic decision component, out of initial scope | Pivot redefined the product as agent-invoked execution, not generated UIs |
| R12 | "Entity miniapp" as a fifth category | GPT §2 | Entity state is a composable pattern (Postgres concurrency + optional DBOS orchestration), not a separate class | ADR-001 and prompt2 §6 fix the A/B/C/D taxonomy |
| R13 | `agent-toolchain` in the initial catalog | plan3_z §9 | Assess separately with its own scope justification; default is later | Must not duplicate the harness reasoning loop |
| R14 | Async handle with short expiry | DeepSeek §5 envelope | Stable run IDs with no arbitrary active-handle expiry; terminal-history retention is a separate policy | Multi-day waits outlive short expirations |
| R15 | DBOS Conductor / multi-executor recovery assumed | GPT §4 | Start with one active executor per DBOS application/family; add capacity or Conductor only on measured multi-process need | Keeps licensing and topology explicit |
| R16 | Single technology selector vs two independent dimensions | Earlier single-taxonomy framing vs prompt2 §6 | Adopt Dimension A (execution) × Dimension B (placement); never one mutually exclusive selector | A durable workflow can orchestrate an isolated task; conflation forbids valid hybrids. Disposition: accepted with modification |
| R17 | "Agent-invoked ⇒ Sprite" vs trust-profile placement | Naive reading of Sprites-for-agents vs prompt2 §2 | Placement follows the trust profile of the code, not the identity of the caller; trusted Go tools stay on Machines | A Sprite per tool call would destroy latency, cost, and auditability. Disposition: accepted with modification |
| R18 | Sprite checkpoint as workflow recovery vs filesystem snapshot | Checkpoint ambiguity vs prompt2 §3 | Sprite checkpoints restore files only; DBOS owns business recovery; Neon holds task identity and results | Conflating them strands workflows on filesystem state. Disposition: accepted with modification |
| R19 | CLI as thin invocation client vs full lifecycle CLI | plan4 §13 vs prompt2 §4 | One `miniapp` binary with two groups — `invoke` (thin REST client) and lifecycle (`init/dev/validate/test/preview/publish/deploy/status/...`); exact surface assessed in Phase 0 | One binary avoids tool sprawl; grouping avoids command sprawl. Disposition: accepted with modification |
| R20 | sprites-go for the full lifecycle vs typed REST client | prompt2 §1/§4 caution | Split by inspected evidence: SDK for lifecycle/exec/filesystem/services/checkpoints/network-policy; typed REST client for connectors/usage/logs/privilege gaps; both behind framework interfaces (Appendix C) | The checked-out SDK verifiably covers the first set and lacks the second. Disposition: requires proof of concept (re-verify at pinned version) |
| R21 | Sprite network default-open vs secure-by-default | Fly default vs framework policy | Framework applies a restrictive network policy before executing any agent-supplied code; default-deny templates per profile | Least privilege cannot wait for a later hardening phase. Disposition: accepted with modification |
| R22 | Sprite→Machine promotion as informal copy vs governed release | prompt2 §7 table row | Promotion produces an immutable versioned artifact with manifest, dependency lock, test results, and provenance, through a review gate | Production trust must be earned per release, not inherited from a workspace. Disposition: accepted unchanged |
| R23 | Hybrid DBOS+Sprite as new pattern | prompt2 §§2–3,7 | DBOS workflow orchestrates; Sprite runs the bounded task; Neon stores task identity/result; Sprite receives no unrestricted credentials | Gives agents isolation without surrendering durability or audit. Disposition: accepted unchanged |

Post-review amendments A1–A14 from `docs/plan4.1_consolidated_review.md`,
plus owner-directed A15/A16, are incorporated throughout this revision
(contracts, phases, gates, and tests); Appendix E traces each amendment
to its plan-text locations.

## 4. Repository inventory and reuse/adapt/replace/retire assessment

The workspace at `/home/chaschel/Documents/go/miniapp` contains only
`docs/` (18 Markdown files including the plan4.1 review set). There is
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
| `docs/plan4.1_consolidated_review.md` | Binding disposition: Phase 0 approved, roadmap conditionally approved subject to A1–A14 | Reuse as the governing review record; amendments incorporated here, reasoning preserved there |
| `docs/plan4.1_adv_review.md`, `docs/plan4.1_review.md` | Individual adversarial reviews (chatgpt/deepseek/gemini/Z) behind the consolidated disposition | Retain as historical review records; not separate requirement sets |
| ADR-001 file, Review 12, OpenLevel code | Referenced but absent | Missing: treat cited claims as unverified |
| Go toolchain, modules, CI, Fly/Neon config | Absent | Greenfield: create in Phase 0/1 under `github.com/PithomLabs/miniapp` |

## 5. Key architecture decisions and ADR references

Phase 0 formalizes these as ADR files under `docs/adr/` (new directory).
ADR-001..010 are carried forward from plan4.md; ADR-011..015 were added in plan4.1; ADR-016/017 were appended for A15/A16. The review2 refinements below update the decisions and implementation controls without introducing another ADR-numbering cycle. ADR and risk-register numbering stays provisional until the Phase 0 ADR files exist; renumbering then requires updating every cross-reference.

| ID | Decision | Choice |
|---|---|---|
| ADR-001 | DBOS adoption policy | Port the governing principles of `docs/DBOS.md` to Go/PostgreSQL/River; default reject; Class C/D adoption requires a workload-specific decision record and human review of business consequence and net complexity |
| ADR-002 | Module and repo layout | One repo, one Go module `github.com/PithomLabs/miniapp`; framework packages plus `miniapps/` references; split modules only if versioning later demands it |
| ADR-003 | Deployment unit | Shared gateway and common Go runtime by default. Class D workflows are isolated by DBOS `AppName` and application-version family; worker process groups/releases must preserve old-version recovery. Reuse the same compiled Go image by default; separate Fly apps or binaries only where release/drain evidence requires them |
| ADR-004 | Invocation architecture | One canonical invocation service; MCP router, REST/OpenAPI, and CLI are thin adapters on the same execution, authorization, and audit path |
| ADR-005 | Authorization model | Two principals (agent + authority), trust modes (`dry-run`, `confirm`, `autonomous`) bound per miniapp version per tenant, server-side enforcement; approval is an explicit policy gate, not a new execution class |
| ADR-006 | External-effect semantics | Honest at-least-once with UNKNOWN quarantine and reconciliation; corrected acceptance language per R2 |
| ADR-007 | Connection strategy | DBOS on Neon direct connection initially; app and River modes decided per workload after compatibility testing; explicit connection budgets |
| ADR-008 | DBOS executor topology | One active executor per DBOS application/workflow family in the initial topology; always available for waits; no scale-to-zero until the wake test passes; Conductor or additional executors only on measured need |
| ADR-009 | Manifest authority | `miniapp.json` is the versioned artifact of record: identity, tools, substrate, capabilities, policy, fixtures, deployment profile, exact dependency manifest |
| ADR-010 | Miniapp-toolchain scope | `agent-toolchain` and any generator work need separate scope justification; default is deferred |
| ADR-011 | Two-dimension classification | Dimension A (execution class) and Dimension B (compute placement) decided independently; admission checks enforce each against its own policy |
| ADR-012 | Compute-placement policy | Machines for trusted production; Sprites for untrusted/generated/interactive work; hybrid DBOS+Sprite for isolated tasks inside durable workflows (table in §10) |
| ADR-013 | Sprite SDK vs REST client | sprites-go (pinned, wrapped) for lifecycle/exec/filesystem/services/checkpoints/network-policy; framework-owned typed REST client for connectors/usage/logs/privilege gaps; both behind interfaces with fakes |
| ADR-014 | Lifecycle CLI | One `miniapp` binary, two command groups (invoke + lifecycle); exact surface frozen in Phase 0 from prompt2's proposals (§13) |
| ADR-015 | Hybrid execution pattern | DBOS orchestrates, Sprite executes the bounded task, Neon holds task identity/result, no unrestricted credentials enter the Sprite |
| ADR-016 | Binary/registry model (A15, refined by F01) | One Go module and statically registered implementation catalog; shared gateway by default. Each Class D miniapp/workflow family has its own DBOS `AppName` and explicit workflow-compatibility version, with a separately selectable worker process group/release that can retain the old artifact while old workflows drain. The same binary/image is the default; separate binaries are not required unless the Fly release-boundary spike proves process groups cannot preserve version isolation |
| ADR-017 | Lifecycle golden path (A16) | One INIT→RETIRE path across the CLI and GitHub Actions CI/CD on shared services and policies; defined in Phase 0, implemented incrementally Phases 1–6, verified end-to-end in Phase 6 |

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
                     +--+------+ +--+-------+ +--+-------------+ +--+---------+
                     | Class A | | River    | | DBOS family   | | Capability |
                     | handler | | workers  | | workers (D,   | | proxy +    |
                     | + tx    | | (B/C)    | | per AppName)  | | token      |
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
|   dbos schema: per-AppName workflow checkpoints, sleeps, events     |
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

Resolve the miniapp/tool version, authenticate the agent principal, bind the
authority principal, enforce tenant/capability/limit/approval policy, and
write the invocation audit record before dispatch. **Class A is synchronous
by contract:** return the result inline within the request; do not force an
agent polling round trip for ordinary reads or commands. Persist an
`invocation_id`/audit record even when no asynchronous run handle is needed.

Class B/C/D operations return a stable `run_id` and use status/signal/cancel
or long-poll operations. A Class A tool that requires a durable wait, retries
independently, or outlives the request must be classified and implemented as
B/C/D rather than silently inheriting an asynchronous lifecycle. An
explicit `wait_ms` may reduce polling for asynchronous runs; it does not
change the execution class.

### 6.4 Unified dev-to-ops lifecycle

Design → Develop → Validate → Publish → Deploy → Operate → Maintain remains
the target lifecycle. Its commands are introduced only in the phase that
owns the capability; the CLI must not expose aspirational commands backed by
stubs. GitHub Actions calls the same framework services/policies as the CLI.
The first golden-path evidence is the Phase 1 slice dogfooding `init`, `dev`,
`validate`, and `test`; preview/publish arrive with Phase 4, deployment and
promotion with Phase 5, and the complete upgrade/rollback/retire journey is
verified in Phase 6. The Sprite extension stays separately gated and does
not block the Machine-only core path.

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
registry/                       discovery, registration, version resolution (static registration in the shared host; ADR-016)
runtime/                        invocation lifecycle, dispatch, run handles
executor/
  executor.go                   substrate interfaces + admission (ADR-001 gate)
  classa/                       plain handler + tx adapter
  river/                        Class B/C adapter (progress, uniqueness)
  dbos/                         Class D adapter (workflows, events, recovery)
deploy/                         Dimension B: targets + lifecycle (public)
  profile.go                    deployment profiles (machine/sprite/hybrid/local)
  machine/                      Fly Machines target adapter (implemented Phase 5)
  sprite/                       Sprite adapter: workspaces in Phase 4, tasks in Phase 5;
                                sprites-go wrapper (pinned)
  rest/                         framework typed REST client (gap APIs; Phase 5)
  tasks.go                      bounded Sprite-task primitive for hybrid flows (Phase 5)
  promotion.go                  publish records (Phase 4); promote/rollback (Phase 5)
capability/
  spec.go                       CapabilitySpec contract
  registry.go                   connector registry + operator-approved origins
  clients/                      typed SaaS/custom-REST clients (no creds held)
  egress/                       SSRF-safe outbound HTTP dialer and redirect policy
auth/
  identity.go                   agent + authority principals
  policy.go                     scopes, trust modes, approvals policy
  enforce.go                    server-side authorization checks
limits/                          per-principal/tenant/capability quotas, circuit breakers, kill switches
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
miniapps/<name>/                one reference miniapp per directory (see §16); implementations statically linked in the shared host, manifests independently versioned (ADR-016)
  miniapp.json  tools/  capabilities/  fixtures/  policy/  README.md
fly/                            fly.toml, process groups, health checks, profiles
docs/adr/                       ADR-001..017 formalized in Phase 0
  miniapps/                       per-Class-C/D DBOS adoption decision records
docs/evidence/                  phase-gate reports and reproducible proof index
```

Public API surface adds `deploy` to plan4's list. Miniapps import only
public packages; CI enforces this with an import checker.

## 8. Public Go API proposals, with representative signatures

Illustrative only; Phase 0 freezes exact signatures. plan4's runtime,
registry, capability, effects, and approvals sketches are retained; the
additions below cover Dimension B. The retained registry sketch resolves
to static registration in the shared host per ADR-016; no plugin-style
hot-loading. Error values stay typed (terminal vs
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
  Files map[string][]byte // staged inputs, size-capped, never credentials (A1)
  Authority TaskAuthority // proxy-issued scoped task identity, no bearer tokens
  NetworkPolicy NetworkPolicy // applied and read back BEFORE execution (R21, A3)
  Timeout time.Duration
  ResultRef string // where the result lands (Neon, not Sprite FS)
}
func RunTask(ctx context.Context, task SpriteTask) (TaskResult, error)
func GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error)
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
(review gate, required checks). Immutable once published. Referenced
deployment profiles and all transitive policy inputs are content-addressed
at publish time: their digests are embedded in the artifact, and `validate`,
`publish`, `promote`, and task admission reject unresolved, unbound, stale,
or mismatched profile references (A4). A profile change requires a new
immutable version and fresh validation/review. Every consequential write tool
MUST declare an `entity_key`/serialization scope or an explicit reviewed
`no_entity` reason. The manifest also declares enforceable blast-radius
limits (per principal × tenant × capability rate/volume caps, max records per
run, monetary caps where applicable, recipient/domain allowlists, and a
kill-switch/circuit-breaker policy reference). For Class C/D miniapps, the
manifest references the approved workload-specific DBOS decision record.

The published release envelope binds the manifest digest to the exact
implementation/build digest, source revision, dependency lock, and deployed
artifact digest. This binding is external to the self-hashed manifest payload
to avoid a circular digest; the registry rejects an implementation whose
release envelope does not match the manifest's declared implementation
identity.

### 9.2 `CapabilitySpec` (per approved operation)

Each capability declares provider/operation, pinned API version, schemas,
connection reference, minimum OAuth/API scopes, tenant/resource scope,
read/write and risk class, attribution mode (`delegated-user` or approved
`service-identity`), provider rate limits, timeout/response-size limits, retry,
idempotency, error taxonomy, audit, fixtures, and its compensation or
reconciliation operation. For OAuth-backed integrations, effective authority
is the intersection of manifest-approved scopes, the grant actually issued,
and the privileges of the SaaS-side identity; a provider denial is
authoritative. Discovery yields candidates; approval is a separate reviewed
step. A tenant-supplied base URL is never passed directly to a generic HTTP
client; destinations are operator-approved and enforced by `capability/egress`
(§14).

### 9.3 Identity and policy

The authority model has two business/security principals: the calling
agent/service and the authority on whose behalf the operation is performed
(delegated human, explicitly approved service identity, or authorized
trigger). Trust mode is bound per miniapp version per tenant. A Sprite task
identity, when the optional extension is enabled, is a credential-less
scoped workload identity derived from this authority; it is not a third
human/business principal and never carries SaaS bearer credentials. Every
provider action rechecks manifest policy, granted scopes, SaaS-side
permissions, tenant/resource scope, quotas, and risk controls.

### 9.4 Invocation and asynchronous run envelope

Class A returns a typed result inline and records an invocation/audit identity;
it does not require `get_run` polling. Class B/C/D return a stable `run_id`,
status, and `poll/signal/cancel` operations. Asynchronous runs have no
arbitrary active-handle expiry; terminal-history retention is a separate
policy. Sprite tasks, if enabled, reuse the async envelope with a distinct
`task_id` nested under the parent run. Provider free text is returned as
explicitly untrusted data, never as instructions; output schemas impose size
caps and pagination/cursors for collection results.

### 9.5 External-effect record

The framework owns `PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN`
with reconcile-by-read, quarantine for unknown outcomes, and the corrected
acceptance target: no duplicate confirmed business effects, no silently lost
operations, and no blind retries of unknown outcomes. An external-write tool
MUST NOT be released in autonomous/confirm write mode until the effect ledger,
reconciliation path, approval policy where required, and crash-boundary
tests are available. Earlier integration phases may exercise write paths only
in `dry-run` or against deterministic test providers. Saga compensation is
idempotent, registered before the forward side effect, and never auto-fires
on UNKNOWN without a safety basis. Entity serialization is explicit per
manifest `entity_key` (see §15).

### 9.6 Sprite-task and promotion records (new)

A Sprite task declares: stable task ID, parent run/workflow ID, snapshot
base, staged inputs (capped, never credentials), command, network policy,
timeout, expected result schema, and result destination in Neon. Every
task is tracked in a durable Neon task registry (A2) carrying the task ID
(unique business identity), parent run/workflow identity, provisioning
state, Sprite ID, policy digest, lease and fencing information, execution
state, result reference, cleanup state, and last error. Provision and
reconcile operations are idempotent on the task identity: the runner
persists a unique task intent before provisioning, and retries recover
the existing task instead of creating another Sprite. Lifecycle:
`STAGED -> PROVISIONING -> POLICY_APPLIED -> RUNNING -> SUCCEEDED |
FAILED | TIMED_OUT | CANCELLED`, with a lease-based reaper resolving
abandoned tasks (revoke authority, terminate the Sprite, record an
UNKNOWN or failed outcome) so the parent workflow can reconcile safely.
The `POLICY_APPLIED` transition requires the expected restrictive policy
to be applied and read back with a digest matching the immutable
deployment profile bound to the task; any failure fails closed without
executing, and the policy is re-checked before each new execution where
configuration could have drifted (A3). Terminal results are copied to
Neon before the workspace may be decommissioned.

A promotion record binds the immutable release artifact — built from
reviewed source in a controlled trusted build environment as an OCI image
(or equivalent content-addressed build artifact) with dependency lock,
build provenance, SBOM where feasible, test and security results, source
revision, manifest digest, embedded deployment-profile digests (A4), and
the review decision with reviewer identity — to its target (A6). Sprite
snapshots may be retained as workspace provenance or restore points but
are never promoted directly as production artifacts. Rollback redeploys a
previously approved immutable artifact under the documented workflow
compatibility strategy.

## 10. Execution-class decision matrix, DBOS adoption policy, and compute placement

### 10.1 Dimension A — execution semantics

The governing policy in `docs/DBOS.md` is adopted by principle, not copied
verbatim. It was written against a different application/job stack. Phase 0
must port its workload taxonomy and qualification gates to Go, PostgreSQL,
River, and DBOS. The policy's business questions remain authoritative; its
examples and alternative-technology references must be rewritten for this
stack.

| Class | Form | Substrate | Governing rule |
|---|---|---|---|
| A | Ordinary operation completed in one request | Plain Go handler + PostgreSQL transaction | Synchronous result; no workflow engine; external writes still use the effect protocol |
| B | One-shot independently retryable job | River by default | Transactional enqueue, unique identity, bounded retries; DBOS only with a documented reason |
| C | Bounded batch or multi-step operation | River or DBOS queue, chosen per workload | Mandatory comparison where partitioned/global concurrency, rate limits, checkpoint recovery, or resume semantics matter; no blanket default |
| D | Durable multi-stage business workflow | DBOS | Durable waits, checkpoints, signals, recovery, version strategy, and operational ownership |

Async, scheduled, or multi-step alone never justifies DBOS. Every Class C/D
miniapp requires a human-reviewed `adr001.md` decision record containing the
business promise, score, Gates 1–4, alternatives, custom machinery, residual
work, operational cost, and links to evidence. Automated admission may check
record presence, required fields, score arithmetic, approved decision state,
and test links; it cannot make the business-consequence or net-complexity
judgment. Those gates require a named reviewer.

Class C pilot evidence must compare the actual workload under River and DBOS
queues when global/per-tenant/per-provider concurrency or per-entity
partitioning is material. River's open-source worker limits are not to be
mistaken for a global concurrency governor; River Pro documents global and
partitioned concurrency limits as Pro functionality. DBOS Go documents
queue-wide and partition-specific concurrency/rate controls. Do not infer
that DBOS always wins: measure code volume, transaction behavior, throughput,
rate-limit correctness, recovery, and operational burden.

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

### 11.1 River (Class B default; Class C subject to comparison)

Use River for independently retryable one-shot jobs. Enqueue in the same
transaction as the motivating application write where required; use stable
unique identities, retries with backoff, scheduling, and inspectable terminal
failures. Do not treat per-process worker counts as global provider limits.
If a workload requires global or per-tenant/per-provider limits, implement a
shared database-backed limiter or choose a queue with verified native
capabilities. For Class C, compare River plus application-owned progress and
idempotency against DBOS queues/partitioning before fixing the substrate.
River's `InsertTx` atomicity and selected Neon connection mode are proven at
the G3 entry gate, before Class B/C production jobs are enabled.

### 11.2 DBOS (Class D; selected Class C pilots)

DBOS workflow replay is application-version-sensitive. Each Class D
miniapp/workflow family therefore has a stable, distinct DBOS `AppName` and
an explicitly managed `ApplicationVersion`; the version is tied to the
workflow implementation compatibility line, not blindly to a manifest
version or the whole repository version. The default deployment is one
worker process group per Class D family, using the shared Go binary/image
with a family selector. A release may start a new family version while the
previous artifact remains available to drain or recover old-version
workflows. Do not retire an old version until its pending/running workflows
are drained or explicitly migrated with a verified patch strategy. If Fly
cannot independently deploy/retain these worker groups, use separate Fly
apps for the affected workflow families; separate compiled binaries are not
a prerequisite.

Workflow code is deterministic: nondeterminism and external I/O belong in
durable steps/transactions; stable workflow IDs and compact IDs-only
payloads are mandatory. Use DBOS queues for rate-limited fan-out and, where
needed, global or per-partition concurrency. `entity_key` can be a partition
key for per-entity single-writer execution, but ordering/fairness and
interaction with queue limits must be tested. The open-source baseline keeps
one active executor per DBOS application/family unless evidence supports a
more available topology. This is one executor per family, not one executor
for the entire platform, and it is not an HA claim. No scale-to-zero claim
until the wake test passes.

### 11.3 Sprite tasks and the hybrid pattern (later extension)

The approved A1–A6 pattern remains unchanged and is implemented only in its
owning phases. DBOS is authoritative for the workflow; Neon is authoritative
for task identity/result; Sprite filesystem checkpoints are workspace
snapshots only. Sprite code receives no provider credentials and accesses
external systems only through the capability proxy. The network policy must
be applied, read back, and matched to the immutable profile digest before
execution. A missing Sprite organization/quota or failed policy round-trip
defers Sprite preview/task work, not the Machine-only Phase 1 core.

## 12. Neon connection and transaction design

Use one primary with separate ownership boundaries for application data,
River-managed tables, and DBOS-managed tables. DBOS uses a direct or proven
session-compatible connection initially. App/River connection modes are
selected by workload-specific tests; schema migrations use direct
connections and never race library-managed migrations.

Tenant context is transaction-local and set inside the same transaction as
all tenant queries (e.g. `SET LOCAL` or transaction-local `set_config`). Do
not use session-level `SET` state or session advisory locks through Neon's
transaction pooler. The tenant-context helper is the only approved entry
point for opening a tenant transaction. Prove isolation under concurrent
pooled connections and `FORCE ROW LEVEL SECURITY`; app-level authorization
remains mandatory.

Budget direct connections explicitly:

```text
peak_direct_connections =
    gateway_direct_pools
  + river_direct_pools
  + Σ(DBOS_family_workers × DBOS_pool_size)
  + migrations_and_ops
  + measured_headroom
```

The calculated peak must remain below the selected Neon compute limit with
headroom. Measure autosuspend behavior and cost under the real DBOS/River
polling configuration; do not assume the production compute can suspend or
that an idle open client connection alone proves it cannot.

**PITR/restore is an external-effect recovery event, not an ordinary restart.**
A database restore rewinds application state and DBOS checkpoints but does
not rewind provider-side effects. Runbook: stop ingress and all workers;
restore to the target point; enumerate all nonterminal operations/workflows
whose external effects may have happened after that point; mark them
`UNKNOWN`/paused; reconcile provider state by idempotency key or read; resume
only after an operator-approved recovery decision. Keep PII out of workflow
payloads so checkpoint restoration/retention does not become an uncontrolled
copy of customer data. Drill the procedure before production GA.

## 13. MCP, HTTP/OpenAPI, and CLI architecture

One canonical invocation service; thin adapters sharing execution,
authorization, and audit. REST/OpenAPI (canonical, versioned) and the
central MCP router (generic tools first, namespaced tools once proven) are
unchanged from plan4.

The CLI remains one `miniapp` binary with two groups (R19, ADR-014). Its command surface is phase-gated: implement a command only when its owning lifecycle capability exists; do not ship no-op placeholders or duplicate `flyctl`/`psql` operations:

- **Invoke group** (thin REST client, plan4 behavior retained): run a
  miniapp tool, poll status, signal, cancel — with stable `--help`, JSON
  output, and exit codes as contract.
- **Lifecycle group** (new): assessed in Phase 0 from prompt2's proposals
  against a no-proliferation rule — every command must map to exactly one
  lifecycle stage in §6.4, own no policy, call shared services, and never
  duplicate `flyctl`, `psql`, or existing platform administration tools (A14).

Proposed assessment (Phase 0 freezes this):

| Proposal | Assessment |
|---|---|
| `init --class --profile` | Keep: scaffold from class + deployment-profile template |
| `dev [--sprite]` | Keep: local dev server or isolated Sprite workspace |
| `validate` | Keep: manifest, schema, policy, placement, and profile-digest admission checks |
| `test` | Keep: thin over `testkit/` (unit → crash-boundary) |
| `preview` | Keep: disposable Neon branch + Sprite preview environment |
| `publish` | Keep: immutable versioned artifact + provenance |
| `deploy` | Keep: Machines services/workers, approved Sprite workspaces |
| `status`, `runs`, `logs` | Keep as one `status` family: `status`, `runs`, `logs` share one query path |
| `promote`, `rollback` | Keep: governed release moves with recorded decisions |
| `doctor` | Add only when it provides diagnostics not already covered by `flyctl`, `psql`, or the host platform |

CI/CD parity (A16): the standard GitHub Actions workflow (validate, test,
build, publish, deploy) calls the same framework services and policies as
the CLI — never a parallel implementation. Environment promotion (local →
preview → staging → production) reuses the same deployment profiles,
migration gates, and secret boundaries with environment-specific
configuration; Phase 0 designs this once, phases implement it
incrementally per §19.

Harness validation still covers one coding-agent harness and one
general-purpose runtime minimum, with transports and approval behavior
verified per harness.

## 14. OAuth, tenant security, and authorization architecture

Keep the two separate authorization boundaries: agent→platform and
platform→provider. For each provider call, compute effective authority as the
intersection of: (1) manifest-approved operation/scopes, (2) the OAuth/API
scopes actually granted, and (3) the privileges of the connected SaaS
identity. Provider `403` is authoritative. Manifest declares whether actions
are attributed to a delegated user or approved service identity.

OAuth code+PKCE is used where supported; refresh-token rotation/revocation
and encrypted-at-rest storage remain mandatory. If a grant expires or is
revoked (`invalid_grant`), mark the connection `NEEDS_REAUTH` and return a
typed `needs_reauth` result with an out-of-band human action link. An agent
cannot complete browser consent or elevate the grant itself. Provider terms,
scopes, rate limits, bulk limits, and app verification requirements are
recorded per connector.

### 14.1 SSRF-safe egress for custom REST

No agent or tenant may supply an arbitrary URL that is fetched directly by
the platform. A custom REST destination must be registered and approved by
an operator. The egress layer enforces HTTPS, exact host/origin allowlists,
DNS A/AAAA resolution, connection-time validation of every resolved IP, and
blocking of loopback, private, link-local, carrier-grade NAT, multicast,
reserved, cloud metadata, Fly private-network/`.internal`, and database
internal destinations. The HTTP client connects to the validated IP while
preserving the approved hostname for Host, TLS SNI, and certificate checks;
no second unchecked DNS resolution is allowed. Redirects are disabled by
default; if explicitly permitted, every hop is revalidated and cross-host
redirects are rejected. Retries and fallback paths use the same policy.
Response size, duration, content type, header size, and decompression limits
are enforced. All custom REST traffic passes through this egress layer.

### 14.2 Blast-radius controls

Limits apply to ordinary miniapps as well as Sprite tasks. Enforce server-side
limits by principal × tenant × capability: request rate, daily/hourly write
count, maximum records per run, monetary/value cap where applicable,
recipient/domain allowlists, concurrency, and provider rate limits. A circuit
breaker trips on configured error/duplicate/UNKNOWN spikes. A kill switch
exists at tenant, miniapp, and capability levels and takes effect without a
redeploy. Limits and kill-switch policy are declared in the manifest and
persisted centrally so horizontal workers cannot bypass them.

### 14.3 Untrusted tool results

Provider-returned strings, CRM notes, emails, document text, and error bodies
are untrusted data even when the provider connection is authenticated. Return
typed fields; mark provider-originated free text as `untrusted: true`; cap
sizes and collection pages; never turn provider text into tool instructions
or trusted error prose. The agent may interpret such content, but it cannot
change platform policy. High-risk writes require the approval policy
regardless of text returned by the provider. CI includes poisoned-record and
prompt-injection fixtures.

### 14.4 Approval channel and separation of duties

Approval is enforced by the platform, not by MCP elicitation. The approver
uses an authenticated channel outside the requesting agent session (for
example an authenticated operator interface or a step-up verified provider
channel). The requesting agent/service cannot approve its own request;
separation-of-duties rules are explicit. Approval binds to the exact
miniapp/version, tenant, authority, canonical input/material hash, and
expiry. Any material parameter or policy change invalidates the decision.

### 14.5 Sprite boundary (when the extension is enabled)

The A1 rule remains absolute: untrusted code receives no SaaS credentials,
refresh tokens, API keys, or bearer tokens. It invokes the capability proxy
using short-lived, scoped platform task authority; the proxy validates
miniapp, tenant/resource scope, operation, risk, quota, and audit policy and
performs the provider call itself. Network policy is defense in depth, not a
credential boundary.

## 15. External-effect, idempotency, reconciliation, approvals, and keyed serialization

The framework owns `PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN`.
Every provider write begins with a durable effect intent and stable business
idempotency identity. Provider-supported idempotency/status APIs are used
where available; otherwise a read-after-write reconciliation method is
specified. A timeout after provider acceptance is UNKNOWN, not FAILED.
Never blindly retry or automatically compensate an UNKNOWN outcome.

The first autonomous/confirm external-write miniapp is gated until the
ledger, reconciler, approval store/channel, blast-radius controls, and
crash-boundary tests are complete. Earlier phases may perform read-only real
provider calls and dry-run/simulated writes only. This corrects the old
sequence in which `crm-write` was placed before the full reconciliation and
approval machinery.

### 15.1 Approval semantics by execution shape

`confirm` is a policy gate, not a fifth execution class. For a single-step
Class A/B command, persist an approval request with state
`PENDING -> APPROVED | REJECTED | EXPIRED`, `expires_at`, canonical
`material_hash`, approver identity/channel, requester identity, and decision
audit. Once approved, execute inline if it remains a valid short Class A
operation, or enqueue a one-shot River job for Class B. Do not introduce
DBOS solely to wait for an approval on one bounded operation. For a larger
Class D process, the same approval table remains the source of truth and the
DBOS workflow waits on a durable message/event with timeout. Recheck expiry,
material hash, authority, and self-approval policy immediately before the
consequential action.

### 15.2 Entity-level serialization

Every consequential write declares an `entity_key` or carries an explicit
reviewed `no_entity` justification. Class A uses a row lock or transaction
scoped advisory lock. Class B/C combines stable unique job/effect identity
with the entity lock in the worker. Class D may use a DBOS partitioned queue
with per-partition concurrency one, keyed by the entity. Workflow IDs alone
do not serialize access to an entity. Tests prove same-entity serialization,
parallelism for different entities, deduplication, and interaction with
provider rate limits.

### 15.3 Saga and compensation

Compensation is registered before the forward side effect, idempotent, and
unwound in LIFO order. It is not automatically executed on UNKNOWN without a
safe basis. Each compensator has its own identity, outcome, reconciliation
path, audit trail, and operator intervention policy.

## 16. Reference miniapp catalog and implementation sequence

Each reference keeps the full content bar (manifest, Go implementation,
contracts, bindings, permissions, substrate declaration, fixtures with
independent expected results, unit/integration/failure tests, local run
docs, deployment profile, examples, and operational notes). A miniapp is a
versioned business verb/outcome, not a thin wrapper around one provider
endpoint. If chaining separate tools can leave an inconsistent business
outcome across a crash, those steps belong inside one miniapp/workflow.
`crm-read` remains a deliberately small Class A forcing fixture, not the
flagship business-value example. All trusted production references default
to Machine placement; hybrid is demonstrated once, deliberately.

| Miniapp | Class | Placement | Phase | Purpose: what it proves |
|---|---|---|---|---|
| `crm-read` | A | Machine | 1 | Typed permission-checked reads; invocation + auth + audit path |
| `lead-followup` | D | Machine | 1 | Business-outcome workflow: durable waits, eligibility re-checks, events, cancel-safe recovery |
| `crm-write` | A | Machine | 4 | First real external write; only after effect ledger, reconciliation, approvals, limits, and crash tests pass; earlier phases are dry-run only |
| `notify-send` | B | Machine | 3 | One-shot job, idempotency key, delivery receipt |
| `report-generate` | B | Machine | 3 | Scheduled one-shot render + delivery |
| `contact-import` | C | Machine | 3 | Class C batch forcing workload; selected substrate (River or DBOS queue) follows the measured pilot, with per-record idempotency and resumability |
| `approval-flow` | D | Machine | 4 | Durable approval events, escalation, timeout |
| `data-sync` | C | Machine | 5 | Pull-merge-push, conflict policy |
| `appointment-journey` | D | Machine | 5 | Multi-touch waits, quiet hours, reschedule-cancel |
| `hybrid-task-demo` | D + task | Hybrid | 5 | DBOS workflow orchestrating one bounded Sprite task: happy path plus the full §18 failure matrix in its automated suite (no second demo) |
| `campaign-orchestration` | D | Machine | 5 | Rate-limited fan-out via queues (only when its provider-effect gate passes) |
| `agent-toolchain` | D | Machine default | Deferred | Resumable authorized agent/tool series; trusted Go execution stays on Machine, hybrid Sprite use needs separate justification |

The forcing set stays `crm-read` + `lead-followup` + `contact-import`; `contact-import` becomes the measured Class C River-vs-DBOS queue pilot, not a preselected River implementation. Business-facing catalog additions must be outcome-shaped (for example `invoice-to-ledger` or `employee-provisioning`), not one miniapp per API endpoint.
`hybrid-task-demo` is intentionally small: its job is to prove the §11.3
pattern (policy-before-exec, bounded task, Neon-held result, typed failure
mapping) across every §18 hybrid failure case, not to be a second product
surface (A9). Phase 2 otherwise covers connector and security integration
rather than catalog breadth (A12).

## 17. Fly deployment and operations model

Trusted workloads run on Fly Machines in a region selected by measured
latency to the Neon primary. Keep a shared gateway and River workers by
default. Deploy each Class D workflow family with an isolated DBOS
`AppName`/version and a process group/release boundary that permits old
workflow code to remain available while workflows drain. A single Go image
may be reused. Do not use blue-green/surge overlap for one DBOS application
until the pinned DBOS SDK's ownership/fencing behavior and Fly's actual
release sequence are demonstrated together. Default to a controlled
stop/drain/start or per-family deployment that prevents two incompatible
versions from racing recovery. One executor per DBOS app is the baseline;
this is not an HA claim.

| Target | Process / unit | Responsibility | Scaling / release rule |
|---|---|---|---|
| Machine | `gateway` | REST/MCP/CLI, Class A, auth, approvals, webhooks, capability proxy | Always on; horizontal scale only with shared limits and idempotency |
| Machine | `river` | Class B/C jobs and effect reconciler | Scale by backlog; global provider limits must be shared, not per-process only |
| Machine | `dbos-<family>` | One Class D miniapp/workflow family per DBOS `AppName` | Explicit app version; retain old artifact/worker until old workflows drain |
| Sprite | Workspaces | Previews and development workspaces | Per-workspace lease; phase-gated extension |
| Sprite | Tasks | Bounded untrusted execution | Per-task lease, proxy-only provider access, Neon result; Phase 5+ |

Fly supports rolling and blue-green deployment strategies; blue-green
starts replacement Machines alongside old ones. The DBOS deployment runbook
must explicitly select a strategy and test the overlap window rather than
assuming one Machine configured in a process group cannot overlap during a
release. `max 1` is a desired worker count, not proof of non-overlap during
all deployment strategies.

Static egress IP is optional and used only where the target SaaS/customer
requires source-IP allowlisting. Verify the current Fly product, app/region
scope, availability, and price when a real provider requires it; do not
allocate it by default. Operations also include health/drain, graceful
shutdown, executor-down/stuck-workflow alerts with named owners, connection
budgets, Neon restore reconciliation, retention/erasure, version drain,
rollback, and full retire procedures. Sprite-specific controls remain A1–A6
gated.

## 18. Test architecture and acceptance criteria

The test pyramid is unit → contract drift → real-Postgres integration on
Neon branches → crash-boundary on real Fly → security probes → harness
compatibility. Every claimed guarantee links to a named test/report, source
revision, environment, command, result, and date in the evidence register.

- **DBOS version/recovery tests:** a Class D family starts under application
version N, a new family version is deployed, old pending workflows resume
only on compatible old code or a tested patch, and the deployment does not
retire the old artifact before drain.
- **Synchronous Class A tests:** Class A returns inline without requiring a
run poll; asynchronous classes return stable run handles. Auth/audit still
records every Class A invocation.
- **Keyed-serialization tests:** same-entity writes serialize; different
entities progress concurrently; retries/deduplication do not violate order.
- **SSRF tests:** reject loopback/private/link-local/metadata/Fly-internal/Neon
destinations, DNS rebinding, IPv4/IPv6 alternate answers, redirects to a
second host, redirect loops, DNS changes during retries, oversized bodies,
unsupported content types, and slow responses. Confirm the transport connects
to the validated IP while preserving TLS hostname validation.
- **Blast-radius tests:** per-agent/tenant/capability rate and volume caps,
max records, value caps, recipient allowlists, circuit breaker, and each
level of kill switch hold under concurrent workers and retry storms.
- **Untrusted-result tests:** poisoned provider records remain marked as
untrusted data and cannot bypass server-side policy; output caps/cursors hold.
- **Approval tests:** expiry, material-input changes, self-approval, agent-only
approval attempts, replay, and out-of-band approver identity are rejected.
- **PITR/recovery tests:** after a simulated rewind, nonterminal effects are
paused/UNKNOWN and reconcile before workflows resume. A production PITR drill
is required before GA.
- **Retention/erasure tests:** PII is not copied into workflow checkpoints;
erasure requests remove or irreversibly de-identify eligible application
records while preserving only legally permitted audit evidence.
- **Placement admission tests:** manifests declaring the wrong target for
  their trust profile are rejected at `validate` with typed errors.
- **Profile-binding tests (A4):** manifests carrying unresolved, unbound,
  stale, or mismatched deployment-profile references are rejected by
  `validate`, `publish`, `promote`, and task admission; a profile edit
  forces re-validation under a new version.
- **Policy-before-exec tests:** a Sprite task whose network policy cannot
  be confirmed never starts; outbound attempts outside the allowlist fail;
  the default-open Fly behavior is verified first, then locked down.
- **Credential-isolation tests (A1):** code with full root privileges in
  a Sprite cannot read, enumerate, or use tenant credentials, reach
  another tenant, or execute an unapproved provider operation; Sprite and
  task payloads, workflow checkpoints, and logs contain zero secrets.
- **Checkpoint-semantics tests:** restore returns files only; workflow
  progress is unaffected and is proven to come from DBOS/Neon, not the
  restored filesystem.
- **Hybrid failure tests (A9):** the `hybrid-task-demo` suite exercises
  crashes at each provisioning transition (task intent, Sprite creation,
  policy application) plus provisioning/policy rejection, task timeout
  with a possibly live Sprite,
  Sprite loss and runner termination mid-task, result-schema
  mismatch, worker crash after Sprite creation but before task-ID
  persistence, worker crash after task completion but before result
  persistence, and cleanup/reaper behavior with repeated invocation —
  proving idempotent task identity, bounded resources, no credential
  leakage, authoritative Neon results, and recoverable parent state.
- **Reaper tests (A2):** a runner killed mid-task leaves no orphan
  compute: the lease reaper decommissions the Sprite, marks the result
  `UNKNOWN`, and the orchestrator reconciles without duplication.
- **Promotion tests (A6):** promotion without trusted-build OCI digest,
  dependency lock, SBOM where feasible, build provenance, source
  revision, test and security results, manifest and profile digests, or
  review decision is refused; rollback redeploys the prior approved
  artifact with its workflow compatibility.
- **Lifecycle CLI tests:** every `miniapp` command maps to one §6.4 stage,
  owns no policy, and round-trips through the same services as MCP/REST.
- **Golden-path tests (A16):** a developer starting from a clean checkout
  takes a reference miniapp from scaffolding through develop, validate,
  test, preview, publish, and deploy to a verified production deployment,
  with the CLI and the GitHub Actions CI/CD using the same framework
  services and policies; operators then inspect, upgrade, recover, and
  roll back or retire it without inventing a separate procedure. This
  journey is demonstrated with a complete working example and covers
  local development, Sprite previews, Neon branches, immutable
  artifacts, environment-specific configuration, migrations, secrets,
  approvals, observability, and rollback.

Acceptance language keeps plan4's four gates (invocation/identity,
multi-tenant, integrations, reliability) and adds: no agent-supplied code
runs on a Sprite without a confirmed restrictive policy; no SaaS
credential, refresh token, or bearer token ever enters an untrusted
Sprite; no Sprite state is authoritative for any business outcome.

## 19. Phased implementation roadmap

The implementation remains incremental. Phase 0 produces contracts and core
proofs; Phase 1 proves the core vertical slice. River, real-provider
security, write approvals/effects, Sprite workspaces, Sprite tasks, and the
full lifecycle arrive only when their dependencies are ready. A failed
extension-specific spike defers that extension; it does not block an
unrelated core phase. A failed core invariant blocks the phase that depends
on it. Each phase remains an independently reviewable coding-agent task.

### Phase 0 — Core architecture contracts and enabling proofs

- Objective: establish the minimum set of contracts and evidence required
  for the Class A/Class D Machine-based vertical slice. Do not implement
  Sprite workspaces, Sprite tasks, promotion, provider connectors, the full
  lifecycle CLI, or production deployment here.
- Packages/artifacts: Go module skeleton, CI skeleton, `contract/`, core
  `deploy/profile` interfaces only, `internal/store` tenant transaction
  helper, migration ownership skeleton, `docs/adr/`, version/evidence
  registers, threat model, and `docs/evidence/` index.
- Tasks:
  1. Name the security owner; accept or supply the missing `prompt2.md`
     remainder (Q8/Appendix D); establish Neon project/branch and region for
     DBOS crash testing. Fly organization/region must be identified for the
     later deploy spike; Sprite organization/quota is only required before
     Phase 4 workspace work.
  2. Port ADR-001 to Go/Postgres/River rather than adopting Hono/pg-boss
     examples verbatim. Define the per-Class-C/D `adr001.md` review template
     and human decision ownership.
  3. Freeze the core manifest, capability, two-principal identity, trust
     modes, sync/async invocation contract, effect record, entity key,
     limits policy, release envelope, and audit envelope. Keep Sprite fields
     explicit but its execution contract in the later extension scope.
  4. Freeze ADR-016 registration/build model and the DBOS family model:
     one stable `AppName` per Class D family, explicit workflow version,
     separate selectable worker release, old-artifact drain/retention. Prove
     the same-image/process-group deployment strategy; use separate Fly apps
     if the provider cannot safely preserve versions per family.
  5. Pin the dependencies needed by Phase 1 with versions, licenses,
     verification dates, upstream source links, and upgrade owners.
  6. Prove DBOS/Neon application transaction integration at the actual pool
     handle with crash-boundary probes; prove the tenant transaction-local
     context and RLS behavior under the selected pool mode; perform one MCP
     invocation/auth round trip with the chosen initial harness.
  7. Define A16 once: the clean-checkout-to-retire lifecycle, CI/CD parity,
     environments, secret/config references, migration ordering, and
     rollback responsibilities. Implement only the Phase 1 bootstrap now.
  8. Define the security/SSRF threat model and test design before any
     configurable custom-REST destination is enabled; actual egress probes
     and live provider OAuth are Phase 2 entry requirements.
  9. Define data classification, IDs-only DBOS payload rules, record-class
     retention, and erasure/de-identification obligations; implementation and
     end-to-end erasure tests land before production GA.
  10. Specify the DBOS Go determinism analyzer rules against the pinned SDK
      and make replay testing part of the Class D CI contract; implement the
      analyzer with the Phase 1 Class D slice.
  11. Produce a failure disposition and evidence record for each core spike.
- Out of scope: River atomicity (required before Phase 3), live SaaS OAuth
  (before Phase 2), Sprite lifecycle round-trip (before Phase 4), Sprite
  task/promotion implementation (Phase 5), real Fly deploy-overlap drill
  (Phase 6). They may be investigated early, but cannot hold G0-Core hostage.
- Exit gate G0-Core: (1) core A/D contracts reviewed/versioned; (2) Phase 1
  dependency pins verified; (3) DBOS/Neon atomicity crash probe passes; (4)
  DBOS family AppName/version/deployment plan is proven or has a documented
  safe topology; (5) tenant transaction-local context/RLS probe passes;
  (6) initial MCP auth/invocation round trip passes; (7) named security owner
  signs the core threat model; (8) Q8 is resolved by owner-supplied remainder
  or explicit acceptance of Appendix D; (9) A15/A16 decisions are recorded;
  (10) every core spike has a dated evidence artifact and failure
  disposition. G0-Core does not include River, Sprite, or first-SaaS proofs.

### Phase 1 — Core vertical slice and lifecycle bootstrap

- Objective: prove synchronous Class A and durable Class D execution on
  trusted Fly Machine-compatible runtime shapes using test connectors.
- Packages: `registry/`, `runtime/`, `executor/classa`, `executor/dbos`,
  `auth/`, core `effects/`, core `approvals/`, `audit/`, REST/MCP/CLI invoke
  adapters, `testkit/`, `crm-read`, and `lead-followup`; include a reviewed
  `adr001.md` decision record for `lead-followup` before enabling its DBOS
  execution.
- Dependencies: G0-Core.
- Tasks: build canonical invocation service; synchronous Class A result;
  stable run handle for Class D; auth-before-dispatch; audit; static registry;
  DBOS per-family `AppName`/version registration; run/status/signal/cancel;
  mock capability endpoint; `init/dev/validate/test`; one harness e2e; a
  pinned-SDK-aware Go analyzer for known nondeterministic workflow constructs.
- Tests: inline Class A; Class D restart/replay; version isolation/drain
  with an old workflow pending; duplicate submit; audit attribution; mock
  tenant/RLS behavior; approval-pending recovery with mock channels;
  analyzer fixtures and deterministic replay tests. CI runs the analyzer
  in strict mode for Class D packages before any Class D production release.
- Exit gate G1: Class A and Class D work on real Postgres with no duplicate
  confirmed effects, complete audit trail, and version-safe workflow
  recovery. No real provider writes are enabled.

### Phase 2 — Provider security and read-only integrations

- Objective: establish provider identity, token broker, capability proxy,
  SSRF-hardened egress, and one real SaaS sandbox plus one controlled custom
  REST destination. Real write operations remain disabled/dry-run.
- Dependencies: G1; resolve Q1 (first SaaS) before starting live OAuth work.
- Tasks: OAuth code+PKCE where supported; permission intersection;
  user/service attribution; refresh rotation/revocation; `NEEDS_REAUTH` and
  out-of-band human re-auth; RLS enforcement; webhook signature/replay
  protection; provider terms/scope notes; egress URL allowlisting, DNS/IP
  validation, redirect and response controls; shared limits and kill
  switches; provider rate-limit handling.
- Tests: tenant/app authorization, token leak, invalid_grant, permission
  intersection, 403 handling, re-auth, SSRF suite, webhook replay, limit and
  circuit-breaker probes against real sandbox plus malicious local targets.
- Exit gate G2: live read integrations and secure custom REST pass; no
  autonomous external write tool is released at this gate.

### Phase 3 — River jobs and Class C substrate decision

- Objective: add Class B jobs and a measured Class C batch adapter.
- Dependencies: G1; G2 for any authorized outbound provider sends.
- Tasks: prove River transactional enqueue against selected Postgres
  connection mode; implement B adapter; compare `contact-import`/`bulk`
  workload implemented as River + progress/idempotency versus DBOS queued
  workflows with partitioned/global concurrency and rate limits; write the
  per-miniapp ADR-001 decision; implement selected Class C substrate only.
- Tests: kill/resume, transactional enqueue, unique identity, global and
  per-tenant/provider rate behavior, per-record idempotency, fairness,
  concurrent worker tests, retention.
- Exit gate G3: River B is green and the Class C decision is explicit,
  evidence-backed, and reviewed. River is not selected for C merely by
  default where its limits would require disproportionate custom machinery.

### Phase 4 — Effects, approvals, first external write, and Sprite workspace preview

- Objective: finish unknown-outcome handling and approvals before releasing
  the first real external write; independently prove the Sprite workspace
  preview/publish extension.
- Dependencies: G2 and G3 for jobs/integration. Sprite workspace work also
  requires the Sprite organization/quota and a passing policy-before-exec
  lifecycle spike; failure defers the Sprite extension only.
- Tasks: effect ledger, reconcile-by-read worker, UNKNOWN quarantine,
  approval table and authenticated out-of-band channel, separation of
  duties, canonical material hash, expiry, saga compensation; release
  `crm-write` only after these controls pass. In a separate stream, build
  Sprite workspace preview, disposable Neon branch integration, publish
  records, profile digests, and phase-appropriate A16 GitHub Actions path.
- Tests: provider accepted-but-response-lost, unknown outcome reconciliation,
  retry/idempotency, approval self-approval and replay, material-change
  invalidation, out-of-band approver, compensator idempotency, workspace
  policy confirmation and isolation, immutable publish digest.
- Exit gate G4: first write has no blind retry path, approvals survive
  restarts, UNKNOWN is quarantined/reconciled, and any shipped Sprite
  preview has proven policy-before-exec. If the Sprite spike fails, G4 may
  pass for the Machine/write stream while preview remains deferred.

### Phase 5 — Catalog expansion, bounded Sprite tasks, and promotion

- Objective: expand business-outcome miniapps; implement Sprite bounded
  tasks and trusted-build promotion only after their A1–A6 controls exist.
- Dependencies: G4; Sprite task/promotion path additionally requires all
  A1–A6 security/provisioning/release controls and the successful Sprite
  provider/API proof.
- Tasks: remaining approved catalog one miniapp at a time; `data-sync`,
  `appointment-journey`, `hybrid-task-demo`; Sprite durable task registry,
  idempotent provisioning, policy-readback barrier, lease reaper, proxy-only
  egress, Neon-held results; trusted build OCI artifact with dependency
  lock, source revision, manifest/profile digests, provenance, SBOM where
  feasible, test/security results, reviewer identity; promote/rollback.
- Tests: entire seven-case hybrid failure matrix, root-in-Sprite credential
  exfiltration/tenant isolation tests, orphan cleanup/reaper, promotion
  refusal without evidence, rollback to previous approved digest.
- Exit gate G5: catalog miniapps meet business-outcome granularity and
  `adr001` evidence; no Sprite task runs without policy confirmation and no
  Sprite state or credentials are authoritative.

### Phase 6 — Production falsification and full lifecycle acceptance

- Objective: validate the claimed runtime/deployment guarantees on real
  Fly/Neon and two harness families; complete the A16 golden path.
- Tasks: DBOS wake/restart and version drain; Fly deploy overlap and
  graceful-shutdown drill; verify no two incompatible versions race one
  DBOS application's recovery; Neon PITR restore/reconcile drill; measure
  region latency, connection/cost budgets, egress if provider IP allowlist
  requires it; two-harness compatibility matrix; preview→publish→deploy→
  operate→upgrade→rollback→retire; retention/erasure and kill-switch drills.
- Exit gate G6: signed falsification/evidence report; all security,
  reliability, deployment, and lifecycle acceptance tests green. No HA
  claim for a single executor per DBOS application/family; no scale-to-zero
  claim without the wake test.

## 20. Dependency/version/license register

All entries UNPINNED at plan time for our module. Phase 0 pins exact
versions with verification dates and source links; `latest` is never a
strategy. The stack restates the tasks' required constraints, not
independent recommendations.

| Role | Candidate source | Version | License | Verify in Phase 0 |
|---|---|---|---|---|
| Language toolchain | Go (version TBD) | UNPINNED | — | Phase 0 selects the minimum required by Phase 1 core dependencies. Optional Sprite SDK/toolchain constraints are verified and adopted in Phase 4; they must not raise the core baseline before that dependency is introduced |
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
| Release builds | TBD in Phase 0 (selected), used Phase 5 | UNPINNED | Verify | Trusted OCI build + SBOM/provenance generation for promotion (A6) |
| Key management | TBD in Phase 0 | UNPINNED | Verify | Envelope encryption + rotation for tenant vault |

## 21. Risk register and unresolved technical questions

K1–K21 are carried forward, with DBOS executor wording interpreted per application/family. K22–K30 capture the additional findings consolidated from review2.

| # | Risk / question | Impact | Mitigation / resolution path |
|---|---|---|---|
| K1 | DBOS Go SDK is younger than TS/Python SDKs; APIs may drift from source-doc claims | High | Pin version; keep workflow logic thin; crash suite per upgrade |
| K2 | Sleep suspension / distributed recovery may be Enterprise-gated | High | Baseline open-source one executor per DBOS application/family, always on; Enterprise only on measured need |
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
| K20 | prompt2.md truncated at §7.1 — later task sections unknown | Medium | Appendix D completions, all labeled; owner confirms or supplies the remainder; bound to G0 (A14) |
| K21 | One executor per DBOS application/family has no HA posture until measured thresholds add capacity | Medium | A11 availability design in Phase 0; health/restart baseline in Phase 1; per-family executor-down and stuck-workflow alerting with drain drills in Phase 6; never claim HA from one Machine |
| K22 | Class D workflow-version mismatch strands old executions during shared-host deploys | High | DBOS AppName/version per family; retain old artifact/worker until drain; deploy-overlap proof in G6 |
| K23 | Custom REST SSRF/DNS rebinding reaches Fly private network or metadata | Critical | Operator-approved origin allowlist; connection-time DNS/IP validation; strict redirect/response policy; SSRF suite before G2 |
| K24 | External writes released before effect reconciliation and approvals are complete | Critical | Dry-run only before G4; first real write gated on effect ledger, reconciler, approval, blast-radius and crash tests |
| K25 | Same entity receives concurrent agent writes | High | Required manifest `entity_key`; row/advisory lock or DBOS partition key; serialization tests |
| K26 | Agent retry storm exceeds tenant/provider business limits | High | Shared quotas, value/record caps, recipient allowlists, circuit breakers, no-redeploy kill switches |
| K27 | Provider text or documents inject instructions into agent context | High | Mark provider free text untrusted, cap and paginate outputs, poisoned-fixture tests; server policy remains authoritative |
| K28 | Neon PITR rewinds DBOS checkpoints but not provider effects | Critical | Stop/restore/UNKNOWN/reconcile/resume runbook and pre-GA restore drill |
| K29 | PII persists in DBOS checkpoints or append-only audit beyond retention/erasure policy | High | IDs-only workflow payloads, explicit erasure/de-identification rules, erasure tests |
| K30 | Review checklist marks plan rereads as proof | Medium | Separate design-consistency review from behavioral evidence; every gate links test/report, commit, environment, command, result, date |

Evidence is phase-scoped. Before Phase 1, require the DBOS/Neon pool-handle crash probe, DBOS family-version/release isolation proof, tenant-context/RLS proof, initial MCP/auth round trip, and pinned Phase 1 dependency register. River atomicity and Class C comparison belong to Phase 3; live SaaS and SSRF probes to Phase 2; Sprite lifecycle/API proof to Phase 4; Fly deployment-overlap and production recovery drills to Phase 6. A later-phase proof must not be treated as a Phase 0 prerequisite.

## 22. Explicit out-of-scope items

AI miniapp generator; unrestricted shell/browser automation; arbitrary
LLM-code deployment; UI generation; BPMN, rules-engine, or
custom-workflow-engine work; rulez engine development; free-form LLM code
execution in the trusted runtime; per-miniapp Fly apps by default; Neon
project-per-tenant except for compliance tiers; connectors beyond the
first SaaS + first custom REST until Phase 5; marketplace, billing, and
metering (productization later).

## 23. Final readiness assessment and implementation gates

- **Approval:** Phase 0 is approved to begin. No further general architecture
  review is required to start implementation.
- **Core gate:** Phase 1 waits on G0-Core, which contains only requirements
  necessary for the Class A/Class D Machine-based slice.
- **Phase-specific gates:** G2 gates real provider security/read integrations;
  G3 gates River and the Class C decision; G4 gates first real writes plus any
  Sprite workspace preview; G5 gates Sprite tasks and trusted promotion;
  G6 gates real Fly deployment, recovery, multi-harness, and complete lifecycle
  acceptance.
- **Non-blocking extension results:** failure of the River pilot defers Class C;
  failure of the Sprite lifecycle/policy spike defers Sprite preview/tasks; a
  missing static egress feature defers only integrations that require IP
  allowlisting. None silently waives a security invariant.
- **Owner items:** Q8/Q9 are G0-Core; Q1 is Phase 2; Q4 is staged as described
  in Appendix B. No owner decision is silently inferred.
- **Evidence:** every gate requires linked behavioral evidence, not a re-read
  of this document. A failed core spike blocks dependent work; a failed
  extension-specific spike defers that extension and does not block unrelated
  phases.

### 23.1 G0-Core pass conditions

G0-Core passes only when all of the following hold:

1. Core Class A/D contracts, identity/authority model, sync/async invocation,
   effect record, entity-key semantics, registry model, and non-circular
   manifest-to-build release binding are reviewed and versioned.
2. Phase 1 dependencies are pinned with versions, licenses, verification
   date, upstream source, and upgrade owner.
3. DBOS/Neon transaction semantics are proven at the actual shared pool-handle
   boundary by crash probes.
4. Each Class D family has a stable DBOS `AppName`, explicit workflow-version
   policy, and an old-version retention/drain strategy; same-image worker
   process-group isolation is proven or the deployment unit is revised.
5. Tenant transaction-local context and RLS behavior are verified under the
   chosen connection mode and concurrency.
6. The initial MCP/REST invocation round trip works with the first target
   harness; the larger two-family harness matrix remains a Phase 6 gate.
7. The named security owner signs the core threat model.
8. The truncated `prompt2.md` is resolved by owner-supplied remainder or
   explicit acceptance of Appendix D.
9. ADR-001 is ported for Go/Postgres/River and the per-Class-C/D decision
   record and human approval responsibilities are defined.
10. A15 binary/registry and A16 golden-path/CI design are recorded.
11. Every core spike has a dated report with test names, source revision,
    environment, commands, results, and failure disposition.

G0-Core does **not** require the River atomic-enqueue spike, live SaaS/Q1,
Sprite provision/exec/teardown, production Fly deployment, or static egress.
Those are gated by the phases that first use them.

### 23.2 Phase-specific proof gates

- **G2 provider gate:** Q1 selected; OAuth and permission-intersection tests;
  SSRF/effective-egress suite green before configurable custom REST; re-auth,
  provider terms, webhooks, quotas, and kill switches verified.
- **G3 River/Class C gate:** River `InsertTx` atomicity and connection mode
  proven; Class C River-vs-DBOS comparison recorded and human-reviewed.
- **G4 effects/Sprite-preview gate:** effect ledger, reconciler, UNKNOWN
  quarantine, out-of-band approval, entity serialization, blast-radius and
  first-write tests green. Sprite workspace preview additionally requires a
  successful policy-application/read-back/exec/teardown proof; otherwise only
  that extension is deferred.
- **G5 Sprite-task/promotion gate:** A1–A6 controls and the full hybrid failure
  matrix pass; trusted artifact provenance and review gate are enforced.
- **G6 production gate:** Fly deploy overlap/drain tests, DBOS recovery/version
  tests, Neon PITR reconciliation drill, two-harness matrix, retention/erasure,
  full A16 lifecycle, and cost/connection/egress evidence signed.

### 23.3 Failure dispositions

A failed spike triggers a scoped redesign or deferral, never an undocumented
workaround:

- DBOS/Neon transaction atomicity fails: stop the DBOS adapter design and
  revise the persistence boundary; do not claim atomic business-write and
  checkpoint behavior.
- DBOS version isolation/deploy drain fails: keep the old deployment topology
  or split the affected Class D family into an independently deployable Fly
  app; do not ship a release that strands old workflows.
- River atomic enqueue fails: use a proven connection mode or revise the
  enqueue boundary before Phase 3; Class C remains unselected pending its pilot.
- Sprite policy/lifecycle fails: defer Sprite preview/task execution; never run
  untrusted code without confirmed restrictive policy.
- SSRF probe fails: block the affected custom REST destination class until the
  egress guard is fixed and the full suite passes.
- Neon RLS/tenant-context probe fails: revise the transaction-local context and
  role model before multi-tenant writes.
- Fly static egress IP unavailable: integrations requiring source-IP allowlist
  remain unsupported until an acceptable network design is proved.
- Pinned SDK lacks required APIs: choose a verified compatible release, use a
  documented supported API, or defer the feature; never code against imagined
  endpoints.

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

Decision status carried forward from the plan4.1 freeze. These questions remain owner decisions; none is invented. Q8 and Q9 gate G0-Core. Q4 is split: Neon project/region for G0-Core, Fly deployment organization/region before G6 deployment drills, Sprite organization/quota before Phase 4 workspace work. Q1 gates Phase 2, not G0-Core, because Phase 1 uses test connectors.

1. Which SaaS should be the first real integration (needs OAuth2 + sandbox
   tenant; idempotency keys and status lookup strongly preferred)?
   Gates Phase 2 live OAuth/provider work, not G0-Core. Status: pending.
2. Which two harness families must pass Phase 6 (default: one coding-agent
   harness and one general-purpose runtime)? Status: pending.
3. Is DBOS Enterprise (sleep suspension, Conductor) budgetarily in scope
   if the open-source wake test fails, or must the design stay
   open-source-only? Status: pending.
4. Do Fly.io and Neon organizations/regions already exist, or should
   Phase 0 create them (which region; is there a Sprite org with quota)?
   Neon project/region gates the DBOS/Neon G0-Core probe. Fly org/region gates the later Fly deploy proof; Sprite org/quota gates Phase 4 workspace work. Status: pending.
5. Is `docs/plan3_review.md` (currently empty) owed a review, or may
   Phase 0 remove it? Status: pending.
6. May `agent-toolchain` be deferred past the initial catalog, or does a
   specific use case require it early? Status: pending.
7. Is the §13 CLI assessment (one binary, two groups, proposed
   keep-list) approved as the Phase 0 freezing basis, or should any
   command be cut/renamed now? Status: pending.
8. prompt2.md ends at §7.1 — is there a remainder to supply, or
   should the Appendix D completions stand? Bound to G0 (§23.1).
   Status: pending — the remainder has NOT been supplied.
9. (New, A7) Who is the named security owner accountable for signing
   the threat model and reviewing capability, privilege, Sprite-profile,
   and approval-boundary changes? Gates G0 condition 9. Status: pending.

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
requirements, and any revised phase plan — is missing. Status at freeze:
the remainder has NOT been supplied; the completions below stand as
architect's completions requiring explicit owner acceptance, bound to G0
per §23.1 and Appendix B Q8. This plan completes those gaps as follows:

- D1: Lifecycle stage requirements inferred from prompt2 §4's seven-stage
  list; encoded as §6.4 with CLI mappings in §13.
- D2: Hybrid failure semantics (§11.3) and Sprite security rules (§14)
  inferred from prompt2 §§2–3 plus the inspected SDK's actual primitives.
- D3: Phase integration (§19) keeps plan4's Phase 0–6 skeleton and folds
  Sprite/lifecycle work into the phases owning their dependencies, since
  no revised phase plan survives in the truncated file.
- D4: Precedence (§3) follows prompt2's PART I ordering (gemini above
  deepseek) as the last complete instruction on the subject.

## Appendix E. Revision history and amendment-disposition checklist

### Revision history

| Date | Revision | Change |
|---|---|---|
| 2026-11-22 | Initial plan4.1 | Consolidated Implementation Plan v2 from prompt2 + sprites-go evidence |
| 2026-11-22 | Frozen baseline | Incorporated A1–A14 from `docs/plan4.1_consolidated_review.md`; no architecture redesign; K21 added; Appendix B Q9 added; §23.1/§23.2 added |
| 2026-11-22 | Frozen baseline (plan4.1) | Recorded owner-directed A15/A16; added ADR-016/017, §18 golden-path tests, G0 conditions 12–13 |
| 2026-11-22 | Consolidated implementation plan v3 (this revision) | Selectively incorporated review2 F01–F24; added DBOS family version isolation, SSRF/effect/approval/entity controls, sync-first Class A, evidence gates, PITR and erasure controls; split core and extension gates |

The consolidated review and the individual adversarial reviews remain
separate historical records under `docs/`; their reasoning is preserved
there and traced here, never silently discarded.

### A1–A16 disposition checklist

A1–A16 remain incorporated from plan4.1. Their prior “verified” labels mean plan-text consistency review only, not implementation proof. Review2 findings F01–F24 and their dispositions are recorded in Appendix F; behavioral validation is evidenced only by gate reports/tests linked from the evidence register. These are implementation refinements, not a new ADR-numbering series.

| # | Amendment | Incorporated in | Verified |
|---|---|---|---|
| A1 | Strict Sprite credential boundary via capability proxy; no SaaS/refresh/bearer credentials to untrusted code | §8 `TaskAuthority`, §9.6, §11.3, §14 credential boundary, §15, §17 secrets split, §18 credential-isolation tests + gates, §19 P0/P2/P5/P6, §23.1(6) | Yes — downscoped-token exception removed everywhere; proxy-issued authority carries TTL/scope/budget/audit |
| A2 | Crash-safe provisioning: durable Neon task registry, idempotent provision/reconcile, lease reaper | §9.6 registry + lifecycle, §11.3, §17 tasks row + reaper, §18 reaper tests, §19 P0/P5 | Yes — intent-first ordering and UNKNOWN-marking reaper specified |
| A3 | Policy-confirmed barrier before Sprite execution | §8 sketch comment, §9.6 `POLICY_APPLIED`, §11.3, §18 policy tests, §19 P0/P5 | Yes — read-back + digest match + fail-closed + re-check rule |
| A4 | Content-addressed deployment profiles in immutable artifacts | §9.1, §9.6 promotion record, §13 `validate`, §17, §18 profile-binding tests, §19 P0/P4, §23.1(7) | Yes — rejection rule covers validate/publish/promote/admission |
| A5 | Adapter phase ownership (Sprite workspaces P4; Machines adapter + Sprite tasks P5; P6 hardens) | §7 annotations, §16, §17, §19 preamble + P0/P4/P5/P6 | Yes — every `deploy/` component has exactly one implementing phase |
| A6 | Trusted-build promotion (OCI + lock + SBOM + provenance), snapshot never the artifact | §9.6, §14 promotion review, §15 (unchanged protocol), §17 promotion row, §18 promotion tests, §19 P4/P5, §20 release-builds row | Yes — build tooling added to the version register as TBD |
| A7 | Named security owner; G0 sign-off names approver | §19 P0 tasks + artifacts, §23.1(9), Appendix B Q9 | Yes — owner identified as pending, not invented |
| A8 | Sprite scope clarity (preview in scope; untrusted-code tasks later infra; generator out) | §2 scope paragraph, §19 preamble | Yes — no task execution pulled forward |
| A9 | Full hybrid failure matrix in `hybrid-task-demo` suite | §16 demo row, §18 hybrid failure tests (7 cases), §19 P5 | Yes — no second demo added, per consolidated rejection |
| A10 | Retention as first-class policy | §9.4, §17 operate view, §19 P0 (requirements) / P2 (input) / P3 (River) / P4 (artifact) | Yes — defined before multi-tenant GA |
| A11 | DBOS availability/recovery ownership; no HA claims | §17 (design + alerts), §19 P0/P1/P6, §21 K21 | Yes — measured thresholds gate any scale-up |
| A12 | Explicit catalog→phase mapping; harness scope (one in P1, matrix in P6) | §16 Phase column + `agent-toolchain` default, §19 P1–P5 packages, §18 (unchanged pyramid) | Yes — every catalog entry maps to exactly one phase or Deferred |
| A13 | G0 failure dispositions per load-bearing spike | §19 P0 tasks/acceptance/exit, §23.1, §23.2 (6 dispositions) | Yes — failure forces re-plan, never waiver |
| A14 | CLI freeze without tool duplication; migration ordering; SDK re-verify; prompt2 bound to G0; `TaskStatus` naming | §12 migration ordering, §13 CLI rule + `validate` row, §8 `GetTaskStatus`, §19 P0/P3, §23.1(11), Appendix B Q8 + D status | Yes — all five seams closed in plan text |
| A15 | Go binary/registry model decided in Phase 0 (shared host vs binaries, static vs runtime registration, identity→implementation mapping, discovery/routing, build/release/rollback); owner-directed default: shared host, static registration, independently versioned manifests | §5 ADR-016, §7 `registry/` + `miniapps/`, §8 registry note, §19 P0, §23.1(12) | Yes — default recorded; independent binaries kept as future option only |
| A16 | Golden-path lifecycle binding e2e: clean checkout → verified production deployment → upgrade/rollback/retire, CLI and GitHub Actions CI/CD on shared services/policies | §5 ADR-017, §6.4, §13 CI/CD parity, §17 retire, §18 golden-path tests, §19 P0/P1/P4/P5/P6 | Yes — design in P0, incremental per the timing table, verified in P6 |

Consolidated rejections honored: no direct bearer-token compromise into
Sprites; no second hybrid demo for failure coverage; no Sprite
task/promotion implementation in Phase 0.

### Follow-up review (A15/A16, narrow scope)

Checked 2026-11-22 against the frozen baseline:

1. A15 default vs ADR-002 (one module), ADR-003 (shared runtime),
   ADR-004 (one invocation service), and R7 (manifest as contract):
   consistent — static registration in the shared host strengthens all
   four; no decision reopened.
2. A15 vs A4/A6: independently versioned manifests with content-addressed
   profiles and trusted-build artifacts compose without conflict; the
   registry routes by manifest identity + immutable version in both.
3. A16 vs A14 CLI rule: CI/CD calls the same services as the CLI, adding
   no parallel implementation and no tool duplication.
4. A16 vs gates: design lands in Phase 0, dogfooding in Phase 1, preview
   and publication in Phase 4, deployment and promotion in Phase 5, and
   full verification in Phase 6 — matching existing phase ownership; no
   phase gains work its dependencies cannot support.
5. No new module dependencies (GitHub Actions is a service, not a Go
   dependency): §20 needs no new row. No new residual risks beyond the
   existing K-set: A15 is decided before G0 passes, A16 is verified at G6.
6. Numbering: ADR-016/017 appended, G0 conditions appended as 12–13;
   existing cross-references (§23.1(9), Appendix B Q8/Q9) unaffected.

The A15/A16 follow-up review remains approved as a historical disposition. This revision selectively incorporates review2 findings that are valid and compatible with the owner-approved product direction; the Appendix F table is the final finding-by-finding disposition. Phase 0 remains approved; Phase 1 waits on G0-Core; Sprite preview/tasks/promotion remain phase-gated and subject to A1–A6.

## Appendix F. Consolidated disposition of adversarial review 2 (F01–F24)

This appendix is the binding reconciliation of `plan4.1_adv_review2.md` with
this plan. “Accept” means the plan incorporates the invariant or required
proof. It does not mean the behavior has already been implemented or proven.

| Finding | Disposition | Consolidated decision |
|---|---|---|
| F01 — shared host vs DBOS workflow versioning | **Accept with refinement** | Retain one Go module, static registration, shared gateway, and same image by default. Isolate each Class D family by DBOS `AppName` and explicit application version; retain old family artifact/workers until drain. Separate Fly apps only if per-family release isolation cannot be proven. |
| F02 — scope inflation / remove Sprites | **Partially accept** | The Phase 0/G0 coupling was too broad. Keep the owner-approved Sprite/lifecycle roadmap, but make it optional and phase-gated. Sprite API failure cannot block G0-Core/Phase 1. Do not remove A1–A16 wholesale. |
| F03 — SSRF and custom REST egress | **Accept** | Add operator-approved origins, HTTPS, connection-time DNS/IP checks, private/reserved/metadata/Fly-internal blocking, redirect revalidation, bounded responses/timeouts, and adversarial probe suite before G2 custom REST. |
| F04 — external writes before reconciler/approvals | **Accept** | Move `crm-write` to Phase 4. Phases 1–2 use read-only real integrations and dry-run writes only. First live write requires effect ledger, reconciler, approval, limits, and crash tests. |
| F05 — `confirm` semantics | **Accept with refinement** | Approval is a policy overlay, not a new execution class. Single-step A/B uses a durable approval table and inline/River dispatch; D waits through DBOS events. Approval is input-hash/authority/expiry bound and out-of-band with separation of duties. |
| F06 — uniform async Class A | **Accept** | Class A is sync-first and returns inline; persist invocation/audit identity. B/C/D return run handles. A tool needing durable wait/retries is not Class A. |
| F07 — Class C progress and River limits | **Accept as a pilot requirement, not an engine mandate** | River global concurrency controls are Pro functionality; DBOS Go queues provide global/partitioned concurrency and rate limits. Compare the actual Class C forcing workload before selecting. |
| F08 — entity serialization | **Accept** | Require manifest `entity_key` or reviewed `no_entity`; A uses row/advisory lock, B/C combines unique identity and locks, D may use DBOS partition concurrency one. Test same-key order and cross-key parallelism. |
| F09 — ADR-001 enforcement | **Accept** | Port ADR-001 to the actual Go/River stack. Class C/D need workload-level decision records; automated checks validate completeness/arithmetic, but humans decide business consequences/net complexity. |
| F10 — blast-radius controls | **Accept** | Add shared per-principal × tenant × capability rate, value, volume, record, recipient and concurrency limits; circuit breakers and no-redeploy kill switches. |
| F11 — untrusted tool results | **Accept** | Mark provider free text untrusted; return typed/capped/paginated results; never treat provider content as policy/instructions; poisoned-fixture tests. |
| F12 — SaaS permission pass-through/re-auth/approver channel | **Accept** | Enforce scope/privilege intersection, declared attribution, typed `NEEDS_REAUTH`, out-of-band reauth and approval channels, and separation of duties. |
| F13 — Neon pooling, PITR, cost | **Accept with evidence qualification** | Use transaction-local tenant context; DBOS direct initially; calculate connection budgets; measure autosuspend/polling cost; stop/restore/mark UNKNOWN/reconcile/resume runbook before GA. |
| F14 — Fly overlap and static egress | **Accept as proof/conditional control** | Test release overlap and DBOS ownership/fencing; no incompatible concurrent recovery; drain old versions. Static egress IP is allocated only when a real provider requires allowlisting and current product/price are verified. |
| F15 — phase overload and late golden path | **Partially accept** | Split G0-Core from River/provider/Sprite/Fly-extension gates. A16 remains incremental: Phase 1 dogfood, Phase 4 preview/publish, Phase 5 deploy/promote, Phase 6 end-to-end retire. One harness in Phase 1; two-harness matrix in Phase 6. |
| F16 — dependency pins/licensing | **Accept** | Keep Phase 0 pinning; record source, version, license, date, and upgrade owner. Do not import Enterprise modules without explicit owner approval. |
| F17 — Go DBOS determinism lint | **Accept as staged control** | Add a Go analyzer/lint rules and replay tests for nondeterministic workflow constructs, using APIs from the pinned SDK. Make strict mode a CI gate before Class D production. |
| F18 — manifest/build drift | **Accept with non-circular binding** | Release envelope binds manifest digest to implementation/build/image digest and source revision. Registry refuses mismatch. Do not put a self-referential binary digest inside a manifest hash without a defined non-circular procedure. |
| F19 — “verified” means reread only | **Accept** | Separate design-consistency review from behavioral proof. Every gate cites evidence artifact, test, commit, environment, command, result, and date. |
| F20 — retention vs erasure | **Accept** | Keep PII out of DBOS payloads/checkpoints; define record-class retention and erasure/de-identification; test erasure while preserving only legally justified audit evidence. |
| F21 — wrapper-shaped catalog/result sizing | **Accept with refinement** | Miniapps map to business outcomes, not provider endpoints. Keep `crm-read` as a narrow forcing fixture; enforce result size caps and pagination. |
| F22 — identity sprawl | **Partially accept** | Keep the approved two-principal model. A Sprite task actor is a credential-less scoped workload identity derived from the authority, not a third human/business principal. |
| F23 — inferred prompt requirements | **Accept** | Q8 remains owner-gated. Appendix D inferences stay explicitly provisional until owner supplies the remainder or accepts Appendix D. No unverified REST gap APIs may be assumed. |
| F24 — SQLite dev backend | **Defer** | Optional local fast loop may be evaluated later. Real Postgres remains mandatory for transaction, concurrency, recovery, and crash acceptance; no SQLite dependency is added to G0. |

### Evidence sources checked 2026-11-22

- DBOS Go workflow versioning and recovery: https://docs.dbos.dev/golang/tutorials/workflow-tutorial and https://docs.dbos.dev/golang/tutorials/upgrading-workflows
- DBOS Go application ownership and per-application workflow/queue/version boundaries: https://docs.dbos.dev/golang/reference/configuration and https://docs.dbos.dev/golang/reference/methods
- DBOS Go global/partition queue concurrency and rate limits: https://docs.dbos.dev/golang/reference/queues
- River Pro global/partition concurrency is a Pro feature: https://riverqueue.com/docs/pro/concurrency-limits
- Neon transaction-pooling constraints and `SET LOCAL` usage: https://neon.com/docs/connect/connection-pooling
- OWASP DNS-rebinding-aware SSRF prevention: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- Fly deployment strategies and static egress IP product/pricing: https://fly.io/zero-downtime-deploys/ and https://fly.io/docs/about/pricing/

These sources support the stated platform/documentation facts, not a claim
that this repository has already implemented or tested them. Fly DBOS
multi-version rollout behavior, selected SDK pins, pool-handle atomicity,
provider OAuth/terms, and actual deployment-region cost remain Phase 0–6
proof obligations as assigned above.
