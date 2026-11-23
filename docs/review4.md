Understood. **Miniapp's next architectural priority is now unequivocally Security and Observability as co-equal, critical foundational capabilities.** Prompt28 has reached its implementation closeout, so we should transition from Stage 3A execution work to independent Gate 2 review and then the two foundational contracts.

I reviewed both uploaded documents. Here is my architectural assessment.

## **1. Prompt28: implementation closeout accepted for Gate 2 review**

The closeout report provides substantial evidence across all six workstreams:

- Authorization matrix: 12 new legs.
- Concurrency: nine race cases and the six-boundary crash matrix.
- C6: Class A recovery integration now execution-proven.
- Revocation: health, grant hashing, CLI verification, and revocation checks.
- Settle: crash-safety work and a genuine red/green fix for the backfill audit race.
- Provider bypass: AST-pinned send-site inventory and per-path tests.

The report records the full suite, race tests, `go vet`, formatting checks, and `scripts/checks.sh` as passing against disposable PostgreSQL 19.&#x20;

prompt28-stage3a-closeout.md

prompt28-stage3a-closeout.md

prompt28-stage3a-closeout.md

prompt28-stage3a-closeout.md



There is one important distinction:

Ready for Gate 2

This is the implementation report's readiness verdict, **not an independent Gate 2 clearance yet**. The independent review remains necessary.

## **2. Settle operator authority: preserve the stop**

I agree with the decision to stop rather than weaken the security boundary.

The decision memo establishes that:

- OSI-2 does not yet define an approved role vocabulary.
- `tenant-ops` is synthetic and cannot be silently promoted into a production role.
- The API-key mechanism is demonstration scaffolding, not a production operator authority.
- No implemented control-plane mechanism supplies an alternative.&#x20;

  prompt28-settle-operator-decision.md



The three owner decisions are:

| Option | Decision                                                                           | Assessment                                    |
| ------ | ---------------------------------------------------------------------------------- | --------------------------------------------- |
| A      | Approve an operator role, membership/assignment rules, and server-side enforcement | Preferred production direction                |
| B      | Introduce a distinct operator authority kind with explicit issuance rules          | Viable alternative                            |
| C      | Allow tenant-authenticated Settle with audit-only attribution                      | Weaker boundary; requires explicit acceptance |

These are not interchangeable implementation details. The decision changes who has authority to resolve ambiguous external effects.&#x20;

prompt28-settle-operator-decision.md



My recommendation is **Option A**, provided we explicitly define who can assign the role, what tenant scope it applies to, and how it is revoked. Do not simply adopt `tenant-ops` without that decision.

Until then, `effects.Settle` remains unwired as intended.

## **3. What Gate 2 must independently verify**

I recommend an adversarial review that concentrates on:

1. Whether the claimed concurrency and crash guarantees are demonstrated by reproducible tests, rather than merely described in reports.
2. Whether the six crash boundaries actually exercise genuine process termination and verify persisted state after restart.
3. Whether refusal paths consistently establish zero provider sends, correct audit attribution, and unchanged protected state.
4. Whether attempt fencing, authorization lock order, C6 recovery, and provider bypass protection preserve the original P21 invariants.
5. Whether the PostgreSQL 19 digest is consistently pinned and whether the unexecuted CI workflow is accurately classified.
6. Whether all residuals and owner-blocked items are correctly carried forward.

One constraint is especially important: Gate 2 must not claim that Classes A–D are all conformant. The report explicitly says Class B/C adapter conformance remains a future workstream.&#x20;

prompt28-stage3a-closeout.md

prompt28-stage3a-closeout.md



## **4. The next foundational architecture work**

After Gate 2, I recommend two coordinated workstreams.

**Security Contract**

Critical priority

Establish the common identity, authentication, authorization, delegation, tenant isolation, capabilities, credential handling, and effect-authorization model.

It must explicitly cover human users, AI agents, services, API, MCP, CLI, and background workers.

**Observability Contract**

Critical priority

Establish audit, structured logs, metrics, distributed traces, correlation identifiers, redaction, cardinality, export, operational diagnostics, and SLO requirements.

OpenTelemetry should provide the common instrumentation and export foundation.

These contracts must be designed together because identity and authority are essential diagnostic context. We need to know not just what happened, but which actor initiated it, on whose behalf, under which authority, and through which execution path.

Then:

1. Implement the common foundations against existing Class A and D paths.
2. Accelerate Class B and C with both contracts integrated from their first implementation.
3. Require cross-class security and observability conformance before declaring A–D production-ready.

This preserves our earlier decision that execution reliability belongs to the framework, while business logic belongs to miniapps.

## **5. Revised Miniapp priority order**

| Priority | Work                                                         | Status                            |
| -------- | ------------------------------------------------------------ | --------------------------------- |
| 1        | Prompt28 independent Gate 2 review                           | Next                              |
| 2        | Settle operator authority decision                           | Owner action                      |
| 3        | Security Contract                                            | Critical foundational             |
| 4        | Observability Contract                                       | Critical foundational             |
| 5        | Shared implementation and conformance for existing A/D paths | After contracts                   |
| 6        | Accelerated Class B and C implementation                     | Built against both contracts      |
| 7        | Cross-class A–D conformance                                  | Required for production readiness |

**No Phase 3 work is authorized by this update.** The immediate next step is Gate 2 review, not more implementation.

The governing architectural principle going forward is:

> **Miniapp must never expose a business capability without an explicit security boundary, and it must never consider an execution capability production-ready without attributable, observable execution.**

I will use this priority order as the basis for future Miniapp architecture decisions, implementation plans, and adversarial reviews.