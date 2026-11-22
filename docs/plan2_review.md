# **Adversarial review: revised Rulez-backed Miniapp Generator plan**

Conditional approval — revisions still required

**I see no fundamental reason to abandon the product thesis or the overall architecture. However, I would not approve this exact plan for implementation unchanged.** It has four structural issues that should be resolved before Phase 1, plus several important weaknesses in its security and falsification gates.

The most consequential problem is that the plan says the first vertical slice will falsify the product risk, but that slice does not include the generator. It proves that a manually authored miniapp can invoke Rulez; it does not prove that the platform can turn a business requirement into a working miniapp.

The other issues concern a premature commitment to Talang, an inconsistent decision-binding contract, and postponing essential Rulez versioning semantics until after the first integrated slice.

## **1. Blocking issues**

### **1.1 Talang has effectively been selected before the investigation**

Phase 0 says to inspect Talang and obtain independent adversarial review. But Phase 1 already commits to a “forked, modernized” evaluator with “no grammar changes.” That is a decision about the implementation before G1 has produced its evidence. The plan has therefore left itself little room to discover that Talang should be adapted differently or replaced.&#x20;

plan2(20261122-045224).md



This is especially important because Rulez's intended requirements include exact decimal arithmetic, JSON objects and arrays, reusable expressions, composition, explicit outcomes and bounded evaluation. The plan has not yet demonstrated that Talang can support these requirements without grammar changes.

**Required correction:** G1 must end with an architecture decision record, not merely a source-inspection report. Compare:

- Reuse Talang substantially unchanged.
- Fork and adapt its parser, evaluator or type system.
- Retain selected Talang components while implementing missing semantics separately.
- Replace it with a purpose-built evaluator if the evidence warrants it.

The decision must include a compatibility test against the minimum Rulez requirements, dependency and license findings, resource-bound behavior, determinism, diagnostics and test coverage.

“No grammar changes” can be the conclusion of that investigation. It should not be an assumption made before it.

### **1.2 Phase 1 does not test the central product proposition**

This is the most important issue.

The product is a miniapp generator. But Phase 1 uses a **manually authored** `MiniappSpec`. The AI generation pipeline does not begin until Phase 3, after Phase 2 has implemented all six miniapp types.&#x20;

plan2(20261122-045224).md



Consequently, G5 proves:

> A manually specified miniapp can execute a pinned Rulez decision through the trusted runtime.

It does not prove:

> A business requirement can be converted into a useful, working miniapp.

Those are different propositions.

The distinction matters because the plan's final section says that Phase 1 should stop the project if the vertical slice fails, rather than allowing the team to rescue it with more infrastructure. But the proposed slice would not expose the primary risk: whether the generator understands requirements, data semantics, capabilities and the intended user task.&#x20;

plan2(20261122-045224).md



**Required correction:** Put a minimal generation path into Phase 1:

```
Business requirement
        ↓
LLM generates MiniappSpec
        ↓
Deterministic validation
        ↓
Trusted miniapp runtime
        ↓
Rulez decision
        ↓
Mock or approved read-only capability
        ↓
Audited, usable miniapp
        ↓
Representative user task completed
```

Keep a manually authored `MiniappSpec` as a control case. It helps distinguish a runtime failure from a generation failure.

Start with one domain and one miniapp type. Do not build all six types before testing the generator. Expand the type library after the first generated app succeeds.

**This is a structural correction, not a request to build a sophisticated AI agent in Phase 1.** The initial generator can be narrow and imperfect. It simply needs to exercise the actual product proposition end to end.

### **1.3 The plan contradicts itself about whether Rulez is optional**

Section 5 correctly says that Rulez is optional for dashboards and reconciliation views, and that some calculations may use approved deterministic domain services. But it then says that every miniapp making a decision declares a `RuleBindingSpec`, even when Rulez is not its primary execution mechanism.&#x20;

plan2(20261122-045224).md



That creates an architectural ambiguity:

- Is `RuleBindingSpec` specifically a reference to Rulez?
- Or is it actually a generic interface for all decision mechanisms?

If it is Rulez-specific, requiring it for decisions performed by other services creates unnecessary coupling. If it is generic, the name and contract are misleading.

For example, a reconciliation service might match records using an approved matching algorithm. Its result may be used in a subsequent Rulez decision, but record matching itself does not necessarily need to be a Rulez program.

