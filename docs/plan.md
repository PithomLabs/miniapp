# The `rulez`-Backed Miniapp Generator: Corrected Framing

## The reframing, precisely

The previous plan mistook the rule builder for the product. It is not. The product is a **flexible miniapp generator** in the GigaCatalyst category — a system that takes a business requirement plus a client's data schema and produces a working, deployable miniapp. The rule builder is **one miniapp type among several**, and `rulez` is the **foundation** that makes generated miniapps correct rather than merely plausible.

The corrected hierarchy is:

| Layer | What it is | Role |
|---|---|---|
| **Miniapp generator** (product) | LLM-driven pipeline: business requirement + schema → `MiniappSpec` → compiled miniapp | What clients buy |
| **Miniapp type library** (patterns) | Rule builder, dashboard, calculator, approval queue, reconciliation view, workflow controller | What the generator composes from |
| **Primitive library** (shell) | Forms, tables, charts, condition builders, preview panels, fixture editors | What the miniapp types are built from |
| **`rulez` binding layer** (differentiator) | Where generated miniapps call deterministic rule evaluation instead of containing business logic in code | What makes us different from GigaCatalyst |
| **`rulez` engine** (foundation) | Bounded expression evaluator + `RuleSpec` + `DecisionSpec` + fixtures + policy | What makes business logic verifiable |

The generator is the product surface. The miniapp type library is the reusable shell. The `rulez` binding is the differentiator. The `rulez` engine is the foundation that makes the differentiator real.

---

## The one-sentence differentiator

> **GigaCatalyst generates apps whose business logic is code; we generate apps whose business logic is `RuleSpec` — deterministically validated, fixture-gated, versioned, and safe to mutate with.**

This is not a cosmetic difference. It changes what kinds of miniapps can be generated safely:

- **GigaCatalyst's approach**: the LLM writes TypeScript. Business logic — "is this customer eligible?", "what discount applies?", "which approver gets this?" — lives in the generated code. Correctness is a QA problem, and mutations are risky. The founder's own disclosure that ~80% of usage is read-only confirms this: even GigaCatalyst knows that generated *mutations* are not yet safe to encourage.

- **Our approach**: the LLM writes a `MiniappSpec` that *references* `RuleSpec` IDs for every business-logic decision. The frontend is still generated, but the logic is not — it is declarative, bounded, and validated before the miniapp can activate. Mutations are safe because the logic governing them is verifiable. **This is why we can generate miniapps with safe mutations, and GigaCatalyst mostly cannot.**

That is the wedge. Not "a better rule engine." Not "a better GigaCatalyst." A miniapp generator whose business logic layer is *correct by construction*, which makes safe mutation miniapps a product category that GigaCatalyst's architecture cannot match without rebuilding on a `rulez`-equivalent foundation.

---

## What "AI-native" means in this framing

AI-native is not "an LLM is in the loop." It means the generator's output is a **bounded, declarative spec** that is compiled deterministically, with quality measured per domain and per requirement type. Specifically:

1. **The LLM produces a `MiniappSpec`, not code.** The spec declares: which miniapp types compose this miniapp, which primitives render each section, which data bindings (capabilities) feed it, which `rulez` rule IDs govern its decisions, and which fixtures must pass before activation. It is data.

2. **The compiler produces the miniapp deterministically.** Given the same `MiniappSpec`, schema hash, pattern library version, and engine version, the compiler produces the same artifact. The artifact is content-addressable, regenerable, and diffable.

3. **The `rulez` binding is first-class in the spec.** Every section of the miniapp that performs a decision declares a `rulez_binding`: which rule ID, which inputs, which outputs. A miniapp without `rulez` bindings is a shell; a miniapp with `rulez` bindings is a governed app.

4. **Quality is measured on a matrix.** First-prompt success rate is tracked **per domain and per miniapp type**. A calculator in the payroll domain and a dashboard in the inventory domain have different success rates, different missing-pattern signatures, and different improvement paths. The matrix is the product's quality report card.

5. **The generator's boundaries are honest.** When a requirement cannot be expressed as a composition of known miniapp types, the generator says so — it produces a partial miniapp with explicit "requires code-step" markers, not a silently incomplete one. That honesty is the product's credibility.

