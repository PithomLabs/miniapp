## prompt

just to clarify, Miniapp is becoming a huge library with a lot of rules as cognitive load for a dev not to mention the security and observability roadmap ahead, how do we ensure that these are being used and enforced? I mean the framework should ensure that these are all incorporated automatically, the goal is that devs are supposed to write business logic only with the Miniapp framework taking care of all the necessary details automatically

## answer

I think you have identified one of the most important architectural challenges in Miniapp. And I would go further: **Miniapp should be designed so that developers cannot accidentally omit critical framework responsibilities simply because they forgot to follow a rule.**

The framework should not become a library of  fifty things developers must remember to use. It should become an environment in which the correct behavior is the default.

Your desired developer experience is:

> I write my business logic. Miniapp takes care of execution, security, authorization, reliability, observability, and the operational details.

That should become a governing design principle.

# **1. The fundamental principle: framework-owned mechanics, application-owned business decisions**

We need a very clear division of responsibility.

| Responsibility                               | Owner                           |
| -------------------------------------------- | ------------------------------- |
| Customer eligibility rules                   | Application                     |
| Refund calculation                           | Application                     |
| Order fulfillment decisions                  | Application                     |
| Business-specific approval criteria          | Application declares the policy |
| Authentication integration                   | Miniapp adapters                |
| Identity verification and propagation        | Miniapp                         |
| Authorization enforcement                    | Miniapp                         |
| Tenant isolation                             | Miniapp                         |
| Transaction management                       | Miniapp                         |
| Workflow execution and recovery              | Miniapp                         |
| External-effect authorization and recording  | Miniapp                         |
| Audit                                        | Miniapp                         |
| Structured logging                           | Miniapp                         |
| Metrics and distributed tracing              | Miniapp                         |
| Credential protection                        | Miniapp                         |
| Execution-context propagation across workers | Miniapp                         |

There is a subtle but important distinction here.

Miniapp cannot invent a business's authorization rules. The application must tell it, for example, which people are allowed to approve refunds.

But once the application declares that policy, Miniapp should enforce it automatically.

**The application declares intent. The framework guarantees enforcement.**

# **2. Build a Miniapp Runtime Kernel**

I recommend making this the central architectural concept.

The Runtime Kernel is the common execution boundary through which every registered business capability passes, regardless of whether it is invoked through API, MCP, CLI, or a background worker.

## **Miniapp Runtime Kernel**

One execution boundary. Multiple invocation surfaces.

**API**

**MCP**

**CLI**

Verified invocation context

Identity · Tenant · Delegation · Correlation

## **Runtime Kernel**

**Security enforcement**

Authentication, authorization, tenant isolation

**Observability**

Audit, logs, metrics, traces

**Execution management**

Transactions, claims, lifecycle

**Reliability**

Retries, recovery, durable execution

**Effect protection**

Authorization, ledger, attempt fencing

**Credential protection**

Secrets, token access, delegation

**Execution adapter**

**Class A / B**

Synchronous / one-shot job

**Class C / D**

Pipeline / durable workflow

**Application business logic**

The developer's primary responsibility

The important feature is that these are not six optional libraries developers must manually assemble.

They are capabilities integrated into the same invocation lifecycle.

A business operation cannot accidentally skip the authorization boundary or fail to generate its trace context because the developer forgot to add middleware.

# **3. Developers should register capabilities, not assemble infrastructure**

I would simplify the public programming model around a concept called a **Business Capability**.

A capability represents something the application can do: issue a refund, create an order, approve a timesheet, process a payroll batch.

It declares the business operation and the requirements under which it may execute.

The following is illustrative future API design, not a claim about Miniapp's current exported API:

```
app.Register("refund.create", miniapp.Capability[RefundRequest, RefundResult]{
    Class: ClassA,

    Authorization: miniapp.Require("refund.create"),

    Execute: func(ctx context.Context, req RefundRequest) (RefundResult, error) {
        // Actual business logic.
        return calculateAndCreateRefund(ctx, req)
    },
})
```

Notice what the developer does not implement:

- Authentication middleware.
- Tenant extraction.
- Transaction begin and commit.
- Invocation IDs.
- Trace creation.
- Structured logging.
- Metrics recording.
- Audit infrastructure.
- Retry and recovery machinery.
- Effect-ledger manipulation.
- Provider authorization.

Miniapp supplies these through its execution machinery.

The application still declares which permission is necessary, and writes the actual refund rules. Miniapp enforces the declared permission.

The key is to keep the registration surface small. We should resist creating dozens of configuration fields for every internal framework concern.

# **4. Enforce automatically at multiple levels**

