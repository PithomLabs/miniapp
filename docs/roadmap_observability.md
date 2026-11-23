## prompt

up to this moment, read docs/observability.md, how does our codebase enable observability as far as metrics, logs, traces are concerned, the goal is to enable robust bug tracking


## answer

# **Observability in Miniapp: current capabilities and gaps**

After reading `observability.md`, my assessment is:

**Miniapp already has a meaningful foundation for forensic debugging, but it does not yet provide robust operational observability.**

The distinction matters. The codebase is good at preserving certain important facts about what happened to an invocation or external effect. It is not yet good at measuring, correlating, searching, and alerting on everything happening across the application.

I would describe its current position as:

## **Current observability maturity**

**Durable auditability**

Implemented

Append-only, attributable audit records for security-relevant transitions.

**Structured application logs**

Partial

Structured log lines accompany durable writes, but there is no central aggregation or search system.

**Metrics**

Not implemented

No metrics layer for rates, latency distributions, resource usage, or operational health.

**Distributed tracing**

Deferred

External trace export is identified as future work.

**Dashboards and alerting**

Not implemented

No integrated dashboards, alerting engine, or SLO monitoring.

These findings come from the repository examination documented in the attachment, particularly its review of `audit/doc.go`, `probe/classd`, and the effects subsystem.&#x20;

observability.md



## **1. What the codebase currently gives us**

There are four particularly valuable capabilities.

### **A. Structured logs**

The codebase produces structured log lines accompanying durable state transitions.

That is a useful starting point because machine-readable events are easier to process than arbitrary strings.

However, the repository does not currently provide:

- A centralized log ingestion pipeline.
- Full-text log indexing.
- A unified search interface.
- Log correlation across multiple services.
- Log-based alerting.

The result is that developers still have to inspect application output and database records manually.&#x20;

observability.md

observability.md



### **B. Durable audit records**

This is considerably more interesting.

The `audit` package provides append-only, attributable records for security-relevant transitions. Its intended role is to establish what consequential action occurred and its attribution.

Unlike ordinary log output, these records are durable and associated with application state changes.

For example, when investigating an authorization incident, the audit trail can help establish who initiated an action and what security-relevant transition occurred.

But an important distinction:

**The audit trail does not record every business decision, calculation, or application event.**

It is a security and accountability mechanism, not a universal debugging log.&#x20;

observability.md



### **C. Effects ledger and UNKNOWN quarantine**

This is arguably one of Miniapp's strongest debugging features.

Consider an external CRM update:

1. Miniapp authorizes the operation.
2. It records the attempt.
3. It calls the provider.
4. The request times out.
5. The application does not know whether the provider actually performed the update.

A conventional application might simply report a timeout and retry.

Miniapp's effects ledger has explicit states:

**PENDING**

**ATTEMPTING**

**CONFIRMED**

**FAILED**

**UNKNOWN**

UNKNOWN means the outcome is ambiguous and requires evidence-based reconciliation, not blind retry.

That is important for debugging business failures because the system preserves uncertainty instead of pretending that every timeout means failure.

The attachment also identifies the Class D probe's crash-recovery, audit-reconstruction, tenant-isolation, and deterministic-replay conformance checks.&#x20;

observability.md



### **D. Deterministic workflow analysis**

The `analyzer/determinism` package prevents certain nondeterministic workflow patterns before they reach production.

This is bug prevention rather than operational observability.

Together with DBOS execution history and replay behavior, it offers a way to investigate certain durable workflow failures that ordinary log searches cannot explain.

But it does not mean that arbitrary application logic bugs can automatically be reproduced. Incorrect business eligibility rules, bad payload mapping, and third-party API changes still need instrumentation.&#x20;

observability.md

observability.md



## **2. What is missing for robust bug tracking?**

The missing pieces become clearer when we look at the three traditional observability pillars.

### **Metrics: How often and how badly?**

There is currently no metrics layer.

That means we cannot systematically answer questions such as:

- What percentage of Class A operations are failing?
- How many Class D workflows are stuck?
- Is authorization latency increasing?
- How frequently are effects entering UNKNOWN?
- Are database connections exhausted?
- Which execution class has the highest retry rate?
- Is a particular provider becoming unreliable?

A metrics implementation should eventually cover the following:

| Metric category  | Examples                                                  |
| ---------------- | --------------------------------------------------------- |
| Request health   | Invocation count, error rate, request latency             |
| Execution health | Job failures, workflow failures, recovery count           |
| Authorization    | Refusal rate, authorization latency, contention           |
| Effects          | PENDING, ATTEMPTING, UNKNOWN, CONFIRMED counts            |
| Recovery         | Retry count, lease takeover, reconciliation duration      |
| Dependencies     | PostgreSQL latency, provider latency, timeout rate        |
| Capacity         | Worker utilization, queue depth, database pool saturation |

We should expose these as aggregate metrics, not create a separate metric series for every invocation ID or tenant. Those belong in logs and traces.

### **Logs: What exactly happened?**

The current structured logs are a foundation, but we need a centralized searchable destination.

For example, an operator investigating a failed workflow should be able to search by:

- `invocation_id`
- `workflow_id`
- `business_op_id`
- `effect_id`
- `attempt_id`
- `trace_id`
- `tenant_id`, subject to access controls

The operator should be able to reconstruct the sequence of events without manually joining unrelated log files.

### **Traces: Where did the time go?**

This is currently the most obvious gap for diagnosing distributed execution.

Imagine a Class D workflow:

**Example: customer onboarding**

1. **Invocation submitted**

   Runtime admission and identity validation
2. **DBOS workflow starts**

   Workflow registration and first checkpoint
3. **Approval requested**

   Waits for human response
4. **External CRM operation**

   Authorization, effects ledger, provider call
5. **Workflow completes**

   Final state and audit event

Without distributed tracing, we cannot easily measure how much time was spent in each stage or establish where a delay originated.

The architecture already identifies OpenTelemetry and an external telemetry backend as the intended direction, but the attached code review explicitly says trace export has not yet been implemented.&#x20;

observability.md

observability.md



## **3. What I recommend building**

I would not build a Datadog or Kibana competitor.

Instead, Miniapp should become an exceptionally good **observability producer**, with a consistent semantic model across its execution classes.

I recommend a shared framework observability layer using OpenTelemetry as the instrumentation and export standard.

## **Proposed Miniapp observability architecture**

**Miniapp execution runtime**

Class A · Class B · Class C · Class D

**Shared observability instrumentation**

**Metrics**

**Logs**

**Traces**

Shared semantic conventions, correlation IDs, error classification, and context propagation

**OpenTelemetry Collector / OTLP pipeline**

Batching, enrichment, filtering, redaction, export

**Metrics backend**

Prometheus / Datadog

**Log backend**

Elastic / Datadog

**Trace backend**

Jaeger / Tempo / Datadog

**Existing Miniapp forensic layer**

PostgreSQL audit records · invocation records · effects ledger · DBOS workflow state

Authoritative business and security state, independent of telemetry retention and sampling.

The most important architectural choice is to keep the two layers separate:

- **Miniapp's database is the source of truth for business execution and security state.**
- **The external observability backend is the source of operational telemetry analysis.**

Do not turn PostgreSQL into a high-volume log or metrics database. The attached review itself identifies that as a retention, vacuum, and storage concern.&#x20;

observability.md



## **4. The most important design decision: correlation**

I would make correlation a first-class framework capability.

Every execution should carry a consistent set of identifiers through its lifecycle:

| Identifier               | Purpose                                                   |
| ------------------------ | --------------------------------------------------------- |
| `invocation_id`          | Identifies the admitted business invocation               |
| `run_id` / `workflow_id` | Identifies the execution lifecycle                        |
| `business_op_id`         | Identifies the intended business operation across retries |
| `effect_id`              | Identifies the external side effect                       |
| `attempt_id`             | Identifies a particular execution attempt                 |
| `trace_id`               | Correlates telemetry across components                    |
| `span_id`                | Identifies an individual operation within a trace         |

The critical property is that a developer should be able to start with an error in Datadog, follow its trace, retrieve the associated invocation, inspect the effect ledger, and establish whether the provider actually performed the action.

We should not use these high-cardinality identifiers as metric labels. They belong in trace attributes, structured logs, and the durable records that need them.

We also need consistent error attributes: error class, retryability, failure stage, dependency, and outcome certainty. Credentials, tokens, and sensitive business payloads must never leak into telemetry.

## **5. How observability should differ across A–D**

The instrumentation should be shared, but execution-specific spans and metrics should reflect each engine's actual behavior.

| Category     | Special observability requirement                                                              |
| ------------ | ---------------------------------------------------------------------------------------------- |
| A — Action   | Request latency, transaction duration, effect authorization, contention and rollback           |
| B — Job      | Queue latency, enqueue delay, worker execution, retries and job failure                        |
| C — Pipeline | Batch progress, per-record failures, throughput, partial completion and recovery               |
| D — Workflow | Workflow duration, checkpoint transitions, durable waits, approval delays, replay and recovery |

