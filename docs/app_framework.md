## prompt

so we have the categories for miniapp, how exactly do we use this framework if I am going to build a Web-based API server or user-facing Web app server, write in markdown only

## chatgpt

The most important distinction is this:

**The Miniapp Framework is your business execution engine. A Web API server or user-facing Web application is one of the ways people and agents access that engine.**

You do not have to choose between building a conventional Web application and using the Miniapp Framework. You combine them.

In fact, I would design the framework so that you can build ordinary Go Web applications on top of it without forcing every HTTP endpoint, page, or database query to become a miniapp.

## **1. Think of the architecture as three layers**

## **Presentation and access layer**

**Web UI**

Browser

**REST API**

HTTP/OpenAPI

**AI Agents**

MCP/CLI

## **Miniapp Framework**

One canonical invocation and authorization path

**Registry**

Discovery and routing

**Authorization**

Identity and policy

**Execution**

A / B / C / D

**Effects and audit**

Reliability and accountability

## **Infrastructure and business systems**

**PostgreSQL**

Business state

**River / DBOS**

Execution

**SaaS APIs**

External systems

The Web application handles presentation, sessions, navigation, and HTTP concerns. The Miniapp Framework handles business operations, their execution semantics, permissions, and recovery.

Both can run in the same Go application initially.

## **2. What happens when you build a Web application?**

Suppose you build a Web-based CRM.

A user logs in, views contacts, updates a lead, imports contacts, and initiates a follow-up campaign.

These are not all the same kind of operation.

| User action                           | Miniapp category | Execution                                                               |
| ------------------------------------- | ---------------- | ----------------------------------------------------------------------- |
| View contact details                  | A                | Immediate response                                                      |
| Update contact information            | A                | Immediate transaction, with effect controls if externally consequential |
| Send one notification                 | B                | Background job through River                                            |
| Import 5,000 contacts                 | C                | Bounded batch with progress and recovery                                |
| Follow up with a lead over seven days | D                | Durable workflow through DBOS                                           |

Notice something important: **the CRM Web application itself is not Class A, B, C, or D.**

It is a Web application that uses different categories for different business operations.

That is how I would expect every future application built on the framework to work.

## **3. How the Web server actually invokes a miniapp**

Consider a user clicking the "Follow up with this lead" button.

The browser sends:

```
POST /api/leads/123/follow-up
Content-Type: application/json
```

The Web server:

1. Validates the HTTP request.
2. Authenticates the browser session.
3. Establishes the tenant and authority context.
4. Converts the request into a canonical miniapp invocation.
5. Calls the framework's invocation service.

Conceptually:

```
Browser
   |
   v
Web route
   |
   v
Session authentication
   |
   v
Authorization and tenant context
   |
   v
Canonical invocation service
   |
   v
Registry
   |
   v
lead-followup (Class D)
   |
   v
DBOS
   |
   v
Neon + CRM capability
```

The invocation service returns an asynchronous run handle.

The Web UI displays:

```
Follow-up sequence started.

Status: Running
Run ID: ...
Next action: Awaiting scheduled follow-up
```

The browser can then request:

```
GET /api/runs/{run_id}
```

Or receive a notification when the workflow changes state.

The crucial point is that the Web server does not implement its own independent DBOS orchestration, authorization, or audit logic.

It uses the same framework execution path as MCP and REST callers.

## **4. How should you organize the Go Web application?**

I recommend starting with a single Go application that hosts both the Web interface and the framework runtime.

An illustrative structure:

```
cmd/
  server/
    main.go

internal/
  web/
    routes.go
    middleware.go
    sessions.go
    handlers/
      dashboard.go
      contacts.go
      leads.go

  views/
  assets/

contract/
registry/
runtime/
executor/
auth/
effects/
approvals/
audit/
capability/
transport/
  service.go
  openapi/
  mcp/
  cli/

miniapps/
  crm-read/
  crm-write/
  notify-send/
  contact-import/
  lead-followup/
```

This is an application structure, not a proposal for additional independent frameworks.

### **What belongs where?**

| Component         | Responsibility                                           |
| ----------------- | -------------------------------------------------------- |
| `internal/web`    | HTTP routes, browser sessions, request/response handling |
| `internal/views`  | HTML rendering and templates                             |
| `internal/assets` | CSS, JavaScript, images                                  |
| `miniapps/`       | Business operations and their contracts                  |
| `runtime/`        | Invocation lifecycle and execution                       |
| `auth/`           | Identity, authorization, and policy                      |
| `effects/`        | External side effects and reconciliation                 |
| `executor/`       | Class A, River, and DBOS execution                       |

The Web layer may have its own query and presentation-composition code. Not every dashboard query or page-rendering operation needs to become a miniapp.

But when an operation represents a meaningful business capability, it should use the framework rather than duplicating business logic in a Web handler.

## **5. A particularly important distinction: Web routes are not miniapps**

Consider these routes:

```
GET  /login
POST /logout
GET  /dashboard
GET  /contacts/123
POST /contacts/123/update
POST /contacts/import
POST /leads/123/follow-up
GET  /runs/abc
GET  /health
```

They serve different purposes.

| Route                  | Responsibility                 |
| ---------------------- | ------------------------------ |
| `/login`               | Web authentication             |
| `/logout`              | Session termination            |
| `/dashboard`           | Page composition               |
| `/contacts/123`        | Read presentation data         |
| `/contacts/123/update` | Invoke a business operation    |
| `/contacts/import`     | Invoke a batch operation       |
| `/leads/123/follow-up` | Invoke a durable workflow      |
| `/runs/abc`            | Observe asynchronous execution |
| `/health`              | Operational health             |

Do not create a miniapp for every route.

Likewise, do not implement consequential business operations directly inside Web handlers.

A Web route should be a thin adapter.

## **6. Authentication: how does a human user fit the two-principal model?**

This deserves particular attention.

Your framework distinguishes:

- **Agent principal:** who initiates an invocation.
- **Authority principal:** whose permission or delegated authority permits the operation.

In an agent-driven operation, these are naturally separate.

In a conventional Web application, the authenticated human user supplies the authority context, while the Web application acts as the calling client.

The framework must establish that relationship explicitly.

For example:

```
Authenticated human:
  user-123

Tenant:
  tenant-456

Calling application:
  crm-web

Authorized operation:
  lead.followup.start
```

The Web server derives this context from its verified session and server-side identity mapping.

Never trust a browser-supplied `tenant_id`, `role`, `authority_id`, or approval flag.

The browser requests the business operation. The server determines who may perform it.

This gives you an important architectural advantage: an AI agent and a human using the Web interface can invoke the same business operation without duplicating its rules.

## **7. What if you build only a Web API server?**

That is even simpler.

You do not need the presentation layer.

Expose the canonical invocation service through versioned REST/OpenAPI endpoints.

For example:

```
/api/v1/tools
/api/v1/tools/{miniapp}/{tool}/invoke
/api/v1/runs/{run_id}
/api/v1/runs/{run_id}/signals
/api/v1/runs/{run_id}/cancel
```

Class A operations return their results inline.

Class B, C, and D operations normally return a run handle, and the caller retrieves progress or signals/cancels the operation as authorized.

The API server does not care whether the caller is a browser, another application, an integration, or an agent harness. Authentication and authority resolution do matter, but the underlying business execution is shared.

## **8. When should the Web application bypass the miniapp framework?**

There are legitimate cases.

Do not force everything through the miniapp abstraction.

Examples that usually remain ordinary Web application functionality:

- Rendering HTML.
- Handling CSS and static assets.
- Browser navigation.
- Managing short-lived Web sessions.
- CSRF protection.
- Formatting presentation data.
- UI-specific validation.
- Health checks.
- Serving documentation.
- Composing multiple read models for a dashboard.

Similarly, a private helper function that calculates a tax amount is not automatically a miniapp.

It may simply be a reusable domain service used by a miniapp.

The distinction is between a meaningful, independently callable business operation and an internal implementation detail.

**Do not turn every Go function, HTTP handler, database query, or UI interaction into a miniapp.**

## **9. The deployment model I recommend**

Initially, keep the deployment simple.

## **One Web gateway, multiple execution workers**

**Fly Machine: gateway**

Go Web application + REST/OpenAPI + MCP + canonical invocation service + Class A

**Fly Machine: River**

Class B and approved Class C jobs

**Fly Machine: DBOS family workers**

Class D workflows, with explicit AppName/version isolation

**Neon PostgreSQL**

Application state, execution state, effects and audit records

You do not need one Fly application per miniapp.

You do not need a separate Web server for every business category.

You can have one Web application with multiple process groups.

If the product grows, you can separate the Web frontend, gateway, and worker deployments where isolation or scaling genuinely requires it.

## **10. The most useful mental model**

I would teach your developers to remember these four distinctions:

| Question                                   | Architectural answer                           |
| ------------------------------------------ | ---------------------------------------------- |
| How does someone access the system?        | Web UI, REST, MCP, or CLI                      |
| What business outcome are they requesting? | A miniapp operation                            |
| How must that operation execute?           | Class A, B, C, or D                            |
| Where should it run?                       | Machine, Sprite, or another approved placement |

These are independent design decisions.

### **A concrete example**

Imagine you eventually build an employee management SaaS platform.

It has a user-facing Web application, a public API, and AI-agent integration.

All three access the same business operations:

- View employee: Class A.
- Update employee profile: Class A.
- Send welcome email: Class B.
- Import employee records: Class C.
- Onboard a new employee across HR, identity, payroll, and access-control systems: Class D.

The Web interface does not need to know how DBOS works.

The AI agent does not need to know how River works.

Neither needs to know how a provider's OAuth refresh mechanism works.

They request business outcomes. The framework handles execution.

**That is the real value of this architecture: you are not merely building a Go framework for miniapps. You are creating a reusable business execution layer that can power traditional Web applications, API products, and AI-agent interfaces simultaneously.**

My recommendation is to make this the standard application integration pattern: build ordinary Web applications as thin presentation and access layers over the Miniapp Framework, with the framework remaining the single owner of consequential business execution.


## deepseek

# Using the Miniapp Framework in a Web API Server or User-Facing Web App

**Audience:** Developers building a web backend or web app on top of the miniapp framework
**Question answered:** Given the framework and the A/B/C/D miniapp taxonomy, how do I actually structure a web application?
**Companion to:** "App Category Classification: A Technical Guide for Business Logic Organization"

---

## 1. What "using the framework" actually means

The framework is **not a web framework**. It does not replace `net/http`, `chi`, `gin`, `echo`, `fiber`, or any UI framework. It does not own routing, templates, static assets, sessions, CORS, or CSRF.

The framework is a **business-logic execution substrate**. It owns:

- the miniapp contract (manifest, tool descriptors, fixtures)
- the invocation lifecycle (auth, dispatch, run handles, audit)
- the execution substrates (plain Go, River, DBOS)
- capabilities (typed external API calls through a credential-isolating proxy)
- effects (intent → attempt → confirmed/failed/unknown → reconcile)
- approvals (durable gates for consequential actions)
- observability (structured audit, traces, run history)

In a web app, the framework sits **below** the HTTP layer and **above** the database. The HTTP layer is a transport adapter over the invocation service — exactly as MCP and CLI are. This is the "thin adapter" principle from ADR-004, applied to web transport.

