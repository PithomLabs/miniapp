# ADR-003: Shared Gateway and Common Runtime by Default

Status: Adopted.

## Context

Per-miniapp Fly applications multiply operational cost (deploys, secrets,
migrations, monitoring) before any isolation need is demonstrated. But
Class D workflow recovery is application-version-sensitive, so one
undifferentiated worker pool would couple every family's releases.

## Decision

Deploy a shared gateway and common Go runtime (one Fly app with
`gateway`, `river`, and `dbos-<family>` process groups) by default,
reusing the same compiled image. Each Class D workflow family gets an
isolated DBOS `AppName`, explicit application version, and a selectable
worker release that can retain the old artifact while old workflows
drain. Separate Fly apps or binaries only where release/drain evidence
requires them.

## Consequences

- Tenant and miniapp isolation comes from authorization, RLS, and
  manifest scoping — not from process boundaries — until proven
  insufficient.
- Family releases are independently drainable without forking the repo
  or the module.
