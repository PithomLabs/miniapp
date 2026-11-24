# Miniapp Observability Contract

Status: PROPOSED architecture (Prompt32 workstream O). Nothing in
this document claims existing implementation unless tagged
[Existing]. Tags: [Existing] / [Partial] / [Documented] /
[Proposed] (legend in `foundational-architecture.md`).

Provenance: `docs/plan4.2.md` §§7 (`audit/`), 17–18 (operate
view, test pyramid); `docs/roadmap_observability.md` (locked
four-pillar decision + sequencing); `docs/observability.md`
(retrospective: producer-not-platform, forensic strength,
operational gaps); `audit/` package sources.

Companion documents: `security-contract.md` (workstream S),
`developer-experience-enforcement.md` (workstream DX),
`foundational-architecture.md` (unified runtime, glossary,
inventory), `foundational-conformance-matrix.md`,
`foundational-roadmap.md`.

## 1. Rationale

Miniapp must answer "which business operation failed, where
it failed, what had already happened, and what recovery is
safe" in minutes, not hours. Today it preserves
authoritative forensic facts (audit rows, effect ledger,
checkpoints) but cannot measure, correlate, search, or
alert across the application: no metrics, no traces, no
aggregation.

The contract therefore makes Miniapp an exceptionally
good observability *producer* with one semantic model
across all four execution classes — and explicitly not a
telemetry backend. Datadog, Elastic, Jaeger/Tempo, or an
OTLP-compatible equivalent owns aggregation, search,
dashboards, and alerting. Two layers, two jobs:

- Miniapp's database is the source of truth for business
  execution and security state (audit, ledger,
  checkpoints). Independent of telemetry availability,
  retention, and sampling.
- The external backend is the source of operational
  analysis (search, aggregation, visualization,
  alerting). It never replaces correctness or audit
  guarantees.

PostgreSQL must never become a high-volume log or metrics
store (retention, vacuum, and cost anti-pattern — already
identified in `observability.md`).

## 2. Current state (observability inventory summary)

[Existing] `audit.Service`: append-only attributed events
(tenant/agent/authority/miniapp/tool/invocation/run/
action/outcome/details) with `Record`/`InsertTx`/`Query`;
refusal/withdrawal/settlement/approval pairing; audit
reconstruction proven by `probe/classd`.

[Existing] Effects ledger as a debugging surface:
explicit PENDING/ATTEMPTING/CONFIRMED/FAILED/UNKNOWN
(+ NOT_APPLIED/CANCELLED/RECONCILING) with attempt
lineage, provider request/reply evidence, and UNKNOWN
quarantine instead of blind retry.

[Existing] `analyzer/determinism`: CI-time rejection of
nondeterministic workflow patterns (bug prevention,
not observability).

[Partial] Structured logs: `slog` lines accompany durable
writes (`audit.Service` logger, grant-hash startup
line). No central ingestion, indexing, search, or
correlation.

[Absent] Metrics layer; distributed tracing; OpenTelemetry
dependency, instrumentation, or export; dashboards,
alerting, SLOs; redaction framework for telemetry;
sampling/retention policy; error-attribute conventions
beyond `contract.ErrorKind`
(terminal/retryable/quarantined).

## 3. The four pillars

Complementary mechanisms, never substitutes:

1. Audit [Existing, extended]: authoritative durable
   records of consequential and security-relevant
   transitions. Stays in PostgreSQL, committed with
   the state change it describes. Extended only with
   new event types (delegation, worker checkpoints,
   telemetry-pipeline health) — never moved to the
   telemetry backend.
2. Logs [Partial → Proposed]: structured operational
   diagnostics (what happened and why). Framework
   emits; OTel log pipeline ships to the backend.
   Every line carries the correlation set (§5) it
   can resolve — never a bare message.
3. Metrics [Proposed]: aggregate quantitative health:
   rates, latency distributions, saturation, queue
   depth, workflow counts. Low-cardinality labels
   only (§6); high-cardinality IDs are forbidden as
   label values (conformance assertion).
