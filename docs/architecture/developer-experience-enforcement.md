# Miniapp Developer Experience and Enforcement Architecture

Status: PROPOSED architecture (Prompt32 workstream DX). Nothing
in this document claims existing implementation unless tagged
[Existing]. Tags: [Existing] / [Partial] / [Documented] /
[Proposed] (legend in `foundational-architecture.md`).

Provenance: `docs/roadmap_dx.md` (locked DX doctrine + five
enforcement levels + Runtime Kernel); `docs/roadmap_classd.md`
(Class D Workflow Kit direction); `docs/plan4.2.md` §§7–8, 13
(public API, CLI groups); current `registry/` + `ToolFunc`
sources (the baseline this proposal is evaluated against).

Companion documents: `security-contract.md` (workstream S),
`observability-contract.md` (workstream O),
`foundational-architecture.md` (unified runtime, glossary,
inventory), `foundational-conformance-matrix.md`,
`foundational-roadmap.md`.

## 1. Rationale

This workstream is the mechanism that makes the Security
and Observability Contracts real. Without it, S and O are
fifty rules developers must remember; with it, they are
properties the framework confers automatically.

Governing doctrine (locked):

> The developer writes business logic. Miniapp owns
> everything required to execute that business logic
> safely, reliably, securely, and observably.

> The complexity of the framework must not leak into the
> complexity of the application.

The acceptance bar is behavioral, not aesthetic: a
developer builds a realistic capability, exposes it on
API/MCP/CLI, and the framework — not the developer's
memory — supplies authentication, authorization, tenant
isolation, transactions, retries, recovery, audit,
logging, metrics, tracing, and effect protection. The
demo application (§10) proves this under adversarial
conditions.

## 2. Current state (DX inventory summary)

[Existing] Registration today is manifest + static
registry: `registry.Register(manifest, impl, envelope)`
with `ToolFunc(ctx, ToolCall)`, where `ToolCall` already
carries server-derived TenantID/Principal/InvocationID/
EntityKey/EffectToken plus the Class A tx. Manifests
(`miniapp.json`) declare identity, tools, trust modes,
approvals, and entity keys; the release envelope binds
manifest digest to implementation identity.

[Existing] Enforcement precedents: the determinism
analyzer (CI-strict for Class D), AST structural guards
(P24-009 policy-map guard, P28 Send-inventory guard with
negative control, P26-001/P29-003 trust-field guard),
`checks.sh` gates (gofmt, pins, enterprise imports,
miniapp-import boundary, fixtures), `preflight`
architectural checks, and `testkit` fixtures
(`ClassAManifest`, `ClassAImpl`, `RegisterClassATool`).

[Existing] CLI invoke group (`invoke/status/signal/
cancel/settle`, `auth verify-revocation`) is thin over
the canonical service; no lifecycle group yet.

[Partial] The current model already hides transactions
(Class A), attempt epochs, and effect state machines —
but developers still hand-assemble manifest +
registration + handler + approval/tool wiring per
surface, with no startup validation, no capability
permission declaration, and no conformance suite for
new adapters.

[Absent] Business Capability abstraction, Runtime
Kernel boundary as a designed API, startup validation
framework, framework-owned adapter conformance suite,
escape-hatch mechanism, Class D Workflow Kit.

## 3. Responsibility split (normative)

| Responsibility | Owner |
|---|---|
| Business rules, calculations, stage ordering | Application |
| Input/output types | Application |
| Which actions need approval / which permission | Application declares |
| Authentication integration (adapters) | Miniapp |
| Identity verification and propagation | Miniapp |
| Authorization enforcement | Miniapp |
| Tenant isolation | Miniapp |
| Transaction management | Miniapp |
| Workflow execution and recovery | Miniapp |
| Effect authorization, ledger, reconciliation | Miniapp |
| Audit, logs, metrics, traces | Miniapp |
| Credential protection | Miniapp |
| Execution-context propagation across workers | Miniapp |

The hinge sentence: the application declares intent
(policies, permissions, approvals); the framework
guarantees enforcement. Miniapp cannot invent a
business's refund-approval rule — but once declared,
the approval mechanism, state tracking, enforcement,
and audit are framework-owned.

## 4. Business Capability proposal [Proposed]

### 4.1 The abstraction

One registration per business operation. Illustrative
shape (design proposal, not an existing API):

```go
app.Register("refund.create", miniapp.Capability[RefundRequest, RefundResult]{
    Class: ClassA,
    Authorization: miniapp.Require("refund.create"),
    Policies: miniapp.Policies{
        Approval: miniapp.ApproveWhen(Over(10000)),
    },
    Execute: func(ctx context.Context, req RefundRequest) (RefundResult, error) {
        return calculateAndCreateRefund(ctx, req) // business logic only
    },
})
```

Declared: capability identity, request/response types,
execution class, required authorization, business
handler, business policies and approval requirements.
Never hand-assembled: auth middleware, tenant
extraction, transactions, invocation identity, audit,
logging, metrics, tracing, retry/recovery, ledgers,
provider authorization.

### 4.2 DX evaluation against implementation constraints

