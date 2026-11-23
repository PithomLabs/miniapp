## prompt

I want Miniapp to be the business logic layer for Go apps but since observability is a cross-cutting concern, would adding authentication and authorization add to the complexity of the project? my thinking is that we are exposing Miniapp as API, MCP and CLI so we might as well protect it from rogue AI agents, humans, bots and whatnot? I mean security should be designed from the get go, not bolted on after

## answer

Your thinking is right. In fact, I would elevate **security to the same foundational architectural level as observability**, and I would establish its contract before accelerating Classes B and C.

There is one important qualification: **Miniapp should own security enforcement, but it should not try to become a complete identity-management product.**

That distinction keeps the architecture manageable.

## **1. Yes, authentication and authorization belong in Miniapp's foundation**

Miniapp is evolving beyond a Go library that executes business logic. It is becoming a business logic and execution layer accessible through multiple interfaces:

- API — applications and services.
- MCP — AI agents and agent platforms.
- CLI — developers, operators, and automated scripts.
- Background execution — River workers and DBOS workflows.

Each interface introduces a different entry point, but they all reach the same underlying business capabilities.

That creates a fundamental security requirement:

**It must not matter whether someone invokes a business operation through HTTP, an MCP tool, a CLI command, or a background worker. The same authorization rules must ultimately govern what that operation is allowed to do.**

Otherwise, the weakest interface becomes a backdoor into the entire framework.

## **2. Separate four concepts that are easy to conflate**

I would make these explicit in the architecture.

| Concept              | Question it answers                                                | Example                                                                          |
| -------------------- | ------------------------------------------------------------------ | -------------------------------------------------------------------------------- |
| Authentication       | Who is making this request?                                        | A human, an AI agent, or a service                                               |
| Authorization        | What is this identity permitted to do?                             | Read a customer, issue a refund, approve a payment                               |
| Business rules       | Is this operation valid under the application's rules?             | A refund is within the permitted refund period                                   |
| Effect authorization | Is this particular external side effect authorized to execute now? | This provider call is still permitted under the current grant and approval state |

The fourth concept is particularly important.

The existing `authorizeAttempt` work in Miniapp's Stage 3A addresses authorization at the consequential external-effect boundary. That is valuable, but it does not automatically establish who may access the API, invoke an MCP tool, or administer a tenant.

We must preserve that distinction.&#x20;

## **3. A unified Miniapp security architecture**

I recommend the following architecture.

**External identities**

Human

AI agent

Service

Authentication adapters

**API**

HTTP adapter

**MCP**

Tool adapter

**CLI**

Command adapter

Verified identity and delegation context

**Miniapp Security Kernel**

**Principal resolution**

Identity, tenant, actor

**Authorization**

Capabilities, policies, resource access

**Delegation**

Agent acting on behalf of a principal

**Credential broker**

Controlled secret access

Authorized business operation

**Classes A–D**

Execution engines

**Effect boundary**

Authorization and approvals

**Cross-cutting audit and observability**

Every relevant security and execution transition is attributable and diagnosable.

The key architectural decision is that API, MCP, and CLI are adapters, not independent security systems. They authenticate callers and translate verified identities into a common Miniapp security context.

## **4. AI agents deserve special treatment**

I agree with your concern about rogue agents, but I would avoid treating an AI agent as inherently trusted or inherently untrusted.

Treat it as a distinct security principal with explicit, limited authority.

For example:

A user delegates an AI agent permission to:

- Read customer records.
- Prepare a refund.
- Create a support ticket.

But the agent is not automatically allowed to execute the refund.

The security context should preserve at least:

| Field           | Meaning                                          |
| --------------- | ------------------------------------------------ |
| `principal_id`  | Identity on whose behalf the action is performed |
| `actor_id`      | The actual agent or user making the request      |
| `tenant_id`     | The tenant boundary                              |
| `delegation_id` | The authorization under which an agent acts      |
| `capabilities`  | The permitted operations                         |
| `expires_at`    | When the authority expires                       |
| `trace_id`      | Diagnostic correlation                           |

This separation is essential. An agent must not be able to claim that it represents an administrator merely by supplying an administrator's identifier.

I would also require:

- Short-lived, narrowly scoped agent credentials.
- Explicit delegation and revocation.
- Tenant isolation.
- Per-tool and per-operation authorization for MCP.
- Approval gates for sensitive operations.
- Reauthorization where necessary when a background worker actually performs a consequential action.
- Complete attribution of agent actions to both actor and delegating principal.

