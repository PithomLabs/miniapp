# Pivot: Agent-Invokable RPA Miniapps on Go + DBOS + River + Fly + Neon

## 1. What the pivot actually changes

The previous plan treated miniapps as **human-facing surfaces** (dashboards, rule builders, calculators). This pivot makes them **agent-facing executables** — RPA-style business process automations that a coding agent (Codex, Claude Code) or a general harness agent (Hermes, OpenClaw) discovers, invokes, and hands work to.

| Dimension | Previous plan | This pivot |
|---|---|---|
| **Consumer** | Non-technical human in a browser | AI agent in a harness session (with human approval gates) |
| **Interface** | Generated UI → HTTP backend | MCP tools → miniapp runtime → durable backend |
| **Unit of value** | A rendered app | A **callable business process** with a stable contract |
| **Composition** | Miniapp types composed in a `MiniappSpec` | Agent composes **tool calls** across miniapps and native tools |
| **What the LLM authors** | `MiniappSpec` (UI + bindings) | Miniapp **manifest + MCP tool surface + durable steps** |
| **Differentiator** | Rulez-backed decisions in generated apps | **Durable, governed, idempotent process execution that agents can safely call** |

What survives unchanged: bounded declarative artifacts, fixture-gated validation, effects-as-intents, capability-gated security, versioned dependency manifests, and the refusal to let agents hold credentials or own durability.

The **precedent is already in production**: MiniMax Code's Mini App is a long-running app with a manifest, a frontend, a backend, and **hosted MCP** — the agent calls the miniapp's tools, and the miniapp hands work back to the agent. Credentials, ports, auth, and rollback publishing are owned by the host, not the miniapp. That is exactly the shape this pivot targets.

---

## 2. The core inversion

> **The agent is the caller. The miniapp is the executor. The agent never holds a credential and never holds durable state.**

Everything follows from this:

| Concern | Owner | Never the agent's job |
|---|---|---|
| Credentials / OAuth tokens | Miniapp's capability proxy | ❌ |
| Durable progress across restarts | DBOS workflow | ❌ |
| Idempotency of side effects | Miniapp step + provider key | ❌ |
| Authorization decision | Miniapp policy check | ❌ |
| Retry / compensation | DBOS or River | ❌ |
| Human approval gate | Miniapp `Recv` / MCP elicitation | ❌ |
| **Deciding what to call** | **Agent** | ✅ |
| **Sequencing tools across systems** | **Agent** | ✅ |
| **Interpreting messy input** | **Agent** | ✅ |

This is the discipline that makes agentic RPA safe. Classic RPA was brittle because the *script* held the brittleness. Agentic RPA is brittle if the *agent* holds the durability and credentials. The miniapp exists precisely to move both out of the agent and into a versioned, tested, auditable unit.

---

## 3. Taxonomy: RPA miniapp classes mapped to `rea.md`

`rea.md` already gave us the classification (Class A/B/C/D, Virtual Objects, Saga). The RPA pivot gives each class a concrete miniapp form and a concrete substrate.

| `rea.md` class | RPA miniapp form | Example | Substrate |
|---|---|---|---|
| **Class A — ordinary op** | Single-call tool: "create contact", "post invoice", "look up order" | Agent calls it once, gets a result | Plain Go handler + Postgres tx. **No queue, no DBOS.** |
| **Class B — simple background job** | Fire-and-forget tool: "send this campaign", "sync this record" | Agent calls, gets a job handle, moves on | **River** — transactional enqueue, unique jobs, retries, web UI |
| **Class C — multi-step bounded op** | Batch tool: "import 500 contacts, validate, report" | Agent calls, polls status, gets result | **River** with a job-step chain, **or** DBOS if steps need checkpoint-level resumption |
| **Class D — durable business workflow** | Process tool: "onboard this customer over 14 days", "approve → provision → notify with human gate" | Agent starts it, gets a workflow handle, may signal it later | **DBOS** — durable sleep, checkpointing, queues, `Recv` for human input |
| **Virtual Object** (per-entity state) | Stateful tool: "this customer's session", "this deal's pipeline state" | Agent calls methods on a keyed object | Postgres row + optimistic lock, **or** DBOS `SetEvent`/`GetEvent` for coordination |
| **Saga / compensation** | Reversible tool: "provision then rollback if billing fails" | Agent gets success or a compensated failure | DBOS steps + explicit compensation steps, **never implicit** |

