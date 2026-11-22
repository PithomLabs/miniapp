# **Consolidated architecture: Fly Machines and Fly Sprites**

Adopt Sprites as a first-class deployment target

`plan3_gemini.md` reinforces the architecture in `plan3_z.md`: the agent is the planner and caller; the miniapp is the governed executor. It adds a clearer separation of responsibilities between the runtime, database, agent interfaces and execution substrates. I would adopt its useful additions, but revise the deployment model substantially.&#x20;

plan3_gemini.md



There is one important distinction I would make before updating the coding-agent prompt:

**Do not classify deployment as simply “long-running = Machines” and “AI-agent miniapp = Sprites.” Classify it along two independent dimensions: business-execution requirements and compute trust profile.**

These are different questions.

- DBOS and River determine how a business process executes, persists progress and recovers.
- Fly Machines and Fly Sprites determine where the code runs, how it is isolated and how its compute lifecycle is managed.

### **1. What Fly Sprites actually add**

The official Fly documentation describes Sprites as persistent, hardware-isolated Linux VMs intended for AI agents, development environments and untrusted code. They have persistent filesystems, automatic idle behavior, checkpoints and restore, and individually addressable service URLs.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.fly.io\&sz=32)

Fly.io+2



The API is also more than a command-execution endpoint. It includes:

| Capability       | Available API operations                            |
| ---------------- | --------------------------------------------------- |
| Sprite lifecycle | Create, list, get, update, delete                   |
| Execution        | Execute command, list sessions, kill session        |
| Persistence      | Create, list, get and restore checkpoints           |
| Filesystem       | Read, write, copy, rename, delete and inspect files |
| Security         | Set network, privileges and resource policies       |
| Services         | Create, start, stop, restart and inspect services   |
| Connectivity     | Connectors for API keys, OAuth and custom APIs      |
| Operations       | Usage, logs and WebSocket sessions                  |