4. Traces [Proposed]: end-to-end causal spans across
   admission → dispatch → execution → effect →
   provider → reconciliation → settle, propagated
   through Class B/C/D worker boundaries.

## 4. OpenTelemetry architecture

### 4.1 Instrumentation [Proposed]

One framework-owned instrumentation package (new
`telemetry/` internal surface, behind interfaces so
adapters never import OTel directly):

- Tracers: framework tracer per execution boundary
  (admission, authorization, transaction, dispatch,
  worker pickup, handler/step, effect attempt,
  provider call, reconciliation, settlement,
  approval wait, workflow lifecycle).
- Meters: the §7 metric set with fixed names, units,
  and label keys — a closed registry; miniapps
  cannot mint ad-hoc framework metrics.
- Loggers: OTel log bridge fed by the same call
  sites that write structured logs today, so local
  `slog` output and exported logs never diverge.

Miniapps instrument nothing for framework events
(DX contract §5). Business-specific diagnostic
events use a separate, explicitly domain-owned API.

### 4.2 Trace propagation [Proposed]

- Sync (A): W3C `traceparent` in-process via
  `context.Context`; inbound REST/MCP headers
  honored when present, minted when absent.
- Queued (B/C): trace context serialized into the
  job/pipeline record at enqueue; the worker
  resumes the trace as a child span (never a new
  root). Loss of the carrier degrades to a linked
  span with the linkage recorded — never silent.
- Durable (D): trace context persisted in the
  workflow's durable state at submission; replay
  re-attaches without re-emitting duplicate spans
  (span emission is not a replayed side effect:
  emit on first execution, link on replay).
- Provider calls: `traceparent` injected into
  outbound capability calls where the provider
  supports context propagation; otherwise the
  provider span records request/response evidence
  locally.

### 4.3 Export: OTLP, Collector, backends [Proposed]

- Vendor-neutral OTLP export (gRPC preferred,
  HTTP fallback) to an operator-run OpenTelemetry
  Collector. Miniapp never speaks Datadog/Elastic/
  vendor APIs directly.
- Exporter abstraction: a `telemetry.Exporter`
  interface with OTLP, in-memory (tests), and
  discarding (explicit dev profile) implementations.
  Configuration is startup-validated; an
  unconfigured production exporter fails startup
  (DX contract, no silent downgrade).
- Collector owns batching, enrichment, filtering,
  tail sampling, and fan-out to the metrics / logs /
  trace backends chosen by the operator.

### 4.4 Failure behavior [Proposed, normative]

Telemetry must never break the business path:

1. Exporter outage: framework behavior is unchanged;
   spans/metrics buffer boundedly, then drop with a
   counted `telemetry.dropped` metric and a local
   log line. Audit and ledger writes are unaffected
   (separate path, no shared fate).
2. Slow exporter: async with bounded queues; no
   synchronous network I/O on any execution hot
   path (admission, authorization tx, effect
   attempt).
3. Misconfiguration: fail startup (loud), never run
   half-instrumented in production profiles.
4. Backend loss: historical telemetry may gap; audit
   reconstruction from PostgreSQL remains the
   ground truth for "what happened".

## 5. Correlation

### 5.1 Identifier set [Proposed conventions on existing IDs]

| Identifier | Source | Meaning |
|---|---|---|
| `principal_id` | Security resolution | On whose behalf |
| `actor_id` | Security resolution | Actual caller (agent/user) |
| `tenant_id` | [Existing] binding | Tenant boundary |
| `invocation_id` | [Existing] M1 | Admitted business invocation |
| `run_id` / `workflow_id` | [Existing] M4 | Execution lifecycle |
| `business_op_id` | [Existing] M3 | Intended operation across retries |
| `effect_id` | [Existing] ledger row | External side effect |
| `attempt_id` | [Existing] minted epoch | One execution attempt |
| `trace_id` / `span_id` | [Proposed] OTel | Telemetry correlation |
| `delegation_id` | [Proposed] §S4.4 | Agent authority scope |