Per the review guardrail, the proposal is judged
against what the framework can actually build:

1. Static registration stays: `Register` runs at
   startup linking into the existing static
   registry (ADR-016) — no plugin hot-loading, no
   reflection-based DI. Generics (`Capability[Req,
   Resp]`) give compile-time request/response
   safety with zero runtime cost.
2. Class is load-bearing, not cosmetic: the class
   selects the executor adapter (existing classa/
   dbos; future river) and its transaction
   boundary. A Class A handler receives the
   admitted tx implicitly (as `ToolCall.Tx` does
   today); it cannot open a second one.
3. `Require(...)` compiles to manifest
   `permissions` + admission checks (S §4.1/§4.2);
   undeclared permissions fail startup validation,
   not first invocation.
4. `Policies.Approval` compiles to the existing
   approvals table + `DecideApproval` flow; the
   framework owns expiry/material-hash/authority
   re-verification.
5. Handler signature discipline: handlers take
   `(ctx, Req)` and return `(Resp, error)`; the
   framework injects verified identity, tenant,
   correlation, and (for D) the workflow context.
   There is no handler parameter through which
   authority can arrive — unifying with S §4.7.
6. Small surface rule: the registration struct
   grows only by owner-reviewed addition.
   Framework internals (attempt epochs, ledgers,
   spans) are never registration fields.

### 4.3 Migration from today's API

`registry.Register` + `ToolFunc` is not removed; the
Capability API is a typed façade that lowers to it.
Existing miniapps (`crm-read`) and all fixtures
keep working. New capabilities use the façade; the
façade is what startup validation and conformance
understand. No flag-day rewrite, no dual execution
path (one lowering, tested once).

## 5. Runtime Kernel: one execution boundary [Proposed]

All surfaces converge on one kernel invocation path:

```text
API adapter   MCP adapter   CLI adapter   worker pickup
     |              |              |              |
     v              v              v              v
+------------------------------------------------------+
| Miniapp Runtime Kernel (one boundary)                |
|  authenticate → resolve → admit → authorize → record |
|  security enforcement + observability emission       |
|  execution management + reliability + effect guard   |
+------------------------------------------------------+
     |              |              |              |
     v              v              v              v
  Class A        Class B        Class C        Class D
  adapter        adapter        adapter        adapter
     |              |              |              |
     +---------------> handler <---------------+
              (business logic only)
```

Kernel properties (normative): every invocation
passes the kernel — no surface may reach an adapter
directly (statically asserted: adapters import only
kernel-approved packages); authorization, tenant
isolation, audit, and telemetry emit at kernel
lifecycle points, never in adapter code; the kernel
is transport-agnostic (no HTTP/MCP/CLI types cross
its boundary).

## 6. Five enforcement mechanisms [Proposed]

1. Typed APIs and compile-time constraints.
   Opaque trusted values (`Caller`, invocation
   context) with unexported fields; capability
   registration checked at compile time where
   expressible (request/response types, class
   tags); authority-bearing parameters
   unconstructible outside the framework.
2. Startup validation. The host validates at boot:
   security providers configured, capability
   declarations ⊆ manifest permissions, execution
   adapters present per declared class, telemetry
   exporter configured, policy consistency
   (approval chains satisfiable, entity keys
   declared or explicitly excused). Failure refuses
   to serve — fail loud, never degrade silently.
3. Runtime enforcement. The kernel enforces
   authorization, isolation, audit, effect
   protection, and observability at the lifecycle
   points in O §7. Handler code cannot skip them:
   there is no call order that reaches execution
   without passing admission.
4. Framework-owned conformance suites. The
   framework ships the suites in
   `foundational-conformance-matrix.md`; every
   adapter (including future B/C) passes shared +
   class-specific suites. Developers run them; they
   do not author them.
5. Static analysis and CI enforcement. Extend the
   existing analyzer/guard/checks precedents:
   direct provider-SDK calls from handlers,
   missing capability declarations, authority-
   carrying request structs, undeclared metrics,
   ID-valued metric labels, telemetry-free
   lifecycle points. CI fails; no warn-and-ship
   for safety-critical rules.

## 7. Escape hatches [Proposed]

Advanced applications sometimes need low-level
control. Every bypass is explicit, justified, and
tested:

- Mechanism: a single `miniapp.Unsafe` (name
  illustrative) subpackage + manifest
  `escapeHatches[]` declaration naming the
  capability, the bypassed guarantee, and the
  justification. Use without declaration fails
  startup validation.
- Justification: reviewed text + issue/decision
  link, recorded in the evidence register.
- Invariant tests: the bypassing miniapp ships
  tests proving the bypassed guarantee still
  holds for its use (or that the residual risk is
  accepted by the owner in writing).
- No silent opt-outs: there is no boolean that
  disables authorization, audit, or effect
  protection. Global disable switches are
  forbidden by CI assertion.

## 8. Go realism boundary (normative)

Honest limits, stated plainly: Miniapp is a Go
library, and arbitrary trusted Go code in the same
process can import anything — `net/http` to a
provider, raw SQL past the ledger, `os.Exit` mid-
attempt. API design + startup validation +
conformance + static analysis make the *supported*
programming model safe by default and make bypasses
loud, reviewable, and testable. They cannot make
bypass *impossible* inside one process.

