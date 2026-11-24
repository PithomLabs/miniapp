# Miniapp Security Contract

Status: PROPOSED architecture (Prompt32 workstream S). Nothing in
this document claims existing implementation unless tagged
[Existing]. Capability tags used throughout: [Existing] =
implemented and tested in-tree; [Partial] = implemented with
known gaps; [Documented] = specified but not implemented;
[Proposed] = new design in this package.

Provenance: `docs/plan4.2.md` §§9.3, 9.7, 14–15 (two-principal
model, trust modes, SSRF/egress, approvals, blast radius);
`docs/roadmap_security.md` (locked principles S1–S9);
`docs/phase1/invocation-lifecycle-contract.md` (M1/M2/M6/D1);
`docs/phase2/authorize-attempt-design.md` (C1–C7);
`docs/phase0/threat-model.md` (extended by §9 here and the
unified threat model in `foundational-architecture.md`).

Companion documents: `observability-contract.md` (workstream
O), `developer-experience-enforcement.md` (workstream DX),
`foundational-architecture.md` (unified runtime, glossary,
inventory), `foundational-conformance-matrix.md`,
`foundational-roadmap.md`.

## 1. Rationale

Miniapp is reachable through REST, MCP, CLI, and background
workers. All four reach the same business capabilities, so
all four must be governed by one authorization model.
Otherwise the weakest interface becomes a backdoor into the
whole framework — the failure this contract exists to
prevent.

Governing doctrine (locked):

> The developer writes business logic. Miniapp owns
> everything required to execute that business logic safely,
> reliably, securely, and observably.

> Framework guarantees must be enforced by construction, not
> delegated to developer discipline.

Two scoping rules bound this contract. First, Miniapp owns
authorization enforcement but does not become an
identity-management product: no user directory, password
system, SSO provider, or IAM console [Proposed boundary,
from roadmap S5]. Host applications and identity providers
may own identity issuance; Miniapp owns the authorization
contract and its enforcement. Second, the existing
`authorizeAttempt` effect boundary [Existing] is
authorization at the consequential-external-effect point —
it does not by itself establish who may call the API, use
an MCP tool, or administer a tenant. This contract designs
the layers above and around it without weakening it.

## 2. Current state (security inventory summary)

The full inventory is the appendix of
`foundational-architecture.md`. Security-relevant rows:

[Existing] Two-principal invocation identity
(`contract.Caller`: AgentID + Authority; tenant from the
server-resolved binding). RequestedTrust is untrusted input
and gated, never authoritative.

[Existing] `auth.Policy`: credential→Binding keys, tenant→
miniapp→trust grants, `Authenticate` / `Authorize`,
pinned-snapshot RLock discipline, credential-free
`ConfigHash`, ten-minute revocation-staleness model,
`/healthz` hash exposure, `verify-revocation` CLI.

[Existing] Kill switches (global/tenant/miniapp/
capability/tool scopes) effective without redeploy;
exact-match `miniapp.settle.operator` role; `SettleEffect`
shared boundary with non-oracle denials and server-derived
attribution.

[Existing] `authorizeAttempt` (C1–C7): ordered lock set,
predicated transitions, attempt fencing, refusal taxonomy,
same-transaction audit pairing.

[Existing] Durable approvals (`approvals` service +
`DecideApproval`): material-hash binding, expiry,
separation of duties at the service layer.

[Existing] Append-only attributed audit (see the
Observability Contract) recording every security-relevant
transition.

[Partial] Transport authentication: REST
`X-Miniapp-Credential` header, MCP credential-in-arguments,
CLI `--credential`/env. All three currently resolve
against demo API-key provisioning — explicitly not
production identity (carried residual). CLI credentials are
visible in process listings (documented limitation).

[Partial] Blast radius: kill switches exist; per-principal
quotas, circuit breakers, and value caps are absent.

[Documented] SSRF-safe egress design, OAuth/token-broker
design, out-of-band approval channel, revocation drill
procedure (NOT EXECUTED).

[Absent] External identity-provider integration,
delegation model, capability-scoped authorization beyond
trust modes, per-operation MCP authorization, worker
re-authorization, token broker implementation, provider
permission intersection, NEEDS_REAUTH.

