# **The new direction: agent-invokable business automation miniapps**

I think this is a stronger and more concrete product direction. It also gives the miniapp concept a much clearer purpose.

**The product is a platform for building, governing and executing reusable business-process automations that AI agents can invoke as tools.**

Think of each miniapp as a small, specialized automation worker: it knows how to perform a particular business task, which SaaS systems it can access, what operations it is allowed to perform, how to handle failures, and what result it must return.

An agent should not need to understand every underlying API or reconstruct a business process from scratch. It should be able to discover an appropriate miniapp, provide validated inputs, invoke it, and receive a result or execution identifier.

I would describe the product as:

> An API-first, agent-invokable automation platform that turns business processes into reusable, governed miniapps, allowing AI agents to execute real business work across SaaS and custom applications through secure, auditable integrations.

There is an important distinction from conventional RPA: **make API automation the default, not screen automation.** Calling a supported SaaS API is generally a more stable integration boundary than automating clicks through a changing interface. Browser or desktop automation can remain a later, separately governed fallback when an application has no usable API.

## **1. What I take from the attached&#x20;**`rea.md`

The useful contribution of `rea.md` is its execution-pattern taxonomy. It does not suggest that every automation should be a durable workflow. It separates ordinary operations, one-shot background jobs, multi-step operations and long-running business workflows. It also introduces per-entity state, saga compensation, concurrency and policy guardrails.

I would preserve these patterns, but express them as **miniapp categories** rather than making Restate or Rea the foundation of the product.

There is good news on the language requirement: DBOS now has an official Go SDK. Its current documentation provides Go workflow, step, queue, concurrency and transaction support. The current quickstart specifies Go 1.26 or higher.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+3



For the pg-boss-like component, **River** is my leading Go/Postgres candidate. It is a transactional job queue with retries, scheduling, unique jobs, multiple queues, periodic jobs and transactional enqueue/completion. It is not a complete durable-workflow substitute, which is precisely why the two roles can be separated.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+2



## **2. The miniapp categories**

I would establish five primary categories. The remaining ideas in `rea.md`—sagas, fan-out/fan-in, approvals and policy guardrails—should be composable patterns rather than separate application categories.

| Miniapp category        | Purpose                                                                     | Execution mechanism                                                                          | Example                                                                  |
| ----------------------- | --------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| **A. Action miniapp**   | Perform one bounded business operation                                      | Go service + PostgreSQL transaction                                                          | Look up a customer, update a contact, create an invoice                  |
| **B. Job miniapp**      | Execute one independent background task                                     | River                                                                                        | Generate a report, synchronize a record, process a webhook               |
| **C. Pipeline miniapp** | Perform several dependent operations with a bounded lifecycle               | River by default; DBOS when durable checkpoints are justified                                | Import, validate, enrich and synchronize a customer batch                |
| **D. Workflow miniapp** | Run a durable business process with waits, external events or approvals     | DBOS                                                                                         | Customer onboarding, invoice collection, multi-stage employee onboarding |
| **E. Entity miniapp**   | Coordinate operations around a particular business entity and its lifecycle | PostgreSQL state + application concurrency controls; DBOS when durable execution is required | Manage an individual customer onboarding case, order or support ticket   |

These categories can be composed. For example, a customer onboarding miniapp might contain an action to validate the customer, a pipeline to synchronize data, a durable workflow to wait for approvals, and an entity state record to track the customer's onboarding status.

### **Additional patterns from Rea**

**Saga and compensation**

A process that changes multiple external systems needs a recovery strategy.

For example:

1. Create a customer in the CRM.
2. Create a billing account.
3. Provision a workspace.
4. Notify the account owner.

If provisioning fails, the miniapp must know what can be reversed, what cannot be reversed, and what requires manual reconciliation.

Compensating actions should be registered and designed before their corresponding external actions. A compensation is not guaranteed to undo the original operation; some external effects are irreversible.

**Fan-out/fan-in**

A miniapp can process many independent records concurrently, then collect and reconcile their results. Use explicit concurrency limits and provider-specific rate limits.