**Required correction:** Introduce a generic `DecisionBindingSpec`, with a Rulez-specific implementation such as `RuleBindingSpec`.

Conceptually:

```
DecisionBindingSpec
    ├── RulezDecisionBinding
    ├── ApprovedDomainFunctionBinding
    └── Other explicitly approved decision providers
```

This is a contract boundary, not a requirement to create an elaborate plugin ecosystem immediately.

Also distinguish three things:

1. Data transformations: projection, formatting and other non-authoritative transformations.
2. Domain calculations: approved, deterministic calculations that may live outside Rulez.
3. Business decisions: explicit outcomes such as eligibility, approval, exception classification or policy routing.

Do not turn every derived number or dashboard aggregation into a Rulez decision just to maintain a uniform governance story.

### **1.4 Phase 1 requires guarantees that Phase 4 postpones**

G5 requires a miniapp to invoke a **pinned Rulez decision**. But Phase 4 is where the plan introduces immutable versions, composition, fixtures, `DecisionSpec`, explanations and full rule governance.&#x20;

plan2(20261122-045224).md



This is an implementation dependency problem.

How can Phase 1 prove that it invokes a pinned decision if the minimum immutable-version and decision-result contracts have not yet been settled?

You do not need the full Rulez management product in Phase 1. You do need its minimum trustworthy execution semantics.

**Move these foundations into Phase 0/1:**

| Must exist in Phase 0/1                                | Can remain in Phase 4                   |
| ------------------------------------------------------ | --------------------------------------- |
| Immutable rule identity and version                    | Full rule-management interface          |
| Rule and engine/language version pinning               | Advanced dependency-management UX       |
| Explicit evaluation inputs and outputs                 | Rich explanation and diagnostics UI     |
| Deterministic, bounded evaluation                      | Advanced composition features           |
| Explicit outcome semantics                             | Full approval and publication lifecycle |
| Independent golden fixtures for the slice              | Large-scale domain fixture libraries    |
| Pure decision execution, without external side effects | Advanced rule governance                |

At a minimum, define explicit distinctions between outcomes such as `MATCH`, `NO_MATCH`, `INDETERMINATE` and `ERROR`. Missing or unverified facts must not silently become false, and an unknown business answer must not be confused with a runtime failure.

Also define a stable decision-result contract. Rulez should return a decision and, where applicable, an effect intent. It should not execute external mutations itself.

Even though mutation execution is deferred to Phase 5, the boundary between a decision and an action should be designed in Phase 0. Otherwise, later action-gateway work may force a redesign of the decision contract.

## **2. Important weaknesses that need explicit treatment**

These are not reasons to abandon the architecture, but the revised plan should address them before reaching the corresponding implementation phases.

### **2.1 Security is moved forward, but the enforcement tests are not**

The plan correctly moves security contracts into Phase 0 and introduces a named security owner. It also correctly recognizes that even read-only miniapps need audit logging.&#x20;

plan2(20261122-045224).md



However, a security owner signing off on boundaries does not demonstrate that the runtime enforces them.

Read-only access can still expose confidential records, personal data or cross-tenant information. I would require the Phase 1 test suite to verify at least:

- The backend derives the authenticated principal and tenant context from a trusted source, not client-supplied identifiers.
- A user cannot access another tenant's records by changing an identifier or request parameter.
- Unauthorized fields and operations remain inaccessible.
- Every capability invocation is authorized server-side.
- The browser receives neither provider credentials nor unrestricted access to external APIs.
- The runtime rejects capabilities that are absent from the approved artifact manifest.

These tests should run even if the first implementation uses mock data. A mock can prove the authorization path's behavior; it cannot prove a real provider integration is correctly configured.

**Important:** Passing G4 should establish the security design. Passing the Phase 1 tests should establish that the initial runtime actually enforces it. Those are separate achievements.

### **2.2 The audit design needs data-minimization rules**

The plan now proposes both generation records and audit records containing information about the requirement, model, generated specification, validation, Rulez evaluations and capability invocations.&#x20;

plan2(20261122-045224).md



That is useful for debugging and accountability, but it creates another data store containing potentially sensitive information.

Do not equate comprehensive auditing with indiscriminately logging every input and output.

Specify:

- What is recorded for each invocation.
- Which values are redacted or excluded.
- Whether sensitive inputs are stored, referenced or represented by hashes.
- Retention and deletion requirements.
- Access permissions for audit and generation records.
- How an investigation can reconstruct a decision without exposing unrelated customer data.

A useful audit event will typically include actor and tenant references, artifact and dependency versions, capability or Rulez identity, authorization result, decision outcome, correlation ID and timestamp. Sensitive input snapshots should be handled separately under explicit retention and access rules.

Also, **audit logging is necessary but not sufficient for governance**. Governance additionally requires enforced permissions, approvals, version lifecycle, accountability and controlled publication.

### **2.3 Do not display arbitrary confidence scores**

The runtime architecture proposes displaying decisions, confidence and explanations.&#x20;

plan2(20261122-045224).md



The term *confidence* needs a precise definition.

A deterministic Rulez evaluation does not ordinarily have a statistical confidence score. An LLM's self-reported confidence is not automatically a calibrated probability that its interpretation or generated app is correct.

For the first versions, I would display:

- Decision outcome.
- Facts and evidence used.
- Rule or decision version.
- Missing, ambiguous or unverified inputs.
- Data freshness and source.
- Explanation or reason codes.

Only show a numerical confidence score when its meaning, measurement method and calibration have been established.

The same applies to semantic mapping. A field mapping should have evidence, provenance and a state such as `CONFIRMED`, `INFERRED`, `AMBIGUOUS` or `UNSUPPORTED`. Do not treat an LLM's own confidence estimate as sufficient proof that an `amount` field means gross pay rather than net pay.

### **2.4 The falsification protocol still lacks actual pass criteria**

The revised protocol is directionally much better. It removes the arbitrary two-prompt limit and the requirement to expose every schema field, and introduces independent evaluation and a more realistic set of test categories.&#x20;

plan2(20261122-045224).md



But it still does not tell the team precisely when a domain passes.

The plan says at least three of four domains must pass both axes. Yet it does not specify:

- How many test cases are needed per domain and complexity category.
- How many miniapp types must be generated and tested in each domain.
- What first-pass validity or task-completion result constitutes a pass.
- What level of human correction is acceptable.
- Which failures are automatic disqualifiers.
- How independent evaluators resolve disputed expected results.

This is not a request to restore an arbitrary headline such as 90%. It is a request to define a defensible benchmark before running it.

I would distinguish two classes of criteria:

**Hard invariants:** No unauthorized access, no unapproved capability execution, no cross-tenant disclosure, no fabricated statutory outcome, and no incorrect result in a required deterministic golden test.

**Quality measures:** First-pass specification validity, clarification frequency, repair iterations, human intervention, task completion and time to complete a task.

A domain should pass only when its required hard invariants hold and its measured quality meets predeclared criteria. Strong results in one domain must not conceal critical failures in another.

For CPF and IRS withholding, the domain packs should also state their exact scope, applicable dates, source register and limitations. Passing a deliberately simplified exercise is evidence about the engine's expressiveness, not evidence that it is a complete statutory compliance engine.

The customer-facing matrix is a useful idea, but publish it with the benchmark's scope, sample size, version and limitations. Do not turn small internal test results into broad reliability claims.

### **2.5 Phase 2 risks rebuilding the over-investment problem**

The plan says Phase 1 should prevent the team from building an impressive substrate without proving the product. But Phase 2 still requires all six miniapp types before the generator arrives in Phase 3.&#x20;

plan2(20261122-045224).md



That creates a smaller version of the same risk.

I would not implement six complete patterns merely because they appear in the initial taxonomy. Build the smallest useful pattern set demonstrated by the first end-to-end generated app. Add the remaining types when real requirements or benchmark cases justify them.

A pattern library is valuable when its patterns are validated and reused, not simply because it has a large number of entries.

### **2.6 Unsupported-capability tickets must not become an escalation path around approval**

The proposal to turn `UNSUPPORTED_CAPABILITY` into a routable work item is sensible. It converts a generator limitation into actionable development work.&#x20;

plan2(20261122-045224).md



But the generated ticket must remain a request for review, not an approval mechanism. The LLM must not create, enable or grant a missing capability simply because a generated app needs it.

The ticket should contain the requested operation, reason, required inputs and closest existing capability, with sensitive data removed. A separate authorized process must approve and provision it.