Put differently: **your web handlers stop being where business logic lives. They become where HTTP lives.**

---

## 2. When the framework applies — and when it doesn't

The framework's own ADR-001 discipline applies to it: it must earn its complexity. A web app should adopt the framework when at least one of the following is true:

| Condition | Why the framework pays off |
|---|---|
| External integrations with non-trivial failure semantics | Capability proxy + effect ledger + reconciler prevent silent data loss |
| Multi-step business processes that must survive restarts | Class C/D substrates give checkpoint recovery without hand-rolling it |
| Effect tracking / reconciliation needs (UNKNOWN outcomes) | The `PENDING → ATTEMPTING → CONFIRMED \| FAILED \| UNKNOWN` protocol is the framework's core competency |
| Approval workflows with separation of duties | Durable approval table + out-of-band channel are first-class |
| Agent exposure (a customer's AI assistant calling your app's business operations) | MCP surface is free once miniapps exist |
| Multi-tenant isolation with audit requirements | Two-principal model + RLS context + audit trail are structural |

**A pure CRUD app with no external side effects, no multi-step processes, and no agent exposure does not need the framework.** Plain `net/http` + Postgres + River is simpler. The framework is not for every app; it is for apps where business operations have non-trivial execution semantics.

The applicability test in practice:

> **Count how many of your web endpoints have UNKNOWN-outcome risk.** If zero or one — plain Go. If two or more — the framework. UNKNOWN-outcome risk means: an external API call that could succeed on the provider side but fail to acknowledge to your server, and you have no way to find out from the provider what happened.

---

## 3. Three shapes

The framework supports three shapes of web application. The differences are in the presentation layer, not the backend.

### Shape 1 — API server (no UI)

Your app exposes REST endpoints consumed by third-party clients, mobile apps, or frontends you don't build. Common for B2B SaaS.

```
Client → HTTP → your API server → invocation service → miniapp → substrate
                                                             ↘ capability proxy → SaaS
```

**Presentation:** JSON REST (canonical), OpenAPI spec (generated from Go types). Optionally MCP for agents.
**Framework usage:** comprehensive — every business operation is a miniapp tool.
**User-facing nuance:** none (no users in the UI sense; the "user" is the API consumer, authenticated by API key or OAuth).

### Shape 2 — User-facing web app (full stack)

Your app has its own frontend (server-rendered HTML, SPA, or htmx+SSR). Users log in, interact with forms, click buttons.

```
User → Browser → your web app HTTP layer → invocation service → miniapp → substrate
                                                                    ↘ capability proxy → SaaS
```

**Presentation:** HTML templates or a JS frontend, session cookies, CSRF.
**Framework usage:** comprehensive for business operations, selective for presentation concerns (rendering, form validation, session management stay in the web layer).
**User-facing nuance:** the user IS the principal — no two-principal model needed. The web app's login flow establishes the user identity, and the web app presents it to the invocation service.

### Shape 3 — Hybrid (web app + agent parity)

Your app has a user-facing frontend **and** exposes the same miniapps to agents via MCP. A customer can use your UI, or their AI assistant can invoke the same business operations.

```
User → Browser → HTTP layer → invocation service → miniapp
Agent → MCP    → MCP router → invocation service → miniapp
                                  ↓
                              one audit trail
```

**Presentation:** web UI + MCP surface.
**Framework usage:** comprehensive for both surfaces.
**User-facing nuance:** the two-principal model applies — user invocations authorize against the user; agent invocations authorize against the delegated user, with the agent's scope intersected. Same miniapp, two entry paths, one audit trail.

**The recommendation for a serious product is Shape 3.** The incremental cost of exposing MCP once miniapps exist is small. The strategic value — your app's business operations become agent-invokable by default — is large.

---

## 4. The three layers

Every web app on the framework has three layers. Confusing them is the most common source of technical debt.

### Layer 1 — Web transport (presentation)

Owns: HTTP routing, request parsing, response rendering, session cookies, CSRF, CORS, static assets, templates, presentation-side form validation.

Does not own: business logic, external calls, database writes to business tables, effect tracking, approval decisions.

**Code shape:** thin handlers. Each handler's job is: parse → authorize at the session level → call invocation service → map result to HTTP.

**Rough line budget:** most handlers should be under 40 lines. If a handler is over 100 lines, business logic has leaked in.

### Layer 2 — Miniapps (business logic)

Owns: the business operations themselves, expressed as tools with declared classes (A/B/C/D). Capability bindings. Effect intents. Fixtures. Approval gates.

Does not own: HTTP parsing, HTML rendering, session state.

**Code shape:** Go packages under `miniapps/<domain>/`, each implementing the framework's contract, each tested against fixtures.

### Layer 3 — Framework infrastructure (cross-cutting)

Owns: invocation lifecycle, auth enforcement, run handles, audit, effects, approvals, capability proxy, token broker, substrate dispatch.

Does not own: your business.

