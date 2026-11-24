# Miniapp Foundational Conformance Matrix

Status: PROPOSED architecture (Prompt32 §8). Nothing in this
document claims existing implementation unless tagged
[Existing]. Tags: [Existing] / [Partial] / [Documented] /
[Proposed] (legend in `foundational-architecture.md`).

Provenance: `docs/prompt32.md` §8; `docs/plan4.2.md` §18
(test pyramid, evidence register F19); `probe/classd`
(precedent: behavior-proving conformance probe);
determinism analyzer + AST-guard precedents.

Companion documents: `security-contract.md` (S),
`observability-contract.md` (O),
`developer-experience-enforcement.md` (DX),
`foundational-architecture.md` (unified runtime, glossary,
inventory), `foundational-roadmap.md`.

## 1. Conformance philosophy

An execution adapter that fails Security, Observability,
or execution-reliability conformance is not
production-ready — mechanically, not aspirationally: the
suites run in CI and gate the work orders that introduce
adapters. Suites test real behavior (persisted state,
emitted telemetry, observed refusals), never interface
presence. Precedent: `probe/classd` proves crash
recovery, isolation, audit reconstruction, and replay
equivalence by execution, not by signature checks; the
P28 guard chain proves structural properties with
negative controls. This package generalizes both
patterns to S/O/DX.

## 2. Shared suite (all classes, all adapters)

Every adapter passes every shared requirement. Each row
names the contract source, the asserted behavior, and
the test shape.

### Security (shared)

| ID | Requirement (source) | Asserted behavior | Test shape |
|---|---|---|---|
| CS-01 | Unauthenticated refusal (S §3.3) | Identical terminal, zero state, paired audit, all surfaces | Scripted calls per surface + DB + audit assert |
| CS-02 | Delegation enforcement (S §4.4) | Expired/forged/scope-exceeded delegation refused | Fixture delegations × capability matrix |
| CS-03 | Anti-spoofing structs (S §4.7) | No request struct carries identity/role/tenant/authority | AST assertion (P26-001 generalization) |
| CS-04 | Checkpoint re-verify (S §5.1) | Revocation between admission and checkpoint refuses | Time/lease-controlled interleaving |
| CS-05 | Redaction (S §3.4, O §6.1) | Credential-shaped fixtures in no emission | Adversarial payloads × all sinks |
| CS-06 | MCP parity (S §7) | Discovery gated; every call admitted | Session matrix (anon/authed/expired) |
| CS-07 | Worker privilege cap (S §5.3) | Worker authority ⊆ persisted record | Privilege-escalation fixtures refused |

### Observability (shared)

| ID | Requirement (source) | Asserted behavior | Test shape |
|---|---|---|---|
| CO-01 | Trace connectedness (O §5) | One trace admission→settle; IDs on every span | In-memory exporter span-tree assert |
| CO-02 | Metric registry closure (O §5.2, §8) | Exact names; no ID labels | Static enumeration + runtime label scan |
| CO-03 | Exporter failure (O §4.4) | Business unaffected; drops counted; audit whole | Kill/hang exporter mid-run |
| CO-04 | Coverage (O §7) | Every lifecycle row emits | Emission-matrix test; missing row fails |
| CO-05 | Audit pairing (O §3) | Security decisions have durable records | Decision × audit join assert |

### Developer experience (shared)

| ID | Requirement (source) | Asserted behavior | Test shape |
|---|---|---|---|
| CDX-01 | Import boundary (DX §12 criterion 1) | Business logic imports no framework-internals | Import-graph assertion per miniapp |
| CDX-02 | Startup refusal (DX §6 mechanism 2) | Bad config refuses to serve | Config matrix × boot assert |
| CDX-03 | Escape-hatch discipline (DX §7) | Undeclared bypass fails CI | Fixture bypasses × guard assert |
| CDX-04 | Bypass impossibility-by-API (DX §6 mechanism 1) | Authority params unconstructible | Compile-fail fixtures (must-not-build) |
| CDX-05 | Adversarial demo rows (DX §10) | All rows pass w/o demo plumbing edits | Demo suite (see roadmap milestone) |