Genuinely hostile or untrusted code requires a
process or sandbox boundary: that is the Sprite
isolation track (plan4.2 Dimension B, A1–A6 gated,
Phase 4/5) — explicitly out of this program's
scope. This contract designs for trusted-but-
fallible developers, not adversaries in-process.
Conflating the two would produce either a false
safety claim or an unusable API; the spec makes
the boundary explicit instead.

## 9. Class D Workflow Kit [Proposed]

Adopting `roadmap_classd.md` as the authoring
target: a thin typed layer over DBOS exposing
`Step`, `Transaction`, `Effect`, `Approval`
(`AwaitApproval`), `Wait`, `Cancel`, `Child`, plus
deterministic helpers (`Now`, `NewID`, `Random`,
`Sleep`). Design constraints (non-negotiable):

- `Step` wraps at-least-once execution; only
  `Transaction` (atomic app-write + durability
  record) and the M3 effect ledger confer
  exactly-once business semantics. The kit never
  presents the two as interchangeable.
- `Effect` composes the existing `authorizeAttempt`
  + ledger + reconciliation services; it adds no
  parallel ledger.
- `AwaitApproval` composes the approvals table +
  DBOS messaging; expiry/authority/self-approval
  re-verification stays framework-owned.
- `Cancel` preserves the recorded ordering
  (durable intent → withdraw unattempted → report
  in-flight → request cancellation → reconcile)
  and the DETACH-ONLY attachment decision.
- Queue-partition serialization composes with —
  never replaces — generation fencing until an
  empirical proof retires a mechanism.
- Analyzer + conformance enforcement (mechanisms
  4–5) make raw-DBOS escape from ordinary
  workflows loud and reviewable.

Staging follows roadmap_classd (façade → business
primitives → scheduling → enforcement → three-
fixture reuse proof), sequenced in
`foundational-roadmap.md`.

## 10. Demo acceptance test design [Proposed]

A realistic business capability (e.g. refund
issuance with an approval threshold), registered
once, exposed through API + MCP + CLI, written
with zero manual security/observability/recovery
code. Adversarial matrix (each a scripted,
asserted scenario):

| Attack / fault | Framework must |
|---|---|
| Unauthenticated caller (all surfaces) | Identical terminal; zero state; audit |
| Unauthorized human / agent | Admission refusal with reason; audit |
| Delegation violation (expired/forged/scope-exceeded) | Refusal at admission or checkpoint |
| Cross-tenant access (ID swap, selector escape) | Fail closed; no oracle |
| Credential/identity spoofing | Untrusted input rejected; no authority conferred |
| Interrupted execution (kill -9 at each boundary) | Crash-boundary semantics; no duplicate confirmed effect |
| Retries and recovery (storm + takeover) | Fencing holds; single confirmation |
| Ambiguous provider outcome | UNKNOWN quarantine + evidence, never blind retry |
| Missing telemetry exporter | Business unaffected; drops counted; audit complete |
| Sensitive-data leakage (credential-shaped fixtures) | Absent from every emission |
| Canonical-boundary bypass attempt (direct provider call) | Static guard fails CI; runtime has no path |

Passing bar: the developer fixes zero scenarios by
editing the demo's business logic — every row is
resolved by framework behavior. Any row requiring
demo-side security/observability code is a DX
architecture failure, recorded as such.

## 11. Owner decisions required

| ID | Decision | Needed before |
|---|---|---|
| DX-D1 | Capability API package shape (`miniapp.Capability` façade vs new package) and naming | Façade implementation |
| DX-D2 | Startup-validation failure taxonomy (which classes refuse to serve vs degrade, per profile) | Startup framework |
| DX-D3 | `Unsafe` escape-hatch mechanism name/shape and review process | First escape hatch |
| DX-D4 | Demo application domain + scope (refund? onboarding? second demo?) | Acceptance-test build |
| DX-D5 | Conformance-suite distribution (in-module `conformance/` vs separate module later) | Suite implementation |
| DX-D6 | Analyzer-rule strictness schedule (which rules fail CI from day one) | Analyzer extension |

## 12. Acceptance criteria (DX implementation)

1. The demo app's business-logic files import no
   security, telemetry, ledger, or recovery
   packages (import-boundary conformance).
2. All §10 adversarial rows pass without
   demo-side framework-plumbing edits.
3. A new adapter failing any shared conformance
   test cannot be declared production-ready (CI
   gate, mechanically enforced).
4. Startup with an undeclared permission, missing
   adapter, or unconfigured production exporter
   refuses to serve (no silent downgrade).
5. Three distinct Class D fixtures reuse the
   Workflow Kit with no lifecycle SQL and no raw
   DBOS calls in ordinary paths (roadmap_classd
   reuse proof).
6. X-01–X-24 + C1–C7 + M1–M6 invariants hold
   unchanged under the façade (existing suites
   green unmodified).
7. Escape-hatch use without declaration +
   justification + invariant tests fails CI.