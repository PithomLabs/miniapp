# Revised Plan: Disposition of the Adversarial Review

## 1. Disposition summary

**Accepted without modification (10 findings):**

| Finding | Correction adopted |
|---|---|
| 1. Rulez ≠ mutation safety | Four-guarantee separation; drop "correct by construction"; adopt the reviewer's replacement language |
| 2. Inaccurate GigaCatalyst comparison | Drop categorical claim; adopt the architectural-difference framing |
| 3. Security scheduled too late | Security contracts + server-side enforcement in Phase 0 |
| 4. MiniappSpec underspecified | Adopt the eight conceptual contracts before compiler work |
| 5. Generated JS artifact | Prefer declarative artifact + trusted runtime; generated JS only via governed extension contracts |
| 6. Rulez mandatory everywhere | Rulez optional per type; adopt the reviewer's type/binding table |
| 7. CapabilitySpec too thin | Adopt the expanded contract; discovery produces candidates, not approved capabilities |
| 8. Versioning too shallow | Explicit dependency manifest; distinguish compiler determinism from generation determinism |
| 9. Falsification criteria | Replace "every field / two prompts / 90%" with the reviewer's corrected metrics and independent evaluator |
| 10. Moat overstated | Reframe as compounding assets; the tested pattern library is the durable part, not the library itself |

**Accepted with clarification (3):**

- **Finding 6:** Rulez optional per type is correct. The additional argument in favor of *optional* rulez is that the platform maintains **one governance story** rather than two — even a dashboard can optionally reference rulez for derived metrics without being forced to, which keeps the governance surface unified.
- **Finding 5:** The declarative-runtime shift **simplifies** the compiler rather than complicating it. "Resolve `MiniappSpec` against the library → emit content-addressed manifest" is strictly less work than "emit HTML+JS," and it eliminates the generated-code attack surface.
- **Finding 10:** The compounding asset is the **tested** pattern library — patterns validated against real requirements across domains — not the library as a catalog. A competitor can copy patterns; copying the validation history is much harder.

**Pushed back on (minor, non-blocking):**

- **Phase 0 duration.** Accepting the G1 research gate means Phase 0 is **1–2 days**, not 0.5. This is a cost, not a rejection.
- **G1 evidence claim.** The public `talon-one/talang` repo's last commit is Oct 2021, so "5+ years of hardening" applies to Talon.One's *internal* use, not the public implementation. The inspection must assess what the public code actually provides, not what the vendor's production usage implies.

**Additional extensions I would add beyond the review (5):**

1. **The generator quality matrix should be a customer-facing artifact**, not only an internal metric. A customer seeing "here is how the generator performs on your domain vs. similar domains" is a trust-building tool.
2. **Phase 1 must include audit logging from day one.** Even a read-only miniapp without audit is a governance hole.
3. **`UNSUPPORTED_CAPABILITY` must produce a routable work item**, not just a display marker — it turns falsification findings into a backlog.
4. **A named security owner** must sign off on G4 and review every capability addition. Phase assignment is not ownership.
5. **Every generated artifact carries a generation record** (original requirement, model identity, generation config, generated spec, validation results) alongside its dependency manifest — this is what makes the LLM-generation audit tractable.

---

## 2. The corrected core claims

**Differentiator (replaces the previous one-sentence version):**

> We turn business requirements and existing data capabilities into deployable miniapps, using a reusable, deterministic, versioned decision engine to keep business logic independently testable, governable, and reusable across applications.

**Guarantee language (replaces "correct by construction"):**

> Business decisions are represented separately from generated application code, validated against explicit contracts, tested against independent fixtures, versioned, and evaluated deterministically.

**Artifact invariant (replaces "no business logic in artifact"):**

> Generated miniapps must not contain authoritative business decisions, security-sensitive authorization decisions, credentials, or direct database access. Such behavior is delegated to trusted backend services, Rulez, and approved capabilities.