**The decision rule from the ADR applies unchanged**: Class A/B/C are *rejected* for DBOS by default. Only Class D earns DBOS — and only after Gate 1–4【from writeup4】. The RPA pivot does not loosen this; if anything it tightens it, because agent-invoked tools are called far more often than human-initiated ones, and DBOS write amplification would scale linearly with agent traffic.

---

## 4. The substrate decision tree (concrete, per miniapp)

```
Miniapp declared
   │
   ├─ Does it survive the HTTP/MCP request? ──── no ──→ Class A: Go handler + Postgres tx
   │
   ├─ Is it one independently retryable operation? ─ yes ──→ Class B: River (InsertTx)
   │
   ├─ Multiple steps, bounded duration, no durable wait? ─ yes ──→ Class C: River chain
   │                                                              (or DBOS if any step
   │                                                               needs checkpoint resumption)
   │
   └─ Durable wait / human gate / multi-day / needs
      resume from exact step after crash? ─────── yes ──→ Class D: DBOS workflow
```

**Why River is the default for B/C:**

- It lives in the **same Postgres database** as application data, so `InsertTx` commits the job atomically with the business change.
- It is **Go-native**, has automatic retries, scheduling, unique jobs, and a web UI.
- It has **cross-language clients** (Go, Java, JS/TS, Ruby, Rust) — relevant if a miniapp's backend ever needs a non-Go sidecar.
- It does **not** carry DBOS's replay-determinism constraint, which is the single biggest source of accidental complexity in agent-generated code.

**Why DBOS is the default for D — and its honest costs:**

- Workflow state is checkpointed in Postgres; a crash resumes from the last completed step.
- DBOS explicitly argues for **co-locating workflow state with application data**, and can even enqueue a workflow from a Postgres `PL/pgSQL` function **in the same transaction** as the business data change — no outbox table required. That is the cleanest possible "miniapp reacts to a data change" trigger.
- **Costs the ADR already flagged and which still hold**: workflows must be deterministic, non-deterministic work must sit behind `RunAsStep`, uncontrolled goroutines are forbidden (use `dbos.RunWorkflow` / `dbos.Go` / `dbos.Select`), external side effects are **not** exactly-once across a crash-before-checkpoint window and need provider idempotency keys, and suspending sleeping workflows out of memory is an Enterprise capability.

**The River + DBOS coexistence rule** (from the ADR, restated for this stack): one authoritative execution owner per workload. River owns B/C. DBOS owns D. A miniapp **declares** which substrate it uses in its manifest; it does not dispatch the same operation to both.

---

## 5. The MCP surface: what the agent actually sees

The MCP tool is the **agent-facing contract**. It is not the implementation. The mapping is:

| MCP concept | Miniapp concept |
|---|---|
| `tool` | A capability: one business operation with a typed schema |
| `tool description` | The business promise: what it does, what it requires, what it returns |
| `input schema` | `CapabilitySpec.input_schema` |
| `output schema` | `CapabilitySpec.output_schema` — including a **handle** for async work |
| `tool annotations` | Read-only vs. destructive vs. idempotent vs. requires-approval |
| `elicitation` / `input-required` | Human approval gate for privileged writes |
| `task-augmented tool` | Long-running Class C/D invocation returning a task handle |

**The tool naming and shape discipline:**