**Human-in-the-loop**

A workflow can pause for an operator or authorized business user to approve, reject or supply missing information. The approval becomes a durable workflow event, not an untracked chat response.

**Per-entity state**

Rea's Virtual Object pattern is particularly interesting. A keyed customer, account or order can have its own state and serialized operations.

However, do not assume DBOS supplies Restate's native Virtual Object abstraction. I would implement the entity state and concurrency invariant explicitly in PostgreSQL. Use transactions, row-level locking or optimistic version checks according to the operation. DBOS manages durable process execution, not the complete business entity model.

**Cross-cutting guardrails**

Adopt the useful ideas from Rea:

- Stable idempotency keys for consequential actions.
- Strict policy enforcement in production.
- Warning mode for development.
- Disabled enforcement only in controlled tests.
- Explicit retry, cancellation and recovery policies.
- Separation between the control plane and execution plane.

The important conceptual change is that these become properties of the miniapp runtime and its execution contracts, not requirements for every miniapp to implement independently.

## **3. The proposed architecture**

I would organize the platform into six parts.

**AI agent harnesses**

Codex · Claude Code · Hermes · OpenClaw · Other MCP clients

**Agent integration gateway**

MCP · REST API · OpenAPI · Authentication · Authorization

**Miniapp control plane**

Registry · Specifications · Validation · Versioning · Publishing · Policies

**Miniapp execution router**

**Action**

Go services

**Job**

River workers

**Pipeline**

River / DBOS

**Workflow**

DBOS

**Capability and connector runtime**

OAuth · Token refresh · API adapters · Permission checks · Rate limits · Idempotency · Audit

**External business systems**

SaaS APIs · Custom REST APIs · Webhooks · Internal applications

Shared infrastructure

Fly.io + Neon PostgreSQL

### **A. Agent integration gateway**

This is the interface through which external agents discover and invoke your miniapps.

I recommend **MCP as the primary agent interface**, with REST and OpenAPI available for conventional software integrations.

This is a practical choice because the current documentation establishes MCP support for Codex, Claude Code, Hermes and OpenClaw. The official Go MCP SDK also supports server and client implementations.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://developers.openai.com\&sz=32)

OpenAI Developers+4



The initial tool surface could be:

| MCP tool           | Purpose                                                             |
| ------------------ | ------------------------------------------------------------------- |
| `list_miniapps`    | Discover available automations                                      |
| `describe_miniapp` | Return purpose, inputs, outputs, permissions and execution behavior |
| `run_miniapp`      | Execute a published miniapp                                         |
| `get_run_status`   | Inspect progress, results and failures                              |
| `approve_run_step` | Submit an authorized human approval                                 |
| `cancel_run`       | Request cancellation                                                |

A synchronous action returns its result immediately. A background job or workflow returns a `run_id`, current status and information about how to retrieve its result.

Do not keep an MCP request open for a business process that may take days.

For the initial implementation, I would use a stable generic `run_miniapp` interface. Later, the server can expose individual published miniapps as separately named MCP tools with their own input schemas, provided the target clients handle dynamic tool discovery reliably.

### **B. Miniapp control plane**

This manages the miniapps themselves.

It needs to support:

- Creating and editing a miniapp.
- Defining its input and output schemas.
- Selecting a category and execution pattern.
- Connecting approved integrations.
- Defining permissions and approval requirements.
- Testing against representative fixtures.
- Publishing immutable versions.
- Disabling or rolling back a published version.

The LLM can help generate and modify specifications, but it must not approve its own generated capabilities or permissions.

### **C. Execution runtime**

The runtime interprets and executes the miniapp specification.

I would strongly favor **declarative miniapps**, rather than generating arbitrary Go, JavaScript or shell scripts for every automation.

For example, a miniapp could define these operations:

```
TRIGGER: agent invocation

1. READ customer from CRM
2. VALIDATE customer information
3. CHECK duplicate customer in billing system
4. IF duplicate:
     RETURN DUPLICATE_FOUND
5. CREATE billing account
6. REQUEST approval for service provisioning
7. WAIT for approval
8. PROVISION workspace
9. UPDATE CRM status
10. RETURN onboarding result
```