## **3. Recommended corrected phase sequence**

I would preserve most of the plan, but change the order and tighten the exit conditions.

| Phase | Corrected objective                          | What must be demonstrated                                                                                                                                                  |
| ----- | -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **0** | Architecture and security foundation         | Talang decision made from evidence; core contracts, identity model, authorization boundaries, audit/data-retention rules and threat model signed off                       |
| **1** | First generated vertical slice               | Requirement → generated `MiniappSpec` → validation → trusted runtime → pinned Rulez decision → approved/mock read capability → audit trail → representative task completed |
| **2** | Expand miniapp patterns                      | Add patterns justified by actual requirements; test their declarative contracts and runtime behavior                                                                       |
| **3** | Improve generator and semantic understanding | Test clarification, mapping, repair, authorization and task completion across a larger requirement set                                                                     |
| **4** | Full Rulez management                        | Advanced composition, temporal lifecycle, fixture library, explanations, approvals and management UX                                                                       |
| **5** | Real integration and controlled mutations    | Approved API integrations, server-side authorization, action gateway, idempotency and reconciliation, initially in a controlled environment                                |
| **6** | Falsification and hardening                  | Predefined domain/type/complexity benchmark; independent evaluation; security and correctness gates satisfied                                                              |
| **7** | Productization                               | Tenant isolation at scale, publishing, sharing, metering, operational support and deployment options                                                                       |

There is one important condition on this sequence: **Phase 5 must not mean that production writes are enabled before the write-specific security and adversarial tests pass.** The Phase 5 mutation work can happen in a controlled staging environment. Production release should have its own explicit authorization, security and operational gate.

## **4. What I would preserve**

The revision has made important improvements that should not be lost in another rewrite.

- The four-guarantee separation is correct: Rule validity, rule behavior, decision correctness and mutation safety are different responsibilities.&#x20;

  plan2(20261122-045224).md

- The declarative artifact and trusted runtime are a stronger default than generating bespoke HTML and JavaScript for every miniapp.&#x20;

  plan2(20261122-045224).md

- The distinction between capability discovery and capability approval is necessary.&#x20;

  plan2(20261122-045224).md

- The generation record is valuable because reproducible compilation does not make LLM generation reproducible.&#x20;

  plan2(20261122-045224).md

- The revised positioning no longer relies on unsupported claims that competitors cannot support governed writes. The differentiator is now the reusable, independently testable and versioned decision layer.&#x20;

  plan2(20261122-045224).md

- The revised moat is directionally sound. Integrations, semantic mappings, domain packs, usage and tested patterns can compound. But validation history is an advantage only when it is accurate, legally usable and connected to genuine customer adoption; it is not an automatic moat.

## **5. Final disposition**

Conditional go

My assessment is:

| Question                                                                 | Assessment                             |
| ------------------------------------------------------------------------ | -------------------------------------- |
| Is the product thesis structurally sound?                                | Yes                                    |
| Is Rulez a reasonable foundation for explicit business decisions?        | Yes, subject to Talang investigation   |
| Is the declarative miniapp architecture sound?                           | Yes                                    |
| Is the division between decisions and external actions sound?            | Yes, but the contracts need tightening |
| Does Phase 1 prove the product thesis as currently written?              | No                                     |
| Are the Rulez and decision contracts sequenced consistently?             | Not yet                                |
| Are the falsification criteria sufficient to support a generality claim? | Not yet                                |
| Should implementation proceed unchanged?                                 | No                                     |

**I would require four changes before approving Phase 1:** make the Talang choice evidence-dependent; include a minimal AI-generated app in the first vertical slice; resolve the generic decision-binding versus Rulez-binding contradiction; and bring minimum immutable decision/version semantics forward from Phase 4.

Then tighten the G6 benchmark and Phase 1 security acceptance tests before relying on their results.

The central architecture does not need another redesign. The danger is subtler: the team could implement a good interpreter, a well-governed runtime and six polished miniapp patterns, yet still fail to prove that a business user can describe a requirement and receive a useful miniapp.

**The first milestone should therefore be one generated miniapp that solves one real-shaped business task—not six miniapp types, not a complete rules-management platform, and not a sophisticated compiler.** Once that works, the rest of the architecture has something concrete to serve.