# **Consolidated review**

Consolidate — no need for a third architecture redesign

**My recommendation is to use&#x20;**`plan3_z.md`**&#x20;as the primary architectural blueprint and incorporate the strongest additions from&#x20;**`plan3_deepseek.md`**.** They are fundamentally aligned. `plan3_z` is more mature in its treatment of the framework, lifecycle, security, external effects and reference implementations. DeepSeek adds useful detail around the agent-facing MCP contract, asynchronous execution handles and the central miniapp router.

The combined architecture is sound, subject to resolving several technical contradictions before implementation.

One important scope distinction: the two deliverables should be **the reusable Go framework library and a catalog of reference miniapps built with that library**. They should be planned and tested separately. The framework must not become a collection of business-specific implementations, and the reference apps must demonstrate that ordinary developers can use the framework without rebuilding its infrastructure.

## **1. What survives from each plan**

| Architecture area                             | Primary source      | Disposition                                          |
| --------------------------------------------- | ------------------- | ---------------------------------------------------- |
| Agent invokes miniapp; miniapp owns execution | Both                | Adopt                                                |
| Go framework + DBOS + River + Fly.io + Neon   | Both                | Adopt, subject to compatibility tests                |
| Class A/B/C/D execution taxonomy              | Both                | Adopt                                                |
| DBOS adoption rubric                          | `DBOS(1).md` / both | Retain as mandatory policy                           |
| Declarative `miniapp.json`                    | DeepSeek            | Adopt and formalize                                  |
| Typed contracts and permission manifest       | `plan3_z`           | Adopt                                                |
| One central MCP router                        | DeepSeek            | Adopt                                                |
| REST/OpenAPI and CLI surfaces                 | `plan3_z`           | Adopt                                                |
| Common asynchronous run handle                | DeepSeek            | Adopt                                                |
| OAuth token broker and capability registry    | `plan3_z`           | Adopt                                                |
| External-effect protocol                      | `plan3_z`           | Adopt, with stronger unknown-outcome handling        |
| Saga compensation                             | Both                | Adopt as explicit framework support                  |
| Virtual Object/per-entity state               | Both                | Adopt as a pattern, not as an assumed DBOS guarantee |
| Fixtures and crash-boundary testing           | Both                | Adopt                                                |
| Reference miniapp catalog                     | `plan3_z`           | Retain as the reference implementation scope         |
| AI miniapp generator                          | DeepSeek            | Defer; not necessary for these two deliverables      |

The core product statement should be:

> A Go framework for building durable, permissioned, agent-invokable business-process miniapps, accompanied by reference implementations demonstrating common business automation patterns across SaaS and custom REST APIs.

This retains the important inversion in DeepSeek: the agent chooses and invokes a business capability; the miniapp owns credentials, execution, recovery and audit.&#x20;

plan3_deepseek.md



`plan3_z` adds the crucial insight that a miniapp is not merely a function. It is a governed automation unit with a contract, execution model, permission manifest, integration bindings, observability and acceptance tests.&#x20;

plan3_z.md



## **2. Corrections required before implementation**

These are the parts I would not copy verbatim.

### **A. River does not automatically become a workflow engine**

The classification is correct:

| Class | Execution                           | Governing rule                                         |
| ----- | ----------------------------------- | ------------------------------------------------------ |
| A     | Go handler + PostgreSQL transaction | No queue or workflow engine                            |
| B     | River                               | One-shot, independently retryable job                  |
| C     | River by default                    | DBOS only when checkpointed recovery genuinely matters |
| D     | DBOS                                | Durable business process                               |

River supports transactional enqueueing, retries, scheduling and unique jobs. Its job insertion can participate in the same transaction as an application data change.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1



However, `River chain` should not be treated as a built-in durable workflow abstraction. For Class C, design explicit job progress and per-record idempotency. Upgrade to DBOS when recovering individual stages, durable waits or cancellation/resumption would otherwise require substantial custom machinery.

This preserves the governing principle in your DBOS adoption policy.

### **B. DBOS durability does not mean exactly-once external effects**

This is a particularly important correction.

The current Go documentation makes the distinction explicit:

- A DBOS workflow resumes from its last completed step.
- A step can execute more than once if its side effect happens but its completion checkpoint is not committed.
- A completed step is not executed again on recovery.
- Database operations can use DBOS transaction integration for stronger transactional guarantees.&#x20;

  [image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

  DBOS Docs+2



Therefore, this statement from `plan3_z`:

> Zero duplicate external effects.

Needs a more precise acceptance condition.

I would define the external-effect state machine as:

```
PENDING
   |
   v
ATTEMPTING
   |
   +------> CONFIRMED
   |
   +------> FAILED
   |
   +------> UNKNOWN
                  |
                  v
             RECONCILIATION
                  |
          +-------+-------+
          v               v
       CONFIRMED        NOT_APPLIED
          |               |
          v               v
       COMPLETE        SAFE RETRY
```

A timeout after sending an HTTP request does not establish that the provider failed to execute it.

Consequently:

- Use stable business-operation and attempt identities.
- Supply provider idempotency keys whenever supported.
- Record provider operation IDs and responses.
- Quarantine unknown outcomes.
- Reconcile before retrying or compensating when the outcome cannot be established.

The actual reliability target is **no duplicate confirmed business effects, no silently lost operations, and no blind retries of unknown outcomes**.

### **C. DBOS Sleep, Fly auto-stop and multi-day workflows need an explicit design**

This is the largest infrastructure contradiction in the plans.

DBOS Go supports durable sleep, but suspending a sleeping workflow out of process memory requires `WithSleepSuspend`, which is documented as an Enterprise feature.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+1



Fly.io's automatic Machine start is triggered by incoming requests through Fly Proxy; it does not automatically guarantee that a database timer can wake a stopped DBOS executor.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://flyio-landing.fly.dev\&sz=32)

Fly Docs+1



Therefore, the statements that Fly Machines can simply stop during long-running waits and that the workflow will later resume are not yet a complete operational architecture.

The implementation plan must compare:

1. Open-source DBOS with an always-available executor for durable scheduling and waits.
2. DBOS Enterprise suspension and coordinated recovery.
3. An external always-on wake-up/scheduling component capable of starting the appropriate worker.

It must demonstrate what happens when every relevant Machine is stopped, the wake-up time arrives, and no HTTP request is received.

Do not design the product's core workflow guarantee around an unproven assumption about Fly autostart.

### **D. Neon connection strategy needs to be explicit**

The plans propose a shared Neon database for application state, DBOS and River. That is reasonable, but the connection strategy cannot be uniform.

DBOS's production guidance says transaction-mode connection pooling is unsuitable because DBOS uses PostgreSQL features such as `LISTEN/NOTIFY`.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+1



Neon's normal pooled endpoint uses PgBouncer in transaction mode.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub



I would therefore plan for:

| Workload             | Initial connection approach                                 |
| -------------------- | ----------------------------------------------------------- |
| DBOS system database | Neon direct connection                                      |
| Application database | Direct or pooled, based on transaction/session requirements |
| River                | Direct or pooled after compatibility testing                |
| Schema migrations    | Direct connection                                           |

There is another important detail: DBOS's transaction integration must be configured correctly. Sharing a database URL is not necessarily equivalent to sharing the same underlying database pool handle. The plan must distinguish the DBOS system pool, application datasource and transaction-completion mechanism, and prove the intended atomicity with crash tests.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+1



### **E. The agent/user identity model needs a proper resolution**

There is an inconsistency between the two plans.

DeepSeek requires a user-delegated token for every tool call and rejects agent-only write authority. `plan3_z` introduces an agent principal with an API key or client credentials, along with a separate human identity and OAuth connections.

I would combine these into a two-principal authorization model:

**Agent principal**

Authenticates Codex, Claude Code, Hermes, OpenClaw or another authorized client.

**Authority principal**

Defines whose authority permits the operation:

- A delegated human user.
- An explicitly approved tenant service identity.
- An authorized system or scheduled trigger.

  **Authorization decision**

The platform checks the tenant, miniapp version, resource scope, connection permissions, operation risk and approval requirements.

This accommodates both interactive and unattended business automation without accidentally granting an AI agent broad administrative authority.

OAuth token exchange under RFC 8693 is useful where the identity provider supports it. It must not be assumed to be available from every SaaS provider.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.rfc-editor.org\&sz=32)

rfc-editor.org



Likewise, refresh-token rotation must account for provider-specific support.

### **F. Do not assume&#x20;**`auth.user_id()`**&#x20;exists in ordinary Neon PostgreSQL**

The DeepSeek plan describes RLS policies that obtain the tenant or user ID from `auth.user_id()`.

That is not a built-in Neon PostgreSQL function. The implementation plan must define the actual tenant-context mechanism, including transaction-local context, policy creation, `FORCE ROW LEVEL SECURITY` and database-role restrictions.

RLS is defense in depth, not a replacement for the application's authentication and authorization model.

### **G. Entity state and Saga are cross-cutting patterns**

The plans correctly identify per-entity execution state and Saga compensation, but there is a technical distinction.

A DBOS workflow ID of `tenant:entity:kind` can provide a stable workflow identity. It does not by itself guarantee that every concurrent operation on that business entity is serialized.

Use PostgreSQL uniqueness, row locking, optimistic concurrency or another proven mechanism for entity invariants. Use DBOS for orchestration and durable progress.

Likewise, compensation must be explicit and idempotent. A failed or unknown external outcome must not automatically trigger a compensating operation that might itself cause business damage.

### **H. Do not make one Fly application per miniapp the default**

`plan3_deepseek` recommends a shared Fly app with process groups, while `plan3_z` contemplates a per-miniapp Fly app or shared deployment.

I would choose a **shared platform runtime by default**, with separately deployable reference miniapp binaries or modules. Dedicated Fly applications or Machine groups should be available when isolation, customer requirements, workload profiles or independent deployment justify them.

The deployment plan should not create a separate MCP server for every miniapp. Both plans correctly identify the need for a centralized registry and router.

## **3. The consolidated architecture I would approve**

**Deliverable 1 — Go miniapp framework**

Contracts · Runtime · Execution adapters · Capabilities · Authorization · Effects · Audit · Test kit

**Common invocation layer**

MCP router · REST/OpenAPI · CLI · Registry · Run handles

**Class A**

Go handlers

**Class B/C**

River

**Class D — DBOS**

Durable workflows, waits, external events, approval and recovery

**Capability proxy and security layer**

Agent identity · delegated authority · OAuth · API credentials · approval policies · external-effect protocol

**Fly.io + Neon PostgreSQL**

API/MCP · River workers · DBOS executor · application state · execution state · audit

**Deliverable 2 — Reference miniapps**

Lead follow-up · Appointment journey · Approval flow · Contact import · Data sync · Reporting · Notifications · CRM operations · Other plan3_z examples