These are not instructions for the LLM to execute arbitrarily. They are a declarative specification interpreted by a trusted Go runtime, using registered and approved operations.

That is how you can build many different business automations without creating a bespoke software project for each one.

## **4. The Go technology allocation**

I would make the following choices.

| Responsibility             | Technology                            | Reason                                            |
| -------------------------- | ------------------------------------- | ------------------------------------------------- |
| Backend and runtime        | Go                                    | Primary implementation language                   |
| API and MCP server         | Go HTTP stack + official Go MCP SDK   | Common agent and application interface            |
| Ordinary data operations   | PostgreSQL transactions               | No durable workflow needed                        |
| Background jobs            | River                                 | Go/Postgres alternative to pg-boss                |
| Durable business workflows | DBOS Go                               | Checkpointing and recovery                        |
| Database                   | Neon PostgreSQL                       | Application state, queue and workflow persistence |
| Deployment                 | Fly.io                                | API and worker deployment                         |
| Integration layer          | Go HTTP clients and provider adapters | SaaS and custom REST APIs                         |

### **Why River and DBOS together?**

Your DBOS adoption policy from `DBOS(1).md` should remain in force.

The separation is:

- **Class A:** ordinary Go application operations.
- **Class B:** River.
- **Class C:** River by default; DBOS when the workflow genuinely needs durable progress or recovery between stages.
- **Class D:** DBOS.

DBOS itself has durable queues, scheduling and concurrency controls. River also has scheduling, retries, queues and uniqueness. That overlap is real. The reason to use both is not to have two competing execution engines for the same work; it is to give lightweight jobs a simpler queue and durable workflows a purpose-built execution model.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+2



Use exactly one authoritative execution owner for each miniapp invocation. Do not send a job through River and then separately submit the same work to DBOS.

### **A very important DBOS qualification**

DBOS Go is a real option today, not a hypothetical port. Its Go SDK supports durable workflows and transactions. Its datasource integration can atomically commit application writes and the corresponding DBOS durability record when configured appropriately.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+1



But there are two deployment caveats.

**First: Neon connection pooling.**

DBOS's production documentation says to use a direct connection or a session-mode connection pooler, not transaction-mode pooling. DBOS uses PostgreSQL features such as `LISTEN/NOTIFY` that are incompatible with transaction pooling. Neon's standard pooled connection endpoint uses transaction mode and does not support `LISTEN/NOTIFY`.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+1



Therefore, I would plan to use:

- Neon direct connection for DBOS.
- A bounded `pgxpool` connection pool.
- Explicit limits on total connections across Fly Machines.
- A separate connection configuration for workloads that can safely use Neon's transaction pooler.

**Do not simply copy the ordinary Neon&#x20;**`-pooler`**&#x20;connection string into DBOS.**

**Second: DBOS recovery when scaling workers.**

DBOS can recover workflows when an executor restarts. But its documentation says Conductor is required for correct recovery across multiple application processes. Conductor also provides distributed recovery and operational workflow management. Production self-hosting has licensing requirements that should be budgeted and verified.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+2



My recommendation:

- Start with one dedicated DBOS executor process.
- Scale the HTTP/MCP service independently.
- Scale River workers independently.
- Add DBOS Conductor when the durable executor needs multiple processes or machines with coordinated recovery.

This makes DBOS feasible on Fly.io, but the multi-executor deployment decision must be explicit.

## **5. Integrations and security: the most important part of the product**

This should not be treated as a secondary feature. The ability to execute real business operations safely is the product's central technical challenge.

There are two separate authentication relationships:

**1. Agent → your platform**

Who is invoking the miniapp? Which organization and permissions does this identity have?

**Platform authorization**

Validate agent identity, user context, tenant membership, miniapp access and execution policy.

**2. Your platform → external SaaS**

Which SaaS account is connected? Which API scopes and resource permissions were granted?

**Capability authorization**

Validate approved operation, connection, API scope, record access, risk level and idempotency.

**External API operation**