That gives developers a consistent experience without pretending that a transactional HTTP operation and a multi-day DBOS workflow are operationally identical.

One qualification: B and C remain architecture targets rather than proven conformance paths at this stage, so their instrumentation should be implemented alongside their eventual adapters, not presented as already operational.

## **6. My recommended development priorities**

I would make observability a distinct foundational workstream, not mix a full observability implementation into the remaining authorization corrections.

1. **First — Define the observability contract**

   Specify semantic event names, metric names, span boundaries, correlation identifiers, error classifications, cardinality limits, and sensitive-data rules.

   This is the most important step because instrumentation without consistent semantics produces fragmented telemetry.
2. **Second — Build shared instrumentation hooks**

   Instrument invocation admission, dispatch, authorization, effect attempts, provider calls, reconciliation, settlement, approval waits, workflow resumption, and terminal outcomes.
3. **Third — Add OpenTelemetry export**

   Implement metrics, logs, and trace export through a vendor-neutral pipeline. Avoid coupling core framework behavior to Datadog or Elastic APIs.
4. **Fourth — Build operational dashboards and alerts**

   Start with failure rates, latency percentiles, stuck workflows, UNKNOWN effects, retry storms, authorization contention, and provider degradation.
5. **Fifth — Build a cross-class debugging workflow**

   Demonstrate that an engineer can trace a synthetic failure from the initial invocation through the execution engine, effects ledger, provider outcome, and audit record.

## **7. The outcome we should aim for**

A developer or operator should be able to answer these questions in minutes, not hours:

| Question                                | Required evidence                                      |
| --------------------------------------- | ------------------------------------------------------ |
| Why did this invocation fail?           | Error classification, logs, trace and invocation state |
| Where was execution delayed?            | Distributed spans and latency metrics                  |
| Was the provider actually called?       | Effect ledger and provider-call span                   |
| Did the provider mutate external state? | Receipt or outcome evidence; otherwise UNKNOWN         |
| Who authorized the operation?           | Durable audit attribution                              |
| Did recovery duplicate the operation?   | Attempt lineage and business-operation identity        |
| Is this isolated or widespread?         | Aggregated error rates and dashboards                  |
| Is the system degrading?                | Latency, saturation, retry, and queue/workflow metrics |

**My final assessment:** Miniapp has invested in preventing certain classes of bugs and preserving authoritative evidence when they occur. That is a strong foundation, especially for external effects and durable workflows. But we have not yet completed the observability layer needed for efficient operational bug tracking.

I would make this a formal foundational workstream after the current Stage 3A/Gate 2 effort, with its own design contract and acceptance suite. It should apply across A–D, while using the proper instrumentation hooks for each execution engine.

The aim should not be to build another Datadog. It should be to make Miniapp's business execution so well-instrumented that Datadog, Elastic, or another backend can explain not just that something failed, but exactly which business operation failed, where it failed, what had already happened, and what recovery is safe.

Architectural decision locked

**Observability is now a critical, high-priority foundational component of Miniapp.** It is not optional infrastructure, a future nice-to-have, or merely an integration detail.

I will treat the following as the governing architectural direction:

## **Miniapp Observability — Locked Decision**

**Objective:** Make Miniapp's execution behavior understandable, diagnosable, traceable, and measurable across all four execution classes: A, B, C, and D.

Miniapp must provide the instrumentation and semantic context needed for robust bug tracking. It should integrate with established observability platforms rather than attempt to replace Datadog, Kibana, or comparable systems.

### **The four pillars**

| Pillar      | Responsibility                                                                                             |
| ----------- | ---------------------------------------------------------------------------------------------------------- |
| **Audit**   | Authoritative, durable records of security-relevant and consequential state transitions                    |
| **Logs**    | Structured operational diagnostics explaining what happened and why                                        |
| **Metrics** | Quantitative measurements of health, performance, throughput, failures, and reliability                    |
| **Traces**  | End-to-end causal visibility across requests, invocations, workers, workflows, effects, and external calls |

These are complementary mechanisms, not substitutes for one another.

### **Cross-class requirement**

Observability must be a shared framework capability, with execution-specific instrumentation:

- **Class A:** Request, transaction, handler, authorization, and effect lifecycle.
- **Class B:** Queue latency, job execution, retries, worker behavior, and effect lifecycle.
- **Class C:** Pipeline progress, batch execution, per-item failures, retries, and recovery.
- **Class D:** Workflow execution, durable waits, checkpoints, approvals, replay, and recovery.