The framework should expose three invocation surfaces, all using the same execution and authorization path:

| Surface      | Role                                      |
| ------------ | ----------------------------------------- |
| REST/OpenAPI | Canonical platform API                    |
| MCP          | Native tool interface for agent harnesses |
| CLI          | Developer and coding-agent interface      |

I would use a central MCP router with namespaced miniapp tools, not one independently configured MCP server per miniapp.

For asynchronous operations, the handle convention from DeepSeek is worth keeping. An agent starts an operation, receives a `run_id`, then retrieves its status or sends a signal through a separately authenticated tool.&#x20;

plan3_deepseek.md



One update: the official Go MCP SDK has moved forward with the MCP `2026-07-28` protocol revision. Its current documentation includes a stateless protocol model and Streamable HTTP support. I would use the official SDK, pin a version and test the actual target harnesses rather than assume they all support the same elicitation or task features.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://go.sdk.modelcontextprotocol.io\&sz=32)

MCP Go SDK+2



## **4. Deliverables and reference implementations**

### **Deliverable 1 — reusable Go framework library**

The library should own the reusable platform capabilities, including:

| Framework component        | Responsibility                                                                     |
| -------------------------- | ---------------------------------------------------------------------------------- |
| Contract package           | Miniapp manifest, tool descriptors, JSON Schemas, input/output and error contracts |
| Registry                   | Miniapp discovery, registration, version and dependency manifest                   |
| Runtime                    | Invocation lifecycle, validation, dispatch and run handles                         |
| Executor adapters          | Class A, River Class B/C, DBOS Class D                                             |
| Capability framework       | Approved SaaS and custom REST API operations                                       |
| Identity and authorization | Agent principals, delegated authorities, policies and trust modes                  |
| Token broker               | Credential resolution, provider authentication and refresh                         |
| Effect protocol            | Idempotency, attempts, reconciliation and compensation                             |
| Approval framework         | Durable approval requests, decisions and audit                                     |
| Observability              | Audit events, execution history and OpenTelemetry                                  |
| Agent interfaces           | MCP, REST/OpenAPI and CLI adapters                                                 |
| Test kit                   | Fixtures, contract tests, crash-boundary tests and strict policy validation        |

The framework should be useful to developers building miniapps independently. Avoid making the business reference examples dependencies of the framework.

### **Deliverable 2 — reference miniapps from&#x20;**`plan3_z.md`

I would retain the full catalog, but not implement everything at the same time.

| Reference implementation | Class | Priority                                       |
| ------------------------ | ----- | ---------------------------------------------- |
| `crm-read`               | A     | First                                          |
| `crm-write`              | A     | First, after write authorization tests         |
| `lead-followup`          | D     | First durable workflow                         |
| `notify-send`            | B     | Early                                          |
| `report-generate`        | B     | Early                                          |
| `contact-import`         | C     | Early                                          |
| `data-sync`              | C     | Early                                          |
| `approval-flow`          | D     | Early                                          |
| `appointment-journey`    | D     | After first durable workflow                   |
| `campaign-orchestration` | D     | Gated on the external-effect protocol          |
| `agent-toolchain`        | D     | Later, subject to separate scope justification |

The `agent-toolchain` example deserves particular attention. It should demonstrate how a durable miniapp can manage an authorized, resumable series of agent/tool operations. It must not duplicate the agent harness's reasoning loop or turn the platform into another general-purpose agent runtime.

`campaign-orchestration` should remain gated until provider rate limits, idempotency, outcome reconciliation and recovery have been demonstrated on real external sends.

This retains the named examples in `plan3_z` without pretending that all of them are equal in risk or implementation priority.&#x20;

plan3_z.md



# **5. Prompt for the coding agent**

The next task should be **implementation planning only**, not coding. The coding agent needs to inspect the existing repository and produce an evidence-based plan for the two deliverables.

I have included the architecture reconciliation and the corrections above in the prompt so the coding agent does not blindly reproduce contradictions in the source documents.

# Task: Produce the Consolidated Implementation Plan for the Go Agent-Invokable Miniapp Framework

## Role

Act as the principal Go systems architect and an adversarial technical reviewer responsible for designing a production-quality, reusable framework for agent-invokable business-process automation.

You are planning two deliverables:

1. **A reusable Go miniapp framework library.**
2. **A catalog of reference miniapp implementations built using that library, following `plan3_z.md`.**

Your assignment is to inspect the existing repository, reconcile the supplied architecture documents, investigate the current dependencies and their actual capabilities, and write a detailed implementation plan.

**Do not implement the framework or reference miniapps in this task.**

The deliverable is an implementation plan that another coding agent or engineering team can execute without having to rediscover architectural decisions, invent missing contracts, or resolve contradictions hidden in the source documents.

---

## 1. Source documents and precedence

Read the following documents completely before planning:

- `plan3_z.md` — primary product and implementation architecture.
- `plan3_deepseek.md` — supplementary agent-invocation, MCP, manifest and security design.
- `rea.md` — miniapp execution-category patterns and their original Restate/Rea mapping.
- `DBOS(1).md` — selective DBOS adoption policy.
- ADR-001 and any amendments, if present in the repository.
- Review 12 reliability retrospective and related remediation documents, if present.
- Relevant existing OpenLevel architecture, security, deployment and testing documentation.

Use this precedence when resolving differences:

1. The explicit two-deliverable requirement in this task.
2. ADR-001 and its DBOS adoption policy.
3. `plan3_z.md` as the primary miniapp architecture.
4. `plan3_deepseek.md` for complementary agent-facing design.
5. `rea.md` for its reusable execution patterns.
6. Existing source code and evidence from the actual repository.

Do not silently merge incompatible claims. Record each material disagreement, the chosen resolution and its rationale.

---

## 2. Repository investigation is mandatory

Before designing the target structure, inspect the actual repository.

At minimum, investigate:

- Repository layout and module boundaries.
- `go.mod`, Go toolchain version and existing Go dependencies.
- Existing application code, business logic and domain models.
- Existing persistence and database migrations.
- Existing queue, workflow, outbox, idempotency and recovery mechanisms.
- Existing API, authentication, authorization and tenant-isolation code.
- Existing logging, tracing, audit and metrics.
- Existing Fly.io configuration and deployment conventions.
- Existing Neon/PostgreSQL integration and connection handling.
- Existing tests, integration harnesses and CI workflows.
- Existing ADRs, runbooks, `AGENTS.md` and engineering constraints.

Create an inventory of what already exists.

Classify each relevant existing component as:

- Reuse.
- Adapt.
- Replace.
- Retire.
- Investigate further.

Do not presume that existing TypeScript/Hono implementations can be reused directly in Go.

Do not assume existing reliability mechanisms are correct merely because they exist. Trace their actual transaction boundaries, failure behavior and invariants.

Inspect the repository's current state before proposing changes. Preserve its existing naming, module structure and documented decisions where they remain valid.

Do not invent repository paths, module names, source files, APIs or dependency versions.

---

## 3. The product and architectural objective

The product is an agent-invokable business-process automation platform.

An AI agent running within a harness such as Codex, Claude Code, Hermes or OpenClaw discovers and invokes a miniapp through a typed interface.

A miniapp is a governed, versioned business-process automation unit. It owns:

- Its input and output contract.
- Its execution lifecycle.
- Its permitted business capabilities.
- Its integration credentials through a trusted platform broker.
- Its authorization and approval boundaries.
- Its idempotency and external-effect protocol.
- Its durable execution when required.
- Its audit and recovery behavior.

The agent is the caller and may decide which miniapp to invoke, interpret user intent and coordinate independent tools.

The miniapp is responsible for safely executing the business process that it promises.

The agent must not receive provider credentials, own durable workflow state, or bypass the miniapp's server-side permission checks.

This is API-first business-process automation inspired by RPA. API automation is the default. Browser or desktop automation must not be introduced as an implicit fallback; it requires a separate, explicitly governed execution capability and security model.

Do not turn this project into:

- A general-purpose AI agent runtime.
- A browser automation framework.
- A UI/dashboard generator.
- A general BPMN replacement.
- A universal business rules engine.
- A new custom workflow engine intended to replace DBOS.
- A free-form LLM code execution system.

The deliverables are the framework and the reference miniapps.

Do not add an AI miniapp generator to the initial implementation scope unless the repository and source documents demonstrate that it is necessary for those two deliverables.

Generated or manually authored miniapps must not be allowed to execute arbitrary unreviewed Go, shell or JavaScript code.

---

## 4. Required technology constraints

The primary stack is:

- Go for the framework and reference implementations.
- DBOS Transact Go for workflows that qualify under ADR-001.
- River for Class B and default Class C background work.
- PostgreSQL through Neon.
- Fly.io for deployment.
- Official Go MCP SDK for the agent-facing MCP surface, subject to current compatibility verification.
- REST/OpenAPI for the canonical HTTP interface.
- CLI support for developers and coding-agent harnesses.

Investigate current versions, minimum Go requirements, APIs, licensing and feature availability.

Pin and justify exact versions in the implementation plan. Never use `latest` as a production dependency strategy.

Verify current DBOS Go SDK APIs directly against its source and documentation. Do not copy TypeScript or Python DBOS examples and assume equivalent Go APIs exist.

At the time of this task, investigate at least:

- DBOS Go workflows, steps, durable sleep, events, queues, recovery and transaction integration.
- River transactional enqueueing, uniqueness, retries, scheduling, job lifecycle and failure inspection.
- Official Go MCP SDK transports, protocol compatibility, authorization and elicitation.
- Neon direct and pooled database connection behavior.
- Fly Machine process groups, health checks, deployment behavior and automatic start/stop.

Record the date of verification and source links.

---

## 5. Deliverable One: reusable Go framework library

Design the public framework library before designing the individual reference miniapps.

The plan must specify its package boundaries, responsibilities, public APIs, dependency direction, lifecycle and testing strategy.

At minimum, investigate and design the following framework components.

### 5.1 Contracts and manifest

Define a versioned declarative manifest, provisionally called `miniapp.json`.

Specify:

- Miniapp identity and version.
- Human-readable description and examples.
- Execution category.
- Input and output schemas.
- Tool descriptors and naming.
- Execution substrate.
- Capability dependencies.
- Connector versions.
- Permission and trust-mode policies.
- Approval requirements.
- Idempotency requirements.
- Timeouts, retry limits and cancellation.
- External-effect declarations.
- Saga compensation declarations.
- Audit requirements.
- Fixture references.
- Exact dependency manifest.
- Compatibility and migration information.

Evaluate whether Go structs should generate JSON Schema and identify the library/tooling used.

Require automated tests that prove Go contracts, published schemas, MCP tool schemas, OpenAPI definitions and documentation remain consistent.

### 5.2 Miniapp registry and runtime

Define:

- Registration lifecycle.
- Miniapp discovery and version resolution.
- Manifest validation.
- Input validation.
- Authorization before dispatch.
- Execution admission.
- Synchronous and asynchronous invocation.
- Run handles.
- Status inspection.
- Cancellation and signal delivery.
- Result retrieval.
- Terminal error representation.
- Recovery and operator intervention.

Define a common execution result contract.

An asynchronous invocation should return an execution identifier and a stable status interface rather than holding an MCP or HTTP request open for hours or days.

Do not give active runs arbitrary short-lived handle expirations. Define terminal execution-history retention separately.

### 5.3 Execution substrate adapters

Implement the abstraction and decision rules for the following classes.

| Class                                       | Required substrate                          |
| ------------------------------------------- | ------------------------------------------- |
| A — ordinary operation                      | Plain Go handler and PostgreSQL transaction |
| B — simple background job                   | River                                       |
| C — bounded multi-step background operation | River by default; DBOS only when justified  |
| D — durable business workflow               | DBOS                                        |

The execution selection mechanism must enforce ADR-001.

Do not use DBOS just because an operation is asynchronous, scheduled or has several steps.

Class C must have an explicit progress model, per-record or per-operation idempotency and a recovery contract. Do not claim that River supplies a durable multi-step workflow abstraction unless verified against its actual APIs.

For Class D, document the DBOS workflow and step discipline, execution identity, version compatibility and recovery strategy.

### 5.4 Identity, authorization and trust modes

Design a complete authorization model.

It must distinguish:

- Agent/service principal.
- Delegated human user, when present.
- Tenant.
- Authorized service identity, when explicitly approved.
- Miniapp version.
- SaaS connection.
- Capability and operation scope.
- Resource/record scope.
- Trust mode.
- Approval authority.

Support interactive and authorized unattended processes without confusing agent identity with user authority.

Do not require an interactive human user for every legitimate scheduled or event-driven workflow. Such execution must instead have an explicitly granted tenant/service authority and bounded permissions.

Do not allow an agent identity to silently inherit broad administrator privileges.

Model the trust modes from `plan3_z.md`:

- `dry-run`
- `confirm`
- `autonomous`

Trust mode is bound to an approved miniapp version and tenant policy. The calling agent must not be able to weaken the configured policy.

A server-side policy check is mandatory for every consequential action.

MCP tool annotations, host confirmation prompts and elicitation are supplementary interaction mechanisms, not security enforcement.

### 5.5 Capability and connector framework

Define a `CapabilitySpec` or equivalent contract for approved operations against SaaS and custom applications.

At minimum:

- Provider and operation identity.
- API version.
- Input and output schema.
- Authentication method.
- Connection reference.
- OAuth scopes or API permissions.
- Tenant and resource scope.
- Read/write classification.
- Risk class.
- Rate limits.
- Timeout.
- Retry semantics.
- Idempotency contract.
- Error taxonomy.
- Audit requirements.
- Test fixtures.
- Compensation or reconciliation operation, where applicable.

Design the connector registry and token broker.

Support SaaS OAuth and custom REST APIs. Treat imported OpenAPI specifications as untrusted candidates requiring endpoint, authentication, permission and network-egress review.

Investigate credential encryption, key management, token refresh, revocation, rotation and secret redaction.

OAuth token exchange under RFC 8693 must be conditional on actual identity-provider support. Do not assume every SaaS provider implements it.

Credentials must never be stored in workflow inputs, serialized step payloads, audit logs or MCP outputs.

### 5.6 External-effect protocol

Design the effect protocol as a shared framework primitive.

It must handle:

- Stable business-operation IDs.
- Stable attempt IDs.
- Provider idempotency keys.
- Effect intents.
- Pre-effect persistence.
- Attempt and outcome records.
- Provider request IDs.
- Confirmed success.
- Confirmed failure.
- Unknown outcome.
- Reconciliation.
- Compensation.
- Quarantine.
- Operator intervention.

Explicitly distinguish workflow durability from external provider guarantees.

DBOS step execution is not exactly-once external execution. Prove the crash window between the provider completing an action and the framework durably recording its outcome.

An unknown external result must not be blindly retried or automatically compensated unless an idempotency guarantee or authoritative reconciliation makes that safe.

Define how compensation registration, execution and failure recovery work.

### 5.7 Approval framework

Design durable approval requests and decisions.

An approval must bind to the exact:

- Miniapp version.
- Invocation.
- Requested operation.
- Target resource.
- Material input or effect-intent hash.
- Approver identity and authority.
- Expiry.
- Decision.

A material change to the requested operation must invalidate the earlier approval.

Define whether self-approval is prohibited for particular operation classes.

Approvals must remain enforced by the platform even when the MCP host does not support elicitation.

### 5.8 Audit and observability

Define structured audit events for:

- Invocation.
- Identity resolution.
- Authorization decision.
- Capability resolution.
- Workflow and job transitions.
- Approval requests and decisions.
- External effect attempts.
- Reconciliation.
- Compensation.
- Cancellation.
- Terminal outcomes.

Include tenant, actor, authority, miniapp/version, operation ID, execution ID and correlation ID.

Specify redaction, retention, access controls and data minimization. Do not store sensitive provider payloads indiscriminately.

Design OpenTelemetry traces from invocation through execution substrate, capability calls and external effects.

### 5.9 Agent-facing interfaces

Define one canonical invocation service and thin adapters.

Provide:

1. REST/OpenAPI.
2. Central MCP router.
3. CLI.