---

## The miniapp taxonomy

The generator does not produce "an app." It produces one or more **miniapp types**, composed. The initial library should be small (six types), each hand-built and tested against at least two domains, and the library should grow only under the same discipline as the engine's DSL: demonstrated need, tested, versioned, capped.

| Miniapp type | What it produces | `rulez` binding | Typical requirement |
|---|---|---|---|
| **Rule builder** | A UI for authoring `RuleSpec`s: condition builder, effect configurator, fixture editor, version viewer | *Produces* `RuleSpec`, does not bind to one | "Let our ops team define discount rules" |
| **Calculator** | Input form → `rulez` evaluation → deterministic output; optional multi-step | Binds to one or more `RuleSpec` IDs | "Compute CPF contribution for this payslip" |
| **Dashboard** | Read-only aggregation views over capability data; optional drill-down | May bind `RuleSpec` for derived metrics | "Show pipeline health by stage" |
| **Approval queue** | List of items requiring decision; each row includes an `rulez`-evaluated recommendation and a one-click approve/reject | Binds to one or more `RuleSpec` IDs for recommendation logic | "Show me deals that need manager approval, with the recommended action" |
| **Reconciliation view** | Side-by-side comparison of two data sources; `rulez`-driven exception flagging | Binds to `RuleSpec` for exception predicates | "Which invoices don't match the PO?" |
| **Workflow controller** | Status board of in-flight processes; `rulez`-driven routing and escalation | Binds to `RuleSpec` for routing decisions | "Route tickets to the right queue, escalate stale ones" |

Three observations follow:

**First**, the rule builder is genuinely just one type. It is important because it is the self-hosting type — the generator can produce a miniapp that generates more rules — but it is not the centerpiece. The centerpiece is that *five of six types bind to `rulez` for their logic*, which is where the differentiator lives.

**Second**, the types compose. A "promotions management" miniapp might be a dashboard (campaign performance) + an approval queue (pending campaign approvals) + a rule builder (edit the discount rules). The generator chooses the composition from the requirement; it does not force a single type per requirement.

**Third**, the types are honest about what they don't do. A calculator cannot do multi-step workflows; that requires a workflow controller. A dashboard cannot mutate; that requires an approval queue or a rule builder. This is the same discipline as the DSL — the surface is bounded, and composition is how you get expressiveness without growing each surface.

---

## The architecture, corrected

```
┌──────────────────────────────────────────────────────────────┐
│  GENERATOR (product surface)                                  │
│  Business requirement + schema + sample data                 │
│      → LLM produces MiniappSpec                              │
│      → deterministic validation (fields, types, rulez bindings)│
│      → retry with feedback on failure (bounded to 2–3)       │
└──────────────────────────┬───────────────────────────────────┘
                           │ MiniappSpec + schema hash + library version
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  COMPILER (deterministic)                                    │
│  MiniappSpec → self-contained HTML+JS artifact               │
│  Content-addressed; regenerable; diffable                    │
└──────────────────────────┬───────────────────────────────────┘
                           │ artifact hash
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  RUNTIME (the generated miniapp)                             │
│  Renders UI shell (generic primitives, tailored layout)      │
│  Calls capabilities via proxy (data binding)                 │
│  Calls rulez via backend (business logic binding)            │
│  No business logic in generated JS                           │
└──────────────────────────┬───────────────────────────────────┘
                           │ /v1/evaluate, /v1/validate, /v1/register
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  FOUNDATION                                                  │
│  rulez engine + RuleSpec + DecisionSpec + PolicySpec +       │
│  fixtures + capability registry + audit                      │
└──────────────────────────────────────────────────────────────┘
```

The invariant: **the generated artifact contains no business logic.** It contains UI, data bindings, and `rulez` bindings. Business logic lives in `RuleSpec`, which is versioned, fixture-gated, and deterministically evaluated. This is what makes the generated miniapp both *usable* (a working UI) and *trustworthy* (a verifiable logic layer underneath).

---

## What the falsification protocol tests

The previous plan tested one axis: does `rulez` express domain X? This plan tests **two axes**, because the product has two claims.

### Axis 1: Engine genericity (unchanged from previous plan)