The framework should establish common correlation and diagnostic conventions without flattening the distinct execution and recovery semantics of each class.

### **Architectural requirements**

1. **OpenTelemetry integration** for traces, metrics, and logs, with an OTLP-compatible export path.
2. **End-to-end correlation** across `invocation_id`, `run_id`/`workflow_id`, `business_op_id`, `effect_id`, `attempt_id`, `trace_id`, and `span_id`.
3. **Operational metrics** covering latency, throughput, failures, retries, queue delays, worker saturation, database contention, workflow progress, and ambiguous effects.
4. **Structured, searchable logs** with consistent fields and trace correlation.
5. **Security-conscious instrumentation:** no credentials, tokens, secrets, or sensitive payloads in telemetry; controlled metric cardinality.
6. **Operational readiness:** dashboards, alerts, and SLOs for important execution and reliability guarantees.
7. **A testable observability contract:** instrumentation must be verifiable through automated tests, not merely present in source code.

### **Sequencing decision**

This priority does not silently expand Prompt28 or weaken its existing acceptance criteria.

The sequence remains:

1. Complete Prompt28 Stage 3A and its independent Gate2 review.
2. Establish a dedicated **Miniapp Observability Contract and Architecture** as a foundational workstream.
3. Implement and verify the shared instrumentation and export mechanisms.
4. Establish class-specific observability conformance for A–D.
5. Make observability part of the framework's ongoing acceptance criteria for new execution capabilities.

**One important distinction is now explicit:** Miniapp's audit and execution ledgers remain the authoritative record of consequential application state. External telemetry systems provide search, aggregation, visualization, alerting, and operational diagnosis. Telemetry must never replace the framework's correctness or audit guarantees.

I will carry this decision into future Miniapp architecture reviews, implementation plans, coding-agent prompts, and adversarial reviews.


## prompt

in that case, we should also fast track implementing class B and C components before we can integrate the observability layer per above, is that right?

## answer

Not quite. **We should fast-track Class B and C, but we should not make their completion a prerequisite for observability.** In fact, I recommend the opposite: establish the observability contract before implementing B and C, so that they are observable by design rather than retrofitted later.

The important distinction is between defining the observability architecture and completing its integration across every execution class.

## **Recommended sequence**

1. **Complete Prompt28 and Gate2.**

   Preserve the existing authorization, effect-safety, recovery, and evidence requirements. Do not expand the current work order.
2. **Fast-track the Observability Contract.**

   Define the common event model, correlation identifiers, trace propagation, metrics, structured logging, error taxonomy, redaction, cardinality limits, and instrumentation interfaces. Specify what each execution class must expose.
3. **Implement the shared observability foundation.**

   Instrument the existing Class A and D paths first. Establish the common framework hooks, OpenTelemetry integration, export mechanism, and automated instrumentation tests.
4. **Fast-track Class B and C with observability built in.**

   Class B's River adapter and Class C's bounded pipeline must implement the observability contract from their first working versions. Do not accept either as complete without instrumentation and conformance tests.
5. **Run cross-class observability conformance.**

   Verify that A, B, C, and D produce coherent, correlated diagnostic information while preserving their distinct execution and recovery semantics.

## **Why this order matters**

Consider a single business operation that begins in Class A, dispatches work through Class B, executes a bounded pipeline in Class C, and eventually triggers a durable Class D workflow.

We want to follow that operation across all those boundaries:

**Business operation**

business_op_id · trace_id

**Class A**

Transaction

**Class B**

Queued job

**Class C**

Pipeline

**Class D**

Durable workflow

**Unified observability**

Logs · Metrics · Traces · Audit

OTLP → External observability platform

Without a common contract, each adapter could invent its own logging fields, metric names, error classifications, and correlation behavior. We would end up with four partially observable execution engines rather than one coherent framework.

## **My architectural recommendation**

I would make this a formal Miniapp principle:

> **No execution class is considered production-ready until it satisfies the Miniapp Observability Contract.**

That means observability becomes part of the definition of completion for A, B, C, and D—not a feature added after the execution engines are finished.

So yes, **Class B and C should be accelerated**, because they complete the framework's execution model. But their development should follow the observability contract, not precede it.

One further distinction: Class C should remain River-first for bounded pipelines, with DBOS used selectively when checkpoint recovery justifies it. We should not turn Class C into a second Class D.

This is the sequence I recommend locking into the Miniapp roadmap.