Propagation rule: every log line, span, metric
exemplar, and audit `details` carries the full subset
resolvable at its emission point. A span without its
`invocation_id` where one exists is a conformance
failure.

### 5.2 Cardinality rules [Proposed, normative]

- High-cardinality identifiers (all of §5.1) are
  forbidden as metric label values. Metrics aggregate
  by class, surface, outcome, error class, provider,
  and workflow family only.
- Logs and traces carry the full IDs; audit rows
  carry the durable subset (already the shape of
  `audit.Event`).
- A conformance test enumerates the metric registry
  and fails on undeclared labels or ID-valued labels.

## 6. Operational guarantees

### 6.1 Redaction and sensitive data [Proposed, normative]

- Never in telemetry: presentation credentials,
  provider credentials, tokens, delegation secrets,
  PII beyond the tenant-scoped minimum needed for
  attribution. Error strings are constructed from
  typed fields, never by interpolating payloads.
- Redaction is enforced at emission (typed
  credential-free views from the token broker) and
  re-checked by the Collector pipeline config
  (defense in depth, not the primary control).
- Audit `details` follows the same rules; the
  existing attribution-only shape is the template.

### 6.2 Sampling, retention, storage [Proposed]

- Head sampling for routine spans; tail sampling at
  the Collector keeps all error/recovery/UNKNOWN
  paths. Audit and ledger rows are never sampled.
- Retention boundaries: telemetry retention is the
  operator's backend policy (days–weeks hot);
  audit/ledger retention follows the record-class
  policy (plan4.2 A10; erasure rules apply —
  telemetry copies of erased PII must age out with
  backend retention, never persist in Miniapp).
- Storage rule: high-volume diagnostics live in the
  backend, never in PostgreSQL (§1).

### 6.3 Diagnostic events and error taxonomy [Proposed]

Error attributes extend `contract.ErrorKind`
[Existing: terminal/retryable/quarantined] with:

- `error.class`: authn/authz/validation/contention/
  dependency/timeout/cancelled/internal.
- `retryable`: derived from kind, explicit on every
  emission.
- `failure.stage`: admission/dispatch/execution/
  effect/provider/recovery/settle.
- `dependency`: postgres/river/dbos/provider:<name>.
- `outcome_certainty`: confirmed/failed/unknown
  (unknown maps to quarantine, never to silent
  retry).

Framework diagnostic events are a closed,
enumerated set (see §7 coverage); business
diagnostics are domain-owned and optional.

### 6.4 Dashboards, alerts, SLOs [Proposed requirements]

The framework ships dashboard/alert *definitions*
(portable, e.g. Collector + backend-agnostic alert
rules), not a dashboard product:

- Required alerts: authorization refusal spikes,
  UNKNOWN-effect growth, stuck workflows/jobs,
  reconciliation lag, worker saturation, provider
  degradation, telemetry-pipeline drops,
  approval-queue aging, kill-switch engagement.
- SLOs (initial): admission latency, Class A
  p99 latency, effect-attempt success rate,
  reconciliation time-to-evidence, approval
  decision latency. Targets are owner decisions
  (O-D5).

### 6.5 Instrumentation testability [Proposed]

Every instrumentation claim is testable: the
in-memory exporter lets conformance tests assert
span trees, metric deltas, and log fields for
scripted scenarios (see
`foundational-conformance-matrix.md`). "Present in
source" without an assertion is not conformance.

## 7. Automatic instrumentation coverage

The framework instruments these lifecycle points
without developer action (DX §§5–6 own the
"invisible" mechanism; this section owns the list):

