# Phase 0 Threat Model (DRAFT — requires security-owner sign-off, Q9)

Scope: the miniapp platform as frozen in `docs/plan4.2.md`: invocation
service, executors (Go/River/DBOS), capability proxy, token broker,
Sprite workspaces/tasks, promotion pipeline, and audit.

## Trust boundaries

1. Agent harness → platform: authenticated agent principal; never trusted
   for authority. Every call re-resolves authority server-side.
2. Platform → SaaS/custom APIs: capability proxy only. Provider
   credentials never leave the brokered boundary (A1).
3. Trusted runtime → Sprite: staged non-credential inputs in, results
   out. Sprites are untrusted execution venues even when platform-owned.
4. Sprite workspace → production: promotion only via trusted builds with
   review (A6). No direct path exists.
5. Tenant ↔ tenant: application authorization plus RLS defense in depth.

## Key threats and controls

| Threat | Control | Proven by |
|---|---|---|
| Agent exceeds authority | Two principals + manifest trust modes, server-side (ADR-005) | Phase 2 auth probes |
| Credential theft from Sprite | Proxy egress; no bearer tokens in untrusted code (A1) | §18 credential-isolation tests |
| Unreviewed network egress | Policy-before-exec with read-back (A3) | Round-trip spike + barrier tests |
| Profile drift bypassing review | Content-addressed profiles (A4) | Profile-binding tests |
| Duplicate/orphan Sprite compute | Idempotent registry + lease reaper (A2) | Crash matrix (A9) |
| Cross-tenant data access | App auth + RLS context + least-privilege roles | RLS spike + tenant probes |
| Silent promotion of unreviewed code | Trusted-build record + review gate (A6) | Promotion refusal tests |
| Replay/forgery of webhooks | HMAC + timestamp + event-ID dedup | Phase 2 webhook tests |
| SSRF via custom REST origin | Operator-approved origins, connection-time DNS/IP validation, redirect/response policy (see `ssrf-test-design.md`) | G2 SSRF probe suite; no custom connector before it passes |
| Audit evasion or log gaps | Mandatory audit events per transition | Phase 1 audit-trail proof |

## Residual risks

- Single DBOS executor availability until A11 thresholds add capacity.
- Providers without idempotency/status lookup force conservative
  quarantine (accepted by design).
- Harness-side approval UX varies; server enforcement is the control.

## Sign-off

- Security owner: _pending (Q9)_
- Threat model approved: _pending G0 condition 9_
