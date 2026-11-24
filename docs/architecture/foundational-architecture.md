# Miniapp Foundational Architecture (Unified)

Status: PROPOSED architecture (Prompt32 synthesis). Nothing in
this document claims existing implementation unless tagged
[Existing]. This is the unification document: the three
contracts are perspectives on the one runtime designed here.

Capability tags (used across the whole package):
[Existing] = implemented and tested in-tree;
[Partial] = implemented with known gaps;
[Documented] = specified but not implemented;
[Proposed] = new design in this package.

Provenance: `docs/prompt32.md` §§1–2, 6, 9–10;
`docs/plan4.2.md` (ADR-001–017, §§6–7, 10–11, R1–R23);
`docs/phase1/invocation-lifecycle-contract.md` (M1–M6,
C6, lock order); `docs/phase2/authorize-attempt-design.md`
(C1–C7); `docs/phase0/threat-model.md` (extended in §6).

Companion documents: `security-contract.md` (S),
`observability-contract.md` (O),
`developer-experience-enforcement.md` (DX),
`foundational-conformance-matrix.md`,
`foundational-roadmap.md`.

Doc map (single ownership — cross-references point here):

| Concern | Owning document |
|---|---|
| Identity, authorization, delegation, credentials | S |
| Audit, logs, metrics, traces, correlation | O |
| Programming model, enforcement, escape hatches | DX |
| Unified runtime, interactions, threat model, glossary, inventory | This doc |
| Conformance suites and matrices | Conformance matrix |
| Sequencing, milestones, residuals, decision register | Roadmap |

## 1. Executive rationale

Miniapp's execution foundations (invocation lifecycle,
effect authorization, attempt fencing, audit) are proven
for the shared boundary and the Class A/D paths. What is
missing is everything that makes those guarantees
*automatic*: today each new surface, adapter, or
capability must re-remember identity, authorization,
telemetry, and recovery. That is a linear accumulation
of cognitive load and a standing invitation to build a
backdoor by omission.

The program therefore designs three contracts as one
runtime architecture:

- Security establishes who may do what (S).
- Observability establishes what happened, why, and
  how (O).
- Developer Experience ensures neither responsibility
  is forgotten or manually reimplemented (DX).

The developer-visible result must feel like one
coherent Miniapp runtime — register a capability,
declare its permissions, write business logic —
while the framework supplies and enforces the rest.
Internal complexity is acceptable and expected;
developer-facing complexity is the metric this
architecture minimizes.

What this program does not do: redesign working
mechanisms (the inventory in §5 forbids it),
implement anything (architecture-only per §12),
flatten execution classes, build an IAM product, a
telemetry backend, or a dashboard product.

## 2. One-runtime architecture

```text
                 human / agent / service / trigger
                              |
     +------------+-----------+-----------+------------+
     |            |           |           |            |
 API adapter  MCP adapter  CLI adapter  worker pickup  approval channel
 (REST)       (session)    (token)      (B/C/D)        (out-of-band)
     |            |           |           |            |
     +------------+-----+-----+-----+----- +------------+
                              |     |
                    +-------------------------------+
                    | MINIAPP RUNTIME KERNEL        |
                    |                               |
                    |  S: authenticate → resolve →  |
                    |     admit → authorize → audit |
                    |  O: correlate → instrument →  |
                    |     export (OTLP)             |
                    |  X: dispatch → claim → fence →|
                    |     reconcile → settle        |
                    +---------------+---------------+
                                    |
        +-------------+-------------+-------------+-------------+
        |             |             |             |             |
   Class A       Class B       Class C       Class D      Settle (ops)
   tx handler    River job     pipeline      DBOS wf      operator only
   [Existing]    [Proposed]    [Proposed]    [Existing]   [Existing]
        |             |             |             |
        +-------------+------+------+-------------+
                                    |
                    +-------------------------------+
                    | CANONICAL EFFECT BOUNDARY     |
                    | authorizeAttempt (C1–C7)      |
                    | ledger + fencing + audit     |
                    | [Existing — unchanged]        |
                    +---------------+---------------+
                                    |
                        capability proxy + broker
                        provider credentials live here
                        [Proposed — tokens/, egress]
                                    |
                              providers
```

Reading the diagram: surfaces authenticate and
translate; the kernel admits, authorizes, records,
and dispatches exactly once per invocation; class
adapters execute with distinct semantics; every
consequential external write funnels through the
one existing effect boundary; provider secrets
never leave the broker. Audit rows commit with
state in PostgreSQL; telemetry flows out through
OTLP to operator backends.

## 3. Cross-contract interaction

S, O, and DX interlock at seven lifecycle points.
The table names the owner of each behavior; the
rule is that no point is owned by two contracts.