For each of four domains — promotions, Singapore CPF, simplified IRS withholding, inventory reorder — the `rulez` engine must express the domain's decision logic as `RuleSpec` packs with fixtures. Falsification findings are logged where the DSL cannot express an edge case, and each finding carries a documented response (capability, composed rule, or code-step).

### Axis 2: Generator genericity (new)

For each of the four domains, the generator must produce a miniapp of the domain's *natural type* from a business requirement in ≤2 prompts:

| Domain | Natural miniapp type | Requirement to generate from |
|---|---|---|
| Promotions | Rule builder | "Let our marketing team define coupon eligibility and discount rules" |
| Singapore CPF | Calculator | "Compute CPF contribution for a payslip" |
| IRS withholding | Calculator | "Compute federal withholding for this paycheck" |
| Inventory reorder | Dashboard + approval queue | "Show me items that need reordering, with the recommended order quantity" |

The miniapp must:
- Reach every field in the schema (no silent omissions)
- Bind correctly to the domain's `RuleSpec` IDs
- Pass the domain's fixtures
- Be usable by a non-technical user without reading documentation (measured: can they complete a representative task in 5 minutes?)

Falsification findings on Axis 2 are logged when:
- The generator takes 3+ prompts to succeed
- A required primitive or miniapp type is missing from the library
- The schema is ambiguous and the confidence report flags it
- The generated miniapp has an unreachable field or a mis-bound rule

**The matrix is the product's quality report.** A domain that passes Axis 1 but fails Axis 2 is a signal that the miniapp type library needs a pattern. A domain that passes Axis 2 but fails Axis 1 is a signal that the engine needs a capability. Both are actionable, and both are visible.

---

## Revised phased plan

Phases 0–3 are unchanged from the previous plan and cover the foundation. The new material starts at Phase 4.

### Phase 0 — `rulez` fork and `RuleSpec` envelope (0.5 day)

Fork Talang into `github.com/PithomLabs/rulez`. Modernize dependencies. Add `RuleSpec` struct, canonical serialization, structural validator. No grammar changes. **Exit:** hand-written promotion rule validates and evaluates.

### Phase 1 — Versioning, composition, fixtures (1.5 days)

`RuleRegistry` with `as_of`, `composes[]` DAG, `validate`/`test` CLI, fixture gate. **Exit:** composed rule with two effective windows evaluates correctly; failing fixture blocks registration.

### Phase 2 — Effects, `DecisionSpec`, conflict resolution (2–3 days)

Effects as intents. `DecisionSpec` decomposes decisions. Priority/salience, `AmbiguousMatchError`. **Exit:** overlapping promotions produce a single deterministic decision; ambiguous rule set raises a clear error.

### Phase 3 — Go backend (2–3 days)

`POST /v1/evaluate`, `/v1/validate`, `/v1/register`, `GET /v1/rules/{id}/versions`. Stateless, version-aware. **Exit:** `curl` works.

### Phase 4 — Miniapp primitive and type library (4–5 days) — **the new core**

Build the reusable shell that every generated miniapp composes from:

- **Primitives** (UI components): scalar input, enum select, date/time picker, list repeater, nested group, condition builder, effect configurator, fixture editor, data table, chart, side-by-side diff, status badge, action button, approval row, preview panel.
- **Miniapp types** (compositions): rule builder, calculator, dashboard, approval queue, reconciliation view, workflow controller — the six types above.
- **`rulez` binding primitives**: an input-to-rule evaluator block, a recommendation display block, a confidence/explanation block. These are the primitives that make a miniapp *governed* rather than merely interactive.
- **Schema adapters**: flat object, nested object, list-of-objects, list-of-scalars, mixed. Each adapter knows how to render the appropriate primitives and how to bind `rulez` inputs.

Each primitive and type is hand-built, tested against at least two domains, versioned. This is the reusable part — the part a competitor wrapping an LLM cannot replicate.

**Exit:** a hand-written `MiniappSpec` for a promotions rule builder compiles into a working miniapp. The library covers the four falsification domains' natural types.

### Phase 5 — Miniapp compiler and generator (4–5 days)

Build the two pipeline components:

