# ADR-009: Manifest as the Artifact of Record

Status: Adopted.

## Context

Without a single governed artifact, tool contracts, permissions, and
deployment settings drift between code, docs, and production.

## Decision

`miniapp.json` is the versioned artifact of record: identity, tools,
substrate, capabilities, policy, fixtures, deployment profile, and the
exact dependency manifest. Published manifests are immutable. Referenced
deployment profiles and transitive policy inputs are content-addressed
at publish time (A4); unbound or stale references are rejected.

## Consequences

- Nothing undeclared is callable; CI proves code, schemas, and docs
  agree (drift tests).
- A profile change requires a new immutable version and fresh review.
- A release envelope binds each manifest digest to its exact
  implementation/build digest outside the self-hashed payload; the
  registry refuses mismatches (non-circular by construction).