**The four guarantees, separated:**

| Guarantee | Established by | Owned by |
|---|---|---|
| Rule validity | Parsing, type checking, static validation | `rulez` engine |
| Rule behavior | Fixtures, independent expected results, scenario tests | `rulez` fixtures + domain packs |
| Decision correctness | Valid business requirements, trustworthy facts, verified domain semantics | Semantic mapping layer + domain owner |
| Mutation safety | Authentication, authorization, approved capabilities, transactional controls, idempotency, action verification | Integration and execution infrastructure |

Rulez addresses the first two fully, contributes to the third, and does not establish the fourth.

---

## 3. The eight conceptual contracts

These are contracts, not necessarily eight Go packages. They must be defined before compiler implementation.

| Contract | Responsibility |
|---|---|
| `MiniappSpec` | Identity, composition, layout, navigation, configuration |
| `DataBindingSpec` | Queries, transforms, field mappings, refresh behavior |
| `CapabilitySpec` | Approved external operations, schemas, auth requirements, permissions, scopes, read/write classification, risk class, rate limits, idempotency, timeout/retry, audit, error contract, fixtures |
| `RuleBindingSpec` | References to immutable rulez decisions + input/output schemas |
| `PolicySpec` | Who can view, modify, publish, share, execute; approval thresholds; spend caps |
| `WorkflowBindingSpec` | References to external durable workflows and their states |
| `ContractSnapshotSpec` | Pinned API and schema contracts |
| `ArtifactManifest` | Exact versions and hashes required to reproduce the miniapp |

**CapabilitySpec in full:**

```
CapabilitySpec
 ├── identity and version
 ├── provider and operation
 ├── input schema
 ├── output schema
 ├── authentication requirements
 ├── authorization scopes
 ├── tenant and record scope
 ├── read/write classification
 ├── risk classification
 ├── rate limits
 ├── idempotency requirements
 ├── timeout and retry semantics
 ├── audit requirements
 ├── error contract
 └── test fixtures
```

API discovery produces **candidate** capability descriptions. Approval is a separate, reviewed step.

---

## 4. The architecture: declarative artifact + trusted runtime

```
LLM
 │
 ▼
MiniappSpec (JSON)  +  schema/sample data  +  capability catalog
 │
 ▼
Deterministic validation (fields exist, types match, capabilities approved,
                           rulez bindings registered, policy satisfied)
 │   │
 │   └── failure → regenerate with specific error (bounded, ≤3)
 ▼
Deterministic compiler
 │
 ▼
Content-addressed Miniapp artifact (manifest + resolved dependencies)
 │
 ▼
Trusted Miniapp Runtime
 ├── Render approved components (from the primitive library)
 ├── Resolve approved data bindings (through capability proxy)
 ├── Invoke Rulez (for declared decisions)
 ├── Invoke approved capabilities (for declared operations)
 ├── Display decisions, confidence, and explanations
 └── Audit every action
```

**The artifact is a manifest, not executable code.** The runtime is trusted, prebuilt, and versioned. The compiler's output is content-addressed and diffable. Generated JavaScript is not the default; it is only reachable through governed extension contracts with isolated execution.

---

## 5. The miniapp type / rulez binding table

| Miniapp type | Rulez binding | Why |
|---|---|---|
| Rule builder | **Produces** `RuleSpec`; uses rulez to validate/test | Self-hosting type |
| Calculator | **Binds** to one or more `RuleSpec`; or approved deterministic domain-calc service | Business decision is the point |
| Dashboard | **Optional** — may reference rulez for derived metrics | Aggregations don't need rule programs |
| Approval queue | **Binds** to `RuleSpec` for recommendations; invokes approved decision action | Recommendation is a decision; execution is a capability |
| Reconciliation view | **Optional** — uses data-matching service; rulez for exception predicates | Matching is data; exception policy is decision |
| Workflow controller | **Optional** — rulez for routing/escalation; durable workflow engine owns process state | Process state is not a rule |