| # | Lifecycle point | S owns | O owns | DX owns |
|---|---|---|---|---|
| 1 | Surface authentication | Adapter verification, Caller resolution | Auth spans + metrics | Adapter interface; startup refusal if unconfigured |
| 2 | Admission | Capability + trust + tenant + approval linkage | Admission audit + spans | Registration façade lowering to admission |
| 3 | Dispatch / worker pickup | Authority-record load + live delegation/grant re-check | Pickup audit + queue-latency + trace link | Executor adapter interface |
| 4 | Execution (handler/step) | In-tx context (A); checkpoint re-verify (B/C/D) | Tx/handler spans, error attributes | Handler signature; Workflow Kit primitives |
| 5 | Effect authorization + send | `authorizeAttempt` (unchanged) + broker use-tokens | Effect/provider spans + attempt metrics | `Effect` primitive lowering to the boundary |
| 6 | Recovery / reconcile / settle | Settle role + non-oracle rules | UNKNOWN metrics, reconcile spans, settle audit | Recovery invisible to handlers; settle CLI thin |
| 7 | Export | Redaction rules + credential-free views | OTLP pipeline + failure behavior | Exporter config surface + test exporter |

Interaction invariants (normative):

- I-1: Security decisions are always audited (S
  defines the decision, O defines the record, DX
  makes emission unavoidable).
- I-2: No telemetry path can alter a security
  decision or a business outcome (O §4.4).
- I-3: No DX convenience can widen authority or
  skip a lifecycle point (DX §6–§8).
- I-4: Correlation IDs flow from the security
  resolution into every emission (O §5.1).
- I-5: Every new lifecycle point needs all three
  columns filled before it ships (conformance
  gate).

Admission sequence (all classes, all surfaces):

```text
caller → adapter.Authenticate → kernel.Resolve → kernel.Admit
   → kernel.Authorize(capability+trust+tenant) → approval?
   → persist invocation + authority record → dispatch
   │         │              │                     │
   O: session│ span    O: admission         O: dispatch
   │         │ audit    span + audit         span + run link
   │         │              │
   └── S: opaque Caller ────┴── S: permission set ──┘
```

Worker pickup sequence (B/C/D):

```text
worker → load authority record → live delegation check
  → live grant check → checkpoint audit+span → execute
  │         │                    │
  │         └── S: refuse if     └── O: pickup span
  │             revoked/expired       linked to trace
  └── DX: adapter interface; no ambient authority
```

## 4. Execution-class integration

Each class satisfies all three contracts without
flattening its semantics. Per-class obligations:

| Concern | A (tx) | B (job) | C (pipeline) | D (workflow) |
|---|---|---|---|---|
| Invocation + identity | In-request Caller [E] | Authority record → worker [P] | Record → items [P] | Record → wf context [P] |
| Authorization checkpoints | Admission [E] | Admission + pickup [P] | Admission + pickup + item [P] | Admission + start + pre-effect [P] |
| Transaction boundary | Single pg tx [E] | Enqueue tx + job tx [P] | Item txs + progress [P] | Steps + RunAsTx [E] |
| Audit events | Terminal + refusal [E] | + pickup/retry [P] | + item outcomes [P] | + waits/signals/replay [P] |
| Logs/metrics/traces | Request/tx [P] | Queue/job [P] | Batch/item [P] | Wf/checkpoint [P] |
| Retry/recovery | Same-key re-entry [E] | Bounded retries [P] | Resume w/o replay [P] | Replay + recovery [E] |
| Conformance | Shared + A suite [P] | Shared + B suite [P] | Shared + C suite [P] | Shared + D suite [P] |
| Failure behavior | Rollback all [E] | Dead-letter + audit [P] | Partial + resume [P] | Pause + reconcile [E] |

Legend: [E] = exists today (mechanism proven, needs
contract conformance wiring); [P] = proposed in this
package (some mechanisms exist, e.g. DBOS adapter,
but contract integration is new).

Non-flattening rules: A's single-transaction
atomicity is never simulated with queues; B's jobs
never inherit workflow replay; C's progress is
explicit record state, never DBOS checkpoints
unless the workload's ADR-001 record justifies
DBOS; D's waits/signals/replay stay DBOS-owned. Class A
keeps bounded same-key recovery with a stable
BusinessOpID (C6 rule, unchanged). Global lock
ordering and the separate authorization
transaction (C1–C3) are preserved under all new
instrumentation and checkpointing.
B/C adapters are greenfield: they implement the
contracts from their first working versions (no
retrofit debt by construction).

## 5. Existing-capability inventory (normative appendix)

Every proposed capability maps to one row. [Existing]
and [Partial] rows cite the source or contract that
proves them; nothing here redesigns a working
mechanism.