Perform the authorized action and record the outcome.

Do not conflate these two identities. The agent's permission to invoke a miniapp does not automatically grant permission to execute every operation through a connected SaaS account.

### **Required integration contract**

Every connector operation should declare:

| Property            | Example                                           |
| ------------------- | ------------------------------------------------- |
| Provider            | CRM provider                                      |
| Operation           | `create_customer`                                 |
| API version         | Pinned provider API version                       |
| Input/output schema | JSON Schema                                       |
| Authentication      | OAuth connection or scoped API credential         |
| Required scopes     | Minimum provider scopes needed                    |
| Resource scope      | Which accounts, records or objects are accessible |
| Risk class          | Read, reversible write, consequential write       |
| Idempotency         | Required key and behavior                         |
| Retry policy        | Safe retry rules                                  |
| Rate limits         | Provider-specific constraints                     |
| Timeout             | Operation deadline                                |
| Audit               | Required invocation and outcome records           |

Integration discovery should produce candidate capabilities, not automatically approved ones.

For SaaS OAuth, use authorization-code flows with PKCE, exact redirect URI matching, least-privilege scopes and appropriate refresh-token protection. These are part of the IETF's OAuth 2.0 Security Best Current Practice.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.rfc-editor.org\&sz=32)

rfc-editor.org+1



For custom applications, support REST API and OpenAPI contract import. Treat imported API definitions as untrusted until their endpoints, operations and authorization requirements have been reviewed.

### **Security rules I would make non-negotiable**

1. The agent never receives raw SaaS access tokens, refresh tokens or API secrets.
2. Tokens are encrypted and stored server-side; workflow payloads contain credential references, not credentials.
3. The backend never trusts client-supplied tenant IDs as proof of identity.
4. Every operation receives server-side authorization, including record-level and tenant-level checks.
5. Read and write permissions are independently configurable.
6. High-risk operations have explicit approval policies, with the approval coming from an authorized principal—not the agent itself.
7. Webhooks are authenticated, replay-protected and deduplicated.
8. Every external mutation has an idempotency strategy, an audit record and an outcome-reconciliation strategy.

MCP authorization is useful for securing the agent-to-platform connection, but it does not replace authorization inside the miniapp execution service. The MCP authorization specification defines the HTTP authorization model, and current OpenAI guidance explicitly requires servers to authorize each request themselves.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1



## **6. What the miniapp specification should contain**

I would define one central contract: `MiniappSpec`.

A miniapp should be a versioned, declarative automation definition, not a loosely structured prompt.

A first version should include:

| Section             | Meaning                                                  |
| ------------------- | -------------------------------------------------------- |
| Identity            | ID, name, description, owner, version                    |
| Category            | Action, job, pipeline, workflow or entity                |
| Trigger             | Agent, API, schedule, webhook or event                   |
| Input schema        | Required arguments and validation                        |
| Output schema       | Result returned to the agent                             |
| Steps               | Ordered operations, branching and approved parallelism   |
| Capability bindings | References to approved API operations                    |
| Authorization       | Required scopes, roles and resource restrictions         |
| Execution policy    | Queue/workflow selection, retry, timeout and concurrency |
| Approval policy     | Human approval requirements                              |
| Compensation        | Reversal and reconciliation operations                   |
| Audit policy        | Events, retention and sensitive-data handling            |
| Dependencies        | Connector, API contract and miniapp versions             |

For example, the `customer_onboarding` miniapp must pin the relevant CRM and billing capabilities, their versions and input/output contracts. A provider API update must not silently change the behavior of an already published miniapp.

The generator's responsibility is to propose a specification. The compiler/runtime is responsible for validating it against the trusted component and capability libraries.

## **7. Deployment on Fly.io and Neon**

I would initially deploy three independently managed execution groups:

| Fly deployment     | Responsibility                                                                |
| ------------------ | ----------------------------------------------------------------------------- |
| `automation-api`   | REST API, remote MCP server, authentication, miniapp discovery and invocation |
| `automation-river` | River job workers for one-shot and bounded background jobs                    |
| `automation-dbos`  | DBOS executor for durable workflows                                           |