## 3. Identity and authentication

### 3.1 Principal taxonomy [Proposed, extends Caller]

Four distinguishable principal kinds, all attributable in
audit:

- Human: an interactive user behind a host application,
  CLI session, or approval channel.
- AI agent: an autonomous caller (harness session) acting
  under explicit delegation with narrow scope.
- Service: a workload identity (cron trigger, webhook
  source after verification, peer service).
- Task (derived, never primary): a credential-less scoped
  workload identity for bounded isolated execution,
  derived from an authority per plan4.2 R22. Not a third
  human/business principal.

`contract.Caller` gains a principal-kind field and an
optional delegation reference; AgentID/Authority semantics
are unchanged. Kind is server-resolved, never
caller-supplied.

### 3.2 Authentication adapters [Partial → Proposed]

Each surface keeps its own adapter; every adapter's only
job is to convert surface credentials into a verified
`Caller`. Adapters own no authorization logic (ADR-004,
unchanged):

- REST: `X-Miniapp-Credential` bearer [Existing] plus a
  host-validated identity header binding for applications
  fronting Miniapp [Proposed]. Adapter interface:
  `AuthenticateRequest(r) (Caller, error)`.
- MCP: per-session authentication at connection setup,
  bound to the session — replacing credential-per-tool-arg
  [Proposed]. Tool arguments carry no identity.
- CLI: explicit user or workload identity via login-token
  file with owner-only permissions (0600), replacing
  env/flag credentials [Proposed]; resolves the
  process-listing residual.
- Workers: no ambient authority. A worker acts only under
  a persisted authority record created at admission (§5);
  there is no worker adapter that mints identity.

Startup validation (see the DX contract) refuses to serve
a surface whose adapter is unconfigured. No silent
trust-localhost fallback.

### 3.3 Trusted principal resolution [Proposed]

Resolution is a framework-owned function
`ResolvePrincipal(verified) Caller` invoked only by
adapters and the admission path. Its rules:

- Identity strings from the wire are claims until the
  adapter's verification succeeds; verified output is an
  opaque `Caller` value miniapps can read but not forge
  (unexported fields, accessor-only).
- Tenant is resolved from the verified identity's
  tenancy, never from a request parameter. Cross-tenant
  identifiers in one request are a fail-closed shape.
- Unknown, expired, or malformed credentials fail with
  one indistinguishable `auth-required` terminal — no
  oracle distinguishing "unknown user" from "bad token".

### 3.4 Credential validation and protection [Proposed]

- Credentials at rest: OS-protected files (0600) for CLI
  tokens; host vault for service credentials; Miniapp
  never logs, echoes, or persists presentation
  credentials. Presentation credentials never enter audit
  `details`, telemetry, or error strings (redaction is a
  conformance assertion, not a guideline).
- Credentials in transit: TLS-terminated by the host;
  Miniapp records the transport-security property of the
  connection (loopback vs TLS vs plaintext) and refuses
  bearer presentation over untrusted plaintext unless the
  deployment profile explicitly allows it (dev-only
  profile, content-addressed per A4).
- Rotation: credential identifiers are versioned;
  verification accepts the current and previous version
  during a bounded rotation window; revocation is
  immediate for the revoked version (unlike grant
  staleness — credentials are checked live at admission,
  not from a snapshot).

### 3.5 External identity providers [Proposed]

Miniapp consumes verified identities; it does not issue
human identities:

- OIDC-shaped integration: the host application (or a
  configured provider) authenticates the user; Miniapp
  validates the token (issuer, audience, expiry,
  signature via the provider's JWKS) and maps provider
  claims to `Caller` + tenant through an explicit,
  reviewed claim-mapping table. No implicit
  admin-claim passthrough: unmapped claims confer
  nothing.
- Service identities are issued by the operator through
  the same credential-versioning scheme as §3.4, bound to
  named manifests and scopes.
- Mapping changes are configuration changes: versioned,
  audited, and subject to the revocation-staleness window
  where snapshots are involved.

What Miniapp will not build: user registration, password
storage, SSO flows, or an admin console. If a deployment
needs those, they live in the host application or a real
IdP.

## 4. Authorization and delegation

### 4.1 Capability-based authorization [Proposed]

Trust modes (`dry-run`/`confirm`/`autonomous`) and grants
[Existing] remain the coarse execution gate. Above them,
this contract adds capability permissions:

- Each Business Capability (DX contract) declares
  `RequiredPermission` strings (e.g. `refund.create`).
- Authority records (persisted at admission for workers,
  resolved live for sync calls) carry a permission set:
  the intersection of (a) manifest-declared permissions,
  (b) grants/scopes actually issued to the authority,
  (c) delegation scope when acting through delegation.
- Enforcement points: admission (capability permission +
  trust mode + tenant), effect authorization (existing
  `authorizeAttempt`, unchanged), and worker checkpoints
  (§5). Denials are typed terminals (`auth-required`,
  `forbidden`, `grant-denied`) with audit pairing per
  the existing refusal discipline.

No permission registry language beyond string
capabilities plus manifest declaration is proposed. A
policy engine is explicitly deferred until real
requirements justify it (owner decision S-D4).

### 4.2 Application-declared permissions [Proposed]

Applications declare; Miniapp enforces. The manifest
(`miniapp.json`) gains a `permissions` section listing
the capability strings the miniapp may request, and each
tool/capability registration states what it requires.
Admission rejects a capability whose requirement is not
in the manifest's declared set (fail closed on
undeclared authority — the dual of P24-009's map-access
guard). RequestedTrust stays untrusted: it may only
narrow, never widen, the resolved authority.

### 4.3 Resource and tenant boundaries [Existing + Proposed]

[Existing] Tenant from the binding; tenant-predicated
operations; RLS defense in depth; transaction-local
tenant context. [Proposed] Resource scoping: authorities
may carry resource selectors (e.g. `customer:region=EU`)
evaluated against the invocation's resource attributes
at admission; unscoped authorities access only
tenant-global operations. Cross-tenant access is never
expressible: no selector crosses tenants.

### 4.4 Explicit agent delegation [Proposed]

The core new mechanism. An agent acts only under a
delegation record:

```text
Delegation {
  id            // delegation_id, server-minted
  principal     // on whose behalf (user/service)
  agent         // actor identity
  tenant        // bound tenant
  capabilities  // narrowed permission set
  constraints   // resource selectors, value caps
  expires_at    // short-lived; no indefinite agent grants
  revoked       // boolean + revoked_at
}
```

Rules: delegation is created by an authenticated
principal (or an approval decision) through a
framework-owned API, never self-asserted by the agent;
every agent invocation resolves its delegation live
(delegations are checked, not snapshotted); expiry and
revocation are enforced at admission and at worker
checkpoints; audit pairs every delegation-gated decision
with `principal_id`, `actor_id`, and `delegation_id`
(see the Observability correlation table). An agent
presenting another principal's identifier without a
live delegation covering it fails closed.

### 4.5 Least privilege, expiry, revocation [Proposed]

- Least privilege is structural: authorities carry only
  declared ∩ granted ∩ delegated permissions; there is
  no wildcard permission and no ambient admin.
- Authority expiry: delegations expire quickly (default
  hours, max bounded by policy); service credentials
  rotate per §3.4; grants keep the ten-minute snapshot
  staleness model [Existing, unchanged].
- Revocation is layered: credential revocation is
  immediate (live check); delegation revocation is
  immediate (live check); grant revocation follows the
  documented deployment procedure with bounded
  staleness. The NOT-EXECUTED drill remains the proof
  obligation for the grant layer.

### 4.6 Approvals [Existing + Proposed]

[Existing] Durable approval table, material-hash
binding, expiry, `DecideApproval` with
separation-of-duties at the service layer.
[Proposed] The approval channel: an authenticated
out-of-band interface (operator surface or verified
provider channel) where the approver identity is
established independently of the requesting session.
The requesting agent can never approve its own request
— enforced by comparing resolved approver identity
against the recorded requester, not by UI convention.
Approval decisions re-verify expiry, material hash,
authority, and self-approval policy immediately before
the consequential action (no TOCTOU between decision
and effect).