### Security

| Capability | Status | Evidence |
|---|---|---|
| Two-principal identity (AgentID + Authority) | [Existing] | `contract/policy.go` Caller; `auth.Binding` |
| Credential→binding authentication | [Existing] | `auth.Policy.Authenticate` + tests |
| Trust modes + grants (tenant→miniapp→mode) | [Existing] | `auth` grants; ConfigHash; staleness model |
| Kill switches (5 scopes) | [Existing] | effects kill-switch matrix (P28 WS-A) |
| Settle operator boundary (exact role, non-oracle) | [Existing] | `runtime.Service.SettleEffect`; P28.1 + P31 review |
| Effect authorization (locks, predicates, fencing) | [Existing] | `authorizeAttempt`; C1–C7; P28 closeout |
| Durable approvals + DecideApproval | [Existing] | `approvals/`; material hash; expiry |
| REST/MCP/CLI authentication | [Partial] | Header/args/flag credentials; demo keys only |
| Blast-radius quotas / breakers / value caps | [Absent] | Plan-level only (plan4.2 §14.2) |
| External IdP integration | [Absent] | — |
| Delegation model | [Absent] | — |
| Capability permissions | [Absent] | Trust modes are the coarse gate |
| Worker re-authorization checkpoints | [Absent] | — |
| Token/credential broker | [Absent] | `tokens/` empty |
| SSRF-safe egress | [Documented] | plan4.2 §14.1; no code |
| OAuth/permission intersection/NEEDS_REAUTH | [Documented] | plan4.2 §14; no code |
| Out-of-band approval channel | [Documented] | Service layer exists; channel UX absent |
| Revocation drill | [Documented] | Procedure exists; NOT EXECUTED |

### Observability

| Capability | Status | Evidence |
|---|---|---|
| Append-only attributed audit | [Existing] | `audit.Service`; refusal/withdrawal/settle pairing |
| Effect ledger as debug surface | [Existing] | M3 states + lineage + UNKNOWN quarantine |
| Determinism analyzer (prevention) | [Existing] | `analyzer/determinism`; CI-strict |
| Structured logs | [Partial] | slog alongside writes; no pipeline |
| Metrics | [Absent] | No registry, no exporter |
| Distributed tracing | [Absent] | No OTel dep; explicitly deferred |
| OTLP export / Collector | [Absent] | — |
| Dashboards / alerts / SLOs | [Absent] | — |
| Redaction framework | [Partial] | Hash excludes creds; no telemetry rules |
| Sampling / retention policy | [Documented] | A10 retention; nothing implemented |
| Error taxonomy | [Partial] | `contract.ErrorKind` 3 kinds; no attributes |

### Developer experience and enforcement

| Capability | Status | Evidence |
|---|---|---|
| Static manifest + registry + ToolCall | [Existing] | `registry/`; ToolFunc; release envelope |
| Analyzer + AST guards + checks.sh + preflight | [Existing] | P24/P26/P28/P29 guard chain |
| testkit fixtures | [Existing] | ClassA manifest/impl/envelope helpers |
| Thin CLI invoke group | [Existing] | invoke/status/signal/cancel/settle |
| Business Capability façade | [Absent] | Proposal in DX §4 |
| Runtime Kernel boundary (designed) | [Absent] | Convergent today, not a designed API |
| Startup validation framework | [Absent] | — |
| Adapter conformance suites | [Absent] | probe/classd is the precedent pattern |
| Escape-hatch mechanism | [Absent] | — |
| Class D Workflow Kit | [Absent] | DBOS adapter exists; kit is new |

### Execution substrates

| Capability | Status | Evidence |
|---|---|---|
| Class A tx executor | [Existing] | `executor/classa`; C6 fixture |
| Class D DBOS executor | [Existing] | `executor/dbos`; probe conformance |
| Class B River adapter | [Absent] | `executor/river/` empty |
| Class C pipeline | [Absent] | Plan-level only (River-first) |
| Sequence subsystem (M5) | [Documented] | Deferral + activation requirements |
| Real provider integration | [Absent] | P24-008 gate; mock only |

## 6. Threat model (extends Phase 0)

Base: `docs/phase0/threat-model.md` (scope: platform as
frozen in plan4.2; trust boundaries agent→platform,
platform→provider, runtime→Sprite, workspace→prod,
tenant↔tenant; sign-off recorded in the G0-Core
evidence). This section adds threats introduced by the
new surfaces and mechanisms in this package; the base
boundaries are unchanged.