This is the most important part of your question.

Documentation and coding conventions are not sufficient. We need several independent enforcement mechanisms.

**Level 1 — API design and compile-time constraints**

Make it difficult to construct invalid invocation contexts or bypass framework-owned boundaries through ordinary public APIs. Use typed contracts, opaque trusted identity values, and restricted interfaces.

**Level 2 — Startup validation**

Verify required security providers, capability declarations, execution adapters, telemetry configuration, and policy consistency. Fail startup when a required security guarantee is missing. Do not silently downgrade.

**Level 3 — Runtime enforcement**

Every invocation passes through the common execution boundary. Authorization, tenant isolation, audit, effect protection, and observability are enforced by the framework at the appropriate lifecycle points.

**Level 4 — Framework-owned conformance tests**

The framework supplies tests that verify each execution adapter preserves the security, reliability, and observability contracts. Developers do not have to invent these tests themselves.

**Level 5 — Static analysis and CI enforcement**

Detect bypasses that cannot be prevented through ordinary API design, such as direct provider SDK calls from business handlers, missing capability declarations, or prohibited nondeterministic workflow operations.

This produces an important distinction:

**A framework guarantee should be mechanically enforced, not merely documented.**

There is one limitation we must acknowledge: Miniapp is a Go library, and application code can still import arbitrary Go packages. We cannot guarantee that arbitrary trusted Go code will never bypass a provider boundary. We can make the supported programming model safe by default, with restricted capabilities, static analysis, and controlled escape hatches. Strong isolation of hostile code requires an actual process or sandbox boundary.

# **5. Observability should be invisible to the application developer**

This is where I would be especially opinionated.

The developer should not have to write:

```
logger.Info("refund started")
metrics.Increment("refund_started")
tracer.Start(...)
audit.Record(...)
```

for every operation.

Miniapp should automatically instrument:

- Invocation start and completion.
- Authorization decisions.
- Transaction outcomes.
- Worker enqueue and execution.
- Retry and recovery.
- Approval waits.
- Effect authorization and provider calls.
- Unknown outcomes and reconciliation.
- Workflow checkpoints and replay.

The framework generates consistent logs, metrics, traces, and appropriate durable audit records.

The developer adds diagnostic events only when business-specific context genuinely helps explain a decision.

For instance, Miniapp can automatically tell us that a refund invocation failed. The application may optionally tell us that it failed because the customer's refund window had expired.

That is the distinction between framework instrumentation and domain diagnostics.

# **6. Security should also be invisible, but never implicit in authority**

The same approach applies to security.

A developer should not manually check tenant ownership in every handler or pass an operator identity through arbitrary function arguments.

Miniapp supplies a verified invocation context and enforces its declared permissions.

For example:

```
Incoming MCP request
       |
       v
Authenticate agent
       |
       v
Resolve principal and delegation
       |
       v
Verify tenant
       |
       v
Authorize capability
       |
       v
Execute business handler
       |
       v
Authorize consequential effect
       |
       v
Persist outcome and audit
```

No transport should get a shortcut.

One qualification matters: the framework cannot know that a particular business action requires a second human approver unless the application declares that requirement. But once declared, the framework owns the approval mechanism, state tracking, enforcement, and audit.

The same principle applies to the Settle operator role we just approved.

The application should not implement the same security logic independently in API, MCP, and CLI. Each adapter should route through the common service boundary.

# **7. Avoid turning Miniapp into a giant public API**

Here is a danger I want to highlight.

We could easily solve the complexity problem by adding more framework abstractions, and accidentally make it worse.

I recommend three rules:

**First: internal complexity is acceptable; developer-facing complexity is not.**

Miniapp can have dozens of internal components. The application developer should interact with a small, stable programming model.

**Second: one authoritative execution path.**

Do not create separate API, MCP, CLI, and worker implementations of business authorization, audit, effect protection, or business execution.

**Third: escape hatches must be explicit.**

There will be legitimate cases where advanced applications need low-level control. But bypassing a framework guarantee must require an explicit mechanism, a documented reason, and tests demonstrating that required invariants remain intact.

There should be no silent opt-outs.

# **8. Make conformance part of the definition of a framework feature**

This is how we ensure future features do not gradually erode the architecture.

When someone adds Class B, the framework should automatically supply the common invocation context, authorization, structured logging, metrics, tracing, and audit hooks.

When someone adds Class C, those same guarantees should apply across batches and individual items, without confusing pipeline progress with durable workflow checkpoints.

When someone adds Class D functionality, the instrumentation and identity context must survive durable waits, restarts, replay, and recovery.

