# **Adversarial review: The Rulez-backed miniapp generator**

**Verdict: CONDITIONAL GO.** The corrected product architecture is viable, and I don't see a fundamental reason to abandon it. However, I would not approve the attached plan for implementation as written.

The hierarchy is now right:

- The **miniapp generator** is the product.
- The **miniapp type and primitive libraries** provide reusable application patterns.
- The **generator and compiler** turn business requirements into deployable miniapps.
- **Rulez** supplies deterministic, testable business logic.
- Integrations provide access to existing business systems.

There are, however, several important gaps. The most serious is an overclaim in the plan: a Rulez-backed application is not automatically safe to mutate. Rulez can make the decision logic easier to validate and audit, but it cannot establish that the underlying business requirement is correct, the input data is trustworthy, or the proposed action is authorized.

I would preserve the product vision but correct these issues before implementation.

## **1. Critical findings**

Critical · Must correct

**Finding 1 — Rulez does not make mutations safe by itself.**

The plan claims that Rulez-backed miniapps can safely mutate because their logic is declarative, bounded, fixture-gated, and deterministic. This is not established by those properties.&#x20;

Pasted text(20261122-044337).txt



A correct expression can use incorrect data. A passing fixture can miss an edge case. A valid decision can still be executed by an unauthorized user, against the wrong tenant, or after its underlying facts have changed.

This is more than wording. It affects the entire security architecture.

We must separate four guarantees:

| Guarantee            | What establishes it                                                                                            |
| -------------------- | -------------------------------------------------------------------------------------------------------------- |
| Rule validity        | Parsing, type checking, static validation                                                                      |
| Rule behavior        | Fixtures, independent expected results, scenario tests                                                         |
| Decision correctness | Valid business requirements, trustworthy facts, verified domain semantics                                      |
| Mutation safety      | Authentication, authorization, approved capabilities, transactional controls, idempotency, action verification |

Rulez principally addresses the first two. It contributes to the third, but cannot guarantee it. The fourth belongs to the integration and execution infrastructure.

**Required correction:** replace the expression "correct by construction" with:

> Business decisions are represented separately from generated application code, validated against explicit contracts, tested against independent fixtures, versioned, and evaluated deterministically.

That is a substantial advantage, without claiming mathematical certainty.

### **Finding 2 — The competitive comparison with GigaCatalyst is inaccurate**

Critical · Correct before using as product strategy

The attached document asserts that GigaCatalyst generates applications whose business logic is TypeScript and argues that it cannot safely support customer mutations without rebuilding its architecture around something equivalent to Rulez.&#x20;

Pasted text(20261122-044337).txt



I would reject that conclusion.

GigaCatalyst's publicly published material describes approved API access, inherited permissions, limited write capabilities, staging validation, human review, versioning, and rollback. Its Trust Center also describes granular write controls, including disabling destructive operations.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://gigacatalyst.com\&sz=32)

gigacatalyst.com+3



Its July 21, 2026 article specifically advocates:

- Approved API contracts.
- Explicit role and tenant permissions.
- Narrow write boundaries.
- Staging validation.
- Human review before publication.
- Versioned changes and rollback.&#x20;

  [image](https://www.google.com/s2/favicons?domain=https://gigacatalyst.com\&sz=32)

  gigacatalyst.com



These are vendor-published descriptions, not independent security audit results. But they are enough to invalidate our categorical claim that GigaCatalyst cannot address governed mutations.

Our differentiator should not depend on an assumption that a competitor lacks security controls.

**The more defensible differentiation is this:**

GigaCatalyst's publicly described model focuses on generating customer-specific applications through governed APIs and generated application code. Our proposed architecture additionally makes business decision logic an independently reusable, declarative, versioned, testable asset through Rulez.

That is a meaningful architectural difference. Whether it creates a better product must be demonstrated through actual applications and customer outcomes.

### **Finding 3 — Security and authorization are scheduled too late**

Critical · Architectural sequencing blocker

The plan places full multi-tenant isolation and `PolicySpec` enforcement in Phase 7, after the generator, miniapp compiler, and multi-domain validation.&#x20;

Pasted text(20261122-044337).txt



That is too late.

Even a generated read-only dashboard can expose customer records that the user should not see. A generated workflow can make a field writable that the source application normally restricts.

OWASP specifically requires tenant identity to be verified against the authenticated principal, rather than trusted from client-supplied tenant identifiers. It also emphasizes enforcement at trusted service boundaries, not in client-side application code.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://cheatsheetseries.owasp.org\&sz=32)

OWASP Cheat Sheet Series+2



GigaCatalyst's own published architecture also treats permissions and approved APIs as foundational rather than optional finishing features.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://gigacatalyst.com\&sz=32)

gigacatalyst.com+1



**Required correction:**

Move the following contracts and enforcement boundaries into Phase 0:

- Verified user and tenant context.
- `CapabilitySpec` and approved capability registry.
- Data-field and operation permissions.
- Miniapp permission manifest.
- Server-side authorization.
- Credential ownership and secret handling.
- API allowlists.
- Read/write classification.
- Audit requirements.
- Evaluation and execution resource limits.

Full enterprise identity, billing, advanced tenant administration, and large-scale isolation infrastructure can come later.

But the authorization architecture cannot.

I would also make the initial miniapps read-only, with writes introduced only after the action gateway has its own acceptance tests.

## **2. Major architectural findings**

### **Finding 4 — The MiniappSpec is not defined sufficiently**

Major · Required before compiler implementation

The plan identifies a `MiniappSpec`, but its actual structural contract remains mostly descriptive.

It says the model produces a specification containing miniapp types, primitives, data bindings, Rulez bindings, and fixtures.&#x20;

Pasted text(20261122-044337).txt



That is a good direction, but a compiler needs an exact contract.

I would require the following logical components:

| Specification          | Responsibility                                                                      |
| ---------------------- | ----------------------------------------------------------------------------------- |
| `MiniappSpec`          | Application identity, composition, layout, navigation, and configuration            |
| `DataBindingSpec`      | Queries, data transformations, field mappings, refresh behavior                     |
| `CapabilitySpec`       | Approved external operations, schemas, authentication requirements, and permissions |
| `RuleBindingSpec`      | References to specific immutable Rulez decisions and input/output schemas           |
| `PolicySpec`           | Who can view, modify, publish, share, and execute capabilities                      |
| `WorkflowBindingSpec`  | References to external durable workflows and their states                           |
| `ContractSnapshotSpec` | Pinned API and schema contracts                                                     |
| `ArtifactManifest`     | Exact versions and hashes needed to reproduce the generated miniapp                 |

This does not mean we need eight separate Go packages or eight independently deployed services.

They are conceptual contracts that need to be defined before implementation.

### **Finding 5 — The generated artifact should not necessarily be self-contained JavaScript**

Major · Reconsider compiler design

The current proposal is:

`MiniappSpec → self-contained HTML + JS artifact`

Pasted text(20261122-044337).txt

I would challenge this.

Why generate new executable JavaScript for every miniapp when we can generate a declarative specification and have a trusted, prebuilt runtime interpret it?

My preferred approach:

```
LLM
 |
 v
MiniappSpec (JSON)
 |
 v
Schema and security validation
 |
 v
Deterministic compiler
 |
 v
Canonical Miniapp artifact
 |
 v
Trusted Miniapp Runtime
 |
 +-- Render approved components
 +-- Resolve approved data bindings
 +-- Invoke Rulez
 +-- Invoke approved capabilities
 +-- Display decisions
```

The compiled artifact can be a versioned, content-addressed manifest rather than arbitrary application code.

This gives us several advantages:

- Less generated executable code.
- Smaller security surface.
- Consistent rendering.
- Easier application diffing.
- More predictable versioning.
- Centralized component fixes.
- Less opportunity for generated code to bypass platform controls.

We may eventually need specialized components or customer extensions. Those can use separately governed extension contracts and isolated execution.

There is nothing inherently wrong with generating JavaScript. I simply would not make it the default architecture before establishing that the declarative runtime cannot meet the requirements.

### **Finding 6 — The six miniapp types are a good starting library, but Rulez must not become mandatory everywhere**

The proposed taxonomy is:

1. Rule builder.
2. Calculator.
3. Dashboard.
4. Approval queue.
5. Reconciliation view.
6. Workflow controller.&#x20;

   Pasted text(20261122-044337).txt



I support this taxonomy as an initial pattern library.

However, I would change the Rulez binding requirements.

| Miniapp type        | Recommended architecture                                                                 |
| ------------------- | ---------------------------------------------------------------------------------------- |
| Rule builder        | Creates RuleSpecs; uses Rulez to validate and test them                                  |
| Calculator          | Rulez or an approved deterministic domain-calculation service                            |
| Dashboard           | Data queries, aggregations, and visualizations; Rulez optional                           |
| Approval queue      | Reads workflow state, evaluates recommendations, invokes an approved decision action     |
| Reconciliation view | Data matching and comparison services, with Rulez for applicable policies and exceptions |
| Workflow controller | Uses a durable workflow service; Rulez may decide routing and escalation                 |

Why?

A dashboard calculating monthly sales totals doesn't need a Rulez program for every aggregate. A long-running approval process should not become an artificial collection of rule evaluations.

The architecture should allow applications to combine Rulez with conventional services, queries, and workflows.

**Rulez should be ubiquitous as a foundation, not compulsory as an execution mechanism for every feature.**

### **Finding 7 — The&#x20;**`CapabilitySpec`**&#x20;needs to be more substantial**

The plan makes capabilities the external data and operation boundary, but does not yet give them a sufficiently detailed contract.

A capability needs more than an endpoint and a JSON schema.

I would include:

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

API discovery should produce candidate capability descriptions, not automatically approved production capabilities.

GigaCatalyst's published guidance makes this distinction too: discovered API information should be validated and approved, including endpoint permissions, data scopes, and write restrictions.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://gigacatalyst.com\&sz=32)

gigacatalyst.com+1



This capability layer is arguably as strategically important as the Rulez binding.

### **Finding 8 — The versioning model is too shallow**

The document refers to schema hashes, library versions, artifact hashes, RuleSpec versions, and deterministic compilation.&#x20;

Pasted text(20261122-044337).txt



Good, but I would require an explicit dependency manifest.

An installed miniapp must pin:

- MiniappSpec version.
- Component-library version.
- Compiler version.
- Runtime version.
- Data-model/schema version.
- Capability versions.
- API contract snapshots.
- Rulez engine and language versions.
- RuleSpec and DecisionSpec versions.
- Domain-pack versions.
- Policy version.

Otherwise, an apparently unchanged miniapp could behave differently after an integration or runtime update.

There are also two different determinism claims:

1. Given a fixed MiniappSpec and fixed dependencies, the compiler should produce the same artifact.
2. Given a fixed user requirement, an LLM may produce a different proposed specification on another run.

The second cannot be treated as deterministic simply because the compiler is deterministic.

We need to retain the original requirement, model identity/version, generation configuration, generated spec, validation results, and all dependency hashes.

For rules, effective dating must also distinguish business-valid time from when the system learned, recorded, approved, and published a version.

### **Finding 9 — The falsification protocol needs correction**

Major · Change before using as acceptance gate

I like the two-axis falsification idea:

- Axis 1: Can Rulez express the domain's decision logic?
- Axis 2: Can the generator create the right type of miniapp for the requirement?

This is one of the strongest ideas in the attached document.&#x20;

Pasted text(20261122-044337).txt



But three acceptance criteria are problematic.

**First: "Reach every field in the schema."**

This is not desirable.

Some fields may be sensitive, irrelevant, administrative, derived, or not intended for user access. A good generator must select the authorized fields needed for the task, not expose every field.

Replace this with:

- Every required input is accounted for.
- Every displayed field has an approved data binding.
- Every requested operation maps to an approved capability.
- Unauthorized fields and operations remain inaccessible.
- Missing or ambiguous facts are explicitly reported.

**Second: "Generate in two prompts."**

This is an arbitrary target.

A straightforward dashboard and a complicated payroll exception application should not have the same prompt expectation.

A well-designed assistant may need to ask five clarification questions to avoid building the wrong payroll application. That is better than generating an incorrect one in two prompts.

Measure:

- Clarification count.
- First-pass specification validity.
- Number of automated repair iterations.
- Human corrections required.
- Task completion rate.
- Correctness against independent fixtures.
- User time to complete a representative task.

Use separate complexity categories.

**Third: "90% generation success" or "zero corrections" needs real test populations.**

The plan should not be allowed to improve its success score by testing only trivial prompts.

Use a benchmark of predetermined requirements with normal cases, ambiguous cases, edge cases, missing data, and hostile or unauthorized requests.

The evaluator must be independent of the generator.

### **Finding 10 — The proposed moat is overstated**

The document claims that the miniapp type library, generator/compiler pipeline, and falsification process are difficult to replicate.&#x20;

Pasted text(20261122-044337).txt



These can become valuable assets, but none is inherently defensible.

A competitor can develop a similar library of forms, dashboards, tables, and approval queues. A compiler that transforms JSON specifications into a rendered application is also reproducible.

The more defensible assets are likely to emerge from:

- High-quality integrations and capability contracts.
- Accurate data-to-business-semantic mapping.
- Domain-specific decision packs with independently verified tests.
- Reliable generated applications that customers use repeatedly.
- Customer-specific customization and adoption history.
- A growing library of tested patterns across real requirements.
- Deployment, security, and governance practices trusted by SaaS vendors.

I would treat the pattern library as a compounding product asset, not the moat itself.

## **3. Sequencing and scope findings**

The phases in the attached plan are:

| Phase                             | Proposed duration |
| --------------------------------- | ----------------- |
| Rulez fork and RuleSpec envelope  | 0.5 day           |
| Versioning, composition, fixtures | 1.5 days          |
| Effects and DecisionSpec          | 2–3 days          |
| Go backend                        | 2–3 days          |
| Miniapp primitives and types      | 4–5 days          |
| Compiler and generator            | 4–5 days          |
| Falsification                     | 5–6 days          |
| Productization                    | Later             |

Pasted text(20261122-044337).txt

The total before productization is approximately 19–23 days.

That is a reasonable prototype program, but I have a sequencing objection.

**The miniapp generator is the product, yet the first substantial end-to-end miniapp appears only after the Rulez runtime, rule registry, effects, and Go backend are built.**

This creates the risk that we spend several weeks producing an excellent rules engine without proving that our actual generator produces useful applications.

I would prefer an early end-to-end vertical slice.

### **Recommended sequence**

| Phase | Main goal                                                                        | Exit evidence                                                 |
| ----- | -------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| 0     | Inspect Talang; freeze MiniappSpec, capability, RuleSpec, and security contracts | Reviewed architecture and threat model                        |
| 1     | Minimal Rulez kernel and one manually authored MiniappSpec                       | Working miniapp invokes a pinned Rulez decision               |
| 2     | Trusted miniapp runtime and primitive library                                    | Calculator, dashboard, and rule-builder patterns work         |
| 3     | AI generator and deterministic compiler                                          | Prompt produces a valid miniapp spec and runnable application |
| 4     | Versioning, composition, fixtures, governance                                    | Changes are validated, reviewable, and reversible             |
| 5     | Data discovery and integration proxy                                             | One real approved data source can power a miniapp             |
| 6     | Write-enabled capabilities                                                       | Narrow, authorized, audited mutation works safely             |
| 7     | Domain falsification and product hardening                                       | Promotions, payroll, IRS, and additional domain evidence      |

I am not proposing that we postpone the Rulez foundation. It should be implemented from the beginning.

I am proposing that we prove the foundation inside the actual product much earlier.

There is another issue in the current Phase 0: it says to fork Talang and create the RuleSpec envelope in half a day, without grammar changes. The previous agreed approach required investigation and independent adversarial review before deciding what to fork or modernize.

That research gate must remain.

Talang is MIT-licensed, so it is a reasonable candidate for a fork. However, the public repository's latest commit I found is October 12, 2021. This does not disprove Talon's internal production experience, but the attached claim of inheriting "five-plus years of hardening" should not be treated as evidence about the publicly available implementation.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1

&#x20;The actual code, dependency state, tests, and design must be inspected.



## **4. Additional issues that deserve architectural decisions**

There are several details that might look minor but will have expensive consequences later.

### **A. What exactly counts as business logic?**

The assertion that generated artifacts contain no business logic is too absolute.&#x20;

Pasted text(20261122-044337).txt



A miniapp still needs presentation logic: formatting, sorting, navigation, local form state, empty-state handling, and temporary calculations.

I would replace that invariant with:

> Generated miniapps must not contain authoritative business decisions, security-sensitive authorization decisions, credentials, or direct database access. Such behavior is delegated to trusted backend services, Rulez, and approved capabilities.

This gives the miniapp runtime enough flexibility without allowing it to become an uncontrolled application backend.

### **B. The data-discovery problem is underestimated**

The plan starts with a business requirement, schema, and sample data.&#x20;

Pasted text(20261122-044337).txt



That is a necessary starting point, but a schema cannot tell an AI what every field means.

A production discovery pipeline should eventually acquire:

- API contracts and endpoint descriptions.
- Field types, descriptions, and relationships.
- Business definitions and units.
- Sample records with explicit provenance.
- Read/write permissions.
- Tenant and object-level access boundaries.
- Valid state transitions.
- Error responses and behavioral constraints.

For example, an `amount` field might represent gross, net, taxable, or payable value. These meanings cannot be safely inferred from its data type alone.

GigaCatalyst's own API-discovery article makes this point: API documentation alone is insufficient because permissions, tenant boundaries, field meanings, and approved write behavior are often distributed across different sources.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://gigacatalyst.com\&sz=32)

gigacatalyst.com+1



I would add a **Schema Understanding and Semantic Mapping** stage, with explicit confidence and human confirmation for ambiguous mappings.

### **C. The workflow-controller type must not become a workflow engine**

Your sixth miniapp type is a workflow controller.

That is a useful user interface, but a workflow involving waits, retries, callbacks, approvals, recovery, and multi-day execution needs a durable execution backend.

The miniapp should control and observe that process. Rulez can decide routing and escalation. An external workflow engine should own authoritative process state.

This avoids turning Rulez into a competing orchestration system.

### **D. The "code-step" fallback needs a stronger boundary**

The document proposes explicit `requires code-step` markers when the generator cannot express a requirement using known miniapp types.&#x20;

Pasted text(20261122-044337).txt



I support the honesty principle, but I would not allow the generator to silently fill such a gap with unrestricted generated code.

The marker should mean:

`UNSUPPORTED_CAPABILITY`

It should identify what is missing, why it is required, and what approved integration or developer extension would resolve it.

That keeps the boundary between declarative generation and arbitrary execution intact.

# **5. Revised architectural decisions**

Here is what I would change in the attached plan before authorizing implementation.

| Decision        | Existing proposal                         | My recommendation                                                                        |
| --------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------- |
| Product         | Miniapp generator                         | Retain                                                                                   |
| Rulez           | Main differentiator                       | Retain, but remove guaranteed-correctness claims                                         |
| RuleSpec        | Authoritative business logic              | Retain                                                                                   |
| MiniappSpec     | LLM-generated application contract        | Define precisely before compiler work                                                    |
| Rulez binding   | Used by nearly every miniapp type         | Optional, whenever explicit business decision logic adds value                           |
| Compiler        | Self-contained HTML/JS                    | Prefer declarative artifacts and trusted runtime                                         |
| API discovery   | Schema and sample data                    | Expand to permissions, semantics, approved capabilities, and contract snapshots          |
| Security        | Full PolicySpec in Phase 7                | Security contracts and server-side enforcement from Phase 0                              |
| Mutations       | Enabled by Rulez-backed logic             | Enabled only by approved, authorized capabilities                                        |
| Rule lifecycle  | Registry and `as_of`                      | Immutable definitions, pinned dependency graph, separate activation and temporal history |
| AI generation   | Prompt to MiniappSpec                     | Requirement interpretation, clarification, generation, validation, and repair            |
| Pattern library | Six initial miniapp types                 | Retain as initial hypotheses with extension mechanisms                                   |
| Falsification   | Four domains, two axes                    | Retain, but improve measurement and independence                                         |
| Moat            | Rulez, patterns, generator, falsification | Add integrations, semantic mappings, customer adoption and operational evidence          |

## **6. The corrected implementation order**

I would retain most of the existing workstreams but change their order to establish the actual product earlier.

1. **Phase 0 — Architecture and security foundation**

   Inspect Talang, independently review findings, define MiniappSpec, RuleSpec, CapabilitySpec, PolicySpec and compiler contracts. Define the threat model and approval boundaries.

   Architecture gate
2. **Phase 1 — First integrated vertical slice**

   Implement the minimum Rulez evaluator, a manually authored MiniappSpec, a trusted renderer, and one mock or approved data capability.

   Demonstrate one working miniapp that invokes Rulez.

   Product thesis gate
3. **Phase 2 — Reusable miniapp library**

   Implement the six initial miniapp types and their component primitives. Establish component schemas, composition rules, and versioning.
4. **Phase 3 — AI generation pipeline**

   Generate MiniappSpec from requirements, data schemas, and available capabilities. Validate, repair, compile, and render.
5. **Phase 4 — Full Rulez management**

   Expand composition, immutable versions, effective dating, fixtures, DecisionSpec, explanations, and rule governance.
6. **Phase 5 — Integration and controlled mutations**

   Build approved API bindings, credential management, authorization, action execution, idempotency, audit, and reconciliation.
7. **Phase 6 — Falsification and hardening**

   Test different domains and miniapp types against independently verified scenarios. Address gaps before expanding the library.
8. **Phase 7 — Productization**

   Tenant isolation, sharing, publication, marketplace, usage metering, customer administration, and enterprise deployment options.

The difference is that we prove a generated application and its Rulez integration early, rather than spending the first several phases building infrastructure without demonstrating the product.

# **7. What I would require before approving implementation**

I would make these six items explicit approval gates.

| Gate | Requirement                                                   | Blocking?                            |
| ---- | ------------------------------------------------------------- | ------------------------------------ |
| G1   | Talang source inspection and independent adversarial review   | Yes                                  |
| G2   | Define the MiniappSpec and trusted runtime boundary           | Yes                                  |
| G3   | Define RuleSpec, decision outcomes, and composition contracts | Yes                                  |
| G4   | Define capability and authorization boundaries                | Yes, especially before writable apps |
| G5   | Prove a complete miniapp-to-Rulez-to-JSON execution path      | Yes                                  |
| G6   | Establish domain and generator falsification methodology      | Before claiming generality           |

For G4, full enterprise security productization can be later. But the initial architecture must establish that generated applications cannot acquire new privileges simply because the LLM generated a button or a workflow.

For G5, I would not accept a separate Rulez evaluator demo and a separate static miniapp demo. They must work together.

# **8. Final go/no-go assessment**

# **CONDITIONAL GO**

Three architectural corrections required

**The underlying product architecture is sound enough to pursue. The current document is not yet an implementation-ready specification.**

## **What I approve**

- The miniapp generator as the main product.
- Rulez as a reusable business decision foundation.
- Declarative MiniappSpec generation.
- A small, composable miniapp type library.
- Talang as the starting language candidate.
- Independent domain and generator falsification.
- Separating external actions from rule evaluation.
  ## **What I reject**
- Claiming Rulez makes mutations safe by itself.
- Claiming GigaCatalyst cannot provide governed mutations.
- Deferring security contracts until productization.
- Requiring Rulez for every miniapp type.
- Treating complete schema-field coverage as a quality metric.
- Assuming a deterministic compiler makes AI generation deterministic.

## **9. My most important recommendation**

I would change the document's one-sentence differentiator.

The current statement is that GigaCatalyst generates applications whose business logic is code, whereas Rulez generates applications whose business logic is RuleSpec.&#x20;

Pasted text(20261122-044337).txt



I would replace it with:

> **We turn business requirements and existing data capabilities into deployable miniapps, using a reusable, deterministic, versioned decision engine to keep business logic independently testable, governable, and reusable across applications.**

This gives us a stronger claim, because it doesn't depend on disparaging a competitor or pretending that deterministic rules alone guarantee safe applications.

The differentiator is the combination of the generator, the declarative application model, the trusted integration boundary, and Rulez.

**My final judgment:** do not abandon or fundamentally redesign this product. Correct the three structural issues, prove one complete generated miniapp, and then expand the reusable foundations. The biggest danger is not that Rulez will fail to evaluate a rule. It is that we build an impressive engine, compiler, and component library without proving that the platform can consistently generate applications that customers genuinely need and trust.