## 3. Class-specific suites

Shared suite plus per-class rows. Each class suite
encodes what makes that class *different*.

Class A: single-tx atomicity under instrumentation
(spans/metrics add no tx behavior change); contention
recovery re-enters admission (existing C6 fixture
adopted as conformance); rollback emits no partial
telemetry claiming success.

Class B: enqueue atomicity with the motivating write;
worker pickup re-verify (CS-04 concretized with
River job rows); retry boundedness + dead-letter
audit; job-attempt lineage in traces.

Class C: item-level authority at distinct selectors;
progress records + resume-without-replay; batch
counters without per-item span explosion
(cardinality assertion); partial-failure outcomes
typed per item.

Class D: durable waits/signals with approval
re-verification; replay emits links not duplicate
spans (O criterion 7); cancellation ordering +
DETACH-ONLY; generation fencing where sequence-bound;
version-drain behavior under the family model.

Matrix (requirement × class — S = shared row
applies, C = class-specific row exists):

| Requirement family | A | B | C | D |
|---|---|---|---|---|
| Unauthenticated/delegation/spoofing (CS-01–03,06) | S | S | S | S |
| Checkpoint re-verify (CS-04) | n/a (single tx) | C | C | C |
| Redaction (CS-05) | S | S | S | S |
| Worker privilege (CS-07) | n/a | C | C | C |
| Trace/metric/coverage (CO-01,02,04,05) | S | S | S | S |
| Exporter failure (CO-03) | S | S | S | S (+replay-link) |
| Import/startup/escape (CDX-01–04) | S | S | S | S |
| Demo adversarial (CDX-05) | S | S | S | S |

## 4. Reuse mechanics [Proposed]

Adapter authors and miniapp developers run suites
without recreating framework fixtures:

- In-module `conformance/` package (location is
  owner decision C-D1): suite runners parameterized
  by adapter-provided hooks (admit, dispatch,
  pickup, effect-attempt, settle) against
  framework-owned fixtures (disposable PG, mock
  provider, in-memory OTel exporter, scripted
  credentials/delegations).
- Fixture injection, not fixture invention: the
  hooks receive fully-built scenarios (revocation
  mid-flight, exporter death, delegation expiry);
  the adapter only executes its normal path.
- DB needs: disposable PostgreSQL at the pinned
  digest (existing testkit convention); suites
  never touch shared or production databases.
- Negative controls throughout: every structural
  suite ships a planted-violation fixture proving
  the matcher is not blind (P24-009/P29-002
  precedent).
- Runtime cost: shared suite must complete in CI
  minutes, not hours — the matrix stays green by
  being runnable, not by being skipped.

## 5. CI integration [Proposed]

- Strict analyzer + structural suites run on every
  change touching the covered surface (existing
  `checks.sh` pattern, extended).
- Shared suite runs per adapter on every change to
  the adapter or the kernel.
- Class suites gate their adapter's work order
  acceptance (B/C cannot land red).
- Suite changes are versioned with the contracts:
  weakening an assertion needs the same review
  level as weakening the contract text.

## 6. Owner decisions required

| ID | Decision | Needed before |
|---|---|---|
| C-D1 | `conformance/` package location + public API shape | Suite implementation |
| C-D2 | CI time budget for the shared suite (parallel shards?) | CI wiring |
| C-D3 | Who may grant a conformance waiver, and its expiry/record rule | First waiver request (none planned) |

## 7. Acceptance criteria (suite implementation)

1. A deliberately broken adapter (skipped
   admission check, dropped span, widened
   authority) fails the shared suite on the
   corresponding row — red/green proven per row
   family at suite introduction.
2. All structural suites have negative controls
   that fail when the matcher is neutered.
3. Adapter authors add no fixtures to run the
   suites (reuse proof: B-adapter work order
   consumes the suite as-is).
4. Suite runtime fits the C-D2 budget on
   reference CI hardware.
5. No waiver exists without owner record +
   expiry (C-D3 audit).