Rulez is **ubiquitous as a foundation, optional as an execution mechanism**. The platform maintains one governance story: every miniapp that makes a decision declares a `RuleBindingSpec`, whether or not that binding is the primary mechanism.

---

## 6. The revised phased plan

| Phase | Main goal | Exit evidence |
|---|---|---|
| **0** | **Architecture and security foundation.** Inspect Talang (public code + dependency state). Define `MiniappSpec`, `RuleSpec`, `CapabilitySpec`, `PolicySpec`, `RuleBindingSpec`, `WorkflowBindingSpec`, `ContractSnapshotSpec`, `ArtifactManifest`. Define threat model and approval boundaries. Name a security owner. | G1 + G2 + G3 + G4 signed off |
| **1** | **First integrated vertical slice.** Minimal `rulez` evaluator (forked, modernized, no grammar changes). One manually authored `MiniappSpec`. Trusted renderer. One mock or approved read-only data capability. Audit logging from day one. | G5: a working miniapp invokes a pinned `rulez` decision through the full path, with audit trail |
| **2** | **Reusable miniapp library.** Implement the six miniapp types and their component primitives. Component schemas, composition rules, versioning. | All six types work with at least one manual `MiniappSpec` each |
| **3** | **AI generation pipeline.** Requirement + schema + sample data + capability catalog → `MiniappSpec`, with clarification, validation, repair, and a generation record. | G6: generator produces a valid, runnable miniapp for one domain with measured metrics |
| **4** | **Full rulez management.** Composition, immutable versions, effective dating (business-valid time **and** system-recorded time), fixtures, `DecisionSpec`, explanations, rule governance. | Composed rules with independent effective windows and a full dependency manifest |
| **5** | **Integration and controlled mutations.** Approved API bindings, credential management, server-side authorization, action execution, idempotency, audit, reconciliation. | Narrow, authorized, audited mutation works end-to-end through an approved capability |
| **6** | **Falsification and hardening.** Domain × type matrix against the corrected protocol (see §7). Address gaps before expanding the library. | ≥3 of 4 domains pass both axes within the corrected criteria; matrix published |
| **7** | **Productization.** Tenant isolation, sharing, publication, marketplace, usage metering, enterprise deployment options. | — |

**The key change from the previous plan:** the vertical slice is Phase 1, not Phase 5. The infrastructure (registry, effects, `DecisionSpec`, backend) is expanded in Phases 4–5 *after* the product thesis is proven with a working generated miniapp.

---

## 7. The six approval gates

| Gate | Requirement | Blocking |
|---|---|---|
| **G1** | Talang source inspection + independent adversarial review of findings | Yes |
| **G2** | `MiniappSpec` and trusted runtime boundary defined | Yes |
| **G3** | `RuleSpec`, decision outcomes, composition contracts defined | Yes |
| **G4** | Capability and authorization boundaries defined; security owner named | Yes, especially before any writable miniapp |
| **G5** | End-to-end miniapp → rulez → JSON path proven working | Yes |
| **G6** | Domain and generator falsification methodology established | Before any claim of generality |

G5 must not accept a separate rulez demo and a separate static miniapp demo. They must work together, through the same runtime, with audit logging.

---

## 8. The corrected falsification protocol

**Axis 1 (engine genericity)** is unchanged from the previous plan: four domains — promotions, Singapore CPF, simplified IRS withholding, inventory reorder — must each be expressible as `RuleSpec` packs with fixtures. Falsification findings are logged with documented escape-hatch responses.

**Axis 2 (generator genericity)** is corrected:

**Replaces "reach every field"** with:
- Every required input is accounted for.
- Every displayed field has an approved data binding.
- Every requested operation maps to an approved capability.
- Unauthorized fields and operations remain inaccessible.
- Missing or ambiguous facts are explicitly reported.