```
rpa.<domain>.<verb>_<noun>          e.g. rpa.crm.create_contact
                                     rpa.billing.post_invoice
                                     rpa.hr.onboard_employee
```

Every async tool returns the same envelope, so agents learn one pattern:

```json
{
  "handle": "wf_01J...",
  "status": "running",
  "poll_tool": "rpa.jobs.status",
  "signal_tool": "rpa.jobs.approve",
  "expires_at": "2026-12-20T00:00:00Z"
}
```

This is the single most important design decision for agent usability: **agents are bad at long-running synchronous waits and good at handles.** A tool that blocks for 14 days will be abandoned by every harness. A tool that returns a handle and a poll/signal contract is composable.

`mcp-go` and the official `modelcontextprotocol/go-sdk` both support stdio, SSE, and Streamable HTTP transports, task-augmented tools, OAuth protected-resource metadata, and elicitation — so the Go implementation path is first-class, not a workaround. The guidance that matters: *"MCP tools are action surfaces, treat each handler with the same rigor as an internal API endpoint: validation, authorization, observability, and clear side-effect boundaries"*.

---

## 6. The miniapp anatomy

```
miniapp/
├── miniapp.json              # manifest: identity, version, MCP tool list,
│                             #   substrate per tool (A/B/C/D), capability deps,
│                             #   policy refs, fixture refs, dependency manifest
├── mcp/
│   └── server.go             # MCP tool registration; thin — validates,
│                             #   authorizes, delegates, returns handle
├── tools/
│   ├── classA_*.go           # direct handlers
│   ├── classB_*.go           # River JobArgs + Worker
│   ├── classC_*.go           # River chain or DBOS workflow
│   └── classD_*.go           # DBOS workflows + steps
├── capabilities/
│   └── *.go                  # typed SaaS/custom-API clients behind the proxy
├── fixtures/
│   └── *.json                # mandatory: input → expected result per tool
└── policy/
    └── policy.json           # risk classes, approval gates, tenant scopes
```

**`miniapp.json` is the artifact of record.** It declares:

```json
{
  "id": "rpa.billing.invoice_to_cash",
  "version": "1.4.2",
  "tools": [
    { "name": "rpa.billing.post_invoice", "substrate": "river",
      "risk": "constrained_write", "approval": "preview",
      "capabilities": ["erp.invoice.create"] },
    { "name": "rpa.billing.chase_invoice", "substrate": "dbos",
      "risk": "constrained_write", "approval": "policy",
      "capabilities": ["email.send", "erp.invoice.get"],
      "durable_wait": "P14D" }
  ],
  "dependencies": {
    "dbos_sdk": "x.y.z", "river": "v0.51.0",
    "capability_registry": "sha256:…", "policy": "v3",
    "fixtures": "sha256:…"
  }
}
```

**No tool can be registered unless its fixtures pass.** This is the same Zerodha `TestData()` contract applied to agent-invokable surfaces — and it matters more here, because an agent will call a tool hundreds of times without a human reading the output.

---

## 7. Security, auth, and tenancy

This is where the pivot is most dangerous, because an agent is now an authenticated caller with a tool surface.