A new adapter that doesn't implement the contracts should fail its conformance suite.

I would formalize this in a **Miniapp Framework Conformance Suite** that runs in CI and is reusable by every execution adapter.

It should test actual guarantees rather than merely checking whether an interface exists.

# **9. The next architectural work should be Developer Experience and Enforcement Architecture**

I recommend one additional deliverable alongside the two contracts we already prioritized.

## **Miniapp foundational architecture**

**1. Security Contract**

Critical

Define identity, authority, delegation, authorization, tenant boundaries, credentials, and enforcement.

**2. Observability Contract**

Critical

Define audit, logs, metrics, traces, correlation, export, diagnostics, and operational guarantees.

**3. Developer Experience and Enforcement Architecture**

Critical

Define how Miniapp makes both contracts automatic, mandatory, testable, and nearly invisible to business-logic developers.

**4. Execution adapters A–D**

Required

Each execution class must satisfy all three foundational contracts.

The third deliverable is not another layer of features. It is the architecture that ensures the first two actually work as intended.

I would also require a small demonstration application as an acceptance test: a developer builds a realistic business capability using Miniapp, exposes it through API, MCP, and CLI, and does not manually implement security middleware, tracing, metrics, audit plumbing, or execution recovery.

We should then deliberately attack that application with unauthorized identities, cross-tenant requests, interrupted executions, retries, and ambiguous provider outcomes. The demonstration should prove that the framework catches these problems without relying on the developer remembering dozens of rules.

## **My strongest recommendation**

Make this a permanent Miniapp doctrine:

> **The developer writes business logic. Miniapp owns everything required to execute that business logic safely, reliably, securely, and observably. Framework guarantees must be enforced by construction, not delegated to developer discipline.**

And I would add one more:

> **The complexity of the framework must not leak into the complexity of the application.**

That is the difference between Miniapp becoming a powerful collection of Go libraries and becoming a genuinely productive business-logic framework.

The next work should therefore include a formal **Miniapp Developer Experience and Enforcement Architecture** specification, coordinated with the Security and Observability Contracts, before accelerated Class B/C implementation. No existing execution guarantees should be weakened to achieve a simpler API.

**Locked as a governing architectural doctrine for Miniapp.**

The following decisions will guide future architecture reviews, roadmaps, implementation plans, and coding-agent prompts.

## **Miniapp Developer Experience and Enforcement Doctrine**

**1. The developer writes business logic. Miniapp owns the execution mechanics.**

Developers should not have to manually assemble authentication, authorization, tenant isolation, transactions, retries, recovery, audit, logging, metrics, tracing, or effect protection.

**2. Three co-equal foundational architecture workstreams**

- **Security Contract:** Identity, authentication, authorization, delegation, tenant isolation, credential protection, and effect authorization.
- **Observability Contract:** Audit, structured logs, metrics, traces, correlation, telemetry export, diagnostics, and operational guarantees.
- **Developer Experience and Enforcement Architecture:** Make Security and Observability automatic, mandatory, testable, and nearly invisible to application developers.

**3. One authoritative execution boundary**

API, MCP, CLI, and background workers must converge on the common Miniapp runtime and security context. No transport-specific shortcuts or privileged backdoors.

**4. Framework enforcement, not developer discipline**

Required guarantees must be enforced through multiple mechanisms:

- Typed APIs and compile-time constraints.
- Startup validation.
- Runtime enforcement.
- Framework-owned conformance tests.
- Static analysis and CI checks.

No silent opt-outs.

**5. Safe by default, explicit escape hatches**

Advanced applications may need lower-level control, but bypassing framework guarantees must be explicit, justified, and independently tested. Arbitrary trusted Go code cannot be made incapable of bypassing its own library; genuinely hostile code requires isolation.

**6. Classes A–D retain distinct execution semantics.**

Every execution class must satisfy the Security, Observability, and Developer Experience contracts. Class B and C must incorporate them from their first working implementations.

**7. Production-readiness rule**

> No business capability is exposed without an explicit security boundary. No execution capability is production-ready without attributable, observable execution and framework conformance.

## **Locked roadmap sequence**

1. Complete Prompt28 Gate 2 and the approved Settle operator work.
2. Define the coordinated Security and Observability Contracts.
3. Define the Developer Experience and Enforcement Architecture.
4. Implement shared foundations against existing A and D paths.
5. Accelerate B and C with all three foundations integrated.
6. Establish cross-class conformance for A–D.

The ultimate product principle is:

> **The complexity of the framework must not leak into the complexity of the application.**

I will treat this as a standing architectural constraint for Miniapp.