| # | Threat | Boundary | Control (owner) |
|---|---|---|---|
| T-1 | Agent forges/principal-swaps identity | Agent→platform | Opaque server-resolved Caller; live delegation (S §3.3, §4.4) |
| T-2 | MCP tool invoked beyond session authority | Agent→platform | Session auth + per-op admission (S §7) |
| T-3 | Worker acts on revoked/expired authority | Execution | Pickup + checkpoint re-verify (S §5) |
| T-4 | Credentials exfiltrated via telemetry/audit/errors | All | Emission redaction + broker views (S §3.4/§8; O §6.1) |
| T-5 | Stale grant/delegation abused | Platform | Layered revocation + drill (S §4.5) |
| T-6 | Approval captured by requester (self-approval) | Channel | Identity-compare enforcement (S §4.6) |
| T-7 | Telemetry outage masks attack | Ops | Drops counted/logged; audit independent (O §4.4) |
| T-8 | Cardinality bomb via crafted IDs | Telemetry | Closed metric registry; ID-label ban (O §5.2) |
| T-9 | Bypass via direct provider call from handler | Execution | Static guard + no runtime path (DX §6 mechanism 5; S §6) |
| T-10 | Silent downgrade (unconfigured security) | Deploy | Startup refusal (DX §6 mechanism 2) |
| T-11 | Escape hatch becomes a backdoor | DX | Declaration + justification + tests (DX §7) |
| T-12 | Replay exhibits stale "allowed" | D execution | Checks re-read authoritative state (S §5.1) |

Residual risks carried (not introduced here):
single-executor availability (A11); providers
without idempotency (conservative quarantine);
harness approval-UX variance (server enforcement
is the control); demo identity until S-D1 lands.

## 7. Glossary (canonical for the package)

- Authority: the resolved permission set under which
  an invocation executes (permissions ∩ grants ∩
  delegation). Server-derived, never requested.
- Authority record: the persisted authority bound to
  an invocation for worker checkpoints.
- Business Capability: one registered business
  operation with identity, types, class,
  authorization, handler, and policies.
- Caller: both principals of an invocation (agent
  client + authority). See `contract.Caller`.
- Conformance: a tested guarantee; an adapter that
  fails shared or class suites is not
  production-ready.
- Delegation: a live, scoped, expiring grant for an
  agent to act for a principal.
- Effect boundary: the canonical
  `authorizeAttempt` + ledger path; the only route
  from execution to provider side effects.
- Invocation: one admitted business operation
  (M1 record); identified by `invocation_id`.
- Run: one execution lifecycle of an invocation
  (M4 record); identified by `run_id`.
- Runtime Kernel: the one framework execution
  boundary all surfaces and workers pass through.
- Telemetry: logs, metrics, and traces shipped to
  external backends; never authoritative.
- Audit: durable attributable records in
  PostgreSQL; authoritative for what happened.

## 8. Migration and compatibility

1. No execution-path rewrite: the Capability
   façade lowers to `registry.Register` +
   `ToolFunc`; the kernel formalizes the path
   adapters already converge on; S/O
   instrumentation wraps existing lifecycle
   points. Existing suites (X-01–X-24, C1–C7,
   M1–M6, P28/P28.1/P30) stay green unmodified
   — a program-level acceptance gate.
2. Manifest evolution is additive: `permissions`,
   `escapeHatches`, delegation references are new
   optional sections; old manifests validate
   unchanged. Breaking manifest changes need a
   versioned migration with dual-read.
3. Static registration (ADR-016) is preserved;
   the façade generates static registrations.
4. B/C are greenfield: no migration, only the
   rule that first versions satisfy the
   contracts (enforced by conformance gates in
   their implementation work orders).
5. Credential migration: demo keys → versioned
   credentials → IdP-backed identity is staged;
   each stage refuses to serve surfaces it
   cannot secure (no mixed-trust serving).
6. Rollback: every implementation milestone is
   independently revertible (see the roadmap);
   contract docs version with the code they
   specify.

## 9. Risks and decision index

Top risks: (a) six-doc drift — contained by the
doc map (§0), single glossary (§7), and the
coverage audit (roadmap acceptance); (b) B/C
schedule pressure skipping conformance —
contained by making suites a gate in their work
orders, not a follow-up; (c) OTel dependency
weight — contained by the exporter abstraction
(O §4.3) and the no-shared-fate rule; (d)
façade leakage (registration struct growth) —
contained by the small-surface rule + owner
review on additions; (e) aspirational claims —
contained by the status-tag discipline (§10 of
each contract + audit checklist).

Owner decisions: S-D1–S-D6 (S §10), O-D1–O-D6
(O §9), DX-D1–DX-D6 (DX §11), C-D1–C-D3
(conformance matrix), R-D1–R-D3 (roadmap).
Consolidated register with sequencing in
`foundational-roadmap.md` §4.