| Control | Implementation | Why |
|---|---|---|
| **Agent identity ≠ user identity** | Every tool call carries a user-delegated token; the miniapp resolves `(agent_id, user_id, tenant_id)` and authorizes against the **user**, not the agent | An agent with broad scopes must not launder a narrow user's permissions into broad writes |
| **Token exchange, not token passthrough** | RFC 8693 token exchange at the gateway swaps the user's upstream token for a downscoped capability token stored in the vault | The agent never sees the SaaS credential |
| **Refresh rotation with family revocation** | OAuth refresh tokens rotate with reuse detection and absolute lifetime | Stolen refresh tokens are detectable and revocable |
| **Tenant isolation at the database** | Neon RLS with `auth.user_id()` extracting the JWT `sub` claim, `FORCE ROW LEVEL SECURITY` on every tenant table | Defense in depth: even a query bug cannot cross tenants |
| **Project-per-tenant when isolation matters** | Neon's own guidance recommends project-per-tenant where isolation or per-tenant restore matters | Compliance-heavy customers get hard isolation, not RLS alone |
| **Approval gates for destructive tools** | MCP tool annotations + `input-required` / elicitation; harnesses support pausing a run without holding a thread or pod | Human approves destructive actions without keeping the agent resident |
| **Per-harness tool policy** | Claude Code / Codex load MCP servers from `.mcp.json` only after workspace trust; `requireApproval` is per-tool per-agent | The harness, not the miniapp, controls what the agent may call |
| **Sandbox boundary** | OpenClaw / Hermes run under OpenShell policy sandboxes with files/network limits | The miniapp is called *from* a sandboxed agent, not run *inside* it |

**The rule that must be enforced in code, not policy:** a miniapp tool handler must fail closed if it cannot resolve a user identity. There is no "agent-only" mode for write tools.

---

## 8. Deployment topology: Fly + Neon, with the latency rule

The co-location rule is non-negotiable and empirically grounded: pairing Fly Machines in Johannesburg with Neon in Frankfurt added **~150 ms per database round-trip**. For a miniapp where every DBOS step writes a checkpoint, that compounds into minutes of added latency per workflow.

```
Region: same as Neon primary (iad, fra, sin, …)

Fly app: rpa-miniapps
├── process group: mcp       (stdio/HTTP MCP servers, scale-to-zero)
├── process group: api       (tool handlers, Class A)
├── process group: river     (River workers, Class B/C)
└── process group: dbos      (DBOS workers, Class D)
        │
        └── Neon Postgres (single primary, same region)
              ├── public schema   → app data + River job tables + RLS
              └── dbos schema     → DBOS system tables
```

- **One Fly app, four process groups** keeps private networking simple and lets `fly-replay` route MCP calls to the right worker.
- **DBOS and River share the same Neon database but different schemas.** This is what makes the transactional enqueue possible: a Class B job (River) and a Class D workflow (DBOS) can both be enqueued in the same transaction as the business row change.
- **Scale-to-zero on the `mcp` group** is acceptable only if the harness's tool-call timeout exceeds cold-start; otherwise keep a warm instance. Agent tool calls are latency-sensitive in a way human UI requests are not.
- **Neon branching** for fixture validation and miniapp regeneration testing — a branch per validation run, discarded after.

---

## 9. Phased plan

| Phase | Goal | Exit gate |
|---|---|---|
| **0** | **Contracts + threat model.** `miniapp.json` schema; MCP tool descriptor spec; `CapabilitySpec`; policy schema; agent-vs-user identity model; approval-gate model; substrate classification rules (A/B/C/D). Name a security owner. | G1: contracts reviewed; threat model signed off |
| **1** | **One Class A + one Class D miniapp, end to end.** A read tool and a durable multi-step tool, both callable from Claude Code and from a Hermes/OpenClaw sandbox. Fixtures mandatory. Audit logging from day one. | G2: agent invokes both tools; durable tool survives a mid-run `fly machine stop` and resumes; audit trail complete |
| **2** | **Capability proxy + OAuth token exchange + RLS.** One real SaaS API (pick one with OAuth2 and a sandbox tenant). Downscoped tokens, refresh rotation, per-tenant RLS, fail-closed authorization. | G3: a tool cannot be called without a user identity; cross-tenant probe fails |
| **3** | **River substrate for Class B/C.** Job args/workers, `InsertTx`, unique jobs, retry policy, dead-letter inspection. | G4: a Class C batch tool runs to completion, retries correctly, and is observable in River UI |
| **4** | **Approval gates + elicitation.** Destructive tools pause for human approval; approval works across Claude Code, Codex, and a general harness. | G5: destructive tool cannot execute without an approved elicitation; the pause survives a restart |
| **5** | **Miniapp generator.** Requirement + capability catalog → `miniapp.json` + tool stubs + fixtures, validated deterministically, repaired with feedback. | G6: two distinct miniapps generated in ≤2 attempts each, fixtures passing |
| **6** | **Falsification.** One SaaS-API miniapp, one custom-REST miniapp, one cross-system RPA flow (e.g. invoice → approval → post → notify). Measure agent-invocation success, recovery rate, approval friction. | G7: three domains pass; falsification findings documented with substrate-level responses |