Neon PostgreSQL stores application state, execution records, River job state and DBOS workflow state. Use clear schema and database-role boundaries, with the DBOS connection configured for a compatible Postgres endpoint.

The API should not need to know how every workflow executes. It submits an invocation to the appropriate execution mechanism and exposes its status through the same public contract.

This also keeps the platform's external contract stable even if the internal execution implementation changes later.

## **8. What I would actually build first**

I would resist the temptation to start with a large library of miniapps or dozens of SaaS connectors.

Build one vertical slice that proves the entire proposition.

**First demonstration: customer onboarding automation**

A business user connects a CRM and a billing application. An agent invokes the onboarding miniapp.

1. **Receive the invocation**

   Validate the agent identity and required customer inputs.
2. **Look up the customer**

   Use an approved CRM read capability.
3. **Check for duplicates**

   Validate the customer against the billing API.
4. **Create the billing account**

   Execute an authorized, idempotent write.
5. **Pause for provisioning approval**

   Use a durable DBOS workflow wait.
6. **Provision and update the CRM**

   Perform approved operations, recording every outcome.
7. **Return the result**

   Provide the agent with the completed result or a recoverable execution status.

This single example exercises API integrations, authorization, a durable workflow, human approval, idempotency, audit and agent invocation.

I would implement it in this order:

**Phase 0 — Foundations**

Define `MiniappSpec`, connector and capability contracts, identity and authorization boundaries, execution status model, and the DBOS/River ownership policy. Prove DBOS Go against Neon using the correct connection mode.

**Phase 1 — Execution runtime**

Implement the Go runtime, MCP gateway, capability registry, River workers and one DBOS workflow. Use a mock SaaS API first.

**Phase 2 — First real integration**

Connect one real SaaS API and one custom REST API. Implement OAuth connection handling, scoped permissions, token refresh, auditing and rate limits.

**Phase 3 — Miniapp authoring**

Create a manual specification for the onboarding miniapp, validate it and publish it. Then introduce AI-assisted generation of `MiniappSpec`.

**Phase 4 — Pattern library**

Add reusable action, job, pipeline, workflow and entity patterns. Test each against failure scenarios, including duplicate invocation, API timeout, expired credentials and partial external mutations.

**Phase 5 — Agent harness validation**

Connect Codex, Claude Code, Hermes and OpenClaw through MCP. Test discovery, invocation, asynchronous status, approval and cancellation.

**Phase 6 — Product hardening**

Add additional connectors, tenant isolation, capability lifecycle management, operational recovery, metering and controlled publishing.

## **9. The architectural decisions I would lock in**

| Decision                | Recommendation                                                                     |
| ----------------------- | ---------------------------------------------------------------------------------- |
| Product                 | Agent-invokable business automation miniapps                                       |
| Automation philosophy   | API-first RPA-style process automation                                             |
| Implementation language | Go                                                                                 |
| Durable execution       | DBOS Go, selectively                                                               |
| Simple job queue        | River                                                                              |
| Database                | Neon PostgreSQL                                                                    |
| Deployment              | Fly.io                                                                             |
| Agent protocol          | MCP first, REST/OpenAPI alongside it                                               |
| Miniapp implementation  | Declarative specification interpreted by trusted Go runtime                        |
| Integration model       | Approved SaaS and custom REST API capabilities                                     |
| Security                | Delegated OAuth, scoped permissions, server-side authorization and audit           |
| Rulez                   | Optional deterministic decision component, not mandatory for every miniapp         |
| UI                      | Optional human control and approval interface, not the primary product requirement |

One strategic distinction is worth emphasizing: **the platform should own business-process execution, but it should not attempt to replace the AI harness.**

Codex, Claude Code, Hermes and OpenClaw already provide the agent loop, context management and reasoning. Your platform should supply the reliable business capabilities they lack: reusable processes, authenticated integrations, controlled operations, durable execution and traceable results.

The resulting product is not another general-purpose AI agent. It is a library and execution platform of reliable business capabilities that many different agents can use.

That is the direction I would commit to.