[image](https://www.google.com/s2/favicons?domain=https://docs.fly.io\&sz=32)

Fly.io+7

Fly also provides a Go example using `github.com/superfly/sprites-go`, which gives this direction a potentially useful native Go integration path. Its maturity and coverage still need to be checked before making it a mandatory framework dependency.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Linux computers for agents



### **2. The deployment distinction I recommend**

**Fly Machines**

Production services

[Do Frontend ao Airflow: Como escalei uma arquitetura complexa no Fly.io (e economizamos dinheiro) | by Anderson Souza | Feb, 2026 | Medium](https://images.openai.com/static-rsc-4/yVaurt5oaCjmF8r8iKGID3vyRzQjT7jKVVgCSr0KtLUocgZv_6s-S6LqSTudvqfcdgm4CseotYLMuuq9otvE9RfyYB5ol8NhyCCR3odAMSfrcJFwxmbhQNWMB68AyUtdg9bBI_zC0wUovmb2wrKAXEIqebicBdw4FMHnLxoNaQ4?purpose=inline)

[medium.com](https://medium.com/%40andersou/do-frontend-ao-airflow-como-escalei-uma-arquitetura-complexa-no-fly-io-e-economizamos-dinheiro-66a81934bb28?utm_source=chatgpt.com)

Use for trusted, published miniapp binaries and platform services that need stable service hosting, production network integration, durable workers and predictable operations.

Examples: API/MCP gateway, River workers, DBOS executors, webhook receivers and production miniapp services.

**Fly Sprites**

Agent workspaces

[GLM-5.1: 754B MoE model, AIME 95.3%, SWE-Bench Pro 58.4%, MIT licensed](https://images.openai.com/static-rsc-4/v8FWH_7_h_eJUJ3HtLMLK5aKDNDBPxAfcwOtJn_pg_HIOPPoGnVboWG-thOGLiCdLdraCk-jlWdCfDLTqxZu_oR5KhefPNUQvCaOqia4aLquBXC-QMTWZZE4LP5JHTXEitICgRuQLuHcqzprvOcG45SPwsrZQhMCaE3FUkP-U00?purpose=inline)

[reeboot.fr](https://reeboot.fr/en/blog/glm-51?utm_source=chatgpt.com)

Use for agent-authored, dynamically generated or untrusted code that needs an isolated Linux computer, persistent working filesystem, command execution, development tools, checkpoint/restore or per-task isolation.

Examples: agent-created miniapp prototypes, custom code adapters, integration experiments, browser/CLI automation and isolated agent execution.

Fly's own architecture describes the agent's trusted process as living on a Machine in production while the code it runs executes separately inside Sprites. It also describes promoting applications built in Sprites to production Machines.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

fly.io+1



**This leads to a correction to the product architecture:** an agent-invokable miniapp does not automatically belong in a Sprite. A trusted Go miniapp exposed as an MCP tool should ordinarily execute through the platform runtime on Machines. A miniapp that needs its own isolated Linux environment because it runs generated code, custom scripts or agent-directed processes is a strong Sprite candidate.

A single business miniapp may use both:

- Its DBOS workflow remains authoritative on a Machine.
- A bounded, isolated agent task runs in a Sprite.
- The workflow stores the task identity and result in Neon, not in the Sprite filesystem.
- The Sprite never receives unrestricted platform or SaaS credentials.

### **3. The deployment matrix**

| Miniapp / workload                                                 | Execution mechanism              | Preferred Fly target |
| ------------------------------------------------------------------ | -------------------------------- | -------------------- |
| Class A: trusted API action                                        | Go handler + PostgreSQL          | Machine              |
| Class B: simple job                                                | River                            | Machine              |
| Class C: bounded pipeline                                          | River, sometimes DBOS            | Machine              |
| Class D: durable workflow                                          | DBOS                             | Machine              |
| Agent-generated code or custom scripts                             | Isolated execution               | Sprite               |
| Agent development and prototyping workspace                        | Interactive Linux environment    | Sprite               |
| Untrusted integration test or code evaluation                      | Isolated execution               | Sprite               |
| Trusted production API created by an agent                         | Normal Go production runtime     | Machine              |
| Agent task that needs isolated execution within a durable workflow | DBOS orchestration + Sprite task | Both                 |

This distinction is important because a Sprite's checkpoint restores filesystem state; it is not a durable business-workflow checkpoint that automatically resumes a transaction or a workflow at the exact instruction. Fly documents checkpointing as a filesystem snapshot and restore.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Linux computers for agents+1



So **Sprites complement DBOS; they do not replace it**.

## **4. The missing deliverable: a first-class Fly deployment experience**

I would extend the Go framework beyond execution and integration abstractions. It should provide a consistent development and operations experience for both Fly compute products.

Fly Sprites have a particularly useful API surface for this purpose: the management API can create and delete Sprites, execute commands, transfer files, manage services, create and restore checkpoints, configure network and resource policies, and inspect usage.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.fly.io\&sz=32)

Fly.io+5



There is also an important security caveat: Fly documents that Sprite outbound networking starts open, and its network-policy API supports domain-based rules. The framework should configure the network policy before executing agent-supplied code.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

fly.io+1



### **The lifecycle I want the framework to make easy**

1. **Design**

   Scaffold a miniapp from a category template. Define its contract, execution class, capabilities, security policy and deployment profile.
2. **Develop**

   Work locally or inside an isolated Sprite. Get typed contracts, fixtures, integration mocks, logs and live feedback.
3. **Validate**

   Run unit, contract, security, concurrency, failure and crash-boundary tests. Provision disposable Neon branches and Sprite previews where needed.
4. **Publish**

   Produce an immutable, versioned artifact with its manifest, dependency lock, test results and provenance.
5. **Deploy**

   Deploy trusted services and durable execution workers to Machines; provision approved agent workspaces and isolated execution environments through Sprites.
6. **Operate**

   Unified status, logs, traces, costs, execution history, checkpoints, approvals, retries, recovery and rollback.
7. **Maintain**

   Upgrade versions safely, migrate workflows, rotate connections, inspect security posture, decommission unused environments and restore prior known-good states.

My preferred developer experience is that the developer does not have to understand the detailed Fly API merely to build and deploy a miniapp. The framework should hide repetitive infrastructure work, without hiding consequential operational decisions.

One possible CLI design is:

```
# Create a miniapp from a category and deployment profile
miniapp init lead-followup --class D

# Start local development
miniapp dev

# Validate contracts and policies
miniapp validate

# Run tests
miniapp test

# Create an isolated preview environment
miniapp preview

# Publish an immutable version
miniapp publish

# Deploy to the selected Fly target
miniapp deploy

# Inspect deployed environments and executions
miniapp status
miniapp runs
miniapp logs

# Promote or roll back a release
miniapp promote
miniapp rollback

# Diagnose infrastructure and dependency problems
miniapp doctor
```

The command names are proposals, not decisions. The implementation plan should assess the best interface and avoid unnecessary command proliferation.

For the Sprite management SDK, there is a published Go project, `github.com/superfly/sprites-go`. Its documented focus is idiomatic remote command execution; the repository also exposes filesystem, service and checkpoint functionality. Some management methods have historically been described as future functionality, so the plan must inspect actual versioned API coverage before deciding to use it for the full lifecycle. A typed REST client may be more appropriate for provisioning and policy management.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1



# **5. Updated coding-agent prompt**

This replaces the previous implementation-planning prompt. It combines the valid architecture from all three plans, preserves the two deliverables, and adds Fly Machines, Fly Sprites and a unified development-to-operations experience.

# Task: Design the Complete Go Miniapp Framework, Reference Implementations, and Fly.io Deployment Platform

## Role

Act as the principal Go systems architect, infrastructure architect and adversarial technical reviewer.

Your responsibility is to inspect the actual repository, reconcile the supplied architecture documents, research current upstream technologies, and produce an implementation plan for a reusable agent-invokable business-process automation framework.

The implementation has exactly two primary deliverables:

1. **A reusable Go miniapp framework library.**
2. **A catalog of reference miniapp implementations built with that framework, following `plan3_z.md`.**

The framework must also provide an excellent, unified developer and operator experience for designing, developing, testing, publishing, deploying, monitoring, updating and maintaining miniapps across the entire software development lifecycle.

Fly.io has two important deployment targets that must be treated as first-class capabilities:

- **Fly Machines:** trusted production services, APIs, workers and durable business-process execution.
- **Fly Sprites:** isolated, persistent Linux computers for agent-oriented development, untrusted or dynamically generated code, and controlled agent execution environments.

**This assignment is implementation planning only. Do not implement the framework or reference miniapps.**

Produce a detailed, evidence-based implementation plan that another coding agent can execute without rediscovering architectural decisions or resolving hidden contradictions.

---

# PART I — SOURCE DOCUMENTS AND ARCHITECTURAL PRECEDENCE

## 1. Read the sources completely

Read:

- `plan3_z.md` — primary miniapp product and implementation architecture.
- `plan3_deepseek.md` — supplementary agent-facing invocation, MCP and manifest design.
- `plan3_gemini.md` — supplementary technical architecture and Go/DBOS/River/Fly/Neon allocation.
- `rea.md` — execution-category patterns and the original Restate/Rea mapping.
- `DBOS(1).md` — DBOS Adoption Policy / ADR-001.
- ADR-001 amendments and the DBOS adoption rubric, if present.
- Review 12 reliability retrospective and remediation material, if present.
- Relevant existing OpenLevel architecture, engineering and operational documents.

Inspect the current repository before proposing code organization.

## 2. Precedence

Resolve contradictions using this precedence:

1. Explicit deliverables in this task.
2. ADR-001 and its amendments.
3. `plan3_z.md`.
4. `plan3_gemini.md`.
5. `plan3_deepseek.md`.
6. `rea.md` for its reusable patterns.
7. Existing repository implementation and verified upstream behavior.

Do not merge contradictory claims without documenting their resolution.

Create a source-reconciliation table that distinguishes:

- Accepted unchanged.
- Accepted with modification.
- Rejected.
- Deferred.
- Requires proof of concept.

---

# PART II — PRODUCT DEFINITION

## 3. Product objective

The product is an agent-invokable business-process automation platform.

AI agents running in harnesses such as Codex, Claude Code, Hermes, OpenClaw and other compatible environments discover, invoke, monitor and interact with miniapps.

A miniapp is a governed, versioned, deployable business-process automation unit.

It owns:

- Typed input and output contracts.
- Execution and lifecycle behavior.
- Authorized business capabilities.
- SaaS and custom REST integrations.
- Credentials through trusted connection infrastructure.
- Authentication and authorization enforcement.
- Idempotency and external-effect safety.
- Durable execution when justified.
- Approval and compensation behavior.
- Audit and recovery.
- Deployment and operational configuration.

The agent is the planner and caller.

The miniapp is the governed executor.

An agent may decide what business action to perform and coordinate separate tools. It must not hold provider credentials, own authoritative durable business state or bypass platform permission checks.

This is API-first business automation inspired by RPA, not brittle screen-scraping by default.

Browser and desktop automation, where necessary, must be an explicitly declared and separately secured capability. It must not become an undocumented fallback.

## 4. Explicit non-goals

Do not turn this project into:

- A general-purpose agent framework.
- A replacement for Codex, Claude Code, Hermes or OpenClaw.
- A universal BPMN engine.
- A general-purpose workflow engine replacing DBOS.
- A UI/dashboard generator.
- An unrestricted shell or browser automation service.
- An arbitrary LLM-generated code deployment platform.
- A rule engine that every miniapp must use.

Rulez can be integrated later as an optional deterministic business-decision component. It is not a mandatory dependency for the framework or every miniapp.

The framework must not execute unreviewed LLM-generated Go, shell scripts or JavaScript inside its trusted production runtime.

---

# PART III — REPOSITORY INVESTIGATION

## 5. Mandatory inventory

Inspect and document the real repository before designing the target implementation.

Investigate:

- Repository layout and Go module boundaries.
- `go.mod` and Go toolchain requirements.
- Existing application code and domain models.
- Current persistence and SQL migrations.
- Existing API and HTTP architecture.
- Authentication, authorization and tenant isolation.
- Existing queues, workflows, outbox and idempotency.
- Recovery, retries, compensation and external-effect protocols.
- Existing logs, metrics, traces and audit.
- Fly.io deployment and configuration.
- Neon/PostgreSQL integration and connection pools.
- CI, integration testing and release processes.
- Existing CLI, documentation, `AGENTS.md`, ADRs and runbooks.

Classify relevant existing components as:

- Reuse.
- Adapt.
- Replace.
- Retire.
- Investigate.

Do not assume existing reliability mechanisms are correct merely because they exist.

Trace actual state transitions, transaction boundaries, concurrent execution behavior and crash recovery.

Do not invent repository paths, current package APIs or module versions.

Do not mechanically transplant TypeScript/Hono infrastructure into Go without examining whether its invariants and design are appropriate.

---

# PART IV — TWO INDEPENDENT ARCHITECTURAL DIMENSIONS

## 6. Execution semantics and compute placement must not be conflated

This is a core architectural requirement.

There are two separate decisions for every miniapp:

**Dimension A: How does its business process execute?**

- Class A: ordinary application operation.
- Class B: simple background job.
- Class C: bounded multi-step operation.
- Class D: durable business workflow.

**Dimension B: Where and under what isolation profile does its code run?**

- Fly Machine.
- Fly Sprite.
- Hybrid Machine + Sprite.
- Local development or test environment.

Do not put DBOS, River, Fly Machines and Fly Sprites into one mutually exclusive technology selector.

DBOS/River determine execution and recovery semantics.

Machines/Sprites determine compute placement, lifecycle, trust boundary and isolation.

A long-running business workflow can use DBOS on a Fly Machine while invoking an isolated Sprite for a bounded agent-execution step.

An MCP-exposed agent tool implemented as a trusted Go handler does not automatically need a Sprite just because an AI agent invokes it.

A persistent Sprite is not a substitute for DBOS business-workflow persistence.

## 7. Required compute-placement policy

Evaluate and document the following baseline.

| Workload                                         | Execution substrate                                      | Preferred Fly target |
| ------------------------------------------------ | -------------------------------------------------------- | -------------------- |
| Trusted Class A API action                       | Go handler + PostgreSQL                                  | Machine              |
| Class B simple job                               | River                                                    | Machine              |
| Class C bounded batch                            | River, selectively DBOS                                  | Machine              |
| Class D durable business workflow                | DBOS                                                     | Machine              |
| Trusted production miniapp API                   | Trusted Go runtime                                       | Machine              |
| Agent-authored or generated code                 | Isolated execution                                       | Sprite               |
| Persistent agent development workspace           | Sprite filesystem/process environment                    | Sprite               |
| Per-invocation isolated code task                | Isolated execution                                       | Sprite               |
| Untrusted code evaluation and testing            | Isolated execution                                       | Sprite               |
| Agent task inside a durable business workflow    | DBOS orchestration + isolated execution                  | Hybrid               |
| Agent-generated prototype promoted to production | Sprite during development, immutable artifact on release | Sprite → Machine     |

This table is the baseline, not a substitute for investigation.

The implementation plan must explain exceptions.

### 7.1 Fly Machines

Prefer Machines for:

- Platform API and MCP gateway.
- Authentication and authorization services.
- Connector and capability proxy.
- Webhook ingress.
- River workers.
- DBOS workflow executors.
- Trusted production miniapp services.
- Durable workflow orchestration.
- Operational components requiring defined service availability, deployment and scaling