**Phase 1 is the real gate.** If a durable Class D miniapp cannot survive a machine kill and resume correctly *when invoked by an agent through MCP*, the architecture is wrong and no later phase fixes it.

---

## 10. What to refuse, explicitly

| Refusal | Reason |
|---|---|
| Letting the agent hold a SaaS token | Destroys the credential boundary; makes revocation impossible |
| Making DBOS the default for every tool | Agent traffic scales linearly; DBOS write amplification would dominate cost and latency |
| Synchronous long-running tools | Harnesses abandon them; always return a handle |
| Fixtures as optional | An agent will call a tool hundreds of times without human review |
| Generated tool code | Tools are generated as *manifests + declarative steps*, compiled against a trusted runtime; never as free-form Go emitted by an LLM |
| Cross-region Fly/Neon pairing | ~150 ms per round-trip compounds across DBOS checkpoints |
| One MCP server per miniapp without a router | Agents degrade with tool sprawl; a single router with namespaced tools is the usable surface |
| Implicit compensation | Sagas must be declared per step; DBOS cannot undo committed effects on its own |

---

## 11. Honest risks

1. **The harness compatibility surface is wider than it looks.** Claude Code, Codex, OpenClaw, and Hermes differ in transport (stdio vs. HTTP), approval mechanism, and sandbox policy. Phase 1 must prove the same miniapp works across at least two harness families, or the "agent-invokable" claim is really "Claude-Code-invokable."

2. **DBOS's Go SDK is younger than its Python/TS SDKs**, and the best-practices guidance itself warns to pin and verify the SDK version before adapting examples. The `DBOSContext` vs `dbos.Context` naming drift in the public guidance is a concrete symptom.

3. **MCP approval semantics are genuinely underspecified.** Research in late 2026 documents the split between server-side annotation and host-side elicitation, and notes the failure mode where "neither side confirms" for destructive calls. The miniapp must not rely on the harness to enforce approval — it must enforce it server-side and treat host approval as an *additional* gate, not the gate.

4. **Agent tool sprawl is the new configuration graveyard.** The same organizational failure that turned BRMS projects into "big buckets o' rules" will turn a miniapp catalog into "big buckets o' tools" unless the manifest, the router, and the per-tenant tool allowlist are enforced from Phase 0.

5. **The generator is the weakest link, not the engine.** As the earlier adversarial review established, the risk is building an impressive substrate without proving that generated miniapps are actually used and trusted. Phase 6's falsification must be run on *real* processes with *real* agents, not synthetic demos.

---

## 12. The one-sentence reframe

> **We generate durable, governed, agent-invokable business-process miniapps — where the agent decides what to do, and the miniapp owns the credentials, the durability, the idempotency, and the audit trail.**

That is a materially different product from a rule builder, a dashboard generator, or a workflow tool. It is also the only framing in which the Go + DBOS + River + Fly + Neon stack is the *right* stack rather than an arbitrary one: River for the fire-and-forget work agents emit constantly, DBOS for the durable processes agents must not own, Neon for the transactional boundary that makes both correct, and Fly for the sandboxed, co-located execution surface.

When you are ready, the next step is the **Phase 0 contract specification prompt** — `miniapp.json` schema, MCP tool descriptor, `CapabilitySpec`, policy schema, identity model, and the A/B/C/D substrate classifier — because every later phase depends on those contracts being exact rather than descriptive.