- **Compiler**: `MiniappSpec` → self-contained HTML+JS artifact, deterministic, content-addressed.
- **Generator**: schema + sample data + requirement → LLM → `MiniappSpec`, validated deterministically (fields exist, types exist, `rulez` bindings point to registered rules), retried with feedback on failure.

**Exit:** for the promotions domain, the generator produces a working rule-builder miniapp from a prompt in ≤2 attempts; the miniapp passes fixtures and is usable by a non-technical user in under 5 minutes.

### Phase 6 — Falsification across domain × type matrix (5–6 days)

Run the two-axis falsification protocol above. Produce the matrix. Log every finding. For each missing primitive, miniapp type, schema adapter, or `rulez` capability, decide: add it (tested, versioned) or document the boundary (honest fallback).

**Exit:** ≥3 of 4 domains generate in ≤2 prompts with zero corrections; the fourth's failures map to a specific library gap that is added and retested. The engine expresses the majority of each domain's logic. The matrix is documented.

### Phase 7 — Productization (post-MVP)

Multi-tenant isolation. Regeneration pipeline (miniapps are regenerable as the library evolves; diff is the review surface). `PolicySpec` enforcement at the miniapp level. Pattern library governance process (cap, tested additions only). Billing/metering.

---

## What the moat actually is

The moat is not the LLM, not the engine alone, not the UI. It is the **combination of four assets**, none of which alone is defensible but together are expensive to replicate:

| Asset | Why it is hard to copy |
|---|---|
| **`rulez` engine + `RuleSpec` + `DecisionSpec` + fixtures** | A bounded, LLM-safe rule substrate with 5+ years of Talang's hardening and a new governance envelope. Copying it means rebuilding the safety story. |
| **Miniapp type and primitive library** | Hand-built, hand-tested shell compositions. A competitor with only an LLM produces inconsistent output; the library is what makes output consistent. |
| **The generator + compiler pipeline** | Deterministic compilation from bounded specs. The LLM is a component; the pipeline is the product. |
| **The falsification-driven governance process** | Per-domain, per-type quality measurement. This is operational, not technical — it cannot be copied from a marketing page. It compounds over time as findings become patterns become library additions. |

The GigaCatalyst comparison is instructive. GigaCatalyst's moat is the proxy + validation pipeline + sandbox. Ours is the `rulez` binding + miniapp type library + generator pipeline. Both are "infrastructure around an LLM." The difference is that our infrastructure makes **safe mutations** possible, and theirs keeps mutations rare because their generated logic is unverifiable. That is a category difference, and it is the basis of the pitch.

---

## Honesty check: what this plan does not solve

The `rulez` foundation solves **logic quality**. It does not solve:

- **UI quality.** A generated miniapp can still be ugly, mis-laid-out, or hard to use. The primitive library helps, but visual quality is the same hard problem GigaCatalyst has, and the same problem Retool has, and the same problem every low-code platform has. We are not claiming to solve it.
- **API binding quality.** The generated miniapp calls capabilities through the proxy; if a capability is missing or the schema is ambiguous, the miniapp may bind to the wrong data. The schema confidence report helps, but does not eliminate this.
- **The long tail of miniapp types.** Six types cover a lot. They will not cover everything. A client whose requirement is "a miniapp that plays a video and asks three questions" is not served by this library, and pretending otherwise is dishonest. The fallback is a partial miniapp with explicit gaps, not a hallucinated miniapp.
- **The engine's domain generality beyond the falsification set.** Four domains is not "all domains." The falsification protocol is honest about its boundary, and the boundary is real.

What the `rulez` foundation *does* solve, and solve uniquely, is: **generated miniapps whose business logic is correct by construction, safe to mutate with, and auditable**. That is a real, defensible, and — critically — *sellable* difference from GigaCatalyst, and it is the difference this plan is built to prove.

---

## Next step

When you are ready, ask for the **Phase 0 implementation prompt**. It will scope a single working session: fork procedure, dependency modernization, `RuleSpec` struct, canonical serialization contract, structural validator, and the acceptance test. Phases 4 and 5 (the miniapp library and the generator pipeline) will each get dedicated prompts, because they are the parts of this plan that differentiate the product from the engine alone, and they deserve careful, focused implementation rather than being assumed to fall out of the foundation.