MCP must not execute a separate copy of the business workflow. It calls the same platform invocation service used by HTTP and CLI.

Avoid one independent MCP server per miniapp.

Evaluate two tool-discovery models:

- Generic discovery/describe/invoke/status tools.
- Dynamically registered namespaced tools such as `rpa.crm.create_contact`.

Choose the most interoperable design and explain tool-sprawl controls.

The MCP layer must expose only the miniapps and operations authorized for the calling principal.

Use the official Go SDK and verify actual transport and protocol compatibility with the target harnesses.

Do not make correctness or security dependent on optional task, elicitation or other client-specific features.

### 5.10 Framework test kit

Plan a developer test kit that makes it difficult to publish an unsafe miniapp.

Include:

- Manifest and schema validation.
- Contract tests.
- Fixture runner.
- Capability test doubles.
- Policy validation.
- Idempotency tests.
- Authorization tests.
- External-effect crash tests.
- Workflow replay checks.
- Recovery tests.
- Version compatibility tests.
- Tenant-isolation tests.
- Concurrency tests.
- Strict/warn policy modes.

Specify what the framework can validate statically and what requires integration testing.

---

## 6. Required technical reconciliations

The implementation plan must explicitly resolve the following issues.

### 6.1 DBOS transaction semantics

Determine how the selected DBOS Go version provides transaction integration.

Distinguish:

- DBOS system database.
- Application datasource.
- The exact underlying database pool handle.
- `RunAsTransaction` or the version-equivalent API.
- The transaction-completion mechanism when application and system databases use different handles.

Do not assert atomicity based merely on both databases being Neon or sharing a connection URL.

Specify and test the crash windows between business writes and DBOS checkpoints.

### 6.2 Neon connection mode

DBOS uses PostgreSQL functionality that is not compatible with Neon's standard transaction-mode pooler, including `LISTEN/NOTIFY`.

Specify the direct-connection strategy for DBOS initially.

Evaluate suitable application and River connection modes separately.

Define connection budgets across Fly Machines and the consequences of horizontal scaling.

### 6.3 DBOS Sleep and Fly Machine lifecycle

DBOS durable sleep persists wake-up time. Suspending sleeping workflows from memory may require an Enterprise feature.

Fly auto-start is request-driven and must not be assumed to wake a stopped worker because a database timer became due.

Compare:

- Always-available DBOS executor.
- Supported DBOS Enterprise suspension and recovery.
- External scheduler/wake-up service.

Propose a practical baseline and an explicit acceptance test:

1. Start a multi-day workflow.
2. Stop the worker.
3. Allow the due time to pass.
4. Ensure the execution environment can be reactivated.
5. Verify that the correct workflow resumes.
6. Verify that no completed effect repeats.

### 6.4 External-effect correctness

Never describe external effects as unconditionally exactly once.

Define the actual guarantees, especially for providers with no idempotency support or no reliable operation-status lookup.

The acceptance target must distinguish confirmed duplicates, lost operations and unresolved outcomes.

### 6.5 Entity concurrency

Do not use a DBOS workflow ID as the sole implementation of a Virtual Object or single-writer entity abstraction.

Specify PostgreSQL uniqueness, transaction locking, optimistic concurrency or another explicit concurrency-control mechanism.

### 6.6 Authorization

Resolve agent principal versus delegated user versus tenant service authority.

Do not blindly mandate RFC 8693 token exchange.

Do not assume `auth.user_id()` is supplied by standard Neon PostgreSQL.

Specify RLS context injection, transaction boundaries, database roles and `FORCE ROW LEVEL SECURITY`, while keeping application authorization as a distinct mandatory control.

### 6.7 Deployment model

Resolve shared runtime versus one Fly application per miniapp.

Propose the simplest baseline that preserves miniapp isolation and independent scaling.

Plan the required process groups, likely including:

- API/MCP.
- Webhook receiver.
- River workers.
- DBOS executor.
- Any independent always-on wake/scheduler component justified by the DBOS design.

The API/MCP and webhook surfaces must not depend on a worker being available.

Do not assume Fly scale-to-zero is compatible with DBOS durable waits without proving the wake-up topology.

### 6.8 Workflow and miniapp version compatibility

Define how running workflows survive software upgrades.

Specify:

- Immutable published miniapp versions.
- DBOS workflow identity.
- Application version compatibility.
- Dependency pinning.
- Migration or drain strategy.
- Rollback behavior.
- Deployment guard.

A breaking workflow change must not silently corrupt or strand existing executions.

---

## 7. Deliverable Two: reference miniapps

The implementation plan must cover the reference catalog in `plan3_z.md`.

Each reference implementation must be a real consumer of the public framework library, not an example that bypasses its APIs.

Each should include:

- Miniapp manifest.
- Go implementation.
- Input/output contract.
- Capability dependencies.
- Permissions and trust-mode configuration.
- Execution substrate declaration.
- Fixtures and independent expected results.
- Unit and integration tests.
- Failure/recovery tests relevant to its class.
- Local run instructions.
- Fly deployment configuration.
- Example MCP, REST and/or CLI invocation.
- Operational documentation.

Retain the following catalog.

| Reference miniapp        | Class                                      |
| ------------------------ | ------------------------------------------ |
| `crm-read`               | A                                          |
| `crm-write`              | A                                          |
| `notify-send`            | B                                          |
| `report-generate`        | B                                          |
| `contact-import`         | C                                          |
| `data-sync`              | C                                          |
| `lead-followup`          | D                                          |
| `appointment-journey`    | D                                          |
| `approval-flow`          | D                                          |
| `agent-toolchain`        | D, subject to separate scope justification |
| `campaign-orchestration` | D, gated                                   |