### 4.7 Server-derived context and anti-spoofing [Proposed]

Normative rules, each with a conformance test:

1. No caller-supplied identity: `Caller` fields are
   set only by adapters and admission.
2. No caller-supplied role: roles come from the
   server-resolved binding (the Settle exact-match
   precedent generalizes).
3. No caller-supplied tenant: tenant derives from
   verified identity; tenant parameters are rejected.
4. No caller-supplied authority: `Intent` carries no
   trust field (P26-001 guard, strengthened by P29-003);
   the generalization is that no request struct carries
   authority — authority arrives via the framework
   context.
5. EffectToken stays an opaque pass-through verified
   against the invocation row [Existing]; it is a
   claim-check, not an authority grant.

## 5. Execution-time security

Security context must survive the journey from admission
to background execution without gaining privilege. The
iron rule: a job or workflow must never become more
privileged merely because execution moved from an
authenticated request to a worker.

### 5.1 Authority persistence and checkpoints

At admission, the runtime persists an authority record
bound to the invocation: resolved `Caller`, permission
set, delegation reference, trust mode, tenant, and
expiry. Workers load this record — they never
re-derive authority from ambient configuration.

Checkpoints where authority is re-verified (not merely
carried):

- Class A: admission only (single transaction); the
  handler runs inside the admitted transaction, so no
  second check is meaningful. Contention recovery
  re-enters through admission.
- Class B: at enqueue (admission) and at worker pickup:
  the worker re-resolves delegation liveness/expiry and
  grant coverage before executing. A revoked delegation
  between enqueue and pickup refuses the job.
- Class C: at pipeline admission, at worker pickup (as
  B), and at item boundaries when items carry distinct
  resource selectors. Pipeline progress never implies
  item authority.
- Class D: at submission (admission), at workflow start
  (worker pickup, as B), and before every consequential
  effect or approval wait: the workflow re-reads the
  authoritative approval/grant state through durable
  steps. Replay re-executes checks against persisted
  outcomes; it never replays a stale "allowed".

Expiry mid-execution fails the subsequent checkpoint,
never retroactively: completed durable work stands;
further consequential actions refuse.

### 5.2 Per-class security responsibilities

| Class | Authenticate | Authorize | Propagate | Re-verify |
|---|---|---|---|---|
| A | Caller at admission | Capability + trust + tenant, one tx | In-tx context only | N/A (single tx) |
| B | Caller at admission | Same, persisted | Authority record → worker | Delegation + grants at pickup |
| C | Caller at admission | Pipeline + item selectors | Authority record → items | At pickup + item boundaries |
| D | Caller at admission | Capability + approval linkage | Authority record → workflow context | At start + before effects/waits |

### 5.3 Worker privilege rule (normative)

A worker's effective authority is at most the persisted
authority record, further narrowed by live delegation
and grant checks. There is no worker identity that
confers additional rights; "the worker needs admin to
do its job" is a design failure — the authority record
must already cover the job's needs, or the job must
request approval through the normal channel.

## 6. Relationship map: invocation → effect → settle

The four authorization layers and their order (each
layer sees only server-derived inputs):

```text
surface adapter          invocation admission       execution checkpoint
  Authenticate             capability + trust         delegation + grants
  credential → Caller      + tenant + approval        live (B/C/D only)
       |                   linkage                        |
       v                          |                       v
  +----------------------------------------------------------+
  | Effect authorization: authorizeAttempt (C1–C7, Existing) |
  | invocation → approval → sequence → kill-switch → grant  |
  | → predicated transition; typed refusals + paired audits  |
  +----------------------------------------------------------+
                                  |
                     operator intervention only:
                     SettleEffect (Existing) — exact settle-
                     operator role, non-oracle, audited
```

Layering rules: admission never authorizes an effect
(it authorizes the invocation); `authorizeAttempt`
never authenticates (it consumes the admitted
invocation + persisted trust); Settle never executes
business logic (it records an operator's evidence
claim). The authorization transaction stays separate
(C1–C3): admission, worker checkpoints, and
observability emission never share a transaction
with effect authorization. Bypass in either
direction is a conformance failure.

## 7. MCP must never become a privileged backdoor