Human confirmation alone must not substitute for authentication or authorization.

## **5. Do not build an entire IAM platform**

This is where I would actively control complexity.

Miniapp should not initially build its own:

- User directory.
- Password management system.
- SSO provider.
- Identity federation service.
- OAuth authorization server.
- Enterprise identity administration console.

Instead, it should support integration with existing identity providers and host applications.

For example:

A Go application might already authenticate users through an OIDC provider. Miniapp should be able to consume that application's verified identity and apply its own business-operation authorization rules.

For an external MCP connection, the MCP adapter can use an appropriate supported authentication mechanism and translate the verified identity into the same security context.

For CLI access, the CLI should obtain an explicitly authenticated user or workload identity rather than silently assuming every local process is trusted.

**Miniapp owns the authorization contract and enforcement. The application or identity provider can own identity issuance.**

I would start with explicit capabilities and a small authorization policy interface, not introduce a complicated policy language or policy engine before real requirements justify it.

## **6. How this affects the execution classes**

Security must follow execution, not merely the original request.

| Class | Security responsibility                                                                                               |
| ----- | --------------------------------------------------------------------------------------------------------------------- |
| A     | Authenticate and authorize the caller before executing the transactional operation; preserve tenant and actor context |
| B     | Preserve authorization provenance across enqueue, worker execution, retries, and lease turnover                       |
| C     | Enforce authorization at pipeline and item boundaries, especially when work spans multiple records                    |
| D     | Preserve identity, delegation, authority, and approval context through durable waits, signals, replay, and recovery   |

One particularly important rule: **A job or workflow must never become more privileged merely because execution moved from an authenticated API request to a background worker.**

The worker must receive a trustworthy representation of the original authority, and sensitive operations must still be subject to the appropriate execution-time authorization checks.

## **7. Revised roadmap priorities**

I recommend that Miniapp now have three foundational cross-cutting contracts:

**1. Security Contract**

Critical

Identity, authentication, authorization, delegation, capabilities, tenant boundaries, secrets, and effect authorization.

**2. Observability Contract**

Critical

Audit, structured logs, metrics, traces, correlation, export, diagnostics, and SLOs.

**3. Execution Contract**

Critical

Shared execution lifecycle and reliability guarantees, implemented without flattening the differences between A, B, C, and D.

The recommended sequence becomes:

1. Finish Prompt28 and Gate2 without expanding its scope.
2. Define the Security Contract and Observability Contract.
3. Implement their shared framework foundations against the existing execution paths.
4. Accelerate Class B and C with both contracts integrated from the beginning.
5. Verify security and observability conformance across A–D.
6. Expose API, MCP, and CLI capabilities under the common security model.

## **My conclusion**

I would make this a governing Miniapp principle:

> **No business capability is exposed without an explicit security boundary, and no execution capability is considered production-ready without attributable, observable execution.**

This is not needless complexity. It is the cost of making Miniapp a trustworthy business logic layer rather than simply another Go workflow library.

And I would be particularly strict about one thing: **MCP must not become a privileged backdoor into Miniapp.** An AI agent should be able to do only what its verified identity, delegated authority, and current authorization permit, regardless of what tools it can discover.

My recommendation is to formalize Security as a foundational architectural workstream alongside Observability, before B/C implementation—not to postpone it until the API, MCP, and CLI interfaces are already in use.

I’ll treat this as a standing architectural decision for Miniapp:

**Security and Observability are co-equal, critical foundational capabilities.**

The locked principles are:

1. Miniapp owns authorization enforcement; identity providers and host applications may own identity issuance.
2. API, MCP, CLI, and background workers must use a common security context and authorization model.
3. Human, AI-agent, and service identities must be distinguishable and attributable.
4. Delegation, least privilege, tenant isolation, credential protection, revocation, and execution-time authorization are foundational requirements.
5. MCP must never become a privileged backdoor.
6. Security and observability contracts must be established before Class B/C implementation.
7. Classes A–D must satisfy both contracts while preserving their distinct execution and recovery semantics.
8. No business capability is exposed without an explicit security boundary; no execution capability is production-ready without attributable, observable execution.
9. Prompt28 and Gate2 remain unchanged in scope.

I’ll use these as governing architectural constraints in subsequent Miniapp planning, implementation prompts, and adversarial reviews.