The plan must recommend an implementation order.

Start with a small vertical slice that exercises the framework through at least one Class A miniapp and one Class D miniapp.

`lead-followup` is the primary durable workflow reference.

Build Class B/C examples after the framework's invocation and recovery contracts have stabilized.

Keep `campaign-orchestration` blocked until the external-effect protocol has demonstrated its correctness against a real provider sandbox.

Do not silently drop `agent-toolchain`. Assess whether it belongs in the initial product, and document its dependency and scope implications.

Every reference implementation must have a clear purpose in proving one or more framework capabilities.

---

## 8. Integration and harness validation

The plan must include at least:

- One real SaaS API with a sandbox or test tenant.
- One custom REST API.
- One cross-system business-process example.
- MCP invocation from at least one coding-agent harness.
- MCP invocation from at least one general-purpose agent runtime.
- HTTP invocation independent of MCP.
- CLI invocation independent of MCP.

At minimum, evaluate Codex or Claude Code and Hermes or OpenClaw.

Do not assume they all implement identical MCP transports, elicitation, tasks or approval behavior.

The same miniapp contract and execution service must work across the adapters.

Human approval must be enforceable at the server level even when a harness does not support native elicitation.

---

## 9. Reliability and security acceptance standards

Define objective test cases and acceptance criteria.

At minimum, include:

### Invocation and identity

- Duplicate agent submissions.
- Missing or invalid agent identity.
- Invalid tenant association.
- Unauthorized miniapp.
- Unauthorized miniapp version.
- Missing delegated authority.
- Invalid tenant service grant.
- Revoked connection.
- Insufficient provider scope.

### Multi-tenant security

- Cross-tenant read attempts.
- Cross-tenant write attempts.
- Manipulated resource IDs.
- RLS policy bypass attempts.
- Database roles that bypass RLS.
- Concurrent tenant-scoped operations.

### External integrations

- OAuth token expiry.
- Refresh failure.
- Revocation.
- Rate limiting.
- Provider outage.
- HTTP timeout.
- Duplicate webhook.
- Webhook replay.
- Invalid signature.
- Unknown provider outcome.
- Provider idempotency behavior.

### Execution reliability

- Double-submit.
- Worker termination during a DBOS step.
- Worker termination between external effect and checkpoint.
- Worker termination during a River job.
- Database connection interruption.
- DBOS recovery.
- River retry and terminal failure.
- Cancellation.
- Approval while worker is stopped.
- Deployment with pending workflows.
- Concurrent actions on the same entity.

### Acceptance language

Use measurable, technically honest outcomes.

Examples:

- No unauthorized capability execution.
- No cross-tenant data access in required probes.
- No lost committed River enqueue.
- No duplicate provider-confirmed business effects under the tested idempotency contract.
- Unknown outcomes are quarantined rather than blindly repeated.
- Completed DBOS steps are not re-executed.
- Interrupted workflows resume under the documented recovery model.
- No unresolved active invocation is silently discarded.

Do not make an unconditional exactly-once guarantee for external APIs.

Run the critical persistence and workflow tests against real PostgreSQL. Use a disposable Neon branch where appropriate, and prove deployment behavior against actual Fly Machines rather than relying only on mocks.

---

## 10. Architecture and dependency rules

The plan must include an explicit dependency-direction diagram.

At a minimum, distinguish:

- Framework public API.
- Internal runtime.
- Execution adapters.
- Integration adapters.
- Security and policy.
- Persistence.
- Agent transports.
- Reference miniapps.

Reference miniapps may depend on the framework's public APIs.

The framework must not import reference miniapps.

Transport adapters must not own business-process logic.

MCP and CLI adapters must not implement their own authorization rules separate from the common invocation service.

Provider-specific clients must not leak credentials or transport-specific behavior into business workflows.

DBOS and River must be isolated behind stable framework contracts to the extent practical, without creating abstractions so generic that their real execution semantics become invisible.

Recommend repository and Go module boundaries from the actual codebase. Justify whether the library and reference implementations should be in one repository, multiple modules, or separate repositories.

---

## 11. Deployment and operations plan

Specify the initial Fly.io deployment topology.

Include:

- Process groups.
- Machine sizing assumptions and how to validate them.
- Region selection relative to Neon.
- Health checks.
- Graceful shutdown.
- Database migration process.
- Worker scaling.
- Autoscaling and auto-stop limitations.
- Secrets.
- Tenant connection credentials.
- Encryption-key management.
- Backup and restore considerations.
- Audit retention.
- DBOS and River failure inspection.
- Alerts for stuck, quarantined or stranded work.
- Release and rollback procedures.

Do not repeat the anecdotal 150 ms cross-region latency claim as a universal fact. Define a benchmark and placement policy based on measured latency.

Separate platform secrets from per-tenant OAuth and API credentials.

Do not assume Fly.io secrets alone constitute a full tenant credential vault or KMS solution. Compare feasible options and recommend one with a clear operational and rotation model.

---

## 12. Required contents of the implementation plan

Write a single comprehensive Markdown plan, recommended path:

`docs/plans/agent-invokable-miniapp-framework-implementation-plan.md`

Use the existing repository's document conventions if there is a more appropriate location.

The plan must contain:

