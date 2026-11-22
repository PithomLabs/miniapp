# ADR-002: Module and Repository Layout

Status: Adopted.

## Context

The framework library and reference miniapps must version together during
initial development without premature multi-module overhead.

## Decision

One repository, one Go module: `github.com/PithomLabs/miniapp`. Framework
packages live at the top level; references live under `miniapps/`. A
split into multiple modules requires a versioning justification and a new
ADR; it is not the default.

## Consequences

- One `go.mod`, one dependency lock, one CI pipeline.
- The framework must never import `miniapps/` (enforced by an import
  checker in CI from Phase 1).