**Code shape:** the framework packages, imported by both web layer and miniapps. Configured once in `main.go`. Never modified per-app (it's a library, not a fork).

---

## 5. Code organization

```
myapp/
├── cmd/
│   └── server/
│       └── main.go                    # wires web, invocation, substrates
│
├── web/                               # Layer 1: presentation
│   ├── router.go                      # chi/net-http routes
│   ├── handlers/                      # thin HTTP handlers
│   │   ├── invoices.go                # 30-80 lines each
│   │   ├── customers.go
│   │   └── payments.go
│   ├── middleware/
│   │   ├── session.go                 # user identity
│   │   ├── csrf.go
│   │   └── cors.go
│   ├── templates/                     # server-rendered (or empty if SPA)
│   └── static/
│
├── miniapps/                          # Layer 2: business logic
│   ├── invoices/
│   │   ├── miniapp.json               # manifest: tools, classes, capabilities
│   │   ├── tools/
│   │   │   ├── create.go              # Class A
│   │   │   ├── list.go                # Class A
│   │   │   ├── post_to_ledger.go      # Class D
│   │   │   └── send_reminder.go       # Class B
│   │   ├── capabilities/              # typed clients to external APIs
│   │   │   └── accounting.go
│   │   └── fixtures/
│   │       ├── create.json
│   │       └── post_to_ledger.json
│   ├── customers/
│   └── payments/
│
├── config/
│   └── env.go                         # Fly secrets, Neon URL, substrate config
│
├── migrations/                        # SQL (app schema, not framework)
│
└── infra/
    └── fly.toml                       # process groups: gateway, river, dbos
```

**Key boundaries enforced by convention and CI:**

- `web/` **imports** `miniapps/` (to register tools) and framework packages (to invoke).
- `miniapps/` **never imports** `web/`. If a miniapp needs presentation data, it returns structured data; the web layer formats it.
- `miniapps/` **never imports** `net/http`. If a miniapp needs HTTP, it does so through the capability layer, not directly.
- Framework packages never import your app.

A CI rule can enforce this with a simple import-checker: "no `web/...` in `miniapps/...`" and "no `net/http` in `miniapps/...`."

---

## 6. Request lifecycle: a web API in motion

Consider `POST /api/invoices/{id}/post` — the user clicks "Post to Ledger" in the UI.

### Sync path (Class A tool)

```
1. Browser: POST /api/invoices/123/post  (session cookie)
2. web/middleware/session.go: resolve user → user_id, tenant_id
3. web/handlers/invoices.go:ParseRequest
4. web/handlers/invoices.go → framework.Invoke(ctx, InvocationRequest{
     Tool: "invoices.post_invoice",
     Input: {invoice_id: "123"},
     Principal: UserPrincipal(user_id, tenant_id),
     IdempotencyKey: "tenant:invoice:123:post",
   })
5. framework:
   a. resolve tool from registry
   b. check manifest: tool is Class A, requires capability "accounting.bill.create"
   c. authorize: user has permission to post invoices in this tenant?
   d. run record created: run_id, started_at, actor=user_id
   e. dispatch to Class A handler
6. Class A handler (miniapps/invoices/tools/post_to_ledger.go):
   a. BEGIN tx
   b. SELECT invoice FOR UPDATE
   c. capability.Call(ctx, "accounting.bill.create", {invoice}) → external
        ↳ capability proxy handles auth, logs, idempotency, returns bill_id
   d. UPDATE invoice SET status='posted', bill_id=...
   e. INSERT effect_record (CONFIRMED)
   f. COMMIT
   g. return {status: 'posted', bill_id}
7. framework: run record updated: completed
8. web/handlers/invoices.go: map result → HTTP 200 {bill_id, status}
9. Browser: renders success
```

The handler in step 3–4 is ~15 lines. All of the interesting work is in the miniapp tool (step 6) and the framework (step 5).

### Async path (Class B/C/D tool)

Consider the same UI action triggering a Class D workflow (post + wait for approval + confirm):

```
1-4. same as above, but tool is "invoices.post_invoice_with_approval" (Class D)
5. framework:
   a-c. same
   d. run record: run_id
   e. dispatch to DBOS: workflow_id = "invoice:123:post:approval"
   f. return RunHandle{run_id, workflow_id, status: "running"}
6. web/handlers/invoices.go: HTTP 202 {run_id, status: "running", poll_url: "/api/runs/{run_id}"}
7. Browser:
   a. receives 202
   b. polls GET /api/runs/{run_id} every 2s (or SSE)
8. Later: workflow hits approval wait → framework marks run status "needs_approval"
9. GET /api/runs/{run_id} → {status: "needs_approval", approval_url: "/approvals/{id}"}
10. User clicks approval link (out-of-band channel enforced) → approves
11. Workflow resumes → completes
12. GET /api/runs/{run_id} → {status: "completed", result: {...}}
```

The web handler is still thin. The frontend handles polling. The framework's run envelope is the same one MCP returns — no special casing for web.

---

## 7. Class A/B/C/D from web handlers — code shapes

### Class A — read or single-record write

```go
// miniapps/invoices/tools/create.go
func Create(ctx context.Context, in CreateInput) (CreateOutput, error) {
    // sync: validate, tx, effect intent if external
    tx, _ := db.Begin(ctx)
    defer tx.Rollback(ctx)

    invoice := ... // insert
    if in.Notify {
        // enqueue Class B job inside the same tx
        framework.Enqueue(ctx, tx, "notifications.invoice_created", ...)
    }
    tx.Commit(ctx)
    return CreateOutput{InvoiceID: invoice.ID}, nil
}
```

```go
// web/handlers/invoices.go
func handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
    principal := session.FromContext(r.Context())
    var in CreateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        http.Error(w, err.Error(), 400); return
    }
    out, err := framework.Invoke[CreateInput, CreateOutput](
        r.Context(), Invocation{
            Tool: "invoices.create",
            Input: in,
            Principal: UserPrincipal(principal.UserID, principal.TenantID),
        })
    if err != nil { writeError(w, err); return }
    writeJSON(w, 201, out)
}
```

**Class A handlers complete synchronously.** The web layer returns the result. The frontend treats it like any REST call.

### Class B — fire-and-forget from web

```go
// web/handlers/invoices.go — user clicks "Send Reminder"
func handleSendReminder(w http.ResponseWriter, r *http.Request) {
    principal := session.FromContext(r.Context())
    invoiceID := chi.URLParam(r, "id")

    handle, err := framework.Invoke[SendReminderInput, RunHandle](
        r.Context(), Invocation{
            Tool: "invoices.send_reminder",   // Class B
            Input: SendReminderInput{InvoiceID: invoiceID},
            Principal: UserPrincipal(principal.UserID, principal.TenantID),
            IdempotencyKey: "tenant:" + principal.TenantID + ":invoice:" + invoiceID + ":reminder",
        })
    if err != nil { writeError(w, err); return }
    // 202 Accepted + run handle
    writeJSON(w, 202, handle)
}
```

Frontend receives `{run_id, status}`, shows "sending…", polls or subscribes.

### Class C — bounded batch from web

```go
// web/handlers/imports.go — user uploads CSV
func handleImport(w http.ResponseWriter, r *http.Request) {
    principal := session.FromContext(r.Context())
    // stage file to object storage or read into memory (bounded)
    rows := parseCSV(r.Body)  // validate shape, cap count

    handle, err := framework.Invoke[ImportInput, RunHandle](
        r.Context(), Invocation{
            Tool: "contacts.bulk_import",   // Class C
            Input: ImportInput{
                Rows: rows,
                BatchID: uuid.New().String(),
            },
            Principal: UserPrincipal(principal.UserID, principal.TenantID),
            IdempotencyKey: "tenant:" + principal.TenantID + ":import:" + batchID,
        })
    if err != nil { writeError(w, err); return }
    writeJSON(w, 202, handle)
}

// web/handlers/runs.go — polled by frontend for progress
func handleGetRun(w http.ResponseWriter, r *http.Request) {
    runID := chi.URLParam(r, "id")
    status, err := framework.GetRun(r.Context(), runID)
    if err != nil { writeError(w, err); return }
    writeJSON(w, 200, status)  // {status, progress: {processed: 250, total: 500}, ...}
}
```

Class C gives the frontend a progress endpoint without custom work — the framework's run record carries progress (`processed`/`total` for batches).

### Class D — durable workflow from web

```go
// web/handlers/onboarding.go — user starts employee onboarding
func handleStartOnboarding(w http.ResponseWriter, r *http.Request) {
    principal := session.FromContext(r.Context())
    var in OnboardingInput
    json.NewDecoder(r.Body).Decode(&in)

    handle, err := framework.Invoke[OnboardingInput, RunHandle](
        r.Context(), Invocation{
            Tool: "hr.onboard_employee",   // Class D
            Input: in,
            Principal: UserPrincipal(principal.UserID, principal.TenantID),
            IdempotencyKey: "tenant:" + principal.TenantID + ":employee:" + in.EmployeeID + ":onboard",
        })
    if err != nil { writeError(w, err); return }
    writeJSON(w, 202, handle)
}

// web/handlers/approvals.go — the approver comes here from an email link
func handleApprove(w http.ResponseWriter, r *http.Request) {
    approvalID := chi.URLParam(r, "id")
    // NOTE: this route is authenticated separately (magic link with step-up)
    // and MUST NOT be reachable from the agent session. See §8.
    err := framework.Approve(r.Context(), approvalID, approverFromSession(r))
    if err != nil { writeError(w, err); return }
    w.WriteHeader(204)
}
```

The Class D workflow can take days. The frontend handles it exactly like a Class C: poll `GET /api/runs/{id}` for status. When the workflow enters approval, the status is `needs_approval` and includes an `approval_url` the user can click.

---

## 8. User-principal vs agent-principal on the same miniapp

This is the single most important section for anyone building Shape 3 (web app + agent parity).

The **same miniapp tool** is invoked by two different callers:

| Caller | Principal model | Authorization | Audit actor |
|---|---|---|---|
| User via web UI | Single principal: the user | User's roles + entity-level access | `actor_type=user, actor_id=user_id` |
| Agent via MCP | Two principals: agent + delegated user | Agent scope ∩ user permissions | `actor_type=agent, actor_id=agent_id, on_behalf_of=user_id` |

The framework's `Invocation` struct has a `Principal` field that carries whichever model applies. The miniapp tool **does not know or care** which model triggered it — it receives a `Principal` and calls `capability.Call(ctx, ...)`, which authorizes against the effective authority.

**Concrete implementation rules:**

1. **The web UI's session never produces an "agent" principal.** A web request is always a user principal. An agent can invoke the same tool, but through a different route (MCP) with a different principal.

2. **The approval route must not be reachable from the agent session.** A separate authentication flow (magic link, separate login with step-up, mobile app push) handles approvals. The agent's token cannot reach `/approvals/*`. This is not policy — it is routing: approval routes live in a separate handler group with a separate middleware chain. Enforce it structurally.

3. **Approval decisions are made by humans, not by the same user who initiated.** For high-risk operations, the framework enforces separation of duties: the approver cannot be the initiator. For user-initiated web operations, this means a second human approves. For agent-initiated operations, the delegated user (or their manager) approves out of band.

4. **The audit trail is unified.** Both user and agent invocations write to the same `tool_invocations` table with the same schema. A query "what did anyone do on invoice 123 yesterday?" returns both, with `actor_type` distinguishing them.

5. **The frontend never sees the agent's token, and the agent never sees the frontend's session cookie.** They are separate authentication flows. The framework's session middleware handles users; the framework's MCP router handles agents; the two do not overlap.

The invocation service is the single point where both principals are resolved to a common `Principal` shape, and everything downstream operates on that shape.

---

## 9. Frontend patterns for async miniapps

Class A returns synchronously — the frontend treats it like any REST call. Class B/C/D return run handles. Three patterns work:

### Pattern 1 — Poll (simplest, works everywhere)

```javascript
const resp = await fetch('/api/invoices/123/post', {method: 'POST'});
const {run_id, status} = await resp.json();
while (true) {
    const s = await (await fetch(`/api/runs/${run_id}`)).json();
    if (s.status === 'completed') return s.result;
    if (s.status === 'failed') throw new Error(s.error);
    if (s.status === 'needs_approval') { showApprovalLink(s.approval_url); return; }
    await sleep(2000);
}
```

Fine for most cases. Costs a request per 2 seconds per active async operation. For a batch import of 500 records, this is 30–300 requests — acceptable, but crude.

### Pattern 2 — Long-poll (better for latency)

Add `wait_ms` to the run endpoint: `GET /api/runs/{id}?wait_ms=20000`. Server holds the connection until status changes or timeout. Framework's `GetRun` supports long-poll.

Reduces request count by ~10× while keeping the client code almost identical. Preferred for user-facing async operations.

### Pattern 3 — Server-Sent Events (best for batches and workflows)

For long-running operations with progress updates, SSE is the right tool:

```javascript
const es = new EventSource(`/api/runs/${run_id}/events`);
es.onmessage = (e) => {
    const update = JSON.parse(e.data);
    renderProgress(update);   // {processed: 250, total: 500}
    if (update.status === 'completed') es.close();
};
```

Framework can publish run transitions to a per-run SSE channel. The web layer's job is just to proxy the channel.

**Recommendation:** Poll for operations under 30 seconds; long-poll for 30s–10min; SSE for >10min or batch progress. The framework supports all three; the choice is presentational.

### Approval pattern (different flow)

Approval is not an async run polling. It is a **separate user journey**:

1. User is shown "This action requires approval. Approver: manager@example.com."
2. User clicks "Request approval" → framework creates approval record → sends out-of-band notification.
3. Approver receives email with magic link → clicks → lands on approval page (separate auth flow, step-up required) → approves.
4. Approval completes → workflow resumes → original user is notified.

The web app has three UI surfaces for this: the initiator's "pending approval" view, the approver's approval page, and the notification channels. The framework provides the approval table and API; the web app provides the UI.

---

## 10. Anti-patterns

The following are the recurring mistakes when adopting this framework for a web app.

### Anti-pattern 1: Business logic in HTTP handlers

```go
// WRONG
func handlePostInvoice(w http.ResponseWriter, r *http.Request) {
    // fetch invoice
    // call accounting API directly
    // update database
    // send notification
}
```

This bypasses the framework entirely: no run record, no effect ledger, no capability proxy, no reconciliation. The next developer who needs to add "wait for approval" or "expose to agents" has to rewrite.

**Fix:** the handler parses HTTP and calls the invocation service. Business logic lives in `miniapps/`.

### Anti-pattern 2: Miniapp calling miniapp directly

```go
// WRONG
func PostInvoice(ctx context.Context, in PostInvoiceInput) (PostInvoiceOutput, error) {
    // ...
    invoice, _ := createInvoiceTool.Create(ctx, ...)  // bypasses framework
}
```

Bypasses the invocation service. The nested call gets no run record, no audit trail link to the parent, no auth check (the parent's principal is assumed).

**Fix:** internal miniapp-to-miniapp calls go through the framework:

```go
invoice, err := framework.Invoke[CreateInput, CreateOutput](ctx, Invocation{
    Tool: "invoices.create",
    Input: ...,
    Principal: ctx.Value(PrincipalKey).(Principal),   // propagate
    ParentRunID: currentRunID,                         // for audit chain
})
```

Or, if the call is purely internal and never agent-exposed, declare it as a helper and document the exemption. But the default is through the framework.

### Anti-pattern 3: Frontend calling MCP

```javascript
// WRONG
const resp = await fetch('http://localhost:3000/mcp', { ... });   // MCP JSON-RPC
```

MCP is for agents. The web app's frontend calls the web app's REST endpoints. If the same business operation is exposed via both, they are two routes to the same invocation service, not one route calling the other.

**Fix:** frontend → REST → web handler → invocation service. Agent → MCP → MCP router → invocation service. Same destination, different routes.

### Anti-pattern 4: Trusting the frontend's authorization

```go
// WRONG
func handleApprove(w http.ResponseWriter, r *http.Request) {
    // frontend only shows approve button to managers, so we're fine
    doApprove(...)
}
```

The frontend hides the button. The API is still callable. Authorization must be enforced server-side, in the framework, against the user's actual roles.

**Fix:** the framework's `Invoke` always authorizes against the principal. There is no "I trust the frontend" mode. The frontend's UI hiding is a UX nicety, not a control.

### Anti-pattern 5: Building Class D workflows for CRUD

```go
// WRONG — a DBOS workflow to read a record
func ReadInvoice(ctx dbos.Context, id string) (Invoice, error) {
    return dbos.RunAsTransaction(ctx, func(...) { return queryInvoice(...) })
}
```

A read has no durable state. Wrapping it in a workflow costs a checkpoint write, a worker thread, and latency, with zero benefit.

**Fix:** Class A for reads. Class D only when ADR-001's gates pass.

### Anti-pattern 6: No run handle for async

```go
// WRONG — returns 202 with no tracking
func handleImport(w http.ResponseWriter, r *http.Request) {
    go doImportAsync(...)   // fire and forget
    w.WriteHeader(202)
}
```

The user has no way to know if the import succeeded. The framework can't audit an operation it doesn't know about.

**Fix:** use the framework's run envelope. Every async operation gets a `run_id` and a status endpoint. Even if you think "the user won't care," they will — and the audit trail needs it.

### Anti-pattern 7: Mixing presentation and business state in the same table

```go
// WRONG — the web app's UI state mixed with business state
CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    amount DECIMAL,
    -- business state
    -- UI state:
    last_viewed_by UUID,
    sort_order INT,
    ...
);
```

The framework's audit trail, effect ledger, and reconciliation all key off business tables. UI state in business tables makes the audit trail noisy and the business tables hard to migrate.

**Fix:** separate the web app's UI state into its own tables or its own schema (`web_state.invoices_ui`), keyed by the business entity ID. Business tables hold business state.

### Anti-pattern 8: Approval route reachable from agent

```go
// WRONG — same route group, same auth for both surfaces
r.Route("/api", func(r chi.Router) {
    r.Use(sessionMiddleware)  // accepts user OR agent tokens
    r.Post("/approvals/{id}/approve", handleApprove)
    r.Get("/invoices", handleListInvoices)
})
```

An agent can call the approval endpoint and approve its own request. This defeats the entire approval gate.

**Fix:** approval routes live in a separate route group with a separate middleware chain. The agent's token family cannot reach them. Structural, not policy.

---

## 11. Worked example: invoice management app

Concrete composition of everything above. A SaaS app where finance teams create, post, and manage invoices, and where finance operators can also ask their AI assistant to do the same.

**Domains:**
- `invoices`: create, list, post to ledger, send reminder, approve
- `customers`: create, list, update
- `payments`: record, reconcile

**Tools by class:**

| Tool | Class | Why |
|---|---|---|
| `invoices.create` | A | Single tx, no external effect (external posting happens later) |
| `invoices.list` | A | Read |
| `invoices.get` | A | Read |
| `invoices.update` | A | Single tx write |
| `invoices.post_to_ledger` | D | External effect + approval gate + saga compensation |
| `invoices.send_reminder` | B | Single external effect, fire-and-forget |
| `invoices.bulk_import` | C | Multi-record with per-record progress |
| `payments.reconcile` | C | Multi-record reconciliation |
| `customers.create` | A | Single tx |
| `customers.list` | A | Read |

**Web routes:**
```
POST   /api/invoices                         → invoices.create           (A, sync)
GET    /api/invoices                         → invoices.list             (A, sync)
GET    /api/invoices/{id}                    → invoices.get              (A, sync)
PATCH  /api/invoices/{id}                    → invoices.update           (A, sync)
POST   /api/invoices/{id}/post               → invoices.post_to_ledger   (D, async, requires approval)
POST   /api/invoices/{id}/reminder           → invoices.send_reminder    (B, async)
POST   /api/invoices/import                  → invoices.bulk_import      (C, async)
GET    /api/runs/{id}                        → framework.GetRun          (long-poll)
GET    /api/runs/{id}/events                 → framework.RunSSE          (SSE)

# Approvals — separate route group, separate middleware
POST   /approvals/{id}/approve               → framework.Approve         (out-of-band auth)
POST   /approvals/{id}/reject                → framework.Reject          (out-of-band auth)
GET    /approvals                            → framework.ListApprovals   (approver's view)
```

**MCP tools (agent parity):**
The same miniapp tools are exposed via MCP with the same names. An AI assistant with the user's delegation can invoke `invoices.list`, `invoices.create`, `invoices.post_to_ledger` (subject to approval), etc. The agent cannot invoke `/approvals/*` — those are web-only and out-of-band.

**The frontend's job:**

- Class A operations: fetch → render (standard REST)
- Class B/C/D operations: fetch → 202 → poll or SSE → render progress → render result
- Approval operations: never called by the initiator's frontend; only by the approver's separate session

**The framework's job:**

- Resolve tool from registry
- Authorize against principal (user or agent)
- Create run record
- Dispatch to substrate
- Call capabilities through the proxy
- Track effect states
- Reconcile UNKNOWN outcomes
- Emit audit events
- Manage the approval table
- Serve `GetRun` and SSE

**The miniapp's job:**

- Implement the business logic (Class A handler, Class B job, Class C batch, Class D workflow)
- Declare the capability bindings
- Provide fixtures
- Declare the class in the manifest

**The user's experience:**

- Fast operations (read, create, update) feel like normal web app operations
- Longer operations (post to ledger, batch import) show a progress indicator and a result when done
- Approval operations notify the approver out of band and produce a result for the initiator when approved

**The agent's experience:**

- Same tools, discovered via MCP `list_tools`
- Same async envelope, same `get_run`
- Approval flows surface as `needs_approval` with a link the user must click (not the agent)
- Same audit trail as the web UI

---

## 12. Summary

**The framework is below the web layer.** Your web handlers stop being where business logic lives; they become transport adapters. The invocation service is the single point through which every business operation passes, whether it originated from a user's click or an agent's tool call.

**Three shapes.** API server (no UI), user-facing web app (UI in the web layer), hybrid (web + MCP parity). Shape 3 is the recommendation for serious products.

**Three layers.** Web transport (presentation), miniapps (business logic), framework infrastructure (cross-cutting). Enforce the boundaries with CI.

**Class A returns sync; Class B/C/D return run handles.** The frontend pattern depends on the class: fetch for A, poll/long-poll/SSE for B/C/D. The run envelope is the same one MCP uses, so no special-casing.

**Same miniapp, two principals.** User requests are single-principal; agent requests are two-principal (agent + delegated user). The miniapp doesn't know the difference — it receives a `Principal` and calls `capability.Call`. Approval routes are structurally unreachable from agent sessions.

**The framework is not for every web app.** Pure CRUD with no external effects and no agent exposure is simpler without it. The framework earns its place when business operations have non-trivial execution semantics — UNKNOWN-outcome risk, multi-step processes, approval requirements, or agent exposure.

**The discipline that makes it work:** every business operation is declared with a class in the manifest; every class carries its obligations; every external effect goes through the capability proxy; every async operation returns a run handle; every approval is enforced server-side in a separate auth flow. Follow the discipline and the web layer stays thin, the audit trail stays complete, and the same business logic serves users and agents identically. Violate it and you get a web app with a framework-shaped hole where the framework should be.


## gemini

## 1. Using the Framework for a Web API Server or Web App Backend

Even when the caller is a **Web-based API server** or a **user-facing Web app server** rather than an AI agent harness, the same core execution model applies: a client requests a business operation, and the miniapp framework executes it durably, safely, and with full auditability. The earlier vision treated miniapps as human-facing UIs or rule builders, whereas the current architecture treats miniapps as **governed execution units** invoked over typed contracts and HTTP/REST APIs.

The main shift for a Web API or Web app backend is **who acts as the caller and how responses/approvals flow**:

* **REST / OpenAPI as the Primary Web Transport:** A Web API server or Web app backend calls the canonical invocation service over signed HTTP/REST endpoints (such as `POST /v1/miniapps/{id}/invocations`, `GET .../invocations/{id}`, signal/approval endpoints, and cancellation) rather than relying on MCP.


* **Human & Service Principals Instead of Agent Harnesses:** A user-facing Web app authenticates the human user (e.g., via SSO/OIDC or session/bearer auth) and passes an explicit authority/user principal alongside a service/client identity, while server-side authorization enforces tenant isolation, roles, trust modes, and blast-radius limits.


* **Sync-First for Web Requests, Async Handles for Long Work:** Fast Class A operations (or operations that finish within a short synchronous budget) return results inline so Web UIs and API clients do not pay unnecessary round-trip overhead. Multi-step, batched, or multi-day Class B/C/D operations return a `run_id` and status immediately so the Web app can poll, subscribe via webhook/SSE/WebSocket on its own edge, or render approval actions without holding an open HTTP request for hours or days.



---

## 2. How Each Miniapp Category Serves a Web Backend

| Miniapp Category | Web API / Web App Use Case | How Your Web Server Interacts |
| --- | --- | --- |
| **Class A (Sync Handler + Neon Tx)** | Fetching profile/order/record data, validating form/API payloads, or performing a self-contained DB update.

 | Calls the endpoint synchronously and renders or returns the result inline.

 |
| **Class B (River Job)** | Sending confirmation emails/notifications, generating exports/reports, or enqueueing work atomically when a user submits a form.

 | Enqueues via `InsertTx` in the same Neon transaction as the business mutation and receives a job/run reference.

 |
| **Class C (River Batch / Bounded Pipeline)** | Importing a CSV/batch of records, running multi-step batch enrichment, or syncing a bounded set of items with per-record progress.

 | Starts the batch, displays progress/status from the progress record, and supports resume/retry without duplicating completed items.

 |
| **Class D (DBOS Durable Workflow)** | Multi-day onboarding, staged provisioning, workflows with durable timers/sleeps, or multi-step processes waiting on external webhooks or human approvals.

 | Starts the workflow, receives a `run_id`, lets users approve/reject or signal events out-of-band, and survives server restarts or multi-day waits.

 |

---

## 3. Concrete Integration Flow for a Web App Server

1. **Authenticate & Establish Context:** Your Web app verifies the logged-in user and tenant context, then calls the miniapp platform using a service/client credential plus the delegated user/authority principal—never trusting a raw client-supplied `tenant_id` without server-side verification.


2. **Send an Idempotent Request:** Every mutation or workflow start includes a deterministic idempotency key (such as `tenant:entity:kind:op`) so network retries, double-clicks, or Web API retries attach to or deduplicate cleanly rather than creating duplicate records or side effects.


3. **Handle Sync vs. Async Responses:**
* If synchronous (Class A / within sync budget), return the typed JSON payload directly to the Web client.


* If asynchronous (Class B/C/D), persist or return the `run_id` and status to the Web client, which can poll status, receive webhook callbacks, or prompt an authorized human approver when a workflow enters a `confirm`/approval wait.




4. **Keep Credentials & Side-Effect State Out of the Web Tier:** Your Web server does not hold raw third-party SaaS OAuth tokens or implement ad-hoc retry/reconciliation loops. The capability proxy / token broker manages external SaaS and SSRF-hardened custom REST calls, while the effect ledger records `PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN` and quarantines `UNKNOWN` outcomes for reconciliation.


5. **Enforce Database & Deployment Boundaries:** Keep Fly.io Machines in the same region as your Neon primary, use transaction-local tenant context (`SET LOCAL` / `set_config(..., true)`) with RLS defense-in-depth, and keep DBOS on a direct Postgres connection with a single dedicated executor until measured scale/recovery needs dictate otherwise.



---

## 4. Practical Dos and Don'ts for Web Backend Devs

* **DO** treat the miniapp framework as your **business-process and external-integration execution layer**, while your Web app owns UI rendering, session/browser auth, and Web-specific presentation state.


* **DO** pass deterministic idempotency keys and explicit authority principals on every call, and enforce out-of-band approval authorization (never letting a caller self-approve restricted actions).


* **DO** mark or sanitize untrusted external text returned from SaaS/REST integrations if your Web app or an LLM assistant displays it back to users.


* **DON'T** hold long-lived HTTP connections open for multi-step Class C or multi-day Class D workflows.


* **DON'T** duplicate workflow state machines, cron/retry loops, or raw SaaS token handling inside your Web server when the miniapp substrate already owns them.


* **DON'T** route simple Class A reads or single-step synchronous actions through DBOS workflows.



If you are building this Web API/app server, which primary client pattern are you targeting first—a **backend-to-backend REST API service** or a **human-facing Web UI with approval/status views**?


## Z

# Building Web API and Web App Servers on the Miniapp Framework

| Field | Value |
|---|---|
| Scope | How to structure a web-based API server and a user-facing web app server on top of the miniapp framework's canonical invocation service and execution classes |
| Basis | plan4.1 sections 2, 6.1, 9.1-9.5, 13, 14, 15, 17; App Category Guidelines (companion document) |
| Audience | Developers building HTTP surfaces (machine API and browser UI) over the framework |
| Version | 1.0 — 2026-11-22 |

---

## 1. The Short Answer

The framework was not designed as a web framework, and you should not try to turn it into one. It was designed as an API-first business-process automation platform with one canonical invocation service and thin adapters in front of it (plan4.1 sections 6.1 and 13). That architecture is precisely what makes both of your scenarios straightforward:

- **A web-based API server** is mostly already there. The signed REST/OpenAPI adapter is the canonical, versioned HTTP surface (section 13), and the `gateway` Machine process exists to serve it — REST/MCP/CLI, Class A handling, and the webhook receiver, always on (section 17). Your job is to configure and extend that adapter: map manifests to OpenAPI, map principals to tokens, map the run envelope to HTTP status codes, and keep business logic where it belongs, in miniapps.
- **A user-facing web app server** is one more thin adapter. You add an HTTP edge that owns browser concerns — login, sessions, cookies, CSRF, HTML or server-rendered pages, SSE progress streams — and every consequential action it performs is an invocation of a declared miniapp tool through the same dispatch, authorization, audit, and approval machinery the agents use. The edge renders; the miniapps execute; the framework owns the hard parts.

The rule that makes both cases safe is section 13's: **one canonical invocation service; thin adapters sharing execution, authorization, and audit — never a parallel implementation** (A16). A web server that re-implements its own job system, approval flow, or state layer next to the framework is not using the framework; it is starting a second codebase with the same bugs. The layering in chapter 2, the principal mapping in chapter 3, and the endpoint-to-class mapping in chapter 6 are the concrete application of that rule to HTTP.

One boundary from section 2 is worth stating up front so expectations are right: a UI/dashboard generator is an explicit non-goal. The framework will not generate or host your web UI. You own the presentation layer; the framework owns execution, authorization, recovery, and audit. This document is about wiring those two halves together correctly.

## 2. The Layering Rule: What Lives Where

Section 6.1's system context names every layer you will touch. The table below restates it as an ownership contract for web work. The binding rule behind it: every consequential action is enforced server-side (section 9.3), and adapters share execution, authorization, and audit (section 13).

| Layer | Owns | Must never contain |
|---|---|---|
| Transport adapters (REST/OpenAPI, MCP, CLI, and your HTML edge) | Protocol concerns: transport-level authentication, sessions, cookies, CSRF, content negotiation, status-code mapping, HTML/SSE rendering, webhook verification | Business logic, durable state, provider calls, class machinery, approval decisions |
| Canonical invocation service | Registry, authorization before dispatch (scopes, tenant, trust mode), run envelope, approvals, audit, blast-radius limits, kill switch | HTML, protocol formatting, per-endpoint special cases |
| Executors | Class A handler + transaction; River workers (B/C); DBOS executor (D); capability proxy and token broker; Sprite runner | Knowledge of which transport triggered the work |
| Neon Postgres | Authoritative state: app schema, river schema, dbos schema; runs, effects, approvals, audit, registry | Transport state such as browser sessions |

Two placement notes follow directly from section 17. First, your web edge lives in the trusted `gateway` Machine process — it is platform infrastructure, not a miniapp, so it is never declared in a manifest and never runs in a Sprite. Second, the miniapps the edge invokes run according to their own declared Dimension B placement per tool (machine/sprite/hybrid/local in the manifest's deployment section, section 9.1). A browser and an agent may end up invoking the same Class A tool; the tool neither knows nor cares which transport delivered the call.

## 3. Principals: How a Browser or API Client Becomes a Caller

Section 9.3 defines the identity model the web surfaces must feed: an **invoking principal** (who performs the call), an **authority principal** (on whose behalf and with whose authority), a tenant, scopes, and a trust mode bound per miniapp version per tenant. Every transport maps onto this same model, and once the mapping is done, dispatch cannot tell the transports apart.

- **Browser session (web app).** The human logs in through the platform's OAuth flows — code + PKCE, refresh rotation, revocation — via the token broker (section 14). The session cookie identifies the invoking principal (the web edge acting for the user); the logged-in human is the authority principal. Revocation fails closed: a revoked session stops passing dispatch before anything runs.
- **API token (API server).** A signed API token or key issued to a client maps to an invoking principal with manifest-declared scopes; the client's owner is the authority principal. Machine clients get no interactive approval UI, so tools they call must be trustworthy in the modes the client can exercise — typically `dry-run` and `autonomous`, with `confirm` reserved for flows that have a human in the loop somewhere.
- **After the gate, no special cases.** Authorization before dispatch, idempotency, effects, approvals, limits, and audit are identical for every caller. An HTTP handler that does anything consequential without going through dispatch is a section 14 violation, not a shortcut.

The trust-mode binding is what keeps the two web scenarios honest. A `confirm`-mode tool invoked by an agent produces a framework approval record and waits; the same tool invoked from a browser produces the same record and the same wait, which the UI renders as a confirmation screen. One mechanism, two renderings — never two mechanisms.

## 4. Case 1 — The Web-Based API Server

The signed REST/OpenAPI adapter is your public machine API. Build it in this order:

1. **Manifests are the source of truth.** Generate the OpenAPI document from the registry: tool names, input/output schemas, idempotency declarations, and error taxonomy come from `miniapp.json` versions (section 9.1). Never hand-write a parallel spec; a hand-written spec drifts from the validated manifest and A16's no-parallel-implementation rule applies to documentation as much as to code.
2. **Authentication at the door, authorization at dispatch.** The adapter verifies the token (signature, expiry, revocation) and maps it to the invoking principal; the invocation service enforces scopes, tenant, and trust mode before dispatch. The adapter performs no permission decisions of its own.
3. **Map the sync-first envelope to HTTP.** This is the core contract (section 9.4: `run_id`, `miniapp_id`, `version`, `status`, poll/signal/cancel). Within the synchronous budget, the tool's result returns inline as `200`. Work that cannot finish in budget returns `202` with the run handle, the status lives at a canonical status endpoint, and the envelope's `wait_ms` long-poll maps to a long-polling status GET. Signal and cancel map to explicit POST endpoints. There is no arbitrary active-handle expiry; terminal-history retention is the framework's retention policy, not the adapter's.
4. **Idempotency is a header.** Client-supplied `Idempotency-Key` maps onto the tool's declared idempotency key template, so duplicate submits collapse into one run (section 15). A reused key with a different payload is a client bug: reject it with a typed conflict error rather than silently running the new payload.
5. **Map the typed error taxonomy to status codes once.** Validation and admission failures (including class/placement admission) are `4xx` with the framework's typed error body; authorization failures are `403`; limit breaches are `429`; conflicts and idempotency mismatches are `409`/`422`; a lost race that produced a run handle is `202`, not an error. Write this mapping once, centrally, and never per endpoint.
6. **Inbound webhooks are receivers, not tools.** The gateway's webhook receiver verifies and deduplicates provider webhooks (section 14), then transactionally enqueues a Class B job in the same transaction as any motivating write. The webhook endpoint itself contains no business logic.
7. **Outbound webhooks and notifications are effects.** They go through the external-effect protocol (`PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN`, section 9.5) with reconciliation; a delivery that dies mid-send is quarantined and reconciled, never lost or blindly retried.

The endpoint-shape-to-class table in chapter 6 is the same decision procedure as the App Category Guidelines, viewed from the HTTP side. Nothing about receiving a web request changes a tool's class: a bulk import is Class C whether an agent, a CLI script, or an API client started it.

## 5. Case 2 — The User-Facing Web App Server

The web app adds exactly one thing to the architecture: a presentation adapter. Everything behind it is the canonical invocation service and the miniapp catalog you already have. The edge's responsibilities are strictly transport and presentation:

- **Login and session management.** OAuth login through the token broker, session cookies, logout, and revocation, mapped to the principal model of chapter 3.
- **CSRF and cookie hygiene.** Standard browser-defense concerns that live entirely at the edge; the framework never sees them.
- **Rendering.** Server-rendered pages or an SPA backend, forms, and pages that display typed results. Provider free text rendered into HTML is data, not instructions: escape it, mark it untrusted, respect the typed fields and size caps — the same rule agents follow, applied to templates.
- **Progress surfaces.** SSE or polling over the run envelope and Class C progress records, never bespoke status tables.

The discipline that keeps this honest is mapping every UI interaction onto a framework mechanism instead of inventing a web-specific twin:

| UI pattern | Framework mechanism | What you must not do |
|---|---|---|
| Confirmation dialog ("are you sure?") | Trust mode `confirm`; approval-gated invocation with the approval record (`PENDING -> APPROVED / REJECTED / EXPIRED`, expiry, material-change invalidation, out-of-band approver, separation of duties) | A second bespoke "confirmations" table or a JS-only confirmation with server trust |
| Progress bar for a long operation | Run envelope status and Class C progress records, polled or streamed over SSE | A parallel jobs/status table in the web database |
| Form POST that mutates data | Class A tool behind dispatch, idempotency key from a hidden form token | Handler-side business logic or direct database access |
| "We'll email you when it's done" | `202` + run handle; Class B job sends the notification as a ledgered effect | Spawning a goroutine or a cron job from the web server |
| Multi-day flow (onboarding, dunning, approvals over days) | Class D durable workflow with durable waits and events | Session state or scheduler tables as the process memory |
| Human approval inside a larger process | Class D with a durable event wait on the framework approval record | Polling an approval table from a cron loop |

Two rules deserve emphasis. First, **sessions are transport identity, never durable state**: if a process must survive a browser closing, a server restart, or a week of silence, it belongs in Class C or D, with the run envelope as its handle. Second, section 2's non-goal — no UI/dashboard generator as the product surface — is not a limitation to work around; it is the split working as designed. You own pixels and sessions; the framework owns execution, authorization, recovery, and audit. Code that blurs that line by doing business logic in templates or handlers is the web-shaped version of the shadow-machinery failure pattern from the App Category Guidelines.

## 6. Mapping Endpoints to Categories

The table below is the practical core of this document: the recurring HTTP shapes and where their work belongs. The class column follows the same ordered questions (Q1-Q4) as the App Category Guidelines; the HTTP shape column shows what the adapter returns.

| Endpoint pattern | Class | HTTP shape | Mechanics |
|---|---|---|---|
| Read or write completing in one transaction | A | `200` inline | One tx, ledgered external calls, idempotency key, entity key |
| Accept-and-process single unit | B | `202` + run handle | Commit motivating write and enqueue with `InsertTx` in the same tx; status endpoint |
| Bulk import/export over many records | C | `202` + run handle + progress | Progress record, per-record idempotency, bounded concurrency; poll or SSE |
| Scheduled generation and delivery | B (scheduled) | No caller; result delivered | River schedule; scheduling is not durability |
| Approval-gated single action | A or B + bounded approval wait | `202` + approval status, or `200` after wait | Framework approval table; bounded wait inside the sync envelope's rules |
| Multi-day business process, saga, cross-system flow | D | `202` + run handle | DBOS workflow; durable sleeps, events, compensation |
| Inbound provider webhook | B (receiver) | `2xx` fast, processing async | Verify and deduplicate, then transactional enqueue |
| Outbound webhook or notification | B via effect ledger | N/A (effect) | `PENDING -> ATTEMPTING -> ...` with reconciliation sweeper |
| Agent-authored UI extension or widget | Placement, not class | Rendered from workspace | Sprite preview; production only via trusted-build promotion |

## 7. Two Request Lifecycles

**Lifecycle A — an API client creates an order.** The client calls `POST /orders` with an `Idempotency-Key`. The gateway verifies the token and maps it to the invoking principal; dispatch checks scopes, tenant, trust mode, and blast-radius limits, and collapses the submit if the key was seen before. The tool is Class A: the handler opens one transaction, writes the order, records the payment effect intent (`PENDING`), calls the provider through the capability proxy with the tenant's brokered credentials, and records the outcome (`CONFIRMED`, `FAILED`, or `UNKNOWN`). Within budget, the adapter returns `200` with typed results; the effect ledger's outcome is visible to the client through a status endpoint. If the process dies after the provider call but before the outcome lands, the effect is `UNKNOWN`, quarantined, and reconciled toward a confirmed or not-applied state — the client's retry with the same key waits on the same identity instead of double-charging. Every window in that story has an owner, because the class and the platform contracts own it.

**Lifecycle B — a user clicks "Upgrade plan" in the web app.** The edge renders the confirm screen because the tool is `confirm` mode; clicking creates a framework approval record and the invocation waits bounded within the envelope's rules, or returns a handle the UI polls. The user approves out-of-band (email link or settings page); the record moves to `APPROVED`; the Class A handler applies the upgrade with the same effect-protocol protection as Lifecycle A. If the approval expires, the record is `EXPIRED`, the UI shows it, and nothing was half-applied. Contrast the neighboring case: an onboarding flow that spans two weeks — welcome emails, manager approval, provisioning across two SaaS systems — is not a confirm button; it is a Class D workflow from the start, with durable waits on events and compensation registered before each external action. The web edge merely renders its progress; it does not participate in its logic.

## 8. What Not to Do

These are the web-specific versions of the failure patterns the category system exists to prevent. Each row is a real pattern that shows up in web servers that grow next to a framework like this one.

| Anti-pattern | Why it fails | Do instead |
|---|---|---|
| Business logic in HTTP handlers or templates | Shadow machinery: unowned crash windows, unaudited effects, untestable outside HTTP | Handlers call declared tools; templates render typed results |
| Goroutines, timers, or cron jobs spawned by the web server | A second queue with no retries, no dead-letter, no ledger | Class B via transactional enqueue, Class D when durable |
| Bespoke approval, confirmation, or job-status tables | Two sources of truth; audit and revocation stop covering the web path | Framework approval records and the run envelope |
| Sessions or hidden form fields as durable state | State dies with the process or the browser; recovery is impossible | Class C/D state in Neon; run handles in the UI |
| HTML fragments or formatting inside miniapp tools | Tools become transport-coupled; agents and API clients get garbage | Tools return typed data; the edge renders |
| A hand-written OpenAPI spec or a second "public API" code path | Drifts from the validated manifest; A16's parallel-implementation violation | Generate OpenAPI from the registry; extend the canonical adapter |
| Browser or client calling providers directly | Breaks the credential boundary (section 14): credentials never leave the broker | All provider calls through the token broker and capability proxy |
| Treating the web edge as a miniapp | Infrastructure is not a tool; placement and admission rules would be meaningless | Edge runs in the trusted `gateway` Machine process; it invokes miniapps |
| Shipping endpoints without `validate`/`publish` because "it is internal" | Unvalidated class declarations, unreviewed placement, no audit trail | Same lifecycle for every surface: validate, test, preview, publish, deploy |

## 9. Build Sequence

Use the framework's own lifecycle commands (section 13), because they are the same path CI uses — the parity rule (A16) means your local commands and the GitHub Actions workflow (validate, test, build, publish, deploy) call the same services and policies:

1. **Scaffold business logic as miniapps.** `miniapp init --class --profile` per miniapp, one class per tool per the App Category Guidelines' decision procedure; declare class, idempotency keys, entity keys, trust modes, limits, and placement profiles in `miniapp.json`.
2. **Develop.** `miniapp dev` runs the local dev server; `--sprite` opens an isolated workspace for anything untrusted or generated.
3. **Validate and test.** `miniapp validate` runs manifest, schema, policy, placement, and profile-digest admission checks — this is where a wrong class or placement fails. `miniapp test` runs unit through crash-boundary tests over `testkit/`.
4. **Preview.** `miniapp preview` gives a disposable Neon branch plus a Sprite preview environment; wire the web edge's staging config to it.
5. **Publish and deploy.** `miniapp publish` produces the immutable versioned artifact with provenance; `miniapp deploy` brings up the Machines services and workers: `gateway` (your REST adapter, Class A, webhook receiver, and the HTML edge), `river` workers, and the dedicated `dbos` executor per section 17's table.
6. **Operate.** `miniapp status`, `runs`, `logs` (one status family) cover execution history, approvals, retries, and recovery; environment promotion local -> preview -> staging -> production reuses the same deployment profiles and migration gates.

The web edge itself follows the Machine operations bar from section 17: health checks, graceful shutdown, and gated versioned migrations. It is always-on infrastructure — it must never depend on worker availability to serve reads, which is exactly why the `gateway` and `river` are separate process groups.

## 10. Quick Reference

**The wiring in one paragraph.** Browser or API client arrives at an adapter (REST/OpenAPI for machines; your session-and-HTML edge for humans). The adapter authenticates transport-level identity and maps it to the invoking principal plus authority principal, tenant, and scopes. The canonical invocation service authorizes before dispatch, applies limits, and routes by the tool's declared class: A executes inline in one transaction; B/C are enqueued transactionally and return a run handle (`202`); D is a gated DBOS workflow with its own record. Results follow the sync-first envelope — inline within budget, otherwise the run handle with poll/signal/cancel. Effects go through the ledger; approvals go through the one approval table; everything lands in Neon and the audit trail. The edge never holds business logic, provider credentials, or durable state.

| Question | Answer |
|---|---|
| Where does my web server live? | In the trusted `gateway` Machine process, as a thin adapter — never as a miniapp, never in a Sprite |
| Where does my business logic live? | In miniapps, one declared class per tool, organized per the App Category Guidelines |
| What does a long operation return? | `202` with the run envelope handle (`run_id`, status, poll/signal/cancel); `wait_ms` long-poll for near-term completion |
| Where does a confirmation dialog come from? | Trust mode `confirm` and the framework approval record — never a bespoke web confirmation table |
| Where do background jobs come from in a web request? | Transactional enqueue in the same commit as the motivating write (Class B/C); never goroutines or cron |
| Who talks to providers? | The token broker and capability proxy, always; credentials never reach the edge, the client, or a Sprite |
| What generates the API spec? | The registry, from validated manifests; hand-written specs drift and violate A16 |
| What state survives a browser closing? | Only what the classes persist in Neon, addressed by run handles and business identity — never sessions |

**Glossary additions for this document.** *Canonical invocation service*: the single service owning registry, authorization, dispatch, run handles, audit, and approvals; all adapters share it. *Thin adapter*: a transport layer (REST, MCP, CLI, HTML edge) that owns protocol concerns only and invokes through the canonical service. *Invoking / authority principal*: the caller identity versus the identity on whose authority the call proceeds (section 9.3). *Run envelope*: the uniform async contract — `run_id`, status, poll, signal, cancel — mapped onto HTTP as `202` plus status endpoints. *Gateway*: the always-on Machine process serving REST/MCP/CLI, Class A execution, webhook receiving, and your web edge. *Capability proxy*: the platform component that performs provider calls on behalf of untrusted code so credentials never leave the broker.


## claude

# Using the App-Category Framework for a Web API Server or User-Facing Web App

- **Audience:** developers building an HTTP API server or a server-rendered / SPA-backed web application in Go on Fly.io and Neon Postgres, using the A/B/C/D categories, DBOS and River
- **Companion documents:** `app_category_guidelines.md` (the categories and their rules), ADR-001 (`DBOS.md`), the plan4.1 review
- **Status of code samples:** illustrative shapes; verify SDK calls against pinned versions

---

## 1. The key idea: categories classify operations, not servers

A web server is a **transport and hosting layer**. It is not itself Class A, B, C or D. The categories apply to each **operation** the server exposes: every route, form action, webhook, scheduled trigger and admin command.

```text
HTTP route / form action / webhook / cron trigger
        │   (transport adapter: decode, authenticate, map result to HTTP)
        ▼
   Use case  ← this is what gets a category (A | B | C | D)
        │   (application shell: category-specific orchestration)
        ▼
   Domain core  (pure business rules, no I/O)
        │
        ▼
   Ports → adapters (Postgres, providers, clock, email, payments)
```

**Rule of thumb:** organize code *use case first, route second*. A route is a thin translator; the use case owns the category.

**Consequence:** the same use cases can later be exposed to AI agents (MCP, REST, CLI) with no change to business logic, because the HTTP layer is just one adapter over the use-case layer.

---

## 2. Which parts of the framework to use for a web app

You do not need the full miniapp framework for an ordinary web application. Adopt it in levels and stop when the next level no longer pays for itself (ADR-001's rule applies to the framework too).

| Level | What you adopt | Adopt when |
|---|---|---|
| **0 — Conventions** | Classify each use case; `domain/` + `app/` split; idempotency identity per write; tenant-scoped transactions | Always, from day one. Costs nothing |
| **1 — Shared helpers** | Idempotency-Key middleware, transaction helper with txn-local tenant context, `runs` table, audit events, effect ledger for external calls | First external write or first retry-prone endpoint |
| **2 — River** | Class B/C workers, transactional enqueue, unique jobs, reconciliation sweeper | First background job (email, PDF, webhook processing, import) |
| **3 — DBOS** | Class D workflows with decision records | First genuinely durable process (checkout with waits, onboarding sequence, approvals, multi-day journeys) |
| **4 — Agent surface** | Manifest, MCP/REST/CLI adapters, capability proxy, trust modes | When AI agents should invoke the same operations |

A pure CRUD admin tool may stay at Level 0–1 forever. That is a success, not a gap.

### What changes versus the agent-facing miniapp context

| Concern | Agent-invoked miniapp | Web API / user-facing web app |
|---|---|---|
| Caller | Agent principal acting for an authority principal | Human user (session/OIDC) or service client (API key/OAuth) |
| Trust modes (`dry-run`/`confirm`/`autonomous`) | Core control | Mostly replaced by UI confirmation dialogs and approval workflows for risky admin actions |
| Result handling | Free text from providers marked untrusted | Output encoding, CSP, XSS protection for anything rendered in a browser |
| Extra web-only threats | — | CSRF, CORS, session fixation, clickjacking, open redirects |
| Latency budget | Agent sync budget (~5 s) | Browser budget is tighter; choose per endpoint (e.g. ~1–2 s p95), otherwise move to B/C/D with a progress UI |
| Capability proxy / token broker | Always | Only if the app calls third-party SaaS on behalf of users |

---

## 3. Classifying the operations of a typical web app

Start by listing every operation, then classify each one. Most will be Class A.

| Operation | Category | Notes |
|---|---|---|
| Render page, list orders, search, view profile | **A** | Read; no effects |
| Login, logout, refresh session | **A** | Authentication is a default DBOS rejection |
| Update profile, create project, edit settings | **A** | One transaction; audit row in the same transaction |
| Submit a form (create record) | **A** | Idempotency key defends against double submit |
| Send welcome email after signup | **B** | Enqueue in the signup transaction |
| Password reset email | **A + B** | Token row with expiry (A); email job (B). **Not D** |
| Email verification | **A + B** | Token table + expiry; no durable workflow needed |
| Payment provider webhook | **B** | Verify signature, acknowledge fast, enqueue, process idempotently |
| Generate invoice PDF | **B** | One retryable job |
| Scheduled report email | **B** | Periodic job |
| CSV import with progress bar | **C** | Per-row idempotency, progress, resume |
| Bulk archive / bulk email to a bounded list | **C** | Bounded concurrency, partial-failure policy |
| Checkout with payment, inventory and fulfillment | **D** (if multi-stage with waits or compensation) | Saga; stable workflow ID per order |
| Trial-ending drip sequence, renewal reminders | **D** | Durable waits over days |
| Multi-step onboarding with external verification | **D** | Waits for external events |
| Admin refund above a threshold | **A/B + approval table** | Bounded human wait; not reclassified to D |
| Account deletion with data erasure across systems | **C or D** | Decide by score; erasure must be reconcilable |
| Subscription lifecycle (trial → active → past_due → canceled) | **State in Postgres + D for the timed steps** | Entity state is a table; dunning/reminders are the durable part |

### Quick decision flow for any endpoint

1. Finishes inside the request and needs no resume? → **A**
2. One independently retryable job? → **B** (respond `202`)
3. Bounded batch with partial progress? → **C** (respond `202` with progress)
4. Durable waits, human/external events, multi-day, checkpointed resume? → **D candidate**; write the ADR-001 decision record first

If unsure, start lower and write the decision template for the higher class.

---

## 4. How the categories map onto HTTP

### 4.1 Response contract

| Category | HTTP behavior | Client experience |
|---|---|---|
| **A** | `200`/`201`/`204` with result, or `4xx` | Immediate result |
| **B/C/D** | `202 Accepted` + `Location: /runs/{run_id}` + body `{run_id, status}` | "Processing…" state; progress, completion, or "needs attention" |

Never return `200` for work that has only been queued.

### 4.2 The `runs` resource (one async pattern for everything)

```text
POST   /imports                 → 202 {run_id}
GET    /runs/{run_id}           → {status, progress?, result?, error?}
GET    /runs/{run_id}?wait=15s  → long-poll (optional)
POST   /runs/{run_id}/cancel    → request cancellation
POST   /runs/{run_id}/signal    → deliver an event (D only), authorized per use case
```

Suggested statuses: `ACCEPTED → QUEUED → RUNNING → WAITING → SUCCEEDED | FAILED | CANCELLED | NEEDS_ATTENTION`. `NEEDS_ATTENTION` covers quarantined UNKNOWN effects and expired approvals.

One table, one UI component, one agent contract.

### 4.3 Idempotency

- Accept an `Idempotency-Key` header on every write route (or derive a stable key from the business identity).
- Store `(tenant, route/use case, key) → run_id/response` in the same transaction as the write.
- A duplicate submit returns the original result or the same `run_id`.
- For browsers, also protect forms with a one-time form token and use Post/Redirect/Get so refresh does not resubmit.

### 4.4 Where the UI learns about progress

| Mechanism | Use when |
|---|---|
| Polling `GET /runs/{id}` with backoff | Default; simplest and robust |
| Long-poll (`wait=`) | Reduce chatter for short jobs |
| Server-sent events | Live progress for imports or long workflows |
| Email/in-app notification | Long waits (hours/days); the user is not watching |

---

## 5. Project layout

```text
cmd/
  web/            main for the gateway process (HTTP + Class A + enqueue/start)
  worker/         main for River workers (Class B/C + sweepers)
  dbos-worker/    main for DBOS workflow workers (Class D)
internal/
  transport/
    httpapi/      JSON handlers, middleware, OpenAPI
    webui/        templates or BFF handlers (if user-facing)
  usecases/
    orders/
      place_order/        # one directory per use case
        domain/           # pure rules (no I/O, clock, rand, framework)
        ports.go          # interfaces for effects
        handler.go        # Class A shell        (or)
        job.go            # Class B/C shell      (or)
        workflow.go       # Class D shell
        usecase.yaml      # class, idempotency, entity key, limits, adr001 link
        adr001.md         # required for C/D
  platform/
    tx/           # transactions + txn-local tenant context
    idempotency/  # Idempotency-Key store
    runs/         # run records and status
    effects/      # ledger and reconciliation
    audit/  authz/  authn/  ratelimit/
migrations/
fly.toml
```

**Dependency rule:** `transport → usecases → domain`. `domain` imports nothing from the framework, River, DBOS or `net/http`.

`usecase.yaml` (lightweight stand-in for the miniapp manifest):

```yaml
id: orders.place_order
class: D
idempotency_key: tenant:order_id
entity_key: order_id
limits: { max_per_minute_per_user: 10 }
adr001: ./adr001.md
```

---

## 6. Code shapes

### 6.1 Thin routes, use case owns the category

```go
mux := http.NewServeMux()
mux.Handle("GET /orders/{id}",  authn(authz("orders.read",   httpapi.Sync(getOrder))))      // Class A
mux.Handle("POST /orders",      authn(authz("orders.place",  idem(httpapi.Async(placeOrder))))) // Class D → 202
mux.Handle("POST /imports",     authn(authz("contacts.import", idem(httpapi.Async(importContacts))))) // Class C → 202
mux.Handle("GET /runs/{id}",    authn(httpapi.RunStatus()))
```

`httpapi.Sync` returns the result inline. `httpapi.Async` records the run, starts the job or workflow, and returns `202` with the run location. The route knows nothing about River or DBOS.

### 6.2 Class A use case (typical web endpoint)

```go
func (u PlaceNote) Handle(ctx context.Context, c Caller, in CreateNoteIn) (Note, error) {
    return u.tx.Run(ctx, c.Tenant, func(tx platform.Tx) (Note, error) {
        owner, err := u.repo.LockProject(ctx, tx, in.ProjectID)    // keyed serial: row lock
        if err != nil { return Note{}, err }
        decision := domain.CanAddNote(owner, c.User, u.clock.Now()) // pure rules
        if !decision.Allowed { return Note{}, apperr.Forbidden(decision.Reason) }
        n := domain.NewNote(in, u.ids.New(), u.clock.Now())
        if err := u.repo.InsertNote(ctx, tx, n); err != nil { return Note{}, err }
        u.audit.Record(tx, c, "note.created", n.ID)                  // same transaction
        return n, nil
    })
}
```

Use the pooled Neon endpoint for this request path. Tenant context must be set **transaction-locally** (`set_config(..., true)` / `SET LOCAL`), never with session-level `SET`.

### 6.3 Class B: enqueue in the same transaction

```go
// in the signup use case's transaction:
_, err = u.river.InsertTx(ctx, tx.Raw(), SendWelcomeArgs{Tenant: c.Tenant, UserID: user.ID},
    &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}})
```

If the transaction rolls back, no job exists. If it commits, the job exists and will run at least once; the worker must be idempotent.

### 6.4 Class D: accept, then start durably

```go
func (u PlaceOrder) Start(ctx context.Context, c Caller, in PlaceOrderIn) (Run, error) {
    runID := runs.IDFor(c.Tenant, "orders.place", in.OrderID)         // stable business identity
    run, created, err := u.runs.Accept(ctx, runID, c, in)              // unique key → duplicate-safe
    if err != nil || !created { return run, err }
    // Start with the same ID so a retry cannot create a second workflow.
    _, err = dbos.RunWorkflow(u.dbos, PlaceOrderWorkflow, WorkflowIn{RunID: runID, OrderID: in.OrderID},
        dbos.WithWorkflowID(runID))
    return run, err
}
```

Crash between `Accept` and `RunWorkflow`: a River periodic sweeper finds `ACCEPTED` runs with no workflow and starts them. Because the workflow ID is the run ID, the start is idempotent. If your pinned DBOS version supports atomically enqueueing a workflow in the same transaction as your application write (when app tables and the DBOS schema share a database), prefer that and confirm it in a spike.

### 6.5 Webhooks: acknowledge fast, process as Class B

```go
func (h Webhooks) Payment(w http.ResponseWriter, r *http.Request) {
    body, ok := h.verify(r)                       // signature + timestamp tolerance
    if !ok { http.Error(w, "invalid", http.StatusBadRequest); return }
    if err := h.inbox.RecordAndEnqueue(r.Context(), body); err != nil { // dedupe by event id + InsertTx
        http.Error(w, "retry", http.StatusServiceUnavailable); return
    }
    w.WriteHeader(http.StatusOK)                  // provider stops retrying only after durable receipt
}
```

---

## 7. Applying the framework: step by step

1. **Inventory** every operation: routes, form actions, webhooks, scheduled tasks, admin commands.
2. **Classify** each with the question flow and, for candidates for C/D, the ADR-001 score. Record the result in `usecase.yaml`.
3. **Decide the patterns:** keyed serialization for entity writes, approval gating for risky admin actions, saga for multi-system flows, reconciliation for external writes.
4. **Create use-case skeletons** with `domain/`, `ports.go` and the category-specific shell. Write the pure rules and their unit tests first.
5. **Wire transports:** routes call use cases; `Async` for B/C/D; shared middleware for authentication, authorization, rate limiting, idempotency and audit.
6. **Add required artifacts** per category (decision records, crash-boundary tests, replay tests, version strategy).
7. **Split processes on Fly:** `web`, `worker`, `dbos-worker`. Only `web` exposes an HTTP service.
8. **Operate:** alerts for failing jobs, stuck or quarantined runs, queue age, approval backlog and executor-down; runbooks per category.
9. **Review reclassification** whenever requirements change (e.g., a job gains a human wait).

---

## 8. Deploying on Fly.io and Neon

| Process group | Runs | Public? | Connections |
|---|---|---|---|
| `web` | HTTP server, Class A, enqueue/start, webhooks | Yes (HTTP service, health checks) | Pooled Neon endpoint for short request transactions |
| `worker` | River workers, sweepers, periodic jobs | No | Direct, or pooled with poll-only (decide by spike) |
| `dbos-worker` | DBOS workflows | No | Direct; share the pool handle with the datasource when you need atomic app write + checkpoint |

Operational rules:

- Run schema migrations as a gated deploy step before new code serves traffic; application migrations must not race library-managed (River/DBOS) migrations.
- The DBOS group runs exactly one Machine per DBOS application, with a graceful-shutdown window and a non-surge deploy strategy; verify overlap behavior in a spike.
- Budget connections: `Σ(machines × pool)` per group plus operations and headroom must fit the Neon compute's `max_connections`. Pooled mode cannot carry `LISTEN/NOTIFY` or session `SET`.
- Always-on workers and polling keep the Neon compute awake; size and price production for always-on.
- Restoring Neon to an earlier point rewinds application data and DBOS checkpoints but not external effects: restore only with a runbook that pauses runs and reconciles effects.

---

## 9. Dos and don'ts for web servers

| Do | Don't |
|---|---|
| Classify every route's use case; default to Class A | Treat the whole server as "async" or "durable" |
| Keep handlers thin; put rules in `domain/` | Put business rules in HTTP handlers or templates |
| Return `202` + a run resource for B/C/D | Hold the request open for long work or return `200` for queued work |
| Enqueue in the same transaction as the write | Fire-and-forget goroutines after responding (lost on deploy or crash) |
| Make every write idempotent (`Idempotency-Key`, form tokens, PRG) | Trust browsers and clients not to double-submit or retry |
| Verify webhook signatures; acknowledge after durable receipt | Do heavy work inline in the webhook handler |
| Use token-row + expiry + job for reset and verification flows | Build reset/verification as workflows |
| Model entity state (subscription, order) as Postgres tables; use D only for the timed or multi-stage parts | Store business state only inside workflow variables |
| Authorize server-side in the use-case layer for every operation | Rely on UI hiding of buttons |
| Encode output, set CSP, protect against CSRF and open redirects | Render provider or user text unescaped |
| Rate-limit and quota per user, tenant and API key | Allow unbounded calls to write or expensive routes |
| Keep UI polling modest with backoff | Poll every few hundred milliseconds |
| Use SSE or notifications for long waits | Keep a browser connection open for hours |
| Record audit events in the business transaction | Reconstruct history from logs |
| Write tests per category (unit → crash-boundary) | Rely on end-to-end clicks alone |
| Expose use cases to agents through an adapter later | Duplicate business logic for agent versus human paths |

### Web-specific anti-patterns

| Anti-pattern | Why it hurts | Fix |
|---|---|---|
| **Cron or timer inside the web process** | Lost on restart; runs twice with two Machines | River periodic job (B) or DBOS durable wait (D) |
| **Session state in workflows** | Sessions are request-bound; couples auth to durable engine | Plain session store in Postgres or signed cookies |
| **Client-side timers as business logic** | Closing the tab cancels the business process | Server-side durable wait |
| **"Just add DBOS" to every form** | Extra writes and versioning burden for no durability benefit | Class A with a transaction |
| **Webhook handler doing the work** | Provider timeouts trigger retries and duplicates | Ack fast, enqueue Class B, process idempotently |
| **Returning raw provider errors to the browser** | Leaks internals; confusing UX | Typed error taxonomy mapped to user-safe messages |
| **Optimistic UI with no reconciliation** | UI shows success that never happened | Show `QUEUED/RUNNING` states from the run resource |

---

## 10. Two worked examples

### 10.1 Order placement in a shop

| Step | Operation | Category | Notes |
|---|---|---|---|
| Browse, cart, view order | Reads and cart edits | A | Row lock on cart for edits |
| Place order | Start checkout | D candidate | Stages: reserve inventory → charge → confirm → fulfill; compensation on failure; payment wait for async methods |
| Payment webhook | Event | B → signals the workflow | Dedupe by event id |
| Invoice PDF | Job | B | |
| Shipping reminder emails | Sequence | D (or B per message if no cross-message state) | Decide by score |
| Admin refund | Command | A/B + approval table | Threshold-based approval |

**Pattern use:** keyed serialization on `order_id` (partitioned DBOS queue or row lock), saga for payment/inventory, effect ledger for provider calls, reconciliation sweeper for UNKNOWN charges.

### 10.2 Internal admin dashboard

All reads and edits are Class A. CSV exports are B. Bulk updates are C. There is no D, so no DBOS. The framework reduces to Level 0–2: classification, layering, idempotency, audit, and River. Adding DBOS would be a mistake under ADR-001.

---

## 11. Checklist for a web use case

```markdown
## Use case review — <area>/<use case>

- [ ] Category recorded in `usecase.yaml`; lowest sufficient class
- [ ] Route is thin; rules live in `domain/`; no business logic in templates
- [ ] A: inline result; one transaction; row lock; audit in same tx
- [ ] B/C/D: `202` + run resource; stable run ID; idempotent start; sweeper for ACCEPTED runs
- [ ] Idempotency-Key/form token/PRG in place for writes
- [ ] External calls via effect ledger; UNKNOWN handled; reconcile-by-read
- [ ] Authn/authz server-side; rate limits and quotas configured
- [ ] CSRF, CORS, output encoding, CSP reviewed (browser-facing routes)
- [ ] C/D: `adr001.md` complete; crash-boundary tests; D: determinism lint, replay test, version strategy
- [ ] Alerts and runbook entries exist for stuck, failed and quarantined runs
```

---

## 12. Summary

- Categories attach to **operations**; the web server is just the transport.
- **Most web operations are Class A.** Resist reaching for queues and workflows.
- Use **B** for single retryable background work, **C** for bounded batches with progress, **D** only where durable waits and multi-stage recovery are part of the business promise.
- Expose async work through **one `runs` resource**, and make every write **idempotent**.
- Keep **domain rules pure** and the shell thin, so a use case can move between categories without a rewrite.
- Adopt the framework **in levels**; stop when the next level stops paying for itself.

## prompt

just to clarify, one of the goals of this miniapp framework is to create robust primitives so that any miniapp implementation do not need to reinvent the wheel in order to create robust apps across the 4 app categories especially class D

## Answer

That is an important architectural clarification. In fact, I would make it an explicit, central objective of the Miniapp framework—not merely an implied benefit.

**The framework is not simply a runtime for executing miniapps. It is a platform that provides the reusable engineering primitives necessary to build robust business applications without requiring every miniapp developer to become an expert in distributed systems, transaction management, workflow orchestration, and failure recovery.**

That distinction changes how we should evaluate Prompt 15.2.

## **1. The framework owns reliability; miniapps own business logic**

The fundamental division should be:

| Responsibility                   | Framework                         | Miniapp                                                     |
| -------------------------------- | --------------------------------- | ----------------------------------------------------------- |
| Authentication and authorization | Provides and enforces             | Declares requirements                                       |
| Tenant isolation                 | Enforces                          | Uses trusted context                                        |
| Invocation and idempotency       | Manages                           | Defines business input                                      |
| Execution lifecycle              | Manages                           | Defines business operation                                  |
| Retry and recovery               | Provides safe mechanisms          | Defines business-specific retry policy where necessary      |
| Audit                            | Provides durable attribution      | Supplies business context                                   |
| Effects and approvals            | Provides secure primitives        | Declares consequential operations and approval requirements |
| Durable waits and signals        | Provides orchestration primitives | Defines when and why to wait                                |
| Cancellation                     | Enforces lifecycle guarantees     | Defines business cancellation behavior                      |
| Version compatibility            | Provides recovery mechanisms      | Declares compatibility requirements                         |
| Business rules                   | Does not invent them              | Owns them                                                   |

The miniapp developer should be able to concentrate on the actual problem being solved rather than repeatedly implementing infrastructure that every other miniapp also needs.

## **2. What this means across the four execution classes**

**Class A — Synchronous operation**

Framework-managed reliability

The developer implements the business operation. The framework handles invocation, authentication, tenant context, transaction boundaries, serialization where required, and audit.

**Class B — One-shot independently retryable job**

Framework-managed reliability

The developer defines the job. The framework provides durable enqueueing, identity, retries, idempotency, and observable outcomes.

**Class C — Bounded batch or multistep execution**

Framework-managed reliability

The developer defines the batch or bounded sequence. The framework provides execution controls, progress, concurrency limits, recovery, and the appropriate execution substrate.

**Class D — Durable business workflow**

Highest primitive-reuse priority

The developer defines the business process. The framework provides durable orchestration, checkpointing, waits, signals, approval integration, effect safety, cancellation, version compatibility, recovery, and lifecycle observability.

Class D is particularly important because its failure modes are substantially more complicated. Without framework-provided primitives, every miniapp author would have to rediscover the same distributed-systems problems.

## **3. The real meaning of Prompt 15 remediation**

This is where your clarification materially changes the acceptance criteria.

Take the five HIGH findings:

- X-01 and X-02 should not merely be fixed in the current approval implementation. The framework should expose an approval primitive that makes bypass and stranded approval workflows structurally difficult or impossible.
- X-03 should result in reusable invocation and workflow identity management, not a special-case locking fix for one miniapp.
- X-04 should result in a reusable, concurrency-safe approval decision mechanism.
- X-05 should result in an effect-reservation primitive that automatically coordinates with framework cancellation.

The same principle applies to the MEDIUM findings.

For example, a future Class D miniapp should not have to implement its own signal recovery loop, settlement audit, effect refusal audit, or workflow attachment mechanism.

**Fixing the framework means making the safe implementation the easiest and preferably the only supported implementation.**

## **4. An additional architectural acceptance criterion**

I recommend adding this objective explicitly to Prompt 15.2 and the P1 design document:

> **Reusable Framework Primitives and Miniapp Authoring Contract**
>
> The Miniapp framework must provide reusable, composable, and safe primitives across execution Classes A, B, C, and D. Miniapp implementations must not independently reinvent framework-level concerns such as invocation identity, idempotency, authorization, tenant isolation, lifecycle transitions, retry and recovery, audit, cancellation, or durable orchestration.
>
> Each primitive must have a documented contract, clearly defined responsibilities, safe default behavior, explicit failure semantics, and conformance tests.
>
> For Class D in particular, a miniapp author must be able to express its business workflow using framework-supported capabilities without independently implementing DBOS lifecycle management, approval recovery, signal delivery recovery, effect reservation, cancellation coordination, or settlement auditing.
>
> Framework internals may use DBOS, River, PostgreSQL, or other appropriate mechanisms, but miniapp authors should interact with the framework's stable abstractions rather than being required to reproduce or directly manage those underlying mechanisms.
>
> The framework must prove primitive reuse through multiple distinct test miniapps or conformance fixtures using the same framework capabilities, without duplicating lifecycle machinery.
>
> The framework remains responsible for technical guarantees; miniapps remain responsible for business rules and business-specific decisions.

## **5. One important distinction**

We should not attempt to turn every business behavior into a generic framework primitive.

For instance:

- The framework provides approval mechanics; the miniapp defines *what* needs approval.
- The framework provides durable waits; the miniapp defines *when* business conditions warrant waiting.
- The framework provides effect safety; the miniapp defines *what* consequential business operation to perform.
- The framework provides cancellation guarantees; the miniapp defines whether the underlying business process permits cancellation at a particular stage.

That separation prevents the framework from becoming an enormous, inflexible business-process engine.

## **My architectural conclusion**

The ultimate measure of Miniapp's success should be:

**How little infrastructure code must an application developer write to produce a reliable application?**

For Class D, a developer should be able to implement a sophisticated, multistage business workflow without personally solving workflow durability, approval races, effect idempotency, cancellation races, or recovery after process failure.

That is a much stronger objective than merely passing the current Class D conformance tests.

It also means Prompt 15 should leave behind two things:

1. A corrected framework with reusable reliability primitives.
2. A clear, stable miniapp authoring contract that makes those primitives accessible and difficult to misuse.

The neutral probe proves that the primitives work. Multiple distinct conformance fixtures should prove that they are genuinely reusable.

**We are building the foundation that makes robust miniapps ordinary to write, not building another framework that requires every developer to become its reliability engineer.**