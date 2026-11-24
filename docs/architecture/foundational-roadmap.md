# Miniapp Foundational Roadmap

Status: PROPOSED architecture (Prompt32 §§9, 11). Nothing in
this document claims existing implementation unless tagged
[Existing]. Tags: [Existing] / [Partial] / [Documented] /
[Proposed] (legend in `foundational-architecture.md`).

Provenance: `docs/prompt32.md` §§9, 11–12;
`docs/plan4.2.md` §§19, 23 (phases, gates — extended, not
replaced); Prompt21–31 findings and dispositions;
Prompt30 residual closeout.

Companion documents: `security-contract.md` (S),
`observability-contract.md` (O),
`developer-experience-enforcement.md` (DX),
`foundational-architecture.md` (unified runtime, glossary,
inventory), `foundational-conformance-matrix.md`.

## 1. Sequencing (per §11, with gates)

Relation to plan4.2: this program inserts foundational
milestones *ahead of* Class B/C implementation (plan4.2
Phase 3) and the first real write (Phase 4). Plan4.2
phases and gates keep their meaning; G2–G4 gain
conformance criteria from this package. No plan4.2 gate
is weakened or skipped.

| Stage | Work | Entry | Exit proof |
|---|---|---|---|
| F-0 | Approve this package (owner review) | Package delivered | Written approval + resolved S/O/DX-D pre-questions |
| F-1 | Shared security foundation on A/D paths: versioned credentials, capability permissions, delegation records, worker checkpoints, redaction | F-0; S-D1/S-D2/S-D5 resolved | S acceptance 1–5 on A/D; existing suites green |
| F-2 | Shared observability foundation on A/D: telemetry pkg, OTel export, correlation, closed metric registry, failure behavior | F-0; O-D1/O-D2 resolved | O acceptance 1–5, 7 on A/D |
| F-3 | DX façade + startup validation + escape-hatch mechanism + demo app + adversarial matrix | F-1, F-2 (demo needs both) | DX acceptance 1–4, 6–7 |
| F-4 | Conformance suites (shared + A + D) in CI | F-1–F-3 | Matrix acceptance 1–5; A/D declared conformant |
| F-5 | Class B adapter with contracts from first version | F-4; River G3 proofs (plan4.2) | B suite green; B declared conformant |
| F-6 | Class C pipeline with contracts from first version | F-4; substrate decision (plan4.2) | C suite green; C declared conformant |
| F-7 | Cross-class A–D conformance + Workflow Kit + reuse proof | F-4–F-6 | Unified trace/assertions; DX-5 three-fixture proof |
| F-8 | Dashboards/alerts/SLOs + drill + production hardening | F-7; O-D3–O-D5 | O-6; drill executed; SLOs wired |

F-1 and F-2 are parallelizable (separate surfaces,
shared kernel points coordinated). F-5/F-6 follow
F-4 strictly: no adapter lands before its suite
exists. Each stage is an independently reviewable,
independently revertible work order.

## 2. Independently testable milestones

- M1: Versioned credentials + CLI token file land;
  demo keys retired; redaction suite green.
- M2: Delegation records + admission/ checkpoint
  enforcement; CS-02/CS-04 green on A/D.
- M3: OTel export with in-memory + OTLP
  exporters; CO-01/CO-03 green.
- M4: Closed metric registry + cardinality
  assertions; CO-02 green.
- M5: Capability façade lowering to the static
  registry; crm-read unchanged; DX-6 green.
- M6: Startup validation refusing bad configs;
  CDX-02 green.
- M7: Demo app + full adversarial matrix;
  CDX-05 green.
- M8: Shared + A + D suites in CI; matrix
  acceptance green.
- M9/M10: B adapter / C pipeline conformant.
- M11: Cross-class conformance + Kit reuse proof.
- M12: Dashboards/alerts/SLOs + executed drill.

Each milestone ships its own evidence (tests,
commands, environments) in the evidence register
(F19 discipline) — never a reread claim.

## 3. Residuals carried (explicit, none dropped)

| Residual | Source | Carried to |
|---|---|---|
| P31-001 leg 1: degenerate-binding refusal test at the Settle boundary | P31 review | F-1 (boundary hardening) — test-only, still open |
| P31-001 leg 2: boundary conflicting-repeat (first-decision-stands + single audit) | P31 review | F-1 — test-only, still open |
| P29-005 abandoned-PENDING wedge execution-unverified | Prompt30 closeout | F-7 cross-class recovery proofs (or explicit deferral with reason) |
| Operational revocation drill NOT EXECUTED | Prompt30 closeout | F-8 (needs authorized isolated replicas) |
| CI execution limitations | Prior closeouts | Each stage states executed-vs-configured |
| M5 sequence activation requirements | Lifecycle §7/M5 | Unchanged; no sequence workload without them |
| P24-008 capability-membership before real providers | Gate 1 | Unchanged; enforced in F-5+ provider work |
| Demo identity / CLI process-listing exposure | P28.1 memo | Closed by M1 |

P31-001 is carried as open test work: architecture
documentation does not resolve it, and no §11 stage
claims it closed without the two legs landing and
passing.

## 4. Consolidated owner-decision register

Pre-F-0 (needed to approve with confidence):
package review itself; confirm the S/O/DX-D
question set is complete.

| ID | Decision | Blocks |
|---|---|---|
| S-D1–S-D6 | IdP, delegation shape, approval UX, policy ceiling, vault standard, service issuance | F-1 per-row |
| O-D1–O-D6 | OTel versions, exporter config, backend refs, retention, SLO targets, registry process | F-2 per-row |
| DX-D1–DX-D6 | Façade shape, startup taxonomy, Unsafe shape, demo scope, suite location, analyzer schedule | F-3 per-row |
| C-D1–C-D3 | Suite location, CI budget, waiver rule | F-4 |
| R-D1 | Stage order confirmation (F-1∥F-2? B before C?) | F-1 start |
| R-D2 | B/C implementation owners + evidence bar | F-5 start |
| R-D3 | Drill environment authorization (replicas + credentials) | F-8 drill |

## 5. Risks

- Conformance skipped under B/C schedule
  pressure → suites gate the work orders (F-4
  before F-5/F-6, non-negotiable).
- OTel/dashboards scope creep into platform
  building → producer-only rule (O §1) with
  owner review on registry/dashboard additions.
- Façade growth into config sprawl → small-
  surface rule + DX-D1 review on additions.
- Aspirational drift (docs claim, code lacks)
  → status-tag discipline + per-stage
  executed-vs-configured reporting.
- Single-threaded owner review bottleneck →
  decisions are per-row with named blockers;
  independent rows proceed.

## 6. Program acceptance criteria

1. All F-0–F-8 exit proofs recorded with
   F19 evidence (test, revision, environment,
   command, result, date).
2. Every S/O/DX acceptance criterion maps to
   passing conformance rows (matrix §2–§3).
3. Existing suites (X/C/M/P28/P28.1/P30)
   green unmodified at every stage.
4. §9 invariants (C1–C7 incl. separate
   authorization transaction, global lock order,
   stable BusinessOpID + bounded recovery,
   atomic transitions + attempt fencing, Settle
   boundary, M5/P24-008 activation, staleness)
   hold at every stage — re-audited, not
   assumed.
5. All §3 residuals closed or explicitly
   re-carried with owner sign-off; nothing
   silently dropped.
6. The demo proves the doctrine: business
   logic only, framework owns the rest, under
   the full adversarial matrix.