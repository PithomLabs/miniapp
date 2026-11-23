## prompt

just to clarify why did we test class A then class D and skipping class B and C for now, what is the rationale

## answer

The rationale is to validate the framework's **two ends of the execution spectrum first**, before introducing the two intermediate execution models.

This is deliberate architectural sequencing, not an assumption that Classes B and C are unimportant or that every workload needs DBOS.

## **1. Class A proves the ordinary execution path**

Phase 1 — Class A

Class A represents an ordinary operation completed within one request, using a Go handler and PostgreSQL transaction.

Our `crm-read` example proves that the framework can:

- Accept a canonical invocation.
- Authenticate and authorize before dispatch.
- Enforce tenant isolation.
- Execute synchronously.
- Return an inline result.
- Record audit and idempotency information.
- Do all this without introducing a workflow engine unnecessarily.

This is important because **Class A is the default execution model**. We don't want every miniapp to incur the complexity of durable orchestration.

## **2. Class D proves the durable execution path**

Phase 1 — Class D

Class D represents the opposite end: a durable, multi-stage business process using DBOS.

It exercises capabilities that ordinary request processing cannot adequately demonstrate:

- Durable workflow submission and stable run handles.
- Multiple checkpointed stages.
- Durable waits and signals.
- Restart and replay.
- Cancellation and lifecycle management.
- Approval-pending recovery.
- Version compatibility across deployments.
- Effect safety and complete audit attribution.

This is the most demanding execution model in the current framework architecture. It tests whether the framework can manage execution that outlives the original request, process, and potentially the deployed application version.

## **3. Why not Class B yet?**

Class B is a one-shot, independently retryable background job.

Its default substrate is **River**, not DBOS.

For example, suppose an application accepts an order and needs to send a confirmation email. That email job can be independently retried without requiring an orchestrated multi-stage workflow.

The important engineering requirement is transactional enqueue: the motivating database operation and job enqueue must be coordinated so that a committed business change does not silently lose its associated job.

But River's `InsertTx` behavior and the selected Neon connection mode require their own integration proof.

The plan therefore places Class B in Phase 3, alongside the River integration work.&#x20;

plan4.2.md



## **4. Why not Class C yet?**

Class C represents bounded batch or multi-step operations.

Consider importing 100,000 customer records. Each record might need validation, transformation, persistence, retry, and progress tracking.

There are two plausible approaches:

**Option A — River plus application-managed progress**

The application maintains batch state, record-level progress, and idempotency. River executes bounded jobs.

**Option B — DBOS queued workflows**

DBOS provides checkpoint recovery, queue partitioning, and potentially more sophisticated orchestration and concurrency controls.

Neither should win automatically.

The actual workload determines the choice. The plan explicitly requires comparing transaction behavior, throughput, concurrency, rate limiting, recovery, implementation complexity, and operational burden.

That is why Phase 3 includes a measured `contact-import`/`bulk` comparison and a workload-specific ADR-001 decision.&#x20;

plan4.2.md



## **5. The architectural logic**

## **Execution-model validation sequence**

**Class A — Synchronous**

# **Request → Result**

Prove the canonical invocation and ordinary transaction path.

**Class D — Durable orchestration**

# **Submit → Run → Recover**

Prove the strongest orchestration and recovery requirements.

**Class B**

## **River**

Independent jobs and transactional enqueue

Phase 3

**Class C**

## **River / DBOS**

Measured workload-specific comparison

Phase 3

Conceptual sequence from the `plan4.2` execution taxonomy and roadmap.

The reasoning is:

1. Prove the simplest execution path first.
2. Prove the most demanding durable execution path next.
3. Once those contracts work, introduce the independent job substrate.
4. Then make an evidence-based decision about bounded batch execution.

This prevents premature architectural decisions about queues and batching from complicating the canonical runtime before its fundamental synchronous and durable execution contracts are proven.

## **6. What this means for the product architecture**

There is an important distinction: **we are not building four independent execution frameworks.**

We are building one framework with different execution adapters and policies.

All four classes should share the appropriate common infrastructure:

- Canonical invocation and authorization.
- Tenant and authority context.
- Registry and manifest contracts.
- Idempotency.
- Audit.
- Effects and approvals where applicable.
- Operational visibility and policy enforcement.

The execution substrate changes according to the workload.

The roadmap makes this explicit: Phase 1 proves A and D; Phase 3 adds B and decides C.&#x20;

plan4.2.md



**My architectural assessment:** this is the right sequencing. It validates both the lightweight and durable paths without prematurely committing the platform to River or a universal batch abstraction.

One important qualification: passing A and D does not automatically prove B and C. Each has its own transactional, retry, concurrency, fairness, and recovery requirements. They deserve their own evidence and acceptance gates rather than being treated as incidental extensions of the existing executors.


## prompt

just to clarify, one of the goals of this miniapp framework is to create robust primitives so that any miniapp implementation do not need to reinvent the wheel in order to create robust apps across the 4 app categories especially class D


## answer

That is an important architectural clarification. In fact, I would make it an explicit, central objective of the Miniapp framework—not merely an implied benefit.

**The framework is not simply a runtime for executing miniapps. It is a platform that provides the reusable engineering primitives necessary to build robust business applications without requiring every miniapp developer to become an expert in distributed systems, transaction management, workflow orchestration, and failure recovery.**

That distinction changes how we should evaluate Prompt 15.2.

## **1. The framework owns reliability; miniapps own business logic**

The fundamental division should be:

| Responsibility                   | Framework                         | Miniapp                                                     |
| -------------------------------- | --------------------------------- | ----------------------------------------------------------- |
| Authentication and authorization | Provides and enforces             | Declares requirements                                       |
| Tenant isolation                 | Enforces                          | Uses trusted context                                        |
| Invocation and idempotency       | Manages                           | Defines business input                                      |
| Execution lifecycle              | Manages                           | Defines business operation                                  |
| Retry and recovery               | Provides safe mechanisms          | Defines business-specific retry policy where necessary      |
| Audit                            | Provides durable attribution      | Supplies business context                                   |
| Effects and approvals            | Provides secure primitives        | Declares consequential operations and approval requirements |
| Durable waits and signals        | Provides orchestration primitives | Defines when and why to wait                                |
| Cancellation                     | Enforces lifecycle guarantees     | Defines business cancellation behavior                      |
| Version compatibility            | Provides recovery mechanisms      | Declares compatibility requirements                         |
| Business rules                   | Does not invent them              | Owns them                                                   |

The miniapp developer should be able to concentrate on the actual problem being solved rather than repeatedly implementing infrastructure that every other miniapp also needs.

## **2. What this means across the four execution classes**

**Class A — Synchronous operation**

Framework-managed reliability

The developer implements the business operation. The framework handles invocation, authentication, tenant context, transaction boundaries, serialization where required, and audit.

**Class B — One-shot independently retryable job**

Framework-managed reliability

The developer defines the job. The framework provides durable enqueueing, identity, retries, idempotency, and observable outcomes.

**Class C — Bounded batch or multistep execution**

Framework-managed reliability

The developer defines the batch or bounded sequence. The framework provides execution controls, progress, concurrency limits, recovery, and the appropriate execution substrate.

**Class D — Durable business workflow**

Highest primitive-reuse priority

The developer defines the business process. The framework provides durable orchestration, checkpointing, waits, signals, approval integration, effect safety, cancellation, version compatibility, recovery, and lifecycle observability.

Class D is particularly important because its failure modes are substantially more complicated. Without framework-provided primitives, every miniapp author would have to rediscover the same distributed-systems problems.

## **3. The real meaning of Prompt 15 remediation**

This is where your clarification materially changes the acceptance criteria.

Take the five HIGH findings:

- X-01 and X-02 should not merely be fixed in the current approval implementation. The framework should expose an approval primitive that makes bypass and stranded approval workflows structurally difficult or impossible.
- X-03 should result in reusable invocation and workflow identity management, not a special-case locking fix for one miniapp.
- X-04 should result in a reusable, concurrency-safe approval decision mechanism.
- X-05 should result in an effect-reservation primitive that automatically coordinates with framework cancellation.

The same principle applies to the MEDIUM findings.

For example, a future Class D miniapp should not have to implement its own signal recovery loop, settlement audit, effect refusal audit, or workflow attachment mechanism.

**Fixing the framework means making the safe implementation the easiest and preferably the only supported implementation.**

## **4. An additional architectural acceptance criterion**

I recommend adding this objective explicitly to Prompt 15.2 and the P1 design document:

> **Reusable Framework Primitives and Miniapp Authoring Contract**
>
> The Miniapp framework must provide reusable, composable, and safe primitives across execution Classes A, B, C, and D. Miniapp implementations must not independently reinvent framework-level concerns such as invocation identity, idempotency, authorization, tenant isolation, lifecycle transitions, retry and recovery, audit, cancellation, or durable orchestration.
>
> Each primitive must have a documented contract, clearly defined responsibilities, safe default behavior, explicit failure semantics, and conformance tests.
>
> For Class D in particular, a miniapp author must be able to express its business workflow using framework-supported capabilities without independently implementing DBOS lifecycle management, approval recovery, signal delivery recovery, effect reservation, cancellation coordination, or settlement auditing.
>
> Framework internals may use DBOS, River, PostgreSQL, or other appropriate mechanisms, but miniapp authors should interact with the framework's stable abstractions rather than being required to reproduce or directly manage those underlying mechanisms.
>
> The framework must prove primitive reuse through multiple distinct test miniapps or conformance fixtures using the same framework capabilities, without duplicating lifecycle machinery.
>
> The framework remains responsible for technical guarantees; miniapps remain responsible for business rules and business-specific decisions.

## **5. One important distinction**

We should not attempt to turn every business behavior into a generic framework primitive.

For instance:

- The framework provides approval mechanics; the miniapp defines *what* needs approval.
- The framework provides durable waits; the miniapp defines *when* business conditions warrant waiting.
- The framework provides effect safety; the miniapp defines *what* consequential business operation to perform.
- The framework provides cancellation guarantees; the miniapp defines whether the underlying business process permits cancellation at a particular stage.

That separation prevents the framework from becoming an enormous, inflexible business-process engine.

## **My architectural conclusion**

The ultimate measure of Miniapp's success should be:

**How little infrastructure code must an application developer write to produce a reliable application?**

For Class D, a developer should be able to implement a sophisticated, multistage business workflow without personally solving workflow durability, approval races, effect idempotency, cancellation races, or recovery after process failure.

That is a much stronger objective than merely passing the current Class D conformance tests.

It also means Prompt 15 should leave behind two things:

1. A corrected framework with reusable reliability primitives.
2. A clear, stable miniapp authoring contract that makes those primitives accessible and difficult to misuse.

The neutral probe proves that the primitives work. Multiple distinct conformance fixtures should prove that they are genuinely reusable.

**We are building the foundation that makes robust miniapps ordinary to write, not building another framework that requires every developer to become its reliability engineer.**