MCP-specific rules (normative):

1. Session authentication (§3.2) precedes all tool
   availability; unauthenticated sessions see no
   tools, not even discovery.
2. Discovery is not authority: listing a tool reveals
   its existence; invoking it passes full admission.
3. Per-tool and per-operation authorization: tool
   routing resolves the capability permission and
   enforces it at admission like every other surface.
4. No MCP-carried identity: tool arguments carry
   business input only; identity arrives via the
   session binding.
5. Delegation is explicit: an agent session's
   delegation record bounds every call in the
   session; the session cannot widen its own scope.
6. Approval-elicitation (if a harness offers it) is
   UX only: the platform's out-of-band approval
   record (§4.6) is the authority, and the harness
   cannot self-approve.

## 8. Credential broker and secrets [Proposed]

The `tokens/` package (currently empty) becomes the
credential broker with four operations: resolve (fetch
a provider credential for an authorized operation),
refresh (rotate OAuth tokens), revoke (drop a grant),
redact (guaranteed credential-free views for audit and
telemetry). Rules:

- Provider credentials live in the broker boundary
  (host vault integration; envelope encryption is an
  owner decision, S-D5). They never enter workflow
  inputs, job payloads, Sprite task inputs, audit
  details, logs, or error strings.
- Capability code receives short-lived use-tokens
  scoped to one operation, never the standing
  credential. Use-tokens are minted at the effect
  boundary and expire with the attempt.
- Rotation and revocation propagate per §4.5 layering;
  `NEEDS_REAUTH` is the typed terminal when a grant
  dies mid-flight, with an out-of-band human action
  link — the agent cannot self-reauthorize.

## 9. Threat coverage

The unified threat model lives in
`foundational-architecture.md` §6 (extending
`docs/phase0/threat-model.md`). Threats this contract
directly addresses, with their control sections:

| Threat | Control |
|---|---|
| Rogue agent exceeds delegation | §4.4 live delegation checks + §5 checkpoints |
| Caller forges identity/role/tenant | §4.7 anti-spoofing rules + opaque Caller |
| MCP tool as backdoor | §7 session auth + per-op admission |
| Worker privilege creep | §5.3 worker privilege rule |
| Credential theft via logs/audit/errors | §3.4 + §8 redaction rules |
| Stale grant abused post-revocation | §4.5 layered revocation + drill proof |
| Self-approval by requesting agent | §4.6 approver/requester identity compare |
| Session fixation / credential replay | §3.4 versioned credentials + live checks |
| Approval TOCTOU | §4.6 re-verification before effect |

## 10. Owner decisions required

| ID | Decision | Needed before |
|---|---|---|
| S-D1 | First external-IdP mechanism (OIDC provider? host-app header trust?) and claim-mapping owner | Security foundation implementation |
| S-D2 | Delegation record shape and storage (new table vs approval-table extension) | Delegation implementation |
| S-D3 | Approval-channel UX owner (operator surface? verified provider channel?) | Approval-channel implementation |
| S-D4 | Policy-expression ceiling: string capabilities suffice, or a policy engine later? | Before any policy-language work |
| S-D5 | Credential-at-rest standard (host vault API? envelope encryption? KMS?) | Token-broker implementation |
| S-D6 | Service-identity issuance process and approver | First service identity |

## 11. Acceptance criteria (contract implementation)

1. Unauthenticated callers fail identically on all four
   surfaces (indistinguishable terminal, zero state
   change, paired audit).
2. An agent without a live delegation covering the
   capability is refused at admission on every surface.
3. Revoking a delegation between enqueue and worker
   pickup refuses the job (B/C/D checkpoint proof).
4. No request struct in the public API carries
   identity, role, tenant, or authority (static
   conformance assertion generalizing P26-001).
5. Credentials appear in no audit row, log line,
   telemetry span, or error string under adversarial
   fixtures (redaction conformance).
6. MCP discovery reveals no tool to unauthenticated
   sessions; every invocation passes admission.
7. Grant-revocation drill executed in an authorized
   environment (closes the Prompt30 residual).
8. All §9 threats map to passing conformance tests in
   `foundational-conformance-matrix.md`.