| Lifecycle point | Audit | Log | Metric | Span |
|---|---|---|---|---|
| Invocation admission (allow/refuse) | yes | yes | count + latency | admission span |
| Authorization decision (incl. refusal reason) | yes | yes | refusal rate | child of admission |
| Class A transaction outcome (commit/rollback/contention) | on terminal | yes | tx duration | tx span |
| Dispatch + worker pickup (B/C/D) | pickup | yes | queue latency | cross-boundary link |
| Retry, lease takeover, recovery | yes | yes | retry count | recovery span |
| Approval request/decision/expiry | yes | yes | queue age | wait span |
| Effect authorization + attempt | yes | yes | attempt outcomes | effect span |
| Provider call (request/response evidence) | ref only | yes | provider latency | provider span |
| Reconciliation + UNKNOWN handling | yes | yes | unknown count/lag | reconcile span |
| Settle (operator decision) | yes | yes | settle count | settle span |
| Workflow checkpoint/wait/replay (D) | checkpoints | yes | workflow counts | workflow spans |

Audit records the security-relevant subset; logs and
spans carry the diagnostic superset; metrics carry
aggregates only. Provider request/response payloads
in spans are evidence-capped and redacted (§6.1).

Per-class additions: A adds contention/rollback
detail; B adds job-attempt lineage; C adds
batch/item progress counters (no per-item spans by
default — cardinality); D adds durable-wait,
signal, replay-link, and version spans.

## 8. Event and metric taxonomy (initial)

Audit action vocabulary extends the existing
observed set (`effect-refused`, `effect-withdrawn`,
`effect-settled`, `approval-requested/denied/expired/
cancelled`, `run-*`, `invocation-ineligible`, …)
with: `delegation-created/revoked/expired`,
`worker-pickup-refused`, `telemetry-degraded`,
`exporter-down/recovered`, `reauth-required`.
Action strings stay lowercase-hyphenated; new
actions require a conformance test asserting their
emission shape.

Metric names (closed registry, `miniapp_` prefix):
`invocations_total`, `invocation_latency_seconds`,
`authorization_refusals_total`, `effect_attempts_total`
(by outcome), `unknown_effects`, `reconciliation_lag_
seconds`, `workflow_active`, `workflow_wait_seconds`,
`job_queue_latency_seconds`, `job_attempts_total`,
`pipeline_items_total` (by outcome), `provider_latency_
seconds` (by provider), `approval_queue_age_seconds`,
`telemetry_dropped_total`, `pool_saturation_ratio`.
Units and label keys are frozen at implementation;
additions need owner review (O-D6).

## 9. Owner decisions required

| ID | Decision | Needed before |
|---|---|---|
| O-D1 | OTel SDK + Collector versions and upgrade owner | Instrumentation implementation |
| O-D2 | Default exporter config per profile (dev discard? staging collector? prod collector URL management) | Export implementation |
| O-D3 | Backend choice(s) for reference dashboards (Prometheus+Grafana? Datadog?) — reference only, never coupled | Dashboard definitions |
| O-D4 | Retention numbers per record/telemetry class (with A10) | Retention implementation |
| O-D5 | Initial SLO targets per §6.4 | SLO wiring |
| O-D6 | Metric-registry change process (who approves new metrics) | First metric addition past §8 |

## 10. Acceptance criteria (contract implementation)

1. A scripted cross-class scenario produces one
   connected trace (in-memory exporter) from
   admission through effect and settle, with every
   span carrying its resolvable §5.1 IDs.
2. Metric registry contains exactly the §8 names;
   no ID-valued labels (static + runtime
   conformance).
3. Killing the exporter mid-run changes no business
   outcome; drops are counted and logged; audit is
   complete (failure-behavior proof).
4. Adversarial fixtures (credential-shaped strings
   in inputs/payloads) appear in no emitted
   telemetry or audit details.
5. All §7 rows emit under test; a coverage test
   fails if a lifecycle point loses instrumentation.
6. Dashboards/alerts of §6.4 exist as portable
   definitions and fire against synthetic load.
7. Replay of a Class D workflow emits no duplicate
   spans (link-on-replay proof).