**Replaces "two prompts"** with measured metrics per complexity category:
- Clarification count
- First-pass specification validity
- Automated repair iterations
- Human corrections required
- Task completion rate
- Correctness against independent fixtures
- User time to complete a representative task

**Replaces "90% / zero corrections"** with a benchmark population:
- Normal cases
- Ambiguous cases
- Edge cases
- Missing data
- Hostile or unauthorized requests

The evaluator must be independent of the generator. The matrix (domain × type × complexity) is the product's quality report and a customer-facing trust artifact.

---

## 9. The reframed moat

The moat is not the engine, the library, or the compiler. It is the **compounding asset set**:

- High-quality integrations and capability contracts.
- Accurate data-to-business-semantic mapping (with confidence and human confirmation).
- Domain-specific decision packs with independently verified tests.
- Reliable generated miniapps that customers use repeatedly.
- Customer-specific customization and adoption history.
- A growing library of **tested** patterns across real requirements.
- Deployment, security, and governance practices trusted by SaaS vendors.

The pattern library is one compounding asset among several. The durable part is the **validation history** — patterns tested against real requirements, capabilities tested against real APIs, semantic mappings confirmed against real data. That history cannot be copied from a marketing page.

---

## 10. What I would add beyond the review

1. **Generation record as a first-class artifact.** Every generated miniapp carries: original requirement, model identity and version, generation configuration, generated `MiniappSpec`, validation results, and the full dependency manifest. This operationalizes the reviewer's "two determinism claims" distinction — the second determinism (LLM generation) is not reproducible, but it is auditable.

2. **Schema Understanding and Semantic Mapping as a Phase 3 sub-stage.** The reviewer notes that an `amount` field might mean gross, net, taxable, or payable. This is not a note — it is a named sub-stage with explicit confidence scores and human confirmation for ambiguous mappings. It belongs in the generation pipeline, not in a later phase.

3. **`UNSUPPORTED_CAPABILITY` produces a routable work item.** When the generator hits a gap, it emits a structured ticket: what capability is missing, why it is required, what would be needed to add it, and which existing capability is the closest match. Falsification findings become a backlog, not just a log.

4. **Named security owner.** Phase assignment is not ownership. A named individual signs G4 and reviews every capability addition, including approved ones. The cadence is defined in Phase 0.

5. **Audit logging from Phase 1, not Phase 5.** Even the Phase 1 read-only slice must log every capability invocation and every rulez evaluation. The audit trail is the governance product; it must exist before writes are enabled.

6. **The quality matrix as a customer artifact.** Internal metrics are necessary but not sufficient. Customers should see: "for your domain and miniapp type, our generator's first-pass validity is X%, average clarifications Y, average repair iterations Z, and here are the cases where it required human correction." Transparency is the trust wedge that GigaCatalyst's vendor-published claims cannot match without equivalent measurement.

---

## 11. Final disposition

**The plan is approved for implementation with the corrections above.** The product thesis is retained. The one-sentence differentiator, guarantee language, and artifact invariant are replaced. The security boundary moves to Phase 0. The vertical slice moves to Phase 1. The falsification protocol uses the corrected metrics. The moat is reframed as compounding assets.

**The biggest remaining risk is not the engine, the compiler, or the pattern library.** It is that the team builds an impressive substrate without proving a single generated miniapp that a real customer uses and trusts. Phase 1 exists specifically to falsify that risk before Phases 2–7 invest in infrastructure. If Phase 1 fails — if the vertical slice does not produce a working, audited, rulez-backed miniapp — the plan should stop and the architecture should be reconsidered, not rescued by additional phases.

The next concrete step is the **Phase 0 implementation prompt**, scoped to: Talang source inspection, the eight contracts, the threat model, and the naming of the security owner. Phase 1 (the vertical slice) gets its own prompt after G1–G4 are signed off.