1. Executive summary and final architecture recommendation.
2. Product boundaries and explicit non-goals.
3. Source-document reconciliation table.
4. Repository inventory and reuse/adapt/replace/retire assessment.
5. Key architecture decisions and ADR references.
6. Target architecture diagrams.
7. Framework library responsibilities and package/module structure.
8. Public Go API proposals, with representative signatures.
9. Manifest, capability, identity, policy, run and external-effect contract proposals.
10. Execution-class decision matrix and DBOS adoption policy.
11. River and DBOS integration design.
12. Neon connection and transaction design.
13. MCP, HTTP/OpenAPI and CLI architecture.
14. OAuth, tenant security and authorization architecture.
15. External-effect, idempotency, reconciliation and Saga design.
16. Reference miniapp catalog and implementation sequence.
17. Fly deployment and operations model.
18. Test architecture and acceptance criteria.
19. Phased implementation roadmap.
20. Dependency/version/license register.
21. Risk register and unresolved technical questions.
22. Explicit out-of-scope items.
23. Final readiness assessment and implementation gates.

For each phase, specify:

- Objective.
- Exact packages/components affected.
- Dependencies.
- Engineering tasks.
- Expected artifacts.
- Unit and integration tests.
- Failure tests.
- Security tests.
- Acceptance criteria.
- Exit gate.
- Known risks.

Provide enough detail to turn each phase into an independent coding-agent implementation task.

---

## 13. Required implementation sequencing

Do not blindly preserve either document's phase order.

Recommend a sequence that proves the framework as early as possible.

At minimum, assess this structure:

**Phase 0 — Repository investigation and architecture contracts**

Resolve module boundaries, core contracts, DBOS/River integration feasibility, threat model, identity model and transaction semantics.

**Phase 1 — Minimal framework vertical slice**

Implement the minimum framework needed for one Class A miniapp and one Class D miniapp using test connectors. Prove the shared invocation API, MCP/HTTP interface, durable recovery, audit and authorization.

**Phase 2 — Security and real SaaS integration**

Implement connector registry, token broker, credential storage, authorization and one real SaaS sandbox integration.

**Phase 3 — River Class B/C support**

Implement the job adapter and representative Class B and C reference miniapps.

**Phase 4 — Approval and external-effect hardening**

Complete approval flows, reconciliation, compensation and unknown-outcome handling.

**Phase 5 — Reference catalog expansion**

Build the remaining reference miniapps, preserving the class distinction and the admission rubric.

**Phase 6 — Deployment, harness compatibility and falsification**

Exercise actual Fly deployment, Neon connection limits, crash recovery and at least two distinct agent harness families.

You may modify this order, but every change must be justified by dependency analysis and an early end-to-end proof.

---

## 14. Engineering discipline

Apply the following rule throughout the plan:

**Solve the underlying system problem, not merely the visible symptom.**

For every important behavior:

- Trace the entire invocation and persistence path.
- Identify authoritative state and invariants.
- Identify every transition where a crash or retry can occur.
- Identify which component owns recovery.
- Identify the security principal and authority for each operation.
- Identify the database transaction boundary.
- Define independent regression tests that prove the invariant.
- Avoid superficial workarounds that leave the underlying issue unresolved.

When evaluating a proposed abstraction, show what complexity it removes and what complexity it introduces.

Avoid designing a framework for hypothetical requirements. Keep the first public API small, stable and sufficient for the reference miniapps.

Do not sacrifice correctness for an attractive architecture diagram.

Do not claim that DBOS, River, PostgreSQL, MCP, Fly or OAuth provides a guarantee that has not been verified for the selected version and configuration.

Clearly label:

- Verified facts.
- Repository findings.
- Architectural recommendations.
- Assumptions.
- Open questions.
- Decisions that require a proof of concept.

---

## 15. Scope restrictions for this task

This task is to produce the implementation plan.

You may create or update the implementation-plan Markdown document and narrowly necessary planning diagrams or ADR documents.

Do not:

- Implement the framework.
- Create or modify Go source packages.
- Change `go.mod`.
- Add dependencies.
- Modify migrations.
- Deploy anything.
- Change Fly or Neon configuration.
- Create real SaaS connections.
- Modify existing business data.
- Perform broad documentation cleanup unrelated to the plan.

Do not overwrite existing documents without first inspecting them and preserving relevant decisions.

At completion, report:

1. The implementation plan's exact repository path.
2. The principal architecture decisions.
3. The major contradictions resolved.
4. The unresolved blockers.
5. The phases that are ready for implementation.
6. The specific evidence or proof of concept still required before coding.

The plan must be detailed enough to act as the architecture baseline for both deliverables: **the Go framework library and its reference miniapp implementations**.

## **6. My final recommendation**

I would make one additional governance decision before asking the coding agent to start implementing: **the first public API of the framework should be designed around the reference miniapps, but the framework must not be shaped around just&#x20;**`lead-followup`**.**

Use `crm-read` to prove Class A, `lead-followup` to prove Class D, and then `contact-import` to prove River-backed Class C. These three examples force the framework to establish its core contracts without prematurely implementing the entire catalog.

The rest of the reference catalog then becomes evidence that the library is actually reusable.

The current upstream documentation also supports the central technical stack: DBOS Go has reached the v1 release series, River provides transactional PostgreSQL jobs, and the official MCP Go SDK is actively maintained. The important caveats are the DBOS connection mode, Enterprise-only sleep suspension, and real external-effect semantics—not a lack of Go ecosystem support.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+4



**I would send the prompt above to the coding agent now and require it to return a reviewed implementation plan before any Go implementation begins.**